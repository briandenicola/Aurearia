package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// CoinLookupService handles coin lookup from images with NGC cert extraction and coin draft enrichment.
type CoinLookupService struct {
	proxy       *AgentProxy
	settingsSvc *SettingsService
	logger      *Logger
	client      *http.Client
}

func NewCoinLookupService(proxy *AgentProxy, settingsSvc *SettingsService, logger *Logger) *CoinLookupService {
	return &CoinLookupService{
		proxy:       proxy,
		settingsSvc: settingsSvc,
		logger:      logger,
		client:      &http.Client{Timeout: 15 * time.Second},
	}
}

// CoinLookupRequest wraps the input for coin lookup.
type CoinLookupRequest struct {
	Images []string `json:"images"` // Data URIs
}

// LookupExtractedData represents extracted data from vision analysis.
type LookupExtractedData struct {
	NGC         *NGCData       `json:"ngc,omitempty"`
	LabelText   string         `json:"labelText,omitempty"`
	CoinFields  map[string]any `json:"coinFields,omitempty"`
	Confidence  string         `json:"confidence"`
	RawAnalysis string         `json:"rawAnalysis"`
}

// NGCData represents extracted NGC certification data.
type NGCData struct {
	CertNumber     string `json:"certNumber"`
	NormalizedCert string `json:"normalizedCert"`
	LookupURL      string `json:"lookupURL"`
	Grade          string `json:"grade,omitempty"`
	Description    string `json:"description,omitempty"`
}

// NumistaCandidate represents a Numista search result.
type NumistaCandidate struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Issuer    string `json:"issuer"`
	Year      string `json:"year"`
	Thumbnail string `json:"thumbnail,omitempty"`
	URL       string `json:"url"`
}

type numistaSearchResponse struct {
	Types []numistaType `json:"types"`
}

type numistaType struct {
	ID               int            `json:"id"`
	Title            string         `json:"title"`
	Issuer           *numistaIssuer `json:"issuer,omitempty"`
	MinYear          *int           `json:"min_year,omitempty"`
	MaxYear          *int           `json:"max_year,omitempty"`
	ObverseThumbnail string         `json:"obverse_thumbnail,omitempty"`
	ReverseThumbnail string         `json:"reverse_thumbnail,omitempty"`
}

type numistaIssuer struct {
	Name string `json:"name"`
}

// CoinLookupResponse is the final response returned to the client.
type CoinLookupResponse struct {
	ExtractedData       LookupExtractedData       `json:"extractedData"`
	NumistaCandidates   []NumistaCandidate        `json:"numistaCandidates"`
	PrefilledDraft      map[string]any            `json:"prefilledDraft,omitempty"`
	CandidateReferences []CandidateReferenceProxy `json:"candidateReferences,omitempty"`
}

var (
	// Matches NGC cert formats: 823160-093, 1234567-001, 2412821034, etc.
	ngcCertRegex      = regexp.MustCompile(`\b(\d{6,7}-?\d{3})\b`)
	ngcCertExactRegex = regexp.MustCompile(`^(\d{6,7})-?(\d{3})$`)
	compactCertRegex  = regexp.MustCompile(`^\d{9,10}$`)
)

// Lookup performs coin lookup from images: extracts NGC certs first, then falls back to structured coin fields.
func (s *CoinLookupService) Lookup(ctx context.Context, userID uint, req CoinLookupRequest) (*CoinLookupResponse, error) {
	logger := s.logger
	logger.Info("coin-lookup", "Starting lookup for user %d with %d images", userID, len(req.Images))

	if len(req.Images) == 0 {
		return nil, fmt.Errorf("at least one image is required")
	}

	// 1. Vision analysis to extract NGC cert, label text, and coin fields
	extractedData, err := s.extractDataFromImages(ctx, req.Images)
	if err != nil {
		logger.Error("coin-lookup", "Vision analysis failed: %v", err)
		return nil, fmt.Errorf("vision analysis failed: %w", err)
	}

	logger.Info("coin-lookup", "Extracted data: NGC=%v, LabelText=%v", extractedData.NGC != nil, extractedData.LabelText != "")

	// 2. Build Numista search query from extracted fields.
	// NGC-slab lookups should return as soon as cert extraction succeeds; Numista
	// enrichment is only needed when there is no cert to verify directly.
	numistaCandidates := []NumistaCandidate{}
	if extractedData.NGC == nil && extractedData.CoinFields != nil {
		candidates, err := s.searchNumista(ctx, extractedData.CoinFields)
		if err != nil {
			logger.Warn("coin-lookup", "Numista search failed: %v", err)
		} else {
			numistaCandidates = candidates
			logger.Info("coin-lookup", "Found %d Numista candidates", len(numistaCandidates))
		}
	}

	// 3. Build prefilled draft for "Add to Collection/Wishlist"
	prefilledDraft := s.buildPrefilledDraft(extractedData, numistaCandidates)

	// 4. Build candidate references from NGC and Numista
	candidateReferences := s.buildCandidateReferences(extractedData, numistaCandidates)

	return &CoinLookupResponse{
		ExtractedData:       *extractedData,
		NumistaCandidates:   numistaCandidates,
		PrefilledDraft:      prefilledDraft,
		CandidateReferences: candidateReferences,
	}, nil
}

// extractDataFromImages uses vision analysis to extract NGC cert, label text, and coin fields.
func (s *CoinLookupService) extractDataFromImages(ctx context.Context, images []string) (*LookupExtractedData, error) {
	logger := s.logger

	// Resolve LLM config
	llmCfg, err := s.settingsSvc.ResolveLLMConfig()
	if err != nil {
		return nil, fmt.Errorf("unable to configure LLM: %w", err)
	}

	// Build vision analysis prompt
	prompt := s.buildVisionPrompt()

	// Call agent proxy for vision analysis
	proxyReq := AnalyzeProxyRequest{
		LLM: llmCfg,
		Coin: CoinDataProxy{
			Name: "Lookup Candidate",
		},
		Images: images,
		Side:   "lookup",
		Prompt: prompt,
	}

	analysis, err := s.proxy.AnalyzeCoin(ctx, proxyReq)
	if err != nil {
		return nil, fmt.Errorf("agent proxy failed: %w", err)
	}

	logger.Debug("coin-lookup", "Raw vision analysis: %s", analysis)

	// Parse the analysis response
	extractedData := &LookupExtractedData{
		RawAnalysis: analysis,
		Confidence:  "medium",
		CoinFields:  make(map[string]any),
	}

	// Extract NGC cert number
	ngcData := s.extractNGCCert(analysis)
	if ngcData != nil {
		extractedData.NGC = ngcData
		logger.Info("coin-lookup", "Extracted NGC cert: %s", ngcData.NormalizedCert)
	}

	// Extract label text (any visible text)
	labelText := s.extractLabelText(analysis)
	if labelText != "" {
		extractedData.LabelText = labelText
		logger.Debug("coin-lookup", "Extracted label text: %s", labelText)
	}

	// Extract coin fields (ruler, era, denomination, material, category)
	coinFields := s.extractCoinFields(analysis)
	if len(coinFields) > 0 {
		extractedData.CoinFields = coinFields
		logger.Debug("coin-lookup", "Extracted coin fields: %d", len(coinFields))
	}

	// Determine confidence
	extractedData.Confidence = s.determineConfidence(extractedData)

	return extractedData, nil
}

// buildVisionPrompt creates a specialized prompt for coin lookup vision analysis.
func (s *CoinLookupService) buildVisionPrompt() string {
	return `You are analyzing a coin or coin slab photo for lookup purposes. Extract:

1. NGC Certification: If this is an NGC slab/holder or the image shows an NGC certification number, extract the NGC certification number (format: XXXXXXX-XXX, e.g., 823160-093 or 1234567-001). Also extract the grade (e.g., "Ch AU", "NGC AU", etc.) and any description text on the label.

2. Visible Text: Extract ALL visible text from the image (inscriptions, labels, holder text, etc.). Be thorough.

3. Coin Attribution: If no NGC slab/cert is visible, analyze the raw coin image like the standard Add Coin image analysis and infer as many draft fields as possible:
   - Name/title (short collector-friendly attribution)
   - Ruler (e.g., "Augustus", "Constantine I", "Philip II")
   - Mint if visible or confidently inferable
   - Era (ancient, medieval, or modern)
   - Denomination (e.g., "Denarius", "Tetradrachm", "Solidus")
   - Material (Gold, Silver, Bronze, Copper, Electrum, Other)
   - Category (Roman, Greek, Byzantine, Modern, Other)
   - Obverse inscription
   - Reverse inscription
   - Obverse description
   - Reverse description
   - Weight in grams and diameter in mm only if visible in text
   - Rarity rating or grade only if visible

Return your response in this EXACT JSON format (no markdown, no extra text):
{
  "ngcCert": "XXXXXXX-XXX or null",
  "ngcGrade": "grade text or null",
  "ngcDescription": "description or null",
  "labelText": "all visible text here",
  "name": "short coin attribution or null",
  "ruler": "ruler name or null",
  "mint": "mint name or null",
  "era": "ancient/medieval/modern or null",
  "denomination": "denomination or null",
  "material": "material or null",
  "category": "category or null",
  "obverseInscription": "obverse legend or null",
  "reverseInscription": "reverse legend or null",
  "obverseDescription": "obverse design or null",
  "reverseDescription": "reverse design or null",
  "weightGrams": number or null,
  "diameterMm": number or null,
  "rarityRating": "rarity text or null",
  "grade": "grade text or null",
  "confidence": "high/medium/low"
}

Be precise. If uncertain, use null. Prefer NGC cert extraction when a slab/cert is present; otherwise return the best standard coin image draft you can infer.`
}

// extractNGCCert parses NGC certification data from analysis text.
func (s *CoinLookupService) extractNGCCert(analysis string) *NGCData {
	// Try to parse as JSON first
	var parsed map[string]any
	if err := json.Unmarshal([]byte(analysis), &parsed); err == nil {
		if certStr, ok := parsed["ngcCert"].(string); ok && certStr != "" && certStr != "null" {
			normalized := normalizeCertNumber(certStr)
			if normalized != "" {
				ngcData := &NGCData{
					CertNumber:     certStr,
					NormalizedCert: normalized,
					LookupURL:      ngcLookupURL(normalized),
				}
				if grade, ok := parsed["ngcGrade"].(string); ok && grade != "" && grade != "null" {
					ngcData.Grade = grade
				}
				if desc, ok := parsed["ngcDescription"].(string); ok && desc != "" && desc != "null" {
					ngcData.Description = desc
				}
				return ngcData
			}
		}
	}

	// Fallback: regex search in raw text
	matches := ngcCertRegex.FindStringSubmatch(analysis)
	if len(matches) > 1 {
		certNumber := matches[1]
		normalized := normalizeCertNumber(certNumber)
		if normalized != "" {
			return &NGCData{
				CertNumber:     certNumber,
				NormalizedCert: normalized,
				LookupURL:      ngcLookupURL(normalized),
			}
		}
	}

	return nil
}

// normalizeCertNumber normalizes NGC cert numbers (e.g., 823160093 -> 823160-093).
func normalizeCertNumber(cert string) string {
	cert = strings.TrimSpace(cert)
	// Remove any extra spaces or formatting
	cert = strings.ReplaceAll(cert, " ", "")

	matches := ngcCertExactRegex.FindStringSubmatch(cert)
	if len(matches) == 3 {
		return matches[1] + "-" + matches[2]
	}
	return ""
}

func compactCertNumber(cert string) string {
	cert = strings.TrimSpace(cert)
	cert = strings.ReplaceAll(cert, " ", "")
	cert = strings.ReplaceAll(cert, "-", "")
	if compactCertRegex.MatchString(cert) {
		return cert
	}
	return ""
}

func ngcLookupURL(cert string) string {
	compact := compactCertNumber(cert)
	if compact == "" {
		return ""
	}
	return fmt.Sprintf("https://www.ngccoin.com/certlookup/%s/NGCAncients/", url.PathEscape(compact))
}

// extractLabelText extracts visible label text from analysis.
func (s *CoinLookupService) extractLabelText(analysis string) string {
	var parsed map[string]any
	if err := json.Unmarshal([]byte(analysis), &parsed); err == nil {
		if labelText, ok := parsed["labelText"].(string); ok && labelText != "" && labelText != "null" {
			return strings.TrimSpace(labelText)
		}
	}
	return ""
}

// extractCoinFields extracts structured coin fields from analysis.
func (s *CoinLookupService) extractCoinFields(analysis string) map[string]any {
	fields := make(map[string]any)

	var parsed map[string]any
	if err := json.Unmarshal([]byte(analysis), &parsed); err != nil {
		return fields
	}

	copyCoinFieldsFromMap(parsed, fields)
	if nested, ok := parsed["coin"].(map[string]any); ok {
		copyCoinFieldsFromMap(nested, fields)
	}

	return fields
}

func copyCoinFieldsFromMap(source map[string]any, target map[string]any) {
	stringFields := map[string][]string{
		"name":               {"name"},
		"ruler":              {"ruler"},
		"mint":               {"mint"},
		"era":                {"era"},
		"denomination":       {"denomination"},
		"material":           {"material"},
		"category":           {"category"},
		"obverseInscription": {"obverseInscription", "obverse_inscription"},
		"reverseInscription": {"reverseInscription", "reverse_inscription"},
		"obverseDescription": {"obverseDescription", "obverse_description"},
		"reverseDescription": {"reverseDescription", "reverse_description"},
		"rarityRating":       {"rarityRating", "rarity_rating"},
		"grade":              {"grade"},
		"notes":              {"notes"},
		"referenceText":      {"referenceText", "reference_text"},
		"referenceUrl":       {"referenceUrl", "reference_url"},
	}
	for targetKey, sourceKeys := range stringFields {
		for _, sourceKey := range sourceKeys {
			if copyStringFieldAs(source, target, sourceKey, targetKey) {
				break
			}
		}
	}

	numberFields := map[string][]string{
		"weightGrams": {"weightGrams", "weight_grams"},
		"diameterMm":  {"diameterMm", "diameter_mm"},
	}
	for targetKey, sourceKeys := range numberFields {
		for _, sourceKey := range sourceKeys {
			if copyNumberFieldAs(source, target, sourceKey, targetKey) {
				break
			}
		}
	}
}

func copyStringFieldAs(source map[string]any, target map[string]any, sourceKey string, targetKey string) bool {
	if _, exists := target[targetKey]; exists {
		return true
	}
	value, ok := source[sourceKey].(string)
	if !ok {
		return false
	}
	value = strings.TrimSpace(value)
	if value == "" || strings.EqualFold(value, "null") {
		return false
	}
	target[targetKey] = value
	return true
}

func copyNumberFieldAs(source map[string]any, target map[string]any, sourceKey string, targetKey string) bool {
	if _, exists := target[targetKey]; exists {
		return true
	}
	switch value := source[sourceKey].(type) {
	case float64:
		if value > 0 {
			target[targetKey] = value
			return true
		}
	case string:
		value = strings.TrimSpace(value)
		if value != "" && !strings.EqualFold(value, "null") {
			target[targetKey] = value
			return true
		}
	}
	return false
}

// determineConfidence assesses overall extraction confidence.
func (s *CoinLookupService) determineConfidence(data *LookupExtractedData) string {
	score := 0
	if data.NGC != nil {
		score += 3
	}
	if data.LabelText != "" {
		score += 2
	}
	if len(data.CoinFields) >= 3 {
		score += 3
	} else if len(data.CoinFields) >= 1 {
		score += 1
	}

	if score >= 6 {
		return "high"
	} else if score >= 3 {
		return "medium"
	}
	return "low"
}

// searchNumista queries Numista API with extracted coin fields.
func (s *CoinLookupService) searchNumista(ctx context.Context, coinFields map[string]any) ([]NumistaCandidate, error) {
	logger := s.logger

	apiKey := s.settingsSvc.GetSetting(SettingNumistaAPIKey)
	if apiKey == "" {
		return nil, fmt.Errorf("Numista API key not configured")
	}

	// Build search query from coin fields
	query := s.buildNumistaQuery(coinFields)
	if query == "" {
		logger.Warn("coin-lookup", "No Numista search query could be built from fields")
		return nil, fmt.Errorf("insufficient data for Numista search")
	}

	logger.Info("coin-lookup", "Numista query: %s", query)

	numistaURL := fmt.Sprintf("https://api.numista.com/v3/types?q=%s&category=coin&count=5&lang=en", url.QueryEscape(query))
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, numistaURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Numista request: %w", err)
	}
	httpReq.Header.Set("Numista-API-Key", apiKey)

	resp, err := s.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to reach Numista API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read Numista response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Numista API returned status %d", resp.StatusCode)
	}

	var result numistaSearchResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse Numista response: %w", err)
	}

	candidates := make([]NumistaCandidate, 0, len(result.Types))
	for _, item := range result.Types {
		candidate := NumistaCandidate{
			ID:    fmt.Sprintf("%d", item.ID),
			Title: item.Title,
			URL:   fmt.Sprintf("https://en.numista.com/catalogue/pieces%d.html", item.ID),
			Year:  formatNumistaYearRange(item.MinYear, item.MaxYear),
		}
		if item.Issuer != nil {
			candidate.Issuer = item.Issuer.Name
		}
		if item.ObverseThumbnail != "" {
			candidate.Thumbnail = item.ObverseThumbnail
		} else {
			candidate.Thumbnail = item.ReverseThumbnail
		}
		candidates = append(candidates, candidate)
	}

	return candidates, nil
}

func formatNumistaYearRange(minYear, maxYear *int) string {
	if minYear == nil && maxYear == nil {
		return ""
	}
	if minYear != nil && maxYear != nil {
		if *minYear == *maxYear {
			return fmt.Sprintf("%d", *minYear)
		}
		return fmt.Sprintf("%d-%d", *minYear, *maxYear)
	}
	if minYear != nil {
		return fmt.Sprintf("%d", *minYear)
	}
	return fmt.Sprintf("%d", *maxYear)
}

// buildNumistaQuery constructs a Numista search query from extracted fields.
func (s *CoinLookupService) buildNumistaQuery(coinFields map[string]any) string {
	parts := []string{}

	if ruler, ok := coinFields["ruler"].(string); ok && ruler != "" {
		parts = append(parts, ruler)
	}
	if denom, ok := coinFields["denomination"].(string); ok && denom != "" {
		parts = append(parts, denom)
	}
	if era, ok := coinFields["era"].(string); ok && era != "" {
		parts = append(parts, era)
	}

	return strings.Join(parts, " ")
}

// buildPrefilledDraft creates a prefilled draft for Add to Collection/Wishlist.
func (s *CoinLookupService) buildPrefilledDraft(data *LookupExtractedData, candidates []NumistaCandidate) map[string]any {
	draft := make(map[string]any)

	// Use extracted coin fields
	if data.CoinFields != nil {
		for k, v := range data.CoinFields {
			draft[k] = v
		}
	}

	// Use top Numista candidate if available
	if len(candidates) > 0 {
		top := candidates[0]
		if _, ok := draft["name"]; !ok {
			draft["name"] = top.Title
		}
		draft["numista_id"] = top.ID
		draft["numista_url"] = top.URL
	}

	if _, ok := draft["name"]; !ok {
		draft["name"] = fallbackDraftName(data.CoinFields)
	}

	if data.RawAnalysis != "" {
		if _, ok := draft["aiAnalysis"]; !ok {
			draft["aiAnalysis"] = data.RawAnalysis
		}
	}

	// Include NGC data in notes if present
	if data.NGC != nil {
		notes := fmt.Sprintf("NGC Cert: %s\n", data.NGC.NormalizedCert)
		if data.NGC.Grade != "" {
			notes += fmt.Sprintf("Grade: %s\n", data.NGC.Grade)
		}
		if data.NGC.Description != "" {
			notes += fmt.Sprintf("Description: %s\n", data.NGC.Description)
		}
		notes += fmt.Sprintf("Lookup URL: %s\n", data.NGC.LookupURL)
		draft["notes"] = notes
	}

	return draft
}

func fallbackDraftName(fields map[string]any) string {
	if fields == nil {
		return "Unidentified Coin"
	}
	parts := []string{}
	if ruler, ok := fields["ruler"].(string); ok && strings.TrimSpace(ruler) != "" {
		parts = append(parts, strings.TrimSpace(ruler))
	}
	if denomination, ok := fields["denomination"].(string); ok && strings.TrimSpace(denomination) != "" {
		parts = append(parts, strings.TrimSpace(denomination))
	}
	if len(parts) == 0 {
		return "Unidentified Coin"
	}
	return strings.Join(parts, " ")
}

// buildCandidateReferences creates CoinReference-compatible data for Add to Wishlist.
func (s *CoinLookupService) buildCandidateReferences(data *LookupExtractedData, candidates []NumistaCandidate) []CandidateReferenceProxy {
	refs := []CandidateReferenceProxy{}

	// Add NGC reference if present
	if data.NGC != nil {
		refs = append(refs, CandidateReferenceProxy{
			Catalog: "NGC",
			Number:  data.NGC.NormalizedCert,
			URI:     data.NGC.LookupURL,
		})
	}

	// Add Numista references
	for _, candidate := range candidates {
		refs = append(refs, CandidateReferenceProxy{
			Catalog: "Numista",
			Number:  candidate.ID,
			URI:     candidate.URL,
		})
	}

	return refs
}
