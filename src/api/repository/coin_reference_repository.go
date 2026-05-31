package repository

import (
	"github.com/briandenicola/ancient-coins-api/models"
	"gorm.io/gorm"
)

// CoinReferenceRepository handles persistence for structured coin references.
type CoinReferenceRepository struct {
	db *gorm.DB
}

// NewCoinReferenceRepository creates a new CoinReferenceRepository.
func NewCoinReferenceRepository(db *gorm.DB) *CoinReferenceRepository {
	return &CoinReferenceRepository{db: db}
}

// WithTx returns a repository instance bound to a transaction.
func (r *CoinReferenceRepository) WithTx(tx *gorm.DB) *CoinReferenceRepository {
	return &CoinReferenceRepository{db: tx}
}

// CoinExists checks if a coin belongs to the given user.
func (r *CoinReferenceRepository) CoinExists(coinID, userID uint) bool {
	var count int64
	r.db.Model(&models.Coin{}).Scopes(OwnedByID(coinID, userID)).Count(&count)
	return count > 0
}

// ListByCoin returns all references for a user-owned coin.
func (r *CoinReferenceRepository) ListByCoin(coinID, userID uint) ([]models.CoinReference, error) {
	var refs []models.CoinReference
	err := r.db.
		Where("coin_id = ?", coinID).
		Where("coin_id IN (?)", r.db.Model(&models.Coin{}).Scopes(OwnedBy(userID)).Select("id")).
		Order("created_at ASC").
		Find(&refs).Error
	return refs, err
}

// GetByID returns a single reference for a user-owned coin.
func (r *CoinReferenceRepository) GetByID(id, coinID, userID uint) (*models.CoinReference, error) {
	var ref models.CoinReference
	err := r.db.
		Where("id = ? AND coin_id = ?", id, coinID).
		Where("coin_id IN (?)", r.db.Model(&models.Coin{}).Scopes(OwnedBy(userID)).Select("id")).
		First(&ref).Error
	if err != nil {
		return nil, err
	}
	return &ref, nil
}

// Create inserts a new reference.
func (r *CoinReferenceRepository) Create(ref *models.CoinReference) error {
	return r.db.Create(ref).Error
}

// CreateBatch inserts multiple references.
func (r *CoinReferenceRepository) CreateBatch(refs []models.CoinReference) error {
	if len(refs) == 0 {
		return nil
	}
	return r.db.Create(&refs).Error
}

// Update updates fields on an existing reference and reloads it.
func (r *CoinReferenceRepository) Update(ref *models.CoinReference, updates map[string]interface{}) error {
	if err := r.db.Model(ref).Updates(updates).Error; err != nil {
		return err
	}
	return r.db.First(ref, ref.ID).Error
}

// Delete removes a reference owned by the user. Returns rows affected.
func (r *CoinReferenceRepository) Delete(id, coinID, userID uint) (int64, error) {
	result := r.db.
		Where("id = ? AND coin_id = ?", id, coinID).
		Where("coin_id IN (?)", r.db.Model(&models.Coin{}).Scopes(OwnedBy(userID)).Select("id")).
		Delete(&models.CoinReference{})
	return result.RowsAffected, result.Error
}

// DeleteByCoin removes all references for a user-owned coin.
func (r *CoinReferenceRepository) DeleteByCoin(coinID, userID uint) error {
	return r.db.
		Where("coin_id = ?", coinID).
		Where("coin_id IN (?)", r.db.Model(&models.Coin{}).Scopes(OwnedBy(userID)).Select("id")).
		Delete(&models.CoinReference{}).Error
}

// ReplaceForCoin replaces all references for a user-owned coin.
func (r *CoinReferenceRepository) ReplaceForCoin(coinID, userID uint, refs []models.CoinReference) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		txRepo := r.WithTx(tx)
		if err := txRepo.DeleteByCoin(coinID, userID); err != nil {
			return err
		}
		for i := range refs {
			refs[i].CoinID = coinID
		}
		return txRepo.CreateBatch(refs)
	})
}
