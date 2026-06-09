package services

import (
	"errors"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	err = db.AutoMigrate(
		&models.User{}, &models.Coin{}, &models.CoinImage{}, &models.CoinReference{}, &models.CatalogRegistry{},
		&models.StorageLocation{}, &models.ValueSnapshot{}, &models.CoinJournal{},
		&models.CoinValueHistory{}, &models.CoinComment{},
		&models.AvailabilityResult{}, &models.AuctionLot{},
		&models.Tag{}, &models.CoinTag{},
	)
	if err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	return db
}

func newTestCoinService(db *gorm.DB) *CoinService {
	repo := repository.NewCoinRepository(db)
	return NewCoinService(repo, nil)
}

func newTestCoinServiceWithCatalogRegistry(db *gorm.DB) *CoinService {
	repo := repository.NewCoinRepository(db)
	catalogRepo := repository.NewCatalogRegistryRepository(db)
	return NewCoinService(repo, nil).WithCatalogRegistrySupport(catalogRepo)
}

func ptrFloat(v float64) *float64 { return &v }

func TestCreateCoin_Success(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestCoinService(db)

	coin := &models.Coin{
		Name:          "Denarius of Augustus",
		Category:      models.CategoryRoman,
		Material:      models.MaterialSilver,
		PurchasePrice: ptrFloat(250.0),
		CurrentValue:  ptrFloat(300.0),
		UserID:        1,
	}

	if err := svc.CreateCoin(coin); err != nil {
		t.Fatalf("CreateCoin failed: %v", err)
	}

	if coin.ID == 0 {
		t.Fatal("expected coin ID to be set after create")
	}

	// Verify coin exists in DB
	var found models.Coin
	if err := db.First(&found, coin.ID).Error; err != nil {
		t.Fatalf("coin not found in DB: %v", err)
	}
	if found.Name != "Denarius of Augustus" {
		t.Errorf("expected name 'Denarius of Augustus', got %q", found.Name)
	}

	// Verify value snapshot was recorded
	var snapshots []models.ValueSnapshot
	db.Where("user_id = ?", 1).Find(&snapshots)
	if len(snapshots) == 0 {
		t.Fatal("expected a value snapshot to be recorded")
	}
	if snapshots[0].CoinCount != 1 {
		t.Errorf("expected snapshot coin count 1, got %d", snapshots[0].CoinCount)
	}
}

func TestUpdateCoin_RecordsValueHistory(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestCoinService(db)

	coin := &models.Coin{
		Name:         "Tetradrachm",
		Category:     models.CategoryGreek,
		Material:     models.MaterialSilver,
		CurrentValue: ptrFloat(500.0),
		UserID:       1,
	}
	if err := svc.CreateCoin(coin); err != nil {
		t.Fatalf("setup: CreateCoin failed: %v", err)
	}

	// Update with a new current value (source != "estimate")
	updates := &models.Coin{CurrentValue: ptrFloat(750.0)}
	if err := svc.UpdateCoin(coin, updates, 1, "manual"); err != nil {
		t.Fatalf("UpdateCoin failed: %v", err)
	}

	// Verify value history was recorded
	var history []models.CoinValueHistory
	db.Where("coin_id = ?", coin.ID).Find(&history)
	if len(history) != 1 {
		t.Fatalf("expected 1 value history entry, got %d", len(history))
	}
	if history[0].Value != 750.0 {
		t.Errorf("expected history value 750, got %f", history[0].Value)
	}
	if history[0].Confidence != "manual" {
		t.Errorf("expected confidence 'manual', got %q", history[0].Confidence)
	}

	// Verify journal entry was created
	var journals []models.CoinJournal
	db.Where("coin_id = ?", coin.ID).Find(&journals)
	if len(journals) != 1 {
		t.Fatalf("expected 1 journal entry, got %d", len(journals))
	}
}

func TestUpdateCoin_EstimateSkipsHistory(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestCoinService(db)

	coin := &models.Coin{
		Name:         "Follis",
		Category:     models.CategoryByzantine,
		Material:     models.MaterialBronze,
		CurrentValue: ptrFloat(100.0),
		UserID:       1,
	}
	if err := svc.CreateCoin(coin); err != nil {
		t.Fatalf("setup: CreateCoin failed: %v", err)
	}

	updates := &models.Coin{CurrentValue: ptrFloat(120.0)}
	if err := svc.UpdateCoin(coin, updates, 1, "estimate"); err != nil {
		t.Fatalf("UpdateCoin failed: %v", err)
	}

	// source="estimate" should NOT record value history
	var count int64
	db.Model(&models.CoinValueHistory{}).Where("coin_id = ?", coin.ID).Count(&count)
	if count != 0 {
		t.Errorf("expected 0 value history entries for estimate source, got %d", count)
	}
}

func TestDeleteCoin_RemovesCoinAndImages(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestCoinService(db)

	coin := &models.Coin{
		Name:     "Sestertius",
		Category: models.CategoryRoman,
		Material: models.MaterialBronze,
		UserID:   1,
	}
	if err := svc.CreateCoin(coin); err != nil {
		t.Fatalf("setup: CreateCoin failed: %v", err)
	}

	// Add an image for this coin
	img := models.CoinImage{CoinID: coin.ID, FilePath: "test.jpg", ImageType: models.ImageTypeObverse}
	db.Create(&img)

	rows, err := svc.DeleteCoin(coin.ID, 1)
	if err != nil {
		t.Fatalf("DeleteCoin failed: %v", err)
	}
	if rows != 1 {
		t.Errorf("expected 1 row affected, got %d", rows)
	}

	// Coin should be gone
	var count int64
	db.Model(&models.Coin{}).Where("id = ?", coin.ID).Count(&count)
	if count != 0 {
		t.Error("expected coin to be deleted")
	}

	// Image should be gone
	db.Model(&models.CoinImage{}).Where("coin_id = ?", coin.ID).Count(&count)
	if count != 0 {
		t.Error("expected coin image to be deleted")
	}
}

func TestDeleteCoin_WrongUser_NoEffect(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestCoinService(db)

	coin := &models.Coin{
		Name:     "Stater",
		Category: models.CategoryGreek,
		Material: models.MaterialGold,
		UserID:   1,
	}
	if err := svc.CreateCoin(coin); err != nil {
		t.Fatalf("setup: CreateCoin failed: %v", err)
	}

	// Try deleting with wrong user ID
	rows, err := svc.DeleteCoin(coin.ID, 999)
	if err != nil {
		t.Fatalf("DeleteCoin failed: %v", err)
	}
	if rows != 0 {
		t.Errorf("expected 0 rows affected for wrong user, got %d", rows)
	}

	// Coin should still exist
	var count int64
	db.Model(&models.Coin{}).Where("id = ?", coin.ID).Count(&count)
	if count != 1 {
		t.Error("expected coin to still exist")
	}
}

func TestPurchaseCoin_UpdatesFields(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestCoinService(db)

	now := time.Now()
	coin := &models.Coin{
		Name:       "Aureus Wishlist",
		Category:   models.CategoryRoman,
		Material:   models.MaterialGold,
		IsWishlist: true,
		UserID:     1,
	}
	if err := svc.CreateCoin(coin); err != nil {
		t.Fatalf("setup: CreateCoin failed: %v", err)
	}

	coin.PurchasePrice = ptrFloat(5000.0)
	coin.PurchaseDate = &now
	coin.PurchaseLocation = "Auction House"

	if err := svc.PurchaseCoin(coin, 1); err != nil {
		t.Fatalf("PurchaseCoin failed: %v", err)
	}

	// Reload and verify
	var updated models.Coin
	db.First(&updated, coin.ID)
	if updated.IsWishlist {
		t.Error("expected IsWishlist to be false after purchase")
	}
	if updated.PurchasePrice == nil || *updated.PurchasePrice != 5000.0 {
		t.Error("expected PurchasePrice to be 5000")
	}
	if updated.PurchaseLocation != "Auction House" {
		t.Errorf("expected PurchaseLocation 'Auction House', got %q", updated.PurchaseLocation)
	}
}

func TestSellCoin_UpdatesFields(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestCoinService(db)

	coin := &models.Coin{
		Name:          "Solidus",
		Category:      models.CategoryByzantine,
		Material:      models.MaterialGold,
		PurchasePrice: ptrFloat(2000.0),
		UserID:        1,
	}

	if err := svc.CreateCoin(coin); err != nil {
		t.Fatalf("setup: CreateCoin failed: %v", err)
	}

	now := time.Now()
	updates := map[string]interface{}{
		"is_sold":    true,
		"sold_price": 3000.0,
		"sold_date":  now,
		"sold_to":    "Private Collector",
	}

	if err := svc.SellCoin(coin, updates, 1); err != nil {
		t.Fatalf("SellCoin failed: %v", err)
	}

	var sold models.Coin
	db.First(&sold, coin.ID)
	if !sold.IsSold {
		t.Error("expected IsSold to be true")
	}
	if sold.SoldPrice == nil || *sold.SoldPrice != 3000.0 {
		t.Error("expected SoldPrice to be 3000")
	}
	if sold.SoldTo != "Private Collector" {
		t.Errorf("expected SoldTo 'Private Collector', got %q", sold.SoldTo)
	}
}

func TestUpdateCoin_AcceptsCustomEraFromCatalogRegistry(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestCoinServiceWithCatalogRegistry(db)
	if err := db.Create(&models.CatalogRegistry{
		Catalog:     "PROV",
		DisplayName: "Provincial References",
		Era:         models.Era("provincial"),
	}).Error; err != nil {
		t.Fatalf("failed to seed catalog registry: %v", err)
	}

	coin := &models.Coin{Name: "Provincial Bronze", UserID: 1, Era: models.EraAncient}
	if err := svc.CreateCoin(coin); err != nil {
		t.Fatalf("setup: CreateCoin failed: %v", err)
	}

	updates := &models.Coin{Era: models.Era("provincial")}
	if err := svc.UpdateCoin(coin, updates, 1, "manual"); err != nil {
		t.Fatalf("UpdateCoin should accept registry-defined era: %v", err)
	}

	var found models.Coin
	if err := db.First(&found, coin.ID).Error; err != nil {
		t.Fatalf("coin not found: %v", err)
	}
	if found.Era != models.Era("provincial") {
		t.Fatalf("expected era provincial, got %q", found.Era)
	}
}

func TestUpdateCoin_RejectsUnsupportedEraWhenCatalogRegistryEnabled(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestCoinServiceWithCatalogRegistry(db)

	coin := &models.Coin{Name: "Test Coin", UserID: 1, Era: models.EraAncient}
	if err := svc.CreateCoin(coin); err != nil {
		t.Fatalf("setup: CreateCoin failed: %v", err)
	}

	updates := &models.Coin{Era: models.Era("unsupported-era")}
	if err := svc.UpdateCoin(coin, updates, 1, "manual"); !errors.Is(err, ErrCoinInvalidEra) {
		t.Fatalf("expected ErrCoinInvalidEra, got %v", err)
	}
}

func TestUpdateCoin_PreservesUnchangedLegacyEra(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestCoinServiceWithCatalogRegistry(db)

	coin := &models.Coin{Name: "Legacy Era Coin", UserID: 1, Era: models.Era("Imperial")}
	if err := db.Create(coin).Error; err != nil {
		t.Fatalf("setup: create legacy era coin failed: %v", err)
	}

	updates := &models.Coin{Name: "Updated Legacy Era Coin", Era: models.Era("Imperial")}
	if err := svc.UpdateCoin(coin, updates, 1, "manual"); err != nil {
		t.Fatalf("UpdateCoin should allow unchanged legacy era: %v", err)
	}

	var found models.Coin
	if err := db.First(&found, coin.ID).Error; err != nil {
		t.Fatalf("coin not found: %v", err)
	}
	if found.Name != "Updated Legacy Era Coin" {
		t.Fatalf("expected updated name, got %q", found.Name)
	}
	if found.Era != models.Era("Imperial") {
		t.Fatalf("expected legacy era to be preserved, got %q", found.Era)
	}
}
