package repository

import (
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"gorm.io/gorm"
)

// PriceAlertRepository encapsulates price-alert persistence.
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

func (r *PriceAlertRepository) ListPendingWithLots() ([]models.PriceAlert, error) {
	var alerts []models.PriceAlert
	statuses := []string{string(models.AuctionStatusWatching), string(models.AuctionStatusBidding)}
	err := r.db.Joins("JOIN auction_lots ON auction_lots.id = price_alerts.auction_lot_id").
		Where("price_alerts.is_triggered = ? AND auction_lots.current_bid IS NOT NULL AND LOWER(auction_lots.status) IN ?", false, statuses).
		Preload("AuctionLot").
		Order("price_alerts.user_id ASC").
		Find(&alerts).Error
	return alerts, err
}

func (r *PriceAlertRepository) MarkTriggeredIfPending(id uint, triggeredAt time.Time) (bool, error) {
	result := r.db.Model(&models.PriceAlert{}).
		Where("id = ? AND is_triggered = ?", id, false).
		Updates(map[string]interface{}{
			"is_triggered": true,
			"triggered_at": triggeredAt,
		})
	return result.RowsAffected == 1, result.Error
}

func (r *PriceAlertRepository) ResetTriggered(id uint) error {
	return r.db.Model(&models.PriceAlert{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"is_triggered": false,
			"triggered_at": nil,
		}).Error
}

// BidReminderRepository encapsulates bid-reminder persistence.
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

func (r *BidReminderRepository) ListPendingWithLots() ([]models.BidReminder, error) {
	var reminders []models.BidReminder
	statuses := []string{string(models.AuctionStatusWatching), string(models.AuctionStatusBidding)}
	err := r.db.Joins("JOIN auction_lots ON auction_lots.id = bid_reminders.auction_lot_id").
		Where("bid_reminders.is_notified = ? AND auction_lots.auction_end_time IS NOT NULL AND LOWER(auction_lots.status) IN ?", false, statuses).
		Preload("AuctionLot").
		Order("bid_reminders.user_id ASC").
		Find(&reminders).Error
	return reminders, err
}

func (r *BidReminderRepository) MarkNotifiedIfPending(id uint, notifiedAt time.Time) (bool, error) {
	result := r.db.Model(&models.BidReminder{}).
		Where("id = ? AND is_notified = ?", id, false).
		Updates(map[string]interface{}{
			"is_notified": true,
			"notified_at": notifiedAt,
		})
	return result.RowsAffected == 1, result.Error
}

func (r *BidReminderRepository) ResetNotified(id uint) error {
	return r.db.Model(&models.BidReminder{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"is_notified": false,
			"notified_at": nil,
		}).Error
}

// AuctionAlertRunRepository encapsulates auction-alert run-log persistence.
type AuctionAlertRunRepository struct {
	db *gorm.DB
}

func NewAuctionAlertRunRepository(db *gorm.DB) *AuctionAlertRunRepository {
	return &AuctionAlertRunRepository{db: db}
}

func (r *AuctionAlertRunRepository) CreateRun(run *models.AuctionAlertRun) error {
	return r.db.Create(run).Error
}

func (r *AuctionAlertRunRepository) CompleteRun(run *models.AuctionAlertRun) error {
	err := r.db.Model(run).Updates(map[string]interface{}{
		"status":                 run.Status,
		"lots_checked":           run.LotsChecked,
		"price_alerts_triggered": run.PriceAlertsTriggered,
		"bid_reminders_sent":     run.BidRemindersSent,
		"duration_ms":            run.DurationMs,
		"completed_at":           run.CompletedAt,
		"error_message":          run.ErrorMessage,
	}).Error
	if err == nil {
		r.PruneOldRuns(100)
	}
	return err
}

func (r *AuctionAlertRunRepository) ListRuns(page, limit int) ([]models.AuctionAlertRun, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	var total int64
	if err := r.db.Model(&models.AuctionAlertRun{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var runs []models.AuctionAlertRun
	offset := (page - 1) * limit
	err := r.db.Order("started_at DESC").Offset(offset).Limit(limit).Find(&runs).Error
	return runs, total, err
}

func (r *AuctionAlertRunRepository) GetLastScheduledRun() *models.AuctionAlertRun {
	var run models.AuctionAlertRun
	err := r.db.Where("trigger_type = ? AND completed_at IS NOT NULL", "scheduled").
		Order("started_at DESC").Limit(1).First(&run).Error
	if err != nil {
		return nil
	}
	return &run
}

func (r *AuctionAlertRunRepository) PruneOldRuns(keep int) {
	var count int64
	r.db.Model(&models.AuctionAlertRun{}).Count(&count)
	if count <= int64(keep) {
		return
	}

	var cutoffRun models.AuctionAlertRun
	if err := r.db.Order("started_at DESC").Offset(keep).Limit(1).First(&cutoffRun).Error; err != nil {
		return
	}

	r.db.Where("started_at <= ?", cutoffRun.StartedAt).Delete(&models.AuctionAlertRun{})
}
