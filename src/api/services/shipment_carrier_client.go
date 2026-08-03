package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
)

var ErrShipmentCarrierUnsupported = errors.New("unsupported shipment carrier")

type ShipmentCarrierClient interface {
	Carrier() models.ShipmentCarrier
	GetTracking(ctx context.Context, trackingNumber string) (ShipmentTrackingSnapshot, error)
}

type ShipmentCarrierClientRegistry struct {
	clients map[models.ShipmentCarrier]ShipmentCarrierClient
}

func NewShipmentCarrierClientRegistry(clients ...ShipmentCarrierClient) *ShipmentCarrierClientRegistry {
	registered := make(map[models.ShipmentCarrier]ShipmentCarrierClient, len(clients))
	for _, client := range clients {
		if client == nil {
			continue
		}
		registered[client.Carrier()] = client
	}
	return &ShipmentCarrierClientRegistry{clients: registered}
}

func (r *ShipmentCarrierClientRegistry) ClientForCarrier(carrier models.ShipmentCarrier) (ShipmentCarrierClient, error) {
	client, ok := r.clients[carrier]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrShipmentCarrierUnsupported, carrier)
	}
	return client, nil
}

type ShipmentTrackingSnapshot struct {
	Carrier             models.ShipmentCarrier
	TrackingNumber      string
	CurrentStatus       models.ShipmentStatus
	CurrentStatusSource models.ShipmentStatusSource
	RawCurrentStatus    string
	EstimatedDeliveryAt *time.Time
	DeliveredAt         *time.Time
	Events              []ShipmentTrackingEvent
	SyncedAt            time.Time
}

type ShipmentTrackingEvent struct {
	EventKey     string
	Status       models.ShipmentStatus
	StatusSource models.ShipmentStatusSource
	OccurredAt   time.Time
	Location     string
	Description  string
	RawStatus    string
	RawPayload   string
}

func NormalizeCarrierShipmentStatus(carrier models.ShipmentCarrier, rawStatus string) models.ShipmentStatus {
	_ = carrier // reserved for carrier-specific mappings
	return normalizeShipmentStatus(rawStatus)
}

func normalizeShipmentStatus(rawStatus string) models.ShipmentStatus {
	status := strings.TrimSpace(strings.ToLower(rawStatus))
	if status == "" {
		return models.ShipmentStatusUnknown
	}

	if containsAny(status, "out for delivery", "with delivery courier", "on vehicle for delivery", "out-for-delivery") {
		return models.ShipmentStatusOutForDelivery
	}
	if containsAny(status, "delivered", "delivery complete", "signed for by") {
		return models.ShipmentStatusDelivered
	}
	if containsAny(status, "exception", "failed delivery", "delivery attempted", "return to sender", "undeliverable") {
		return models.ShipmentStatusException
	}
	if containsAny(status, "returned", "return completed", "returned to sender") {
		return models.ShipmentStatusReturned
	}
	if containsAny(status, "in transit", "arrived at", "departed", "moving through network", "processed through facility") {
		return models.ShipmentStatusInTransit
	}
	if containsAny(status, "label created", "shipment information sent", "pre-shipment", "pending acceptance") {
		return models.ShipmentStatusLabelCreated
	}
	if containsAny(status, "pending", "not found", "unknown") {
		return models.ShipmentStatusPending
	}

	return models.ShipmentStatusUnknown
}

func containsAny(value string, fragments ...string) bool {
	for _, fragment := range fragments {
		if strings.Contains(value, fragment) {
			return true
		}
	}
	return false
}
