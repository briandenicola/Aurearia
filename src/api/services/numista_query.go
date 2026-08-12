package services

import (
	"strings"
	"unicode"

	"github.com/briandenicola/ancient-coins-api/models"
	"golang.org/x/text/cases"
	"golang.org/x/text/unicode/norm"
)

type NumistaQueryPlan struct {
	Version string
	Primary string
	Relaxed string
}

type NumistaQueryBuilder struct {
	mintAliases map[string]string
}

func NewNumistaQueryBuilder() *NumistaQueryBuilder {
	return &NumistaQueryBuilder{mintAliases: map[string]string{
		"smn":  "Nicomedia",
		"smnt": "Nicomedia",
	}}
}

func (b *NumistaQueryBuilder) AliasCount() int {
	return len(b.mintAliases)
}

func (b *NumistaQueryBuilder) Build(evidence models.NumistaEvidence) NumistaQueryPlan {
	subject := conciseSubject(evidence)
	reverse := dedupeQueryTokens(strings.Join(nonEmptyStrings([]string{
		evidence.ReverseInscription, evidence.ReverseType,
	}), " "))
	mint := b.normalizedMint(evidence.Mint)

	return NumistaQueryPlan{
		Version: models.NumistaQueryGenerationVersion,
		Primary: boundedQuery(dedupeQueryTokens(strings.Join(nonEmptyStrings([]string{
			subject, reverse, mint,
		}), " "))),
		Relaxed: boundedQuery(dedupeQueryTokens(strings.Join(nonEmptyStrings([]string{
			subject, mint,
		}), " "))),
	}
}

func conciseSubject(evidence models.NumistaEvidence) string {
	if issuer := strings.TrimSpace(evidence.Issuer); issuer != "" {
		return issuer
	}
	title := strings.TrimSpace(evidence.Title)
	if title == "" {
		return ""
	}
	words := strings.Fields(title)
	if len(words) > 8 {
		words = words[:8]
	}
	return strings.Join(words, " ")
}

func (b *NumistaQueryBuilder) normalizedMint(value string) string {
	value = strings.TrimSpace(norm.NFKC.String(value))
	if value == "" {
		return ""
	}
	if alias, ok := b.mintAliases[normalizeMintAlias(value)]; ok {
		return alias
	}
	hasLetter := false
	hasLower := false
	for _, token := range strings.Fields(value) {
		letters := 0
		uppercase := 0
		for _, r := range token {
			if unicode.IsLetter(r) {
				letters++
				if unicode.IsUpper(r) {
					uppercase++
				}
			}
		}
		if letters > 1 && uppercase == letters {
			return ""
		}
	}
	for _, r := range value {
		switch {
		case unicode.IsLetter(r):
			hasLetter = true
			hasLower = hasLower || unicode.IsLower(r)
		case unicode.IsSpace(r), r == '\'', r == '’':
		default:
			return ""
		}
	}
	if !hasLetter || !hasLower {
		return ""
	}
	return strings.Join(strings.Fields(value), " ")
}

func normalizeMintAlias(value string) string {
	value = cases.Fold().String(norm.NFKC.String(value))
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) || r == '.' || r == '-' || r == '‐' || r == '‑' || r == '–' || r == '—' {
			return -1
		}
		return r
	}, value)
}

func dedupeQueryTokens(value string) string {
	seen := make(map[string]struct{})
	result := make([]string, 0)
	for _, token := range strings.Fields(norm.NFKC.String(value)) {
		key := cases.Fold().String(strings.Trim(token, ".,;:"))
		if key == "" {
			continue
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, token)
	}
	return strings.Join(result, " ")
}

func boundedQuery(value string) string {
	value = strings.TrimSpace(value)
	runes := []rune(value)
	if len(runes) <= models.NumistaMaxQueryLength {
		return value
	}
	return strings.TrimSpace(string(runes[:models.NumistaMaxQueryLength]))
}
