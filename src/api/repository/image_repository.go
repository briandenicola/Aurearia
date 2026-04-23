package repository

import (
	"github.com/briandenicola/ancient-coins-api/models"
	"gorm.io/gorm"
)

// ImageRepository encapsulates all image-related database operations.
type ImageRepository struct {
	db *gorm.DB
}

// NewImageRepository creates a new ImageRepository.
func NewImageRepository(db *gorm.DB) *ImageRepository {
	return &ImageRepository{db: db}
}

// FindCoinByOwner returns a coin if it belongs to the given user.
func (r *ImageRepository) FindCoinByOwner(coinID, userID uint) (*models.Coin, error) {
	var coin models.Coin
	err := r.db.Where("id = ? AND user_id = ?", coinID, userID).First(&coin).Error
	if err != nil {
		return nil, err
	}
	return &coin, nil
}

// CreateImage inserts a new image record.
func (r *ImageRepository) CreateImage(image *models.CoinImage) error {
	return r.db.Create(image).Error
}

// FindImage returns an image by ID and coin ID.
func (r *ImageRepository) FindImage(imageID, coinID uint) (*models.CoinImage, error) {
	var image models.CoinImage
	err := r.db.Where("id = ? AND coin_id = ?", imageID, coinID).First(&image).Error
	if err != nil {
		return nil, err
	}
	return &image, nil
}

// DeleteImage removes an image record.
func (r *ImageRepository) DeleteImage(image *models.CoinImage) error {
	return r.db.Delete(image).Error
}

// SetPrimaryAndCreate clears the primary flag on existing images and creates
// a new image record in a single transaction.
func (r *ImageRepository) SetPrimaryAndCreate(coinID uint, image *models.CoinImage) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.CoinImage{}).Where("coin_id = ?", coinID).Update("is_primary", false).Error; err != nil {
			return err
		}
		return tx.Create(image).Error
	})
}
