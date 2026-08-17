package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
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

// fakeDeepNumistaClient is an in-memory stand-in for services.NumistaClient
// used only by these handler tests - no live HTTP is ever reached
// (T054: no-network CI requirement).
type fakeDeepNumistaClient struct {
	searchCalls atomic.Int32
	candidates  []models.NumistaCandidate
	err         error
}

func (f *fakeDeepNumistaClient) Search(_ context.Context, _ string, _ int) ([]models.NumistaCandidate, error) {
	f.searchCalls.Add(1)
	if f.err != nil {
		return nil, f.err
	}
	return f.candidates, nil
}

func (f *fakeDeepNumistaClient) Detail(_ context.Context, id int) (models.NumistaCandidate, error) {
	if f.err != nil {
		return models.NumistaCandidate{}, f.err
	}
	return models.NumistaCandidate{ID: id, Title: "Test Coin"}, nil
}

// fakeDeepNomismaClient is an in-memory stand-in for services.NomismaClient.
type fakeDeepNomismaClient struct {
	searchCalls atomic.Int32
	candidates  []services.NomismaCandidate
	kind        services.NomismaErrorKind
	err         error
}

func (f *fakeDeepNomismaClient) Search(_ context.Context, _ string, _ int) ([]services.NomismaCandidate, services.NomismaErrorKind, error) {
	f.searchCalls.Add(1)
	return f.candidates, f.kind, f.err
}

// fakeDeepOCREClient is an in-memory stand-in for services.OCREClient.
type fakeDeepOCREClient struct {
	searchCalls atomic.Int32
	candidates  []services.OCRECandidate
	kind        services.OCREErrorKind
	err         error
}

func (f *fakeDeepOCREClient) Search(_ context.Context, _ services.OCREQueryParams, _ int) ([]services.OCRECandidate, services.OCREErrorKind, error) {
	f.searchCalls.Add(1)
	return f.candidates, f.kind, f.err
}

var deepProviderToolsTestDBCounter int64

func setupDeepProviderToolsTest(t *testing.T, numistaBudget int) (*gin.Engine, *services.InternalTokenService, *fakeDeepNumistaClient, *fakeDeepNomismaClient) {
	router, tokenSvc, numista, nomisma, _ := setupDeepProviderToolsTestWithOCRE(t, numistaBudget, 3)
	return router, tokenSvc, numista, nomisma
}

func setupDeepProviderToolsTestWithOCRE(t *testing.T, numistaBudget, ocreBudget int) (*gin.Engine, *services.InternalTokenService, *fakeDeepNumistaClient, *fakeDeepNomismaClient, *fakeDeepOCREClient) {
	return setupDeepProviderToolsTestWithOCREFlag(t, numistaBudget, ocreBudget, true)
}

func setupDeepProviderToolsTestWithOCREFlag(t *testing.T, numistaBudget, ocreBudget int, ocreEnabled bool) (*gin.Engine, *services.InternalTokenService, *fakeDeepNumistaClient, *fakeDeepNomismaClient, *fakeDeepOCREClient) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	dsn := fmt.Sprintf("file:deep_provider_tools_%d_%d?mode=memory&cache=shared", time.Now().UnixNano(), atomic.AddInt64(&deepProviderToolsTestDBCounter, 1))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&models.AppSetting{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := db.Create(&models.AppSetting{
		Key: services.SettingDeepIdentificationNumistaCallBudget, Value: fmt.Sprintf("%d", numistaBudget),
	}).Error; err != nil {
		t.Fatalf("seed numista budget: %v", err)
	}
	if err := db.Create(&models.AppSetting{
		Key: services.SettingDeepIdentificationOCRECallBudget, Value: fmt.Sprintf("%d", ocreBudget),
	}).Error; err != nil {
		t.Fatalf("seed ocre budget: %v", err)
	}
	if err := db.Create(&models.AppSetting{
		Key: services.SettingDeepIdentificationOCREEnabled, Value: strconv.FormatBool(ocreEnabled),
	}).Error; err != nil {
		t.Fatalf("seed ocre enabled: %v", err)
	}
	settingsSvc := services.NewSettingsService(repository.NewSettingsRepository(db))

	numistaClient := &fakeDeepNumistaClient{candidates: []models.NumistaCandidate{{ID: 1, Title: "Denarius"}}}
	nomismaClient := &fakeDeepNomismaClient{candidates: []services.NomismaCandidate{{URI: "http://nomisma.org/id/ar", Label: "AR"}}}
	ocreClient := &fakeDeepOCREClient{candidates: []services.OCRECandidate{{
		TypeURI: "https://numismatics.org/ocre/id/ric.2.hdn.39b", Label: "RIC II Hadrian 39b",
		MatchedFields: []string{"ruler:hadrian"}, Confidence: 0.86, Explanation: "Matched ruler hadrian.",
	}}}
	ocreCache := services.NewOCRECache()
	tokenSvc := services.NewInternalTokenService("test-secret")
	budgets := services.NewDeepProviderBudgetTracker()
	handler := NewDeepProviderToolsHandler(numistaClient, nomismaClient, ocreClient, ocreCache, settingsSvc, budgets, services.NewLogger(10))

	router := gin.New()
	authed := router.Group("/api/internal/tools")
	authed.Use(func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		tokenString := authHeader
		if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			tokenString = authHeader[7:]
		}
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization required"})
			return
		}
		userID, jobID, err := tokenSvc.VerifyForJob(tokenString)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			return
		}
		c.Set("userId", userID)
		c.Set("deepJobId", jobID)
		c.Next()
	})
	authed.POST("/numista_search", handler.NumistaSearch)
	authed.POST("/numista_detail", handler.NumistaDetail)
	authed.POST("/nomisma_search", handler.NomismaSearch)
	authed.POST("/ocre_search", handler.OCRESearch)

	return router, tokenSvc, numistaClient, nomismaClient, ocreClient
}

func TestDeepProviderTools_UnauthenticatedRequestReturns401(t *testing.T) {
	router, _, _, _ := setupDeepProviderToolsTest(t, 4)

	req := httptest.NewRequest(http.MethodPost, "/api/internal/tools/numista_search", bytes.NewBufferString(`{"query":"denarius"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestDeepProviderTools_ForeignOrTamperedTokenReturns401(t *testing.T) {
	router, tokenSvc, _, _ := setupDeepProviderToolsTest(t, 4)

	token, err := tokenSvc.MintForJob(1, 5)
	if err != nil {
		t.Fatalf("MintForJob: %v", err)
	}
	tampered := token[:len(token)-1] + "z"

	req := httptest.NewRequest(http.MethodPost, "/api/internal/tools/numista_search", bytes.NewBufferString(`{"query":"denarius"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tampered)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for tampered/foreign token, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestDeepProviderTools_NumistaSearchHappyPath(t *testing.T) {
	router, tokenSvc, numistaClient, _ := setupDeepProviderToolsTest(t, 4)
	token, _ := tokenSvc.MintForJob(1, 5)

	req := httptest.NewRequest(http.MethodPost, "/api/internal/tools/numista_search", bytes.NewBufferString(`{"query":"denarius","limit":5}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp["status"] != "ok" {
		t.Fatalf("expected status=ok, got %v", resp["status"])
	}
	if numistaClient.searchCalls.Load() != 1 {
		t.Fatalf("expected exactly one upstream call, got %d", numistaClient.searchCalls.Load())
	}
}

func TestDeepProviderTools_NumistaSearchBudgetExceededReturnsQuotaLimited(t *testing.T) {
	router, tokenSvc, numistaClient, _ := setupDeepProviderToolsTest(t, 2)
	token, _ := tokenSvc.MintForJob(1, 5)

	call := func() (int, map[string]any) {
		req := httptest.NewRequest(http.MethodPost, "/api/internal/tools/numista_search", bytes.NewBufferString(`{"query":"denarius"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		var resp map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		return rec.Code, resp
	}

	for i := 0; i < 2; i++ {
		code, resp := call()
		if code != http.StatusOK || resp["status"] != "ok" {
			t.Fatalf("call %d: expected 200/ok, got %d/%v", i, code, resp["status"])
		}
	}

	code, resp := call()
	if code != http.StatusOK {
		t.Fatalf("expected 200 (quota_limited is a typed status, not an HTTP error), got %d", code)
	}
	if resp["status"] != "quota_limited" {
		t.Fatalf("expected status=quota_limited after budget exhausted, got %v", resp["status"])
	}
	if numistaClient.searchCalls.Load() != 2 {
		t.Fatalf("expected the upstream client to be called only twice (3rd call short-circuited), got %d", numistaClient.searchCalls.Load())
	}
}

func TestDeepProviderTools_NumistaDetailUnconfiguredMapsToTypedStatus(t *testing.T) {
	router, tokenSvc, numistaClient, _ := setupDeepProviderToolsTest(t, 4)
	numistaClient.err = &services.NumistaError{Kind: services.NumistaErrorUnconfigured}
	token, _ := tokenSvc.MintForJob(1, 5)

	req := httptest.NewRequest(http.MethodPost, "/api/internal/tools/numista_detail", bytes.NewBufferString(`{"id":123}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp["status"] != "unconfigured" {
		t.Fatalf("expected status=unconfigured, got %v", resp["status"])
	}
}

func TestDeepProviderTools_NomismaSearchIndependentBudgetFromNumista(t *testing.T) {
	router, tokenSvc, numistaClient, nomismaClient := setupDeepProviderToolsTest(t, 1)
	token, _ := tokenSvc.MintForJob(1, 5)

	// Exhaust the (budget=1) Numista budget for this job.
	req := httptest.NewRequest(http.MethodPost, "/api/internal/tools/numista_search", bytes.NewBufferString(`{"query":"denarius"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	// Nomisma (fixed budget=3, independent tracker key) must still work.
	nreq := httptest.NewRequest(http.MethodPost, "/api/internal/tools/nomisma_search", bytes.NewBufferString(`{"query":"Roma"}`))
	nreq.Header.Set("Content-Type", "application/json")
	nreq.Header.Set("Authorization", "Bearer "+token)
	nrec := httptest.NewRecorder()
	router.ServeHTTP(nrec, nreq)
	if nrec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", nrec.Code, nrec.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(nrec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp["status"] != "ok" {
		t.Fatalf("expected nomisma status=ok despite exhausted numista budget, got %v", resp["status"])
	}
	if numistaClient.searchCalls.Load() != 1 || nomismaClient.searchCalls.Load() != 1 {
		t.Fatalf("expected one call each, got numista=%d nomisma=%d", numistaClient.searchCalls.Load(), nomismaClient.searchCalls.Load())
	}
}

// TestDeepProviderTools_NomismaSearchInvalidRequestNotReportedAsUpstreamFailure
// is the Task G regression test: an over-length/malformed query is a
// client-side bug (Go's NomismaClient.Search never issues the upstream HTTP
// call for it), and MUST be surfaced distinctly from a real upstream outage
// ("unavailable") so the Python node never misreports "our bug" as "Nomisma
// failed".
func TestDeepProviderTools_NomismaSearchInvalidRequestNotReportedAsUpstreamFailure(t *testing.T) {
	router, tokenSvc, _, nomismaClient := setupDeepProviderToolsTest(t, 3)
	nomismaClient.kind = services.NomismaErrorInvalidRequest
	nomismaClient.err = fmt.Errorf("invalid Nomisma query")
	token, _ := tokenSvc.MintForJob(1, 5)

	nreq := httptest.NewRequest(http.MethodPost, "/api/internal/tools/nomisma_search", bytes.NewBufferString(`{"query":"anything"}`))
	nreq.Header.Set("Content-Type", "application/json")
	nreq.Header.Set("Authorization", "Bearer "+token)
	nrec := httptest.NewRecorder()
	router.ServeHTTP(nrec, nreq)
	if nrec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", nrec.Code, nrec.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(nrec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp["status"] != "invalid_request" {
		t.Fatalf("expected status=invalid_request (never unavailable), got %v", resp["status"])
	}
}

// ---- Feature 345 (T013/T034): OCRE internal tool handler tests ----

func ocreCall(t *testing.T, router *gin.Engine, token, body string) (int, map[string]any) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/internal/tools/ocre_search", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	var resp map[string]any
	if rec.Body.Len() > 0 {
		_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	}
	return rec.Code, resp
}

const ocreHadrianBody = `{"ruler":"hadrian","denomination":"denarius","mint":"rome","limit":5}`

func TestDeepProviderTools_OCRESearchOK(t *testing.T) {
	router, tokenSvc, _, _, ocreClient := setupDeepProviderToolsTestWithOCRE(t, 4, 3)
	token, _ := tokenSvc.MintForJob(1, 5)
	code, resp := ocreCall(t, router, token, ocreHadrianBody)
	if code != http.StatusOK || resp["status"] != "ok" {
		t.Fatalf("expected 200/ok, got %d/%v", code, resp["status"])
	}
	if resp["attribution"] != "Coin type data: Online Coins of the Roman Empire (OCRE), American Numismatic Society \u2014 ODbL 1.0." {
		t.Fatalf("unexpected attribution: %v", resp["attribution"])
	}
	cands, _ := resp["candidates"].([]any)
	if len(cands) != 1 {
		t.Fatalf("expected one candidate, got %v", resp["candidates"])
	}
	if ocreClient.searchCalls.Load() != 1 {
		t.Fatalf("expected exactly one upstream call, got %d", ocreClient.searchCalls.Load())
	}
}

func TestDeepProviderTools_OCRESearchEmpty(t *testing.T) {
	router, tokenSvc, _, _, ocreClient := setupDeepProviderToolsTestWithOCRE(t, 4, 3)
	ocreClient.candidates = nil
	ocreClient.kind = services.OCREErrorNoMatch
	token, _ := tokenSvc.MintForJob(1, 5)
	code, resp := ocreCall(t, router, token, ocreHadrianBody)
	if code != http.StatusOK || resp["status"] != "empty" {
		t.Fatalf("expected 200/empty, got %d/%v", code, resp["status"])
	}
}

func TestDeepProviderTools_OCRESearchInvalidResponse(t *testing.T) {
	router, tokenSvc, _, _, ocreClient := setupDeepProviderToolsTestWithOCRE(t, 4, 3)
	ocreClient.candidates = nil
	ocreClient.kind = services.OCREErrorInvalidResponse
	token, _ := tokenSvc.MintForJob(1, 5)
	code, resp := ocreCall(t, router, token, ocreHadrianBody)
	if code != http.StatusOK {
		t.Fatalf("expected 200 (never a 5xx for an upstream problem), got %d", code)
	}
	if resp["status"] != "invalid_response" {
		t.Fatalf("expected status=invalid_response, got %v", resp["status"])
	}
}

func TestDeepProviderTools_OCRESearchUnavailableNever5xx(t *testing.T) {
	router, tokenSvc, _, _, ocreClient := setupDeepProviderToolsTestWithOCRE(t, 4, 3)
	ocreClient.candidates = nil
	ocreClient.kind = services.OCREErrorUnavailable
	token, _ := tokenSvc.MintForJob(1, 5)
	code, resp := ocreCall(t, router, token, ocreHadrianBody)
	if code != http.StatusOK || resp["status"] != "unavailable" {
		t.Fatalf("expected 200/unavailable, got %d/%v", code, resp["status"])
	}
}

func TestDeepProviderTools_OCRESearchCancelled(t *testing.T) {
	router, tokenSvc, _, _, ocreClient := setupDeepProviderToolsTestWithOCRE(t, 4, 3)
	ocreClient.candidates = nil
	ocreClient.kind = services.OCREErrorCancelled
	token, _ := tokenSvc.MintForJob(1, 5)
	code, resp := ocreCall(t, router, token, ocreHadrianBody)
	if code != http.StatusOK || resp["status"] != "cancelled" {
		t.Fatalf("expected 200/cancelled, got %d/%v", code, resp["status"])
	}
}

func TestDeepProviderTools_OCRESearchTimeoutMapsToWireTimeout(t *testing.T) {
	// F2: a client-reported OCREErrorTimeout must surface as the wire status
	// "timeout" (never "unavailable"), so the Python node maps it to timed_out.
	router, tokenSvc, _, _, ocreClient := setupDeepProviderToolsTestWithOCRE(t, 4, 3)
	ocreClient.candidates = nil
	ocreClient.kind = services.OCREErrorTimeout
	token, _ := tokenSvc.MintForJob(1, 5)
	code, resp := ocreCall(t, router, token, ocreHadrianBody)
	if code != http.StatusOK || resp["status"] != "timeout" {
		t.Fatalf("expected 200/timeout, got %d/%v", code, resp["status"])
	}
}

func TestDeepProviderTools_OCRESearchFlagOffZeroCall(t *testing.T) {
	// F1: defense in depth — with the OCRE flag OFF, a direct job-token
	// invocation must NOT reach the client. Expect a typed, non-contributing
	// "unavailable" and exactly zero upstream calls (FR-004/FR-016, SC-004).
	router, tokenSvc, _, _, ocreClient := setupDeepProviderToolsTestWithOCREFlag(t, 4, 3, false)
	token, _ := tokenSvc.MintForJob(1, 5)
	code, resp := ocreCall(t, router, token, ocreHadrianBody)
	if code != http.StatusOK || resp["status"] != "unavailable" {
		t.Fatalf("expected 200/unavailable while flag off, got %d/%v", code, resp["status"])
	}
	if cands, _ := resp["candidates"].([]any); len(cands) != 0 {
		t.Fatalf("expected zero candidates while flag off, got %v", resp["candidates"])
	}
	if ocreClient.searchCalls.Load() != 0 {
		t.Fatalf("expected ZERO upstream calls while OCRE flag is off, got %d", ocreClient.searchCalls.Load())
	}
}

func TestDeepProviderTools_OCRESearchQuotaLimitedIndependentBudget(t *testing.T) {
	router, tokenSvc, _, _, ocreClient := setupDeepProviderToolsTestWithOCRE(t, 4, 1)
	token, _ := tokenSvc.MintForJob(1, 5)
	// First call (distinct params) consumes the (budget=1) OCRE budget.
	if code, resp := ocreCall(t, router, token, `{"ruler":"hadrian","limit":5}`); code != http.StatusOK || resp["status"] != "ok" {
		t.Fatalf("first call: expected 200/ok, got %d/%v", code, resp["status"])
	}
	// Second call with *different* params misses the cache and is over
	// budget: typed quota_limited, HTTP 200, no new upstream call.
	code, resp := ocreCall(t, router, token, `{"ruler":"trajan","limit":5}`)
	if code != http.StatusOK || resp["status"] != "quota_limited" {
		t.Fatalf("expected 200/quota_limited, got %d/%v", code, resp["status"])
	}
	if ocreClient.searchCalls.Load() != 1 {
		t.Fatalf("expected the over-budget call to short-circuit, got %d calls", ocreClient.searchCalls.Load())
	}
}

func TestDeepProviderTools_OCRESearchMissingTokenReturns401(t *testing.T) {
	router, _, _, _, _ := setupDeepProviderToolsTestWithOCRE(t, 4, 3)
	code, _ := ocreCall(t, router, "", ocreHadrianBody)
	if code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for a missing job token, got %d", code)
	}
}

func TestDeepProviderTools_OCRESearchUnparseableBodyReturns400(t *testing.T) {
	router, tokenSvc, _, _, _ := setupDeepProviderToolsTestWithOCRE(t, 4, 3)
	token, _ := tokenSvc.MintForJob(1, 5)
	code, _ := ocreCall(t, router, token, `{not valid json`)
	if code != http.StatusBadRequest {
		t.Fatalf("expected 400 for an unparseable body, got %d", code)
	}
}

func TestDeepProviderTools_OCRESearchNoSignalNoCall(t *testing.T) {
	router, tokenSvc, _, _, ocreClient := setupDeepProviderToolsTestWithOCRE(t, 4, 3)
	token, _ := tokenSvc.MintForJob(1, 5)
	// Only a material + legend: no type-bearing signal → empty, zero calls.
	code, resp := ocreCall(t, router, token, `{"material":"silver","legend_tokens":["cos"]}`)
	if code != http.StatusOK || resp["status"] != "empty" {
		t.Fatalf("expected 200/empty for a signal-less request, got %d/%v", code, resp["status"])
	}
	if ocreClient.searchCalls.Load() != 0 {
		t.Fatalf("expected zero upstream calls for a signal-less request, got %d", ocreClient.searchCalls.Load())
	}
}

func TestDeepProviderTools_OCRESearchCacheHitSkipsUpstream(t *testing.T) {
	router, tokenSvc, _, _, ocreClient := setupDeepProviderToolsTestWithOCRE(t, 4, 5)
	token, _ := tokenSvc.MintForJob(1, 5)
	if code, resp := ocreCall(t, router, token, ocreHadrianBody); code != http.StatusOK || resp["status"] != "ok" {
		t.Fatalf("first call: expected 200/ok, got %d/%v", code, resp["status"])
	}
	if code, resp := ocreCall(t, router, token, ocreHadrianBody); code != http.StatusOK || resp["status"] != "ok" {
		t.Fatalf("second call: expected 200/ok cache hit, got %d/%v", code, resp["status"])
	}
	if ocreClient.searchCalls.Load() != 1 {
		t.Fatalf("expected the second identical call to hit the cache (one upstream call total), got %d", ocreClient.searchCalls.Load())
	}
}
