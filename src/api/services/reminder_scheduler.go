package services

import (
	"fmt"
	"sync"
	"time"

	"github.com/briandenicola/ancient-coins-api/repository"
)

// ReminderScheduler runs a daily job that fires in-app (and best-effort Pushover)
// notifications for purchase reminders whose remind_date has arrived.
type ReminderScheduler struct {
	repo        *repository.PurchaseReminderRepository
	notifSvc    *NotificationService
	settingsSvc *SettingsService
	logger      *Logger
	stopCh      chan struct{}
	once        sync.Once
	statusMu    sync.RWMutex
	isRunning   bool
}

// NewReminderScheduler creates a new scheduler.
func NewReminderScheduler(
	repo *repository.PurchaseReminderRepository,
	notifSvc *NotificationService,
	settingsSvc *SettingsService,
	logger *Logger,
) *ReminderScheduler {
	return &ReminderScheduler{
		repo:        repo,
		notifSvc:    notifSvc,
		settingsSvc: settingsSvc,
		logger:      logger,
		stopCh:      make(chan struct{}),
	}
}

// Start begins the daily loop. Call from a goroutine.
func (s *ReminderScheduler) Start() {
	s.logger.Info("scheduler", "Reminder scheduler started")

	// Brief startup delay so the process finishes initializing.
	select {
	case <-time.After(30 * time.Second):
	case <-s.stopCh:
		return
	}

	for {
		wait := s.timeUntilNextRun()
		s.logger.Info("scheduler", "Next reminder check in %s", wait)

		select {
		case <-time.After(wait):
		case <-s.stopCh:
			s.logger.Info("scheduler", "Reminder scheduler stopped")
			return
		}

		s.runCycle()
	}
}

// Stop signals the scheduler to shut down. Safe to call multiple times.
func (s *ReminderScheduler) Stop() {
	s.once.Do(func() { close(s.stopCh) })
}

// RunNow executes one immediate cycle (implements Scheduler interface).
func (s *ReminderScheduler) RunNow() error {
	s.runCycle()
	return nil
}

// GetStatus returns the scheduler runtime status.
func (s *ReminderScheduler) GetStatus() SchedulerStatus {
	s.statusMu.RLock()
	running := s.isRunning
	s.statusMu.RUnlock()

	enabled := s.settingsSvc.GetSetting(SettingReminderCheckEnabled) == "true"
	return SchedulerStatus{
		Name:      "Reminder Check",
		Enabled:   enabled,
		IsRunning: running,
		NextRunIn: s.timeUntilNextRun(),
	}
}

// timeUntilNextRun returns the duration until the next daily anchor (HH:MM).
func (s *ReminderScheduler) timeUntilNextRun() time.Duration {
	now := time.Now()
	h, m := s.getStartTime()
	anchor := time.Date(now.Year(), now.Month(), now.Day(), h, m, 0, 0, now.Location())
	if anchor.After(now) {
		return anchor.Sub(now)
	}
	return anchor.Add(24 * time.Hour).Sub(now)
}

// getStartTime parses HH:MM from settings, defaulting to 08:00.
func (s *ReminderScheduler) getStartTime() (int, int) {
	raw := s.settingsSvc.GetSetting(SettingReminderCheckStartTime)
	var h, m int
	if _, err := fmt.Sscanf(raw, "%d:%d", &h, &m); err != nil || h < 0 || h > 23 || m < 0 || m > 59 {
		return 8, 0
	}
	return h, m
}

// runCycle processes all pending reminders whose remind_date has arrived.
func (s *ReminderScheduler) runCycle() {
	if s.settingsSvc.GetSetting(SettingReminderCheckEnabled) != "true" {
		s.logger.Debug("scheduler", "Reminder check disabled, skipping cycle")
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

	reminders, err := s.repo.ListDueReminders()
	if err != nil {
		s.logger.Error("scheduler", "Reminder scheduler: failed to list pending reminders: %v", err)
		return
	}

	due := 0
	notified := 0
	skipped := 0

	for _, r := range reminders {
		if s.isDue(r.RemindDate, r.Timezone) {
			due++
			coinName := r.Coin.Name
			if coinName == "" {
				coinName = "Unnamed coin"
			}
			marked, err := s.repo.MarkNotified(r.ID)
			if err != nil {
				s.logger.Error("scheduler", "Reminder %d: failed to mark notified: %v", r.ID, err)
				continue
			}
			if !marked {
				// Already transitioned since we fetched the list.
				skipped++
				continue
			}
			s.notifSvc.NotifyPurchaseReminder(r.UserID, r.ID, r.CoinID, coinName)
			s.logger.Info("scheduler", "Reminder %d notified for user %d (coin %d: %s)", r.ID, r.UserID, r.CoinID, coinName)
			notified++
		} else {
			skipped++
		}
	}

	s.logger.Info("scheduler", "Reminder scheduler cycle complete — %d due, %d notified, %d skipped",
		due, notified, skipped)
}

// isDue returns true if remindDate <= today in the given IANA timezone.
// String comparison is valid for YYYY-MM-DD because lexicographic order
// matches chronological order.
func (s *ReminderScheduler) isDue(remindDate, timezone string) bool {
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		// Unknown timezone — treat as due to prevent silently stuck reminders.
		s.logger.Warn("scheduler", "Reminder has unrecognized timezone %q; treating as due", timezone)
		return true
	}
	today := time.Now().In(loc).Format("2006-01-02")
	return remindDate <= today
}
