package repository

import (
	"fmt"
	"strings"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// CoinListFilters holds optional filters for listing coins.
type CoinListFilters struct {
	Category  string
	Era       string
	Search    string
	Wishlist  *bool
	Sold      *bool
	TagID     *uint
	SortField string
	SortOrder string
	Seed      *int // for SortField == "random"; deterministic per-seed shuffle
	Page      int
	Limit     int
}

// OwnedCoinFilters are lightweight scoped filters used by collection chat tools.
type OwnedCoinFilters struct {
	Category string
	Material string
	Era      string
	Ruler    string
	Wishlist *bool
	Sold     *bool
	Search   string
}

// CategoryCount holds a category name and its coin count.
type CategoryCount struct {
	Category string `json:"category"`
	Count    int64  `json:"count"`
}

// MaterialCount holds a material name and its coin count.
type MaterialCount struct {
	Material string `json:"material"`
	Count    int64  `json:"count"`
}

// GradeCount holds a grade and its coin count.
type GradeCount struct {
	Grade string `json:"grade"`
	Count int64  `json:"count"`
}

// EraCount holds an era and its coin count.
type EraCount struct {
	Era   string `json:"era"`
	Count int64  `json:"count"`
}

// RulerCount holds a ruler name and its coin count.
type RulerCount struct {
	Ruler string `json:"ruler"`
	Count int64  `json:"count"`
}

// PriceRange holds a price range label and its coin count.
type PriceRange struct {
	Range string `json:"range"`
	Count int64  `json:"count"`
}

// ValueSummary holds aggregate value metrics.
type ValueSummary struct {
	TotalPurchasePrice float64 `json:"totalPurchasePrice"`
	TotalCurrentValue  float64 `json:"totalCurrentValue"`
	AvgPurchasePrice   float64 `json:"avgPurchasePrice"`
	AvgCurrentValue    float64 `json:"avgCurrentValue"`
}

// SoldSummary holds aggregate sold coin metrics.
type SoldSummary struct {
	TotalSoldPrice    float64 `json:"totalSoldPrice"`
	TotalPurchaseCost float64 `json:"totalPurchaseCost"`
}

// CollectionStats holds all aggregate statistics for a user's collection.
type CollectionStats struct {
	TotalCoins    int64           `json:"totalCoins"`
	TotalWishlist int64           `json:"totalWishlist"`
	TotalSold     int64           `json:"totalSold"`
	ByCategory    []CategoryCount `json:"byCategory"`
	ByMaterial    []MaterialCount `json:"byMaterial"`
	ByGrade       []GradeCount    `json:"byGrade"`
	ByEra         []EraCount      `json:"byEra"`
	ByRuler       []RulerCount    `json:"byRuler"`
	ByPriceRange  []PriceRange    `json:"byPriceRange"`
	Values        ValueSummary    `json:"values"`
	SoldValues    SoldSummary     `json:"soldValues"`
}

// DistributionCell holds a single cell in the era × category heat map.
type DistributionCell struct {
	Era      string `json:"era"`
	Category string `json:"category"`
	Count    int    `json:"count"`
}

// CoinRepository encapsulates all coin-related database operations.
type CoinRepository struct {
	db *gorm.DB
}

// NewCoinRepository creates a new CoinRepository.
func NewCoinRepository(db *gorm.DB) *CoinRepository {
	return &CoinRepository{db: db}
}

// WithTx returns a shallow copy of the repository that uses tx for all queries.
func (r *CoinRepository) WithTx(tx *gorm.DB) *CoinRepository {
	return &CoinRepository{db: tx}
}

// DB exposes the underlying *gorm.DB so callers can create transactions.
func (r *CoinRepository) DB() *gorm.DB {
	return r.db
}

func applyOwnedFilterConditions(query *gorm.DB, filters OwnedCoinFilters) *gorm.DB {
	if filters.Category != "" {
		query = query.Where("category = ?", filters.Category)
	}
	if filters.Material != "" {
		query = query.Where("material = ?", filters.Material)
	}
	if filters.Era != "" {
		query = query.Where("era = ?", filters.Era)
	}
	if filters.Ruler != "" {
		query = query.Where("LOWER(ruler) LIKE LOWER(?)", "%"+filters.Ruler+"%")
	}
	if filters.Wishlist != nil {
		query = query.Where("is_wishlist = ?", *filters.Wishlist)
	}
	if filters.Sold != nil {
		query = query.Where("is_sold = ?", *filters.Sold)
	}
	if filters.Search != "" {
		term := "%" + strings.TrimSpace(filters.Search) + "%"
		query = query.Where(
			"name LIKE ? OR denomination LIKE ? OR ruler LIKE ? OR notes LIKE ?",
			term, term, term, term,
		)
	}
	return query
}

var allowedSortFields = map[string]string{
	"created_at":    "created_at",
	"updated_at":    "updated_at",
	"current_value": "current_value",
	"purchase_date": "purchase_date",
	"name":          "name",
}

var searchFields = []string{
	"name", "denomination", "ruler", "era", "mint",
	"obverse_inscription", "reverse_inscription", "notes", "rarity_rating",
}

// List returns a paginated, filtered list of coins for a user.
func (r *CoinRepository) List(userID uint, filters CoinListFilters) ([]models.Coin, int64, error) {
	query := r.db.Scopes(OwnedBy(userID)).Preload("Images").Preload("Tags").Preload("References").Preload("StorageLocation")

	if filters.Category != "" {
		query = query.Where("category = ?", filters.Category)
	}
	if filters.Era != "" {
		query = query.Where("era = ?", filters.Era)
	}
	if filters.Wishlist != nil {
		query = query.Where("is_wishlist = ?", *filters.Wishlist)
	}
	if filters.Sold != nil {
		query = query.Where("is_sold = ?", *filters.Sold)
	}
	if filters.TagID != nil {
		query = query.Where("id IN (SELECT coin_id FROM coin_tags WHERE tag_id = ?)", *filters.TagID)
	}
	if filters.Search != "" {
		term := "%" + filters.Search + "%"
		sub := r.db.Where("name LIKE ?", term)
		for _, f := range searchFields[1:] {
			sub = sub.Or(f+" LIKE ?", term)
		}
		query = query.Where(sub)
	}

	var total int64
	if err := query.Model(&models.Coin{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	page := filters.Page
	if page < 1 {
		page = 1
	}
	limit := filters.Limit
	if limit < 1 || limit > 100 {
		limit = 50
	}
	offset := (page - 1) * limit

	var coins []models.Coin

	// Seeded deterministic random sort: same seed produces the same order across pages.
	// Hash mixes id with the seed and Knuth's golden-ratio multiplier (2654435761) so
	// the modulo wraps for every row, producing a true permutation. Without the large
	// multiplier, `id*seed + seed` is monotonic in id and degenerates to insertion
	// order. abs() protects against any 64-bit overflow.
	// Seed is bound via %d after being validated by strconv.Atoi in the handler — safe
	// from SQL injection. Note: gorm.Expr placeholders inside Order() are silently
	// dropped by this GORM build, so the int is formatted directly into the SQL.
	if filters.SortField == "random" && filters.Seed != nil {
		seed := *filters.Seed
		orderExpr := fmt.Sprintf("abs((id * 2654435761) + (id * %d) + %d) %% 1000000", seed, seed)
		if err := query.Order(orderExpr).Offset(offset).Limit(limit).Find(&coins).Error; err != nil {
			return nil, 0, err
		}
		return coins, total, nil
	}

	col, ok := allowedSortFields[filters.SortField]
	if !ok {
		col = "updated_at"
	}
	order := filters.SortOrder
	if order != "asc" && order != "desc" {
		order = "desc"
	}

	if err := query.
		Order(clause.OrderByColumn{
			Column: clause.Column{Name: col},
			Desc:   order != "asc",
		}).
		Offset(offset).
		Limit(limit).
		Find(&coins).Error; err != nil {
		return nil, 0, err
	}
	return coins, total, nil
}

// FindByID returns a single coin owned by the user, with images preloaded.
func (r *CoinRepository) FindByID(id uint, userID uint) (*models.Coin, error) {
	var coin models.Coin
	err := r.db.Scopes(OwnedByID(id, userID)).Preload("Images").Preload("Tags").Preload("References").Preload("StorageLocation").First(&coin).Error
	if err != nil {
		return nil, err
	}
	return &coin, nil
}

// FindOwnedByNameCandidates returns up to limit owned coins with name matches.
func (r *CoinRepository) FindOwnedByNameCandidates(userID uint, name string, limit int) ([]models.Coin, error) {
	if limit < 1 {
		limit = 5
	}
	var coins []models.Coin
	err := r.db.Model(&models.Coin{}).
		Scopes(OwnedBy(userID)).
		Where("LOWER(name) LIKE LOWER(?)", "%"+strings.TrimSpace(name)+"%").
		Order("name ASC").
		Limit(limit).
		Find(&coins).Error
	return coins, err
}

// ListOwnedByFilters returns owner-scoped coins filtered for collection chat queries.
func (r *CoinRepository) ListOwnedByFilters(userID uint, filters OwnedCoinFilters, limit int) ([]models.Coin, error) {
	if limit < 1 {
		limit = 5
	}
	var coins []models.Coin
	query := r.db.Model(&models.Coin{}).Scopes(OwnedBy(userID))
	query = applyOwnedFilterConditions(query, filters)
	err := query.
		Order(clause.OrderByColumn{Column: clause.Column{Name: "updated_at"}, Desc: true}).
		Limit(limit).
		Find(&coins).Error
	return coins, err
}

// CountOwnedByFilters counts owner-scoped coins matching collection chat filters.
func (r *CoinRepository) CountOwnedByFilters(userID uint, filters OwnedCoinFilters) (int64, error) {
	var count int64
	query := r.db.Model(&models.Coin{}).Scopes(OwnedBy(userID))
	query = applyOwnedFilterConditions(query, filters)
	err := query.Count(&count).Error
	return count, err
}

// TopOwnedByCurrentValue returns the top N active collection coins by current value.
func (r *CoinRepository) TopOwnedByCurrentValue(userID uint, limit int) ([]models.Coin, error) {
	if limit < 1 {
		limit = 3
	}
	if limit > 10 {
		limit = 10
	}
	var coins []models.Coin
	err := r.db.Model(&models.Coin{}).
		Scopes(ActiveCollection(userID)).
		Where("current_value IS NOT NULL").
		Order(clause.OrderByColumn{Column: clause.Column{Name: "current_value"}, Desc: true}).
		Limit(limit).
		Find(&coins).Error
	return coins, err
}

// Create inserts a new coin and returns it with images preloaded.
func (r *CoinRepository) Create(coin *models.Coin) error {
	if err := r.db.Create(coin).Error; err != nil {
		return err
	}
	return r.db.Preload("Images").Preload("References").Preload("StorageLocation").First(coin, coin.ID).Error
}

// Update applies changes to an existing coin and reloads it with images.
func (r *CoinRepository) Update(existing *models.Coin, updates *models.Coin) error {
	if err := r.db.Model(existing).Updates(updates).Error; err != nil {
		return err
	}
	return r.db.Preload("Images").Preload("References").Preload("StorageLocation").First(existing, existing.ID).Error
}

// UpdateField updates a single field on a coin.
func (r *CoinRepository) UpdateField(coin *models.Coin, field string, value interface{}) error {
	if err := r.db.Model(coin).Update(field, value).Error; err != nil {
		return err
	}
	return r.db.Preload("Images").Preload("References").Preload("StorageLocation").First(coin, coin.ID).Error
}

// UpdateFields updates multiple fields on a coin using a map.
func (r *CoinRepository) UpdateFields(coin *models.Coin, updates map[string]interface{}) error {
	if err := r.db.Model(coin).Updates(updates).Error; err != nil {
		return err
	}
	return r.db.Preload("Images").Preload("References").Preload("StorageLocation").First(coin, coin.ID).Error
}

// UpdateStorageLocationID updates a coin storage-location foreign key, including clearing it.
func (r *CoinRepository) UpdateStorageLocationID(coin *models.Coin, storageLocationID *uint) error {
	if err := r.db.Model(coin).Update("storage_location_id", storageLocationID).Error; err != nil {
		return err
	}
	return r.db.Preload("Images").Preload("References").Preload("StorageLocation").First(coin, coin.ID).Error
}

// Delete removes a coin and all associated data (images, journals, value
// history) in a single transaction. Returns rows affected for the coin delete.
func (r *CoinRepository) Delete(id uint, userID uint) (int64, error) {
	var rowsAffected int64
	err := r.db.Transaction(func(tx *gorm.DB) error {
		result := tx.Scopes(OwnedByID(id, userID)).Delete(&models.Coin{})
		if result.Error != nil {
			return result.Error
		}
		rowsAffected = result.RowsAffected
		if rowsAffected == 0 {
			return nil
		}
		if err := tx.Where("coin_id = ?", id).Delete(&models.CoinImage{}).Error; err != nil {
			return err
		}
		if err := tx.Where("coin_id = ?", id).Delete(&models.CoinJournal{}).Error; err != nil {
			return err
		}
		if err := tx.Where("coin_id = ?", id).Delete(&models.CoinValueHistory{}).Error; err != nil {
			return err
		}
		if err := tx.Where("coin_id = ?", id).Delete(&models.CoinComment{}).Error; err != nil {
			return err
		}
		if err := tx.Where("coin_id = ?", id).Delete(&models.AvailabilityResult{}).Error; err != nil {
			return err
		}
		// Nullify auction lot references (lot survives, just unlinked)
		if err := tx.Model(&models.AuctionLot{}).Where("coin_id = ?", id).Update("coin_id", nil).Error; err != nil {
			return err
		}
		return nil
	})
	return rowsAffected, err
}

// BulkDelete removes multiple coins and all associated data in a single transaction.
func (r *CoinRepository) BulkDelete(coinIDs []uint, userID uint) (int64, error) {
	var rowsAffected int64
	err := r.db.Transaction(func(tx *gorm.DB) error {
		result := tx.Where("id IN ? AND user_id = ?", coinIDs, userID).Delete(&models.Coin{})
		if result.Error != nil {
			return result.Error
		}
		rowsAffected = result.RowsAffected
		if rowsAffected == 0 {
			return nil
		}
		if err := tx.Where("coin_id IN ?", coinIDs).Delete(&models.CoinImage{}).Error; err != nil {
			return err
		}
		if err := tx.Where("coin_id IN ?", coinIDs).Delete(&models.CoinJournal{}).Error; err != nil {
			return err
		}
		if err := tx.Where("coin_id IN ?", coinIDs).Delete(&models.CoinValueHistory{}).Error; err != nil {
			return err
		}
		if err := tx.Where("coin_id IN ?", coinIDs).Delete(&models.CoinComment{}).Error; err != nil {
			return err
		}
		if err := tx.Where("coin_id IN ?", coinIDs).Delete(&models.AvailabilityResult{}).Error; err != nil {
			return err
		}
		if err := tx.Where("coin_id IN ?", coinIDs).Delete(&models.CoinTag{}).Error; err != nil {
			return err
		}
		if err := tx.Model(&models.AuctionLot{}).Where("coin_id IN ?", coinIDs).Update("coin_id", nil).Error; err != nil {
			return err
		}
		return nil
	})
	return rowsAffected, err
}

// BulkMarkSold marks multiple coins as sold in a single transaction.
func (r *CoinRepository) BulkMarkSold(coinIDs []uint, userID uint) (int64, error) {
	result := r.db.Model(&models.Coin{}).
		Where("id IN ? AND user_id = ? AND is_sold = ?", coinIDs, userID, false).
		Updates(map[string]interface{}{
			"is_sold":   true,
			"sold_date": time.Now(),
		})
	return result.RowsAffected, result.Error
}

// BulkAssignLocation assigns a storage location to multiple coins. A nil storageLocationID clears the location.
func (r *CoinRepository) BulkAssignLocation(coinIDs []uint, storageLocationID *uint, userID uint) (int64, error) {
	result := r.db.Model(&models.Coin{}).
		Where("id IN ? AND user_id = ?", coinIDs, userID).
		Update("storage_location_id", storageLocationID)
	return result.RowsAffected, result.Error
}

// GetByIDs returns coins matching the given IDs for the given user.
func (r *CoinRepository) GetByIDs(coinIDs []uint, userID uint) ([]models.Coin, error) {
	var coins []models.Coin
	err := r.db.Where("id IN ? AND user_id = ?", coinIDs, userID).
		Preload("Images").Preload("Tags").Preload("References").Preload("StorageLocation").Find(&coins).Error
	return coins, err
}

// GetStats returns aggregate collection statistics for a user.
func (r *CoinRepository) GetStats(userID uint) (*CollectionStats, error) {
	stats := &CollectionStats{}
	active := ActiveCollection(userID)

	r.db.Model(&models.Coin{}).Scopes(active).Count(&stats.TotalCoins)
	r.db.Model(&models.Coin{}).Where("user_id = ? AND is_wishlist = ?", userID, true).Count(&stats.TotalWishlist)
	r.db.Model(&models.Coin{}).Where("user_id = ? AND is_sold = ?", userID, true).Count(&stats.TotalSold)

	r.db.Model(&models.Coin{}).
		Select("category, count(*) as count").
		Scopes(active).Group("category").
		Scan(&stats.ByCategory)

	r.db.Model(&models.Coin{}).
		Select("material, count(*) as count").
		Scopes(active).Group("material").
		Scan(&stats.ByMaterial)

	r.db.Model(&models.Coin{}).
		Select("grade, count(*) as count").
		Where("user_id = ? AND is_wishlist = ? AND is_sold = ? AND grade != ''", userID, false, false).
		Group("grade").Order("count DESC").
		Scan(&stats.ByGrade)

	r.db.Model(&models.Coin{}).
		Select("era, count(*) as count").
		Where("user_id = ? AND is_wishlist = ? AND is_sold = ? AND era != ''", userID, false, false).
		Group("era").Order("count DESC").
		Scan(&stats.ByEra)

	r.db.Model(&models.Coin{}).
		Select("ruler, count(*) as count").
		Where("user_id = ? AND is_wishlist = ? AND is_sold = ? AND ruler != ''", userID, false, false).
		Group("ruler").Order("count DESC").Limit(10).
		Scan(&stats.ByRuler)

	r.db.Model(&models.Coin{}).
		Select(`CASE
			WHEN purchase_price < 50 THEN 'Under $50'
			WHEN purchase_price >= 50 AND purchase_price < 200 THEN '$50 - $200'
			WHEN purchase_price >= 200 AND purchase_price < 500 THEN '$200 - $500'
			WHEN purchase_price >= 500 AND purchase_price < 1000 THEN '$500 - $1K'
			ELSE '$1K+'
		END as `+"`range`"+`, count(*) as count`).
		Where("user_id = ? AND is_wishlist = ? AND is_sold = ? AND purchase_price IS NOT NULL", userID, false, false).
		Group("`range`").
		Scan(&stats.ByPriceRange)

	r.db.Model(&models.Coin{}).
		Select("COALESCE(SUM(purchase_price), 0) as total_purchase_price, COALESCE(SUM(current_value), 0) as total_current_value, COALESCE(AVG(purchase_price), 0) as avg_purchase_price, COALESCE(AVG(current_value), 0) as avg_current_value").
		Scopes(active).
		Scan(&stats.Values)

	r.db.Model(&models.Coin{}).
		Select("COALESCE(SUM(sold_price), 0) as total_sold_price, COALESCE(SUM(purchase_price), 0) as total_purchase_cost").
		Where("user_id = ? AND is_sold = ?", userID, true).
		Scan(&stats.SoldValues)

	return stats, nil
}

// GetDistribution returns era × category cross-tabulation counts for the heat map.
func (r *CoinRepository) GetDistribution(userID uint) ([]DistributionCell, error) {
	var cells []DistributionCell
	err := r.db.Model(&models.Coin{}).
		Select("era, category, COUNT(*) as count").
		Where("user_id = ? AND is_wishlist = ? AND is_sold = ? AND era != '' AND category != ''", userID, false, false).
		Group("era, category").
		Order("era, category").
		Scan(&cells).Error
	return cells, err
}

// validSuggestionColumns is the allowlist of columns permitted in Suggestions queries.
var validSuggestionColumns = map[string]bool{
	"name":              true,
	"denomination":      true,
	"ruler":             true,
	"era":               true,
	"purchase_location": true,
}

// Suggestions returns distinct values for an autocomplete field.
func (r *CoinRepository) Suggestions(userID uint, column string, q string) ([]string, error) {
	if !validSuggestionColumns[column] {
		return nil, fmt.Errorf("invalid suggestion column: %s", column)
	}

	var values []string
	query := r.db.Model(&models.Coin{}).
		Where("user_id = ? AND "+column+" != ''", userID).
		Distinct(column).
		Order(column)

	if q != "" {
		query = query.Where(column+" LIKE ?", "%"+q+"%")
	}

	err := query.Limit(20).Pluck(column, &values).Error
	return values, err
}

// GetValueHistory returns portfolio-level value snapshots for a user.
func (r *CoinRepository) GetValueHistory(userID uint) ([]models.ValueSnapshot, error) {
	var snapshots []models.ValueSnapshot
	err := r.db.Scopes(OwnedBy(userID)).
		Order("recorded_at ASC").
		Find(&snapshots).Error
	return snapshots, err
}

// GetCoinValueHistory returns value tracking entries for a specific coin.
func (r *CoinRepository) GetCoinValueHistory(coinID uint, userID uint) ([]models.CoinValueHistory, error) {
	var entries []models.CoinValueHistory
	err := r.db.Where("coin_id = ? AND user_id = ?", coinID, userID).
		Order("recorded_at ASC").
		Find(&entries).Error
	return entries, err
}

// GetWishlistWithURLs returns wishlist coins with a non-empty ReferenceURL for a user.
func (r *CoinRepository) GetWishlistWithURLs(userID uint) ([]models.Coin, error) {
	var coins []models.Coin
	err := r.db.Scopes(OwnedBy(userID)).
		Where("is_wishlist = ? AND reference_url != ''", true).
		Preload("Images").
		Find(&coins).Error
	return coins, err
}

// GetAllWishlistWithURLs returns all users' wishlist coins with a non-empty ReferenceURL.
func (r *CoinRepository) GetAllWishlistWithURLs() ([]models.Coin, error) {
	var coins []models.Coin
	err := r.db.Where("is_wishlist = ? AND reference_url != ''", true).
		Preload("Images").
		Find(&coins).Error
	return coins, err
}

// UpdateListingStatus updates only the listing-check fields on a coin.
func (r *CoinRepository) UpdateListingStatus(coinID uint, status, reason string, checkedAt time.Time) error {
	return r.db.Model(&models.Coin{}).Where("id = ?", coinID).
		Updates(map[string]interface{}{
			"listing_status":       status,
			"listing_checked_at":   checkedAt,
			"listing_check_reason": reason,
		}).Error
}

// CoinExists checks if a coin exists for the given user.
func (r *CoinRepository) CoinExists(coinID uint, userID uint) (bool, error) {
	var count int64
	err := r.db.Model(&models.Coin{}).Scopes(OwnedByID(coinID, userID)).Count(&count).Error
	return count > 0, err
}

// RecordValueHistory creates a coin value history entry.
func (r *CoinRepository) RecordValueHistory(entry *models.CoinValueHistory) error {
	return r.db.Create(entry).Error
}

// CreateJournalEntry creates a journal entry for a coin.
func (r *CoinRepository) CreateJournalEntry(entry *models.CoinJournal) error {
	return r.db.Create(entry).Error
}

// RecordValueSnapshot captures the current total value, invested amount,
// and coin count for a user.
func (r *CoinRepository) RecordValueSnapshot(userID uint) error {
	type result struct {
		TotalValue    float64
		TotalInvested float64
		CoinCount     int64
	}
	var res result
	r.db.Model(&models.Coin{}).
		Select("COALESCE(SUM(current_value), 0) as total_value, COALESCE(SUM(purchase_price), 0) as total_invested, COUNT(*) as coin_count").
		Where("user_id = ? AND is_wishlist = ?", userID, false).
		Scan(&res)

	snapshot := models.ValueSnapshot{
		UserID:        userID,
		TotalValue:    res.TotalValue,
		TotalInvested: res.TotalInvested,
		CoinCount:     res.CoinCount,
		RecordedAt:    time.Now(),
	}
	return r.db.Create(&snapshot).Error
}
