package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
)

const numistaLookupBodyLimit = 32 * 1024

type NumistaHandler struct {
	lookup *services.NumistaLookupService
}

type numistaLookupWireRequest struct {
	Query    string                   `json:"query"`
	Path     models.NumistaLookupPath `json:"path"`
	Evidence *models.NumistaEvidence  `json:"evidence"`
}

func NewNumistaHandler(lookup *services.NumistaLookupService) *NumistaHandler {
	return &NumistaHandler{lookup: lookup}
}

// Lookup runs a typed broad Numista catalog search.
//
//	@Summary		Run broad Numista lookup
//	@Description	Searches Numista through the shared application lookup service and returns explained, application-owned candidates.
//	@Tags			Numista
//	@Accept			json
//	@Produce		json
//	@Param			request	body		NumistaLookupRequestSwagger	true	"Lookup request"
//	@Success		200		{object}	NumistaLookupOutcomeSwagger
//	@Failure		400		{object}	ErrorResponse
//	@Failure		401		{object}	ErrorResponse
//	@Failure		500		{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/numista/lookup [post]
func (h *NumistaHandler) Lookup(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, numistaLookupBodyLimit)
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	var wireRequest numistaLookupWireRequest
	if err := decoder.Decode(&wireRequest); err != nil || wireRequest.Evidence == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Numista lookup request"})
		return
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Numista lookup request"})
		return
	}
	request := models.NumistaLookupRequest{
		Query: wireRequest.Query, Path: wireRequest.Path, Evidence: *wireRequest.Evidence,
	}
	outcome, err := h.lookup.Lookup(c.Request.Context(), request)
	if err != nil {
		if c.Request.Context().Err() != nil {
			return
		}
		if validationRequest := request.Validate(); validationRequest != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": validationRequest.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Numista lookup failed"})
		return
	}
	c.JSON(http.StatusOK, outcome)
}

// Search is the deprecated compatibility adapter for the legacy Numista response.
//
//	@Summary		Search Numista catalog
//	@Description	Deprecated compatibility route preserving the legacy count/types response.
//	@Tags			Numista
//	@Produce		json
//	@Param			q	query		string	true	"Search query"	minlength(1)	maxlength(500)
//	@Success		200	{object}	LegacyNumistaSearchResponseSwagger
//	@Failure		400	{object}	ErrorResponse
//	@Failure		401	{object}	ErrorResponse
//	@Failure		503	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/numista/search [get]
func (h *NumistaHandler) Search(c *gin.Context) {
	query := c.Query("q")
	if strings.TrimSpace(query) == "" || len([]rune(query)) > models.NumistaMaxQueryLength {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Query parameter 'q' must contain 1 to 500 characters"})
		return
	}
	response, err := h.lookup.LegacySearch(c.Request.Context(), query)
	if err != nil {
		if c.Request.Context().Err() != nil {
			return
		}
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Numista lookup is unavailable"})
		return
	}
	c.JSON(http.StatusOK, response)
}
