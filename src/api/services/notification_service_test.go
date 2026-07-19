package services

import (
	"strings"
	"testing"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupNotificationServiceDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("failed to get sql db: %v", err)
	}
	// NotifyAuctionPriceAlert/NotifyAuctionBidReminder fire Pushover on a background goroutine;
	// pin the pool to one connection so it can't land on a second, unmigrated ":memory:" instance.
	sqlDB.SetMaxOpenConns(1)
	if err := db.AutoMigrate(&models.User{}, &models.Notification{}, &models.AppSetting{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	return db
}

func newTestNotificationService(db *gorm.DB) *NotificationService {
	settingsSvc := NewSettingsService(repository.NewSettingsRepository(db))
	logger := NewLogger(100)
	return NewNotificationService(
		repository.NewNotificationRepository(db),
		nil,
		repository.NewUserRepository(db),
		NewPushoverService(settingsSvc, logger),
		logger,
	)
}

func TestNotifyAuctionPriceAlertCreatesNotification(t *testing.T) {
	db := setupNotificationServiceDB(t)
	svc := newTestNotificationService(db)

	user := models.User{Username: "bidder", Email: "bidder@example.com", PasswordHash: "hash"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	bid := 150.0
	lot := models.AuctionLot{
		UserID:       user.ID,
		AuctionHouse: "CNG",
		SaleName:     "Keystone 17",
		LotNumber:    95,
		CurrentBid:   &bid,
		Currency:     "USD",
		SourceURL:    "https://cngcoins.com/lot/95",
	}

	svc.NotifyAuctionPriceAlert(user.ID, lot, 100)

	var notifications []models.Notification
	if err := db.Where("user_id = ?", user.ID).Find(&notifications).Error; err != nil {
		t.Fatalf("failed to query notifications: %v", err)
	}
	if len(notifications) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(notifications))
	}
	n := notifications[0]
	if n.Type != NotificationTypeAuctionPriceAlert {
		t.Errorf("type = %q, want %q", n.Type, NotificationTypeAuctionPriceAlert)
	}
	if n.ReferenceID != lot.ID {
		t.Errorf("referenceId = %d, want %d", n.ReferenceID, lot.ID)
	}
	if n.ReferenceURL != lot.SourceURL {
		t.Errorf("referenceUrl = %q, want %q", n.ReferenceURL, lot.SourceURL)
	}
}

func TestNotifyAuctionBidReminderCreatesNotification(t *testing.T) {
	db := setupNotificationServiceDB(t)
	svc := newTestNotificationService(db)

	user := models.User{Username: "bidder", Email: "bidder@example.com", PasswordHash: "hash"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	lot := models.AuctionLot{
		UserID:       user.ID,
		AuctionHouse: "CNG",
		SaleName:     "Keystone 17",
		LotNumber:    95,
		SourceURL:    "https://cngcoins.com/lot/95",
	}

	svc.NotifyAuctionBidReminder(user.ID, lot, 30)

	var notifications []models.Notification
	if err := db.Where("user_id = ? AND type = ?", user.ID, NotificationTypeAuctionBidReminder).Find(&notifications).Error; err != nil {
		t.Fatalf("failed to query notifications: %v", err)
	}
	if len(notifications) != 1 {
		t.Fatalf("expected 1 bid reminder notification, got %d", len(notifications))
	}
}

func TestNotifyAuctionEndingSoonCreatesSingleConsolidatedNotification(t *testing.T) {
	db := setupNotificationServiceDB(t)
	svc := newTestNotificationService(db)

	user := models.User{Username: "bidder", Email: "bidder@example.com", PasswordHash: "hash"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	lots := []models.AuctionLot{
		{AuctionHouse: "The Coin Cabinet", SaleName: "Ancients Auction 35", LotNumber: 30},
		{AuctionHouse: "Classical Numismatic Group", SaleName: "Keystone 17", LotNumber: 95},
	}

	svc.NotifyAuctionEndingSoon(user.ID, lots)

	var notifications []models.Notification
	if err := db.Where("user_id = ? AND type = ?", user.ID, NotificationTypeAuctionEndingSoon).Find(&notifications).Error; err != nil {
		t.Fatalf("failed to query notifications: %v", err)
	}
	if len(notifications) != 1 {
		t.Fatalf("expected 1 consolidated notification, got %d", len(notifications))
	}
	n := notifications[0]
	if n.Title != "Auctions Ending Soon" {
		t.Errorf("title = %q, want %q", n.Title, "Auctions Ending Soon")
	}
	for _, want := range []string{"The Coin Cabinet - Ancients Auction 35 (Lot 30)", "Classical Numismatic Group - Keystone 17 (Lot 95)"} {
		if !strings.Contains(n.Message, want) {
			t.Errorf("message %q missing %q", n.Message, want)
		}
	}
}

func TestNotifyAuctionEndingSoonNoLotsCreatesNoNotification(t *testing.T) {
	db := setupNotificationServiceDB(t)
	svc := newTestNotificationService(db)

	user := models.User{Username: "bidder", Email: "bidder@example.com", PasswordHash: "hash"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	svc.NotifyAuctionEndingSoon(user.ID, nil)

	var count int64
	db.Model(&models.Notification{}).Where("user_id = ?", user.ID).Count(&count)
	if count != 0 {
		t.Fatalf("expected 0 notifications, got %d", count)
	}
}
