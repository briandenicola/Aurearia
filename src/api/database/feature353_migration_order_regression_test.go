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

// ---------------------------------------------------------------------------------------
// BRUTUS (QA) FOLLOW-UP: production still fails at "DROP TABLE availability_runs" with
// SQLite error 787 (SQLITE_CONSTRAINT_FOREIGNKEY) after hotfix 1df5a99.
//
// WHY THE TESTS ABOVE PASSED INCORRECTLY:
//
// legacyAvailabilityRun / legacyAvailabilityResult (declared in database_test.go) carry NO
// gorm relation/foreignKey tags whatsoever -- no `User User gorm:"foreignKey:UserID"` belongs-to
// and no `Results []AvailabilityResult gorm:"foreignKey:RunID"` has-many. Verified empirically
// below (TestPreFeature353FixtureShapeMatchesRealProductionHistory): AutoMigrate-ing that
// fixture produces availability_runs/availability_results tables with ZERO physical FK
// constraints (PRAGMA foreign_key_list returns no rows for either table). That is not what
// production's pre-353 schema actually looked like.
//
// The REAL pre-353 AvailabilityRun/AvailabilityResult shape (models/availability_check.go as
// of commit 553084c, the last commit before Feature 353 added CycleID) declared:
//
//	User    User                 `gorm:"foreignKey:UserID"`            // belongs-to, no constraint:-
//	Results []AvailabilityResult `gorm:"foreignKey:RunID"`              // has-many, no constraint:-
//
// GORM creates a REAL SQLite FOREIGN KEY constraint for both of these relations by default.
// So production's on-disk availability_runs table has (or had, until users.id 0 legacy rows
// forced enforcement off during those historical inserts) a physical FK on user_id ->
// users(id), and -- critically, and NEVER suppressed by the hotfix -- availability_results
// has a physical FK on run_id -> availability_runs(id) that is still live today (models
// .AvailabilityRun.Results is untouched by 1df5a99; only the User field got `constraint:-`).
//
// When AutoMigrate now needs to rebuild availability_runs (to add the cycle_id column + its
// FK to availability_cycles), GORM/glebarez's SQLite migrator does:
//
//	CREATE TABLE availability_runs__temp (... cycle_id FK to availability_cycles ...)
//	INSERT INTO availability_runs__temp SELECT ... FROM availability_runs
//	DROP TABLE availability_runs        <-- FAILS HERE with PRAGMA foreign_keys=ON
//	ALTER TABLE availability_runs__temp RENAME TO availability_runs
//
// The DROP fails because availability_results.run_id still has a live, physical FK pointing
// at the OLD availability_runs table, and Connect() enables `PRAGMA foreign_keys=ON` before
// AutoMigrate runs (database.go:75) -- reproduced deterministically below, verbatim to the
// production error text, independent of the AvailabilityCycle/AvailabilityRun ordering fix.
//
// Reordering AvailabilityCycle before AvailabilityRun (1df5a99) fixed the FIRST, DIFFERENT
// failure ("no such table: main.availability_cycles"); it did nothing about this SECOND,
// still-live failure, because the fixture used to validate it never had the
// results->runs physical FK in the first place. TestDatabaseGoRegistersAvailabilityCycleBefore
// AvailabilityRun remains a useful, narrowly-scoped guard against the ordering regression it
// was written for -- but it is not, and never was, a guard against this FK-787 failure, and
// must not be read as proof the migration path is safe end-to-end.
// ---------------------------------------------------------------------------------------

// trueLegacyAvailabilityRun mirrors models.AvailabilityRun exactly as it shipped in
// production immediately before Feature 353 (commit 553084c) -- including the belongs-to
// User relation and the has-many Results relation, BOTH without `constraint:-`, so AutoMigrate
// generates the exact physical FK constraints production's on-disk schema actually has.
type trueLegacyAvailabilityRun struct {
	ID            uint        `gorm:"primaryKey"`
	UserID        uint        `gorm:"not null"`
	User          models.User `gorm:"foreignKey:UserID"`
	TriggerType   string      `gorm:"type:varchar(20);not null"`
	TriggerUserID *uint
	Status        string `gorm:"type:varchar(20);not null;default:completed"`
	FailMessage   string `gorm:"type:text"`
	CoinsChecked  int
	Available     int
	Unavailable   int
	Unknown       int
	Errors        int
	DurationMs    int64
	StartedAt     time.Time `gorm:"not null"`
	CompletedAt   *time.Time
	Results       []trueLegacyAvailabilityResult `gorm:"foreignKey:RunID"`
	CreatedAt     time.Time
}

func (trueLegacyAvailabilityRun) TableName() string { return "availability_runs" }

// trueLegacyAvailabilityResult mirrors models.AvailabilityResult verbatim; this shape has
// never changed and its physical run_id -> availability_runs(id) FK is still live in
// production today (1df5a99 never touched this model).
type trueLegacyAvailabilityResult struct {
	ID         uint `gorm:"primaryKey"`
	RunID      uint `gorm:"not null;index"`
	CoinID     uint `gorm:"not null"`
	CoinName   string
	URL        string
	Status     string `gorm:"type:varchar(20);not null"`
	Reason     string `gorm:"type:text"`
	HttpStatus *int
	AgentUsed  bool      `gorm:"default:false"`
	CheckedAt  time.Time `gorm:"not null"`
}

func (trueLegacyAvailabilityResult) TableName() string { return "availability_results" }

// productionModelConstructors is a mechanical, exhaustive registry mapping every model type
// name that can appear in database.go's real `DB.AutoMigrate(...)` call to a constructor for
// that model. TestProductionModelConstructorsCoverRealAutoMigrateList asserts this set exactly
// matches what database.go's live source currently registers, so if a model is ever added to
// (or removed from) the production call without updating this map, that drift itself fails
// loudly -- the whole point of item 5 (fixture drift must not be able to hide a repeat of this
// incident) instead of tests quietly continuing to exercise a stale, hand-picked subset.
var productionModelConstructors = map[string]func() any{
	"User":                          func() any { return &models.User{} },
	"StorageLocation":               func() any { return &models.StorageLocation{} },
	"MintLocation":                  func() any { return &models.MintLocation{} },
	"Coin":                          func() any { return &models.Coin{} },
	"CoinImage":                     func() any { return &models.CoinImage{} },
	"CoinReference":                 func() any { return &models.CoinReference{} },
	"CatalogRegistry":               func() any { return &models.CatalogRegistry{} },
	"AppSetting":                    func() any { return &models.AppSetting{} },
	"ApiKey":                        func() any { return &models.ApiKey{} },
	"RefreshToken":                  func() any { return &models.RefreshToken{} },
	"WebAuthnCredential":            func() any { return &models.WebAuthnCredential{} },
	"SecurityEvent":                 func() any { return &models.SecurityEvent{} },
	"IPRule":                        func() any { return &models.IPRule{} },
	"OIDCProvider":                  func() any { return &models.OIDCProvider{} },
	"ExternalIdentity":              func() any { return &models.ExternalIdentity{} },
	"OIDCAuthState":                 func() any { return &models.OIDCAuthState{} },
	"ValueSnapshot":                 func() any { return &models.ValueSnapshot{} },
	"CoinJournal":                   func() any { return &models.CoinJournal{} },
	"Note":                          func() any { return &models.Note{} },
	"CoinIntakeDraft":               func() any { return &models.CoinIntakeDraft{} },
	"QuickCaptureDraft":             func() any { return &models.QuickCaptureDraft{} },
	"QuickCaptureDraftImage":        func() any { return &models.QuickCaptureDraftImage{} },
	"QuickCaptureDraftReference":    func() any { return &models.QuickCaptureDraftReference{} },
	"DraftLifecycleEvent":           func() any { return &models.DraftLifecycleEvent{} },
	"AgentConversation":             func() any { return &models.AgentConversation{} },
	"CollectionUpdateProposal":      func() any { return &models.CollectionUpdateProposal{} },
	"SetBuilderRun":                 func() any { return &models.SetBuilderRun{} },
	"SetProposal":                   func() any { return &models.SetProposal{} },
	"ProposalSlot":                  func() any { return &models.ProposalSlot{} },
	"Follow":                        func() any { return &models.Follow{} },
	"CoinComment":                   func() any { return &models.CoinComment{} },
	"CoinValueHistory":              func() any { return &models.CoinValueHistory{} },
	"Shipment":                      func() any { return &models.Shipment{} },
	"ShipmentEvent":                 func() any { return &models.ShipmentEvent{} },
	"AuctionLot":                    func() any { return &models.AuctionLot{} },
	"AvailabilityCycle":             func() any { return &models.AvailabilityCycle{} },
	"AvailabilityRun":               func() any { return &models.AvailabilityRun{} },
	"AvailabilityResult":            func() any { return &models.AvailabilityResult{} },
	"WishlistSearchAlert":           func() any { return &models.WishlistSearchAlert{} },
	"AlertRun":                      func() any { return &models.AlertRun{} },
	"AlertCandidate":                func() any { return &models.AlertCandidate{} },
	"CandidateProvenance":           func() any { return &models.CandidateProvenance{} },
	"CandidateReviewAction":         func() any { return &models.CandidateReviewAction{} },
	"Notification":                  func() any { return &models.Notification{} },
	"AIJob":                         func() any { return &models.AIJob{} },
	"Tag":                           func() any { return &models.Tag{} },
	"CoinTag":                       func() any { return &models.CoinTag{} },
	"CoinSet":                       func() any { return &models.CoinSet{} },
	"CoinSetMembership":             func() any { return &models.CoinSetMembership{} },
	"CoinSetTarget":                 func() any { return &models.CoinSetTarget{} },
	"CoinSetValuationSnapshot":      func() any { return &models.CoinSetValuationSnapshot{} },
	"CoinSetMilestoneAlert":         func() any { return &models.CoinSetMilestoneAlert{} },
	"SmartCriteriaTemplate":         func() any { return &models.SmartCriteriaTemplate{} },
	"CoinRecommendation":            func() any { return &models.CoinRecommendation{} },
	"RecommendationFeedback":        func() any { return &models.RecommendationFeedback{} },
	"Showcase":                      func() any { return &models.Showcase{} },
	"ShowcaseCoin":                  func() any { return &models.ShowcaseCoin{} },
	"AuctionEvent":                  func() any { return &models.AuctionEvent{} },
	"PriceAlert":                    func() any { return &models.PriceAlert{} },
	"BidReminder":                   func() any { return &models.BidReminder{} },
	"AuctionAlertRun":               func() any { return &models.AuctionAlertRun{} },
	"ValuationRun":                  func() any { return &models.ValuationRun{} },
	"ValuationResult":               func() any { return &models.ValuationResult{} },
	"AuctionEndingRun":              func() any { return &models.AuctionEndingRun{} },
	"AuctionWatchBidDigestRun":      func() any { return &models.AuctionWatchBidDigestRun{} },
	"FeaturedCoin":                  func() any { return &models.FeaturedCoin{} },
	"CoinOfDayRun":                  func() any { return &models.CoinOfDayRun{} },
	"CollectionHealthSnapshot":      func() any { return &models.CollectionHealthSnapshot{} },
	"CollectionHealthSnapshotRun":   func() any { return &models.CollectionHealthSnapshotRun{} },
	"RomanImperialFigure":           func() any { return &models.RomanImperialFigure{} },
	"RomanImperialFigureHighlight":  func() any { return &models.RomanImperialFigureHighlight{} },
	"DeepIdentificationJob":         func() any { return &models.DeepIdentificationJob{} },
	"DeepIdentificationEvent":       func() any { return &models.DeepIdentificationEvent{} },
	"DeepIdentificationProviderRun": func() any { return &models.DeepIdentificationProviderRun{} },
	"DeepIdentificationArtifact":    func() any { return &models.DeepIdentificationArtifact{} },
}

// readProductionAutoMigrateModelNames parses database.go's REAL, CURRENT source text and
// returns the ordered list of model type names passed to its live `DB.AutoMigrate(...)` call
// -- the actual production migration path/config, not a hand-selected model list.
func readProductionAutoMigrateModelNames(t *testing.T) []string {
	t.Helper()
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
	nameRe := regexp.MustCompile(`&models\.(\w+)\{\}`)
	matches := nameRe.FindAllStringSubmatch(call, -1)
	names := make([]string, 0, len(matches))
	for _, m := range matches {
		names = append(names, m[1])
	}
	if len(names) == 0 {
		t.Fatal("failed to extract any model names from database.go's AutoMigrate call")
	}
	return names
}

// buildProductionAutoMigrateModels builds the exact, live, ordered model instance list that
// database.go's Connect() passes to DB.AutoMigrate -- so the test below exercises the actual
// production migration path/config (all ~75 registered models, in their real order), not a
// hand-selected model list that can hide GORM reconciliation problems triggered by models
// unrelated to AvailabilityRun/Result.
func buildProductionAutoMigrateModels(t *testing.T) []any {
	t.Helper()
	names := readProductionAutoMigrateModelNames(t)
	models := make([]any, 0, len(names))
	for _, name := range names {
		ctor, ok := productionModelConstructors[name]
		if !ok {
			t.Fatalf("productionModelConstructors is missing an entry for %q, which database.go's "+
				"real AutoMigrate call currently registers -- update productionModelConstructors so "+
				"this test keeps exercising the actual production model list instead of silently "+
				"drifting behind it", name)
		}
		models = append(models, ctor())
	}
	return models
}

// TestProductionModelConstructorsCoverRealAutoMigrateList is the drift guard for item 5: it
// asserts productionModelConstructors' key set is EXACTLY the model set database.go's live
// AutoMigrate call currently registers -- no more, no less -- so this fixture cannot silently
// fall behind (missing a newly-added model) or silently over-claim (keeping a stale entry for
// a model no longer registered) without failing loudly.
func TestProductionModelConstructorsCoverRealAutoMigrateList(t *testing.T) {
	names := readProductionAutoMigrateModelNames(t)
	seen := make(map[string]bool, len(names))
	for _, name := range names {
		seen[name] = true
		if _, ok := productionModelConstructors[name]; !ok {
			t.Errorf("database.go registers &models.%s{} in AutoMigrate but productionModelConstructors has no entry for it", name)
		}
	}
	for name := range productionModelConstructors {
		if !seen[name] {
			t.Errorf("productionModelConstructors has a stale entry for %q that database.go's AutoMigrate call no longer registers", name)
		}
	}
}

// seedTrueLegacyAvailabilityFixture builds a REAL on-disk SQLite file shaped exactly like
// production's actual pre-Feature-353 physical schema: availability_runs with a live FK on
// user_id -> users(id) (from the un-suppressed `User User gorm:"foreignKey:UserID"` belongs-to
// relation that shipped from the very first availability-check commit, 8d52d27) and
// availability_results with a live FK on run_id -> availability_runs(id) (from the
// `Results []AvailabilityResult gorm:"foreignKey:RunID"` has-many relation, which 1df5a99 never
// touched and which is still physically present in models.AvailabilityRun today).
//
// Historical legacy admin rows (UserID = 0, a documented, expected pre-Feature-353 condition
// per tasks.md T006/T038) are seeded with `PRAGMA foreign_keys=OFF`, mirroring the only way
// such rows can exist at all once the user_id FK is physically present: they were written
// before/without FK enforcement for that connection (SQLite's `PRAGMA foreign_keys` is a
// per-connection setting that is never persisted in the database file itself, so historical
// inserts under a connection that never set it are exactly how a physically-FK'd column ends
// up holding a value, like 0, that violates the constraint). `PRAGMA foreign_keys=ON` is then
// enabled -- exactly as Connect() does at every real startup (database.go:75) -- before the
// production AutoMigrate list runs, so enforcement is live for the migration itself, matching
// production exactly.
func seedTrueLegacyAvailabilityFixture(t *testing.T, dbPath string) (*gorm.DB, feature353LegacySeed) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open on-disk legacy db: %v", err)
	}
	closeGormDB(t, db)

	// Historical inserts (including UserID=0 admin rows) happened without FK enforcement.
	if err := db.Exec("PRAGMA foreign_keys=OFF").Error; err != nil {
		t.Fatalf("failed to disable foreign_keys pragma for historical seeding: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &trueLegacyAvailabilityRun{}, &trueLegacyAvailabilityResult{}); err != nil {
		t.Fatalf("failed to create true pre-Feature-353 legacy schema: %v", err)
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
		run := trueLegacyAvailabilityRun{
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
			result := trueLegacyAvailabilityResult{
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
	}

	for i := 0; i < 5; i++ {
		run := trueLegacyAvailabilityRun{
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
			result := trueLegacyAvailabilityResult{
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
	}

	db.Model(&trueLegacyAvailabilityRun{}).Count(&seed.totalRuns)
	db.Model(&trueLegacyAvailabilityResult{}).Count(&seed.totalResults)
	if seed.totalRuns != 8 || seed.totalResults != 30 {
		t.Fatalf("fixture setup sanity check failed: runs=%d results=%d", seed.totalRuns, seed.totalResults)
	}

	// Now enable enforcement exactly as Connect() does before AutoMigrate runs.
	if err := db.Exec("PRAGMA foreign_keys=ON").Error; err != nil {
		t.Fatalf("failed to enable foreign_keys pragma: %v", err)
	}

	return db, seed
}

// pragmaForeignKeyList returns the from/to column pairs SQLite reports for `table` via
// PRAGMA foreign_key_list, e.g. "user_id->users.id", so intentional physical constraints can
// be asserted explicitly and fixture drift (accidentally omitting or adding one) cannot hide
// a repeat of this incident (item 5).
func pragmaForeignKeyList(t *testing.T, db *gorm.DB, table string) []string {
	t.Helper()
	rows, err := db.Raw("PRAGMA foreign_key_list(" + table + ")").Rows()
	if err != nil {
		t.Fatalf("PRAGMA foreign_key_list(%s): %v", table, err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var id, seq int
		var refTable, from, to, onUpdate, onDelete, match string
		if err := rows.Scan(&id, &seq, &refTable, &from, &to, &onUpdate, &onDelete, &match); err != nil {
			t.Fatalf("scan PRAGMA foreign_key_list(%s): %v", table, err)
		}
		out = append(out, from+"->"+refTable+"."+to)
	}
	return out
}

// TestPreFeature353FixtureShapeMatchesRealProductionHistory proves the OLD fixture
// (legacyAvailabilityRun/legacyAvailabilityResult in database_test.go, used by
// TestFeature353Migration_ProductionOrderPreservesLegacyDataAndAddsCycleSupport above) has NO
// physical FK constraints at all -- unlike real production -- which is exactly why that test
// passed incorrectly. It also proves trueLegacyAvailabilityRun/trueLegacyAvailabilityResult
// (this file) reproduce the real, intentional physical constraints production actually has.
func TestPreFeature353FixtureShapeMatchesRealProductionHistory(t *testing.T) {
	t.Run("old_hand_picked_fixture_has_no_physical_FKs_unlike_production", func(t *testing.T) {
		dir := t.TempDir()
		db, err := gorm.Open(sqlite.Open(filepath.Join(dir, "old_fixture.db")), &gorm.Config{})
		if err != nil {
			t.Fatalf("open: %v", err)
		}
		closeGormDB(t, db)
		if err := db.AutoMigrate(&models.User{}, &legacyAvailabilityRun{}, &legacyAvailabilityResult{}); err != nil {
			t.Fatalf("automigrate: %v", err)
		}
		if fks := pragmaForeignKeyList(t, db, "availability_runs"); len(fks) != 0 {
			t.Fatalf("expected the OLD fixture's availability_runs to have NO physical FKs (proving it diverges from production), got %v", fks)
		}
		if fks := pragmaForeignKeyList(t, db, "availability_results"); len(fks) != 0 {
			t.Fatalf("expected the OLD fixture's availability_results to have NO physical FKs (proving it diverges from production), got %v", fks)
		}
	})

	t.Run("true_legacy_fixture_has_the_real_intentional_physical_FKs", func(t *testing.T) {
		dir := t.TempDir()
		db, seed := seedTrueLegacyAvailabilityFixture(t, filepath.Join(dir, "true_fixture.db"))
		_ = seed
		runFKs := pragmaForeignKeyList(t, db, "availability_runs")
		resultFKs := pragmaForeignKeyList(t, db, "availability_results")
		if !containsString(runFKs, "user_id->users.id") {
			t.Fatalf("expected availability_runs.user_id -> users.id physical FK (matches production history), got %v", runFKs)
		}
		if !containsString(resultFKs, "run_id->availability_runs.id") {
			t.Fatalf("expected availability_results.run_id -> availability_runs.id physical FK (matches production, still live today), got %v", resultFKs)
		}
	})
}

func containsString(list []string, want string) bool {
	for _, v := range list {
		if v == want {
			return true
		}
	}
	return false
}

// TestFeature353Migration_RealProductionAutoMigrateListStillFailsWithFK787 is the corrected
// regression test: it runs the ACTUAL production migration path/config -- the real, live,
// full ~75-model AutoMigrate list from database.go (not a hand-selected 4-model list) -- against
// an on-disk fixture with the real physical FK constraints production's schema has, with
// `PRAGMA foreign_keys=ON` enabled first exactly as Connect() does.
//
// CURRENT RESULT (documenting the incomplete production fix, per this ticket): this
// deterministically reproduces "constraint failed: FOREIGN KEY constraint failed (787)" at
// `DROP TABLE availability_runs`, because availability_results.run_id's physical FK to
// availability_runs (never suppressed by 1df5a99) blocks GORM's temp-table rebuild of
// availability_runs (needed to add the cycle_id column/FK). Per this ticket's item 6: since
// the production fix is INCOMPLETE, this test intentionally FAILS (t.Fatal) with a BLOCK
// message instead of being weakened to pass -- do not "fix" this test to tolerate the error;
// fix database.go instead (out of scope for this authorized change) and this test will then
// exercise the full success/invariant path below.
func TestFeature353Migration_RealProductionAutoMigrateListStillFailsWithFK787(t *testing.T) {
	dir := t.TempDir()
	db, seed := seedTrueLegacyAvailabilityFixture(t, filepath.Join(dir, "prod_order.db"))

	beforeRunFKs := pragmaForeignKeyList(t, db, "availability_runs")
	beforeResultFKs := pragmaForeignKeyList(t, db, "availability_results")
	if !containsString(beforeRunFKs, "user_id->users.id") {
		t.Fatalf("fixture setup error: expected availability_runs.user_id -> users.id before migration, got %v", beforeRunFKs)
	}
	if !containsString(beforeResultFKs, "run_id->availability_runs.id") {
		t.Fatalf("fixture setup error: expected availability_results.run_id -> availability_runs.id before migration, got %v", beforeResultFKs)
	}

	models_ := buildProductionAutoMigrateModels(t)
	err := db.AutoMigrate(models_...)

	if err != nil {
		if !strings.Contains(err.Error(), "787") || !strings.Contains(err.Error(), "FOREIGN KEY") {
			t.Fatalf("BLOCK: production-ordered AutoMigrate failed with an UNEXPECTED error (not the "+
				"known FK-787 incident) against the true legacy fixture: %v -- investigate before "+
				"assuming this is the same incident", err)
		}
		t.Fatalf("BLOCK: Feature 353 production migration fix (hotfix 1df5a99) is INCOMPLETE. The real "+
			"production AutoMigrate list (%d models, actual database.go order) still fails against a "+
			"fixture with production's real physical FK constraints (availability_results.run_id -> "+
			"availability_runs.id, never suppressed by the hotfix): %v. This is the exact "+
			"'DROP TABLE availability_runs' / FK 787 failure reported in production. Do NOT weaken "+
			"this test to pass -- database.go's AutoMigrate/temp-table-rebuild strategy for "+
			"availability_runs must be fixed (e.g. temporarily disabling FK enforcement around just this "+
			"table's rebuild, or suppressing/rebuilding the results->runs constraint symmetrically) before "+
			"this test -- and production -- can pass.", len(models_), err)
	}

	// If/when the production fix is completed, the invariants below must all hold.
	assertFeature353MigrationInvariants(t, db, seed)

	if !db.Migrator().HasTable("availability_cycles") {
		t.Fatal("expected availability_cycles table after full production AutoMigrate")
	}
	afterRunFKs := pragmaForeignKeyList(t, db, "availability_runs")
	afterResultFKs := pragmaForeignKeyList(t, db, "availability_results")
	if !containsString(afterResultFKs, "run_id->availability_runs.id") {
		t.Fatalf("expected availability_results.run_id -> availability_runs.id to remain a valid physical FK after migration, got %v", afterResultFKs)
	}
	if containsString(afterRunFKs, "user_id->users.id") {
		t.Log("note: availability_runs.user_id -> users.id FK still physically present after migration " +
			"(models.AvailabilityRun.User has constraint:- but that only suppresses FK generation for a " +
			"freshly-created table, not one rebuilt from an existing physical constraint)")
	}
}

// TestFeature353Migration_RepeatedUpgradeIsDeterministic runs the exact same real production
// AutoMigrate list against a fresh, identically-seeded true-legacy fixture 6 times, to prove
// the FK-787 failure (or, once fixed, the success path) is fully deterministic and not a race
// or flaky SQLite temp-table artifact.
func TestFeature353Migration_RepeatedUpgradeIsDeterministic(t *testing.T) {
	const iterations = 6
	var firstErrText string
	var sawSuccess, sawFailure bool

	for i := 0; i < iterations; i++ {
		dir := t.TempDir()
		db, _ := seedTrueLegacyAvailabilityFixture(t, filepath.Join(dir, "repeat.db"))
		models_ := buildProductionAutoMigrateModels(t)
		err := db.AutoMigrate(models_...)
		if db.Migrator().HasTable("availability_runs__temp") {
			t.Fatalf("iteration %d: availability_runs__temp leftover from an interrupted/partial rebuild", i)
		}
		if err == nil {
			sawSuccess = true
			continue
		}
		sawFailure = true
		if firstErrText == "" {
			firstErrText = err.Error()
		} else if err.Error() != firstErrText {
			t.Fatalf("iteration %d: non-deterministic migration error -- first iteration got %q, this iteration got %q", i, firstErrText, err.Error())
		}
	}

	if sawSuccess && sawFailure {
		t.Fatalf("non-deterministic migration outcome across %d iterations: some succeeded, some failed with %q", iterations, firstErrText)
	}
	if sawFailure {
		t.Fatalf("BLOCK: production migration fix is incomplete -- deterministically failed all %d iterations with: %s", iterations, firstErrText)
	}
}

// TestFeature353Migration_FreshInstallSucceedsWithRealProductionList covers a brand-new
// install (no pre-existing tables at all) using the ACTUAL production AutoMigrate list, and
// then re-runs it a second time to prove idempotency across a "restart" -- both required by
// item 4, both exercised through the real migration path/config rather than a hand-picked
// subset.
func TestFeature353Migration_FreshInstallSucceedsWithRealProductionList(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "fresh_full.db")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open fresh on-disk db: %v", err)
	}
	closeGormDB(t, db)
	if err := db.Exec("PRAGMA foreign_keys=ON").Error; err != nil {
		t.Fatalf("failed to enable foreign_keys pragma: %v", err)
	}

	models_ := buildProductionAutoMigrateModels(t)
	if err := db.AutoMigrate(models_...); err != nil {
		t.Fatalf("fresh-install AutoMigrate with the real production model list failed: %v", err)
	}
	if !db.Migrator().HasTable(&models.AvailabilityCycle{}) {
		t.Fatal("expected availability_cycles table to exist after fresh-database migration")
	}
	if !db.Migrator().HasColumn(&models.AvailabilityRun{}, "CycleID") {
		t.Fatal("expected availability_runs.cycle_id column to exist after fresh-database migration")
	}

	// Second restart: must be idempotent.
	if err := db.AutoMigrate(models_...); err != nil {
		t.Fatalf("second (restart) AutoMigrate with the real production model list failed (idempotency broken): %v", err)
	}
	if db.Migrator().HasTable("availability_runs__temp") {
		t.Fatal("availability_runs__temp leftover after second (idempotent) AutoMigrate pass")
	}
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
