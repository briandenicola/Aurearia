package services

import (
	"errors"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupAuctionAlertServiceDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("failed to get sql db: %v", err)
	}
	// NotificationService fires Pushover delivery on a background goroutine; a second
	// connection to an unshared ":memory:" sqlite DB would see an empty schema, so this
	// pins the pool to the single connection that ran AutoMigrate (see wishlist_search_alert_service_test.go).
	sqlDB.SetMaxOpenConns(1)
	if err := db.AutoMigrate(&models.User{}, &models.AuctionLot{}, &models.PriceAlert{}, &models.BidReminder{}, &models.AuctionAlertRun{}, &models.AppSetting{}, &models.Notification{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	return db
}

func newTestAuctionNotificationService(db *gorm.DB, logger *Logger) *NotificationService {
	settingsSvc := NewSettingsService(repository.NewSettingsRepository(db))
	return NewNotificationService(
		repository.NewNotificationRepository(db),
		nil,
		repository.NewUserRepository(db),
		NewPushoverService(settingsSvc, logger),
		logger,
	)
}

// TestAuctionAlertEvaluatorClaimsAlertsEvenWithoutPushover guards against the pre-F027
// behavior this replaced: an alert/reminder used to be gated on the user having Pushover
// configured (ErrPushoverNotConfigured aborted the whole notification, leaving it pending
// forever for a non-Pushover user). Now the in-app notification is the channel of record —
// it must fire, and the alert/reminder must be claimed, regardless of Pushover.
func TestAuctionAlertEvaluatorClaimsAlertsEvenWithoutPushover(t *testing.T) {
	db := setupAuctionAlertServiceDB(t)
	user := models.User{
		Username: "bidder", Email: "bidder@example.com", PasswordHash: "hash",
		// Deliberately no Pushover configured.
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	now := time.Now()
	currentBid := 125.0
	endTime := now.Add(20 * time.Minute)
	lot := models.AuctionLot{
		UserID:         user.ID,
		NumisBidsURL:   "https://example.com/lot",
		SourceURL:      "https://example.com/lot",
		Title:          "Tracked lot",
		AuctionHouse:   "CNG",
		SaleName:       "Keystone",
		LotNumber:      42,
		Status:         models.AuctionStatusBidding,
		CurrentBid:     &currentBid,
		Currency:       "USD",
		AuctionEndTime: &endTime,
	}
	if err := db.Create(&lot).Error; err != nil {
		t.Fatalf("failed to create lot: %v", err)
	}
	alert := models.PriceAlert{UserID: user.ID, AuctionLotID: lot.ID, TargetPrice: 100, Direction: "above"}
	reminder := models.BidReminder{UserID: user.ID, AuctionLotID: lot.ID, MinutesBefore: 30}
	if err := db.Create(&alert).Error; err != nil {
		t.Fatalf("failed to create alert: %v", err)
	}
	if err := db.Create(&reminder).Error; err != nil {
		t.Fatalf("failed to create reminder: %v", err)
	}

	logger := NewLogger(100)
	evaluator := NewAuctionAlertEvaluator(
		repository.NewPriceAlertRepository(db),
		repository.NewBidReminderRepository(db),
		newTestAuctionNotificationService(db, logger),
		logger,
	)

	result, err := evaluator.Evaluate(now)
	if err != nil {
		t.Fatalf("Evaluate() error = %v, want success even without Pushover configured", err)
	}
	if result.PriceAlertsTriggered != 1 || result.BidRemindersSent != 1 {
		t.Fatalf("result = %+v, want one alert and one reminder claimed", result)
	}

	var reloadedAlert models.PriceAlert
	if err := db.First(&reloadedAlert, alert.ID).Error; err != nil {
		t.Fatalf("failed to reload alert: %v", err)
	}
	if !reloadedAlert.IsTriggered || reloadedAlert.TriggeredAt == nil {
		t.Fatalf("alert was not claimed despite no Pushover configured: %+v", reloadedAlert)
	}
	var reloadedReminder models.BidReminder
	if err := db.First(&reloadedReminder, reminder.ID).Error; err != nil {
		t.Fatalf("failed to reload reminder: %v", err)
	}
	if !reloadedReminder.IsNotified || reloadedReminder.NotifiedAt == nil {
		t.Fatalf("reminder was not claimed despite no Pushover configured: %+v", reloadedReminder)
	}

	var notifications []models.Notification
	if err := db.Where("user_id = ?", user.ID).Find(&notifications).Error; err != nil {
		t.Fatalf("failed to query notifications: %v", err)
	}
	if len(notifications) != 2 {
		t.Fatalf("got %d in-app notifications, want 2 (one price alert, one bid reminder)", len(notifications))
	}
}

// TestAuctionAlertSchedulerSucceedsWithoutPushover mirrors the evaluator-level test at the
// scheduler level: a run should complete successfully purely on the strength of the in-app
// notification channel, without Pushover configured.
func TestAuctionAlertSchedulerSucceedsWithoutPushover(t *testing.T) {
	db := setupAuctionAlertServiceDB(t)
	user := models.User{Username: "bidder", Email: "bidder@example.com", PasswordHash: "hash"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	currentBid := 125.0
	lot := models.AuctionLot{
		UserID:       user.ID,
		NumisBidsURL: "https://example.com/lot",
		SourceURL:    "https://example.com/lot",
		Title:        "Tracked lot",
		Status:       models.AuctionStatusBidding,
		CurrentBid:   &currentBid,
		Currency:     "USD",
	}
	if err := db.Create(&lot).Error; err != nil {
		t.Fatalf("failed to create lot: %v", err)
	}
	alert := models.PriceAlert{UserID: user.ID, AuctionLotID: lot.ID, TargetPrice: 100, Direction: "above"}
	if err := db.Create(&alert).Error; err != nil {
		t.Fatalf("failed to create alert: %v", err)
	}

	settingsSvc := NewSettingsService(repository.NewSettingsRepository(db))
	logger := NewLogger(100)
	evaluator := NewAuctionAlertEvaluator(
		repository.NewPriceAlertRepository(db),
		repository.NewBidReminderRepository(db),
		newTestAuctionNotificationService(db, logger),
		logger,
	)
	runRepo := repository.NewAuctionAlertRunRepository(db)
	scheduler := NewAuctionAlertScheduler(evaluator, runRepo, nil, settingsSvc, logger)

	run, err := scheduler.RunNowWithTrigger(&user.ID)
	if err != nil {
		t.Fatalf("RunNowWithTrigger() error = %v, want success even without Pushover configured", err)
	}
	if run == nil {
		t.Fatalf("RunNowWithTrigger() run = nil")
	}
	if run.Status != "success" {
		t.Fatalf("run status = %q, want success (%s)", run.Status, run.ErrorMessage)
	}

	var reloadedAlert models.PriceAlert
	if err := db.First(&reloadedAlert, alert.ID).Error; err != nil {
		t.Fatalf("failed to reload alert: %v", err)
	}
	if !reloadedAlert.IsTriggered {
		t.Fatalf("alert was not claimed: %+v", reloadedAlert)
	}
}

func TestAuctionAlertServiceCreateRequiresOwnedWatchedLot(t *testing.T) {
	db := setupAuctionAlertServiceDB(t)
	owner := models.User{Username: "owner", Email: "owner@example.com", PasswordHash: "hash"}
	other := models.User{Username: "other", Email: "other@example.com", PasswordHash: "hash"}
	if err := db.Create(&owner).Error; err != nil {
		t.Fatalf("failed to create owner: %v", err)
	}
	if err := db.Create(&other).Error; err != nil {
		t.Fatalf("failed to create other: %v", err)
	}
	lot := models.AuctionLot{
		UserID:       owner.ID,
		NumisBidsURL: "https://example.com/lot",
		SourceURL:    "https://example.com/lot",
		Title:        "Watched lot",
		Status:       models.AuctionStatusWatching,
	}
	if err := db.Create(&lot).Error; err != nil {
		t.Fatalf("failed to create lot: %v", err)
	}

	service := NewAuctionAlertService(
		repository.NewPriceAlertRepository(db),
		repository.NewBidReminderRepository(db),
		repository.NewAuctionLotRepository(db),
	)

	if _, err := service.CreateAlert(owner.ID, PriceAlertCreateRequest{AuctionLotID: lot.ID, TargetPrice: 100, Direction: "above"}); err != nil {
		t.Fatalf("owner CreateAlert() error = %v", err)
	}
	if _, err := service.CreateReminder(owner.ID, BidReminderCreateRequest{AuctionLotID: lot.ID, MinutesBefore: 30}); err != nil {
		t.Fatalf("owner CreateReminder() error = %v", err)
	}
	if _, err := service.CreateAlert(other.ID, PriceAlertCreateRequest{AuctionLotID: lot.ID, TargetPrice: 100}); !errors.Is(err, ErrAuctionLotNotWatchable) {
		t.Fatalf("other CreateAlert() error = %v, want ErrAuctionLotNotWatchable", err)
	}

	if err := db.Model(&lot).Update("status", string(models.AuctionStatusWon)).Error; err != nil {
		t.Fatalf("failed to update lot status: %v", err)
	}
	if _, err := service.CreateReminder(owner.ID, BidReminderCreateRequest{AuctionLotID: lot.ID, MinutesBefore: 30}); !errors.Is(err, ErrAuctionLotNotWatchable) {
		t.Fatalf("won lot CreateReminder() error = %v, want ErrAuctionLotNotWatchable", err)
	}
}

func TestAuctionAlertEvaluatorTriggersOnce(t *testing.T) {
	db := setupAuctionAlertServiceDB(t)
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

	now := time.Now()
	currentBid := 125.0
	endTime := now.Add(20 * time.Minute)
	lot := models.AuctionLot{
		UserID:         user.ID,
		NumisBidsURL:   "https://example.com/lot",
		SourceURL:      "https://example.com/lot",
		Title:          "Tracked lot",
		AuctionHouse:   "CNG",
		SaleName:       "Keystone",
		LotNumber:      42,
		Status:         models.AuctionStatusBidding,
		CurrentBid:     &currentBid,
		Currency:       "USD",
		AuctionEndTime: &endTime,
	}
	if err := db.Create(&lot).Error; err != nil {
		t.Fatalf("failed to create lot: %v", err)
	}
	alert := models.PriceAlert{UserID: user.ID, AuctionLotID: lot.ID, TargetPrice: 100, Direction: "above"}
	reminder := models.BidReminder{UserID: user.ID, AuctionLotID: lot.ID, MinutesBefore: 30}
	if err := db.Create(&alert).Error; err != nil {
		t.Fatalf("failed to create alert: %v", err)
	}
	if err := db.Create(&reminder).Error; err != nil {
		t.Fatalf("failed to create reminder: %v", err)
	}

	logger := NewLogger(100)
	evaluator := NewAuctionAlertEvaluator(
		repository.NewPriceAlertRepository(db),
		repository.NewBidReminderRepository(db),
		newTestAuctionNotificationService(db, logger),
		logger,
	)

	result, err := evaluator.Evaluate(now)
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}
	if result.LotsChecked != 1 || result.PriceAlertsTriggered != 1 || result.BidRemindersSent != 1 {
		t.Fatalf("first result = %+v, want one lot, one alert, one reminder", result)
	}

	var notifications []models.Notification
	if err := db.Where("user_id = ?", user.ID).Find(&notifications).Error; err != nil {
		t.Fatalf("failed to query notifications: %v", err)
	}
	if len(notifications) != 2 {
		t.Fatalf("got %d in-app notifications after first Evaluate(), want 2", len(notifications))
	}

	result, err = evaluator.Evaluate(now.Add(time.Minute))
	if err != nil {
		t.Fatalf("second Evaluate() error = %v", err)
	}
	if result.PriceAlertsTriggered != 0 || result.BidRemindersSent != 0 {
		t.Fatalf("second result = %+v, want idempotent no-op", result)
	}

	var reloadedAlert models.PriceAlert
	if err := db.First(&reloadedAlert, alert.ID).Error; err != nil {
		t.Fatalf("failed to reload alert: %v", err)
	}
	if !reloadedAlert.IsTriggered || reloadedAlert.TriggeredAt == nil {
		t.Fatalf("alert not marked triggered: %+v", reloadedAlert)
	}
	var reloadedReminder models.BidReminder
	if err := db.First(&reloadedReminder, reminder.ID).Error; err != nil {
		t.Fatalf("failed to reload reminder: %v", err)
	}
	if !reloadedReminder.IsNotified || reloadedReminder.NotifiedAt == nil {
		t.Fatalf("reminder not marked notified: %+v", reloadedReminder)
	}
}
