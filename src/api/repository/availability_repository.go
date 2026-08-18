package repository

import (
	"fmt"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"gorm.io/gorm"
)

// AvailabilityRepository encapsulates all availability-check related DB operations.
type AvailabilityRepository struct {
	db        *gorm.DB
	cycleRepo *AvailabilityCycleRepository
}

// NewAvailabilityRepository creates a new AvailabilityRepository.
func NewAvailabilityRepository(db *gorm.DB) *AvailabilityRepository {
	return &AvailabilityRepository{db: db}
}

// WithCycleRepo attaches the AvailabilityCycleRepository so that CompleteChildRun/FailChildRun
// can aggregate and finalize the parent AvailabilityCycle atomically in the same transaction
// as the child run's terminal update. Optional: child runs with a nil CycleID (owner-triggered
// checks) work correctly even when this is never set.
func (r *AvailabilityRepository) WithCycleRepo(cycleRepo *AvailabilityCycleRepository) *AvailabilityRepository {
	r.cycleRepo = cycleRepo
	return r
}

// CreateRun inserts a new availability run.
func (r *AvailabilityRepository) CreateRun(run *models.AvailabilityRun) error {
	return r.db.Create(run).Error
}

// CompleteRun updates a run's stats, sets status to completed, and records the completion timestamp.
// Retention for legacy/global runs is no longer pruned automatically here — retention is now
// per-owner (see PruneTerminalChildRunsForOwner) and per-cycle (see AvailabilityCycleRepository.
// PruneOldCycles). PruneOldRuns remains available for any existing callers/tests that need it.
func (r *AvailabilityRepository) CompleteRun(run *models.AvailabilityRun) error {
	return r.db.Model(run).Updates(map[string]interface{}{
		"status":        models.AvailabilityRunStatusCompleted,
		"coins_checked": run.CoinsChecked,
		"available":     run.Available,
		"unavailable":   run.Unavailable,
		"unknown":       run.Unknown,
		"errors":        run.Errors,
		"duration_ms":   run.DurationMs,
		"completed_at":  run.CompletedAt,
	}).Error
}

// CreateChildRun inserts a new child AvailabilityRun. Child runs always belong to exactly one
// owner; UserID == 0 (the legacy admin-run sentinel) is rejected here (FR-002, SC-001).
func (r *AvailabilityRepository) CreateChildRun(run *models.AvailabilityRun) error {
	if run.UserID == 0 {
		return fmt.Errorf("child availability run requires UserID > 0")
	}
	return r.db.Create(run).Error
}

// CompleteChildRun finalizes a child run as completed, then — inside the same transaction —
// aggregates and updates its parent cycle's child counts (and finalizes the cycle if every
// child has reached a terminal state). Returns whether the parent cycle was finalized by this
// call. Post-commit, it prunes this owner's terminal child runs to the retention limit and, if
// the parent was finalized, prunes old cycles. Never touches queued/running rows.
func (r *AvailabilityRepository) CompleteChildRun(run *models.AvailabilityRun) (bool, error) {
	now := time.Now()
	run.Status = models.AvailabilityRunStatusCompleted
	run.CompletedAt = &now
	run.DurationMs = now.Sub(run.StartedAt).Milliseconds()
	return r.completeChildRunTx(run)
}

// FailChildRun finalizes a child run as failed with the given (already-generic) message, then
// performs the same parent aggregation/finalization/pruning as CompleteChildRun.
func (r *AvailabilityRepository) FailChildRun(run *models.AvailabilityRun, message string) (bool, error) {
	now := time.Now()
	run.Status = models.AvailabilityRunStatusFailed
	run.FailMessage = message
	run.CompletedAt = &now
	run.DurationMs = now.Sub(run.StartedAt).Milliseconds()
	return r.completeChildRunTx(run)
}

func (r *AvailabilityRepository) completeChildRunTx(run *models.AvailabilityRun) (bool, error) {
	parentFinalized := false
	err := r.db.Transaction(func(tx *gorm.DB) error {
		updates := map[string]interface{}{
			"status":        run.Status,
			"coins_checked": run.CoinsChecked,
			"available":     run.Available,
			"unavailable":   run.Unavailable,
			"unknown":       run.Unknown,
			"errors":        run.Errors,
			"duration_ms":   run.DurationMs,
			"completed_at":  run.CompletedAt,
			"fail_message":  run.FailMessage,
		}
		if err := tx.Model(&models.AvailabilityRun{}).Where("id = ?", run.ID).Updates(updates).Error; err != nil {
			return err
		}
		if run.CycleID == nil || r.cycleRepo == nil {
			return nil
		}
		counts, err := r.cycleRepo.aggregateChildCountsTx(tx, *run.CycleID)
		if err != nil {
			return err
		}
		status, terminal := deriveCycleStatus(counts)
		cycleUpdates := map[string]interface{}{
			"total_children":     counts.Total,
			"queued_children":    counts.Queued,
			"running_children":   counts.Running,
			"completed_children": counts.Completed,
			"failed_children":    counts.Failed,
			"status":             status,
		}
		if terminal {
			finalizedAt := time.Now()
			cycleUpdates["completed_at"] = &finalizedAt
			if status != models.AvailabilityCycleStatusCompleted {
				cycleUpdates["fail_message"] = models.GenericAvailabilityFailureMessage
			}
		}
		if err := tx.Model(&models.AvailabilityCycle{}).Where("id = ?", *run.CycleID).Updates(cycleUpdates).Error; err != nil {
			return err
		}
		parentFinalized = terminal
		return nil
	})
	if err != nil {
		return false, err
	}

	r.PruneTerminalChildRunsForOwner(run.UserID, 20)
	if parentFinalized && run.CycleID != nil && r.cycleRepo != nil {
		r.cycleRepo.PruneOldCycles(20)
	}
	return parentFinalized, nil
}

// RecoverStaleChildRuns marks child runs (belonging to a cycle) stuck in "running" past the
// timeout as failed with the generic failure message, aggregating/finalizing their parent
// cycle as a side effect. Child runs are executed synchronously (not individually queued), so
// a crash mid-execution can only be recovered by failing the orphaned child outright — there is
// no queued state to resume from.
func (r *AvailabilityRepository) RecoverStaleChildRuns(timeout time.Duration) {
	cutoff := time.Now().Add(-timeout)
	var stale []models.AvailabilityRun
	if err := r.db.Where("status = ? AND cycle_id IS NOT NULL AND started_at < ?",
		models.AvailabilityRunStatusRunning, cutoff).Find(&stale).Error; err != nil {
		return
	}
	for i := range stale {
		run := stale[i]
		_, _ = r.FailChildRun(&run, models.GenericAvailabilityFailureMessage)
	}
}

// ListRunsForOwner returns paginated child availability runs for a single owner, newest first.
func (r *AvailabilityRepository) ListRunsForOwner(userID uint, page, limit int) ([]models.AvailabilityRun, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	var total int64
	if err := r.db.Model(&models.AvailabilityRun{}).Where("user_id = ?", userID).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var runs []models.AvailabilityRun
	offset := (page - 1) * limit
	err := r.db.Where("user_id = ?", userID).
		Order("started_at DESC").Offset(offset).Limit(limit).Find(&runs).Error
	return runs, total, err
}

// GetOwnedRunWithResults returns a single run with its per-coin results, scoped to the owner —
// refuses to return a run belonging to a different user (404 for cross-owner reads).
func (r *AvailabilityRepository) GetOwnedRunWithResults(userID, runID uint) (*models.AvailabilityRun, error) {
	var run models.AvailabilityRun
	err := r.db.Preload("Results").
		Where("id = ? AND user_id = ?", runID, userID).
		First(&run).Error
	if err != nil {
		return nil, err
	}
	return &run, nil
}

var terminalRunStatuses = []string{models.AvailabilityRunStatusCompleted, models.AvailabilityRunStatusFailed}

// PruneTerminalChildRunsForOwner keeps only the most recent `keep` terminal child runs for a
// single owner, deleting older terminal runs and their results. Never touches queued/running rows.
func (r *AvailabilityRepository) PruneTerminalChildRunsForOwner(userID uint, keep int) {
	var count int64
	r.db.Model(&models.AvailabilityRun{}).
		Where("user_id = ? AND status IN ?", userID, terminalRunStatuses).
		Count(&count)
	if count <= int64(keep) {
		return
	}

	var cutoffRun models.AvailabilityRun
	if err := r.db.Where("user_id = ? AND status IN ?", userID, terminalRunStatuses).
		Order("completed_at DESC").Offset(keep).Limit(1).First(&cutoffRun).Error; err != nil {
		return
	}
	if cutoffRun.CompletedAt == nil {
		return
	}

	r.db.Transaction(func(tx *gorm.DB) error {
		idsSubquery := tx.Model(&models.AvailabilityRun{}).Select("id").
			Where("user_id = ? AND status IN ? AND completed_at <= ?", userID, terminalRunStatuses, cutoffRun.CompletedAt)
		if err := tx.Where("run_id IN (?)", idsSubquery).Delete(&models.AvailabilityResult{}).Error; err != nil {
			return err
		}
		return tx.Where("user_id = ? AND status IN ? AND completed_at <= ?", userID, terminalRunStatuses, cutoffRun.CompletedAt).
			Delete(&models.AvailabilityRun{}).Error
	})
}

// CreateResult inserts a single availability check result.
func (r *AvailabilityRepository) CreateResult(result *models.AvailabilityResult) error {
	return r.db.Create(result).Error
}

// UpdateResult updates an existing availability check result (used by agent escalation).
func (r *AvailabilityRepository) UpdateResult(result *models.AvailabilityResult) error {
	return r.db.Model(result).Updates(map[string]interface{}{
		"status":     result.Status,
		"reason":     result.Reason,
		"agent_used": result.AgentUsed,
	}).Error
}

// EnqueueManualRun creates a queued manual run if no queued or running manual run exists
// within the given since window (duplicate-run protection).
func (r *AvailabilityRepository) EnqueueManualRun(run *models.AvailabilityRun, since time.Time) (bool, error) {
	acquired := false
	err := r.db.Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Model(&models.AvailabilityRun{}).
			Where("trigger_type = ? AND status IN ? AND started_at >= ?",
				"manual",
				[]string{models.AvailabilityRunStatusQueued, models.AvailabilityRunStatusRunning},
				since,
			).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return nil
		}
		if err := tx.Create(run).Error; err != nil {
			return err
		}
		acquired = true
		return nil
	})
	return acquired, err
}

// ClaimQueuedRun atomically transitions a queued run to running status.
// Returns (run, true, nil) when claimed, (nil, false, nil) if already claimed.
func (r *AvailabilityRepository) ClaimQueuedRun(runID uint) (*models.AvailabilityRun, bool, error) {
	now := time.Now()
	var run models.AvailabilityRun
	claimed := false
	err := r.db.Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&models.AvailabilityRun{}).
			Where("id = ? AND status = ?", runID, models.AvailabilityRunStatusQueued).
			Updates(map[string]interface{}{
				"status":     models.AvailabilityRunStatusRunning,
				"started_at": now,
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

// FailRun marks a run as failed with an error message.
func (r *AvailabilityRepository) FailRun(run *models.AvailabilityRun, message string) error {
	now := time.Now()
	return r.db.Model(run).Updates(map[string]interface{}{
		"status":       models.AvailabilityRunStatusFailed,
		"fail_message": message,
		"completed_at": &now,
		"duration_ms":  now.Sub(run.StartedAt).Milliseconds(),
	}).Error
}

// RecoverStaleRuns resets stuck running runs (started before the timeout cutoff) back to queued
// and returns the IDs of all currently queued runs.
func (r *AvailabilityRepository) RecoverStaleRuns(timeout time.Duration) ([]uint, error) {
	cutoff := time.Now().Add(-timeout)
	if err := r.db.Model(&models.AvailabilityRun{}).
		Where("status = ? AND started_at < ?", models.AvailabilityRunStatusRunning, cutoff).
		Updates(map[string]interface{}{
			"status": models.AvailabilityRunStatusQueued,
		}).Error; err != nil {
		return nil, err
	}
	var ids []uint
	err := r.db.Model(&models.AvailabilityRun{}).
		Where("status = ?", models.AvailabilityRunStatusQueued).
		Order("started_at ASC").
		Pluck("id", &ids).Error
	return ids, err
}

// GetLastScheduledRun returns the most recent completed "scheduled" availability run, or nil if none.
func (r *AvailabilityRepository) GetLastScheduledRun() *models.AvailabilityRun {
	var run models.AvailabilityRun
	err := r.db.Where("trigger_type = ? AND completed_at IS NOT NULL", "scheduled").
		Order("started_at DESC").Limit(1).First(&run).Error
	if err != nil {
		return nil
	}
	return &run
}

// ListRuns returns paginated availability runs, newest first.
func (r *AvailabilityRepository) ListRuns(page, limit int) ([]models.AvailabilityRun, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	var total int64
	if err := r.db.Model(&models.AvailabilityRun{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var runs []models.AvailabilityRun
	offset := (page - 1) * limit
	err := r.db.Preload("User", func(db *gorm.DB) *gorm.DB {
		return db.Select("id, username")
	}).Order("started_at DESC").Offset(offset).Limit(limit).Find(&runs).Error
	if err != nil {
		return nil, 0, err
	}
	// Populate UserName from preloaded User
	for i := range runs {
		runs[i].UserName = runs[i].User.Username
	}
	return runs, total, err
}

// GetRunWithResults returns a single run with all its per-coin results.
func (r *AvailabilityRepository) GetRunWithResults(runID uint) (*models.AvailabilityRun, error) {
	var run models.AvailabilityRun
	err := r.db.Preload("Results").Preload("User", func(db *gorm.DB) *gorm.DB {
		return db.Select("id, username")
	}).First(&run, runID).Error
	if err != nil {
		return nil, err
	}
	run.UserName = run.User.Username
	return &run, nil
}

// PruneOldRuns keeps only the most recent `keep` runs, deleting older runs and their results.
func (r *AvailabilityRepository) PruneOldRuns(keep int) {
	var count int64
	r.db.Model(&models.AvailabilityRun{}).Count(&count)
	if count <= int64(keep) {
		return
	}

	var cutoffRun models.AvailabilityRun
	if err := r.db.Order("started_at DESC").Offset(keep).Limit(1).First(&cutoffRun).Error; err != nil {
		return
	}

	// Delete results and runs in a single transaction
	r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("run_id IN (?)",
			tx.Model(&models.AvailabilityRun{}).Select("id").Where("started_at <= ?", cutoffRun.StartedAt),
		).Delete(&models.AvailabilityResult{}).Error; err != nil {
			return err
		}
		return tx.Where("started_at <= ?", cutoffRun.StartedAt).Delete(&models.AvailabilityRun{}).Error
	})
}
