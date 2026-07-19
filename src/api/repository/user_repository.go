package repository

import (
	"github.com/briandenicola/ancient-coins-api/models"
	"gorm.io/gorm"
)

// UserRepository encapsulates all user-related database operations.
type UserRepository struct {
	db *gorm.DB
}

// NewUserRepository creates a new UserRepository.
func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

// FindByID returns a user by primary key.
func (r *UserRepository) FindByID(id uint) (*models.User, error) {
	var user models.User
	err := r.db.First(&user, id).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// FindByEmail returns a user with the given email, excluding the specified user ID.
func (r *UserRepository) FindByEmail(email string, excludeID uint) (*models.User, error) {
	var user models.User
	err := r.db.Where("email = ? AND id != ?", email, excludeID).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// UpdateField updates a single field on a user.
func (r *UserRepository) UpdateField(user *models.User, field string, value interface{}) error {
	return r.db.Model(user).Update(field, value).Error
}

// UpdateFields updates multiple fields on a user using a map.
func (r *UserRepository) UpdateFields(user *models.User, updates map[string]interface{}) error {
	return r.db.Model(user).Updates(updates).Error
}

// Reload re-fetches a user from the database.
func (r *UserRepository) Reload(user *models.User) error {
	return r.db.First(user, user.ID).Error
}

// GetCoinsWithImages returns all coins for a user with export-related relations preloaded.
func (r *UserRepository) GetCoinsWithImages(userID uint) ([]models.Coin, error) {
	var coins []models.Coin
	err := r.db.Where("user_id = ?", userID).Preload("Images").Preload("References").Preload("StorageLocation").Find(&coins).Error
	return coins, err
}

// CreateCoin inserts a new coin record.
func (r *UserRepository) CreateCoin(coin *models.Coin) error {
	return r.db.Create(coin).Error
}

// UpdateProfileWithPrivacyupdates user profile fields and, if the user is
// going private, removes all followers in a single transaction.
func (r *UserRepository) UpdateProfileWithPrivacy(user *models.User, updates map[string]interface{}, goingPrivate bool) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if goingPrivate {
			if err := tx.Where("following_id = ?", user.ID).Delete(&models.Follow{}).Error; err != nil {
				return err
			}
		}
		return tx.Model(user).Updates(updates).Error
	})
}

// ListCoinOfDayEnabled returns all users who have opted in to the Coin of the Day feature.
func (r *UserRepository) ListCoinOfDayEnabled() ([]models.User, error) {
	var users []models.User
	err := r.db.Where("coin_of_day_enabled = ?", true).Find(&users).Error
	return users, err
}

// ListUsersWithAuctionCredentials returns every user with at least one auction provider
// (NumisBids or CNG) configured, for background watchlist sync. This is intentionally NOT
// gated on Pushover/notification preferences (see specs/_backlog/F026) — sync should keep a
// user's CurrentBid/status fresh regardless of whether they've opted into push notifications;
// Pushover configuration should only ever gate whether a *notification* is sent, checked
// separately at the point each notification is sent (see notificationUser in
// auction_alert_service.go and the equivalent checks in auction_ending_scheduler.go /
// auction_watch_bid_digest_scheduler.go).
func (r *UserRepository) ListUsersWithAuctionCredentials() ([]models.User, error) {
	var users []models.User
	err := r.db.
		Where("(numis_bids_username <> '' AND numis_bids_password <> '') OR (cng_username <> '' AND cng_password <> '')").
		Find(&users).Error
	return users, err
}
