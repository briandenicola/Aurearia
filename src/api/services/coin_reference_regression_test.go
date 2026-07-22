package services

// Regression tests for: UNIQUE constraint failed coin_references.id
// Root cause: NormalizeAndValidate formerly passed through non-zero IDs from
// agent/source data, causing GORM to INSERT with an existing primary key.
//
// Fix layers (all three must hold):
//  1. NormalizeAndValidate zeros ref.ID and ref.CoinID before returning.
//  2. createPreparedCoinInTx detaches coin.References before Create so GORM
//     never auto-cascades associations.
//  3. prepareCoinForCreate drops References entirely for wishlist coins.

import (
	"testing"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
)

// TestNormalizeAndValidate_ZeroesIncomingReferenceIDs is the pure unit test for
// layer 1: NormalizeAndValidate must strip any incoming ID before returning so
// that downstream CreateBatch always generates a fresh primary key.
func TestNormalizeAndValidate_ZeroesIncomingReferenceIDs(t *testing.T) {
	db := setupTestDB(t)
	if err := db.Create(&models.CatalogRegistry{
		Catalog:     "RSC",
		DisplayName: "Roman Silver Coins",
		Era:         models.EraAncient,
	}).Error; err != nil {
		t.Fatalf("seed catalog: %v", err)
	}

	catalogRepo := repository.NewCatalogRegistryRepository(db)
	refRepo := repository.NewCoinReferenceRepository(db)
	svc := NewCoinReferenceService(refRepo, catalogRepo)

	// Seed a real row so existingID is a genuine occupied primary key.
	existingRef := models.CoinReference{CoinID: 99, Catalog: "RSC", Number: "100"}
	if err := db.Create(&existingRef).Error; err != nil {
		t.Fatalf("seed existing ref: %v", err)
	}
	if existingRef.ID == 0 {
		t.Fatal("seed produced ID=0; test pre-condition broken")
	}

	// Pass a reference that carries the existing row''s ID.
	input := []models.CoinReference{
		{ID: existingRef.ID, Catalog: "RSC", Number: "200"},
	}
	normalized, err := svc.NormalizeAndValidate(input)
	if err != nil {
		t.Fatalf("NormalizeAndValidate failed: %v", err)
	}
	if len(normalized) != 1 {
		t.Fatalf("expected 1 normalized ref, got %d", len(normalized))
	}
	if normalized[0].ID != 0 {
		t.Errorf("expected ID=0 after normalization, got %d", normalized[0].ID)
	}
	if normalized[0].CoinID != 0 {
		t.Errorf("expected CoinID=0 after normalization, got %d", normalized[0].CoinID)
	}
}

// TestUpdateCoin_CrossCoinReferenceIDNotReusedOnUpdate tests layer 1 + update path
// together: updating a coin with references that carry non-zero IDs from a
// different coin must not violate the UNIQUE constraint.
func TestUpdateCoin_CrossCoinReferenceIDNotReusedOnUpdate(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestCoinServiceWithReferences(db)
	if err := db.Create(&models.CatalogRegistry{
		Catalog:     "RSC",
		DisplayName: "Roman Silver Coins",
		Era:         models.EraAncient,
	}).Error; err != nil {
		t.Fatalf("seed catalog: %v", err)
	}

	// Coin A owns a reference row.
	coinA := &models.Coin{Name: "Coin A", Category: models.CategoryRoman, Material: models.MaterialSilver, UserID: 1}
	if err := svc.CreateCoin(coinA); err != nil {
		t.Fatalf("create coin A: %v", err)
	}
	refA := models.CoinReference{CoinID: coinA.ID, Catalog: "RSC", Number: "300"}
	if err := db.Create(&refA).Error; err != nil {
		t.Fatalf("seed refA: %v", err)
	}

	// Coin B is the update target.
	coinB := &models.Coin{Name: "Coin B", Category: models.CategoryRoman, Material: models.MaterialSilver, UserID: 1}
	if err := svc.CreateCoin(coinB); err != nil {
		t.Fatalf("create coin B: %v", err)
	}

	// Update coinB with a reference carrying refA''s ID — simulating the agent
	// returning stale IDs from source/database data.
	updates := &models.Coin{
		References: []models.CoinReference{
			{ID: refA.ID, Catalog: "RSC", Number: "400"},
		},
	}
	if err := svc.UpdateCoinWithFields(coinB, updates, []string{}, 1, "manual", false); err != nil {
		t.Fatalf("UpdateCoinWithFields with stale reference ID failed: %v", err)
	}

	var refs []models.CoinReference
	if err := db.Where("coin_id = ?", coinB.ID).Find(&refs).Error; err != nil {
		t.Fatalf("query coinB refs: %v", err)
	}
	if len(refs) != 1 {
		t.Fatalf("expected 1 reference on coinB, got %d", len(refs))
	}
	if refs[0].ID == refA.ID {
		t.Errorf("reference on coinB reused refA.ID=%d; expected a fresh primary key", refA.ID)
	}
	if refs[0].CoinID != coinB.ID {
		t.Errorf("reference CoinID=%d, want %d", refs[0].CoinID, coinB.ID)
	}
	if refs[0].Number != "400" {
		t.Errorf("expected number=400, got %q", refs[0].Number)
	}
	// refA must still exist and belong to coinA.
	var originalRef models.CoinReference
	if err := db.First(&originalRef, refA.ID).Error; err != nil {
		t.Fatalf("refA missing after update: %v", err)
	}
	if originalRef.CoinID != coinA.ID {
		t.Errorf("refA.CoinID mutated to %d, want %d", originalRef.CoinID, coinA.ID)
	}
}

// TestCreateCoin_CollectionCoinSecondReferenceWithSameIDDoesNotConflict ensures
// that creating two collection coins each receiving a reference whose incoming ID
// matches the same existing row both succeed and each gets a distinct new ID.
func TestCreateCoin_CollectionCoinSecondReferenceWithSameIDDoesNotConflict(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestCoinServiceWithReferences(db)
	if err := db.Create(&models.CatalogRegistry{
		Catalog:     "RSC",
		DisplayName: "Roman Silver Coins",
		Era:         models.EraAncient,
	}).Error; err != nil {
		t.Fatalf("seed catalog: %v", err)
	}

	// Seed a reference row with a real primary key.
	sourceRef := models.CoinReference{CoinID: 99, Catalog: "RSC", Number: "500"}
	if err := db.Create(&sourceRef).Error; err != nil {
		t.Fatalf("seed sourceRef: %v", err)
	}

	make := func(name, num string) *models.Coin {
		return &models.Coin{
			Name:     name,
			Category: models.CategoryRoman,
			Material: models.MaterialSilver,
			UserID:   1,
			References: []models.CoinReference{
				{ID: sourceRef.ID, Catalog: "RSC", Number: num},
			},
		}
	}

	coin1 := make("Coin One", "501")
	if err := svc.CreateCoin(coin1); err != nil {
		t.Fatalf("first coin create failed: %v", err)
	}
	coin2 := make("Coin Two", "502")
	if err := svc.CreateCoin(coin2); err != nil {
		t.Fatalf("second coin create with same incoming ID failed: %v", err)
	}

	var refs1, refs2 []models.CoinReference
	if err := db.Where("coin_id = ?", coin1.ID).Find(&refs1).Error; err != nil {
		t.Fatalf("query coin1 refs: %v", err)
	}
	if err := db.Where("coin_id = ?", coin2.ID).Find(&refs2).Error; err != nil {
		t.Fatalf("query coin2 refs: %v", err)
	}
	if len(refs1) != 1 || len(refs2) != 1 {
		t.Fatalf("expected 1 ref per coin, got coin1=%d coin2=%d", len(refs1), len(refs2))
	}
	if refs1[0].ID == sourceRef.ID || refs2[0].ID == sourceRef.ID {
		t.Error("a coin reference reused the seeded ID; expected fresh primary keys")
	}
	if refs1[0].ID == refs2[0].ID {
		t.Errorf("two coins received identical reference IDs (%d)", refs1[0].ID)
	}
}