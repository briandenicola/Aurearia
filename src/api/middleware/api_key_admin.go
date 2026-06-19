package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// RejectAPIKeyAuth denies requests authenticated with X-API-Key.
func RejectAPIKeyAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, exists := c.Get("apiKeyId"); exists {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "JWT authentication required"})
			return
		}
		c.Next()
	}
}
