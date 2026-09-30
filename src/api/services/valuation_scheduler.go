package services

import (
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

	scheduleLoop{
		category: "valuation-scheduler",
		name:     "valuation check",
		logger:   s.logger,
		stopCh:   s.stopCh,
		next:     s.nextRun,
		run:      s.runCycle,
	}.Run()
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

// timeUntilNextRun returns the delay until the next scheduled run.
func (s *ValuationScheduler) timeUntilNextRun() time.Duration {
	now := time.Now()
	return max(s.nextRun(now, false).Sub(now), 0)
}

// nextRun returns the start time every N days after the last completed
// scheduled run (so restarts don't reset the schedule), in the schedule zone.
func (s *ValuationScheduler) nextRun(now time.Time, catchUp bool) time.Time {
	var last *time.Time
	if run := s.valRepo.GetLastScheduledRun(); run != nil {
		last = run.CompletedAt
	}
	h, m := s.getStartTime()
	schedule := dailySchedule{
		Hour:     h,
		Minute:   m,
		Interval: time.Duration(s.getIntervalDays()) * 24 * time.Hour,
		Location: scheduleLocation(s.settingsSvc, s.logger),
	}
	return schedule.next(now, last, catchUp)
}

// getStartTime parses HH:MM from settings, defaults to 03:00.
func (s *ValuationScheduler) getStartTime() (int, int) {
	return parseStartTime(s.settingsSvc.GetSetting(SettingValuationCheckStartTime), 3, 0)
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
