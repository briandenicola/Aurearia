package handlers

import (
	"net/http"
	"strconv"

	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// CoinOfDayHandler exposes user-facing endpoints for the Coin of the Day feature.
type CoinOfDayHandler struct {
	repo   *repository.FeaturedCoinRepository
	logger *services.Logger
}

// NewCoinOfDayHandler creates a new CoinOfDayHandler.
func NewCoinOfDayHandler(repo *repository.FeaturedCoinRepository, logger *services.Logger) *CoinOfDayHandler {
	return &CoinOfDayHandler{repo: repo, logger: logger}
}

// Get returns a single featured-coin record for the authenticated user.
//
//	@Summary		Get a featured coin
//	@Description	Returns one Coin-of-the-Day record (with the populated coin) for the authenticated user.
//	@Tags			FeaturedCoin
//	@Produce		json
//	@Param			id	path		int	true	"FeaturedCoin ID"
//	@Success		200	{object}	models.FeaturedCoin
//	@Failure		401	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/featured-coins/{id} [get]
func (h *CoinOfDayHandler) Get(c *gin.Context) {
	userID := c.GetUint("userId")
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid featured-coin ID"})
		return
	}

	fc, err := h.repo.FindByIDForUser(uint(id), userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Featured coin not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch featured coin"})
		return
	}
	c.JSON(http.StatusOK, fc)
}

// Latest returns the most recent featured coin for the authenticated user.
//
//	@Summary		Get latest featured coin
//	@Description	Returns the most recent Coin-of-the-Day record for the authenticated user.
//	@Tags			FeaturedCoin
//	@Produce		json
//	@Success		200	{object}	models.FeaturedCoin
//	@Failure		401	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/featured-coins/latest [get]
func (h *CoinOfDayHandler) Latest(c *gin.Context) {
	userID := c.GetUint("userId")
	fc, err := h.repo.GetLatestForUser(userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "No featured coin yet"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch featured coin"})
		return
	}
	c.JSON(http.StatusOK, fc)
}
