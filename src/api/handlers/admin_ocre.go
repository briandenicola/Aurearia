package handlers

import (
	"net/http"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
)

// ocreHealthSettingsProvider is the narrow settings dependency of the admin
// OCRE health handler — it reads the validated deep-identification settings
// snapshot (enablement flag, call budget, gate validity).
type ocreHealthSettingsProvider interface {
	GetDeepIdentificationSettings() services.DeepIdentificationSettings
}

// ocreHealthProviderRunReader is the narrow persistence dependency — a
// bounded, non-user-scoped read of the latest OCRE provider-run outcome.
type ocreHealthProviderRunReader interface {
	GetLatestProviderStatus(provider models.DeepProviderName) (*models.DeepIdentificationProviderRun, error)
}

// AdminOCREHandler serves the admin-only OCRE Deep Analysis health surface
// (Feature 345 US4). It exposes enablement/gate state and the last recorded
// provider-run outcome class — never any per-job user content.
type AdminOCREHandler struct {
	settings ocreHealthSettingsProvider
	runs     ocreHealthProviderRunReader
}

func NewAdminOCREHandler(settings ocreHealthSettingsProvider, runs ocreHealthProviderRunReader) *AdminOCREHandler {
	return &AdminOCREHandler{settings: settings, runs: runs}
}

// Health returns bounded OCRE Deep Analysis enablement and operational health.
//
//	@Summary		Get OCRE Deep Analysis operational health
//	@Description	Returns the OCRE Deep Analysis enablement flag, per-job call budget, configuration gate validity, and the last recorded provider-run outcome class and timestamp. No per-job user content is exposed. Admin only.
//	@Tags			Admin
//	@Produce		json
//	@Success		200	{object}	models.OCREHealthSummary
//	@Failure		401	{object}	ErrorResponse
//	@Failure		403	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/admin/deep-identification/ocre/health [get]
func (h *AdminOCREHandler) Health(c *gin.Context) {
	summary := models.OCREHealthSummary{}
	if h.settings != nil {
		s := h.settings.GetDeepIdentificationSettings()
		summary.Enabled = s.OCREEnabled
		summary.CallBudget = s.OCRECallBudget
		summary.GateValidated = s.Valid
	}
	if h.runs != nil {
		if run, err := h.runs.GetLatestProviderStatus(models.DeepProviderOCRE); err == nil && run != nil {
			status := run.Status
			summary.LastOutcome = &status
			summary.LastCheckedAt = run.CompletedAt
		}
	}
	c.JSON(http.StatusOK, summary)
}
