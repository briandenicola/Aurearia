package repository

import (
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"gorm.io/gorm"
)

type PriceAlertRepository struct {
	db *gorm.DB
}

func NewPriceAlertRepository(db *gorm.DB) *PriceAlertRepository {
	return &PriceAlertRepository{db: db}
}

func (r *PriceAlertRepository) Create(alert *models.PriceAlert) error {
	return r.db.Create(alert).Error
}

func (r *PriceAlertRepository) Delete(id uint, userID uint) error {
	return r.db.Where("id = ? AND user_id = ?", id, userID).Delete(&models.PriceAlert{}).Error
}

func (r *PriceAlertRepository) ListByUser(userID uint) ([]models.PriceAlert, error) {
	var alerts []models.PriceAlert
	err := r.db.Where("user_id = ?", userID).
		Preload("AuctionLot").
		Order("created_at DESC").
		Find(&alerts).Error
	return alerts, err
}

func (r *PriceAlertRepository) ListByLot(lotID uint, userID uint) ([]models.PriceAlert, error) {
	var alerts []models.PriceAlert
	err := r.db.Where("auction_lot_id = ? AND user_id = ?", lotID, userID).Find(&alerts).Error
	return alerts, err
}

// CheckAndTrigger checks all un-triggered alerts for a user and triggers any that match.
// Returns the triggered alerts (used by sync flow to send notifications).
func (r *PriceAlertRepository) CheckAndTrigger(userID uint) ([]models.PriceAlert, error) {
	var alerts []models.PriceAlert
	err := r.db.Where("user_id = ? AND is_triggered = ?", userID, false).
		Preload("AuctionLot").
		Find(&alerts).Error
	if err != nil {
		return nil, err
	}

	var triggered []models.PriceAlert
	now := time.Now()
	for _, alert := range alerts {
		bid := alert.AuctionLot.CurrentBid
		if bid == nil {
			continue
		}
		shouldTrigger := false
		if alert.Direction == "above" && *bid >= alert.TargetPrice {
			shouldTrigger = true
		} else if alert.Direction == "below" && *bid <= alert.TargetPrice {
			shouldTrigger = true
		}
		if shouldTrigger {
			alert.IsTriggered = true
			alert.TriggeredAt = &now
			r.db.Save(&alert)
			triggered = append(triggered, alert)
		}
	}
	return triggered, nil
}

// Bid Reminders

type BidReminderRepository struct {
	db *gorm.DB
}

func NewBidReminderRepository(db *gorm.DB) *BidReminderRepository {
	return &BidReminderRepository{db: db}
}

func (r *BidReminderRepository) Create(reminder *models.BidReminder) error {
	return r.db.Create(reminder).Error
}

func (r *BidReminderRepository) Delete(id uint, userID uint) error {
	return r.db.Where("id = ? AND user_id = ?", id, userID).Delete(&models.BidReminder{}).Error
}

func (r *BidReminderRepository) ListByUser(userID uint) ([]models.BidReminder, error) {
	var reminders []models.BidReminder
	err := r.db.Where("user_id = ?", userID).
		Preload("AuctionLot").
		Order("created_at DESC").
		Find(&reminders).Error
	return reminders, err
}

// GetDueReminders returns un-notified reminders whose auction end time is within minutesBefore.
func (r *BidReminderRepository) GetDueReminders(userID uint) ([]models.BidReminder, error) {
	var reminders []models.BidReminder
	err := r.db.Where("user_id = ? AND is_notified = ?", userID, false).
		Preload("AuctionLot").
		Find(&reminders).Error
	if err != nil {
		return nil, err
	}

	var due []models.BidReminder
	now := time.Now()
	for _, rem := range reminders {
		endTime := rem.AuctionLot.AuctionEndTime
		if endTime == nil {
			continue
		}
		threshold := endTime.Add(-time.Duration(rem.MinutesBefore) * time.Minute)
		if now.After(threshold) && now.Before(*endTime) {
			rem.IsNotified = true
			rem.NotifiedAt = &now
			r.db.Save(&rem)
			due = append(due, rem)
		}
	}
	return due, nil
}
