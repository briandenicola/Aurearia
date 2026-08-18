package models

import "time"

const (
	AvailabilityRunStatusQueued    = "queued"
	AvailabilityRunStatusRunning   = "running"
	AvailabilityRunStatusCompleted = "completed"
	AvailabilityRunStatusFailed    = "failed"
)

// Trigger type values written to *new* AvailabilityRun rows (feature 353). Pre-existing
// rows (including legacy admin rows with UserID = 0 and their historical "manual"/"scheduled"
// values) are never rewritten to these constants — they keep whatever value they already
// have and remain readable through the legacy admin endpoints unmodified.
const (
	AvailabilityRunTriggerOwner     = "owner"
	AvailabilityRunTriggerScheduled = "scheduled"
	AvailabilityRunTriggerAdmin     = "admin"
)

// GenericAvailabilityFailureMessage is the single generic, safe-for-display failure
// message used for AvailabilityRun.FailMessage, AvailabilityCycle.FailMessage, and the
// wishlist_availability_run notification body. It intentionally never includes URLs,
// query text, or internal error details (FR-015).
const GenericAvailabilityFailureMessage = "The availability check could not be completed due to an internal error. Please try again later."

// AvailabilityRun records a single execution of the wishlist availability checker.
// New rows created for the owner/scheduled/admin trigger types are always "child" runs
// scoped to exactly one user (UserID > 0); CycleID links admin/scheduled children back to
// their parent AvailabilityCycle. Owner-triggered runs (TriggerType = "owner") never belong
// to a cycle and always have a nil CycleID.
type AvailabilityRun struct {
	ID            uint                 `gorm:"primaryKey" json:"id"`
	UserID        uint                 `gorm:"not null" json:"userId"`
	User          User                 `gorm:"foreignKey:UserID" json:"-"`
	UserName      string               `gorm:"-" json:"userName"`
	CycleID       *uint                `gorm:"index" json:"cycleId"`
	TriggerType   string               `gorm:"type:varchar(20);not null" json:"triggerType"`
	TriggerUserID *uint                `json:"triggerUserId"`
	Status        string               `gorm:"type:varchar(20);not null;default:completed" json:"status"`
	FailMessage   string               `gorm:"type:text" json:"failMessage,omitempty"`
	CoinsChecked  int                  `json:"coinsChecked"`
	Available     int                  `json:"available"`
	Unavailable   int                  `json:"unavailable"`
	Unknown       int                  `json:"unknown"`
	Errors        int                  `json:"errors"`
	DurationMs    int64                `json:"durationMs"`
	StartedAt     time.Time            `gorm:"not null" json:"startedAt"`
	CompletedAt   *time.Time           `json:"completedAt"`
	Results       []AvailabilityResult `gorm:"foreignKey:RunID" json:"results,omitempty"`
	CreatedAt     time.Time            `json:"createdAt"`
}

// AvailabilityResult records the check outcome for a single coin in a run.
type AvailabilityResult struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	RunID      uint      `gorm:"not null;index" json:"runId"`
	CoinID     uint      `gorm:"not null" json:"coinId"`
	CoinName   string    `json:"coinName"`
	URL        string    `json:"url"`
	Status     string    `gorm:"type:varchar(20);not null" json:"status"`
	Reason     string    `gorm:"type:text" json:"reason"`
	HttpStatus *int      `json:"httpStatus"`
	AgentUsed  bool      `gorm:"default:false" json:"agentUsed"`
	CheckedAt  time.Time `gorm:"not null" json:"checkedAt"`
}
