package repository

import (
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// setupAvailabilityCycleTestDB creates an in-memory SQLite DB with the tables needed to
// exercise AvailabilityCycleRepository in isolation (Feature 353).
func setupAvailabilityCycleTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	if err := db.AutoMigrate(
		&models.User{},
		&models.AvailabilityCycle{},
		&models.AvailabilityRun{},
		&models.AvailabilityResult{},
	); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	return db
}

func newCycle(triggerType string, status string, startedAt time.Time) *models.AvailabilityCycle {
	c := &models.AvailabilityCycle{
		TriggerType: triggerType,
		Status:      status,
		StartedAt:   startedAt,
	}
	return c
}

// --- T003: EnqueueCycle idempotency (FR-007, US2 AC2) ---

func TestEnqueueCycle_FirstCallAcquires(t *testing.T) {
	db := setupAvailabilityCycleTestDB(t)
	repo := NewAvailabilityCycleRepository(db)

	since := time.Now().Add(-5 * time.Minute)
	cycle := newCycle(models.AvailabilityRunTriggerAdmin, models.AvailabilityCycleStatusQueued, time.Now())

	acquired, err := repo.EnqueueCycle(cycle, since)
	if err != nil {
		t.Fatalf("EnqueueCycle: %v", err)
	}
	if !acquired {
		t.Fatal("expected first EnqueueCycle call to acquire")
	}
	if cycle.ID == 0 {
		t.Fatal("expected cycle to be persisted with a non-zero ID")
	}
}

func TestEnqueueCycle_DuplicateWithinWindowRejected(t *testing.T) {
	db := setupAvailabilityCycleTestDB(t)
	repo := NewAvailabilityCycleRepository(db)

	since := time.Now().Add(-5 * time.Minute)
	first := newCycle(models.AvailabilityRunTriggerAdmin, models.AvailabilityCycleStatusQueued, time.Now())
	acquired, err := repo.EnqueueCycle(first, since)
	if err != nil || !acquired {
		t.Fatalf("expected first enqueue to succeed, acquired=%v err=%v", acquired, err)
	}

	second := newCycle(models.AvailabilityRunTriggerAdmin, models.AvailabilityCycleStatusQueued, time.Now())
	acquired2, err := repo.EnqueueCycle(second, since)
	if err != nil {
		t.Fatalf("EnqueueCycle second call: %v", err)
	}
	if acquired2 {
		t.Fatal("expected duplicate enqueue within the window to be rejected")
	}
	if second.ID != 0 {
		t.Fatal("expected no row created for the rejected duplicate cycle")
	}

	var count int64
	db.Model(&models.AvailabilityCycle{}).Count(&count)
	if count != 1 {
		t.Fatalf("expected exactly 1 cycle row to exist, got %d", count)
	}
}

func TestEnqueueCycle_OutsideWindowAllowsNewCycle(t *testing.T) {
	db := setupAvailabilityCycleTestDB(t)
	repo := NewAvailabilityCycleRepository(db)

	// Seed a queued cycle that started well before the 5-minute window.
	stale := newCycle(models.AvailabilityRunTriggerScheduled, models.AvailabilityCycleStatusQueued, time.Now().Add(-10*time.Minute))
	if err := db.Create(stale).Error; err != nil {
		t.Fatalf("seed stale cycle: %v", err)
	}

	since := time.Now().Add(-5 * time.Minute)
	next := newCycle(models.AvailabilityRunTriggerAdmin, models.AvailabilityCycleStatusQueued, time.Now())
	acquired, err := repo.EnqueueCycle(next, since)
	if err != nil {
		t.Fatalf("EnqueueCycle: %v", err)
	}
	if !acquired {
		t.Fatal("expected enqueue to succeed when only stale (outside-window) cycles exist")
	}
}

func TestEnqueueCycle_RunningCycleBlocksNewOne(t *testing.T) {
	db := setupAvailabilityCycleTestDB(t)
	repo := NewAvailabilityCycleRepository(db)

	running := newCycle(models.AvailabilityRunTriggerScheduled, models.AvailabilityCycleStatusRunning, time.Now())
	if err := db.Create(running).Error; err != nil {
		t.Fatalf("seed running cycle: %v", err)
	}

	since := time.Now().Add(-5 * time.Minute)
	next := newCycle(models.AvailabilityRunTriggerAdmin, models.AvailabilityCycleStatusQueued, time.Now())
	acquired, err := repo.EnqueueCycle(next, since)
	if err != nil {
		t.Fatalf("EnqueueCycle: %v", err)
	}
	if acquired {
		t.Fatal("expected enqueue to be rejected while a cycle is running")
	}
}

// --- T003: ClaimCycle transitions queued -> running exactly once (FR-008) ---

func TestClaimCycle_TransitionsQueuedToRunningExactlyOnce(t *testing.T) {
	db := setupAvailabilityCycleTestDB(t)
	repo := NewAvailabilityCycleRepository(db)

	cycle := newCycle(models.AvailabilityRunTriggerAdmin, models.AvailabilityCycleStatusQueued, time.Now())
	if err := db.Create(cycle).Error; err != nil {
		t.Fatalf("seed cycle: %v", err)
	}

	claimedCycle, claimed, err := repo.ClaimCycle(cycle.ID)
	if err != nil {
		t.Fatalf("ClaimCycle: %v", err)
	}
	if !claimed {
		t.Fatal("expected first claim to succeed")
	}
	if claimedCycle.Status != models.AvailabilityCycleStatusRunning {
		t.Fatalf("expected status running after claim, got %q", claimedCycle.Status)
	}

	// A second claim attempt must not succeed again.
	_, claimedAgain, err := repo.ClaimCycle(cycle.ID)
	if err != nil {
		t.Fatalf("ClaimCycle second call: %v", err)
	}
	if claimedAgain {
		t.Fatal("expected second claim of an already-running cycle to fail")
	}

	var reloaded models.AvailabilityCycle
	db.First(&reloaded, cycle.ID)
	if reloaded.Status != models.AvailabilityCycleStatusRunning {
		t.Fatalf("expected status to remain running, got %q", reloaded.Status)
	}
}

func TestClaimCycle_NonQueuedCycleNotClaimed(t *testing.T) {
	db := setupAvailabilityCycleTestDB(t)
	repo := NewAvailabilityCycleRepository(db)

	cycle := newCycle(models.AvailabilityRunTriggerAdmin, models.AvailabilityCycleStatusCompleted, time.Now())
	if err := db.Create(cycle).Error; err != nil {
		t.Fatalf("seed cycle: %v", err)
	}

	_, claimed, err := repo.ClaimCycle(cycle.ID)
	if err != nil {
		t.Fatalf("ClaimCycle: %v", err)
	}
	if claimed {
		t.Fatal("expected a completed cycle to never be claimable")
	}
}

// --- T003: AggregateChildCounts truth table (FR-008, US2 AC3/AC4) ---

func seedChildRun(t *testing.T, db *gorm.DB, cycleID uint, userID uint, status string) models.AvailabilityRun {
	t.Helper()
	run := models.AvailabilityRun{
		UserID:      userID,
		CycleID:     &cycleID,
		TriggerType: models.AvailabilityRunTriggerAdmin,
		Status:      status,
		StartedAt:   time.Now(),
	}
	if err := db.Create(&run).Error; err != nil {
		t.Fatalf("seed child run: %v", err)
	}
	return run
}

func TestAggregateChildCounts_TruthTable(t *testing.T) {
	tests := []struct {
		name          string
		statuses      []string
		wantStatus    string
		wantTerminal  bool
		wantTotal     int
		wantQueued    int
		wantRunning   int
		wantCompleted int
		wantFailed    int
	}{
		{
			name:          "all completed -> completed",
			statuses:      []string{models.AvailabilityRunStatusCompleted, models.AvailabilityRunStatusCompleted, models.AvailabilityRunStatusCompleted},
			wantStatus:    models.AvailabilityCycleStatusCompleted,
			wantTerminal:  true,
			wantTotal:     3,
			wantCompleted: 3,
		},
		{
			name:         "all failed -> failed",
			statuses:     []string{models.AvailabilityRunStatusFailed, models.AvailabilityRunStatusFailed},
			wantStatus:   models.AvailabilityCycleStatusFailed,
			wantTerminal: true,
			wantTotal:    2,
			wantFailed:   2,
		},
		{
			name:          "mixed completed+failed -> partial_failure",
			statuses:      []string{models.AvailabilityRunStatusCompleted, models.AvailabilityRunStatusFailed, models.AvailabilityRunStatusCompleted},
			wantStatus:    models.AvailabilityCycleStatusPartialFailure,
			wantTerminal:  true,
			wantTotal:     3,
			wantCompleted: 2,
			wantFailed:    1,
		},
		{
			name:          "any queued -> running (not terminal)",
			statuses:      []string{models.AvailabilityRunStatusCompleted, models.AvailabilityRunStatusQueued},
			wantStatus:    models.AvailabilityCycleStatusRunning,
			wantTerminal:  false,
			wantTotal:     2,
			wantQueued:    1,
			wantCompleted: 1,
		},
		{
			name:          "any running -> running (not terminal)",
			statuses:      []string{models.AvailabilityRunStatusRunning, models.AvailabilityRunStatusCompleted, models.AvailabilityRunStatusFailed},
			wantStatus:    models.AvailabilityCycleStatusRunning,
			wantTerminal:  false,
			wantTotal:     3,
			wantRunning:   1,
			wantCompleted: 1,
			wantFailed:    1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupAvailabilityCycleTestDB(t)
			repo := NewAvailabilityCycleRepository(db)

			cycle := newCycle(models.AvailabilityRunTriggerAdmin, models.AvailabilityCycleStatusRunning, time.Now())
			if err := db.Create(cycle).Error; err != nil {
				t.Fatalf("seed cycle: %v", err)
			}

			user := models.User{Username: "u-" + tt.name, Email: tt.name + "@test.com"}
			db.Create(&user)

			for _, status := range tt.statuses {
				seedChildRun(t, db, cycle.ID, user.ID, status)
			}

			counts, err := repo.AggregateChildCounts(cycle.ID)
			if err != nil {
				t.Fatalf("AggregateChildCounts: %v", err)
			}
			if counts.Total != tt.wantTotal {
				t.Errorf("Total = %d, want %d", counts.Total, tt.wantTotal)
			}
			if counts.Queued != tt.wantQueued {
				t.Errorf("Queued = %d, want %d", counts.Queued, tt.wantQueued)
			}
			if counts.Running != tt.wantRunning {
				t.Errorf("Running = %d, want %d", counts.Running, tt.wantRunning)
			}
			if counts.Completed != tt.wantCompleted {
				t.Errorf("Completed = %d, want %d", counts.Completed, tt.wantCompleted)
			}
			if counts.Failed != tt.wantFailed {
				t.Errorf("Failed = %d, want %d", counts.Failed, tt.wantFailed)
			}
		})
	}
}

// --- T003 / T032: PruneOldCycles keeps <=20 terminal cycles and nulls out cycle_id on
// surviving children first (FR-019, US4 AC2) ---

func TestPruneOldCycles_KeepsAtMost20Terminal(t *testing.T) {
	db := setupAvailabilityCycleTestDB(t)
	repo := NewAvailabilityCycleRepository(db)

	user := models.User{Username: "prune-owner", Email: "prune-owner@test.com"}
	db.Create(&user)

	// Seed 25 terminal cycles, oldest first, each completed at a distinct, increasing time.
	base := time.Now().Add(-25 * time.Hour)
	var cycleIDs []uint
	for i := 0; i < 25; i++ {
		completedAt := base.Add(time.Duration(i) * time.Hour)
		cycle := &models.AvailabilityCycle{
			TriggerType: models.AvailabilityRunTriggerAdmin,
			Status:      models.AvailabilityCycleStatusCompleted,
			StartedAt:   completedAt,
			CompletedAt: &completedAt,
		}
		if err := db.Create(cycle).Error; err != nil {
			t.Fatalf("seed cycle %d: %v", i, err)
		}
		cycleIDs = append(cycleIDs, cycle.ID)
	}

	// Attach a surviving child run to the very oldest cycle so we can verify its CycleID is
	// nulled (not deleted) when that cycle is pruned.
	oldestCycleID := cycleIDs[0]
	survivingChild := seedChildRun(t, db, oldestCycleID, user.ID, models.AvailabilityRunStatusCompleted)

	repo.PruneOldCycles(20)

	var remaining int64
	db.Model(&models.AvailabilityCycle{}).Count(&remaining)
	if remaining != 20 {
		t.Fatalf("expected 20 remaining cycles, got %d", remaining)
	}

	// The oldest cycle should be gone.
	var oldestCount int64
	db.Model(&models.AvailabilityCycle{}).Where("id = ?", oldestCycleID).Count(&oldestCount)
	if oldestCount != 0 {
		t.Fatalf("expected oldest cycle %d to be pruned", oldestCycleID)
	}

	// The surviving child run must still exist, with its CycleID nulled — never deleted.
	var reloadedChild models.AvailabilityRun
	if err := db.First(&reloadedChild, survivingChild.ID).Error; err != nil {
		t.Fatalf("expected surviving child run to still exist: %v", err)
	}
	if reloadedChild.CycleID != nil {
		t.Fatalf("expected surviving child's CycleID to be nulled after parent prune, got %v", *reloadedChild.CycleID)
	}
}

func TestPruneOldCycles_NeverTouchesQueuedOrRunning(t *testing.T) {
	db := setupAvailabilityCycleTestDB(t)
	repo := NewAvailabilityCycleRepository(db)

	// 25 terminal cycles plus 1 queued and 1 running that must never be pruned.
	base := time.Now().Add(-25 * time.Hour)
	for i := 0; i < 25; i++ {
		completedAt := base.Add(time.Duration(i) * time.Hour)
		db.Create(&models.AvailabilityCycle{
			TriggerType: models.AvailabilityRunTriggerAdmin,
			Status:      models.AvailabilityCycleStatusCompleted,
			StartedAt:   completedAt,
			CompletedAt: &completedAt,
		})
	}
	queued := &models.AvailabilityCycle{TriggerType: models.AvailabilityRunTriggerAdmin, Status: models.AvailabilityCycleStatusQueued, StartedAt: time.Now()}
	running := &models.AvailabilityCycle{TriggerType: models.AvailabilityRunTriggerScheduled, Status: models.AvailabilityCycleStatusRunning, StartedAt: time.Now()}
	db.Create(queued)
	db.Create(running)

	repo.PruneOldCycles(20)

	var terminalCount int64
	db.Model(&models.AvailabilityCycle{}).Where("status IN ?", []string{
		models.AvailabilityCycleStatusCompleted, models.AvailabilityCycleStatusFailed, models.AvailabilityCycleStatusPartialFailure,
	}).Count(&terminalCount)
	if terminalCount != 20 {
		t.Fatalf("expected 20 terminal cycles remaining, got %d", terminalCount)
	}

	var queuedStill, runningStill models.AvailabilityCycle
	if err := db.First(&queuedStill, queued.ID).Error; err != nil {
		t.Fatalf("expected queued cycle to survive pruning: %v", err)
	}
	if err := db.First(&runningStill, running.ID).Error; err != nil {
		t.Fatalf("expected running cycle to survive pruning: %v", err)
	}
}

// --- T039 support: RecoverStaleCycles ---

func TestRecoverStaleCycles_FinalizesCycleWhoseChildrenAreAllTerminal(t *testing.T) {
	db := setupAvailabilityCycleTestDB(t)
	repo := NewAvailabilityCycleRepository(db)

	user := models.User{Username: "stale-owner", Email: "stale-owner@test.com"}
	db.Create(&user)

	staleStart := time.Now().Add(-20 * time.Minute)
	cycle := &models.AvailabilityCycle{
		TriggerType: models.AvailabilityRunTriggerAdmin,
		Status:      models.AvailabilityCycleStatusRunning,
		StartedAt:   staleStart,
	}
	if err := db.Create(cycle).Error; err != nil {
		t.Fatalf("seed cycle: %v", err)
	}
	seedChildRun(t, db, cycle.ID, user.ID, models.AvailabilityRunStatusCompleted)
	seedChildRun(t, db, cycle.ID, user.ID, models.AvailabilityRunStatusFailed)

	if _, err := repo.RecoverStaleCycles(15 * time.Minute); err != nil {
		t.Fatalf("RecoverStaleCycles: %v", err)
	}

	var reloaded models.AvailabilityCycle
	db.First(&reloaded, cycle.ID)
	if reloaded.Status != models.AvailabilityCycleStatusPartialFailure {
		t.Fatalf("expected cycle finalized to partial_failure, got %q", reloaded.Status)
	}
	if reloaded.CompletedAt == nil {
		t.Fatal("expected CompletedAt to be set on finalized stale cycle")
	}
}

func TestRecoverStaleCycles_LeavesCycleWithActiveChildrenAlone(t *testing.T) {
	db := setupAvailabilityCycleTestDB(t)
	repo := NewAvailabilityCycleRepository(db)

	user := models.User{Username: "active-owner", Email: "active-owner@test.com"}
	db.Create(&user)

	staleStart := time.Now().Add(-20 * time.Minute)
	cycle := &models.AvailabilityCycle{
		TriggerType: models.AvailabilityRunTriggerAdmin,
		Status:      models.AvailabilityCycleStatusRunning,
		StartedAt:   staleStart,
	}
	if err := db.Create(cycle).Error; err != nil {
		t.Fatalf("seed cycle: %v", err)
	}
	seedChildRun(t, db, cycle.ID, user.ID, models.AvailabilityRunStatusCompleted)
	seedChildRun(t, db, cycle.ID, user.ID, models.AvailabilityRunStatusRunning)

	if _, err := repo.RecoverStaleCycles(15 * time.Minute); err != nil {
		t.Fatalf("RecoverStaleCycles: %v", err)
	}

	var reloaded models.AvailabilityCycle
	db.First(&reloaded, cycle.ID)
	if reloaded.Status != models.AvailabilityCycleStatusRunning {
		t.Fatalf("expected cycle with an active child to remain running, got %q", reloaded.Status)
	}
	if reloaded.CompletedAt != nil {
		t.Fatal("expected CompletedAt to remain nil while a child is still active")
	}
}
