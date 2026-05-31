package repository

import (
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"gorm.io/gorm"
)

// CollectionUpdateRepository encapsulates collection chat proposal persistence.
type CollectionUpdateRepository struct {
	db *gorm.DB
}

// NewCollectionUpdateRepository creates a new CollectionUpdateRepository.
func NewCollectionUpdateRepository(db *gorm.DB) *CollectionUpdateRepository {
	return &CollectionUpdateRepository{db: db}
}

// WithTx returns a shallow copy that uses tx for all operations.
func (r *CollectionUpdateRepository) WithTx(tx *gorm.DB) *CollectionUpdateRepository {
	return &CollectionUpdateRepository{db: tx}
}

// CreateProposal inserts a new proposal.
func (r *CollectionUpdateRepository) CreateProposal(proposal *models.CollectionUpdateProposal) error {
	return r.db.Create(proposal).Error
}

// FindOwnedProposal returns a proposal by id scoped to the owning user.
func (r *CollectionUpdateRepository) FindOwnedProposal(userID uint, proposalID string) (*models.CollectionUpdateProposal, error) {
	var proposal models.CollectionUpdateProposal
	err := r.db.Where("id = ? AND user_id = ?", proposalID, userID).First(&proposal).Error
	if err != nil {
		return nil, err
	}
	return &proposal, nil
}

// MarkCommitted atomically marks a pending proposal as committed.
func (r *CollectionUpdateRepository) MarkCommitted(userID uint, proposalID string, committedAt time.Time) error {
	result := r.db.Model(&models.CollectionUpdateProposal{}).
		Where("id = ? AND user_id = ? AND status = ?", proposalID, userID, models.CollectionUpdateProposalPending).
		Updates(map[string]interface{}{
			"status":       models.CollectionUpdateProposalCommitted,
			"committed_at": committedAt,
			"updated_at":   committedAt,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// MarkCancelled atomically marks a pending proposal as cancelled.
func (r *CollectionUpdateRepository) MarkCancelled(userID uint, proposalID string, cancelledAt time.Time) error {
	result := r.db.Model(&models.CollectionUpdateProposal{}).
		Where("id = ? AND user_id = ? AND status = ?", proposalID, userID, models.CollectionUpdateProposalPending).
		Updates(map[string]interface{}{
			"status":       models.CollectionUpdateProposalCancelled,
			"cancelled_at": cancelledAt,
			"updated_at":   cancelledAt,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// MarkExpired atomically marks a pending proposal as expired.
func (r *CollectionUpdateRepository) MarkExpired(userID uint, proposalID string, expiredAt time.Time) error {
	result := r.db.Model(&models.CollectionUpdateProposal{}).
		Where("id = ? AND user_id = ? AND status = ?", proposalID, userID, models.CollectionUpdateProposalPending).
		Updates(map[string]interface{}{
			"status":     models.CollectionUpdateProposalExpired,
			"updated_at": expiredAt,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
