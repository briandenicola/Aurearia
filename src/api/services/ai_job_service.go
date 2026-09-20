package services

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
)

const (
	aiJobAnalyzeTimeout  = 5 * time.Minute
	aiJobEstimateTimeout = 3 * time.Minute
)

var (
	ErrAIJobInvalidSide        = errors.New("side must be 'obverse' or 'reverse'")
	ErrAIJobNoImages           = errors.New("coin has no matching image to analyze")
	ErrAIJobNoImagesForGrading = errors.New("coin has no image available for grading")
	ErrAIJobStopped            = errors.New("AI job workers are stopping")
)

type AIJobAgent interface {
	AnalyzeCoin(ctx context.Context, req AnalyzeProxyRequest) (string, error)
	GradeCoin(ctx context.Context, req GradeProxyRequest) (string, error)
	CollectPortfolioReview(ctx context.Context, req PortfolioReviewProxyRequest) (string, error)
}

type aiJobInferenceError struct{ error }

type AIJobService struct {
	repo        *repository.AIJobRepository
	agentProxy  AIJobAgent
	userRepo    *repository.UserRepository
	settingsSvc *SettingsService
	notifSvc    *NotificationService
	logger      *Logger
	wake        chan struct{}
	ctx         context.Context
	cancel      context.CancelFunc
	done        chan struct{}
	startOnce   sync.Once
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
	ctx, cancel := context.WithCancel(context.Background())
	return &AIJobService{
		repo:        repo,
		agentProxy:  agentProxy,
		userRepo:    userRepo,
		settingsSvc: settingsSvc,
		notifSvc:    notifSvc,
		logger:      logger,
		wake:        make(chan struct{}, 1),
		ctx:         ctx,
		cancel:      cancel,
		done:        make(chan struct{}),
	}
}

func (s *AIJobService) StartWorkers(workerCount int) {
	if workerCount < 1 {
		workerCount = 1
	}
	s.startOnce.Do(func() {
		go func() {
			defer close(s.done)
			for s.ctx.Err() == nil {
				if err := s.repo.ReconcileInterruptedJobs(); err == nil {
					break
				} else {
					s.logger.Error("ai-jobs", "Failed to reconcile interrupted jobs: %v", err)
				}
				select {
				case <-s.ctx.Done():
					return
				case <-time.After(time.Second):
				}
			}
			var workers sync.WaitGroup
			for i := 0; i < workerCount; i++ {
				workers.Go(s.worker)
			}
			workers.Wait()
		}()
	})
}

func (s *AIJobService) StopWorkers(ctx context.Context) error {
	s.cancel()
	s.startOnce.Do(func() { close(s.done) })
	select {
	case <-s.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *AIJobService) EnqueueAnalysis(userID, coinID uint, side string) (*models.AIJob, bool, error) {
	if s.ctx.Err() != nil {
		return nil, false, ErrAIJobStopped
	}
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
	if s.ctx.Err() != nil {
		return nil, false, ErrAIJobStopped
	}
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
	if s.ctx.Err() != nil {
		return nil, false, ErrAIJobStopped
	}
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
	case s.wake <- struct{}{}:
	default:
	}
}

func (s *AIJobService) worker() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for s.ctx.Err() == nil {
		ids, err := s.repo.ListQueuedIDs()
		if err != nil {
			s.logger.Error("ai-jobs", "Failed to list queued jobs: %v", err)
		} else {
			for _, id := range ids {
				if s.ctx.Err() != nil {
					return
				}
				s.processJob(id)
			}
		}
		select {
		case <-s.ctx.Done():
			return
		case <-s.wake:
		case <-ticker.C:
		}
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
			return
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
			select {
			case <-s.ctx.Done():
				return s.ctx.Err()
			case <-time.After(backoff):
			}
		}
		if err := s.ctx.Err(); err != nil {
			return err
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
		var inferenceErr *aiJobInferenceError
		if !errors.As(processErr, &inferenceErr) || !isRetryableError(inferenceErr.error) {
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

	ctx, cancel := context.WithTimeout(s.ctx, aiJobAnalyzeTimeout)
	defer cancel()
	analysis, err := s.agentProxy.AnalyzeCoin(ctx, AnalyzeProxyRequest{
		LLM:    llmCfg,
		Coin:   buildCoinDataProxy(coin),
		Images: base64Images,
		Side:   job.Side,
		Prompt: prompt,
	})
	if err != nil {
		return &aiJobInferenceError{err}
	}
	if err := s.ctx.Err(); err != nil {
		return err
	}

	column := "obverse_analysis"
	switch job.Side {
	case "reverse":
		column = "reverse_analysis"
	case "":
		column = "ai_analysis"
	}
	result := map[string]string{"analysis": analysis, "side": job.Side}
	resultJSON, _ := json.Marshal(result)
	if err := s.repo.CompleteAnalysis(job, column, analysis, string(resultJSON)); err != nil {
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

	ctx, cancel := context.WithTimeout(s.ctx, aiJobAnalyzeTimeout)
	defer cancel()
	report, err := s.agentProxy.GradeCoin(ctx, GradeProxyRequest{
		LLM:    llmCfg,
		Coin:   buildCoinDataProxy(coin),
		Images: base64Images,
	})
	if err != nil {
		return &aiJobInferenceError{err}
	}
	if err := s.ctx.Err(); err != nil {
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

	ctx, cancel := context.WithTimeout(s.ctx, aiJobEstimateTimeout)
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
		return &aiJobInferenceError{err}
	}
	if err := s.ctx.Err(); err != nil {
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
		resized, err := resizeForAnalysis(data)
		if err != nil {
			s.logger.Warn("ai-jobs", "Failed to resize image %s for job %d: %v", p, jobID, err)
			continue
		}
		if len(resized) != len(data) {
			s.logger.Info("ai-jobs", "Resized image %s from %d to %d bytes for job %d", p, len(data), len(resized), jobID)
		}
		base64Images = append(base64Images, base64.StdEncoding.EncodeToString(resized))
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
