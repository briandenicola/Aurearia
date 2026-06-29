package repository

import (
	"math"
	"testing"
	"time"

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
		&models.User{}, &models.StorageLocation{}, &models.Coin{}, &models.CoinImage{}, &models.CoinReference{},
		&models.ValueSnapshot{}, &models.CoinJournal{},
		&models.CoinValueHistory{}, &models.CoinComment{},
		&models.AvailabilityResult{}, &models.AuctionLot{},
		&models.Tag{}, &models.CoinTag{},
		&models.CoinSet{}, &models.CoinSetMembership{},
		&models.QuickCaptureDraft{}, &models.QuickCaptureDraftImage{}, &models.DraftLifecycleEvent{},
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

func TestCoinRepository_QuickCaptureDraftsExcludedAndPromotedCoinAppearsOnce(t *testing.T) {
	db := setupTestDB(t)
	repo := NewCoinRepository(db)
	userID := uint(1)

	active := models.Coin{Name: "Active normal coin", Category: models.CategoryRoman, Material: models.MaterialSilver, UserID: userID, IsWishlist: false, IsSold: false}
	wishlist := models.Coin{Name: "Wishlist coin", Category: models.CategoryGreek, Material: models.MaterialGold, UserID: userID, IsWishlist: true, IsSold: false}
	sold := models.Coin{Name: "Sold coin", Category: models.CategoryRoman, Material: models.MaterialBronze, UserID: userID, IsWishlist: false, IsSold: true}
	promoted := models.Coin{Name: "Promoted Quick Capture coin", Category: models.CategoryRoman, Material: models.MaterialSilver, UserID: userID, IsWishlist: false, IsSold: false}
	for _, coin := range []*models.Coin{&active, &wishlist, &sold, &promoted} {
		if err := db.Create(coin).Error; err != nil {
			t.Fatalf("seed coin: %v", err)
		}
	}
	promotedCoinID := promoted.ID
	drafts := []models.QuickCaptureDraft{
		{UserID: userID, WorkingTitle: "Sparse active draft", Status: models.QuickCaptureDraftStatusActive},
		{UserID: userID, WorkingTitle: "Promoted draft", Status: models.QuickCaptureDraftStatusPromoted, PromotedCoinID: &promotedCoinID},
	}
	for i := range drafts {
		if err := db.Create(&drafts[i]).Error; err != nil {
			t.Fatalf("seed draft: %v", err)
		}
	}

	activeFilter := false
	coins, total, err := repo.List(userID, CoinListFilters{Wishlist: &activeFilter, Sold: &activeFilter, Page: 1, Limit: 50, SortField: "name", SortOrder: "asc"})
	if err != nil {
		t.Fatalf("list active coins: %v", err)
	}
	if total != 2 || len(coins) != 2 {
		t.Fatalf("expected only active normal/promoted coins, total=%d len=%d coins=%+v", total, len(coins), coins)
	}
	names := map[string]int{}
	for _, coin := range coins {
		names[coin.Name]++
	}
	if names["Sparse active draft"] != 0 || names["Promoted draft"] != 0 {
		t.Fatalf("quick-capture draft rows leaked into normal coin list: %v", names)
	}
	if names["Promoted Quick Capture coin"] != 1 {
		t.Fatalf("promoted normal coin should appear exactly once, got names=%v", names)
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

func ptrTime(v time.Time) *time.Time { return &v }

func assertFloatNear(t *testing.T, got float64, want float64) {
	t.Helper()
	if math.Abs(got-want) > 0.0001 {
		t.Fatalf("expected %.4f, got %.4f", want, got)
	}
}

func TestCoinRepository_GetInvestmentBreakdown_InvalidDimension(t *testing.T) {
	db := setupTestDB(t)
	repo := NewCoinRepository(db)

	segments, err := repo.GetInvestmentBreakdown(1, "ruler")
	if err == nil {
		t.Fatal("expected invalid dimension error")
	}
	if segments != nil {
		t.Fatalf("expected nil segments for invalid dimension, got %#v", segments)
	}
}

func TestCoinRepository_GetInvestmentBreakdown_MaterialAggregatesConfidenceCounts(t *testing.T) {
	db := setupTestDB(t)
	repo := NewCoinRepository(db)
	jan := time.Date(2024, time.January, 15, 0, 0, 0, 0, time.UTC)

	coins := []models.Coin{
		{Name: "Silver Valued", Category: models.CategoryRoman, Material: models.MaterialSilver, UserID: 1, PurchasePrice: ptrFloat(100), CurrentValue: ptrFloat(150), PurchaseDate: ptrTime(jan)},
		{Name: "Silver Fallback", Category: models.CategoryRoman, Material: models.MaterialSilver, UserID: 1, PurchasePrice: ptrFloat(200), CurrentValue: nil, PurchaseDate: ptrTime(jan)},
		{Name: "Gold Missing Cost", Category: models.CategoryRoman, Material: models.MaterialGold, UserID: 1, PurchasePrice: nil, CurrentValue: ptrFloat(80), PurchaseDate: ptrTime(jan)},
		{Name: "Wishlist Excluded", Category: models.CategoryRoman, Material: models.MaterialGold, UserID: 1, IsWishlist: true, PurchasePrice: ptrFloat(999), CurrentValue: ptrFloat(999), PurchaseDate: ptrTime(jan)},
		{Name: "Sold Excluded", Category: models.CategoryRoman, Material: models.MaterialSilver, UserID: 1, IsSold: true, PurchasePrice: ptrFloat(999), CurrentValue: ptrFloat(999), PurchaseDate: ptrTime(jan)},
		{Name: "Other User Excluded", Category: models.CategoryRoman, Material: models.MaterialSilver, UserID: 2, PurchasePrice: ptrFloat(999), CurrentValue: ptrFloat(999), PurchaseDate: ptrTime(jan)},
	}
	for i := range coins {
		if err := db.Create(&coins[i]).Error; err != nil {
			t.Fatalf("Create coin %q failed: %v", coins[i].Name, err)
		}
	}

	segments, err := repo.GetInvestmentBreakdown(1, InvestmentBreakdownMaterial)
	if err != nil {
		t.Fatalf("GetInvestmentBreakdown failed: %v", err)
	}
	if len(segments) != 2 {
		t.Fatalf("expected 2 material segments, got %d: %#v", len(segments), segments)
	}

	silver := segments[0]
	if silver.Label != string(models.MaterialSilver) {
		t.Fatalf("expected Silver first by invested total, got %q", silver.Label)
	}
	assertFloatNear(t, silver.Invested, 300)
	assertFloatNear(t, silver.CurrentValue, 350)
	assertFloatNear(t, silver.GainLoss, 50)
	assertFloatNear(t, silver.GainLossPct, 16.6666667)
	if silver.CoinCount != 2 || silver.MissingCurrentValueCount != 1 || silver.MissingPurchasePriceCount != 0 {
		t.Fatalf("unexpected Silver counts: coin=%d missingCurrent=%d missingPurchase=%d", silver.CoinCount, silver.MissingCurrentValueCount, silver.MissingPurchasePriceCount)
	}

	gold := segments[1]
	if gold.Label != string(models.MaterialGold) {
		t.Fatalf("expected Gold second, got %q", gold.Label)
	}
	assertFloatNear(t, gold.Invested, 0)
	assertFloatNear(t, gold.CurrentValue, 80)
	assertFloatNear(t, gold.GainLoss, 80)
	assertFloatNear(t, gold.GainLossPct, 0)
	if gold.CoinCount != 1 || gold.MissingCurrentValueCount != 0 || gold.MissingPurchasePriceCount != 1 {
		t.Fatalf("unexpected Gold counts: coin=%d missingCurrent=%d missingPurchase=%d", gold.CoinCount, gold.MissingCurrentValueCount, gold.MissingPurchasePriceCount)
	}
}

func TestCoinRepository_GetInvestmentBreakdown_PurchaseMonthAggregatesConfidenceCounts(t *testing.T) {
	db := setupTestDB(t)
	repo := NewCoinRepository(db)
	jan := time.Date(2024, time.January, 15, 0, 0, 0, 0, time.UTC)
	feb := time.Date(2024, time.February, 20, 0, 0, 0, 0, time.UTC)

	coins := []models.Coin{
		{Name: "Jan Valued", Category: models.CategoryRoman, Material: models.MaterialSilver, UserID: 1, PurchasePrice: ptrFloat(100), CurrentValue: ptrFloat(120), PurchaseDate: ptrTime(jan)},
		{Name: "Jan Missing Current", Category: models.CategoryRoman, Material: models.MaterialSilver, UserID: 1, PurchasePrice: ptrFloat(200), CurrentValue: nil, PurchaseDate: ptrTime(jan)},
		{Name: "Feb Missing Cost", Category: models.CategoryRoman, Material: models.MaterialGold, UserID: 1, PurchasePrice: nil, CurrentValue: ptrFloat(80), PurchaseDate: ptrTime(feb)},
		{Name: "No Date Excluded", Category: models.CategoryRoman, Material: models.MaterialGold, UserID: 1, PurchasePrice: ptrFloat(999), CurrentValue: ptrFloat(999)},
	}
	for i := range coins {
		if err := db.Create(&coins[i]).Error; err != nil {
			t.Fatalf("Create coin %q failed: %v", coins[i].Name, err)
		}
	}

	segments, err := repo.GetInvestmentBreakdown(1, InvestmentBreakdownPurchaseMonth)
	if err != nil {
		t.Fatalf("GetInvestmentBreakdown failed: %v", err)
	}
	if len(segments) != 2 {
		t.Fatalf("expected 2 purchase-month segments, got %d: %#v", len(segments), segments)
	}

	janSegment := segments[0]
	if janSegment.Label != "Jan 2024" || janSegment.Year == nil || *janSegment.Year != 2024 || janSegment.Month == nil || *janSegment.Month != 1 {
		t.Fatalf("unexpected January label/date fields: %#v", janSegment)
	}
	assertFloatNear(t, janSegment.Invested, 300)
	assertFloatNear(t, janSegment.CurrentValue, 320)
	if janSegment.CoinCount != 2 || janSegment.MissingCurrentValueCount != 1 || janSegment.MissingPurchasePriceCount != 0 {
		t.Fatalf("unexpected January counts: coin=%d missingCurrent=%d missingPurchase=%d", janSegment.CoinCount, janSegment.MissingCurrentValueCount, janSegment.MissingPurchasePriceCount)
	}

	febSegment := segments[1]
	if febSegment.Label != "Feb 2024" || febSegment.Year == nil || *febSegment.Year != 2024 || febSegment.Month == nil || *febSegment.Month != 2 {
		t.Fatalf("unexpected February label/date fields: %#v", febSegment)
	}
	assertFloatNear(t, febSegment.Invested, 0)
	assertFloatNear(t, febSegment.CurrentValue, 80)
	if febSegment.CoinCount != 1 || febSegment.MissingCurrentValueCount != 0 || febSegment.MissingPurchasePriceCount != 1 {
		t.Fatalf("unexpected February counts: coin=%d missingCurrent=%d missingPurchase=%d", febSegment.CoinCount, febSegment.MissingCurrentValueCount, febSegment.MissingPurchasePriceCount)
	}
}

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

// TestCoinRepository_Update_PreservesSets tests regression where updating a coin
// with Sets relation must NOT corrupt existing memberships by recreating them
// without AddedAt (violating NOT NULL constraint).
func TestCoinRepository_Update_PreservesSets(t *testing.T) {
	db := setupTestDB(t)
	coinRepo := NewCoinRepository(db)
	setRepo := NewSetRepository(db)

	// Create a coin
	coin := &models.Coin{
		Name:     "Test Aureus",
		Category: models.CategoryRoman,
		Material: models.MaterialGold,
		UserID:   1,
	}
	if err := coinRepo.Create(coin); err != nil {
		t.Fatalf("Create coin failed: %v", err)
	}

	// Create a set
	set := &models.CoinSet{
		UserID:  1,
		Name:    "Roman Gold",
		SetType: models.CoinSetTypeOpen,
	}
	if err := setRepo.Create(set); err != nil {
		t.Fatalf("Create set failed: %v", err)
	}

	// Add coin to set via SetRepository (correct path with AddedAt)
	if err := setRepo.AddCoinToSet(coin.ID, set.ID, 1, ""); err != nil {
		t.Fatalf("AddCoinToSet failed: %v", err)
	}

	// Verify membership was created with AddedAt
	var membership models.CoinSetMembership
	if err := db.Where("coin_id = ? AND set_id = ?", coin.ID, set.ID).First(&membership).Error; err != nil {
		t.Fatalf("membership not found: %v", err)
	}
	if membership.AddedAt.IsZero() {
		t.Fatal("membership.AddedAt is zero; should be set by AddCoinToSet")
	}
	originalAddedAt := membership.AddedAt

	// Now update the coin via CoinRepository
	updates := &models.Coin{
		Name: "Updated Aureus",
	}
	if err := coinRepo.Update(coin, updates); err != nil {
		t.Fatalf("Update coin failed: %v", err)
	}

	// Verify coin was updated
	if coin.Name != "Updated Aureus" {
		t.Errorf("expected updated name 'Updated Aureus', got %q", coin.Name)
	}

	// Critical: verify membership still exists with the same AddedAt
	var updatedMembership models.CoinSetMembership
	if err := db.Where("coin_id = ? AND set_id = ?", coin.ID, set.ID).First(&updatedMembership).Error; err != nil {
		t.Fatalf("membership disappeared after coin update: %v", err)
	}
	if updatedMembership.AddedAt.IsZero() {
		t.Fatal("membership.AddedAt is zero after update; Omit('Sets') failed")
	}
	if !updatedMembership.AddedAt.Equal(originalAddedAt) {
		t.Errorf("membership.AddedAt changed from %v to %v; should be preserved", originalAddedAt, updatedMembership.AddedAt)
	}
}

// TestCoinRepository_Update_WithSetsField tests that passing coin.Sets in an update
// does NOT modify memberships due to Omit("Sets") in Update method.
func TestCoinRepository_Update_WithSetsField(t *testing.T) {
	db := setupTestDB(t)
	coinRepo := NewCoinRepository(db)
	setRepo := NewSetRepository(db)

	// Create coin and two sets
	coin := &models.Coin{
		Name:     "Test Solidus",
		Category: models.CategoryByzantine,
		UserID:   1,
	}
	if err := coinRepo.Create(coin); err != nil {
		t.Fatalf("Create coin failed: %v", err)
	}

	set1 := &models.CoinSet{UserID: 1, Name: "Byzantine Core", SetType: models.CoinSetTypeOpen}
	set2 := &models.CoinSet{UserID: 1, Name: "High Grade", SetType: models.CoinSetTypeOpen}
	if err := setRepo.Create(set1); err != nil {
		t.Fatalf("Create set1 failed: %v", err)
	}
	if err := setRepo.Create(set2); err != nil {
		t.Fatalf("Create set2 failed: %v", err)
	}

	// Add coin to set1 only
	if err := setRepo.AddCoinToSet(coin.ID, set1.ID, 1, "initial"); err != nil {
		t.Fatalf("AddCoinToSet failed: %v", err)
	}

	var count int64
	db.Model(&models.CoinSetMembership{}).Where("coin_id = ?", coin.ID).Count(&count)
	if count != 1 {
		t.Fatalf("expected 1 membership before update, got %d", count)
	}

	// Attempt to update coin with coin.Sets = [set2] (should be ignored by Omit)
	updates := &models.Coin{
		Name: "Updated Solidus",
		Sets: []models.CoinSet{*set2},
	}
	if err := coinRepo.Update(coin, updates); err != nil {
		t.Fatalf("Update coin failed: %v", err)
	}

	// Verify name was updated but Sets relationship was NOT replaced
	if coin.Name != "Updated Solidus" {
		t.Errorf("expected name 'Updated Solidus', got %q", coin.Name)
	}

	// Should still have exactly 1 membership (set1), not replaced by set2
	db.Model(&models.CoinSetMembership{}).Where("coin_id = ?", coin.ID).Count(&count)
	if count != 1 {
		t.Errorf("expected 1 membership after update (should be ignored), got %d", count)
	}

	var membership models.CoinSetMembership
	if err := db.Where("coin_id = ? AND set_id = ?", coin.ID, set1.ID).First(&membership).Error; err != nil {
		t.Fatal("original membership (set1) disappeared; Omit('Sets') failed")
	}
	if membership.AddedAt.IsZero() {
		t.Fatal("membership.AddedAt is zero; should be preserved")
	}

	// Verify set2 was NOT added
	var set2Count int64
	db.Model(&models.CoinSetMembership{}).Where("coin_id = ? AND set_id = ?", coin.ID, set2.ID).Count(&set2Count)
	if set2Count != 0 {
		t.Error("set2 was added despite Omit('Sets'); update should not touch Sets")
	}
}

func TestCoinRepository_Update_PreservesLoadedAssociations(t *testing.T) {
	db := setupTestDB(t)
	coinRepo := NewCoinRepository(db)
	setRepo := NewSetRepository(db)
	tagRepo := NewTagRepository(db)

	location := models.StorageLocation{UserID: 1, Name: "Cabinet A"}
	if err := db.Create(&location).Error; err != nil {
		t.Fatalf("Create storage location failed: %v", err)
	}
	coin := &models.Coin{
		Name:              "Associated Coin",
		Category:          models.CategoryRoman,
		Material:          models.MaterialSilver,
		UserID:            1,
		StorageLocationID: &location.ID,
	}
	if err := coinRepo.Create(coin); err != nil {
		t.Fatalf("Create coin failed: %v", err)
	}
	if err := db.Create(&models.CoinImage{CoinID: coin.ID, FilePath: "coins/original.jpg", ImageType: models.ImageTypeObverse, IsPrimary: true}).Error; err != nil {
		t.Fatalf("Create image failed: %v", err)
	}
	if err := db.Create(&models.CoinReference{CoinID: coin.ID, Catalog: "RIC", Volume: "I", Number: "1"}).Error; err != nil {
		t.Fatalf("Create reference failed: %v", err)
	}
	tag := &models.Tag{UserID: 1, Name: "Favorites", Color: "#c9a84c"}
	if err := tagRepo.Create(tag); err != nil {
		t.Fatalf("Create tag failed: %v", err)
	}
	if err := tagRepo.AttachToCoin(coin.ID, tag.ID, 1); err != nil {
		t.Fatalf("Attach tag failed: %v", err)
	}
	set := &models.CoinSet{UserID: 1, Name: "Roman Core", SetType: models.CoinSetTypeOpen}
	if err := setRepo.Create(set); err != nil {
		t.Fatalf("Create set failed: %v", err)
	}
	if err := setRepo.AddCoinToSet(coin.ID, set.ID, 1, "original"); err != nil {
		t.Fatalf("AddCoinToSet failed: %v", err)
	}

	loaded, err := coinRepo.FindByID(coin.ID, 1)
	if err != nil {
		t.Fatalf("FindByID failed: %v", err)
	}
	if len(loaded.Images) != 1 || len(loaded.References) != 1 || len(loaded.Tags) != 1 || len(loaded.Sets) != 1 || loaded.StorageLocation == nil {
		t.Fatalf("expected loaded associations before update, got images=%d refs=%d tags=%d sets=%d storage=%v", len(loaded.Images), len(loaded.References), len(loaded.Tags), len(loaded.Sets), loaded.StorageLocation)
	}

	incomingTag := models.Tag{ID: tag.ID + 100, UserID: 1, Name: "Incoming"}
	incomingSet := models.CoinSet{ID: set.ID + 100, UserID: 1, Name: "Incoming Set", SetType: models.CoinSetTypeOpen}
	updates := &models.Coin{
		Name: "Updated Associated Coin",
		Images: []models.CoinImage{
			{ID: loaded.Images[0].ID + 100, CoinID: loaded.ID, FilePath: "coins/incoming.jpg", ImageType: models.ImageTypeReverse},
		},
		References: []models.CoinReference{
			{ID: loaded.References[0].ID + 100, CoinID: loaded.ID, Catalog: "RIC", Volume: "II", Number: "2"},
		},
		Tags:            []models.Tag{incomingTag},
		Sets:            []models.CoinSet{incomingSet},
		StorageLocation: &models.StorageLocation{ID: location.ID + 100, UserID: 1, Name: "Incoming Location"},
	}
	if err := coinRepo.Update(loaded, updates); err != nil {
		t.Fatalf("Update coin failed: %v", err)
	}

	var found models.Coin
	if err := db.Preload("Images").Preload("References").Preload("Tags").Preload("Sets").Preload("StorageLocation").First(&found, coin.ID).Error; err != nil {
		t.Fatalf("coin not found after update: %v", err)
	}
	if found.Name != "Updated Associated Coin" {
		t.Fatalf("expected name update, got %q", found.Name)
	}
	if len(found.Images) != 1 || found.Images[0].FilePath != "coins/original.jpg" {
		t.Fatalf("expected original image to remain, got %#v", found.Images)
	}
	if len(found.References) != 1 || found.References[0].Number != "1" {
		t.Fatalf("expected original reference to remain, got %#v", found.References)
	}
	if len(found.Tags) != 1 || found.Tags[0].ID != tag.ID {
		t.Fatalf("expected original tag to remain, got %#v", found.Tags)
	}
	if len(found.Sets) != 1 || found.Sets[0].ID != set.ID {
		t.Fatalf("expected original set to remain, got %#v", found.Sets)
	}
	if found.StorageLocationID == nil || *found.StorageLocationID != location.ID {
		t.Fatalf("expected original storage location %d to remain, got %v", location.ID, found.StorageLocationID)
	}
	var incomingImageCount int64
	if err := db.Model(&models.CoinImage{}).Where("file_path = ?", "coins/incoming.jpg").Count(&incomingImageCount).Error; err != nil {
		t.Fatalf("failed to count incoming image: %v", err)
	}
	if incomingImageCount != 0 {
		t.Fatal("incoming image association was persisted by scalar update")
	}
}

func TestCoinRepository_Update_WithSelectedFieldsPersistsExplicitZeroValues(t *testing.T) {
	db := setupTestDB(t)
	coinRepo := NewCoinRepository(db)

	purchasePrice := 125.0
	currentValue := 175.0
	weight := 3.5
	diameter := 18.0
	coin := &models.Coin{
		Name:             "Zero Value Coin",
		Category:         models.CategoryRoman,
		Material:         models.MaterialSilver,
		UserID:           1,
		Notes:            "clear me",
		ReferenceURL:     "https://example.test/ref",
		ReferenceText:    "clear reference text",
		PurchaseLocation: "Old dealer",
		IsPrivate:        true,
		IsWishlist:       true,
		IsSold:           true,
		PurchasePrice:    &purchasePrice,
		CurrentValue:     &currentValue,
		WeightGrams:      &weight,
		DiameterMm:       &diameter,
	}
	if err := coinRepo.Create(coin); err != nil {
		t.Fatalf("Create coin failed: %v", err)
	}

	zero := 0.0
	updates := &models.Coin{
		Notes:            "",
		ReferenceURL:     "",
		ReferenceText:    "",
		PurchaseLocation: "",
		IsPrivate:        false,
		IsWishlist:       false,
		IsSold:           false,
		PurchasePrice:    &zero,
		CurrentValue:     &zero,
		WeightGrams:      &zero,
		DiameterMm:       &zero,
	}
	if err := coinRepo.Update(
		coin,
		updates,
		"Notes",
		"ReferenceURL",
		"ReferenceText",
		"PurchaseLocation",
		"IsPrivate",
		"IsWishlist",
		"IsSold",
		"PurchasePrice",
		"CurrentValue",
		"WeightGrams",
		"DiameterMm",
	); err != nil {
		t.Fatalf("Update coin failed: %v", err)
	}

	var found models.Coin
	if err := db.First(&found, coin.ID).Error; err != nil {
		t.Fatalf("coin not found after update: %v", err)
	}
	if found.Notes != "" || found.ReferenceURL != "" || found.ReferenceText != "" || found.PurchaseLocation != "" {
		t.Fatalf("expected empty string clears to persist, got notes=%q refURL=%q refText=%q purchaseLocation=%q",
			found.Notes, found.ReferenceURL, found.ReferenceText, found.PurchaseLocation)
	}
	if found.IsPrivate || found.IsWishlist || found.IsSold {
		t.Fatalf("expected false booleans to persist, got private=%v wishlist=%v sold=%v", found.IsPrivate, found.IsWishlist, found.IsSold)
	}
	if found.PurchasePrice == nil || *found.PurchasePrice != 0 ||
		found.CurrentValue == nil || *found.CurrentValue != 0 ||
		found.WeightGrams == nil || *found.WeightGrams != 0 ||
		found.DiameterMm == nil || *found.DiameterMm != 0 {
		t.Fatalf("expected explicit numeric zeros to persist, got purchase=%v current=%v weight=%v diameter=%v",
			found.PurchasePrice, found.CurrentValue, found.WeightGrams, found.DiameterMm)
	}
	if found.Name != "Zero Value Coin" || found.Category != models.CategoryRoman || found.Material != models.MaterialSilver {
		t.Fatalf("omitted fields changed unexpectedly: name=%q category=%q material=%q", found.Name, found.Category, found.Material)
	}
}

func TestCoinRepository_Update_WithSelectedFieldsPersistsNilNullableScalars(t *testing.T) {
	db := setupTestDB(t)
	coinRepo := NewCoinRepository(db)

	purchasePrice := 125.0
	currentValue := 175.0
	weight := 3.5
	diameter := 18.0
	purchaseDate := time.Date(2024, time.January, 15, 0, 0, 0, 0, time.UTC)
	soldPrice := 150.0
	soldDate := time.Date(2025, time.February, 20, 0, 0, 0, 0, time.UTC)
	coin := &models.Coin{
		Name:          "Nil Nullable Coin",
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
	if err := coinRepo.Create(coin); err != nil {
		t.Fatalf("Create coin failed: %v", err)
	}

	updates := &models.Coin{}
	if err := coinRepo.Update(
		coin,
		updates,
		"PurchasePrice",
		"CurrentValue",
		"PurchaseDate",
		"SoldPrice",
		"SoldDate",
		"WeightGrams",
		"DiameterMm",
	); err != nil {
		t.Fatalf("Update coin failed: %v", err)
	}

	var found models.Coin
	if err := db.First(&found, coin.ID).Error; err != nil {
		t.Fatalf("coin not found after update: %v", err)
	}
	if found.PurchasePrice != nil || found.CurrentValue != nil || found.PurchaseDate != nil ||
		found.SoldPrice != nil || found.SoldDate != nil || found.WeightGrams != nil || found.DiameterMm != nil {
		t.Fatalf("expected selected nil nullable scalars to persist, got purchase=%v current=%v purchaseDate=%v sold=%v soldDate=%v weight=%v diameter=%v",
			found.PurchasePrice, found.CurrentValue, found.PurchaseDate, found.SoldPrice, found.SoldDate, found.WeightGrams, found.DiameterMm)
	}
	if found.Name != "Nil Nullable Coin" || found.Category != models.CategoryRoman || found.Material != models.MaterialSilver {
		t.Fatalf("omitted fields changed unexpectedly: name=%q category=%q material=%q", found.Name, found.Category, found.Material)
	}
}

func TestCoinRepository_UpdateStorageLocationID_PersistsNullClear(t *testing.T) {
	db := setupTestDB(t)
	coinRepo := NewCoinRepository(db)

	location := models.StorageLocation{UserID: 1, Name: "Cabinet A"}
	if err := db.Create(&location).Error; err != nil {
		t.Fatalf("Create storage location failed: %v", err)
	}
	coin := &models.Coin{
		Name:              "Storage Clear Coin",
		Category:          models.CategoryRoman,
		Material:          models.MaterialSilver,
		UserID:            1,
		StorageLocationID: &location.ID,
	}
	if err := coinRepo.Create(coin); err != nil {
		t.Fatalf("Create coin failed: %v", err)
	}

	if err := coinRepo.UpdateStorageLocationID(coin, nil); err != nil {
		t.Fatalf("UpdateStorageLocationID clear failed: %v", err)
	}

	var found models.Coin
	if err := db.First(&found, coin.ID).Error; err != nil {
		t.Fatalf("coin not found after clear: %v", err)
	}
	if found.StorageLocationID != nil {
		t.Fatalf("expected storage location NULL, got %v", found.StorageLocationID)
	}
	if coin.StorageLocationID != nil || coin.StorageLocation != nil {
		t.Fatalf("expected reloaded coin to have nil storage fields, got id=%v location=%v", coin.StorageLocationID, coin.StorageLocation)
	}
}

func TestCoinRepository_UpdateHelpers_WithLoadedSetsDoNotSyncMemberships(t *testing.T) {
	tests := []struct {
		name   string
		update func(repo *CoinRepository, coin *models.Coin) error
		assert func(t *testing.T, db *gorm.DB, coinID uint)
	}{
		{
			name: "UpdateField",
			update: func(repo *CoinRepository, coin *models.Coin) error {
				return repo.UpdateField(coin, "name", "Updated Field Coin")
			},
			assert: func(t *testing.T, db *gorm.DB, coinID uint) {
				t.Helper()
				var found models.Coin
				if err := db.First(&found, coinID).Error; err != nil {
					t.Fatalf("coin not found: %v", err)
				}
				if found.Name != "Updated Field Coin" {
					t.Fatalf("expected updated name, got %q", found.Name)
				}
			},
		},
		{
			name: "UpdateFields",
			update: func(repo *CoinRepository, coin *models.Coin) error {
				return repo.UpdateFields(coin, map[string]interface{}{"name": "Updated Fields Coin"})
			},
			assert: func(t *testing.T, db *gorm.DB, coinID uint) {
				t.Helper()
				var found models.Coin
				if err := db.First(&found, coinID).Error; err != nil {
					t.Fatalf("coin not found: %v", err)
				}
				if found.Name != "Updated Fields Coin" {
					t.Fatalf("expected updated name, got %q", found.Name)
				}
			},
		},
		{
			name: "UpdateStorageLocationID",
			update: func(repo *CoinRepository, coin *models.Coin) error {
				storageLocationID := uint(5)
				return repo.UpdateStorageLocationID(coin, &storageLocationID)
			},
			assert: func(t *testing.T, db *gorm.DB, coinID uint) {
				t.Helper()
				var found models.Coin
				if err := db.First(&found, coinID).Error; err != nil {
					t.Fatalf("coin not found: %v", err)
				}
				if found.StorageLocationID == nil || *found.StorageLocationID != 5 {
					t.Fatalf("expected storage location 5, got %v", found.StorageLocationID)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupTestDB(t)
			coinRepo := NewCoinRepository(db)
			setRepo := NewSetRepository(db)

			coin := &models.Coin{
				Name:     "Loaded Set Coin",
				Category: models.CategoryRoman,
				UserID:   1,
			}
			if err := coinRepo.Create(coin); err != nil {
				t.Fatalf("Create coin failed: %v", err)
			}

			set := &models.CoinSet{UserID: 1, Name: "Loaded Set", SetType: models.CoinSetTypeOpen}
			if err := setRepo.Create(set); err != nil {
				t.Fatalf("Create set failed: %v", err)
			}
			if err := setRepo.AddCoinToSet(coin.ID, set.ID, 1, ""); err != nil {
				t.Fatalf("AddCoinToSet failed: %v", err)
			}

			loaded, err := coinRepo.FindByID(coin.ID, 1)
			if err != nil {
				t.Fatalf("FindByID failed: %v", err)
			}
			if len(loaded.Sets) != 1 {
				t.Fatalf("expected preloaded set, got %d", len(loaded.Sets))
			}

			var originalMembership models.CoinSetMembership
			if err := db.Where("coin_id = ? AND set_id = ?", coin.ID, set.ID).First(&originalMembership).Error; err != nil {
				t.Fatalf("membership not found: %v", err)
			}

			if err := tt.update(coinRepo, loaded); err != nil {
				t.Fatalf("%s failed: %v", tt.name, err)
			}
			tt.assert(t, db, coin.ID)

			var memberships []models.CoinSetMembership
			if err := db.Where("coin_id = ?", coin.ID).Find(&memberships).Error; err != nil {
				t.Fatalf("failed to query memberships: %v", err)
			}
			if len(memberships) != 1 {
				t.Fatalf("expected exactly 1 membership after %s, got %d", tt.name, len(memberships))
			}
			if memberships[0].SetID != set.ID {
				t.Fatalf("expected original membership to remain, got set ID %d", memberships[0].SetID)
			}
			if memberships[0].AddedAt.IsZero() {
				t.Fatal("membership AddedAt should remain populated")
			}
			if !memberships[0].AddedAt.Equal(originalMembership.AddedAt) {
				t.Fatalf("membership AddedAt changed from %v to %v", originalMembership.AddedAt, memberships[0].AddedAt)
			}
		})
	}
}
