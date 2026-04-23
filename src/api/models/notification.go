package models

import "time"

type Notification struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	UserID       uint      `gorm:"not null;index" json:"userId"`
	User         User      `gorm:"foreignKey:UserID" json:"-"`
	Type         string    `gorm:"not null;index" json:"type"` // wishlist_unavailable, friend_new_coin
	Title        string    `gorm:"not null" json:"title"`
	Message      string    `gorm:"type:text;not null" json:"message"`
	ReferenceID  uint      `gorm:"default:0" json:"referenceId"`
	ReferenceURL string    `json:"referenceUrl,omitempty"`
	IsRead       bool      `gorm:"default:false;index" json:"isRead"`
	CreatedAt    time.Time `json:"createdAt"`
}
