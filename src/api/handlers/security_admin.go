package handlers

import (
	"errors"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
)

type SecurityExposureConfig struct {
	PublicAppURL             string
	WebAuthnOrigin           string
	CORSOrigins              []string
	TrustedProxiesConfigured bool
	AgentInternalTokenSet    bool
	RegistrationMode         string
	BackupStatus             string
}

type SecurityAdminHandler struct {
	securitySvc *services.SecurityService
	settingsSvc *services.SettingsService
	cfg         SecurityExposureConfig
}

type IPRuleCreateRequest struct {
	CIDR            string `json:"cidr" binding:"required"`
	Reason          string `json:"reason"`
	ExpiresAt       string `json:"expiresAt"`
	DurationMinutes int    `json:"durationMinutes"`
}

func NewSecurityAdminHandler(securitySvc *services.SecurityService, settingsSvc *services.SettingsService, cfg SecurityExposureConfig) *SecurityAdminHandler {
	return &SecurityAdminHandler{securitySvc: securitySvc, settingsSvc: settingsSvc, cfg: cfg}
}

// SecuritySummary returns a 24h auth/security event summary.
//
//	@Summary		Get security summary
//	@Description	Returns aggregate auth/security counters and backup status. Admin only.
//	@Tags			Admin Security
//	@Produce		json
//	@Success		200	{object}	map[string]interface{}
//	@Failure		500	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/admin/security/summary [get]
func (h *SecurityAdminHandler) SecuritySummary(c *gin.Context) {
	summary, err := h.securitySvc.Summary()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load security summary"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"summary": summary, "backupStatus": h.settingsSvc.GetSetting(services.SettingBackupStatus)})
}

// SecurityEvents returns persisted auth/security events.
//
//	@Summary		List security events
//	@Description	Returns persisted auth/security events with optional type, username, clientIp, and limit filters. Admin only.
//	@Tags			Admin Security
//	@Produce		json
//	@Param			type		query		string	false	"Event type"
//	@Param			username	query		string	false	"Username"
//	@Param			clientIp	query		string	false	"Client IP"
//	@Param			limit		query		int		false	"Maximum events"	default(100)
//	@Success		200			{object}	map[string]interface{}
//	@Failure		500			{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/admin/security/events [get]
func (h *SecurityAdminHandler) SecurityEvents(c *gin.Context) {
	limit := 100
	if raw := c.Query("limit"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 && n <= 500 {
			limit = n
		}
	}
	events, err := h.securitySvc.ListEvents(repository.SecurityEventFilters{
		Type:     c.Query("type"),
		Username: c.Query("username"),
		ClientIP: c.Query("clientIp"),
		Limit:    limit,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load security events"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"events": events, "count": len(events)})
}

// ListIPRules returns admin-managed IP/CIDR deny rules.
//
//	@Summary		List IP deny rules
//	@Description	Returns active IP/CIDR deny rules by default. Admin only.
//	@Tags			Admin Security
//	@Produce		json
//	@Param			includeExpired	query		bool	false	"Include expired rules"
//	@Success		200				{object}	map[string]interface{}
//	@Failure		500				{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/admin/security/ip-rules [get]
func (h *SecurityAdminHandler) ListIPRules(c *gin.Context) {
	rules, err := h.securitySvc.ListIPRules(c.Query("includeExpired") == "true")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load IP rules"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ipRules": rules})
}

// CreateIPRule creates an admin-managed IP/CIDR deny rule.
//
//	@Summary		Create IP deny rule
//	@Description	Creates an IP/CIDR deny rule with optional expiration. Admin only.
//	@Tags			Admin Security
//	@Accept			json
//	@Produce		json
//	@Param			body	body		IPRuleCreateRequest	true	"IP rule"
//	@Success		201		{object}	MessageResponse
//	@Failure		400		{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/admin/security/ip-rules [post]
func (h *SecurityAdminHandler) CreateIPRule(c *gin.Context) {
	var req IPRuleCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "Invalid request payload", err)
		return
	}
	var expires time.Time
	if req.ExpiresAt != "" {
		parsed, err := time.Parse(time.RFC3339, req.ExpiresAt)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid expiresAt"})
			return
		}
		expires = parsed
	} else if req.DurationMinutes > 0 {
		expires = time.Now().Add(time.Duration(req.DurationMinutes) * time.Minute)
	}
	adminID := c.GetUint("userId")
	if err := h.securitySvc.CreateIPRule(req.CIDR, req.Reason, expires, &adminID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid IP rule"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "IP rule created"})
}

// DeleteIPRule removes an IP/CIDR deny rule.
//
//	@Summary		Delete IP deny rule
//	@Description	Deletes an IP/CIDR deny rule. Admin only.
//	@Tags			Admin Security
//	@Produce		json
//	@Param			id	path		int	true	"IP rule ID"
//	@Success		200	{object}	MessageResponse
//	@Failure		400	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/admin/security/ip-rules/{id} [delete]
func (h *SecurityAdminHandler) DeleteIPRule(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid IP rule ID"})
		return
	}
	if err := h.securitySvc.DeleteIPRule(uint(id), c.GetUint("userId")); err != nil {
		if errors.Is(err, services.ErrIPRuleNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "IP rule not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete IP rule"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "IP rule deleted"})
}

// UnlockUser clears a persisted account lockout.
//
//	@Summary		Unlock user
//	@Description	Clears a user's account lock state. Admin only.
//	@Tags			Admin Security
//	@Produce		json
//	@Param			id	path		int	true	"User ID"
//	@Success		200	{object}	MessageResponse
//	@Failure		400	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/admin/users/{id}/unlock [post]
func (h *SecurityAdminHandler) UnlockUser(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}
	if err := h.securitySvc.UnlockUser(uint(id), c.GetUint("userId")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to unlock user"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "User unlocked"})
}

// ExposureCheck returns public-facing configuration warnings.
//
//	@Summary		Get exposure check
//	@Description	Returns release-readiness warnings for public-facing configuration. Admin only.
//	@Tags			Admin Security
//	@Produce		json
//	@Success		200	{object}	map[string]interface{}
//	@Security		BearerAuth
//	@Router			/admin/security/exposure-check [get]
func (h *SecurityAdminHandler) ExposureCheck(c *gin.Context) {
	mode := h.settingsSvc.GetSetting(services.SettingRegistrationMode)
	publicURL := h.settingsSvc.GetSetting(services.SettingPublicAppURL)
	warnings := make([]string, 0)
	if publicURL == "" {
		warnings = append(warnings, "PublicAppURL is blank")
	} else if _, err := url.ParseRequestURI(publicURL); err != nil {
		warnings = append(warnings, "PublicAppURL is invalid")
	}
	for _, origin := range strings.Split(h.cfg.WebAuthnOrigin, ",") {
		origin = strings.TrimSpace(origin)
		if origin != "" && !strings.HasPrefix(origin, "https://") && !strings.Contains(origin, "localhost") {
			warnings = append(warnings, "WebAuthn origin should use HTTPS")
			break
		}
	}
	if os.Getenv("GIN_MODE") == "release" {
		if len(h.cfg.CORSOrigins) == 0 {
			warnings = append(warnings, "CORS origins are not explicitly configured")
		}
		for _, origin := range h.cfg.CORSOrigins {
			if origin == "*" || strings.Contains(origin, "localhost") {
				warnings = append(warnings, "CORS origin is not release-safe: "+origin)
			}
		}
		if !h.cfg.TrustedProxiesConfigured {
			warnings = append(warnings, "Trusted proxies are not configured")
		}
	}
	if mode != "closed" {
		warnings = append(warnings, "Registration mode is "+mode)
	}
	c.JSON(http.StatusOK, gin.H{
		"warnings": warnings,
		"config": gin.H{
			"publicAppURL":             publicURL,
			"webauthnOrigin":           h.cfg.WebAuthnOrigin,
			"trustedProxiesConfigured": h.cfg.TrustedProxiesConfigured,
			"agentInternalTokenSet":    h.cfg.AgentInternalTokenSet,
			"registrationMode":         mode,
			"backupStatus":             h.settingsSvc.GetSetting(services.SettingBackupStatus),
		},
	})
}
