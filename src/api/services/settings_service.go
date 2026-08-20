package services

import (
	"fmt"
	"strconv"
	"strings"
	"time"

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
	SettingNumistaSearchTTLHours              = "NumistaSearchTTLHours"
	SettingNumistaDetailTTLHours              = "NumistaDetailTTLHours"
	SettingNumistaEnrichmentLimit             = "NumistaEnrichmentLimit"
	SettingNumistaSearchResultLimit           = "NumistaSearchResultLimit"
	SettingNumistaSearchTimeoutSeconds        = "NumistaSearchTimeoutSeconds"
	SettingNumistaDetailTimeoutSeconds        = "NumistaDetailTimeoutSeconds"
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
	SettingUSPSAPIBaseURL                     = "USPSAPIBaseURL"
	SettingUSPSAPIKey                         = "USPSAPIKey"
	SettingUSPSAPIKeyHeader                   = "USPSAPIKeyHeader"
	SettingUPSAPIBaseURL                      = "UPSAPIBaseURL"
	SettingUPSTokenURL                        = "UPSTokenURL"
	SettingUPSClientID                        = "UPSClientID"
	SettingUPSClientSecret                    = "UPSClientSecret"
	SettingUPSScope                           = "UPSScope"
	SettingFedExAPIBaseURL                    = "FedExAPIBaseURL"
	SettingFedExTokenURL                      = "FedExTokenURL"
	SettingFedExClientID                      = "FedExClientID"
	SettingFedExClientSecret                  = "FedExClientSecret"
	SettingFedExScope                         = "FedExScope"
	SettingParcelAppEnabled                   = "ParcelAppEnabled"
	SettingShipmentSyncEnabled                = "ShipmentSyncEnabled"
	SettingShipmentSyncInterval               = "ShipmentSyncInterval"
	SettingShipmentSyncStartTime              = "ShipmentSyncStartTime"
	SettingShipmentSyncBatchSize              = "ShipmentSyncBatchSize"

	// 344-deep-agentic-coin-identification settings (data-model.md §8).
	SettingDeepIdentificationEnabled            = "DeepIdentificationEnabled"
	SettingDeepIdentificationWorkerCount        = "DeepIdentificationWorkerCount"
	SettingDeepIdentificationMaxActivePerUser   = "DeepIdentificationMaxActivePerUser"
	SettingDeepIdentificationQueueDepth         = "DeepIdentificationQueueDepth"
	SettingDeepIdentificationHardTimeoutSeconds = "DeepIdentificationHardTimeoutSeconds"
	// SettingDeepIdentificationQuickLookupTimeoutSeconds bounds the
	// quick-evidence extraction pass inside Deep Analysis (a full vision LLM
	// round trip via CoinLookupService.Lookup, `runner.go:112`). Validated
	// range 5-300s; 300s mirrors agent_proxy.go's own non-streaming
	// `requestClient` ceiling (5 minutes) and is never exceeded (351 T011,
	// FR-038).
	SettingDeepIdentificationQuickLookupTimeoutSeconds = "DeepIdentificationQuickLookupTimeoutSeconds"
	SettingDeepIdentificationEventRetentionHours       = "DeepIdentificationEventRetentionHours"
	SettingDeepIdentificationResultRetentionDays       = "DeepIdentificationResultRetentionDays"
	SettingDeepIdentificationMaxProviders              = "DeepIdentificationMaxProviders"
	SettingDeepIdentificationNumistaCallBudget         = "DeepIdentificationNumistaCallBudget"
	SettingDeepIdentificationOCREEnabled               = "DeepIdentificationOCREEnabled"
	SettingDeepIdentificationOCRECallBudget            = "DeepIdentificationOCRECallBudget"
	SettingDeepIdentificationRPCEnabled                = "DeepIdentificationRPCEnabled"

	// 355-wishlist-purchase-reminders: daily scheduler settings (FR-015).
	SettingReminderCheckEnabled   = "ReminderCheckEnabled"
	SettingReminderCheckStartTime = "ReminderCheckStartTime"
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
	SettingNumistaSearchTTLHours:              "24",
	SettingNumistaDetailTTLHours:              "168",
	SettingNumistaEnrichmentLimit:             "5",
	SettingNumistaSearchResultLimit:           "20",
	SettingNumistaSearchTimeoutSeconds:        "4",
	SettingNumistaDetailTimeoutSeconds:        "3",
	SettingAnthropicAPIKey:                    "",
	SettingAnthropicModel:                     "claude-sonnet-5",
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
	SettingUSPSAPIBaseURL:                     "",
	SettingUSPSAPIKey:                         "",
	SettingUSPSAPIKeyHeader:                   "X-API-Key",
	SettingUPSAPIBaseURL:                      "",
	SettingUPSTokenURL:                        "",
	SettingUPSClientID:                        "",
	SettingUPSClientSecret:                    "",
	SettingUPSScope:                           "",
	SettingFedExAPIBaseURL:                    "",
	SettingFedExTokenURL:                      "",
	SettingFedExClientID:                      "",
	SettingFedExClientSecret:                  "",
	SettingFedExScope:                         "",
	SettingParcelAppEnabled:                   "false",
	SettingShipmentSyncEnabled:                "false",
	SettingShipmentSyncInterval:               "20",
	SettingShipmentSyncStartTime:              "09:00",
	SettingShipmentSyncBatchSize:              "100",

	// 344-deep-agentic-coin-identification defaults (data-model.md §8).
	// Enabled defaults to false: this is a feature-flagged, safe-by-default
	// capability; flipping it on is an explicit out-of-band rollout step.
	SettingDeepIdentificationEnabled:            "false",
	SettingDeepIdentificationWorkerCount:        "2",
	SettingDeepIdentificationMaxActivePerUser:   "1",
	SettingDeepIdentificationQueueDepth:         "32",
	SettingDeepIdentificationHardTimeoutSeconds: "420",
	// 351-vision-first-deep-identification default (T011/T012): a full
	// vision LLM round trip through the Python service needs more than the
	// prior 15s magic literal; 90s is proportionate to the work performed
	// while remaining well under the 300s agent-proxy ceiling. Pairing this
	// with the raised 420s hard timeout above keeps the post-quick-lookup
	// pipeline budget at 420-90-20=310s, at or above the pre-change ~265s
	// (see .squad/decisions/inbox/cassius-quick-lookup-budget.md).
	SettingDeepIdentificationQuickLookupTimeoutSeconds: "90",
	SettingDeepIdentificationEventRetentionHours:       "24",
	SettingDeepIdentificationResultRetentionDays:       "90",
	SettingDeepIdentificationMaxProviders:              "4",
	SettingDeepIdentificationNumistaCallBudget:         "4",
	SettingDeepIdentificationOCREEnabled:               "false",
	SettingDeepIdentificationOCRECallBudget:            "3",
	SettingDeepIdentificationRPCEnabled:                "false",

	// 355-wishlist-purchase-reminders defaults (FR-015).
	SettingReminderCheckEnabled:   "true",
	SettingReminderCheckStartTime: "08:00",
}

type NumistaSettings struct {
	SearchTTL         time.Duration
	DetailTTL         time.Duration
	EnrichmentLimit   int
	SearchResultLimit int
	SearchTimeout     time.Duration
	DetailTimeout     time.Duration
	Valid             bool
}

// GetNumistaSettings reads and validates live Numista settings. Invalid values
// fall back independently to documented defaults and mark the snapshot invalid.
func (s *SettingsService) GetNumistaSettings() NumistaSettings {
	valid := true
	readInt := func(key string, fallback, minimum, maximum int) int {
		value, err := strconv.Atoi(strings.TrimSpace(s.GetSetting(key)))
		if err != nil || value < minimum || value > maximum {
			valid = false
			return fallback
		}
		return value
	}
	searchTTLHours := readInt(SettingNumistaSearchTTLHours, 24, 1, 720)
	detailTTLHours := readInt(SettingNumistaDetailTTLHours, 168, 1, 2160)
	searchTimeoutSeconds := readInt(SettingNumistaSearchTimeoutSeconds, 4, 1, 10)
	detailTimeoutSeconds := readInt(SettingNumistaDetailTimeoutSeconds, 3, 1, 10)
	return NumistaSettings{
		SearchTTL:         time.Duration(searchTTLHours) * time.Hour,
		DetailTTL:         time.Duration(detailTTLHours) * time.Hour,
		EnrichmentLimit:   readInt(SettingNumistaEnrichmentLimit, 5, 1, 10),
		SearchResultLimit: readInt(SettingNumistaSearchResultLimit, 20, 1, 50),
		SearchTimeout:     time.Duration(searchTimeoutSeconds) * time.Second,
		DetailTimeout:     time.Duration(detailTimeoutSeconds) * time.Second,
		Valid:             valid,
	}
}

// DeepIdentificationSettings is a validated snapshot of the live worker
// pool / retention bounds for the deep agentic identification job service
// (data-model.md §8). Invalid values fall back independently to documented
// defaults and mark the snapshot invalid.
type DeepIdentificationSettings struct {
	Enabled          bool
	WorkerCount      int
	MaxActivePerUser int
	QueueDepth       int
	HardTimeout      time.Duration
	// QuickLookupTimeout bounds the quick-evidence extraction pass inside
	// Deep Analysis (351 T011/FR-038). It is consumed from the same ctx as
	// the overall HardTimeout, so it is a prefix of, not additional to,
	// that budget (see deepPipelineBounds and Run's remaining-budget check
	// in deep_identification_pipeline_runner.go).
	QuickLookupTimeout time.Duration
	EventRetention     time.Duration
	ResultRetention    time.Duration
	MaxProviders       int
	NumistaCallBudget  int
	OCREEnabled        bool
	OCRECallBudget     int
	RPCEnabled         bool
	Valid              bool
}

// GetDeepIdentificationSettings reads and validates the live deep
// identification settings, mirroring the GetNumistaSettings pattern.
func (s *SettingsService) GetDeepIdentificationSettings() DeepIdentificationSettings {
	valid := true
	readInt := func(key string, fallback, minimum, maximum int) int {
		value, err := strconv.Atoi(strings.TrimSpace(s.GetSetting(key)))
		if err != nil || value < minimum || value > maximum {
			valid = false
			return fallback
		}
		return value
	}
	readBool := func(key string, fallback bool) bool {
		value := strings.TrimSpace(strings.ToLower(s.GetSetting(key)))
		switch value {
		case "true":
			return true
		case "false":
			return false
		default:
			valid = false
			return fallback
		}
	}
	hardTimeoutSeconds := readInt(SettingDeepIdentificationHardTimeoutSeconds, 420, 1, 900)
	quickLookupTimeoutSeconds := readInt(SettingDeepIdentificationQuickLookupTimeoutSeconds, 90, 5, 300)
	eventRetentionHours := readInt(SettingDeepIdentificationEventRetentionHours, 24, 1, 720)
	resultRetentionDays := readInt(SettingDeepIdentificationResultRetentionDays, 90, 1, 3650)
	return DeepIdentificationSettings{
		Enabled:            readBool(SettingDeepIdentificationEnabled, false),
		WorkerCount:        readInt(SettingDeepIdentificationWorkerCount, 2, 1, 32),
		MaxActivePerUser:   readInt(SettingDeepIdentificationMaxActivePerUser, 1, 1, 10),
		QueueDepth:         readInt(SettingDeepIdentificationQueueDepth, 32, 1, 1000),
		HardTimeout:        time.Duration(hardTimeoutSeconds) * time.Second,
		QuickLookupTimeout: time.Duration(quickLookupTimeoutSeconds) * time.Second,
		EventRetention:     time.Duration(eventRetentionHours) * time.Hour,
		ResultRetention:    time.Duration(resultRetentionDays) * 24 * time.Hour,
		MaxProviders:       readInt(SettingDeepIdentificationMaxProviders, 4, 1, 10),
		NumistaCallBudget:  readInt(SettingDeepIdentificationNumistaCallBudget, 4, 1, 20),
		OCREEnabled:        readBool(SettingDeepIdentificationOCREEnabled, false),
		OCRECallBudget:     readInt(SettingDeepIdentificationOCRECallBudget, 3, 1, 20),
		RPCEnabled:         readBool(SettingDeepIdentificationRPCEnabled, false),
		Valid:              valid,
	}
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

// SplitSettingList parses a newline-delimited AppSetting value (the shape
// used by CoinCategories/CoinEras and similar admin-defined lists) into a
// trimmed, non-empty slice, mirroring how the frontend's parseOptionList
// treats the same setting shape.
func SplitSettingList(value string) []string {
	var out []string
	for _, line := range strings.Split(value, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
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
