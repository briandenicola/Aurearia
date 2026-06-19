package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestClientIPIgnoresForwardedHeadersFromUntrustedSource(t *testing.T) {
	router := clientIPTestRouter(t, []string{"10.0.0.1"})

	req := httptest.NewRequest(http.MethodGet, "/client-ip", nil)
	req.RemoteAddr = "203.0.113.50:34567"
	req.Header.Set("X-Forwarded-For", "198.51.100.25")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if got := w.Body.String(); got != "203.0.113.50" {
		t.Fatalf("expected untrusted X-Forwarded-For to be ignored, got client IP %q", got)
	}
}

func TestClientIPHonorsForwardedHeadersThroughTrustedProxy(t *testing.T) {
	router := clientIPTestRouter(t, []string{"10.0.0.1"})

	req := httptest.NewRequest(http.MethodGet, "/client-ip", nil)
	req.RemoteAddr = "10.0.0.1:34567"
	req.Header.Set("X-Forwarded-For", "198.51.100.25")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if got := w.Body.String(); got != "198.51.100.25" {
		t.Fatalf("expected trusted proxy X-Forwarded-For to be honored, got client IP %q", got)
	}
}

func clientIPTestRouter(t *testing.T, trustedProxies []string) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	if err := router.SetTrustedProxies(trustedProxies); err != nil {
		t.Fatalf("failed to configure trusted proxies: %v", err)
	}
	router.GET("/client-ip", func(c *gin.Context) {
		c.String(http.StatusOK, c.ClientIP())
	})
	return router
}
