package services

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/briandenicola/ancient-coins-api/models"
)

const pushoverAPIURL = "https://api.pushover.net/1/messages.json"

// pushoverMessageLimit is Pushover's hard cap on the message field; exceeding it causes the
// API to reject the request outright, so any batched message must stay within it even after
// adding per-lot detail (specs/_backlog/F027).
const pushoverMessageLimit = 1024

// buildBatchedLotMessage renders one Pushover body listing several auction lots under a
// header, dropping lots (with a trailing summary line) once the body would exceed
// pushoverMessageLimit — so a long list can never cause an API rejection. renderLot supplies
// the per-lot block, which is the only thing that differs between batched auction messages
// (the watch-bid digest and the newly-tracked-lots push).
func buildBatchedLotMessage(header string, lots []models.AuctionLot, renderLot func(models.AuctionLot) string) string {
	var builder strings.Builder
	builder.WriteString(header)

	for i, lot := range lots {
		entry := renderLot(lot)
		omittedNote := fmt.Sprintf("… %d more lot(s) omitted\n", len(lots)-i)
		if builder.Len()+len(entry) > pushoverMessageLimit-len(omittedNote) {
			builder.WriteString(omittedNote)
			break
		}
		builder.WriteString(entry)
	}

	return strings.TrimRight(builder.String(), "\n")
}

// ErrPushoverNotConfigured is returned when Pushover credentials are not set.
var ErrPushoverNotConfigured = fmt.Errorf("pushover not configured")

// PushoverService handles sending push notifications via the Pushover API.
type PushoverService struct {
	logger      *Logger
	settingsSvc *SettingsService
	httpClient  *http.Client
	apiURL      string
}

// PushoverMessage describes a single Pushover notification payload.
type PushoverMessage struct {
	UserKey  string
	Title    string
	Message  string
	URL      string
	URLTitle string
	HTML     bool
}

// NewPushoverService creates a new PushoverService.
func NewPushoverService(settingsSvc *SettingsService, logger *Logger) *PushoverService {
	return &PushoverService{
		settingsSvc: settingsSvc,
		logger:      logger,
		httpClient:  http.DefaultClient,
		apiURL:      pushoverAPIURL,
	}
}

// SendNotification sends a push notification to the specified user via Pushover.
// The app token is read from admin settings; userKey is per-user.
func (s *PushoverService) SendNotification(userKey, title, message, refURL string) error {
	return s.SendMessage(PushoverMessage{
		UserKey: userKey,
		Title:   title,
		Message: message,
		URL:     refURL,
	})
}

// SendMessage sends a push notification to the specified user via Pushover.
// The app token is read from admin settings; userKey is per-user.
func (s *PushoverService) SendMessage(message PushoverMessage) error {
	appToken := s.settingsSvc.GetSetting(SettingPushoverAppToken)
	if appToken == "" {
		s.logger.Warn("pushover", "Pushover app token not configured in admin settings")
		return ErrPushoverNotConfigured
	}
	if message.UserKey == "" {
		return ErrPushoverNotConfigured
	}

	form := url.Values{}
	form.Set("token", appToken)
	form.Set("user", message.UserKey)
	form.Set("title", message.Title)
	form.Set("message", message.Message)
	if message.URL != "" {
		form.Set("url", message.URL)
		if message.URLTitle != "" {
			form.Set("url_title", message.URLTitle)
		}
	}
	if message.HTML {
		form.Set("html", "1")
	}

	client := s.httpClient
	if client == nil {
		client = http.DefaultClient
	}
	apiURL := s.apiURL
	if apiURL == "" {
		apiURL = pushoverAPIURL
	}

	resp, err := client.Post(apiURL, "application/x-www-form-urlencoded", strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("pushover request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("pushover returned status %d", resp.StatusCode)
	}

	s.logger.Debug("pushover", "Notification sent to user (title: %s)", message.Title)
	return nil
}
