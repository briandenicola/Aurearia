package handlers

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"gorm.io/gorm"
)

func setupWebAuthnHandlerForTest(t *testing.T, origins string) (*WebAuthnHandler, *gorm.DB) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.WebAuthnCredential{}); err != nil {
		t.Fatalf("failed to migrate db: %v", err)
	}

	repo := repository.NewWebAuthnRepository(db)
	handler, err := NewWebAuthnHandler("localhost", origins, nil, repo, services.NewLogger(50))
	if err != nil {
		t.Fatalf("failed to create handler: %v", err)
	}

	return handler, db
}

func setupWebAuthnHandlerWithAuthForTest(t *testing.T, origins string) (*WebAuthnHandler, *gorm.DB) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.WebAuthnCredential{}, &models.SecurityEvent{}, &models.IPRule{}, &models.RefreshToken{}); err != nil {
		t.Fatalf("failed to migrate db: %v", err)
	}

	authRepo := repository.NewAuthRepository(db)
	securitySvc := services.NewSecurityService(repository.NewSecurityRepository(db))
	authSvc := services.NewAuthService(authRepo, "webauthn-test-secret").WithSecurity(securitySvc)
	authHandler := NewAuthHandler("webauthn-test-secret", authRepo, authSvc)
	handler, err := NewWebAuthnHandler("localhost", origins, authHandler, repository.NewWebAuthnRepository(db), services.NewLogger(50))
	if err != nil {
		t.Fatalf("failed to create handler: %v", err)
	}

	return handler, db
}

func createWebAuthnTestUser(t *testing.T, db *gorm.DB, username string) *models.User {
	t.Helper()
	user := &models.User{
		Email:        username + "@example.com",
		Username:     username,
		PasswordHash: "hashed-password",
		Role:         models.RoleUser,
	}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}
	return user
}

func decodeErrorResponse(t *testing.T, body *bytes.Buffer) map[string]string {
	t.Helper()
	var resp map[string]string
	if err := json.Unmarshal(body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	return resp
}

func TestWebAuthnHandlerLoginFinishRejectsLockedAccountBeforeAssertion(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler, db := setupWebAuthnHandlerWithAuthForTest(t, "http://localhost:8080")
	user := createWebAuthnTestUser(t, db, "locked-webauthn-user")
	lockedUntil := time.Now().Add(time.Hour)
	if err := db.Model(&models.User{}).Where("id = ?", user.ID).Update("locked_until", lockedUntil).Error; err != nil {
		t.Fatalf("failed to lock user: %v", err)
	}

	handler.storeSession(sessionKey("login", user.ID), &webauthn.SessionData{
		Challenge: "test-challenge",
		Expires:   time.Now().Add(2 * time.Minute),
	})

	router := gin.New()
	router.POST("/auth/webauthn/login/finish", handler.LoginFinish)

	req := httptest.NewRequest(http.MethodPost, "/auth/webauthn/login/finish?username=locked-webauthn-user", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://localhost:8080")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d: %s", http.StatusUnauthorized, rr.Code, rr.Body.String())
	}

	resp := decodeErrorResponse(t, rr.Body)
	if got := resp["error"]; got != "Invalid credentials" {
		t.Fatalf("expected lockout gate error, got %q", got)
	}

	var webAuthnFailures int64
	if err := db.Model(&models.SecurityEvent{}).Where("type = ?", models.SecurityEventWebAuthnLoginFailure).Count(&webAuthnFailures).Error; err != nil {
		t.Fatalf("failed to count WebAuthn failures: %v", err)
	}
	if webAuthnFailures != 0 {
		t.Fatalf("expected lockout gate to reject before assertion parsing, got %d WebAuthn failures", webAuthnFailures)
	}
}

func TestWebAuthnHandlerTokenIssuanceUsesSharedAuthResponseShape(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler, db := setupWebAuthnHandlerWithAuthForTest(t, "http://localhost:8080")
	user := createWebAuthnTestUser(t, db, "webauthn-token-user")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	handler.auth.issueTokens(c, *user, http.StatusOK)

	if w.Code != http.StatusOK {
		t.Fatalf("expected token issuance status 200, got %d: %s", w.Code, w.Body.String())
	}
	if w.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("expected no-store auth response, got %q", w.Header().Get("Cache-Control"))
	}

	var resp struct {
		Token        string `json:"token"`
		RefreshToken string `json:"refreshToken"`
		User         struct {
			ID       uint   `json:"id"`
			Username string `json:"username"`
			Email    string `json:"email"`
		} `json:"user"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse auth response: %v", err)
	}
	if resp.Token == "" || resp.RefreshToken == "" {
		t.Fatalf("expected access and refresh tokens, got %s", w.Body.String())
	}
	if resp.User.ID != user.ID || resp.User.Username != user.Username || resp.User.Email != user.Email {
		t.Fatalf("expected auth response user to match WebAuthn user, got %+v", resp.User)
	}
}

func TestWebAuthnHandlerLoginBeginReturnsRequestOptionsWithChallenge(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler, db := setupWebAuthnHandlerForTest(t, "http://localhost:8080")
	user := createWebAuthnTestUser(t, db, "login-begin-user")
	rawCredentialID := []byte("credential-id")
	if err := db.Create(&models.WebAuthnCredential{
		UserID:          user.ID,
		CredentialID:    base64.RawURLEncoding.EncodeToString(rawCredentialID),
		PublicKey:       []byte("public-key"),
		AttestationType: "none",
		Name:            "iPhone passkey",
	}).Error; err != nil {
		t.Fatalf("failed to create credential: %v", err)
	}

	router := gin.New()
	router.POST("/auth/webauthn/login/begin", handler.LoginBegin)

	req := httptest.NewRequest(http.MethodPost, "/auth/webauthn/login/begin", bytes.NewBufferString(`{"username":"login-begin-user"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}

	var resp struct {
		Username string `json:"username"`
		Options  struct {
			Challenge          string `json:"challenge"`
			RelyingPartyID     string `json:"rpId"`
			AllowedCredentials []struct {
				ID   string `json:"id"`
				Type string `json:"type"`
			} `json:"allowCredentials"`
			PublicKey map[string]interface{} `json:"publicKey"`
		} `json:"options"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp.Username != "login-begin-user" {
		t.Fatalf("expected username %q, got %q", "login-begin-user", resp.Username)
	}
	if resp.Options.Challenge == "" {
		t.Fatal("expected non-empty options.challenge")
	}
	if resp.Options.RelyingPartyID != "localhost" {
		t.Fatalf("expected rpId %q, got %q", "localhost", resp.Options.RelyingPartyID)
	}
	if len(resp.Options.AllowedCredentials) != 1 {
		t.Fatalf("expected one allowed credential, got %d", len(resp.Options.AllowedCredentials))
	}
	if resp.Options.AllowedCredentials[0].ID != base64.RawURLEncoding.EncodeToString(rawCredentialID) {
		t.Fatalf("expected allowed credential id to match stored credential")
	}
	if resp.Options.AllowedCredentials[0].Type != "public-key" {
		t.Fatalf("expected allowed credential type public-key, got %q", resp.Options.AllowedCredentials[0].Type)
	}
	if resp.Options.PublicKey != nil {
		t.Fatal("expected options to be the PublicKeyCredentialRequestOptions object, not nested under publicKey")
	}

	session, _, ok := handler.loadSession(sessionKey("login", user.ID))
	if !ok {
		t.Fatal("expected login session to be stored")
	}
	if session.Challenge != resp.Options.Challenge {
		t.Fatalf("expected response challenge to match stored session challenge")
	}
}

func TestWebAuthnHandlerLoginFinishDisallowedOrigin(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler, db := setupWebAuthnHandlerForTest(t, "http://localhost:8080")
	user := createWebAuthnTestUser(t, db, "webauthn-user")

	handler.storeSession(sessionKey("login", user.ID), &webauthn.SessionData{
		Challenge: "test-challenge",
		Expires:   time.Now().Add(2 * time.Minute),
	})

	router := gin.New()
	router.POST("/auth/webauthn/login/finish", handler.LoginFinish)

	req := httptest.NewRequest(http.MethodPost, "/auth/webauthn/login/finish?username=webauthn-user", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://evil.example")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, rr.Code)
	}

	resp := decodeErrorResponse(t, rr.Body)
	if got := resp["error"]; got != "WebAuthn origin not allowed" {
		t.Fatalf("expected disallowed-origin error, got %q", got)
	}
}

func TestWebAuthnHandlerLoginFinishExpiredSession(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler, db := setupWebAuthnHandlerForTest(t, "http://localhost:8080")
	user := createWebAuthnTestUser(t, db, "expired-session-user")

	now := time.Date(2025, 1, 10, 12, 0, 0, 0, time.UTC)
	handler.now = func() time.Time { return now }

	sessionKey := sessionKey("login", user.ID)
	handler.sessionMu.Lock()
	handler.sessions[sessionKey] = webauthnCeremonySession{
		data:      &webauthn.SessionData{Challenge: "test-challenge"},
		expiresAt: now.Add(-1 * time.Second),
	}
	handler.sessionMu.Unlock()

	router := gin.New()
	router.POST("/auth/webauthn/login/finish", handler.LoginFinish)

	req := httptest.NewRequest(http.MethodPost, "/auth/webauthn/login/finish?username=expired-session-user", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://localhost:8080")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}

	resp := decodeErrorResponse(t, rr.Body)
	if got := resp["error"]; got != "Login session expired. Please start login again." {
		t.Fatalf("expected expired-session error, got %q", got)
	}

	handler.sessionMu.RLock()
	_, exists := handler.sessions[sessionKey]
	handler.sessionMu.RUnlock()
	if exists {
		t.Fatal("expected expired session to be removed from memory")
	}
}

func TestWebAuthnHandlerLoginFinishMissingSession(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler, db := setupWebAuthnHandlerForTest(t, "http://localhost:8080")
	_ = createWebAuthnTestUser(t, db, "missing-session-user")

	router := gin.New()
	router.POST("/auth/webauthn/login/finish", handler.LoginFinish)

	req := httptest.NewRequest(http.MethodPost, "/auth/webauthn/login/finish?username=missing-session-user", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://localhost:8080")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}

	resp := decodeErrorResponse(t, rr.Body)
	if got := resp["error"]; got != "Login session missing. Please start login again." {
		t.Fatalf("expected missing-session error, got %q", got)
	}
}

func TestWebAuthnHandlerLoadCredentialsRestoresBackupFlags(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler, db := setupWebAuthnHandlerForTest(t, "http://localhost:8080")
	user := createWebAuthnTestUser(t, db, "backup-flags-user")

	rawCredentialID := []byte("test-credential-id")
	if err := db.Create(&models.WebAuthnCredential{
		UserID:         user.ID,
		CredentialID:   base64.RawURLEncoding.EncodeToString(rawCredentialID),
		PublicKey:      []byte("public-key-data"),
		SignCount:      5,
		BackupEligible: boolPtr(true),
		BackupState:    boolPtr(true),
		Name:           "Test passkey",
	}).Error; err != nil {
		t.Fatalf("failed to create credential: %v", err)
	}

	creds := handler.loadCredentials(user.ID)
	if len(creds) != 1 {
		t.Fatalf("expected 1 credential, got %d", len(creds))
	}

	if !creds[0].Flags.BackupEligible {
		t.Fatal("expected BackupEligible flag to be restored as true")
	}
	if !creds[0].Flags.BackupState {
		t.Fatal("expected BackupState flag to be restored as true")
	}
	if creds[0].Authenticator.SignCount != 5 {
		t.Fatalf("expected SignCount 5, got %d", creds[0].Authenticator.SignCount)
	}
}

func TestWebAuthnHandlerLoadCredentialsBootstrapsLegacyBackupFlagsFromAssertion(t *testing.T) {
	rawCredentialID := []byte("legacy-credential-id")
	credentialID := base64.RawURLEncoding.EncodeToString(rawCredentialID)
	assertion := &protocol.ParsedCredentialAssertionData{
		ParsedPublicKeyCredential: protocol.ParsedPublicKeyCredential{
			ParsedCredential: protocol.ParsedCredential{
				ID:   credentialID,
				Type: string(protocol.PublicKeyCredentialType),
			},
		},
		Response: protocol.ParsedAssertionResponse{
			AuthenticatorData: protocol.AuthenticatorData{
				Flags: protocol.FlagBackupEligible | protocol.FlagBackupState,
			},
		},
	}

	creds := webauthnCredentialsFromModels([]models.WebAuthnCredential{
		{
			CredentialID: credentialID,
			PublicKey:    []byte("public-key-data"),
			SignCount:    5,
			Name:         "Legacy passkey",
		},
	}, assertion)
	if len(creds) != 1 {
		t.Fatalf("expected 1 credential, got %d", len(creds))
	}
	if !creds[0].Flags.BackupEligible {
		t.Fatal("expected legacy credential BackupEligible to be bootstrapped from assertion")
	}
	if !creds[0].Flags.BackupState {
		t.Fatal("expected legacy credential BackupState to be bootstrapped from assertion")
	}
}

func TestWebAuthnHandlerLoadCredentialsKeepsStoredBackupEligibleOverAssertion(t *testing.T) {
	rawCredentialID := []byte("stored-credential-id")
	credentialID := base64.RawURLEncoding.EncodeToString(rawCredentialID)
	assertion := &protocol.ParsedCredentialAssertionData{
		ParsedPublicKeyCredential: protocol.ParsedPublicKeyCredential{
			ParsedCredential: protocol.ParsedCredential{
				ID:   credentialID,
				Type: string(protocol.PublicKeyCredentialType),
			},
		},
		Response: protocol.ParsedAssertionResponse{
			AuthenticatorData: protocol.AuthenticatorData{
				Flags: protocol.FlagBackupEligible,
			},
		},
	}

	creds := webauthnCredentialsFromModels([]models.WebAuthnCredential{
		{
			CredentialID:    credentialID,
			PublicKey:       []byte("public-key-data"),
			BackupEligible:  boolPtr(false),
			BackupState:     boolPtr(false),
			Name:            "Stored passkey",
			AttestationType: "none",
		},
	}, assertion)
	if len(creds) != 1 {
		t.Fatalf("expected 1 credential, got %d", len(creds))
	}
	if creds[0].Flags.BackupEligible {
		t.Fatal("expected stored BackupEligible=false to be preserved when assertion says true")
	}
}
