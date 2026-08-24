package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func cspForPath(t *testing.T, path string) string {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(ContentSecurityPolicy())
	router.GET(path, func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	req := httptest.NewRequest(http.MethodGet, path, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w.Header().Get("Content-Security-Policy")
}

func TestContentSecurityPolicyIsSetOnAppRoutes(t *testing.T) {
	csp := cspForPath(t, "/api/coins")
	if csp == "" {
		t.Fatal("expected a Content-Security-Policy header")
	}

	// The SPA keeps its access token in web storage, so blocking injected
	// script is the whole point of this header. If a future change relaxes
	// script-src to 'unsafe-inline' or 'unsafe-eval', that protection is gone.
	for _, forbidden := range []string{"script-src 'self' 'unsafe-inline'", "'unsafe-eval'"} {
		if strings.Contains(csp, forbidden) {
			t.Fatalf("app CSP must not contain %q: %s", forbidden, csp)
		}
	}

	required := []string{
		"default-src 'self'",
		"object-src 'none'",
		"frame-ancestors 'none'",
		"base-uri 'self'",
		// @imgly/background-removal compiles an ONNX model to WASM and
		// dynamically imports its worker/WASM glue as a blob: module.
		"script-src 'self' 'wasm-unsafe-eval' blob:",
		// Proxied auction images arrive as blob: URLs; the mint map needs OSM tiles.
		"img-src 'self' data: blob: " + OSMTileHost,
		"worker-src 'self' blob:",
	}
	for _, directive := range required {
		if !strings.Contains(csp, directive) {
			t.Fatalf("app CSP missing %q: %s", directive, csp)
		}
	}
}

// TestContentSecurityPolicyAllowsBackgroundRemovalBlobScriptImport locks the
// exact production fix: @imgly/background-removal dynamically imports a
// blob: URL as a script/module (not just a worker or fetch target), so
// script-src — not just worker-src/connect-src — must permit blob:. Without
// this, the browser blocks the dynamic import with "Failed to fetch
// dynamically imported module" and background removal reports no available
// backend. This must never regress to bare 'self' 'wasm-unsafe-eval', and
// must never gain 'unsafe-inline' or 'unsafe-eval' to get there.
func TestContentSecurityPolicyAllowsBackgroundRemovalBlobScriptImport(t *testing.T) {
	csp := cspForPath(t, "/api/coins")

	if !strings.Contains(csp, "script-src 'self' 'wasm-unsafe-eval' blob:") {
		t.Fatalf("app CSP must allow blob: script imports for background removal: %s", csp)
	}

	for _, forbidden := range []string{"'unsafe-inline'", "'unsafe-eval'"} {
		var scriptSrc string
		for _, directive := range strings.Split(csp, "; ") {
			if strings.HasPrefix(directive, "script-src") {
				scriptSrc = directive
				break
			}
		}
		if strings.Contains(scriptSrc, forbidden) {
			t.Fatalf("script-src must not contain %q: %s", forbidden, scriptSrc)
		}
	}
}

func TestContentSecurityPolicyRelaxesOnlySwagger(t *testing.T) {
	swagger := cspForPath(t, "/swagger/index.html")
	if !strings.Contains(swagger, "script-src 'self' 'unsafe-inline'") {
		t.Fatalf("swagger CSP should allow its inline bootstrap script: %s", swagger)
	}
	if strings.Contains(swagger, OSMTileHost) {
		t.Fatalf("swagger CSP should not carry SPA-specific allowances: %s", swagger)
	}
}

func TestStrictTransportSecurityOnlyOverTLS(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	if err := router.SetTrustedProxies([]string{"10.0.0.0/8"}); err != nil {
		t.Fatalf("failed to configure trusted proxies: %v", err)
	}
	router.Use(SecurityHeaders())
	router.GET("/", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	// Plaintext LAN deployment: no HSTS, or the user can lock themselves out.
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "192.168.1.10:1234"
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if got := w.Header().Get("Strict-Transport-Security"); got != "" {
		t.Fatalf("expected no HSTS on a plaintext request, got %q", got)
	}

	// Behind a TLS-terminating reverse proxy: HSTS is expected.
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.1.2.3:1234"
	req.Header.Set("X-Forwarded-Proto", "https")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if got := w.Header().Get("Strict-Transport-Security"); got == "" {
		t.Fatal("expected HSTS when forwarded proto is https")
	}
}
