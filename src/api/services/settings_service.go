package services

import (
	"fmt"

	"github.com/briandenicola/ancient-coins-api/repository"
)

const (
	SettingAIProvider                         = "AIProvider"
	SettingOllamaURL                          = "OllamaURL"
	SettingOllamaModel                        = "OllamaModel"
	SettingObversePrompt                      = "ObversePrompt"
	SettingReversePrompt                      = "ReversePrompt"
	SettingTextExtractionPrompt               = "TextExtractionPrompt"
	SettingOllamaTimeout                      = "OllamaTimeout"
	SettingLogLevel                           = "LogLevel"
	SettingPublicAppURL                       = "PublicAppURL"
	SettingNumistaAPIKey                      = "NumistaAPIKey"
	SettingAnthropicAPIKey                    = "AnthropicAPIKey"
	SettingAnthropicModel                     = "AnthropicModel"
	SettingCoinSearchPrompt                   = "CoinSearchPrompt"
	SettingCoinShowsPrompt                    = "CoinShowsPrompt"
	SettingValuationPrompt                    = "ValuationPrompt"
	SettingSearXNGURL                         = "SearXNGURL"
	SettingPushoverAppToken                   = "PushoverAppToken"
	SettingWishlistCheckEnabled               = "WishlistCheckEnabled"
	SettingWishlistCheckInterval              = "WishlistCheckInterval"
	SettingWishlistCheckStartTime             = "WishlistCheckStartTime"
	SettingWishlistSearchAlertsCheckEnabled   = "WishlistSearchAlertsCheckEnabled"
	SettingWishlistSearchAlertsCheckStartTime = "WishlistSearchAlertsCheckStartTime"
	SettingValuationCheckEnabled              = "ValuationCheckEnabled"
	SettingValuationCheckInterval             = "ValuationCheckIntervalDays"
	SettingValuationCheckStartTime            = "ValuationCheckStartTime"
	SettingValuationMaxCoins                  = "ValuationMaxCoins"
	SettingAuctionEndingCheckEnabled          = "AuctionEndingCheckEnabled"
	SettingAuctionEndingCheckInterval         = "AuctionEndingCheckInterval"
	SettingAuctionEndingCheckStartTime        = "AuctionEndingCheckStartTime"
	SettingAuctionWatchBidDigestEnabled       = "AuctionWatchBidDigestEnabled"
	SettingAuctionWatchBidDigestInterval      = "AuctionWatchBidDigestInterval"
	SettingAuctionWatchBidDigestStartTime     = "AuctionWatchBidDigestStartTime"
	SettingAuctionAlertsCheckEnabled          = "AuctionAlertsCheckEnabled"
	SettingAuctionAlertsCheckInterval         = "AuctionAlertsCheckInterval"
	SettingAuctionAlertsCheckStartTime        = "AuctionAlertsCheckStartTime"
	SettingCoinOfDayEnabled                   = "CoinOfDayEnabled"
	SettingCoinOfDayStartTime                 = "CoinOfDayStartTime"
	SettingCollectionHealthSnapshotsEnabled   = "CollectionHealthSnapshotsEnabled"
	SettingCollectionHealthSnapshotsStartTime = "CollectionHealthSnapshotsStartTime"
	SettingExternalToolServerEnabled          = "ExternalToolServerEnabled"
	SettingRegistrationMode                   = "RegistrationMode"
	SettingBackupStatus                       = "BackupStatus"
	SettingSetSnapshotEnabled                 = "SetSnapshotEnabled"
	SettingSetSnapshotStartTime               = "SetSnapshotStartTime"
	SettingCoinCategories                     = "CoinCategories"
	SettingCoinEras                           = "CoinEras"
)

const DefaultObversePrompt = `You are an expert numismatist specializing in ancient and modern coins. Analyze the obverse (front) of this coin and provide:
1. **Identification** – Confirm or correct the coin's identification
2. **Portrait / Design** – Describe the obverse design in detail
3. **Inscriptions** – Read all visible inscriptions and legends
4. **Condition** – Assess the obverse condition and grade
5. **Die Details** – Note any die varieties, errors, or notable features
6. **Authenticity** – Any observations relevant to authenticity`

const DefaultReversePrompt = `You are an expert numismatist specializing in ancient and modern coins. Analyze the reverse (back) of this coin and provide:
1. **Identification** – Confirm or correct the coin's identification from the reverse
2. **Design** – Describe the reverse design, motifs, and symbols in detail
3. **Inscriptions** – Read all visible inscriptions, legends, and mint marks
4. **Condition** – Assess the reverse condition and grade
5. **Die Details** – Note any die varieties, errors, or notable features
6. **Authenticity** – Any observations relevant to authenticity`

const DefaultTextExtractionPrompt = `Extract ALL text visible in this image exactly as written.
This is a store card or certificate that accompanies a coin purchase.
Preserve the original layout and formatting as much as possible.
Include store name, coin description, price, grade, reference numbers, dates, and any other text.
Return ONLY the extracted text, no commentary.`

var settingDefaults = map[string]string{
	SettingAIProvider:                         "",
	SettingOllamaURL:                          "http://localhost:11434",
	SettingOllamaModel:                        "llava",
	SettingObversePrompt:                      DefaultObversePrompt,
	SettingReversePrompt:                      DefaultReversePrompt,
	SettingTextExtractionPrompt:               DefaultTextExtractionPrompt,
	SettingOllamaTimeout:                      "300",
	SettingLogLevel:                           "info",
	SettingPublicAppURL:                       "",
	SettingNumistaAPIKey:                      "",
	SettingAnthropicAPIKey:                    "",
	SettingAnthropicModel:                     "claude-sonnet-4-20250514",
	SettingCoinSearchPrompt:                   "",
	SettingCoinShowsPrompt:                    "",
	SettingValuationPrompt:                    "",
	SettingSearXNGURL:                         "",
	SettingPushoverAppToken:                   "",
	SettingWishlistCheckEnabled:               "false",
	SettingWishlistCheckInterval:              "120",
	SettingWishlistCheckStartTime:             "02:00",
	SettingWishlistSearchAlertsCheckEnabled:   "false",
	SettingWishlistSearchAlertsCheckStartTime: "03:00",
	SettingValuationCheckEnabled:              "false",
	SettingValuationCheckInterval:             "7",
	SettingValuationCheckStartTime:            "03:00",
	SettingValuationMaxCoins:                  "50",
	SettingAuctionEndingCheckEnabled:          "false",
	SettingAuctionEndingCheckInterval:         "1440",
	SettingAuctionEndingCheckStartTime:        "08:00",
	SettingAuctionWatchBidDigestEnabled:       "false",
	SettingAuctionWatchBidDigestInterval:      "1440",
	SettingAuctionWatchBidDigestStartTime:     "08:00",
	SettingAuctionAlertsCheckEnabled:          "false",
	SettingAuctionAlertsCheckInterval:         "60",
	SettingAuctionAlertsCheckStartTime:        "08:00",
	SettingCoinOfDayEnabled:                   "false",
	SettingCoinOfDayStartTime:                 "07:00",
	SettingCollectionHealthSnapshotsEnabled:   "false",
	SettingCollectionHealthSnapshotsStartTime: "04:30",
	SettingExternalToolServerEnabled:          "false",
	SettingRegistrationMode:                   "closed",
	SettingBackupStatus:                       "not_configured",
	SettingSetSnapshotEnabled:                 "false",
	SettingSetSnapshotStartTime:               "04:00",
	SettingCoinCategories:                     "Roman\nGreek\nByzantine\nModern\nOther",
	SettingCoinEras:                           "ancient\nmedieval\nmodern",
}

// SettingsService provides access to application settings backed by the database.
type SettingsService struct {
	repo *repository.SettingsRepository
}

// NewSettingsService creates a new SettingsService.
func NewSettingsService(repo *repository.SettingsRepository) *SettingsService {
	return &SettingsService{repo: repo}
}

// GetSetting returns the value for a given key, falling back to defaults.
func (s *SettingsService) GetSetting(key string) string {
	setting, err := s.repo.FindByKey(key)
	if err != nil {
		if def, ok := settingDefaults[key]; ok {
			return def
		}
		return ""
	}
	// Treat empty prompt settings as unset so the default is used.
	// AIProvider intentionally allows empty (means unconfigured).
	if setting.Value == "" && key != SettingAIProvider {
		if def, ok := settingDefaults[key]; ok {
			return def
		}
	}
	return setting.Value
}

// SetSetting creates or updates a setting value.
func (s *SettingsService) SetSetting(key, value string) error {
	return s.repo.Upsert(key, value)
}

// GetAllSettings returns all settings merged with defaults.
func (s *SettingsService) GetAllSettings() map[string]string {
	result := make(map[string]string)
	for k, v := range settingDefaults {
		result[k] = v
	}

	settings, _ := s.repo.FindAll()
	for _, st := range settings {
		if st.Value != "" {
			result[st.Key] = st.Value
		}
	}
	return result
}

// GetSettingDefaults returns the built-in default values for all settings.
func (s *SettingsService) GetSettingDefaults() map[string]string {
	result := make(map[string]string)
	for k, v := range settingDefaults {
		result[k] = v
	}
	return result
}

// SyncLogLevel reads the LogLevel setting from the DB and applies it to the logger.
func (s *SettingsService) SyncLogLevel(logger *Logger) {
	level := s.GetSetting(SettingLogLevel)
	logger.SetLevel(level)
}

// ResolveLLMConfig reads AI provider settings and returns a configured LLMConfig.
func (s *SettingsService) ResolveLLMConfig() (LLMConfig, error) {
	provider := s.GetSetting(SettingAIProvider)
	if provider == "" {
		return LLMConfig{}, fmt.Errorf("AI provider not configured. Please select Anthropic or Ollama in Admin Settings.")
	}

	cfg := LLMConfig{
		Provider: provider,
	}

	switch provider {
	case "anthropic":
		cfg.APIKey = s.GetSetting(SettingAnthropicAPIKey)
		cfg.Model = s.GetSetting(SettingAnthropicModel)
		if cfg.APIKey == "" {
			return LLMConfig{}, fmt.Errorf("Anthropic API key is required")
		}
	case "ollama":
		cfg.Model = s.GetSetting(SettingOllamaModel)
		cfg.OllamaURL = s.GetSetting(SettingOllamaURL)
		cfg.SearXNGURL = s.GetSetting(SettingSearXNGURL)
	default:
		return LLMConfig{}, fmt.Errorf("unknown AI provider: %s", provider)
	}

	return cfg, nil
}
