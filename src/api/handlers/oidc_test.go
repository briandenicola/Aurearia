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
	if err := db.AutoMigrate(&models.User{}, &models.RefreshToken{}, &models.WebAuthnCredential{}, &models.SecurityEvent{}, &models.IPRule{}, &models.OIDCProvider{}, &models.ExternalIdentity{}, &models.OIDCAuthState{}); err != nil {
		t.Fatalf("failed to migrate test db: %v", err)
	}

	authSvc := services.NewAuthService(repository.NewAuthRepository(db), "oidc-handler-test-secret")
	securitySvc := services.NewSecurityService(repository.NewSecurityRepository(db))
	oidcSvc := services.NewOIDCService(repository.NewOIDCRepository(db), nil).WithAuth(authSvc).WithSecurity(securitySvc)
	handler := NewOIDCHandler(oidcSvc)

	router := gin.New()
	router.POST("/api/auth/oidc/:providerId/start", handler.StartLogin)
	router.GET("/api/auth/oidc/:providerId/callback", handler.Callback)
	router.POST("/api/auth/oidc/:providerId/link/start", func(c *gin.Context) {
		c.Set("userId", uint(1))
		handler.StartLink(c)
	})
	router.GET("/api/auth/oidc/:providerId/link/callback", handler.LinkCallback)
	router.GET("/api/user/oidc-identities", func(c *gin.Context) {
		c.Set("userId", uint(1))
		handler.ListLinkedIdentities(c)
	})
	router.DELETE("/api/user/oidc-identities/:identityId", func(c *gin.Context) {
		c.Set("userId", uint(1))
		handler.UnlinkIdentity(c)
	})

	return db, router
}

func TestOIDCHandlerLinkCallbackSuccessLinksAuthenticatedUser(t *testing.T) {
	db, router := setupOIDCLoginHandlerTest(t)
	issuer := startMockOIDCHandlerProvider(t, "link-subject-123456", "collector@example.com", true)
	provider := createOIDCHandlerProvider(t, db, issuer)
	user := createOIDCHandlerUser(t, db, "collector", "collector@example.com")
	if user.ID != 1 {
		t.Fatalf("test route sets userId=1, got user ID %d", user.ID)
	}

	startBody, _ := json.Marshal(map[string]string{"redirectPath": "/settings"})
	startReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/auth/oidc/%d/link/start", provider.ID), bytes.NewReader(startBody))
	startReq.Header.Set("Content-Type", "application/json")
	startReq.Host = "app.example"
	startRec := httptest.NewRecorder()
	router.ServeHTTP(startRec, startReq)
	if startRec.Code != http.StatusOK {
		t.Fatalf("expected link start status 200, got %d: %s", startRec.Code, startRec.Body.String())
	}

	var startResp struct {
		AuthorizationURL string `json:"authorizationUrl"`
	}
	if err := json.Unmarshal(startRec.Body.Bytes(), &startResp); err != nil {
		t.Fatalf("failed to decode link start response: %v", err)
	}
	authURL, err := url.Parse(startResp.AuthorizationURL)
	if err != nil {
		t.Fatalf("failed to parse authorization URL: %v", err)
	}
	currentOIDCHandlerTestNonce = authURL.Query().Get("nonce")
	callbackPath := fmt.Sprintf("/api/auth/oidc/%d/link/callback?code=valid-code&state=%s", provider.ID, url.QueryEscape(authURL.Query().Get("state")))
	callbackReq := httptest.NewRequest(http.MethodGet, callbackPath, nil)
	callbackReq.Host = "app.example"
	callbackRec := httptest.NewRecorder()
	router.ServeHTTP(callbackRec, callbackReq)
	if callbackRec.Code != http.StatusOK {
		t.Fatalf("expected link callback status 200, got %d: %s", callbackRec.Code, callbackRec.Body.String())
	}

	var resp struct {
		Message  string `json:"message"`
		Identity struct {
			ProviderID     uint   `json:"providerId"`
			Email          string `json:"email"`
			SubjectPreview string `json:"subjectPreview"`
		} `json:"identity"`
	}
	if err := json.Unmarshal(callbackRec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode link callback response: %v", err)
	}
	if resp.Identity.ProviderID != provider.ID || resp.Identity.Email != user.Email || resp.Identity.SubjectPreview != "link-sub..." {
		t.Fatalf("unexpected link response: %+v", resp)
	}
}

func TestOIDCHandlerLinkedIdentitiesListAndUnlinkErrors(t *testing.T) {
	db, router := setupOIDCLoginHandlerTest(t)
	provider := createOIDCHandlerProvider(t, db, "http://localhost:19001")
	user := createOIDCHandlerUser(t, db, "collector", "collector@example.com")
	if user.ID != 1 {
		t.Fatalf("test route sets userId=1, got user ID %d", user.ID)
	}
	identity := models.ExternalIdentity{UserID: user.ID, ProviderID: provider.ID, Issuer: provider.IssuerURL, Subject: "subject-preview-value", Email: user.Email, EmailVerified: true}
	if err := db.Create(&identity).Error; err != nil {
		t.Fatalf("failed to create identity: %v", err)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/user/oidc-identities", nil)
	listRec := httptest.NewRecorder()
	router.ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("expected list status 200, got %d: %s", listRec.Code, listRec.Body.String())
	}
	if strings.Contains(listRec.Body.String(), "subject-preview-value") || !strings.Contains(listRec.Body.String(), "subject-...") {
		t.Fatalf("expected subject-safe preview only, got %s", listRec.Body.String())
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/user/oidc-identities/%d", identity.ID), nil)
	deleteRec := httptest.NewRecorder()
	router.ServeHTTP(deleteRec, deleteReq)
	if deleteRec.Code != http.StatusOK {
		t.Fatalf("expected unlink success with local password, got %d: %s", deleteRec.Code, deleteRec.Body.String())
	}

	oidcOnly := models.User{Username: "oidc-only", Email: "oidc-only@example.com", PasswordHash: "", Role: models.RoleUser}
	if err := db.Create(&oidcOnly).Error; err != nil {
		t.Fatalf("failed to create oidc-only user: %v", err)
	}
	oidcOnlyIdentity := models.ExternalIdentity{UserID: user.ID, ProviderID: provider.ID, Issuer: provider.IssuerURL, Subject: "only-subject", Email: oidcOnly.Email, EmailVerified: true}
	if err := db.Create(&oidcOnlyIdentity).Error; err != nil {
		t.Fatalf("failed to create oidc-only identity: %v", err)
	}
	if err := db.Model(&models.User{}).Where("id = ?", user.ID).Update("password_hash", "").Error; err != nil {
		t.Fatalf("failed to make user oidc-only: %v", err)
	}
	blockReq := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/user/oidc-identities/%d", oidcOnlyIdentity.ID), nil)
	blockRec := httptest.NewRecorder()
	router.ServeHTTP(blockRec, blockReq)
	if blockRec.Code != http.StatusConflict || !strings.Contains(blockRec.Body.String(), "last usable sign-in method") {
		t.Fatalf("expected no usable sign-in conflict, got %d: %s", blockRec.Code, blockRec.Body.String())
	}

	missingReq := httptest.NewRequest(http.MethodDelete, "/api/user/oidc-identities/999", nil)
	missingRec := httptest.NewRecorder()
	router.ServeHTTP(missingRec, missingReq)
	if missingRec.Code != http.StatusNotFound {
		t.Fatalf("expected not-found unlink status, got %d: %s", missingRec.Code, missingRec.Body.String())
	}
}

func TestOIDCHandlerMapsProviderDeniedCategory(t *testing.T) {
	_, router := setupOIDCLoginHandlerTest(t)
	req := httptest.NewRequest(http.MethodGet, "/api/auth/oidc/1/callback?error=access_denied", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "provider denied access") {
		t.Fatalf("expected provider denied category, got %d: %s", rec.Code, rec.Body.String())
	}
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

func TestOIDCHandlerDistinctOIDCErrorCategories(t *testing.T) {
	t.Run("denied provider callback is a bad request without token leakage", func(t *testing.T) {
		_, router := setupOIDCLoginHandlerTest(t)
		req := httptest.NewRequest(http.MethodGet, "/api/auth/oidc/123/callback?error=access_denied&state=opaque-state", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assertOIDCHandlerError(t, rec, http.StatusBadRequest, "OIDC provider denied access")
	})

	t.Run("disabled provider start is a conflict", func(t *testing.T) {
		db, router := setupOIDCLoginHandlerTest(t)
		issuer := startMockOIDCHandlerProvider(t, "subject-123", "collector@example.com", true)
		provider := createOIDCHandlerProvider(t, db, issuer)
		if err := db.Model(&models.OIDCProvider{}).Where("id = ?", provider.ID).Update("enabled", false).Error; err != nil {
			t.Fatalf("failed to disable provider: %v", err)
		}

		rec := oidcHandlerStartRequest(t, router, provider.ID, "/")

		assertOIDCHandlerError(t, rec, http.StatusConflict, "OIDC provider is disabled")
	})

	t.Run("unsafe redirect is a validation bad request", func(t *testing.T) {
		db, router := setupOIDCLoginHandlerTest(t)
		issuer := startMockOIDCHandlerProvider(t, "subject-123", "collector@example.com", true)
		provider := createOIDCHandlerProvider(t, db, issuer)

		rec := oidcHandlerStartRequest(t, router, provider.ID, "https://evil.example/callback")

		assertOIDCHandlerError(t, rec, http.StatusBadRequest, "Invalid redirect path")
	})

	t.Run("provider discovery failure is a setup error", func(t *testing.T) {
		db, router := setupOIDCLoginHandlerTest(t)
		discoveryFailure := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "discovery down", http.StatusInternalServerError)
		}))
		t.Cleanup(discoveryFailure.Close)
		provider := createOIDCHandlerProvider(t, db, discoveryFailure.URL)

		rec := oidcHandlerStartRequest(t, router, provider.ID, "/")

		assertOIDCHandlerError(t, rec, http.StatusInternalServerError, "OIDC provider is misconfigured")
	})

	t.Run("token validation failure is unauthorized", func(t *testing.T) {
		db, router := setupOIDCLoginHandlerTest(t)
		issuer := startMockOIDCHandlerProvider(t, "subject-123", "collector@example.com", false)
		provider := createOIDCHandlerProvider(t, db, issuer)
		user := createOIDCHandlerUser(t, db, "collector", "collector@example.com")
		if err := db.Create(&models.ExternalIdentity{UserID: user.ID, ProviderID: provider.ID, Issuer: issuer, Subject: "subject-123", Email: user.Email, EmailVerified: false}).Error; err != nil {
			t.Fatalf("failed to create external identity: %v", err)
		}

		rec := oidcHandlerCallbackAfterStart(t, router, provider.ID)

		assertOIDCHandlerError(t, rec, http.StatusUnauthorized, "OIDC validation failed")
	})

	t.Run("matching local email without explicit link is an account conflict", func(t *testing.T) {
		db, router := setupOIDCLoginHandlerTest(t)
		issuer := startMockOIDCHandlerProvider(t, "new-subject", "collector@example.com", true)
		provider := createOIDCHandlerProvider(t, db, issuer)
		createOIDCHandlerUser(t, db, "collector", "collector@example.com")

		rec := oidcHandlerCallbackAfterStart(t, router, provider.ID)

		assertOIDCHandlerError(t, rec, http.StatusConflict, "Sign in locally and link this OIDC identity from Account Settings")
	})
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

func oidcHandlerStartRequest(t *testing.T, router http.Handler, providerID uint, redirectPath string) *httptest.ResponseRecorder {
	t.Helper()
	startBody, _ := json.Marshal(map[string]string{"redirectPath": redirectPath})
	startReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/auth/oidc/%d/start", providerID), bytes.NewReader(startBody))
	startReq.Header.Set("Content-Type", "application/json")
	startReq.Host = "app.example"
	startRec := httptest.NewRecorder()
	router.ServeHTTP(startRec, startReq)
	return startRec
}

func oidcHandlerCallbackAfterStart(t *testing.T, router http.Handler, providerID uint) *httptest.ResponseRecorder {
	t.Helper()
	startRec := oidcHandlerStartRequest(t, router, providerID, "/")
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
	currentOIDCHandlerTestNonce = authURL.Query().Get("nonce")
	callbackPath := fmt.Sprintf("/api/auth/oidc/%d/callback?code=valid-code&state=%s", providerID, url.QueryEscape(authURL.Query().Get("state")))
	callbackReq := httptest.NewRequest(http.MethodGet, callbackPath, nil)
	callbackReq.Host = "app.example"
	callbackRec := httptest.NewRecorder()
	router.ServeHTTP(callbackRec, callbackReq)
	return callbackRec
}

func assertOIDCHandlerError(t *testing.T, rec *httptest.ResponseRecorder, expectedStatus int, expectedMessage string) {
	t.Helper()
	if rec.Code != expectedStatus {
		t.Fatalf("expected status %d, got %d: %s", expectedStatus, rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), expectedMessage) {
		t.Fatalf("expected response to contain %q, got %s", expectedMessage, rec.Body.String())
	}
	for _, leaked := range []string{"provider-access-token", "valid-code", currentOIDCHandlerTestNonce} {
		if leaked != "" && strings.Contains(rec.Body.String(), leaked) {
			t.Fatalf("error response leaked provider-only secret %q: %s", leaked, rec.Body.String())
		}
	}
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
