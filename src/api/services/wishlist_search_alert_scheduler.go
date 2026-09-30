package services

import (
	"sync"
	"time"

	"github.com/briandenicola/ancient-coins-api/repository"
)

// WishlistSearchAlertScheduler runs a daily sweep that enqueues scheduled runs
// for WishlistSearchAlerts whose per-alert cadence (daily/weekly/monthly) has
// elapsed since their last run. Cadence was previously stored but never acted
// on, so alerts with an automatic cadence never ran again after their first
// manual "Run Now" (see issue #483).
type WishlistSearchAlertScheduler struct {
	alertSvc    *WishlistSearchAlertService
	repo        *repository.WishlistSearchAlertRepository
	settingsSvc *SettingsService
	logger      *Logger
	stopCh      chan struct{}
	once        sync.Once
	statusMu    sync.RWMutex
	isRunning   bool
}

// NewWishlistSearchAlertScheduler creates a new scheduler.
func NewWishlistSearchAlertScheduler(
	alertSvc *WishlistSearchAlertService,
	repo *repository.WishlistSearchAlertRepository,
	settingsSvc *SettingsService,
	logger *Logger,
) *WishlistSearchAlertScheduler {
	return &WishlistSearchAlertScheduler{
		alertSvc:    alertSvc,
		repo:        repo,
		settingsSvc: settingsSvc,
		logger:      logger,
		stopCh:      make(chan struct{}),
	}
}

// Start begins the daily sweep loop. Call from a goroutine.
func (s *WishlistSearchAlertScheduler) Start() {
	s.logger.Info("scheduler", "Wishlist search alert scheduler started")
	scheduleLoop{
		category:     "scheduler",
		name:         "wishlist search alert sweep",
		logger:       s.logger,
		stopCh:       s.stopCh,
		initialDelay: 30 * time.Second,
		next:         func(now time.Time, _ bool) time.Time { return s.nextRun(now) },
		run:          s.runCycle,
	}.Run()
}

// Stop signals the scheduler to shut down. Safe to call multiple times.
func (s *WishlistSearchAlertScheduler) Stop() {
	s.once.Do(func() { close(s.stopCh) })
}

// RunNow executes one immediate sweep of due alerts.
func (s *WishlistSearchAlertScheduler) RunNow() error {
	s.runCycle()
	return nil
}

// GetStatus returns the scheduler runtime status.
func (s *WishlistSearchAlertScheduler) GetStatus() SchedulerStatus {
	s.statusMu.RLock()
	running := s.isRunning
	s.statusMu.RUnlock()

	enabled := s.settingsSvc.GetSetting(SettingWishlistSearchAlertsCheckEnabled) == "true"
	return SchedulerStatus{
		Name:      "wishlist_search_alerts",
		Enabled:   enabled,
		IsRunning: running,
		NextRunIn: s.timeUntilNextRun(),
	}
}

// timeUntilNextRun calculates the delay until the next daily sweep, anchored
// to the configured start time.
func (s *WishlistSearchAlertScheduler) timeUntilNextRun() time.Duration {
	now := time.Now()
	return s.nextRun(now).Sub(now)
}

func (s *WishlistSearchAlertScheduler) nextRun(now time.Time) time.Time {
	h, m := s.getStartTime()
	return dailySchedule{Hour: h, Minute: m, Location: scheduleLocation(s.settingsSvc, s.logger)}.slotAfter(now)
}

// getStartTime parses HH:MM from settings, defaults to 03:00.
func (s *WishlistSearchAlertScheduler) getStartTime() (int, int) {
	return parseStartTime(s.settingsSvc.GetSetting(SettingWishlistSearchAlertsCheckStartTime), 3, 0)
}

// runCycle finds every active alert whose cadence interval has elapsed and
// enqueues a scheduled discovery run for it.
func (s *WishlistSearchAlertScheduler) runCycle() {
	enabled := s.settingsSvc.GetSetting(SettingWishlistSearchAlertsCheckEnabled)
	if enabled != "true" {
		s.logger.Debug("scheduler", "Wishlist search alert scheduled checks disabled, skipping cycle")
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

	alerts, err := s.repo.GetDueAlerts(time.Now())
	if err != nil {
		s.logger.Error("scheduler", "Failed to fetch due wishlist search alerts: %s", err)
		return
	}

	if len(alerts) == 0 {
		s.logger.Info("scheduler", "No wishlist search alerts due for a scheduled run")
		return
	}

	s.logger.Info("scheduler", "Found %d wishlist search alerts due for a scheduled run", len(alerts))
	for _, alert := range alerts {
		if _, err := s.alertSvc.QueueScheduledRun(alert); err != nil {
			s.logger.Error("scheduler", "Failed to queue scheduled run for alert %d: %s", alert.ID, err)
		}
	}
}
