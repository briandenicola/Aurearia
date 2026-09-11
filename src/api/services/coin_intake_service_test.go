package services

import (
	"encoding/json"
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

var coinIntakeTestDBCounter uint64

func setupCoinIntakeTest(t *testing.T) (*CoinIntakeService, *gorm.DB) {
	t.Helper()
	dsn := fmt.Sprintf("file:coin_intake_%d_%d?mode=memory&cache=shared", time.Now().UnixNano(), atomic.AddUint64(&coinIntakeTestDBCounter, 1))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(
		&models.User{}, &models.StorageLocation{}, &models.Coin{}, &models.CoinImage{},
		&models.CoinReference{}, &models.ValueSnapshot{}, &models.CoinJournal{},
		&models.CoinIntakeDraft{},
	); err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE UNIQUE INDEX idx_coins_storage_location_slot_unique
		ON coins(storage_location_id, storage_slot) WHERE storage_slot IS NOT NULL`).Error; err != nil {
		t.Fatal(err)
	}
	coinRepo := repository.NewCoinRepository(db)
	coinService := NewCoinService(coinRepo, nil).
		WithStorageLocationSupport(repository.NewStorageLocationRepository(db))
	service := NewCoinIntakeService(repository.NewCoinIntakeDraftRepository(db), coinRepo, nil, nil).
		WithCoinService(coinService)
	return service, db
}

func seedCoinIntakeDraft(t *testing.T, db *gorm.DB, userID uint, payload map[string]interface{}) models.CoinIntakeDraft {
	t.Helper()
	encoded, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	draft := models.CoinIntakeDraft{
		UserID:            userID,
		DraftPayload:      string(encoded),
		ConfidenceSummary: `{}`,
		Evidence:          `[]`,
		UnresolvedFields:  `[]`,
		Status:            models.CoinIntakeDraftStatusDrafted,
		ExpiresAt:         time.Now().Add(time.Hour),
	}
	if err := db.Create(&draft).Error; err != nil {
		t.Fatal(err)
	}
	return draft
}

func TestCoinIntakeService_CommitDraftUsesSharedStorageAssignmentContract(t *testing.T) {
	service, db := setupCoinIntakeTest(t)
	rows, columns := 2, 2
	standard := models.StorageLocation{UserID: 1, Name: "Safe", Type: models.StorageLocationTypeStandard}
	tray := models.StorageLocation{UserID: 1, Name: "Tray", Type: models.StorageLocationTypeTray, Rows: &rows, Columns: &columns}
	foreignTray := models.StorageLocation{UserID: 2, Name: "Foreign", Type: models.StorageLocationTypeTray, Rows: &rows, Columns: &columns}
	if err := db.Create(&[]*models.StorageLocation{&standard, &tray, &foreignTray}).Error; err != nil {
		t.Fatal(err)
	}
	occupiedSlot := 2
	if err := db.Create(&models.Coin{UserID: 1, Name: "Occupant", Category: models.CategoryRoman, StorageLocationID: &tray.ID, StorageSlot: &occupiedSlot}).Error; err != nil {
		t.Fatal(err)
	}

	standardDraft := seedCoinIntakeDraft(t, db, 1, map[string]interface{}{
		"name": "Intake Standard", "category": "Roman", "storageLocationId": standard.ID,
	})
	result, err := service.CommitDraft(1, IntakeCommitRequest{DraftID: standardDraft.ID, Confirm: true})
	if err != nil {
		t.Fatalf("commit standard assignment: %v", err)
	}
	var standardCoin models.Coin
	if err := db.First(&standardCoin, result.CoinID).Error; err != nil {
		t.Fatal(err)
	}
	if standardCoin.StorageLocationID == nil || *standardCoin.StorageLocationID != standard.ID || standardCoin.StorageSlot != nil {
		t.Fatalf("standard intake assignment = %#v", standardCoin)
	}

	tests := []struct {
		name    string
		payload map[string]interface{}
		wantErr error
	}{
		{name: "missing tray slot", payload: map[string]interface{}{"name": "Missing", "category": "Roman", "storageLocationId": tray.ID}, wantErr: ErrStorageSlotRequired},
		{name: "out of range tray slot", payload: map[string]interface{}{"name": "Range", "category": "Roman", "storageLocationId": tray.ID, "storageSlot": 5}, wantErr: ErrStorageSlotInvalid},
		{name: "cross owner tray", payload: map[string]interface{}{"name": "Foreign", "category": "Roman", "storageLocationId": foreignTray.ID, "storageSlot": 1}, wantErr: ErrStorageLocationNotFound},
		{name: "occupied tray slot", payload: map[string]interface{}{"name": "Occupied", "category": "Roman", "storageLocationId": tray.ID, "storageSlot": occupiedSlot}, wantErr: ErrStorageSlotOccupied},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			draft := seedCoinIntakeDraft(t, db, 1, tt.payload)
			if _, err := service.CommitDraft(1, IntakeCommitRequest{DraftID: draft.ID, Confirm: true}); !errors.Is(err, tt.wantErr) {
				t.Fatalf("CommitDraft error = %v, want %v", err, tt.wantErr)
			}
			var persisted models.CoinIntakeDraft
			if err := db.First(&persisted, draft.ID).Error; err != nil {
				t.Fatal(err)
			}
			if persisted.Status != models.CoinIntakeDraftStatusDrafted || persisted.ConfirmedCoinID != nil {
				t.Fatalf("failed assignment partially confirmed draft: %#v", persisted)
			}
		})
	}
}
