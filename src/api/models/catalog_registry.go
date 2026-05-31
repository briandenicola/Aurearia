package models

import "time"

// CatalogRegistry defines validation rules for supported catalog codes.
type CatalogRegistry struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	Catalog        string    `gorm:"type:varchar(32);uniqueIndex;not null" json:"catalog" binding:"required,max=32"`
	DisplayName    string    `gorm:"type:varchar(128);not null" json:"displayName" binding:"required,max=128"`
	Era            Era       `gorm:"type:varchar(20);not null" json:"era"`
	VolumeRequired bool      `gorm:"not null;default:false" json:"volumeRequired"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}
