package repository

import (
	"github.com/briandenicola/ancient-coins-api/models"
	"gorm.io/gorm"
)

// AuctionWatchBidDigestRepository encapsulates watch-bid-digest run persistence.
type AuctionWatchBidDigestRepository struct {
	db *gorm.DB
}

func NewAuctionWatchBidDigestRepository(db *gorm.DB) *AuctionWatchBidDigestRepository {
	return &AuctionWatchBidDigestRepository{db: db}
}

func (r *AuctionWatchBidDigestRepository) CreateRun(run *models.AuctionWatchBidDigestRun) error {
	return r.db.Create(run).Error
}

func (r *AuctionWatchBidDigestRepository) CompleteRun(run *models.AuctionWatchBidDigestRun) error {
	err := r.db.Model(run).Updates(map[string]interface{}{
		"status":        run.Status,
		"lots_checked":  run.LotsChecked,
		"digests_sent":  run.DigestsSent,
		"duration_ms":   run.DurationMs,
		"completed_at":  run.CompletedAt,
		"error_message": run.ErrorMessage,
	}).Error
	if err == nil {
		r.PruneOldRuns(100)
	}
	return err
}

func (r *AuctionWatchBidDigestRepository) ListRuns(page, limit int) ([]models.AuctionWatchBidDigestRun, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	var total int64
	if err := r.db.Model(&models.AuctionWatchBidDigestRun{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var runs []models.AuctionWatchBidDigestRun
	offset := (page - 1) * limit
	err := r.db.Order("started_at DESC").Offset(offset).Limit(limit).Find(&runs).Error
	return runs, total, err
}

func (r *AuctionWatchBidDigestRepository) GetLastScheduledRun() *models.AuctionWatchBidDigestRun {
	var run models.AuctionWatchBidDigestRun
	err := r.db.Where("trigger_type = ? AND completed_at IS NOT NULL", "scheduled").
		Order("started_at DESC").Limit(1).First(&run).Error
	if err != nil {
		return nil
	}
	return &run
}

func (r *AuctionWatchBidDigestRepository) PruneOldRuns(keep int) {
	var count int64
	r.db.Model(&models.AuctionWatchBidDigestRun{}).Count(&count)
	if count <= int64(keep) {
		return
	}

	var cutoffRun models.AuctionWatchBidDigestRun
	if err := r.db.Order("started_at DESC").Offset(keep).Limit(1).First(&cutoffRun).Error; err != nil {
		return
	}

	r.db.Where("started_at <= ?", cutoffRun.StartedAt).Delete(&models.AuctionWatchBidDigestRun{})
}
