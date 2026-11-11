package services

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"net/url"
	"os"
	"strings"
	"syscall"
	"testing"
	"time"
)

type failingResolver struct{ err error }

func (r failingResolver) LookupNetIP(context.Context, string, string) ([]netip.Addr, error) {
	return nil, r.err
}

func pingStatus(t *testing.T, status int, body string) error {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/documents/" || r.Header.Get("Authorization") != "Token tkn" {
			w.WriteHeader(http.StatusTeapot)
			return
		}
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return (&HTTPPaperlessClient{client: srv.Client()}).Ping(context.Background(), srv.URL, "tkn")
}

func TestHTTPPaperlessClient_PingSucceedsWithValidToken(t *testing.T) {
	if err := pingStatus(t, http.StatusOK, `{"count":0,"results":[]}`); err != nil {
		t.Fatalf("expected ping to succeed, got %v", err)
	}
}

func TestDescribePaperlessError_HTTPStatuses(t *testing.T) {
	for _, tc := range []struct {
		status int
		want   string
	}{
		{http.StatusUnauthorized, "rejected the API token"},
		{http.StatusForbidden, "rejected the API token"},
		{http.StatusNotFound, "404 for the connection test"},
		{http.StatusInternalServerError, "HTTP 500 for the connection test. Response: boom"},
	} {
		err := pingStatus(t, tc.status, "boom")
		if got := DescribePaperlessError(err); !strings.Contains(got, tc.want) {
			t.Errorf("status %d: got %q, want it to contain %q", tc.status, got, tc.want)
		}
	}
}

func TestDescribePaperlessError_UnresolvableDockerHostname(t *testing.T) {
	client := &HTTPPaperlessClient{client: newPaperlessHTTPClient(failingResolver{
		err: &net.DNSError{Err: "no such host", Name: "paperless-ngx", IsNotFound: true},
	})}
	err := client.Ping(context.Background(), "http://paperless-ngx:8000", "tkn")
	got := DescribePaperlessError(err)
	if !strings.Contains(got, `could not resolve "paperless-ngx"`) || !strings.Contains(got, "same Docker network") {
		t.Fatalf("got %q", got)
	}
}

func TestDescribePaperlessError_HTTPSAgainstPlainHTTP(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()
	client := &HTTPPaperlessClient{client: srv.Client()}
	err := client.Ping(context.Background(), strings.Replace(srv.URL, "http://", "https://", 1), "tkn")
	if got := DescribePaperlessError(err); !strings.Contains(got, "answered with plain HTTP") {
		t.Fatalf("got %q from %v", got, err)
	}
}

func TestDescribePaperlessError_Timeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
	}))
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	err := (&HTTPPaperlessClient{client: srv.Client()}).Ping(ctx, srv.URL, "tkn")
	if got := DescribePaperlessError(err); !strings.Contains(got, "Timed out") {
		t.Fatalf("got %q from %v", got, err)
	}
}

func TestDescribePaperlessError_ConnectionRefusedAndUnreachable(t *testing.T) {
	wrap := func(errno syscall.Errno) error {
		return &url.Error{Op: "Get", URL: "http://10.0.0.5:8000/api/documents/", Err: &net.OpError{
			Op: "dial", Net: "tcp", Err: os.NewSyscallError("connect", errno),
		}}
	}
	if got := DescribePaperlessError(wrap(syscall.ECONNREFUSED)); !strings.Contains(got, "Connection refused") {
		t.Errorf("refused: got %q", got)
	}
	if got := DescribePaperlessError(wrap(syscall.EHOSTUNREACH)); !strings.Contains(got, "unreachable") {
		t.Errorf("unreachable: got %q", got)
	}
}

func TestDescribePaperlessError_BlockedAndNotConfigured(t *testing.T) {
	if got := DescribePaperlessError(ErrOutboundTargetBlocked); !strings.Contains(got, "blocked address") {
		t.Errorf("blocked: got %q", got)
	}
	if got := DescribePaperlessError(ErrPaperlessNotConfigured); got != ErrPaperlessNotConfigured.Error() {
		t.Errorf("not configured: got %q", got)
	}
	if got := DescribePaperlessError(errors.New("something odd")); got != "Could not reach paperless-ngx: something odd" {
		t.Errorf("fallback: got %q", got)
	}
}

func TestDescribePaperlessError_NeverIncludesToken(t *testing.T) {
	err := pingStatus(t, http.StatusInternalServerError, "server error")
	if strings.Contains(DescribePaperlessError(err), "tkn") {
		t.Fatal("description leaked the API token")
	}
}

// paperlessV3Server mimics paperless-ngx 3.x: it accepts only API versions 9
// and 10 in the Accept header and answers 406 to anything else.
func paperlessV3Server(t *testing.T, tasksBody string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		accept := r.Header.Get("Accept")
		if strings.Contains(accept, "version=") && !strings.Contains(accept, "version=9") && !strings.Contains(accept, "version=10") {
			w.WriteHeader(http.StatusNotAcceptable)
			_, _ = w.Write([]byte(`{"detail":"Invalid version in \"Accept\" header."}`))
			return
		}
		switch r.URL.Path {
		case "/api/documents/post_document/":
			_, _ = w.Write([]byte(`"task-1"`))
		case "/api/tasks/":
			_, _ = w.Write([]byte(tasksBody))
		default:
			_, _ = w.Write([]byte(`{"count":0,"results":[]}`))
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestHTTPPaperlessClient_WorksWithPaperlessV3VersionRules(t *testing.T) {
	srv := paperlessV3Server(t, `[{"status":"SUCCESS","related_document":"42","result":"ok"}]`)
	client := &HTTPPaperlessClient{client: srv.Client()}
	ctx := context.Background()
	if err := client.Ping(ctx, srv.URL, "tkn"); err != nil {
		t.Fatalf("ping: %v", err)
	}
	taskID, err := client.PostDocument(ctx, srv.URL, "tkn", "Coin – Invoice", "inv.pdf", "application/pdf", strings.NewReader("PDF"))
	if err != nil || taskID != "task-1" {
		t.Fatalf("upload: %q %v", taskID, err)
	}
	task, err := client.GetTask(ctx, srv.URL, "tkn", taskID)
	if err != nil || task.DocumentID == nil || *task.DocumentID != 42 {
		t.Fatalf("task: %+v %v", task, err)
	}
}

func TestHTTPPaperlessClient_GetTaskAcceptsPaginatedTasks(t *testing.T) {
	srv := paperlessV3Server(t, `{"count":1,"results":[{"status":"SUCCESS","related_document":7}]}`)
	task, err := (&HTTPPaperlessClient{client: srv.Client()}).GetTask(context.Background(), srv.URL, "tkn", "task-1")
	if err != nil || task.Status != PaperlessTaskSuccess || task.DocumentID == nil || *task.DocumentID != 7 {
		t.Fatalf("task: %+v %v", task, err)
	}
}

func TestDescribePaperlessError_NotAcceptable(t *testing.T) {
	err := pingStatus(t, http.StatusNotAcceptable, `{"detail":"Invalid version"}`)
	if got := DescribePaperlessError(err); !strings.Contains(got, "HTTP 406") {
		t.Fatalf("got %q", got)
	}
}
