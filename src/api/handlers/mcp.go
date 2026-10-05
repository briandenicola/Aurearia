package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const mcpMaxRequestBodyBytes = 128 * 1024

type MCPHandler struct {
	service *services.MCPService
	logger  *services.Logger
}

func NewMCPHandler(service *services.MCPService, logger *services.Logger) *MCPHandler {
	return &MCPHandler{service: service, logger: logger}
}

type mcpSearchInput struct {
	Query string `json:"query,omitempty" jsonschema:"Optional text to match against owned coin fields"`
	Limit *int   `json:"limit,omitempty" jsonschema:"Maximum results to return"`
}

type mcpCoinIDInput struct {
	CoinID uint `json:"coinId" jsonschema:"Owned coin identifier"`
}

type mcpLimitInput struct {
	Limit *int `json:"limit,omitempty" jsonschema:"Maximum results to return"`
}

type mcpCoinListOutput struct {
	Coins []services.CollectionCoinSummary `json:"coins"`
}

type mcpCoinOutput struct {
	Coin *services.CollectionCoinSummary `json:"coin"`
}

type mcpStatsOutput struct {
	Stats *repository.CollectionStats `json:"stats"`
}

type mcpAuctionListInput struct {
	Status string `json:"status,omitempty" jsonschema:"Optional status: watching, bidding, won, lost, or passed"`
	Source string `json:"source,omitempty" jsonschema:"Optional source: numisbids or cng"`
	Search string `json:"search,omitempty" jsonschema:"Optional title, description, or auction house search"`
	Page   int    `json:"page,omitempty" jsonschema:"Page number starting at 1"`
	Limit  int    `json:"limit,omitempty" jsonschema:"Page size from 1 through 50"`
}

type mcpAuctionIDInput struct {
	AuctionLotID uint `json:"auctionLotId" jsonschema:"Owned auction lot identifier"`
}

type mcpAuctionOutput struct {
	Lot *services.MCPAuctionLotSummary `json:"lot"`
}

type mcpAuctionCountsInput struct {
	Source string `json:"source,omitempty" jsonschema:"Optional source: numisbids or cng"`
}

type mcpAuctionCountsOutput struct {
	Counts map[string]int64 `json:"counts"`
}

type mcpCopilotStartInput struct {
	ThreadID       string         `json:"threadId,omitempty" jsonschema:"Existing owner-scoped thread identifier; omit to create a thread"`
	Goal           string         `json:"goal" jsonschema:"Goal for the durable Coin Copilot run"`
	AppContext     map[string]any `json:"appContext,omitempty" jsonschema:"Optional bounded route context"`
	IdempotencyKey string         `json:"idempotencyKey" jsonschema:"Stable printable key used to deduplicate retries"`
}

type mcpCopilotRunIDInput struct {
	RunID string `json:"runId" jsonschema:"Owner-scoped Coin Copilot run identifier"`
}

type mcpCopilotResumeInput struct {
	RunID                     string `json:"runId" jsonschema:"Owner-scoped paused run identifier"`
	Answer                    string `json:"answer" jsonschema:"Answer to the pending clarification"`
	ExpectedCheckpointVersion int64  `json:"expectedCheckpointVersion" jsonschema:"Checkpoint version returned by the paused run"`
	IdempotencyKey            string `json:"idempotencyKey" jsonschema:"Stable printable key used to deduplicate retries"`
}

// Serve handles the stateless MCP Streamable HTTP endpoint.
//
//	@Summary		Aurearia Model Context Protocol endpoint
//	@Description	Stateless Streamable HTTP MCP endpoint. Requires an API key with read capability; Copilot tools additionally require copilot capability.
//	@Tags			MCP
//	@Accept			json
//	@Produce		json
//	@Param			X-API-Key	header		string	true	"API key"
//	@Param			body		body		object	true	"MCP JSON-RPC request"
//	@Success		200			{object}	object
//	@Failure		400			{object}	ErrorResponse
//	@Failure		401			{object}	ErrorResponse
//	@Failure		403			{object}	ErrorResponse
//	@Failure		413			{object}	ErrorResponse
//	@Failure		503			{object}	ErrorResponse
//	@Security		ApiKeyAuth
//	@Router			/mcp [post]
func (h *MCPHandler) Serve(c *gin.Context) {
	userID := c.GetUint("userId")
	capabilities, ok := c.Get("apiKeyCapabilities")
	capabilityString, capabilityOK := capabilities.(string)
	if userID == 0 || !ok || !capabilityOK {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	server := h.newServer(userID, capabilityString)
	transport := mcp.NewStreamableHTTPHandler(
		func(*http.Request) *mcp.Server { return server },
		&mcp.StreamableHTTPOptions{
			Stateless:                    true,
			JSONResponse:                 true,
			MaxRequestBodyBytes:          mcpMaxRequestBodyBytes,
			PropagateRequestCancellation: true,
		},
	)
	transport.ServeHTTP(c.Writer, c.Request)
}

func (h *MCPHandler) newServer(userID uint, capabilities string) *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "aurearia",
		Version: "1.0.0",
	}, nil)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "search_collection",
		Description: "Search coins owned by the authenticated user. Read-only.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input mcpSearchInput) (*mcp.CallToolResult, mcpCoinListOutput, error) {
		coins, err := h.service.SearchCollection(userID, input.Query, input.Limit)
		return nil, mcpCoinListOutput{Coins: coins}, h.safeToolError("search_collection", err)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_coin",
		Description: "Get one coin owned by the authenticated user. Read-only.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input mcpCoinIDInput) (*mcp.CallToolResult, mcpCoinOutput, error) {
		coin, err := h.service.GetCoin(userID, input.CoinID)
		return nil, mcpCoinOutput{Coin: coin}, h.safeToolError("get_coin", err)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_wishlist",
		Description: "List wishlist coins owned by the authenticated user with optional search. Read-only.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input mcpSearchInput) (*mcp.CallToolResult, mcpCoinListOutput, error) {
		coins, err := h.service.ListWishlist(userID, input.Query, input.Limit)
		return nil, mcpCoinListOutput{Coins: coins}, h.safeToolError("list_wishlist", err)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "collection_stats",
		Description: "Read aggregate collection, wishlist, sold, distribution, and value statistics.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input struct{}) (*mcp.CallToolResult, mcpStatsOutput, error) {
		stats, err := h.service.CollectionStats(userID)
		return nil, mcpStatsOutput{Stats: stats}, h.safeToolError("collection_stats", err)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "top_coins_by_value",
		Description: "List the authenticated user's highest-value active collection coins. Read-only.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input mcpLimitInput) (*mcp.CallToolResult, mcpCoinListOutput, error) {
		coins, err := h.service.TopCoinsByValue(userID, input.Limit)
		return nil, mcpCoinListOutput{Coins: coins}, h.safeToolError("top_coins_by_value", err)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_auction_lots",
		Description: "List auction lots owned by the authenticated user with bounded filters and pagination. Read-only.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input mcpAuctionListInput) (*mcp.CallToolResult, *services.MCPAuctionListResult, error) {
		result, err := h.service.ListAuctionLots(userID, services.MCPAuctionListInput{
			Status: input.Status, Source: input.Source, Search: input.Search,
			Page: input.Page, Limit: input.Limit,
		})
		return nil, result, h.safeToolError("list_auction_lots", err)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_auction_lot",
		Description: "Get one auction lot owned by the authenticated user. Read-only.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input mcpAuctionIDInput) (*mcp.CallToolResult, mcpAuctionOutput, error) {
		lot, err := h.service.GetAuctionLot(userID, input.AuctionLotID)
		return nil, mcpAuctionOutput{Lot: lot}, h.safeToolError("get_auction_lot", err)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "auction_counts",
		Description: "Read per-status counts for the authenticated user's auction lots.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input mcpAuctionCountsInput) (*mcp.CallToolResult, mcpAuctionCountsOutput, error) {
		counts, err := h.service.AuctionCounts(userID, input.Source)
		return nil, mcpAuctionCountsOutput{Counts: counts}, h.safeToolError("auction_counts", err)
	})

	if models.HasAPICapability(capabilities, "copilot") {
		h.addCopilotTools(server, userID)
	}
	return server
}

func (h *MCPHandler) addCopilotTools(server *mcp.Server, userID uint) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "start_copilot_run",
		Description: "Start or idempotently replay a bounded durable Coin Copilot run.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input mcpCopilotStartInput) (*mcp.CallToolResult, copilotRunEnvelope, error) {
		run, reused, err := h.service.StartCopilot(userID, services.CoinCopilotStartInput{
			ThreadID: input.ThreadID, Goal: input.Goal, AppContext: input.AppContext,
			IdempotencyKey: input.IdempotencyKey,
		})
		if err != nil {
			return nil, copilotRunEnvelope{}, h.safeToolError("start_copilot_run", err)
		}
		return nil, copilotRunEnvelope{Run: toCopilotRunDTO(run), Reused: reused}, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_copilot_run",
		Description: "Read the current state and final answer of an owner-scoped Coin Copilot run.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input mcpCopilotRunIDInput) (*mcp.CallToolResult, copilotRunEnvelope, error) {
		run, err := h.service.GetCopilotRun(userID, input.RunID)
		if err != nil {
			return nil, copilotRunEnvelope{}, h.safeToolError("get_copilot_run", err)
		}
		return nil, copilotRunEnvelope{Run: toCopilotRunDTO(run)}, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "resume_copilot_run",
		Description: "Resume a paused owner-scoped Coin Copilot run from its expected checkpoint.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input mcpCopilotResumeInput) (*mcp.CallToolResult, copilotRunEnvelope, error) {
		run, reused, err := h.service.ResumeCopilot(userID, input.RunID, services.CoinCopilotResumeInput{
			Answer: input.Answer, ExpectedCheckpointVersion: input.ExpectedCheckpointVersion,
			IdempotencyKey: input.IdempotencyKey,
		})
		if err != nil {
			return nil, copilotRunEnvelope{}, h.safeToolError("resume_copilot_run", err)
		}
		return nil, copilotRunEnvelope{Run: toCopilotRunDTO(run), Reused: reused}, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "cancel_copilot_run",
		Description: "Request cancellation of a queued or running owner-scoped Coin Copilot run.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input mcpCopilotRunIDInput) (*mcp.CallToolResult, copilotRunEnvelope, error) {
		run, _, err := h.service.CancelCopilot(userID, input.RunID)
		if err != nil {
			return nil, copilotRunEnvelope{}, h.safeToolError("cancel_copilot_run", err)
		}
		return nil, copilotRunEnvelope{Run: toCopilotRunDTO(run)}, nil
	})
}

func (h *MCPHandler) safeToolError(tool string, err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, services.ErrCoinNotFound),
		errors.Is(err, services.ErrCopilotNotFound),
		repository.IsRecordNotFound(err):
		return errors.New("resource not found")
	case errors.Is(err, services.ErrMCPInvalidRequest),
		errors.Is(err, services.ErrCopilotInvalidRequest):
		return errors.New("invalid tool input")
	case errors.Is(err, services.ErrCopilotDisabled),
		errors.Is(err, services.ErrCopilotUnavailable):
		return errors.New("Coin Copilot is unavailable")
	case errors.Is(err, services.ErrCopilotIdempotencyConflict),
		errors.Is(err, services.ErrCopilotCapacity),
		errors.Is(err, services.ErrCopilotQueueFull),
		errors.Is(err, services.ErrCopilotStateConflict),
		errors.Is(err, services.ErrCopilotThreadHasActiveRun):
		return errors.New("Coin Copilot request conflicts with current state")
	default:
		if h.logger != nil {
			h.logger.Error("mcp", "%s failed: %v", tool, err)
		}
		return errors.New("tool request failed")
	}
}
