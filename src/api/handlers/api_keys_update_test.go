package handlers

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strconv"
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

// A signed-in owner changes an existing key's access in place: the key keeps
// working with its new capabilities. Another user's key, a revoked key and an
// unknown combination are refused.
func TestApiKeyUpdate_ChangesAccessInPlace(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.ApiKey{}); err != nil {
		t.Fatal(err)
	}
	owner := models.User{Username: "owner", Email: "owner@test.local", PasswordHash: "hash", Role: models.RoleUser}
	other := models.User{Username: "other", Email: "other@test.local", PasswordHash: "hash", Role: models.RoleUser}
	db.Create(&owner)
	db.Create(&other)
	repo := repository.NewApiKeyRepository(db)
	plain := "ak_sync_app"
	key := models.ApiKey{UserID: owner.ID, KeyHash: services.HashAPIKey(plain, adminAPIKeyAuthTestSecret), KeyPrefix: "ak_test", Name: "sync app", Capabilities: "read"}
	db.Create(&key)
	revokedAt := time.Now()
	revoked := models.ApiKey{UserID: owner.ID, KeyHash: "revoked-hash", KeyPrefix: "ak_old", Name: "old", Capabilities: "read", RevokedAt: &revokedAt}
	db.Create(&revoked)

	r := gin.New()
	group := r.Group("/api")
	group.Use(middleware.AuthRequired(adminAPIKeyAuthTestSecret, repo))
	h := NewApiKeyHandler(repo, services.NewLogger(10), adminAPIKeyAuthTestSecret)
	group.PATCH("/auth/api-keys/:id", middleware.RejectAPIKeyAuth(), h.Update)
	group.GET("/whoami", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"capabilities": c.GetString("apiKeyCapabilities")}) })

	patch := func(token string, id uint, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPatch, "/api/auth/api-keys/"+strconv.FormatUint(uint64(id), 10), bytes.NewBufferString(body))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w
	}
	ownerJWT := makeAdminAPIKeyAuthTestJWT(t, owner.ID, models.RoleUser)

	if w := patch(ownerJWT, key.ID, `{"scope":"read,write"}`); w.Code != http.StatusOK {
		t.Fatalf("owner PATCH = %d: %s", w.Code, w.Body.String())
	}
	// The same key now authenticates with write.
	req := httptest.NewRequest(http.MethodGet, "/api/whoami", nil)
	req.Header.Set("X-API-Key", plain)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK || !bytes.Contains(w.Body.Bytes(), []byte(`"read,write"`)) {
		t.Fatalf("key after the change = %d %s", w.Code, w.Body.String())
	}

	for _, tc := range []struct {
		name  string
		token string
		id    uint
		body  string
		want  int
	}{
		{"unknown combination", ownerJWT, key.ID, `{"scope":"admin"}`, http.StatusBadRequest},
		{"another user's key", makeAdminAPIKeyAuthTestJWT(t, other.ID, models.RoleUser), key.ID, `{"scope":"read"}`, http.StatusNotFound},
		{"revoked key", ownerJWT, revoked.ID, `{"scope":"read,write"}`, http.StatusNotFound},
	} {
		if w := patch(tc.token, tc.id, tc.body); w.Code != tc.want {
			t.Errorf("%s = %d, want %d: %s", tc.name, w.Code, tc.want, w.Body.String())
		}
	}
	var stored models.ApiKey
	db.First(&stored, key.ID)
	if stored.Capabilities != "read,write" {
		t.Fatalf("stored capabilities = %q", stored.Capabilities)
	}
}
