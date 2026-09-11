package repository

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var storageTrayTestCounter uint64

func storageTrayTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:storage_tray_%d_%d?mode=memory&cache=shared&_pragma=busy_timeout(5000)", time.Now().UnixNano(), atomic.AddUint64(&storageTrayTestCounter, 1))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent), DisableForeignKeyConstraintWhenMigrating: true})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.AutoMigrate(&models.StorageLocation{}, &models.Coin{}, &models.CoinImage{}); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestStorageLocationRepositoryOccupancyAndAggregateAreOwnerScoped(t *testing.T) {
	db := storageTrayTestDB(t)
	repo := NewStorageLocationRepository(db)
	rows, columns, slot := 3, 3, 6
	ownerTray := models.StorageLocation{UserID: 1, Name: "Owner Tray", Type: "tray", Rows: &rows, Columns: &columns}
	otherTray := models.StorageLocation{UserID: 2, Name: "Other Tray", Type: "tray", Rows: &rows, Columns: &columns}
	if err := db.Create(&ownerTray).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&otherTray).Error; err != nil {
		t.Fatal(err)
	}
	coin := models.Coin{UserID: 1, Name: "Denarius", DiameterMm: floatPtr(19), StorageLocationID: &ownerTray.ID, StorageSlot: &slot}
	if err := db.Create(&coin).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.CoinImage{CoinID: coin.ID, FilePath: "coin.jpg", ImageType: "obverse", IsPrimary: true}).Error; err != nil {
		t.Fatal(err)
	}
	occupancy, err := repo.Occupancy(ownerTray.ID, 1, &coin.ID)
	if err != nil || len(occupancy.OccupiedSlots) != 1 || occupancy.OccupiedSlots[0] != 6 || occupancy.CurrentCoinSlot == nil {
		t.Fatalf("occupancy=%#v err=%v", occupancy, err)
	}
	trays, err := repo.ListTrayAggregates(1)
	if err != nil || len(trays) != 1 || len(trays[0].Coins) != 1 || trays[0].Coins[0].StorageSlot != 6 {
		t.Fatalf("trays=%#v err=%v", trays, err)
	}
	if trays[0].Coins[0].Image == nil || trays[0].Coins[0].Image.FilePath != "coin.jpg" {
		t.Fatalf("minimal image missing: %#v", trays[0].Coins[0])
	}
}

func floatPtr(value float64) *float64 { return &value }

func TestStorageLocationRepositoryNamesAreCaseInsensitivePerOwner(t *testing.T) {
	db := storageTrayTestDB(t)
	repo := NewStorageLocationRepository(db)
	if err := repo.Create(&models.StorageLocation{UserID: 1, Name: " Cabinet A ", Type: "standard"}); err != nil {
		t.Fatal(err)
	}

	exists, err := repo.ExistsByName(1, "cabinet a")
	if err != nil || !exists {
		t.Fatalf("same-owner lookup exists=%v err=%v", exists, err)
	}
	exists, err = repo.ExistsByName(2, "cabinet a")
	if err != nil || exists {
		t.Fatalf("cross-owner lookup exists=%v err=%v", exists, err)
	}
}

func TestStorageLocationRepositoryConcurrentCaseVariantNamesHaveOneWinner(t *testing.T) {
	db := storageTrayTestDB(t)
	if err := db.Exec(`CREATE UNIQUE INDEX idx_storage_locations_user_name_nocase
		ON storage_locations(user_id, name COLLATE NOCASE)`).Error; err != nil {
		t.Fatal(err)
	}
	repo := NewStorageLocationRepository(db)
	start := make(chan struct{})
	results := make(chan error, 2)
	var wg sync.WaitGroup
	for _, name := range []string{"Cabinet A", "cabinet a"} {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			<-start
			results <- repo.Create(&models.StorageLocation{
				UserID: 1, Name: name, Type: models.StorageLocationTypeStandard,
			})
		}(name)
	}
	close(start)
	wg.Wait()
	close(results)

	successes, conflicts := 0, 0
	for err := range results {
		switch {
		case err == nil:
			successes++
		case errors.Is(err, ErrStorageLocationNameConflict):
			conflicts++
		default:
			t.Fatalf("unexpected create error: %v", err)
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("successes=%d conflicts=%d, want one of each", successes, conflicts)
	}

	if err := repo.Create(&models.StorageLocation{
		UserID: 2, Name: "CABINET A", Type: models.StorageLocationTypeStandard,
	}); err != nil {
		t.Fatalf("same name for another owner must remain valid: %v", err)
	}
}
