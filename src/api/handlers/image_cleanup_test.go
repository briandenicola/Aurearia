package handlers

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestImageCleanupHTTPPendingAndRetry(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "cleanup.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	t.Cleanup(func() { sqlDB.Close() })
	if err := db.AutoMigrate(&models.Coin{}, &models.CoinImage{}, &models.ImageCleanup{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.Coin{ID: 1, UserID: 1, Name: "Fixture"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.CoinImage{ID: 1, CoinID: 1, FilePath: "image.jpg"}).Error; err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	blocked := filepath.Join(dir, "image.jpg")
	if err := os.Mkdir(blocked, 0700); err != nil {
		t.Fatal(err)
	}
	repo := repository.NewImageRepository(db)
	h := NewImageHandler(dir, repo, services.NewImageService(repo, dir), services.NewLogger(10))
	router := gin.New()
	router.Use(func(c *gin.Context) { c.Set("userId", uint(1)) })
	router.DELETE("/coins/:id/images/:imageId", h.Delete)
	request := func() *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest("DELETE", "/coins/1/images/1", nil))
		return rec
	}
	if rec := request(); rec.Code != 202 || !strings.Contains(rec.Body.String(), `"cleanupPending":true`) {
		t.Fatalf("pending response: %d %s", rec.Code, rec.Body.String())
	}
	if err := os.Remove(blocked); err != nil {
		t.Fatal(err)
	}
	if rec := request(); rec.Code != 200 {
		t.Fatalf("retry: %d %s", rec.Code, rec.Body.String())
	}
	if rec := request(); rec.Code != 404 {
		t.Fatalf("completed: %d %s", rec.Code, rec.Body.String())
	}
}
