package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// deepHandlerTestDBCounter avoids in-memory SQLite shared-cache DSN
// collisions between fast-running tests (see the same issue documented in
// services/deep_identification_service_test.go).
var deepHandlerTestDBCounter int64

func deepTestPNGBytes(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.RGBA{R: 10, G: 20, B: 30, A: 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode png fixture: %v", err)
	}
	return buf.Bytes()
}

type deepHandlerTestDeps struct {
	router      *gin.Engine
	db          *gorm.DB
	svc         *services.DeepIdentificationService
	coinRepo    *repository.CoinRepository
	proposalSvc *services.DeepIdentificationProposalService
}

func setupDeepIdentificationHandlerTest(t *testing.T, userID uint, enabled bool) deepHandlerTestDeps {
	t.Helper()
	gin.SetMode(gin.TestMode)
	dsn := fmt.Sprintf("file:deep_identification_handler_%d_%d?mode=memory&cache=shared", time.Now().UnixNano(), atomic.AddInt64(&deepHandlerTestDBCounter, 1))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(
		&models.User{}, &models.Coin{}, &models.CoinImage{}, &models.CoinReference{}, &models.ValueSnapshot{}, &models.CoinJournal{}, &models.AppSetting{},
		&models.DeepIdentificationJob{}, &models.DeepIdentificationEvent{},
		&models.DeepIdentificationProviderRun{}, &models.DeepIdentificationArtifact{},
		&models.QuickCaptureDraft{}, &models.QuickCaptureDraftImage{}, &models.QuickCaptureDraftReference{}, &models.DraftLifecycleEvent{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := db.Create(&models.User{ID: userID, Username: fmt.Sprintf("user-%d", userID), Email: fmt.Sprintf("user-%d@example.com", userID), PasswordHash: "x"}).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	uploadDir := t.TempDir()
	imageRepo := repository.NewImageRepository(db)
	imageSvc := services.NewImageService(imageRepo, uploadDir)
	settingsSvc := services.NewSettingsService(repository.NewSettingsRepository(db))
	if enabled {
		if err := db.Create(&models.AppSetting{Key: services.SettingDeepIdentificationEnabled, Value: "true"}).Error; err != nil {
			t.Fatalf("seed enabled flag: %v", err)
		}
	}
	repo := repository.NewDeepIdentificationRepository(db)
	svc := services.NewDeepIdentificationService(repo, imageRepo, imageSvc, settingsSvc, services.NewLogger(10), uploadDir)

	coinRepo := repository.NewCoinRepository(db)
	notifRepo := repository.NewNotificationRepository(db)
	socialRepo := repository.NewSocialRepository(db)
	userRepo := repository.NewUserRepository(db)
	notifSvc := services.NewNotificationService(notifRepo, socialRepo, userRepo, services.NewPushoverService(settingsSvc, services.NewLogger(10)), services.NewLogger(10))
	coinSvc := services.NewCoinService(coinRepo, notifSvc)
	quickCaptureRepo := repository.NewQuickCaptureRepository(db)
	quickCaptureSvc := services.NewQuickCaptureService(quickCaptureRepo, uploadDir).WithCoinValidation(coinSvc)
	proposalSvc := services.NewDeepIdentificationProposalService(repo, coinRepo, coinSvc, quickCaptureSvc)

	handler := NewDeepIdentificationHandler(svc, settingsSvc, services.NewLogger(10)).WithProposalSupport(proposalSvc)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("userId", userID)
		c.Next()
	})
	router.POST("/api/deep-identification/jobs", handler.CreateJob)
	router.GET("/api/deep-identification/jobs", handler.ListJobs)
	router.GET("/api/deep-identification/jobs/:id", handler.GetJob)
	router.GET("/api/deep-identification/jobs/:id/events", handler.StreamEvents)
	router.POST("/api/deep-identification/jobs/:id/cancel", handler.Cancel)
	router.POST("/api/deep-identification/jobs/:id/retry", handler.Retry)
	router.PATCH("/api/deep-identification/jobs/:id/proposal", handler.UpdateProposal)
	router.POST("/api/deep-identification/jobs/:id/apply", handler.ApplyProposal)

	return deepHandlerTestDeps{router: router, db: db, svc: svc, coinRepo: coinRepo, proposalSvc: proposalSvc}
}

// deepTestPNGVariant returns a valid PNG with a unique trailing marker byte
// appended (ignored by magic-byte detection but keeps otherwise-identical
// image fixtures distinguishable by content hash across roles/hints in a
// single request).
func deepTestPNGVariant(t *testing.T, marker byte) []byte {
	t.Helper()
	return append(deepTestPNGBytes(t), marker)
}

func multipartWithImages(t *testing.T, fields map[string]string, obverse, reverse bool, hintCount int) (*bytes.Buffer, string) {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for k, v := range fields {
		if err := writer.WriteField(k, v); err != nil {
			t.Fatalf("write field %s: %v", k, err)
		}
	}
	writeImage := func(field string, marker byte) {
		part, err := writer.CreateFormFile(field, field+".png")
		if err != nil {
			t.Fatalf("create form file %s: %v", field, err)
		}
		if _, err := part.Write(deepTestPNGVariant(t, marker)); err != nil {
			t.Fatalf("write %s bytes: %v", field, err)
		}
	}
	if obverse {
		writeImage("obverse", 0x01)
	}
	if reverse {
		writeImage("reverse", 0x02)
	}
	for i := 0; i < hintCount; i++ {
		part, err := writer.CreateFormFile("hints", fmt.Sprintf("hint-%d.png", i))
		if err != nil {
			t.Fatalf("create hint form file: %v", err)
		}
		if _, err := part.Write(deepTestPNGVariant(t, byte(0x10+i))); err != nil {
			t.Fatalf("write hint bytes: %v", err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}
	return &body, writer.FormDataContentType()
}

func TestDeepIdentificationHandler_CreateJob_MissingReverseReturns422(t *testing.T) {
	deps := setupDeepIdentificationHandlerTest(t, 1, true)
	body, contentType := multipartWithImages(t, nil, true, false, 0)

	req := httptest.NewRequest(http.MethodPost, "/api/deep-identification/jobs", body)
	req.Header.Set("Content-Type", contentType)
	rec := httptest.NewRecorder()
	deps.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp["code"] != "missing_reverse" {
		t.Fatalf("expected code=missing_reverse, got %v", resp["code"])
	}
}

func TestDeepIdentificationHandler_CreateJob_MissingObverseReturns422(t *testing.T) {
	deps := setupDeepIdentificationHandlerTest(t, 1, true)
	body, contentType := multipartWithImages(t, nil, false, true, 0)

	req := httptest.NewRequest(http.MethodPost, "/api/deep-identification/jobs", body)
	req.Header.Set("Content-Type", contentType)
	rec := httptest.NewRecorder()
	deps.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp["code"] != "missing_obverse" {
		t.Fatalf("expected code=missing_obverse, got %v", resp["code"])
	}
}

func TestDeepIdentificationHandler_CreateJob_HappyPathReturns202(t *testing.T) {
	deps := setupDeepIdentificationHandlerTest(t, 1, true)
	body, contentType := multipartWithImages(t, map[string]string{"notes": "test notes"}, true, true, 1)

	req := httptest.NewRequest(http.MethodPost, "/api/deep-identification/jobs", body)
	req.Header.Set("Content-Type", contentType)
	rec := httptest.NewRecorder()
	deps.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", rec.Code, rec.Body.String())
	}
	var env deepJobEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("unmarshal envelope: %v", err)
	}
	if env.Job.Status != string(models.DeepJobStatusQueued) {
		t.Fatalf("expected queued status, got %s", env.Job.Status)
	}
	if env.Reused {
		t.Fatal("first submission should not be reused")
	}
}

func TestDeepIdentificationHandler_CreateJob_DisabledReturns403(t *testing.T) {
	deps := setupDeepIdentificationHandlerTest(t, 1, false)
	body, contentType := multipartWithImages(t, nil, true, true, 0)

	req := httptest.NewRequest(http.MethodPost, "/api/deep-identification/jobs", body)
	req.Header.Set("Content-Type", contentType)
	rec := httptest.NewRecorder()
	deps.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestDeepIdentificationHandler_CreateJob_HintDuplicatesObverseReturns422(t *testing.T) {
	deps := setupDeepIdentificationHandlerTest(t, 1, true)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	obversePart, err := writer.CreateFormFile("obverse", "obverse.png")
	if err != nil {
		t.Fatalf("create obverse part: %v", err)
	}
	shared := deepTestPNGVariant(t, 0x01)
	if _, err := obversePart.Write(shared); err != nil {
		t.Fatalf("write obverse bytes: %v", err)
	}
	reversePart, err := writer.CreateFormFile("reverse", "reverse.png")
	if err != nil {
		t.Fatalf("create reverse part: %v", err)
	}
	if _, err := reversePart.Write(deepTestPNGVariant(t, 0x02)); err != nil {
		t.Fatalf("write reverse bytes: %v", err)
	}
	hintPart, err := writer.CreateFormFile("hints", "hint-0.png")
	if err != nil {
		t.Fatalf("create hint part: %v", err)
	}
	// Same bytes as the obverse upload: mislabeled coin-face image.
	if _, err := hintPart.Write(shared); err != nil {
		t.Fatalf("write hint bytes: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/deep-identification/jobs", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec := httptest.NewRecorder()
	deps.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp["code"] != "hint_image_in_coin_role" {
		t.Fatalf("expected code=hint_image_in_coin_role, got %v", resp["code"])
	}
}

func TestDeepIdentificationHandler_ListJobs_StatusFilter(t *testing.T) {
	deps := setupDeepIdentificationHandlerTest(t, 1, true)
	body, contentType := multipartWithImages(t, nil, true, true, 0)
	req := httptest.NewRequest(http.MethodPost, "/api/deep-identification/jobs", body)
	req.Header.Set("Content-Type", contentType)
	rec := httptest.NewRecorder()
	deps.router.ServeHTTP(rec, req)
	var env deepJobEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("unmarshal envelope: %v", err)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/deep-identification/jobs?status=queued", nil)
	listRec := httptest.NewRecorder()
	deps.router.ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", listRec.Code, listRec.Body.String())
	}
	var listResp deepJobListResponse
	if err := json.Unmarshal(listRec.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("unmarshal list response: %v", err)
	}
	if len(listResp.Jobs) != 1 || listResp.Jobs[0].ID != env.Job.ID {
		t.Fatalf("expected exactly the queued job to be returned, got %+v", listResp.Jobs)
	}

	emptyReq := httptest.NewRequest(http.MethodGet, "/api/deep-identification/jobs?status=completed", nil)
	emptyRec := httptest.NewRecorder()
	deps.router.ServeHTTP(emptyRec, emptyReq)
	var emptyResp deepJobListResponse
	if err := json.Unmarshal(emptyRec.Body.Bytes(), &emptyResp); err != nil {
		t.Fatalf("unmarshal empty list response: %v", err)
	}
	if len(emptyResp.Jobs) != 0 {
		t.Fatalf("expected no completed jobs, got %+v", emptyResp.Jobs)
	}
}

func TestDeepIdentificationHandler_GetJob_CrossUserReturns404(t *testing.T) {
	deps := setupDeepIdentificationHandlerTest(t, 1, true)
	if err := deps.db.Create(&models.User{ID: 2, Username: "other", Email: "other@example.com", PasswordHash: "x"}).Error; err != nil {
		t.Fatalf("create other user: %v", err)
	}
	otherJob := &models.DeepIdentificationJob{
		UserID: 2, Source: models.DeepJobSourceIntake,
		InputFingerprint: "other-user-fp",
		ExpiresAt:        time.Now().Add(90 * 24 * time.Hour),
	}
	if err := deps.db.Create(otherJob).Error; err != nil {
		t.Fatalf("seed other user job: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/deep-identification/jobs/%d", otherJob.ID), nil)
	rec := httptest.NewRecorder()
	deps.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for cross-user job access, got %d: %s", rec.Code, rec.Body.String())
	}
}

// TestDeepIdentificationHandler_CreateJob_ForeignOwnedCoinIdReturns404 covers
// T091's create-time cross-user path, which is distinct from GetJob's
// cross-user check above: it exercises a request where `coinId` names a
// coin owned by a *different* user, not an existing deep-identification job.
func TestDeepIdentificationHandler_CreateJob_ForeignOwnedCoinIdReturns404(t *testing.T) {
	deps := setupDeepIdentificationHandlerTest(t, 1, true)
	if err := deps.db.Create(&models.User{ID: 2, Username: "coin-owner", Email: "coin-owner@example.com", PasswordHash: "x"}).Error; err != nil {
		t.Fatalf("create coin owner: %v", err)
	}
	otherCoin := &models.Coin{UserID: 2, Name: "Someone Else's Coin"}
	if err := deps.db.Create(otherCoin).Error; err != nil {
		t.Fatalf("seed other user's coin: %v", err)
	}

	body, contentType := multipartWithImages(t, map[string]string{"coinId": fmt.Sprintf("%d", otherCoin.ID)}, true, true, 0)
	req := httptest.NewRequest(http.MethodPost, "/api/deep-identification/jobs", body)
	req.Header.Set("Content-Type", contentType)
	rec := httptest.NewRecorder()
	deps.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for a foreign-owned coinId on create, got %d: %s", rec.Code, rec.Body.String())
	}
	var jobCount int64
	deps.db.Model(&models.DeepIdentificationJob{}).Count(&jobCount)
	if jobCount != 0 {
		t.Fatalf("expected no job row created for a foreign-owned coinId, got %d", jobCount)
	}
}

// TestDeepIdentificationHandler_CancelRetry_CrossUserReturns404 covers the
// remaining T091 cross-user surfaces: cancel and retry of a job owned by a
// different user must both 404, exactly like GetJob, never leaking the
// job's existence or its current status to a non-owner.
func TestDeepIdentificationHandler_CancelRetry_CrossUserReturns404(t *testing.T) {
	deps := setupDeepIdentificationHandlerTest(t, 1, true)
	if err := deps.db.Create(&models.User{ID: 2, Username: "other", Email: "other@example.com", PasswordHash: "x"}).Error; err != nil {
		t.Fatalf("create other user: %v", err)
	}
	otherJob := &models.DeepIdentificationJob{
		UserID: 2, Source: models.DeepJobSourceIntake,
		InputFingerprint: "other-user-cancel-retry-fp",
		ExpiresAt:        time.Now().Add(90 * 24 * time.Hour),
	}
	if err := deps.db.Create(otherJob).Error; err != nil {
		t.Fatalf("seed other user job: %v", err)
	}

	cancelReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/deep-identification/jobs/%d/cancel", otherJob.ID), nil)
	cancelRec := httptest.NewRecorder()
	deps.router.ServeHTTP(cancelRec, cancelReq)
	if cancelRec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for cross-user cancel, got %d: %s", cancelRec.Code, cancelRec.Body.String())
	}

	retryReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/deep-identification/jobs/%d/retry", otherJob.ID), nil)
	retryRec := httptest.NewRecorder()
	deps.router.ServeHTTP(retryRec, retryReq)
	if retryRec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for cross-user retry, got %d: %s", retryRec.Code, retryRec.Body.String())
	}
}

func TestDeepIdentificationHandler_GetJob_NotBlockedByFeatureFlagOff(t *testing.T) {
	// Create the job while the feature is enabled, then simulate the flag
	// being turned off afterward (FR-008: GET of an already-running/queued
	// job must not be blocked by a later flag flip).
	deps := setupDeepIdentificationHandlerTest(t, 1, true)
	body, contentType := multipartWithImages(t, nil, true, true, 0)
	req := httptest.NewRequest(http.MethodPost, "/api/deep-identification/jobs", body)
	req.Header.Set("Content-Type", contentType)
	rec := httptest.NewRecorder()
	deps.router.ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("setup: expected 202, got %d: %s", rec.Code, rec.Body.String())
	}
	var env deepJobEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("unmarshal envelope: %v", err)
	}

	if err := deps.db.Model(&models.AppSetting{}).
		Where("key = ?", services.SettingDeepIdentificationEnabled).
		Update("value", "false").Error; err != nil {
		t.Fatalf("disable flag: %v", err)
	}

	getReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/deep-identification/jobs/%d", env.Job.ID), nil)
	getRec := httptest.NewRecorder()
	deps.router.ServeHTTP(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("expected GET to remain available after flag flip, got %d: %s", getRec.Code, getRec.Body.String())
	}
}

func settleDeepJobCompleted(t *testing.T, db *gorm.DB, jobID uint) {
	t.Helper()
	// Mirrors repository.SettleTerminal's own column set (including
	// active_key, which the production terminal path always updates) so
	// this test fixture doesn't leave the row looking "active" for
	// duplicate-submit reuse purposes.
	if err := db.Model(&models.DeepIdentificationJob{}).Where("id = ?", jobID).
		Updates(map[string]interface{}{
			"status":       models.DeepJobStatusCompleted,
			"completed_at": time.Now(),
			"active_key":   gorm.Expr("id"),
		}).Error; err != nil {
		t.Fatalf("settle job %d completed: %v", jobID, err)
	}
}

func TestDeepIdentificationHandler_RetryLineageAndDepthCap(t *testing.T) {
	deps := setupDeepIdentificationHandlerTest(t, 1, true)
	body, contentType := multipartWithImages(t, nil, true, true, 0)
	req := httptest.NewRequest(http.MethodPost, "/api/deep-identification/jobs", body)
	req.Header.Set("Content-Type", contentType)
	rec := httptest.NewRecorder()
	deps.router.ServeHTTP(rec, req)
	var env deepJobEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("unmarshal envelope: %v", err)
	}
	currentID := env.Job.ID

	for depth := 1; depth <= services.MaxDeepIdentificationRetryDepth; depth++ {
		settleDeepJobCompleted(t, deps.db, currentID)
		retryReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/deep-identification/jobs/%d/retry", currentID), nil)
		retryRec := httptest.NewRecorder()
		deps.router.ServeHTTP(retryRec, retryReq)
		if retryRec.Code != http.StatusAccepted {
			t.Fatalf("retry at depth %d: expected 202, got %d: %s", depth, retryRec.Code, retryRec.Body.String())
		}
		var retryEnv deepJobEnvelope
		if err := json.Unmarshal(retryRec.Body.Bytes(), &retryEnv); err != nil {
			t.Fatalf("unmarshal retry envelope: %v", err)
		}
		if retryEnv.Job.RetryOfJobID == nil || *retryEnv.Job.RetryOfJobID != currentID {
			t.Fatalf("retry at depth %d: expected retryOfJobId=%d, got %v", depth, currentID, retryEnv.Job.RetryOfJobID)
		}
		currentID = retryEnv.Job.ID
	}

	// One more retry beyond the cap (3) is rejected with 409.
	settleDeepJobCompleted(t, deps.db, currentID)
	finalRetryReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/deep-identification/jobs/%d/retry", currentID), nil)
	finalRetryRec := httptest.NewRecorder()
	deps.router.ServeHTTP(finalRetryRec, finalRetryReq)
	if finalRetryRec.Code != http.StatusConflict {
		t.Fatalf("expected 409 beyond retry depth cap, got %d: %s", finalRetryRec.Code, finalRetryRec.Body.String())
	}
}

func TestDeepIdentificationHandler_CancelVsCompleteReturnsSettledState(t *testing.T) {
	deps := setupDeepIdentificationHandlerTest(t, 1, true)
	body, contentType := multipartWithImages(t, nil, true, true, 0)
	req := httptest.NewRequest(http.MethodPost, "/api/deep-identification/jobs", body)
	req.Header.Set("Content-Type", contentType)
	rec := httptest.NewRecorder()
	deps.router.ServeHTTP(rec, req)
	var env deepJobEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("unmarshal envelope: %v", err)
	}

	// Simulate the job having already completed naturally (won the race)
	// before the cancel request arrives.
	settleDeepJobCompleted(t, deps.db, env.Job.ID)

	cancelReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/deep-identification/jobs/%d/cancel", env.Job.ID), nil)
	cancelRec := httptest.NewRecorder()
	deps.router.ServeHTTP(cancelRec, cancelReq)
	if cancelRec.Code != http.StatusConflict {
		t.Fatalf("expected 409 for cancel-after-complete race, got %d: %s", cancelRec.Code, cancelRec.Body.String())
	}
	var cancelEnv deepJobEnvelope
	if err := json.Unmarshal(cancelRec.Body.Bytes(), &cancelEnv); err != nil {
		t.Fatalf("unmarshal cancel envelope: %v", err)
	}
	if cancelEnv.Job.Status != string(models.DeepJobStatusCompleted) {
		t.Fatalf("expected settled state 'completed' reported, got %s", cancelEnv.Job.Status)
	}
}
