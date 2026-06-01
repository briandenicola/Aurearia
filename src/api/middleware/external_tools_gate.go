package middleware

import (
	"net/http"

	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
)

// ExternalToolServerEnabled returns middleware that gates the external tool server
// based on the ExternalToolServerEnabled admin setting. Returns 503 when disabled.
func ExternalToolServerEnabled(settingsSvc *services.SettingsService) gin.HandlerFunc {
	return func(c *gin.Context) {
		enabled := settingsSvc.GetSetting(services.SettingExternalToolServerEnabled)
		if enabled != "true" {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "External tool server is disabled"})
			return
		}
		c.Next()
	}
}
