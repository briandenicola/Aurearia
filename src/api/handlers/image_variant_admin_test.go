package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupImageVariantAdminRouter(t *testing.T) (*gin.Engine, *repository.ImageVariantRunRepository) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:image_variant_admin_%d?mode=memory&cache=shared", time.Now().UnixNano())), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.Coin{}, &models.CoinImage{}, &models.AppSetting{}, &models.ImageVariantRun{}); err != nil {
		t.Fatal(err)
	}
	runs := repository.NewImageVariantRunRepository(db)
	settingsSvc := services.NewSettingsService(repository.NewSettingsRepository(db))
	scheduler := services.NewImageVariantScheduler(repository.NewImageRepository(db), runs, t.TempDir(), settingsSvc, services.NewLogger(100))
	h := NewImageVariantAdminHandler(scheduler)
	r := gin.New()
	r.POST("/api/admin/image-variants/run", h.TriggerRun)
	r.GET("/api/admin/image-variants/status", h.GetStatus)
	r.GET("/api/admin/image-variant-runs", h.ListRuns)
	return r, runs
}

func serveImageVariantAdmin(r *gin.Engine, method, path string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(method, path, nil))
	return w
}

func TestImageVariantAdminRunNowStartsAPassAndRecordsIt(t *testing.T) {
	r, _ := setupImageVariantAdminRouter(t)
	w := serveImageVariantAdmin(r, http.MethodPost, "/api/admin/image-variants/run")
	if w.Code != http.StatusAccepted {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}

	var body struct {
		Runs  []models.ImageVariantRun `json:"runs"`
		Total int64                    `json:"total"`
	}
	deadline := time.Now().Add(10 * time.Second)
	for {
		w = serveImageVariantAdmin(r, http.MethodGet, "/api/admin/image-variant-runs?page=1&limit=5")
		if w.Code != http.StatusOK {
			t.Fatalf("list status=%d", w.Code)
		}
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if body.Total == 1 && body.Runs[0].Status != "running" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("no finished run recorded: %s", w.Body.String())
		}
		time.Sleep(10 * time.Millisecond)
	}
	if body.Runs[0].TriggerType != "manual" || body.Runs[0].Status != "success" {
		t.Fatalf("run=%+v", body.Runs[0])
	}
}

func TestImageVariantAdminStatus(t *testing.T) {
	r, _ := setupImageVariantAdminRouter(t)
	w := serveImageVariantAdmin(r, http.MethodGet, "/api/admin/image-variants/status")
	var status services.SchedulerStatus
	if w.Code != http.StatusOK || json.Unmarshal(w.Body.Bytes(), &status) != nil {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if status.Name != "Image Variants" || !status.Enabled || status.IsRunning || status.NextRunIn <= 0 {
		t.Fatalf("status=%+v", status)
	}
}
