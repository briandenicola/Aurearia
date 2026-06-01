package handlers

import (
	"net/http"
	"strconv"

	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
)

// HealthHandler serves scorecard endpoints for authenticated users.
type HealthHandler struct {
	svc    *services.HealthService
	logger *services.Logger
}

// NewHealthHandler creates a new HealthHandler.
func NewHealthHandler(svc *services.HealthService, logger *services.Logger) *HealthHandler {
	return &HealthHandler{svc: svc, logger: logger}
}

// CollectionSummary returns collection-level health summary data.
func (h *HealthHandler) CollectionSummary(c *gin.Context) {
	userID := c.GetUint("userId")
	summary, err := h.svc.GetCollectionHealthSummary(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch health summary"})
		return
	}
	c.JSON(http.StatusOK, summary)
}

// ListCoinHealth returns per-coin health scores and checklist items.
func (h *HealthHandler) ListCoinHealth(c *gin.Context) {
	userID := c.GetUint("userId")

	scope := c.DefaultQuery("scope", "all")
	if scope != "all" && scope != "needs_attention" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "scope must be one of: all, needs_attention"})
		return
	}

	page := 1
	if pageStr := c.DefaultQuery("page", "1"); pageStr != "" {
		p, err := strconv.Atoi(pageStr)
		if err != nil || p < 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "page must be an integer >= 1"})
			return
		}
		page = p
	}

	limit := 25
	if limitStr := c.DefaultQuery("limit", "25"); limitStr != "" {
		l, err := strconv.Atoi(limitStr)
		if err != nil || l < 1 || l > 100 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "limit must be an integer between 1 and 100"})
			return
		}
		limit = l
	}

	list, err := h.svc.ListCoinHealth(userID, page, limit, scope)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch coin health list"})
		return
	}
	c.JSON(http.StatusOK, list)
}

// GetCoinHealth godoc
//
//	@Summary		Get metadata health for a single coin
//	@Description	Returns the computed metadata health score, grade, dimension scores, and checklist of missing items for a specific coin owned by the authenticated user
//	@Tags			Health
//	@Security		BearerAuth
//	@Produce		json
//	@Param			id	path		int	true	"Coin ID"
//	@Success		200	{object}	services.CoinHealthItem
//	@Failure		400	{object}	map[string]string
//	@Failure		404	{object}	map[string]string
//	@Failure		500	{object}	map[string]string
//	@Router			/coins/{id}/health [get]
func (h *HealthHandler) GetCoinHealth(c *gin.Context) {
	userID := c.GetUint("userId")

	coinIDStr := c.Param("id")
	coinID, err := strconv.ParseUint(coinIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid coin ID"})
		return
	}

	health, err := h.svc.GetCoinHealth(uint(coinID), userID)
	if err != nil {
		h.logger.Warn("health", "Failed to fetch coin health for coin %d, user %d: %v", coinID, userID, err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Coin not found or not in active collection"})
		return
	}

	c.JSON(http.StatusOK, health)
}
