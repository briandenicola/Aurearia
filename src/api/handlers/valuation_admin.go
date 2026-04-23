package handlers

import (
	"net/http"
	"strconv"

	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
)

// ValuationAdminHandler handles HTTP requests for valuation run history.
type ValuationAdminHandler struct {
	valRepo *repository.ValuationRepository
	valSvc  *services.ValuationService
}

// NewValuationAdminHandler creates a new ValuationAdminHandler.
func NewValuationAdminHandler(
	valRepo *repository.ValuationRepository,
	valSvc *services.ValuationService,
) *ValuationAdminHandler {
	return &ValuationAdminHandler{valRepo: valRepo, valSvc: valSvc}
}

// ListValuationRuns returns paginated valuation run history.
//
//	@Summary		List valuation runs
//	@Description	Returns paginated history of scheduled collection valuation runs.
//	@Tags			Admin
//	@Produce		json
//	@Param			page	query		int	false	"Page number"	default(1)
//	@Param			limit	query		int	false	"Items per page"	default(20)
//	@Success		200		{object}	map[string]interface{}
//	@Failure		401		{object}	ErrorResponse
//	@Failure		403		{object}	ErrorResponse
//	@Failure		500		{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/admin/valuation-runs [get]
func (h *ValuationAdminHandler) ListRuns(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	runs, total, err := h.valRepo.ListRuns(page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list valuation runs"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"runs":  runs,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

// GetRunDetail returns a single valuation run with all per-coin results.
//
//	@Summary		Get valuation run detail
//	@Description	Returns a single valuation run with all per-coin results.
//	@Tags			Admin
//	@Produce		json
//	@Param			id	path		int	true	"Run ID"
//	@Success		200	{object}	map[string]interface{}
//	@Failure		400	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/admin/valuation-runs/{id} [get]
func (h *ValuationAdminHandler) GetRunDetail(c *gin.Context) {
	runID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid run ID"})
		return
	}

	run, err := h.valRepo.GetRunWithResults(uint(runID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Valuation run not found"})
		return
	}

	c.JSON(http.StatusOK, run)
}

// TriggerValuation manually triggers a valuation run for the admin user.
//
//	@Summary		Trigger manual valuation
//	@Description	Manually triggers a collection valuation for all users. Runs asynchronously.
//	@Tags			Admin
//	@Produce		json
//	@Success		202	{object}	map[string]interface{}
//	@Failure		401	{object}	ErrorResponse
//	@Failure		403	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/admin/valuation-runs/trigger [post]
func (h *ValuationAdminHandler) TriggerValuation(c *gin.Context) {
	triggerUserID := c.GetUint("userId")

	// Get all users with owned coins
	userIDs, err := h.valRepo.GetUsersWithOwnedCoins()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch users"})
		return
	}

	if len(userIDs) == 0 {
		c.JSON(http.StatusOK, gin.H{"message": "No users with owned coins found"})
		return
	}

	// Run valuation asynchronously — return immediately
	go func() {
		for _, userID := range userIDs {
			_, err := h.valSvc.ValuateCollectionForUser(userID, "manual", &triggerUserID)
			if err != nil {
				services.AppLogger.Error("valuation", "Manual valuation failed for user %d: %s", userID, err)
			}
		}
	}()

	c.JSON(http.StatusAccepted, gin.H{"message": "Valuation started", "users": len(userIDs)})
}

// CancelValuation cancels a running valuation run.
//
//	@Summary		Cancel a valuation run
//	@Description	Signals a running valuation to stop after the current coin.
//	@Tags			Admin
//	@Produce		json
//	@Param			id	path		int	true	"Run ID"
//	@Success		200	{object}	map[string]interface{}
//	@Failure		400	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/admin/valuation-runs/{id}/cancel [post]
func (h *ValuationAdminHandler) CancelValuation(c *gin.Context) {
	runID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid run ID"})
		return
	}

	found := h.valSvc.CancelRun(uint(runID))
	if found {
		c.JSON(http.StatusOK, gin.H{"message": "Cancellation signal sent"})
	} else {
		c.JSON(http.StatusOK, gin.H{"message": "Run marked as cancelled (was not active in memory)"})
	}
}
