package repository

import (
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"gorm.io/gorm"
)

type CoinIntakeDraftRepository struct {
	db *gorm.DB
}

func NewCoinIntakeDraftRepository(db *gorm.DB) *CoinIntakeDraftRepository {
	return &CoinIntakeDraftRepository{db: db}
}

func (r *CoinIntakeDraftRepository) WithTx(tx *gorm.DB) *CoinIntakeDraftRepository {
	return &CoinIntakeDraftRepository{db: tx}
}

func (r *CoinIntakeDraftRepository) DB() *gorm.DB {
	return r.db
}

func (r *CoinIntakeDraftRepository) Create(draft *models.CoinIntakeDraft) error {
	return r.db.Create(draft).Error
}

func (r *CoinIntakeDraftRepository) FindByIDForUser(draftID, userID uint) (*models.CoinIntakeDraft, error) {
	var draft models.CoinIntakeDraft
	err := r.db.Where("id = ? AND user_id = ?", draftID, userID).First(&draft).Error
	if err != nil {
		return nil, err
	}
	return &draft, nil
}

func (r *CoinIntakeDraftRepository) UpdateStatus(draftID, userID uint, status string) error {
	return r.db.Model(&models.CoinIntakeDraft{}).
		Where("id = ? AND user_id = ?", draftID, userID).
		Update("status", status).
		Error
}

func (r *CoinIntakeDraftRepository) MarkConfirmedIfDrafted(draftID, userID uint, confirmedAt time.Time) (int64, error) {
	res := r.db.Model(&models.CoinIntakeDraft{}).
		Where("id = ? AND user_id = ? AND status = ?", draftID, userID, models.CoinIntakeDraftStatusDrafted).
		Updates(map[string]interface{}{
			"status":       models.CoinIntakeDraftStatusConfirmed,
			"confirmed_at": confirmedAt,
		})
	return res.RowsAffected, res.Error
}

func (r *CoinIntakeDraftRepository) AttachConfirmedCoin(draftID, userID, coinID uint) error {
	return r.db.Model(&models.CoinIntakeDraft{}).
		Where("id = ? AND user_id = ?", draftID, userID).
		Update("confirmed_coin_id", coinID).
		Error
}
