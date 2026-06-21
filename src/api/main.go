package main

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/briandenicola/ancient-coins-api/config"
	"github.com/briandenicola/ancient-coins-api/database"
	_ "github.com/briandenicola/ancient-coins-api/docs"
	"github.com/briandenicola/ancient-coins-api/handlers"
	"github.com/briandenicola/ancient-coins-api/middleware"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

type SchedulerRegistry struct {
	schedulers []services.Scheduler
}

func (r *SchedulerRegistry) Register(scheduler services.Scheduler) {
	r.schedulers = append(r.schedulers, scheduler)
}

func (r *SchedulerRegistry) StartAll() {
	for _, scheduler := range r.schedulers {
		go scheduler.Start()
	}
}

//	@title						Ancient Coins API
//	@version					1.0
//	@description				REST API for managing an ancient coin collection. Supports coin CRUD, image uploads, AI-powered analysis via Ollama, user management, and admin features.
//	@BasePath					/api
//	@securityDefinitions.apikey	BearerAuth
//	@in							header
//	@name						Authorization
//	@description				Enter your JWT token with the Bearer prefix, e.g. "Bearer eyJhbGci..."

//	@securityDefinitions.apikey	ApiKeyAuth
//	@in							header
//	@name						X-API-Key
//	@description				Enter your API key, e.g. "ak_a1b2c3d4..."

func main() {
	cfg := config.Load()

	database.Connect(cfg.DBPath)

	// Create logger and settings service
	logger := services.NewLogger(1000)
	settingsRepo := repository.NewSettingsRepository(database.DB)
	settingsSvc := services.NewSettingsService(settingsRepo)
	settingsSvc.SyncLogLevel(logger)

	// Create internal token service for Python agent callbacks
	internalTokenSvc := services.NewInternalTokenService(cfg.JWTSecret)

	logger.Info("startup", "Application starting")
	logger.Info("startup", "Database connected: %s", cfg.DBPath)

	// Ensure upload directory exists
	if err := os.MkdirAll(cfg.UploadDir, 0755); err != nil {
		log.Fatalf("Failed to create upload directory: %v", err)
	}
	logger.Debug("startup", "Upload directory: %s", cfg.UploadDir)

	r := gin.Default()
	if err := r.SetTrustedProxies(cfg.TrustedProxyList()); err != nil {
		log.Fatalf("Failed to configure trusted proxies: %v", err)
	}
	r.MaxMultipartMemory = middleware.DefaultMultipartMemoryBytes
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.ResolvedClientIP())

	// CORS middleware — restrict to configured origins
	allowedOrigins := cfg.AllowedOrigins()
	r.Use(func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		allowed := false
		for _, o := range allowedOrigins {
			if o == origin {
				allowed = true
				break
			}
		}
		if allowed {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Credentials", "true")
		}
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// Serve Vue SPA from wwwroot
	wwwroot := filepath.Join(".", "wwwroot")
	if _, err := os.Stat(wwwroot); err == nil {
		r.Static("/assets", filepath.Join(wwwroot, "assets"))
		r.StaticFile("/coin-logo.jpg", filepath.Join(wwwroot, "coin-logo.jpg"))
		r.StaticFile("/manifest.webmanifest", filepath.Join(wwwroot, "manifest.webmanifest"))
		r.StaticFile("/sw.js", filepath.Join(wwwroot, "sw.js"))
		r.StaticFile("/registerSW.js", filepath.Join(wwwroot, "registerSW.js"))

		// SPA fallback
		r.NoRoute(func(c *gin.Context) {
			// Don't serve index.html for API routes
			if len(c.Request.URL.Path) >= 4 && c.Request.URL.Path[:4] == "/api" {
				c.JSON(404, gin.H{"error": "Not found"})
				return
			}
			if len(c.Request.URL.Path) >= 8 && c.Request.URL.Path[:8] == "/uploads" {
				c.JSON(404, gin.H{"error": "Not found"})
				return
			}
			c.File(filepath.Join(wwwroot, "index.html"))
		})
	}

	// Health check (no auth, for container orchestration)
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Swagger docs
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Auth routes (public) — rate limited to prevent brute force
	authRepo := repository.NewAuthRepository(database.DB)
	securityRepo := repository.NewSecurityRepository(database.DB)
	securitySvc := services.NewSecurityService(securityRepo)
	authSvc := services.NewAuthService(authRepo, cfg.JWTSecret).WithSettings(settingsSvc).WithSecurity(securitySvc)
	authHandler := handlers.NewAuthHandler(cfg.JWTSecret, authRepo, authSvc)
	webauthnRepo := repository.NewWebAuthnRepository(database.DB)
	webauthnHandler, err := handlers.NewWebAuthnHandler(cfg.WebAuthnID, cfg.WebAuthnOrigin, authHandler, webauthnRepo, logger)
	if err != nil {
		log.Fatalf("Failed to initialize WebAuthn: %v", err)
	}
	apiKeyRepo := repository.NewApiKeyRepository(database.DB)
	apiKeyAuth := apiKeyRepo // implements middleware.ApiKeyAuthenticator
	imageRepo := repository.NewImageRepository(database.DB)
	imageSvc := services.NewImageService(imageRepo, cfg.UploadDir)
	imageHandler := handlers.NewImageHandler(cfg.UploadDir, imageRepo, imageSvc, logger)

	authRateLimit := middleware.RateLimit(10, 1*time.Minute)
	apiRateLimit := middleware.AuthenticatedRateLimit(600, 1*time.Minute) // Authenticated browsing
	writeRateLimit := middleware.AuthenticatedRateLimit(30, 1*time.Minute) // Write operations

	r.Use(middleware.IPDenyRules(securitySvc))
	r.GET("/uploads/*filepath", middleware.AuthRequiredWithSecurity(cfg.JWTSecret, apiKeyAuth, securitySvc), apiRateLimit, imageHandler.ServeUpload)

	api := r.Group("/api")
	api.Use(middleware.RequestBodyLimit(middleware.DefaultRequestBodyLimitBytes))
	{
		api.GET("/auth/setup", authHandler.NeedsSetup)
		api.POST("/auth/register", authRateLimit, authHandler.Register)
		api.POST("/auth/login", authRateLimit, authHandler.Login)
		api.POST("/auth/refresh", authRateLimit, authHandler.Refresh)

		// WebAuthn public routes (login ceremony)
		api.POST("/auth/webauthn/login/begin", authRateLimit, webauthnHandler.LoginBegin)
		api.POST("/auth/webauthn/login/finish", authRateLimit, webauthnHandler.LoginFinish)
		api.GET("/auth/webauthn/check", webauthnHandler.CheckCredentials)

		// Public showcase route (no auth)
		publicShowcaseRepo := repository.NewShowcaseRepository(database.DB)
		publicShowcaseHandler := handlers.NewShowcaseHandler(publicShowcaseRepo)
		api.GET("/showcase/:slug", publicShowcaseHandler.GetPublicShowcase)
		api.GET("/showcase/:slug/uploads/*filepath", imageHandler.ServePublicShowcaseUpload)
	}

	// Protected routes
	agentProxy := services.NewAgentProxy(cfg.AgentServiceURL, cfg.AgentInternalServiceToken, logger)
	availRepo := repository.NewAvailabilityRepository(database.DB)
	coinRepo := repository.NewCoinRepository(database.DB)
	socialRepo := repository.NewSocialRepository(database.DB)
	notifRepo := repository.NewNotificationRepository(database.DB)
	valRepo := repository.NewValuationRepository(database.DB)
	auctionEndingRepo := repository.NewAuctionEndingRepository(database.DB)
	userRepoForVal := repository.NewUserRepository(database.DB)
	auctionLotRepo := repository.NewAuctionLotRepository(database.DB)
	pushoverSvc := services.NewPushoverService(settingsSvc, logger)
	notifSvc := services.NewNotificationService(notifRepo, socialRepo, userRepoForVal, pushoverSvc, logger)
	availSvc := services.NewAvailabilityService(coinRepo, availRepo, agentProxy, notifSvc, pushoverSvc, userRepoForVal, settingsSvc, logger)
	valSvc := services.NewValuationService(coinRepo, valRepo, agentProxy, userRepoForVal, pushoverSvc, settingsSvc, logger)
	healthRepo := repository.NewHealthRepository(database.DB)
	healthSvc := services.NewHealthService(healthRepo, logger)

	// Create schedulers before routes so they can be passed to admin handlers
	availScheduler := services.NewAvailabilityScheduler(availSvc, coinRepo, availRepo, settingsSvc, logger)
	valScheduler := services.NewValuationScheduler(valSvc, coinRepo, valRepo, settingsSvc, logger)
	auctionEndingScheduler := services.NewAuctionEndingScheduler(auctionLotRepo, auctionEndingRepo, userRepoForVal, pushoverSvc, settingsSvc, logger)
	healthScheduler := services.NewCollectionHealthScheduler(healthSvc, settingsSvc, logger)
	featuredCoinRepo := repository.NewFeaturedCoinRepository(database.DB)
	coinOfDayScheduler := services.NewCoinOfDayScheduler(featuredCoinRepo, userRepoForVal, coinRepo, notifSvc, settingsSvc, logger)
	schedulerRegistry := &SchedulerRegistry{}
	schedulerRegistry.Register(availScheduler)
	schedulerRegistry.Register(valScheduler)
	schedulerRegistry.Register(auctionEndingScheduler)
	schedulerRegistry.Register(healthScheduler)

	// Create shared repositories for cross-group access
	journalRepo := repository.NewJournalRepository(database.DB)
	collectionProposalRepo := repository.NewCollectionUpdateRepository(database.DB)
	noteRepo := repository.NewNoteRepository(database.DB)
	collectionSvc := services.NewCollectionToolsService(coinRepo, collectionProposalRepo)

	protected := api.Group("")
	protected.Use(middleware.AuthRequiredWithSecurity(cfg.JWTSecret, apiKeyAuth, securitySvc))
	protected.Use(apiRateLimit)
	{
		coinReferenceRepo := repository.NewCoinReferenceRepository(database.DB)
		storageLocationRepo := repository.NewStorageLocationRepository(database.DB)
		mintLocationRepo := repository.NewMintLocationRepository(database.DB)
		catalogRegistryRepo := repository.NewCatalogRegistryRepository(database.DB)
		intakeDraftRepo := repository.NewCoinIntakeDraftRepository(database.DB)
		coinReferenceSvc := services.NewCoinReferenceService(coinReferenceRepo, catalogRegistryRepo)
		referenceMigrationSvc := services.NewReferenceMigrationService(database.DB, coinReferenceRepo, catalogRegistryRepo, journalRepo)
		catalogRegistrySvc := services.NewCatalogRegistryService(catalogRegistryRepo)
		catalogRegistryHandler := handlers.NewCatalogRegistryHandler(catalogRegistrySvc)
		coinSvc := services.NewCoinService(coinRepo, notifSvc).WithReferenceSupport(coinReferenceRepo, coinReferenceSvc).WithStorageLocationSupport(storageLocationRepo).WithCatalogRegistrySupport(catalogRegistryRepo).WithSettingsSupport(settingsSvc)
		coinHandler := handlers.NewCoinHandler(coinRepo, coinSvc, logger)
		coinReferenceHandler := handlers.NewCoinReferenceHandler(coinReferenceRepo, coinReferenceSvc, referenceMigrationSvc)
		coinIntakeSvc := services.NewCoinIntakeService(intakeDraftRepo, coinRepo, agentProxy, settingsSvc)
		coinIntakeHandler := handlers.NewCoinIntakeHandler(coinIntakeSvc, logger)
		coinLookupSvc := services.NewCoinLookupService(agentProxy, settingsSvc, logger)
		coinLookupHandler := handlers.NewCoinLookupHandler(coinLookupSvc, logger)
		protected.GET("/coins", coinHandler.List)
		protected.GET("/coins/:id", coinHandler.Get)
		protected.POST("/coins", coinHandler.Create)
		protected.POST("/coins/intake/draft", writeRateLimit, coinIntakeHandler.CreateDraft)
		protected.POST("/coins/intake/commit", writeRateLimit, coinIntakeHandler.CommitDraft)
		protected.POST("/coins/lookup", writeRateLimit, coinLookupHandler.Lookup)
		protected.PUT("/coins/:id", coinHandler.Update)
		protected.GET("/coins/:id/references", coinReferenceHandler.List)
		protected.POST("/coins/:id/references", coinReferenceHandler.Create)
		protected.PUT("/coins/:id/references/:referenceId", coinReferenceHandler.Update)
		protected.DELETE("/coins/:id/references/:referenceId", coinReferenceHandler.Delete)
		protected.POST("/references/migrate-legacy", coinReferenceHandler.MigrateLegacy)
		protected.POST("/coins/:id/purchase", coinHandler.Purchase)
		protected.POST("/coins/:id/sell", coinHandler.Sell)
		protected.DELETE("/coins/:id", coinHandler.Delete)
		protected.GET("/catalogs", catalogRegistryHandler.List)

		noteSvc := services.NewNoteService(noteRepo)
		noteHandler := handlers.NewNoteHandler(noteSvc)
		protected.GET("/notes", noteHandler.List)
		protected.POST("/notes", noteHandler.Create)
		protected.GET("/notes/:id", noteHandler.Get)
		protected.PUT("/notes/:id", noteHandler.Update)
		protected.DELETE("/notes/:id", noteHandler.Delete)

		storageLocationSvc := services.NewStorageLocationService(storageLocationRepo)
		storageLocationHandler := handlers.NewStorageLocationHandler(storageLocationSvc)
		protected.GET("/storage-locations", storageLocationHandler.List)
		protected.POST("/storage-locations", storageLocationHandler.Create)
		protected.PUT("/storage-locations/:id", storageLocationHandler.Update)
		protected.DELETE("/storage-locations/:id", storageLocationHandler.Delete)
		mintLocationSvc := services.NewMintLocationService(mintLocationRepo)
		mintLocationHandler := handlers.NewMintLocationHandler(mintLocationSvc)
		protected.GET("/mint-locations", mintLocationHandler.List)

		tagRepo := repository.NewTagRepository(database.DB)
		tagHandler := handlers.NewTagHandler(tagRepo)
		setRepo := repository.NewSetRepository(database.DB)
		protected.GET("/tags", tagHandler.List)
		protected.POST("/tags", tagHandler.Create)
		protected.PUT("/tags/:id", tagHandler.Update)
		protected.DELETE("/tags/:id", tagHandler.Delete)
		bulkHandler := handlers.NewBulkHandler(coinRepo, tagRepo, storageLocationRepo, setRepo)
		protected.POST("/coins/bulk", bulkHandler.BulkAction)

		protected.POST("/coins/:id/tags", tagHandler.AttachToCoin)
		protected.DELETE("/coins/:id/tags/:tagId", tagHandler.DetachFromCoin)

		// Sets - new endpoints for coin sets
		setService := services.NewSetService(setRepo, tagRepo, notifRepo)
		setHandler := handlers.NewSetHandler(setRepo, setService)
		setSnapshotScheduler := services.NewSetSnapshotScheduler(setService, settingsSvc, logger)
		go setSnapshotScheduler.Start()
		protected.GET("/sets", setHandler.List)
		protected.GET("/sets/templates", setHandler.GetTemplates)
		protected.POST("/sets/import-csv", setHandler.CreateFromCSV)
		protected.POST("/sets/compare", setHandler.CompareSets)
		protected.POST("/sets/preview-smart", setHandler.PreviewSmartSet)
		protected.POST("/sets", setHandler.Create)
		protected.GET("/sets/:id", setHandler.Get)
		protected.PUT("/sets/:id", setHandler.Update)
		protected.DELETE("/sets/:id", setHandler.Delete)
		protected.GET("/sets/:id/coins", setHandler.GetCoins)
		protected.POST("/sets/:id/coins", setHandler.AddCoin)
		protected.PUT("/sets/:id/coins/order", setHandler.ReorderCoins)
		protected.DELETE("/sets/:id/coins/:coinId", setHandler.RemoveCoin)
		protected.GET("/sets/:id/completion", setHandler.GetCompletion)
		protected.POST("/sets/:id/snapshot", setHandler.CreateSnapshot)
		protected.GET("/sets/:id/trends", setHandler.GetTrends)
		protected.GET("/sets/:id/analytics", setHandler.GetAnalytics)

		journalHandler := handlers.NewJournalHandler(journalRepo)
		protected.GET("/coins/:id/journal", journalHandler.ListEntries)
		protected.POST("/coins/:id/journal", journalHandler.AddEntry)
		protected.DELETE("/coins/:id/journal/:entryId", journalHandler.DeleteEntry)

		protected.GET("/stats", coinHandler.Stats)
		healthHandler := handlers.NewHealthHandler(healthSvc, logger)
		protected.GET("/stats/health", healthHandler.CollectionSummary)
		protected.GET("/coins/health", healthHandler.ListCoinHealth)
		protected.GET("/coins/:id/health", healthHandler.GetCoinHealth)
		protected.GET("/stats/distribution", coinHandler.Distribution)
		protected.GET("/stats/investment-breakdown", coinHandler.InvestmentBreakdown)
		protected.GET("/value-history", coinHandler.ValueHistory)
		protected.GET("/coins/:id/value-history", coinHandler.CoinValueHistory)
		protected.GET("/suggestions", coinHandler.Suggestions)

		protected.POST("/coins/:id/images", writeRateLimit, imageHandler.Upload)
		protected.POST("/coins/:id/images/base64", writeRateLimit, imageHandler.UploadBase64)
		protected.DELETE("/coins/:id/images/:imageId", imageHandler.Delete)
		protected.GET("/uploads/*filepath", imageHandler.ServeUpload)
		protected.GET("/proxy-image", imageHandler.ProxyImage)
		protected.GET("/scrape-image", imageHandler.ScrapeImage)

		analysisRepo := repository.NewAnalysisRepository(database.DB)
		analysisHandler := handlers.NewAnalysisHandler(analysisRepo, agentProxy, settingsSvc, logger)
		protected.POST("/coins/:id/analyze", writeRateLimit, analysisHandler.Analyze)
		protected.DELETE("/coins/:id/analyze", analysisHandler.DeleteAnalysis)
		protected.POST("/extract-text", analysisHandler.ExtractText)
		protected.GET("/ollama-status", analysisHandler.OllamaStatus)
		protected.GET("/ai-status", analysisHandler.AIStatus)

		// Coin of the Day (user-facing)
		coinOfDayHandler := handlers.NewCoinOfDayHandler(featuredCoinRepo, logger)
		protected.GET("/featured-coins/latest", coinOfDayHandler.Latest)
		protected.GET("/featured-coins/:id", coinOfDayHandler.Get)

		numistaHandler := handlers.NewNumistaHandler(settingsSvc)
		protected.GET("/numista/search", numistaHandler.Search)

		auctionLotSvc := services.NewAuctionLotService(auctionLotRepo, coinRepo)
		nbSvc := services.NewNumisBidsService(logger)
		auctionUserRepo := repository.NewUserRepository(database.DB)
		auctionLotHandler := handlers.NewAuctionLotHandler(auctionLotRepo, auctionLotSvc, auctionUserRepo, nbSvc, logger)
		protected.GET("/auctions", auctionLotHandler.List)
		protected.GET("/auctions/counts", auctionLotHandler.Counts)
		protected.PUT("/auctions/bulk-link-event", auctionLotHandler.BulkLinkEvent)
		protected.GET("/auctions/:id", auctionLotHandler.Get)
		protected.POST("/auctions", auctionLotHandler.Create)
		protected.PUT("/auctions/:id", auctionLotHandler.Update)
		protected.PUT("/auctions/:id/status", auctionLotHandler.UpdateStatus)
		protected.PUT("/auctions/:id/event", auctionLotHandler.LinkEvent)
		protected.POST("/auctions/:id/convert", auctionLotHandler.ConvertToCoin)
		protected.DELETE("/auctions/:id", auctionLotHandler.Delete)
		protected.POST("/auctions/import", writeRateLimit, auctionLotHandler.ImportFromURL)
		protected.POST("/auctions/sync", writeRateLimit, auctionLotHandler.SyncWatchlist)
		protected.POST("/auctions/validate-credentials", auctionLotHandler.ValidateNumisBids)

		// Wishlist availability checking
		availHandler := handlers.NewAvailabilityHandler(availSvc, availRepo, coinRepo)
		protected.POST("/wishlist/check-availability", availHandler.CheckAvailability)
		protected.PUT("/coins/:id/listing-status", availHandler.UpdateListingStatus)

		agentRepo := repository.NewAgentRepository(database.DB)
		userRepo := repository.NewUserRepository(database.DB)
		contentGuard := services.NewContentGuard(logger)
		agentHandler := handlers.NewAgentHandler(agentRepo, userRepo, journalRepo, agentProxy, collectionSvc, settingsSvc, internalTokenSvc, contentGuard, logger, cfg.AgentInternalCallbackURL)
		protected.POST("/agent/chat", writeRateLimit, agentHandler.ChatStream)
		protected.POST("/agent/collection/proposals/:proposalId/commit", writeRateLimit, agentHandler.CommitCollectionProposal)
		protected.POST("/agent/collection/proposals/:proposalId/cancel", writeRateLimit, agentHandler.CancelCollectionProposal)
		protected.POST("/coins/:id/estimate-value", writeRateLimit, agentHandler.EstimateValue)
		protected.GET("/agent/models", agentHandler.ListModels)
		protected.GET("/agent/coin-search-prompt", agentHandler.GetCoinSearchPrompt)
		protected.GET("/agent/coin-shows-prompt", agentHandler.GetCoinShowsPrompt)
		protected.GET("/agent/valuation-prompt", agentHandler.GetValuationPrompt)
		protected.GET("/agent/portfolio-summary", agentHandler.PortfolioSummary)
		protected.GET("/agent/status", agentHandler.AgentStatus)

		conversationRepo := repository.NewConversationRepository(database.DB)
		convHandler := handlers.NewConversationHandler(conversationRepo)
		protected.GET("/agent/conversations", convHandler.List)
		protected.GET("/agent/conversations/:id", convHandler.Get)
		protected.POST("/agent/conversations", convHandler.Save)
		protected.DELETE("/agent/conversations/:id", convHandler.Delete)

		// User self-service routes
		userHandler := handlers.NewUserHandler(cfg.UploadDir, userRepo, pushoverSvc, logger)
		protected.GET("/auth/me", userHandler.GetMe)
		protected.POST("/auth/change-password", userHandler.ChangePassword)
		protected.PUT("/user/profile", userHandler.UpdateProfile)
		protected.POST("/user/avatar", userHandler.UploadAvatar)
		protected.DELETE("/user/avatar", userHandler.DeleteAvatar)
		protected.GET("/user/export", userHandler.ExportCollection)
		protected.GET("/user/export/catalog", userHandler.ExportCatalogPDF)
		protected.POST("/user/import", writeRateLimit, userHandler.ImportCollection)
		protected.POST("/notifications/test-pushover", userHandler.TestPushover)

		// Social routes
		socialSvc := services.NewSocialService(socialRepo, notifSvc)
		socialHandler := handlers.NewSocialHandler(socialRepo, socialSvc, logger)
		protected.POST("/social/follow/:userId", socialHandler.FollowUser)
		protected.DELETE("/social/follow/:userId", socialHandler.UnfollowUser)
		protected.PUT("/social/followers/:userId/accept", socialHandler.AcceptFollower)
		protected.PUT("/social/followers/:userId/block", socialHandler.BlockFollower)
		protected.DELETE("/social/followers/:userId/block", socialHandler.UnblockFollower)
		protected.GET("/social/followers", socialHandler.GetFollowers)
		protected.GET("/social/following", socialHandler.GetFollowing)
		protected.GET("/social/blocked", socialHandler.GetBlockedUsers)
		protected.GET("/social/following/:userId/coins", socialHandler.GetFollowingCoins)
		protected.GET("/social/following/:userId/coins/:coinId", socialHandler.GetFollowingCoinDetail)
		protected.GET("/users/search", socialHandler.SearchUsers)
		protected.GET("/users/:username", socialHandler.GetPublicProfile)
		protected.POST("/social/coins/:coinId/comments", socialHandler.AddComment)
		protected.GET("/social/coins/:coinId/comments", socialHandler.GetComments)
		protected.DELETE("/social/coins/:coinId/comments/:commentId", socialHandler.DeleteComment)
		protected.PUT("/social/coins/:coinId/rating", socialHandler.RateCoin)
		protected.GET("/social/coins/:coinId/rating", socialHandler.GetCoinRating)

		// Notification routes
		notifHandler := handlers.NewNotificationHandler(notifRepo)
		protected.GET("/notifications", notifHandler.List)
		protected.GET("/notifications/unread-count", notifHandler.UnreadCount)
		protected.PUT("/notifications/:id/read", notifHandler.MarkRead)
		protected.PUT("/notifications/read-all", notifHandler.MarkAllRead)
		protected.DELETE("/notifications/:id", notifHandler.Delete)

		// API key management
		apiKeyHandler := handlers.NewApiKeyHandler(apiKeyRepo, logger, cfg.JWTSecret)
		protected.POST("/auth/api-keys", apiKeyHandler.Generate)
		protected.GET("/auth/api-keys", apiKeyHandler.List)
		protected.DELETE("/auth/api-keys/:id", apiKeyHandler.Revoke)

		// WebAuthn registration (requires auth)
		protected.POST("/auth/webauthn/register/begin", webauthnHandler.RegisterBegin)
		protected.POST("/auth/webauthn/register/finish", webauthnHandler.RegisterFinish)
		protected.GET("/auth/webauthn/credentials", webauthnHandler.ListCredentials)
		protected.DELETE("/auth/webauthn/credentials/:id", webauthnHandler.DeleteCredential)

		// Showcase routes
		showcaseRepo := repository.NewShowcaseRepository(database.DB)
		showcaseHandler := handlers.NewShowcaseHandler(showcaseRepo)
		protected.GET("/showcases", showcaseHandler.ListShowcases)
		protected.GET("/showcases/:id", showcaseHandler.GetShowcase)
		protected.POST("/showcases", showcaseHandler.CreateShowcase)
		protected.PUT("/showcases/:id", showcaseHandler.UpdateShowcase)
		protected.DELETE("/showcases/:id", showcaseHandler.DeleteShowcase)
		protected.PUT("/showcases/:id/coins", showcaseHandler.SetShowcaseCoins)

		// Calendar / Auction Event routes
		eventRepo := repository.NewAuctionEventRepository(database.DB)
		calendarHandler := handlers.NewCalendarHandler(eventRepo, auctionLotRepo)
		protected.GET("/calendar", calendarHandler.GetCalendar)
		protected.GET("/calendar/events", calendarHandler.ListEvents)
		protected.GET("/calendar/events/:id", calendarHandler.GetEvent)
		protected.POST("/calendar/events", calendarHandler.CreateEvent)
		protected.PUT("/calendar/events/:id", calendarHandler.UpdateEvent)
		protected.DELETE("/calendar/events/:id", calendarHandler.DeleteEvent)

		// Price Alerts & Bid Reminders
		priceAlertRepo := repository.NewPriceAlertRepository(database.DB)
		bidReminderRepo := repository.NewBidReminderRepository(database.DB)
		alertHandler := handlers.NewAlertHandler(priceAlertRepo, bidReminderRepo)
		protected.GET("/alerts", alertHandler.ListAlerts)
		protected.POST("/alerts", alertHandler.CreateAlert)
		protected.DELETE("/alerts/:id", alertHandler.DeleteAlert)
		protected.GET("/reminders", alertHandler.ListReminders)
		protected.POST("/reminders", alertHandler.CreateReminder)
		protected.DELETE("/reminders/:id", alertHandler.DeleteReminder)
	}

	// Admin-only routes
	admin := api.Group("/admin")
	admin.Use(middleware.AuthRequiredWithSecurity(cfg.JWTSecret, apiKeyAuth, securitySvc))
	admin.Use(middleware.RejectAPIKeyAuth())
	admin.Use(handlers.AdminRequired())
	{
		adminRepo := repository.NewAdminRepository(database.DB)
		adminHandler := handlers.NewAdminHandler(cfg.UploadDir, adminRepo, agentProxy, settingsSvc, logger)
		admin.GET("/users", adminHandler.ListUsers)
		admin.DELETE("/users/:id", adminHandler.DeleteUser)
		admin.POST("/users/:id/reset-password", adminHandler.ResetPassword)
		admin.PUT("/users/:id/role", adminHandler.UpdateUserRole)
		admin.GET("/settings", adminHandler.GetSettings)
		admin.GET("/settings/defaults", adminHandler.GetSettingDefaults)
		admin.PUT("/settings", adminHandler.UpdateSettings)
		admin.GET("/logs", adminHandler.GetLogs)
		admin.GET("/test-anthropic", adminHandler.TestAnthropicConnection)
		admin.GET("/test-searxng", adminHandler.TestSearXNGConnection)

		securityAdminHandler := handlers.NewSecurityAdminHandler(securitySvc, settingsSvc, handlers.SecurityExposureConfig{
			PublicAppURL:             settingsSvc.GetSetting(services.SettingPublicAppURL),
			WebAuthnOrigin:           cfg.WebAuthnOrigin,
			CORSOrigins:              cfg.AllowedOrigins(),
			TrustedProxiesConfigured: cfg.TrustedProxies != "",
			AgentInternalTokenSet:    cfg.AgentInternalServiceToken != "",
			RegistrationMode:         settingsSvc.GetSetting(services.SettingRegistrationMode),
			BackupStatus:             settingsSvc.GetSetting(services.SettingBackupStatus),
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
		mintLocationSvc := services.NewMintLocationService(mintLocationRepo)
		mintLocationHandler := handlers.NewMintLocationHandler(mintLocationSvc)
		admin.POST("/mint-locations", mintLocationHandler.Create)
		admin.PUT("/mint-locations/:id", mintLocationHandler.Update)
		admin.DELETE("/mint-locations/:id", mintLocationHandler.Delete)

		// Availability check run history (reuse availRepo from outer scope)
		adminAvailHandler := handlers.NewAvailabilityHandler(nil, availRepo, nil)
		admin.GET("/availability-runs", adminAvailHandler.ListRuns)
		admin.GET("/availability-runs/:id", adminAvailHandler.GetRunDetail)

		// Valuation run history and manual trigger
		valAdminHandler := handlers.NewValuationAdminHandler(valRepo, valSvc, logger)
		admin.GET("/valuation-runs", valAdminHandler.ListRuns)
		admin.GET("/valuation-runs/:id", valAdminHandler.GetRunDetail)
		admin.POST("/valuation-runs/trigger", valAdminHandler.TriggerValuation)
		admin.POST("/valuation-runs/:id/cancel", valAdminHandler.CancelValuation)

		// Auction ending run history and manual trigger
		auctionEndingAdminHandler := handlers.NewAuctionEndingAdminHandler(auctionEndingRepo, auctionEndingScheduler, logger)
		admin.GET("/auction-ending-runs", auctionEndingAdminHandler.ListRuns)
		admin.POST("/auction-ending/run", auctionEndingAdminHandler.TriggerRun)

		// Coin of the Day manual trigger
		coinOfDayAdminHandler := handlers.NewCoinOfDayAdminHandler(coinOfDayScheduler, logger)
		admin.POST("/coin-of-day/run", coinOfDayAdminHandler.TriggerRun)

		// Aggregate health metrics
		adminHealthHandler := handlers.NewAdminHealthHandler(healthSvc, healthScheduler, logger)
		admin.GET("/health/summary", adminHealthHandler.Summary)
		admin.POST("/collection-health-snapshots/run", adminHealthHandler.TriggerSnapshotRun)

		// API key rotation notification trigger
		apiKeyAdminHandler := handlers.NewApiKeyAdminHandler(apiKeyRepo, notifSvc, logger)
		admin.POST("/api-keys/notify-rotation", apiKeyAdminHandler.NotifyRotationRequired)

		// Auction ending debug endpoint
		auctionDebugHandler := handlers.NewAuctionEndingDebugHandler(auctionLotRepo)
		admin.GET("/auction-ending/debug", auctionDebugHandler.DebugGetAuctionEndingInfo)
	}

	// #218 external tool server - public versioned route group
	// Middleware chain: kill-switch gate → API-key auth → per-key rate limiter
	externalToolsRateLimit := middleware.ExternalAPIKeyRateLimit(50, 1*time.Minute)

	// Unauthenticated OpenAPI spec endpoint (respects kill-switch only)
	toolsSpec := api.Group("/v1/tools")
	toolsSpec.Use(middleware.ExternalToolServerEnabled(settingsSvc))
	{
		openapiHandler := handlers.NewExternalToolsOpenAPIHandler()
		toolsSpec.GET("/openapi.json", openapiHandler.GetOpenAPISpec)
	}

	// Authenticated tool endpoints (auth + rate limit)
	v1Tools := api.Group("/v1/tools")
	v1Tools.Use(middleware.ExternalToolServerEnabled(settingsSvc))
	v1Tools.Use(middleware.AuthRequiredWithSecurity(cfg.JWTSecret, apiKeyAuth, securitySvc))
	v1Tools.Use(externalToolsRateLimit)
	{
		externalToolsHandler := handlers.NewExternalToolsHandler(collectionSvc)

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

	// Internal tools (protected by internal token for Python agent callbacks)
	internal := r.Group("/api/internal/tools")
	internal.Use(middleware.InternalTokenRequired(internalTokenSvc))
	{
		internalToolsHandler := handlers.NewInternalToolsHandler(collectionSvc, logger)
		internal.POST("/search_my_collection", internalToolsHandler.SearchMyCollection)
		internal.POST("/get_coin", internalToolsHandler.GetCoin)
		internal.POST("/collection_summary", internalToolsHandler.CollectionSummary)
		internal.POST("/top_coins_by_value", internalToolsHandler.TopCoinsByValue)
		internal.POST("/propose_update", internalToolsHandler.ProposeUpdate)
		internal.POST("/commit_update", internalToolsHandler.CommitUpdate)
	}

	log.Printf("Starting server on :%s", cfg.Port)
	logger.Info("startup", "Server starting on port %s", cfg.Port)
	logger.Info("startup", "Log level: %s", logger.GetLevel())

	// Warn if callback URL is likely misconfigured in release mode
	if os.Getenv("GIN_MODE") == "release" && strings.Contains(cfg.AgentInternalCallbackURL, "localhost") {
		logger.Warn("startup", "AGENT_INTERNAL_CALLBACK_URL is set to '%s' in release mode. Collection chat (#217) will fail in multi-container deployments. Set it to the API container's network address (e.g., http://app:8080).", cfg.AgentInternalCallbackURL)
	}
	if os.Getenv("GIN_MODE") == "release" && cfg.AgentInternalServiceToken == "" {
		log.Fatal("FATAL: AGENT_INTERNAL_SERVICE_TOKEN must be set in production")
	}

	// Check Ollama connectivity at startup (blocks until complete)
	func() {
		ollamaURL := settingsSvc.GetSetting(services.SettingOllamaURL)
		ollamaModel := settingsSvc.GetSetting(services.SettingOllamaModel)
		svc := services.NewOllamaService(ollamaURL, 10, logger)
		available, msg := svc.CheckModel(ollamaModel)
		if available {
			logger.Info("startup", "Ollama: %s", msg)
		} else {
			logger.Warn("startup", "Ollama: %s — AI features will be unavailable until resolved", msg)
		}
	}()

	// Start schedulers
	schedulerRegistry.StartAll()
	go coinOfDayScheduler.Start()

	// Startup API key rotation sync:
	// keep notifying users with pre-cutoff keys until those keys are revoked/recreated.
	apiKeyRotationSvc := services.NewAPIKeyRotationService(apiKeyRepo, notifRepo, notifSvc, settingsSvc, logger)
	apiKeyRotationSvc.SyncFromStartup()

	logger.Info("startup", "Application ready")
	log.Println("Application ready")

	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
