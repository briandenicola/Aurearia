package services

import (
	"errors"
	"strings"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"gorm.io/gorm"
)

var (
	ErrCatalogNotFound     = errors.New("catalog not found")
	ErrCatalogDuplicate    = errors.New("catalog code already exists")
	ErrCatalogInUse        = errors.New("catalog is in use by existing references")
	ErrCatalogInvalidEra   = errors.New("era must be ancient, medieval, or modern")
	ErrCatalogCodeRequired = errors.New("catalog code is required")
	ErrCatalogNameRequired = errors.New("display name is required")
)

// CatalogRegistryService manages catalog registry rules.
type CatalogRegistryService struct {
	repo *repository.CatalogRegistryRepository
}

// NewCatalogRegistryService creates a new CatalogRegistryService.
func NewCatalogRegistryService(repo *repository.CatalogRegistryRepository) *CatalogRegistryService {
	return &CatalogRegistryService{repo: repo}
}

// List returns all catalog registry entries.
func (s *CatalogRegistryService) List() ([]models.CatalogRegistry, error) {
	return s.repo.List()
}

// Create creates a new catalog registry entry after validation.
func (s *CatalogRegistryService) Create(entry models.CatalogRegistry) (models.CatalogRegistry, error) {
	entry.Catalog = strings.ToUpper(strings.TrimSpace(entry.Catalog))
	entry.DisplayName = strings.TrimSpace(entry.DisplayName)

	if entry.Catalog == "" {
		return entry, ErrCatalogCodeRequired
	}
	if entry.DisplayName == "" {
		return entry, ErrCatalogNameRequired
	}
	if !isValidEra(entry.Era) {
		return entry, ErrCatalogInvalidEra
	}

	// Check for duplicate code
	existing, err := s.repo.FindByCatalog(entry.Catalog)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return entry, err
	}
	if existing != nil {
		return entry, ErrCatalogDuplicate
	}

	if err := s.repo.Create(&entry); err != nil {
		return entry, err
	}

	return entry, nil
}

// Update updates an existing catalog registry entry after validation.
func (s *CatalogRegistryService) Update(id uint, entry models.CatalogRegistry) (models.CatalogRegistry, error) {
	existing, err := s.repo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return entry, ErrCatalogNotFound
		}
		return entry, err
	}

	entry.Catalog = strings.ToUpper(strings.TrimSpace(entry.Catalog))
	entry.DisplayName = strings.TrimSpace(entry.DisplayName)

	if entry.Catalog == "" {
		return entry, ErrCatalogCodeRequired
	}
	if entry.DisplayName == "" {
		return entry, ErrCatalogNameRequired
	}
	if !isValidEra(entry.Era) {
		return entry, ErrCatalogInvalidEra
	}

	// If code changed, check for duplicate
	if entry.Catalog != existing.Catalog {
		duplicate, err := s.repo.FindByCatalog(entry.Catalog)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return entry, err
		}
		if duplicate != nil {
			return entry, ErrCatalogDuplicate
		}
	}

	updates := map[string]interface{}{
		"catalog":         entry.Catalog,
		"display_name":    entry.DisplayName,
		"era":             entry.Era,
		"volume_required": entry.VolumeRequired,
	}

	if err := s.repo.Update(existing, updates); err != nil {
		return entry, err
	}

	// Return updated record
	updated, err := s.repo.FindByID(id)
	if err != nil {
		return entry, err
	}

	return *updated, nil
}

// Delete deletes a catalog registry entry if it's not in use.
func (s *CatalogRegistryService) Delete(id uint) error {
	existing, err := s.repo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrCatalogNotFound
		}
		return err
	}

	// Check if catalog is in use
	count, err := s.repo.CountReferencesUsing(existing.Catalog)
	if err != nil {
		return err
	}
	if count > 0 {
		return ErrCatalogInUse
	}

	return s.repo.Delete(id)
}

func isValidEra(era models.Era) bool {
	return era == models.EraAncient || era == models.EraMedieval || era == models.EraModern
}
