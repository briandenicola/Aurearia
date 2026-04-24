package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
)

type NumistaHandler struct {
	client      *http.Client
	settingsSvc *services.SettingsService
}

func NewNumistaHandler(settingsSvc *services.SettingsService) *NumistaHandler {
	return &NumistaHandler{
		client:      &http.Client{Timeout: 15 * time.Second},
		settingsSvc: settingsSvc,
	}
}

// Search proxies a search request to the Numista API.
//
//	@Summary		Search Numista catalog
//	@Description	Searches the Numista coin catalog by query string. Requires a Numista API key in admin settings.
//	@Tags			Numista
//	@Produce		json
//	@Param			q	query		string	true	"Search query"
//	@Success		200	{object}	object
//	@Failure		400	{object}	ErrorResponse
//	@Failure		401	{object}	ErrorResponse
//	@Failure		503	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/numista/search [get]
func (h *NumistaHandler) Search(c *gin.Context) {
	apiKey := h.settingsSvc.GetSetting(services.SettingNumistaAPIKey)
	if apiKey == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Numista API key not configured. Set it in Admin → Settings."})
		return
	}

	q := c.Query("q")
	if q == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Query parameter 'q' is required"})
		return
	}

	numistaURL := fmt.Sprintf("https://api.numista.com/v3/types?q=%s&category=coin&count=10&lang=en", url.QueryEscape(q))

	req, err := http.NewRequest("GET", numistaURL, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create request"})
		return
	}
	req.Header.Set("Numista-API-Key", apiKey)

	resp, err := h.client.Do(req)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Failed to reach Numista API"})
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read response"})
		return
	}

	if resp.StatusCode != http.StatusOK {
		c.JSON(resp.StatusCode, gin.H{"error": "Numista API error"})
		return
	}

	var result interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse response"})
		return
	}

	c.JSON(http.StatusOK, result)
}
