package repository

import (
	"errors"

	"github.com/briandenicola/ancient-coins-api/models"
	"gorm.io/gorm"
)

type CollectorProfileRepository struct {
	db *gorm.DB
}

func NewCollectorProfileRepository(db *gorm.DB) *CollectorProfileRepository {
	return &CollectorProfileRepository{db: db}
}

func (r *CollectorProfileRepository) Get(userID uint) (*models.CollectorProfile, error) {
	var profile models.CollectorProfile
	err := r.db.Scopes(OwnedBy(userID)).First(&profile).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &profile, nil
}

// Replace atomically replaces the authenticated owner's complete profile.
// UserID on the supplied value is deliberately ignored.
func (r *CollectorProfileRepository) Replace(userID uint, profile *models.CollectorProfile) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Scopes(OwnedBy(userID)).Delete(&models.CollectorProfile{}).Error; err != nil {
			return err
		}
		profile.ID = 0
		profile.UserID = userID
		profile.CreatedAt = profile.CreatedAt.UTC()
		profile.UpdatedAt = profile.UpdatedAt.UTC()
		return tx.Create(profile).Error
	})
}
