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
	return HasAPICapability(a.Capabilities, "read")
}

// HasWrite returns true if the API key can perform write operations.
func (a *ApiKey) HasWrite() bool {
	return HasAPICapability(a.Capabilities, "write")
}

// HasAPICapability checks comma-separated API-key capabilities using exact tokens.
// Write capability implies read capability.
func HasAPICapability(capabilities string, required string) bool {
	hasRead := false
	hasWrite := false
	for _, capability := range strings.Split(capabilities, ",") {
		switch strings.TrimSpace(capability) {
		case "read":
			hasRead = true
		case "write":
			hasWrite = true
		}
	}

	switch required {
	case "read":
		return hasRead || hasWrite
	case "write":
		return hasWrite
	default:
		return false
	}
}
