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

// setupAvailSchedulerWithService creates a full scheduler with a real AvailabilityService
// backed by an in-memory DB, suitable for async processing tests.
func setupAvailSchedulerWithService(t *testing.T, listingURL string) (*AvailabilityScheduler, *repository.AvailabilityRepository, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	if err := db.AutoMigrate(
		&models.User{},
		&models.Coin{},
		&models.CoinImage{},
		&models.AvailabilityRun{},
		&models.AvailabilityResult{},
		&models.AppSetting{},
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
	settingsRepo := repository.NewSettingsRepository(db)
	settingsSvc := NewSettingsService(settingsRepo)
	logger := NewLogger(100)
	availSvc := NewAvailabilityService(coinRepo, availRepo, nil, nil, nil, nil, settingsSvc, logger)
	scheduler := NewAvailabilityScheduler(availSvc, coinRepo, availRepo, settingsSvc, logger)
	return scheduler, availRepo, db
}

// TestAvailabilityScheduler_RunNowEnqueuesWithoutBlocking verifies that RunNowWithTrigger
// returns immediately with a queued run and does NOT process coins synchronously.
func TestAvailabilityScheduler_RunNowEnqueuesWithoutBlocking(t *testing.T) {
	agentCalled := false
	listing := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		agentCalled = true
		t.Fatal("listing server should not be called until worker processes the queued run")
	}))
	defer listing.Close()

	// Use a fresh DB (no listing server hit expected during enqueue)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("db: %v", err)
	}
	db.AutoMigrate(&models.User{}, &models.Coin{}, &models.CoinImage{}, &models.AvailabilityRun{}, &models.AvailabilityResult{}, &models.AppSetting{})
	owner := models.User{Username: "owner", Email: "owner@test.com"}
	db.Create(&owner)
	db.Create(&models.Coin{UserID: owner.ID, Name: "Coin", ReferenceURL: listing.URL, IsWishlist: true})

	coinRepo := repository.NewCoinRepository(db)
	availRepo := repository.NewAvailabilityRepository(db)
	settingsSvc := NewSettingsService(repository.NewSettingsRepository(db))
	logger := NewLogger(100)
	svc := NewAvailabilityService(coinRepo, availRepo, nil, nil, nil, nil, settingsSvc, logger)
	scheduler := NewAvailabilityScheduler(svc, coinRepo, availRepo, settingsSvc, logger)

	triggerID := owner.ID
	run, err := scheduler.RunNowWithTrigger(&triggerID)
	if err != nil {
		t.Fatalf("RunNowWithTrigger: %v", err)
	}
	if run.Status != models.AvailabilityRunStatusQueued {
		t.Fatalf("expected queued status, got %q", run.Status)
	}
	if agentCalled {
		t.Fatal("listing server was called synchronously during RunNowWithTrigger")
	}
}

// TestAvailabilityScheduler_DuplicateRunBlocked verifies that a second RunNowWithTrigger
// call is rejected while a queued or running manual run exists.
func TestAvailabilityScheduler_DuplicateRunBlocked(t *testing.T) {
	scheduler, _, _ := setupAvailSchedulerWithService(t, "https://example.test/coin")

	id := uint(1)
	if _, err := scheduler.RunNowWithTrigger(&id); err != nil {
		t.Fatalf("first enqueue: %v", err)
	}
	_, err := scheduler.RunNowWithTrigger(&id)
	if err == nil {
		t.Fatal("expected error for duplicate run, got nil")
	}
	if err != ErrAvailabilityRunInProgress {
		t.Fatalf("expected ErrAvailabilityRunInProgress, got %v", err)
	}
}

// TestAvailabilityScheduler_ProcessRun_Completed verifies that ProcessRun claims a queued
// run, checks all user coins, and marks the run completed.
func TestAvailabilityScheduler_ProcessRun_Completed(t *testing.T) {
	listing := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<html><body><p>Add to Cart</p></body></html>`))
	}))
	defer listing.Close()

	scheduler, availRepo, db := setupAvailSchedulerWithService(t, listing.URL)

	triggerID := uint(1)
	run, err := scheduler.RunNowWithTrigger(&triggerID)
	if err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	if err := scheduler.ProcessRun(run.ID); err != nil {
		t.Fatalf("ProcessRun: %v", err)
	}

	var completed models.AvailabilityRun
	if err := db.First(&completed, run.ID).Error; err != nil {
		t.Fatalf("load run: %v", err)
	}
	if completed.Status != models.AvailabilityRunStatusCompleted {
		t.Fatalf("expected completed, got %q", completed.Status)
	}
	if completed.CoinsChecked != 1 {
		t.Fatalf("expected 1 coin checked, got %d", completed.CoinsChecked)
	}
	if completed.Available != 1 {
		t.Fatalf("expected 1 available, got %d", completed.Available)
	}

	// Verify result record was created
	var results []models.AvailabilityResult
	db.Where("run_id = ?", run.ID).Find(&results)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	_ = availRepo
}

// TestAvailabilityScheduler_StaleRunRecovery verifies that StartWorkers re-queues
// any runs that were stuck in running state (e.g. from a crashed process).
func TestAvailabilityScheduler_StaleRunRecovery(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("db: %v", err)
	}
	db.AutoMigrate(&models.User{}, &models.Coin{}, &models.CoinImage{}, &models.AvailabilityRun{}, &models.AvailabilityResult{}, &models.AppSetting{})

	owner := models.User{Username: "u"}
	db.Create(&owner)

	// Seed a run that was in "running" state and started more than the stale timeout ago
	staleStart := time.Now().Add(-(availabilityStaleRunTimeout + time.Minute))
	staleRun := &models.AvailabilityRun{
		UserID:      owner.ID,
		TriggerType: "manual",
		Status:      models.AvailabilityRunStatusRunning,
		StartedAt:   staleStart,
	}
	db.Create(staleRun)

	availRepo := repository.NewAvailabilityRepository(db)
	ids, err := availRepo.RecoverStaleRuns(availabilityStaleRunTimeout)
	if err != nil {
		t.Fatalf("RecoverStaleRuns: %v", err)
	}
	if len(ids) != 1 || ids[0] != staleRun.ID {
		t.Fatalf("expected stale run %d to be recovered, got %v", staleRun.ID, ids)
	}

	// Verify the run was reset to queued in the DB
	var recovered models.AvailabilityRun
	db.First(&recovered, staleRun.ID)
	if recovered.Status != models.AvailabilityRunStatusQueued {
		t.Fatalf("expected queued after recovery, got %q", recovered.Status)
	}
}

// TestAvailabilityScheduler_ProcessRun_IdempotentWhenAlreadyClaimed verifies that
// ProcessRun silently no-ops when the run has already been claimed by another worker.
func TestAvailabilityScheduler_ProcessRun_IdempotentWhenAlreadyClaimed(t *testing.T) {
	scheduler, _, db := setupAvailSchedulerWithService(t, "https://example.test/coin")

	triggerID := uint(1)
	run, err := scheduler.RunNowWithTrigger(&triggerID)
	if err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	// Manually set to running to simulate another worker claiming it
	db.Model(run).Update("status", models.AvailabilityRunStatusRunning)

	// ProcessRun should return nil (no-op)
	if err := scheduler.ProcessRun(run.ID); err != nil {
		t.Fatalf("ProcessRun on already-running run should return nil, got: %v", err)
	}
}
