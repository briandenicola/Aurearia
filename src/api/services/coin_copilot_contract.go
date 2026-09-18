package services

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/briandenicola/ancient-coins-api/models"
)

const (
	CoinCopilotSchemaVersion          = 1
	CoinCopilotMaxEventBytes          = 64 * 1024
	CoinCopilotMaxPromptBytes         = 4000
	CoinCopilotMaxMessages            = 50
	CoinCopilotMaxMessageBytes        = 100000
	CoinCopilotMaxPlanItems           = 12
	CoinCopilotMaxPlanTitle           = 200
	CoinCopilotMaxClarification       = 500
	CoinCopilotHistoryRunLimit        = 10
	CoinCopilotHistoryMaxBytes        = 20000
	CoinCopilotHistoryMessageMaxBytes = 4000
)

var (
	ErrInvalidCopilotFrame = errors.New("invalid coin copilot frame")
	ErrDuplicateFrame      = errors.New("duplicate coin copilot frame")
)

var CoinCopilotAllowedTools = []string{
	"search_my_collection",
	"get_coin",
	"collection_summary",
	"top_coins_by_value",
	"portfolio_review",
	"gap_analysis",
}

var copilotCallbackTools = map[string]bool{
	"search_my_collection": true,
	"get_coin":             true,
	"collection_summary":   true,
	"top_coins_by_value":   true,
}

func IsCoinCopilotToolAllowed(tool string) bool {
	for _, allowed := range CoinCopilotAllowedTools {
		if tool == allowed {
			return true
		}
	}
	return false
}

func IsCoinCopilotCallbackTool(tool string) bool { return copilotCallbackTools[tool] }

type CopilotMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type CopilotPlanItem struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Status string `json:"status"`
}

type CopilotCompletedTool struct {
	ToolCallID     string          `json:"tool_call_id"`
	ToolName       string          `json:"tool_name"`
	ResultDigest   string          `json:"result_digest"`
	Result         json.RawMessage `json:"result"`
	OriginalBytes  int             `json:"original_bytes"`
	PersistedBytes int             `json:"persisted_bytes"`
	Truncated      bool            `json:"truncated"`
}

type CopilotClarification struct {
	Question  string   `json:"question"`
	InputType string   `json:"input_type"`
	Choices   []string `json:"choices"`
}

type CopilotUsage struct {
	Iterations   int   `json:"iterations"`
	ToolCalls    int   `json:"tool_calls"`
	InputTokens  int64 `json:"input_tokens"`
	OutputTokens int64 `json:"output_tokens"`
}

type CopilotCheckpointState struct {
	SchemaVersion        int                    `json:"schema_version"`
	Messages             []CopilotMessage       `json:"messages"`
	Plan                 []CopilotPlanItem      `json:"plan"`
	CompletedTools       []CopilotCompletedTool `json:"completed_tools"`
	PendingClarification *CopilotClarification  `json:"pending_clarification"`
	NextAction           string                 `json:"next_action"`
	Counters             CopilotUsage           `json:"counters"`
}

type CopilotLimitsProxy struct {
	MaxIterations               int `json:"max_iterations"`
	MaxToolCalls                int `json:"max_tool_calls"`
	MaxConcurrentTools          int `json:"max_concurrent_tools"`
	HardTimeoutSeconds          int `json:"hard_timeout_seconds"`
	MaxPersistedToolResultBytes int `json:"max_persisted_tool_result_bytes"`
}

type CopilotCheckpointProxy struct {
	Version              int64                  `json:"version"`
	Plan                 []CopilotPlanItem      `json:"plan"`
	CompletedTools       []CopilotCompletedTool `json:"completed_tools"`
	PendingClarification *CopilotClarification  `json:"pending_clarification"`
	NextAction           string                 `json:"next_action"`
	Counters             CopilotUsage           `json:"counters"`
}

type CopilotExecuteProxyRequest struct {
	SchemaVersion  int                    `json:"schema_version"`
	ThreadID       string                 `json:"thread_id"`
	RunID          string                 `json:"run_id"`
	ExecutionID    string                 `json:"execution_id"`
	Goal           string                 `json:"goal"`
	Messages       []CopilotMessage       `json:"messages"`
	Checkpoint     CopilotCheckpointProxy `json:"checkpoint"`
	AppContext     map[string]any         `json:"app_context"`
	LLM            LLMConfig              `json:"llm"`
	Limits         CopilotLimitsProxy     `json:"limits"`
	ToolsBaseURL   string                 `json:"tools_base_url"`
	ExecutionToken string                 `json:"execution_token"`
	AllowedTools   []string               `json:"allowed_tools"`
}

type CopilotAgentFrame struct {
	SchemaVersion int             `json:"schema_version"`
	RunID         string          `json:"run_id"`
	ExecutionID   string          `json:"execution_id"`
	FrameID       string          `json:"frame_id"`
	Type          string          `json:"type"`
	Payload       json.RawMessage `json:"payload"`
}

var forbiddenCopilotField = regexp.MustCompile(`(?i)"(?:reasoning|thought|scratchpad|rawPrompt|raw_prompt|rawToolInput|raw_tool_input|rawToolOutput|raw_tool_output|chain_of_thought)"\s*:`)
var tokenShapedValue = regexp.MustCompile(`(?i)(bearer\s+[A-Za-z0-9._~+/\-=]{12,}|(?:api[_-]?key|token|secret|password)\s*[:=]\s*["']?[A-Za-z0-9._~+/\-=]{8,})`)

func ValidateCopilotFrame(frame CopilotAgentFrame, run *models.CoinCopilotRun) error {
	if frame.SchemaVersion != CoinCopilotSchemaVersion || frame.RunID != run.ID ||
		frame.ExecutionID != run.ExecutionID || frame.FrameID == "" {
		return ErrInvalidCopilotFrame
	}
	switch frame.Type {
	case "plan_updated", "tool_started", "tool_completed", "checkpoint",
		"clarification_required", "completed", "failed", "usage":
	default:
		return ErrInvalidCopilotFrame
	}
	if len(frame.Payload) == 0 || len(frame.Payload) > CoinCopilotMaxEventBytes || !json.Valid(frame.Payload) ||
		forbiddenCopilotField.Match(frame.Payload) {
		return ErrInvalidCopilotFrame
	}
	return nil
}

func SanitizeCopilotJSON(raw []byte, maxBytes int) ([]byte, int, bool, string, error) {
	if !json.Valid(raw) || forbiddenCopilotField.Match(raw) {
		return nil, 0, false, "", ErrInvalidCopilotFrame
	}
	originalBytes := len(raw)
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, 0, false, "", ErrInvalidCopilotFrame
	}
	value = sanitizeCopilotValue(value)
	sanitized, err := json.Marshal(value)
	if err != nil {
		return nil, 0, false, "", err
	}
	digest := sha256.Sum256(sanitized)
	digestText := hex.EncodeToString(digest[:])
	if maxBytes <= 0 || len(sanitized) <= maxBytes {
		return sanitized, originalBytes, false, digestText, nil
	}
	summary := map[string]any{
		"truncated":      true,
		"original_bytes": originalBytes,
		"digest":         digestText,
		"summary":        "Tool result exceeded the persisted-result limit.",
	}
	bounded, err := json.Marshal(summary)
	if err != nil {
		return nil, 0, false, "", err
	}
	if len(bounded) > maxBytes {
		return nil, 0, false, "", fmt.Errorf("%w: persisted result limit too small", ErrInvalidCopilotFrame)
	}
	return bounded, originalBytes, true, digestText, nil
}

func sanitizeCopilotValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			lower := strings.ToLower(key)
			if strings.Contains(lower, "token") || strings.Contains(lower, "secret") ||
				strings.Contains(lower, "password") || strings.Contains(lower, "api_key") ||
				strings.Contains(lower, "apikey") {
				typed[key] = "[REDACTED]"
				continue
			}
			typed[key] = sanitizeCopilotValue(child)
		}
		return typed
	case []any:
		for i, child := range typed {
			typed[i] = sanitizeCopilotValue(child)
		}
		return typed
	case string:
		return tokenShapedValue.ReplaceAllString(typed, "[REDACTED]")
	}
	return value
}

func SanitizeCopilotText(value string, max int) string {
	value = tokenShapedValue.ReplaceAllString(value, "[REDACTED]")
	if max > 0 && len(value) > max {
		return value[:max]
	}
	return value
}

func CopilotCheckpointDigest(stateJSON string) string {
	sum := sha256.Sum256([]byte(stateJSON))
	return hex.EncodeToString(sum[:])
}

func ValidateCopilotCheckpoint(state CopilotCheckpointState, run *models.CoinCopilotRun) error {
	if state.SchemaVersion != 1 || len(state.Messages) > CoinCopilotMaxMessages || len(state.Plan) > CoinCopilotMaxPlanItems {
		return ErrInvalidCopilotFrame
	}
	total := 0
	for _, message := range state.Messages {
		if (message.Role != "user" && message.Role != "assistant") || message.Content == "" {
			return ErrInvalidCopilotFrame
		}
		total += len(message.Content)
	}
	if total > CoinCopilotMaxMessageBytes {
		return ErrInvalidCopilotFrame
	}
	for _, item := range state.Plan {
		if item.ID == "" || item.Title == "" || len(item.Title) > CoinCopilotMaxPlanTitle {
			return ErrInvalidCopilotFrame
		}
		switch item.Status {
		case "pending", "in_progress", "completed", "skipped", "failed":
		default:
			return ErrInvalidCopilotFrame
		}
	}
	if state.PendingClarification != nil {
		if len(state.PendingClarification.Question) == 0 || len(state.PendingClarification.Question) > CoinCopilotMaxClarification ||
			len(state.PendingClarification.Choices) > 10 {
			return ErrInvalidCopilotFrame
		}
		switch state.PendingClarification.InputType {
		case "text", "single_choice", "boolean":
		default:
			return ErrInvalidCopilotFrame
		}
	}
	switch state.NextAction {
	case "continue", "await_clarification", "finish":
	default:
		return ErrInvalidCopilotFrame
	}
	if state.Counters.Iterations < 0 || state.Counters.ToolCalls < 0 ||
		state.Counters.InputTokens < 0 || state.Counters.OutputTokens < 0 ||
		state.Counters.Iterations > run.MaxIterations || state.Counters.ToolCalls > run.MaxToolCalls {
		return ErrInvalidCopilotFrame
	}
	seen := map[string]bool{}
	for _, tool := range state.CompletedTools {
		if tool.ToolCallID == "" || !IsCoinCopilotToolAllowed(tool.ToolName) || seen[tool.ToolCallID] {
			return ErrInvalidCopilotFrame
		}
		seen[tool.ToolCallID] = true
	}
	return nil
}
