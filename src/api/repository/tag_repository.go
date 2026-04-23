package repository

import (
	"strings"

	"github.com/briandenicola/ancient-coins-api/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// TagRepository encapsulates all tag-related database operations.
type TagRepository struct {
	db *gorm.DB
}

// NewTagRepository creates a new TagRepository.
func NewTagRepository(db *gorm.DB) *TagRepository {
	return &TagRepository{db: db}
}

// List returns all tags belonging to the given user.
func (r *TagRepository) List(userID uint) ([]models.Tag, error) {
	var tags []models.Tag
	err := r.db.Scopes(OwnedBy(userID)).Order("name ASC").Find(&tags).Error
	return tags, err
}

// Create inserts a new tag. Name is trimmed and checked for case-insensitive uniqueness.
func (r *TagRepository) Create(tag *models.Tag) error {
	tag.Name = strings.TrimSpace(tag.Name)
	return r.db.Create(tag).Error
}

// Update modifies a tag's name and/or color.
func (r *TagRepository) Update(tag *models.Tag, updates map[string]interface{}) error {
	if name, ok := updates["name"]; ok {
		updates["name"] = strings.TrimSpace(name.(string))
	}
	return r.db.Model(tag).Updates(updates).Error
}

// GetByID finds a tag by ID and user ID.
func (r *TagRepository) GetByID(id, userID uint) (*models.Tag, error) {
	var tag models.Tag
	err := r.db.Scopes(OwnedByID(id, userID)).First(&tag).Error
	if err != nil {
		return nil, err
	}
	return &tag, nil
}

// Delete removes a tag and its coin associations.
func (r *TagRepository) Delete(id, userID uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Delete associations first
		if err := tx.Where("tag_id = ?", id).Delete(&models.CoinTag{}).Error; err != nil {
			return err
		}
		result := tx.Scopes(OwnedByID(id, userID)).Delete(&models.Tag{})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
}

// AttachToCoin links a tag to a coin. Both must belong to the given user.
// Idempotent — silently ignores if already attached.
func (r *TagRepository) AttachToCoin(coinID, tagID, userID uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Verify coin ownership
		var coinCount int64
		if err := tx.Model(&models.Coin{}).Where("id = ? AND user_id = ?", coinID, userID).Count(&coinCount).Error; err != nil {
			return err
		}
		if coinCount == 0 {
			return gorm.ErrRecordNotFound
		}
		// Verify tag ownership
		var tagCount int64
		if err := tx.Model(&models.Tag{}).Where("id = ? AND user_id = ?", tagID, userID).Count(&tagCount).Error; err != nil {
			return err
		}
		if tagCount == 0 {
			return gorm.ErrRecordNotFound
		}
		// Insert or ignore
		return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&models.CoinTag{
			CoinID: coinID,
			TagID:  tagID,
		}).Error
	})
}

// DetachFromCoin removes a tag from a coin. Both must belong to the given user.
func (r *TagRepository) DetachFromCoin(coinID, tagID, userID uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Verify coin ownership
		var coinCount int64
		if err := tx.Model(&models.Coin{}).Where("id = ? AND user_id = ?", coinID, userID).Count(&coinCount).Error; err != nil {
			return err
		}
		if coinCount == 0 {
			return gorm.ErrRecordNotFound
		}
		return tx.Where("coin_id = ? AND tag_id = ?", coinID, tagID).Delete(&models.CoinTag{}).Error
	})
}

// GetTagsForCoin returns all tags attached to a specific coin.
func (r *TagRepository) GetTagsForCoin(coinID uint) ([]models.Tag, error) {
	var tags []models.Tag
	err := r.db.
		Joins("JOIN coin_tags ON coin_tags.tag_id = tags.id").
		Where("coin_tags.coin_id = ?", coinID).
		Order("tags.name ASC").
		Find(&tags).Error
	return tags, err
}

// CountByUser returns the total number of tags for a user.
func (r *TagRepository) CountByUser(userID uint) (int64, error) {
	var count int64
	err := r.db.Model(&models.Tag{}).Scopes(OwnedBy(userID)).Count(&count).Error
	return count, err
}

// ExistsByName checks if a tag with the given name already exists for the user (case-insensitive).
func (r *TagRepository) ExistsByName(userID uint, name string) (bool, error) {
	var count int64
	err := r.db.Model(&models.Tag{}).
		Where("user_id = ? AND LOWER(name) = LOWER(?)", userID, strings.TrimSpace(name)).
		Count(&count).Error
	return count > 0, err
}

// BulkAttachToCoin attaches a tag to multiple coins. All coins and the tag must belong to the user.
func (r *TagRepository) BulkAttachToCoin(coinIDs []uint, tagID, userID uint) (int64, error) {
	var affected int64
	err := r.db.Transaction(func(tx *gorm.DB) error {
		// Verify tag ownership
		var tagCount int64
		if err := tx.Model(&models.Tag{}).Where("id = ? AND user_id = ?", tagID, userID).Count(&tagCount).Error; err != nil {
			return err
		}
		if tagCount == 0 {
			return gorm.ErrRecordNotFound
		}
		// Verify all coins belong to user
		var coinCount int64
		if err := tx.Model(&models.Coin{}).Where("id IN ? AND user_id = ?", coinIDs, userID).Count(&coinCount).Error; err != nil {
			return err
		}
		if coinCount == 0 {
			return gorm.ErrRecordNotFound
		}
		// Insert associations, ignore duplicates
		for _, coinID := range coinIDs {
			result := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&models.CoinTag{
				CoinID: coinID,
				TagID:  tagID,
			})
			if result.Error != nil {
				return result.Error
			}
			affected += result.RowsAffected
		}
		return nil
	})
	return affected, err
}
