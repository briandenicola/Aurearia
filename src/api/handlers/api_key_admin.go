package handlers

import (
	"net/http"
	"sort"
	"strings"

	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
)

type ApiKeyAdminHandler struct {
	apiKeyRepo *repository.ApiKeyRepository
	notifSvc   *services.NotificationService
	logger     *services.Logger
}

func NewApiKeyAdminHandler(apiKeyRepo *repository.ApiKeyRepository, notifSvc *services.NotificationService, logger *services.Logger) *ApiKeyAdminHandler {
	return &ApiKeyAdminHandler{
		apiKeyRepo: apiKeyRepo,
		notifSvc:   notifSvc,
		logger:     logger,
	}
}

// NotifyRotationRequired creates per-user notifications for all users with active API keys.
//
//	@Summary		Create API key rotation notifications
//	@Description	Notifies each user with active API keys and lists key names that must be recreated. Replaces existing notifications of this type.
//	@Tags			Admin
//	@Produce		json
//	@Success		200	{object}	map[string]interface{}
//	@Failure		401	{object}	ErrorResponse
//	@Failure		403	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/admin/api-keys/notify-rotation [post]
func (h *ApiKeyAdminHandler) NotifyRotationRequired(c *gin.Context) {
	keys, err := h.apiKeyRepo.ListActive()
	if err != nil {
		h.logger.Error("api_keys", "Failed listing active API keys for rotation notifications: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list active API keys"})
		return
	}

	keyNamesByUser := make(map[uint][]string)
	for _, key := range keys {
		name := strings.TrimSpace(key.Name)
		if name == "" {
			name = key.KeyPrefix
		}
		keyNamesByUser[key.UserID] = append(keyNamesByUser[key.UserID], name)
	}

	type notifiedUser struct {
		UserID   uint     `json:"userId"`
		KeyNames []string `json:"keyNames"`
	}
	notified := make([]notifiedUser, 0, len(keyNamesByUser))
	failed := 0

	for userID, keyNames := range keyNamesByUser {
		if err := h.notifSvc.NotifyAPIKeyRotationRequired(userID, keyNames); err != nil {
			failed++
			continue
		}
		sort.Strings(keyNames)
		notified = append(notified, notifiedUser{
			UserID:   userID,
			KeyNames: keyNames,
		})
	}

	sort.Slice(notified, func(i, j int) bool {
		return notified[i].UserID < notified[j].UserID
	})

	c.JSON(http.StatusOK, gin.H{
		"notifiedUsers":        len(notified),
		"failedUsers":          failed,
		"notificationsCreated": len(notified),
		"users":                notified,
	})
}
