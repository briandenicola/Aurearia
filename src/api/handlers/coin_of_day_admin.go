package handlers

import (
	"net/http"

	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
)

// CoinOfDayAdminHandler exposes admin endpoints for the Coin of the Day scheduler.
type CoinOfDayAdminHandler struct {
	scheduler *services.CoinOfDayScheduler
	logger    *services.Logger
}

// NewCoinOfDayAdminHandler creates a new CoinOfDayAdminHandler.
func NewCoinOfDayAdminHandler(scheduler *services.CoinOfDayScheduler, logger *services.Logger) *CoinOfDayAdminHandler {
	return &CoinOfDayAdminHandler{scheduler: scheduler, logger: logger}
}

// TriggerRun manually triggers a Coin of the Day pick for all opted-in users.
//
//	@Summary		Trigger manual coin-of-the-day pick
//	@Description	Picks one coin per opted-in user and sends notifications. Runs synchronously.
//	@Tags			Admin
//	@Produce		json
//	@Success		200	{object}	map[string]interface{}
//	@Failure		401	{object}	ErrorResponse
//	@Failure		403	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/admin/coin-of-day/run [post]
func (h *CoinOfDayAdminHandler) TriggerRun(c *gin.Context) {
	picked, skipped, errs, err := h.scheduler.RunNow()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to run coin-of-day pick"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"picked":  picked,
		"skipped": skipped,
		"errors":  errs,
	})
}
