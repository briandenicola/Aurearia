package services

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"sync"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
)

type NumistaTelemetryEvent struct {
	OccurredAt          time.Time
	Path                models.NumistaLookupPath
	Operation           string
	Status              models.NumistaLookupStatus
	CacheHit            bool
	Refreshed           bool
	ElapsedMilliseconds int64
	CandidateCount      int
	DetailAttemptCount  int
	DetailSuccessCount  int
	DetailFailureCount  int
	RetryCount          int
	RetryAfterSeconds   *int
	CorrelationDigest   string
}

type NumistaTelemetry struct {
	mu       sync.RWMutex
	capacity int
	events   []NumistaTelemetryEvent
}

func NewNumistaTelemetry(capacity int) *NumistaTelemetry {
	if capacity <= 0 {
		capacity = 500
	}
	return &NumistaTelemetry{capacity: capacity, events: make([]NumistaTelemetryEvent, 0, capacity)}
}

func (t *NumistaTelemetry) Record(event NumistaTelemetryEvent) {
	if event.OccurredAt.IsZero() {
		event.OccurredAt = time.Now().UTC()
	}
	event.CorrelationDigest = boundedDigest(event.CorrelationDigest)
	t.mu.Lock()
	if len(t.events) == t.capacity {
		copy(t.events, t.events[1:])
		t.events[len(t.events)-1] = event
	} else {
		t.events = append(t.events, event)
	}
	t.mu.Unlock()
}

func (t *NumistaTelemetry) Health(configured, configurationValid bool) models.NumistaHealthSummary {
	t.mu.RLock()
	events := append([]NumistaTelemetryEvent(nil), t.events...)
	t.mu.RUnlock()

	summary := models.NumistaHealthSummary{
		Configured: configured, ConfigurationValid: configurationValid,
		StatusCounts: make(map[models.NumistaLookupStatus]int),
	}
	if len(events) == 0 {
		return summary
	}
	durations := make([]int64, 0, len(events))
	for _, event := range events {
		summary.StatusCounts[event.Status]++
		if event.Operation == "broad" {
			summary.BroadRequestCount++
		} else if event.Operation == "detail" {
			summary.DetailRequestCount++
		}
		if event.CacheHit {
			summary.CacheHitCount++
		}
		if event.Refreshed {
			summary.CacheRefreshCount++
		}
		summary.EnrichmentAttempted += event.DetailAttemptCount
		summary.EnrichmentSucceeded += event.DetailSuccessCount
		summary.EnrichmentFailed += event.DetailFailureCount
		durations = append(durations, event.ElapsedMilliseconds)
		if event.Status == models.NumistaStatusQuotaLimited {
			at := event.OccurredAt.UTC()
			summary.LastQuotaLimitedAt = &at
			summary.LastRetryAfterSeconds = event.RetryAfterSeconds
		}
	}
	last := events[len(events)-1]
	lastAt := last.OccurredAt.UTC()
	summary.LastOutcome = last.Status
	summary.LastCheckedAt = &lastAt
	totalCache := summary.CacheHitCount + summary.CacheRefreshCount
	if totalCache > 0 {
		summary.CacheHitRate = float64(summary.CacheHitCount) / float64(totalCache)
	}
	sort.Slice(durations, func(i, j int) bool { return durations[i] < durations[j] })
	summary.P50ElapsedMs = percentile(durations, 0.50)
	summary.P95ElapsedMs = percentile(durations, 0.95)
	return summary
}

func NumistaCorrelationDigest(path models.NumistaLookupPath, query string) string {
	sum := sha256.Sum256([]byte(string(path) + "|" + NormalizeNumistaText(query)))
	return hex.EncodeToString(sum[:8])
}

func boundedDigest(value string) string {
	if len(value) == 16 {
		if _, err := hex.DecodeString(value); err == nil {
			return value
		}
	}
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:8])
}

func percentile(sorted []int64, percentile float64) int64 {
	if len(sorted) == 0 {
		return 0
	}
	index := int(float64(len(sorted)-1)*percentile + 0.5)
	if index >= len(sorted) {
		index = len(sorted) - 1
	}
	return sorted[index]
}
