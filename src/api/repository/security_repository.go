package repository

import (
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"gorm.io/gorm"
)

type SecurityEventFilters struct {
	Type     string
	Username string
	ClientIP string
	Limit    int
}

type SecurityRepository struct {
	db *gorm.DB
}

func NewSecurityRepository(db *gorm.DB) *SecurityRepository {
	return &SecurityRepository{db: db}
}

func (r *SecurityRepository) CreateEvent(event *models.SecurityEvent) error {
	return r.db.Create(event).Error
}

func (r *SecurityRepository) ListEvents(filters SecurityEventFilters) ([]models.SecurityEvent, error) {
	limit := filters.Limit
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	q := r.db.Model(&models.SecurityEvent{})
	if filters.Type != "" {
		q = q.Where("type = ?", filters.Type)
	}
	if filters.Username != "" {
		q = q.Where("username = ?", filters.Username)
	}
	if filters.ClientIP != "" {
		q = q.Where("client_ip = ?", filters.ClientIP)
	}
	var events []models.SecurityEvent
	err := q.Order("created_at DESC").Limit(limit).Find(&events).Error
	return events, err
}

func (r *SecurityRepository) CountEvents(eventType models.SecurityEventType, since time.Time) (int64, error) {
	var count int64
	err := r.db.Model(&models.SecurityEvent{}).Where("type = ? AND created_at >= ?", eventType, since).Count(&count).Error
	return count, err
}

func (r *SecurityRepository) CountFailuresForUser(userID uint, since time.Time) (int64, error) {
	var success models.SecurityEvent
	if err := r.db.Where("type IN ? AND user_id = ?", []models.SecurityEventType{
		models.SecurityEventPasswordLoginSuccess,
		models.SecurityEventWebAuthnLoginSuccess,
	}, userID).
		Order("created_at DESC").
		First(&success).Error; err == nil && success.CreatedAt.After(since) {
		since = success.CreatedAt
	} else if err != nil && err != gorm.ErrRecordNotFound {
		return 0, err
	}
	var count int64
	err := r.db.Model(&models.SecurityEvent{}).
		Where("type IN ? AND user_id = ? AND created_at > ?", []models.SecurityEventType{
			models.SecurityEventPasswordLoginFailure,
			models.SecurityEventWebAuthnLoginFailure,
		}, userID, since).
		Count(&count).Error
	return count, err
}

func (r *SecurityRepository) CountFailuresForIP(clientIP string, since time.Time) (int64, error) {
	var count int64
	err := r.db.Model(&models.SecurityEvent{}).
		Where("type IN ? AND client_ip = ? AND created_at >= ?", []models.SecurityEventType{
			models.SecurityEventPasswordLoginFailure,
			models.SecurityEventWebAuthnLoginFailure,
		}, clientIP, since).
		Count(&count).Error
	return count, err
}

func (r *SecurityRepository) FindUserByUsername(username string) (*models.User, error) {
	var user models.User
	err := r.db.Where("username = ?", username).First(&user).Error
	return &user, err
}

func (r *SecurityRepository) LockUser(userID uint, until time.Time) error {
	return r.db.Model(&models.User{}).Where("id = ?", userID).Update("locked_until", until).Error
}

func (r *SecurityRepository) UnlockUser(userID uint) error {
	return r.db.Model(&models.User{}).Where("id = ?", userID).Update("locked_until", nil).Error
}

func (r *SecurityRepository) CountAdmins() (int64, error) {
	var count int64
	err := r.db.Model(&models.User{}).Where("role = ?", models.RoleAdmin).Count(&count).Error
	return count, err
}

func (r *SecurityRepository) CreateIPRule(rule *models.IPRule) error {
	return r.db.Create(rule).Error
}

func (r *SecurityRepository) ListIPRules(includeExpired bool) ([]models.IPRule, error) {
	q := r.db.Model(&models.IPRule{})
	if !includeExpired {
		now := time.Now()
		q = q.Where("expires_at IS NULL OR expires_at > ?", now)
	}
	var rules []models.IPRule
	err := q.Order("created_at DESC").Find(&rules).Error
	return rules, err
}

func (r *SecurityRepository) DeleteIPRule(id uint) (int64, error) {
	result := r.db.Delete(&models.IPRule{}, id)
	return result.RowsAffected, result.Error
}
