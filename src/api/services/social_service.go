package services

import (
	"errors"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
)

var (
	ErrSelfFollow       = errors.New("cannot follow yourself")
	ErrUserNotFound     = errors.New("user not found")
	ErrUserNotPublic    = errors.New("this user is not accepting followers")
	ErrBlocked          = errors.New("you are blocked by this user")
	ErrFollowPending    = errors.New("follow request already pending")
	ErrAlreadyFollowing = errors.New("already following this user")
)

// UserListItem represents a user in a list with follow and coin metadata.
type UserListItem struct {
	ID          uint   `json:"id"`
	Username    string `json:"username"`
	AvatarPath  string `json:"avatarPath"`
	IsPublic    bool   `json:"isPublic"`
	Bio         string `json:"bio"`
	IsFollowing bool   `json:"isFollowing"`
	CoinCount   int64  `json:"coinCount"`
}

// PublicProfile represents a user's public profile with social metadata.
type PublicProfile struct {
	ID             uint   `json:"id"`
	Username       string `json:"username"`
	AvatarPath     string `json:"avatarPath"`
	IsPublic       bool   `json:"isPublic"`
	Bio            string `json:"bio"`
	IsFollowing    bool   `json:"isFollowing"`
	FollowStatus   string `json:"followStatus"`
	CoinCount      int64  `json:"coinCount"`
	FollowerCount  int64  `json:"followerCount"`
	FollowingCount int64  `json:"followingCount"`
}

// SocialService handles social interaction business logic.
type SocialService struct {
	repo     *repository.SocialRepository
	notifSvc *NotificationService
}

// NewSocialService creates a new SocialService.
func NewSocialService(repo *repository.SocialRepository, notifSvc *NotificationService) *SocialService {
	return &SocialService{repo: repo, notifSvc: notifSvc}
}

// FollowUser processes a follow request. Returns the follow status
// ("accepted" or "pending") on success.
func (s *SocialService) FollowUser(followerID, targetID uint) (string, error) {
	if targetID == followerID {
		return "", ErrSelfFollow
	}

	target, err := s.repo.FindUser(targetID)
	if err != nil {
		return "", ErrUserNotFound
	}
	if !target.IsPublic {
		return "", ErrUserNotPublic
	}

	existing, err := s.repo.FindFollow(followerID, targetID)
	if err == nil {
		if existing.Status == "blocked" {
			return "", ErrBlocked
		}
		if existing.Status == "pending" {
			return "", ErrFollowPending
		}
		return "", ErrAlreadyFollowing
	}

	follow := models.Follow{
		FollowerID:  followerID,
		FollowingID: targetID,
		Status:      "pending",
	}

	if err := s.repo.CreateFollow(&follow); err != nil {
		return "", ErrAlreadyFollowing
	}

	if s.notifSvc != nil {
		s.notifSvc.NotifyFollowRequest(followerID, targetID)
	}

	return "pending", nil
}

// BuildUserList builds a list of users with coin counts and follow state
// relative to the current user.
func (s *SocialService) BuildUserList(users []models.User, currentUserID uint) []UserListItem {
	follows, _ := s.repo.GetAllFollows(currentUserID)
	followSet := map[uint]bool{}
	for _, f := range follows {
		if f.Status == "accepted" {
			followSet[f.FollowingID] = true
		}
	}

	result := make([]UserListItem, 0, len(users))
	for _, u := range users {
		var coinCount int64
		if u.IsPublic {
			coinCount = s.repo.CountPublicCoins(u.ID)
		}
		result = append(result, UserListItem{
			ID:          u.ID,
			Username:    u.Username,
			AvatarPath:  u.AvatarPath,
			IsPublic:    u.IsPublic,
			Bio:         u.Bio,
			IsFollowing: followSet[u.ID],
			CoinCount:   coinCount,
		})
	}
	return result
}

// CanViewCoins returns true if the viewer is an accepted follower of the owner.
func (s *SocialService) CanViewCoins(viewerID, ownerID uint) bool {
	return s.repo.IsAcceptedFollower(viewerID, ownerID)
}

// GetPublicProfileData assembles a user's public profile with social metadata.
func (s *SocialService) GetPublicProfileData(username string, viewerID uint) (*PublicProfile, error) {
	user, err := s.repo.FindUserByUsername(username)
	if err != nil {
		return nil, ErrUserNotFound
	}

	var followStatus string
	var isFollowing bool
	follow, err := s.repo.FindFollow(viewerID, user.ID)
	if err == nil {
		followStatus = follow.Status
		isFollowing = follow.Status == "accepted"
	}

	var coinCount int64
	if user.IsPublic && isFollowing {
		coinCount = s.repo.CountPublicCoins(user.ID)
	}

	return &PublicProfile{
		ID:             user.ID,
		Username:       user.Username,
		AvatarPath:     user.AvatarPath,
		IsPublic:       user.IsPublic,
		Bio:            user.Bio,
		IsFollowing:    isFollowing,
		FollowStatus:   followStatus,
		CoinCount:      coinCount,
		FollowerCount:  s.repo.GetFollowerCount(user.ID),
		FollowingCount: s.repo.GetFollowingCount(user.ID),
	}, nil
}
