package services

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
)

func TestStorageAssignmentContractAcrossCoinCreationWorkflows(t *testing.T) {
	db := setupTestDB(t)
	if err := db.Exec(`CREATE UNIQUE INDEX idx_coins_storage_location_slot_unique
		ON coins(storage_location_id, storage_slot) WHERE storage_slot IS NOT NULL`).Error; err != nil {
		t.Fatal(err)
	}
	coinSvc := newTestCoinServiceWithStorage(db)
	rows, columns := 2, 2
	tray := models.StorageLocation{UserID: 1, Name: "Tray", Type: models.StorageLocationTypeTray, Rows: &rows, Columns: &columns}
	standard := models.StorageLocation{UserID: 1, Name: "Safe", Type: models.StorageLocationTypeStandard}
	if err := db.Create(&[]*models.StorageLocation{&tray, &standard}).Error; err != nil {
		t.Fatal(err)
	}

	t.Run("manual create and JSON clients share validation", func(t *testing.T) {
		manual := &models.Coin{UserID: 1, Name: "Manual", Category: models.CategoryRoman, StorageLocationID: &tray.ID}
		if err := coinSvc.CreateCoin(manual); !errors.Is(err, ErrStorageSlotRequired) {
			t.Fatalf("manual create error = %v", err)
		}

		var fromJSON models.Coin
		payload := []byte(`{"name":"JSON client","category":"Roman","userId":1,"storageLocationId":` +
			jsonUint(tray.ID) + `,"storageSlot":1}`)
		if err := json.Unmarshal(payload, &fromJSON); err != nil {
			t.Fatal(err)
		}
		if err := coinSvc.CreateCoin(&fromJSON); err != nil {
			t.Fatalf("JSON client assignment: %v", err)
		}
	})

	t.Run("manual update, duplicate, and bulk preserve their placement rules", func(t *testing.T) {
		slotTwo := 2
		coin := &models.Coin{UserID: 1, Name: "Workflow coin", Category: models.CategoryRoman, StorageLocationID: &tray.ID, StorageSlot: &slotTwo}
		if err := coinSvc.CreateCoin(coin); err != nil {
			t.Fatal(err)
		}
		if err := coinSvc.UpdateCoinWithAssignmentFields(coin, &models.Coin{StorageLocationID: &standard.ID}, nil, 1, "manual", true, false); err != nil {
			t.Fatalf("manual standard move: %v", err)
		}
		if coin.StorageSlot != nil {
			t.Fatalf("manual move retained stale slot: %#v", coin)
		}

		coin.StorageLocationID, coin.StorageSlot = &tray.ID, &slotTwo
		if err := repository.NewCoinRepository(db).UpdateStorageAssignment(coin, &tray.ID, &slotTwo); err != nil {
			t.Fatal(err)
		}
		duplicate, err := coinSvc.DuplicateCoin(coin.ID, 1)
		if err != nil {
			t.Fatal(err)
		}
		if duplicate.StorageLocationID != nil || duplicate.StorageSlot != nil {
			t.Fatalf("duplicate retained tray assignment: %#v", duplicate)
		}
		if _, err := coinSvc.BulkAssignLocation([]uint{coin.ID}, &tray.ID, 1); !errors.Is(err, ErrStorageSlotRequired) {
			t.Fatalf("bulk tray assignment error = %v", err)
		}
		if _, err := coinSvc.BulkAssignLocation([]uint{coin.ID}, &standard.ID, 1); err != nil {
			t.Fatalf("bulk standard assignment: %v", err)
		}
		var persisted models.Coin
		if err := db.First(&persisted, coin.ID).Error; err != nil {
			t.Fatal(err)
		}
		if persisted.StorageLocationID == nil || *persisted.StorageLocationID != standard.ID || persisted.StorageSlot != nil {
			t.Fatalf("bulk standard assignment did not clear slot: %#v", persisted)
		}
	})

	t.Run("intake validates explicit placement while quick capture and deep identification cannot inject it", func(t *testing.T) {
		intakeDB := setupTestDB(t)
		if err := intakeDB.AutoMigrate(&models.CoinIntakeDraft{}); err != nil {
			t.Fatal(err)
		}
		intakeCoinRepo := repository.NewCoinRepository(intakeDB)
		intakeCoinSvc := NewCoinService(intakeCoinRepo, nil).
			WithStorageLocationSupport(repository.NewStorageLocationRepository(intakeDB))
		intakeSvc := NewCoinIntakeService(repository.NewCoinIntakeDraftRepository(intakeDB), intakeCoinRepo, nil, nil).
			WithCoinService(intakeCoinSvc)
		intakeTray := models.StorageLocation{UserID: 1, Name: "Intake Tray", Type: models.StorageLocationTypeTray, Rows: &rows, Columns: &columns}
		if err := intakeDB.Create(&intakeTray).Error; err != nil {
			t.Fatal(err)
		}
		draft := seedCoinIntakeDraft(t, intakeDB, 1, map[string]interface{}{
			"name": "Intake", "category": "Roman", "storageLocationId": intakeTray.ID,
		})
		if _, err := intakeSvc.CommitDraft(1, IntakeCommitRequest{DraftID: draft.ID, Confirm: true}); !errors.Is(err, ErrStorageSlotRequired) {
			t.Fatalf("intake error = %v", err)
		}

		promoteType := reflect.TypeOf(PromoteOverrides{})
		if _, ok := promoteType.FieldByName("StorageLocationID"); ok {
			t.Fatal("Quick Capture overrides unexpectedly expose storageLocationId")
		}
		if _, ok := promoteType.FieldByName("StorageSlot"); ok {
			t.Fatal("Quick Capture overrides unexpectedly expose storageSlot")
		}
		if _, ok := deepProposalCoinFieldAllowlist["storageLocationId"]; ok {
			t.Fatal("deep-identification allowlist unexpectedly exposes storageLocationId")
		}
		if _, ok := deepProposalCoinFieldAllowlist["storageSlot"]; ok {
			t.Fatal("deep-identification allowlist unexpectedly exposes storageSlot")
		}
	})
}

func jsonUint(value uint) string {
	encoded, _ := json.Marshal(value)
	return string(encoded)
}
