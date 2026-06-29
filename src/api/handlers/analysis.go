package handlers

import (
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
)

type AnalysisHandler struct {
	repo        *repository.AnalysisRepository
	proxy       *services.AgentProxy
	settingsSvc *services.SettingsService
	logger      *services.Logger
}

func NewAnalysisHandler(repo *repository.AnalysisRepository, proxy *services.AgentProxy, settingsSvc *services.SettingsService, logger *services.Logger) *AnalysisHandler {
	return &AnalysisHandler{repo: repo, proxy: proxy, settingsSvc: settingsSvc, logger: logger}
}

// DeleteAnalysis clears obverse or reverse analysis for a coin.
//
//	@Summary		Delete coin analysis
//	@Description	Clears the obverse or reverse AI analysis text for a specific coin.
//	@Tags			Analysis
//	@Produce		json
//	@Param			id		path		int		true	"Coin ID"
//	@Param			side	query		string	true	"Which side's analysis to clear"	Enums(obverse, reverse)
//	@Success		200		{object}	DeleteAnalysisResponse
//	@Failure		400		{object}	ErrorResponse
//	@Failure		401		{object}	ErrorResponse
//	@Failure		404		{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/coins/{id}/analyze [delete]
func (h *AnalysisHandler) DeleteAnalysis(c *gin.Context) {
	logger := h.logger
	userID := c.GetUint("userId")
	coinID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid coin ID"})
		return
	}

	side := c.Query("side")
	if side != "obverse" && side != "reverse" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "side query param must be 'obverse' or 'reverse'"})
		return
	}

	coin, err := h.repo.FindCoinWithImages(uint(coinID), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Coin not found"})
		return
	}

	// Whitelist column names to prevent SQL injection
	columnMap := map[string]string{
		"obverse": "obverse_analysis",
		"reverse": "reverse_analysis",
	}
	column, ok := columnMap[side]
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid side value"})
		return
	}
	if err := h.repo.UpdateCoinField(coin, column, ""); err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to clear analysis", err)
		return
	}
	logger.Info("analysis", "Cleared %s analysis for coin %d", side, coinID)

	// Reload to return updated coin
	coin, err = h.repo.ReloadCoinWithImages(uint(coinID))
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to reload coin", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"coin": coin})
}

// ExtractText accepts an image upload and returns extracted text via Ollama.
//
//	@Summary		Extract text from image
//	@Description	Uploads an image and uses Ollama to extract visible text (e.g., coin inscriptions).
//	@Tags			Analysis
//	@Accept			multipart/form-data
//	@Produce		json
//	@Param			image	formData	file	true	"Image file"
//	@Success		200		{object}	ExtractTextResponse
//	@Failure		400		{object}	ErrorResponse
//	@Failure		401		{object}	ErrorResponse
//	@Failure		500		{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/extract-text [post]
func (h *AnalysisHandler) ExtractText(c *gin.Context) {
	logger := h.logger
	logger.Info("extract-text", "Text extraction request received")

	file, err := c.FormFile("image")
	if err != nil {
		logger.Warn("extract-text", "No image file in request: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "No image file provided"})
		return
	}

	logger.Debug("extract-text", "File: %s, size: %d bytes", file.Filename, file.Size)

	f, err := file.Open()
	if err != nil {
		logger.Error("extract-text", "Failed to open uploaded file: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read image"})
		return
	}
	defer f.Close()

	imageData, err := io.ReadAll(f)
	if err != nil {
		logger.Error("extract-text", "Failed to read image data: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read image data"})
		return
	}

	ollamaURL := h.settingsSvc.GetSetting(services.SettingOllamaURL)
	ollamaModel := h.settingsSvc.GetSetting(services.SettingOllamaModel)
	ollamaTimeout, _ := strconv.Atoi(h.settingsSvc.GetSetting(services.SettingOllamaTimeout))
	customPrompt := h.settingsSvc.GetSetting(services.SettingTextExtractionPrompt)

	logger.Debug("extract-text", "Sending to Ollama: URL=%s, Model=%s, Timeout=%ds", ollamaURL, ollamaModel, ollamaTimeout)
	logger.Debug("extract-text", "Custom extraction prompt: [%s]", customPrompt)

	ollamaSvc := services.NewOllamaService(ollamaURL, ollamaTimeout, h.logger)
	text, err := ollamaSvc.ExtractTextFromImage(imageData, ollamaModel, customPrompt)
	if err != nil {
		logger.Error("extract-text", "Text extraction failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Text extraction failed. Check Ollama configuration."})
		return
	}

	logger.Info("extract-text", "Extraction complete (%d chars)", len(text))
	logger.Trace("extract-text", "Extracted text: %s", text)

	c.JSON(http.StatusOK, gin.H{"text": text})
}

// OllamaStatus checks Ollama connectivity and model availability.
//
//	@Summary		Check Ollama status
//	@Description	Returns whether the configured Ollama instance is reachable and the model is available.
//	@Tags			Analysis
//	@Produce		json
//	@Success		200	{object}	OllamaStatusResponse
//	@Failure		401	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/ollama-status [get]
func (h *AnalysisHandler) OllamaStatus(c *gin.Context) {
	logger := h.logger
	logger.Debug("ollama-status", "Checking Ollama status")

	ollamaURL := h.settingsSvc.GetSetting(services.SettingOllamaURL)
	ollamaModel := h.settingsSvc.GetSetting(services.SettingOllamaModel)

	ollamaSvc := services.NewOllamaService(ollamaURL, 10, h.logger)
	available, message := ollamaSvc.CheckModel(ollamaModel)

	logger.Info("ollama-status", "Ollama available=%v, model=%s, message=%s", available, ollamaModel, message)

	c.JSON(http.StatusOK, gin.H{
		"available": available,
		"model":     ollamaModel,
		"url":       ollamaURL,
		"message":   message,
	})
}

// AIStatus checks availability of the currently configured AI provider.
//
//	@Summary		Check AI provider status
//	@Description	Returns whether the configured AI provider (Anthropic or Ollama) is usable for coin analysis.
//	@Tags			Analysis
//	@Produce		json
//	@Success		200	{object}	AIStatusResponse
//	@Failure		401	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/ai-status [get]
func (h *AnalysisHandler) AIStatus(c *gin.Context) {
	logger := h.logger
	provider := h.settingsSvc.GetSetting(services.SettingAIProvider)

	switch provider {
	case "anthropic":
		model := h.settingsSvc.GetSetting(services.SettingAnthropicModel)
		apiKey := h.settingsSvc.GetSetting(services.SettingAnthropicAPIKey)
		if apiKey == "" {
			logger.Info("ai-status", "Anthropic selected but API key is missing")
			c.JSON(http.StatusOK, gin.H{
				"available": false,
				"provider":  "anthropic",
				"model":     model,
				"message":   "Anthropic API key is not configured. Set it in Admin → AI Configuration.",
			})
			return
		}
		logger.Info("ai-status", "Anthropic configured, model=%s", model)
		c.JSON(http.StatusOK, gin.H{
			"available": true,
			"provider":  "anthropic",
			"model":     model,
			"message":   fmt.Sprintf("Anthropic provider configured (model: %s)", model),
		})
	case "ollama":
		ollamaURL := h.settingsSvc.GetSetting(services.SettingOllamaURL)
		ollamaModel := h.settingsSvc.GetSetting(services.SettingOllamaModel)
		ollamaSvc := services.NewOllamaService(ollamaURL, 10, h.logger)
		available, message := ollamaSvc.CheckModel(ollamaModel)
		logger.Info("ai-status", "Ollama available=%v, model=%s, message=%s", available, ollamaModel, message)
		c.JSON(http.StatusOK, gin.H{
			"available": available,
			"provider":  "ollama",
			"model":     ollamaModel,
			"message":   message,
		})
	default:
		logger.Info("ai-status", "No AI provider configured")
		c.JSON(http.StatusOK, gin.H{
			"available": false,
			"provider":  "",
			"model":     "",
			"message":   "AI provider not configured. Choose Anthropic or Ollama in Admin → AI Configuration.",
		})
	}
}
