package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/gin-gonic/gin"
)

var copilotSSEPingInterval = 15 * time.Second

// StreamEvents godoc
//
//	@Summary		Replay and follow Coin Copilot events
//	@Tags			Coin Copilot
//	@Produce		text/event-stream
//	@Param			runId			path	string	true	"Run ID"
//	@Param			since			query	int		false	"Resume after sequence"
//	@Param			Last-Event-ID	header	int		false	"Resume after sequence"
//	@Success		200
//	@Failure		404	{object}	ErrorResponse
//	@Failure		429	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/agent/copilot/runs/{runId}/events [get]
func (h *CoinCopilotHandler) StreamEvents(c *gin.Context) {
	userID := c.GetUint("userId")
	runID := c.Param("runId")
	run, err := h.service.GetRun(userID, runID)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	ch, unsubscribe, ok := h.service.Broker().Subscribe(runID)
	if !ok {
		respondError(c, http.StatusTooManyRequests, "Too many concurrent streams for this run", nil)
		return
	}
	defer unsubscribe()
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		respondError(c, http.StatusInternalServerError, "Streaming is not supported", nil)
		return
	}
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	c.Status(http.StatusOK)
	flusher.Flush()

	since := parseCopilotSince(c)
	lastSeq := since
	writeEvent := func(event models.CoinCopilotEvent) {
		payload := json.RawMessage(event.PayloadJSON)
		envelope, _ := json.Marshal(map[string]any{
			"seq": event.Seq, "threadId": event.ThreadID, "runId": event.RunID,
			"executionId": event.ExecutionID, "type": event.Type,
			"ts": event.CreatedAt.UTC().Format(time.RFC3339), "payload": payload,
		})
		fmt.Fprintf(c.Writer, "id: %d\nevent: %s\ndata: %s\n\n", event.Seq, event.Type, envelope)
		flusher.Flush()
		lastSeq = event.Seq
	}
	writeEnd := func(current *models.CoinCopilotRun) {
		fmt.Fprintf(c.Writer, "event: end\ndata: {\"runId\":%q,\"status\":%q}\n\n", current.ID, current.Status)
		flusher.Flush()
	}
	replay, err := h.service.ListEventsSince(userID, runID, since)
	if err != nil {
		return
	}
	if run.EventsPrunedAt != nil && since > 0 && (len(replay) == 0 || replay[0].Seq != since+1) {
		earliest := run.LastSeq
		if len(replay) > 0 {
			earliest = replay[0].Seq
		}
		fmt.Fprintf(c.Writer, "event: stream_truncated\ndata: {\"runId\":%q,\"status\":%q,\"earliestSeq\":%d,\"lastSeq\":%d}\n\n", run.ID, run.Status, earliest, run.LastSeq)
		flusher.Flush()
	}
	for _, event := range replay {
		writeEvent(event)
	}
	fresh, err := h.service.GetRun(userID, runID)
	if err != nil {
		return
	}
	if models.IsCopilotRunTerminal(fresh.Status) {
		writeEnd(fresh)
		return
	}
	ticker := time.NewTicker(copilotSSEPingInterval)
	defer ticker.Stop()
	for {
		select {
		case <-c.Request.Context().Done():
			return
		case <-ticker.C:
			fmt.Fprint(c.Writer, ": ping\n\n")
			flusher.Flush()
		case <-ch:
			events, err := h.service.ListEventsSince(userID, runID, lastSeq)
			if err != nil {
				return
			}
			for _, event := range events {
				writeEvent(event)
			}
			fresh, err := h.service.GetRun(userID, runID)
			if err != nil {
				return
			}
			if models.IsCopilotRunTerminal(fresh.Status) {
				writeEnd(fresh)
				return
			}
		}
	}
}

func parseCopilotSince(c *gin.Context) int64 {
	if raw, exists := c.GetQuery("since"); exists {
		if value, err := strconv.ParseInt(raw, 10, 64); err == nil && value >= 0 {
			return value
		}
		return 0
	}
	if value, err := strconv.ParseInt(c.GetHeader("Last-Event-ID"), 10, 64); err == nil && value >= 0 {
		return value
	}
	return 0
}
