package models

import "time"

// ValuationRun records a single execution of the scheduled collection valuation.
type ValuationRun struct {
	ID            uint              `gorm:"primaryKey" json:"id"`
	UserID        uint              `gorm:"not null" json:"userId"`
	User          User              `gorm:"foreignKey:UserID" json:"-"`
	TriggerType   string            `gorm:"type:varchar(20);not null" json:"triggerType"`
	TriggerUserID *uint             `json:"triggerUserId"`
	Status        string            `gorm:"type:varchar(20);not null;default:'running'" json:"status"`
	TotalCoins    int               `json:"totalCoins"`
	CoinsChecked  int               `json:"coinsChecked"`
	CoinsUpdated  int               `json:"coinsUpdated"`
	CoinsSkipped  int               `json:"coinsSkipped"`
	Errors        int               `json:"errors"`
	DurationMs    int64             `json:"durationMs"`
	StartedAt     time.Time         `gorm:"not null" json:"startedAt"`
	CompletedAt   *time.Time        `json:"completedAt"`
	ErrorMessage  string            `gorm:"type:text" json:"errorMessage,omitempty"`
	Results       []ValuationResult `gorm:"foreignKey:RunID" json:"results,omitempty"`
	CreatedAt     time.Time         `json:"createdAt"`
}

// ValuationResult records the valuation outcome for a single coin in a run.
type ValuationResult struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	RunID          uint      `gorm:"not null;index" json:"runId"`
	CoinID         uint      `gorm:"not null" json:"coinId"`
	CoinName       string    `json:"coinName"`
	PreviousValue  *float64  `json:"previousValue"`
	EstimatedValue float64   `json:"estimatedValue"`
	Confidence     string    `gorm:"type:varchar(20)" json:"confidence"`
	Reasoning      string    `gorm:"type:text" json:"reasoning"`
	Status         string    `gorm:"type:varchar(20);not null" json:"status"`
	ErrorMessage   string    `gorm:"type:text" json:"errorMessage,omitempty"`
	CheckedAt      time.Time `gorm:"not null" json:"checkedAt"`
}
