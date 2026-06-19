package services

import (
	"strings"
	"testing"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupCollectionToolsServiceTest(t *testing.T) (*CollectionToolsService, uint) {
	t.Helper()

	dbName := strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())
	db, err := gorm.Open(sqlite.Open("file:"+dbName+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.Coin{}, &models.CoinImage{}, &models.CoinReference{}); err != nil {
		t.Fatalf("failed to migrate test db: %v", err)
	}

	user := models.User{Username: "tray-user", PasswordHash: "hash", Email: "tray-user@test.example"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	return NewCollectionToolsService(repository.NewCoinRepository(db), nil), user.ID
}

func floatPtr(value float64) *float64 {
	return &value
}

func TestSearchMyCollectionFindsCoinsMissingSize(t *testing.T) {
	service, userID := setupCollectionToolsServiceTest(t)

	coinRepo := service.coinRepo
	coins := []models.Coin{
		{
			Name:        "Measured Denarius",
			UserID:      userID,
			Category:    models.CategoryRoman,
			Material:    models.MaterialSilver,
			WeightGrams: floatPtr(3.2),
			DiameterMm:  floatPtr(18),
		},
		{
			Name:        "Unmeasured Follis",
			UserID:      userID,
			Category:    models.CategoryRoman,
			Material:    models.MaterialBronze,
			WeightGrams: floatPtr(8.1),
			DiameterMm:  nil,
		},
		{
			Name:        "Zero Diameter Bronze",
			UserID:      userID,
			Category:    models.CategoryRoman,
			Material:    models.MaterialBronze,
			WeightGrams: floatPtr(6.4),
			DiameterMm:  floatPtr(0),
		},
	}
	for i := range coins {
		if err := coinRepo.Create(&coins[i]); err != nil {
			t.Fatalf("failed to create coin: %v", err)
		}
	}

	results, err := service.SearchMyCollection(userID, "coins missing size", nil)
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 coins missing size, got %d", len(results))
	}
	for _, coin := range results {
		if coin.Name == "Measured Denarius" {
			t.Fatalf("measured coin should not be returned for missing size")
		}
		if !containsMissingField(coin.MissingFields, "diameterMm") {
			t.Fatalf("expected %s to report missing diameterMm, got %#v", coin.Name, coin.MissingFields)
		}
	}
}

func TestCollectionSummaryIncludesMissingFieldCounts(t *testing.T) {
	service, userID := setupCollectionToolsServiceTest(t)

	coinRepo := service.coinRepo
	coins := []models.Coin{
		{Name: "Complete Coin", UserID: userID, Category: models.CategoryRoman, Material: models.MaterialSilver, WeightGrams: floatPtr(3.5), DiameterMm: floatPtr(18), CurrentValue: floatPtr(100)},
		{Name: "Missing Diameter", UserID: userID, Category: models.CategoryRoman, Material: models.MaterialBronze, WeightGrams: floatPtr(7), CurrentValue: floatPtr(80)},
		{Name: "Missing Weight", UserID: userID, Category: models.CategoryGreek, Material: models.MaterialSilver, DiameterMm: floatPtr(22), CurrentValue: floatPtr(120)},
	}
	for i := range coins {
		if err := coinRepo.Create(&coins[i]); err != nil {
			t.Fatalf("failed to create coin: %v", err)
		}
	}

	summary, err := service.CollectionSummary(userID)
	if err != nil {
		t.Fatalf("summary failed: %v", err)
	}

	if summary.MissingFields["diameterMm"] != 1 {
		t.Fatalf("expected 1 missing diameterMm, got %d", summary.MissingFields["diameterMm"])
	}
	if summary.MissingFields["weightGrams"] != 1 {
		t.Fatalf("expected 1 missing weightGrams, got %d", summary.MissingFields["weightGrams"])
	}
}

func containsMissingField(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
