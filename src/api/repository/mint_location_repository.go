package repository

import (
	"github.com/briandenicola/ancient-coins-api/models"
	"gorm.io/gorm"
)

// MintLocationRepository encapsulates global mint-location database operations.
type MintLocationRepository struct {
	db *gorm.DB
}

// NewMintLocationRepository creates a new MintLocationRepository.
func NewMintLocationRepository(db *gorm.DB) *MintLocationRepository {
	return &MintLocationRepository{db: db}
}

// List returns all mint locations ordered by display name.
func (r *MintLocationRepository) List() ([]models.MintLocation, error) {
	var locations []models.MintLocation
	err := r.db.Order("display_name ASC").Find(&locations).Error
	return locations, err
}

// Create inserts a new mint location.
func (r *MintLocationRepository) Create(location *models.MintLocation) error {
	return r.db.Create(location).Error
}

// Update modifies a mint location's editable fields.
func (r *MintLocationRepository) Update(location *models.MintLocation, updates map[string]interface{}) error {
	return r.db.Model(location).Updates(updates).Error
}

// FindByID returns a mint location by ID.
func (r *MintLocationRepository) FindByID(id uint) (*models.MintLocation, error) {
	var location models.MintLocation
	if err := r.db.First(&location, id).Error; err != nil {
		return nil, err
	}
	return &location, nil
}

// FindByNormalizedName returns a mint location by normalized display name.
func (r *MintLocationRepository) FindByNormalizedName(normalizedName string) (*models.MintLocation, error) {
	var location models.MintLocation
	if err := r.db.Where("normalized_name = ?", normalizedName).First(&location).Error; err != nil {
		return nil, err
	}
	return &location, nil
}

// Delete removes a mint location by ID.
func (r *MintLocationRepository) Delete(id uint) error {
	result := r.db.Delete(&models.MintLocation{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
