package services

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
)

type OllamaService struct {
	BaseURL string
	Client  *http.Client
}

type ollamaRequest struct {
	Model  string    `json:"model"`
	Prompt string    `json:"prompt"`
	Images []string  `json:"images"`
	Stream bool      `json:"stream"`
}

type ollamaResponse struct {
	Response string `json:"response"`
	Done     bool   `json:"done"`
}

func NewOllamaService(baseURL string, timeoutSeconds int) *OllamaService {
	if timeoutSeconds <= 0 {
		timeoutSeconds = 300
	}
	return &OllamaService{
		BaseURL: strings.TrimRight(baseURL, "/"),
		Client: &http.Client{
			Timeout: time.Duration(timeoutSeconds) * time.Second,
		},
	}
}

// ExtractTextFromImage sends an image to Ollama and asks it to extract all visible text
func (s *OllamaService) ExtractTextFromImage(imageData []byte, model string, customPrompt string) (string, error) {
	logger := AppLogger
	if model == "" {
		model = "llava"
	}

	logger.Debug("ollama", "ExtractText: model=%s, image_size=%d bytes", model, len(imageData))

	base64Image := base64.StdEncoding.EncodeToString(imageData)

	logger.Debug("ollama", "ExtractText prompt: %s", customPrompt)

	reqBody := ollamaRequest{
		Model:  model,
		Prompt: customPrompt,
		Images: []string{base64Image},
		Stream: false,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	logger.Trace("ollama", "ExtractText request payload: %d bytes", len(jsonBody))
	logger.Debug("ollama", "ExtractText calling %s/api/generate", s.BaseURL)

	resp, err := s.Client.Post(
		s.BaseURL+"/api/generate",
		"application/json",
		bytes.NewReader(jsonBody),
	)
	if err != nil {
		logger.Error("ollama", "ExtractText HTTP request failed: %v", err)
		return "", fmt.Errorf("failed to call Ollama: %w", err)
	}
	defer resp.Body.Close()

	logger.Debug("ollama", "ExtractText response status: %d", resp.StatusCode)

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		logger.Error("ollama", "ExtractText error response: %s", string(body))
		return "", fmt.Errorf("Ollama returned status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	logger.Trace("ollama", "ExtractText response body: %d bytes", len(body))

	var result ollamaResponse
	if err := json.Unmarshal(body, &result); err != nil {
		logger.Error("ollama", "ExtractText failed to parse response: %v", err)
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	logger.Trace("ollama", "ExtractText result: %s", result.Response)
	return result.Response, nil
}

// CheckModel calls the Ollama "show model" API to verify the model is available
func (s *OllamaService) CheckModel(model string) (bool, string) {
	logger := AppLogger
	if model == "" {
		model = "llava"
	}

	logger.Debug("ollama", "Checking model availability: %s at %s", model, s.BaseURL)

	reqBody, _ := json.Marshal(map[string]string{"name": model})

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Post(
		s.BaseURL+"/api/show",
		"application/json",
		bytes.NewReader(reqBody),
	)
	if err != nil {
		msg := fmt.Sprintf("Cannot connect to Ollama at %s: %v", s.BaseURL, err)
		logger.Warn("ollama", "%s", msg)
		return false, msg
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		msg := fmt.Sprintf("Model '%s' not available (status %d): %s", model, resp.StatusCode, string(body))
		logger.Warn("ollama", "%s", msg)
		return false, msg
	}

	var showResp map[string]interface{}
	body, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(body, &showResp); err != nil {
		msg := fmt.Sprintf("Invalid response from Ollama: %v", err)
		logger.Warn("ollama", "%s", msg)
		return false, msg
	}

	logger.Info("ollama", "Model '%s' is available", model)
	return true, fmt.Sprintf("Model '%s' is available and ready", model)
}

func buildCoinContext(coin models.Coin) string {
	var sb strings.Builder
	if coin.Name != "" {
		sb.WriteString(fmt.Sprintf("The coin is cataloged as: %s\n", coin.Name))
	}
	if coin.Category != "" {
		sb.WriteString(fmt.Sprintf("Category: %s\n", coin.Category))
	}
	if coin.Denomination != "" {
		sb.WriteString(fmt.Sprintf("Denomination: %s\n", coin.Denomination))
	}
	if coin.Ruler != "" {
		sb.WriteString(fmt.Sprintf("Ruler: %s\n", coin.Ruler))
	}
	return sb.String()
}

