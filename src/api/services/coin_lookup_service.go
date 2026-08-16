package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/briandenicola/ancient-coins-api/models"
)

// CoinLookupService handles quick coin lookup from images with NGC cert extraction and minimum draft enrichment.
type CoinLookupService struct {
	proxy        *AgentProxy
	settingsSvc  *SettingsService
	logger       *Logger
	queryBuilder *NumistaQueryBuilder
}

func NewCoinLookupService(
	proxy *AgentProxy,
	settingsSvc *SettingsService,
	logger *Logger,
	builders ...*NumistaQueryBuilder,
) *CoinLookupService {
	builder := NewNumistaQueryBuilder()
	if len(builders) > 0 && builders[0] != nil {
		builder = builders[0]
	}
	return &CoinLookupService{
		proxy:        proxy,
		settingsSvc:  settingsSvc,
		logger:       logger,
		queryBuilder: builder,
	}
}

// CoinLookupRequest wraps the input for coin lookup.
type CoinLookupRequest struct {
	Images     []string `json:"images"` // Data URIs
	ImageRoles []string `json:"imageRoles,omitempty"`
	Notes      string   `json:"notes,omitempty"`
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

// CoinLookupResponse is the final response returned to the client.
type CoinLookupResponse struct {
	ExtractedData        LookupExtractedData          `json:"extractedData"`
	NumistaCandidates    []NumistaCandidate           `json:"numistaCandidates"`
	ProposedNumistaQuery string                       `json:"proposedNumistaQuery,omitempty"`
	NumistaEvidence      models.NumistaEvidence       `json:"numistaEvidence"`
	NumistaLookup        *models.NumistaLookupOutcome `json:"numistaLookup"`
	PrefilledDraft       map[string]any               `json:"prefilledDraft,omitempty"`
	CandidateReferences  []CandidateReferenceProxy    `json:"candidateReferences,omitempty"`
}

var (
	// Matches NGC cert formats: 823160-093, 1234567-001, 2412821034, etc.
	ngcCertRegex      = regexp.MustCompile(`\b(\d{6,7}-?\d{3})\b`)
	ngcCertExactRegex = regexp.MustCompile(`^(\d{6,7})-?(\d{3})$`)
	compactCertRegex  = regexp.MustCompile(`^\d{9,10}$`)
)

// Lookup performs coin lookup from images: extracts NGC certs first, then falls back to minimum structured draft fields.
func (s *CoinLookupService) Lookup(ctx context.Context, userID uint, req CoinLookupRequest) (*CoinLookupResponse, error) {
	logger := s.logger
	logger.Info("coin-lookup", "Starting lookup for user %d with %d images", userID, len(req.Images))

	if len(req.Images) == 0 {
		return nil, fmt.Errorf("at least one image is required")
	}

	// 1. Vision analysis to extract NGC cert, label text, and coin fields
	extractedData, err := s.extractDataFromImages(ctx, req.Images, req.ImageRoles, req.Notes)
	if err != nil {
		logger.Error("coin-lookup", "Vision analysis failed: %v", err)
		return nil, fmt.Errorf("vision analysis failed: %w", err)
	}

	logger.Info("coin-lookup", "Extracted data: NGC=%v, LabelText=%v", extractedData.NGC != nil, extractedData.LabelText != "")

	numistaCandidates := []NumistaCandidate{}
	evidence := models.NumistaEvidence{}
	proposedQuery := ""
	if extractedData.NGC == nil {
		evidence = buildPhotoNumistaEvidence(extractedData)
		proposedQuery = s.queryBuilder.Build(evidence).Primary
	}

	// 3. Build prefilled draft for "Add to Collection/Wishlist"
	prefilledDraft := s.buildPrefilledDraft(extractedData, nil)

	// 4. Deprecated aliases retain NGC compatibility but never expose unselected Numista results.
	candidateReferences := s.buildCandidateReferences(extractedData, nil)

	return &CoinLookupResponse{
		ExtractedData:        *extractedData,
		NumistaCandidates:    numistaCandidates,
		ProposedNumistaQuery: proposedQuery,
		NumistaEvidence:      evidence,
		NumistaLookup:        nil,
		PrefilledDraft:       prefilledDraft,
		CandidateReferences:  candidateReferences,
	}, nil
}

// extractDataFromImages uses vision analysis to extract NGC cert, label text, and coin fields.
func (s *CoinLookupService) extractDataFromImages(ctx context.Context, images, imageRoles []string, notes string) (*LookupExtractedData, error) {
	logger := s.logger

	// Resolve LLM config
	llmCfg, err := s.settingsSvc.ResolveLLMConfig()
	if err != nil {
		return nil, fmt.Errorf("unable to configure LLM: %w", err)
	}

	// Build vision analysis prompt
	prompt := s.buildVisionPrompt(imageRoles)

	// Call agent proxy for vision analysis
	formatOutput := false
	proxyReq := AnalyzeProxyRequest{
		LLM: llmCfg,
		Coin: CoinDataProxy{
			Name:  "Lookup Candidate",
			Notes: notes,
		},
		Images:       images,
		Side:         "lookup",
		Prompt:       prompt,
		FormatOutput: &formatOutput,
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

// buildVisionPrompt creates a specialized prompt for quick coin lookup vision analysis.
func (s *CoinLookupService) buildVisionPrompt(imageRoles []string) string {
	prompt := `You are analyzing a coin or coin slab photo for a quick capture draft. Be fast and conservative. Extract only details visible or strongly inferable from the image.

1. NGC Certification: If this is an NGC slab/holder or the image shows an NGC certification number, extract the NGC certification number (format: XXXXXXX-XXX, e.g., 823160-093 or 1234567-001). Also extract the grade (e.g., "Ch AU", "NGC AU", etc.) and any description text on the label.

2. Visible Text: Extract visible slab/label text and obvious coin inscriptions. Do not attempt a full transcription if unclear.

3. Minimum Coin Draft: Infer only the minimum useful draft fields:
   - Name/title (short collector-friendly attribution)
   - Ruler (e.g., "Augustus", "Constantine I", "Philip II")
   - Era (ancient, medieval, or modern)
   - Denomination (e.g., "Denarius", "Tetradrachm", "Solidus")
   - Date or date range when visible or strongly inferable
   - Material (Gold, Silver, Bronze, Copper, Electrum, Other)
   - Category (Roman, Greek, Byzantine, Modern, Other)
   - One short obverse description
   - One short reverse description if a reverse image is present
   - Grade only if visible on a slab/label

Return your response in this EXACT JSON format (no markdown, no extra text):
{
  "ngcCert": "XXXXXXX-XXX or null",
  "ngcGrade": "grade text or null",
  "ngcDescription": "description or null",
  "labelText": "all visible text here",
  "name": "short coin attribution or null",
  "ruler": "ruler name or null",
  "mint": null,
  "era": "ancient/medieval/modern or null",
  "denomination": "denomination or null",
  "dateRange": "date or date range text or null",
  "material": "material or null",
  "category": "category or null",
  "obverseInscription": "obverse legend or null",
  "reverseInscription": "reverse legend or null",
  "obverseDescription": "short obverse design or null",
  "reverseDescription": "short reverse design or null",
  "weightGrams": null,
  "diameterMm": null,
  "rarityRating": null,
  "grade": "grade text or null",
  "confidence": "high/medium/low"
}

Be precise. If uncertain, use null. Do not include long history, market analysis, catalog references, or broad commentary. Prefer NGC cert extraction when a slab/cert is present; otherwise return the smallest useful draft.`
	if len(imageRoles) == 0 {
		return prompt
	}

	roles := make([]string, 0, len(imageRoles))
	for index, role := range imageRoles {
		roles = append(roles, fmt.Sprintf("image %d is %s", index+1, role))
	}
	return prompt + "\n\nImage roles: " + strings.Join(roles, "; ") + "."
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
	for _, line := range strings.Split(analysis, "\n") {
		parts := strings.SplitN(strings.TrimSpace(line), ":", 2)
		if len(parts) == 2 && strings.EqualFold(strings.TrimSpace(parts[0]), "label text") {
			value := cleanInferredValue(parts[1])
			if value != "" && !isNonSpecificExtraction(value) {
				return value
			}
		}
	}
	if strings.Contains(analysis, "/") {
		for _, line := range strings.Split(analysis, "\n") {
			line = cleanInferredValue(line)
			upper := strings.ToUpper(line)
			if strings.Count(line, "/") >= 2 &&
				(strings.Contains(upper, "NGC") || strings.Contains(upper, "EMPIRE") || strings.Contains(upper, "MINT")) {
				return line
			}
		}
	}
	return ""
}

// extractCoinFields extracts structured coin fields from analysis.
func (s *CoinLookupService) extractCoinFields(analysis string) map[string]any {
	fields := make(map[string]any)

	var parsed map[string]any
	if err := json.Unmarshal([]byte(analysis), &parsed); err != nil {
		mergeLinePatternFields(analysis, fields)
		mergeNGCLabelFields(analysis, fields)
		return fields
	}

	copyCoinFieldsFromMap(parsed, fields)
	if nested, ok := parsed["coin"].(map[string]any); ok {
		copyCoinFieldsFromMap(nested, fields)
	}
	for _, key := range []string{"labelText", "ngcDescription", "description", "notes"} {
		if value, ok := parsed[key].(string); ok {
			mergeLinePatternFields(value, fields)
			mergeNGCLabelFields(value, fields)
		}
	}
	mergeLinePatternFields(analysis, fields)
	mergeNGCLabelFields(analysis, fields)

	return fields
}

func mergeLinePatternFields(text string, fields map[string]any) {
	fieldAliases := map[string]string{
		"name":                "name",
		"title":               "name",
		"ruler":               "ruler",
		"issuer":              "ruler",
		"denomination":        "denomination",
		"type":                "denomination",
		"category":            "category",
		"material":            "material",
		"metal":               "material",
		"mint":                "mint",
		"mintmark":            "mint",
		"mint mark":           "mint",
		"era":                 "era",
		"grade":               "grade",
		"obverse":             "obverseDescription",
		"obverse description": "obverseDescription",
		"reverse":             "reverseDescription",
		"reverse description": "reverseDescription",
	}
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(strings.TrimLeft(line, "-*• "))
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(strings.Trim(parts[0], "*_` ")))
		targetKey, ok := fieldAliases[key]
		if !ok {
			continue
		}
		copyInferredStringField(fields, targetKey, parts[1])
	}
}

func mergeNGCLabelFields(text string, fields map[string]any) {
	for _, label := range candidateLabelTexts(text) {
		segments := splitLabelSegments(label)
		if len(segments) < 2 {
			continue
		}
		for _, segment := range segments {
			upper := strings.ToUpper(segment)
			switch {
			case strings.Contains(upper, "ROMAN EMPIRE"):
				copyInferredStringField(fields, "category", "Roman")
				copyInferredStringField(fields, "era", "ancient")
			case strings.Contains(upper, "GREEK"):
				copyInferredStringField(fields, "category", "Greek")
				copyInferredStringField(fields, "era", "ancient")
			case strings.Contains(upper, "BYZANTINE"):
				copyInferredStringField(fields, "category", "Byzantine")
				copyInferredStringField(fields, "era", "medieval")
			}
			if strings.Contains(upper, " MINT") || strings.HasSuffix(upper, "MINT") {
				mint := strings.TrimSpace(strings.TrimSuffix(strings.TrimSuffix(segment, "Mint"), "MINT"))
				copyInferredStringField(fields, "mint", titleVisibleMint(mint))
			}
			if ruler := parseVisibleRuler(segment); ruler != "" {
				copyInferredStringField(fields, "ruler", ruler)
			}
			material, denomination := parseMaterialDenomination(segment)
			if material != "" {
				copyInferredStringField(fields, "material", material)
			}
			if denomination != "" {
				copyInferredStringField(fields, "denomination", denomination)
			}
		}
	}
	if _, hasName := fields["name"]; !hasName {
		ruler, rulerOK := fields["ruler"].(string)
		denomination, denominationOK := fields["denomination"].(string)
		if rulerOK && denominationOK {
			copyInferredStringField(fields, "name", strings.TrimSpace(ruler+" "+denomination))
		}
	}
}

func candidateLabelTexts(text string) []string {
	candidates := []string{text}
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "/") {
			candidates = append(candidates, line)
		}
	}
	return candidates
}

func splitLabelSegments(label string) []string {
	rawSegments := strings.Split(label, "/")
	segments := make([]string, 0, len(rawSegments))
	for _, segment := range rawSegments {
		segment = cleanInferredValue(segment)
		if segment != "" {
			segments = append(segments, segment)
		}
	}
	return segments
}

func parseVisibleRuler(segment string) string {
	segment = cleanInferredValue(segment)
	if segment == "" || strings.Contains(strings.ToUpper(segment), "MINT") {
		return ""
	}
	if comma := strings.Index(segment, ","); comma > 0 {
		segment = strings.TrimSpace(segment[:comma])
	}
	if strings.ContainsAny(segment, "0123456789") {
		return ""
	}
	upper := strings.ToUpper(segment)
	if strings.Contains(upper, "EMPIRE") || strings.Contains(upper, "NGC") {
		return ""
	}
	words := strings.Fields(segment)
	if len(words) < 2 || len(words) > 4 {
		return ""
	}
	return segment
}

func parseMaterialDenomination(segment string) (string, string) {
	words := strings.Fields(cleanInferredValue(segment))
	if len(words) < 2 {
		return "", ""
	}
	materialCodes := map[string]string{
		"AV": "Gold",
		"AR": "Silver",
		"AE": "Bronze",
		"BI": "Billon",
	}
	material, ok := materialCodes[strings.ToUpper(strings.Trim(words[0], "."))]
	if !ok {
		return "", ""
	}
	denomination := strings.Join(words[1:], " ")
	return material, denomination
}

func titleVisibleMint(mint string) string {
	words := strings.Fields(strings.ToLower(mint))
	for i, word := range words {
		if word == "" {
			continue
		}
		words[i] = strings.ToUpper(word[:1]) + word[1:]
	}
	return strings.Join(words, " ")
}

func copyInferredStringField(fields map[string]any, key string, rawValue string) {
	if _, exists := fields[key]; exists {
		return
	}
	value := cleanInferredValue(rawValue)
	if value == "" || isNonSpecificExtraction(value) {
		return
	}
	fields[key] = value
}

func cleanInferredValue(value string) string {
	value = strings.TrimSpace(value)
	value = strings.Trim(value, "\"'`*_ ")
	value = strings.TrimSuffix(value, ".")
	return strings.TrimSpace(value)
}

func isNonSpecificExtraction(value string) bool {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if normalized == "" || normalized == "null" || normalized == "n/a" || normalized == "unknown" {
		return true
	}
	nonSpecific := []string{
		"not visible",
		"not shown",
		"not provided",
		"unclear",
		"illegible",
		"cannot determine",
		"can't determine",
	}
	for _, marker := range nonSpecific {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	return false
}

func copyCoinFieldsFromMap(source map[string]any, target map[string]any) {
	stringFields := map[string][]string{
		"name":               {"name"},
		"ruler":              {"ruler"},
		"mint":               {"mint"},
		"era":                {"era"},
		"denomination":       {"denomination"},
		"dateRange":          {"dateRange", "date_range", "date", "year"},
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

// buildNumistaQuery constructs a Numista search query from extracted fields.
func (s *CoinLookupService) buildNumistaQuery(coinFields map[string]any) string {
	builder := s.queryBuilder
	if builder == nil {
		builder = NewNumistaQueryBuilder()
	}
	return builder.Build(buildPhotoNumistaEvidence(&LookupExtractedData{CoinFields: coinFields})).Primary
}

func buildPhotoNumistaEvidence(data *LookupExtractedData) models.NumistaEvidence {
	if data == nil {
		return models.NumistaEvidence{}
	}
	fields := data.CoinFields
	return models.NumistaEvidence{
		Title:              boundedEvidenceField(stringField(fields, "name"), 200),
		Issuer:             boundedEvidenceField(stringField(fields, "ruler"), 200),
		Denomination:       boundedEvidenceField(stringField(fields, "denomination"), 100),
		Mint:               boundedEvidenceField(stringField(fields, "mint"), 200),
		DateText:           boundedEvidenceField(firstStringField(fields, "dateRange", "date", "year"), 100),
		Material:           boundedEvidenceField(stringField(fields, "material"), 100),
		ObverseInscription: boundedEvidenceField(stringField(fields, "obverseInscription"), 500),
		ReverseInscription: boundedEvidenceField(stringField(fields, "reverseInscription"), 500),
		ReverseType:        boundedEvidenceField(stringField(fields, "reverseDescription"), 500),
		VisibleText:        boundedEvidenceField(data.LabelText, 500),
	}
}

func buildPhotoNumistaQuery(evidence models.NumistaEvidence) string {
	return NewNumistaQueryBuilder().Build(evidence).Primary
}

func stringField(fields map[string]any, key string) string {
	if value, ok := fields[key].(string); ok {
		return value
	}
	return ""
}

func firstStringField(fields map[string]any, keys ...string) string {
	for _, key := range keys {
		if value := stringField(fields, key); strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func boundedEvidenceField(value string, maxRunes int) string {
	value = strings.TrimSpace(value)
	if value == "" || isNonSpecificExtraction(value) {
		return ""
	}
	runes := []rune(value)
	if len(runes) > maxRunes {
		value = strings.TrimSpace(string(runes[:maxRunes]))
	}
	return value
}

func nonEmptyStrings(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			result = append(result, value)
		}
	}
	return result
}

// buildPrefilledDraft creates a prefilled draft for Add to Collection/Wishlist.
func (s *CoinLookupService) buildPrefilledDraft(data *LookupExtractedData, _ []NumistaCandidate) map[string]any {
	draft := make(map[string]any)

	// Use extracted coin fields
	if data.CoinFields != nil {
		for k, v := range data.CoinFields {
			draft[k] = v
		}
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
		notesParts := []string{
			fmt.Sprintf("NGC Cert: %s", data.NGC.NormalizedCert),
		}
		if data.NGC.Grade != "" {
			notesParts = append(notesParts, fmt.Sprintf("Grade: %s", data.NGC.Grade))
		}
		if data.NGC.Description != "" {
			notesParts = append(notesParts, fmt.Sprintf("Description: %s", data.NGC.Description))
		}
		notesParts = append(notesParts, fmt.Sprintf("Lookup URL: %s", data.NGC.LookupURL))
		if existingNotes, ok := draft["notes"].(string); ok && strings.TrimSpace(existingNotes) != "" {
			notesParts = append(notesParts, strings.TrimSpace(existingNotes))
		}
		draft["notes"] = strings.Join(notesParts, "\n")
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
func (s *CoinLookupService) buildCandidateReferences(data *LookupExtractedData, _ []NumistaCandidate) []CandidateReferenceProxy {
	refs := []CandidateReferenceProxy{}

	// Add NGC reference if present
	if data.NGC != nil {
		refs = append(refs, CandidateReferenceProxy{
			Catalog: "NGC",
			Number:  data.NGC.NormalizedCert,
			URI:     data.NGC.LookupURL,
		})
	}

	return refs
}
