package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"gorm.io/gorm"
)

func TestCoinCopilotInitialExecutionIncludesBoundedPublicThreadHistory(t *testing.T) {
	db, service := newCopilotServiceTest(t)
	thread := &models.CoinCopilotThread{ID: "cct_public_history", UserID: 7, Title: "History"}
	if err := db.Create(thread).Error; err != nil {
		t.Fatal(err)
	}
	base := time.Now().UTC().Add(-time.Hour)
	for i := 0; i < CoinCopilotHistoryRunLimit+2; i++ {
		run := &models.CoinCopilotRun{
			ID: fmt.Sprintf("ccr_prior_%02d", i), ThreadID: thread.ID, UserID: 7, Status: models.CopilotRunCompleted,
			Goal:                    fmt.Sprintf("prior goal %02d %s", i, strings.Repeat("g", CoinCopilotHistoryMessageMaxBytes)),
			FinalAnswer:             fmt.Sprintf("prior answer %02d %s", i, strings.Repeat("a", CoinCopilotHistoryMessageMaxBytes)),
			AppContextJSON:          `{"private":"must-not-leak"}`,
			StartIdempotencyKeyHash: fmt.Sprintf("history-key-%02d", i), StartRequestFingerprint: "fingerprint",
			MaxIterations: 8, MaxToolCalls: 12, MaxConcurrentTools: 1, HardTimeoutSeconds: 120,
			MaxPersistedToolResultBytes: 32768, CreatedAt: base.Add(time.Duration(i) * time.Minute),
		}
		if err := db.Create(run).Error; err != nil {
			t.Fatal(err)
		}
	}
	current := &models.CoinCopilotRun{
		ID: "ccr_current_history", ThreadID: thread.ID, UserID: 7, Status: models.CopilotRunRunning,
		Goal: "current goal", StartIdempotencyKeyHash: "current-history-key", StartRequestFingerprint: "fingerprint",
		ExecutionID: "cce_current_history", MaxIterations: 8, MaxToolCalls: 12, MaxConcurrentTools: 1,
		HardTimeoutSeconds: 120, MaxPersistedToolResultBytes: 32768,
	}
	if err := db.Create(current).Error; err != nil {
		t.Fatal(err)
	}
	request, err := service.executionRequest(current, LLMConfig{}, "token", CopilotLimitsProxy{})
	if err != nil {
		t.Fatal(err)
	}
	if got := request.Messages[len(request.Messages)-1]; got.Role != "user" || got.Content != current.Goal {
		t.Fatalf("last message = %#v", got)
	}
	history := request.Messages[:len(request.Messages)-1]
	if len(history) == 0 || len(history) > CoinCopilotHistoryRunLimit*2 {
		t.Fatalf("history message count = %d", len(history))
	}
	total := 0
	encoded, _ := json.Marshal(history)
	if strings.Contains(string(encoded), "must-not-leak") || strings.Contains(string(encoded), "prior goal 00") {
		t.Fatalf("history leaked private or out-of-bound content: %s", encoded)
	}
	for _, message := range history {
		total += len(message.Content)
		if len(message.Content) > CoinCopilotHistoryMessageMaxBytes {
			t.Fatalf("history message exceeded bound: %d", len(message.Content))
		}
	}
	if total > CoinCopilotHistoryMaxBytes {
		t.Fatalf("history bytes = %d", total)
	}
	if !strings.Contains(string(encoded), "prior goal 11") {
		t.Fatalf("most recent history missing: %s", encoded)
	}
}

func TestCoinCopilotExecutionIncludesOptionalBoundedCollectorContext(t *testing.T) {
	_, service := newCopilotServiceTest(t)
	run := &models.CoinCopilotRun{
		ID: "ccr_collector_context", ThreadID: "cct_collector_context", UserID: 7,
		Status: models.CopilotRunRunning, Goal: "Act as my collection curator",
		ExecutionID: "cce_collector_context", MaxIterations: 8, MaxToolCalls: 12,
		MaxConcurrentTools: 1, HardTimeoutSeconds: 120, MaxPersistedToolResultBytes: 32768,
	}

	withoutProfile, err := service.executionRequest(run, LLMConfig{}, "token", CopilotLimitsProxy{})
	if err != nil {
		t.Fatal(err)
	}
	if withoutProfile.CollectorContext != nil {
		t.Fatalf("absent profile should be omitted: %#v", withoutProfile.CollectorContext)
	}

	minimum, maximum := 100.0, 500.0
	currency := "USD"
	if _, err := service.collectorProfileSvc.Replace(7, CollectorProfileInput{
		BudgetMin: &minimum, BudgetMax: &maximum, Currency: &currency,
		PreferredPeriods: []string{"Roman Imperial"}, PreferredCategories: []string{"Roman"},
		PreferredDealers: []string{"VCoins"},
		CollectingGoals:  []string{"Build a representative Probus mint set"},
	}); err != nil {
		t.Fatal(err)
	}

	request, err := service.executionRequest(run, LLMConfig{}, "token", CopilotLimitsProxy{})
	if err != nil {
		t.Fatal(err)
	}
	if request.CollectorContext == nil ||
		request.CollectorContext.Currency == nil ||
		*request.CollectorContext.Currency != "USD" ||
		len(request.CollectorContext.CollectingGoals) != 1 {
		t.Fatalf("collector context = %#v", request.CollectorContext)
	}
	encoded, err := json.Marshal(request.CollectorContext)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"user_id", "owner_id", "token", "secret", "action_url"} {
		if strings.Contains(string(encoded), forbidden) {
			t.Fatalf("collector context leaked forbidden field %q: %s", forbidden, encoded)
		}
	}
}

func TestCoinCopilotWorkerPersistsCheckpointAndSingleTerminalEvent(t *testing.T) {
	db, service := newCopilotServiceTest(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/copilot/execute" || r.Header.Get("X-Internal-Service-Token") != "internal" {
			http.Error(w, "bad request", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintln(w, `event: checkpoint`)
		fmt.Fprintln(w, `data: {"schema_version":1,"run_id":"ccr_worker","execution_id":"cce_worker","frame_id":"frm_1","type":"checkpoint","payload":{"schema_version":1,"messages":[{"role":"user","content":"Summarize"}],"plan":[],"completed_tools":[],"pending_clarification":null,"next_action":"finish","counters":{"iterations":1,"tool_calls":0,"input_tokens":10,"output_tokens":5}}}`)
		fmt.Fprintln(w)
		fmt.Fprintln(w, `event: completed`)
		fmt.Fprintln(w, `data: {"schema_version":1,"run_id":"ccr_worker","execution_id":"cce_worker","frame_id":"frm_2","type":"completed","payload":{"answer":"Done.","usage":{"iterations":1,"tool_calls":0,"input_tokens":10,"output_tokens":5}}}`)
		fmt.Fprintln(w)
	}))
	defer server.Close()
	service.proxy = NewAgentProxy(server.URL, "internal", NewLogger(10))
	repo := repository.NewCoinCopilotRepository(db)
	thread := &models.CoinCopilotThread{ID: "cct_worker", UserID: 7, Title: "Worker"}
	run := &models.CoinCopilotRun{
		ID: "ccr_worker", ThreadID: thread.ID, UserID: 7, Status: models.CopilotRunRunning,
		Goal: "Summarize", StartIdempotencyKeyHash: "worker-key", StartRequestFingerprint: "fingerprint",
		ExecutionID: "cce_worker", ExecutionAttempt: 1, MaxIterations: 8, MaxToolCalls: 12,
		MaxConcurrentTools: 1, HardTimeoutSeconds: 120, MaxPersistedToolResultBytes: 32768,
	}
	if err := repo.CreateRun(thread, run); err != nil {
		t.Fatal(err)
	}
	service.runExecution(context.Background(), run)
	stored, err := repo.GetRun(run.ID, 7)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Status != models.CopilotRunCompleted || stored.FinalAnswer != "Done." || stored.CheckpointVersion != 1 {
		t.Fatalf("stored run = %#v", stored)
	}
	events, err := repo.ListEventsSince(run.ID, 7, 0)
	if err != nil {
		t.Fatal(err)
	}
	terminal := 0
	for _, event := range events {
		if event.Type == models.CopilotEventRunCompleted || event.Type == models.CopilotEventRunFailed || event.Type == models.CopilotEventRunCancelled {
			terminal++
		}
	}
	if terminal != 1 {
		t.Fatalf("terminal events = %d, events=%#v", terminal, events)
	}
}

func TestCoinCopilotWorkerPublishesValidatedSpecialistProjection(t *testing.T) {
	db, service := newCopilotServiceTest(t)
	repo := repository.NewCoinCopilotRepository(db)
	thread := &models.CoinCopilotThread{ID: "cct_specialist", UserID: 7, Title: "Specialist"}
	run := &models.CoinCopilotRun{
		ID: "ccr_specialist", ThreadID: thread.ID, UserID: thread.UserID, Status: models.CopilotRunRunning,
		Goal: "Find market examples", StartIdempotencyKeyHash: "specialist-key", StartRequestFingerprint: "fingerprint",
		ExecutionID: "cce_specialist", ExecutionAttempt: 1, MaxIterations: 8, MaxToolCalls: 12,
		MaxConcurrentTools: 1, HardTimeoutSeconds: 120, MaxPersistedToolResultBytes: 32768,
	}
	if err := repo.CreateRun(thread, run); err != nil {
		t.Fatal(err)
	}
	result, err := loadCoinCopilotFixture[json.RawMessage](
		t,
		filepath.Join("specialists", "market_search_complete.json"),
	)
	if err != nil {
		t.Fatal(err)
	}
	payload, _ := json.Marshal(map[string]any{
		"tool_call_id": "call_market", "tool_name": "market_search", "step_id": "step_market",
		"status": "succeeded", "duration_ms": 25, "result_summary": "One market result.", "result": result,
	})
	frame := CopilotAgentFrame{
		SchemaVersion: 1, RunID: run.ID, ExecutionID: run.ExecutionID,
		FrameID: "frm_market", Type: "tool_completed", Payload: payload,
	}
	if err := service.applyFrame(run, frame); err != nil {
		t.Fatal(err)
	}
	events, err := repo.ListEventsSince(run.ID, run.UserID, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].Type != models.CopilotEventToolCompleted {
		t.Fatalf("events = %#v", events)
	}
	var public struct {
		SpecialistResult map[string]any `json:"specialistResult"`
	}
	if err := json.Unmarshal([]byte(events[0].PayloadJSON), &public); err != nil {
		t.Fatal(err)
	}
	items, _ := public.SpecialistResult["items"].([]any)
	if public.SpecialistResult["capability"] != "market_search" || len(items) != 1 {
		t.Fatalf("specialist projection = %#v", public.SpecialistResult)
	}
	item, _ := items[0].(map[string]any)
	if item["sourceUrl"] != fixtureSourceURL(t, result) {
		t.Fatalf("specialist item = %#v", item)
	}
	logs := service.logger.GetLogs(10)
	if len(logs) != 1 {
		t.Fatalf("specialist logs=%#v", logs)
	}
	message := logs[0].Message
	for _, field := range []string{
		"run_id=ccr_specialist",
		"execution_id=cce_specialist",
		"capability=market_search",
		"aggregate_outcome=complete",
		"duration_ms=25",
		"item_count=1",
		"truncated=false",
	} {
		if !strings.Contains(message, field) {
			t.Fatalf("specialist log missing %q: %s", field, message)
		}
	}
	for _, forbidden := range []string{
		fixtureSourceURL(t, result),
		"Find market examples",
		"credential",
		"prompt",
	} {
		if strings.Contains(message, forbidden) {
			t.Fatalf("specialist log leaked %q: %s", forbidden, message)
		}
	}
}

func TestCoinCopilotWorkerDoesNotLogSpecialistCompletionBeforePersistence(t *testing.T) {
	db, service := newCopilotServiceTest(t)
	result, err := loadCoinCopilotFixture[json.RawMessage](
		t,
		filepath.Join("specialists", "market_search_complete.json"),
	)
	if err != nil {
		t.Fatal(err)
	}
	payload, _ := json.Marshal(map[string]any{
		"tool_call_id": "call_market", "tool_name": "market_search", "step_id": "step_market",
		"status": "succeeded", "duration_ms": 25, "result_summary": "One market result.", "result": result,
	})
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatal(err)
	}
	run := &models.CoinCopilotRun{
		ID: "ccr_failed_persistence", UserID: 7, ExecutionID: "cce_failed_persistence",
		MaxPersistedToolResultBytes: 32768,
	}
	frame := CopilotAgentFrame{
		SchemaVersion: 1, RunID: run.ID, ExecutionID: run.ExecutionID,
		FrameID: "frm_market", Type: "tool_completed", Payload: payload,
	}
	if err := service.applyFrame(run, frame); err == nil {
		t.Fatal("applyFrame succeeded after the event store was closed")
	}
	if logs := service.logger.GetLogs(10); len(logs) != 0 {
		t.Fatalf("specialist completion was logged before persistence: %#v", logs)
	}
}

func TestCoinCopilotWorkerRejectsDuplicateFrames(t *testing.T) {
	db, service := newCopilotServiceTest(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		frame := `{"schema_version":1,"run_id":"ccr_duplicate","execution_id":"cce_duplicate","frame_id":"frm_same","type":"plan_updated","payload":{"plan":[]}}`
		fmt.Fprintf(w, "data: %s\n\ndata: %s\n\n", frame, frame)
	}))
	defer server.Close()
	service.proxy = NewAgentProxy(server.URL, "internal", NewLogger(10))
	repo := repository.NewCoinCopilotRepository(db)
	thread := &models.CoinCopilotThread{ID: "cct_duplicate", UserID: 7, Title: "Duplicate"}
	run := &models.CoinCopilotRun{
		ID: "ccr_duplicate", ThreadID: thread.ID, UserID: 7, Status: models.CopilotRunRunning,
		Goal: "Summarize", StartIdempotencyKeyHash: "duplicate-key", StartRequestFingerprint: "fingerprint",
		ExecutionID: "cce_duplicate", ExecutionAttempt: 1, MaxIterations: 8, MaxToolCalls: 12,
		MaxConcurrentTools: 1, HardTimeoutSeconds: 120, MaxPersistedToolResultBytes: 32768,
	}

	if err := repo.CreateRun(thread, run); err != nil {
		t.Fatal(err)
	}
	service.runExecution(context.Background(), run)
	stored, err := repo.GetRun(run.ID, 7)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Status != models.CopilotRunFailed || stored.FailureCode != "invalid_agent_frame" {
		t.Fatalf("duplicate frame run = %#v", stored)
	}
}

func TestCoinCopilotWorkerRejectsReplayedSpecialistCallID(t *testing.T) {
	db, service := newCopilotServiceTest(t)
	result, err := loadCoinCopilotFixture[json.RawMessage](
		t,
		filepath.Join("specialists", "market_search_complete.json"),
	)
	if err != nil {
		t.Fatal(err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		for index := 1; index <= 2; index++ {
			fmt.Fprintf(
				w,
				"data: {\"schema_version\":1,\"run_id\":\"ccr_replayed_call\",\"execution_id\":\"cce_replayed_call\",\"frame_id\":\"frm_%d\",\"type\":\"tool_completed\",\"payload\":{\"tool_call_id\":\"call_market\",\"tool_name\":\"market_search\",\"step_id\":\"step_market\",\"status\":\"succeeded\",\"duration_ms\":1,\"result_summary\":\"Market result.\",\"result\":%s}}\n\n",
				index,
				resultJSON,
			)
		}
	}))
	defer server.Close()
	service.proxy = NewAgentProxy(server.URL, "internal", NewLogger(10))
	repo := repository.NewCoinCopilotRepository(db)
	thread := &models.CoinCopilotThread{ID: "cct_replayed_call", UserID: 7, Title: "Replay"}
	run := &models.CoinCopilotRun{
		ID: "ccr_replayed_call", ThreadID: thread.ID, UserID: thread.UserID, Status: models.CopilotRunRunning,
		Goal: "Find market examples", StartIdempotencyKeyHash: "replayed-call-key", StartRequestFingerprint: "fingerprint",
		ExecutionID: "cce_replayed_call", ExecutionAttempt: 1, MaxIterations: 8, MaxToolCalls: 12,
		MaxConcurrentTools: 3, HardTimeoutSeconds: 120, MaxPersistedToolResultBytes: 32768,
	}
	if err := repo.CreateRun(thread, run); err != nil {
		t.Fatal(err)
	}

	service.runExecution(context.Background(), run)

	stored, err := repo.GetRun(run.ID, run.UserID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Status != models.CopilotRunFailed || stored.FailureCode != "invalid_agent_frame" {
		t.Fatalf("replayed call run=%#v", stored)
	}
	events, err := repo.ListEventsSince(run.ID, run.UserID, 0)
	if err != nil {
		t.Fatal(err)
	}
	completions := 0
	for _, event := range events {
		if event.Type == models.CopilotEventToolCompleted {
			completions++
		}
	}
	if completions != 1 {
		t.Fatalf("tool completion events=%d, want 1", completions)
	}
}

func TestCoinCopilotWorkerHeartbeatsAndLeavesShutdownForRecovery(t *testing.T) {
	db, service := newCopilotServiceTest(t)
	oldInterval := copilotHeartbeatInterval
	copilotHeartbeatInterval = 10 * time.Millisecond
	t.Cleanup(func() { copilotHeartbeatInterval = oldInterval })
	requestStarted := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		w.(http.Flusher).Flush()
		close(requestStarted)
		<-r.Context().Done()
	}))
	defer server.Close()
	service.proxy = NewAgentProxy(server.URL, "internal", NewLogger(10))
	repo := repository.NewCoinCopilotRepository(db)
	thread := &models.CoinCopilotThread{ID: "cct_shutdown", UserID: 7, Title: "Shutdown"}
	run := &models.CoinCopilotRun{
		ID: "ccr_shutdown", ThreadID: thread.ID, UserID: 7, Status: models.CopilotRunRunning,
		Goal: "Summarize", StartIdempotencyKeyHash: "shutdown-key", StartRequestFingerprint: "fingerprint",
		ExecutionID: "cce_shutdown", ExecutionAttempt: 1, MaxIterations: 8, MaxToolCalls: 12,
		MaxConcurrentTools: 1, HardTimeoutSeconds: 120, MaxPersistedToolResultBytes: 32768,
	}
	if err := repo.CreateRun(thread, run); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		service.runExecution(ctx, run)
		close(done)
	}()
	<-requestStarted
	time.Sleep(35 * time.Millisecond)
	beforeCancel, err := repo.GetRun(run.ID, 7)
	if err != nil {
		t.Fatal(err)
	}
	if beforeCancel.HeartbeatAt == nil {
		t.Fatal("worker did not persist a heartbeat")
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("worker did not stop after shutdown")
	}
	stored, err := repo.GetRun(run.ID, 7)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Status != models.CopilotRunRunning {
		t.Fatalf("shutdown status = %q, want running for stale recovery", stored.Status)
	}
}

func TestCoinCopilotExecutionTokenTTLUsesRemainingBudgetAndAbsoluteCap(t *testing.T) {
	now := time.Date(2026, 9, 17, 20, 0, 0, 0, time.UTC)
	ctx, cancel := context.WithDeadline(context.Background(), now.Add(100*time.Second))
	defer cancel()
	if got := coinCopilotExecutionTokenTTL(ctx, now); got != 130*time.Second {
		t.Fatalf("remaining-budget TTL = %v, want 130s", got)
	}
	ctx, cancel = context.WithDeadline(context.Background(), now.Add(200*time.Second))
	defer cancel()
	if got := coinCopilotExecutionTokenTTL(ctx, now); got != 180*time.Second {
		t.Fatalf("capped TTL = %v, want 180s", got)
	}
}

func TestCoinCopilotWorkerStartupRecoversStaleCancellationAndReleasesCapacity(t *testing.T) {
	db, service := newCopilotServiceTest(t)
	repo := repository.NewCoinCopilotRepository(db)
	thread := &models.CoinCopilotThread{ID: "cct_startup_cancel", UserID: 23, Title: "Startup cancel"}
	oldHeartbeat := time.Now().UTC().Add(-time.Minute)
	run := &models.CoinCopilotRun{
		ID: "ccr_startup_cancel", ThreadID: thread.ID, UserID: thread.UserID, Status: models.CopilotRunCancelRequested,
		Goal: "Cancel at startup", StartIdempotencyKeyHash: "startup-cancel-key", StartRequestFingerprint: "fingerprint",
		ExecutionID: "cce_startup_cancel", HeartbeatAt: &oldHeartbeat,
		MaxIterations: 8, MaxToolCalls: 12, MaxConcurrentTools: 1,
		HardTimeoutSeconds: 120, MaxPersistedToolResultBytes: 32768,
	}
	if err := repo.CreateRun(thread, run); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	service.StartWorkers(ctx)
	stored, err := repo.GetRun(run.ID, run.UserID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Status != models.CopilotRunCancelled {
		t.Fatalf("startup recovery status = %q, want cancelled", stored.Status)
	}
	next, _, err := service.Start(run.UserID, CoinCopilotStartInput{
		Goal: "Capacity after cancellation recovery", IdempotencyKey: "after-startup-recovery",
	})
	if err != nil || next == nil {
		t.Fatalf("start after recovery run=%#v err=%v", next, err)
	}
}

func TestFeature362ResumeRejectsChangedDurableWorkerBindingWithoutExecution(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*gorm.DB, *models.CoinCopilotRun, *models.CoinCopilotCheckpoint)
	}{
		{"checkpoint version", func(db *gorm.DB, run *models.CoinCopilotRun, _ *models.CoinCopilotCheckpoint) {
			_ = db.Model(run).Update("checkpoint_version", run.CheckpointVersion+1).Error
		}},
		{"current execution", func(db *gorm.DB, run *models.CoinCopilotRun, _ *models.CoinCopilotCheckpoint) {
			_ = db.Model(run).Update("execution_id", "cce_stale").Error
		}},
		{"saved checkpoint", func(db *gorm.DB, _ *models.CoinCopilotRun, checkpoint *models.CoinCopilotCheckpoint) {
			_ = db.Model(checkpoint).Update("state_json", `{"tampered":true}`).Error
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			db, service := newCopilotServiceTest(t)
			if err := db.AutoMigrate(&models.DeepIdentificationJob{}, &models.CoinCopilotDeepHandoff{}); err != nil {
				t.Fatal(err)
			}
			run, _, err := service.Start(7, CoinCopilotStartInput{
				Goal: "Analyze this coin", AppContext: map[string]any{"route": "/coins/42", "activeCoinId": float64(42)},
				IdempotencyKey: "feature362-start",
			})
			if err != nil {
				t.Fatal(err)
			}
			now := time.Now().UTC()
			if err := db.Model(run).Updates(map[string]any{
				"status": models.CopilotRunPaused, "execution_id": "cce_paused",
				"checkpoint_version": 1, "resume_deadline": now.Add(time.Hour),
			}).Error; err != nil {
				t.Fatal(err)
			}
			state := `{"schema_version":1,"messages":[],"plan":[],"completed_tools":[],"pending_clarification":{"question":"Continue?","input_type":"boolean","choices":[]},"next_action":"await_clarification","counters":{"iterations":1,"tool_calls":0,"input_tokens":0,"output_tokens":0}}`
			if err := db.Create(&models.CoinCopilotCheckpoint{
				RunID: run.ID, ThreadID: run.ThreadID, UserID: 7, ExecutionID: "cce_paused",
				Version: 1, StateJSON: state, StateDigest: CopilotCheckpointDigest(state),
			}).Error; err != nil {
				t.Fatal(err)
			}
			input := CoinCopilotResumeInput{Answer: "Yes", ExpectedCheckpointVersion: 1, IdempotencyKey: "feature362-resume"}
			resumed, reused, err := service.Resume(7, run.ID, input)
			if err != nil || reused {
				t.Fatalf("resume run=%+v reused=%v err=%v", resumed, reused, err)
			}
			checkpoint, err := service.repo.GetLatestCheckpoint(run.ID, 7)
			if err != nil {
				t.Fatal(err)
			}
			test.mutate(db, resumed, checkpoint)
			if _, _, err := service.Resume(7, run.ID, input); !errors.Is(err, ErrCopilotIdempotencyConflict) {
				t.Fatalf("changed replay error=%v, want idempotency conflict", err)
			}
			for name, model := range map[string]any{
				"handoffs": &models.CoinCopilotDeepHandoff{},
				"jobs":     &models.DeepIdentificationJob{},
			} {
				var count int64
				if err := db.Model(model).Count(&count).Error; err != nil || count != 0 {
					t.Fatalf("%s count=%d err=%v", name, count, err)
				}
			}
		})
	}
}

// fixtureSourceURL reads one field from a raw fixture so tests do not need a
// Go-side copy of the specialist schema.
func fixtureSourceURL(t *testing.T, raw json.RawMessage) string {
	t.Helper()
	var decoded struct {
		Items []struct {
			SourceURL string `json:"source_url"`
		} `json:"items"`
	}
	if err := json.Unmarshal(raw, &decoded); err != nil || len(decoded.Items) == 0 {
		t.Fatalf("decode fixture source url: %v", err)
	}
	return decoded.Items[0].SourceURL
}
