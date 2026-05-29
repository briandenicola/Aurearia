package services

import (
	"strings"
	"time"

	"github.com/briandenicola/ancient-coins-api/repository"
)

const (
	NotificationTypeAPIKeyRotationRequired = "api_key_rotation_required"
	SettingAPIKeyRotationCutoffAt          = "APIKeyRotationCutoffAt"
)

// APIKeyRotationService keeps API key rotation notifications in sync.
// Keys created before the rollout cutoff are considered legacy and must be rotated.
type APIKeyRotationService struct {
	apiKeyRepo  *repository.ApiKeyRepository
	notifRepo   *repository.NotificationRepository
	notifSvc    *NotificationService
	settingsSvc *SettingsService
	logger      *Logger
}

func NewAPIKeyRotationService(
	apiKeyRepo *repository.ApiKeyRepository,
	notifRepo *repository.NotificationRepository,
	notifSvc *NotificationService,
	settingsSvc *SettingsService,
	logger *Logger,
) *APIKeyRotationService {
	return &APIKeyRotationService{
		apiKeyRepo:  apiKeyRepo,
		notifRepo:   notifRepo,
		notifSvc:    notifSvc,
		settingsSvc: settingsSvc,
		logger:      logger,
	}
}

// SyncFromStartup ensures startup-driven notifications are present for users who
// still have legacy API keys and removes stale notices once rotation is complete.
func (s *APIKeyRotationService) SyncFromStartup() {
	cutoff, err := s.getOrCreateCutoff()
	if err != nil {
		s.logger.Error("api_keys", "Failed to resolve API key rotation cutoff: %v", err)
		return
	}
	if err := s.syncForCutoff(cutoff); err != nil {
		s.logger.Error("api_keys", "Failed syncing API key rotation notifications: %v", err)
		return
	}
}

func (s *APIKeyRotationService) getOrCreateCutoff() (time.Time, error) {
	raw := strings.TrimSpace(s.settingsSvc.GetSetting(SettingAPIKeyRotationCutoffAt))
	if raw != "" {
		return time.Parse(time.RFC3339, raw)
	}

	cutoff := time.Now().UTC().Truncate(time.Second)
	if err := s.settingsSvc.SetSetting(SettingAPIKeyRotationCutoffAt, cutoff.Format(time.RFC3339)); err != nil {
		return time.Time{}, err
	}
	s.logger.Info("api_keys", "Initialized API key rotation cutoff at %s", cutoff.Format(time.RFC3339))
	return cutoff, nil
}

func (s *APIKeyRotationService) syncForCutoff(cutoff time.Time) error {
	keys, err := s.apiKeyRepo.ListActiveCreatedBefore(cutoff)
	if err != nil {
		return err
	}

	keyNamesByUser := make(map[uint][]string)
	for _, key := range keys {
		name := strings.TrimSpace(key.Name)
		if name == "" {
			name = key.KeyPrefix
		}
		keyNamesByUser[key.UserID] = append(keyNamesByUser[key.UserID], name)
	}

	for userID, keyNames := range keyNamesByUser {
		if err := s.notifSvc.NotifyAPIKeyRotationRequired(userID, keyNames); err != nil {
			s.logger.Error("api_keys", "Failed creating rotation notice for user %d: %v", userID, err)
		}
	}

	notifiedUserIDs, err := s.notifRepo.ListUserIDsByType(NotificationTypeAPIKeyRotationRequired)
	if err != nil {
		return err
	}
	for _, userID := range notifiedUserIDs {
		if _, stillNeedsRotation := keyNamesByUser[userID]; stillNeedsRotation {
			continue
		}
		if err := s.notifRepo.DeleteByUserAndType(userID, NotificationTypeAPIKeyRotationRequired); err != nil {
			s.logger.Error("api_keys", "Failed clearing rotation notice for user %d: %v", userID, err)
		}
	}

	s.logger.Info("api_keys", "API key rotation startup sync complete: %d users still require rotation", len(keyNamesByUser))
	return nil
}
