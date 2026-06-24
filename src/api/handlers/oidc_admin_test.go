package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupOIDCAdminHandlerTest(t *testing.T) (*gorm.DB, *gin.Engine) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.SecurityEvent{}, &models.IPRule{}, &models.RefreshToken{}, &models.OIDCProvider{}, &models.ExternalIdentity{}); err != nil {
		t.Fatalf("failed to migrate test db: %v", err)
	}
	oidcRepo := repository.NewOIDCRepository(db)
	securitySvc := services.NewSecurityService(repository.NewSecurityRepository(db))
	handler := NewOIDCHandler(services.NewOIDCService(oidcRepo, nil).WithSecurity(securitySvc))

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("userId", uint(1))
		c.Next()
	})
	router.GET("/admin/oidc/providers", handler.ListAdminProviders)
	router.POST("/admin/oidc/providers", handler.CreateAdminProvider)
	router.PUT("/admin/oidc/providers/:providerId", handler.UpdateAdminProvider)
	return db, router
}

func oidcAdminJSONRequest(t *testing.T, router http.Handler, method, path string, body map[string]any) *httptest.ResponseRecorder {
	t.Helper()
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("failed to marshal request body: %v", err)
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	return recorder
}

func oidcAdminProviderPayload(secret string) map[string]any {
	return map[string]any{
		"name":                 "entra-work",
		"displayName":          "Microsoft Entra",
		"providerType":         "entra",
		"enabled":              true,
		"issuerUrl":            "https://login.microsoftonline.com/tenant/v2.0",
		"clientId":             "client-id",
		"clientSecret":         secret,
		"scopes":               []string{"openid", "profile", "email"},
		"callbackPath":         "/api/auth/oidc/1/callback",
		"requireVerifiedEmail": true,
	}
}

func TestOIDCAdminHandlerCreateAndListRedactClientSecret(t *testing.T) {
	_, router := setupOIDCAdminHandlerTest(t)
	secret := "super-secret-value"

	create := oidcAdminJSONRequest(t, router, http.MethodPost, "/admin/oidc/providers", oidcAdminProviderPayload(secret))
	if create.Code != http.StatusCreated {
		t.Fatalf("expected create status 201, got %d body=%s", create.Code, create.Body.String())
	}
	if strings.Contains(create.Body.String(), secret) {
		t.Fatalf("expected create response to redact client secret, got %s", create.Body.String())
	}
	if !strings.Contains(create.Body.String(), `"clientSecretConfigured":true`) {
		t.Fatalf("expected create response to expose configured flag, got %s", create.Body.String())
	}

	list := httptest.NewRecorder()
	router.ServeHTTP(list, httptest.NewRequest(http.MethodGet, "/admin/oidc/providers", nil))
	if list.Code != http.StatusOK {
		t.Fatalf("expected list status 200, got %d body=%s", list.Code, list.Body.String())
	}
	if strings.Contains(list.Body.String(), secret) {
		t.Fatalf("expected list response to redact client secret, got %s", list.Body.String())
	}
	if !strings.Contains(list.Body.String(), `"clientSecretConfigured":true`) {
		t.Fatalf("expected list response to expose configured flag, got %s", list.Body.String())
	}
}

func TestOIDCAdminHandlerUpdatePreservesSecretOnEmptyOrRedactedSecret(t *testing.T) {
	db, router := setupOIDCAdminHandlerTest(t)
	create := oidcAdminJSONRequest(t, router, http.MethodPost, "/admin/oidc/providers", oidcAdminProviderPayload("original-secret"))
	if create.Code != http.StatusCreated {
		t.Fatalf("expected create status 201, got %d body=%s", create.Code, create.Body.String())
	}

	emptySecret := oidcAdminProviderPayload("")
	emptySecret["displayName"] = "Microsoft Entra Updated"
	update := oidcAdminJSONRequest(t, router, http.MethodPut, "/admin/oidc/providers/1", emptySecret)
	if update.Code != http.StatusOK {
		t.Fatalf("expected empty-secret update status 200, got %d body=%s", update.Code, update.Body.String())
	}
	assertOIDCProviderSecret(t, db, 1, "original-secret")
	if strings.Contains(update.Body.String(), "original-secret") {
		t.Fatalf("expected update response to redact original secret, got %s", update.Body.String())
	}

	redactedSecret := oidcAdminProviderPayload("Configured")
	redactedSecret["displayName"] = "Microsoft Entra Redacted Update"
	update = oidcAdminJSONRequest(t, router, http.MethodPut, "/admin/oidc/providers/1", redactedSecret)
	if update.Code != http.StatusOK {
		t.Fatalf("expected redacted-secret update status 200, got %d body=%s", update.Code, update.Body.String())
	}
	assertOIDCProviderSecret(t, db, 1, "original-secret")

	rotatedSecret := oidcAdminProviderPayload("rotated-secret")
	rotatedSecret["displayName"] = "Microsoft Entra Rotated"
	update = oidcAdminJSONRequest(t, router, http.MethodPut, "/admin/oidc/providers/1", rotatedSecret)
	if update.Code != http.StatusOK {
		t.Fatalf("expected rotated-secret update status 200, got %d body=%s", update.Code, update.Body.String())
	}
	assertOIDCProviderSecret(t, db, 1, "rotated-secret")
	if strings.Contains(update.Body.String(), "rotated-secret") {
		t.Fatalf("expected rotated update response to redact new secret, got %s", update.Body.String())
	}
}

func assertOIDCProviderSecret(t *testing.T, db *gorm.DB, id uint, expected string) {
	t.Helper()
	var provider models.OIDCProvider
	if err := db.First(&provider, id).Error; err != nil {
		t.Fatalf("failed to load provider %d: %v", id, err)
	}
	if provider.ClientSecret != expected {
		t.Fatalf("expected provider secret %q, got %q", expected, provider.ClientSecret)
	}
}
