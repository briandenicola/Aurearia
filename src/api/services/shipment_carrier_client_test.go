package services

import (
	"context"
	"errors"
	"testing"

	"github.com/briandenicola/ancient-coins-api/models"
)

type stubShipmentClient struct {
	carrier models.ShipmentCarrier
}

func (s *stubShipmentClient) Carrier() models.ShipmentCarrier {
	return s.carrier
}

func (s *stubShipmentClient) GetTracking(context.Context, string) (ShipmentTrackingSnapshot, error) {
	return ShipmentTrackingSnapshot{}, nil
}

func TestShipmentCarrierClientRegistry_ResolvesRegisteredCarrier(t *testing.T) {
	registry := NewShipmentCarrierClientRegistry(
		&stubShipmentClient{carrier: models.ShipmentCarrierUSPS},
		&stubShipmentClient{carrier: models.ShipmentCarrierUPS},
	)

	client, err := registry.ClientForCarrier(models.ShipmentCarrierUPS)
	if err != nil {
		t.Fatalf("resolve client: %v", err)
	}
	if client.Carrier() != models.ShipmentCarrierUPS {
		t.Fatalf("resolved carrier = %s, want %s", client.Carrier(), models.ShipmentCarrierUPS)
	}
}

func TestShipmentCarrierClientRegistry_ReturnsUnsupportedError(t *testing.T) {
	registry := NewShipmentCarrierClientRegistry(&stubShipmentClient{carrier: models.ShipmentCarrierUSPS})

	_, err := registry.ClientForCarrier(models.ShipmentCarrierFedEx)
	if !errors.Is(err, ErrShipmentCarrierNotConfigured) {
		t.Fatalf("expected not configured error, got: %v", err)
	}
}

func TestShipmentCarrierClientRegistry_NormalizesCarrierValue(t *testing.T) {
	registry := NewShipmentCarrierClientRegistry(&stubShipmentClient{carrier: models.ShipmentCarrierFedEx})

	client, err := registry.ClientForCarrier(models.ShipmentCarrier(" FedEx "))
	if err != nil {
		t.Fatalf("resolve normalized carrier: %v", err)
	}
	if client.Carrier() != models.ShipmentCarrierFedEx {
		t.Fatalf("resolved carrier = %s, want %s", client.Carrier(), models.ShipmentCarrierFedEx)
	}
}

func TestShipmentCarrierClientRegistry_UnknownCarrierReturnsUnsupportedError(t *testing.T) {
	registry := NewShipmentCarrierClientRegistry(&stubShipmentClient{carrier: models.ShipmentCarrierUSPS})

	_, err := registry.ClientForCarrier(models.ShipmentCarrier("dhl"))
	if !errors.Is(err, ErrShipmentCarrierUnsupported) {
		t.Fatalf("expected unsupported carrier error, got: %v", err)
	}
}

func TestNormalizeCarrierShipmentStatus(t *testing.T) {
	tests := []struct {
		name     string
		raw      string
		expected models.ShipmentStatus
	}{
		{name: "out for delivery", raw: "Out for Delivery", expected: models.ShipmentStatusOutForDelivery},
		{name: "delivered", raw: "Delivered, Left with Individual", expected: models.ShipmentStatusDelivered},
		{name: "exception", raw: "Delivery Exception - Address Issue", expected: models.ShipmentStatusException},
		{name: "returned", raw: "Returned to Sender", expected: models.ShipmentStatusReturned},
		{name: "in transit", raw: "In Transit to Next Facility", expected: models.ShipmentStatusInTransit},
		{name: "label created", raw: "Label Created", expected: models.ShipmentStatusLabelCreated},
		{name: "pending", raw: "Pending", expected: models.ShipmentStatusPending},
		{name: "unknown", raw: "Custom status from carrier", expected: models.ShipmentStatusUnknown},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := NormalizeCarrierShipmentStatus(models.ShipmentCarrierUSPS, tc.raw)
			if got != tc.expected {
				t.Fatalf("status for %q = %s, want %s", tc.raw, got, tc.expected)
			}
		})
	}
}
