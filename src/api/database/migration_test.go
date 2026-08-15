package database

import (
	"os"
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

type legacyCoinSet struct {
	ID           uint   `gorm:"primaryKey"`
	UserID       uint   `gorm:"not null"`
	Name         string `gorm:"not null"`
	SetType      string `gorm:"type:varchar(20);not null;default:'open'"`
	CreationMode string `gorm:"type:varchar(20)"`
}

func (legacyCoinSet) TableName() string { return "coin_sets" }

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
	if err := db.AutoMigrate(&models.User{}, &models.Coin{}, &models.QuickCaptureDraft{}, &models.QuickCaptureDraftImage{}, &models.QuickCaptureDraftReference{}, &models.DraftLifecycleEvent{}); err != nil {
		t.Fatalf("quick capture automigrate failed: %v", err)
	}
	for _, table := range []string{"quick_capture_drafts", "quick_capture_draft_images", "quick_capture_draft_references", "draft_lifecycle_events"} {
		if !db.Migrator().HasTable(table) {
			t.Fatalf("expected table %s", table)
		}
	}
	if !db.Migrator().HasIndex(&models.QuickCaptureDraftReference{}, "DraftID") {
		t.Fatal("expected unique draft reference index")
	}
	if !db.Migrator().HasIndex(&models.QuickCaptureDraftReference{}, "UserID") {
		t.Fatal("expected owner index")
	}
}

type preNumistaQuickCaptureDraft struct {
	ID             uint
	UserID         uint
	WorkingTitle   string
	Status         string
	PromotedCoinID *uint
}

func (preNumistaQuickCaptureDraft) TableName() string { return "quick_capture_drafts" }

func TestQuickCaptureSelectedReferenceMigrationIsAdditiveAndRollbackCompatible(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	fixture, err := os.ReadFile("testdata/pre_numista_quick_capture.sql")
	if err != nil {
		t.Fatalf("read pre-feature fixture: %v", err)
	}
	if err := db.Exec(string(fixture)).Error; err != nil {
		t.Fatalf("apply pre-feature fixture: %v", err)
	}

	if err := db.AutoMigrate(&models.QuickCaptureDraftReference{}); err != nil {
		t.Fatalf("add selected-reference table: %v", err)
	}
	if !db.Migrator().HasTable(&models.QuickCaptureDraftReference{}) {
		t.Fatal("selected-reference table was not created")
	}

	var drafts []models.QuickCaptureDraft
	if err := db.Preload("SelectedNumistaReference").Order("id").Find(&drafts).Error; err != nil {
		t.Fatalf("new binary could not read pre-feature drafts: %v", err)
	}
	if len(drafts) != 2 || drafts[0].SelectedNumistaReference != nil || drafts[1].SelectedNumistaReference != nil {
		t.Fatalf("existing drafts should remain readable with no relation: %#v", drafts)
	}

	var oldBinaryDrafts []preNumistaQuickCaptureDraft
	if err := db.Order("id").Find(&oldBinaryDrafts).Error; err != nil {
		t.Fatalf("old binary shape could not read additive schema: %v", err)
	}
	if len(oldBinaryDrafts) != 2 || oldBinaryDrafts[0].Status != "active" || oldBinaryDrafts[1].Status != "promoted" {
		t.Fatalf("rollback compatibility changed draft rows: %#v", oldBinaryDrafts)
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

func TestMigrateCoinSetTypes_NormalizesLegacyValues(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	if err := db.AutoMigrate(&legacyCoinSet{}); err != nil {
		t.Fatalf("failed to migrate legacy coin_sets: %v", err)
	}

	legacy := []legacyCoinSet{
		{UserID: 1, Name: "Open Legacy", SetType: "open"},
		{UserID: 1, Name: "Defined Legacy", SetType: "defined"},
		{UserID: 1, Name: "Dynamic Legacy", SetType: "dynamic"},
	}
	if err := db.Create(&legacy).Error; err != nil {
		t.Fatalf("failed to seed legacy sets: %v", err)
	}

	if err := migrateCoinSetTypes(db); err != nil {
		t.Fatalf("migrateCoinSetTypes failed: %v", err)
	}

	var sets []legacyCoinSet
	if err := db.Order("id ASC").Find(&sets).Error; err != nil {
		t.Fatalf("failed to reload sets: %v", err)
	}
	if got := sets[0].SetType; got != "standard" {
		t.Fatalf("expected first set type standard, got %q", got)
	}
	if got := sets[1].SetType; got != "goal" {
		t.Fatalf("expected second set type goal, got %q", got)
	}
	if got := sets[2].SetType; got != "agentic" {
		t.Fatalf("expected dynamic to migrate to agentic, got %q", got)
	}
	if got := sets[2].CreationMode; got != "dynamic" {
		t.Fatalf("expected dynamic legacy set to have creation_mode dynamic, got %q", got)
	}
}

// TestDeepIdentificationModelsAutoMigrate follows
// TestQuickCaptureModelsAutoMigrate: it asserts the four new
// 344-deep-agentic-coin-identification models migrate additively (no
// existing table/column is altered) and that their expected indexes exist.
func TestDeepIdentificationModelsAutoMigrate(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	// Migrate a representative slice of pre-existing models first, mirroring
	// how database.go's real AutoMigrate call already includes User/Coin
	// before these new tables, to prove additivity.
	if err := db.AutoMigrate(&models.User{}, &models.Coin{}); err != nil {
		t.Fatalf("pre-existing automigrate failed: %v", err)
	}
	if err := db.AutoMigrate(
		&models.DeepIdentificationJob{},
		&models.DeepIdentificationEvent{},
		&models.DeepIdentificationProviderRun{},
		&models.DeepIdentificationArtifact{},
	); err != nil {
		t.Fatalf("deep identification automigrate failed: %v", err)
	}

	for _, table := range []string{
		"deep_identification_jobs",
		"deep_identification_events",
		"deep_identification_provider_runs",
		"deep_identification_artifacts",
	} {
		if !db.Migrator().HasTable(table) {
			t.Fatalf("expected table %s", table)
		}
	}

	if !db.Migrator().HasTable(&models.User{}) || !db.Migrator().HasTable(&models.Coin{}) {
		t.Fatal("expected pre-existing users/coins tables to remain present")
	}

	if !db.Migrator().HasIndex(&models.DeepIdentificationEvent{}, "uix_deep_events_job_seq") {
		t.Fatal("expected unique job/seq index on deep_identification_events")
	}
	if !db.Migrator().HasIndex(&models.DeepIdentificationProviderRun{}, "uix_deep_provider_run_job_provider") {
		t.Fatal("expected unique job/provider index on deep_identification_provider_runs")
	}
	if !db.Migrator().HasIndex(&models.DeepIdentificationJob{}, "idx_deep_jobs_user_status_created") {
		t.Fatal("expected user/status/created index on deep_identification_jobs")
	}
	if !db.Migrator().HasIndex(&models.DeepIdentificationJob{}, "idx_deep_jobs_status_heartbeat") {
		t.Fatal("expected status/heartbeat index on deep_identification_jobs")
	}
	if !db.Migrator().HasIndex(&models.DeepIdentificationJob{}, "idx_deep_jobs_expires") {
		t.Fatal("expected expires_at index on deep_identification_jobs")
	}

	if err := db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS uix_deep_artifact_job_role ON deep_identification_artifacts(job_id, role) WHERE role <> 'hint'`).Error; err != nil {
		t.Fatalf("expected partial unique artifact role index to be creatable: %v", err)
	}
}
