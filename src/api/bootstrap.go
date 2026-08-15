package main

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/briandenicola/ancient-coins-api/config"
	"github.com/briandenicola/ancient-coins-api/middleware"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func newHTTPRouter(cfg *config.Config) *gin.Engine {
	router := gin.Default()
	if err := router.SetTrustedProxies(cfg.TrustedProxyList()); err != nil {
		log.Fatalf("Failed to configure trusted proxies: %v", err)
	}
	router.MaxMultipartMemory = middleware.DefaultMultipartMemoryBytes
	router.Use(middleware.SecurityHeaders())
	router.Use(middleware.ResolvedClientIP())
	router.Use(corsMiddleware(cfg.AllowedOrigins()))

	wwwroot := filepath.Join(".", "wwwroot")
	if _, err := os.Stat(wwwroot); err == nil {
		configureStaticRoutes(router, wwwroot)
	}

	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})
	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	return router
}

func corsMiddleware(allowedOrigins []string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		allowed[origin] = struct{}{}
	}

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if _, ok := allowed[origin]; ok {
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
	}
}

type serverRuntime struct {
	router             *gin.Engine
	config             *config.Config
	logger             *services.Logger
	settings           *services.SettingsService
	schedulers         *SchedulerRegistry
	coinOfDayScheduler *services.CoinOfDayScheduler
	apiKeys            *repository.ApiKeyRepository
	notifications      *repository.NotificationRepository
	notificationSvc    *services.NotificationService
}

func runServer(runtime serverRuntime) {
	log.Printf("Starting server on :%s", runtime.config.Port)
	runtime.logger.Info("startup", "Server starting on port %s", runtime.config.Port)
	runtime.logger.Info("startup", "Log level: %s", runtime.logger.GetLevel())

	validateProductionAgentConfig(runtime.config, runtime.logger)
	checkOllamaAtStartup(runtime.settings, runtime.logger)

	runtime.schedulers.StartAll()
	go runtime.coinOfDayScheduler.Start()

	apiKeyRotationSvc := services.NewAPIKeyRotationService(
		runtime.apiKeys,
		runtime.notifications,
		runtime.notificationSvc,
		runtime.settings,
		runtime.logger,
	)
	apiKeyRotationSvc.SyncFromStartup()

	runtime.logger.Info("startup", "Application ready")
	log.Println("Application ready")

	if err := runtime.router.Run(":" + runtime.config.Port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func validateProductionAgentConfig(cfg *config.Config, logger *services.Logger) {
	if os.Getenv("GIN_MODE") != "release" {
		return
	}
	if strings.Contains(cfg.AgentInternalCallbackURL, "localhost") {
		logger.Warn("startup", "AGENT_INTERNAL_CALLBACK_URL is set to '%s' in release mode. Collection chat (#217) will fail in multi-container deployments. Set it to the API container's network address (e.g., http://app:8080).", cfg.AgentInternalCallbackURL)
	}
	if cfg.AgentInternalServiceToken == "" {
		log.Fatal("FATAL: AGENT_INTERNAL_SERVICE_TOKEN must be set in production")
	}
}

func checkOllamaAtStartup(settings *services.SettingsService, logger *services.Logger) {
	ollamaURL := settings.GetSetting(services.SettingOllamaURL)
	ollamaModel := settings.GetSetting(services.SettingOllamaModel)
	svc := services.NewOllamaService(ollamaURL, 10, logger)
	available, msg := svc.CheckModel(ollamaModel)
	if available {
		logger.Info("startup", "Ollama: %s", msg)
		return
	}
	logger.Warn("startup", "Ollama: %s — AI features will be unavailable until resolved", msg)
}
