package handlers

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

func setupOIDCLoginHandlerTest(t *testing.T) (*gorm.DB, *gin.Engine) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.RefreshToken{}, &models.SecurityEvent{}, &models.IPRule{}, &models.OIDCProvider{}, &models.ExternalIdentity{}, &models.OIDCAuthState{}); err != nil {
		t.Fatalf("failed to migrate test db: %v", err)
	}

	authSvc := services.NewAuthService(repository.NewAuthRepository(db), "oidc-handler-test-secret")
	securitySvc := services.NewSecurityService(repository.NewSecurityRepository(db))
	oidcSvc := services.NewOIDCService(repository.NewOIDCRepository(db), nil).WithAuth(authSvc).WithSecurity(securitySvc)
	handler := NewOIDCHandler(oidcSvc)

	router := gin.New()
	router.POST("/api/auth/oidc/:providerId/start", handler.StartLogin)
	router.GET("/api/auth/oidc/:providerId/callback", handler.Callback)

	return db, router
}

func TestOIDCHandlerCallbackSuccessIssuesAuthResponseForLinkedIdentity(t *testing.T) {
	db, router := setupOIDCLoginHandlerTest(t)
	issuer := startMockOIDCHandlerProvider(t, "subject-123", "collector@example.com", true)
	provider := createOIDCHandlerProvider(t, db, issuer)
	user := createOIDCHandlerUser(t, db, "collector", "collector@example.com")
	if err := db.Create(&models.ExternalIdentity{UserID: user.ID, ProviderID: provider.ID, Issuer: issuer, Subject: "subject-123", Email: user.Email, EmailVerified: true}).Error; err != nil {
		t.Fatalf("failed to create external identity: %v", err)
	}

	startBody, _ := json.Marshal(map[string]string{"redirectPath": "/"})
	startReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/auth/oidc/%d/start", provider.ID), bytes.NewReader(startBody))
	startReq.Header.Set("Content-Type", "application/json")
	startReq.Host = "app.example"
	startRec := httptest.NewRecorder()
	router.ServeHTTP(startRec, startReq)
	if startRec.Code != http.StatusOK {
		t.Fatalf("expected start status 200, got %d: %s", startRec.Code, startRec.Body.String())
	}

	var startResp struct {
		AuthorizationURL string `json:"authorizationUrl"`
	}
	if err := json.Unmarshal(startRec.Body.Bytes(), &startResp); err != nil {
		t.Fatalf("failed to decode start response: %v", err)
	}
	authURL, err := url.Parse(startResp.AuthorizationURL)
	if err != nil {
		t.Fatalf("failed to parse authorization URL: %v", err)
	}
	state := authURL.Query().Get("state")
	currentOIDCHandlerTestNonce = authURL.Query().Get("nonce")
	if state == "" || currentOIDCHandlerTestNonce == "" {
		t.Fatalf("expected state and nonce in authorization URL, got %q", startResp.AuthorizationURL)
	}

	callbackPath := fmt.Sprintf("/api/auth/oidc/%d/callback?code=valid-code&state=%s", provider.ID, url.QueryEscape(state))
	callbackReq := httptest.NewRequest(http.MethodGet, callbackPath, nil)
	callbackReq.Host = "app.example"
	callbackRec := httptest.NewRecorder()
	router.ServeHTTP(callbackRec, callbackReq)
	if callbackRec.Code != http.StatusOK {
		t.Fatalf("expected callback status 200, got %d: %s", callbackRec.Code, callbackRec.Body.String())
	}
	if callbackRec.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("expected no-store auth response, got %q", callbackRec.Header().Get("Cache-Control"))
	}

	var authResp struct {
		Token        string `json:"token"`
		RefreshToken string `json:"refreshToken"`
		User         struct {
			ID       uint   `json:"id"`
			Username string `json:"username"`
			Email    string `json:"email"`
			Role     string `json:"role"`
		} `json:"user"`
	}
	if err := json.Unmarshal(callbackRec.Body.Bytes(), &authResp); err != nil {
		t.Fatalf("failed to decode callback response: %v", err)
	}
	if authResp.Token == "" || authResp.RefreshToken == "" {
		t.Fatalf("expected app JWT and refresh token in callback response, got %s", callbackRec.Body.String())
	}
	if authResp.User.ID != user.ID || authResp.User.Username != user.Username || authResp.User.Email != user.Email {
		t.Fatalf("expected linked user in callback response, got %+v", authResp.User)
	}
	for _, leaked := range []string{"provider-access-token", "valid-code", currentOIDCHandlerTestNonce} {
		if strings.Contains(callbackRec.Body.String(), leaked) {
			t.Fatalf("callback response leaked provider-only secret %q: %s", leaked, callbackRec.Body.String())
		}
	}
}

func startMockOIDCHandlerProvider(t *testing.T, subject, email string, emailVerified bool) string {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/.well-known/openid-configuration":
			writeOIDCHandlerJSON(t, w, map[string]string{
				"issuer":                 server.URL,
				"authorization_endpoint": server.URL + "/authorize",
				"token_endpoint":         server.URL + "/token",
				"jwks_uri":               server.URL + "/jwks",
			})
		case "/jwks":
			writeOIDCHandlerJSON(t, w, map[string]any{"keys": []map[string]string{oidcHandlerRSAJWK(&key.PublicKey)}})
		case "/token":
			expiresAt := time.Now().Add(time.Hour)
			claims := jwt.MapClaims{
				"iss":            server.URL,
				"aud":            "client-id",
				"sub":            subject,
				"email":          email,
				"email_verified": emailVerified,
				"nonce":          currentOIDCHandlerTestNonce,
				"iat":            time.Now().Unix(),
				"exp":            expiresAt.Unix(),
			}
			token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
			token.Header["kid"] = "test-key"
			signed, err := token.SignedString(key)
			if err != nil {
				t.Fatalf("failed to sign id token: %v", err)
			}
			writeOIDCHandlerJSON(t, w, map[string]any{"access_token": "provider-access-token", "token_type": "Bearer", "id_token": signed})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)
	return server.URL
}

var currentOIDCHandlerTestNonce string

func createOIDCHandlerProvider(t *testing.T, db *gorm.DB, issuer string) models.OIDCProvider {
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
		RequireVerifiedEmail: true,
	}
	if err := db.Create(&provider).Error; err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}
	provider.CallbackPath = fmt.Sprintf("/api/auth/oidc/%d/callback", provider.ID)
	if err := db.Save(&provider).Error; err != nil {
		t.Fatalf("failed to update provider callback path: %v", err)
	}
	return provider
}

func createOIDCHandlerUser(t *testing.T, db *gorm.DB, username, email string) models.User {
	t.Helper()
	user := models.User{Username: username, Email: email, PasswordHash: "local-password-hash", Role: models.RoleUser}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}
	return user
}

func writeOIDCHandlerJSON(t *testing.T, w http.ResponseWriter, value any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		t.Fatalf("failed to write json: %v", err)
	}
}

func oidcHandlerRSAJWK(pub *rsa.PublicKey) map[string]string {
	return map[string]string{
		"kty": "RSA",
		"use": "sig",
		"kid": "test-key",
		"alg": "RS256",
		"n":   base64.RawURLEncoding.EncodeToString(pub.N.Bytes()),
		"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(pub.E)).Bytes()),
	}
}
