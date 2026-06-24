package models

import "time"

type OIDCProviderType string

const (
	OIDCProviderTypeEntra    OIDCProviderType = "entra"
	OIDCProviderTypePocketID OIDCProviderType = "pocket_id"
	OIDCProviderTypeGeneric  OIDCProviderType = "generic"
)

type OIDCProviderTestStatus string

const (
	OIDCProviderTestStatusUnknown OIDCProviderTestStatus = "unknown"
	OIDCProviderTestStatusOK      OIDCProviderTestStatus = "ok"
	OIDCProviderTestStatusFailed  OIDCProviderTestStatus = "failed"
)

type OIDCProvider struct {
	ID                   uint                   `gorm:"primaryKey" json:"id"`
	Name                 string                 `gorm:"type:varchar(100);not null;uniqueIndex" json:"name"`
	DisplayName          string                 `gorm:"type:varchar(150);not null" json:"displayName"`
	ProviderType         OIDCProviderType       `gorm:"type:varchar(32);not null;index" json:"providerType"`
	Enabled              bool                   `gorm:"default:false;index" json:"enabled"`
	IssuerURL            string                 `gorm:"type:text;not null;uniqueIndex:idx_oidc_provider_issuer_client" json:"issuerUrl"`
	ClientID             string                 `gorm:"type:text;not null;uniqueIndex:idx_oidc_provider_issuer_client" json:"clientId"`
	ClientSecret         string                 `gorm:"type:text;not null" json:"-"`
	Scopes               StringList             `gorm:"type:text;not null" json:"scopes"`
	CallbackPath         string                 `gorm:"type:text;not null" json:"callbackPath"`
	RequireVerifiedEmail bool                   `gorm:"default:true;not null" json:"requireVerifiedEmail"`
	LastTestedAt         *time.Time             `gorm:"index" json:"lastTestedAt,omitempty"`
	LastTestStatus       OIDCProviderTestStatus `gorm:"type:varchar(32);default:'unknown';not null" json:"lastTestStatus"`
	LastTestMessage      string                 `gorm:"type:text" json:"lastTestMessage,omitempty"`
	CreatedAt            time.Time              `json:"createdAt"`
	UpdatedAt            time.Time              `json:"updatedAt"`
}

func (p OIDCProvider) ClientSecretConfigured() bool {
	return p.ClientSecret != ""
}
