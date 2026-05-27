package models

import "time"

// AvailabilityRun records a single execution of the wishlist availability checker.
type AvailabilityRun struct {
	ID            uint                 `gorm:"primaryKey" json:"id"`
	UserID        uint                 `gorm:"not null" json:"userId"`
	User          User                 `gorm:"foreignKey:UserID" json:"-"`
	UserName      string               `gorm:"-" json:"userName"`
	TriggerType   string               `gorm:"type:varchar(20);not null" json:"triggerType"`
	TriggerUserID *uint                `json:"triggerUserId"`
	CoinsChecked  int                  `json:"coinsChecked"`
	Available     int                  `json:"available"`
	Unavailable   int                  `json:"unavailable"`
	Unknown       int                  `json:"unknown"`
	Errors        int                  `json:"errors"`
	DurationMs    int64                `json:"durationMs"`
	StartedAt     time.Time            `gorm:"not null" json:"startedAt"`
	CompletedAt   *time.Time           `json:"completedAt"`
	Results       []AvailabilityResult `gorm:"foreignKey:RunID" json:"results,omitempty"`
	CreatedAt     time.Time            `json:"createdAt"`
}

// AvailabilityResult records the check outcome for a single coin in a run.
type AvailabilityResult struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	RunID      uint      `gorm:"not null;index" json:"runId"`
	CoinID     uint      `gorm:"not null" json:"coinId"`
	CoinName   string    `json:"coinName"`
	URL        string    `json:"url"`
	Status     string    `gorm:"type:varchar(20);not null" json:"status"`
	Reason     string    `gorm:"type:text" json:"reason"`
	HttpStatus *int      `json:"httpStatus"`
	AgentUsed  bool      `gorm:"default:false" json:"agentUsed"`
	CheckedAt  time.Time `gorm:"not null" json:"checkedAt"`
}
