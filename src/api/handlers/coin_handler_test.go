package handlers

import (
	"bytes"
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
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

const coinTestJWTSecret = "coin-handler-test-secret"

func setupCoinHandlerTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	err = db.AutoMigrate(
		&models.User{}, &models.Coin{}, &models.CoinImage{}, &models.CoinReference{}, &models.CatalogRegistry{},
		&models.ValueSnapshot{}, &models.CoinJournal{},
		&models.CoinValueHistory{}, &models.CoinComment{},
		&models.AvailabilityResult{}, &models.AuctionLot{},
		&models.Tag{}, &models.CoinTag{},
	)
	if err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	return db
}

func makeCoinTestJWT(userID uint) string {
	claims := jwt.MapClaims{
		"userId":   float64(userID),
		"username": "testuser",
		"role":     "user",
		"exp":      time.Now().Add(15 * time.Minute).Unix(),
		"iat":      time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, _ := token.SignedString([]byte(coinTestJWTSecret))
	return signed
}

// authMiddleware is a simplified version for testing that extracts userId from JWT
func coinTestAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		tokenString := authHeader[len("Bearer "):]
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return []byte(coinTestJWTSecret), nil
		})
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		claims := token.Claims.(jwt.MapClaims)
		c.Set("userId", uint(claims["userId"].(float64)))
		c.Set("userRole", claims["role"])
		c.Next()
	}
}

func setupCoinHandlerRouter(t *testing.T) (*gin.Engine, *gorm.DB) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db := setupCoinHandlerTestDB(t)
	coinRepo := repository.NewCoinRepository(db)
	catalogRegistryRepo := repository.NewCatalogRegistryRepository(db)
	coinSvc := services.NewCoinService(coinRepo, nil).WithCatalogRegistrySupport(catalogRegistryRepo)
	handler := NewCoinHandler(coinRepo, coinSvc, services.NewLogger(100))

	r := gin.New()
	protected := r.Group("/api")
	protected.Use(coinTestAuthMiddleware())
	protected.GET("/coins", handler.List)
	protected.GET("/coins/:id", handler.Get)
	protected.POST("/coins", handler.Create)
	protected.PUT("/coins/:id", handler.Update)
	protected.DELETE("/coins/:id", handler.Delete)

	return r, db
}

func createTestUser(t *testing.T, db *gorm.DB, id uint, username string) {
	t.Helper()
	user := models.User{ID: id, Username: username, PasswordHash: "hash", Email: username + "@test.com"}
	db.Create(&user)
}

func authHeader(userID uint) string {
	return "Bearer " + makeCoinTestJWT(userID)
}

// --- List ---

func TestCoinHandler_List_Authenticated(t *testing.T) {
	router, db := setupCoinHandlerRouter(t)
	createTestUser(t, db, 1, "listuser")

	// Create a coin for user 1
	coin := models.Coin{Name: "Test Denarius", Category: models.CategoryRoman, Material: models.MaterialSilver, UserID: 1}
	db.Create(&coin)

	req := httptest.NewRequest(http.MethodGet, "/api/coins", nil)
	req.Header.Set("Authorization", authHeader(1))
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	coins, ok := resp["coins"].([]interface{})
	if !ok {
		t.Fatal("expected coins array in response")
	}
	if len(coins) != 1 {
		t.Errorf("expected 1 coin, got %d", len(coins))
	}
}

func TestCoinHandler_List_OnlyOwnCoins(t *testing.T) {
	router, db := setupCoinHandlerRouter(t)
	createTestUser(t, db, 1, "user1")
	createTestUser(t, db, 2, "user2")

	db.Create(&models.Coin{Name: "User1 Coin", Category: models.CategoryRoman, Material: models.MaterialSilver, UserID: 1})
	db.Create(&models.Coin{Name: "User2 Coin", Category: models.CategoryGreek, Material: models.MaterialGold, UserID: 2})

	req := httptest.NewRequest(http.MethodGet, "/api/coins", nil)
	req.Header.Set("Authorization", authHeader(1))
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	coins := resp["coins"].([]interface{})
	if len(coins) != 1 {
		t.Errorf("expected 1 coin (own), got %d", len(coins))
	}
}

// --- Get ---

func TestCoinHandler_Get_OwnCoin(t *testing.T) {
	router, db := setupCoinHandlerRouter(t)
	createTestUser(t, db, 1, "getuser")

	coin := models.Coin{Name: "My Aureus", Category: models.CategoryRoman, Material: models.MaterialGold, UserID: 1}
	db.Create(&coin)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/coins/%d", coin.ID), nil)
	req.Header.Set("Authorization", authHeader(1))
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCoinHandler_Get_OtherUserCoin_NotFound(t *testing.T) {
	router, db := setupCoinHandlerRouter(t)
	createTestUser(t, db, 1, "owner")
	createTestUser(t, db, 2, "intruder")

	coin := models.Coin{Name: "Private Coin", Category: models.CategoryGreek, Material: models.MaterialSilver, UserID: 1}
	db.Create(&coin)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/coins/%d", coin.ID), nil)
	req.Header.Set("Authorization", authHeader(2)) // Different user
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for other user's coin, got %d", w.Code)
	}
}

// --- Create ---

func TestCoinHandler_Create_Success(t *testing.T) {
	router, db := setupCoinHandlerRouter(t)
	createTestUser(t, db, 1, "creator")

	coinData := map[string]interface{}{
		"name":     "New Tetradrachm",
		"category": "Greek",
		"material": "Silver",
	}
	body, _ := json.Marshal(coinData)

	req := httptest.NewRequest(http.MethodPost, "/api/coins", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authHeader(1))
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["name"] != "New Tetradrachm" {
		t.Errorf("expected name 'New Tetradrachm', got %v", resp["name"])
	}
	if resp["userId"] != float64(1) {
		t.Errorf("expected userId 1, got %v", resp["userId"])
	}
}

func TestCoinHandler_Create_InvalidPayload(t *testing.T) {
	router, db := setupCoinHandlerRouter(t)
	createTestUser(t, db, 1, "creator")

	req := httptest.NewRequest(http.MethodPost, "/api/coins", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authHeader(1))
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid JSON, got %d", w.Code)
	}
}

// --- Update ---

func TestCoinHandler_Update_OwnCoin(t *testing.T) {
	router, db := setupCoinHandlerRouter(t)
	createTestUser(t, db, 1, "updater")

	coin := models.Coin{Name: "Old Name", Category: models.CategoryRoman, Material: models.MaterialBronze, UserID: 1}
	db.Create(&coin)

	updates := map[string]interface{}{
		"name":     "Updated Name",
		"category": "Roman",
		"material": "Bronze",
	}

	body, _ := json.Marshal(updates)

	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/coins/%d", coin.ID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authHeader(1))
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCoinHandler_Update_CustomRegistryEraAccepted(t *testing.T) {
	router, db := setupCoinHandlerRouter(t)
	createTestUser(t, db, 1, "updater")

	if err := db.Create(&models.CatalogRegistry{
		Catalog:     "PROV",
		DisplayName: "Provincial References",
		Era:         models.Era("provincial"),
	}).Error; err != nil {
		t.Fatalf("failed to seed catalog registry: %v", err)
	}
	coin := models.Coin{Name: "Old Era", Category: models.CategoryRoman, Material: models.MaterialBronze, UserID: 1, Era: models.EraAncient}
	db.Create(&coin)

	updates := map[string]interface{}{
		"name":     "Updated Era",
		"category": "Roman",
		"material": "Bronze",
		"era":      "provincial",
	}
	body, _ := json.Marshal(updates)

	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/coins/%d", coin.ID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authHeader(1))
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for custom registry era, got %d: %s", w.Code, w.Body.String())
	}

	var found models.Coin
	if err := db.First(&found, coin.ID).Error; err != nil {
		t.Fatalf("coin not found: %v", err)
	}
	if found.Era != models.Era("provincial") {
		t.Fatalf("expected era provincial, got %q", found.Era)
	}
}

func TestCoinHandler_Update_OtherUserCoin_NotFound(t *testing.T) {
	router, db := setupCoinHandlerRouter(t)
	createTestUser(t, db, 1, "owner")
	createTestUser(t, db, 2, "attacker")

	coin := models.Coin{Name: "Owner Coin", Category: models.CategoryGreek, Material: models.MaterialGold, UserID: 1}
	db.Create(&coin)

	updates := map[string]interface{}{"name": "Stolen Name"}
	body, _ := json.Marshal(updates)

	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/coins/%d", coin.ID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authHeader(2))
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for other user's coin, got %d", w.Code)
	}
}

// --- Delete ---

func TestCoinHandler_Delete_OwnCoin(t *testing.T) {
	router, db := setupCoinHandlerRouter(t)
	createTestUser(t, db, 1, "deleter")

	coin := models.Coin{Name: "To Delete", Category: models.CategoryRoman, Material: models.MaterialBronze, UserID: 1}
	db.Create(&coin)

	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/coins/%d", coin.ID), nil)
	req.Header.Set("Authorization", authHeader(1))
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify coin is gone
	var count int64
	db.Model(&models.Coin{}).Where("id = ?", coin.ID).Count(&count)
	if count != 0 {
		t.Error("expected coin to be deleted from DB")
	}
}

func TestCoinHandler_Delete_OtherUserCoin_NotFound(t *testing.T) {
	router, db := setupCoinHandlerRouter(t)
	createTestUser(t, db, 1, "owner")
	createTestUser(t, db, 2, "attacker")

	coin := models.Coin{Name: "Protected Coin", Category: models.CategoryGreek, Material: models.MaterialSilver, UserID: 1}
	db.Create(&coin)

	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/coins/%d", coin.ID), nil)
	req.Header.Set("Authorization", authHeader(2))
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}

	// Verify coin still exists
	var count int64
	db.Model(&models.Coin{}).Where("id = ?", coin.ID).Count(&count)
	if count != 1 {
		t.Error("expected coin to still exist in DB")
	}
}

// --- Unauthenticated ---

func TestCoinHandler_Unauthenticated(t *testing.T) {
	router, _ := setupCoinHandlerRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/coins", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 without auth, got %d", w.Code)
	}
}
