package handlers

import (
	"encoding/json"
	"fmt"
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

func setupHealthHandlerTestDB(t *testing.T) *gorm.DB {
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

func setupHealthHandlerRouter(t *testing.T) (*gin.Engine, *gorm.DB, uint) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db := setupHealthHandlerTestDB(t)

	user := models.User{Username: "collector", Email: "collector@test.com"}
	db.Create(&user)

	healthRepo := repository.NewHealthRepository(db)
	logger := services.NewLogger(100)
	healthSvc := services.NewHealthService(healthRepo, logger)
	handler := NewHealthHandler(healthSvc, logger)

	r := gin.New()
	// Mock JWT middleware
	r.Use(func(c *gin.Context) {
		c.Set("userId", user.ID)
		c.Next()
	})

	protected := r.Group("/api")
	protected.GET("/stats/health", handler.CollectionSummary)
	protected.GET("/coins/health", handler.ListCoinHealth)

	return r, db, user.ID
}

// --- T014: Contract-style handler test for GET /api/stats/health ---

func TestHealthHandler_CollectionSummary_Success(t *testing.T) {
	router, db, userID := setupHealthHandlerRouter(t)

	// Seed active coins
	for i := 0; i < 3; i++ {
		db.Create(&models.Coin{
			Name:     "Test Coin",
			Category: models.CategoryRoman,
			UserID:   userID,
		})
	}

	req := httptest.NewRequest(http.MethodGet, "/api/stats/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp services.CollectionHealthSummary
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp.EligibleCoinCount != 3 {
		t.Errorf("expected eligibleCoinCount=3, got %d", resp.EligibleCoinCount)
	}
	if resp.Grade == "" {
		t.Error("expected grade to be set")
	}
	if resp.Weights.Metadata != 40 {
		t.Errorf("expected metadata weight=40, got %d", resp.Weights.Metadata)
	}
	if resp.Trend30D.Status == "" {
		t.Error("expected trend30d status to be set")
	}
}

func TestHealthHandler_CollectionSummary_EmptyCollection(t *testing.T) {
	router, _, _ := setupHealthHandlerRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/stats/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp services.CollectionHealthSummary
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp.EligibleCoinCount != 0 {
		t.Errorf("expected eligibleCoinCount=0, got %d", resp.EligibleCoinCount)
	}
	if resp.Score != 0 {
		t.Errorf("expected score=0, got %d", resp.Score)
	}
	if resp.Grade != services.HealthGradeF {
		t.Errorf("expected grade=F, got %s", resp.Grade)
	}
}

// --- T027: Contract-style handler test for GET /api/coins/health ---

func TestHealthHandler_ListCoinHealth_Success(t *testing.T) {
	router, db, userID := setupHealthHandlerRouter(t)

	for i := 0; i < 5; i++ {
		db.Create(&models.Coin{
			Name:     fmt.Sprintf("Coin %d", i),
			Category: models.CategoryRoman,
			UserID:   userID,
		})
	}

	req := httptest.NewRequest(http.MethodGet, "/api/coins/health?page=1&limit=25", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp services.CoinHealthListResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if len(resp.Coins) != 5 {
		t.Errorf("expected 5 coins, got %d", len(resp.Coins))
	}
	if resp.Pagination.Total != 5 {
		t.Errorf("expected total=5, got %d", resp.Pagination.Total)
	}
	if resp.Pagination.Page != 1 {
		t.Errorf("expected page=1, got %d", resp.Pagination.Page)
	}
	if resp.Pagination.Limit != 25 {
		t.Errorf("expected limit=25, got %d", resp.Pagination.Limit)
	}

	// Verify coin structure
	coin := resp.Coins[0]
	if coin.CoinID == 0 {
		t.Error("expected coinID to be set")
	}
	if coin.Title == "" {
		t.Error("expected title to be set")
	}
	if coin.Grade == "" {
		t.Error("expected grade to be set")
	}
	if coin.MissingItems == nil {
		t.Error("expected missingItems to be initialized")
	}
	if coin.QuickActions == nil {
		t.Error("expected quickActions to be initialized")
	}
}

func TestHealthHandler_ListCoinHealth_Pagination(t *testing.T) {
	router, db, userID := setupHealthHandlerRouter(t)

	for i := 0; i < 30; i++ {
		db.Create(&models.Coin{
			Name:     "Test Coin",
			Category: models.CategoryRoman,
			UserID:   userID,
		})
	}

	req := httptest.NewRequest(http.MethodGet, "/api/coins/health?page=2&limit=10", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp services.CoinHealthListResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if len(resp.Coins) != 10 {
		t.Errorf("expected 10 coins on page 2, got %d", len(resp.Coins))
	}
	if resp.Pagination.Page != 2 {
		t.Errorf("expected page=2, got %d", resp.Pagination.Page)
	}
	if resp.Pagination.Total != 30 {
		t.Errorf("expected total=30, got %d", resp.Pagination.Total)
	}
}

func TestHealthHandler_ListCoinHealth_InvalidPageParam(t *testing.T) {
	router, _, _ := setupHealthHandlerRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/coins/health?page=invalid", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["error"] == nil {
		t.Error("expected error message in response")
	}
}

func TestHealthHandler_ListCoinHealth_InvalidLimitParam(t *testing.T) {
	router, _, _ := setupHealthHandlerRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/coins/health?limit=invalid", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHealthHandler_ListCoinHealth_LimitOutOfBounds(t *testing.T) {
	router, _, _ := setupHealthHandlerRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/coins/health?limit=200", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHealthHandler_ListCoinHealth_InvalidScope(t *testing.T) {
	router, _, _ := setupHealthHandlerRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/coins/health?scope=invalid", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["error"] == nil {
		t.Error("expected error message in response")
	}
}

func TestHealthHandler_ListCoinHealth_ScopeAll(t *testing.T) {
	router, db, userID := setupHealthHandlerRouter(t)

	for i := 0; i < 3; i++ {
		db.Create(&models.Coin{
			Name:     "Test Coin",
			Category: models.CategoryRoman,
			UserID:   userID,
		})
	}

	req := httptest.NewRequest(http.MethodGet, "/api/coins/health?scope=all", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp services.CoinHealthListResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if len(resp.Coins) != 3 {
		t.Errorf("expected 3 coins with scope=all, got %d", len(resp.Coins))
	}
}

func TestHealthHandler_ListCoinHealth_ScopeNeedsAttention(t *testing.T) {
	router, db, userID := setupHealthHandlerRouter(t)

	for i := 0; i < 3; i++ {
		db.Create(&models.Coin{
			Name:     "Test Coin",
			Category: models.CategoryRoman,
			UserID:   userID,
		})
	}

	req := httptest.NewRequest(http.MethodGet, "/api/coins/health?scope=needs_attention", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp services.CoinHealthListResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if len(resp.Coins) != 3 {
		t.Errorf("expected 3 coins with scope=needs_attention, got %d", len(resp.Coins))
	}
}

func TestHealthHandler_ListCoinHealth_EmptyCollection(t *testing.T) {
	router, _, _ := setupHealthHandlerRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/coins/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp services.CoinHealthListResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if len(resp.Coins) != 0 {
		t.Errorf("expected 0 coins, got %d", len(resp.Coins))
	}
	if resp.Pagination.Total != 0 {
		t.Errorf("expected total=0, got %d", resp.Pagination.Total)
	}
}

func TestHealthHandler_ListCoinHealth_DefaultPaginationParams(t *testing.T) {
	router, db, userID := setupHealthHandlerRouter(t)

	for i := 0; i < 5; i++ {
		db.Create(&models.Coin{
			Name:     "Test Coin",
			Category: models.CategoryRoman,
			UserID:   userID,
		})
	}

	req := httptest.NewRequest(http.MethodGet, "/api/coins/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp services.CoinHealthListResponse
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp.Pagination.Page != 1 {
		t.Errorf("expected default page=1, got %d", resp.Pagination.Page)
	}
	if resp.Pagination.Limit != 25 {
		t.Errorf("expected default limit=25, got %d", resp.Pagination.Limit)
	}
}
