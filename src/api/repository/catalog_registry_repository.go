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

// FindByID returns a registry entry by ID.
func (r *CatalogRegistryRepository) FindByID(id uint) (*models.CatalogRegistry, error) {
	var entry models.CatalogRegistry
	err := r.db.First(&entry, id).Error
	if err != nil {
		return nil, err
	}
	return &entry, nil
}

// Create creates a new registry entry.
func (r *CatalogRegistryRepository) Create(entry *models.CatalogRegistry) error {
	return r.db.Create(entry).Error
}

// Update updates a registry entry with the given updates map.
func (r *CatalogRegistryRepository) Update(entry *models.CatalogRegistry, updates map[string]interface{}) error {
	return r.db.Model(entry).Updates(updates).Error
}

// Delete deletes a registry entry by ID.
func (r *CatalogRegistryRepository) Delete(id uint) error {
	return r.db.Delete(&models.CatalogRegistry{}, id).Error
}

// CountReferencesUsing returns the count of coin_references using this catalog code.
func (r *CatalogRegistryRepository) CountReferencesUsing(catalog string) (int64, error) {
	var count int64
	err := r.db.Table("coin_references").
		Where("catalog = ?", strings.ToUpper(strings.TrimSpace(catalog))).
		Count(&count).Error
	return count, err
}
