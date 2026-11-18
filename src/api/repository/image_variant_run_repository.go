package repository

import (
	"github.com/briandenicola/ancient-coins-api/models"
	"gorm.io/gorm"
)

// imageVariantRunsKept is how many runs the history keeps.
const imageVariantRunsKept = 100

// ImageVariantRunRepository stores the image variant check's run history.
type ImageVariantRunRepository struct {
	db *gorm.DB
}

// NewImageVariantRunRepository creates a new ImageVariantRunRepository.
func NewImageVariantRunRepository(db *gorm.DB) *ImageVariantRunRepository {
	return &ImageVariantRunRepository{db: db}
}

// CreateRun inserts a run as it starts.
func (r *ImageVariantRunRepository) CreateRun(run *models.ImageVariantRun) error {
	return r.db.Create(run).Error
}

// CompleteRun saves a run's outcome and keeps only the newest runs.
func (r *ImageVariantRunRepository) CompleteRun(run *models.ImageVariantRun) error {
	err := r.db.Model(run).Updates(map[string]interface{}{
		"status":        run.Status,
		"checked":       run.Checked,
		"made":          run.Made,
		"failed":        run.Failed,
		"duration_ms":   run.DurationMs,
		"completed_at":  run.CompletedAt,
		"failures":      run.Failures,
		"error_message": run.ErrorMessage,
	}).Error
	if err == nil {
		r.pruneOldRuns(imageVariantRunsKept)
	}
	return err
}

// MarkInterrupted marks runs still recorded as running as stopped. A run is
// left running only when the API exited during a pass without finishing it.
func (r *ImageVariantRunRepository) MarkInterrupted() error {
	return r.db.Model(&models.ImageVariantRun{}).
		Where("status = ?", "running").
		Updates(map[string]interface{}{
			"status":        "stopped",
			"error_message": "The API stopped during this pass. The next pass carries on.",
		}).Error
}

// ListRuns returns runs newest first.
func (r *ImageVariantRunRepository) ListRuns(page, limit int) ([]models.ImageVariantRun, int64, error) {
	page, limit = normalizePageLimit(page, limit)
	var total int64
	if err := r.db.Model(&models.ImageVariantRun{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var runs []models.ImageVariantRun
	err := r.db.Order("started_at DESC, id DESC").Offset((page - 1) * limit).Limit(limit).Find(&runs).Error
	return runs, total, err
}

func (r *ImageVariantRunRepository) pruneOldRuns(keep int) {
	var cutoff models.ImageVariantRun
	if err := r.db.Order("started_at DESC, id DESC").Offset(keep).Limit(1).First(&cutoff).Error; err != nil {
		return
	}
	r.db.Where("id <= ?", cutoff.ID).Delete(&models.ImageVariantRun{})
}
