package handlers

import (
	"net/http"
	"strconv"

	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
)

type AuctionAlertAdminHandler struct {
	scheduler *services.AuctionAlertScheduler
	runRepo   *repository.AuctionAlertRunRepository
}

func NewAuctionAlertAdminHandler(
	scheduler *services.AuctionAlertScheduler,
	runRepo *repository.AuctionAlertRunRepository,
) *AuctionAlertAdminHandler {
	return &AuctionAlertAdminHandler{scheduler: scheduler, runRepo: runRepo}
}

// ListRuns returns paginated auction alert run history.
//
//	@Summary		List auction alert runs
//	@Description	Returns paginated history of price alert and bid reminder scheduler runs.
//	@Tags			Admin
//	@Produce		json
//	@Param			page	query		int	false	"Page number"	default(1)
//	@Param			limit	query		int	false	"Items per page"	default(20)
//	@Success		200		{object}	map[string]interface{}
//	@Failure		401		{object}	ErrorResponse
//	@Failure		403		{object}	ErrorResponse
//	@Failure		500		{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/admin/auction-alert-runs [get]
func (h *AuctionAlertAdminHandler) ListRuns(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	runs, total, err := h.runRepo.ListRuns(page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list auction alert runs"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"runs":  runs,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

// RunNow manually triggers an auction alert evaluation.
//
//	@Summary		Trigger manual auction alert check
//	@Description	Refreshes watched auction lots and evaluates price alerts and bid reminders.
//	@Tags			Admin
//	@Produce		json
//	@Success		200	{object}	map[string]interface{}
//	@Failure		401	{object}	ErrorResponse
//	@Failure		403	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/admin/auction-alerts/run [post]
func (h *AuctionAlertAdminHandler) RunNow(c *gin.Context) {
	triggerUserID := c.GetUint("userId")
	run, err := h.scheduler.RunNowWithTrigger(&triggerUserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to run auction alert check"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"runId":                run.ID,
		"lotsChecked":          run.LotsChecked,
		"priceAlertsTriggered": run.PriceAlertsTriggered,
		"bidRemindersSent":     run.BidRemindersSent,
		"status":               run.Status,
		"durationMs":           run.DurationMs,
	})
}

// GetStatus returns auction alert scheduler status.
//
//	@Summary		Get auction alert scheduler status
//	@Description	Returns runtime status for the auction alert scheduler.
//	@Tags			Admin
//	@Produce		json
//	@Success		200	{object}	services.SchedulerStatus
//	@Failure		401	{object}	ErrorResponse
//	@Failure		403	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/admin/auction-alerts/status [get]
func (h *AuctionAlertAdminHandler) GetStatus(c *gin.Context) {
	c.JSON(http.StatusOK, h.scheduler.GetStatus())
}
