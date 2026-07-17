package repository

import (
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupCollectionHealthSnapshotRunRepository(t *testing.T) (*CollectionHealthSnapshotRunRepository, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&models.CollectionHealthSnapshotRun{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return NewCollectionHealthSnapshotRunRepository(db), db
}

func TestCollectionHealthSnapshotRunRepository_CreateCompleteAndList(t *testing.T) {
	repo, _ := setupCollectionHealthSnapshotRunRepository(t)

	run := &models.CollectionHealthSnapshotRun{
		TriggerType: "scheduled",
		Status:      "running",
		StartedAt:   time.Now(),
	}
	if err := repo.CreateRun(run); err != nil {
		t.Fatalf("CreateRun: %v", err)
	}
	if run.ID == 0 {
		t.Fatal("expected run ID to be assigned")
	}

	completedAt := time.Now()
	run.Status = "success"
	run.UsersEligible = 5
	run.UsersSnapshotted = 4
	run.UsersFailed = 1
	run.CompletedAt = &completedAt
	run.DurationMs = 42
	if err := repo.CompleteRun(run); err != nil {
		t.Fatalf("CompleteRun: %v", err)
	}

	runs, total, err := repo.ListRuns(1, 20)
	if err != nil {
		t.Fatalf("ListRuns: %v", err)
	}
	if total != 1 || len(runs) != 1 {
		t.Fatalf("expected 1 run, got total=%d len=%d", total, len(runs))
	}
	got := runs[0]
	if got.Status != "success" || got.UsersEligible != 5 || got.UsersSnapshotted != 4 || got.UsersFailed != 1 {
		t.Fatalf("unexpected run after complete: %+v", got)
	}
	if got.CompletedAt == nil {
		t.Fatal("expected CompletedAt to be persisted")
	}
}

func TestCollectionHealthSnapshotRunRepository_ListRunsNewestFirst(t *testing.T) {
	repo, db := setupCollectionHealthSnapshotRunRepository(t)

	older := &models.CollectionHealthSnapshotRun{TriggerType: "scheduled", Status: "success", StartedAt: time.Now().Add(-2 * time.Hour)}
	newer := &models.CollectionHealthSnapshotRun{TriggerType: "manual", Status: "success", StartedAt: time.Now().Add(-1 * time.Minute)}
	if err := db.Create(older).Error; err != nil {
		t.Fatalf("seed older: %v", err)
	}
	if err := db.Create(newer).Error; err != nil {
		t.Fatalf("seed newer: %v", err)
	}

	runs, total, err := repo.ListRuns(1, 20)
	if err != nil {
		t.Fatalf("ListRuns: %v", err)
	}
	if total != 2 || len(runs) != 2 {
		t.Fatalf("expected 2 runs, got total=%d len=%d", total, len(runs))
	}
	if runs[0].ID != newer.ID {
		t.Fatalf("expected newest run first, got run %d", runs[0].ID)
	}
}

func TestCollectionHealthSnapshotRunRepository_PruneOldRuns(t *testing.T) {
	repo, db := setupCollectionHealthSnapshotRunRepository(t)

	for i := 0; i < 5; i++ {
		run := &models.CollectionHealthSnapshotRun{
			TriggerType: "scheduled",
			Status:      "success",
			StartedAt:   time.Now().Add(time.Duration(i) * time.Minute),
		}
		if err := db.Create(run).Error; err != nil {
			t.Fatalf("seed run %d: %v", i, err)
		}
	}

	repo.PruneOldRuns(2)

	var count int64
	db.Model(&models.CollectionHealthSnapshotRun{}).Count(&count)
	if count != 2 {
		t.Fatalf("expected 2 runs to remain after pruning, got %d", count)
	}
}
