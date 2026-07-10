package repository

import (
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"gorm.io/gorm"
)

// CoinOfDayRunRepository encapsulates Coin of the Day run history persistence.
type CoinOfDayRunRepository struct {
	db *gorm.DB
}

func NewCoinOfDayRunRepository(db *gorm.DB) *CoinOfDayRunRepository {
	return &CoinOfDayRunRepository{db: db}
}

func (r *CoinOfDayRunRepository) CreateRunIfNoActive(run *models.CoinOfDayRun, runningSince time.Time) (*models.CoinOfDayRun, bool, error) {
	var existing models.CoinOfDayRun
	acquired := false
	err := r.db.Transaction(func(tx *gorm.DB) error {
		err := tx.Where("status IN ? AND started_at >= ?", []models.CoinOfDayRunStatus{models.CoinOfDayRunStatusQueued, models.CoinOfDayRunStatusRunning}, runningSince).
			Order("started_at DESC").
			First(&existing).Error
		if err == nil {
			return nil
		}
		if err != nil && err != gorm.ErrRecordNotFound {
			return err
		}
		if err := tx.Create(run).Error; err != nil {
			return err
		}
		existing = *run
		acquired = true
		return nil
	})
	return &existing, acquired, err
}

func (r *CoinOfDayRunRepository) ClaimQueuedRun(runID uint) (*models.CoinOfDayRun, bool, error) {
	now := time.Now()
	var run models.CoinOfDayRun
	claimed := false
	err := r.db.Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&models.CoinOfDayRun{}).
			Where("id = ? AND status = ?", runID, models.CoinOfDayRunStatusQueued).
			Updates(map[string]interface{}{
				"status":        models.CoinOfDayRunStatusRunning,
				"started_at":    now,
				"completed_at":  nil,
				"picked":        0,
				"skipped":       0,
				"errors":        0,
				"error_message": "",
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

func (r *CoinOfDayRunRepository) UpdateRun(run *models.CoinOfDayRun) error {
	return r.db.Save(run).Error
}

func (r *CoinOfDayRunRepository) RecoverStaleRuns(timeout time.Duration) ([]uint, error) {
	cutoff := time.Now().Add(-timeout)
	if err := r.db.Model(&models.CoinOfDayRun{}).
		Where("status = ? AND started_at < ?", models.CoinOfDayRunStatusRunning, cutoff).
		Updates(map[string]interface{}{
			"status":        models.CoinOfDayRunStatusQueued,
			"completed_at":  nil,
			"picked":        0,
			"skipped":       0,
			"errors":        0,
			"error_message": "",
		}).Error; err != nil {
		return nil, err
	}
	var ids []uint
	err := r.db.Model(&models.CoinOfDayRun{}).
		Where("status = ?", models.CoinOfDayRunStatusQueued).
		Order("created_at ASC").
		Pluck("id", &ids).Error
	return ids, err
}

func (r *CoinOfDayRunRepository) ListRuns(page, limit int) ([]models.CoinOfDayRun, int64, error) {
	page, limit = normalizePageLimit(page, limit)
	query := r.db.Model(&models.CoinOfDayRun{})
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var runs []models.CoinOfDayRun
	err := query.Order("started_at DESC").Offset((page - 1) * limit).Limit(limit).Find(&runs).Error
	return runs, total, err
}

func (r *CoinOfDayRunRepository) GetRun(id uint) (*models.CoinOfDayRun, error) {
	var run models.CoinOfDayRun
	if err := r.db.First(&run, id).Error; err != nil {
		return nil, err
	}
	return &run, nil
}
