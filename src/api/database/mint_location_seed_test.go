package database

import (
	"testing"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupMintLocationSeedDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	if err := db.AutoMigrate(&models.AppSetting{}, &models.MintLocation{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	return db
}

func TestSeedMintLocationsIdempotent(t *testing.T) {
	db := setupMintLocationSeedDB(t)

	if err := seedMintLocations(db); err != nil {
		t.Fatalf("first seed failed: %v", err)
	}
	var firstCount int64
	if err := db.Model(&models.MintLocation{}).Count(&firstCount).Error; err != nil {
		t.Fatalf("count failed: %v", err)
	}
	if firstCount != 17 {
		t.Fatalf("expected 17 seeded mint locations, got %d", firstCount)
	}

	if err := seedMintLocations(db); err != nil {
		t.Fatalf("second seed failed: %v", err)
	}
	var secondCount int64
	if err := db.Model(&models.MintLocation{}).Count(&secondCount).Error; err != nil {
		t.Fatalf("count failed: %v", err)
	}
	if secondCount != firstCount {
		t.Fatalf("expected idempotent count %d, got %d", firstCount, secondCount)
	}
}

func TestSeedMintLocationsDoesNotRecreateAfterVersionRecorded(t *testing.T) {
	db := setupMintLocationSeedDB(t)

	if err := seedMintLocations(db); err != nil {
		t.Fatalf("seed failed: %v", err)
	}
	if err := db.Where("normalized_name = ?", models.NormalizeMintLocationName("Rome")).Delete(&models.MintLocation{}).Error; err != nil {
		t.Fatalf("delete seeded Rome failed: %v", err)
	}
	if err := seedMintLocations(db); err != nil {
		t.Fatalf("second seed failed: %v", err)
	}

	var count int64
	if err := db.Model(&models.MintLocation{}).Where("normalized_name = ?", models.NormalizeMintLocationName("Rome")).Count(&count).Error; err != nil {
		t.Fatalf("count Rome failed: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected deleted seeded mint to remain deleted, got count %d", count)
	}
}
