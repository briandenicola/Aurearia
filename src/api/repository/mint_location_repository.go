package repository

import (
	"github.com/briandenicola/ancient-coins-api/models"
	"gorm.io/gorm"
)

// MintLocationRepository encapsulates mint-location database operations,
// covering both global (admin-curated, UserID nil) and private (per-user)
// entries.
type MintLocationRepository struct {
	db *gorm.DB
}

// NewMintLocationRepository creates a new MintLocationRepository.
func NewMintLocationRepository(db *gorm.DB) *MintLocationRepository {
	return &MintLocationRepository{db: db}
}

// ListGlobal returns all global mint locations ordered by display name.
func (r *MintLocationRepository) ListGlobal() ([]models.MintLocation, error) {
	var locations []models.MintLocation
	err := r.db.Where("user_id IS NULL").Order("display_name ASC").Find(&locations).Error
	return locations, err
}

// ListVisibleTo returns global mint locations plus the given user's own
// private ones - the set a user may pick from or see on their mint map.
func (r *MintLocationRepository) ListVisibleTo(userID uint) ([]models.MintLocation, error) {
	var locations []models.MintLocation
	err := r.db.Where("user_id IS NULL OR user_id = ?", userID).Order("display_name ASC").Find(&locations).Error
	return locations, err
}

// Create inserts a new mint location - global if UserID is nil, private otherwise.
func (r *MintLocationRepository) Create(location *models.MintLocation) error {
	return r.db.Create(location).Error
}

// Update modifies a mint location's editable fields.
func (r *MintLocationRepository) Update(location *models.MintLocation, updates map[string]interface{}) error {
	return r.db.Model(location).Updates(updates).Error
}

// FindByID returns a mint location by ID regardless of ownership. Callers
// must apply their own ownership/scope check before trusting the result.
func (r *MintLocationRepository) FindByID(id uint) (*models.MintLocation, error) {
	var location models.MintLocation
	if err := r.db.First(&location, id).Error; err != nil {
		return nil, err
	}
	return &location, nil
}

// FindOwnedByID returns a private mint location owned by the given user.
// Never returns a global entry, even if the ID matches one.
func (r *MintLocationRepository) FindOwnedByID(id, userID uint) (*models.MintLocation, error) {
	var location models.MintLocation
	if err := r.db.Where("id = ? AND user_id = ?", id, userID).First(&location).Error; err != nil {
		return nil, err
	}
	return &location, nil
}

// FindByNormalizedName returns a global mint location by normalized display name.
func (r *MintLocationRepository) FindByNormalizedName(normalizedName string) (*models.MintLocation, error) {
	var location models.MintLocation
	if err := r.db.Where("user_id IS NULL AND normalized_name = ?", normalizedName).First(&location).Error; err != nil {
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

// ExistsVisibleTo reports whether a mint location exists and is visible to
// userID - i.e. it's global, or it's a private entry owned by that user.
func (r *MintLocationRepository) ExistsVisibleTo(id, userID uint) (bool, error) {
	var count int64
	err := r.db.Model(&models.MintLocation{}).
		Where("id = ? AND (user_id IS NULL OR user_id = ?)", id, userID).
		Count(&count).Error
	return count > 0, err
}

// CountCoinsUsing returns how many coins reference this mint location.
func (r *MintLocationRepository) CountCoinsUsing(id uint) (int64, error) {
	var count int64
	err := r.db.Model(&models.Coin{}).Where("mint_location_id = ?", id).Count(&count).Error
	return count, err
}
