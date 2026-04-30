package services

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

const pushoverAPIURL = "https://api.pushover.net/1/messages.json"

// PushoverService handles sending push notifications via the Pushover API.
type PushoverService struct {
	logger      *Logger
	settingsSvc *SettingsService
}

// NewPushoverService creates a new PushoverService.
func NewPushoverService(settingsSvc *SettingsService, logger *Logger) *PushoverService {
	return &PushoverService{
		settingsSvc: settingsSvc,
		logger:      logger,
	}
}

// ErrPushoverNotConfigured is returned when the Pushover app token is not set.
var ErrPushoverNotConfigured = fmt.Errorf("pushover app token not configured")

// SendNotification sends a push notification to the specified user via Pushover.
func (s *PushoverService) SendNotification(userKey, title, message, refURL string) error {
	appToken := s.settingsSvc.GetSetting(SettingPushoverAppToken)
	if appToken == "" {
		s.logger.Warn("pushover", "Pushover app token not configured, skipping notification")
		return ErrPushoverNotConfigured
	}

	form := url.Values{}
	form.Set("token", appToken)
	form.Set("user", userKey)
	form.Set("title", title)
	form.Set("message", message)
	if refURL != "" {
		form.Set("url", refURL)
	}

	resp, err := http.Post(pushoverAPIURL, "application/x-www-form-urlencoded", strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("pushover request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("pushover returned status %d", resp.StatusCode)
	}

	s.logger.Debug("pushover", "Notification sent to user (title: %s)", title)
	return nil
}
