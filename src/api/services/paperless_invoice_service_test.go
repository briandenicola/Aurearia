package services

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

var paperlessServiceDBCounter atomic.Uint64

type fakePaperlessClient struct {
	mu         sync.Mutex
	files      map[PaperlessFileKind]*PaperlessFile
	fileDocID  int
	postErr    error
	postTitle  string
	tasks      []*PaperlessTask
	taskCalls  int
	postedBody string
}

func (f *fakePaperlessClient) PostDocument(_ context.Context, _, token, title, _, _ string, body io.Reader) (string, error) {
	if token != "secret-token" {
		return "", fmt.Errorf("unexpected token %q", token)
	}
	f.postTitle = title
	b, _ := io.ReadAll(body)
	f.postedBody = string(b)
	if f.postErr != nil {
		return "", f.postErr
	}
	return "task-123", nil
}

func (f *fakePaperlessClient) GetDocumentFile(_ context.Context, _, token string, documentID int, kind PaperlessFileKind) (*PaperlessFile, error) {
	if token != "secret-token" {
		return nil, fmt.Errorf("unexpected token %q", token)
	}
	f.fileDocID = documentID
	file, ok := f.files[kind]
	if !ok {
		return nil, &PaperlessError{Operation: string(kind), StatusCode: http.StatusNotFound}
	}
	return file, nil
}

func (f *fakePaperlessClient) setTasks(tasks ...*PaperlessTask) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.tasks = tasks
	f.taskCalls = 0
}

func (f *fakePaperlessClient) Ping(context.Context, string, string) error { return nil }

func (f *fakePaperlessClient) GetTask(_ context.Context, _, _, taskID string) (*PaperlessTask, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if taskID != "task-123" {
		return nil, fmt.Errorf("unexpected task %q", taskID)
	}
	idx := f.taskCalls
	if idx >= len(f.tasks) {
		idx = len(f.tasks) - 1
	}
	f.taskCalls++
	return f.tasks[idx], nil
}

func setupPaperlessInvoiceService(t *testing.T, client PaperlessClient, configured bool) (*PaperlessInvoiceService, *gorm.DB, models.User, models.Coin) {
	t.Helper()
	dsn := fmt.Sprintf("file:paperless_service_%d_%d?mode=memory&cache=shared", time.Now().UnixNano(), paperlessServiceDBCounter.Add(1))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)
	if err := db.AutoMigrate(&models.User{}, &models.StorageLocation{}, &models.MintLocation{}, &models.Tag{}, &models.CoinSet{}, &models.Coin{}, &models.CoinImage{}, &models.CoinReference{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	user := models.User{Username: "collector", PasswordHash: "hash"}
	if configured {
		user.PaperlessURL = "https://paperless.example.com"
		user.PaperlessAPIToken = "secret-token"
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	coin := models.Coin{Name: "Gordian III", UserID: user.ID, VendorInvoice: "INV-1"}
	if err := db.Create(&coin).Error; err != nil {
		t.Fatalf("create coin: %v", err)
	}
	svc := NewPaperlessInvoiceService(repository.NewCoinRepository(db), repository.NewUserRepository(db), nil, client).
		WithPolling(time.Millisecond, 50*time.Millisecond).
		WithBackgroundPolling(time.Hour, time.Millisecond)
	return svc, db, user, coin
}

func storedDocumentID(t *testing.T, db *gorm.DB, coinID uint) *int {
	t.Helper()
	var coin models.Coin
	if err := db.First(&coin, coinID).Error; err != nil {
		t.Fatalf("reload coin: %v", err)
	}
	return coin.PaperlessDocumentID
}

func TestPaperlessInvoiceService_UploadLinksDocumentOnSuccess(t *testing.T) {
	docID := 42
	client := &fakePaperlessClient{tasks: []*PaperlessTask{{Status: PaperlessTaskStarted}, {Status: PaperlessTaskSuccess, DocumentID: &docID}}}
	svc, db, user, coin := setupPaperlessInvoiceService(t, client, true)

	result, err := svc.UploadInvoice(context.Background(), user.ID, coin.ID, "inv.pdf", "application/pdf", strings.NewReader("%PDF"))
	if err != nil {
		t.Fatalf("upload: %v", err)
	}
	if result.Status != "linked" || result.DocumentID == nil || *result.DocumentID != 42 {
		t.Fatalf("unexpected result: %+v", result)
	}
	if got := storedDocumentID(t, db, coin.ID); got == nil || *got != 42 {
		t.Fatalf("coin not linked, got %v", got)
	}
	if client.postTitle != "Gordian III – Invoice INV-1" || client.postedBody != "%PDF" {
		t.Fatalf("unexpected upload title=%q body=%q", client.postTitle, client.postedBody)
	}
}

func TestPaperlessInvoiceService_UploadReturnsPendingThenResolves(t *testing.T) {
	client := &fakePaperlessClient{tasks: []*PaperlessTask{{Status: PaperlessTaskPending}}}
	svc, db, user, coin := setupPaperlessInvoiceService(t, client, true)

	result, err := svc.UploadInvoice(context.Background(), user.ID, coin.ID, "inv.pdf", "application/pdf", strings.NewReader("x"))
	if err != nil {
		t.Fatalf("upload: %v", err)
	}
	if result.Status != "pending" || result.TaskID != "task-123" {
		t.Fatalf("expected pending, got %+v", result)
	}
	if storedDocumentID(t, db, coin.ID) != nil {
		t.Fatal("pending upload must not link a document")
	}

	docID := 7
	client.setTasks(&PaperlessTask{Status: PaperlessTaskSuccess, DocumentID: &docID})
	resolved, err := svc.ResolveTask(context.Background(), user.ID, coin.ID, "task-123")
	if err != nil || resolved.Status != "linked" {
		t.Fatalf("resolve: %+v %v", resolved, err)
	}
	if got := storedDocumentID(t, db, coin.ID); got == nil || *got != 7 {
		t.Fatalf("coin not linked after resolve, got %v", got)
	}
}

func TestPaperlessInvoiceService_FailureAndConfigErrors(t *testing.T) {
	client := &fakePaperlessClient{tasks: []*PaperlessTask{{Status: PaperlessTaskFailure, Result: "duplicate of #3"}}}
	svc, _, user, coin := setupPaperlessInvoiceService(t, client, true)
	if _, err := svc.UploadInvoice(context.Background(), user.ID, coin.ID, "inv.pdf", "", strings.NewReader("x")); !errors.Is(err, ErrPaperlessTaskFailed) || !strings.Contains(err.Error(), "duplicate of #3") {
		t.Fatalf("expected task failure, got %v", err)
	}

	unconfigured, _, user2, coin2 := setupPaperlessInvoiceService(t, &fakePaperlessClient{}, false)
	if _, err := unconfigured.UploadInvoice(context.Background(), user2.ID, coin2.ID, "inv.pdf", "", strings.NewReader("x")); !errors.Is(err, ErrPaperlessNotConfigured) {
		t.Fatalf("expected not configured, got %v", err)
	}
	if _, err := unconfigured.UploadInvoice(context.Background(), user2.ID+99, coin2.ID, "inv.pdf", "", strings.NewReader("x")); !errors.Is(err, ErrPaperlessCoinNotFound) {
		t.Fatalf("expected other user's coin to be not found, got %v", err)
	}
}

func TestHTTPPaperlessClient_PostDocumentAndGetTask(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Token tkn" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/paperless/api/documents/post_document/":
			_, params, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
			reader := multipart.NewReader(r.Body, params["boundary"])
			fields := map[string]string{}
			for {
				part, err := reader.NextPart()
				if err != nil {
					break
				}
				b, _ := io.ReadAll(part)
				fields[part.FormName()] = string(b)
			}
			if fields["title"] != "Coin – Invoice" || fields["document"] != "PDFDATA" {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			_, _ = w.Write([]byte(`"abc-123"`))
		case r.Method == http.MethodGet && r.URL.Path == "/paperless/api/documents/55/preview/":
			w.Header().Set("Content-Type", "application/pdf")
			_, _ = w.Write([]byte("%PDF-1.7"))
		case r.Method == http.MethodGet && r.URL.Path == "/paperless/api/tasks/" && r.URL.Query().Get("task_id") == "abc-123":
			_, _ = w.Write([]byte(`[{"status":"SUCCESS","related_document":"55","result":"ok"}]`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	client := &HTTPPaperlessClient{client: srv.Client()}
	base := srv.URL + "/paperless/"
	taskID, err := client.PostDocument(context.Background(), base, "tkn", "Coin – Invoice", "inv.pdf", "application/pdf", strings.NewReader("PDFDATA"))
	if err != nil || taskID != "abc-123" {
		t.Fatalf("post: %q %v", taskID, err)
	}
	task, err := client.GetTask(context.Background(), base, "tkn", taskID)
	if err != nil || task.Status != PaperlessTaskSuccess || task.DocumentID == nil || *task.DocumentID != 55 {
		t.Fatalf("task: %+v %v", task, err)
	}
	file, err := client.GetDocumentFile(context.Background(), base, "tkn", 55, PaperlessFilePreview)
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	previewBody, _ := io.ReadAll(file.Body)
	file.Body.Close()
	if file.ContentType != "application/pdf" || string(previewBody) != "%PDF-1.7" {
		t.Fatalf("preview: %q %q", file.ContentType, previewBody)
	}
	if _, err := client.GetDocumentFile(context.Background(), base, "tkn", 0, PaperlessFilePreview); err == nil {
		t.Fatal("document id 0 must be rejected")
	}
	_, err = client.PostDocument(context.Background(), base, "wrong", "", "inv.pdf", "", strings.NewReader("x"))
	var pErr *PaperlessError
	if !errors.As(err, &pErr) || pErr.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 PaperlessError, got %v", err)
	}
}

func TestParsePaperlessDocumentID(t *testing.T) {
	cases := map[string]*int{`12`: intPtr(12), `"34"`: intPtr(34), `null`: nil, ``: nil, `"x"`: nil, `0`: nil}
	for raw, want := range cases {
		got := parsePaperlessDocumentID([]byte(raw))
		if (got == nil) != (want == nil) || (got != nil && *got != *want) {
			t.Fatalf("parse %q = %v, want %v", raw, got, want)
		}
	}
}

func intPtr(v int) *int { return &v }

func TestIsDisallowedPaperlessIP(t *testing.T) {
	allowed := []string{"192.168.1.20", "10.0.0.5", "172.18.0.4", "100.101.102.103", "203.0.113.9", "fd00::10", "2001:db8::1"}
	blocked := []string{"127.0.0.1", "::1", "169.254.169.254", "fe80::1", "0.0.0.0", "224.0.0.1", "255.255.255.255", "198.18.0.1", "::"}
	for _, raw := range allowed {
		if IsDisallowedPaperlessIP(net.ParseIP(raw)) {
			t.Errorf("%s should be allowed for paperless-ngx", raw)
		}
	}
	for _, raw := range blocked {
		if !IsDisallowedPaperlessIP(net.ParseIP(raw)) {
			t.Errorf("%s should be blocked for paperless-ngx", raw)
		}
	}
}

type staticPaperlessResolver map[string][]netip.Addr

func (r staticPaperlessResolver) LookupNetIP(_ context.Context, _, host string) ([]netip.Addr, error) {
	if addrs, ok := r[host]; ok {
		return addrs, nil
	}
	return nil, fmt.Errorf("no such host %s", host)
}

func TestPaperlessDialContext_BlocksMetadataAndLoopbackHostnames(t *testing.T) {
	dial := paperlessDialContext(staticPaperlessResolver{
		"metadata.internal": {netip.MustParseAddr("169.254.169.254")},
		"sneaky.local":      {netip.MustParseAddr("127.0.0.1")},
	})
	for _, host := range []string{"metadata.internal:80", "sneaky.local:8000", "127.0.0.1:8000", "[::1]:8000"} {
		if _, err := dial(context.Background(), "tcp", host); !errors.Is(err, ErrOutboundTargetBlocked) {
			t.Errorf("dial %s: expected blocked, got %v", host, err)
		}
	}
}

func storedTaskID(t *testing.T, db *gorm.DB, coinID uint) string {
	t.Helper()
	var coin models.Coin
	if err := db.First(&coin, coinID).Error; err != nil {
		t.Fatalf("reload coin: %v", err)
	}
	return coin.PaperlessTaskID
}

func TestPaperlessInvoiceService_BackgroundLinksAfterRequestReturns(t *testing.T) {
	client := &fakePaperlessClient{tasks: []*PaperlessTask{{Status: PaperlessTaskStarted}}}
	svc, db, user, coin := setupPaperlessInvoiceService(t, client, true)
	svc.WithBackgroundPolling(5*time.Millisecond, 5*time.Second)

	result, err := svc.UploadInvoice(context.Background(), user.ID, coin.ID, "inv.pdf", "application/pdf", strings.NewReader("x"))
	if err != nil || result.Status != "pending" {
		t.Fatalf("expected pending upload, got %+v %v", result, err)
	}
	if got := storedTaskID(t, db, coin.ID); got != "task-123" {
		t.Fatalf("pending task id not stored on coin, got %q", got)
	}

	docID := 91
	client.setTasks(&PaperlessTask{Status: PaperlessTaskSuccess, DocumentID: &docID})
	deadline := time.Now().Add(3 * time.Second)
	for storedDocumentID(t, db, coin.ID) == nil && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if got := storedDocumentID(t, db, coin.ID); got == nil || *got != 91 {
		t.Fatalf("background follow-up did not link the document, got %v", got)
	}
	if got := storedTaskID(t, db, coin.ID); got != "" {
		t.Fatalf("task id should be cleared once linked, got %q", got)
	}
}

func TestPaperlessInvoiceService_ResolveIgnoresStaleTask(t *testing.T) {
	client := &fakePaperlessClient{tasks: []*PaperlessTask{{Status: PaperlessTaskStarted}}}
	svc, _, user, coin := setupPaperlessInvoiceService(t, client, true)
	if _, err := svc.ResolveTask(context.Background(), user.ID, coin.ID, "task-123"); !errors.Is(err, ErrPaperlessTaskFailed) {
		t.Fatalf("a task the coin is not waiting on must not be resolved, got %v", err)
	}
}

func TestPaperlessInvoiceService_OpenDocumentOnlyServesLinkedDocument(t *testing.T) {
	client := &fakePaperlessClient{files: map[PaperlessFileKind]*PaperlessFile{
		PaperlessFilePreview:   {Body: io.NopCloser(strings.NewReader("%PDF")), ContentType: "application/pdf", ContentLength: 4},
		PaperlessFileThumbnail: {Body: io.NopCloser(strings.NewReader("png")), ContentType: "image/webp", ContentLength: 3},
	}}
	svc, db, user, coin := setupPaperlessInvoiceService(t, client, true)

	if _, err := svc.OpenDocument(context.Background(), user.ID, coin.ID, PaperlessFilePreview); !errors.Is(err, ErrPaperlessNoDocument) {
		t.Fatalf("expected no document, got %v", err)
	}
	if err := db.Model(&models.Coin{}).Where("id = ?", coin.ID).Update("paperless_document_id", 55).Error; err != nil {
		t.Fatal(err)
	}
	file, err := svc.OpenDocument(context.Background(), user.ID, coin.ID, PaperlessFilePreview)
	if err != nil {
		t.Fatalf("open preview: %v", err)
	}
	body, _ := io.ReadAll(file.Body)
	if file.ContentType != "application/pdf" || string(body) != "%PDF" || client.fileDocID != 55 {
		t.Fatalf("unexpected preview %q %q doc=%d", file.ContentType, body, client.fileDocID)
	}
	if _, err := svc.OpenDocument(context.Background(), user.ID+99, coin.ID, PaperlessFilePreview); !errors.Is(err, ErrPaperlessCoinNotFound) {
		t.Fatalf("another user's coin must not be readable, got %v", err)
	}
}

func TestPaperlessInvoiceService_OpenDocumentRejectsUndisplayableTypes(t *testing.T) {
	for _, ct := range []string{"text/html; charset=utf-8", "image/svg+xml", "application/octet-stream", ""} {
		client := &fakePaperlessClient{files: map[PaperlessFileKind]*PaperlessFile{
			PaperlessFilePreview: {Body: io.NopCloser(strings.NewReader("<script>")), ContentType: ct, ContentLength: 8},
		}}
		svc, db, user, coin := setupPaperlessInvoiceService(t, client, true)
		db.Model(&models.Coin{}).Where("id = ?", coin.ID).Update("paperless_document_id", 5)
		if _, err := svc.OpenDocument(context.Background(), user.ID, coin.ID, PaperlessFilePreview); !errors.Is(err, ErrPaperlessUnsupportedFile) {
			t.Fatalf("content type %q must be rejected, got %v", ct, err)
		}
	}
}

func TestPaperlessInvoiceService_UnlinkKeepsPaperlessUntouched(t *testing.T) {
	client := &fakePaperlessClient{}
	svc, db, user, coin := setupPaperlessInvoiceService(t, client, true)
	db.Model(&models.Coin{}).Where("id = ?", coin.ID).Updates(map[string]interface{}{"paperless_document_id": 12, "paperless_task_id": "t"})
	if err := svc.UnlinkDocument(user.ID, coin.ID); err != nil {
		t.Fatalf("unlink: %v", err)
	}
	if storedDocumentID(t, db, coin.ID) != nil || storedTaskID(t, db, coin.ID) != "" {
		t.Fatal("unlink should clear the document and task on the coin")
	}
	if client.fileDocID != 0 || client.postedBody != "" {
		t.Fatal("unlink must not call paperless-ngx")
	}
}
