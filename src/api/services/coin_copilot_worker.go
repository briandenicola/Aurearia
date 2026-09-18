package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
)

const copilotWorkerPollInterval = 100 * time.Millisecond

var copilotHeartbeatInterval = 15 * time.Second

func (s *CoinCopilotService) StartWorkers(ctx context.Context) {
	settings := s.settingsSvc.GetCoinCopilotSettings()
	_, _ = s.repo.RecoverStale(time.Now().UTC().Add(-45*time.Second), settings.ResumeWindow)
	for i := 0; i < settings.WorkerCount; i++ {
		go s.workerLoop(ctx, fmt.Sprintf("copilot-%d", i+1))
	}
	go s.janitorLoop(ctx)
}

func (s *CoinCopilotService) workerLoop(ctx context.Context, workerID string) {
	ticker := time.NewTicker(copilotWorkerPollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.wake:
		case <-ticker.C:
		}
		executionID := newCopilotID("cce_")
		run, claimed, err := s.repo.ClaimNextQueuedRun(workerID, executionID)
		if err != nil || !claimed {
			continue
		}
		s.runExecution(ctx, run)
	}
}

func (s *CoinCopilotService) runExecution(parent context.Context, run *models.CoinCopilotRun) {
	timeout := time.Duration(run.HardTimeoutSeconds) * time.Second
	ctx, cancel := context.WithTimeout(parent, timeout)
	s.registerExecution(run.ExecutionID, cancel)
	defer func() {
		cancel()
		s.unregisterExecution(run.ExecutionID)
		s.clearToolCalls(run.ExecutionID)
		s.tokenSvc.RevokeCopilotExecution(run.ExecutionID)
	}()
	go func() {
		ticker := time.NewTicker(copilotHeartbeatInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				_ = s.repo.Heartbeat(run.ID, run.ExecutionID)
			}
		}
	}()

	limits := CopilotLimitsProxy{
		MaxIterations: run.MaxIterations, MaxToolCalls: run.MaxToolCalls, MaxConcurrentTools: run.MaxConcurrentTools,
		HardTimeoutSeconds: run.HardTimeoutSeconds, MaxPersistedToolResultBytes: run.MaxPersistedToolResultBytes,
	}
	startPayload, _ := json.Marshal(map[string]any{
		"status": "running", "executionId": run.ExecutionID, "attempt": run.ExecutionAttempt, "limits": map[string]any{
			"maxIterations": limits.MaxIterations, "maxToolCalls": limits.MaxToolCalls,
			"maxConcurrentTools": limits.MaxConcurrentTools, "hardTimeoutSeconds": limits.HardTimeoutSeconds,
			"maxPersistedToolResultBytes": limits.MaxPersistedToolResultBytes,
		},
	})
	if event, err := s.repo.AppendEvent(run.ID, run.UserID, run.ExecutionID, models.CopilotEventRunStarted, string(startPayload)); err == nil {
		s.broker.Publish(event.RunID)
	} else {
		s.failExecution(run, "internal", "Coin Copilot could not start.")
		return
	}

	llm, err := s.settingsSvc.ResolveLLMConfig()
	if err != nil {
		s.failExecution(run, "model_tool_calling_unsupported", "The configured model is unavailable.")
		return
	}
	tokenTTL := coinCopilotExecutionTokenTTL(ctx, time.Now().UTC())
	token, err := s.tokenSvc.MintForCopilotExecution(run.UserID, run.ID, run.ExecutionID, CoinCopilotAllowedTools, tokenTTL)
	if err != nil {
		s.failExecution(run, "internal", "Coin Copilot could not authorize this execution.")
		return
	}
	request, err := s.executionRequest(run, llm, token, limits)
	if err != nil {
		s.failExecution(run, "internal", "Coin Copilot could not restore its checkpoint.")
		return
	}
	seenFrames := map[string]bool{}
	var mu sync.Mutex
	streamErr := s.proxy.StreamCoinCopilot(ctx, request, func(frame CopilotAgentFrame) error {
		mu.Lock()
		defer mu.Unlock()
		if seenFrames[frame.FrameID] {
			return ErrDuplicateFrame
		}
		seenFrames[frame.FrameID] = true
		fresh, err := s.repo.GetCurrentExecution(run.ID, run.ExecutionID, run.UserID)
		if err != nil {
			return ErrCopilotStateConflict
		}
		if err := ValidateCopilotFrame(frame, fresh); err != nil {
			return err
		}
		return s.applyFrame(fresh, frame)
	})
	if streamErr == nil {
		return
	}
	fresh, err := s.repo.GetRun(run.ID, run.UserID)
	if err != nil || models.IsCopilotRunTerminal(fresh.Status) || fresh.Status == models.CopilotRunPaused {
		return
	}
	if fresh.Status == models.CopilotRunCancelRequested {
		s.settleCancelled(fresh)
		return
	}
	if errors.Is(ctx.Err(), context.Canceled) {
		return
	}
	code := "agent_unavailable"
	message := "Coin Copilot execution was interrupted."
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		code = "time_limit_exceeded"
		message = "Coin Copilot reached its time limit."
	}
	if errors.Is(streamErr, ErrInvalidCopilotFrame) || errors.Is(streamErr, ErrDuplicateFrame) {
		code = "invalid_agent_frame"
		message = "Coin Copilot returned an invalid execution frame."
	}
	s.failExecution(fresh, code, message)
}

func coinCopilotExecutionTokenTTL(ctx context.Context, now time.Time) time.Duration {
	remaining := time.Duration(0)
	if deadline, ok := ctx.Deadline(); ok {
		remaining = deadline.Sub(now)
	}
	return min(remaining+coinCopilotExecutionTokenBuffer, coinCopilotExecutionTokenMaxTTL)
}

func (s *CoinCopilotService) executionRequest(run *models.CoinCopilotRun, llm LLMConfig, token string, limits CopilotLimitsProxy) (CopilotExecuteProxyRequest, error) {
	state := CopilotCheckpointState{
		SchemaVersion: 1,
		Messages:      []CopilotMessage{{Role: "user", Content: run.Goal}},
		Plan:          []CopilotPlanItem{}, CompletedTools: []CopilotCompletedTool{}, NextAction: "continue",
	}
	version := int64(0)
	if run.CheckpointVersion > 0 {
		checkpoint, err := s.repo.GetLatestCheckpoint(run.ID, run.UserID)
		if err != nil {
			return CopilotExecuteProxyRequest{}, err
		}
		if err := json.Unmarshal([]byte(checkpoint.StateJSON), &state); err != nil {
			return CopilotExecuteProxyRequest{}, err
		}
		version = checkpoint.Version
	} else {
		history, err := s.repo.ListSettledRunsForHistory(run.ThreadID, run.ID, run.UserID, CoinCopilotHistoryRunLimit)
		if err != nil {
			return CopilotExecuteProxyRequest{}, err
		}
		state.Messages = append(copilotPublicHistory(history), state.Messages...)
	}
	var appContext map[string]any
	if run.AppContextJSON != "" {
		if err := json.Unmarshal([]byte(run.AppContextJSON), &appContext); err != nil {
			return CopilotExecuteProxyRequest{}, err
		}
	}
	return CopilotExecuteProxyRequest{
		SchemaVersion: 1, ThreadID: run.ThreadID, RunID: run.ID, ExecutionID: run.ExecutionID,
		Goal: run.Goal, Messages: state.Messages,
		Checkpoint: CopilotCheckpointProxy{
			Version: version, Plan: state.Plan, CompletedTools: state.CompletedTools,
			PendingClarification: state.PendingClarification, NextAction: state.NextAction, Counters: state.Counters,
		},
		AppContext: appContext, LLM: llm, Limits: limits, ToolsBaseURL: s.toolsBaseURL,
		ExecutionToken: token, AllowedTools: append([]string(nil), CoinCopilotAllowedTools...),
	}, nil
}

func copilotPublicHistory(runs []models.CoinCopilotRun) []CopilotMessage {
	groups := make([][]CopilotMessage, 0, len(runs))
	remaining := CoinCopilotHistoryMaxBytes
	for _, run := range runs {
		if remaining <= 0 {
			break
		}
		goal := SanitizeCopilotText(run.Goal, min(CoinCopilotHistoryMessageMaxBytes, remaining))
		if goal == "" {
			continue
		}
		group := []CopilotMessage{{Role: "user", Content: goal}}
		remaining -= len(goal)
		if run.FinalAnswer != "" && remaining > 0 {
			answer := SanitizeCopilotText(run.FinalAnswer, min(CoinCopilotHistoryMessageMaxBytes, remaining))
			if answer != "" {
				group = append(group, CopilotMessage{Role: "assistant", Content: answer})
				remaining -= len(answer)
			}
		}
		groups = append(groups, group)
	}
	messages := make([]CopilotMessage, 0, len(groups)*2)
	for i := len(groups) - 1; i >= 0; i-- {
		messages = append(messages, groups[i]...)
	}
	return messages
}

func (s *CoinCopilotService) applyFrame(run *models.CoinCopilotRun, frame CopilotAgentFrame) error {
	switch frame.Type {
	case "checkpoint":
		var state CopilotCheckpointState
		if err := json.Unmarshal(frame.Payload, &state); err != nil {
			return ErrInvalidCopilotFrame
		}
		for i := range state.CompletedTools {
			if isCoinCopilotSpecialistTool(state.CompletedTools[i].ToolName) {
				if _, err := DecodeCopilotSpecialistResult(state.CompletedTools[i].Result, state.CompletedTools[i].ToolName); err != nil {
					return err
				}
			}
			bounded, originalBytes, truncated, digest, err := SanitizeCopilotJSON(state.CompletedTools[i].Result, run.MaxPersistedToolResultBytes)
			if err != nil {
				return err
			}
			state.CompletedTools[i].Result = bounded
			state.CompletedTools[i].OriginalBytes = originalBytes
			state.CompletedTools[i].PersistedBytes = len(bounded)
			state.CompletedTools[i].Truncated = truncated
			state.CompletedTools[i].ResultDigest = digest
		}
		if ValidateCopilotCheckpoint(state, run) != nil {
			return ErrInvalidCopilotFrame
		}
		stateJSON, _ := json.Marshal(state)
		if _, err := s.repo.CommitCheckpoint(run.ID, run.UserID, run.ExecutionID, string(stateJSON), CopilotCheckpointDigest(string(stateJSON))); err != nil {
			return err
		}
		return s.repo.UpdateUsage(run.ID, run.ExecutionID, models.CoinCopilotRun{
			IterationCount: state.Counters.Iterations, ToolCallCount: state.Counters.ToolCalls,
			InputTokens: state.Counters.InputTokens, OutputTokens: state.Counters.OutputTokens,
		})
	case "usage":
		var usage CopilotUsage
		if err := json.Unmarshal(frame.Payload, &usage); err != nil || usage.Iterations < 0 || usage.ToolCalls < 0 ||
			usage.InputTokens < 0 || usage.OutputTokens < 0 ||
			usage.Iterations > run.MaxIterations || usage.ToolCalls > run.MaxToolCalls {
			return ErrInvalidCopilotFrame
		}
		return s.repo.UpdateUsage(run.ID, run.ExecutionID, models.CoinCopilotRun{
			IterationCount: usage.Iterations, ToolCallCount: usage.ToolCalls, InputTokens: usage.InputTokens,
			OutputTokens: usage.OutputTokens,
		})
	case "plan_updated":
		var payload struct {
			Plan []CopilotPlanItem `json:"plan"`
		}
		if err := json.Unmarshal(frame.Payload, &payload); err != nil || len(payload.Plan) > CoinCopilotMaxPlanItems {
			return ErrInvalidCopilotFrame
		}
		for _, item := range payload.Plan {
			if item.ID == "" || item.Title == "" || len(item.Title) > CoinCopilotMaxPlanTitle {
				return ErrInvalidCopilotFrame
			}
			switch item.Status {
			case "pending", "in_progress", "completed", "skipped", "failed":
			default:
				return ErrInvalidCopilotFrame
			}
		}
		return s.appendFrameEvent(run, models.CopilotEventPlanUpdated, frame.Payload)
	case "tool_started":
		var payload struct {
			ToolCallID string `json:"tool_call_id"`
			ToolName   string `json:"tool_name"`
			StepID     string `json:"step_id"`
		}
		if err := json.Unmarshal(frame.Payload, &payload); err != nil || payload.ToolCallID == "" || payload.StepID == "" || !IsCoinCopilotToolAllowed(payload.ToolName) {
			return ErrInvalidCopilotFrame
		}
		public, _ := json.Marshal(map[string]string{"toolCallId": payload.ToolCallID, "toolName": payload.ToolName, "stepId": payload.StepID})
		return s.appendFrameEvent(run, models.CopilotEventToolStarted, public)
	case "tool_completed":
		var payload struct {
			ToolCallID string          `json:"tool_call_id"`
			ToolName   string          `json:"tool_name"`
			StepID     string          `json:"step_id"`
			Status     string          `json:"status"`
			DurationMS int64           `json:"duration_ms"`
			Summary    string          `json:"result_summary"`
			Result     json.RawMessage `json:"result"`
		}
		if err := json.Unmarshal(frame.Payload, &payload); err != nil || payload.ToolCallID == "" || payload.StepID == "" || !IsCoinCopilotToolAllowed(payload.ToolName) || payload.DurationMS < 0 {
			return ErrInvalidCopilotFrame
		}
		switch payload.Status {
		case "succeeded", "failed", "cancelled", "rejected":
		default:
			return ErrInvalidCopilotFrame
		}
		_, _, truncated, _, err := SanitizeCopilotJSON(payload.Result, run.MaxPersistedToolResultBytes)
		if err != nil {
			return err
		}
		publicPayload := map[string]any{
			"toolCallId": payload.ToolCallID, "toolName": payload.ToolName, "stepId": payload.StepID,
			"status": payload.Status, "durationMs": payload.DurationMS,
			"resultSummary": SanitizeCopilotText(payload.Summary, 300), "truncated": truncated,
		}
		if isCoinCopilotSpecialistTool(payload.ToolName) && payload.Status == "succeeded" {
			result, err := DecodeCopilotSpecialistResult(payload.Result, payload.ToolName)
			if err != nil {
				return err
			}
			specialistResult, err := ProjectCopilotSpecialistResult(result, payload.ToolName)
			if err != nil {
				return err
			}
			publicPayload["specialistResult"] = specialistResult
		}
		public, _ := json.Marshal(publicPayload)
		return s.appendFrameEvent(run, models.CopilotEventToolCompleted, public)
	case "clarification_required":
		var payload struct {
			Question  string   `json:"question"`
			InputType string   `json:"input_type"`
			Choices   []string `json:"choices"`
		}
		if err := json.Unmarshal(frame.Payload, &payload); err != nil || payload.Question == "" ||
			len(payload.Question) > CoinCopilotMaxClarification || len(payload.Choices) > 10 {
			return ErrInvalidCopilotFrame
		}
		switch payload.InputType {
		case "text", "single_choice", "boolean":
		default:
			return ErrInvalidCopilotFrame
		}
		if err := s.appendFrameEvent(run, models.CopilotEventClarificationRequired, []byte(mustJSON(map[string]any{
			"question":  SanitizeCopilotText(payload.Question, CoinCopilotMaxClarification),
			"inputType": payload.InputType, "choices": payload.Choices, "checkpointVersion": run.CheckpointVersion,
		}))); err != nil {
			return err
		}
		now := time.Now().UTC()
		deadline := now.Add(s.settingsSvc.GetCoinCopilotSettings().ResumeWindow)
		won, _, err := s.repo.TransitionWithEvent(run.ID, run.UserID, run.ExecutionID,
			[]models.CopilotRunStatus{models.CopilotRunRunning}, models.CopilotRunPaused,
			map[string]interface{}{"paused_at": now, "resume_deadline": deadline, "worker_id": "", "heartbeat_at": nil},
			models.CopilotEventRunPaused, mustJSON(map[string]any{
				"reason": "clarification_required", "checkpointVersion": run.CheckpointVersion, "resumeDeadline": deadline,
			}))
		if err == nil && won {
			s.broker.Publish(run.ID)
			return nil
		}
		return ErrCopilotStateConflict
	case "completed":
		var payload struct {
			Answer string       `json:"answer"`
			Usage  CopilotUsage `json:"usage"`
		}
		if err := json.Unmarshal(frame.Payload, &payload); err != nil || payload.Answer == "" || len(payload.Answer) > CoinCopilotMaxMessageBytes {
			return ErrInvalidCopilotFrame
		}
		if payload.Usage.Iterations < 0 || payload.Usage.ToolCalls < 0 ||
			payload.Usage.InputTokens < 0 || payload.Usage.OutputTokens < 0 ||
			payload.Usage.Iterations > run.MaxIterations || payload.Usage.ToolCalls > run.MaxToolCalls {
			return ErrInvalidCopilotFrame
		}
		answer := SanitizeCopilotText(payload.Answer, CoinCopilotMaxMessageBytes)
		won, _, err := s.repo.TransitionWithEvent(run.ID, run.UserID, run.ExecutionID,
			[]models.CopilotRunStatus{models.CopilotRunRunning}, models.CopilotRunCompleted,
			map[string]interface{}{
				"final_answer": answer, "iteration_count": payload.Usage.Iterations, "tool_call_count": payload.Usage.ToolCalls,
				"input_tokens": payload.Usage.InputTokens, "output_tokens": payload.Usage.OutputTokens,
				"worker_id": "", "heartbeat_at": nil,
			}, models.CopilotEventRunCompleted, mustJSON(map[string]any{"answer": answer, "usage": publicUsage(payload.Usage)}))
		if err == nil && won {
			s.broker.Publish(run.ID)
			return nil
		}
		return ErrCopilotStateConflict
	case "failed":
		var payload struct {
			Code      string       `json:"code"`
			Message   string       `json:"message"`
			Retryable bool         `json:"retryable"`
			Usage     CopilotUsage `json:"usage"`
		}
		if err := json.Unmarshal(frame.Payload, &payload); err != nil || !allowedCopilotFailureCode(payload.Code) {
			return ErrInvalidCopilotFrame
		}
		if payload.Usage.Iterations < 0 || payload.Usage.ToolCalls < 0 ||
			payload.Usage.InputTokens < 0 || payload.Usage.OutputTokens < 0 ||
			payload.Usage.Iterations > run.MaxIterations || payload.Usage.ToolCalls > run.MaxToolCalls {
			return ErrInvalidCopilotFrame
		}
		won, _, err := s.repo.TransitionWithEvent(run.ID, run.UserID, run.ExecutionID,
			[]models.CopilotRunStatus{models.CopilotRunRunning}, models.CopilotRunFailed,
			map[string]interface{}{
				"failure_code": payload.Code, "failure_message": SanitizeCopilotText(payload.Message, 300),
				"iteration_count": payload.Usage.Iterations, "tool_call_count": payload.Usage.ToolCalls,
				"input_tokens": payload.Usage.InputTokens, "output_tokens": payload.Usage.OutputTokens,
				"worker_id": "", "heartbeat_at": nil,
			}, models.CopilotEventRunFailed, mustJSON(map[string]any{
				"code": payload.Code, "message": SanitizeCopilotText(payload.Message, 300),
				"retryable": payload.Retryable, "usage": publicUsage(payload.Usage),
			}))
		if err == nil && won {
			s.broker.Publish(run.ID)
			return nil
		}
		return ErrCopilotStateConflict
	}
	return ErrInvalidCopilotFrame
}

func (s *CoinCopilotService) appendFrameEvent(run *models.CoinCopilotRun, eventType models.CopilotEventType, payload []byte) error {
	sanitized, _, _, _, err := SanitizeCopilotJSON(payload, CoinCopilotMaxEventBytes)
	if err != nil {
		return err
	}
	event, err := s.repo.AppendEvent(run.ID, run.UserID, run.ExecutionID, eventType, string(sanitized))
	if err == nil {
		s.broker.Publish(event.RunID)
	}
	return err
}

func (s *CoinCopilotService) failExecution(run *models.CoinCopilotRun, code, message string) {
	won, _, err := s.repo.TransitionWithEvent(run.ID, run.UserID, run.ExecutionID,
		[]models.CopilotRunStatus{models.CopilotRunRunning}, models.CopilotRunFailed,
		map[string]interface{}{"failure_code": code, "failure_message": message, "worker_id": "", "heartbeat_at": nil},
		models.CopilotEventRunFailed, mustJSON(map[string]any{"code": code, "message": message, "retryable": code != "internal", "usage": publicUsage(CopilotUsage{})}))
	if err == nil && won {
		s.broker.Publish(run.ID)
	}
}

func (s *CoinCopilotService) settleCancelled(run *models.CoinCopilotRun) {
	won, _, err := s.repo.TransitionWithEvent(run.ID, run.UserID, run.ExecutionID,
		[]models.CopilotRunStatus{models.CopilotRunCancelRequested, models.CopilotRunRunning}, models.CopilotRunCancelled,
		map[string]interface{}{"worker_id": "", "heartbeat_at": nil},
		models.CopilotEventRunCancelled, `{"reason":"owner_cancelled"}`)
	if err == nil && won {
		s.broker.Publish(run.ID)
	}
}

func (s *CoinCopilotService) janitorLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.RecoverAndPrune(); err != nil && s.logger != nil {
				s.logger.Error("coin-copilot", "recovery/janitor failed: %v", err)
			}
		}
	}
}

func allowedCopilotFailureCode(code string) bool {
	switch code {
	case "agent_unavailable", "execution_lost", "invalid_agent_frame", "invalid_tool_call",
		"iteration_limit_exceeded", "tool_limit_exceeded", "time_limit_exceeded",
		"model_tool_calling_unsupported", "resume_window_expired", "internal":
		return true
	default:
		return false
	}
}

func mustJSON(value any) string {
	encoded, _ := json.Marshal(value)
	return string(encoded)
}

func publicUsage(usage CopilotUsage) map[string]any {
	return map[string]any{
		"iterations": usage.Iterations, "toolCalls": usage.ToolCalls, "inputTokens": usage.InputTokens,
		"outputTokens": usage.OutputTokens,
	}
}
