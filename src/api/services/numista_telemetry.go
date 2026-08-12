package services

import (
	"crypto/sha256"
	"encoding/hex"
	"math"
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
	CacheOutcome        NumistaCacheOutcome
	ElapsedMilliseconds int64
	CandidateCount      int
	DetailAttemptCount  int
	DetailSuccessCount  int
	DetailFailureCount  int
	RetryCount          int
	RetryAfterSeconds   *int
	CorrelationDigest   string
	Cancelled           bool
	Source              models.NumistaQuerySource
	Attempt             models.NumistaSearchAttempt
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
		if event.Cancelled {
			summary.CancelledRequestCount++
			if event.CacheOutcome == NumistaCacheOutcomeCoalescedWaiter {
				summary.CoalescedRequestCount++
			}
			continue
		}
		if event.CacheOutcome != NumistaCacheOutcomeCoalescedWaiter {
			switch event.Source {
			case models.NumistaQuerySourceGenerated:
				summary.GeneratedQueryCount++
			case models.NumistaQuerySourceUserEdited:
				summary.UserEditedQueryCount++
			case models.NumistaQuerySourceManual:
				summary.ManualQueryCount++
			}
			if event.Attempt == models.NumistaSearchAttemptRelaxed {
				summary.RelaxedAttemptCount++
			}
		}
		switch event.CacheOutcome {
		case NumistaCacheOutcomeFreshHit:
			summary.FreshCacheHitCount++
		case NumistaCacheOutcomeCoalescedWaiter:
			summary.CoalescedRequestCount++
		case NumistaCacheOutcomeLoader:
			summary.ProviderLoadCount++
			if event.Operation == "broad" {
				summary.BroadRequestCount++
			} else if event.Operation == "detail" {
				summary.DetailRequestCount++
			}
			if event.Status != "" {
				summary.StatusCounts[event.Status]++
			}
			if event.Status != "" &&
				event.Status != models.NumistaStatusSuccess &&
				event.Status != models.NumistaStatusEmpty {
				summary.ProviderFailureCount++
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
		case NumistaCacheOutcomeBypass:
			if event.Status == models.NumistaStatusUnconfigured {
				summary.StatusCounts[event.Status]++
			}
		}
	}
	for index := len(events) - 1; index >= 0; index-- {
		event := events[index]
		ownsOutcome := !event.Cancelled && (event.CacheOutcome == NumistaCacheOutcomeLoader ||
			(event.CacheOutcome == NumistaCacheOutcomeBypass &&
				event.Status == models.NumistaStatusUnconfigured))
		if !ownsOutcome || event.Status == "" {
			continue
		}
		lastAt := event.OccurredAt.UTC()
		summary.LastOutcome = event.Status
		summary.LastCheckedAt = &lastAt
		break
	}
	cacheDecisions := summary.FreshCacheHitCount + summary.ProviderLoadCount
	if cacheDecisions > 0 {
		summary.FreshCacheHitRate = float64(summary.FreshCacheHitCount) / float64(cacheDecisions)
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
	if len(sorted) == 1 || percentile <= 0 {
		return sorted[0]
	}
	if percentile >= 1 {
		return sorted[len(sorted)-1]
	}
	position := float64(len(sorted)-1) * percentile
	lower := int(math.Floor(position))
	upper := int(math.Ceil(position))
	if lower == upper {
		return sorted[lower]
	}
	value := float64(sorted[lower]) +
		(float64(sorted[upper]-sorted[lower]) * (position - float64(lower)))
	return int64(math.Round(value))
}
