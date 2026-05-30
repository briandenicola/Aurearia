package handlers

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type AdminHandler struct {
	UploadDir   string
	repo        *repository.AdminRepository
	agentProxy  *services.AgentProxy
	settingsSvc *services.SettingsService
	logger      *services.Logger
}

func NewAdminHandler(uploadDir string, repo *repository.AdminRepository, agentProxy *services.AgentProxy, settingsSvc *services.SettingsService, logger *services.Logger) *AdminHandler {
	return &AdminHandler{UploadDir: uploadDir, repo: repo, agentProxy: agentProxy, settingsSvc: settingsSvc, logger: logger}
}

// AdminRequired middleware ensures only admin users can access
func AdminRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, _ := c.Get("userRole")
		if role != string(models.RoleAdmin) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
			return
		}
		c.Next()
	}
}

// ListUsers returns all users for admin management.
//
//	@Summary		List all users
//	@Description	Returns a list of all registered users. Admin only.
//	@Tags			Admin
//	@Produce		json
//	@Success		200	{array}		UserDTO
//	@Failure		401	{object}	ErrorResponse
//	@Failure		403	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/admin/users [get]
func (h *AdminHandler) ListUsers(c *gin.Context) {
	users, _ := h.repo.ListUsers()

	var result []UserDTO
	for _, u := range users {
		result = append(result, UserDTO{
			ID:        u.ID,
			Username:  u.Username,
			Role:      u.Role,
			CreatedAt: u.CreatedAt.Format("2006-01-02T15:04:05Z"),
		})
	}

	c.JSON(http.StatusOK, result)
}

// DeleteUser removes a user and their coins.
//
//	@Summary		Delete a user
//	@Description	Deletes a user and all their associated coins and images. Cannot delete yourself. Admin only.
//	@Tags			Admin
//	@Produce		json
//	@Param			id	path		int	true	"User ID"
//	@Success		200	{object}	MessageResponse
//	@Failure		400	{object}	ErrorResponse
//	@Failure		401	{object}	ErrorResponse
//	@Failure		403	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/admin/users/{id} [delete]
func (h *AdminHandler) DeleteUser(c *gin.Context) {
	adminID := c.GetUint("userId")
	targetID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	if uint(targetID) == adminID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot delete your own account"})
		return
	}

	// Delete user's coin images, coins, then user
	rowsAffected, _ := h.repo.DeleteUserCascade(uint(targetID))
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User deleted"})
}

// ResetPassword allows admin to set a new password for a user.
//
//	@Summary		Reset user password
//	@Description	Sets a new password for the specified user. Admin only.
//	@Tags			Admin
//	@Accept			json
//	@Produce		json
//	@Param			id		path		int						true	"User ID"
//	@Param			body	body		ResetPasswordRequest	true	"New password"
//	@Success		200		{object}	MessageResponse
//	@Failure		400		{object}	ErrorResponse
//	@Failure		401		{object}	ErrorResponse
//	@Failure		403		{object}	ErrorResponse
//	@Failure		404		{object}	ErrorResponse
//	@Failure		500		{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/admin/users/{id}/reset-password [post]
func (h *AdminHandler) ResetPassword(c *gin.Context) {
	targetID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	var req struct {
		NewPassword string `json:"newPassword" binding:"required,min=6"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "Invalid request payload", err)
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	result, _ := h.repo.ResetPassword(uint(targetID), string(hash))
	if result == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Password reset"})
}

// UpdateUserRole allows admin to change a user's role.
//
//	@Summary		Update user role
//	@Description	Updates the role for the specified user. Cannot update your own role. Admin only.
//	@Tags			Admin
//	@Accept			json
//	@Produce		json
//	@Param			id		path		int						true	"User ID"
//	@Param			body	body		UpdateUserRoleRequest	true	"Role update payload"
//	@Success		200		{object}	map[string]string
//	@Failure		400		{object}	ErrorResponse
//	@Failure		401		{object}	ErrorResponse
//	@Failure		403		{object}	ErrorResponse
//	@Failure		404		{object}	ErrorResponse
//	@Failure		500		{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/admin/users/{id}/role [put]
func (h *AdminHandler) UpdateUserRole(c *gin.Context) {
	adminID := c.GetUint("userId")
	targetID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	if uint(targetID) == adminID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot change your own role"})
		return
	}

	var req struct {
		Role models.UserRole `json:"role" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "Invalid request payload", err)
		return
	}

	if req.Role != models.RoleAdmin && req.Role != models.RoleUser {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role"})
		return
	}

	result, err := h.repo.UpdateUserRole(uint(targetID), req.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user role"})
		return
	}
	if result == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Role updated", "role": req.Role})
}

// GetSettings returns all app settings.
//
//	@Summary		Get application settings
//	@Description	Returns all application settings merged with defaults. Admin only.
//	@Tags			Admin
//	@Produce		json
//	@Success		200	{object}	map[string]string
//	@Failure		401	{object}	ErrorResponse
//	@Failure		403	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/admin/settings [get]
func (h *AdminHandler) GetSettings(c *gin.Context) {
	settings := h.settingsSvc.GetAllSettings()
	c.JSON(http.StatusOK, settings)
}

// GetSettingDefaults returns the built-in default values for all settings.
//
//	@Summary		Get setting defaults
//	@Description	Returns the built-in default values for all application settings. Admin only.
//	@Tags			Admin
//	@Produce		json
//	@Success		200	{object}	map[string]string
//	@Failure		401	{object}	ErrorResponse
//	@Failure		403	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/admin/settings/defaults [get]
func (h *AdminHandler) GetSettingDefaults(c *gin.Context) {
	c.JSON(http.StatusOK, h.settingsSvc.GetSettingDefaults())
}

// UpdateSettings updates app settings.
//
//	@Summary		Update application settings
//	@Description	Updates one or more application settings. Syncs log level if changed. Admin only.
//	@Tags			Admin
//	@Accept			json
//	@Produce		json
//	@Param			body	body		[]SettingInput	true	"Settings to update"
//	@Success		200		{object}	SettingsUpdateResponse
//	@Failure		400		{object}	ErrorResponse
//	@Failure		401		{object}	ErrorResponse
//	@Failure		403		{object}	ErrorResponse
//	@Failure		500		{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/admin/settings [put]
func (h *AdminHandler) UpdateSettings(c *gin.Context) {
	var settings []struct {
		Key   string `json:"key" binding:"required"`
		Value string `json:"value"`
	}
	if err := c.ShouldBindJSON(&settings); err != nil {
		respondError(c, http.StatusBadRequest, "Invalid request payload", err)
		return
	}

	for _, s := range settings {
		if err := h.settingsSvc.SetSetting(s.Key, s.Value); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save setting: " + s.Key})
			return
		}
	}

	// Sync log level if it was updated
	h.settingsSvc.SyncLogLevel(h.logger)

	// Push log level to Python agent service
	if h.agentProxy != nil {
		logLevel := h.settingsSvc.GetSetting(services.SettingLogLevel)
		h.agentProxy.SetLogLevel(c.Request.Context(), logLevel)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Settings updated"})
}

// GetLogs returns recent application logs.
//
//	@Summary		Get application logs
//	@Description	Returns recent in-memory application logs, optionally filtered by level. Admin only.
//	@Tags			Admin
//	@Produce		json
//	@Param			limit	query		int		false	"Maximum number of log entries"	default(500)
//	@Param			level	query		string	false	"Filter by log level"	Enums(trace, debug, info, warn, error)
//	@Success		200		{object}	LogsResponse
//	@Failure		401		{object}	ErrorResponse
//	@Failure		403		{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/admin/logs [get]
func (h *AdminHandler) GetLogs(c *gin.Context) {
	limit := 500
	if l := c.Query("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			limit = n
		}
	}

	level := c.Query("level")
	logs := h.logger.GetLogs(limit)

	if level != "" {
		filtered := make([]services.LogEntry, 0)
		for _, entry := range logs {
			if entry.Level == level {
				filtered = append(filtered, entry)
			}
		}
		logs = filtered
	}

	// Merge agent service logs if available
	if h.agentProxy != nil {
		agentLogs := h.agentProxy.FetchLogs(c.Request.Context(), limit, level)
		logs = mergeLogsByTimestamp(logs, agentLogs)
	}

	c.JSON(http.StatusOK, gin.H{
		"logs":     logs,
		"count":    len(logs),
		"logLevel": h.logger.GetLevel(),
	})
}

// mergeLogsByTimestamp merges two sorted log slices by timestamp (oldest first).
func mergeLogsByTimestamp(a, b []services.LogEntry) []services.LogEntry {
	merged := make([]services.LogEntry, 0, len(a)+len(b))
	i, j := 0, 0
	for i < len(a) && j < len(b) {
		if a[i].Timestamp <= b[j].Timestamp {
			merged = append(merged, a[i])
			i++
		} else {
			merged = append(merged, b[j])
			j++
		}
	}
	merged = append(merged, a[i:]...)
	merged = append(merged, b[j:]...)
	return merged
}

// TestAnthropicConnection validates the Anthropic API key by listing models.
func (h *AdminHandler) TestAnthropicConnection(c *gin.Context) {
	apiKey := h.settingsSvc.GetSetting(services.SettingAnthropicAPIKey)
	if apiKey == "" {
		c.JSON(http.StatusOK, gin.H{"available": false, "message": "Anthropic API key is not configured"})
		return
	}

	req, err := http.NewRequestWithContext(c.Request.Context(), "GET", "https://api.anthropic.com/v1/models", nil)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"available": false, "message": "Failed to create request"})
		return
	}
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		log.Printf("[handler] TestAnthropicConnection: %v", err)
		c.JSON(http.StatusOK, gin.H{"available": false, "message": "Connection failed"})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		c.JSON(http.StatusOK, gin.H{"available": true, "message": "Anthropic API key is valid"})
	} else {
		body, _ := io.ReadAll(resp.Body)
		msg := string(body)
		if len(msg) > 200 {
			msg = msg[:200]
		}
		c.JSON(http.StatusOK, gin.H{"available": false, "message": fmt.Sprintf("API returned HTTP %d: %s", resp.StatusCode, msg)})
	}
}

// TestSearXNGConnection validates the SearXNG endpoint is reachable.
func (h *AdminHandler) TestSearXNGConnection(c *gin.Context) {
	searxngURL := h.settingsSvc.GetSetting(services.SettingSearXNGURL)
	if searxngURL == "" {
		c.JSON(http.StatusOK, gin.H{"available": false, "message": "SearXNG URL is not configured"})
		return
	}

	req, err := http.NewRequestWithContext(c.Request.Context(), "GET", searxngURL, nil)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"available": false, "message": "Invalid URL"})
		return
	}

	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		log.Printf("[handler] TestSearXNGConnection: %v", err)
		c.JSON(http.StatusOK, gin.H{"available": false, "message": "Connection failed"})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		c.JSON(http.StatusOK, gin.H{"available": true, "message": fmt.Sprintf("SearXNG is reachable at %s", searxngURL)})
	} else {
		c.JSON(http.StatusOK, gin.H{"available": false, "message": fmt.Sprintf("SearXNG returned HTTP %d", resp.StatusCode)})
	}
}
