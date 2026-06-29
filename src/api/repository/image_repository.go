package repository

import (
	"github.com/briandenicola/ancient-coins-api/models"
	"gorm.io/gorm"
)

// CoinImageMediaAccess contains the ownership and visibility metadata needed
// to authorize serving an uploaded coin image.
type CoinImageMediaAccess struct {
	FilePath       string
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

// FindCoinImageMediaByPath returns image ownership and visibility metadata for a stored file path.
func (r *ImageRepository) FindCoinImageMediaByPath(filePath string) (*CoinImageMediaAccess, error) {
	var media CoinImageMediaAccess
	tx := r.db.Table("coin_images").
		Select(`coin_images.file_path,
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
