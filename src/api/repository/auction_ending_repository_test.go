package repository

import (
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupAuctionEndingRepoTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	err = db.AutoMigrate(&models.AuctionEndingRun{})
	if err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	return db
}

// TestAuctionEndingRepository_CreateRun verifies that creating a run assigns an ID and sets timestamps.
func TestAuctionEndingRepository_CreateRun(t *testing.T) {
	db := setupAuctionEndingRepoTestDB(t)
	repo := NewAuctionEndingRepository(db)

	now := time.Now()
	run := &models.AuctionEndingRun{
		TriggerType: "manual",
		Status:      "running",
		StartedAt:   now,
	}

	if err := repo.CreateRun(run); err != nil {
		t.Fatalf("CreateRun failed: %v", err)
	}

	if run.ID == 0 {
		t.Error("expected run ID to be assigned")
	}
	if run.StartedAt.IsZero() {
		t.Error("expected StartedAt to be set")
	}
}

// TestAuctionEndingRepository_CompleteRun verifies that finalizing a run updates status, timestamps, and counts.
func TestAuctionEndingRepository_CompleteRun(t *testing.T) {
	db := setupAuctionEndingRepoTestDB(t)
	repo := NewAuctionEndingRepository(db)

	run := &models.AuctionEndingRun{
		TriggerType: "scheduled",
		Status:      "running",
		StartedAt:   time.Now().Add(-5 * time.Minute),
	}
	repo.CreateRun(run)

	// Finalize the run
	now := time.Now()
	run.Status = "success"
	run.CompletedAt = &now
	run.LotsChecked = 10
	run.AlertsSent = 3
	run.DurationMs = 5000

	if err := repo.CompleteRun(run); err != nil {
		t.Fatalf("CompleteRun failed: %v", err)
	}

	// Verify updates persisted
	var fetched models.AuctionEndingRun
	db.First(&fetched, run.ID)
	if fetched.Status != "success" {
		t.Errorf("expected status 'success', got %q", fetched.Status)
	}
	if fetched.CompletedAt == nil || fetched.CompletedAt.IsZero() {
		t.Error("expected CompletedAt to be set")
	}
	if fetched.LotsChecked != 10 {
		t.Errorf("expected LotsChecked=10, got %d", fetched.LotsChecked)
	}
	if fetched.AlertsSent != 3 {
		t.Errorf("expected AlertsSent=3, got %d", fetched.AlertsSent)
	}
}

// TestAuctionEndingRepository_CompleteRun_WithError verifies that error runs persist error messages.
func TestAuctionEndingRepository_CompleteRun_WithError(t *testing.T) {
	db := setupAuctionEndingRepoTestDB(t)
	repo := NewAuctionEndingRepository(db)

	run := &models.AuctionEndingRun{
		TriggerType: "manual",
		Status:      "running",
		StartedAt:   time.Now(),
	}
	repo.CreateRun(run)

	now := time.Now()
	run.Status = "error"
	run.CompletedAt = &now
	run.ErrorMessage = "database connection lost"
	run.DurationMs = 1000

	if err := repo.CompleteRun(run); err != nil {
		t.Fatalf("CompleteRun failed: %v", err)
	}

	var fetched models.AuctionEndingRun
	db.First(&fetched, run.ID)
	if fetched.Status != "error" {
		t.Errorf("expected status 'error', got %q", fetched.Status)
	}
	if fetched.ErrorMessage != "database connection lost" {
		t.Errorf("expected error message to persist, got %q", fetched.ErrorMessage)
	}
}

// TestAuctionEndingRepository_ListRuns verifies paginated run history returns newest first.
func TestAuctionEndingRepository_ListRuns(t *testing.T) {
	db := setupAuctionEndingRepoTestDB(t)
	repo := NewAuctionEndingRepository(db)

	// Create 3 runs at different times
	for i := 0; i < 3; i++ {
		run := &models.AuctionEndingRun{
			TriggerType: "scheduled",
			Status:      "success",
			StartedAt:   time.Now().Add(time.Duration(-i) * time.Hour),
		}
		repo.CreateRun(run)
	}

	runs, total, err := repo.ListRuns(1, 10)
	if err != nil {
		t.Fatalf("ListRuns failed: %v", err)
	}
	if total != 3 {
		t.Errorf("expected total=3, got %d", total)
	}
	if len(runs) != 3 {
		t.Fatalf("expected 3 runs, got %d", len(runs))
	}

	// Verify newest first
	if !runs[0].StartedAt.After(runs[1].StartedAt) {
		t.Error("expected runs to be sorted newest first")
	}
}

// TestAuctionEndingRepository_ListRuns_Pagination verifies pagination respects limit parameter.
func TestAuctionEndingRepository_ListRuns_Pagination(t *testing.T) {
	db := setupAuctionEndingRepoTestDB(t)
	repo := NewAuctionEndingRepository(db)

	// Create 5 runs
	for i := 0; i < 5; i++ {
		run := &models.AuctionEndingRun{
			TriggerType: "manual",
			Status:      "success",
			StartedAt:   time.Now().Add(time.Duration(-i) * time.Minute),
		}
		repo.CreateRun(run)
	}

	runs, total, err := repo.ListRuns(1, 3)
	if err != nil {
		t.Fatalf("ListRuns failed: %v", err)
	}
	if total != 5 {
		t.Errorf("expected total=5, got %d", total)
	}
	if len(runs) != 3 {
		t.Errorf("expected 3 runs (limit=3), got %d", len(runs))
	}
}

// TestAuctionEndingRepository_ListRuns_EmptySlice verifies empty result returns empty slice, not nil.
func TestAuctionEndingRepository_ListRuns_EmptySlice(t *testing.T) {
	db := setupAuctionEndingRepoTestDB(t)
	repo := NewAuctionEndingRepository(db)

	runs, total, err := repo.ListRuns(1, 20)
	if err != nil {
		t.Fatalf("ListRuns failed: %v", err)
	}
	if total != 0 {
		t.Errorf("expected total=0, got %d", total)
	}
	if runs == nil {
		t.Error("expected empty slice, got nil")
	}
	if len(runs) != 0 {
		t.Errorf("expected len=0, got %d", len(runs))
	}
}

// TestAuctionEndingRepository_ListRuns_LimitZero verifies limit=0 applies a sane default (20).
func TestAuctionEndingRepository_ListRuns_LimitZero(t *testing.T) {
	db := setupAuctionEndingRepoTestDB(t)
	repo := NewAuctionEndingRepository(db)

	// Create 25 runs
	for i := 0; i < 25; i++ {
		run := &models.AuctionEndingRun{
			TriggerType: "scheduled",
			Status:      "success",
			StartedAt:   time.Now().Add(time.Duration(-i) * time.Minute),
		}
		repo.CreateRun(run)
	}

	runs, total, err := repo.ListRuns(1, 0)
	if err != nil {
		t.Fatalf("ListRuns failed: %v", err)
	}
	if total != 25 {
		t.Errorf("expected total=25, got %d", total)
	}
	// Default should be 20
	if len(runs) != 20 {
		t.Errorf("expected default limit=20, got %d", len(runs))
	}
}

// TestAuctionEndingRepository_ListRuns_NegativeLimit verifies negative limit applies a sane default.
func TestAuctionEndingRepository_ListRuns_NegativeLimit(t *testing.T) {
	db := setupAuctionEndingRepoTestDB(t)
	repo := NewAuctionEndingRepository(db)

	// Create 3 runs
	for i := 0; i < 3; i++ {
		run := &models.AuctionEndingRun{
			TriggerType: "manual",
			Status:      "success",
			StartedAt:   time.Now().Add(time.Duration(-i) * time.Minute),
		}
		repo.CreateRun(run)
	}

	runs, total, err := repo.ListRuns(1, -5)
	if err != nil {
		t.Fatalf("ListRuns failed: %v", err)
	}
	if total != 3 {
		t.Errorf("expected total=3, got %d", total)
	}
	// Should apply default (20) but only return 3 since that's all we have
	if len(runs) != 3 {
		t.Errorf("expected 3 runs, got %d", len(runs))
	}
}

// TestAuctionEndingRepository_GetRunByID verifies fetching a single run by ID.
func TestAuctionEndingRepository_GetRunByID(t *testing.T) {
	db := setupAuctionEndingRepoTestDB(t)
	repo := NewAuctionEndingRepository(db)

	run := &models.AuctionEndingRun{
		TriggerType: "manual",
		Status:      "success",
		StartedAt:   time.Now(),
		LotsChecked: 5,
		AlertsSent:  2,
	}
	repo.CreateRun(run)

	fetched, err := repo.GetRunByID(run.ID)
	if err != nil {
		t.Fatalf("GetRunByID failed: %v", err)
	}
	if fetched.ID != run.ID {
		t.Errorf("expected ID=%d, got %d", run.ID, fetched.ID)
	}
	if fetched.LotsChecked != 5 {
		t.Errorf("expected LotsChecked=5, got %d", fetched.LotsChecked)
	}
	if fetched.AlertsSent != 2 {
		t.Errorf("expected AlertsSent=2, got %d", fetched.AlertsSent)
	}
}

// TestAuctionEndingRepository_GetRunByID_NotFound verifies error when run doesn't exist.
func TestAuctionEndingRepository_GetRunByID_NotFound(t *testing.T) {
	db := setupAuctionEndingRepoTestDB(t)
	repo := NewAuctionEndingRepository(db)

	_, err := repo.GetRunByID(9999)
	if err == nil {
		t.Error("expected error when fetching non-existent run")
	}
}

func TestAuctionEndingRepository_GetLastScheduledRun(t *testing.T) {
	db := setupAuctionEndingRepoTestDB(t)
	repo := NewAuctionEndingRepository(db)

	manualCompleted := time.Now().Add(-3 * time.Hour)
	repo.CreateRun(&models.AuctionEndingRun{
		TriggerType: "manual",
		Status:      "success",
		StartedAt:   manualCompleted,
		CompletedAt: &manualCompleted,
	})

	scheduledOldCompleted := time.Now().Add(-2 * time.Hour)
	repo.CreateRun(&models.AuctionEndingRun{
		TriggerType: "scheduled",
		Status:      "success",
		StartedAt:   scheduledOldCompleted,
		CompletedAt: &scheduledOldCompleted,
	})

	scheduledRunning := time.Now().Add(-1 * time.Hour)
	repo.CreateRun(&models.AuctionEndingRun{
		TriggerType: "scheduled",
		Status:      "running",
		StartedAt:   scheduledRunning,
	})

	scheduledLatestCompleted := time.Now().Add(-30 * time.Minute)
	repo.CreateRun(&models.AuctionEndingRun{
		TriggerType: "scheduled",
		Status:      "success",
		StartedAt:   scheduledLatestCompleted,
		CompletedAt: &scheduledLatestCompleted,
	})

	lastRun := repo.GetLastScheduledRun()
	if lastRun == nil {
		t.Fatal("expected last scheduled run, got nil")
	}
	if lastRun.TriggerType != "scheduled" {
		t.Fatalf("expected scheduled run, got %q", lastRun.TriggerType)
	}
	if lastRun.CompletedAt == nil {
		t.Fatal("expected completed_at to be set")
	}
	if diff := lastRun.CompletedAt.Sub(scheduledLatestCompleted); diff < -time.Second || diff > time.Second {
		t.Fatalf("expected latest completed scheduled run near %v, got %v", scheduledLatestCompleted, *lastRun.CompletedAt)
	}
}

// TestAuctionEndingRepository_MarkRunning_Success verifies CAS transition queued → running.
func TestAuctionEndingRepository_MarkRunning_Success(t *testing.T) {
	db := setupAuctionEndingRepoTestDB(t)
	repo := NewAuctionEndingRepository(db)

	run := &models.AuctionEndingRun{
		TriggerType: "manual",
		Status:      "queued",
		StartedAt:   time.Now(),
	}
	repo.CreateRun(run)

	claimed, ok, err := repo.MarkRunning(run.ID)
	if err != nil {
		t.Fatalf("MarkRunning error: %v", err)
	}
	if !ok {
		t.Fatal("expected MarkRunning to succeed")
	}
	if claimed.Status != "running" {
		t.Errorf("expected status running, got %q", claimed.Status)
	}
}

// TestAuctionEndingRepository_MarkRunning_AlreadyClaimed verifies CAS returns false when not queued.
func TestAuctionEndingRepository_MarkRunning_AlreadyClaimed(t *testing.T) {
	db := setupAuctionEndingRepoTestDB(t)
	repo := NewAuctionEndingRepository(db)

	run := &models.AuctionEndingRun{
		TriggerType: "manual",
		Status:      "running", // already claimed
		StartedAt:   time.Now(),
	}
	repo.CreateRun(run)

	_, ok, err := repo.MarkRunning(run.ID)
	if err != nil {
		t.Fatalf("MarkRunning error: %v", err)
	}
	if ok {
		t.Error("expected MarkRunning to return false for already-running run")
	}
}

// TestAuctionEndingRepository_FindActiveRun_ReturnsQueued verifies FindActiveRun returns queued run.
func TestAuctionEndingRepository_FindActiveRun_ReturnsQueued(t *testing.T) {
	db := setupAuctionEndingRepoTestDB(t)
	repo := NewAuctionEndingRepository(db)

	run := &models.AuctionEndingRun{
		TriggerType: "manual",
		Status:      "queued",
		StartedAt:   time.Now(),
	}
	repo.CreateRun(run)

	active := repo.FindActiveRun()
	if active == nil {
		t.Fatal("expected FindActiveRun to return queued run")
	}
	if active.ID != run.ID {
		t.Errorf("expected run ID=%d, got %d", run.ID, active.ID)
	}
}

// TestAuctionEndingRepository_FindActiveRun_ReturnsRunning verifies FindActiveRun returns running run.
func TestAuctionEndingRepository_FindActiveRun_ReturnsRunning(t *testing.T) {
	db := setupAuctionEndingRepoTestDB(t)
	repo := NewAuctionEndingRepository(db)

	run := &models.AuctionEndingRun{
		TriggerType: "manual",
		Status:      "running",
		StartedAt:   time.Now(),
	}
	repo.CreateRun(run)

	active := repo.FindActiveRun()
	if active == nil {
		t.Fatal("expected FindActiveRun to return running run")
	}
	if active.ID != run.ID {
		t.Errorf("expected run ID=%d, got %d", run.ID, active.ID)
	}
}

// TestAuctionEndingRepository_FindActiveRun_NilWhenTerminal verifies FindActiveRun returns nil for terminal runs.
func TestAuctionEndingRepository_FindActiveRun_NilWhenTerminal(t *testing.T) {
	db := setupAuctionEndingRepoTestDB(t)
	repo := NewAuctionEndingRepository(db)

	now := time.Now()
	run := &models.AuctionEndingRun{
		TriggerType: "manual",
		Status:      "success",
		StartedAt:   now,
		CompletedAt: &now,
	}
	repo.CreateRun(run)

	active := repo.FindActiveRun()
	if active != nil {
		t.Errorf("expected nil for terminal run, got ID=%d status=%s", active.ID, active.Status)
	}
}

// TestAuctionEndingRepository_RecoverStaleRuns verifies stale queued/running runs are marked error.
func TestAuctionEndingRepository_RecoverStaleRuns(t *testing.T) {
	db := setupAuctionEndingRepoTestDB(t)
	repo := NewAuctionEndingRepository(db)

	timeout := 30 * time.Minute
	staleTime := time.Now().Add(-2 * time.Hour)

	// Stale queued run
	r1 := &models.AuctionEndingRun{TriggerType: "manual", Status: "queued", StartedAt: staleTime}
	repo.CreateRun(r1)

	// Stale running run
	r2 := &models.AuctionEndingRun{TriggerType: "manual", Status: "running", StartedAt: staleTime}
	repo.CreateRun(r2)

	// Fresh queued run (should be left alone)
	r3 := &models.AuctionEndingRun{TriggerType: "manual", Status: "queued", StartedAt: time.Now()}
	repo.CreateRun(r3)

	// Completed run (should be left alone)
	now := time.Now()
	r4 := &models.AuctionEndingRun{TriggerType: "scheduled", Status: "success", StartedAt: staleTime, CompletedAt: &now}
	repo.CreateRun(r4)

	recovered := repo.RecoverStaleRuns(timeout)
	if recovered != 2 {
		t.Errorf("expected 2 stale runs recovered, got %d", recovered)
	}

	var runs []models.AuctionEndingRun
	db.Order("id ASC").Find(&runs)

	for _, r := range runs {
		switch r.ID {
		case r1.ID, r2.ID:
			if r.Status != "error" {
				t.Errorf("stale run #%d expected error, got %q", r.ID, r.Status)
			}
			if r.CompletedAt == nil {
				t.Errorf("stale run #%d expected CompletedAt set", r.ID)
			}
		case r3.ID:
			if r.Status != "queued" {
				t.Errorf("fresh run #%d expected queued, got %q", r.ID, r.Status)
			}
		case r4.ID:
			if r.Status != "success" {
				t.Errorf("completed run #%d expected success, got %q", r.ID, r.Status)
			}
		}
	}
}
