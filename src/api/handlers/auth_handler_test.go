package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

const testJWTSecret = "handler-test-jwt-secret"

func setupAuthHandlerTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	err = db.AutoMigrate(&models.User{}, &models.RefreshToken{})
	if err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	return db
}

func setupAuthHandlerRouter(t *testing.T) (*gin.Engine, *gorm.DB) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db := setupAuthHandlerTestDB(t)
	authRepo := repository.NewAuthRepository(db)
	authSvc := services.NewAuthService(authRepo, testJWTSecret)
	handler := NewAuthHandler(testJWTSecret, authRepo, authSvc)

	r := gin.New()
	api := r.Group("/api")
	api.POST("/auth/register", handler.Register)
	api.POST("/auth/login", handler.Login)
	api.POST("/auth/refresh", handler.Refresh)
	api.GET("/auth/setup", handler.NeedsSetup)

	return r, db
}

func registerTestUser(t *testing.T, router *gin.Engine, username, email, password string) map[string]interface{} {
	t.Helper()
	body, _ := json.Marshal(map[string]string{
		"username": username,
		"email":    email,
		"password": password,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	return resp
}

// --- Register Tests ---

func TestRegisterHandler_Success(t *testing.T) {
	router, _ := setupAuthHandlerRouter(t)

	body, _ := json.Marshal(map[string]string{
		"username": "newuser",
		"email":    "new@example.com",
		"password": "password123",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp["token"] == nil || resp["token"] == "" {
		t.Error("expected token in response")
	}
	if resp["refreshToken"] == nil || resp["refreshToken"] == "" {
		t.Error("expected refreshToken in response")
	}
	if resp["user"] == nil {
		t.Error("expected user in response")
	}
}

func TestRegisterHandler_MissingFields(t *testing.T) {
	router, _ := setupAuthHandlerRouter(t)

	// Missing password
	body, _ := json.Marshal(map[string]string{
		"username": "nopass",
		"email":    "nopass@example.com",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing password, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRegisterHandler_ShortPassword(t *testing.T) {
	router, _ := setupAuthHandlerRouter(t)

	body, _ := json.Marshal(map[string]string{
		"username": "shortpass",
		"email":    "short@example.com",
		"password": "abc", // min=6
	})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for short password, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRegisterHandler_DuplicateUsername(t *testing.T) {
	router, _ := setupAuthHandlerRouter(t)

	registerTestUser(t, router, "dupuser", "dup1@example.com", "password123")

	// Try registering same username
	body, _ := json.Marshal(map[string]string{
		"username": "dupuser",
		"email":    "dup2@example.com",
		"password": "password456",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d: %s", w.Code, w.Body.String())
	}
}

// --- Login Tests ---

func TestLoginHandler_Success(t *testing.T) {
	router, _ := setupAuthHandlerRouter(t)

	registerTestUser(t, router, "loginuser", "login@example.com", "password123")

	body, _ := json.Marshal(map[string]string{
		"username": "loginuser",
		"password": "password123",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["token"] == nil {
		t.Error("expected token in login response")
	}
	if resp["refreshToken"] == nil {
		t.Error("expected refreshToken in login response")
	}
}

func TestLoginHandler_InvalidCredentials(t *testing.T) {
	router, _ := setupAuthHandlerRouter(t)

	registerTestUser(t, router, "loginuser2", "login2@example.com", "password123")

	body, _ := json.Marshal(map[string]string{
		"username": "loginuser2",
		"password": "wrongpassword",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestLoginHandler_NonExistentUser(t *testing.T) {
	router, _ := setupAuthHandlerRouter(t)

	body, _ := json.Marshal(map[string]string{
		"username": "nobody",
		"password": "password123",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestLoginHandler_MissingFields(t *testing.T) {
	router, _ := setupAuthHandlerRouter(t)

	body, _ := json.Marshal(map[string]string{
		"username": "onlyuser",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

// --- Refresh Tests ---

func TestRefreshHandler_Success(t *testing.T) {
	router, _ := setupAuthHandlerRouter(t)

	resp := registerTestUser(t, router, "refreshuser", "refresh@example.com", "password123")
	refreshToken, ok := resp["refreshToken"].(string)
	if !ok || refreshToken == "" {
		t.Fatal("registration did not return a refresh token")
	}

	body, _ := json.Marshal(map[string]string{
		"refreshToken": refreshToken,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var refreshResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &refreshResp)
	if refreshResp["token"] == nil {
		t.Error("expected new token in refresh response")
	}
	if refreshResp["refreshToken"] == nil {
		t.Error("expected new refreshToken in refresh response")
	}
}

func TestRefreshHandler_InvalidToken(t *testing.T) {
	router, _ := setupAuthHandlerRouter(t)

	body, _ := json.Marshal(map[string]string{
		"refreshToken": "rt_completely_invalid_token",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRefreshHandler_MissingToken(t *testing.T) {
	router, _ := setupAuthHandlerRouter(t)

	body, _ := json.Marshal(map[string]string{})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

// --- Setup Check ---

func TestNeedsSetup_NoUsers(t *testing.T) {
	router, _ := setupAuthHandlerRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/auth/setup", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["needsSetup"] != true {
		t.Errorf("expected needsSetup=true when no users, got %v", resp["needsSetup"])
	}
}

func TestNeedsSetup_WithUsers(t *testing.T) {
	router, _ := setupAuthHandlerRouter(t)

	registerTestUser(t, router, "firstuser", "first@example.com", "password123")

	req := httptest.NewRequest(http.MethodGet, "/api/auth/setup", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["needsSetup"] != false {
		t.Errorf("expected needsSetup=false after registration, got %v", resp["needsSetup"])
	}
}
