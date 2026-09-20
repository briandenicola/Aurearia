package models

import "time"

// ImageCleanup survives metadata deletion until all three image files are gone.
// No foreign keys: deleting a coin/account must not discard pending cleanup.
type ImageCleanup struct {
	ImageID   uint   `gorm:"primaryKey;autoIncrement:false"`
	CoinID    uint   `gorm:"not null"`
	UserID    uint   `gorm:"not null"`
	FilePath  string `gorm:"not null"`
	CreatedAt time.Time
}
