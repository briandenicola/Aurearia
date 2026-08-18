package models

import "time"

const (
	AvailabilityCycleStatusQueued         = "queued"
	AvailabilityCycleStatusRunning        = "running"
	AvailabilityCycleStatusCompleted      = "completed"
	AvailabilityCycleStatusFailed         = "failed"
	AvailabilityCycleStatusPartialFailure = "partial_failure"
)

// AvailabilityCycle is the parent record for an admin- or scheduler-triggered wishlist
// availability check that fans out into one AvailabilityRun (child) per affected wishlist
// owner. It owns the operational roll-up counts for its children and is finalized
// automatically as its children reach a terminal state (completed/failed). Legacy
// single-run admin checks (AvailabilityRun rows with UserID = 0, predating this feature)
// are never retroactively attached to a synthesized cycle — this table only contains
// cycles created going forward.
type AvailabilityCycle struct {
	ID                uint              `gorm:"primaryKey" json:"id"`
	TriggerType       string            `gorm:"type:varchar(20);not null" json:"triggerType"`
	TriggerUserID     *uint             `json:"triggerUserId"`
	Status            string            `gorm:"type:varchar(20);not null;default:queued" json:"status"`
	TotalChildren     int               `json:"totalChildren"`
	QueuedChildren    int               `json:"queuedChildren"`
	RunningChildren   int               `json:"runningChildren"`
	CompletedChildren int               `json:"completedChildren"`
	FailedChildren    int               `json:"failedChildren"`
	FailMessage       string            `gorm:"type:text" json:"failMessage,omitempty"`
	StartedAt         time.Time         `gorm:"not null" json:"startedAt"`
	CompletedAt       *time.Time        `json:"completedAt"`
	CreatedAt         time.Time         `json:"createdAt"`
	Children          []AvailabilityRun `gorm:"foreignKey:CycleID" json:"children,omitempty"`
}
