package repository

import (
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"gorm.io/gorm"
)

// ApiKeyRepository encapsulates database operations for API key management.
type ApiKeyRepository struct {
	db *gorm.DB
}

// NewApiKeyRepository creates a new ApiKeyRepository.
func NewApiKeyRepository(db *gorm.DB) *ApiKeyRepository {
	return &ApiKeyRepository{db: db}
}

// Create persists a new API key.
func (r *ApiKeyRepository) Create(apiKey *models.ApiKey) error {
	return r.db.Create(apiKey).Error
}

// ListByUser returns all API keys for a user, ordered by creation date descending.
func (r *ApiKeyRepository) ListByUser(userID uint) ([]models.ApiKey, error) {
	var keys []models.ApiKey
	if err := r.db.Scopes(OwnedBy(userID)).Order("created_at DESC").Find(&keys).Error; err != nil {
		return nil, err
	}
	return keys, nil
}

// FindByIDAndUser finds an API key by its ID and owning user.
func (r *ApiKeyRepository) FindByIDAndUser(keyID uint, userID uint) (*models.ApiKey, error) {
	var apiKey models.ApiKey
	if err := r.db.Scopes(OwnedByID(keyID, userID)).First(&apiKey).Error; err != nil {
		return nil, err
	}
	return &apiKey, nil
}

// Revoke sets the revoked_at timestamp on an API key.
func (r *ApiKeyRepository) Revoke(apiKey *models.ApiKey) error {
	now := time.Now()
	return r.db.Model(apiKey).Update("revoked_at", &now).Error
}
