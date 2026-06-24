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

func setupAvailabilityAdminRouter(t *testing.T, listingURL string) (*gin.Engine, *gorm.DB, uint, uint) {
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

	return router, db, adminUser.ID, regularUser.ID
}

func TestAvailabilityHandler_TriggerRun_AsAdmin(t *testing.T) {
	listing := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<html><body><div>Sold</div></body></html>`))
	}))
	defer listing.Close()

	router, db, adminID, regularID := setupAvailabilityAdminRouter(t, listing.URL)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/availability/run", nil)
	req.Header.Set("Authorization", "Bearer admin-token")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp["message"] != "Availability check run completed" {
		t.Fatalf("unexpected response message: %q", resp["message"])
	}

	var run models.AvailabilityRun
	if err := db.First(&run).Error; err != nil {
		t.Fatalf("expected availability run to be created: %v", err)
	}
	if run.UserID != regularID {
		t.Fatalf("expected run for wishlist owner %d, got %d", regularID, run.UserID)
	}
	if run.TriggerType != "manual" {
		t.Fatalf("expected manual trigger, got %q", run.TriggerType)
	}
	if run.TriggerUserID == nil || *run.TriggerUserID != adminID {
		t.Fatalf("expected trigger user ID %d, got %v", adminID, run.TriggerUserID)
	}
	if run.Unavailable != 1 || run.Unknown != 0 {
		t.Fatalf("expected sold listing to count unavailable=1 unknown=0, got unavailable=%d unknown=%d", run.Unavailable, run.Unknown)
	}
}

func TestAvailabilityHandler_TriggerRun_AsRegularUser(t *testing.T) {
	router, _, _, _ := setupAvailabilityAdminRouter(t, "https://example.test/coin")

	req := httptest.NewRequest(http.MethodPost, "/api/admin/availability/run", nil)
	req.Header.Set("Authorization", "Bearer user-token")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAvailabilityHandler_TriggerRun_NoAuth(t *testing.T) {
	router, _, _, _ := setupAvailabilityAdminRouter(t, "https://example.test/coin")

	req := httptest.NewRequest(http.MethodPost, "/api/admin/availability/run", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}
