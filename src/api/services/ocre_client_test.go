package services

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// Feature 345 T007/T033 — httptest-backed OCRE client tests. A guard
// transport fails the test if any request ever reaches a host other than the
// httptest server, so no live nomisma.org call can happen in CI.

type forbiddenOCREHostTransport struct {
	t           *testing.T
	allowedHost string
	inner       http.RoundTripper
}

func (rt *forbiddenOCREHostTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.URL.Host != rt.allowedHost {
		rt.t.Fatalf("test attempted a live network call to %s — forbidden", req.URL.Host)
	}
	return rt.inner.RoundTrip(req)
}

func newGuardedOCREClient(t *testing.T, server *httptest.Server) *HTTPOCREClient {
	t.Helper()
	client := NewHTTPOCREClientForTest(server.URL)
	client.client = &http.Client{
		Timeout:   ocreRequestTimeout,
		Transport: &forbiddenOCREHostTransport{t: t, allowedHost: server.Listener.Addr().String(), inner: http.DefaultTransport},
	}
	return client
}

const ocreOKBody = `{"head":{"vars":["type","label"]},"results":{"bindings":[
	{"type":{"type":"uri","value":"http://numismatics.org/ocre/id/ric.2.hdn.39b"},"label":{"type":"literal","xml:lang":"en","value":"RIC II Hadrian 39b"}}
]}}`

func hadrianParams() OCREQueryParams {
	return NewOCREQueryParams("hadrian", "denarius", "rome", "", nil, "", 5)
}

func TestHTTPOCREClient_Search_OK(t *testing.T) {
	var gotAccept, gotUA, gotMethod string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAccept = r.Header.Get("Accept")
		gotUA = r.Header.Get("User-Agent")
		gotMethod = r.Method
		w.Header().Set("Content-Type", ocreAcceptSPARQLJSON)
		_, _ = w.Write([]byte(ocreOKBody))
	}))
	defer server.Close()

	client := newGuardedOCREClient(t, server)
	candidates, kind, err := client.Search(context.Background(), hadrianParams(), 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if kind != "" {
		t.Fatalf("expected success kind, got %q", kind)
	}
	if len(candidates) != 1 || candidates[0].TypeURI != "https://numismatics.org/ocre/id/ric.2.hdn.39b" {
		t.Fatalf("unexpected candidates: %+v", candidates)
	}
	if gotMethod != http.MethodGet {
		t.Fatalf("expected GET, got %s", gotMethod)
	}
	if gotAccept != ocreAcceptSPARQLJSON {
		t.Fatalf("expected SPARQL-results Accept header, got %q", gotAccept)
	}
	if gotUA == "" || strings.Contains(strings.ToLower(gotUA), "go-http-client") {
		t.Fatalf("expected a fixed non-default User-Agent, got %q", gotUA)
	}
}

func TestHTTPOCREClient_Search_EmptyBindings(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"head":{"vars":["type","label"]},"results":{"bindings":[]}}`))
	}))
	defer server.Close()

	client := newGuardedOCREClient(t, server)
	candidates, kind, err := client.Search(context.Background(), hadrianParams(), 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if kind != OCREErrorNoMatch {
		t.Fatalf("expected no_match, got %q", kind)
	}
	if len(candidates) != 0 {
		t.Fatalf("expected zero candidates, got %+v", candidates)
	}
}

func TestHTTPOCREClient_Search_HTTP500(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := newGuardedOCREClient(t, server)
	_, kind, err := client.Search(context.Background(), hadrianParams(), 5)
	// The caller must only ever see the typed kind — never a raw transport
	// error type it has to interpret (T033).
	if kind != OCREErrorUnavailable {
		t.Fatalf("expected unavailable, got %q (err=%v)", kind, err)
	}
}

func TestHTTPOCREClient_Search_MalformedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{not valid json`))
	}))
	defer server.Close()

	client := newGuardedOCREClient(t, server)
	_, kind, err := client.Search(context.Background(), hadrianParams(), 5)
	if kind != OCREErrorInvalidResponse {
		t.Fatalf("expected invalid_response, got %q (err=%v)", kind, err)
	}
}

func TestHTTPOCREClient_Search_OversizeBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", ocreAcceptSPARQLJSON)
		big := strings.Repeat("a", (1<<20)+1024)
		_, _ = w.Write([]byte(big))
	}))
	defer server.Close()

	client := newGuardedOCREClient(t, server)
	_, kind, _ := client.Search(context.Background(), hadrianParams(), 5)
	if kind != OCREErrorInvalidResponse {
		t.Fatalf("expected invalid_response for an oversize body, got %q", kind)
	}
}

func TestHTTPOCREClient_Search_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := newGuardedOCREClient(t, server)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()
	_, kind, err := client.Search(ctx, hadrianParams(), 5)
	if err == nil {
		t.Fatal("expected an error")
	}
	// F2: a caller context deadline is a timeout, not a generic unavailable.
	if kind != OCREErrorTimeout {
		t.Fatalf("expected timeout on a context deadline, got %q", kind)
	}
}

func TestHTTPOCREClient_Search_ClientTimeout(t *testing.T) {
	// F2: the client's own http.Client.Timeout (no caller deadline) must also
	// map to the typed timeout kind, so a slow upstream flows as timed_out.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(60 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := &HTTPOCREClient{baseURL: server.URL, client: &http.Client{Timeout: 5 * time.Millisecond}}
	_, kind, err := client.Search(context.Background(), hadrianParams(), 5)
	if err == nil {
		t.Fatal("expected an error")
	}
	if kind != OCREErrorTimeout {
		t.Fatalf("expected timeout on http.Client.Timeout, got %q", kind)
	}
}

func TestHTTPOCREClient_Search_CancelledContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := newGuardedOCREClient(t, server)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, kind, err := client.Search(ctx, hadrianParams(), 5)
	if err == nil {
		t.Fatal("expected an error")
	}
	if kind != OCREErrorCancelled && kind != OCREErrorUnavailable {
		t.Fatalf("expected cancelled/unavailable, got %q", kind)
	}
}

func TestHTTPOCREClient_Search_OffHostBindingRejected(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", ocreAcceptSPARQLJSON)
		_, _ = w.Write([]byte(`{"results":{"bindings":[
			{"type":{"type":"uri","value":"http://evil.example.com/ocre/id/ric.2.hdn.39b"},"label":{"type":"literal","value":"Spoofed"}}
		]}}`))
	}))
	defer server.Close()

	client := newGuardedOCREClient(t, server)
	candidates, kind, err := client.Search(context.Background(), hadrianParams(), 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// The off-host binding is dropped, leaving zero candidates → no_match.
	if kind != OCREErrorNoMatch || len(candidates) != 0 {
		t.Fatalf("expected off-host binding dropped to no_match, got kind=%q candidates=%+v", kind, candidates)
	}
}

func TestHTTPOCREClient_Search_NoSignalRejected(t *testing.T) {
	client := NewHTTPOCREClientForTest("http://example.invalid")
	_, kind, err := client.Search(context.Background(), NewOCREQueryParams("", "", "", "silver", nil, "", 5), 5)
	if err == nil {
		t.Fatal("expected an error for a signal-less query")
	}
	if kind != OCREErrorInvalidRequest {
		t.Fatalf("expected invalid_request, got %q", kind)
	}
}

func TestHTTPOCREClient_Search_ConnectionRefused(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to reserve a port: %v", err)
	}
	addr := listener.Addr().String()
	listener.Close()

	client := NewHTTPOCREClientForTest("http://" + addr)
	_, kind, err := client.Search(context.Background(), hadrianParams(), 5)
	if err == nil {
		t.Fatal("expected a connection error")
	}
	if kind != OCREErrorUnavailable {
		t.Fatalf("expected unavailable, got %q", kind)
	}
}

// T033: consolidate the failure-isolation contract — for every transport or
// parse failure the caller sees ONLY a typed OCREErrorKind (always set, drawn
// from the known set) and an empty candidate slice. A raw net/JSON error type
// is never the classification the caller must interpret; the returned err is
// diagnostic-only and never leaks partial/unsanitized candidate data.
func TestHTTPOCREClient_Search_RawErrorsNeverSurfacedOnlyTypedKind(t *testing.T) {
	typedKinds := map[OCREErrorKind]bool{
		OCREErrorUnavailable:     true,
		OCREErrorTimeout:         true,
		OCREErrorInvalidResponse: true,
		OCREErrorCancelled:       true,
	}

	t.Run("http_500", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()
		candidates, kind, _ := newGuardedOCREClient(t, server).Search(context.Background(), hadrianParams(), 5)
		if !typedKinds[kind] {
			t.Fatalf("expected a typed kind, got %q", kind)
		}
		if len(candidates) != 0 {
			t.Fatalf("no candidate data may leak on failure, got %d", len(candidates))
		}
	})

	t.Run("malformed_json", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", ocreAcceptSPARQLJSON)
			_, _ = w.Write([]byte(`{not valid json`))
		}))
		defer server.Close()
		candidates, kind, _ := newGuardedOCREClient(t, server).Search(context.Background(), hadrianParams(), 5)
		if kind != OCREErrorInvalidResponse {
			t.Fatalf("expected invalid_response, got %q", kind)
		}
		if len(candidates) != 0 {
			t.Fatalf("no candidate data may leak on a parse failure, got %d", len(candidates))
		}
	})

	t.Run("timeout", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(50 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
		defer cancel()
		candidates, kind, _ := newGuardedOCREClient(t, server).Search(ctx, hadrianParams(), 5)
		if !typedKinds[kind] {
			t.Fatalf("expected a typed kind on timeout, got %q", kind)
		}
		if len(candidates) != 0 {
			t.Fatalf("no candidate data may leak on timeout, got %d", len(candidates))
		}
	})
}

func TestHTTPOCREClient_DefaultBaseURLIsRealHost(t *testing.T) {
	client := NewHTTPOCREClient()
	if client.baseURL != defaultOCREBaseURL {
		t.Fatalf("expected default base URL %q, got %q", defaultOCREBaseURL, client.baseURL)
	}
	if !strings.HasPrefix(client.baseURL, "https://nomisma.org/query") {
		t.Fatalf("expected the fixed Nomisma query endpoint, got %q", client.baseURL)
	}
}
