package services

import (
	"errors"
	"net/url"
	"strings"
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
	if err := db.AutoMigrate(&models.User{}, &models.AuctionLot{}, &models.PriceAlert{}, &models.BidReminder{}, &models.AuctionAlertRun{}, &models.AppSetting{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	return db
}

func TestAuctionAlertEvaluatorFailedNotificationRemainsRetryable(t *testing.T) {
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

	settingsSvc := NewSettingsService(repository.NewSettingsRepository(db))
	evaluator := NewAuctionAlertEvaluator(
		repository.NewPriceAlertRepository(db),
		repository.NewBidReminderRepository(db),
		repository.NewUserRepository(db),
		NewPushoverService(settingsSvc, NewLogger(100)),
		NewLogger(100),
	)

	result, err := evaluator.Evaluate(now)
	if err == nil {
		t.Fatalf("Evaluate() error = nil, want notification failure")
	}
	if result.PriceAlertsTriggered != 0 || result.BidRemindersSent != 0 {
		t.Fatalf("result = %+v, want no successful notifications", result)
	}

	var reloadedAlert models.PriceAlert
	if err := db.First(&reloadedAlert, alert.ID).Error; err != nil {
		t.Fatalf("failed to reload alert: %v", err)
	}
	if reloadedAlert.IsTriggered || reloadedAlert.TriggeredAt != nil {
		t.Fatalf("failed notification consumed alert: %+v", reloadedAlert)
	}
	var reloadedReminder models.BidReminder
	if err := db.First(&reloadedReminder, reminder.ID).Error; err != nil {
		t.Fatalf("failed to reload reminder: %v", err)
	}
	if reloadedReminder.IsNotified || reloadedReminder.NotifiedAt != nil {
		t.Fatalf("failed notification consumed reminder: %+v", reloadedReminder)
	}
}

func TestAuctionAlertSchedulerRecordsNotificationFailures(t *testing.T) {
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
		repository.NewUserRepository(db),
		NewPushoverService(settingsSvc, logger),
		logger,
	)
	runRepo := repository.NewAuctionAlertRunRepository(db)
	scheduler := NewAuctionAlertScheduler(evaluator, runRepo, nil, settingsSvc, logger)

	run, err := scheduler.RunNowWithTrigger(&user.ID)
	if err == nil {
		t.Fatalf("RunNowWithTrigger() error = nil, want notification failure")
	}
	if run == nil {
		t.Fatalf("RunNowWithTrigger() run = nil")
	}
	if run.Status != "error" {
		t.Fatalf("run status = %q, want error", run.Status)
	}
	if !strings.Contains(run.ErrorMessage, "notification") {
		t.Fatalf("run error message = %q, want notification failure", run.ErrorMessage)
	}

	var persisted models.AuctionAlertRun
	if err := db.First(&persisted, run.ID).Error; err != nil {
		t.Fatalf("failed to reload run: %v", err)
	}
	if persisted.Status != "error" || persisted.ErrorMessage == "" {
		t.Fatalf("persisted run did not surface failure: %+v", persisted)
	}
	var reloadedAlert models.PriceAlert
	if err := db.First(&reloadedAlert, alert.ID).Error; err != nil {
		t.Fatalf("failed to reload alert: %v", err)
	}
	if reloadedAlert.IsTriggered || reloadedAlert.TriggeredAt != nil {
		t.Fatalf("failed scheduler notification consumed alert: %+v", reloadedAlert)
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

	var captured url.Values
	pushoverSvc, cleanup := newTestPushoverService(t, &captured)
	defer cleanup()
	evaluator := NewAuctionAlertEvaluator(
		repository.NewPriceAlertRepository(db),
		repository.NewBidReminderRepository(db),
		repository.NewUserRepository(db),
		pushoverSvc,
		NewLogger(100),
	)

	result, err := evaluator.Evaluate(now)
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}
	if result.LotsChecked != 1 || result.PriceAlertsTriggered != 1 || result.BidRemindersSent != 1 {
		t.Fatalf("first result = %+v, want one lot, one alert, one reminder", result)
	}
	if captured.Get("user") != "user-key" {
		t.Fatalf("pushover user = %q, want user-key", captured.Get("user"))
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
