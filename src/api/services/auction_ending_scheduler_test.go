package services

import (
	"net/url"
	"strings"
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

func TestAuctionEndingNotifyUserSendsEndingSoonAlert(t *testing.T) {
	db := setupAuctionEndingSchedulerDB(t)
	if err := db.AutoMigrate(&models.User{}); err != nil {
		t.Fatalf("failed to migrate users: %v", err)
	}

	user := models.User{
		Username:        "bidder",
		Email:           "bidder@example.com",
		PasswordHash:    "hash",
		PushoverEnabled: true,
		PushoverUserKey: "user-key",
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	var captured url.Values
	pushoverSvc, cleanup := newTestPushoverService(t, &captured)
	defer cleanup()

	settingsSvc := NewSettingsService(repository.NewSettingsRepository(db))
	scheduler := NewAuctionEndingScheduler(
		nil,
		repository.NewAuctionEndingRepository(db),
		repository.NewUserRepository(db),
		pushoverSvc,
		settingsSvc,
		NewLogger(100),
	)

	bidOne := 125.5
	bidTwo := 300.0
	sent := scheduler.notifyUser(user.ID, []models.AuctionLot{
		{AuctionHouse: "The Coin Cabinet", SaleName: "Ancients Auction 35", LotNumber: 30, CurrentBid: &bidOne, Currency: "GBP"},
		{AuctionHouse: "Classical Numismatic Group", SaleName: "Keystone 17", LotNumber: 95, CurrentBid: &bidTwo, Currency: "USD"},
	})
	if !sent {
		t.Fatal("notifyUser returned false")
	}

	if got := captured.Get("title"); got != "Auctions Ending Soon" {
		t.Fatalf("title = %q, want Auctions Ending Soon", got)
	}
	message := captured.Get("message")
	for _, want := range []string{
		"2 auction(s) you are bidding on end within 24 hours:",
		"The Coin Cabinet - Ancients Auction 35 (Lot 30)",
		"Classical Numismatic Group - Keystone 17 (Lot 95)",
	} {
		if !strings.Contains(message, want) {
			t.Fatalf("message %q missing %q", message, want)
		}
	}
}
