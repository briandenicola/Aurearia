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
	if err := db.AutoMigrate(&models.User{}, &models.Coin{}, &models.CoinImage{}, &models.ValueSnapshot{}, &models.QuickCaptureDraft{}, &models.QuickCaptureDraftImage{}, &models.DraftLifecycleEvent{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	return db
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
	); !errors.Is(err, ErrDraftNotEditable) {
		t.Fatalf("expected inactive update to fail, got %v", err)
	}
}
