package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

func setupAdminHandlerRecoveryTest(t *testing.T) (*gin.Engine, *gorm.DB) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	if err := db.AutoMigrate(
		&models.User{},
		&models.SecurityEvent{},
		&models.Coin{},
		&models.CoinImage{},
		&models.CoinJournal{},
		&models.CoinValueHistory{},
		&models.CoinComment{},
		&models.AgentConversation{},
		&models.ValueSnapshot{},
		&models.ApiKey{},
		&models.RefreshToken{},
		&models.WebAuthnCredential{},
		&models.ExternalIdentity{},
		&models.OIDCAuthState{},
		&models.Follow{},
	); err != nil {
		t.Fatalf("failed to migrate test db: %v", err)
	}

	adminRepo := repository.NewAdminRepository(db)
	securitySvc := services.NewSecurityService(repository.NewSecurityRepository(db))
	recoverySvc := services.NewAdminRecoveryService(adminRepo, securitySvc)
	handler := NewAdminHandler("", adminRepo, recoverySvc, nil, nil, nil)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		if raw := c.GetHeader("X-Test-User-ID"); raw != "" {
			var id uint
			if _, err := fmt.Sscanf(raw, "%d", &id); err == nil {
				c.Set("userId", id)
			}
		}
		c.Next()
	})
	router.DELETE("/admin/users/:id", handler.DeleteUser)
	router.PUT("/admin/users/:id/role", handler.UpdateUserRole)
	return router, db
}

func createAdminHandlerTestUser(t *testing.T, db *gorm.DB, username string, role models.UserRole, passwordHash string) models.User {
	t.Helper()
	user := models.User{
		Username:     username,
		Email:        username + "@example.com",
		PasswordHash: passwordHash,
		Role:         role,
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}
	return user
}

func TestAdminHandlerDeleteUserBlocksFinalLocalAdmin(t *testing.T) {
	router, db := setupAdminHandlerRecoveryTest(t)
	oidcAdmin := createAdminHandlerTestUser(t, db, "oidc-admin", models.RoleAdmin, "")
	target := createAdminHandlerTestUser(t, db, "final-local-admin", models.RoleAdmin, "local-password-hash")

	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/admin/users/%d", target.ID), nil)
	req.Header.Set("X-Test-User-ID", fmt.Sprint(oidcAdmin.ID))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409 for final local admin delete, got %d: %s", w.Code, w.Body.String())
	}
	assertAdminHandlerError(t, w, services.FinalLocalAdminRecoveryMessage)

	var current models.User
	if err := db.First(&current, target.ID).Error; err != nil {
		t.Fatalf("expected final local admin to remain after blocked delete: %v", err)
	}
}

func TestAdminHandlerDeleteUserAllowsNonFinalLocalAdmin(t *testing.T) {
	router, db := setupAdminHandlerRecoveryTest(t)
	actor := createAdminHandlerTestUser(t, db, "actor-admin", models.RoleAdmin, "local-password-hash")
	target := createAdminHandlerTestUser(t, db, "target-admin", models.RoleAdmin, "local-password-hash")

	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/admin/users/%d", target.ID), nil)
	req.Header.Set("X-Test-User-ID", fmt.Sprint(actor.ID))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected non-final local admin delete to succeed, got %d: %s", w.Code, w.Body.String())
	}
	if err := db.First(&models.User{}, target.ID).Error; !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected target admin to be deleted, got %v", err)
	}
}

func TestAdminHandlerUpdateRoleBlocksFinalLocalAdminDemotion(t *testing.T) {
	router, db := setupAdminHandlerRecoveryTest(t)
	oidcAdmin := createAdminHandlerTestUser(t, db, "oidc-admin", models.RoleAdmin, "")
	target := createAdminHandlerTestUser(t, db, "final-local-admin", models.RoleAdmin, "local-password-hash")

	body, _ := json.Marshal(map[string]string{"role": string(models.RoleUser)})
	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/admin/users/%d/role", target.ID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", fmt.Sprint(oidcAdmin.ID))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409 for final local admin demotion, got %d: %s", w.Code, w.Body.String())
	}
	assertAdminHandlerError(t, w, services.FinalLocalAdminRecoveryMessage)

	var current models.User
	if err := db.First(&current, target.ID).Error; err != nil {
		t.Fatalf("expected final local admin to remain after blocked demotion: %v", err)
	}
	if current.Role != models.RoleAdmin {
		t.Fatalf("expected blocked demotion to preserve admin role, got %q", current.Role)
	}
}

func TestAdminHandlerUpdateRoleAllowsNonFinalLocalAdminDemotion(t *testing.T) {
	router, db := setupAdminHandlerRecoveryTest(t)
	actor := createAdminHandlerTestUser(t, db, "actor-admin", models.RoleAdmin, "local-password-hash")
	target := createAdminHandlerTestUser(t, db, "target-admin", models.RoleAdmin, "local-password-hash")

	body, _ := json.Marshal(map[string]string{"role": string(models.RoleUser)})
	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/admin/users/%d/role", target.ID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", fmt.Sprint(actor.ID))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected non-final local admin demotion to succeed, got %d: %s", w.Code, w.Body.String())
	}

	var current models.User
	if err := db.First(&current, target.ID).Error; err != nil {
		t.Fatalf("failed to reload demoted admin: %v", err)
	}
	if current.Role != models.RoleUser {
		t.Fatalf("expected target to be demoted to user, got %q", current.Role)
	}
}

func assertAdminHandlerError(t *testing.T, w *httptest.ResponseRecorder, expected string) {
	t.Helper()
	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse error response: %v", err)
	}
	if resp["error"] != expected {
		t.Fatalf("expected error %q, got %q", expected, resp["error"])
	}
}
