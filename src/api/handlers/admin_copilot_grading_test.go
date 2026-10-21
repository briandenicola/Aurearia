package handlers

import (
	"bytes"
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

func setupCopilotGradingRouter(t *testing.T) (*gin.Engine, *gorm.DB, *services.CopilotGradingScheduler) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.AppSetting{}, &models.CoinCopilotThread{}, &models.CoinCopilotRun{},
		&models.CoinCopilotCheckpoint{}, &models.CoinCopilotRunGrade{}); err != nil {
		t.Fatal(err)
	}
	gradeRepo := repository.NewCoinCopilotGradeRepository(db)
	scheduler := services.NewCopilotGradingScheduler(
		gradeRepo, repository.NewCoinCopilotRepository(db), nil,
		services.NewSettingsService(repository.NewSettingsRepository(db)), services.NewLogger(10),
	)
	handler := NewAdminCopilotGradingHandler(gradeRepo, scheduler)
	router := gin.New()
	router.GET("/admin/copilot-grades", handler.ListGrades)
	router.GET("/admin/copilot-grades/summary", handler.Summary)
	router.GET("/admin/copilot-grades/:id", handler.GetGrade)
	router.PUT("/admin/copilot-grades/:id/dispute", handler.SetDispute)
	router.GET("/admin/copilot-grading/status", handler.GetStatus)
	router.POST("/admin/copilot-grading/run", handler.RunNow)
	return router, db, scheduler
}

func seedHandlerGrade(t *testing.T, db *gorm.DB, id string, quality float64) models.CoinCopilotRunGrade {
	t.Helper()
	completedAt := time.Now().UTC()
	thread := models.CoinCopilotThread{ID: "cct_" + id, UserID: 4, Title: id}
	run := models.CoinCopilotRun{
		ID: id, ThreadID: thread.ID, UserID: 4, Status: models.CopilotRunCompleted, Goal: "Is it Caligula?",
		FinalAnswer: "Yes, it is Caligula.", StartIdempotencyKeyHash: id, StartRequestFingerprint: "fp",
		ExecutionID: "cce_" + id, MaxIterations: 8, MaxToolCalls: 12, MaxConcurrentTools: 1,
		HardTimeoutSeconds: 120, MaxPersistedToolResultBytes: 32768, CompletedAt: &completedAt,
	}
	grade := models.CoinCopilotRunGrade{
		RunID: id, ThreadID: thread.ID, UserID: 4, Quality: quality, CitesSources: "yes",
		AnswersDirectly: true, Grounded: true, Accurate: true, NoOfferMenu: true, NoFalseInability: true,
		Explanation: "ok", JudgeModel: "claude-sonnet-5-5", GradedAt: completedAt,
	}
	for _, row := range []any{&thread, &run, &grade} {
		if err := db.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}
	return grade
}

func serveCopilotGrading(router *gin.Engine, method, path string, body any) *httptest.ResponseRecorder {
	var reader *bytes.Reader
	if body != nil {
		payload, _ := json.Marshal(body)
		reader = bytes.NewReader(payload)
	} else {
		reader = bytes.NewReader(nil)
	}
	request := httptest.NewRequest(method, path, reader)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
}

func TestAdminCopilotGradingListsWorstFirstAndRejectsBadFilters(t *testing.T) {
	router, db, _ := setupCopilotGradingRouter(t)
	seedHandlerGrade(t, db, "ccr_good", 1)
	seedHandlerGrade(t, db, "ccr_bad", 0.25)

	recorder := serveCopilotGrading(router, http.MethodGet, "/admin/copilot-grades?limit=1", nil)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", recorder.Code, recorder.Body.String())
	}
	var page CopilotGradeListResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &page); err != nil {
		t.Fatal(err)
	}
	if page.Total != 2 || len(page.Grades) != 1 || page.Grades[0].RunID != "ccr_bad" || page.Grades[0].Question != "Is it Caligula?" {
		t.Fatalf("page = %#v", page)
	}

	if recorder := serveCopilotGrading(router, http.MethodGet, "/admin/copilot-grades?disputed=maybe", nil); recorder.Code != http.StatusBadRequest {
		t.Fatalf("bad filter status = %d", recorder.Code)
	}
}

func TestAdminCopilotGradingDisputeRemovesTheGradeFromTheSummary(t *testing.T) {
	router, db, _ := setupCopilotGradingRouter(t)
	bad := seedHandlerGrade(t, db, "ccr_bad", 0.25)
	seedHandlerGrade(t, db, "ccr_good", 1)

	recorder := serveCopilotGrading(router, http.MethodPut, "/admin/copilot-grades/"+jsonID(bad.ID)+"/dispute",
		CopilotGradeDisputeRequest{Disputed: true, Note: "The answer cited RPC correctly."})
	if recorder.Code != http.StatusOK {
		t.Fatalf("dispute status = %d body=%s", recorder.Code, recorder.Body.String())
	}
	var row repository.CopilotGradeRow
	if err := json.Unmarshal(recorder.Body.Bytes(), &row); err != nil || !row.Disputed || row.DisputeNote == "" {
		t.Fatalf("row = %#v err=%v", row, err)
	}

	recorder = serveCopilotGrading(router, http.MethodGet, "/admin/copilot-grades/summary?days=7", nil)
	var summary repository.CopilotGradeSummary
	if err := json.Unmarshal(recorder.Body.Bytes(), &summary); err != nil {
		t.Fatal(err)
	}
	if summary.Graded != 1 || summary.Disputed != 1 || summary.AvgQuality != 1 || summary.Days != 7 {
		t.Fatalf("summary = %#v", summary)
	}

	if recorder := serveCopilotGrading(router, http.MethodPut, "/admin/copilot-grades/999/dispute", CopilotGradeDisputeRequest{Disputed: true}); recorder.Code != http.StatusNotFound {
		t.Fatalf("missing grade status = %d", recorder.Code)
	}
	if recorder := serveCopilotGrading(router, http.MethodGet, "/admin/copilot-grades/abc", nil); recorder.Code != http.StatusBadRequest {
		t.Fatalf("bad id status = %d", recorder.Code)
	}
}

func TestAdminCopilotGradingRunNowReportsOverlap(t *testing.T) {
	router, _, scheduler := setupCopilotGradingRouter(t)
	if err := scheduler.StartManualRun(); err != nil {
		t.Fatal(err)
	}
	// The first run skips quickly (no Anthropic key); wait for it to finish.
	deadline := time.Now().Add(2 * time.Second)
	for scheduler.GetStatus().IsRunning && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}

	recorder := serveCopilotGrading(router, http.MethodGet, "/admin/copilot-grading/status", nil)
	var status services.CopilotGradingStatus
	if err := json.Unmarshal(recorder.Body.Bytes(), &status); err != nil {
		t.Fatal(err)
	}
	if status.Name != "copilot-grading" || status.LastCycle == nil || status.LastCycle.Skipped == "" {
		t.Fatalf("status = %#v", status)
	}
	if recorder := serveCopilotGrading(router, http.MethodPost, "/admin/copilot-grading/run", nil); recorder.Code != http.StatusAccepted {
		t.Fatalf("run status = %d", recorder.Code)
	}
}

func jsonID(id uint) string {
	payload, _ := json.Marshal(id)
	return string(payload)
}
