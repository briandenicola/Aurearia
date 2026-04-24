package services

import (
	"testing"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupSettingsTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	if err := db.AutoMigrate(&models.AppSetting{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	return db
}

func newTestSettingsService(t *testing.T) (*SettingsService, *gorm.DB) {
	t.Helper()
	db := setupSettingsTestDB(t)
	repo := repository.NewSettingsRepository(db)
	svc := NewSettingsService(repo)
	return svc, db
}

func TestGetSetting_ExistingKey(t *testing.T) {
	svc, db := newTestSettingsService(t)

	db.Create(&models.AppSetting{Key: "TestKey", Value: "TestValue"})

	got := svc.GetSetting("TestKey")
	if got != "TestValue" {
		t.Errorf("GetSetting(TestKey) = %q, want %q", got, "TestValue")
	}
}

func TestGetSetting_MissingKeyWithDefault(t *testing.T) {
	svc, _ := newTestSettingsService(t)

	got := svc.GetSetting(SettingOllamaURL)
	if got != "http://localhost:11434" {
		t.Errorf("GetSetting(OllamaURL) = %q, want default", got)
	}
}

func TestGetSetting_MissingKeyNoDefault(t *testing.T) {
	svc, _ := newTestSettingsService(t)

	got := svc.GetSetting("NonExistentKey")
	if got != "" {
		t.Errorf("GetSetting(NonExistentKey) = %q, want empty string", got)
	}
}

func TestGetSetting_EmptyValueReturnsDefault(t *testing.T) {
	svc, db := newTestSettingsService(t)

	db.Create(&models.AppSetting{Key: SettingOllamaModel, Value: ""})

	got := svc.GetSetting(SettingOllamaModel)
	if got != "llava" {
		t.Errorf("GetSetting(OllamaModel) with empty DB value = %q, want default %q", got, "llava")
	}
}

func TestGetSetting_EmptyAIProviderReturnsEmpty(t *testing.T) {
	svc, db := newTestSettingsService(t)

	db.Create(&models.AppSetting{Key: SettingAIProvider, Value: ""})

	got := svc.GetSetting(SettingAIProvider)
	if got != "" {
		t.Errorf("GetSetting(AIProvider) with empty value = %q, want empty (special case)", got)
	}
}

func TestSetSetting_CreatesNew(t *testing.T) {
	svc, _ := newTestSettingsService(t)

	if err := svc.SetSetting("NewKey", "NewValue"); err != nil {
		t.Fatalf("SetSetting failed: %v", err)
	}

	got := svc.GetSetting("NewKey")
	if got != "NewValue" {
		t.Errorf("after SetSetting, GetSetting = %q, want %q", got, "NewValue")
	}
}

func TestSetSetting_UpdatesExisting(t *testing.T) {
	svc, db := newTestSettingsService(t)

	if err := svc.SetSetting("Key", "Original"); err != nil {
		t.Fatalf("SetSetting (create) failed: %v", err)
	}
	if err := svc.SetSetting("Key", "Updated"); err != nil {
		t.Fatalf("SetSetting (update) failed: %v", err)
	}

	got := svc.GetSetting("Key")
	if got != "Updated" {
		t.Errorf("after update, GetSetting = %q, want %q", got, "Updated")
	}

	var count int64
	db.Model(&models.AppSetting{}).Where("key = ?", "Key").Count(&count)
	if count != 1 {
		t.Errorf("expected 1 row for key, got %d", count)
	}
}

func TestGetAllSettings_IncludesDefaultsAndDBValues(t *testing.T) {
	svc, db := newTestSettingsService(t)

	db.Create(&models.AppSetting{Key: SettingOllamaURL, Value: "http://custom:11434"})

	all := svc.GetAllSettings()

	if all[SettingOllamaURL] != "http://custom:11434" {
		t.Errorf("GetAllSettings[OllamaURL] = %q, want custom value", all[SettingOllamaURL])
	}

	if all[SettingOllamaModel] != "llava" {
		t.Errorf("GetAllSettings[OllamaModel] = %q, want default", all[SettingOllamaModel])
	}

	if len(all) < len(settingDefaults) {
		t.Errorf("GetAllSettings returned %d entries, want at least %d", len(all), len(settingDefaults))
	}
}

func TestGetAllSettings_EmptyDBValuesUseDefaults(t *testing.T) {
	svc, db := newTestSettingsService(t)

	db.Create(&models.AppSetting{Key: SettingOllamaModel, Value: ""})

	all := svc.GetAllSettings()
	if all[SettingOllamaModel] != "llava" {
		t.Errorf("GetAllSettings[OllamaModel] with empty DB value = %q, want default", all[SettingOllamaModel])
	}
}

func TestGetSettingDefaults_ReturnsIndependentCopy(t *testing.T) {
	svc, _ := newTestSettingsService(t)

	defaults := svc.GetSettingDefaults()
	defaults["FakeKey"] = "FakeValue"

	defaults2 := svc.GetSettingDefaults()
	if _, exists := defaults2["FakeKey"]; exists {
		t.Error("GetSettingDefaults returned a reference to the internal map, not a copy")
	}
}
