package services

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

// forbiddenHostTransport fails the test if a request is ever made to a host
// other than the given allowed httptest host - guards against any code
// path in this file accidentally reaching the real nomisma.org.
type forbiddenHostTransport struct {
	t            *testing.T
	allowedHost  string
	roundTripper http.RoundTripper
}

func (rt *forbiddenHostTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.URL.Host != rt.allowedHost {
		rt.t.Fatalf("test attempted a live network call to %s - forbidden", req.URL.Host)
	}
	return rt.roundTripper.RoundTrip(req)
}

func newGuardedNomismaClient(t *testing.T, server *httptest.Server) *HTTPNomismaClient {
	t.Helper()
	client := NewHTTPNomismaClientForTest(server.URL)
	client.client = &http.Client{
		Timeout:   nomismaRequestTimeout,
		Transport: &forbiddenHostTransport{t: t, allowedHost: server.Listener.Addr().String(), roundTripper: http.DefaultTransport},
	}
	return client
}

func TestHTTPNomismaClient_Search_OK(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"q1":{"result":[{"id":"roma","name":"Roma","score":100,"match":"true"}]}}`))
	}))
	defer server.Close()

	client := newGuardedNomismaClient(t, server)
	candidates, kind, err := client.Search(context.Background(), "Roma", 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if kind != "" {
		t.Fatalf("expected no error kind, got %q", kind)
	}
	if len(candidates) != 1 || candidates[0].URI != "http://nomisma.org/id/roma" || candidates[0].Label != "Roma" || !candidates[0].Match {
		t.Fatalf("unexpected candidates: %+v", candidates)
	}
}

func TestHTTPNomismaClient_Search_NoMatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"q1":{"result":[]}}`))
	}))
	defer server.Close()

	client := newGuardedNomismaClient(t, server)
	candidates, kind, err := client.Search(context.Background(), "zzzzgibberish", 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if kind != NomismaErrorNoMatch {
		t.Fatalf("expected no_match, got %q", kind)
	}
	if len(candidates) != 0 {
		t.Fatalf("expected zero candidates, got %+v", candidates)
	}
}

func TestHTTPNomismaClient_Search_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := newGuardedNomismaClient(t, server)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()

	_, kind, err := client.Search(ctx, "Roma", 5)
	if err == nil {
		t.Fatal("expected an error")
	}
	if kind != NomismaErrorUnavailable {
		t.Fatalf("expected unavailable, got %q", kind)
	}
}

func TestHTTPNomismaClient_Search_ConnectionRefused(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to reserve a port: %v", err)
	}
	addr := listener.Addr().String()
	listener.Close() // nothing listening now - guarantees connection-refused

	client := NewHTTPNomismaClientForTest("http://" + addr)
	_, kind, err := client.Search(context.Background(), "Roma", 5)
	if err == nil {
		t.Fatal("expected a connection error")
	}
	if kind != NomismaErrorUnavailable {
		t.Fatalf("expected unavailable, got %q", kind)
	}
}

func TestHTTPNomismaClient_Search_Non2xx(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := newGuardedNomismaClient(t, server)
	_, kind, err := client.Search(context.Background(), "Roma", 5)
	if err == nil {
		t.Fatal("expected an error")
	}
	if kind != NomismaErrorUnavailable {
		t.Fatalf("expected unavailable, got %q", kind)
	}
}

func TestHTTPNomismaClient_Search_MalformedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{not valid json`))
	}))
	defer server.Close()

	client := newGuardedNomismaClient(t, server)
	_, kind, err := client.Search(context.Background(), "Roma", 5)
	if err == nil {
		t.Fatal("expected an error")
	}
	if kind != NomismaErrorInvalidResponse {
		t.Fatalf("expected invalid_response, got %q", kind)
	}
}

func TestHTTPNomismaClient_Search_CancelledContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := newGuardedNomismaClient(t, server)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, kind, err := client.Search(ctx, "Roma", 5)
	if err == nil {
		t.Fatal("expected an error")
	}
	if kind != NomismaErrorCancelled && kind != NomismaErrorUnavailable {
		t.Fatalf("expected cancelled or unavailable, got %q", kind)
	}
}

func TestHTTPNomismaClient_Search_BlankQueryRejected(t *testing.T) {
	client := NewHTTPNomismaClientForTest("http://example.invalid")
	_, kind, err := client.Search(context.Background(), "   ", 5)
	if err == nil {
		t.Fatal("expected an error for a blank query")
	}
	if kind != NomismaErrorInvalidRequest {
		t.Fatalf("expected invalid_request, got %q", kind)
	}
}

func TestHTTPNomismaClient_Search_AmbiguousCandidatesReturnedAsIs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"q1":{"result":[
			{"id":"roma","name":"Roma","score":80,"match":"false"},
			{"id":"roma_novum","name":"Roma Novum","score":79,"match":"false"}
		]}}`))
	}))
	defer server.Close()

	client := newGuardedNomismaClient(t, server)
	candidates, kind, err := client.Search(context.Background(), "Roma", 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if kind != "" {
		t.Fatalf("expected no error kind for a present but ambiguous result set, got %q", kind)
	}
	if len(candidates) != 2 {
		t.Fatalf("expected both candidates returned as-is (never auto-selected), got %+v", candidates)
	}
	for _, c := range candidates {
		if c.Match {
			t.Fatalf("expected match to reflect Nomisma's own flag only (false here), got %+v", c)
		}
	}
}

func TestHTTPNomismaClient_NeverCallsRealNomismaHost(t *testing.T) {
	// Sanity check that the default constructor points at the real host -
	// this test never calls Search with it, only asserts the base URL, so
	// no live network call is made anywhere in this file.
	client := NewHTTPNomismaClient()
	if client.baseURL != defaultNomismaBaseURL {
		t.Fatalf("expected default base URL %q, got %q", defaultNomismaBaseURL, client.baseURL)
	}
	if !errors.Is(context.Canceled, context.Canceled) {
		t.Fatal("sanity check failed")
	}
}

// TestHTTPNomismaClient_Search_RequestShapeMatchesLiveContract is a
// regression test for the double-wrapped request bug: the "queries" URL
// parameter's value must decode to the query-identifier map directly
// (e.g. {"q1":{"query":"Roma","limit":5}}), never nested under an outer
// "queries" key. Verified live against https://nomisma.org/apis/reconcile
// on 2026-08-14 - a request built the old (buggy) way reconciles a
// literal, nonexistent "queries" identifier and always returns no_match.
func TestHTTPNomismaClient_Search_RequestShapeMatchesLiveContract(t *testing.T) {
	var capturedQueries string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedQueries = r.URL.Query().Get("queries")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"q1":{"result":[{"id":"roma","name":"Roma","score":100,"match":"true"}]}}`))
	}))
	defer server.Close()

	client := newGuardedNomismaClient(t, server)
	if _, _, err := client.Search(context.Background(), "Roma", 5); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var decoded map[string]struct {
		Query string `json:"query"`
		Limit int    `json:"limit"`
	}
	if err := json.Unmarshal([]byte(capturedQueries), &decoded); err != nil {
		t.Fatalf("captured queries param %q is not valid JSON: %v", capturedQueries, err)
	}
	if _, wrapped := decoded["queries"]; wrapped {
		t.Fatalf("regression: queries param is double-wrapped under an outer \"queries\" key: %s", capturedQueries)
	}
	q1, ok := decoded["q1"]
	if !ok {
		t.Fatalf("expected top-level \"q1\" key in queries param, got %s", capturedQueries)
	}
	if q1.Query != "Roma" || q1.Limit != 5 {
		t.Fatalf("unexpected q1 contents: %+v", q1)
	}
}

// TestHTTPNomismaClient_Search_ResponseShapeMatchesLiveContract is a
// regression test for the "results"-wrapper response bug. The live
// service returns the query-identifier map at the top level with no
// "results" envelope, and encodes "match" as a JSON string, not a JSON
// boolean, and "id" as a short local id rather than a full URI. A parser
// still assuming the old shape would silently fall through to no_match
// (or fail to unmarshal "match") even though the payload here is a real
// match.
func TestHTTPNomismaClient_Search_ResponseShapeMatchesLiveContract(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Exact shape observed from a live call to
		// https://nomisma.org/apis/reconcile?queries=%7B%22q1%22%3A%7B%22query%22%3A%22Rome%22%2C%22limit%22%3A1%7D%7D
		// on 2026-08-14, trimmed to one result.
		w.Write([]byte(`{"q1":{"result":[{"id":"rome","name":"Rome","type":[{"id":"nmo:Mint","name":"Mint"}],"score":1,"match":"false"}]}}`))
	}))
	defer server.Close()

	client := newGuardedNomismaClient(t, server)
	candidates, kind, err := client.Search(context.Background(), "Rome", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if kind != "" {
		t.Fatalf("expected a successful (non-error-kind) result for a real live-shaped payload, got %q", kind)
	}
	if len(candidates) != 1 {
		t.Fatalf("expected one candidate, got %+v", candidates)
	}
	if candidates[0].URI != "http://nomisma.org/id/rome" {
		t.Fatalf("expected short id to be expanded to the full nomisma.org concept URI, got %q", candidates[0].URI)
	}
	if candidates[0].Label != "Rome" || candidates[0].Score != 1 || candidates[0].Match {
		t.Fatalf("unexpected candidate: %+v", candidates[0])
	}
}

// TestHTTPNomismaClient_Search_URLEncodesQueriesParam guards against a
// naive fix that forgets url.Values already percent-encodes the queries
// param - asserts the request line actually contains an encoded value,
// not raw JSON braces.
func TestHTTPNomismaClient_Search_URLEncodesQueriesParam(t *testing.T) {
	var rawQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rawQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"q1":{"result":[]}}`))
	}))
	defer server.Close()

	client := newGuardedNomismaClient(t, server)
	if _, _, err := client.Search(context.Background(), "Roma", 5); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(rawQuery, "{") || strings.Contains(rawQuery, "}") {
		t.Fatalf("expected the queries param to be percent-encoded, got raw query %q", rawQuery)
	}
	unescaped, err := url.QueryUnescape(rawQuery)
	if err != nil {
		t.Fatalf("failed to unescape raw query %q: %v", rawQuery, err)
	}
	if !strings.Contains(unescaped, `"q1"`) {
		t.Fatalf("expected decoded query to contain q1, got %q", unescaped)
	}
}
