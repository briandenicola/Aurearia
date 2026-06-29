package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
)

type AgentHandler struct {
	repo          *repository.AgentRepository
	userRepo      *repository.UserRepository
	journalRepo   *repository.JournalRepository
	proxy         *services.AgentProxy
	collectionSvc *services.CollectionToolsService
	settingsSvc   *services.SettingsService
	tokenSvc      *services.InternalTokenService
	guard         *services.ContentGuard
	logger        *services.Logger
	toolsBaseURL  string
}

func NewAgentHandler(
	repo *repository.AgentRepository,
	userRepo *repository.UserRepository,
	journalRepo *repository.JournalRepository,
	proxy *services.AgentProxy,
	collectionSvc *services.CollectionToolsService,
	settingsSvc *services.SettingsService,
	tokenSvc *services.InternalTokenService,
	guard *services.ContentGuard,
	logger *services.Logger,
	toolsBaseURL string,
) *AgentHandler {
	return &AgentHandler{
		repo:          repo,
		userRepo:      userRepo,
		journalRepo:   journalRepo,
		proxy:         proxy,
		collectionSvc: collectionSvc,
		settingsSvc:   settingsSvc,
		tokenSvc:      tokenSvc,
		guard:         guard,
		logger:        logger,
		toolsBaseURL:  toolsBaseURL,
	}
}

// resolveLLMConfig wraps settingsSvc.ResolveLLMConfig for handler use,
// returning an error message string for HTTP responses.
func (h *AgentHandler) resolveLLMConfig() (services.LLMConfig, string) {
	cfg, err := h.settingsSvc.ResolveLLMConfig()
	if err != nil {
		return services.LLMConfig{}, err.Error()
	}
	return cfg, ""
}

// Chat request/response types

type AgentChatRequest struct {
	Message    string                          `json:"message" binding:"required"`
	History    []AgentChatMessage              `json:"history"`
	AppContext *services.CollectionChatContext `json:"appContext,omitempty"`
}

type AgentChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type CoinSuggestion struct {
	Name                string                     `json:"name"`
	Description         string                     `json:"description"`
	Category            string                     `json:"category"`
	Era                 string                     `json:"era"`
	Ruler               string                     `json:"ruler"`
	Material            string                     `json:"material"`
	Denomination        string                     `json:"denomination"`
	EstPrice            string                     `json:"estPrice"`
	ImageURL            string                     `json:"imageUrl"`
	SourceURL           string                     `json:"sourceUrl"`
	SourceName          string                     `json:"sourceName"`
	CandidateReferences []CandidateReferenceDTORef `json:"candidateReferences,omitempty"`
}

type CandidateReferenceDTORef struct {
	Catalog string `json:"catalog"`
	Volume  string `json:"volume,omitempty"`
	Number  string `json:"number"`
	URI     string `json:"uri,omitempty"`
}

type AgentChatResponse struct {
	Message     string           `json:"message"`
	Suggestions []CoinSuggestion `json:"suggestions"`
}

const DefaultCoinSearchPrompt = `You are a numismatic search specialist focused on Greek and Roman coinage up through the Byzantine Era. You specialize in finding that rare gem of a coin for just the right price.

CRITICAL RULES:
- Search for coins that are CURRENTLY FOR SALE — never return sold items or past auction results
- ONLY search reputable dealer sites: vcoins.com, ma-shops.com, forumancientcoins.com, biddr.com, catawiki.com, hjbltd.com
- Add "for sale" or "buy now" to your search queries
- For EACH result, you MUST provide the exact URL to the listing page
- NEVER invent, guess, or recall URLs from memory — only use URLs from search results
- Return ONLY results you actually found in your search
- If a listing says "SOLD", "Auction ended", or "Realized price" — SKIP IT
- ACSSearch.info is a PAST auction archive — do NOT use it
- Quality over quantity — 2 verified, available results beat 5 questionable ones
- Flag any concerns about authenticity or condition
- Mention dealer/auction house reputation if known`

const DefaultCoinShowsPrompt = `You are a coin show search specialist focused on numismatic conventions and collecting events.

Search for upcoming coin shows, expos, and conventions, especially those featuring ancient, Greek, Roman, or Byzantine coinage. Also include major general numismatic shows.

Key organizations and websites to search:
- coinshows.com (comprehensive coin show directory)
- money.org (ANA — American Numismatic Association events)
- pngdealers.org (PNG show schedule)
- nyinc.info (New York International Numismatic Convention)
- biddr.com (live auctions tied to shows)

CRITICAL RULES:
- ONLY return shows with FUTURE dates — never list past events
- Focus on shows within the next 30 days unless the user specifies a different timeframe
- When the user says "near me" or asks for nearby shows, only include shows within approximately 50 miles of their location
- For each show, find: name, dates, location, venue, website URL
- If the user mentions a location, prioritize shows near that area
- Note any special exhibits, notable dealers, or auction events at the show`

func (h *AgentHandler) getCoinSearchPrompt() string {
	prompt := h.settingsSvc.GetSetting(services.SettingCoinSearchPrompt)
	if prompt == "" {
		prompt = DefaultCoinSearchPrompt
	}
	return prompt
}

func (h *AgentHandler) getCoinShowsPrompt(userID uint) string {
	prompt := h.settingsSvc.GetSetting(services.SettingCoinShowsPrompt)
	if prompt == "" {
		prompt = DefaultCoinShowsPrompt
	}

	// Inject current date so the agent knows what "upcoming" means
	now := time.Now()
	deadline := now.AddDate(0, 0, 30)
	datePreamble := fmt.Sprintf("Today's date is %s. Unless the user specifies a different timeframe, only return shows between now and %s (next 30 days).\n\n",
		now.Format("January 2, 2006"), deadline.Format("January 2, 2006"))
	prompt = datePreamble + prompt

	if user, err := h.userRepo.FindByID(userID); err == nil && user.ZipCode != "" {
		prompt = fmt.Sprintf("The user's location ZIP code is %s. Use this to prioritize nearby coin shows, dealers, and events when relevant.\n\n%s", user.ZipCode, prompt)
	} else {
		prompt = fmt.Sprintf("The user has not set a ZIP code. If they ask about nearby coin shows or local events, ask them where they are located before searching.\n\n%s", prompt)
	}
	return prompt
}

// ChatStream handles a streaming conversation with the AI agent via SSE.
//
//	@Summary		Chat with coin search agent (streaming)
//	@Description	Send a message to the AI agent. Response is streamed as Server-Sent Events.
//	@Tags			Agent
//	@Accept			json
//	@Produce		text/event-stream
//	@Param			body	body		AgentChatRequest	true	"Chat message"
//	@Success		200		{object}	AgentChatResponse
//	@Failure		400		{object}	ErrorResponse
//	@Failure		503		{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/agent/chat [post]
func (h *AgentHandler) ChatStream(c *gin.Context) {
	logger := h.logger
	userID := c.GetUint("userId")

	var req AgentChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Message is required"})
		return
	}

	// Content moderation: validate message and history
	if err := h.guard.ValidateMessage(req.Message, userID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Your message could not be processed. Please keep questions related to coin collecting."})
		return
	}
	proxyHistory := make([]services.ChatMessageProxy, 0, len(req.History))
	for _, msg := range req.History {
		proxyHistory = append(proxyHistory, services.ChatMessageProxy{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}
	if err := h.guard.ValidateHistory(proxyHistory, userID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Conversation history could not be validated."})
		return
	}

	// Mint internal token for Python agent to call back into collection tools
	internalToken, err := h.tokenSvc.Mint(userID)
	if err != nil {
		logger.Error("agent", "failed to mint internal token: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Service error"})
		return
	}

	// Resolve LLM provider from explicit setting
	llmCfg, errMsg := h.resolveLLMConfig()
	if errMsg != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": errMsg})
		return
	}

	// Look up user zip code for location context
	var zipCode string
	if user, err := h.userRepo.FindByID(userID); err == nil {
		zipCode = user.ZipCode
	}

	// Fetch portfolio summary so the agent has it if the router sends to portfolio team
	var portfolio *services.PortfolioData
	if summary, err := h.repo.GetPortfolioSummary(userID); err == nil && summary.TotalCoins > 0 {
		portfolio = buildPortfolioData(summary)
	}

	proxyReq := services.AgentChatProxyRequest{
		LLM: llmCfg,
		User: services.UserContextProxy{
			UserID:  userID,
			ZipCode: zipCode,
		},
		Message:          req.Message,
		History:          proxyHistory,
		AppContext:       req.AppContext,
		CoinSearchPrompt: h.getCoinSearchPrompt(),
		CoinShowsPrompt:  h.getCoinShowsPrompt(userID),
		Portfolio:        portfolio,
		InternalToken:    internalToken,
		ToolsBaseURL:     h.toolsBaseURL,
	}

	if err := h.proxy.StreamChat(c.Request.Context(), c.Writer, proxyReq); err != nil {
		logger.Error("agent", "Chat stream proxy failed: %v", err)
		// Only send JSON error if headers haven't been sent yet
		if !c.Writer.Written() {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Agent service unavailable"})
		}
	}
}

type CommitCollectionProposalRequest struct {
	ProposalToken string `json:"proposalToken" binding:"required"`
	Confirm       bool   `json:"confirm"`
}

// CommitCollectionProposal commits a pending collection update proposal.
//
//	@Summary		Commit collection proposal
//	@Description	Commits a pending AI-proposed collection update after explicit user confirmation.
//	@Tags			Agent
//	@Accept			json
//	@Produce		json
//	@Param			proposalId	path		string					true	"Proposal ID"
//	@Param			body		body		CommitCollectionProposalRequest	true	"Commit confirmation"
//	@Success		200			{object}	services.CommitCollectionProposalResult
//	@Failure		400			{object}	ErrorResponse
//	@Failure		404			{object}	ErrorResponse
//	@Failure		409			{object}	ErrorResponse
//	@Failure		503			{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/agent/collection/proposals/{proposalId}/commit [post]
func (h *AgentHandler) CommitCollectionProposal(c *gin.Context) {
	if h.collectionSvc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Collection tools unavailable"})
		return
	}

	var req CommitCollectionProposalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "proposalToken and confirm are required"})
		return
	}
	if !req.Confirm {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Explicit confirmation is required"})
		return
	}

	userID := c.GetUint("userId")
	proposalID := c.Param("proposalId")
	result, err := h.collectionSvc.CommitProposal(userID, proposalID, req.ProposalToken, req.Confirm)
	if err != nil {
		switch {
		case repository.IsRecordNotFound(err):
			c.JSON(http.StatusNotFound, gin.H{"error": "Proposal not found"})
		case errors.Is(err, services.ErrProposalTokenInvalid):
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid proposal token"})
		case errors.Is(err, services.ErrProposalConfirmationReq):
			c.JSON(http.StatusBadRequest, gin.H{"error": "Explicit confirmation is required"})
		case errors.Is(err, services.ErrProposalStateConflict):
			c.JSON(http.StatusConflict, gin.H{"error": "Proposal is no longer pending"})
		default:
			h.logger.Error("agent", "commit proposal failed: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit proposal"})
		}
		return
	}

	c.JSON(http.StatusOK, result)
}

// CancelCollectionProposal cancels a pending collection update proposal.
//
//	@Summary		Cancel collection proposal
//	@Description	Cancels a pending AI-proposed collection update for the authenticated user.
//	@Tags			Agent
//	@Produce		json
//	@Param			proposalId	path		string	true	"Proposal ID"
//	@Success		200			{object}	services.CancelCollectionProposalResult
//	@Failure		404			{object}	ErrorResponse
//	@Failure		409			{object}	ErrorResponse
//	@Failure		503			{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/agent/collection/proposals/{proposalId}/cancel [post]
func (h *AgentHandler) CancelCollectionProposal(c *gin.Context) {
	if h.collectionSvc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Collection tools unavailable"})
		return
	}

	userID := c.GetUint("userId")
	proposalID := c.Param("proposalId")
	result, err := h.collectionSvc.CancelProposal(userID, proposalID)
	if err != nil {
		switch {
		case repository.IsRecordNotFound(err):
			c.JSON(http.StatusNotFound, gin.H{"error": "Proposal not found"})
		case errors.Is(err, services.ErrProposalStateConflict):
			c.JSON(http.StatusConflict, gin.H{"error": "Proposal is no longer pending"})
		default:
			h.logger.Error("agent", "cancel proposal failed: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to cancel proposal"})
		}
		return
	}

	c.JSON(http.StatusOK, result)
}

// AgentStatus returns the current AI provider configuration status.
//
//	@Summary		Get agent status
//	@Description	Returns whether an AI provider is selected for agent features.
//	@Tags			Agent
//	@Produce		json
//	@Success		200	{object}	object
//	@Security		BearerAuth
//	@Router			/agent/status [get]
func (h *AgentHandler) AgentStatus(c *gin.Context) {
	provider := h.settingsSvc.GetSetting(services.SettingAIProvider)
	configured := provider == "anthropic" || provider == "ollama"

	c.JSON(http.StatusOK, gin.H{
		"provider":   provider,
		"configured": configured,
	})
}

// ListModels returns the list of available Anthropic models.
//
//	@Summary		List available Anthropic models
//	@Description	Returns a curated list of Anthropic models suitable for the coin search agent.
//	@Tags			Agent
//	@Produce		json
//	@Success		200	{array}	object
//	@Security		BearerAuth
//	@Router			/agent/models [get]
func (h *AgentHandler) ListModels(c *gin.Context) {
	apiKey := h.settingsSvc.GetSetting(services.SettingAnthropicAPIKey)
	if apiKey == "" {
		// Return defaults if no API key
		c.JSON(http.StatusOK, getDefaultModels())
		return
	}

	// Try to fetch from Anthropic API
	req, err := http.NewRequest("GET", "https://api.anthropic.com/v1/models", nil)
	if err != nil {
		c.JSON(http.StatusOK, getDefaultModels())
		return
	}
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		c.JSON(http.StatusOK, getDefaultModels())
		return
	}
	defer resp.Body.Close()

	var result struct {
		Data []struct {
			ID          string `json:"id"`
			DisplayName string `json:"display_name"`
		} `json:"data"`
	}
	body, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(body, &result); err != nil || len(result.Data) == 0 {
		c.JSON(http.StatusOK, getDefaultModels())
		return
	}

	models := make([]map[string]string, 0, len(result.Data))
	for _, m := range result.Data {
		models = append(models, map[string]string{
			"id":   m.ID,
			"name": m.DisplayName,
		})
	}
	c.JSON(http.StatusOK, models)
}

func getDefaultModels() []map[string]string {
	return []map[string]string{
		{"id": "claude-sonnet-4-20250514", "name": "Claude Sonnet 4"},
		{"id": "claude-haiku-4-20250414", "name": "Claude Haiku 4"},
		{"id": "claude-opus-4-20250514", "name": "Claude Opus 4"},
	}
}

// GetCoinSearchPrompt returns the current coin search prompt.
//
//	@Summary		Get coin search prompt
//	@Tags			Agent
//	@Produce		json
//	@Success		200	{object}	object
//	@Security		BearerAuth
//	@Router			/agent/coin-search-prompt [get]
func (h *AgentHandler) GetCoinSearchPrompt(c *gin.Context) {
	prompt := h.settingsSvc.GetSetting(services.SettingCoinSearchPrompt)
	if prompt == "" {
		prompt = DefaultCoinSearchPrompt
	}
	c.JSON(http.StatusOK, gin.H{
		"prompt":  prompt,
		"default": DefaultCoinSearchPrompt,
	})
}

// GetCoinShowsPrompt returns the current coin shows prompt.
//
//	@Summary		Get coin shows prompt
//	@Tags			Agent
//	@Produce		json
//	@Success		200	{object}	object
//	@Security		BearerAuth
//	@Router			/agent/coin-shows-prompt [get]
func (h *AgentHandler) GetCoinShowsPrompt(c *gin.Context) {
	prompt := h.settingsSvc.GetSetting(services.SettingCoinShowsPrompt)
	if prompt == "" {
		prompt = DefaultCoinShowsPrompt
	}
	c.JSON(http.StatusOK, gin.H{
		"prompt":  prompt,
		"default": DefaultCoinShowsPrompt,
	})
}

// GetValuationPrompt returns the current valuation prompt.
//
//	@Summary		Get valuation prompt
//	@Tags			Agent
//	@Produce		json
//	@Success		200	{object}	object
//	@Security		BearerAuth
//	@Router			/agent/valuation-prompt [get]
func (h *AgentHandler) GetValuationPrompt(c *gin.Context) {
	prompt := h.settingsSvc.GetSetting(services.SettingValuationPrompt)
	if prompt == "" {
		prompt = services.DefaultValuationPrompt
	}
	c.JSON(http.StatusOK, gin.H{
		"prompt":  prompt,
		"default": services.DefaultValuationPrompt,
	})
}

// PortfolioSummary returns aggregated collection data for AI portfolio analysis.
//
//	@Summary		Get portfolio summary for AI analysis
//	@Tags			Agent
//	@Produce		json
//	@Success		200	{object}	object
//	@Security		BearerAuth
//	@Router			/agent/portfolio-summary [get]
func (h *AgentHandler) PortfolioSummary(c *gin.Context) {
	userID := c.GetUint("userId")

	summary, _ := h.repo.GetPortfolioSummary(userID)

	c.JSON(http.StatusOK, gin.H{
		"totalCoins":    summary.TotalCoins,
		"totalValue":    summary.TotalValue,
		"totalInvested": summary.TotalInvested,
		"categories":    summary.Categories,
		"materials":     summary.Materials,
		"eras":          summary.Eras,
		"rulers":        summary.Rulers,
		"topCoins":      summary.TopCoins,
		"missingFields": summary.MissingFields,
	})
}

// buildPortfolioData converts the repository PortfolioSummary into the proxy PortfolioData.
func buildPortfolioData(s *repository.PortfolioSummary) *services.PortfolioData {
	cats := make(map[string]int, len(s.Categories))
	for _, c := range s.Categories {
		cats[c.Category] = c.Count
	}
	mats := make(map[string]int, len(s.Materials))
	for _, m := range s.Materials {
		mats[m.Material] = m.Count
	}
	eras := make([]map[string]any, 0, len(s.Eras))
	for _, e := range s.Eras {
		eras = append(eras, map[string]any{"name": e.Era, "count": e.Count})
	}
	rulers := make([]map[string]any, 0, len(s.Rulers))
	for _, r := range s.Rulers {
		rulers = append(rulers, map[string]any{"name": r.Ruler, "count": r.Count})
	}
	coins := make([]services.PortfolioCoinProxy, 0, len(s.TopCoins))
	for _, tc := range s.TopCoins {
		var cv float64
		if tc.CurrentValue != nil {
			cv = *tc.CurrentValue
		}
		coins = append(coins, services.PortfolioCoinProxy{
			Name:         tc.Name,
			Category:     tc.Category,
			Era:          string(tc.Era),
			Ruler:        tc.Ruler,
			Grade:        tc.Grade,
			CurrentValue: cv,
		})
	}
	return &services.PortfolioData{
		TotalCoins:    int(s.TotalCoins),
		TotalValue:    s.TotalValue,
		TotalInvested: s.TotalInvested,
		Categories:    cats,
		Materials:     mats,
		Eras:          eras,
		Rulers:        rulers,
		TopCoins:      coins,
		MissingFields: s.MissingFields,
	}
}
