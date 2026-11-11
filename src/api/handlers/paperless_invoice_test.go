package handlers

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

type stubPaperlessClient struct {
	contentType string
	body        string
	pingErr     error
}

func (s *stubPaperlessClient) PostDocument(context.Context, string, string, string, string, string, io.Reader) (string, error) {
	return "", errors.New("not used")
}

func (s *stubPaperlessClient) Ping(context.Context, string, string) error { return s.pingErr }

func (s *stubPaperlessClient) GetTask(context.Context, string, string, string) (*services.PaperlessTask, error) {
	return nil, errors.New("not used")
}

func (s *stubPaperlessClient) GetDocumentFile(_ context.Context, _, _ string, _ int, _ services.PaperlessFileKind) (*services.PaperlessFile, error) {
	return &services.PaperlessFile{Body: io.NopCloser(strings.NewReader(s.body)), ContentType: s.contentType, ContentLength: int64(len(s.body))}, nil
}

func setupPaperlessInvoiceRouter(t *testing.T, client services.PaperlessClient) (*gin.Engine, *gorm.DB, models.Coin) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.StorageLocation{}, &models.MintLocation{}, &models.Tag{}, &models.CoinSet{}, &models.Coin{}, &models.CoinImage{}, &models.CoinReference{}); err != nil {
		t.Fatal(err)
	}
	user := models.User{Username: "collector", PasswordHash: "x", PaperlessURL: "https://paperless.example.com", PaperlessAPIToken: "tkn"}
	db.Create(&user)
	coin := models.Coin{Name: "Denarius", UserID: user.ID}
	db.Create(&coin)

	svc := services.NewPaperlessInvoiceService(repository.NewCoinRepository(db), repository.NewUserRepository(db), nil, client)
	h := NewPaperlessInvoiceHandler(svc)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("userId", user.ID); c.Next() })
	r.GET("/api/coins/:id/invoice/paperless/document", h.InvoiceDocument)
	r.DELETE("/api/coins/:id/invoice/paperless", h.UnlinkInvoice)
	r.POST("/api/paperless/test", h.TestConnection)
	return r, db, coin
}

func TestPaperlessTestConnection_ReportsSuccess(t *testing.T) {
	router, _, _ := setupPaperlessInvoiceRouter(t, &stubPaperlessClient{})
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/paperless/test", nil))
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"ok":true`) {
		t.Fatalf("expected ok, got %d %s", w.Code, w.Body.String())
	}
}

func TestPaperlessTestConnection_ReportsTheActualCause(t *testing.T) {
	dnsErr := &net.DNSError{Err: "no such host", Name: "paperless-ngx", IsNotFound: true}
	router, _, _ := setupPaperlessInvoiceRouter(t, &stubPaperlessClient{pingErr: dnsErr})
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/paperless/test", nil))
	if w.Code != http.StatusBadGateway || !strings.Contains(w.Body.String(), `could not resolve \"paperless-ngx\"`) {
		t.Fatalf("expected 502 naming the DNS failure, got %d %s", w.Code, w.Body.String())
	}
}

func TestPaperlessInvoiceDocument_StreamsWithoutCachingAndSandboxed(t *testing.T) {
	router, db, coin := setupPaperlessInvoiceRouter(t, &stubPaperlessClient{contentType: "application/pdf", body: "%PDF-1.7"})
	db.Model(&models.Coin{}).Where("id = ?", coin.ID).Update("paperless_document_id", 9)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/coins/1/invoice/paperless/document", nil))

	if w.Code != http.StatusOK || w.Body.String() != "%PDF-1.7" {
		t.Fatalf("expected the PDF, got %d %q", w.Code, w.Body.String())
	}
	for header, want := range map[string]string{
		"Content-Type":            "application/pdf",
		"Cache-Control":           "no-store",
		"X-Content-Type-Options":  "nosniff",
		"Content-Security-Policy": "default-src 'none'; sandbox",
	} {
		if got := w.Header().Get(header); got != want {
			t.Errorf("%s = %q, want %q", header, got, want)
		}
	}
}

func TestPaperlessInvoiceDocument_NoLinkedDocumentIs404(t *testing.T) {
	router, _, _ := setupPaperlessInvoiceRouter(t, &stubPaperlessClient{contentType: "application/pdf", body: "x"})
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/coins/1/invoice/paperless/document", nil))
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestPaperlessInvoiceDocument_RefusesHTML(t *testing.T) {
	router, db, coin := setupPaperlessInvoiceRouter(t, &stubPaperlessClient{contentType: "text/html", body: "<script>alert(1)</script>"})
	db.Model(&models.Coin{}).Where("id = ?", coin.ID).Update("paperless_document_id", 9)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/coins/1/invoice/paperless/document", nil))
	if w.Code != http.StatusBadGateway || strings.Contains(w.Body.String(), "<script>") {
		t.Fatalf("HTML from paperless must never be served, got %d %q", w.Code, w.Body.String())
	}
}

func TestPaperlessInvoiceUnlink_Returns204(t *testing.T) {
	router, db, coin := setupPaperlessInvoiceRouter(t, &stubPaperlessClient{})
	db.Model(&models.Coin{}).Where("id = ?", coin.ID).Update("paperless_document_id", 9)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodDelete, "/api/coins/1/invoice/paperless", nil))
	var reloaded models.Coin
	db.First(&reloaded, coin.ID)
	if w.Code != http.StatusNoContent || reloaded.PaperlessDocumentID != nil {
		t.Fatalf("expected 204 and cleared link, got %d %v", w.Code, reloaded.PaperlessDocumentID)
	}
}
