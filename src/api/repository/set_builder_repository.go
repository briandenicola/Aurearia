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

// ClaimQueuedRun atomically transitions a queued run to running status.
func (r *SetBuilderRepository) ClaimQueuedRun(runID uint, startedAt time.Time) (*models.SetBuilderRun, bool, error) {
	var run models.SetBuilderRun
	claimed := false
	err := r.db.Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&models.SetBuilderRun{}).
			Where("id = ? AND status = ?", runID, models.SetBuilderRunStatusQueued).
			Updates(map[string]interface{}{
				"status":     models.SetBuilderRunStatusRunning,
				"started_at": startedAt,
				"updated_at": startedAt,
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return nil
		}
		claimed = true
		return tx.First(&run, runID).Error
	})
	return &run, claimed, err
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

// RecoverStaleRuns resets stuck running runs to queued and returns all queued run IDs.
func (r *SetBuilderRepository) RecoverStaleRuns(timeout time.Duration) ([]uint, error) {
	cutoff := time.Now().Add(-timeout)
	if err := r.db.Model(&models.SetBuilderRun{}).
		Where("status = ? AND started_at < ?", models.SetBuilderRunStatusRunning, cutoff).
		Updates(map[string]interface{}{
			"status":     models.SetBuilderRunStatusQueued,
			"started_at": nil,
			"updated_at": time.Now(),
		}).Error; err != nil {
		return nil, err
	}
	var ids []uint
	err := r.db.Model(&models.SetBuilderRun{}).
		Where("status = ?", models.SetBuilderRunStatusQueued).
		Order("created_at ASC").
		Pluck("id", &ids).Error
	return ids, err
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

// UpdatePendingProposalWithSlots replaces editable proposal fields and roster slots while pending.
func (r *SetBuilderRepository) UpdatePendingProposalWithSlots(proposalID, userID uint, updates map[string]interface{}, slots []models.ProposalSlot) (*models.SetProposal, error) {
	var proposal models.SetProposal
	err := r.db.Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&models.SetProposal{}).
			Where("id = ? AND user_id = ? AND status = ?", proposalID, userID, models.SetProposalStatusPending).
			Updates(updates)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		if len(slots) > 0 {
			if err := tx.Where("proposal_id = ?", proposalID).Delete(&models.ProposalSlot{}).Error; err != nil {
				return err
			}
			for i := range slots {
				slots[i].ProposalID = proposalID
				if err := tx.Create(&slots[i]).Error; err != nil {
					return err
				}
			}
		}
		return tx.Preload("Run").Preload("Slots", func(db *gorm.DB) *gorm.DB {
			return db.Order("proposal_slots.sort_order ASC")
		}).First(&proposal, proposalID).Error
	})
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

// ApproveProposalWithSet atomically creates the approved Agentic set roster and marks the proposal approved.
func (r *SetBuilderRepository) ApproveProposalWithSet(proposalID, userID uint, set *models.CoinSet, targets []models.CoinSetTarget, approvedAt time.Time) (*models.CoinSet, error) {
	var approvedSet models.CoinSet
	err := r.db.Transaction(func(tx *gorm.DB) error {
		var proposal models.SetProposal
		if err := tx.Scopes(OwnedByID(proposalID, userID)).Preload("Slots").First(&proposal).Error; err != nil {
			return err
		}
		if proposal.Status == models.SetProposalStatusApproved && proposal.ApprovalSetID != nil {
			return tx.Scopes(OwnedByID(*proposal.ApprovalSetID, userID)).First(&approvedSet).Error
		}
		if proposal.Status != models.SetProposalStatusPending {
			return gorm.ErrRecordNotFound
		}
		if err := tx.Create(set).Error; err != nil {
			return err
		}
		for i := range targets {
			targets[i].SetID = set.ID
			if err := tx.Create(&targets[i]).Error; err != nil {
				return err
			}
		}
		result := tx.Model(&models.SetProposal{}).
			Where("id = ? AND user_id = ? AND status = ?", proposalID, userID, models.SetProposalStatusPending).
			Updates(map[string]interface{}{
				"status":          models.SetProposalStatusApproved,
				"approved_at":     approvedAt,
				"approval_set_id": set.ID,
				"updated_at":      approvedAt,
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		approvedSet = *set
		return nil
	})
	return &approvedSet, err
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
