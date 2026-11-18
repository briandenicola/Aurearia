package models

import "time"

// ImageVariantRun records one pass of the image variant check, scheduled or
// started from the admin page.
type ImageVariantRun struct {
	ID          uint   `gorm:"primaryKey" json:"id"`
	TriggerType string `gorm:"type:varchar(20);not null" json:"triggerType"`
	// Status is running, success, stopped (the API shut down part way) or
	// error (the photos could not be listed).
	Status      string     `gorm:"type:varchar(20);not null;default:'running'" json:"status"`
	Checked     int        `json:"checked"`
	Made        int        `json:"made"`
	Failed      int        `json:"failed"`
	DurationMs  int64      `json:"durationMs"`
	StartedAt   time.Time  `gorm:"not null;index" json:"startedAt"`
	CompletedAt *time.Time `json:"completedAt"`
	// Failures names up to ten photos that could not be resized and why, one per line.
	Failures     string    `gorm:"type:text" json:"failures,omitempty"`
	ErrorMessage string    `gorm:"type:text" json:"errorMessage,omitempty"`
	CreatedAt    time.Time `json:"createdAt"`
}
