package repository

import (
	"github.com/briandenicola/ancient-coins-api/models"
	"gorm.io/gorm"
)

// RatingResult holds aggregate rating data.
type RatingResult struct {
	Avg   float64 `json:"average"`
	Count int64   `json:"count"`
}

// CommentWithAuthor holds a comment joined with its author info.
type CommentWithAuthor struct {
	models.CoinComment
	Username   string `json:"username"`
	AvatarPath string `json:"avatarPath"`
}

// UserSummary holds basic user info for social views.
type UserSummary struct {
	ID         uint   `json:"id"`
	Username   string `json:"username"`
	AvatarPath string `json:"avatarPath"`
	IsPublic   bool   `json:"isPublic"`
	Bio        string `json:"bio"`
}

// SocialRepository encapsulates all social/follow-related database operations.
type SocialRepository struct {
	db *gorm.DB
}

// NewSocialRepository creates a new SocialRepository.
func NewSocialRepository(db *gorm.DB) *SocialRepository {
	return &SocialRepository{db: db}
}

// FindUser returns a user by primary key.
func (r *SocialRepository) FindUser(id uint) (*models.User, error) {
	var user models.User
	err := r.db.First(&user, id).Error
	return &user, err
}

// FindUserByUsername returns a user by username.
func (r *SocialRepository) FindUserByUsername(username string) (*models.User, error) {
	var user models.User
	err := r.db.Where("username = ?", username).First(&user).Error
	return &user, err
}

// FindFollow returns the follow relationship between two users, if any.
func (r *SocialRepository) FindFollow(followerID, followingID uint) (*models.Follow, error) {
	var follow models.Follow
	err := r.db.Where("follower_id = ? AND following_id = ?", followerID, followingID).First(&follow).Error
	return &follow, err
}

// IsAcceptedFollower checks if followerID has an accepted follow on followingID.
func (r *SocialRepository) IsAcceptedFollower(followerID, followingID uint) bool {
	var count int64
	r.db.Model(&models.Follow{}).
		Where("follower_id = ? AND following_id = ? AND status = ?", followerID, followingID, "accepted").
		Count(&count)
	return count > 0
}

// CreateFollow inserts a new follow record.
func (r *SocialRepository) CreateFollow(follow *models.Follow) error {
	return r.db.Create(follow).Error
}

// DeleteFollow removes a non-blocked follow relationship.
func (r *SocialRepository) DeleteFollow(followerID, followingID uint) (int64, error) {
	result := r.db.Where("follower_id = ? AND following_id = ? AND status != ?", followerID, followingID, "blocked").
		Delete(&models.Follow{})
	return result.RowsAffected, result.Error
}

// AcceptFollow updates a pending follow to accepted.
func (r *SocialRepository) AcceptFollow(followerID, followingID uint) (int64, error) {
	result := r.db.Model(&models.Follow{}).
		Where("follower_id = ? AND following_id = ? AND status = ?", followerID, followingID, "pending").
		Update("status", "accepted")
	return result.RowsAffected, result.Error
}

// BlockUser blocks a user. Creates or updates the follow record.
func (r *SocialRepository) BlockUser(followerID, followingID uint) error {
	var follow models.Follow
	if err := r.db.Where("follower_id = ? AND following_id = ?", followerID, followingID).First(&follow).Error; err != nil {
		follow = models.Follow{
			FollowerID:  followerID,
			FollowingID: followingID,
			Status:      "blocked",
		}
		return r.db.Create(&follow).Error
	}
	return r.db.Model(&follow).Update("status", "blocked").Error
}

// UnblockUser removes a block relationship.
func (r *SocialRepository) UnblockUser(followerID, followingID uint) (int64, error) {
	result := r.db.Where("follower_id = ? AND following_id = ? AND status = ?", followerID, followingID, "blocked").
		Delete(&models.Follow{})
	return result.RowsAffected, result.Error
}

// GetBlockedUsers returns users blocked by the given user.
func (r *SocialRepository) GetBlockedUsers(userID uint) ([]UserSummary, error) {
	var follows []models.Follow
	r.db.Where("following_id = ? AND status = ?", userID, "blocked").Find(&follows)

	if len(follows) == 0 {
		return []UserSummary{}, nil
	}

	ids := make([]uint, len(follows))
	for i, f := range follows {
		ids[i] = f.FollowerID
	}

	var users []models.User
	r.db.Where("id IN ?", ids).Find(&users)

	result := make([]UserSummary, len(users))
	for i, u := range users {
		result[i] = UserSummary{ID: u.ID, Username: u.Username, AvatarPath: u.AvatarPath}
	}
	return result, nil
}

// GetFollowers returns users following the given user (pending + accepted) with status.
func (r *SocialRepository) GetFollowers(userID uint) ([]models.Follow, []models.User, error) {
	var follows []models.Follow
	r.db.Where("following_id = ? AND status IN ?", userID, []string{"pending", "accepted"}).Find(&follows)

	if len(follows) == 0 {
		return follows, nil, nil
	}

	ids := make([]uint, len(follows))
	for i, f := range follows {
		ids[i] = f.FollowerID
	}

	var users []models.User
	r.db.Where("id IN ?", ids).Find(&users)
	return follows, users, nil
}

// GetFollowing returns users the given user follows (accepted only).
func (r *SocialRepository) GetFollowing(userID uint) ([]models.User, error) {
	var follows []models.Follow
	r.db.Where("follower_id = ? AND status = ?", userID, "accepted").Find(&follows)

	if len(follows) == 0 {
		return nil, nil
	}

	ids := make([]uint, len(follows))
	for i, f := range follows {
		ids[i] = f.FollowingID
	}

	var users []models.User
	r.db.Where("id IN ?", ids).Find(&users)
	return users, nil
}

// GetAllFollows returns all follow records where followerID is the follower.
func (r *SocialRepository) GetAllFollows(followerID uint) ([]models.Follow, error) {
	var follows []models.Follow
	err := r.db.Where("follower_id = ?", followerID).Find(&follows).Error
	return follows, err
}

// GetAcceptedFollowerIDs returns the user IDs of all accepted followers for the given user.
func (r *SocialRepository) GetAcceptedFollowerIDs(userID uint) ([]uint, error) {
	var ids []uint
	err := r.db.Model(&models.Follow{}).
		Where("following_id = ? AND status = ?", userID, "accepted").
		Pluck("follower_id", &ids).Error
	return ids, err
}

// GetUserByID returns a user by primary key.
func (r *SocialRepository) GetUserByID(id uint) (*models.User, error) {
	var user models.User
	if err := r.db.First(&user, id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// SearchUserssearches for public users by username prefix, excluding the given user.
func (r *SocialRepository) SearchUsers(query string, excludeUserID uint) ([]models.User, error) {
	var users []models.User
	err := r.db.Where("username LIKE ? AND id != ? AND is_public = ?", query+"%", excludeUserID, true).
		Limit(20).Find(&users).Error
	return users, err
}

// CountPublicCoins returns the number of public, active coins for a user.
func (r *SocialRepository) CountPublicCoins(userID uint) int64 {
	var count int64
	r.db.Model(&models.Coin{}).Scopes(PublicCoins(userID)).Count(&count)
	return count
}

// GetPublicCoins returns public, active coins for a user with images preloaded.
func (r *SocialRepository) GetPublicCoins(userID uint) ([]models.Coin, error) {
	var coins []models.Coin
	err := r.db.Scopes(PublicCoins(userID)).
		Preload("Images").
		Order("updated_at DESC").
		Find(&coins).Error
	return coins, err
}

// FindPublicCoin returns a single public, active coin by ID and user.
func (r *SocialRepository) FindPublicCoin(coinID, userID uint) (*models.Coin, error) {
	var coin models.Coin
	err := r.db.Where("id = ? AND user_id = ? AND is_private = false AND is_wishlist = false AND is_sold = false", coinID, userID).
		Preload("Images").First(&coin).Error
	return &coin, err
}

// FindCoin returns a coin by primary key (no ownership filter).
func (r *SocialRepository) FindCoin(coinID uint) (*models.Coin, error) {
	var coin models.Coin
	err := r.db.First(&coin, coinID).Error
	return &coin, err
}

// GetFollowerCount returns the number of accepted followers for a user.
func (r *SocialRepository) GetFollowerCount(userID uint) int64 {
	var count int64
	r.db.Model(&models.Follow{}).Where("following_id = ? AND status = ?", userID, "accepted").Count(&count)
	return count
}

// GetFollowingCount returns the number of users a user follows.
func (r *SocialRepository) GetFollowingCount(userID uint) int64 {
	var count int64
	r.db.Model(&models.Follow{}).Where("follower_id = ? AND status = ?", userID, "accepted").Count(&count)
	return count
}

// GetCommentsWithAuthors returns comments for a coin with author info, avoiding N+1 queries.
func (r *SocialRepository) GetCommentsWithAuthors(coinID uint) ([]CommentWithAuthor, error) {
	var comments []models.CoinComment
	r.db.Where("coin_id = ?", coinID).Order("created_at DESC").Find(&comments)

	if len(comments) == 0 {
		return []CommentWithAuthor{}, nil
	}

	// Collect unique user IDs
	userIDSet := map[uint]bool{}
	for _, c := range comments {
		userIDSet[c.UserID] = true
	}
	userIDs := make([]uint, 0, len(userIDSet))
	for id := range userIDSet {
		userIDs = append(userIDs, id)
	}

	// Batch load users
	var users []models.User
	r.db.Where("id IN ?", userIDs).Find(&users)
	userMap := map[uint]models.User{}
	for _, u := range users {
		userMap[u.ID] = u
	}

	result := make([]CommentWithAuthor, len(comments))
	for i, cm := range comments {
		u := userMap[cm.UserID]
		result[i] = CommentWithAuthor{
			CoinComment: cm,
			Username:    u.Username,
			AvatarPath:  u.AvatarPath,
		}
	}
	return result, nil
}

// GetRatingStats returns aggregate rating stats for a coin.
func (r *SocialRepository) GetRatingStats(coinID uint) RatingResult {
	var result RatingResult
	r.db.Model(&models.CoinComment{}).
		Select("COALESCE(AVG(NULLIF(rating, 0)), 0) as avg, COUNT(NULLIF(rating, 0)) as count").
		Where("coin_id = ?", coinID).
		Scan(&result)
	return result
}

// GetUserRating returns the current user's rating for a coin, or 0 if none.
func (r *SocialRepository) GetUserRating(coinID, userID uint) int {
	var comment models.CoinComment
	if err := r.db.Where("coin_id = ? AND user_id = ? AND rating > 0", coinID, userID).First(&comment).Error; err == nil {
		return comment.Rating
	}
	return 0
}

// CreateComment inserts a new comment.
func (r *SocialRepository) CreateComment(comment *models.CoinComment) error {
	return r.db.Create(comment).Error
}

// FindComment returns a comment by ID and coin ID.
func (r *SocialRepository) FindComment(commentID, coinID uint) (*models.CoinComment, error) {
	var comment models.CoinComment
	err := r.db.Where("id = ? AND coin_id = ?", commentID, coinID).First(&comment).Error
	return &comment, err
}

// DeleteComment deletes a comment.
func (r *SocialRepository) DeleteComment(comment *models.CoinComment) error {
	return r.db.Delete(comment).Error
}

// UpsertRating creates or updates a star rating for a coin.
func (r *SocialRepository) UpsertRating(coinID, userID uint, rating int) error {
	var existing models.CoinComment
	if err := r.db.Where("coin_id = ? AND user_id = ? AND rating > 0", coinID, userID).First(&existing).Error; err == nil {
		return r.db.Model(&existing).Update("rating", rating).Error
	}
	return r.db.Create(&models.CoinComment{
		CoinID:  coinID,
		UserID:  userID,
		Comment: "",
		Rating:  rating,
	}).Error
}
