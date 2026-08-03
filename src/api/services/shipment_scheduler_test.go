package services

import (
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
)

func TestShipmentScheduler_RunNow_SyncsShipments(t *testing.T) {
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
					},
				},
			},
		},
	}
	h := setupShipmentServiceHarness(t, stub)
	if err := h.db.AutoMigrate(&models.AppSetting{}); err != nil {
		t.Fatalf("migrate app_setting: %v", err)
	}
	settingsSvc := NewSettingsService(repository.NewSettingsRepository(h.db))
	if err := settingsSvc.SetSetting(SettingShipmentSyncEnabled, "true"); err != nil {
		t.Fatalf("enable shipment sync: %v", err)
	}

	shipment, err := h.service.UpsertShipmentForCoin(h.user.ID, h.coin.ID, models.ShipmentCarrierUSPS, "94001000", "", "")
	if err != nil {
		t.Fatalf("create shipment: %v", err)
	}

	scheduler := NewShipmentScheduler(h.service, settingsSvc, NewLogger(10))
	if err := scheduler.RunNow(); err != nil {
		t.Fatalf("run shipment scheduler now: %v", err)
	}

	updated, err := h.service.GetShipmentByID(h.user.ID, shipment.ID)
	if err != nil {
		t.Fatalf("reload shipment: %v", err)
	}
	if updated.CurrentStatus != models.ShipmentStatusOutForDelivery {
		t.Fatalf("status = %s, want %s", updated.CurrentStatus, models.ShipmentStatusOutForDelivery)
	}
	if len(updated.Events) != 1 {
		t.Fatalf("events len = %d, want 1", len(updated.Events))
	}
}

func TestShipmentScheduler_RunCycle_SkipsWhenDisabled(t *testing.T) {
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
	if err := h.db.AutoMigrate(&models.AppSetting{}); err != nil {
		t.Fatalf("migrate app_setting: %v", err)
	}
	settingsSvc := NewSettingsService(repository.NewSettingsRepository(h.db))

	shipment, err := h.service.UpsertShipmentForCoin(h.user.ID, h.coin.ID, models.ShipmentCarrierUSPS, "94001000", "", "")
	if err != nil {
		t.Fatalf("create shipment: %v", err)
	}

	scheduler := NewShipmentScheduler(h.service, settingsSvc, NewLogger(10))
	scheduler.runCycle()

	updated, err := h.service.GetShipmentByID(h.user.ID, shipment.ID)
	if err != nil {
		t.Fatalf("reload shipment: %v", err)
	}
	if updated.CurrentStatus != models.ShipmentStatusPending {
		t.Fatalf("status = %s, want %s", updated.CurrentStatus, models.ShipmentStatusPending)
	}
}
