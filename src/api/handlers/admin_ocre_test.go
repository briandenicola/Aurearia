package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/middleware"
	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

const adminOCRETestJWTSecret = "phase-6-admin-ocre-test-secret"

func setupAdminOCRERouter(t *testing.T) (*gin.Engine, *repository.SettingsRepository, *repository.DeepIdentificationRepository, *gorm.DB) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	if err := db.AutoMigrate(&models.AppSetting{}, &models.DeepIdentificationProviderRun{}); err != nil {
		t.Fatalf("migrate database: %v", err)
	}
	settingsRepo := repository.NewSettingsRepository(db)
	settings := services.NewSettingsService(settingsRepo)
	deepRepo := repository.NewDeepIdentificationRepository(db)
	handler := NewAdminOCREHandler(settings, deepRepo)

	router := gin.New()
	admin := router.Group("/api/admin")
	admin.Use(middleware.AuthRequired(adminOCRETestJWTSecret, nil))
	admin.Use(AdminRequired())
	admin.GET("/deep-identification/ocre/health", handler.Health)
	return router, settingsRepo, deepRepo, db
}

func adminOCREToken(t *testing.T, role models.UserRole) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"userId": float64(1),
		"role":   string(role),
		"exp":    time.Now().Add(time.Hour).Unix(),
	})
	signed, err := token.SignedString([]byte(adminOCRETestJWTSecret))
	if err != nil {
		t.Fatalf("sign JWT: %v", err)
	}
	return signed
}

func getAdminOCREHealth(t *testing.T, router http.Handler, token string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, "/api/admin/deep-identification/ocre/health", nil)
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}

// T040: default (flag-off) health reports a disabled but gate-valid provider
// with a null last outcome — no OCRE run has ever occurred.
func TestAdminOCREHealthDefaultDisabledNullOutcome(t *testing.T) {
	router, _, _, _ := setupAdminOCRERouter(t)
	response := getAdminOCREHealth(t, router, adminOCREToken(t, models.RoleAdmin))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	var summary models.OCREHealthSummary
	if err := json.Unmarshal(response.Body.Bytes(), &summary); err != nil {
		t.Fatalf("decode summary: %v", err)
	}
	if summary.Enabled {
		t.Fatalf("expected default OCRE disabled, got enabled")
	}
	if summary.CallBudget != 3 {
		t.Fatalf("expected default call budget 3, got %d", summary.CallBudget)
	}
	if !summary.GateValidated {
		t.Fatalf("expected default configuration to be gate-valid")
	}
	if summary.LastOutcome != nil || summary.LastCheckedAt != nil {
		t.Fatalf("expected null last outcome, got outcome=%v at=%v", summary.LastOutcome, summary.LastCheckedAt)
	}
}

// T040: with the flag on and a persisted OCRE run, health reflects enablement
// and the most recent outcome only.
func TestAdminOCREHealthReflectsFlagAndLatestRun(t *testing.T) {
	router, settingsRepo, deepRepo, db := setupAdminOCRERouter(t)
	if err := settingsRepo.Upsert(services.SettingDeepIdentificationOCREEnabled, "true"); err != nil {
		t.Fatal(err)
	}
	if err := settingsRepo.Upsert(services.SettingDeepIdentificationOCRECallBudget, "5"); err != nil {
		t.Fatal(err)
	}

	// An older no_match followed by a newer contributed run for OCRE.
	older := time.Date(2026, 8, 10, 12, 0, 0, 0, time.UTC)
	newer := time.Date(2026, 8, 11, 12, 0, 0, 0, time.UTC)
	if err := db.Create(&models.DeepIdentificationProviderRun{
		JobID: 1, UserID: 1, Provider: models.DeepProviderOCRE,
		Status: models.DeepProviderRunNoMatch, Automatable: true, CreatedAt: older, CompletedAt: &older,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.DeepIdentificationProviderRun{
		JobID: 2, UserID: 1, Provider: models.DeepProviderOCRE,
		Status: models.DeepProviderRunContributed, Automatable: true, CallCount: 2, LatencyMS: 40, CreatedAt: newer, CompletedAt: &newer,
	}).Error; err != nil {
		t.Fatal(err)
	}
	// A different provider's run must never surface on the OCRE health surface.
	if err := db.Create(&models.DeepIdentificationProviderRun{
		JobID: 3, UserID: 1, Provider: models.DeepProviderNumista,
		Status: models.DeepProviderRunFailed, Automatable: true, CreatedAt: newer.Add(time.Hour),
	}).Error; err != nil {
		t.Fatal(err)
	}
	_ = deepRepo

	response := getAdminOCREHealth(t, router, adminOCREToken(t, models.RoleAdmin))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	var summary models.OCREHealthSummary
	if err := json.Unmarshal(response.Body.Bytes(), &summary); err != nil {
		t.Fatalf("decode summary: %v", err)
	}
	if !summary.Enabled || summary.CallBudget != 5 || !summary.GateValidated {
		t.Fatalf("unexpected enablement: %+v", summary)
	}
	if summary.LastOutcome == nil || *summary.LastOutcome != models.DeepProviderRunContributed {
		t.Fatalf("expected latest outcome contributed, got %v", summary.LastOutcome)
	}
	if summary.LastCheckedAt == nil || !summary.LastCheckedAt.Equal(newer) {
		t.Fatalf("expected last checked at %v, got %v", newer, summary.LastCheckedAt)
	}
}

// T040: an invalid enablement configuration is reported as gate-invalid.
func TestAdminOCREHealthReportsInvalidGate(t *testing.T) {
	router, settingsRepo, _, _ := setupAdminOCRERouter(t)
	if err := settingsRepo.Upsert(services.SettingDeepIdentificationOCREEnabled, "not-a-bool"); err != nil {
		t.Fatal(err)
	}
	response := getAdminOCREHealth(t, router, adminOCREToken(t, models.RoleAdmin))
	var summary models.OCREHealthSummary
	if err := json.Unmarshal(response.Body.Bytes(), &summary); err != nil {
		t.Fatalf("decode summary: %v", err)
	}
	if summary.GateValidated {
		t.Fatalf("expected gate-invalid for unparseable flag")
	}
}

// T040: the OCRE health endpoint is admin-gated, identical to Numista health.
func TestAdminOCREHealthRequiresAuthenticationAndAdminRole(t *testing.T) {
	router, _, _, _ := setupAdminOCRERouter(t)
	if response := getAdminOCREHealth(t, router, ""); response.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated status = %d, want 401: %s", response.Code, response.Body.String())
	}
	if response := getAdminOCREHealth(t, router, adminOCREToken(t, models.RoleUser)); response.Code != http.StatusForbidden {
		t.Fatalf("non-admin status = %d, want 403: %s", response.Code, response.Body.String())
	}
}

// T040 (repository): GetLatestProviderStatus returns only the most recent OCRE
// row and never leaks another provider's run.
func TestGetLatestProviderStatusReturnsMostRecentOCREOnly(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	if err := db.AutoMigrate(&models.DeepIdentificationProviderRun{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	repo := repository.NewDeepIdentificationRepository(db)

	// No rows yet → (nil, nil).
	run, err := repo.GetLatestProviderStatus(models.DeepProviderOCRE)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if run != nil {
		t.Fatalf("expected nil for no rows, got %+v", run)
	}

	older := time.Date(2026, 8, 10, 12, 0, 0, 0, time.UTC)
	newer := time.Date(2026, 8, 11, 12, 0, 0, 0, time.UTC)
	seed := []models.DeepIdentificationProviderRun{
		{JobID: 1, UserID: 1, Provider: models.DeepProviderOCRE, Status: models.DeepProviderRunNoMatch, CreatedAt: older, CompletedAt: &older},
		{JobID: 2, UserID: 1, Provider: models.DeepProviderOCRE, Status: models.DeepProviderRunContributed, CreatedAt: newer, CompletedAt: &newer},
		{JobID: 3, UserID: 1, Provider: models.DeepProviderNumista, Status: models.DeepProviderRunFailed, CreatedAt: newer.Add(time.Hour), CompletedAt: &newer},
		{JobID: 4, UserID: 1, Provider: models.DeepProviderOCRE, Status: models.DeepProviderRunRunning, CreatedAt: newer.Add(2 * time.Hour)},
	}
	for i := range seed {
		if err := db.Create(&seed[i]).Error; err != nil {
			t.Fatal(err)
		}
	}

	run, err = repo.GetLatestProviderStatus(models.DeepProviderOCRE)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if run == nil {
		t.Fatalf("expected a run, got nil")
	}
	if run.Provider != models.DeepProviderOCRE || run.Status != models.DeepProviderRunContributed {
		t.Fatalf("expected most recent OCRE contributed, got %s/%s", run.Provider, run.Status)
	}
	if run.CompletedAt == nil || !run.CompletedAt.Equal(newer) {
		t.Fatalf("expected completed_at %v, got %v", newer, run.CompletedAt)
	}
}
