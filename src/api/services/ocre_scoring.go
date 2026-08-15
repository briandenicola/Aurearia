package services

import (
	"net/url"
	"sort"
	"strings"
)

// Feature 345 — deterministic OCRE candidate scoring.
//
// Score is a pure function: identical (params, rows) always yields identical
// output (SC-005). It de-duplicates by canonical TypeURI *before* ranking,
// applies fixed field weights (authority > denomination > mint > material),
// adds a bounded legend-token-in-label bonus, clamps confidence to [0,1],
// and orders results by (-Confidence, -len(MatchedFields), TypeURI asc) so
// ties are fully and stably broken. Ambiguity is preserved — multiple
// plausible types are surfaced, never collapsed to one — subject to a hard
// cap on distinct types.

const (
	ocreWeightAuthority    = 0.40
	ocreWeightDenomination = 0.25
	ocreWeightMint         = 0.20
	ocreWeightMaterial     = 0.10
	ocreWeightOCREID       = 0.90 // a confirmed known-id is a strong direct hit
	ocreLegendBonusPer     = 0.02
	ocreLegendBonusMax     = 0.10
	ocreMaxCandidates      = 8
	ocreExplanationMaxLen  = 500
)

// ocreScoreCanonicalHost returns true and the canonicalized https URI when
// uri's host is exactly numismatics.org; otherwise ("", false). This is the
// scoring-time guard mirrored by the client's post-return re-validation.
func ocreScoreCanonicalHost(uri string) (string, bool) {
	parsed, err := url.Parse(strings.TrimSpace(uri))
	if err != nil {
		return "", false
	}
	if !strings.EqualFold(parsed.Hostname(), ocreCanonicalHost) {
		return "", false
	}
	// Normalize scheme to https so the emitted citation is canonical and
	// deterministic regardless of the upstream scheme.
	parsed.Scheme = "https"
	return parsed.String(), true
}

// Score converts raw parsed SPARQL rows into deterministically ranked OCRE
// candidates for the given bound params.
func Score(params OCREQueryParams, rows []ocreParsedRow) []OCRECandidate {
	// Field attributes that were bound this query: because Template E binds
	// every present slug as a *required* triple, every returned row matched
	// all of them, so matched-field attribution is param-derived.
	type boundField struct {
		label  string // e.g. "ruler:hadrian"
		weight float64
	}
	var bound []boundField
	if params.OCREIDSlug != "" {
		bound = append(bound, boundField{"ocre_id:" + params.OCREIDSlug, ocreWeightOCREID})
	}
	if params.RulerSlug != "" {
		bound = append(bound, boundField{"ruler:" + params.RulerSlug, ocreWeightAuthority})
	}
	if params.DenominationSlug != "" {
		bound = append(bound, boundField{"denomination:" + params.DenominationSlug, ocreWeightDenomination})
	}
	if params.MintSlug != "" {
		bound = append(bound, boundField{"mint:" + params.MintSlug, ocreWeightMint})
	}
	if params.MaterialSlug != "" {
		bound = append(bound, boundField{"material:" + params.MaterialSlug, ocreWeightMaterial})
	}

	baseScore := 0.0
	matchedFields := make([]string, 0, len(bound))
	for _, f := range bound {
		baseScore += f.weight
		matchedFields = append(matchedFields, f.label)
	}

	seen := make(map[string]bool, len(rows))
	candidates := make([]OCRECandidate, 0, len(rows))
	for _, row := range rows {
		canonical, ok := ocreScoreCanonicalHost(row.TypeURI)
		if !ok {
			continue // off-host URI is dropped, never surfaced (FR-011)
		}
		if seen[canonical] {
			continue // de-dup by canonical TypeURI before ranking
		}
		seen[canonical] = true

		legendMatches := ocreLegendMatches(params.LegendTokens, row.Label)
		bonus := float64(legendMatches) * ocreLegendBonusPer
		if bonus > ocreLegendBonusMax {
			bonus = ocreLegendBonusMax
		}
		confidence := baseScore + bonus
		if confidence < 0 {
			confidence = 0
		}
		if confidence > 1 {
			confidence = 1
		}

		fields := make([]string, len(matchedFields))
		copy(fields, matchedFields)
		if legendMatches > 0 {
			fields = append(fields, "legend")
		}

		candidates = append(candidates, OCRECandidate{
			TypeURI:       canonical,
			Label:         row.Label,
			MatchedFields: fields,
			Confidence:    confidence,
			Explanation:   ocreExplanation(matchedFields, legendMatches),
		})
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].Confidence != candidates[j].Confidence {
			return candidates[i].Confidence > candidates[j].Confidence
		}
		if len(candidates[i].MatchedFields) != len(candidates[j].MatchedFields) {
			return len(candidates[i].MatchedFields) > len(candidates[j].MatchedFields)
		}
		return candidates[i].TypeURI < candidates[j].TypeURI
	})

	if len(candidates) > ocreMaxCandidates {
		candidates = candidates[:ocreMaxCandidates]
	}
	return candidates
}

// ocreLegendMatches counts how many bounded legend tokens appear (as a
// case-folded substring) in the candidate label.
func ocreLegendMatches(tokens []string, label string) int {
	if len(tokens) == 0 {
		return 0
	}
	lowerLabel := strings.ToLower(label)
	count := 0
	for _, token := range tokens {
		if token != "" && strings.Contains(lowerLabel, token) {
			count++
		}
	}
	return count
}

// ocreExplanation builds a bounded, human-readable summary of what matched.
func ocreExplanation(matchedFields []string, legendMatches int) string {
	parts := make([]string, 0, len(matchedFields)+1)
	for _, f := range matchedFields {
		kv := strings.SplitN(f, ":", 2)
		if len(kv) == 2 {
			parts = append(parts, kv[0]+" "+strings.ReplaceAll(kv[1], "_", " "))
		} else {
			parts = append(parts, f)
		}
	}
	if legendMatches > 0 {
		parts = append(parts, "legend tokens")
	}
	var explanation string
	if len(parts) == 0 {
		explanation = "No specific fields matched."
	} else {
		explanation = "Matched " + strings.Join(parts, ", ") + "."
	}
	if len(explanation) > ocreExplanationMaxLen {
		explanation = explanation[:ocreExplanationMaxLen]
	}
	return explanation
}
