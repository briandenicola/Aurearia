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

func setupUserSwipeNavHandlerTest(t *testing.T) (*gorm.DB, *repository.UserRepository) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}); err != nil {
		t.Fatalf("migrate db: %v", err)
	}
	return db, repository.NewUserRepository(db)
}

// TestGetMeDefaultsPWASwipeNavEnabledToFalse verifies new-user default is false.
func TestGetMeDefaultsPWASwipeNavEnabledToFalse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, userRepo := setupUserSwipeNavHandlerTest(t)
	user := models.User{Username: "collector", PasswordHash: "hash"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	handler := NewUserHandler("", userRepo, nil, services.NewLogger(10))
	req := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Set("userId", user.ID)

	handler.GetMe(c)

	if w.Code != http.StatusOK {
		t.Fatalf("GetMe status = %d body=%s", w.Code, w.Body.String())
	}
	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("parse response: %v", err)
	}
	if body["pwaSwipeNavEnabled"] != false {
		t.Errorf("pwaSwipeNavEnabled = %v, want false by default", body["pwaSwipeNavEnabled"])
	}
}

// TestUpdateProfileSetsPWASwipeNavEnabled verifies the field is persisted when set to true.
func TestUpdateProfileSetsPWASwipeNavEnabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, userRepo := setupUserSwipeNavHandlerTest(t)
	user := models.User{Username: "collector", PasswordHash: "hash"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	handler := NewUserHandler("", userRepo, nil, services.NewLogger(10))
	body := bytes.NewBufferString(`{"pwaSwipeNavEnabled": true}`)
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

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("parse response: %v", err)
	}
	if resp["pwaSwipeNavEnabled"] != true {
		t.Errorf("response pwaSwipeNavEnabled = %v, want true", resp["pwaSwipeNavEnabled"])
	}

	var stored models.User
	if err := db.First(&stored, user.ID).Error; err != nil {
		t.Fatalf("reload user: %v", err)
	}
	if !stored.PWASwipeNavEnabled {
		t.Fatal("expected PWASwipeNavEnabled to be true in DB after update")
	}
}

// TestUpdateProfileClearsPWASwipeNavEnabled verifies the field is persisted when explicitly set to false.
func TestUpdateProfileClearsPWASwipeNavEnabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, userRepo := setupUserSwipeNavHandlerTest(t)
	user := models.User{Username: "collector", PasswordHash: "hash", PWASwipeNavEnabled: true}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	handler := NewUserHandler("", userRepo, nil, services.NewLogger(10))
	body := bytes.NewBufferString(`{"pwaSwipeNavEnabled": false}`)
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

	var stored models.User
	if err := db.First(&stored, user.ID).Error; err != nil {
		t.Fatalf("reload user: %v", err)
	}
	if stored.PWASwipeNavEnabled {
		t.Fatal("expected PWASwipeNavEnabled to be false in DB after explicit false update")
	}
}

// TestUpdateProfileLeavesPWASwipeNavEnabledUnchangedWhenOmitted verifies pointer
// semantics: omitting the key leaves the stored value unchanged.
func TestUpdateProfileLeavesPWASwipeNavEnabledUnchangedWhenOmitted(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, userRepo := setupUserSwipeNavHandlerTest(t)
	user := models.User{Username: "collector", PasswordHash: "hash", PWASwipeNavEnabled: true}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	handler := NewUserHandler("", userRepo, nil, services.NewLogger(10))
	body := bytes.NewBufferString(`{"bio":"numismatist"}`)
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

	var stored models.User
	if err := db.First(&stored, user.ID).Error; err != nil {
		t.Fatalf("reload user: %v", err)
	}
	if !stored.PWASwipeNavEnabled {
		t.Fatal("expected PWASwipeNavEnabled to remain true when omitted from update")
	}
}

// TestUpdateProfilePWASwipeNavEnabledOnlyAffectsAuthenticatedUser verifies that
// the update targets the authenticated userId from context, never a body-supplied ID.
func TestUpdateProfilePWASwipeNavEnabledOnlyAffectsAuthenticatedUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, userRepo := setupUserSwipeNavHandlerTest(t)

	attacker := models.User{Username: "attacker", PasswordHash: "hash"}
	victim := models.User{Username: "victim", Email: "victim@example.test", PasswordHash: "hash"}
	if err := db.Create(&attacker).Error; err != nil {
		t.Fatalf("create attacker: %v", err)
	}
	if err := db.Create(&victim).Error; err != nil {
		t.Fatalf("create victim: %v", err)
	}

	handler := NewUserHandler("", userRepo, nil, services.NewLogger(10))
	req := httptest.NewRequest(http.MethodPut, "/user/profile", bytes.NewBufferString(`{"pwaSwipeNavEnabled": true}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Set("userId", attacker.ID)

	handler.UpdateProfile(c)

	if w.Code != http.StatusOK {
		t.Fatalf("UpdateProfile status = %d body=%s", w.Code, w.Body.String())
	}

	var storedVictim models.User
	if err := db.First(&storedVictim, victim.ID).Error; err != nil {
		t.Fatalf("reload victim: %v", err)
	}
	if storedVictim.PWASwipeNavEnabled {
		t.Fatal("victim PWASwipeNavEnabled was changed — ownership control is broken")
	}

	var storedAttacker models.User
	if err := db.First(&storedAttacker, attacker.ID).Error; err != nil {
		t.Fatalf("reload attacker: %v", err)
	}
	if !storedAttacker.PWASwipeNavEnabled {
		t.Fatal("expected attacker own PWASwipeNavEnabled to be updated")
	}
}
