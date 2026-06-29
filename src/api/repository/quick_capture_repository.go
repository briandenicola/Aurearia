package repository

import (
	"errors"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"gorm.io/gorm"
)

// ErrDraftNotEditable is returned when an update is attempted on a draft that is not active.
var ErrDraftNotEditable = errors.New("quick capture draft is not active")

// ErrDraftNotClaimable is returned when the CAS claim in PromoteDraftTransaction finds no active row.
var ErrDraftNotClaimable = errors.New("quick capture draft cannot be claimed for promotion: not active or concurrent modification")

type QuickCaptureRepository struct {
	db *gorm.DB
}

func NewQuickCaptureRepository(db *gorm.DB) *QuickCaptureRepository {
	return &QuickCaptureRepository{db: db}
}

// WithTx returns a shallow copy of the repository scoped to the given transaction.
func (r *QuickCaptureRepository) WithTx(tx *gorm.DB) *QuickCaptureRepository {
	return &QuickCaptureRepository{db: tx}
}

func (r *QuickCaptureRepository) CreateDraft(draft *models.QuickCaptureDraft) error {
	return r.db.Create(draft).Error
}

func (r *QuickCaptureRepository) CreateDraftWithImages(
	draft *models.QuickCaptureDraft,
	createdEvent *models.DraftLifecycleEvent,
	buildImages func(draftID uint) ([]models.QuickCaptureDraftImage, error),
) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(draft).Error; err != nil {
			return err
		}

		createdEvent.DraftID = draft.ID
		if err := tx.Create(createdEvent).Error; err != nil {
			return err
		}

		images, err := buildImages(draft.ID)
		if err != nil {
			return err
		}
		for _, image := range images {
			if err := tx.Create(&image).Error; err != nil {
				return err
			}
			event := models.DraftLifecycleEvent{
				DraftID:   draft.ID,
				UserID:    image.UserID,
				EventType: models.DraftLifecycleEventImageAdded,
				Message:   "Draft image added",
				CreatedAt: time.Now().UTC(),
			}
			if err := tx.Create(&event).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *QuickCaptureRepository) GetDraftForOwner(draftID, userID uint) (*models.QuickCaptureDraft, error) {
	var draft models.QuickCaptureDraft
	err := r.db.Preload("Images", func(db *gorm.DB) *gorm.DB {
		return db.Order("display_order ASC, id ASC")
	}).Where("id = ? AND user_id = ?", draftID, userID).First(&draft).Error
	if err != nil {
		return nil, err
	}
	return &draft, nil
}

func (r *QuickCaptureRepository) ListDraftsForOwner(userID uint, status models.QuickCaptureDraftStatus, page, limit int) ([]models.QuickCaptureDraft, int64, error) {
	var drafts []models.QuickCaptureDraft
	var total int64
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}
	query := r.db.Model(&models.QuickCaptureDraft{}).Where("user_id = ? AND status = ?", userID, status)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := query.Preload("Images", func(db *gorm.DB) *gorm.DB {
		return db.Order("display_order ASC, id ASC")
	}).Order("updated_at DESC").Offset((page - 1) * limit).Limit(limit).Find(&drafts).Error
	return drafts, total, err
}

func (r *QuickCaptureRepository) AddDraftImage(image *models.QuickCaptureDraftImage) error {
	return r.db.Create(image).Error
}

func (r *QuickCaptureRepository) AddLifecycleEvent(event *models.DraftLifecycleEvent) error {
	return r.db.Create(event).Error
}

func (r *QuickCaptureRepository) DiscardDraft(draftID, userID uint) (*models.QuickCaptureDraft, error) {
	now := time.Now().UTC()
	return r.updateDraftStatus(draftID, userID, models.QuickCaptureDraftStatusDiscarded, &now)
}

func (r *QuickCaptureRepository) updateDraftStatus(draftID, userID uint, status models.QuickCaptureDraftStatus, discardedAt *time.Time) (*models.QuickCaptureDraft, error) {
	_, err := r.GetDraftForOwner(draftID, userID)
	if err != nil {
		return nil, err
	}
	updates := map[string]interface{}{"status": status}
	if discardedAt != nil {
		updates["discarded_at"] = discardedAt
	}
	if err := r.db.Model(&models.QuickCaptureDraft{}).
		Where("id = ? AND user_id = ?", draftID, userID).
		Updates(updates).Error; err != nil {
		return nil, err
	}
	return r.GetDraftForOwner(draftID, userID)
}

func (r *QuickCaptureRepository) DB() *gorm.DB {
	return r.db
}

// UpdateDraftTransaction atomically updates scalar fields, removes images, adds new images, and records a lifecycle event.
// It returns the refreshed draft and the file paths of any images removed (for caller clean-up).
// Returns ErrDraftNotEditable if the draft is not in active state.
func (r *QuickCaptureRepository) UpdateDraftTransaction(
	draftID, userID uint,
	fieldUpdates map[string]interface{},
	removeByIDs []uint,
	removeByTypes []models.ImageType,
	addImages []models.QuickCaptureDraftImage,
	event *models.DraftLifecycleEvent,
) (draft *models.QuickCaptureDraft, removedFilePaths []string, err error) {
	err = r.db.Transaction(func(tx *gorm.DB) error {
		var d models.QuickCaptureDraft
		if err2 := tx.Preload("Images", func(db *gorm.DB) *gorm.DB {
			return db.Order("display_order ASC, id ASC")
		}).Where("id = ? AND user_id = ?", draftID, userID).First(&d).Error; err2 != nil {
			return err2
		}
		if d.Status != models.QuickCaptureDraftStatusActive {
			return ErrDraftNotEditable
		}

		// Collect IDs to remove
		removeIDSet := make(map[uint]bool, len(removeByIDs))
		for _, id := range removeByIDs {
			removeIDSet[id] = true
		}
		removeTypeSet := make(map[models.ImageType]bool, len(removeByTypes))
		for _, t := range removeByTypes {
			removeTypeSet[t] = true
		}

		for _, img := range d.Images {
			if removeIDSet[img.ID] || removeTypeSet[img.ImageType] {
				removedFilePaths = append(removedFilePaths, img.FilePath)
				if err2 := tx.Exec("DELETE FROM quick_capture_draft_images WHERE id = ? AND draft_id = ? AND user_id = ?", img.ID, draftID, userID).Error; err2 != nil {
					return err2
				}
			}
		}

		// Update scalar fields
		if len(fieldUpdates) > 0 {
			if err2 := tx.Model(&models.QuickCaptureDraft{}).
				Where("id = ? AND user_id = ?", draftID, userID).
				Updates(fieldUpdates).Error; err2 != nil {
				return err2
			}
		}

		// Add new images
		for i := range addImages {
			addImages[i].DraftID = draftID
			if err2 := tx.Create(&addImages[i]).Error; err2 != nil {
				return err2
			}
		}

		// Lifecycle event
		event.DraftID = draftID
		if err2 := tx.Create(event).Error; err2 != nil {
			return err2
		}

		// Reload updated draft
		var refreshed models.QuickCaptureDraft
		if err2 := tx.Preload("Images", func(db *gorm.DB) *gorm.DB {
			return db.Order("display_order ASC, id ASC")
		}).First(&refreshed, draftID).Error; err2 != nil {
			return err2
		}
		draft = &refreshed
		return nil
	})
	return
}

// PromoteDraftTransaction atomically claims an active draft for promotion, creates the coin and its images,
// marks the draft as promoted, and records a lifecycle event.
// Returns ErrDraftNotClaimable when the draft is not active or another request claimed it concurrently.
func (r *QuickCaptureRepository) PromoteDraftTransaction(
	draftID, userID uint,
	coin *models.Coin,
) (draft *models.QuickCaptureDraft, createdCoin *models.Coin, err error) {
	err = r.db.Transaction(func(tx *gorm.DB) error {
		// CAS: claim exactly one active row
		result := tx.Model(&models.QuickCaptureDraft{}).
			Where("id = ? AND user_id = ? AND status = ?", draftID, userID, models.QuickCaptureDraftStatusActive).
			Update("status", models.QuickCaptureDraftStatusPromoting)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return ErrDraftNotClaimable
		}

		// Load draft with images for image transfer
		var d models.QuickCaptureDraft
		if err2 := tx.Preload("Images").First(&d, draftID).Error; err2 != nil {
			return err2
		}

		// Persist coin
		if err2 := tx.Create(coin).Error; err2 != nil {
			return err2
		}
		createdCoin = coin

		// Transfer draft images → coin images
		for _, di := range d.Images {
			ci := models.CoinImage{
				CoinID:    coin.ID,
				FilePath:  di.FilePath,
				ImageType: di.ImageType,
				IsPrimary: di.IsPrimary,
			}
			if err2 := tx.Create(&ci).Error; err2 != nil {
				return err2
			}
		}

		if err2 := recordValueSnapshotInTx(tx, userID); err2 != nil {
			return err2
		}

		// Mark draft promoted
		now := time.Now().UTC()
		if err2 := tx.Model(&models.QuickCaptureDraft{}).
			Where("id = ? AND user_id = ?", draftID, userID).
			Updates(map[string]interface{}{
				"status":           models.QuickCaptureDraftStatusPromoted,
				"promoted_coin_id": coin.ID,
				"promoted_at":      now,
			}).Error; err2 != nil {
			return err2
		}
		d.Status = models.QuickCaptureDraftStatusPromoted
		d.PromotedCoinID = &coin.ID
		d.PromotedAt = &now
		draft = &d

		// Lifecycle event
		promotedEvent := models.DraftLifecycleEvent{
			DraftID:   draftID,
			UserID:    userID,
			EventType: models.DraftLifecycleEventPromoted,
			Message:   "Draft promoted to coin",
			CoinID:    &coin.ID,
			CreatedAt: time.Now().UTC(),
		}
		return tx.Create(&promotedEvent).Error
	})
	return
}

func recordValueSnapshotInTx(tx *gorm.DB, userID uint) error {
	type result struct {
		TotalValue    float64
		TotalInvested float64
		CoinCount     int64
	}
	var res result
	if err := tx.Model(&models.Coin{}).
		Select("COALESCE(SUM(current_value), 0) as total_value, COALESCE(SUM(purchase_price), 0) as total_invested, COUNT(*) as coin_count").
		Where("user_id = ? AND is_wishlist = ?", userID, false).
		Scan(&res).Error; err != nil {
		return err
	}

	snapshot := models.ValueSnapshot{
		UserID:        userID,
		TotalValue:    res.TotalValue,
		TotalInvested: res.TotalInvested,
		CoinCount:     res.CoinCount,
		RecordedAt:    time.Now(),
	}
	return tx.Create(&snapshot).Error
}
