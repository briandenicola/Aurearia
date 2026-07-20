package services

import (
	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
)

// RomanImperialFigureService provides read access to the curated F028
// imperial-figure reference dataset (see specs/_backlog/F028-imperial-figures.md).
type RomanImperialFigureService struct {
	repo *repository.RomanImperialFigureRepository
}

// NewRomanImperialFigureService creates a new RomanImperialFigureService.
func NewRomanImperialFigureService(repo *repository.RomanImperialFigureRepository) *RomanImperialFigureService {
	return &RomanImperialFigureService{repo: repo}
}

// Search returns figures matching an optional name/alias query and/or role filter.
func (s *RomanImperialFigureService) Search(query string, role models.ImperialFigureRole, limit int) ([]models.RomanImperialFigure, error) {
	return s.repo.Search(query, role, limit)
}

// FindByID returns a single figure by ID, e.g. to resolve a previously
// selected figure's display name when a coin is opened for editing.
func (s *RomanImperialFigureService) FindByID(id uint) (*models.RomanImperialFigure, error) {
	return s.repo.FindByID(id)
}
