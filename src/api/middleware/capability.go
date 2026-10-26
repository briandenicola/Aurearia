package middleware

import (
	"net/http"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/gin-gonic/gin"
)

// RequireCapability returns middleware that enforces a specific capability (read or write)
// for API key authenticated requests. Returns 403 if the required capability is missing.
func RequireCapability(scope string) gin.HandlerFunc {
	return func(c *gin.Context) {
		capabilities, exists := c.Get("apiKeyCapabilities")
		if !exists {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Insufficient capability"})
			return
		}

		capStr, ok := capabilities.(string)
		if !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Insufficient capability"})
			return
		}

		if !models.HasAPICapability(capStr, scope) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Insufficient capability"})
			return
		}

		c.Next()
	}
}

// RequireAPIKeyWriteForChanges lets an API key change data only when it has
// write capability. A request authenticated by API key with a method other
// than GET, HEAD or OPTIONS is refused with 403 unless the key has write.
// JWT (browser) requests and API-key reads pass unchanged. Mount it on route
// groups whose handlers do not check capabilities themselves.
func RequireAPIKeyWriteForChanges() gin.HandlerFunc {
	return func(c *gin.Context) {
		capabilities, isAPIKey := c.Get("apiKeyCapabilities")
		if !isAPIKey {
			c.Next()
			return
		}
		switch c.Request.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			c.Next()
			return
		}
		capStr, ok := capabilities.(string)
		if !ok || !models.HasAPICapability(capStr, "write") {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "This API key is read-only. Use a key with write capability to change data."})
			return
		}
		c.Next()
	}
}
