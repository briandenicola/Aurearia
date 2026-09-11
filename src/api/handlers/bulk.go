package handlers

import (
	"errors"
	"net/http"

	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
)

// BulkActionRequest is the request body for bulk coin actions.
type BulkActionRequest struct {
	CoinIDs           []uint `json:"coinIds" binding:"required"`
	Action            string `json:"action" binding:"required"`
	TagID             *uint  `json:"tagId"`
	SetID             *uint  `json:"setId"`
	StorageLocationID *uint  `json:"storageLocationId"`
}

// BulkHandler handles bulk operations on coins.
type BulkHandler struct {
	coinRepo            *repository.CoinRepository
	tagRepo             *repository.TagRepository
	storageLocationRepo *repository.StorageLocationRepository
	setRepo             *repository.SetRepository
	coinService         *services.CoinService
}

func (h *BulkHandler) WithCoinService(coinService *services.CoinService) *BulkHandler {
	h.coinService = coinService
	return h
}

// NewBulkHandler creates a new BulkHandler.
func NewBulkHandler(coinRepo *repository.CoinRepository, tagRepo *repository.TagRepository, storageLocationRepo *repository.StorageLocationRepository, setRepo *repository.SetRepository) *BulkHandler {
	return &BulkHandler{coinRepo: coinRepo, tagRepo: tagRepo, storageLocationRepo: storageLocationRepo, setRepo: setRepo}
}

// BulkAction performs a bulk operation on the selected coins.
//
//	@Summary		Bulk coin action
//	@Description	Performs a bulk action (tag, set, delete, sell, export, assign-location) on selected coins.
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

	var req BulkActionRequest
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
		var affected int64
		var err error
		if h.coinService != nil {
			affected, err = h.coinService.BulkDeleteCoins(req.CoinIDs, userID)
		} else {
			affected, err = h.coinRepo.BulkDelete(req.CoinIDs, userID)
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete coins"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Coins deleted", "affected": affected})

	case "sell":
		var affected int64
		var err error
		if h.coinService != nil {
			affected, err = h.coinService.BulkMarkSold(req.CoinIDs, userID)
		} else {
			affected, err = h.coinRepo.BulkMarkSold(req.CoinIDs, userID)
		}
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

	case "set":
		if req.SetID == nil || *req.SetID == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "setId is required for set action"})
			return
		}
		affected, err := h.setRepo.BulkAddCoinsToSet(req.CoinIDs, *req.SetID, userID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Set applied", "affected": affected})

	case "export":
		coins, err := h.coinRepo.GetByIDs(req.CoinIDs, userID)
		if err != nil || len(coins) == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "No matching coins found"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"coins": coins, "total": len(coins)})

	case "assign-location":
		if h.coinService == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Storage assignment service unavailable"})
			return
		}
		affected, err := h.coinService.BulkAssignLocation(req.CoinIDs, req.StorageLocationID, userID)
		if err != nil {
			if errors.Is(err, services.ErrStorageLocationNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": "Storage location not found"})
				return
			}
			if errors.Is(err, services.ErrStorageSlotRequired) {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Tray assignments require choosing a slot on each coin.", "code": "validation_error", "field": "storageLocationId"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to assign storage location"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Storage location assigned", "affected": affected})

	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid action. Must be: tag, set, delete, sell, export, or assign-location"})
	}
}
