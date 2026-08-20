package services

import (
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func day(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func money(v float64) *float64 { return &v }

func setupTimeMachine(t *testing.T) (*TimeMachineService, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(
		&models.Coin{}, &models.CoinValueHistory{}, &models.CollectionHealthSnapshot{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	svc := NewTimeMachineService(repository.NewTimeMachineRepository(db)).
		WithClock(func() time.Time { return day(2026, time.August, 20) })
	return svc, db
}

// A coin bought after the target date must not appear, and one sold before it
// must have dropped out — that is the whole point of the feature.
func TestTimeMachine_OwnershipWindow(t *testing.T) {
	svc, db := setupTimeMachine(t)

	purchased := day(2024, time.March, 10)
	soldOn := day(2025, time.January, 5)
	laterPurchase := day(2026, time.February, 1)

	coins := []models.Coin{
		{Name: "Held throughout", UserID: 1, PurchasePrice: money(100), PurchaseDate: &purchased},
		{Name: "Sold in 2025", UserID: 1, PurchasePrice: money(200), PurchaseDate: &purchased, IsSold: true, SoldDate: &soldOn},
		{Name: "Bought in 2026", UserID: 1, PurchasePrice: money(300), PurchaseDate: &laterPurchase},
		{Name: "Wishlist only", UserID: 1, PurchasePrice: money(400), PurchaseDate: &purchased, IsWishlist: true},
	}
	if err := db.Create(&coins).Error; err != nil {
		t.Fatalf("create coins: %v", err)
	}

	// Mid-2024: only the first two exist, neither sold yet.
	snapshot, err := svc.GetSnapshot(1, day(2024, time.June, 1))
	if err != nil {
		t.Fatalf("GetSnapshot: %v", err)
	}
	if snapshot.CoinCount != 2 {
		t.Fatalf("mid-2024 coinCount = %d, want 2", snapshot.CoinCount)
	}
	if snapshot.TotalValue != 300 {
		t.Fatalf("mid-2024 totalValue = %v, want 300", snapshot.TotalValue)
	}

	// Mid-2025: the sold coin has dropped out; the 2026 purchase is not yet made.
	snapshot, err = svc.GetSnapshot(1, day(2025, time.June, 1))
	if err != nil {
		t.Fatalf("GetSnapshot: %v", err)
	}
	if snapshot.CoinCount != 1 {
		t.Fatalf("mid-2025 coinCount = %d, want 1 (sold coin should be gone)", snapshot.CoinCount)
	}

	// Today: the 2026 purchase is included; the sold coin still is not.
	snapshot, err = svc.GetSnapshot(1, day(2026, time.August, 20))
	if err != nil {
		t.Fatalf("GetSnapshot: %v", err)
	}
	if snapshot.CoinCount != 2 {
		t.Fatalf("today coinCount = %d, want 2", snapshot.CoinCount)
	}
}

// A coin bought on exactly the requested date must be included: the user
// scrubbing to "the day I bought it" expects to see it.
func TestTimeMachine_PurchaseDateIsInclusive(t *testing.T) {
	svc, db := setupTimeMachine(t)

	purchased := day(2025, time.July, 4)
	if err := db.Create(&models.Coin{
		Name: "Bought today", UserID: 1, PurchasePrice: money(50), PurchaseDate: &purchased,
	}).Error; err != nil {
		t.Fatalf("create coin: %v", err)
	}

	sameDay, err := svc.GetSnapshot(1, purchased)
	if err != nil {
		t.Fatalf("GetSnapshot: %v", err)
	}
	if sameDay.CoinCount != 1 {
		t.Fatalf("coin bought on the requested date should be included, got count %d", sameDay.CoinCount)
	}

	dayBefore, err := svc.GetSnapshot(1, purchased.AddDate(0, 0, -1))
	if err != nil {
		t.Fatalf("GetSnapshot: %v", err)
	}
	if dayBefore.CoinCount != 0 {
		t.Fatalf("coin should not exist the day before purchase, got count %d", dayBefore.CoinCount)
	}
}

// Value must come from the newest valuation at or before the target date, not
// the coin's current value — otherwise the timeline just replays today's prices.
func TestTimeMachine_UsesHistoricalValuationNotCurrentValue(t *testing.T) {
	svc, db := setupTimeMachine(t)

	purchased := day(2023, time.January, 1)
	coin := models.Coin{
		Name: "Trajan denarius", UserID: 1,
		PurchasePrice: money(100), CurrentValue: money(900), PurchaseDate: &purchased,
	}
	if err := db.Create(&coin).Error; err != nil {
		t.Fatalf("create coin: %v", err)
	}

	history := []models.CoinValueHistory{
		{CoinID: coin.ID, UserID: 1, Value: 300, Confidence: "medium", RecordedAt: day(2024, time.January, 1)},
		{CoinID: coin.ID, UserID: 1, Value: 600, Confidence: "medium", RecordedAt: day(2025, time.January, 1)},
		{CoinID: coin.ID, UserID: 1, Value: 900, Confidence: "medium", RecordedAt: day(2026, time.January, 1)},
	}
	if err := db.Create(&history).Error; err != nil {
		t.Fatalf("create history: %v", err)
	}

	for _, tc := range []struct {
		name        string
		at          time.Time
		wantValue   float64
		fromHistory bool
	}{
		{"before any valuation falls back to purchase price", day(2023, time.June, 1), 100, false},
		{"after first valuation", day(2024, time.June, 1), 300, true},
		{"after second valuation", day(2025, time.June, 1), 600, true},
		{"after third valuation", day(2026, time.June, 1), 900, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			snapshot, err := svc.GetSnapshot(1, tc.at)
			if err != nil {
				t.Fatalf("GetSnapshot: %v", err)
			}
			if snapshot.TotalValue != tc.wantValue {
				t.Fatalf("totalValue = %v, want %v", snapshot.TotalValue, tc.wantValue)
			}
			if len(snapshot.TopCoins) != 1 || snapshot.TopCoins[0].ValueFromHistory != tc.fromHistory {
				t.Fatalf("valueFromHistory = %v, want %v", snapshot.TopCoins[0].ValueFromHistory, tc.fromHistory)
			}
			wantHistoryCount := 0
			if tc.fromHistory {
				wantHistoryCount = 1
			}
			if snapshot.ValueBasis.FromValuationHistory != wantHistoryCount {
				t.Fatalf("valueBasis.fromValuationHistory = %d, want %d",
					snapshot.ValueBasis.FromValuationHistory, wantHistoryCount)
			}
		})
	}
}

// Coins with no purchase date cannot be placed on the timeline. They must be
// excluded from the totals but reported, so the UI can say the numbers are
// partial rather than quietly understating the collection.
func TestTimeMachine_UndatedCoinsAreExcludedButReported(t *testing.T) {
	svc, db := setupTimeMachine(t)

	purchased := day(2024, time.March, 1)
	coins := []models.Coin{
		{Name: "Dated", UserID: 1, PurchasePrice: money(100), PurchaseDate: &purchased},
		{Name: "Undated A", UserID: 1, PurchasePrice: money(500)},
		{Name: "Undated B", UserID: 1, PurchasePrice: money(700)},
	}
	if err := db.Create(&coins).Error; err != nil {
		t.Fatalf("create coins: %v", err)
	}

	snapshot, err := svc.GetSnapshot(1, day(2026, time.January, 1))
	if err != nil {
		t.Fatalf("GetSnapshot: %v", err)
	}
	if snapshot.CoinCount != 1 || snapshot.TotalValue != 100 {
		t.Fatalf("undated coins leaked into totals: count=%d value=%v", snapshot.CoinCount, snapshot.TotalValue)
	}
	if snapshot.UndatedCoinCount != 2 {
		t.Fatalf("undatedCoinCount = %d, want 2", snapshot.UndatedCoinCount)
	}
}

func TestTimeMachine_ScopesToTheRequestingUser(t *testing.T) {
	svc, db := setupTimeMachine(t)

	purchased := day(2024, time.March, 1)
	coins := []models.Coin{
		{Name: "Mine", UserID: 1, PurchasePrice: money(100), PurchaseDate: &purchased},
		{Name: "Theirs", UserID: 2, PurchasePrice: money(9999), PurchaseDate: &purchased},
	}
	if err := db.Create(&coins).Error; err != nil {
		t.Fatalf("create coins: %v", err)
	}

	snapshot, err := svc.GetSnapshot(1, day(2026, time.January, 1))
	if err != nil {
		t.Fatalf("GetSnapshot: %v", err)
	}
	if snapshot.CoinCount != 1 || snapshot.TotalValue != 100 {
		t.Fatalf("another user's coins leaked in: count=%d value=%v", snapshot.CoinCount, snapshot.TotalValue)
	}
}

func TestTimeMachine_RejectsFutureDates(t *testing.T) {
	svc, _ := setupTimeMachine(t)

	if _, err := svc.GetSnapshot(1, day(2026, time.August, 21)); err != ErrTimeMachineFutureDate {
		t.Fatalf("expected ErrTimeMachineFutureDate, got %v", err)
	}
	// Today itself is allowed.
	if _, err := svc.GetSnapshot(1, day(2026, time.August, 20)); err != nil {
		t.Fatalf("today should be allowed, got %v", err)
	}
}

func TestTimeMachine_BreakdownsAndGain(t *testing.T) {
	svc, db := setupTimeMachine(t)

	purchased := day(2024, time.March, 1)
	coins := []models.Coin{
		{Name: "A", UserID: 1, Category: models.CategoryRoman, Material: models.MaterialSilver, Era: models.EraAncient, PurchasePrice: money(100), PurchaseDate: &purchased},
		{Name: "B", UserID: 1, Category: models.CategoryRoman, Material: models.MaterialGold, Era: models.EraAncient, PurchasePrice: money(200), PurchaseDate: &purchased},
		{Name: "C", UserID: 1, Category: models.CategoryGreek, Material: models.MaterialSilver, Era: models.EraAncient, PurchasePrice: money(50), PurchaseDate: &purchased},
	}
	if err := db.Create(&coins).Error; err != nil {
		t.Fatalf("create coins: %v", err)
	}

	snapshot, err := svc.GetSnapshot(1, day(2025, time.January, 1))
	if err != nil {
		t.Fatalf("GetSnapshot: %v", err)
	}

	if len(snapshot.ByCategory) != 2 {
		t.Fatalf("byCategory buckets = %d, want 2", len(snapshot.ByCategory))
	}
	// Ordered by count descending: Roman (2) before Greek (1).
	if snapshot.ByCategory[0].Label != "Roman" || snapshot.ByCategory[0].Count != 2 {
		t.Fatalf("byCategory[0] = %+v, want Roman x2", snapshot.ByCategory[0])
	}
	if snapshot.ByCategory[0].Value != 300 {
		t.Fatalf("Roman value = %v, want 300", snapshot.ByCategory[0].Value)
	}
	if snapshot.TotalInvested != 350 {
		t.Fatalf("totalInvested = %v, want 350", snapshot.TotalInvested)
	}
	// No valuation history, so value == purchase price and gain is zero.
	if snapshot.UnrealizedGain != 0 {
		t.Fatalf("unrealizedGain = %v, want 0", snapshot.UnrealizedGain)
	}
}

func TestTimeMachine_HealthScoreUsesMostRecentPriorSnapshot(t *testing.T) {
	svc, db := setupTimeMachine(t)

	snapshots := []models.CollectionHealthSnapshot{
		{UserID: 1, SnapshotDate: day(2025, time.January, 1), Score: 60},
		{UserID: 1, SnapshotDate: day(2025, time.June, 1), Score: 75},
	}
	if err := db.Create(&snapshots).Error; err != nil {
		t.Fatalf("create snapshots: %v", err)
	}

	before, err := svc.GetSnapshot(1, day(2024, time.December, 1))
	if err != nil {
		t.Fatalf("GetSnapshot: %v", err)
	}
	if before.HealthScore != nil {
		t.Fatalf("expected no health score before the first snapshot, got %v", *before.HealthScore)
	}

	between, err := svc.GetSnapshot(1, day(2025, time.March, 1))
	if err != nil {
		t.Fatalf("GetSnapshot: %v", err)
	}
	if between.HealthScore == nil || *between.HealthScore != 60 {
		t.Fatalf("expected the January score of 60, got %v", between.HealthScore)
	}

	after, err := svc.GetSnapshot(1, day(2025, time.August, 1))
	if err != nil {
		t.Fatalf("GetSnapshot: %v", err)
	}
	if after.HealthScore == nil || *after.HealthScore != 75 {
		t.Fatalf("expected the June score of 75, got %v", after.HealthScore)
	}
}

func TestTimeMachine_Bounds(t *testing.T) {
	svc, db := setupTimeMachine(t)

	empty, err := svc.GetBounds(1)
	if err != nil {
		t.Fatalf("GetBounds: %v", err)
	}
	if empty.HasData {
		t.Fatal("expected hasData=false for a collection with no dated coins")
	}

	first := day(2019, time.May, 4)
	later := day(2023, time.September, 9)
	coins := []models.Coin{
		{Name: "First", UserID: 1, PurchaseDate: &first},
		{Name: "Later", UserID: 1, PurchaseDate: &later},
	}
	if err := db.Create(&coins).Error; err != nil {
		t.Fatalf("create coins: %v", err)
	}

	bounds, err := svc.GetBounds(1)
	if err != nil {
		t.Fatalf("GetBounds: %v", err)
	}
	if !bounds.HasData {
		t.Fatal("expected hasData=true")
	}
	if bounds.EarliestDate != "2019-05-04" {
		t.Fatalf("earliestDate = %q, want 2019-05-04", bounds.EarliestDate)
	}
	// Latest is today, not the last purchase: the user can scrub to the present.
	if bounds.LatestDate != "2026-08-20" {
		t.Fatalf("latestDate = %q, want today (2026-08-20)", bounds.LatestDate)
	}
}
