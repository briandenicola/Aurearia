package services

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Feature 345 — bounded in-memory TTL cache for OCRE search responses.
//
// Mirrors nomisma_cache.go. The cache key is a SHA-256 digest of the *bound*
// parameter set (validated slugs + sorted legend tokens + limit) plus a
// flagGeneration token, so (a) raw text never appears in cache internals and
// (b) an enable/disable toggle (which changes flagGeneration) can never reuse
// a stale entry. Negative (no_match) results are cached; transient failures
// (unavailable/invalid_response/timeout/cancelled) are never cached, so an
// outage never gets "stuck" for the TTL window.

const (
	ocreCacheTTL      = 10 * time.Minute
	ocreCacheMaxEntry = 200
	ocreCacheFieldSep = "\x1f"
)

// OCRESearchStatus is the cached outcome class of an OCRE search.
type OCRESearchStatus string

const (
	OCRESearchOK          OCRESearchStatus = "ok"
	OCRESearchNoMatch     OCRESearchStatus = "no_match"
	OCRESearchUnavailable OCRESearchStatus = "unavailable"
)

type ocreCacheEntry struct {
	status     OCRESearchStatus
	candidates []OCRECandidate
	expiresAt  time.Time
}

// OCRECache is a small, bounded, in-memory TTL cache for OCRE search
// responses only. Never a substitute for persisted evidence.
type OCRECache struct {
	mu      sync.Mutex
	entries map[string]ocreCacheEntry
	now     func() time.Time
}

// NewOCRECache creates a bounded OCRE search cache using the system clock.
func NewOCRECache() *OCRECache {
	return &OCRECache{entries: make(map[string]ocreCacheEntry), now: time.Now}
}

// NewOCRECacheForTest creates an OCRECache with an injectable clock for
// deterministic TTL/expiry tests.
func NewOCRECacheForTest(now func() time.Time) *OCRECache {
	return &OCRECache{entries: make(map[string]ocreCacheEntry), now: now}
}

// OCRECacheKey returns the identity used for a cached search: a SHA-256
// digest over the bound parameters and the flagGeneration token. Any single
// differing bound parameter — or a flag toggle — yields a distinct key.
func OCRECacheKey(params OCREQueryParams, flagGeneration string) string {
	tokens := append([]string(nil), params.LegendTokens...)
	sort.Strings(tokens)
	parts := []string{
		params.RulerSlug,
		params.DenominationSlug,
		params.MintSlug,
		params.MaterialSlug,
		strings.Join(tokens, ","),
		params.OCREIDSlug,
		strconv.Itoa(params.boundLimit()),
		flagGeneration,
	}
	sum := sha256.Sum256([]byte(strings.Join(parts, ocreCacheFieldSep)))
	return hex.EncodeToString(sum[:])
}

// Get returns a cached (status, candidates) pair, if present and unexpired.
func (c *OCRECache) Get(params OCREQueryParams, flagGeneration string) (OCRESearchStatus, []OCRECandidate, bool) {
	key := OCRECacheKey(params, flagGeneration)
	c.mu.Lock()
	defer c.mu.Unlock()
	entry, ok := c.entries[key]
	if !ok {
		return "", nil, false
	}
	if !c.now().Before(entry.expiresAt) {
		delete(c.entries, key)
		return "", nil, false
	}
	return entry.status, cloneOCRECandidates(entry.candidates), true
}

// Set stores a search outcome. Callers MUST NOT call this for a transient
// failure (unavailable/invalid_response/timeout/cancelled) — only ok/no_match
// results are cacheable (FR-011). An unavailable status is defensively
// ignored here as well.
func (c *OCRECache) Set(params OCREQueryParams, flagGeneration string, status OCRESearchStatus, candidates []OCRECandidate) {
	if status == OCRESearchUnavailable {
		return
	}
	key := OCRECacheKey(params, flagGeneration)
	now := c.now()
	c.mu.Lock()
	defer c.mu.Unlock()
	c.removeExpiredLocked(now)
	if _, exists := c.entries[key]; !exists && len(c.entries) >= ocreCacheMaxEntry {
		c.evictOldestLocked()
	}
	c.entries[key] = ocreCacheEntry{
		status:     status,
		candidates: cloneOCRECandidates(candidates),
		expiresAt:  now.Add(ocreCacheTTL),
	}
}

func (c *OCRECache) removeExpiredLocked(now time.Time) {
	for key, entry := range c.entries {
		if !now.Before(entry.expiresAt) {
			delete(c.entries, key)
		}
	}
}

func (c *OCRECache) evictOldestLocked() {
	var oldestKey string
	var oldest time.Time
	for key, entry := range c.entries {
		if oldestKey == "" || entry.expiresAt.Before(oldest) {
			oldestKey, oldest = key, entry.expiresAt
		}
	}
	if oldestKey != "" {
		delete(c.entries, oldestKey)
	}
}

func cloneOCRECandidates(value []OCRECandidate) []OCRECandidate {
	if value == nil {
		return nil
	}
	cloned := make([]OCRECandidate, len(value))
	copy(cloned, value)
	for i := range cloned {
		if value[i].MatchedFields != nil {
			fields := make([]string, len(value[i].MatchedFields))
			copy(fields, value[i].MatchedFields)
			cloned[i].MatchedFields = fields
		}
	}
	return cloned
}
