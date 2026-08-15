package models

import "time"

// DeepProviderName enumerates the provider identifiers used across the
// router, provider-run rows, and event payloads.
type DeepProviderName string

const (
	DeepProviderNomisma DeepProviderName = "nomisma"
	DeepProviderNumista DeepProviderName = "numista"
	DeepProviderNGC     DeepProviderName = "ngc"
	DeepProviderOCRE    DeepProviderName = "ocre"
	DeepProviderRPC     DeepProviderName = "rpc"
)

// DeepProviderRunStatus is the per-provider-attempt state machine
// (data-model.md §2.2). "not_automated" and "unavailable" are first-class
// terminal statuses and MUST NOT be rendered/synthesized as "no_match".
type DeepProviderRunStatus string

const (
	DeepProviderRunPending      DeepProviderRunStatus = "pending"
	DeepProviderRunRunning      DeepProviderRunStatus = "running"
	DeepProviderRunContributed  DeepProviderRunStatus = "contributed"
	DeepProviderRunNoMatch      DeepProviderRunStatus = "no_match"
	DeepProviderRunFailed       DeepProviderRunStatus = "failed"
	DeepProviderRunTimedOut     DeepProviderRunStatus = "timed_out"
	DeepProviderRunSkipped      DeepProviderRunStatus = "skipped"
	DeepProviderRunNotAutomated DeepProviderRunStatus = "not_automated"
	DeepProviderRunUnavailable  DeepProviderRunStatus = "unavailable"
)

// DeepIdentificationProviderRun is a single provider attempt row within a
// job (data-model.md §4). One row per provider per job; retries create a
// new job (and therefore new provider-run rows).
type DeepIdentificationProviderRun struct {
	ID          uint                  `gorm:"primaryKey" json:"id"`
	JobID       uint                  `gorm:"not null;index:uix_deep_provider_run_job_provider,priority:1,unique" json:"jobId"`
	UserID      uint                  `gorm:"not null;index" json:"userId"`
	Provider    DeepProviderName      `gorm:"type:varchar(24);not null;index:uix_deep_provider_run_job_provider,priority:2,unique" json:"provider"`
	Status      DeepProviderRunStatus `gorm:"type:varchar(20);not null;default:'pending'" json:"status"`
	Automatable bool                  `gorm:"not null;default:true" json:"automatable"`
	ClaimsJSON  string                `gorm:"type:text" json:"claims,omitempty"`
	Confidence  float64               `gorm:"not null;default:0" json:"confidence"`
	CallCount   int                   `gorm:"not null;default:0" json:"callCount"`
	LatencyMS   int                   `gorm:"not null;default:0" json:"latencyMs"`
	ErrorKind   string                `gorm:"type:varchar(32)" json:"errorKind,omitempty"`
	StartedAt   *time.Time            `json:"startedAt,omitempty"`
	CompletedAt *time.Time            `json:"completedAt,omitempty"`
	CreatedAt   time.Time             `json:"createdAt"`
	UpdatedAt   time.Time             `json:"updatedAt"`
}

// IsDeepProviderRunTerminal reports whether a provider-run status is one
// that will not transition further within the same job attempt.
func IsDeepProviderRunTerminal(status DeepProviderRunStatus) bool {
	switch status {
	case DeepProviderRunContributed, DeepProviderRunNoMatch, DeepProviderRunFailed,
		DeepProviderRunTimedOut, DeepProviderRunSkipped, DeepProviderRunNotAutomated,
		DeepProviderRunUnavailable:
		return true
	default:
		return false
	}
}
