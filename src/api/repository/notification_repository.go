package repository

import (
	"github.com/briandenicola/ancient-coins-api/models"
	"gorm.io/gorm"
)

type NotificationRepository struct {
	db *gorm.DB
}

func NewNotificationRepository(db *gorm.DB) *NotificationRepository {
	return &NotificationRepository{db: db}
}

// Create inserts a new notification.
func (r *NotificationRepository) Create(n *models.Notification) error {
	return r.db.Create(n).Error
}

// GetByUser returns notifications for a user, newest first, with pagination.
func (r *NotificationRepository) GetByUser(userID uint, limit, offset int) ([]models.Notification, int64, error) {
	var total int64
	r.db.Model(&models.Notification{}).Scopes(OwnedBy(userID)).Count(&total)

	var notifications []models.Notification
	err := r.db.Scopes(OwnedBy(userID)).
		Order("created_at DESC").
		Limit(limit).Offset(offset).
		Find(&notifications).Error
	return notifications, total, err
}

// GetUnreadCount returns the number of unread notifications for a user.
func (r *NotificationRepository) GetUnreadCount(userID uint) (int64, error) {
	var count int64
	err := r.db.Model(&models.Notification{}).
		Scopes(OwnedBy(userID)).
		Where("is_read = ?", false).
		Count(&count).Error
	return count, err
}

// MarkRead marks a single notification as read for the given user.
func (r *NotificationRepository) MarkRead(id, userID uint) error {
	result := r.db.Model(&models.Notification{}).
		Scopes(OwnedByID(id, userID)).
		Update("is_read", true)
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return result.Error
}

// MarkAllRead marks all unread notifications as read for the given user.
func (r *NotificationRepository) MarkAllRead(userID uint) error {
	return r.db.Model(&models.Notification{}).
		Scopes(OwnedBy(userID)).
		Where("is_read = ?", false).
		Update("is_read", true).Error
}

// Delete removes a single notification for the given user.
func (r *NotificationRepository) Delete(id, userID uint) error {
	result := r.db.Scopes(OwnedByID(id, userID)).Delete(&models.Notification{})
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return result.Error
}
