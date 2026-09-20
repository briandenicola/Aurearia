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

func loadDeepAnalysisHandoffFixture(t *testing.T, name string) map[string]json.RawMessage {
	t.Helper()
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve handoff fixture path")
	}
	path := filepath.Join(
		filepath.Dir(currentFile),
		"..", "..", "..", "specs", "362-coin-copilot-attribution",
		"contracts", "fixtures", name,
	)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var fixture map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatalf("decode %s: %v", name, err)
	}
	return fixture
}

func TestCoinCopilotSharedContractFixtures(t *testing.T) {
	request, err := loadCoinCopilotFixture[CopilotExecuteProxyRequest](t, "valid_execute_request.json")
	if err != nil {
		t.Fatalf("valid execute request rejected: %v", err)
	}
	if request.SchemaVersion != CoinCopilotSchemaVersion || request.RunID != "ccr_fixture" ||
		request.ExecutionID != "cce_fixture" || len(request.AllowedTools) != 6 {
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
	bounded, truncated, err := SanitizeCopilotJSON(raw, 512)
	if err != nil {
		t.Fatal(err)
	}

	if !truncated || len(bounded) > 512 {
		t.Fatalf("unexpected bounds truncated=%v len=%d", truncated, len(bounded))
	}
	if bytes.Contains(bounded, []byte("sk-secret")) || bytes.Contains(bounded, []byte("abcdefghijklmnopqrstuvwxyz")) {
		t.Fatalf("secret leaked: %s", bounded)
	}
}

func TestSanitizeCopilotJSONPreservesPythonDecimalLexemes(t *testing.T) {
	raw := []byte(`{"current_bid":99.50,"estimate":250.0}`)
	bounded, truncated, err := SanitizeCopilotJSON(raw, 1024)
	if err != nil {
		t.Fatal(err)
	}
	const expected = `{"current_bid":99.50,"estimate":250.0}`
	if string(bounded) != expected || truncated {
		t.Fatalf("unexpected canonical decimal result: %s", bounded)
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

func TestValidateCopilotSpecialistQueryBounds(t *testing.T) {
	if err := ValidateCopilotSpecialistQuery(CopilotSpecialistQuery{Query: "Domitian denarius"}); err != nil {
		t.Fatalf("default-limit query rejected: %v", err)
	}
	for _, query := range []CopilotSpecialistQuery{
		{},
		{Query: strings.Repeat("x", 501)},
		{Query: "coin", Limit: -1},
		{Query: "coin", Limit: 11},
	} {
		if !errors.Is(ValidateCopilotSpecialistQuery(query), ErrInvalidCopilotFrame) {
			t.Fatalf("invalid query accepted: %#v", query)
		}
	}
}

func canonicalPaddingEnvelope(t *testing.T, size int) []byte {
	t.Helper()
	const empty = `{"padding":""}`
	if size < len(empty) {
		t.Fatalf("requested envelope size %d is too small", size)
	}
	raw, err := json.Marshal(map[string]string{"padding": strings.Repeat("x", size-len(empty))})
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) != size {
		t.Fatalf("canonical envelope bytes=%d want=%d", len(raw), size)
	}
	return raw
}

func TestDeepAnalysisHandoffCanonicalFixturesAndStrictRequestDrift(t *testing.T) {
	valid := loadDeepAnalysisHandoffFixture(t, "deep-analysis-handoff-valid.json")
	var requests map[string]json.RawMessage
	if err := json.Unmarshal(valid["requests"], &requests); err != nil {
		t.Fatal(err)
	}
	for _, operation := range []string{"request", "status", "rerun"} {
		decoded, err := DecodeDeepAnalysisHandoffRequest(requests[operation])
		if err != nil {
			t.Fatalf("%s request rejected: %v", operation, err)
		}
		if decoded.Operation != operation {
			t.Fatalf("operation=%q want=%q", decoded.Operation, operation)
		}
	}
	var results map[string]json.RawMessage
	if err := json.Unmarshal(valid["results"], &results); err != nil {
		t.Fatal(err)
	}
	result, err := DecodeDeepAnalysisHandoffResult(
		results["status_complete"],
		DeepAnalysisHandoffMaxPublicEventBytes,
	)
	if err != nil {
		t.Fatalf("valid outcome-discriminated result rejected: %v", err)
	}
	if result.Outcome != "status" || result.Job == nil || result.Job.Status != "completed" {
		t.Fatalf("unexpected status result: %#v", result)
	}

	invalid := loadDeepAnalysisHandoffFixture(t, "deep-analysis-handoff-invalid.json")
	var requestCases []struct {
		Name    string          `json:"name"`
		Payload json.RawMessage `json:"payload"`
	}
	if err := json.Unmarshal(invalid["request_cases"], &requestCases); err != nil {
		t.Fatal(err)
	}
	for _, testCase := range requestCases {
		if len(testCase.Payload) == 0 || testCase.Name == "duplicate_call" {
			continue
		}
		t.Run(testCase.Name, func(t *testing.T) {
			if _, err := DecodeDeepAnalysisHandoffRequest(testCase.Payload); err == nil {
				t.Fatal("invalid handoff request was accepted")
			}
		})
	}
	var resultCases []struct {
		Name    string          `json:"name"`
		Payload json.RawMessage `json:"payload"`
	}
	if err := json.Unmarshal(invalid["result_cases"], &resultCases); err != nil {
		t.Fatal(err)
	}
	for _, testCase := range resultCases {
		if testCase.Name != "result_level_status_discriminant" {
			continue
		}
		if _, err := DecodeDeepAnalysisHandoffResult(
			testCase.Payload,
			DeepAnalysisHandoffMaxPublicEventBytes,
		); err == nil {
			t.Fatal("result-level status discriminant was accepted")
		}
	}
}

func TestDeepAnalysisHandoffRequestAndPublicEventHaveIndependent64KiBLimits(t *testing.T) {
	atLimit := canonicalPaddingEnvelope(t, DeepAnalysisHandoffMaxRequestBytes)
	overLimit := canonicalPaddingEnvelope(t, DeepAnalysisHandoffMaxRequestBytes+1)
	if err := ValidateDeepAnalysisHandoffRequestEnvelope(atLimit); err != nil {
		t.Fatalf("65,536-byte request rejected: %v", err)
	}
	if err := ValidateDeepAnalysisHandoffRequestEnvelope(overLimit); err == nil {
		t.Fatal("65,537-byte request accepted")
	}

	atEventLimit := canonicalPaddingEnvelope(t, DeepAnalysisHandoffMaxPublicEventBytes)
	overEventLimit := canonicalPaddingEnvelope(t, DeepAnalysisHandoffMaxPublicEventBytes+1)
	if err := ValidateDeepAnalysisHandoffPublicEventEnvelope(atEventLimit); err != nil {
		t.Fatalf("65,536-byte public event rejected: %v", err)
	}
	if err := ValidateDeepAnalysisHandoffPublicEventEnvelope(overEventLimit); err == nil {
		t.Fatal("65,537-byte public event accepted")
	}
}

func TestPublicCopilotToolResultCamelizesAndDropsInternals(t *testing.T) {
	raw := []byte(`{"schema_version":1,"capability":"market_search","outcome":"complete",` +
		`"provider_attempts":[{"provider":"configured_dealer_search","status":"success"}],` +
		`"items":[{"kind":"dealer_listing","source_url":"https://www.vcoins.com/x/1",` +
		`"dealer_name":"VCoins","listed_price":130,"image_url":"https://www.vcoins.com/i.jpg",` +
		`"candidate_references":[{"catalog":"RIC","number":"80"}]}],` +
		`"truncation":{"truncated":false,"omitted_items":0}}`)

	result, err := PublicCopilotToolResult(raw)
	if err != nil {
		t.Fatal(err)
	}
	if _, present := result["providerAttempts"]; present {
		t.Fatal("internal provider attempts reached the browser payload")
	}
	if _, present := result["schemaVersion"]; present {
		t.Fatal("internal schema version reached the browser payload")
	}
	items, _ := result["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("expected one item, got %d", len(items))
	}
	item, _ := items[0].(map[string]any)
	for _, key := range []string{"sourceUrl", "dealerName", "listedPrice", "imageUrl", "candidateReferences"} {
		if _, present := item[key]; !present {
			t.Fatalf("expected camelCase key %q in %#v", key, item)
		}
	}
	truncation, _ := result["truncation"].(map[string]any)
	if _, present := truncation["omittedItems"]; !present {
		t.Fatalf("expected camelCase truncation, got %#v", truncation)
	}
}
