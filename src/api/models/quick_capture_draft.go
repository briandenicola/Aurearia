package models

import "time"

type QuickCaptureDraftStatus string

const (
	QuickCaptureDraftStatusActive    QuickCaptureDraftStatus = "active"
	QuickCaptureDraftStatusPromoting QuickCaptureDraftStatus = "promoting"
	QuickCaptureDraftStatusPromoted  QuickCaptureDraftStatus = "promoted"
	QuickCaptureDraftStatusDiscarded QuickCaptureDraftStatus = "discarded"
)

type DraftLifecycleEventType string

const (
	DraftLifecycleEventCreated                   DraftLifecycleEventType = "created"
	DraftLifecycleEventUpdated                   DraftLifecycleEventType = "updated"
	DraftLifecycleEventImageAdded                DraftLifecycleEventType = "image_added"
	DraftLifecycleEventImageRemoved              DraftLifecycleEventType = "image_removed"
	DraftLifecycleEventPromotionStarted          DraftLifecycleEventType = "promotion_started"
	DraftLifecycleEventPromoted                  DraftLifecycleEventType = "promoted"
	DraftLifecycleEventPromotionReused           DraftLifecycleEventType = "promotion_reused"
	DraftLifecycleEventDiscarded                 DraftLifecycleEventType = "discarded"
	DraftLifecycleEventPromotionFailedValidation DraftLifecycleEventType = "promotion_failed_validation"
)

type QuickCaptureDraft struct {
	ID                uint                     `gorm:"primaryKey" json:"id"`
	UserID            uint                     `gorm:"not null;index" json:"userId"`
	User              User                     `gorm:"foreignKey:UserID" json:"-"`
	WorkingTitle      string                   `gorm:"size:200" json:"workingTitle" binding:"max=200"`
	DateRange         string                   `gorm:"size:200" json:"dateRange" binding:"max=200"`
	Era               string                   `gorm:"size:64" json:"era" binding:"max=64"`
	AcquisitionSource string                   `gorm:"size:500" json:"acquisitionSource" binding:"max=500"`
	PurchasePrice     *float64                 `json:"purchasePrice"`
	Notes             string                   `gorm:"type:text" json:"notes" binding:"max=5000"`
	Source            string                   `gorm:"size:40" json:"source" binding:"max=40"`
	NGCCertNumber     string                   `gorm:"size:32" json:"ngcCertNumber" binding:"max=32"`
	NGCLookupURL      string                   `gorm:"size:500" json:"ngcLookupUrl" binding:"max=500"`
	NGCGrade          string                   `gorm:"size:100" json:"ngcGrade" binding:"max=100"`
	LabelText         string                   `gorm:"type:text" json:"labelText" binding:"max=5000"`
	AIConfidence      string                   `gorm:"size:20" json:"aiConfidence" binding:"max=20"`
	Status            QuickCaptureDraftStatus  `gorm:"type:varchar(20);not null;default:'active';index" json:"status"`
	PromotedCoinID    *uint                    `json:"promotedCoinId"`
	PromotedAt        *time.Time               `json:"promotedAt"`
	DiscardedAt       *time.Time               `json:"discardedAt"`
	Images            []QuickCaptureDraftImage `gorm:"foreignKey:DraftID" json:"images"`
	CreatedAt         time.Time                `json:"createdAt"`
	UpdatedAt         time.Time                `json:"updatedAt"`
}

type QuickCaptureDraftImage struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	DraftID      uint      `gorm:"not null;index" json:"draftId"`
	UserID       uint      `gorm:"not null;index" json:"-"`
	FilePath     string    `gorm:"not null;index" json:"filePath"`
	ImageType    ImageType `gorm:"type:varchar(20);default:'other'" json:"imageType"`
	IsPrimary    bool      `gorm:"default:false" json:"isPrimary"`
	DisplayOrder int       `gorm:"default:0" json:"displayOrder"`
	CreatedAt    time.Time `json:"createdAt"`
}

type DraftLifecycleEvent struct {
	ID        uint                    `gorm:"primaryKey" json:"id"`
	DraftID   uint                    `gorm:"not null;index" json:"draftId"`
	UserID    uint                    `gorm:"not null;index" json:"userId"`
	EventType DraftLifecycleEventType `gorm:"type:varchar(40);not null" json:"eventType"`
	Message   string                  `gorm:"size:500" json:"message"`
	CoinID    *uint                   `json:"coinId"`
	CreatedAt time.Time               `json:"createdAt"`
}

func IsValidQuickCaptureDraftStatus(status QuickCaptureDraftStatus) bool {
	switch status {
	case QuickCaptureDraftStatusActive, QuickCaptureDraftStatusPromoting, QuickCaptureDraftStatusPromoted, QuickCaptureDraftStatusDiscarded:
		return true
	default:
		return false
	}
}

func IsValidDraftLifecycleEventType(eventType DraftLifecycleEventType) bool {
	switch eventType {
	case DraftLifecycleEventCreated, DraftLifecycleEventUpdated, DraftLifecycleEventImageAdded, DraftLifecycleEventImageRemoved,
		DraftLifecycleEventPromotionStarted, DraftLifecycleEventPromoted, DraftLifecycleEventPromotionReused,
		DraftLifecycleEventDiscarded, DraftLifecycleEventPromotionFailedValidation:
		return true
	default:
		return false
	}
}
