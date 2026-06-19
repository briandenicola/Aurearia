package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// setupCapabilityTestRouter creates a test router with a mock handler that sets capabilities
// and a protected route that requires a specific capability scope.
func setupCapabilityTestRouter(capabilities string, scope string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	// Mock middleware that sets capabilities in context (simulates what auth middleware does)
	r.Use(func(c *gin.Context) {
		if capabilities != "" {
			c.Set("apiKeyCapabilities", capabilities)
		}
		c.Next()
	})

	// Apply RequireCapability middleware
	r.Use(RequireCapability(scope))

	// Protected route
	r.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	return r
}

// Test 1: Read-scoped key can access read-scoped endpoint
func TestRequireCapability_ReadKeyAllowsRead(t *testing.T) {
	router := setupCapabilityTestRouter("read", "read")

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for read-scoped key on read endpoint, got %d: %s", w.Code, w.Body.String())
	}
}

// Test 2: Read-scoped key cannot access write-scoped endpoint
func TestRequireCapability_ReadKeyDeniesWrite(t *testing.T) {
	router := setupCapabilityTestRouter("read", "write")

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 for read-scoped key on write endpoint, got %d", w.Code)
	}
}

// Test 3: Write-scoped key can access write-scoped endpoint
func TestRequireCapability_WriteKeyAllowsWrite(t *testing.T) {
	router := setupCapabilityTestRouter("read,write", "write")

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for write-scoped key on write endpoint, got %d: %s", w.Code, w.Body.String())
	}
}

// Test 4: Write-scoped key can access read-scoped endpoint (write implies read)
func TestRequireCapability_WriteKeyAllowsRead(t *testing.T) {
	router := setupCapabilityTestRouter("read,write", "read")

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for write-scoped key on read endpoint, got %d: %s", w.Code, w.Body.String())
	}
}

// Test 5: No capability in context (JWT-style auth) is denied
func TestRequireCapability_NoCapabilityDenied(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	// No capability set in context (simulates JWT auth without API key)
	r.Use(RequireCapability("read"))

	r.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 when no capability in context, got %d", w.Code)
	}
}

// Test 6: Empty capability string is denied
func TestRequireCapability_EmptyCapabilityDenied(t *testing.T) {
	router := setupCapabilityTestRouter("", "read")

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 for empty capability string, got %d", w.Code)
	}
}

// Test 7: Invalid scope parameter is denied
func TestRequireCapability_InvalidScopeDenied(t *testing.T) {
	router := setupCapabilityTestRouter("read,write", "invalid")

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 for invalid scope, got %d", w.Code)
	}
}

// Test 8: Non-string capability value in context is denied
func TestRequireCapability_NonStringCapabilityDenied(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	// Set a non-string value in context (type mismatch)
	r.Use(func(c *gin.Context) {
		c.Set("apiKeyCapabilities", 12345) // integer instead of string
		c.Next()
	})

	r.Use(RequireCapability("read"))

	r.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 for non-string capability value, got %d", w.Code)
	}
}

// Test 9: Protected handler does not execute when capability check fails
func TestRequireCapability_HandlerNotExecutedOnDeny(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	r.Use(RequireCapability("read"))

	handlerExecuted := false
	r.GET("/protected", func(c *gin.Context) {
		handlerExecuted = true
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if handlerExecuted {
		t.Error("protected handler should not execute when capability check fails")
	}

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

// Test 10: Protected handler executes when capability check succeeds
func TestRequireCapability_HandlerExecutedOnAllow(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	// Set valid read capability
	r.Use(func(c *gin.Context) {
		c.Set("apiKeyCapabilities", "read")
		c.Next()
	})

	r.Use(RequireCapability("read"))

	handlerExecuted := false
	r.GET("/protected", func(c *gin.Context) {
		handlerExecuted = true
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if !handlerExecuted {
		t.Error("protected handler should execute when capability check succeeds")
	}

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRequireCapability_RejectsSubstringMatches(t *testing.T) {
	testCases := []struct {
		name         string
		capabilities string
		scope        string
	}{
		{name: "readwrite does not grant read", capabilities: "readwrite", scope: "read"},
		{name: "readwrite does not grant write", capabilities: "readwrite", scope: "write"},
		{name: "xwritex does not grant write", capabilities: "xwritex", scope: "write"},
		{name: "xwritex does not grant read", capabilities: "xwritex", scope: "read"},
		{name: "notread does not grant read", capabilities: "notread", scope: "read"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			router := setupCapabilityTestRouter(tc.capabilities, tc.scope)
			req := httptest.NewRequest(http.MethodGet, "/protected", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != http.StatusForbidden {
				t.Fatalf("expected 403 for malformed capabilities %q requiring %q, got %d: %s", tc.capabilities, tc.scope, w.Code, w.Body.String())
			}
		})
	}
}
