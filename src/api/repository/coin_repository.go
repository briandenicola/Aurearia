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
	SetID     *uint
	SortField string
	SortOrder string
	Seed      *int // for SortField == "random"; deterministic per-seed shuffle
	Page      int
	Limit     int
}

// OwnedCoinFilters are lightweight scoped filters used by collection chat tools.
type OwnedCoinFilters struct {
	Category      string
	Material      string
	Era           string
	Ruler         string
	Wishlist      *bool
	Sold          *bool
	Search        string
	MissingFields []string
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

const (
	InvestmentBreakdownPurchaseMonth = "purchase-month"
	InvestmentBreakdownMaterial      = "material"
)

// InvestmentBreakdownSegment holds one portfolio investment chart segment.
type InvestmentBreakdownSegment struct {
	Label                     string  `json:"label"`
	Year                      *int    `json:"year,omitempty"`
	Month                     *int    `json:"month,omitempty"`
	Invested                  float64 `json:"invested"`
	CurrentValue              float64 `json:"currentValue"`
	GainLoss                  float64 `json:"gainLoss"`
	GainLossPct               float64 `json:"gainLossPct"`
	CoinCount                 int64   `json:"coinCount"`
	MissingCurrentValueCount  int64   `json:"missingCurrentValueCount"`
	MissingPurchasePriceCount int64   `json:"missingPurchasePriceCount"`
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

func omitCoinRelationships(db *gorm.DB) *gorm.DB {
	return db.Omit("Tags", "Sets")
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
	missingConditions := make([]string, 0, len(filters.MissingFields))
	for _, field := range filters.MissingFields {
		if condition, ok := ownedMissingFieldConditions[field]; ok {
			missingConditions = append(missingConditions, "("+condition+")")
		}
	}
	if len(missingConditions) > 0 {
		query = query.Where(strings.Join(missingConditions, " OR "))
	}
	return query
}

var ownedMissingFieldConditions = map[string]string{
	"denomination":    "denomination = ''",
	"ruler":           "ruler = ''",
	"era":             "era = ''",
	"mint":            "mint = ''",
	"material":        "material = '' OR material = 'Other'",
	"weightGrams":     "weight_grams IS NULL OR weight_grams <= 0",
	"diameterMm":      "diameter_mm IS NULL OR diameter_mm <= 0",
	"grade":           "grade = ''",
	"purchasePrice":   "purchase_price IS NULL",
	"currentValue":    "current_value IS NULL",
	"purchaseDate":    "purchase_date IS NULL",
	"storageLocation": "storage_location_id IS NULL",
	"notes":           "notes = ''",
	"referenceUrl":    "reference_url = ''",
	"referenceText":   "reference_text = ''",
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
	query := r.db.Scopes(OwnedBy(userID)).Preload("Images").Preload("Tags").Preload("Sets").Preload("References").Preload("StorageLocation")

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
	if filters.SetID != nil {
		query = query.Where("id IN (SELECT coin_id FROM coin_set_memberships WHERE set_id = ?)", *filters.SetID)
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
	err := r.db.Scopes(OwnedByID(id, userID)).Preload("Images").Preload("Tags").Preload("Sets").Preload("References").Preload("StorageLocation").First(&coin).Error
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

// Duplicate creates an owner-scoped copy of a coin without copying media or card state rows.
func (r *CoinRepository) Duplicate(id uint, userID uint) (*models.Coin, error) {
	var source models.Coin
	if err := r.db.Scopes(OwnedByID(id, userID)).
		Preload("References").
		Preload("StorageLocation").
		First(&source).Error; err != nil {
		return nil, err
	}

	duplicate := models.Coin{
		Name:                  source.Name + " (duplicate)",
		Category:              source.Category,
		Denomination:          source.Denomination,
		Ruler:                 source.Ruler,
		Era:                   source.Era,
		Mint:                  source.Mint,
		Material:              source.Material,
		WeightGrams:           source.WeightGrams,
		DiameterMm:            source.DiameterMm,
		Grade:                 source.Grade,
		ObverseInscription:    source.ObverseInscription,
		ReverseInscription:    source.ReverseInscription,
		ObverseDescription:    source.ObverseDescription,
		ReverseDescription:    source.ReverseDescription,
		RarityRating:          source.RarityRating,
		PurchasePrice:         source.PurchasePrice,
		CurrentValue:          source.CurrentValue,
		CurrentValueUpdatedAt: source.CurrentValueUpdatedAt,
		PurchaseDate:          source.PurchaseDate,
		PurchaseLocation:      source.PurchaseLocation,
		Notes:                 source.Notes,
		AIAnalysis:            source.AIAnalysis,
		ObverseAnalysis:       source.ObverseAnalysis,
		ReverseAnalysis:       source.ReverseAnalysis,
		ReferenceURL:          source.ReferenceURL,
		ReferenceText:         source.ReferenceText,
		IsWishlist:            source.IsWishlist,
		IsSold:                source.IsSold,
		SoldPrice:             source.SoldPrice,
		SoldDate:              source.SoldDate,
		SoldTo:                source.SoldTo,
		ListingStatus:         source.ListingStatus,
		ListingCheckedAt:      source.ListingCheckedAt,
		ListingCheckReason:    source.ListingCheckReason,
		StorageLocationID:     source.StorageLocationID,
		IsPrivate:             source.IsPrivate,
		UserID:                source.UserID,
	}
	if err := r.db.Omit("Images", "References", "Tags", "Sets", "StorageLocation", "User").Create(&duplicate).Error; err != nil {
		return nil, err
	}

	if len(source.References) > 0 {
		references := make([]models.CoinReference, 0, len(source.References))
		for _, ref := range source.References {
			references = append(references, models.CoinReference{
				CoinID:        duplicate.ID,
				Catalog:       ref.Catalog,
				Volume:        ref.Volume,
				Number:        ref.Number,
				InvoiceNumber: ref.InvoiceNumber,
				URI:           ref.URI,
			})
		}
		if err := r.db.Create(&references).Error; err != nil {
			return nil, err
		}
	}

	var tagIDs []uint
	if err := r.db.Table("coin_tags").
		Select("coin_tags.tag_id").
		Joins("JOIN tags ON tags.id = coin_tags.tag_id").
		Where("coin_tags.coin_id = ? AND tags.user_id = ?", source.ID, userID).
		Scan(&tagIDs).Error; err != nil {
		return nil, err
	}
	if len(tagIDs) > 0 {
		coinTags := make([]models.CoinTag, 0, len(tagIDs))
		for _, tagID := range tagIDs {
			coinTags = append(coinTags, models.CoinTag{CoinID: duplicate.ID, TagID: tagID})
		}
		if err := r.db.Create(&coinTags).Error; err != nil {
			return nil, err
		}
	}

	var memberships []models.CoinSetMembership
	if err := r.db.Table("coin_set_memberships").
		Select("coin_set_memberships.*").
		Joins("JOIN coin_sets ON coin_sets.id = coin_set_memberships.set_id").
		Where("coin_set_memberships.coin_id = ? AND coin_sets.user_id = ?", source.ID, userID).
		Scan(&memberships).Error; err != nil {
		return nil, err
	}
	if len(memberships) > 0 {
		copiedMemberships := make([]models.CoinSetMembership, 0, len(memberships))
		for _, membership := range memberships {
			copiedMemberships = append(copiedMemberships, models.CoinSetMembership{
				SetID:     membership.SetID,
				CoinID:    duplicate.ID,
				AddedAt:   membership.AddedAt,
				SortOrder: membership.SortOrder,
				Notes:     membership.Notes,
			})
		}
		if err := r.db.Create(&copiedMemberships).Error; err != nil {
			return nil, err
		}
	}

	if err := r.db.Preload("Images").Preload("References").Preload("Tags").Preload("Sets").Preload("StorageLocation").First(&duplicate, duplicate.ID).Error; err != nil {
		return nil, err
	}
	return &duplicate, nil
}

// Update applies a scalar patch to an existing coin and reloads read associations.
// When selectFields are supplied, only those fields are persisted, including
// explicit zero values; relationship changes stay on their dedicated paths.
func (r *CoinRepository) Update(existing *models.Coin, updates *models.Coin, selectFields ...string) error {
	// Relationship changes are managed through dedicated tag/set methods.
	// Coin sets require explicit membership writes because AddedAt is NOT NULL.
	query := omitCoinRelationships(r.db.Model(existing))
	if len(selectFields) > 0 {
		query = query.Select(selectFields)
	}
	if err := query.Updates(updates).Error; err != nil {
		return err
	}
	return r.db.Preload("Images").Preload("References").Preload("StorageLocation").First(existing, existing.ID).Error
}

// UpdateField updates one explicit column without syncing loaded associations.
func (r *CoinRepository) UpdateField(coin *models.Coin, field string, value interface{}) error {
	if err := omitCoinRelationships(r.db.Model(coin)).Update(field, value).Error; err != nil {
		return err
	}
	return r.db.Preload("Images").Preload("References").Preload("StorageLocation").First(coin, coin.ID).Error
}

// UpdateFields updates multiple explicit columns, including zero/nil values, without syncing loaded associations.
func (r *CoinRepository) UpdateFields(coin *models.Coin, updates map[string]interface{}) error {
	if err := omitCoinRelationships(r.db.Model(coin)).Updates(updates).Error; err != nil {
		return err
	}
	return r.db.Preload("Images").Preload("References").Preload("StorageLocation").First(coin, coin.ID).Error
}

// UpdateStorageLocationID updates only the storage-location foreign key, including clearing it.
func (r *CoinRepository) UpdateStorageLocationID(coin *models.Coin, storageLocationID *uint) error {
	query := r.db.Model(&models.Coin{}).Where("id = ? AND user_id = ?", coin.ID, coin.UserID)
	if storageLocationID == nil {
		if err := query.Update("storage_location_id", nil).Error; err != nil {
			return err
		}
	} else {
		if err := query.Update("storage_location_id", *storageLocationID).Error; err != nil {
			return err
		}
	}
	if err := r.db.Preload("Images").Preload("References").Preload("StorageLocation").First(coin, coin.ID).Error; err != nil {
		return err
	}
	if storageLocationID == nil {
		coin.StorageLocation = nil
	}
	return nil
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

// GetInvestmentBreakdown returns active-collection investment aggregates for the requested dimension.
func (r *CoinRepository) GetInvestmentBreakdown(userID uint, dimension string) ([]InvestmentBreakdownSegment, error) {
	switch dimension {
	case InvestmentBreakdownPurchaseMonth:
		return r.getInvestmentBreakdownByPurchaseMonth(userID)
	case InvestmentBreakdownMaterial:
		return r.getInvestmentBreakdownByMaterial(userID)
	default:
		return nil, fmt.Errorf("unsupported investment breakdown dimension: %s", dimension)
	}
}

func (r *CoinRepository) getInvestmentBreakdownByPurchaseMonth(userID uint) ([]InvestmentBreakdownSegment, error) {
	var segments []InvestmentBreakdownSegment
	err := r.db.Model(&models.Coin{}).
		Select(`
			CAST(strftime('%Y', purchase_date) AS INTEGER) AS year,
			CAST(strftime('%m', purchase_date) AS INTEGER) AS month,
			COALESCE(SUM(purchase_price), 0) AS invested,
			COALESCE(SUM(COALESCE(current_value, purchase_price, 0)), 0) AS current_value,
			COALESCE(SUM(COALESCE(current_value, purchase_price, 0)), 0) - COALESCE(SUM(purchase_price), 0) AS gain_loss,
			CASE
				WHEN COALESCE(SUM(purchase_price), 0) = 0 THEN 0
				ELSE ((COALESCE(SUM(COALESCE(current_value, purchase_price, 0)), 0) - COALESCE(SUM(purchase_price), 0)) / COALESCE(SUM(purchase_price), 0)) * 100
			END AS gain_loss_pct,
			COUNT(*) AS coin_count,
			SUM(CASE WHEN current_value IS NULL THEN 1 ELSE 0 END) AS missing_current_value_count,
			SUM(CASE WHEN purchase_price IS NULL THEN 1 ELSE 0 END) AS missing_purchase_price_count`).
		Scopes(ActiveCollection(userID)).
		Where("purchase_date IS NOT NULL").
		Group("year, month").
		Order("year ASC, month ASC").
		Scan(&segments).Error
	if err != nil {
		return nil, err
	}
	for i := range segments {
		if segments[i].Year != nil && segments[i].Month != nil {
			segments[i].Label = fmt.Sprintf("%s %04d", time.Month(*segments[i].Month).String()[:3], *segments[i].Year)
		}
	}
	return segments, nil
}

func (r *CoinRepository) getInvestmentBreakdownByMaterial(userID uint) ([]InvestmentBreakdownSegment, error) {
	var segments []InvestmentBreakdownSegment
	err := r.db.Model(&models.Coin{}).
		Select(`
			CASE WHEN TRIM(COALESCE(material, '')) = '' THEN 'Other' ELSE material END AS label,
			COALESCE(SUM(purchase_price), 0) AS invested,
			COALESCE(SUM(COALESCE(current_value, purchase_price, 0)), 0) AS current_value,
			COALESCE(SUM(COALESCE(current_value, purchase_price, 0)), 0) - COALESCE(SUM(purchase_price), 0) AS gain_loss,
			CASE
				WHEN COALESCE(SUM(purchase_price), 0) = 0 THEN 0
				ELSE ((COALESCE(SUM(COALESCE(current_value, purchase_price, 0)), 0) - COALESCE(SUM(purchase_price), 0)) / COALESCE(SUM(purchase_price), 0)) * 100
			END AS gain_loss_pct,
			COUNT(*) AS coin_count,
			SUM(CASE WHEN current_value IS NULL THEN 1 ELSE 0 END) AS missing_current_value_count,
			SUM(CASE WHEN purchase_price IS NULL THEN 1 ELSE 0 END) AS missing_purchase_price_count`).
		Scopes(ActiveCollection(userID)).
		Group("label").
		Order("invested DESC, label ASC").
		Scan(&segments).Error
	return segments, err
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

// FindWishlistByReferenceURL returns an owned wishlist coin matching the exact source URL.
func (r *CoinRepository) FindWishlistByReferenceURL(userID uint, referenceURL string) (*models.Coin, error) {
	var coin models.Coin
	err := r.db.Scopes(OwnedBy(userID)).
		Where("is_wishlist = ? AND reference_url = ?", true, referenceURL).
		First(&coin).Error
	if err != nil {
		return nil, err
	}
	return &coin, nil
}

// FindWishlistBySourceAlertCandidateID returns an owned wishlist coin converted from a candidate.
func (r *CoinRepository) FindWishlistBySourceAlertCandidateID(userID uint, candidateID uint) (*models.Coin, error) {
	var coin models.Coin
	err := r.db.Scopes(OwnedBy(userID)).
		Where("is_wishlist = ? AND source_alert_candidate_id = ?", true, candidateID).
		First(&coin).Error
	if err != nil {
		return nil, err
	}
	return &coin, nil
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
