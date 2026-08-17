package services

import (
	"strings"

	"github.com/briandenicola/ancient-coins-api/models"
)

// CatalogParseReason classifies why ParseCatalogReferenceText did or did not
// return a usable reference. It follows the same exported int-enum +
// name-map convention as services.LogLevel (see logger.go).
type CatalogParseReason int

const (
	// CatalogParseOK means a catalog was resolved and registered. The
	// reference may still have NeedsVolume set — that is not a failure,
	// just a lower-confidence result.
	CatalogParseOK CatalogParseReason = iota
	// CatalogParseEmpty means the input was empty/whitespace-only (or,
	// practically unreachable, had no tokens after trimming). Catalog and
	// RawText are both empty.
	CatalogParseEmpty
	// CatalogParseUnrecognizedCatalog means the first token did not match
	// any known catalog alias. Catalog is empty; the raw offending token is
	// the first field of RawText.
	CatalogParseUnrecognizedCatalog
	// CatalogParseNotInRegistry means the first token resolved to a known
	// canonical catalog code, but that code has no entry in the registry
	// map passed by the caller. Catalog holds the resolved code.
	CatalogParseNotInRegistry
	// CatalogParseNoNumber means the catalog resolved and is registered,
	// does NOT require a volume, but no number token remained to parse.
	// Catalog holds the resolved code.
	CatalogParseNoNumber
)

var catalogParseReasonNames = map[CatalogParseReason]string{
	CatalogParseOK:                  "ok",
	CatalogParseEmpty:               "empty",
	CatalogParseUnrecognizedCatalog: "unrecognized_catalog",
	CatalogParseNotInRegistry:       "not_in_registry",
	CatalogParseNoNumber:            "no_number",
}

// String implements fmt.Stringer for readable test failure output and logs.
func (r CatalogParseReason) String() string {
	if name, ok := catalogParseReasonNames[r]; ok {
		return name
	}
	return "unknown"
}

// ParsedCatalogReference is the structured result of parsing free-form catalog
// reference text (e.g. legacy rarity_rating strings, or deep-identification
// proposals) into a Catalog / Volume / Number tuple with a confidence score.
//
// This type intentionally carries no sentinel values and no policy decisions:
// callers (ReferenceMigrationService, Feature 352 deep-identification, etc.)
// decide for themselves what to do when NeedsVolume is true. See FR-016/FR-019
// in specs/352-deep-identification-structured-results/spec.md.
type ParsedCatalogReference struct {
	// Catalog is the alias-normalized, canonical catalog code (e.g. "RIC")
	// once the first token has been successfully resolved via
	// normalizeCatalogAlias — regardless of whether the catalog is in the
	// caller's registry, and regardless of whether a volume/number was
	// found. It is empty ONLY when Reason is CatalogParseEmpty or
	// CatalogParseUnrecognizedCatalog (no canonical code could be
	// determined at all). Catalog never means anything else on any path.
	Catalog string
	// Volume is the parsed volume token (Roman numeral, short numeric, or
	// alphabetic like "Cop"). Empty when no volume was found/required.
	Volume string
	// Number is the remaining catalog number text (may include qualifiers
	// like "cf." or letter suffixes like "256a").
	Number string
	// Confidence is a score in [0, 1] per FR-017's confidence table:
	// 0.90 for a clean parse (volume not required, or a Roman-numeral
	// volume), 0.50 for an inferred (non-Roman but plausible) volume token,
	// and 0.30 when NeedsVolume is true.
	Confidence float64
	// NeedsVolume is true when the resolved catalog requires a volume but
	// none could be parsed from the text. The helper does NOT substitute a
	// placeholder value here — that is migration-specific policy.
	NeedsVolume bool
	// RawText is the origin text segment the parse was based on: the text
	// before the first ";" when present, otherwise the full trimmed input.
	// Empty only when Reason is CatalogParseEmpty.
	RawText string
	// Reason classifies the outcome. CatalogParseOK when a usable catalog
	// was resolved (even if NeedsVolume is true); one of the other values
	// otherwise, explaining precisely what could not be determined so a
	// caller can build its own messaging without re-deriving parse state.
	Reason CatalogParseReason
}

// ParseCatalogReferenceText parses the first catalog reference from free-form
// text (e.g. "RIC II 207", "Sear 1625", "RIC 207; Cohen 15") against a known
// catalog registry, and returns the parsed structure along with whether a
// usable catalog reference was found.
//
// This is the shared parsing core extracted from
// ReferenceMigrationService.parseLegacyReference (Feature 352 Phase 1). It is
// behaviour-equivalent to that method's parsing logic, minus the
// migration-specific Volume:"0" sentinel and journal-message policy, which
// remain in ReferenceMigrationService.
func ParseCatalogReferenceText(text string, registry map[string]*models.CatalogRegistry) (ParsedCatalogReference, bool) {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return ParsedCatalogReference{Reason: CatalogParseEmpty}, false
	}

	parts := strings.SplitN(trimmed, ";", 2)
	first := strings.TrimSpace(parts[0])
	if first == "" {
		return ParsedCatalogReference{Reason: CatalogParseEmpty}, false
	}

	tokens := strings.Fields(first)
	if len(tokens) == 0 {
		return ParsedCatalogReference{Reason: CatalogParseEmpty}, false
	}

	origText := trimmed
	if len(parts) > 1 {
		origText = first
	}

	catalogToken := strings.ToUpper(tokens[0])
	catalogNormalized := normalizeCatalogAlias(catalogToken)
	if catalogNormalized == "" {
		return ParsedCatalogReference{RawText: origText, Reason: CatalogParseUnrecognizedCatalog}, false
	}

	regEntry, ok := registry[catalogNormalized]
	if !ok {
		return ParsedCatalogReference{Catalog: catalogNormalized, RawText: origText, Reason: CatalogParseNotInRegistry}, false
	}

	remaining := tokens[1:]

	if regEntry.VolumeRequired {
		if len(remaining) == 0 {
			return ParsedCatalogReference{
				Catalog:     catalogNormalized,
				NeedsVolume: true,
				Confidence:  0.30,
				RawText:     origText,
				Reason:      CatalogParseOK,
			}, true
		}

		volCandidate := remaining[0]
		if isRomanNumeral(volCandidate) || isPlausibleVolumeToken(volCandidate) {
			rest := remaining[1:]

			if len(rest) == 0 {
				return ParsedCatalogReference{
					Catalog:     catalogNormalized,
					NeedsVolume: true,
					Confidence:  0.30,
					RawText:     origText,
					Reason:      CatalogParseOK,
				}, true
			}

			confidence := 0.50
			if isRomanNumeral(volCandidate) {
				confidence = 0.90
			}

			return ParsedCatalogReference{
				Catalog:    catalogNormalized,
				Volume:     volCandidate,
				Number:     strings.Join(rest, " "),
				Confidence: confidence,
				RawText:    origText,
				Reason:     CatalogParseOK,
			}, true
		}

		return ParsedCatalogReference{
			Catalog:     catalogNormalized,
			NeedsVolume: true,
			Confidence:  0.30,
			RawText:     origText,
			Reason:      CatalogParseOK,
		}, true
	}

	if len(remaining) == 0 {
		return ParsedCatalogReference{Catalog: catalogNormalized, RawText: origText, Reason: CatalogParseNoNumber}, false
	}

	return ParsedCatalogReference{
		Catalog:    catalogNormalized,
		Number:     strings.Join(remaining, " "),
		Confidence: 0.90,
		RawText:    origText,
		Reason:     CatalogParseOK,
	}, true
}

// normalizeCatalogAlias maps known aliases to canonical catalog codes.
// Moved verbatim from ReferenceMigrationService (Feature 352 Phase 1).
//
// Do NOT add "NGC" here (FR-007, F-6): it would newly migrate legacy
// rarity_rating text starting with "NGC" that today is left untouched.
func normalizeCatalogAlias(token string) string {
	upper := strings.ToUpper(token)
	switch upper {
	case "RIC", "RPC", "SNG", "CRAWFORD", "CNI", "KM", "Y", "CRAIG", "REDBOOK":
		return upper
	case "SEAR", "SRCV":
		return "SEAR"
	case "SPINK":
		return "SPINK"
	case "DUPLESSY":
		return "DUPLESSY"
	default:
		return ""
	}
}

// isRomanNumeral checks if a token is a valid Roman numeral.
// Moved verbatim from ReferenceMigrationService (Feature 352 Phase 1).
func isRomanNumeral(str string) bool {
	str = strings.ToUpper(str)
	for _, ch := range str {
		if ch != 'I' && ch != 'V' && ch != 'X' && ch != 'L' && ch != 'C' && ch != 'D' && ch != 'M' {
			return false
		}
	}
	return len(str) > 0
}

// isPlausibleVolumeToken checks if a token looks like it could be a volume (not purely numeric like a catalog number).
// Accepts Roman numerals, short numeric strings (1-3 digits), or alphabetic tokens (e.g. "Cop" for SNG Copenhagen).
// Moved verbatim from ReferenceMigrationService (Feature 352 Phase 1).
func isPlausibleVolumeToken(str string) bool {
	if len(str) == 0 {
		return false
	}

	if isRomanNumeral(str) {
		return true
	}

	if len(str) <= 3 {
		allDigits := true
		for _, ch := range str {
			if ch < '0' || ch > '9' {
				allDigits = false
				break
			}
		}
		if allDigits {
			return true
		}
	}

	allLetters := true
	for _, ch := range str {
		if (ch < 'A' || ch > 'Z') && (ch < 'a' || ch > 'z') {
			allLetters = false
			break
		}
	}
	return allLetters
}
