package handlers

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupImageMediaRouter(t *testing.T) (*gin.Engine, *gorm.DB, string) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:image_media_%d?mode=memory&cache=shared", time.Now().UnixNano())), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.Coin{}, &models.CoinImage{}, &models.Follow{}, &models.Showcase{}, &models.ShowcaseCoin{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	uploadDir := filepath.Join("testdata", fmt.Sprintf("media-auth-%d", time.Now().UnixNano()))
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		t.Fatalf("failed to create test upload dir: %v", err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(uploadDir); err != nil {
			t.Fatalf("failed to remove test upload dir: %v", err)
		}
	})

	imageRepo := repository.NewImageRepository(db)
	imageSvc := services.NewImageService(imageRepo, uploadDir)
	handler := NewImageHandler(uploadDir, imageRepo, imageSvc, services.NewLogger(100))

	r := gin.New()
	r.GET("/uploads/*filepath", coinTestAuthMiddleware(), handler.ServeUpload)
	return r, db, uploadDir
}

func createStoredCoinImage(t *testing.T, db *gorm.DB, uploadDir string, ownerID uint, private bool) string {
	t.Helper()

	owner := models.User{ID: ownerID, Username: fmt.Sprintf("owner-%d", ownerID), Email: fmt.Sprintf("owner-%d@example.com", ownerID), PasswordHash: "hash"}
	other := models.User{ID: 2, Username: "other", Email: "other@example.com", PasswordHash: "hash"}
	if err := db.Create(&owner).Error; err != nil {
		t.Fatalf("failed to create owner: %v", err)
	}
	if ownerID != 2 {
		if err := db.Create(&other).Error; err != nil {
			t.Fatalf("failed to create other user: %v", err)
		}
	}

	coin := models.Coin{
		Name:      "Private Denarius",
		Category:  models.CategoryRoman,
		Material:  models.MaterialSilver,
		UserID:    ownerID,
		IsPrivate: private,
	}
	if err := db.Create(&coin).Error; err != nil {
		t.Fatalf("failed to create coin: %v", err)
	}

	relPath := filepath.ToSlash(filepath.Join(fmt.Sprintf("coin-%d", coin.ID), "private.jpg"))
	fullPath := filepath.Join(uploadDir, filepath.FromSlash(relPath))
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		t.Fatalf("failed to create coin upload dir: %v", err)
	}
	if err := os.WriteFile(fullPath, []byte("owner-private-image"), 0644); err != nil {
		t.Fatalf("failed to write image file: %v", err)
	}

	image := models.CoinImage{CoinID: coin.ID, FilePath: relPath, ImageType: models.ImageTypeObverse}
	if err := db.Create(&image).Error; err != nil {
		t.Fatalf("failed to create image record: %v", err)
	}
	return relPath
}

func TestImageHandler_ServeUpload_AuthorizesPrivateCoinMedia(t *testing.T) {
	router, db, uploadDir := setupImageMediaRouter(t)
	relPath := createStoredCoinImage(t, db, uploadDir, 1, true)
	url := "/uploads/" + relPath

	t.Run("unauthenticated request is rejected", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, url, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("different user cannot fetch private media", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, url, nil)
		req.Header.Set("Authorization", authHeader(2))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("owner can fetch private media", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, url, nil)
		req.Header.Set("Authorization", authHeader(1))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
		if !strings.Contains(w.Body.String(), "owner-private-image") {
			t.Fatalf("expected image bytes in response, got %q", w.Body.String())
		}
	})
}

func TestImageHandler_ServeUpload_RejectsTraversal(t *testing.T) {
	router, db, uploadDir := setupImageMediaRouter(t)
	createStoredCoinImage(t, db, uploadDir, 1, true)

	req := httptest.NewRequest(http.MethodGet, "/uploads/coin-1/%2e%2e/coin-1/private.jpg", nil)
	req.Header.Set("Authorization", authHeader(1))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for traversal path, got %d: %s", w.Code, w.Body.String())
	}
}
