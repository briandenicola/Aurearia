package repository

import (
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"gorm.io/gorm"
)

// AvailabilityRepository encapsulates all availability-check related DB operations.
type AvailabilityRepository struct {
	db *gorm.DB
}

// NewAvailabilityRepository creates a new AvailabilityRepository.
func NewAvailabilityRepository(db *gorm.DB) *AvailabilityRepository {
	return &AvailabilityRepository{db: db}
}

// CreateRun inserts a new availability run.
func (r *AvailabilityRepository) CreateRun(run *models.AvailabilityRun) error {
	return r.db.Create(run).Error
}

// CompleteRun updates a run's stats, sets status to completed, and records the completion timestamp.
func (r *AvailabilityRepository) CompleteRun(run *models.AvailabilityRun) error {
	err := r.db.Model(run).Updates(map[string]interface{}{
		"status":        models.AvailabilityRunStatusCompleted,
		"coins_checked": run.CoinsChecked,
		"available":     run.Available,
		"unavailable":   run.Unavailable,
		"unknown":       run.Unknown,
		"errors":        run.Errors,
		"duration_ms":   run.DurationMs,
		"completed_at":  run.CompletedAt,
	}).Error
	if err == nil {
		r.PruneOldRuns(100)
	}
	return err
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
