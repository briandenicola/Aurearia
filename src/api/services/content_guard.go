package services

import (
	"errors"
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// Content moderation and prompt injection detection for agent chat.

var (
	ErrBlockedContent    = errors.New("message contains inappropriate content")
	ErrPromptInjection   = errors.New("message contains disallowed instructions")
	ErrMessageTooLong    = errors.New("message exceeds maximum length")
	ErrHistoryTooLong    = errors.New("conversation history exceeds maximum length")
	ErrInvalidRole       = errors.New("invalid message role in history")
)

const (
	MaxMessageLength = 4000
	MaxHistoryLength = 50
	MaxHistoryChars  = 100000
)

// blockedPatterns are regex patterns for clearly inappropriate content.
// Uses word boundaries to reduce false positives (e.g., "drug store" won't match).
var blockedPatterns = []*regexp.Regexp{
	regexp.MustCompile(`\b(porn(ography|ographic)?|xxx|hentai|nsfw)\b`),
	regexp.MustCompile(`(?i)\bhow\s+to\s+(make|build|create)\s+(an?\s+)?(explosive\s+device|bomb|explosive|weapon|poison)\b`),
	regexp.MustCompile(`\b(how\s+to\s+(hack|crack|break\s+into))\b`),
	regexp.MustCompile(`\b(child\s+(abuse|exploitation|pornography))\b`),
	regexp.MustCompile(`\b(kill\s+(myself|yourself|someone|people))\b`),
	regexp.MustCompile(`\b(how\s+to\s+commit\s+suicide)\b`),
	regexp.MustCompile(`\b(racial\s+slur|white\s+supremac|ethnic\s+cleansing)\b`),
}

// injectionPatterns detect common prompt injection attempts.
// Kept high-precision to avoid false positives.
var injectionPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)ignore\s+(all\s+)?(previous|prior|above|earlier)\s+(instructions|prompts|rules|directions)`),
	regexp.MustCompile(`(?i)forget\s+(all\s+)?(your|previous|prior)\s+(instructions|rules|prompts|training)`),
	regexp.MustCompile(`(?i)you\s+are\s+now\s+(a|an|my)\s+`),
	regexp.MustCompile(`(?i)new\s+(system\s+)?persona`),
	regexp.MustCompile(`(?i)^\s*system\s*:\s*`),
	regexp.MustCompile(`(?i)^\s*\[system\]\s*`),
	regexp.MustCompile(`(?i)developer\s+mode\s+(enabled|activated|on)`),
	regexp.MustCompile(`(?i)do\s+anything\s+now`),
	regexp.MustCompile(`(?i)(disregard|override)\s+(all\s+)?(safety|content)\s+(filters?|policies|guidelines|rules)`),
	regexp.MustCompile(`(?i)jailbreak`),
	regexp.MustCompile(`(?i)pretend\s+(you('re|\s+are)\s+)?(not\s+)?(an?\s+)?(ai|assistant|chatbot|constrained)`),
}

var validRoles = map[string]bool{
	"user":      true,
	"assistant": true,
}

// ContentGuard validates and moderates agent chat messages.
type ContentGuard struct {
	logger *Logger
}

func NewContentGuard(logger *Logger) *ContentGuard {
	return &ContentGuard{logger: logger}
}

// normalize lowercases and applies Unicode NFKC normalization to collapse
// tricky character variants (e.g., fullwidth, accented lookalikes).
func normalize(s string) string {
	s = norm.NFKC.String(s)
	s = strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsSpace(r) {
			return unicode.ToLower(r)
		}
		return r
	}, s)
	return s
}

// ValidateMessage checks the current user message for blocked content,
// prompt injection, and length limits.
func (g *ContentGuard) ValidateMessage(message string, userID uint) error {
	if len(message) > MaxMessageLength {
		return ErrMessageTooLong
	}

	normalized := normalize(message)

	for _, p := range blockedPatterns {
		if p.MatchString(normalized) {
			g.logger.Warn("content_guard",
				"Blocked content detected from user %d (pattern: %s)",
				userID, p.String())
			return ErrBlockedContent
		}
	}

	for _, p := range injectionPatterns {
		if p.MatchString(normalized) {
			g.logger.Warn("content_guard",
				"Prompt injection attempt from user %d (pattern: %s)",
				userID, p.String())
			return ErrPromptInjection
		}
	}

	return nil
}

// ValidateHistory checks chat history for structural validity and
// scans user messages for blocked content.
func (g *ContentGuard) ValidateHistory(history []ChatMessageProxy, userID uint) error {
	if len(history) > MaxHistoryLength {
		return ErrHistoryTooLong
	}

	totalChars := 0
	for _, msg := range history {
		if !validRoles[msg.Role] {
			return ErrInvalidRole
		}
		totalChars += len(msg.Content)
		if totalChars > MaxHistoryChars {
			return ErrHistoryTooLong
		}

		// Only scan user messages in history for blocked content
		if msg.Role == "user" {
			normalized := normalize(msg.Content)
			for _, p := range blockedPatterns {
				if p.MatchString(normalized) {
					g.logger.Warn("content_guard",
						"Blocked content in history from user %d", userID)
					return ErrBlockedContent
				}
			}
		}
	}

	return nil
}
