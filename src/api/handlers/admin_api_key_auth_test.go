package handlers

import (
	"bytes"
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
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

const adminAPIKeyAuthTestSecret = "admin-api-key-auth-test-secret"

func setupAdminAPIKeyAuthRouter(t *testing.T) (*gin.Engine, *gorm.DB, uint, uint) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.ApiKey{}); err != nil {
		t.Fatalf("failed to migrate test db: %v", err)
	}

	adminUser := models.User{Username: "admin", Email: "admin@test.local", PasswordHash: "hash", Role: models.RoleAdmin}
	regularUser := models.User{Username: "user", Email: "user@test.local", PasswordHash: "hash", Role: models.RoleUser}
	if err := db.Create(&adminUser).Error; err != nil {
		t.Fatalf("failed to create admin user: %v", err)
	}
	if err := db.Create(&regularUser).Error; err != nil {
		t.Fatalf("failed to create regular user: %v", err)
	}

	r := gin.New()
	apiKeyAuth := repository.NewApiKeyRepository(db)
	admin := r.Group("/api/admin")
	admin.Use(middleware.AuthRequired(adminAPIKeyAuthTestSecret, apiKeyAuth))
	admin.Use(middleware.RejectAPIKeyAuth())
	admin.Use(AdminRequired())
	admin.GET("/settings", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })
	admin.PUT("/settings", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })
	admin.DELETE("/users/:id", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })
	admin.POST("/users/:id/reset-password", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })

	return r, db, adminUser.ID, regularUser.ID
}

func makeAdminAPIKeyAuthTestJWT(t *testing.T, userID uint, role models.UserRole) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"userId": float64(userID),
		"role":   string(role),
		"exp":    time.Now().Add(15 * time.Minute).Unix(),
		"iat":    time.Now().Unix(),
	})
	signed, err := token.SignedString([]byte(adminAPIKeyAuthTestSecret))
	if err != nil {
		t.Fatalf("failed to sign JWT: %v", err)
	}
	return signed
}

func createAdminAPIKeyAuthTestKey(t *testing.T, db *gorm.DB, userID uint, plainKey string, capabilities *string) string {
	t.Helper()
	apiKey := models.ApiKey{
		UserID:    userID,
		KeyHash:   services.HashAPIKey(plainKey, adminAPIKeyAuthTestSecret),
		KeyPrefix: "ak_test",
		Name:      "test key",
	}
	if capabilities != nil {
		apiKey.Capabilities = *capabilities
	}
	if err := db.Create(&apiKey).Error; err != nil {
		t.Fatalf("failed to create API key: %v", err)
	}
	return plainKey
}

func TestAdminRoutes_AdminJWTAllowed(t *testing.T) {
	router, _, adminID, _ := setupAdminAPIKeyAuthRouter(t)
	token := makeAdminAPIKeyAuthTestJWT(t, adminID, models.RoleAdmin)

	for _, tc := range adminAuthRouteCases() {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, bytes.NewBufferString(tc.body))
			req.Header.Set("Authorization", "Bearer "+token)
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("expected 200 for admin JWT, got %d: %s", w.Code, w.Body.String())
			}
		})
	}
}

func TestAdminRoutes_NonAdminJWTForbidden(t *testing.T) {
	router, _, _, userID := setupAdminAPIKeyAuthRouter(t)
	token := makeAdminAPIKeyAuthTestJWT(t, userID, models.RoleUser)

	for _, tc := range adminAuthRouteCases() {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, bytes.NewBufferString(tc.body))
			req.Header.Set("Authorization", "Bearer "+token)
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != http.StatusForbidden {
				t.Fatalf("expected 403 for non-admin JWT, got %d: %s", w.Code, w.Body.String())
			}
		})
	}
}

func TestAdminRoutes_AdminOwnedReadAPIKeyForbidden(t *testing.T) {
	router, db, adminID, _ := setupAdminAPIKeyAuthRouter(t)
	read := "read"
	plainKey := createAdminAPIKeyAuthTestKey(t, db, adminID, "ak_admin_read_forbidden", &read)

	for _, tc := range adminAuthRouteCases() {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, bytes.NewBufferString(tc.body))
			req.Header.Set("X-API-Key", plainKey)
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != http.StatusForbidden {
				t.Fatalf("expected 403 for admin-owned read API key, got %d: %s", w.Code, w.Body.String())
			}
		})
	}
}

func TestAdminRoutes_AllAPIKeysForbidden(t *testing.T) {
	router, db, adminID, userID := setupAdminAPIKeyAuthRouter(t)
	read := "read"
	write := "read,write"
	keys := map[string]string{
		"admin default key": createAdminAPIKeyAuthTestKey(t, db, adminID, "ak_admin_default_forbidden", nil),
		"admin read key":    createAdminAPIKeyAuthTestKey(t, db, adminID, "ak_admin_read_forbidden_settings", &read),
		"admin write key":   createAdminAPIKeyAuthTestKey(t, db, adminID, "ak_admin_write_forbidden", &write),
		"user read key":     createAdminAPIKeyAuthTestKey(t, db, userID, "ak_user_read_forbidden", &read),
	}

	for name, plainKey := range keys {
		t.Run(name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/admin/settings", nil)
			req.Header.Set("X-API-Key", plainKey)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != http.StatusForbidden {
				t.Fatalf("expected 403 for %s, got %d: %s", name, w.Code, w.Body.String())
			}
		})
	}
}

type adminAuthRouteCase struct {
	method string
	path   string
	body   string
}

func adminAuthRouteCases() []adminAuthRouteCase {
	return []adminAuthRouteCase{
		{method: http.MethodGet, path: "/api/admin/settings"},
		{method: http.MethodPut, path: "/api/admin/settings", body: `[{"key":"LogLevel","value":"info"}]`},
		{method: http.MethodDelete, path: "/api/admin/users/2"},
		{method: http.MethodPost, path: "/api/admin/users/2/reset-password", body: `{"newPassword":"newpass123"}`},
	}
}
