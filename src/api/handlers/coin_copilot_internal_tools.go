package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
)

type CoinCopilotInternalToolsHandler struct {
	collectionSvc *services.CollectionToolsService
	copilotSvc    *services.CoinCopilotService
	logger        *services.Logger
}

func NewCoinCopilotInternalToolsHandler(collectionSvc *services.CollectionToolsService, copilotSvc *services.CoinCopilotService, logger *services.Logger) *CoinCopilotInternalToolsHandler {
	return &CoinCopilotInternalToolsHandler{collectionSvc: collectionSvc, copilotSvc: copilotSvc, logger: logger}
}

type copilotToolCallRequest struct {
	ToolCallID string `json:"tool_call_id"`
	Query      string `json:"query,omitempty"`
	Limit      *int   `json:"limit,omitempty"`
	CoinID     uint   `json:"coin_id,omitempty"`
}

func (h *CoinCopilotInternalToolsHandler) execute(c *gin.Context, tool string, operation func(uint, copilotToolCallRequest) (any, error)) {
	claimsValue, ok := c.Get("copilotClaims")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	claims, ok := claimsValue.(*services.CopilotExecutionClaims)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 64*1024)
	var request copilotToolCallRequest
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil || request.ToolCallID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	if err := h.copilotSvc.AuthorizeToolCall(claims, request.ToolCallID, tool); err != nil {
		status := http.StatusConflict
		if errors.Is(err, services.ErrCopilotInvalidRequest) {
			status = http.StatusBadRequest
		}
		c.JSON(status, gin.H{"error": "Tool call rejected"})
		return
	}
	result, err := operation(claims.UserID, request)
	if err != nil {
		h.copilotSvc.FinishToolCall(claims.ExecutionID, request.ToolCallID, false)
		if errors.Is(err, services.ErrCopilotInvalidRequest) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
			return
		}
		if errors.Is(err, services.ErrCoinNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Coin not found"})
			return
		}
		if h.logger != nil {
			h.logger.Error("coin-copilot", "tool failed run=%s execution=%s tool=%s: %v", claims.RunID, claims.ExecutionID, tool, err)
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Tool execution failed"})
		return
	}
	h.copilotSvc.FinishToolCall(claims.ExecutionID, request.ToolCallID, true)
	c.JSON(http.StatusOK, result)
}

func (h *CoinCopilotInternalToolsHandler) SearchMyCollection(c *gin.Context) {
	h.execute(c, "search_my_collection", func(userID uint, request copilotToolCallRequest) (any, error) {
		if request.Query == "" || len(request.Query) > 4000 {
			return nil, services.ErrCopilotInvalidRequest
		}
		coins, err := h.collectionSvc.SearchMyCollection(userID, request.Query, request.Limit)
		return gin.H{"coins": coins}, err
	})
}

func (h *CoinCopilotInternalToolsHandler) GetCoin(c *gin.Context) {
	h.execute(c, "get_coin", func(userID uint, request copilotToolCallRequest) (any, error) {
		if request.CoinID == 0 {
			return nil, services.ErrCopilotInvalidRequest
		}
		coin, err := h.collectionSvc.GetCoin(userID, request.CoinID)
		return gin.H{"coin": coin}, err
	})
}

func (h *CoinCopilotInternalToolsHandler) CollectionSummary(c *gin.Context) {
	h.execute(c, "collection_summary", func(userID uint, request copilotToolCallRequest) (any, error) {
		summary, err := h.collectionSvc.CollectionSummary(userID)
		return gin.H{"summary": summary}, err
	})
}

func (h *CoinCopilotInternalToolsHandler) TopCoinsByValue(c *gin.Context) {
	h.execute(c, "top_coins_by_value", func(userID uint, request copilotToolCallRequest) (any, error) {
		coins, err := h.collectionSvc.TopCoinsByValue(userID, request.Limit)
		return gin.H{"coins": coins}, err
	})
}
