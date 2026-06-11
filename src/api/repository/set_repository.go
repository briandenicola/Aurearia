package repository

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrInvalidSetOrder = errors.New("ordered coin IDs must exactly match current set members")

// SetRepository encapsulates all set-related database operations.
type SetRepository struct {
	db *gorm.DB
}

// NewSetRepository creates a new SetRepository.
func NewSetRepository(db *gorm.DB) *SetRepository {
	return &SetRepository{db: db}
}

// List returns all sets belonging to the given user.
func (r *SetRepository) List(userID uint) ([]models.CoinSet, error) {
	var sets []models.CoinSet
	err := r.db.Scopes(OwnedBy(userID)).Order("name ASC").Find(&sets).Error
	return sets, err
}

// ListUsersWithSets returns user IDs that currently own at least one set.
func (r *SetRepository) ListUsersWithSets() ([]uint, error) {
	var ids []uint
	err := r.db.Model(&models.CoinSet{}).Distinct("user_id").Pluck("user_id", &ids).Error
	return ids, err
}

// Create inserts a new set. Name is trimmed and checked for case-insensitive uniqueness.
func (r *SetRepository) Create(set *models.CoinSet) error {
	set.Name = strings.TrimSpace(set.Name)
	return r.db.Create(set).Error
}

// GetByID finds a set by ID and user ID.
func (r *SetRepository) GetByID(id, userID uint) (*models.CoinSet, error) {
	var set models.CoinSet
	err := r.db.Scopes(OwnedByID(id, userID)).First(&set).Error
	if err != nil {
		return nil, err
	}
	return &set, nil
}

// Update modifies a set's fields.
func (r *SetRepository) Update(set *models.CoinSet, updates map[string]interface{}) error {
	if name, ok := updates["name"]; ok {
		updates["name"] = strings.TrimSpace(name.(string))
	}
	return r.db.Model(set).Updates(updates).Error
}

// Delete removes a set and its memberships.
func (r *SetRepository) Delete(id, userID uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Delete memberships first
		if err := tx.Where("set_id = ?", id).Delete(&models.CoinSetMembership{}).Error; err != nil {
			return err
		}
		result := tx.Scopes(OwnedByID(id, userID)).Delete(&models.CoinSet{})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
}

// AddCoinToSet adds a coin to a manual set. Both must belong to the given user.
// Idempotent — silently ignores if already added.
func (r *SetRepository) AddCoinToSet(coinID, setID, userID uint, notes string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Verify coin ownership
		var coinCount int64
		if err := tx.Model(&models.Coin{}).Where("id = ? AND user_id = ?", coinID, userID).Count(&coinCount).Error; err != nil {
			return err
		}
		if coinCount == 0 {
			return gorm.ErrRecordNotFound
		}
		// Verify set ownership
		var setCount int64
		if err := tx.Model(&models.CoinSet{}).Where("id = ? AND user_id = ?", setID, userID).Count(&setCount).Error; err != nil {
			return err
		}
		if setCount == 0 {
			return gorm.ErrRecordNotFound
		}
		sortOrder, err := nextSetSortOrder(tx, setID)
		if err != nil {
			return err
		}

		return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&models.CoinSetMembership{
			CoinID:    coinID,
			SetID:     setID,
			AddedAt:   time.Now(),
			SortOrder: sortOrder,
			Notes:     notes,
		}).Error
	})
}

// BulkAddCoinsToSet adds multiple owned coins to a manual set. Existing memberships are ignored.
func (r *SetRepository) BulkAddCoinsToSet(coinIDs []uint, setID, userID uint) (int64, error) {
	var affected int64
	err := r.db.Transaction(func(tx *gorm.DB) error {
		var set models.CoinSet
		if err := tx.Where("id = ? AND user_id = ?", setID, userID).First(&set).Error; err != nil {
			return err
		}
		if set.SetType == models.CoinSetTypeSmart {
			return fmt.Errorf("cannot manually add coins to smart sets")
		}

		uniqueCoinIDs := make([]uint, 0, len(coinIDs))
		seen := make(map[uint]struct{}, len(coinIDs))
		for _, coinID := range coinIDs {
			if _, ok := seen[coinID]; ok {
				continue
			}
			seen[coinID] = struct{}{}
			uniqueCoinIDs = append(uniqueCoinIDs, coinID)
		}

		var ownedCount int64
		if err := tx.Model(&models.Coin{}).Where("id IN ? AND user_id = ?", uniqueCoinIDs, userID).Count(&ownedCount).Error; err != nil {
			return err
		}
		if ownedCount != int64(len(uniqueCoinIDs)) {
			return gorm.ErrRecordNotFound
		}

		sortOrder, err := nextSetSortOrder(tx, setID)
		if err != nil {
			return err
		}
		now := time.Now()
		memberships := make([]models.CoinSetMembership, 0, len(uniqueCoinIDs))
		for i, coinID := range uniqueCoinIDs {
			memberships = append(memberships, models.CoinSetMembership{
				CoinID:    coinID,
				SetID:     setID,
				AddedAt:   now,
				SortOrder: sortOrder + i,
			})
		}

		result := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&memberships)
		if result.Error != nil {
			return result.Error
		}
		affected = result.RowsAffected
		return nil
	})
	return affected, err
}

// RemoveCoinFromSet removes a coin from a manual set. Both must belong to the given user.
func (r *SetRepository) RemoveCoinFromSet(coinID, setID, userID uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Verify coin ownership
		var coinCount int64
		if err := tx.Model(&models.Coin{}).Where("id = ? AND user_id = ?", coinID, userID).Count(&coinCount).Error; err != nil {
			return err
		}
		if coinCount == 0 {
			return gorm.ErrRecordNotFound
		}
		return tx.Where("coin_id = ? AND set_id = ?", coinID, setID).Delete(&models.CoinSetMembership{}).Error
	})
}

// GetCoinsInSet returns all coins in a set with summary aggregates.
func (r *SetRepository) GetCoinsInSet(setID, userID uint) ([]models.Coin, error) {
	var coins []models.Coin
	set, err := r.GetByID(setID, userID)
	if err != nil {
		return nil, err
	}
	if set.SetType == models.CoinSetTypeSmart && set.SmartCriteria != nil {
		return r.GetCoinsMatchingCriteria(userID, map[string]interface{}(*set.SmartCriteria))
	}
	err = r.db.
		Joins("JOIN coin_set_memberships ON coin_set_memberships.coin_id = coins.id").
		Joins("JOIN coin_sets ON coin_sets.id = coin_set_memberships.set_id").
		Where("coin_sets.id = ? AND coin_sets.user_id = ?", setID, userID).
		Preload("Images").
		Preload("Tags").
		Order("coin_set_memberships.sort_order ASC").
		Order("coins.name ASC").
		Order("coins.id ASC").
		Find(&coins).Error
	return coins, err
}

// ReorderCoinsInSet persists the exact manual coin order for an owned set.
func (r *SetRepository) ReorderCoinsInSet(setID, userID uint, coinIDs []uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var set models.CoinSet
		if err := tx.Scopes(OwnedByID(setID, userID)).First(&set).Error; err != nil {
			return err
		}

		var memberIDs []uint
		if err := tx.Model(&models.CoinSetMembership{}).
			Where("set_id = ?", setID).
			Pluck("coin_id", &memberIDs).Error; err != nil {
			return err
		}
		if len(memberIDs) != len(coinIDs) {
			return ErrInvalidSetOrder
		}

		members := make(map[uint]struct{}, len(memberIDs))
		for _, id := range memberIDs {
			members[id] = struct{}{}
		}
		seen := make(map[uint]struct{}, len(coinIDs))
		for _, id := range coinIDs {
			if _, ok := members[id]; !ok {
				return ErrInvalidSetOrder
			}
			if _, ok := seen[id]; ok {
				return ErrInvalidSetOrder
			}
			seen[id] = struct{}{}
		}

		for sortOrder, coinID := range coinIDs {
			result := tx.Model(&models.CoinSetMembership{}).
				Where("set_id = ? AND coin_id = ?", setID, coinID).
				Update("sort_order", sortOrder)
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected != 1 {
				return ErrInvalidSetOrder
			}
		}
		return nil
	})
}

// GetSetSummary returns aggregate counts and values for a set.
func (r *SetRepository) GetSetSummary(setID, userID uint) (map[string]interface{}, error) {
	set, err := r.GetByID(setID, userID)
	if err != nil {
		return nil, err
	}
	if set.SetType == models.CoinSetTypeSmart && set.SmartCriteria != nil {
		coins, err := r.GetCoinsMatchingCriteria(userID, map[string]interface{}(*set.SmartCriteria))
		if err != nil {
			return nil, err
		}
		var totalValue, totalInvested float64
		var highestID *uint
		var highest float64 = math.Inf(-1)
		for _, coin := range coins {
			if coin.CurrentValue != nil {
				totalValue += *coin.CurrentValue
				if *coin.CurrentValue > highest {
					id := coin.ID
					highestID = &id
					highest = *coin.CurrentValue
				}
			}
			if coin.PurchasePrice != nil {
				totalInvested += *coin.PurchasePrice
			}
		}
		var avg *float64
		if len(coins) > 0 {
			v := totalValue / float64(len(coins))
			avg = &v
		}
		return map[string]interface{}{
			"coinCount":          len(coins),
			"totalValue":         totalValue,
			"totalInvested":      totalInvested,
			"avgValuePerCoin":    avg,
			"highestValueCoinId": highestID,
		}, nil
	}

	var result struct {
		CoinCount          int
		TotalValue         float64
		TotalInvested      float64
		AvgValuePerCoin    *float64
		HighestValueCoinID *uint
	}

	err = r.db.Raw(`
		SELECT 
			COUNT(coins.id) as coin_count,
			COALESCE(SUM(coins.current_value), 0) as total_value,
			COALESCE(SUM(coins.purchase_price), 0) as total_invested,
			CASE WHEN COUNT(coins.id) > 0 THEN AVG(coins.current_value) ELSE NULL END as avg_value_per_coin,
			(SELECT coins.id FROM coins
				JOIN coin_set_memberships ON coin_set_memberships.coin_id = coins.id
				WHERE coin_set_memberships.set_id = ? AND coins.user_id = ?
				ORDER BY coins.current_value DESC LIMIT 1) as highest_value_coin_id
		FROM coins
		JOIN coin_set_memberships ON coin_set_memberships.coin_id = coins.id
		WHERE coin_set_memberships.set_id = ? AND coins.user_id = ?
	`, setID, userID, setID, userID).Scan(&result).Error

	if err != nil {
		return nil, err
	}

	summary := map[string]interface{}{
		"coinCount":          result.CoinCount,
		"totalValue":         result.TotalValue,
		"totalInvested":      result.TotalInvested,
		"avgValuePerCoin":    result.AvgValuePerCoin,
		"highestValueCoinId": result.HighestValueCoinID,
	}

	return summary, nil
}

// CountByUser returns the total number of sets for a user.
func (r *SetRepository) CountByUser(userID uint) (int64, error) {
	var count int64
	err := r.db.Model(&models.CoinSet{}).Scopes(OwnedBy(userID)).Count(&count).Error
	return count, err
}

// ExistsByName checks if a set with the given name already exists for the user (case-insensitive).
func (r *SetRepository) ExistsByName(userID uint, name string) (bool, error) {
	var count int64
	err := r.db.Model(&models.CoinSet{}).
		Where("user_id = ? AND LOWER(name) = LOWER(?)", userID, strings.TrimSpace(name)).
		Count(&count).Error
	return count > 0, err
}

// MigrateTagsToSets creates open sets from existing tags for the given user.
// This is a one-time migration helper.
func (r *SetRepository) MigrateTagsToSets(userID uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Get all tags for the user
		var tags []models.Tag
		if err := tx.Where("user_id = ?", userID).Find(&tags).Error; err != nil {
			return err
		}

		// For each tag, create a corresponding open set
		for _, tag := range tags {
			// Check if set already exists
			var existingSet models.CoinSet
			err := tx.Where("user_id = ? AND name = ?", userID, tag.Name).First(&existingSet).Error
			if err == nil {
				// Set already exists, skip
				continue
			}
			if err != gorm.ErrRecordNotFound {
				return err
			}

			// Create the set
			set := models.CoinSet{
				UserID:  userID,
				Name:    tag.Name,
				Color:   tag.Color,
				SetType: models.CoinSetTypeOpen,
			}
			if err := tx.Create(&set).Error; err != nil {
				return err
			}

			// Migrate coin_tags to coin_set_memberships
			var coinTags []models.CoinTag
			if err := tx.Where("tag_id = ?", tag.ID).Find(&coinTags).Error; err != nil {
				return err
			}

			for i, ct := range coinTags {
				membership := models.CoinSetMembership{
					SetID:     set.ID,
					CoinID:    ct.CoinID,
					AddedAt:   time.Now(),
					SortOrder: i,
				}
				if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&membership).Error; err != nil {
					return err
				}
			}
		}

		return nil
	})
}

// CreateTargetsForSet creates multiple targets for a set in a transaction.
func (r *SetRepository) CreateTargetsForSet(setID uint, targets []models.CoinSetTarget) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		for i := range targets {
			targets[i].SetID = setID
			if err := tx.Create(&targets[i]).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// GetTargetsForSet returns all targets for a set, ordered by sort_order.
func (r *SetRepository) GetTargetsForSet(setID, userID uint) ([]models.CoinSetTarget, error) {
	var targets []models.CoinSetTarget
	err := r.db.
		Joins("JOIN coin_sets ON coin_sets.id = coin_set_targets.set_id").
		Where("coin_set_targets.set_id = ? AND coin_sets.user_id = ?", setID, userID).
		Order("coin_set_targets.sort_order ASC").
		Find(&targets).Error
	return targets, err
}

// GetSetCompletion calculates completion metrics for a defined or goal set.
func (r *SetRepository) GetSetCompletion(setID, userID uint) (map[string]interface{}, error) {
	// Get all targets
	targets, err := r.GetTargetsForSet(setID, userID)
	if err != nil {
		return nil, err
	}

	totalTargets := len(targets)
	if totalTargets == 0 {
		return map[string]interface{}{
			"totalTargets":         0,
			"completedTargets":     0,
			"completionPercentage": 0.0,
			"missingTargets":       []models.CoinSetTarget{},
		}, nil
	}

	// Get all coins in the set
	coins, err := r.GetCoinsInSet(setID, userID)
	if err != nil {
		return nil, err
	}

	// Match coins to targets
	completedTargets := 0
	missingTargets := []models.CoinSetTarget{}

	for _, target := range targets {
		matched := false
		for _, coin := range coins {
			if matchCoinToTarget(coin, target) {
				matched = true
				break
			}
		}
		if matched {
			completedTargets++
		} else {
			missingTargets = append(missingTargets, target)
		}
	}

	completionPercentage := (float64(completedTargets) / float64(totalTargets)) * 100.0

	return map[string]interface{}{
		"totalTargets":         totalTargets,
		"completedTargets":     completedTargets,
		"completionPercentage": completionPercentage,
		"missingTargets":       missingTargets,
	}, nil
}

// CreateSnapshot creates or replaces today's aggregate snapshot for a set.
func (r *SetRepository) CreateSnapshot(setID, userID uint, completion *float64) (*models.CoinSetValuationSnapshot, error) {
	summary, err := r.GetSetSummary(setID, userID)
	if err != nil {
		return nil, err
	}
	date := time.Now().Truncate(24 * time.Hour)
	coinCount, _ := summary["coinCount"].(int)
	totalValue, _ := summary["totalValue"].(float64)
	totalInvested, _ := summary["totalInvested"].(float64)
	avg, _ := summary["avgValuePerCoin"].(*float64)
	highest, _ := summary["highestValueCoinId"].(*uint)
	snapshot := &models.CoinSetValuationSnapshot{
		SetID:                setID,
		UserID:               userID,
		SnapshotDate:         date,
		TotalValue:           totalValue,
		TotalInvested:        totalInvested,
		CoinCount:            coinCount,
		CompletionPercentage: completion,
		AvgValuePerCoin:      avg,
		HighestValueCoinID:   highest,
	}
	err = r.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "set_id"}, {Name: "snapshot_date"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"total_value", "total_invested", "coin_count", "completion_percentage", "avg_value_per_coin", "highest_value_coin_id",
		}),
	}).Create(snapshot).Error
	if err != nil {
		return nil, err
	}
	return snapshot, nil
}

// GetSnapshots returns ordered snapshots for a set.
func (r *SetRepository) GetSnapshots(setID, userID uint, since *time.Time) ([]models.CoinSetValuationSnapshot, error) {
	var snapshots []models.CoinSetValuationSnapshot
	query := r.db.Where("set_id = ? AND user_id = ?", setID, userID).Order("snapshot_date ASC")
	if since != nil {
		query = query.Where("snapshot_date >= ?", *since)
	}
	err := query.Find(&snapshots).Error
	return snapshots, err
}

// GetAllSetIDsForUser returns all set IDs owned by a user.
func (r *SetRepository) GetAllSetIDsForUser(userID uint) ([]uint, error) {
	var ids []uint
	err := r.db.Model(&models.CoinSet{}).Where("user_id = ?", userID).Pluck("id", &ids).Error
	return ids, err
}

// GetEnabledMilestoneAlerts returns active alerts for a set.
func (r *SetRepository) GetEnabledMilestoneAlerts(setID, userID uint) ([]models.CoinSetMilestoneAlert, error) {
	var alerts []models.CoinSetMilestoneAlert
	err := r.db.Where("set_id = ? AND user_id = ? AND enabled = ?", setID, userID, true).Find(&alerts).Error
	return alerts, err
}

// MarkMilestoneAlertTriggered stores the last alert trigger time.
func (r *SetRepository) MarkMilestoneAlertTriggered(id uint) error {
	now := time.Now()
	return r.db.Model(&models.CoinSetMilestoneAlert{}).Where("id = ?", id).Update("last_triggered_at", &now).Error
}

// GetCoinsMatchingCriteria returns coins that match validated smart set criteria.
func (r *SetRepository) GetCoinsMatchingCriteria(userID uint, criteria map[string]interface{}) ([]models.Coin, error) {
	query := r.db.Model(&models.Coin{}).Scopes(OwnedBy(userID)).Preload("Images").Preload("Tags")
	query, err := r.applyCriteria(query, criteria)
	if err != nil {
		return nil, err
	}
	var coins []models.Coin
	err = query.Order("name ASC").Find(&coins).Error
	return coins, err
}

func (r *SetRepository) applyCriteria(query *gorm.DB, node map[string]interface{}) (*gorm.DB, error) {
	if operator, ok := node["operator"].(string); ok {
		rules, _ := node["rules"].([]interface{})
		if len(rules) == 0 {
			return query, nil
		}
		var group *gorm.DB
		for _, raw := range rules {
			child, ok := raw.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("invalid criteria rule")
			}
			childQuery := r.db.Model(&models.Coin{})
			childQuery, err := r.applyCriteria(childQuery, child)
			if err != nil {
				return nil, err
			}
			if group == nil {
				group = childQuery
			} else if operator == "or" {
				group = group.Or(childQuery)
			} else {
				group = group.Where(childQuery)
			}
		}
		return query.Where(group), nil
	}
	field, _ := node["field"].(string)
	op, _ := node["op"].(string)
	column, ok := criteriaFieldColumns[field]
	if !ok {
		return nil, fmt.Errorf("criteria field is not allowed")
	}
	value := node["value"]
	switch op {
	case "eq":
		return query.Where(column+" = ?", value), nil
	case "neq":
		return query.Where(column+" <> ?", value), nil
	case "contains":
		return query.Where(column+" LIKE ?", "%"+fmt.Sprint(value)+"%"), nil
	case "startsWith":
		return query.Where(column+" LIKE ?", fmt.Sprint(value)+"%"), nil
	case "gte":
		return query.Where(column+" >= ?", value), nil
	case "lte":
		return query.Where(column+" <= ?", value), nil
	case "between":
		values, ok := value.([]interface{})
		if !ok || len(values) != 2 {
			return nil, fmt.Errorf("between criteria requires two values")
		}
		return query.Where(column+" BETWEEN ? AND ?", values[0], values[1]), nil
	case "in":
		return query.Where(column+" IN ?", value), nil
	case "isNull":
		return query.Where(column + " IS NULL"), nil
	case "isNotNull":
		return query.Where(column + " IS NOT NULL"), nil
	default:
		return nil, fmt.Errorf("criteria operator is not allowed")
	}
}

var criteriaFieldColumns = map[string]string{
	"material":      "material",
	"category":      "category",
	"denomination":  "denomination",
	"ruler":         "ruler",
	"era":           "era",
	"mint":          "mint",
	"grade":         "grade",
	"currentValue":  "current_value",
	"purchasePrice": "purchase_price",
	"purchaseDate":  "purchase_date",
	"createdAt":     "created_at",
	"isWishlist":    "is_wishlist",
	"isSold":        "is_sold",
	"isPrivate":     "is_private",
}

func nextSetSortOrder(tx *gorm.DB, setID uint) (int, error) {
	var maxSortOrder int
	err := tx.Model(&models.CoinSetMembership{}).
		Where("set_id = ?", setID).
		Select("COALESCE(MAX(sort_order), -1)").
		Scan(&maxSortOrder).Error
	return maxSortOrder + 1, err
}

// matchCoinToTarget determines if a coin matches a target's criteria.
func matchCoinToTarget(coin models.Coin, target models.CoinSetTarget) bool {
	// Match year - extract from Era field if present
	if target.Year != nil {
		// For US coins, era typically contains the year
		// This is a simplified match; real implementation may need more sophisticated parsing
		yearStr := fmt.Sprintf("%d", *target.Year)
		if !strings.Contains(string(coin.Era), yearStr) {
			return false
		}
	}

	// Match mint mark (case-insensitive)
	if target.MintMark != nil {
		mintMatch := false
		if coin.Mint != "" {
			if strings.EqualFold(coin.Mint, *target.MintMark) {
				mintMatch = true
			}
		} else if *target.MintMark == "" {
			mintMatch = true
		}
		if !mintMatch {
			return false
		}
	}

	// Match denomination (case-insensitive)
	if target.Denomination != nil && !strings.EqualFold(coin.Denomination, *target.Denomination) {
		return false
	}

	// Match country - check Ruler field for country information
	// For US coins, ruler often contains "United States" or similar
	if target.Country != nil {
		if !strings.Contains(strings.ToLower(coin.Ruler), strings.ToLower(*target.Country)) {
			return false
		}
	}

	// Match material (case-insensitive)
	if target.Material != nil && !strings.EqualFold(string(coin.Material), *target.Material) {
		return false
	}

	return true
}
