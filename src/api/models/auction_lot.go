package models

import "time"

type AuctionLotStatus string

const (
	AuctionStatusWatching AuctionLotStatus = "watching"
	AuctionStatusBidding  AuctionLotStatus = "bidding"
	AuctionStatusWon      AuctionLotStatus = "won"
	AuctionStatusLost     AuctionLotStatus = "lost"
	AuctionStatusPassed   AuctionLotStatus = "passed"
)

type AuctionLot struct {
	ID           uint             `gorm:"primaryKey" json:"id"`
	NumisBidsURL string           `gorm:"not null" json:"numisBidsUrl"`
	SaleID       string           `json:"saleId"`
	LotNumber    int              `json:"lotNumber"`
	AuctionHouse string           `json:"auctionHouse"`
	SaleName     string           `json:"saleName"`
	SaleDate       *time.Time       `json:"saleDate"`
	AuctionEndTime *time.Time       `json:"auctionEndTime"`
	Title        string           `gorm:"not null" json:"title"`
	Description  string           `gorm:"type:text" json:"description"`
	Category     Category         `gorm:"type:varchar(20);default:'Other'" json:"category"`
	Estimate     *float64         `json:"estimate"`
	CurrentBid   *float64         `json:"currentBid"`
	MaxBid       *float64         `json:"maxBid"`
	Currency     string           `gorm:"default:'USD'" json:"currency"`
	Status       AuctionLotStatus `gorm:"type:varchar(20);default:'watching'" json:"status"`
	ImageURL     string           `json:"imageUrl"`
	CoinID       *uint            `json:"coinId"`
	Coin         *Coin            `gorm:"foreignKey:CoinID" json:"coin,omitempty"`
	EventID      *uint            `json:"eventId"`
	Event        *AuctionEvent    `gorm:"foreignKey:EventID" json:"event,omitempty"`
	UserID       uint             `gorm:"not null" json:"userId"`
	User         User             `gorm:"foreignKey:UserID" json:"-"`
	CreatedAt    time.Time        `json:"createdAt"`
	UpdatedAt    time.Time        `json:"updatedAt"`
}
