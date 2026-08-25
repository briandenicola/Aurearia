package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/briandenicola/ancient-coins-api/middleware"
	"github.com/gin-gonic/gin"
)

// TestBackgroundRemovalWorkerAssetIsReachableWithWorkerCSP wires the CSP
// middleware ahead of static routes exactly as newHTTPRouter does (see
// bootstrap.go: router.Use(middleware.ContentSecurityPolicy()) runs before
// configureStaticRoutes registers r.Static("/assets", ...)), then confirms
// two things end-to-end through the real static file handler — not just the
// middleware in isolation:
//
//  1. A file placed under wwwroot/assets/workers/ is actually served by
//     Gin's static handler and its response carries the worker CSP
//     ('unsafe-eval' present). The filename is arbitrary on purpose — this
//     locks the /assets/workers/ path *namespace*, not a specific
//     Vite-generated hash, per the instruction not to assume generated
//     hashes.
//  2. A sibling file directly under wwwroot/assets/ (not under workers/)
//     is served by the same r.Static("/assets", ...) registration but
//     keeps the ordinary app CSP, with no 'unsafe-eval' anywhere — proving
//     the split is on request path, not on some property of the static
//     handler itself.
func TestBackgroundRemovalWorkerAssetIsReachableWithWorkerCSP(t *testing.T) {
	gin.SetMode(gin.TestMode)

	wwwroot := t.TempDir()
	workerDir := filepath.Join(wwwroot, "assets", "workers")
	if err := os.MkdirAll(workerDir, 0o755); err != nil {
		t.Fatalf("failed to create worker asset dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workerDir, "backgroundRemovalWorker-test.js"), []byte("/* worker chunk */"), 0o644); err != nil {
		t.Fatalf("failed to write worker chunk: %v", err)
	}
	if err := os.WriteFile(filepath.Join(wwwroot, "assets", "app-test.js"), []byte("/* app chunk */"), 0o644); err != nil {
		t.Fatalf("failed to write app chunk: %v", err)
	}

	router := gin.New()
	router.Use(middleware.ContentSecurityPolicy())
	configureStaticRoutes(router, wwwroot)

	workerReq := httptest.NewRequest(http.MethodGet, "/assets/workers/backgroundRemovalWorker-test.js", nil)
	workerW := httptest.NewRecorder()
	router.ServeHTTP(workerW, workerReq)

	if workerW.Code != http.StatusOK {
		t.Fatalf("worker asset: expected 200, got %d: %s", workerW.Code, workerW.Body.String())
	}
	workerCSP := workerW.Header().Get("Content-Security-Policy")
	if !strings.Contains(workerCSP, "'unsafe-eval'") {
		t.Fatalf("worker asset response must carry 'unsafe-eval': %s", workerCSP)
	}

	appReq := httptest.NewRequest(http.MethodGet, "/assets/app-test.js", nil)
	appW := httptest.NewRecorder()
	router.ServeHTTP(appW, appReq)

	if appW.Code != http.StatusOK {
		t.Fatalf("app asset: expected 200, got %d: %s", appW.Code, appW.Body.String())
	}
	appCSPHeader := appW.Header().Get("Content-Security-Policy")
	if strings.Contains(appCSPHeader, "'unsafe-eval'") {
		t.Fatalf("app asset response must not carry 'unsafe-eval': %s", appCSPHeader)
	}
}

func TestConfigureStaticRoutesServesBackgroundRemovalAssets(t *testing.T) {
	gin.SetMode(gin.TestMode)

	wwwroot := t.TempDir()
	assetDir := filepath.Join(wwwroot, "imgly-background-removal")
	if err := os.MkdirAll(assetDir, 0o755); err != nil {
		t.Fatalf("failed to create asset dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(wwwroot, "index.html"), []byte("<html>spa</html>"), 0o644); err != nil {
		t.Fatalf("failed to write index: %v", err)
	}
	if err := os.WriteFile(filepath.Join(assetDir, "resources.json"), []byte(`{"ok":true}`), 0o644); err != nil {
		t.Fatalf("failed to write resources: %v", err)
	}
	if err := os.WriteFile(filepath.Join(assetDir, "chunkhash"), []byte("model-chunk"), 0o644); err != nil {
		t.Fatalf("failed to write chunk: %v", err)
	}

	router := gin.New()
	configureStaticRoutes(router, wwwroot)

	for _, tc := range []struct {
		path string
		want string
	}{
		{path: "/imgly-background-removal/resources.json", want: `{"ok":true}`},
		{path: "/imgly-background-removal/chunkhash", want: "model-chunk"},
	} {
		req := httptest.NewRequest(http.MethodGet, tc.path, nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("%s: expected 200, got %d: %s", tc.path, w.Code, w.Body.String())
		}
		if got := w.Body.String(); got != tc.want {
			t.Fatalf("%s: expected %q, got %q", tc.path, tc.want, got)
		}
	}
}
