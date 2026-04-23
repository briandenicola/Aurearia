package services

import (
	"github.com/briandenicola/ancient-coins-api/models"
	"gorm.io/gorm"
)

const (
	SettingAIProvider           = "AIProvider"
	SettingOllamaURL            = "OllamaURL"
	SettingOllamaModel          = "OllamaModel"
	SettingObversePrompt        = "ObversePrompt"
	SettingReversePrompt        = "ReversePrompt"
	SettingTextExtractionPrompt = "TextExtractionPrompt"
	SettingOllamaTimeout        = "OllamaTimeout"
	SettingLogLevel             = "LogLevel"
	SettingNumistaAPIKey        = "NumistaAPIKey"
	SettingAnthropicAPIKey      = "AnthropicAPIKey"
	SettingAnthropicModel       = "AnthropicModel"
	SettingCoinSearchPrompt     = "CoinSearchPrompt"
	SettingCoinShowsPrompt      = "CoinShowsPrompt"
	SettingValuationPrompt      = "ValuationPrompt"
	SettingSearXNGURL           = "SearXNGURL"
	SettingWishlistCheckEnabled    = "WishlistCheckEnabled"
	SettingWishlistCheckInterval   = "WishlistCheckInterval"
	SettingWishlistCheckStartTime  = "WishlistCheckStartTime"
	SettingValuationCheckEnabled   = "ValuationCheckEnabled"
	SettingValuationCheckInterval  = "ValuationCheckIntervalDays"
	SettingValuationCheckStartTime = "ValuationCheckStartTime"
	SettingValuationMaxCoins       = "ValuationMaxCoins"
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
	SettingAIProvider:           "",
	SettingOllamaURL:            "http://localhost:11434",
	SettingOllamaModel:          "llava",
	SettingObversePrompt:        DefaultObversePrompt,
	SettingReversePrompt:        DefaultReversePrompt,
	SettingTextExtractionPrompt: DefaultTextExtractionPrompt,
	SettingOllamaTimeout:        "300",
	SettingLogLevel:             "info",
	SettingNumistaAPIKey:        "",
	SettingAnthropicAPIKey:      "",
	SettingAnthropicModel:       "claude-sonnet-4-20250514",
	SettingCoinSearchPrompt:     "",
	SettingCoinShowsPrompt:      "",
	SettingValuationPrompt:      "",
	SettingSearXNGURL:           "",
	SettingWishlistCheckEnabled:     "false",
	SettingWishlistCheckInterval:    "120",
	SettingWishlistCheckStartTime:   "02:00",
	SettingValuationCheckEnabled:    "false",
	SettingValuationCheckInterval:   "7",
	SettingValuationCheckStartTime:  "03:00",
	SettingValuationMaxCoins:        "50",
}

var settingsDB *gorm.DB

// InitSettings sets the database connection used by the settings service.
// Must be called before any GetSetting/SetSetting calls.
func InitSettings(db *gorm.DB) {
	settingsDB = db
}

func GetSetting(key string) string {
	var setting models.AppSetting
	if err := settingsDB.Where("key = ?", key).First(&setting).Error; err != nil {
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

func SetSetting(key, value string) error {
	var setting models.AppSetting
	result := settingsDB.Where("key = ?", key).First(&setting)
	if result.Error != nil {
		setting = models.AppSetting{Key: key, Value: value}
		return settingsDB.Create(&setting).Error
	}
	setting.Value = value
	return settingsDB.Save(&setting).Error
}

func GetAllSettings() map[string]string {
	result := make(map[string]string)
	for k, v := range settingDefaults {
		result[k] = v
	}

	var settings []models.AppSetting
	settingsDB.Find(&settings)
	for _, s := range settings {
		if s.Value != "" {
			result[s.Key] = s.Value
		}
	}
	return result
}

func GetSettingDefaults() map[string]string {
	result := make(map[string]string)
	for k, v := range settingDefaults {
		result[k] = v
	}
	return result
}
