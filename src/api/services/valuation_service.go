package services

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
)

const (
	valuationBaseDelay    = 3 * time.Second  // minimum delay between AI calls
	valuationBatchSize    = 10               // pause after this many coins
	valuationBatchPause   = 30 * time.Second // cool-down between batches
	valuationStaleTimeout = 1 * time.Hour
	valuationMinFields    = 3
	valuationMaxRetries   = 2
	valuationRetryDelay   = 60 * time.Second // back off on rate-limit / 5xx errors
)

// cancelMap tracks cancellation channels for active valuation runs, keyed by run ID.
var cancelMap sync.Map

// ValuationService orchestrates bulk AI-powered coin valuation.
type ValuationService struct {
	coinRepo   *repository.CoinRepository
	valRepo    *repository.ValuationRepository
	agentProxy *AgentProxy
	userRepo   *repository.UserRepository
	logger     *Logger
}

// NewValuationService creates a new ValuationService.
func NewValuationService(
	coinRepo *repository.CoinRepository,
	valRepo *repository.ValuationRepository,
	agentProxy *AgentProxy,
	userRepo *repository.UserRepository,
) *ValuationService {
	return &ValuationService{
		coinRepo:   coinRepo,
		valRepo:    valRepo,
		agentProxy: agentProxy,
		userRepo:   userRepo,
		logger:     AppLogger,
	}
}

// ResolveLLMConfig reads AI provider settings and returns a configured LLMConfig.
func ResolveLLMConfig() (LLMConfig, error) {
	provider := GetSetting(SettingAIProvider)
	if provider == "" {
		return LLMConfig{}, fmt.Errorf("AI provider not configured. Please select Anthropic or Ollama in Admin Settings.")
	}

	cfg := LLMConfig{
		Provider:   provider,
		OllamaURL:  GetSetting(SettingOllamaURL),
		SearXNGURL: GetSetting(SettingSearXNGURL),
	}

	switch provider {
	case "anthropic":
		cfg.APIKey = GetSetting(SettingAnthropicAPIKey)
		cfg.Model = GetSetting(SettingAnthropicModel)
		if cfg.APIKey == "" {
			return LLMConfig{}, fmt.Errorf("Anthropic API key is required")
		}
	case "ollama":
		cfg.Model = GetSetting(SettingOllamaModel)
	default:
		return LLMConfig{}, fmt.Errorf("unknown AI provider: %s", provider)
	}

	return cfg, nil
}

// BuildCoinDescription constructs a text description of a coin for AI valuation.
func BuildCoinDescription(coin *models.Coin) string {
	var parts []string
	if coin.Name != "" {
		parts = append(parts, fmt.Sprintf("Name: %s", coin.Name))
	}
	if coin.Category != "" && coin.Category != "Other" {
		parts = append(parts, fmt.Sprintf("Category: %s", string(coin.Category)))
	}
	if coin.Denomination != "" {
		parts = append(parts, fmt.Sprintf("Denomination: %s", coin.Denomination))
	}
	if coin.Ruler != "" {
		parts = append(parts, fmt.Sprintf("Ruler: %s", coin.Ruler))
	}
	if coin.Era != "" {
		parts = append(parts, fmt.Sprintf("Era: %s", coin.Era))
	}
	if coin.Material != "" && coin.Material != "Other" {
		parts = append(parts, fmt.Sprintf("Material: %s", string(coin.Material)))
	}
	if coin.Grade != "" {
		parts = append(parts, fmt.Sprintf("Grade/Condition: %s", coin.Grade))
	}
	if coin.WeightGrams != nil {
		parts = append(parts, fmt.Sprintf("Weight: %.2fg", *coin.WeightGrams))
	}
	if coin.DiameterMm != nil {
		parts = append(parts, fmt.Sprintf("Diameter: %.1fmm", *coin.DiameterMm))
	}
	if coin.RarityRating != "" {
		parts = append(parts, fmt.Sprintf("Rarity/RIC: %s", coin.RarityRating))
	}
	if coin.Mint != "" {
		parts = append(parts, fmt.Sprintf("Mint: %s", coin.Mint))
	}
	if coin.PurchasePrice != nil {
		parts = append(parts, fmt.Sprintf("Purchase Price: $%.2f", *coin.PurchasePrice))
	}
	return strings.Join(parts, "\n")
}

// coinHasEnoughMetadata returns true if the coin has enough informative fields
// to produce a meaningful AI valuation.
func coinHasEnoughMetadata(coin *models.Coin) bool {
	count := 0
	if coin.Denomination != "" {
		count++
	}
	if coin.Ruler != "" {
		count++
	}
	if coin.Era != "" {
		count++
	}
	if coin.Mint != "" {
		count++
	}
	if coin.Grade != "" {
		count++
	}
	if coin.WeightGrams != nil {
		count++
	}
	if coin.DiameterMm != nil {
		count++
	}
	if coin.RarityRating != "" {
		count++
	}
	if coin.Material != "" && coin.Material != "Other" {
		count++
	}
	if coin.Category != "" && coin.Category != "Other" {
		count++
	}
	return count >= valuationMinFields
}

// ValuateCollectionForUser runs bulk valuation for all owned coins of a user.
func (s *ValuationService) ValuateCollectionForUser(
	userID uint, triggerType string, triggerUserID *uint,
) (*models.ValuationRun, error) {
	// Recover stale runs before checking for active ones
	s.valRepo.RecoverStaleRuns(valuationStaleTimeout)

	// Concurrency guard
	if hasActive, err := s.valRepo.HasActiveRun(userID); err != nil {
		return nil, fmt.Errorf("failed to check active runs: %w", err)
	} else if hasActive {
		return nil, fmt.Errorf("a valuation run is already in progress for this user")
	}

	// Get max coins setting
	maxCoins := 50
	if maxStr := GetSetting(SettingValuationMaxCoins); maxStr != "" {
		if v, err := strconv.Atoi(maxStr); err == nil && v > 0 {
			maxCoins = v
		}
	}

	// Get owned, non-sold, non-wishlist coins
	coins, err := s.valRepo.GetOwnedCoins(userID, maxCoins)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch coins: %w", err)
	}

	startedAt := time.Now()
	run := &models.ValuationRun{
		UserID:        userID,
		TriggerType:   triggerType,
		TriggerUserID: triggerUserID,
		Status:        "running",
		TotalCoins:    len(coins),
		StartedAt:     startedAt,
	}
	if err := s.valRepo.CreateRun(run); err != nil {
		return nil, fmt.Errorf("failed to create run record: %w", err)
	}

	// Register a cancellation channel for this run
	cancelCh := make(chan struct{})
	cancelMap.Store(run.ID, cancelCh)

	// Ensure run is finalized even on panic
	defer func() {
		cancelMap.Delete(run.ID)
		if run.Status == "running" {
			run.Status = "failed"
			run.ErrorMessage = "run terminated unexpectedly"
			completedAt := time.Now()
			run.CompletedAt = &completedAt
			run.DurationMs = completedAt.Sub(startedAt).Milliseconds()
			s.valRepo.CompleteRun(run)
		}
	}()

	// Resolve LLM config
	llmCfg, err := ResolveLLMConfig()
	if err != nil {
		run.Status = "failed"
		run.ErrorMessage = err.Error()
		completedAt := time.Now()
		run.CompletedAt = &completedAt
		run.DurationMs = completedAt.Sub(startedAt).Milliseconds()
		s.valRepo.CompleteRun(run)
		return run, nil
	}

	// Get user context for the agent
	var zipCode string
	if user, err := s.userRepo.FindByID(userID); err == nil {
		zipCode = user.ZipCode
	}

	// Get valuation prompt
	valuationPrompt := GetSetting(SettingValuationPrompt)

	s.logger.Info("valuation", "Starting bulk valuation for user %d: %d coins (max %d)", userID, len(coins), maxCoins)

	var checked, updated, skipped, errCount int
	batchCount := 0

	// updateProgress syncs in-memory counters to the DB so the frontend can poll
	updateProgress := func() {
		run.CoinsChecked = checked
		run.CoinsUpdated = updated
		run.CoinsSkipped = skipped
		run.Errors = errCount
		s.valRepo.UpdateRunProgress(run)
	}

	for i, coin := range coins {
		// Check for cancellation
		select {
		case <-cancelCh:
			s.logger.Info("valuation", "Run %d cancelled by user after %d coins", run.ID, checked+skipped)
			completedAt := time.Now()
			run.Status = "cancelled"
			run.CoinsChecked = checked
			run.CoinsUpdated = updated
			run.CoinsSkipped = skipped
			run.Errors = errCount
			run.DurationMs = completedAt.Sub(startedAt).Milliseconds()
			run.CompletedAt = &completedAt
			run.ErrorMessage = "cancelled by user"
			s.valRepo.CompleteRun(run)
			return run, nil
		default:
		}

		// Skip coins with insufficient metadata
		if !coinHasEnoughMetadata(&coin) {
			skipped++
			s.valRepo.AddResult(&models.ValuationResult{
				RunID:         run.ID,
				CoinID:        coin.ID,
				CoinName:      coin.Name,
				PreviousValue: coin.CurrentValue,
				Status:        "skipped",
				ErrorMessage:  "insufficient metadata for valuation",
				CheckedAt:     time.Now(),
			})
			updateProgress()
			continue
		}

		checked++
		batchCount++
		description := BuildCoinDescription(&coin)
		userMessage := fmt.Sprintf("Estimate the current market value of this coin:\n\n%s\n\n"+
			"Return ONLY the JSON block as specified in your instructions. No preamble or extra text.", description)

		proxyReq := PortfolioReviewProxyRequest{
			LLM: llmCfg,
			User: UserContextProxy{
				UserID:  userID,
				ZipCode: zipCode,
			},
			Message:         userMessage,
			ValuationPrompt: valuationPrompt,
		}

		// Retry loop with exponential backoff for transient/rate-limit errors
		var aiText string
		var callErr error
		for attempt := 0; attempt <= valuationMaxRetries; attempt++ {
			if attempt > 0 {
				backoff := valuationRetryDelay * time.Duration(attempt)
				s.logger.Warn("valuation", "Coin %d (%s): retry %d after %s", coin.ID, coin.Name, attempt, backoff)
				time.Sleep(backoff)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
			aiText, callErr = s.agentProxy.CollectPortfolioReview(ctx, proxyReq)
			cancel()

			if callErr == nil {
				break
			}
			// Only retry on rate-limit or server errors
			if !isRetryableError(callErr) {
				break
			}
		}

		if callErr != nil {
			errCount++
			s.logger.Warn("valuation", "Coin %d (%s) failed: %v", coin.ID, coin.Name, callErr)
			s.valRepo.AddResult(&models.ValuationResult{
				RunID:         run.ID,
				CoinID:        coin.ID,
				CoinName:      coin.Name,
				PreviousValue: coin.CurrentValue,
				Status:        "error",
				ErrorMessage:  callErr.Error(),
				CheckedAt:     time.Now(),
			})
			updateProgress()
			continue
		}

		estimate := ParseValueEstimate(aiText)

		result := &models.ValuationResult{
			RunID:          run.ID,
			CoinID:         coin.ID,
			CoinName:       coin.Name,
			PreviousValue:  coin.CurrentValue,
			EstimatedValue: estimate.EstimatedValue,
			Confidence:     estimate.Confidence,
			Reasoning:      estimate.Reasoning,
			Status:         "success",
			CheckedAt:      time.Now(),
		}

		if estimate.EstimatedValue > 0 {
			s.updateCoinValuation(&coin, userID, &estimate)
			updated++
		}

		s.valRepo.AddResult(result)
		updateProgress()

		s.logger.Debug("valuation", "Coin %d (%s): $%.2f (%s confidence)",
			coin.ID, coin.Name, estimate.EstimatedValue, estimate.Confidence)

		// Rate limiting: pause between each call, with a longer batch cool-down
		if i < len(coins)-1 {
			if batchCount >= valuationBatchSize {
				s.logger.Debug("valuation", "Batch of %d complete, pausing %s before next batch", batchCount, valuationBatchPause)
				time.Sleep(valuationBatchPause)
				batchCount = 0
			} else {
				time.Sleep(valuationBaseDelay)
			}
		}
	}

	// Record one value snapshot for the user at the end
	s.coinRepo.RecordValueSnapshot(userID)

	// Complete the run
	completedAt := time.Now()
	run.Status = "completed"
	run.CoinsChecked = checked
	run.CoinsUpdated = updated
	run.CoinsSkipped = skipped
	run.Errors = errCount
	run.DurationMs = completedAt.Sub(startedAt).Milliseconds()
	run.CompletedAt = &completedAt

	if err := s.valRepo.CompleteRun(run); err != nil {
		s.logger.Error("valuation", "Failed to complete run %d: %s", run.ID, err)
	}

	s.logger.Info("valuation", "Run %d complete: %d checked, %d updated, %d skipped, %d errors (%dms)",
		run.ID, checked, updated, skipped, errCount, run.DurationMs)

	return run, nil
}

// CancelRun signals a running valuation to stop after the current coin.
// Returns true if a running valuation was found and signalled.
func (s *ValuationService) CancelRun(runID uint) bool {
	if ch, ok := cancelMap.Load(runID); ok {
		close(ch.(chan struct{}))
		return true
	}
	// If not in memory (maybe stale), mark it in the DB directly
	s.valRepo.CancelRun(runID)
	return false
}

// updateCoinValuation updates a single coin's current value and records
// a value history entry and journal entry.
func (s *ValuationService) updateCoinValuation(coin *models.Coin, userID uint, estimate *ValueEstimate) {
	newValue := estimate.EstimatedValue

	// Update the coin's current value
	if err := s.coinRepo.UpdateField(coin, "current_value", newValue); err != nil {
		s.logger.Error("valuation", "Failed to update current value for coin %d: %v", coin.ID, err)
		return
	}

	// Record value history
	s.coinRepo.RecordValueHistory(&models.CoinValueHistory{
		CoinID:     coin.ID,
		UserID:     userID,
		Value:      newValue,
		Confidence: estimate.Confidence,
		RecordedAt: time.Now(),
	})

	// Record journal entry
	journalText := fmt.Sprintf("Scheduled AI Value Estimate: $%.2f (%s confidence)", newValue, estimate.Confidence)
	s.coinRepo.CreateJournalEntry(&models.CoinJournal{
		CoinID: coin.ID,
		UserID: userID,
		Entry:  journalText,
	})
}

// isRetryableError returns true for rate-limit (429) and server (5xx) errors
// that are worth retrying after a backoff.
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "429") ||
		strings.Contains(msg, "rate") ||
		strings.Contains(msg, "overloaded") ||
		strings.Contains(msg, "529") ||
		strings.Contains(msg, "500") ||
		strings.Contains(msg, "502") ||
		strings.Contains(msg, "503") ||
		strings.Contains(msg, "504")
}
