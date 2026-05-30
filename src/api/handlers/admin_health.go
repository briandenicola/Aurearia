package handlers

import (
	"net/http"

	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
)

// AdminHealthHandler serves aggregate health metrics for admin users.
type AdminHealthHandler struct {
	svc    *services.HealthService
	logger *services.Logger
}

// NewAdminHealthHandler creates a new AdminHealthHandler.
func NewAdminHealthHandler(svc *services.HealthService, logger *services.Logger) *AdminHealthHandler {
	return &AdminHealthHandler{svc: svc, logger: logger}
}

// Summary returns aggregate health metrics across users.
func (h *AdminHealthHandler) Summary(c *gin.Context) {
	summary, err := h.svc.GetAdminHealthSummary()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch admin health summary"})
		return
	}
	c.JSON(http.StatusOK, summary)
}
