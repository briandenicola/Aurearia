package handlers

import (
	"bytes"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupCredentialEncryptionHandlerTest(t *testing.T) (*gorm.DB, *repository.UserRepository, *services.CredentialEncryptionService) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}); err != nil {
		t.Fatalf("migrate db: %v", err)
	}
	credentialSvc, err := services.NewCredentialEncryptionService(base64.StdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef")))
	if err != nil {
		t.Fatalf("credential service: %v", err)
	}
	return db, repository.NewUserRepository(db), credentialSvc
}

func TestUserHandlerUpdateProfileEncryptsAuctionCredentials(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, userRepo, credentialSvc := setupCredentialEncryptionHandlerTest(t)
	user := models.User{Username: "collector", PasswordHash: "hash"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	handler := NewUserHandler("", userRepo, nil, services.NewLogger(10), credentialSvc)
	body := bytes.NewBufferString(`{"numisBidsUsername":"nb-user","numisBidsPassword":"nb-secret","cngUsername":"cng-user","cngPassword":"cng-secret"}`)
	req := httptest.NewRequest(http.MethodPut, "/user/profile", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Set("userId", user.ID)

	handler.UpdateProfile(c)

	if w.Code != http.StatusOK {
		t.Fatalf("UpdateProfile status = %d body=%s", w.Code, w.Body.String())
	}
	if strings.Contains(w.Body.String(), "nb-secret") || strings.Contains(w.Body.String(), "cng-secret") {
		t.Fatalf("profile response leaked credential: %s", w.Body.String())
	}

	var stored models.User
	if err := db.First(&stored, user.ID).Error; err != nil {
		t.Fatalf("reload user: %v", err)
	}
	if stored.NumisBidsPassword == "nb-secret" || stored.CNGPassword == "cng-secret" {
		t.Fatal("auction credentials were stored as plaintext")
	}
	if !strings.HasPrefix(stored.NumisBidsPassword, "enc:v1:") || !strings.HasPrefix(stored.CNGPassword, "enc:v1:") {
		t.Fatalf("auction credentials were not encrypted: nb=%q cng=%q", stored.NumisBidsPassword, stored.CNGPassword)
	}
}

func TestAuctionLotHandlerLazyMigratesLegacyAuctionCredentials(t *testing.T) {
	db, userRepo, credentialSvc := setupCredentialEncryptionHandlerTest(t)
	user := models.User{Username: "collector", PasswordHash: "hash", NumisBidsPassword: "legacy-nb", CNGPassword: "legacy-cng"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	handler := NewAuctionLotHandler(nil, nil, userRepo, nil, nil, nil, credentialSvc)
	plain, shouldMigrate, err := handler.decryptStoredCredential(user.ID, "numis_bids_password", user.NumisBidsPassword)
	if err != nil {
		t.Fatalf("decrypt legacy NumisBids credential: %v", err)
	}
	if plain != "legacy-nb" || !shouldMigrate {
		t.Fatalf("legacy decrypt plain=%q shouldMigrate=%t", plain, shouldMigrate)
	}
	handler.migrateStoredCredential(&user, "numis_bids_password", plain)

	var stored models.User
	if err := db.First(&stored, user.ID).Error; err != nil {
		t.Fatalf("reload user: %v", err)
	}
	if stored.NumisBidsPassword == "legacy-nb" || !strings.HasPrefix(stored.NumisBidsPassword, "enc:v1:") {
		t.Fatalf("legacy NumisBids credential was not migrated: %q", stored.NumisBidsPassword)
	}
	decrypted, wasEncrypted, err := credentialSvc.DecryptStringWithAAD(stored.NumisBidsPassword, auctionCredentialAAD(user.ID, "numis_bids_password"))
	if err != nil {
		t.Fatalf("decrypt migrated credential: %v", err)
	}
	if !wasEncrypted || decrypted != "legacy-nb" {
		t.Fatalf("migrated decrypt wasEncrypted=%t plain=%q", wasEncrypted, decrypted)
	}
}
