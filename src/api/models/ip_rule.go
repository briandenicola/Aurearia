package models

import "time"

type IPRule struct {
	ID        uint       `gorm:"primaryKey" json:"id"`
	CIDR      string     `gorm:"column:cidr;type:varchar(64);not null;index" json:"cidr"`
	Reason    string     `gorm:"type:text" json:"reason"`
	ExpiresAt *time.Time `gorm:"index" json:"expiresAt,omitempty"`
	CreatedBy *uint      `gorm:"index" json:"createdBy,omitempty"`
	CreatedAt time.Time  `json:"createdAt"`
}
