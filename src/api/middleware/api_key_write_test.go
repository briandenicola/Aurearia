package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// setupWriteGuardRouter simulates the protected group: an auth step that
// sets apiKeyCapabilities only for API-key requests, then the write guard.
func setupWriteGuardRouter(capabilities *string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		if capabilities != nil {
			c.Set("apiKeyCapabilities", *capabilities)
		}
		c.Next()
	})
	r.Use(RequireAPIKeyWriteForChanges())
	ok := func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) }
	r.GET("/coins", ok)
	r.HEAD("/coins", ok)
	r.OPTIONS("/coins", ok)
	r.POST("/coins", ok)
	r.PUT("/coins/1", ok)
	r.PATCH("/coins/1", ok)
	r.DELETE("/coins/1", ok)
	return r
}

func writeGuardStatus(r *gin.Engine, method, path string) int {
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(method, path, nil))
	return w.Code
}

func TestRequireAPIKeyWriteForChanges_ReadKeyCanOnlyRead(t *testing.T) {
	caps := "read"
	r := setupWriteGuardRouter(&caps)
	for _, m := range []string{http.MethodGet, http.MethodHead, http.MethodOptions} {
		if got := writeGuardStatus(r, m, "/coins"); got != http.StatusOK {
			t.Errorf("read key %s = %d, want 200", m, got)
		}
	}
	for _, tc := range []struct{ method, path string }{
		{http.MethodPost, "/coins"}, {http.MethodPut, "/coins/1"}, {http.MethodPatch, "/coins/1"}, {http.MethodDelete, "/coins/1"},
	} {
		if got := writeGuardStatus(r, tc.method, tc.path); got != http.StatusForbidden {
			t.Errorf("read key %s = %d, want 403", tc.method, got)
		}
	}
}

func TestRequireAPIKeyWriteForChanges_CopilotOnlyKeyCannotWrite(t *testing.T) {
	caps := "read,copilot"
	if got := writeGuardStatus(setupWriteGuardRouter(&caps), http.MethodPost, "/coins"); got != http.StatusForbidden {
		t.Errorf("read+copilot key POST = %d, want 403", got)
	}
}

func TestRequireAPIKeyWriteForChanges_WriteKeyUnchanged(t *testing.T) {
	for _, caps := range []string{"write", "read,write", "read, write,copilot"} {
		c := caps
		r := setupWriteGuardRouter(&c)
		for _, tc := range []struct{ method, path string }{
			{http.MethodGet, "/coins"}, {http.MethodPost, "/coins"}, {http.MethodPut, "/coins/1"}, {http.MethodDelete, "/coins/1"},
		} {
			if got := writeGuardStatus(r, tc.method, tc.path); got != http.StatusOK {
				t.Errorf("key %q %s = %d, want 200", caps, tc.method, got)
			}
		}
	}
}

func TestRequireAPIKeyWriteForChanges_JWTRequestsUnchanged(t *testing.T) {
	r := setupWriteGuardRouter(nil) // no apiKeyCapabilities: a browser (JWT) request
	for _, tc := range []struct{ method, path string }{
		{http.MethodGet, "/coins"}, {http.MethodPost, "/coins"}, {http.MethodPut, "/coins/1"}, {http.MethodDelete, "/coins/1"},
	} {
		if got := writeGuardStatus(r, tc.method, tc.path); got != http.StatusOK {
			t.Errorf("JWT %s = %d, want 200", tc.method, got)
		}
	}
}

func TestRequireAPIKeyWriteForChanges_MalformedCapabilitiesRefused(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("apiKeyCapabilities", 42); c.Next() })
	r.Use(RequireAPIKeyWriteForChanges())
	r.POST("/coins", func(c *gin.Context) { c.Status(http.StatusOK) })
	if got := writeGuardStatus(r, http.MethodPost, "/coins"); got != http.StatusForbidden {
		t.Errorf("non-string capabilities POST = %d, want 403", got)
	}
}
