package services

import (
	"errors"
	"fmt"
	"testing"

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
	collection, err := service.SearchCollection(1, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(collection) != 1 || collection[0].ID != ownerCollection.ID {
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
	if _, err := service.SearchCollection(1, "", &zero); !errors.Is(err, ErrMCPInvalidRequest) {
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
	collection, err := service.SearchCollection(1, "", &generous)
	if err != nil || len(collection) != 20 {
		t.Fatalf("search len=%d err=%v, want 20 capped results", len(collection), err)
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
