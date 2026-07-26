package handlers

import (
	"errors"
	"net/http"

	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
)

// SetBuilderHandler handles Agentic set builder proposal workflow requests.
type SetBuilderHandler struct {
	service *services.SetBuilderService
}

// NewSetBuilderHandler creates a new SetBuilderHandler.
func NewSetBuilderHandler(service *services.SetBuilderService) *SetBuilderHandler {
	return &SetBuilderHandler{service: service}
}

type createSetBuilderRunRequest struct {
	Prompt string `json:"prompt"`
}

// CreateRun queues an Agentic set proposal request.
//
//	@Summary		Queue Agentic set proposal
//	@Description	Submits a natural-language Agentic set prompt for asynchronous proposal generation. No set is created by this endpoint.
//	@Tags			sets
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body		createSetBuilderRunRequest	true	"Agentic set prompt"
//	@Success		202		{object}	object{run=object}
//	@Failure		400		{object}	object{error=string}
//	@Failure		401		{object}	object{error=string}
//	@Failure		500		{object}	object{error=string}
//	@Router			/set-builder/runs [post]
func (h *SetBuilderHandler) CreateRun(c *gin.Context) {
	var body createSetBuilderRunRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	run, err := h.service.CreateRun(c.GetUint("userId"), services.SetBuilderRunRequest{Prompt: body.Prompt})
	if err != nil {
		switch {
		case errors.Is(err, services.ErrSetBuilderPromptRequired):
			c.JSON(http.StatusBadRequest, gin.H{"error": "Set builder prompt is required"})
		case errors.Is(err, services.ErrSetBuilderPromptTooLong):
			c.JSON(http.StatusBadRequest, gin.H{"error": "Set builder prompt is too long"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to submit set proposal request"})
		}
		return
	}

	c.JSON(http.StatusAccepted, gin.H{"run": run})
}
