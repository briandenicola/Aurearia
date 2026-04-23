package repository

import (
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"gorm.io/gorm"
)

// AuthRepository encapsulates all auth-related database operations.
type AuthRepository struct {
	db *gorm.DB
}

// NewAuthRepository creates a new AuthRepository.
func NewAuthRepository(db *gorm.DB) *AuthRepository {
	return &AuthRepository{db: db}
}

// CountUsers returns the total number of users.
func (r *AuthRepository) CountUsers() int64 {
	var count int64
	r.db.Model(&models.User{}).Count(&count)
	return count
}

// CreateUser inserts a new user record.
func (r *AuthRepository) CreateUser(user *models.User) error {
	return r.db.Create(user).Error
}

// FindUserByUsername returns a user by username.
func (r *AuthRepository) FindUserByUsername(username string) (*models.User, error) {
	var user models.User
	err := r.db.Where("username = ?", username).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// FindUserByID returns a user by primary key.
func (r *AuthRepository) FindUserByID(id uint) (*models.User, error) {
	var user models.User
	err := r.db.First(&user, id).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// FindRefreshToken returns an active, non-revoked refresh token by hash.
func (r *AuthRepository) FindRefreshToken(tokenHash string) (*models.RefreshToken, error) {
	var rt models.RefreshToken
	err := r.db.Where("token_hash = ? AND revoked_at IS NULL", tokenHash).First(&rt).Error
	if err != nil {
		return nil, err
	}
	return &rt, nil
}

// CreateRefreshTokeninserts a new refresh token record.
func (r *AuthRepository) CreateRefreshToken(rt *models.RefreshToken) error {
	return r.db.Create(rt).Error
}

// RotateRefreshToken revokes the old token and creates a new one in a single
// transaction to prevent token loss or duplication.
func (r *AuthRepository) RotateRefreshToken(old *models.RefreshToken, newToken *models.RefreshToken) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		now := time.Now()
		if err := tx.Model(old).Update("revoked_at", &now).Error; err != nil {
			return err
		}
		return tx.Create(newToken).Error
	})
}
