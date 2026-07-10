package repository

import (
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"gorm.io/gorm"
)

// AuctionEndingRepository encapsulates all auction-ending-run related DB operations.
type AuctionEndingRepository struct {
	db *gorm.DB
}

// NewAuctionEndingRepository creates a new AuctionEndingRepository.
func NewAuctionEndingRepository(db *gorm.DB) *AuctionEndingRepository {
	return &AuctionEndingRepository{db: db}
}

// CreateRun inserts a new auction ending run.
func (r *AuctionEndingRepository) CreateRun(run *models.AuctionEndingRun) error {
	return r.db.Create(run).Error
}

// CompleteRun updates a run's stats and completion timestamp.
func (r *AuctionEndingRepository) CompleteRun(run *models.AuctionEndingRun) error {
	err := r.db.Model(run).Updates(map[string]interface{}{
		"status":        run.Status,
		"lots_checked":  run.LotsChecked,
		"alerts_sent":   run.AlertsSent,
		"duration_ms":   run.DurationMs,
		"completed_at":  run.CompletedAt,
		"error_message": run.ErrorMessage,
	}).Error
	if err == nil {
		r.PruneOldRuns(100)
	}
	return err
}

// MarkRunning transitions a queued run to running status using an atomic CAS update.
// Returns (run, true, nil) when the claim succeeds, or (nil, false, nil) when the run
// was already claimed by another goroutine (RowsAffected == 0).
func (r *AuctionEndingRepository) MarkRunning(runID uint) (*models.AuctionEndingRun, bool, error) {
	result := r.db.Model(&models.AuctionEndingRun{}).
		Where("id = ? AND status = ?", runID, models.AuctionEndingRunStatusQueued).
		Update("status", models.AuctionEndingRunStatusRunning)
	if result.Error != nil {
		return nil, false, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, false, nil
	}
	var run models.AuctionEndingRun
	if err := r.db.First(&run, runID).Error; err != nil {
		return nil, false, err
	}
	return &run, true, nil
}

// FindActiveRun returns the most recent queued or running auction ending run, or nil if none.
func (r *AuctionEndingRepository) FindActiveRun() *models.AuctionEndingRun {
	var run models.AuctionEndingRun
	err := r.db.Where("status IN ?", []string{
		models.AuctionEndingRunStatusQueued,
		models.AuctionEndingRunStatusRunning,
	}).Order("started_at DESC").Limit(1).First(&run).Error
	if err != nil {
		return nil
	}
	return &run
}

// RecoverStaleRuns marks any queued or running runs older than the given timeout as
// error so they do not block future manual triggers after a process restart.
// Returns the number of runs recovered.
func (r *AuctionEndingRepository) RecoverStaleRuns(timeout time.Duration) int64 {
	cutoff := time.Now().Add(-timeout)
	now := time.Now()
	result := r.db.Model(&models.AuctionEndingRun{}).
		Where("status IN ? AND started_at < ?", []string{
			models.AuctionEndingRunStatusQueued,
			models.AuctionEndingRunStatusRunning,
		}, cutoff).
		Updates(map[string]interface{}{
			"status":        models.AuctionEndingRunStatusError,
			"error_message": "recovered: process restarted while run was in-flight",
			"completed_at":  now,
		})
	return result.RowsAffected
}

// ListRuns returns paginated auction ending runs, newest first.
func (r *AuctionEndingRepository) ListRuns(page, limit int) ([]models.AuctionEndingRun, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	var total int64
	if err := r.db.Model(&models.AuctionEndingRun{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var runs []models.AuctionEndingRun
	offset := (page - 1) * limit
	err := r.db.Order("started_at DESC").Offset(offset).Limit(limit).Find(&runs).Error
	return runs, total, err
}

// GetRunByID returns a single auction ending run by ID.
func (r *AuctionEndingRepository) GetRunByID(runID uint) (*models.AuctionEndingRun, error) {
	var run models.AuctionEndingRun
	err := r.db.First(&run, runID).Error
	if err != nil {
		return nil, err
	}
	return &run, nil
}

// GetLastScheduledRun returns the most recent completed "scheduled" run, or nil if none.
func (r *AuctionEndingRepository) GetLastScheduledRun() *models.AuctionEndingRun {
	var run models.AuctionEndingRun
	err := r.db.Where("trigger_type = ? AND completed_at IS NOT NULL", "scheduled").
		Order("started_at DESC").Limit(1).First(&run).Error
	if err != nil {
		return nil
	}
	return &run
}

// PruneOldRuns keeps only the most recent `keep` runs, deleting older runs.
func (r *AuctionEndingRepository) PruneOldRuns(keep int) {
	var count int64
	r.db.Model(&models.AuctionEndingRun{}).Count(&count)
	if count <= int64(keep) {
		return
	}

	var cutoffRun models.AuctionEndingRun
	if err := r.db.Order("started_at DESC").Offset(keep).Limit(1).First(&cutoffRun).Error; err != nil {
		return
	}

	r.db.Where("started_at <= ?", cutoffRun.StartedAt).Delete(&models.AuctionEndingRun{})
}
