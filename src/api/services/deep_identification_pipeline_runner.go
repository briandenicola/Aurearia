package services

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
)

// DeepIdentificationPipelineRunner is the Phase 7 implementation of
// DeepPipelineRunner: it loads a job's artifacts, resolves LLM config and
// per-run bounds, mints a job-scoped internal token, and streams the run
// through AgentProxy.StreamDeepIdentification, persisting each frame as a
// DeepIdentificationEvent (contracts/agent-internal-contract.md, T070-T072).
//
// It never appends its own terminal event: repository.SettleTerminal
// (called by deep_identification_service.go's runJob after Run returns)
// already appends exactly one terminal event transactionally.
type DeepIdentificationPipelineRunner struct {
	proxy        *AgentProxy
	repo         *repository.DeepIdentificationRepository
	settingsSvc  *SettingsService
	tokenSvc     *InternalTokenService
	toolsBaseURL string
	logger       *Logger
	broker       *DeepIdentificationBroker
	coinLookup   *CoinLookupService
}

func (r *DeepIdentificationPipelineRunner) WithQuickEvidence(coinLookup *CoinLookupService) *DeepIdentificationPipelineRunner {
	r.coinLookup = coinLookup
	return r
}

// NewDeepIdentificationPipelineRunner constructs the real pipeline runner,
// wired in via DeepIdentificationService.SetPipelineRunner (main.go).
// broker may be nil in older call sites/tests that do not exercise live
// SSE fan-out; a nil broker simply means no wake is published (persisted
// events and later replay are unaffected).
func NewDeepIdentificationPipelineRunner(
	proxy *AgentProxy,
	repo *repository.DeepIdentificationRepository,
	settingsSvc *SettingsService,
	tokenSvc *InternalTokenService,
	toolsBaseURL string,
	logger *Logger,
	broker *DeepIdentificationBroker,
) *DeepIdentificationPipelineRunner {
	return &DeepIdentificationPipelineRunner{
		proxy:        proxy,
		repo:         repo,
		settingsSvc:  settingsSvc,
		tokenSvc:     tokenSvc,
		toolsBaseURL: toolsBaseURL,
		logger:       logger,
		broker:       broker,
	}
}

// deepPipelineDefaultMaxConcurrency/ProviderTimeout/RecursionLimit are fixed,
// documented defaults (see .squad/decisions/inbox for rationale): the live
// DeepIdentificationSettings snapshot (settings_service.go) exposes
// MaxProviders and HardTimeout but not yet a separate concurrency/per-
// provider-timeout/recursion-limit knob, so these are derived in code
// rather than adding new settings keys this phase.
const (
	deepPipelineDefaultMaxConcurrency    = 2
	deepPipelineDefaultProviderTimeoutS  = 45
	deepPipelineDefaultRecursionLimit    = 12
	deepPipelineHardTimeoutSafetyMarginS = 20
	deepPipelineMinTotalTimeoutS         = 30
	deepPipelineMaxTotalTimeoutS         = 900
	// deepPipelineTokenTTLMarginS pads the job-scoped internal token's TTL
	// beyond total_timeout_s so a slow-to-return final HTTP round trip near
	// the deadline is never rejected as expired mid-flight.
	deepPipelineTokenTTLMarginS = 30
)

// Run implements DeepPipelineRunner. ctx is deep_identification_service.go's
// timeoutCtx (bounded by DeepIdentificationSettings.HardTimeout and
// cancellable on explicit user cancel); it is passed straight through to
// StreamDeepIdentification so HTTP-level cancellation is automatic (T071).
func (r *DeepIdentificationPipelineRunner) Run(ctx context.Context, job *models.DeepIdentificationJob) (*DeepPipelineResult, error) {
	unsettledStatus := models.DeepProviderRunFailed
	unsettledErrorKind := "upstream"
	defer func() {
		if err := r.repo.SettleRunningProviderRuns(job.ID, job.UserID, unsettledStatus, unsettledErrorKind, time.Now()); err != nil && r.logger != nil {
			r.logger.Error("deep-identification", "failed to settle unfinished provider runs for job %d: %v", job.ID, err)
		}
	}()

	images, err := r.loadImages(job.ID)
	if err != nil {
		return nil, fmt.Errorf("load artifacts for job %d: %w", job.ID, err)
	}

	llmCfg, err := r.settingsSvc.ResolveLLMConfig()
	if err != nil {
		return nil, fmt.Errorf("resolve llm config: %w", err)
	}

	settings := r.settingsSvc.GetDeepIdentificationSettings()
	quickCtx, cancelQuick := context.WithTimeout(ctx, 15*time.Second)
	quickEvidence := r.extractQuickEvidence(quickCtx, job.UserID, images, job.Notes)
	cancelQuick()

	bounds := deepPipelineBounds(settings)
	if deadline, ok := ctx.Deadline(); ok {
		remainingDuration := time.Until(deadline)
		if remainingDuration <= 0 {
			return nil, context.DeadlineExceeded
		}
		remaining := int((remainingDuration + time.Second - 1) / time.Second)
		if remaining > deepPipelineHardTimeoutSafetyMarginS {
			remaining -= deepPipelineHardTimeoutSafetyMarginS
		}
		if remaining < bounds.TotalTimeoutS {
			bounds.TotalTimeoutS = remaining
		}
	}

	internalToken, err := r.tokenSvc.MintForJobWithTTL(
		job.UserID, job.ID,
		time.Duration(bounds.TotalTimeoutS+deepPipelineTokenTTLMarginS)*time.Second,
	)
	if err != nil {
		return nil, fmt.Errorf("mint internal token: %w", err)
	}

	var providerOverride []string
	if job.RequestedProviders != "" {
		providerOverride = strings.Split(job.RequestedProviders, ",")
	}

	proxyReq := DeepIdentifyProxyRequest{
		JobID:            job.ID,
		SchemaVersion:    1,
		LLM:              llmCfg,
		Images:           images,
		Notes:            job.Notes,
		QuickEvidence:    quickEvidence,
		ProviderOverride: providerOverride,
		ProviderCatalog:  deepPipelineProviderCatalog(settings),
		Bounds:           bounds,
		ToolsBaseURL:     r.toolsBaseURL,
		InternalToken:    internalToken,
	}

	var lastSynthesis json.RawMessage
	var lastErrorCode, lastErrorMessage string
	var seq int64

	// providerClaims accumulates each provider_result frame's validated
	// claims keyed by provider, indexed positionally so the terminal
	// synthesis' proposed_fields.evidence_refs (contract §5) can be
	// resolved into full citation-bearing evidence when the rich proposal
	// document is built (B1: no citations/confidence are dropped). The full
	// per-provider claim list (in the emitted order) is retained verbatim so
	// each synthesis `claim_index` still resolves to the exact claim the
	// synthesizer saw; per-claim citation-host re-validation happens later in
	// buildDeepProposalDocumentJSON, not by reindexing here.
	providerClaims := map[string][]deepProposalClaim{}
	providerStartedAt := map[models.DeepProviderName]time.Time{}

	onFrame := func(frame DeepIdentifyFrame) error {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		// persistPayload is the JSON persisted into the user-visible event
		// log. It defaults to the raw internal frame, but privacy/event-bloat
		// sensitive frames (provider_result) are reduced to the bounded
		// public payload defined in contracts/sse-events.md §2 before
		// persistence, so full provider claims/citations never enter the
		// owner-facing, replayable event stream (FR-036) even though the
		// runner itself consumes them in-memory to build the proposal.
		persistPayload := string(frame.Raw)
		switch frame.Type {
		case "provider_started":
			var payload struct {
				Provider    string `json:"provider"`
				Automatable *bool  `json:"automatable"`
			}
			if jsonErr := json.Unmarshal(frame.Raw, &payload); jsonErr == nil {
				if provider, ok := deepProviderName(payload.Provider); ok {
					startedAt := time.Now()
					providerStartedAt[provider] = startedAt
					automatable := true
					if payload.Automatable != nil {
						automatable = *payload.Automatable
					}
					if recordErr := r.repo.RecordProviderStarted(job.ID, job.UserID, provider, automatable, startedAt); recordErr != nil && r.logger != nil {
						r.logger.Error("deep-identification", "failed to record provider start job=%d provider=%s: %v", job.ID, provider, recordErr)
					}
				}
			}
		case "router_selected":
			var payload struct {
				Selected  []string `json:"selected"`
				Rationale string   `json:"rationale"`
				Skipped   []struct {
					Provider string `json:"provider"`
					Reason   string `json:"reason"`
				} `json:"skipped"`
			}
			if jsonErr := json.Unmarshal(frame.Raw, &payload); jsonErr == nil {
				persistPayload = deepRouterSelectedPublicPayloadJSON(payload.Selected, payload.Rationale, payload.Skipped)
				if updateErr := r.repo.RecordRouterSelection(job.ID, job.UserID, payload.Selected, payload.Rationale); updateErr != nil && r.logger != nil {
					r.logger.Error("deep-identification", "failed to persist router selection for job %d: %v", job.ID, updateErr)
				}
			}
		case "synthesis":
			var payload struct {
				Report json.RawMessage `json:"report"`
			}
			if jsonErr := json.Unmarshal(frame.Raw, &payload); jsonErr == nil && len(payload.Report) > 0 {
				lastSynthesis = payload.Report
			} else {
				// Some producers may emit the DeepSynthesis fields inline
				// (no wrapping "report" key) - fall back to treating the
				// whole payload minus "type" as the report.
				lastSynthesis = frame.Raw
			}
		case "provider_result":
			// The internal provider_result frame carries the full
			// ProviderEvidence (contract §3/§4). Consume its claims for the
			// proposal, but persist only the bounded public payload.
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
			if jsonErr := json.Unmarshal(frame.Raw, &evidence); jsonErr == nil && evidence.Provider != "" {
				providerClaims[evidence.Provider] = evidence.Claims
				persistPayload = deepProviderResultPublicPayloadJSON(
					evidence.Provider, evidence.Status, evidence.Confidence,
					evidence.ErrorKind, evidence.LinkOut, evidence.Claims,
				)
				if provider, ok := deepProviderName(evidence.Provider); ok {
					completedAt := time.Now()
					startedAt, found := providerStartedAt[provider]
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
					if recordErr := r.repo.RecordProviderResult(
						job.ID, job.UserID, provider, status, automatable,
						evidence.Confidence, evidence.CallCount, latencyMS, errorKind,
						startedAt, completedAt,
					); recordErr != nil {
						if r.logger != nil {
							r.logger.Error("deep-identification", "failed to record provider result job=%d provider=%s: %v", job.ID, provider, recordErr)
						}
					} else if r.logger != nil {
						r.logger.Info("deep-identification", "provider settled job=%d provider=%s status=%s latency_ms=%d call_count=%d", job.ID, provider, status, latencyMS, evidence.CallCount)
					}
				}
			}
		case "evaluation":
			var payload struct {
				DisagreementCount int `json:"disagreement_count"`
				ResolvedCount     int `json:"resolved_count"`
			}
			if jsonErr := json.Unmarshal(frame.Raw, &payload); jsonErr == nil {
				persistPayload = fmt.Sprintf(
					`{"disagreementCount":%d,"resolvedCount":%d}`,
					payload.DisagreementCount,
					payload.ResolvedCount,
				)
			}
		case "progress":
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
				persistPayload = string(encoded)
			}
		case "error":
			var payload struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			}
			if jsonErr := json.Unmarshal(frame.Raw, &payload); jsonErr == nil {
				lastErrorCode = payload.Code
				lastErrorMessage = payload.Message
			}
		}

		eventType, ok := deepPipelineEventType(frame.Type)
		if !ok {
			// synthesis/error are terminal and are surfaced through the
			// returned DeepPipelineResult/error instead of AppendEvent, so
			// SettleTerminal remains the single writer of the terminal
			// event row.
			return nil
		}
		seq++
		if _, appendErr := r.repo.AppendEvent(job.ID, job.UserID, eventType, persistPayload); appendErr != nil {
			if r.logger != nil {
				r.logger.Error("deep-identification", "failed to append event for job %d: %v", job.ID, appendErr)
			}
		} else if r.broker != nil {
			r.broker.Publish(job.ID)
		}
		return nil
	}

	streamErr := r.proxy.StreamDeepIdentification(ctx, proxyReq, onFrame)

	if ctx.Err() != nil {
		// Cancelled or timed out: runJob's status switch checks
		// jobCtx.Err()/timeoutCtx.Err() before ever consulting the
		// returned result, so any partial evidence gathered above is
		// never applied (FR-018/FR-019). Returning ctx.Err() here keeps
		// Run's contract honest even though the caller does not depend on
		// this specific error value in that branch.
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			unsettledStatus = models.DeepProviderRunTimedOut
			unsettledErrorKind = "timeout"
		} else {
			unsettledStatus = models.DeepProviderRunSkipped
			unsettledErrorKind = ""
		}
		return nil, ctx.Err()
	}

	if streamErr != nil {
		if errors.Is(streamErr, ErrDeepStreamEndedWithoutTerminal) {
			return nil, streamErr
		}
		return nil, fmt.Errorf("deep identification stream failed: %w", streamErr)
	}

	if lastSynthesis != nil {
		var partial struct {
			PartialSuccess bool `json:"partial_success"`
		}
		_ = json.Unmarshal(lastSynthesis, &partial)
		if partial.PartialSuccess {
			unsettledStatus = models.DeepProviderRunTimedOut
			unsettledErrorKind = "timeout"
		} else {
			unsettledStatus = models.DeepProviderRunSkipped
			unsettledErrorKind = ""
		}
		return &DeepPipelineResult{
			ReportJSON:   string(lastSynthesis),
			ProposalJSON: buildDeepProposalDocumentJSON(lastSynthesis, job.CoinID, providerClaims),
			Partial:      partial.PartialSuccess,
		}, nil
	}

	// A terminal frame was observed (StreamDeepIdentification would
	// otherwise have returned ErrDeepStreamEndedWithoutTerminal) but it was
	// an `error` frame, not `synthesis`.
	if lastErrorCode == "" {
		lastErrorCode = "internal"
	}
	if lastErrorMessage == "" {
		lastErrorMessage = "The identification pipeline reported an error."
	}
	return nil, fmt.Errorf("deep identification pipeline error %s: %s", lastErrorCode, lastErrorMessage)
}

func (r *DeepIdentificationPipelineRunner) extractQuickEvidence(
	ctx context.Context,
	userID uint,
	images []DeepIdentifyImageProxy,
	notes string,
) *DeepQuickEvidenceProxy {
	if r.coinLookup == nil {
		return nil
	}
	lookupImages := make([]string, 0, len(images))
	imageRoles := make([]string, 0, len(images))
	for _, image := range images {
		lookupImages = append(lookupImages, image.DataURI)
		role := image.Role
		if role == "hint" {
			role = "notes"
		}
		imageRoles = append(imageRoles, role)
	}
	result, err := r.coinLookup.Lookup(ctx, userID, CoinLookupRequest{
		Images:     lookupImages,
		ImageRoles: imageRoles,
		Notes:      notes,
	})
	if err != nil {
		if r.logger != nil {
			r.logger.Warn("deep-identification", "quick evidence extraction failed for user %d: %v", userID, err)
		}
		return nil
	}

	keys := make([]string, 0, len(result.ExtractedData.CoinFields))
	for key := range result.ExtractedData.CoinFields {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	if len(keys) > 50 {
		keys = keys[:50]
	}
	coinFields := make(map[string]string, len(keys))
	for _, key := range keys {
		value := result.ExtractedData.CoinFields[key]
		if value == nil {
			continue
		}
		normalized := truncateDeepEvidence(strings.TrimSpace(fmt.Sprint(value)), 500)
		if normalized != "" && normalized != "<nil>" {
			coinFields[truncateDeepEvidence(key, 100)] = normalized
		}
	}
	evidence := &DeepQuickEvidenceProxy{
		LabelText:    truncateDeepEvidence(result.ExtractedData.LabelText, 2000),
		CoinFields:   coinFields,
		Confidence:   truncateDeepEvidence(result.ExtractedData.Confidence, 16),
		NumistaQuery: truncateDeepEvidence(result.ProposedNumistaQuery, 300),
	}
	if ngc := result.ExtractedData.NGC; ngc != nil {
		certNumber := ngc.NormalizedCert
		if certNumber == "" {
			certNumber = ngc.CertNumber
		}
		evidence.NGC = &DeepQuickEvidenceNGCProxy{
			CertNumber: truncateDeepEvidence(certNumber, 40),
			Grade:      truncateDeepEvidence(ngc.Grade, 32),
			LookupURL:  boundedDeepEvidenceURL(ngc.LookupURL),
		}
	}
	return evidence
}

func truncateDeepEvidence(value string, maxRunes int) string {
	value = strings.TrimSpace(value)
	runes := []rune(value)
	if len(runes) <= maxRunes {
		return value
	}
	return string(runes[:maxRunes])
}

func boundedDeepEvidenceURL(value string) string {
	value = strings.TrimSpace(value)
	if len(value) > 500 {
		return ""
	}
	return value
}

func deepProviderName(value string) (models.DeepProviderName, bool) {
	provider := models.DeepProviderName(value)
	switch provider {
	case models.DeepProviderNomisma, models.DeepProviderNumista, models.DeepProviderNGC,
		models.DeepProviderOCRE, models.DeepProviderRPC:
		return provider, true
	default:
		return "", false
	}
}

func deepProviderRunStatus(value string) models.DeepProviderRunStatus {
	switch models.DeepProviderRunStatus(value) {
	case models.DeepProviderRunContributed:
		return models.DeepProviderRunContributed
	case models.DeepProviderRunNoMatch:
		return models.DeepProviderRunNoMatch
	case models.DeepProviderRunTimedOut:
		return models.DeepProviderRunTimedOut
	case models.DeepProviderRunSkipped:
		return models.DeepProviderRunSkipped
	case models.DeepProviderRunNotAutomated:
		return models.DeepProviderRunNotAutomated
	case models.DeepProviderRunUnavailable:
		return models.DeepProviderRunUnavailable
	case models.DeepProviderRunFailed:
		return models.DeepProviderRunFailed
	default:
		if value == "match" {
			return models.DeepProviderRunContributed
		}
		return models.DeepProviderRunFailed
	}
}

func deepProviderErrorKind(value string) string {
	switch value {
	case "timeout", "quota", "unconfigured", "upstream", "invalid_response":
		return value
	default:
		return ""
	}
}

type deepRouterSkippedPublic struct {
	Provider string `json:"provider"`
	Reason   string `json:"reason"`
}

func deepRouterSelectedPublicPayloadJSON(
	selected []string,
	rationale string,
	skipped []struct {
		Provider string `json:"provider"`
		Reason   string `json:"reason"`
	},
) string {
	publicSkipped := make([]deepRouterSkippedPublic, 0, len(skipped))
	for _, item := range skipped {
		publicSkipped = append(publicSkipped, deepRouterSkippedPublic(item))
	}
	encoded, err := json.Marshal(struct {
		SelectedProviders []string                  `json:"selectedProviders"`
		Rationale         string                    `json:"rationale"`
		Skipped           []deepRouterSkippedPublic `json:"skipped"`
	}{selected, rationale, publicSkipped})
	if err != nil {
		return "{}"
	}
	return string(encoded)
}

func deepProgressMessage(phase string) string {
	switch phase {
	case "image_evidence_ready":
		return "Image evidence prepared"
	case "provider_fanout_started":
		return "Running selected providers"
	case "evaluation_started":
		return "Comparing provider findings"
	default:
		return "Deep Analysis is progressing"
	}
}

// loadImages reads each non-deleted artifact's bytes from disk and encodes
// them as data URIs for the Python request (contract §2). Hint artifacts
// are included like any other role; the Python model marks their vision-
// prompt exclusion by role alone (FR-004).
func (r *DeepIdentificationPipelineRunner) loadImages(jobID uint) ([]DeepIdentifyImageProxy, error) {
	artifacts, err := r.repo.ListArtifacts(jobID)
	if err != nil {
		return nil, err
	}
	images := make([]DeepIdentifyImageProxy, 0, len(artifacts))
	for _, a := range artifacts {
		if a.DeletedAt != nil {
			continue
		}
		data, readErr := os.ReadFile(a.FilePath)
		if readErr != nil {
			return nil, fmt.Errorf("read artifact %d: %w", a.ID, readErr)
		}
		mimeType := a.MimeType
		if mimeType == "" {
			mimeType = "image/jpeg"
		}
		dataURI := fmt.Sprintf("data:%s;base64,%s", mimeType, base64.StdEncoding.EncodeToString(data))
		images = append(images, DeepIdentifyImageProxy{
			Role:    string(a.Role),
			DataURI: dataURI,
		})
	}
	return images, nil
}

// deepPipelineBounds derives the per-run bounds Go supplies to Python
// (contract §2/§6) from the live settings snapshot plus fixed, documented
// defaults - see the const block above.
func deepPipelineBounds(settings DeepIdentificationSettings) DeepIdentifyBoundsProxy {
	totalTimeoutS := int(settings.HardTimeout.Seconds()) - deepPipelineHardTimeoutSafetyMarginS
	if totalTimeoutS < deepPipelineMinTotalTimeoutS {
		totalTimeoutS = deepPipelineMinTotalTimeoutS
	}
	if totalTimeoutS > deepPipelineMaxTotalTimeoutS {
		totalTimeoutS = deepPipelineMaxTotalTimeoutS
	}
	maxProviders := settings.MaxProviders
	if maxProviders < 1 {
		maxProviders = 1
	}
	if maxProviders > 10 {
		maxProviders = 10
	}
	return DeepIdentifyBoundsProxy{
		MaxProviders:     maxProviders,
		MaxConcurrency:   deepPipelineDefaultMaxConcurrency,
		ProviderTimeoutS: deepPipelineDefaultProviderTimeoutS,
		TotalTimeoutS:    totalTimeoutS,
		RecursionLimit:   deepPipelineDefaultRecursionLimit,
	}
}

// deepPipelineProviderCatalog builds the fixed MVP provider catalog
// (contract §2 example): Nomisma/Numista automated via existing Go
// services/internal tools; NGC is OCR/link-out only; RPC is always typed
// not_automated/unavailable this phase (deferred T156 gate, flag not read
// here). OCRE is conditionally automated (Feature 345): when the
// DeepIdentificationOCREEnabled setting is on it becomes a normal automatable
// catalog entry with its own call budget; when off it stays a typed
// not_automated entry — byte-for-byte the previous beta behavior — so the
// flag is the sole enable/rollback control (FR-004/FR-016).
func deepPipelineProviderCatalog(settings DeepIdentificationSettings) []DeepProviderCatalogEntryProxy {
	ocreEntry := DeepProviderCatalogEntryProxy{Provider: "ocre", Automatable: false, Reason: "provider_disabled"}
	if settings.OCREEnabled {
		ocreEntry = DeepProviderCatalogEntryProxy{Provider: "ocre", Automatable: true, CallBudget: settings.OCRECallBudget}
	}
	return []DeepProviderCatalogEntryProxy{
		{Provider: "numista", Automatable: true, CallBudget: settings.NumistaCallBudget},
		{Provider: "nomisma", Automatable: true, CallBudget: settings.MaxProviders},
		{
			Provider:    "ngc",
			Automatable: false,
			Reason:      "terms_prohibit_automated_access",
			LinkOut:     "https://www.ngccoin.com/verify/",
		},
		ocreEntry,
		{Provider: "rpc", Automatable: false, Reason: "no_public_api"},
	}
}

// deepPipelineEventType maps a Python SSE frame `type` (contract §3) to the
// persisted DeepIdentificationEventType. `synthesis`/`error` are terminal
// and intentionally excluded (ok=false) - they surface through the
// returned DeepPipelineResult/error instead, so SettleTerminal remains the
// only writer of the terminal event row.
func deepPipelineEventType(frameType string) (models.DeepIdentificationEventType, bool) {
	switch frameType {
	case "router_selected":
		return models.DeepEventRouterSelected, true
	case "provider_started":
		return models.DeepEventProviderStarted, true
	case "provider_result":
		return models.DeepEventProviderResult, true
	case "evaluation":
		return models.DeepEventEvaluation, true
	case "synthesis_started":
		return models.DeepEventSynthesisStart, true
	case "progress":
		return models.DeepEventProgress, true
	default:
		return "", false
	}
}

// deepProviderResultPublicPayload is the bounded, owner-facing payload
// persisted for a provider_result event (contracts/sse-events.md §2):
// `{provider, status, confidence, claimCount, errorKind?, linkOut?}`. The
// full per-claim citation/excerpt evidence from the internal frame is
// deliberately excluded — it is consumed only in-memory by the runner to
// build the confirm-gated proposal, never leaked into the replayable,
// user-visible event log (FR-036, privacy/event-bloat decision).
type deepProviderResultPublicPayload struct {
	Provider   string  `json:"provider"`
	Status     string  `json:"status"`
	Confidence float64 `json:"confidence"`
	ClaimCount int     `json:"claimCount"`
	ErrorKind  string  `json:"errorKind,omitempty"`
	LinkOut    string  `json:"linkOut,omitempty"`
}

// deepProviderResultPublicPayloadJSON reduces a full internal
// ProviderEvidence frame to the bounded public payload. claimCount counts
// only citation-host-allowlisted claims, so a compromised/buggy provider
// frame injecting off-allowlist citations can neither inflate the count nor
// surface an arbitrary URL to the owner (SC-006, Principle V). Falls back to
// an empty JSON object only on the (unreachable) marshal error path.
func deepProviderResultPublicPayloadJSON(provider, status string, confidence float64, errorKind, linkOut string, claims []deepProposalClaim) string {
	claimCount := 0
	for _, claim := range claims {
		if deepCitationHostAllowed(provider, claim.Citation) {
			claimCount++
		}
	}
	payload := deepProviderResultPublicPayload{
		Provider:   provider,
		Status:     status,
		Confidence: confidence,
		ClaimCount: claimCount,
		ErrorKind:  errorKind,
		LinkOut:    linkOut,
	}
	out, err := json.Marshal(payload)
	if err != nil {
		return "{}"
	}
	return string(out)
}

// deepCitationHostAllowlist mirrors the Python
// merge.CITATION_HOST_ALLOWLIST (contracts/agent-internal-contract.md §4):
// every persisted claim's citation host must belong to the emitting
// provider's canonical allowlist. Python drops non-allowlisted citations
// before emission; Go re-checks here before the citation is persisted into
// a proposal, so a compromised/buggy provider frame can never inject an
// arbitrary citation URL into stored owner-facing data (SC-006, Principle V).
var deepCitationHostAllowlist = map[string]map[string]bool{
	"numista": {"en.numista.com": true, "api.numista.com": true},
	"nomisma": {"nomisma.org": true},
	"ngc":     {"www.ngccoin.com": true},
	"ocre":    {"numismatics.org": true},
	"rpc":     {"rpc.ashmus.ox.ac.uk": true},
}

// deepCitationHostAllowed reports whether citation is an absolute http(s)
// URL whose host is on provider's canonical allowlist.
func deepCitationHostAllowed(provider, citation string) bool {
	allow, ok := deepCitationHostAllowlist[provider]
	if !ok {
		return false
	}
	u, err := url.Parse(citation)
	if err != nil {
		return false
	}
	return allow[strings.ToLower(u.Hostname())]
}

// buildDeepProposalDocumentJSON translates the terminal synthesis'
// `proposed_fields` (contract §5) into the rich, owner-editable
// deepProposalDocument the proposal/apply endpoints and the frontend
// DeepProposalEditor consume (data-model.md §7, OpenAPI Proposal schema).
//
// It is the single writer of ProposalJSON and enforces two invariants:
//  1. saved-coin jobs retain only coin-writable fields, while intake jobs
//     translate findings into the existing draft-writable title/notes fields;
//  2. each field's per-provider evidence is resolved from the streamed
//     provider_result claims via evidence_refs and re-validated against the
//     citation host allowlist, preserving citations + confidence while
//     rejecting any non-allowlisted citation (SC-006).
//
// Owner-decision fields are initialized to their pristine state
// (ownerEdited=false, ownerValue=null, accepted=null) so nothing is
// auto-accepted and no coin data is written until an explicit confirm.
type deepSynthesisProposedField struct {
	Value        string  `json:"value"`
	Confidence   float64 `json:"confidence"`
	EvidenceRefs []struct {
		Provider   string `json:"provider"`
		ClaimIndex *int   `json:"claim_index"`
	} `json:"evidence_refs"`
}

func buildDeepProposalDocumentJSON(reportJSON json.RawMessage, targetCoinID *uint, providerClaims map[string][]deepProposalClaim) string {
	var report struct {
		Narrative      string                                `json:"narrative"`
		ProposedFields map[string]deepSynthesisProposedField `json:"proposed_fields"`
	}
	if err := json.Unmarshal(reportJSON, &report); err != nil {
		return ""
	}

	if targetCoinID == nil {
		fields := buildDeepIntakeProposalFields(report.Narrative, report.ProposedFields)
		if len(fields) == 0 {
			return ""
		}
		out, err := json.Marshal(deepProposalDocument{SchemaVersion: 1, Fields: fields})
		if err != nil {
			return ""
		}
		return string(out)
	}

	if len(report.ProposedFields) == 0 {
		return ""
	}

	fields := make(map[string]*deepProposalFieldEntry, len(report.ProposedFields))
	for name, pf := range report.ProposedFields {
		if _, allowed := deepProposalCoinFieldAllowlist[name]; !allowed {
			continue // restrict to the existing update allowlist
		}
		entry := &deepProposalFieldEntry{
			Proposed:    pf.Value,
			Confidence:  pf.Confidence,
			OwnerEdited: false,
			OwnerValue:  nil,
			Accepted:    nil,
		}
		for _, ref := range pf.EvidenceRefs {
			// `provider: "image"` refs (contract §5) carry no citation and
			// are intentionally not rendered as a Claim.
			if ref.Provider == "" || ref.Provider == "image" || ref.ClaimIndex == nil {
				continue
			}
			claims, ok := providerClaims[ref.Provider]
			if !ok || *ref.ClaimIndex < 0 || *ref.ClaimIndex >= len(claims) {
				continue
			}
			claim := claims[*ref.ClaimIndex]
			if !deepCitationHostAllowed(ref.Provider, claim.Citation) {
				continue
			}
			entry.Evidence = append(entry.Evidence, claim)
		}
		fields[name] = entry
	}

	if len(fields) == 0 {
		return ""
	}

	doc := deepProposalDocument{
		SchemaVersion: 1,
		TargetCoinID:  targetCoinID,
		Fields:        fields,
	}
	out, err := json.Marshal(doc)
	if err != nil {
		return ""
	}
	return string(out)
}

func buildDeepIntakeProposalFields(
	narrative string,
	proposedFields map[string]deepSynthesisProposedField,
) map[string]*deepProposalFieldEntry {
	fields := make(map[string]*deepProposalFieldEntry)
	newEntry := func(value string) *deepProposalFieldEntry {
		return &deepProposalFieldEntry{
			Proposed: value, OwnerEdited: false, OwnerValue: nil, Accepted: nil,
		}
	}

	for _, name := range []string{"era", "dateRange"} {
		if proposed, ok := proposedFields[name]; ok && strings.TrimSpace(proposed.Value) != "" {
			entry := newEntry(proposed.Value)
			entry.Confidence = proposed.Confidence
			fields[name] = entry
		}
	}

	titleParts := make([]string, 0, 2)
	for _, name := range []string{"ruler", "denomination"} {
		if proposed, ok := proposedFields[name]; ok && strings.TrimSpace(proposed.Value) != "" {
			titleParts = append(titleParts, strings.TrimSpace(proposed.Value))
		}
	}
	if len(titleParts) > 0 {
		fields["workingTitle"] = newEntry(truncateDeepProposalText(strings.Join(titleParts, " "), 200))
	}

	notesParts := make([]string, 0, 2)
	if trimmed := strings.TrimSpace(narrative); trimmed != "" {
		notesParts = append(notesParts, trimmed)
	}
	fieldNames := make([]string, 0, len(proposedFields))
	for name, proposed := range proposedFields {
		if _, allowed := deepProposalCoinFieldAllowlist[name]; allowed && strings.TrimSpace(proposed.Value) != "" {
			fieldNames = append(fieldNames, name)
		}
	}
	sort.Strings(fieldNames)
	if len(fieldNames) > 0 {
		findings := make([]string, 0, len(fieldNames)+1)
		findings = append(findings, "Deep Analysis findings:")
		for _, name := range fieldNames {
			findings = append(findings, fmt.Sprintf("%s: %s", name, proposedFields[name].Value))
		}
		notesParts = append(notesParts, strings.Join(findings, "\n"))
	}
	if len(notesParts) > 0 {
		fields["notes"] = newEntry(truncateDeepProposalText(strings.Join(notesParts, "\n\n"), 5000))
	}
	return fields
}

func truncateDeepProposalText(value string, maxRunes int) string {
	runes := []rune(value)
	if len(runes) <= maxRunes {
		return value
	}
	return string(runes[:maxRunes])
}
