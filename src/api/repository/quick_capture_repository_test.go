package repository

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func newQuickCaptureRepositoryTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:quick_capture_repository_%d?mode=memory&cache=shared", time.Now().UnixNano())), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.Coin{}, &models.CoinImage{}, &models.CoinReference{}, &models.ValueSnapshot{}, &models.QuickCaptureDraft{}, &models.QuickCaptureDraftImage{}, &models.QuickCaptureDraftReference{}, &models.DraftLifecycleEvent{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	return db
}

func selectedDraftReference(userID uint, number string) *models.QuickCaptureDraftReference {
	return &models.QuickCaptureDraftReference{
		UserID: userID, Catalog: "Numista", Number: number,
		URI: fmt.Sprintf("https://en.numista.com/catalogue/pieces%s.html", number),
	}
}

func TestQuickCaptureRepositorySelectedReferenceCreatePreserveReplaceRemoveAndOwnerIsolation(t *testing.T) {
	db := newQuickCaptureRepositoryTestDB(t)
	repo := NewQuickCaptureRepository(db)
	userID := uint(7)
	draft := &models.QuickCaptureDraft{UserID: userID, WorkingTitle: "Selected", Status: models.QuickCaptureDraftStatusActive}
	event := &models.DraftLifecycleEvent{UserID: userID, EventType: models.DraftLifecycleEventCreated, CreatedAt: time.Now().UTC()}
	if err := repo.CreateDraftWithImages(draft, event, selectedDraftReference(userID, "123"), func(uint) ([]models.QuickCaptureDraftImage, error) {
		return nil, nil
	}); err != nil {
		t.Fatalf("create selected draft: %v", err)
	}
	found, err := repo.GetDraftForOwner(draft.ID, userID)
	if err != nil || found.SelectedNumistaReference == nil || found.SelectedNumistaReference.Number != "123" {
		t.Fatalf("selected relation not preloaded: draft=%#v err=%v", found, err)
	}
	if _, err := repo.GetDraftForOwner(draft.ID, userID+1); err == nil {
		t.Fatal("other owner read selected draft")
	}

	updateEvent := func() *models.DraftLifecycleEvent {
		return &models.DraftLifecycleEvent{UserID: userID, EventType: models.DraftLifecycleEventUpdated, CreatedAt: time.Now().UTC()}
	}
	preserved, _, err := repo.UpdateDraftTransaction(
		draft.ID, userID, map[string]interface{}{"working_title": "Preserved"},
		nil, nil, nil, updateEvent(), DraftReferenceMutation{},
	)
	if err != nil || preserved.SelectedNumistaReference == nil || preserved.SelectedNumistaReference.Number != "123" {
		t.Fatalf("omitted update did not preserve selection: %#v err=%v", preserved, err)
	}

	replaced, _, err := repo.UpdateDraftTransaction(
		draft.ID, userID, nil, nil, nil, nil, updateEvent(),
		DraftReferenceMutation{Replace: selectedDraftReference(userID, "456")},
	)
	if err != nil || replaced.SelectedNumistaReference == nil || replaced.SelectedNumistaReference.Number != "456" {
		t.Fatalf("replace failed: %#v err=%v", replaced, err)
	}
	var count int64
	if err := db.Model(&models.QuickCaptureDraftReference{}).Where("draft_id = ?", draft.ID).Count(&count).Error; err != nil || count != 1 {
		t.Fatalf("one-to-one guard failed: count=%d err=%v", count, err)
	}

	cleared, _, err := repo.UpdateDraftTransaction(
		draft.ID, userID, nil, nil, nil, nil, updateEvent(),
		DraftReferenceMutation{Clear: true},
	)
	if err != nil || cleared.SelectedNumistaReference != nil {
		t.Fatalf("clear failed: %#v err=%v", cleared, err)
	}
}

func TestQuickCaptureRepositorySelectedReferenceSurvivesDiscard(t *testing.T) {
	db := newQuickCaptureRepositoryTestDB(t)
	repo := NewQuickCaptureRepository(db)
	draft := models.QuickCaptureDraft{UserID: 7, WorkingTitle: "History", Status: models.QuickCaptureDraftStatusActive}
	if err := db.Create(&draft).Error; err != nil {
		t.Fatal(err)
	}
	ref := selectedDraftReference(7, "987")
	ref.DraftID = draft.ID
	if err := db.Create(ref).Error; err != nil {
		t.Fatal(err)
	}
	discarded, err := repo.DiscardDraft(draft.ID, 7)
	if err != nil {
		t.Fatal(err)
	}
	if discarded.SelectedNumistaReference == nil || discarded.SelectedNumistaReference.Number != "987" {
		t.Fatalf("discard should retain history relation: %#v", discarded)
	}
}

func TestQuickCaptureRepositoryPromotionCopiesSelectedReferenceForCollectionAndWishlistExactlyOnce(t *testing.T) {
	for _, wishlist := range []bool{false, true} {
		t.Run(fmt.Sprintf("wishlist_%v", wishlist), func(t *testing.T) {
			db := newQuickCaptureRepositoryTestDB(t)
			repo := NewQuickCaptureRepository(db)
			draft := models.QuickCaptureDraft{UserID: 7, WorkingTitle: "Promote", Status: models.QuickCaptureDraftStatusActive}
			if err := db.Create(&draft).Error; err != nil {
				t.Fatal(err)
			}
			ref := selectedDraftReference(7, "12345")
			ref.DraftID = draft.ID
			if err := db.Create(ref).Error; err != nil {
				t.Fatal(err)
			}
			coin := &models.Coin{
				UserID: 7, Name: "Promoted", Category: models.CategoryRoman,
				Material: models.MaterialSilver, Era: models.EraAncient, IsWishlist: wishlist,
			}
			_, created, err := repo.PromoteDraftTransaction(draft.ID, 7, coin)
			if err != nil {
				t.Fatal(err)
			}
			var refs []models.CoinReference
			if err := db.Where("coin_id = ?", created.ID).Find(&refs).Error; err != nil {
				t.Fatal(err)
			}
			if len(refs) != 1 || refs[0].Catalog != "Numista" || refs[0].Number != "12345" {
				t.Fatalf("selected reference copy mismatch: %#v", refs)
			}
			if _, _, err := repo.PromoteDraftTransaction(draft.ID, 7, &models.Coin{UserID: 7, Name: "Duplicate"}); !errors.Is(err, ErrDraftNotClaimable) {
				t.Fatalf("repeated repository promotion should not copy again: %v", err)
			}
			var count int64
			if err := db.Model(&models.CoinReference{}).Where("coin_id = ?", created.ID).Count(&count).Error; err != nil || count != 1 {
				t.Fatalf("reference duplicated: count=%d err=%v", count, err)
			}
		})
	}
}

func TestQuickCaptureRepositoryPromotionReferenceFailureRollsBackEverything(t *testing.T) {
	db := newQuickCaptureRepositoryTestDB(t)
	repo := NewQuickCaptureRepository(db)
	draft := models.QuickCaptureDraft{UserID: 7, WorkingTitle: "Rollback", Status: models.QuickCaptureDraftStatusActive}
	if err := db.Create(&draft).Error; err != nil {
		t.Fatal(err)
	}
	ref := selectedDraftReference(7, "55")
	ref.DraftID = draft.ID
	if err := db.Create(ref).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Callback().Create().Before("gorm:create").Register("fail_coin_reference_copy", func(tx *gorm.DB) {
		if _, ok := tx.Statement.Dest.(*models.CoinReference); ok {
			tx.AddError(errors.New("forced reference copy failure"))
		}
	}); err != nil {
		t.Fatal(err)
	}
	_, _, err := repo.PromoteDraftTransaction(draft.ID, 7, &models.Coin{
		UserID: 7, Name: "Rollback", Category: models.CategoryRoman, Material: models.MaterialSilver, Era: models.EraAncient,
	})
	if err == nil {
		t.Fatal("expected promotion failure")
	}
	var refreshed models.QuickCaptureDraft
	if err := db.First(&refreshed, draft.ID).Error; err != nil {
		t.Fatal(err)
	}
	if refreshed.Status != models.QuickCaptureDraftStatusActive || refreshed.PromotedCoinID != nil {
		t.Fatalf("draft claim was not rolled back: %#v", refreshed)
	}
	var coinCount int64
	if err := db.Model(&models.Coin{}).Count(&coinCount).Error; err != nil || coinCount != 0 {
		t.Fatalf("coin write was not rolled back: count=%d err=%v", coinCount, err)
	}
}

func TestQuickCaptureRepositoryConcurrentPromotionCreatesAtMostOneCoinAndReference(t *testing.T) {
	db := newQuickCaptureRepositoryTestDB(t)
	repo := NewQuickCaptureRepository(db)
	draft := models.QuickCaptureDraft{UserID: 7, WorkingTitle: "Concurrent", Status: models.QuickCaptureDraftStatusActive}
	if err := db.Create(&draft).Error; err != nil {
		t.Fatal(err)
	}
	ref := selectedDraftReference(7, "77")
	ref.DraftID = draft.ID
	if err := db.Create(ref).Error; err != nil {
		t.Fatal(err)
	}

	start := make(chan struct{})
	results := make(chan error, 2)
	for i := 0; i < 2; i++ {
		go func() {
			<-start
			_, _, err := repo.PromoteDraftTransaction(draft.ID, 7, &models.Coin{
				UserID: 7, Name: "Concurrent", Category: models.CategoryRoman,
				Material: models.MaterialSilver, Era: models.EraAncient,
			})
			results <- err
		}()
	}
	close(start)
	err1, err2 := <-results, <-results
	if err1 != nil && err2 != nil {
		t.Fatalf("expected one concurrent promotion to succeed: %v / %v", err1, err2)
	}
	var coins, refs int64
	if err := db.Model(&models.Coin{}).Count(&coins).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&models.CoinReference{}).Count(&refs).Error; err != nil {
		t.Fatal(err)
	}
	if coins != 1 || refs != 1 {
		t.Fatalf("concurrent promotion duplicated writes: coins=%d refs=%d errors=%v/%v", coins, refs, err1, err2)
	}
}

func TestQuickCaptureRepositoryCreateIsOwnerScopedAndPreloadsImages(t *testing.T) {
	db := newQuickCaptureRepositoryTestDB(t)
	repo := NewQuickCaptureRepository(db)
	owner := models.User{Username: "owner", Email: "owner@example.com", PasswordHash: "x"}
	other := models.User{Username: "other", Email: "other@example.com", PasswordHash: "x"}
	if err := db.Create(&owner).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&other).Error; err != nil {
		t.Fatal(err)
	}

	draft := &models.QuickCaptureDraft{UserID: owner.ID, WorkingTitle: "Unattributed", Status: models.QuickCaptureDraftStatusActive}
	if err := repo.CreateDraft(draft); err != nil {
		t.Fatalf("create draft: %v", err)
	}
	if err := repo.AddDraftImage(&models.QuickCaptureDraftImage{DraftID: draft.ID, UserID: owner.ID, FilePath: "quick-capture-draft-1/a.png", ImageType: models.ImageTypeObverse, IsPrimary: true}); err != nil {
		t.Fatalf("add image: %v", err)
	}

	found, err := repo.GetDraftForOwner(draft.ID, owner.ID)
	if err != nil {
		t.Fatalf("owner should read draft: %v", err)
	}
	if len(found.Images) != 1 {
		t.Fatalf("expected preloaded image, got %d", len(found.Images))
	}
	if _, err := repo.GetDraftForOwner(draft.ID, other.ID); err == nil {
		t.Fatal("non-owner should not read draft")
	}
}

func TestQuickCaptureRepositoryListsActiveDraftsByOwnerAndUpdatedOrder(t *testing.T) {
	db := newQuickCaptureRepositoryTestDB(t)
	repo := NewQuickCaptureRepository(db)
	owner := uint(10)
	other := uint(20)
	now := time.Now().UTC()

	drafts := []models.QuickCaptureDraft{
		{UserID: owner, WorkingTitle: "Older active", Status: models.QuickCaptureDraftStatusActive, UpdatedAt: now.Add(-2 * time.Hour)},
		{UserID: owner, WorkingTitle: "Discarded", Status: models.QuickCaptureDraftStatusDiscarded, UpdatedAt: now.Add(-1 * time.Hour)},
		{UserID: other, WorkingTitle: "Other owner", Status: models.QuickCaptureDraftStatusActive, UpdatedAt: now},
		{UserID: owner, WorkingTitle: "Newest active", Status: models.QuickCaptureDraftStatusActive, UpdatedAt: now.Add(-30 * time.Minute)},
	}
	for i := range drafts {
		if err := db.Create(&drafts[i]).Error; err != nil {
			t.Fatalf("create draft %d: %v", i, err)
		}
	}
	selected := selectedDraftReference(owner, "12345")
	selected.DraftID = drafts[3].ID
	if err := db.Create(selected).Error; err != nil {
		t.Fatalf("create owner selected reference: %v", err)
	}
	otherSelected := selectedDraftReference(other, "99999")
	otherSelected.DraftID = drafts[2].ID
	if err := db.Create(otherSelected).Error; err != nil {
		t.Fatalf("create other-owner selected reference: %v", err)
	}

	found, total, err := repo.ListDraftsForOwner(owner, models.QuickCaptureDraftStatusActive, 1, 50)
	if err != nil {
		t.Fatalf("list active drafts: %v", err)
	}
	if total != 2 || len(found) != 2 {
		t.Fatalf("expected 2 owner active drafts, got total=%d len=%d", total, len(found))
	}
	if found[0].WorkingTitle != "Newest active" || found[1].WorkingTitle != "Older active" {
		t.Fatalf("expected updated_at desc order, got %q then %q", found[0].WorkingTitle, found[1].WorkingTitle)
	}
	if found[0].SelectedNumistaReference == nil ||
		found[0].SelectedNumistaReference.Number != "12345" ||
		found[0].SelectedNumistaReference.UserID != owner {
		t.Fatalf("owner selected reference was not preloaded: %#v", found[0].SelectedNumistaReference)
	}
	if found[1].SelectedNumistaReference != nil {
		t.Fatalf("unselected owner draft should not acquire a relation: %#v", found[1].SelectedNumistaReference)
	}
}

func TestQuickCaptureRepositoryPromoteDraftTransactionCreatesCoinImagesSnapshotAndClaimsOnce(t *testing.T) {
	db := newQuickCaptureRepositoryTestDB(t)
	repo := NewQuickCaptureRepository(db)
	userID := uint(7)
	price := 42.5
	draft := models.QuickCaptureDraft{UserID: userID, WorkingTitle: "Augustus denarius", Status: models.QuickCaptureDraftStatusActive, PurchasePrice: &price}
	if err := db.Create(&draft).Error; err != nil {
		t.Fatalf("create draft: %v", err)
	}
	if err := db.Create(&models.QuickCaptureDraftImage{DraftID: draft.ID, UserID: userID, FilePath: "quick-capture-draft-1/obverse.png", ImageType: models.ImageTypeObverse, IsPrimary: true}).Error; err != nil {
		t.Fatalf("create draft image: %v", err)
	}
	coin := &models.Coin{UserID: userID, Name: "Augustus denarius", Category: models.CategoryRoman, Material: models.MaterialSilver, Era: models.EraAncient, PurchasePrice: &price, CurrentValue: &price}

	promoted, createdCoin, err := repo.PromoteDraftTransaction(draft.ID, userID, coin)
	if err != nil {
		t.Fatalf("promote draft: %v", err)
	}
	if promoted.Status != models.QuickCaptureDraftStatusPromoted || promoted.PromotedCoinID == nil || *promoted.PromotedCoinID != createdCoin.ID {
		t.Fatalf("draft not linked as promoted: status=%s promotedCoinId=%v coin=%d", promoted.Status, promoted.PromotedCoinID, createdCoin.ID)
	}
	var coinCount, coinImageCount, snapshotCount int64
	if err := db.Model(&models.Coin{}).Where("user_id = ?", userID).Count(&coinCount).Error; err != nil {
		t.Fatalf("count coins: %v", err)
	}
	if err := db.Model(&models.CoinImage{}).Where("coin_id = ?", createdCoin.ID).Count(&coinImageCount).Error; err != nil {
		t.Fatalf("count coin images: %v", err)
	}
	if err := db.Model(&models.ValueSnapshot{}).Where("user_id = ?", userID).Count(&snapshotCount).Error; err != nil {
		t.Fatalf("count snapshots: %v", err)
	}
	if coinCount != 1 || coinImageCount != 1 || snapshotCount != 1 {
		t.Fatalf("expected one coin, image, and snapshot; got coins=%d images=%d snapshots=%d", coinCount, coinImageCount, snapshotCount)
	}

	_, _, err = repo.PromoteDraftTransaction(draft.ID, userID, &models.Coin{UserID: userID, Name: "Duplicate"})
	if !errors.Is(err, ErrDraftNotClaimable) {
		t.Fatalf("expected second promotion claim to fail, got %v", err)
	}
}

func TestQuickCaptureRepositoryUpdateAndDiscardDraft(t *testing.T) {
	db := newQuickCaptureRepositoryTestDB(t)
	repo := NewQuickCaptureRepository(db)
	userID := uint(7)
	draft := models.QuickCaptureDraft{UserID: userID, WorkingTitle: "Original", Status: models.QuickCaptureDraftStatusActive}
	if err := db.Create(&draft).Error; err != nil {
		t.Fatalf("create draft: %v", err)
	}
	image := models.QuickCaptureDraftImage{DraftID: draft.ID, UserID: userID, FilePath: "quick-capture-draft-1/old.png", ImageType: models.ImageTypeObverse, IsPrimary: true}
	if err := db.Create(&image).Error; err != nil {
		t.Fatalf("create image: %v", err)
	}

	updated, removed, err := repo.UpdateDraftTransaction(
		draft.ID,
		userID,
		map[string]interface{}{"working_title": "Updated"},
		[]uint{image.ID},
		nil,
		[]models.QuickCaptureDraftImage{{UserID: userID, FilePath: "quick-capture-draft-1/new.png", ImageType: models.ImageTypeReverse}},
		&models.DraftLifecycleEvent{UserID: userID, EventType: models.DraftLifecycleEventUpdated, Message: "updated", CreatedAt: time.Now().UTC()},
		DraftReferenceMutation{},
	)
	if err != nil {
		t.Fatalf("update draft: %v", err)
	}
	if updated.WorkingTitle != "Updated" || len(updated.Images) != 1 || updated.Images[0].ImageType != models.ImageTypeReverse {
		t.Fatalf("unexpected updated draft: %#v", updated)
	}
	if len(removed) != 1 || removed[0] != image.FilePath {
		t.Fatalf("expected removed image path %q, got %#v", image.FilePath, removed)
	}

	discarded, err := repo.DiscardDraft(draft.ID, userID)
	if err != nil {
		t.Fatalf("discard draft: %v", err)
	}
	if discarded.Status != models.QuickCaptureDraftStatusDiscarded || discarded.DiscardedAt == nil {
		t.Fatalf("expected discarded draft with timestamp, got %#v", discarded)
	}
	if _, _, err := repo.UpdateDraftTransaction(
		draft.ID,
		userID,
		map[string]interface{}{"working_title": "Should not save"},
		nil,
		nil,
		nil,
		&models.DraftLifecycleEvent{UserID: userID, EventType: models.DraftLifecycleEventUpdated, Message: "updated", CreatedAt: time.Now().UTC()},
		DraftReferenceMutation{},
	); !errors.Is(err, ErrDraftNotEditable) {
		t.Fatalf("expected inactive update to fail, got %v", err)
	}
}
