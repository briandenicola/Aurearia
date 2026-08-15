package services

import (
	"encoding/json"
	"regexp"
	"sort"
	"strings"
)

// Feature 345 — OCRE automated Deep Analysis provider.
//
// ocre_query.go is the *only* place a dynamic value is ever turned into a
// fragment of a SPARQL query, and it does so under a hard rule (SC-010 /
// FR-006): the query skeleton is a compile-time constant and the sole
// dynamic content is a set of pre-validated Nomisma id slugs, each
// interpolated exclusively inside `<http://nomisma.org/id/{slug}>` /
// `<http://numismatics.org/ocre/id/{slug}>` URI brackets. No caller-supplied
// text — legends, notes, free descriptions — ever reaches the query string.
// Any value that fails slug validation is DROPPED (treated as absent),
// never interpolated, so the emitted query is byte-identical for a given
// surviving slug set regardless of how adversarial the raw inputs were.

const (
	// ocreSlugMaxLen bounds a Nomisma id slug so a pathological value can
	// never balloon the query string.
	ocreSlugMaxLen = 64
	// ocreLegendTokenMaxLen / ocreLegendTokenMax bound the scoring-only
	// legend token set (never interpolated into SPARQL).
	ocreLegendTokenMaxLen = 32
	ocreLegendTokenMax    = 12
	// ocreNomismaIDBaseURI / ocreOCREIDBaseURI are the fixed URI stems into
	// which a validated slug is interpolated.
	ocreNomismaIDBaseURI = "http://nomisma.org/id/"
	ocreOCREIDBaseURI    = "http://numismatics.org/ocre/id/"
	// ocreCanonicalHost is the only host a returned type URI may carry.
	ocreCanonicalHost = "numismatics.org"
)

// ocreSlugPattern is the general Nomisma id slug shape. ocreIDPattern is the
// stricter OCRE type-id shape (Template K subject).
var (
	ocreSlugPattern = regexp.MustCompile(`^[a-z0-9]([a-z0-9_.-]*[a-z0-9])?$`)
	ocreIDPattern   = regexp.MustCompile(`^ric\.[0-9a-z_.()]+$`)
	ocreWhitespace  = regexp.MustCompile(`\s+`)
	ocreLegendToken = regexp.MustCompile(`^[a-z0-9]+$`)
)

// OCREQueryParams is the normalized, validated, escaped input set for one
// OCRE query. Every string field is either the empty string or a value that
// has already passed slug validation — construct it only via
// NewOCREQueryParams so that invariant holds.
type OCREQueryParams struct {
	RulerSlug        string
	DenominationSlug string
	MintSlug         string
	MaterialSlug     string
	LegendTokens     []string // scoring-only, never in SPARQL
	OCREIDSlug       string
	Limit            int
}

// NewOCREQueryParams normalizes and validates raw provider inputs into a
// bound parameter set. Every value is lower-cased, whitespace-collapsed to a
// single underscore, then matched against the slug regex; anything that
// still fails is dropped (returned as ""), never interpolated. Legend tokens
// are kept only when they are pure alphanumerics and are used for scoring
// exclusively.
func NewOCREQueryParams(ruler, denomination, mint, material string, legendTokens []string, ocreID string, limit int) OCREQueryParams {
	params := OCREQueryParams{
		RulerSlug:        validateOCRESlug(ruler),
		DenominationSlug: validateOCRESlug(denomination),
		MintSlug:         validateOCRESlug(mint),
		MaterialSlug:     validateOCRESlug(material),
		OCREIDSlug:       validateOCREID(ocreID),
		LegendTokens:     normalizeOCRELegendTokens(legendTokens),
		Limit:            limit,
	}
	return params
}

// validateOCRESlug normalizes and validates a single slug. Returns "" if the
// value cannot be represented as a safe Nomisma id slug.
func validateOCRESlug(raw string) string {
	candidate := ocreWhitespace.ReplaceAllString(strings.ToLower(strings.TrimSpace(raw)), "_")
	if candidate == "" || len(candidate) > ocreSlugMaxLen {
		return ""
	}
	if !ocreSlugPattern.MatchString(candidate) {
		return ""
	}
	return candidate
}

// validateOCREID validates a known OCRE type-id slug (Template K). It must
// satisfy both the general slug shape and the stricter `ric.` id shape.
func validateOCREID(raw string) string {
	candidate := ocreWhitespace.ReplaceAllString(strings.ToLower(strings.TrimSpace(raw)), "_")
	if candidate == "" || len(candidate) > ocreSlugMaxLen {
		return ""
	}
	if !ocreIDPattern.MatchString(candidate) {
		return ""
	}
	return candidate
}

func normalizeOCRELegendTokens(raw []string) []string {
	if len(raw) == 0 {
		return nil
	}
	seen := make(map[string]bool, len(raw))
	tokens := make([]string, 0, len(raw))
	for _, token := range raw {
		folded := strings.ToLower(strings.TrimSpace(token))
		if folded == "" || len(folded) > ocreLegendTokenMaxLen || !ocreLegendToken.MatchString(folded) {
			continue
		}
		if seen[folded] {
			continue
		}
		seen[folded] = true
		tokens = append(tokens, folded)
		if len(tokens) >= ocreLegendTokenMax {
			break
		}
	}
	sort.Strings(tokens)
	return tokens
}

// HasSignal reports whether at least one type-bearing slug decoded, i.e.
// whether a SPARQL query is worth running at all. Material and legend tokens
// alone are never a sufficient signal (data-model §2 invariants).
func (p OCREQueryParams) HasSignal() bool {
	return p.RulerSlug != "" || p.DenominationSlug != "" || p.MintSlug != "" || p.OCREIDSlug != ""
}

// UsesTemplateK reports whether the known-id confirm template applies.
func (p OCREQueryParams) UsesTemplateK() bool {
	return p.OCREIDSlug != ""
}

// boundLimit clamps the requested result limit into a sane bound.
func (p OCREQueryParams) boundLimit() int {
	limit := p.Limit
	if limit <= 0 {
		limit = 5
	}
	if limit > 25 {
		limit = 25
	}
	return limit
}

// BuildQuery returns the exact SPARQL 1.1 query string for this parameter
// set. The skeleton is constant; only validated slugs appear, inside URI
// brackets. Template K (known-id confirm) is selected iff OCREIDSlug is set,
// otherwise Template E (evidence search).
func (p OCREQueryParams) BuildQuery() string {
	if p.UsesTemplateK() {
		return p.buildTemplateK()
	}
	return p.buildTemplateE()
}

// ocreQueryPrefixes is the constant PREFIX block shared by both templates.
const ocreQueryPrefixes = "PREFIX nmo: <http://nomisma.org/ontology#> " +
	"PREFIX skos: <http://www.w3.org/2004/02/skos/core#> " +
	"PREFIX rdf: <http://www.w3.org/1999/02/22-rdf-syntax-ns#> "

// buildTemplateE is the evidence-search template. Every present slug becomes
// a required triple pattern binding ?type to that Nomisma authority record.
func (p OCREQueryParams) buildTemplateE() string {
	var b strings.Builder
	b.WriteString(ocreQueryPrefixes)
	b.WriteString("SELECT ?type ?label WHERE { ")
	b.WriteString("?type rdf:type nmo:TypeSeriesItem ; skos:prefLabel ?label . ")
	if p.RulerSlug != "" {
		b.WriteString("?type nmo:hasAuthority <")
		b.WriteString(ocreNomismaIDBaseURI)
		b.WriteString(p.RulerSlug)
		b.WriteString("> . ")
	}
	if p.DenominationSlug != "" {
		b.WriteString("?type nmo:hasDenomination <")
		b.WriteString(ocreNomismaIDBaseURI)
		b.WriteString(p.DenominationSlug)
		b.WriteString("> . ")
	}
	if p.MintSlug != "" {
		b.WriteString("?type nmo:hasMint <")
		b.WriteString(ocreNomismaIDBaseURI)
		b.WriteString(p.MintSlug)
		b.WriteString("> . ")
	}
	if p.MaterialSlug != "" {
		b.WriteString("?type nmo:hasMaterial <")
		b.WriteString(ocreNomismaIDBaseURI)
		b.WriteString(p.MaterialSlug)
		b.WriteString("> . ")
	}
	b.WriteString(`FILTER(LANG(?label)="en") } LIMIT `)
	b.WriteString(itoaOCRE(p.boundLimit()))
	return b.String()
}

// buildTemplateK is the known-id confirm template: a single fixed subject
// (the OCRE type URI) whose English preferred label is requested.
func (p OCREQueryParams) buildTemplateK() string {
	var b strings.Builder
	b.WriteString(ocreQueryPrefixes)
	b.WriteString("SELECT ?type ?label WHERE { ")
	b.WriteString("BIND(<")
	b.WriteString(ocreOCREIDBaseURI)
	b.WriteString(p.OCREIDSlug)
	b.WriteString("> AS ?type) ")
	b.WriteString("?type skos:prefLabel ?label . ")
	b.WriteString(`FILTER(LANG(?label)="en") } LIMIT 1`)
	return b.String()
}

func itoaOCRE(n int) string {
	// small, allocation-light int->string for the single LIMIT value.
	if n == 0 {
		return "0"
	}
	digits := [3]byte{}
	i := len(digits)
	for n > 0 && i > 0 {
		i--
		digits[i] = byte('0' + n%10)
		n /= 10
	}
	return string(digits[i:])
}

// ocreParsedRow is one raw SPARQL binding row (type URI + label) after
// parsing, before scoring/de-dup. Kept deliberately small: matched-field
// attribution is derived from the bound params, not from the row.
type ocreParsedRow struct {
	TypeURI string
	Label   string
}

// ocreSPARQLResults is the standard SPARQL 1.1 JSON results envelope.
type ocreSPARQLResults struct {
	Results struct {
		Bindings []map[string]struct {
			Type  string `json:"type"`
			Value string `json:"value"`
		} `json:"bindings"`
	} `json:"results"`
}

// ParseOCRESPARQLResults parses a SPARQL 1.1 JSON results body into rows.
// Rows lacking a type URI or an English label are skipped. It never panics
// and returns an error only when the body is not the expected JSON shape.
func ParseOCRESPARQLResults(body []byte) ([]ocreParsedRow, error) {
	var parsed ocreSPARQLResults
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, err
	}
	rows := make([]ocreParsedRow, 0, len(parsed.Results.Bindings))
	for _, binding := range parsed.Results.Bindings {
		typeCell, ok := binding["type"]
		if !ok {
			continue
		}
		labelCell := binding["label"]
		typeURI := strings.TrimSpace(typeCell.Value)
		label := strings.TrimSpace(labelCell.Value)
		if typeURI == "" || label == "" {
			continue
		}
		rows = append(rows, ocreParsedRow{TypeURI: typeURI, Label: label})
	}
	return rows, nil
}
