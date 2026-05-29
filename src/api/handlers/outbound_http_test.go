package handlers

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"

	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
)

type fakeResolver struct {
	lookup func(ctx context.Context, network, host string) ([]netip.Addr, error)
}

func (f fakeResolver) LookupNetIP(ctx context.Context, network, host string) ([]netip.Addr, error) {
	return f.lookup(ctx, network, host)
}

func TestValidateOutboundURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		rawURL  string
		wantErr bool
	}{
		{name: "allows public https URL", rawURL: "https://example.com/image.jpg", wantErr: false},
		{name: "blocks loopback IP", rawURL: "http://127.0.0.1/image.jpg", wantErr: true},
		{name: "blocks localhost hostname", rawURL: "http://localhost/image.jpg", wantErr: true},
		{name: "blocks link local IP", rawURL: "http://169.254.169.254/latest/meta-data", wantErr: true},
		{name: "blocks unsupported scheme", rawURL: "file:///etc/passwd", wantErr: true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := validateOutboundURL(tt.rawURL)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateOutboundURL() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRestrictedDialContext_BlocksPrivateResolution(t *testing.T) {
	t.Parallel()

	dial := restrictedDialContext(fakeResolver{
		lookup: func(ctx context.Context, network, host string) ([]netip.Addr, error) {
			return []netip.Addr{netip.MustParseAddr("127.0.0.1")}, nil
		},
	})

	_, err := dial(context.Background(), "tcp", "example.com:80")
	if err == nil {
		t.Fatal("expected blocked outbound address error")
	}
	if !isOutboundTargetBlockedError(err) {
		t.Fatalf("expected blocked outbound address error, got %v", err)
	}
}

func TestRestrictedDialContext_ResolvesPerConnectAttempt(t *testing.T) {
	t.Parallel()

	lookups := 0
	dial := restrictedDialContext(fakeResolver{
		lookup: func(ctx context.Context, network, host string) ([]netip.Addr, error) {
			lookups++
			return []netip.Addr{netip.MustParseAddr("127.0.0.1")}, nil
		},
	})

	_, _ = dial(context.Background(), "tcp", "example.com:80")
	_, _ = dial(context.Background(), "tcp", "example.com:80")

	if lookups != 2 {
		t.Fatalf("expected resolver to be called on each connect attempt, got %d calls", lookups)
	}
}

func TestProxyImage_BlocksConnectTimePrivateResolution(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldFactory := outboundHTTPClientFactory
	outboundHTTPClientFactory = func() *http.Client {
		return newRestrictedHTTPClient(fakeResolver{
			lookup: func(ctx context.Context, network, host string) ([]netip.Addr, error) {
				return []netip.Addr{netip.MustParseAddr("127.0.0.1")}, nil
			},
		})
	}
	t.Cleanup(func() { outboundHTTPClientFactory = oldFactory })

	handler := &ImageHandler{logger: services.NewLogger(10)}
	router := gin.New()
	router.GET("/proxy-image", handler.ProxyImage)

	req := httptest.NewRequest(http.MethodGet, "/proxy-image?url=http://example.com/image.jpg", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d (%s)", w.Code, w.Body.String())
	}
}

func TestScrapeImage_BlocksConnectTimePrivateResolution(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldFactory := outboundHTTPClientFactory
	outboundHTTPClientFactory = func() *http.Client {
		return newRestrictedHTTPClient(fakeResolver{
			lookup: func(ctx context.Context, network, host string) ([]netip.Addr, error) {
				return []netip.Addr{netip.MustParseAddr("127.0.0.1")}, nil
			},
		})
	}
	t.Cleanup(func() { outboundHTTPClientFactory = oldFactory })

	handler := &ImageHandler{logger: services.NewLogger(10)}
	router := gin.New()
	router.GET("/scrape-image", handler.ScrapeImage)

	req := httptest.NewRequest(http.MethodGet, "/scrape-image?url=http://example.com/page", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d (%s)", w.Code, w.Body.String())
	}
}

func TestRedirectPolicy_BlocksDisallowedHost(t *testing.T) {
	t.Parallel()

	client := newRestrictedHTTPClient(nil)
	req, err := http.NewRequest(http.MethodGet, "http://127.0.0.1/redirect", nil)
	if err != nil {
		t.Fatalf("failed to build request: %v", err)
	}
	prev, err := http.NewRequest(http.MethodGet, "https://example.com/start", nil)
	if err != nil {
		t.Fatalf("failed to build previous request: %v", err)
	}

	err = client.CheckRedirect(req, []*http.Request{prev})
	if err == nil {
		t.Fatal("expected redirect to be rejected")
	}
	if !isOutboundTargetBlockedError(err) {
		t.Fatalf("expected blocked outbound error, got %v", err)
	}
}

func TestIsOutboundTargetBlockedError_Unwraps(t *testing.T) {
	t.Parallel()
	err := fmt.Errorf("wrapped: %w", errOutboundTargetBlocked)
	if !isOutboundTargetBlockedError(err) {
		t.Fatal("expected wrapped error to be detected")
	}
}
