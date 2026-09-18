package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

// Feature 360's Go-to-Python DTOs are intentionally isolated from handlers and
// persistence. They mirror exploration-agent.openapi.yaml exactly.

const (
	ExplorationDecisionSchemaVersion = "aurearia.browser-exploration-decision/v1"
	ExplorationDecisionPath          = "/internal/browser-exploration/decide"
)

type ExplorationDecisionRequest struct {
	SchemaVersion  string                   `json:"schemaVersion"`
	RunID          string                   `json:"runId"`
	StepID         string                   `json:"stepId"`
	Provider       string                   `json:"provider"`
	Model          string                   `json:"model"`
	Workflow       string                   `json:"workflow"`
	Goal           string                   `json:"goal"`
	AllowedActions []string                 `json:"allowedActions"`
	AllowedRoutes  []string                 `json:"allowedRoutes"`
	Observations   []ExplorationObservation `json:"observations"`
}

type ExplorationObservation struct {
	EvidenceID string `json:"evidenceId"`
	Kind       string `json:"kind"`
	Route      string `json:"route"`
	Summary    string `json:"summary"`
}

type ExplorationDecisionResponse struct {
	SchemaVersion     string                   `json:"schemaVersion"`
	Action            string                   `json:"action"`
	Target            *ExplorationActionTarget `json:"target"`
	Rationale         string                   `json:"rationale"`
	SuspectedFindings []ExplorationModelTriage `json:"suspectedFindings"`
	Usage             ExplorationModelUsage    `json:"usage"`
}

type ExplorationActionTarget struct {
	Route          string `json:"route,omitempty"`
	Role           string `json:"role,omitempty"`
	AccessibleName string `json:"accessibleName,omitempty"`
	Label          string `json:"label,omitempty"`
	Value          string `json:"value,omitempty"`
	FixtureID      string `json:"fixtureId,omitempty"`
	Width          int    `json:"width,omitempty"`
	Height         int    `json:"height,omitempty"`
	Milliseconds   int    `json:"milliseconds,omitempty"`
}

type ExplorationModelTriage struct {
	GeneratedByModel  bool     `json:"generatedByModel"`
	Summary           string   `json:"summary"`
	SuggestedCategory string   `json:"suggestedCategory"`
	SuggestedSeverity string   `json:"suggestedSeverity"`
	Confidence        float64  `json:"confidence"`
	EvidenceIDs       []string `json:"evidenceIds"`
}

type ExplorationModelUsage struct {
	InputTokens  int `json:"inputTokens"`
	OutputTokens int `json:"outputTokens"`
}

var (
	ErrExplorationRateLimited         = errors.New("exploration provider rate limited")
	ErrExplorationInvalidOutput       = errors.New("exploration model output invalid")
	ErrExplorationProviderUnavailable = errors.New("exploration provider unavailable")
)

func explorationProviderHeaders(provider string) (map[string]string, error) {
	switch provider {
	case "anthropic":
		key := strings.TrimSpace(os.Getenv("AI_BROWSER_ANTHROPIC_API_KEY"))
		if key == "" && !strings.EqualFold(os.Getenv("AI_BROWSER_FAKE_MODEL"), "true") {
			return nil, fmt.Errorf("AI_BROWSER_ANTHROPIC_API_KEY is required")
		}
		return map[string]string{"X-AI-Browser-Anthropic-Api-Key": key}, nil
	case "ollama":
		url := strings.TrimSpace(os.Getenv("AI_BROWSER_OLLAMA_URL"))
		if url == "" && !strings.EqualFold(os.Getenv("AI_BROWSER_FAKE_MODEL"), "true") {
			return nil, fmt.Errorf("AI_BROWSER_OLLAMA_URL is required")
		}
		if url != "" && !strings.HasPrefix(url, "http://") {
			return nil, fmt.Errorf("AI_BROWSER_OLLAMA_URL must use http")
		}
		return map[string]string{"X-AI-Browser-Ollama-Url": url}, nil
	default:
		return nil, fmt.Errorf("unsupported exploration provider %q", provider)
	}
}

// DecideBrowserExploration forwards one bounded decision request. It reads only
// Feature 360's dedicated environment variables and propagates the caller's
// context deadline/cancellation to the agent request.
func (p *AgentProxy) DecideBrowserExploration(ctx context.Context, req ExplorationDecisionRequest) (*ExplorationDecisionResponse, error) {
	headers, err := explorationProviderHeaders(req.Provider)
	if err != nil {
		return nil, err
	}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal exploration request: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+ExplorationDecisionPath, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create exploration request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	p.attachInternalCredential(httpReq)
	for key, value := range headers {
		httpReq.Header.Set(key, value)
	}
	resp, err := p.requestClient.Do(httpReq)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, ctxErr
		}
		return nil, fmt.Errorf("%w: %v", ErrExplorationProviderUnavailable, err)
	}
	defer resp.Body.Close()
	responseBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("%w: read response", ErrExplorationProviderUnavailable)
	}
	if resp.StatusCode != http.StatusOK {
		switch resp.StatusCode {
		case http.StatusTooManyRequests:
			return nil, ErrExplorationRateLimited
		case http.StatusUnprocessableEntity, http.StatusBadRequest:
			return nil, ErrExplorationInvalidOutput
		default:
			return nil, ErrExplorationProviderUnavailable
		}
	}
	var result ExplorationDecisionResponse
	decoder := json.NewDecoder(bytes.NewReader(responseBody))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&result); err != nil {
		return nil, fmt.Errorf("%w: malformed response", ErrExplorationInvalidOutput)
	}
	if result.SchemaVersion != ExplorationDecisionSchemaVersion || result.Usage.InputTokens < 0 || result.Usage.OutputTokens < 0 {
		return nil, ErrExplorationInvalidOutput
	}
	return &result, nil
}
