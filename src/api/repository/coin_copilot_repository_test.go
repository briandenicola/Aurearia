package repository

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func newCopilotRepositoryTestDB(t *testing.T) (*gorm.DB, *CoinCopilotRepository) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.CoinCopilotThread{}, &models.CoinCopilotRun{}, &models.CoinCopilotCheckpoint{}, &models.CoinCopilotEvent{}, &models.CoinCopilotResumeRequest{}); err != nil {
		t.Fatal(err)
	}
	return db, NewCoinCopilotRepository(db)
}

func seedCopilotRun(t *testing.T, repo *CoinCopilotRepository, userID uint, status models.CopilotRunStatus) *models.CoinCopilotRun {
	t.Helper()
	thread := &models.CoinCopilotThread{ID: "cct_test", UserID: userID, Title: "Test"}
	run := &models.CoinCopilotRun{
		ID: "ccr_test", ThreadID: thread.ID, UserID: userID, Status: status, Goal: "test",
		StartIdempotencyKeyHash: "key", StartRequestFingerprint: "fingerprint",
		ExecutionID: "cce_test", MaxIterations: 8, MaxToolCalls: 12, MaxConcurrentTools: 1,
		HardTimeoutSeconds: 120, MaxPersistedToolResultBytes: 32768,
	}
	if err := repo.CreateRun(thread, run); err != nil {
		t.Fatal(err)
	}
	return run
}

func TestCoinCopilotRepositoryOwnerScopeHidesForeignIDs(t *testing.T) {
	_, repo := newCopilotRepositoryTestDB(t)
	run := seedCopilotRun(t, repo, 7, models.CopilotRunQueued)
	if _, err := repo.GetRun(run.ID, 8); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("foreign run lookup error = %v, want record not found", err)
	}
	if _, _, err := repo.GetThread(run.ThreadID, 8); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("foreign thread lookup error = %v, want record not found", err)
	}
}

func TestCoinCopilotRepositoryHistoryIsOwnerScopedAndBounded(t *testing.T) {
	db, repo := newCopilotRepositoryTestDB(t)
	thread := &models.CoinCopilotThread{ID: "cct_history", UserID: 7, Title: "History"}
	if err := db.Create(thread).Error; err != nil {
		t.Fatal(err)
	}
	base := time.Now().UTC().Add(-time.Hour)
	for i := 0; i < 5; i++ {
		run := &models.CoinCopilotRun{
			ID: fmt.Sprintf("ccr_history_%d", i), ThreadID: thread.ID, UserID: 7, Status: models.CopilotRunCompleted,
			Goal: fmt.Sprintf("goal-%d", i), FinalAnswer: fmt.Sprintf("answer-%d", i),
			StartIdempotencyKeyHash: fmt.Sprintf("key-%d", i), StartRequestFingerprint: "fingerprint",
			MaxIterations: 8, MaxToolCalls: 12, MaxConcurrentTools: 1, HardTimeoutSeconds: 120,
			MaxPersistedToolResultBytes: 32768, CreatedAt: base.Add(time.Duration(i) * time.Minute),
		}
		if err := db.Create(run).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Create(&models.CoinCopilotRun{
		ID: "ccr_history_foreign", ThreadID: thread.ID, UserID: 8, Status: models.CopilotRunCompleted,
		Goal: "foreign-private-goal", FinalAnswer: "foreign-private-answer",
		StartIdempotencyKeyHash: "foreign-key", StartRequestFingerprint: "fingerprint",
		MaxIterations: 8, MaxToolCalls: 12, MaxConcurrentTools: 1, HardTimeoutSeconds: 120,
		MaxPersistedToolResultBytes: 32768, CreatedAt: base.Add(10 * time.Minute),
	}).Error; err != nil {
		t.Fatal(err)
	}
	runs, err := repo.ListSettledRunsForHistory(thread.ID, "ccr_current", 7, 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(runs) != 3 || runs[0].Goal != "goal-4" || runs[2].Goal != "goal-2" {
		t.Fatalf("history = %#v", runs)
	}
	for _, run := range runs {
		if run.UserID != 0 || run.AppContextJSON != "" || run.Goal == "foreign-private-goal" {
			t.Fatalf("history leaked private or foreign fields: %#v", run)
		}
	}
}

func TestCoinCopilotRepositoryEventSequenceAndTerminalUniqueness(t *testing.T) {
	db, repo := newCopilotRepositoryTestDB(t)
	run := seedCopilotRun(t, repo, 7, models.CopilotRunRunning)
	first, err := repo.AppendEvent(run.ID, run.UserID, run.ExecutionID, models.CopilotEventRunStarted, `{}`)
	if err != nil {
		t.Fatal(err)
	}
	second, err := repo.AppendEvent(run.ID, run.UserID, run.ExecutionID, models.CopilotEventPlanUpdated, `{"plan":[]}`)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.AppendEvent(run.ID, run.UserID, "cce_stale", models.CopilotEventPlanUpdated, `{"plan":[]}`); !errors.Is(err, ErrCopilotTransitionConflict) {
		t.Fatalf("stale execution append error = %v, want transition conflict", err)
	}
	if first.Seq != 1 || second.Seq != 2 {
		t.Fatalf("sequences = %d,%d, want 1,2", first.Seq, second.Seq)
	}
	won, _, err := repo.TransitionWithEvent(run.ID, run.UserID, run.ExecutionID,
		[]models.CopilotRunStatus{models.CopilotRunRunning}, models.CopilotRunCompleted,
		map[string]interface{}{"final_answer": "done"}, models.CopilotEventRunCompleted, `{"answer":"done"}`)
	if err != nil || !won {
		t.Fatalf("first terminal transition won=%v err=%v", won, err)
	}
	if _, err := repo.AppendEvent(run.ID, run.UserID, run.ExecutionID, models.CopilotEventPlanUpdated, `{"plan":[]}`); !errors.Is(err, ErrCopilotTransitionConflict) {
		t.Fatalf("post-terminal append error = %v, want transition conflict", err)
	}
	won, _, err = repo.TransitionWithEvent(run.ID, run.UserID, run.ExecutionID,
		[]models.CopilotRunStatus{models.CopilotRunRunning}, models.CopilotRunFailed,
		map[string]interface{}{"failure_code": "internal"}, models.CopilotEventRunFailed, `{"code":"internal"}`)
	if err != nil || won {
		t.Fatalf("second terminal transition won=%v err=%v", won, err)
	}
	var terminalCount int64
	if err := db.Model(&models.CoinCopilotEvent{}).Where("run_id = ? AND type IN ?", run.ID,
		[]models.CopilotEventType{models.CopilotEventRunCompleted, models.CopilotEventRunFailed, models.CopilotEventRunCancelled}).Count(&terminalCount).Error; err != nil {
		t.Fatal(err)
	}
	if terminalCount != 1 {
		t.Fatalf("terminal event count = %d, want 1", terminalCount)
	}
}

func TestCoinCopilotRepositoryRejectsInvalidEventAndTransitionPairs(t *testing.T) {
	_, repo := newCopilotRepositoryTestDB(t)
	run := seedCopilotRun(t, repo, 7, models.CopilotRunRunning)
	if _, err := repo.AppendEvent(run.ID, run.UserID, run.ExecutionID, models.CopilotEventType("invented"), `{}`); !errors.Is(err, ErrCopilotTransitionConflict) {
		t.Fatalf("unknown event error = %v", err)
	}
	if _, err := repo.AppendEvent(run.ID, run.UserID, run.ExecutionID, models.CopilotEventRunCompleted, `{}`); !errors.Is(err, ErrCopilotTransitionConflict) {
		t.Fatalf("terminal append error = %v", err)
	}
	if won, _, err := repo.TransitionWithEvent(
		run.ID, run.UserID, run.ExecutionID, []models.CopilotRunStatus{models.CopilotRunRunning},
		models.CopilotRunCompleted, map[string]any{}, models.CopilotEventRunFailed, `{}`,
	); !errors.Is(err, ErrCopilotTransitionConflict) || won {
		t.Fatalf("mismatched terminal transition won=%v err=%v", won, err)
	}
}

func TestCoinCopilotRepositoryResumeAndDeleteCascade(t *testing.T) {
	db, repo := newCopilotRepositoryTestDB(t)
	run := seedCopilotRun(t, repo, 7, models.CopilotRunPaused)
	deadline := time.Now().UTC().Add(time.Hour)
	if err := db.Model(run).Updates(map[string]interface{}{"checkpoint_version": 1, "resume_deadline": deadline}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.CoinCopilotCheckpoint{
		RunID: run.ID, ThreadID: run.ThreadID, UserID: run.UserID, ExecutionID: run.ExecutionID,
		Version: 1, StateJSON: `{"schema_version":1}`, StateDigest: "digest",
	}).Error; err != nil {
		t.Fatal(err)
	}
	resumed, err := repo.Resume(run.ID, run.UserID, "resume-key", "body", "cce_next", "answer", 1, `{"schema_version":1}`, "digest2")
	if err != nil {
		t.Fatal(err)
	}
	if resumed.Status != models.CopilotRunQueued || resumed.CheckpointVersion != 2 {
		t.Fatalf("resumed run = %#v", resumed)
	}
	events, err := repo.ListEventsSince(run.ID, run.UserID, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].Type != models.CopilotEventRunResumed || events[0].Seq != 1 {
		t.Fatalf("resume events = %#v", events)
	}
	if err := db.Model(resumed).Updates(map[string]interface{}{"status": models.CopilotRunCompleted, "completed_at": time.Now().UTC()}).Error; err != nil {
		t.Fatal(err)
	}
	if err := repo.DeleteThread(run.ThreadID, run.UserID); err != nil {
		t.Fatal(err)
	}
	for _, model := range []any{&models.CoinCopilotRun{}, &models.CoinCopilotCheckpoint{}, &models.CoinCopilotResumeRequest{}, &models.CoinCopilotEvent{}} {
		var count int64
		if err := db.Model(model).Count(&count).Error; err != nil || count != 0 {
			t.Fatalf("%T count=%d err=%v", model, count, err)
		}
	}
}
