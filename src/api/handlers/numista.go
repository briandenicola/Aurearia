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
const numistaEnrichmentBodyLimit = 256 * 1024

type NumistaHandler struct {
	lookup *services.NumistaLookupService
}

type numistaLookupWireRequest struct {
	Query             string                     `json:"query"`
	Path              models.NumistaLookupPath   `json:"path"`
	Evidence          *models.NumistaEvidence    `json:"evidence"`
	QuerySource       *models.NumistaQuerySource `json:"querySource"`
	GenerationVersion string                     `json:"generationVersion"`
}

type numistaQueryProposalWireRequest struct {
	Path     models.NumistaLookupPath `json:"path"`
	Evidence *models.NumistaEvidence  `json:"evidence"`
}

type numistaEnrichmentWireRequest struct {
	Query             string                    `json:"query"`
	Path              models.NumistaLookupPath  `json:"path"`
	Evidence          *models.NumistaEvidence   `json:"evidence"`
	Candidates        []models.NumistaCandidate `json:"candidates"`
	QuerySource       models.NumistaQuerySource `json:"querySource"`
	GenerationVersion string                    `json:"generationVersion"`
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
	if err := decoder.Decode(&wireRequest); err != nil ||
		wireRequest.Evidence == nil || wireRequest.QuerySource == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Numista lookup request"})
		return
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Numista lookup request"})
		return
	}
	request := models.NumistaLookupRequest{
		Query: wireRequest.Query, Path: wireRequest.Path, Evidence: *wireRequest.Evidence,
		QuerySource: *wireRequest.QuerySource, GenerationVersion: wireRequest.GenerationVersion,
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
	outcome = roleSafeNumistaOutcome(c, outcome)
	c.JSON(http.StatusOK, outcome)
}

// QueryProposal builds a local versioned Numista text-query proposal.
//
//	@Summary		Build a Numista query proposal
//	@Description	Builds a generated text-query proposal locally without contacting Numista.
//	@Tags			Numista
//	@Accept			json
//	@Produce		json
//	@Param			request	body		NumistaQueryProposalRequestSwagger	true	"Proposal request"
//	@Success		200		{object}	NumistaQueryProposalSwagger
//	@Failure		400		{object}	ErrorResponse
//	@Failure		401		{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/numista/query-proposal [post]
func (h *NumistaHandler) QueryProposal(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, numistaLookupBodyLimit)
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	var wireRequest numistaQueryProposalWireRequest
	if err := decoder.Decode(&wireRequest); err != nil || wireRequest.Evidence == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Numista query proposal request"})
		return
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Numista query proposal request"})
		return
	}
	proposal, err := h.lookup.Propose(models.NumistaQueryProposalRequest{
		Path: wireRequest.Path, Evidence: *wireRequest.Evidence,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, proposal)
}

// Enrich retrieves details for a server-ranked bounded candidate subset.
//
//	@Summary		Enrich and rerank Numista candidates
//	@Description	Reranks the complete broad candidate set, enriches at most five server-selected candidates with concurrency two, and retains candidates when details fail.
//	@Tags			Numista
//	@Accept			json
//	@Produce		json
//	@Param			request	body		NumistaEnrichmentRequestSwagger	true	"Enrichment request"
//	@Success		200		{object}	NumistaLookupOutcomeSwagger
//	@Failure		400		{object}	ErrorResponse
//	@Failure		401		{object}	ErrorResponse
//	@Failure		500		{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/numista/enrich [post]
func (h *NumistaHandler) Enrich(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, numistaEnrichmentBodyLimit)
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	var wireRequest numistaEnrichmentWireRequest
	if err := decoder.Decode(&wireRequest); err != nil || wireRequest.Evidence == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Numista enrichment request"})
		return
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Numista enrichment request"})
		return
	}
	request := models.NumistaEnrichmentRequest{
		NumistaLookupRequest: models.NumistaLookupRequest{
			Query: wireRequest.Query, Path: wireRequest.Path, Evidence: *wireRequest.Evidence,
			QuerySource: wireRequest.QuerySource, GenerationVersion: wireRequest.GenerationVersion,
		},
		Candidates: wireRequest.Candidates,
	}
	if err := request.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	outcome, err := h.lookup.Enrich(c.Request.Context(), request)
	if err != nil {
		if c.Request.Context().Err() != nil {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Numista enrichment failed"})
		return
	}
	c.JSON(http.StatusOK, roleSafeNumistaOutcome(c, outcome))
}

func roleSafeNumistaOutcome(c *gin.Context, outcome models.NumistaLookupOutcome) models.NumistaLookupOutcome {
	if outcome.Status != models.NumistaStatusUnconfigured {
		return outcome
	}
	role, _ := c.Get("userRole")
	if role != string(models.RoleAdmin) {
		outcome.GuidanceCode = "numista_contact_administrator"
	}
	return outcome
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
