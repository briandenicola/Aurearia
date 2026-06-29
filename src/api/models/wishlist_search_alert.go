package models

import (
	"time"
)

type WishlistAlertCadence string

const (
	WishlistAlertCadenceManual  WishlistAlertCadence = "manual"
	WishlistAlertCadenceDaily   WishlistAlertCadence = "daily"
	WishlistAlertCadenceWeekly  WishlistAlertCadence = "weekly"
	WishlistAlertCadenceMonthly WishlistAlertCadence = "monthly"
)

type AlertRunTriggerType string

const (
	AlertRunTriggerManual    AlertRunTriggerType = "manual"
	AlertRunTriggerScheduled AlertRunTriggerType = "scheduled"
)

type AlertRunStatus string

const (
	AlertRunStatusQueued      AlertRunStatus = "queued"
	AlertRunStatusRunning     AlertRunStatus = "running"
	AlertRunStatusCompleted   AlertRunStatus = "completed"
	AlertRunStatusFailed      AlertRunStatus = "failed"
	AlertRunStatusPartial     AlertRunStatus = "partial"
	AlertRunStatusRateLimited AlertRunStatus = "rate_limited"
	AlertRunStatusCancelled   AlertRunStatus = "cancelled"
)

type CandidateProvenanceStatus string

const (
	CandidateProvenanceVerified   CandidateProvenanceStatus = "verified"
	CandidateProvenancePartial    CandidateProvenanceStatus = "partial"
	CandidateProvenanceUnverified CandidateProvenanceStatus = "unverified"
)

type AlertCandidateState string

const (
	AlertCandidateStateActive      AlertCandidateState = "active"
	AlertCandidateStateDismissed   AlertCandidateState = "dismissed"
	AlertCandidateStateConverted   AlertCandidateState = "converted"
	AlertCandidateStateSuppressed  AlertCandidateState = "suppressed"
	AlertCandidateStateNeedsReview AlertCandidateState = "needs_review"
)

type CandidateReviewActionType string

const (
	CandidateReviewDismissed                  CandidateReviewActionType = "dismissed"
	CandidateReviewRestored                   CandidateReviewActionType = "restored"
	CandidateReviewConverted                  CandidateReviewActionType = "converted"
	CandidateReviewCriteriaAdjusted           CandidateReviewActionType = "criteria_adjusted"
	CandidateReviewDuplicateWarningAcked      CandidateReviewActionType = "duplicate_warning_acknowledged"
	CandidateReviewAvailabilitySeparationNote CandidateReviewActionType = "availability_separation_note"
)

type WishlistSearchAlert struct {
	ID               uint                 `gorm:"primaryKey" json:"id"`
	UserID           uint                 `gorm:"not null;index" json:"userId"`
	User             User                 `gorm:"foreignKey:UserID" json:"-"`
	Name             string               `gorm:"not null;size:200" json:"name"`
	RulerOrIssuer    string               `gorm:"size:200" json:"rulerOrIssuer"`
	CoinType         string               `gorm:"size:200" json:"coinType"`
	DateFrom         *int                 `json:"dateFrom"`
	DateTo           *int                 `json:"dateTo"`
	Mint             string               `gorm:"size:200" json:"mint"`
	Material         string               `gorm:"size:100" json:"material"`
	GradeOrCondition string               `gorm:"size:200" json:"gradeOrCondition"`
	PriceMin         *float64             `json:"priceMin"`
	PriceMax         *float64             `json:"priceMax"`
	Currency         string               `gorm:"size:3;default:'USD'" json:"currency"`
	DealerPreference string               `gorm:"size:500" json:"dealerPreference"`
	SourceFilters    StringList           `gorm:"type:text" json:"sourceFilters"`
	Keywords         string               `gorm:"size:500" json:"keywords"`
	Notes            string               `gorm:"type:text" json:"notes"`
	Cadence          WishlistAlertCadence `gorm:"type:varchar(20);not null;default:'manual';index" json:"cadence"`
	IsActive         bool                 `gorm:"not null;default:true;index" json:"isActive"`
	LastRunAt        *time.Time           `json:"lastRunAt"`
	Runs             []AlertRun           `gorm:"foreignKey:AlertID" json:"-"`
	Candidates       []AlertCandidate     `gorm:"foreignKey:AlertID" json:"-"`
	CreatedAt        time.Time            `json:"createdAt"`
	UpdatedAt        time.Time            `json:"updatedAt"`
	DeletedAt        *time.Time           `gorm:"index" json:"-"`
}

type AlertRun struct {
	ID               uint                `gorm:"primaryKey" json:"id"`
	AlertID          uint                `gorm:"not null;index" json:"alertId"`
	Alert            WishlistSearchAlert `gorm:"foreignKey:AlertID" json:"-"`
	UserID           uint                `gorm:"not null;index" json:"userId"`
	User             User                `gorm:"foreignKey:UserID" json:"-"`
	TriggerType      AlertRunTriggerType `gorm:"type:varchar(20);not null" json:"triggerType"`
	Status           AlertRunStatus      `gorm:"type:varchar(20);not null;index" json:"status"`
	StartedAt        time.Time           `gorm:"not null" json:"startedAt"`
	CompletedAt      *time.Time          `json:"completedAt"`
	DurationMs       int64               `json:"durationMs"`
	CriteriaSnapshot string              `gorm:"type:text;not null" json:"criteriaSnapshot"`
	ResultCount      int                 `json:"resultCount"`
	NewCount         int                 `json:"newCount"`
	DuplicateCount   int                 `json:"duplicateCount"`
	DismissedCount   int                 `json:"dismissedCount"`
	PartialWarnings  StringList          `gorm:"type:text" json:"partialWarnings"`
	ErrorMessage     string              `gorm:"type:text" json:"errorMessage"`
	RateLimitStatus  string              `gorm:"size:50;default:'ok'" json:"rateLimitStatus"`
	Candidates       []AlertCandidate    `gorm:"foreignKey:RunID" json:"candidates,omitempty"`
	CreatedAt        time.Time           `json:"createdAt"`
}

type AlertCandidate struct {
	ID                     uint                      `gorm:"primaryKey" json:"id"`
	UserID                 uint                      `gorm:"not null;index" json:"userId"`
	User                   User                      `gorm:"foreignKey:UserID" json:"-"`
	AlertID                uint                      `gorm:"not null;index" json:"alertId"`
	Alert                  WishlistSearchAlert       `gorm:"foreignKey:AlertID" json:"-"`
	RunID                  uint                      `gorm:"not null;index" json:"runId"`
	Run                    AlertRun                  `gorm:"foreignKey:RunID" json:"-"`
	SourceURL              string                    `gorm:"not null;size:2000" json:"sourceUrl"`
	CanonicalSourceURL     string                    `gorm:"size:2000;index" json:"canonicalSourceUrl"`
	SourceName             string                    `gorm:"size:500" json:"sourceName"`
	Title                  string                    `gorm:"not null;size:500" json:"title"`
	NormalizedTitle        string                    `gorm:"not null;size:500;index" json:"normalizedTitle"`
	ObservedPrice          *float64                  `json:"observedPrice"`
	ObservedCurrency       string                    `gorm:"size:3" json:"observedCurrency"`
	ReasonForMatch         string                    `gorm:"type:text;not null" json:"reasonForMatch"`
	Fields                 StringMap                 `gorm:"type:text" json:"fields"`
	LastSeenAt             time.Time                 `gorm:"not null;index" json:"lastSeenAt"`
	FirstSeenAt            time.Time                 `gorm:"not null" json:"firstSeenAt"`
	ProvenanceStatus       CandidateProvenanceStatus `gorm:"type:varchar(20);not null;index" json:"provenanceStatus"`
	LifecycleState         AlertCandidateState       `gorm:"type:varchar(20);not null;index" json:"lifecycleState"`
	DuplicateKey           string                    `gorm:"not null;size:128;uniqueIndex" json:"duplicateKey"`
	DuplicateOfCandidateID *uint                     `json:"duplicateOfCandidateId"`
	MatchingWishlistCoinID *uint                     `json:"matchingWishlistCoinId"`
	ConvertedCoinID        *uint                     `json:"convertedCoinId"`
	DismissalReason        string                    `gorm:"size:100" json:"dismissalReason"`
	Provenance             []CandidateProvenance     `gorm:"foreignKey:CandidateID" json:"provenance,omitempty"`
	ReviewActions          []CandidateReviewAction   `gorm:"foreignKey:CandidateID" json:"-"`
	CreatedAt              time.Time                 `json:"createdAt"`
	UpdatedAt              time.Time                 `json:"updatedAt"`
}

type CandidateProvenance struct {
	ID                uint                      `gorm:"primaryKey" json:"id"`
	CandidateID       uint                      `gorm:"not null;index" json:"candidateId"`
	Candidate         AlertCandidate            `gorm:"foreignKey:CandidateID" json:"-"`
	Field             string                    `gorm:"not null;size:100" json:"field"`
	Value             string                    `gorm:"type:text;not null" json:"value"`
	SourceURL         string                    `gorm:"not null;size:2000" json:"sourceUrl"`
	ObservedAt        time.Time                 `gorm:"not null" json:"observedAt"`
	Confidence        string                    `gorm:"not null;size:20" json:"confidence"`
	VerificationState CandidateProvenanceStatus `gorm:"type:varchar(20);not null" json:"verificationState"`
	Notes             string                    `gorm:"type:text" json:"notes"`
}

type CandidateReviewAction struct {
	ID          uint                      `gorm:"primaryKey" json:"id"`
	CandidateID uint                      `gorm:"not null;index" json:"candidateId"`
	Candidate   AlertCandidate            `gorm:"foreignKey:CandidateID" json:"-"`
	UserID      uint                      `gorm:"not null;index" json:"userId"`
	User        User                      `gorm:"foreignKey:UserID" json:"-"`
	Action      CandidateReviewActionType `gorm:"type:varchar(50);not null" json:"action"`
	Reason      string                    `gorm:"size:100" json:"reason"`
	Metadata    string                    `gorm:"type:text" json:"metadata"`
	CreatedAt   time.Time                 `json:"createdAt"`
}
