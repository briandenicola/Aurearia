package middleware

import (
	"net/http"
	"strings"

	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
)

const clientIPContextKey = "clientIP"

func ResolvedClientIP() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(clientIPContextKey, c.ClientIP())
		c.Next()
	}
}

func ClientIP(c *gin.Context) string {
	if value, ok := c.Get(clientIPContextKey); ok {
		if ip, ok := value.(string); ok {
			return ip
		}
	}
	return c.ClientIP()
}

// isHTTPS reports whether the original client connection was TLS-terminated,
// either directly or at a trusted reverse proxy. c.Request.TLS covers the
// direct case; X-Forwarded-Proto covers the proxied one and is only
// trustworthy because gin strips forwarded headers from untrusted peers (see
// SetTrustedProxies in bootstrap.go).
func isHTTPS(c *gin.Context) bool {
	if c.Request.TLS != nil {
		return true
	}
	return strings.EqualFold(c.GetHeader("X-Forwarded-Proto"), "https")
}

func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		// Only sent over TLS. Browsers ignore HSTS on a plaintext response,
		// and emitting it unconditionally would be a trap for the LAN-only
		// deployments this app explicitly supports.
		if isHTTPS(c) {
			c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("X-Frame-Options", "DENY")
		c.Header("Cross-Origin-Opener-Policy", "same-origin")
		c.Header("Cross-Origin-Embedder-Policy", "credentialless")
		c.Header("Permissions-Policy", "camera=(self), microphone=(), geolocation=()")
		c.Next()
	}
}

func IPDenyRules(securitySvc *services.SecurityService) gin.HandlerFunc {
	return func(c *gin.Context) {
		if securitySvc != nil && securitySvc.IsIPDenied(ClientIP(c)) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Access denied"})
			return
		}
		c.Next()
	}
}
