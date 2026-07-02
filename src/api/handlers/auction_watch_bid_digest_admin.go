package handlers

import (
	"net/http"
	"strconv"

	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
)

type AuctionWatchBidDigestAdminHandler struct {
	scheduler *services.AuctionWatchBidDigestScheduler
	runRepo   *repository.AuctionWatchBidDigestRepository
}

func NewAuctionWatchBidDigestAdminHandler(
	scheduler *services.AuctionWatchBidDigestScheduler,
	runRepo *repository.AuctionWatchBidDigestRepository,
) *AuctionWatchBidDigestAdminHandler {
	return &AuctionWatchBidDigestAdminHandler{scheduler: scheduler, runRepo: runRepo}
}

// RunNow manually triggers an auction watch bid digest.
//
//	@Summary		Trigger manual auction watch bid digest
//	@Description	Refreshes watched auction lots and sends current-bid digest notifications.
//	@Tags			Admin
//	@Produce		json
//	@Success		202	{object}	map[string]interface{}
//	@Failure		401	{object}	ErrorResponse
//	@Failure		403	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/admin/auction-watch-bid-digest/run [post]
func (h *AuctionWatchBidDigestAdminHandler) RunNow(c *gin.Context) {
	if err := h.scheduler.RunNow(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to run auction watch bid digest"})
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"message": "Auction watch bid digest started"})
}

// ListRuns returns paginated auction watch bid digest run history.
//
//	@Summary		List auction watch bid digest runs
//	@Description	Returns paginated history of auction watch bid digest runs.
//	@Tags			Admin
//	@Produce		json
//	@Param			page	query		int	false	"Page number"	default(1)
//	@Param			limit	query		int	false	"Items per page"	default(10)
//	@Success		200		{object}	map[string]interface{}
//	@Failure		401		{object}	ErrorResponse
//	@Failure		403		{object}	ErrorResponse
//	@Failure		500		{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/admin/auction-watch-bid-digest-runs [get]
func (h *AuctionWatchBidDigestAdminHandler) ListRuns(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	runs, total, err := h.runRepo.ListRuns(page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list auction watch bid digest runs"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"runs":  runs,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

// GetStatus returns auction watch bid digest scheduler status.
//
//	@Summary		Get auction watch bid digest scheduler status
//	@Description	Returns runtime status for the auction watch bid digest scheduler.
//	@Tags			Admin
//	@Produce		json
//	@Success		200	{object}	services.SchedulerStatus
//	@Failure		401	{object}	ErrorResponse
//	@Failure		403	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/admin/auction-watch-bid-digest/status [get]
func (h *AuctionWatchBidDigestAdminHandler) GetStatus(c *gin.Context) {
	c.JSON(http.StatusOK, h.scheduler.GetStatus())
}
