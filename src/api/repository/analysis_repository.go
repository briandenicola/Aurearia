package repository

import (
	"github.com/briandenicola/ancient-coins-api/models"
	"gorm.io/gorm"
)

// AnalysisRepository encapsulates database operations for coin analysis.
type AnalysisRepository struct {
	db *gorm.DB
}

// NewAnalysisRepository creates a new AnalysisRepository.
func NewAnalysisRepository(db *gorm.DB) *AnalysisRepository {
	return &AnalysisRepository{db: db}
}

// FindCoinWithImages finds a coin by ID and user ID with images preloaded.
func (r *AnalysisRepository) FindCoinWithImages(coinID uint, userID uint) (*models.Coin, error) {
	var coin models.Coin
	if err := r.db.Where("id = ? AND user_id = ?", coinID, userID).Preload("Images").Preload("StorageLocation").First(&coin).Error; err != nil {
		return nil, err
	}
	return &coin, nil
}

// UpdateCoinField updates a single field on a coin.
func (r *AnalysisRepository) UpdateCoinField(coin *models.Coin, field string, value interface{}) error {
	return r.db.Model(coin).Update(field, value).Error
}

// ReloadCoinWithImages reloads a coin by ID with images preloaded.
func (r *AnalysisRepository) ReloadCoinWithImages(coinID uint) (*models.Coin, error) {
	var coin models.Coin
	if err := r.db.Where("id = ?", coinID).Preload("Images").Preload("StorageLocation").First(&coin).Error; err != nil {
		return nil, err
	}
	return &coin, nil
}
