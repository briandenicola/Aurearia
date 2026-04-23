package handlers

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strconv"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
)

type ApiKeyHandler struct {
	repo *repository.ApiKeyRepository
}

func NewApiKeyHandler(repo *repository.ApiKeyRepository) *ApiKeyHandler {
	return &ApiKeyHandler{repo: repo}
}

type generateApiKeyRequest struct {
	Name string `json:"name" binding:"required,max=100" example:"My Script"`
}

type generateApiKeyResponse struct {
	Key    string        `json:"key" example:"ak_a1b2c3d4e5f6..."`
	ApiKey models.ApiKey `json:"apiKey"`
}

// Generate creates a new API key for the authenticated user.
//
//	@Summary		Generate an API key
//	@Description	Creates a new API key. The key is returned once and cannot be retrieved again.
//	@Tags			API Keys
//	@Accept			json
//	@Produce		json
//	@Param			body	body		generateApiKeyRequest	true	"Key name for identification"
//	@Success		201		{object}	generateApiKeyResponse
//	@Failure		400		{object}	ErrorResponse
//	@Failure		401		{object}	ErrorResponse
//	@Failure		500		{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/auth/api-keys [post]
func (h *ApiKeyHandler) Generate(c *gin.Context) {
	logger := services.AppLogger
	userID := c.GetUint("userId")

	var req generateApiKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "Invalid request payload", err)
		return
	}

	// Generate a random 32-byte key
	rawKey := make([]byte, 32)
	if _, err := rand.Read(rawKey); err != nil {
		logger.Error("api-keys", "Failed to generate random key: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate key"})
		return
	}

	plainKey := "ak_" + hex.EncodeToString(rawKey)

	// Hash for storage
	hash := sha256.Sum256([]byte(plainKey))
	keyHash := hex.EncodeToString(hash[:])

	// Last 8 chars as prefix for display
	keyPrefix := plainKey[len(plainKey)-8:]

	apiKey := models.ApiKey{
		UserID:    userID,
		KeyHash:   keyHash,
		KeyPrefix: keyPrefix,
		Name:      req.Name,
	}

	if err := h.repo.Create(&apiKey); err != nil {
		logger.Error("api-keys", "Failed to save API key: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create API key"})
		return
	}

	logger.Info("api-keys", "Generated API key '%s' (prefix: ...%s) for user %d", req.Name, keyPrefix, userID)

	c.JSON(http.StatusCreated, gin.H{
		"key":    plainKey,
		"apiKey": apiKey,
	})
}

// List returns all API keys for the authenticated user.
//
//	@Summary		List API keys
//	@Description	Returns all API keys for the authenticated user. Key values are not included, only metadata.
//	@Tags			API Keys
//	@Produce		json
//	@Success		200	{array}		models.ApiKey
//	@Failure		401	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/auth/api-keys [get]
func (h *ApiKeyHandler) List(c *gin.Context) {
	userID := c.GetUint("userId")

	keys, _ := h.repo.ListByUser(userID)

	c.JSON(http.StatusOK, keys)
}

// Revoke soft-deletes an API key by setting its revoked_at timestamp.
//
//	@Summary		Revoke an API key
//	@Description	Revokes an API key so it can no longer be used for authentication.
//	@Tags			API Keys
//	@Produce		json
//	@Param			id	path		int	true	"API Key ID"
//	@Success		200	{object}	MessageResponse
//	@Failure		400	{object}	ErrorResponse
//	@Failure		401	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/auth/api-keys/{id} [delete]
func (h *ApiKeyHandler) Revoke(c *gin.Context) {
	logger := services.AppLogger
	userID := c.GetUint("userId")

	keyID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid key ID"})
		return
	}

	var apiKey models.ApiKey
	found, err := h.repo.FindByIDAndUser(uint(keyID), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "API key not found"})
		return
	}
	apiKey = *found

	h.repo.Revoke(&apiKey)

	logger.Info("api-keys", "Revoked API key %d '%s' for user %d", apiKey.ID, apiKey.Name, userID)

	c.JSON(http.StatusOK, gin.H{"message": "API key revoked"})
}
