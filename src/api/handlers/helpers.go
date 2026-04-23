package handlers

import (
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// respondError sends a consistent JSON error response and logs server-side details.
func respondError(c *gin.Context, status int, clientMsg string, err error) {
	if err != nil {
		log.Printf("[%s %s] %s: %v", c.Request.Method, c.Request.URL.Path, clientMsg, err)
	}
	c.JSON(status, gin.H{"error": clientMsg})
}

// parseID extracts a uint path parameter and returns 400 on invalid or zero values.
func parseID(c *gin.Context, param string) (uint, bool) {
	id, err := strconv.ParseUint(c.Param(param), 10, 64)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return 0, false
	}
	return uint(id), true
}
