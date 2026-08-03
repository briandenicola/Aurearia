package handlers

import (
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
)

type ShipmentHandler struct {
	svc *services.ShipmentService
}

func NewShipmentHandler(svc *services.ShipmentService) *ShipmentHandler {
	return &ShipmentHandler{svc: svc}
}

// UpsertForCoin creates or updates the shipment attached to a coin.
//
//	@Summary		Create or update coin shipment
//	@Description	Attaches or updates a shipment record (carrier, tracking, notes) for a coin owned by the authenticated user.
//	@Tags			Coins
//	@Accept			json
//	@Produce		json
//	@Param			id		path		int						true	"Coin ID"
//	@Param			body	body		ShipmentUpsertRequest	true	"Shipment payload"
//	@Success		200		{object}	ShipmentEnvelopeResponse
//	@Failure		400		{object}	ErrorResponse
//	@Failure		401		{object}	ErrorResponse
//	@Failure		404		{object}	ErrorResponse
//	@Failure		500		{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/coins/{id}/shipment [put]
func (h *ShipmentHandler) UpsertForCoin(c *gin.Context) {
	userID := c.GetUint("userId")
	coinID, err := parseUintPathParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid coin ID"})
		return
	}

	var req ShipmentUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	shipment, err := h.svc.UpsertShipmentForCoin(
		userID,
		coinID,
		models.ShipmentCarrier(req.Carrier),
		req.TrackingNumber,
		req.Notes,
		req.ManualCarrierName,
	)
	if err != nil {
		if handleShipmentServiceError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save shipment"})
		return
	}

	c.JSON(http.StatusOK, shipmentEnvelope(shipment))
}

// GetForCoin returns the shipment attached to a coin.
//
//	@Summary		Get coin shipment
//	@Description	Returns the shipment record and timeline for a coin owned by the authenticated user.
//	@Tags			Coins
//	@Produce		json
//	@Param			id	path		int	true	"Coin ID"
//	@Success		200	{object}	ShipmentEnvelopeResponse
//	@Failure		400	{object}	ErrorResponse
//	@Failure		401	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/coins/{id}/shipment [get]
func (h *ShipmentHandler) GetForCoin(c *gin.Context) {
	userID := c.GetUint("userId")
	coinID, err := parseUintPathParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid coin ID"})
		return
	}

	shipment, err := h.svc.GetShipmentForCoin(userID, coinID)
	if err != nil {
		if handleShipmentServiceError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch shipment"})
		return
	}
	c.JSON(http.StatusOK, shipmentEnvelope(shipment))
}

// DeleteForCoin removes the shipment attached to a coin.
//
//	@Summary		Delete coin shipment
//	@Description	Deletes a shipment record attached to a coin owned by the authenticated user.
//	@Tags			Coins
//	@Produce		json
//	@Param			id	path		int	true	"Coin ID"
//	@Success		200	{object}	MessageResponse
//	@Failure		400	{object}	ErrorResponse
//	@Failure		401	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/coins/{id}/shipment [delete]
func (h *ShipmentHandler) DeleteForCoin(c *gin.Context) {
	userID := c.GetUint("userId")
	coinID, err := parseUintPathParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid coin ID"})
		return
	}

	shipment, err := h.svc.GetShipmentForCoin(userID, coinID)
	if err != nil {
		if handleShipmentServiceError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch shipment"})
		return
	}
	if err := h.svc.DeleteShipment(userID, shipment.ID); err != nil {
		if handleShipmentServiceError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete shipment"})
		return
	}
	c.JSON(http.StatusOK, MessageResponse{Message: "Shipment deleted"})
}

// SetManualOverride toggles manual override for a coin shipment.
//
//	@Summary		Set shipment manual override
//	@Description	Enables/disables manual shipment status override for a coin shipment.
//	@Tags			Coins
//	@Accept			json
//	@Produce		json
//	@Param			id		path		int								true	"Coin ID"
//	@Param			body	body		ShipmentManualOverrideRequest	true	"Manual override payload"
//	@Success		200		{object}	ShipmentEnvelopeResponse
//	@Failure		400		{object}	ErrorResponse
//	@Failure		401		{object}	ErrorResponse
//	@Failure		404		{object}	ErrorResponse
//	@Failure		500		{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/coins/{id}/shipment/manual-override [put]
func (h *ShipmentHandler) SetManualOverride(c *gin.Context) {
	userID := c.GetUint("userId")
	coinID, err := parseUintPathParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid coin ID"})
		return
	}
	var req ShipmentManualOverrideRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	shipment, err := h.svc.GetShipmentForCoin(userID, coinID)
	if err != nil {
		if handleShipmentServiceError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch shipment"})
		return
	}
	updated, err := h.svc.SetManualOverride(userID, shipment.ID, req.Enabled, models.ShipmentStatus(req.Status), req.Note)
	if err != nil {
		if handleShipmentServiceError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update manual override"})
		return
	}
	c.JSON(http.StatusOK, shipmentEnvelope(updated))
}

// SyncForCoin runs carrier sync for the shipment attached to a coin.
//
//	@Summary		Sync coin shipment status
//	@Description	Fetches latest shipment status/events from the configured carrier integration for the coin shipment.
//	@Tags			Coins
//	@Produce		json
//	@Param			id	path		int	true	"Coin ID"
//	@Success		200	{object}	ShipmentEnvelopeResponse
//	@Failure		400	{object}	ErrorResponse
//	@Failure		401	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/coins/{id}/shipment/sync [post]
func (h *ShipmentHandler) SyncForCoin(c *gin.Context) {
	userID := c.GetUint("userId")
	coinID, err := parseUintPathParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid coin ID"})
		return
	}
	shipment, err := h.svc.GetShipmentForCoin(userID, coinID)
	if err != nil {
		if handleShipmentServiceError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch shipment"})
		return
	}
	updated, err := h.svc.SyncShipment(c.Request.Context(), shipment.ID, userID)
	if err != nil {
		if handleShipmentServiceError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to sync shipment"})
		return
	}
	c.JSON(http.StatusOK, shipmentEnvelope(updated))
}

func handleShipmentServiceError(c *gin.Context, err error) bool {
	switch {
	case errors.Is(err, services.ErrShipmentCoinNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "Coin not found"})
		return true
	case errors.Is(err, services.ErrShipmentNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "Shipment not found"})
		return true
	case errors.Is(err, services.ErrShipmentCarrierRequired),
		errors.Is(err, services.ErrShipmentTrackingRequired),
		errors.Is(err, services.ErrShipmentCarrierNameRequired),
		errors.Is(err, services.ErrShipmentCarrierUnsupported):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return true
	default:
		return false
	}
}

func shipmentEnvelope(shipment *models.Shipment) ShipmentEnvelopeResponse {
	return ShipmentEnvelopeResponse{
		Shipment:    shipment,
		TrackingURL: shipmentTrackingURL(shipment),
	}
}

func shipmentTrackingURL(shipment *models.Shipment) string {
	if shipment == nil || strings.TrimSpace(shipment.TrackingNumber) == "" {
		return ""
	}
	tracking := url.QueryEscape(strings.TrimSpace(shipment.TrackingNumber))
	switch shipment.Carrier {
	case models.ShipmentCarrierUSPS:
		return "https://tools.usps.com/go/TrackConfirmAction?qtc_tLabels1=" + tracking
	case models.ShipmentCarrierUPS:
		return "https://www.ups.com/track?tracknum=" + tracking
	case models.ShipmentCarrierFedEx:
		return "https://www.fedex.com/fedextrack/?trknbr=" + tracking
	default:
		return ""
	}
}

func parseUintPathParam(c *gin.Context, key string) (uint, error) {
	value, err := strconv.ParseUint(c.Param(key), 10, 32)
	return uint(value), err
}
