package models

import "time"

type RecommendationTargetType string

const (
	RecommendationTargetTypeSet RecommendationTargetType = "set"
	RecommendationTargetTypeTag RecommendationTargetType = "tag"
)

type RecommendationStatus string

const (
	RecommendationStatusPending   RecommendationStatus = "pending"
	RecommendationStatusAccepted  RecommendationStatus = "accepted"
	RecommendationStatusRejected  RecommendationStatus = "rejected"
	RecommendationStatusDismissed RecommendationStatus = "dismissed"
)

type CoinRecommendation struct {
	ID         uint                     `gorm:"primaryKey" json:"id"`
	UserID     uint                     `gorm:"not null;index;uniqueIndex:idx_reco_target" json:"userId"`
	CoinID     uint                     `gorm:"not null;index;uniqueIndex:idx_reco_target" json:"coinId"`
	TargetType RecommendationTargetType `gorm:"type:varchar(20);not null;uniqueIndex:idx_reco_target" json:"targetType"`
	TargetID   uint                     `gorm:"not null;uniqueIndex:idx_reco_target" json:"targetId"`
	Score      float64                  `gorm:"not null;default:0" json:"score"`
	Confidence string                   `gorm:"type:varchar(20);not null" json:"confidence"`
	Reasons    StringList               `gorm:"type:text;not null" json:"reasons"`
	Status     RecommendationStatus     `gorm:"type:varchar(20);not null;default:'pending'" json:"status"`
	CreatedAt  time.Time                `json:"createdAt"`
	UpdatedAt  time.Time                `json:"updatedAt"`
}

type RecommendationFeedbackAction string

const (
	RecommendationFeedbackActionAccept RecommendationFeedbackAction = "accept"
	RecommendationFeedbackActionReject RecommendationFeedbackAction = "reject"
)

type RecommendationFeedback struct {
	ID               uint                         `gorm:"primaryKey" json:"id"`
	RecommendationID uint                         `gorm:"not null;index" json:"recommendationId"`
	UserID           uint                         `gorm:"not null;index" json:"userId"`
	Action           RecommendationFeedbackAction `gorm:"type:varchar(20);not null" json:"action"`
	Context          StringMap                    `gorm:"type:text;not null" json:"context"`
	CreatedAt        time.Time                    `json:"createdAt"`
}
