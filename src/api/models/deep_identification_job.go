package models

import "time"

// DeepJobStatus is the terminal/non-terminal state machine for a
// DeepIdentificationJob (data-model.md §2.1). Terminal states are one-way;
// no repository method ever transitions a terminal job back to a
// non-terminal one.
type DeepJobStatus string

const (
	DeepJobStatusQueued    DeepJobStatus = "queued"
	DeepJobStatusRunning   DeepJobStatus = "running"
	DeepJobStatusCompleted DeepJobStatus = "completed"
	DeepJobStatusPartial   DeepJobStatus = "partial"
	DeepJobStatusFailed    DeepJobStatus = "failed"
	DeepJobStatusCancelled DeepJobStatus = "cancelled"
)

// IsDeepJobTerminal reports whether status is one of the four terminal
// states (completed, partial, failed, cancelled).
func IsDeepJobTerminal(status DeepJobStatus) bool {
	switch status {
	case DeepJobStatusCompleted, DeepJobStatusPartial, DeepJobStatusFailed, DeepJobStatusCancelled:
		return true
	default:
		return false
	}
}

// DeepJobSource identifies whether a job originated from new-intake upload
// or from an existing saved coin (spec US1/US2).
type DeepJobSource string

const (
	DeepJobSourceIntake    DeepJobSource = "intake"
	DeepJobSourceSavedCoin DeepJobSource = "saved_coin"
)

// DeepIdentificationJob is the sibling job aggregate for the deep agentic
// coin identification feature (data-model.md §2). It is intentionally
// separate from models.AIJob, which remains untouched.
type DeepIdentificationJob struct {
	ID                 uint          `gorm:"primaryKey" json:"id"`
	UserID             uint          `gorm:"not null;index:idx_deep_jobs_user_status_created,priority:1;index:idx_deep_jobs_user_coin,priority:1;index:uix_deep_jobs_active_fingerprint,priority:1,unique" json:"userId"`
	User               User          `gorm:"foreignKey:UserID" json:"-"`
	CoinID             *uint         `gorm:"index:idx_deep_jobs_user_coin,priority:2" json:"coinId,omitempty"`
	Coin               *Coin         `gorm:"foreignKey:CoinID" json:"-"`
	Status             DeepJobStatus `gorm:"type:varchar(20);not null;default:'queued';index:idx_deep_jobs_user_status_created,priority:2;index:idx_deep_jobs_status_heartbeat,priority:1" json:"status"`
	Source             DeepJobSource `gorm:"type:varchar(20);not null" json:"source"`
	InputFingerprint   string        `gorm:"type:char(64);not null;index:uix_deep_jobs_active_fingerprint,priority:2,unique" json:"-"`
	Notes              string        `gorm:"type:text" json:"-"`
	RequestedProviders string        `gorm:"type:text" json:"requestedProviders,omitempty"`
	SelectedProviders  string        `gorm:"type:text" json:"selectedProviders,omitempty"`
	RouterRationale    string        `gorm:"type:text" json:"routerRationale,omitempty"`
	RetryOfJobID       *uint         `json:"retryOfJobId,omitempty"`
	RetryDepth         int           `gorm:"not null;default:0" json:"retryDepth"`
	AttemptCount       int           `gorm:"not null;default:0" json:"-"`
	CancelRequestedAt  *time.Time    `json:"cancelRequestedAt,omitempty"`
	LastSeq            int64         `gorm:"not null;default:0" json:"-"`
	HeartbeatAt        *time.Time    `gorm:"index:idx_deep_jobs_status_heartbeat,priority:2" json:"-"`
	WorkerID           string        `gorm:"type:varchar(64)" json:"-"`
	ReportJSON         string        `gorm:"type:text" json:"report,omitempty"`
	ProposalJSON       string        `gorm:"type:text" json:"proposal,omitempty"`
	PartialSuccess     bool          `gorm:"not null;default:false" json:"partialSuccess"`
	FailureCode        string        `gorm:"type:varchar(40)" json:"failureCode,omitempty"`
	FailureMessage     string        `gorm:"type:varchar(300)" json:"failureMessage,omitempty"`
	AppliedCoinID      *uint         `json:"appliedCoinId,omitempty"`
	AppliedCoinExists  bool          `gorm:"->;column:applied_coin_exists;-:migration" json:"-"`
	AppliedDraftID     *uint         `json:"appliedDraftId,omitempty"`
	AppliedAt          *time.Time    `json:"appliedAt,omitempty"`
	StartedAt          *time.Time    `json:"startedAt,omitempty"`
	CompletedAt        *time.Time    `json:"completedAt,omitempty"`
	ExpiresAt          time.Time     `gorm:"index:idx_deep_jobs_expires" json:"-"`
	EventsPrunedAt     *time.Time    `json:"-"`
	CreatedAt          time.Time     `gorm:"index:idx_deep_jobs_user_status_created,priority:3" json:"createdAt"`
	UpdatedAt          time.Time     `json:"updatedAt"`

	// ActiveKey is a stored-expression style guard column used together with
	// InputFingerprint+UserID to enforce idempotent duplicate-submit reuse
	// (data-model.md §2.3): "active" while status is queued/running, else the
	// row id (as text). It is maintained by the repository, not by GORM
	// hooks, because SQLite generated-column semantics for this pattern are
	// simplest to keep explicit and testable. The sentinel is the literal
	// string "active" (not "1") so it can never collide with a real numeric
	// job id used as the terminal-state marker.
	ActiveKey string `gorm:"type:varchar(20);not null;default:'active';index:uix_deep_jobs_active_fingerprint,priority:3,unique" json:"-"`
}

// DeepIdentificationNoExpirySentinel is the far-future retention marker for
// terminal completed/partial deep-identification jobs. It is used instead of
// nullable expires_at to keep SQLite migrations additive and non-destructive.
var DeepIdentificationNoExpirySentinel = time.Date(9999, 12, 31, 0, 0, 0, 0, time.UTC)
