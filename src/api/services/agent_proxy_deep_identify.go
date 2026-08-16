package services

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Deep Agentic Coin Identification (344-deep-agentic-coin-identification)
// Go -> Python streaming contract (contracts/agent-internal-contract.md §2/§3).
// Kept in its own file, separate from agent_proxy.go's Feature-341-era proxy
// methods, to keep both files proportional (repository modularity norm).

// DeepIdentifyImageProxy mirrors app.models.requests.DeepIdentifyImage.
type DeepIdentifyImageProxy struct {
	Role     string `json:"role"`
	DataURI  string `json:"data_uri"`
	HintKind string `json:"hint_kind,omitempty"`
}

// DeepProviderCatalogEntryProxy mirrors app.models.requests.DeepProviderCatalogEntry.
type DeepProviderCatalogEntryProxy struct {
	Provider    string `json:"provider"`
	Automatable bool   `json:"automatable"`
	CallBudget  int    `json:"call_budget,omitempty"`
	Reason      string `json:"reason,omitempty"`
	LinkOut     string `json:"link_out,omitempty"`
}

// DeepIdentifyBoundsProxy mirrors app.models.requests.DeepIdentifyBounds.
type DeepIdentifyBoundsProxy struct {
	MaxProviders     int `json:"max_providers"`
	MaxConcurrency   int `json:"max_concurrency"`
	ProviderTimeoutS int `json:"provider_timeout_s"`
	TotalTimeoutS    int `json:"total_timeout_s"`
	RecursionLimit   int `json:"recursion_limit"`
}

// DeepIdentifyProxyRequest mirrors app.models.requests.DeepIdentifyRequest
// exactly (field-for-field, including the `llm` key name used consistently
// by every other Go->Python proxy request in this file).
type DeepIdentifyProxyRequest struct {
	JobID            uint                            `json:"job_id"`
	SchemaVersion    int                             `json:"schema_version"`
	LLM              LLMConfig                       `json:"llm"`
	Images           []DeepIdentifyImageProxy        `json:"images"`
	Notes            string                          `json:"notes,omitempty"`
	ProviderOverride []string                        `json:"provider_override,omitempty"`
	ProviderCatalog  []DeepProviderCatalogEntryProxy `json:"provider_catalog"`
	Bounds           DeepIdentifyBoundsProxy         `json:"bounds"`
	ToolsBaseURL     string                          `json:"tools_base_url,omitempty"`
	InternalToken    string                          `json:"internal_token,omitempty"`
}

// DeepIdentifyFrame is one parsed SSE "data:" line from the Python stream
// (contracts/agent-internal-contract.md §3): the sanitized envelope type
// plus its raw JSON payload, left un-decoded so the caller can pick apart
// only the fields it needs per type.
type DeepIdentifyFrame struct {
	Type string
	Raw  json.RawMessage
}

// ErrDeepStreamEndedWithoutTerminal is returned by StreamDeepIdentification
// when the Python SSE stream reaches EOF without ever emitting a
// `synthesis` or `error` frame (contract §3: "Go treats stream EOF without
// either as agent_unavailable"). The caller (the pipeline runner) surfaces
// this as a plain error; deep_identification_service.go's runJob already
// maps any non-nil pipeline error to failed:agent_unavailable, so no
// special-casing is required at that layer (T072).
var ErrDeepStreamEndedWithoutTerminal = errors.New("deep identification stream ended without a terminal frame")

// deepIdentifyTerminalFrameTypes are the two frame types that end the
// stream (contract §3). Exactly one of them must appear before EOF.
var deepIdentifyTerminalFrameTypes = map[string]bool{
	"synthesis": true,
	"error":     true,
}

// StreamDeepIdentification posts to `{AGENT_SERVICE_URL}/api/deep-identify/stream`
// and invokes onFrame once per parsed SSE frame, in arrival order (T070).
//
// Cancellation (T071): ctx is used directly for the outbound HTTP request,
// so when the caller's context is cancelled or times out (deep_identification_
// service.go's runJob cancels/times out jobCtx/timeoutCtx), the underlying
// HTTP round trip is aborted immediately by net/http and this method returns
// promptly with ctx.Err(). No partial evidence is returned to the caller in
// that case beyond whatever onFrame already persisted as historical events;
// the pipeline result itself is never produced, so runJob's status switch
// (which checks jobCtx.Err()/timeoutCtx.Err() before ever consulting a
// returned result) always settles a cancelled/timed-out job correctly
// regardless of what this method returns.
//
// EOF-without-terminal (T072): if the stream ends without a `synthesis` or
// `error` frame ever being observed, this returns
// ErrDeepStreamEndedWithoutTerminal so the caller can propagate a non-nil
// error, which runJob's existing switch already maps to
// failed:agent_unavailable.
func (p *AgentProxy) StreamDeepIdentification(ctx context.Context, req DeepIdentifyProxyRequest, onFrame func(frame DeepIdentifyFrame) error) error {
	logger := p.logger

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/api/deep-identify/stream", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	p.attachInternalCredential(httpReq)

	resp, err := p.streamClient.Do(httpReq)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if logger != nil {
			logger.Error("agent-proxy", "deep-identify stream request failed: %v", err)
		}
		return fmt.Errorf("agent service unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		errMsg := string(respBody)
		if len(errMsg) > 200 {
			errMsg = errMsg[:200] + "... (truncated)"
		}
		if logger != nil {
			logger.Error("agent-proxy", "deep-identify stream returned %d: %s", resp.StatusCode, errMsg)
		}
		return agentServiceHTTPError(resp.StatusCode, respBody)
	}

	sawTerminal := false
	scanner := bufio.NewScanner(resp.Body)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 4*1024*1024) // a synthesis frame's report/proposal can be sizeable

	for scanner.Scan() {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if strings.TrimSpace(data) == "" {
			continue
		}

		var probe struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal([]byte(data), &probe); err != nil {
			// Malformed frame: never silently promoted to a fabricated
			// result. Skip it and keep reading; EOF-without-terminal
			// handling below covers the case where nothing usable ever
			// arrives.
			if logger != nil {
				logger.Error("agent-proxy", "deep-identify stream: malformed frame skipped: %v", err)
			}
			continue
		}
		if deepIdentifyTerminalFrameTypes[probe.Type] {
			sawTerminal = true
		}

		if onFrame != nil {
			if err := onFrame(DeepIdentifyFrame{Type: probe.Type, Raw: json.RawMessage(data)}); err != nil {
				return err
			}
		}

		if sawTerminal {
			break
		}
	}

	if err := scanner.Err(); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf("stream read error: %w", err)
	}

	if ctx.Err() != nil {
		return ctx.Err()
	}

	if !sawTerminal {
		return ErrDeepStreamEndedWithoutTerminal
	}

	return nil
}
