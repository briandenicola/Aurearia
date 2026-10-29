package repository

import (
	"errors"
	"github.com/briandenicola/ancient-coins-api/models"
	"gorm.io/gorm"
)

// CoinImageMediaAccess contains the ownership and visibility metadata needed
// to authorize serving an uploaded coin image.
type CoinImageMediaAccess struct {
	FilePath       string
	ImageType      models.ImageType
	CoinUserID     uint
	CoinIsPrivate  bool
	CoinIsWishlist bool
	CoinIsSold     bool
	OwnerIsPublic  bool
}

type DraftImageMediaAccess struct {
	FilePath string
	UserID   uint
}

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

func (r *ImageRepository) ListAttributionCards(coinID uint) ([]models.CoinImage, error) {
	cards := make([]models.CoinImage, 0)
	err := r.db.Where("coin_id = ? AND image_type = ?", coinID, models.ImageTypeCard).
		Order("id ASC").Find(&cards).Error
	return cards, err
}

// FindCoinImageByType returns the most recently uploaded image of the given
// type for a coin (used by deep identification to resolve an obverse/reverse
// role from a saved coin when no new upload is provided for that role).
func (r *ImageRepository) FindCoinImageByType(coinID uint, imageType models.ImageType) (*models.CoinImage, error) {
	var image models.CoinImage
	err := r.db.Where("coin_id = ? AND image_type = ?", coinID, imageType).
		Order("id DESC").
		First(&image).Error
	if err != nil {
		return nil, err
	}
	return &image, nil
}

// ListOwnedCoinPhotos returns ordinary photos (never attribution cards) for
// the given coins, restricted to coins owned by userID. Rows are ordered per
// coin with the primary image first, then obverse, reverse, and the rest.
func (r *ImageRepository) ListOwnedCoinPhotos(userID uint, coinIDs []uint) ([]models.CoinImage, error) {
	images := make([]models.CoinImage, 0)
	if len(coinIDs) == 0 {
		return images, nil
	}
	err := r.db.Model(&models.CoinImage{}).
		Joins("JOIN coins ON coins.id = coin_images.coin_id").
		Where("coins.user_id = ? AND coin_images.coin_id IN ? AND coin_images.image_type <> ?",
			userID, coinIDs, models.ImageTypeCard).
		Order("coin_images.coin_id ASC").
		Order("coin_images.is_primary DESC").
		Order(`CASE coin_images.image_type
			WHEN 'obverse' THEN 0
			WHEN 'reverse' THEN 1
			ELSE 2
		END`).
		Order("coin_images.id ASC").
		Find(&images).Error
	return images, err
}

// FindCoinImageMediaByPath returns image ownership and visibility metadata for a stored file path.
func (r *ImageRepository) FindCoinImageMediaByPath(filePath string) (*CoinImageMediaAccess, error) {
	var media CoinImageMediaAccess
	tx := r.db.Table("coin_images").
		Select(`coin_images.file_path,
			coin_images.image_type,
			coins.user_id AS coin_user_id,
			coins.is_private AS coin_is_private,
			coins.is_wishlist AS coin_is_wishlist,
			coins.is_sold AS coin_is_sold,
			users.is_public AS owner_is_public`).
		Joins("JOIN coins ON coins.id = coin_images.coin_id").
		Joins("JOIN users ON users.id = coins.user_id").
		Where("coin_images.file_path = ?", filePath).
		Limit(1).
		Scan(&media)
	if tx.Error != nil {
		return nil, tx.Error
	}
	if tx.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &media, nil
}

// FindAvatarOwnerByPath returns the user whose avatar_path matches filePath.
func (r *ImageRepository) FindAvatarOwnerByPath(filePath string) (*models.User, error) {
	var user models.User
	err := r.db.Where("avatar_path = ?", filePath).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *ImageRepository) FindDraftImageMediaByPath(filePath string) (*DraftImageMediaAccess, error) {
	var media DraftImageMediaAccess
	tx := r.db.Table("quick_capture_draft_images").
		Select("file_path, user_id").
		Where("file_path = ?", filePath).
		Limit(1).
		Scan(&media)
	if tx.Error != nil {
		return nil, tx.Error
	}
	if tx.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &media, nil
}

// IsAcceptedFollower checks whether followerID is accepted to view followingID's collection.
func (r *ImageRepository) IsAcceptedFollower(followerID, followingID uint) bool {
	var count int64
	r.db.Model(&models.Follow{}).
		Where("follower_id = ? AND following_id = ? AND status = ?", followerID, followingID, "accepted").
		Count(&count)
	return count > 0
}

// CoinImagePathInActiveShowcase checks whether filePath belongs to a coin in an active public showcase.
func (r *ImageRepository) CoinImagePathInActiveShowcase(slug, filePath string) (bool, error) {
	var count int64
	err := r.db.Table("coin_images").
		Joins("JOIN showcase_coins ON showcase_coins.coin_id = coin_images.coin_id").
		Joins("JOIN showcases ON showcases.id = showcase_coins.showcase_id").
		Joins("JOIN coins ON coins.id = coin_images.coin_id").
		Where("showcases.slug = ? AND showcases.is_active = ? AND showcases.user_id = coins.user_id AND coin_images.file_path = ?", slug, true, filePath).
		Where("coin_images.image_type IS NULL OR coin_images.image_type <> ?", models.ImageTypeCard).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// DeleteImage removes an image record.
func (r *ImageRepository) DeleteImage(image *models.CoinImage) error {
	return r.db.Delete(image).Error
}

func (r *ImageRepository) DeleteAndQueueCleanup(imageID, coinID, userID uint) (*models.ImageCleanup, error) {
	var cleanup models.ImageCleanup
	err := r.db.Transaction(func(tx *gorm.DB) error {
		err := tx.Where("image_id = ? AND coin_id = ? AND user_id = ?", imageID, coinID, userID).First(&cleanup).Error
		if err == nil {
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		var image models.CoinImage
		if err := tx.Where("id = ? AND coin_id = ?", imageID, coinID).First(&image).Error; err != nil {
			return err
		}
		cleanup = models.ImageCleanup{ImageID: image.ID, CoinID: coinID, UserID: userID, FilePath: image.FilePath}
		if err := tx.Create(&cleanup).Error; err != nil {
			return err
		}
		return tx.Delete(&image).Error
	})
	return &cleanup, err
}

func (r *ImageRepository) PendingCleanups(after uint) ([]models.ImageCleanup, error) {
	var cleanups []models.ImageCleanup
	err := r.db.Where("image_id > ?", after).Order("image_id").Limit(100).Find(&cleanups).Error
	return cleanups, err
}

func (r *ImageRepository) CompleteCleanup(imageID uint) error {
	return r.db.Where("image_id = ?", imageID).Delete(&models.ImageCleanup{}).Error
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
