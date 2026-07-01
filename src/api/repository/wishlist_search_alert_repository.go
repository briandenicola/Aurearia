package repository

import (
	"fmt"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"gorm.io/gorm"
)

type WishlistSearchAlertFilters struct {
	Active *bool
	Page   int
	Limit  int
}

type AlertCandidateFilters struct {
	State            string
	ProvenanceStatus string
	Page             int
	Limit            int
}

type WishlistSearchAlertRepository struct {
	db *gorm.DB
}

type TransactionalCoinCreator interface {
	CreateCoinInTransaction(tx *gorm.DB) (uint, error)
}

func NewWishlistSearchAlertRepository(db *gorm.DB) *WishlistSearchAlertRepository {
	return &WishlistSearchAlertRepository{db: db}
}

func (r *WishlistSearchAlertRepository) DB() *gorm.DB {
	return r.db
}

func (r *WishlistSearchAlertRepository) WithTx(tx *gorm.DB) *WishlistSearchAlertRepository {
	return &WishlistSearchAlertRepository{db: tx}
}

func normalizePageLimit(page, limit int) (int, int) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return page, limit
}

func (r *WishlistSearchAlertRepository) CreateAlert(alert *models.WishlistSearchAlert) error {
	return r.db.Select("*").Create(alert).Error
}

func (r *WishlistSearchAlertRepository) ListAlerts(userID uint, filters WishlistSearchAlertFilters) ([]models.WishlistSearchAlert, int64, error) {
	page, limit := normalizePageLimit(filters.Page, filters.Limit)
	query := r.db.Model(&models.WishlistSearchAlert{}).Scopes(OwnedBy(userID)).Where("deleted_at IS NULL")
	if filters.Active != nil {
		query = query.Where("is_active = ?", *filters.Active)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var alerts []models.WishlistSearchAlert
	err := query.Order("updated_at DESC").Offset((page - 1) * limit).Limit(limit).Find(&alerts).Error
	return alerts, total, err
}

func (r *WishlistSearchAlertRepository) GetAlert(id, userID uint) (*models.WishlistSearchAlert, error) {
	var alert models.WishlistSearchAlert
	err := r.db.Scopes(OwnedByID(id, userID)).Where("deleted_at IS NULL").First(&alert).Error
	if err != nil {
		return nil, err
	}
	return &alert, nil
}

func (r *WishlistSearchAlertRepository) UpdateAlert(alert *models.WishlistSearchAlert) error {
	return r.db.Save(alert).Error
}

func (r *WishlistSearchAlertRepository) DeleteAlert(id, userID uint) error {
	now := time.Now()
	return r.db.Model(&models.WishlistSearchAlert{}).
		Scopes(OwnedByID(id, userID)).
		Where("deleted_at IS NULL").
		Update("deleted_at", now).Error
}

func (r *WishlistSearchAlertRepository) CreateRun(run *models.AlertRun) error {
	return r.db.Create(run).Error
}

func (r *WishlistSearchAlertRepository) CreateManualRunIfAvailable(run *models.AlertRun, runningSince time.Time) (bool, error) {
	acquired := false
	err := r.db.Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Model(&models.AlertRun{}).
			Where("alert_id = ? AND user_id = ? AND status IN ? AND started_at >= ?", run.AlertID, run.UserID, []models.AlertRunStatus{models.AlertRunStatusQueued, models.AlertRunStatusRunning}, runningSince).
			Count(&count).Error; err != nil {
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

func (r *WishlistSearchAlertRepository) ClaimQueuedRun(runID uint) (*models.AlertRun, bool, error) {
	now := time.Now()
	var run models.AlertRun
	claimed := false
	err := r.db.Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&models.AlertRun{}).
			Where("id = ? AND status = ?", runID, models.AlertRunStatusQueued).
			Updates(map[string]interface{}{
				"status":            models.AlertRunStatusRunning,
				"started_at":        now,
				"completed_at":      nil,
				"duration_ms":       0,
				"error_message":     "",
				"rate_limit_status": "ok",
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

func (r *WishlistSearchAlertRepository) UpdateRun(run *models.AlertRun) error {
	return r.db.Save(run).Error
}

func (r *WishlistSearchAlertRepository) RecoverStaleAlertRuns(timeout time.Duration) ([]uint, error) {
	cutoff := time.Now().Add(-timeout)
	if err := r.db.Model(&models.AlertRun{}).
		Where("status = ? AND started_at < ?", models.AlertRunStatusRunning, cutoff).
		Updates(map[string]interface{}{
			"status":            models.AlertRunStatusQueued,
			"completed_at":      nil,
			"duration_ms":       0,
			"error_message":     "",
			"rate_limit_status": "ok",
		}).Error; err != nil {
		return nil, err
	}
	var ids []uint
	err := r.db.Model(&models.AlertRun{}).
		Where("status = ?", models.AlertRunStatusQueued).
		Order("created_at ASC").
		Pluck("id", &ids).Error
	return ids, err
}

func (r *WishlistSearchAlertRepository) ListRuns(alertID, userID uint, page, limit int) ([]models.AlertRun, int64, error) {
	page, limit = normalizePageLimit(page, limit)
	query := r.db.Model(&models.AlertRun{}).Where("alert_id = ? AND user_id = ?", alertID, userID)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var runs []models.AlertRun
	err := query.Order("started_at DESC").Offset((page - 1) * limit).Limit(limit).Find(&runs).Error
	return runs, total, err
}

func (r *WishlistSearchAlertRepository) GetRun(alertID, runID, userID uint) (*models.AlertRun, error) {
	var run models.AlertRun
	err := r.db.Preload("Candidates.Provenance").Where("id = ? AND alert_id = ? AND user_id = ?", runID, alertID, userID).First(&run).Error
	if err != nil {
		return nil, err
	}
	return &run, nil
}

func (r *WishlistSearchAlertRepository) FindCandidateByDuplicateKey(userID uint, duplicateKey string) (*models.AlertCandidate, error) {
	var candidate models.AlertCandidate
	err := r.db.Preload("Provenance").Where("user_id = ? AND duplicate_key = ?", userID, duplicateKey).First(&candidate).Error
	if err != nil {
		return nil, err
	}
	return &candidate, nil
}

func (r *WishlistSearchAlertRepository) FindCandidateByCanonicalURL(userID, alertID uint, canonicalURL string) (*models.AlertCandidate, error) {
	if canonicalURL == "" {
		return nil, gorm.ErrRecordNotFound
	}
	var candidate models.AlertCandidate
	err := r.db.Preload("Provenance").
		Where("user_id = ? AND alert_id = ? AND canonical_source_url = ?", userID, alertID, canonicalURL).
		Order("last_seen_at DESC").
		First(&candidate).Error
	if err != nil {
		return nil, err
	}
	return &candidate, nil
}

func (r *WishlistSearchAlertRepository) CreateCandidate(candidate *models.AlertCandidate, provenance []models.CandidateProvenance) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(candidate).Error; err != nil {
			return err
		}
		for i := range provenance {
			provenance[i].CandidateID = candidate.ID
		}
		if len(provenance) > 0 {
			if err := tx.Create(&provenance).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *WishlistSearchAlertRepository) UpdateCandidate(candidate *models.AlertCandidate) error {
	return r.db.Save(candidate).Error
}

func (r *WishlistSearchAlertRepository) ReplaceCandidateProvenance(candidateID uint, provenance []models.CandidateProvenance) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("candidate_id = ?", candidateID).Delete(&models.CandidateProvenance{}).Error; err != nil {
			return err
		}
		for i := range provenance {
			provenance[i].CandidateID = candidateID
		}
		if len(provenance) > 0 {
			return tx.Create(&provenance).Error
		}
		return nil
	})
}

func (r *WishlistSearchAlertRepository) GetCandidate(alertID, candidateID, userID uint) (*models.AlertCandidate, error) {
	var candidate models.AlertCandidate
	err := r.db.Preload("Provenance").
		Where("id = ? AND alert_id = ? AND user_id = ?", candidateID, alertID, userID).
		First(&candidate).Error
	if err != nil {
		return nil, err
	}
	return &candidate, nil
}

func (r *WishlistSearchAlertRepository) ListCandidates(alertID, userID uint, filters AlertCandidateFilters) ([]models.AlertCandidate, int64, error) {
	page, limit := normalizePageLimit(filters.Page, filters.Limit)
	query := r.db.Model(&models.AlertCandidate{}).Where("alert_id = ? AND user_id = ?", alertID, userID)
	if filters.State != "" {
		query = query.Where("lifecycle_state = ?", filters.State)
	} else {
		query = query.Where("lifecycle_state IN ?", []string{string(models.AlertCandidateStateActive), string(models.AlertCandidateStateNeedsReview)})
	}
	if filters.ProvenanceStatus != "" {
		query = query.Where("provenance_status = ?", filters.ProvenanceStatus)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var candidates []models.AlertCandidate
	err := query.Preload("Provenance").Order("last_seen_at DESC").Offset((page - 1) * limit).Limit(limit).Find(&candidates).Error
	return candidates, total, err
}

func (r *WishlistSearchAlertRepository) CreateReviewAction(action *models.CandidateReviewAction) error {
	return r.db.Create(action).Error
}

func (r *WishlistSearchAlertRepository) UpdateCandidateWithReviewAction(candidate *models.AlertCandidate, action *models.CandidateReviewAction) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(candidate).Error; err != nil {
			return err
		}
		if action != nil {
			return tx.Create(action).Error
		}
		return nil
	})
}

func (r *WishlistSearchAlertRepository) ConvertCandidateToWishlist(
	candidate *models.AlertCandidate,
	coinCreator TransactionalCoinCreator,
	action *models.CandidateReviewAction,
) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		coinID, err := coinCreator.CreateCoinInTransaction(tx)
		if err != nil {
			return err
		}
		candidate.LifecycleState = models.AlertCandidateStateConverted
		candidate.ConvertedCoinID = &coinID
		candidate.MatchingWishlistCoinID = &coinID
		if err := tx.Save(candidate).Error; err != nil {
			return err
		}
		if action != nil {
			action.Metadata = mergeActionMetadata(action.Metadata, map[string]interface{}{"coinId": coinID})
			if err := tx.Create(action).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *WishlistSearchAlertRepository) FindWishlistByReferenceURL(userID uint, referenceURL string) (*models.Coin, error) {
	var coin models.Coin
	err := r.db.Scopes(OwnedBy(userID)).
		Where("is_wishlist = ? AND reference_url = ?", true, referenceURL).
		First(&coin).Error
	if err != nil {
		return nil, err
	}
	return &coin, nil
}

func (r *WishlistSearchAlertRepository) FindWishlistBySourceAlertCandidateID(userID uint, candidateID uint) (*models.Coin, error) {
	var coin models.Coin
	err := r.db.Scopes(OwnedBy(userID)).
		Where("is_wishlist = ? AND source_alert_candidate_id = ?", true, candidateID).
		First(&coin).Error
	if err != nil {
		return nil, err
	}
	return &coin, nil
}

func mergeActionMetadata(existing string, values map[string]interface{}) string {
	if existing != "" {
		return existing
	}
	if len(values) == 0 {
		return ""
	}
	// Keep repository free of business decisions; metadata is best-effort audit text.
	out := "{"
	first := true
	for key, value := range values {
		if !first {
			out += ","
		}
		first = false
		out += `"` + key + `":"` + fmt.Sprint(value) + `"`
	}
	out += "}"
	return out
}
