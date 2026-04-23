package handlers

import (
	"net/http"

	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/gin-gonic/gin"
)

// BulkHandler handles bulk operations on coins.
type BulkHandler struct {
	coinRepo *repository.CoinRepository
	tagRepo  *repository.TagRepository
}

// NewBulkHandler creates a new BulkHandler.
func NewBulkHandler(coinRepo *repository.CoinRepository, tagRepo *repository.TagRepository) *BulkHandler {
	return &BulkHandler{coinRepo: coinRepo, tagRepo: tagRepo}
}

// BulkAction performs a bulk operation on the selected coins.
//
//	@Summary		Bulk coin action
//	@Description	Performs a bulk action (tag, delete, sell, export) on selected coins.
//	@Tags			Coins
//	@Accept			json
//	@Produce		json
//	@Param			body	body		BulkActionRequest	true	"Bulk action request"
//	@Success		200		{object}	map[string]interface{}
//	@Failure		400		{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/coins/bulk [post]
func (h *BulkHandler) BulkAction(c *gin.Context) {
	userID := c.GetUint("userId")

	var req struct {
		CoinIDs []uint `json:"coinIds" binding:"required"`
		Action  string `json:"action" binding:"required"`
		TagID   *uint  `json:"tagId"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "coinIds and action are required"})
		return
	}

	if len(req.CoinIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No coins selected"})
		return
	}
	if len(req.CoinIDs) > 200 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Maximum 200 coins per bulk operation"})
		return
	}

	switch req.Action {
	case "delete":
		affected, err := h.coinRepo.BulkDelete(req.CoinIDs, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete coins"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Coins deleted", "affected": affected})

	case "sell":
		affected, err := h.coinRepo.BulkMarkSold(req.CoinIDs, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to mark coins as sold"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Coins marked as sold", "affected": affected})

	case "tag":
		if req.TagID == nil || *req.TagID == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "tagId is required for tag action"})
			return
		}
		affected, err := h.tagRepo.BulkAttachToCoin(req.CoinIDs, *req.TagID, userID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Tag or coins not found"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Tag applied", "affected": affected})

	case "export":
		coins, err := h.coinRepo.GetByIDs(req.CoinIDs, userID)
		if err != nil || len(coins) == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "No matching coins found"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"coins": coins, "total": len(coins)})

	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid action. Must be: tag, delete, sell, or export"})
	}
}
