package services

import (
	"strings"
	"unicode"
)

// NormalizeMatchValue canonicalizes a value for case/punctuation-insensitive
// matching, mirroring models.NormalizeMintLocationName's approach for mint
// names. Used to match an AI-suggested or free-text category/era value
// against an admin-defined list without requiring an exact string match.
func NormalizeMatchValue(value string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(value)) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// MatchKnownValue finds the known value that best matches candidate: an
// exact normalized match wins outright; otherwise a known value is accepted
// if one normalized form fully contains the other (handles near-misses like
// an AI suggesting "Roman Republic" against a known "Roman"). Returns the
// matched known value in its original casing and true, or ("", false) if
// nothing matches confidently - callers should treat "no match" as a signal
// to ask for confirmation rather than guessing further.
func MatchKnownValue(candidate string, known []string) (string, bool) {
	normalizedCandidate := NormalizeMatchValue(candidate)
	if normalizedCandidate == "" {
		return "", false
	}
	for _, k := range known {
		if NormalizeMatchValue(k) == normalizedCandidate {
			return k, true
		}
	}
	for _, k := range known {
		nk := NormalizeMatchValue(k)
		if nk == "" {
			continue
		}
		if strings.Contains(normalizedCandidate, nk) || strings.Contains(nk, normalizedCandidate) {
			return k, true
		}
	}
	return "", false
}
