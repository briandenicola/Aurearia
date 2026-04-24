package middleware

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// ApiKeyAuthenticator abstracts the DB operations needed for API key auth.
type ApiKeyAuthenticator interface {
	FindActiveByHash(keyHash string) (*models.ApiKey, error)
	FindUserByID(id uint) (*models.User, error)
	UpdateLastUsed(apiKey *models.ApiKey, t time.Time)
}

func AuthRequired(jwtSecret string, apiKeyAuth ApiKeyAuthenticator) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Try API key auth first
		apiKey := c.GetHeader("X-API-Key")
		if apiKey != "" {
			if authenticateApiKey(c, apiKey, apiKeyAuth) {
				c.Next()
				return
			}
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid API key"})
			return
		}

		// Fall back to JWT bearer auth (header or query param for image proxy)
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			if tokenParam := c.Query("token"); tokenParam != "" {
				authHeader = "Bearer " + tokenParam
			}
		}
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Bearer token required"})
			return
		}

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(jwtSecret), nil
		})

		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims"})
			return
		}

		userID, ok := claims["userId"].(float64)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID in token"})
			return
		}

		role, _ := claims["role"].(string)

		c.Set("userId", uint(userID))
		c.Set("userRole", role)
		c.Next()
	}
}

func authenticateApiKey(c *gin.Context, plainKey string, auth ApiKeyAuthenticator) bool {
	hash := sha256.Sum256([]byte(plainKey))
	keyHash := hex.EncodeToString(hash[:])

	apiKey, err := auth.FindActiveByHash(keyHash)
	if err != nil {
		return false
	}

	// Look up the user to get their role
	user, err := auth.FindUserByID(apiKey.UserID)
	if err != nil {
		return false
	}

	// Update last used timestamp
	auth.UpdateLastUsed(apiKey, time.Now())

	c.Set("userId", apiKey.UserID)
	c.Set("userRole", string(user.Role))
	return true
}
