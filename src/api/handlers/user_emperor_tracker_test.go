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

func setupUserEmperorTrackerHandlerTest(t *testing.T) (*gorm.DB, *repository.UserRepository) {
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

func TestUserHandlerGetMeDefaultsEmperorTrackerFieldsToFalse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, userRepo := setupUserEmperorTrackerHandlerTest(t)
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
		t.Fatalf("failed to parse response: %v", err)
	}
	for _, field := range []string{
		"emperorTrackerEnabled",
		"emperorTrackerShowUsurpers",
		"emperorTrackerShowEmpresses",
		"emperorTrackerShowOtherFigures",
	} {
		if body[field] != false {
			t.Errorf("%s = %v, want false by default", field, body[field])
		}
	}
}

func TestUserHandlerUpdateProfileTogglesEmperorTrackerFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, userRepo := setupUserEmperorTrackerHandlerTest(t)
	user := models.User{Username: "collector", PasswordHash: "hash"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	handler := NewUserHandler("", userRepo, nil, services.NewLogger(10))
	body := bytes.NewBufferString(`{
		"emperorTrackerEnabled": true,
		"emperorTrackerShowUsurpers": true,
		"emperorTrackerShowEmpresses": true,
		"emperorTrackerShowOtherFigures": true
	}`)
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
	if !stored.EmperorTrackerEnabled || !stored.EmperorTrackerShowUsurpers ||
		!stored.EmperorTrackerShowEmpresses || !stored.EmperorTrackerShowOtherFigures {
		t.Fatalf("expected all emperor tracker fields enabled, got %+v", stored)
	}
}

func TestUserHandlerUpdateProfileLeavesEmperorTrackerFieldsUnchangedWhenOmitted(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, userRepo := setupUserEmperorTrackerHandlerTest(t)
	user := models.User{Username: "collector", PasswordHash: "hash", EmperorTrackerEnabled: true}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	handler := NewUserHandler("", userRepo, nil, services.NewLogger(10))
	body := bytes.NewBufferString(`{"bio":"just updating bio"}`)
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
	if !stored.EmperorTrackerEnabled {
		t.Fatal("expected EmperorTrackerEnabled to remain true when omitted from update")
	}
}
