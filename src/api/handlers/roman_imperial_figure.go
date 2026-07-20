package handlers

import (
	"net/http"
	"strconv"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
)

// RomanImperialFigureHandler serves the curated F028 imperial-figure
// reference dataset for the coin-form "Imperial figure" type-ahead picker.
type RomanImperialFigureHandler struct {
	svc *services.RomanImperialFigureService
}

// NewRomanImperialFigureHandler creates a new RomanImperialFigureHandler.
func NewRomanImperialFigureHandler(svc *services.RomanImperialFigureService) *RomanImperialFigureHandler {
	return &RomanImperialFigureHandler{svc: svc}
}

type romanImperialFigureListResponse struct {
	Figures []models.RomanImperialFigure `json:"figures"`
}

var validImperialFigureRoles = map[string]bool{
	string(models.ImperialFigureRoleEmperor): true,
	string(models.ImperialFigureRoleEmpress): true,
	string(models.ImperialFigureRoleCaesar):  true,
	string(models.ImperialFigureRoleUsurper): true,
	string(models.ImperialFigureRoleOther):   true,
}

// Search returns Roman imperial figures matching an optional name/alias
// query and/or role filter, for the coin-form type-ahead picker.
//
//	@Summary		Search Roman imperial figures
//	@Description	Returns curated Roman imperial figures (emperors, empresses, Caesars, usurpers, other) filtered by an optional name/alias query and/or role, for the coin-form "Imperial figure" type-ahead picker.
//	@Tags			Roman Imperial Figures
//	@Produce		json
//	@Param			q		query		string	false	"Name/alias search query"
//	@Param			role	query		string	false	"Filter by role: emperor, empress, caesar, usurper, other"
//	@Param			limit	query		int		false	"Max results (default 50, capped at 200)"
//	@Success		200		{object}	romanImperialFigureListResponse
//	@Failure		400		{object}	ErrorResponse
//	@Failure		401		{object}	ErrorResponse
//	@Failure		500		{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/roman-imperial-figures [get]
func (h *RomanImperialFigureHandler) Search(c *gin.Context) {
	role := c.Query("role")
	if role != "" && !validImperialFigureRoles[role] {
		respondError(c, http.StatusBadRequest, "Invalid role filter", nil)
		return
	}

	limit := 50
	if raw := c.Query("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 {
			respondError(c, http.StatusBadRequest, "Invalid limit", err)
			return
		}
		limit = parsed
	}

	figures, err := h.svc.Search(c.Query("q"), models.ImperialFigureRole(role), limit)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to search Roman imperial figures", err)
		return
	}
	c.JSON(http.StatusOK, romanImperialFigureListResponse{Figures: figures})
}

// Get returns a single Roman imperial figure by ID, used to resolve a coin's
// previously selected figure name when reopening it for editing.
//
//	@Summary		Get a Roman imperial figure
//	@Description	Returns a single curated Roman imperial figure by ID.
//	@Tags			Roman Imperial Figures
//	@Produce		json
//	@Param			id	path		int	true	"Roman imperial figure ID"
//	@Success		200	{object}	models.RomanImperialFigure
//	@Failure		400	{object}	ErrorResponse
//	@Failure		401	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/roman-imperial-figures/{id} [get]
func (h *RomanImperialFigureHandler) Get(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	figure, err := h.svc.FindByID(id)
	if err != nil {
		if repository.IsRecordNotFound(err) {
			respondError(c, http.StatusNotFound, "Roman imperial figure not found", err)
			return
		}
		respondError(c, http.StatusInternalServerError, "Failed to get Roman imperial figure", err)
		return
	}
	c.JSON(http.StatusOK, figure)
}
