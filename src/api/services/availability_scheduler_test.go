package services

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// setupAvailSchedulerDB creates an in-memory SQLite DB with required tables.
func setupAvailSchedulerDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	if err := db.AutoMigrate(&models.AppSetting{}, &models.User{}, &models.AvailabilityRun{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	return db
}

// newTestAvailabilityScheduler builds a minimal scheduler for timing tests.
func newTestAvailabilityScheduler(t *testing.T, db *gorm.DB) *AvailabilityScheduler {
	t.Helper()
	settingsRepo := repository.NewSettingsRepository(db)
	settingsSvc := NewSettingsService(settingsRepo)
	availRepo := repository.NewAvailabilityRepository(db)
	return NewAvailabilityScheduler(nil, nil, availRepo, settingsSvc, NewLogger(100))
}

// TestTimeUntilNextRun_NoHistory verifies anchor-based scheduling when no run
// history exists.
func TestTimeUntilNextRun_NoHistory(t *testing.T) {
	db := setupAvailSchedulerDB(t)
	s := newTestAvailabilityScheduler(t, db)

	// Configure start time to 2 hours in the future, interval = 1440 min.
	settingsSvc := s.settingsSvc
	future := time.Now().Add(2 * time.Hour)
	startTime := future.Format("15:04")
	if err := settingsSvc.SetSetting(SettingWishlistCheckStartTime, startTime); err != nil {
		t.Fatalf("failed to set start time: %v", err)
	}
	if err := settingsSvc.SetSetting(SettingWishlistCheckInterval, "1440"); err != nil {
		t.Fatalf("failed to set interval: %v", err)
	}

	wait := s.timeUntilNextRun()

	// Should wait approximately 2 hours (within a few seconds of tolerance).
	if wait < 1*time.Hour+55*time.Minute || wait > 2*time.Hour+5*time.Minute {
		t.Errorf("expected ~2h wait, got %v", wait)
	}
}

// TestTimeUntilNextRun_UsesLastRun verifies that the interval is measured from
// the most recent completed scheduled run, not recalculated from today's anchor.
func TestTimeUntilNextRun_UsesLastRun(t *testing.T) {
	db := setupAvailSchedulerDB(t)
	s := newTestAvailabilityScheduler(t, db)

	// Set interval to 1440 minutes.
	if err := s.settingsSvc.SetSetting(SettingWishlistCheckInterval, "1440"); err != nil {
		t.Fatalf("failed to set interval: %v", err)
	}

	// Seed a completed scheduled run that happened 60 minutes ago.
	sixtyMinsAgo := time.Now().Add(-60 * time.Minute)
	completedAt := time.Now().Add(-59 * time.Minute)
	user := models.User{Username: "testuser"}
	db.Create(&user)
	run := &models.AvailabilityRun{
		UserID:      user.ID,
		TriggerType: "scheduled",
		StartedAt:   sixtyMinsAgo,
		CompletedAt: &completedAt,
	}
	if err := db.Create(run).Error; err != nil {
		t.Fatalf("failed to seed run: %v", err)
	}

	wait := s.timeUntilNextRun()

	// With a last run 60 minutes ago and 1440-minute interval, the next run
	// should be in approximately 1380 minutes (~23 hours).
	expectedMin := 1379 * time.Minute
	expectedMax := 1381 * time.Minute
	if wait < expectedMin || wait > expectedMax {
		t.Errorf("expected wait ~1380m, got %v", wait)
	}
}

// TestTimeUntilNextRun_Overdue verifies that when the last run is further back
// than the configured interval, the scheduler returns 0 to run immediately.
func TestTimeUntilNextRun_Overdue(t *testing.T) {
	db := setupAvailSchedulerDB(t)
	s := newTestAvailabilityScheduler(t, db)

	// Set interval to 60 minutes.
	if err := s.settingsSvc.SetSetting(SettingWishlistCheckInterval, "60"); err != nil {
		t.Fatalf("failed to set interval: %v", err)
	}

	// Seed a run that completed 120 minutes ago — clearly overdue.
	twoHoursAgo := time.Now().Add(-120 * time.Minute)
	completedAt := time.Now().Add(-119 * time.Minute)
	user := models.User{Username: "testuser2"}
	db.Create(&user)
	run := &models.AvailabilityRun{
		UserID:      user.ID,
		TriggerType: "scheduled",
		StartedAt:   twoHoursAgo,
		CompletedAt: &completedAt,
	}
	if err := db.Create(run).Error; err != nil {
		t.Fatalf("failed to seed run: %v", err)
	}

	wait := s.timeUntilNextRun()

	if wait != 0 {
		t.Errorf("expected 0 (immediate) for overdue run, got %v", wait)
	}
}

// TestTimeUntilNextRun_IgnoresManualRuns verifies that manual runs are not
// counted as the scheduling anchor.
func TestTimeUntilNextRun_IgnoresManualRuns(t *testing.T) {
	db := setupAvailSchedulerDB(t)
	s := newTestAvailabilityScheduler(t, db)

	// Set interval to 1440 minutes, start time 2 hours in the future.
	future := time.Now().Add(2 * time.Hour)
	startTime := future.Format("15:04")
	if err := s.settingsSvc.SetSetting(SettingWishlistCheckStartTime, startTime); err != nil {
		t.Fatalf("failed to set start time: %v", err)
	}
	if err := s.settingsSvc.SetSetting(SettingWishlistCheckInterval, "1440"); err != nil {
		t.Fatalf("failed to set interval: %v", err)
	}

	// Seed only a MANUAL run (not scheduled) 5 minutes ago.
	fiveMinsAgo := time.Now().Add(-5 * time.Minute)
	completedAt := time.Now().Add(-4 * time.Minute)
	user := models.User{Username: "testuser3"}
	db.Create(&user)
	run := &models.AvailabilityRun{
		UserID:      user.ID,
		TriggerType: "manual", // not scheduled
		StartedAt:   fiveMinsAgo,
		CompletedAt: &completedAt,
	}
	if err := db.Create(run).Error; err != nil {
		t.Fatalf("failed to seed run: %v", err)
	}

	wait := s.timeUntilNextRun()

	// Should still wait ~2h (falls back to anchor-based calculation because no
	// scheduled run exists).
	if wait < 1*time.Hour+55*time.Minute || wait > 2*time.Hour+5*time.Minute {
		t.Errorf("expected ~2h wait (ignoring manual run), got %v", wait)
	}
}

// setupAvailSchedulerWithService creates a full scheduler + cycle repository backed by an
// in-memory DB, wired exactly like main.go (availRepo.WithCycleRepo + scheduler.WithCycleRepo),
// suitable for cycle-based async processing tests (Feature 353).
func setupAvailSchedulerWithService(t *testing.T, listingURL string) (*AvailabilityScheduler, *repository.AvailabilityCycleRepository, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	// StartWorkers processes cycles on a background goroutine while the test polls the same
	// DB from the main goroutine; pin the pool to one connection so it can't land on a second,
	// unmigrated ":memory:" instance (same pattern used for Pushover-goroutine tests).
	if sqlDB, err := db.DB(); err == nil {
		sqlDB.SetMaxOpenConns(1)
	}
	if err := db.AutoMigrate(
		&models.User{},
		&models.Coin{},
		&models.CoinImage{},
		&models.AvailabilityCycle{},
		&models.AvailabilityRun{},
		&models.AvailabilityResult{},
		&models.AppSetting{},
		&models.Notification{},
	); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	owner := models.User{Username: "owner", Email: "owner@test.com"}
	db.Create(&owner)
	db.Create(&models.Coin{
		UserID:       owner.ID,
		Name:         "Test Coin",
		ReferenceURL: listingURL,
		IsWishlist:   true,
	})

	coinRepo := repository.NewCoinRepository(db)
	availRepo := repository.NewAvailabilityRepository(db)
	availCycleRepo := repository.NewAvailabilityCycleRepository(db)
	availRepo.WithCycleRepo(availCycleRepo)
	settingsRepo := repository.NewSettingsRepository(db)
	settingsSvc := NewSettingsService(settingsRepo)
	logger := NewLogger(100)
	availSvc := NewAvailabilityService(coinRepo, availRepo, nil, nil, nil, nil, settingsSvc, logger).WithCycleRepo(availCycleRepo)
	scheduler := NewAvailabilityScheduler(availSvc, coinRepo, availRepo, settingsSvc, logger).WithCycleRepo(availCycleRepo)
	return scheduler, availCycleRepo, db
}

// TestAvailabilityScheduler_RunNowEnqueuesWithoutBlocking verifies that RunNowWithTrigger
// returns immediately with a queued cycle and does NOT process any coins synchronously.
func TestAvailabilityScheduler_RunNowEnqueuesWithoutBlocking(t *testing.T) {
	agentCalled := false
	listing := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		agentCalled = true
		t.Fatal("listing server should not be called until worker processes the queued cycle")
	}))
	defer listing.Close()

	scheduler, _, _ := setupAvailSchedulerWithService(t, listing.URL)

	triggerID := uint(1)
	cycle, err := scheduler.RunNowWithTrigger(&triggerID)
	if err != nil {
		t.Fatalf("RunNowWithTrigger: %v", err)
	}
	if cycle.Status != models.AvailabilityCycleStatusQueued {
		t.Fatalf("expected queued status, got %q", cycle.Status)
	}
	if cycle.TriggerType != models.AvailabilityRunTriggerAdmin {
		t.Fatalf("expected admin trigger type, got %q", cycle.TriggerType)
	}
	if agentCalled {
		t.Fatal("listing server was called synchronously during RunNowWithTrigger")
	}
}

// TestAvailabilityScheduler_DuplicateRunBlocked verifies that a second RunNowWithTrigger
// call is rejected (409-equivalent sentinel) while a queued or running cycle exists
// (FR-007, US2 AC2 — T016).
func TestAvailabilityScheduler_DuplicateRunBlocked(t *testing.T) {
	scheduler, _, _ := setupAvailSchedulerWithService(t, "https://example.test/coin")

	id := uint(1)
	if _, err := scheduler.RunNowWithTrigger(&id); err != nil {
		t.Fatalf("first enqueue: %v", err)
	}
	_, err := scheduler.RunNowWithTrigger(&id)
	if err == nil {
		t.Fatal("expected error for duplicate cycle, got nil")
	}
	if err != ErrAvailabilityRunInProgress {
		t.Fatalf("expected ErrAvailabilityRunInProgress, got %v", err)
	}
}

// TestAvailabilityScheduler_ProcessCycle_Completed verifies that ProcessCycle claims a
// queued cycle, fans out one child run per wishlist owner, and marks both the child and
// the parent cycle completed.
func TestAvailabilityScheduler_ProcessCycle_Completed(t *testing.T) {
	listing := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<html><body><p>Add to Cart</p></body></html>`))
	}))
	defer listing.Close()

	scheduler, availCycleRepo, db := setupAvailSchedulerWithService(t, listing.URL)

	triggerID := uint(1)
	cycle, err := scheduler.RunNowWithTrigger(&triggerID)
	if err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	if err := scheduler.ProcessCycle(cycle.ID); err != nil {
		t.Fatalf("ProcessCycle: %v", err)
	}

	var completedCycle models.AvailabilityCycle
	if err := db.First(&completedCycle, cycle.ID).Error; err != nil {
		t.Fatalf("load cycle: %v", err)
	}
	if completedCycle.Status != models.AvailabilityCycleStatusCompleted {
		t.Fatalf("expected cycle completed, got %q", completedCycle.Status)
	}
	if completedCycle.CompletedChildren != 1 || completedCycle.TotalChildren < 1 {
		t.Fatalf("expected 1 completed child, got completed=%d total=%d", completedCycle.CompletedChildren, completedCycle.TotalChildren)
	}

	var childRuns []models.AvailabilityRun
	db.Where("cycle_id = ?", cycle.ID).Find(&childRuns)
	if len(childRuns) != 1 {
		t.Fatalf("expected exactly 1 child run, got %d", len(childRuns))
	}
	child := childRuns[0]
	if child.Status != models.AvailabilityRunStatusCompleted {
		t.Fatalf("expected child completed, got %q", child.Status)
	}
	if child.UserID == 0 {
		t.Fatal("expected child run UserID > 0")
	}
	if child.CoinsChecked != 1 || child.Available != 1 {
		t.Fatalf("expected 1 coin checked/available, got checked=%d available=%d", child.CoinsChecked, child.Available)
	}

	// Verify result record was created
	var results []models.AvailabilityResult
	db.Where("run_id = ?", child.ID).Find(&results)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	_ = availCycleRepo
}

// TestAvailabilityScheduler_StaleChildRunRecovery verifies that RecoverStaleChildRuns fails
// orphaned "running" child runs from a previous crashed process and finalizes their parent
// cycle as a side effect (there is no queued state for a child to resume from — Feature 353
// replaces the old queued-manual-run recovery model with fail-and-aggregate for children).
func TestAvailabilityScheduler_StaleChildRunRecovery(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("db: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.Coin{}, &models.CoinImage{}, &models.AvailabilityCycle{}, &models.AvailabilityRun{}, &models.AvailabilityResult{}, &models.AppSetting{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	owner := models.User{Username: "u"}
	db.Create(&owner)

	cycle := &models.AvailabilityCycle{
		TriggerType: models.AvailabilityRunTriggerAdmin,
		Status:      models.AvailabilityCycleStatusRunning,
		StartedAt:   time.Now().Add(-(availabilityStaleRunTimeout + time.Minute)),
	}
	db.Create(cycle)

	// Seed a child run that was in "running" state and started more than the stale timeout ago.
	staleStart := time.Now().Add(-(availabilityStaleRunTimeout + time.Minute))
	staleRun := &models.AvailabilityRun{
		UserID:      owner.ID,
		CycleID:     &cycle.ID,
		TriggerType: models.AvailabilityRunTriggerAdmin,
		Status:      models.AvailabilityRunStatusRunning,
		StartedAt:   staleStart,
	}
	db.Create(staleRun)

	availCycleRepo := repository.NewAvailabilityCycleRepository(db)
	availRepo := repository.NewAvailabilityRepository(db)
	availRepo.WithCycleRepo(availCycleRepo)

	availRepo.RecoverStaleChildRuns(availabilityStaleRunTimeout)

	var recoveredRun models.AvailabilityRun
	db.First(&recoveredRun, staleRun.ID)
	if recoveredRun.Status != models.AvailabilityRunStatusFailed {
		t.Fatalf("expected stale child run to be failed (not requeued), got %q", recoveredRun.Status)
	}

	var recoveredCycle models.AvailabilityCycle
	db.First(&recoveredCycle, cycle.ID)
	if recoveredCycle.Status != models.AvailabilityCycleStatusFailed {
		t.Fatalf("expected parent cycle aggregated to failed once its only child failed, got %q", recoveredCycle.Status)
	}
}

// TestAvailabilityScheduler_ProcessCycle_IdempotentWhenAlreadyClaimed verifies that
// ProcessCycle silently no-ops when the cycle has already been claimed by another worker.
func TestAvailabilityScheduler_ProcessCycle_IdempotentWhenAlreadyClaimed(t *testing.T) {
	scheduler, availCycleRepo, db := setupAvailSchedulerWithService(t, "https://example.test/coin")

	triggerID := uint(1)
	cycle, err := scheduler.RunNowWithTrigger(&triggerID)
	if err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	// Manually set to running to simulate another worker claiming it
	db.Model(&models.AvailabilityCycle{}).Where("id = ?", cycle.ID).Update("status", models.AvailabilityCycleStatusRunning)

	// ProcessCycle should return nil (no-op)
	if err := scheduler.ProcessCycle(cycle.ID); err != nil {
		t.Fatalf("ProcessCycle on already-running cycle should return nil, got: %v", err)
	}
	_ = availCycleRepo
}

// --- T039: RecoverStaleCycles wiring through StartWorkers ---

// TestAvailabilityScheduler_StartWorkers_ReQueuesQueuedCycles verifies that StartWorkers
// discovers any cycle left in "queued" status by a previous process (enqueued but never
// claimed before a restart) and re-enqueues it into the in-memory worker queue so it will
// still be processed.
func TestAvailabilityScheduler_StartWorkers_ReQueuesQueuedCycles(t *testing.T) {
	listing := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<html><body><p>Add to Cart</p></body></html>`))
	}))
	defer listing.Close()

	scheduler, availCycleRepo, db := setupAvailSchedulerWithService(t, listing.URL)

	// Seed a cycle stuck in "queued" as if the process crashed before a worker claimed it.
	queuedCycle := &models.AvailabilityCycle{
		TriggerType: models.AvailabilityRunTriggerAdmin,
		Status:      models.AvailabilityCycleStatusQueued,
		StartedAt:   time.Now(),
	}
	if err := db.Create(queuedCycle).Error; err != nil {
		t.Fatalf("seed queued cycle: %v", err)
	}

	scheduler.StartWorkers(1)
	defer scheduler.Stop()

	// Poll briefly for the worker to claim and complete the re-queued cycle.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		var reloaded models.AvailabilityCycle
		db.First(&reloaded, queuedCycle.ID)
		if reloaded.Status == models.AvailabilityCycleStatusCompleted {
			_ = availCycleRepo
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("expected StartWorkers to re-queue and eventually complete the previously-queued cycle")
}

// TestAvailabilityScheduler_RecoverStaleCycles_FinalizesTerminalChildrenOnBoot verifies the
// boot-time recovery path: a "running" cycle whose children are all terminal is finalized,
// while a "running" cycle with an active child is left alone (T039).
func TestAvailabilityScheduler_RecoverStaleCycles_FinalizesTerminalChildrenOnBoot(t *testing.T) {
	scheduler, availCycleRepo, db := setupAvailSchedulerWithService(t, "https://example.test/coin")
	_ = scheduler

	owner := models.User{Username: "boot-owner", Email: "boot-owner@test.com"}
	db.Create(&owner)

	staleStart := time.Now().Add(-(availabilityStaleRunTimeout + time.Minute))

	finishedCycle := &models.AvailabilityCycle{TriggerType: models.AvailabilityRunTriggerAdmin, Status: models.AvailabilityCycleStatusRunning, StartedAt: staleStart}
	db.Create(finishedCycle)
	db.Create(&models.AvailabilityRun{UserID: owner.ID, CycleID: &finishedCycle.ID, TriggerType: models.AvailabilityRunTriggerAdmin, Status: models.AvailabilityRunStatusCompleted, StartedAt: staleStart})

	activeCycle := &models.AvailabilityCycle{TriggerType: models.AvailabilityRunTriggerScheduled, Status: models.AvailabilityCycleStatusRunning, StartedAt: staleStart}
	db.Create(activeCycle)
	db.Create(&models.AvailabilityRun{UserID: owner.ID, CycleID: &activeCycle.ID, TriggerType: models.AvailabilityRunTriggerScheduled, Status: models.AvailabilityRunStatusRunning, StartedAt: staleStart})

	if _, err := availCycleRepo.RecoverStaleCycles(availabilityStaleRunTimeout); err != nil {
		t.Fatalf("RecoverStaleCycles: %v", err)
	}

	var reloadedFinished, reloadedActive models.AvailabilityCycle
	db.First(&reloadedFinished, finishedCycle.ID)
	db.First(&reloadedActive, activeCycle.ID)

	if reloadedFinished.Status != models.AvailabilityCycleStatusCompleted {
		t.Fatalf("expected cycle with all-terminal children to be finalized completed, got %q", reloadedFinished.Status)
	}
	if reloadedActive.Status != models.AvailabilityCycleStatusRunning {
		t.Fatalf("expected cycle with an active child to remain running, got %q", reloadedActive.Status)
	}
}
