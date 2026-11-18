package handlers

import (
	"net/http"
	"strconv"

	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
)

// ImageVariantAdminHandler exposes admin endpoints for the image variant check.
type ImageVariantAdminHandler struct {
	scheduler *services.ImageVariantScheduler
}

// NewImageVariantAdminHandler creates a new ImageVariantAdminHandler.
func NewImageVariantAdminHandler(scheduler *services.ImageVariantScheduler) *ImageVariantAdminHandler {
	return &ImageVariantAdminHandler{scheduler: scheduler}
}

// TriggerRun starts an image variant pass in the background.
//
//	@Summary		Run the image variant check now
//	@Description	Starts a pass that makes the thumb and medium variants coin photos are missing, whether or not the daily check is turned on, and returns at once. The run history shows its outcome. Admin only.
//	@Tags			Admin
//	@Produce		json
//	@Success		202	{object}	map[string]string
//	@Failure		401	{object}	ErrorResponse
//	@Failure		403	{object}	ErrorResponse
//	@Failure		409	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/admin/image-variants/run [post]
func (h *ImageVariantAdminHandler) TriggerRun(c *gin.Context) {
	if !h.scheduler.StartManualRun() {
		c.JSON(http.StatusConflict, gin.H{"error": "An image variant pass is already running. Its result will appear in the run history when it finishes."})
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"message": "Image variant pass started. Its result will appear in the run history when it finishes."})
}

// GetStatus returns the image variant scheduler's runtime status.
//
//	@Summary		Get image variant scheduler status
//	@Description	Returns whether the daily image variant check is on, whether a pass is running, and the time until the next one. Admin only.
//	@Tags			Admin
//	@Produce		json
//	@Success		200	{object}	services.SchedulerStatus
//	@Failure		401	{object}	ErrorResponse
//	@Failure		403	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/admin/image-variants/status [get]
func (h *ImageVariantAdminHandler) GetStatus(c *gin.Context) {
	c.JSON(http.StatusOK, h.scheduler.GetStatus())
}

// ListRuns returns the image variant check's run history.
//
//	@Summary		List image variant runs
//	@Description	Returns paginated history of image variant passes, newest first. Admin only.
//	@Tags			Admin
//	@Produce		json
//	@Param			page	query		int	false	"Page number"	default(1)
//	@Param			limit	query		int	false	"Items per page"	default(20)
//	@Success		200		{object}	map[string]interface{}
//	@Failure		401		{object}	ErrorResponse
//	@Failure		403		{object}	ErrorResponse
//	@Failure		500		{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/admin/image-variant-runs [get]
func (h *ImageVariantAdminHandler) ListRuns(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	runs, total, err := h.scheduler.ListRuns(page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list image variant runs"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"runs": runs, "total": total, "page": page, "limit": limit})
}
