package services

import (
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/briandenicola/ancient-coins-api/repository"
)

// AvailabilityScheduler runs periodic wishlist availability checks.
type AvailabilityScheduler struct {
	svc         *AvailabilityService
	coinRepo    *repository.CoinRepository
	availRepo   *repository.AvailabilityRepository
	settingsSvc *SettingsService
	logger      *Logger
	stopCh      chan struct{}
	once        sync.Once
	statusMu    sync.RWMutex
	isRunning   bool
}

// NewAvailabilityScheduler creates a new scheduler.
func NewAvailabilityScheduler(
	svc *AvailabilityService,
	coinRepo *repository.CoinRepository,
	availRepo *repository.AvailabilityRepository,
	settingsSvc *SettingsService,
	logger *Logger,
) *AvailabilityScheduler {
	return &AvailabilityScheduler{
		svc:         svc,
		coinRepo:    coinRepo,
		availRepo:   availRepo,
		settingsSvc: settingsSvc,
		logger:      logger,
		stopCh:      make(chan struct{}),
	}
}

// Start begins the periodic check loop. Call from a goroutine.
func (s *AvailabilityScheduler) Start() {
	s.logger.Info("scheduler", "Wishlist availability scheduler started")

	// Initial delay to let the app finish startup
	select {
	case <-time.After(30 * time.Second):
	case <-s.stopCh:
		return
	}

	for {
		// Wait until the next scheduled time before running
		wait := s.timeUntilNextRun()
		s.logger.Info("scheduler", "Next availability check in %s", wait)

		select {
		case <-time.After(wait):
		case <-s.stopCh:
			s.logger.Info("scheduler", "Scheduler stopped")
			return
		}

		s.runCycle()
	}
}

// Stop signals the scheduler to shut down. Safe to call multiple times.
func (s *AvailabilityScheduler) Stop() {
	s.once.Do(func() { close(s.stopCh) })
}

// RunNow executes one immediate availability cycle.
func (s *AvailabilityScheduler) RunNow() error {
	return s.RunNowWithTrigger(nil)
}

// RunNowWithTrigger executes one immediate availability cycle and records the triggering admin user.
func (s *AvailabilityScheduler) RunNowWithTrigger(triggerUserID *uint) error {
	s.runCycleWithTrigger("manual", triggerUserID)
	return nil
}

// GetStatus returns the scheduler runtime status.
func (s *AvailabilityScheduler) GetStatus() SchedulerStatus {
	s.statusMu.RLock()
	running := s.isRunning
	s.statusMu.RUnlock()

	enabled := s.settingsSvc.GetSetting(SettingWishlistCheckEnabled) == "true"
	return SchedulerStatus{
		Name:      "availability",
		Enabled:   enabled,
		IsRunning: running,
		NextRunIn: s.timeUntilNextRun(),
	}
}

// timeUntilNextRun calculates the delay until the next scheduled run.
// If there is a previous completed scheduled run, the interval is measured from
// that completion timestamp so app restarts do not reset the schedule. Falls
// back to the start-time anchor calculation only when no run history exists.
func (s *AvailabilityScheduler) timeUntilNextRun() time.Duration {
	now := time.Now()
	interval := s.getInterval()

	// Anchor to the last actual scheduled run so the interval is always
	// measured from the previous execution, regardless of restarts.
	lastRun := s.availRepo.GetLastScheduledRun()
	if lastRun != nil && lastRun.CompletedAt != nil {
		nextFromLast := lastRun.CompletedAt.Add(interval)
		if nextFromLast.Before(now) {
			// Overdue — run immediately (catches up after a long outage)
			s.logger.Info("scheduler", "Last scheduled run completed %s ago, overdue — running now", now.Sub(*lastRun.CompletedAt).Round(time.Minute))
			return 0
		}
		return nextFromLast.Sub(now)
	}

	// No previous run — use today's start-time as the anchor.
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

// getStartTime parses HH:MM from settings, defaults to 02:00.
func (s *AvailabilityScheduler) getStartTime() (int, int) {
	raw := s.settingsSvc.GetSetting(SettingWishlistCheckStartTime)
	var h, m int
	if _, err := fmt.Sscanf(raw, "%d:%d", &h, &m); err != nil || h < 0 || h > 23 || m < 0 || m > 59 {
		return 2, 0
	}
	return h, m
}

// getInterval returns the configured check interval.
func (s *AvailabilityScheduler) getInterval() time.Duration {
	minStr := s.settingsSvc.GetSetting(SettingWishlistCheckInterval)
	mins, err := strconv.Atoi(minStr)
	if err != nil || mins < 5 {
		mins = 120
	}
	return time.Duration(mins) * time.Minute
}

// runCycle executes one full availability check for all users.
func (s *AvailabilityScheduler) runCycle() {
	enabled := s.settingsSvc.GetSetting(SettingWishlistCheckEnabled)
	if enabled != "true" {
		s.logger.Debug("scheduler", "Wishlist checking disabled, skipping cycle")
		return
	}

	s.runCycleWithTrigger("scheduled", nil)
}

func (s *AvailabilityScheduler) runCycleWithTrigger(triggerType string, triggerUserID *uint) {
	s.statusMu.Lock()
	s.isRunning = true
	s.statusMu.Unlock()
	defer func() {
		s.statusMu.Lock()
		s.isRunning = false
		s.statusMu.Unlock()
	}()

	s.logger.Info("scheduler", "Starting %s availability check cycle", triggerType)

	coins, err := s.coinRepo.GetAllWishlistWithURLs()
	if err != nil {
		s.logger.Error("scheduler", "Failed to fetch all wishlist coins: %s", err)
		return
	}

	if len(coins) == 0 {
		s.logger.Info("scheduler", "No wishlist coins with URLs found")
		return
	}

	// Group coins by user
	userCoins := make(map[uint]bool)
	for _, coin := range coins {
		userCoins[coin.UserID] = true
	}

	s.logger.Info("scheduler", "Found %d coins across %d users", len(coins), len(userCoins))

	for userID := range userCoins {
		_, err := s.svc.CheckWishlistForUser(userID, triggerType, triggerUserID)
		if err != nil {
			s.logger.Error("scheduler", "%s check failed for user %d: %s", triggerType, userID, err)
		}
	}

	s.logger.Info("scheduler", "%s availability check cycle complete", triggerType)
}
