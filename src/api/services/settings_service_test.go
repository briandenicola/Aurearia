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

func TestGetSetting_CoinCategories_ReturnsDefault(t *testing.T) {
	svc, _ := newTestSettingsService(t)

	got := svc.GetSetting(SettingCoinCategories)
	expected := "Roman\nGreek\nByzantine\nModern\nOther"
	if got != expected {
		t.Errorf("GetSetting(CoinCategories) = %q, want default %q", got, expected)
	}
}

func TestGetSetting_CoinEras_ReturnsDefault(t *testing.T) {
	svc, _ := newTestSettingsService(t)

	got := svc.GetSetting(SettingCoinEras)
	expected := "ancient\nmedieval\nmodern"
	if got != expected {
		t.Errorf("GetSetting(CoinEras) = %q, want default %q", got, expected)
	}
}

func TestResolveLLMConfigAnthropicOmitsOllamaOnlyURLs(t *testing.T) {
	svc, db := newTestSettingsService(t)

	db.Create(&models.AppSetting{Key: SettingAIProvider, Value: "anthropic"})
	db.Create(&models.AppSetting{Key: SettingAnthropicAPIKey, Value: "anthropic-key"})
	db.Create(&models.AppSetting{Key: SettingAnthropicModel, Value: "claude-test"})
	db.Create(&models.AppSetting{Key: SettingOllamaURL, Value: "https://ai.denicolafamily.com"})
	db.Create(&models.AppSetting{Key: SettingSearXNGURL, Value: "https://search.denicolafamily.com"})

	cfg, err := svc.ResolveLLMConfig()
	if err != nil {
		t.Fatalf("ResolveLLMConfig returned error: %v", err)
	}

	if cfg.Provider != "anthropic" {
		t.Fatalf("Provider = %q, want anthropic", cfg.Provider)
	}
	if cfg.APIKey != "anthropic-key" {
		t.Fatalf("APIKey = %q, want anthropic-key", cfg.APIKey)
	}
	if cfg.Model != "claude-test" {
		t.Fatalf("Model = %q, want claude-test", cfg.Model)
	}
	if cfg.OllamaURL != "" || cfg.SearXNGURL != "" {
		t.Fatalf("Anthropic config included Ollama-only URLs: OllamaURL=%q SearXNGURL=%q", cfg.OllamaURL, cfg.SearXNGURL)
	}
}

func TestResolveLLMConfigOllamaIncludesOllamaOnlyURLs(t *testing.T) {
	svc, db := newTestSettingsService(t)

	db.Create(&models.AppSetting{Key: SettingAIProvider, Value: "ollama"})
	db.Create(&models.AppSetting{Key: SettingOllamaModel, Value: "llava-test"})
	db.Create(&models.AppSetting{Key: SettingOllamaURL, Value: "http://ollama:11434"})
	db.Create(&models.AppSetting{Key: SettingSearXNGURL, Value: "http://searxng:8080"})
	db.Create(&models.AppSetting{Key: SettingAnthropicAPIKey, Value: "anthropic-key"})

	cfg, err := svc.ResolveLLMConfig()
	if err != nil {
		t.Fatalf("ResolveLLMConfig returned error: %v", err)
	}

	if cfg.Provider != "ollama" {
		t.Fatalf("Provider = %q, want ollama", cfg.Provider)
	}
	if cfg.Model != "llava-test" {
		t.Fatalf("Model = %q, want llava-test", cfg.Model)
	}
	if cfg.OllamaURL != "http://ollama:11434" || cfg.SearXNGURL != "http://searxng:8080" {
		t.Fatalf("Ollama config missing Ollama URLs: OllamaURL=%q SearXNGURL=%q", cfg.OllamaURL, cfg.SearXNGURL)
	}
	if cfg.APIKey != "" {
		t.Fatalf("Ollama config included Anthropic API key")
	}
}

func TestSetSetting_CoinCategories_AllowsCustomization(t *testing.T) {
	svc, _ := newTestSettingsService(t)

	customCategories := "Imperial\nRepublican\nProvincial\nOther"
	if err := svc.SetSetting(SettingCoinCategories, customCategories); err != nil {
		t.Fatalf("SetSetting(CoinCategories) failed: %v", err)
	}

	got := svc.GetSetting(SettingCoinCategories)
	if got != customCategories {
		t.Errorf("after SetSetting, GetSetting(CoinCategories) = %q, want %q", got, customCategories)
	}
}

func TestSetSetting_CoinEras_AllowsCustomization(t *testing.T) {
	svc, _ := newTestSettingsService(t)

	customEras := "BC\nAD 1-500\nAD 500-1500\nModern"
	if err := svc.SetSetting(SettingCoinEras, customEras); err != nil {
		t.Fatalf("SetSetting(CoinEras) failed: %v", err)
	}

	got := svc.GetSetting(SettingCoinEras)
	if got != customEras {
		t.Errorf("after SetSetting, GetSetting(CoinEras) = %q, want %q", got, customEras)
	}
}

func TestGetAllSettings_IncludesCoinCategoriesAndEras(t *testing.T) {
	svc, _ := newTestSettingsService(t)

	all := svc.GetAllSettings()

	if _, ok := all[SettingCoinCategories]; !ok {
		t.Error("GetAllSettings does not include CoinCategories")
	}

	if _, ok := all[SettingCoinEras]; !ok {
		t.Error("GetAllSettings does not include CoinEras")
	}

	expectedCategories := "Roman\nGreek\nByzantine\nModern\nOther"
	if all[SettingCoinCategories] != expectedCategories {
		t.Errorf("GetAllSettings[CoinCategories] = %q, want default %q", all[SettingCoinCategories], expectedCategories)
	}

	expectedEras := "ancient\nmedieval\nmodern"
	if all[SettingCoinEras] != expectedEras {
		t.Errorf("GetAllSettings[CoinEras] = %q, want default %q", all[SettingCoinEras], expectedEras)
	}
}

func TestGetAllSettings_IncludesPublicAppURLDefault(t *testing.T) {
	svc, _ := newTestSettingsService(t)

	all := svc.GetAllSettings()
	if got, ok := all[SettingPublicAppURL]; !ok || got != "" {
		t.Errorf("GetAllSettings[PublicAppURL] = %q (present %v), want blank default", got, ok)
	}
}
