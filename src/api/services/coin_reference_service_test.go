package services

// Feature 352 Phase 2 — CoinReferenceService.AppendForCoin.
//
// Contract under test (FR-013/FR-014, spec.md section 4.3):
//  1. loads the coin's existing references,
//  2. drops any proposed element whose (Catalog, Volume, Number) triple —
//     compared case-insensitively via dedupeKey — already exists on the coin
//     (or earlier in the same input batch),
//  3. validates each surviving element through NormalizeAndValidateOne,
//  4. inserts only the survivors,
//  5. deletes nothing.
//
// Applying the same catalogReferences twice MUST be idempotent by
// construction (FR-014); the service must not rely on catching a unique
// constraint violation to achieve that.

import (
	"errors"
	"testing"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"gorm.io/gorm"
)

// newTestCoinReferenceService wires a CoinReferenceService against the given
// test DB, mirroring the construction used by newTestCoinServiceWithReferences
// in coin_service_test.go.
func newTestCoinReferenceService(db *gorm.DB) *CoinReferenceService {
	refRepo := repository.NewCoinReferenceRepository(db)
	catalogRepo := repository.NewCatalogRegistryRepository(db)
	return NewCoinReferenceService(refRepo, catalogRepo)
}

// seedCatalog inserts a CatalogRegistry row for use by AppendForCoin tests.
func seedCatalog(t *testing.T, db *gorm.DB, catalog string, volumeRequired bool) {
	t.Helper()
	if err := db.Create(&models.CatalogRegistry{
		Catalog:        catalog,
		DisplayName:    catalog,
		Era:            models.EraAncient,
		VolumeRequired: volumeRequired,
	}).Error; err != nil {
		t.Fatalf("seed catalog %s: %v", catalog, err)
	}
}

// seedCoin creates a bare coin owned by userID and returns its ID.
func seedCoin(t *testing.T, db *gorm.DB, userID uint) uint {
	t.Helper()
	coin := &models.Coin{
		Name:     "Test Coin",
		Category: models.CategoryRoman,
		Material: models.MaterialSilver,
		UserID:   userID,
	}
	if err := db.Create(coin).Error; err != nil {
		t.Fatalf("seed coin: %v", err)
	}
	return coin.ID
}

func TestAppendForCoin_PreservesExistingAndReturnsOnlyInserted(t *testing.T) {
	db := setupTestDB(t)
	seedCatalog(t, db, "RSC", false)
	coinID := seedCoin(t, db, 1)

	refRepo := repository.NewCoinReferenceRepository(db)
	svc := newTestCoinReferenceService(db)

	// Seed one pre-existing reference directly, bypassing the service, so we
	// know exactly what "existing" means for this test.
	existing := models.CoinReference{CoinID: coinID, Catalog: "RSC", Number: "100"}
	if err := db.Create(&existing).Error; err != nil {
		t.Fatalf("seed existing ref: %v", err)
	}

	inserted, err := svc.AppendForCoin(coinID, 1, []models.CoinReference{
		{Catalog: "RSC", Number: "200"},
		{Catalog: "RSC", Number: "300"},
	})
	if err != nil {
		t.Fatalf("AppendForCoin failed: %v", err)
	}
	if len(inserted) != 2 {
		t.Fatalf("expected 2 inserted refs returned, got %d", len(inserted))
	}
	for _, r := range inserted {
		if r.ID == 0 {
			t.Errorf("inserted ref missing generated ID: %+v", r)
		}
		if r.Number == "100" {
			t.Errorf("AppendForCoin returned the pre-existing ref, want only newly inserted rows")
		}
	}

	all, err := refRepo.ListByCoin(coinID, 1)
	if err != nil {
		t.Fatalf("ListByCoin failed: %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("expected 3 total references on coin (1 existing + 2 inserted), got %d", len(all))
	}
	var sawExisting bool
	for _, r := range all {
		if r.ID == existing.ID && r.Number == "100" {
			sawExisting = true
		}
	}
	if !sawExisting {
		t.Error("pre-existing reference was not preserved")
	}
}

func TestAppendForCoin_IdempotentAcrossRepeatedCalls(t *testing.T) {
	db := setupTestDB(t)
	seedCatalog(t, db, "RSC", false)
	coinID := seedCoin(t, db, 1)
	refRepo := repository.NewCoinReferenceRepository(db)
	svc := newTestCoinReferenceService(db)

	input := []models.CoinReference{
		{Catalog: "RSC", Number: "100"},
	}

	first, err := svc.AppendForCoin(coinID, 1, input)
	if err != nil {
		t.Fatalf("first AppendForCoin failed: %v", err)
	}
	if len(first) != 1 {
		t.Fatalf("expected 1 inserted ref on first call, got %d", len(first))
	}

	// Repeat the identical call: FR-014 requires this to be a no-op, not an
	// error, and it must not rely on catching a unique constraint violation.
	second, err := svc.AppendForCoin(coinID, 1, input)
	if err != nil {
		t.Fatalf("repeated AppendForCoin with identical input returned an error, want idempotent no-op: %v", err)
	}
	if len(second) != 0 {
		t.Fatalf("expected 0 inserted refs on repeated call, got %d: %+v", len(second), second)
	}

	all, err := refRepo.ListByCoin(coinID, 1)
	if err != nil {
		t.Fatalf("ListByCoin failed: %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("expected exactly 1 reference after repeated append, got %d", len(all))
	}
}

func TestAppendForCoin_CaseInsensitiveDuplicateAgainstExistingIsSkipped(t *testing.T) {
	db := setupTestDB(t)
	seedCatalog(t, db, "RSC", false)
	coinID := seedCoin(t, db, 1)
	refRepo := repository.NewCoinReferenceRepository(db)
	svc := newTestCoinReferenceService(db)

	existing := models.CoinReference{CoinID: coinID, Catalog: "RSC", Volume: "ii", Number: "abc"}
	if err := db.Create(&existing).Error; err != nil {
		t.Fatalf("seed existing ref: %v", err)
	}

	// Same triple, different case on every field.
	inserted, err := svc.AppendForCoin(coinID, 1, []models.CoinReference{
		{Catalog: "rsc", Volume: "II", Number: "ABC"},
	})
	if err != nil {
		t.Fatalf("AppendForCoin failed: %v", err)
	}
	if len(inserted) != 0 {
		t.Fatalf("expected case-insensitive duplicate to be skipped, got %d inserted: %+v", len(inserted), inserted)
	}

	all, err := refRepo.ListByCoin(coinID, 1)
	if err != nil {
		t.Fatalf("ListByCoin failed: %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("expected reference count to remain 1, got %d", len(all))
	}
}

func TestAppendForCoin_CaseInsensitiveDuplicateWithinSameBatchIsSkipped(t *testing.T) {
	db := setupTestDB(t)
	seedCatalog(t, db, "RSC", false)
	coinID := seedCoin(t, db, 1)
	refRepo := repository.NewCoinReferenceRepository(db)
	svc := newTestCoinReferenceService(db)

	// No pre-existing references at all; the duplicate pair is entirely
	// within this one input batch, with the second occurrence differing only
	// by case. Per FR-013 step 2, the earlier occurrence in the batch wins
	// and the later duplicate is dropped, not the DB unique index catching it.
	inserted, err := svc.AppendForCoin(coinID, 1, []models.CoinReference{
		{Catalog: "RSC", Volume: "I", Number: "1"},
		{Catalog: "rsc", Volume: "i", Number: "1"},
		{Catalog: "RSC", Volume: "I", Number: "2"},
	})
	if err != nil {
		t.Fatalf("AppendForCoin failed: %v", err)
	}
	if len(inserted) != 2 {
		t.Fatalf("expected 2 survivors (intra-batch duplicate dropped), got %d: %+v", len(inserted), inserted)
	}

	all, err := refRepo.ListByCoin(coinID, 1)
	if err != nil {
		t.Fatalf("ListByCoin failed: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("expected 2 references persisted, got %d", len(all))
	}
}

func TestAppendForCoin_UnknownCatalogRejectedWithNoPartialWrite(t *testing.T) {
	db := setupTestDB(t)
	seedCatalog(t, db, "RSC", false)
	coinID := seedCoin(t, db, 1)
	refRepo := repository.NewCoinReferenceRepository(db)
	svc := newTestCoinReferenceService(db)

	_, err := svc.AppendForCoin(coinID, 1, []models.CoinReference{
		{Catalog: "RSC", Number: "1"},   // valid, would insert if allowed to proceed
		{Catalog: "BOGUS", Number: "1"}, // unknown catalog, must fail validation
	})
	if err == nil {
		t.Fatal("expected an error for an unknown catalog, got nil")
	}
	if !errorsIsUnknownCatalog(err) {
		t.Errorf("expected ErrReferenceUnknownCatalog, got: %v", err)
	}

	all, err := refRepo.ListByCoin(coinID, 1)
	if err != nil {
		t.Fatalf("ListByCoin failed: %v", err)
	}
	if len(all) != 0 {
		t.Fatalf("expected no partial writes after a rejected batch, got %d references: %+v", len(all), all)
	}
}

func TestAppendForCoin_VolumeRequiredEmptyRejectedWithNoPartialWrite(t *testing.T) {
	db := setupTestDB(t)
	seedCatalog(t, db, "RSC", false)
	seedCatalog(t, db, "RPC", true) // volume required
	coinID := seedCoin(t, db, 1)
	refRepo := repository.NewCoinReferenceRepository(db)
	svc := newTestCoinReferenceService(db)

	_, err := svc.AppendForCoin(coinID, 1, []models.CoinReference{
		{Catalog: "RSC", Number: "1"}, // valid, would insert if allowed to proceed
		{Catalog: "RPC", Number: "1"}, // missing required volume, must fail validation
	})
	if err == nil {
		t.Fatal("expected an error for a missing required volume, got nil")
	}
	if !errorsIsVolumeRequired(err) {
		t.Errorf("expected ErrReferenceVolumeRequired, got: %v", err)
	}

	all, err := refRepo.ListByCoin(coinID, 1)
	if err != nil {
		t.Fatalf("ListByCoin failed: %v", err)
	}
	if len(all) != 0 {
		t.Fatalf("expected no partial writes after a rejected batch, got %d references: %+v", len(all), all)
	}
}

func TestAppendForCoin_InsertedRefsAreScopedToRequestedCoin(t *testing.T) {
	db := setupTestDB(t)
	seedCatalog(t, db, "RSC", false)
	coinID := seedCoin(t, db, 1)
	otherCoinID := seedCoin(t, db, 1)
	refRepo := repository.NewCoinReferenceRepository(db)
	svc := newTestCoinReferenceService(db)

	// Input carries a bogus/stale CoinID; AppendForCoin must ignore it and
	// scope every inserted row to the coinID parameter, never the payload.
	inserted, err := svc.AppendForCoin(coinID, 1, []models.CoinReference{
		{CoinID: otherCoinID, Catalog: "RSC", Number: "1"},
	})
	if err != nil {
		t.Fatalf("AppendForCoin failed: %v", err)
	}
	if len(inserted) != 1 {
		t.Fatalf("expected 1 inserted ref, got %d", len(inserted))
	}
	if inserted[0].CoinID != coinID {
		t.Errorf("inserted ref CoinID=%d, want %d (requested coin, not payload CoinID=%d)", inserted[0].CoinID, coinID, otherCoinID)
	}

	otherRefs, err := refRepo.ListByCoin(otherCoinID, 1)
	if err != nil {
		t.Fatalf("ListByCoin(otherCoinID) failed: %v", err)
	}
	if len(otherRefs) != 0 {
		t.Fatalf("expected the other coin to receive no references, got %d: %+v", len(otherRefs), otherRefs)
	}
}

func TestAppendForCoin_ExistingReferenceOnAnotherUsersCoinDoesNotBlockDedupe(t *testing.T) {
	db := setupTestDB(t)
	seedCatalog(t, db, "RSC", false)
	// coinA belongs to user 1; coinB (different owner) happens to carry the
	// identical (Catalog, Volume, Number) triple. Ownership scoping in
	// ListByCoin (repository/coin_reference_repository.go) means user 2's
	// dedupe set must only ever be built from user 2's own coin.
	coinA := seedCoin(t, db, 1)
	coinB := seedCoin(t, db, 2)
	refRepo := repository.NewCoinReferenceRepository(db)
	svc := newTestCoinReferenceService(db)

	existingOnA := models.CoinReference{CoinID: coinA, Catalog: "RSC", Number: "42"}
	if err := db.Create(&existingOnA).Error; err != nil {
		t.Fatalf("seed existing ref on coinA: %v", err)
	}

	inserted, err := svc.AppendForCoin(coinB, 2, []models.CoinReference{
		{Catalog: "RSC", Number: "42"},
	})
	if err != nil {
		t.Fatalf("AppendForCoin for user 2 / coinB failed: %v", err)
	}
	if len(inserted) != 1 {
		t.Fatalf("expected the identical triple on a different user's coin to be inserted (not treated as a dup), got %d", len(inserted))
	}

	refsB, err := refRepo.ListByCoin(coinB, 2)
	if err != nil {
		t.Fatalf("ListByCoin(coinB) failed: %v", err)
	}
	if len(refsB) != 1 {
		t.Fatalf("expected coinB to have 1 reference, got %d", len(refsB))
	}

	refsA, err := refRepo.ListByCoin(coinA, 1)
	if err != nil {
		t.Fatalf("ListByCoin(coinA) failed: %v", err)
	}
	if len(refsA) != 1 {
		t.Fatalf("expected coinA's original reference to be untouched, got %d", len(refsA))
	}
}

func TestAppendForCoin_EmptyInputIsSuccessfulNoOp(t *testing.T) {
	db := setupTestDB(t)
	seedCatalog(t, db, "RSC", false)
	coinID := seedCoin(t, db, 1)
	svc := newTestCoinReferenceService(db)

	inserted, err := svc.AppendForCoin(coinID, 1, []models.CoinReference{})
	if err != nil {
		t.Fatalf("AppendForCoin with empty input returned an error: %v", err)
	}
	if inserted == nil {
		t.Error("expected an empty, non-nil slice for empty input")
	}
	if len(inserted) != 0 {
		t.Fatalf("expected 0 inserted refs for empty input, got %d", len(inserted))
	}
}

func errorsIsUnknownCatalog(err error) bool {
	return errors.Is(err, ErrReferenceUnknownCatalog)
}

func errorsIsVolumeRequired(err error) bool {
	return errors.Is(err, ErrReferenceVolumeRequired)
}
