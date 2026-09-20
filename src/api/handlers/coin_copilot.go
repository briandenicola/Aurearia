package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
)

type CoinCopilotHandler struct {
	service *services.CoinCopilotService
	logger  *services.Logger
}

func NewCoinCopilotHandler(service *services.CoinCopilotService, logger *services.Logger) *CoinCopilotHandler {
	return &CoinCopilotHandler{service: service, logger: logger}
}

type copilotStartRequest struct {
	ThreadID   string         `json:"threadId"`
	Goal       string         `json:"goal"`
	AppContext map[string]any `json:"appContext"`
}

type copilotResumeRequest struct {
	Answer                    string `json:"answer"`
	ExpectedCheckpointVersion int64  `json:"expectedCheckpointVersion"`
}

type copilotUsageDTO struct {
	Iterations   int   `json:"iterations"`
	ToolCalls    int   `json:"toolCalls"`
	InputTokens  int64 `json:"inputTokens"`
	OutputTokens int64 `json:"outputTokens"`
}

type copilotRunDTO struct {
	ID                string          `json:"id"`
	ThreadID          string          `json:"threadId"`
	Status            string          `json:"status"`
	Goal              string          `json:"goal"`
	CheckpointVersion int64           `json:"checkpointVersion"`
	LastSeq           int64           `json:"lastSeq"`
	Attempt           int             `json:"attempt"`
	FinalAnswer       *string         `json:"finalAnswer"`
	FailureCode       *string         `json:"failureCode"`
	FailureMessage    *string         `json:"failureMessage"`
	ResumeDeadline    *time.Time      `json:"resumeDeadline"`
	Usage             copilotUsageDTO `json:"usage"`
	CreatedAt         time.Time       `json:"createdAt"`
	UpdatedAt         time.Time       `json:"updatedAt"`
}

type copilotRunEnvelope struct {
	Run    copilotRunDTO `json:"run"`
	Reused bool          `json:"reused"`
}

type copilotThreadMessageDTO struct {
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	RunID     *string   `json:"runId"`
	CreatedAt time.Time `json:"createdAt"`
}

type copilotThreadDTO struct {
	ID        string                    `json:"id"`
	Title     string                    `json:"title"`
	Messages  []copilotThreadMessageDTO `json:"messages"`
	Runs      []copilotRunDTO           `json:"runs"`
	CreatedAt time.Time                 `json:"createdAt"`
	UpdatedAt time.Time                 `json:"updatedAt"`
}

func toCopilotRunDTO(run *models.CoinCopilotRun) copilotRunDTO {
	var answer, failureCode, failureMessage *string
	if run.FinalAnswer != "" {
		answer = &run.FinalAnswer
	}
	if run.FailureCode != "" {
		failureCode = &run.FailureCode
	}
	if run.FailureMessage != "" {
		failureMessage = &run.FailureMessage
	}
	return copilotRunDTO{
		ID: run.ID, ThreadID: run.ThreadID, Status: string(run.Status), Goal: run.Goal,
		CheckpointVersion: run.CheckpointVersion, LastSeq: run.LastSeq, Attempt: run.ExecutionAttempt,
		FinalAnswer: answer, FailureCode: failureCode, FailureMessage: failureMessage,
		ResumeDeadline: run.ResumeDeadline,
		Usage: copilotUsageDTO{
			Iterations: run.IterationCount, ToolCalls: run.ToolCallCount, InputTokens: run.InputTokens,
			OutputTokens: run.OutputTokens,
		},
		CreatedAt: run.CreatedAt, UpdatedAt: run.UpdatedAt,
	}
}

func decodeCopilotJSON(c *gin.Context, target any) error {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 128*1024)
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("request body must contain exactly one JSON value")
	}
	return nil
}

// Capability godoc
//
//	@Summary		Resolve Coin Copilot or legacy mode
//	@Tags			Coin Copilot
//	@Produce		json
//	@Success		200	{object}	services.CoinCopilotCapability
//	@Security		BearerAuth
//	@Router			/agent/copilot/capability [get]
func (h *CoinCopilotHandler) Capability(c *gin.Context) {
	c.JSON(http.StatusOK, h.service.Capability())
}

// Start godoc
//
//	@Summary		Start a durable Coin Copilot run
//	@Tags			Coin Copilot
//	@Accept			json
//	@Produce		json
//	@Param			Idempotency-Key	header		string					true	"Idempotency key"
//	@Param			body			body		copilotStartRequest		true	"Run request"
//	@Success		202				{object}	copilotRunEnvelope
//	@Success		200				{object}	copilotRunEnvelope
//	@Failure		400				{object}	ErrorResponse
//	@Failure		409				{object}	ErrorResponse
//	@Failure		503				{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/agent/copilot/runs [post]
func (h *CoinCopilotHandler) Start(c *gin.Context) {
	var request copilotStartRequest
	if err := decodeCopilotJSON(c, &request); err != nil {
		respondError(c, http.StatusBadRequest, "Invalid request", nil)
		return
	}
	run, reused, err := h.service.Start(c.GetUint("userId"), services.CoinCopilotStartInput{
		ThreadID: request.ThreadID, Goal: request.Goal, AppContext: request.AppContext,
		IdempotencyKey: c.GetHeader("Idempotency-Key"),
	})
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	status := http.StatusAccepted
	if reused {
		status = http.StatusOK
	}
	c.JSON(status, copilotRunEnvelope{Run: toCopilotRunDTO(run), Reused: reused})
}

// GetRun godoc
//
//	@Summary		Read an owner-scoped Coin Copilot run
//	@Tags			Coin Copilot
//	@Produce		json
//	@Param			runId	path		string	true	"Run ID"
//	@Success		200		{object}	copilotRunEnvelope
//	@Failure		404		{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/agent/copilot/runs/{runId} [get]
func (h *CoinCopilotHandler) GetRun(c *gin.Context) {
	run, err := h.service.GetRun(c.GetUint("userId"), c.Param("runId"))
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, copilotRunEnvelope{Run: toCopilotRunDTO(run)})
}

// GetThread godoc
//
//	@Summary		Read an owner-scoped Coin Copilot thread
//	@Tags			Coin Copilot
//	@Produce		json
//	@Param			threadId	path		string	true	"Thread ID"
//	@Success		200			{object}	copilotThreadDTO
//	@Failure		404			{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/agent/copilot/threads/{threadId} [get]
func (h *CoinCopilotHandler) GetThread(c *gin.Context) {
	thread, runs, err := h.service.GetThread(c.GetUint("userId"), c.Param("threadId"))
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	dto := copilotThreadDTO{ID: thread.ID, Title: thread.Title, CreatedAt: thread.CreatedAt, UpdatedAt: thread.UpdatedAt}
	for i := range runs {
		run := &runs[i]
		dto.Runs = append(dto.Runs, toCopilotRunDTO(run))
		runID := run.ID
		dto.Messages = append(dto.Messages, copilotThreadMessageDTO{Role: "user", Content: run.Goal, RunID: &runID, CreatedAt: run.CreatedAt})
		if run.FinalAnswer != "" {
			createdAt := run.UpdatedAt
			if run.CompletedAt != nil {
				createdAt = *run.CompletedAt
			}
			dto.Messages = append(dto.Messages, copilotThreadMessageDTO{Role: "assistant", Content: run.FinalAnswer, RunID: &runID, CreatedAt: createdAt})
		}
	}
	c.JSON(http.StatusOK, gin.H{"thread": dto})
}

// DeleteThread godoc
//
//	@Summary		Delete a settled Coin Copilot thread
//	@Tags			Coin Copilot
//	@Param			threadId	path	string	true	"Thread ID"
//	@Success		204
//	@Failure		404	{object}	ErrorResponse
//	@Failure		409	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/agent/copilot/threads/{threadId} [delete]
func (h *CoinCopilotHandler) DeleteThread(c *gin.Context) {
	if err := h.service.DeleteThread(c.GetUint("userId"), c.Param("threadId")); err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// Cancel godoc
//
//	@Summary		Cancel a Coin Copilot run
//	@Tags			Coin Copilot
//	@Produce		json
//	@Param			runId	path		string	true	"Run ID"
//	@Success		202		{object}	copilotRunEnvelope
//	@Success		200		{object}	copilotRunEnvelope
//	@Failure		404		{object}	ErrorResponse
//	@Failure		409		{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/agent/copilot/runs/{runId}/cancel [post]
func (h *CoinCopilotHandler) Cancel(c *gin.Context) {
	run, immediate, err := h.service.Cancel(c.GetUint("userId"), c.Param("runId"))
	if err != nil && run == nil {
		h.respondServiceError(c, err)
		return
	}
	if err != nil {
		c.JSON(http.StatusConflict, copilotRunEnvelope{Run: toCopilotRunDTO(run)})
		return
	}
	status := http.StatusAccepted
	if immediate && run.Status == models.CopilotRunCancelled {
		status = http.StatusOK
	}
	c.JSON(status, copilotRunEnvelope{Run: toCopilotRunDTO(run)})
}

// Resume godoc
//
//	@Summary		Resume a paused Coin Copilot run
//	@Tags			Coin Copilot
//	@Accept			json
//	@Produce		json
//	@Param			runId			path		string					true	"Run ID"
//	@Param			Idempotency-Key	header		string					true	"Idempotency key"
//	@Param			body			body		copilotResumeRequest	true	"Resume request"
//	@Success		202				{object}	copilotRunEnvelope
//	@Success		200				{object}	copilotRunEnvelope
//	@Failure		400				{object}	ErrorResponse
//	@Failure		404				{object}	ErrorResponse
//	@Failure		409				{object}	ErrorResponse
//	@Failure		503				{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/agent/copilot/runs/{runId}/resume [post]
func (h *CoinCopilotHandler) Resume(c *gin.Context) {
	var request copilotResumeRequest
	if err := decodeCopilotJSON(c, &request); err != nil {
		respondError(c, http.StatusBadRequest, "Invalid request", nil)
		return
	}
	run, reused, err := h.service.Resume(c.GetUint("userId"), c.Param("runId"), services.CoinCopilotResumeInput{
		Answer: request.Answer, ExpectedCheckpointVersion: request.ExpectedCheckpointVersion,
		IdempotencyKey: c.GetHeader("Idempotency-Key"),
	})
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	status := http.StatusAccepted
	if reused {
		status = http.StatusOK
	}
	c.JSON(status, copilotRunEnvelope{Run: toCopilotRunDTO(run), Reused: reused})
}

func (h *CoinCopilotHandler) respondServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, services.ErrCopilotNotFound):
		respondError(c, http.StatusNotFound, "Coin Copilot resource not found", nil)
	case errors.Is(err, services.ErrCopilotInvalidRequest):
		respondError(c, http.StatusBadRequest, "Invalid request", nil)
	case errors.Is(err, services.ErrCopilotIdempotencyConflict):
		respondError(c, http.StatusConflict, "Idempotency key conflict", nil)
	case errors.Is(err, services.ErrCopilotCapacity), errors.Is(err, services.ErrCopilotQueueFull),
		errors.Is(err, services.ErrCopilotStateConflict), errors.Is(err, services.ErrCopilotThreadHasActiveRun):
		respondError(c, http.StatusConflict, "Coin Copilot request conflicts with current state", nil)
	case errors.Is(err, services.ErrCopilotDisabled), errors.Is(err, services.ErrCopilotUnavailable):
		respondError(c, http.StatusServiceUnavailable, "Coin Copilot is unavailable; use legacy chat", nil)
	default:
		if h.logger != nil {
			h.logger.Error("coin-copilot", "request failed: %v", err)
		}
		respondError(c, http.StatusInternalServerError, "Coin Copilot request failed", nil)
	}
}
