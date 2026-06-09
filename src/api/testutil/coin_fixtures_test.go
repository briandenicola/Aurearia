package testutil

import (
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestGoldenCoinFixtureCatalogCoversRequiredTraits(t *testing.T) {
	catalog := GoldenCoinFixtureCatalog()
	if len(catalog) != len(GoldenCoinFixtureNames()) {
		t.Fatalf("expected %d catalog entries, got %d", len(GoldenCoinFixtureNames()), len(catalog))
	}

	required := []GoldenCoinTrait{
		TraitRoman,
		TraitGreek,
		TraitByzantine,
		TraitWishlist,
		TraitSold,
		TraitPrivate,
		TraitTagged,
		TraitSetMember,
		TraitStorageLocation,
		TraitImageHeavy,
		TraitValued,
		TraitReferenceRich,
		TraitLegacyCustomEra,
	}
	seen := map[GoldenCoinTrait]bool{}
	for _, info := range catalog {
		for _, trait := range info.Traits {
			seen[trait] = true
		}
	}
	for _, trait := range required {
		if !seen[trait] {
			t.Fatalf("missing required golden fixture trait %q", trait)
		}
	}
}

func TestBuildGoldenCoinFixtureReturnsIndependentClones(t *testing.T) {
	first := BuildTaggedFollisStorage(42)
	second := BuildTaggedFollisStorage(42)

	first.Name = "mutated"
	first.Tags[0].Name = "mutated tag"
	first.StorageLocation.Name = "mutated storage"
	*first.WeightGrams = 99

	if second.Name == "mutated" {
		t.Fatal("coin name mutation leaked between fixture builds")
	}
	if second.Tags[0].Name == "mutated tag" {
		t.Fatal("tag mutation leaked between fixture builds")
	}
	if second.StorageLocation.Name == "mutated storage" {
		t.Fatal("storage mutation leaked between fixture builds")
	}
	if *second.WeightGrams == 99 {
		t.Fatal("pointer field mutation leaked between fixture builds")
	}
}

func TestPersistGoldenCollectionSeedsAssociations(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:golden_fixtures?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	if err := db.AutoMigrate(
		&models.User{}, &models.StorageLocation{}, &models.Coin{}, &models.CoinImage{}, &models.CoinReference{},
		&models.Tag{}, &models.CoinTag{}, &models.CoinSet{}, &models.CoinSetMembership{},
	); err != nil {
		t.Fatalf("failed to migrate test db: %v", err)
	}
	user := models.User{ID: 42, Username: "fixture-user", PasswordHash: "hash", Email: "fixture-user@test.example"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("failed to seed user: %v", err)
	}

	collection, err := PersistGoldenCollection(db, user.ID)
	if err != nil {
		t.Fatalf("PersistGoldenCollection failed: %v", err)
	}
	if len(collection.Coins) != len(GoldenCoinFixtureNames()) {
		t.Fatalf("expected %d persisted coins, got %d", len(GoldenCoinFixtureNames()), len(collection.Coins))
	}

	var tagged, imageHeavy, referenceRich models.Coin
	for _, coin := range collection.Coins {
		switch coin.Name {
		case "Diocletian Follis Tagged Storage":
			tagged = coin
		case "Syracuse Drachm Image Heavy":
			imageHeavy = coin
		case "Vespasian Denarius Reference Rich":
			referenceRich = coin
		}
	}
	if tagged.ID == 0 || tagged.StorageLocation == nil || len(tagged.Tags) != 2 {
		t.Fatalf("expected tagged storage fixture to preload storage and two tags, got id=%d storage=%v tags=%d", tagged.ID, tagged.StorageLocation, len(tagged.Tags))
	}
	if len(imageHeavy.Images) != 4 {
		t.Fatalf("expected image-heavy fixture to persist four images, got %d", len(imageHeavy.Images))
	}
	if len(referenceRich.References) != 3 || len(referenceRich.Sets) != 1 {
		t.Fatalf("expected reference-rich fixture to persist three references and one set, got refs=%d sets=%d", len(referenceRich.References), len(referenceRich.Sets))
	}

	var membership models.CoinSetMembership
	if err := db.Where("coin_id = ?", referenceRich.ID).First(&membership).Error; err != nil {
		t.Fatalf("expected reference-rich membership: %v", err)
	}
	if membership.AddedAt.IsZero() || !membership.AddedAt.Equal(time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)) {
		t.Fatalf("expected deterministic membership AddedAt, got %v", membership.AddedAt)
	}
}
