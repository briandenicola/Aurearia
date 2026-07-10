package models

import "time"

type CoinOfDayRunTriggerType string

const (
	CoinOfDayRunTriggerManual    CoinOfDayRunTriggerType = "manual"
	CoinOfDayRunTriggerScheduled CoinOfDayRunTriggerType = "scheduled"
)

type CoinOfDayRunStatus string

const (
	CoinOfDayRunStatusQueued    CoinOfDayRunStatus = "queued"
	CoinOfDayRunStatusRunning   CoinOfDayRunStatus = "running"
	CoinOfDayRunStatusCompleted CoinOfDayRunStatus = "completed"
	CoinOfDayRunStatusFailed    CoinOfDayRunStatus = "failed"
)

type CoinOfDayRun struct {
	ID            uint                    `gorm:"primaryKey" json:"id"`
	TriggerType   CoinOfDayRunTriggerType `gorm:"type:varchar(20);not null;index" json:"triggerType"`
	TriggerUserID *uint                   `gorm:"index" json:"triggerUserId"`
	TriggerUser   *User                   `gorm:"foreignKey:TriggerUserID" json:"-"`
	Status        CoinOfDayRunStatus      `gorm:"type:varchar(20);not null;index" json:"status"`
	StartedAt     time.Time               `gorm:"not null;index" json:"startedAt"`
	CompletedAt   *time.Time              `json:"completedAt"`
	Picked        int                     `gorm:"not null;default:0" json:"picked"`
	Skipped       int                     `gorm:"not null;default:0" json:"skipped"`
	Errors        int                     `gorm:"not null;default:0" json:"errors"`
	ErrorMessage  string                  `gorm:"type:text" json:"errorMessage"`
	CreatedAt     time.Time               `json:"createdAt"`
	UpdatedAt     time.Time               `json:"updatedAt"`
}
