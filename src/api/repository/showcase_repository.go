package repository

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/briandenicola/ancient-coins-api/models"
	"gorm.io/gorm"
)

type ShowcaseRepository struct {
	db *gorm.DB
}

func NewShowcaseRepository(db *gorm.DB) *ShowcaseRepository {
	return &ShowcaseRepository{db: db}
}

func (r *ShowcaseRepository) Create(showcase *models.Showcase) error {
	if showcase.Slug == "" {
		showcase.Slug = generateSlug(showcase.Title)
	}
	return r.db.Create(showcase).Error
}

func (r *ShowcaseRepository) Update(showcase *models.Showcase) error {
	return r.db.Save(showcase).Error
}

func (r *ShowcaseRepository) Delete(id uint, userID uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("showcase_id IN (SELECT id FROM showcases WHERE id = ? AND user_id = ?)", id, userID).Delete(&models.ShowcaseCoin{}).Error; err != nil {
			return err
		}
		return tx.Where("id = ? AND user_id = ?", id, userID).Delete(&models.Showcase{}).Error
	})
}

func (r *ShowcaseRepository) GetByID(id uint, userID uint) (*models.Showcase, error) {
	var showcase models.Showcase
	err := r.db.Where("id = ? AND user_id = ?", id, userID).First(&showcase).Error
	return &showcase, err
}

func (r *ShowcaseRepository) ListByUser(userID uint) ([]models.Showcase, error) {
	var showcases []models.Showcase
	err := r.db.Where("user_id = ?", userID).Order("created_at DESC").Find(&showcases).Error
	return showcases, err
}

// GetBySlug returns a public showcase by slug. Does not require auth.
func (r *ShowcaseRepository) GetBySlug(slug string) (*models.Showcase, error) {
	var showcase models.Showcase
	err := r.db.Where("slug = ? AND is_active = ?", slug, true).First(&showcase).Error
	return &showcase, err
}

// GetShowcaseCoins returns coins in a showcase with images, ordered by sort_order.
func (r *ShowcaseRepository) GetShowcaseCoins(showcaseID uint) ([]models.Coin, error) {
	var coins []models.Coin
	err := r.db.
		Joins("JOIN showcase_coins ON showcase_coins.coin_id = coins.id").
		Where("showcase_coins.showcase_id = ?", showcaseID).
		Preload("Images").
		Preload("Tags").
		Order("showcase_coins.sort_order ASC").
		Find(&coins).Error
	return coins, err
}

func (r *ShowcaseRepository) SetCoins(showcaseID uint, userID uint, coinIDs []uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Verify showcase belongs to user
		var count int64
		tx.Model(&models.Showcase{}).Where("id = ? AND user_id = ?", showcaseID, userID).Count(&count)
		if count == 0 {
			return fmt.Errorf("showcase not found")
		}

		// Verify all coins belong to user
		if len(coinIDs) > 0 {
			var coinCount int64
			tx.Model(&models.Coin{}).Where("id IN ? AND user_id = ?", coinIDs, userID).Count(&coinCount)
			if int(coinCount) != len(coinIDs) {
				return fmt.Errorf("one or more coins not found")
			}
		}

		// Remove existing coins
		if err := tx.Where("showcase_id = ?", showcaseID).Delete(&models.ShowcaseCoin{}).Error; err != nil {
			return err
		}

		// Add new coins with sort order
		for i, coinID := range coinIDs {
			sc := models.ShowcaseCoin{
				ShowcaseID: showcaseID,
				CoinID:     coinID,
				SortOrder:  i,
			}
			if err := tx.Create(&sc).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *ShowcaseRepository) GetShowcaseCoinEntries(showcaseID uint) ([]models.ShowcaseCoin, error) {
	var entries []models.ShowcaseCoin
	err := r.db.Where("showcase_id = ?", showcaseID).Order("sort_order ASC").Find(&entries).Error
	return entries, err
}

// GetOwnerUsername returns the username of the showcase owner.
func (r *ShowcaseRepository) GetOwnerUsername(userID uint) (string, error) {
	var user models.User
	err := r.db.Select("username").Where("id = ?", userID).First(&user).Error
	return user.Username, err
}

func generateSlug(title string) string {
	slug := strings.ToLower(title)
	slug = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			return r
		}
		if r == ' ' || r == '-' {
			return '-'
		}
		return -1
	}, slug)
	// Remove consecutive dashes
	for strings.Contains(slug, "--") {
		slug = strings.ReplaceAll(slug, "--", "-")
	}
	slug = strings.Trim(slug, "-")
	if slug == "" {
		slug = "showcase"
	}
	// Append random suffix for uniqueness
	b := make([]byte, 4)
	rand.Read(b)
	return slug + "-" + hex.EncodeToString(b)
}
