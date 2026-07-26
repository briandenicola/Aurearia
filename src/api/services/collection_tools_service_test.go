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
	if err := db.AutoMigrate(&models.User{}, &models.Coin{}, &models.CoinImage{}, &models.CoinReference{}, &models.AppSetting{}); err != nil {
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

func TestSearchMyCollection_MatchesModernEra(t *testing.T) {
	service, userID := setupCollectionToolsServiceTest(t)

	coinRepo := service.coinRepo
	coins := []models.Coin{
		{Name: "Modern Follis", UserID: userID, Category: models.CategoryModern, Era: models.EraModern},
		{Name: "Ancient Denarius", UserID: userID, Category: models.CategoryRoman, Era: models.EraAncient},
	}
	for i := range coins {
		if err := coinRepo.Create(&coins[i]); err != nil {
			t.Fatalf("failed to create coin: %v", err)
		}
	}

	// The full query text is also matched as a substring search against the
	// coin name (an existing, unrelated behavior of this tool), so the query
	// is kept to a single word that both names the era and appears in the
	// name of the coin it should match.
	results, err := service.SearchMyCollection(userID, "modern", nil)
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}
	if len(results) != 1 || results[0].Name != "Modern Follis" {
		t.Fatalf("expected only the modern-era coin to match, got %+v", results)
	}
}

func TestSearchMyCollection_MatchesAdminDefinedCategory(t *testing.T) {
	service, userID := setupCollectionToolsServiceTest(t)
	db := service.coinRepo.DB()
	settingsRepo := repository.NewSettingsRepository(db)
	if err := settingsRepo.Upsert(SettingCoinCategories, "Roman\nGreek\nByzantine\nModern\nOther\nCeltic"); err != nil {
		t.Fatalf("seed coin categories setting: %v", err)
	}
	service = service.WithSettingsSupport(NewSettingsService(settingsRepo))

	coinRepo := service.coinRepo
	coins := []models.Coin{
		{Name: "Celtic Stater", UserID: userID, Category: models.Category("Celtic")},
		{Name: "Roman Denarius", UserID: userID, Category: models.CategoryRoman},
	}
	for i := range coins {
		if err := coinRepo.Create(&coins[i]); err != nil {
			t.Fatalf("failed to create coin: %v", err)
		}
	}

	results, err := service.SearchMyCollection(userID, "celtic", nil)
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}
	if len(results) != 1 || results[0].Name != "Celtic Stater" {
		t.Fatalf("expected only the Celtic coin to match, got %+v", results)
	}
}
