package models

import (
	"strings"
	"time"
)

type ApiKey struct {
	ID           uint       `gorm:"primaryKey" json:"id"`
	UserID       uint       `gorm:"not null;index" json:"userId"`
	User         User       `gorm:"foreignKey:UserID" json:"-"`
	KeyHash      string     `gorm:"not null;uniqueIndex" json:"-"`
	KeyPrefix    string     `gorm:"not null" json:"keyPrefix"`
	Name         string     `gorm:"not null" json:"name"`
	Capabilities string     `gorm:"not null;default:read" json:"capabilities"`
	CreatedAt    time.Time  `json:"createdAt"`
	LastUsedAt   *time.Time `json:"lastUsedAt"`
	RevokedAt    *time.Time `json:"revokedAt"`
}

// HasRead returns true if the API key can perform read operations.
// Write capability implies read capability.
func (a *ApiKey) HasRead() bool {
	return strings.Contains(a.Capabilities, "read") || strings.Contains(a.Capabilities, "write")
}

// HasWrite returns true if the API key can perform write operations.
func (a *ApiKey) HasWrite() bool {
	return strings.Contains(a.Capabilities, "write")
}
