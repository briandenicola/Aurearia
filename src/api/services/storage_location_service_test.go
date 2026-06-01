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
