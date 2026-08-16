package services

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"sync"
	"time"
)

const (
	nomismaCacheTTL      = 10 * time.Minute
	nomismaCacheMaxEntry = 200
)

// NomismaSearchStatus is the outcome of a Nomisma search, mirroring
// data-model.md's NomismaSearchOutcome. It is the shape cached and
// returned to callers - "ok" (candidates present), "no_match" (zero
// candidates, not an error), or "unavailable" (never cached).
type NomismaSearchStatus string

const (
	NomismaSearchOK          NomismaSearchStatus = "ok"
	NomismaSearchNoMatch     NomismaSearchStatus = "no_match"
	NomismaSearchUnavailable NomismaSearchStatus = "unavailable"
)

type nomismaCacheEntry struct {
	status     NomismaSearchStatus
	candidates []NomismaCandidate
	expiresAt  time.Time
}

// NomismaCache is a small, bounded, in-memory TTL cache for Nomisma search
// responses only. Never a substitute for the persisted confirmed link.
// Provider failures (unavailable/invalid_response) are never cached, so a
// transient outage never gets "stuck" for the TTL window. A zero-result
// search IS cached (as a negative entry) for the same short TTL.
type NomismaCache struct {
	mu      sync.Mutex
	entries map[string]nomismaCacheEntry
	now     func() time.Time
}

// NewNomismaCache creates a bounded Nomisma search cache using the system
// clock.
func NewNomismaCache() *NomismaCache {
	return &NomismaCache{entries: make(map[string]nomismaCacheEntry), now: time.Now}
}

// NewNomismaCacheForTest creates a NomismaCache with an injectable clock,
// for deterministic TTL/expiry tests.
func NewNomismaCacheForTest(now func() time.Time) *NomismaCache {
	return &NomismaCache{entries: make(map[string]nomismaCacheEntry), now: now}
}

// NomismaCacheKey returns the identity used for a cached search: a SHA-256
// digest of the normalized (trimmed, lower-cased, whitespace-collapsed)
// query text, so raw query text never appears in cache internals/logs.
func NomismaCacheKey(query string) string {
	sum := sha256.Sum256([]byte(normalizeNomismaQuery(query)))
	return hex.EncodeToString(sum[:])
}

func normalizeNomismaQuery(query string) string {
	fields := strings.Fields(strings.ToLower(strings.TrimSpace(query)))
	return strings.Join(fields, " ")
}

// Get returns a cached (status, candidates) pair for query, if present and
// unexpired.
func (c *NomismaCache) Get(query string) (NomismaSearchStatus, []NomismaCandidate, bool) {
	key := NomismaCacheKey(query)
	c.mu.Lock()
	defer c.mu.Unlock()
	entry, ok := c.entries[key]
	if !ok {
		return "", nil, false
	}
	now := c.now()
	if !now.Before(entry.expiresAt) {
		delete(c.entries, key)
		return "", nil, false
	}
	return entry.status, cloneNomismaCandidates(entry.candidates), true
}

// Set stores a search outcome for query. Callers MUST NOT call this for an
// "unavailable"/invalid_response outcome - only "ok"/"no_match" results are
// ever cacheable (FR-011).
func (c *NomismaCache) Set(query string, status NomismaSearchStatus, candidates []NomismaCandidate) {
	if status == NomismaSearchUnavailable {
		return
	}
	key := NomismaCacheKey(query)
	now := c.now()
	c.mu.Lock()
	defer c.mu.Unlock()
	c.removeExpiredLocked(now)
	if _, exists := c.entries[key]; !exists && len(c.entries) >= nomismaCacheMaxEntry {
		c.evictOldestLocked()
	}
	c.entries[key] = nomismaCacheEntry{
		status:     status,
		candidates: cloneNomismaCandidates(candidates),
		expiresAt:  now.Add(nomismaCacheTTL),
	}
}

func (c *NomismaCache) removeExpiredLocked(now time.Time) {
	for key, entry := range c.entries {
		if !now.Before(entry.expiresAt) {
			delete(c.entries, key)
		}
	}
}

// evictOldestLocked evicts the entry with the soonest expiry (LRU-by-expiry,
// mirroring numista_cache.go's eviction shape).
func (c *NomismaCache) evictOldestLocked() {
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

func cloneNomismaCandidates(value []NomismaCandidate) []NomismaCandidate {
	if value == nil {
		return nil
	}
	cloned := make([]NomismaCandidate, len(value))
	copy(cloned, value)
	return cloned
}
