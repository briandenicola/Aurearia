package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
)

// AvailabilityHandler handles HTTP requests for wishlist availability checks.
type AvailabilityHandler struct {
	svc            *services.AvailabilityService
	scheduler      *services.AvailabilityScheduler
	availRepo      *repository.AvailabilityRepository
	availCycleRepo *repository.AvailabilityCycleRepository
	coinRepo       *repository.CoinRepository
}

// NewAvailabilityHandler creates a new AvailabilityHandler.
func NewAvailabilityHandler(
	svc *services.AvailabilityService,
	scheduler *services.AvailabilityScheduler,
	availRepo *repository.AvailabilityRepository,
	coinRepo *repository.CoinRepository,
) *AvailabilityHandler {
	return &AvailabilityHandler{svc: svc, scheduler: scheduler, availRepo: availRepo, coinRepo: coinRepo}
}

// WithCycleRepo attaches the AvailabilityCycleRepository used by the admin cycle endpoints.
func (h *AvailabilityHandler) WithCycleRepo(availCycleRepo *repository.AvailabilityCycleRepository) *AvailabilityHandler {
	h.availCycleRepo = availCycleRepo
	return h
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

	run, err := h.svc.CheckWishlistForUser(userID, "owner", &triggerUserID, nil)
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

// TriggerRun enqueues an asynchronous wishlist availability check cycle for all users.
//
//	@Summary		Trigger manual wishlist availability check
//	@Description	Enqueues a wishlist availability check cycle (one child run per owner) and returns immediately. Duplicate requests while a cycle is queued or running are rejected.
//	@Tags			Admin
//	@Produce		json
//	@Success		202	{object}	map[string]interface{}
//	@Failure		401	{object}	ErrorResponse
//	@Failure		403	{object}	ErrorResponse
//	@Failure		409	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/admin/availability/run [post]
func (h *AvailabilityHandler) TriggerRun(c *gin.Context) {
	triggerUserID := c.GetUint("userId")

	cycle, err := h.scheduler.RunNowWithTrigger(&triggerUserID)
	if err != nil {
		if errors.Is(err, services.ErrAvailabilityRunInProgress) {
			c.JSON(http.StatusConflict, gin.H{"error": "An availability check is already queued or running"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to enqueue availability check"})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"cycleId": cycle.ID,
		"status":  cycle.Status,
		"message": "Availability check queued",
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

// ListRuns returns paginated legacy availability check run history (admin only).
//
//	@Summary		List legacy availability check runs
//	@Description	Returns paginated history of legacy (pre-cycle) wishlist availability check runs, including UserID=0 admin rows.
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

// GetRunDetail returns a single legacy availability run with all per-coin results.
//
//	@Summary		Get legacy availability run detail
//	@Description	Returns a single legacy (pre-cycle) availability check run with all per-coin results.
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

// ListOwnerRuns returns the authenticated owner's own availability run history.
//
//	@Summary		List my wishlist availability runs
//	@Description	Returns paginated availability check run history scoped to the authenticated user.
//	@Tags			Wishlist
//	@Produce		json
//	@Param			page	query		int	false	"Page number"	default(1)
//	@Param			limit	query		int	false	"Items per page"	default(20)
//	@Success		200		{object}	map[string]interface{}
//	@Failure		401		{object}	ErrorResponse
//	@Failure		500		{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/wishlist/availability-runs [get]
func (h *AvailabilityHandler) ListOwnerRuns(c *gin.Context) {
	userID := c.GetUint("userId")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	runs, total, err := h.availRepo.ListRunsForOwner(userID, page, limit)
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

// GetOwnerRunDetail returns a single availability run (with per-coin results) owned by the
// authenticated user. Returns 404 for runs belonging to a different owner.
//
//	@Summary		Get my wishlist availability run detail
//	@Description	Returns a single availability check run with per-coin results, scoped to the authenticated user.
//	@Tags			Wishlist
//	@Produce		json
//	@Param			id	path		int	true	"Run ID"
//	@Success		200	{object}	map[string]interface{}
//	@Failure		400	{object}	ErrorResponse
//	@Failure		401	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/wishlist/availability-runs/{id} [get]
func (h *AvailabilityHandler) GetOwnerRunDetail(c *gin.Context) {
	userID := c.GetUint("userId")
	runID, err := strconv.ParseUint(c.Param("id"), 10, strconv.IntSize)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid run ID"})
		return
	}

	run, err := h.availRepo.GetOwnedRunWithResults(userID, uint(runID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Run not found"})
		return
	}

	c.JSON(http.StatusOK, run)
}

// ListCycles returns paginated availability cycle (parent) history (admin only).
//
//	@Summary		List availability cycles
//	@Description	Returns paginated history of admin/scheduled availability cycles with roll-up child counts.
//	@Tags			Admin
//	@Produce		json
//	@Param			page	query		int	false	"Page number"	default(1)
//	@Param			limit	query		int	false	"Items per page"	default(20)
//	@Success		200		{object}	map[string]interface{}
//	@Failure		401		{object}	ErrorResponse
//	@Failure		403		{object}	ErrorResponse
//	@Failure		500		{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/admin/availability-cycles [get]
func (h *AvailabilityHandler) ListCycles(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	cycles, total, err := h.availCycleRepo.ListCycles(page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list cycles"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"cycles": cycles,
		"total":  total,
		"page":   page,
		"limit":  limit,
	})
}

// GetCycleDetail returns a single availability cycle with its child run summaries.
// No per-coin results are exposed here — drill into a specific child run via the owner or
// legacy run-detail endpoints for that.
//
//	@Summary		Get availability cycle detail
//	@Description	Returns a single availability cycle with its per-owner child run summaries (no per-coin results).
//	@Tags			Admin
//	@Produce		json
//	@Param			id	path		int	true	"Cycle ID"
//	@Success		200	{object}	map[string]interface{}
//	@Failure		400	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/admin/availability-cycles/{id} [get]
func (h *AvailabilityHandler) GetCycleDetail(c *gin.Context) {
	cycleID, err := strconv.ParseUint(c.Param("id"), 10, strconv.IntSize)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid cycle ID"})
		return
	}

	cycle, err := h.availCycleRepo.GetCycleWithChildren(uint(cycleID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Cycle not found"})
		return
	}

	c.JSON(http.StatusOK, cycle)
}
