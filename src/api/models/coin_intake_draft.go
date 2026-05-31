package models

import "time"

const (
	CoinIntakeDraftStatusDrafted   = "drafted"
	CoinIntakeDraftStatusConfirmed = "confirmed"
	CoinIntakeDraftStatusDiscarded = "discarded"
	CoinIntakeDraftStatusExpired   = "expired"
)

// CoinIntakeDraft stores an AI-generated candidate payload before explicit user confirmation.
type CoinIntakeDraft struct {
	ID                uint       `gorm:"primaryKey" json:"id"`
	UserID            uint       `gorm:"not null;index" json:"userId"`
	DraftPayload      string     `gorm:"type:text;not null" json:"draftPayload"`
	ConfidenceSummary string     `gorm:"type:text;not null" json:"confidenceSummary"`
	Evidence          string     `gorm:"type:text;not null" json:"evidence"`
	UnresolvedFields  string     `gorm:"type:text" json:"unresolvedFields"`
	Status            string     `gorm:"type:varchar(20);not null;index" json:"status"`
	ExpiresAt         time.Time  `gorm:"not null;index" json:"expiresAt"`
	ConfirmedAt       *time.Time `json:"confirmedAt"`
	ConfirmedCoinID   *uint      `gorm:"index" json:"confirmedCoinId"`
	CreatedAt         time.Time  `json:"createdAt"`
	UpdatedAt         time.Time  `json:"updatedAt"`
}
