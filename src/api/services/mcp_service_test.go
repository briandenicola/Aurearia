package services

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func newMCPServiceReadTest(t *testing.T) (*gorm.DB, *MCPService) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=private"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.Coin{}, &models.AuctionLot{}); err != nil {
		t.Fatal(err)
	}
	coinRepo := repository.NewCoinRepository(db)
	auctionRepo := repository.NewAuctionLotRepository(db)
	collectionSvc := NewCollectionToolsService(coinRepo, nil)
	return db, NewMCPService(collectionSvc, coinRepo, auctionRepo, nil, nil)
}

func TestMCPServiceScopesWishlistAndAuctionsToOwner(t *testing.T) {
	db, service := newMCPServiceReadTest(t)
	ownerWishlist := models.Coin{UserID: 1, Name: "Owner wishlist", IsWishlist: true}
	foreignWishlist := models.Coin{UserID: 2, Name: "Foreign wishlist", IsWishlist: true}
	ownerCollection := models.Coin{UserID: 1, Name: "Owner collection"}
	if err := db.Create(&[]*models.Coin{&ownerWishlist, &foreignWishlist, &ownerCollection}).Error; err != nil {
		t.Fatal(err)
	}
	ownerLot := models.AuctionLot{UserID: 1, Title: "Owner lot", NumisBidsURL: "https://example.test/1"}
	foreignLot := models.AuctionLot{UserID: 2, Title: "Foreign lot", NumisBidsURL: "https://example.test/2"}
	if err := db.Create(&[]*models.AuctionLot{&ownerLot, &foreignLot}).Error; err != nil {
		t.Fatal(err)
	}

	wishlist, err := service.ListWishlist(1, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(wishlist) != 1 || wishlist[0].ID != ownerWishlist.ID {
		t.Fatalf("wishlist=%+v, want only owner coin %d", wishlist, ownerWishlist.ID)
	}
	collection, err := service.SearchCollection(1, "", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(collection.Coins) != 1 || collection.Coins[0].ID != ownerCollection.ID || collection.Total != 1 {
		t.Fatalf("collection=%+v, want only active owner coin %d", collection, ownerCollection.ID)
	}
	lots, err := service.ListAuctionLots(1, MCPAuctionListInput{})
	if err != nil {
		t.Fatal(err)
	}
	if len(lots.Lots) != 1 || lots.Lots[0].ID != ownerLot.ID || lots.Total != 1 {
		t.Fatalf("lots=%+v, want only owner lot %d", lots, ownerLot.ID)
	}
	if _, err := service.GetCoin(1, foreignWishlist.ID); !errors.Is(err, ErrCoinNotFound) {
		t.Fatalf("foreign coin err=%v, want ErrCoinNotFound", err)
	}
	if _, err := service.GetAuctionLot(1, foreignLot.ID); !repository.IsRecordNotFound(err) {
		t.Fatalf("foreign lot err=%v, want record not found", err)
	}
}

func TestMCPServiceValidatesAuctionFiltersAndBounds(t *testing.T) {
	_, service := newMCPServiceReadTest(t)
	for _, input := range []MCPAuctionListInput{
		{Status: "deleted"},
		{Source: "other"},
		{Page: -1},
		{Limit: 51},
		{Search: string(make([]byte, 201))},
	} {
		if _, err := service.ListAuctionLots(1, input); !errors.Is(err, ErrMCPInvalidRequest) {
			t.Fatalf("input=%+v err=%v, want ErrMCPInvalidRequest", input, err)
		}
	}
	if _, err := service.AuctionCounts(1, "other"); !errors.Is(err, ErrMCPInvalidRequest) {
		t.Fatalf("counts err=%v, want ErrMCPInvalidRequest", err)
	}
	zero := 0
	if _, err := service.SearchCollection(1, "", &zero, nil); !errors.Is(err, ErrMCPInvalidRequest) {
		t.Fatalf("search limit 0 err=%v, want ErrMCPInvalidRequest", err)
	}
	if _, err := service.TopCoinsByValue(1, &zero); !errors.Is(err, ErrMCPInvalidRequest) {
		t.Fatalf("top coins limit 0 err=%v, want ErrMCPInvalidRequest", err)
	}
	if _, err := service.ListWishlist(1, string(make([]byte, 201)), nil); !errors.Is(err, ErrMCPInvalidRequest) {
		t.Fatalf("wishlist query err=%v, want ErrMCPInvalidRequest", err)
	}
}

func TestMCPServiceCapsOversizedLimitsInsteadOfRejecting(t *testing.T) {
	db, service := newMCPServiceReadTest(t)
	coins := make([]*models.Coin, 0, 85)
	for i := 0; i < 25; i++ {
		value := float64(i + 1)
		coins = append(coins, &models.Coin{UserID: 1, Name: fmt.Sprintf("Collection %02d", i), CurrentValue: &value})
	}
	for i := 0; i < 60; i++ {
		coins = append(coins, &models.Coin{UserID: 1, Name: fmt.Sprintf("Wishlist %02d", i), IsWishlist: true})
	}
	if err := db.Create(&coins).Error; err != nil {
		t.Fatal(err)
	}

	generous := 500
	collection, err := service.SearchCollection(1, "", &generous, nil)
	if err != nil || len(collection.Coins) != 20 || collection.Total != 25 {
		t.Fatalf("search page=%+v err=%v, want 20 capped results of 25", collection, err)
	}
	wishlist, err := service.ListWishlist(1, "", &generous)
	if err != nil || len(wishlist) != 50 {
		t.Fatalf("wishlist len=%d err=%v, want 50 capped results", len(wishlist), err)
	}
	top, err := service.TopCoinsByValue(1, &generous)
	if err != nil || len(top) != 10 {
		t.Fatalf("top coins len=%d err=%v, want 10 capped results", len(top), err)
	}
}

func TestMCPServiceSearchPagesThroughAllMatchesWithTotal(t *testing.T) {
	db, service := newMCPServiceReadTest(t)
	coins := make([]*models.Coin, 0, 46)
	for i := 0; i < 45; i++ {
		coins = append(coins, &models.Coin{UserID: 1, Name: fmt.Sprintf("Denarius %02d", i)})
	}
	coins = append(coins, &models.Coin{UserID: 2, Name: "Foreign denarius"})
	if err := db.Create(&coins).Error; err != nil {
		t.Fatal(err)
	}

	limit := 20
	seen := map[uint]bool{}
	for _, want := range []struct{ offset, size int }{{0, 20}, {20, 20}, {40, 5}, {45, 0}} {
		offset := want.offset
		page, err := service.SearchCollection(1, "denarius", &limit, &offset)
		if err != nil {
			t.Fatal(err)
		}
		if page.Total != 45 || page.Offset != offset || page.Limit != 20 || len(page.Coins) != want.size {
			t.Fatalf("offset %d page total=%d offset=%d limit=%d len=%d, want 45/%d/20/%d",
				offset, page.Total, page.Offset, page.Limit, len(page.Coins), offset, want.size)
		}
		for _, coin := range page.Coins {
			if seen[coin.ID] {
				t.Fatalf("coin %d repeated across pages", coin.ID)
			}
			seen[coin.ID] = true
		}
	}
	if len(seen) != 45 {
		t.Fatalf("paged %d distinct coins, want 45", len(seen))
	}

	for _, bad := range []int{-1, mcpMaxSearchOffset + 1} {
		offset := bad
		if _, err := service.SearchCollection(1, "", nil, &offset); !errors.Is(err, ErrMCPInvalidRequest) {
			t.Fatalf("offset %d err=%v, want ErrMCPInvalidRequest", bad, err)
		}
	}
}

func TestMCPServiceListAuctionLotsUpcomingFiltersAndSortsByEnd(t *testing.T) {
	db, service := newMCPServiceReadTest(t)
	now := time.Date(2026, 10, 31, 12, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }
	at := func(d time.Duration) *time.Time { v := now.Add(d); return &v }

	lots := []*models.AuctionLot{
		{UserID: 1, Title: "ends in 3d", NumisBidsURL: "u1", Status: models.AuctionStatusBidding, AuctionEndTime: at(72 * time.Hour)},
		{UserID: 1, Title: "ended yesterday", NumisBidsURL: "u2", Status: models.AuctionStatusBidding, AuctionEndTime: at(-24 * time.Hour)},
		// auction_end_time wins over sale_date: sale date is future but the lot already closed.
		{UserID: 1, Title: "closed, future sale date", NumisBidsURL: "u3", Status: models.AuctionStatusBidding, SaleDate: at(48 * time.Hour), AuctionEndTime: at(-time.Hour)},
		{UserID: 1, Title: "sale date in 1d", NumisBidsURL: "u4", Status: models.AuctionStatusBidding, SaleDate: at(24 * time.Hour)},
		{UserID: 1, Title: "ends in 1h", NumisBidsURL: "u5", Status: models.AuctionStatusBidding, AuctionEndTime: at(time.Hour)},
		{UserID: 1, Title: "no dates", NumisBidsURL: "u6", Status: models.AuctionStatusBidding},
		{UserID: 1, Title: "watching, ends in 2h", NumisBidsURL: "u7", Status: models.AuctionStatusWatching, AuctionEndTime: at(2 * time.Hour)},
		{UserID: 2, Title: "foreign, ends in 30m", NumisBidsURL: "u8", Status: models.AuctionStatusBidding, AuctionEndTime: at(30 * time.Minute)},
	}
	if err := db.Create(&lots).Error; err != nil {
		t.Fatal(err)
	}

	result, err := service.ListAuctionLots(1, MCPAuctionListInput{Status: string(models.AuctionStatusBidding), Upcoming: true})
	if err != nil {
		t.Fatal(err)
	}
	got := make([]string, 0, len(result.Lots))
	for _, lot := range result.Lots {
		got = append(got, lot.Title)
	}
	want := []string{"ends in 1h", "sale date in 1d", "ends in 3d"}
	if fmt.Sprint(got) != fmt.Sprint(want) || result.Total != int64(len(want)) {
		t.Fatalf("got %v (total %d), want %v", got, result.Total, want)
	}

	// Without upcoming the existing behavior is unchanged: every owned bidding lot.
	all, err := service.ListAuctionLots(1, MCPAuctionListInput{Status: string(models.AuctionStatusBidding)})
	if err != nil {
		t.Fatal(err)
	}
	if all.Total != 6 {
		t.Fatalf("total without upcoming = %d, want 6", all.Total)
	}

	// Pagination over the upcoming ordering is stable.
	page2, err := service.ListAuctionLots(1, MCPAuctionListInput{Status: string(models.AuctionStatusBidding), Upcoming: true, Page: 2, Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(page2.Lots) != 1 || page2.Lots[0].Title != "ends in 3d" {
		t.Fatalf("page 2 = %+v, want only 'ends in 3d'", page2.Lots)
	}
}
