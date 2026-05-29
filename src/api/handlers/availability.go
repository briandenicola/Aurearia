package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
)

// AvailabilityHandler handles HTTP requests for wishlist availability checks.
type AvailabilityHandler struct {
	svc       *services.AvailabilityService
	availRepo *repository.AvailabilityRepository
	coinRepo  *repository.CoinRepository
}

// NewAvailabilityHandler creates a new AvailabilityHandler.
func NewAvailabilityHandler(
	svc *services.AvailabilityService,
	availRepo *repository.AvailabilityRepository,
	coinRepo *repository.CoinRepository,
) *AvailabilityHandler {
	return &AvailabilityHandler{svc: svc, availRepo: availRepo, coinRepo: coinRepo}
}

// CheckAvailability triggers a wishlist availability check for the authenticated user.
//
//	@Summary		Check wishlist availability
//	@Description	Checks all wishlist items with reference URLs to see if they are still available for purchase.
//	@Tags			Wishlist
//	@Produce		json
//	@Success		200	{object}	map[string]interface{}
//	@Failure		401	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/wishlist/check-availability [post]
func (h *AvailabilityHandler) CheckAvailability(c *gin.Context) {
	userID := c.GetUint("userId")
	triggerUserID := userID

	run, err := h.svc.CheckWishlistForUser(userID, "manual", &triggerUserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check availability"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"runId":        run.ID,
		"coinsChecked": run.CoinsChecked,
		"available":    run.Available,
		"unavailable":  run.Unavailable,
		"unknown":      run.Unknown,
		"durationMs":   run.DurationMs,
	})
}

// UpdateListingStatus allows a user to dismiss or reset a coin's listing status.
//
//	@Summary		Update coin listing status
//	@Description	Updates the listing status of a coin (e.g., dismiss an unavailable notice).
//	@Tags			Wishlist
//	@Accept			json
//	@Produce		json
//	@Param			id		path		int		true	"Coin ID"
//	@Param			body	body		object	true	"Status update"
//	@Success		200		{object}	map[string]interface{}
//	@Failure		400		{object}	ErrorResponse
//	@Failure		404		{object}	ErrorResponse
//	@Failure		500		{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/coins/{id}/listing-status [put]
func (h *AvailabilityHandler) UpdateListingStatus(c *gin.Context) {
	userID := c.GetUint("userId")
	coinID, err := strconv.ParseUint(c.Param("id"), 10, strconv.IntSize)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid coin ID"})
		return
	}

	var body struct {
		Status string `json:"status"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Verify coin belongs to user
	exists, err := h.coinRepo.CoinExists(uint(coinID), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Coin not found"})
		return
	}

	reason := ""
	if body.Status == "" {
		reason = "Status cleared by user"
	}

	if err := h.coinRepo.UpdateListingStatus(uint(coinID), body.Status, reason, time.Now()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update listing status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Listing status updated"})
}

// ListRuns returns paginated availability check run history (admin only).
//
//	@Summary		List availability check runs
//	@Description	Returns paginated history of wishlist availability check runs.
//	@Tags			Admin
//	@Produce		json
//	@Param			page	query		int	false	"Page number"	default(1)
//	@Param			limit	query		int	false	"Items per page"	default(20)
//	@Success		200		{object}	map[string]interface{}
//	@Failure		401		{object}	ErrorResponse
//	@Failure		403		{object}	ErrorResponse
//	@Failure		500		{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/admin/availability-runs [get]
func (h *AvailabilityHandler) ListRuns(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	runs, total, err := h.availRepo.ListRuns(page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list runs"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"runs":  runs,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

// GetRunDetail returns a single availability run with all per-coin results.
//
//	@Summary		Get availability run detail
//	@Description	Returns a single availability check run with all per-coin results.
//	@Tags			Admin
//	@Produce		json
//	@Param			id	path		int	true	"Run ID"
//	@Success		200	{object}	map[string]interface{}
//	@Failure		400	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/admin/availability-runs/{id} [get]
func (h *AvailabilityHandler) GetRunDetail(c *gin.Context) {
	runID, err := strconv.ParseUint(c.Param("id"), 10, strconv.IntSize)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid run ID"})
		return
	}

	run, err := h.availRepo.GetRunWithResults(uint(runID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Run not found"})
		return
	}

	c.JSON(http.StatusOK, run)
}
