package handlers

import (
	"net/http"
	"strconv"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/gin-gonic/gin"
)

type AlertHandler struct {
	alertRepo    *repository.PriceAlertRepository
	reminderRepo *repository.BidReminderRepository
}

func NewAlertHandler(alertRepo *repository.PriceAlertRepository, reminderRepo *repository.BidReminderRepository) *AlertHandler {
	return &AlertHandler{alertRepo: alertRepo, reminderRepo: reminderRepo}
}

// ListAlerts returns all price alerts for the current user.
func (h *AlertHandler) ListAlerts(c *gin.Context) {
	userID := c.GetUint("userId")
	alerts, err := h.alertRepo.ListByUser(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list alerts"})
		return
	}

	result := make([]gin.H, 0, len(alerts))
	for _, a := range alerts {
		result = append(result, gin.H{
			"id":           a.ID,
			"auctionLotId": a.AuctionLotID,
			"lotTitle":     a.AuctionLot.Title,
			"targetPrice":  a.TargetPrice,
			"direction":    a.Direction,
			"isTriggered":  a.IsTriggered,
			"triggeredAt":  a.TriggeredAt,
			"createdAt":    a.CreatedAt,
		})
	}
	c.JSON(http.StatusOK, gin.H{"alerts": result})
}

// CreateAlert creates a new price alert.
func (h *AlertHandler) CreateAlert(c *gin.Context) {
	userID := c.GetUint("userId")

	var req struct {
		AuctionLotID uint    `json:"auctionLotId" binding:"required"`
		TargetPrice  float64 `json:"targetPrice" binding:"required"`
		Direction    string  `json:"direction"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "auctionLotId and targetPrice are required"})
		return
	}

	direction := req.Direction
	if direction != "above" && direction != "below" {
		direction = "above"
	}

	alert := models.PriceAlert{
		AuctionLotID: req.AuctionLotID,
		UserID:       userID,
		TargetPrice:  req.TargetPrice,
		Direction:    direction,
	}

	if err := h.alertRepo.Create(&alert); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create alert"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"id": alert.ID})
}

// DeleteAlert deletes a price alert.
func (h *AlertHandler) DeleteAlert(c *gin.Context) {
	userID := c.GetUint("userId")
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid alert ID"})
		return
	}
	if err := h.alertRepo.Delete(uint(id), userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete alert"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Alert deleted"})
}

// ListReminders returns all bid reminders for the current user.
func (h *AlertHandler) ListReminders(c *gin.Context) {
	userID := c.GetUint("userId")
	reminders, err := h.reminderRepo.ListByUser(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list reminders"})
		return
	}

	result := make([]gin.H, 0, len(reminders))
	for _, r := range reminders {
		result = append(result, gin.H{
			"id":            r.ID,
			"auctionLotId":  r.AuctionLotID,
			"lotTitle":      r.AuctionLot.Title,
			"minutesBefore": r.MinutesBefore,
			"isNotified":    r.IsNotified,
			"notifiedAt":    r.NotifiedAt,
			"createdAt":     r.CreatedAt,
		})
	}
	c.JSON(http.StatusOK, gin.H{"reminders": result})
}

// CreateReminder creates a new bid reminder.
func (h *AlertHandler) CreateReminder(c *gin.Context) {
	userID := c.GetUint("userId")

	var req struct {
		AuctionLotID  uint `json:"auctionLotId" binding:"required"`
		MinutesBefore int  `json:"minutesBefore"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "auctionLotId is required"})
		return
	}

	minutes := req.MinutesBefore
	if minutes <= 0 {
		minutes = 30
	}

	reminder := models.BidReminder{
		AuctionLotID:  req.AuctionLotID,
		UserID:        userID,
		MinutesBefore: minutes,
	}

	if err := h.reminderRepo.Create(&reminder); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create reminder"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"id": reminder.ID})
}

// DeleteReminder deletes a bid reminder.
func (h *AlertHandler) DeleteReminder(c *gin.Context) {
	userID := c.GetUint("userId")
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid reminder ID"})
		return
	}
	if err := h.reminderRepo.Delete(uint(id), userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete reminder"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Reminder deleted"})
}
