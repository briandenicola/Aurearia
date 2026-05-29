package handlers

import (
	"bytes"
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
