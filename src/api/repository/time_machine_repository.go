package repository

import (
	"errors"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"gorm.io/gorm"
)

// TimeMachineRepository reconstructs the state of a collection as it stood on a
// past date, from data the app already records: purchase/sold dates on coins and
// the per-coin valuation history written by the valuation scheduler.
type TimeMachineRepository struct {
	db *gorm.DB
}

func NewTimeMachineRepository(db *gorm.DB) *TimeMachineRepository {
	return &TimeMachineRepository{db: db}
}

// TimeMachineCoin is one coin as it stood on the requested date.
type TimeMachineCoin struct {
	ID            uint
	Name          string
	Category      string
	Material      string
	Era           string
	Ruler         string
	PurchasePrice *float64
	// ValueAsOf is the most recent recorded valuation at or before the
	// requested date, falling back to the purchase price.
	ValueAsOf float64
	// ValueFromHistory distinguishes a real historical valuation from the
	// purchase-price fallback, so the UI can be honest about which it is.
	ValueFromHistory bool
	PurchaseDate     *time.Time
}

// CollectionBounds describes the range of dates the time machine can address.
type CollectionBounds struct {
	EarliestPurchase *time.Time
	LatestPurchase   *time.Time
}

// GetCollectionAsOf returns every coin the user owned on asOf, valued as of that
// date.
//
// "Owned on date D" means: not a wishlist entry, acquired on or before D, and
// either never sold or sold after D. Coins with no purchase date cannot be
// placed on a timeline and are excluded — the caller reports how many were
// skipped so the UI can say so rather than silently under-counting.
//
// Per-coin value resolves to the newest coin_value_history row recorded at or
// before D. Valuations are only written when the valuation scheduler runs, so
// early dates commonly fall back to purchase price; ValueFromHistory records
// which basis was used for each coin.
func (r *TimeMachineRepository) GetCollectionAsOf(userID uint, asOf time.Time) ([]TimeMachineCoin, error) {
	var rows []struct {
		ID               uint
		Name             string
		Category         string
		Material         string
		Era              string
		Ruler            string
		PurchasePrice    *float64
		PurchaseDate     *time.Time
		HistoricalValue  *float64
		ValueAsOf        float64
		ValueFromHistory bool
	}

	err := r.db.Raw(`
		SELECT
			owned.id                AS id,
			owned.name              AS name,
			owned.category          AS category,
			owned.material          AS material,
			owned.era               AS era,
			owned.ruler             AS ruler,
			owned.purchase_price    AS purchase_price,
			owned.purchase_date     AS purchase_date,
			owned.historical_value  AS historical_value,
			COALESCE(owned.historical_value, owned.purchase_price, 0) AS value_as_of,
			CASE WHEN owned.historical_value IS NOT NULL THEN 1 ELSE 0 END AS value_from_history
		FROM (
			SELECT
				coins.id, coins.name, coins.category, coins.material, coins.era,
				coins.ruler, coins.purchase_price, coins.purchase_date,
				(
					SELECT coin_value_histories.value
					FROM coin_value_histories
					WHERE coin_value_histories.coin_id = coins.id
					  AND coin_value_histories.recorded_at <= ?
					ORDER BY coin_value_histories.recorded_at DESC
					LIMIT 1
				) AS historical_value
			FROM coins
			WHERE coins.user_id = ?
			  AND coins.is_wishlist = 0
			  AND coins.purchase_date IS NOT NULL
			  AND coins.purchase_date <= ?
			  AND (coins.is_sold = 0 OR coins.sold_date IS NULL OR coins.sold_date > ?)
		) AS owned
		ORDER BY value_as_of DESC, owned.id ASC
	`, asOf, userID, asOf, asOf).Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	coins := make([]TimeMachineCoin, 0, len(rows))
	for _, row := range rows {
		coins = append(coins, TimeMachineCoin{
			ID:               row.ID,
			Name:             row.Name,
			Category:         row.Category,
			Material:         row.Material,
			Era:              row.Era,
			Ruler:            row.Ruler,
			PurchasePrice:    row.PurchasePrice,
			ValueAsOf:        row.ValueAsOf,
			ValueFromHistory: row.ValueFromHistory,
			PurchaseDate:     row.PurchaseDate,
		})
	}
	return coins, nil
}

// CountUndatedCoins returns how many non-wishlist coins have no purchase date
// and are therefore invisible to the time machine at any date.
func (r *TimeMachineRepository) CountUndatedCoins(userID uint) (int64, error) {
	var count int64
	err := r.db.Raw(`
		SELECT COUNT(*) FROM coins
		WHERE user_id = ? AND is_wishlist = 0 AND purchase_date IS NULL
	`, userID).Scan(&count).Error
	return count, err
}

// GetBounds returns the earliest and latest purchase dates on record, which
// define the range the timeline scrubber can span.
func (r *TimeMachineRepository) GetBounds(userID uint) (CollectionBounds, error) {
	// Selecting the column itself rather than MIN()/MAX() keeps the driver's
	// datetime typing: SQLite returns an aggregate over a datetime column as an
	// untyped string, which will not scan into *time.Time.
	edge := func(order string) (*time.Time, error) {
		var coin models.Coin
		err := r.db.Select("purchase_date").
			Where("user_id = ? AND is_wishlist = 0 AND purchase_date IS NOT NULL", userID).
			Order("purchase_date " + order).
			Limit(1).
			Take(&coin).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		if err != nil {
			return nil, err
		}
		return coin.PurchaseDate, nil
	}

	earliest, err := edge("ASC")
	if err != nil {
		return CollectionBounds{}, err
	}
	latest, err := edge("DESC")
	if err != nil {
		return CollectionBounds{}, err
	}
	return CollectionBounds{EarliestPurchase: earliest, LatestPurchase: latest}, nil
}

// GetHealthScoreAsOf returns the collection health score from the most recent
// daily snapshot at or before asOf. Returns nil when no snapshot predates the
// requested date — health snapshots only began when the feature shipped, so
// early dates legitimately have none.
func (r *TimeMachineRepository) GetHealthScoreAsOf(userID uint, asOf time.Time) (*int, error) {
	var row struct{ Score *int }
	err := r.db.Raw(`
		SELECT score FROM collection_health_snapshots
		WHERE user_id = ? AND snapshot_date <= ?
		ORDER BY snapshot_date DESC
		LIMIT 1
	`, userID, asOf).Scan(&row).Error
	if err != nil {
		return nil, err
	}
	return row.Score, nil
}
