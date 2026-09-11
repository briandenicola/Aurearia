package database

import (
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var feature358DBCounter uint64

func feature358DB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:feature358_%d_%d?mode=memory&cache=shared", time.Now().UnixNano(), atomic.AddUint64(&feature358DBCounter, 1))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent), DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	t.Cleanup(func() { _ = sqlDB.Close() })
	return db
}

func TestFeature358MigrationPreservesLegacyAssignmentsAndIsIdempotent(t *testing.T) {
	db := feature358DB(t)
	createdAt := time.Date(2024, time.March, 4, 5, 6, 7, 0, time.UTC)
	updatedAt := time.Date(2025, time.April, 5, 6, 7, 8, 0, time.UTC)
	if err := db.Exec(`CREATE TABLE storage_locations (
		id integer PRIMARY KEY, user_id integer NOT NULL, name text NOT NULL,
		sort_order integer NOT NULL DEFAULT 0, created_at datetime, updated_at datetime
	)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE TABLE coins (
		id integer PRIMARY KEY, user_id integer NOT NULL, name text,
		storage_location_id integer
	)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO storage_locations(id,user_id,name,sort_order,created_at,updated_at) VALUES
			(11,1,'Safe',2,?,?),(22,2,'Safe',3,?,?);
		INSERT INTO coins(id,user_id,name,storage_location_id) VALUES
			(101,1,'Denarius',11),(102,1,'Aureus',11),(202,2,'Follis',22)`,
		createdAt, updatedAt, createdAt, updatedAt).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.StorageLocation{}, &models.Coin{}); err != nil {
		t.Fatal(err)
	}
	for pass := 0; pass < 2; pass++ {
		if err := migrateStructuredStorage(db); err != nil {
			t.Fatalf("pass %d: %v", pass+1, err)
		}
	}
	var locations []models.StorageLocation
	if err := db.Order("id").Find(&locations).Error; err != nil {
		t.Fatal(err)
	}
	if len(locations) != 2 || locations[0].ID != 11 || locations[0].Type != models.StorageLocationTypeStandard || locations[0].Rows != nil {
		t.Fatalf("legacy locations changed: %#v", locations)
	}
	if !locations[0].CreatedAt.Equal(createdAt) || !locations[0].UpdatedAt.Equal(updatedAt) {
		t.Fatalf("legacy timestamps changed: created=%s updated=%s", locations[0].CreatedAt, locations[0].UpdatedAt)
	}
	var coins []models.Coin
	if err := db.Where("id IN ?", []uint{101, 102}).Order("id").Find(&coins).Error; err != nil {
		t.Fatal(err)
	}
	if len(coins) != 2 {
		t.Fatalf("legacy coins changed: %#v", coins)
	}
	for _, coin := range coins {
		if coin.StorageLocationID == nil || *coin.StorageLocationID != 11 || coin.StorageSlot != nil {
			t.Fatalf("legacy assignment changed: %#v", coin)
		}
	}
	if !db.Migrator().HasIndex(&models.Coin{}, "idx_coins_storage_location_slot_unique") {
		t.Fatal("partial unique slot index missing")
	}
	if !db.Migrator().HasIndex(&models.StorageLocation{}, "idx_storage_locations_user_name_nocase") {
		t.Fatal("case-insensitive owner/name unique index missing")
	}
}

func TestFeature358MigrationRejectsDuplicateSlots(t *testing.T) {
	db := feature358DB(t)
	if err := db.AutoMigrate(&models.StorageLocation{}, &models.Coin{}); err != nil {
		t.Fatal(err)
	}

	rows, columns, slot := 2, 2, 1
	location := models.StorageLocation{UserID: 1, Name: "Tray", Type: models.StorageLocationTypeTray, Rows: &rows, Columns: &columns}
	if err := db.Create(&location).Error; err != nil {
		t.Fatal(err)
	}
	// Seed malformed legacy data before the authoritative index exists.
	if err := db.Exec("DROP INDEX IF EXISTS idx_coins_storage_location_slot_unique").Error; err != nil {
		t.Fatal(err)
	}
	for _, id := range []uint{1, 2} {
		coin := models.Coin{ID: id, UserID: 1, Name: fmt.Sprintf("Coin %d", id), StorageLocationID: &location.ID, StorageSlot: &slot}
		if err := db.Create(&coin).Error; err != nil {
			t.Fatal(err)
		}
	}
	err := migrateStructuredStorage(db)
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "unique") {
		t.Fatalf("expected visible duplicate-index failure, got %v", err)
	}
	var count int64
	if err := db.Model(&models.Coin{}).Count(&count).Error; err != nil || count != 2 {
		t.Fatalf("migration altered malformed rows: count=%d err=%v", count, err)
	}
}

func TestFeature358MigrationRejectsCaseVariantNamesWithoutChangingRows(t *testing.T) {
	db := feature358DB(t)
	if err := db.AutoMigrate(&models.StorageLocation{}, &models.Coin{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("DROP INDEX IF EXISTS idx_storage_locations_user_name_nocase").Error; err != nil {
		t.Fatal(err)
	}
	locations := []models.StorageLocation{
		{ID: 11, UserID: 1, Name: "Cabinet A", Type: models.StorageLocationTypeStandard},
		{ID: 12, UserID: 1, Name: "cabinet a", Type: models.StorageLocationTypeStandard},
	}
	if err := db.Create(&locations).Error; err != nil {
		t.Fatal(err)
	}

	err := migrateStructuredStorage(db)
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "case-insensitive duplicate") {
		t.Fatalf("expected visible duplicate-name migration failure, got %v", err)
	}
	var persisted []models.StorageLocation
	if queryErr := db.Order("id").Find(&persisted).Error; queryErr != nil {
		t.Fatal(queryErr)
	}
	if len(persisted) != 2 || persisted[0].Name != "Cabinet A" || persisted[1].Name != "cabinet a" {
		t.Fatalf("migration changed duplicate rows: %#v", persisted)
	}
}
