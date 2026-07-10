package handlers

import (
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

func setupAvailabilityHandlerTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	if err := db.AutoMigrate(
		&models.User{},
		&models.Coin{},
		&models.CoinImage{},
		&models.AvailabilityRun{},
		&models.AvailabilityResult{},
		&models.AppSetting{},
	); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	return db
}

func setupAvailabilityAdminRouter(t *testing.T, listingURL string) (*gin.Engine, *gorm.DB, *services.AvailabilityScheduler, uint, uint) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db := setupAvailabilityHandlerTestDB(t)
	adminUser := models.User{Username: "admin", Email: "admin@test.com", Role: models.RoleAdmin}
	regularUser := models.User{Username: "user", Email: "user@test.com", Role: models.RoleUser}
	db.Create(&adminUser)
	db.Create(&regularUser)
	db.Create(&models.Coin{
		UserID:       regularUser.ID,
		Name:         "Wishlist Coin",
		ReferenceURL: listingURL,
		IsWishlist:   true,
	})

	coinRepo := repository.NewCoinRepository(db)
	availRepo := repository.NewAvailabilityRepository(db)
	settingsSvc := services.NewSettingsService(repository.NewSettingsRepository(db))
	logger := services.NewLogger(100)
	availSvc := services.NewAvailabilityService(coinRepo, availRepo, nil, nil, nil, nil, settingsSvc, logger)
	scheduler := services.NewAvailabilityScheduler(availSvc, coinRepo, availRepo, settingsSvc, logger)
	handler := NewAvailabilityHandler(nil, scheduler, availRepo, nil)

	router := gin.New()
	admin := router.Group("/api/admin")
	admin.Use(func(c *gin.Context) {
		switch c.GetHeader("Authorization") {
		case "Bearer admin-token":
			c.Set("userId", adminUser.ID)
			c.Set("userRole", string(models.RoleAdmin))
		case "Bearer user-token":
			c.Set("userId", regularUser.ID)
			c.Set("userRole", string(models.RoleUser))
		default:
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			c.Abort()
			return
		}
		c.Next()
	})
	admin.Use(AdminRequired())
	admin.POST("/availability/run", handler.TriggerRun)

	return router, db, scheduler, adminUser.ID, regularUser.ID
}

func TestAvailabilityHandler_TriggerRun_AsAdmin(t *testing.T) {
	listing := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<html><body><div>Sold</div></body></html>`))
	}))
	defer listing.Close()

	router, db, scheduler, adminID, _ := setupAvailabilityAdminRouter(t, listing.URL)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/availability/run", nil)
	req.Header.Set("Authorization", "Bearer admin-token")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	runIDFloat, ok := resp["runId"].(float64)
	if !ok || runIDFloat <= 0 {
		t.Fatalf("expected runId in response, got %+v", resp)
	}
	if resp["status"] != models.AvailabilityRunStatusQueued {
		t.Fatalf("expected status=queued, got %q", resp["status"])
	}

	runID := uint(runIDFloat)

	// Verify run is in DB as queued before worker processes it
	var queuedRun models.AvailabilityRun
	if err := db.First(&queuedRun, runID).Error; err != nil {
		t.Fatalf("expected availability run to be created: %v", err)
	}
	if queuedRun.TriggerType != "manual" {
		t.Fatalf("expected manual trigger, got %q", queuedRun.TriggerType)
	}
	if queuedRun.TriggerUserID == nil || *queuedRun.TriggerUserID != adminID {
		t.Fatalf("expected trigger user ID %d, got %v", adminID, queuedRun.TriggerUserID)
	}
	if queuedRun.Status != models.AvailabilityRunStatusQueued {
		t.Fatalf("expected queued status before processing, got %q", queuedRun.Status)
	}

	// Process the run synchronously (simulates what the worker does)
	if err := scheduler.ProcessRun(runID); err != nil {
		t.Fatalf("process run: %v", err)
	}

	// Verify run is completed with correct counts
	var completedRun models.AvailabilityRun
	if err := db.First(&completedRun, runID).Error; err != nil {
		t.Fatalf("expected run to persist: %v", err)
	}
	if completedRun.Status != models.AvailabilityRunStatusCompleted {
		t.Fatalf("expected status=completed, got %q", completedRun.Status)
	}
	if completedRun.Unavailable != 1 || completedRun.Unknown != 0 {
		t.Fatalf("expected sold listing to count unavailable=1 unknown=0, got unavailable=%d unknown=%d",
			completedRun.Unavailable, completedRun.Unknown)
	}
}

func TestAvailabilityHandler_TriggerRun_DuplicateBlocked(t *testing.T) {
	router, _, _, _, _ := setupAvailabilityAdminRouter(t, "https://example.test/coin")

	// First request should be accepted
	req := httptest.NewRequest(http.MethodPost, "/api/admin/availability/run", nil)
	req.Header.Set("Authorization", "Bearer admin-token")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusAccepted {
		t.Fatalf("first request: expected 202, got %d: %s", w.Code, w.Body.String())
	}

	// Second request while first is still queued should be rejected
	req2 := httptest.NewRequest(http.MethodPost, "/api/admin/availability/run", nil)
	req2.Header.Set("Authorization", "Bearer admin-token")
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)
	if w2.Code != http.StatusConflict {
		t.Fatalf("duplicate request: expected 409, got %d: %s", w2.Code, w2.Body.String())
	}
}

func TestAvailabilityHandler_TriggerRun_AsRegularUser(t *testing.T) {
	router, _, _, _, _ := setupAvailabilityAdminRouter(t, "https://example.test/coin")

	req := httptest.NewRequest(http.MethodPost, "/api/admin/availability/run", nil)
	req.Header.Set("Authorization", "Bearer user-token")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAvailabilityHandler_TriggerRun_NoAuth(t *testing.T) {
	router, _, _, _, _ := setupAvailabilityAdminRouter(t, "https://example.test/coin")

	req := httptest.NewRequest(http.MethodPost, "/api/admin/availability/run", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}
