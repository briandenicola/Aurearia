package handlers

import (
	"log"
	"net/http"
	"strconv"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
)

type SocialHandler struct {
	repo   *repository.SocialRepository
	svc    *services.SocialService
	logger *services.Logger
}

func NewSocialHandler(repo *repository.SocialRepository, svc *services.SocialService, logger *services.Logger) *SocialHandler {
	return &SocialHandler{repo: repo, svc: svc, logger: logger}
}

// FollowUser sends a follow request to another user.
func (h *SocialHandler) FollowUser(c *gin.Context) {
	logger := h.logger
	userID := c.GetUint("userId")
	targetID, err := strconv.ParseUint(c.Param("userId"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	status, err := h.svc.FollowUser(userID, uint(targetID))
	if err != nil {
		log.Printf("[handler] FollowUser: %v", err)
		switch err {
		case services.ErrSelfFollow:
			c.JSON(http.StatusBadRequest, gin.H{"error": "Unable to follow user"})
		case services.ErrUserNotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		case services.ErrUserNotPublic:
			c.JSON(http.StatusForbidden, gin.H{"error": "Action not allowed"})
		case services.ErrBlocked:
			c.JSON(http.StatusForbidden, gin.H{"error": "Action not allowed"})
		case services.ErrFollowPending:
			c.JSON(http.StatusConflict, gin.H{"error": "Request already exists"})
		case services.ErrAlreadyFollowing:
			c.JSON(http.StatusConflict, gin.H{"error": "Already following user"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to follow user"})
		}
		return
	}

	_ = status
	logger.Info("social", "User %d sent follow request to user %d", userID, targetID)
	c.JSON(http.StatusCreated, gin.H{"message": "Follow request sent"})
}

// UnfollowUser unfollows a user (removes any follow relationship).
func (h *SocialHandler) UnfollowUser(c *gin.Context) {
	logger := h.logger
	userID := c.GetUint("userId")
	targetID, err := strconv.ParseUint(c.Param("userId"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	rows, err := h.repo.DeleteFollow(userID, uint(targetID))
	if err != nil || rows == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Not following this user"})
		return
	}

	logger.Info("social", "User %d unfollowed user %d", userID, targetID)
	c.JSON(http.StatusOK, gin.H{"message": "Unfollowed user"})
}

// AcceptFollower accepts a pending follow request.
func (h *SocialHandler) AcceptFollower(c *gin.Context) {
	logger := h.logger
	userID := c.GetUint("userId")
	followerID, err := strconv.ParseUint(c.Param("userId"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	rows, err := h.repo.AcceptFollow(uint(followerID), userID)
	if err != nil || rows == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "No pending follow request from this user"})
		return
	}

	logger.Info("social", "User %d accepted follower %d", userID, followerID)
	c.JSON(http.StatusOK, gin.H{"message": "Follower accepted"})
}

// BlockFollower blocks a user from following.
func (h *SocialHandler) BlockFollower(c *gin.Context) {
	logger := h.logger
	userID := c.GetUint("userId")
	followerID, err := strconv.ParseUint(c.Param("userId"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	if err := h.repo.BlockUser(uint(followerID), userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to block user"})
		return
	}

	logger.Info("social", "User %d blocked follower %d", userID, followerID)
	c.JSON(http.StatusOK, gin.H{"message": "User blocked"})
}

// UnblockFollower removes a block on a user.
func (h *SocialHandler) UnblockFollower(c *gin.Context) {
	logger := h.logger
	userID := c.GetUint("userId")
	followerID, err := strconv.ParseUint(c.Param("userId"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	rows, err := h.repo.UnblockUser(uint(followerID), userID)
	if err != nil || rows == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "User is not blocked"})
		return
	}

	logger.Info("social", "User %d unblocked user %d", userID, followerID)
	c.JSON(http.StatusOK, gin.H{"message": "User unblocked"})
}

// GetBlockedUsers returns users blocked by the authenticated user.
func (h *SocialHandler) GetBlockedUsers(c *gin.Context) {
	userID := c.GetUint("userId")

	blocked, err := h.repo.GetBlockedUsers(userID)
	if err != nil {
		h.logger.Error("social", "Failed to get blocked users: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get blocked users"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"blocked": blocked})
}

// GetFollowers returns users who follow the authenticated user (pending + accepted).
func (h *SocialHandler) GetFollowers(c *gin.Context) {
	userID := c.GetUint("userId")

	follows, users, err := h.repo.GetFollowers(userID)
	if err != nil {
		h.logger.Error("social", "Failed to get followers: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get followers"})
		return
	}

	if len(follows) == 0 {
		c.JSON(http.StatusOK, gin.H{"followers": []interface{}{}})
		return
	}

	statusMap := map[uint]string{}
	for _, f := range follows {
		statusMap[f.FollowerID] = f.Status
	}

	var result []gin.H
	for _, u := range users {
		result = append(result, gin.H{
			"id":         u.ID,
			"username":   u.Username,
			"avatarPath": u.AvatarPath,
			"isPublic":   u.IsPublic,
			"bio":        u.Bio,
			"status":     statusMap[u.ID],
		})
	}
	if result == nil {
		result = []gin.H{}
	}

	c.JSON(http.StatusOK, gin.H{"followers": result})
}

// GetFollowing returns users the authenticated user follows (accepted only).
func (h *SocialHandler) GetFollowing(c *gin.Context) {
	userID := c.GetUint("userId")

	users, err := h.repo.GetFollowing(userID)
	if err != nil {
		h.logger.Error("social", "Failed to get following: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get following"})
		return
	}

	if len(users) == 0 {
		c.JSON(http.StatusOK, gin.H{"following": []interface{}{}})
		return
	}

	result := h.svc.BuildUserList(users, userID)
	items := make([]gin.H, len(result))
	for i, item := range result {
		items[i] = gin.H{
			"id":          item.ID,
			"username":    item.Username,
			"avatarPath":  item.AvatarPath,
			"isPublic":    item.IsPublic,
			"bio":         item.Bio,
			"isFollowing": item.IsFollowing,
			"coinCount":   item.CoinCount,
		}
	}
	c.JSON(http.StatusOK, gin.H{"following": items})
}

// SearchUsers searches for users by username prefix (public users only).
func (h *SocialHandler) SearchUsers(c *gin.Context) {
	userID := c.GetUint("userId")
	query := c.Query("q")
	if query == "" || len(query) < 2 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Search query must be at least 2 characters"})
		return
	}

	users, err := h.repo.SearchUsers(query, userID)
	if err != nil {
		h.logger.Error("social", "Failed to search users: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to search users"})
		return
	}

	follows, err := h.repo.GetAllFollows(userID)
	if err != nil {
		h.logger.Error("social", "Failed to get follows: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get follows"})
		return
	}
	followStatusMap := map[uint]string{}
	for _, f := range follows {
		followStatusMap[f.FollowingID] = f.Status
	}

	var result []gin.H
	for _, u := range users {
		coinCount := h.repo.CountPublicCoins(u.ID)
		status := followStatusMap[u.ID]
		result = append(result, gin.H{
			"id":           u.ID,
			"username":     u.Username,
			"avatarPath":   u.AvatarPath,
			"isPublic":     u.IsPublic,
			"bio":          u.Bio,
			"isFollowing":  status == "accepted",
			"followStatus": status,
			"coinCount":    coinCount,
		})
	}

	if result == nil {
		result = []gin.H{}
	}
	c.JSON(http.StatusOK, gin.H{"users": result})
}

// GetPublicProfile returns a user's public profile.
func (h *SocialHandler) GetPublicProfile(c *gin.Context) {
	username := c.Param("username")
	currentUserID := c.GetUint("userId")

	profile, err := h.svc.GetPublicProfileData(username, currentUserID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":             profile.ID,
		"username":       profile.Username,
		"avatarPath":     profile.AvatarPath,
		"isPublic":       profile.IsPublic,
		"bio":            profile.Bio,
		"isFollowing":    profile.IsFollowing,
		"followStatus":   profile.FollowStatus,
		"coinCount":      profile.CoinCount,
		"followerCount":  profile.FollowerCount,
		"followingCount": profile.FollowingCount,
	})
}

// GetFollowingCoins returns a followed user's public coins (limited fields for gallery).
func (h *SocialHandler) GetFollowingCoins(c *gin.Context) {
	currentUserID := c.GetUint("userId")
	targetID, err := strconv.ParseUint(c.Param("userId"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	if !h.svc.CanViewCoins(currentUserID, uint(targetID)) {
		c.JSON(http.StatusForbidden, gin.H{"error": "You must be an accepted follower to view their coins"})
		return
	}

	target, err := h.repo.FindUser(uint(targetID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	if !target.IsPublic {
		c.JSON(http.StatusForbidden, gin.H{"error": "This user's collection is private"})
		return
	}

	coins, err := h.repo.GetPublicCoins(uint(targetID))
	if err != nil {
		h.logger.Error("social", "Failed to get public coins: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get coins"})
		return
	}

	var result []gin.H
	for _, coin := range coins {
		result = append(result, limitedCoinData(coin))
	}
	if result == nil {
		result = []gin.H{}
	}

	c.JSON(http.StatusOK, gin.H{
		"coins":    result,
		"username": target.Username,
	})
}

// GetFollowingCoinDetail returns a single coin with limited fields for a follower.
func (h *SocialHandler) GetFollowingCoinDetail(c *gin.Context) {
	currentUserID := c.GetUint("userId")
	targetID, err := strconv.ParseUint(c.Param("userId"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}
	coinID, err := strconv.ParseUint(c.Param("coinId"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid coin ID"})
		return
	}

	if !h.svc.CanViewCoins(currentUserID, uint(targetID)) {
		c.JSON(http.StatusForbidden, gin.H{"error": "You must be an accepted follower to view their coins"})
		return
	}

	coin, err := h.repo.FindPublicCoin(uint(coinID), uint(targetID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Coin not found"})
		return
	}

	comments, err := h.repo.GetCommentsWithAuthors(uint(coinID))
	if err != nil {
		h.logger.Error("social", "Failed to get comments: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get comments"})
		return
	}
	var commentResults []gin.H
	for _, cm := range comments {
		commentResults = append(commentResults, gin.H{
			"id":         cm.ID,
			"coinId":     cm.CoinID,
			"userId":     cm.UserID,
			"username":   cm.Username,
			"avatarPath": cm.AvatarPath,
			"comment":    cm.Comment,
			"rating":     cm.Rating,
			"createdAt":  cm.CreatedAt,
		})
	}
	if commentResults == nil {
		commentResults = []gin.H{}
	}

	ratingResult := h.repo.GetRatingStats(uint(coinID))
	userRating := h.repo.GetUserRating(uint(coinID), currentUserID)

	result := limitedCoinData(*coin)
	result["comments"] = commentResults
	result["rating"] = gin.H{
		"average":    ratingResult.Avg,
		"count":      ratingResult.Count,
		"userRating": userRating,
	}

	c.JSON(http.StatusOK, result)
}

// AddComment adds a comment (with optional rating) to a coin.
func (h *SocialHandler) AddComment(c *gin.Context) {
	logger := h.logger
	currentUserID := c.GetUint("userId")
	coinID, err := strconv.ParseUint(c.Param("coinId"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid coin ID"})
		return
	}

	var req struct {
		Comment string `json:"comment" binding:"required,max=2000"`
		Rating  int    `json:"rating"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "Invalid request payload", err)
		return
	}

	if req.Rating < 0 || req.Rating > 5 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Rating must be between 0 and 5"})
		return
	}

	coin, err := h.repo.FindCoin(uint(coinID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Coin not found"})
		return
	}

	if coin.IsPrivate {
		c.JSON(http.StatusForbidden, gin.H{"error": "Cannot comment on a private coin"})
		return
	}

	if coin.UserID != currentUserID {
		if !h.svc.CanViewCoins(currentUserID, coin.UserID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "You must be an accepted follower to comment on their coins"})
			return
		}
	}

	comment := models.CoinComment{
		CoinID:  uint(coinID),
		UserID:  currentUserID,
		Comment: req.Comment,
		Rating:  req.Rating,
	}

	if err := h.repo.CreateComment(&comment); err != nil {
		logger.Error("social", "Failed to create comment: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create comment"})
		return
	}

	user, err := h.repo.FindUser(currentUserID)
	if err != nil {
		h.logger.Error("social", "Failed to find user for comment response: %v", err)
	}

	logger.Info("social", "User %d commented on coin %d", currentUserID, coinID)
	c.JSON(http.StatusCreated, gin.H{
		"id":         comment.ID,
		"coinId":     comment.CoinID,
		"userId":     comment.UserID,
		"username":   user.Username,
		"avatarPath": user.AvatarPath,
		"comment":    comment.Comment,
		"rating":     comment.Rating,
		"createdAt":  comment.CreatedAt,
	})
}

// GetComments returns comments on a coin.
func (h *SocialHandler) GetComments(c *gin.Context) {
	currentUserID := c.GetUint("userId")
	coinID, err := strconv.ParseUint(c.Param("coinId"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid coin ID"})
		return
	}

	coin, err := h.repo.FindCoin(uint(coinID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Coin not found"})
		return
	}

	if coin.UserID != currentUserID {
		if !h.svc.CanViewCoins(currentUserID, coin.UserID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
			return
		}
	}

	comments, err := h.repo.GetCommentsWithAuthors(uint(coinID))
	if err != nil {
		h.logger.Error("social", "Failed to get comments: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get comments"})
		return
	}

	var result []gin.H
	for _, cm := range comments {
		result = append(result, gin.H{
			"id":         cm.ID,
			"coinId":     cm.CoinID,
			"userId":     cm.UserID,
			"username":   cm.Username,
			"avatarPath": cm.AvatarPath,
			"comment":    cm.Comment,
			"rating":     cm.Rating,
			"createdAt":  cm.CreatedAt,
		})
	}
	if result == nil {
		result = []gin.H{}
	}

	c.JSON(http.StatusOK, gin.H{"comments": result})
}

// DeleteComment deletes a comment. Owner of coin or commenter can delete.
func (h *SocialHandler) DeleteComment(c *gin.Context) {
	logger := h.logger
	currentUserID := c.GetUint("userId")
	coinID, err := strconv.ParseUint(c.Param("coinId"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid coin ID"})
		return
	}
	commentID, err := strconv.ParseUint(c.Param("commentId"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid comment ID"})
		return
	}

	comment, err := h.repo.FindComment(uint(commentID), uint(coinID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Comment not found"})
		return
	}

	coin, err := h.repo.FindCoin(uint(coinID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Coin not found"})
		return
	}
	if comment.UserID != currentUserID && coin.UserID != currentUserID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Cannot delete this comment"})
		return
	}

	h.repo.DeleteComment(comment)
	logger.Info("social", "Comment %d deleted by user %d", commentID, currentUserID)
	c.JSON(http.StatusOK, gin.H{"message": "Comment deleted"})
}

// RateCoin upserts a star rating for a coin.
func (h *SocialHandler) RateCoin(c *gin.Context) {
	currentUserID := c.GetUint("userId")
	coinID, err := strconv.ParseUint(c.Param("coinId"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid coin ID"})
		return
	}

	var req struct {
		Rating int `json:"rating" binding:"required,min=1,max=5"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Rating must be between 1 and 5"})
		return
	}

	coin, err := h.repo.FindCoin(uint(coinID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Coin not found"})
		return
	}

	if coin.UserID != currentUserID {
		if !h.svc.CanViewCoins(currentUserID, coin.UserID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "You must be an accepted follower to rate their coins"})
			return
		}
	}

	h.repo.UpsertRating(uint(coinID), currentUserID, req.Rating)

	ratingResult := h.repo.GetRatingStats(uint(coinID))
	c.JSON(http.StatusOK, gin.H{
		"average":    ratingResult.Avg,
		"count":      ratingResult.Count,
		"userRating": req.Rating,
	})
}

// GetCoinRating returns aggregate and user-specific rating for a coin.
func (h *SocialHandler) GetCoinRating(c *gin.Context) {
	currentUserID := c.GetUint("userId")
	coinID, err := strconv.ParseUint(c.Param("coinId"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid coin ID"})
		return
	}

	ratingResult := h.repo.GetRatingStats(uint(coinID))
	userRating := h.repo.GetUserRating(uint(coinID), currentUserID)

	c.JSON(http.StatusOK, gin.H{
		"average":    ratingResult.Avg,
		"count":      ratingResult.Count,
		"userRating": userRating,
	})
}

// Helper: strip sensitive fields from coin for follower view
func limitedCoinData(coin models.Coin) gin.H {
	return gin.H{
		"id":           coin.ID,
		"name":         coin.Name,
		"category":     coin.Category,
		"denomination": coin.Denomination,
		"ruler":        coin.Ruler,
		"era":          coin.Era,
		"material":     coin.Material,
		"grade":        coin.Grade,
		"images":       coin.Images,
	}
}
