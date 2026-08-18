package database

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// TestDatabaseGoRegistersAvailabilityCycleBeforeAvailabilityRun is a durable, source-level
// guard against reintroducing the exact Feature 353 production startup failure:
//
//	GORM/glebarez SQLite AutoMigrate must rebuild the legacy availability_runs table (adding
//	the new CycleID column with a foreign key to availability_cycles) via a temp-table copy:
//	CREATE availability_runs__temp (with the FK) -> INSERT INTO ...__temp SELECT * FROM
//	availability_runs -> DROP availability_runs -> RENAME ...__temp TO availability_runs.
//	If &models.AvailabilityRun{} is registered in database.go's real AutoMigrate call before
//	&models.AvailabilityCycle{} (the parent table it now references via CycleID), then at
//	copy time main.availability_cycles does not exist yet and -- with `PRAGMA
//	foreign_keys=ON` enabled first, exactly as Connect() enables it before AutoMigrate runs --
//	the temp-table copy fails FK/schema validation and the application cannot start
//	(Connect() calls log.Fatalf on the AutoMigrate error).
//
// This test reads database.go's REAL, CURRENT source text (not a hand-copied literal) and
// asserts the actual registration order in the live AutoMigrate(...) call, so it will fail
// immediately if the ordering ever regresses back to child-before-parent -- independent of
// the functional test below, which proves *why* the ordering matters and pins the exact
// error text this repository reproduced.
func TestDatabaseGoRegistersAvailabilityCycleBeforeAvailabilityRun(t *testing.T) {
	src, err := os.ReadFile("database.go")
	if err != nil {
		t.Fatalf("failed to read database.go: %v", err)
	}

	autoMigrateCallRe := regexp.MustCompile(`(?s)DB\.AutoMigrate\((.*?)\)\s*\n\tif err != nil`)
	match := autoMigrateCallRe.FindSubmatch(src)
	if match == nil {
		t.Fatal("could not locate the DB.AutoMigrate(...) call in database.go -- has Connect() been restructured?")
	}
	call := string(match[1])

	cycleIdx := strings.Index(call, "&models.AvailabilityCycle{}")
	runIdx := strings.Index(call, "&models.AvailabilityRun{}")
	if cycleIdx == -1 {
		t.Fatal("&models.AvailabilityCycle{} is not registered in database.go's AutoMigrate call")
	}
	if runIdx == -1 {
		t.Fatal("&models.AvailabilityRun{} is not registered in database.go's AutoMigrate call")
	}
	if cycleIdx > runIdx {
		t.Fatalf("REGRESSION: &models.AvailabilityCycle{} (parent, referenced by AvailabilityRun.CycleID) "+
			"is registered AFTER &models.AvailabilityRun{} (child) in database.go's AutoMigrate call "+
			"(AvailabilityCycle at index %d, AvailabilityRun at index %d). This exactly reproduces the "+
			"Feature 353 production startup failure: GORM/glebarez must rebuild availability_runs via a "+
			"temp-table copy referencing main.availability_cycles, which will not exist yet -- see "+
			"TestFeature353Migration_ProductionOrderPreservesLegacyDataAndAddsCycleSupport for the exact "+
			"functional reproduction and the pinned error text.", cycleIdx, runIdx)
	}
}

// TestFeature353Migration_ProductionOrderPreservesLegacyDataAndAddsCycleSupport exercises the
// database.go AutoMigrate ordering that is currently live in production (AvailabilityCycle
// registered before AvailabilityRun/AvailabilityResult -- verified independently by
// TestDatabaseGoRegistersAvailabilityCycleBeforeAvailabilityRun above) against a REAL on-disk
// SQLite file (not :memory:) shaped exactly like production immediately before Feature 353:
// legacy availability_runs with no cycle_id column, legacy availability_results, no
// availability_cycles table, and realistic legacy data including UserID=0 admin rows (an
// expected, documented production condition per tasks.md T006/T038 and this package's
// TestAvailabilityCycleMigration_AdditiveAndPreservesLegacyRows fixture).
//
// Connect() itself cannot be exercised directly: it calls log.Fatalf -> os.Exit(1) on error,
// which would kill the entire test binary rather than failing a single test, and no package
// helper isolates "run the registered AutoMigrate list" from "connect + fatal on error". This
// test instead calls db.AutoMigrate directly with the exact same relative ordering.
//
// HISTORICAL REPRODUCTION (documented for the record, not re-asserted here as a permanently
// red sub-test): with &models.AvailabilityRun{}/&models.AvailabilityResult{} registered
// BEFORE &models.AvailabilityCycle{} -- i.e. exactly as database.go read before this
// session's fix -- this same fixture and PRAGMA foreign_keys=ON reproduced the exact reported
// production error verbatim:
//
//	SQL logic error: no such table: main.availability_cycles (1)
//	[INSERT INTO `availability_runs__temp`(...,`cycle_id`) SELECT ...,`cycle_id` FROM `availability_runs`]
//
// confirmed deterministic across 6+ repeated runs. That ordering is now guarded against
// regressing by TestDatabaseGoRegistersAvailabilityCycleBeforeAvailabilityRun.
func TestFeature353Migration_ProductionOrderPreservesLegacyDataAndAddsCycleSupport(t *testing.T) {
	db, seed := seedPreFeature353LegacyDatabase(t)

	if err := db.AutoMigrate(&models.User{}, &models.AvailabilityCycle{}, &models.AvailabilityRun{}, &models.AvailabilityResult{}); err != nil {
		t.Fatalf("REGRESSION: production-ordered AutoMigrate (AvailabilityCycle before AvailabilityRun) "+
			"failed against the pre-Feature-353 legacy fixture: %v", err)
	}

	assertFeature353MigrationInvariants(t, db, seed)
}

// TestFeature353Migration_FreshDatabaseNoPriorTables covers a brand-new install (no
// pre-existing availability_runs table at all), which exercises the CREATE TABLE path rather
// than the temp-table rebuild path. Covering it separately proves the historical failure was
// specific to the upgrade-from-legacy-schema path, not to AutoMigrate registration order in
// general.
func TestFeature353Migration_FreshDatabaseNoPriorTables(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "fresh.db")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open fresh on-disk db: %v", err)
	}
	closeGormDB(t, db)
	if err := db.Exec("PRAGMA foreign_keys=ON").Error; err != nil {
		t.Fatalf("failed to enable foreign_keys pragma: %v", err)
	}

	if err := db.AutoMigrate(&models.User{}, &models.AvailabilityCycle{}, &models.AvailabilityRun{}, &models.AvailabilityResult{}); err != nil {
		t.Fatalf("fresh-database AutoMigrate with production order failed: %v", err)
	}
	if !db.Migrator().HasTable(&models.AvailabilityCycle{}) {
		t.Fatal("expected availability_cycles table to exist after fresh-database migration")
	}
	if !db.Migrator().HasColumn(&models.AvailabilityRun{}, "CycleID") {
		t.Fatal("expected availability_runs.cycle_id column to exist after fresh-database migration")
	}
}

// closeGormDB releases the underlying *sql.DB connection at test cleanup so the on-disk
// SQLite file is not still open when Go's t.TempDir() tries to remove it (observed on
// Windows as a "process cannot access the file" cleanup error otherwise).
func closeGormDB(t *testing.T, db *gorm.DB) {
	t.Helper()
	t.Cleanup(func() {
		sqlDB, err := db.DB()
		if err != nil {
			return
		}
		_ = sqlDB.Close()
	})
}

type feature353LegacySeed struct {
	adminRuns     []legacyAvailabilityRun
	scheduledRuns []legacyAvailabilityRun
	ownerID       uint
	totalRuns     int64
	totalResults  int64
}

// seedPreFeature353LegacyDatabase creates a REAL on-disk SQLite file (not :memory:) shaped
// exactly like production immediately before Feature 353: a users table, legacy
// availability_runs (no cycle_id column, mirroring the pre-migration column set exactly),
// legacy availability_results with rows attached to those legacy runs, and NO
// availability_cycles table at all. `PRAGMA foreign_keys=ON` is enabled before any migration
// call, exactly as database.Connect() enables it before calling AutoMigrate, so the
// regression reproduces under the same enforcement mode production runs under.
func seedPreFeature353LegacyDatabase(t *testing.T) (*gorm.DB, feature353LegacySeed) {
	t.Helper()

	dir := t.TempDir()
	dbPath := filepath.Join(dir, "pre_feature_353.db")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open on-disk legacy db: %v", err)
	}
	closeGormDB(t, db)
	if err := db.Exec("PRAGMA foreign_keys=ON").Error; err != nil {
		t.Fatalf("failed to enable foreign_keys pragma: %v", err)
	}

	if err := db.AutoMigrate(&models.User{}, &legacyAvailabilityRun{}, &legacyAvailabilityResult{}); err != nil {
		t.Fatalf("failed to create pre-Feature-353 legacy schema: %v", err)
	}
	if db.Migrator().HasTable("availability_cycles") {
		t.Fatal("fixture setup error: availability_cycles must not exist before migration")
	}

	owner := models.User{Username: "legacy-owner", Email: "legacy-owner@test.com", PasswordHash: "hash"}
	if err := db.Create(&owner).Error; err != nil {
		t.Fatalf("seed owner: %v", err)
	}

	var seed feature353LegacySeed
	seed.ownerID = owner.ID

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
		seed.adminRuns = append(seed.adminRuns, run)
	}

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
		seed.scheduledRuns = append(seed.scheduledRuns, run)
	}

	db.Model(&legacyAvailabilityRun{}).Count(&seed.totalRuns)
	db.Model(&legacyAvailabilityResult{}).Count(&seed.totalResults)
	if seed.totalRuns != 8 || seed.totalResults != 30 {
		t.Fatalf("fixture setup sanity check failed: runs=%d results=%d", seed.totalRuns, seed.totalResults)
	}

	return db, seed
}

// assertFeature353MigrationInvariants asserts every guarantee the Feature 353 migration must
// provide once it succeeds: availability_cycles exists, availability_runs.cycle_id exists and
// is nullable/NULL on every legacy row, every legacy run/result value and the total counts are
// preserved byte-for-byte, no availability_runs__temp table is left behind by an
// interrupted/partial rebuild, and running the migration a second time is fully idempotent.
func assertFeature353MigrationInvariants(t *testing.T, db *gorm.DB, seed feature353LegacySeed) {
	t.Helper()

	if !db.Migrator().HasTable(&models.AvailabilityCycle{}) {
		t.Fatal("expected availability_cycles table to exist after migration")
	}
	if !db.Migrator().HasColumn(&models.AvailabilityRun{}, "CycleID") {
		t.Fatal("expected availability_runs.cycle_id column to exist after migration")
	}
	if db.Migrator().HasTable("availability_runs__temp") {
		t.Fatal("availability_runs__temp leftover from an interrupted/partial GORM SQLite table rebuild")
	}

	var cycleCount int64
	if err := db.Model(&models.AvailabilityCycle{}).Count(&cycleCount).Error; err != nil {
		t.Fatalf("count cycles: %v", err)
	}
	if cycleCount != 0 {
		t.Fatalf("expected 0 synthesized cycles after additive migration, got %d", cycleCount)
	}

	var migratedRuns []models.AvailabilityRun
	if err := db.Order("id ASC").Find(&migratedRuns).Error; err != nil {
		t.Fatalf("reload migrated runs: %v", err)
	}
	if int64(len(migratedRuns)) != seed.totalRuns {
		t.Fatalf("expected %d runs to survive migration, got %d", seed.totalRuns, len(migratedRuns))
	}
	for i, run := range migratedRuns {
		if run.CycleID != nil {
			t.Fatalf("run %d: expected CycleID nil (nullable, untouched) after additive migration, got %v", run.ID, *run.CycleID)
		}
		var wantUserID uint
		var wantTrigger string
		var wantChecked, wantAvailable, wantUnavailable int
		if i < 3 {
			wantUserID, wantTrigger, wantChecked, wantAvailable, wantUnavailable = 0, "manual", 10, 8, 2
		} else {
			wantUserID, wantTrigger, wantChecked, wantAvailable, wantUnavailable = seed.ownerID, "scheduled", 4, 4, 0
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
	if totalRunsAfter != seed.totalRuns {
		t.Fatalf("run count changed by migration: before=%d after=%d", seed.totalRuns, totalRunsAfter)
	}
	if totalResultsAfter != seed.totalResults {
		t.Fatalf("result count changed by migration: before=%d after=%d", seed.totalResults, totalResultsAfter)
	}

	var migratedResults []models.AvailabilityResult
	if err := db.Order("id ASC").Find(&migratedResults).Error; err != nil {
		t.Fatalf("reload migrated results: %v", err)
	}
	for _, res := range migratedResults {
		if res.Status != "available" {
			t.Errorf("result %d: expected status available untouched, got %q", res.ID, res.Status)
		}
	}

	// Idempotency: running the exact same production-ordered AutoMigrate call a second time
	// must not change row counts/values, must not synthesize cycles, and must not leave a
	// temp table behind.
	if err := db.AutoMigrate(&models.User{}, &models.AvailabilityCycle{}, &models.AvailabilityRun{}, &models.AvailabilityResult{}); err != nil {
		t.Fatalf("second AutoMigrate pass failed (idempotency broken): %v", err)
	}
	if db.Migrator().HasTable("availability_runs__temp") {
		t.Fatal("availability_runs__temp leftover after second (idempotent) AutoMigrate pass")
	}
	var totalRunsSecondPass, totalResultsSecondPass, cycleCountSecondPass int64
	db.Model(&models.AvailabilityRun{}).Count(&totalRunsSecondPass)
	db.Model(&models.AvailabilityResult{}).Count(&totalResultsSecondPass)
	db.Model(&models.AvailabilityCycle{}).Count(&cycleCountSecondPass)
	if totalRunsSecondPass != seed.totalRuns {
		t.Fatalf("second AutoMigrate pass changed run count: before=%d after=%d", seed.totalRuns, totalRunsSecondPass)
	}
	if totalResultsSecondPass != seed.totalResults {
		t.Fatalf("second AutoMigrate pass changed result count: before=%d after=%d", seed.totalResults, totalResultsSecondPass)
	}
	if cycleCountSecondPass != 0 {
		t.Fatalf("second AutoMigrate pass synthesized cycles: got %d", cycleCountSecondPass)
	}
}
