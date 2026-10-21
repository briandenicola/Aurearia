package models

import "time"

// CoinCopilotRunGrade is the rubric judge's verdict on one completed Coin
// Copilot run. Question and answer stay on the run; a grade is deleted with
// its conversation.
type CoinCopilotRunGrade struct {
	ID                uint       `gorm:"primaryKey" json:"id"`
	RunID             string     `gorm:"type:varchar(80);not null;uniqueIndex:uix_copilot_grades_run" json:"runId"`
	ThreadID          string     `gorm:"type:varchar(80);not null;index" json:"threadId"`
	UserID            uint       `gorm:"not null;index" json:"-"`
	Quality           float64    `gorm:"not null;index" json:"quality"`
	AnswersDirectly   bool       `gorm:"not null" json:"answersDirectly"`
	Grounded          bool       `gorm:"not null" json:"grounded"`
	Accurate          bool       `gorm:"not null" json:"accurate"`
	CitesSources      string     `gorm:"type:varchar(16);not null" json:"citesSources"`
	NoOfferMenu       bool       `gorm:"not null" json:"noOfferMenu"`
	NoFalseInability  bool       `gorm:"not null" json:"noFalseInability"`
	Explanation       string     `gorm:"type:text;not null" json:"explanation"`
	JudgeModel        string     `gorm:"type:varchar(100);not null" json:"judgeModel"`
	JudgeInputTokens  int64      `gorm:"not null;default:0" json:"judgeInputTokens"`
	JudgeOutputTokens int64      `gorm:"not null;default:0" json:"judgeOutputTokens"`
	Disputed          bool       `gorm:"not null;default:false;index" json:"disputed"`
	DisputeNote       string     `gorm:"type:varchar(500)" json:"disputeNote,omitempty"`
	DisputedAt        *time.Time `json:"disputedAt,omitempty"`
	GradedAt          time.Time  `gorm:"not null;index" json:"gradedAt"`
}
