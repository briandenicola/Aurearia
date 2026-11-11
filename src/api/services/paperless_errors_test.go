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

func TestHTTPPaperlessClient_GetTaskReadsV10TaskFormat(t *testing.T) {
	for name, body := range map[string]string{
		// paperless-ngx 3.x (API v10): paginated, related_document_ids list.
		"related_document_ids": `{"count":1,"results":[{"task_id":"task-1","status":"SUCCESS","related_document_ids":[42],"result_data":{"document_id":42}}]}`,
		// v10 with only result_data populated.
		"result_data only": `{"count":1,"results":[{"status":"success","related_document_ids":[],"result_data":{"document_id":42}}]}`,
		// v9 shape still works.
		"v9 related_document": `[{"status":"SUCCESS","related_document":"42","result":"ok"}]`,
	} {
		t.Run(name, func(t *testing.T) {
			srv := paperlessV3Server(t, body)
			task, err := (&HTTPPaperlessClient{client: srv.Client()}).GetTask(context.Background(), srv.URL, "tkn", "task-1")
			if err != nil || task.Status != PaperlessTaskSuccess || task.DocumentID == nil || *task.DocumentID != 42 {
				t.Fatalf("task: %+v %v", task, err)
			}
		})
	}
}

func TestHTTPPaperlessClient_GetTaskReportsFailures(t *testing.T) {
	srv := paperlessV3Server(t, `{"count":1,"results":[{"status":"FAILURE","related_document_ids":[],"result":"Not consuming inv.pdf: It is a duplicate."}]}`)
	task, err := (&HTTPPaperlessClient{client: srv.Client()}).GetTask(context.Background(), srv.URL, "tkn", "task-1")
	if err != nil || task.Status != PaperlessTaskFailure || task.Result != "Not consuming inv.pdf: It is a duplicate." {
		t.Fatalf("task: %+v %v", task, err)
	}
}

// Recorded from the owner's paperless-ngx 3.3.3 (API v10), trimmed to two tasks.
const paperlessV10TasksResponse = `{"count":9464,"next":null,"previous":null,"results":[{"id":9463,"task_id":"9713b04a-fb87-451e-bd58-0831eaaad05e","task_type":"consume_file","task_type_display":"Consume File","trigger_source":"api_upload","trigger_source_display":"API Upload","status":"success","status_display":"Success","date_created":"2026-11-11T22:13:12.955151Z","date_started":"2026-11-11T22:13:12.970956Z","date_done":"2026-11-11T22:13:17.816012Z","duration_seconds":4.845056,"wait_time_seconds":0.015805,"input_data":{"filename":"invoice.pdf","mime_type":"application/pdf"},"result_data":{"document_id":4965},"related_document_ids":[4965],"acknowledged":false,"owner":4},{"id":9464,"task_id":"46036ced-a4b4-45ec-82c5-8725a4770872","task_type":"mail_fetch","status":"success","result_data":null,"related_document_ids":[],"acknowledged":false,"owner":null}]}`

func TestHTTPPaperlessClient_GetTaskReadsRealPaperless333Response(t *testing.T) {
	srv := paperlessV3Server(t, paperlessV10TasksResponse)
	task, err := (&HTTPPaperlessClient{client: srv.Client()}).GetTask(context.Background(), srv.URL, "tkn", "9713b04a-fb87-451e-bd58-0831eaaad05e")
	if err != nil || task.Status != PaperlessTaskSuccess || task.DocumentID == nil || *task.DocumentID != 4965 {
		t.Fatalf("task: %+v %v", task, err)
	}
}
