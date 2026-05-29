package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

const (
	DefaultRequestBodyLimitBytes int64 = 30 << 20
	DefaultMultipartMemoryBytes  int64 = 8 << 20
)

func RequestBodyLimit(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if maxBytes <= 0 {
			c.Next()
			return
		}

		if c.Request.ContentLength > maxBytes {
			c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, gin.H{"error": "Request body too large"})
			return
		}

		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		c.Next()
	}
}
