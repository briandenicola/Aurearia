package models

import "time"

// AuctionEndingRun records a single execution of the scheduled auction ending checker.
type AuctionEndingRun struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	TriggerType   string    `gorm:"type:varchar(20);not null" json:"triggerType"`
	TriggerUserID *uint     `json:"triggerUserId"`
	Status        string    `gorm:"type:varchar(20);not null;default:'running'" json:"status"`
	LotsChecked   int       `json:"lotsChecked"`
	AlertsSent    int       `json:"alertsSent"`
	DurationMs    int64     `json:"durationMs"`
	StartedAt     time.Time `gorm:"not null" json:"startedAt"`
	CompletedAt   *time.Time `json:"completedAt"`
	ErrorMessage  string    `gorm:"type:text" json:"errorMessage,omitempty"`
	CreatedAt     time.Time `json:"createdAt"`
}
