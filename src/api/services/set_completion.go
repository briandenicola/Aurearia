package services

import (
	"github.com/briandenicola/ancient-coins-api/repository"
)

// SetCompletion handles set completion calculations.
type SetCompletion struct {
	repo *repository.SetRepository
}

// NewSetCompletion creates a new SetCompletion service.
func NewSetCompletion(repo *repository.SetRepository) *SetCompletion {
	return &SetCompletion{repo: repo}
}

// GetCompletion calculates and returns completion metrics for a set.
func (s *SetCompletion) GetCompletion(setID, userID uint) (map[string]interface{}, error) {
	return s.repo.GetSetCompletion(setID, userID)
}
