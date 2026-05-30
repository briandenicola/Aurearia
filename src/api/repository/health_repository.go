package repository

import (
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"gorm.io/gorm"
)

// HealthRepository encapsulates health scorecard data access.
type HealthRepository struct {
	db *gorm.DB
}

// NewHealthRepository creates a new HealthRepository.
func NewHealthRepository(db *gorm.DB) *HealthRepository {
	return &HealthRepository{db: db}
}

// EligibleCoinRow is a compact row shape for score input queries.
type EligibleCoinRow struct {
	CoinID            uint
	Title             string
	Category          string
	Denomination      string
	Ruler             string
	Era               string
	Mint              string
	Material          string
	Grade             string
	ReferenceURL      string
	PurchaseDate      *time.Time
	CurrentValue      *float64
	AIAnalysis        string
	ObverseAnalysis   string
	ReverseAnalysis   string
	UpdatedAt         time.Time
	PrimaryImageCount int64
	ImageCount        int64
}

// SnapshotBaselineRow captures historical score context for trend comparisons.
type SnapshotBaselineRow struct {
	Score        int
	SnapshotDate time.Time
}

// ListEligibleCoins returns active collection coins for a user.
func (r *HealthRepository) ListEligibleCoins(userID uint) ([]EligibleCoinRow, error) {
	rows := []EligibleCoinRow{}
	err := r.db.Model(&models.Coin{}).
		Scopes(ActiveCollection(userID)).
		Select(
			"id AS coin_id, name AS title, category, denomination, ruler, era, mint, material, grade, reference_url, purchase_date, current_value, ai_analysis, obverse_analysis, reverse_analysis, updated_at, "+
				"(SELECT COUNT(*) FROM coin_images WHERE coin_id = coins.id AND is_primary = true) AS primary_image_count, "+
				"(SELECT COUNT(*) FROM coin_images WHERE coin_id = coins.id) AS image_count",
		).
		Order("updated_at DESC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}

// ListEligibleCoinsPaged returns active collection coins with optional scope filtering.
func (r *HealthRepository) ListEligibleCoinsPaged(userID uint, page, limit int, scope string) ([]EligibleCoinRow, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 25
	}

	query := r.db.Model(&models.Coin{}).Scopes(ActiveCollection(userID))

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	rows := []EligibleCoinRow{}
	offset := (page - 1) * limit

	selectClause := "id AS coin_id, name AS title, category, denomination, ruler, era, mint, material, grade, reference_url, purchase_date, current_value, ai_analysis, obverse_analysis, reverse_analysis, updated_at, " +
		"(SELECT COUNT(*) FROM coin_images WHERE coin_id = coins.id AND is_primary = true) AS primary_image_count, " +
		"(SELECT COUNT(*) FROM coin_images WHERE coin_id = coins.id) AS image_count"

	orderClause := "updated_at DESC, id ASC"
	if scope == "needs_attention" {
		orderClause = "updated_at ASC, id ASC"
	}

	err := query.
		Select(selectClause).
		Order(orderClause).
		Offset(offset).
		Limit(limit).
		Scan(&rows).Error
	if err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

// GetSnapshotBaseline returns a snapshot around the provided baseline date for a user.
func (r *HealthRepository) GetSnapshotBaseline(userID uint, baselineDate time.Time) (*SnapshotBaselineRow, error) {
	var row SnapshotBaselineRow
	err := r.db.Model(&models.CollectionHealthSnapshot{}).
		Select("score, snapshot_date").
		Where("user_id = ? AND snapshot_date <= ?", userID, baselineDate).
		Order("snapshot_date DESC").
		First(&row).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &row, nil
}

// UpsertCollectionHealthSnapshot inserts or updates a per-user daily snapshot.
func (r *HealthRepository) UpsertCollectionHealthSnapshot(snapshot *models.CollectionHealthSnapshot) error {
	var existing models.CollectionHealthSnapshot
	err := r.db.Where("user_id = ? AND snapshot_date = ?", snapshot.UserID, snapshot.SnapshotDate).First(&existing).Error

	if err == gorm.ErrRecordNotFound {
		return r.db.Create(snapshot).Error
	}
	if err != nil {
		return err
	}

	snapshot.ID = existing.ID
	snapshot.CreatedAt = existing.CreatedAt
	return r.db.Save(snapshot).Error
}

// ListUsersWithEligibleCoins returns users with at least one active collection coin.
func (r *HealthRepository) ListUsersWithEligibleCoins() ([]uint, error) {
	userIDs := []uint{}
	err := r.db.Model(&models.Coin{}).
		Where("is_wishlist = ? AND is_sold = ?", false, false).
		Distinct("user_id").
		Pluck("user_id", &userIDs).Error
	if err != nil {
		return nil, err
	}
	return userIDs, nil
}

// ListAllEligibleCoins returns all active collection coins across all users.
func (r *HealthRepository) ListAllEligibleCoins() ([]EligibleCoinRow, error) {
	rows := []EligibleCoinRow{}
	err := r.db.Model(&models.Coin{}).
		Where("is_wishlist = ? AND is_sold = ?", false, false).
		Select(
			"id AS coin_id, name AS title, category, denomination, ruler, era, mint, material, grade, reference_url, purchase_date, current_value, ai_analysis, obverse_analysis, reverse_analysis, updated_at, "+
				"(SELECT COUNT(*) FROM coin_images WHERE coin_id = coins.id AND is_primary = true) AS primary_image_count, "+
				"(SELECT COUNT(*) FROM coin_images WHERE coin_id = coins.id) AS image_count",
		).
		Order("updated_at DESC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}

