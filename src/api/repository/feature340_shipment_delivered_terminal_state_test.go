package repository

// Independent QA regression coverage for the shipment "delivered is a
// terminal state for automatic sync" change, owned by Brutus (Tester/QA).
//
// Deliberately a NEW dedicated file, separate from
// shipment_repository_test.go (Cassius's own test file, which already adds
// one delivered-shipment fixture to its existing
// TestShipmentRepository_ListSyncCandidates_FiltersManualOverride test) —
// this avoids merge collisions and adds coverage his single fixture does
// not: every non-delivered status remains eligible, a delivered shipment is
// excluded from ListSyncCandidates regardless of whether
// ManualOverrideEnabled is true or false (i.e. whether the delivered status
// was reached via a manual override or via a provider/carrier sync write),
// and reverting a delivered shipment back to a non-delivered status resumes
// its eligibility for automatic polling.

import (
	"testing"

	"github.com/briandenicola/ancient-coins-api/models"
)

func TestFeature340_ListSyncCandidates_ExcludesDeliveredRegardlessOfManualOverrideFlag(t *testing.T) {
	repo := setupShipmentRepository(t)

	deliveredManual := &models.Shipment{
		UserID:                3001,
		CoinID:                3101,
		Carrier:               models.ShipmentCarrierUSPS,
		TrackingNumber:        "9400-3101",
		CurrentStatus:         models.ShipmentStatusDelivered,
		CurrentStatusSource:   models.ShipmentStatusSourceManual,
		ManualOverrideEnabled: true, // delivered reached via a manual override
	}
	// Delivered via provider/API sync, with the manual override flag left
	// false — this is the case the plain `manual_override_enabled = false`
	// filter alone would NOT catch, so it specifically proves the new
	// `current_status <> delivered` clause is doing real work.
	deliveredProviderSet := &models.Shipment{
		UserID:                3001,
		CoinID:                3102,
		Carrier:               models.ShipmentCarrierUSPS,
		TrackingNumber:        "9400-3102",
		CurrentStatus:         models.ShipmentStatusDelivered,
		CurrentStatusSource:   models.ShipmentStatusSourceAPI,
		ManualOverrideEnabled: false,
	}
	stillInTransit := &models.Shipment{
		UserID:         3001,
		CoinID:         3103,
		Carrier:        models.ShipmentCarrierUSPS,
		TrackingNumber: "9400-3103",
		CurrentStatus:  models.ShipmentStatusInTransit,
	}

	for _, shipment := range []*models.Shipment{deliveredManual, deliveredProviderSet, stillInTransit} {
		if err := repo.Create(shipment); err != nil {
			t.Fatalf("create shipment: %v", err)
		}
	}

	carrier := models.ShipmentCarrierUSPS
	candidates, err := repo.ListSyncCandidates(&carrier, 10)
	if err != nil {
		t.Fatalf("list sync candidates: %v", err)
	}
	if len(candidates) != 1 {
		t.Fatalf("candidate len = %d, want 1 (only the still-in-transit shipment)", len(candidates))
	}
	if candidates[0].CoinID != stillInTransit.CoinID {
		t.Fatalf("candidate coin_id = %d, want %d", candidates[0].CoinID, stillInTransit.CoinID)
	}
}

func TestFeature340_ListSyncCandidates_AllNonDeliveredStatusesRemainEligible(t *testing.T) {
	repo := setupShipmentRepository(t)

	nonDeliveredStatuses := []models.ShipmentStatus{
		models.ShipmentStatusPending,
		models.ShipmentStatusLabelCreated,
		models.ShipmentStatusInTransit,
		models.ShipmentStatusOutForDelivery,
		models.ShipmentStatusException,
		models.ShipmentStatusReturned,
		models.ShipmentStatusUnknown,
	}

	for i, status := range nonDeliveredStatuses {
		shipment := &models.Shipment{
			UserID:         4001,
			CoinID:         uint(4100 + i),
			Carrier:        models.ShipmentCarrierUSPS,
			TrackingNumber: "9400-41" + string(rune('0'+i)),
			CurrentStatus:  status,
		}
		if err := repo.Create(shipment); err != nil {
			t.Fatalf("create shipment status=%s: %v", status, err)
		}
	}

	carrier := models.ShipmentCarrierUSPS
	candidates, err := repo.ListSyncCandidates(&carrier, 100)
	if err != nil {
		t.Fatalf("list sync candidates: %v", err)
	}
	if len(candidates) != len(nonDeliveredStatuses) {
		t.Fatalf("candidate len = %d, want %d — every non-delivered status must remain sync-eligible", len(candidates), len(nonDeliveredStatuses))
	}
}

func TestFeature340_ListSyncCandidates_RevertingFromDeliveredResumesPolling(t *testing.T) {
	repo := setupShipmentRepository(t)

	shipment := &models.Shipment{
		UserID:         5001,
		CoinID:         5101,
		Carrier:        models.ShipmentCarrierUSPS,
		TrackingNumber: "9400-5101",
		CurrentStatus:  models.ShipmentStatusDelivered,
	}
	if err := repo.Create(shipment); err != nil {
		t.Fatalf("create shipment: %v", err)
	}

	carrier := models.ShipmentCarrierUSPS
	candidates, err := repo.ListSyncCandidates(&carrier, 10)
	if err != nil {
		t.Fatalf("list sync candidates (delivered): %v", err)
	}
	if len(candidates) != 0 {
		t.Fatalf("candidate len = %d, want 0 while delivered", len(candidates))
	}

	// Simulate a correction: the shipment is returned to the carrier and its
	// status is reverted to "exception" (e.g. an incorrect delivery scan was
	// corrected, or the package was returned to sender after being marked
	// delivered in error). Automatic polling must resume.
	shipment.CurrentStatus = models.ShipmentStatusException
	if err := repo.Update(shipment); err != nil {
		t.Fatalf("revert shipment status: %v", err)
	}

	candidates, err = repo.ListSyncCandidates(&carrier, 10)
	if err != nil {
		t.Fatalf("list sync candidates (reverted): %v", err)
	}
	if len(candidates) != 1 {
		t.Fatalf("candidate len = %d, want 1 after reverting away from delivered", len(candidates))
	}
	if candidates[0].ID != shipment.ID {
		t.Fatalf("candidate id = %d, want %d", candidates[0].ID, shipment.ID)
	}
}
