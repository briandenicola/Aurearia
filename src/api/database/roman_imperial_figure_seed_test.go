package database

import (
	"testing"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupRomanImperialFigureSeedDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	if err := db.AutoMigrate(&models.AppSetting{}, &models.RomanImperialFigure{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	return db
}

func TestSeedRomanImperialFiguresIdempotent(t *testing.T) {
	db := setupRomanImperialFigureSeedDB(t)

	if err := seedRomanImperialFigures(db); err != nil {
		t.Fatalf("first seed failed: %v", err)
	}
	var firstCount int64
	if err := db.Model(&models.RomanImperialFigure{}).Count(&firstCount).Error; err != nil {
		t.Fatalf("count failed: %v", err)
	}
	if firstCount != 153 {
		t.Fatalf("expected 153 seeded imperial figures, got %d", firstCount)
	}

	if err := seedRomanImperialFigures(db); err != nil {
		t.Fatalf("second seed failed: %v", err)
	}
	var secondCount int64
	if err := db.Model(&models.RomanImperialFigure{}).Count(&secondCount).Error; err != nil {
		t.Fatalf("count failed: %v", err)
	}
	if secondCount != firstCount {
		t.Fatalf("expected idempotent count %d, got %d", firstCount, secondCount)
	}
}

func TestSeedRomanImperialFiguresDoesNotRecreateAfterVersionRecorded(t *testing.T) {
	db := setupRomanImperialFigureSeedDB(t)

	if err := seedRomanImperialFigures(db); err != nil {
		t.Fatalf("seed failed: %v", err)
	}
	if err := db.Where("normalized_name = ?", "augustus").Delete(&models.RomanImperialFigure{}).Error; err != nil {
		t.Fatalf("delete seeded Augustus failed: %v", err)
	}
	if err := seedRomanImperialFigures(db); err != nil {
		t.Fatalf("second seed failed: %v", err)
	}

	var count int64
	if err := db.Model(&models.RomanImperialFigure{}).Where("normalized_name = ?", "augustus").Count(&count).Error; err != nil {
		t.Fatalf("count Augustus failed: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected deleted seeded figure to remain deleted, got count %d", count)
	}
}

func TestSeedRomanImperialFiguresRoleCountsMatchDefaultGoal(t *testing.T) {
	db := setupRomanImperialFigureSeedDB(t)
	if err := seedRomanImperialFigures(db); err != nil {
		t.Fatalf("seed failed: %v", err)
	}

	var emperorCount int64
	if err := db.Model(&models.RomanImperialFigure{}).Where("role = ?", models.ImperialFigureRoleEmperor).Count(&emperorCount).Error; err != nil {
		t.Fatalf("count emperors failed: %v", err)
	}
	if emperorCount != 87 {
		t.Fatalf("expected 87 role=emperor figures (the default completion goal), got %d", emperorCount)
	}
}

func TestSeedRomanImperialFiguresNoDuplicateNormalizedNames(t *testing.T) {
	db := setupRomanImperialFigureSeedDB(t)
	if err := seedRomanImperialFigures(db); err != nil {
		t.Fatalf("seed failed: %v", err)
	}

	var total int64
	if err := db.Model(&models.RomanImperialFigure{}).Count(&total).Error; err != nil {
		t.Fatalf("count total failed: %v", err)
	}
	var distinct int64
	if err := db.Model(&models.RomanImperialFigure{}).Distinct("normalized_name").Count(&distinct).Error; err != nil {
		t.Fatalf("count distinct failed: %v", err)
	}
	if distinct != total {
		t.Fatalf("expected all %d normalized names to be unique, got %d distinct", total, distinct)
	}
}
