package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupCoinCopilotHandlerTest(t *testing.T) (*gin.Engine, *gorm.DB, *services.CoinCopilotService) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}

	if err := db.AutoMigrate(
		&models.AppSetting{}, &models.CoinCopilotThread{}, &models.CoinCopilotRun{},
		&models.CoinCopilotCheckpoint{}, &models.CoinCopilotEvent{}, &models.CoinCopilotResumeRequest{},
	); err != nil {
		t.Fatal(err)
	}
	settings := services.NewSettingsService(repository.NewSettingsRepository(db))
	_ = settings.SetSetting(services.SettingCoinCopilotEnabled, "true")
	_ = settings.SetSetting(services.SettingAIProvider, "anthropic")
	_ = settings.SetSetting(services.SettingAnthropicAPIKey, "test")
	agent := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/copilot/capability" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(`{"supported":true}`))
	}))
	t.Cleanup(agent.Close)
	service := services.NewCoinCopilotService(
		repository.NewCoinCopilotRepository(db), settings,
		services.NewAgentProxy(agent.URL, "internal", services.NewLogger(10)),
		services.NewInternalTokenService("01234567890123456789012345678901"),
		services.NewLogger(10), "http://api:8080",
	)
	handler := NewCoinCopilotHandler(service, services.NewLogger(10))
	router := gin.New()
	router.Use(func(c *gin.Context) { c.Set("userId", uint(7)); c.Next() })
	router.GET("/capability", handler.Capability)
	router.POST("/runs", handler.Start)
	router.GET("/runs/:runId", handler.GetRun)
	router.POST("/runs/:runId/cancel", handler.Cancel)
	router.GET("/runs/:runId/events", handler.StreamEvents)
	router.GET("/threads/:threadId", handler.GetThread)
	router.DELETE("/threads/:threadId", handler.DeleteThread)
	router.POST("/runs/:runId/resume", handler.Resume)
	return router, db, service
}

func TestCoinCopilotRunDTOExposesTokensWithoutEstimatedCost(t *testing.T) {
	encoded, err := json.Marshal(toCopilotRunDTO(&models.CoinCopilotRun{
		ID: "ccr_usage", ThreadID: "cct_usage", InputTokens: 120, OutputTokens: 45,
	}))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(encoded, []byte("cost")) {
		t.Fatalf("cost field leaked in public run DTO: %s", encoded)
	}
	if !bytes.Contains(encoded, []byte(`"inputTokens":120`)) ||
		!bytes.Contains(encoded, []byte(`"outputTokens":45`)) {
		t.Fatalf("token usage missing from public run DTO: %s", encoded)
	}
}

func TestCoinCopilotHandlerCapabilityStartValidationAndForeign404(t *testing.T) {
	router, _, service := setupCoinCopilotHandlerTest(t)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/capability", nil))
	if recorder.Code != http.StatusOK || !bytes.Contains(recorder.Body.Bytes(), []byte(`"mode":"copilot"`)) {
		t.Fatalf("capability status=%d body=%s", recorder.Code, recorder.Body.String())
	}

	recorder = httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/runs", bytes.NewBufferString(`{"goal":"Compare my bronzes"}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotency-Key", "handler-start")
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusAccepted {
		t.Fatalf("start status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	var response struct {
		Run copilotRunDTO `json:"run"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if _, err := service.GetRun(8, response.Run.ID); err == nil {
		t.Fatal("foreign service read unexpectedly succeeded")
	}

	recorder = httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodPost, "/runs", bytes.NewBufferString(`{"goal":"x","unknown":true}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotency-Key", "handler-invalid")
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("unknown field status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	recorder = httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodPost, "/runs", bytes.NewBufferString(`{"goal":"x"}{"goal":"y"}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotency-Key", "handler-multiple")
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("multiple JSON values status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestCoinCopilotSSETerminalReplayAndSincePrecedence(t *testing.T) {
	router, db, service := setupCoinCopilotHandlerTest(t)
	run, _, err := service.Start(7, services.CoinCopilotStartInput{Goal: "Summary", IdempotencyKey: "sse-start"})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&models.CoinCopilotRun{}).Where("id = ?", run.ID).Updates(map[string]any{
		"status": models.CopilotRunRunning, "execution_id": "cce_sse",
	}).Error; err != nil {
		t.Fatal(err)
	}
	repo := repository.NewCoinCopilotRepository(db)
	if _, err := repo.AppendEvent(run.ID, 7, "cce_sse", models.CopilotEventRunStarted, `{"status":"running"}`); err != nil {
		t.Fatal(err)
	}
	if _, _, err := repo.TransitionWithEvent(run.ID, 7, "cce_sse", []models.CopilotRunStatus{models.CopilotRunRunning},
		models.CopilotRunCompleted, map[string]interface{}{"final_answer": "done"},
		models.CopilotEventRunCompleted, `{"answer":"done"}`); err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/runs/"+run.ID+"/events?since=1", nil)
	request.Header.Set("Last-Event-ID", "0")
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK || bytes.Contains(recorder.Body.Bytes(), []byte("run_started")) ||
		!bytes.Contains(recorder.Body.Bytes(), []byte("run_completed")) || !bytes.Contains(recorder.Body.Bytes(), []byte("event: end")) {
		t.Fatalf("SSE status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestCoinCopilotHandlerThreadDeleteConflictCascadeAndForeign404(t *testing.T) {
	router, db, service := setupCoinCopilotHandlerTest(t)
	run, _, err := service.Start(7, services.CoinCopilotStartInput{Goal: "Summary", IdempotencyKey: "delete-start"})
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodDelete, "/threads/"+run.ThreadID, nil))
	if recorder.Code != http.StatusConflict {
		t.Fatalf("active delete status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if err := db.Model(&models.CoinCopilotRun{}).Where("id = ?", run.ID).
		Updates(map[string]any{"status": models.CopilotRunCompleted, "completed_at": time.Now().UTC()}).Error; err != nil {
		t.Fatal(err)
	}
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodDelete, "/threads/"+run.ThreadID, nil))
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("settled delete status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	var runCount int64
	if err := db.Model(&models.CoinCopilotRun{}).Where("id = ?", run.ID).Count(&runCount).Error; err != nil {
		t.Fatal(err)
	}
	if runCount != 0 {
		t.Fatalf("cascade retained %d runs", runCount)
	}
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/threads/"+run.ThreadID, nil))
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("deleted thread read status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestCoinCopilotSSEKeepaliveTruncationAndConnectionLimit(t *testing.T) {
	router, db, service := setupCoinCopilotHandlerTest(t)
	run, _, err := service.Start(7, services.CoinCopilotStartInput{Goal: "Summary", IdempotencyKey: "sse-controls"})
	if err != nil {
		t.Fatal(err)
	}
	oldPing := copilotSSEPingInterval
	copilotSSEPingInterval = 5 * time.Millisecond
	t.Cleanup(func() { copilotSSEPingInterval = oldPing })
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/runs/"+run.ID+"/events", nil).WithContext(ctx))
	if recorder.Code != http.StatusOK || !bytes.Contains(recorder.Body.Bytes(), []byte(": ping")) {
		t.Fatalf("keepalive status=%d body=%s", recorder.Code, recorder.Body.String())
	}

	unsubscribes := make([]func(), 0, 3)
	for range 3 {
		_, unsubscribe, ok := service.Broker().Subscribe(run.ID)
		if !ok {
			t.Fatal("failed to reserve stream slot")
		}
		unsubscribes = append(unsubscribes, unsubscribe)
	}
	defer func() {
		for _, unsubscribe := range unsubscribes {
			unsubscribe()
		}
	}()
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/runs/"+run.ID+"/events", nil))
	if recorder.Code != http.StatusTooManyRequests {
		t.Fatalf("connection limit status=%d body=%s", recorder.Code, recorder.Body.String())
	}

	now := time.Now().UTC()
	if err := db.Model(&models.CoinCopilotRun{}).Where("id = ?", run.ID).Updates(map[string]any{
		"status": models.CopilotRunCompleted, "completed_at": now, "last_seq": 5, "events_pruned_at": now,
	}).Error; err != nil {
		t.Fatal(err)
	}
	for _, unsubscribe := range unsubscribes {
		unsubscribe()
	}
	unsubscribes = nil
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/runs/"+run.ID+"/events?since=1", nil))
	if recorder.Code != http.StatusOK || !bytes.Contains(recorder.Body.Bytes(), []byte("event: stream_truncated")) ||
		!bytes.Contains(recorder.Body.Bytes(), []byte("event: end")) {
		t.Fatalf("truncated stream status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}
