package handlers

import (
	"errors"
	"net/http"
	"strings"

	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
)

type OIDCHandler struct {
	svc *services.OIDCService
}

func NewOIDCHandler(svc *services.OIDCService) *OIDCHandler {
	return &OIDCHandler{svc: svc}
}

type oidcAdminProviderListResponse struct {
	Providers []services.OIDCAdminProviderDTO `json:"providers"`
}

type oidcPublicProviderListResponse struct {
	Providers []services.OIDCPublicProviderDTO `json:"providers"`
}

type oidcLinkedIdentityListResponse struct {
	Identities []services.OIDCLinkedIdentityDTO `json:"identities"`
}

// ListPublicProviders returns enabled OIDC providers for login.
//
//	@Summary		List public OIDC providers
//	@Description	Returns enabled OIDC providers safe for unauthenticated login UI. Secrets, issuer URLs, and client IDs are omitted.
//	@Tags			OIDC
//	@Produce		json
//	@Success		200	{object}	oidcPublicProviderListResponse
//	@Failure		500	{object}	ErrorResponse
//	@Router			/auth/oidc/providers [get]
func (h *OIDCHandler) ListPublicProviders(c *gin.Context) {
	providers, err := h.svc.ListPublicProviders()
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to list OIDC providers", err)
		return
	}
	c.JSON(http.StatusOK, oidcPublicProviderListResponse{Providers: providers})
}

// StartLogin starts the OIDC authorization-code + PKCE login flow.
//
//	@Summary		Start OIDC login
//	@Description	Creates short-lived state with PKCE and nonce and returns the provider authorization URL.
//	@Tags			OIDC
//	@Accept			json
//	@Produce		json
//	@Param			providerId	path		int							true	"Provider ID"
//	@Param			body		body		services.OIDCStartLoginInput	true	"Login start payload"
//	@Success		200			{object}	services.OIDCStartLoginResult
//	@Failure		400			{object}	ErrorResponse
//	@Failure		404			{object}	ErrorResponse
//	@Failure		409			{object}	ErrorResponse
//	@Failure		500			{object}	ErrorResponse
//	@Router			/auth/oidc/{providerId}/start [post]
func (h *OIDCHandler) StartLogin(c *gin.Context) {
	id, ok := parseID(c, "providerId")
	if !ok {
		return
	}
	var body services.OIDCStartLoginInput
	if err := c.ShouldBindJSON(&body); err != nil && !errors.Is(err, http.ErrBodyNotAllowed) {
		respondError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.svc.StartLogin(c.Request.Context(), id, body.RedirectPath, oidcRequestOrigin(c))
	if err != nil {
		h.handleOIDCError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

// Callback completes OIDC login and returns the existing AuthResponse JSON.
//
//	@Summary		Complete OIDC login
//	@Description	Exchanges the provider code, validates the ID token, finds a linked identity, and returns app JWT/refresh tokens in the JSON body. Tokens are never placed in URL query strings.
//	@Tags			OIDC
//	@Produce		json
//	@Param			providerId	path		int		true	"Provider ID"
//	@Param			code		query		string	true	"Authorization code"
//	@Param			state		query		string	true	"Opaque state"
//	@Success		200			{object}	AuthResponse
//	@Failure		400			{object}	ErrorResponse
//	@Failure		401			{object}	ErrorResponse
//	@Failure		404			{object}	ErrorResponse
//	@Failure		409			{object}	ErrorResponse
//	@Failure		500			{object}	ErrorResponse
//	@Router			/auth/oidc/{providerId}/callback [get]
func (h *OIDCHandler) Callback(c *gin.Context) {
	noStore(c)
	id, ok := parseID(c, "providerId")
	if !ok {
		return
	}
	if c.Query("error") != "" {
		h.svc.RecordLoginFailure(id, oidcAuditContext(c), "provider denied login")
		h.handleOIDCError(c, services.ErrOIDCProviderDenied)
		return
	}
	result, err := h.svc.CompleteLoginCallback(c.Request.Context(), id, c.Query("code"), c.Query("state"), oidcRequestOrigin(c), oidcAuditContext(c))
	if err != nil {
		h.handleOIDCError(c, err)
		return
	}
	writeAuthResponse(c, http.StatusOK, result)
}

// StartLink starts a protected OIDC account-linking flow.
//
//	@Summary		Start OIDC account link
//	@Description	Creates short-lived link state with PKCE and nonce for the authenticated user and returns the provider authorization URL.
//	@Tags			OIDC
//	@Accept			json
//	@Produce		json
//	@Param			providerId	path		int							true	"Provider ID"
//	@Param			body		body		services.OIDCStartLoginInput	true	"Link start payload"
//	@Success		200			{object}	services.OIDCStartLoginResult
//	@Failure		400			{object}	ErrorResponse
//	@Failure		401			{object}	ErrorResponse
//	@Failure		404			{object}	ErrorResponse
//	@Failure		409			{object}	ErrorResponse
//	@Failure		500			{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/auth/oidc/{providerId}/link/start [post]
func (h *OIDCHandler) StartLink(c *gin.Context) {
	id, ok := parseID(c, "providerId")
	if !ok {
		return
	}
	var body services.OIDCStartLoginInput
	if err := c.ShouldBindJSON(&body); err != nil && !errors.Is(err, http.ErrBodyNotAllowed) {
		respondError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.svc.StartLink(c.Request.Context(), id, c.GetUint("userId"), body.RedirectPath, oidcRequestOrigin(c))
	if err != nil {
		h.handleOIDCError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

// LinkCallback completes OIDC account linking.
//
//	@Summary		Complete OIDC account link
//	@Description	Exchanges the provider code, validates the ID token, and links the external identity to the user recorded in state.
//	@Tags			OIDC
//	@Produce		json
//	@Param			providerId	path		int		true	"Provider ID"
//	@Param			code		query		string	true	"Authorization code"
//	@Param			state		query		string	true	"Opaque state"
//	@Success		200			{object}	services.OIDCLinkCallbackResult
//	@Failure		400			{object}	ErrorResponse
//	@Failure		401			{object}	ErrorResponse
//	@Failure		404			{object}	ErrorResponse
//	@Failure		409			{object}	ErrorResponse
//	@Failure		500			{object}	ErrorResponse
//	@Router			/auth/oidc/{providerId}/link/callback [get]
func (h *OIDCHandler) LinkCallback(c *gin.Context) {
	noStore(c)
	id, ok := parseID(c, "providerId")
	if !ok {
		return
	}
	if c.Query("error") != "" {
		h.handleOIDCError(c, services.ErrOIDCProviderDenied)
		return
	}
	result, err := h.svc.CompleteLinkCallback(c.Request.Context(), id, c.Query("code"), c.Query("state"), oidcRequestOrigin(c), oidcAuditContext(c))
	if err != nil {
		h.handleOIDCError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

// ListLinkedIdentities lists OIDC identities linked to the authenticated user.
//
//	@Summary		List linked OIDC identities
//	@Description	Returns linked OIDC identities for the current user with subject-safe previews.
//	@Tags			OIDC
//	@Produce		json
//	@Success		200	{object}	oidcLinkedIdentityListResponse
//	@Failure		401	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/user/oidc-identities [get]
func (h *OIDCHandler) ListLinkedIdentities(c *gin.Context) {
	identities, err := h.svc.ListLinkedIdentities(c.GetUint("userId"))
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to list OIDC identities", err)
		return
	}
	c.JSON(http.StatusOK, oidcLinkedIdentityListResponse{Identities: identities})
}

// UnlinkIdentity unlinks an OIDC identity from the authenticated user.
//
//	@Summary		Unlink OIDC identity
//	@Description	Unlinks an OIDC identity unless it would leave the account without a usable sign-in method.
//	@Tags			OIDC
//	@Produce		json
//	@Param			identityId	path		int	true	"Identity ID"
//	@Success		200			{object}	MessageResponse
//	@Failure		400			{object}	ErrorResponse
//	@Failure		401			{object}	ErrorResponse
//	@Failure		404			{object}	ErrorResponse
//	@Failure		409			{object}	ErrorResponse
//	@Failure		500			{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/user/oidc-identities/{identityId} [delete]
func (h *OIDCHandler) UnlinkIdentity(c *gin.Context) {
	id, ok := parseID(c, "identityId")
	if !ok {
		return
	}
	if err := h.svc.UnlinkIdentity(id, c.GetUint("userId"), oidcAuditContext(c)); err != nil {
		h.handleOIDCError(c, err)
		return
	}
	c.JSON(http.StatusOK, MessageResponse{Message: "OIDC identity unlinked"})
}

// ListAdminProviders returns all OIDC providers for admin configuration.
//
//	@Summary		List admin OIDC providers
//	@Description	Returns all configured OIDC providers with client secrets redacted.
//	@Tags			OIDC
//	@Produce		json
//	@Success		200	{object}	oidcAdminProviderListResponse
//	@Failure		401	{object}	ErrorResponse
//	@Failure		403	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/admin/oidc/providers [get]
func (h *OIDCHandler) ListAdminProviders(c *gin.Context) {
	providers, err := h.svc.ListAdminProviders()
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to list OIDC providers", err)
		return
	}
	c.JSON(http.StatusOK, oidcAdminProviderListResponse{Providers: providers})
}

// CreateAdminProvider creates an OIDC provider.
//
//	@Summary		Create OIDC provider
//	@Description	Creates an admin-managed OIDC provider. Client secrets are write-only.
//	@Tags			OIDC
//	@Accept			json
//	@Produce		json
//	@Param			body	body		services.OIDCAdminProviderInput	true	"OIDC provider"
//	@Success		201		{object}	services.OIDCAdminProviderDTO
//	@Failure		400		{object}	ErrorResponse
//	@Failure		401		{object}	ErrorResponse
//	@Failure		403		{object}	ErrorResponse
//	@Failure		409		{object}	ErrorResponse
//	@Failure		500		{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/admin/oidc/providers [post]
func (h *OIDCHandler) CreateAdminProvider(c *gin.Context) {
	var body services.OIDCAdminProviderInput
	if err := c.ShouldBindJSON(&body); err != nil {
		respondError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	provider, err := h.svc.CreateAdminProvider(c.Request.Context(), body, oidcAuditContext(c))
	if err != nil {
		h.handleOIDCError(c, err)
		return
	}
	c.JSON(http.StatusCreated, provider)
}

// UpdateAdminProvider updates an OIDC provider.
//
//	@Summary		Update OIDC provider
//	@Description	Updates an admin-managed OIDC provider. Empty or omitted clientSecret preserves the existing secret.
//	@Tags			OIDC
//	@Accept			json
//	@Produce		json
//	@Param			providerId	path		int								true	"Provider ID"
//	@Param			body		body		services.OIDCAdminProviderInput	true	"OIDC provider"
//	@Success		200			{object}	services.OIDCAdminProviderDTO
//	@Failure		400			{object}	ErrorResponse
//	@Failure		401			{object}	ErrorResponse
//	@Failure		403			{object}	ErrorResponse
//	@Failure		404			{object}	ErrorResponse
//	@Failure		409			{object}	ErrorResponse
//	@Failure		500			{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/admin/oidc/providers/{providerId} [put]
func (h *OIDCHandler) UpdateAdminProvider(c *gin.Context) {
	id, ok := parseID(c, "providerId")
	if !ok {
		return
	}
	var body services.OIDCAdminProviderInput
	if err := c.ShouldBindJSON(&body); err != nil {
		respondError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	provider, err := h.svc.UpdateAdminProvider(c.Request.Context(), id, body, oidcAuditContext(c))
	if err != nil {
		h.handleOIDCError(c, err)
		return
	}
	c.JSON(http.StatusOK, provider)
}

// DeleteAdminProvider deletes an OIDC provider when it has no linked identities.
//
//	@Summary		Delete OIDC provider
//	@Description	Deletes an OIDC provider only when no external identities reference it.
//	@Tags			OIDC
//	@Produce		json
//	@Param			providerId	path		int	true	"Provider ID"
//	@Success		200			{object}	MessageResponse
//	@Failure		400			{object}	ErrorResponse
//	@Failure		401			{object}	ErrorResponse
//	@Failure		403			{object}	ErrorResponse
//	@Failure		404			{object}	ErrorResponse
//	@Failure		409			{object}	ErrorResponse
//	@Failure		500			{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/admin/oidc/providers/{providerId} [delete]
func (h *OIDCHandler) DeleteAdminProvider(c *gin.Context) {
	id, ok := parseID(c, "providerId")
	if !ok {
		return
	}
	if err := h.svc.DeleteAdminProvider(id, oidcAuditContext(c)); err != nil {
		h.handleOIDCError(c, err)
		return
	}
	c.JSON(http.StatusOK, MessageResponse{Message: "OIDC provider deleted successfully"})
}

// TestAdminProvider validates OIDC discovery metadata for a provider.
//
//	@Summary		Test OIDC provider
//	@Description	Validates provider discovery metadata and records safe status without exposing secrets.
//	@Tags			OIDC
//	@Produce		json
//	@Param			providerId	path		int	true	"Provider ID"
//	@Success		200			{object}	services.OIDCProviderTestResult
//	@Failure		400			{object}	ErrorResponse
//	@Failure		401			{object}	ErrorResponse
//	@Failure		403			{object}	ErrorResponse
//	@Failure		404			{object}	ErrorResponse
//	@Failure		500			{object}	services.OIDCProviderTestResult
//	@Security		BearerAuth
//	@Router			/admin/oidc/providers/{providerId}/test [post]
func (h *OIDCHandler) TestAdminProvider(c *gin.Context) {
	id, ok := parseID(c, "providerId")
	if !ok {
		return
	}
	result, err := h.svc.TestAdminProvider(c.Request.Context(), id, oidcAuditContext(c))
	if err != nil {
		if errors.Is(err, services.ErrOIDCProviderDiscovery) {
			c.JSON(http.StatusOK, result)
			return
		}
		h.handleOIDCError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *OIDCHandler) handleOIDCError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, services.ErrOIDCProviderNotFound):
		respondError(c, http.StatusNotFound, "OIDC provider not found", err)
	case errors.Is(err, services.ErrOIDCProviderInvalid):
		respondError(c, http.StatusBadRequest, "Invalid OIDC provider configuration", err)
	case errors.Is(err, services.ErrOIDCProviderSecretMissing):
		respondError(c, http.StatusBadRequest, "OIDC client secret is required", err)
	case errors.Is(err, services.ErrOIDCProviderConfiguration):
		respondError(c, http.StatusInternalServerError, "OIDC provider is misconfigured", err)
	case errors.Is(err, services.ErrOIDCProviderDenied):
		respondError(c, http.StatusBadRequest, "OIDC provider denied access", err)
	case errors.Is(err, services.ErrOIDCProviderDuplicate):
		respondError(c, http.StatusConflict, "OIDC provider already exists", err)
	case errors.Is(err, services.ErrOIDCProviderInUse):
		respondError(c, http.StatusConflict, "OIDC provider has linked identities", err)
	case errors.Is(err, services.ErrOIDCProviderDisabled):
		respondError(c, http.StatusConflict, "OIDC provider is disabled", err)
	case errors.Is(err, services.ErrOIDCInvalidRedirect):
		respondError(c, http.StatusBadRequest, "Invalid redirect path", err)
	case errors.Is(err, services.ErrOIDCInvalidState):
		respondError(c, http.StatusBadRequest, "Invalid OIDC state", err)
	case errors.Is(err, services.ErrOIDCCodeExchangeFailed):
		respondError(c, http.StatusBadRequest, "OIDC authorization code was rejected", err)
	case errors.Is(err, services.ErrOIDCValidationFailed):
		respondError(c, http.StatusUnauthorized, "OIDC validation failed", err)
	case errors.Is(err, services.ErrOIDCIdentityNotLinked):
		respondError(c, http.StatusUnauthorized, "OIDC identity is not linked", err)
	case errors.Is(err, services.ErrOIDCIdentityNotFound):
		respondError(c, http.StatusNotFound, "OIDC identity not found", err)
	case errors.Is(err, services.ErrOIDCIdentityAlreadyLinked):
		respondError(c, http.StatusConflict, "OIDC identity is already linked to another account", err)
	case errors.Is(err, services.ErrOIDCAccountConflict):
		respondError(c, http.StatusConflict, "Sign in locally and link this OIDC identity from Account Settings", err)
	case errors.Is(err, services.ErrOIDCNoUsableSignInMethod):
		respondError(c, http.StatusConflict, "Cannot unlink the last usable sign-in method", err)
	case errors.Is(err, services.ErrOIDCTokenIssueFailed):
		respondError(c, http.StatusInternalServerError, "Failed to issue app session", err)
	default:
		respondError(c, http.StatusInternalServerError, "Failed to process OIDC provider request", err)
	}
}

func oidcAuditContext(c *gin.Context) services.OIDCAuditContext {
	adminID, _ := c.Get("userId")
	id, _ := adminID.(uint)
	return services.OIDCAuditContext{
		AdminID:   id,
		ClientIP:  c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
	}
}

func oidcRequestOrigin(c *gin.Context) string {
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	if forwardedProto := strings.TrimSpace(c.GetHeader("X-Forwarded-Proto")); forwardedProto == "https" || forwardedProto == "http" {
		scheme = forwardedProto
	}
	host := c.Request.Host
	if forwardedHost := strings.TrimSpace(c.GetHeader("X-Forwarded-Host")); forwardedHost != "" {
		host = forwardedHost
	}
	return scheme + "://" + host
}
