package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// Brutus (QA): Phase 3 coverage for POST /set-builder/runs.
//
// Correction plan acceptance target (specs/011-dynamic-set-builder-correction-plan.md,
// Phase 0 + Phase 3 #1): "Submitting an Agentic prompt no longer creates a set
// immediately" and "Approval is the only code path that creates the actual set."
// This file proves the currently wired submit endpoint honors that contract at the
// HTTP layer without touching handler/service/repository implementation files that
// are under active development.

func setupSetBuilderHandlerTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	err = db.AutoMigrate(
		&models.User{},
		&models.CoinSet{}, &models.CoinSetTarget{},
		&models.SetBuilderRun{}, &models.SetProposal{}, &models.ProposalSlot{},
		&models.Notification{},
	)
	if err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	return db
}

func setupSetBuilderHandlerRouter(t *testing.T) (*gin.Engine, *gorm.DB) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db := setupSetBuilderHandlerTestDB(t)
	setBuilderRepo := repository.NewSetBuilderRepository(db)
	notifRepo := repository.NewNotificationRepository(db)
	setBuilderService := services.NewSetBuilderService(setBuilderRepo, notifRepo)
	handler := NewSetBuilderHandler(setBuilderService)

	r := gin.New()
	protected := r.Group("/api")
	protected.Use(coinTestAuthMiddleware())
	protected.POST("/set-builder/runs", handler.CreateRun)
	return r, db
}

func performSetBuilderCreateRunRequest(t *testing.T, router *gin.Engine, token string, body map[string]interface{}) *httptest.ResponseRecorder {
	t.Helper()
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/set-builder/runs", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func countRows(t *testing.T, db *gorm.DB, model interface{}) int64 {
	t.Helper()
	var count int64
	if err := db.Model(model).Count(&count).Error; err != nil {
		t.Fatalf("count rows: %v", err)
	}
	return count
}

func TestSetBuilderHandlerCreateRunRequiresAuth(t *testing.T) {
	router, _ := setupSetBuilderHandlerRouter(t)
	w := performSetBuilderCreateRunRequest(t, router, "", map[string]interface{}{"prompt": "All twelve Caesars"})
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without auth, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSetBuilderHandlerCreateRunRejectsBlankPrompt(t *testing.T) {
	router, db := setupSetBuilderHandlerRouter(t)
	token := makeCoinTestJWT(1)

	w := performSetBuilderCreateRunRequest(t, router, token, map[string]interface{}{"prompt": "   "})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for blank prompt, got %d: %s", w.Code, w.Body.String())
	}
	if got := countRows(t, db, &models.SetBuilderRun{}); got != 0 {
		t.Fatalf("blank prompt must not persist a run, got %d", got)
	}
}

// TestSetBuilderHandlerCreateRunPersistsQueuedRunWithoutCreatingSet is the primary
// regression for the correction plan's Phase 0 acceptance criterion: submitting an
// Agentic prompt must queue a run for the Python workflow and must never create a
// CoinSet, SetProposal, or CoinSetTarget directly.
func TestSetBuilderHandlerCreateRunPersistsQueuedRunWithoutCreatingSet(t *testing.T) {
	router, db := setupSetBuilderHandlerRouter(t)
	token := makeCoinTestJWT(7)

	w := performSetBuilderCreateRunRequest(t, router, token, map[string]interface{}{
		"prompt": "  All twelve Caesars from Augustus to Domitian  ",
	})
	if w.Code != http.StatusAccepted {
		t.Fatalf("expected 202 Accepted, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Run models.SetBuilderRun `json:"run"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Run.ID == 0 {
		t.Fatalf("expected persisted run id in response, got %#v", resp.Run)
	}
	if resp.Run.UserID != 7 {
		t.Fatalf("expected run scoped to authenticated user 7, got %d", resp.Run.UserID)
	}
	if resp.Run.Status != models.SetBuilderRunStatusQueued {
		t.Fatalf("expected queued status, got %q", resp.Run.Status)
	}
	if resp.Run.Prompt != "All twelve Caesars from Augustus to Domitian" {
		t.Fatalf("expected trimmed prompt persisted, got %q", resp.Run.Prompt)
	}

	var persisted models.SetBuilderRun
	if err := db.First(&persisted, resp.Run.ID).Error; err != nil {
		t.Fatalf("expected run persisted in db: %v", err)
	}
	if persisted.UserID != 7 || persisted.Status != models.SetBuilderRunStatusQueued {
		t.Fatalf("unexpected persisted run: %#v", persisted)
	}

	if got := countRows(t, db, &models.CoinSet{}); got != 0 {
		t.Fatalf("submitting a prompt must not create a CoinSet, got %d", got)
	}
	if got := countRows(t, db, &models.CoinSetTarget{}); got != 0 {
		t.Fatalf("submitting a prompt must not create CoinSetTarget rows, got %d", got)
	}
	if got := countRows(t, db, &models.SetProposal{}); got != 0 {
		t.Fatalf("submitting a prompt must not create a SetProposal before the workflow completes, got %d", got)
	}
}

func TestSetBuilderHandlerCreateRunIsScopedPerUser(t *testing.T) {
	router, db := setupSetBuilderHandlerRouter(t)

	wUserOne := performSetBuilderCreateRunRequest(t, router, makeCoinTestJWT(1), map[string]interface{}{"prompt": "All US silver dollars"})
	if wUserOne.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", wUserOne.Code, wUserOne.Body.String())
	}
	wUserTwo := performSetBuilderCreateRunRequest(t, router, makeCoinTestJWT(2), map[string]interface{}{"prompt": "All US silver dollars"})
	if wUserTwo.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", wUserTwo.Code, wUserTwo.Body.String())
	}

	var runs []models.SetBuilderRun
	if err := db.Order("id ASC").Find(&runs).Error; err != nil {
		t.Fatalf("list runs: %v", err)
	}
	if len(runs) != 2 {
		t.Fatalf("expected two independently queued runs (one per user), got %d", len(runs))
	}
	if runs[0].UserID != 1 || runs[1].UserID != 2 {
		t.Fatalf("expected runs scoped to their own submitting user, got %#v", runs)
	}
}
