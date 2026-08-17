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
	quickCtx, cancelQuick := deepQuickLookupContext(ctx, settings)
	quickEvidence, quickOutcome := r.extractQuickEvidence(quickCtx, job.UserID, images, job.Notes)
	cancelQuick()
	r.emitQuickLookupOutcomeEvent(job, quickOutcome)

	bounds := deepPipelineBounds(settings)
	bounds, deadlineErr := deepPipelineApplyRemainingBudget(ctx, bounds)
	if deadlineErr != nil {
		return nil, deadlineErr
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

	var seq int64
	translator := newDeepFrameTranslator(r, job)

	onFrame := func(frame DeepIdentifyFrame) error {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		persistPayload := translator.translate(frame)

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

	if translator.lastSynthesis != nil {
		var partial struct {
			PartialSuccess bool `json:"partial_success"`
		}
		_ = json.Unmarshal(translator.lastSynthesis, &partial)
		if partial.PartialSuccess {
			unsettledStatus = models.DeepProviderRunTimedOut
			unsettledErrorKind = "timeout"
		} else {
			unsettledStatus = models.DeepProviderRunSkipped
			unsettledErrorKind = ""
		}
		return &DeepPipelineResult{
			ReportJSON:   deepPipelineAugmentReportWithQuickLookupOutcome(string(translator.lastSynthesis), quickOutcome),
			ProposalJSON: buildDeepProposalDocumentJSON(translator.lastSynthesis, job.CoinID, translator.providerClaims),
			Partial:      partial.PartialSuccess,
		}, nil
	}

	// A terminal frame was observed (StreamDeepIdentification would
	// otherwise have returned ErrDeepStreamEndedWithoutTerminal) but it was
	// an `error` frame, not `synthesis`.
	lastErrorCode := translator.lastErrorCode
	lastErrorMessage := translator.lastErrorMessage
	if lastErrorCode == "" {
		lastErrorCode = "internal"
	}
	if lastErrorMessage == "" {
		lastErrorMessage = "The identification pipeline reported an error."
	}
	return nil, fmt.Errorf("deep identification pipeline error %s: %s", lastErrorCode, lastErrorMessage)
}

// quickLookupOutcome is the typed result of the quick-evidence extraction
// pass inside Deep Analysis (data-model.md §5, 351 T014). It replaces the
// prior "log a Warn and return nil" pattern, which conflated "no quick
// evidence found" with "the quick lookup did not complete" — the defect
// that made the Maximinus run report zero NGC data even though standalone
// Quick Lookup succeeded a minute later.
type quickLookupOutcome string

const (
	// quickLookupOutcomeOK: the lookup completed and returned at least one
	// usable field (label text, a coin field, confidence, an NGC cert, or a
	// proposed Numista query).
	quickLookupOutcomeOK quickLookupOutcome = "ok"
	// quickLookupOutcomeNoData: the lookup completed but genuinely found
	// nothing usable in the images.
	quickLookupOutcomeNoData quickLookupOutcome = "no_data"
	// quickLookupOutcomeUnavailable: the lookup did not complete — deadline
	// exceeded, a Lookup error, or the quick-lookup subsystem was not wired
	// for this runner instance. Distinct from no_data (data-model.md §5).
	quickLookupOutcomeUnavailable quickLookupOutcome = "unavailable"
)

// deepQuickLookupProgressMessage returns the fixed, sanitized message text
// for a quick-lookup progress event. It carries only the outcome class —
// never label text, cert numbers, notes, image data, or query strings
// (FR-030, 344 FR-036).
func deepQuickLookupProgressMessage(outcome quickLookupOutcome) string {
	switch outcome {
	case quickLookupOutcomeOK:
		return "Quick lookup found supporting data"
	case quickLookupOutcomeNoData:
		return "Quick lookup completed with no supporting data"
	case quickLookupOutcomeUnavailable:
		return "Quick lookup did not complete"
	default:
		return "Quick lookup outcome unknown"
	}
}

// emitQuickLookupOutcomeEvent appends the typed quick-lookup outcome as a
// `progress` event (contracts/sse-events.md §2 shape: {phase, message}) — no
// new SSE vocabulary is introduced (351 T015). The payload carries only the
// fixed outcome-class message above.
func (r *DeepIdentificationPipelineRunner) emitQuickLookupOutcomeEvent(job *models.DeepIdentificationJob, outcome quickLookupOutcome) {
	payload, err := json.Marshal(map[string]string{
		"phase":   "quick_lookup",
		"message": deepQuickLookupProgressMessage(outcome),
	})
	if err != nil {
		return
	}
	if _, appendErr := r.repo.AppendEvent(job.ID, job.UserID, models.DeepEventProgress, string(payload)); appendErr != nil {
		if r.logger != nil {
			r.logger.Error("deep-identification", "failed to append quick-lookup progress event for job %d: %v", job.ID, appendErr)
		}
		return
	}
	if r.broker != nil {
		r.broker.Publish(job.ID)
	}
}

// deepPipelineAugmentReportWithQuickLookupOutcome adds the typed quick-lookup
// outcome as an additive `quickLookupOutcome` key on the synthesized report
// JSON (no schema migration — the report column already exists, mirroring
// how data-model.md §4 documents `image_hypothesis` as an additive report
// key). Old readers ignore the unknown key; new readers can distinguish
// "no cert data existed" from "the quick lookup did not complete" (T016,
// owned by Aurelia). On any marshal failure the original report is returned
// unchanged rather than dropping the synthesized report.
func deepPipelineAugmentReportWithQuickLookupOutcome(reportJSON string, outcome quickLookupOutcome) string {
	var report map[string]json.RawMessage
	if err := json.Unmarshal([]byte(reportJSON), &report); err != nil {
		return reportJSON
	}
	encodedOutcome, err := json.Marshal(string(outcome))
	if err != nil {
		return reportJSON
	}
	report["quickLookupOutcome"] = encodedOutcome
	out, err := json.Marshal(report)
	if err != nil {
		return reportJSON
	}
	return string(out)
}

// deepQuickEvidenceIsEmpty reports whether a successfully-returned quick
// evidence proxy has no usable content at all, which distinguishes the
// no_data outcome from ok (351 T014). Confidence is deliberately excluded:
// CoinLookupService.determineConfidence always returns "low"/"medium"/"high"
// even when nothing was extracted, so it is never itself evidence of data.
func deepQuickEvidenceIsEmpty(evidence *DeepQuickEvidenceProxy) bool {
	if evidence == nil {
		return true
	}
	return evidence.LabelText == "" &&
		len(evidence.CoinFields) == 0 &&
		evidence.NumistaQuery == "" &&
		evidence.NGC == nil
}

func (r *DeepIdentificationPipelineRunner) extractQuickEvidence(
	ctx context.Context,
	userID uint,
	images []DeepIdentifyImageProxy,
	notes string,
) (*DeepQuickEvidenceProxy, quickLookupOutcome) {
	if r.coinLookup == nil {
		return nil, quickLookupOutcomeUnavailable
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
		return nil, quickLookupOutcomeUnavailable
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
	if deepQuickEvidenceIsEmpty(evidence) {
		return nil, quickLookupOutcomeNoData
	}
	return evidence, quickLookupOutcomeOK
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

// deepQuickLookupContext bounds the quick-evidence extraction pass (a full
// vision LLM round trip) to the admin-tunable
// SettingDeepIdentificationQuickLookupTimeoutSeconds, replacing the prior
// `15*time.Second` magic literal (351 T011, FR-038). It is consumed from the
// same ctx as the overall job hard timeout, so every second spent here is a
// deduction from - not an addition to - deepPipelineApplyRemainingBudget's
// result below; extracted as its own function so T013 can assert the
// deadline directly without exercising the full Run pipeline.
func deepQuickLookupContext(ctx context.Context, settings DeepIdentificationSettings) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, settings.QuickLookupTimeout)
}

// deepPipelineApplyRemainingBudget clamps bounds.TotalTimeoutS to whatever
// budget remains on ctx's deadline (runner.go:116-123 pre-351) after the
// quick-lookup pass has already consumed part of it, minus the hard-timeout
// safety margin. Extracted so T012's measured interaction between
// SettingDeepIdentificationQuickLookupTimeoutSeconds and
// SettingDeepIdentificationHardTimeoutSeconds can be regression-tested
// directly (T013) rather than only asserted by inspection.
func deepPipelineApplyRemainingBudget(ctx context.Context, bounds DeepIdentifyBoundsProxy) (DeepIdentifyBoundsProxy, error) {
	deadline, ok := ctx.Deadline()
	if !ok {
		return bounds, nil
	}
	remainingDuration := time.Until(deadline)
	if remainingDuration <= 0 {
		return bounds, context.DeadlineExceeded
	}
	remaining := int((remainingDuration + time.Second - 1) / time.Second)
	if remaining > deepPipelineHardTimeoutSafetyMarginS {
		remaining -= deepPipelineHardTimeoutSafetyMarginS
	}
	if remaining < bounds.TotalTimeoutS {
		bounds.TotalTimeoutS = remaining
	}
	return bounds, nil
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
