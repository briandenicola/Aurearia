package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

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
	router.Use(middleware.ContentSecurityPolicy())
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
	// cancelBackground stops context-aware background work (currently the
	// deep-identification workers and janitor) during shutdown.
	cancelBackground context.CancelFunc
}

// shutdownGracePeriod bounds how long we wait for in-flight HTTP requests to
// finish before forcing the process down. Container runtimes typically send
// SIGKILL 10s after SIGTERM, so this stays under that budget.
const shutdownGracePeriod = 8 * time.Second

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

	srv := &http.Server{
		Addr:    ":" + runtime.config.Port,
		Handler: runtime.router,
	}

	// Serve in the background so the main goroutine can wait on a signal.
	// ErrServerClosed is the expected result of a clean Shutdown, not a fault.
	serveErr := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErr <- err
			return
		}
		serveErr <- nil
	}()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serveErr:
		if err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
		return
	case sig := <-signals:
		runtime.logger.Info("shutdown", "Received %s, shutting down", sig)
		log.Printf("Received %s, shutting down", sig)
	}

	shutdownRuntime(runtime, srv)
}

// shutdownRuntime drains the HTTP listener first so no new request can start,
// then stops background work. Ordering matters: stopping the schedulers first
// would let an in-flight request enqueue work that nothing is left to run.
func shutdownRuntime(runtime serverRuntime, srv *http.Server) {
	ctx, cancel := context.WithTimeout(context.Background(), shutdownGracePeriod)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		// A timeout here means requests were still in flight at the deadline;
		// they are dropped, but everything else still shuts down cleanly.
		runtime.logger.Warn("shutdown", "HTTP shutdown did not complete cleanly: %v", err)
	}

	if runtime.schedulers != nil {
		runtime.schedulers.StopAll()
	}
	if runtime.coinOfDayScheduler != nil {
		runtime.coinOfDayScheduler.Stop()
	}
	if runtime.cancelBackground != nil {
		runtime.cancelBackground()
	}

	// Queue-backed worker pools (AI jobs, wishlist alerts, availability,
	// coin-of-day) have no stop hook yet; jobs left mid-flight are picked up
	// by each service's stale-job recovery on the next start.
	runtime.logger.Info("shutdown", "Shutdown complete")
	log.Println("Shutdown complete")
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
