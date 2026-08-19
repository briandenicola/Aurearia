package repository

import (
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func newDeepIdentificationTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:deep_identification_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	if err := db.AutoMigrate(
		&models.User{}, &models.Coin{},
		&models.DeepIdentificationJob{}, &models.DeepIdentificationEvent{},
		&models.DeepIdentificationProviderRun{}, &models.DeepIdentificationArtifact{},
	); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("failed to get sql.DB: %v", err)
	}
	// SQLite writers serialize regardless; keep pool small but >1 so
	// concurrent goroutines can genuinely race through the repository
	// methods (each of which uses its own short transaction).
	sqlDB.SetMaxOpenConns(4)
	return db
}

func createDeepTestUser(t *testing.T, db *gorm.DB, username string) models.User {
	t.Helper()
	user := models.User{Username: username, Email: username + "@example.com", PasswordHash: "x"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("failed to create user %s: %v", username, err)
	}
	return user
}

func timePtr(value time.Time) *time.Time {
	return &value
}

func TestDeepIdentificationRepository_OwnerScoping(t *testing.T) {
	db := newDeepIdentificationTestDB(t)
	repo := NewDeepIdentificationRepository(db)

	owner := createDeepTestUser(t, db, "owner")
	other := createDeepTestUser(t, db, "other")

	job := &models.DeepIdentificationJob{UserID: owner.ID, Source: models.DeepJobSourceIntake, InputFingerprint: "fp-owner-scope", ExpiresAt: time.Now().Add(90 * 24 * time.Hour)}
	created, reused, err := repo.CreateJob(job)
	if err != nil || reused {
		t.Fatalf("CreateJob() = %v, reused=%v, err=%v", created, reused, err)
	}

	if _, err := repo.GetJob(created.ID, other.ID); err == nil {
		t.Fatal("expected cross-user GetJob to return not-found error")
	}
	if _, err := repo.GetJob(created.ID, owner.ID); err != nil {
		t.Fatalf("expected owner GetJob to succeed: %v", err)
	}
	if err := repo.RequestCancel(created.ID, other.ID); err == nil {
		t.Fatal("expected cross-user RequestCancel to fail")
	}
}

func TestDeepIdentificationRepository_RecordRouterSelectionIsOwnerScoped(t *testing.T) {
	db := newDeepIdentificationTestDB(t)
	repo := NewDeepIdentificationRepository(db)
	owner := createDeepTestUser(t, db, "router-owner")
	other := createDeepTestUser(t, db, "router-other")
	job := &models.DeepIdentificationJob{
		UserID: owner.ID, Source: models.DeepJobSourceIntake,
		InputFingerprint: "fp-router-selection", ExpiresAt: time.Now().Add(90 * 24 * time.Hour),
	}
	created, _, err := repo.CreateJob(job)
	if err != nil {
		t.Fatal(err)
	}

	if err := repo.RecordRouterSelection(created.ID, other.ID, []string{"numista"}, "wrong owner"); err != nil {
		t.Fatal(err)
	}
	unchanged, err := repo.GetJob(created.ID, owner.ID)
	if err != nil {
		t.Fatal(err)
	}
	if unchanged.SelectedProviders != "" || unchanged.RouterRationale != "" {
		t.Fatal("cross-owner router update must not modify the job")
	}

	if err := repo.RecordRouterSelection(created.ID, owner.ID, []string{"nomisma", "numista"}, "image evidence"); err != nil {
		t.Fatal(err)
	}
	updated, err := repo.GetJob(created.ID, owner.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.SelectedProviders != "nomisma,numista" || updated.RouterRationale != "image evidence" {
		t.Fatalf("unexpected router selection persistence: %#v", updated)
	}
}

func TestDeepIdentificationRepository_ObservabilityAggregatesOnlyOperationalData(t *testing.T) {
	db := newDeepIdentificationTestDB(t)
	repo := NewDeepIdentificationRepository(db)
	user := createDeepTestUser(t, db, "observability-owner")
	base := time.Now().Add(-time.Hour)

	statuses := []models.DeepJobStatus{
		models.DeepJobStatusCompleted,
		models.DeepJobStatusPartial,
		models.DeepJobStatusFailed,
		models.DeepJobStatusCancelled,
	}

	for i, status := range statuses {
		started := base.Add(time.Duration(i) * time.Second)
		completed := started.Add(time.Duration(i+1) * 100 * time.Millisecond)
		job := models.DeepIdentificationJob{
			UserID: user.ID, Source: models.DeepJobSourceIntake,
			InputFingerprint: fmt.Sprintf("observability-terminal-%d", i),
			Status:           status, StartedAt: timePtr(started), CompletedAt: timePtr(completed),
			Notes: "must not be selected", ReportJSON: `{"private":"report"}`,
			ExpiresAt: time.Now().Add(time.Hour), ActiveKey: fmt.Sprintf("%d", i+1),
		}
		if err := db.Create(&job).Error; err != nil {
			t.Fatalf("seed terminal job %d: %v", i, err)
		}
	}
	queued := models.DeepIdentificationJob{
		UserID: user.ID, Source: models.DeepJobSourceIntake,
		InputFingerprint: "observability-queued", Status: models.DeepJobStatusQueued,
		ExpiresAt: time.Now().Add(time.Hour),
	}
	if err := db.Create(&queued).Error; err != nil {
		t.Fatalf("seed queued job: %v", err)
	}

	started := base
	completed := base.Add(250 * time.Millisecond)
	if err := repo.RecordProviderResult(
		1, user.ID, models.DeepProviderNumista, models.DeepProviderRunContributed,
		true, 0.9, 2, 250, "upstream", started, completed,
	); err != nil {
		t.Fatalf("record provider result: %v", err)
	}
	if err := db.Model(&models.DeepIdentificationProviderRun{}).
		Where("job_id = ? AND provider = ?", 1, models.DeepProviderNumista).
		Update("claims_json", `{"private":"claim"}`).Error; err != nil {
		t.Fatalf("seed prohibited legacy claims: %v", err)
	}
	if err := repo.RecordProviderResult(
		1, user.ID, models.DeepProviderNumista, models.DeepProviderRunContributed,
		true, 0.9, 2, 250, "", started, completed,
	); err != nil {
		t.Fatalf("rewrite provider result: %v", err)
	}

	var run models.DeepIdentificationProviderRun
	if err := db.Where("job_id = ? AND provider = ?", 1, models.DeepProviderNumista).First(&run).Error; err != nil {
		t.Fatalf("load provider run: %v", err)
	}
	if run.ClaimsJSON != "" {
		t.Fatalf("provider observability row retained claims: %q", run.ClaimsJSON)
	}

	metrics, err := repo.GetObservabilityMetrics()
	if err != nil {
		t.Fatalf("GetObservabilityMetrics: %v", err)
	}
	if metrics.PartialSuccessRate != 0.25 {
		t.Fatalf("partial success rate = %v, want 0.25", metrics.PartialSuccessRate)
	}
	if metrics.Duration.P50MS != 200 || metrics.Duration.P95MS != 400 {
		t.Fatalf("duration percentiles = %+v, want p50=200 p95=400", metrics.Duration)
	}
	if metrics.QueueDepth != 1 {
		t.Fatalf("queue depth = %d, want 1", metrics.QueueDepth)
	}
	provider := metrics.Providers[models.DeepProviderNumista]
	if provider.StatusCounts[models.DeepProviderRunContributed] != 1 ||
		provider.Latency.P50MS != 250 || provider.Latency.P95MS != 250 {
		t.Fatalf("provider metrics = %+v", provider)
	}
}

func TestDeepIdentificationRepository_SettleRunningProviderRuns(t *testing.T) {
	db := newDeepIdentificationTestDB(t)
	repo := NewDeepIdentificationRepository(db)
	user := createDeepTestUser(t, db, "provider-settlement-owner")
	started := time.Now().Add(-250 * time.Millisecond)
	if err := repo.RecordProviderStarted(41, user.ID, models.DeepProviderNumista, true, started); err != nil {
		t.Fatal(err)
	}
	if err := repo.RecordProviderResult(
		41, user.ID, models.DeepProviderNomisma, models.DeepProviderRunContributed,
		true, 0.8, 1, 50, "", started, started.Add(50*time.Millisecond),
	); err != nil {
		t.Fatal(err)
	}

	completed := time.Now()
	if err := repo.SettleRunningProviderRuns(
		41, user.ID, models.DeepProviderRunTimedOut, "timeout", completed,
	); err != nil {
		t.Fatal(err)
	}

	var timedOut models.DeepIdentificationProviderRun
	if err := db.Where("job_id = ? AND provider = ?", 41, models.DeepProviderNumista).First(&timedOut).Error; err != nil {
		t.Fatal(err)
	}
	if timedOut.Status != models.DeepProviderRunTimedOut || timedOut.CompletedAt == nil ||
		timedOut.ErrorKind != "timeout" || timedOut.LatencyMS <= 0 {
		t.Fatalf("unfinished provider was not settled: %+v", timedOut)
	}
	var contributed models.DeepIdentificationProviderRun
	if err := db.Where("job_id = ? AND provider = ?", 41, models.DeepProviderNomisma).First(&contributed).Error; err != nil {
		t.Fatal(err)
	}
	if contributed.Status != models.DeepProviderRunContributed {
		t.Fatalf("settlement changed an already terminal provider: %+v", contributed)
	}
}

func TestDeepIdentificationRepository_IdempotentDuplicateSubmit(t *testing.T) {
	db := newDeepIdentificationTestDB(t)
	repo := NewDeepIdentificationRepository(db)
	owner := createDeepTestUser(t, db, "dupuser")

	first := &models.DeepIdentificationJob{UserID: owner.ID, Source: models.DeepJobSourceIntake, InputFingerprint: "fp-dup", ExpiresAt: time.Now().Add(90 * 24 * time.Hour)}
	createdFirst, reusedFirst, err := repo.CreateJob(first)
	if err != nil {
		t.Fatalf("first CreateJob failed: %v", err)
	}
	if reusedFirst {
		t.Fatal("first CreateJob should not be reused")
	}

	second := &models.DeepIdentificationJob{UserID: owner.ID, Source: models.DeepJobSourceIntake, InputFingerprint: "fp-dup", ExpiresAt: time.Now().Add(90 * 24 * time.Hour)}
	createdSecond, reusedSecond, err := repo.CreateJob(second)
	if err != nil {
		t.Fatalf("second CreateJob failed: %v", err)
	}
	if !reusedSecond {
		t.Fatal("duplicate submit should reuse the existing active job")
	}
	if createdSecond.ID != createdFirst.ID {
		t.Fatalf("expected reused job id %d, got %d", createdFirst.ID, createdSecond.ID)
	}

	// After the first job settles terminal, a new submission with the same
	// fingerprint must create a fresh job rather than reusing the terminal one.
	if _, err := repo.SettleTerminal(createdFirst.ID, deepJobActiveStatuses, models.DeepJobStatusCompleted, "{}", "{}", "", ""); err != nil {
		t.Fatalf("SettleTerminal failed: %v", err)
	}
	third := &models.DeepIdentificationJob{UserID: owner.ID, Source: models.DeepJobSourceIntake, InputFingerprint: "fp-dup", ExpiresAt: time.Now().Add(90 * 24 * time.Hour)}
	createdThird, reusedThird, err := repo.CreateJob(third)
	if err != nil {
		t.Fatalf("third CreateJob failed: %v", err)
	}
	if reusedThird {
		t.Fatal("after terminal settle, a duplicate submit must create a new job, not reuse the terminal one")
	}
	if createdThird.ID == createdFirst.ID {
		t.Fatal("expected a distinct job id after the prior job settled terminal")
	}
}

func TestDeepIdentificationRepository_ConcurrentAppendEventUniqueSequence(t *testing.T) {
	db := newDeepIdentificationTestDB(t)
	repo := NewDeepIdentificationRepository(db)
	owner := createDeepTestUser(t, db, "eventuser")

	job := &models.DeepIdentificationJob{UserID: owner.ID, Source: models.DeepJobSourceIntake, InputFingerprint: "fp-events", ExpiresAt: time.Now().Add(90 * 24 * time.Hour)}
	created, _, err := repo.CreateJob(job)
	if err != nil {
		t.Fatalf("CreateJob failed: %v", err)
	}

	const goroutines = 20
	var wg sync.WaitGroup
	errs := make(chan error, goroutines)
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			if _, err := repo.AppendEvent(created.ID, owner.ID, models.DeepEventProgress, fmt.Sprintf(`{"n":%d}`, n)); err != nil {
				errs <- err
			}
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatalf("AppendEvent failed under concurrency: %v", err)
	}

	events, err := repo.ListEventsSince(created.ID, owner.ID, 0)
	if err != nil {
		t.Fatalf("ListEventsSince failed: %v", err)
	}
	if len(events) != goroutines {
		t.Fatalf("expected %d events, got %d", goroutines, len(events))
	}
	seen := map[int64]bool{}
	for _, e := range events {
		if seen[e.Seq] {
			t.Fatalf("duplicate seq %d observed", e.Seq)
		}
		seen[e.Seq] = true
	}
	for i := int64(1); i <= goroutines; i++ {
		if !seen[i] {
			t.Fatalf("gap in sequence: missing seq %d", i)
		}
	}
}

func TestDeepIdentificationRepository_TerminalSettleRace(t *testing.T) {
	const iterations = 50
	for iter := 0; iter < iterations; iter++ {
		db := newDeepIdentificationTestDB(t)
		repo := NewDeepIdentificationRepository(db)
		owner := createDeepTestUser(t, db, fmt.Sprintf("raceuser%d", iter))

		job := &models.DeepIdentificationJob{
			UserID: owner.ID, Source: models.DeepJobSourceIntake,
			InputFingerprint: fmt.Sprintf("fp-race-%d", iter),
			Status:           models.DeepJobStatusRunning,
			ExpiresAt:        time.Now().Add(90 * 24 * time.Hour),
		}
		if err := db.Create(job).Error; err != nil {
			t.Fatalf("failed to seed running job: %v", err)
		}

		var wins int32
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			won, err := repo.SettleTerminal(job.ID, deepJobActiveStatuses, models.DeepJobStatusCancelled, "{}", "{}", "", "")
			if err != nil {
				t.Errorf("cancel settle error: %v", err)
				return
			}
			if won {
				atomic.AddInt32(&wins, 1)
			}
		}()
		go func() {
			defer wg.Done()
			won, err := repo.SettleTerminal(job.ID, deepJobActiveStatuses, models.DeepJobStatusCompleted, "{}", "{}", "", "")
			if err != nil {
				t.Errorf("complete settle error: %v", err)
				return
			}
			if won {
				atomic.AddInt32(&wins, 1)
			}
		}()
		wg.Wait()

		if wins != 1 {
			t.Fatalf("iteration %d: expected exactly one settle winner, got %d", iter, wins)
		}

		var terminalCount int64
		if err := db.Model(&models.DeepIdentificationEvent{}).
			Where("job_id = ? AND type = ?", job.ID, models.DeepEventTerminal).
			Count(&terminalCount).Error; err != nil {
			t.Fatalf("failed to count terminal events: %v", err)
		}
		if terminalCount != 1 {
			t.Fatalf("iteration %d: expected exactly one terminal event, got %d", iter, terminalCount)
		}
	}
}

func TestDeepIdentificationRepository_StaleRecovery(t *testing.T) {
	db := newDeepIdentificationTestDB(t)
	repo := NewDeepIdentificationRepository(db)
	owner := createDeepTestUser(t, db, "staleuser")

	staleHeartbeat := time.Now().Add(-1 * time.Hour)
	job := &models.DeepIdentificationJob{
		UserID: owner.ID, Source: models.DeepJobSourceIntake,
		InputFingerprint: "fp-stale", Status: models.DeepJobStatusRunning,
		HeartbeatAt: &staleHeartbeat, ExpiresAt: time.Now().Add(90 * 24 * time.Hour),
	}
	if err := db.Create(job).Error; err != nil {
		t.Fatalf("failed to seed stale job: %v", err)
	}

	recovered, err := repo.RecoverStaleJobs(15 * time.Minute)
	if err != nil {
		t.Fatalf("RecoverStaleJobs failed: %v", err)
	}
	if len(recovered) != 1 || recovered[0] != job.ID {
		t.Fatalf("expected job %d to be recovered, got %v", job.ID, recovered)
	}

	var reloaded models.DeepIdentificationJob
	if err := db.First(&reloaded, job.ID).Error; err != nil {
		t.Fatalf("failed to reload job: %v", err)
	}
	if reloaded.Status != models.DeepJobStatusFailed {
		t.Fatalf("expected status failed, got %s", reloaded.Status)
	}
	if reloaded.FailureCode != "stale_restart" {
		t.Fatalf("expected failure code stale_restart, got %s", reloaded.FailureCode)
	}

	var terminalCount int64
	if err := db.Model(&models.DeepIdentificationEvent{}).
		Where("job_id = ? AND type = ?", job.ID, models.DeepEventTerminal).
		Count(&terminalCount).Error; err != nil {
		t.Fatalf("failed to count terminal events: %v", err)
	}
	if terminalCount != 1 {
		t.Fatalf("expected exactly one terminal event, got %d", terminalCount)
	}
}

// newDeepIdentificationFileTestDB opens a real on-disk SQLite database (not
// the in-memory shared-cache DSN used by newDeepIdentificationTestDB) with
// WAL journal mode, mirroring how database.Connect provisions the
// production database. This is required to exercise real SQLite locking:
// SQLite silently ignores WAL mode for ":memory:"/shared-cache databases,
// so the deferred-transaction lock-upgrade contention this test targets
// cannot be reproduced against an in-memory DB - it needs a real file.
//
// dsnParams, if non-empty, is appended verbatim as the DSN query string
// (e.g. models.SQLiteConcurrencyDSNParams, "_txlock=immediate&_pragma=
// busy_timeout(5000)"). Callers should pass models.SQLiteConcurrencyDSNParams
// rather than a hand-written literal: repository/ must not import database/
// (architecture_test.go: TestNoDirectDatabaseImports), so the production DSN
// built in database.Connect (src/api/database/database.go) and this test
// both derive from that one models/ constant instead of each keeping their
// own copy in sync by hand.
func newDeepIdentificationFileTestDB(t *testing.T, dsnParams string) *gorm.DB {
	t.Helper()
	path := filepath.Join(t.TempDir(), "deep_identification.db")
	dsn := path
	if dsnParams != "" {
		dsn = path + "?" + dsnParams
	}
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open file test db: %v", err)
	}
	db.Exec("PRAGMA journal_mode=WAL")
	if err := db.AutoMigrate(
		&models.User{}, &models.Coin{},
		&models.DeepIdentificationJob{}, &models.DeepIdentificationEvent{},
		&models.DeepIdentificationProviderRun{}, &models.DeepIdentificationArtifact{},
	); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("failed to get sql.DB: %v", err)
	}
	sqlDB.SetMaxOpenConns(16)
	// Windows cannot remove a file (t.TempDir()'s cleanup) while it is
	// still open, so the sqlite connection pool must be closed before the
	// temp dir teardown runs.
	t.Cleanup(func() { sqlDB.Close() })
	return db
}

// TestDeepIdentificationRepository_ConcurrentClaimNoLockContention is the
// regression test for the production "database is locked (5) (SQLITE_BUSY)"
// defect logged by deep-identification workers: multiple worker goroutines
// calling ClaimNextQueuedJob concurrently against a real on-disk WAL
// database. ClaimNextQueuedJob's transaction does a SELECT (queued job)
// followed by an UPDATE inside the *same* GORM db.Transaction(), which - on
// SQLite's default *deferred* transaction mode - takes a read lock on the
// SELECT and then must upgrade to a write lock for the UPDATE. Under
// concurrent claims, that upgrade races with another connection's write
// lock and SQLite fails it outright (SQLITE_BUSY / SQLITE_BUSY_SNAPSHOT) -
// a plain busy_timeout does not help an upgrade conflict, only an initial
// lock wait. Verified against this exact reproduction: with the DSN used
// for the "before" case below (no _txlock, no busy_timeout - i.e. what
// database.Connect used before this fix), 30 goroutines racing 300 claims
// reliably produced ~29 SQLITE_BUSY errors. This test asserts the fixed
// DSN (_txlock=immediate, so every transaction takes its write lock at
// BEGIN instead of upgrading later, plus busy_timeout so acquiring that
// lock waits instead of failing under load) produces zero claim errors and
// claims every queued job exactly once.
func TestDeepIdentificationRepository_ConcurrentClaimNoLockContention(t *testing.T) {
	const jobCount = 150
	const workerCount = 20

	db := newDeepIdentificationFileTestDB(t, models.SQLiteConcurrencyDSNParams)
	repo := NewDeepIdentificationRepository(db)
	owner := createDeepTestUser(t, db, "claimuser")

	for i := 0; i < jobCount; i++ {
		job := &models.DeepIdentificationJob{
			UserID: owner.ID, Source: models.DeepJobSourceIntake,
			InputFingerprint: fmt.Sprintf("fp-claim-%d", i),
			ExpiresAt:        time.Now().Add(90 * 24 * time.Hour),
		}
		if _, _, err := repo.CreateJob(job); err != nil {
			t.Fatalf("failed to seed queued job %d: %v", i, err)
		}
	}

	var claimErrs int32
	var claimedTotal int32
	claimedIDs := make(chan uint, jobCount)
	var wg sync.WaitGroup
	for w := 0; w < workerCount; w++ {
		wg.Add(1)
		go func(workerID string) {
			defer wg.Done()
			for {
				job, claimed, err := repo.ClaimNextQueuedJob(workerID)
				if err != nil {
					atomic.AddInt32(&claimErrs, 1)
					t.Logf("worker %s claim error: %v", workerID, err)
					return
				}
				if !claimed {
					return
				}
				atomic.AddInt32(&claimedTotal, 1)
				claimedIDs <- job.ID
			}
		}(fmt.Sprintf("worker-%d", w))
	}
	wg.Wait()
	close(claimedIDs)

	if claimErrs != 0 {
		t.Fatalf("expected zero claim errors with the fixed DSN, got %d", claimErrs)
	}
	if claimedTotal != jobCount {
		t.Fatalf("expected all %d jobs claimed, got %d", jobCount, claimedTotal)
	}
	seen := make(map[uint]bool, jobCount)
	for id := range claimedIDs {
		if seen[id] {
			t.Fatalf("job %d claimed more than once - concurrent claim double-dequeue", id)
		}
		seen[id] = true
	}

	var stillQueued int64
	if err := db.Model(&models.DeepIdentificationJob{}).
		Where("status = ?", models.DeepJobStatusQueued).
		Count(&stillQueued).Error; err != nil {
		t.Fatalf("failed to count remaining queued jobs: %v", err)
	}
	if stillQueued != 0 {
		t.Fatalf("expected no jobs left queued, got %d", stillQueued)
	}
}

func TestDeepIdentificationRepository_EventPruningPreservesReport(t *testing.T) {
	db := newDeepIdentificationTestDB(t)
	repo := NewDeepIdentificationRepository(db)
	owner := createDeepTestUser(t, db, "pruneuser")

	job := &models.DeepIdentificationJob{UserID: owner.ID, Source: models.DeepJobSourceIntake, InputFingerprint: "fp-prune", ExpiresAt: time.Now().Add(90 * 24 * time.Hour)}
	created, _, err := repo.CreateJob(job)
	if err != nil {
		t.Fatalf("CreateJob failed: %v", err)
	}
	if _, err := repo.AppendEvent(created.ID, owner.ID, models.DeepEventJobAccepted, "{}"); err != nil {
		t.Fatalf("AppendEvent failed: %v", err)
	}
	if _, err := repo.SettleTerminal(created.ID, deepJobActiveStatuses, models.DeepJobStatusCompleted, `{"narrative":"done"}`, "{}", "", ""); err != nil {
		t.Fatalf("SettleTerminal failed: %v", err)
	}

	// Backdate completed_at so the retention cutoff catches it.
	past := time.Now().Add(-48 * time.Hour)
	if err := db.Model(&models.DeepIdentificationJob{}).Where("id = ?", created.ID).Update("completed_at", past).Error; err != nil {
		t.Fatalf("failed to backdate completed_at: %v", err)
	}

	if err := repo.PruneEventsBefore(time.Now().Add(-24 * time.Hour)); err != nil {
		t.Fatalf("PruneEventsBefore failed: %v", err)
	}

	var reloaded models.DeepIdentificationJob
	if err := db.First(&reloaded, created.ID).Error; err != nil {
		t.Fatalf("failed to reload job: %v", err)
	}
	if reloaded.EventsPrunedAt == nil {
		t.Fatal("expected EventsPrunedAt to be set")
	}
	if reloaded.ReportJSON != `{"narrative":"done"}` {
		t.Fatalf("expected report to be preserved after pruning, got %q", reloaded.ReportJSON)
	}

	var eventCount int64
	if err := db.Model(&models.DeepIdentificationEvent{}).Where("job_id = ?", created.ID).Count(&eventCount).Error; err != nil {
		t.Fatalf("failed to count events: %v", err)
	}
	if eventCount != 0 {
		t.Fatalf("expected all pre-cutoff events pruned, got %d remaining", eventCount)
	}
}

func TestDeepIdentificationRepository_SettleTerminalUsesSentinelForCompletedAndPartial(t *testing.T) {
	db := newDeepIdentificationTestDB(t)
	repo := NewDeepIdentificationRepository(db)
	owner := createDeepTestUser(t, db, "settle-sentinel")

	makeJob := func(fp string) uint {
		job := &models.DeepIdentificationJob{
			UserID:           owner.ID,
			Source:           models.DeepJobSourceIntake,
			Status:           models.DeepJobStatusQueued,
			InputFingerprint: fp,
			ExpiresAt:        time.Now().Add(48 * time.Hour),
		}
		created, _, err := repo.CreateJob(job)
		if err != nil {
			t.Fatalf("CreateJob failed: %v", err)
		}
		return created.ID
	}

	completedID := makeJob("fp-sentinel-completed")
	if _, err := repo.SettleTerminal(completedID, deepJobActiveStatuses, models.DeepJobStatusCompleted, "{}", "{}", "", ""); err != nil {
		t.Fatalf("SettleTerminal completed failed: %v", err)
	}
	partialID := makeJob("fp-sentinel-partial")
	if _, err := repo.SettleTerminal(partialID, deepJobActiveStatuses, models.DeepJobStatusPartial, "{}", "{}", "", ""); err != nil {
		t.Fatalf("SettleTerminal partial failed: %v", err)
	}

	var completed models.DeepIdentificationJob
	if err := db.First(&completed, completedID).Error; err != nil {
		t.Fatal(err)
	}
	if !completed.ExpiresAt.Equal(models.DeepIdentificationNoExpirySentinel) {
		t.Fatalf("expected completed expires_at sentinel, got %s", completed.ExpiresAt.UTC())
	}
	var partial models.DeepIdentificationJob
	if err := db.First(&partial, partialID).Error; err != nil {
		t.Fatal(err)
	}
	if !partial.ExpiresAt.Equal(models.DeepIdentificationNoExpirySentinel) {
		t.Fatalf("expected partial expires_at sentinel, got %s", partial.ExpiresAt.UTC())
	}
}

func TestDeepIdentificationRepository_SettleTerminalPreservesExpiryForFailed(t *testing.T) {
	db := newDeepIdentificationTestDB(t)
	repo := NewDeepIdentificationRepository(db)
	owner := createDeepTestUser(t, db, "settle-failed")

	initialExpiry := time.Now().Add(36 * time.Hour).UTC().Truncate(time.Second)
	job := &models.DeepIdentificationJob{
		UserID:           owner.ID,
		Source:           models.DeepJobSourceIntake,
		Status:           models.DeepJobStatusQueued,
		InputFingerprint: "fp-sentinel-failed",
		ExpiresAt:        initialExpiry,
	}
	created, _, err := repo.CreateJob(job)
	if err != nil {
		t.Fatalf("CreateJob failed: %v", err)
	}
	if _, err := repo.SettleTerminal(created.ID, deepJobActiveStatuses, models.DeepJobStatusFailed, "", "", "x", "err"); err != nil {
		t.Fatalf("SettleTerminal failed failed: %v", err)
	}

	var reloaded models.DeepIdentificationJob
	if err := db.First(&reloaded, created.ID).Error; err != nil {
		t.Fatal(err)
	}
	if !reloaded.ExpiresAt.UTC().Truncate(time.Second).Equal(initialExpiry) {
		t.Fatalf("expected failed expiry to be preserved (%s), got %s", initialExpiry, reloaded.ExpiresAt.UTC())
	}
}

func TestDeepIdentificationRepository_ListJobsIncludesAppliedCoinExistsProjection(t *testing.T) {
	db := newDeepIdentificationTestDB(t)
	repo := NewDeepIdentificationRepository(db)
	owner := createDeepTestUser(t, db, "projection-owner")

	coin := models.Coin{
		UserID:       owner.ID,
		Name:         "Projection Coin",
		Category:     "Roman",
		Material:     "Silver",
		IsWishlist:   true,
		IsPrivate:    true,
		Denomination: "Denarius",
	}
	if err := db.Create(&coin).Error; err != nil {
		t.Fatalf("seed coin: %v", err)
	}
	job := &models.DeepIdentificationJob{
		UserID:           owner.ID,
		Source:           models.DeepJobSourceIntake,
		Status:           models.DeepJobStatusCompleted,
		InputFingerprint: "fp-projection",
		ExpiresAt:        models.DeepIdentificationNoExpirySentinel,
		AppliedCoinID:    &coin.ID,
		AppliedAt:        ptrTime(time.Now()),
	}
	if err := db.Create(job).Error; err != nil {
		t.Fatalf("seed job: %v", err)
	}

	list, err := repo.ListJobs(owner.ID, DeepJobListFilters{})
	if err != nil {
		t.Fatalf("ListJobs failed: %v", err)
	}
	if len(list) != 1 || !list[0].AppliedCoinExists {
		t.Fatalf("expected appliedCoinExists=true for existing linked coin, got %+v", list)
	}

	if err := db.Delete(&coin).Error; err != nil {
		t.Fatalf("delete coin: %v", err)
	}
	list, err = repo.ListJobs(owner.ID, DeepJobListFilters{})
	if err != nil {
		t.Fatalf("ListJobs after delete failed: %v", err)
	}
	if len(list) != 1 || list[0].AppliedCoinExists {
		t.Fatalf("expected appliedCoinExists=false after coin deletion, got %+v", list)
	}
}

func TestDeepIdentificationRepository_DeleteJobOwnerTerminalOnly(t *testing.T) {
	db := newDeepIdentificationTestDB(t)
	repo := NewDeepIdentificationRepository(db)
	owner := createDeepTestUser(t, db, "delete-owner")

	job := &models.DeepIdentificationJob{
		UserID:           owner.ID,
		Source:           models.DeepJobSourceIntake,
		Status:           models.DeepJobStatusCompleted,
		InputFingerprint: "fp-delete-terminal",
		ExpiresAt:        models.DeepIdentificationNoExpirySentinel,
	}
	if _, _, err := repo.CreateJob(job); err != nil {
		t.Fatalf("CreateJob failed: %v", err)
	}
	if err := db.Create(&models.DeepIdentificationProviderRun{JobID: job.ID, UserID: owner.ID, Provider: "numista", Status: models.DeepProviderRunContributed}).Error; err != nil {
		t.Fatalf("seed provider run: %v", err)
	}
	if err := db.Create(&models.DeepIdentificationEvent{JobID: job.ID, UserID: owner.ID, Seq: 1, Type: models.DeepEventProgress, PayloadJSON: "{}"}).Error; err != nil {
		t.Fatalf("seed event: %v", err)
	}
	if err := db.Create(&models.DeepIdentificationArtifact{JobID: job.ID, UserID: owner.ID, Role: models.DeepArtifactRoleHint, Origin: models.DeepArtifactOriginUploaded, FilePath: "tmp", ContentHash: "hash"}).Error; err != nil {
		t.Fatalf("seed artifact: %v", err)
	}

	if err := repo.DeleteJob(owner.ID, job.ID); err != nil {
		t.Fatalf("DeleteJob failed: %v", err)
	}
	var jobs int64
	if err := db.Model(&models.DeepIdentificationJob{}).Where("id = ?", job.ID).Count(&jobs).Error; err != nil {
		t.Fatalf("count jobs: %v", err)
	}
	if jobs != 0 {
		t.Fatalf("expected hard-deleted job, got count=%d", jobs)
	}

	running := &models.DeepIdentificationJob{
		UserID:           owner.ID,
		Source:           models.DeepJobSourceIntake,
		Status:           models.DeepJobStatusRunning,
		InputFingerprint: "fp-delete-running",
		ExpiresAt:        time.Now().Add(24 * time.Hour),
	}
	if _, _, err := repo.CreateJob(running); err != nil {
		t.Fatalf("CreateJob running failed: %v", err)
	}
	if err := repo.DeleteJob(owner.ID, running.ID); !errors.Is(err, ErrDeepJobNotTerminal) {
		t.Fatalf("expected ErrDeepJobNotTerminal, got %v", err)
	}
}
