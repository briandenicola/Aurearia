package middleware

import (
	"net/http"

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

func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("X-Frame-Options", "DENY")
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
