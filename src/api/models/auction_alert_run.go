package models

import "time"

// AuctionAlertRun records a scheduled or manual price-alert/bid-reminder execution.
type AuctionAlertRun struct {
	ID                   uint       `gorm:"primaryKey" json:"id"`
	TriggerType          string     `gorm:"type:varchar(20);not null" json:"triggerType"`
	TriggerUserID        *uint      `json:"triggerUserId"`
	Status               string     `gorm:"type:varchar(20);not null;default:'running'" json:"status"`
	LotsChecked          int        `json:"lotsChecked"`
	PriceAlertsTriggered int        `json:"priceAlertsTriggered"`
	BidRemindersSent     int        `json:"bidRemindersSent"`
	DurationMs           int64      `json:"durationMs"`
	StartedAt            time.Time  `gorm:"not null" json:"startedAt"`
	CompletedAt          *time.Time `json:"completedAt"`
	ErrorMessage         string     `gorm:"type:text" json:"errorMessage,omitempty"`
	CreatedAt            time.Time  `json:"createdAt"`
}
