package main

import (
	"github.com/briandenicola/ancient-coins-api/database"
	"github.com/briandenicola/ancient-coins-api/handlers"
	"github.com/briandenicola/ancient-coins-api/middleware"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
)

// registerAdminRoutes registers the /api/admin endpoints. These reject API-key
// auth outright and additionally require the admin role.
func registerAdminRoutes(api *gin.RouterGroup, d *appDeps) {
	admin := api.Group("/admin")
	admin.Use(middleware.AuthRequiredWithSecurity(d.cfg.JWTSecret, d.apiKeyAuth, d.securitySvc))
	admin.Use(middleware.RejectAPIKeyAuth())
	admin.Use(handlers.AdminRequired())
	{
		adminRepo := repository.NewAdminRepository(database.DB)
		adminRecoverySvc := services.NewAdminRecoveryService(adminRepo, d.securitySvc)
		adminHandler := handlers.NewAdminHandler(d.cfg.UploadDir, adminRepo, adminRecoverySvc, d.agentProxy, d.settingsSvc, d.logger)
		admin.GET("/users", adminHandler.ListUsers)
		admin.DELETE("/users/:id", adminHandler.DeleteUser)
		admin.POST("/users/:id/reset-password", adminHandler.ResetPassword)
		admin.PUT("/users/:id/role", adminHandler.UpdateUserRole)
		admin.GET("/settings", adminHandler.GetSettings)
		admin.GET("/settings/defaults", adminHandler.GetSettingDefaults)
		admin.PUT("/settings", adminHandler.UpdateSettings)
		admin.GET("/logs", adminHandler.GetLogs)
		adminNumistaHandler := handlers.NewAdminNumistaHandler(d.numistaTelemetry, d.settingsSvc)
		admin.GET("/numista/health", adminNumistaHandler.Health)
		adminOCREHandler := handlers.NewAdminOCREHandler(d.settingsSvc, d.deepIdentificationRepo)
		admin.GET("/deep-identification/ocre/health", adminOCREHandler.Health)
		admin.GET("/test-anthropic", adminHandler.TestAnthropicConnection)
		admin.GET("/test-searxng", adminHandler.TestSearXNGConnection)

		oidcHandler := handlers.NewOIDCHandler(d.oidcSvc)
		admin.GET("/oidc/providers", oidcHandler.ListAdminProviders)
		admin.POST("/oidc/providers", oidcHandler.CreateAdminProvider)
		admin.PUT("/oidc/providers/:providerId", oidcHandler.UpdateAdminProvider)
		admin.DELETE("/oidc/providers/:providerId", oidcHandler.DeleteAdminProvider)
		admin.POST("/oidc/providers/:providerId/test", oidcHandler.TestAdminProvider)

		securityAdminHandler := handlers.NewSecurityAdminHandler(d.securitySvc, d.settingsSvc, handlers.SecurityExposureConfig{
			PublicAppURL:             d.settingsSvc.GetSetting(services.SettingPublicAppURL),
			WebAuthnOrigin:           d.cfg.WebAuthnOrigin,
			CORSOrigins:              d.cfg.AllowedOrigins(),
			TrustedProxiesConfigured: d.cfg.TrustedProxies != "",
			AgentInternalTokenSet:    d.cfg.AgentInternalServiceToken != "",
			RegistrationMode:         d.settingsSvc.GetSetting(services.SettingRegistrationMode),
			BackupStatus:             d.settingsSvc.GetSetting(services.SettingBackupStatus),
		})
		admin.GET("/security/summary", securityAdminHandler.SecuritySummary)
		admin.GET("/security/events", securityAdminHandler.SecurityEvents)
		admin.GET("/security/ip-rules", securityAdminHandler.ListIPRules)
		admin.POST("/security/ip-rules", securityAdminHandler.CreateIPRule)
		admin.DELETE("/security/ip-rules/:id", securityAdminHandler.DeleteIPRule)
		admin.POST("/users/:id/unlock", securityAdminHandler.UnlockUser)
		admin.GET("/security/exposure-check", securityAdminHandler.ExposureCheck)

		// Catalog registry management (shared handler from protected scope)
		catalogRegistryRepo := repository.NewCatalogRegistryRepository(database.DB)
		catalogRegistrySvc := services.NewCatalogRegistryService(catalogRegistryRepo)
		catalogRegistryHandler := handlers.NewCatalogRegistryHandler(catalogRegistrySvc)
		admin.POST("/catalogs", catalogRegistryHandler.Create)
		admin.PUT("/catalogs/:id", catalogRegistryHandler.Update)
		admin.DELETE("/catalogs/:id", catalogRegistryHandler.Delete)

		// Mint location management
		mintLocationRepo := repository.NewMintLocationRepository(database.DB)
		nomismaClient := services.NewHTTPNomismaClient()
		nomismaCache := services.NewNomismaCache()
		mintLocationSvc := services.NewMintLocationService(mintLocationRepo).WithNomisma(nomismaClient, nomismaCache)
		mintLocationHandler := handlers.NewMintLocationHandler(mintLocationSvc)
		admin.POST("/mint-locations", mintLocationHandler.Create)
		admin.PUT("/mint-locations/:id", mintLocationHandler.Update)
		admin.DELETE("/mint-locations/:id", mintLocationHandler.Delete)
		admin.GET("/mint-locations/:id/nomisma/search", mintLocationHandler.SearchNomisma)
		admin.POST("/mint-locations/:id/nomisma", mintLocationHandler.LinkNomisma)
		admin.DELETE("/mint-locations/:id/nomisma", mintLocationHandler.UnlinkNomisma)

		// Availability check run history and manual trigger (reuse outer scope services)
		adminAvailHandler := handlers.NewAvailabilityHandler(nil, d.availScheduler, d.availRepo, nil).WithCycleRepo(d.availCycleRepo)
		admin.GET("/availability-runs", adminAvailHandler.ListRuns)
		admin.GET("/availability-runs/:id", adminAvailHandler.GetRunDetail)
		admin.POST("/availability/run", adminAvailHandler.TriggerRun)
		admin.GET("/availability-cycles", adminAvailHandler.ListCycles)
		admin.GET("/availability-cycles/:id", adminAvailHandler.GetCycleDetail)

		// Valuation run history and manual trigger
		valAdminHandler := handlers.NewValuationAdminHandler(d.valRepo, d.valSvc, d.logger)
		admin.GET("/valuation-runs", valAdminHandler.ListRuns)
		admin.GET("/valuation-runs/:id", valAdminHandler.GetRunDetail)
		admin.POST("/valuation-runs/trigger", valAdminHandler.TriggerValuation)
		admin.POST("/valuation-runs/:id/cancel", valAdminHandler.CancelValuation)

		// Auction ending run history and manual trigger
		auctionEndingAdminHandler := handlers.NewAuctionEndingAdminHandler(d.auctionEndingRepo, d.auctionEndingScheduler, d.logger)
		admin.GET("/auction-ending-runs", auctionEndingAdminHandler.ListRuns)
		admin.GET("/auction-ending-runs/:id", auctionEndingAdminHandler.GetRun)
		admin.POST("/auction-ending/run", auctionEndingAdminHandler.TriggerRun)
		auctionWatchBidDigestAdminHandler := handlers.NewAuctionWatchBidDigestAdminHandler(d.auctionWatchBidDigestScheduler, d.auctionWatchBidDigestRepo)
		admin.GET("/auction-watch-bid-digest-runs", auctionWatchBidDigestAdminHandler.ListRuns)
		admin.GET("/auction-watch-bid-digest/status", auctionWatchBidDigestAdminHandler.GetStatus)
		admin.POST("/auction-watch-bid-digest/run", auctionWatchBidDigestAdminHandler.RunNow)
		auctionAlertAdminHandler := handlers.NewAuctionAlertAdminHandler(d.auctionAlertScheduler, d.auctionAlertRunRepo)
		admin.GET("/auction-alert-runs", auctionAlertAdminHandler.ListRuns)
		admin.GET("/auction-alerts/status", auctionAlertAdminHandler.GetStatus)
		admin.POST("/auction-alerts/run", auctionAlertAdminHandler.RunNow)

		// Coin of the Day manual trigger
		coinOfDayAdminHandler := handlers.NewCoinOfDayAdminHandler(d.coinOfDayScheduler, d.logger)
		admin.POST("/coin-of-day/run", coinOfDayAdminHandler.TriggerRun)
		admin.GET("/coin-of-day-runs", coinOfDayAdminHandler.ListRuns)
		admin.GET("/coin-of-day-runs/:id", coinOfDayAdminHandler.GetRun)

		// Aggregate health metrics
		adminHealthHandler := handlers.NewAdminHealthHandler(d.healthSvc, d.healthScheduler, d.logger)
		admin.GET("/health/summary", adminHealthHandler.Summary)
		admin.POST("/collection-health-snapshots/run", adminHealthHandler.TriggerSnapshotRun)
		admin.GET("/collection-health-snapshot-runs", adminHealthHandler.ListSnapshotRuns)
		admin.GET("/collection-health/status", adminHealthHandler.GetSnapshotStatus)

		// Deep Identification aggregate operational metrics
		adminDeepIdentificationHandler := handlers.NewAdminDeepIdentificationHandler(d.deepIdentificationSvc)
		admin.GET("/deep-identification/observability", adminDeepIdentificationHandler.Observability)

		// API key rotation notification trigger
		apiKeyAdminHandler := handlers.NewApiKeyAdminHandler(d.apiKeyRepo, d.notifSvc, d.logger)
		admin.POST("/api-keys/notify-rotation", apiKeyAdminHandler.NotifyRotationRequired)

		// Auction ending debug endpoint
		auctionDebugHandler := handlers.NewAuctionEndingDebugHandler(d.auctionLotRepo)
		admin.GET("/auction-ending/debug", auctionDebugHandler.DebugGetAuctionEndingInfo)
	}
}
