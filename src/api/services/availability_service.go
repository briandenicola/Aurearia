package services

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
)

const (
	availabilityHTTPTimeout = 10 * time.Second
	availabilityRateDelay   = 750 * time.Millisecond
	// Keep in sync with src/agent/app/models/requests.py MAX_AVAILABILITY_ITEMS.
	availabilityAgentBatchSize = 10
	availabilityUserAgent      = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) " +
		"AppleWebKit/537.36 (KHTML, like Gecko) " +
		"Chrome/131.0.0.0 Safari/537.36"
	// Max body bytes to read for keyword detection (prevent memory exhaustion).
	maxBodyReadBytes = 512 * 1024 // 512KB
)

var soldHTMLPattern = regexp.MustCompile(`(?i)>\s*sold\s*<`)

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
	pushoverSvc *PushoverService
	userRepo    *repository.UserRepository
	settingsSvc *SettingsService
	logger      *Logger
}

// NewAvailabilityService creates a new AvailabilityService.
func NewAvailabilityService(
	coinRepo *repository.CoinRepository,
	availRepo *repository.AvailabilityRepository,
	agentProxy *AgentProxy,
	notifSvc *NotificationService,
	pushoverSvc *PushoverService,
	userRepo *repository.UserRepository,
	settingsSvc *SettingsService,
	logger *Logger,
) *AvailabilityService {
	return &AvailabilityService{
		coinRepo:    coinRepo,
		availRepo:   availRepo,
		agentProxy:  agentProxy,
		notifSvc:    notifSvc,
		pushoverSvc: pushoverSvc,
		userRepo:    userRepo,
		settingsSvc: settingsSvc,
		logger:      logger,
	}
}

// CheckURL performs an HTTP GET to check basic connectivity and status.
// For HTTP 200 responses, reads the body and checks for common sold/unavailable indicators.
// If no clear signal is found, marks as "unknown" for AI agent escalation.
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

	// For HTTP 200, read body and check for sold/unavailable keywords before
	// escalating to the AI agent. This catches common cases and reduces agent load.
	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, maxBodyReadBytes))
	if err != nil {
		// If body read fails, fall back to agent escalation
		result.Status = "unknown"
		result.Reason = "Requires AI analysis to determine availability"
		return result, nil
	}

	bodyLower := strings.ToLower(string(bodyBytes))

	// Check for strong sold indicators.
	if soldHTMLPattern.MatchString(bodyLower) {
		result.Status = "unavailable"
		result.Reason = "Detected as sold/unavailable"
		return result, nil
	}

	soldIndicators := []string{
		"status: sold", // VCoins and others
		"this item is sold",
		"no longer available",
		"item has been sold",
		"sold out",
	}

	for _, indicator := range soldIndicators {
		if strings.Contains(bodyLower, indicator) {
			result.Status = "unavailable"
			result.Reason = "Detected as sold/unavailable"
			return result, nil
		}
	}

	// Check for availability indicators (lower confidence, but helpful)
	availIndicators := []string{
		"add to cart",
		"add to basket",
		"buy now",
		"available for purchase",
		"in stock",
	}

	hasAvailIndicator := false
	for _, indicator := range availIndicators {
		if strings.Contains(bodyLower, indicator) {
			hasAvailIndicator = true
			break
		}
	}

	if hasAvailIndicator {
		result.Status = "available"
		result.Reason = "Detected purchase option in page content"
		return result, nil
	}

	// No clear signal — escalate to AI agent
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
	seenAmbiguousURLs := make(map[string]struct{})

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

		// Collect ambiguous results (still "unknown" after keyword check) for agent escalation
		if result.Status == "unknown" && result.HttpStatus != nil && *result.HttpStatus == 200 {
			if _, exists := seenAmbiguousURLs[coin.ReferenceURL]; !exists {
				ambiguousItems = append(ambiguousItems, AvailabilityCheckProxyItem{
					URL:      coin.ReferenceURL,
					CoinName: coin.Name,
				})
				seenAmbiguousURLs[coin.ReferenceURL] = struct{}{}
			}
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

	// Send Pushover notification if configured
	s.notifyRunComplete(userID, run)

	return run, nil
}

// escalateToAgent sends ambiguous URLs to the Python agent for LLM analysis
// and updates the corresponding results in-place.
func (s *AvailabilityService) escalateToAgent(
	runID uint,
	allResults []coinResult,
	ambiguousItems []AvailabilityCheckProxyItem,
) {
	llmConfig, err := s.settingsSvc.ResolveLLMConfig()
	if err != nil {
		s.logger.Warn("availability", "Unable to resolve AI provider config, skipping agent escalation: %v", err)
		return
	}

	resolvedURLs := make(map[string]struct{})
	for start := 0; start < len(ambiguousItems); start += availabilityAgentBatchSize {
		end := start + availabilityAgentBatchSize
		if end > len(ambiguousItems) {
			end = len(ambiguousItems)
		}
		batch := ambiguousItems[start:end]

		req := AvailabilityCheckProxyRequest{
			LLM:   llmConfig,
			Items: batch,
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		resp, err := s.agentProxy.CheckAvailability(ctx, req)
		cancel()
		if err != nil {
			s.logger.Warn("availability", "Agent escalation batch %d-%d failed (graceful fallback): %s", start+1, end, err)
			continue
		}

		// Build a lookup from URL → agent verdict for this batch.
		verdictMap := make(map[string]AvailabilityVerdictProxy)
		for _, v := range resp.Results {
			verdictMap[v.URL] = v
		}

		// Update allResults with agent verdicts.
		for i := range allResults {
			cr := &allResults[i]
			verdict, ok := verdictMap[cr.coin.ReferenceURL]
			if !ok {
				continue
			}

			cr.dbResult.Status = verdict.Status
			cr.dbResult.Reason = fmt.Sprintf("[Agent] %s", verdict.Reason)
			cr.dbResult.AgentUsed = true
			resolvedURLs[cr.coin.ReferenceURL] = struct{}{}

			if err := s.availRepo.UpdateResult(cr.dbResult); err != nil {
				s.logger.Warn("availability", "Failed to persist agent verdict for coin %d: %s", cr.coin.ID, err)
			}
		}
	}

	s.logger.Info("availability", "Agent resolved %d/%d ambiguous URLs", len(resolvedURLs), len(ambiguousItems))
}

// notifyRunComplete sends a Pushover notification with run details.
func (s *AvailabilityService) notifyRunComplete(userID uint, run *models.AvailabilityRun) {
	if s.pushoverSvc == nil || s.userRepo == nil {
		return
	}
	user, err := s.userRepo.FindByID(userID)
	if err != nil || user == nil || !user.PushoverEnabled || user.PushoverUserKey == "" {
		return
	}

	duration := time.Duration(run.DurationMs) * time.Millisecond
	msg := fmt.Sprintf("Checked: %d | Available: %d | Unavailable: %d | Unknown: %d | Duration: %s",
		run.CoinsChecked, run.Available, run.Unavailable, run.Unknown, duration.Round(time.Second))

	go func() {
		if err := s.pushoverSvc.SendNotification(user.PushoverUserKey, "Availability Check Complete", msg, ""); err != nil {
			s.logger.Error("pushover", "Failed to send availability run notification: %v", err)
		}
	}()
}
