package services

import (
	"sync/atomic"
	"testing"

	"github.com/briandenicola/ancient-coins-api/models"
	"gorm.io/gorm"
)

func listSetsValue(v float64) *float64 { return &v }

// seedListSetsFixture creates one set of each type for user 1, each holding the
// same three coins, so the batched and per-set paths can be compared directly.
func seedListSetsFixture(t *testing.T, db *gorm.DB) map[models.CoinSetType]uint {
	t.Helper()

	coins := []models.Coin{
		{Name: "Trajan", UserID: 1, CurrentValue: listSetsValue(300), PurchasePrice: listSetsValue(200)},
		{Name: "Augustus", UserID: 1, CurrentValue: listSetsValue(500), PurchasePrice: listSetsValue(450)},
		// No CurrentValue: SQL AVG() skips this row, which is exactly the
		// semantic difference the batch query has to preserve.
		{Name: "Hadrian", UserID: 1, PurchasePrice: listSetsValue(75)},
	}
	if err := db.Create(&coins).Error; err != nil {
		t.Fatalf("create coins: %v", err)
	}

	ids := map[models.CoinSetType]uint{}
	for _, setType := range []models.CoinSetType{
		models.CoinSetTypeStandard,
		models.CoinSetTypeGoal,
		models.CoinSetTypeAgentic,
	} {
		set := models.CoinSet{UserID: 1, Name: string(setType) + " set", SetType: setType}
		if err := db.Create(&set).Error; err != nil {
			t.Fatalf("create %s set: %v", setType, err)
		}
		for _, coin := range coins {
			membership := models.CoinSetMembership{SetID: set.ID, CoinID: coin.ID}
			if err := db.Create(&membership).Error; err != nil {
				t.Fatalf("create membership: %v", err)
			}
		}
		ids[setType] = set.ID
	}
	return ids
}

// The batched summary must agree with the per-set GetSetSummary it replaced,
// including AVG() ignoring NULL current_value and the highest-value coin id.
func TestListSets_BatchedSummaryMatchesPerSetSummary(t *testing.T) {
	service, db := setupSetServiceTestWithDB(t)
	ids := seedListSetsFixture(t, db)

	listed, err := service.ListSets(1)
	if err != nil {
		t.Fatalf("ListSets failed: %v", err)
	}

	byID := make(map[uint]map[string]interface{}, len(listed))
	for _, entry := range listed {
		byID[entry["id"].(uint)] = entry
	}

	for setType, setID := range ids {
		want, err := service.repo.GetSetSummary(setID, 1)
		if err != nil {
			t.Fatalf("%s: GetSetSummary failed: %v", setType, err)
		}
		got := byID[setID]
		if got == nil {
			t.Fatalf("%s: set %d missing from ListSets output", setType, setID)
		}

		if got["coinCount"] != want["coinCount"] {
			t.Errorf("%s coinCount: batched %v, per-set %v", setType, got["coinCount"], want["coinCount"])
		}
		if got["totalValue"] != want["totalValue"] {
			t.Errorf("%s totalValue: batched %v, per-set %v", setType, got["totalValue"], want["totalValue"])
		}
	}
}

// Regression guard for the N+1 this replaced: query count must not scale with
// the number of sets.
func TestListSets_QueryCountDoesNotScaleWithSetCount(t *testing.T) {
	countQueries := func(t *testing.T, setCount int) int64 {
		t.Helper()
		service, db := setupSetServiceTestWithDB(t)

		coin := models.Coin{Name: "Denarius", UserID: 1, CurrentValue: listSetsValue(100)}
		if err := db.Create(&coin).Error; err != nil {
			t.Fatalf("create coin: %v", err)
		}
		for i := 0; i < setCount; i++ {
			set := models.CoinSet{UserID: 1, Name: string(rune('A'+i)) + " set", SetType: models.CoinSetTypeStandard}
			if err := db.Create(&set).Error; err != nil {
				t.Fatalf("create set: %v", err)
			}
			if err := db.Create(&models.CoinSetMembership{SetID: set.ID, CoinID: coin.ID}).Error; err != nil {
				t.Fatalf("create membership: %v", err)
			}
		}

		var queries int64
		cb := db.Callback().Raw().After("gorm:raw")
		if err := cb.Register("test:count_raw", func(*gorm.DB) { atomic.AddInt64(&queries, 1) }); err != nil {
			t.Fatalf("register raw callback: %v", err)
		}
		qcb := db.Callback().Query().After("gorm:query")
		if err := qcb.Register("test:count_query", func(*gorm.DB) { atomic.AddInt64(&queries, 1) }); err != nil {
			t.Fatalf("register query callback: %v", err)
		}

		if _, err := service.ListSets(1); err != nil {
			t.Fatalf("ListSets failed: %v", err)
		}
		return atomic.LoadInt64(&queries)
	}

	small := countQueries(t, 2)
	large := countQueries(t, 12)

	// Ten extra sets previously meant ~30 extra queries. Allow a little slack
	// for unrelated per-call queries, but nothing proportional to set count.
	if large > small+3 {
		t.Fatalf("query count scales with set count: %d sets -> %d queries, %d sets -> %d queries",
			2, small, 12, large)
	}
	t.Logf("queries: 2 sets = %d, 12 sets = %d", small, large)
}
