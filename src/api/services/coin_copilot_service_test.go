package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func newCopilotServiceTest(t *testing.T) (*gorm.DB, *CoinCopilotService) {
	t.Helper()
	path := fmt.Sprintf("coin_copilot_service_%d.db", time.Now().UnixNano())
	dsn := path + "?" + models.SQLiteConcurrencyDSNParams
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(16)
	if err := db.Exec("PRAGMA journal_mode=WAL").Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = sqlDB.Close()
		_ = os.Remove(path)
		_ = os.Remove(path + "-wal")
		_ = os.Remove(path + "-shm")
	})
	if err := db.AutoMigrate(
		&models.AppSetting{}, &models.CoinCopilotThread{}, &models.CoinCopilotRun{},
		&models.CoinCopilotCheckpoint{}, &models.CoinCopilotEvent{}, &models.CoinCopilotResumeRequest{},
	); err != nil {
		t.Fatal(err)
	}
	settings := NewSettingsService(repository.NewSettingsRepository(db))
	_ = settings.SetSetting(SettingCoinCopilotEnabled, "true")
	_ = settings.SetSetting(SettingAIProvider, "anthropic")
	_ = settings.SetSetting(SettingAnthropicAPIKey, "test-key")
	agent := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/copilot/capability" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(`{"supported":true}`))
	}))
	t.Cleanup(agent.Close)
	service := NewCoinCopilotService(
		repository.NewCoinCopilotRepository(db), settings,
		NewAgentProxy(agent.URL, "internal", NewLogger(10)),
		NewInternalTokenService("01234567890123456789012345678901"), NewLogger(10), "http://api:8080",
	)
	return db, service
}

func TestCoinCopilotStartIdempotencyOwnerScopeAndCancel(t *testing.T) {
	_, service := newCopilotServiceTest(t)
	input := CoinCopilotStartInput{Goal: "Compare my Roman coins", IdempotencyKey: "start-1"}
	first, reused, err := service.Start(7, input)
	if err != nil || reused {
		t.Fatalf("start run=%#v reused=%v err=%v", first, reused, err)
	}
	if first.MaxConcurrentTools != 3 {
		t.Fatalf("max concurrent tools=%d, want 3", first.MaxConcurrentTools)
	}
	second, reused, err := service.Start(7, input)
	if err != nil || !reused || second.ID != first.ID {
		t.Fatalf("idempotent replay run=%#v reused=%v err=%v", second, reused, err)
	}
	input.Goal = "Different request"
	if _, _, err := service.Start(7, input); !errors.Is(err, ErrCopilotIdempotencyConflict) {
		t.Fatalf("mismatched replay error=%v", err)
	}
	if _, _, err := service.Start(7, CoinCopilotStartInput{Goal: "Another run", IdempotencyKey: "start-other"}); !errors.Is(err, ErrCopilotCapacity) {
		t.Fatalf("one-active-run error=%v", err)
	}
	if _, err := service.GetRun(8, first.ID); !errors.Is(err, ErrCopilotNotFound) {
		t.Fatalf("foreign read error=%v", err)
	}
	cancelled, immediate, err := service.Cancel(7, first.ID)
	if err != nil || !immediate || cancelled.Status != models.CopilotRunCancelled {
		t.Fatalf("cancel run=%#v immediate=%v err=%v", cancelled, immediate, err)
	}
}

func TestCoinCopilotQueueBackpressure(t *testing.T) {
	_, service := newCopilotServiceTest(t)
	_ = service.settingsSvc.SetSetting(SettingCoinCopilotQueueDepth, "1")
	if _, _, err := service.Start(7, CoinCopilotStartInput{Goal: "First queued run", IdempotencyKey: "queue-first"}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := service.Start(8, CoinCopilotStartInput{Goal: "Second queued run", IdempotencyKey: "queue-second"}); !errors.Is(err, ErrCopilotQueueFull) {
		t.Fatalf("queue backpressure error=%v", err)
	}
}

func TestCoinCopilotConcurrentStartsEnforceOwnerCapacity(t *testing.T) {
	db, service := newCopilotServiceTest(t)
	type result struct {
		run *models.CoinCopilotRun
		err error
	}
	start := make(chan struct{})
	results := make(chan result, 2)
	for i := 0; i < 2; i++ {
		go func(i int) {
			<-start
			run, _, err := service.Start(7, CoinCopilotStartInput{
				Goal: fmt.Sprintf("Concurrent owner request %d", i), IdempotencyKey: fmt.Sprintf("owner-key-%d", i),
			})
			results <- result{run: run, err: err}
		}(i)
	}
	close(start)
	var accepted, rejected int
	for i := 0; i < 2; i++ {
		outcome := <-results
		switch {
		case outcome.err == nil:
			accepted++
		case errors.Is(outcome.err, ErrCopilotCapacity):
			rejected++
		default:
			t.Fatalf("unexpected concurrent owner result: run=%#v err=%v", outcome.run, outcome.err)
		}
	}
	if accepted != 1 || rejected != 1 {
		t.Fatalf("accepted=%d rejected=%d, want 1 each", accepted, rejected)
	}
	var count int64
	if err := db.Model(&models.CoinCopilotRun{}).Where("user_id = ?", 7).Count(&count).Error; err != nil || count != 1 {
		t.Fatalf("owner run count=%d err=%v", count, err)
	}
}

func TestCoinCopilotConcurrentStartsEnforceGlobalQueueDepth(t *testing.T) {
	db, service := newCopilotServiceTest(t)
	if err := service.settingsSvc.SetSetting(SettingCoinCopilotQueueDepth, "1"); err != nil {
		t.Fatal(err)
	}
	type result struct {
		run *models.CoinCopilotRun
		err error
	}
	start := make(chan struct{})
	results := make(chan result, 2)
	for i, userID := range []uint{7, 8} {
		go func(i int, userID uint) {
			<-start
			run, _, err := service.Start(userID, CoinCopilotStartInput{
				Goal: fmt.Sprintf("Concurrent queue request %d", i), IdempotencyKey: fmt.Sprintf("queue-key-%d", i),
			})
			results <- result{run: run, err: err}
		}(i, userID)
	}
	close(start)
	var accepted, rejected int
	for i := 0; i < 2; i++ {
		outcome := <-results
		switch {
		case outcome.err == nil:
			accepted++
		case errors.Is(outcome.err, ErrCopilotQueueFull):
			rejected++
		default:
			t.Fatalf("unexpected concurrent queue result: run=%#v err=%v", outcome.run, outcome.err)
		}
	}
	if accepted != 1 || rejected != 1 {
		t.Fatalf("accepted=%d rejected=%d, want 1 each", accepted, rejected)
	}
	var count int64
	if err := db.Model(&models.CoinCopilotRun{}).Where("status = ?", models.CopilotRunQueued).Count(&count).Error; err != nil || count != 1 {
		t.Fatalf("queued run count=%d err=%v", count, err)
	}
}

func TestCoinCopilotConcurrentStartReplayPreservesIdempotency(t *testing.T) {
	db, service := newCopilotServiceTest(t)
	type result struct {
		run    *models.CoinCopilotRun
		reused bool
		err    error
	}
	start := make(chan struct{})
	results := make(chan result, 2)
	for i := 0; i < 2; i++ {
		go func() {
			<-start
			run, reused, err := service.Start(7, CoinCopilotStartInput{
				Goal: "Same concurrent request", IdempotencyKey: "same-concurrent-key",
			})
			results <- result{run: run, reused: reused, err: err}
		}()
	}
	close(start)
	first := <-results
	second := <-results
	if first.err != nil || second.err != nil || first.run == nil || second.run == nil || first.run.ID != second.run.ID {
		t.Fatalf("concurrent replay results: first=%#v second=%#v", first, second)
	}
	if first.reused == second.reused {
		t.Fatalf("reused flags = %v,%v, want one original and one replay", first.reused, second.reused)
	}
	var count int64
	if err := db.Model(&models.CoinCopilotRun{}).Where("user_id = ?", 7).Count(&count).Error; err != nil || count != 1 {
		t.Fatalf("idempotent run count=%d err=%v", count, err)
	}
}

func TestCoinCopilotCapabilityFailsClosedForInvalidBudgets(t *testing.T) {
	_, service := newCopilotServiceTest(t)
	_ = service.settingsSvc.SetSetting(SettingCoinCopilotMaxToolCalls, "999")
	capability := service.Capability()
	if capability.Mode != "legacy" || capability.Reason == nil || *capability.Reason != "temporarily_unavailable" {
		t.Fatalf("capability = %#v", capability)
	}
	if _, _, err := service.Start(7, CoinCopilotStartInput{Goal: "Summary", IdempotencyKey: "invalid-budget"}); !errors.Is(err, ErrCopilotUnavailable) {
		t.Fatalf("start with invalid budgets error=%v", err)
	}
}

func TestCoinCopilotStartFailedPreflightCreatesNoRun(t *testing.T) {
	tests := []struct {
		name    string
		handler http.HandlerFunc
		timeout time.Duration
	}{
		{
			name: "binding failure",
			handler: func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(`{"supported":false}`))
			},
		},
		{
			name: "agent error",
			handler: func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "unavailable", http.StatusServiceUnavailable)
			},
		},
		{
			name: "malformed response",
			handler: func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(`{"supported":"yes"}`))
			},
		},
		{
			name:    "timeout",
			timeout: 20 * time.Millisecond,
			handler: func(w http.ResponseWriter, r *http.Request) {
				select {
				case <-r.Context().Done():
				case <-time.After(200 * time.Millisecond):
				}
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			db, service := newCopilotServiceTest(t)
			agent := httptest.NewServer(test.handler)
			defer agent.Close()
			proxy := NewAgentProxy(agent.URL, "internal", NewLogger(10))
			if test.timeout > 0 {
				proxy.requestClient = &http.Client{Timeout: test.timeout}
			}
			service.proxy = proxy

			capability := service.Capability()
			if capability.Mode != "legacy" || capability.Reason == nil || *capability.Reason != "model_tool_calling_unsupported" {
				t.Fatalf("capability = %#v", capability)
			}
			if _, _, err := service.Start(7, CoinCopilotStartInput{Goal: "Summary", IdempotencyKey: "failed-preflight"}); !errors.Is(err, ErrCopilotUnavailable) {
				t.Fatalf("start error = %v", err)
			}
			var runs, threads int64
			if err := db.Model(&models.CoinCopilotRun{}).Count(&runs).Error; err != nil {
				t.Fatal(err)
			}
			if err := db.Model(&models.CoinCopilotThread{}).Count(&threads).Error; err != nil {
				t.Fatal(err)
			}
			if runs != 0 || threads != 0 {
				t.Fatalf("failed preflight persisted rows: runs=%d threads=%d", runs, threads)
			}
		})
	}
}

func TestCoinCopilotResumeIdempotencyAndStaleVersion(t *testing.T) {
	db, service := newCopilotServiceTest(t)
	run, _, err := service.Start(7, CoinCopilotStartInput{Goal: "Compare bronzes", IdempotencyKey: "start-2"})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	deadline := now.Add(time.Hour)
	executionID := "cce_initial"
	if err := db.Model(&models.CoinCopilotRun{}).Where("id = ?", run.ID).Updates(map[string]any{
		"status": models.CopilotRunPaused, "execution_id": executionID, "checkpoint_version": 1,
		"paused_at": now, "resume_deadline": deadline,
	}).Error; err != nil {
		t.Fatal(err)
	}
	state := `{"schema_version":1,"messages":[{"role":"user","content":"Compare bronzes"}],"plan":[],"completed_tools":[],"pending_clarification":{"question":"Owned only?","input_type":"boolean","choices":[]},"next_action":"await_clarification","counters":{"iterations":1,"tool_calls":0,"input_tokens":1,"output_tokens":1}}`
	if err := db.Create(&models.CoinCopilotCheckpoint{
		RunID: run.ID, ThreadID: run.ThreadID, UserID: 7, ExecutionID: executionID,
		Version: 1, StateJSON: state, StateDigest: CopilotCheckpointDigest(state),
	}).Error; err != nil {
		t.Fatal(err)
	}
	resumed, reused, err := service.Resume(7, run.ID, CoinCopilotResumeInput{Answer: "Yes", ExpectedCheckpointVersion: 1, IdempotencyKey: "resume-1"})
	if err != nil || reused || resumed.Status != models.CopilotRunQueued || resumed.CheckpointVersion != 2 {
		t.Fatalf("resume run=%#v reused=%v err=%v", resumed, reused, err)
	}
	replay, reused, err := service.Resume(7, run.ID, CoinCopilotResumeInput{Answer: "Yes", ExpectedCheckpointVersion: 1, IdempotencyKey: "resume-1"})
	if err != nil || !reused || replay.ID != run.ID {
		t.Fatalf("resume replay run=%#v reused=%v err=%v", replay, reused, err)
	}
	if _, _, err := service.Resume(7, run.ID, CoinCopilotResumeInput{Answer: "No", ExpectedCheckpointVersion: 1, IdempotencyKey: "resume-1"}); !errors.Is(err, ErrCopilotIdempotencyConflict) {
		t.Fatalf("resume conflict error=%v", err)
	}
}

func TestCoinCopilotResumedExecutionUsesFreshTimeoutForCallbacks(t *testing.T) {
	db, service := newCopilotServiceTest(t)
	repo := repository.NewCoinCopilotRepository(db)
	oldStart := time.Now().UTC().Add(-10 * time.Minute)
	oldExecutionStart := oldStart
	deadline := time.Now().UTC().Add(time.Hour)
	thread := &models.CoinCopilotThread{ID: "cct_resume_timeout", UserID: 7, Title: "Resume timeout"}
	run := &models.CoinCopilotRun{
		ID: "ccr_resume_timeout", ThreadID: thread.ID, UserID: 7, Status: models.CopilotRunPaused,
		Goal: "Resume after a long pause", StartIdempotencyKeyHash: "resume-timeout-key", StartRequestFingerprint: "fingerprint",
		ExecutionID: "cce_old", CheckpointVersion: 1, MaxIterations: 8, MaxToolCalls: 12, MaxConcurrentTools: 1,
		HardTimeoutSeconds: 120, MaxPersistedToolResultBytes: 32768,
		StartedAt: &oldStart, ExecutionStartedAt: &oldExecutionStart, ResumeDeadline: &deadline,
	}
	if err := repo.CreateRun(thread, run); err != nil {
		t.Fatal(err)
	}
	state := `{"schema_version":1,"messages":[{"role":"user","content":"Resume after a long pause"}],"plan":[],"completed_tools":[],"pending_clarification":{"question":"Continue?","input_type":"boolean","choices":[]},"next_action":"await_clarification","counters":{"iterations":1,"tool_calls":0,"input_tokens":1,"output_tokens":1}}`
	if err := db.Create(&models.CoinCopilotCheckpoint{
		RunID: run.ID, ThreadID: run.ThreadID, UserID: 7, ExecutionID: run.ExecutionID,
		Version: 1, StateJSON: state, StateDigest: CopilotCheckpointDigest(state),
	}).Error; err != nil {
		t.Fatal(err)
	}
	resumed, _, err := service.Resume(7, run.ID, CoinCopilotResumeInput{
		Answer: "Yes", ExpectedCheckpointVersion: 1, IdempotencyKey: "resume-timeout",
	})
	if err != nil {
		t.Fatal(err)
	}
	resumeExecutionID := resumed.ExecutionID
	claimed, ok, err := repo.ClaimNextQueuedRun("worker", "cce_worker_generated")
	if err != nil || !ok {
		t.Fatalf("claim ok=%v err=%v", ok, err)
	}
	if claimed.ExecutionID != resumeExecutionID || claimed.ExecutionID == "cce_worker_generated" {
		t.Fatalf("claim execution ID = %q, want persisted resume ID %q", claimed.ExecutionID, resumeExecutionID)
	}
	if claimed.StartedAt == nil || !claimed.StartedAt.Equal(oldStart) {
		t.Fatalf("initial started_at changed: %#v", claimed.StartedAt)
	}
	if claimed.ExecutionStartedAt == nil || time.Since(*claimed.ExecutionStartedAt) > time.Second {
		t.Fatalf("execution_started_at was not refreshed: %#v", claimed.ExecutionStartedAt)
	}
	claims := &CopilotExecutionClaims{
		UserID: 7, RunID: run.ID, ExecutionID: claimed.ExecutionID, AllowedTools: []string{"get_coin"},
	}
	if err := service.AuthorizeToolCall(claims, "call_after_resume", "get_coin"); err != nil {
		t.Fatalf("resumed callback rejected against old run start: %v", err)
	}
}

func TestCoinCopilotToolLimitCountsPersistedAndInflightCallsOnce(t *testing.T) {
	db, service := newCopilotServiceTest(t)
	repo := repository.NewCoinCopilotRepository(db)
	now := time.Now().UTC()
	thread := &models.CoinCopilotThread{ID: "cct_tool_limit", UserID: 7, Title: "Tool limit"}
	run := &models.CoinCopilotRun{
		ID: "ccr_tool_limit", ThreadID: thread.ID, UserID: 7, Status: models.CopilotRunRunning,
		Goal: "Use tools", StartIdempotencyKeyHash: "tool-limit-key", StartRequestFingerprint: "fingerprint",
		ExecutionID: "cce_tool_limit", MaxIterations: 8, MaxToolCalls: 12, MaxConcurrentTools: 1,
		HardTimeoutSeconds: 120, MaxPersistedToolResultBytes: 32768,
		StartedAt: &now, ExecutionStartedAt: &now,
	}
	if err := repo.CreateRun(thread, run); err != nil {
		t.Fatal(err)
	}
	claims := &CopilotExecutionClaims{
		UserID: 7, RunID: run.ID, ExecutionID: run.ExecutionID, AllowedTools: []string{"get_coin"},
	}
	for i := 1; i <= 6; i++ {
		callID := fmt.Sprintf("call_%d", i)
		if err := service.AuthorizeToolCall(claims, callID, "get_coin"); err != nil {
			t.Fatalf("authorize call %d: %v", i, err)
		}
		service.FinishToolCall(run.ExecutionID, callID, true)
	}
	completed := make([]CopilotCompletedTool, 6)
	for i := range completed {
		completed[i] = CopilotCompletedTool{
			ToolCallID: fmt.Sprintf("call_%d", i+1), ToolName: "get_coin", Result: json.RawMessage(`{"ok":true}`),
		}
	}
	checkpoint := CopilotCheckpointState{
		SchemaVersion: 1, Messages: []CopilotMessage{{Role: "user", Content: run.Goal}},
		Plan: []CopilotPlanItem{}, CompletedTools: completed, NextAction: "continue",
		Counters: CopilotUsage{Iterations: 1, ToolCalls: 6},
	}
	payload, err := json.Marshal(checkpoint)
	if err != nil {
		t.Fatal(err)
	}
	if err := service.applyFrame(run, CopilotAgentFrame{
		SchemaVersion: 1, RunID: run.ID, ExecutionID: run.ExecutionID, FrameID: "checkpoint_1",
		Type: "checkpoint", Payload: payload,
	}); err != nil {
		t.Fatal(err)
	}
	for i := 7; i <= run.MaxToolCalls; i++ {
		callID := fmt.Sprintf("call_%d", i)
		if err := service.AuthorizeToolCall(claims, callID, "get_coin"); err != nil {
			t.Fatalf("authorize call %d: %v", i, err)
		}
		service.FinishToolCall(run.ExecutionID, callID, true)
		completed = append(completed, CopilotCompletedTool{
			ToolCallID: callID, ToolName: "get_coin", Result: json.RawMessage(`{"ok":true}`),
		})
	}
	checkpoint.CompletedTools = completed
	checkpoint.Counters.ToolCalls = run.MaxToolCalls
	payload, err = json.Marshal(checkpoint)
	if err != nil {
		t.Fatal(err)
	}
	if err := service.applyFrame(run, CopilotAgentFrame{
		SchemaVersion: 1, RunID: run.ID, ExecutionID: run.ExecutionID, FrameID: "checkpoint_2",
		Type: "checkpoint", Payload: payload,
	}); err != nil {
		t.Fatal(err)
	}
	stored, err := repo.GetRun(run.ID, run.UserID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.ToolCallCount != run.MaxToolCalls || stored.CheckpointVersion != 2 {
		t.Fatalf("persisted tool budget = %#v", stored)
	}
	if err := service.AuthorizeToolCall(claims, "call_13", "get_coin"); !errors.Is(err, ErrCopilotStateConflict) {
		t.Fatalf("limit+1 error = %v", err)
	}
}

func TestCoinCopilotToolConcurrencyAllowsThreeInflightCalls(t *testing.T) {
	db, service := newCopilotServiceTest(t)
	repo := repository.NewCoinCopilotRepository(db)
	now := time.Now().UTC()
	thread := &models.CoinCopilotThread{ID: "cct_tool_concurrency", UserID: 7, Title: "Tool concurrency"}
	run := &models.CoinCopilotRun{
		ID: "ccr_tool_concurrency", ThreadID: thread.ID, UserID: 7, Status: models.CopilotRunRunning,
		Goal: "Use independent tools", StartIdempotencyKeyHash: "tool-concurrency-key",
		StartRequestFingerprint: "fingerprint", ExecutionID: "cce_tool_concurrency",
		MaxIterations: 8, MaxToolCalls: 12, MaxConcurrentTools: 3,
		HardTimeoutSeconds: 120, MaxPersistedToolResultBytes: 32768,
		StartedAt: &now, ExecutionStartedAt: &now,
	}
	if err := repo.CreateRun(thread, run); err != nil {
		t.Fatal(err)
	}
	claims := &CopilotExecutionClaims{
		UserID: 7, RunID: run.ID, ExecutionID: run.ExecutionID, AllowedTools: []string{"get_coin"},
	}
	for i := 1; i <= 3; i++ {
		if err := service.AuthorizeToolCall(claims, fmt.Sprintf("call_%d", i), "get_coin"); err != nil {
			t.Fatalf("authorize concurrent call %d: %v", i, err)
		}
	}
	if err := service.AuthorizeToolCall(claims, "call_4", "get_coin"); !errors.Is(err, ErrCopilotStateConflict) {
		t.Fatalf("fourth concurrent call error=%v, want state conflict", err)
	}
	service.FinishToolCall(run.ExecutionID, "call_1", true)
	if err := service.AuthorizeToolCall(claims, "call_4", "get_coin"); err != nil {
		t.Fatalf("replacement call after completion: %v", err)
	}
}

func TestCoinCopilotRecoverExpireAndPrune(t *testing.T) {
	db, service := newCopilotServiceTest(t)
	repo := repository.NewCoinCopilotRepository(db)
	create := func(id string, status models.CopilotRunStatus, executionID string) *models.CoinCopilotRun {
		t.Helper()
		thread := &models.CoinCopilotThread{ID: "cct_" + id, UserID: 7, Title: id}
		run := &models.CoinCopilotRun{
			ID: "ccr_" + id, ThreadID: thread.ID, UserID: 7, Status: status, Goal: id,
			StartIdempotencyKeyHash: "key_" + id, StartRequestFingerprint: "fingerprint_" + id,
			ExecutionID: executionID, MaxIterations: 8, MaxToolCalls: 12, MaxConcurrentTools: 1,
			HardTimeoutSeconds: 120, MaxPersistedToolResultBytes: 32768,
		}
		if err := repo.CreateRun(thread, run); err != nil {
			t.Fatal(err)
		}
		return run
	}
	staleWithoutCheckpoint := create("stale_empty", models.CopilotRunRunning, "cce_stale_empty")
	staleWithCheckpoint := create("stale_checkpoint", models.CopilotRunRunning, "cce_stale_checkpoint")
	staleCancelRequested := create("stale_cancel_requested", models.CopilotRunCancelRequested, "cce_stale_cancel_requested")
	state := `{"schema_version":1,"messages":[],"plan":[],"completed_tools":[],"pending_clarification":null,"next_action":"continue","counters":{"iterations":0,"tool_calls":0,"input_tokens":0,"output_tokens":0}}`
	if _, err := repo.CommitCheckpoint(staleWithCheckpoint.ID, 7, staleWithCheckpoint.ExecutionID, state, CopilotCheckpointDigest(state)); err != nil {
		t.Fatal(err)
	}
	oldHeartbeat := time.Now().UTC().Add(-time.Minute)
	if err := db.Model(&models.CoinCopilotRun{}).
		Where("id IN ?", []string{staleWithoutCheckpoint.ID, staleWithCheckpoint.ID, staleCancelRequested.ID}).
		Update("heartbeat_at", oldHeartbeat).Error; err != nil {
		t.Fatal(err)
	}
	expired := create("expired", models.CopilotRunPaused, "cce_expired")
	pastDeadline := time.Now().UTC().Add(-time.Minute)
	if err := db.Model(&models.CoinCopilotRun{}).Where("id = ?", expired.ID).
		Updates(map[string]any{"resume_deadline": pastDeadline, "checkpoint_version": 1}).Error; err != nil {
		t.Fatal(err)
	}
	oldTerminal := create("old_terminal", models.CopilotRunRunning, "cce_old_terminal")
	if _, err := repo.AppendEvent(oldTerminal.ID, 7, oldTerminal.ExecutionID, models.CopilotEventPlanUpdated, `{"plan":[]}`); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.CommitCheckpoint(oldTerminal.ID, 7, oldTerminal.ExecutionID, state, CopilotCheckpointDigest(state)); err != nil {
		t.Fatal(err)
	}
	if won, _, err := repo.TransitionWithEvent(
		oldTerminal.ID, 7, oldTerminal.ExecutionID, []models.CopilotRunStatus{models.CopilotRunRunning},
		models.CopilotRunCompleted, map[string]any{"final_answer": "done"},
		models.CopilotEventRunCompleted, `{"answer":"done"}`,
	); err != nil || !won {
		t.Fatalf("terminal transition won=%v err=%v", won, err)
	}
	if err := db.Model(&models.CoinCopilotRun{}).Where("id = ?", oldTerminal.ID).
		Update("completed_at", time.Now().UTC().Add(-40*24*time.Hour)).Error; err != nil {
		t.Fatal(err)
	}
	if err := service.RecoverAndPrune(); err != nil {
		t.Fatal(err)
	}
	assertStatus := func(runID string, want models.CopilotRunStatus) {
		t.Helper()
		run, err := repo.GetRun(runID, 7)
		if err != nil {
			t.Fatal(err)
		}
		if run.Status != want {
			t.Fatalf("%s status = %q, want %q", runID, run.Status, want)
		}
	}
	assertStatus(staleWithoutCheckpoint.ID, models.CopilotRunFailed)
	assertStatus(staleWithCheckpoint.ID, models.CopilotRunPaused)
	assertStatus(staleCancelRequested.ID, models.CopilotRunCancelled)
	assertStatus(expired.ID, models.CopilotRunFailed)
	if err := service.RecoverAndPrune(); err != nil {
		t.Fatal(err)
	}
	var cancelledEvents int64
	if err := db.Model(&models.CoinCopilotEvent{}).
		Where("run_id = ? AND type = ?", staleCancelRequested.ID, models.CopilotEventRunCancelled).
		Count(&cancelledEvents).Error; err != nil {
		t.Fatal(err)
	}
	if cancelledEvents != 1 {
		t.Fatalf("cancelled event count = %d, want 1", cancelledEvents)
	}
	if err := service.DeleteThread(7, staleCancelRequested.ThreadID); err != nil {
		t.Fatalf("delete recovered cancelled thread: %v", err)
	}
	var eventCount, checkpointCount int64
	if err := db.Model(&models.CoinCopilotEvent{}).Where("run_id = ?", oldTerminal.ID).Count(&eventCount).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&models.CoinCopilotCheckpoint{}).Where("run_id = ?", oldTerminal.ID).Count(&checkpointCount).Error; err != nil {
		t.Fatal(err)
	}
	if eventCount != 0 || checkpointCount != 0 {
		t.Fatalf("retained old data: events=%d checkpoints=%d", eventCount, checkpointCount)
	}
}

func TestCoinCopilotConcurrentRecoverySettlesCancellationOnce(t *testing.T) {
	db, service := newCopilotServiceTest(t)
	repo := repository.NewCoinCopilotRepository(db)
	thread := &models.CoinCopilotThread{ID: "cct_recovery_race", UserID: 17, Title: "Recovery race"}
	oldHeartbeat := time.Now().UTC().Add(-time.Minute)
	run := &models.CoinCopilotRun{
		ID: "ccr_recovery_race", ThreadID: thread.ID, UserID: thread.UserID, Status: models.CopilotRunCancelRequested,
		Goal: "Cancel me", StartIdempotencyKeyHash: "recovery-race-key", StartRequestFingerprint: "fingerprint",
		ExecutionID: "cce_recovery_race", HeartbeatAt: &oldHeartbeat,
		MaxIterations: 8, MaxToolCalls: 12, MaxConcurrentTools: 1,
		HardTimeoutSeconds: 120, MaxPersistedToolResultBytes: 32768,
	}
	if err := repo.CreateRun(thread, run); err != nil {
		t.Fatal(err)
	}
	start := make(chan struct{})
	errs := make(chan error, 2)
	for i := 0; i < 2; i++ {
		go func() {
			<-start
			errs <- service.RecoverAndPrune()
		}()
	}
	close(start)
	for i := 0; i < 2; i++ {
		if err := <-errs; err != nil {
			t.Fatalf("concurrent recovery error: %v", err)
		}
	}
	stored, err := repo.GetRun(run.ID, run.UserID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Status != models.CopilotRunCancelled {
		t.Fatalf("recovered status = %q, want cancelled", stored.Status)
	}
	var events int64
	if err := db.Model(&models.CoinCopilotEvent{}).
		Where("run_id = ? AND type = ?", run.ID, models.CopilotEventRunCancelled).
		Count(&events).Error; err != nil {
		t.Fatal(err)
	}
	if events != 1 {
		t.Fatalf("run_cancelled events = %d, want 1", events)
	}
}
