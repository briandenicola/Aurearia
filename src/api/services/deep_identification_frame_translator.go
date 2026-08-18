package services

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
)

// deepFrameTranslator holds the per-Run mutable state used to translate
// each SSE frame from the Python pipeline into the payload persisted as a
// DeepIdentificationEvent and, for provider_result/synthesis/error frames,
// into the accumulated evidence (providerClaims/lastSynthesis/
// lastErrorCode/lastErrorMessage) used to build the terminal
// DeepPipelineResult (T104: extracted from Run's inline 8-case closure so
// each frame-type translation is independently unit-testable).
//
// A translator is scoped to exactly one DeepIdentificationPipelineRunner.Run
// call and is driven synchronously, in arrival order, by
// AgentProxy.StreamDeepIdentification's single onFrame callback (see
// agent_proxy_deep_identify.go) - there is never concurrent access to its
// fields, so no locking is introduced here.
type deepFrameTranslator struct {
	r   *DeepIdentificationPipelineRunner
	job *models.DeepIdentificationJob

	lastSynthesis    json.RawMessage
	lastErrorCode    string
	lastErrorMessage string

	// providerClaims accumulates each provider_result frame's validated
	// claims keyed by provider, indexed positionally so the terminal
	// synthesis' proposed_fields.evidence_refs (contract §5) can be
	// resolved into full citation-bearing evidence when the rich proposal
	// document is built (B1: no citations/confidence are dropped). The full
	// per-provider claim list (in the emitted order) is retained verbatim so
	// each synthesis `claim_index` still resolves to the exact claim the
	// synthesizer saw; per-claim citation-host re-validation happens later in
	// buildDeepProposalDocumentJSON, not by reindexing here.
	providerClaims    map[string][]deepProposalClaim
	providerStartedAt map[models.DeepProviderName]time.Time
}

// newDeepFrameTranslator constructs a translator for one Run invocation.
func newDeepFrameTranslator(r *DeepIdentificationPipelineRunner, job *models.DeepIdentificationJob) *deepFrameTranslator {
	return &deepFrameTranslator{
		r:                 r,
		job:               job,
		providerClaims:    map[string][]deepProposalClaim{},
		providerStartedAt: map[models.DeepProviderName]time.Time{},
	}
}

// translate dispatches one frame to its per-type handler and returns the
// JSON persisted into the user-visible event log. It defaults to the raw
// internal frame, but privacy/event-bloat sensitive frames (provider_result)
// are reduced to the bounded public payload defined in
// contracts/sse-events.md §2 before persistence, so full provider
// claims/citations never enter the owner-facing, replayable event stream
// (FR-036) even though the runner itself consumes them in-memory to build
// the proposal.
func (t *deepFrameTranslator) translate(frame DeepIdentifyFrame) string {
	persistPayload := string(frame.Raw)
	switch frame.Type {
	case "provider_started":
		t.handleProviderStarted(frame)
	case "router_selected":
		persistPayload = t.handleRouterSelected(frame, persistPayload)
	case "synthesis":
		t.handleSynthesis(frame)
	case "provider_result":
		persistPayload = t.handleProviderResult(frame, persistPayload)
	case "evaluation":
		persistPayload = t.handleEvaluation(frame, persistPayload)
	case "progress":
		persistPayload = t.handleProgress(frame, persistPayload)
	case "error":
		t.handleError(frame)
	}
	return persistPayload
}

// handleProviderStarted records a provider_started frame's start time (for
// later latency computation in handleProviderResult) and persists the
// provider-started row. It never overrides persistPayload - the raw frame
// is what gets persisted for this event type.
func (t *deepFrameTranslator) handleProviderStarted(frame DeepIdentifyFrame) {
	var payload struct {
		Provider    string `json:"provider"`
		Automatable *bool  `json:"automatable"`
	}
	if jsonErr := json.Unmarshal(frame.Raw, &payload); jsonErr == nil {
		if provider, ok := deepProviderName(payload.Provider); ok {
			startedAt := time.Now()
			t.providerStartedAt[provider] = startedAt
			automatable := true
			if payload.Automatable != nil {
				automatable = *payload.Automatable
			}
			if recordErr := t.r.repo.RecordProviderStarted(t.job.ID, t.job.UserID, provider, automatable, startedAt); recordErr != nil && t.r.logger != nil {
				t.r.logger.Error("deep-identification", "failed to record provider start job=%d provider=%s: %v", t.job.ID, provider, recordErr)
			}
		}
	}
}

// handleRouterSelected reduces a router_selected frame to its public
// payload and persists the router's selection decision. Returns
// defaultPayload unchanged if the frame fails to parse.
func (t *deepFrameTranslator) handleRouterSelected(frame DeepIdentifyFrame, defaultPayload string) string {
	var payload struct {
		Selected  []string `json:"selected"`
		Rationale string   `json:"rationale"`
		Skipped   []struct {
			Provider string `json:"provider"`
			Reason   string `json:"reason"`
		} `json:"skipped"`
	}
	if jsonErr := json.Unmarshal(frame.Raw, &payload); jsonErr == nil {
		persistPayload := deepRouterSelectedPublicPayloadJSON(payload.Selected, payload.Rationale, payload.Skipped)
		if updateErr := t.r.repo.RecordRouterSelection(t.job.ID, t.job.UserID, payload.Selected, payload.Rationale); updateErr != nil && t.r.logger != nil {
			t.r.logger.Error("deep-identification", "failed to persist router selection for job %d: %v", t.job.ID, updateErr)
		}
		return persistPayload
	}
	return defaultPayload
}

// handleSynthesis captures the terminal synthesis report. synthesis is not
// a persisted event type (deepPipelineEventType returns false for it) - it
// is surfaced only through the returned DeepPipelineResult, so this never
// touches persistPayload.
func (t *deepFrameTranslator) handleSynthesis(frame DeepIdentifyFrame) {
	var payload struct {
		Report json.RawMessage `json:"report"`
	}
	if jsonErr := json.Unmarshal(frame.Raw, &payload); jsonErr == nil && len(payload.Report) > 0 {
		t.lastSynthesis = payload.Report
	} else {
		// Some producers may emit the DeepSynthesis fields inline (no
		// wrapping "report" key) - fall back to treating the whole payload
		// minus "type" as the report.
		t.lastSynthesis = frame.Raw
	}
}

// handleProviderResult consumes a provider_result frame's full internal
// ProviderEvidence (contract §3/§4) for the in-memory proposal and provider
// run bookkeeping, but returns only the bounded public payload to persist.
// Returns defaultPayload unchanged if the frame fails to parse or has no
// provider name.
func (t *deepFrameTranslator) handleProviderResult(frame DeepIdentifyFrame, defaultPayload string) string {
	var evidence struct {
		Provider    string              `json:"provider"`
		Status      string              `json:"status"`
		Automatable *bool               `json:"automatable"`
		Confidence  float64             `json:"confidence"`
		CallCount   int                 `json:"call_count"`
		ErrorKind   string              `json:"error_kind"`
		LinkOut     string              `json:"link_out"`
		Claims      []deepProposalClaim `json:"claims"`
	}
	jsonErr := json.Unmarshal(frame.Raw, &evidence)
	if jsonErr != nil || evidence.Provider == "" {
		return defaultPayload
	}

	t.providerClaims[evidence.Provider] = evidence.Claims
	persistPayload := deepProviderResultPublicPayloadJSON(
		evidence.Provider, evidence.Status, evidence.Confidence,
		evidence.ErrorKind, evidence.LinkOut, evidence.Claims,
	)
	if provider, ok := deepProviderName(evidence.Provider); ok {
		completedAt := time.Now()
		startedAt, found := t.providerStartedAt[provider]
		if !found {
			startedAt = completedAt
		}
		latencyMS := int(completedAt.Sub(startedAt).Milliseconds())
		if found && latencyMS < 1 {
			latencyMS = 1
		}
		status := deepProviderRunStatus(evidence.Status)
		errorKind := deepProviderErrorKind(evidence.ErrorKind)
		automatable := status != models.DeepProviderRunNotAutomated
		if evidence.Automatable != nil {
			automatable = *evidence.Automatable
		}
		if recordErr := t.r.repo.RecordProviderResult(
			t.job.ID, t.job.UserID, provider, status, automatable,
			evidence.Confidence, evidence.CallCount, latencyMS, errorKind,
			startedAt, completedAt,
		); recordErr != nil {
			if t.r.logger != nil {
				t.r.logger.Error("deep-identification", "failed to record provider result job=%d provider=%s: %v", t.job.ID, provider, recordErr)
			}
		} else if t.r.logger != nil {
			t.r.logger.Info("deep-identification", "provider settled job=%d provider=%s status=%s latency_ms=%d call_count=%d", t.job.ID, provider, status, latencyMS, evidence.CallCount)
		}
	}
	return persistPayload
}

// handleEvaluation reduces an evaluation frame to its public payload.
// Returns defaultPayload unchanged if the frame fails to parse.
func (t *deepFrameTranslator) handleEvaluation(frame DeepIdentifyFrame, defaultPayload string) string {
	var payload struct {
		DisagreementCount int `json:"disagreement_count"`
		ResolvedCount     int `json:"resolved_count"`
	}
	if jsonErr := json.Unmarshal(frame.Raw, &payload); jsonErr == nil {
		return fmt.Sprintf(
			`{"disagreementCount":%d,"resolvedCount":%d}`,
			payload.DisagreementCount,
			payload.ResolvedCount,
		)
	}
	return defaultPayload
}

// handleProgress re-encodes a progress frame to exactly {phase, message} -
// a deliberate strict whitelist (contracts/sse-events.md §2). This is
// intentionally asymmetric with provider_started (persisted verbatim) and
// provider_result (reduced to a bounded six-field payload); do not "unify"
// the three. Returns defaultPayload unchanged if the frame fails to parse.
func (t *deepFrameTranslator) handleProgress(frame DeepIdentifyFrame, defaultPayload string) string {
	var payload struct {
		Stage   string `json:"stage"`
		Phase   string `json:"phase"`
		Message string `json:"message"`
	}
	if jsonErr := json.Unmarshal(frame.Raw, &payload); jsonErr == nil {
		phase := payload.Phase
		if phase == "" {
			phase = payload.Stage
		}
		message := payload.Message
		if message == "" {
			message = deepProgressMessage(phase)
		}
		encoded, _ := json.Marshal(map[string]string{"phase": phase, "message": message})
		return string(encoded)
	}
	return defaultPayload
}

// handleError captures the terminal error frame's code/message. error is
// not a persisted event type (deepPipelineEventType returns false for it) -
// it is surfaced only through Run's returned error, so this never touches
// persistPayload.
func (t *deepFrameTranslator) handleError(frame DeepIdentifyFrame) {
	var payload struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	if jsonErr := json.Unmarshal(frame.Raw, &payload); jsonErr == nil {
		t.lastErrorCode = payload.Code
		t.lastErrorMessage = payload.Message
	}
}
