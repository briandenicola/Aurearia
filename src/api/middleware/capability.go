package middleware

import (
	"net/http"
	"strings"

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

		switch scope {
		case "read":
			// Read requires either "read" or "write" (write implies read)
			if !strings.Contains(capStr, "read") && !strings.Contains(capStr, "write") {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Insufficient capability"})
				return
			}
		case "write":
			// Write requires explicit "write" capability
			if !strings.Contains(capStr, "write") {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Insufficient capability"})
				return
			}
		default:
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Insufficient capability"})
			return
		}

		c.Next()
	}
}
