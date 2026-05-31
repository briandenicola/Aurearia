package models

import "time"

type CollectionUpdateProposalStatus string

const (
	CollectionUpdateProposalPending   CollectionUpdateProposalStatus = "pending"
	CollectionUpdateProposalCommitted CollectionUpdateProposalStatus = "committed"
	CollectionUpdateProposalCancelled CollectionUpdateProposalStatus = "cancelled"
	CollectionUpdateProposalExpired   CollectionUpdateProposalStatus = "expired"
)

// CollectionUpdateProposal stores a confirm-gated chat update before commit.
type CollectionUpdateProposal struct {
	ID            string                         `gorm:"primaryKey;type:varchar(64)" json:"id"`
	UserID        uint                           `gorm:"not null;index" json:"userId"`
	CoinID        uint                           `gorm:"not null;index" json:"coinId"`
	TokenHash     string                         `gorm:"not null;index" json:"-"`
	Status        CollectionUpdateProposalStatus `gorm:"not null;type:varchar(20);index" json:"status"`
	ChangesJSON   string                         `gorm:"type:text;not null" json:"-"`
	ChangedFields string                         `gorm:"type:text;not null" json:"-"`
	ExpiresAt     time.Time                      `gorm:"not null;index" json:"expiresAt"`
	CommittedAt   *time.Time                     `json:"committedAt,omitempty"`
	CancelledAt   *time.Time                     `json:"cancelledAt,omitempty"`
	CreatedAt     time.Time                      `json:"createdAt"`
	UpdatedAt     time.Time                      `json:"updatedAt"`
}
