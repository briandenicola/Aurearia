package handlers

import (
	"errors"
	"net/http"

	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
)

// ExternalToolsHandler handles collection tool operations for external API key authenticated clients.
// All routes are protected by ExternalToolServerEnabled gate, API key auth, and capability middleware.
type ExternalToolsHandler struct {
	collectionSvc *services.CollectionToolsService
}

func NewExternalToolsHandler(
	collectionSvc *services.CollectionToolsService,
) *ExternalToolsHandler {
	return &ExternalToolsHandler{
		collectionSvc: collectionSvc,
	}
}

// SearchMyCollection godoc
// @Summary      Search user's collection (external)
// @Description  Search the authenticated API key owner's collection by query filters. Requires 'read' capability.
// @Tags         External Tools
// @Accept       json
// @Produce      json
// @Param        X-API-Key header string true "API key (e.g., ak_...)"
// @Param        body body SearchMyCollectionRequest true "Search parameters"
// @Success      200 {object} map[string]interface{} "coins: array of coin summaries"
// @Failure      400 {object} map[string]interface{}
// @Failure      401 {object} map[string]interface{}
// @Failure      403 {object} map[string]interface{}
// @Failure      503 {object} map[string]interface{}
// @Security     ApiKeyAuth
// @Router       /v1/tools/search_my_collection [post]
func (h *ExternalToolsHandler) SearchMyCollection(c *gin.Context) {
	userID, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userIDUint, ok := userID.(uint)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req SearchMyCollectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	coins, err := h.collectionSvc.SearchMyCollection(userIDUint, req.Query, req.Limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "An error occurred"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"coins": coins})
}

// GetCoin godoc
// @Summary      Get a single coin by ID (external)
// @Description  Retrieve a coin from the authenticated API key owner's collection. Requires 'read' capability.
// @Tags         External Tools
// @Accept       json
// @Produce      json
// @Param        X-API-Key header string true "API key (e.g., ak_...)"
// @Param        body body GetCoinRequest true "Coin ID"
// @Success      200 {object} map[string]interface{} "coin: coin summary"
// @Failure      400 {object} map[string]interface{}
// @Failure      401 {object} map[string]interface{}
// @Failure      403 {object} map[string]interface{}
// @Failure      404 {object} map[string]interface{}
// @Failure      503 {object} map[string]interface{}
// @Security     ApiKeyAuth
// @Router       /v1/tools/get_coin [post]
func (h *ExternalToolsHandler) GetCoin(c *gin.Context) {
	userID, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userIDUint, ok := userID.(uint)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req GetCoinRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	coin, err := h.collectionSvc.GetCoin(userIDUint, req.CoinID)
	if err != nil {
		if errors.Is(err, services.ErrCoinNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Coin not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "An error occurred"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"coin": coin})
}

// CollectionSummary godoc
// @Summary      Get collection aggregate summary (external)
// @Description  Retrieve aggregate statistics for the authenticated API key owner's collection. Requires 'read' capability.
// @Tags         External Tools
// @Accept       json
// @Produce      json
// @Param        X-API-Key header string true "API key (e.g., ak_...)"
// @Success      200 {object} map[string]interface{} "summary: aggregate summary"
// @Failure      401 {object} map[string]interface{}
// @Failure      403 {object} map[string]interface{}
// @Failure      503 {object} map[string]interface{}
// @Security     ApiKeyAuth
// @Router       /v1/tools/collection_summary [post]
func (h *ExternalToolsHandler) CollectionSummary(c *gin.Context) {
	userID, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userIDUint, ok := userID.(uint)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	summary, err := h.collectionSvc.CollectionSummary(userIDUint)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "An error occurred"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"summary": summary})
}

// TopCoinsByValue godoc
// @Summary      Get top coins by current value (external)
// @Description  Retrieve the top coins by current value from the authenticated API key owner's collection. Requires 'read' capability.
// @Tags         External Tools
// @Accept       json
// @Produce      json
// @Param        X-API-Key header string true "API key (e.g., ak_...)"
// @Param        body body TopCoinsByValueRequest false "Limit (default 3, max 10)"
// @Success      200 {object} map[string]interface{} "coins: array of coin summaries"
// @Failure      400 {object} map[string]interface{}
// @Failure      401 {object} map[string]interface{}
// @Failure      403 {object} map[string]interface{}
// @Failure      503 {object} map[string]interface{}
// @Security     ApiKeyAuth
// @Router       /v1/tools/top_coins_by_value [post]
func (h *ExternalToolsHandler) TopCoinsByValue(c *gin.Context) {
	userID, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userIDUint, ok := userID.(uint)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req TopCoinsByValueRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	coins, err := h.collectionSvc.TopCoinsByValue(userIDUint, req.Limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "An error occurred"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"coins": coins})
}

// ProposeUpdate godoc
// @Summary      Create an update proposal (external)
// @Description  Create a proposal to update allowlisted fields on a coin. Requires 'write' capability. Returns proposal preview with token; no coin write occurs yet.
// @Tags         External Tools
// @Accept       json
// @Produce      json
// @Param        X-API-Key header string true "API key (e.g., ak_...)"
// @Param        body body ProposeUpdateRequest true "Coin ID and changes"
// @Success      200 {object} map[string]interface{} "proposal: proposal preview with token"
// @Failure      400 {object} map[string]interface{}
// @Failure      401 {object} map[string]interface{}
// @Failure      403 {object} map[string]interface{}
// @Failure      404 {object} map[string]interface{}
// @Failure      503 {object} map[string]interface{}
// @Security     ApiKeyAuth
// @Router       /v1/tools/propose_update [post]
func (h *ExternalToolsHandler) ProposeUpdate(c *gin.Context) {
	userID, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userIDUint, ok := userID.(uint)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req ProposeUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	proposal, err := h.collectionSvc.ProposeUpdate(userIDUint, req.CoinID, req.Changes)
	if err != nil {
		if errors.Is(err, services.ErrCoinNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Coin not found"})
			return
		}
		if errors.Is(err, services.ErrInvalidFieldChanges) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "An error occurred"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"proposal": proposal})
}

// CommitUpdate godoc
// @Summary      Commit an update proposal (external)
// @Description  Commit a previously created proposal with explicit confirmation. Requires 'write' capability and valid proposal token. Journals with source 'external_tool_server'.
// @Tags         External Tools
// @Accept       json
// @Produce      json
// @Param        X-API-Key header string true "API key (e.g., ak_...)"
// @Param        body body CommitUpdateRequest true "Proposal ID, token, and confirmation"
// @Success      200 {object} map[string]interface{} "result: commit result"
// @Failure      400 {object} map[string]interface{}
// @Failure      401 {object} map[string]interface{}
// @Failure      403 {object} map[string]interface{}
// @Failure      404 {object} map[string]interface{}
// @Failure      409 {object} map[string]interface{}
// @Failure      503 {object} map[string]interface{}
// @Security     ApiKeyAuth
// @Router       /v1/tools/commit_update [post]
func (h *ExternalToolsHandler) CommitUpdate(c *gin.Context) {
	userID, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userIDUint, ok := userID.(uint)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req CommitUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// Collect API key metadata from context with defensive checks
	apiKeyID, exists := c.Get("apiKeyId")
	if !exists {
		c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient capability"})
		return
	}
	apiKeyIDUint, ok := apiKeyID.(uint)
	if !ok {
		c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient capability"})
		return
	}

	apiKeyName, exists := c.Get("apiKeyName")
	if !exists {
		c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient capability"})
		return
	}
	apiKeyNameStr, ok := apiKeyName.(string)
	if !ok {
		c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient capability"})
		return
	}

	apiKeyCap, exists := c.Get("apiKeyCapabilities")
	if !exists {
		c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient capability"})
		return
	}
	apiKeyCapStr, ok := apiKeyCap.(string)
	if !ok {
		c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient capability"})
		return
	}

	// Call service with external source and originating actor metadata
	result, err := h.collectionSvc.CommitUpdateExternal(
		userIDUint,
		req.ProposalID,
		req.Token,
		req.Confirm,
		apiKeyIDUint,
		apiKeyNameStr,
		apiKeyCapStr,
	)
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "An error occurred"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"result": result})
}
