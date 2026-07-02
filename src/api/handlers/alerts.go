package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
)

type AlertHandler struct {
	service *services.AuctionAlertService
}

func NewAlertHandler(service *services.AuctionAlertService) *AlertHandler {
	return &AlertHandler{service: service}
}

// ListAlerts returns all price alerts for the current user.
//
//	@Summary		List price alerts
//	@Description	Returns price alerts owned by the authenticated user.
//	@Tags			Alerts
//	@Produce		json
//	@Success		200	{object}	object
//	@Failure		500	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/alerts [get]
func (h *AlertHandler) ListAlerts(c *gin.Context) {
	userID := c.GetUint("userId")
	alerts, err := h.service.ListAlerts(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list alerts"})
		return
	}

	result := make([]gin.H, 0, len(alerts))
	for _, a := range alerts {
		result = append(result, priceAlertResponse(a))
	}
	c.JSON(http.StatusOK, gin.H{"alerts": result})
}

// CreateAlert creates a new price alert.
//
//	@Summary		Create price alert
//	@Description	Creates a one-shot price alert for an auction lot watched by the authenticated user.
//	@Tags			Alerts
//	@Accept			json
//	@Produce		json
//	@Param			body	body		services.PriceAlertCreateRequest	true	"Request payload"
//	@Success		201		{object}	object
//	@Failure		400		{object}	ErrorResponse
//	@Failure		500		{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/alerts [post]
func (h *AlertHandler) CreateAlert(c *gin.Context) {
	userID := c.GetUint("userId")

	var req services.PriceAlertCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "auctionLotId and targetPrice are required"})
		return
	}

	alert, err := h.service.CreateAlert(userID, req)
	if err != nil {
		writeAlertServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, priceAlertResponse(*alert))
}

// DeleteAlert deletes a price alert.
//
//	@Summary		Delete price alert
//	@Description	Deletes a price alert owned by the authenticated user.
//	@Tags			Alerts
//	@Produce		json
//	@Param			id	path		int	true	"Alert ID"
//	@Success		200	{object}	MessageResponse
//	@Failure		400	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/alerts/{id} [delete]
func (h *AlertHandler) DeleteAlert(c *gin.Context) {
	userID := c.GetUint("userId")
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid alert ID"})
		return
	}
	if err := h.service.DeleteAlert(uint(id), userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete alert"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Alert deleted"})
}

// ListReminders returns all bid reminders for the current user.
//
//	@Summary		List bid reminders
//	@Description	Returns bid reminders owned by the authenticated user.
//	@Tags			Alerts
//	@Produce		json
//	@Success		200	{object}	object
//	@Failure		500	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/reminders [get]
func (h *AlertHandler) ListReminders(c *gin.Context) {
	userID := c.GetUint("userId")
	reminders, err := h.service.ListReminders(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list reminders"})
		return
	}

	result := make([]gin.H, 0, len(reminders))
	for _, r := range reminders {
		result = append(result, bidReminderResponse(r))
	}
	c.JSON(http.StatusOK, gin.H{"reminders": result})
}

// CreateReminder creates a new bid reminder.
//
//	@Summary		Create bid reminder
//	@Description	Creates a one-shot bid reminder for an auction lot watched by the authenticated user.
//	@Tags			Alerts
//	@Accept			json
//	@Produce		json
//	@Param			body	body		services.BidReminderCreateRequest	true	"Request payload"
//	@Success		201		{object}	object
//	@Failure		400		{object}	ErrorResponse
//	@Failure		500		{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/reminders [post]
func (h *AlertHandler) CreateReminder(c *gin.Context) {
	userID := c.GetUint("userId")

	var req services.BidReminderCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "auctionLotId is required"})
		return
	}

	reminder, err := h.service.CreateReminder(userID, req)
	if err != nil {
		writeAlertServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, bidReminderResponse(*reminder))
}

// DeleteReminder deletes a bid reminder.
//
//	@Summary		Delete bid reminder
//	@Description	Deletes a bid reminder owned by the authenticated user.
//	@Tags			Alerts
//	@Produce		json
//	@Param			id	path		int	true	"Reminder ID"
//	@Success		200	{object}	MessageResponse
//	@Failure		400	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/reminders/{id} [delete]
func (h *AlertHandler) DeleteReminder(c *gin.Context) {
	userID := c.GetUint("userId")
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid reminder ID"})
		return
	}
	if err := h.service.DeleteReminder(uint(id), userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete reminder"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Reminder deleted"})
}

func priceAlertResponse(a models.PriceAlert) gin.H {
	return gin.H{
		"id":           a.ID,
		"auctionLotId": a.AuctionLotID,
		"lotTitle":     a.AuctionLot.Title,
		"targetPrice":  a.TargetPrice,
		"direction":    a.Direction,
		"isTriggered":  a.IsTriggered,
		"triggeredAt":  a.TriggeredAt,
		"createdAt":    a.CreatedAt,
	}
}

func bidReminderResponse(r models.BidReminder) gin.H {
	return gin.H{
		"id":            r.ID,
		"auctionLotId":  r.AuctionLotID,
		"lotTitle":      r.AuctionLot.Title,
		"minutesBefore": r.MinutesBefore,
		"isNotified":    r.IsNotified,
		"notifiedAt":    r.NotifiedAt,
		"createdAt":     r.CreatedAt,
	}
}

func writeAlertServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, services.ErrAuctionLotNotWatchable):
		c.JSON(http.StatusBadRequest, gin.H{"error": "Auction lot must be one of your watched or bidding lots"})
	case errors.Is(err, services.ErrInvalidAlertDirection):
		c.JSON(http.StatusBadRequest, gin.H{"error": "direction must be above or below"})
	case errors.Is(err, services.ErrInvalidTargetPrice):
		c.JSON(http.StatusBadRequest, gin.H{"error": "targetPrice must be greater than zero"})
	case errors.Is(err, services.ErrInvalidReminderWindow):
		c.JSON(http.StatusBadRequest, gin.H{"error": "minutesBefore must be between 1 and 10080"})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save alert"})
	}
}
