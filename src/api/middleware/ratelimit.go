package middleware

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type rateLimiter struct {
	mu      sync.Mutex
	clients map[string]*clientState
	limit   int
	window  time.Duration
}

type clientState struct {
	count   int
	resetAt time.Time
}

func newRateLimiter(limit int, window time.Duration) *rateLimiter {
	rl := &rateLimiter{
		clients: make(map[string]*clientState),
		limit:   limit,
		window:  window,
	}
	// Periodic cleanup of expired entries
	go func() {
		for {
			time.Sleep(window * 2)
			rl.mu.Lock()
			now := time.Now()
			for ip, cs := range rl.clients {
				if now.After(cs.resetAt) {
					delete(rl.clients, ip)
				}
			}
			rl.mu.Unlock()
		}
	}()
	return rl
}

func (rl *rateLimiter) allow(clientIP string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cs, exists := rl.clients[clientIP]
	if !exists || now.After(cs.resetAt) {
		rl.clients[clientIP] = &clientState{count: 1, resetAt: now.Add(rl.window)}
		return true
	}
	cs.count++
	return cs.count <= rl.limit
}

// RateLimit returns middleware that limits requests per IP.
// limit: max requests allowed within the window duration.
func RateLimit(limit int, window time.Duration) gin.HandlerFunc {
	rl := newRateLimiter(limit, window)
	return func(c *gin.Context) {
		if !rl.allow(ClientIP(c)) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "Too many requests. Please try again later.",
			})
			return
		}
		c.Next()
	}
}

// AuthenticatedRateLimit returns middleware that limits requests per authenticated
// user/API key, with client IP as a fallback for routes that run before auth.
func AuthenticatedRateLimit(limit int, window time.Duration) gin.HandlerFunc {
	return rateLimitByKey(limit, window, authenticatedRateLimitKey)
}

func rateLimitByKey(limit int, window time.Duration, keyFunc func(*gin.Context) string) gin.HandlerFunc {
	rl := newRateLimiter(limit, window)
	return func(c *gin.Context) {
		if !rl.allow(keyFunc(c)) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "Too many requests. Please try again later.",
			})
			return
		}
		c.Next()
	}
}

func authenticatedRateLimitKey(c *gin.Context) string {
	if apiKeyId, exists := c.Get("apiKeyId"); exists {
		if id, ok := apiKeyId.(uint); ok {
			return fmt.Sprintf("apikey:%d", id)
		}
	}
	if userID, exists := c.Get("userId"); exists {
		if id, ok := userID.(uint); ok {
			return fmt.Sprintf("user:%d", id)
		}
	}
	return ClientIP(c)
}

// ExternalAPIKeyRateLimit returns middleware that enforces stricter per-key rate limiting
// for external tool server endpoints. Keys by API key ID (preferred) or falls back to client IP.
// Provides a single unified bucket; read/write distinction is a future enhancement.
func ExternalAPIKeyRateLimit(limit int, window time.Duration) gin.HandlerFunc {
	return rateLimitByKey(limit, window, authenticatedRateLimitKey)
}
