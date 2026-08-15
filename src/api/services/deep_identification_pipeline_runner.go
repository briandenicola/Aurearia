package services

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
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
	images, err := r.loadImages(job.ID)
	if err != nil {
		return nil, fmt.Errorf("load artifacts for job %d: %w", job.ID, err)
	}

	llmCfg, err := r.settingsSvc.ResolveLLMConfig()
	if err != nil {
		return nil, fmt.Errorf("resolve llm config: %w", err)
	}

	settings := r.settingsSvc.GetDeepIdentificationSettings()
	bounds := deepPipelineBounds(settings)

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
		ProviderOverride: providerOverride,
		ProviderCatalog:  deepPipelineProviderCatalog(settings),
		Bounds:           bounds,
		ToolsBaseURL:     r.toolsBaseURL,
		InternalToken:    internalToken,
	}

	var lastSynthesis json.RawMessage
	var lastErrorCode, lastErrorMessage string
	var seq int64

	onFrame := func(frame DeepIdentifyFrame) error {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		switch frame.Type {
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
		if _, appendErr := r.repo.AppendEvent(job.ID, job.UserID, eventType, string(frame.Raw)); appendErr != nil {
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
		return &DeepPipelineResult{
			ReportJSON:   string(lastSynthesis),
			ProposalJSON: deepPipelineProposalJSON(lastSynthesis),
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
// services/internal tools; NGC is OCR/link-out only; OCRE/RPC are always
// typed not_automated/unavailable this phase regardless of the
// OCREEnabled/RPCEnabled settings flags, which are reserved for the
// deferred T155/T156 gates and intentionally not read here.
func deepPipelineProviderCatalog(settings DeepIdentificationSettings) []DeepProviderCatalogEntryProxy {
	return []DeepProviderCatalogEntryProxy{
		{Provider: "numista", Automatable: true, CallBudget: settings.NumistaCallBudget},
		{Provider: "nomisma", Automatable: true, CallBudget: settings.MaxProviders},
		{
			Provider:    "ngc",
			Automatable: false,
			Reason:      "terms_prohibit_automated_access",
			LinkOut:     "https://www.ngccoin.com/verify/",
		},
		{Provider: "ocre", Automatable: false, Reason: "pending_license_validation"},
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

// deepPipelineProposalJSON extracts a flat field->value map from the
// synthesis report's `proposed_fields` (contract §5) as the stored
// proposal payload. Later phases (Quick Capture promotion / saved-coin
// update review) define the exact confirm/apply shape; this phase only
// needs a faithful, lossless-enough snapshot of what the pipeline proposed
// so nothing is fabricated or silently dropped.
func deepPipelineProposalJSON(reportJSON json.RawMessage) string {
	var report struct {
		ProposedFields map[string]struct {
			Value string `json:"value"`
		} `json:"proposed_fields"`
	}
	if err := json.Unmarshal(reportJSON, &report); err != nil || len(report.ProposedFields) == 0 {
		return ""
	}
	fields := make(map[string]string, len(report.ProposedFields))
	for field, entry := range report.ProposedFields {
		fields[field] = entry.Value
	}
	out, err := json.Marshal(map[string]any{"fields": fields})
	if err != nil {
		return ""
	}
	return string(out)
}
