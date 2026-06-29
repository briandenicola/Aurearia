package database

import (
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

type legacyCoinWithoutStorageLocation struct {
	ID        uint            `gorm:"primaryKey"`
	Name      string          `gorm:"not null"`
	Category  models.Category `gorm:"type:varchar(20);not null;default:'Other'"`
	UserID    uint            `gorm:"not null"`
	User      models.User     `gorm:"foreignKey:UserID"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (legacyCoinWithoutStorageLocation) TableName() string { return "coins" }

type legacyCoinImage struct {
	ID        uint                             `gorm:"primaryKey"`
	CoinID    uint                             `gorm:"not null"`
	Coin      legacyCoinWithoutStorageLocation `gorm:"foreignKey:CoinID"`
	FilePath  string                           `gorm:"not null"`
	ImageType models.ImageType                 `gorm:"type:varchar(20);default:'other'"`
	IsPrimary bool                             `gorm:"default:false"`
	CreatedAt time.Time
}

func (legacyCoinImage) TableName() string { return "coin_images" }

func TestAutoMigrateAddsStorageLocationToExistingCoinTableWithReferences(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}

	if err := db.Exec("PRAGMA foreign_keys=ON").Error; err != nil {
		t.Fatalf("failed to enable foreign keys: %v", err)
	}

	if err := db.AutoMigrate(&models.User{}, &legacyCoinWithoutStorageLocation{}, &legacyCoinImage{}); err != nil {
		t.Fatalf("failed to create legacy schema: %v", err)
	}
	if err := db.Exec(`INSERT INTO users (id, username, email, password_hash) VALUES (1, 'cassius', 'cassius@example.test', 'hash')`).Error; err != nil {
		t.Fatalf("failed to seed user: %v", err)
	}
	if err := db.Exec(`INSERT INTO coins (id, name, category, user_id) VALUES (1, 'Legacy Denarius', 'Roman', 1)`).Error; err != nil {
		t.Fatalf("failed to seed coin: %v", err)
	}
	if err := db.Exec(`INSERT INTO coin_images (id, coin_id, file_path, image_type) VALUES (1, 1, 'legacy.jpg', 'obverse')`).Error; err != nil {
		t.Fatalf("failed to seed coin image: %v", err)
	}

	if err := db.AutoMigrate(&models.User{}, &models.StorageLocation{}, &models.Coin{}, &models.CoinImage{}); err != nil {
		t.Fatalf("AutoMigrate failed on legacy coin table with references: %v", err)
	}
	if !db.Migrator().HasTable(&models.StorageLocation{}) {
		t.Fatal("expected storage_locations table to be migrated")
	}
	if !db.Migrator().HasColumn(&models.Coin{}, "StorageLocationID") {
		t.Fatal("expected coins.storage_location_id to be migrated")
	}

	var imageCount int64
	if err := db.Model(&models.CoinImage{}).Where("coin_id = ?", 1).Count(&imageCount).Error; err != nil {
		t.Fatalf("failed to count migrated coin images: %v", err)
	}
	if imageCount != 1 {
		t.Fatalf("expected existing coin image to survive migration, got %d", imageCount)
	}
}

func TestQuickCaptureModelsAutoMigrate(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.Coin{}, &models.QuickCaptureDraft{}, &models.QuickCaptureDraftImage{}, &models.DraftLifecycleEvent{}); err != nil {
		t.Fatalf("quick capture automigrate failed: %v", err)
	}
	for _, table := range []string{"quick_capture_drafts", "quick_capture_draft_images", "draft_lifecycle_events"} {
		if !db.Migrator().HasTable(table) {
			t.Fatalf("expected table %s", table)
		}
	}
}

func TestWishlistSearchAlertModelsAutoMigrate(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.Coin{}, &models.WishlistSearchAlert{}, &models.AlertRun{}, &models.AlertCandidate{}, &models.CandidateProvenance{}, &models.CandidateReviewAction{}); err != nil {
		t.Fatalf("wishlist search alert automigrate failed: %v", err)
	}
	for _, table := range []string{"wishlist_search_alerts", "alert_runs", "alert_candidates", "candidate_provenances", "candidate_review_actions"} {
		if !db.Migrator().HasTable(table) {
			t.Fatalf("expected table %s", table)
		}
	}
	if !db.Migrator().HasColumn(&models.Coin{}, "SourceAlertCandidateID") {
		t.Fatal("expected coins.source_alert_candidate_id to be migrated")
	}
}
