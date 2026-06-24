package repository

import (
	"errors"

	"github.com/briandenicola/ancient-coins-api/models"
	"gorm.io/gorm"
)

var ErrFinalLocalAdminBlocked = errors.New("final local admin recovery account cannot be removed")

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

func (r *AdminRepository) GetUserByID(userID uint) (*models.User, error) {
	var user models.User
	err := r.db.First(&user, userID).Error
	return &user, err
}

func (r *AdminRepository) CountLocalRecoveryAdmins() (int64, error) {
	return countLocalRecoveryAdmins(r.db)
}

// DeleteUserCascade deletes a user and all associated data in a transaction.
func (r *AdminRepository) DeleteUserCascade(userID uint) (int64, error) {
	var rowsAffected int64
	err := r.db.Transaction(func(tx *gorm.DB) error {
		result, err := deleteUserCascadeInTx(tx, userID)
		rowsAffected = result.RowsAffected
		if err != nil {
			return err
		}
		return result.Error
	})
	return rowsAffected, err
}

func (r *AdminRepository) DeleteUserCascadeWithRecoveryGuard(userID uint) (int64, *models.User, error) {
	var rowsAffected int64
	var blockedUser *models.User
	err := r.db.Transaction(func(tx *gorm.DB) error {
		var user models.User
		if err := tx.First(&user, userID).Error; err != nil {
			return err
		}
		if userHasUsableLocalAdminCredentials(user) {
			count, err := countLocalRecoveryAdmins(tx)
			if err != nil {
				return err
			}
			if count <= 1 {
				copyUser := user
				blockedUser = &copyUser
				return ErrFinalLocalAdminBlocked
			}
		}
		result, err := deleteUserCascadeInTx(tx, userID)
		rowsAffected = result.RowsAffected
		if err != nil {
			return err
		}
		return result.Error
	})
	return rowsAffected, blockedUser, err
}

// ResetPassword updates a user's password hash. Returns rows affected.
func (r *AdminRepository) ResetPassword(userID uint, passwordHash string) (int64, error) {
	result := r.db.Model(&models.User{}).Where("id = ?", userID).Update("password_hash", passwordHash)
	return result.RowsAffected, result.Error
}

// UpdateUserRole updates a user's role. Returns rows affected.
func (r *AdminRepository) UpdateUserRole(userID uint, role models.UserRole) (int64, error) {
	result := r.db.Model(&models.User{}).Where("id = ?", userID).Update("role", role)
	return result.RowsAffected, result.Error
}

func (r *AdminRepository) UpdateUserRoleWithRecoveryGuard(userID uint, role models.UserRole) (int64, *models.User, error) {
	var rowsAffected int64
	var blockedUser *models.User
	err := r.db.Transaction(func(tx *gorm.DB) error {
		var user models.User
		if err := tx.First(&user, userID).Error; err != nil {
			return err
		}
		if role != models.RoleAdmin && userHasUsableLocalAdminCredentials(user) {
			count, err := countLocalRecoveryAdmins(tx)
			if err != nil {
				return err
			}
			if count <= 1 {
				copyUser := user
				blockedUser = &copyUser
				return ErrFinalLocalAdminBlocked
			}
		}
		result := tx.Model(&models.User{}).Where("id = ?", userID).Update("role", role)
		rowsAffected = result.RowsAffected
		return result.Error
	})
	return rowsAffected, blockedUser, err
}

func countLocalRecoveryAdmins(db *gorm.DB) (int64, error) {
	var count int64
	err := db.Model(&models.User{}).
		Where("role = ? AND password_hash <> ?", models.RoleAdmin, "").
		Count(&count).Error
	return count, err
}

func userHasUsableLocalAdminCredentials(user models.User) bool {
	return user.Role == models.RoleAdmin && user.PasswordHash != ""
}

func deleteUserCascadeInTx(tx *gorm.DB, userID uint) (*gorm.DB, error) {
	var coinIDs []uint
	if err := tx.Model(&models.Coin{}).Where("user_id = ?", userID).Pluck("id", &coinIDs).Error; err != nil {
		return nil, err
	}
	if len(coinIDs) > 0 {
		if err := tx.Where("coin_id IN ?", coinIDs).Delete(&models.CoinImage{}).Error; err != nil {
			return nil, err
		}
		if err := tx.Where("coin_id IN ?", coinIDs).Delete(&models.CoinJournal{}).Error; err != nil {
			return nil, err
		}
		if err := tx.Where("coin_id IN ?", coinIDs).Delete(&models.CoinValueHistory{}).Error; err != nil {
			return nil, err
		}
		if err := tx.Where("coin_id IN ?", coinIDs).Delete(&models.CoinComment{}).Error; err != nil {
			return nil, err
		}
	}
	if err := tx.Where("user_id = ?", userID).Delete(&models.CoinComment{}).Error; err != nil {
		return nil, err
	}
	if err := tx.Where("user_id = ?", userID).Delete(&models.Coin{}).Error; err != nil {
		return nil, err
	}
	if err := tx.Where("user_id = ?", userID).Delete(&models.AgentConversation{}).Error; err != nil {
		return nil, err
	}
	if err := tx.Where("user_id = ?", userID).Delete(&models.ValueSnapshot{}).Error; err != nil {
		return nil, err
	}
	if err := tx.Where("user_id = ?", userID).Delete(&models.ApiKey{}).Error; err != nil {
		return nil, err
	}
	if err := tx.Where("user_id = ?", userID).Delete(&models.RefreshToken{}).Error; err != nil {
		return nil, err
	}
	if err := tx.Where("user_id = ?", userID).Delete(&models.WebAuthnCredential{}).Error; err != nil {
		return nil, err
	}
	if tx.Migrator().HasTable(&models.ExternalIdentity{}) {
		if err := tx.Where("user_id = ?", userID).Delete(&models.ExternalIdentity{}).Error; err != nil {
			return nil, err
		}
	}
	if tx.Migrator().HasTable(&models.OIDCAuthState{}) {
		if err := tx.Where("user_id = ?", userID).Delete(&models.OIDCAuthState{}).Error; err != nil {
			return nil, err
		}
	}
	if err := tx.Where("follower_id = ? OR following_id = ?", userID, userID).Delete(&models.Follow{}).Error; err != nil {
		return nil, err
	}
	return tx.Delete(&models.User{}, userID), nil
}
