package models

import "time"

type Showcase struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	UserID      uint           `gorm:"not null;index" json:"userId"`
	User        User           `gorm:"foreignKey:UserID" json:"-"`
	Slug        string         `gorm:"uniqueIndex;size:100;not null" json:"slug"`
	Title       string         `gorm:"size:200;not null" json:"title"`
	Description string         `gorm:"type:text" json:"description"`
	IsActive    bool           `gorm:"default:true" json:"isActive"`
	Coins       []ShowcaseCoin `gorm:"foreignKey:ShowcaseID" json:"coins,omitempty"`
	CreatedAt   time.Time      `json:"createdAt"`
	UpdatedAt   time.Time      `json:"updatedAt"`
}

type ShowcaseCoin struct {
	ID         uint `gorm:"primaryKey" json:"id"`
	ShowcaseID uint `gorm:"not null;uniqueIndex:idx_showcase_coin" json:"showcaseId"`
	CoinID     uint `gorm:"not null;uniqueIndex:idx_showcase_coin" json:"coinId"`
	Coin       Coin `gorm:"foreignKey:CoinID" json:"-"`
	SortOrder  int  `gorm:"default:0" json:"sortOrder"`
}

type AuctionEvent struct {
	ID           uint       `gorm:"primaryKey" json:"id"`
	UserID       uint       `gorm:"not null;index" json:"userId"`
	User         User       `gorm:"foreignKey:UserID" json:"-"`
	Title        string     `gorm:"size:300;not null" json:"title"`
	AuctionHouse string     `gorm:"size:200" json:"auctionHouse"`
	StartDate    *time.Time `json:"startDate"`
	EndDate      *time.Time `json:"endDate"`
	URL          string     `gorm:"size:500" json:"url"`
	Notes        string     `gorm:"type:text" json:"notes"`
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
}
