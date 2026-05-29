package services

import (
	"strings"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupAPIKeyRotationTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	if err := db.AutoMigrate(&models.AppSetting{}, &models.User{}, &models.ApiKey{}, &models.Notification{}); err != nil {
		t.Fatalf("failed to migrate test db: %v", err)
	}
	return db
}

func newTestAPIKeyRotationService(t *testing.T, db *gorm.DB) *APIKeyRotationService {
	t.Helper()
	settingsRepo := repository.NewSettingsRepository(db)
	settingsSvc := NewSettingsService(settingsRepo)
	notifRepo := repository.NewNotificationRepository(db)
	userRepo := repository.NewUserRepository(db)
	socialRepo := repository.NewSocialRepository(db)
	notifSvc := NewNotificationService(notifRepo, socialRepo, userRepo, nil, NewLogger(100))
	apiKeyRepo := repository.NewApiKeyRepository(db)
	return NewAPIKeyRotationService(apiKeyRepo, notifRepo, notifSvc, settingsSvc, NewLogger(100))
}

func TestAPIKeyRotationStartupSync_CreatesCutoffAndNotification(t *testing.T) {
	db := setupAPIKeyRotationTestDB(t)
	svc := newTestAPIKeyRotationService(t, db)

	user := models.User{Username: "alice", PasswordHash: "hash"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	legacyKey := models.ApiKey{
		UserID:    user.ID,
		KeyHash:   "h1",
		KeyPrefix: "abcd1234",
		Name:      "Legacy Key",
		CreatedAt: time.Now().UTC().Add(-2 * time.Hour),
	}
	if err := db.Create(&legacyKey).Error; err != nil {
		t.Fatalf("failed to create api key: %v", err)
	}

	svc.SyncFromStartup()

	var cutoffSetting models.AppSetting
	if err := db.First(&cutoffSetting, "key = ?", SettingAPIKeyRotationCutoffAt).Error; err != nil {
		t.Fatalf("expected cutoff setting, got error: %v", err)
	}
	if _, err := time.Parse(time.RFC3339, cutoffSetting.Value); err != nil {
		t.Fatalf("cutoff setting should be RFC3339, got %q: %v", cutoffSetting.Value, err)
	}

	var notifications []models.Notification
	if err := db.Where("user_id = ? AND type = ?", user.ID, NotificationTypeAPIKeyRotationRequired).Find(&notifications).Error; err != nil {
		t.Fatalf("failed to fetch notifications: %v", err)
	}
	if len(notifications) != 1 {
		t.Fatalf("expected 1 rotation notification, got %d", len(notifications))
	}
	if !strings.Contains(notifications[0].Message, "Legacy Key") {
		t.Fatalf("expected notification message to contain key name, got %q", notifications[0].Message)
	}
}

func TestAPIKeyRotationStartupSync_ClearsNotificationAfterRotation(t *testing.T) {
	db := setupAPIKeyRotationTestDB(t)
	svc := newTestAPIKeyRotationService(t, db)

	user := models.User{Username: "bob", PasswordHash: "hash"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	legacyKey := models.ApiKey{
		UserID:    user.ID,
		KeyHash:   "legacy",
		KeyPrefix: "old11111",
		Name:      "Old Script",
		CreatedAt: time.Now().UTC().Add(-2 * time.Hour),
	}
	if err := db.Create(&legacyKey).Error; err != nil {
		t.Fatalf("failed to create legacy key: %v", err)
	}

	// First startup sync creates the reminder.
	svc.SyncFromStartup()

	var cutoffSetting models.AppSetting
	if err := db.First(&cutoffSetting, "key = ?", SettingAPIKeyRotationCutoffAt).Error; err != nil {
		t.Fatalf("failed to load cutoff setting: %v", err)
	}
	cutoff, err := time.Parse(time.RFC3339, cutoffSetting.Value)
	if err != nil {
		t.Fatalf("failed to parse cutoff: %v", err)
	}

	// Simulate rotation: revoke old key and create replacement after cutoff.
	now := time.Now().UTC()
	if err := db.Model(&legacyKey).Update("revoked_at", &now).Error; err != nil {
		t.Fatalf("failed to revoke legacy key: %v", err)
	}

	newKey := models.ApiKey{
		UserID:    user.ID,
		KeyHash:   "new",
		KeyPrefix: "new22222",
		Name:      "Rotated Key",
		CreatedAt: cutoff.Add(1 * time.Second),
	}
	if err := db.Create(&newKey).Error; err != nil {
		t.Fatalf("failed to create replacement key: %v", err)
	}

	// Next startup sync should clear the stale reminder.
	svc.SyncFromStartup()

	var remaining int64
	if err := db.Model(&models.Notification{}).
		Where("user_id = ? AND type = ?", user.ID, NotificationTypeAPIKeyRotationRequired).
		Count(&remaining).Error; err != nil {
		t.Fatalf("failed counting notifications: %v", err)
	}
	if remaining != 0 {
		t.Fatalf("expected rotation notification to be cleared after rotation, found %d", remaining)
	}
}
