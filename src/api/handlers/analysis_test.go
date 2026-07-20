package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupAIStatusHandlerDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	if err := db.AutoMigrate(&models.AppSetting{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	return db
}

func aiStatusRequest(t *testing.T, handler *AnalysisHandler) map[string]interface{} {
	t.Helper()
	gin.SetMode(gin.TestMode)
	req := httptest.NewRequest(http.MethodGet, "/ai-status", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	handler.AIStatus(c)

	if w.Code != http.StatusOK {
		t.Fatalf("AIStatus status = %d, want 200, body=%s", w.Code, w.Body.String())
	}
	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	return body
}

func TestAIStatusReturnsUnavailableWhenNoProviderConfigured(t *testing.T) {
	db := setupAIStatusHandlerDB(t)
	settingsSvc := services.NewSettingsService(repository.NewSettingsRepository(db))
	handler := NewAnalysisHandler(nil, nil, settingsSvc, services.NewLogger(100))

	body := aiStatusRequest(t, handler)

	if body["available"] != false {
		t.Errorf("available = %v, want false", body["available"])
	}
	if body["provider"] != "" {
		t.Errorf("provider = %v, want empty", body["provider"])
	}
}

func TestAIStatusReturnsAvailableForConfiguredAnthropic(t *testing.T) {
	db := setupAIStatusHandlerDB(t)
	settingsSvc := services.NewSettingsService(repository.NewSettingsRepository(db))
	if err := settingsSvc.SetSetting(services.SettingAIProvider, "anthropic"); err != nil {
		t.Fatalf("failed to set provider: %v", err)
	}
	if err := settingsSvc.SetSetting(services.SettingAnthropicAPIKey, "test-key"); err != nil {
		t.Fatalf("failed to set api key: %v", err)
	}
	if err := settingsSvc.SetSetting(services.SettingAnthropicModel, "claude-sonnet-4-5"); err != nil {
		t.Fatalf("failed to set model: %v", err)
	}
	handler := NewAnalysisHandler(nil, nil, settingsSvc, services.NewLogger(100))

	body := aiStatusRequest(t, handler)

	if body["available"] != true {
		t.Errorf("available = %v, want true", body["available"])
	}
	if body["provider"] != "anthropic" {
		t.Errorf("provider = %v, want anthropic", body["provider"])
	}
	if body["model"] != "claude-sonnet-4-5" {
		t.Errorf("model = %v, want claude-sonnet-4-5", body["model"])
	}
}

func TestAIStatusReturnsUnavailableForAnthropicMissingAPIKey(t *testing.T) {
	db := setupAIStatusHandlerDB(t)
	settingsSvc := services.NewSettingsService(repository.NewSettingsRepository(db))
	if err := settingsSvc.SetSetting(services.SettingAIProvider, "anthropic"); err != nil {
		t.Fatalf("failed to set provider: %v", err)
	}
	handler := NewAnalysisHandler(nil, nil, settingsSvc, services.NewLogger(100))

	body := aiStatusRequest(t, handler)

	if body["available"] != false {
		t.Errorf("available = %v, want false", body["available"])
	}
	message, _ := body["message"].(string)
	if message == "" {
		t.Error("expected a message explaining the missing API key")
	}
}

func TestAIStatusReturnsAvailableForReachableOllamaModel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/show" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"license":"MIT"}`))
	}))
	defer server.Close()

	db := setupAIStatusHandlerDB(t)
	settingsSvc := services.NewSettingsService(repository.NewSettingsRepository(db))
	if err := settingsSvc.SetSetting(services.SettingAIProvider, "ollama"); err != nil {
		t.Fatalf("failed to set provider: %v", err)
	}
	if err := settingsSvc.SetSetting(services.SettingOllamaURL, server.URL); err != nil {
		t.Fatalf("failed to set ollama url: %v", err)
	}
	if err := settingsSvc.SetSetting(services.SettingOllamaModel, "llava"); err != nil {
		t.Fatalf("failed to set ollama model: %v", err)
	}
	handler := NewAnalysisHandler(nil, nil, settingsSvc, services.NewLogger(100))

	body := aiStatusRequest(t, handler)

	if body["available"] != true {
		t.Errorf("available = %v, want true", body["available"])
	}
	if body["provider"] != "ollama" {
		t.Errorf("provider = %v, want ollama", body["provider"])
	}
}

func TestAIStatusReturnsUnavailableWhenOllamaUnreachable(t *testing.T) {
	db := setupAIStatusHandlerDB(t)
	settingsSvc := services.NewSettingsService(repository.NewSettingsRepository(db))
	if err := settingsSvc.SetSetting(services.SettingAIProvider, "ollama"); err != nil {
		t.Fatalf("failed to set provider: %v", err)
	}
	if err := settingsSvc.SetSetting(services.SettingOllamaURL, "http://127.0.0.1:1"); err != nil {
		t.Fatalf("failed to set ollama url: %v", err)
	}
	handler := NewAnalysisHandler(nil, nil, settingsSvc, services.NewLogger(100))

	body := aiStatusRequest(t, handler)

	if body["available"] != false {
		t.Errorf("available = %v, want false", body["available"])
	}
	message, _ := body["message"].(string)
	if message == "" {
		t.Error("expected a message explaining the connection failure")
	}
}

func TestAIStatusReturnsUnavailableWhenOllamaModelMissing(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"model not found"}`))
	}))
	defer server.Close()

	db := setupAIStatusHandlerDB(t)
	settingsSvc := services.NewSettingsService(repository.NewSettingsRepository(db))
	if err := settingsSvc.SetSetting(services.SettingAIProvider, "ollama"); err != nil {
		t.Fatalf("failed to set provider: %v", err)
	}
	if err := settingsSvc.SetSetting(services.SettingOllamaURL, server.URL); err != nil {
		t.Fatalf("failed to set ollama url: %v", err)
	}
	if err := settingsSvc.SetSetting(services.SettingOllamaModel, "missing-model"); err != nil {
		t.Fatalf("failed to set ollama model: %v", err)
	}
	handler := NewAnalysisHandler(nil, nil, settingsSvc, services.NewLogger(100))

	body := aiStatusRequest(t, handler)

	if body["available"] != false {
		t.Errorf("available = %v, want false", body["available"])
	}
}
