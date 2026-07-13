package models

import "time"

type AuctionLotStatus string
type AuctionSource string

const (
	AuctionStatusWatching AuctionLotStatus = "watching"
	AuctionStatusBidding  AuctionLotStatus = "bidding"
	AuctionStatusWon      AuctionLotStatus = "won"
	AuctionStatusLost     AuctionLotStatus = "lost"
	AuctionStatusPassed   AuctionLotStatus = "passed"
)

const (
	AuctionSourceNumisBids AuctionSource = "numisbids"
	AuctionSourceCNG       AuctionSource = "cng"
)

type AuctionLot struct {
	ID             uint             `gorm:"primaryKey" json:"id"`
	NumisBidsURL   string           `gorm:"not null" json:"numisBidsUrl"`
	Source         AuctionSource    `gorm:"type:varchar(20);default:'numisbids';index" json:"source"`
	SourceURL      string           `gorm:"index" json:"sourceUrl"`
	SourceLotID    string           `gorm:"index" json:"sourceLotId,omitempty"`
	SourceSaleID   string           `gorm:"index" json:"sourceSaleId,omitempty"`
	SaleID         string           `json:"saleId"`
	LotNumber      int              `json:"lotNumber"`
	AuctionHouse   string           `json:"auctionHouse"`
	SaleName       string           `json:"saleName"`
	SaleDate       *time.Time       `json:"saleDate"`
	AuctionEndTime *time.Time       `json:"auctionEndTime"`
	Title          string           `gorm:"not null" json:"title"`
	Description    string           `gorm:"type:text" json:"description"`
	Notes          string           `gorm:"type:text" json:"notes"`
	Category       Category         `gorm:"type:varchar(20);default:'Other'" json:"category"`
	Estimate       *float64         `json:"estimate"`
	InitialBid     *float64         `json:"initialBid"`
	CurrentBid     *float64         `json:"currentBid"`
	MaxBid         *float64         `json:"maxBid"`
	WinningBid     *float64         `json:"winningBid"`
	Currency       string           `gorm:"default:'USD'" json:"currency"`
	Status         AuctionLotStatus `gorm:"type:varchar(20);default:'watching'" json:"status"`
	ImageURL       string           `json:"imageUrl"`
	CoinID         *uint            `json:"coinId"`
	Coin           *Coin            `gorm:"foreignKey:CoinID" json:"coin,omitempty"`
	EventID        *uint            `json:"eventId"`
	Event          *AuctionEvent    `gorm:"foreignKey:EventID" json:"event,omitempty"`
	UserID         uint             `gorm:"not null" json:"userId"`
	User           User             `gorm:"foreignKey:UserID" json:"-"`
	CreatedAt      time.Time        `json:"createdAt"`
	UpdatedAt      time.Time        `json:"updatedAt"`
}
