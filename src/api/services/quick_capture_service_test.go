package services

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func newQuickCaptureServiceForTest(t *testing.T) *QuickCaptureService {
	t.Helper()
	svc, _ := newQuickCaptureServiceAndDBForTest(t, t.TempDir())
	return svc
}

func newQuickCaptureServiceAndDBForTest(t *testing.T, uploadDir string) (*QuickCaptureService, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:quick_capture_service_%d?mode=memory&cache=shared", time.Now().UnixNano())), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.Coin{}, &models.CoinImage{}, &models.CoinReference{}, &models.CatalogRegistry{}, &models.ValueSnapshot{}, &models.QuickCaptureDraft{}, &models.QuickCaptureDraftImage{}, &models.QuickCaptureDraftReference{}, &models.DraftLifecycleEvent{}, &models.AppSetting{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := db.Create(&models.CatalogRegistry{
		Catalog: "Numista", DisplayName: "Numista", Era: models.EraModern,
	}).Error; err != nil {
		t.Fatalf("seed Numista registry: %v", err)
	}
	referenceSvc := NewCoinReferenceService(
		repository.NewCoinReferenceRepository(db),
		repository.NewCatalogRegistryRepository(db),
	)
	return NewQuickCaptureService(repository.NewQuickCaptureRepository(db), uploadDir).
		WithReferenceValidation(referenceSvc), db
}

func TestQuickCaptureServiceRequiresMinimumIdentity(t *testing.T) {
	svc := newQuickCaptureServiceForTest(t)
	_, err := svc.CreateDraft(CreateQuickCaptureDraftInput{UserID: 1})
	if !errors.Is(err, ErrQuickCaptureMinimumIdentity) {
		t.Fatalf("expected minimum identity error, got %v", err)
	}
}

func TestQuickCaptureServiceRejectsInvalidPrice(t *testing.T) {
	svc := newQuickCaptureServiceForTest(t)
	price := -1.0
	_, err := svc.CreateDraft(CreateQuickCaptureDraftInput{UserID: 1, WorkingTitle: "Draft", PurchasePrice: &price})
	if !errors.Is(err, ErrQuickCaptureInvalidPrice) {
		t.Fatalf("expected invalid price error, got %v", err)
	}
}

func selectedNumistaServiceRef(t *testing.T, id int) *models.SelectedNumistaReference {
	t.Helper()
	ref, err := models.NewSelectedNumistaReference(id)
	if err != nil {
		t.Fatal(err)
	}
	return &ref
}

func TestQuickCaptureServiceSelectedReferenceCreatePreserveReplaceClearAndValidationRollback(t *testing.T) {
	svc, _ := newQuickCaptureServiceAndDBForTest(t, t.TempDir())
	draft, err := svc.CreateDraft(CreateQuickCaptureDraftInput{
		UserID: 1, WorkingTitle: "Selected",
		SelectedNumistaReference: selectedNumistaServiceRef(t, 123),
	})
	if err != nil {
		t.Fatal(err)
	}

	if draft.SelectedNumistaReference == nil || draft.SelectedNumistaReference.Number != "123" {
		t.Fatalf("create selection missing: %#v", draft)
	}

	preserved, err := svc.UpdateDraft(1, draft.ID, UpdateQuickCaptureDraftInput{
		WorkingTitle: "Preserved",
	})
	if err != nil || preserved.SelectedNumistaReference == nil || preserved.SelectedNumistaReference.Number != "123" {
		t.Fatalf("omitted selection was not preserved: %#v err=%v", preserved, err)
	}

	invalid := &models.SelectedNumistaReference{
		Catalog: "Numista", Number: "456",
		URI: "https://en.numista.com/catalogue/pieces999.html",
	}
	if _, err := svc.UpdateDraft(1, draft.ID, UpdateQuickCaptureDraftInput{
		WorkingTitle: "Invalid must not save", SelectedNumistaProvided: true,
		SelectedNumistaReference: invalid,
	}); !errors.Is(err, ErrQuickCaptureInvalidReference) {
		t.Fatalf("expected invalid reference error, got %v", err)
	}
	afterInvalid, err := svc.GetDraft(1, draft.ID)
	if err != nil || afterInvalid.WorkingTitle != "Preserved" ||
		afterInvalid.SelectedNumistaReference == nil || afterInvalid.SelectedNumistaReference.Number != "123" {
		t.Fatalf("validation failure mutated draft: %#v err=%v", afterInvalid, err)
	}

	replaced, err := svc.UpdateDraft(1, draft.ID, UpdateQuickCaptureDraftInput{
		WorkingTitle: "Replaced", SelectedNumistaProvided: true,
		SelectedNumistaReference: selectedNumistaServiceRef(t, 456),
	})
	if err != nil || replaced.SelectedNumistaReference == nil || replaced.SelectedNumistaReference.Number != "456" {
		t.Fatalf("replace failed: %#v err=%v", replaced, err)
	}
	if _, err := svc.UpdateDraft(1, draft.ID, UpdateQuickCaptureDraftInput{
		WorkingTitle: "Conflict", SelectedNumistaProvided: true,
		SelectedNumistaReference: selectedNumistaServiceRef(t, 789),
		ClearSelectedNumista:     true,
	}); !errors.Is(err, ErrQuickCaptureReferenceConflict) {
		t.Fatalf("expected clear/replace conflict, got %v", err)
	}

	cleared, err := svc.UpdateDraft(1, draft.ID, UpdateQuickCaptureDraftInput{
		WorkingTitle: "Cleared", ClearSelectedNumista: true,
	})
	if err != nil || cleared.SelectedNumistaReference != nil {
		t.Fatalf("clear failed: %#v err=%v", cleared, err)
	}
}

func TestQuickCaptureServiceListDraftsReturnsOnlyOwnerSelections(t *testing.T) {
	svc, _ := newQuickCaptureServiceAndDBForTest(t, t.TempDir())
	selected, err := svc.CreateDraft(CreateQuickCaptureDraftInput{
		UserID: 1, WorkingTitle: "Selected owner draft",
		SelectedNumistaReference: selectedNumistaServiceRef(t, 12345),
	})
	if err != nil {
		t.Fatal(err)
	}
	unselected, err := svc.CreateDraft(CreateQuickCaptureDraftInput{
		UserID: 1, WorkingTitle: "Unselected owner draft",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CreateDraft(CreateQuickCaptureDraftInput{
		UserID: 2, WorkingTitle: "Other owner draft",
		SelectedNumistaReference: selectedNumistaServiceRef(t, 99999),
	}); err != nil {
		t.Fatal(err)
	}

	drafts, total, err := svc.ListDrafts(1, models.QuickCaptureDraftStatusActive, 1, 50)
	if err != nil {
		t.Fatalf("list owner drafts: %v", err)
	}
	if total != 2 || len(drafts) != 2 {
		t.Fatalf("owner-scoped total mismatch: total=%d drafts=%#v", total, drafts)
	}

	byID := make(map[uint]models.QuickCaptureDraft, len(drafts))
	for _, draft := range drafts {
		byID[draft.ID] = draft
	}
	if got := byID[selected.ID].SelectedNumistaReference; got == nil ||
		got.Catalog != "Numista" || got.Number != "12345" ||
		got.URI != "https://en.numista.com/catalogue/pieces12345.html" {
		t.Fatalf("selected list projection mismatch: %#v", got)
	}
	if got := byID[unselected.ID].SelectedNumistaReference; got != nil {
		t.Fatalf("unselected list projection should be nil: %#v", got)
	}
}

func TestQuickCaptureServiceSelectedReferencePromotionCollectionWishlistAndNoSelection(t *testing.T) {
	for _, test := range []struct {
		name      string
		target    QuickCapturePromotionTarget
		selection *models.SelectedNumistaReference
		wantRefs  int64
	}{
		{"collection", QuickCapturePromotionTargetCollection, selectedNumistaServiceRef(t, 101), 1},
		{"wishlist", QuickCapturePromotionTargetWishlist, selectedNumistaServiceRef(t, 202), 1},
		{"none", QuickCapturePromotionTargetCollection, nil, 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			svc, db := newQuickCaptureServiceAndDBForTest(t, t.TempDir())
			draft, err := svc.CreateDraft(CreateQuickCaptureDraftInput{
				UserID: 1, WorkingTitle: "Promote", Era: string(models.EraAncient),
				SelectedNumistaReference: test.selection,
			})
			if err != nil {
				t.Fatal(err)
			}
			first, err := svc.PromoteDraft(1, draft.ID, PromoteDraftInput{Confirm: true, Target: test.target})
			if err != nil {
				t.Fatal(err)
			}
			second, err := svc.PromoteDraft(1, draft.ID, PromoteDraftInput{Confirm: true})
			if err != nil || !second.AlreadyPromoted || second.CoinID != first.CoinID {
				t.Fatalf("repeated promotion not idempotent: %#v err=%v", second, err)
			}
			var refs int64
			if err := db.Model(&models.CoinReference{}).Where("coin_id = ?", first.CoinID).Count(&refs).Error; err != nil {
				t.Fatal(err)
			}
			if refs != test.wantRefs {
				t.Fatalf("reference count=%d want %d", refs, test.wantRefs)
			}
		})
	}
}

func TestQuickCaptureServicePersistsFindCoinMetadata(t *testing.T) {
	svc := newQuickCaptureServiceForTest(t)
	draft, err := svc.CreateDraft(CreateQuickCaptureDraftInput{
		UserID:        1,
		WorkingTitle:  "Augustus Denarius",
		Source:        "find_coin_ai",
		NGCCertNumber: "1234567-001",
		NGCLookupURL:  "https://www.ngccoin.com/certlookup/1234567001/NGCAncients/",
		NGCGrade:      "Ch VF",
		LabelText:     "NGC Ancients Augustus Denarius",
		AIConfidence:  "high",
		Images: []QuickCaptureImageUpload{{
			Filename:  "obverse.png",
			Data:      validQuickCapturePNG(),
			ImageType: string(models.ImageTypeObverse),
			IsPrimary: true,
		}},
	})
	if err != nil {
		t.Fatalf("create draft: %v", err)
	}
	if draft.Source != "find_coin_ai" || draft.NGCCertNumber != "1234567-001" || draft.NGCGrade != "Ch VF" {
		t.Fatalf("expected find coin metadata to persist, got %#v", draft)
	}
	if draft.LabelText == "" || draft.AIConfidence != "high" {
		t.Fatalf("expected label text and confidence to persist, got label=%q confidence=%q", draft.LabelText, draft.AIConfidence)
	}
}

func TestQuickCaptureServiceRollsBackDraftWhenImageSaveFails(t *testing.T) {
	uploadRoot := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(uploadRoot, []byte("blocks directory creation"), 0644); err != nil {
		t.Fatalf("create blocker file: %v", err)
	}
	svc, db := newQuickCaptureServiceAndDBForTest(t, uploadRoot)

	_, err := svc.CreateDraft(CreateQuickCaptureDraftInput{
		UserID:       1,
		WorkingTitle: "Draft with image",
		Images: []QuickCaptureImageUpload{{
			Filename:  "obverse.png",
			Data:      validQuickCapturePNG(),
			ImageType: string(models.ImageTypeObverse),
			IsPrimary: true,
		}},
	})
	if err == nil {
		t.Fatal("expected image save failure")
	}
	assertNoQuickCaptureRows(t, db)
}

func TestQuickCaptureServiceRollsBackDraftAndRemovesFilesWhenImageInsertFails(t *testing.T) {
	uploadRoot := t.TempDir()
	svc, db := newQuickCaptureServiceAndDBForTest(t, uploadRoot)
	if err := db.Callback().Create().Before("gorm:create").Register("quick_capture_image_insert_failure", func(tx *gorm.DB) {
		if _, ok := tx.Statement.Dest.(*models.QuickCaptureDraftImage); ok {
			tx.AddError(errors.New("forced image insert failure"))
		}
	}); err != nil {
		t.Fatalf("register callback: %v", err)
	}

	_, err := svc.CreateDraft(CreateQuickCaptureDraftInput{
		UserID:       1,
		WorkingTitle: "Draft with image",
		Images: []QuickCaptureImageUpload{{
			Filename:  "obverse.png",
			Data:      validQuickCapturePNG(),
			ImageType: string(models.ImageTypeObverse),
			IsPrimary: true,
		}},
	})
	if err == nil {
		t.Fatal("expected image insert failure")
	}
	assertNoQuickCaptureRows(t, db)
	entries, readErr := os.ReadDir(uploadRoot)
	if readErr != nil {
		t.Fatalf("read upload root: %v", readErr)
	}
	if len(entries) != 0 {
		t.Fatalf("expected written image cleanup, found %d entries", len(entries))
	}
}

func TestQuickCaptureServiceUpdatePreservesObverseWhenOnlyDetailIsAdded(t *testing.T) {
	svc, _ := newQuickCaptureServiceAndDBForTest(t, t.TempDir())
	draft, err := svc.CreateDraft(CreateQuickCaptureDraftInput{
		UserID:       1,
		WorkingTitle: "Draft",
		Images: []QuickCaptureImageUpload{{
			Filename:  "obverse.png",
			Data:      validQuickCapturePNG(),
			ImageType: string(models.ImageTypeObverse),
			IsPrimary: true,
		}},
	})
	if err != nil {
		t.Fatalf("create draft: %v", err)
	}

	updated, err := svc.UpdateDraft(1, draft.ID, UpdateQuickCaptureDraftInput{
		UserID:         1,
		WorkingTitle:   "Draft",
		ReplaceObverse: true,
		NewImages: []QuickCaptureImageUpload{{
			Filename:  "detail.png",
			Data:      validQuickCapturePNG(),
			ImageType: string(models.ImageTypeDetail),
		}},
	})
	if err != nil {
		t.Fatalf("update draft: %v", err)
	}
	counts := map[models.ImageType]int{}
	for _, img := range updated.Images {
		counts[img.ImageType]++
	}
	if counts[models.ImageTypeObverse] != 1 || counts[models.ImageTypeDetail] != 1 {
		t.Fatalf("expected existing obverse plus new detail, got counts %#v", counts)
	}
}

func TestQuickCaptureServiceDiscardIsIdempotentAndPromotedDraftConflicts(t *testing.T) {
	svc, _ := newQuickCaptureServiceAndDBForTest(t, t.TempDir())
	draft, err := svc.CreateDraft(CreateQuickCaptureDraftInput{UserID: 1, WorkingTitle: "Discard me"})
	if err != nil {
		t.Fatalf("create draft: %v", err)
	}
	first, err := svc.DiscardDraft(1, draft.ID)
	if err != nil {
		t.Fatalf("discard draft: %v", err)
	}
	second, err := svc.DiscardDraft(1, draft.ID)
	if err != nil {
		t.Fatalf("discard draft again: %v", err)
	}
	if first.Status != models.QuickCaptureDraftStatusDiscarded || second.Status != models.QuickCaptureDraftStatusDiscarded {
		t.Fatalf("expected discarded status, got %s then %s", first.Status, second.Status)
	}

	promotedDraft, err := svc.CreateDraft(CreateQuickCaptureDraftInput{UserID: 1, WorkingTitle: "Promote me"})
	if err != nil {
		t.Fatalf("create promoted draft: %v", err)
	}
	if _, err := svc.PromoteDraft(1, promotedDraft.ID, PromoteDraftInput{Confirm: true}); err != nil {
		t.Fatalf("promote draft: %v", err)
	}
	if _, err := svc.DiscardDraft(1, promotedDraft.ID); !errors.Is(err, ErrQuickCaptureDraftAlreadyPromoted) {
		t.Fatalf("expected promoted draft discard conflict, got %v", err)
	}
}

func TestQuickCaptureServicePromoteDraftValidatesAndIsIdempotent(t *testing.T) {
	price := 42.5
	svc, db := newQuickCaptureServiceAndDBForTest(t, t.TempDir())
	missingName, err := svc.CreateDraft(CreateQuickCaptureDraftInput{UserID: 1, Notes: "Needs a name before promotion"})
	if err != nil {
		t.Fatalf("create missing-name draft: %v", err)
	}
	_, err = svc.PromoteDraft(1, missingName.ID, PromoteDraftInput{Confirm: true})
	var validationErr *QuickCapturePromotionValidationError
	if !errors.As(err, &validationErr) || validationErr.Fields["name"] == "" {
		t.Fatalf("expected field-level name validation, got %v", err)
	}

	draft, err := svc.CreateDraft(CreateQuickCaptureDraftInput{
		UserID:        1,
		WorkingTitle:  "Augustus denarius",
		Era:           string(models.EraAncient),
		PurchasePrice: &price,
		Images: []QuickCaptureImageUpload{{
			Filename:  "obverse.png",
			Data:      validQuickCapturePNG(),
			ImageType: string(models.ImageTypeObverse),
			IsPrimary: true,
		}},
	})
	if err != nil {
		t.Fatalf("create promotable draft: %v", err)
	}
	first, err := svc.PromoteDraft(1, draft.ID, PromoteDraftInput{Confirm: true})
	if err != nil {
		t.Fatalf("first promote: %v", err)
	}
	second, err := svc.PromoteDraft(1, draft.ID, PromoteDraftInput{Confirm: true})
	if err != nil {
		t.Fatalf("second promote: %v", err)
	}
	if second.CoinID != first.CoinID || !second.AlreadyPromoted {
		t.Fatalf("expected idempotent existing coin response, first=%#v second=%#v", first, second)
	}
	var coinCount, coinImageCount, snapshotCount int64
	if err := db.Model(&models.Coin{}).Where("user_id = ?", uint(1)).Count(&coinCount).Error; err != nil {
		t.Fatalf("count coins: %v", err)
	}
	if err := db.Model(&models.CoinImage{}).Where("coin_id = ?", first.CoinID).Count(&coinImageCount).Error; err != nil {
		t.Fatalf("count images: %v", err)
	}
	if err := db.Model(&models.ValueSnapshot{}).Where("user_id = ?", uint(1)).Count(&snapshotCount).Error; err != nil {
		t.Fatalf("count snapshots: %v", err)
	}
	if coinCount != 1 || coinImageCount != 1 || snapshotCount != 1 {
		t.Fatalf("expected one promoted coin/image/snapshot, got coins=%d images=%d snapshots=%d", coinCount, coinImageCount, snapshotCount)
	}
	active, total, err := svc.ListDrafts(1, models.QuickCaptureDraftStatusActive, 1, 50)
	if err != nil {
		t.Fatalf("list active drafts: %v", err)
	}
	if total != 1 || len(active) != 1 || active[0].ID != missingName.ID {
		t.Fatalf("promoted draft should be hidden from active list, total=%d active=%#v", total, active)
	}
}

func TestQuickCaptureServicePromoteDraftDistinguishesExplicitEmptyOverridesFromOmitted(t *testing.T) {
	stringPtr := func(value string) *string { return &value }
	pricePtr := func(value float64) *float64 { return &value }
	svc, db := newQuickCaptureServiceAndDBForTest(t, t.TempDir())

	fallbackDraft, err := svc.CreateDraft(CreateQuickCaptureDraftInput{
		UserID:            1,
		WorkingTitle:      "Saved title",
		Era:               string(models.EraAncient),
		AcquisitionSource: "Saved source",
		Notes:             "Saved notes",
	})
	if err != nil {
		t.Fatalf("create fallback draft: %v", err)
	}
	fallbackResult, err := svc.PromoteDraft(1, fallbackDraft.ID, PromoteDraftInput{Confirm: true})
	if err != nil {
		t.Fatalf("promote fallback draft: %v", err)
	}
	var fallbackCoin models.Coin
	if err := db.First(&fallbackCoin, fallbackResult.CoinID).Error; err != nil {
		t.Fatalf("load fallback coin: %v", err)
	}
	if fallbackCoin.Name != "Saved title" || fallbackCoin.PurchaseLocation != "Saved source" || fallbackCoin.Notes != "Saved notes" {
		t.Fatalf("omitted overrides should fall back to saved draft fields, got name=%q source=%q notes=%q", fallbackCoin.Name, fallbackCoin.PurchaseLocation, fallbackCoin.Notes)
	}

	priceDraft, err := svc.CreateDraft(CreateQuickCaptureDraftInput{
		UserID:            1,
		WorkingTitle:      "Saved priced title",
		Era:               string(models.EraAncient),
		AcquisitionSource: "Saved source",
		PurchasePrice:     pricePtr(100),
	})
	if err != nil {
		t.Fatalf("create priced draft: %v", err)
	}
	priceFallbackResult, err := svc.PromoteDraft(1, priceDraft.ID, PromoteDraftInput{Confirm: true})
	if err != nil {
		t.Fatalf("promote priced fallback draft: %v", err)
	}
	var priceFallbackCoin models.Coin
	if err := db.First(&priceFallbackCoin, priceFallbackResult.CoinID).Error; err != nil {
		t.Fatalf("load price fallback coin: %v", err)
	}
	if priceFallbackCoin.PurchasePrice == nil || *priceFallbackCoin.PurchasePrice != 100 {
		t.Fatalf("omitted price override should use saved draft price, got %v", priceFallbackCoin.PurchasePrice)
	}

	clearedPriceDraft, err := svc.CreateDraft(CreateQuickCaptureDraftInput{
		UserID:            1,
		WorkingTitle:      "Saved priced title",
		Era:               string(models.EraAncient),
		AcquisitionSource: "Saved source",
		PurchasePrice:     pricePtr(100),
	})
	if err != nil {
		t.Fatalf("create cleared price draft: %v", err)
	}
	clearedPriceResult, err := svc.PromoteDraft(1, clearedPriceDraft.ID, PromoteDraftInput{
		Confirm: true,
		Overrides: PromoteOverrides{
			Name:             stringPtr("Current priced title"),
			PurchasePriceSet: true,
		},
	})
	if err != nil {
		t.Fatalf("promote cleared price draft: %v", err)
	}
	var clearedPriceCoin models.Coin
	if err := db.First(&clearedPriceCoin, clearedPriceResult.CoinID).Error; err != nil {
		t.Fatalf("load cleared price coin: %v", err)
	}
	if clearedPriceCoin.PurchasePrice != nil || clearedPriceCoin.CurrentValue != nil {
		t.Fatalf("explicit nil price override should clear saved draft price/current value, got price=%v value=%v", clearedPriceCoin.PurchasePrice, clearedPriceCoin.CurrentValue)
	}

	clearedDraft, err := svc.CreateDraft(CreateQuickCaptureDraftInput{
		UserID:            1,
		WorkingTitle:      "Saved title",
		Era:               string(models.EraAncient),
		AcquisitionSource: "Saved source",
		Notes:             "Saved notes",
	})
	if err != nil {
		t.Fatalf("create cleared draft: %v", err)
	}
	clearedResult, err := svc.PromoteDraft(1, clearedDraft.ID, PromoteDraftInput{
		Confirm: true,
		Overrides: PromoteOverrides{
			Name:             stringPtr("Current title"),
			PurchaseLocation: stringPtr(""),
			Notes:            stringPtr(""),
		},
	})
	if err != nil {
		t.Fatalf("promote cleared draft: %v", err)
	}
	var clearedCoin models.Coin
	if err := db.First(&clearedCoin, clearedResult.CoinID).Error; err != nil {
		t.Fatalf("load cleared coin: %v", err)
	}
	if clearedCoin.Name != "Current title" || clearedCoin.PurchaseLocation != "" || clearedCoin.Notes != "" {
		t.Fatalf("explicit empty overrides should not use saved draft fields, got name=%q source=%q notes=%q", clearedCoin.Name, clearedCoin.PurchaseLocation, clearedCoin.Notes)
	}

	missingTitleDraft, err := svc.CreateDraft(CreateQuickCaptureDraftInput{
		UserID:       1,
		WorkingTitle: "Saved title",
		Era:          string(models.EraAncient),
	})
	if err != nil {
		t.Fatalf("create missing-title draft: %v", err)
	}
	_, err = svc.PromoteDraft(1, missingTitleDraft.ID, PromoteDraftInput{
		Confirm: true,
		Overrides: PromoteOverrides{
			Name: stringPtr(""),
		},
	})
	var validationErr *QuickCapturePromotionValidationError
	if !errors.As(err, &validationErr) || validationErr.Fields["name"] == "" {
		t.Fatalf("expected explicit empty name override to fail validation, got %v", err)
	}
}

func TestQuickCaptureServicePromoteDraftCanTargetWishlist(t *testing.T) {
	price := 42.5
	svc, db := newQuickCaptureServiceAndDBForTest(t, t.TempDir())
	draft, err := svc.CreateDraft(CreateQuickCaptureDraftInput{
		UserID:        1,
		WorkingTitle:  "Wishlist denarius",
		Era:           string(models.EraAncient),
		PurchasePrice: &price,
	})
	if err != nil {
		t.Fatalf("create promotable draft: %v", err)
	}

	result, err := svc.PromoteDraft(1, draft.ID, PromoteDraftInput{
		Confirm: true,
		Target:  QuickCapturePromotionTargetWishlist,
	})
	if err != nil {
		t.Fatalf("promote to wishlist: %v", err)
	}
	if result.Target != QuickCapturePromotionTargetWishlist {
		t.Fatalf("expected wishlist target in result, got %q", result.Target)
	}

	var coin models.Coin
	if err := db.First(&coin, result.CoinID).Error; err != nil {
		t.Fatalf("load promoted coin: %v", err)
	}
	if !coin.IsWishlist || coin.IsSold {
		t.Fatalf("expected promoted coin to be wishlist and unsold, wishlist=%v sold=%v", coin.IsWishlist, coin.IsSold)
	}
	if coin.UserID != 1 {
		t.Fatalf("expected promoted coin to preserve owner 1, got %d", coin.UserID)
	}
}

func TestQuickCaptureServicePromoteDraftRejectsInvalidTarget(t *testing.T) {
	svc, _ := newQuickCaptureServiceAndDBForTest(t, t.TempDir())
	draft, err := svc.CreateDraft(CreateQuickCaptureDraftInput{UserID: 1, WorkingTitle: "Target validation"})
	if err != nil {
		t.Fatalf("create draft: %v", err)
	}

	_, err = svc.PromoteDraft(1, draft.ID, PromoteDraftInput{Confirm: true, Target: "archive"})
	var validationErr *QuickCapturePromotionValidationError
	if !errors.As(err, &validationErr) || validationErr.Fields["target"] == "" {
		t.Fatalf("expected target validation error, got %v", err)
	}
}

func validQuickCapturePNG() []byte {
	return []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A,
		0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53,
		0xDE, 0x00, 0x00, 0x00, 0x0C, 0x49, 0x44, 0x41,
		0x54, 0x08, 0xD7, 0x63, 0xF8, 0xCF, 0xC0, 0x00,
		0x00, 0x03, 0x01, 0x01, 0x00, 0x18, 0xDD, 0x8D,
		0xB0, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4E,
		0x44, 0xAE, 0x42, 0x60, 0x82,
	}
}

func TestQuickCaptureServicePromoteDraft_AcceptsAdminConfiguredEraAndCategoryWhenCoinValidationWired(t *testing.T) {
	svc, db := newQuickCaptureServiceAndDBForTest(t, t.TempDir())

	settingsRepo := repository.NewSettingsRepository(db)
	if err := settingsRepo.Upsert(SettingCoinEras, "Roman Provincial Year 12"); err != nil {
		t.Fatalf("seed coin eras setting: %v", err)
	}
	if err := settingsRepo.Upsert(SettingCoinCategories, "Roman\nGreek\nByzantine\nModern\nOther\nCeltic"); err != nil {
		t.Fatalf("seed coin categories setting: %v", err)
	}
	coinSvc := NewCoinService(repository.NewCoinRepository(db), nil).WithSettingsSupport(NewSettingsService(settingsRepo))
	svc = svc.WithCoinValidation(coinSvc)

	stringPtr := func(value string) *string { return &value }
	draft, err := svc.CreateDraft(CreateQuickCaptureDraftInput{
		UserID:       1,
		WorkingTitle: "Provincial bronze",
		Era:          "Roman Provincial Year 12",
	})
	if err != nil {
		t.Fatalf("create draft: %v", err)
	}

	result, err := svc.PromoteDraft(1, draft.ID, PromoteDraftInput{
		Confirm:   true,
		Overrides: PromoteOverrides{Category: stringPtr("Celtic")},
	})
	if err != nil {
		t.Fatalf("expected promotion to accept admin-configured era/category, got %v", err)
	}

	var coin models.Coin
	if err := db.First(&coin, result.CoinID).Error; err != nil {
		t.Fatalf("load promoted coin: %v", err)
	}
	if coin.Era != models.Era("Roman Provincial Year 12") {
		t.Fatalf("expected custom era to persist, got %q", coin.Era)
	}
	if coin.Category != models.Category("Celtic") {
		t.Fatalf("expected custom category to persist, got %q", coin.Category)
	}
}

func TestQuickCaptureServicePromoteDraft_RejectsCustomEraWithoutCoinValidationWired(t *testing.T) {
	// Baseline: when no CoinService is wired (WithCoinValidation not called),
	// promotion falls back to built-in defaults only - never laxer than
	// today's behavior, even though it can't consult admin settings.
	svc, _ := newQuickCaptureServiceAndDBForTest(t, t.TempDir())

	draft, err := svc.CreateDraft(CreateQuickCaptureDraftInput{
		UserID:       1,
		WorkingTitle: "Provincial bronze",
		Era:          "Roman Provincial Year 12",
	})
	if err != nil {
		t.Fatalf("create draft: %v", err)
	}

	_, err = svc.PromoteDraft(1, draft.ID, PromoteDraftInput{Confirm: true})
	var validationErr *QuickCapturePromotionValidationError
	if !errors.As(err, &validationErr) || validationErr.Fields["era"] == "" {
		t.Fatalf("expected field-level era validation without coin validation wired, got %v", err)
	}
}

func assertNoQuickCaptureRows(t *testing.T, db *gorm.DB) {
	t.Helper()
	for name, model := range map[string]interface{}{
		"drafts":           &models.QuickCaptureDraft{},
		"draft images":     &models.QuickCaptureDraftImage{},
		"lifecycle events": &models.DraftLifecycleEvent{},
	} {
		var count int64
		if err := db.Model(model).Count(&count).Error; err != nil {
			t.Fatalf("count %s: %v", name, err)
		}
		if count != 0 {
			t.Fatalf("expected no %s after rollback, got %d", name, count)
		}
	}
}
