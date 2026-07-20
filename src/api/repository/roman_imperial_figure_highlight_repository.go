package repository

import (
	"github.com/briandenicola/ancient-coins-api/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// RomanImperialFigureHighlightRepository stores user-selected emperor tracker display coins.
type RomanImperialFigureHighlightRepository struct {
	db *gorm.DB
}

// NewRomanImperialFigureHighlightRepository creates a highlight repository.
func NewRomanImperialFigureHighlightRepository(db *gorm.DB) *RomanImperialFigureHighlightRepository {
	return &RomanImperialFigureHighlightRepository{db: db}
}

// ListForUser returns all highlight selections for the user.
func (r *RomanImperialFigureHighlightRepository) ListForUser(userID uint) ([]models.RomanImperialFigureHighlight, error) {
	var highlights []models.RomanImperialFigureHighlight
	err := r.db.Where("user_id = ?", userID).Find(&highlights).Error
	return highlights, err
}

// Upsert stores or replaces the highlighted coin for one user/figure pair.
func (r *RomanImperialFigureHighlightRepository) Upsert(highlight *models.RomanImperialFigureHighlight) error {
	return r.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "roman_imperial_figure_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"coin_id", "updated_at"}),
	}).Create(highlight).Error
}

// Delete removes a user highlight selection for one figure.
func (r *RomanImperialFigureHighlightRepository) Delete(userID, figureID uint) error {
	return r.db.Where("user_id = ? AND roman_imperial_figure_id = ?", userID, figureID).
		Delete(&models.RomanImperialFigureHighlight{}).Error
}
