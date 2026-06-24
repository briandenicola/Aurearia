package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/middleware"
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
	err = db.AutoMigrate(&models.User{}, &models.RefreshToken{}, &models.AppSetting{}, &models.SecurityEvent{}, &models.IPRule{}, &models.OIDCProvider{})
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

func setupAuthHandlerRouterWithSettings(t *testing.T) (*gin.Engine, *gorm.DB, *services.SettingsService) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db := setupAuthHandlerTestDB(t)
	authRepo := repository.NewAuthRepository(db)
	settingsSvc := services.NewSettingsService(repository.NewSettingsRepository(db))
	securitySvc := services.NewSecurityService(repository.NewSecurityRepository(db))
	oidcRepo := repository.NewOIDCRepository(db)
	authSvc := services.NewAuthService(authRepo, testJWTSecret).WithSettings(settingsSvc).WithSecurity(securitySvc).WithOIDC(oidcRepo)
	handler := NewAuthHandler(testJWTSecret, authRepo, authSvc)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("clientIP", c.ClientIP())
		c.Next()
	})
	api := r.Group("/api")
	api.POST("/auth/register", handler.Register)
	api.POST("/auth/login", handler.Login)
	api.POST("/auth/refresh", handler.Refresh)
	api.GET("/auth/setup", handler.NeedsSetup)

	return r, db, settingsSvc
}

func setupAuthHandlerRouterWithLoginLimit(t *testing.T, limit int) (*gin.Engine, *gorm.DB) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db := setupAuthHandlerTestDB(t)
	authRepo := repository.NewAuthRepository(db)
	authSvc := services.NewAuthService(authRepo, testJWTSecret)
	handler := NewAuthHandler(testJWTSecret, authRepo, authSvc)

	r := gin.New()
	api := r.Group("/api")
	api.POST("/auth/register", handler.Register)
	api.POST("/auth/login", middleware.RateLimit(limit, time.Minute), handler.Login)

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

func TestRegisterHandler_RegistrationModeDefaultAllowsLocalUsersWhenNoOIDCProviderEnabled(t *testing.T) {
	routerWithSettings, _, _ := setupAuthHandlerRouterWithSettings(t)
	first := registerTestUser(t, routerWithSettings, "first", "first@example.com", "password123")
	if first["token"] == nil {
		t.Fatalf("expected first user registration to be allowed when setup is empty: %#v", first)
	}

	body, _ := json.Marshal(map[string]string{
		"username": "second",
		"email":    "second@example.com",
		"password": "password123",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	routerWithSettings.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 when no OIDC provider is enabled, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRegisterHandler_RegistrationModeDefaultAllowsLocalUsersWhenOIDCProviderDisabled(t *testing.T) {
	routerWithSettings, db, _ := setupAuthHandlerRouterWithSettings(t)
	registerTestUser(t, routerWithSettings, "first", "first@example.com", "password123")
	provider := models.OIDCProvider{
		Name:                 "entra",
		DisplayName:          "Microsoft Entra ID",
		ProviderType:         models.OIDCProviderTypeEntra,
		Enabled:              false,
		IssuerURL:            "https://login.microsoftonline.com/test-tenant/v2.0",
		ClientID:             "client-id",
		ClientSecret:         "client-secret",
		Scopes:               models.StringList{"openid", "email", "profile"},
		CallbackPath:         "/api/auth/oidc/1/callback",
		RequireVerifiedEmail: true,
	}
	if err := db.Create(&provider).Error; err != nil {
		t.Fatalf("failed to create disabled OIDC provider: %v", err)
	}

	body, _ := json.Marshal(map[string]string{
		"username": "second",
		"email":    "second@example.com",
		"password": "password123",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	routerWithSettings.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 when OIDC provider is disabled, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRegisterHandler_RegistrationModeDefaultClosedWhenOIDCProviderEnabled(t *testing.T) {
	routerWithSettings, db, _ := setupAuthHandlerRouterWithSettings(t)
	registerTestUser(t, routerWithSettings, "first", "first@example.com", "password123")
	provider := models.OIDCProvider{
		Name:                 "entra",
		DisplayName:          "Microsoft Entra ID",
		ProviderType:         models.OIDCProviderTypeEntra,
		Enabled:              true,
		IssuerURL:            "https://login.microsoftonline.com/test-tenant/v2.0",
		ClientID:             "client-id",
		ClientSecret:         "client-secret",
		Scopes:               models.StringList{"openid", "email", "profile"},
		CallbackPath:         "/api/auth/oidc/1/callback",
		RequireVerifiedEmail: true,
	}
	if err := db.Create(&provider).Error; err != nil {
		t.Fatalf("failed to create OIDC provider: %v", err)
	}

	body, _ := json.Marshal(map[string]string{
		"username": "second",
		"email":    "second@example.com",
		"password": "password123",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	routerWithSettings.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 when OIDC provider is enabled and registration is closed, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRegisterHandler_RegistrationModeOpenAllowsSecondUser(t *testing.T) {
	router, _, settingsSvc := setupAuthHandlerRouterWithSettings(t)
	registerTestUser(t, router, "first", "first@example.com", "password123")
	if err := settingsSvc.SetSetting(services.SettingRegistrationMode, "open"); err != nil {
		t.Fatalf("failed to open registration: %v", err)
	}

	body, _ := json.Marshal(map[string]string{
		"username": "second",
		"email":    "second@example.com",
		"password": "password123",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 when registration is open, got %d: %s", w.Code, w.Body.String())
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

func TestLocalPasswordLoginAndRefreshRegressionWithOIDCSchema(t *testing.T) {
	router, db := setupAuthHandlerRouter(t)
	if err := db.AutoMigrate(&models.OIDCProvider{}, &models.ExternalIdentity{}, &models.OIDCAuthState{}); err != nil {
		t.Fatalf("failed to migrate OIDC schema: %v", err)
	}

	registerTestUser(t, router, "local-oidc-regression", "local-oidc@example.com", "password123")

	loginBody, _ := json.Marshal(map[string]string{
		"username": "local-oidc-regression",
		"password": "password123",
	})
	loginReq := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")
	loginW := httptest.NewRecorder()
	router.ServeHTTP(loginW, loginReq)

	if loginW.Code != http.StatusOK {
		t.Fatalf("expected local login to stay successful with OIDC schema present, got %d: %s", loginW.Code, loginW.Body.String())
	}

	var loginResp struct {
		Token        string  `json:"token"`
		RefreshToken string  `json:"refreshToken"`
		User         UserDTO `json:"user"`
	}
	if err := json.Unmarshal(loginW.Body.Bytes(), &loginResp); err != nil {
		t.Fatalf("failed to parse login response: %v", err)
	}
	if loginResp.Token == "" || loginResp.RefreshToken == "" {
		t.Fatalf("expected local login auth response tokens, got token=%q refresh=%q", loginResp.Token, loginResp.RefreshToken)
	}
	if loginResp.User.Username != "local-oidc-regression" {
		t.Fatalf("expected local login user payload to be preserved, got %+v", loginResp.User)
	}

	refreshBody, _ := json.Marshal(map[string]string{"refreshToken": loginResp.RefreshToken})
	refreshReq := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", bytes.NewReader(refreshBody))
	refreshReq.Header.Set("Content-Type", "application/json")
	refreshW := httptest.NewRecorder()
	router.ServeHTTP(refreshW, refreshReq)

	if refreshW.Code != http.StatusOK {
		t.Fatalf("expected refresh to stay successful with OIDC schema present, got %d: %s", refreshW.Code, refreshW.Body.String())
	}

	var refreshResp struct {
		Token        string  `json:"token"`
		RefreshToken string  `json:"refreshToken"`
		User         UserDTO `json:"user"`
	}
	if err := json.Unmarshal(refreshW.Body.Bytes(), &refreshResp); err != nil {
		t.Fatalf("failed to parse refresh response: %v", err)
	}
	if refreshResp.Token == "" || refreshResp.RefreshToken == "" {
		t.Fatalf("expected refreshed auth response tokens, got token=%q refresh=%q", refreshResp.Token, refreshResp.RefreshToken)
	}
	if refreshResp.RefreshToken == loginResp.RefreshToken {
		t.Fatal("expected refresh token rotation to remain one-time-use")
	}
	if refreshResp.User.Username != "local-oidc-regression" {
		t.Fatalf("expected refreshed user payload to be preserved, got %+v", refreshResp.User)
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

func TestLoginHandler_BadPasswordThresholdReturns429GenericResponse(t *testing.T) {
	router, _ := setupAuthHandlerRouterWithLoginLimit(t, 2)
	registerTestUser(t, router, "limiteduser", "limited@example.com", "password123")

	for i := 0; i < 2; i++ {
		w := loginFromIP(t, router, "limiteduser", "wrong-password", "203.0.113.10")
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("attempt %d: expected 401 before threshold, got %d: %s", i+1, w.Code, w.Body.String())
		}
		assertErrorMessage(t, w, "Invalid credentials")
	}

	w := loginFromIP(t, router, "limiteduser", "wrong-password", "203.0.113.10")
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 after same-IP threshold, got %d: %s", w.Code, w.Body.String())
	}
	assertErrorMessage(t, w, "Too many requests. Please try again later.")
}

func TestLoginHandler_NonexistentUsernameContributesToIPThrottleWithoutEnumeration(t *testing.T) {
	router, _ := setupAuthHandlerRouterWithLoginLimit(t, 2)
	registerTestUser(t, router, "knownuser", "known@example.com", "password123")

	first := loginFromIP(t, router, "missing-user", "password123", "203.0.113.11")
	if first.Code != http.StatusUnauthorized {
		t.Fatalf("expected nonexistent username to return generic 401, got %d: %s", first.Code, first.Body.String())
	}
	assertErrorMessage(t, first, "Invalid credentials")

	second := loginFromIP(t, router, "knownuser", "wrong-password", "203.0.113.11")
	if second.Code != http.StatusUnauthorized {
		t.Fatalf("expected second bad login before threshold to return 401, got %d: %s", second.Code, second.Body.String())
	}
	assertErrorMessage(t, second, "Invalid credentials")

	throttled := loginFromIP(t, router, "knownuser", "password123", "203.0.113.11")
	if throttled.Code != http.StatusTooManyRequests {
		t.Fatalf("expected nonexistent username attempt to consume same IP throttle bucket, got %d: %s", throttled.Code, throttled.Body.String())
	}
	assertErrorMessage(t, throttled, "Too many requests. Please try again later.")
}

func loginFromIP(t *testing.T, router *gin.Engine, username, password, ip string) *httptest.ResponseRecorder {
	t.Helper()
	body, _ := json.Marshal(map[string]string{
		"username": username,
		"password": password,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = ip + ":12345"
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func assertErrorMessage(t *testing.T, w *httptest.ResponseRecorder, expected string) {
	t.Helper()
	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse error response: %v; body=%s", err, w.Body.String())
	}
	if resp["error"] != expected {
		t.Fatalf("expected generic error %q, got %q", expected, resp["error"])
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
