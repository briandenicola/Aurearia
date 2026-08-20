package handlers

import (
	"errors"
	"net/http"

	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
)

// PurchaseReminderHandler handles CRUD endpoints for purchase reminders.
type PurchaseReminderHandler struct {
	svc    *services.PurchaseReminderService
	logger *services.Logger
}

// NewPurchaseReminderHandler constructs the handler.
func NewPurchaseReminderHandler(svc *services.PurchaseReminderService, logger *services.Logger) *PurchaseReminderHandler {
	return &PurchaseReminderHandler{svc: svc, logger: logger}
}

// createOrUpdateReminderRequest is the JSON body for POST /coins/:id/reminder.
type createOrUpdateReminderRequest struct {
	RemindDate string `json:"remindDate" binding:"required"`
	Timezone   string `json:"timezone" binding:"required"`
}

// CreateOrUpdate creates a new reminder or updates the existing active one for a wishlist coin.
//
//	@Summary		Create or update a purchase reminder
//	@Description	Sets a date-based purchase reminder on a wishlist coin. If an active reminder already exists it is updated in-place (status reset to pending). Returns 201 on create, 200 on update.
//	@Tags			Purchase Reminders
//	@Accept			json
//	@Produce		json
//	@Param			id		path	int									true	"Coin ID"
//	@Param			body	body	createOrUpdateReminderRequest		true	"Reminder payload"
//	@Success		201		{object}	models.PurchaseReminder
//	@Success		200		{object}	models.PurchaseReminder
//	@Failure		400		{object}	ErrorResponse
//	@Failure		404		{object}	ErrorResponse
//	@Failure		409		{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/coins/{id}/reminder [post]
func (h *PurchaseReminderHandler) CreateOrUpdate(c *gin.Context) {
	coinID, ok := parseID(c, "id")
	if !ok {
		return
	}
	var req createOrUpdateReminderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid reminder payload"})
		return
	}
	userID := c.GetUint("userId")
	reminder, isNew, err := h.svc.CreateOrUpdate(userID, coinID, req.RemindDate, req.Timezone)
	if err != nil {
		respondReminderError(c, err)
		return
	}
	if isNew {
		c.JSON(http.StatusCreated, reminder)
	} else {
		c.JSON(http.StatusOK, reminder)
	}
}

// Get returns the active reminder for a coin, if any.
//
//	@Summary		Get purchase reminder
//	@Description	Returns the active (pending or notified) reminder for the given coin.
//	@Tags			Purchase Reminders
//	@Produce		json
//	@Param			id	path	int	true	"Coin ID"
//	@Success		200	{object}	models.PurchaseReminder
//	@Failure		404	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/coins/{id}/reminder [get]
func (h *PurchaseReminderHandler) Get(c *gin.Context) {
	coinID, ok := parseID(c, "id")
	if !ok {
		return
	}
	reminder, err := h.svc.GetForCoin(c.GetUint("userId"), coinID)
	if err != nil {
		respondReminderError(c, err)
		return
	}
	if reminder == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "No active reminder"})
		return
	}
	c.JSON(http.StatusOK, reminder)
}

// Cancel cancels the active reminder for a coin.
//
//	@Summary		Cancel purchase reminder
//	@Description	Cancels the active reminder for the given coin. Returns 204 on success, 404 if none exists.
//	@Tags			Purchase Reminders
//	@Param			id	path	int	true	"Coin ID"
//	@Success		204
//	@Failure		404	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/coins/{id}/reminder [delete]
func (h *PurchaseReminderHandler) Cancel(c *gin.Context) {
	coinID, ok := parseID(c, "id")
	if !ok {
		return
	}
	if err := h.svc.Cancel(c.GetUint("userId"), coinID); err != nil {
		respondReminderError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// List returns all active reminders for the authenticated user.
//
//	@Summary		List purchase reminders
//	@Description	Returns all active (pending + notified) purchase reminders for the current user.
//	@Tags			Purchase Reminders
//	@Produce		json
//	@Success		200	{object}	map[string]interface{}
//	@Security		BearerAuth
//	@Router			/purchase-reminders [get]
func (h *PurchaseReminderHandler) List(c *gin.Context) {
	reminders, err := h.svc.ListForUser(c.GetUint("userId"))
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to list reminders", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"reminders": reminders})
}

// respondReminderError maps service sentinel errors to HTTP status codes.
func respondReminderError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, services.ErrInvalidTimezone):
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid timezone"})
	case errors.Is(err, services.ErrRemindDatePast):
		c.JSON(http.StatusBadRequest, gin.H{"error": "Remind date must be today or in the future"})
	case errors.Is(err, services.ErrCoinNotWishlist):
		c.JSON(http.StatusConflict, gin.H{"error": "Reminders are only available for wishlist coins"})
	case errors.Is(err, services.ErrReminderNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "Not found"})
	default:
		respondError(c, http.StatusInternalServerError, "Failed to process reminder", err)
	}
}
