package services

import (
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
)

// auctionEndingStaleTimeout is the threshold after which a queued/running run is
// considered stale on startup and will be recovered to an error state.
const auctionEndingStaleTimeout = 30 * time.Minute

// AuctionEndingScheduler runs periodic checks for auction lots ending within the next 24 hours.
type AuctionEndingScheduler struct {
	auctionRepo       *repository.AuctionLotRepository
	auctionEndingRepo *repository.AuctionEndingRepository
	userRepo          *repository.UserRepository
	pushoverSvc       *PushoverService
	settingsSvc       *SettingsService
	logger            *Logger
	stopCh            chan struct{}
	once              sync.Once
	lastNotified      map[uint]string // userID -> date string (YYYY-MM-DD)
	mu                sync.RWMutex
	statusMu          sync.RWMutex
	isRunning         bool
}

// NewAuctionEndingScheduler creates a new scheduler.
func NewAuctionEndingScheduler(
	auctionRepo *repository.AuctionLotRepository,
	auctionEndingRepo *repository.AuctionEndingRepository,
	userRepo *repository.UserRepository,
	pushoverSvc *PushoverService,
	settingsSvc *SettingsService,
	logger *Logger,
) *AuctionEndingScheduler {
	return &AuctionEndingScheduler{
		auctionRepo:       auctionRepo,
		auctionEndingRepo: auctionEndingRepo,
		userRepo:          userRepo,
		pushoverSvc:       pushoverSvc,
		settingsSvc:       settingsSvc,
		logger:            logger,
		stopCh:            make(chan struct{}),
		lastNotified:      make(map[uint]string),
	}
}

// Start begins the periodic check loop. Call from a goroutine.
func (s *AuctionEndingScheduler) Start() {
	s.logger.Info("scheduler", "Auction ending scheduler started")

	// Recover any stale queued/running runs from a previous process lifecycle.
	s.recoverStaleRuns()

	// Initial delay to let the app finish startup
	select {
	case <-time.After(30 * time.Second):
	case <-s.stopCh:
		return
	}

	for {
		// Wait until the next scheduled time before running
		wait := s.timeUntilNextRun()
		s.logger.Info("scheduler", "Next auction ending check in %s", wait)

		select {
		case <-time.After(wait):
		case <-s.stopCh:
			s.logger.Info("scheduler", "Auction ending scheduler stopped")
			return
		}

		s.runCycle()
	}
}

// Stop signals the scheduler to shut down. Safe to call multiple times.
func (s *AuctionEndingScheduler) Stop() {
	s.once.Do(func() { close(s.stopCh) })
}

// RunNow executes one immediate auction-ending cycle.
func (s *AuctionEndingScheduler) RunNow() error {
	_, err := s.RunNowWithTrigger(nil)
	return err
}

// GetStatus returns the scheduler runtime status.
func (s *AuctionEndingScheduler) GetStatus() SchedulerStatus {
	s.statusMu.RLock()
	running := s.isRunning
	s.statusMu.RUnlock()

	enabled := s.settingsSvc.GetSetting(SettingAuctionEndingCheckEnabled) == "true"
	return SchedulerStatus{
		Name:      "auction-ending",
		Enabled:   enabled,
		IsRunning: running,
		NextRunIn: s.timeUntilNextRun(),
	}
}

// timeUntilNextRun calculates the delay until the next scheduled run.
// Uses the last completed scheduled run as the primary anchor and falls back
// to AuctionEndingCheckStartTime (HH:MM) when no scheduled history exists.
func (s *AuctionEndingScheduler) timeUntilNextRun() time.Duration {
	now := time.Now()
	interval := s.getInterval()

	lastRun := s.auctionEndingRepo.GetLastScheduledRun()
	if lastRun != nil && lastRun.CompletedAt != nil {
		nextFromLast := lastRun.CompletedAt.Add(interval)
		if nextFromLast.Before(now) {
			s.logger.Info("scheduler", "Last auction ending run completed %s ago, overdue — running now", now.Sub(*lastRun.CompletedAt).Round(time.Minute))
			return 0
		}
		return nextFromLast.Sub(now)
	}

	startHour, startMin := s.getStartTime()
	anchor := time.Date(now.Year(), now.Month(), now.Day(), startHour, startMin, 0, 0, now.Location())

	// If anchor is in the future, that's the next run
	if anchor.After(now) {
		return anchor.Sub(now)
	}

	// Find the next occurrence: anchor + N*interval that is still in the future
	elapsed := now.Sub(anchor)
	periods := int(elapsed/interval) + 1
	next := anchor.Add(time.Duration(periods) * interval)
	return next.Sub(now)
}

// getStartTime parses HH:MM from settings, defaults to 08:00.
func (s *AuctionEndingScheduler) getStartTime() (int, int) {
	raw := s.settingsSvc.GetSetting(SettingAuctionEndingCheckStartTime)
	var h, m int
	if _, err := fmt.Sscanf(raw, "%d:%d", &h, &m); err != nil || h < 0 || h > 23 || m < 0 || m > 59 {
		return 8, 0
	}
	return h, m
}

// getInterval returns the configured check interval.
func (s *AuctionEndingScheduler) getInterval() time.Duration {
	minStr := s.settingsSvc.GetSetting(SettingAuctionEndingCheckInterval)
	mins, err := strconv.Atoi(minStr)
	if err != nil || mins < 5 {
		mins = 1440 // Default: 24 hours
	}
	return time.Duration(mins) * time.Minute
}

// runCycle executes one full auction ending check for all users (scheduled path).
func (s *AuctionEndingScheduler) runCycle() {
	enabled := s.settingsSvc.GetSetting(SettingAuctionEndingCheckEnabled)
	if enabled != "true" {
		s.logger.Debug("scheduler", "Auction ending check disabled, skipping cycle")
		return
	}

	s.runCycleWithTrigger("scheduled", nil)
}

// runCycleWithTrigger executes one full auction ending check synchronously and logs the run.
// Used by the scheduled path.
func (s *AuctionEndingScheduler) runCycleWithTrigger(triggerType string, triggerUserID *uint) (*models.AuctionEndingRun, error) {
	s.statusMu.Lock()
	s.isRunning = true
	s.statusMu.Unlock()
	defer func() {
		s.statusMu.Lock()
		s.isRunning = false
		s.statusMu.Unlock()
	}()

	s.logger.Info("scheduler", "Starting %s auction ending check cycle", triggerType)
	startedAt := time.Now()

	run := &models.AuctionEndingRun{
		TriggerType:   triggerType,
		TriggerUserID: triggerUserID,
		Status:        models.AuctionEndingRunStatusRunning,
		StartedAt:     startedAt,
	}
	if err := s.auctionEndingRepo.CreateRun(run); err != nil {
		s.logger.Error("scheduler", "Failed to create run record: %s", err)
		return nil, err
	}

	s.executeCycle(run)
	return run, nil
}

// RunNowWithTrigger enqueues an immediate auction ending check and returns the queued run
// record immediately (202-style). The caller should poll run history for terminal status.
// If a queued or running run already exists it is returned without enqueuing a duplicate.
func (s *AuctionEndingScheduler) RunNowWithTrigger(triggerUserID *uint) (*models.AuctionEndingRun, error) {
	// Dedup: reuse any active (queued or running) run.
	if active := s.auctionEndingRepo.FindActiveRun(); active != nil {
		s.logger.Info("scheduler", "Auction ending run #%d already active (%s), reusing", active.ID, active.Status)
		return active, nil
	}

	// Create the queued record immediately so the caller gets an ID to poll.
	now := time.Now()
	run := &models.AuctionEndingRun{
		TriggerType:   "manual",
		TriggerUserID: triggerUserID,
		Status:        models.AuctionEndingRunStatusQueued,
		StartedAt:     now,
	}
	if err := s.auctionEndingRepo.CreateRun(run); err != nil {
		s.logger.Error("scheduler", "Failed to create queued auction ending run: %s", err)
		return nil, err
	}

	// Process asynchronously in a background goroutine.
	go s.processQueuedRun(run.ID)

	return run, nil
}

// processQueuedRun atomically claims a queued run and executes the auction ending cycle.
func (s *AuctionEndingScheduler) processQueuedRun(runID uint) {
	run, claimed, err := s.auctionEndingRepo.MarkRunning(runID)
	if err != nil {
		s.logger.Error("scheduler", "Failed to claim queued auction ending run #%d: %s", runID, err)
		return
	}
	if !claimed {
		s.logger.Info("scheduler", "Auction ending run #%d already claimed, skipping", runID)
		return
	}

	s.statusMu.Lock()
	s.isRunning = true
	s.statusMu.Unlock()
	defer func() {
		s.statusMu.Lock()
		s.isRunning = false
		s.statusMu.Unlock()
	}()

	s.executeCycle(run)
}

// executeCycle performs the core auction-ending check work on an already-persisted run
// that is in the running state. It updates the run to a terminal state when done.
func (s *AuctionEndingScheduler) executeCycle(run *models.AuctionEndingRun) {
	startedAt := run.StartedAt
	s.logger.Info("scheduler", "Executing %s auction ending check cycle for run #%d", run.TriggerType, run.ID)

	lots, err := s.auctionRepo.GetEndingSoon()
	if err != nil {
		s.logger.Error("scheduler", "Failed to fetch auction lots ending soon: %s", err)
		now := time.Now()
		run.Status = models.AuctionEndingRunStatusError
		run.ErrorMessage = fmt.Sprintf("Failed to fetch lots: %v", err)
		run.CompletedAt = &now
		run.DurationMs = time.Since(startedAt).Milliseconds()
		s.auctionEndingRepo.CompleteRun(run)
		return
	}

	run.LotsChecked = len(lots)

	if len(lots) == 0 {
		s.logger.Info("scheduler", "No auction lots ending soon")
		now := time.Now()
		run.Status = models.AuctionEndingRunStatusSuccess
		run.CompletedAt = &now
		run.DurationMs = time.Since(startedAt).Milliseconds()
		s.auctionEndingRepo.CompleteRun(run)
		return
	}

	// Group lots by user
	userLots := make(map[uint][]models.AuctionLot)
	for _, lot := range lots {
		userLots[lot.UserID] = append(userLots[lot.UserID], lot)
	}

	s.logger.Info("scheduler", "Found %d lots ending soon across %d users", len(lots), len(userLots))

	today := time.Now().Format("2006-01-02")
	alertsSent := 0

	for userID, userLotList := range userLots {
		// Check idempotency — don't notify same user twice for same day
		s.mu.RLock()
		lastDate := s.lastNotified[userID]
		s.mu.RUnlock()

		if lastDate == today {
			s.logger.Debug("scheduler", "Already notified user %d today, skipping", userID)
			continue
		}

		sent := s.notifyUser(userID, userLotList)
		if sent {
			alertsSent++
			// Mark as notified
			s.mu.Lock()
			s.lastNotified[userID] = today
			s.mu.Unlock()
		}
	}

	run.AlertsSent = alertsSent

	s.logger.Info("scheduler", "%s auction ending check cycle complete — %d lots checked, %d alerts sent", run.TriggerType, run.LotsChecked, run.AlertsSent)

	now := time.Now()
	run.Status = models.AuctionEndingRunStatusSuccess
	run.CompletedAt = &now
	run.DurationMs = time.Since(startedAt).Milliseconds()
	s.auctionEndingRepo.CompleteRun(run)
}

// recoverStaleRuns marks any queued or running runs older than auctionEndingStaleTimeout
// as error so they do not block future manual triggers after a process restart.
func (s *AuctionEndingScheduler) recoverStaleRuns() {
	recovered := s.auctionEndingRepo.RecoverStaleRuns(auctionEndingStaleTimeout)
	if recovered > 0 {
		s.logger.Warn("scheduler", "Recovered %d stale auction ending run(s) to error state", recovered)
	}
}

// notifyUser sends a consolidated Pushover notification to one user for their ending auctions.
// Returns true if a notification was sent, false otherwise.
func (s *AuctionEndingScheduler) notifyUser(userID uint, lots []models.AuctionLot) bool {
	user, err := s.userRepo.FindByID(userID)
	if err != nil || user == nil {
		s.logger.Warn("scheduler", "Failed to find user %d: %v", userID, err)
		return false
	}

	if !user.PushoverEnabled || user.PushoverUserKey == "" {
		s.logger.Debug("scheduler", "User %d does not have Pushover enabled", userID)
		return false
	}

	title := "Auctions Ending Soon"
	message := fmt.Sprintf("%d auction(s) you are bidding on end within 24 hours:\n\n", len(lots))

	for _, lot := range lots {
		auctionHouse := lot.AuctionHouse
		if auctionHouse == "" {
			auctionHouse = "Unknown House"
		}
		saleName := lot.SaleName
		if saleName == "" {
			saleName = "Sale"
		}
		message += fmt.Sprintf("- %s - %s (Lot %d)\n", auctionHouse, saleName, lot.LotNumber)
	}

	// Send notification
	sent := false
	if err := s.pushoverSvc.SendNotification(user.PushoverUserKey, title, message, ""); err != nil {
		s.logger.Error("pushover", "Failed to send auction ending notification to user %d: %v", userID, err)
	} else {
		s.logger.Info("scheduler", "Notified user %d of %d ending auctions", userID, len(lots))
		sent = true
	}
	return sent
}
