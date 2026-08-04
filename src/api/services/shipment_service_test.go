package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
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

type stubParcelAppClient struct {
	deliveries []ParcelAppDelivery
	listCalls  int
	addCalls   int
	added      []string
	err        error
}

func (c *stubParcelAppClient) ListDeliveries(_ context.Context, _ string) ([]ParcelAppDelivery, error) {
	c.listCalls++
	if c.err != nil {
		return nil, c.err
	}
	return c.deliveries, nil
}

func (c *stubParcelAppClient) AddDelivery(_ context.Context, _ string, trackingNumber, _ string) error {
	c.addCalls++
	c.added = append(c.added, trackingNumber)
	return c.err
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
		&models.CoinJournal{},
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

func TestShipmentService_UpsertParcelShipment_AddsMissingParcelDelivery(t *testing.T) {
	h := setupShipmentServiceHarness(t)
	if err := h.db.AutoMigrate(&models.AppSetting{}); err != nil {
		t.Fatalf("migrate app_setting: %v", err)
	}
	if err := h.db.Model(&models.User{}).Where("id = ?", h.user.ID).Update("parcel_app_api_key", "parcel-key").Error; err != nil {
		t.Fatalf("set parcel key: %v", err)
	}
	settingsSvc := NewSettingsService(repository.NewSettingsRepository(h.db))
	if err := settingsSvc.SetSetting(SettingParcelAppEnabled, "true"); err != nil {
		t.Fatalf("enable parcel app: %v", err)
	}
	parcel := &stubParcelAppClient{}
	h.service.WithParcelAppSupport(repository.NewUserRepository(h.db), settingsSvc, NewDisabledCredentialEncryptionService(), parcel)

	shipment, err := h.service.UpsertShipmentForCoin(h.user.ID, h.coin.ID, models.ShipmentCarrierParcel, "PX-100", "", "")
	if err != nil {
		t.Fatalf("create parcel shipment: %v", err)
	}
	if shipment.Carrier != models.ShipmentCarrierParcel {
		t.Fatalf("carrier = %s, want parcel", shipment.Carrier)
	}
	if shipment.ManualOverrideEnabled {
		t.Fatalf("parcel shipments should not enable manual override by default")
	}
	if shipment.CurrentStatus != models.ShipmentStatusLabelCreated {
		t.Fatalf("status = %s, want %s", shipment.CurrentStatus, models.ShipmentStatusLabelCreated)
	}
	if parcel.listCalls != 1 || parcel.addCalls != 1 || len(parcel.added) != 1 || parcel.added[0] != "PX-100" {
		t.Fatalf("parcel calls = list:%d add:%d added:%v, want one list and one add for PX-100", parcel.listCalls, parcel.addCalls, parcel.added)
	}
}

func TestShipmentService_UpsertParcelShipment_PersistsWhenParcelAPIFails(t *testing.T) {
	h := setupShipmentServiceHarness(t)
	if err := h.db.AutoMigrate(&models.AppSetting{}); err != nil {
		t.Fatalf("migrate app_setting: %v", err)
	}
	if err := h.db.Model(&models.User{}).Where("id = ?", h.user.ID).Update("parcel_app_api_key", "parcel-key").Error; err != nil {
		t.Fatalf("set parcel key: %v", err)
	}
	settingsSvc := NewSettingsService(repository.NewSettingsRepository(h.db))
	if err := settingsSvc.SetSetting(SettingParcelAppEnabled, "true"); err != nil {
		t.Fatalf("enable parcel app: %v", err)
	}
	parcel := &stubParcelAppClient{err: errors.New("parcel rate limited")}
	h.service.WithParcelAppSupport(repository.NewUserRepository(h.db), settingsSvc, NewDisabledCredentialEncryptionService(), parcel)

	shipment, err := h.service.UpsertShipmentForCoin(h.user.ID, h.coin.ID, models.ShipmentCarrierParcel, "9402150105800000607499", "", "")
	if err != nil {
		t.Fatalf("create parcel shipment should persist despite ParcelApp error: %v", err)
	}
	if shipment.TrackingNumber != "9402150105800000607499" {
		t.Fatalf("tracking = %q, want saved tracking number", shipment.TrackingNumber)
	}
	if shipment.LastSyncError != "parcel rate limited" {
		t.Fatalf("last sync error = %q, want ParcelApp failure recorded", shipment.LastSyncError)
	}
}

func TestShipmentService_SyncParcelShipment_ReturnsShipmentWhenParcelAPIFails(t *testing.T) {
	h := setupShipmentServiceHarness(t)
	if err := h.db.AutoMigrate(&models.AppSetting{}); err != nil {
		t.Fatalf("migrate app_setting: %v", err)
	}
	if err := h.db.Model(&models.User{}).Where("id = ?", h.user.ID).Update("parcel_app_api_key", "parcel-key").Error; err != nil {
		t.Fatalf("set parcel key: %v", err)
	}
	settingsSvc := NewSettingsService(repository.NewSettingsRepository(h.db))
	if err := settingsSvc.SetSetting(SettingParcelAppEnabled, "true"); err != nil {
		t.Fatalf("enable parcel app: %v", err)
	}
	parcel := &stubParcelAppClient{}
	h.service.WithParcelAppSupport(repository.NewUserRepository(h.db), settingsSvc, NewDisabledCredentialEncryptionService(), parcel)
	shipment, err := h.service.UpsertShipmentForCoin(h.user.ID, h.coin.ID, models.ShipmentCarrierParcel, "9402150105800000607499", "", "")
	if err != nil {
		t.Fatalf("create parcel shipment: %v", err)
	}

	parcel.err = errors.New("parcel unavailable")
	synced, err := h.service.SyncShipment(context.Background(), shipment.ID, h.user.ID)
	if err != nil {
		t.Fatalf("sync parcel shipment should return saved shipment despite ParcelApp error: %v", err)
	}
	if synced.ID != shipment.ID {
		t.Fatalf("synced shipment id = %d, want %d", synced.ID, shipment.ID)
	}
	if synced.LastSyncError != "parcel unavailable" {
		t.Fatalf("last sync error = %q, want ParcelApp failure recorded", synced.LastSyncError)
	}
}

func TestShipmentService_SyncCandidates_GroupsParcelRequestsByUser(t *testing.T) {
	h := setupShipmentServiceHarness(t)
	if err := h.db.AutoMigrate(&models.AppSetting{}); err != nil {
		t.Fatalf("migrate app_setting: %v", err)
	}
	if err := h.db.Model(&models.User{}).Where("id = ?", h.user.ID).Update("parcel_app_api_key", "parcel-key").Error; err != nil {
		t.Fatalf("set parcel key: %v", err)
	}
	secondCoin := models.Coin{Name: "Denarius", UserID: h.user.ID}
	if err := h.db.Create(&secondCoin).Error; err != nil {
		t.Fatalf("create second coin: %v", err)
	}
	settingsSvc := NewSettingsService(repository.NewSettingsRepository(h.db))
	parcel := &stubParcelAppClient{}
	h.service.WithParcelAppSupport(repository.NewUserRepository(h.db), settingsSvc, NewDisabledCredentialEncryptionService(), parcel)
	if _, err := h.service.UpsertShipmentForCoin(h.user.ID, h.coin.ID, models.ShipmentCarrierParcel, "PX-100", "", ""); err != nil {
		t.Fatalf("create first parcel shipment: %v", err)
	}
	if _, err := h.service.UpsertShipmentForCoin(h.user.ID, secondCoin.ID, models.ShipmentCarrierParcel, "PX-200", "", ""); err != nil {
		t.Fatalf("create second parcel shipment: %v", err)
	}
	parcel.deliveries = []ParcelAppDelivery{
		{TrackingNumber: "PX-100", StatusCode: 2},
		{TrackingNumber: "PX-200", StatusCode: 4},
	}
	if err := settingsSvc.SetSetting(SettingParcelAppEnabled, "true"); err != nil {
		t.Fatalf("enable parcel app: %v", err)
	}

	carrier := models.ShipmentCarrierParcel
	summary, err := h.service.SyncCandidates(context.Background(), &carrier, 10)
	if err != nil {
		t.Fatalf("sync candidates: %v", err)
	}
	if summary.Checked != 2 || summary.Updated != 2 || summary.Failed != 0 {
		t.Fatalf("summary = %+v, want 2 checked/updated", summary)
	}
	if parcel.listCalls != 1 {
		t.Fatalf("parcel list calls = %d, want 1 per user", parcel.listCalls)
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

	journalEntries, err := repository.NewJournalRepository(h.db).GetEntries(h.coin.ID, h.user.ID)
	if err != nil {
		t.Fatalf("load journal entries: %v", err)
	}
	if len(journalEntries) == 0 {
		t.Fatalf("expected shipment status journal entry")
	}
	if !strings.Contains(journalEntries[0].Entry, "Shipment status updated to Exception") {
		t.Fatalf("journal entry = %q, expected shipment status update", journalEntries[0].Entry)
	}
}
