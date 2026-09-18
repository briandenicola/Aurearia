package services

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/briandenicola/ancient-coins-api/models"
)

func TestCoinCopilotContractAllFramesAndForbiddenReasoning(t *testing.T) {
	run := &models.CoinCopilotRun{ID: "ccr_1", ExecutionID: "cce_1"}
	for _, frameType := range []string{"plan_updated", "tool_started", "tool_completed", "checkpoint", "clarification_required", "completed", "failed", "usage"} {
		frame := CopilotAgentFrame{SchemaVersion: 1, RunID: run.ID, ExecutionID: run.ExecutionID, FrameID: "frm_" + frameType, Type: frameType, Payload: json.RawMessage(`{}`)}
		if err := ValidateCopilotFrame(frame, run); err != nil {
			t.Fatalf("%s rejected: %v", frameType, err)
		}
	}
	frame := CopilotAgentFrame{SchemaVersion: 1, RunID: run.ID, ExecutionID: run.ExecutionID, FrameID: "frm_bad", Type: "plan_updated", Payload: json.RawMessage(`{"reasoning":"secret"}`)}
	if !errors.Is(ValidateCopilotFrame(frame, run), ErrInvalidCopilotFrame) {
		t.Fatal("forbidden reasoning field was accepted")
	}
	frame.RunID = "ccr_other"
	if !errors.Is(ValidateCopilotFrame(frame, run), ErrInvalidCopilotFrame) {
		t.Fatal("mismatched run was accepted")
	}
}

func TestSanitizeCopilotJSONTruncatesAndRedactsSecrets(t *testing.T) {
	raw := []byte(`{"note":"Bearer abcdefghijklmnopqrstuvwxyz","api_key":"sk-secret","payload":"` + strings.Repeat("x", 5000) + `"}`)
	bounded, original, truncated, digest, err := SanitizeCopilotJSON(raw, 512)
	if err != nil {
		t.Fatal(err)
	}
	if original != len(raw) || !truncated || len(bounded) > 512 || len(digest) != 64 {
		t.Fatalf("unexpected bounds original=%d truncated=%v len=%d digest=%q", original, truncated, len(bounded), digest)
	}
	if bytes.Contains(bounded, []byte("sk-secret")) || bytes.Contains(bounded, []byte("abcdefghijklmnopqrstuvwxyz")) {
		t.Fatalf("secret leaked: %s", bounded)
	}
}

func TestValidateCopilotCheckpointRejectsDuplicateAndOverBudgetTools(t *testing.T) {
	run := &models.CoinCopilotRun{MaxIterations: 8, MaxToolCalls: 1}
	state := CopilotCheckpointState{
		SchemaVersion: 1, Messages: []CopilotMessage{{Role: "user", Content: "x"}},
		Plan: []CopilotPlanItem{}, NextAction: "continue",
		Counters: CopilotUsage{ToolCalls: 2},
	}
	if !errors.Is(ValidateCopilotCheckpoint(state, run), ErrInvalidCopilotFrame) {
		t.Fatal("over-budget checkpoint was accepted")
	}
	state.Counters.ToolCalls = 1
	state.CompletedTools = []CopilotCompletedTool{
		{ToolCallID: "call_1", ToolName: "get_coin"},
		{ToolCallID: "call_1", ToolName: "get_coin"},
	}
	if !errors.Is(ValidateCopilotCheckpoint(state, run), ErrInvalidCopilotFrame) {
		t.Fatal("duplicate tool call was accepted")
	}
}

func TestCopilotUsageAndLimitsDoNotExposeEstimatedCost(t *testing.T) {
	usage, err := json.Marshal(CopilotUsage{Iterations: 1, ToolCalls: 2, InputTokens: 30, OutputTokens: 10})
	if err != nil {
		t.Fatal(err)
	}
	limits, err := json.Marshal(CopilotLimitsProxy{
		MaxIterations: 8, MaxToolCalls: 12, MaxConcurrentTools: 1,
		HardTimeoutSeconds: 120, MaxPersistedToolResultBytes: 32768,
	})
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(usage, []byte("cost")) || bytes.Contains(limits, []byte("cost")) {
		t.Fatalf("cost field leaked from contract: usage=%s limits=%s", usage, limits)
	}
}
