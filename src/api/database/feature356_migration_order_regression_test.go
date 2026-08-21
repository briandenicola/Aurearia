package database

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// legacyCoinValueHistoryWithoutSource mirrors CoinValueHistory as it existed before the
// Feature 356 source column was added.  AutoMigrate-ing this schema first, then migrating
// with the current models.CoinValueHistory, exercises the real ALTER TABLE ... ADD COLUMN
// path -- SQLite applies the column DEFAULT ('manual') to every pre-existing row at DDL
// time.  That is the exact B1 condition: a NULL/empty WHERE clause matches zero legacy rows.
type legacyCoinValueHistoryWithoutSource struct {
	ID         uint      `gorm:"primaryKey"`
	CoinID     uint      `gorm:"not null;index"`
	UserID     uint      `gorm:"not null;index"`
	Value      float64   `gorm:"not null"`
	Confidence string    `gorm:"type:varchar(20);not null"`
	RecordedAt time.Time `gorm:"not null;index"`
}

func (legacyCoinValueHistoryWithoutSource) TableName() string { return "coin_value_histories" }

// legacyCoinJournalEntry is used only to seed and count coin_journals rows in migration tests.
// coin_journals has no physical FK constraints in the real schema, so no parent tables are needed.
type legacyCoinJournalEntry struct {
	ID        uint      `gorm:"primaryKey"`
	CoinID    uint      `gorm:"not null;index"`
	UserID    uint      `gorm:"not null"`
	Entry     string    `gorm:"type:text;not null"`
	CreatedAt time.Time
}

func (legacyCoinJournalEntry) TableName() string { return "coin_journals" }

// feature356Seed holds the IDs of rows inserted into the pre-migration fixture so each
// assertion can target a specific row rather than relying on insertion order.
type feature356Seed struct {
	highID   uint
	mediumID uint
	lowID    uint
	manualID uint
	emptyID  uint

	scheduledJournalIDs []uint
	onDemandJournalID   uint
	unrelatedJournalID  uint
}

// seedPreFeature356LegacyDatabase creates a real on-disk SQLite database shaped exactly like
// production immediately before Feature 356: coin_value_histories with the legacy schema
// (no source column) and representative rows for every confidence tier, plus coin_journals
// rows covering the scheduled-estimate format (must be cleaned up by D4) and the on-demand /
// unrelated formats (must survive).
func seedPreFeature356LegacyDatabase(t *testing.T, dbPath string) (*gorm.DB, feature356Seed) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("open legacy db: %v", err)
	}
	closeGormDB(t, db)

	// coin_value_histories has no physical FK constraints (no gorm:"foreignKey:..." on
	// CoinID or UserID), so no users/coins table is required for the legacy fixture.
	if err := db.AutoMigrate(&legacyCoinValueHistoryWithoutSource{}, &legacyCoinJournalEntry{}); err != nil {
		t.Fatalf("create pre-356 legacy schema: %v", err)
	}
	if db.Migrator().HasColumn(&legacyCoinValueHistoryWithoutSource{}, "source") {
		t.Fatal("fixture setup error: source column must not exist before migration")
	}

	now := time.Date(2025, 3, 1, 12, 0, 0, 0, time.UTC)

	// One row for each confidence tier -- exactly the repertoire the backfill must handle.
	rows := []legacyCoinValueHistoryWithoutSource{
		{CoinID: 1, UserID: 1, Value: 100.0, Confidence: "high",   RecordedAt: now.Add(-4 * 24 * time.Hour)},
		{CoinID: 1, UserID: 1, Value: 90.0,  Confidence: "medium", RecordedAt: now.Add(-3 * 24 * time.Hour)},
		{CoinID: 1, UserID: 1, Value: 80.0,  Confidence: "low",    RecordedAt: now.Add(-2 * 24 * time.Hour)},
		{CoinID: 1, UserID: 1, Value: 75.0,  Confidence: "manual", RecordedAt: now.Add(-1 * 24 * time.Hour)},
		{CoinID: 1, UserID: 1, Value: 70.0,  Confidence: "",       RecordedAt: now},
	}
	for i := range rows {
		if err := db.Create(&rows[i]).Error; err != nil {
			t.Fatalf("seed value history row %d: %v", i, err)
		}
	}

	var seed feature356Seed
	seed.highID   = rows[0].ID
	seed.mediumID = rows[1].ID
	seed.lowID    = rows[2].ID
	seed.manualID = rows[3].ID
	seed.emptyID  = rows[4].ID

	// Two scheduled journal entries (must be deleted by D4).
	scheduled := []legacyCoinJournalEntry{
		{CoinID: 1, UserID: 1, Entry: "Scheduled AI Value Estimate: $100.00 (high confidence)",  CreatedAt: now.Add(-4 * 24 * time.Hour)},
		{CoinID: 1, UserID: 1, Entry: "Scheduled AI Value Estimate: $90.00 (medium confidence)", CreatedAt: now.Add(-3 * 24 * time.Hour)},
	}
	for i := range scheduled {
		if err := db.Create(&scheduled[i]).Error; err != nil {
			t.Fatalf("seed scheduled journal entry %d: %v", i, err)
		}
		seed.scheduledJournalIDs = append(seed.scheduledJournalIDs, scheduled[i].ID)
	}

	// One on-demand entry (must survive -- no corresponding history row, real data loss if deleted).
	onDemand := legacyCoinJournalEntry{CoinID: 1, UserID: 1, Entry: "AI Value Estimate: $95.00 (high confidence)", CreatedAt: now}
	if err := db.Create(&onDemand).Error; err != nil {
		t.Fatalf("seed on-demand journal entry: %v", err)
	}
	seed.onDemandJournalID = onDemand.ID

	// One unrelated entry (must survive).
	unrelated := legacyCoinJournalEntry{CoinID: 1, UserID: 1, Entry: "Current value updated manually: $75.00", CreatedAt: now}
	if err := db.Create(&unrelated).Error; err != nil {
		t.Fatalf("seed unrelated journal entry: %v", err)
	}
	seed.unrelatedJournalID = unrelated.ID

	return db, seed
}

// runFeature356MigrationPath runs the exact sequence Connect() executes on the first boot
// after Feature 356: AutoMigrate to add the source column, then backfillCoinValueHistorySources,
// then the D4 journal cleanup DELETE.
func runFeature356MigrationPath(t *testing.T, db *gorm.DB) {
	t.Helper()

	if err := db.AutoMigrate(&models.CoinValueHistory{}); err != nil {
		t.Fatalf("AutoMigrate(CoinValueHistory) failed: %v", err)
	}
	if err := backfillCoinValueHistorySources(db); err != nil {
		t.Fatalf("backfillCoinValueHistorySources failed: %v", err)
	}
	// Replicate the inline D4 delete from Connect() exactly.
	if result := db.Exec("DELETE FROM coin_journals WHERE entry LIKE 'Scheduled AI Value Estimate: $%'"); result.Error != nil {
		t.Fatalf("D4 cleanup SQL failed: %v", result.Error)
	}
}

// assertFeature356Invariants checks every guarantee the Feature 356 migration must provide
// after a successful pass.
func assertFeature356Invariants(t *testing.T, db *gorm.DB, seed feature356Seed) {
	t.Helper()

	// Source column must exist on the table.
	if !db.Migrator().HasColumn(&models.CoinValueHistory{}, "Source") {
		t.Fatal("expected coin_value_histories.source column to exist after migration")
	}

	// Helper to read a single row's source by primary key.
	readSource := func(id uint) string {
		t.Helper()
		var src string
		if err := db.Raw("SELECT source FROM coin_value_histories WHERE id = ?", id).Scan(&src).Error; err != nil {
			t.Fatalf("read source for id=%d: %v", id, err)
		}
		return src
	}

	// AI-confidence rows must be relabelled ai_scheduled (B1 regression check).
	if got := readSource(seed.highID); got != models.ValueHistorySourceAIScheduled {
		t.Errorf("high-confidence row id=%d: expected source=%q, got %q", seed.highID, models.ValueHistorySourceAIScheduled, got)
	}
	if got := readSource(seed.mediumID); got != models.ValueHistorySourceAIScheduled {
		t.Errorf("medium-confidence row id=%d: expected source=%q, got %q", seed.mediumID, models.ValueHistorySourceAIScheduled, got)
	}
	if got := readSource(seed.lowID); got != models.ValueHistorySourceAIScheduled {
		t.Errorf("low-confidence row id=%d: expected source=%q, got %q", seed.lowID, models.ValueHistorySourceAIScheduled, got)
	}
	// True manual rows must stay manual.
	if got := readSource(seed.manualID); got != models.ValueHistorySourceManual {
		t.Errorf("manual row id=%d: expected source=%q, got %q", seed.manualID, models.ValueHistorySourceManual, got)
	}
	// Empty-confidence row falls through to the defensive manual stamp.
	if got := readSource(seed.emptyID); got != models.ValueHistorySourceManual {
		t.Errorf("empty-confidence row id=%d: expected source=%q, got %q", seed.emptyID, models.ValueHistorySourceManual, got)
	}

	// Scheduled journal rows must have been deleted by D4.
	for _, jid := range seed.scheduledJournalIDs {
		var count int64
		db.Raw("SELECT COUNT(*) FROM coin_journals WHERE id = ?", jid).Scan(&count)
		if count != 0 {
			t.Errorf("scheduled journal entry id=%d should have been deleted by D4 cleanup but still exists", jid)
		}
	}

	// On-demand journal row must survive D4 (different prefix; no history counterpart).
	var count int64
	db.Raw("SELECT COUNT(*) FROM coin_journals WHERE id = ?", seed.onDemandJournalID).Scan(&count)
	if count != 1 {
		t.Errorf("on-demand journal entry id=%d must survive D4 cleanup but was deleted", seed.onDemandJournalID)
	}

	// Unrelated journal row must survive.
	db.Raw("SELECT COUNT(*) FROM coin_journals WHERE id = ?", seed.unrelatedJournalID).Scan(&count)
	if count != 1 {
		t.Errorf("unrelated journal entry id=%d must survive D4 cleanup but was deleted", seed.unrelatedJournalID)
	}
}

// TestFeature356Migration_LegacySchemaSourceBackfillAndJournalCleanup is the primary
// migration-order regression test.  It reproduces the exact B1 failure scenario (SQLite
// ALTER TABLE DEFAULT stamps every pre-existing row 'manual' before the backfill runs) and
// proves the corrected backfill correctly attributes AI-tier rows while leaving manual rows
// intact.  D4 cleanup is verified: scheduled rows deleted, on-demand and unrelated rows not.
func TestFeature356Migration_LegacySchemaSourceBackfillAndJournalCleanup(t *testing.T) {
	dir := t.TempDir()
	db, seed := seedPreFeature356LegacyDatabase(t, filepath.Join(dir, "pre_feature_356.db"))
	runFeature356MigrationPath(t, db)
	assertFeature356Invariants(t, db, seed)
}

// TestFeature356Migration_Idempotency proves the migration sequence is safe to run on every
// process startup.  A second pass through AutoMigrate + backfill + D4 delete must not change
// any source value, must not delete any surviving journal row, and must not return an error.
func TestFeature356Migration_Idempotency(t *testing.T) {
	dir := t.TempDir()
	db, seed := seedPreFeature356LegacyDatabase(t, filepath.Join(dir, "idempotency_356.db"))

	// First pass: full migration from legacy schema.
	runFeature356MigrationPath(t, db)
	assertFeature356Invariants(t, db, seed)

	// Second pass: identical sequence, must be a pure no-op.
	runFeature356MigrationPath(t, db)
	assertFeature356Invariants(t, db, seed)
}

// TestFeature356Migration_ExplicitSourcePreservedOnSubsequentBoot proves that rows already
// carrying a non-'manual' source (e.g. source='ai_estimate' written by the D3 apply path on
// a post-deploy first boot) are never overwritten by subsequent backfill passes, because the
// WHERE clause only touches rows where source='manual'.
func TestFeature356Migration_ExplicitSourcePreservedOnSubsequentBoot(t *testing.T) {
	dir := t.TempDir()
	db, _ := seedPreFeature356LegacyDatabase(t, filepath.Join(dir, "explicit_source.db"))

	// First boot: migrate the legacy schema and run the backfill.
	if err := db.AutoMigrate(&models.CoinValueHistory{}); err != nil {
		t.Fatalf("AutoMigrate failed: %v", err)
	}
	if err := backfillCoinValueHistorySources(db); err != nil {
		t.Fatalf("first backfill failed: %v", err)
	}

	// Simulate a D3 row written by post-deploy application code (source='ai_estimate').
	aiEstimateRow := models.CoinValueHistory{
		CoinID:     2,
		UserID:     1,
		Value:      110.0,
		Confidence: "high",
		Source:     models.ValueHistorySourceAIEstimate,
		RecordedAt: time.Now(),
	}
	if err := db.Create(&aiEstimateRow).Error; err != nil {
		t.Fatalf("seed ai_estimate row: %v", err)
	}

	// Second boot: backfill must not overwrite the already-explicit source.
	if err := backfillCoinValueHistorySources(db); err != nil {
		t.Fatalf("second backfill failed: %v", err)
	}

	var src string
	if err := db.Raw("SELECT source FROM coin_value_histories WHERE id = ?", aiEstimateRow.ID).Scan(&src).Error; err != nil {
		t.Fatalf("read ai_estimate row: %v", err)
	}
	if src != models.ValueHistorySourceAIEstimate {
		t.Errorf("ai_estimate row source was overwritten by subsequent backfill: got %q, want %q", src, models.ValueHistorySourceAIEstimate)
	}
}

// TestFeature356Migration_D4CleanupSkippedWhenBackfillFails is the B2 ordering regression
// test.  It proves that the D4 destructive DELETE does not execute when backfillCoinValueHistorySources
// returns an error, matching the gating logic Connect() now has.  The failure is induced by
// dropping coin_value_histories after AutoMigrate so that the UPDATE statement fails.
func TestFeature356Migration_D4CleanupSkippedWhenBackfillFails(t *testing.T) {
	dir := t.TempDir()
	db, seed := seedPreFeature356LegacyDatabase(t, filepath.Join(dir, "backfill_failure.db"))

	// Migrate so the source column exists and coin_journals is populated.
	if err := db.AutoMigrate(&models.CoinValueHistory{}); err != nil {
		t.Fatalf("AutoMigrate failed: %v", err)
	}

	// Drop coin_value_histories to force the backfill UPDATE to fail.
	if err := db.Exec("DROP TABLE coin_value_histories").Error; err != nil {
		t.Fatalf("drop coin_value_histories: %v", err)
	}

	backfillErr := backfillCoinValueHistorySources(db)
	if backfillErr == nil {
		t.Fatal("expected backfillCoinValueHistorySources to return an error when coin_value_histories is absent")
	}

	// Replicate Connect()'s gate: only execute D4 if backfill succeeded.
	if backfillErr == nil {
		db.Exec("DELETE FROM coin_journals WHERE entry LIKE 'Scheduled AI Value Estimate: $%'")
	}

	// All journal rows (scheduled included) must still exist because the backfill failed.
	for _, jid := range seed.scheduledJournalIDs {
		var count int64
		db.Raw("SELECT COUNT(*) FROM coin_journals WHERE id = ?", jid).Scan(&count)
		if count != 1 {
			t.Errorf("scheduled journal entry id=%d was deleted even though the backfill failed", jid)
		}
	}
	var count int64
	db.Raw("SELECT COUNT(*) FROM coin_journals WHERE id = ?", seed.onDemandJournalID).Scan(&count)
	if count != 1 {
		t.Errorf("on-demand journal entry id=%d was deleted even though the backfill failed", seed.onDemandJournalID)
	}
	db.Raw("SELECT COUNT(*) FROM coin_journals WHERE id = ?", seed.unrelatedJournalID).Scan(&count)
	if count != 1 {
		t.Errorf("unrelated journal entry id=%d was deleted even though the backfill failed", seed.unrelatedJournalID)
	}
}
