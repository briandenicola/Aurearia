package repository

import (
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// setupAvailabilityRepoTestDB creates an in-memory SQLite DB with the tables needed to
// exercise AvailabilityRepository (Feature 353 child-run behavior).
func setupAvailabilityRepoTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	if err := db.AutoMigrate(
		&models.User{},
		&models.AvailabilityCycle{},
		&models.AvailabilityRun{},
		&models.AvailabilityResult{},
	); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	return db
}

func newChildRun(userID uint, triggerType, status string) *models.AvailabilityRun {
	return &models.AvailabilityRun{
		UserID:      userID,
		TriggerType: triggerType,
		Status:      status,
		StartedAt:   time.Now(),
	}
}

// --- T004: CreateChildRun rejects UserID == 0 (FR-002, SC-001) ---

func TestCreateChildRun_RejectsZeroUserID(t *testing.T) {
	db := setupAvailabilityRepoTestDB(t)
	repo := NewAvailabilityRepository(db)

	run := newChildRun(0, models.AvailabilityRunTriggerAdmin, models.AvailabilityRunStatusQueued)
	if err := repo.CreateChildRun(run); err == nil {
		t.Fatal("expected CreateChildRun to reject UserID == 0")
	}

	var count int64
	db.Model(&models.AvailabilityRun{}).Count(&count)
	if count != 0 {
		t.Fatalf("expected no row to be created for a rejected child run, got %d", count)
	}
}

func TestCreateChildRun_AcceptsPositiveUserID(t *testing.T) {
	db := setupAvailabilityRepoTestDB(t)
	repo := NewAvailabilityRepository(db)

	user := models.User{Username: "owner", Email: "owner@test.com"}
	db.Create(&user)

	run := newChildRun(user.ID, models.AvailabilityRunTriggerOwner, models.AvailabilityRunStatusRunning)
	if err := repo.CreateChildRun(run); err != nil {
		t.Fatalf("CreateChildRun: %v", err)
	}
	if run.ID == 0 {
		t.Fatal("expected child run to be persisted with a non-zero ID")
	}
}

// --- T004 / T031: CompleteChildRun retention (FR-018, US4 AC1) ---

func TestCompleteChildRun_PrunesToKeep20TerminalPerOwner(t *testing.T) {
	db := setupAvailabilityRepoTestDB(t)
	repo := NewAvailabilityRepository(db)

	user := models.User{Username: "retention-owner", Email: "retention-owner@test.com"}
	db.Create(&user)

	// Seed 20 already-terminal (completed) runs at increasing completed_at timestamps.
	base := time.Now().Add(-30 * time.Hour)
	for i := 0; i < 20; i++ {
		completedAt := base.Add(time.Duration(i) * time.Hour)
		run := &models.AvailabilityRun{
			UserID:      user.ID,
			TriggerType: models.AvailabilityRunTriggerOwner,
			Status:      models.AvailabilityRunStatusCompleted,
			StartedAt:   completedAt,
			CompletedAt: &completedAt,
		}
		if err := db.Create(run).Error; err != nil {
			t.Fatalf("seed terminal run %d: %v", i, err)
		}
		// Attach a result row to prove result cascade-deletion on prune.
		db.Create(&models.AvailabilityResult{RunID: run.ID, CoinID: 1, Status: "available", CheckedAt: completedAt})
	}

	var beforeCount int64
	db.Model(&models.AvailabilityRun{}).Where("user_id = ?", user.ID).Count(&beforeCount)
	if beforeCount != 20 {
		t.Fatalf("expected 20 seeded runs, got %d", beforeCount)
	}

	// Complete a 21st run — this should push the total to 21, then prune down to 20.
	newRun := newChildRun(user.ID, models.AvailabilityRunTriggerOwner, models.AvailabilityRunStatusRunning)
	if err := repo.CreateChildRun(newRun); err != nil {
		t.Fatalf("CreateChildRun: %v", err)
	}
	db.Create(&models.AvailabilityResult{RunID: newRun.ID, CoinID: 1, Status: "available", CheckedAt: time.Now()})
	if _, err := repo.CompleteChildRun(newRun); err != nil {
		t.Fatalf("CompleteChildRun: %v", err)
	}

	var afterCount int64
	db.Model(&models.AvailabilityRun{}).Where("user_id = ? AND status IN ?", user.ID,
		[]string{models.AvailabilityRunStatusCompleted, models.AvailabilityRunStatusFailed}).Count(&afterCount)
	if afterCount != 20 {
		t.Fatalf("expected exactly 20 terminal runs remaining for owner, got %d", afterCount)
	}

	// The newest run must survive.
	var stillExists models.AvailabilityRun
	if err := db.First(&stillExists, newRun.ID).Error; err != nil {
		t.Fatalf("expected the newly completed run to survive pruning: %v", err)
	}

	// Results for pruned runs must also be gone (cascade).
	var resultCount int64
	db.Model(&models.AvailabilityResult{}).Count(&resultCount)
	if resultCount != 20 {
		t.Fatalf("expected 20 surviving result rows (one per surviving run), got %d", resultCount)
	}
}

func TestCompleteChildRun_NeverPrunesQueuedOrRunning(t *testing.T) {
	db := setupAvailabilityRepoTestDB(t)
	repo := NewAvailabilityRepository(db)

	user := models.User{Username: "mixed-owner", Email: "mixed-owner@test.com"}
	db.Create(&user)

	// 20 terminal runs.
	base := time.Now().Add(-30 * time.Hour)
	for i := 0; i < 20; i++ {
		completedAt := base.Add(time.Duration(i) * time.Hour)
		db.Create(&models.AvailabilityRun{
			UserID:      user.ID,
			TriggerType: models.AvailabilityRunTriggerOwner,
			Status:      models.AvailabilityRunStatusCompleted,
			StartedAt:   completedAt,
			CompletedAt: &completedAt,
		})
	}
	// Plus 1 queued and 1 running — must never be pruned.
	queuedRun := &models.AvailabilityRun{UserID: user.ID, TriggerType: models.AvailabilityRunTriggerOwner, Status: models.AvailabilityRunStatusQueued, StartedAt: time.Now()}
	runningRun := &models.AvailabilityRun{UserID: user.ID, TriggerType: models.AvailabilityRunTriggerOwner, Status: models.AvailabilityRunStatusRunning, StartedAt: time.Now()}
	db.Create(queuedRun)
	db.Create(runningRun)

	// Complete one more terminal run to trigger pruning.
	newRun := newChildRun(user.ID, models.AvailabilityRunTriggerOwner, models.AvailabilityRunStatusRunning)
	repo.CreateChildRun(newRun)
	if _, err := repo.CompleteChildRun(newRun); err != nil {
		t.Fatalf("CompleteChildRun: %v", err)
	}

	var terminalCount int64
	db.Model(&models.AvailabilityRun{}).Where("user_id = ? AND status IN ?", user.ID,
		[]string{models.AvailabilityRunStatusCompleted, models.AvailabilityRunStatusFailed}).Count(&terminalCount)
	if terminalCount != 20 {
		t.Fatalf("expected 20 terminal runs, got %d", terminalCount)
	}

	var queuedStill, runningStill models.AvailabilityRun
	if err := db.First(&queuedStill, queuedRun.ID).Error; err != nil {
		t.Fatalf("expected queued run to survive pruning: %v", err)
	}
	if err := db.First(&runningStill, runningRun.ID).Error; err != nil {
		t.Fatalf("expected running run to survive pruning: %v", err)
	}
}

// --- T004: ListRunsForOwner scoping (FR-017, FR-022, US1 AC1) ---

func TestListRunsForOwner_ReturnsOnlyThatOwnersRuns(t *testing.T) {
	db := setupAvailabilityRepoTestDB(t)
	repo := NewAvailabilityRepository(db)

	ownerA := models.User{Username: "ownerA", Email: "ownerA@test.com"}
	ownerB := models.User{Username: "ownerB", Email: "ownerB@test.com"}
	db.Create(&ownerA)
	db.Create(&ownerB)

	db.Create(newChildRun(ownerA.ID, models.AvailabilityRunTriggerOwner, models.AvailabilityRunStatusCompleted))
	db.Create(newChildRun(ownerA.ID, models.AvailabilityRunTriggerScheduled, models.AvailabilityRunStatusCompleted))
	db.Create(newChildRun(ownerB.ID, models.AvailabilityRunTriggerAdmin, models.AvailabilityRunStatusCompleted))

	runs, total, err := repo.ListRunsForOwner(ownerA.ID, 1, 20)
	if err != nil {
		t.Fatalf("ListRunsForOwner: %v", err)
	}
	if total != 2 {
		t.Fatalf("expected total=2 for ownerA, got %d", total)
	}
	for _, r := range runs {
		if r.UserID != ownerA.ID {
			t.Fatalf("expected only ownerA's runs, found run for user %d", r.UserID)
		}
	}
}

// --- T004: GetOwnedRunWithResults refuses cross-owner reads (FR-017, FR-023) ---

func TestGetOwnedRunWithResults_RefusesCrossOwnerRead(t *testing.T) {
	db := setupAvailabilityRepoTestDB(t)
	repo := NewAvailabilityRepository(db)

	ownerA := models.User{Username: "ownerA2", Email: "ownerA2@test.com"}
	ownerB := models.User{Username: "ownerB2", Email: "ownerB2@test.com"}
	db.Create(&ownerA)
	db.Create(&ownerB)

	run := newChildRun(ownerA.ID, models.AvailabilityRunTriggerOwner, models.AvailabilityRunStatusCompleted)
	db.Create(run)
	db.Create(&models.AvailabilityResult{RunID: run.ID, CoinID: 1, Status: "available", CheckedAt: time.Now()})

	// Owner A can read their own run.
	got, err := repo.GetOwnedRunWithResults(ownerA.ID, run.ID)
	if err != nil {
		t.Fatalf("expected owner to read their own run: %v", err)
	}
	if got.ID != run.ID {
		t.Fatalf("expected run %d, got %d", run.ID, got.ID)
	}
	if len(got.Results) != 1 {
		t.Fatalf("expected 1 preloaded result, got %d", len(got.Results))
	}

	// Owner B must not be able to read ownerA's run.
	if _, err := repo.GetOwnedRunWithResults(ownerB.ID, run.ID); err == nil {
		t.Fatal("expected cross-owner read to be refused with an error")
	}
}
