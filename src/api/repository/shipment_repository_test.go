package repository

import (
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupShipmentRepository(t *testing.T) *ShipmentRepository {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&models.Shipment{}, &models.ShipmentEvent{}); err != nil {
		t.Fatalf("migrate db: %v", err)
	}
	return NewShipmentRepository(db)
}

func TestShipmentRepository_GetByIDForUser_OwnerScoped(t *testing.T) {
	repo := setupShipmentRepository(t)
	shipment := &models.Shipment{
		UserID:         10,
		CoinID:         100,
		Carrier:        models.ShipmentCarrierUSPS,
		TrackingNumber: "9400-1000",
		CurrentStatus:  models.ShipmentStatusPending,
	}
	if err := repo.Create(shipment); err != nil {
		t.Fatalf("create shipment: %v", err)
	}

	if _, err := repo.GetByIDForUser(shipment.ID, 11); !IsRecordNotFound(err) {
		t.Fatalf("non-owner get error = %v", err)
	}

	got, err := repo.GetByIDForUser(shipment.ID, 10)
	if err != nil {
		t.Fatalf("owner get: %v", err)
	}
	if got.TrackingNumber != "9400-1000" {
		t.Fatalf("tracking number = %q, want 9400-1000", got.TrackingNumber)
	}
}

func TestShipmentRepository_UpsertEvent_DeduplicatesByEventKey(t *testing.T) {
	repo := setupShipmentRepository(t)
	shipment := &models.Shipment{
		UserID:         12,
		CoinID:         102,
		Carrier:        models.ShipmentCarrierUPS,
		TrackingNumber: "1Z-123",
		CurrentStatus:  models.ShipmentStatusInTransit,
	}
	if err := repo.Create(shipment); err != nil {
		t.Fatalf("create shipment: %v", err)
	}

	now := time.Now()
	first := &models.ShipmentEvent{
		ShipmentID:   shipment.ID,
		UserID:       shipment.UserID,
		EventKey:     "ups:scan-001",
		Status:       models.ShipmentStatusInTransit,
		StatusSource: models.ShipmentStatusSourceAPI,
		OccurredAt:   now.Add(-1 * time.Hour),
		Location:     "Hub A",
		Description:  "Arrived at facility",
		RawStatus:    "IN_TRANSIT",
	}
	created, err := repo.UpsertEvent(first)
	if err != nil {
		t.Fatalf("upsert first event: %v", err)
	}
	if !created {
		t.Fatalf("first upsert should create event")
	}

	second := &models.ShipmentEvent{
		ShipmentID:   shipment.ID,
		UserID:       shipment.UserID,
		EventKey:     "ups:scan-001",
		Status:       models.ShipmentStatusOutForDelivery,
		StatusSource: models.ShipmentStatusSourceAPI,
		OccurredAt:   now,
		Location:     "Local depot",
		Description:  "Out for delivery",
		RawStatus:    "OUT_FOR_DELIVERY",
	}
	created, err = repo.UpsertEvent(second)
	if err != nil {
		t.Fatalf("upsert second event: %v", err)
	}
	if created {
		t.Fatalf("second upsert should update existing event, not create")
	}

	events, err := repo.ListEventsForShipment(shipment.ID, shipment.UserID)
	if err != nil {
		t.Fatalf("list events: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("events len = %d, want 1", len(events))
	}
	if events[0].Status != models.ShipmentStatusOutForDelivery {
		t.Fatalf("event status = %s, want %s", events[0].Status, models.ShipmentStatusOutForDelivery)
	}
}

func TestShipmentRepository_ListSyncCandidates_FiltersManualOverride(t *testing.T) {
	repo := setupShipmentRepository(t)
	oldSync := time.Now().Add(-12 * time.Hour)

	shipments := []*models.Shipment{
		{
			UserID:         1,
			CoinID:         200,
			Carrier:        models.ShipmentCarrierUSPS,
			TrackingNumber: "9400-2000",
			CurrentStatus:  models.ShipmentStatusInTransit,
		},
		{
			UserID:                1,
			CoinID:                201,
			Carrier:               models.ShipmentCarrierUSPS,
			TrackingNumber:        "9400-2001",
			CurrentStatus:         models.ShipmentStatusInTransit,
			ManualOverrideEnabled: true,
		},
		{
			UserID:         2,
			CoinID:         202,
			Carrier:        models.ShipmentCarrierUSPS,
			TrackingNumber: "9400-2002",
			CurrentStatus:  models.ShipmentStatusInTransit,
			LastSyncedAt:   &oldSync,
		},
	}
	for _, shipment := range shipments {
		if err := repo.Create(shipment); err != nil {
			t.Fatalf("create shipment: %v", err)
		}
	}

	carrier := models.ShipmentCarrierUSPS
	candidates, err := repo.ListSyncCandidates(&carrier, 10)
	if err != nil {
		t.Fatalf("list sync candidates: %v", err)
	}
	if len(candidates) != 2 {
		t.Fatalf("candidate len = %d, want 2", len(candidates))
	}
	if candidates[0].CoinID != 200 {
		t.Fatalf("first candidate coin_id = %d, want 200 (never synced first)", candidates[0].CoinID)
	}
	if candidates[1].CoinID != 202 {
		t.Fatalf("second candidate coin_id = %d, want 202", candidates[1].CoinID)
	}
}
