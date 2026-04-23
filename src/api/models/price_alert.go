package models

import "time"

type PriceAlert struct {
	ID           uint       `gorm:"primaryKey" json:"id"`
	AuctionLotID uint       `gorm:"not null;index" json:"auctionLotId"`
	AuctionLot   AuctionLot `gorm:"foreignKey:AuctionLotID" json:"-"`
	UserID       uint       `gorm:"not null;index" json:"userId"`
	User         User       `gorm:"foreignKey:UserID" json:"-"`
	TargetPrice  float64    `gorm:"not null" json:"targetPrice"`
	Direction    string     `gorm:"type:varchar(10);default:'above'" json:"direction"` // "above" or "below"
	IsTriggered  bool       `gorm:"default:false" json:"isTriggered"`
	TriggeredAt  *time.Time `json:"triggeredAt"`
	CreatedAt    time.Time  `json:"createdAt"`
}

type BidReminder struct {
	ID             uint       `gorm:"primaryKey" json:"id"`
	AuctionLotID   uint       `gorm:"not null;index" json:"auctionLotId"`
	AuctionLot     AuctionLot `gorm:"foreignKey:AuctionLotID" json:"-"`
	UserID         uint       `gorm:"not null;index" json:"userId"`
	User           User       `gorm:"foreignKey:UserID" json:"-"`
	MinutesBefore  int        `gorm:"not null;default:30" json:"minutesBefore"`
	IsNotified     bool       `gorm:"default:false" json:"isNotified"`
	NotifiedAt     *time.Time `json:"notifiedAt"`
	CreatedAt      time.Time  `json:"createdAt"`
}
