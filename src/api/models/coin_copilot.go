package models

import "time"

type CopilotRunStatus string

const (
	CopilotRunQueued          CopilotRunStatus = "queued"
	CopilotRunRunning         CopilotRunStatus = "running"
	CopilotRunPaused          CopilotRunStatus = "paused"
	CopilotRunCancelRequested CopilotRunStatus = "cancel_requested"
	CopilotRunCompleted       CopilotRunStatus = "completed"
	CopilotRunFailed          CopilotRunStatus = "failed"
	CopilotRunCancelled       CopilotRunStatus = "cancelled"
)

func IsCopilotRunTerminal(status CopilotRunStatus) bool {
	return status == CopilotRunCompleted || status == CopilotRunFailed || status == CopilotRunCancelled
}

type CopilotEventType string

const (
	CopilotEventRunStarted            CopilotEventType = "run_started"
	CopilotEventPlanUpdated           CopilotEventType = "plan_updated"
	CopilotEventToolStarted           CopilotEventType = "tool_started"
	CopilotEventToolCompleted         CopilotEventType = "tool_completed"
	CopilotEventClarificationRequired CopilotEventType = "clarification_required"
	CopilotEventRunPaused             CopilotEventType = "run_paused"
	CopilotEventRunResumed            CopilotEventType = "run_resumed"
	CopilotEventRunCancelled          CopilotEventType = "run_cancelled"
	CopilotEventRunCompleted          CopilotEventType = "run_completed"
	CopilotEventRunFailed             CopilotEventType = "run_failed"
)

func IsCopilotEventType(value CopilotEventType) bool {
	switch value {
	case CopilotEventRunStarted, CopilotEventPlanUpdated, CopilotEventToolStarted,
		CopilotEventToolCompleted, CopilotEventClarificationRequired, CopilotEventRunPaused,
		CopilotEventRunResumed, CopilotEventRunCancelled, CopilotEventRunCompleted,
		CopilotEventRunFailed:
		return true
	default:
		return false
	}
}

type CoinCopilotThread struct {
	ID        string    `gorm:"type:varchar(80);primaryKey" json:"id"`
	UserID    uint      `gorm:"not null;index:idx_copilot_threads_user_updated,priority:1" json:"-"`
	Title     string    `gorm:"type:varchar(200);not null" json:"title"`
	LastRunID *string   `gorm:"type:varchar(80)" json:"lastRunId,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `gorm:"index:idx_copilot_threads_user_updated,priority:2" json:"updatedAt"`
}

type CoinCopilotRun struct {
	ID                          string           `gorm:"type:varchar(80);primaryKey" json:"id"`
	ThreadID                    string           `gorm:"type:varchar(80);not null;index:idx_copilot_runs_thread_created,priority:1" json:"threadId"`
	UserID                      uint             `gorm:"not null;index:idx_copilot_runs_user_status_created,priority:1;index:uix_copilot_start_key,priority:1,unique" json:"-"`
	Status                      CopilotRunStatus `gorm:"type:varchar(24);not null;index:idx_copilot_runs_user_status_created,priority:2;index:idx_copilot_runs_status_heartbeat,priority:1" json:"status"`
	Goal                        string           `gorm:"type:text;not null" json:"goal"`
	AppContextJSON              string           `gorm:"type:text" json:"-"`
	StartIdempotencyKeyHash     string           `gorm:"type:char(64);not null;index:uix_copilot_start_key,priority:2,unique" json:"-"`
	StartRequestFingerprint     string           `gorm:"type:char(64);not null" json:"-"`
	ExecutionID                 string           `gorm:"type:varchar(80)" json:"-"`
	ExecutionAttempt            int              `gorm:"not null;default:0" json:"attempt"`
	CheckpointVersion           int64            `gorm:"not null;default:0" json:"checkpointVersion"`
	LastSeq                     int64            `gorm:"not null;default:0" json:"lastSeq"`
	CancelRequestedAt           *time.Time       `json:"cancelRequestedAt,omitempty"`
	PausedAt                    *time.Time       `json:"pausedAt,omitempty"`
	HeartbeatAt                 *time.Time       `gorm:"index:idx_copilot_runs_status_heartbeat,priority:2" json:"-"`
	WorkerID                    string           `gorm:"type:varchar(64)" json:"-"`
	FailureCode                 string           `gorm:"type:varchar(48)" json:"failureCode,omitempty"`
	FailureMessage              string           `gorm:"type:varchar(300)" json:"failureMessage,omitempty"`
	FinalAnswer                 string           `gorm:"type:text" json:"finalAnswer,omitempty"`
	IterationCount              int              `gorm:"not null;default:0" json:"-"`
	ToolCallCount               int              `gorm:"not null;default:0" json:"-"`
	InputTokens                 int64            `gorm:"not null;default:0" json:"-"`
	OutputTokens                int64            `gorm:"not null;default:0" json:"-"`
	MaxIterations               int              `gorm:"not null" json:"-"`
	MaxToolCalls                int              `gorm:"not null" json:"-"`
	MaxConcurrentTools          int              `gorm:"not null;default:1" json:"-"`
	HardTimeoutSeconds          int              `gorm:"not null" json:"-"`
	MaxPersistedToolResultBytes int              `gorm:"not null" json:"-"`
	ResumeDeadline              *time.Time       `json:"resumeDeadline,omitempty"`
	EventsPrunedAt              *time.Time       `json:"-"`
	CheckpointsPrunedAt         *time.Time       `json:"-"`
	StartedAt                   *time.Time       `json:"startedAt,omitempty"`
	ExecutionStartedAt          *time.Time       `json:"-"`
	CompletedAt                 *time.Time       `json:"completedAt,omitempty"`
	CreatedAt                   time.Time        `gorm:"index:idx_copilot_runs_user_status_created,priority:3;index:idx_copilot_runs_thread_created,priority:2" json:"createdAt"`
	UpdatedAt                   time.Time        `json:"updatedAt"`
}

type CoinCopilotCheckpoint struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	RunID       string    `gorm:"type:varchar(80);not null;index:uix_copilot_checkpoints_run_version,priority:1,unique" json:"runId"`
	ThreadID    string    `gorm:"type:varchar(80);not null;index" json:"threadId"`
	UserID      uint      `gorm:"not null;index" json:"-"`
	ExecutionID string    `gorm:"type:varchar(80);not null" json:"executionId"`
	Version     int64     `gorm:"not null;index:uix_copilot_checkpoints_run_version,priority:2,unique" json:"version"`
	StateJSON   string    `gorm:"type:text;not null" json:"state"`
	StateDigest string    `gorm:"type:char(64);not null" json:"stateDigest"`
	CreatedAt   time.Time `gorm:"index" json:"createdAt"`
}

type CoinCopilotEvent struct {
	ID          uint             `gorm:"primaryKey" json:"id"`
	RunID       string           `gorm:"type:varchar(80);not null;index:uix_copilot_events_run_seq,priority:1,unique" json:"runId"`
	ThreadID    string           `gorm:"type:varchar(80);not null;index" json:"threadId"`
	UserID      uint             `gorm:"not null;index" json:"-"`
	ExecutionID string           `gorm:"type:varchar(80)" json:"executionId"`
	Seq         int64            `gorm:"not null;index:uix_copilot_events_run_seq,priority:2,unique" json:"seq"`
	Type        CopilotEventType `gorm:"type:varchar(40);not null" json:"type"`
	PayloadJSON string           `gorm:"type:text;not null" json:"payload"`
	CreatedAt   time.Time        `gorm:"index:idx_copilot_events_created" json:"createdAt"`
}

type CoinCopilotResumeRequest struct {
	ID                        uint      `gorm:"primaryKey" json:"id"`
	RunID                     string    `gorm:"type:varchar(80);not null;index:uix_copilot_resume_key,priority:1,unique" json:"runId"`
	UserID                    uint      `gorm:"not null;index" json:"-"`
	IdempotencyKeyHash        string    `gorm:"type:char(64);not null;index:uix_copilot_resume_key,priority:2,unique" json:"-"`
	RequestFingerprint        string    `gorm:"type:char(64);not null" json:"-"`
	AcceptedCheckpointVersion int64     `gorm:"not null" json:"acceptedCheckpointVersion"`
	ExecutionID               string    `gorm:"type:varchar(80);not null" json:"executionId"`
	CreatedAt                 time.Time `gorm:"index" json:"createdAt"`
}

type CoinCopilotDeepHandoffOperation string

const (
	CoinCopilotDeepHandoffOperationRequest CoinCopilotDeepHandoffOperation = "request"
	CoinCopilotDeepHandoffOperationRerun   CoinCopilotDeepHandoffOperation = "rerun"
)

type CoinCopilotDeepHandoffTargetKind string

const (
	CoinCopilotDeepHandoffTargetCoin  CoinCopilotDeepHandoffTargetKind = "coin"
	CoinCopilotDeepHandoffTargetDraft CoinCopilotDeepHandoffTargetKind = "draft"
)

type CoinCopilotDeepHandoffOutcome string

const (
	CoinCopilotDeepHandoffOutcomeCreated      CoinCopilotDeepHandoffOutcome = "created"
	CoinCopilotDeepHandoffOutcomeReusedActive CoinCopilotDeepHandoffOutcome = "reused_active"
	CoinCopilotDeepHandoffOutcomeReusedResult CoinCopilotDeepHandoffOutcome = "reused_result"
)

// CoinCopilotDeepHandoff is the durable idempotency/admission binding between
// one authorized Copilot execution and the existing Deep Analysis engine.
// DeepJobID is the selected result job; PriorJobID is solely the explicit
// rerun input and is never overloaded as the result.
type CoinCopilotDeepHandoff struct {
	ID                        uint                             `gorm:"primaryKey" json:"id"`
	UserID                    uint                             `gorm:"not null;index:uix_copilot_deep_handoff_key,priority:1,unique;index:idx_copilot_deep_handoff_owner_job,priority:1" json:"-"`
	RunID                     string                           `gorm:"type:varchar(80);not null;index:uix_copilot_deep_handoff_key,priority:2,unique" json:"runId"`
	ExecutionID               string                           `gorm:"type:varchar(80);not null" json:"executionId"`
	HandoffKeyHash            string                           `gorm:"type:char(64);not null;index:uix_copilot_deep_handoff_key,priority:3,unique" json:"-"`
	RequestFingerprint        string                           `gorm:"type:char(64);not null" json:"-"`
	ExpectedCheckpointVersion int64                            `gorm:"not null" json:"expectedCheckpointVersion"`
	AppContextDigest          string                           `gorm:"type:char(64);not null" json:"-"`
	Operation                 CoinCopilotDeepHandoffOperation  `gorm:"type:varchar(12);not null" json:"operation"`
	TargetKind                CoinCopilotDeepHandoffTargetKind `gorm:"type:varchar(12);not null" json:"targetKind"`
	TargetID                  uint                             `gorm:"not null" json:"targetId"`
	PriorJobID                *uint                            `gorm:"index" json:"priorJobId,omitempty"`
	TargetSnapshotFingerprint string                           `gorm:"type:char(64);not null" json:"-"`
	DeepJobID                 uint                             `gorm:"not null;index:idx_copilot_deep_handoff_owner_job,priority:2;index:idx_copilot_deep_handoff_job" json:"deepJobId"`
	AdmissionOutcome          CoinCopilotDeepHandoffOutcome    `gorm:"type:varchar(24);not null" json:"admissionOutcome"`
	CreatedAt                 time.Time                        `json:"createdAt"`
	UpdatedAt                 time.Time                        `json:"updatedAt"`
}

func IsValidCoinCopilotDeepHandoff(handoff *CoinCopilotDeepHandoff) bool {
	if handoff == nil || handoff.UserID == 0 || handoff.RunID == "" ||
		handoff.ExecutionID == "" || handoff.HandoffKeyHash == "" ||
		handoff.RequestFingerprint == "" || handoff.ExpectedCheckpointVersion < 0 ||
		handoff.AppContextDigest == "" || handoff.TargetID == 0 ||
		handoff.TargetSnapshotFingerprint == "" || handoff.DeepJobID == 0 {
		return false
	}
	if handoff.TargetKind != CoinCopilotDeepHandoffTargetCoin &&
		handoff.TargetKind != CoinCopilotDeepHandoffTargetDraft {
		return false
	}
	switch handoff.AdmissionOutcome {
	case CoinCopilotDeepHandoffOutcomeCreated,
		CoinCopilotDeepHandoffOutcomeReusedActive,
		CoinCopilotDeepHandoffOutcomeReusedResult:
	default:
		return false
	}
	switch handoff.Operation {
	case CoinCopilotDeepHandoffOperationRequest:
		return handoff.PriorJobID == nil
	case CoinCopilotDeepHandoffOperationRerun:
		return handoff.PriorJobID != nil && *handoff.PriorJobID > 0 &&
			*handoff.PriorJobID != handoff.DeepJobID
	default:
		return false
	}
}
