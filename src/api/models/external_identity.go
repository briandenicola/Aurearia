package models

import "time"

type ExternalIdentity struct {
	ID            uint         `gorm:"primaryKey" json:"id"`
	UserID        uint         `gorm:"not null;index;uniqueIndex:idx_external_identity_user_provider_subject" json:"userId"`
	User          User         `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
	ProviderID    uint         `gorm:"not null;index;uniqueIndex:idx_external_identity_subject;uniqueIndex:idx_external_identity_user_provider_subject" json:"providerId"`
	Provider      OIDCProvider `gorm:"constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;" json:"-"`
	Issuer        string       `gorm:"type:text;not null;uniqueIndex:idx_external_identity_subject;uniqueIndex:idx_external_identity_user_provider_subject" json:"issuer"`
	Subject       string       `gorm:"type:text;not null;uniqueIndex:idx_external_identity_subject;uniqueIndex:idx_external_identity_user_provider_subject" json:"-"`
	Email         string       `gorm:"type:varchar(254);index" json:"email,omitempty"`
	EmailVerified bool         `gorm:"default:false;not null" json:"emailVerified"`
	DisplayName   string       `gorm:"type:varchar(150)" json:"displayName,omitempty"`
	LastLoginAt   *time.Time   `gorm:"index" json:"lastLoginAt,omitempty"`
	CreatedAt     time.Time    `json:"createdAt"`
	UpdatedAt     time.Time    `json:"updatedAt"`
}
