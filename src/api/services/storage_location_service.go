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
	ErrStorageLocationTypeInvalid = errors.New("storage location type must be standard or tray")
	ErrStorageLocationDimensions  = errors.New("tray rows and columns must each be between 1 and 20")
	ErrTrayOccupied               = errors.New("occupied tray dimensions cannot be changed")
	ErrLocationReferenced         = ErrStorageLocationInUse
)

type StorageLocationWrite struct {
	Name      string
	Type      string
	Rows      *int
	Columns   *int
	SortOrder int
}

type StorageLocationUpdate struct {
	Name       *string
	Rows       *int
	Columns    *int
	RowsSet    bool
	ColumnsSet bool
	SortOrder  *int
}

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
	return s.CreateTyped(userID, StorageLocationWrite{Name: name, Type: models.StorageLocationTypeStandard, SortOrder: sortOrder})
}

func (s *StorageLocationService) CreateTyped(userID uint, input StorageLocationWrite) (*models.StorageLocation, error) {
	name := input.Name
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
	locationType := strings.ToLower(strings.TrimSpace(input.Type))
	if locationType == "" {
		locationType = models.StorageLocationTypeStandard
	}
	rows, columns, err := validateStorageLocationShape(locationType, input.Rows, input.Columns)
	if err != nil {
		return nil, err
	}
	location := &models.StorageLocation{
		UserID: userID, Name: name, Type: locationType, Rows: rows,
		Columns: columns, SortOrder: input.SortOrder,
	}
	if err := s.repo.Create(location); err != nil {
		if errors.Is(err, repository.ErrStorageLocationNameConflict) {
			return nil, ErrStorageLocationDuplicate
		}
		return nil, err
	}
	if rows != nil && columns != nil {
		location.Capacity = *rows * *columns
	}
	return location, nil
}

// Update validates and updates a storage location.
func (s *StorageLocationService) Update(id, userID uint, name *string, sortOrder *int) (*models.StorageLocation, error) {
	return s.UpdateTyped(id, userID, StorageLocationUpdate{Name: name, SortOrder: sortOrder})
}

func (s *StorageLocationService) UpdateTyped(id, userID uint, input StorageLocationUpdate) (*models.StorageLocation, error) {
	location, err := s.repo.GetByID(id, userID)
	if err != nil {
		if repository.IsRecordNotFound(err) {
			return nil, ErrStorageLocationNotFound
		}
		return nil, err
	}
	updates := make(map[string]interface{})
	if input.Name != nil {
		trimmed := strings.TrimSpace(*input.Name)
		if trimmed == "" || len(trimmed) > maxStorageLocationNameLength {
			return nil, ErrStorageLocationNameInvalid
		}
		if !strings.EqualFold(trimmed, location.Name) {
			exists, err := s.repo.ExistsByNameExcluding(userID, trimmed, id)
			if err != nil {
				return nil, err
			}
			if exists {
				return nil, ErrStorageLocationDuplicate
			}
		}
		updates["name"] = trimmed
	}
	if input.SortOrder != nil {
		updates["sort_order"] = *input.SortOrder
	}
	if input.RowsSet || input.ColumnsSet {
		if location.Type != models.StorageLocationTypeTray {
			return nil, ErrStorageLocationDimensions
		}
		rows, columns := location.Rows, location.Columns
		if input.RowsSet {
			rows = input.Rows
		}
		if input.ColumnsSet {
			columns = input.Columns
		}
		rows, columns, err = validateStorageLocationShape(location.Type, rows, columns)
		if err != nil {
			return nil, err
		}
		changed := location.Rows == nil || location.Columns == nil || *location.Rows != *rows || *location.Columns != *columns
		if changed {
			occupied, err := s.repo.CountOccupied(id, userID)
			if err != nil {
				return nil, err
			}
			if occupied > 0 {
				return nil, ErrTrayOccupied
			}
			updates["rows"] = *rows
			updates["columns"] = *columns
		}
	}
	if len(updates) == 0 {
		return location, nil
	}
	if err := s.repo.Transaction(func(txRepo *repository.StorageLocationRepository) error {
		current, err := txRepo.GetByID(id, userID)
		if err != nil {
			return err
		}
		if _, resizingRows := updates["rows"]; resizingRows {
			occupied, err := txRepo.CountOccupied(id, userID)
			if err != nil {
				return err
			}
			if occupied > 0 {
				return ErrTrayOccupied
			}
		}
		return txRepo.Update(current, updates)
	}); err != nil {
		if errors.Is(err, repository.ErrStorageLocationNameConflict) {
			return nil, ErrStorageLocationDuplicate
		}
		return nil, err
	}
	return s.repo.GetByID(id, userID)
}

// Delete removes an unused storage location. It returns the number of referencing coins on conflict.
func (s *StorageLocationService) Delete(id, userID uint) (int64, error) {
	var count int64
	err := s.repo.Transaction(func(txRepo *repository.StorageLocationRepository) error {
		if _, err := txRepo.GetByID(id, userID); err != nil {
			if repository.IsRecordNotFound(err) {
				return ErrStorageLocationNotFound
			}
			return err
		}
		var err error
		count, err = txRepo.CountCoinsUsing(id, userID)
		if err != nil {
			return err
		}
		if count > 0 {
			return ErrStorageLocationInUse
		}
		return txRepo.Delete(id, userID)
	})
	return count, err
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

func validateStorageLocationShape(locationType string, rows, columns *int) (*int, *int, error) {
	switch locationType {
	case models.StorageLocationTypeStandard:
		if rows != nil || columns != nil {
			return nil, nil, ErrStorageLocationDimensions
		}
		return nil, nil, nil
	case models.StorageLocationTypeTray:
		if rows == nil || columns == nil || *rows < 1 || *rows > 20 || *columns < 1 || *columns > 20 || (*rows)*(*columns) > 400 {
			return nil, nil, ErrStorageLocationDimensions
		}
		return rows, columns, nil
	default:
		return nil, nil, ErrStorageLocationTypeInvalid
	}
}

func (s *StorageLocationService) Get(id, userID uint) (*models.StorageLocation, error) {
	location, err := s.repo.GetByID(id, userID)
	if repository.IsRecordNotFound(err) {
		return nil, ErrStorageLocationNotFound
	}
	return location, err
}

func (s *StorageLocationService) Occupancy(id, userID uint, coinID *uint) (*repository.TrayOccupancy, error) {
	result, err := s.repo.Occupancy(id, userID, coinID)
	if repository.IsRecordNotFound(err) {
		return nil, ErrStorageLocationNotFound
	}
	return result, err
}

func (s *StorageLocationService) ListTrayAggregates(userID uint) ([]repository.TrayAggregate, error) {
	return s.repo.ListTrayAggregates(userID)
}
