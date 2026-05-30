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

func setupAdminHealthHandlerTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	err = db.AutoMigrate(
		&models.User{},
		&models.Coin{},
		&models.CoinImage{},
		&models.CollectionHealthSnapshot{},
	)
	if err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	return db
}

func setupAdminHealthHandlerRouter(t *testing.T, isAdmin bool) (*gin.Engine, *gorm.DB, uint) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db := setupAdminHealthHandlerTestDB(t)

	role := models.RoleUser
	if isAdmin {
		role = models.RoleAdmin
	}
	user := models.User{Username: "admin", Email: "admin@test.com", Role: role}
	db.Create(&user)

	healthRepo := repository.NewHealthRepository(db)
	logger := services.NewLogger(100)
	healthSvc := services.NewHealthService(healthRepo, logger)
	handler := NewAdminHealthHandler(healthSvc, logger)

	r := gin.New()
	// Mock JWT middleware
	r.Use(func(c *gin.Context) {
		c.Set("userId", user.ID)
		c.Set("isAdmin", isAdmin)
		c.Next()
	})

	// Mock admin auth check
	adminGroup := r.Group("/api/admin")
	adminGroup.Use(func(c *gin.Context) {
		if !c.GetBool("isAdmin") {
			c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
			c.Abort()
			return
		}
		c.Next()
	})
	adminGroup.GET("/health/summary", handler.Summary)

	return r, db, user.ID
}

// --- T039: Admin endpoint auth and payload tests ---

func TestAdminHealthHandler_Summary_Success(t *testing.T) {
	router, db, userID := setupAdminHealthHandlerRouter(t, true)

	// Seed some coins
	for i := 0; i < 5; i++ {
		db.Create(&models.Coin{
			Name:     "Test Coin",
			Category: models.CategoryRoman,
			UserID:   userID,
		})
	}

	req := httptest.NewRequest(http.MethodGet, "/api/admin/health/summary", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp services.AdminHealthSummary
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	// Verify response structure
	if resp.LowScoreThreshold != services.HealthLowScoreThreshold {
		t.Errorf("expected lowScoreThreshold=%d, got %d", services.HealthLowScoreThreshold, resp.LowScoreThreshold)
	}
	if resp.TopMissingFields == nil {
		t.Error("expected topMissingFields to be initialized")
	}
	if resp.EligibleCoinCount != 5 {
		t.Errorf("expected eligibleCoinCount=5, got %d", resp.EligibleCoinCount)
	}
}

func TestAdminHealthHandler_Summary_EmptySystem(t *testing.T) {
	router, _, _ := setupAdminHealthHandlerRouter(t, true)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/health/summary", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp services.AdminHealthSummary
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp.MedianScore != 0 {
		t.Errorf("expected medianScore=0, got %d", resp.MedianScore)
	}
	if resp.LowScorePercentage != 0 {
		t.Errorf("expected lowScorePercentage=0, got %f", resp.LowScorePercentage)
	}
	if resp.EligibleCoinCount != 0 {
		t.Errorf("expected eligibleCoinCount=0, got %d", resp.EligibleCoinCount)
	}
}

func TestAdminHealthHandler_Summary_ForbiddenForNonAdmin(t *testing.T) {
	router, _, _ := setupAdminHealthHandlerRouter(t, false)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/health/summary", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["error"] == nil {
		t.Error("expected error message in response")
	}
}

func TestAdminHealthHandler_Summary_ResponseStructure(t *testing.T) {
	router, db, userID := setupAdminHealthHandlerRouter(t, true)

	// Create coins with varying metadata to generate missing field stats
	db.Create(&models.Coin{
		Name:         "Complete Coin",
		Category:     models.CategoryRoman,
		Denomination: "Denarius",
		Ruler:        "Augustus",
		Era:          "Imperial",
		Mint:         "Rome",
		Material:     models.MaterialSilver,
		Grade:        "VF",
		UserID:       userID,
	})
	db.Create(&models.Coin{
		Name:     "Incomplete Coin",
		Category: models.CategoryRoman,
		UserID:   userID,
	})

	req := httptest.NewRequest(http.MethodGet, "/api/admin/health/summary", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp services.AdminHealthSummary
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	// Verify all required fields are present
	if resp.MedianScore < 0 || resp.MedianScore > 100 {
		t.Errorf("medianScore out of range: %d", resp.MedianScore)
	}
	if resp.LowScorePercentage < 0 || resp.LowScorePercentage > 100 {
		t.Errorf("lowScorePercentage out of range: %f", resp.LowScorePercentage)
	}
	if resp.TopMissingFields == nil {
		t.Error("topMissingFields should be initialized (empty array or populated)")
	}
}

func TestAdminHealthHandler_Summary_MultipleUsers(t *testing.T) {
	router, db, _ := setupAdminHealthHandlerRouter(t, true)

	// Create multiple users with coins
	user1 := models.User{Username: "user1", Email: "user1@test.com"}
	user2 := models.User{Username: "user2", Email: "user2@test.com"}
	db.Create(&user1)
	db.Create(&user2)

	db.Create(&models.Coin{Name: "Coin U1-1", Category: models.CategoryRoman, UserID: user1.ID})
	db.Create(&models.Coin{Name: "Coin U1-2", Category: models.CategoryRoman, UserID: user1.ID})
	db.Create(&models.Coin{Name: "Coin U2-1", Category: models.CategoryGreek, UserID: user2.ID})

	req := httptest.NewRequest(http.MethodGet, "/api/admin/health/summary", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp services.AdminHealthSummary
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	// Should include coins from all users
	if resp.EligibleCoinCount != 3 {
		t.Errorf("expected eligibleCoinCount=3 across all users, got %d", resp.EligibleCoinCount)
	}
}

func TestAdminHealthHandler_Summary_ExcludesWishlistAndSold(t *testing.T) {
	router, db, userID := setupAdminHealthHandlerRouter(t, true)

	db.Create(&models.Coin{
		Name:       "Active Coin",
		Category:   models.CategoryRoman,
		UserID:     userID,
		IsWishlist: false,
		IsSold:     false,
	})
	db.Create(&models.Coin{
		Name:       "Wishlist Coin",
		Category:   models.CategoryRoman,
		UserID:     userID,
		IsWishlist: true,
		IsSold:     false,
	})
	db.Create(&models.Coin{
		Name:       "Sold Coin",
		Category:   models.CategoryRoman,
		UserID:     userID,
		IsWishlist: false,
		IsSold:     true,
	})

	req := httptest.NewRequest(http.MethodGet, "/api/admin/health/summary", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp services.AdminHealthSummary
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	// Should only count the active coin
	if resp.EligibleCoinCount != 1 {
		t.Errorf("expected eligibleCoinCount=1 (active only), got %d", resp.EligibleCoinCount)
	}
}
