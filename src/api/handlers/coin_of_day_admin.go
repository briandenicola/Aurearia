package handlers

import (
	"net/http"
	"strconv"

	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
)

// CoinOfDayAdminHandler exposes admin endpoints for the Coin of the Day scheduler.
type CoinOfDayAdminHandler struct {
	scheduler *services.CoinOfDayScheduler
	logger    *services.Logger
}

// NewCoinOfDayAdminHandler creates a new CoinOfDayAdminHandler.
func NewCoinOfDayAdminHandler(scheduler *services.CoinOfDayScheduler, logger *services.Logger) *CoinOfDayAdminHandler {
	return &CoinOfDayAdminHandler{scheduler: scheduler, logger: logger}
}

// TriggerRun queues a manual Coin of the Day pick for asynchronous processing.
//
//	@Summary		Trigger manual coin-of-the-day pick
//	@Description	Queues a run and returns immediately with a durable run id.
//	@Tags			Admin
//	@Produce		json
//	@Success		202	{object}	map[string]interface{}
//	@Failure		401	{object}	ErrorResponse
//	@Failure		403	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/admin/coin-of-day/run [post]
func (h *CoinOfDayAdminHandler) TriggerRun(c *gin.Context) {
	triggerUserID := c.GetUint("userId")
	run, err := h.scheduler.RunNowWithTrigger(&triggerUserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to run coin-of-day pick"})
		return
	}
	c.JSON(http.StatusAccepted, gin.H{
		"runId":  run.ID,
		"status": run.Status,
	})
}

// ListRuns returns paginated coin-of-the-day run history.
//
//	@Summary		List coin-of-the-day runs
//	@Description	Returns paginated history of Coin of the Day runs.
//	@Tags			Admin
//	@Produce		json
//	@Param			page	query		int	false	"Page number"	default(1)
//	@Param			limit	query		int	false	"Items per page"	default(20)
//	@Success		200		{object}	map[string]interface{}
//	@Failure		401		{object}	ErrorResponse
//	@Failure		403		{object}	ErrorResponse
//	@Failure		500		{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/admin/coin-of-day-runs [get]
func (h *CoinOfDayAdminHandler) ListRuns(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	runs, total, err := h.scheduler.ListRuns(page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list coin-of-day runs"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"runs":  runs,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

// GetRun returns a single coin-of-the-day run.
//
//	@Summary		Get coin-of-the-day run
//	@Description	Returns one Coin of the Day run with status and counters.
//	@Tags			Admin
//	@Produce		json
//	@Param			id	path		int	true	"Run ID"
//	@Success		200	{object}	models.CoinOfDayRun
//	@Failure		400	{object}	ErrorResponse
//	@Failure		401	{object}	ErrorResponse
//	@Failure		403	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/admin/coin-of-day-runs/{id} [get]
func (h *CoinOfDayAdminHandler) GetRun(c *gin.Context) {
	runID, err := strconv.ParseUint(c.Param("id"), 10, strconv.IntSize)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid run ID"})
		return
	}
	run, err := h.scheduler.GetRun(uint(runID))
	if err != nil {
		if repository.IsRecordNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Run not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get coin-of-day run"})
		return
	}
	c.JSON(http.StatusOK, run)
}
