package repository

import (
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"gorm.io/gorm"
)

// PurchaseReminderRepository handles all DB access for PurchaseReminder records.
type PurchaseReminderRepository struct {
	db *gorm.DB
}

// NewPurchaseReminderRepository creates a new repository backed by db.
func NewPurchaseReminderRepository(db *gorm.DB) *PurchaseReminderRepository {
	return &PurchaseReminderRepository{db: db}
}

// WithTx returns a copy of this repository that operates within tx.
func (r *PurchaseReminderRepository) WithTx(tx *gorm.DB) *PurchaseReminderRepository {
	return &PurchaseReminderRepository{db: tx}
}

// DB returns the underlying *gorm.DB (for callers that orchestrate transactions).
func (r *PurchaseReminderRepository) DB() *gorm.DB {
	return r.db
}

// CreateReminder inserts a new reminder row and sets its ID.
func (r *PurchaseReminderRepository) CreateReminder(reminder *models.PurchaseReminder) error {
	return r.db.Create(reminder).Error
}

// FindActiveByCoinAndUser returns the active (pending or notified) reminder for
// a coin/user pair, or nil if none exists.
func (r *PurchaseReminderRepository) FindActiveByCoinAndUser(coinID, userID uint) (*models.PurchaseReminder, error) {
	var reminder models.PurchaseReminder
	err := r.db.
		Where("coin_id = ? AND user_id = ? AND status IN ?", coinID, userID, []string{"pending", "notified"}).
		Limit(1).
		First(&reminder).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &reminder, err
}

// UpdateReminder saves the reminder's mutable fields (remind_date, timezone, status,
// notified_at, cancelled_at) using GORM's Save so zero-values are written.
func (r *PurchaseReminderRepository) UpdateReminder(reminder *models.PurchaseReminder) error {
	return r.db.Model(reminder).
		Select("remind_date", "timezone", "status", "notified_at", "cancelled_at").
		Updates(reminder).Error
}

// CancelActiveForCoin transitions all active reminders for a coin+user to cancelled.
// Uses the receiver's db (which may be a tx-scoped *gorm.DB via WithTx).
func (r *PurchaseReminderRepository) CancelActiveForCoin(coinID, userID uint) error {
	now := time.Now()
	return r.db.Model(&models.PurchaseReminder{}).
		Where("coin_id = ? AND user_id = ? AND status IN ?", coinID, userID, []string{"pending", "notified"}).
		Updates(map[string]interface{}{
			"status":       "cancelled",
			"cancelled_at": now,
		}).Error
}

// CancelActiveForCoinAllUsers transitions all active reminders for a coin (any user)
// to cancelled. Used when a coin is hard-deleted.
func (r *PurchaseReminderRepository) CancelActiveForCoinAllUsers(coinID uint) error {
	now := time.Now()
	return r.db.Model(&models.PurchaseReminder{}).
		Where("coin_id = ? AND status IN ?", coinID, []string{"pending", "notified"}).
		Updates(map[string]interface{}{
			"status":       "cancelled",
			"cancelled_at": now,
		}).Error
}

// ListDueReminders returns all pending reminders. The scheduler evaluates
// whether each is due by comparing RemindDate against today in the stored timezone.
func (r *PurchaseReminderRepository) ListDueReminders() ([]models.PurchaseReminder, error) {
	var reminders []models.PurchaseReminder
	err := r.db.
		Preload("Coin").
		Where("status = ?", "pending").
		Find(&reminders).Error
	return reminders, err
}

// ListActiveByUser returns all active (pending + notified) reminders for a user,
// ordered by remind_date ascending, with Coin preloaded for name access.
func (r *PurchaseReminderRepository) ListActiveByUser(userID uint) ([]models.PurchaseReminder, error) {
	var reminders []models.PurchaseReminder
	err := r.db.
		Preload("Coin").
		Where("user_id = ? AND status IN ?", userID, []string{"pending", "notified"}).
		Order("remind_date ASC").
		Find(&reminders).Error
	return reminders, err
}

// MarkNotified atomically transitions a reminder from pending to notified.
// Returns (true, nil) if the transition occurred, (false, nil) if the reminder
// was already notified or cancelled (idempotent — no error).
func (r *PurchaseReminderRepository) MarkNotified(reminderID uint) (bool, error) {
	now := time.Now()
	result := r.db.Model(&models.PurchaseReminder{}).
		Where("id = ? AND status = ?", reminderID, "pending").
		Updates(map[string]interface{}{
			"status":      "notified",
			"notified_at": now,
		})
	return result.RowsAffected > 0, result.Error
}
