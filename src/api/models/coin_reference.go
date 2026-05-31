package models

import "time"

// CoinReference stores a normalized catalog attribution for a coin.
type CoinReference struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	CoinID    uint      `gorm:"not null;index;index:idx_coin_ref_unique,unique" json:"coinId"`
	Catalog   string    `gorm:"type:varchar(32);not null;index;index:idx_coin_ref_unique,unique" json:"catalog" binding:"required,max=32"`
	Volume    string    `gorm:"type:varchar(64);index:idx_coin_ref_unique,unique" json:"volume" binding:"max=64"`
	Number    string    `gorm:"type:varchar(128);not null;index:idx_coin_ref_unique,unique" json:"number" binding:"required,max=128"`
	Certainty string    `gorm:"type:varchar(32)" json:"certainty" binding:"max=32"`
	URI       string    `gorm:"column:uri;type:varchar(2000)" json:"uri" binding:"max=2000"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}
