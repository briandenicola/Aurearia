package services

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestCoinCopilotProxyDisconnectAndCancellation(t *testing.T) {
	t.Run("disconnect without terminal", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = w.Write([]byte("data: {\"schema_version\":1,\"run_id\":\"ccr\",\"execution_id\":\"cce\",\"frame_id\":\"one\",\"type\":\"usage\",\"payload\":{}}\n\n"))
		}))
		defer server.Close()
		proxy := NewAgentProxy(server.URL, "internal", NewLogger(10))
		err := proxy.StreamCoinCopilot(context.Background(), CopilotExecuteProxyRequest{}, func(CopilotAgentFrame) error { return nil })
		if !errors.Is(err, ErrCopilotStreamEndedWithoutTerminal) {
			t.Fatalf("disconnect error = %v", err)
		}
	})
	t.Run("caller cancellation", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(http.StatusOK)
			w.(http.Flusher).Flush()
			<-r.Context().Done()
		}))
		defer server.Close()
		proxy := NewAgentProxy(server.URL, "internal", NewLogger(10))
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
		defer cancel()
		err := proxy.StreamCoinCopilot(ctx, CopilotExecuteProxyRequest{}, nil)
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("cancellation error = %v", err)
		}
	})
}

func TestCoinCopilotProxyErrorSanitization(t *testing.T) {
	sanitized := sanitizeAgentErrorBodyForLog([]byte(`{"execution_token":"secret-value","detail":[{"loc":["body","llm","api_key"],"input":"anthropic-secret"}]}`), 400)
	if strings.Contains(sanitized, "secret-value") || strings.Contains(sanitized, "anthropic-secret") {
		t.Fatalf("secret leaked in sanitized error: %s", sanitized)
	}
	if !strings.Contains(sanitized, "[REDACTED]") {
		t.Fatalf("redaction marker missing: %s", sanitized)
	}
}

func TestCoinCopilotProxyOllamaCapabilityFailsClosed(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		response string
		want     bool
	}{
		{name: "anthropic binding succeeds", provider: "anthropic", response: `{"supported":true}`, want: true},
		{name: "anthropic binding fails", provider: "anthropic", response: `{"supported":false}`, want: false},
		{name: "ollama tools supported", provider: "ollama", response: `{"supported":true}`, want: true},
		{name: "ollama tools unsupported", provider: "ollama", response: `{"supported":false}`, want: false},
		{name: "malformed response", provider: "anthropic", response: `{"supported":"yes"}`, want: false},
		{name: "unknown response field", provider: "anthropic", response: `{"supported":true,"detail":"unexpected"}`, want: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/api/copilot/capability" {
					http.NotFound(w, r)
					return
				}
				if r.Header.Get("X-Internal-Service-Token") != "internal" {
					t.Errorf("internal credential was not attached")
				}
				var request map[string]json.RawMessage
				if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
					t.Errorf("decode request: %v", err)
				}
				if len(request) != 1 || request["llm"] == nil {
					t.Errorf("preflight request included fields beyond llm configuration: %#v", request)
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(test.response))
			}))
			defer server.Close()
			proxy := NewAgentProxy(server.URL, "internal", NewLogger(10))
			llm := LLMConfig{Provider: test.provider, Model: "test"}
			if test.provider == "anthropic" {
				llm.APIKey = "secret"
			} else {
				llm.OllamaURL = "http://ollama:11434"
			}
			if got := proxy.SupportsCoinCopilot(llm); got != test.want {
				t.Fatalf("SupportsCoinCopilot() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestCoinCopilotProxyCapabilityTimeoutFailsClosed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-time.After(200 * time.Millisecond):
		}
	}))
	defer server.Close()
	proxy := NewAgentProxy(server.URL, "internal", NewLogger(10))
	proxy.requestClient = &http.Client{Timeout: 20 * time.Millisecond}
	if proxy.SupportsCoinCopilot(LLMConfig{Provider: "anthropic", APIKey: "secret", Model: "test"}) {
		t.Fatal("timed-out capability preflight was accepted")
	}
}
