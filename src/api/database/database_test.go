package database

import (
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// legacyAvailabilityRun mirrors the pre-353 AvailabilityRun shape (no CycleID column).
// Legacy rows include admin-triggered rows with UserID = 0 (a global, non-owner-scoped run
// from before the per-owner "child run" model existed) alongside per-owner scheduled rows.
type legacyAvailabilityRun struct {
	ID            uint   `gorm:"primaryKey"`
	UserID        uint   `gorm:"not null"`
	TriggerType   string `gorm:"type:varchar(20);not null"`
	TriggerUserID *uint
	Status        string `gorm:"type:varchar(20);not null;default:completed"`
	FailMessage   string `gorm:"type:text"`
	CoinsChecked  int
	Available     int
	Unavailable   int
	Unknown       int
	Errors        int
	DurationMs    int64
	StartedAt     time.Time
	CompletedAt   *time.Time
	CreatedAt     time.Time
}

func (legacyAvailabilityRun) TableName() string { return "availability_runs" }

type legacyAvailabilityResult struct {
	ID         uint `gorm:"primaryKey"`
	RunID      uint `gorm:"not null;index"`
	CoinID     uint `gorm:"not null"`
	CoinName   string
	URL        string
	Status     string `gorm:"type:varchar(20);not null"`
	Reason     string `gorm:"type:text"`
	HttpStatus *int
	AgentUsed  bool `gorm:"default:false"`
	CheckedAt  time.Time
}

func (legacyAvailabilityResult) TableName() string { return "availability_results" }

// T036: the 353-wishlist-availability-run-observability migration (new AvailabilityCycle
// table + AvailabilityRun.CycleID nullable column) is purely additive — every pre-existing
// legacy row/result must survive byte-equivalent, gain a NULL CycleID, and no cycle rows may
// ever be synthesized for them. Re-running AutoMigrate a second time must be fully idempotent.
func TestAvailabilityCycleMigration_AdditiveAndPreservesLegacyRows(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}

	// 1. Build the pre-feature schema and seed legacy rows: 3 admin rows (UserID = 0, 5
	// results each) and 5 per-owner scheduled rows (UserID > 0, 3 results each).
	if err := db.AutoMigrate(&models.User{}, &legacyAvailabilityRun{}, &legacyAvailabilityResult{}); err != nil {
		t.Fatalf("failed to create legacy schema: %v", err)
	}

	owner := models.User{Username: "legacy-owner", Email: "legacy-owner@test.com", PasswordHash: "hash"}
	if err := db.Create(&owner).Error; err != nil {
		t.Fatalf("seed owner: %v", err)
	}

	var legacyAdminRuns []legacyAvailabilityRun
	for i := 0; i < 3; i++ {
		run := legacyAvailabilityRun{
			UserID:       0,
			TriggerType:  "manual",
			Status:       models.AvailabilityRunStatusCompleted,
			CoinsChecked: 10,
			Available:    8,
			Unavailable:  2,
			StartedAt:    time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		}
		if err := db.Create(&run).Error; err != nil {
			t.Fatalf("seed legacy admin run: %v", err)
		}
		for j := 0; j < 5; j++ {
			result := legacyAvailabilityResult{
				RunID:    run.ID,
				CoinID:   uint(j + 1),
				CoinName: "Legacy Coin",
				URL:      "https://legacy.example.test/coin",
				Status:   "available",
			}
			if err := db.Create(&result).Error; err != nil {
				t.Fatalf("seed legacy admin result: %v", err)
			}
		}
		legacyAdminRuns = append(legacyAdminRuns, run)
	}

	var legacyScheduledRuns []legacyAvailabilityRun
	for i := 0; i < 5; i++ {
		run := legacyAvailabilityRun{
			UserID:       owner.ID,
			TriggerType:  "scheduled",
			Status:       models.AvailabilityRunStatusCompleted,
			CoinsChecked: 4,
			Available:    4,
			StartedAt:    time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
		}
		if err := db.Create(&run).Error; err != nil {
			t.Fatalf("seed legacy scheduled run: %v", err)
		}
		for j := 0; j < 3; j++ {
			result := legacyAvailabilityResult{
				RunID:    run.ID,
				CoinID:   uint(j + 1),
				CoinName: "Owner Coin",
				URL:      "https://owner.example.test/coin",
				Status:   "available",
			}
			if err := db.Create(&result).Error; err != nil {
				t.Fatalf("seed legacy scheduled result: %v", err)
			}
		}
		legacyScheduledRuns = append(legacyScheduledRuns, run)
	}

	var totalRunsBefore, totalResultsBefore int64
	db.Model(&legacyAvailabilityRun{}).Count(&totalRunsBefore)
	db.Model(&legacyAvailabilityResult{}).Count(&totalResultsBefore)
	if totalRunsBefore != 8 || totalResultsBefore != 30 {
		t.Fatalf("fixture setup sanity check failed: runs=%d results=%d", totalRunsBefore, totalResultsBefore)
	}

	// 2. Apply the real Feature 353 migration: AvailabilityCycle is new, AvailabilityRun
	// gains CycleID. This mirrors exactly the additive AutoMigrate call in database.Connect.
	if err := db.AutoMigrate(&models.User{}, &models.AvailabilityRun{}, &models.AvailabilityResult{}, &models.AvailabilityCycle{}); err != nil {
		t.Fatalf("AvailabilityCycle migration failed: %v", err)
	}

	// 3. No cycle rows may ever be synthesized for pre-existing runs.
	var cycleCount int64
	if err := db.Model(&models.AvailabilityCycle{}).Count(&cycleCount).Error; err != nil {
		t.Fatalf("count cycles: %v", err)
	}
	if cycleCount != 0 {
		t.Fatalf("expected 0 synthesized cycles after additive migration, got %d", cycleCount)
	}

	// 4. Every pre-existing row must be byte-equivalent on every legacy field, and gain a
	// NULL CycleID.
	var migratedRuns []models.AvailabilityRun
	if err := db.Order("id ASC").Find(&migratedRuns).Error; err != nil {
		t.Fatalf("reload migrated runs: %v", err)
	}
	if len(migratedRuns) != 8 {
		t.Fatalf("expected 8 runs to survive migration, got %d", len(migratedRuns))
	}
	for i, run := range migratedRuns {
		if run.CycleID != nil {
			t.Fatalf("run %d: expected CycleID nil after additive migration, got %v", run.ID, *run.CycleID)
		}
		var wantUserID uint
		var wantTrigger string
		var wantChecked, wantAvailable, wantUnavailable int
		if i < 3 {
			wantUserID, wantTrigger, wantChecked, wantAvailable, wantUnavailable = 0, "manual", 10, 8, 2
		} else {
			wantUserID, wantTrigger, wantChecked, wantAvailable, wantUnavailable = owner.ID, "scheduled", 4, 4, 0
		}
		if run.UserID != wantUserID {
			t.Errorf("run %d: expected UserID %d, got %d", run.ID, wantUserID, run.UserID)
		}
		if run.TriggerType != wantTrigger {
			t.Errorf("run %d: expected TriggerType %q untouched, got %q", run.ID, wantTrigger, run.TriggerType)
		}
		if run.Status != models.AvailabilityRunStatusCompleted {
			t.Errorf("run %d: expected status completed untouched, got %q", run.ID, run.Status)
		}
		if run.CoinsChecked != wantChecked || run.Available != wantAvailable || run.Unavailable != wantUnavailable {
			t.Errorf("run %d: counts changed by migration: checked=%d available=%d unavailable=%d",
				run.ID, run.CoinsChecked, run.Available, run.Unavailable)
		}
	}

	var totalRunsAfter, totalResultsAfter int64
	db.Model(&models.AvailabilityRun{}).Count(&totalRunsAfter)
	db.Model(&models.AvailabilityResult{}).Count(&totalResultsAfter)
	if totalRunsAfter != totalRunsBefore {
		t.Fatalf("run count changed by migration: before=%d after=%d", totalRunsBefore, totalRunsAfter)
	}
	if totalResultsAfter != totalResultsBefore {
		t.Fatalf("result count changed by migration: before=%d after=%d", totalResultsBefore, totalResultsAfter)
	}

	// Results themselves must also be byte-equivalent (never cascade-touched by the migration).
	var migratedResults []models.AvailabilityResult
	if err := db.Order("id ASC").Find(&migratedResults).Error; err != nil {
		t.Fatalf("reload migrated results: %v", err)
	}
	for _, res := range migratedResults {
		if res.Status != "available" {
			t.Errorf("result %d: expected status available untouched, got %q", res.ID, res.Status)
		}
	}

	// 5. Idempotency: running AutoMigrate a second time must not change row counts or values.
	if err := db.AutoMigrate(&models.User{}, &models.AvailabilityRun{}, &models.AvailabilityResult{}, &models.AvailabilityCycle{}); err != nil {
		t.Fatalf("second AutoMigrate pass failed: %v", err)
	}
	var totalRunsSecondPass, totalResultsSecondPass, cycleCountSecondPass int64
	db.Model(&models.AvailabilityRun{}).Count(&totalRunsSecondPass)
	db.Model(&models.AvailabilityResult{}).Count(&totalResultsSecondPass)
	db.Model(&models.AvailabilityCycle{}).Count(&cycleCountSecondPass)
	if totalRunsSecondPass != totalRunsBefore {
		t.Fatalf("second AutoMigrate pass changed run count: before=%d after=%d", totalRunsBefore, totalRunsSecondPass)
	}
	if totalResultsSecondPass != totalResultsBefore {
		t.Fatalf("second AutoMigrate pass changed result count: before=%d after=%d", totalResultsBefore, totalResultsSecondPass)
	}
	if cycleCountSecondPass != 0 {
		t.Fatalf("second AutoMigrate pass synthesized cycles: got %d", cycleCountSecondPass)
	}

	_ = legacyAdminRuns
	_ = legacyScheduledRuns
}
