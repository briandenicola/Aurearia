package models

import "time"

type AuctionLotStatus string
type AuctionSource string
type AuctionLotStatusSource string

const (
	AuctionStatusWatching AuctionLotStatus = "watching"
	AuctionStatusBidding  AuctionLotStatus = "bidding"
	AuctionStatusWon      AuctionLotStatus = "won"
	AuctionStatusLost     AuctionLotStatus = "lost"
	AuctionStatusPassed   AuctionLotStatus = "passed"
)

const (
	// AuctionLotStatusSourceSync means the current status was set automatically by
	// watchlist sync detecting a provider-reported outcome (currently CNG only).
	AuctionLotStatusSourceSync AuctionLotStatusSource = "sync"
	// AuctionLotStatusSourceManual means the current status was set by an explicit
	// user action (the manual status override).
	AuctionLotStatusSourceManual AuctionLotStatusSource = "manual"
)

const (
	AuctionSourceNumisBids AuctionSource = "numisbids"
	AuctionSourceCNG       AuctionSource = "cng"
)

type AuctionLot struct {
	ID             uint          `gorm:"primaryKey" json:"id"`
	NumisBidsURL   string        `gorm:"not null" json:"numisBidsUrl"`
	Source         AuctionSource `gorm:"type:varchar(20);default:'numisbids';index" json:"source"`
	SourceURL      string        `gorm:"index" json:"sourceUrl"`
	SourceLotID    string        `gorm:"index" json:"sourceLotId,omitempty"`
	SourceSaleID   string        `gorm:"index" json:"sourceSaleId,omitempty"`
	SaleID         string        `json:"saleId"`
	LotNumber      int           `json:"lotNumber"`
	AuctionHouse   string        `json:"auctionHouse"`
	SaleName       string        `json:"saleName"`
	SaleDate       *time.Time    `json:"saleDate"`
	AuctionEndTime *time.Time    `json:"auctionEndTime"`
	Title          string        `gorm:"not null" json:"title"`
	Description    string        `gorm:"type:text" json:"description"`
	Notes          string        `gorm:"type:text" json:"notes"`
	Category       Category      `gorm:"type:varchar(20);default:'Other'" json:"category"`
	Estimate       *float64      `json:"estimate"`
	InitialBid     *float64      `json:"initialBid"`
	CurrentBid     *float64      `json:"currentBid"`
	// LastDigestBid is the CurrentBid this lot carried the last time it was reported in a
	// watch-bid digest. It is the baseline the next digest compares against ("up from
	// 75.00"), so it only moves when a digest is actually delivered — never on sync. Kept
	// out of the JSON contract: it is digest bookkeeping, not lot data the UI renders
	// (specs/_backlog/F032).
	LastDigestBid *float64 `json:"-"`
	MaxBid        *float64 `json:"maxBid"`
	// IsOutbid records whether the provider reports someone else holding the winning bid on
	// a lot the user has a bid on. It is provider truth (CNG names the winning bidder), not
	// a maxBid/currentBid comparison, which proxy bidding makes unreliable: a ceiling above
	// the current bid can still be losing, and one below it can still be leading. Only lots
	// still open for bidding carry it; a closed lot is won/lost/passed, never outbid
	// (specs/_backlog/F034).
	IsOutbid   bool             `gorm:"default:false" json:"isOutbid"`
	WinningBid *float64         `json:"winningBid"`
	Currency   string           `gorm:"default:'USD'" json:"currency"`
	Status     AuctionLotStatus `gorm:"type:varchar(20);default:'watching'" json:"status"`
	// StatusSource records whether the current Status was set by watchlist sync
	// auto-detecting a provider-reported outcome, or by an explicit manual override.
	// Only meaningful once Status is won/lost; see specs/_backlog/F024.
	StatusSource AuctionLotStatusSource `gorm:"type:varchar(10);default:'manual'" json:"statusSource"`
	ImageURL     string                 `json:"imageUrl"`
	CoinID       *uint                  `json:"coinId"`
	Coin         *Coin                  `gorm:"foreignKey:CoinID" json:"coin,omitempty"`
	EventID      *uint                  `json:"eventId"`
	Event        *AuctionEvent          `gorm:"foreignKey:EventID" json:"event,omitempty"`
	UserID       uint                   `gorm:"not null" json:"userId"`
	User         User                   `gorm:"foreignKey:UserID" json:"-"`
	CreatedAt    time.Time              `json:"createdAt"`
	UpdatedAt    time.Time              `json:"updatedAt"`
}
