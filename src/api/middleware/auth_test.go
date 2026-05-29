package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

const testSecret = "middleware-test-secret"

func setupMiddlewareTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	err = db.AutoMigrate(&models.User{}, &models.ApiKey{})
	if err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	return db
}

func makeTestJWT(secret string, claims jwt.MapClaims) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, _ := token.SignedString([]byte(secret))
	return signed
}

func makeValidJWT(userID uint) string {
	return makeTestJWT(testSecret, jwt.MapClaims{
		"userId":   float64(userID),
		"username": "testuser",
		"role":     "user",
		"exp":      time.Now().Add(15 * time.Minute).Unix(),
		"iat":      time.Now().Unix(),
	})
}

func makeExpiredJWT(userID uint) string {
	return makeTestJWT(testSecret, jwt.MapClaims{
		"userId":   float64(userID),
		"username": "testuser",
		"role":     "user",
		"exp":      time.Now().Add(-1 * time.Hour).Unix(),
		"iat":      time.Now().Add(-2 * time.Hour).Unix(),
	})
}

func setupAuthRouter(db *gorm.DB) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	apiKeyAuth := repository.NewApiKeyRepository(db)
	r.Use(AuthRequired(testSecret, apiKeyAuth))
	r.GET("/protected", func(c *gin.Context) {
		userID := c.GetUint("userId")
		role, _ := c.Get("userRole")
		c.JSON(http.StatusOK, gin.H{"userId": userID, "role": role})
	})
	return r
}

// --- JWT Tests ---

func TestAuthMiddleware_ValidJWT(t *testing.T) {
	db := setupMiddlewareTestDB(t)
	router := setupAuthRouter(db)

	token := makeValidJWT(42)
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuthMiddleware_MissingAuthHeader(t *testing.T) {
	db := setupMiddlewareTestDB(t)
	router := setupAuthRouter(db)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestAuthMiddleware_MalformedToken(t *testing.T) {
	db := setupMiddlewareTestDB(t)
	router := setupAuthRouter(db)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer not-a-real-jwt")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestAuthMiddleware_ExpiredToken(t *testing.T) {
	db := setupMiddlewareTestDB(t)
	router := setupAuthRouter(db)

	token := makeExpiredJWT(42)
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestAuthMiddleware_WrongSigningSecret(t *testing.T) {
	db := setupMiddlewareTestDB(t)
	router := setupAuthRouter(db)

	token := makeTestJWT("wrong-secret", jwt.MapClaims{
		"userId":   float64(1),
		"username": "testuser",
		"role":     "user",
		"exp":      time.Now().Add(15 * time.Minute).Unix(),
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestAuthMiddleware_NoBearerPrefix(t *testing.T) {
	db := setupMiddlewareTestDB(t)
	router := setupAuthRouter(db)

	token := makeValidJWT(1)
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", token) // Missing "Bearer " prefix
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

// --- API Key Tests ---

func TestAuthMiddleware_ValidAPIKey(t *testing.T) {
	db := setupMiddlewareTestDB(t)

	user := models.User{ID: 1, Username: "apiuser", PasswordHash: "hash", Role: models.RoleUser}
	db.Create(&user)

	plainKey := "ak_test_valid_key_12345"
	keyHash := services.HashAPIKey(plainKey, testSecret)

	apiKey := models.ApiKey{
		UserID:    user.ID,
		KeyHash:   keyHash,
		KeyPrefix: "ak_test",
		Name:      "Test Key",
	}
	db.Create(&apiKey)

	router := setupAuthRouter(db)
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("X-API-Key", plainKey)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuthMiddleware_InvalidAPIKey(t *testing.T) {
	db := setupMiddlewareTestDB(t)
	router := setupAuthRouter(db)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("X-API-Key", "ak_nonexistent_key")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestAuthMiddleware_RevokedAPIKey(t *testing.T) {
	db := setupMiddlewareTestDB(t)

	user := models.User{ID: 1, Username: "revokeuser", PasswordHash: "hash", Role: models.RoleUser}
	db.Create(&user)

	plainKey := "ak_test_revoked_key"
	keyHash := services.HashAPIKey(plainKey, testSecret)

	now := time.Now()
	apiKey := models.ApiKey{
		UserID:    user.ID,
		KeyHash:   keyHash,
		KeyPrefix: "ak_test",
		Name:      "Revoked Key",
		RevokedAt: &now,
	}
	db.Create(&apiKey)

	router := setupAuthRouter(db)
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("X-API-Key", plainKey)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for revoked key, got %d", w.Code)
	}
}

// --- Token via query param ---

func TestAuthMiddleware_TokenQueryParam(t *testing.T) {
	db := setupMiddlewareTestDB(t)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	apiKeyAuth := repository.NewApiKeyRepository(db)
	r.Use(AuthRequired(testSecret, apiKeyAuth))
	r.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	token := makeValidJWT(1)
	req := httptest.NewRequest(http.MethodGet, "/protected?token="+token, nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 with query param token, got %d: %s", w.Code, w.Body.String())
	}
}
