package services

import (
	"fmt"

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

// NotifyCoinOfDay creates an in-app notification and Pushover alert for the
// user's daily featured coin. The ReferenceID points to the FeaturedCoin record
// so the frontend can open the dedicated modal.
func (s *NotificationService) NotifyCoinOfDay(userID uint, featuredCoinID uint, coinName, summary string) {
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

	go s.sendPushover(userID, title, message, "")
}

// sendPushover checks if the user has Pushover enabled and sends a push notification.
func (s *NotificationService) sendPushover(userID uint, title, message, refURL string) {
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

	if err := s.pushoverSvc.SendNotification(user.PushoverUserKey, title, message, refURL); err != nil {
		s.logger.Error("pushover", "Failed to send Pushover notification to user %d: %v", userID, err)
	}
}
