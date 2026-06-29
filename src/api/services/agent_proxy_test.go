package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAgentProxyFetchLogsSendsInternalCredential(t *testing.T) {
	const token = "test-internal-service-token"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-Internal-Service-Token"); got != token {
			t.Fatalf("expected internal token header %q, got %q", token, got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"logs":[{"timestamp":"2026-06-19T08:00:00Z","level":"info","component":"agent","message":"ok"}]}`))
	}))
	defer server.Close()

	proxy := NewAgentProxy(server.URL, token, NewLogger(10))
	logs := proxy.FetchLogs(context.Background(), 10, "")
	if len(logs) != 1 {
		t.Fatalf("expected 1 log from agent proxy, got %d", len(logs))
	}
}

func TestAgentProxyFetchLogsWithoutCredentialIsRejected(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Internal-Service-Token") == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		_, _ = w.Write([]byte(`{"logs":[]}`))
	}))
	defer server.Close()

	proxy := NewAgentProxy(server.URL, "", NewLogger(10))
	logs := proxy.FetchLogs(context.Background(), 10, "")
	if logs != nil {
		t.Fatalf("expected no logs when internal credential is missing, got %#v", logs)
	}
}

func TestAgentProxyAnalyzeCoinSendsInternalCredential(t *testing.T) {
	const token = "test-internal-service-token"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/analyze" {
			t.Fatalf("expected /api/analyze path, got %s", r.URL.Path)
		}
		if got := r.Header.Get("X-Internal-Service-Token"); got != token {
			t.Fatalf("expected internal token header %q, got %q", token, got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"analysis":"authenticated analysis"}`))
	}))
	defer server.Close()

	proxy := NewAgentProxy(server.URL, token, NewLogger(10))
	analysis, err := proxy.AnalyzeCoin(context.Background(), AnalyzeProxyRequest{
		LLM: LLMConfig{Provider: "anthropic", APIKey: "provider-key", Model: "claude-test"},
		Coin: CoinDataProxy{
			ID:   42,
			Name: "Test Denarius",
		},
		Images: []string{"base64-image"},
		Side:   "obverse",
	})
	if err != nil {
		t.Fatalf("AnalyzeCoin returned error: %v", err)
	}
	if analysis != "authenticated analysis" {
		t.Fatalf("AnalyzeCoin analysis = %q, want authenticated analysis", analysis)
	}
}

func TestAgentProxyAnalyzeCoinInternalCredentialConfigErrorIsNotProviderKeyFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"detail":"Internal service credential is not configured"}`))
	}))
	defer server.Close()

	proxy := NewAgentProxy(server.URL, "go-token", NewLogger(10))
	_, err := proxy.AnalyzeCoin(context.Background(), AnalyzeProxyRequest{
		LLM: LLMConfig{Provider: "anthropic", APIKey: "valid-provider-key", Model: "claude-test"},
		Coin: CoinDataProxy{
			ID:   42,
			Name: "Test Denarius",
		},
		Images: []string{"base64-image"},
		Side:   "obverse",
	})
	if err == nil {
		t.Fatal("expected AnalyzeCoin to return an error for internal service credential config failure")
	}

	errText := strings.ToLower(err.Error())
	if strings.Contains(errText, "anthropic") || strings.Contains(errText, "provider") || strings.Contains(errText, "api key") {
		t.Fatalf("internal credential config failure was misclassified as provider-key failure: %v", err)
	}
	if !strings.Contains(errText, "set agent_internal_service_token on both go api and python agent service") {
		t.Fatalf("expected actionable internal credential configuration error, got %v", err)
	}
}

func TestLLMConfigJSONOmitsProviderIrrelevantFields(t *testing.T) {
	body, err := json.Marshal(LLMConfig{
		Provider:  "anthropic",
		APIKey:    "anthropic-key",
		Model:     "claude-test",
		OllamaURL: "",
	})
	if err != nil {
		t.Fatalf("marshal LLMConfig: %v", err)
	}

	bodyText := string(body)
	if strings.Contains(bodyText, "ollama_url") || strings.Contains(bodyText, "searxng_url") {
		t.Fatalf("Anthropic LLMConfig JSON included provider-irrelevant URLs: %s", bodyText)
	}

	body, err = json.Marshal(LLMConfig{
		Provider:   "ollama",
		Model:      "llava-test",
		OllamaURL:  "http://ollama:11434",
		SearXNGURL: "http://searxng:8080",
	})
	if err != nil {
		t.Fatalf("marshal LLMConfig: %v", err)
	}

	bodyText = string(body)
	if strings.Contains(bodyText, "api_key") {
		t.Fatalf("Ollama LLMConfig JSON included Anthropic API key field: %s", bodyText)
	}
	if !strings.Contains(bodyText, "ollama_url") {
		t.Fatalf("Ollama LLMConfig JSON omitted ollama_url: %s", bodyText)
	}
}

func TestAgentChatProxyRequestJSONIncludesTypedAppContext(t *testing.T) {
	activeCoinID := uint(42)
	req := AgentChatProxyRequest{
		LLM: LLMConfig{
			Provider: "ollama",
			Model:    "test-model",
		},
		User: UserContextProxy{
			UserID:  7,
			ZipCode: "60601",
		},
		Message: "update this coin",
		History: []ChatMessageProxy{
			{Role: "user", Content: "hello"},
		},
		AppContext: &CollectionChatContext{
			Route:        "/coin/42",
			ActiveCoinID: &activeCoinID,
		},
		CoinSearchPrompt: "search prompt",
		CoinShowsPrompt:  "shows prompt",
		InternalToken:    "token",
		ToolsBaseURL:     "http://coins:8080",
	}

	body, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("unmarshal request: %v", err)
	}

	if _, ok := payload["appContext"]; ok {
		t.Fatalf("outbound Python payload used frontend key appContext: %s", body)
	}

	appContext, ok := payload["app_context"].(map[string]any)
	if !ok {
		t.Fatalf("app_context missing or wrong type in payload: %#v", payload["app_context"])
	}
	if got := appContext["route"]; got != "/coin/42" {
		t.Fatalf("app_context.route = %#v, want /coin/42", got)
	}
	if got := appContext["activeCoinId"]; got != float64(42) {
		t.Fatalf("app_context.activeCoinId = %#v, want 42", got)
	}
}

func TestAgentChatProxyRequestJSONOmitsNilAppContext(t *testing.T) {
	req := AgentChatProxyRequest{
		LLM: LLMConfig{
			Provider: "anthropic",
			Model:    "claude-test",
		},
		User: UserContextProxy{
			UserID: 1,
		},
		Message:          "find denarii",
		History:          []ChatMessageProxy{},
		CoinSearchPrompt: "search prompt",
		CoinShowsPrompt:  "shows prompt",
	}

	body, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("unmarshal request: %v", err)
	}
	if _, ok := payload["app_context"]; ok {
		t.Fatalf("app_context should be omitted when nil: %s", body)
	}
}

func TestAgentProxyDiscoverAlertCandidatesSendsTypedPayload(t *testing.T) {
	const token = "test-internal-service-token"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/search/alerts" {
			t.Fatalf("expected /api/search/alerts path, got %s", r.URL.Path)
		}
		if got := r.Header.Get("X-Internal-Service-Token"); got != token {
			t.Fatalf("expected internal token header %q, got %q", token, got)
		}
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		alert := payload["alert"].(map[string]any)
		if alert["max_candidates"] != float64(20) {
			t.Fatalf("max_candidates = %#v", alert["max_candidates"])
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"candidates":[{"source_url":"https://dealer.example/item","title":"Domitian Denarius","reason_for_match":"matches","last_seen_at":"2026-06-29T17:00:00Z","provenance_status":"verified","provenance":[{"field":"source_url","value":"https://dealer.example/item","source_url":"https://dealer.example/item","observed_at":"2026-06-29T17:00:00Z","confidence":"high","verification_state":"verified"}]}],"warnings":[],"partial":false}`))
	}))
	defer server.Close()

	proxy := NewAgentProxy(server.URL, token, NewLogger(10))
	resp, err := proxy.DiscoverAlertCandidates(context.Background(), AlertDiscoveryProxyRequest{
		LLM: LLMConfig{Provider: "ollama", Model: "test"},
		Alert: AlertDiscoveryRequestDetail{
			AlertID:       1,
			MaxCandidates: 20,
			CriteriaSnapshot: AlertDiscoveryCriteriaSnapshotProxy{
				Name: "Domitian",
			},
		},
	})
	if err != nil {
		t.Fatalf("DiscoverAlertCandidates returned error: %v", err)
	}
	if len(resp.Candidates) != 1 || resp.Candidates[0].SourceURL == "" {
		t.Fatalf("unexpected discovery response: %+v", resp)
	}
}

func TestAgentProxyDiscoverAlertCandidatesSanitizesHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"detail":"secret stack trace with provider-key"}`))
	}))
	defer server.Close()
	proxy := NewAgentProxy(server.URL, "token", NewLogger(10))
	_, err := proxy.DiscoverAlertCandidates(context.Background(), AlertDiscoveryProxyRequest{})
	if err == nil {
		t.Fatal("expected error")
	}
	if strings.Contains(err.Error(), "secret") || strings.Contains(err.Error(), "provider-key") {
		t.Fatalf("error leaked response body: %v", err)
	}
}
