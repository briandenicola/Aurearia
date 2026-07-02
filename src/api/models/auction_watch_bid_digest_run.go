package models

import "time"

// AuctionWatchBidDigestRun records a scheduled or manual auction watch bid digest execution.
type AuctionWatchBidDigestRun struct {
	ID            uint       `gorm:"primaryKey" json:"id"`
	TriggerType   string     `gorm:"type:varchar(20);not null" json:"triggerType"`
	TriggerUserID *uint      `json:"triggerUserId"`
	Status        string     `gorm:"type:varchar(20);not null;default:'running'" json:"status"`
	LotsChecked   int        `json:"lotsChecked"`
	DigestsSent   int        `json:"digestsSent"`
	DurationMs    int64      `json:"durationMs"`
	StartedAt     time.Time  `gorm:"not null" json:"startedAt"`
	CompletedAt   *time.Time `json:"completedAt"`
	ErrorMessage  string     `gorm:"type:text" json:"errorMessage,omitempty"`
	CreatedAt     time.Time  `json:"createdAt"`
}
