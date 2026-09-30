package services

import (
	"sync"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
)

// CollectionHealthScheduler persists daily collection health snapshots.
type CollectionHealthScheduler struct {
	svc         *HealthService
	runRepo     *repository.CollectionHealthSnapshotRunRepository
	settingsSvc *SettingsService
	logger      *Logger
	stopCh      chan struct{}
	once        sync.Once
	statusMu    sync.RWMutex
	isRunning   bool
}

// NewCollectionHealthScheduler creates a new collection health snapshot scheduler.
func NewCollectionHealthScheduler(svc *HealthService, runRepo *repository.CollectionHealthSnapshotRunRepository, settingsSvc *SettingsService, logger *Logger) *CollectionHealthScheduler {
	return &CollectionHealthScheduler{
		svc:         svc,
		runRepo:     runRepo,
		settingsSvc: settingsSvc,
		logger:      logger,
		stopCh:      make(chan struct{}),
	}
}

// ListRuns returns paginated collection health snapshot run history.
func (s *CollectionHealthScheduler) ListRuns(page, limit int) ([]models.CollectionHealthSnapshotRun, int64, error) {
	return s.runRepo.ListRuns(page, limit)
}

// Start begins the periodic daily loop.
func (s *CollectionHealthScheduler) Start() {
	s.logger.Info("health-scheduler", "Collection health scheduler started")
	scheduleLoop{
		category:     "health-scheduler",
		name:         "health snapshot",
		logger:       s.logger,
		stopCh:       s.stopCh,
		initialDelay: 45 * time.Second,
		next:         func(now time.Time, _ bool) time.Time { return s.nextRun(now) },
		run:          s.runCycle,
	}.Run()
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
	return s.nextRun(now).Sub(now)
}

func (s *CollectionHealthScheduler) nextRun(now time.Time) time.Time {
	h, m := s.getStartTime()
	return dailySchedule{Hour: h, Minute: m, Location: scheduleLocation(s.settingsSvc, s.logger)}.slotAfter(now)
}

func (s *CollectionHealthScheduler) getStartTime() (int, int) {
	return parseStartTime(s.settingsSvc.GetSetting(SettingCollectionHealthSnapshotsStartTime), 4, 30)
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
	run := &models.CollectionHealthSnapshotRun{
		TriggerType: triggerType,
		Status:      "running",
		StartedAt:   started,
	}
	if err := s.runRepo.CreateRun(run); err != nil {
		s.logger.Error("health-scheduler", "Failed to create collection health snapshot run: %v", err)
		return
	}

	userIDs, err := s.svc.repo.ListUsersWithEligibleCoins()
	if err != nil {
		s.logger.Error("health-scheduler", "Failed to fetch eligible users: %v", err)
		completedAt := time.Now()
		run.Status = "error"
		run.ErrorMessage = err.Error()
		run.CompletedAt = &completedAt
		run.DurationMs = completedAt.Sub(started).Milliseconds()
		if completeErr := s.runRepo.CompleteRun(run); completeErr != nil {
			s.logger.Error("health-scheduler", "Failed to complete collection health snapshot run: %v", completeErr)
		}
		return
	}

	snapshotDate := time.Date(started.Year(), started.Month(), started.Day(), 0, 0, 0, 0, started.Location())
	successes := 0
	failures := 0
	for _, userID := range userIDs {
		if err := s.svc.SnapshotUserHealth(userID, snapshotDate); err != nil {
			s.logger.Error("health-scheduler", "Failed to snapshot user %d (%s): %v", userID, triggerType, err)
			failures++
			continue
		}
		successes++
	}

	completedAt := time.Now()
	run.UsersEligible = len(userIDs)
	run.UsersSnapshotted = successes
	run.UsersFailed = failures
	run.Status = "success"
	if failures > 0 && successes == 0 && len(userIDs) > 0 {
		run.Status = "error"
	}
	run.CompletedAt = &completedAt
	run.DurationMs = completedAt.Sub(started).Milliseconds()
	if err := s.runRepo.CompleteRun(run); err != nil {
		s.logger.Error("health-scheduler", "Failed to complete collection health snapshot run: %v", err)
	}

	s.logger.Info("health-scheduler", "%s cycle complete in %s (%d/%d users snapped)", triggerType, time.Since(started), successes, len(userIDs))
}
