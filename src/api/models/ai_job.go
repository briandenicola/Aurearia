package models

import "time"

type AIJobType string

const (
	AIJobTypeAnalysis      AIJobType = "analysis"
	AIJobTypeValueEstimate AIJobType = "value_estimate"
	AIJobTypeCoinGrading   AIJobType = "coin_grading"
)

type AIJobStatus string

const (
	AIJobStatusQueued    AIJobStatus = "queued"
	AIJobStatusRunning   AIJobStatus = "running"
	AIJobStatusCompleted AIJobStatus = "completed"
	AIJobStatusFailed    AIJobStatus = "failed"
)

type AIJob struct {
	ID           uint        `gorm:"primaryKey" json:"id"`
	UserID       uint        `gorm:"not null;index:idx_ai_jobs_user_coin_type_side_status,priority:1" json:"userId"`
	User         User        `gorm:"foreignKey:UserID" json:"-"`
	CoinID       uint        `gorm:"not null;index:idx_ai_jobs_user_coin_type_side_status,priority:2" json:"coinId"`
	Coin         Coin        `gorm:"foreignKey:CoinID" json:"-"`
	JobType      AIJobType   `gorm:"type:varchar(32);not null;index:idx_ai_jobs_user_coin_type_side_status,priority:3" json:"jobType"`
	Side         string      `gorm:"type:varchar(16);index:idx_ai_jobs_user_coin_type_side_status,priority:4" json:"side,omitempty"`
	Status       AIJobStatus `gorm:"type:varchar(20);not null;default:'queued';index:idx_ai_jobs_user_coin_type_side_status,priority:5" json:"status"`
	Result       string      `gorm:"type:text" json:"result,omitempty"`
	ErrorMessage string      `gorm:"type:text" json:"errorMessage,omitempty"`
	Attempts     int         `gorm:"not null;default:0" json:"attempts"`
	StartedAt    *time.Time  `json:"startedAt,omitempty"`
	CompletedAt  *time.Time  `json:"completedAt,omitempty"`
	CreatedAt    time.Time   `json:"createdAt"`
	UpdatedAt    time.Time   `json:"updatedAt"`
}
