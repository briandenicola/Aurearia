package repository

import (
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"gorm.io/gorm"
)

// FeaturedCoinRepository encapsulates all FeaturedCoin DB operations.
type FeaturedCoinRepository struct {
	db *gorm.DB
}

// NewFeaturedCoinRepository creates a new FeaturedCoinRepository.
func NewFeaturedCoinRepository(db *gorm.DB) *FeaturedCoinRepository {
	return &FeaturedCoinRepository{db: db}
}

// Create inserts a new featured-coin record.
func (r *FeaturedCoinRepository) Create(fc *models.FeaturedCoin) error {
	return r.db.Create(fc).Error
}

// FindByID returns a single featured-coin record by ID (does not enforce ownership).
func (r *FeaturedCoinRepository) FindByID(id uint) (*models.FeaturedCoin, error) {
	var fc models.FeaturedCoin
	err := r.db.Preload("Coin.Images").First(&fc, id).Error
	if err != nil {
		return nil, err
	}
	return &fc, nil
}

// FindByIDForUser returns a single featured-coin record by ID, scoped to the user.
func (r *FeaturedCoinRepository) FindByIDForUser(id, userID uint) (*models.FeaturedCoin, error) {
	var fc models.FeaturedCoin
	err := r.db.Scopes(OwnedByID(id, userID)).Preload("Coin.Images").First(&fc).Error
	if err != nil {
		return nil, err
	}
	return &fc, nil
}

// GetLatestForUser returns the most recent featured-coin record for a user.
func (r *FeaturedCoinRepository) GetLatestForUser(userID uint) (*models.FeaturedCoin, error) {
	var fc models.FeaturedCoin
	err := r.db.Scopes(OwnedBy(userID)).
		Order("featured_at DESC").
		Preload("Coin.Images").
		First(&fc).Error
	if err != nil {
		return nil, err
	}
	return &fc, nil
}

// HasBeenFeaturedToday returns true if the user already has a feature record
// dated today (UTC date match).
func (r *FeaturedCoinRepository) HasBeenFeaturedToday(userID uint, today time.Time) (bool, error) {
	startOfDay := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	var count int64
	err := r.db.Model(&models.FeaturedCoin{}).
		Scopes(OwnedBy(userID)).
		Where("featured_at >= ? AND featured_at < ?", startOfDay, endOfDay).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// PickNextCoinID returns the next coin ID to feature for the user, enforcing
// a cycle: prefer coins never featured; otherwise pick the coin whose most
// recent feature is oldest. Returns 0 if user has no eligible coins.
//
// Eligible pool: owned coins that are NOT wishlist and NOT sold.
func (r *FeaturedCoinRepository) PickNextCoinID(userID uint) (uint, error) {
	type row struct {
		ID uint
	}

	var rows []row
	// Left join: coins joined to their most recent featured_coin record.
	// Sort coins never-shown first (LastShown IS NULL DESC), then by oldest LastShown.
	err := r.db.Raw(`
		SELECT c.id AS id,
			MAX(fc.featured_at) AS last_shown
		FROM coins c
		LEFT JOIN featured_coins fc
			ON fc.coin_id = c.id AND fc.user_id = c.user_id
		WHERE c.user_id = ?
			AND c.is_wishlist = 0
			AND c.is_sold = 0
		GROUP BY c.id
		ORDER BY (last_shown IS NULL) DESC, last_shown ASC, c.id ASC
		LIMIT 1
	`, userID).Scan(&rows).Error
	if err != nil {
		return 0, err
	}
	if len(rows) == 0 {
		return 0, nil
	}
	return rows[0].ID, nil
}
