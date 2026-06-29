package repository

import (
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"gorm.io/gorm"
)

type AIJobRepository struct {
	db *gorm.DB
}

func NewAIJobRepository(db *gorm.DB) *AIJobRepository {
	return &AIJobRepository{db: db}
}

func (r *AIJobRepository) FindCoinWithImages(coinID, userID uint) (*models.Coin, error) {
	var coin models.Coin
	err := r.db.Scopes(OwnedByID(coinID, userID)).
		Preload("Images").
		Preload("StorageLocation").
		First(&coin).Error
	if err != nil {
		return nil, err
	}
	return &coin, nil
}

func (r *AIJobRepository) EnqueueOrFindActive(userID, coinID uint, jobType models.AIJobType, side string) (*models.AIJob, bool, error) {
	var job models.AIJob
	var created bool
	err := r.db.Transaction(func(tx *gorm.DB) error {
		err := tx.Where("user_id = ? AND coin_id = ? AND job_type = ? AND side = ? AND status IN ?",
			userID, coinID, jobType, side, []models.AIJobStatus{models.AIJobStatusQueued, models.AIJobStatusRunning}).
			Order("created_at ASC").
			First(&job).Error
		if err == nil {
			return nil
		}
		if err != gorm.ErrRecordNotFound {
			return err
		}
		job = models.AIJob{
			UserID:  userID,
			CoinID:  coinID,
			JobType: jobType,
			Side:    side,
			Status:  models.AIJobStatusQueued,
		}
		created = true
		return tx.Create(&job).Error
	})
	return &job, created, err
}

func (r *AIJobRepository) GetByIDForUser(jobID, userID uint) (*models.AIJob, error) {
	var job models.AIJob
	if err := r.db.Scopes(OwnedByID(jobID, userID)).First(&job).Error; err != nil {
		return nil, err
	}
	return &job, nil
}

func (r *AIJobRepository) ListForCoin(userID, coinID uint, activeOnly bool) ([]models.AIJob, error) {
	query := r.db.Scopes(OwnedBy(userID)).Where("coin_id = ?", coinID)
	if activeOnly {
		query = query.Where("status IN ?", []models.AIJobStatus{models.AIJobStatusQueued, models.AIJobStatusRunning})
	}
	var jobs []models.AIJob
	err := query.Order("created_at DESC").Find(&jobs).Error
	return jobs, err
}

func (r *AIJobRepository) ClaimQueued(jobID uint) (*models.AIJob, bool, error) {
	now := time.Now()
	var job models.AIJob
	var claimed bool
	err := r.db.Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&models.AIJob{}).
			Where("id = ? AND status = ?", jobID, models.AIJobStatusQueued).
			Updates(map[string]interface{}{
				"status":     models.AIJobStatusRunning,
				"started_at": now,
				"attempts":   gorm.Expr("attempts + 1"),
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return nil
		}
		claimed = true
		return tx.First(&job, jobID).Error
	})
	return &job, claimed, err
}

func (r *AIJobRepository) Complete(jobID uint, result string) error {
	now := time.Now()
	return r.db.Model(&models.AIJob{}).
		Where("id = ?", jobID).
		Updates(map[string]interface{}{
			"status":        models.AIJobStatusCompleted,
			"result":        result,
			"error_message": "",
			"completed_at":  now,
		}).Error
}

func (r *AIJobRepository) Fail(jobID uint, message string) error {
	now := time.Now()
	return r.db.Model(&models.AIJob{}).
		Where("id = ?", jobID).
		Updates(map[string]interface{}{
			"status":        models.AIJobStatusFailed,
			"error_message": message,
			"completed_at":  now,
		}).Error
}

func (r *AIJobRepository) UpdateCoinAnalysis(coinID, userID uint, column, analysis string) error {
	result := r.db.Model(&models.Coin{}).
		Scopes(OwnedByID(coinID, userID)).
		Update(column, analysis)
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return result.Error
}

func (r *AIJobRepository) CreateJournalEntry(entry *models.CoinJournal) error {
	return r.db.Create(entry).Error
}

func (r *AIJobRepository) RecoverStaleJobs(timeout time.Duration) ([]uint, error) {
	cutoff := time.Now().Add(-timeout)
	if err := r.db.Model(&models.AIJob{}).
		Where("status = ? AND started_at < ?", models.AIJobStatusRunning, cutoff).
		Updates(map[string]interface{}{
			"status":        models.AIJobStatusQueued,
			"started_at":    nil,
			"error_message": "",
		}).Error; err != nil {
		return nil, err
	}
	var ids []uint
	err := r.db.Model(&models.AIJob{}).
		Where("status = ?", models.AIJobStatusQueued).
		Order("created_at ASC").
		Pluck("id", &ids).Error
	return ids, err
}
