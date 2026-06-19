package services

import (
	"errors"
	"strings"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
)

const maxStorageLocationsPerUser = 100
const maxStorageLocationNameLength = 100

var (
	ErrStorageLocationNotFound    = errors.New("storage location not found")
	ErrStorageLocationNameInvalid = errors.New("storage location name must be 1-100 characters")
	ErrStorageLocationDuplicate   = errors.New("a storage location with this name already exists")
	ErrStorageLocationInUse       = errors.New("storage location is in use")
	ErrStorageLocationLimit       = errors.New("maximum of 100 storage locations allowed")
)

// StorageLocationService handles storage-location business rules.
type StorageLocationService struct {
	repo *repository.StorageLocationRepository
}

// NewStorageLocationService creates a new StorageLocationService.
func NewStorageLocationService(repo *repository.StorageLocationRepository) *StorageLocationService {
	return &StorageLocationService{repo: repo}
}

// List returns all storage locations for a user.
func (s *StorageLocationService) List(userID uint) ([]models.StorageLocation, error) {
	return s.repo.List(userID)
}

// Create validates and creates a storage location.
func (s *StorageLocationService) Create(userID uint, name string, sortOrder int) (*models.StorageLocation, error) {
	name = strings.TrimSpace(name)
	if name == "" || len(name) > maxStorageLocationNameLength {
		return nil, ErrStorageLocationNameInvalid
	}
	count, err := s.repo.CountByUser(userID)
	if err != nil {
		return nil, err
	}
	if count >= maxStorageLocationsPerUser {
		return nil, ErrStorageLocationLimit
	}
	exists, err := s.repo.ExistsByName(userID, name)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrStorageLocationDuplicate
	}
	location := &models.StorageLocation{UserID: userID, Name: name, SortOrder: sortOrder}
	if err := s.repo.Create(location); err != nil {
		return nil, err
	}
	return location, nil
}

// Update validates and updates a storage location.
func (s *StorageLocationService) Update(id, userID uint, name *string, sortOrder *int) (*models.StorageLocation, error) {
	location, err := s.repo.GetByID(id, userID)
	if err != nil {
		if repository.IsRecordNotFound(err) {
			return nil, ErrStorageLocationNotFound
		}
		return nil, err
	}
	updates := make(map[string]interface{})
	if name != nil {
		trimmed := strings.TrimSpace(*name)
		if trimmed == "" || len(trimmed) > maxStorageLocationNameLength {
			return nil, ErrStorageLocationNameInvalid
		}
		if !strings.EqualFold(trimmed, location.Name) {
			exists, err := s.repo.ExistsByName(userID, trimmed)
			if err != nil {
				return nil, err
			}
			if exists {
				return nil, ErrStorageLocationDuplicate
			}
		}
		updates["name"] = trimmed
	}
	if sortOrder != nil {
		updates["sort_order"] = *sortOrder
	}
	if len(updates) == 0 {
		return location, nil
	}
	if err := s.repo.Update(location, updates); err != nil {
		return nil, err
	}
	return s.repo.GetByID(id, userID)
}

// Delete removes an unused storage location. It returns the number of referencing coins on conflict.
func (s *StorageLocationService) Delete(id, userID uint) (int64, error) {
	if _, err := s.repo.GetByID(id, userID); err != nil {
		if repository.IsRecordNotFound(err) {
			return 0, ErrStorageLocationNotFound
		}
		return 0, err
	}
	count, err := s.repo.CountCoinsUsing(id, userID)
	if err != nil {
		return 0, err
	}
	if count > 0 {
		return count, ErrStorageLocationInUse
	}
	return 0, s.repo.Delete(id, userID)
}

// ValidateOwnership verifies a storage location belongs to the user.
func (s *StorageLocationService) ValidateOwnership(id, userID uint) error {
	exists, err := s.repo.ExistsByID(id, userID)
	if err != nil {
		return err
	}
	if !exists {
		return ErrStorageLocationNotFound
	}
	return nil
}
