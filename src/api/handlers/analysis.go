package handlers

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/briandenicola/ancient-coins-api/models"
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

func (h *AnalysisHandler) resolveLLMConfig() (services.LLMConfig, string) {
	cfg, err := h.settingsSvc.ResolveLLMConfig()
	if err != nil {
		return services.LLMConfig{}, err.Error()
	}
	return cfg, ""
}

// Analyze runs AI analysis on a coin's images using Ollama.
//
//	@Summary		Analyze coin images
//	@Description	Sends coin images to Ollama for AI-powered analysis. Can analyze a specific side (obverse/reverse) or all images.
//	@Tags			Analysis
//	@Produce		json
//	@Param			id		path		int		true	"Coin ID"
//	@Param			side	query		string	false	"Analyze a specific side only"	Enums(obverse, reverse)
//	@Success		200		{object}	AnalysisResponse
//	@Failure		400		{object}	ErrorResponse
//	@Failure		401		{object}	ErrorResponse
//	@Failure		404		{object}	ErrorResponse
//	@Failure		500		{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/coins/{id}/analyze [post]
func (h *AnalysisHandler) Analyze(c *gin.Context) {
	logger := h.logger
	userID := c.GetUint("userId")
	coinID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		logger.Warn("analysis", "Invalid coin ID param: %s", c.Param("id"))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid coin ID"})
		return
	}

	logger.Info("analysis", "Starting analysis for coin %d (user %d)", coinID, userID)

	coin, err := h.repo.FindCoinWithImages(uint(coinID), userID)
	if err != nil {
		logger.Warn("analysis", "Coin %d not found for user %d: %v", coinID, userID, err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Coin not found"})
		return
	}

	logger.Debug("analysis", "Coin loaded: %s, images: %d", coin.Name, len(coin.Images))

	if len(coin.Images) == 0 {
		logger.Warn("analysis", "Coin %d has no images to analyze", coinID)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Coin has no images to analyze"})
		return
	}

	// Filter by side if requested
	side := c.Query("side")
	var analyzeImages []models.CoinImage

	if side == "obverse" || side == "reverse" {
		for _, img := range coin.Images {
			if string(img.ImageType) == side {
				analyzeImages = append(analyzeImages, img)
			}
		}
		if len(analyzeImages) == 0 {
			logger.Warn("analysis", "Coin %d has no %s image", coinID, side)
			c.JSON(http.StatusBadRequest, gin.H{"error": "No " + side + " image found"})
			return
		}
		logger.Info("analysis", "Analyzing %s side only for coin %d", side, coinID)
	} else {
		analyzeImages = coin.Images
	}

	// Pick the prompt based on which side is being analyzed
	var customPrompt string
	switch side {
	case "obverse":
		customPrompt = h.settingsSvc.GetSetting(services.SettingObversePrompt)
	case "reverse":
		customPrompt = h.settingsSvc.GetSetting(services.SettingReversePrompt)
	default:
		customPrompt = h.settingsSvc.GetSetting(services.SettingObversePrompt)
	}

	// Read base64 images from files
	var base64Images []string
	for _, img := range analyzeImages {
		p := filepath.Join("uploads", img.FilePath)
		data, err := os.ReadFile(p)
		if err != nil {
			logger.Warn("analysis", "Failed to read image %s: %v", p, err)
			continue
		}
		base64Images = append(base64Images, base64.StdEncoding.EncodeToString(data))
	}

	if len(base64Images) == 0 {
		logger.Warn("analysis", "No valid images could be read for coin %d", coinID)
		c.JSON(http.StatusBadRequest, gin.H{"error": "No valid images found"})
		return
	}

	// Resolve LLM provider from explicit setting
	llmCfg, errMsg := h.resolveLLMConfig()
	if errMsg != "" {
		respondError(c, http.StatusBadRequest, "Unable to configure analysis provider", fmt.Errorf("%s", errMsg))
		return
	}

	var purchasePrice, currentValue float64
	if coin.PurchasePrice != nil {
		purchasePrice = *coin.PurchasePrice
	}
	if coin.CurrentValue != nil {
		currentValue = *coin.CurrentValue
	}

	proxyReq := services.AnalyzeProxyRequest{
		LLM: llmCfg,
		Coin: services.CoinDataProxy{
			ID:            int(coin.ID),
			Name:          coin.Name,
			Ruler:         coin.Ruler,
			Era:           coin.Era,
			Denomination:  coin.Denomination,
			Material:      string(coin.Material),
			Category:      string(coin.Category),
			Grade:         coin.Grade,
			PurchasePrice: purchasePrice,
			CurrentValue:  currentValue,
			Notes:         coin.Notes,
		},
		Images: base64Images,
		Side:   side,
		Prompt: customPrompt,
	}

	logger.Info("analysis", "Sending %d image(s) to agent service for coin %d", len(base64Images), coinID)
	analysis, err := h.proxy.AnalyzeCoin(c.Request.Context(), proxyReq)
	if err != nil {
		logger.Error("analysis", "AI analysis failed for coin %d: %v", coinID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "AI analysis failed. Check agent service configuration."})
		return
	}

	logger.Info("analysis", "Analysis complete for coin %d (%d chars)", coinID, len(analysis))
	logger.Trace("analysis", "Analysis result: %s", analysis)

	// Store in the appropriate field based on side
	switch side {
	case "obverse":
		h.repo.UpdateCoinField(coin, "obverse_analysis", analysis)
		coin.ObverseAnalysis = analysis
	case "reverse":
		h.repo.UpdateCoinField(coin, "reverse_analysis", analysis)
		coin.ReverseAnalysis = analysis
	default:
		h.repo.UpdateCoinField(coin, "ai_analysis", analysis)
		coin.AIAnalysis = analysis
	}

	c.JSON(http.StatusOK, gin.H{
		"analysis": analysis,
		"side":     side,
		"coin":     coin,
	})
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
