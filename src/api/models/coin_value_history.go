package models

import "time"

// Source values for CoinValueHistory.Source
const (
	ValueHistorySourceManual      = "manual"
	ValueHistorySourceAIScheduled = "ai_scheduled"
	ValueHistorySourceAIEstimate  = "ai_estimate"
)

type CoinValueHistory struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	CoinID     uint      `gorm:"not null;index" json:"coinId"`
	UserID     uint      `gorm:"not null;index" json:"userId"`
	Value      float64   `gorm:"not null" json:"value"`
	Confidence string    `gorm:"type:varchar(20);not null" json:"confidence"`
	// Source identifies the origin of this value entry: manual, ai_scheduled, or ai_estimate.
	// Legacy rows with no source default to 'manual' via backfill in database.go.
	Source     string    `gorm:"type:varchar(20);not null;default:'manual'" json:"source"`
	RecordedAt time.Time `gorm:"not null;index" json:"recordedAt"`
}
