package middleware

import (
	"net/http"
	"strings"

	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
)

// InternalTokenRequired validates the internal service token and sets userId in context.
// Returns 401 with generic message if token is missing, malformed, or expired.
func InternalTokenRequired(tokenSvc *services.InternalTokenService) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization required"})
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization format"})
			return
		}

		userID, err := tokenSvc.Verify(tokenString)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			return
		}

		c.Set("userId", userID)
		c.Next()
	}
}

// InternalJobTokenRequired validates a job-scoped internal token (minted via
// InternalTokenService.MintForJob) for the deep-identification provider-tool
// endpoints (contracts/agent-internal-contract.md §7). It sets both userId
// and deepJobId in context so handlers can enforce per-job call budgets and
// reject a token that does not carry a valid job binding ("foreign-job"
// calls, T054).
func InternalJobTokenRequired(tokenSvc *services.InternalTokenService) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization required"})
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization format"})
			return
		}

		userID, jobID, err := tokenSvc.VerifyForJob(tokenString)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			return
		}

		c.Set("userId", userID)
		c.Set("deepJobId", jobID)
		c.Next()
	}
}
