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
	result, err := loadCoinCopilotFixture[CopilotSpecialistResult](
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
		SpecialistResult CopilotSpecialistPublicResult `json:"specialistResult"`
	}
	if err := json.Unmarshal([]byte(events[0].PayloadJSON), &public); err != nil {
		t.Fatal(err)
	}
	if public.SpecialistResult.Capability != "market_search" ||
		len(public.SpecialistResult.Items) != 1 ||
		public.SpecialistResult.Items[0].SourceURL != result.Items[0].SourceURL {
		t.Fatalf("specialist projection = %#v", public.SpecialistResult)
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
		"provider_id=cng_dealer_search",
		"provider_outcome=success",
		"aggregate_outcome=complete",
		"duration_ms=25",
		"item_count=1",
		"original_bytes=",
		"persisted_bytes=",
		"truncated=false",
		"digest=",
	} {
		if !strings.Contains(message, field) {
			t.Fatalf("specialist log missing %q: %s", field, message)
		}
	}
	for _, forbidden := range []string{
		result.Items[0].SourceURL,
		result.Items[0].Title,
		"Find market examples",
		"credential",
		"prompt",
	} {
		if strings.Contains(message, forbidden) {
			t.Fatalf("specialist log leaked %q: %s", forbidden, message)
		}
	}
}

func TestCoinCopilotWorkerRejectsTamperedSpecialistCheckpointMetadata(t *testing.T) {
	mutations := map[string]func(*CopilotCompletedTool){
		"digest":          func(tool *CopilotCompletedTool) { tool.ResultDigest = strings.Repeat("0", 64) },
		"original bytes":  func(tool *CopilotCompletedTool) { tool.OriginalBytes++ },
		"persisted bytes": func(tool *CopilotCompletedTool) { tool.PersistedBytes++ },
		"truncated":       func(tool *CopilotCompletedTool) { tool.Truncated = true },
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			db, service := newCopilotServiceTest(t)
			repo := repository.NewCoinCopilotRepository(db)
			thread := &models.CoinCopilotThread{ID: "cct_metadata_" + strings.ReplaceAll(name, " ", "_"), UserID: 7, Title: "Metadata"}
			run := &models.CoinCopilotRun{
				ID: "ccr_metadata_" + strings.ReplaceAll(name, " ", "_"), ThreadID: thread.ID, UserID: thread.UserID,
				Status: models.CopilotRunRunning, Goal: "Find market examples",
				StartIdempotencyKeyHash: "metadata-key-" + name, StartRequestFingerprint: "fingerprint",
				ExecutionID: "cce_metadata", ExecutionAttempt: 1, MaxIterations: 8, MaxToolCalls: 12,
				MaxConcurrentTools: 3, HardTimeoutSeconds: 120, MaxPersistedToolResultBytes: 32768,
			}
			if err := repo.CreateRun(thread, run); err != nil {
				t.Fatal(err)
			}
			result, err := loadCoinCopilotFixture[CopilotSpecialistResult](
				t,
				filepath.Join("specialists", "market_search_complete.json"),
			)
			if err != nil {
				t.Fatal(err)
			}
			raw, err := json.Marshal(result)
			if err != nil {
				t.Fatal(err)
			}
			bounded, originalBytes, truncated, digest, err := SanitizeCopilotJSON(raw, run.MaxPersistedToolResultBytes)
			if err != nil {
				t.Fatal(err)
			}
			tool := CopilotCompletedTool{
				ToolCallID: "call_market", ToolName: "market_search", ResultDigest: digest,
				Result: bounded, OriginalBytes: originalBytes, PersistedBytes: len(bounded), Truncated: truncated,
			}
			mutate(&tool)
			state := CopilotCheckpointState{
				SchemaVersion:  1,
				Messages:       []CopilotMessage{{Role: "user", Content: run.Goal}},
				Plan:           []CopilotPlanItem{},
				CompletedTools: []CopilotCompletedTool{tool},
				NextAction:     "continue",
				Counters:       CopilotUsage{Iterations: 1, ToolCalls: 1},
			}
			payload, err := json.Marshal(state)
			if err != nil {
				t.Fatal(err)
			}
			frame := CopilotAgentFrame{
				SchemaVersion: 1, RunID: run.ID, ExecutionID: run.ExecutionID,
				FrameID: "frm_checkpoint", Type: "checkpoint", Payload: payload,
			}
			if err := service.applyFrame(run, frame); !errors.Is(err, ErrInvalidCopilotFrame) {
				t.Fatalf("tampered checkpoint error=%v, want invalid frame", err)
			}
		})
	}
}

func TestCoinCopilotWorkerRejectsTamperedSpecialistResult(t *testing.T) {
	mutations := map[string]func(map[string]any){
		"extra field": func(result map[string]any) { result["unexpected"] = true },
		"capability mismatch": func(result map[string]any) {
			result["capability"] = "auction_search"
		},
		"item kind mismatch": func(result map[string]any) {
			result["items"].([]any)[0].(map[string]any)["kind"] = "auction_lot"
		},
		"fabricated provenance": func(result map[string]any) {
			item := result["items"].([]any)[0].(map[string]any)
			item["provenance"].([]any)[0].(map[string]any)["source_url"] = "https://attacker.example/listing"
		},
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			db, service := newCopilotServiceTest(t)
			repo := repository.NewCoinCopilotRepository(db)
			suffix := strings.ReplaceAll(name, " ", "_")
			thread := &models.CoinCopilotThread{ID: "cct_result_" + suffix, UserID: 7, Title: "Result"}
			run := &models.CoinCopilotRun{
				ID: "ccr_result_" + suffix, ThreadID: thread.ID, UserID: thread.UserID,
				Status: models.CopilotRunRunning, Goal: "Find market examples",
				StartIdempotencyKeyHash: "result-key-" + name, StartRequestFingerprint: "fingerprint",
				ExecutionID: "cce_result", ExecutionAttempt: 1, MaxIterations: 8, MaxToolCalls: 12,
				MaxConcurrentTools: 3, HardTimeoutSeconds: 120, MaxPersistedToolResultBytes: 32768,
			}
			if err := repo.CreateRun(thread, run); err != nil {
				t.Fatal(err)
			}
			result, err := loadCoinCopilotFixture[map[string]any](
				t,
				filepath.Join("specialists", "market_search_complete.json"),
			)
			if err != nil {
				t.Fatal(err)
			}
			mutate(result)
			payload, err := json.Marshal(map[string]any{
				"tool_call_id": "call_market", "tool_name": "market_search", "step_id": "step_market",
				"status": "succeeded", "duration_ms": 1, "result_summary": "Market result.", "result": result,
			})
			if err != nil {
				t.Fatal(err)
			}
			frame := CopilotAgentFrame{
				SchemaVersion: 1, RunID: run.ID, ExecutionID: run.ExecutionID,
				FrameID: "frm_result", Type: "tool_completed", Payload: payload,
			}
			if err := service.applyFrame(run, frame); !errors.Is(err, ErrInvalidCopilotFrame) {
				t.Fatalf("tampered result error=%v, want invalid frame", err)
			}
		})
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
	result, err := loadCoinCopilotFixture[CopilotSpecialistResult](
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
