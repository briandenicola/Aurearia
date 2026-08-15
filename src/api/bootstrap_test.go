package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCORSMiddlewareAllowsOnlyConfiguredOrigins(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(corsMiddleware([]string{"https://coins.example.com"}))
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	allowed := httptest.NewRecorder()
	allowedRequest := httptest.NewRequest(http.MethodGet, "/test", nil)
	allowedRequest.Header.Set("Origin", "https://coins.example.com")
	router.ServeHTTP(allowed, allowedRequest)
	if got := allowed.Header().Get("Access-Control-Allow-Origin"); got != "https://coins.example.com" {
		t.Fatalf("allowed origin header = %q", got)
	}
	if got := allowed.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Fatalf("allow credentials header = %q", got)
	}

	rejected := httptest.NewRecorder()
	rejectedRequest := httptest.NewRequest(http.MethodGet, "/test", nil)
	rejectedRequest.Header.Set("Origin", "https://attacker.example")
	router.ServeHTTP(rejected, rejectedRequest)
	if got := rejected.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("rejected origin header = %q, want empty", got)
	}
}

func TestCORSMiddlewareAnswersPreflightWithoutCallingHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handlerCalled := false
	router.Use(corsMiddleware([]string{"https://coins.example.com"}))
	router.OPTIONS("/test", func(c *gin.Context) {
		handlerCalled = true
		c.Status(http.StatusNoContent)
	})

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodOptions, "/test", nil)
	request.Header.Set("Origin", "https://coins.example.com")
	router.ServeHTTP(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf("preflight status = %d, want %d", response.Code, http.StatusNoContent)
	}
	if handlerCalled {
		t.Fatal("preflight called downstream handler")
	}
}
