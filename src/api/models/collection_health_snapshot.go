package models

import "time"

// CollectionHealthSnapshot stores one daily collection-health state per user.
type CollectionHealthSnapshot struct {
	ID                uint      `gorm:"primaryKey" json:"id"`
	UserID            uint      `gorm:"not null;index:idx_health_snapshot_user_date,unique;index" json:"userId"`
	SnapshotDate      time.Time `gorm:"type:date;not null;index:idx_health_snapshot_user_date,unique;index" json:"snapshotDate"`
	Score             int       `gorm:"not null" json:"score"`
	GradeA            int       `gorm:"not null;default:0" json:"gradeA"`
	GradeB            int       `gorm:"not null;default:0" json:"gradeB"`
	GradeC            int       `gorm:"not null;default:0" json:"gradeC"`
	GradeD            int       `gorm:"not null;default:0" json:"gradeD"`
	GradeF            int       `gorm:"not null;default:0" json:"gradeF"`
	EligibleCoinCount int       `gorm:"not null;default:0" json:"eligibleCoinCount"`
	CreatedAt         time.Time `json:"createdAt"`
	UpdatedAt         time.Time `json:"updatedAt"`
}
