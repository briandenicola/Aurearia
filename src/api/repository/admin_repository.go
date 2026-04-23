package repository

import (
	"github.com/briandenicola/ancient-coins-api/models"
	"gorm.io/gorm"
)

// AdminRepository encapsulates database operations for the admin handler.
type AdminRepository struct {
	db *gorm.DB
}

// NewAdminRepository creates a new AdminRepository.
func NewAdminRepository(db *gorm.DB) *AdminRepository {
	return &AdminRepository{db: db}
}

// ListUsers returns all users.
func (r *AdminRepository) ListUsers() ([]models.User, error) {
	var users []models.User
	if err := r.db.Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

// DeleteUserCascade deletes a user and all associated data in a transaction.
func (r *AdminRepository) DeleteUserCascade(userID uint) (int64, error) {
	var rowsAffected int64
	err := r.db.Transaction(func(tx *gorm.DB) error {
		var coinIDs []uint
		if err := tx.Model(&models.Coin{}).Where("user_id = ?", userID).Pluck("id", &coinIDs).Error; err != nil {
			return err
		}
		if len(coinIDs) > 0 {
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
		}
		// Also delete comments the user made on other users' coins
		if err := tx.Where("user_id = ?", userID).Delete(&models.CoinComment{}).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ?", userID).Delete(&models.Coin{}).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ?", userID).Delete(&models.AgentConversation{}).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ?", userID).Delete(&models.ValueSnapshot{}).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ?", userID).Delete(&models.ApiKey{}).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ?", userID).Delete(&models.RefreshToken{}).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ?", userID).Delete(&models.WebAuthnCredential{}).Error; err != nil {
			return err
		}
		if err := tx.Where("follower_id = ? OR following_id = ?", userID, userID).Delete(&models.Follow{}).Error; err != nil {
			return err
		}

		result := tx.Delete(&models.User{}, userID)
		rowsAffected = result.RowsAffected
		return result.Error
	})
	return rowsAffected, err
}

// ResetPassword updates a user's password hash. Returns rows affected.
func (r *AdminRepository) ResetPassword(userID uint, passwordHash string) (int64, error) {
	result := r.db.Model(&models.User{}).Where("id = ?", userID).Update("password_hash", passwordHash)
	return result.RowsAffected, result.Error
}
