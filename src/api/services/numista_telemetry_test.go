package services

import (
	"encoding/json"
	"reflect"
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
				Operation: "broad", Status: status, CacheHit: i%2 == 0,
				Refreshed: i%2 != 0, ElapsedMilliseconds: int64(i),
				CorrelationDigest: strings.Repeat("secret-query", 4),
			})
		}(i)
	}
	wg.Wait()
	summary := telemetry.Health(true, true)
	if summary.BroadRequestCount != 10 || summary.P95ElapsedMs == 0 {
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
			CacheHit:            i == 0,
			Refreshed:           i == 1,
			CorrelationDigest:   NumistaCorrelationDigest(models.NumistaLookupPathDirect, "private query"),
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
	})

	summary := telemetry.Health(true, true)
	for _, status := range statuses {
		want := 1
		if status == models.NumistaStatusSuccess {
			want = 2
		}
		if summary.StatusCounts[status] != want {
			t.Fatalf("status %q count=%d, want %d: %+v", status, summary.StatusCounts[status], want, summary)
		}
	}
	if summary.StatusCounts[models.NumistaStatusSuccess] != 2 ||
		summary.BroadRequestCount != 6 || summary.DetailRequestCount != 1 ||
		summary.CacheHitCount != 1 || summary.CacheRefreshCount != 1 || summary.CacheHitRate != 0.5 ||
		summary.P50ElapsedMs != 40 || summary.P95ElapsedMs != 70 ||
		summary.EnrichmentAttempted != 4 || summary.EnrichmentSucceeded != 3 || summary.EnrichmentFailed != 1 ||
		summary.LastQuotaLimitedAt == nil || summary.LastRetryAfterSeconds == nil ||
		*summary.LastRetryAfterSeconds != retry {
		t.Fatalf("unexpected aggregate: %+v", summary)
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
