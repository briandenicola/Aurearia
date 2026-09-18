package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/handlers"
	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

type coinCopilotSeamAgent struct {
	internalToken string
	tokenVerifier *services.InternalTokenService

	mu               sync.Mutex
	executeCount     int
	requests         []services.CopilotExecuteProxyRequest
	tokens           []string
	blockedStarted   chan int
	blockedFinished  chan int
	errs             chan error
	specialistResult map[string]any
}

func newCoinCopilotSeamAgent(t *testing.T, internalToken, tokenSecret string) *coinCopilotSeamAgent {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(
		"..", "..", "agent", "tests", "fixtures", "coin_copilot", "specialists", "market_search_complete.json",
	))
	if err != nil {
		t.Fatal(err)
	}
	var specialistResult map[string]any
	if err := json.Unmarshal(raw, &specialistResult); err != nil {
		t.Fatal(err)
	}
	return &coinCopilotSeamAgent{
		internalToken:    internalToken,
		tokenVerifier:    services.NewInternalTokenService(tokenSecret),
		blockedStarted:   make(chan int, 2),
		blockedFinished:  make(chan int, 2),
		errs:             make(chan error, 20),
		specialistResult: specialistResult,
	}
}

func (a *coinCopilotSeamAgent) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/api/copilot/capability":
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"supported":true}`)
	case "/api/copilot/execute":
		a.execute(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (a *coinCopilotSeamAgent) execute(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("X-Internal-Service-Token") != a.internalToken {
		a.recordError(errors.New("missing internal service credential"))
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var request services.CopilotExecuteProxyRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		a.recordError(fmt.Errorf("decode execute request: %w", err))
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		a.recordError(fmt.Errorf("execute request trailing JSON: %v", err))
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	claims, err := a.tokenVerifier.VerifyForCopilotExecution(request.ExecutionToken, "collection_summary")
	if err != nil {
		a.recordError(fmt.Errorf("verify execution token: %w", err))
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if claims.UserID != 7 || claims.RunID != request.RunID || claims.ExecutionID != request.ExecutionID {
		a.recordError(fmt.Errorf("execution claims do not match request: %#v %#v", claims, request))
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	a.mu.Lock()
	a.executeCount++
	attempt := a.executeCount
	a.requests = append(a.requests, request)
	a.tokens = append(a.tokens, request.ExecutionToken)
	a.mu.Unlock()

	w.Header().Set("Content-Type", "text/event-stream")
	switch attempt {
	case 1:
		a.writeFrame(w, request, "frm_tool_start", "tool_started", map[string]any{
			"tool_call_id": "call_market", "tool_name": "market_search", "step_id": "step_1",
		})
		a.writeFrame(w, request, "frm_tool_done", "tool_completed", map[string]any{
			"tool_call_id": "call_market", "tool_name": "market_search", "step_id": "step_1",
			"status": "succeeded", "duration_ms": 2, "result_summary": "Market evidence ready.",
			"result": a.specialistResult,
		})
		a.writeFrame(w, request, "frm_tool_truncated", "tool_completed", map[string]any{
			"tool_call_id": "call_market_truncated", "tool_name": "market_search", "step_id": "step_2",
			"status": "succeeded", "duration_ms": 2, "result_summary": "Market evidence exceeded the result limit.",
			"result": map[string]any{
				"truncated": true, "original_bytes": 65536, "digest": strings.Repeat("a", 64),
				"summary": "Tool result exceeded the persisted-result limit.",
			},
		})
		a.writeSpecialistCheckpoint(w, request, "frm_checkpoint_1")
		a.blockedStarted <- attempt
		<-r.Context().Done()
		a.blockedFinished <- attempt
	case 2:
		if len(request.Checkpoint.CompletedTools) != 2 ||
			request.Checkpoint.CompletedTools[0].ToolCallID != "call_market" ||
			request.Checkpoint.CompletedTools[0].ToolName != "market_search" ||
			request.Checkpoint.CompletedTools[1].ToolCallID != "call_market_truncated" ||
			!request.Checkpoint.CompletedTools[1].Truncated {
			a.recordError(fmt.Errorf("specialist checkpoint was not hydrated: %#v", request.Checkpoint))
			http.Error(w, "invalid checkpoint", http.StatusBadRequest)
			return
		}
		a.writeFrame(w, request, "frm_completed_2", "completed", map[string]any{
			"answer": "Resumed from the durable checkpoint.",
			"usage":  map[string]any{"iterations": 2, "tool_calls": 2, "input_tokens": 30, "output_tokens": 12},
		})
	case 3:
		a.writeCheckpoint(w, request, "frm_checkpoint_3", 1)
		a.blockedStarted <- attempt
		<-r.Context().Done()
		a.blockedFinished <- attempt
	default:
		a.recordError(fmt.Errorf("unexpected execution attempt %d", attempt))
		http.Error(w, "unexpected execution", http.StatusInternalServerError)
	}
}

func (a *coinCopilotSeamAgent) writeSpecialistCheckpoint(
	w http.ResponseWriter,
	request services.CopilotExecuteProxyRequest,
	frameID string,
) {
	raw, err := json.Marshal(a.specialistResult)
	if err != nil {
		a.recordError(fmt.Errorf("marshal specialist result: %w", err))
		return
	}
	bounded, originalBytes, truncated, digest, err := services.SanitizeCopilotJSON(raw, 32768)
	if err != nil {
		a.recordError(fmt.Errorf("bound specialist result: %w", err))
		return
	}
	var result map[string]any
	if err := json.Unmarshal(bounded, &result); err != nil {
		a.recordError(fmt.Errorf("decode bounded specialist result: %w", err))
		return
	}
	truncatedResult := map[string]any{
		"truncated": true, "original_bytes": 65536, "digest": strings.Repeat("a", 64),
		"summary": "Tool result exceeded the persisted-result limit.",
	}
	truncatedRaw, err := json.Marshal(truncatedResult)
	if err != nil {
		a.recordError(fmt.Errorf("marshal truncated specialist result: %w", err))
		return
	}
	a.writeFrame(w, request, frameID, "checkpoint", map[string]any{
		"schema_version": 1,
		"messages":       []map[string]any{{"role": "user", "content": request.Goal}},
		"plan":           []map[string]any{{"id": "step_1", "title": "Search dealer listings", "status": "completed"}},
		"completed_tools": []map[string]any{
			{
				"tool_call_id": "call_market", "tool_name": "market_search", "result_digest": digest,
				"result": result, "original_bytes": originalBytes, "persisted_bytes": len(bounded), "truncated": truncated,
			},
			{
				"tool_call_id": "call_market_truncated", "tool_name": "market_search",
				"result_digest": strings.Repeat("a", 64), "result": truncatedResult,
				"original_bytes": 65536, "persisted_bytes": len(truncatedRaw), "truncated": true,
			},
		},
		"pending_clarification": nil,
		"next_action":           "continue",
		"counters": map[string]any{
			"iterations": 1, "tool_calls": 2, "input_tokens": 20, "output_tokens": 5,
		},
	})
}

func (a *coinCopilotSeamAgent) writeCheckpoint(w http.ResponseWriter, request services.CopilotExecuteProxyRequest, frameID string, toolCalls int) {
	a.writeFrame(w, request, frameID, "checkpoint", map[string]any{
		"schema_version": 1,
		"messages":       []map[string]any{{"role": "user", "content": request.Goal}},
		"plan":           []map[string]any{{"id": "step_1", "title": "Summarize collection", "status": "completed"}},
		"completed_tools": []map[string]any{{
			"tool_call_id": "call_1", "tool_name": "collection_summary", "result_digest": "",
			"result": map[string]any{"totalCoins": 2}, "original_bytes": 0, "persisted_bytes": 0, "truncated": false,
		}},
		"pending_clarification": nil,
		"next_action":           "continue",
		"counters": map[string]any{
			"iterations": 1, "tool_calls": toolCalls, "input_tokens": 20, "output_tokens": 5,
		},
	})
}

func (a *coinCopilotSeamAgent) writeFrame(w http.ResponseWriter, request services.CopilotExecuteProxyRequest, frameID, frameType string, payload any) {
	frame := map[string]any{
		"schema_version": 1, "run_id": request.RunID, "execution_id": request.ExecutionID,
		"frame_id": frameID, "type": frameType, "payload": payload,
	}
	encoded, err := json.Marshal(frame)
	if err != nil {
		a.recordError(fmt.Errorf("marshal frame: %w", err))
		return
	}
	_, _ = fmt.Fprintf(w, "data: %s\n\n", encoded)
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (a *coinCopilotSeamAgent) recordError(err error) {
	select {
	case a.errs <- err:
	default:
	}
}

func (a *coinCopilotSeamAgent) snapshot() ([]services.CopilotExecuteProxyRequest, []string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]services.CopilotExecuteProxyRequest(nil), a.requests...), append([]string(nil), a.tokens...)
}

func newCoinCopilotSeamService(t *testing.T, db *gorm.DB, agentURL, internalToken, tokenSecret string) (*services.CoinCopilotService, *services.InternalTokenService) {
	t.Helper()
	settings := services.NewSettingsService(repository.NewSettingsRepository(db))
	for key, value := range map[string]string{
		services.SettingCoinCopilotEnabled:          "true",
		services.SettingCoinCopilotWorkerCount:      "1",
		services.SettingAIProvider:                  "anthropic",
		services.SettingAnthropicAPIKey:             "seam-test-key",
		services.SettingCoinCopilotMaxActivePerUser: "1",
	} {
		if err := settings.SetSetting(key, value); err != nil {
			t.Fatalf("set %s: %v", key, err)
		}
	}
	tokenSvc := services.NewInternalTokenService(tokenSecret)
	service := services.NewCoinCopilotService(
		repository.NewCoinCopilotRepository(db),
		settings,
		services.NewAgentProxy(agentURL, internalToken, services.NewLogger(100)),
		tokenSvc,
		services.NewLogger(100),
		"http://api.test",
	)
	return service, tokenSvc
}

func waitForCoinCopilotRun(t *testing.T, service *services.CoinCopilotService, runID string, predicate func(*models.CoinCopilotRun) bool) *models.CoinCopilotRun {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		run, err := service.GetRun(7, runID)
		if err == nil && predicate(run) {
			return run
		}
		time.Sleep(10 * time.Millisecond)
	}
	run, err := service.GetRun(7, runID)
	t.Fatalf("run %s did not reach expected state; final=%#v err=%v", runID, run, err)
	return nil
}

func waitForCoinCopilotAttempt(t *testing.T, ch <-chan int, want int) {
	t.Helper()
	select {
	case got := <-ch:
		if got != want {
			t.Fatalf("execution attempt signal=%d, want %d", got, want)
		}
	case <-time.After(3 * time.Second):
		t.Fatalf("execution attempt %d did not reach fake agent", want)
	}
}

func TestCoinCopilotSeamRestartResumeCancellationAndTerminalSSE(t *testing.T) {
	gin.SetMode(gin.TestMode)
	dsn := fmt.Sprintf("file:coin_copilot_seam_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(
		&models.AppSetting{}, &models.CoinCopilotThread{}, &models.CoinCopilotRun{},
		&models.CoinCopilotCheckpoint{}, &models.CoinCopilotEvent{}, &models.CoinCopilotResumeRequest{},
	); err != nil {
		t.Fatal(err)
	}

	const (
		internalToken = "coin-copilot-seam-internal"
		tokenSecret   = "coin-copilot-seam-token-secret"
	)
	agent := newCoinCopilotSeamAgent(t, internalToken, tokenSecret)
	server := httptest.NewServer(agent)
	defer server.Close()
	repo := repository.NewCoinCopilotRepository(db)

	firstService, firstTokenSvc := newCoinCopilotSeamService(t, db, server.URL, internalToken, tokenSecret)
	firstCtx, stopFirst := context.WithCancel(context.Background())
	firstService.StartWorkers(firstCtx)
	run, reused, err := firstService.Start(7, services.CoinCopilotStartInput{
		Goal: "Summarize my collection", IdempotencyKey: "seam-initial",
	})
	if err != nil || reused {
		t.Fatalf("start run=%#v reused=%v err=%v", run, reused, err)
	}
	waitForCoinCopilotAttempt(t, agent.blockedStarted, 1)
	running := waitForCoinCopilotRun(t, firstService, run.ID, func(current *models.CoinCopilotRun) bool {
		return current.Status == models.CopilotRunRunning && current.CheckpointVersion == 1 && current.LastSeq == 4
	})
	firstExecutionID := running.ExecutionID
	requests, tokens := agent.snapshot()
	if len(requests) != 1 || requests[0].ExecutionID != firstExecutionID || len(tokens) != 1 {
		t.Fatalf("initial execution request mismatch: requests=%#v tokens=%d", requests, len(tokens))
	}
	firstToken := tokens[0]
	stopFirst()
	waitForCoinCopilotAttempt(t, agent.blockedFinished, 1)
	tokenRevoked := false
	for deadline := time.Now().Add(time.Second); time.Now().Before(deadline); {
		if _, err := firstTokenSvc.VerifyForCopilotExecution(firstToken, "collection_summary"); errors.Is(err, services.ErrInvalidInternalToken) {
			tokenRevoked = true
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if !tokenRevoked {
		t.Fatal("stopped execution token remained valid")
	}
	if err := db.Model(&models.CoinCopilotRun{}).Where("id = ?", run.ID).
		Update("heartbeat_at", time.Now().UTC().Add(-time.Minute)).Error; err != nil {
		t.Fatal(err)
	}

	secondService, _ := newCoinCopilotSeamService(t, db, server.URL, internalToken, tokenSecret)
	if err := secondService.RecoverAndPrune(); err != nil {
		t.Fatal(err)
	}
	paused := waitForCoinCopilotRun(t, secondService, run.ID, func(current *models.CoinCopilotRun) bool {
		return current.Status == models.CopilotRunPaused
	})
	if paused.CheckpointVersion != 1 {
		t.Fatalf("recovered checkpoint version=%d, want 1", paused.CheckpointVersion)
	}
	resumed, replayed, err := secondService.Resume(7, run.ID, services.CoinCopilotResumeInput{
		Answer: "Focus on Roman coins", ExpectedCheckpointVersion: 1, IdempotencyKey: "seam-resume",
	})
	if err != nil || replayed || resumed.Status != models.CopilotRunQueued {
		t.Fatalf("resume run=%#v replayed=%v err=%v", resumed, replayed, err)
	}
	secondCtx, stopSecond := context.WithCancel(context.Background())
	secondService.StartWorkers(secondCtx)
	completed := waitForCoinCopilotRun(t, secondService, run.ID, func(current *models.CoinCopilotRun) bool {
		return current.Status == models.CopilotRunCompleted
	})
	requests, tokens = agent.snapshot()
	if len(requests) < 2 || len(tokens) < 2 {
		t.Fatalf("resume execution was not captured: requests=%d tokens=%d", len(requests), len(tokens))
	}
	resumeRequest := requests[1]
	if resumeRequest.ExecutionID == firstExecutionID || tokens[1] == firstToken ||
		resumeRequest.Checkpoint.Version != 2 || completed.ExecutionAttempt != 2 {
		t.Fatalf("resume did not use fresh execution state: request=%#v run=%#v", resumeRequest, completed)
	}
	if resumeRequest.ExecutionID != resumed.ExecutionID {
		t.Fatalf("worker claim replaced resume execution id: resume=%s execute=%s", resumed.ExecutionID, resumeRequest.ExecutionID)
	}

	checkpoint, err := repo.GetLatestCheckpoint(run.ID, 7)
	if err != nil {
		t.Fatal(err)
	}
	events, err := repo.ListEventsSince(run.ID, 7, 0)
	if err != nil {
		t.Fatal(err)
	}
	if checkpoint.Version != completed.CheckpointVersion || completed.LastSeq != int64(len(events)) {
		t.Fatalf("durable state mismatch: checkpoint=%d runCheckpoint=%d lastSeq=%d events=%d",
			checkpoint.Version, completed.CheckpointVersion, completed.LastSeq, len(events))
	}
	for i, event := range events {
		if event.Seq != int64(i+1) {
			t.Fatalf("event sequence gap at index %d: %#v", i, events)
		}
	}
	if events[len(events)-1].Type != models.CopilotEventRunCompleted {
		t.Fatalf("last event=%s, want run_completed", events[len(events)-1].Type)
	}
	specialistCompletions := 0
	for _, event := range events {
		if event.Type == models.CopilotEventToolCompleted &&
			bytes.Contains([]byte(event.PayloadJSON), []byte(`"toolCallId":"call_market"`)) {
			specialistCompletions++
		}
	}
	if specialistCompletions != 1 {
		t.Fatalf("specialist completion events=%d, want 1", specialistCompletions)
	}
	truncatedCompletions := 0
	for _, event := range events {
		if event.Type == models.CopilotEventToolCompleted &&
			bytes.Contains([]byte(event.PayloadJSON), []byte(`"toolCallId":"call_market_truncated"`)) &&
			bytes.Contains([]byte(event.PayloadJSON), []byte(`"truncated":true`)) {
			truncatedCompletions++
		}
	}
	if truncatedCompletions != 1 {
		t.Fatalf("truncated specialist completion events=%d, want 1", truncatedCompletions)
	}

	handler := handlers.NewCoinCopilotHandler(secondService, services.NewLogger(10))
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("userId", uint(7))
		c.Next()
	})
	router.GET("/runs/:runId/events", handler.StreamEvents)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/runs/"+run.ID+"/events", nil))
	if recorder.Code != http.StatusOK ||
		!bytes.Contains(recorder.Body.Bytes(), []byte("event: run_completed")) ||
		!bytes.Contains(recorder.Body.Bytes(), []byte("event: end")) {
		t.Fatalf("terminal SSE status=%d body=%s", recorder.Code, recorder.Body.String())
	}

	cancelRun, reused, err := secondService.Start(7, services.CoinCopilotStartInput{
		Goal: "Prepare a cancellable summary", IdempotencyKey: "seam-cancel",
	})
	if err != nil || reused {
		t.Fatalf("start cancellable run=%#v reused=%v err=%v", cancelRun, reused, err)
	}
	waitForCoinCopilotAttempt(t, agent.blockedStarted, 3)
	waitForCoinCopilotRun(t, secondService, cancelRun.ID, func(current *models.CoinCopilotRun) bool {
		return current.Status == models.CopilotRunRunning && current.CheckpointVersion == 1
	})
	stopSecond()
	waitForCoinCopilotAttempt(t, agent.blockedFinished, 3)
	if err := db.Model(&models.CoinCopilotRun{}).Where("id = ?", cancelRun.ID).
		Update("heartbeat_at", time.Now().UTC().Add(-time.Minute)).Error; err != nil {
		t.Fatal(err)
	}
	restartedService, _ := newCoinCopilotSeamService(t, db, server.URL, internalToken, tokenSecret)
	if err := restartedService.RecoverAndPrune(); err != nil {
		t.Fatal(err)
	}
	cancelled, immediate, err := restartedService.Cancel(7, cancelRun.ID)
	if err != nil || !immediate || cancelled.Status != models.CopilotRunCancelled {
		t.Fatalf("cancel after restart run=%#v immediate=%v err=%v", cancelled, immediate, err)
	}
	cancelEvents, err := repo.ListEventsSince(cancelRun.ID, 7, 0)
	if err != nil {
		t.Fatal(err)
	}
	terminalCount := 0
	for _, event := range cancelEvents {
		if event.Type == models.CopilotEventRunCompleted ||
			event.Type == models.CopilotEventRunFailed ||
			event.Type == models.CopilotEventRunCancelled {
			terminalCount++
		}
	}
	if terminalCount != 1 || cancelEvents[len(cancelEvents)-1].Type != models.CopilotEventRunCancelled {
		t.Fatalf("cancel terminal events=%d events=%#v", terminalCount, cancelEvents)
	}

	select {
	case err := <-agent.errs:
		t.Fatal(err)
	default:
	}
}
