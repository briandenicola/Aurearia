package services

import (
	"context"
	"net/http"
	"net/http/httptest"
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
