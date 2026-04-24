package handlers

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
)

type WebAuthnHandler struct {
	webAuthn  *webauthn.WebAuthn
	auth      *AuthHandler
	repo      *repository.WebAuthnRepository
	rpID      string
	rpOrigins []string
	sessions  map[string]*webauthn.SessionData
	sessionMu sync.RWMutex
	logger    *services.Logger
}

// webAuthnUser wraps our User model to satisfy the webauthn.User interface.
type webAuthnUser struct {
	user        models.User
	credentials []webauthn.Credential
}

func (u *webAuthnUser) WebAuthnID() []byte {
	buf := make([]byte, 4)
	buf[0] = byte(u.user.ID >> 24)
	buf[1] = byte(u.user.ID >> 16)
	buf[2] = byte(u.user.ID >> 8)
	buf[3] = byte(u.user.ID)
	return buf
}

func (u *webAuthnUser) WebAuthnName() string        { return u.user.Username }
func (u *webAuthnUser) WebAuthnDisplayName() string  { return u.user.Username }
func (u *webAuthnUser) WebAuthnCredentials() []webauthn.Credential { return u.credentials }

func NewWebAuthnHandler(rpID, rpOrigin string, authHandler *AuthHandler, repo *repository.WebAuthnRepository, logger *services.Logger) (*WebAuthnHandler, error) {
	// Support comma-separated origins
	origins := strings.Split(rpOrigin, ",")
	for i := range origins {
		origins[i] = strings.TrimSpace(origins[i])
	}

	wconfig := &webauthn.Config{
		RPDisplayName: "Ancient Coins",
		RPID:          rpID,
		RPOrigins:     origins,
		AuthenticatorSelection: protocol.AuthenticatorSelection{
			AuthenticatorAttachment: protocol.Platform,
			ResidentKey:             protocol.ResidentKeyRequirementPreferred,
			UserVerification:        protocol.VerificationPreferred,
		},
	}

	w, err := webauthn.New(wconfig)
	if err != nil {
		return nil, err
	}

	return &WebAuthnHandler{
		webAuthn:  w,
		auth:      authHandler,
		repo:      repo,
		rpID:      rpID,
		rpOrigins: origins,
		sessions:  make(map[string]*webauthn.SessionData),
		logger:    logger,
	}, nil
}

// getWebAuthnForRequest returns a WebAuthn instance configured with the request's
// actual origin. This handles cases where the app is accessed from a different
// origin than the configured default (e.g., PWA on a mobile device).
func (h *WebAuthnHandler) getWebAuthnForRequest(c *gin.Context) *webauthn.WebAuthn {
	origin := c.GetHeader("Origin")
	if origin == "" {
		// Derive origin from the request
		scheme := "https"
		if proto := c.GetHeader("X-Forwarded-Proto"); proto != "" {
			scheme = proto
		} else if c.Request.TLS == nil {
			scheme = "http"
		}
		origin = scheme + "://" + c.Request.Host
	}

	// Check if the origin is already in the configured list
	for _, o := range h.rpOrigins {
		if o == origin {
			return h.webAuthn
		}
	}

	// Origin not in configured list — create instance with this origin included
	logger := h.logger
	logger.Info("webauthn", "Request origin %q not in configured origins %v, adding dynamically", origin, h.rpOrigins)

	allOrigins := append([]string{}, h.rpOrigins...)
	allOrigins = append(allOrigins, origin)

	wconfig := &webauthn.Config{
		RPDisplayName: "Ancient Coins",
		RPID:          h.rpID,
		RPOrigins:     allOrigins,
		AuthenticatorSelection: protocol.AuthenticatorSelection{
			AuthenticatorAttachment: protocol.Platform,
			ResidentKey:             protocol.ResidentKeyRequirementPreferred,
			UserVerification:        protocol.VerificationPreferred,
		},
	}

	w, err := webauthn.New(wconfig)
	if err != nil {
		logger.Error("webauthn", "Failed to create WebAuthn with origin %q: %v", origin, err)
		return h.webAuthn
	}
	return w
}

func (h *WebAuthnHandler) loadCredentials(userID uint) []webauthn.Credential {
	creds, _ := h.repo.LoadCredentials(userID)

	result := make([]webauthn.Credential, len(creds))
	for i, c := range creds {
		credID, _ := base64.RawURLEncoding.DecodeString(c.CredentialID)
		result[i] = webauthn.Credential{
			ID:              credID,
			PublicKey:       c.PublicKey,
			AttestationType: c.AttestationType,
			Authenticator: webauthn.Authenticator{
				SignCount: c.SignCount,
			},
		}
	}
	return result
}

// RegisterBegin starts the WebAuthn registration ceremony.
//
//	@Summary		Begin WebAuthn registration
//	@Description	Starts credential registration for the authenticated user. Returns options for navigator.credentials.create().
//	@Tags			WebAuthn
//	@Produce		json
//	@Success		200	{object}	map[string]interface{}
//	@Failure		401	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/auth/webauthn/register/begin [post]
func (h *WebAuthnHandler) RegisterBegin(c *gin.Context) {
	userID := c.GetUint("userId")

	var user models.User
	found, err := h.repo.FindUserByID(userID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}
	user = *found

	wUser := &webAuthnUser{
		user:        user,
		credentials: h.loadCredentials(userID),
	}

	options, session, err := h.webAuthn.BeginRegistration(wUser)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to begin registration"})
		return
	}

	// Store session keyed by user ID
	h.sessionMu.Lock()
	h.sessions[sessionKey("reg", userID)] = session
	h.sessionMu.Unlock()

	c.JSON(http.StatusOK, options)
}

// RegisterFinish completes the WebAuthn registration ceremony.
//
//	@Summary		Finish WebAuthn registration
//	@Description	Completes credential registration. Stores the new credential for future biometric logins.
//	@Tags			WebAuthn
//	@Accept			json
//	@Produce		json
//	@Param			body	body		map[string]interface{}	true	"Credential attestation response"
//	@Success		200		{object}	map[string]interface{}
//	@Failure		400		{object}	ErrorResponse
//	@Failure		401		{object}	ErrorResponse
//	@Failure		500		{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/auth/webauthn/register/finish [post]
func (h *WebAuthnHandler) RegisterFinish(c *gin.Context) {
	userID := c.GetUint("userId")
	logger := h.logger

	var user models.User
	found, err := h.repo.FindUserByID(userID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}
	user = *found

	h.sessionMu.RLock()
	session, ok := h.sessions[sessionKey("reg", userID)]
	h.sessionMu.RUnlock()
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No registration session found"})
		return
	}

	wUser := &webAuthnUser{
		user:        user,
		credentials: h.loadCredentials(userID),
	}

	// Pre-read the body so we can restore it for go-webauthn and parse name after
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		logger.Error("webauthn", "Failed to read request body: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read request body"})
		return
	}
	c.Request.Body = io.NopCloser(bytes.NewReader(bodyBytes))

	w := h.getWebAuthnForRequest(c)
	credential, err := w.FinishRegistration(wUser, *session, c.Request)
	if err != nil {
		logger.Error("webauthn", "Registration failed for user %s: %v", user.Username, err)
		respondError(c, http.StatusBadRequest, "Registration failed", err)
		return
	}

	// Clean up session
	h.sessionMu.Lock()
	delete(h.sessions, sessionKey("reg", userID))
	h.sessionMu.Unlock()

	// Parse optional name from the pre-read body
	credName := "Biometric key"
	var bodyMap map[string]interface{}
	if json.Unmarshal(bodyBytes, &bodyMap) == nil {
		if name, ok := bodyMap["name"].(string); ok && name != "" {
			credName = name
		}
	}

	// Store credential
	dbCred := models.WebAuthnCredential{
		UserID:          userID,
		CredentialID:    base64.RawURLEncoding.EncodeToString(credential.ID),
		PublicKey:       credential.PublicKey,
		AttestationType: credential.AttestationType,
		SignCount:       credential.Authenticator.SignCount,
		Name:            credName,
	}
	if err := h.repo.CreateCredential(&dbCred); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save credential"})
		return
	}

	logger.Info("webauthn", "Credential registered for user %s", user.Username)
	c.JSON(http.StatusOK, gin.H{"status": "ok", "credential": dbCred})
}

// LoginBegin starts the WebAuthn authentication ceremony.
//
//	@Summary		Begin WebAuthn login
//	@Description	Starts biometric authentication for a user. Returns options for navigator.credentials.get().
//	@Tags			WebAuthn
//	@Accept			json
//	@Produce		json
//	@Param			body	body		map[string]interface{}	true	"Username"
//	@Success		200		{object}	map[string]interface{}
//	@Failure		400		{object}	ErrorResponse
//	@Failure		404		{object}	ErrorResponse
//	@Failure		500		{object}	ErrorResponse
//	@Router			/auth/webauthn/login/begin [post]
func (h *WebAuthnHandler) LoginBegin(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "Invalid request payload", err)
		return
	}

	var user models.User
	found, err := h.repo.FindUserByUsername(req.Username)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	user = *found

	creds := h.loadCredentials(user.ID)
	if len(creds) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No biometric credentials registered"})
		return
	}

	wUser := &webAuthnUser{
		user:        user,
		credentials: creds,
	}

	options, session, err := h.webAuthn.BeginLogin(wUser)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to begin login"})
		return
	}

	h.sessionMu.Lock()
	h.sessions[sessionKey("login", user.ID)] = session
	h.sessionMu.Unlock()

	// Include username so the frontend can pass it back
	c.JSON(http.StatusOK, gin.H{
		"options":  options,
		"username": user.Username,
	})
}

// LoginFinish completes the WebAuthn authentication and issues tokens.
//
//	@Summary		Finish WebAuthn login
//	@Description	Completes biometric authentication and returns access + refresh tokens.
//	@Tags			WebAuthn
//	@Accept			json
//	@Produce		json
//	@Param			body	body		map[string]interface{}	true	"Credential assertion response"
//	@Success		200		{object}	AuthResponse
//	@Failure		400		{object}	ErrorResponse
//	@Failure		401		{object}	ErrorResponse
//	@Failure		500		{object}	ErrorResponse
//	@Router			/auth/webauthn/login/finish [post]
func (h *WebAuthnHandler) LoginFinish(c *gin.Context) {
	logger := h.logger

	// Username is passed as a query param so we know which user's session to look up
	username := c.Query("username")
	if username == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "username query parameter required"})
		return
	}

	var user models.User
	found, err := h.repo.FindUserByUsername(username)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}
	user = *found

	h.sessionMu.RLock()
	session, ok := h.sessions[sessionKey("login", user.ID)]
	h.sessionMu.RUnlock()
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No login session found"})
		return
	}

	wUser := &webAuthnUser{
		user:        user,
		credentials: h.loadCredentials(user.ID),
	}

	w := h.getWebAuthnForRequest(c)
	credential, err := w.FinishLogin(wUser, *session, c.Request)
	if err != nil {
		logger.Error("webauthn", "Login failed for user %s: %v", username, err)
		respondError(c, http.StatusUnauthorized, "Authentication failed", err)
		return
	}

	// Clean up session
	h.sessionMu.Lock()
	delete(h.sessions, sessionKey("login", user.ID))
	h.sessionMu.Unlock()

	// Update sign count
	credID := base64.RawURLEncoding.EncodeToString(credential.ID)
	h.repo.UpdateSignCount(credID, user.ID, credential.Authenticator.SignCount)

	logger.Info("webauthn", "Biometric login succeeded for user %s", username)
	// Issue tokens
	h.auth.issueTokens(c, user, http.StatusOK)
}

// ListCredentials returns all WebAuthn credentials for the authenticated user.
//
//	@Summary		List WebAuthn credentials
//	@Description	Returns all registered biometric credentials for the authenticated user.
//	@Tags			WebAuthn
//	@Produce		json
//	@Success		200	{array}		models.WebAuthnCredential
//	@Failure		401	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/auth/webauthn/credentials [get]
func (h *WebAuthnHandler) ListCredentials(c *gin.Context) {
	userID := c.GetUint("userId")
	creds, _ := h.repo.LoadCredentials(userID)
	c.JSON(http.StatusOK, creds)
}

// DeleteCredential removes a WebAuthn credential.
//
//	@Summary		Delete WebAuthn credential
//	@Description	Removes a registered biometric credential.
//	@Tags			WebAuthn
//	@Param			id	path		int	true	"Credential ID"
//	@Success		200	{object}	map[string]interface{}
//	@Failure		401	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/auth/webauthn/credentials/{id} [delete]
func (h *WebAuthnHandler) DeleteCredential(c *gin.Context) {
	userID := c.GetUint("userId")
	credID := c.Param("id")

	rowsAffected, _ := h.repo.DeleteCredential(credID, userID)
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Credential not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

// CheckCredentials returns whether a user has registered biometric credentials.
//
//	@Summary		Check biometric availability
//	@Description	Returns whether the given username has registered WebAuthn credentials for biometric login.
//	@Tags			WebAuthn
//	@Param			username	query		string	true	"Username to check"
//	@Produce		json
//	@Success		200	{object}	map[string]interface{}
//	@Router			/auth/webauthn/check [get]
func (h *WebAuthnHandler) CheckCredentials(c *gin.Context) {
	username := c.Query("username")
	if username == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "username required"})
		return
	}

	var user models.User
	found, err := h.repo.FindUserByUsername(username)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"available": false})
		return
	}
	user = *found

	count, _ := h.repo.CountCredentials(user.ID)
	c.JSON(http.StatusOK, gin.H{"available": count > 0})
}

func sessionKey(prefix string, userID uint) string {
	return prefix + "_" + fmt.Sprint(userID)
}
