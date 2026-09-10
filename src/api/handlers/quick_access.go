package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
)

type QuickAccessHandler struct {
	service *services.QuickAccessService
	logger  *services.Logger
}

func NewQuickAccessHandler(service *services.QuickAccessService, logger *services.Logger) *QuickAccessHandler {
	return &QuickAccessHandler{service: service, logger: logger}
}

// List returns the authenticated user's mixed Quick Access pins.
//
//	@Summary		List Quick Access pins
//	@Description	Returns eligible user-owned pins newest first as a typed discriminated union.
//	@Tags			Quick Access
//	@Produce		json
//	@Success		200	{object}	services.QuickAccessListDTO
//	@Failure		401	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/quick-access [get]
func (h *QuickAccessHandler) List(c *gin.Context) {
	result, err := h.service.List(c.GetUint("userId"))
	if err != nil {
		h.logError("list", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load quick access"})
		return
	}
	c.JSON(http.StatusOK, result)
}

// Pin adds an eligible target to Quick Access.
//
//	@Summary		Pin a Quick Access target
//	@Description	Idempotently pins one eligible user-owned target while preserving its original pin time.
//	@Tags			Quick Access
//	@Produce		json
//	@Param			type	path		string	true	"Target type: coin, coin_set, auction_lot, calendar_event"
//	@Param			id		path		int		true	"Target ID"
//	@Success		200		{object}	services.QuickAccessItemDTO
//	@Success		201		{object}	services.QuickAccessItemDTO
//	@Failure		400		{object}	ErrorResponse
//	@Failure		401		{object}	ErrorResponse
//	@Failure		404		{object}	ErrorResponse
//	@Failure		500		{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/quick-access/{type}/{id} [put]
func (h *QuickAccessHandler) Pin(c *gin.Context) {
	targetType, targetID, ok := parseQuickAccessTarget(c)
	if !ok {
		return
	}
	item, created, err := h.service.Pin(c.GetUint("userId"), targetType, targetID)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrInvalidQuickAccessTarget):
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid quick access target"})
		case errors.Is(err, services.ErrPinLimitReached):
			c.JSON(http.StatusBadRequest, gin.H{"error": services.ErrPinLimitReached.Error()})
		case errors.Is(err, services.ErrQuickAccessTargetNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "Quick access item not found"})
		default:
			h.logError("pin", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to pin quick access item"})
		}
		return
	}
	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	c.JSON(status, item)
}

// Unpin removes a target from Quick Access.
//
//	@Summary		Unpin a Quick Access target
//	@Description	Idempotently removes one target from the authenticated user's Quick Access list.
//	@Tags			Quick Access
//	@Param			type	path	string	true	"Target type: coin, coin_set, auction_lot, calendar_event"
//	@Param			id		path	int		true	"Target ID"
//	@Success		204
//	@Failure		400	{object}	ErrorResponse
//	@Failure		401	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/quick-access/{type}/{id} [delete]
func (h *QuickAccessHandler) Unpin(c *gin.Context) {
	targetType, targetID, ok := parseQuickAccessTarget(c)
	if !ok {
		return
	}
	if err := h.service.Unpin(c.GetUint("userId"), targetType, targetID); err != nil {
		if errors.Is(err, services.ErrInvalidQuickAccessTarget) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid quick access target"})
			return
		}
		h.logError("unpin", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to unpin quick access item"})
		return
	}
	c.Status(http.StatusNoContent)
}

func parseQuickAccessTarget(c *gin.Context) (targetType models.QuickAccessTargetType, targetID uint, ok bool) {
	parsedType, err := services.ParseQuickAccessTargetType(c.Param("type"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid quick access target"})
		return "", 0, false
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid quick access target"})
		return "", 0, false
	}
	return parsedType, uint(id), true
}

func (h *QuickAccessHandler) logError(operation string, err error) {
	if h.logger != nil {
		h.logger.Error("quick-access", "%s failed: %v", operation, err)
	}
}
