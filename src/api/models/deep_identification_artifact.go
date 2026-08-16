package models

import "time"

// DeepArtifactRole distinguishes the deep-identification artifact role
// vocabulary from models.ImageType (data-model.md §5).
type DeepArtifactRole string

const (
	DeepArtifactRoleObverse DeepArtifactRole = "obverse"
	DeepArtifactRoleReverse DeepArtifactRole = "reverse"
	DeepArtifactRoleHint    DeepArtifactRole = "hint"
)

// DeepArtifactOrigin identifies whether an artifact came from a fresh
// upload or was reused from an existing saved coin image.
type DeepArtifactOrigin string

const (
	DeepArtifactOriginUploaded       DeepArtifactOrigin = "uploaded"
	DeepArtifactOriginSavedCoinImage DeepArtifactOrigin = "saved_coin_image"
)

// DeepIdentificationArtifact is an image reference (obverse/reverse/hint)
// attached to a job (data-model.md §5). Hint artifacts are always
// Ephemeral=true, never referenced by models.CoinImage, never served by any
// coin-image endpoint, and never embedded/linked in ReportJSON.
type DeepIdentificationArtifact struct {
	ID                uint               `gorm:"primaryKey" json:"id"`
	JobID             uint               `gorm:"not null;index:uix_deep_artifact_job_role,priority:1" json:"jobId"`
	UserID            uint               `gorm:"not null;index" json:"userId"`
	Role              DeepArtifactRole   `gorm:"type:varchar(12);not null;index:uix_deep_artifact_job_role,priority:2" json:"role"`
	Origin            DeepArtifactOrigin `gorm:"type:varchar(20);not null" json:"origin"`
	SourceCoinImageID *uint              `json:"sourceCoinImageId,omitempty"`
	FilePath          string             `gorm:"type:varchar(512)" json:"-"`
	ContentHash       string             `gorm:"type:char(64)" json:"-"`
	ByteSize          int64              `json:"byteSize,omitempty"`
	MimeType          string             `gorm:"type:varchar(40)" json:"mimeType,omitempty"`
	Ephemeral         bool               `gorm:"not null;default:false" json:"ephemeral"`
	DeletedAt         *time.Time         `json:"-"`
	CreatedAt         time.Time          `json:"createdAt"`
}
