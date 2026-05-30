package services

import (
	"fmt"
	"sync"
	"time"
)

// CollectionHealthScheduler persists daily collection health snapshots.
type CollectionHealthScheduler struct {
	svc         *HealthService
	settingsSvc *SettingsService
	logger      *Logger
	stopCh      chan struct{}
	once        sync.Once
	statusMu    sync.RWMutex
	isRunning   bool
}

// NewCollectionHealthScheduler creates a new collection health snapshot scheduler.
func NewCollectionHealthScheduler(svc *HealthService, settingsSvc *SettingsService, logger *Logger) *CollectionHealthScheduler {
	return &CollectionHealthScheduler{
		svc:         svc,
		settingsSvc: settingsSvc,
		logger:      logger,
		stopCh:      make(chan struct{}),
	}
}

// Start begins the periodic daily loop.
func (s *CollectionHealthScheduler) Start() {
	s.logger.Info("health-scheduler", "Collection health scheduler started")

	select {
	case <-time.After(45 * time.Second):
	case <-s.stopCh:
		return
	}

	for {
		wait := s.timeUntilNextRun()
		s.logger.Info("health-scheduler", "Next health snapshot in %s", wait)

		select {
		case <-time.After(wait):
		case <-s.stopCh:
			s.logger.Info("health-scheduler", "Collection health scheduler stopped")
			return
		}

		s.runCycle()
	}
}

// Stop signals the scheduler to shut down.
func (s *CollectionHealthScheduler) Stop() {
	s.once.Do(func() { close(s.stopCh) })
}

// RunNow executes one immediate snapshot cycle.
func (s *CollectionHealthScheduler) RunNow() error {
	s.runCycleWithTrigger("manual")
	return nil
}

// GetStatus returns scheduler runtime status.
func (s *CollectionHealthScheduler) GetStatus() SchedulerStatus {
	s.statusMu.RLock()
	running := s.isRunning
	s.statusMu.RUnlock()

	enabled := s.settingsSvc.GetSetting(SettingCollectionHealthSnapshotsEnabled) == "true"
	return SchedulerStatus{
		Name:      "collection-health",
		Enabled:   enabled,
		IsRunning: running,
		NextRunIn: s.timeUntilNextRun(),
	}
}

func (s *CollectionHealthScheduler) timeUntilNextRun() time.Duration {
	now := time.Now()
	hour, minute := s.getStartTime()
	next := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location())
	if !next.After(now) {
		next = next.Add(24 * time.Hour)
	}
	return next.Sub(now)
}

func (s *CollectionHealthScheduler) getStartTime() (int, int) {
	raw := s.settingsSvc.GetSetting(SettingCollectionHealthSnapshotsStartTime)
	var h, m int
	if _, err := fmt.Sscanf(raw, "%d:%d", &h, &m); err != nil || h < 0 || h > 23 || m < 0 || m > 59 {
		return 4, 30
	}
	return h, m
}

func (s *CollectionHealthScheduler) runCycle() {
	if s.settingsSvc.GetSetting(SettingCollectionHealthSnapshotsEnabled) != "true" {
		s.logger.Debug("health-scheduler", "Collection health snapshots disabled, skipping cycle")
		return
	}
	s.runCycleWithTrigger("scheduled")
}

func (s *CollectionHealthScheduler) runCycleWithTrigger(triggerType string) {
	s.statusMu.Lock()
	s.isRunning = true
	s.statusMu.Unlock()
	defer func() {
		s.statusMu.Lock()
		s.isRunning = false
		s.statusMu.Unlock()
	}()

	started := time.Now()
	userIDs, err := s.svc.repo.ListUsersWithEligibleCoins()
	if err != nil {
		s.logger.Error("health-scheduler", "Failed to fetch eligible users: %v", err)
		return
	}

	snapshotDate := time.Date(started.Year(), started.Month(), started.Day(), 0, 0, 0, 0, started.Location())
	successes := 0
	for _, userID := range userIDs {
		if err := s.svc.SnapshotUserHealth(userID, snapshotDate); err != nil {
			s.logger.Error("health-scheduler", "Failed to snapshot user %d (%s): %v", userID, triggerType, err)
			continue
		}
		successes++
	}

	s.logger.Info("health-scheduler", "%s cycle complete in %s (%d/%d users snapped)", triggerType, time.Since(started), successes, len(userIDs))
}
