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

type emperorTrackerHighlightRequest struct {
	CoinID *uint `json:"coinId"`
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

// SetHighlight updates or clears the authenticated user's highlighted coin
// for an imperial figure in the emperor tracker tray.
//
//	@Summary		Set highlighted emperor tracker coin
//	@Description	Sets the user-selected highlighted coin for one Roman imperial figure. The coin must be an active Roman collection coin matched to that figure. Send coinId:null to clear the explicit choice.
//	@Tags			Emperor Tracker
//	@Accept			json
//	@Produce		json
//	@Param			figureId	path	int								true	"Roman imperial figure ID"
//	@Param			body		body	emperorTrackerHighlightRequest	true	"Highlight selection"
//	@Success		204
//	@Failure		400	{object}	ErrorResponse
//	@Failure		401	{object}	ErrorResponse
//	@Failure		403	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/stats/emperors/highlights/{figureId} [put]
func (h *EmperorTrackerHandler) SetHighlight(c *gin.Context) {
	userID := c.GetUint("userId")
	figureID, ok := parseID(c, "figureId")
	if !ok {
		return
	}

	user, err := h.userRepo.FindByID(userID)
	if err != nil {
		respondError(c, http.StatusNotFound, "User not found", err)
		return
	}
	if !user.EmperorTrackerEnabled {
		respondError(c, http.StatusForbidden, "Emperor tracker is not enabled for this account", nil)
		return
	}

	var req emperorTrackerHighlightRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "Invalid request payload", err)
		return
	}
	if req.CoinID == nil {
		if err := h.svc.ClearHighlight(userID, figureID); err != nil {
			respondError(c, http.StatusInternalServerError, "Failed to clear emperor tracker highlight", err)
			return
		}
		c.Status(http.StatusNoContent)
		return
	}
	if err := h.svc.SetHighlight(userID, figureID, *req.CoinID); err != nil {
		if repository.IsRecordNotFound(err) {
			respondError(c, http.StatusBadRequest, "Highlighted coin must be an active Roman coin matched to this figure", err)
			return
		}
		respondError(c, http.StatusInternalServerError, "Failed to set emperor tracker highlight", err)
		return
	}
	c.Status(http.StatusNoContent)
}
