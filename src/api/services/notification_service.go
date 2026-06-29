package services

import (
	"fmt"
	"html"
	"net/url"
	"sort"
	"strings"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
)

// NotificationService handles creating and managing notifications.
type NotificationService struct {
	notifRepo   *repository.NotificationRepository
	socialRepo  *repository.SocialRepository
	userRepo    *repository.UserRepository
	pushoverSvc *PushoverService
	logger      *Logger
}

const NotificationTypeFollowRequest = "follow_request"

// NewNotificationService creates a new NotificationService.
func NewNotificationService(
	notifRepo *repository.NotificationRepository,
	socialRepo *repository.SocialRepository,
	userRepo *repository.UserRepository,
	pushoverSvc *PushoverService,
	logger *Logger,
) *NotificationService {
	return &NotificationService{
		notifRepo:   notifRepo,
		socialRepo:  socialRepo,
		userRepo:    userRepo,
		pushoverSvc: pushoverSvc,
		logger:      logger,
	}
}

// NotifyWishlistUnavailable creates a notification when a wishlist coin
// is detected as no longer available.
func (s *NotificationService) NotifyWishlistUnavailable(userID uint, coin models.Coin, reason string) {
	coinName := coin.Name
	if coinName == "" {
		coinName = "Unnamed coin"
	}

	title := "Wishlist item unavailable"
	message := fmt.Sprintf("%s appears to no longer be available. %s", coinName, reason)

	n := &models.Notification{
		UserID:       userID,
		Type:         "wishlist_unavailable",
		Title:        title,
		Message:      message,
		ReferenceID:  coin.ID,
		ReferenceURL: coin.ReferenceURL,
	}

	if err := s.notifRepo.Create(n); err != nil {
		s.logger.Error("notifications", "Failed to create wishlist notification for user %d, coin %d: %v", userID, coin.ID, err)
	}

	go s.sendPushover(userID, title, message, coin.ReferenceURL)
}

// NotifyNewCoin creates notifications for all accepted followers when a user
// adds a new coin to their collection (non-wishlist only).
func (s *NotificationService) NotifyNewCoin(ownerID uint, coin models.Coin) {
	if coin.IsWishlist {
		return
	}

	followers, err := s.socialRepo.GetAcceptedFollowerIDs(ownerID)
	if err != nil {
		s.logger.Error("notifications", "Failed to get followers for user %d: %v", ownerID, err)
		return
	}

	if len(followers) == 0 {
		return
	}

	// Look up the owner's username for the message
	ownerName := fmt.Sprintf("User #%d", ownerID)
	if user, err := s.socialRepo.GetUserByID(ownerID); err == nil && user != nil {
		ownerName = user.Username
	}

	coinName := coin.Name
	if coinName == "" {
		coinName = "a new coin"
	}

	for _, followerID := range followers {
		n := &models.Notification{
			UserID:      followerID,
			Type:        "friend_new_coin",
			Title:       "New coin added",
			Message:     fmt.Sprintf("%s added %s to their collection.", ownerName, coinName),
			ReferenceID: coin.ID,
		}
		if err := s.notifRepo.Create(n); err != nil {
			s.logger.Error("notifications", "Failed to notify follower %d about coin %d: %v", followerID, coin.ID, err)
		}
		go s.sendPushover(followerID, "New coin added", fmt.Sprintf("%s added %s to their collection.", ownerName, coinName), "")
	}

	s.logger.Debug("notifications", "Notified %d followers about new coin %d from user %d", len(followers), coin.ID, ownerID)
}

// NotifyFollowRequest creates a notification for a user who received a new
// follower request.
func (s *NotificationService) NotifyFollowRequest(followerID, targetID uint) {
	if followerID == 0 || targetID == 0 || followerID == targetID {
		return
	}

	followerName := fmt.Sprintf("User #%d", followerID)
	if user, err := s.socialRepo.GetUserByID(followerID); err == nil && user != nil {
		followerName = user.Username
	}

	title := "New follower request"
	message := fmt.Sprintf("%s requested to follow you.", followerName)

	n := &models.Notification{
		UserID:       targetID,
		Type:         NotificationTypeFollowRequest,
		Title:        title,
		Message:      message,
		ReferenceID:  followerID,
		ReferenceURL: "/followers",
	}

	if err := s.notifRepo.Create(n); err != nil {
		s.logger.Error("notifications", "Failed to create follow-request notification for user %d from follower %d: %v", targetID, followerID, err)
		return
	}

	go s.sendPushover(targetID, title, message, "/followers")
}

// NotifyCoinOfDay creates an in-app notification and Pushover alert for the
// user's daily featured coin. The ReferenceID points to the FeaturedCoin record
// so the frontend can open the dedicated modal.
func (s *NotificationService) NotifyCoinOfDay(userID uint, featuredCoinID, coinID uint, coinName, summary string) {
	if coinName == "" {
		coinName = "Today's coin"
	}

	title := "Coin of the Day"
	message := coinName
	if summary != "" {
		// Keep notification message short — the modal shows the full summary.
		preview := summary
		if len(preview) > 140 {
			preview = preview[:137] + "..."
		}
		message = fmt.Sprintf("%s — %s", coinName, preview)
	}

	n := &models.Notification{
		UserID:      userID,
		Type:        "coin_of_day",
		Title:       title,
		Message:     message,
		ReferenceID: featuredCoinID,
	}

	if err := s.notifRepo.Create(n); err != nil {
		s.logger.Error("notifications", "Failed to create coin-of-day notification for user %d: %v", userID, err)
	}

	go s.sendPushoverMessage(userID, buildCoinOfDayPushoverMessage(title, coinID, coinName, summary, s.publicAppBaseURL()))
}

// NotifyAIJobCompleted creates a notification when an asynchronous AI job completes.
func (s *NotificationService) NotifyAIJobCompleted(userID, jobID, coinID uint, coinName, jobType string) {
	if coinName == "" {
		coinName = "coin"
	}
	label := formatAIJobType(jobType)
	title := fmt.Sprintf("AI %s complete", label)
	message := fmt.Sprintf("%s is ready.", coinName)
	refURL := fmt.Sprintf("/coin/%d", coinID)
	n := &models.Notification{
		UserID:       userID,
		Type:         "ai_job_completed",
		Title:        title,
		Message:      message,
		ReferenceID:  jobID,
		ReferenceURL: refURL,
	}
	if err := s.notifRepo.Create(n); err != nil {
		s.logger.Error("notifications", "Failed to create AI job completion notification for user %d, job %d: %v", userID, jobID, err)
	}
	go s.sendPushover(userID, title, message, refURL)
}

// NotifyAIJobFailed creates a notification when an asynchronous AI job fails.
func (s *NotificationService) NotifyAIJobFailed(userID, jobID, coinID uint, jobType, reason string) {
	label := formatAIJobType(jobType)
	title := fmt.Sprintf("AI %s failed", label)
	message := fmt.Sprintf("AI %s could not be completed.", label)
	if reason != "" {
		message = fmt.Sprintf("%s Please check AI provider configuration and try again.", message)
	}
	refURL := fmt.Sprintf("/coin/%d", coinID)
	n := &models.Notification{
		UserID:       userID,
		Type:         "ai_job_failed",
		Title:        title,
		Message:      message,
		ReferenceID:  jobID,
		ReferenceURL: refURL,
	}
	if err := s.notifRepo.Create(n); err != nil {
		s.logger.Error("notifications", "Failed to create AI job failure notification for user %d, job %d: %v", userID, jobID, err)
	}
	go s.sendPushover(userID, title, message, refURL)
}

// NotifyValuationRunComplete creates an in-app notification when a background valuation run completes.
func (s *NotificationService) NotifyValuationRunComplete(userID, runID uint, checked, updated, skipped, errors int) {
	title := "Valuation complete"
	message := fmt.Sprintf("Checked: %d | Updated: %d | Skipped: %d | Errors: %d", checked, updated, skipped, errors)
	n := &models.Notification{
		UserID:       userID,
		Type:         "valuation_complete",
		Title:        title,
		Message:      message,
		ReferenceID:  runID,
		ReferenceURL: "/stats/value-trends",
	}
	if err := s.notifRepo.Create(n); err != nil {
		s.logger.Error("notifications", "Failed to create valuation completion notification for user %d, run %d: %v", userID, runID, err)
	}
}

// NotifyAPIKeyRotationRequired creates a single actionable notification that lists
// active API key names that must be recreated.
func (s *NotificationService) NotifyAPIKeyRotationRequired(userID uint, keyNames []string) error {
	if len(keyNames) == 0 {
		return nil
	}

	normalized := make([]string, 0, len(keyNames))
	seen := make(map[string]struct{}, len(keyNames))
	for _, keyName := range keyNames {
		name := strings.TrimSpace(keyName)
		if name == "" {
			continue
		}
		if _, exists := seen[name]; exists {
			continue
		}
		seen[name] = struct{}{}
		normalized = append(normalized, name)
	}
	if len(normalized) == 0 {
		return nil
	}
	sort.Strings(normalized)

	n := &models.Notification{
		UserID:       userID,
		Type:         NotificationTypeAPIKeyRotationRequired,
		Title:        "Action required: Recreate API keys",
		Message:      fmt.Sprintf("Recreate these API keys in Settings: %s", strings.Join(normalized, ", ")),
		ReferenceURL: "/settings",
	}
	if err := s.notifRepo.ReplaceByUserAndType(n); err != nil {
		s.logger.Error("notifications", "Failed to create API key rotation notification for user %d: %v", userID, err)
		return err
	}
	return nil
}

// sendPushover checks if the user has Pushover enabled and sends a push notification.
func (s *NotificationService) sendPushover(userID uint, title, message, refURL string) {
	s.sendPushoverMessage(userID, PushoverMessage{
		Title:   title,
		Message: message,
		URL:     refURL,
	})
}

// sendPushoverMessage checks if the user has Pushover enabled and sends a push notification.
func (s *NotificationService) sendPushoverMessage(userID uint, message PushoverMessage) {
	if s.pushoverSvc == nil || s.userRepo == nil {
		return
	}

	user, err := s.userRepo.FindByID(userID)
	if err != nil || user == nil {
		return
	}

	if !user.PushoverEnabled || user.PushoverUserKey == "" {
		return
	}

	message.UserKey = user.PushoverUserKey
	if err := s.pushoverSvc.SendMessage(message); err != nil {
		s.logger.Error("pushover", "Failed to send Pushover notification to user %d: %v", userID, err)
	}
}

func (s *NotificationService) publicAppBaseURL() string {
	if s == nil || s.pushoverSvc == nil || s.pushoverSvc.settingsSvc == nil {
		return ""
	}
	return s.pushoverSvc.settingsSvc.GetSetting(SettingPublicAppURL)
}

func buildCoinOfDayPushoverMessage(title string, coinID uint, coinName, summary, publicAppBaseURL string) PushoverMessage {
	if coinName == "" {
		coinName = "Today's coin"
	}

	body := fmt.Sprintf("<b>%s</b>", html.EscapeString(coinName))
	if summary != "" {
		body = fmt.Sprintf("%s — %s", body, html.EscapeString(truncateRunes(summary, 140)))
	}

	coinURL := buildCoinOfDayURL(publicAppBaseURL, coinID)
	if coinURL != "" {
		body = fmt.Sprintf("%s — <a href=\"%s\">Open coin</a>", body, html.EscapeString(coinURL))
	}

	return PushoverMessage{
		Title:   title,
		Message: body,
		URL:     coinURL,
		HTML:    true,
	}
}

func formatAIJobType(jobType string) string {
	switch jobType {
	case "analysis":
		return "analysis"
	case "value_estimate":
		return "value estimate"
	default:
		return "AI job"
	}
}

func buildCoinOfDayURL(publicAppBaseURL string, coinID uint) string {
	base := strings.TrimRight(strings.TrimSpace(publicAppBaseURL), "/")
	if base == "" {
		return ""
	}
	parsed, err := url.Parse(base)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return ""
	}
	if parsed.Scheme != "https" && parsed.Scheme != "http" {
		return ""
	}
	return fmt.Sprintf("%s/coin/%d", base, coinID)
}

func truncateRunes(value string, max int) string {
	runes := []rune(value)
	if len(runes) <= max {
		return value
	}
	return string(runes[:max-3]) + "..."
}
