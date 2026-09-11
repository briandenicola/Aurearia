package models

import "time"

const (
	StorageLocationTypeStandard = "standard"
	StorageLocationTypeTray     = "tray"
)

// StorageLocation represents a user-defined storage location for coins.
type StorageLocation struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"not null;index;uniqueIndex:idx_user_storage_location_name" json:"userId"`
	Name      string    `gorm:"not null;type:varchar(100);uniqueIndex:idx_user_storage_location_name" json:"name" minLength:"1" maxLength:"100"`
	Type      string    `gorm:"not null;type:varchar(16);default:standard;index" json:"type" enums:"standard,tray" default:"standard"`
	Rows      *int      `json:"rows" minimum:"1" maximum:"20" extensions:"x-nullable"`
	Columns   *int      `json:"columns" minimum:"1" maximum:"20" extensions:"x-nullable"`
	SortOrder int       `gorm:"not null;default:0" json:"sortOrder"`
	Occupied  int64     `gorm:"-" json:"occupied" minimum:"0" maximum:"400"`
	Capacity  int       `gorm:"-" json:"capacity" minimum:"0" maximum:"400"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}
