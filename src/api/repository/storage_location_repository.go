package repository

import (
	"errors"
	"strings"

	"github.com/briandenicola/ancient-coins-api/models"
	"gorm.io/gorm"
)

var ErrStorageLocationNameConflict = errors.New("storage location name conflict")

type TrayOccupancy struct {
	LocationID      uint  `json:"locationId" minimum:"1"`
	Rows            int   `json:"rows" minimum:"1" maximum:"20"`
	Columns         int   `json:"columns" minimum:"1" maximum:"20"`
	Capacity        int   `json:"capacity" minimum:"1" maximum:"400"`
	OccupiedSlots   []int `json:"occupiedSlots" minimum:"1" maximum:"400"`
	CurrentCoinSlot *int  `json:"currentCoinSlot" minimum:"1" maximum:"400" extensions:"x-nullable"`
}

type TrayImage struct {
	FilePath  string `json:"filePath"`
	ImageType string `json:"imageType,omitempty"`
}

type TrayCoin struct {
	ID         uint     `json:"id"`
	Name       string   `json:"name"`
	DiameterMm *float64 `json:"diameterMm"`
	// StorageSlot is the one-based row-major position within the tray.
	StorageSlot int        `json:"storageSlot" minimum:"1" maximum:"400"`
	Image       *TrayImage `json:"image"`
}

type TrayAggregate struct {
	ID       uint       `json:"id"`
	Name     string     `json:"name"`
	Rows     int        `json:"rows" minimum:"1" maximum:"20"`
	Columns  int        `json:"columns" minimum:"1" maximum:"20"`
	Capacity int        `json:"capacity" minimum:"1" maximum:"400"`
	Occupied int        `json:"occupied" minimum:"0" maximum:"400"`
	Coins    []TrayCoin `json:"coins"`
}

// StorageLocationRepository encapsulates all storage-location database operations.
type StorageLocationRepository struct {
	db *gorm.DB
}

// NewStorageLocationRepository creates a new StorageLocationRepository.
func NewStorageLocationRepository(db *gorm.DB) *StorageLocationRepository {
	return &StorageLocationRepository{db: db}
}

func (r *StorageLocationRepository) WithTx(tx *gorm.DB) *StorageLocationRepository {
	return &StorageLocationRepository{db: tx}
}

func (r *StorageLocationRepository) Transaction(fn func(*StorageLocationRepository) error) error {
	return r.db.Transaction(func(tx *gorm.DB) error { return fn(r.WithTx(tx)) })
}

// List returns all storage locations belonging to the given user.
func (r *StorageLocationRepository) List(userID uint) ([]models.StorageLocation, error) {
	var locations []models.StorageLocation
	err := r.db.Model(&models.StorageLocation{}).
		Select(`storage_locations.*,
			(SELECT COUNT(*) FROM coins WHERE coins.user_id = storage_locations.user_id AND coins.storage_location_id = storage_locations.id AND coins.storage_slot IS NOT NULL) AS occupied`).
		Scopes(OwnedBy(userID)).Order("sort_order ASC").Order("name ASC").Find(&locations).Error
	for i := range locations {
		if locations[i].Rows != nil && locations[i].Columns != nil {
			locations[i].Capacity = *locations[i].Rows * *locations[i].Columns
		}
	}
	return locations, err
}

// Create inserts a new storage location.
func (r *StorageLocationRepository) Create(location *models.StorageLocation) error {
	location.Name = strings.TrimSpace(location.Name)
	if err := r.db.Create(location).Error; err != nil {
		if isStorageLocationNameConflict(err) {
			return ErrStorageLocationNameConflict
		}
		return err
	}
	return nil
}

// Update modifies a storage location's editable fields.
func (r *StorageLocationRepository) Update(location *models.StorageLocation, updates map[string]interface{}) error {
	if name, ok := updates["name"]; ok {
		updates["name"] = strings.TrimSpace(name.(string))
	}
	if err := r.db.Model(location).Updates(updates).Error; err != nil {
		if isStorageLocationNameConflict(err) {
			return ErrStorageLocationNameConflict
		}
		return err
	}
	return nil
}

func isStorageLocationNameConflict(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "idx_storage_locations_user_name_nocase") ||
		(strings.Contains(message, "unique constraint failed") &&
			strings.Contains(message, "storage_locations.user_id") &&
			strings.Contains(message, "storage_locations.name"))
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

func (r *StorageLocationRepository) ExistsByNameExcluding(userID uint, name string, excludeID uint) (bool, error) {
	var count int64
	err := r.db.Model(&models.StorageLocation{}).
		Scopes(OwnedBy(userID)).
		Where("id <> ? AND LOWER(name) = LOWER(?)", excludeID, strings.TrimSpace(name)).
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

func (r *StorageLocationRepository) CountOccupied(id, userID uint) (int64, error) {
	var count int64
	err := r.db.Model(&models.Coin{}).Scopes(OwnedBy(userID)).
		Where("storage_location_id = ? AND storage_slot IS NOT NULL", id).Count(&count).Error
	return count, err
}

func (r *StorageLocationRepository) Occupancy(id, userID uint, coinID *uint) (*TrayOccupancy, error) {
	location, err := r.GetByID(id, userID)
	if err != nil {
		return nil, err
	}
	if location.Type != models.StorageLocationTypeTray || location.Rows == nil || location.Columns == nil {
		return nil, gorm.ErrRecordNotFound
	}
	var slots []int
	if err := r.db.Model(&models.Coin{}).Scopes(OwnedBy(userID)).
		Where("storage_location_id = ? AND storage_slot IS NOT NULL", id).
		Order("storage_slot ASC").Pluck("storage_slot", &slots).Error; err != nil {
		return nil, err
	}
	result := &TrayOccupancy{
		LocationID: id, Rows: *location.Rows, Columns: *location.Columns,
		Capacity: *location.Rows * *location.Columns, OccupiedSlots: slots,
	}
	if coinID != nil {
		var coin models.Coin
		if err := r.db.Scopes(OwnedByID(*coinID, userID)).Select("id", "storage_location_id", "storage_slot").First(&coin).Error; err != nil {
			return nil, err
		}
		if coin.StorageLocationID != nil && *coin.StorageLocationID == id {
			result.CurrentCoinSlot = coin.StorageSlot
		}
	}
	return result, nil
}

// ListTrayAggregates returns all owner trays and their positioned minimal coin
// data in a bounded two-query read.
func (r *StorageLocationRepository) ListTrayAggregates(userID uint) ([]TrayAggregate, error) {
	var locations []models.StorageLocation
	if err := r.db.Scopes(OwnedBy(userID)).
		Where("type = ?", models.StorageLocationTypeTray).
		Order("sort_order ASC").Order("name ASC").Find(&locations).Error; err != nil {
		return nil, err
	}
	trays := make([]TrayAggregate, 0, len(locations))
	byID := make(map[uint]int, len(locations))
	for _, location := range locations {
		if location.Rows == nil || location.Columns == nil {
			continue
		}
		byID[location.ID] = len(trays)
		trays = append(trays, TrayAggregate{
			ID: location.ID, Name: location.Name, Rows: *location.Rows,
			Columns: *location.Columns, Capacity: *location.Rows * *location.Columns,
			Coins: []TrayCoin{},
		})
	}
	if len(trays) == 0 {
		return trays, nil
	}
	type row struct {
		ID                uint
		Name              string
		DiameterMm        *float64
		StorageLocationID uint
		StorageSlot       int
		FilePath          *string
		ImageType         *string
	}
	var rows []row
	err := r.db.Table("coins").
		Select(`coins.id, coins.name, coins.diameter_mm, coins.storage_location_id, coins.storage_slot,
			(SELECT file_path FROM coin_images WHERE coin_images.coin_id = coins.id ORDER BY is_primary DESC, id ASC LIMIT 1) AS file_path,
			(SELECT image_type FROM coin_images WHERE coin_images.coin_id = coins.id ORDER BY is_primary DESC, id ASC LIMIT 1) AS image_type`).
		Scopes(OwnedBy(userID)).
		Where("coins.storage_location_id IN ? AND coins.storage_slot IS NOT NULL", keys(byID)).
		Order("coins.storage_location_id ASC, coins.storage_slot ASC").Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, item := range rows {
		index, ok := byID[item.StorageLocationID]
		if !ok {
			continue
		}
		coin := TrayCoin{ID: item.ID, Name: item.Name, DiameterMm: item.DiameterMm, StorageSlot: item.StorageSlot}
		if item.FilePath != nil {
			coin.Image = &TrayImage{FilePath: *item.FilePath}
			if item.ImageType != nil {
				coin.Image.ImageType = *item.ImageType
			}
		}
		trays[index].Coins = append(trays[index].Coins, coin)
		trays[index].Occupied++
	}
	return trays, nil
}

func keys(values map[uint]int) []uint {
	result := make([]uint, 0, len(values))
	for id := range values {
		result = append(result, id)
	}
	return result
}
