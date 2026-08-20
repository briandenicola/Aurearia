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
