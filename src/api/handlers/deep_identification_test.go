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
		&models.CatalogRegistry{},
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
	coinRefSvc := services.NewCoinReferenceService(repository.NewCoinReferenceRepository(db), repository.NewCatalogRegistryRepository(db))
	proposalSvc := services.NewDeepIdentificationProposalService(repo, coinRepo, coinSvc, quickCaptureSvc, coinRefSvc)

	handler := NewDeepIdentificationHandler(svc, settingsSvc, services.NewLogger(10)).WithProposalSupport(proposalSvc)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("userId", userID)
		c.Next()
	})
	router.GET("/api/deep-identification/capability", handler.Capability)
	router.POST("/api/deep-identification/jobs", handler.CreateJob)
	router.GET("/api/deep-identification/jobs", handler.ListJobs)
	router.GET("/api/deep-identification/jobs/:id", handler.GetJob)
	router.DELETE("/api/deep-identification/jobs/:id", handler.DeleteJob)
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

// TestDeepIdentificationHandler_CreateJob_AtCapacityWithDifferentFingerprintReturns409
// is the regression test for the wrong-job-returned defect: with the
// default MaxActivePerUser=1, a second submission whose image bytes (and
// therefore InputFingerprint) genuinely differ from the user's first
// in-flight job must be refused with 409 job_at_capacity, never handed the
// first job's envelope as if it were an answer for the second submission.
// This is an approved breaking change to a shipped endpoint (see
// .squad/decisions/inbox/cassius-job-at-capacity.md): the endpoint used to
// return 200 reused=true with the unrelated job in this exact scenario.
func TestDeepIdentificationHandler_CreateJob_AtCapacityWithDifferentFingerprintReturns409(t *testing.T) {
	deps := setupDeepIdentificationHandlerTest(t, 1, true)

	firstBody, firstContentType := multipartWithImages(t, nil, true, true, 0)
	firstReq := httptest.NewRequest(http.MethodPost, "/api/deep-identification/jobs", firstBody)
	firstReq.Header.Set("Content-Type", firstContentType)
	firstRec := httptest.NewRecorder()
	deps.router.ServeHTTP(firstRec, firstReq)
	if firstRec.Code != http.StatusAccepted {
		t.Fatalf("expected first submission to be accepted with 202, got %d: %s", firstRec.Code, firstRec.Body.String())
	}

	// A distinct image (different marker bytes -> different sha256 ->
	// different InputFingerprint) submitted while the first job is still
	// queued/running and MaxActivePerUser=1 (the default) is at capacity.
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	obversePart, err := writer.CreateFormFile("obverse", "obverse.png")
	if err != nil {
		t.Fatalf("create obverse part: %v", err)
	}
	if _, err := obversePart.Write(deepTestPNGVariant(t, 0xAA)); err != nil {
		t.Fatalf("write obverse bytes: %v", err)
	}
	reversePart, err := writer.CreateFormFile("reverse", "reverse.png")
	if err != nil {
		t.Fatalf("create reverse part: %v", err)
	}
	if _, err := reversePart.Write(deepTestPNGVariant(t, 0xBB)); err != nil {
		t.Fatalf("write reverse bytes: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/deep-identification/jobs", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec := httptest.NewRecorder()
	deps.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409 for a non-matching submission at capacity, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp["code"] != "job_at_capacity" {
		t.Fatalf("expected code=job_at_capacity, got %v", resp["code"])
	}
	if _, hasJob := resp["job"]; hasJob {
		t.Fatalf("409 job_at_capacity response must not include an unrelated job envelope, got %+v", resp)
	}

	var jobCount int64
	deps.db.Model(&models.DeepIdentificationJob{}).Count(&jobCount)
	if jobCount != 1 {
		t.Fatalf("expected the refused submission to create no job row, got %d job rows", jobCount)
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

// F1: the capability probe reflects the feature flag so normal-user UI can
// hide/disable the Deep Analysis entry point, while the backend remains
// authoritative (job creation is independently gated above).
func TestDeepIdentificationHandler_Capability_ReflectsFeatureFlag(t *testing.T) {
	for _, tc := range []struct {
		name    string
		enabled bool
	}{
		{"enabled", true},
		{"disabled", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			deps := setupDeepIdentificationHandlerTest(t, 1, tc.enabled)
			req := httptest.NewRequest(http.MethodGet, "/api/deep-identification/capability", nil)
			rec := httptest.NewRecorder()
			deps.router.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
			}
			var resp struct {
				Enabled   bool     `json:"enabled"`
				Providers []string `json:"providers"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
				t.Fatalf("decode capability response: %v", err)
			}
			if resp.Enabled != tc.enabled {
				t.Fatalf("expected enabled=%v, got %v", tc.enabled, resp.Enabled)
			}
			if len(resp.Providers) != 2 || resp.Providers[0] != "nomisma" || resp.Providers[1] != "numista" {
				t.Fatalf("expected default eligible providers, got %#v", resp.Providers)
			}
		})
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

func TestDeepIdentificationHandler_DeleteJob_Returns204ForOwnerTerminalJob(t *testing.T) {
	deps := setupDeepIdentificationHandlerTest(t, 1, true)
	job := &models.DeepIdentificationJob{
		UserID:           1,
		Source:           models.DeepJobSourceIntake,
		Status:           models.DeepJobStatusCompleted,
		InputFingerprint: fmt.Sprintf("fp-delete-%d", time.Now().UnixNano()),
		ExpiresAt:        time.Now().Add(24 * time.Hour),
	}
	if err := deps.db.Create(job).Error; err != nil {
		t.Fatalf("seed job: %v", err)
	}
	if err := deps.db.Create(&models.DeepIdentificationEvent{JobID: job.ID, UserID: 1, Seq: 1, Type: models.DeepEventProgress, PayloadJSON: "{}"}).Error; err != nil {
		t.Fatalf("seed event: %v", err)
	}

	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/deep-identification/jobs/%d", job.ID), nil)
	rec := httptest.NewRecorder()
	deps.router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rec.Code, rec.Body.String())
	}
	var count int64
	if err := deps.db.Model(&models.DeepIdentificationJob{}).Where("id = ?", job.ID).Count(&count).Error; err != nil {
		t.Fatalf("count jobs: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected deleted job row, found count=%d", count)
	}
}

func TestDeepIdentificationHandler_DeleteJob_Returns409ForNonTerminal(t *testing.T) {
	deps := setupDeepIdentificationHandlerTest(t, 1, true)
	job := &models.DeepIdentificationJob{
		UserID:           1,
		Source:           models.DeepJobSourceIntake,
		Status:           models.DeepJobStatusRunning,
		InputFingerprint: fmt.Sprintf("fp-delete-nonterminal-%d", time.Now().UnixNano()),
		ExpiresAt:        time.Now().Add(24 * time.Hour),
	}
	if err := deps.db.Create(job).Error; err != nil {
		t.Fatalf("seed job: %v", err)
	}
	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/deep-identification/jobs/%d", job.ID), nil)
	rec := httptest.NewRecorder()
	deps.router.ServeHTTP(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestDeepIdentificationHandler_DeleteJob_Returns404ForNonOwner(t *testing.T) {
	deps := setupDeepIdentificationHandlerTest(t, 1, true)
	other := models.User{ID: 2, Username: "other", Email: "other@example.com", PasswordHash: "x"}
	if err := deps.db.Create(&other).Error; err != nil {
		t.Fatalf("seed other user: %v", err)
	}
	job := &models.DeepIdentificationJob{
		UserID:           2,
		Source:           models.DeepJobSourceIntake,
		Status:           models.DeepJobStatusCompleted,
		InputFingerprint: fmt.Sprintf("fp-delete-other-%d", time.Now().UnixNano()),
		ExpiresAt:        time.Now().Add(24 * time.Hour),
	}
	if err := deps.db.Create(job).Error; err != nil {
		t.Fatalf("seed job: %v", err)
	}
	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/deep-identification/jobs/%d", job.ID), nil)
	rec := httptest.NewRecorder()
	deps.router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

// seedDeepHandlerIntakeJobWithProposal seeds a completed intake job (no
// source coin) with an already-accepted denomination+workingTitle proposal,
// ready to POST .../apply against, for T072/T073 handler-level tests.
func seedDeepHandlerIntakeJobWithProposal(t *testing.T, db *gorm.DB, userID uint) uint {
	t.Helper()
	proposal := `{"schemaVersion":1,"fields":{
		"denomination":{"proposed":"Denarius","confidence":0.8,"ownerEdited":false,"ownerValue":null,"accepted":true},
		"workingTitle":{"proposed":"Trajan Denarius","confidence":0.8,"ownerEdited":false,"ownerValue":null,"accepted":null}
	}}`
	job := &models.DeepIdentificationJob{
		UserID:           userID,
		Source:           models.DeepJobSourceIntake,
		Status:           models.DeepJobStatusCompleted,
		InputFingerprint: fmt.Sprintf("fp-handler-%d-%d", time.Now().UnixNano(), atomic.AddInt64(&deepHandlerTestDBCounter, 1)),
		ExpiresAt:        time.Now().Add(90 * 24 * time.Hour),
		ReportJSON:       `{"schemaVersion":1,"narrative":"n","coverage":[]}`,
		ProposalJSON:     proposal,
	}
	if err := db.Create(job).Error; err != nil {
		t.Fatalf("seed intake job: %v", err)
	}
	return job.ID
}

// T072/T073: POST .../apply with target=wishlist plumbs through to
// DeepIdentificationProposalService.Apply and creates a wishlist coin.
func TestDeepIdentificationHandler_ApplyProposal_WishlistTargetCreatesCoin(t *testing.T) {
	deps := setupDeepIdentificationHandlerTest(t, 1, true)
	jobID := seedDeepHandlerIntakeJobWithProposal(t, deps.db, 1)

	reqBody, _ := json.Marshal(map[string]any{"target": "wishlist"})
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/deep-identification/jobs/%d/apply", jobID), bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	deps.router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for wishlist apply, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp deepApplyResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal apply response: %v", err)
	}
	if resp.CoinID == nil {
		t.Fatal("expected coinId in wishlist apply response")
	}
	var coin models.Coin
	if err := deps.db.First(&coin, *resp.CoinID).Error; err != nil {
		t.Fatal(err)
	}
	if !coin.IsWishlist {
		t.Fatal("expected the created coin to have IsWishlist=true")
	}
	if coin.Name != "Trajan Denarius" {
		t.Fatalf("expected derived name from workingTitle, got %q", coin.Name)
	}
}

// T073: unknown apply targets are rejected exactly as today, through the
// closed switch (binding:"oneof" plus normalizeDeepApplyTarget).
func TestDeepIdentificationHandler_ApplyProposal_UnknownTargetReturns400(t *testing.T) {
	deps := setupDeepIdentificationHandlerTest(t, 1, true)
	jobID := seedDeepHandlerIntakeJobWithProposal(t, deps.db, 1)

	reqBody, _ := json.Marshal(map[string]any{"target": "not-a-real-target"})
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/deep-identification/jobs/%d/apply", jobID), bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	deps.router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for unknown apply target, got %d: %s", rec.Code, rec.Body.String())
	}

	var coinCount int64
	deps.db.Model(&models.Coin{}).Count(&coinCount)
	if coinCount != 0 {
		t.Fatal("expected no coin created for a rejected unknown target")
	}
}
