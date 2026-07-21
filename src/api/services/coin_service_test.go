package services

import (
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

var testDBCounter uint64

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dbName := fmt.Sprintf("file:coin_service_%d_%d?mode=memory&cache=shared", time.Now().UnixNano(), atomic.AddUint64(&testDBCounter, 1))
	db, err := gorm.Open(sqlite.Open(dbName), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	err = db.AutoMigrate(
		&models.User{}, &models.Coin{}, &models.CoinImage{}, &models.CoinReference{}, &models.CatalogRegistry{}, &models.AppSetting{},
		&models.MintLocation{},
		&models.StorageLocation{}, &models.ValueSnapshot{}, &models.CoinJournal{},
		&models.CoinValueHistory{}, &models.CoinComment{},
		&models.AvailabilityResult{}, &models.AuctionLot{},
		&models.Tag{}, &models.CoinTag{},
		&models.CoinSet{}, &models.CoinSetMembership{},
		&models.Showcase{}, &models.ShowcaseCoin{},
	)
	if err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	return db
}

func newTestCoinService(db *gorm.DB) *CoinService {
	repo := repository.NewCoinRepository(db)
	return NewCoinService(repo, nil)
}

func newTestCoinServiceWithCatalogRegistry(db *gorm.DB) *CoinService {
	repo := repository.NewCoinRepository(db)
	catalogRepo := repository.NewCatalogRegistryRepository(db)
	return NewCoinService(repo, nil).WithCatalogRegistrySupport(catalogRepo)
}

func newTestCoinServiceWithSettings(db *gorm.DB) *CoinService {
	repo := repository.NewCoinRepository(db)
	settingsRepo := repository.NewSettingsRepository(db)
	return NewCoinService(repo, nil).WithSettingsSupport(NewSettingsService(settingsRepo))
}

func newTestCoinServiceWithStorage(db *gorm.DB) *CoinService {
	repo := repository.NewCoinRepository(db)
	storageRepo := repository.NewStorageLocationRepository(db)
	return NewCoinService(repo, nil).WithStorageLocationSupport(storageRepo)
}

func newTestCoinServiceWithReferences(db *gorm.DB) *CoinService {
	repo := repository.NewCoinRepository(db)
	catalogRepo := repository.NewCatalogRegistryRepository(db)
	refRepo := repository.NewCoinReferenceRepository(db)
	refSvc := NewCoinReferenceService(refRepo, catalogRepo)
	return NewCoinService(repo, nil).WithReferenceSupport(refRepo, refSvc)
}

func ptrFloat(v float64) *float64 { return &v }

func TestCreateCoin_Success(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestCoinService(db)

	coin := &models.Coin{
		Name:          "Denarius of Augustus",
		Category:      models.CategoryRoman,
		Material:      models.MaterialSilver,
		PurchasePrice: ptrFloat(250.0),
		CurrentValue:  ptrFloat(300.0),
		UserID:        1,
	}

	if err := svc.CreateCoin(coin); err != nil {
		t.Fatalf("CreateCoin failed: %v", err)
	}

	if coin.ID == 0 {
		t.Fatal("expected coin ID to be set after create")
	}

	// Verify coin exists in DB
	var found models.Coin
	if err := db.First(&found, coin.ID).Error; err != nil {
		t.Fatalf("coin not found in DB: %v", err)
	}
	if found.Name != "Denarius of Augustus" {
		t.Errorf("expected name 'Denarius of Augustus', got %q", found.Name)
	}

	// Verify value snapshot was recorded
	var snapshots []models.ValueSnapshot
	db.Where("user_id = ?", 1).Find(&snapshots)
	if len(snapshots) == 0 {
		t.Fatal("expected a value snapshot to be recorded")
	}
	if snapshots[0].CoinCount != 1 {
		t.Errorf("expected snapshot coin count 1, got %d", snapshots[0].CoinCount)
	}
}

func TestDuplicateCoin_CopiesFieldsAssociationsAndExcludesMediaAndCardState(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestCoinService(db)

	purchasePrice := 250.0
	currentValue := 375.0
	weight := 3.42
	diameter := 19.5
	now := time.Now().UTC().Truncate(time.Second)
	storageLocation := models.StorageLocation{UserID: 1, Name: "Tray A"}
	if err := db.Create(&storageLocation).Error; err != nil {
		t.Fatalf("failed to seed storage location: %v", err)
	}
	source := models.Coin{
		Name:                  "Augustus Denarius",
		Category:              models.CategoryRoman,
		Denomination:          "Denarius",
		Ruler:                 "Augustus",
		Era:                   models.EraAncient,
		Mint:                  "Lugdunum",
		Material:              models.MaterialSilver,
		WeightGrams:           &weight,
		DiameterMm:            &diameter,
		Grade:                 "VF",
		ObverseInscription:    "CAESAR AVGVSTVS",
		ReverseInscription:    "DIVI F",
		ObverseDescription:    "Laureate head right",
		ReverseDescription:    "Bull butting right",
		RarityRating:          "RIC I 167a",
		PurchasePrice:         &purchasePrice,
		CurrentValue:          &currentValue,
		CurrentValueUpdatedAt: &now,
		PurchaseDate:          &now,
		PurchaseLocation:      "Dealer",
		Notes:                 "Important provenance",
		AIAnalysis:            "Combined analysis",
		ObverseAnalysis:       "Obverse analysis",
		ReverseAnalysis:       "Reverse analysis",
		ReferenceURL:          "https://example.test/ref",
		ReferenceText:         "Reference text",
		IsSold:                true,
		SoldPrice:             &currentValue,
		SoldDate:              &now,
		SoldTo:                "Collector",
		ListingStatus:         "available",
		ListingCheckedAt:      &now,
		ListingCheckReason:    "checked",
		StorageLocationID:     &storageLocation.ID,
		IsPrivate:             true,
		UserID:                1,
	}
	if err := db.Create(&source).Error; err != nil {
		t.Fatalf("failed to seed source coin: %v", err)
	}
	if err := db.Create(&models.CoinImage{CoinID: source.ID, FilePath: "coins/source-obverse.jpg", ImageType: models.ImageTypeObverse, IsPrimary: true}).Error; err != nil {
		t.Fatalf("failed to seed image: %v", err)
	}
	if err := db.Create(&models.CoinReference{CoinID: source.ID, Catalog: "RIC", Volume: "I", Number: "167a", InvoiceNumber: "INV-1", URI: "https://example.test/ric"}).Error; err != nil {
		t.Fatalf("failed to seed reference: %v", err)
	}
	tag := models.Tag{UserID: 1, Name: "Silver", Color: "#cccccc"}
	if err := db.Create(&tag).Error; err != nil {
		t.Fatalf("failed to seed tag: %v", err)
	}
	if err := db.Create(&models.CoinTag{CoinID: source.ID, TagID: tag.ID}).Error; err != nil {
		t.Fatalf("failed to seed coin tag: %v", err)
	}
	set := models.CoinSet{UserID: 1, Name: "Twelve Caesars", SetType: models.CoinSetTypeOpen}
	if err := db.Create(&set).Error; err != nil {
		t.Fatalf("failed to seed set: %v", err)
	}
	if err := db.Create(&models.CoinSetMembership{CoinID: source.ID, SetID: set.ID, AddedAt: now, SortOrder: 7, Notes: "set note"}).Error; err != nil {
		t.Fatalf("failed to seed set membership: %v", err)
	}
	showcase := models.Showcase{UserID: 1, Slug: "public-tray", Title: "Public Tray"}
	if err := db.Create(&showcase).Error; err != nil {
		t.Fatalf("failed to seed showcase: %v", err)
	}
	if err := db.Create(&models.ShowcaseCoin{ShowcaseID: showcase.ID, CoinID: source.ID, SortOrder: 3}).Error; err != nil {
		t.Fatalf("failed to seed showcase coin: %v", err)
	}

	duplicate, err := svc.DuplicateCoin(source.ID, 1)
	if err != nil {
		t.Fatalf("DuplicateCoin failed: %v", err)
	}

	if duplicate.ID == source.ID {
		t.Fatal("expected duplicate to receive a new ID")
	}
	if duplicate.Name != "Augustus Denarius (duplicate)" {
		t.Fatalf("unexpected duplicate name: %q", duplicate.Name)
	}
	if duplicate.UserID != source.UserID || duplicate.Category != source.Category || duplicate.Material != source.Material || duplicate.Era != source.Era {
		t.Fatalf("duplicate did not preserve owner/category/material/era: %#v", duplicate)
	}
	if duplicate.AIAnalysis != source.AIAnalysis || duplicate.ObverseAnalysis != source.ObverseAnalysis || duplicate.ReverseAnalysis != source.ReverseAnalysis {
		t.Fatalf("duplicate did not preserve analysis fields")
	}
	if duplicate.StorageLocationID == nil || *duplicate.StorageLocationID != storageLocation.ID {
		t.Fatalf("expected storage location %d, got %v", storageLocation.ID, duplicate.StorageLocationID)
	}
	if duplicate.CurrentValue == nil || *duplicate.CurrentValue != currentValue || duplicate.PurchasePrice == nil || *duplicate.PurchasePrice != purchasePrice {
		t.Fatalf("duplicate did not preserve value fields")
	}
	if len(duplicate.Images) != 0 {
		t.Fatalf("expected images to be excluded, got %#v", duplicate.Images)
	}
	if len(duplicate.References) != 1 || duplicate.References[0].CoinID != duplicate.ID || duplicate.References[0].Catalog != "RIC" {
		t.Fatalf("expected copied reference for duplicate, got %#v", duplicate.References)
	}
	if len(duplicate.Tags) != 1 || duplicate.Tags[0].ID != tag.ID {
		t.Fatalf("expected copied tag association, got %#v", duplicate.Tags)
	}
	if len(duplicate.Sets) != 1 || duplicate.Sets[0].ID != set.ID {
		t.Fatalf("expected copied set association, got %#v", duplicate.Sets)
	}
	var copiedMembership models.CoinSetMembership
	if err := db.Where("coin_id = ? AND set_id = ?", duplicate.ID, set.ID).First(&copiedMembership).Error; err != nil {
		t.Fatalf("expected copied set membership: %v", err)
	}
	if copiedMembership.SortOrder != 7 || copiedMembership.Notes != "set note" || !copiedMembership.AddedAt.Equal(now) {
		t.Fatalf("membership metadata was not copied correctly: %#v", copiedMembership)
	}
	var imageCount, showcaseCount int64
	if err := db.Model(&models.CoinImage{}).Where("coin_id = ?", duplicate.ID).Count(&imageCount).Error; err != nil {
		t.Fatalf("failed to count duplicate images: %v", err)
	}
	if err := db.Model(&models.ShowcaseCoin{}).Where("coin_id = ?", duplicate.ID).Count(&showcaseCount).Error; err != nil {
		t.Fatalf("failed to count duplicate showcase rows: %v", err)
	}
	if imageCount != 0 || showcaseCount != 0 {
		t.Fatalf("expected media/card state to be excluded, got images=%d showcaseRows=%d", imageCount, showcaseCount)
	}
}

func TestDuplicateCoin_RejectsOtherUsersCoin(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestCoinService(db)

	source := models.Coin{Name: "Private Coin", Category: models.CategoryGreek, UserID: 1}
	if err := db.Create(&source).Error; err != nil {
		t.Fatalf("failed to seed source coin: %v", err)
	}

	duplicate, err := svc.DuplicateCoin(source.ID, 2)
	if !repository.IsRecordNotFound(err) {
		t.Fatalf("expected not found for another user's coin, duplicate=%#v err=%v", duplicate, err)
	}
	var count int64
	if err := db.Model(&models.Coin{}).Where("name = ?", "Private Coin (duplicate)").Count(&count).Error; err != nil {
		t.Fatalf("failed to count duplicate coins: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected no duplicate to be created, got %d", count)
	}
}

func TestUpdateCoin_RecordsValueHistory(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestCoinService(db)

	coin := &models.Coin{
		Name:         "Tetradrachm",
		Category:     models.CategoryGreek,
		Material:     models.MaterialSilver,
		CurrentValue: ptrFloat(500.0),
		UserID:       1,
	}
	if err := svc.CreateCoin(coin); err != nil {
		t.Fatalf("setup: CreateCoin failed: %v", err)
	}

	// Update with a new current value (source != "estimate")
	updates := &models.Coin{CurrentValue: ptrFloat(750.0)}
	if err := svc.UpdateCoin(coin, updates, 1, "manual"); err != nil {
		t.Fatalf("UpdateCoin failed: %v", err)
	}

	// Verify value history was recorded
	var history []models.CoinValueHistory
	db.Where("coin_id = ?", coin.ID).Find(&history)
	if len(history) != 1 {
		t.Fatalf("expected 1 value history entry, got %d", len(history))
	}
	if history[0].Value != 750.0 {
		t.Errorf("expected history value 750, got %f", history[0].Value)
	}
	if history[0].Confidence != "manual" {
		t.Errorf("expected confidence 'manual', got %q", history[0].Confidence)
	}

	// Verify journal entry was created
	var journals []models.CoinJournal
	db.Where("coin_id = ?", coin.ID).Find(&journals)
	if len(journals) != 1 {
		t.Fatalf("expected 1 journal entry, got %d", len(journals))
	}

	var snapshots []models.ValueSnapshot
	if err := db.Where("user_id = ?", uint(1)).Find(&snapshots).Error; err != nil {
		t.Fatalf("failed to query value snapshots: %v", err)
	}
	if len(snapshots) != 2 {
		t.Fatalf("expected create and update value snapshots, got %d", len(snapshots))
	}
}

func TestCreateCoin_RejectsNonOwnedStorageLocation(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestCoinServiceWithStorage(db)

	otherLocation := models.StorageLocation{UserID: 2, Name: "Other Cabinet"}
	if err := db.Create(&otherLocation).Error; err != nil {
		t.Fatalf("failed to seed storage location: %v", err)
	}
	coin := &models.Coin{
		Name:              "Stored Coin",
		Category:          models.CategoryRoman,
		Material:          models.MaterialBronze,
		UserID:            1,
		StorageLocationID: &otherLocation.ID,
	}

	if err := svc.CreateCoin(coin); !errors.Is(err, ErrStorageLocationNotFound) {
		t.Fatalf("expected ErrStorageLocationNotFound, got %v", err)
	}

	var count int64
	if err := db.Model(&models.Coin{}).Where("name = ?", "Stored Coin").Count(&count).Error; err != nil {
		t.Fatalf("failed to count coins: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected rejected coin not to be created, got %d rows", count)
	}
}

func TestUpdateCoin_ValidatesAndClearsStorageLocation(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestCoinServiceWithStorage(db)

	ownedLocation := models.StorageLocation{UserID: 1, Name: "Owned Cabinet"}
	otherLocation := models.StorageLocation{UserID: 2, Name: "Other Cabinet"}
	if err := db.Create(&ownedLocation).Error; err != nil {
		t.Fatalf("failed to seed owned storage location: %v", err)
	}
	if err := db.Create(&otherLocation).Error; err != nil {
		t.Fatalf("failed to seed other storage location: %v", err)
	}
	coin := &models.Coin{
		Name:              "Stored Coin",
		Category:          models.CategoryRoman,
		Material:          models.MaterialSilver,
		UserID:            1,
		StorageLocationID: &ownedLocation.ID,
	}
	if err := db.Create(coin).Error; err != nil {
		t.Fatalf("failed to seed coin: %v", err)
	}

	if err := svc.UpdateCoinWithFields(coin, &models.Coin{StorageLocationID: &otherLocation.ID}, nil, 1, "manual", true); !errors.Is(err, ErrStorageLocationNotFound) {
		t.Fatalf("expected ErrStorageLocationNotFound for non-owned location, got %v", err)
	}
	var found models.Coin
	if err := db.First(&found, coin.ID).Error; err != nil {
		t.Fatalf("coin not found: %v", err)
	}
	if found.StorageLocationID == nil || *found.StorageLocationID != ownedLocation.ID {
		t.Fatalf("expected original storage location %d after rejected update, got %v", ownedLocation.ID, found.StorageLocationID)
	}

	if err := svc.UpdateCoinWithFields(coin, &models.Coin{}, nil, 1, "manual", true); err != nil {
		t.Fatalf("expected explicit storage-location clear to succeed: %v", err)
	}
	if err := db.First(&found, coin.ID).Error; err != nil {
		t.Fatalf("coin not found after clear: %v", err)
	}
	if found.StorageLocationID != nil {
		t.Fatalf("expected storage location to be cleared, got %v", found.StorageLocationID)
	}
}

func TestUpdateCoin_NormalizesAndReplacesReferences(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestCoinServiceWithReferences(db)
	if err := db.Create(&models.CatalogRegistry{
		Catalog:        "RIC",
		DisplayName:    "Roman Imperial Coinage",
		Era:            models.EraAncient,
		VolumeRequired: true,
	}).Error; err != nil {
		t.Fatalf("failed to seed catalog registry: %v", err)
	}

	coin := &models.Coin{Name: "Reference Coin", Category: models.CategoryRoman, Material: models.MaterialSilver, UserID: 1}
	if err := svc.CreateCoin(coin); err != nil {
		t.Fatalf("setup: CreateCoin failed: %v", err)
	}
	if err := db.Create(&models.CoinReference{CoinID: coin.ID, Catalog: "RIC", Volume: "I", Number: "1"}).Error; err != nil {
		t.Fatalf("failed to seed existing reference: %v", err)
	}

	updates := &models.Coin{
		References: []models.CoinReference{
			{Catalog: " ric ", Volume: " II ", Number: " 12 "},
		},
	}
	if err := svc.UpdateCoinWithFields(coin, updates, []string{}, 1, "manual", false); err != nil {
		t.Fatalf("UpdateCoinWithFields failed: %v", err)
	}

	var refs []models.CoinReference
	if err := db.Where("coin_id = ?", coin.ID).Find(&refs).Error; err != nil {
		t.Fatalf("failed to query references: %v", err)
	}
	if len(refs) != 1 {
		t.Fatalf("expected one replacement reference, got %d", len(refs))
	}
	if refs[0].Catalog != "RIC" || refs[0].Volume != "II" || refs[0].Number != "12" {
		t.Fatalf("expected normalized RIC II 12, got %#v", refs[0])
	}
}

func TestCreateCoin_DropsReferencesForWishlist(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestCoinServiceWithReferences(db)
	if err := db.Create(&models.CatalogRegistry{
		Catalog:     "RSC",
		DisplayName: "Roman Silver Coins",
		Era:         models.EraAncient,
	}).Error; err != nil {
		t.Fatalf("failed to seed catalog registry: %v", err)
	}

	existing := models.Coin{Name: "Existing", Category: models.CategoryRoman, Material: models.MaterialSilver, UserID: 1}
	if err := svc.CreateCoin(&existing); err != nil {
		t.Fatalf("seed existing coin: %v", err)
	}
	existingRef := models.CoinReference{CoinID: existing.ID, Catalog: "RSC", Number: "234"}
	if err := db.Create(&existingRef).Error; err != nil {
		t.Fatalf("seed existing reference: %v", err)
	}

	wishlist := &models.Coin{
		Name:       "Gordian III Denarius",
		Category:   models.CategoryRoman,
		Material:   models.MaterialSilver,
		UserID:     1,
		IsWishlist: true,
		References: []models.CoinReference{
			{ID: existingRef.ID, Catalog: "RSC", Number: "234"},
		},
	}
	if err := svc.CreateCoin(wishlist); err != nil {
		t.Fatalf("CreateCoin wishlist with references failed: %v", err)
	}

	if len(wishlist.References) != 0 {
		t.Fatalf("expected wishlist response to omit references, got %#v", wishlist.References)
	}
	var refCount int64
	if err := db.Model(&models.CoinReference{}).Where("coin_id = ?", wishlist.ID).Count(&refCount).Error; err != nil {
		t.Fatalf("failed to count wishlist references: %v", err)
	}
	if refCount != 0 {
		t.Fatalf("expected wishlist coin to persist zero references, got %d", refCount)
	}
}

// TestCreateCoin_WishlistInvariant_AgentStylePayload proves the wishlist
// invariant on the realistic agent path: when CreateCoin receives a wishlist
// coin payload whose References slice carries non-zero IDs (as the agent sends
// from source/database data), the resulting DB row must have zero references.
// This test bypasses ConvertCandidate and calls CreateCoin directly to isolate
// the service-layer guard in createPreparedCoinInTx.
func TestCreateCoin_WishlistInvariant_AgentStylePayload(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestCoinServiceWithReferences(db)
	if err := db.Create(&models.CatalogRegistry{
		Catalog:     "RIC",
		DisplayName: "Roman Imperial Coinage",
		Era:         models.EraAncient,
	}).Error; err != nil {
		t.Fatalf("seed catalog: %v", err)
	}

	// Seed a real reference row — its ID represents an occupied primary key
	// that the agent might have looked up from an existing collection coin.
	sourceCoin := &models.Coin{Name: "Source", Category: models.CategoryRoman, Material: models.MaterialSilver, UserID: 1}
	if err := svc.CreateCoin(sourceCoin); err != nil {
		t.Fatalf("seed source coin: %v", err)
	}
	sourceRef := models.CoinReference{CoinID: sourceCoin.ID, Catalog: "RIC", Number: "1"}
	if err := db.Create(&sourceRef).Error; err != nil {
		t.Fatalf("seed source ref: %v", err)
	}

	// Agent-style wishlist payload: IsWishlist=true, References carries a
	// non-zero ID copied from the source coin's reference.
	wishlist := &models.Coin{
		Name:       "Domitian RIC 1",
		Category:   models.CategoryRoman,
		Era:        models.EraAncient,
		Material:   models.MaterialSilver,
		UserID:     1,
		IsWishlist: true,
		References: []models.CoinReference{
			{ID: sourceRef.ID, Catalog: "RIC", Number: "1"},
		},
	}
	if err := svc.CreateCoin(wishlist); err != nil {
		t.Fatalf("CreateCoin with agent-style wishlist payload: %v", err)
	}
	if !wishlist.IsWishlist {
		t.Error("result coin must be a wishlist coin")
	}

	// The invariant: no references may be persisted for a wishlist coin.
	var refCount int64
	if err := db.Model(&models.CoinReference{}).Where("coin_id = ?", wishlist.ID).Count(&refCount).Error; err != nil {
		t.Fatalf("count wishlist refs: %v", err)
	}
	if refCount != 0 {
		t.Errorf("wishlist invariant violated: expected 0 persisted references, got %d", refCount)
	}

	// The source reference must remain untouched.
	var unchanged models.CoinReference
	if err := db.First(&unchanged, sourceRef.ID).Error; err != nil {
		t.Fatalf("source reference missing after wishlist create: %v", err)
	}
	if unchanged.CoinID != sourceCoin.ID {
		t.Errorf("source reference CoinID mutated to %d, want %d", unchanged.CoinID, sourceCoin.ID)
	}
}

func TestUpdateCoin_EstimateSkipsHistory(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestCoinService(db)

	coin := &models.Coin{
		Name:         "Follis",
		Category:     models.CategoryByzantine,
		Material:     models.MaterialBronze,
		CurrentValue: ptrFloat(100.0),
		UserID:       1,
	}
	if err := svc.CreateCoin(coin); err != nil {
		t.Fatalf("setup: CreateCoin failed: %v", err)
	}

	updates := &models.Coin{CurrentValue: ptrFloat(120.0)}
	if err := svc.UpdateCoin(coin, updates, 1, "estimate"); err != nil {
		t.Fatalf("UpdateCoin failed: %v", err)
	}

	// source="estimate" should NOT record value history
	var count int64
	db.Model(&models.CoinValueHistory{}).Where("coin_id = ?", coin.ID).Count(&count)
	if count != 0 {
		t.Errorf("expected 0 value history entries for estimate source, got %d", count)
	}
}

func TestDeleteCoin_RemovesCoinAndImages(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestCoinService(db)

	coin := &models.Coin{
		Name:     "Sestertius",
		Category: models.CategoryRoman,
		Material: models.MaterialBronze,
		UserID:   1,
	}
	if err := svc.CreateCoin(coin); err != nil {
		t.Fatalf("setup: CreateCoin failed: %v", err)
	}

	// Add an image for this coin
	img := models.CoinImage{CoinID: coin.ID, FilePath: "test.jpg", ImageType: models.ImageTypeObverse}
	db.Create(&img)

	rows, err := svc.DeleteCoin(coin.ID, 1)
	if err != nil {
		t.Fatalf("DeleteCoin failed: %v", err)
	}
	if rows != 1 {
		t.Errorf("expected 1 row affected, got %d", rows)
	}

	// Coin should be gone
	var count int64
	db.Model(&models.Coin{}).Where("id = ?", coin.ID).Count(&count)
	if count != 0 {
		t.Error("expected coin to be deleted")
	}

	// Image should be gone
	db.Model(&models.CoinImage{}).Where("coin_id = ?", coin.ID).Count(&count)
	if count != 0 {
		t.Error("expected coin image to be deleted")
	}
}

func TestDeleteCoin_WrongUser_NoEffect(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestCoinService(db)

	coin := &models.Coin{
		Name:     "Stater",
		Category: models.CategoryGreek,
		Material: models.MaterialGold,
		UserID:   1,
	}
	if err := svc.CreateCoin(coin); err != nil {
		t.Fatalf("setup: CreateCoin failed: %v", err)
	}

	// Try deleting with wrong user ID
	rows, err := svc.DeleteCoin(coin.ID, 999)
	if err != nil {
		t.Fatalf("DeleteCoin failed: %v", err)
	}
	if rows != 0 {
		t.Errorf("expected 0 rows affected for wrong user, got %d", rows)
	}

	// Coin should still exist
	var count int64
	db.Model(&models.Coin{}).Where("id = ?", coin.ID).Count(&count)
	if count != 1 {
		t.Error("expected coin to still exist")
	}
}

func TestPurchaseCoin_UpdatesFields(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestCoinService(db)

	now := time.Now()
	coin := &models.Coin{
		Name:       "Aureus Wishlist",
		Category:   models.CategoryRoman,
		Material:   models.MaterialGold,
		IsWishlist: true,
		UserID:     1,
	}
	if err := svc.CreateCoin(coin); err != nil {
		t.Fatalf("setup: CreateCoin failed: %v", err)
	}

	coin.PurchasePrice = ptrFloat(5000.0)
	coin.PurchaseDate = &now
	coin.PurchaseLocation = "Auction House"

	if err := svc.PurchaseCoin(coin, 1); err != nil {
		t.Fatalf("PurchaseCoin failed: %v", err)
	}

	// Reload and verify
	var updated models.Coin
	db.First(&updated, coin.ID)
	if updated.IsWishlist {
		t.Error("expected IsWishlist to be false after purchase")
	}
	if updated.PurchasePrice == nil || *updated.PurchasePrice != 5000.0 {
		t.Error("expected PurchasePrice to be 5000")
	}
	if updated.PurchaseLocation != "Auction House" {
		t.Errorf("expected PurchaseLocation 'Auction House', got %q", updated.PurchaseLocation)
	}
}

func TestSellCoin_UpdatesFields(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestCoinService(db)

	coin := &models.Coin{
		Name:          "Solidus",
		Category:      models.CategoryByzantine,
		Material:      models.MaterialGold,
		PurchasePrice: ptrFloat(2000.0),
		UserID:        1,
	}

	if err := svc.CreateCoin(coin); err != nil {
		t.Fatalf("setup: CreateCoin failed: %v", err)
	}

	now := time.Now()
	updates := map[string]interface{}{
		"is_sold":    true,
		"sold_price": 3000.0,
		"sold_date":  now,
		"sold_to":    "Private Collector",
	}

	if err := svc.SellCoin(coin, updates, 1); err != nil {
		t.Fatalf("SellCoin failed: %v", err)
	}

	var sold models.Coin
	db.First(&sold, coin.ID)
	if !sold.IsSold {
		t.Error("expected IsSold to be true")
	}
	if sold.SoldPrice == nil || *sold.SoldPrice != 3000.0 {
		t.Error("expected SoldPrice to be 3000")
	}
	if sold.SoldTo != "Private Collector" {
		t.Errorf("expected SoldTo 'Private Collector', got %q", sold.SoldTo)
	}
}

func TestUpdateCoin_AcceptsCustomEraFromCatalogRegistry(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestCoinServiceWithCatalogRegistry(db)
	if err := db.Create(&models.CatalogRegistry{
		Catalog:     "PROV",
		DisplayName: "Provincial References",
		Era:         models.Era("provincial"),
	}).Error; err != nil {
		t.Fatalf("failed to seed catalog registry: %v", err)
	}

	coin := &models.Coin{Name: "Provincial Bronze", UserID: 1, Era: models.EraAncient}
	if err := svc.CreateCoin(coin); err != nil {
		t.Fatalf("setup: CreateCoin failed: %v", err)
	}

	updates := &models.Coin{Era: models.Era("provincial")}
	if err := svc.UpdateCoin(coin, updates, 1, "manual"); err != nil {
		t.Fatalf("UpdateCoin should accept registry-defined era: %v", err)
	}

	var found models.Coin
	if err := db.First(&found, coin.ID).Error; err != nil {
		t.Fatalf("coin not found: %v", err)
	}
	if found.Era != models.Era("provincial") {
		t.Fatalf("expected era provincial, got %q", found.Era)
	}
}

func TestUpdateCoin_AcceptsAdminConfiguredEra(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestCoinServiceWithSettings(db)
	settingsRepo := repository.NewSettingsRepository(db)
	if err := settingsRepo.Upsert(SettingCoinEras, "Republican Rome\nRoman Empire\n500-480 BC"); err != nil {
		t.Fatalf("failed to seed coin eras setting: %v", err)
	}

	coin := &models.Coin{Name: "Imperial Bronze", UserID: 1, Era: models.EraAncient}
	if err := svc.CreateCoin(coin); err != nil {
		t.Fatalf("setup: CreateCoin failed: %v", err)
	}

	updates := &models.Coin{Era: models.Era("Roman Empire")}
	if err := svc.UpdateCoin(coin, updates, 1, "manual"); err != nil {
		t.Fatalf("UpdateCoin should accept admin-configured era: %v", err)
	}

	var found models.Coin
	if err := db.First(&found, coin.ID).Error; err != nil {
		t.Fatalf("coin not found: %v", err)
	}
	if found.Era != models.Era("Roman Empire") {
		t.Fatalf("expected era Roman Empire, got %q", found.Era)
	}
}

func TestUpdateCoin_RejectsUnsupportedEraWhenCatalogRegistryEnabled(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestCoinServiceWithCatalogRegistry(db)

	coin := &models.Coin{Name: "Test Coin", UserID: 1, Era: models.EraAncient}
	if err := svc.CreateCoin(coin); err != nil {
		t.Fatalf("setup: CreateCoin failed: %v", err)
	}

	updates := &models.Coin{Era: models.Era("unsupported-era")}
	if err := svc.UpdateCoin(coin, updates, 1, "manual"); !errors.Is(err, ErrCoinInvalidEra) {
		t.Fatalf("expected ErrCoinInvalidEra, got %v", err)
	}
}

func TestUpdateCoin_PreservesUnchangedLegacyEra(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestCoinServiceWithCatalogRegistry(db)

	coin := &models.Coin{Name: "Legacy Era Coin", UserID: 1, Era: models.Era("Imperial")}
	if err := db.Create(coin).Error; err != nil {
		t.Fatalf("setup: create legacy era coin failed: %v", err)
	}

	updates := &models.Coin{Name: "Updated Legacy Era Coin", Era: models.Era("Imperial")}
	if err := svc.UpdateCoin(coin, updates, 1, "manual"); err != nil {
		t.Fatalf("UpdateCoin should allow unchanged legacy era: %v", err)
	}

	var found models.Coin
	if err := db.First(&found, coin.ID).Error; err != nil {
		t.Fatalf("coin not found: %v", err)
	}
	if found.Name != "Updated Legacy Era Coin" {
		t.Fatalf("expected updated name, got %q", found.Name)
	}
	if found.Era != models.Era("Imperial") {
		t.Fatalf("expected legacy era to be preserved, got %q", found.Era)
	}
}
