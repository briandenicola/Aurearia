package services

import (
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupAuctionEndingSchedulerDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	if err := db.AutoMigrate(&models.AppSetting{}, &models.AuctionEndingRun{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	return db
}

func newTestAuctionEndingScheduler(t *testing.T, db *gorm.DB) *AuctionEndingScheduler {
	t.Helper()
	settingsRepo := repository.NewSettingsRepository(db)
	settingsSvc := NewSettingsService(settingsRepo)
	auctionEndingRepo := repository.NewAuctionEndingRepository(db)
	return NewAuctionEndingScheduler(nil, auctionEndingRepo, nil, nil, settingsSvc, NewLogger(100))
}

func TestAuctionEndingTimeUntilNextRun_UsesLastCompletedRun(t *testing.T) {
	db := setupAuctionEndingSchedulerDB(t)
	s := newTestAuctionEndingScheduler(t, db)

	if err := s.settingsSvc.SetSetting(SettingAuctionEndingCheckInterval, "120"); err != nil {
		t.Fatalf("failed to set interval: %v", err)
	}

	completedAt := time.Now().Add(-60 * time.Minute)
	run := &models.AuctionEndingRun{
		TriggerType: "scheduled",
		Status:      "success",
		StartedAt:   completedAt.Add(-2 * time.Minute),
		CompletedAt: &completedAt,
	}
	if err := db.Create(run).Error; err != nil {
		t.Fatalf("failed to seed run: %v", err)
	}

	wait := s.timeUntilNextRun()
	if wait < 59*time.Minute || wait > 61*time.Minute {
		t.Fatalf("expected ~60m wait, got %v", wait)
	}
}

func TestAuctionEndingTimeUntilNextRun_Overdue(t *testing.T) {
	db := setupAuctionEndingSchedulerDB(t)
	s := newTestAuctionEndingScheduler(t, db)

	if err := s.settingsSvc.SetSetting(SettingAuctionEndingCheckInterval, "60"); err != nil {
		t.Fatalf("failed to set interval: %v", err)
	}

	completedAt := time.Now().Add(-2 * time.Hour)
	run := &models.AuctionEndingRun{
		TriggerType: "scheduled",
		Status:      "error",
		StartedAt:   completedAt.Add(-2 * time.Minute),
		CompletedAt: &completedAt,
	}
	if err := db.Create(run).Error; err != nil {
		t.Fatalf("failed to seed run: %v", err)
	}

	wait := s.timeUntilNextRun()
	if wait != 0 {
		t.Fatalf("expected immediate run (0), got %v", wait)
	}
}

func TestAuctionEndingTimeUntilNextRun_IgnoresManualRuns(t *testing.T) {
	db := setupAuctionEndingSchedulerDB(t)
	s := newTestAuctionEndingScheduler(t, db)

	future := time.Now().Add(2 * time.Hour)
	if err := s.settingsSvc.SetSetting(SettingAuctionEndingCheckStartTime, future.Format("15:04")); err != nil {
		t.Fatalf("failed to set start time: %v", err)
	}
	if err := s.settingsSvc.SetSetting(SettingAuctionEndingCheckInterval, "1440"); err != nil {
		t.Fatalf("failed to set interval: %v", err)
	}

	completedAt := time.Now().Add(-30 * time.Minute)
	run := &models.AuctionEndingRun{
		TriggerType: "manual",
		Status:      "success",
		StartedAt:   completedAt.Add(-2 * time.Minute),
		CompletedAt: &completedAt,
	}
	if err := db.Create(run).Error; err != nil {
		t.Fatalf("failed to seed run: %v", err)
	}

	wait := s.timeUntilNextRun()
	if wait < 1*time.Hour+55*time.Minute || wait > 2*time.Hour+5*time.Minute {
		t.Fatalf("expected ~2h wait when only manual runs exist, got %v", wait)
	}
}
