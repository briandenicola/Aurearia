package main

import (
	"os"
	"path/filepath"
	"strings"
	_ "time/tzdata" // embed IANA timezone database so Alpine runtime can resolve IANA zones

	"github.com/briandenicola/ancient-coins-api/config"
	"github.com/briandenicola/ancient-coins-api/database"
	"github.com/briandenicola/ancient-coins-api/docs"
	"github.com/briandenicola/ancient-coins-api/middleware"
	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
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

// StopAll signals every registered scheduler to stop. Each Stop is guarded by
// a sync.Once in its implementation, so this is safe to call more than once.
func (r *SchedulerRegistry) StopAll() {
	for _, scheduler := range r.schedulers {
		scheduler.Stop()
	}
}

func buildShipmentCarrierClients(settingsSvc *services.SettingsService, logger *services.Logger) []services.ShipmentCarrierClient {
	carrierClients := make([]services.ShipmentCarrierClient, 0, 3)

	if strings.TrimSpace(settingsSvc.GetSetting(services.SettingUSPSAPIBaseURL)) != "" {
		uspsClient, err := services.NewUSPSShipmentCarrierClient(services.USPSShipmentClientConfigFromSettings(settingsSvc), nil)
		if err != nil {
			logger.Warn("shipment", "USPS client not configured: %v", err)
		} else {
			carrierClients = append(carrierClients, uspsClient)
		}
	}
	if strings.TrimSpace(settingsSvc.GetSetting(services.SettingUPSAPIBaseURL)) != "" {
		upsClient, err := services.NewUPSShipmentCarrierClient(services.UPSShipmentClientConfigFromSettings(settingsSvc), nil)
		if err != nil {
			logger.Warn("shipment", "UPS client not configured: %v", err)
		} else {
			carrierClients = append(carrierClients, upsClient)
		}
	}
	if strings.TrimSpace(settingsSvc.GetSetting(services.SettingFedExAPIBaseURL)) != "" {
		fedexClient, err := services.NewFedExShipmentCarrierClient(services.FedExShipmentClientConfigFromSettings(settingsSvc), nil)
		if err != nil {
			logger.Warn("shipment", "FedEx client not configured: %v", err)
		} else {
			carrierClients = append(carrierClients, fedexClient)
		}
	}

	return carrierClients
}

//	@title						Aurearia API
//	@version					4.0.0
//	@description				REST API for managing a personal coin collection. Supports coin CRUD, image uploads, AI-powered analysis, user management, auction tracking, and admin features.
//	@BasePath					/api
//	@securityDefinitions.apikey	BearerAuth
//	@in							header
//	@name						Authorization
//	@description				Enter your JWT token with the Bearer prefix, e.g. "Bearer eyJhbGci..."

//	@securityDefinitions.apikey	ApiKeyAuth
//	@in							header
//	@name						X-API-Key
//	@description				Enter your API key, e.g. "ak_a1b2c3d4..."

// loadAppVersion resolves the running build's version from the single
// canonical root VERSION file (F9/T096) so the live Swagger UI, the
// generated OpenAPI doc, and the Vue UI never drift from one another again.
// Falls back to the swag-baked literal (kept in sync by `task openapi`) if
// the file is unreadable, e.g. a `go run .` invocation from an unexpected
// working directory.
func loadAppVersion() string {
	for _, p := range []string{"VERSION", filepath.Join("..", "..", "VERSION")} {
		if data, err := os.ReadFile(p); err == nil {
			if v := strings.TrimSpace(string(data)); v != "" {
				return v
			}
		}
	}
	return docs.SwaggerInfo.Version
}

func main() {
	cfg := config.Load()

	docs.SwaggerInfo.Version = loadAppVersion()

	database.Connect(cfg.DBPath)

	d, cancelBackground := buildDeps(cfg)

	r := newHTTPRouter(cfg)
	r.Use(middleware.IPDenyRules(d.securitySvc))
	r.GET("/uploads/*filepath", middleware.AuthRequiredWithSecurity(cfg.JWTSecret, d.apiKeyAuth, d.securitySvc), d.apiRateLimit, d.imageHandler.ServeUpload)

	api := r.Group("/api")
	api.Use(middleware.RequestBodyLimit(middleware.DefaultRequestBodyLimitBytes))

	registerPublicRoutes(api, d)
	registerProtectedRoutes(api, d)
	registerAdminRoutes(api, d)
	registerExternalToolRoutes(api, d)
	registerInternalToolRoutes(r, d)

	runServer(serverRuntime{
		router:             r,
		config:             cfg,
		logger:             d.logger,
		settings:           d.settingsSvc,
		schedulers:         d.schedulerRegistry,
		coinOfDayScheduler: d.coinOfDayScheduler,
		apiKeys:            d.apiKeyRepo,
		notifications:      d.notifRepo,
		notificationSvc:    d.notifSvc,
		cancelBackground:   cancelBackground,
	})
}

func configureStaticRoutes(r *gin.Engine, wwwroot string) {
	r.Static("/assets", filepath.Join(wwwroot, "assets"))
	r.Static("/imgly-background-removal", filepath.Join(wwwroot, "imgly-background-removal"))
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
