package models

import "time"

type OIDCFlowType string

const (
	OIDCFlowTypeLogin OIDCFlowType = "login"
	OIDCFlowTypeLink  OIDCFlowType = "link"
)

type OIDCAuthState struct {
	ID         uint         `gorm:"primaryKey" json:"id"`
	StateHash  string       `gorm:"type:varchar(128);not null;uniqueIndex" json:"-"`
	ProviderID uint         `gorm:"not null;index" json:"providerId"`
	Provider   OIDCProvider `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
	FlowType   OIDCFlowType `gorm:"type:varchar(16);not null;index" json:"flowType"`
	UserID     *uint        `gorm:"index" json:"userId,omitempty"`
	User       *User        `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
	// PKCEVerifier is stored in plaintext because RFC 7636 requires the original verifier during token exchange.
	PKCEVerifier string     `gorm:"column:pkce_verifier_hash;type:varchar(128);not null" json:"-"`
	NonceHash    string     `gorm:"type:varchar(128);not null" json:"-"`
	RedirectPath string     `gorm:"type:text;not null" json:"redirectPath"`
	RedirectURI  string     `gorm:"type:text" json:"-"`
	ExpiresAt    time.Time  `gorm:"not null;index" json:"expiresAt"`
	ConsumedAt   *time.Time `gorm:"index" json:"consumedAt,omitempty"`
	CreatedAt    time.Time  `json:"createdAt"`
}
