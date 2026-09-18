package services

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/briandenicola/ancient-coins-api/models"
)

func loadCoinCopilotFixture[T any](t *testing.T, name string) (T, error) {
	t.Helper()
	var value T
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		return value, errors.New("resolve contract test path")
	}
	path := filepath.Join(filepath.Dir(currentFile), "..", "..", "agent", "tests", "fixtures", "coin_copilot", name)
	raw, err := os.ReadFile(path)
	if err != nil {
		return value, fmt.Errorf("read %s: %w", path, err)
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&value); err != nil {
		return value, err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			return value, errors.New("fixture contains multiple JSON values")
		}
		return value, err
	}
	return value, nil
}

func decodeCoinCopilotFixturePayload[T any](raw json.RawMessage) (T, error) {
	var value T
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&value); err != nil {
		return value, err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			return value, errors.New("payload contains multiple JSON values")
		}
		return value, err
	}
	return value, nil
}

func TestCoinCopilotSharedContractFixtures(t *testing.T) {
	request, err := loadCoinCopilotFixture[CopilotExecuteProxyRequest](t, "valid_execute_request.json")
	if err != nil {
		t.Fatalf("valid execute request rejected: %v", err)
	}
	if request.SchemaVersion != CoinCopilotSchemaVersion || request.RunID != "ccr_fixture" ||
		request.ExecutionID != "cce_fixture" || len(request.AllowedTools) != len(CoinCopilotAllowedTools) {
		t.Fatalf("unexpected execute fixture: %#v", request)
	}

	run := &models.CoinCopilotRun{
		ID: "ccr_fixture", ExecutionID: "cce_fixture", MaxIterations: 8, MaxToolCalls: 12,
	}
	for _, name := range []string{
		"valid_checkpoint_frame.json",
		"valid_tool_completed_frame.json",
		"valid_completed_frame.json",
	} {
		frame, err := loadCoinCopilotFixture[CopilotAgentFrame](t, name)
		if err != nil {
			t.Fatalf("%s failed strict decode: %v", name, err)
		}
		if err := ValidateCopilotFrame(frame, run); err != nil {
			t.Fatalf("%s failed frame validation: %v", name, err)
		}
		if frame.Type == "checkpoint" {
			state, err := decodeCoinCopilotFixturePayload[CopilotCheckpointState](frame.Payload)
			if err != nil {
				t.Fatalf("%s failed strict checkpoint decode: %v", name, err)
			}
			if err := ValidateCopilotCheckpoint(state, run); err != nil {
				t.Fatalf("%s failed checkpoint validation: %v", name, err)
			}
		}
	}
}

func TestCoinCopilotSharedContractFixturesRejectInvalidPayloads(t *testing.T) {
	if _, err := loadCoinCopilotFixture[CopilotExecuteProxyRequest](t, "invalid_execute_extra_field.json"); err == nil {
		t.Fatal("execute request with an extra field was accepted")
	}

	reasoning, err := loadCoinCopilotFixture[CopilotAgentFrame](t, "invalid_frame_reasoning.json")
	if err != nil {
		t.Fatalf("decode reasoning fixture: %v", err)
	}
	run := &models.CoinCopilotRun{
		ID: "ccr_fixture", ExecutionID: "cce_fixture", MaxIterations: 8, MaxToolCalls: 12,
	}
	if !errors.Is(ValidateCopilotFrame(reasoning, run), ErrInvalidCopilotFrame) {
		t.Fatal("frame containing reasoning was accepted")
	}

	duplicate, err := loadCoinCopilotFixture[CopilotAgentFrame](t, "invalid_checkpoint_duplicate_call_id.json")
	if err != nil {
		t.Fatalf("decode duplicate-call fixture: %v", err)
	}
	if err := ValidateCopilotFrame(duplicate, run); err != nil {
		t.Fatalf("duplicate-call fixture failed outer frame validation: %v", err)
	}
	state, err := decodeCoinCopilotFixturePayload[CopilotCheckpointState](duplicate.Payload)
	if err != nil {
		t.Fatalf("decode duplicate-call checkpoint: %v", err)
	}
	if !errors.Is(ValidateCopilotCheckpoint(state, run), ErrInvalidCopilotFrame) {
		t.Fatal("checkpoint containing duplicate tool-call ids was accepted")
	}
}

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
