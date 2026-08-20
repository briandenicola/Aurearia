package main

import (
	"github.com/briandenicola/ancient-coins-api/database"
	"github.com/briandenicola/ancient-coins-api/handlers"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/gin-gonic/gin"
)

// registerPublicRoutes registers the unauthenticated /api endpoints: login,
// registration, the OIDC and WebAuthn login ceremonies, and the public
// showcase reader.
func registerPublicRoutes(api *gin.RouterGroup, d *appDeps) {
	api.GET("/auth/setup", d.authHandler.NeedsSetup)
	api.POST("/auth/register", d.authRateLimit, d.authHandler.Register)
	api.POST("/auth/login", d.authRateLimit, d.authHandler.Login)
	api.POST("/auth/refresh", d.authRateLimit, d.authHandler.Refresh)
	oidcHandler := handlers.NewOIDCHandler(d.oidcSvc)
	api.GET("/auth/oidc/providers", d.authRateLimit, oidcHandler.ListPublicProviders)
	api.POST("/auth/oidc/:providerId/start", d.authRateLimit, oidcHandler.StartLogin)
	api.GET("/auth/oidc/:providerId/callback", d.authRateLimit, oidcHandler.Callback)
	api.GET("/auth/oidc/:providerId/link/callback", d.authRateLimit, oidcHandler.LinkCallback)

	// WebAuthn public routes (login ceremony)
	api.POST("/auth/webauthn/login/begin", d.authRateLimit, d.webauthnHandler.LoginBegin)
	api.POST("/auth/webauthn/login/finish", d.authRateLimit, d.webauthnHandler.LoginFinish)
	api.GET("/auth/webauthn/check", d.webauthnHandler.CheckCredentials)

	// Public showcase route (no auth)
	publicShowcaseRepo := repository.NewShowcaseRepository(database.DB)
	publicShowcaseHandler := handlers.NewShowcaseHandler(publicShowcaseRepo)
	api.GET("/showcase/:slug", publicShowcaseHandler.GetPublicShowcase)
	api.GET("/showcase/:slug/uploads/*filepath", d.imageHandler.ServePublicShowcaseUpload)
}
