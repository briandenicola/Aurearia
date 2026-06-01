package repository

import (
	"strings"

	"github.com/briandenicola/ancient-coins-api/models"
	"gorm.io/gorm"
)

// StorageLocationRepository encapsulates all storage-location database operations.
type StorageLocationRepository struct {
	db *gorm.DB
}

// NewStorageLocationRepository creates a new StorageLocationRepository.
func NewStorageLocationRepository(db *gorm.DB) *StorageLocationRepository {
	return &StorageLocationRepository{db: db}
}

// List returns all storage locations belonging to the given user.
func (r *StorageLocationRepository) List(userID uint) ([]models.StorageLocation, error) {
	var locations []models.StorageLocation
	err := r.db.Scopes(OwnedBy(userID)).Order("sort_order ASC").Order("name ASC").Find(&locations).Error
	return locations, err
}

// Create inserts a new storage location.
func (r *StorageLocationRepository) Create(location *models.StorageLocation) error {
	location.Name = strings.TrimSpace(location.Name)
	return r.db.Create(location).Error
}

// Update modifies a storage location's editable fields.
func (r *StorageLocationRepository) Update(location *models.StorageLocation, updates map[string]interface{}) error {
	if name, ok := updates["name"]; ok {
		updates["name"] = strings.TrimSpace(name.(string))
	}
	return r.db.Model(location).Updates(updates).Error
}

// GetByID finds a storage location by ID and user ID.
func (r *StorageLocationRepository) GetByID(id, userID uint) (*models.StorageLocation, error) {
	var location models.StorageLocation
	err := r.db.Scopes(OwnedByID(id, userID)).First(&location).Error
	if err != nil {
		return nil, err
	}
	return &location, nil
}

// ExistsByID checks whether a storage location exists for the given user.
func (r *StorageLocationRepository) ExistsByID(id, userID uint) (bool, error) {
	var count int64
	err := r.db.Model(&models.StorageLocation{}).Scopes(OwnedByID(id, userID)).Count(&count).Error
	return count > 0, err
}

// Delete removes a storage location owned by the user.
func (r *StorageLocationRepository) Delete(id, userID uint) error {
	result := r.db.Scopes(OwnedByID(id, userID)).Delete(&models.StorageLocation{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// CountByUser returns the total number of storage locations for a user.
func (r *StorageLocationRepository) CountByUser(userID uint) (int64, error) {
	var count int64
	err := r.db.Model(&models.StorageLocation{}).Scopes(OwnedBy(userID)).Count(&count).Error
	return count, err
}

// ExistsByName checks if a storage location with the given name already exists for the user (case-insensitive).
func (r *StorageLocationRepository) ExistsByName(userID uint, name string) (bool, error) {
	var count int64
	err := r.db.Model(&models.StorageLocation{}).
		Where("user_id = ? AND LOWER(name) = LOWER(?)", userID, strings.TrimSpace(name)).
		Count(&count).Error
	return count > 0, err
}

// CountCoinsUsing returns how many coins owned by the user reference a storage location.
func (r *StorageLocationRepository) CountCoinsUsing(id, userID uint) (int64, error) {
	var count int64
	err := r.db.Model(&models.Coin{}).
		Scopes(OwnedBy(userID)).
		Where("storage_location_id = ?", id).
		Count(&count).Error
	return count, err
}
