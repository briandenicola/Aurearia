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
	svc         *AvailabilityService
	coinRepo    *repository.CoinRepository
	availRepo   *repository.AvailabilityRepository
	settingsSvc *SettingsService
	logger      *Logger
	stopCh      chan struct{}
	once        sync.Once
	statusMu    sync.RWMutex
	isRunning   bool
	queue       chan uint
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

// StartWorkers recovers any stale runs from a previous process and starts background
// worker goroutines that process queued manual availability runs.
func (s *AvailabilityScheduler) StartWorkers(workerCount int) {
	if workerCount < 1 {
		workerCount = 1
	}
	if ids, err := s.availRepo.RecoverStaleRuns(availabilityStaleRunTimeout); err == nil {
		for _, id := range ids {
			s.enqueueRunID(id)
		}
	}
	for i := 0; i < workerCount; i++ {
		go s.worker()
	}
}

func (s *AvailabilityScheduler) enqueueRunID(runID uint) {
	select {
	case s.queue <- runID:
	default:
		go func() { s.queue <- runID }()
	}
}

func (s *AvailabilityScheduler) worker() {
	for runID := range s.queue {
		_ = s.processRun(runID)
	}
}

// processRun claims a queued run and executes the manual availability cycle.
func (s *AvailabilityScheduler) processRun(runID uint) error {
	return s.ProcessRun(runID)
}

// ProcessRun claims a queued run and executes the manual availability cycle.
// Exported for use in tests.
func (s *AvailabilityScheduler) ProcessRun(runID uint) error {
	run, claimed, err := s.availRepo.ClaimQueuedRun(runID)
	if err != nil {
		s.logger.Error("scheduler", "Failed to claim availability run %d: %v", runID, err)
		return err
	}
	if !claimed {
		return nil
	}

	s.logger.Info("scheduler", "Processing manual availability run %d", runID)
	if err := s.svc.RunManualCycle(run); err != nil {
		s.logger.Error("scheduler", "Manual availability run %d failed: %v", runID, err)
		if failErr := s.availRepo.FailRun(run, err.Error()); failErr != nil {
			s.logger.Error("scheduler", "Failed to mark run %d as failed: %v", runID, failErr)
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

// RunNowWithTrigger enqueues an immediate availability run and returns the queued run record.
// Returns ErrAvailabilityRunInProgress if a queued or running manual run already exists.
func (s *AvailabilityScheduler) RunNowWithTrigger(triggerUserID *uint) (*models.AvailabilityRun, error) {
	userID := uint(0)
	if triggerUserID != nil {
		userID = *triggerUserID
	}

	run := &models.AvailabilityRun{
		UserID:        userID,
		TriggerType:   "manual",
		TriggerUserID: triggerUserID,
		Status:        models.AvailabilityRunStatusQueued,
		StartedAt:     time.Now(),
	}

	since := time.Now().Add(-availabilityManualRunWindow)
	acquired, err := s.availRepo.EnqueueManualRun(run, since)
	if err != nil {
		return nil, fmt.Errorf("enqueue manual availability run: %w", err)
	}
	if !acquired {
		return nil, ErrAvailabilityRunInProgress
	}

	s.enqueueRunID(run.ID)
	s.logger.Info("scheduler", "Manual availability run %d queued", run.ID)
	return run, nil
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
