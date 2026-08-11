package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
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

const adminNumistaTestJWTSecret = "phase-6-admin-numista-test-secret"

func setupAdminNumistaRouter(t *testing.T) (*gin.Engine, *repository.SettingsRepository, *services.NumistaTelemetry) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open settings database: %v", err)
	}
	if err := db.AutoMigrate(&models.AppSetting{}); err != nil {
		t.Fatalf("migrate settings database: %v", err)
	}
	settingsRepo := repository.NewSettingsRepository(db)
	settings := services.NewSettingsService(settingsRepo)
	telemetry := services.NewNumistaTelemetry(20)
	handler := NewAdminNumistaHandler(telemetry, settings)

	router := gin.New()
	admin := router.Group("/api/admin")
	admin.Use(middleware.AuthRequired(adminNumistaTestJWTSecret, nil))
	admin.Use(AdminRequired())
	admin.GET("/numista/health", handler.Health)
	return router, settingsRepo, telemetry
}

func adminNumistaToken(t *testing.T, role models.UserRole) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"userId": float64(1),
		"role":   string(role),
		"exp":    time.Now().Add(time.Hour).Unix(),
	})
	signed, err := token.SignedString([]byte(adminNumistaTestJWTSecret))
	if err != nil {
		t.Fatalf("sign JWT: %v", err)
	}
	return signed
}

func getAdminNumistaHealth(
	t *testing.T,
	router http.Handler,
	token string,
) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, "/api/admin/numista/health", nil)
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}

func TestAdminNumistaHealthReturnsExactEmptyRedactedSummary(t *testing.T) {
	router, _, _ := setupAdminNumistaRouter(t)
	response := getAdminNumistaHealth(t, router, adminNumistaToken(t, models.RoleAdmin))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}

	var body map[string]json.RawMessage
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	gotKeys := make([]string, 0, len(body))
	for key := range body {
		gotKeys = append(gotKeys, key)
	}
	sort.Strings(gotKeys)
	wantKeys := []string{
		"broadRequestCount", "cancelledRequestCount", "coalescedRequestCount",
		"configurationValid", "configured", "detailRequestCount", "enrichmentAttempted",
		"enrichmentFailed", "enrichmentSucceeded", "p50ElapsedMs", "p95ElapsedMs", "statusCounts",
		"freshCacheHitCount", "freshCacheHitRate", "providerFailureCount", "providerLoadCount",
	}
	sort.Strings(wantKeys)
	if strings.Join(gotKeys, ",") != strings.Join(wantKeys, ",") {
		t.Fatalf("JSON keys = %v, want exact empty contract %v", gotKeys, wantKeys)
	}

	var summary models.NumistaHealthSummary
	if err := json.Unmarshal(response.Body.Bytes(), &summary); err != nil {
		t.Fatalf("decode summary: %v", err)
	}
	if summary.Configured || !summary.ConfigurationValid || len(summary.StatusCounts) != 0 ||
		summary.BroadRequestCount != 0 || summary.DetailRequestCount != 0 ||
		summary.LastCheckedAt != nil || summary.LastQuotaLimitedAt != nil ||
		summary.LastRetryAfterSeconds != nil {
		t.Fatalf("unexpected empty summary: %+v", summary)
	}
}

func TestAdminNumistaHealthReturnsPopulatedOperationalSummary(t *testing.T) {
	router, settings, telemetry := setupAdminNumistaRouter(t)
	if err := settings.Upsert(services.SettingNumistaAPIKey, "server-only-key"); err != nil {
		t.Fatal(err)
	}
	retryAfter := 120
	telemetry.Record(services.NumistaTelemetryEvent{
		OccurredAt: time.Date(2026, 8, 11, 12, 0, 0, 0, time.UTC),
		Path:       models.NumistaLookupPathDirect, Operation: "broad",
		Status: models.NumistaStatusSuccess, CacheOutcome: services.NumistaCacheOutcomeFreshHit,
		ElapsedMilliseconds: 15,
		CorrelationDigest:   "collector query and inscription",
	})
	telemetry.Record(services.NumistaTelemetryEvent{
		OccurredAt: time.Date(2026, 8, 11, 12, 1, 0, 0, time.UTC),
		Path:       models.NumistaLookupPathPhoto, Operation: "detail",
		Status: models.NumistaStatusQuotaLimited, CacheOutcome: services.NumistaCacheOutcomeLoader,
		ElapsedMilliseconds: 95,
		DetailAttemptCount:  5, DetailSuccessCount: 3, DetailFailureCount: 2,
		RetryCount: 1, RetryAfterSeconds: &retryAfter,
		CorrelationDigest: "visible dealer label and raw provider error",
	})

	response := getAdminNumistaHealth(t, router, adminNumistaToken(t, models.RoleAdmin))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	var summary models.NumistaHealthSummary
	if err := json.Unmarshal(response.Body.Bytes(), &summary); err != nil {
		t.Fatalf("decode summary: %v", err)
	}
	if !summary.Configured || !summary.ConfigurationValid ||
		summary.StatusCounts[models.NumistaStatusSuccess] != 0 ||
		summary.StatusCounts[models.NumistaStatusQuotaLimited] != 1 ||
		summary.BroadRequestCount != 0 || summary.DetailRequestCount != 1 ||
		summary.FreshCacheHitCount != 1 || summary.CoalescedRequestCount != 0 ||
		summary.ProviderLoadCount != 1 || summary.ProviderFailureCount != 1 ||
		summary.FreshCacheHitRate != 0.5 ||
		summary.P50ElapsedMs != 95 || summary.P95ElapsedMs != 95 ||
		summary.EnrichmentAttempted != 5 || summary.EnrichmentSucceeded != 3 || summary.EnrichmentFailed != 2 ||
		summary.LastOutcome != models.NumistaStatusQuotaLimited ||
		summary.LastRetryAfterSeconds == nil || *summary.LastRetryAfterSeconds != retryAfter {
		t.Fatalf("unexpected populated summary: %+v", summary)
	}
}

func TestAdminNumistaHealthReportsInvalidLiveConfiguration(t *testing.T) {
	router, settings, _ := setupAdminNumistaRouter(t)
	if err := settings.Upsert(services.SettingNumistaAPIKey, "configured-key"); err != nil {
		t.Fatal(err)
	}
	if err := settings.Upsert(services.SettingNumistaSearchTTLHours, "0"); err != nil {
		t.Fatal(err)
	}

	response := getAdminNumistaHealth(t, router, adminNumistaToken(t, models.RoleAdmin))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	var summary models.NumistaHealthSummary
	if err := json.Unmarshal(response.Body.Bytes(), &summary); err != nil {
		t.Fatalf("decode summary: %v", err)
	}
	if !summary.Configured || summary.ConfigurationValid {
		t.Fatalf("invalid stored configuration was not reported safely: %+v", summary)
	}
}

func TestAdminNumistaHealthRequiresAuthenticationAndAdminRole(t *testing.T) {
	router, _, _ := setupAdminNumistaRouter(t)
	if response := getAdminNumistaHealth(t, router, ""); response.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated status = %d, want 401: %s", response.Code, response.Body.String())
	}
	if response := getAdminNumistaHealth(t, router, adminNumistaToken(t, models.RoleUser)); response.Code != http.StatusForbidden {
		t.Fatalf("non-admin status = %d, want 403: %s", response.Code, response.Body.String())
	}
}

func TestAdminNumistaHealthJSONNeverExposesCredentialsOrCollectorText(t *testing.T) {
	router, settings, telemetry := setupAdminNumistaRouter(t)
	const key = "numista-secret-api-key"
	const query = "IMP TRAIANO obverse inscription"
	const label = "private dealer label"
	const rawError = "provider raw error response"
	if err := settings.Upsert(services.SettingNumistaAPIKey, key); err != nil {
		t.Fatal(err)
	}
	telemetry.Record(services.NumistaTelemetryEvent{
		OccurredAt: time.Date(2026, 8, 11, 12, 0, 0, 0, time.UTC),
		Path:       models.NumistaLookupPathPhoto, Operation: "broad", Status: models.NumistaStatusUnavailable,
		CorrelationDigest: strings.Join([]string{query, label, rawError}, " "),
	})

	response := getAdminNumistaHealth(t, router, adminNumistaToken(t, models.RoleAdmin))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	body := response.Body.String()
	for _, forbidden := range []string{key, "IMP TRAIANO", "inscription", label, rawError, "correlationDigest"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("health JSON leaked %q: %s", forbidden, body)
		}
	}
}
