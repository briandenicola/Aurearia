package repository

import (
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"gorm.io/gorm"
)

type ShipmentRepository struct {
	db *gorm.DB
}

func NewShipmentRepository(db *gorm.DB) *ShipmentRepository {
	return &ShipmentRepository{db: db}
}

func (r *ShipmentRepository) DB() *gorm.DB {
	return r.db
}

func (r *ShipmentRepository) WithTx(tx *gorm.DB) *ShipmentRepository {
	return &ShipmentRepository{db: tx}
}

func (r *ShipmentRepository) Create(shipment *models.Shipment) error {
	return r.db.Create(shipment).Error
}

func (r *ShipmentRepository) Update(shipment *models.Shipment) error {
	return r.db.Save(shipment).Error
}

func (r *ShipmentRepository) GetByIDForUser(id, userID uint) (*models.Shipment, error) {
	var shipment models.Shipment
	err := r.db.Scopes(OwnedByID(id, userID)).
		Preload("Events", func(db *gorm.DB) *gorm.DB {
			return db.Order("occurred_at DESC, id DESC")
		}).
		First(&shipment).Error
	if err != nil {
		return nil, err
	}
	return &shipment, nil
}

func (r *ShipmentRepository) GetByCoinIDForUser(coinID, userID uint) (*models.Shipment, error) {
	var shipment models.Shipment
	err := r.db.Where("coin_id = ? AND user_id = ?", coinID, userID).
		Preload("Events", func(db *gorm.DB) *gorm.DB {
			return db.Order("occurred_at DESC, id DESC")
		}).
		First(&shipment).Error
	if err != nil {
		return nil, err
	}
	return &shipment, nil
}

func (r *ShipmentRepository) DeleteForUser(id, userID uint) error {
	result := r.db.Scopes(OwnedByID(id, userID)).Delete(&models.Shipment{})
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return result.Error
}

func (r *ShipmentRepository) UpsertEvent(event *models.ShipmentEvent) (bool, error) {
	created := false
	err := r.db.Transaction(func(tx *gorm.DB) error {
		var existing models.ShipmentEvent
		err := tx.Where("shipment_id = ? AND event_key = ?", event.ShipmentID, event.EventKey).First(&existing).Error
		if err == nil {
			existing.Status = event.Status
			existing.StatusSource = event.StatusSource
			existing.OccurredAt = event.OccurredAt
			existing.Location = event.Location
			existing.Description = event.Description
			existing.RawStatus = event.RawStatus
			existing.RawPayload = event.RawPayload
			return tx.Save(&existing).Error
		}
		if err != nil && err != gorm.ErrRecordNotFound {
			return err
		}
		if err := tx.Create(event).Error; err != nil {
			return err
		}
		created = true
		return nil
	})
	return created, err
}

func (r *ShipmentRepository) ListEventsForShipment(shipmentID, userID uint) ([]models.ShipmentEvent, error) {
	var events []models.ShipmentEvent
	err := r.db.Where("shipment_id = ? AND user_id = ?", shipmentID, userID).
		Order("occurred_at DESC, id DESC").
		Find(&events).Error
	return events, err
}

func (r *ShipmentRepository) ListSyncCandidates(carrier *models.ShipmentCarrier, limit int) ([]models.Shipment, error) {
	if limit < 1 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}
	query := r.db.Model(&models.Shipment{}).
		Where("manual_override_enabled = ?", false).
		Where("tracking_number <> ''")
	if carrier != nil {
		query = query.Where("carrier = ?", *carrier)
	}
	var shipments []models.Shipment
	err := query.
		Order("last_synced_at IS NULL DESC, last_synced_at ASC, id ASC").
		Limit(limit).
		Find(&shipments).Error
	return shipments, err
}

func (r *ShipmentRepository) MarkSyncSuccess(
	shipmentID, userID uint,
	status models.ShipmentStatus,
	source models.ShipmentStatusSource,
	estimatedDeliveryAt *time.Time,
	deliveredAt *time.Time,
) error {
	now := time.Now()
	return r.db.Model(&models.Shipment{}).
		Scopes(OwnedByID(shipmentID, userID)).
		Updates(map[string]interface{}{
			"current_status":        status,
			"current_status_source": source,
			"estimated_delivery_at": estimatedDeliveryAt,
			"delivered_at":          deliveredAt,
			"last_synced_at":        now,
			"last_sync_error":       "",
		}).Error
}

func (r *ShipmentRepository) MarkSyncFailure(shipmentID, userID uint, syncErr string) error {
	now := time.Now()
	return r.db.Model(&models.Shipment{}).
		Scopes(OwnedByID(shipmentID, userID)).
		Updates(map[string]interface{}{
			"last_synced_at":  now,
			"last_sync_error": syncErr,
		}).Error
}
