package services

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

type fakeCopilotGrader struct {
	requests []CopilotGradeProxyRequest
	failFor  map[string]bool
}

func (f *fakeCopilotGrader) GradeCopilotRun(_ context.Context, request CopilotGradeProxyRequest) (*CopilotGradeProxyResponse, error) {
	f.requests = append(f.requests, request)
	if f.failFor[request.Answer] {
		return nil, errors.New("judge unavailable")
	}
	return &CopilotGradeProxyResponse{
		Quality: 0.8, AnswersDirectly: true, Grounded: true, Accurate: true, CitesSources: "not_applicable",
		NoOfferMenu: false, NoFalseInability: true, Explanation: "Ends with a menu.", JudgeModel: request.LLM.Model,
		InputTokens: 900, OutputTokens: 120,
	}, nil
}

func newCopilotGradingTestScheduler(t *testing.T) (*gorm.DB, *CopilotGradingScheduler, *fakeCopilotGrader, *SettingsService) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.AppSetting{}, &models.CoinCopilotThread{}, &models.CoinCopilotRun{},
		&models.CoinCopilotCheckpoint{}, &models.CoinCopilotRunGrade{}); err != nil {
		t.Fatal(err)
	}
	settings := NewSettingsService(repository.NewSettingsRepository(db))
	grader := &fakeCopilotGrader{failFor: map[string]bool{}}
	scheduler := NewCopilotGradingScheduler(
		repository.NewCoinCopilotGradeRepository(db), repository.NewCoinCopilotRepository(db), grader, settings, NewLogger(100),
	)
	return db, scheduler, grader, settings
}

func seedGradingRun(t *testing.T, db *gorm.DB, id, answer string, state CopilotCheckpointState) {
	t.Helper()
	completedAt := time.Now().UTC()
	thread := models.CoinCopilotThread{ID: "cct_" + id, UserID: 3, Title: id}
	run := models.CoinCopilotRun{
		ID: id, ThreadID: thread.ID, UserID: 3, Status: models.CopilotRunCompleted, Goal: "goal " + id,
		FinalAnswer: answer, StartIdempotencyKeyHash: id, StartRequestFingerprint: "fp", ExecutionID: "cce_" + id,
		MaxIterations: 8, MaxToolCalls: 12, MaxConcurrentTools: 1, HardTimeoutSeconds: 120,
		MaxPersistedToolResultBytes: 32768, CompletedAt: &completedAt,
	}
	stateJSON, _ := json.Marshal(state)
	for _, row := range []any{&thread, &run, &models.CoinCopilotCheckpoint{
		RunID: id, ThreadID: thread.ID, UserID: 3, ExecutionID: run.ExecutionID, Version: 1,
		StateJSON: string(stateJSON), StateDigest: "digest",
	}} {
		if err := db.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}
}

func enableAnthropic(t *testing.T, settings *SettingsService) {
	t.Helper()
	for key, value := range map[string]string{SettingAIProvider: "anthropic", SettingAnthropicAPIKey: "sk-test"} {
		if err := settings.SetSetting(key, value); err != nil {
			t.Fatal(err)
		}
	}
}

func TestCopilotGradingGradesRunsFromTheirCheckpoint(t *testing.T) {
	db, scheduler, grader, settings := newCopilotGradingTestScheduler(t)
	enableAnthropic(t, settings)
	seedGradingRun(t, db, "ccr_one", "The Athens owl leads.", CopilotCheckpointState{
		Messages: []CopilotMessage{
			{Role: "user", Content: "Earlier question"},
			{Role: "assistant", Content: "Earlier answer"},
			{Role: "user", Content: "What are my top coins?"},
			{Role: "assistant", Content: "The Athens owl leads."},
		},
		CompletedTools: []CopilotCompletedTool{
			{ToolCallID: "call_1", ToolName: "top_coins_by_value", Result: json.RawMessage(`{"coins":[]}`)},
		},
	})

	if err := scheduler.RunNow(); err != nil {
		t.Fatal(err)
	}

	if len(grader.requests) != 1 {
		t.Fatalf("grader calls = %d", len(grader.requests))
	}
	request := grader.requests[0]
	if request.LLM.Model != "claude-sonnet-5-5" || request.LLM.APIKey != "sk-test" {
		t.Fatalf("judge config = %#v", request.LLM)
	}
	if len(request.Messages) != 3 || request.Messages[2].Content != "What are my top coins?" {
		t.Fatalf("messages = %#v (the graded answer must not be repeated as history)", request.Messages)
	}
	if request.Answer != "The Athens owl leads." || len(request.Tools) != 1 || request.Tools[0].ToolName != "top_coins_by_value" {
		t.Fatalf("request = %#v", request)
	}
	var grade models.CoinCopilotRunGrade
	if err := db.First(&grade).Error; err != nil {
		t.Fatal(err)
	}
	if grade.RunID != "ccr_one" || grade.Quality != 0.8 || grade.NoOfferMenu || grade.JudgeInputTokens != 900 {
		t.Fatalf("grade = %#v", grade)
	}
	status := scheduler.GetGradingStatus()
	if status.LastCycle == nil || status.LastCycle.Graded != 1 || status.LastCycle.TriggerType != "manual" {
		t.Fatalf("status = %#v", status)
	}

	if err := scheduler.RunNow(); err != nil {
		t.Fatal(err)
	}
	if len(grader.requests) != 1 {
		t.Fatalf("a graded run was graded again: %d calls", len(grader.requests))
	}
}

func TestCopilotGradingCountsFailuresAndRespectsTheDailyCap(t *testing.T) {
	db, scheduler, grader, settings := newCopilotGradingTestScheduler(t)
	enableAnthropic(t, settings)
	if err := settings.SetSetting(SettingCopilotGradingDailyCap, "2"); err != nil {
		t.Fatal(err)
	}
	state := CopilotCheckpointState{Messages: []CopilotMessage{{Role: "user", Content: "q"}}}
	for _, id := range []string{"ccr_a", "ccr_b", "ccr_c"} {
		seedGradingRun(t, db, id, "answer "+id, state)
	}
	for _, id := range []string{"ccr_a", "ccr_b", "ccr_c"} {
		grader.failFor["answer "+id] = true
	}

	if err := scheduler.RunNow(); err != nil {
		t.Fatal(err)
	}

	cycle := scheduler.GetGradingStatus().LastCycle
	if len(grader.requests) != 2 || cycle.Failed != 2 || cycle.Graded != 0 {
		t.Fatalf("calls=%d cycle=%#v", len(grader.requests), cycle)
	}
	var grades int64
	if err := db.Model(&models.CoinCopilotRunGrade{}).Count(&grades).Error; err != nil || grades != 0 {
		t.Fatalf("failed grades were stored: %d err=%v", grades, err)
	}
}

func TestCopilotGradingSkipsWithoutAnAnthropicKey(t *testing.T) {
	db, scheduler, grader, _ := newCopilotGradingTestScheduler(t)
	seedGradingRun(t, db, "ccr_skip", "answer", CopilotCheckpointState{Messages: []CopilotMessage{{Role: "user", Content: "q"}}})

	if err := scheduler.RunNow(); err != nil {
		t.Fatal(err)
	}

	cycle := scheduler.GetGradingStatus().LastCycle
	if len(grader.requests) != 0 || cycle == nil || cycle.Skipped == "" {
		t.Fatalf("calls=%d cycle=%#v", len(grader.requests), cycle)
	}
}

func TestCopilotGradingScheduledCycleHonorsTheEnabledSwitchAndRejectsOverlap(t *testing.T) {
	_, scheduler, _, settings := newCopilotGradingTestScheduler(t)
	enableAnthropic(t, settings)

	scheduler.runCycle()
	if scheduler.GetGradingStatus().LastCycle != nil {
		t.Fatal("disabled schedule still ran")
	}
	if !scheduler.claim() {
		t.Fatal("claim failed on an idle scheduler")
	}
	if err := scheduler.StartManualRun(); !errors.Is(err, ErrCopilotGradingRunning) {
		t.Fatalf("overlapping run error = %v", err)
	}
	if status := scheduler.GetStatus(); !status.IsRunning || status.Enabled || status.Name != "copilot-grading" {
		t.Fatalf("status = %#v", status)
	}
}

func TestAgentProxyGradeCopilotRunUsesInternalCredential(t *testing.T) {
	const token = "test-internal-service-token"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/copilot/grade" || r.Method != http.MethodPost {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("X-Internal-Service-Token"); got != token {
			t.Fatalf("internal token header = %q", got)
		}
		var body CopilotGradeProxyRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Answer != "answer" {
			t.Fatalf("body = %#v err=%v", body, err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"quality":0.5,"answers_directly":false,"grounded":true,"accurate":true,"cites_sources":"no","no_offer_menu":true,"no_false_inability":true,"explanation":"x","judge_model":"claude-sonnet-5-5","input_tokens":10,"output_tokens":2}`))
	}))
	defer server.Close()

	proxy := NewAgentProxy(server.URL, token, NewLogger(10))
	verdict, err := proxy.GradeCopilotRun(context.Background(), CopilotGradeProxyRequest{
		LLM:      LLMConfig{Provider: "anthropic", APIKey: "secret", Model: "claude-sonnet-5-5"},
		Messages: []CopilotMessage{{Role: "user", Content: "q"}},
		Answer:   "answer",
	})
	if err != nil {
		t.Fatal(err)
	}
	if verdict.Quality != 0.5 || verdict.AnswersDirectly || verdict.CitesSources != "no" {
		t.Fatalf("verdict = %#v", verdict)
	}
}

func TestAgentProxyGradeCopilotRunReportsAgentErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"detail":"Copilot grading produced no verdict"}`))
	}))
	defer server.Close()

	proxy := NewAgentProxy(server.URL, "token", NewLogger(10))
	if _, err := proxy.GradeCopilotRun(context.Background(), CopilotGradeProxyRequest{Answer: "a"}); err == nil {
		t.Fatal("expected an error for a 502 from the agent")
	}
}
