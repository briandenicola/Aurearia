package handlers

import (
	"net/http"
	"strconv"

	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
)

// AuctionEndingAdminHandler handles HTTP requests for auction ending run history.
type AuctionEndingAdminHandler struct {
	auctionEndingRepo *repository.AuctionEndingRepository
	scheduler         *services.AuctionEndingScheduler
	logger            *services.Logger
}

// NewAuctionEndingAdminHandler creates a new AuctionEndingAdminHandler.
func NewAuctionEndingAdminHandler(
	auctionEndingRepo *repository.AuctionEndingRepository,
	scheduler *services.AuctionEndingScheduler,
	logger *services.Logger,
) *AuctionEndingAdminHandler {
	return &AuctionEndingAdminHandler{
		auctionEndingRepo: auctionEndingRepo,
		scheduler:         scheduler,
		logger:            logger,
	}
}

// ListRuns returns paginated auction ending run history.
//
//	@Summary		List auction ending runs
//	@Description	Returns paginated history of auction ending check runs.
//	@Tags			Admin
//	@Produce		json
//	@Param			page	query		int	false	"Page number"	default(1)
//	@Param			limit	query		int	false	"Items per page"	default(20)
//	@Success		200		{object}	map[string]interface{}
//	@Failure		401		{object}	ErrorResponse
//	@Failure		403		{object}	ErrorResponse
//	@Failure		500		{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/admin/auction-ending-runs [get]
func (h *AuctionEndingAdminHandler) ListRuns(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	runs, total, err := h.auctionEndingRepo.ListRuns(page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list auction ending runs"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"runs":  runs,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

// TriggerRun manually triggers an auction ending check.
//
//	@Summary		Trigger manual auction ending check
//	@Description	Manually triggers an auction ending check for all users. Runs synchronously and returns the run details.
//	@Tags			Admin
//	@Produce		json
//	@Success		200	{object}	map[string]interface{}
//	@Failure		401	{object}	ErrorResponse
//	@Failure		403	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/admin/auction-ending/run [post]
func (h *AuctionEndingAdminHandler) TriggerRun(c *gin.Context) {
	triggerUserID := c.GetUint("userId")

	run, err := h.scheduler.RunNowWithTrigger(&triggerUserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to run auction ending check"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"runId":       run.ID,
		"lotsChecked": run.LotsChecked,
		"alertsSent":  run.AlertsSent,
		"status":      run.Status,
		"durationMs":  run.DurationMs,
	})
}
