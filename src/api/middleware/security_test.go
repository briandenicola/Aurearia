package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestSecurityHeadersEnableCrossOriginIsolationForWASM(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(SecurityHeaders())
	router.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if got := w.Header().Get("Cross-Origin-Opener-Policy"); got != "same-origin" {
		t.Fatalf("expected COOP same-origin, got %q", got)
	}
	if got := w.Header().Get("Cross-Origin-Embedder-Policy"); got != "credentialless" {
		t.Fatalf("expected COEP credentialless, got %q", got)
	}
}

func TestIPDenyRulesBlocksPublicAndProtectedRoutes(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	if err := db.AutoMigrate(&models.IPRule{}, &models.SecurityEvent{}); err != nil {
		t.Fatalf("failed to migrate ip rules: %v", err)
	}
	securitySvc := services.NewSecurityService(repository.NewSecurityRepository(db))
	if err := securitySvc.CreateIPRule("198.51.100.0/24", "blocked network", time.Now().Add(time.Hour), nil); err != nil {
		t.Fatalf("failed to create ban: %v", err)
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(ResolvedClientIP(), IPDenyRules(securitySvc))
	router.POST("/api/auth/login", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	router.GET("/api/coins", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	for _, path := range []string{"/api/auth/login", "/api/coins"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		if path == "/api/auth/login" {
			req.Method = http.MethodPost
		}
		req.RemoteAddr = "198.51.100.25:12345"
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("%s: expected banned IP to receive 403, got %d: %s", path, w.Code, w.Body.String())
		}
	}
}

func TestResolvedClientIPIgnoresSpoofedForwardedForFromUntrustedPeer(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	if err := router.SetTrustedProxies([]string{"10.0.0.0/8"}); err != nil {
		t.Fatalf("failed to configure trusted proxies: %v", err)
	}
	router.Use(ResolvedClientIP())
	router.GET("/ip", func(c *gin.Context) {
		c.String(http.StatusOK, ClientIP(c))
	})

	req := httptest.NewRequest(http.MethodGet, "/ip", nil)
	req.RemoteAddr = "198.51.100.10:12345"
	req.Header.Set("X-Forwarded-For", "203.0.113.99")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if got := w.Body.String(); got != "198.51.100.10" {
		t.Fatalf("expected untrusted peer IP, got %q", got)
	}
}

func TestResolvedClientIPUsesForwardedForFromTrustedProxy(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	if err := router.SetTrustedProxies([]string{"10.0.0.0/8"}); err != nil {
		t.Fatalf("failed to configure trusted proxies: %v", err)
	}
	router.Use(ResolvedClientIP())
	router.GET("/ip", func(c *gin.Context) {
		c.String(http.StatusOK, ClientIP(c))
	})

	req := httptest.NewRequest(http.MethodGet, "/ip", nil)
	req.RemoteAddr = "10.1.2.3:12345"
	req.Header.Set("X-Forwarded-For", "203.0.113.99")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if got := w.Body.String(); got != "203.0.113.99" {
		t.Fatalf("expected forwarded client IP from trusted proxy, got %q", got)
	}
}
