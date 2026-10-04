package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/middleware"
	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"gorm.io/gorm"
)

const mcpRouteTestSecret = "mcp-route-test-secret"

var mcpRouteTestSequence atomic.Uint64

type mcpRouteFixture struct {
	db            *gorm.DB
	server        *httptest.Server
	settings      *services.SettingsService
	keys          map[string]string
	ownerUserID   uint
	ownerCoinID   uint
	foreignCoinID uint
	ownerLotID    uint
	foreignLotID  uint
}

type apiKeyRoundTripper struct {
	apiKey string
	base   http.RoundTripper
}

func (t apiKeyRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	cloned := request.Clone(request.Context())
	cloned.Header = request.Header.Clone()
	cloned.Header.Set("X-API-Key", t.apiKey)
	return t.base.RoundTrip(cloned)
}

func setupMCPRouteFixture(t *testing.T, rateLimit int) *mcpRouteFixture {
	t.Helper()
	gin.SetMode(gin.TestMode)
	dsn := fmt.Sprintf("file:mcp-route-%d?mode=memory&cache=shared", mcpRouteTestSequence.Add(1))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.AutoMigrate(
		&models.User{}, &models.ApiKey{}, &models.AppSetting{}, &models.Coin{}, &models.CoinImage{},
		&models.CoinReference{}, &models.Tag{}, &models.CoinSet{}, &models.CoinSetMembership{},
		&models.StorageLocation{}, &models.MintLocation{}, &models.AuctionLot{},
		&models.CoinCopilotThread{}, &models.CoinCopilotRun{}, &models.CoinCopilotCheckpoint{},
		&models.CoinCopilotEvent{}, &models.CoinCopilotResumeRequest{}, &models.DeepIdentificationJob{},
		&models.CoinCopilotDeepHandoff{},
	); err != nil {
		t.Fatal(err)
	}

	owner := models.User{ID: 71, Username: "mcp-owner", Email: "mcp-owner@example.test", PasswordHash: "hash", Role: models.RoleUser}
	foreign := models.User{ID: 72, Username: "mcp-foreign", Email: "mcp-foreign@example.test", PasswordHash: "hash", Role: models.RoleUser}
	if err := db.Create(&[]*models.User{&owner, &foreign}).Error; err != nil {
		t.Fatal(err)
	}
	ownerCoin := models.Coin{UserID: owner.ID, Name: "Owner coin"}
	ownerWishlist := models.Coin{UserID: owner.ID, Name: "Owner wishlist", IsWishlist: true}
	foreignCoin := models.Coin{UserID: foreign.ID, Name: "Foreign coin"}
	if err := db.Create(&[]*models.Coin{&ownerCoin, &ownerWishlist, &foreignCoin}).Error; err != nil {
		t.Fatal(err)
	}
	ownerLot := models.AuctionLot{UserID: owner.ID, Title: "Owner lot", NumisBidsURL: "https://example.test/owner"}
	foreignLot := models.AuctionLot{UserID: foreign.ID, Title: "Foreign lot", NumisBidsURL: "https://example.test/foreign"}
	if err := db.Create(&[]*models.AuctionLot{&ownerLot, &foreignLot}).Error; err != nil {
		t.Fatal(err)
	}

	settings := services.NewSettingsService(repository.NewSettingsRepository(db))
	for key, value := range map[string]string{
		services.SettingExternalToolServerEnabled: "true",
		services.SettingCoinCopilotEnabled:        "true",
		services.SettingAIProvider:                "anthropic",
		services.SettingAnthropicAPIKey:           "test",
	} {
		if err := settings.SetSetting(key, value); err != nil {
			t.Fatal(err)
		}
	}
	agent := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/api/copilot/capability" {
			http.NotFound(writer, request)
			return
		}
		_, _ = writer.Write([]byte(`{"supported":true}`))
	}))
	t.Cleanup(agent.Close)

	copilotSvc := services.NewCoinCopilotService(
		repository.NewCoinCopilotRepository(db), settings,
		services.NewAgentProxy(agent.URL, "internal", services.NewLogger(10)),
		services.NewInternalTokenService("01234567890123456789012345678901"),
		services.NewLogger(10), "http://api:8080",
	)
	coinRepo := repository.NewCoinRepository(db)
	mcpSvc := services.NewMCPService(
		services.NewCollectionToolsService(coinRepo, nil),
		coinRepo,
		repository.NewAuctionLotRepository(db),
		copilotSvc,
	)
	handler := NewMCPHandler(mcpSvc, services.NewLogger(10))

	keys := map[string]string{
		"read":      "ak_mcp_read",
		"copilot":   "ak_mcp_copilot",
		"malformed": "ak_mcp_malformed",
		"revoked":   "ak_mcp_revoked",
	}
	capabilities := map[string]string{
		"read": "read", "copilot": "read,copilot",
		"malformed": "read,copilotx", "revoked": "read",
	}
	now := time.Now()
	for name, plain := range keys {
		key := models.ApiKey{
			UserID: owner.ID, KeyHash: services.HashAPIKey(plain, mcpRouteTestSecret),
			KeyPrefix: plain, Name: name, Capabilities: capabilities[name],
		}
		if name == "revoked" {
			key.RevokedAt = &now
		}
		if err := db.Create(&key).Error; err != nil {
			t.Fatal(err)
		}
	}

	router := gin.New()
	router.POST(
		"/api/mcp",
		middleware.ExternalToolServerEnabled(settings),
		middleware.AuthRequired(mcpRouteTestSecret, repository.NewApiKeyRepository(db)),
		middleware.ExternalAPIKeyRateLimit(rateLimit, time.Minute),
		middleware.RequireCapability("read"),
		handler.Serve,
	)
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)
	return &mcpRouteFixture{
		db: db, server: server, settings: settings, keys: keys,
		ownerUserID: owner.ID,
		ownerCoinID: ownerCoin.ID, foreignCoinID: foreignCoin.ID,
		ownerLotID: ownerLot.ID, foreignLotID: foreignLot.ID,
	}
}

func connectMCPRoute(t *testing.T, fixture *mcpRouteFixture, keyName string) *mcp.ClientSession {
	t.Helper()
	client := mcp.NewClient(&mcp.Implementation{Name: "route-test-client", Version: "1"}, nil)
	httpClient := &http.Client{Transport: apiKeyRoundTripper{
		apiKey: fixture.keys[keyName],
		base:   http.DefaultTransport,
	}}
	session, err := client.Connect(context.Background(), &mcp.StreamableClientTransport{
		Endpoint:             fixture.server.URL + "/api/mcp",
		HTTPClient:           httpClient,
		MaxRetries:           -1,
		DisableStandaloneSSE: true,
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	return session
}

func callMCPTool(t *testing.T, session *mcp.ClientSession, name string, arguments any) *mcp.CallToolResult {
	t.Helper()
	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: name, Arguments: arguments})
	if err != nil {
		t.Fatalf("%s protocol error: %v", name, err)
	}
	return result
}

func TestMCPToolDiscoveryRequiresExactCopilotCapability(t *testing.T) {
	tests := []struct {
		keyName     string
		wantCopilot bool
	}{
		{keyName: "read"},
		{keyName: "copilot", wantCopilot: true},
		{keyName: "malformed"},
	}

	for _, tc := range tests {
		t.Run(tc.keyName, func(t *testing.T) {
			fixture := setupMCPRouteFixture(t, 100)
			session := connectMCPRoute(t, fixture, tc.keyName)
			result, err := session.ListTools(context.Background(), nil)
			if err != nil {
				t.Fatal(err)
			}
			names := make([]string, 0, len(result.Tools))
			for _, tool := range result.Tools {
				names = append(names, tool.Name)
			}
			sort.Strings(names)

			for _, readTool := range []string{
				"auction_counts", "collection_stats", "get_auction_lot", "get_coin",
				"list_auction_lots", "list_wishlist", "search_collection", "top_coins_by_value",
			} {
				if !containsString(names, readTool) {
					t.Fatalf("tools %v missing read tool %q", names, readTool)
				}
			}
			for _, copilotTool := range []string{
				"cancel_copilot_run", "get_copilot_run", "resume_copilot_run", "start_copilot_run",
			} {
				if got := containsString(names, copilotTool); got != tc.wantCopilot {
					t.Fatalf("tool %q present=%v, want %v; tools=%v", copilotTool, got, tc.wantCopilot, names)
				}
			}
			wantCount := 8
			if tc.wantCopilot {
				wantCount = 12
			}
			if len(names) != wantCount {
				t.Fatalf("tool count=%d, want %d: %v", len(names), wantCount, names)
			}
		})
	}
}

func TestMCPRouteInvokesEveryToolAndPreservesOwnerBoundary(t *testing.T) {
	fixture := setupMCPRouteFixture(t, 100)
	session := connectMCPRoute(t, fixture, "copilot")
	readCalls := []struct {
		name      string
		arguments any
	}{
		{name: "search_collection", arguments: map[string]any{"query": "Owner"}},
		{name: "get_coin", arguments: map[string]any{"coinId": fixture.ownerCoinID}},
		{name: "list_wishlist", arguments: map[string]any{}},
		{name: "collection_stats", arguments: map[string]any{}},
		{name: "top_coins_by_value", arguments: map[string]any{"limit": 1}},
		{name: "list_auction_lots", arguments: map[string]any{}},
		{name: "get_auction_lot", arguments: map[string]any{"auctionLotId": fixture.ownerLotID}},
		{name: "auction_counts", arguments: map[string]any{}},
	}
	for _, call := range readCalls {
		if result := callMCPTool(t, session, call.name, call.arguments); result.IsError {
			t.Fatalf("%s returned tool error: %+v", call.name, result.Content)
		}
	}

	for _, call := range []struct {
		name      string
		arguments any
	}{
		{name: "get_coin", arguments: map[string]any{"coinId": fixture.foreignCoinID}},
		{name: "get_auction_lot", arguments: map[string]any{"auctionLotId": fixture.foreignLotID}},
		{name: "get_copilot_run", arguments: map[string]any{"runId": "foreign-run"}},
		{name: "resume_copilot_run", arguments: map[string]any{
			"runId": "foreign-run", "answer": "continue",
			"expectedCheckpointVersion": 1, "idempotencyKey": "route-resume",
		}},
		{name: "cancel_copilot_run", arguments: map[string]any{"runId": "foreign-run"}},
	} {
		result := callMCPTool(t, session, call.name, call.arguments)
		encoded, err := json.Marshal(result)
		if err != nil {
			t.Fatal(err)
		}
		if !result.IsError || !bytes.Contains(encoded, []byte("resource not found")) {
			t.Fatalf("%s result=%s, want safe not-found tool error", call.name, encoded)
		}
		if bytes.Contains(encoded, []byte("Foreign")) || bytes.Contains(encoded, []byte("record not found")) {
			t.Fatalf("%s leaked internal or foreign data: %s", call.name, encoded)
		}
	}

	start := callMCPTool(t, session, "start_copilot_run", map[string]any{
		"goal": "Summarize my collection", "idempotencyKey": "route-start",
	})
	if start.IsError {
		t.Fatalf("start_copilot_run returned tool error: %+v", start.Content)
	}
	startJSON, err := json.Marshal(start.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}
	var started copilotRunEnvelope
	if err := json.Unmarshal(startJSON, &started); err != nil {
		t.Fatal(err)
	}
	if started.Run.ID == "" {
		t.Fatalf("start_copilot_run missing run ID: %s", startJSON)
	}
	if result := callMCPTool(t, session, "get_copilot_run", map[string]any{
		"runId": started.Run.ID,
	}); result.IsError {
		t.Fatalf("get_copilot_run returned tool error: %+v", result.Content)
	}
	if result := callMCPTool(t, session, "cancel_copilot_run", map[string]any{
		"runId": started.Run.ID,
	}); result.IsError {
		t.Fatalf("cancel_copilot_run returned tool error: %+v", result.Content)
	}
	var cancelled models.CoinCopilotRun
	if err := fixture.db.First(&cancelled, "id = ?", started.Run.ID).Error; err != nil {
		t.Fatal(err)
	}
	if cancelled.Status != models.CopilotRunCancelled {
		t.Fatalf("cancelled run status=%s, want %s", cancelled.Status, models.CopilotRunCancelled)
	}

	deadline := time.Now().UTC().Add(time.Hour)
	pausedAt := time.Now().UTC()
	pausedThread := models.CoinCopilotThread{
		ID: "cct_route_paused", UserID: fixture.ownerUserID, Title: "Paused route test",
	}
	pausedRun := models.CoinCopilotRun{
		ID: "ccr_route_paused", ThreadID: pausedThread.ID, UserID: fixture.ownerUserID,
		Status: models.CopilotRunPaused, Goal: "Continue route test",
		StartIdempotencyKeyHash: "route-paused-start-key",
		StartRequestFingerprint: "route-paused-fingerprint",
		ExecutionID:             "cce_route_paused", CheckpointVersion: 1,
		PausedAt: &pausedAt, ResumeDeadline: &deadline,
		MaxIterations: 5, MaxToolCalls: 10, MaxConcurrentTools: 1,
		HardTimeoutSeconds: 60, MaxPersistedToolResultBytes: 4096,
	}
	checkpointState := `{"schema_version":1,"messages":[],"plan":[],"completed_tools":[],"next_action":"clarification","counters":{}}`
	checkpoint := models.CoinCopilotCheckpoint{
		RunID: pausedRun.ID, ThreadID: pausedThread.ID, UserID: fixture.ownerUserID,
		ExecutionID: pausedRun.ExecutionID, Version: 1, StateJSON: checkpointState,
		StateDigest: services.CopilotCheckpointDigest(checkpointState),
	}
	if err := fixture.db.Create(&pausedThread).Error; err != nil {
		t.Fatal(err)
	}
	if err := fixture.db.Create(&pausedRun).Error; err != nil {
		t.Fatal(err)
	}
	if err := fixture.db.Create(&checkpoint).Error; err != nil {
		t.Fatal(err)
	}
	resumed := callMCPTool(t, session, "resume_copilot_run", map[string]any{
		"runId": pausedRun.ID, "answer": "Continue",
		"expectedCheckpointVersion": 1, "idempotencyKey": "route-resume-success",
	})
	if resumed.IsError {
		t.Fatalf("resume_copilot_run returned tool error: %+v", resumed.Content)
	}
	if err := fixture.db.First(&pausedRun, "id = ?", pausedRun.ID).Error; err != nil {
		t.Fatal(err)
	}
	if pausedRun.Status != models.CopilotRunQueued || pausedRun.CheckpointVersion != 2 {
		t.Fatalf("resumed run status=%s checkpoint=%d, want queued/2", pausedRun.Status, pausedRun.CheckpointVersion)
	}

	writeResult, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "propose_update", Arguments: map[string]any{"coinId": fixture.ownerCoinID},
	})
	if err == nil && (writeResult == nil || !writeResult.IsError) {
		t.Fatalf("write-shaped tool unexpectedly callable: %+v", writeResult)
	}
}

func TestMCPRouteRejectsDisabledInvalidRevokedAndOversizedRequests(t *testing.T) {
	fixture := setupMCPRouteFixture(t, 100)
	request := func(key string, body []byte) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(http.MethodPost, "/api/mcp", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json, text/event-stream")
		if key != "" {
			req.Header.Set("X-API-Key", key)
		}
		recorder := httptest.NewRecorder()
		fixture.server.Config.Handler.ServeHTTP(recorder, req)
		return recorder
	}

	if recorder := request("", []byte(`{}`)); recorder.Code != http.StatusUnauthorized {
		t.Fatalf("missing key status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if recorder := request("ak_invalid", []byte(`{}`)); recorder.Code != http.StatusUnauthorized {
		t.Fatalf("invalid key status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if recorder := request(fixture.keys["revoked"], []byte(`{}`)); recorder.Code != http.StatusUnauthorized {
		t.Fatalf("revoked key status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	oversized := bytes.Repeat([]byte("x"), mcpMaxRequestBodyBytes+1)
	if recorder := request(fixture.keys["read"], oversized); recorder.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversized status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if err := fixture.settings.SetSetting(services.SettingExternalToolServerEnabled, "false"); err != nil {
		t.Fatal(err)
	}
	if recorder := request(fixture.keys["read"], []byte(`{}`)); recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("disabled status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestMCPRouteAppliesPerKeyRateLimit(t *testing.T) {
	fixture := setupMCPRouteFixture(t, 1)
	for attempt := 1; attempt <= 2; attempt++ {
		req := httptest.NewRequest(http.MethodPost, "/api/mcp", strings.NewReader(`{}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-API-Key", fixture.keys["read"])
		recorder := httptest.NewRecorder()
		fixture.server.Config.Handler.ServeHTTP(recorder, req)
		if attempt == 2 && recorder.Code != http.StatusTooManyRequests {
			t.Fatalf("second request status=%d body=%s", recorder.Code, recorder.Body.String())
		}
	}
}

func TestMCPToolSurfaceContainsNoMutationOrArbitraryExecutionTools(t *testing.T) {
	fixture := setupMCPRouteFixture(t, 100)
	session := connectMCPRoute(t, fixture, "copilot")
	result, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, tool := range result.Tools {
		for _, forbidden := range []string{
			"create", "update", "delete", "commit", "write", "import", "sync",
			"convert", "filesystem", "shell", "database", "http_request",
		} {
			if strings.Contains(tool.Name, forbidden) {
				t.Fatalf("forbidden MCP tool registered: %s", tool.Name)
			}
		}
	}
	if len(result.Tools) != 12 {
		t.Fatalf("tools=%d, want 12", len(result.Tools))
	}
}

func containsString(values []string, target string) bool {
	i := sort.SearchStrings(values, target)
	if i < len(values) && values[i] == target {
		return true
	}
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
