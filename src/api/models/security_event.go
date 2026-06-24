package models

import "time"

type SecurityEventType string

const (
	SecurityEventPasswordLoginSuccess SecurityEventType = "password_login_success"
	SecurityEventPasswordLoginFailure SecurityEventType = "password_login_failure"
	SecurityEventWebAuthnLoginSuccess SecurityEventType = "webauthn_login_success"
	SecurityEventWebAuthnLoginFailure SecurityEventType = "webauthn_login_failure"
	SecurityEventRefreshFailure       SecurityEventType = "refresh_failure"
	SecurityEventAPIKeyAuthFailure    SecurityEventType = "api_key_auth_failure"
	SecurityEventAccountLockout       SecurityEventType = "account_lockout"
	SecurityEventAccountUnlock        SecurityEventType = "account_unlock"
	SecurityEventIPRuleCreated        SecurityEventType = "ip_rule_created"
	SecurityEventIPRuleDeleted        SecurityEventType = "ip_rule_deleted"
	SecurityEventOIDCLoginSuccess     SecurityEventType = "oidc_login_success"
	SecurityEventOIDCLoginFailure     SecurityEventType = "oidc_login_failure"
	SecurityEventOIDCLinkSuccess      SecurityEventType = "oidc_link_success"
	SecurityEventOIDCLinkFailure      SecurityEventType = "oidc_link_failure"
	SecurityEventOIDCUnlinkSuccess    SecurityEventType = "oidc_unlink_success"
	SecurityEventOIDCUnlinkFailure    SecurityEventType = "oidc_unlink_failure"
	SecurityEventOIDCProviderChanged  SecurityEventType = "oidc_provider_config_changed"
	SecurityEventOIDCProviderTestFail SecurityEventType = "oidc_provider_test_failure"
	SecurityEventFinalAdminBlocked    SecurityEventType = "final_local_admin_blocked"
)

type SecurityEvent struct {
	ID        uint              `gorm:"primaryKey" json:"id"`
	Type      SecurityEventType `gorm:"type:varchar(64);not null;index" json:"type"`
	UserID    *uint             `gorm:"index" json:"userId,omitempty"`
	Username  string            `gorm:"type:varchar(100);index" json:"username,omitempty"`
	ClientIP  string            `gorm:"column:client_ip;type:varchar(64);index" json:"clientIp,omitempty"`
	UserAgent string            `gorm:"type:text" json:"userAgent,omitempty"`
	Message   string            `gorm:"type:text" json:"message,omitempty"`
	CreatedAt time.Time         `gorm:"index" json:"createdAt"`
}
