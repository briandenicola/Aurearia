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
