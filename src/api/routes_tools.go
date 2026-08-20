package main

import (
	"github.com/briandenicola/ancient-coins-api/handlers"
	"github.com/briandenicola/ancient-coins-api/middleware"
	"github.com/gin-gonic/gin"
)

// registerExternalToolRoutes registers the external tool server surface
// (/api/v1/tools) consumed by OpenWebUI/LibreChat-style clients holding
// scoped API keys.
func registerExternalToolRoutes(api *gin.RouterGroup, d *appDeps) {
	toolsSpec := api.Group("/v1/tools")
	toolsSpec.Use(middleware.ExternalToolServerEnabled(d.settingsSvc))
	{
		openapiHandler := handlers.NewExternalToolsOpenAPIHandler()
		toolsSpec.GET("/openapi.json", openapiHandler.GetOpenAPISpec)
	}

	// Authenticated tool endpoints (auth + rate limit)
	v1Tools := api.Group("/v1/tools")
	v1Tools.Use(middleware.ExternalToolServerEnabled(d.settingsSvc))
	v1Tools.Use(middleware.AuthRequiredWithSecurity(d.cfg.JWTSecret, d.apiKeyAuth, d.securitySvc))
	v1Tools.Use(d.externalToolsRateLimit)
	{
		externalToolsHandler := handlers.NewExternalToolsHandler(d.collectionSvc)

		// Read tools (require 'read' capability)
		readTools := v1Tools.Group("")
		readTools.Use(middleware.RequireCapability("read"))
		{
			readTools.POST("/search_my_collection", externalToolsHandler.SearchMyCollection)
			readTools.POST("/get_coin", externalToolsHandler.GetCoin)
			readTools.POST("/collection_summary", externalToolsHandler.CollectionSummary)
			readTools.POST("/top_coins_by_value", externalToolsHandler.TopCoinsByValue)
		}

		// Write tools (require 'write' capability)
		writeTools := v1Tools.Group("")
		writeTools.Use(middleware.RequireCapability("write"))
		{
			writeTools.POST("/propose_update", externalToolsHandler.ProposeUpdate)
			writeTools.POST("/commit_update", externalToolsHandler.CommitUpdate)
		}
	}
}
