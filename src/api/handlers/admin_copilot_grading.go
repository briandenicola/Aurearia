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

// AdminCopilotGradingHandler serves graded Coin Copilot conversations for admin review.
type AdminCopilotGradingHandler struct {
	repo      *repository.CoinCopilotGradeRepository
	scheduler *services.CopilotGradingScheduler
}

// NewAdminCopilotGradingHandler creates a new AdminCopilotGradingHandler.
func NewAdminCopilotGradingHandler(repo *repository.CoinCopilotGradeRepository, scheduler *services.CopilotGradingScheduler) *AdminCopilotGradingHandler {
	return &AdminCopilotGradingHandler{repo: repo, scheduler: scheduler}
}

// CopilotGradeListResponse is one page of graded Copilot conversations.
type CopilotGradeListResponse struct {
	Grades []repository.CopilotGradeRow `json:"grades"`
	Total  int64                        `json:"total"`
	Page   int                          `json:"page"`
	Limit  int                          `json:"limit"`
}

// CopilotGradeDisputeRequest marks or clears disagreement with a grade.
type CopilotGradeDisputeRequest struct {
	Disputed bool   `json:"disputed"`
	Note     string `json:"note" binding:"max=500"`
}

// ListGrades returns graded Copilot conversations, worst first by default.
//
//	@Summary		List graded Copilot conversations
//	@Description	Returns rubric grades for completed Coin Copilot runs with their question and answer. Admin only.
//	@Tags			Admin
//	@Produce		json
//	@Param			page		query		int		false	"Page number"	default(1)
//	@Param			limit		query		int		false	"Items per page"	default(20)
//	@Param			sort		query		string	false	"Sort order"	Enums(worst, recent)	default(worst)
//	@Param			disputed	query		bool	false	"Only disputed (true) or undisputed (false) grades"
//	@Success		200			{object}	CopilotGradeListResponse
//	@Failure		401			{object}	ErrorResponse
//	@Failure		403			{object}	ErrorResponse
//	@Failure		500			{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/admin/copilot-grades [get]
func (h *AdminCopilotGradingHandler) ListGrades(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	filter := repository.CopilotGradeListFilter{Page: page, Limit: limit, Sort: c.DefaultQuery("sort", "worst")}
	if value, ok := c.GetQuery("disputed"); ok {
		disputed, err := strconv.ParseBool(value)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "disputed must be true or false"})
			return
		}
		filter.Disputed = &disputed
	}
	grades, total, err := h.repo.List(filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list Copilot grades"})
		return
	}
	c.JSON(http.StatusOK, CopilotGradeListResponse{Grades: grades, Total: total, Page: max(page, 1), Limit: min(max(limit, 1), 100)})
}

// GetGrade returns one graded Copilot conversation.
//
//	@Summary		Get a graded Copilot conversation
//	@Description	Returns one rubric grade with its question and answer. Admin only.
//	@Tags			Admin
//	@Produce		json
//	@Param			id	path		int	true	"Grade ID"
//	@Success		200	{object}	repository.CopilotGradeRow
//	@Failure		400	{object}	ErrorResponse
//	@Failure		401	{object}	ErrorResponse
//	@Failure		403	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/admin/copilot-grades/{id} [get]
func (h *AdminCopilotGradingHandler) GetGrade(c *gin.Context) {
	id, ok := parseGradeID(c)
	if !ok {
		return
	}
	grade, err := h.repo.Get(id)
	if errors.Is(err, repository.ErrCopilotGradeNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Grade not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load Copilot grade"})
		return
	}
	c.JSON(http.StatusOK, grade)
}

// Summary returns the quality trend and per-criterion failure rates.
//
//	@Summary		Summarize Copilot grades
//	@Description	Returns average quality, a daily trend and per-criterion failure rates over undisputed grades. Admin only.
//	@Tags			Admin
//	@Produce		json
//	@Param			days	query		int	false	"Window in days (1-365)"	default(30)
//	@Success		200		{object}	repository.CopilotGradeSummary
//	@Failure		401		{object}	ErrorResponse
//	@Failure		403		{object}	ErrorResponse
//	@Failure		500		{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/admin/copilot-grades/summary [get]
func (h *AdminCopilotGradingHandler) Summary(c *gin.Context) {
	days, err := strconv.Atoi(c.DefaultQuery("days", "30"))
	if err != nil || days < 1 || days > 365 {
		days = 30
	}
	summary, err := h.repo.Summary(days, time.Now().UTC())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to summarize Copilot grades"})
		return
	}
	c.JSON(http.StatusOK, summary)
}

// SetDispute marks a grade as wrong (or clears that mark) so it leaves the trend.
//
//	@Summary		Dispute a Copilot grade
//	@Description	Marks or clears an admin's disagreement with a grade. Disputed grades are excluded from the summary. Admin only.
//	@Tags			Admin
//	@Accept			json
//	@Produce		json
//	@Param			id		path		int							true	"Grade ID"
//	@Param			request	body		CopilotGradeDisputeRequest	true	"Dispute"
//	@Success		200		{object}	repository.CopilotGradeRow
//	@Failure		400		{object}	ErrorResponse
//	@Failure		401		{object}	ErrorResponse
//	@Failure		403		{object}	ErrorResponse
//	@Failure		404		{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/admin/copilot-grades/{id}/dispute [put]
func (h *AdminCopilotGradingHandler) SetDispute(c *gin.Context) {
	id, ok := parseGradeID(c)
	if !ok {
		return
	}
	var request CopilotGradeDisputeRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		respondError(c, http.StatusBadRequest, "Invalid request payload", err)
		return
	}
	err := h.repo.SetDispute(id, request.Disputed, services.SanitizeCopilotText(request.Note, 500), time.Now().UTC())
	if errors.Is(err, repository.ErrCopilotGradeNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Grade not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update Copilot grade"})
		return
	}
	h.GetGrade(c)
}

// GetStatus returns the grading scheduler status and its last cycle.
//
//	@Summary		Get Copilot grading status
//	@Description	Returns whether nightly Copilot grading is enabled or running, when it runs next, and the last cycle's result. Admin only.
//	@Tags			Admin
//	@Produce		json
//	@Success		200	{object}	services.CopilotGradingStatus
//	@Failure		401	{object}	ErrorResponse
//	@Failure		403	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/admin/copilot-grading/status [get]
func (h *AdminCopilotGradingHandler) GetStatus(c *gin.Context) {
	c.JSON(http.StatusOK, h.scheduler.GetGradingStatus())
}

// RunNow starts grading pending runs in the background.
//
//	@Summary		Run Copilot grading now
//	@Description	Starts grading completed, ungraded Copilot runs in the background, up to the daily cap. Admin only.
//	@Tags			Admin
//	@Produce		json
//	@Success		202	{object}	map[string]string
//	@Failure		401	{object}	ErrorResponse
//	@Failure		403	{object}	ErrorResponse
//	@Failure		409	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/admin/copilot-grading/run [post]
func (h *AdminCopilotGradingHandler) RunNow(c *gin.Context) {
	if err := h.scheduler.StartManualRun(); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Copilot grading is already running"})
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"message": "Copilot grading started"})
}

func parseGradeID(c *gin.Context) (uint, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid grade ID"})
		return 0, false
	}
	return uint(id), true
}
