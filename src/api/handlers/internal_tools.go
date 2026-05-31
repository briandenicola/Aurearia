package handlers

import (
	"errors"
	"net/http"

	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
)

// InternalToolsHandler handles collection tool operations for the internal Python agent.
// All routes are protected by InternalTokenRequired middleware.
type InternalToolsHandler struct {
	collectionSvc *services.CollectionToolsService
	logger        *services.Logger
}

func NewInternalToolsHandler(
	collectionSvc *services.CollectionToolsService,
	logger *services.Logger,
) *InternalToolsHandler {
	return &InternalToolsHandler{
		collectionSvc: collectionSvc,
		logger:        logger,
	}
}

// SearchMyCollectionRequest represents the request body for search_my_collection
type SearchMyCollectionRequest struct {
	Query string `json:"query" binding:"required"`
	Limit *int   `json:"limit"`
}

// GetCoinRequest represents the request body for get_coin
type GetCoinRequest struct {
	CoinID uint `json:"coin_id" binding:"required"`
}

// TopCoinsByValueRequest represents the request body for top_coins_by_value
type TopCoinsByValueRequest struct {
	Limit *int `json:"limit"`
}

// ProposeUpdateRequest represents the request body for propose_update
type ProposeUpdateRequest struct {
	CoinID  uint           `json:"coin_id" binding:"required"`
	Changes map[string]any `json:"changes" binding:"required"`
}

// CommitUpdateRequest represents the request body for commit_update
type CommitUpdateRequest struct {
	ProposalID string `json:"proposal_id" binding:"required"`
	Token      string `json:"token" binding:"required"`
	Confirm    bool   `json:"confirm" binding:"required"`
}

// SearchMyCollection godoc
// @Summary      Search user's collection
// @Description  Search the authenticated user's collection by query filters
// @Tags         Internal
// @Accept       json
// @Produce      json
// @Param        Authorization header string true "Bearer {internal_token}"
// @Param        body body SearchMyCollectionRequest true "Search parameters"
// @Success      200 {object} map[string]interface{} "coins: array of coin summaries"
// @Failure      400 {object} map[string]interface{}
// @Failure      401 {object} map[string]interface{}
// @Failure      500 {object} map[string]interface{}
// @Router       /internal/tools/search_my_collection [post]
func (h *InternalToolsHandler) SearchMyCollection(c *gin.Context) {
	userID, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req SearchMyCollectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	coins, err := h.collectionSvc.SearchMyCollection(userID.(uint), req.Query, req.Limit)
	if err != nil {
		h.logger.Error("internal-tools", "SearchMyCollection error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "An error occurred"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"coins": coins})
}

// GetCoin godoc
// @Summary      Get a single coin by ID
// @Description  Retrieve a coin from the authenticated user's collection
// @Tags         Internal
// @Accept       json
// @Produce      json
// @Param        Authorization header string true "Bearer {internal_token}"
// @Param        body body GetCoinRequest true "Coin ID"
// @Success      200 {object} map[string]interface{} "coin: coin summary"
// @Failure      400 {object} map[string]interface{}
// @Failure      401 {object} map[string]interface{}
// @Failure      404 {object} map[string]interface{}
// @Failure      500 {object} map[string]interface{}
// @Router       /internal/tools/get_coin [post]
func (h *InternalToolsHandler) GetCoin(c *gin.Context) {
	userID, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req GetCoinRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	coin, err := h.collectionSvc.GetCoin(userID.(uint), req.CoinID)
	if err != nil {
		if errors.Is(err, services.ErrCoinNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Coin not found"})
			return
		}
		h.logger.Error("internal-tools", "GetCoin error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "An error occurred"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"coin": coin})
}

// CollectionSummary godoc
// @Summary      Get collection aggregate summary
// @Description  Retrieve aggregate statistics for the authenticated user's collection
// @Tags         Internal
// @Accept       json
// @Produce      json
// @Param        Authorization header string true "Bearer {internal_token}"
// @Success      200 {object} map[string]interface{} "summary: aggregate summary"
// @Failure      401 {object} map[string]interface{}
// @Failure      500 {object} map[string]interface{}
// @Router       /internal/tools/collection_summary [post]
func (h *InternalToolsHandler) CollectionSummary(c *gin.Context) {
	userID, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	summary, err := h.collectionSvc.CollectionSummary(userID.(uint))
	if err != nil {
		h.logger.Error("internal-tools", "CollectionSummary error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "An error occurred"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"summary": summary})
}

// TopCoinsByValue godoc
// @Summary      Get top coins by current value
// @Description  Retrieve the top coins by current value from the authenticated user's collection
// @Tags         Internal
// @Accept       json
// @Produce      json
// @Param        Authorization header string true "Bearer {internal_token}"
// @Param        body body TopCoinsByValueRequest false "Limit (default 3, max 10)"
// @Success      200 {object} map[string]interface{} "coins: array of coin summaries"
// @Failure      400 {object} map[string]interface{}
// @Failure      401 {object} map[string]interface{}
// @Failure      500 {object} map[string]interface{}
// @Router       /internal/tools/top_coins_by_value [post]
func (h *InternalToolsHandler) TopCoinsByValue(c *gin.Context) {
	userID, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req TopCoinsByValueRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	coins, err := h.collectionSvc.TopCoinsByValue(userID.(uint), req.Limit)
	if err != nil {
		h.logger.Error("internal-tools", "TopCoinsByValue error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "An error occurred"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"coins": coins})
}

// ProposeUpdate godoc
// @Summary      Create an update proposal
// @Description  Create a proposal to update allowlisted fields on a coin
// @Tags         Internal
// @Accept       json
// @Produce      json
// @Param        Authorization header string true "Bearer {internal_token}"
// @Param        body body ProposeUpdateRequest true "Coin ID and changes"
// @Success      200 {object} map[string]interface{} "proposal: proposal preview with token"
// @Failure      400 {object} map[string]interface{}
// @Failure      401 {object} map[string]interface{}
// @Failure      404 {object} map[string]interface{}
// @Failure      500 {object} map[string]interface{}
// @Router       /internal/tools/propose_update [post]
func (h *InternalToolsHandler) ProposeUpdate(c *gin.Context) {
	userID, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req ProposeUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	proposal, err := h.collectionSvc.ProposeUpdate(userID.(uint), req.CoinID, req.Changes)
	if err != nil {
		if errors.Is(err, services.ErrCoinNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Coin not found"})
			return
		}
		if errors.Is(err, services.ErrInvalidFieldChanges) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		h.logger.Error("internal-tools", "ProposeUpdate error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "An error occurred"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"proposal": proposal})
}

// CommitUpdate godoc
// @Summary      Commit an update proposal
// @Description  Commit a previously created proposal with explicit confirmation
// @Tags         Internal
// @Accept       json
// @Produce      json
// @Param        Authorization header string true "Bearer {internal_token}"
// @Param        body body CommitUpdateRequest true "Proposal ID, token, and confirmation"
// @Success      200 {object} map[string]interface{} "result: commit result"
// @Failure      400 {object} map[string]interface{}
// @Failure      401 {object} map[string]interface{}
// @Failure      404 {object} map[string]interface{}
// @Failure      500 {object} map[string]interface{}
// @Router       /internal/tools/commit_update [post]
func (h *InternalToolsHandler) CommitUpdate(c *gin.Context) {
	userID, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req CommitUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	result, err := h.collectionSvc.CommitUpdate(userID.(uint), req.ProposalID, req.Token, req.Confirm)
	if err != nil {
		if errors.Is(err, services.ErrProposalConfirmationReq) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Confirmation required"})
			return
		}
		if errors.Is(err, services.ErrProposalStateConflict) {
			c.JSON(http.StatusConflict, gin.H{"error": "Proposal is not in pending state or has expired"})
			return
		}
		if errors.Is(err, services.ErrProposalTokenInvalid) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid proposal token"})
			return
		}
		h.logger.Error("internal-tools", "CommitUpdate error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "An error occurred"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"result": result})
}
