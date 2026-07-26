package repository

import (
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"gorm.io/gorm"
)

// SetBuilderRepository owns persistence for Agentic set builder runs and proposals.
type SetBuilderRepository struct {
	db *gorm.DB
}

// NewSetBuilderRepository creates a new SetBuilderRepository.
func NewSetBuilderRepository(db *gorm.DB) *SetBuilderRepository {
	return &SetBuilderRepository{db: db}
}

// CreateRun inserts a new builder run.
func (r *SetBuilderRepository) CreateRun(run *models.SetBuilderRun) error {
	return r.db.Create(run).Error
}

// GetRunForUser returns a run scoped to the owning user.
func (r *SetBuilderRepository) GetRunForUser(runID, userID uint) (*models.SetBuilderRun, error) {
	var run models.SetBuilderRun
	if err := r.db.Scopes(OwnedByID(runID, userID)).First(&run).Error; err != nil {
		return nil, err
	}
	return &run, nil
}

// StartRun atomically marks a queued run as running.
func (r *SetBuilderRepository) StartRun(runID, userID uint, startedAt time.Time) error {
	result := r.db.Model(&models.SetBuilderRun{}).
		Where("id = ? AND user_id = ? AND status = ?", runID, userID, models.SetBuilderRunStatusQueued).
		Updates(map[string]interface{}{
			"status":     models.SetBuilderRunStatusRunning,
			"started_at": startedAt,
			"updated_at": startedAt,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// CompleteRun records the successful workflow result metadata.
func (r *SetBuilderRepository) CompleteRun(runID, userID uint, completedAt time.Time, transcriptSummary string, usedTurns, tokensUsed *int) error {
	result := r.db.Model(&models.SetBuilderRun{}).
		Where("id = ? AND user_id = ?", runID, userID).
		Updates(map[string]interface{}{
			"status":             models.SetBuilderRunStatusCompleted,
			"transcript_summary": transcriptSummary,
			"error_message":      "",
			"used_turns":         usedTurns,
			"tokens_used":        tokensUsed,
			"completed_at":       completedAt,
			"updated_at":         completedAt,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// FailRun records a visible workflow failure.
func (r *SetBuilderRepository) FailRun(runID, userID uint, failedAt time.Time, message, terminationReason string) error {
	result := r.db.Model(&models.SetBuilderRun{}).
		Where("id = ? AND user_id = ?", runID, userID).
		Updates(map[string]interface{}{
			"status":             models.SetBuilderRunStatusFailed,
			"error_message":      message,
			"termination_reason": terminationReason,
			"completed_at":       failedAt,
			"updated_at":         failedAt,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// CreateProposalWithSlots stores a pending review proposal and its roster without creating any set.
func (r *SetBuilderRepository) CreateProposalWithSlots(proposal *models.SetProposal, slots []models.ProposalSlot) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(proposal).Error; err != nil {
			return err
		}
		for i := range slots {
			slots[i].ProposalID = proposal.ID
			if err := tx.Create(&slots[i]).Error; err != nil {
				return err
			}
		}
		return tx.Preload("Run").Preload("Slots", func(db *gorm.DB) *gorm.DB {
			return db.Order("proposal_slots.sort_order ASC")
		}).First(proposal, proposal.ID).Error
	})
}

// FindPendingProposalByIdempotencyKey returns an existing unexpired pending proposal for duplicate prompt submission.
func (r *SetBuilderRepository) FindPendingProposalByIdempotencyKey(userID uint, idempotencyKey string, now time.Time) (*models.SetProposal, error) {
	var proposal models.SetProposal
	err := r.db.
		Where("user_id = ? AND idempotency_key = ? AND status = ? AND expires_at > ?", userID, idempotencyKey, models.SetProposalStatusPending, now).
		Order("created_at ASC").
		Preload("Run").
		Preload("Slots", func(db *gorm.DB) *gorm.DB {
			return db.Order("proposal_slots.sort_order ASC")
		}).
		First(&proposal).Error
	if err != nil {
		return nil, err
	}
	return &proposal, nil
}

// ListProposals returns the current user's proposals ordered newest first.
func (r *SetBuilderRepository) ListProposals(userID uint, limit int) ([]models.SetProposal, error) {
	if limit <= 0 || limit > 100 {
		limit = 100
	}
	var proposals []models.SetProposal
	err := r.db.Scopes(OwnedBy(userID)).
		Order("created_at DESC").
		Limit(limit).
		Find(&proposals).Error
	return proposals, err
}

// GetProposalForUser loads one proposal, scoped to the owning user, with its run and slots.
func (r *SetBuilderRepository) GetProposalForUser(proposalID, userID uint) (*models.SetProposal, error) {
	var proposal models.SetProposal
	err := r.db.Scopes(OwnedByID(proposalID, userID)).
		Preload("Run").
		Preload("Slots", func(db *gorm.DB) *gorm.DB {
			return db.Order("proposal_slots.sort_order ASC")
		}).
		First(&proposal).Error
	if err != nil {
		return nil, err
	}
	return &proposal, nil
}

// MarkApproved atomically transitions a pending proposal to approved and records the created set ID.
func (r *SetBuilderRepository) MarkApproved(proposalID, userID, setID uint, approvedAt time.Time) error {
	result := r.db.Model(&models.SetProposal{}).
		Where("id = ? AND user_id = ? AND status = ?", proposalID, userID, models.SetProposalStatusPending).
		Updates(map[string]interface{}{
			"status":          models.SetProposalStatusApproved,
			"approved_at":     approvedAt,
			"approval_set_id": setID,
			"updated_at":      approvedAt,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// MarkRejected atomically rejects a pending proposal without creating a set.
func (r *SetBuilderRepository) MarkRejected(proposalID, userID uint, rejectedAt time.Time, reason string) error {
	result := r.db.Model(&models.SetProposal{}).
		Where("id = ? AND user_id = ? AND status = ?", proposalID, userID, models.SetProposalStatusPending).
		Updates(map[string]interface{}{
			"status":           models.SetProposalStatusRejected,
			"rejected_at":      rejectedAt,
			"rejection_reason": reason,
			"updated_at":       rejectedAt,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// MarkCreationFailed keeps a proposal reviewable after approval-time creation fails.
func (r *SetBuilderRepository) MarkCreationFailed(proposalID, userID uint, failedAt time.Time, message string) error {
	result := r.db.Model(&models.SetProposal{}).
		Where("id = ? AND user_id = ?", proposalID, userID).
		Updates(map[string]interface{}{
			"status":        models.SetProposalStatusCreationFailed,
			"error_message": message,
			"updated_at":    failedAt,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// MarkExpired atomically expires a pending proposal.
func (r *SetBuilderRepository) MarkExpired(proposalID, userID uint, expiredAt time.Time) error {
	result := r.db.Model(&models.SetProposal{}).
		Where("id = ? AND user_id = ? AND status = ?", proposalID, userID, models.SetProposalStatusPending).
		Updates(map[string]interface{}{
			"status":     models.SetProposalStatusExpired,
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
