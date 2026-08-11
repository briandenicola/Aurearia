package handlers

import (
	"net/http"
	"strings"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
)

// AdminNumistaHandler serves redacted Numista configuration and rolling health.
type AdminNumistaHandler struct {
	telemetry *services.NumistaTelemetry
	settings  services.NumistaSettingsProvider
}

func NewAdminNumistaHandler(
	telemetry *services.NumistaTelemetry,
	settings services.NumistaSettingsProvider,
) *AdminNumistaHandler {
	return &AdminNumistaHandler{telemetry: telemetry, settings: settings}
}

// Health returns redacted Numista configuration and rolling operational health.
//
//	@Summary		Get Numista operational health
//	@Description	Returns redacted Numista configuration validity and bounded rolling status, latency, cache, quota, and enrichment aggregates. Admin only.
//	@Tags			Admin
//	@Produce		json
//	@Success		200	{object}	models.NumistaHealthSummary
//	@Failure		401	{object}	ErrorResponse
//	@Failure		403	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/admin/numista/health [get]
func (h *AdminNumistaHandler) Health(c *gin.Context) {
	configured := false
	configurationValid := false
	if h.settings != nil {
		configured = strings.TrimSpace(h.settings.GetSetting(services.SettingNumistaAPIKey)) != ""
		configurationValid = h.settings.GetNumistaSettings().Valid
	}

	summary := models.NumistaHealthSummary{
		Configured:         configured,
		ConfigurationValid: configurationValid,
		StatusCounts:       make(map[models.NumistaLookupStatus]int),
	}
	if h.telemetry != nil {
		summary = h.telemetry.Health(configured, configurationValid)
	}
	c.JSON(http.StatusOK, summary)
}
