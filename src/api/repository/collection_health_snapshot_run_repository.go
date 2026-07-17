package repository

import (
	"github.com/briandenicola/ancient-coins-api/models"
	"gorm.io/gorm"
)

// CollectionHealthSnapshotRunRepository encapsulates collection health snapshot run persistence.
type CollectionHealthSnapshotRunRepository struct {
	db *gorm.DB
}

// NewCollectionHealthSnapshotRunRepository creates a new CollectionHealthSnapshotRunRepository.
func NewCollectionHealthSnapshotRunRepository(db *gorm.DB) *CollectionHealthSnapshotRunRepository {
	return &CollectionHealthSnapshotRunRepository{db: db}
}

// CreateRun inserts a new collection health snapshot run.
func (r *CollectionHealthSnapshotRunRepository) CreateRun(run *models.CollectionHealthSnapshotRun) error {
	return r.db.Create(run).Error
}

// CompleteRun updates a run's stats and completion timestamp.
func (r *CollectionHealthSnapshotRunRepository) CompleteRun(run *models.CollectionHealthSnapshotRun) error {
	err := r.db.Model(run).Updates(map[string]interface{}{
		"status":            run.Status,
		"users_eligible":    run.UsersEligible,
		"users_snapshotted": run.UsersSnapshotted,
		"users_failed":      run.UsersFailed,
		"duration_ms":       run.DurationMs,
		"completed_at":      run.CompletedAt,
		"error_message":     run.ErrorMessage,
	}).Error
	if err == nil {
		r.PruneOldRuns(100)
	}
	return err
}

// ListRuns returns paginated collection health snapshot runs, newest first.
func (r *CollectionHealthSnapshotRunRepository) ListRuns(page, limit int) ([]models.CollectionHealthSnapshotRun, int64, error) {
	page, limit = normalizePageLimit(page, limit)

	var total int64
	if err := r.db.Model(&models.CollectionHealthSnapshotRun{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var runs []models.CollectionHealthSnapshotRun
	offset := (page - 1) * limit
	err := r.db.Order("started_at DESC").Offset(offset).Limit(limit).Find(&runs).Error
	return runs, total, err
}

// PruneOldRuns keeps only the most recent `keep` runs, deleting older runs.
func (r *CollectionHealthSnapshotRunRepository) PruneOldRuns(keep int) {
	var count int64
	r.db.Model(&models.CollectionHealthSnapshotRun{}).Count(&count)
	if count <= int64(keep) {
		return
	}

	var cutoffRun models.CollectionHealthSnapshotRun
	if err := r.db.Order("started_at DESC").Offset(keep).Limit(1).First(&cutoffRun).Error; err != nil {
		return
	}

	r.db.Where("started_at <= ?", cutoffRun.StartedAt).Delete(&models.CollectionHealthSnapshotRun{})
}
