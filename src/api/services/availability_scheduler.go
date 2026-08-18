package services

import (
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
)

const (
	availabilityRunQueueSize    = 50
	availabilityManualRunWindow = 5 * time.Minute
	availabilityStaleRunTimeout = 15 * time.Minute
)

var ErrAvailabilityRunInProgress = errors.New("a manual availability run is already queued or running")

// AvailabilityScheduler runs periodic wishlist availability checks.
type AvailabilityScheduler struct {
	svc            *AvailabilityService
	coinRepo       *repository.CoinRepository
	availRepo      *repository.AvailabilityRepository
	availCycleRepo *repository.AvailabilityCycleRepository
	settingsSvc    *SettingsService
	logger         *Logger
	stopCh         chan struct{}
	once           sync.Once
	statusMu       sync.RWMutex
	isRunning      bool
	queue          chan uint
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
		queue:       make(chan uint, availabilityRunQueueSize),
	}
}

// WithCycleRepo attaches the AvailabilityCycleRepository the scheduler needs to enqueue,
// claim, and recover AvailabilityCycle parents.
func (s *AvailabilityScheduler) WithCycleRepo(availCycleRepo *repository.AvailabilityCycleRepository) *AvailabilityScheduler {
	s.availCycleRepo = availCycleRepo
	return s
}

// StartWorkers recovers any stale cycles/children from a previous process and starts
// background worker goroutines that process queued availability cycles.
func (s *AvailabilityScheduler) StartWorkers(workerCount int) {
	if workerCount < 1 {
		workerCount = 1
	}
	// Fail any child run orphaned by a crash mid-execution — this may finalize its parent
	// cycle as a side effect via CompleteChildRun's aggregation.
	s.availRepo.RecoverStaleChildRuns(availabilityStaleRunTimeout)
	if s.availCycleRepo != nil {
		if ids, err := s.availCycleRepo.RecoverStaleCycles(availabilityStaleRunTimeout); err == nil {
			for _, id := range ids {
				s.enqueueCycleID(id)
			}
		}
	}
	for i := 0; i < workerCount; i++ {
		go s.worker()
	}
}

func (s *AvailabilityScheduler) enqueueCycleID(cycleID uint) {
	select {
	case s.queue <- cycleID:
	default:
		go func() { s.queue <- cycleID }()
	}
}

func (s *AvailabilityScheduler) worker() {
	for cycleID := range s.queue {
		_ = s.processCycle(cycleID)
	}
}

// processCycle claims a queued cycle and executes its per-owner fan-out.
func (s *AvailabilityScheduler) processCycle(cycleID uint) error {
	return s.ProcessCycle(cycleID)
}

// ProcessCycle claims a queued availability cycle and executes the per-owner fan-out.
// Exported for use in tests.
func (s *AvailabilityScheduler) ProcessCycle(cycleID uint) error {
	if s.availCycleRepo == nil {
		return fmt.Errorf("availability cycle repository not configured")
	}
	cycle, claimed, err := s.availCycleRepo.ClaimCycle(cycleID)
	if err != nil {
		s.logger.Error("scheduler", "Failed to claim availability cycle %d: %v", cycleID, err)
		return err
	}
	if !claimed {
		return nil
	}

	s.logger.Info("scheduler", "Processing availability cycle %d (%s)", cycleID, cycle.TriggerType)
	if err := s.svc.RunAdminCycle(cycle); err != nil {
		s.logger.Error("scheduler", "Availability cycle %d failed: %v", cycleID, err)
		if failErr := s.availCycleRepo.FinalizeCycle(cycleID, models.AvailabilityCycleStatusFailed, models.GenericAvailabilityFailureMessage); failErr != nil {
			s.logger.Error("scheduler", "Failed to finalize failed cycle %d: %v", cycleID, failErr)
		}
		return err
	}
	return nil
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
	_, err := s.RunNowWithTrigger(nil)
	return err
}

// RunNowWithTrigger enqueues an immediate admin AvailabilityCycle and returns the queued
// cycle record. Returns ErrAvailabilityRunInProgress if a queued or running cycle already
// exists within the duplicate-protection window.
func (s *AvailabilityScheduler) RunNowWithTrigger(triggerUserID *uint) (*models.AvailabilityCycle, error) {
	if s.availCycleRepo == nil {
		return nil, fmt.Errorf("availability cycle repository not configured")
	}

	cycle := &models.AvailabilityCycle{
		TriggerType:   models.AvailabilityRunTriggerAdmin,
		TriggerUserID: triggerUserID,
		Status:        models.AvailabilityCycleStatusQueued,
		StartedAt:     time.Now(),
	}

	since := time.Now().Add(-availabilityManualRunWindow)
	acquired, err := s.availCycleRepo.EnqueueCycle(cycle, since)
	if err != nil {
		return nil, fmt.Errorf("enqueue availability cycle: %w", err)
	}
	if !acquired {
		return nil, ErrAvailabilityRunInProgress
	}

	s.enqueueCycleID(cycle.ID)
	s.logger.Info("scheduler", "Availability cycle %d queued (admin trigger)", cycle.ID)
	return cycle, nil
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

	s.runScheduledCycle()
}

// runScheduledCycle enqueues a scheduled AvailabilityCycle (subject to the same duplicate
// guard as admin-triggered cycles) and processes it synchronously so timeUntilNextRun's
// "last completed" anchor reflects this cycle once it fully finishes — preserving the
// previous scheduled-run-blocking behavior of the Start() loop.
func (s *AvailabilityScheduler) runScheduledCycle() {
	if s.availCycleRepo == nil {
		s.logger.Error("scheduler", "Cannot run scheduled availability cycle: cycle repository not configured")
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

	s.logger.Info("scheduler", "Starting scheduled availability check cycle")

	cycle := &models.AvailabilityCycle{
		TriggerType: models.AvailabilityRunTriggerScheduled,
		Status:      models.AvailabilityCycleStatusQueued,
		StartedAt:   time.Now(),
	}

	since := time.Now().Add(-availabilityManualRunWindow)
	acquired, err := s.availCycleRepo.EnqueueCycle(cycle, since)
	if err != nil {
		s.logger.Error("scheduler", "Failed to enqueue scheduled availability cycle: %v", err)
		return
	}
	if !acquired {
		s.logger.Info("scheduler", "Scheduled availability cycle skipped: a cycle is already queued or running")
		return
	}

	s.logger.Info("scheduler", "Scheduled availability cycle %d queued", cycle.ID)
	if err := s.ProcessCycle(cycle.ID); err != nil {
		s.logger.Error("scheduler", "Scheduled availability cycle %d failed: %v", cycle.ID, err)
	}

	s.logger.Info("scheduler", "Scheduled availability check cycle complete")
}
