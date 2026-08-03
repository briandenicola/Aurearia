package services

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

var shipmentServiceDBCounter atomic.Uint64

type stubShipmentCarrierClient struct {
	carrier   models.ShipmentCarrier
	snapshots map[string]ShipmentTrackingSnapshot
	err       error
}

func (c *stubShipmentCarrierClient) Carrier() models.ShipmentCarrier { return c.carrier }

func (c *stubShipmentCarrierClient) GetTracking(_ context.Context, trackingNumber string) (ShipmentTrackingSnapshot, error) {
	if c.err != nil {
		return ShipmentTrackingSnapshot{}, c.err
	}
	snapshot, ok := c.snapshots[trackingNumber]
	if !ok {
		return ShipmentTrackingSnapshot{}, errors.New("tracking not found")
	}
	return snapshot, nil
}

type shipmentServiceHarness struct {
	db           *gorm.DB
	coinRepo     *repository.CoinRepository
	shipmentRepo *repository.ShipmentRepository
	service      *ShipmentService
	user         models.User
	coin         models.Coin
}

func setupShipmentServiceHarness(t *testing.T, clients ...ShipmentCarrierClient) shipmentServiceHarness {
	t.Helper()
	dsn := fmt.Sprintf("file:shipment_service_%d_%d?mode=memory&cache=shared", time.Now().UnixNano(), shipmentServiceDBCounter.Add(1))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("db handle: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)
	if err := db.AutoMigrate(
		&models.User{},
		&models.StorageLocation{},
		&models.MintLocation{},
		&models.Tag{},
		&models.CoinSet{},
		&models.Coin{},
		&models.CoinImage{},
		&models.CoinReference{},
		&models.Notification{},
		&models.Shipment{},
		&models.ShipmentEvent{},
	); err != nil {
		t.Fatalf("migrate db: %v", err)
	}

	user := models.User{Username: "shipper", PasswordHash: "hash"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	coin := models.Coin{Name: "Antoninianus", UserID: user.ID}
	if err := db.Create(&coin).Error; err != nil {
		t.Fatalf("create coin: %v", err)
	}

	coinRepo := repository.NewCoinRepository(db)
	shipmentRepo := repository.NewShipmentRepository(db)
	notifSvc := NewNotificationService(repository.NewNotificationRepository(db), nil, repository.NewUserRepository(db), nil, NewLogger(10))
	service := NewShipmentService(shipmentRepo, coinRepo, NewShipmentCarrierClientRegistry(clients...), notifSvc, NewLogger(10))

	return shipmentServiceHarness{
		db:           db,
		coinRepo:     coinRepo,
		shipmentRepo: shipmentRepo,
		service:      service,
		user:         user,
		coin:         coin,
	}
}

func TestShipmentService_UpsertShipmentForCoin_CreateThenUpdate(t *testing.T) {
	h := setupShipmentServiceHarness(t)

	created, err := h.service.UpsertShipmentForCoin(h.user.ID, h.coin.ID, models.ShipmentCarrierUSPS, "94001000", "first note", "")
	if err != nil {
		t.Fatalf("create shipment: %v", err)
	}
	if created.TrackingNumber != "94001000" {
		t.Fatalf("tracking = %q, want 94001000", created.TrackingNumber)
	}
	if !created.ManualOverrideEnabled {
		t.Fatalf("expected manual override enabled for manual-first tracking")
	}
	if created.CurrentStatus != models.ShipmentStatusPending {
		t.Fatalf("current status = %s, want %s", created.CurrentStatus, models.ShipmentStatusPending)
	}
	if created.CurrentStatusSource != models.ShipmentStatusSourceManual {
		t.Fatalf("status source = %s, want %s", created.CurrentStatusSource, models.ShipmentStatusSourceManual)
	}

	updated, err := h.service.UpsertShipmentForCoin(h.user.ID, h.coin.ID, models.ShipmentCarrierUSPS, "94001001", "updated note", "")
	if err != nil {
		t.Fatalf("update shipment: %v", err)
	}
	if updated.ID != created.ID {
		t.Fatalf("expected same shipment id, got create=%d update=%d", created.ID, updated.ID)
	}
	if updated.TrackingNumber != "94001001" || updated.Notes != "updated note" {
		t.Fatalf("unexpected updated shipment: %+v", updated)
	}
	if !updated.ManualOverrideEnabled {
		t.Fatalf("expected manual override enabled after update")
	}
}

func TestShipmentService_SyncShipment_UpdatesStatusAndMergesTimeline(t *testing.T) {
	now := time.Now().UTC()
	stub := &stubShipmentCarrierClient{
		carrier: models.ShipmentCarrierUSPS,
		snapshots: map[string]ShipmentTrackingSnapshot{
			"94001000": {
				Carrier:             models.ShipmentCarrierUSPS,
				TrackingNumber:      "94001000",
				CurrentStatus:       models.ShipmentStatusOutForDelivery,
				CurrentStatusSource: models.ShipmentStatusSourceAPI,
				Events: []ShipmentTrackingEvent{
					{
						EventKey:     "evt-1",
						Status:       models.ShipmentStatusOutForDelivery,
						StatusSource: models.ShipmentStatusSourceAPI,
						OccurredAt:   now,
						Location:     "Boston",
						Description:  "Out for delivery",
						RawStatus:    "OUT FOR DELIVERY",
						RawPayload:   `{"id":"evt-1"}`,
					},
				},
			},
		},
	}
	h := setupShipmentServiceHarness(t, stub)

	shipment, err := h.service.UpsertShipmentForCoin(h.user.ID, h.coin.ID, models.ShipmentCarrierUSPS, "94001000", "", "")
	if err != nil {
		t.Fatalf("create shipment: %v", err)
	}
	if _, err := h.service.SetManualOverride(h.user.ID, shipment.ID, false, models.ShipmentStatusPending, ""); err != nil {
		t.Fatalf("disable manual override: %v", err)
	}

	synced, err := h.service.SyncShipment(context.Background(), shipment.ID, h.user.ID)
	if err != nil {
		t.Fatalf("sync shipment: %v", err)
	}
	if synced.CurrentStatus != models.ShipmentStatusOutForDelivery {
		t.Fatalf("current status = %s, want %s", synced.CurrentStatus, models.ShipmentStatusOutForDelivery)
	}
	if len(synced.Events) != 1 {
		t.Fatalf("events len = %d, want 1", len(synced.Events))
	}

	// Re-sync same snapshot: upsert by event key should avoid duplicates.
	synced, err = h.service.SyncShipment(context.Background(), shipment.ID, h.user.ID)
	if err != nil {
		t.Fatalf("sync shipment second pass: %v", err)
	}
	if len(synced.Events) != 1 {
		t.Fatalf("events len after second sync = %d, want 1", len(synced.Events))
	}
}

func TestShipmentService_SetManualOverride_DisablesSyncUpdates(t *testing.T) {
	now := time.Now().UTC()
	stub := &stubShipmentCarrierClient{
		carrier: models.ShipmentCarrierUSPS,
		snapshots: map[string]ShipmentTrackingSnapshot{
			"94001000": {
				Carrier:             models.ShipmentCarrierUSPS,
				TrackingNumber:      "94001000",
				CurrentStatus:       models.ShipmentStatusDelivered,
				CurrentStatusSource: models.ShipmentStatusSourceAPI,
				Events: []ShipmentTrackingEvent{
					{
						EventKey:     "evt-1",
						Status:       models.ShipmentStatusDelivered,
						StatusSource: models.ShipmentStatusSourceAPI,
						OccurredAt:   now,
					},
				},
			},
		},
	}
	h := setupShipmentServiceHarness(t, stub)
	shipment, err := h.service.UpsertShipmentForCoin(h.user.ID, h.coin.ID, models.ShipmentCarrierUSPS, "94001000", "", "")
	if err != nil {
		t.Fatalf("create shipment: %v", err)
	}

	if _, err := h.service.SetManualOverride(h.user.ID, shipment.ID, true, models.ShipmentStatusException, "manual hold"); err != nil {
		t.Fatalf("set manual override: %v", err)
	}

	synced, err := h.service.SyncShipment(context.Background(), shipment.ID, h.user.ID)
	if err != nil {
		t.Fatalf("sync shipment: %v", err)
	}
	if synced.CurrentStatus != models.ShipmentStatusException {
		t.Fatalf("status after sync with manual override = %s, want %s", synced.CurrentStatus, models.ShipmentStatusException)
	}
}
