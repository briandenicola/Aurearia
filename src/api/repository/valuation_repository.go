package repository

import (
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"gorm.io/gorm"
)

// ValuationRepository encapsulates all valuation-run related DB operations.
type ValuationRepository struct {
	db *gorm.DB
}

// NewValuationRepository creates a new ValuationRepository.
func NewValuationRepository(db *gorm.DB) *ValuationRepository {
	return &ValuationRepository{db: db}
}

// CreateRun inserts a new valuation run.
func (r *ValuationRepository) CreateRun(run *models.ValuationRun) error {
	return r.db.Create(run).Error
}

// CompleteRun updates a run's stats and completion timestamp.
func (r *ValuationRepository) CompleteRun(run *models.ValuationRun) error {
	err := r.db.Model(run).Updates(map[string]interface{}{
		"status":         run.Status,
		"coins_checked":  run.CoinsChecked,
		"coins_updated":  run.CoinsUpdated,
		"coins_skipped":  run.CoinsSkipped,
		"errors":         run.Errors,
		"duration_ms":    run.DurationMs,
		"completed_at":   run.CompletedAt,
		"error_message":  run.ErrorMessage,
	}).Error
	if err == nil {
		r.PruneOldRuns(100)
	}
	return err
}

// AddResult inserts a single valuation result.
func (r *ValuationRepository) AddResult(result *models.ValuationResult) error {
	return r.db.Create(result).Error
}

// UpdateRunProgress updates the in-progress counters on a running valuation run.
func (r *ValuationRepository) UpdateRunProgress(run *models.ValuationRun) error {
	return r.db.Model(run).Updates(map[string]interface{}{
		"total_coins":   run.TotalCoins,
		"coins_checked": run.CoinsChecked,
		"coins_updated": run.CoinsUpdated,
		"coins_skipped": run.CoinsSkipped,
		"errors":        run.Errors,
	}).Error
}

// HasActiveRun returns true if there is a running valuation for the given user.
func (r *ValuationRepository) HasActiveRun(userID uint) (bool, error) {
	var count int64
	err := r.db.Model(&models.ValuationRun{}).
		Where("user_id = ? AND status = ?", userID, "running").
		Count(&count).Error
	return count > 0, err
}

// CancelRun marks a running valuation run as cancelled in the DB.
func (r *ValuationRepository) CancelRun(runID uint) {
	now := time.Now()
	r.db.Model(&models.ValuationRun{}).Where("id = ? AND status = ?", runID, "running").
		Updates(map[string]interface{}{
			"status":        "cancelled",
			"error_message": "cancelled by user",
			"completed_at":  now,
		})
}

// RecoverStaleRuns marks runs that have been "running" longer than the timeout as failed.
func (r *ValuationRepository) RecoverStaleRuns(timeout time.Duration) {
	cutoff := time.Now().Add(-timeout)
	r.db.Model(&models.ValuationRun{}).
		Where("status = ? AND started_at < ?", "running", cutoff).
		Updates(map[string]interface{}{
			"status":        "failed",
			"error_message": "run timed out (stale recovery)",
			"completed_at":  time.Now(),
		})
}

// GetOwnedCoins returns owned (not sold, not wishlist) coins for a user, limited to maxCoins.
func (r *ValuationRepository) GetOwnedCoins(userID uint, maxCoins int) ([]models.Coin, error) {
	var coins []models.Coin
	err := r.db.Where("user_id = ? AND is_wishlist = ? AND is_sold = ?", userID, false, false).
		Order("updated_at DESC").
		Limit(maxCoins).
		Find(&coins).Error
	return coins, err
}

// ListRuns returns paginated valuation runs, newest first.
func (r *ValuationRepository) ListRuns(page, limit int) ([]models.ValuationRun, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	var total int64
	if err := r.db.Model(&models.ValuationRun{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var runs []models.ValuationRun
	offset := (page - 1) * limit
	err := r.db.Preload("User", func(db *gorm.DB) *gorm.DB {
		return db.Select("id, username")
	}).Order("started_at DESC").Offset(offset).Limit(limit).Find(&runs).Error
	return runs, total, err
}

// GetRunWithResults returns a single run with all its per-coin results.
func (r *ValuationRepository) GetRunWithResults(runID uint) (*models.ValuationRun, error) {
	var run models.ValuationRun
	err := r.db.Preload("Results").Preload("User", func(db *gorm.DB) *gorm.DB {
		return db.Select("id, username")
	}).First(&run, runID).Error
	if err != nil {
		return nil, err
	}
	return &run, nil
}

// GetUsersWithOwnedCoins returns distinct user IDs that have non-sold, non-wishlist coins.
func (r *ValuationRepository) GetUsersWithOwnedCoins() ([]uint, error) {
	var userIDs []uint
	err := r.db.Model(&models.Coin{}).
		Where("is_wishlist = ? AND is_sold = ?", false, false).
		Distinct("user_id").
		Pluck("user_id", &userIDs).Error
	return userIDs, err
}

// PruneOldRuns keeps only the most recent `keep` runs, deleting older runs and their results.
func (r *ValuationRepository) PruneOldRuns(keep int) {
	var count int64
	r.db.Model(&models.ValuationRun{}).Count(&count)
	if count <= int64(keep) {
		return
	}

	var cutoffRun models.ValuationRun
	if err := r.db.Order("started_at DESC").Offset(keep).Limit(1).First(&cutoffRun).Error; err != nil {
		return
	}

	// Delete results for old runs, then the runs themselves
	r.db.Where("run_id IN (?)",
		r.db.Model(&models.ValuationRun{}).Select("id").Where("started_at <= ?", cutoffRun.StartedAt),
	).Delete(&models.ValuationResult{})
	r.db.Where("started_at <= ?", cutoffRun.StartedAt).Delete(&models.ValuationRun{})
}
