package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
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

var coinHandlerDBCounter atomic.Uint64

func setupCoinHandlerTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:coin_handler_%d_%d?mode=memory&cache=shared", time.Now().UnixNano(), coinHandlerDBCounter.Add(1))), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	err = db.AutoMigrate(
		&models.User{}, &models.StorageLocation{}, &models.Coin{}, &models.CoinImage{}, &models.CoinReference{}, &models.CatalogRegistry{}, &models.AppSetting{},
		&models.ValueSnapshot{}, &models.CoinJournal{}, &models.Note{},
		&models.CoinValueHistory{}, &models.CoinComment{},
		&models.AvailabilityResult{}, &models.AuctionLot{},
		&models.Tag{}, &models.CoinTag{},
		&models.CoinSet{}, &models.CoinSetMembership{},
		&models.Showcase{}, &models.ShowcaseCoin{},
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
	coinReferenceRepo := repository.NewCoinReferenceRepository(db)
	coinReferenceSvc := services.NewCoinReferenceService(coinReferenceRepo, catalogRegistryRepo)
	storageLocationRepo := repository.NewStorageLocationRepository(db)
	settingsRepo := repository.NewSettingsRepository(db)
	settingsSvc := services.NewSettingsService(settingsRepo)
	coinSvc := services.NewCoinService(coinRepo, nil).
		WithReferenceSupport(coinReferenceRepo, coinReferenceSvc).
		WithStorageLocationSupport(storageLocationRepo).
		WithCatalogRegistrySupport(catalogRegistryRepo).
		WithSettingsSupport(settingsSvc)
	handler := NewCoinHandler(coinRepo, coinSvc, services.NewLogger(100))

	r := gin.New()
	protected := r.Group("/api")
	protected.Use(coinTestAuthMiddleware())
	protected.GET("/coins", handler.List)
	protected.GET("/coins/:id", handler.Get)
	protected.GET("/stats/investment-breakdown", handler.InvestmentBreakdown)
	protected.POST("/coins", handler.Create)
	protected.PUT("/coins/:id", handler.Update)
	protected.POST("/coins/:id/duplicate", handler.Duplicate)
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

// --- Duplicate ---

func TestCoinHandler_Duplicate_PreservesOwnedDataAndAssociationsExcludingMedia(t *testing.T) {
	router, db := setupCoinHandlerRouter(t)
	createTestUser(t, db, 1, "owner")

	location := models.StorageLocation{UserID: 1, Name: "Tray A"}
	if err := db.Create(&location).Error; err != nil {
		t.Fatalf("failed to seed storage location: %v", err)
	}
	purchasePrice := 125.50
	currentValue := 175.75
	purchaseDate := time.Date(2025, time.March, 14, 0, 0, 0, 0, time.UTC)
	coin := models.Coin{
		Name:               "Augustus Denarius",
		Category:           models.CategoryRoman,
		Denomination:       "Denarius",
		Ruler:              "Augustus",
		Era:                models.EraAncient,
		Mint:               "Rome",
		Material:           models.MaterialSilver,
		PurchasePrice:      &purchasePrice,
		CurrentValue:       &currentValue,
		PurchaseDate:       &purchaseDate,
		PurchaseLocation:   "Auction",
		Notes:              "Cabinet tone",
		ReferenceURL:       "https://example.test/reference",
		ReferenceText:      "RIC I 1",
		IsPrivate:          true,
		StorageLocationID:  &location.ID,
		UserID:             1,
		ObverseInscription: "CAESAR",
		ReverseInscription: "AVGVSTVS",
		ObverseDescription: "Laureate head",
		ReverseDescription: "Temple",
		AIAnalysis:         "Existing analysis",
		ObverseAnalysis:    "Obverse analysis",
		ReverseAnalysis:    "Reverse analysis",
		ListingStatus:      "available",
		ListingCheckReason: "seeded",
		ListingCheckedAt:   &purchaseDate,
	}
	if err := db.Create(&coin).Error; err != nil {
		t.Fatalf("failed to seed coin: %v", err)
	}
	if err := db.Create(&models.CoinImage{CoinID: coin.ID, FilePath: "uploads/original-obverse.jpg", ImageType: models.ImageTypeObverse, IsPrimary: true}).Error; err != nil {
		t.Fatalf("failed to seed image: %v", err)
	}
	if err := db.Create(&models.CoinReference{CoinID: coin.ID, Catalog: "RIC", Volume: "I", Number: "12", InvoiceNumber: "INV-1", URI: "https://example.test/ric"}).Error; err != nil {
		t.Fatalf("failed to seed reference: %v", err)
	}
	tag := models.Tag{UserID: 1, Name: "Favorites", Color: "#c9a84c"}
	if err := db.Create(&tag).Error; err != nil {
		t.Fatalf("failed to seed tag: %v", err)
	}
	if err := repository.NewTagRepository(db).AttachToCoin(coin.ID, tag.ID, 1); err != nil {
		t.Fatalf("failed to attach tag: %v", err)
	}
	setRepo := repository.NewSetRepository(db)
	set := models.CoinSet{UserID: 1, Name: "Roman Core", SetType: models.CoinSetTypeOpen}
	if err := setRepo.Create(&set); err != nil {
		t.Fatalf("failed to seed set: %v", err)
	}
	if err := setRepo.AddCoinToSet(coin.ID, set.ID, 1, "keeper"); err != nil {
		t.Fatalf("failed to attach set: %v", err)
	}
	if err := db.Model(&models.CoinSetMembership{}).Where("coin_id = ? AND set_id = ?", coin.ID, set.ID).Update("sort_order", 7).Error; err != nil {
		t.Fatalf("failed to set membership sort order: %v", err)
	}
	var originalMembership models.CoinSetMembership
	if err := db.Where("coin_id = ? AND set_id = ?", coin.ID, set.ID).First(&originalMembership).Error; err != nil {
		t.Fatalf("failed to load original membership: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/coins/%d/duplicate", coin.ID), nil)
	req.Header.Set("Authorization", authHeader(1))
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var duplicated models.Coin
	if err := json.Unmarshal(w.Body.Bytes(), &duplicated); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if duplicated.ID == 0 || duplicated.ID == coin.ID {
		t.Fatalf("expected a new coin id, got %d from original %d", duplicated.ID, coin.ID)
	}
	if duplicated.Name != "Augustus Denarius (duplicate)" {
		t.Fatalf("expected duplicate suffix, got %q", duplicated.Name)
	}
	if duplicated.UserID != 1 || duplicated.Category != coin.Category || duplicated.Material != coin.Material || duplicated.Era != coin.Era {
		t.Fatalf("expected owned scalar data to be preserved, got user=%d category=%q material=%q era=%q", duplicated.UserID, duplicated.Category, duplicated.Material, duplicated.Era)
	}
	if duplicated.StorageLocationID == nil || *duplicated.StorageLocationID != location.ID {
		t.Fatalf("expected storage location %d to be preserved, got %v", location.ID, duplicated.StorageLocationID)
	}
	if len(duplicated.Images) != 0 {
		t.Fatalf("expected duplicate response to exclude images/media, got %#v", duplicated.Images)
	}
	if len(duplicated.Tags) != 1 || duplicated.Tags[0].ID != tag.ID {
		t.Fatalf("expected tag association to be preserved, got %#v", duplicated.Tags)
	}
	if len(duplicated.Sets) != 1 || duplicated.Sets[0].ID != set.ID {
		t.Fatalf("expected set association to be preserved, got %#v", duplicated.Sets)
	}
	if len(duplicated.References) != 1 || duplicated.References[0].Catalog != "RIC" || duplicated.References[0].Number != "12" {
		t.Fatalf("expected reference association to be preserved, got %#v", duplicated.References)
	}

	var newImageCount int64
	if err := db.Model(&models.CoinImage{}).Where("coin_id = ?", duplicated.ID).Count(&newImageCount).Error; err != nil {
		t.Fatalf("failed to count duplicated images: %v", err)
	}
	if newImageCount != 0 {
		t.Fatalf("expected no image rows for duplicated coin, got %d", newImageCount)
	}
	var membership models.CoinSetMembership
	if err := db.Where("coin_id = ? AND set_id = ?", duplicated.ID, set.ID).First(&membership).Error; err != nil {
		t.Fatalf("expected duplicated set membership: %v", err)
	}
	if membership.Notes != originalMembership.Notes || membership.SortOrder != originalMembership.SortOrder || !membership.AddedAt.Equal(originalMembership.AddedAt) {
		t.Fatalf("expected membership metadata to be preserved, got %#v want %#v", membership, originalMembership)
	}
	var snapshotCount int64
	if err := db.Model(&models.ValueSnapshot{}).Where("user_id = ?", uint(1)).Count(&snapshotCount).Error; err != nil {
		t.Fatalf("failed to count value snapshots: %v", err)
	}
	if snapshotCount != 1 {
		t.Fatalf("expected duplicate to record one value snapshot, got %d", snapshotCount)
	}
}

func TestCoinHandler_Duplicate_RejectsNonOwnedCoin(t *testing.T) {
	router, db := setupCoinHandlerRouter(t)
	createTestUser(t, db, 1, "owner")
	createTestUser(t, db, 2, "intruder")

	coin := models.Coin{Name: "Private Coin", Category: models.CategoryGreek, Material: models.MaterialSilver, UserID: 1}
	if err := db.Create(&coin).Error; err != nil {
		t.Fatalf("failed to seed coin: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/coins/%d/duplicate", coin.ID), nil)
	req.Header.Set("Authorization", authHeader(2))
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for non-owned coin, got %d: %s", w.Code, w.Body.String())
	}
	var duplicateCount int64
	if err := db.Model(&models.Coin{}).Where("name = ? AND user_id = ?", "Private Coin (duplicate)", uint(2)).Count(&duplicateCount).Error; err != nil {
		t.Fatalf("failed to count duplicates: %v", err)
	}
	if duplicateCount != 0 {
		t.Fatalf("expected no duplicate for non-owner, got %d", duplicateCount)
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

func TestCoinHandler_Update_OneFieldPreservesAssociationsAndReadOnlyFields(t *testing.T) {
	router, db := setupCoinHandlerRouter(t)
	createTestUser(t, db, 1, "updater")
	createTestUser(t, db, 2, "other")

	location := models.StorageLocation{UserID: 1, Name: "Tray A"}
	if err := db.Create(&location).Error; err != nil {
		t.Fatalf("failed to seed storage location: %v", err)
	}
	currentValue := 250.0
	valuationTime := time.Now().Add(-2 * time.Hour).UTC().Truncate(time.Second)
	coin := models.Coin{
		Name:                  "Original Denarius",
		Category:              models.CategoryRoman,
		Material:              models.MaterialSilver,
		Era:                   models.EraAncient,
		UserID:                1,
		CurrentValue:          &currentValue,
		CurrentValueUpdatedAt: &valuationTime,
		AIAnalysis:            "existing analysis",
		StorageLocationID:     &location.ID,
	}
	if err := db.Create(&coin).Error; err != nil {
		t.Fatalf("failed to seed coin: %v", err)
	}
	originalID := coin.ID
	originalCreatedAt := coin.CreatedAt

	if err := db.Create(&models.CoinImage{CoinID: coin.ID, FilePath: "coins/original.jpg", ImageType: models.ImageTypeObverse, IsPrimary: true}).Error; err != nil {
		t.Fatalf("failed to seed image: %v", err)
	}
	if err := db.Create(&models.CoinReference{CoinID: coin.ID, Catalog: "RIC", Volume: "II", Number: "12"}).Error; err != nil {
		t.Fatalf("failed to seed reference: %v", err)
	}
	tag := models.Tag{UserID: 1, Name: "Favorites", Color: "#c9a84c"}
	if err := db.Create(&tag).Error; err != nil {
		t.Fatalf("failed to seed tag: %v", err)
	}
	if err := repository.NewTagRepository(db).AttachToCoin(coin.ID, tag.ID, 1); err != nil {
		t.Fatalf("failed to attach tag: %v", err)
	}
	set := models.CoinSet{UserID: 1, Name: "Roman Core", SetType: models.CoinSetTypeOpen}
	if err := repository.NewSetRepository(db).Create(&set); err != nil {
		t.Fatalf("failed to seed set: %v", err)
	}
	if err := repository.NewSetRepository(db).AddCoinToSet(coin.ID, set.ID, 1, "keeper"); err != nil {
		t.Fatalf("failed to attach set: %v", err)
	}

	updates := map[string]interface{}{
		"id":                    originalID + 100,
		"userId":                2,
		"name":                  "Renamed Denarius",
		"createdAt":             time.Now().Add(-24 * time.Hour),
		"aiAnalysis":            "incoming analysis",
		"currentValueUpdatedAt": time.Now(),
		"images": []map[string]interface{}{
			{"id": 999, "coinId": coin.ID, "filePath": "coins/replacement.jpg", "imageType": "reverse"},
		},
		"tags": []map[string]interface{}{
			{"id": tag.ID + 100, "name": "Incoming"},
		},
		"sets": []map[string]interface{}{
			{"id": set.ID + 100, "name": "Incoming Set"},
		},
		"storageLocation": map[string]interface{}{"id": location.ID + 100, "name": "Incoming Location"},
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

	var found models.Coin
	if err := db.Preload("Images").Preload("References").Preload("Tags").Preload("Sets").First(&found, originalID).Error; err != nil {
		t.Fatalf("updated coin not found: %v", err)
	}
	if found.ID != originalID {
		t.Fatalf("expected read-only id %d to be preserved, got %d", originalID, found.ID)
	}
	if found.UserID != 1 {
		t.Fatalf("expected read-only userId 1 to be preserved, got %d", found.UserID)
	}
	if !found.CreatedAt.Equal(originalCreatedAt) {
		t.Fatalf("expected read-only createdAt to be preserved, got %v want %v", found.CreatedAt, originalCreatedAt)
	}
	if found.Name != "Renamed Denarius" {
		t.Fatalf("expected name update, got %q", found.Name)
	}
	if found.Category != models.CategoryRoman || found.Material != models.MaterialSilver || found.Era != models.EraAncient {
		t.Fatalf("unexpected scalar sibling mutation: category=%q material=%q era=%q", found.Category, found.Material, found.Era)
	}
	if found.AIAnalysis != "existing analysis" {
		t.Fatalf("expected read-only aiAnalysis to be preserved, got %q", found.AIAnalysis)
	}
	if found.CurrentValueUpdatedAt == nil || !found.CurrentValueUpdatedAt.Equal(valuationTime) {
		t.Fatalf("expected currentValueUpdatedAt to be preserved, got %v want %v", found.CurrentValueUpdatedAt, valuationTime)
	}
	if found.StorageLocationID == nil || *found.StorageLocationID != location.ID {
		t.Fatalf("expected storage location %d to remain, got %v", location.ID, found.StorageLocationID)
	}
	if len(found.Images) != 1 || found.Images[0].FilePath != "coins/original.jpg" {
		t.Fatalf("expected original image association to remain, got %#v", found.Images)
	}
	if len(found.References) != 1 || found.References[0].Number != "12" {
		t.Fatalf("expected original reference association to remain, got %#v", found.References)
	}
	if len(found.Tags) != 1 || found.Tags[0].ID != tag.ID {
		t.Fatalf("expected original tag association to remain, got %#v", found.Tags)
	}
	if len(found.Sets) != 1 || found.Sets[0].ID != set.ID {
		t.Fatalf("expected original set association to remain, got %#v", found.Sets)
	}

	var valueHistoryCount int64
	if err := db.Model(&models.CoinValueHistory{}).Where("coin_id = ?", coin.ID).Count(&valueHistoryCount).Error; err != nil {
		t.Fatalf("failed to count value history: %v", err)
	}
	if valueHistoryCount != 0 {
		t.Fatalf("expected no manual value history for name-only edit, got %d", valueHistoryCount)
	}
	var snapshotCount int64
	if err := db.Model(&models.ValueSnapshot{}).Where("user_id = ?", uint(1)).Count(&snapshotCount).Error; err != nil {
		t.Fatalf("failed to count value snapshots: %v", err)
	}
	if snapshotCount != 1 {
		t.Fatalf("expected one value snapshot for update, got %d", snapshotCount)
	}
}

func TestCoinHandler_Update_IgnoresUnknownReadOnlyAndBroadRelationshipFields(t *testing.T) {
	router, db := setupCoinHandlerRouter(t)
	createTestUser(t, db, 1, "updater")
	createTestUser(t, db, 2, "other")

	coin := models.Coin{Name: "Typed Contract Coin", Category: models.CategoryRoman, Material: models.MaterialSilver, UserID: 1}
	if err := db.Create(&coin).Error; err != nil {
		t.Fatalf("failed to seed coin: %v", err)
	}
	if err := db.Create(&models.CoinImage{CoinID: coin.ID, FilePath: "coins/original.jpg", ImageType: models.ImageTypeObverse}).Error; err != nil {
		t.Fatalf("failed to seed image: %v", err)
	}
	if err := db.Create(&models.CoinReference{CoinID: coin.ID, Catalog: "RIC", Number: "1"}).Error; err != nil {
		t.Fatalf("failed to seed reference: %v", err)
	}
	tag := models.Tag{UserID: 1, Name: "Original Tag"}
	if err := db.Create(&tag).Error; err != nil {
		t.Fatalf("failed to seed tag: %v", err)
	}
	if err := repository.NewTagRepository(db).AttachToCoin(coin.ID, tag.ID, 1); err != nil {
		t.Fatalf("failed to attach tag: %v", err)
	}
	set := models.CoinSet{UserID: 1, Name: "Original Set", SetType: models.CoinSetTypeOpen}
	if err := repository.NewSetRepository(db).Create(&set); err != nil {
		t.Fatalf("failed to seed set: %v", err)
	}
	if err := repository.NewSetRepository(db).AddCoinToSet(coin.ID, set.ID, 1, "original"); err != nil {
		t.Fatalf("failed to attach set: %v", err)
	}

	updates := map[string]interface{}{
		"name":       "Typed Contract Coin Updated",
		"unknownKey": "ignored",
		"userId":     2,
		"storageLocation": map[string]interface{}{
			"id":   99,
			"name": "Injected Location",
		},
		"images": []map[string]interface{}{
			{"filePath": "coins/injected.jpg", "imageType": "reverse"},
		},
		"tags": []map[string]interface{}{
			{"id": tag.ID + 100, "name": "Injected Tag"},
		},
		"sets": []map[string]interface{}{
			{"id": set.ID + 100, "name": "Injected Set"},
		},
	}
	body, _ := json.Marshal(updates)

	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/coins/%d", coin.ID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authHeader(1))
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 with ignored unknown/broad fields, got %d: %s", w.Code, w.Body.String())
	}

	var found models.Coin
	if err := db.Preload("Images").Preload("References").Preload("Tags").Preload("Sets").First(&found, coin.ID).Error; err != nil {
		t.Fatalf("updated coin not found: %v", err)
	}
	if found.Name != "Typed Contract Coin Updated" {
		t.Fatalf("expected allowlisted name update, got %q", found.Name)
	}
	if found.UserID != 1 {
		t.Fatalf("expected read-only userId to remain 1, got %d", found.UserID)
	}
	if len(found.Images) != 1 || found.Images[0].FilePath != "coins/original.jpg" {
		t.Fatalf("expected image relationship payload to be ignored, got %#v", found.Images)
	}
	if len(found.References) != 1 || found.References[0].Number != "1" {
		t.Fatalf("expected omitted references to be preserved, got %#v", found.References)
	}
	if len(found.Tags) != 1 || found.Tags[0].ID != tag.ID {
		t.Fatalf("expected tag relationship payload to be ignored, got %#v", found.Tags)
	}
	if len(found.Sets) != 1 || found.Sets[0].ID != set.ID {
		t.Fatalf("expected set relationship payload to be ignored, got %#v", found.Sets)
	}
}

func TestCoinHandler_Update_WithSetsPayloadPreservesMemberships(t *testing.T) {
	router, db := setupCoinHandlerRouter(t)
	createTestUser(t, db, 1, "updater")

	setRepo := repository.NewSetRepository(db)
	coin := models.Coin{Name: "Old Name", Category: models.CategoryRoman, Material: models.MaterialSilver, UserID: 1}
	if err := db.Create(&coin).Error; err != nil {
		t.Fatalf("failed to seed coin: %v", err)
	}

	originalSet := models.CoinSet{UserID: 1, Name: "Original Set", SetType: models.CoinSetTypeOpen}
	incomingSet := models.CoinSet{UserID: 1, Name: "Incoming Set", SetType: models.CoinSetTypeOpen}
	if err := setRepo.Create(&originalSet); err != nil {
		t.Fatalf("failed to seed original set: %v", err)
	}
	if err := setRepo.Create(&incomingSet); err != nil {
		t.Fatalf("failed to seed incoming set: %v", err)
	}
	if err := setRepo.AddCoinToSet(coin.ID, originalSet.ID, 1, ""); err != nil {
		t.Fatalf("failed to seed membership: %v", err)
	}

	var originalMembership models.CoinSetMembership
	if err := db.Where("coin_id = ? AND set_id = ?", coin.ID, originalSet.ID).First(&originalMembership).Error; err != nil {
		t.Fatalf("failed to find seeded membership: %v", err)
	}
	if originalMembership.AddedAt.IsZero() {
		t.Fatal("seeded membership should have AddedAt")
	}

	updates := map[string]interface{}{
		"name":     "Updated Name",
		"category": "Roman",
		"material": "Silver",
		"sets": []map[string]interface{}{
			{
				"id":      incomingSet.ID,
				"userId":  1,
				"name":    incomingSet.Name,
				"setType": string(models.CoinSetTypeOpen),
			},
		},
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

	var updatedCoin models.Coin
	if err := db.First(&updatedCoin, coin.ID).Error; err != nil {
		t.Fatalf("updated coin not found: %v", err)
	}
	if updatedCoin.Name != "Updated Name" {
		t.Fatalf("expected updated coin name, got %q", updatedCoin.Name)
	}

	var memberships []models.CoinSetMembership
	if err := db.Where("coin_id = ?", coin.ID).Order("set_id ASC").Find(&memberships).Error; err != nil {
		t.Fatalf("failed to query memberships: %v", err)
	}
	if len(memberships) != 1 {
		t.Fatalf("expected update to preserve exactly 1 membership, got %d", len(memberships))
	}
	if memberships[0].SetID != originalSet.ID {
		t.Fatalf("expected original set membership to remain, got set ID %d", memberships[0].SetID)
	}
	if memberships[0].AddedAt.IsZero() {
		t.Fatal("membership AddedAt should remain populated after coin update")
	}
	if !memberships[0].AddedAt.Equal(originalMembership.AddedAt) {
		t.Fatalf("membership AddedAt changed from %v to %v", originalMembership.AddedAt, memberships[0].AddedAt)
	}
}

func TestCoinHandler_Update_StorageLocationWithSetsPreservesMemberships(t *testing.T) {
	router, db := setupCoinHandlerRouter(t)
	createTestUser(t, db, 1, "updater")

	setRepo := repository.NewSetRepository(db)
	coin := models.Coin{Name: "Stored Coin", Category: models.CategoryRoman, Material: models.MaterialSilver, UserID: 1}
	if err := db.Create(&coin).Error; err != nil {
		t.Fatalf("failed to seed coin: %v", err)
	}
	set := models.CoinSet{UserID: 1, Name: "Storage Test Set", SetType: models.CoinSetTypeOpen}
	if err := setRepo.Create(&set); err != nil {
		t.Fatalf("failed to seed set: %v", err)
	}
	if err := setRepo.AddCoinToSet(coin.ID, set.ID, 1, ""); err != nil {
		t.Fatalf("failed to seed membership: %v", err)
	}
	location := models.StorageLocation{UserID: 1, Name: "Tray 1"}
	if err := db.Create(&location).Error; err != nil {
		t.Fatalf("failed to seed storage location: %v", err)
	}

	var originalMembership models.CoinSetMembership
	if err := db.Where("coin_id = ? AND set_id = ?", coin.ID, set.ID).First(&originalMembership).Error; err != nil {
		t.Fatalf("failed to find seeded membership: %v", err)
	}

	updates := map[string]interface{}{
		"name":              "Stored Coin",
		"category":          "Roman",
		"material":          "Silver",
		"storageLocationId": location.ID,
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

	var updatedCoin models.Coin
	if err := db.First(&updatedCoin, coin.ID).Error; err != nil {
		t.Fatalf("updated coin not found: %v", err)
	}
	if updatedCoin.StorageLocationID == nil || *updatedCoin.StorageLocationID != location.ID {
		t.Fatalf("expected storage location %d, got %v", location.ID, updatedCoin.StorageLocationID)
	}

	var memberships []models.CoinSetMembership
	if err := db.Where("coin_id = ?", coin.ID).Find(&memberships).Error; err != nil {
		t.Fatalf("failed to query memberships: %v", err)
	}
	if len(memberships) != 1 {
		t.Fatalf("expected update to preserve exactly 1 membership, got %d", len(memberships))
	}
	if memberships[0].SetID != set.ID {
		t.Fatalf("expected original set membership to remain, got set ID %d", memberships[0].SetID)
	}
	if memberships[0].AddedAt.IsZero() {
		t.Fatal("membership AddedAt should remain populated after storage location update")
	}
	if !memberships[0].AddedAt.Equal(originalMembership.AddedAt) {
		t.Fatalf("membership AddedAt changed from %v to %v", originalMembership.AddedAt, memberships[0].AddedAt)
	}
}

func TestCoinHandler_Update_ClearsStorageLocationWhenExplicitNull(t *testing.T) {
	router, db := setupCoinHandlerRouter(t)
	createTestUser(t, db, 1, "updater")

	location := models.StorageLocation{UserID: 1, Name: "Cabinet A"}
	if err := db.Create(&location).Error; err != nil {
		t.Fatalf("failed to seed storage location: %v", err)
	}
	coin := models.Coin{
		Name:              "Stored Coin",
		Category:          models.CategoryRoman,
		Material:          models.MaterialSilver,
		UserID:            1,
		StorageLocationID: &location.ID,
	}
	if err := db.Create(&coin).Error; err != nil {
		t.Fatalf("failed to seed coin: %v", err)
	}

	updates := map[string]interface{}{
		"name":              "Stored Coin",
		"category":          "Roman",
		"material":          "Silver",
		"storageLocationId": nil,
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

	var updatedCoin models.Coin
	if err := db.First(&updatedCoin, coin.ID).Error; err != nil {
		t.Fatalf("updated coin not found: %v", err)
	}
	if updatedCoin.StorageLocationID != nil {
		t.Fatalf("expected storage location to be cleared, got %v", updatedCoin.StorageLocationID)
	}
}

func TestCoinHandler_Update_RejectsNonOwnedStorageLocation(t *testing.T) {
	router, db := setupCoinHandlerRouter(t)
	createTestUser(t, db, 1, "updater")
	createTestUser(t, db, 2, "other")

	otherLocation := models.StorageLocation{UserID: 2, Name: "Other Cabinet"}
	if err := db.Create(&otherLocation).Error; err != nil {
		t.Fatalf("failed to seed other storage location: %v", err)
	}
	coin := models.Coin{Name: "Stored Coin", Category: models.CategoryRoman, Material: models.MaterialSilver, UserID: 1}
	if err := db.Create(&coin).Error; err != nil {
		t.Fatalf("failed to seed coin: %v", err)
	}

	body, _ := json.Marshal(map[string]interface{}{"storageLocationId": otherLocation.ID})
	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/coins/%d", coin.ID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authHeader(1))
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for non-owned storage location, got %d: %s", w.Code, w.Body.String())
	}

	var found models.Coin
	if err := db.First(&found, coin.ID).Error; err != nil {
		t.Fatalf("coin not found: %v", err)
	}
	if found.StorageLocationID != nil {
		t.Fatalf("expected storage location to remain unset after rejected update, got %v", found.StorageLocationID)
	}
}

func TestCoinHandler_Update_PersistsExplicitFalseAndEmptyStringClears(t *testing.T) {
	router, db := setupCoinHandlerRouter(t)
	createTestUser(t, db, 1, "updater")

	coin := models.Coin{
		Name:               "Clearable Coin",
		Category:           models.CategoryRoman,
		Material:           models.MaterialSilver,
		UserID:             1,
		Notes:              "clear me",
		ReferenceURL:       "https://example.test/ref",
		ReferenceText:      "clear reference text",
		PurchaseLocation:   "Old dealer",
		SoldTo:             "Old buyer",
		IsPrivate:          true,
		IsWishlist:         true,
		IsSold:             true,
		ObverseDescription: "clear obverse",
		ReverseDescription: "clear reverse",
	}
	if err := db.Create(&coin).Error; err != nil {
		t.Fatalf("failed to seed coin: %v", err)
	}

	updates := map[string]interface{}{
		"notes":              "",
		"referenceUrl":       "",
		"referenceText":      "",
		"purchaseLocation":   "",
		"soldTo":             "",
		"obverseDescription": "",
		"reverseDescription": "",
		"isPrivate":          false,
		"isWishlist":         false,
		"isSold":             false,
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

	var found models.Coin
	if err := db.First(&found, coin.ID).Error; err != nil {
		t.Fatalf("updated coin not found: %v", err)
	}
	if found.Notes != "" || found.ReferenceURL != "" || found.ReferenceText != "" ||
		found.PurchaseLocation != "" || found.SoldTo != "" ||
		found.ObverseDescription != "" || found.ReverseDescription != "" {
		t.Fatalf("expected explicit empty strings to persist, got notes=%q refURL=%q refText=%q purchaseLocation=%q soldTo=%q obv=%q rev=%q",
			found.Notes, found.ReferenceURL, found.ReferenceText, found.PurchaseLocation, found.SoldTo, found.ObverseDescription, found.ReverseDescription)
	}
	if found.IsPrivate || found.IsWishlist || found.IsSold {
		t.Fatalf("expected explicit false booleans to persist, got private=%v wishlist=%v sold=%v", found.IsPrivate, found.IsWishlist, found.IsSold)
	}
	if found.Category != models.CategoryRoman || found.Material != models.MaterialSilver || found.Name != "Clearable Coin" {
		t.Fatalf("omitted sibling fields changed unexpectedly: name=%q category=%q material=%q", found.Name, found.Category, found.Material)
	}
}

func TestCoinHandler_Update_CurrentValueCreatesManualHistoryAndSnapshot(t *testing.T) {
	router, db := setupCoinHandlerRouter(t)
	createTestUser(t, db, 1, "updater")

	currentValue := 100.0
	coin := models.Coin{
		Name:         "Value Coin",
		Category:     models.CategoryGreek,
		Material:     models.MaterialSilver,
		UserID:       1,
		CurrentValue: &currentValue,
	}
	if err := db.Create(&coin).Error; err != nil {
		t.Fatalf("failed to seed coin: %v", err)
	}

	body, _ := json.Marshal(map[string]interface{}{"currentValue": 125.0})
	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/coins/%d", coin.ID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authHeader(1))
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var found models.Coin
	if err := db.First(&found, coin.ID).Error; err != nil {
		t.Fatalf("updated coin not found: %v", err)
	}
	if found.CurrentValue == nil || *found.CurrentValue != 125.0 {
		t.Fatalf("expected current value 125, got %v", found.CurrentValue)
	}
	if found.CurrentValueUpdatedAt == nil {
		t.Fatal("expected manual current-value edit to set CurrentValueUpdatedAt")
	}

	var history []models.CoinValueHistory
	if err := db.Where("coin_id = ?", coin.ID).Find(&history).Error; err != nil {
		t.Fatalf("failed to query value history: %v", err)
	}
	if len(history) != 1 || history[0].Value != 125.0 || history[0].Confidence != "manual" {
		t.Fatalf("expected one manual value-history row for 125, got %#v", history)
	}

	var journalCount int64
	if err := db.Model(&models.CoinJournal{}).Where("coin_id = ?", coin.ID).Count(&journalCount).Error; err != nil {
		t.Fatalf("failed to count journal entries: %v", err)
	}
	if journalCount != 1 {
		t.Fatalf("expected one journal entry, got %d", journalCount)
	}
	var snapshotCount int64
	if err := db.Model(&models.ValueSnapshot{}).Where("user_id = ?", uint(1)).Count(&snapshotCount).Error; err != nil {
		t.Fatalf("failed to count value snapshots: %v", err)
	}
	if snapshotCount != 1 {
		t.Fatalf("expected one value snapshot, got %d", snapshotCount)
	}
}

func TestCoinHandler_Update_ClearsNullableScalarsWhenExplicitNullAndPreservesWhenOmitted(t *testing.T) {
	router, db := setupCoinHandlerRouter(t)
	createTestUser(t, db, 1, "updater")

	purchasePrice := 125.0
	currentValue := 175.0
	weight := 3.5
	diameter := 18.0
	purchaseDate := time.Date(2024, time.January, 15, 0, 0, 0, 0, time.UTC)
	soldPrice := 150.0
	soldDate := time.Date(2025, time.February, 20, 0, 0, 0, 0, time.UTC)
	coin := models.Coin{
		Name:          "Nullable Coin",
		Category:      models.CategoryRoman,
		Material:      models.MaterialSilver,
		UserID:        1,
		PurchasePrice: &purchasePrice,
		CurrentValue:  &currentValue,
		WeightGrams:   &weight,
		DiameterMm:    &diameter,
		PurchaseDate:  &purchaseDate,
		SoldPrice:     &soldPrice,
		SoldDate:      &soldDate,
	}
	if err := db.Create(&coin).Error; err != nil {
		t.Fatalf("failed to seed coin: %v", err)
	}

	updates := map[string]interface{}{
		"purchasePrice": nil,
		"currentValue":  nil,
		"purchaseDate":  nil,
		"soldPrice":     nil,
		"soldDate":      nil,
		"weightGrams":   nil,
		"diameterMm":    nil,
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

	var found models.Coin
	if err := db.First(&found, coin.ID).Error; err != nil {
		t.Fatalf("updated coin not found: %v", err)
	}
	if found.PurchasePrice != nil || found.CurrentValue != nil || found.PurchaseDate != nil ||
		found.SoldPrice != nil || found.SoldDate != nil || found.WeightGrams != nil || found.DiameterMm != nil {
		t.Fatalf("expected explicit nulls to clear nullable scalars, got purchase=%v current=%v purchaseDate=%v sold=%v soldDate=%v weight=%v diameter=%v",
			found.PurchasePrice, found.CurrentValue, found.PurchaseDate, found.SoldPrice, found.SoldDate, found.WeightGrams, found.DiameterMm)
	}
	if found.Name != "Nullable Coin" || found.Category != models.CategoryRoman || found.Material != models.MaterialSilver {
		t.Fatalf("omitted non-nullable fields changed unexpectedly: name=%q category=%q material=%q", found.Name, found.Category, found.Material)
	}

	preserved := models.Coin{
		Name:          "Preserved Nullable Coin",
		Category:      models.CategoryRoman,
		Material:      models.MaterialSilver,
		UserID:        1,
		PurchasePrice: &purchasePrice,
		CurrentValue:  &currentValue,
		WeightGrams:   &weight,
		DiameterMm:    &diameter,
		PurchaseDate:  &purchaseDate,
		SoldPrice:     &soldPrice,
		SoldDate:      &soldDate,
	}
	if err := db.Create(&preserved).Error; err != nil {
		t.Fatalf("failed to seed preserved coin: %v", err)
	}
	nameOnlyBody, _ := json.Marshal(map[string]interface{}{"name": "Renamed Preserved Nullable Coin"})
	nameOnlyReq := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/coins/%d", preserved.ID), bytes.NewReader(nameOnlyBody))
	nameOnlyReq.Header.Set("Content-Type", "application/json")
	nameOnlyReq.Header.Set("Authorization", authHeader(1))
	nameOnlyRecorder := httptest.NewRecorder()

	router.ServeHTTP(nameOnlyRecorder, nameOnlyReq)

	if nameOnlyRecorder.Code != http.StatusOK {
		t.Fatalf("expected 200 for name-only update, got %d: %s", nameOnlyRecorder.Code, nameOnlyRecorder.Body.String())
	}
	var preservedFound models.Coin
	if err := db.First(&preservedFound, preserved.ID).Error; err != nil {
		t.Fatalf("preserved coin not found: %v", err)
	}
	if preservedFound.PurchasePrice == nil || *preservedFound.PurchasePrice != purchasePrice ||
		preservedFound.CurrentValue == nil || *preservedFound.CurrentValue != currentValue ||
		preservedFound.PurchaseDate == nil || !preservedFound.PurchaseDate.Equal(purchaseDate) ||
		preservedFound.SoldPrice == nil || *preservedFound.SoldPrice != soldPrice ||
		preservedFound.SoldDate == nil || !preservedFound.SoldDate.Equal(soldDate) ||
		preservedFound.WeightGrams == nil || *preservedFound.WeightGrams != weight ||
		preservedFound.DiameterMm == nil || *preservedFound.DiameterMm != diameter {
		t.Fatalf("expected omitted nullable scalars to be preserved, got purchase=%v current=%v purchaseDate=%v sold=%v soldDate=%v weight=%v diameter=%v",
			preservedFound.PurchasePrice, preservedFound.CurrentValue, preservedFound.PurchaseDate, preservedFound.SoldPrice, preservedFound.SoldDate, preservedFound.WeightGrams, preservedFound.DiameterMm)
	}
}

func TestCoinHandler_Update_ReplacesStructuredReferences(t *testing.T) {
	router, db := setupCoinHandlerRouter(t)
	createTestUser(t, db, 1, "updater")

	if err := db.Create(&models.CatalogRegistry{
		Catalog:        "RIC",
		DisplayName:    "Roman Imperial Coinage",
		Era:            models.EraAncient,
		VolumeRequired: true,
	}).Error; err != nil {
		t.Fatalf("failed to seed catalog registry: %v", err)
	}
	coin := models.Coin{Name: "Reference Coin", Category: models.CategoryRoman, Material: models.MaterialSilver, UserID: 1}
	if err := db.Create(&coin).Error; err != nil {
		t.Fatalf("failed to seed coin: %v", err)
	}
	if err := db.Create(&models.CoinReference{CoinID: coin.ID, Catalog: "RIC", Volume: "I", Number: "1"}).Error; err != nil {
		t.Fatalf("failed to seed reference: %v", err)
	}

	updates := map[string]interface{}{
		"name":     "Reference Coin",
		"category": "Roman",
		"material": "Silver",
		"references": []map[string]interface{}{
			{
				"catalog": " ric ",
				"volume":  " II ",
				"number":  " 12 ",
			},
		},
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

	var refs []models.CoinReference
	if err := db.Where("coin_id = ?", coin.ID).Find(&refs).Error; err != nil {
		t.Fatalf("failed to query references: %v", err)
	}
	if len(refs) != 1 {
		t.Fatalf("expected exactly 1 replacement reference, got %d", len(refs))
	}
	if refs[0].Catalog != "RIC" || refs[0].Volume != "II" || refs[0].Number != "12" {
		t.Fatalf("expected normalized replacement reference RIC II 12, got %#v", refs[0])
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

func TestCoinHandler_Update_AdminConfiguredEraAccepted(t *testing.T) {
	router, db := setupCoinHandlerRouter(t)
	createTestUser(t, db, 1, "updater")

	if err := repository.NewSettingsRepository(db).Upsert(services.SettingCoinEras, "Republican Rome\nRoman Empire\n500-480 BC"); err != nil {
		t.Fatalf("failed to seed coin eras setting: %v", err)
	}
	coin := models.Coin{Name: "Old Era", Category: models.CategoryRoman, Material: models.MaterialBronze, UserID: 1, Era: models.EraAncient}
	db.Create(&coin)

	updates := map[string]interface{}{
		"name":     "Updated Era",
		"category": "Roman",
		"material": "Bronze",
		"era":      "Roman Empire",
	}
	body, _ := json.Marshal(updates)

	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/coins/%d", coin.ID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authHeader(1))
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for admin-configured era, got %d: %s", w.Code, w.Body.String())
	}

	var found models.Coin
	if err := db.First(&found, coin.ID).Error; err != nil {
		t.Fatalf("coin not found: %v", err)
	}
	if found.Era != models.Era("Roman Empire") {
		t.Fatalf("expected era Roman Empire, got %q", found.Era)
	}
}

func TestCoinHandler_Update_PreservesUnchangedLegacyEra(t *testing.T) {
	router, db := setupCoinHandlerRouter(t)
	createTestUser(t, db, 1, "updater")

	coin := models.Coin{Name: "Legacy Era", Category: models.CategoryRoman, Material: models.MaterialBronze, UserID: 1, Era: models.Era("Imperial")}
	if err := db.Create(&coin).Error; err != nil {
		t.Fatalf("failed to seed coin: %v", err)
	}

	updates := map[string]interface{}{
		"name":     "Updated Legacy Era",
		"category": "Roman",
		"material": "Bronze",
		"era":      "Imperial",
	}
	body, _ := json.Marshal(updates)

	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/coins/%d", coin.ID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authHeader(1))
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for unchanged legacy era, got %d: %s", w.Code, w.Body.String())
	}

	var found models.Coin
	if err := db.First(&found, coin.ID).Error; err != nil {
		t.Fatalf("coin not found: %v", err)
	}
	if found.Name != "Updated Legacy Era" {
		t.Fatalf("expected updated name, got %q", found.Name)
	}
	if found.Era != models.Era("Imperial") {
		t.Fatalf("expected legacy era to be preserved, got %q", found.Era)
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

// --- Investment breakdown ---

func TestCoinHandler_InvestmentBreakdown_PurchaseMonth(t *testing.T) {
	router, db := setupCoinHandlerRouter(t)
	createTestUser(t, db, 1, "investor")
	createTestUser(t, db, 2, "other")

	purchaseDate := time.Date(2024, time.January, 15, 0, 0, 0, 0, time.UTC)
	noDatePrice := 999.0
	p100, p50, pSold, pWishlist, pOther := 100.0, 50.0, 80.0, 70.0, 60.0
	cv150, cvSold, cvWishlist, cvOther := 150.0, 90.0, 75.0, 65.0
	coins := []models.Coin{
		{Name: "January Denarius", Category: models.CategoryRoman, Material: models.MaterialSilver, UserID: 1, PurchasePrice: &p100, CurrentValue: &cv150, PurchaseDate: &purchaseDate},
		{Name: "January Fallback", Category: models.CategoryRoman, Material: models.MaterialBronze, UserID: 1, PurchasePrice: &p50, PurchaseDate: &purchaseDate},
		{Name: "No Date", Category: models.CategoryGreek, Material: models.MaterialGold, UserID: 1, PurchasePrice: &noDatePrice},
		{Name: "Sold Date", Category: models.CategoryGreek, Material: models.MaterialGold, UserID: 1, PurchasePrice: &pSold, CurrentValue: &cvSold, PurchaseDate: &purchaseDate, IsSold: true},
		{Name: "Wishlist Date", Category: models.CategoryGreek, Material: models.MaterialGold, UserID: 1, PurchasePrice: &pWishlist, CurrentValue: &cvWishlist, PurchaseDate: &purchaseDate, IsWishlist: true},
		{Name: "Other User Date", Category: models.CategoryGreek, Material: models.MaterialGold, UserID: 2, PurchasePrice: &pOther, CurrentValue: &cvOther, PurchaseDate: &purchaseDate},
	}
	if err := db.Create(&coins).Error; err != nil {
		t.Fatalf("failed to seed investment coins: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/stats/investment-breakdown?dimension=purchase-month", nil)
	req.Header.Set("Authorization", authHeader(1))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Dimension string                                  `json:"dimension"`
		Segments  []repository.InvestmentBreakdownSegment `json:"segments"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Dimension != repository.InvestmentBreakdownPurchaseMonth {
		t.Fatalf("unexpected dimension %q", resp.Dimension)
	}
	if len(resp.Segments) != 1 {
		t.Fatalf("expected 1 purchase-month segment, got %d: %#v", len(resp.Segments), resp.Segments)
	}
	segment := resp.Segments[0]
	if segment.Label != "Jan 2024" || segment.Year == nil || *segment.Year != 2024 || segment.Month == nil || *segment.Month != 1 {
		t.Fatalf("unexpected purchase label/date fields: %#v", segment)
	}
	if segment.CoinCount != 2 || segment.MissingCurrentValueCount != 1 || segment.MissingPurchasePriceCount != 0 {
		t.Fatalf("unexpected confidence counts: %#v", segment)
	}
	if segment.Invested != 150 || segment.CurrentValue != 200 || segment.GainLoss != 50 {
		t.Fatalf("unexpected values: %#v", segment)
	}
	if math.Abs(segment.GainLossPct-33.3333333333) > 0.0001 {
		t.Fatalf("unexpected gainLossPct: %f", segment.GainLossPct)
	}
}

func TestCoinHandler_InvestmentBreakdown_Material(t *testing.T) {
	router, db := setupCoinHandlerRouter(t)
	createTestUser(t, db, 1, "materialuser")

	p100, p40, p10, pSold := 100.0, 40.0, 10.0, 1000.0
	cv120, cv12, cvSold := 120.0, 12.0, 1200.0
	coins := []models.Coin{
		{Name: "Silver A", Category: models.CategoryRoman, Material: models.MaterialSilver, UserID: 1, PurchasePrice: &p100, CurrentValue: &cv120},
		{Name: "Silver B", Category: models.CategoryRoman, Material: models.MaterialSilver, UserID: 1, PurchasePrice: &p40},
		{Name: "Blank Material", Category: models.CategoryGreek, Material: "", UserID: 1, PurchasePrice: &p10, CurrentValue: &cv12},
		{Name: "Sold Silver", Category: models.CategoryRoman, Material: models.MaterialSilver, UserID: 1, PurchasePrice: &pSold, CurrentValue: &cvSold, IsSold: true},
	}
	if err := db.Create(&coins).Error; err != nil {
		t.Fatalf("failed to seed material coins: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/stats/investment-breakdown?dimension=material", nil)
	req.Header.Set("Authorization", authHeader(1))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Dimension string                                  `json:"dimension"`
		Segments  []repository.InvestmentBreakdownSegment `json:"segments"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	segmentsByLabel := map[string]repository.InvestmentBreakdownSegment{}
	for _, segment := range resp.Segments {
		segmentsByLabel[segment.Label] = segment
	}
	silver, ok := segmentsByLabel["Silver"]
	if !ok {
		t.Fatalf("expected Silver segment, got %#v", resp.Segments)
	}
	if silver.CoinCount != 2 || silver.Invested != 140 || silver.CurrentValue != 160 || silver.GainLoss != 20 || silver.MissingCurrentValueCount != 1 {
		t.Fatalf("unexpected Silver segment: %#v", silver)
	}
	other, ok := segmentsByLabel["Other"]
	if !ok {
		t.Fatalf("expected blank material to fall back to Other, got %#v", resp.Segments)
	}
	if other.CoinCount != 1 || other.Invested != 10 || other.CurrentValue != 12 || other.GainLoss != 2 {
		t.Fatalf("unexpected Other segment: %#v", other)
	}
}

func TestCoinHandler_InvestmentBreakdown_InvalidDimension(t *testing.T) {
	router, db := setupCoinHandlerRouter(t)
	createTestUser(t, db, 1, "invaliddimension")

	req := httptest.NewRequest(http.MethodGet, "/api/stats/investment-breakdown?dimension=era", nil)
	req.Header.Set("Authorization", authHeader(1))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

// TestCoinHandler_ActiveCollectionCountInvariant verifies that the canonical
// "active collection" count (owned AND NOT wishlist AND NOT sold) is identical
// across three query paths:
//   - /coins?wishlist=false&sold=false total
//   - /stats totalCoins
//   - internal collection_summary tool totalCoins
//
// This locks the predicate contract: no divergence is permitted.
func TestCoinHandler_ActiveCollectionCountInvariant(t *testing.T) {
	router, db := setupCoinHandlerRouter(t)
	createTestUser(t, db, 1, "countuser")

	coinRepo := repository.NewCoinRepository(db)
	collectionSvc := services.NewCollectionToolsService(coinRepo, nil)

	coins := []models.Coin{
		{Name: "Active 1", Category: models.CategoryRoman, Material: models.MaterialSilver, UserID: 1, IsWishlist: false, IsSold: false},
		{Name: "Active 2", Category: models.CategoryGreek, Material: models.MaterialGold, UserID: 1, IsWishlist: false, IsSold: false},
		{Name: "Active 3", Category: models.CategoryByzantine, Material: models.MaterialBronze, UserID: 1, IsWishlist: false, IsSold: false},
		{Name: "Wishlist 1", Category: models.CategoryRoman, Material: models.MaterialGold, UserID: 1, IsWishlist: true, IsSold: false},
		{Name: "Wishlist 2", Category: models.CategoryGreek, Material: models.MaterialSilver, UserID: 1, IsWishlist: true, IsSold: false},
		{Name: "Sold 1", Category: models.CategoryRoman, Material: models.MaterialSilver, UserID: 1, IsWishlist: false, IsSold: true},
	}
	if err := db.Create(&coins).Error; err != nil {
		t.Fatalf("failed to seed mixed coin collection: %v", err)
	}

	// Path 1: /coins?wishlist=false&sold=false total
	req := httptest.NewRequest(http.MethodGet, "/api/coins?wishlist=false&sold=false", nil)
	req.Header.Set("Authorization", authHeader(1))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 from /coins, got %d: %s", w.Code, w.Body.String())
	}
	var listResp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("failed to decode /coins response: %v", err)
	}
	listTotal := int64(listResp["total"].(float64))
	if listTotal != 3 {
		t.Errorf("/coins?wishlist=false&sold=false: expected total=3, got %d", listTotal)
	}

	// Path 2: /stats totalCoins
	setupStatsRouter := func(t *testing.T, db *gorm.DB) *gin.Engine {
		gin.SetMode(gin.TestMode)
		coinRepo := repository.NewCoinRepository(db)
		catalogRegistryRepo := repository.NewCatalogRegistryRepository(db)
		coinReferenceRepo := repository.NewCoinReferenceRepository(db)
		coinReferenceSvc := services.NewCoinReferenceService(coinReferenceRepo, catalogRegistryRepo)
		storageLocationRepo := repository.NewStorageLocationRepository(db)
		coinSvc := services.NewCoinService(coinRepo, nil).
			WithReferenceSupport(coinReferenceRepo, coinReferenceSvc).
			WithStorageLocationSupport(storageLocationRepo).
			WithCatalogRegistrySupport(catalogRegistryRepo)
		handler := NewCoinHandler(coinRepo, coinSvc, services.NewLogger(100))
		r := gin.New()
		protected := r.Group("/api")
		protected.Use(coinTestAuthMiddleware())
		protected.GET("/stats", handler.Stats)
		return r
	}
	statsRouter := setupStatsRouter(t, db)
	req2 := httptest.NewRequest(http.MethodGet, "/api/stats", nil)
	req2.Header.Set("Authorization", authHeader(1))
	w2 := httptest.NewRecorder()
	statsRouter.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("expected 200 from /stats, got %d: %s", w2.Code, w2.Body.String())
	}
	var statsResp map[string]interface{}
	if err := json.Unmarshal(w2.Body.Bytes(), &statsResp); err != nil {
		t.Fatalf("failed to decode /stats response: %v", err)
	}
	statsTotalCoins := int64(statsResp["totalCoins"].(float64))
	if statsTotalCoins != 3 {
		t.Errorf("/stats: expected totalCoins=3, got %d", statsTotalCoins)
	}

	// Path 3: collection_summary tool totalCoins
	summary, err := collectionSvc.CollectionSummary(1)
	if err != nil {
		t.Fatalf("CollectionSummary error: %v", err)
	}
	if summary.TotalCoins != 3 {
		t.Errorf("CollectionSummary: expected totalCoins=3, got %d", summary.TotalCoins)
	}

	// Invariant: all three paths must return the same count
	if listTotal != statsTotalCoins || statsTotalCoins != summary.TotalCoins {
		t.Errorf("INVARIANT VIOLATION: /coins total=%d, /stats totalCoins=%d, collection_summary totalCoins=%d (expected all=3)",
			listTotal, statsTotalCoins, summary.TotalCoins)
	}

	// Verify wishlist and sold counts are correct
	statsWishlist := int64(statsResp["totalWishlist"].(float64))
	statsSold := int64(statsResp["totalSold"].(float64))
	if statsWishlist != 2 {
		t.Errorf("expected totalWishlist=2, got %d", statsWishlist)
	}
	if statsSold != 1 {
		t.Errorf("expected totalSold=1, got %d", statsSold)
	}
	if summary.TotalWishlist != 2 {
		t.Errorf("collection_summary: expected totalWishlist=2, got %d", summary.TotalWishlist)
	}
	if summary.TotalSold != 1 {
		t.Errorf("collection_summary: expected totalSold=1, got %d", summary.TotalSold)
	}
}
