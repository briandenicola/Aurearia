package repository

import (
	"testing"

	"github.com/briandenicola/ancient-coins-api/models"
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
		&models.User{}, &models.Coin{}, &models.CoinImage{}, &models.CoinReference{},
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

func TestCoinRepository_CreateAndGet(t *testing.T) {
	db := setupTestDB(t)
	repo := NewCoinRepository(db)

	coin := &models.Coin{
		Name:     "Test Denarius",
		Category: models.CategoryRoman,
		Material: models.MaterialSilver,
		UserID:   1,
	}

	if err := repo.Create(coin); err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if coin.ID == 0 {
		t.Fatal("expected coin ID to be set")
	}

	found, err := repo.FindByID(coin.ID, 1)
	if err != nil {
		t.Fatalf("FindByID failed: %v", err)
	}
	if found.Name != "Test Denarius" {
		t.Errorf("expected name 'Test Denarius', got %q", found.Name)
	}
	if found.Category != models.CategoryRoman {
		t.Errorf("expected category Roman, got %q", found.Category)
	}
}

func TestCoinRepository_FindByID_WrongUser(t *testing.T) {
	db := setupTestDB(t)
	repo := NewCoinRepository(db)

	coin := &models.Coin{Name: "Private Coin", Category: models.CategoryGreek, UserID: 1}
	repo.Create(coin)

	_, err := repo.FindByID(coin.ID, 999)
	if err == nil {
		t.Fatal("expected error when fetching coin with wrong user ID")
	}
}

func TestCoinRepository_WithTx(t *testing.T) {
	db := setupTestDB(t)
	repo := NewCoinRepository(db)

	err := db.Transaction(func(tx *gorm.DB) error {
		txRepo := repo.WithTx(tx)
		coin := &models.Coin{
			Name:     "TX Coin",
			Category: models.CategoryRoman,
			UserID:   1,
		}
		if err := txRepo.Create(coin); err != nil {
			return err
		}

		// Should be visible within the transaction
		found, err := txRepo.FindByID(coin.ID, 1)
		if err != nil {
			return err
		}
		if found.Name != "TX Coin" {
			t.Errorf("expected 'TX Coin', got %q", found.Name)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("transaction failed: %v", err)
	}

	// Should also be visible after commit
	var count int64
	db.Model(&models.Coin{}).Where("name = ?", "TX Coin").Count(&count)
	if count != 1 {
		t.Error("expected coin to persist after transaction commit")
	}
}

func TestCoinRepository_Delete(t *testing.T) {
	db := setupTestDB(t)
	repo := NewCoinRepository(db)

	coin := &models.Coin{Name: "To Delete", Category: models.CategoryRoman, UserID: 1}
	repo.Create(coin)

	// Add related data
	db.Create(&models.CoinImage{CoinID: coin.ID, FilePath: "img.jpg"})
	db.Create(&models.CoinJournal{CoinID: coin.ID, UserID: 1, Entry: "test"})

	rows, err := repo.Delete(coin.ID, 1)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	if rows != 1 {
		t.Errorf("expected 1 row affected, got %d", rows)
	}

	var coinCount, imgCount, journalCount int64
	db.Model(&models.Coin{}).Where("id = ?", coin.ID).Count(&coinCount)
	db.Model(&models.CoinImage{}).Where("coin_id = ?", coin.ID).Count(&imgCount)
	db.Model(&models.CoinJournal{}).Where("coin_id = ?", coin.ID).Count(&journalCount)

	if coinCount != 0 {
		t.Error("coin should be deleted")
	}
	if imgCount != 0 {
		t.Error("coin image should be deleted")
	}
	if journalCount != 0 {
		t.Error("journal entry should be deleted")
	}
}

func TestCoinRepository_CoinExists(t *testing.T) {
	db := setupTestDB(t)
	repo := NewCoinRepository(db)

	coin := &models.Coin{Name: "Exists Test", Category: models.CategoryRoman, UserID: 1}
	repo.Create(coin)

	exists, err := repo.CoinExists(coin.ID, 1)
	if err != nil {
		t.Fatalf("CoinExists failed: %v", err)
	}
	if !exists {
		t.Error("expected coin to exist")
	}

	exists, err = repo.CoinExists(coin.ID, 999)
	if err != nil {
		t.Fatalf("CoinExists failed: %v", err)
	}
	if exists {
		t.Error("expected coin to not exist for wrong user")
	}
}

func TestCoinRepository_Scopes_OwnedBy(t *testing.T) {
	db := setupTestDB(t)

	// Create coins for two users
	db.Create(&models.Coin{Name: "User1 Coin A", Category: models.CategoryRoman, UserID: 1})
	db.Create(&models.Coin{Name: "User1 Coin B", Category: models.CategoryGreek, UserID: 1})
	db.Create(&models.Coin{Name: "User2 Coin", Category: models.CategoryRoman, UserID: 2})

	var coins []models.Coin
	db.Scopes(OwnedBy(1)).Find(&coins)
	if len(coins) != 2 {
		t.Errorf("expected 2 coins for user 1, got %d", len(coins))
	}

	db.Scopes(OwnedBy(2)).Find(&coins)
	if len(coins) != 1 {
		t.Errorf("expected 1 coin for user 2, got %d", len(coins))
	}
}

func TestCoinRepository_Scopes_ActiveCollection(t *testing.T) {
	db := setupTestDB(t)

	db.Create(&models.Coin{Name: "Active", Category: models.CategoryRoman, UserID: 1, IsWishlist: false, IsSold: false})
	db.Create(&models.Coin{Name: "Wishlist", Category: models.CategoryRoman, UserID: 1, IsWishlist: true, IsSold: false})
	db.Create(&models.Coin{Name: "Sold", Category: models.CategoryRoman, UserID: 1, IsWishlist: false, IsSold: true})

	var coins []models.Coin
	db.Scopes(ActiveCollection(1)).Find(&coins)
	if len(coins) != 1 {
		t.Fatalf("expected 1 active coin, got %d", len(coins))
	}
	if coins[0].Name != "Active" {
		t.Errorf("expected 'Active', got %q", coins[0].Name)
	}
}

func TestCoinRepository_Scopes_PublicCoins(t *testing.T) {
	db := setupTestDB(t)

	db.Create(&models.Coin{Name: "Public", Category: models.CategoryRoman, UserID: 1, IsPrivate: false})
	db.Create(&models.Coin{Name: "Private", Category: models.CategoryRoman, UserID: 1, IsPrivate: true})
	db.Create(&models.Coin{Name: "Wishlist", Category: models.CategoryRoman, UserID: 1, IsWishlist: true})

	var coins []models.Coin
	db.Scopes(PublicCoins(1)).Find(&coins)
	if len(coins) != 1 {
		t.Fatalf("expected 1 public coin, got %d", len(coins))
	}
	if coins[0].Name != "Public" {
		t.Errorf("expected 'Public', got %q", coins[0].Name)
	}
}

func TestCoinRepository_RecordValueSnapshot(t *testing.T) {
	db := setupTestDB(t)
	repo := NewCoinRepository(db)

	// Create two coins (not wishlist) with known values
	db.Create(&models.Coin{
		Name: "Coin A", Category: models.CategoryRoman, UserID: 1,
		PurchasePrice: ptrFloat(100.0), CurrentValue: ptrFloat(150.0),
	})
	db.Create(&models.Coin{
		Name: "Coin B", Category: models.CategoryRoman, UserID: 1,
		PurchasePrice: ptrFloat(200.0), CurrentValue: ptrFloat(250.0),
	})
	// Wishlist coin should be excluded
	db.Create(&models.Coin{
		Name: "Wishlist", Category: models.CategoryRoman, UserID: 1,
		IsWishlist: true, PurchasePrice: ptrFloat(9999.0),
	})

	if err := repo.RecordValueSnapshot(1); err != nil {
		t.Fatalf("RecordValueSnapshot failed: %v", err)
	}

	var snap models.ValueSnapshot
	db.Where("user_id = ?", 1).First(&snap)
	if snap.CoinCount != 2 {
		t.Errorf("expected coin count 2, got %d", snap.CoinCount)
	}
	if snap.TotalInvested != 300.0 {
		t.Errorf("expected total invested 300, got %f", snap.TotalInvested)
	}
	if snap.TotalValue != 400.0 {
		t.Errorf("expected total value 400, got %f", snap.TotalValue)
	}
}

func ptrFloat(v float64) *float64 { return &v }

func TestCoinRepository_List_RandomSort(t *testing.T) {
	db := setupTestDB(t)
	repo := NewCoinRepository(db)

	// Insert 10 coins for the same user.
	for i := 1; i <= 10; i++ {
		if err := repo.Create(&models.Coin{
			Name:     "Coin",
			Category: models.CategoryRoman,
			UserID:   1,
		}); err != nil {
			t.Fatalf("Create failed: %v", err)
		}
	}

	listIDs := func(filters CoinListFilters) []uint {
		coins, _, err := repo.List(1, filters)
		if err != nil {
			t.Fatalf("List failed: %v", err)
		}
		ids := make([]uint, len(coins))
		for i, c := range coins {
			ids[i] = c.ID
		}
		return ids
	}

	// Baseline: created_at desc (newest first) — should be 10, 9, 8, ... 1.
	desc := listIDs(CoinListFilters{SortField: "created_at", SortOrder: "desc", Page: 1, Limit: 50})

	// Same seed twice must yield the same order (deterministic).
	seed := 12345
	a := listIDs(CoinListFilters{SortField: "random", Seed: &seed, Page: 1, Limit: 50})
	b := listIDs(CoinListFilters{SortField: "random", Seed: &seed, Page: 1, Limit: 50})
	if len(a) != 10 || len(b) != 10 {
		t.Fatalf("expected 10 coins, got %d and %d", len(a), len(b))
	}
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("random sort not deterministic for same seed at index %d: %v vs %v", i, a, b)
		}
	}

	// Random ordering must NOT equal the natural insertion / created_at order.
	differs := false
	for i := range a {
		if a[i] != desc[i] {
			differs = true
			break
		}
	}
	if !differs {
		t.Fatalf("random sort produced the same order as created_at desc; the seed has no effect: %v", a)
	}

	// A different seed should produce a different ordering than the first seed.
	seed2 := 99999
	c := listIDs(CoinListFilters{SortField: "random", Seed: &seed2, Page: 1, Limit: 50})
	differs = false
	for i := range a {
		if a[i] != c[i] {
			differs = true
			break
		}
	}
	if !differs {
		t.Fatalf("different seeds produced identical ordering; seed has no effect: %v", a)
	}
}
