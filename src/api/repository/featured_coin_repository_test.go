package repository

import (
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupFeaturedCoinTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.Coin{}, &models.FeaturedCoin{}); err != nil {
		t.Fatalf("failed to migrate test db: %v", err)
	}
	return db
}

func createFeaturedCoinTestUser(t *testing.T, db *gorm.DB, username string) models.User {
	t.Helper()

	user := models.User{
		Username:     username,
		Email:        username + "@example.com",
		PasswordHash: "hash",
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}
	return user
}

func createFeaturedCoinTestCoin(t *testing.T, db *gorm.DB, userID uint, name string, wishlist, sold bool) models.Coin {
	t.Helper()

	coin := models.Coin{
		Name:       name,
		Category:   models.CategoryRoman,
		Material:   models.MaterialSilver,
		Era:        models.EraAncient,
		UserID:     userID,
		IsWishlist: wishlist,
		IsSold:     sold,
	}
	if err := db.Create(&coin).Error; err != nil {
		t.Fatalf("failed to create coin: %v", err)
	}
	return coin
}

func createFeaturedCoinRecord(t *testing.T, db *gorm.DB, userID, coinID uint, featuredAt time.Time) {
	t.Helper()

	if err := db.Create(&models.FeaturedCoin{
		UserID:     userID,
		CoinID:     coinID,
		Summary:    "Featured coin",
		FeaturedAt: featuredAt,
	}).Error; err != nil {
		t.Fatalf("failed to create featured coin: %v", err)
	}
}

func TestPickNextCoinIDPrefersNeverFeaturedCoin(t *testing.T) {
	db := setupFeaturedCoinTestDB(t)
	repo := NewFeaturedCoinRepository(db)
	user := createFeaturedCoinTestUser(t, db, "never-featured")
	shown := createFeaturedCoinTestCoin(t, db, user.ID, "Shown", false, false)
	neverShown := createFeaturedCoinTestCoin(t, db, user.ID, "Never shown", false, false)

	createFeaturedCoinRecord(t, db, user.ID, shown.ID, time.Now().Add(-24*time.Hour))

	got, err := repo.PickNextCoinID(user.ID)
	if err != nil {
		t.Fatalf("PickNextCoinID returned error: %v", err)
	}
	if got != neverShown.ID {
		t.Fatalf("PickNextCoinID = %d, want never-shown coin %d", got, neverShown.ID)
	}
}

func TestPickNextCoinIDChoosesOldestFeaturedCoinWhenCycleRestarts(t *testing.T) {
	db := setupFeaturedCoinTestDB(t)
	repo := NewFeaturedCoinRepository(db)
	user := createFeaturedCoinTestUser(t, db, "cycle")
	oldestShown := createFeaturedCoinTestCoin(t, db, user.ID, "Oldest shown", false, false)
	recentlyShown := createFeaturedCoinTestCoin(t, db, user.ID, "Recently shown", false, false)

	createFeaturedCoinRecord(t, db, user.ID, oldestShown.ID, time.Now().Add(-48*time.Hour))
	createFeaturedCoinRecord(t, db, user.ID, recentlyShown.ID, time.Now().Add(-24*time.Hour))

	got, err := repo.PickNextCoinID(user.ID)
	if err != nil {
		t.Fatalf("PickNextCoinID returned error: %v", err)
	}
	if got != oldestShown.ID {
		t.Fatalf("PickNextCoinID = %d, want oldest featured coin %d", got, oldestShown.ID)
	}
}

func TestPickNextCoinIDExcludesWishlistAndSoldCoins(t *testing.T) {
	db := setupFeaturedCoinTestDB(t)
	repo := NewFeaturedCoinRepository(db)
	user := createFeaturedCoinTestUser(t, db, "eligible")
	createFeaturedCoinTestCoin(t, db, user.ID, "Wishlist", true, false)
	createFeaturedCoinTestCoin(t, db, user.ID, "Sold", false, true)

	got, err := repo.PickNextCoinID(user.ID)
	if err != nil {
		t.Fatalf("PickNextCoinID returned error: %v", err)
	}
	if got != 0 {
		t.Fatalf("PickNextCoinID = %d, want 0 when no eligible coins exist", got)
	}
}
