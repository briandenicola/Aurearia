package services

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// AgentProxy forwards requests to the Python LangGraph agent service.
type AgentProxy struct {
	baseURL       string
	streamClient  *http.Client // No timeout — SSE streams can run long
	requestClient *http.Client // Short timeout for non-streaming requests
}

func NewAgentProxy(baseURL string) *AgentProxy {
	return &AgentProxy{
		baseURL:       strings.TrimRight(baseURL, "/"),
		streamClient:  &http.Client{Timeout: 0},
		requestClient: &http.Client{Timeout: 5 * time.Minute},
	}
}

// --- Request / response types matching the Python agent service ---

type LLMConfig struct {
	Provider   string `json:"provider"`
	APIKey     string `json:"api_key"`
	Model      string `json:"model"`
	OllamaURL  string `json:"ollama_url"`
	SearXNGURL string `json:"searxng_url"`
}

type UserContextProxy struct {
	UserID  uint   `json:"user_id"`
	ZipCode string `json:"zip_code"`
}

type ChatMessageProxy struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type AgentChatProxyRequest struct {
	LLM              LLMConfig          `json:"llm"`
	User             UserContextProxy   `json:"user"`
	Message          string             `json:"message"`
	History          []ChatMessageProxy `json:"history"`
	CoinSearchPrompt string             `json:"coin_search_prompt"`
	CoinShowsPrompt  string             `json:"coin_shows_prompt"`
	Portfolio        *PortfolioData     `json:"portfolio,omitempty"`
}

type PortfolioData struct {
	TotalCoins    int                  `json:"total_coins"`
	TotalValue    float64              `json:"total_value"`
	TotalInvested float64              `json:"total_invested"`
	Categories    map[string]int       `json:"categories"`
	Materials     map[string]int       `json:"materials"`
	Eras          []map[string]any     `json:"eras"`
	Rulers        []map[string]any     `json:"rulers"`
	TopCoins      []PortfolioCoinProxy `json:"top_coins"`
}

type PortfolioCoinProxy struct {
	Name          string  `json:"name"`
	Category      string  `json:"category"`
	Material      string  `json:"material"`
	Era           string  `json:"era"`
	Ruler         string  `json:"ruler"`
	Grade         string  `json:"grade"`
	PurchasePrice float64 `json:"purchase_price"`
	CurrentValue  float64 `json:"current_value"`
}

type PortfolioReviewProxyRequest struct {
	LLM             LLMConfig          `json:"llm"`
	User            UserContextProxy   `json:"user"`
	Portfolio       PortfolioData      `json:"portfolio"`
	Message         string             `json:"message"`
	History         []ChatMessageProxy `json:"history"`
	ValuationPrompt string             `json:"valuation_prompt"`
}

type CoinDataProxy struct {
	ID            int     `json:"id"`
	Name          string  `json:"name"`
	Ruler         string  `json:"ruler"`
	Era           string  `json:"era"`
	Denomination  string  `json:"denomination"`
	Material      string  `json:"material"`
	Category      string  `json:"category"`
	Grade         string  `json:"grade"`
	PurchasePrice float64 `json:"purchase_price"`
	CurrentValue  float64 `json:"current_value"`
	Notes         string  `json:"notes"`
}

type AnalyzeProxyRequest struct {
	LLM    LLMConfig     `json:"llm"`
	Coin   CoinDataProxy `json:"coin"`
	Images []string      `json:"images"`
	Side   string        `json:"side"`
	Prompt string        `json:"prompt"`
}

type AnalyzeProxyResponse struct {
	Analysis string `json:"analysis"`
}

// AvailabilityCheckProxyItem represents a single coin URL to check.
type AvailabilityCheckProxyItem struct {
	URL      string `json:"url"`
	CoinName string `json:"coin_name"`
}

// AvailabilityCheckProxyRequest is sent to the Python agent.
type AvailabilityCheckProxyRequest struct {
	LLM   LLMConfig                    `json:"llm"`
	Items []AvailabilityCheckProxyItem `json:"items"`
}

// AvailabilityVerdictProxy is a single verdict from the Python agent.
type AvailabilityVerdictProxy struct {
	URL        string `json:"url"`
	CoinName   string `json:"coin_name"`
	Status     string `json:"status"`
	Reason     string `json:"reason"`
	Confidence string `json:"confidence"`
}

// AvailabilityCheckProxyResponse is the response from the Python agent.
type AvailabilityCheckProxyResponse struct {
	Results []AvailabilityVerdictProxy `json:"results"`
}

// StreamChat POSTs to the Python agent's /api/search/coins endpoint and
// transparently proxies the SSE stream back to the caller.
func (p *AgentProxy) StreamChat(ctx context.Context, w http.ResponseWriter, req AgentChatProxyRequest) error {
	return p.proxySSE(ctx, w, "/api/search/coins", req)
}

// CollectPortfolioReviewPOSTs to /api/portfolio/review, reads the full SSE
// stream, and returns the final message text (from the "done" event).
func (p *AgentProxy) CollectPortfolioReview(ctx context.Context, req PortfolioReviewProxyRequest) (string, error) {
	return p.collectSSE(ctx, "/api/portfolio/review", req)
}

// AnalyzeCoin POSTs to /api/analyze and returns the analysis text.
func (p *AgentProxy) AnalyzeCoin(ctx context.Context, req AnalyzeProxyRequest) (string, error) {
	logger := AppLogger

	body, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("marshal analyze request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/api/analyze", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create analyze request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.requestClient.Do(httpReq)
	if err != nil {
		logger.Error("agent-proxy", "Analyze request failed: %v", err)
		return "", fmt.Errorf("agent service unreachable: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		errMsg := string(respBody)
		if len(errMsg) > 200 {
			errMsg = errMsg[:200] + "... (truncated)"
		}
		logger.Error("agent-proxy", "Analyze returned %d: %s", resp.StatusCode, errMsg)
		return "", fmt.Errorf("agent service returned HTTP %d", resp.StatusCode)
	}

	var result AnalyzeProxyResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("parse analyze response: %w", err)
	}
	return result.Analysis, nil
}

// CheckAvailability POSTs to the Python agent's /api/check-availability endpoint.
func (p *AgentProxy) CheckAvailability(ctx context.Context, req AvailabilityCheckProxyRequest) (*AvailabilityCheckProxyResponse, error) {
	logger := AppLogger

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal availability check request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/api/check-availability", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create availability check request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.requestClient.Do(httpReq)
	if err != nil {
		logger.Error("agent-proxy", "Availability check request failed: %v", err)
		return nil, fmt.Errorf("agent service unreachable: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		errMsg := string(respBody)
		if len(errMsg) > 200 {
			errMsg = errMsg[:200] + "... (truncated)"
		}
		logger.Error("agent-proxy", "Availability check returned %d: %s", resp.StatusCode, errMsg)
		return nil, fmt.Errorf("agent service returned HTTP %d", resp.StatusCode)
	}

	var result AvailabilityCheckProxyResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("parse availability check response: %w", err)
	}
	return &result, nil
}

// FetchLogsretrieves log entries from the Python agent's /logs endpoint
// and returns them as LogEntry slices compatible with the Go logger format.
func (p *AgentProxy) FetchLogs(ctx context.Context, limit int, level string) []LogEntry {
	url := fmt.Sprintf("%s/logs?limit=%d", p.baseURL, limit)
	if level != "" {
		url += "&level=" + level
	}

	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil
	}
	resp, err := (&http.Client{Timeout: 5 * time.Second}).Do(httpReq)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil
	}

	var result struct {
		Logs []LogEntry `json:"logs"`
	}
	body, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(body, &result); err != nil {
		return nil
	}

	// Tag each entry so the UI can distinguish agent vs api logs
	for i := range result.Logs {
		result.Logs[i].Message = "[agent] " + result.Logs[i].Message
	}
	return result.Logs
}

// SetLogLevel pushes a new log level to the Python agent service.
func (p *AgentProxy) SetLogLevel(ctx context.Context, level string) {
	payload := []byte(fmt.Sprintf(`{"level":"%s"}`, level))
	httpReq, err := http.NewRequestWithContext(ctx, "PUT", p.baseURL+"/log-level", bytes.NewReader(payload))
	if err != nil {
		return
	}
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := (&http.Client{Timeout: 5 * time.Second}).Do(httpReq)
	if err != nil {
		return
	}
	resp.Body.Close()
}

// proxySSE is the shared helper that posts JSON to the Python service and
// forwards the SSE byte stream line-by-line back to the Go response writer.
func (p *AgentProxy) proxySSE(ctx context.Context, w http.ResponseWriter, path string, payload any) error {
	logger := AppLogger

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.streamClient.Do(httpReq)
	if err != nil {
		logger.Error("agent-proxy", "SSE proxy request to %s failed: %v", path, err)
		return fmt.Errorf("agent service unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		// Truncate error body to avoid logging sensitive data (API keys in echoed requests)
		errMsg := string(respBody)
		if len(errMsg) > 200 {
			errMsg = errMsg[:200] + "... (truncated)"
		}
		logger.Error("agent-proxy", "SSE proxy %s returned %d: %s", path, resp.StatusCode, errMsg)
		return fmt.Errorf("agent service returned HTTP %d", resp.StatusCode)
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		return fmt.Errorf("response writer does not support flushing")
	}

	scanner := bufio.NewScanner(resp.Body)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		fmt.Fprintf(w, "%s\n", line)
		// Flush after every blank line (SSE event boundary) or data line
		if line == "" || strings.HasPrefix(line, "data:") {
			flusher.Flush()
		}
	}

	if err := scanner.Err(); err != nil {
		logger.Error("agent-proxy", "SSE scanner error on %s: %v", path, err)
		return fmt.Errorf("stream read error: %w", err)
	}

	return nil
}

// collectSSE posts to the Python service, reads the full SSE stream, and
// returns the final message from the "done" event. Used for non-streaming
// endpoints (like value estimation) that need a complete response.
func (p *AgentProxy) collectSSE(ctx context.Context, path string, payload any) (string, error) {
	logger := AppLogger

	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.streamClient.Do(httpReq)
	if err != nil {
		logger.Error("agent-proxy", "collectSSE request to %s failed: %v", path, err)
		return "", fmt.Errorf("agent service unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		errMsg := string(respBody)
		if len(errMsg) > 200 {
			errMsg = errMsg[:200] + "... (truncated)"
		}
		logger.Error("agent-proxy", "collectSSE %s returned %d: %s", path, resp.StatusCode, errMsg)
		return "", fmt.Errorf("agent service returned HTTP %d", resp.StatusCode)
	}

	// Read all SSE events and extract the "done" event's message
	var fullMessage string
	scanner := bufio.NewScanner(resp.Body)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		var event struct {
			Type    string `json:"type"`
			Message string `json:"message"`
			Text    string `json:"text"`
		}
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			continue
		}
		if event.Type == "done" && event.Message != "" {
			fullMessage = event.Message
		}
	}

	if err := scanner.Err(); err != nil {
		return fullMessage, fmt.Errorf("stream read error: %w", err)
	}

	return fullMessage, nil
}
