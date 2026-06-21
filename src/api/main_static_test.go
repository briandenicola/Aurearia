package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
)

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
