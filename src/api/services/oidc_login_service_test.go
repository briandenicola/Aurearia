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
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/glebarez/sqlite"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

func setupOIDCLoginServiceTest(t *testing.T) (*gorm.DB, *OIDCService) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.RefreshToken{}, &models.SecurityEvent{}, &models.OIDCProvider{}, &models.ExternalIdentity{}, &models.OIDCAuthState{}); err != nil {
		t.Fatalf("failed to migrate test db: %v", err)
	}
	authSvc := NewAuthService(repository.NewAuthRepository(db), "test-jwt-secret")
	return db, NewOIDCService(repository.NewOIDCRepository(db), nil).WithAuth(authSvc).WithSecurity(NewSecurityService(repository.NewSecurityRepository(db)))
}

func TestOIDCServiceLoginCallbackIssuesTokensForLinkedIdentity(t *testing.T) {
	db, svc := setupOIDCLoginServiceTest(t)
	issuer := startMockOIDCProvider(t, "subject-123", "collector@example.com", true)
	provider := createOIDCLoginProvider(t, db, issuer)
	user := createOIDCLoginUser(t, db, "collector", "collector@example.com")
	if err := db.Create(&models.ExternalIdentity{UserID: user.ID, ProviderID: provider.ID, Issuer: issuer, Subject: "subject-123", Email: user.Email, EmailVerified: true}).Error; err != nil {
		t.Fatalf("failed to create external identity: %v", err)
	}

	start, err := svc.StartLogin(context.Background(), provider.ID, "/", "http://app.example")
	if err != nil {
		t.Fatalf("start login failed: %v", err)
	}
	authURL, _ := url.Parse(start.AuthorizationURL)
	currentOIDCTestNonce = authURL.Query().Get("nonce")
	result, err := svc.CompleteLoginCallback(context.Background(), provider.ID, "valid-code", authURL.Query().Get("state"), "http://app.example", OIDCAuditContext{})
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

func TestOIDCServiceLoginCallbackBlocksMatchingEmailWithoutLink(t *testing.T) {
	db, svc := setupOIDCLoginServiceTest(t)
	issuer := startMockOIDCProvider(t, "new-subject", "collector@example.com", true)
	provider := createOIDCLoginProvider(t, db, issuer)
	createOIDCLoginUser(t, db, "collector", "collector@example.com")

	start, err := svc.StartLogin(context.Background(), provider.ID, "/", "http://app.example")
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

	start, err := svc.StartLogin(context.Background(), provider.ID, "/", "http://app.example")
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

	start, err := svc.StartLogin(context.Background(), provider.ID, "/", "http://app.example")
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

			start, err := svc.StartLogin(context.Background(), provider.ID, "/", "http://app.example")
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

func startMockOIDCProvider(t *testing.T, subject, email string, emailVerified bool) string {
	t.Helper()
	return startMockOIDCProviderWithOptions(t, oidcMockProviderOptions{
		Subject:       subject,
		Email:         email,
		EmailVerified: emailVerified,
	})
}

type oidcMockProviderOptions struct {
	Subject              string
	Email                string
	EmailVerified        bool
	Issuer               string
	Audience             string
	ExpiresAt            time.Time
	OmitSubject          bool
	SignWithUntrustedKey bool
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
