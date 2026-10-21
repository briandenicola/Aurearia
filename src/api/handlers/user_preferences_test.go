package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
)

func putProfile(t *testing.T, handler *UserHandler, userID uint, payload string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPut, "/user/profile", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Set("userId", userID)
	handler.UpdateProfile(c)
	return w
}

// TestUpdateProfilePersistsPreferences verifies partial preference updates are
// stored, returned, and leave unspecified preferences untouched.
func TestUpdateProfilePersistsPreferences(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, userRepo := setupUserSwipeNavHandlerTest(t)
	user := models.User{Username: "collector", PasswordHash: "hash"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	handler := NewUserHandler("", userRepo, nil, services.NewLogger(10))

	w := putProfile(t, handler, user.ID, `{"preferences":{"theme":"capitoline","timezone":"America/Chicago","defaultView":"grid","defaultSort":"created_at_desc","trayFeltColor":"navy"}}`)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", w.Code, w.Body.String())
	}
	w = putProfile(t, handler, user.ID, `{"preferences":{"trayFeltColor":"green"}}`)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", w.Code, w.Body.String())
	}

	var body struct {
		Preferences map[string]string `json:"preferences"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("parse response: %v", err)
	}
	want := map[string]string{
		"theme": "capitoline", "timezone": "America/Chicago", "defaultView": "grid",
		"defaultSort": "created_at_desc", "trayFeltColor": "green",
	}
	for k, v := range want {
		if body.Preferences[k] != v {
			t.Errorf("preferences[%s] = %q, want %q", k, body.Preferences[k], v)
		}
	}
}

// TestUpdateProfileRejectsInvalidPreferences verifies unknown values are refused
// without writing anything.
func TestUpdateProfileRejectsInvalidPreferences(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, userRepo := setupUserSwipeNavHandlerTest(t)
	user := models.User{Username: "collector", PasswordHash: "hash"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	handler := NewUserHandler("", userRepo, nil, services.NewLogger(10))

	for _, payload := range []string{
		`{"preferences":{"theme":"neon"}}`,
		`{"preferences":{"timezone":"Mars/Olympus"}}`,
		`{"preferences":{"defaultView":"list"}}`,
		`{"preferences":{"defaultSort":"name; drop table users"}}`,
		`{"preferences":{"trayFeltColor":"purple"}}`,
	} {
		if w := putProfile(t, handler, user.ID, payload); w.Code != http.StatusBadRequest {
			t.Errorf("payload %s: status = %d, want 400", payload, w.Code)
		}
	}
	var reloaded models.User
	db.First(&reloaded, user.ID)
	if reloaded.PrefTheme != "" || reloaded.PrefTimezone != "" || reloaded.PrefDefaultView != "" || reloaded.PrefDefaultSort != "" || reloaded.PrefTrayFeltColor != "" {
		t.Errorf("invalid preferences were persisted: %+v", reloaded)
	}
}
