package services

import (
	"sync"
	"time"
)

// SetSnapshotScheduler captures daily set valuation snapshots when enabled.
type SetSnapshotScheduler struct {
	setSvc      *SetService
	settingsSvc *SettingsService
	logger      *Logger
	stopCh      chan struct{}
	once        sync.Once
}

func NewSetSnapshotScheduler(setSvc *SetService, settingsSvc *SettingsService, logger *Logger) *SetSnapshotScheduler {
	return &SetSnapshotScheduler{
		setSvc:      setSvc,
		settingsSvc: settingsSvc,
		logger:      logger,
		stopCh:      make(chan struct{}),
	}
}

func (s *SetSnapshotScheduler) Start() {
	s.logger.Info("set-snapshot-scheduler", "Set snapshot scheduler started")
	scheduleLoop{
		category: "set-snapshot-scheduler",
		name:     "set snapshot",
		logger:   s.logger,
		stopCh:   s.stopCh,
		next:     func(now time.Time, _ bool) time.Time { return s.nextRun(now) },
		run: func() {
			if s.settingsSvc.GetSetting(SettingSetSnapshotEnabled) != "true" {
				return
			}
			if err := s.setSvc.CreateSnapshotsForAllUsers(); err != nil {
				s.logger.Error("set-snapshot-scheduler", "Set snapshot cycle failed: %v", err)
			}
		},
	}.Run()
}

func (s *SetSnapshotScheduler) Stop() {
	s.once.Do(func() { close(s.stopCh) })
}

func (s *SetSnapshotScheduler) timeUntilNextRun() time.Duration {
	now := time.Now()
	return s.nextRun(now).Sub(now)
}

func (s *SetSnapshotScheduler) nextRun(now time.Time) time.Time {
	h, m := parseStartTime(s.settingsSvc.GetSetting(SettingSetSnapshotStartTime), 4, 0)
	return dailySchedule{Hour: h, Minute: m, Location: scheduleLocation(s.settingsSvc, s.logger)}.slotAfter(now)
}
