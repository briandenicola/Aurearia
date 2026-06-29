package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
)

type AIJobHandler struct {
	service *services.AIJobService
	logger  *services.Logger
}

func NewAIJobHandler(service *services.AIJobService, logger *services.Logger) *AIJobHandler {
	return &AIJobHandler{service: service, logger: logger}
}

// Analyze enqueues AI analysis for one side of a coin.
//
//	@Summary		Queue coin image analysis
//	@Description	Queues asynchronous AI-powered analysis for the obverse or reverse image of a coin. If side is omitted, all coin images are analyzed.
//	@Tags			Analysis
//	@Produce		json
//	@Param			id		path		int		true	"Coin ID"
//	@Param			side	query		string	false	"Analyze a specific side"	Enums(obverse, reverse)
//	@Success		202		{object}	services.AIJobSubmissionResponse
//	@Failure		400		{object}	ErrorResponse
//	@Failure		401		{object}	ErrorResponse
//	@Failure		404		{object}	ErrorResponse
//	@Failure		500		{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/coins/{id}/analyze [post]
func (h *AIJobHandler) Analyze(c *gin.Context) {
	coinID, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	job, _, err := h.service.EnqueueAnalysis(c.GetUint("userId"), coinID, c.Query("side"))
	if err != nil {
		h.respondJobError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, services.AIJobSubmissionResponse{Job: *job})
}

// EstimateValue enqueues an AI value estimate for a coin.
//
//	@Summary		Queue coin value estimate
//	@Description	Queues asynchronous AI-powered current value estimation for a coin owned by the authenticated user.
//	@Tags			Agent
//	@Produce		json
//	@Param			id	path		int	true	"Coin ID"
//	@Success		202	{object}	services.AIJobSubmissionResponse
//	@Failure		400	{object}	ErrorResponse
//	@Failure		401	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/coins/{id}/estimate-value [post]
func (h *AIJobHandler) EstimateValue(c *gin.Context) {
	coinID, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	job, _, err := h.service.EnqueueValueEstimate(c.GetUint("userId"), coinID)
	if err != nil {
		h.respondJobError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, services.AIJobSubmissionResponse{Job: *job})
}

// GetJob returns an AI job by ID for the authenticated user.
//
//	@Summary		Get AI job
//	@Description	Returns user-scoped AI job status and result details.
//	@Tags			AI Jobs
//	@Produce		json
//	@Param			id	path		int	true	"AI Job ID"
//	@Success		200	{object}	models.AIJob
//	@Failure		400	{object}	ErrorResponse
//	@Failure		401	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/ai-jobs/{id} [get]
func (h *AIJobHandler) GetJob(c *gin.Context) {
	jobID, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	job, err := h.service.GetJob(c.GetUint("userId"), jobID)
	if err != nil {
		if repository.IsRecordNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "AI job not found"})
			return
		}
		h.logger.Error("ai-jobs", "Failed to get job %d: %v", jobID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get AI job"})
		return
	}
	c.JSON(http.StatusOK, job)
}

// ListCoinJobs returns AI jobs for one coin.
//
//	@Summary		List coin AI jobs
//	@Description	Returns user-scoped AI jobs for a coin. Set activeOnly=true to include only queued/running jobs.
//	@Tags			AI Jobs
//	@Produce		json
//	@Param			id			path	int		true	"Coin ID"
//	@Param			activeOnly	query	bool	false	"Only queued or running jobs"
//	@Success		200			{array}	models.AIJob
//	@Failure		400			{object}	ErrorResponse
//	@Failure		401			{object}	ErrorResponse
//	@Failure		404			{object}	ErrorResponse
//	@Failure		500			{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/coins/{id}/ai-jobs [get]
func (h *AIJobHandler) ListCoinJobs(c *gin.Context) {
	coinID, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	activeOnly := c.Query("activeOnly") == "true"
	jobs, err := h.service.ListCoinJobs(c.GetUint("userId"), coinID, activeOnly)
	if err != nil {
		if repository.IsRecordNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Coin not found"})
			return
		}
		h.logger.Error("ai-jobs", "Failed to list jobs for coin %d: %v", coinID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list AI jobs"})
		return
	}
	c.JSON(http.StatusOK, jobs)
}

func parseUintParam(c *gin.Context, name string) (uint, bool) {
	id, err := strconv.ParseUint(c.Param(name), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid " + name})
		return 0, false
	}
	return uint(id), true
}

func (h *AIJobHandler) respondJobError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, services.ErrAIJobInvalidSide):
		c.JSON(http.StatusBadRequest, gin.H{"error": "side query param must be omitted, 'obverse', or 'reverse'"})
	case errors.Is(err, services.ErrAIJobNoImages):
		c.JSON(http.StatusBadRequest, gin.H{"error": "No matching image found"})
	case repository.IsRecordNotFound(err):
		c.JSON(http.StatusNotFound, gin.H{"error": "Coin not found"})
	default:
		h.logger.Error("ai-jobs", "Failed to enqueue AI job: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to enqueue AI job"})
	}
}
