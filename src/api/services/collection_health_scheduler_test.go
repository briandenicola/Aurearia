package services

import (
	"testing"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupCollectionHealthSchedulerDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	if err := db.AutoMigrate(
		&models.AppSetting{},
		&models.User{},
		&models.Coin{},
		&models.CoinImage{},
		&models.CollectionHealthSnapshot{},
		&models.CollectionHealthSnapshotRun{},
	); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	return db
}

func newTestCollectionHealthScheduler(t *testing.T, db *gorm.DB) (*CollectionHealthScheduler, *repository.CollectionHealthSnapshotRunRepository) {
	t.Helper()
	healthRepo := repository.NewHealthRepository(db)
	healthSvc := NewHealthService(healthRepo, NewLogger(100))
	runRepo := repository.NewCollectionHealthSnapshotRunRepository(db)
	settingsSvc := NewSettingsService(repository.NewSettingsRepository(db))
	return NewCollectionHealthScheduler(healthSvc, runRepo, settingsSvc, NewLogger(100)), runRepo
}

// TestCollectionHealthScheduler_RunNowLogsCompletedRun verifies that a manual run
// creates a run-history row with accurate user counters, matching the run-logging
// pattern used by every other scheduler in the app (issue #484 admin alignment).
func TestCollectionHealthScheduler_RunNowLogsCompletedRun(t *testing.T) {
	db := setupCollectionHealthSchedulerDB(t)
	s, runRepo := newTestCollectionHealthScheduler(t, db)

	for i := 0; i < 2; i++ {
		user := models.User{Username: "user", Email: "user@test.com"}
		db.Create(&user)
		db.Create(&models.Coin{UserID: user.ID, Name: "Coin", IsWishlist: false, IsSold: false})
	}

	if err := s.RunNow(); err != nil {
		t.Fatalf("RunNow: %v", err)
	}

	runs, total, err := runRepo.ListRuns(1, 20)
	if err != nil {
		t.Fatalf("ListRuns: %v", err)
	}
	if total != 1 || len(runs) != 1 {
		t.Fatalf("expected 1 run, got total=%d len=%d", total, len(runs))
	}

	run := runs[0]
	if run.TriggerType != "manual" {
		t.Errorf("triggerType = %q, want manual", run.TriggerType)
	}
	if run.Status != "success" {
		t.Errorf("status = %q, want success", run.Status)
	}
	if run.UsersEligible != 2 || run.UsersSnapshotted != 2 || run.UsersFailed != 0 {
		t.Errorf("counters = eligible=%d snapshotted=%d failed=%d, want 2/2/0", run.UsersEligible, run.UsersSnapshotted, run.UsersFailed)
	}
	if run.CompletedAt == nil {
		t.Error("expected CompletedAt to be set")
	}

	var snapshotCount int64
	db.Model(&models.CollectionHealthSnapshot{}).Count(&snapshotCount)
	if snapshotCount != 2 {
		t.Errorf("expected 2 snapshots persisted, got %d", snapshotCount)
	}
}

// TestCollectionHealthScheduler_RunNowNoEligibleUsers verifies a run still gets
// logged (as a zero-user success) when there are no eligible users, rather than
// silently returning without a record.
func TestCollectionHealthScheduler_RunNowNoEligibleUsers(t *testing.T) {
	db := setupCollectionHealthSchedulerDB(t)
	s, runRepo := newTestCollectionHealthScheduler(t, db)

	if err := s.RunNow(); err != nil {
		t.Fatalf("RunNow: %v", err)
	}

	runs, total, err := runRepo.ListRuns(1, 20)
	if err != nil {
		t.Fatalf("ListRuns: %v", err)
	}
	if total != 1 || len(runs) != 1 {
		t.Fatalf("expected 1 run, got total=%d len=%d", total, len(runs))
	}
	if runs[0].UsersEligible != 0 || runs[0].Status != "success" {
		t.Errorf("run = %+v, want UsersEligible=0 Status=success", runs[0])
	}
}

// TestCollectionHealthScheduler_ListRuns verifies pagination passthrough.
func TestCollectionHealthScheduler_ListRuns(t *testing.T) {
	db := setupCollectionHealthSchedulerDB(t)
	s, _ := newTestCollectionHealthScheduler(t, db)

	if err := s.RunNow(); err != nil {
		t.Fatalf("RunNow 1: %v", err)
	}
	if err := s.RunNow(); err != nil {
		t.Fatalf("RunNow 2: %v", err)
	}

	runs, total, err := s.ListRuns(1, 20)
	if err != nil {
		t.Fatalf("ListRuns: %v", err)
	}
	if total != 2 || len(runs) != 2 {
		t.Fatalf("expected 2 runs, got total=%d len=%d", total, len(runs))
	}
}
