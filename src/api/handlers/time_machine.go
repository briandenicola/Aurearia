package handlers

import (
	"errors"
	"net/http"

	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
)

// TimeMachineHandler serves the historical collection replay.
type TimeMachineHandler struct {
	svc *services.TimeMachineService
}

func NewTimeMachineHandler(svc *services.TimeMachineService) *TimeMachineHandler {
	return &TimeMachineHandler{svc: svc}
}

// GetSnapshot returns the collection as it stood on a past date.
//
//	@Summary		Get the collection as of a past date
//	@Description	Reconstructs the collection as it stood on the requested date from purchase/sold dates and recorded valuation history. Coins without a purchase date cannot be placed on the timeline and are reported separately in undatedCoinCount.
//	@Tags			Stats
//	@Produce		json
//	@Param			date	query		string	true	"Target date (YYYY-MM-DD)"
//	@Success		200		{object}	services.TimeMachineSnapshot
//	@Failure		400		{object}	ErrorResponse
//	@Failure		401		{object}	ErrorResponse
//	@Failure		500		{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/stats/time-machine [get]
func (h *TimeMachineHandler) GetSnapshot(c *gin.Context) {
	userID := c.GetUint("userId")

	raw := c.Query("date")
	if raw == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "date query parameter is required (YYYY-MM-DD)"})
		return
	}
	asOf, err := services.ParseDate(raw)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "date must be in YYYY-MM-DD format"})
		return
	}

	snapshot, err := h.svc.GetSnapshot(userID, asOf)
	if err != nil {
		if errors.Is(err, services.ErrTimeMachineFutureDate) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "date cannot be in the future"})
			return
		}
		respondError(c, http.StatusInternalServerError, "Failed to reconstruct collection snapshot", err)
		return
	}

	c.JSON(http.StatusOK, snapshot)
}

// GetBounds returns the addressable date range for the timeline scrubber.
//
//	@Summary		Get the time machine's date range
//	@Description	Returns the earliest acquisition on record and today, defining the range the timeline scrubber can span. hasData is false when no coin has a purchase date.
//	@Tags			Stats
//	@Produce		json
//	@Success		200	{object}	services.TimeMachineBounds
//	@Failure		401	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/stats/time-machine/bounds [get]
func (h *TimeMachineHandler) GetBounds(c *gin.Context) {
	userID := c.GetUint("userId")

	bounds, err := h.svc.GetBounds(userID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to resolve time machine bounds", err)
		return
	}

	c.JSON(http.StatusOK, bounds)
}
