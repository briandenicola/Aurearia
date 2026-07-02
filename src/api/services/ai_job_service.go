package services

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
)

const (
	aiJobQueueSize       = 100
	aiJobStaleTimeout    = time.Hour
	aiJobAnalyzeTimeout  = 5 * time.Minute
	aiJobEstimateTimeout = 3 * time.Minute
)

var (
	ErrAIJobInvalidSide        = errors.New("side must be 'obverse' or 'reverse'")
	ErrAIJobNoImages           = errors.New("coin has no matching image to analyze")
	ErrAIJobNoImagesForGrading = errors.New("coin has no image available for grading")
)

type AIJobAgent interface {
	AnalyzeCoin(ctx context.Context, req AnalyzeProxyRequest) (string, error)
	GradeCoin(ctx context.Context, req GradeProxyRequest) (string, error)
	CollectPortfolioReview(ctx context.Context, req PortfolioReviewProxyRequest) (string, error)
}

type AIJobService struct {
	repo        *repository.AIJobRepository
	agentProxy  AIJobAgent
	userRepo    *repository.UserRepository
	settingsSvc *SettingsService
	notifSvc    *NotificationService
	logger      *Logger
	queue       chan uint
}

type AIJobSubmissionResponse struct {
	Job models.AIJob `json:"job"`
}

type ValueEstimateResult struct {
	EstimatedValue float64             `json:"estimatedValue"`
	Confidence     string              `json:"confidence"`
	Reasoning      string              `json:"reasoning"`
	Comparables    []ValueEstimateComp `json:"comparables"`
}

func NewAIJobService(
	repo *repository.AIJobRepository,
	agentProxy AIJobAgent,
	userRepo *repository.UserRepository,
	settingsSvc *SettingsService,
	notifSvc *NotificationService,
	logger *Logger,
) *AIJobService {
	return &AIJobService{
		repo:        repo,
		agentProxy:  agentProxy,
		userRepo:    userRepo,
		settingsSvc: settingsSvc,
		notifSvc:    notifSvc,
		logger:      logger,
		queue:       make(chan uint, aiJobQueueSize),
	}
}

func (s *AIJobService) StartWorkers(workerCount int) {
	if workerCount < 1 {
		workerCount = 1
	}
	if ids, err := s.repo.RecoverStaleJobs(aiJobStaleTimeout); err == nil {
		for _, id := range ids {
			s.enqueueID(id)
		}
	} else {
		s.logger.Warn("ai-jobs", "Failed to recover stale AI jobs: %v", err)
	}
	for i := 0; i < workerCount; i++ {
		go s.worker()
	}
}

func (s *AIJobService) EnqueueAnalysis(userID, coinID uint, side string) (*models.AIJob, bool, error) {
	if side != "" && side != "obverse" && side != "reverse" {
		return nil, false, ErrAIJobInvalidSide
	}
	coin, err := s.repo.FindCoinWithImages(coinID, userID)
	if err != nil {
		return nil, false, err
	}
	if !hasImageForSide(coin, side) {
		return nil, false, ErrAIJobNoImages
	}
	job, created, err := s.repo.EnqueueOrFindActive(userID, coinID, models.AIJobTypeAnalysis, side)
	if err != nil {
		return nil, false, err
	}
	s.enqueueID(job.ID)
	return job, created, nil
}

func (s *AIJobService) EnqueueValueEstimate(userID, coinID uint) (*models.AIJob, bool, error) {
	if _, err := s.repo.FindCoinWithImages(coinID, userID); err != nil {
		return nil, false, err
	}
	job, created, err := s.repo.EnqueueOrFindActive(userID, coinID, models.AIJobTypeValueEstimate, "")
	if err != nil {
		return nil, false, err
	}
	s.enqueueID(job.ID)
	return job, created, nil
}

func (s *AIJobService) EnqueueCoinGrading(userID, coinID uint) (*models.AIJob, bool, error) {
	coin, err := s.repo.FindCoinWithImages(coinID, userID)
	if err != nil {
		return nil, false, err
	}
	if len(coin.Images) == 0 {
		return nil, false, ErrAIJobNoImagesForGrading
	}
	job, created, err := s.repo.EnqueueOrFindActive(userID, coinID, models.AIJobTypeCoinGrading, "")
	if err != nil {
		return nil, false, err
	}
	s.enqueueID(job.ID)
	return job, created, nil
}

func (s *AIJobService) GetJob(userID, jobID uint) (*models.AIJob, error) {
	return s.repo.GetByIDForUser(jobID, userID)
}

func (s *AIJobService) ListCoinJobs(userID, coinID uint, activeOnly bool) ([]models.AIJob, error) {
	if _, err := s.repo.FindCoinWithImages(coinID, userID); err != nil {
		return nil, err
	}
	return s.repo.ListForCoin(userID, coinID, activeOnly)
}

func (s *AIJobService) enqueueID(jobID uint) {
	select {
	case s.queue <- jobID:
	default:
		go func() { s.queue <- jobID }()
	}
}

func (s *AIJobService) worker() {
	for jobID := range s.queue {
		s.processJob(jobID)
	}
}

func (s *AIJobService) processJob(jobID uint) {
	job, claimed, err := s.repo.ClaimQueued(jobID)
	if err != nil {
		s.logger.Error("ai-jobs", "Failed to claim job %d: %v", jobID, err)
		return
	}
	if !claimed {
		return
	}

	processErr := s.processJobWithRetry(job)

	if processErr != nil {
		s.logger.Error("ai-jobs", "Job %d failed: %v", job.ID, processErr)
		if err := s.repo.Fail(job.ID, processErr.Error()); err != nil {
			s.logger.Error("ai-jobs", "Failed to persist job %d failure: %v", job.ID, err)
		}
		s.notifyFailure(job, processErr.Error())
	}
}

func (s *AIJobService) processJobWithRetry(job *models.AIJob) error {
	var processErr error
	for attempt := 0; attempt <= valuationMaxRetries; attempt++ {
		if attempt > 0 {
			backoff := valuationRetryDelay * time.Duration(attempt)
			s.logger.Warn("ai-jobs", "Job %d retry %d after %s", job.ID, attempt, backoff)
			time.Sleep(backoff)
		}

		switch job.JobType {
		case models.AIJobTypeAnalysis:
			processErr = s.processAnalysisJob(job)
		case models.AIJobTypeValueEstimate:
			processErr = s.processValueEstimateJob(job)
		case models.AIJobTypeCoinGrading:
			processErr = s.processCoinGradingJob(job)
		default:
			return fmt.Errorf("unknown AI job type: %s", job.JobType)
		}
		if processErr == nil {
			return nil
		}
		if !isRetryableError(processErr) {
			return processErr
		}
	}
	return processErr
}

func (s *AIJobService) processAnalysisJob(job *models.AIJob) error {
	coin, err := s.repo.FindCoinWithImages(job.CoinID, job.UserID)
	if err != nil {
		return fmt.Errorf("coin not found")
	}
	images := coin.Images
	if job.Side == "obverse" || job.Side == "reverse" {
		images = filterImagesBySide(coin.Images, job.Side)
	}
	if len(images) == 0 {
		return ErrAIJobNoImages
	}

	base64Images := s.readImagesAsBase64(images, job.ID)
	if len(base64Images) == 0 {
		return fmt.Errorf("no valid images found")
	}

	llmCfg, err := s.settingsSvc.ResolveLLMConfig()
	if err != nil {
		return err
	}
	prompt := s.settingsSvc.GetSetting(SettingObversePrompt)
	if job.Side == "reverse" {
		prompt = s.settingsSvc.GetSetting(SettingReversePrompt)
	}

	ctx, cancel := context.WithTimeout(context.Background(), aiJobAnalyzeTimeout)
	defer cancel()
	analysis, err := s.agentProxy.AnalyzeCoin(ctx, AnalyzeProxyRequest{
		LLM:    llmCfg,
		Coin:   buildCoinDataProxy(coin),
		Images: base64Images,
		Side:   job.Side,
		Prompt: prompt,
	})
	if err != nil {
		return err
	}

	column := "obverse_analysis"
	switch job.Side {
	case "reverse":
		column = "reverse_analysis"
	case "":
		column = "ai_analysis"
	}
	if err := s.repo.UpdateCoinAnalysis(job.CoinID, job.UserID, column, analysis); err != nil {
		return err
	}
	result := map[string]string{"analysis": analysis, "side": job.Side}
	resultJSON, _ := json.Marshal(result)
	if err := s.repo.Complete(job.ID, string(resultJSON)); err != nil {
		return err
	}
	s.notifyComplete(job, coin.Name)
	return nil
}

func (s *AIJobService) processCoinGradingJob(job *models.AIJob) error {
	coin, err := s.repo.FindCoinWithImages(job.CoinID, job.UserID)
	if err != nil {
		return fmt.Errorf("coin not found")
	}
	if len(coin.Images) == 0 {
		return ErrAIJobNoImagesForGrading
	}

	base64Images := s.readImagesAsBase64(coin.Images, job.ID)
	if len(base64Images) == 0 {
		return fmt.Errorf("no valid images found")
	}

	llmCfg, err := s.settingsSvc.ResolveLLMConfig()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), aiJobAnalyzeTimeout)
	defer cancel()
	report, err := s.agentProxy.GradeCoin(ctx, GradeProxyRequest{
		LLM:    llmCfg,
		Coin:   buildCoinDataProxy(coin),
		Images: base64Images,
	})
	if err != nil {
		return err
	}
	if report == "" {
		return fmt.Errorf("no grading report from AI")
	}

	result := map[string]string{"gradingReport": report}
	resultJSON, _ := json.Marshal(result)
	if err := s.repo.Complete(job.ID, string(resultJSON)); err != nil {
		return err
	}
	s.notifyComplete(job, coin.Name)
	return nil
}

func (s *AIJobService) processValueEstimateJob(job *models.AIJob) error {
	coin, err := s.repo.FindCoinWithImages(job.CoinID, job.UserID)
	if err != nil {
		return fmt.Errorf("coin not found")
	}
	llmCfg, err := s.settingsSvc.ResolveLLMConfig()
	if err != nil {
		return err
	}

	var zipCode string
	if user, err := s.userRepo.FindByID(job.UserID); err == nil {
		zipCode = user.ZipCode
	}
	description := BuildCoinDescription(coin)
	userMessage := fmt.Sprintf("Estimate the current market value of this coin:\n\n%s\n\n"+
		"Return ONLY the JSON block as specified in your instructions. No preamble or extra text.", description)

	ctx, cancel := context.WithTimeout(context.Background(), aiJobEstimateTimeout)
	defer cancel()
	aiText, err := s.agentProxy.CollectPortfolioReview(ctx, PortfolioReviewProxyRequest{
		LLM: llmCfg,
		User: UserContextProxy{
			UserID:  job.UserID,
			ZipCode: zipCode,
		},
		Message:         userMessage,
		ValuationPrompt: s.getValuationPrompt(),
	})
	if err != nil {
		return err
	}
	if aiText == "" {
		return fmt.Errorf("no response from AI")
	}

	estimate := ParseValueEstimate(aiText)
	result := ValueEstimateResult{
		EstimatedValue: estimate.EstimatedValue,
		Confidence:     estimate.Confidence,
		Reasoning:      estimate.Reasoning,
		Comparables:    estimate.Comparables,
	}
	resultJSON, _ := json.Marshal(result)
	if estimate.EstimatedValue > 0 {
		journalText := fmt.Sprintf("AI Value Estimate: $%.2f (%s confidence)", estimate.EstimatedValue, estimate.Confidence)
		if err := s.repo.CreateJournalEntry(&models.CoinJournal{
			CoinID: job.CoinID,
			UserID: job.UserID,
			Entry:  journalText,
		}); err != nil {
			return err
		}
	}
	if err := s.repo.Complete(job.ID, string(resultJSON)); err != nil {
		return err
	}
	s.notifyComplete(job, coin.Name)
	return nil
}

func (s *AIJobService) readImagesAsBase64(images []models.CoinImage, jobID uint) []string {
	base64Images := make([]string, 0, len(images))
	for _, img := range images {
		p := filepath.Join("uploads", img.FilePath)
		data, err := os.ReadFile(p)
		if err != nil {
			s.logger.Warn("ai-jobs", "Failed to read image %s for job %d: %v", p, jobID, err)
			continue
		}
		base64Images = append(base64Images, base64.StdEncoding.EncodeToString(data))
	}
	return base64Images
}

func (s *AIJobService) getValuationPrompt() string {
	if prompt := s.settingsSvc.GetSetting(SettingValuationPrompt); prompt != "" {
		return prompt
	}
	return DefaultValuationPrompt
}

func (s *AIJobService) notifyComplete(job *models.AIJob, coinName string) {
	if s.notifSvc != nil {
		s.notifSvc.NotifyAIJobCompleted(job.UserID, job.ID, job.CoinID, coinName, string(job.JobType))
	}
}

func (s *AIJobService) notifyFailure(job *models.AIJob, reason string) {
	if s.notifSvc != nil {
		s.notifSvc.NotifyAIJobFailed(job.UserID, job.ID, job.CoinID, string(job.JobType), reason)
	}
}

func hasImageForSide(coin *models.Coin, side string) bool {
	if side == "" {
		return len(coin.Images) > 0
	}
	return len(filterImagesBySide(coin.Images, side)) > 0
}

func filterImagesBySide(images []models.CoinImage, side string) []models.CoinImage {
	filtered := make([]models.CoinImage, 0, len(images))
	for _, img := range images {
		if string(img.ImageType) == side {
			filtered = append(filtered, img)
		}
	}
	return filtered
}

func buildCoinDataProxy(coin *models.Coin) CoinDataProxy {
	var purchasePrice, currentValue float64
	if coin.PurchasePrice != nil {
		purchasePrice = *coin.PurchasePrice
	}
	if coin.CurrentValue != nil {
		currentValue = *coin.CurrentValue
	}
	return CoinDataProxy{
		ID:            int(coin.ID),
		Name:          coin.Name,
		Ruler:         coin.Ruler,
		Era:           string(coin.Era),
		Denomination:  coin.Denomination,
		Material:      string(coin.Material),
		Category:      string(coin.Category),
		Grade:         coin.Grade,
		PurchasePrice: purchasePrice,
		CurrentValue:  currentValue,
		Notes:         coin.Notes,
	}
}
