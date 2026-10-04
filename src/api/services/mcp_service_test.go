package services

import (
	"errors"
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
	return db, NewMCPService(collectionSvc, coinRepo, auctionRepo, nil)
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
	tooMany := 21
	if _, err := service.SearchCollection(1, "", &tooMany); !errors.Is(err, ErrMCPInvalidRequest) {
		t.Fatalf("search limit err=%v, want ErrMCPInvalidRequest", err)
	}
	if _, err := service.ListWishlist(1, string(make([]byte, 201)), nil); !errors.Is(err, ErrMCPInvalidRequest) {
		t.Fatalf("wishlist query err=%v, want ErrMCPInvalidRequest", err)
	}
}
