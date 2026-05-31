package repository

import (
	"strings"

	"github.com/briandenicola/ancient-coins-api/models"
	"gorm.io/gorm"
)

// CatalogRegistryRepository handles registry rule lookups.
type CatalogRegistryRepository struct {
	db *gorm.DB
}

// NewCatalogRegistryRepository creates a new CatalogRegistryRepository.
func NewCatalogRegistryRepository(db *gorm.DB) *CatalogRegistryRepository {
	return &CatalogRegistryRepository{db: db}
}

// List returns all registry entries ordered by catalog code.
func (r *CatalogRegistryRepository) List() ([]models.CatalogRegistry, error) {
	var entries []models.CatalogRegistry
	err := r.db.Order("catalog ASC").Find(&entries).Error
	return entries, err
}

// FindByCatalog returns a registry entry by catalog code.
func (r *CatalogRegistryRepository) FindByCatalog(catalog string) (*models.CatalogRegistry, error) {
	var entry models.CatalogRegistry
	err := r.db.Where("catalog = ?", strings.ToUpper(strings.TrimSpace(catalog))).First(&entry).Error
	if err != nil {
		return nil, err
	}
	return &entry, nil
}
