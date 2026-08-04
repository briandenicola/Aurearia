package services

import (
	"context"
	"encoding/json"
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
	ErrParcelAppDisabled           = errors.New("ParcelApp shipment tracking is disabled")
	ErrParcelAppAPIKeyRequired     = errors.New("ParcelApp API key is required")
)

type ShipmentService struct {
	shipmentRepo    *repository.ShipmentRepository
	coinRepo        *repository.CoinRepository
	carrierRegistry *ShipmentCarrierClientRegistry
	notifSvc        *NotificationService
	userRepo        *repository.UserRepository
	settingsSvc     *SettingsService
	credentials     *CredentialEncryptionService
	parcelClient    ParcelAppClient
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

func (s *ShipmentService) WithParcelAppSupport(
	userRepo *repository.UserRepository,
	settingsSvc *SettingsService,
	credentials *CredentialEncryptionService,
	parcelClient ParcelAppClient,
) *ShipmentService {
	s.userRepo = userRepo
	s.settingsSvc = settingsSvc
	s.credentials = credentials
	if s.credentials == nil {
		s.credentials = NewDisabledCredentialEncryptionService()
	}
	s.parcelClient = parcelClient
	return s
}

func (s *ShipmentService) UpsertShipmentForCoin(
	userID uint,
	coinID uint,
	carrier models.ShipmentCarrier,
	trackingNumber string,
	notes string,
	manualCarrierName string,
) (*models.Shipment, error) {
	coin, err := s.coinRepo.FindByID(coinID, userID)
	if err != nil {
		if repository.IsRecordNotFound(err) {
			return nil, ErrShipmentCoinNotFound
		}
		return nil, err
	}

	normalized, err := normalizeShipmentInput(carrier, trackingNumber, manualCarrierName)
	if err != nil {
		return nil, err
	}

	if normalized.carrier == models.ShipmentCarrierParcel {
		return s.upsertParcelShipmentForCoin(context.Background(), userID, coinID, coin.Name, normalized.trackingNumber, notes)
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
		shipment.ManualOverrideEnabled = true
		if shipment.CurrentStatus == "" {
			shipment.CurrentStatus = models.ShipmentStatusPending
		}
		shipment.CurrentStatusSource = models.ShipmentStatusSourceManual
		shipment.ManualOverrideStatus = shipment.CurrentStatus
		if updateErr := s.shipmentRepo.Update(shipment); updateErr != nil {
			return nil, updateErr
		}
		return s.shipmentRepo.GetByIDForUser(shipment.ID, userID)
	}

	now := time.Now().UTC()
	newShipment := &models.Shipment{
		UserID:                  userID,
		CoinID:                  coinID,
		Carrier:                 normalized.carrier,
		ManualCarrierName:       normalized.manualCarrierName,
		TrackingNumber:          normalized.trackingNumber,
		CurrentStatus:           models.ShipmentStatusPending,
		CurrentStatusSource:     models.ShipmentStatusSourceManual,
		Notes:                   strings.TrimSpace(notes),
		ManualOverrideEnabled:   true,
		ManualOverrideStatus:    models.ShipmentStatusPending,
		ManualOverrideUpdatedAt: &now,
	}
	if err := s.shipmentRepo.Create(newShipment); err != nil {
		return nil, err
	}
	return s.shipmentRepo.GetByIDForUser(newShipment.ID, userID)
}

func (s *ShipmentService) upsertParcelShipmentForCoin(
	ctx context.Context,
	userID uint,
	coinID uint,
	coinTitle string,
	trackingNumber string,
	notes string,
) (*models.Shipment, error) {
	shipment, err := s.shipmentRepo.GetByCoinIDForUser(coinID, userID)
	if err != nil && !repository.IsRecordNotFound(err) {
		return nil, err
	}

	if shipment != nil && err == nil {
		shipment.Carrier = models.ShipmentCarrierParcel
		shipment.ManualCarrierName = ""
		shipment.TrackingNumber = trackingNumber
		shipment.Notes = strings.TrimSpace(notes)
		shipment.ManualOverrideEnabled = false
		shipment.ManualOverrideStatus = ""
		shipment.ManualOverrideNote = ""
		shipment.ManualOverrideUpdatedAt = nil
		if shipment.CurrentStatus == "" {
			shipment.CurrentStatus = models.ShipmentStatusPending
		}
		if shipment.CurrentStatusSource == "" || shipment.CurrentStatusSource == models.ShipmentStatusSourceManual {
			shipment.CurrentStatusSource = models.ShipmentStatusSourceAPI
		}
		if updateErr := s.shipmentRepo.Update(shipment); updateErr != nil {
			return nil, updateErr
		}
	} else {
		newShipment := &models.Shipment{
			UserID:                userID,
			CoinID:                coinID,
			Carrier:               models.ShipmentCarrierParcel,
			TrackingNumber:        trackingNumber,
			CurrentStatus:         models.ShipmentStatusPending,
			CurrentStatusSource:   models.ShipmentStatusSourceAPI,
			Notes:                 strings.TrimSpace(notes),
			ManualOverrideEnabled: false,
		}
		if err := s.shipmentRepo.Create(newShipment); err != nil {
			return nil, err
		}
		shipment = newShipment
	}

	if s.parcelAppEnabled() {
		updated, err := s.syncParcelShipment(ctx, shipment, strings.TrimSpace(coinTitle), true)
		if err != nil {
			return nil, err
		}
		return updated, nil
	}
	return s.shipmentRepo.GetByIDForUser(shipment.ID, userID)
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

	previousStatus := shipment.CurrentStatus
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

	if enabled && previousStatus != status {
		entry := &models.CoinJournal{
			CoinID: shipment.CoinID,
			UserID: shipment.UserID,
			Entry:  buildShipmentStatusJournalEntry(status, shipment.ManualOverrideNote),
		}
		if err := s.coinRepo.CreateJournalEntry(entry); err != nil {
			return nil, err
		}
	}

	return s.GetShipmentByID(userID, shipmentID)
}

func (s *ShipmentService) SyncShipment(ctx context.Context, shipmentID, userID uint) (*models.Shipment, error) {
	shipment, err := s.GetShipmentByID(userID, shipmentID)
	if err != nil {
		return nil, err
	}
	updated, err := s.syncSingleShipment(ctx, shipment, true)
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
	parcelByUser := map[uint][]models.Shipment{}
	for _, candidate := range candidates {
		if candidate.Carrier == models.ShipmentCarrierParcel {
			parcelByUser[candidate.UserID] = append(parcelByUser[candidate.UserID], candidate)
			continue
		}
		_, syncErr := s.syncSingleShipment(ctx, &candidate, false)
		if syncErr != nil {
			summary.Failed++
			if s.logger != nil {
				s.logger.Warn("shipment", "sync failed shipment=%d user=%d: %v", candidate.ID, candidate.UserID, syncErr)
			}
			continue
		}
		summary.Updated++
	}
	for userID, shipments := range parcelByUser {
		userSummary := s.syncParcelShipmentsForUser(ctx, userID, shipments)
		summary.Updated += userSummary.Updated
		summary.Failed += userSummary.Failed
	}
	return summary, nil
}

func (s *ShipmentService) syncSingleShipment(ctx context.Context, shipment *models.Shipment, allowParcelCreate bool) (*models.Shipment, error) {
	if shipment.ManualOverrideEnabled {
		return shipment, nil
	}
	if shipment.Carrier == models.ShipmentCarrierParcel {
		return s.syncParcelShipment(ctx, shipment, "", allowParcelCreate)
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

func (s *ShipmentService) syncParcelShipmentsForUser(ctx context.Context, userID uint, shipments []models.Shipment) ShipmentSyncSummary {
	summary := ShipmentSyncSummary{Checked: len(shipments)}
	if !s.parcelAppEnabled() {
		for _, shipment := range shipments {
			_ = s.shipmentRepo.MarkSyncFailure(shipment.ID, shipment.UserID, ErrParcelAppDisabled.Error())
		}
		summary.Failed = len(shipments)
		return summary
	}
	apiKey, err := s.parcelAPIKeyForUser(userID)
	if err != nil {
		for _, shipment := range shipments {
			_ = s.shipmentRepo.MarkSyncFailure(shipment.ID, shipment.UserID, err.Error())
		}
		summary.Failed = len(shipments)
		return summary
	}
	deliveries, err := s.parcelClient.ListDeliveries(ctx, apiKey)
	if err != nil {
		for _, shipment := range shipments {
			_ = s.shipmentRepo.MarkSyncFailure(shipment.ID, shipment.UserID, err.Error())
		}
		summary.Failed = len(shipments)
		return summary
	}
	byTracking := parcelDeliveriesByTracking(deliveries)
	for i := range shipments {
		delivery, ok := byTracking[normalizeTrackingLookup(shipments[i].TrackingNumber)]
		if !ok {
			err := errors.New("ParcelApp delivery not found")
			_ = s.shipmentRepo.MarkSyncFailure(shipments[i].ID, shipments[i].UserID, err.Error())
			summary.Failed++
			continue
		}
		if _, err := s.applyShipmentSnapshot(&shipments[i], parcelDeliveryToSnapshot(delivery)); err != nil {
			summary.Failed++
			continue
		}
		summary.Updated++
	}
	return summary
}

func (s *ShipmentService) syncParcelShipment(ctx context.Context, shipment *models.Shipment, description string, allowCreate bool) (*models.Shipment, error) {
	if !s.parcelAppEnabled() {
		_ = s.shipmentRepo.MarkSyncFailure(shipment.ID, shipment.UserID, ErrParcelAppDisabled.Error())
		return nil, ErrParcelAppDisabled
	}
	apiKey, err := s.parcelAPIKeyForUser(shipment.UserID)
	if err != nil {
		_ = s.shipmentRepo.MarkSyncFailure(shipment.ID, shipment.UserID, err.Error())
		return nil, err
	}
	deliveries, err := s.parcelClient.ListDeliveries(ctx, apiKey)
	if err != nil {
		_ = s.shipmentRepo.MarkSyncFailure(shipment.ID, shipment.UserID, err.Error())
		return nil, err
	}
	if delivery, ok := parcelDeliveriesByTracking(deliveries)[normalizeTrackingLookup(shipment.TrackingNumber)]; ok {
		return s.applyShipmentSnapshot(shipment, parcelDeliveryToSnapshot(delivery))
	}
	if !allowCreate {
		err := errors.New("ParcelApp delivery not found")
		_ = s.shipmentRepo.MarkSyncFailure(shipment.ID, shipment.UserID, err.Error())
		return nil, err
	}
	if strings.TrimSpace(description) == "" {
		coin, err := s.coinRepo.FindByID(shipment.CoinID, shipment.UserID)
		if err == nil {
			description = coin.Name
		}
	}
	if strings.TrimSpace(description) == "" {
		description = "Coin shipment"
	}
	if err := s.parcelClient.AddDelivery(ctx, apiKey, shipment.TrackingNumber, description); err != nil {
		_ = s.shipmentRepo.MarkSyncFailure(shipment.ID, shipment.UserID, err.Error())
		return nil, err
	}
	snapshot := ShipmentTrackingSnapshot{
		Carrier:             models.ShipmentCarrierParcel,
		TrackingNumber:      shipment.TrackingNumber,
		CurrentStatus:       models.ShipmentStatusLabelCreated,
		CurrentStatusSource: models.ShipmentStatusSourceAPI,
		Events: []ShipmentTrackingEvent{{
			EventKey:     fmt.Sprintf("parcel:%s:created", normalizeTrackingLookup(shipment.TrackingNumber)),
			Status:       models.ShipmentStatusLabelCreated,
			StatusSource: models.ShipmentStatusSourceAPI,
			OccurredAt:   time.Now().UTC(),
			Description:  "Added to ParcelApp",
			RawStatus:    "8",
		}},
	}
	return s.applyShipmentSnapshot(shipment, snapshot)
}

func (s *ShipmentService) applyShipmentSnapshot(shipment *models.Shipment, snapshot ShipmentTrackingSnapshot) (*models.Shipment, error) {
	for _, event := range snapshot.Events {
		rawPayload := event.RawPayload
		if rawPayload == "" {
			if encoded, err := json.Marshal(event); err == nil {
				rawPayload = string(encoded)
			}
		}
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
			RawPayload:   rawPayload,
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

func (s *ShipmentService) parcelAppEnabled() bool {
	return s.settingsSvc != nil &&
		s.parcelClient != nil &&
		s.settingsSvc.GetSetting(SettingParcelAppEnabled) == "true"
}

func (s *ShipmentService) parcelAPIKeyForUser(userID uint) (string, error) {
	if s.userRepo == nil {
		return "", ErrParcelAppAPIKeyRequired
	}
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(user.ParcelAppAPIKey) == "" {
		return "", ErrParcelAppAPIKeyRequired
	}
	credentials := s.credentials
	if credentials == nil {
		credentials = NewDisabledCredentialEncryptionService()
	}
	plain, _, err := credentials.DecryptStringWithAAD(user.ParcelAppAPIKey, UserCredentialAAD(userID, "parcel_app_api_key"))
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(plain) == "" {
		return "", ErrParcelAppAPIKeyRequired
	}
	return strings.TrimSpace(plain), nil
}

func parcelDeliveriesByTracking(deliveries []ParcelAppDelivery) map[string]ParcelAppDelivery {
	out := make(map[string]ParcelAppDelivery, len(deliveries))
	for _, delivery := range deliveries {
		key := normalizeTrackingLookup(delivery.TrackingNumber)
		if key != "" {
			out[key] = delivery
		}
	}
	return out
}

func normalizeTrackingLookup(value string) string {
	return strings.ToUpper(strings.TrimSpace(value))
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
	case models.ShipmentCarrierUSPS, models.ShipmentCarrierUPS, models.ShipmentCarrierFedEx, models.ShipmentCarrierParcel:
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

func buildShipmentStatusJournalEntry(status models.ShipmentStatus, note string) string {
	entry := fmt.Sprintf("Shipment status updated to %s.", shipmentStatusLabel(status))
	if trimmedNote := strings.TrimSpace(note); trimmedNote != "" {
		entry += " Note: " + trimmedNote
	}
	return entry
}

func shipmentStatusLabel(status models.ShipmentStatus) string {
	parts := strings.Split(string(status), "_")
	for idx, part := range parts {
		if part == "" {
			continue
		}
		parts[idx] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, " ")
}
