package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
)

const CollectorProfileBodyLimit int64 = 16 * 1024

type CollectorProfileHandler struct {
	service *services.CollectorProfileService
}

type collectorProfileRequest struct {
	BudgetMin           *float64 `json:"budgetMin"`
	BudgetMax           *float64 `json:"budgetMax"`
	Currency            *string  `json:"currency"`
	PreferredPeriods    []string `json:"preferredPeriods"`
	PreferredCategories []string `json:"preferredCategories"`
	ExcludedCategories  []string `json:"excludedCategories"`
	PreferredDealers    []string `json:"preferredDealers"`
	CollectingGoals     []string `json:"collectingGoals"`
}

func NewCollectorProfileHandler(service *services.CollectorProfileService) *CollectorProfileHandler {
	return &CollectorProfileHandler{service: service}
}

// Get returns the authenticated owner's private collector profile.
//
//	@Summary		Get collector profile
//	@Description	Returns the authenticated owner's saved collector profile or neutral defaults.
//	@Tags			Collector Profile
//	@Produce		json
//	@Success		200	{object}	services.CollectorProfileResponse
//	@Failure		401	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/collector-profile [get]
func (h *CollectorProfileHandler) Get(c *gin.Context) {
	userID := c.GetUint("userId")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}
	profile, err := h.service.Get(userID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to load collector profile", err)
		return
	}
	c.JSON(http.StatusOK, profile)
}

// Put atomically replaces the authenticated owner's private collector profile.
//
//	@Summary		Replace collector profile
//	@Description	Validates and atomically replaces the authenticated owner's complete collector profile.
//	@Tags			Collector Profile
//	@Accept			json
//	@Produce		json
//	@Param			profile	body		collectorProfileRequest	true	"Collector profile"
//	@Success		200		{object}	services.CollectorProfileResponse
//	@Failure		400		{object}	ErrorResponse
//	@Failure		401		{object}	ErrorResponse
//	@Failure		413		{object}	ErrorResponse
//	@Failure		500		{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/collector-profile [put]
func (h *CollectorProfileHandler) Put(c *gin.Context) {
	userID := c.GetUint("userId")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, CollectorProfileBodyLimit)
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	var request collectorProfileRequest
	if err := decoder.Decode(&request); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "Collector profile payload is too large"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid collector profile payload"})
		return
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "Collector profile payload is too large"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid collector profile payload"})
		return
	}
	profile, err := h.service.Replace(userID, services.CollectorProfileInput{
		BudgetMin: request.BudgetMin, BudgetMax: request.BudgetMax, Currency: request.Currency,
		PreferredPeriods: request.PreferredPeriods, PreferredCategories: request.PreferredCategories,
		ExcludedCategories: request.ExcludedCategories, PreferredDealers: request.PreferredDealers,
		CollectingGoals: request.CollectingGoals,
	})
	if err != nil {
		var validation *services.CollectorProfileValidationError
		if errors.As(err, &validation) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid collector profile", "field": validation.Field})
			return
		}
		respondError(c, http.StatusInternalServerError, "Failed to save collector profile", err)
		return
	}
	c.JSON(http.StatusOK, profile)
}
