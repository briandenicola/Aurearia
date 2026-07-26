package database

import (
	"testing"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupCoinMintLocationBackfillDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.AppSetting{}, &models.MintLocation{}, &models.Coin{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	return db
}

func TestBackfillCoinMintLocations_LinksMatchingGlobalMint(t *testing.T) {
	db := setupCoinMintLocationBackfillDB(t)

	global := models.MintLocation{
		DisplayName:    "Rome",
		NormalizedName: models.NormalizeMintLocationName("Rome"),
		Lat:            41.9,
		Lng:            12.5,
		Aliases:        models.StringList{"Roma"},
	}
	if err := db.Create(&global).Error; err != nil {
		t.Fatalf("seed mint location failed: %v", err)
	}

	exactMatch := models.Coin{Name: "Denarius", UserID: 1, Mint: "Rome"}
	aliasMatch := models.Coin{Name: "As", UserID: 1, Mint: "  roma! "}
	noMatch := models.Coin{Name: "Drachm", UserID: 1, Mint: "Unknown Mint City"}
	blank := models.Coin{Name: "Blank", UserID: 1, Mint: ""}
	for _, c := range []*models.Coin{&exactMatch, &aliasMatch, &noMatch, &blank} {
		if err := db.Create(c).Error; err != nil {
			t.Fatalf("seed coin failed: %v", err)
		}
	}

	if err := backfillCoinMintLocations(db); err != nil {
		t.Fatalf("backfill failed: %v", err)
	}

	var reloadedExact models.Coin
	if err := db.First(&reloadedExact, exactMatch.ID).Error; err != nil {
		t.Fatalf("reload failed: %v", err)
	}
	if reloadedExact.MintLocationID == nil || *reloadedExact.MintLocationID != global.ID {
		t.Fatalf("expected exact-match coin linked to %d, got %+v", global.ID, reloadedExact.MintLocationID)
	}

	var reloadedAlias models.Coin
	if err := db.First(&reloadedAlias, aliasMatch.ID).Error; err != nil {
		t.Fatalf("reload failed: %v", err)
	}
	if reloadedAlias.MintLocationID == nil || *reloadedAlias.MintLocationID != global.ID {
		t.Fatalf("expected alias-match coin linked to %d, got %+v", global.ID, reloadedAlias.MintLocationID)
	}

	var reloadedNoMatch models.Coin
	if err := db.First(&reloadedNoMatch, noMatch.ID).Error; err != nil {
		t.Fatalf("reload failed: %v", err)
	}
	if reloadedNoMatch.MintLocationID != nil {
		t.Fatalf("expected no-match coin to remain unlinked, got %+v", reloadedNoMatch.MintLocationID)
	}

	var reloadedBlank models.Coin
	if err := db.First(&reloadedBlank, blank.ID).Error; err != nil {
		t.Fatalf("reload failed: %v", err)
	}
	if reloadedBlank.MintLocationID != nil {
		t.Fatalf("expected blank-mint coin to remain unlinked, got %+v", reloadedBlank.MintLocationID)
	}
}

func TestBackfillCoinMintLocations_DoesNotCrossLinkAnotherUsersPrivateMint(t *testing.T) {
	db := setupCoinMintLocationBackfillDB(t)

	ownerID := uint(2)
	private := models.MintLocation{
		UserID:         &ownerID,
		DisplayName:    "Custom Mint",
		NormalizedName: models.NormalizeMintLocationName("Custom Mint"),
		Lat:            1,
		Lng:            1,
		Aliases:        models.StringList{},
	}
	if err := db.Create(&private).Error; err != nil {
		t.Fatalf("seed private mint failed: %v", err)
	}

	otherUsersCoin := models.Coin{Name: "Coin", UserID: 1, Mint: "Custom Mint"}
	if err := db.Create(&otherUsersCoin).Error; err != nil {
		t.Fatalf("seed coin failed: %v", err)
	}

	if err := backfillCoinMintLocations(db); err != nil {
		t.Fatalf("backfill failed: %v", err)
	}

	var reloaded models.Coin
	if err := db.First(&reloaded, otherUsersCoin.ID).Error; err != nil {
		t.Fatalf("reload failed: %v", err)
	}
	if reloaded.MintLocationID != nil {
		t.Fatalf("expected coin not to link to another user's private mint, got %+v", reloaded.MintLocationID)
	}
}

func TestBackfillCoinMintLocations_LinksOwnersPrivateMint(t *testing.T) {
	db := setupCoinMintLocationBackfillDB(t)

	ownerID := uint(1)
	private := models.MintLocation{
		UserID:         &ownerID,
		DisplayName:    "Custom Mint",
		NormalizedName: models.NormalizeMintLocationName("Custom Mint"),
		Lat:            1,
		Lng:            1,
		Aliases:        models.StringList{},
	}
	if err := db.Create(&private).Error; err != nil {
		t.Fatalf("seed private mint failed: %v", err)
	}

	ownersCoin := models.Coin{Name: "Coin", UserID: 1, Mint: "Custom Mint"}
	if err := db.Create(&ownersCoin).Error; err != nil {
		t.Fatalf("seed coin failed: %v", err)
	}

	if err := backfillCoinMintLocations(db); err != nil {
		t.Fatalf("backfill failed: %v", err)
	}

	var reloaded models.Coin
	if err := db.First(&reloaded, ownersCoin.ID).Error; err != nil {
		t.Fatalf("reload failed: %v", err)
	}
	if reloaded.MintLocationID == nil || *reloaded.MintLocationID != private.ID {
		t.Fatalf("expected coin linked to owner's own private mint %d, got %+v", private.ID, reloaded.MintLocationID)
	}
}

func TestBackfillCoinMintLocations_IdempotentAndVersioned(t *testing.T) {
	db := setupCoinMintLocationBackfillDB(t)

	global := models.MintLocation{
		DisplayName:    "Rome",
		NormalizedName: models.NormalizeMintLocationName("Rome"),
		Lat:            41.9,
		Lng:            12.5,
		Aliases:        models.StringList{},
	}
	if err := db.Create(&global).Error; err != nil {
		t.Fatalf("seed mint location failed: %v", err)
	}
	coin := models.Coin{Name: "Denarius", UserID: 1, Mint: "Rome"}
	if err := db.Create(&coin).Error; err != nil {
		t.Fatalf("seed coin failed: %v", err)
	}

	if err := backfillCoinMintLocations(db); err != nil {
		t.Fatalf("first backfill failed: %v", err)
	}

	// Simulate a user manually unlinking the coin afterward - a second run
	// (same version) must not relink it, since the migration only runs once.
	if err := db.Model(&models.Coin{}).Where("id = ?", coin.ID).Update("mint_location_id", nil).Error; err != nil {
		t.Fatalf("manual unlink failed: %v", err)
	}

	if err := backfillCoinMintLocations(db); err != nil {
		t.Fatalf("second backfill failed: %v", err)
	}

	var reloaded models.Coin
	if err := db.First(&reloaded, coin.ID).Error; err != nil {
		t.Fatalf("reload failed: %v", err)
	}
	if reloaded.MintLocationID != nil {
		t.Fatalf("expected manually-unlinked coin to stay unlinked after a repeat run, got %+v", reloaded.MintLocationID)
	}
}
