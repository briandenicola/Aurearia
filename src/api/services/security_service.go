package services

import (
	"errors"
	"net"
	"strings"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"gorm.io/gorm"
)

const (
	AccountFailureLimit = 5
	IPFailureLimit      = 20
	FailureWindow       = 15 * time.Minute
	LockoutDuration     = 15 * time.Minute
	TempBanDuration     = 15 * time.Minute
)

var (
	ErrAccountLocked = errors.New("account locked")
	ErrIPDenied      = errors.New("ip denied")
	ErrInvalidIPRule = errors.New("invalid ip rule")
)

type SecuritySummary struct {
	Since             time.Time `json:"since"`
	LoginFailures     int64     `json:"loginFailures"`
	LoginSuccesses    int64     `json:"loginSuccesses"`
	RefreshFailures   int64     `json:"refreshFailures"`
	APIKeyFailures    int64     `json:"apiKeyFailures"`
	ActiveIPRuleCount int       `json:"activeIpRuleCount"`
}

type SecurityService struct {
	repo *repository.SecurityRepository
	now  func() time.Time
}

func NewSecurityService(repo *repository.SecurityRepository) *SecurityService {
	return &SecurityService{repo: repo, now: time.Now}
}

func (s *SecurityService) RecordEvent(eventType models.SecurityEventType, userID *uint, username, clientIP, userAgent, message string) {
	_ = s.repo.CreateEvent(&models.SecurityEvent{
		Type:      eventType,
		UserID:    userID,
		Username:  username,
		ClientIP:  clientIP,
		UserAgent: userAgent,
		Message:   message,
		CreatedAt: s.now(),
	})
}

func (s *SecurityService) CheckAccountAllowed(username string) error {
	user, err := s.repo.FindUserByUsername(username)
	if err != nil {
		return nil
	}
	if user.LockedUntil != nil && user.LockedUntil.After(s.now()) {
		if s.isOnlyAdmin(user) {
			_ = s.repo.UnlockUser(user.ID)
			return nil
		}
		return ErrAccountLocked
	}
	return nil
}

func (s *SecurityService) RecordPasswordSuccess(user models.User, clientIP, userAgent string) {
	userID := user.ID
	s.RecordEvent(models.SecurityEventPasswordLoginSuccess, &userID, user.Username, clientIP, userAgent, "")
	if user.LockedUntil != nil {
		_ = s.repo.UnlockUser(user.ID)
	}
}

func (s *SecurityService) RecordWebAuthnSuccess(user models.User, clientIP, userAgent string) {
	userID := user.ID
	s.RecordEvent(models.SecurityEventWebAuthnLoginSuccess, &userID, user.Username, clientIP, userAgent, "")
	if user.LockedUntil != nil {
		_ = s.repo.UnlockUser(user.ID)
	}
}

func (s *SecurityService) RecordPasswordFailure(username, clientIP, userAgent string) {
	var userID *uint
	user, err := s.repo.FindUserByUsername(username)
	if err == nil {
		id := user.ID
		userID = &id
	}
	s.RecordEvent(models.SecurityEventPasswordLoginFailure, userID, username, clientIP, userAgent, "")

	if userID != nil {
		count, _ := s.repo.CountFailuresForUser(*userID, s.now().Add(-FailureWindow))
		if count >= AccountFailureLimit {
			if s.canLockUser(user) {
				until := s.now().Add(LockoutDuration)
				_ = s.repo.LockUser(*userID, until)
				s.RecordEvent(models.SecurityEventAccountLockout, userID, username, clientIP, userAgent, "too many failed password attempts")
			}
		}
	}

	if clientIP != "" {
		count, _ := s.repo.CountFailuresForIP(clientIP, s.now().Add(-FailureWindow))
		if count >= IPFailureLimit {
			_ = s.CreateIPRule(clientIP, "temporary ban after failed password attempts", s.now().Add(TempBanDuration), nil)
		}
	}
}

func (s *SecurityService) canLockUser(user *models.User) bool {
	return user != nil && !s.isOnlyAdmin(user)
}

func (s *SecurityService) isOnlyAdmin(user *models.User) bool {
	if user == nil || user.Role != models.RoleAdmin {
		return false
	}
	admins, err := s.repo.CountAdmins()
	return err == nil && admins <= 1
}

func (s *SecurityService) ListEvents(filters repository.SecurityEventFilters) ([]models.SecurityEvent, error) {
	return s.repo.ListEvents(filters)
}

func (s *SecurityService) Summary() (SecuritySummary, error) {
	since := s.now().Add(-24 * time.Hour)
	failures, err := s.repo.CountEvents(models.SecurityEventPasswordLoginFailure, since)
	if err != nil {
		return SecuritySummary{}, err
	}
	successes, err := s.repo.CountEvents(models.SecurityEventPasswordLoginSuccess, since)
	if err != nil {
		return SecuritySummary{}, err
	}
	refreshFailures, err := s.repo.CountEvents(models.SecurityEventRefreshFailure, since)
	if err != nil {
		return SecuritySummary{}, err
	}
	apiKeyFailures, err := s.repo.CountEvents(models.SecurityEventAPIKeyAuthFailure, since)
	if err != nil {
		return SecuritySummary{}, err
	}
	rules, err := s.repo.ListIPRules(false)
	if err != nil {
		return SecuritySummary{}, err
	}
	return SecuritySummary{Since: since, LoginFailures: failures, LoginSuccesses: successes, RefreshFailures: refreshFailures, APIKeyFailures: apiKeyFailures, ActiveIPRuleCount: len(rules)}, nil
}

func (s *SecurityService) CreateIPRule(cidrOrIP, reason string, expiresAt time.Time, createdBy *uint) error {
	cidr := strings.TrimSpace(cidrOrIP)
	if cidr == "" {
		return ErrInvalidIPRule
	}
	if !strings.Contains(cidr, "/") {
		ip := net.ParseIP(cidr)
		if ip == nil {
			return ErrInvalidIPRule
		}
		if ip.To4() != nil {
			cidr += "/32"
		} else {
			cidr += "/128"
		}
	}
	if _, _, err := net.ParseCIDR(cidr); err != nil {
		return ErrInvalidIPRule
	}
	var expires *time.Time
	if !expiresAt.IsZero() {
		expires = &expiresAt
	}
	if reason == "" {
		reason = "admin-created deny rule"
	}
	if err := s.repo.CreateIPRule(&models.IPRule{CIDR: cidr, Reason: reason, ExpiresAt: expires, CreatedBy: createdBy}); err != nil {
		return err
	}
	s.RecordEvent(models.SecurityEventIPRuleCreated, createdBy, "", "", "", cidr)
	return nil
}

func (s *SecurityService) ListIPRules(includeExpired bool) ([]models.IPRule, error) {
	return s.repo.ListIPRules(includeExpired)
}

func (s *SecurityService) DeleteIPRule(id uint, adminID uint) error {
	rows, err := s.repo.DeleteIPRule(id)
	if err != nil {
		return err
	}
	if rows == 0 {
		return gorm.ErrRecordNotFound
	}
	s.RecordEvent(models.SecurityEventIPRuleDeleted, &adminID, "", "", "", "")
	return nil
}

func (s *SecurityService) IsIPDenied(clientIP string) bool {
	ip := net.ParseIP(clientIP)
	if ip == nil {
		return false
	}
	rules, err := s.repo.ListIPRules(true)
	if err != nil {
		return false
	}
	for _, rule := range rules {
		if rule.ExpiresAt != nil && !rule.ExpiresAt.After(s.now()) {
			continue
		}
		_, network, err := net.ParseCIDR(rule.CIDR)
		if err == nil && network.Contains(ip) {
			return true
		}
	}
	return false
}

func (s *SecurityService) UnlockUser(userID uint, adminID uint) error {
	if err := s.repo.UnlockUser(userID); err != nil {
		return err
	}
	s.RecordEvent(models.SecurityEventAccountUnlock, &adminID, "", "", "", "")
	return nil
}
