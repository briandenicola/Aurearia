package middleware

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestAuthenticatedRateLimit_AllowsNormalAuthenticatedBrowsingBurst(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("userId", uint(42))
		c.Next()
	})
	router.Use(AuthenticatedRateLimit(600, time.Minute))

	paths := []string{
		"/api/notifications/unread-count",
		"/api/auth/me",
		"/api/tags",
		"/api/coins",
		"/api/sets",
		"/api/storage-locations",
	}
	for _, path := range paths {
		router.GET(path, func(c *gin.Context) {
			c.Status(http.StatusOK)
		})
	}

	for i := 0; i < 150; i++ {
		path := paths[i%len(paths)]
		if path == "/api/coins" {
			path += "?page=" + strconv.Itoa((i%5)+1)
		}
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.RemoteAddr = "192.0.2.10:12345"
		router.ServeHTTP(w, req)

		if w.Code == http.StatusTooManyRequests {
			t.Fatalf("authenticated browsing request %d to %s was unexpectedly rate limited", i+1, path)
		}
		if w.Code != http.StatusOK {
			t.Fatalf("request %d to %s returned %d, want 200", i+1, path, w.Code)
		}
	}
}

func TestAuthenticatedRateLimit_KeysByUserNotSharedIP(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		userID, err := strconv.Atoi(c.GetHeader("X-Test-User-ID"))
		if err != nil {
			t.Fatalf("invalid test user id: %v", err)
		}
		c.Set("userId", uint(userID))
		c.Next()
	})
	router.Use(AuthenticatedRateLimit(2, time.Minute))
	router.GET("/api/auth/me", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	for i := 0; i < 2; i++ {
		w := performRateLimitRequest(router, "1")
		if w.Code != http.StatusOK {
			t.Fatalf("user 1 request %d returned %d, want 200", i+1, w.Code)
		}
	}
	w := performRateLimitRequest(router, "2")
	if w.Code != http.StatusOK {
		t.Fatalf("user 2 sharing the same client IP returned %d, want 200", w.Code)
	}
	w = performRateLimitRequest(router, "1")
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("user 1 third request returned %d, want 429", w.Code)
	}
}

func TestAuthenticatedRateLimit_StillLimitsAbusiveAuthenticatedRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("userId", uint(42))
		c.Next()
	})
	router.Use(AuthenticatedRateLimit(2, time.Minute))
	router.POST("/api/coins", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/coins", nil)
		req.RemoteAddr = "192.0.2.10:12345"
		router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("request %d returned %d, want 200", i+1, w.Code)
		}
	}
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/coins", nil)
	req.RemoteAddr = "192.0.2.10:12345"
	router.ServeHTTP(w, req)
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("third write returned %d, want 429", w.Code)
	}
}

func performRateLimitRequest(router *gin.Engine, userID string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	req.RemoteAddr = "192.0.2.10:12345"
	req.Header.Set("X-Test-User-ID", userID)
	router.ServeHTTP(w, req)
	return w
}
