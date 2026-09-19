package main

import (
	"github.com/briandenicola/ancient-coins-api/handlers"
	"github.com/briandenicola/ancient-coins-api/middleware"
	"github.com/gin-gonic/gin"
)

// registerInternalToolRoutes registers the callback surface used by the Python
// agent service. It is authenticated by internal service token, never by a
// user credential.
func registerInternalToolRoutes(r *gin.Engine, d *appDeps) {
	internal := r.Group("/api/internal/tools")
	internal.Use(middleware.InternalTokenRequired(d.internalTokenSvc))
	{
		internalToolsHandler := handlers.NewInternalToolsHandler(d.collectionSvc, d.logger)
		internal.POST("/search_my_collection", internalToolsHandler.SearchMyCollection)
		internal.POST("/get_coin", internalToolsHandler.GetCoin)
		internal.POST("/collection_summary", internalToolsHandler.CollectionSummary)
		internal.POST("/top_coins_by_value", internalToolsHandler.TopCoinsByValue)
		internal.POST("/propose_update", internalToolsHandler.ProposeUpdate)
		internal.POST("/commit_update", internalToolsHandler.CommitUpdate)
	}

	copilotHandler := handlers.NewCoinCopilotInternalToolsHandler(
		d.collectionSvc, d.coinCopilotSvc, d.logger, d.deepAnalysisHandoffSvc,
	)
	copilot := r.Group("/api/internal/copilot/tools")
	{
		copilot.POST("/search_my_collection", middleware.CoinCopilotExecutionTokenRequired(d.internalTokenSvc, "search_my_collection"), copilotHandler.SearchMyCollection)
		copilot.POST("/get_coin", middleware.CoinCopilotExecutionTokenRequired(d.internalTokenSvc, "get_coin"), copilotHandler.GetCoin)
		copilot.POST("/collection_summary", middleware.CoinCopilotExecutionTokenRequired(d.internalTokenSvc, "collection_summary"), copilotHandler.CollectionSummary)
		copilot.POST("/top_coins_by_value", middleware.CoinCopilotExecutionTokenRequired(d.internalTokenSvc, "top_coins_by_value"), copilotHandler.TopCoinsByValue)
		copilot.POST("/deep_analysis_handoff", middleware.CoinCopilotExecutionTokenRequired(d.internalTokenSvc, "deep_analysis_handoff"), copilotHandler.DeepAnalysisHandoff)
	}

	// Deep identification provider-tool boundary (Phase 6, T051): job-scoped
	// token auth (distinct from the userID-only token above), shared route
	// prefix per contracts/agent-internal-contract.md §7.
	internalDeepProviderTools := r.Group("/api/internal/tools")
	internalDeepProviderTools.Use(middleware.InternalJobTokenRequired(d.internalTokenSvc))
	{
		deepProviderToolsHandler := handlers.NewDeepProviderToolsHandler(
			d.numistaClient, d.deepNomismaClient, d.deepOCREClient, d.deepOCRECache, d.settingsSvc, d.deepProviderBudgets, d.logger,
		)
		internalDeepProviderTools.POST("/numista_search", deepProviderToolsHandler.NumistaSearch)
		internalDeepProviderTools.POST("/numista_detail", deepProviderToolsHandler.NumistaDetail)
		internalDeepProviderTools.POST("/nomisma_search", deepProviderToolsHandler.NomismaSearch)
		internalDeepProviderTools.POST("/ocre_search", deepProviderToolsHandler.OCRESearch)
	}
}
