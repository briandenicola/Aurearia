package services

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
)

const (
	availabilityHTTPTimeout = 10 * time.Second
	availabilityRateDelay   = 750 * time.Millisecond
	availabilityUserAgent   = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) " +
		"AppleWebKit/537.36 (KHTML, like Gecko) " +
		"Chrome/131.0.0.0 Safari/537.36"
)

// URLCheckResult holds the outcome of checking a single URL.
type URLCheckResult struct {
	Status     string
	Reason     string
	HttpStatus *int
	AgentUsed  bool
}

// coinResult tracks a check result paired with its coin and DB record.
type coinResult struct {
	coin     models.Coin
	result   *URLCheckResult
	dbResult *models.AvailabilityResult
}

// AvailabilityService orchestrates wishlist availability checking.
type AvailabilityService struct {
	coinRepo    *repository.CoinRepository
	availRepo   *repository.AvailabilityRepository
	agentProxy  *AgentProxy
	notifSvc    *NotificationService
	settingsSvc *SettingsService
	logger      *Logger
}

// NewAvailabilityService creates a new AvailabilityService.
func NewAvailabilityService(
	coinRepo *repository.CoinRepository,
	availRepo *repository.AvailabilityRepository,
	agentProxy *AgentProxy,
	notifSvc *NotificationService,
	settingsSvc *SettingsService,
	logger *Logger,
) *AvailabilityService {
	return &AvailabilityService{
		coinRepo:    coinRepo,
		availRepo:   availRepo,
		agentProxy:  agentProxy,
		notifSvc:    notifSvc,
		settingsSvc: settingsSvc,
		logger:      logger,
	}
}

// CheckURL performs an HTTP GET to check basic connectivity and status.
// All successful (HTTP 200) responses are marked as "unknown" and escalated to the AI agent
// to avoid false positives from simple keyword matching.
func (s *AvailabilityService) CheckURL(url string) (*URLCheckResult, error) {
	client := &http.Client{
		Timeout: availabilityHTTPTimeout,
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return &URLCheckResult{
			Status: "unknown",
			Reason: fmt.Sprintf("Failed to create request: %s", err),
		}, err
	}
	req.Header.Set("User-Agent", availabilityUserAgent)

	resp, err := client.Do(req)
	if err != nil {
		return &URLCheckResult{
			Status: "unknown",
			Reason: fmt.Sprintf("Connection failed: %s", err),
		}, nil
	}
	defer resp.Body.Close()

	statusCode := resp.StatusCode
	result := &URLCheckResult{HttpStatus: &statusCode}

	if statusCode == 404 || statusCode == 410 {
		result.Status = "unavailable"
		result.Reason = fmt.Sprintf("Page not found (HTTP %d)", statusCode)
		return result, nil
	}

	if statusCode >= 500 {
		result.Status = "unknown"
		result.Reason = fmt.Sprintf("Server error (HTTP %d)", statusCode)
		return result, nil
	}

	if statusCode != 200 {
		result.Status = "unknown"
		result.Reason = fmt.Sprintf("Unexpected HTTP status %d", statusCode)
		return result, nil
	}

	// For HTTP 200, mark as unknown and let the Python agent analyze
	// Simple keyword matching produces too many false positives
	result.Status = "unknown"
	result.Reason = "Requires AI analysis to determine availability"
	return result, nil
}

// CheckWishlistForUser runs availability checks for all wishlist items with URLs.
// Go performs fast HTTP status checks, then escalates all 200 OK responses
// to the Python agent for AI-powered analysis to avoid false positives from keyword matching.
func (s *AvailabilityService) CheckWishlistForUser(
	userID uint, triggerType string, triggerUserID *uint,
) (*models.AvailabilityRun, error) {
	coins, err := s.coinRepo.GetWishlistWithURLs(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch wishlist coins: %w", err)
	}

	startedAt := time.Now()
	run := &models.AvailabilityRun{
		UserID:        userID,
		TriggerType:   triggerType,
		TriggerUserID: triggerUserID,
		StartedAt:     startedAt,
	}
	if err := s.availRepo.CreateRun(run); err != nil {
		return nil, fmt.Errorf("failed to create run record: %w", err)
	}

	s.logger.Info("availability", "Starting check for user %d: %d coins with URLs", userID, len(coins))

	var available, unavailable, unknown, errCount int

	// Track results and ambiguous items for agent escalation
	var allResults []coinResult
	var ambiguousItems []AvailabilityCheckProxyItem

	for i, coin := range coins {
		result, _ := s.CheckURL(coin.ReferenceURL)

		avResult := &models.AvailabilityResult{
			RunID:      run.ID,
			CoinID:     coin.ID,
			CoinName:   coin.Name,
			URL:        coin.ReferenceURL,
			Status:     result.Status,
			Reason:     result.Reason,
			HttpStatus: result.HttpStatus,
			AgentUsed:  false,
			CheckedAt:  time.Now(),
		}
		if err := s.availRepo.CreateResult(avResult); err != nil {
			s.logger.Error("availability", "Failed to save result for coin %d: %s", coin.ID, err)
		}

		allResults = append(allResults, coinResult{coin: coin, result: result, dbResult: avResult})

		// Collect ambiguous results for agent escalation
		if result.Status == "unknown" && result.HttpStatus != nil && *result.HttpStatus == 200 {
			ambiguousItems = append(ambiguousItems, AvailabilityCheckProxyItem{
				URL:      coin.ReferenceURL,
				CoinName: coin.Name,
			})
		}

		s.logger.Debug("availability", "Coin %d (%s): %s — %s", coin.ID, coin.Name, result.Status, result.Reason)

		// Rate-limit between requests (skip after last)
		if i < len(coins)-1 {
			time.Sleep(availabilityRateDelay)
		}
	}

	// Escalate ambiguous results to the Python agent
	if len(ambiguousItems) > 0 && s.agentProxy != nil {
		s.logger.Info("availability", "Escalating %d ambiguous URLs to agent", len(ambiguousItems))
		s.escalateToAgent(run.ID, allResults, ambiguousItems)
	}

	// Tally final stats after any agent updates
	for _, cr := range allResults {
		switch cr.dbResult.Status {
		case "available":
			available++
		case "unavailable":
			unavailable++
		default:
			unknown++
		}

		// Update coin's listing status
		if err := s.coinRepo.UpdateListingStatus(cr.coin.ID, cr.dbResult.Status, cr.dbResult.Reason, time.Now()); err != nil {
			s.logger.Error("availability", "Failed to update listing status for coin %d: %s", cr.coin.ID, err)
		}

		// Notify user when a coin newly becomes unavailable
		if cr.dbResult.Status == "unavailable" && cr.coin.ListingStatus != "unavailable" && s.notifSvc != nil {
			s.notifSvc.NotifyWishlistUnavailable(userID, cr.coin, cr.dbResult.Reason)
		}
	}

	// Complete the run
	completedAt := time.Now()
	run.CoinsChecked = len(coins)
	run.Available = available
	run.Unavailable = unavailable
	run.Unknown = unknown
	run.Errors = errCount
	run.DurationMs = completedAt.Sub(startedAt).Milliseconds()
	run.CompletedAt = &completedAt

	if err := s.availRepo.CompleteRun(run); err != nil {
		s.logger.Error("availability", "Failed to complete run %d: %s", run.ID, err)
	}

	s.logger.Info("availability", "Run %d complete: %d checked, %d available, %d unavailable, %d unknown (%dms)",
		run.ID, len(coins), available, unavailable, unknown, run.DurationMs)

	return run, nil
}

// escalateToAgent sends ambiguous URLs to the Python agent for LLM analysis
// and updates the corresponding results in-place.
func (s *AvailabilityService) escalateToAgent(
	runID uint,
	allResults []coinResult,
	ambiguousItems []AvailabilityCheckProxyItem,
) {
	// Build LLM config from app settings
	provider := s.settingsSvc.GetSetting(SettingAIProvider)
	if provider == "" {
		s.logger.Warn("availability", "No AI provider configured, skipping agent escalation")
		return
	}

	llmConfig := LLMConfig{
		Provider:   provider,
		APIKey:     s.settingsSvc.GetSetting(SettingAnthropicAPIKey),
		Model:      s.settingsSvc.GetSetting(SettingAnthropicModel),
		OllamaURL:  s.settingsSvc.GetSetting(SettingOllamaURL),
		SearXNGURL: s.settingsSvc.GetSetting(SettingSearXNGURL),
	}

	req := AvailabilityCheckProxyRequest{
		LLM:   llmConfig,
		Items: ambiguousItems,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	resp, err := s.agentProxy.CheckAvailability(ctx, req)
	if err != nil {
		s.logger.Warn("availability", "Agent escalation failed (graceful fallback): %s", err)
		return
	}

	// Build a lookup from URL → agent verdict
	verdictMap := make(map[string]AvailabilityVerdictProxy)
	for _, v := range resp.Results {
		verdictMap[v.URL] = v
	}

	// Update allResults with agent verdicts
	for i := range allResults {
		cr := &allResults[i]
		verdict, ok := verdictMap[cr.coin.ReferenceURL]
		if !ok {
			continue
		}

		cr.dbResult.Status = verdict.Status
		cr.dbResult.Reason = fmt.Sprintf("[Agent] %s", verdict.Reason)
		cr.dbResult.AgentUsed = true

		// Update the DB result record
		s.availRepo.UpdateResult(cr.dbResult)
	}

	s.logger.Info("availability", "Agent resolved %d/%d ambiguous URLs", len(resp.Results), len(ambiguousItems))
}
