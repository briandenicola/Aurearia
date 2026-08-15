package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/gin-gonic/gin"
)

// deepMaxConcurrentStreamsPerJob is the "max 3 concurrent streams per job
// per owner" cap from contracts/sse-events.md §4; the 4th concurrent
// connection attempt gets 429.
const deepMaxConcurrentStreamsPerJob = 3

// deepSSEPingInterval is the `: ping` keepalive comment cadence required by
// contracts/sse-events.md §1.
const deepSSEPingInterval = 15 * time.Second

// StreamEvents serves the replayable, resumable SSE event stream for a
// Deep Analysis job (T095/T096, contracts/sse-events.md). It always
// replays persisted events from storage first (ListEventsSince), then
// subscribes to the in-process broker for the live tail, so a reconnecting
// client receives every missed event exactly once, in order, whether the
// process that produced them is still the one serving the reconnect or
// not (FR-009, FR-015, FR-016).
//
//	@Summary		Stream a Deep Analysis job's events (SSE)
//	@Tags			Deep Identification
//	@Produce		text/event-stream
//	@Param			id		path	int		true	"Job ID"
//	@Param			since	query	int		false	"Resume after this sequence number"
//	@Success		200
//	@Failure		401	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Failure		410	{object}	ErrorResponse
//	@Failure		429	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/deep-identification/jobs/{id}/events [get]
func (h *DeepIdentificationHandler) StreamEvents(c *gin.Context) {
	userID := c.GetUint("userId")
	jobID, ok := parseID(c, "id")
	if !ok {
		return
	}

	job, err := h.service.GetJob(jobID, userID)
	if err != nil {
		if repository.IsRecordNotFound(err) {
			respondError(c, http.StatusNotFound, "Deep Analysis job not found", nil)
			return
		}
		respondError(c, http.StatusInternalServerError, "Failed to load Deep Analysis job", err)
		return
	}
	if !job.ExpiresAt.IsZero() && job.ExpiresAt.Before(time.Now()) {
		respondError(c, http.StatusGone, "Deep Analysis job result has expired", nil)
		return
	}

	broker := h.service.Broker()
	if broker.SubscriberCount(jobID) >= deepMaxConcurrentStreamsPerJob {
		respondError(c, http.StatusTooManyRequests, "Too many concurrent streams for this job", nil)
		return
	}

	since := parseDeepEventsSince(c)

	w := c.Writer
	flusher, canFlush := w.(http.Flusher)
	if !canFlush {
		respondError(c, http.StatusInternalServerError, "Streaming is not supported by this server", nil)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	ch, unsubscribe := broker.Subscribe(jobID)
	defer unsubscribe()

	ctx := c.Request.Context()
	lastSeq := since

	// writeFrame returns true once a terminal event has been written (the
	// caller must then emit `event: end` and close, per contract §2).
	writeFrame := func(ev models.DeepIdentificationEvent) bool {
		fmt.Fprintf(w, "id: %d\nevent: %s\ndata: %s\n\n", ev.Seq, ev.Type, deepSSEEnvelopeJSON(jobID, ev))
		flusher.Flush()
		lastSeq = ev.Seq
		return ev.Type == models.DeepEventTerminal
	}

	writeEnd := func(status models.DeepJobStatus) {
		fmt.Fprintf(w, "event: end\ndata: {\"jobId\":%d,\"status\":%q}\n\n", jobID, status)
		flusher.Flush()
	}

	events, err := h.service.ListEventsSince(jobID, userID, since)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to load Deep Analysis events", err)
		return
	}

	// stream_truncated: the caller asked to resume after `since`, but the
	// surviving retained events do not pick up immediately after it (a
	// retention sweep already pruned the gap). This is a control frame:
	// it consumes no sequence number (`id:` omitted, contract §1/§2).
	if job.EventsPrunedAt != nil && since > 0 && (len(events) == 0 || events[0].Seq != since+1) {
		earliest := job.LastSeq
		if len(events) > 0 {
			earliest = events[0].Seq
		}
		fmt.Fprintf(w, "event: stream_truncated\ndata: {\"status\":%q,\"earliestSeq\":%d,\"lastSeq\":%d}\n\n", job.Status, earliest, job.LastSeq)
		flusher.Flush()
	}

	terminalSent := false
	for _, ev := range events {
		if writeFrame(ev) {
			terminalSent = true
		}
	}

	// If the job is already terminal but its terminal event row was itself
	// pruned (data-model.md §3: PruneEventsBefore deletes ALL events for a
	// pruned job), synthesize the terminal frame from the durable job row
	// so a client is never left waiting forever (contract §3 "Events
	// pruned, job terminal" row) - the full report/proposal remain
	// reachable via GET even though the event history is gone.
	if !terminalSent && models.IsDeepJobTerminal(job.Status) {
		writeFrame(deepSyntheticTerminalEvent(job))
		terminalSent = true
	}

	if terminalSent {
		writeEnd(job.Status)
		return
	}

	pingTicker := time.NewTicker(deepSSEPingInterval)
	defer pingTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-pingTicker.C:
			fmt.Fprint(w, ": ping\n\n")
			flusher.Flush()
		case <-ch:
			more, err := h.service.ListEventsSince(jobID, userID, lastSeq)
			if err != nil {
				return
			}
			for _, ev := range more {
				if writeFrame(ev) {
					fresh, getErr := h.service.GetJob(jobID, userID)
					if getErr == nil {
						writeEnd(fresh.Status)
					} else {
						writeEnd(job.Status)
					}
					return
				}
			}
		}
	}
}

// parseDeepEventsSince resolves the resume point per contract §3: an
// explicit `?since=` query parameter wins over the `Last-Event-ID` header
// (native EventSource cannot set custom headers on reconnect, so `since` is
// the primary mechanism for the fetch-based reader in
// useDeepIdentificationStream.ts; Last-Event-ID is honored too for any
// standards-based client).
func parseDeepEventsSince(c *gin.Context) int64 {
	if raw := c.Query("since"); raw != "" {
		if n, err := strconv.ParseInt(raw, 10, 64); err == nil && n >= 0 {
			return n
		}
	}
	if raw := c.GetHeader("Last-Event-ID"); raw != "" {
		if n, err := strconv.ParseInt(raw, 10, 64); err == nil && n >= 0 {
			return n
		}
	}
	return 0
}

// deepSSEEnvelopeJSON builds the single-line JSON envelope described in
// contracts/sse-events.md §1. PayloadJSON is already-sanitized JSON text
// (or empty), so it is embedded verbatim rather than re-marshaled.
func deepSSEEnvelopeJSON(jobID uint, ev models.DeepIdentificationEvent) string {
	payload := ev.PayloadJSON
	if payload == "" {
		payload = "{}"
	}
	return fmt.Sprintf(`{"seq":%d,"jobId":%d,"type":%q,"ts":%q,"payload":%s}`, ev.Seq, jobID, ev.Type, ev.CreatedAt.UTC().Format(time.RFC3339), payload)
}

// deepSyntheticTerminalEvent reconstructs the terminal envelope's payload
// shape (contract §2: `{status, partialSuccess, failureCode?, hasReport,
// hasProposal}`) directly from the job row for a terminal job whose event
// history has already been pruned.
func deepSyntheticTerminalEvent(job *models.DeepIdentificationJob) models.DeepIdentificationEvent {
	payload := fmt.Sprintf(
		`{"status":%q,"partialSuccess":%t,"failureCode":%q,"hasReport":%t,"hasProposal":%t}`,
		job.Status, job.PartialSuccess, job.FailureCode, job.ReportJSON != "", job.ProposalJSON != "",
	)
	completedAt := job.CompletedAt
	ts := job.UpdatedAt
	if completedAt != nil {
		ts = *completedAt
	}
	return models.DeepIdentificationEvent{
		JobID:       job.ID,
		UserID:      job.UserID,
		Seq:         job.LastSeq,
		Type:        models.DeepEventTerminal,
		PayloadJSON: payload,
		CreatedAt:   ts,
	}
}
