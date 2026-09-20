package handlers

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"sort"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/middleware"
	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestCoinCopilotCallbackRoutesRemainExactReadOnlySet(t *testing.T) {
	source, err := os.ReadFile("../routes_internal.go")
	if err != nil {
		t.Fatal(err)
	}
	blockPattern := regexp.MustCompile(`(?s)copilot := r\.Group\("/api/internal/copilot/tools"\).*?\n\t}`)
	block := blockPattern.Find(source)
	if block == nil {
		t.Fatal("Coin Copilot callback route group not found")
	}
	routePattern := regexp.MustCompile(`copilot\.POST\("/([^"]+)"`)
	matches := routePattern.FindAllSubmatch(block, -1)
	routes := make([]string, 0, len(matches))
	for _, match := range matches {
		routes = append(routes, string(match[1]))
	}
	sort.Strings(routes)
	want := []string{"collection_summary", "deep_analysis_handoff", "get_coin", "search_my_collection", "top_coins_by_value"}
	if len(routes) != len(want) {
		t.Fatalf("callback routes=%v, want %v", routes, want)
	}
	for index := range want {
		if routes[index] != want[index] {
			t.Fatalf("callback routes=%v, want %v", routes, want)
		}
	}
	for _, forbidden := range []string{
		"market_search", "auction_search", "price_trends", "similar_lots",
		"write", "update", "delete", "approval", "deep_identification",
		"fetch", "shell", "filesystem", "database",
	} {
		if bytes.Contains(block, []byte(forbidden)) {
			t.Fatalf("callback route block contains forbidden capability %q", forbidden)
		}
	}
}

func TestCoinCopilotInternalToolBindsExecutionOwnerAndCallID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}

	if err := db.AutoMigrate(
		&models.User{}, &models.Coin{}, &models.CollectionUpdateProposal{}, &models.AppSetting{},
		&models.CoinCopilotThread{}, &models.CoinCopilotRun{}, &models.CoinCopilotCheckpoint{},
		&models.CoinCopilotEvent{}, &models.CoinCopilotResumeRequest{},
	); err != nil {
		t.Fatal(err)
	}

	_ = db.Create(&models.User{ID: 7, Username: "owner", Email: "owner@example.test", PasswordHash: "x"}).Error
	_ = db.Create(&models.User{ID: 8, Username: "other", Email: "other@example.test", PasswordHash: "x"}).Error
	_ = db.Create(&models.Coin{ID: 9, UserID: 7, Name: "Owned"}).Error
	_ = db.Create(&models.Coin{ID: 10, UserID: 8, Name: "Foreign"}).Error
	copilotRepo := repository.NewCoinCopilotRepository(db)
	thread := &models.CoinCopilotThread{ID: "cct_internal", UserID: 7, Title: "Test"}
	run := &models.CoinCopilotRun{
		ID: "ccr_internal", ThreadID: thread.ID, UserID: 7, Status: models.CopilotRunRunning,
		Goal: "test", StartIdempotencyKeyHash: "key", StartRequestFingerprint: "fingerprint",
		ExecutionID: "cce_internal", MaxIterations: 8, MaxToolCalls: 12, MaxConcurrentTools: 1,
		HardTimeoutSeconds: 120, MaxPersistedToolResultBytes: 32768,
		ExecutionStartedAt: func() *time.Time { now := time.Now().UTC(); return &now }(),
	}
	if err := copilotRepo.CreateRun(thread, run); err != nil {
		t.Fatal(err)
	}
	tokenSvc := services.NewInternalTokenService("01234567890123456789012345678901")
	settings := services.NewSettingsService(repository.NewSettingsRepository(db))
	copilotSvc := services.NewCoinCopilotService(copilotRepo, settings, services.NewAgentProxy("", "", services.NewLogger(10)), tokenSvc, services.NewLogger(10), "")
	collectionSvc := services.NewCollectionToolsService(repository.NewCoinRepository(db), repository.NewCollectionUpdateRepository(db))
	handler := NewCoinCopilotInternalToolsHandler(collectionSvc, copilotSvc, services.NewLogger(10))
	router := gin.New()
	router.POST("/get_coin", middleware.CoinCopilotExecutionTokenRequired(tokenSvc, "get_coin"), handler.GetCoin)
	router.POST("/search", middleware.CoinCopilotExecutionTokenRequired(tokenSvc, "search_my_collection"), handler.SearchMyCollection)
	token, err := tokenSvc.MintForCopilotExecution(7, run.ID, run.ExecutionID, []string{"get_coin", "search_my_collection"}, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	call := func() *httptest.ResponseRecorder {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/get_coin", bytes.NewBufferString(`{"tool_call_id":"call_1","coin_id":10}`))
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(recorder, request)
		return recorder
	}
	if recorder := call(); recorder.Code != http.StatusNotFound {
		t.Fatalf("foreign coin status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if recorder := call(); recorder.Code != http.StatusNotFound {
		t.Fatalf("failed call should be retryable, status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	success := func(token, callID string) *httptest.ResponseRecorder {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/search", bytes.NewBufferString(`{"tool_call_id":"`+callID+`","query":"Owned"}`))
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(recorder, request)
		return recorder
	}
	if recorder := success(token, "call_2"); recorder.Code != http.StatusOK {
		t.Fatalf("owned coin status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if recorder := success(token, "call_2"); recorder.Code != http.StatusConflict {
		t.Fatalf("duplicate completed call status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	wrongExecution, err := tokenSvc.MintForCopilotExecution(7, run.ID, "cce_wrong", []string{"search_my_collection"}, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if recorder := success(wrongExecution, "call_3"); recorder.Code != http.StatusConflict {
		t.Fatalf("wrong execution status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	wrongOwner, err := tokenSvc.MintForCopilotExecution(8, run.ID, run.ExecutionID, []string{"search_my_collection"}, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if recorder := success(wrongOwner, "call_4"); recorder.Code != http.StatusConflict {
		t.Fatalf("wrong owner status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if err := db.Model(&models.CoinCopilotRun{}).Where("id = ?", run.ID).Update("tool_call_count", run.MaxToolCalls).Error; err != nil {
		t.Fatal(err)
	}
	if recorder := success(token, "call_5"); recorder.Code != http.StatusConflict {
		t.Fatalf("exhausted budget status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestDeepAnalysisHandoffAcceptsOnlyCanonicalExecutionToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tokenSvc := services.NewInternalTokenService("01234567890123456789012345678901")
	router := gin.New()
	router.POST(
		"/deep_analysis_handoff",
		middleware.CoinCopilotExecutionTokenRequired(tokenSvc, "deep_analysis_handoff"),
		func(c *gin.Context) { c.Status(http.StatusNoContent) },
	)

	valid, err := tokenSvc.MintForCopilotExecution(
		7, "ccr_handoff", "cce_handoff", []string{"deep_analysis_handoff"}, time.Minute,
	)
	if err != nil {
		t.Fatalf("mint canonical execution token: %v", err)
	}
	internalService, err := tokenSvc.Mint(7)
	if err != nil {
		t.Fatal(err)
	}
	deepJob, err := tokenSvc.MintForJob(7, 314)
	if err != nil {
		t.Fatal(err)
	}
	wrongTool, err := tokenSvc.MintForCopilotExecution(
		7, "ccr_handoff", "cce_handoff", []string{"get_coin"}, time.Minute,
	)
	if err != nil {
		t.Fatal(err)
	}
	revoked, err := tokenSvc.MintForCopilotExecution(
		7, "ccr_handoff", "cce_revoked", []string{"deep_analysis_handoff"}, time.Minute,
	)
	if err != nil {
		t.Fatal(err)
	}
	tokenSvc.RevokeCopilotExecution("cce_revoked")

	for name, token := range map[string]string{
		"missing":                   "",
		"user JWT":                  "header.payload.signature",
		"internal service token":    internalService,
		"Deep job token":            deepJob,
		"wrong tool execution":      wrongTool,
		"revoked execution token":   revoked,
		"malformed execution token": "not-a-token",
	} {
		t.Run(name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPost, "/deep_analysis_handoff", nil)
			if token != "" {
				request.Header.Set("Authorization", "Bearer "+token)
			}
			router.ServeHTTP(recorder, request)
			if recorder.Code != http.StatusUnauthorized {
				t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
			}
		})
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/deep_analysis_handoff", nil)
	request.Header.Set("Authorization", "Bearer "+valid)
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("canonical execution token status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestFeature362HandoffNondisclosureBodiesAreCanonicalAndCauseIndependent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name    string
		outcome string
		want    string
	}{
		{"unknown id", "not_eligible", `{"outcome":"not_eligible","reason":null}`},
		{"foreign job", "not_eligible", `{"outcome":"not_eligible","reason":null}`},
		{"legacy unbound intake", "not_eligible", `{"outcome":"not_eligible","reason":null}`},
		{"unknown source", "not_eligible", `{"outcome":"not_eligible","reason":null}`},
		{"arbitrary unbound job", "not_eligible", `{"outcome":"not_eligible","reason":null}`},
		{"deleted unbound coin", "not_eligible", `{"outcome":"not_eligible","reason":null}`},
		{"deleted discarded or promoted unbound draft", "not_eligible", `{"outcome":"not_eligible","reason":null}`},
		{"previously bound target disappeared", "target_unavailable", `{"outcome":"target_unavailable","reason":null}`},
		{"previously bound draft promoted", "target_unavailable", `{"outcome":"target_unavailable","reason":null}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			context, _ := gin.CreateTestContext(recorder)
			writeDeepAnalysisHandoffResult(context, services.DeepAnalysisHandoffResult{Outcome: test.outcome})
			if recorder.Code != http.StatusOK {
				t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
			}
			if got := recorder.Body.String(); got != test.want {
				t.Fatalf("public body=%q want exact canonical bytes %q", got, test.want)
			}
			for _, forbidden := range []string{
				"job", "target", "source", "owner", "deleted", "promoted",
				"legacy", "unknown", "internal", "diagnostic",
			} {
				if bytes.Contains(recorder.Body.Bytes(), []byte(`"`+forbidden+`"`)) {
					t.Fatalf("public body leaked internal cause/metadata %q: %s", forbidden, recorder.Body.String())
				}
			}
		})
	}
}
