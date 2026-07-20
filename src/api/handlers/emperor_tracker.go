package handlers

import (
	"net/http"

	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
)

// emperorTrackerSuggestionLimit caps the V1 "what to pursue next" list.
const emperorTrackerSuggestionLimit = 10

// EmperorTrackerHandler serves F028's /stats/emperors progress data.
type EmperorTrackerHandler struct {
	svc      *services.EmperorTrackerService
	userRepo *repository.UserRepository
}

// NewEmperorTrackerHandler creates a new EmperorTrackerHandler.
func NewEmperorTrackerHandler(svc *services.EmperorTrackerService, userRepo *repository.UserRepository) *EmperorTrackerHandler {
	return &EmperorTrackerHandler{svc: svc, userRepo: userRepo}
}

// GetProgress returns the authenticated user's F028 emperor-collection
// completion progress: the primary "commonly accepted Augustuses" goal plus
// V1 suggestions, and any of the three optional categories (usurpers,
// empresses, other figures) the user has enabled in Settings.
//
//	@Summary		Get emperor tracker progress
//	@Description	Returns the authenticated user's Roman-emperor collection completion progress. Requires emperorTrackerEnabled to be set on the user's profile.
//	@Tags			Emperor Tracker
//	@Produce		json
//	@Success		200	{object}	services.EmperorTrackerResult
//	@Failure		401	{object}	ErrorResponse
//	@Failure		403	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/stats/emperors [get]
func (h *EmperorTrackerHandler) GetProgress(c *gin.Context) {
	userID := c.GetUint("userId")

	user, err := h.userRepo.FindByID(userID)
	if err != nil {
		respondError(c, http.StatusNotFound, "User not found", err)
		return
	}
	if !user.EmperorTrackerEnabled {
		respondError(c, http.StatusForbidden, "Emperor tracker is not enabled for this account", nil)
		return
	}

	result, err := h.svc.FullProgress(
		userID,
		user.EmperorTrackerShowUsurpers,
		user.EmperorTrackerShowEmpresses,
		user.EmperorTrackerShowOtherFigures,
		emperorTrackerSuggestionLimit,
	)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to compute emperor tracker progress", err)
		return
	}
	c.JSON(http.StatusOK, result)
}
