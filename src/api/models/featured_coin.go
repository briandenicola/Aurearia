package models

import "time"

// FeaturedCoin records one user's "Coin of the Day" selection.
// One row is inserted per user per day when the scheduler picks a coin.
// The Summary is snapshotted at creation so the modal can render without
// recomputing AI analysis. Selection enforces a cycle: each owned coin is
// featured at most once before any is featured again.
type FeaturedCoin struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	UserID     uint      `gorm:"not null;index:idx_featured_user_date" json:"userId"`
	User       User      `gorm:"foreignKey:UserID" json:"-"`
	CoinID     uint      `gorm:"not null;index" json:"coinId"`
	Coin       *Coin     `gorm:"foreignKey:CoinID" json:"coin,omitempty"`
	Summary    string    `gorm:"type:text" json:"summary"`
	FeaturedAt time.Time `gorm:"not null;index:idx_featured_user_date" json:"featuredAt"`
	CreatedAt  time.Time `json:"createdAt"`
}
