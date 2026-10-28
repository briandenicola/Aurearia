package main

// Read-only API keys on the protected REST routes.
//
// Regression guard: an API key with only "read" capability could create,
// change and delete coins, and mint a new read+write key for itself through
// POST /api/auth/api-keys, because the protected group never checked
// capabilities. The guard is mounted on the real group here (buildDeps and
// registerProtectedRoutes, as main does), so a route added later is covered.

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/briandenicola/ancient-coins-api/config"
	"github.com/briandenicola/ancient-coins-api/database"
	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
)

func TestProtectedRoutes_ReadOnlyAPIKeyCannotChangeData(t *testing.T) {
	gin.SetMode(gin.TestMode)
	dir := t.TempDir()
	t.Setenv("JWT_SECRET", "protected-api-key-write-test-secret-0123456789")
	t.Setenv("UPLOAD_DIR", dir)
	cfg := config.Load()
	cfg.DBPath = filepath.Join(dir, "write-guard.db")
	database.Connect(cfg.DBPath)
	d, cancel := buildDeps(cfg)
	defer cancel()
	router := gin.New()
	registerProtectedRoutes(router.Group("/api"), d)

	db := database.DB
	owner := models.User{Username: "key-owner", Email: "key-owner@example.test", PasswordHash: "x", Role: models.RoleUser}
	if err := db.Create(&owner).Error; err != nil {
		t.Fatal(err)
	}
	newKey := func(plain, capabilities string) {
		t.Helper()
		key := models.ApiKey{UserID: owner.ID, KeyHash: services.HashAPIKey(plain, cfg.JWTSecret), KeyPrefix: "ak_test", Name: capabilities, Capabilities: capabilities}
		if err := db.Create(&key).Error; err != nil {
			t.Fatal(err)
		}
	}
	newKey("ak_read_only", "read")
	newKey("ak_read_write", "read,write")
	coin := models.Coin{UserID: owner.ID, Name: "Denarius of Trajan", Category: models.CategoryRoman, Denomination: "Denarius"}
	if err := db.Create(&coin).Error; err != nil {
		t.Fatal(err)
	}
	coinPath := "/api/coins/" + strconv.FormatUint(uint64(coin.ID), 10)

	call := func(key, method, path, body string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
		req.Header.Set("X-API-Key", key)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		return w
	}
	countKeys := func() int64 {
		var n int64
		db.Model(&models.ApiKey{}).Where("user_id = ?", owner.ID).Count(&n)
		return n
	}

	// A read key still reads.
	for _, path := range []string{"/api/coins", coinPath, "/api/auth/api-keys", "/api/lifestore/records"} {
		if w := call("ak_read_only", http.MethodGet, path, ""); w.Code != http.StatusOK {
			t.Errorf("read key GET %s = %d, want 200: %s", path, w.Code, w.Body.String())
		}
	}

	// A read key changes nothing.
	for _, tc := range []struct{ method, path, body string }{
		{http.MethodPost, "/api/coins", `{"name":"Forged","category":"Roman","denomination":"As"}`},
		{http.MethodPut, coinPath, `{"name":"Renamed","category":"Roman","denomination":"Denarius"}`},
		{http.MethodDelete, coinPath, ""},
		{http.MethodPost, "/api/auth/api-keys", `{"name":"escalated","capabilities":"read,write"}`},
	} {
		if w := call("ak_read_only", tc.method, tc.path, tc.body); w.Code != http.StatusForbidden {
			t.Errorf("read key %s %s = %d, want 403: %s", tc.method, tc.path, w.Code, w.Body.String())
		}
	}
	var stored models.Coin
	if err := db.First(&stored, coin.ID).Error; err != nil || stored.Name != "Denarius of Trajan" {
		t.Fatalf("coin after read-key writes = %+v, %v", stored, err)
	}
	if n := countKeys(); n != 2 {
		t.Fatalf("a read key minted a key: %d keys", n)
	}

	// A write key changes data, but no key manages keys: that is done signed
	// in, so a key can't mint another or raise its own access.
	for _, tc := range []struct{ method, path, body string }{
		{http.MethodPost, "/api/auth/api-keys", `{"name":"another","scope":"read,write,copilot"}`},
		{http.MethodPatch, "/api/auth/api-keys/2", `{"scope":"read,write,copilot"}`},
		{http.MethodDelete, "/api/auth/api-keys/1", ""},
	} {
		if w := call("ak_read_write", tc.method, tc.path, tc.body); w.Code != http.StatusForbidden {
			t.Errorf("write key %s %s = %d, want 403: %s", tc.method, tc.path, w.Code, w.Body.String())
		}
	}
	var keys []models.ApiKey
	db.Where("user_id = ?", owner.ID).Order("id").Find(&keys)
	if len(keys) != 2 || keys[1].Capabilities != "read,write" || keys[0].RevokedAt != nil {
		t.Fatalf("keys after a write key tried to manage them = %+v", keys)
	}
	if w := call("ak_read_write", http.MethodDelete, coinPath, ""); w.Code == http.StatusForbidden {
		t.Errorf("write key DELETE %s = 403: %s", coinPath, w.Body.String())
	}
}
