package services

import (
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
)

// AuctionEndingScheduler runs periodic checks for auction lots ending today.
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

// timeUntilNextRun calculates the delay until the next scheduled run.
// Uses AuctionEndingCheckStartTime (HH:MM) as the daily anchor and
// AuctionEndingCheckInterval (minutes) as the repeat cadence.
func (s *AuctionEndingScheduler) timeUntilNextRun() time.Duration {
	now := time.Now()
	startHour, startMin := s.getStartTime()
	interval := s.getInterval()

	// Build today's anchor from the start time
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

// runCycle executes one full auction ending check for all users.
func (s *AuctionEndingScheduler) runCycle() {
	enabled := s.settingsSvc.GetSetting(SettingAuctionEndingCheckEnabled)
	if enabled != "true" {
		s.logger.Debug("scheduler", "Auction ending check disabled, skipping cycle")
		return
	}

	s.runCycleWithTrigger("scheduled", nil)
}

// RunNow executes an immediate auction ending check for all users (manual trigger).
func (s *AuctionEndingScheduler) RunNow(triggerUserID *uint) (*models.AuctionEndingRun, error) {
	return s.runCycleWithTrigger("manual", triggerUserID)
}

// runCycleWithTrigger executes one full auction ending check and logs the run.
func (s *AuctionEndingScheduler) runCycleWithTrigger(triggerType string, triggerUserID *uint) (*models.AuctionEndingRun, error) {
	s.logger.Info("scheduler", "Starting %s auction ending check cycle", triggerType)
	startedAt := time.Now()

	// Create run record
	run := &models.AuctionEndingRun{
		TriggerType:   triggerType,
		TriggerUserID: triggerUserID,
		Status:        "running",
		StartedAt:     startedAt,
	}
	if err := s.auctionEndingRepo.CreateRun(run); err != nil {
		s.logger.Error("scheduler", "Failed to create run record: %s", err)
		return nil, err
	}

	// Execute the check
	lots, err := s.auctionRepo.GetEndingToday()
	if err != nil {
		s.logger.Error("scheduler", "Failed to fetch auction lots ending today: %s", err)
		now := time.Now()
		run.Status = "error"
		run.ErrorMessage = fmt.Sprintf("Failed to fetch lots: %v", err)
		run.CompletedAt = &now
		run.DurationMs = time.Since(startedAt).Milliseconds()
		s.auctionEndingRepo.CompleteRun(run)
		return run, err
	}

	run.LotsChecked = len(lots)

	if len(lots) == 0 {
		s.logger.Info("scheduler", "No auction lots ending today")
		now := time.Now()
		run.Status = "success"
		run.CompletedAt = &now
		run.DurationMs = time.Since(startedAt).Milliseconds()
		s.auctionEndingRepo.CompleteRun(run)
		return run, nil
	}

	// Group lots by user
	userLots := make(map[uint][]models.AuctionLot)
	for _, lot := range lots {
		userLots[lot.UserID] = append(userLots[lot.UserID], lot)
	}

	s.logger.Info("scheduler", "Found %d lots ending today across %d users", len(lots), len(userLots))

	today := time.Now().Format("2006-01-02")
	alertsSent := 0

	for userID, lots := range userLots {
		// Check idempotency — don't notify same user twice for same day
		s.mu.RLock()
		lastDate := s.lastNotified[userID]
		s.mu.RUnlock()

		if lastDate == today {
			s.logger.Debug("scheduler", "Already notified user %d today, skipping", userID)
			continue
		}

		sent := s.notifyUser(userID, lots)
		if sent {
			alertsSent++
			// Mark as notified
			s.mu.Lock()
			s.lastNotified[userID] = today
			s.mu.Unlock()
		}
	}

	run.AlertsSent = alertsSent

	s.logger.Info("scheduler", "%s auction ending check cycle complete — %d lots checked, %d alerts sent", triggerType, run.LotsChecked, run.AlertsSent)

	now := time.Now()
	run.Status = "success"
	run.CompletedAt = &now
	run.DurationMs = time.Since(startedAt).Milliseconds()
	s.auctionEndingRepo.CompleteRun(run)

	return run, nil
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

	// Build consolidated message
	title := "Auctions Ending Today"
	message := fmt.Sprintf("%d auction(s) you are bidding on end today:\n\n", len(lots))

	for _, lot := range lots {
		auctionHouse := lot.AuctionHouse
		if auctionHouse == "" {
			auctionHouse = "Unknown House"
		}
		saleName := lot.SaleName
		if saleName == "" {
			saleName = "Sale"
		}
		message += fmt.Sprintf("• %s - %s (Lot %d)\n", auctionHouse, saleName, lot.LotNumber)
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
