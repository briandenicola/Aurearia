package services

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/glebarez/sqlite"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/oauth2"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

func setupOIDCLoginServiceTest(t *testing.T) (*gorm.DB, *OIDCService) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.RefreshToken{}, &models.SecurityEvent{}, &models.WebAuthnCredential{}, &models.OIDCProvider{}, &models.ExternalIdentity{}, &models.OIDCAuthState{}); err != nil {
		t.Fatalf("failed to migrate test db: %v", err)
	}
	authSvc := NewAuthService(repository.NewAuthRepository(db), "test-jwt-secret")
	return db, NewOIDCService(repository.NewOIDCRepository(db), nil).WithAuth(authSvc).WithSecurity(NewSecurityService(repository.NewSecurityRepository(db)))
}

func TestOIDCServiceLoginCallbackIssuesTokensForLinkedIdentity(t *testing.T) {
	db, svc := setupOIDCLoginServiceTest(t)
	issuer := startMockOIDCProviderWithOptions(t, oidcMockProviderOptions{
		Subject:             "subject-123",
		Email:               "collector@example.com",
		EmailVerified:       true,
		ExpectedRedirectURI: "http://app.example/auth/oidc/callback/1",
	})
	provider := createOIDCLoginProvider(t, db, issuer)
	user := createOIDCLoginUser(t, db, "collector", "collector@example.com")
	if err := db.Create(&models.ExternalIdentity{UserID: user.ID, ProviderID: provider.ID, Issuer: issuer, Subject: "subject-123", Email: user.Email, EmailVerified: true}).Error; err != nil {
		t.Fatalf("failed to create external identity: %v", err)
	}

	start, err := svc.StartLogin(context.Background(), provider.ID, "/", "/auth/oidc/callback/1", "http://app.example")
	if err != nil {
		t.Fatalf("start login failed: %v", err)
	}
	authURL, _ := url.Parse(start.AuthorizationURL)
	currentOIDCTestNonce = authURL.Query().Get("nonce")
	result, err := svc.CompleteLoginCallback(context.Background(), provider.ID, "valid-code", authURL.Query().Get("state"), "http://internal-api:8080", OIDCAuditContext{})
	if err != nil {
		t.Fatalf("callback failed: %v", err)
	}
	if result.Token == "" || result.RefreshToken == "" || result.User.ID != user.ID {
		t.Fatalf("expected app auth tokens for linked user, got %+v", result)
	}

	var identity models.ExternalIdentity
	if err := db.First(&identity, "user_id = ?", user.ID).Error; err != nil {
		t.Fatalf("failed to reload identity: %v", err)
	}
	if identity.LastLoginAt == nil {
		t.Fatal("expected last_login_at to be updated")
	}
}

func TestOIDCServiceStartLoginUsesFrontendCallbackPath(t *testing.T) {
	db, svc := setupOIDCLoginServiceTest(t)
	issuer := startMockOIDCProvider(t, "subject-123", "collector@example.com", true)
	provider := createOIDCLoginProvider(t, db, issuer)

	start, err := svc.StartLogin(context.Background(), provider.ID, "/", "/auth/oidc/callback/1", "http://app.example")
	if err != nil {
		t.Fatalf("start login failed: %v", err)
	}
	authURL, err := url.Parse(start.AuthorizationURL)
	if err != nil {
		t.Fatalf("failed to parse authorization URL %q: %v", start.AuthorizationURL, err)
	}
	if redirectURI := authURL.Query().Get("redirect_uri"); redirectURI != "http://app.example/auth/oidc/callback/1" {
		t.Fatalf("expected frontend login callback redirect URI, got %q", redirectURI)
	}
}

func TestOIDCServiceStartLoginUsesEntraAuthorizationEndpoint(t *testing.T) {
	db, svc := setupOIDCLoginServiceTest(t)
	issuer := startMockEntraDiscoveryProvider(t)
	provider := createOIDCLoginProvider(t, db, issuer)
	provider.ProviderType = models.OIDCProviderTypeEntra
	if err := db.Save(&provider).Error; err != nil {
		t.Fatalf("failed to save Entra provider: %v", err)
	}

	start, err := svc.StartLogin(context.Background(), provider.ID, "/", "", "http://app.example")
	if err != nil {
		t.Fatalf("start login failed: %v", err)
	}
	authURL, err := url.Parse(start.AuthorizationURL)
	if err != nil {
		t.Fatalf("failed to parse authorization URL %q: %v", start.AuthorizationURL, err)
	}
	if !strings.Contains(authURL.Path, "/oauth2/v2.0/authorize") {
		t.Fatalf("expected Entra authorization endpoint path, got %q from %q", authURL.Path, start.AuthorizationURL)
	}
	if strings.Contains(authURL.Path, "/oauth2/v2.0/token") {
		t.Fatalf("expected browser URL not to use Entra token endpoint path, got %q", authURL.Path)
	}
	runtime, err := svc.BuildRuntimeConfig(context.Background(), provider)
	if err != nil {
		t.Fatalf("failed to build runtime config: %v", err)
	}
	if runtime.OAuth2Config.Endpoint.AuthStyle != oauth2.AuthStyleInParams {
		t.Fatalf("expected Entra token exchange to send client secret in params, got auth style %v", runtime.OAuth2Config.Endpoint.AuthStyle)
	}
}

func TestOIDCServiceLoginCallbackReportsInvalidEntraClientSecret(t *testing.T) {
	db, svc := setupOIDCLoginServiceTest(t)
	issuer := startMockOIDCProviderWithOptions(t, oidcMockProviderOptions{
		Subject:                    "subject-123",
		Email:                      "collector@example.com",
		EmailVerified:              true,
		TokenErrorCode:             "invalid_client",
		TokenErrorDescription:      "AADSTS7000215: Invalid client secret provided.",
		TokenErrorHTTPResponseCode: http.StatusUnauthorized,
	})
	provider := createOIDCLoginProvider(t, db, issuer)

	start, err := svc.StartLogin(context.Background(), provider.ID, "/", "", "http://app.example")
	if err != nil {
		t.Fatalf("start login failed: %v", err)
	}
	authURL, _ := url.Parse(start.AuthorizationURL)
	currentOIDCTestNonce = authURL.Query().Get("nonce")

	_, err = svc.CompleteLoginCallback(context.Background(), provider.ID, "valid-code", authURL.Query().Get("state"), "http://app.example", OIDCAuditContext{})
	if !errors.Is(err, ErrOIDCCodeExchangeFailed) {
		t.Fatalf("expected code exchange failure, got %v", err)
	}
	detail := OIDCClientErrorDetail(err)
	if !strings.Contains(detail, "client secret Value") {
		t.Fatalf("expected actionable client secret detail, got %q", detail)
	}
	if reason := oidcFailureReason(err); !strings.Contains(reason, "client secret Value") {
		t.Fatalf("expected security event reason to include safe detail, got %q", reason)
	}
}

func TestOIDCServiceLoginCallbackBlocksMatchingEmailWithoutLink(t *testing.T) {
	db, svc := setupOIDCLoginServiceTest(t)
	issuer := startMockOIDCProvider(t, "new-subject", "collector@example.com", true)
	provider := createOIDCLoginProvider(t, db, issuer)
	createOIDCLoginUser(t, db, "collector", "collector@example.com")

	start, err := svc.StartLogin(context.Background(), provider.ID, "/", "", "http://app.example")
	if err != nil {
		t.Fatalf("start login failed: %v", err)
	}
	authURL, _ := url.Parse(start.AuthorizationURL)
	currentOIDCTestNonce = authURL.Query().Get("nonce")
	if _, err := svc.CompleteLoginCallback(context.Background(), provider.ID, "valid-code", authURL.Query().Get("state"), "http://app.example", OIDCAuditContext{}); err != ErrOIDCAccountConflict {
		t.Fatalf("expected account conflict, got %v", err)
	}
}

func TestOIDCServiceLoginCallbackRejectsUnverifiedEmailWhenRequired(t *testing.T) {
	db, svc := setupOIDCLoginServiceTest(t)
	issuer := startMockOIDCProvider(t, "subject-123", "collector@example.com", false)
	provider := createOIDCLoginProvider(t, db, issuer)
	user := createOIDCLoginUser(t, db, "collector", "collector@example.com")
	if err := db.Create(&models.ExternalIdentity{UserID: user.ID, ProviderID: provider.ID, Issuer: issuer, Subject: "subject-123", Email: user.Email, EmailVerified: false}).Error; err != nil {
		t.Fatalf("failed to create external identity: %v", err)
	}

	start, err := svc.StartLogin(context.Background(), provider.ID, "/", "", "http://app.example")
	if err != nil {
		t.Fatalf("start login failed: %v", err)
	}
	authURL, _ := url.Parse(start.AuthorizationURL)
	currentOIDCTestNonce = authURL.Query().Get("nonce")
	if _, err := svc.CompleteLoginCallback(context.Background(), provider.ID, "valid-code", authURL.Query().Get("state"), "http://app.example", OIDCAuditContext{}); err != ErrOIDCValidationFailed {
		t.Fatalf("expected validation failure, got %v", err)
	}
}

func TestOIDCServiceLoginCallbackRejectsInvalidStateAndReplay(t *testing.T) {
	db, svc := setupOIDCLoginServiceTest(t)
	issuer := startMockOIDCProvider(t, "subject-123", "collector@example.com", true)
	provider := createOIDCLoginProvider(t, db, issuer)
	user := createOIDCLoginUser(t, db, "collector", "collector@example.com")
	if err := db.Create(&models.ExternalIdentity{UserID: user.ID, ProviderID: provider.ID, Issuer: issuer, Subject: "subject-123", Email: user.Email, EmailVerified: true}).Error; err != nil {
		t.Fatalf("failed to create external identity: %v", err)
	}

	start, err := svc.StartLogin(context.Background(), provider.ID, "/", "", "http://app.example")
	if err != nil {
		t.Fatalf("start login failed: %v", err)
	}
	authURL, _ := url.Parse(start.AuthorizationURL)
	currentOIDCTestNonce = authURL.Query().Get("nonce")

	if _, err := svc.CompleteLoginCallback(context.Background(), provider.ID, "valid-code", "wrong-state", "http://app.example", OIDCAuditContext{}); !errors.Is(err, ErrOIDCInvalidState) {
		t.Fatalf("expected invalid state error, got %v", err)
	}

	if _, err := svc.CompleteLoginCallback(context.Background(), provider.ID, "valid-code", authURL.Query().Get("state"), "http://app.example", OIDCAuditContext{}); err != nil {
		t.Fatalf("expected first callback to succeed before replay assertion, got %v", err)
	}
	if _, err := svc.CompleteLoginCallback(context.Background(), provider.ID, "valid-code", authURL.Query().Get("state"), "http://app.example", OIDCAuditContext{}); !errors.Is(err, ErrOIDCInvalidState) {
		t.Fatalf("expected replayed state error, got %v", err)
	}
}

func TestOIDCServiceLoginCallbackRejectsInvalidTokenClaims(t *testing.T) {
	tests := []struct {
		name      string
		options   oidcMockProviderOptions
		nonceFunc func(string) string
	}{
		{
			name:      "invalid nonce",
			options:   oidcMockProviderOptions{Subject: "subject-123", Email: "collector@example.com", EmailVerified: true},
			nonceFunc: func(string) string { return "wrong-nonce" },
		},
		{
			name:    "invalid issuer",
			options: oidcMockProviderOptions{Subject: "subject-123", Email: "collector@example.com", EmailVerified: true, Issuer: "https://wrong-issuer.example"},
		},
		{
			name:    "invalid audience",
			options: oidcMockProviderOptions{Subject: "subject-123", Email: "collector@example.com", EmailVerified: true, Audience: "wrong-client-id"},
		},
		{
			name:    "expired token",
			options: oidcMockProviderOptions{Subject: "subject-123", Email: "collector@example.com", EmailVerified: true, ExpiresAt: time.Now().Add(-time.Hour)},
		},
		{
			name:    "bad signature",
			options: oidcMockProviderOptions{Subject: "subject-123", Email: "collector@example.com", EmailVerified: true, SignWithUntrustedKey: true},
		},
		{
			name:    "missing subject",
			options: oidcMockProviderOptions{Email: "collector@example.com", EmailVerified: true, OmitSubject: true},
		},
		{
			name:    "unverified email",
			options: oidcMockProviderOptions{Subject: "subject-123", Email: "collector@example.com", EmailVerified: false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, svc := setupOIDCLoginServiceTest(t)
			issuer := startMockOIDCProviderWithOptions(t, tt.options)
			provider := createOIDCLoginProvider(t, db, issuer)

			start, err := svc.StartLogin(context.Background(), provider.ID, "/", "", "http://app.example")
			if err != nil {
				t.Fatalf("start login failed: %v", err)
			}
			authURL, _ := url.Parse(start.AuthorizationURL)
			nonce := authURL.Query().Get("nonce")
			if tt.nonceFunc != nil {
				nonce = tt.nonceFunc(nonce)
			}
			currentOIDCTestNonce = nonce

			_, err = svc.CompleteLoginCallback(context.Background(), provider.ID, "valid-code", authURL.Query().Get("state"), "http://app.example", OIDCAuditContext{})
			if !errors.Is(err, ErrOIDCValidationFailed) {
				t.Fatalf("expected token validation failure, got %v", err)
			}
		})
	}
}

func TestOIDCServiceUnlinkIdentityOwnershipAndLocalPassword(t *testing.T) {
	t.Run("success removes owned identity when local password remains", func(t *testing.T) {
		db, svc := setupOIDCLoginServiceTest(t)
		provider := createOIDCLoginProvider(t, db, "http://localhost")
		user := createOIDCLoginUser(t, db, "collector", "collector@example.com")
		identity := createOIDCLoginIdentity(t, db, user.ID, provider.ID, "subject-123")

		if err := svc.UnlinkIdentity(identity.ID, user.ID, OIDCAuditContext{}); err != nil {
			t.Fatalf("unlink failed: %v", err)
		}
		assertOIDCIdentityCount(t, db, user.ID, 0)
	})

	t.Run("not-owned identity is not found for current user", func(t *testing.T) {
		db, svc := setupOIDCLoginServiceTest(t)
		provider := createOIDCLoginProvider(t, db, "http://localhost")
		owner := createOIDCLoginUser(t, db, "owner", "owner@example.com")
		other := createOIDCLoginUser(t, db, "other", "other@example.com")
		identity := createOIDCLoginIdentity(t, db, owner.ID, provider.ID, "subject-123")

		if err := svc.UnlinkIdentity(identity.ID, other.ID, OIDCAuditContext{}); !errors.Is(err, ErrOIDCIdentityNotFound) {
			t.Fatalf("expected not-linked error for not-owned identity, got %v", err)
		}
		assertOIDCIdentityCount(t, db, owner.ID, 1)
	})
}

func TestOIDCServiceLinkCallbackCreatesIdentityForAuthenticatedUser(t *testing.T) {
	db, svc := setupOIDCLoginServiceTest(t)
	issuer := startMockOIDCProvider(t, "link-subject-123456", "collector@example.com", true)
	provider := createOIDCLoginProvider(t, db, issuer)
	user := createOIDCLoginUser(t, db, "collector", "collector@example.com")

	start, err := svc.StartLink(context.Background(), provider.ID, user.ID, "/settings", "", "http://app.example")
	if err != nil {
		t.Fatalf("start link failed: %v", err)
	}
	authURL, _ := url.Parse(start.AuthorizationURL)
	if redirectURI := authURL.Query().Get("redirect_uri"); redirectURI != "http://app.example/api/auth/oidc/1/link/callback" {
		t.Fatalf("expected link callback redirect URI, got %q", redirectURI)
	}
	currentOIDCTestNonce = authURL.Query().Get("nonce")

	result, err := svc.CompleteLinkCallback(context.Background(), provider.ID, "valid-code", authURL.Query().Get("state"), "http://app.example", OIDCAuditContext{})
	if err != nil {
		t.Fatalf("link callback failed: %v", err)
	}

	if result.Identity.ProviderID != provider.ID || result.Identity.Email != user.Email || result.Identity.SubjectPreview != "link-sub..." {
		t.Fatalf("unexpected linked identity response: %+v", result.Identity)
	}
	var count int64
	if err := db.Model(&models.ExternalIdentity{}).Where("user_id = ? AND subject = ?", user.ID, "link-subject-123456").Count(&count).Error; err != nil {
		t.Fatalf("failed to count identities: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected identity to be created once, got %d", count)
	}
}

func TestOIDCServiceLinkStartAllowsFrontendCallbackPath(t *testing.T) {
	db, svc := setupOIDCLoginServiceTest(t)
	issuer := startMockOIDCProvider(t, "frontend-link-subject", "collector@example.com", true)
	provider := createOIDCLoginProvider(t, db, issuer)
	user := createOIDCLoginUser(t, db, "collector", "collector@example.com")

	start, err := svc.StartLink(context.Background(), provider.ID, user.ID, "/settings?tab=account", "/settings/oidc/link/callback/1", "http://app.example")
	if err != nil {
		t.Fatalf("start link failed: %v", err)
	}
	authURL, _ := url.Parse(start.AuthorizationURL)
	if redirectURI := authURL.Query().Get("redirect_uri"); redirectURI != "http://app.example/settings/oidc/link/callback/1" {
		t.Fatalf("expected frontend link callback redirect URI, got %q", redirectURI)
	}
}

func TestOIDCServiceLinkCallbackReusesPersistedRedirectURI(t *testing.T) {
	db, svc := setupOIDCLoginServiceTest(t)
	issuer := startMockOIDCProviderWithOptions(t, oidcMockProviderOptions{
		Subject:             "link-proxy-subject",
		Email:               "collector@example.com",
		EmailVerified:       true,
		ExpectedRedirectURI: "http://app.example/settings/oidc/link/callback/1",
	})
	provider := createOIDCLoginProvider(t, db, issuer)
	user := createOIDCLoginUser(t, db, "collector", "collector@example.com")

	start, err := svc.StartLink(context.Background(), provider.ID, user.ID, "/settings?tab=account", "/settings/oidc/link/callback/1", "http://app.example")
	if err != nil {
		t.Fatalf("start link failed: %v", err)
	}
	authURL, _ := url.Parse(start.AuthorizationURL)
	if redirectURI := authURL.Query().Get("redirect_uri"); redirectURI != "http://app.example/settings/oidc/link/callback/1" {
		t.Fatalf("expected link callback redirect URI, got %q", redirectURI)
	}
	currentOIDCTestNonce = authURL.Query().Get("nonce")

	result, err := svc.CompleteLinkCallback(context.Background(), provider.ID, "valid-code", authURL.Query().Get("state"), "http://internal-api:8080", OIDCAuditContext{})
	if err != nil {
		t.Fatalf("link callback with different origin failed: %v", err)
	}
	if result.Identity.ProviderID != provider.ID || result.Identity.Email != user.Email {
		t.Fatalf("unexpected linked identity response: %+v", result.Identity)
	}
	var count int64
	if err := db.Model(&models.ExternalIdentity{}).Where("user_id = ? AND subject = ?", user.ID, "link-proxy-subject").Count(&count).Error; err != nil {
		t.Fatalf("failed to count identities: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected identity to be created once, got %d", count)
	}
}

func TestOIDCServiceLinkCallbackBlocksIdentityLinkedToAnotherUser(t *testing.T) {
	db, svc := setupOIDCLoginServiceTest(t)
	issuer := startMockOIDCProvider(t, "shared-subject", "collector@example.com", true)
	provider := createOIDCLoginProvider(t, db, issuer)
	owner := createOIDCLoginUser(t, db, "owner", "owner@example.com")
	user := createOIDCLoginUser(t, db, "collector", "collector@example.com")
	if err := db.Create(&models.ExternalIdentity{UserID: owner.ID, ProviderID: provider.ID, Issuer: issuer, Subject: "shared-subject", Email: owner.Email, EmailVerified: true}).Error; err != nil {
		t.Fatalf("failed to create existing identity: %v", err)
	}

	start, err := svc.StartLink(context.Background(), provider.ID, user.ID, "/settings", "", "http://app.example")
	if err != nil {
		t.Fatalf("start link failed: %v", err)
	}
	authURL, _ := url.Parse(start.AuthorizationURL)
	currentOIDCTestNonce = authURL.Query().Get("nonce")

	_, err = svc.CompleteLinkCallback(context.Background(), provider.ID, "valid-code", authURL.Query().Get("state"), "http://app.example", OIDCAuditContext{})
	if !errors.Is(err, ErrOIDCIdentityAlreadyLinked) {
		t.Fatalf("expected already linked error, got %v", err)
	}
}

func TestOIDCServiceLinkCallbackBlocksVerifiedEmailForAnotherLocalUser(t *testing.T) {
	db, svc := setupOIDCLoginServiceTest(t)
	issuer := startMockOIDCProvider(t, "new-subject", "owner@example.com", true)
	provider := createOIDCLoginProvider(t, db, issuer)
	createOIDCLoginUser(t, db, "owner", "owner@example.com")
	user := createOIDCLoginUser(t, db, "collector", "collector@example.com")

	start, err := svc.StartLink(context.Background(), provider.ID, user.ID, "/settings", "", "http://app.example")
	if err != nil {
		t.Fatalf("start link failed: %v", err)
	}
	authURL, _ := url.Parse(start.AuthorizationURL)
	currentOIDCTestNonce = authURL.Query().Get("nonce")

	_, err = svc.CompleteLinkCallback(context.Background(), provider.ID, "valid-code", authURL.Query().Get("state"), "http://app.example", OIDCAuditContext{})
	if !errors.Is(err, ErrOIDCAccountConflict) {
		t.Fatalf("expected account conflict, got %v", err)
	}
}

func TestOIDCServiceUnlinkIdentityGuardsLastUsableSignInMethod(t *testing.T) {
	db, svc := setupOIDCLoginServiceTest(t)
	issuer := startMockOIDCProvider(t, "unlink-subject", "oidc-only@example.com", true)
	provider := createOIDCLoginProvider(t, db, issuer)
	user := models.User{Username: "oidc-only", Email: "oidc-only@example.com", PasswordHash: "", Role: models.RoleUser}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}
	identity := models.ExternalIdentity{UserID: user.ID, ProviderID: provider.ID, Issuer: issuer, Subject: "unlink-subject", Email: user.Email, EmailVerified: true}
	if err := db.Create(&identity).Error; err != nil {
		t.Fatalf("failed to create identity: %v", err)
	}

	if err := svc.UnlinkIdentity(identity.ID, user.ID, OIDCAuditContext{}); !errors.Is(err, ErrOIDCNoUsableSignInMethod) {
		t.Fatalf("expected no usable sign-in guard, got %v", err)
	}

	if err := db.Create(&models.WebAuthnCredential{UserID: user.ID, CredentialID: "cred-1", PublicKey: []byte("key")}).Error; err != nil {
		t.Fatalf("failed to create webauthn credential: %v", err)
	}
	if err := svc.UnlinkIdentity(identity.ID, user.ID, OIDCAuditContext{}); err != nil {
		t.Fatalf("expected unlink to succeed with passkey credential, got %v", err)
	}
}

func TestOIDCServiceLinkCallbackFailsWhenRedirectURINotStored(t *testing.T) {
	db, svc := setupOIDCLoginServiceTest(t)
	issuer := startMockOIDCProvider(t, "link-subject-no-uri", "collector@example.com", true)
	provider := createOIDCLoginProvider(t, db, issuer)
	user := createOIDCLoginUser(t, db, "collector", "collector@example.com")

	start, err := svc.StartLink(context.Background(), provider.ID, user.ID, "/settings", "/settings/oidc/link/callback/1", "http://app.example")
	if err != nil {
		t.Fatalf("start link failed: %v", err)
	}
	authURL, _ := url.Parse(start.AuthorizationURL)
	state := authURL.Query().Get("state")

	var authState models.OIDCAuthState
	if err := db.Where("state_hash = ?", hashOIDCSecret(state)).First(&authState).Error; err != nil {
		t.Fatalf("failed to find auth state: %v", err)
	}
	authState.RedirectURI = ""
	if err := db.Save(&authState).Error; err != nil {
		t.Fatalf("failed to clear RedirectURI: %v", err)
	}

	currentOIDCTestNonce = authURL.Query().Get("nonce")
	_, err = svc.CompleteLinkCallback(context.Background(), provider.ID, "valid-code", state, "http://app.example", OIDCAuditContext{})
	if !errors.Is(err, ErrOIDCInvalidState) {
		t.Fatalf("expected invalid state error for missing redirect URI, got %v", err)
	}
	if err == nil || !strings.Contains(err.Error(), "stored redirect URI missing") {
		t.Fatalf("expected error message about missing stored redirect URI, got %v", err)
	}
}

func startMockOIDCProvider(t *testing.T, subject, email string, emailVerified bool) string {
	t.Helper()
	return startMockOIDCProviderWithOptions(t, oidcMockProviderOptions{
		Subject:       subject,
		Email:         email,
		EmailVerified: emailVerified,
	})
}

type oidcMockProviderOptions struct {
	Subject                    string
	Email                      string
	EmailVerified              bool
	Issuer                     string
	Audience                   string
	ExpiresAt                  time.Time
	OmitSubject                bool
	SignWithUntrustedKey       bool
	ExpectedRedirectURI        string
	TokenErrorCode             string
	TokenErrorDescription      string
	TokenErrorHTTPResponseCode int
}

func startMockOIDCProviderWithOptions(t *testing.T, options oidcMockProviderOptions) string {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/.well-known/openid-configuration":
			writeJSON(t, w, map[string]string{
				"issuer":                 server.URL,
				"authorization_endpoint": server.URL + "/authorize",
				"token_endpoint":         server.URL + "/token",
				"jwks_uri":               server.URL + "/jwks",
			})
		case "/jwks":
			writeJSON(t, w, map[string]any{"keys": []map[string]string{rsaJWK(&key.PublicKey)}})
		case "/token":
			if err := r.ParseForm(); err != nil {
				http.Error(w, "bad form", http.StatusBadRequest)
				return
			}
			if options.ExpectedRedirectURI != "" && r.Form.Get("redirect_uri") != options.ExpectedRedirectURI {
				http.Error(w, "redirect_uri mismatch", http.StatusBadRequest)
				return
			}
			if options.TokenErrorCode != "" {
				status := options.TokenErrorHTTPResponseCode
				if status == 0 {
					status = http.StatusBadRequest
				}
				w.WriteHeader(status)
				writeJSON(t, w, map[string]string{
					"error":             options.TokenErrorCode,
					"error_description": options.TokenErrorDescription,
				})
				return
			}
			nonce := r.Form.Get("nonce")
			if nonce == "" {
				nonce = currentOIDCTestNonce
			}
			issuer := options.Issuer
			if issuer == "" {
				issuer = server.URL
			}
			audience := options.Audience
			if audience == "" {
				audience = "client-id"
			}
			expiresAt := options.ExpiresAt
			if expiresAt.IsZero() {
				expiresAt = time.Now().Add(time.Hour)
			}
			claims := jwt.MapClaims{
				"iss":            issuer,
				"aud":            audience,
				"email":          options.Email,
				"email_verified": options.EmailVerified,
				"nonce":          nonce,
				"iat":            time.Now().Unix(),
				"exp":            expiresAt.Unix(),
			}
			if !options.OmitSubject {
				claims["sub"] = options.Subject
			}
			token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
			token.Header["kid"] = "test-key"
			signingKey := key
			if options.SignWithUntrustedKey {
				signingKey, err = rsa.GenerateKey(rand.Reader, 2048)
				if err != nil {
					t.Fatalf("failed to generate untrusted signing key: %v", err)
				}
			}
			signed, err := token.SignedString(signingKey)
			if err != nil {
				t.Fatalf("failed to sign id token: %v", err)
			}
			writeJSON(t, w, map[string]any{"access_token": "provider-access-token", "token_type": "Bearer", "id_token": signed})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)
	return server.URL
}

func startMockEntraDiscoveryProvider(t *testing.T) string {
	t.Helper()
	var server *httptest.Server
	tenantPath := "/tenant-id/v2.0"
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, ".well-known/openid-configuration") {
			http.NotFound(w, r)
			return
		}
		writeJSON(t, w, map[string]string{
			"issuer":                 server.URL + tenantPath,
			"authorization_endpoint": server.URL + "/tenant-id/oauth2/v2.0/authorize",
			"token_endpoint":         server.URL + "/tenant-id/oauth2/v2.0/token",
			"jwks_uri":               server.URL + "/tenant-id/discovery/v2.0/keys",
		})
	}))
	t.Cleanup(server.Close)
	return server.URL + tenantPath
}

var currentOIDCTestNonce string

func createOIDCLoginProvider(t *testing.T, db *gorm.DB, issuer string) models.OIDCProvider {
	t.Helper()
	provider := models.OIDCProvider{
		Name:                 "mock-provider",
		DisplayName:          "Mock Provider",
		ProviderType:         models.OIDCProviderTypeGeneric,
		Enabled:              true,
		IssuerURL:            issuer,
		ClientID:             "client-id",
		ClientSecret:         "client-secret",
		Scopes:               models.StringList{"openid", "profile", "email"},
		CallbackPath:         "/api/auth/oidc/1/callback",
		RequireVerifiedEmail: true,
	}
	if err := db.Create(&provider).Error; err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}
	return provider
}

func createOIDCLoginUser(t *testing.T, db *gorm.DB, username, email string) models.User {
	t.Helper()
	user := models.User{Username: username, Email: email, PasswordHash: "local-password-hash", Role: models.RoleUser}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}
	return user
}

func createOIDCLoginIdentity(t *testing.T, db *gorm.DB, userID, providerID uint, subject string) models.ExternalIdentity {
	t.Helper()
	identity := models.ExternalIdentity{
		UserID:        userID,
		ProviderID:    providerID,
		Issuer:        "http://localhost",
		Subject:       subject,
		Email:         "collector@example.com",
		EmailVerified: true,
	}
	if err := db.Create(&identity).Error; err != nil {
		t.Fatalf("failed to create external identity: %v", err)
	}
	return identity
}

func assertOIDCIdentityCount(t *testing.T, db *gorm.DB, userID uint, expected int64) {
	t.Helper()
	var count int64
	if err := db.Model(&models.ExternalIdentity{}).Where("user_id = ?", userID).Count(&count).Error; err != nil {
		t.Fatalf("failed to count identities: %v", err)
	}
	if count != expected {
		t.Fatalf("expected %d identities for user %d, got %d", expected, userID, count)
	}
}

func writeJSON(t *testing.T, w http.ResponseWriter, value any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		t.Fatalf("failed to write json: %v", err)
	}
}

func rsaJWK(pub *rsa.PublicKey) map[string]string {
	return map[string]string{
		"kty": "RSA",
		"use": "sig",
		"kid": "test-key",
		"alg": "RS256",
		"n":   base64.RawURLEncoding.EncodeToString(pub.N.Bytes()),
		"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(pub.E)).Bytes()),
	}
}
