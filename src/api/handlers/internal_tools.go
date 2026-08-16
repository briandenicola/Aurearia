package handlers

import (
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
)

// InternalToolsHandler handles collection tool operations for the internal Python agent.
// All routes are protected by InternalTokenRequired middleware.
type InternalToolsHandler struct {
	collectionSvc *services.CollectionToolsService
	logger        *services.Logger
}

func NewInternalToolsHandler(
	collectionSvc *services.CollectionToolsService,
	logger *services.Logger,
) *InternalToolsHandler {
	return &InternalToolsHandler{
		collectionSvc: collectionSvc,
		logger:        logger,
	}
}

// SearchMyCollectionRequest represents the request body for search_my_collection
type SearchMyCollectionRequest struct {
	Query string `json:"query" binding:"required"`
	Limit *int   `json:"limit"`
}

// GetCoinRequest represents the request body for get_coin
type GetCoinRequest struct {
	CoinID uint `json:"coin_id" binding:"required"`
}

// TopCoinsByValueRequest represents the request body for top_coins_by_value
type TopCoinsByValueRequest struct {
	Limit *int `json:"limit"`
}

// ProposeUpdateRequest represents the request body for propose_update
type ProposeUpdateRequest struct {
	CoinID  uint           `json:"coin_id" binding:"required"`
	Changes map[string]any `json:"changes" binding:"required"`
}

// CommitUpdateRequest represents the request body for commit_update
type CommitUpdateRequest struct {
	ProposalID string `json:"proposal_id" binding:"required"`
	Token      string `json:"token" binding:"required"`
	Confirm    bool   `json:"confirm" binding:"required"`
}

// SearchMyCollection godoc
// @Summary      Search user's collection
// @Description  Search the authenticated user's collection by query filters
// @Tags         Internal
// @Accept       json
// @Produce      json
// @Param        Authorization header string true "Bearer {internal_token}"
// @Param        body body SearchMyCollectionRequest true "Search parameters"
// @Success      200 {object} map[string]interface{} "coins: array of coin summaries"
// @Failure      400 {object} map[string]interface{}
// @Failure      401 {object} map[string]interface{}
// @Failure      500 {object} map[string]interface{}
// @Router       /internal/tools/search_my_collection [post]
func (h *InternalToolsHandler) SearchMyCollection(c *gin.Context) {
	userID, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req SearchMyCollectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	coins, err := h.collectionSvc.SearchMyCollection(userID.(uint), req.Query, req.Limit)
	if err != nil {
		h.logger.Error("internal-tools", "SearchMyCollection error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "An error occurred"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"coins": coins})
}

// GetCoin godoc
// @Summary      Get a single coin by ID
// @Description  Retrieve a coin from the authenticated user's collection
// @Tags         Internal
// @Accept       json
// @Produce      json
// @Param        Authorization header string true "Bearer {internal_token}"
// @Param        body body GetCoinRequest true "Coin ID"
// @Success      200 {object} map[string]interface{} "coin: coin summary"
// @Failure      400 {object} map[string]interface{}
// @Failure      401 {object} map[string]interface{}
// @Failure      404 {object} map[string]interface{}
// @Failure      500 {object} map[string]interface{}
// @Router       /internal/tools/get_coin [post]
func (h *InternalToolsHandler) GetCoin(c *gin.Context) {
	userID, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req GetCoinRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	coin, err := h.collectionSvc.GetCoin(userID.(uint), req.CoinID)
	if err != nil {
		if errors.Is(err, services.ErrCoinNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Coin not found"})
			return
		}
		h.logger.Error("internal-tools", "GetCoin error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "An error occurred"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"coin": coin})
}

// CollectionSummary godoc
// @Summary      Get collection aggregate summary
// @Description  Retrieve aggregate statistics for the authenticated user's collection
// @Tags         Internal
// @Accept       json
// @Produce      json
// @Param        Authorization header string true "Bearer {internal_token}"
// @Success      200 {object} map[string]interface{} "summary: aggregate summary"
// @Failure      401 {object} map[string]interface{}
// @Failure      500 {object} map[string]interface{}
// @Router       /internal/tools/collection_summary [post]
func (h *InternalToolsHandler) CollectionSummary(c *gin.Context) {
	userID, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	summary, err := h.collectionSvc.CollectionSummary(userID.(uint))
	if err != nil {
		h.logger.Error("internal-tools", "CollectionSummary error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "An error occurred"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"summary": summary})
}

// TopCoinsByValue godoc
// @Summary      Get top coins by current value
// @Description  Retrieve the top coins by current value from the authenticated user's collection
// @Tags         Internal
// @Accept       json
// @Produce      json
// @Param        Authorization header string true "Bearer {internal_token}"
// @Param        body body TopCoinsByValueRequest false "Limit (default 3, max 10)"
// @Success      200 {object} map[string]interface{} "coins: array of coin summaries"
// @Failure      400 {object} map[string]interface{}
// @Failure      401 {object} map[string]interface{}
// @Failure      500 {object} map[string]interface{}
// @Router       /internal/tools/top_coins_by_value [post]
func (h *InternalToolsHandler) TopCoinsByValue(c *gin.Context) {
	userID, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req TopCoinsByValueRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	coins, err := h.collectionSvc.TopCoinsByValue(userID.(uint), req.Limit)
	if err != nil {
		h.logger.Error("internal-tools", "TopCoinsByValue error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "An error occurred"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"coins": coins})
}

// ProposeUpdate godoc
// @Summary      Create an update proposal
// @Description  Create a proposal to update allowlisted fields on a coin
// @Tags         Internal
// @Accept       json
// @Produce      json
// @Param        Authorization header string true "Bearer {internal_token}"
// @Param        body body ProposeUpdateRequest true "Coin ID and changes"
// @Success      200 {object} map[string]interface{} "proposal: proposal preview with token"
// @Failure      400 {object} map[string]interface{}
// @Failure      401 {object} map[string]interface{}
// @Failure      404 {object} map[string]interface{}
// @Failure      500 {object} map[string]interface{}
// @Router       /internal/tools/propose_update [post]
func (h *InternalToolsHandler) ProposeUpdate(c *gin.Context) {
	userID, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req ProposeUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	proposal, err := h.collectionSvc.ProposeUpdate(userID.(uint), req.CoinID, req.Changes)
	if err != nil {
		if errors.Is(err, services.ErrCoinNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Coin not found"})
			return
		}
		if errors.Is(err, services.ErrInvalidFieldChanges) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		h.logger.Error("internal-tools", "ProposeUpdate error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "An error occurred"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"proposal": proposal})
}

// CommitUpdate godoc
// @Summary      Commit an update proposal
// @Description  Commit a previously created proposal with explicit confirmation
// @Tags         Internal
// @Accept       json
// @Produce      json
// @Param        Authorization header string true "Bearer {internal_token}"
// @Param        body body CommitUpdateRequest true "Proposal ID, token, and confirmation"
// @Success      200 {object} map[string]interface{} "result: commit result"
// @Failure      400 {object} map[string]interface{}
// @Failure      401 {object} map[string]interface{}
// @Failure      404 {object} map[string]interface{}
// @Failure      500 {object} map[string]interface{}
// @Router       /internal/tools/commit_update [post]
func (h *InternalToolsHandler) CommitUpdate(c *gin.Context) {
	userID, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req CommitUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	result, err := h.collectionSvc.CommitUpdate(userID.(uint), req.ProposalID, req.Token, req.Confirm)
	if err != nil {
		if errors.Is(err, services.ErrProposalConfirmationReq) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Confirmation required"})
			return
		}
		if errors.Is(err, services.ErrProposalStateConflict) {
			c.JSON(http.StatusConflict, gin.H{"error": "Proposal is not in pending state or has expired"})
			return
		}
		if errors.Is(err, services.ErrProposalTokenInvalid) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid proposal token"})
			return
		}
		h.logger.Error("internal-tools", "CommitUpdate error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "An error occurred"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"result": result})
}

// deepNomismaCallBudget is fixed (not a settings key) per tasks.md T052 /
// contracts/agent-internal-contract.md §7 sample catalog entry
// ("nomisma", "call_budget": 3) - Nomisma has no per-deployment tuning
// knob, unlike Numista's SettingDeepIdentificationNumistaCallBudget.
const deepNomismaCallBudget = 3

// DeepProviderToolsHandler exposes the Go-hosted provider tool boundary
// consumed by the Python deep identification pipeline
// (contracts/agent-internal-contract.md §7). Routes are protected by
// middleware.InternalJobTokenRequired, which binds each call to a single
// (userID, jobID) pair so budgets are enforced per job run and a
// forged/foreign job binding is rejected before reaching this handler.
type DeepProviderToolsHandler struct {
	numistaClient services.NumistaClient
	nomismaClient services.NomismaClient
	ocreClient    services.OCREClient
	ocreCache     *services.OCRECache
	settingsSvc   *services.SettingsService
	budgets       *services.DeepProviderBudgetTracker
	logger        *services.Logger
}

// NewDeepProviderToolsHandler constructs the provider-tool handler.
func NewDeepProviderToolsHandler(
	numistaClient services.NumistaClient,
	nomismaClient services.NomismaClient,
	ocreClient services.OCREClient,
	ocreCache *services.OCRECache,
	settingsSvc *services.SettingsService,
	budgets *services.DeepProviderBudgetTracker,
	logger *services.Logger,
) *DeepProviderToolsHandler {
	return &DeepProviderToolsHandler{
		numistaClient: numistaClient,
		nomismaClient: nomismaClient,
		ocreClient:    ocreClient,
		ocreCache:     ocreCache,
		settingsSvc:   settingsSvc,
		budgets:       budgets,
		logger:        logger,
	}
}

// deepProviderJobID reads the job binding set by InternalJobTokenRequired.
// Returns (0, false) if the middleware did not run (defensive - should be
// unreachable in production routing).
func deepProviderJobID(c *gin.Context) (uint, bool) {
	raw, exists := c.Get("deepJobId")
	if !exists {
		return 0, false
	}
	jobID, ok := raw.(uint)
	return jobID, ok
}

// NumistaSearchRequest is the body for POST /internal/tools/numista_search.
type NumistaSearchRequest struct {
	Query string `json:"query" binding:"required"`
	Limit int    `json:"limit"`
}

// NumistaDetailRequest is the body for POST /internal/tools/numista_detail.
type NumistaDetailRequest struct {
	ID int `json:"id" binding:"required"`
}

// NomismaSearchRequest is the body for POST /internal/tools/nomisma_search.
type NomismaSearchRequest struct {
	Query string `json:"query" binding:"required"`
	Limit int    `json:"limit"`
}

// NumistaSearch godoc
// @Summary      Deep-identification Numista search tool
// @Description  Job-scoped, call-budget-limited Numista search for the Python deep identification pipeline
// @Tags         Internal
// @Accept       json
// @Produce      json
// @Param        Authorization header string true "******"
// @Param        body body NumistaSearchRequest true "Search query"
// @Success      200 {object} map[string]interface{} "status, candidates, attribution"
// @Failure      400 {object} map[string]interface{}
// @Failure      401 {object} map[string]interface{}
// @Router       /internal/tools/numista_search [post]
func (h *DeepProviderToolsHandler) NumistaSearch(c *gin.Context) {
	jobID, ok := deepProviderJobID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing job binding"})
		return
	}

	var req NumistaSearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	limit := req.Limit
	if limit <= 0 {
		limit = 5
	}

	budget := h.settingsSvc.GetDeepIdentificationSettings().NumistaCallBudget
	if allowed, callCount := h.budgets.TryConsume(jobID, "numista", budget); !allowed {
		h.logger.Warn("internal-tools", "deep identification numista call budget exceeded for job %d (count=%d, budget=%d)", jobID, callCount, budget)
		c.JSON(http.StatusOK, gin.H{"status": "quota_limited", "candidates": []models.NumistaCandidate{}, "attribution": "Source: Numista"})
		return
	}

	candidates, err := h.numistaClient.Search(c.Request.Context(), req.Query, limit)
	status, candidates := deepNumistaSearchStatus(candidates, err)
	c.JSON(http.StatusOK, gin.H{"status": status, "candidates": candidates, "attribution": "Source: Numista"})
}

// NumistaDetail godoc
// @Summary      Deep-identification Numista detail tool
// @Description  Job-scoped, call-budget-limited Numista detail lookup for the Python deep identification pipeline
// @Tags         Internal
// @Accept       json
// @Produce      json
// @Param        Authorization header string true "******"
// @Param        body body NumistaDetailRequest true "Numista catalogue ID"
// @Success      200 {object} map[string]interface{} "status, candidate, identifier"
// @Failure      400 {object} map[string]interface{}
// @Failure      401 {object} map[string]interface{}
// @Router       /internal/tools/numista_detail [post]
func (h *DeepProviderToolsHandler) NumistaDetail(c *gin.Context) {
	jobID, ok := deepProviderJobID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing job binding"})
		return
	}

	var req NumistaDetailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	budget := h.settingsSvc.GetDeepIdentificationSettings().NumistaCallBudget
	if allowed, callCount := h.budgets.TryConsume(jobID, "numista", budget); !allowed {
		h.logger.Warn("internal-tools", "deep identification numista call budget exceeded for job %d (count=%d, budget=%d)", jobID, callCount, budget)
		c.JSON(http.StatusOK, gin.H{"status": "quota_limited"})
		return
	}

	candidate, err := h.numistaClient.Detail(c.Request.Context(), req.ID)
	status := deepNumistaDetailStatus(err)
	if status != "ok" {
		c.JSON(http.StatusOK, gin.H{"status": status})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": status, "candidate": candidate, "identifier": "N#" + strconv.Itoa(req.ID)})
}

// NomismaSearch godoc
// @Summary      Deep-identification Nomisma search tool
// @Description  Job-scoped, call-budget-limited Nomisma reconciliation search for the Python deep identification pipeline
// @Tags         Internal
// @Accept       json
// @Produce      json
// @Param        Authorization header string true "******"
// @Param        body body NomismaSearchRequest true "Search query"
// @Success      200 {object} map[string]interface{} "status, candidates, attribution"
// @Failure      400 {object} map[string]interface{}
// @Failure      401 {object} map[string]interface{}
// @Router       /internal/tools/nomisma_search [post]
func (h *DeepProviderToolsHandler) NomismaSearch(c *gin.Context) {
	jobID, ok := deepProviderJobID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing job binding"})
		return
	}

	var req NomismaSearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	limit := req.Limit
	if limit <= 0 {
		limit = 5
	}

	if allowed, callCount := h.budgets.TryConsume(jobID, "nomisma", deepNomismaCallBudget); !allowed {
		h.logger.Warn("internal-tools", "deep identification nomisma call budget exceeded for job %d (count=%d, budget=%d)", jobID, callCount, deepNomismaCallBudget)
		c.JSON(http.StatusOK, gin.H{"status": "unavailable", "candidates": []services.NomismaCandidate{}, "attribution": "Data: Nomisma.org (CC BY)"})
		return
	}

	candidates, kind, err := h.nomismaClient.Search(c.Request.Context(), req.Query, limit)
	status := "ok"
	if err != nil || kind != "" {
		switch kind {
		case services.NomismaErrorNoMatch:
			status = "empty"
		default:
			status = "unavailable"
		}
	}
	if candidates == nil {
		candidates = []services.NomismaCandidate{}
	}
	c.JSON(http.StatusOK, gin.H{"status": status, "candidates": candidates, "attribution": "Data: Nomisma.org (CC BY)"})
}

// ocreAttribution is the exact, fixed OCRE attribution string (contract §6 /
// FR-019). It is emitted only when OCRE is actually queried, is byte-for-byte
// distinct from every other provider's attribution, and is rendered with a
// dedicated component on the web tier (OCREAttribution.vue).
const ocreAttribution = "Coin type data: Online Coins of the Roman Empire (OCRE), American Numismatic Society — ODbL 1.0."

// OCRESearchRequest is the body for POST /internal/tools/ocre_search
// (contract §1). All fields are optional except that at least one
// type-bearing signal (ruler/denomination/mint/ocre_id) must decode; Go
// re-validates every value into a Nomisma id slug before it can reach SPARQL.
type OCRESearchRequest struct {
	Ruler        string   `json:"ruler"`
	Denomination string   `json:"denomination"`
	Mint         string   `json:"mint"`
	Material     string   `json:"material"`
	LegendTokens []string `json:"legend_tokens"`
	OCREID       string   `json:"ocre_id"`
	Limit        int      `json:"limit"`
}

// OCRESearchResponse is the always-HTTP-200 response for ocre_search
// (contract §1). `status` is one of ok|empty|invalid_response|unavailable|
// timeout|quota_limited|cancelled. An upstream/transport problem never
// produces a 4xx/5xx here — only a missing job binding (401) or an
// unparseable body (400) do.
type OCRESearchResponse struct {
	Status      string                   `json:"status"`
	Candidates  []services.OCRECandidate `json:"candidates"`
	Attribution string                   `json:"attribution"`
}

// OCRESearch godoc
// @Summary      Deep-identification OCRE search tool
// @Description  Job-scoped, call-budget-limited OCRE (Nomisma SPARQL) coin-type search for the Python deep identification pipeline
// @Tags         Internal
// @Accept       json
// @Produce      json
// @Param        Authorization header string true "******"
// @Param        body body OCRESearchRequest true "Bound OCRE query signals"
// @Success      200 {object} OCRESearchResponse "status, candidates, attribution"
// @Failure      400 {object} map[string]interface{}
// @Failure      401 {object} map[string]interface{}
// @Router       /internal/tools/ocre_search [post]
func (h *DeepProviderToolsHandler) OCRESearch(c *gin.Context) {
	jobID, ok := deepProviderJobID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing job binding"})
		return
	}

	var req OCRESearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	settings := h.settingsSvc.GetDeepIdentificationSettings()

	// Defense in depth (FR-004/FR-016): even a valid job token must not reach
	// upstream while the OCRE flag is off. The Python node already short-
	// circuits to a not_automated row without a call, so this branch is only
	// reachable by a direct internal invocation that bypasses the node — answer
	// with a typed, non-contributing "unavailable" and never touch the client,
	// cache, or budget. Rollback (flag off) therefore guarantees zero OCRE
	// upstream calls at the boundary itself, not merely at the caller.
	if !settings.OCREEnabled {
		h.logger.Info("internal-tools", "OCRE search invoked for job %d while OCRE flag disabled; returning unavailable without an upstream call", jobID)
		c.JSON(http.StatusOK, OCRESearchResponse{Status: "unavailable", Candidates: []services.OCRECandidate{}, Attribution: ocreAttribution})
		return
	}

	params := services.NewOCREQueryParams(
		req.Ruler, req.Denomination, req.Mint, req.Material, req.LegendTokens, req.OCREID, req.Limit,
	)
	flagGeneration := strconv.FormatBool(settings.OCREEnabled)

	// No decodable type-bearing signal → nothing to query, and never a call
	// or a budget charge (data-model §2 invariant).
	if !params.HasSignal() {
		c.JSON(http.StatusOK, OCRESearchResponse{Status: "empty", Candidates: []services.OCRECandidate{}, Attribution: ocreAttribution})
		return
	}

	// Cache-check first: a cache hit is not an upstream call, so it neither
	// consumes budget nor re-queries Nomisma.
	if h.ocreCache != nil {
		if status, cached, hit := h.ocreCache.Get(params, flagGeneration); hit {
			c.JSON(http.StatusOK, OCRESearchResponse{
				Status:      ocreCacheStatusToWire(status),
				Candidates:  ocreValidateCandidates(cached),
				Attribution: ocreAttribution,
			})
			return
		}
	}

	if allowed, callCount := h.budgets.TryConsume(jobID, "ocre", settings.OCRECallBudget); !allowed {
		h.logger.Warn("internal-tools", "deep identification OCRE call budget exceeded for job %d (count=%d, budget=%d)", jobID, callCount, settings.OCRECallBudget)
		c.JSON(http.StatusOK, OCRESearchResponse{Status: "quota_limited", Candidates: []services.OCRECandidate{}, Attribution: ocreAttribution})
		return
	}

	candidates, kind, _ := h.ocreClient.Search(c.Request.Context(), params, req.Limit)
	status := ocreErrorKindToWire(kind, len(candidates))
	candidates = ocreValidateCandidates(candidates)

	// Cache only settled ok/no_match outcomes; transient failures are never
	// cached so an outage never gets "stuck" for the TTL window.
	if h.ocreCache != nil {
		switch status {
		case "ok":
			h.ocreCache.Set(params, flagGeneration, services.OCRESearchOK, candidates)
		case "empty":
			h.ocreCache.Set(params, flagGeneration, services.OCRESearchNoMatch, nil)
		}
	}

	c.JSON(http.StatusOK, OCRESearchResponse{Status: status, Candidates: candidates, Attribution: ocreAttribution})
}

// ocreErrorKindToWire maps a typed OCREErrorKind to the ocre_search wire
// status vocabulary (data-model §4). An empty kind with candidates is "ok";
// an empty kind with zero candidates (or an explicit no_match) is "empty".
func ocreErrorKindToWire(kind services.OCREErrorKind, candidateCount int) string {
	switch kind {
	case "":
		if candidateCount == 0 {
			return "empty"
		}
		return "ok"
	case services.OCREErrorNoMatch:
		return "empty"
	case services.OCREErrorInvalidResponse:
		return "invalid_response"
	case services.OCREErrorTimeout:
		return "timeout"
	case services.OCREErrorCancelled:
		return "cancelled"
	case services.OCREErrorInvalidRequest:
		return "empty"
	default: // OCREErrorUnavailable and any unknown kind
		return "unavailable"
	}
}

// ocreCacheStatusToWire maps a cached OCRESearchStatus to the wire status.
func ocreCacheStatusToWire(status services.OCRESearchStatus) string {
	switch status {
	case services.OCRESearchOK:
		return "ok"
	case services.OCRESearchNoMatch:
		return "empty"
	default:
		return "unavailable"
	}
}

// ocreValidateCandidates re-validates every candidate's citation host is
// numismatics.org before emission (FR-011, defense in depth on top of the
// client's own re-check). Always returns a non-nil slice.
func ocreValidateCandidates(candidates []services.OCRECandidate) []services.OCRECandidate {
	out := make([]services.OCRECandidate, 0, len(candidates))
	for _, candidate := range candidates {
		u, err := url.Parse(candidate.TypeURI)
		if err != nil {
			continue
		}
		if strings.EqualFold(u.Hostname(), "numismatics.org") {
			out = append(out, candidate)
		}
	}
	return out
}

// simplified status vocabulary of contracts/agent-internal-contract.md §7
// ("ok|empty|unconfigured|quota_limited|timeout|unavailable"), always
// returning a non-nil candidates slice.
func deepNumistaSearchStatus(candidates []models.NumistaCandidate, err error) (string, []models.NumistaCandidate) {
	if candidates == nil {
		candidates = []models.NumistaCandidate{}
	}
	if err == nil {
		if len(candidates) == 0 {
			return "empty", candidates
		}
		return "ok", candidates
	}
	var numistaErr *services.NumistaError
	if errors.As(err, &numistaErr) {
		switch numistaErr.Kind {
		case services.NumistaErrorUnconfigured, services.NumistaErrorUnauthorized:
			return "unconfigured", []models.NumistaCandidate{}
		case services.NumistaErrorQuotaLimited:
			return "quota_limited", []models.NumistaCandidate{}
		case services.NumistaErrorTimeout:
			return "timeout", []models.NumistaCandidate{}
		}
	}
	return "unavailable", []models.NumistaCandidate{}
}

// deepNumistaDetailStatus maps a raw NumistaClient.Detail error to the same
// simplified status vocabulary as deepNumistaSearchStatus.
func deepNumistaDetailStatus(err error) string {
	if err == nil {
		return "ok"
	}
	var numistaErr *services.NumistaError
	if errors.As(err, &numistaErr) {
		switch numistaErr.Kind {
		case services.NumistaErrorUnconfigured, services.NumistaErrorUnauthorized:
			return "unconfigured"
		case services.NumistaErrorQuotaLimited:
			return "quota_limited"
		case services.NumistaErrorTimeout:
			return "timeout"
		}
	}
	return "unavailable"
}
