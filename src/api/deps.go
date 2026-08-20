package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/briandenicola/ancient-coins-api/config"
	"github.com/briandenicola/ancient-coins-api/database"
	"github.com/briandenicola/ancient-coins-api/handlers"
	"github.com/briandenicola/ancient-coins-api/middleware"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
)

// appDeps is the application's dependency container. It exists because the
// composition root previously lived as ~750 lines inside main(), where the
// route groups closed over 40+ locals and no part could be read or moved in
// isolation. Construction happens once in buildDeps; the register*Routes
// functions in routes_*.go read what they need from here.
//
// This is deliberately a plain hand-wired struct rather than a DI framework:
// the wiring order encodes real initialisation dependencies (deepIdentificationSvc
// must have its pipeline runner set before its workers start, availRepo must be
// given its cycle repo before the scheduler reads it), and hand-wiring keeps
// that order explicit and greppable.
type appDeps struct {
	cfg                            *config.Config
	logger                         *services.Logger
	settingsSvc                    *services.SettingsService
	securitySvc                    *services.SecurityService
	oidcSvc                        *services.OIDCService
	authHandler                    *handlers.AuthHandler
	webauthnHandler                *handlers.WebAuthnHandler
	apiKeyRepo                     *repository.ApiKeyRepository
	apiKeyAuth                     middleware.ApiKeyAuthenticator
	imageSvc                       *services.ImageService
	imageHandler                   *handlers.ImageHandler
	internalTokenSvc               *services.InternalTokenService
	credentialEncryptionSvc        *services.CredentialEncryptionService
	numistaLookupSvc               *services.NumistaLookupService
	numistaTelemetry               *services.NumistaTelemetry
	authRateLimit                  gin.HandlerFunc
	apiRateLimit                   gin.HandlerFunc
	writeRateLimit                 gin.HandlerFunc
	agentProxy                     *services.AgentProxy
	coinRepo                       *repository.CoinRepository
	journalRepo                    *repository.JournalRepository
	noteRepo                       *repository.NoteRepository
	socialRepo                     *repository.SocialRepository
	notifRepo                      *repository.NotificationRepository
	notifSvc                       *services.NotificationService
	pushoverSvc                    *services.PushoverService
	collectionSvc                  *services.CollectionToolsService
	availRepo                      *repository.AvailabilityRepository
	availCycleRepo                 *repository.AvailabilityCycleRepository
	availSvc                       *services.AvailabilityService
	availScheduler                 *services.AvailabilityScheduler
	valRepo                        *repository.ValuationRepository
	valSvc                         *services.ValuationService
	aiJobSvc                       *services.AIJobService
	coinLookupSvc                  *services.CoinLookupService
	deepIdentificationRepo         *repository.DeepIdentificationRepository
	deepIdentificationSvc          *services.DeepIdentificationService
	healthSvc                      *services.HealthService
	healthScheduler                *services.CollectionHealthScheduler
	shipmentSvc                    *services.ShipmentService
	wishlistSearchAlertSvc         *services.WishlistSearchAlertService
	auctionLotRepo                 *repository.AuctionLotRepository
	auctionEndingRepo              *repository.AuctionEndingRepository
	auctionEndingScheduler         *services.AuctionEndingScheduler
	auctionWatchBidDigestRepo      *repository.AuctionWatchBidDigestRepository
	auctionWatchBidDigestScheduler *services.AuctionWatchBidDigestScheduler
	auctionAlertRunRepo            *repository.AuctionAlertRunRepository
	auctionAlertScheduler          *services.AuctionAlertScheduler
	priceAlertRepo                 *repository.PriceAlertRepository
	bidReminderRepo                *repository.BidReminderRepository
	featuredCoinRepo               *repository.FeaturedCoinRepository
	coinOfDayScheduler             *services.CoinOfDayScheduler
	schedulerRegistry              *SchedulerRegistry
	purchaseReminderRepo           *repository.PurchaseReminderRepository
	purchaseReminderSvc            *services.PurchaseReminderService
	timeMachineSvc                 *services.TimeMachineService
	externalToolsRateLimit         gin.HandlerFunc
	numistaClient                  *services.HTTPNumistaClient
	deepNomismaClient              *services.HTTPNomismaClient
	deepOCREClient                 *services.HTTPOCREClient
	deepOCRECache                  *services.OCRECache
	deepProviderBudgets            *services.DeepProviderBudgetTracker
}

// buildDeps constructs every repository, service, scheduler, and shared piece
// of middleware the routes need. The returned CancelFunc stops context-aware
// background work and is handed to runServer for graceful shutdown.
//
// database.Connect must have been called before this runs.
func buildDeps(cfg *config.Config) (*appDeps, context.CancelFunc) {
	// Create logger and settings service
	logger := services.NewLogger(1000)
	settingsRepo := repository.NewSettingsRepository(database.DB)
	settingsSvc := services.NewSettingsService(settingsRepo)
	settingsSvc.SyncLogLevel(logger)
	numistaClock := services.NewSystemNumistaClock()
	numistaCache := services.NewNumistaCache(numistaClock, 500, 5000)
	numistaTelemetry := services.NewNumistaTelemetry(500)
	numistaQueryBuilder := services.NewNumistaQueryBuilder()
	numistaClient, err := services.NewHTTPNumistaClient(services.NumistaClientConfig{
		APIKey:        func() string { return settingsSvc.GetSetting(services.SettingNumistaAPIKey) },
		SearchTimeout: func() time.Duration { return settingsSvc.GetNumistaSettings().SearchTimeout },
		DetailTimeout: func() time.Duration { return settingsSvc.GetNumistaSettings().DetailTimeout },
	})
	if err != nil {
		log.Fatalf("Failed to configure Numista client: %v", err)
	}
	numistaLookupSvc := services.NewNumistaLookupService(
		numistaClient, numistaCache, services.NewNumistaV1Scorer(),
		numistaTelemetry, settingsSvc, numistaClock, numistaQueryBuilder,
	)

	// Create internal token service for Python agent callbacks
	internalTokenSvc := services.NewInternalTokenService(cfg.JWTSecret)
	// Deep identification provider-tool boundary (Phase 6, T051-T054):
	// job-scoped call-budget tracking, shared across the numista_search/
	// numista_detail/nomisma_search internal tool endpoints.
	deepProviderBudgets := services.NewDeepProviderBudgetTracker()
	deepNomismaClient := services.NewHTTPNomismaClient()
	// Feature 345: OCRE automated Deep Analysis provider — the single
	// Nomisma-SPARQL HTTP boundary + its bounded search cache, shared with
	// the ocre_search internal tool endpoint (default-off; gated by the
	// DeepIdentificationOCREEnabled setting via the provider catalog).
	deepOCREClient := services.NewHTTPOCREClient()
	deepOCRECache := services.NewOCRECache()
	credentialEncryptionSvc := services.NewDisabledCredentialEncryptionService()
	if cfg.AuctionCredentialEncryptionKey != "" {
		var err error
		credentialEncryptionSvc, err = services.NewCredentialEncryptionService(cfg.AuctionCredentialEncryptionKey)
		if err != nil {
			log.Fatalf("Failed to configure auction credential encryption: %v", err)
		}
	}

	logger.Info("startup", "Application starting")
	logger.Info("startup", "Database connected: %s", cfg.DBPath)

	// Ensure upload directory exists
	if err := os.MkdirAll(cfg.UploadDir, 0755); err != nil {
		log.Fatalf("Failed to create upload directory: %v", err)
	}
	logger.Debug("startup", "Upload directory: %s", cfg.UploadDir)

	// Auth routes (public) — rate limited to prevent brute force
	authRepo := repository.NewAuthRepository(database.DB)
	securityRepo := repository.NewSecurityRepository(database.DB)
	securitySvc := services.NewSecurityService(securityRepo)
	oidcRepo := repository.NewOIDCRepository(database.DB)
	authSvc := services.NewAuthService(authRepo, cfg.JWTSecret).WithSettings(settingsSvc).WithSecurity(securitySvc).WithOIDC(oidcRepo)
	oidcSvc := services.NewOIDCService(oidcRepo, services.NewDefaultOIDCDiscoveryFactory()).WithSecurity(securitySvc).WithAuth(authSvc)
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
	apiRateLimit := middleware.AuthenticatedRateLimit(600, 1*time.Minute)  // Authenticated browsing
	writeRateLimit := middleware.AuthenticatedRateLimit(30, 1*time.Minute) // Write operations

	// Protected routes
	agentProxy := services.NewAgentProxy(cfg.AgentServiceURL, cfg.AgentInternalServiceToken, logger)
	availRepo := repository.NewAvailabilityRepository(database.DB)
	availCycleRepo := repository.NewAvailabilityCycleRepository(database.DB)
	availRepo.WithCycleRepo(availCycleRepo)
	coinRepo := repository.NewCoinRepository(database.DB)
	wishlistSearchAlertRepo := repository.NewWishlistSearchAlertRepository(database.DB)
	socialRepo := repository.NewSocialRepository(database.DB)
	notifRepo := repository.NewNotificationRepository(database.DB)
	valRepo := repository.NewValuationRepository(database.DB)
	auctionEndingRepo := repository.NewAuctionEndingRepository(database.DB)
	userRepoForVal := repository.NewUserRepository(database.DB)
	auctionLotRepo := repository.NewAuctionLotRepository(database.DB)
	pushoverSvc := services.NewPushoverService(settingsSvc, logger)
	notifSvc := services.NewNotificationService(notifRepo, socialRepo, userRepoForVal, pushoverSvc, logger)
	availSvc := services.NewAvailabilityService(coinRepo, availRepo, agentProxy, notifSvc, pushoverSvc, userRepoForVal, settingsSvc, logger).WithCycleRepo(availCycleRepo)
	wishlistSearchAlertSvc := services.NewWishlistSearchAlertService(wishlistSearchAlertRepo).WithDiscovery(agentProxy, settingsSvc)
	wishlistSearchAlertSvc.StartWorkers(1)
	wishlistSearchAlertScheduler := services.NewWishlistSearchAlertScheduler(wishlistSearchAlertSvc, wishlistSearchAlertRepo, settingsSvc, logger)
	valSvc := services.NewValuationService(coinRepo, valRepo, agentProxy, userRepoForVal, pushoverSvc, notifSvc, settingsSvc, logger)
	aiJobRepo := repository.NewAIJobRepository(database.DB)
	aiJobSvc := services.NewAIJobService(aiJobRepo, agentProxy, userRepoForVal, settingsSvc, notifSvc, logger)
	aiJobSvc.StartWorkers(1)
	coinLookupSvc := services.NewCoinLookupService(agentProxy, settingsSvc, logger, numistaQueryBuilder)
	deepIdentificationRepo := repository.NewDeepIdentificationRepository(database.DB)
	deepIdentificationCatalogRegistryRepo := repository.NewCatalogRegistryRepository(database.DB)
	deepIdentificationSvc := services.NewDeepIdentificationService(deepIdentificationRepo, imageRepo, imageSvc, settingsSvc, logger, cfg.UploadDir)
	deepIdentificationSvc.SetProviderBudgetTracker(deepProviderBudgets)
	deepIdentificationSvc.SetInternalTokenService(internalTokenSvc)
	deepIdentificationSvc.SetPipelineRunner(services.NewDeepIdentificationPipelineRunner(
		agentProxy, deepIdentificationRepo, settingsSvc, internalTokenSvc, cfg.AgentInternalCallbackURL, logger, deepIdentificationSvc.Broker(), deepIdentificationCatalogRegistryRepo,
	).WithQuickEvidence(coinLookupSvc))
	// Cancelled on SIGTERM/SIGINT by shutdownRuntime so the deep-identification
	// workers and janitor unwind instead of dying mid-write.
	backgroundCtx, cancelBackground := context.WithCancel(context.Background())
	deepIdentificationSvc.StartWorkers(backgroundCtx)
	deepIdentificationSvc.StartJanitor(backgroundCtx)
	timeMachineRepo := repository.NewTimeMachineRepository(database.DB)
	timeMachineSvc := services.NewTimeMachineService(timeMachineRepo)
	healthRepo := repository.NewHealthRepository(database.DB)
	healthSvc := services.NewHealthService(healthRepo, logger)
	auctionWatchBidDigestRepo := repository.NewAuctionWatchBidDigestRepository(database.DB)
	priceAlertRepo := repository.NewPriceAlertRepository(database.DB)
	bidReminderRepo := repository.NewBidReminderRepository(database.DB)
	auctionAlertRunRepo := repository.NewAuctionAlertRunRepository(database.DB)
	shipmentRepo := repository.NewShipmentRepository(database.DB)
	shipmentSvc := services.NewShipmentService(
		shipmentRepo,
		coinRepo,
		services.NewShipmentCarrierClientRegistry(buildShipmentCarrierClients(settingsSvc, logger)...),
		notifSvc,
		logger,
	).WithParcelAppSupport(userRepoForVal, settingsSvc, credentialEncryptionSvc, services.NewHTTPParcelAppClient())

	// Create schedulers before routes so they can be passed to admin handlers
	availScheduler := services.NewAvailabilityScheduler(availSvc, coinRepo, availRepo, settingsSvc, logger).WithCycleRepo(availCycleRepo)
	availScheduler.StartWorkers(1)
	valScheduler := services.NewValuationScheduler(valSvc, coinRepo, valRepo, settingsSvc, logger)
	nbWatchSyncSvc := services.NewNumisBidsService(logger)
	cngWatchSyncSvc := services.NewCNGAuctionService(logger)
	auctionWatchlistSyncSvc := services.NewAuctionWatchlistSyncService(auctionLotRepo, userRepoForVal, nbWatchSyncSvc, cngWatchSyncSvc, credentialEncryptionSvc, logger)
	auctionEndingScheduler := services.NewAuctionEndingScheduler(auctionLotRepo, auctionEndingRepo, userRepoForVal, pushoverSvc, notifSvc, settingsSvc, logger)
	auctionWatchBidDigestScheduler := services.NewAuctionWatchBidDigestScheduler(auctionLotRepo, auctionWatchBidDigestRepo, userRepoForVal, pushoverSvc, auctionWatchlistSyncSvc, settingsSvc, logger)
	auctionAlertEvaluator := services.NewAuctionAlertEvaluator(priceAlertRepo, bidReminderRepo, notifSvc, logger)
	auctionAlertScheduler := services.NewAuctionAlertScheduler(auctionAlertEvaluator, auctionAlertRunRepo, auctionWatchlistSyncSvc, settingsSvc, logger)
	shipmentScheduler := services.NewShipmentScheduler(shipmentSvc, settingsSvc, logger)
	collectionHealthSnapshotRunRepo := repository.NewCollectionHealthSnapshotRunRepository(database.DB)
	healthScheduler := services.NewCollectionHealthScheduler(healthSvc, collectionHealthSnapshotRunRepo, settingsSvc, logger)
	featuredCoinRepo := repository.NewFeaturedCoinRepository(database.DB)
	coinOfDayRunRepo := repository.NewCoinOfDayRunRepository(database.DB)
	coinOfDayScheduler := services.NewCoinOfDayScheduler(featuredCoinRepo, coinOfDayRunRepo, userRepoForVal, coinRepo, notifSvc, settingsSvc, logger)
	coinOfDayScheduler.SetWishlistSummaryClient(agentProxy)
	coinOfDayScheduler.StartWorkers(1)
	schedulerRegistry := &SchedulerRegistry{}
	purchaseReminderRepo := repository.NewPurchaseReminderRepository(database.DB)
	purchaseReminderSvc := services.NewPurchaseReminderService(purchaseReminderRepo, coinRepo, logger)
	reminderScheduler := services.NewReminderScheduler(purchaseReminderRepo, notifSvc, settingsSvc, logger)
	schedulerRegistry.Register(availScheduler)
	schedulerRegistry.Register(valScheduler)
	schedulerRegistry.Register(auctionEndingScheduler)
	schedulerRegistry.Register(auctionWatchBidDigestScheduler)
	schedulerRegistry.Register(auctionAlertScheduler)
	schedulerRegistry.Register(shipmentScheduler)
	schedulerRegistry.Register(healthScheduler)
	schedulerRegistry.Register(wishlistSearchAlertScheduler)
	schedulerRegistry.Register(reminderScheduler)

	// Create shared repositories for cross-group access
	journalRepo := repository.NewJournalRepository(database.DB)
	collectionProposalRepo := repository.NewCollectionUpdateRepository(database.DB)
	noteRepo := repository.NewNoteRepository(database.DB)
	collectionSvc := services.NewCollectionToolsService(coinRepo, collectionProposalRepo).WithSettingsSupport(settingsSvc)

	// #218 external tool server: per-key rate limiter shared by the
	// authenticated /api/v1/tools routes.
	externalToolsRateLimit := middleware.ExternalAPIKeyRateLimit(50, 1*time.Minute)

	return &appDeps{
		cfg:                            cfg,
		logger:                         logger,
		settingsSvc:                    settingsSvc,
		securitySvc:                    securitySvc,
		oidcSvc:                        oidcSvc,
		authHandler:                    authHandler,
		webauthnHandler:                webauthnHandler,
		apiKeyRepo:                     apiKeyRepo,
		apiKeyAuth:                     apiKeyAuth,
		imageSvc:                       imageSvc,
		imageHandler:                   imageHandler,
		internalTokenSvc:               internalTokenSvc,
		credentialEncryptionSvc:        credentialEncryptionSvc,
		numistaLookupSvc:               numistaLookupSvc,
		numistaTelemetry:               numistaTelemetry,
		authRateLimit:                  authRateLimit,
		apiRateLimit:                   apiRateLimit,
		writeRateLimit:                 writeRateLimit,
		agentProxy:                     agentProxy,
		coinRepo:                       coinRepo,
		journalRepo:                    journalRepo,
		noteRepo:                       noteRepo,
		socialRepo:                     socialRepo,
		notifRepo:                      notifRepo,
		notifSvc:                       notifSvc,
		pushoverSvc:                    pushoverSvc,
		collectionSvc:                  collectionSvc,
		availRepo:                      availRepo,
		availCycleRepo:                 availCycleRepo,
		availSvc:                       availSvc,
		availScheduler:                 availScheduler,
		valRepo:                        valRepo,
		valSvc:                         valSvc,
		aiJobSvc:                       aiJobSvc,
		coinLookupSvc:                  coinLookupSvc,
		deepIdentificationRepo:         deepIdentificationRepo,
		deepIdentificationSvc:          deepIdentificationSvc,
		healthSvc:                      healthSvc,
		healthScheduler:                healthScheduler,
		shipmentSvc:                    shipmentSvc,
		wishlistSearchAlertSvc:         wishlistSearchAlertSvc,
		auctionLotRepo:                 auctionLotRepo,
		auctionEndingRepo:              auctionEndingRepo,
		auctionEndingScheduler:         auctionEndingScheduler,
		auctionWatchBidDigestRepo:      auctionWatchBidDigestRepo,
		auctionWatchBidDigestScheduler: auctionWatchBidDigestScheduler,
		auctionAlertRunRepo:            auctionAlertRunRepo,
		auctionAlertScheduler:          auctionAlertScheduler,
		priceAlertRepo:                 priceAlertRepo,
		bidReminderRepo:                bidReminderRepo,
		featuredCoinRepo:               featuredCoinRepo,
		coinOfDayScheduler:             coinOfDayScheduler,
		schedulerRegistry:              schedulerRegistry,
		purchaseReminderRepo:           purchaseReminderRepo,
		purchaseReminderSvc:            purchaseReminderSvc,
		timeMachineSvc:                 timeMachineSvc,
		externalToolsRateLimit:         externalToolsRateLimit,
		numistaClient:                  numistaClient,
		deepNomismaClient:              deepNomismaClient,
		deepOCREClient:                 deepOCREClient,
		deepOCRECache:                  deepOCRECache,
		deepProviderBudgets:            deepProviderBudgets,
	}, cancelBackground
}
