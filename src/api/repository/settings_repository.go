package repository

import (
	"github.com/briandenicola/ancient-coins-api/models"
	"gorm.io/gorm"
)

// SettingsRepository encapsulates all app-settings-related database operations.
type SettingsRepository struct {
	db *gorm.DB
}

// NewSettingsRepository creates a new SettingsRepository.
func NewSettingsRepository(db *gorm.DB) *SettingsRepository {
	return &SettingsRepository{db: db}
}

// FindByKey returns a single AppSetting by key.
func (r *SettingsRepository) FindByKey(key string) (*models.AppSetting, error) {
	var setting models.AppSetting
	err := r.db.Where("key = ?", key).First(&setting).Error
	return &setting, err
}

// Upsert creates or updates a setting by key.
func (r *SettingsRepository) Upsert(key, value string) error {
	var setting models.AppSetting
	result := r.db.Where("key = ?", key).First(&setting)
	if result.Error != nil {
		setting = models.AppSetting{Key: key, Value: value}
		return r.db.Create(&setting).Error
	}
	setting.Value = value
	return r.db.Save(&setting).Error
}

// FindAll returns all stored settings.
func (r *SettingsRepository) FindAll() ([]models.AppSetting, error) {
	var settings []models.AppSetting
	err := r.db.Find(&settings).Error
	return settings, err
}
