package services

import (
	"errors"
	"strings"
	"testing"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupAdminRecoveryServiceTest(t *testing.T) (*gorm.DB, *AdminRecoveryService) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.SecurityEvent{}); err != nil {
		t.Fatalf("failed to migrate test db: %v", err)
	}
	securitySvc := NewSecurityService(repository.NewSecurityRepository(db))
	return db, NewAdminRecoveryService(repository.NewAdminRepository(db), securitySvc)
}

func createAdminRecoveryTestUser(t *testing.T, db *gorm.DB, username string, role models.UserRole, passwordHash string) models.User {
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

func TestAdminRecoveryServiceBlocksFinalLocalAdminRemovalPaths(t *testing.T) {
	tests := []struct {
		name      string
		guard     func(*AdminRecoveryService, uint, *uint) error
		operation string
	}{
		{name: "delete", guard: (*AdminRecoveryService).EnsureCanDeleteUser, operation: "delete user"},
		{name: "demote", guard: (*AdminRecoveryService).EnsureCanDemoteUser, operation: "demote admin"},
		{name: "disable local auth", guard: (*AdminRecoveryService).EnsureCanDisableLocalAuth, operation: "disable local auth"},
		{name: "convert to OIDC-only", guard: (*AdminRecoveryService).EnsureCanConvertToOIDCOnly, operation: "convert to OIDC-only"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, svc := setupAdminRecoveryServiceTest(t)
			admin := createAdminRecoveryTestUser(t, db, "only-admin", models.RoleAdmin, "local-password-hash")
			actorID := admin.ID

			err := tt.guard(svc, admin.ID, &actorID)
			if !errors.Is(err, ErrFinalLocalAdmin) {
				t.Fatalf("expected ErrFinalLocalAdmin, got %v", err)
			}

			var event models.SecurityEvent
			if err := db.Where("type = ?", models.SecurityEventFinalAdminBlocked).First(&event).Error; err != nil {
				t.Fatalf("failed to load security event: %v", err)
			}
			if event.UserID == nil || *event.UserID != admin.ID {
				t.Fatalf("expected event for final admin %d, got %+v", admin.ID, event.UserID)
			}
			if event.Username != admin.Username {
				t.Fatalf("expected event username %q, got %q", admin.Username, event.Username)
			}
			if !strings.Contains(event.Message, tt.operation) {
				t.Fatalf("expected event message to include operation %q, got %q", tt.operation, event.Message)
			}
		})
	}
}

func TestAdminRecoveryServiceAllowsNonFinalLocalAdminRemovalPaths(t *testing.T) {
	tests := []struct {
		name  string
		guard func(*AdminRecoveryService, uint, *uint) error
	}{
		{name: "delete", guard: (*AdminRecoveryService).EnsureCanDeleteUser},
		{name: "demote", guard: (*AdminRecoveryService).EnsureCanDemoteUser},
		{name: "disable local auth", guard: (*AdminRecoveryService).EnsureCanDisableLocalAuth},
		{name: "convert to OIDC-only", guard: (*AdminRecoveryService).EnsureCanConvertToOIDCOnly},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, svc := setupAdminRecoveryServiceTest(t)
			target := createAdminRecoveryTestUser(t, db, "target-admin", models.RoleAdmin, "local-password-hash")
			actor := createAdminRecoveryTestUser(t, db, "recovery-admin", models.RoleAdmin, "local-password-hash")

			if err := tt.guard(svc, target.ID, &actor.ID); err != nil {
				t.Fatalf("expected non-final local admin operation to be allowed, got %v", err)
			}

			var count int64
			if err := db.Model(&models.SecurityEvent{}).Where("type = ?", models.SecurityEventFinalAdminBlocked).Count(&count).Error; err != nil {
				t.Fatalf("failed to count security events: %v", err)
			}
			if count != 0 {
				t.Fatalf("expected no final-local-admin security events, got %d", count)
			}
		})
	}
}

func TestAdminRecoveryServiceAllowsNonAdminOrOIDCOnlyTargetRemoval(t *testing.T) {
	tests := []struct {
		name         string
		role         models.UserRole
		passwordHash string
	}{
		{name: "non-admin local user", role: models.RoleUser, passwordHash: "local-password-hash"},
		{name: "OIDC-only admin", role: models.RoleAdmin, passwordHash: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, svc := setupAdminRecoveryServiceTest(t)
			createAdminRecoveryTestUser(t, db, "recovery-admin", models.RoleAdmin, "local-password-hash")
			target := createAdminRecoveryTestUser(t, db, "target-"+tt.name, tt.role, tt.passwordHash)

			if err := svc.EnsureCanDeleteUser(target.ID, nil); err != nil {
				t.Fatalf("expected non-recovery-path target delete to be allowed, got %v", err)
			}
		})
	}
}

func TestAdminRecoveryServiceIgnoresOIDCOnlyAdminsForRecovery(t *testing.T) {
	db, svc := setupAdminRecoveryServiceTest(t)
	localAdmin := createAdminRecoveryTestUser(t, db, "local-admin", models.RoleAdmin, "local-password-hash")
	createAdminRecoveryTestUser(t, db, "oidc-only-admin", models.RoleAdmin, "")
	actorID := localAdmin.ID

	count, err := svc.CountLocalRecoveryAdmins()
	if err != nil {
		t.Fatalf("failed to count recovery admins: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected only local admin to count for recovery, got %d", count)
	}

	err = svc.EnsureCanConvertToOIDCOnly(localAdmin.ID, &actorID)
	if !errors.Is(err, ErrFinalLocalAdmin) {
		t.Fatalf("expected OIDC-only conversion of final local admin to be blocked, got %v", err)
	}
}
