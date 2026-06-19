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
