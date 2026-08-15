package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// T077: each Python frame type translates to the correct persisted event
// type; cancellation propagates (context cancel closes the outbound HTTP
// request); EOF-without-terminal => agent_unavailable (via a non-nil error
// that deep_identification_service.go's runJob maps to that failure code).

func sseFrame(t *testing.T, frameType string, extra map[string]any) string {
	t.Helper()
	payload := map[string]any{"type": frameType}
	for k, v := range extra {
		payload[k] = v
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal test frame: %v", err)
	}
	return "data: " + string(raw) + "\n\n"
}

func TestStreamDeepIdentificationTranslatesFrameTypes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher := w.(http.Flusher)
		frames := []string{
			sseFrame(t, "router_selected", map[string]any{"selected": []string{"numista"}}),
			sseFrame(t, "provider_started", map[string]any{"provider": "numista"}),
			sseFrame(t, "provider_result", map[string]any{"provider": "numista", "status": "match"}),
			sseFrame(t, "evaluation", map[string]any{"disagreements": []string{}}),
			sseFrame(t, "progress", map[string]any{"phase": "synthesis", "message": "working"}),
			sseFrame(t, "synthesis", map[string]any{"report": map[string]any{"partial_success": false, "proposed_fields": map[string]any{}}}),
		}
		for _, f := range frames {
			_, _ = w.Write([]byte(f))
			flusher.Flush()
		}
	}))
	defer server.Close()

	proxy := NewAgentProxy(server.URL, "test-token", NewLogger(10))

	var seen []string
	err := proxy.StreamDeepIdentification(context.Background(), DeepIdentifyProxyRequest{JobID: 1}, func(frame DeepIdentifyFrame) error {
		seen = append(seen, frame.Type)
		return nil
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	expected := []string{"router_selected", "provider_started", "provider_result", "evaluation", "progress", "synthesis"}
	if len(seen) != len(expected) {
		t.Fatalf("expected %d frames, got %d: %#v", len(expected), len(seen), seen)
	}
	for i, ft := range expected {
		if seen[i] != ft {
			t.Fatalf("frame %d: expected type %q, got %q", i, ft, seen[i])
		}
	}

	// Verify the persisted-event-type mapping used by the pipeline runner
	// agrees with the frame types actually observed (excluding the
	// terminal `synthesis` frame, which is intentionally excluded).
	for _, ft := range []string{"router_selected", "provider_started", "provider_result", "evaluation", "progress", "synthesis_started"} {
		if _, ok := deepPipelineEventType(ft); !ok {
			t.Fatalf("expected frame type %q to map to a persisted event type", ft)
		}
	}
	for _, terminal := range []string{"synthesis", "error"} {
		if _, ok := deepPipelineEventType(terminal); ok {
			t.Fatalf("terminal frame type %q must not be mapped to a persisted event type (SettleTerminal owns it)", terminal)
		}
	}
}

func TestStreamDeepIdentificationCancellationClosesRequest(t *testing.T) {
	started := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher := w.(http.Flusher)
		_, _ = w.Write([]byte(sseFrame(t, "progress", map[string]any{"phase": "start"})))
		flusher.Flush()
		close(started)
		// Block until the client disconnects (context cancelled) or the
		// test's overall timeout fires, whichever first - simulates a
		// long-running Python pipeline.
		<-r.Context().Done()
	}))
	defer server.Close()

	proxy := NewAgentProxy(server.URL, "test-token", NewLogger(10))
	ctx, cancel := context.WithCancel(context.Background())

	errCh := make(chan error, 1)
	go func() {
		errCh <- proxy.StreamDeepIdentification(ctx, DeepIdentifyProxyRequest{JobID: 1}, func(frame DeepIdentifyFrame) error {
			return nil
		})
	}()

	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("server never received the request")
	}
	cancel()

	select {
	case err := <-errCh:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context.Canceled after cancellation, got %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("StreamDeepIdentification did not return promptly after context cancellation")
	}
}

func TestStreamDeepIdentificationEOFWithoutTerminalFrame(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher := w.(http.Flusher)
		_, _ = w.Write([]byte(sseFrame(t, "progress", map[string]any{"phase": "start"})))
		flusher.Flush()
		// Stream ends here without ever emitting synthesis or error.
	}))
	defer server.Close()

	proxy := NewAgentProxy(server.URL, "test-token", NewLogger(10))
	err := proxy.StreamDeepIdentification(context.Background(), DeepIdentifyProxyRequest{JobID: 1}, func(frame DeepIdentifyFrame) error {
		return nil
	})
	if !errors.Is(err, ErrDeepStreamEndedWithoutTerminal) {
		t.Fatalf("expected ErrDeepStreamEndedWithoutTerminal, got %v", err)
	}
}

func TestStreamDeepIdentificationErrorFrameIsTerminal(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher := w.(http.Flusher)
		_, _ = w.Write([]byte(sseFrame(t, "error", map[string]any{"code": "llm_unavailable", "message": "boom"})))
		flusher.Flush()
	}))
	defer server.Close()

	proxy := NewAgentProxy(server.URL, "test-token", NewLogger(10))
	var gotType string
	err := proxy.StreamDeepIdentification(context.Background(), DeepIdentifyProxyRequest{JobID: 1}, func(frame DeepIdentifyFrame) error {
		gotType = frame.Type
		return nil
	})
	if err != nil {
		t.Fatalf("expected no error for a well-formed terminal error frame, got %v", err)
	}
	if gotType != "error" {
		t.Fatalf("expected the error frame to reach onFrame, got %q", gotType)
	}
}

func TestStreamDeepIdentificationMalformedFrameSkippedNotFabricated(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher := w.(http.Flusher)
		_, _ = w.Write([]byte("data: {not-json\n\n"))
		flusher.Flush()
		_, _ = w.Write([]byte(sseFrame(t, "synthesis", map[string]any{"report": map[string]any{"partial_success": false}})))
		flusher.Flush()
	}))
	defer server.Close()

	proxy := NewAgentProxy(server.URL, "test-token", NewLogger(10))
	var seen []string
	err := proxy.StreamDeepIdentification(context.Background(), DeepIdentifyProxyRequest{JobID: 1}, func(frame DeepIdentifyFrame) error {
		seen = append(seen, frame.Type)
		return nil
	})
	if err != nil {
		t.Fatalf("expected no error (malformed frame skipped, terminal frame still reached), got %v", err)
	}
	if len(seen) != 1 || seen[0] != "synthesis" {
		t.Fatalf("expected only the synthesis frame to be delivered, got %#v", seen)
	}
}

func TestDeepPipelineProposalJSONExtractsFieldValues(t *testing.T) {
	report := json.RawMessage(fmt.Sprintf(`{"proposed_fields":{"denomination":{"value":"denarius","confidence":0.8},"ruler":{"value":"Severus","confidence":0.6}}}`))
	out := deepPipelineProposalJSON(report)
	var decoded struct {
		Fields map[string]string `json:"fields"`
	}
	if err := json.Unmarshal([]byte(out), &decoded); err != nil {
		t.Fatalf("expected valid JSON, got error %v (raw: %s)", err, out)
	}
	if decoded.Fields["denomination"] != "denarius" || decoded.Fields["ruler"] != "Severus" {
		t.Fatalf("expected extracted field values, got %#v", decoded.Fields)
	}
}

func TestDeepPipelineProposalJSONEmptyWhenNoProposedFields(t *testing.T) {
	if out := deepPipelineProposalJSON(json.RawMessage(`{"narrative":"no fields here"}`)); out != "" {
		t.Fatalf("expected empty proposal JSON, got %q", out)
	}
}

func TestDeepPipelineProviderCatalogNeverAutomatesOCREOrRPC(t *testing.T) {
	settings := DeepIdentificationSettings{MaxProviders: 4, NumistaCallBudget: 4}
	catalog := deepPipelineProviderCatalog(settings)
	found := map[string]DeepProviderCatalogEntryProxy{}
	for _, entry := range catalog {
		found[entry.Provider] = entry
	}
	for _, provider := range []string{"ocre", "rpc", "ngc"} {
		entry, ok := found[provider]
		if !ok {
			t.Fatalf("expected a catalog entry for %q", provider)
		}
		if entry.Automatable {
			t.Fatalf("provider %q must never be automatable (deferred T155/T156 or terms-prohibited): %#v", provider, entry)
		}
	}
	for _, provider := range []string{"numista", "nomisma"} {
		entry, ok := found[provider]
		if !ok || !entry.Automatable {
			t.Fatalf("expected provider %q to be automatable, got %#v", provider, entry)
		}
	}
}

func TestDeepPipelineBoundsClampToContractLimits(t *testing.T) {
	bounds := deepPipelineBounds(DeepIdentificationSettings{HardTimeout: 300 * time.Second, MaxProviders: 4})
	if bounds.TotalTimeoutS <= 0 || bounds.TotalTimeoutS > 900 {
		t.Fatalf("expected total_timeout_s within (0,900], got %d", bounds.TotalTimeoutS)
	}
	if bounds.MaxConcurrency < 1 || bounds.MaxConcurrency > 10 {
		t.Fatalf("expected max_concurrency within [1,10], got %d", bounds.MaxConcurrency)
	}
	if bounds.ProviderTimeoutS < 1 || bounds.ProviderTimeoutS > 120 {
		t.Fatalf("expected provider_timeout_s within [1,120], got %d", bounds.ProviderTimeoutS)
	}
	if bounds.RecursionLimit < 1 || bounds.RecursionLimit > 50 {
		t.Fatalf("expected recursion_limit within [1,50], got %d", bounds.RecursionLimit)
	}
}
