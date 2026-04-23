package services

import (
	"fmt"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
)

// NotificationService handles creating and managing notifications.
type NotificationService struct {
	notifRepo  *repository.NotificationRepository
	socialRepo *repository.SocialRepository
}

// NewNotificationService creates a new NotificationService.
func NewNotificationService(
	notifRepo *repository.NotificationRepository,
	socialRepo *repository.SocialRepository,
) *NotificationService {
	return &NotificationService{
		notifRepo:  notifRepo,
		socialRepo: socialRepo,
	}
}

// NotifyWishlistUnavailable creates a notification when a wishlist coin
// is detected as no longer available.
func (s *NotificationService) NotifyWishlistUnavailable(userID uint, coin models.Coin, reason string) {
	coinName := coin.Name
	if coinName == "" {
		coinName = "Unnamed coin"
	}

	n := &models.Notification{
		UserID:       userID,
		Type:         "wishlist_unavailable",
		Title:        "Wishlist item unavailable",
		Message:      fmt.Sprintf("%s appears to no longer be available. %s", coinName, reason),
		ReferenceID:  coin.ID,
		ReferenceURL: coin.ReferenceURL,
	}

	if err := s.notifRepo.Create(n); err != nil {
		AppLogger.Error("notifications", "Failed to create wishlist notification for user %d, coin %d: %v", userID, coin.ID, err)
	}
}

// NotifyNewCoin creates notifications for all accepted followers when a user
// adds a new coin to their collection (non-wishlist only).
func (s *NotificationService) NotifyNewCoin(ownerID uint, coin models.Coin) {
	if coin.IsWishlist {
		return
	}

	followers, err := s.socialRepo.GetAcceptedFollowerIDs(ownerID)
	if err != nil {
		AppLogger.Error("notifications", "Failed to get followers for user %d: %v", ownerID, err)
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
			AppLogger.Error("notifications", "Failed to notify follower %d about coin %d: %v", followerID, coin.ID, err)
		}
	}

	AppLogger.Debug("notifications", "Notified %d followers about new coin %d from user %d", len(followers), coin.ID, ownerID)
}
