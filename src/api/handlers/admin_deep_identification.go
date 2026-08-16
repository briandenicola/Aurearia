package handlers

import (
	"net/http"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
)

type deepIdentificationObservabilityProvider interface {
	GetObservabilitySummary() (*models.DeepIdentificationObservabilitySummary, error)
}

// AdminDeepIdentificationHandler serves redacted Deep Identification
// operational metrics to authenticated administrators.
type AdminDeepIdentificationHandler struct {
	observability deepIdentificationObservabilityProvider
}

func NewAdminDeepIdentificationHandler(observability deepIdentificationObservabilityProvider) *AdminDeepIdentificationHandler {
	return &AdminDeepIdentificationHandler{observability: observability}
}

// Observability returns aggregate Deep Identification operational metrics.
//
//	@Summary		Get Deep Identification observability
//	@Description	Returns redacted aggregate job, provider, SSE, queue, cleanup, and janitor metrics. Never includes notes, queries, claims, reports, tokens, or other user content. Admin only.
//	@Tags			Admin
//	@Produce		json
//	@Success		200	{object}	models.DeepIdentificationObservabilitySummary
//	@Failure		401	{object}	ErrorResponse
//	@Failure		403	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/admin/deep-identification/observability [get]
func (h *AdminDeepIdentificationHandler) Observability(c *gin.Context) {
	summary, err := h.observability.GetObservabilitySummary()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch Deep Identification observability"})
		return
	}
	c.JSON(http.StatusOK, summary)
}

var _ deepIdentificationObservabilityProvider = (*services.DeepIdentificationService)(nil)
