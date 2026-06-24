package services

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupOIDCServiceTest(t *testing.T) (*gorm.DB, *OIDCService) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.SecurityEvent{}, &models.OIDCProvider{}, &models.ExternalIdentity{}); err != nil {
		t.Fatalf("failed to migrate test db: %v", err)
	}
	return db, NewOIDCService(repository.NewOIDCRepository(db), nil)
}

func TestOIDCServiceTestAdminProviderDiscoversEntraTenantIssuer(t *testing.T) {
	db, svc := setupOIDCServiceTest(t)
	issuer, closeServer := startOIDCDiscoveryServer(t, "/tenant-id/v2.0", nil)
	defer closeServer()
	provider := createOIDCTestProvider(t, db, models.OIDCProviderTypeEntra, issuer)

	result, err := svc.TestAdminProvider(context.Background(), provider.ID, OIDCAuditContext{})
	if err != nil {
		t.Fatalf("expected Entra discovery to succeed, got %v result=%+v", err, result)
	}
	if !result.Available || result.Message != "Discovery succeeded" {
		t.Fatalf("expected successful discovery result, got %+v", result)
	}
	if result.Issuer != issuer {
		t.Fatalf("expected issuer %q, got %q", issuer, result.Issuer)
	}
	if result.AuthorizationEndpoint != issuer+"/authorize" || result.TokenEndpoint != issuer+"/token" {
		t.Fatalf("expected discovered endpoints for issuer %q, got %+v", issuer, result)
	}

	var persisted models.OIDCProvider
	if err := db.First(&persisted, provider.ID).Error; err != nil {
		t.Fatalf("failed to reload provider: %v", err)
	}
	if persisted.LastTestStatus != models.OIDCProviderTestStatusOK || persisted.LastTestMessage != "Discovery succeeded" || persisted.LastTestedAt == nil {
		t.Fatalf("expected persisted successful test status, got status=%q message=%q tested=%v", persisted.LastTestStatus, persisted.LastTestMessage, persisted.LastTestedAt)
	}
}

func TestOIDCServiceTestAdminProviderDiscoversPocketIDIssuer(t *testing.T) {
	db, svc := setupOIDCServiceTest(t)
	issuer, closeServer := startOIDCDiscoveryServer(t, "", nil)
	defer closeServer()
	provider := createOIDCTestProvider(t, db, models.OIDCProviderTypePocketID, issuer)

	result, err := svc.TestAdminProvider(context.Background(), provider.ID, OIDCAuditContext{})
	if err != nil {
		t.Fatalf("expected Pocket ID discovery to succeed, got %v result=%+v", err, result)
	}
	if !result.Available || result.Issuer != issuer {
		t.Fatalf("expected successful Pocket ID discovery for %q, got %+v", issuer, result)
	}
}

func TestOIDCServiceTestAdminProviderRecordsFailedDiscoveryStatus(t *testing.T) {
	db, svc := setupOIDCServiceTest(t)
	issuer, closeServer := startOIDCDiscoveryServer(t, "", func(metadata map[string]string) {
		metadata["issuer"] = issuerForMismatch
	})
	defer closeServer()
	provider := createOIDCTestProvider(t, db, models.OIDCProviderTypePocketID, issuer)

	result, err := svc.TestAdminProvider(context.Background(), provider.ID, OIDCAuditContext{})
	if !errors.Is(err, ErrOIDCProviderDiscovery) {
		t.Fatalf("expected discovery error, got %v result=%+v", err, result)
	}
	if result.Available {
		t.Fatalf("expected unavailable result, got %+v", result)
	}

	var persisted models.OIDCProvider
	if err := db.First(&persisted, provider.ID).Error; err != nil {
		t.Fatalf("failed to reload provider: %v", err)
	}
	if persisted.LastTestStatus != models.OIDCProviderTestStatusFailed || persisted.LastTestedAt == nil {
		t.Fatalf("expected persisted failed status, got status=%q tested=%v", persisted.LastTestStatus, persisted.LastTestedAt)
	}
}

const issuerForMismatch = "http://localhost/wrong-issuer"

func startOIDCDiscoveryServer(t *testing.T, issuerPath string, mutate func(map[string]string)) (string, func()) {
	t.Helper()
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, ".well-known/openid-configuration") {
			http.NotFound(w, r)
			return
		}
		issuer := server.URL + issuerPath
		metadata := map[string]string{
			"issuer":                 issuer,
			"authorization_endpoint": issuer + "/authorize",
			"token_endpoint":         issuer + "/token",
			"jwks_uri":               issuer + "/jwks",
		}
		if mutate != nil {
			mutate(metadata)
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(metadata); err != nil {
			t.Fatalf("failed to encode discovery metadata: %v", err)
		}
	}))
	return server.URL + issuerPath, server.Close
}

func createOIDCTestProvider(t *testing.T, db *gorm.DB, providerType models.OIDCProviderType, issuer string) models.OIDCProvider {
	t.Helper()
	provider := models.OIDCProvider{
		Name:                 "provider-" + strings.ReplaceAll(strings.TrimPrefix(string(providerType), "pocket_"), "_", "-"),
		DisplayName:          string(providerType),
		ProviderType:         providerType,
		Enabled:              true,
		IssuerURL:            issuer,
		ClientID:             "client-id",
		ClientSecret:         "client-secret",
		Scopes:               models.StringList{"openid", "profile", "email"},
		CallbackPath:         "/api/auth/oidc/1/callback",
		RequireVerifiedEmail: true,
		LastTestStatus:       models.OIDCProviderTestStatusUnknown,
	}
	if providerType == models.OIDCProviderTypePocketID {
		provider.Name = "pocket-home"
	}
	if err := db.Create(&provider).Error; err != nil {
		t.Fatalf("failed to create OIDC provider: %v", err)
	}
	return provider
}
