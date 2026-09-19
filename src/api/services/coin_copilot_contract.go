package services

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/briandenicola/ancient-coins-api/models"
)

const (
	CoinCopilotSchemaVersion                   = 1
	CoinCopilotMaxEventBytes                   = 64 * 1024
	CoinCopilotMaxPromptBytes                  = 4000
	CoinCopilotMaxMessages                     = 50
	CoinCopilotMaxMessageBytes                 = 100000
	CoinCopilotMaxPlanItems                    = 12
	CoinCopilotMaxPlanTitle                    = 200
	CoinCopilotMaxClarification                = 500
	CoinCopilotHistoryRunLimit                 = 10
	CoinCopilotHistoryMaxBytes                 = 20000
	CoinCopilotHistoryMessageMaxBytes          = 4000
	DeepAnalysisHandoffMaxRequestBytes         = 65_536
	DeepAnalysisHandoffMaxPublicEventBytes     = 65_536
	DeepAnalysisHandoffMaxPersistedResultBytes = 32_768
)

var (
	ErrInvalidCopilotFrame = errors.New("invalid coin copilot frame")
	ErrDuplicateFrame      = errors.New("duplicate coin copilot frame")
)

var CoinCopilotAllowedTools = []string{
	"search_my_collection",
	"get_coin",
	"collection_summary",
	"top_coins_by_value",
	"deep_analysis_handoff",
	"portfolio_review",
	"gap_analysis",
	"market_search",
	"auction_search",
	"price_trends",
	"similar_lots",
}

var copilotCallbackTools = map[string]bool{
	"search_my_collection":  true,
	"get_coin":              true,
	"collection_summary":    true,
	"top_coins_by_value":    true,
	"deep_analysis_handoff": true,
}

func IsCoinCopilotToolAllowed(tool string) bool {
	for _, allowed := range CoinCopilotAllowedTools {
		if tool == allowed {
			return true
		}
	}
	return false
}

func IsCoinCopilotCallbackTool(tool string) bool { return copilotCallbackTools[tool] }

type CopilotMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type CopilotPlanItem struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Status string `json:"status"`
}

type CopilotCompletedTool struct {
	ToolCallID     string          `json:"tool_call_id"`
	ToolName       string          `json:"tool_name"`
	ResultDigest   string          `json:"result_digest"`
	Result         json.RawMessage `json:"result"`
	OriginalBytes  int             `json:"original_bytes"`
	PersistedBytes int             `json:"persisted_bytes"`
	Truncated      bool            `json:"truncated"`
}

type CopilotBoundedToolResult struct {
	Truncated     bool   `json:"truncated"`
	OriginalBytes int    `json:"original_bytes"`
	Digest        string `json:"digest"`
	Summary       string `json:"summary"`
}

type CopilotClarification struct {
	Question  string   `json:"question"`
	InputType string   `json:"input_type"`
	Choices   []string `json:"choices"`
}

type CopilotUsage struct {
	Iterations   int   `json:"iterations"`
	ToolCalls    int   `json:"tool_calls"`
	InputTokens  int64 `json:"input_tokens"`
	OutputTokens int64 `json:"output_tokens"`
}

type CopilotSpecialistQuery struct {
	Query string `json:"query"`
	Limit int    `json:"limit,omitempty"`
}

type CopilotFieldProvenance struct {
	Field             string `json:"field"`
	SourceURL         string `json:"source_url"`
	ObservedAt        string `json:"observed_at"`
	Confidence        string `json:"confidence"`
	VerificationState string `json:"verification_state"`
}

type CopilotSpecialistEvidence struct {
	Kind                string                   `json:"kind"`
	SourceURL           string                   `json:"source_url"`
	CanonicalSourceID   string                   `json:"canonical_source_id"`
	Provider            string                   `json:"provider"`
	ObservedAt          string                   `json:"observed_at"`
	Confidence          string                   `json:"confidence"`
	VerificationState   string                   `json:"verification_state"`
	Title               string                   `json:"title"`
	Description         *string                  `json:"description"`
	DealerName          *string                  `json:"dealer_name,omitempty"`
	ListedPrice         *float64                 `json:"listed_price,omitempty"`
	Currency            *string                  `json:"currency,omitempty"`
	Availability        *string                  `json:"availability,omitempty"`
	AuctionHouse        *string                  `json:"auction_house,omitempty"`
	SaleName            *string                  `json:"sale_name,omitempty"`
	LotNumber           *string                  `json:"lot_number,omitempty"`
	SaleDate            *string                  `json:"sale_date,omitempty"`
	Estimate            *float64                 `json:"estimate,omitempty"`
	CurrentBid          *float64                 `json:"current_bid,omitempty"`
	LotStatus           *string                  `json:"lot_status,omitempty"`
	Amount              *float64                 `json:"amount,omitempty"`
	PriceBasis          *string                  `json:"price_basis,omitempty"`
	Ruler               *string                  `json:"ruler,omitempty"`
	Denomination        *string                  `json:"denomination,omitempty"`
	Era                 *string                  `json:"era,omitempty"`
	Material            *string                  `json:"material,omitempty"`
	SimilarityScore     float64                  `json:"similarity_score,omitempty"`
	MatchedAttributes   []string                 `json:"matched_attributes,omitempty"`
	MaterialDifferences []string                 `json:"material_differences,omitempty"`
	Provenance          []CopilotFieldProvenance `json:"provenance"`
}

type CopilotPriceTrend struct {
	State               string   `json:"state"`
	SampleSize          int      `json:"sample_size"`
	DateFrom            *string  `json:"date_from"`
	DateTo              *string  `json:"date_to"`
	Currency            *string  `json:"currency"`
	PriceBasis          *string  `json:"price_basis"`
	Low                 *float64 `json:"low"`
	Median              *float64 `json:"median"`
	High                *float64 `json:"high"`
	Confidence          string   `json:"confidence"`
	Limitations         []string `json:"limitations"`
	SupportingSourceIDs []string `json:"supporting_source_ids"`
}

type CopilotProviderAttempt struct {
	Provider      string  `json:"provider"`
	Status        string  `json:"status"`
	ObservedAt    string  `json:"observed_at"`
	AcceptedItems int     `json:"accepted_items"`
	WarningCode   *string `json:"warning_code"`
}

type CopilotSpecialistTruncation struct {
	Truncated      bool   `json:"truncated"`
	OriginalBytes  int    `json:"original_bytes"`
	PersistedBytes int    `json:"persisted_bytes"`
	Digest         string `json:"digest"`
	OmittedItems   int    `json:"omitted_items"`
}

type CopilotSpecialistResult struct {
	SchemaVersion    int                         `json:"schema_version"`
	Capability       string                      `json:"capability"`
	Outcome          string                      `json:"outcome"`
	Items            []CopilotSpecialistEvidence `json:"items"`
	Trend            *CopilotPriceTrend          `json:"trend"`
	ProviderAttempts []CopilotProviderAttempt    `json:"provider_attempts"`
	Warnings         []string                    `json:"warnings"`
	Truncation       CopilotSpecialistTruncation `json:"truncation"`
}

type CopilotSpecialistPublicEvidence struct {
	Kind                string   `json:"kind"`
	Title               string   `json:"title"`
	SourceURL           string   `json:"sourceUrl"`
	ObservedAt          string   `json:"observedAt"`
	Confidence          string   `json:"confidence"`
	VerificationState   string   `json:"verificationState"`
	Description         *string  `json:"description,omitempty"`
	DealerName          *string  `json:"dealerName,omitempty"`
	ListedPrice         *float64 `json:"listedPrice,omitempty"`
	Currency            *string  `json:"currency,omitempty"`
	Availability        *string  `json:"availability,omitempty"`
	Ruler               *string  `json:"ruler,omitempty"`
	Denomination        *string  `json:"denomination,omitempty"`
	Era                 *string  `json:"era,omitempty"`
	Material            *string  `json:"material,omitempty"`
	Facts               []string `json:"facts"`
	MatchedAttributes   []string `json:"matchedAttributes"`
	MaterialDifferences []string `json:"materialDifferences"`
}

type CopilotSpecialistPublicTrend struct {
	State               string   `json:"state"`
	SampleSize          int      `json:"sampleSize"`
	DateFrom            *string  `json:"dateFrom"`
	DateTo              *string  `json:"dateTo"`
	Currency            *string  `json:"currency"`
	PriceBasis          *string  `json:"priceBasis"`
	Low                 *float64 `json:"low"`
	Median              *float64 `json:"median"`
	High                *float64 `json:"high"`
	Confidence          string   `json:"confidence"`
	Limitations         []string `json:"limitations"`
	SupportingSourceIDs []string `json:"supportingSourceIds"`
}

type CopilotSpecialistPublicTruncation struct {
	Truncated      bool   `json:"truncated"`
	OriginalBytes  int    `json:"originalBytes"`
	PersistedBytes int    `json:"persistedBytes"`
	Digest         string `json:"digest"`
	OmittedItems   int    `json:"omittedItems"`
}

type CopilotSpecialistPublicResult struct {
	Capability string                            `json:"capability"`
	Outcome    string                            `json:"outcome"`
	Items      []CopilotSpecialistPublicEvidence `json:"items"`
	Trend      *CopilotSpecialistPublicTrend     `json:"trend"`
	Warnings   []string                          `json:"warnings"`
	Truncation CopilotSpecialistPublicTruncation `json:"truncation"`
}

type CopilotCheckpointState struct {
	SchemaVersion        int                    `json:"schema_version"`
	Messages             []CopilotMessage       `json:"messages"`
	Plan                 []CopilotPlanItem      `json:"plan"`
	CompletedTools       []CopilotCompletedTool `json:"completed_tools"`
	PendingClarification *CopilotClarification  `json:"pending_clarification"`
	NextAction           string                 `json:"next_action"`
	Counters             CopilotUsage           `json:"counters"`
}

type CopilotLimitsProxy struct {
	MaxIterations               int `json:"max_iterations"`
	MaxToolCalls                int `json:"max_tool_calls"`
	MaxConcurrentTools          int `json:"max_concurrent_tools"`
	HardTimeoutSeconds          int `json:"hard_timeout_seconds"`
	MaxPersistedToolResultBytes int `json:"max_persisted_tool_result_bytes"`
}

type CopilotCheckpointProxy struct {
	Version              int64                  `json:"version"`
	Plan                 []CopilotPlanItem      `json:"plan"`
	CompletedTools       []CopilotCompletedTool `json:"completed_tools"`
	PendingClarification *CopilotClarification  `json:"pending_clarification"`
	NextAction           string                 `json:"next_action"`
	Counters             CopilotUsage           `json:"counters"`
}

type CopilotExecuteProxyRequest struct {
	SchemaVersion    int                    `json:"schema_version"`
	ThreadID         string                 `json:"thread_id"`
	RunID            string                 `json:"run_id"`
	ExecutionID      string                 `json:"execution_id"`
	Goal             string                 `json:"goal"`
	Messages         []CopilotMessage       `json:"messages"`
	Checkpoint       CopilotCheckpointProxy `json:"checkpoint"`
	AppContext       map[string]any         `json:"app_context"`
	CollectorContext *CollectorContext      `json:"collector_context,omitempty"`
	LLM              LLMConfig              `json:"llm"`
	Limits           CopilotLimitsProxy     `json:"limits"`
	ToolsBaseURL     string                 `json:"tools_base_url"`
	ExecutionToken   string                 `json:"execution_token"`
	AllowedTools     []string               `json:"allowed_tools"`
}

type CopilotAgentFrame struct {
	SchemaVersion int             `json:"schema_version"`
	RunID         string          `json:"run_id"`
	ExecutionID   string          `json:"execution_id"`
	FrameID       string          `json:"frame_id"`
	Type          string          `json:"type"`
	Payload       json.RawMessage `json:"payload"`
}

var forbiddenCopilotField = regexp.MustCompile(`(?i)"(?:reasoning|thought|scratchpad|rawPrompt|raw_prompt|rawToolInput|raw_tool_input|rawToolOutput|raw_tool_output|chain_of_thought)"\s*:`)
var tokenShapedValue = regexp.MustCompile(`(?i)(bearer\s+[A-Za-z0-9._~+/\-=]{12,}|(?:api[_-]?key|token|secret|password)\s*[:=]\s*["']?[A-Za-z0-9._~+/\-=]{8,})`)

func ValidateCopilotFrame(frame CopilotAgentFrame, run *models.CoinCopilotRun) error {
	if frame.SchemaVersion != CoinCopilotSchemaVersion || frame.RunID != run.ID ||
		frame.ExecutionID != run.ExecutionID || frame.FrameID == "" {
		return ErrInvalidCopilotFrame
	}
	switch frame.Type {
	case "plan_updated", "tool_started", "tool_completed", "checkpoint",
		"clarification_required", "completed", "failed", "usage":
	default:
		return ErrInvalidCopilotFrame
	}
	if len(frame.Payload) == 0 || len(frame.Payload) > CoinCopilotMaxEventBytes || !json.Valid(frame.Payload) ||
		forbiddenCopilotField.Match(frame.Payload) {
		return ErrInvalidCopilotFrame
	}
	return nil
}

func SanitizeCopilotJSON(raw []byte, maxBytes int) ([]byte, int, bool, string, error) {
	if !json.Valid(raw) || forbiddenCopilotField.Match(raw) {
		return nil, 0, false, "", ErrInvalidCopilotFrame
	}
	originalBytes := len(raw)
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, 0, false, "", ErrInvalidCopilotFrame
	}
	value = sanitizeCopilotValue(value)
	sanitized, err := marshalCopilotCanonical(value)
	if err != nil {
		return nil, 0, false, "", err
	}
	digest := sha256.Sum256(sanitized)
	digestText := hex.EncodeToString(digest[:])
	if maxBytes <= 0 || len(sanitized) <= maxBytes {
		return sanitized, originalBytes, false, digestText, nil
	}
	summary := map[string]any{
		"truncated":      true,
		"original_bytes": originalBytes,
		"digest":         digestText,
		"summary":        "Tool result exceeded the persisted-result limit.",
	}
	bounded, err := marshalCopilotCanonical(summary)
	if err != nil {
		return nil, 0, false, "", err
	}
	if len(bounded) > maxBytes {
		return nil, 0, false, "", fmt.Errorf("%w: persisted result limit too small", ErrInvalidCopilotFrame)
	}
	return bounded, originalBytes, true, digestText, nil
}

func marshalCopilotCanonical(value any) ([]byte, error) {
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		return nil, err
	}
	return bytes.TrimSuffix(buffer.Bytes(), []byte{'\n'}), nil
}

func sanitizeCopilotValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			lower := strings.ToLower(key)
			if strings.Contains(lower, "token") || strings.Contains(lower, "secret") ||
				strings.Contains(lower, "password") || strings.Contains(lower, "api_key") ||
				strings.Contains(lower, "apikey") {
				typed[key] = "[REDACTED]"
				continue
			}
			typed[key] = sanitizeCopilotValue(child)
		}
		return typed
	case []any:
		for i, child := range typed {
			typed[i] = sanitizeCopilotValue(child)
		}
		return typed
	case string:
		return tokenShapedValue.ReplaceAllString(typed, "[REDACTED]")
	}
	return value
}

func SanitizeCopilotText(value string, max int) string {
	value = tokenShapedValue.ReplaceAllString(value, "[REDACTED]")
	if max > 0 && len(value) > max {
		return value[:max]
	}
	return value
}

func CopilotCheckpointDigest(stateJSON string) string {
	sum := sha256.Sum256([]byte(stateJSON))
	return hex.EncodeToString(sum[:])
}

func ValidateCopilotSpecialistQuery(query CopilotSpecialistQuery) error {
	if utf8.RuneCountInString(query.Query) < 1 || utf8.RuneCountInString(query.Query) > 500 ||
		query.Limit < 0 || query.Limit > 10 {
		return ErrInvalidCopilotFrame
	}
	return nil
}

func validSpecialistTimestamp(value string) bool {
	parsed, err := time.Parse(time.RFC3339, value)
	return err == nil && strings.HasSuffix(value, "Z") && parsed.Location() == time.UTC
}

func validSpecialistURL(value string) bool {
	if len(value) == 0 || len(value) > 2048 {
		return false
	}
	parsed, err := url.ParseRequestURI(value)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil {
		return false
	}
	host := strings.TrimSuffix(strings.ToLower(parsed.Hostname()), ".")
	if host == "" || host == "localhost" || strings.HasSuffix(host, ".localhost") ||
		strings.HasSuffix(host, ".local") {
		return false
	}
	if ip := net.ParseIP(host); ip != nil && !ip.IsGlobalUnicast() {
		return false
	}
	if ip := net.ParseIP(host); ip != nil && ip.IsPrivate() {
		return false
	}
	return true
}

func validSpecialistConfidence(value string) bool {
	return value == "high" || value == "medium" || value == "low"
}

func validateBoundedSpecialistStrings(values []string, maximum, itemMaximum int) bool {
	if len(values) > maximum {
		return false
	}
	for _, value := range values {
		if utf8.RuneCountInString(value) > itemMaximum {
			return false
		}
	}
	return true
}

func validateCopilotSpecialistEvidence(item CopilotSpecialistEvidence, expectedKind string) bool {
	if item.Kind != expectedKind || !validSpecialistURL(item.SourceURL) ||
		!validSpecialistURL(item.CanonicalSourceID) ||
		len(item.Provider) == 0 || len(item.Provider) > 64 ||
		!validSpecialistTimestamp(item.ObservedAt) ||
		!validSpecialistConfidence(item.Confidence) ||
		(item.VerificationState != "verified" && item.VerificationState != "partial") ||
		utf8.RuneCountInString(item.Title) < 1 || utf8.RuneCountInString(item.Title) > 300 ||
		len(item.Provenance) < 1 || len(item.Provenance) > 20 {
		return false
	}
	if item.Description != nil && utf8.RuneCountInString(*item.Description) > 500 {
		return false
	}
	seenFields := map[string]bool{}
	for _, provenance := range item.Provenance {
		if provenance.Field == "" || seenFields[provenance.Field] ||
			provenance.SourceURL != item.SourceURL ||
			!validSpecialistTimestamp(provenance.ObservedAt) ||
			!validSpecialistConfidence(provenance.Confidence) ||
			(provenance.VerificationState != "verified" && provenance.VerificationState != "partial") {
			return false
		}
		seenFields[provenance.Field] = true
	}
	if !seenFields["title"] {
		return false
	}
	if expectedKind == "similar_lot" {
		return item.SimilarityScore >= 0 && item.SimilarityScore <= 1 &&
			len(item.MatchedAttributes) >= 1 &&
			validateBoundedSpecialistStrings(item.MatchedAttributes, 20, 200) &&
			validateBoundedSpecialistStrings(item.MaterialDifferences, 20, 200)
	}
	return len(item.MatchedAttributes) == 0 && len(item.MaterialDifferences) == 0
}

func validCopilotSpecialistSource(capability string, item CopilotSpecialistEvidence) bool {
	parsed, err := url.Parse(item.SourceURL)
	if err != nil {
		return false
	}
	host := strings.TrimSuffix(strings.ToLower(parsed.Hostname()), ".")
	hostMatches := func(allowed string) bool {
		return host == allowed || strings.HasSuffix(host, "."+allowed)
	}
	switch capability {
	case "market_search":
		if item.Provider != "cng_dealer_search" {
			return false
		}
		for _, allowed := range []string{
			"biddr.com", "catawiki.com", "cngcoins.com", "forumancientcoins.com",
			"hjbltd.com", "ma-shops.com", "vcoins.com",
		} {
			if hostMatches(allowed) {
				return true
			}
		}
		return false
	case "auction_search", "price_trends", "similar_lots":
		return item.Provider == "numisbids" && hostMatches("numisbids.com")
	default:
		return false
	}
}

func validateCopilotPriceTrend(trend *CopilotPriceTrend, items []CopilotSpecialistEvidence) bool {
	if trend == nil || trend.SampleSize < 0 || trend.SampleSize > len(items) ||
		!validSpecialistConfidence(trend.Confidence) ||
		!validateBoundedSpecialistStrings(trend.Limitations, 10, 500) ||
		len(trend.SupportingSourceIDs) > 10 {
		return false
	}
	switch trend.State {
	case "rising", "stable", "declining":
		if trend.SampleSize < 3 || trend.DateFrom == nil || trend.DateTo == nil ||
			trend.Currency == nil || trend.PriceBasis == nil {
			return false
		}
	case "unknown":
	default:
		return false
	}
	for _, sourceID := range trend.SupportingSourceIDs {
		if !validSpecialistURL(sourceID) {
			return false
		}
	}
	return true
}

func ValidateCopilotSpecialistResult(result CopilotSpecialistResult, invokedCapability string) error {
	kinds := map[string]string{
		"market_search":  "dealer_listing",
		"auction_search": "auction_lot",
		"price_trends":   "sale_observation",
		"similar_lots":   "similar_lot",
	}
	expectedKind, ok := kinds[result.Capability]
	if !ok || result.SchemaVersion != 1 || result.Capability != invokedCapability ||
		len(result.Items) > 10 || len(result.ProviderAttempts) > 10 ||
		!validateBoundedSpecialistStrings(result.Warnings, 10, 500) {
		return ErrInvalidCopilotFrame
	}
	switch result.Outcome {
	case "complete":
		if len(result.Items) == 0 {
			return ErrInvalidCopilotFrame
		}
	case "partial":
		if len(result.Items) == 0 {
			return ErrInvalidCopilotFrame
		}
	case "no_match", "unavailable":
		if len(result.Items) != 0 {
			return ErrInvalidCopilotFrame
		}
	default:
		return ErrInvalidCopilotFrame
	}
	hasDegradedAttempt := false
	for _, attempt := range result.ProviderAttempts {
		if len(attempt.Provider) == 0 || len(attempt.Provider) > 64 ||
			!validSpecialistTimestamp(attempt.ObservedAt) ||
			attempt.AcceptedItems < 0 || attempt.AcceptedItems > 10 {
			return ErrInvalidCopilotFrame
		}
		switch attempt.Status {
		case "success", "no_match":
			if attempt.WarningCode != nil {
				return ErrInvalidCopilotFrame
			}
		case "timeout", "failure", "unavailable", "malformed":
			hasDegradedAttempt = true
			if attempt.WarningCode == nil || *attempt.WarningCode == "" {
				return ErrInvalidCopilotFrame
			}
		default:
			return ErrInvalidCopilotFrame
		}
	}
	if (result.Outcome == "partial" || result.Outcome == "unavailable") && !hasDegradedAttempt {
		return ErrInvalidCopilotFrame
	}
	if result.Outcome == "no_match" && hasDegradedAttempt {
		return ErrInvalidCopilotFrame
	}
	for _, item := range result.Items {
		if !validateCopilotSpecialistEvidence(item, expectedKind) ||
			!validCopilotSpecialistSource(result.Capability, item) {
			return ErrInvalidCopilotFrame
		}
	}
	if result.Capability == "price_trends" {
		if result.Trend == nil {
			if result.Outcome != "unavailable" {
				return ErrInvalidCopilotFrame
			}
		} else if !validateCopilotPriceTrend(result.Trend, result.Items) {
			return ErrInvalidCopilotFrame
		}
	} else if result.Trend != nil {
		return ErrInvalidCopilotFrame
	}
	truncation := result.Truncation
	if truncation.OriginalBytes < 0 || truncation.PersistedBytes < 0 ||
		truncation.PersistedBytes > truncation.OriginalBytes || truncation.OmittedItems < 0 ||
		len(truncation.Digest) != 64 {
		return ErrInvalidCopilotFrame
	}
	if _, err := hex.DecodeString(truncation.Digest); err != nil {
		return ErrInvalidCopilotFrame
	}
	return nil
}

func isCoinCopilotSpecialistTool(toolName string) bool {
	switch toolName {
	case "market_search", "auction_search", "price_trends", "similar_lots":
		return true
	default:
		return false
	}
}

func DecodeCopilotSpecialistResult(raw json.RawMessage, invokedCapability string) (CopilotSpecialistResult, error) {
	var result CopilotSpecialistResult
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&result); err != nil {
		return result, ErrInvalidCopilotFrame
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return result, ErrInvalidCopilotFrame
	}
	if err := ValidateCopilotSpecialistResult(result, invokedCapability); err != nil {
		return result, err
	}
	return result, nil
}

func DecodeCopilotBoundedToolResult(raw json.RawMessage) (CopilotBoundedToolResult, error) {
	var result CopilotBoundedToolResult
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&result); err != nil {
		return result, ErrInvalidCopilotFrame
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) ||
		!result.Truncated || result.OriginalBytes < 0 || len(result.Digest) != 64 ||
		result.Summary != "Tool result exceeded the persisted-result limit." {
		return result, ErrInvalidCopilotFrame
	}
	if _, err := hex.DecodeString(result.Digest); err != nil {
		return result, ErrInvalidCopilotFrame
	}
	return result, nil
}

func ProjectCopilotSpecialistResult(result CopilotSpecialistResult, invokedCapability string) (CopilotSpecialistPublicResult, error) {
	if err := ValidateCopilotSpecialistResult(result, invokedCapability); err != nil {
		return CopilotSpecialistPublicResult{}, err
	}
	public := CopilotSpecialistPublicResult{
		Capability: result.Capability,
		Outcome:    result.Outcome,
		Items:      make([]CopilotSpecialistPublicEvidence, 0, len(result.Items)),
		Warnings:   sanitizeSpecialistStrings(result.Warnings, 500),
		Truncation: CopilotSpecialistPublicTruncation{
			Truncated: result.Truncation.Truncated, OriginalBytes: result.Truncation.OriginalBytes,
			PersistedBytes: result.Truncation.PersistedBytes, Digest: result.Truncation.Digest,
			OmittedItems: result.Truncation.OmittedItems,
		},
	}
	for _, item := range result.Items {
		proven := make(map[string]bool, len(item.Provenance))
		for _, provenance := range item.Provenance {
			proven[provenance.Field] = true
		}
		projected := CopilotSpecialistPublicEvidence{
			Kind: item.Kind, Title: SanitizeCopilotText(item.Title, 300),
			SourceURL: item.SourceURL, ObservedAt: item.ObservedAt,
			Confidence: item.Confidence, VerificationState: item.VerificationState,
			Facts: []string{}, MatchedAttributes: sanitizeSpecialistStrings(item.MatchedAttributes, 200),
			MaterialDifferences: sanitizeSpecialistStrings(item.MaterialDifferences, 200),
		}
		projectString := func(field string, value *string, max int) *string {
			if !proven[field] || value == nil {
				return nil
			}
			sanitized := SanitizeCopilotText(*value, max)
			if sanitized == "" {
				return nil
			}
			return &sanitized
		}
		projected.Description = projectString("description", item.Description, 2000)
		projected.DealerName = projectString("dealer_name", item.DealerName, 300)
		if proven["listed_price"] && item.ListedPrice != nil {
			value := *item.ListedPrice
			projected.ListedPrice = &value
		}
		projected.Currency = projectString("currency", item.Currency, 3)
		projected.Availability = projectString("availability", item.Availability, 50)
		projected.Ruler = projectString("ruler", item.Ruler, 200)
		projected.Denomination = projectString("denomination", item.Denomination, 200)
		projected.Era = projectString("era", item.Era, 100)
		projected.Material = projectString("material", item.Material, 100)
		addSpecialistFact := func(field, label, value string) {
			if proven[field] && value != "" {
				projected.Facts = append(projected.Facts, label+": "+SanitizeCopilotText(value, 300))
			}
		}
		addSpecialistFact("dealer_name", "Dealer", pointerString(item.DealerName))
		if proven["listed_price"] && item.ListedPrice != nil {
			addSpecialistFact("listed_price", "Price", specialistMoney(item.Currency, *item.ListedPrice))
		}
		addSpecialistFact("availability", "Availability", pointerString(item.Availability))
		addSpecialistFact("auction_house", "Auction house", pointerString(item.AuctionHouse))
		addSpecialistFact("sale_name", "Sale", pointerString(item.SaleName))
		addSpecialistFact("lot_number", "Lot", pointerString(item.LotNumber))
		addSpecialistFact("sale_date", "Sale date", pointerString(item.SaleDate))
		if proven["estimate"] && item.Estimate != nil {
			addSpecialistFact("estimate", "Estimate", specialistMoney(item.Currency, *item.Estimate))
		}
		if proven["current_bid"] && item.CurrentBid != nil {
			addSpecialistFact("current_bid", "Current bid", specialistMoney(item.Currency, *item.CurrentBid))
		}
		addSpecialistFact("lot_status", "Lot status", pointerString(item.LotStatus))
		addSpecialistFact("ruler", "Ruler", pointerString(item.Ruler))
		addSpecialistFact("denomination", "Denomination", pointerString(item.Denomination))
		addSpecialistFact("era", "Era", pointerString(item.Era))
		addSpecialistFact("material", "Material", pointerString(item.Material))
		public.Items = append(public.Items, projected)
	}
	if result.Trend != nil {
		public.Trend = &CopilotSpecialistPublicTrend{
			State: result.Trend.State, SampleSize: result.Trend.SampleSize,
			DateFrom: result.Trend.DateFrom, DateTo: result.Trend.DateTo,
			Currency: result.Trend.Currency, PriceBasis: result.Trend.PriceBasis,
			Low: result.Trend.Low, Median: result.Trend.Median, High: result.Trend.High,
			Confidence:          result.Trend.Confidence,
			Limitations:         sanitizeSpecialistStrings(result.Trend.Limitations, 500),
			SupportingSourceIDs: append([]string(nil), result.Trend.SupportingSourceIDs...),
		}
	}
	return public, nil
}

func pointerString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func specialistMoney(currency *string, amount float64) string {
	value := fmt.Sprintf("%.2f", amount)
	value = strings.TrimSuffix(strings.TrimSuffix(value, "0"), ".")
	if currency == nil || *currency == "" {
		return value
	}
	return *currency + " " + value
}

func sanitizeSpecialistStrings(values []string, maximum int) []string {
	sanitized := make([]string, len(values))
	for i, value := range values {
		sanitized[i] = SanitizeCopilotText(value, maximum)
	}
	return sanitized
}

func ValidateCopilotCheckpoint(state CopilotCheckpointState, run *models.CoinCopilotRun) error {
	if state.SchemaVersion != 1 || len(state.Messages) > CoinCopilotMaxMessages || len(state.Plan) > CoinCopilotMaxPlanItems {
		return ErrInvalidCopilotFrame
	}
	total := 0
	for _, message := range state.Messages {
		if (message.Role != "user" && message.Role != "assistant") || message.Content == "" {
			return ErrInvalidCopilotFrame
		}
		total += len(message.Content)
	}
	if total > CoinCopilotMaxMessageBytes {
		return ErrInvalidCopilotFrame
	}
	for _, item := range state.Plan {
		if item.ID == "" || item.Title == "" || len(item.Title) > CoinCopilotMaxPlanTitle {
			return ErrInvalidCopilotFrame
		}
		switch item.Status {
		case "pending", "in_progress", "completed", "skipped", "failed":
		default:
			return ErrInvalidCopilotFrame
		}
	}
	if state.PendingClarification != nil {
		if len(state.PendingClarification.Question) == 0 || len(state.PendingClarification.Question) > CoinCopilotMaxClarification ||
			len(state.PendingClarification.Choices) > 10 {
			return ErrInvalidCopilotFrame
		}
		switch state.PendingClarification.InputType {
		case "text", "single_choice", "boolean":
		default:
			return ErrInvalidCopilotFrame
		}
	}
	switch state.NextAction {
	case "continue", "await_clarification", "finish":
	default:
		return ErrInvalidCopilotFrame
	}
	if state.Counters.Iterations < 0 || state.Counters.ToolCalls < 0 ||
		state.Counters.InputTokens < 0 || state.Counters.OutputTokens < 0 ||
		state.Counters.Iterations > run.MaxIterations || state.Counters.ToolCalls > run.MaxToolCalls {
		return ErrInvalidCopilotFrame
	}
	seen := map[string]bool{}
	for _, tool := range state.CompletedTools {
		if tool.ToolCallID == "" || !IsCoinCopilotToolAllowed(tool.ToolName) || seen[tool.ToolCallID] {
			return ErrInvalidCopilotFrame
		}
		seen[tool.ToolCallID] = true
	}
	return nil
}

type DeepAnalysisHandoffTarget struct {
	Type string `json:"type"`
	ID   uint   `json:"id"`
}

type DeepAnalysisHandoffRequest struct {
	ToolCallID                string                     `json:"tool_call_id"`
	HandoffIdempotencyKey     string                     `json:"handoff_idempotency_key,omitempty"`
	ExpectedCheckpointVersion int64                      `json:"expected_checkpoint_version"`
	Operation                 string                     `json:"operation"`
	Target                    *DeepAnalysisHandoffTarget `json:"target,omitempty"`
	JobID                     *uint                      `json:"job_id,omitempty"`
}

type DeepAnalysisHandoffJob struct {
	ID          uint    `json:"id"`
	Source      string  `json:"source"`
	Status      string  `json:"status"`
	Reused      bool    `json:"reused"`
	CreatedAt   string  `json:"created_at"`
	CompletedAt *string `json:"completed_at"`
}

type DeepAnalysisHandoffEvidence struct {
	Provider string `json:"provider"`
	Source   string `json:"source"`
	URL      string `json:"url"`
	Summary  string `json:"summary"`
}

type DeepAnalysisHandoffField struct {
	Name       string                        `json:"name"`
	Value      string                        `json:"value"`
	Confidence float64                       `json:"confidence"`
	Evidence   []DeepAnalysisHandoffEvidence `json:"evidence"`
}

type DeepAnalysisHandoffDisagreement struct {
	Field   string `json:"field"`
	Summary string `json:"summary"`
}

type DeepAnalysisHandoffCoverage struct {
	Provider string `json:"provider"`
	Status   string `json:"status"`
}

type DeepAnalysisHandoffAttribution struct {
	Provider string `json:"provider"`
	Label    string `json:"label"`
}

type DeepAnalysisHandoffResultBody struct {
	State               string                            `json:"state"`
	Narrative           string                            `json:"narrative"`
	PartialSuccess      bool                              `json:"partial_success"`
	ImageOnly           bool                              `json:"image_only"`
	Fields              []DeepAnalysisHandoffField        `json:"fields"`
	Disagreements       []DeepAnalysisHandoffDisagreement `json:"disagreements"`
	UnresolvedQuestions []string                          `json:"unresolved_questions"`
	Coverage            []DeepAnalysisHandoffCoverage     `json:"coverage"`
	Attributions        []DeepAnalysisHandoffAttribution  `json:"attributions"`
	Limitations         []string                          `json:"limitations"`
}

type DeepAnalysisHandoffTruncation struct {
	Truncated            bool   `json:"truncated"`
	OriginalBytes        int    `json:"original_bytes"`
	PersistedBytes       int    `json:"persisted_bytes"`
	Digest               string `json:"digest"`
	OmittedFields        int    `json:"omitted_fields"`
	OmittedEvidence      int    `json:"omitted_evidence"`
	OmittedDisagreements int    `json:"omitted_disagreements"`
	OmittedQuestions     int    `json:"omitted_questions"`
}

type DeepAnalysisHandoffResult struct {
	SchemaVersion int     `json:"schema_version,omitempty"`
	Operation     string  `json:"operation,omitempty"`
	Outcome       string  `json:"outcome"`
	Reason        *string `json:"reason"`
	Target        *struct {
		Type         string `json:"type"`
		ID           uint   `json:"id"`
		DisplayLabel string `json:"display_label"`
	} `json:"target,omitempty"`
	Job                    *DeepAnalysisHandoffJob        `json:"job,omitempty"`
	InputDigest            string                         `json:"input_digest,omitempty"`
	ReviewURL              string                         `json:"review_url,omitempty"`
	FreshAnalysisAvailable bool                           `json:"fresh_analysis_available,omitempty"`
	Result                 *DeepAnalysisHandoffResultBody `json:"result,omitempty"`
	Truncation             *DeepAnalysisHandoffTruncation `json:"truncation,omitempty"`
	Limitations            []string                       `json:"limitations,omitempty"`
}

func ValidateDeepAnalysisHandoffRequestEnvelope(raw []byte) error {
	return validateDeepAnalysisHandoffEnvelope(raw, DeepAnalysisHandoffMaxRequestBytes)
}

func ValidateDeepAnalysisHandoffPublicEventEnvelope(raw []byte) error {
	return validateDeepAnalysisHandoffEnvelope(raw, DeepAnalysisHandoffMaxPublicEventBytes)
}

func validateDeepAnalysisHandoffEnvelope(raw []byte, maximum int) error {
	if len(raw) == 0 || len(raw) > maximum || !json.Valid(raw) ||
		forbiddenCopilotField.Match(raw) || tokenShapedValue.Match(raw) {
		return ErrInvalidCopilotFrame
	}
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return ErrInvalidCopilotFrame
	}
	canonical, err := json.Marshal(sanitizeCopilotValue(value))
	if err != nil || len(canonical) > maximum {
		return ErrInvalidCopilotFrame
	}
	return nil
}

func DecodeDeepAnalysisHandoffRequest(raw []byte) (DeepAnalysisHandoffRequest, error) {
	var request DeepAnalysisHandoffRequest
	if err := ValidateDeepAnalysisHandoffRequestEnvelope(raw); err != nil {
		return request, err
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		return request, ErrInvalidCopilotFrame
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return request, ErrInvalidCopilotFrame
	}
	if err := ValidateDeepAnalysisHandoffRequest(request); err != nil {
		return request, err
	}
	return request, nil
}

func ValidateDeepAnalysisHandoffRequest(request DeepAnalysisHandoffRequest) error {
	if len(request.ToolCallID) < 1 || len(request.ToolCallID) > 200 ||
		request.ExpectedCheckpointVersion < 0 {
		return ErrInvalidCopilotFrame
	}
	if request.Target != nil &&
		(request.Target.ID == 0 || (request.Target.Type != "coin" && request.Target.Type != "draft")) {
		return ErrInvalidCopilotFrame
	}
	hasKey := request.HandoffIdempotencyKey != ""
	if hasKey && (!printableASCII(request.HandoffIdempotencyKey) || len(request.HandoffIdempotencyKey) > 128) {
		return ErrInvalidCopilotFrame
	}
	switch request.Operation {
	case "request":
		if request.Target == nil || request.JobID != nil || !hasKey {
			return ErrInvalidCopilotFrame
		}
	case "status":
		if request.Target != nil || request.JobID == nil || *request.JobID == 0 || hasKey {
			return ErrInvalidCopilotFrame
		}
	case "rerun":
		if request.Target == nil || request.JobID == nil || *request.JobID == 0 || !hasKey {
			return ErrInvalidCopilotFrame
		}
	default:
		return ErrInvalidCopilotFrame
	}
	return nil
}

func printableASCII(value string) bool {
	for _, character := range []byte(value) {
		if character < 0x20 || character > 0x7e {
			return false
		}
	}
	return true
}

func DecodeDeepAnalysisHandoffResult(raw []byte, maximum int) (DeepAnalysisHandoffResult, error) {
	var result DeepAnalysisHandoffResult
	if err := validateDeepAnalysisHandoffEnvelope(raw, maximum); err != nil {
		return result, err
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&result); err != nil {
		return result, ErrInvalidCopilotFrame
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return result, ErrInvalidCopilotFrame
	}
	if err := ValidateDeepAnalysisHandoffResult(result); err != nil {
		return result, err
	}
	return result, nil
}

func ValidateDeepAnalysisHandoffResult(result DeepAnalysisHandoffResult) error {
	if !oneOf(result.Outcome,
		"accepted", "reused_active", "reused_result", "status", "retry_available",
		"missing_images", "target_unavailable", "not_eligible", "unavailable", "cancelled") {
		return ErrInvalidCopilotFrame
	}
	if result.Reason != nil && !oneOf(*result.Reason,
		"missing_obverse", "missing_reverse", "missing_both", "duplicate_faces",
		"target_changed", "draft_inactive", "source_coin_missing", "deep_disabled",
		"copilot_disabled", "attribution_disabled", "model_unsupported", "job_at_capacity",
		"queue_full", "result_missing", "result_expired", "stale", "cancelled") {
		return ErrInvalidCopilotFrame
	}
	if result.Outcome == "not_eligible" || result.Outcome == "target_unavailable" {
		if result.Reason != nil || result.SchemaVersion != 0 || result.Operation != "" ||
			result.Target != nil || result.Job != nil || result.InputDigest != "" ||
			result.ReviewURL != "" || result.Result != nil || result.Truncation != nil ||
			len(result.Limitations) != 0 {
			return ErrInvalidCopilotFrame
		}
		return nil
	}
	if result.SchemaVersion != 1 || !oneOf(result.Operation, "request", "status", "rerun") {
		return ErrInvalidCopilotFrame
	}
	if result.Target != nil &&
		(result.Target.ID == 0 || !oneOf(result.Target.Type, "coin", "draft") || result.Target.DisplayLabel == "") {
		return ErrInvalidCopilotFrame
	}
	if result.Job != nil {
		if result.Job.ID == 0 || !oneOf(result.Job.Source, "intake", "saved_coin", "copilot_draft") ||
			!oneOf(result.Job.Status, "queued", "running", "completed", "partial", "failed", "cancelled") ||
			result.Job.CreatedAt == "" {
			return ErrInvalidCopilotFrame
		}
		if result.ReviewURL != fmt.Sprintf("/deep-analysis/%d", result.Job.ID) {
			return ErrInvalidCopilotFrame
		}
	} else if result.ReviewURL != "" {
		return ErrInvalidCopilotFrame
	}
	if result.InputDigest != "" && !lowerSHA256.MatchString(result.InputDigest) {
		return ErrInvalidCopilotFrame
	}
	if result.Result != nil {
		if !oneOf(result.Result.State,
			"not_ready", "complete", "partial", "no_match", "failed", "cancelled", "stale", "missing_result") {
			return ErrInvalidCopilotFrame
		}
		fieldNames := map[string]bool{}
		for _, field := range result.Result.Fields {
			if field.Name == "" || field.Value == "" || math.IsNaN(field.Confidence) ||
				math.IsInf(field.Confidence, 0) || field.Confidence < 0 || field.Confidence > 1 ||
				fieldNames[field.Name] {
				return ErrInvalidCopilotFrame
			}
			fieldNames[field.Name] = true
			evidenceKeys := map[string]bool{}
			for _, evidence := range field.Evidence {
				evidenceKey := evidence.Provider + "\x00" + evidence.Source + "\x00" + evidence.URL
				if evidence.Provider == "" || evidence.Source == "" || evidence.Summary == "" ||
					!safeDeepAnalysisCitation(evidence.URL) || evidenceKeys[evidenceKey] {
					return ErrInvalidCopilotFrame
				}
				evidenceKeys[evidenceKey] = true
			}
		}
		coverageProviders := map[string]bool{}
		for _, coverage := range result.Result.Coverage {
			if !oneOf(coverage.Provider, "numista", "nomisma", "ngc", "ocre", "rpc") ||
				!oneOf(coverage.Status, "pending", "running", "contributed", "no_match", "failed",
					"timed_out", "skipped", "not_automated", "unavailable") ||
				coverageProviders[coverage.Provider] {
				return ErrInvalidCopilotFrame
			}
			coverageProviders[coverage.Provider] = true
		}
		attributionProviders := map[string]bool{}
		for _, attribution := range result.Result.Attributions {
			if !oneOf(attribution.Provider, "numista", "nomisma", "ngc", "ocre", "rpc") ||
				attribution.Label == "" || attributionProviders[attribution.Provider] {
				return ErrInvalidCopilotFrame
			}
			attributionProviders[attribution.Provider] = true
		}
	}
	if result.Truncation != nil {
		metadata := result.Truncation
		if metadata.OriginalBytes < 0 || metadata.PersistedBytes < 0 ||
			metadata.PersistedBytes > DeepAnalysisHandoffMaxPersistedResultBytes ||
			!lowerSHA256.MatchString(metadata.Digest) ||
			metadata.OmittedFields < 0 || metadata.OmittedEvidence < 0 ||
			metadata.OmittedDisagreements < 0 || metadata.OmittedQuestions < 0 {
			return ErrInvalidCopilotFrame
		}
	}
	return nil
}

var lowerSHA256 = regexp.MustCompile(`^[0-9a-f]{64}$`)

func oneOf(value string, allowed ...string) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
}

func safeDeepAnalysisCitation(raw string) bool {
	return validSpecialistURL(raw)
}
