package models

import "time"

// StorageLocation represents a user-defined storage location for coins.
type StorageLocation struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"not null;index;uniqueIndex:idx_user_storage_location_name" json:"userId"`
	Name      string    `gorm:"not null;type:varchar(100);uniqueIndex:idx_user_storage_location_name" json:"name"`
	SortOrder int       `gorm:"not null;default:0" json:"sortOrder"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}
