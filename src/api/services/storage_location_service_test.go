package services

import (
	"errors"
	"testing"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
)

func newTestStorageLocationService(dbRepo *repository.StorageLocationRepository) *StorageLocationService {
	return NewStorageLocationService(dbRepo)
}

func TestStorageLocationService_RejectsCaseInsensitiveDuplicate(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewStorageLocationRepository(db)
	svc := newTestStorageLocationService(repo)

	if _, err := svc.Create(1, "Tray A", 0); err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if _, err := svc.Create(1, " tray a ", 1); !errors.Is(err, ErrStorageLocationDuplicate) {
		t.Fatalf("expected duplicate error, got %v", err)
	}
}

func TestStorageLocationService_DeleteBlocksWhenCoinsReferenceLocation(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewStorageLocationRepository(db)
	svc := newTestStorageLocationService(repo)

	location, err := svc.Create(1, "Safe", 0)
	if err != nil {
		t.Fatalf("Create location failed: %v", err)
	}
	coin := models.Coin{Name: "Denarius", UserID: 1, StorageLocationID: &location.ID}
	if err := db.Create(&coin).Error; err != nil {
		t.Fatalf("Create coin failed: %v", err)
	}

	count, err := svc.Delete(location.ID, 1)
	if !errors.Is(err, ErrStorageLocationInUse) {
		t.Fatalf("expected in-use error, got %v", err)
	}
	if count != 1 {
		t.Fatalf("expected count 1, got %d", count)
	}
}

func TestCoinService_RejectsStorageLocationOwnedByAnotherUser(t *testing.T) {
	db := setupTestDB(t)
	storageRepo := repository.NewStorageLocationRepository(db)
	location, err := NewStorageLocationService(storageRepo).Create(2, "Other User Safe", 0)
	if err != nil {
		t.Fatalf("Create location failed: %v", err)
	}
	coinRepo := repository.NewCoinRepository(db)
	svc := NewCoinService(coinRepo, nil).WithStorageLocationSupport(storageRepo)
	coin := &models.Coin{Name: "Follis", UserID: 1, StorageLocationID: &location.ID}
	if err := svc.CreateCoin(coin); !errors.Is(err, ErrStorageLocationNotFound) {
		t.Fatalf("expected storage location not found error, got %v", err)
	}
}

func TestStorageLocationService_TrayLifecycleAndAssignmentRules(t *testing.T) {
	db := setupTestDB(t)
	storageRepo := repository.NewStorageLocationRepository(db)
	locationSvc := NewStorageLocationService(storageRepo)
	rows, columns := 3, 3
	tray, err := locationSvc.CreateTyped(1, StorageLocationWrite{
		Name: " Cabinet A ", Type: models.StorageLocationTypeTray, Rows: &rows, Columns: &columns,
	})
	if err != nil || tray.Name != "Cabinet A" || tray.Capacity != 9 {
		t.Fatalf("create tray=%#v err=%v", tray, err)
	}
	coinSvc := NewCoinService(repository.NewCoinRepository(db), nil).WithStorageLocationSupport(storageRepo)
	slot := 9
	first := &models.Coin{Name: "First", UserID: 1, StorageLocationID: &tray.ID, StorageSlot: &slot}
	if err := coinSvc.CreateCoin(first); err != nil {
		t.Fatalf("assign last slot: %v", err)
	}
	second := &models.Coin{Name: "Second", UserID: 1, StorageLocationID: &tray.ID, StorageSlot: &slot}
	if err := coinSvc.CreateCoin(second); !errors.Is(err, ErrStorageSlotOccupied) {
		t.Fatalf("expected occupied conflict, got %v", err)
	}
	newRows := 2
	if _, err := locationSvc.UpdateTyped(tray.ID, 1, StorageLocationUpdate{Rows: &newRows, RowsSet: true}); !errors.Is(err, ErrTrayOccupied) {
		t.Fatalf("expected occupied resize conflict, got %v", err)
	}
}

func TestCoinService_StandardAndTrayAssignmentValidation(t *testing.T) {
	db := setupTestDB(t)
	storageRepo := repository.NewStorageLocationRepository(db)
	locationSvc := NewStorageLocationService(storageRepo)
	standard, _ := locationSvc.Create(1, "Safe", 0)
	rows, columns := 1, 1
	tray, _ := locationSvc.CreateTyped(1, StorageLocationWrite{Name: "Tray", Type: "tray", Rows: &rows, Columns: &columns})
	slot := 1
	coinSvc := NewCoinService(repository.NewCoinRepository(db), nil).WithStorageLocationSupport(storageRepo)
	if err := coinSvc.CreateCoin(&models.Coin{Name: "Bad standard", UserID: 1, StorageLocationID: &standard.ID, StorageSlot: &slot}); !errors.Is(err, ErrStorageSlotInvalid) {
		t.Fatalf("expected standard slot validation, got %v", err)
	}
	if err := coinSvc.CreateCoin(&models.Coin{Name: "Unslotted tray", UserID: 1, StorageLocationID: &tray.ID}); !errors.Is(err, ErrStorageSlotRequired) {
		t.Fatalf("expected required tray slot, got %v", err)
	}
}
