package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
)

var (
	ErrShipmentNotFound            = errors.New("shipment not found")
	ErrShipmentCoinNotFound        = errors.New("coin not found")
	ErrShipmentTrackingRequired    = errors.New("tracking number is required")
	ErrShipmentCarrierRequired     = errors.New("shipment carrier is required")
	ErrShipmentCarrierNameRequired = errors.New("manual carrier name is required when carrier is other")
)

type ShipmentService struct {
	shipmentRepo    *repository.ShipmentRepository
	coinRepo        *repository.CoinRepository
	carrierRegistry *ShipmentCarrierClientRegistry
	notifSvc        *NotificationService
	logger          *Logger
}

type ShipmentSyncSummary struct {
	Checked int `json:"checked"`
	Updated int `json:"updated"`
	Failed  int `json:"failed"`
}

func NewShipmentService(
	shipmentRepo *repository.ShipmentRepository,
	coinRepo *repository.CoinRepository,
	carrierRegistry *ShipmentCarrierClientRegistry,
	notifSvc *NotificationService,
	logger *Logger,
) *ShipmentService {
	return &ShipmentService{
		shipmentRepo:    shipmentRepo,
		coinRepo:        coinRepo,
		carrierRegistry: carrierRegistry,
		notifSvc:        notifSvc,
		logger:          logger,
	}
}

func (s *ShipmentService) UpsertShipmentForCoin(
	userID uint,
	coinID uint,
	carrier models.ShipmentCarrier,
	trackingNumber string,
	notes string,
	manualCarrierName string,
) (*models.Shipment, error) {
	if _, err := s.coinRepo.FindByID(coinID, userID); err != nil {
		if repository.IsRecordNotFound(err) {
			return nil, ErrShipmentCoinNotFound
		}
		return nil, err
	}

	normalized, err := normalizeShipmentInput(carrier, trackingNumber, manualCarrierName)
	if err != nil {
		return nil, err
	}

	shipment, err := s.shipmentRepo.GetByCoinIDForUser(coinID, userID)
	if err != nil && !repository.IsRecordNotFound(err) {
		return nil, err
	}

	if shipment != nil && err == nil {
		shipment.Carrier = normalized.carrier
		shipment.ManualCarrierName = normalized.manualCarrierName
		shipment.TrackingNumber = normalized.trackingNumber
		shipment.Notes = strings.TrimSpace(notes)
		if updateErr := s.shipmentRepo.Update(shipment); updateErr != nil {
			return nil, updateErr
		}
		return s.shipmentRepo.GetByIDForUser(shipment.ID, userID)
	}

	newShipment := &models.Shipment{
		UserID:            userID,
		CoinID:            coinID,
		Carrier:           normalized.carrier,
		ManualCarrierName: normalized.manualCarrierName,
		TrackingNumber:    normalized.trackingNumber,
		CurrentStatus:     models.ShipmentStatusPending,
		Notes:             strings.TrimSpace(notes),
	}
	if err := s.shipmentRepo.Create(newShipment); err != nil {
		return nil, err
	}
	return s.shipmentRepo.GetByIDForUser(newShipment.ID, userID)
}

func (s *ShipmentService) GetShipmentForCoin(userID, coinID uint) (*models.Shipment, error) {
	shipment, err := s.shipmentRepo.GetByCoinIDForUser(coinID, userID)
	if err != nil {
		if repository.IsRecordNotFound(err) {
			return nil, ErrShipmentNotFound
		}
		return nil, err
	}
	return shipment, nil
}

func (s *ShipmentService) GetShipmentByID(userID, shipmentID uint) (*models.Shipment, error) {
	shipment, err := s.shipmentRepo.GetByIDForUser(shipmentID, userID)
	if err != nil {
		if repository.IsRecordNotFound(err) {
			return nil, ErrShipmentNotFound
		}
		return nil, err
	}
	return shipment, nil
}

func (s *ShipmentService) DeleteShipment(userID, shipmentID uint) error {
	err := s.shipmentRepo.DeleteForUser(shipmentID, userID)
	if err != nil {
		if repository.IsRecordNotFound(err) {
			return ErrShipmentNotFound
		}
		return err
	}
	return nil
}

func (s *ShipmentService) SetManualOverride(userID, shipmentID uint, enabled bool, status models.ShipmentStatus, note string) (*models.Shipment, error) {
	shipment, err := s.GetShipmentByID(userID, shipmentID)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	shipment.ManualOverrideEnabled = enabled
	shipment.ManualOverrideNote = strings.TrimSpace(note)
	if enabled {
		shipment.ManualOverrideStatus = status
		shipment.CurrentStatus = status
		shipment.CurrentStatusSource = models.ShipmentStatusSourceManual
		shipment.ManualOverrideUpdatedAt = &now
	} else {
		shipment.ManualOverrideStatus = ""
		shipment.ManualOverrideUpdatedAt = nil
	}

	if err := s.shipmentRepo.Update(shipment); err != nil {
		return nil, err
	}
	return s.GetShipmentByID(userID, shipmentID)
}

func (s *ShipmentService) SyncShipment(ctx context.Context, shipmentID, userID uint) (*models.Shipment, error) {
	shipment, err := s.GetShipmentByID(userID, shipmentID)
	if err != nil {
		return nil, err
	}
	updated, err := s.syncSingleShipment(ctx, shipment)
	if err != nil {
		return nil, err
	}
	return updated, nil
}

func (s *ShipmentService) SyncCandidates(ctx context.Context, carrier *models.ShipmentCarrier, limit int) (ShipmentSyncSummary, error) {
	candidates, err := s.shipmentRepo.ListSyncCandidates(carrier, limit)
	if err != nil {
		return ShipmentSyncSummary{}, err
	}
	summary := ShipmentSyncSummary{Checked: len(candidates)}
	for _, candidate := range candidates {
		_, syncErr := s.syncSingleShipment(ctx, &candidate)
		if syncErr != nil {
			summary.Failed++
			if s.logger != nil {
				s.logger.Warn("shipment", "sync failed shipment=%d user=%d: %v", candidate.ID, candidate.UserID, syncErr)
			}
			continue
		}
		summary.Updated++
	}
	return summary, nil
}

func (s *ShipmentService) syncSingleShipment(ctx context.Context, shipment *models.Shipment) (*models.Shipment, error) {
	if shipment.ManualOverrideEnabled {
		return shipment, nil
	}

	client, err := s.carrierRegistry.ClientForCarrier(shipment.Carrier)
	if err != nil {
		_ = s.shipmentRepo.MarkSyncFailure(shipment.ID, shipment.UserID, err.Error())
		return nil, err
	}

	snapshot, err := client.GetTracking(ctx, shipment.TrackingNumber)
	if err != nil {
		_ = s.shipmentRepo.MarkSyncFailure(shipment.ID, shipment.UserID, err.Error())
		return nil, err
	}

	for _, event := range snapshot.Events {
		_, upsertErr := s.shipmentRepo.UpsertEvent(&models.ShipmentEvent{
			ShipmentID:   shipment.ID,
			UserID:       shipment.UserID,
			EventKey:     event.EventKey,
			Status:       event.Status,
			StatusSource: event.StatusSource,
			OccurredAt:   event.OccurredAt,
			Location:     event.Location,
			Description:  event.Description,
			RawStatus:    event.RawStatus,
			RawPayload:   event.RawPayload,
		})
		if upsertErr != nil {
			return nil, upsertErr
		}
	}

	previousStatus := shipment.CurrentStatus
	if err := s.shipmentRepo.MarkSyncSuccess(
		shipment.ID,
		shipment.UserID,
		snapshot.CurrentStatus,
		snapshot.CurrentStatusSource,
		snapshot.EstimatedDeliveryAt,
		snapshot.DeliveredAt,
	); err != nil {
		return nil, err
	}

	updated, err := s.shipmentRepo.GetByIDForUser(shipment.ID, shipment.UserID)
	if err != nil {
		return nil, err
	}

	if s.notifSvc != nil && shouldNotifyShipmentTransition(previousStatus, updated.CurrentStatus) {
		go s.notifSvc.NotifyShipmentStatusTransition(updated.UserID, updated.CoinID, updated.ID, previousStatus, updated.CurrentStatus)
	}

	return updated, nil
}

type normalizedShipmentInput struct {
	carrier           models.ShipmentCarrier
	trackingNumber    string
	manualCarrierName string
}

func normalizeShipmentInput(
	carrier models.ShipmentCarrier,
	trackingNumber string,
	manualCarrierName string,
) (normalizedShipmentInput, error) {
	normalizedCarrier := models.ShipmentCarrier(strings.TrimSpace(strings.ToLower(string(carrier))))
	if normalizedCarrier == "" {
		return normalizedShipmentInput{}, ErrShipmentCarrierRequired
	}
	normalizedTracking := strings.TrimSpace(trackingNumber)
	if normalizedTracking == "" {
		return normalizedShipmentInput{}, ErrShipmentTrackingRequired
	}

	switch normalizedCarrier {
	case models.ShipmentCarrierUSPS, models.ShipmentCarrierUPS, models.ShipmentCarrierFedEx:
		return normalizedShipmentInput{
			carrier:        normalizedCarrier,
			trackingNumber: normalizedTracking,
		}, nil
	case models.ShipmentCarrierOther:
		normalizedName := strings.TrimSpace(manualCarrierName)
		if normalizedName == "" {
			return normalizedShipmentInput{}, ErrShipmentCarrierNameRequired
		}
		return normalizedShipmentInput{
			carrier:           normalizedCarrier,
			trackingNumber:    normalizedTracking,
			manualCarrierName: normalizedName,
		}, nil
	default:
		return normalizedShipmentInput{}, fmt.Errorf("%w: %s", ErrShipmentCarrierRequired, normalizedCarrier)
	}
}

func shouldNotifyShipmentTransition(previous, current models.ShipmentStatus) bool {
	if previous == current {
		return false
	}
	switch current {
	case models.ShipmentStatusOutForDelivery, models.ShipmentStatusDelivered, models.ShipmentStatusException, models.ShipmentStatusReturned:
		return true
	default:
		return false
	}
}
