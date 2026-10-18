package repository

import (
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
)

func TestAttributionCardsExcludedFromPhotoQueries(t *testing.T) {
	db := setupTestDB(t)
	if err := db.AutoMigrate(&models.FeaturedCoin{}, &models.Showcase{}, &models.ShowcaseCoin{}); err != nil {
		t.Fatal(err)
	}
	owner := models.User{Username: "card-owner", Email: "card-owner@example.test", PasswordHash: "x", IsPublic: true}
	if err := db.Create(&owner).Error; err != nil {
		t.Fatal(err)
	}
	one := 1
	tray := models.StorageLocation{UserID: owner.ID, Name: "Tray", Type: models.StorageLocationTypeTray, Rows: &one, Columns: &one}
	if err := db.Create(&tray).Error; err != nil {
		t.Fatal(err)
	}
	coin := models.Coin{Name: "Coin with private card", UserID: owner.ID, StorageLocationID: &tray.ID, StorageSlot: &one}
	if err := db.Create(&coin).Error; err != nil {
		t.Fatal(err)
	}
	// A legacy primary card must not win fallback selection or count as a face.
	card := models.CoinImage{CoinID: coin.ID, ImageType: models.ImageTypeCard, FilePath: "coin-1/original.png", IsPrimary: true}
	photo := models.CoinImage{CoinID: coin.ID, ImageType: models.ImageTypeDetail, FilePath: "coin-1/detail.png"}
	foreignPhoto := models.CoinImage{CoinID: coin.ID + 999, ImageType: models.ImageTypeOther, FilePath: "coin-999/other.png"}
	for _, record := range []*models.CoinImage{&card, &photo, &foreignPhoto} {
		if err := db.Create(record).Error; err != nil {
			t.Fatal(err)
		}
	}
	featured := models.FeaturedCoin{UserID: owner.ID, CoinID: coin.ID, FeaturedAt: time.Now()}
	if err := db.Create(&featured).Error; err != nil {
		t.Fatal(err)
	}
	showcase := models.Showcase{UserID: owner.ID, Slug: "private-card-test", Title: "Showcase", IsActive: true}
	if err := db.Create(&showcase).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.ShowcaseCoin{ShowcaseID: showcase.ID, CoinID: coin.ID}).Error; err != nil {
		t.Fatal(err)
	}
	set := models.CoinSet{UserID: owner.ID, Name: "Card test"}
	if err := db.Create(&set).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.CoinSetMembership{SetID: set.ID, CoinID: coin.ID, AddedAt: time.Now()}).Error; err != nil {
		t.Fatal(err)
	}
	assertPhotos := func(t *testing.T, images []models.CoinImage, err error) {
		t.Helper()
		if err != nil || len(images) != 1 || images[0].ID != photo.ID {
			t.Fatalf("expected only the owned coin photo, got %+v: %v", images, err)
		}
	}
	for _, test := range []struct {
		name string
		load func() (*models.Coin, error)
	}{
		{"owner detail", func() (*models.Coin, error) { return NewCoinRepository(db).FindByID(coin.ID, owner.ID) }},
		{"follower detail", func() (*models.Coin, error) { return NewSocialRepository(db).FindPublicCoin(coin.ID, owner.ID) }},
		{"analysis", func() (*models.Coin, error) { return NewAnalysisRepository(db).FindCoinWithImages(coin.ID, owner.ID) }},
		{"AI job", func() (*models.Coin, error) { return NewAIJobRepository(db).FindCoinWithImages(coin.ID, owner.ID) }},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := test.load()
			if err != nil {
				t.Fatal(err)
			}
			assertPhotos(t, got.Images, nil)
		})
	}
	fc, err := NewFeaturedCoinRepository(db).GetLatestForUser(owner.ID)
	if err != nil {
		t.Fatal(err)
	}
	assertPhotos(t, fc.Coin.Images, nil)
	public, err := NewSocialRepository(db).GetPublicCoins(owner.ID)
	if err != nil || len(public) != 1 {
		t.Fatalf("public coins: %v %+v", err, public)
	}
	assertPhotos(t, public[0].Images, nil)
	shown, err := NewShowcaseRepository(db).GetShowcaseCoins(showcase.ID, owner.ID)
	if err != nil || len(shown) != 1 {
		t.Fatalf("showcase coins: %v %+v", err, shown)
	}
	assertPhotos(t, shown[0].Images, nil)
	members, err := NewSetRepository(db).GetCoinsInSet(set.ID, owner.ID)
	if err != nil || len(members) != 1 {
		t.Fatalf("set coins: %v %+v", err, members)
	}
	assertPhotos(t, members[0].Images, nil)
	exported, err := NewUserRepository(db).GetCoinsWithImages(owner.ID, true)
	if err != nil || len(exported) != 1 || len(exported[0].Images) != 2 {
		t.Fatalf("owner export must retain photos AND originals: %v %+v", err, exported)
	}
	catalog, err := NewUserRepository(db).GetCoinsWithImages(owner.ID, false)
	if err != nil || len(catalog) != 1 {
		t.Fatalf("owner catalog: %v %+v", err, catalog)
	}
	assertPhotos(t, catalog[0].Images, nil)
	for _, withPhoto := range []bool{true, false} {
		if !withPhoto {
			if err := db.Delete(&photo).Error; err != nil {
				t.Fatal(err)
			}
		}
		quick, err := NewQuickAccessRepository(db).FindCoins(owner.ID, []uint{coin.ID})
		if err != nil || len(quick) != 1 {
			t.Fatalf("quick access: %v %+v", err, quick)
		}
		expectedPath := ""
		expectedCount := int64(0)
		if withPhoto {
			expectedPath = photo.FilePath
			expectedCount = 1
		}
		if quick[0].PrimaryImageURL != expectedPath {
			t.Fatalf("quick access selected an original: %+v", quick[0])
		}
		health, err := NewHealthRepository(db).GetSingleEligibleCoin(coin.ID, owner.ID)
		if err != nil || health.ImageCount != expectedCount || health.PrimaryImageCount != 0 {
			t.Fatalf("health must count coin photos only: %v %+v", err, health)
		}
		trays, err := NewStorageLocationRepository(db).ListTrayAggregates(owner.ID)
		if err != nil || len(trays) != 1 || len(trays[0].Coins) != 1 {
			t.Fatalf("tray query: %v %+v", err, trays)
		}
		if int64(len(trays[0].Coins[0].Images)) != expectedCount {
			t.Fatalf("tray must not display a card-only coin as photographed: %+v", trays[0].Coins[0].Images)
		}
		if withPhoto && trays[0].Coins[0].Images[0].FilePath != photo.FilePath {
			t.Fatal("tray fallback selected a private card")
		}
	}
}
