package repository

import (
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupCoinOfDayRunTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	if err := db.AutoMigrate(&models.CoinOfDayRun{}); err != nil {
		t.Fatalf("failed to migrate test db: %v", err)
	}
	return db
}

func TestCoinOfDayRunRepositoryCreateRunIfNoActiveReusesActiveRun(t *testing.T) {
	db := setupCoinOfDayRunTestDB(t)
	repo := NewCoinOfDayRunRepository(db)
	existing := &models.CoinOfDayRun{
		TriggerType: models.CoinOfDayRunTriggerManual,
		Status:      models.CoinOfDayRunStatusRunning,
		StartedAt:   time.Now(),
	}
	if err := db.Create(existing).Error; err != nil {
		t.Fatalf("seed run: %v", err)
	}
	candidate := &models.CoinOfDayRun{
		TriggerType: models.CoinOfDayRunTriggerManual,
		Status:      models.CoinOfDayRunStatusQueued,
		StartedAt:   time.Now(),
	}
	run, acquired, err := repo.CreateRunIfNoActive(candidate, time.Now().Add(-30*time.Minute))
	if err != nil {
		t.Fatalf("CreateRunIfNoActive: %v", err)
	}
	if acquired {
		t.Fatalf("expected existing run reuse, got acquired=true")
	}
	if run.ID != existing.ID {
		t.Fatalf("expected reused run id=%d, got %d", existing.ID, run.ID)
	}
}

func TestCoinOfDayRunRepositoryClaimQueuedRun(t *testing.T) {
	db := setupCoinOfDayRunTestDB(t)
	repo := NewCoinOfDayRunRepository(db)
	run := &models.CoinOfDayRun{
		TriggerType: models.CoinOfDayRunTriggerManual,
		Status:      models.CoinOfDayRunStatusQueued,
		StartedAt:   time.Now().Add(-5 * time.Minute),
	}
	if err := db.Create(run).Error; err != nil {
		t.Fatalf("seed run: %v", err)
	}
	claimedRun, claimed, err := repo.ClaimQueuedRun(run.ID)
	if err != nil {
		t.Fatalf("ClaimQueuedRun: %v", err)
	}
	if !claimed {
		t.Fatalf("expected run to be claimed")
	}
	if claimedRun.Status != models.CoinOfDayRunStatusRunning {
		t.Fatalf("expected status=running, got %s", claimedRun.Status)
	}
}

func TestCoinOfDayRunRepositoryRecoverStaleRuns(t *testing.T) {
	db := setupCoinOfDayRunTestDB(t)
	repo := NewCoinOfDayRunRepository(db)
	stale := &models.CoinOfDayRun{
		TriggerType: models.CoinOfDayRunTriggerScheduled,
		Status:      models.CoinOfDayRunStatusRunning,
		StartedAt:   time.Now().Add(-2 * time.Hour),
	}
	if err := db.Create(stale).Error; err != nil {
		t.Fatalf("seed stale run: %v", err)
	}
	ids, err := repo.RecoverStaleRuns(30 * time.Minute)
	if err != nil {
		t.Fatalf("RecoverStaleRuns: %v", err)
	}
	if len(ids) != 1 || ids[0] != stale.ID {
		t.Fatalf("expected recovered stale run id=%d, got %v", stale.ID, ids)
	}
	var reloaded models.CoinOfDayRun
	if err := db.First(&reloaded, stale.ID).Error; err != nil {
		t.Fatalf("reload stale run: %v", err)
	}
	if reloaded.Status != models.CoinOfDayRunStatusQueued {
		t.Fatalf("expected stale run queued, got %s", reloaded.Status)
	}
}
