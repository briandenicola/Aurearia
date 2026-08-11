package services

import (
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/briandenicola/ancient-coins-api/models"
	"golang.org/x/text/unicode/norm"
)

type NumistaScorer interface {
	Score(request models.NumistaLookupRequest, candidate models.NumistaCandidate) models.NumistaRelevanceAssessment
	Rank(request models.NumistaLookupRequest, candidates []models.NumistaCandidate) []models.NumistaCandidate
}

type NumistaV1Scorer struct{}

func NewNumistaV1Scorer() *NumistaV1Scorer { return &NumistaV1Scorer{} }

type numistaDimension struct {
	field     string
	weight    float64
	request   string
	candidate string
}

func (s *NumistaV1Scorer) Score(request models.NumistaLookupRequest, candidate models.NumistaCandidate) models.NumistaRelevanceAssessment {
	titleEvidence := request.Evidence.Title
	if strings.TrimSpace(titleEvidence) == "" {
		titleEvidence = request.Query
	}
	dimensions := []numistaDimension{
		{field: "title", weight: 15, request: titleEvidence, candidate: candidate.Title},
		{field: "issuer", weight: 12, request: request.Evidence.Issuer, candidate: candidate.Issuer},
		{field: "denomination", weight: 12, request: request.Evidence.Denomination, candidate: candidate.Denomination},
		{field: "mint", weight: 10, request: request.Evidence.Mint, candidate: candidate.Mint},
	}
	lowerWeightDimensions := []numistaDimension{
		{field: "material", weight: 5, request: request.Evidence.Material, candidate: candidate.Material},
		{
			field: "inscription", weight: 3,
			request:   joinEvidence(request.Evidence.ObverseInscription, request.Evidence.ReverseInscription, request.Evidence.VisibleText),
			candidate: joinEvidence(candidate.ObverseInscription, candidate.ReverseInscription),
		},
	}

	weighted := 0.0
	applicable := 0.0
	reasons := make([]models.NumistaRelevanceReason, 0, 8)
	if request.Evidence.ExactNumistaID != nil {
		applicable += 35
		similarity := -1.0
		kind, code, label := models.NumistaReasonConflict, "exact_id_conflict", "Numista identifier differs"
		if *request.Evidence.ExactNumistaID == candidate.ID {
			similarity = 1
			kind, code, label = models.NumistaReasonMatch, "exact_id_match", "Exact Numista identifier match"
		}
		weighted += 35 * similarity
		reasons = append(reasons, models.NumistaRelevanceReason{Field: "exact_id", Kind: kind, Code: code, Label: label})
	}
	applyDimensions := func(items []numistaDimension) {
		for _, dimension := range items {
			if strings.TrimSpace(dimension.request) == "" {
				continue
			}
			applicable += dimension.weight
			similarity, reason := scoreTextDimension(dimension)
			weighted += dimension.weight * similarity
			reasons = append(reasons, reason)
		}
	}
	applyDimensions(dimensions)
	if strings.TrimSpace(request.Evidence.DateText) != "" {
		applicable += 8
		similarity, reason := scoreDate(request.Evidence.DateText, candidate.MinYear, candidate.MaxYear)
		weighted += 8 * similarity
		reasons = append(reasons, reason)
	}
	applyDimensions(lowerWeightDimensions)

	score := 50
	if applicable > 0 {
		score = int(math.Round(50 + 50*weighted/applicable))
		if score < 0 {
			score = 0
		} else if score > 100 {
			score = 100
		}
	}
	return models.NumistaRelevanceAssessment{
		ScoringVersion: models.NumistaScoringVersion,
		Score:          score,
		Band:           relevanceBand(score),
		Reasons:        reasons,
	}
}

func (s *NumistaV1Scorer) Rank(request models.NumistaLookupRequest, candidates []models.NumistaCandidate) []models.NumistaCandidate {
	ranked := cloneCandidates(candidates)
	for i := range ranked {
		ranked[i].Assessment = s.Score(request, ranked[i])
	}
	sort.SliceStable(ranked, func(i, j int) bool {
		return numistaCandidateRanksBefore(request, ranked[i], ranked[j])
	})
	return ranked
}

func numistaCandidateRanksBefore(
	request models.NumistaLookupRequest,
	left models.NumistaCandidate,
	right models.NumistaCandidate,
) bool {
	if left.Assessment.Score != right.Assessment.Score {
		return left.Assessment.Score > right.Assessment.Score
	}
	leftExact := request.Evidence.ExactNumistaID != nil && left.ID == *request.Evidence.ExactNumistaID
	rightExact := request.Evidence.ExactNumistaID != nil && right.ID == *request.Evidence.ExactNumistaID
	if leftExact != rightExact {
		return leftExact
	}
	leftComplete, rightComplete := candidateCompleteness(left), candidateCompleteness(right)
	if leftComplete != rightComplete {
		return leftComplete > rightComplete
	}
	leftTitle, rightTitle := NormalizeNumistaText(left.Title), NormalizeNumistaText(right.Title)
	if leftTitle != rightTitle {
		return leftTitle < rightTitle
	}
	if left.ID != right.ID {
		return left.ID < right.ID
	}
	return left.ProviderPosition < right.ProviderPosition
}

func NormalizeNumistaText(value string) string {
	value = strings.ToLower(norm.NFKC.String(value))
	var builder strings.Builder
	lastSpace := true
	for _, r := range value {
		if r == 'ς' {
			r = 'σ'
		}
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			builder.WriteRune(r)
			lastSpace = false
		} else if !lastSpace {
			builder.WriteByte(' ')
			lastSpace = true
		}
	}
	return strings.TrimSpace(builder.String())
}

func scoreTextDimension(dimension numistaDimension) (float64, models.NumistaRelevanceReason) {
	if strings.TrimSpace(dimension.candidate) == "" {
		return 0, models.NumistaRelevanceReason{
			Field: dimension.field, Kind: models.NumistaReasonUnavailable,
			Code: "candidate_value_missing", Label: fieldLabel(dimension.field) + " unavailable for comparison",
		}
	}
	left, right := boundedTokens(dimension.request), boundedTokens(dimension.candidate)
	if len(left) == 0 || len(right) == 0 {
		return 0, models.NumistaRelevanceReason{
			Field: dimension.field, Kind: models.NumistaReasonUnavailable,
			Code: "normalized_value_missing", Label: fieldLabel(dimension.field) + " unavailable for comparison",
		}
	}
	matches := 0
	for token := range left {
		if _, ok := right[token]; ok {
			matches++
		}
	}
	if matches == 0 {
		return -1, models.NumistaRelevanceReason{
			Field: dimension.field, Kind: models.NumistaReasonConflict,
			Code: dimension.field + "_conflict", Label: fieldLabel(dimension.field) + " does not match",
		}
	}
	similarity := 2 * float64(matches) / float64(len(left)+len(right))
	if similarity > 1 {
		similarity = 1
	}
	return similarity, models.NumistaRelevanceReason{
		Field: dimension.field, Kind: models.NumistaReasonMatch,
		Code: dimension.field + "_match", Label: fieldLabel(dimension.field) + " supports this candidate",
	}
}

func boundedTokens(value string) map[string]struct{} {
	tokens := strings.Fields(NormalizeNumistaText(value))
	if len(tokens) > 100 {
		tokens = tokens[:100]
	}
	result := make(map[string]struct{}, len(tokens))
	for _, token := range tokens {
		result[token] = struct{}{}
	}
	return result
}

var numistaDateRangePattern = regexp.MustCompile(
	`(?i)^\s*([+-]?\d{1,4})\s*(BCE|BC|CE|AD)?(?:\s*(?:-|–|—|\bto\b)\s*([+-]?\d{1,4})\s*(BCE|BC|CE|AD)?)?\s*$`,
)

func scoreDate(value string, minYear, maxYear *int) (float64, models.NumistaRelevanceReason) {
	requestMin, requestMax, ok := parseDateRange(value)
	if !ok || (minYear == nil && maxYear == nil) {
		return 0, models.NumistaRelevanceReason{
			Field: "date", Kind: models.NumistaReasonUnavailable,
			Code: "date_unavailable", Label: "Date unavailable for reliable comparison",
		}
	}
	candidateMin, candidateMax := requestMin, requestMax
	if minYear != nil {
		candidateMin = *minYear
	}
	if maxYear != nil {
		candidateMax = *maxYear
	} else if minYear != nil {
		candidateMax = *minYear
	}
	if requestMin <= candidateMax && candidateMin <= requestMax {
		return 1, models.NumistaRelevanceReason{
			Field: "date", Kind: models.NumistaReasonMatch,
			Code: "date_overlap", Label: "Date ranges overlap",
		}
	}
	return -1, models.NumistaRelevanceReason{
		Field: "date", Kind: models.NumistaReasonConflict,
		Code: "date_conflict", Label: "Date ranges do not overlap",
	}
}

func parseDateRange(value string) (int, int, bool) {
	match := numistaDateRangePattern.FindStringSubmatch(strings.TrimSpace(value))
	if match == nil {
		return 0, 0, false
	}

	firstEra, secondEra := strings.ToUpper(match[2]), strings.ToUpper(match[4])
	if match[3] != "" {
		if firstEra == "" {
			firstEra = secondEra
		}
		if secondEra == "" {
			secondEra = firstEra
		}
	}

	first, ok := parseNumistaYear(match[1], firstEra)
	if !ok {
		return 0, 0, false
	}
	if match[3] == "" {
		return first, first, true
	}
	second, ok := parseNumistaYear(match[3], secondEra)
	if !ok {
		return 0, 0, false
	}
	if first > second {
		first, second = second, first
	}
	return first, second, true
}

func parseNumistaYear(value, era string) (int, bool) {
	year, err := strconv.Atoi(value)
	if err != nil {
		return 0, false
	}
	switch era {
	case "BCE", "BC":
		if year > 0 {
			year = -year
		}
	case "CE", "AD":
		if year < 0 {
			return 0, false
		}
	case "":
	default:
		return 0, false
	}
	return year, true
}

func relevanceBand(score int) string {
	if score >= 80 {
		return "strong"
	}
	if score >= 60 {
		return "possible"
	}
	return "weak"
}

func candidateCompleteness(candidate models.NumistaCandidate) int {
	values := []string{
		candidate.Issuer, candidate.Denomination, candidate.Mint, candidate.Material,
		candidate.ObverseInscription, candidate.ReverseInscription, candidate.ObverseThumbnail, candidate.ReverseThumbnail,
	}
	count := 0
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			count++
		}
	}
	if candidate.MinYear != nil || candidate.MaxYear != nil {
		count++
	}
	return count
}

func joinEvidence(values ...string) string {
	parts := make([]string, 0, len(values))
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			parts = append(parts, trimmed)
		}
	}
	return strings.Join(parts, " ")
}

func fieldLabel(field string) string {
	switch field {
	case "exact_id":
		return "Numista identifier"
	case "issuer":
		return "Issuer"
	case "denomination":
		return "Denomination"
	case "mint":
		return "Mint"
	case "material":
		return "Material"
	case "inscription":
		return "Inscription evidence"
	default:
		return "Title"
	}
}
