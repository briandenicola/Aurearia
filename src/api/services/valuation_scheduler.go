package services

import (
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/briandenicola/ancient-coins-api/repository"
)

// ValuationScheduler runs periodic collection valuation checks.
type ValuationScheduler struct {
	svc         *ValuationService
	coinRepo    *repository.CoinRepository
	valRepo     *repository.ValuationRepository
	settingsSvc *SettingsService
	logger      *Logger
	stopCh      chan struct{}
	once        sync.Once
	statusMu    sync.RWMutex
	isRunning   bool
}

// NewValuationScheduler creates a new scheduler.
func NewValuationScheduler(
	svc *ValuationService,
	coinRepo *repository.CoinRepository,
	valRepo *repository.ValuationRepository,
	settingsSvc *SettingsService,
	logger *Logger,
) *ValuationScheduler {
	return &ValuationScheduler{
		svc:         svc,
		coinRepo:    coinRepo,
		valRepo:     valRepo,
		settingsSvc: settingsSvc,
		logger:      logger,
		stopCh:      make(chan struct{}),
	}
}

// Start begins the periodic valuation loop. Call from a goroutine.
func (s *ValuationScheduler) Start() {
	s.logger.Info("valuation-scheduler", "Collection valuation scheduler started")

	// Initial delay to let the app finish startup
	select {
	case <-time.After(60 * time.Second):
	case <-s.stopCh:
		return
	}

	enabled := s.settingsSvc.GetSetting(SettingValuationCheckEnabled)
	startTime := s.settingsSvc.GetSetting(SettingValuationCheckStartTime)
	intervalDays := s.settingsSvc.GetSetting(SettingValuationCheckInterval)
	s.logger.Info("valuation-scheduler", "Settings — enabled: %s, startTime: %s, intervalDays: %s", enabled, startTime, intervalDays)

	for {
		wait := s.timeUntilNextRun()
		s.logger.Info("valuation-scheduler", "Next valuation check in %s", wait)

		select {
		case <-time.After(wait):
		case <-s.stopCh:
			s.logger.Info("valuation-scheduler", "Scheduler stopped")
			return
		}

		s.runCycle()
	}
}

// Stop signals the scheduler to shut down. Safe to call multiple times.
func (s *ValuationScheduler) Stop() {
	s.once.Do(func() { close(s.stopCh) })
}

// RunNow executes one immediate manual valuation cycle for all users.
func (s *ValuationScheduler) RunNow() error {
	s.runCycleWithTrigger("manual", nil)
	return nil
}

// GetStatus returns the scheduler runtime status.
func (s *ValuationScheduler) GetStatus() SchedulerStatus {
	s.statusMu.RLock()
	running := s.isRunning
	s.statusMu.RUnlock()

	enabled := s.settingsSvc.GetSetting(SettingValuationCheckEnabled) == "true"
	return SchedulerStatus{
		Name:      "valuation",
		Enabled:   enabled,
		IsRunning: running,
		NextRunIn: s.timeUntilNextRun(),
	}
}

// timeUntilNextRun calculates delay until the next scheduled run.
// If there is a previous completed scheduled run, uses that as the anchor so
// app restarts don't reset the schedule. Falls back to the start-time based
// calculation only when no run history exists.
func (s *ValuationScheduler) timeUntilNextRun() time.Duration {
	now := time.Now()
	intervalDays := s.getIntervalDays()
	interval := time.Duration(intervalDays) * 24 * time.Hour

	// Check last completed scheduled run
	lastRun := s.valRepo.GetLastScheduledRun()
	if lastRun != nil && lastRun.CompletedAt != nil {
		nextFromLast := lastRun.CompletedAt.Add(interval)
		if nextFromLast.Before(now) {
			// Overdue — run immediately
			s.logger.Info("valuation-scheduler", "Last scheduled run completed %s ago, overdue — running now", now.Sub(*lastRun.CompletedAt).Round(time.Minute))
			return 0
		}
		return nextFromLast.Sub(now)
	}

	// No previous run — use today's start time as anchor
	startHour, startMin := s.getStartTime()
	anchor := time.Date(now.Year(), now.Month(), now.Day(), startHour, startMin, 0, 0, now.Location())

	if anchor.After(now) {
		return anchor.Sub(now)
	}

	// Find the next occurrence after now
	elapsed := now.Sub(anchor)
	periods := int(elapsed/interval) + 1
	next := anchor.Add(time.Duration(periods) * interval)
	return next.Sub(now)
}

// getStartTime parses HH:MM from settings, defaults to 03:00.
func (s *ValuationScheduler) getStartTime() (int, int) {
	raw := s.settingsSvc.GetSetting(SettingValuationCheckStartTime)
	var h, m int
	if _, err := fmt.Sscanf(raw, "%d:%d", &h, &m); err != nil || h < 0 || h > 23 || m < 0 || m > 59 {
		return 3, 0
	}
	return h, m
}

// getIntervalDays returns the configured check interval in days.
func (s *ValuationScheduler) getIntervalDays() int {
	dayStr := s.settingsSvc.GetSetting(SettingValuationCheckInterval)
	days, err := strconv.Atoi(dayStr)
	if err != nil || days < 1 {
		days = 7
	}
	return days
}

// runCycle executes one full valuation check for all users with owned coins.
func (s *ValuationScheduler) runCycle() {
	enabled := s.settingsSvc.GetSetting(SettingValuationCheckEnabled)
	if enabled != "true" {
		s.logger.Info("valuation-scheduler", "Collection valuation disabled, skipping cycle")
		return
	}

	s.runCycleWithTrigger("scheduled", nil)
}

func (s *ValuationScheduler) runCycleWithTrigger(triggerType string, triggerUserID *uint) {
	s.statusMu.Lock()
	s.isRunning = true
	s.statusMu.Unlock()
	defer func() {
		s.statusMu.Lock()
		s.isRunning = false
		s.statusMu.Unlock()
	}()

	s.logger.Info("valuation-scheduler", "Starting %s valuation cycle", triggerType)

	// Get distinct user IDs that have owned coins
	userIDs, err := s.svc.valRepo.GetUsersWithOwnedCoins()
	if err != nil {
		s.logger.Error("valuation-scheduler", "Failed to fetch users: %s", err)
		return
	}

	if len(userIDs) == 0 {
		s.logger.Info("valuation-scheduler", "No users with owned coins found")
		return
	}

	s.logger.Info("valuation-scheduler", "Found %d users with owned coins", len(userIDs))

	for _, userID := range userIDs {
		_, err := s.svc.ValuateCollectionForUser(userID, triggerType, triggerUserID)
		if err != nil {
			s.logger.Error("valuation-scheduler", "%s valuation failed for user %d: %s", triggerType, userID, err)
		}
	}

	s.logger.Info("valuation-scheduler", "%s valuation cycle complete", triggerType)
}
