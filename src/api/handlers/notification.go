package handlers

import (
	"net/http"
	"strconv"

	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/gin-gonic/gin"
)

// NotificationHandler handles notification endpoints.
type NotificationHandler struct {
	repo *repository.NotificationRepository
}

// NewNotificationHandler creates a new NotificationHandler.
func NewNotificationHandler(repo *repository.NotificationRepository) *NotificationHandler {
	return &NotificationHandler{repo: repo}
}

// List returns paginated notifications for the authenticated user.
//
//	@Summary		List notifications
//	@Description	Returns notifications for the current user, newest first.
//	@Tags			Notifications
//	@Produce		json
//	@Param			page	query		int	false	"Page number"	default(1)
//	@Param			limit	query		int	false	"Items per page"	default(20)
//	@Success		200		{object}	map[string]interface{}
//	@Failure		401		{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/notifications [get]
func (h *NotificationHandler) List(c *gin.Context) {
	userID := c.GetUint("userId")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	notifications, total, err := h.repo.GetByUser(userID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch notifications"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"notifications": notifications,
		"total":         total,
		"page":          page,
		"limit":         limit,
	})
}

// UnreadCount returns the number of unread notifications for the authenticated user.
//
//	@Summary		Unread notification count
//	@Description	Returns the count of unread notifications.
//	@Tags			Notifications
//	@Produce		json
//	@Success		200	{object}	map[string]interface{}
//	@Failure		401	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/notifications/unread-count [get]
func (h *NotificationHandler) UnreadCount(c *gin.Context) {
	userID := c.GetUint("userId")
	count, err := h.repo.GetUnreadCount(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get unread count"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"count": count})
}

// MarkRead marks a single notification as read.
//
//	@Summary		Mark notification read
//	@Description	Marks a specific notification as read.
//	@Tags			Notifications
//	@Param			id	path	int	true	"Notification ID"
//	@Success		200	{object}	map[string]interface{}
//	@Failure		404	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/notifications/{id}/read [put]
func (h *NotificationHandler) MarkRead(c *gin.Context) {
	userID := c.GetUint("userId")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid notification ID"})
		return
	}

	if err := h.repo.MarkRead(uint(id), userID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Notification not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Marked as read"})
}

// MarkAllRead marks all notifications as read for the authenticated user.
//
//	@Summary		Mark all notifications read
//	@Description	Marks all unread notifications as read.
//	@Tags			Notifications
//	@Success		200	{object}	map[string]interface{}
//	@Security		BearerAuth
//	@Router			/notifications/read-all [put]
func (h *NotificationHandler) MarkAllRead(c *gin.Context) {
	userID := c.GetUint("userId")
	if err := h.repo.MarkAllRead(userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to mark all as read"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "All marked as read"})
}

// Delete removes a single notification.
//
//	@Summary		Delete notification
//	@Description	Deletes a specific notification.
//	@Tags			Notifications
//	@Param			id	path	int	true	"Notification ID"
//	@Success		200	{object}	map[string]interface{}
//	@Failure		404	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/notifications/{id} [delete]
func (h *NotificationHandler) Delete(c *gin.Context) {
	userID := c.GetUint("userId")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid notification ID"})
		return
	}

	if err := h.repo.Delete(uint(id), userID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Notification not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Notification deleted"})
}
