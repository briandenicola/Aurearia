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
	"time"
)

var ErrCopilotStreamEndedWithoutTerminal = errors.New("coin copilot stream ended without terminal frame")

type coinCopilotCapabilityRequest struct {
	LLM LLMConfig `json:"llm"`
}

type coinCopilotCapabilityResponse struct {
	Supported bool `json:"supported"`
}

func (p *AgentProxy) SupportsCoinCopilot(llm LLMConfig) bool {
	if p.baseURL == "" {
		return false
	}
	body, err := json.Marshal(coinCopilotCapabilityRequest{LLM: llm})
	if err != nil {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/api/copilot/capability", bytes.NewReader(body))
	if err != nil {
		return false
	}
	req.Header.Set("Content-Type", "application/json")
	p.attachInternalCredential(req)
	resp, err := p.requestClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false
	}
	var payload coinCopilotCapabilityResponse
	decoder := json.NewDecoder(io.LimitReader(resp.Body, 64*1024))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		return false
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return false
	}
	return payload.Supported
}

func (p *AgentProxy) StreamCoinCopilot(ctx context.Context, request CopilotExecuteProxyRequest, onFrame func(CopilotAgentFrame) error) error {
	body, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("marshal copilot request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/api/copilot/execute", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create copilot request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	p.attachInternalCredential(req)
	resp, err := p.streamClient.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf("coin copilot agent unavailable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		responseBody, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
		if p.logger != nil {
			p.logger.Error("coin-copilot", "agent execute returned status=%d body=%s", resp.StatusCode, sanitizeAgentErrorBodyForLog(responseBody, 400))
		}
		return agentServiceHTTPError(resp.StatusCode, responseBody)
	}

	terminal := false
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		line := scanner.Text()
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "" {
			continue
		}
		var frame CopilotAgentFrame
		decoder := json.NewDecoder(strings.NewReader(data))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&frame); err != nil {
			return ErrInvalidCopilotFrame
		}
		if onFrame != nil {
			if err := onFrame(frame); err != nil {
				return err
			}
		}
		if frame.Type == "completed" || frame.Type == "failed" || frame.Type == "clarification_required" {
			terminal = true
			break
		}
	}
	if err := scanner.Err(); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf("coin copilot stream read: %w", err)
	}
	if !terminal {
		return ErrCopilotStreamEndedWithoutTerminal
	}
	return nil
}
