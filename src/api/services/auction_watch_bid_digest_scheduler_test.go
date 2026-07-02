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

func setupAuctionWatchBidDigestSchedulerDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	if err := db.AutoMigrate(&models.AppSetting{}, &models.User{}, &models.AuctionWatchBidDigestRun{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	return db
}

func newTestAuctionWatchBidDigestScheduler(t *testing.T, db *gorm.DB) *AuctionWatchBidDigestScheduler {
	t.Helper()
	settingsRepo := repository.NewSettingsRepository(db)
	settingsSvc := NewSettingsService(settingsRepo)
	runRepo := repository.NewAuctionWatchBidDigestRepository(db)
	return NewAuctionWatchBidDigestScheduler(nil, runRepo, nil, nil, nil, settingsSvc, NewLogger(100))
}

func TestAuctionWatchBidDigestTimeUntilNextRun_UsesLastCompletedRun(t *testing.T) {
	db := setupAuctionWatchBidDigestSchedulerDB(t)
	s := newTestAuctionWatchBidDigestScheduler(t, db)

	if err := s.settingsSvc.SetSetting(SettingAuctionWatchBidDigestInterval, "120"); err != nil {
		t.Fatalf("failed to set interval: %v", err)
	}

	completedAt := time.Now().Add(-60 * time.Minute)
	run := &models.AuctionWatchBidDigestRun{
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

func TestAuctionWatchBidDigestNotifyUserIncludesCurrentHighBids(t *testing.T) {
	db := setupAuctionWatchBidDigestSchedulerDB(t)
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

	scheduler := NewAuctionWatchBidDigestScheduler(
		nil,
		repository.NewAuctionWatchBidDigestRepository(db),
		repository.NewUserRepository(db),
		pushoverSvc,
		nil,
		NewSettingsService(repository.NewSettingsRepository(db)),
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

	if got := captured.Get("title"); got != "Auction Watch Bid Digest" {
		t.Fatalf("title = %q, want Auction Watch Bid Digest", got)
	}
	message := captured.Get("message")
	for _, want := range []string{
		"2 watched auction lot(s):",
		"The Coin Cabinet - Ancients Auction 35 (Lot 30): current high bid 125.50 GBP",
		"Classical Numismatic Group - Keystone 17 (Lot 95): current high bid 300.00 USD",
	} {
		if !strings.Contains(message, want) {
			t.Fatalf("message %q missing %q", message, want)
		}
	}
}

func TestFormatAuctionBidHandlesMissingBid(t *testing.T) {
	if got := formatAuctionBid(nil, "USD"); got != "current high bid unavailable" {
		t.Fatalf("formatAuctionBid(nil) = %q", got)
	}
}
