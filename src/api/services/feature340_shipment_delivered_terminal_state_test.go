package services

// Independent QA regression coverage for the shipment "delivered is a
// terminal state" change, owned by Brutus (Tester/QA).
//
// Deliberately a NEW dedicated file, separate from shipment_service_test.go
// (Cassius's existing test file, which currently has no coverage of the
// delivered-shipment guard in syncSingleShipment) — this avoids merge
// collisions while closing a real gap: proving that once a shipment is
// delivered — whether that status was reached via a manual override or via
// a provider/carrier sync write — neither the manual/direct sync entrypoint
// (SyncShipment) nor the automatic entrypoint (SyncCandidates) ever invokes
// the carrier client or the ParcelApp client for it again.

import (
	"context"
	"testing"

	"github.com/briandenicola/ancient-coins-api/models"
)

// TestFeature340_SyncShipment_ManuallyDeliveredShipment_NeverCallsCarrier
// proves the direct/manual single-shipment sync entrypoint (SyncShipment)
// short-circuits before any carrier API call when the shipment's delivered
// status was set via a manual override.
func TestFeature340_SyncShipment_ManuallyDeliveredShipment_NeverCallsCarrier(t *testing.T) {
	carrierClient := &stubShipmentCarrierClient{carrier: models.ShipmentCarrierUSPS}
	h := setupShipmentServiceHarness(t, carrierClient)

	shipment, err := h.service.UpsertShipmentForCoin(h.user.ID, h.coin.ID, models.ShipmentCarrierUSPS, "9400-6001", "", "")
	if err != nil {
		t.Fatalf("create shipment: %v", err)
	}

	if _, err := h.service.SetManualOverride(h.user.ID, shipment.ID, true, models.ShipmentStatusDelivered, "picked up in person"); err != nil {
		t.Fatalf("set manual override delivered: %v", err)
	}

	if _, err := h.service.SyncShipment(context.Background(), shipment.ID, h.user.ID); err != nil {
		t.Fatalf("sync shipment: %v", err)
	}
	if carrierClient.calls != 0 {
		t.Fatalf("carrier calls = %d, want 0 for a manually-delivered shipment", carrierClient.calls)
	}
}

// TestFeature340_SyncShipment_ProviderDeliveredShipment_NeverCallsCarrierAgain
// proves the same guard holds when the delivered status was written by the
// provider/carrier sync path itself (CurrentStatusSource = carrier_api,
// ManualOverrideEnabled = false) rather than by a manual override — the
// first sync call is expected to reach the carrier and observe "delivered",
// but any subsequent sync of that same shipment must not call the carrier
// again.
func TestFeature340_SyncShipment_ProviderDeliveredShipment_NeverCallsCarrierAgain(t *testing.T) {
	carrierClient := &stubShipmentCarrierClient{
		carrier: models.ShipmentCarrierUSPS,
		snapshots: map[string]ShipmentTrackingSnapshot{
			"9400-6002": {
				CurrentStatus:       models.ShipmentStatusDelivered,
				CurrentStatusSource: models.ShipmentStatusSourceAPI,
				RawCurrentStatus:    "DELIVERED",
			},
		},
	}
	h := setupShipmentServiceHarness(t, carrierClient)

	shipment, err := h.service.UpsertShipmentForCoin(h.user.ID, h.coin.ID, models.ShipmentCarrierUSPS, "9400-6002", "", "")
	if err != nil {
		t.Fatalf("create shipment: %v", err)
	}
	// UpsertShipmentForCoin enables ManualOverrideEnabled for a manual-first
	// tracking number; disable it here so the shipment behaves like one
	// whose lifecycle is driven purely by provider sync (mirrors how a
	// non-manual/auto-discovered shipment would look).
	if _, err := h.service.SetManualOverride(h.user.ID, shipment.ID, false, "", ""); err != nil {
		t.Fatalf("disable manual override: %v", err)
	}

	updated, err := h.service.SyncShipment(context.Background(), shipment.ID, h.user.ID)
	if err != nil {
		t.Fatalf("first sync: %v", err)
	}
	if carrierClient.calls != 1 {
		t.Fatalf("carrier calls after first sync = %d, want 1", carrierClient.calls)
	}
	if updated.CurrentStatus != models.ShipmentStatusDelivered {
		t.Fatalf("current status = %s, want delivered after first sync", updated.CurrentStatus)
	}
	if updated.CurrentStatusSource != models.ShipmentStatusSourceAPI {
		t.Fatalf("current status source = %s, want carrier_api", updated.CurrentStatusSource)
	}

	if _, err := h.service.SyncShipment(context.Background(), shipment.ID, h.user.ID); err != nil {
		t.Fatalf("second sync: %v", err)
	}
	if carrierClient.calls != 1 {
		t.Fatalf("carrier calls after second sync = %d, want still 1 (no re-sync of a delivered shipment)", carrierClient.calls)
	}
}

// TestFeature340_SyncCandidates_SkipsDeliveredShipment proves the automatic
// scheduled-sync entrypoint never reaches the carrier for a delivered
// shipment, while a still-in-transit shipment for the same user/carrier is
// still synced normally.
func TestFeature340_SyncCandidates_SkipsDeliveredShipment(t *testing.T) {
	carrierClient := &stubShipmentCarrierClient{
		carrier: models.ShipmentCarrierUSPS,
		snapshots: map[string]ShipmentTrackingSnapshot{
			"9400-6003": {CurrentStatus: models.ShipmentStatusOutForDelivery, RawCurrentStatus: "OUT_FOR_DELIVERY"},
		},
	}
	h := setupShipmentServiceHarness(t, carrierClient)

	deliveredShipment := &models.Shipment{
		UserID:                h.user.ID,
		CoinID:                h.coin.ID,
		Carrier:               models.ShipmentCarrierUSPS,
		TrackingNumber:        "9400-6099",
		CurrentStatus:         models.ShipmentStatusDelivered,
		CurrentStatusSource:   models.ShipmentStatusSourceAPI,
		ManualOverrideEnabled: false,
	}
	if err := h.shipmentRepo.Create(deliveredShipment); err != nil {
		t.Fatalf("create delivered shipment: %v", err)
	}

	inTransitShipment := &models.Shipment{
		UserID:                h.user.ID,
		CoinID:                h.coin.ID + 1,
		Carrier:               models.ShipmentCarrierUSPS,
		TrackingNumber:        "9400-6003",
		CurrentStatus:         models.ShipmentStatusInTransit,
		ManualOverrideEnabled: false,
	}
	if err := h.shipmentRepo.Create(inTransitShipment); err != nil {
		t.Fatalf("create in-transit shipment: %v", err)
	}

	carrier := models.ShipmentCarrierUSPS
	summary, err := h.service.SyncCandidates(context.Background(), &carrier, 10)
	if err != nil {
		t.Fatalf("sync candidates: %v", err)
	}
	if carrierClient.calls != 1 {
		t.Fatalf("carrier calls = %d, want 1 (only the in-transit shipment should be checked)", carrierClient.calls)
	}
	if summary.Checked != 1 {
		t.Fatalf("summary.Checked = %d, want 1 (delivered shipment excluded at the repository level)", summary.Checked)
	}
	if summary.Updated != 1 {
		t.Fatalf("summary.Updated = %d, want 1", summary.Updated)
	}

	refetchedDelivered, err := h.shipmentRepo.GetByIDForUser(deliveredShipment.ID, h.user.ID)
	if err != nil {
		t.Fatalf("refetch delivered shipment: %v", err)
	}
	if refetchedDelivered.CurrentStatus != models.ShipmentStatusDelivered {
		t.Fatalf("delivered shipment status changed to %s, want unchanged delivered", refetchedDelivered.CurrentStatus)
	}
}

// TestFeature340_SyncShipment_DeliveredParcelShipment_NeverCallsParcelApp
// proves the syncSingleShipment delivered guard fires before the
// Parcel-specific branch, so a delivered Parcel-carrier shipment's direct
// sync never calls the ParcelApp client either.
func TestFeature340_SyncShipment_DeliveredParcelShipment_NeverCallsParcelApp(t *testing.T) {
	h := setupShipmentServiceHarness(t)
	parcel := &stubParcelAppClient{}
	h.service.WithParcelAppSupport(nil, nil, nil, parcel)

	shipment := &models.Shipment{
		UserID:                h.user.ID,
		CoinID:                h.coin.ID,
		Carrier:               models.ShipmentCarrierParcel,
		TrackingNumber:        "PX-6004",
		CurrentStatus:         models.ShipmentStatusDelivered,
		CurrentStatusSource:   models.ShipmentStatusSourceAPI,
		ManualOverrideEnabled: false,
	}
	if err := h.shipmentRepo.Create(shipment); err != nil {
		t.Fatalf("create delivered parcel shipment: %v", err)
	}

	if _, err := h.service.SyncShipment(context.Background(), shipment.ID, h.user.ID); err != nil {
		t.Fatalf("sync shipment: %v", err)
	}
	if parcel.listCalls != 0 || parcel.addCalls != 0 {
		t.Fatalf("parcel calls = list:%d add:%d, want 0/0 for a delivered parcel shipment", parcel.listCalls, parcel.addCalls)
	}
}
