package models

import "time"

// CollectionHealthSnapshotRun records a scheduled or manual collection health snapshot execution.
type CollectionHealthSnapshotRun struct {
	ID               uint       `gorm:"primaryKey" json:"id"`
	TriggerType      string     `gorm:"type:varchar(20);not null" json:"triggerType"`
	Status           string     `gorm:"type:varchar(20);not null;default:'running'" json:"status"`
	UsersEligible    int        `json:"usersEligible"`
	UsersSnapshotted int        `json:"usersSnapshotted"`
	UsersFailed      int        `json:"usersFailed"`
	DurationMs       int64      `json:"durationMs"`
	StartedAt        time.Time  `gorm:"not null" json:"startedAt"`
	CompletedAt      *time.Time `json:"completedAt"`
	ErrorMessage     string     `gorm:"type:text" json:"errorMessage,omitempty"`
	CreatedAt        time.Time  `json:"createdAt"`
}
