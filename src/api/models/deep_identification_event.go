package models

import "time"

// DeepIdentificationEventType enumerates the sanitized, append-only event
// envelope kinds persisted for replayable SSE (data-model.md §3,
// contracts/sse-events.md).
type DeepIdentificationEventType string

const (
	DeepEventJobAccepted     DeepIdentificationEventType = "job_accepted"
	DeepEventStatusChanged   DeepIdentificationEventType = "status_changed"
	DeepEventRouterSelected  DeepIdentificationEventType = "router_selected"
	DeepEventProviderStarted DeepIdentificationEventType = "provider_started"
	DeepEventProviderResult  DeepIdentificationEventType = "provider_result"
	DeepEventEvaluation      DeepIdentificationEventType = "evaluation"
	DeepEventSynthesisStart  DeepIdentificationEventType = "synthesis_started"
	DeepEventProgress        DeepIdentificationEventType = "progress"
	DeepEventTerminal        DeepIdentificationEventType = "terminal"
)

// DeepIdentificationEvent is an append-only, per-job monotonically
// sequenced event row (data-model.md §3). No update/delete-by-id path is
// exposed by the repository - only AppendEvent, ListEventsSince, and
// PruneEventsBefore.
type DeepIdentificationEvent struct {
	ID          uint                        `gorm:"primaryKey" json:"id"`
	JobID       uint                        `gorm:"not null;index:uix_deep_events_job_seq,priority:1,unique" json:"jobId"`
	UserID      uint                        `gorm:"not null;index" json:"userId"`
	Seq         int64                       `gorm:"not null;index:uix_deep_events_job_seq,priority:2,unique" json:"seq"`
	Type        DeepIdentificationEventType `gorm:"type:varchar(32);not null" json:"type"`
	PayloadJSON string                      `gorm:"type:text" json:"payload,omitempty"`
	CreatedAt   time.Time                   `gorm:"index:idx_deep_events_created" json:"createdAt"`
}
