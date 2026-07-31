package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
)

type CoinRecommendationHandler struct {
	service *services.CoinRecommendationService
}

func NewCoinRecommendationHandler(service *services.CoinRecommendationService) *CoinRecommendationHandler {
	return &CoinRecommendationHandler{service: service}
}

// List returns generated recommendations for a coin owned by the authenticated user.
//
//	@Summary		List coin recommendations
//	@Description	Returns suggested missing set/tag assignments with confidence and reasons
//	@Tags			coins
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		int	true	"Coin ID"
//	@Success		200	{object}	object{recommendations=[]services.CoinRecommendationItem}
//	@Failure		400	{object}	object{error=string}
//	@Failure		404	{object}	object{error=string}
//	@Failure		500	{object}	object{error=string}
//	@Router			/coins/{id}/recommendations [get]
func (h *CoinRecommendationHandler) List(c *gin.Context) {
	userID := c.GetUint("userId")
	coinID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid coin ID"})
		return
	}

	items, err := h.service.ListForCoin(uint(coinID), userID)
	if err != nil {
		if repository.IsRecordNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Coin not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load recommendations"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"recommendations": items})
}

// Accept applies a recommendation for the authenticated user's coin.
//
//	@Summary		Accept recommendation
//	@Description	Applies recommended set/tag assignment and marks recommendation accepted
//	@Tags			coins
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id				path		int	true	"Coin ID"
//	@Param			recommendationId	path		int	true	"Recommendation ID"
//	@Success		200				{object}	object{message=string}
//	@Failure		400				{object}	object{error=string}
//	@Failure		404				{object}	object{error=string}
//	@Failure		500				{object}	object{error=string}
//	@Router			/coins/{id}/recommendations/{recommendationId}/accept [post]
func (h *CoinRecommendationHandler) Accept(c *gin.Context) {
	h.applyDecision(c, true)
}

// Reject marks a recommendation as rejected.
//
//	@Summary		Reject recommendation
//	@Description	Marks recommendation rejected and records feedback
//	@Tags			coins
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id				path		int	true	"Coin ID"
//	@Param			recommendationId	path		int	true	"Recommendation ID"
//	@Success		200				{object}	object{message=string}
//	@Failure		400				{object}	object{error=string}
//	@Failure		404				{object}	object{error=string}
//	@Failure		500				{object}	object{error=string}
//	@Router			/coins/{id}/recommendations/{recommendationId}/reject [post]
func (h *CoinRecommendationHandler) Reject(c *gin.Context) {
	h.applyDecision(c, false)
}

func (h *CoinRecommendationHandler) applyDecision(c *gin.Context, accept bool) {
	userID := c.GetUint("userId")
	coinID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid coin ID"})
		return
	}
	recID, err := strconv.ParseUint(c.Param("recommendationId"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid recommendation ID"})
		return
	}

	if accept {
		err = h.service.Accept(uint(coinID), uint(recID), userID)
	} else {
		err = h.service.Reject(uint(coinID), uint(recID), userID)
	}
	if err != nil {
		if errors.Is(err, services.ErrRecommendationNotFound) || repository.IsRecordNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Recommendation not found"})
			return
		}
		if errors.Is(err, services.ErrRecommendationTarget) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Recommendation target is invalid"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update recommendation"})
		return
	}

	if accept {
		c.JSON(http.StatusOK, gin.H{"message": "Recommendation accepted"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Recommendation rejected"})
}
