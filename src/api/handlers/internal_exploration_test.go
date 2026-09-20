package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
)

type explorationDeciderStub struct {
	request services.ExplorationDecisionRequest
	err     error
}

func (s *explorationDeciderStub) DecideBrowserExploration(_ context.Context, req services.ExplorationDecisionRequest) (*services.ExplorationDecisionResponse, error) {
	s.request = req
	if s.err != nil {
		return nil, s.err
	}
	return &services.ExplorationDecisionResponse{
		SchemaVersion: services.ExplorationDecisionSchemaVersion,
		Action:        "finish", Rationale: "done", SuspectedFindings: []services.ExplorationModelTriage{},
		Usage: services.ExplorationModelUsage{InputTokens: 1, OutputTokens: 1},
	}, nil
}

func validExplorationBody(t *testing.T) []byte {
	t.Helper()
	value := services.ExplorationDecisionRequest{
		SchemaVersion: services.ExplorationDecisionSchemaVersion,
		RunID:         "aibr_20260918T120000Z_012345abcdef", StepID: "step-001",
		Provider: "anthropic", Model: "test-model", Workflow: "edit-one-field",
		Goal: "finish", AllowedActions: []string{"finish"}, AllowedRoutes: []string{"/coins/1/edit"},
		Observations: []services.ExplorationObservation{},
	}
	body, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return body
}

func callExploration(t *testing.T, enabled bool, token, header string, decider ExplorationDecider, body []byte) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	RegisterInternalExplorationRoute(r, enabled, token, decider)
	req := httptest.NewRequest(http.MethodPost, "/api/internal/browser-exploration/decide", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if header != "" {
		req.Header.Set("Authorization", header)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestInternalExplorationDisabledAndBearerAuthentication(t *testing.T) {
	stub := &explorationDeciderStub{}
	if got := callExploration(t, false, "secret", "Bearer secret", stub, validExplorationBody(t)).Code; got != http.StatusNotFound {
		t.Fatalf("disabled route status=%d", got)
	}
	for _, header := range []string{"", "Bearer wrong", "Basic secret"} {
		if got := callExploration(t, true, "secret", header, stub, validExplorationBody(t)).Code; got != http.StatusUnauthorized {
			t.Fatalf("header %q status=%d", header, got)
		}
	}
}

func TestInternalExplorationValidatesAndMapsStableErrors(t *testing.T) {
	stub := &explorationDeciderStub{}
	if got := callExploration(t, true, "secret", "Bearer secret", stub, []byte(`{"unknown":true}`)).Code; got != http.StatusBadRequest {
		t.Fatalf("invalid request status=%d", got)
	}
	for _, tc := range []struct {
		err  error
		want int
	}{
		{services.ErrExplorationRateLimited, http.StatusTooManyRequests},
		{services.ErrExplorationInvalidOutput, http.StatusUnprocessableEntity},
		{services.ErrExplorationProviderUnavailable, http.StatusBadGateway},
		{context.DeadlineExceeded, http.StatusGatewayTimeout},
		{errors.New("other"), http.StatusBadGateway},
	} {
		stub.err = tc.err
		if got := callExploration(t, true, "secret", "Bearer secret", stub, validExplorationBody(t)).Code; got != tc.want {
			t.Fatalf("error %v status=%d want=%d", tc.err, got, tc.want)
		}
	}
}

func TestExplorationProxyUsesDedicatedEnvironmentAndPropagatesCancellation(t *testing.T) {
	t.Setenv("AI_BROWSER_ANTHROPIC_API_KEY", "dedicated-test-key")
	t.Setenv("ANTHROPIC_API_KEY", "production-setting-must-not-be-read")
	var receivedKey string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedKey = r.Header.Get("X-AI-Browser-Anthropic-Api-Key")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"schemaVersion":"aurearia.browser-exploration-decision/v1","action":"finish","target":null,"rationale":"done","suspectedFindings":[],"usage":{"inputTokens":1,"outputTokens":1}}`))
	}))
	defer upstream.Close()
	proxy := services.NewAgentProxy(upstream.URL, "agent-token", services.NewLogger(10))
	reqBody := services.ExplorationDecisionRequest{
		SchemaVersion: services.ExplorationDecisionSchemaVersion,
		RunID:         "aibr_20260918T120000Z_012345abcdef", StepID: "step-001",
		Provider: "anthropic", Model: "test-model", Workflow: "edit-one-field", Goal: "finish",
		AllowedActions: []string{"finish"}, AllowedRoutes: []string{"/coin/1"}, Observations: []services.ExplorationObservation{},
	}
	if _, err := proxy.DecideBrowserExploration(context.Background(), reqBody); err != nil {
		t.Fatal(err)
	}
	if receivedKey != "dedicated-test-key" {
		t.Fatalf("dedicated key not forwarded: %q", receivedKey)
	}
	if receivedKey == os.Getenv("ANTHROPIC_API_KEY") {
		t.Fatal("production provider setting was read")
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := proxy.DecideBrowserExploration(ctx, reqBody); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled request error=%v", err)
	}

	blocked := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer blocked.Close()
	proxy = services.NewAgentProxy(blocked.URL, "agent-token", services.NewLogger(10))
	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	if _, err := proxy.DecideBrowserExploration(ctx, reqBody); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("deadline request error=%v", err)
	}
}
