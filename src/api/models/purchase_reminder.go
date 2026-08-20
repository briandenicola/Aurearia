package models

import "time"

// PurchaseReminder records a user's intent to be reminded about a wishlist coin
// on a specific date. One active (pending/notified) reminder per (coin, user).
// RemindDate is stored as YYYY-MM-DD (varchar) to avoid SQLite date-type ambiguity
// and make per-timezone evaluation explicit in the scheduler.
type PurchaseReminder struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	CoinID      uint       `gorm:"not null;index" json:"coinId"`
	Coin        Coin       `gorm:"foreignKey:CoinID" json:"-"`
	UserID      uint       `gorm:"not null;index" json:"userId"`
	User        User       `gorm:"foreignKey:UserID" json:"-"`
	RemindDate  string     `gorm:"type:varchar(10);not null;index" json:"remindDate"`
	Timezone    string     `gorm:"type:varchar(64);not null" json:"timezone"`
	Status      string     `gorm:"type:varchar(20);not null;default:'pending';index" json:"status"`
	NotifiedAt  *time.Time `json:"notifiedAt,omitempty"`
	CancelledAt *time.Time `json:"cancelledAt,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`

	// CoinName is a computed field populated by service/repository joins; not stored in DB.
	CoinName string `gorm:"-" json:"coinName,omitempty"`
}
