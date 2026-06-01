package services

import (
	"testing"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestParseLegacyReference(t *testing.T) {
	svc, _ := setupMigrationTestSvc(t)
	registry := map[string]*models.CatalogRegistry{
		"RIC":      {Catalog: "RIC", VolumeRequired: true},
		"RPC":      {Catalog: "RPC", VolumeRequired: true},
		"SNG":      {Catalog: "SNG", VolumeRequired: true},
		"SEAR":     {Catalog: "SEAR", VolumeRequired: false},
		"CRAWFORD": {Catalog: "CRAWFORD", VolumeRequired: false},
		"SPINK":    {Catalog: "SPINK", VolumeRequired: false},
		"DUPLESSY": {Catalog: "DUPLESSY", VolumeRequired: false},
	}

	tests := []struct {
		name          string
		input         string
		wantCatalog   string
		wantVolume    string
		wantNumber    string
		wantJournal   bool
		wantNil       bool
		wantLogPrefix string
	}{
		{
			name:        "RIC with Roman numeral volume",
			input:       "RIC II 207",
			wantCatalog: "RIC",
			wantVolume:  "II",
			wantNumber:  "207",
			wantJournal: false,
		},
		{
			name:        "RIC with different Roman volume",
			input:       "RIC VII 162",
			wantCatalog: "RIC",
			wantVolume:  "VII",
			wantNumber:  "162",
			wantJournal: false,
		},
		{
			name:        "RIC with numeric volume",
			input:       "RIC 2 207",
			wantCatalog: "RIC",
			wantVolume:  "2",
			wantNumber:  "207",
			wantJournal: false,
		},
		{
			name:        "RIC bare number (no volume) - sentinel volume 0",
			input:       "RIC 207",
			wantCatalog: "RIC",
			wantVolume:  "0",
			wantNumber:  "",
			wantJournal: true,
		},
		{
			name:        "RPC with volume",
			input:       "RPC I 1234",
			wantCatalog: "RPC",
			wantVolume:  "I",
			wantNumber:  "1234",
			wantJournal: false,
		},
		{
			name:        "SNG with volume",
			input:       "SNG Cop 123",
			wantCatalog: "SNG",
			wantVolume:  "Cop",
			wantNumber:  "123",
			wantJournal: false,
		},
		{
			name:        "Sear without volume (alias normalization)",
			input:       "Sear 1625",
			wantCatalog: "SEAR",
			wantVolume:  "",
			wantNumber:  "1625",
			wantJournal: false,
		},
		{
			name:        "SRCV alias to SEAR",
			input:       "SRCV 1625",
			wantCatalog: "SEAR",
			wantVolume:  "",
			wantNumber:  "1625",
			wantJournal: false,
		},
		{
			name:        "Spink alias normalization",
			input:       "Spink 3002",
			wantCatalog: "SPINK",
			wantVolume:  "",
			wantNumber:  "3002",
			wantJournal: false,
		},
		{
			name:        "Duplessy alias normalization",
			input:       "Duplessy 456",
			wantCatalog: "DUPLESSY",
			wantVolume:  "",
			wantNumber:  "456",
			wantJournal: false,
		},
		{
			name:        "Crawford without volume",
			input:       "Crawford 50",
			wantCatalog: "CRAWFORD",
			wantVolume:  "",
			wantNumber:  "50",
			wantJournal: false,
		},
		{
			name:        "Number with letter suffix",
			input:       "RIC II 256a",
			wantCatalog: "RIC",
			wantVolume:  "II",
			wantNumber:  "256a",
			wantJournal: false,
		},
		{
			name:        "Number with qualifier",
			input:       "RIC II cf. 88",
			wantCatalog: "RIC",
			wantVolume:  "II",
			wantNumber:  "cf. 88",
			wantJournal: false,
		},
		{
			name:          "Multi-reference - parse first only",
			input:         "RIC II 207; Cohen 15",
			wantCatalog:   "RIC",
			wantVolume:    "II",
			wantNumber:    "207",
			wantJournal:   false,
			wantLogPrefix: "",
		},
		{
			name:          "Multi-reference with bare RIC - sentinel volume 0",
			input:         "RIC 207; Sear 1625",
			wantCatalog:   "RIC",
			wantVolume:    "0",
			wantNumber:    "",
			wantJournal:   true,
			wantLogPrefix: "Legacy RIC reference imported",
		},
		{
			name:          "Unrecognized catalog",
			input:         "BMCRE 123",
			wantNil:       true,
			wantLogPrefix: "unrecognized catalog",
		},
		{
			name:    "Empty string",
			input:   "",
			wantNil: true,
		},
		{
			name:    "Whitespace only",
			input:   "   ",
			wantNil: true,
		},
		{
			name:          "Catalog only, no number",
			input:         "RIC",
			wantCatalog:   "RIC",
			wantVolume:    "0",
			wantNumber:    "",
			wantJournal:   true,
			wantLogPrefix: "Legacy RIC reference imported",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ref, needsJournal, logMsg := svc.parseLegacyReference(tt.input, registry)

			if tt.wantNil {
				if ref != nil {
					t.Errorf("expected nil reference, got %+v", ref)
				}
				return
			}

			if ref == nil {
				t.Fatalf("expected reference, got nil (logMsg: %s)", logMsg)
			}

			if ref.Catalog != tt.wantCatalog {
				t.Errorf("catalog = %q, want %q", ref.Catalog, tt.wantCatalog)
			}
			if ref.Volume != tt.wantVolume {
				t.Errorf("volume = %q, want %q", ref.Volume, tt.wantVolume)
			}
			if ref.Number != tt.wantNumber {
				t.Errorf("number = %q, want %q", ref.Number, tt.wantNumber)
			}
			if needsJournal != tt.wantJournal {
				t.Errorf("needsJournal = %v, want %v", needsJournal, tt.wantJournal)
			}
			if tt.wantLogPrefix != "" {
				if logMsg == "" {
					t.Errorf("expected logMsg with prefix %q, got empty", tt.wantLogPrefix)
				} else if len(logMsg) < len(tt.wantLogPrefix) || logMsg[:len(tt.wantLogPrefix)] != tt.wantLogPrefix {
					t.Errorf("logMsg = %q, want prefix %q", logMsg, tt.wantLogPrefix)
				}
			}
		})
	}
}

func TestMigrationIdempotency(t *testing.T) {
	svc, db := setupMigrationTestSvc(t)

	if err := db.Exec(`INSERT INTO users (id, username, email, password_hash) VALUES (1, 'test', 'test@example.test', 'hash')`).Error; err != nil {
		t.Fatalf("failed to seed user: %v", err)
	}
	if err := db.Exec(`INSERT INTO coins (id, name, category, user_id, rarity_rating) VALUES (1, 'Test Denarius', 'Roman', 1, 'RIC II 207')`).Error; err != nil {
		t.Fatalf("failed to seed coin: %v", err)
	}

	result1, err := svc.MigrateLegacyReferences(1)
	if err != nil {
		t.Fatalf("first migration failed: %v", err)
	}
	if result1.Succeeded != 1 {
		t.Errorf("expected 1 succeeded after first run, got %d", result1.Succeeded)
	}

	var firstCount int64
	if err := db.Model(&models.CoinReference{}).Count(&firstCount).Error; err != nil {
		t.Fatalf("failed to count references after first run: %v", err)
	}
	if firstCount != 1 {
		t.Errorf("expected 1 reference after first run, got %d", firstCount)
	}

	result2, err := svc.MigrateLegacyReferences(1)
	if err != nil {
		t.Fatalf("second migration failed: %v", err)
	}
	if result2.Skipped != 1 {
		t.Errorf("expected 1 skipped after second run, got %d", result2.Skipped)
	}

	var secondCount int64
	if err := db.Model(&models.CoinReference{}).Count(&secondCount).Error; err != nil {
		t.Fatalf("failed to count references after second run: %v", err)
	}
	if secondCount != firstCount {
		t.Errorf("expected count to remain %d after second run, got %d", firstCount, secondCount)
	}
}

func TestMigrationWithExistingReference(t *testing.T) {
	svc, db := setupMigrationTestSvc(t)

	if err := db.Exec(`INSERT INTO users (id, username, email, password_hash) VALUES (1, 'test', 'test@example.test', 'hash')`).Error; err != nil {
		t.Fatalf("failed to seed user: %v", err)
	}
	if err := db.Exec(`INSERT INTO coins (id, name, category, user_id, rarity_rating) VALUES (1, 'Test Denarius', 'Roman', 1, 'RIC II 207')`).Error; err != nil {
		t.Fatalf("failed to seed coin: %v", err)
	}

	existingRef := models.CoinReference{
		CoinID:    1,
		Catalog:   "RIC",
		Volume:    "II",
		Number:    "207",
		Certainty: "high",
	}
	if err := db.Create(&existingRef).Error; err != nil {
		t.Fatalf("failed to create existing reference: %v", err)
	}

	result, err := svc.MigrateLegacyReferences(1)
	if err != nil {
		t.Fatalf("migration failed: %v", err)
	}
	if result.Skipped != 1 {
		t.Errorf("expected 1 skipped, got %d", result.Skipped)
	}

	var count int64
	if err := db.Model(&models.CoinReference{}).Where("coin_id = ?", 1).Count(&count).Error; err != nil {
		t.Fatalf("failed to count references: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 reference (existing only), got %d", count)
	}

	var ref models.CoinReference
	if err := db.Where("coin_id = ?", 1).First(&ref).Error; err != nil {
		t.Fatalf("failed to fetch reference: %v", err)
	}
	if ref.Certainty != "high" {
		t.Errorf("expected existing reference certainty=high, got %q", ref.Certainty)
	}
}

func TestMigrationWithVolume0Sentinel(t *testing.T) {
	svc, db := setupMigrationTestSvc(t)

	if err := db.Exec(`INSERT INTO users (id, username, email, password_hash) VALUES (1, 'test', 'test@example.test', 'hash')`).Error; err != nil {
		t.Fatalf("failed to seed user: %v", err)
	}
	if err := db.Exec(`INSERT INTO coins (id, name, category, user_id, rarity_rating) VALUES (1, 'Bare RIC', 'Roman', 1, 'RIC 207')`).Error; err != nil {
		t.Fatalf("failed to seed coin: %v", err)
	}

	result, err := svc.MigrateLegacyReferences(1)
	if err != nil {
		t.Fatalf("migration failed: %v", err)
	}
	if result.Succeeded != 1 {
		t.Errorf("expected 1 succeeded, got %d", result.Succeeded)
	}

	var ref models.CoinReference
	if err := db.Where("coin_id = ?", 1).First(&ref).Error; err != nil {
		t.Fatalf("failed to fetch reference: %v", err)
	}
	if ref.Volume != "0" {
		t.Errorf("expected volume=0 sentinel, got %q", ref.Volume)
	}
	if ref.Number != "" {
		t.Errorf("expected empty number for volume=0 sentinel, got %q", ref.Number)
	}

	var journalCount int64
	if err := db.Model(&models.CoinJournal{}).Where("coin_id = ?", 1).Count(&journalCount).Error; err != nil {
		t.Fatalf("failed to count journal entries: %v", err)
	}
	if journalCount < 2 {
		t.Errorf("expected at least 2 journal entries (success + manual review), got %d", journalCount)
	}
}

func TestMigrationUserScoped(t *testing.T) {
	svc, db := setupMigrationTestSvc(t)

	if err := db.Exec(`INSERT INTO users (id, username, email, password_hash) VALUES (1, 'user1', 'user1@test.test', 'hash')`).Error; err != nil {
		t.Fatalf("failed to seed user1: %v", err)
	}
	if err := db.Exec(`INSERT INTO users (id, username, email, password_hash) VALUES (2, 'user2', 'user2@test.test', 'hash')`).Error; err != nil {
		t.Fatalf("failed to seed user2: %v", err)
	}
	if err := db.Exec(`INSERT INTO coins (id, name, category, user_id, rarity_rating) VALUES (1, 'Coin A', 'Roman', 1, 'RIC II 100')`).Error; err != nil {
		t.Fatalf("failed to seed coin for user1: %v", err)
	}
	if err := db.Exec(`INSERT INTO coins (id, name, category, user_id, rarity_rating) VALUES (2, 'Coin B', 'Roman', 2, 'RIC III 200')`).Error; err != nil {
		t.Fatalf("failed to seed coin for user2: %v", err)
	}

	result, err := svc.MigrateLegacyReferences(1)
	if err != nil {
		t.Fatalf("migration for user1 failed: %v", err)
	}
	if result.Succeeded != 1 {
		t.Errorf("expected 1 succeeded for user1, got %d", result.Succeeded)
	}

	var user1Refs int64
	if err := db.Model(&models.CoinReference{}).Where("coin_id = ?", 1).Count(&user1Refs).Error; err != nil {
		t.Fatalf("failed to count user1 references: %v", err)
	}
	if user1Refs != 1 {
		t.Errorf("expected 1 reference for user1, got %d", user1Refs)
	}

	var user2Refs int64
	if err := db.Model(&models.CoinReference{}).Where("coin_id = ?", 2).Count(&user2Refs).Error; err != nil {
		t.Fatalf("failed to count user2 references: %v", err)
	}
	if user2Refs != 0 {
		t.Errorf("expected 0 references for user2 (not migrated), got %d", user2Refs)
	}
}

func setupMigrationTestSvc(t *testing.T) (*ReferenceMigrationService, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	if err := db.Exec("PRAGMA foreign_keys=ON").Error; err != nil {
		t.Fatalf("failed to enable foreign keys: %v", err)
	}
	if err := db.AutoMigrate(
		&models.User{},
		&models.Coin{},
		&models.CoinReference{},
		&models.CoinJournal{},
		&models.CatalogRegistry{},
	); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	seedRegistry := []models.CatalogRegistry{
		{Catalog: "RIC", DisplayName: "Roman Imperial Coinage", VolumeRequired: true},
		{Catalog: "RPC", DisplayName: "Roman Provincial Coinage", VolumeRequired: true},
		{Catalog: "SNG", DisplayName: "Sylloge Nummorum Graecorum", VolumeRequired: true},
		{Catalog: "SEAR", DisplayName: "Sear", VolumeRequired: false},
		{Catalog: "CRAWFORD", DisplayName: "Crawford", VolumeRequired: false},
		{Catalog: "SPINK", DisplayName: "Spink", VolumeRequired: false},
		{Catalog: "DUPLESSY", DisplayName: "Duplessy", VolumeRequired: false},
	}
	if err := db.Create(&seedRegistry).Error; err != nil {
		t.Fatalf("failed to seed registry: %v", err)
	}

	coinRefRepo := repository.NewCoinReferenceRepository(db)
	registryRepo := repository.NewCatalogRegistryRepository(db)
	journalRepo := repository.NewJournalRepository(db)
	svc := NewReferenceMigrationService(db, coinRefRepo, registryRepo, journalRepo)

	return svc, db
}
