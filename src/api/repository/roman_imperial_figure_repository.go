package repository

import (
	"strings"

	"github.com/briandenicola/ancient-coins-api/models"
	"gorm.io/gorm"
)

// RomanImperialFigureRepository encapsulates read access to the global,
// seeded RomanImperialFigure reference table (see F028).
type RomanImperialFigureRepository struct {
	db *gorm.DB
}

// NewRomanImperialFigureRepository creates a new RomanImperialFigureRepository.
func NewRomanImperialFigureRepository(db *gorm.DB) *RomanImperialFigureRepository {
	return &RomanImperialFigureRepository{db: db}
}

// List returns every seeded figure, ordered chronologically.
func (r *RomanImperialFigureRepository) List() ([]models.RomanImperialFigure, error) {
	var figures []models.RomanImperialFigure
	err := r.db.Order("sort_order ASC").Find(&figures).Error
	return figures, err
}

// ListByRole returns every seeded figure with the given role, ordered chronologically.
func (r *RomanImperialFigureRepository) ListByRole(role models.ImperialFigureRole) ([]models.RomanImperialFigure, error) {
	var figures []models.RomanImperialFigure
	err := r.db.Where("role = ?", role).Order("sort_order ASC").Find(&figures).Error
	return figures, err
}

// FindByID returns a single figure by ID.
func (r *RomanImperialFigureRepository) FindByID(id uint) (*models.RomanImperialFigure, error) {
	var figure models.RomanImperialFigure
	if err := r.db.First(&figure, id).Error; err != nil {
		return nil, err
	}
	return &figure, nil
}

// Search returns figures matching an optional name/alias substring query
// and/or an optional role filter, ordered chronologically and capped at
// limit. An empty query or role is treated as "no filter" on that axis.
func (r *RomanImperialFigureRepository) Search(query string, role models.ImperialFigureRole, limit int) ([]models.RomanImperialFigure, error) {
	db := r.db.Model(&models.RomanImperialFigure{})

	trimmed := strings.TrimSpace(query)
	if trimmed != "" {
		like := "%" + trimmed + "%"
		normalizedLike := "%" + models.NormalizeMintLocationName(trimmed) + "%"
		db = db.Where("name LIKE ? OR aliases LIKE ? OR normalized_name LIKE ?", like, like, normalizedLike)
	}
	if role != "" {
		db = db.Where("role = ?", role)
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	var figures []models.RomanImperialFigure
	err := db.Order("sort_order ASC").Limit(limit).Find(&figures).Error
	return figures, err
}
