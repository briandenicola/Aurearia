package services

import (
	"encoding/json"
	"reflect"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
)

func TestNumistaTelemetryBoundedConcurrentAggregate(t *testing.T) {
	telemetry := NewNumistaTelemetry(10)
	var wg sync.WaitGroup
	for i := range 20 {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			status := models.NumistaStatusSuccess
			if i == 19 {
				status = models.NumistaStatusQuotaLimited
			}
			telemetry.Record(NumistaTelemetryEvent{
				OccurredAt: time.Unix(int64(i), 0), Path: models.NumistaLookupPathDirect,
				Operation: "broad", Status: status,
				CacheOutcome: func() NumistaCacheOutcome {
					if i%2 == 0 {
						return NumistaCacheOutcomeFreshHit
					}
					return NumistaCacheOutcomeLoader
				}(),
				ElapsedMilliseconds: int64(i),
				CorrelationDigest:   strings.Repeat("secret-query", 4),
			})
		}(i)
	}
	wg.Wait()
	summary := telemetry.Health(true, true)
	if summary.BroadRequestCount+summary.FreshCacheHitCount != 10 || summary.P95ElapsedMs == 0 {
		t.Fatalf("unexpected aggregate: %+v", summary)
	}
	telemetry.mu.RLock()
	defer telemetry.mu.RUnlock()
	for _, event := range telemetry.events {
		if len(event.CorrelationDigest) > 16 || strings.Contains(event.CorrelationDigest, "secret-querysecret-query") {
			t.Fatalf("unsafe correlation digest: %q", event.CorrelationDigest)
		}
	}
}

func TestNumistaTelemetryEmptyRing(t *testing.T) {
	summary := NewNumistaTelemetry(5).Health(false, false)
	if summary.BroadRequestCount != 0 || summary.P50ElapsedMs != 0 || summary.StatusCounts == nil {
		t.Fatalf("unexpected empty summary: %+v", summary)
	}
}

func TestNumistaTelemetryQuotaAndEnrichmentAggregate(t *testing.T) {
	retry := 45
	at := time.Date(2026, 8, 11, 1, 0, 0, 0, time.UTC)
	telemetry := NewNumistaTelemetry(5)
	telemetry.Record(NumistaTelemetryEvent{
		OccurredAt: at, Path: models.NumistaLookupPathPhoto, Operation: "detail",
		Status: models.NumistaStatusQuotaLimited, DetailAttemptCount: 3,
		DetailSuccessCount: 1, DetailFailureCount: 2, RetryAfterSeconds: &retry,
		CacheOutcome: NumistaCacheOutcomeLoader,
	})
	summary := telemetry.Health(true, true)
	if summary.DetailRequestCount != 1 || summary.EnrichmentAttempted != 3 ||
		summary.EnrichmentSucceeded != 1 || summary.EnrichmentFailed != 2 ||
		summary.LastQuotaLimitedAt == nil || summary.LastRetryAfterSeconds == nil ||
		*summary.LastRetryAfterSeconds != 45 {
		t.Fatalf("unexpected quota/enrichment summary: %+v", summary)
	}
}

func TestNumistaTelemetryStatusCacheLatencyAndOperationAggregates(t *testing.T) {
	statuses := []models.NumistaLookupStatus{
		models.NumistaStatusSuccess,
		models.NumistaStatusEmpty,
		models.NumistaStatusUnconfigured,
		models.NumistaStatusQuotaLimited,
		models.NumistaStatusTimeout,
		models.NumistaStatusUnavailable,
	}
	telemetry := NewNumistaTelemetry(10)
	retry := 90
	for i, status := range statuses {
		event := NumistaTelemetryEvent{
			OccurredAt:          time.Date(2026, 8, 11, 1, i, 0, 0, time.UTC),
			Path:                models.NumistaLookupPathDirect,
			Operation:           "broad",
			Status:              status,
			ElapsedMilliseconds: int64((i + 1) * 10),
			CorrelationDigest:   NumistaCorrelationDigest(models.NumistaLookupPathDirect, "private query"),
		}
		if i == 0 {
			event.CacheOutcome = NumistaCacheOutcomeFreshHit
		} else {
			event.CacheOutcome = NumistaCacheOutcomeLoader
		}
		if status == models.NumistaStatusQuotaLimited {
			event.RetryAfterSeconds = &retry
		}
		telemetry.Record(event)
	}
	telemetry.Record(NumistaTelemetryEvent{
		OccurredAt: time.Date(2026, 8, 11, 2, 0, 0, 0, time.UTC),
		Path:       models.NumistaLookupPathPhoto, Operation: "detail", Status: models.NumistaStatusSuccess,
		ElapsedMilliseconds: 70, DetailAttemptCount: 4, DetailSuccessCount: 3, DetailFailureCount: 1,
		CacheOutcome: NumistaCacheOutcomeLoader,
	})

	summary := telemetry.Health(true, true)
	for _, status := range statuses {
		want := 1
		if summary.StatusCounts[status] != want {
			t.Fatalf("status %q count=%d, want %d: %+v", status, summary.StatusCounts[status], want, summary)
		}
	}
	if summary.StatusCounts[models.NumistaStatusSuccess] != 1 ||
		summary.BroadRequestCount != 5 || summary.DetailRequestCount != 1 ||
		summary.FreshCacheHitCount != 1 || summary.ProviderLoadCount != 6 ||
		summary.ProviderFailureCount != 4 || summary.FreshCacheHitRate != 1.0/7.0 ||
		summary.P50ElapsedMs != 45 || summary.P95ElapsedMs != 68 ||
		summary.EnrichmentAttempted != 4 || summary.EnrichmentSucceeded != 3 || summary.EnrichmentFailed != 1 ||
		summary.LastQuotaLimitedAt == nil || summary.LastRetryAfterSeconds == nil ||
		*summary.LastRetryAfterSeconds != retry {
		t.Fatalf("unexpected aggregate: %+v", summary)
	}

}

func TestNumistaTelemetryAggregationOwnershipByEventKind(t *testing.T) {
	retry := 60
	at := time.Date(2026, 8, 11, 3, 0, 0, 0, time.UTC)
	poisoned := NumistaTelemetryEvent{
		OccurredAt: at, Path: models.NumistaLookupPathPhoto, Operation: "detail",
		Status: models.NumistaStatusQuotaLimited, ElapsedMilliseconds: 90,
		DetailAttemptCount: 3, DetailSuccessCount: 2, DetailFailureCount: 1,
		RetryCount: 2, RetryAfterSeconds: &retry, Cancelled: true,
	}
	tests := []struct {
		name  string
		event NumistaTelemetryEvent
		check func(*testing.T, models.NumistaHealthSummary)
	}{
		{
			name: "cancelled loader owns only cancellation",
			event: func() NumistaTelemetryEvent {
				event := poisoned
				event.CacheOutcome = NumistaCacheOutcomeLoader
				return event
			}(),
			check: func(t *testing.T, summary models.NumistaHealthSummary) {
				assertOnlyReuseAggregate(t, summary, 0, 0, 1)
			},
		},
		{
			name: "fresh hit owns only fresh cache count",
			event: func() NumistaTelemetryEvent {
				event := poisoned
				event.CacheOutcome = NumistaCacheOutcomeFreshHit
				event.Cancelled = false
				return event
			}(),
			check: func(t *testing.T, summary models.NumistaHealthSummary) {
				assertOnlyReuseAggregate(t, summary, 1, 0, 0)
			},
		},
		{
			name: "coalesced waiter owns only coalesced count",
			event: func() NumistaTelemetryEvent {
				event := poisoned
				event.CacheOutcome = NumistaCacheOutcomeCoalescedWaiter
				event.Cancelled = false
				return event
			}(),
			check: func(t *testing.T, summary models.NumistaHealthSummary) {
				assertOnlyReuseAggregate(t, summary, 0, 1, 0)
			},
		},
		{
			name: "bypass cancellation owns only cancellation count",
			event: func() NumistaTelemetryEvent {
				event := poisoned
				event.CacheOutcome = NumistaCacheOutcomeBypass
				return event
			}(),
			check: func(t *testing.T, summary models.NumistaHealthSummary) {
				assertOnlyReuseAggregate(t, summary, 0, 0, 1)
			},
		},
		{
			name: "bypass bookkeeping owns no aggregates",
			event: func() NumistaTelemetryEvent {
				event := poisoned
				event.CacheOutcome = NumistaCacheOutcomeBypass
				event.Cancelled = false
				return event
			}(),
			check: func(t *testing.T, summary models.NumistaHealthSummary) {
				assertOnlyReuseAggregate(t, summary, 0, 0, 0)
			},
		},
		{
			name: "unconfigured bypass owns only configuration status",
			event: NumistaTelemetryEvent{
				OccurredAt: at, Path: models.NumistaLookupPathDirect, Operation: "broad",
				Status: models.NumistaStatusUnconfigured, CacheOutcome: NumistaCacheOutcomeBypass,
			},
			check: func(t *testing.T, summary models.NumistaHealthSummary) {
				if len(summary.StatusCounts) != 1 ||
					summary.StatusCounts[models.NumistaStatusUnconfigured] != 1 ||
					summary.LastOutcome != models.NumistaStatusUnconfigured ||
					summary.LastCheckedAt == nil || summary.ProviderLoadCount != 0 ||
					summary.BroadRequestCount != 0 || summary.P50ElapsedMs != 0 ||
					summary.P95ElapsedMs != 0 || summary.LastQuotaLimitedAt != nil {
					t.Fatalf("unconfigured bypass polluted provider aggregates: %+v", summary)
				}
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			telemetry := NewNumistaTelemetry(1)
			telemetry.Record(test.event)
			test.check(t, telemetry.Health(true, true))
		})
	}
}

func assertOnlyReuseAggregate(
	t *testing.T,
	summary models.NumistaHealthSummary,
	freshHits int,
	coalesced int,
	cancelled int,
) {
	t.Helper()
	if summary.FreshCacheHitCount != freshHits ||
		summary.CoalescedRequestCount != coalesced ||
		summary.CancelledRequestCount != cancelled ||
		summary.BroadRequestCount != 0 || summary.DetailRequestCount != 0 ||
		summary.ProviderLoadCount != 0 || summary.ProviderFailureCount != 0 ||
		len(summary.StatusCounts) != 0 || summary.P50ElapsedMs != 0 || summary.P95ElapsedMs != 0 ||
		summary.EnrichmentAttempted != 0 || summary.EnrichmentSucceeded != 0 ||
		summary.EnrichmentFailed != 0 || summary.LastOutcome != "" ||
		summary.LastCheckedAt != nil || summary.LastQuotaLimitedAt != nil ||
		summary.LastRetryAfterSeconds != nil {
		t.Fatalf("non-loader polluted provider aggregates: %+v", summary)
	}
}

func TestNumistaTelemetryPercentileBoundaries(t *testing.T) {
	tests := []struct {
		name   string
		values []int64
		p50    int64
		p95    int64
	}{
		{name: "empty", p50: 0, p95: 0},
		{name: "one", values: []int64{40}, p50: 40, p95: 40},
		{name: "two", values: []int64{10, 20}, p50: 15, p95: 20},
		{name: "odd", values: []int64{10, 20, 30}, p50: 20, p95: 29},
		{name: "even", values: []int64{10, 20, 30, 40}, p50: 25, p95: 39},
		{name: "larger unsorted", values: []int64{70, 10, 50, 30, 60, 20, 40}, p50: 40, p95: 67},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			values := append([]int64(nil), test.values...)
			sort.Slice(values, func(i, j int) bool { return values[i] < values[j] })
			if got := percentile(values, 0.50); got != test.p50 {
				t.Fatalf("p50=%d, want %d for %v", got, test.p50, test.values)
			}
			if got := percentile(values, 0.95); got != test.p95 {
				t.Fatalf("p95=%d, want %d for %v", got, test.p95, test.values)
			}
		})
	}
}

func TestNumistaTelemetryContractCannotStoreOrExposeSensitiveText(t *testing.T) {
	eventType := reflect.TypeOf(NumistaTelemetryEvent{})
	for i := 0; i < eventType.NumField(); i++ {
		name := strings.ToLower(eventType.Field(i).Name)
		for _, prohibited := range []string{"apikey", "query", "evidence", "inscription", "label", "rawerror", "responsebody"} {
			if strings.Contains(name, prohibited) {
				t.Fatalf("telemetry event exposes prohibited field %q", eventType.Field(i).Name)
			}
		}
	}

	const sensitive = "secret-key IMP TRAIANO private label"
	telemetry := NewNumistaTelemetry(2)
	telemetry.Record(NumistaTelemetryEvent{
		OccurredAt: time.Date(2026, 8, 11, 1, 0, 0, 0, time.UTC),
		Path:       models.NumistaLookupPathDirect, Operation: "broad", Status: models.NumistaStatusSuccess,
		CorrelationDigest: sensitive,
	})
	telemetry.mu.RLock()
	stored := telemetry.events[0]
	telemetry.mu.RUnlock()
	if stored.CorrelationDigest == sensitive || len(stored.CorrelationDigest) != 16 {
		t.Fatalf("sensitive correlation input was not replaced by a bounded digest: %q", stored.CorrelationDigest)
	}
	data, err := json.Marshal(telemetry.Health(true, true))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), sensitive) || strings.Contains(string(data), "IMP TRAIANO") {
		t.Fatalf("health summary leaked sensitive text: %s", data)
	}
}
