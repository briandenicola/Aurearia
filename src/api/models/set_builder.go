package models

import "time"

type SetBuilderRunStatus string

const (
	SetBuilderRunStatusQueued    SetBuilderRunStatus = "queued"
	SetBuilderRunStatusRunning   SetBuilderRunStatus = "running"
	SetBuilderRunStatusCompleted SetBuilderRunStatus = "completed"
	SetBuilderRunStatusFailed    SetBuilderRunStatus = "failed"
)

type SetProposalStatus string

const (
	SetProposalStatusPending        SetProposalStatus = "pending"
	SetProposalStatusApproved       SetProposalStatus = "approved"
	SetProposalStatusRejected       SetProposalStatus = "rejected"
	SetProposalStatusExpired        SetProposalStatus = "expired"
	SetProposalStatusCreationFailed SetProposalStatus = "creation_failed"
)

type ProposalSlotVerificationStatus string

const (
	ProposalSlotVerificationVerified   ProposalSlotVerificationStatus = "verified"
	ProposalSlotVerificationUnverified ProposalSlotVerificationStatus = "unverified"
)

// SetBuilderRun records one Python agent workflow execution for Agentic set building.
type SetBuilderRun struct {
	ID                uint                `gorm:"primaryKey" json:"id"`
	UserID            uint                `gorm:"not null;index" json:"userId"`
	User              User                `gorm:"foreignKey:UserID" json:"-"`
	Prompt            string              `gorm:"type:text;not null" json:"prompt"`
	Status            SetBuilderRunStatus `gorm:"type:varchar(20);not null;default:'queued';index" json:"status"`
	Provider          string              `gorm:"type:varchar(50)" json:"provider,omitempty"`
	Model             string              `gorm:"type:varchar(100)" json:"model,omitempty"`
	Feedback          string              `gorm:"type:text" json:"feedback,omitempty"`
	TranscriptSummary string              `gorm:"type:text" json:"transcriptSummary,omitempty"`
	ErrorMessage      string              `gorm:"type:text" json:"errorMessage,omitempty"`
	TerminationReason string              `gorm:"type:text" json:"terminationReason,omitempty"`
	MaxTurns          *int                `json:"maxTurns,omitempty"`
	UsedTurns         *int                `json:"usedTurns,omitempty"`
	TokenBudget       *int                `json:"tokenBudget,omitempty"`
	TokensUsed        *int                `json:"tokensUsed,omitempty"`
	StartedAt         *time.Time          `json:"startedAt,omitempty"`
	CompletedAt       *time.Time          `json:"completedAt,omitempty"`
	CreatedAt         time.Time           `json:"createdAt"`
	UpdatedAt         time.Time           `json:"updatedAt"`
}

// SetProposal stores the human-reviewable output of an Agentic set builder run.
type SetProposal struct {
	ID              uint              `gorm:"primaryKey" json:"id"`
	UserID          uint              `gorm:"not null;index:idx_set_proposals_user_status,priority:1;index:idx_set_proposals_user_idempotency,priority:1" json:"userId"`
	User            User              `gorm:"foreignKey:UserID" json:"-"`
	BuilderRunID    uint              `gorm:"not null;index" json:"builderRunId"`
	Run             SetBuilderRun     `gorm:"foreignKey:BuilderRunID" json:"run,omitempty"`
	OriginalPrompt  string            `gorm:"type:text;not null" json:"originalPrompt"`
	Status          SetProposalStatus `gorm:"type:varchar(20);not null;default:'pending';index:idx_set_proposals_user_status,priority:2" json:"status"`
	ProposedName    string            `gorm:"type:varchar(80);not null" json:"proposedName"`
	ProposedSlug    string            `gorm:"type:varchar(100)" json:"proposedSlug,omitempty"`
	Description     string            `gorm:"type:text" json:"description"`
	Color           string            `gorm:"type:varchar(7)" json:"color"`
	SelectedScope   string            `gorm:"type:text" json:"selectedScope,omitempty"`
	ScopeOptions    *JSONObject       `gorm:"type:text" json:"scopeOptions,omitempty"`
	RosterPayload   *JSONObject       `gorm:"type:text" json:"rosterPayload,omitempty"`
	PreMatchSummary *JSONObject       `gorm:"type:text" json:"preMatchSummary,omitempty"`
	IdempotencyKey  string            `gorm:"type:varchar(128);not null;index:idx_set_proposals_user_idempotency,priority:2" json:"idempotencyKey"`
	ExpiresAt       time.Time         `gorm:"not null;index" json:"expiresAt"`
	ApprovedAt      *time.Time        `json:"approvedAt,omitempty"`
	RejectedAt      *time.Time        `json:"rejectedAt,omitempty"`
	RejectionReason string            `gorm:"type:text" json:"rejectionReason,omitempty"`
	ApprovalSetID   *uint             `gorm:"index" json:"approvalSetId,omitempty"`
	ErrorMessage    string            `gorm:"type:text" json:"errorMessage,omitempty"`
	Slots           []ProposalSlot    `gorm:"foreignKey:ProposalID" json:"slots,omitempty"`
	CreatedAt       time.Time         `json:"createdAt"`
	UpdatedAt       time.Time         `json:"updatedAt"`
}

// ProposalSlot is one proposed roster slot that can become a set target after approval.
type ProposalSlot struct {
	ID                 uint                           `gorm:"primaryKey" json:"id"`
	ProposalID         uint                           `gorm:"not null;index" json:"proposalId"`
	Label              string                         `gorm:"not null" json:"label"`
	Criteria           *JSONObject                    `gorm:"type:text" json:"criteria,omitempty"`
	GroupName          string                         `gorm:"type:varchar(100)" json:"group,omitempty"`
	SortOrder          int                            `gorm:"not null" json:"sortOrder"`
	VerificationStatus ProposalSlotVerificationStatus `gorm:"type:varchar(20);not null;default:'verified'" json:"verificationStatus"`
	SourceNote         string                         `gorm:"type:text" json:"sourceNote,omitempty"`
	ValidationNote     string                         `gorm:"type:text" json:"validationNote,omitempty"`
	CreatedAt          time.Time                      `json:"createdAt"`
	UpdatedAt          time.Time                      `json:"updatedAt"`
}
