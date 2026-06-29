package repository

import (
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
)

func TestValuationRepository_GetOwnedCoinsPrioritizesStaleValuations(t *testing.T) {
	db := setupTestDB(t)
	repo := NewValuationRepository(db)

	now := time.Now().UTC().Truncate(time.Second)
	oldestValuation := now.AddDate(0, -6, 0)
	staleValuation := now.AddDate(0, -1, 0)
	recentValuation := now.Add(-24 * time.Hour)

	coins := []models.Coin{
		{
			Name: "recently edited but never valued", Category: models.CategoryRoman, UserID: 1,
			UpdatedAt: now,
		},
		{
			Name: "oldest valuation", Category: models.CategoryRoman, UserID: 1,
			CurrentValueUpdatedAt: &oldestValuation,
			UpdatedAt:             now.Add(-time.Hour),
		},
		{
			Name: "stale valuation", Category: models.CategoryRoman, UserID: 1,
			CurrentValueUpdatedAt: &staleValuation,
			UpdatedAt:             now.Add(-2 * time.Hour),
		},
		{
			Name: "recent valuation", Category: models.CategoryRoman, UserID: 1,
			CurrentValueUpdatedAt: &recentValuation,
			UpdatedAt:             now.AddDate(0, 0, -30),
		},
		{
			Name: "wishlist never valued", Category: models.CategoryRoman, UserID: 1,
			IsWishlist: true,
		},
		{
			Name: "sold never valued", Category: models.CategoryRoman, UserID: 1,
			IsSold: true,
		},
		{
			Name: "other user never valued", Category: models.CategoryRoman, UserID: 2,
		},
	}
	for i := range coins {
		if err := db.Create(&coins[i]).Error; err != nil {
			t.Fatalf("failed to create coin %q: %v", coins[i].Name, err)
		}
	}

	got, err := repo.GetOwnedCoins(1, 3)
	if err != nil {
		t.Fatalf("GetOwnedCoins failed: %v", err)
	}

	wantNames := []string{
		"recently edited but never valued",
		"oldest valuation",
		"stale valuation",
	}
	if len(got) != len(wantNames) {
		t.Fatalf("expected %d coins, got %d: %#v", len(wantNames), len(got), got)
	}
	for i, want := range wantNames {
		if got[i].Name != want {
			t.Fatalf("coin %d = %q, want %q", i, got[i].Name, want)
		}
	}
}
