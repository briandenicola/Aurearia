package services

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
)

type phase6TelemetryResponse struct {
	candidates []models.NumistaCandidate
	err        error
	elapsed    time.Duration
}

type phase6TelemetryClient struct {
	mu        sync.Mutex
	clock     *fakeNumistaClock
	responses map[string]phase6TelemetryResponse
}

func (c *phase6TelemetryClient) Search(_ context.Context, query string, _ int) ([]models.NumistaCandidate, error) {
	c.mu.Lock()
	response := c.responses[query]
	c.mu.Unlock()
	c.clock.Add(response.elapsed)
	return cloneCandidates(response.candidates), response.err
}

func (c *phase6TelemetryClient) Detail(context.Context, int) (models.NumistaCandidate, error) {
	return models.NumistaCandidate{}, nil
}

type phase6TakeoverClient struct {
	calls          atomic.Int32
	firstStarted   chan struct{}
	firstCancelled chan struct{}
	replacementErr error
}

func (c *phase6TakeoverClient) Search(ctx context.Context, _ string, _ int) ([]models.NumistaCandidate, error) {
	if c.calls.Add(1) == 1 {
		close(c.firstStarted)
		<-ctx.Done()
		close(c.firstCancelled)
		return nil, ctx.Err()
	}
	return []models.NumistaCandidate{{ID: 99, Title: "Replacement"}}, c.replacementErr
}

func (c *phase6TakeoverClient) Detail(context.Context, int) (models.NumistaCandidate, error) {
	return models.NumistaCandidate{}, nil
}

type phase6LateResultClient struct {
	calls              atomic.Int32
	firstStarted       chan struct{}
	firstSawCancel     chan struct{}
	firstRelease       chan struct{}
	replacementStarted chan struct{}
	replacementRelease chan struct{}
	firstErr           error
	replacementErr     error
}

func (c *phase6LateResultClient) awaitResult(
	ctx context.Context,
) error {
	if c.calls.Add(1) == 1 {
		close(c.firstStarted)
		<-ctx.Done()
		close(c.firstSawCancel)
		<-c.firstRelease
		return c.firstErr
	}
	close(c.replacementStarted)
	<-c.replacementRelease
	return c.replacementErr
}

func (c *phase6LateResultClient) Search(
	ctx context.Context,
	_ string,
	_ int,
) ([]models.NumistaCandidate, error) {
	err := c.awaitResult(ctx)
	return []models.NumistaCandidate{{ID: 99, Title: "Accepted replacement"}}, err
}

func (c *phase6LateResultClient) Detail(
	ctx context.Context,
	id int,
) (models.NumistaCandidate, error) {
	err := c.awaitResult(ctx)
	return models.NumistaCandidate{ID: id, Title: "Accepted replacement detail"}, err
}

func TestNumistaLookupTelemetryIntegratesStatusesCacheRefreshAndLatency(t *testing.T) {
	retryAfter := 90
	clock := &fakeNumistaClock{now: time.Date(2026, 8, 11, 10, 0, 0, 0, time.UTC)}
	client := &phase6TelemetryClient{clock: clock, responses: map[string]phase6TelemetryResponse{
		"success":     {candidates: []models.NumistaCandidate{{ID: 1, Title: "Coin"}}, elapsed: 10 * time.Millisecond},
		"empty":       {elapsed: 20 * time.Millisecond},
		"quota":       {err: &NumistaError{Kind: NumistaErrorQuotaLimited, RetryAfterSeconds: &retryAfter}, elapsed: 30 * time.Millisecond},
		"timeout":     {err: &NumistaError{Kind: NumistaErrorTimeout}, elapsed: 40 * time.Millisecond},
		"unavailable": {err: &NumistaError{Kind: NumistaErrorUnavailable}, elapsed: 50 * time.Millisecond},
	}}
	settings := &fakeNumistaSettings{key: "configured", config: NumistaSettings{
		SearchTTL: time.Hour, SearchResultLimit: 20, Valid: true,
	}}
	telemetry := NewNumistaTelemetry(20)
	service := NewNumistaLookupService(
		client, NewNumistaCache(clock, 20, 20), NewNumistaV1Scorer(), telemetry, settings, clock,
	)

	for _, query := range []string{"success", "success", "empty", "quota", "timeout", "unavailable"} {
		if _, err := service.Lookup(context.Background(), models.NumistaLookupRequest{
			Query: query, Path: models.NumistaLookupPathDirect,
		}); err != nil {
			t.Fatalf("lookup %q error = %v", query, err)
		}
	}
	settings.key = ""
	if _, err := service.Lookup(context.Background(), models.NumistaLookupRequest{
		Query: "unconfigured", Path: models.NumistaLookupPathPhoto,
	}); err != nil {
		t.Fatalf("unconfigured lookup error = %v", err)
	}

	summary := telemetry.Health(false, false)
	for _, status := range []models.NumistaLookupStatus{
		models.NumistaStatusSuccess,
		models.NumistaStatusEmpty,
		models.NumistaStatusUnconfigured,
		models.NumistaStatusQuotaLimited,
		models.NumistaStatusTimeout,
		models.NumistaStatusUnavailable,
	} {
		want := 1
		if status == models.NumistaStatusSuccess {
			want = 1
		}
		if summary.StatusCounts[status] != want {
			t.Fatalf("status %q count = %d, want %d: %+v", status, summary.StatusCounts[status], want, summary)
		}
	}
	if summary.BroadRequestCount != 5 ||
		summary.FreshCacheHitCount != 1 || summary.CoalescedRequestCount != 0 ||
		summary.ProviderLoadCount != 5 || summary.ProviderFailureCount != 3 ||
		summary.FreshCacheHitRate != 1.0/6.0 ||
		summary.P50ElapsedMs != 30 || summary.P95ElapsedMs != 48 {
		t.Fatalf("unexpected broad/cache/latency summary: %+v", summary)
	}
	if summary.LastQuotaLimitedAt == nil || summary.LastRetryAfterSeconds == nil ||
		*summary.LastRetryAfterSeconds != retryAfter {
		t.Fatalf("quota timing missing from summary: %+v", summary)
	}
}

func TestNumistaLookupTelemetryUsesActualDetailClientRetryAndCachePath(t *testing.T) {
	clock := &fakeNumistaClock{now: time.Date(2026, 8, 11, 11, 0, 0, 0, time.UTC)}
	var attempts atomic.Int32
	transport := numistaRoundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.URL.Path != "/types/42" {
			t.Fatalf("unexpected detail path %q", request.URL.Path)
		}
		clock.Add(25 * time.Millisecond)
		if attempts.Add(1) == 1 {
			return &http.Response{
				StatusCode: http.StatusServiceUnavailable,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(`{}`)),
			}, nil
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body: io.NopCloser(strings.NewReader(
				`{"id":42,"title":"Detailed Coin","composition":{"text":"Silver"}}`,
			)),
		}, nil
	})
	client, err := NewHTTPNumistaClient(NumistaClientConfig{
		BaseURL: "https://numista.test", HTTPClient: &http.Client{Transport: transport},
		APIKey: func() string { return "server-only-key" }, RetrySleeper: immediateNumistaRetry,
	})
	if err != nil {
		t.Fatal(err)
	}
	settings := &fakeNumistaSettings{key: "configured", config: NumistaSettings{
		DetailTTL: 7 * 24 * time.Hour, Valid: true,
	}}
	telemetry := NewNumistaTelemetry(10)
	service := NewNumistaLookupService(
		client, NewNumistaCache(clock, 10, 10), NewNumistaV1Scorer(), telemetry, settings, clock,
	)

	detail, metadata, err := service.LookupDetail(context.Background(), models.NumistaLookupPathDirect, 42)
	if err != nil || detail.ID != 42 || detail.Material != "Silver" || metadata == nil || metadata.Hit {
		t.Fatalf("fresh detail=%+v metadata=%+v error=%v", detail, metadata, err)
	}
	cached, metadata, err := service.LookupDetail(context.Background(), models.NumistaLookupPathPhoto, 42)
	if err != nil || cached.ID != 42 || metadata == nil || !metadata.Hit ||
		cached.EnrichmentState != models.NumistaEnrichmentCached {
		t.Fatalf("cached detail=%+v metadata=%+v error=%v", cached, metadata, err)
	}

	summary := telemetry.Health(true, true)
	if attempts.Load() != 2 || summary.DetailRequestCount != 1 ||
		summary.ProviderLoadCount != 1 || summary.FreshCacheHitCount != 1 ||
		summary.CoalescedRequestCount != 0 || summary.ProviderFailureCount != 0 ||
		summary.EnrichmentAttempted != 1 || summary.EnrichmentSucceeded != 1 ||
		summary.EnrichmentFailed != 0 || summary.P50ElapsedMs != 50 || summary.P95ElapsedMs != 50 {
		t.Fatalf("unexpected actual detail aggregate attempts=%d summary=%+v", attempts.Load(), summary)
	}
	telemetry.mu.RLock()
	events := append([]NumistaTelemetryEvent(nil), telemetry.events...)
	telemetry.mu.RUnlock()
	if len(events) != 2 ||
		events[0].CacheOutcome != NumistaCacheOutcomeLoader || events[0].RetryCount != 1 ||
		events[1].CacheOutcome != NumistaCacheOutcomeFreshHit || events[1].RetryCount != 0 {
		t.Fatalf("detail retry/cache ownership is not truthful: %+v", events)
	}
}

func TestNumistaLookupTelemetryCoalescesDetailRetryOwnership(t *testing.T) {
	const callers = 5
	clock := &fakeNumistaClock{now: time.Date(2026, 8, 11, 11, 30, 0, 0, time.UTC)}
	started := make(chan struct{})
	release := make(chan struct{})
	var attempts atomic.Int32
	transport := numistaRoundTripFunc(func(*http.Request) (*http.Response, error) {
		attempt := attempts.Add(1)
		clock.Add(10 * time.Millisecond)
		if attempt == 1 {
			close(started)
			<-release
			return &http.Response{
				StatusCode: http.StatusServiceUnavailable,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(`{}`)),
			}, nil
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"id":7,"title":"Shared Detail"}`)),
		}, nil
	})
	client, err := NewHTTPNumistaClient(NumistaClientConfig{
		BaseURL: "https://numista.test", HTTPClient: &http.Client{Transport: transport},
		APIKey: func() string { return "server-only-key" }, RetrySleeper: immediateNumistaRetry,
	})
	if err != nil {
		t.Fatal(err)
	}
	settings := &fakeNumistaSettings{key: "configured", config: NumistaSettings{
		DetailTTL: 7 * 24 * time.Hour, Valid: true,
	}}
	telemetry := NewNumistaTelemetry(20)
	service := NewNumistaLookupService(
		client, NewNumistaCache(clock, 20, 20), NewNumistaV1Scorer(), telemetry, settings, clock,
	)

	results := make(chan *models.NumistaCacheMetadata, callers)
	errors := make(chan error, callers)
	for caller := range callers {
		go func(caller int) {
			path := models.NumistaLookupPathDirect
			if caller%2 != 0 {
				path = models.NumistaLookupPathPhoto
			}
			_, metadata, err := service.LookupDetail(context.Background(), path, 7)
			results <- metadata
			errors <- err
		}(caller)
	}
	<-started
	waitForDetailWaiters(t, service.cache, NumistaDetailCacheKey(7), callers)
	close(release)

	loaders := 0
	coalesced := 0
	for range callers {
		if err := <-errors; err != nil {
			t.Fatalf("detail fan-in error=%v", err)
		}
		metadata := <-results
		if metadata == nil {
			t.Fatal("detail fan-in returned no cache metadata")
		}
		if metadata.Hit {
			t.Fatalf("coalesced detail was reported as a fresh cache hit: %+v", metadata)
		}
		if metadata.Coalesced {
			coalesced++
		} else {
			loaders++
		}
	}
	summary := telemetry.Health(true, true)
	if attempts.Load() != 2 || loaders != 1 || coalesced != callers-1 ||
		summary.BroadRequestCount != 0 || summary.DetailRequestCount != 1 ||
		summary.ProviderLoadCount != 1 || summary.FreshCacheHitCount != 0 ||
		summary.CoalescedRequestCount != callers-1 || summary.ProviderFailureCount != 0 ||
		summary.EnrichmentAttempted != 1 || summary.EnrichmentSucceeded != 1 ||
		summary.P50ElapsedMs != 20 || summary.P95ElapsedMs != 20 {
		t.Fatalf(
			"detail fan-in attempts=%d loaders=%d coalesced=%d summary=%+v",
			attempts.Load(), loaders, coalesced, summary,
		)
	}
	telemetry.mu.RLock()
	events := append([]NumistaTelemetryEvent(nil), telemetry.events...)
	telemetry.mu.RUnlock()
	totalRetries := 0
	for _, event := range events {
		totalRetries += event.RetryCount
	}
	if totalRetries != 1 {
		t.Fatalf("detail retry was not owned by exactly one loader: %+v", events)
	}
}

func TestNumistaLookupTelemetryUsesExactPercentilesAndBoundedRetention(t *testing.T) {
	telemetry := NewNumistaTelemetry(5)
	for i, elapsed := range []int64{10, 20, 30, 40, 50, 60, 70} {
		telemetry.Record(NumistaTelemetryEvent{
			OccurredAt: time.Date(2026, 8, 11, 12, i, 0, 0, time.UTC),
			Path:       models.NumistaLookupPathDirect, Operation: "broad",
			Status: models.NumistaStatusSuccess, ElapsedMilliseconds: elapsed,
			CacheOutcome:      NumistaCacheOutcomeLoader,
			CorrelationDigest: NumistaCorrelationDigest(models.NumistaLookupPathDirect, "same query"),
		})
	}
	summary := telemetry.Health(true, true)
	if summary.BroadRequestCount != 5 || summary.P50ElapsedMs != 50 || summary.P95ElapsedMs != 68 {
		t.Fatalf("retained-window aggregate = %+v, want durations 30..70 with p50=50 p95=68", summary)
	}
	telemetry.mu.RLock()
	defer telemetry.mu.RUnlock()
	if len(telemetry.events) != 5 ||
		telemetry.events[0].ElapsedMilliseconds != 30 ||
		telemetry.events[4].ElapsedMilliseconds != 70 {
		t.Fatalf("retained events = %+v, want newest bounded window", telemetry.events)
	}
}

func TestNumistaLookupTelemetryUsesActualHTTPClientRetryCounts(t *testing.T) {
	clock := &fakeNumistaClock{now: time.Date(2026, 8, 11, 14, 0, 0, 0, time.UTC)}
	attempts := make(map[string]int)
	transport := numistaRoundTripFunc(func(request *http.Request) (*http.Response, error) {
		query := request.URL.Query().Get("q")
		attempts[query]++
		clock.Add(time.Duration(len(query)) * time.Millisecond)
		status := http.StatusOK
		body := `{"types":[]}`
		switch query {
		case "retry-one", "retry-another":
			if attempts[query] == 1 {
				status = http.StatusServiceUnavailable
				body = `{}`
			}
		case "bad-request":
			status = http.StatusBadRequest
		case "unauthorized":
			status = http.StatusUnauthorized
		case "forbidden":
			status = http.StatusForbidden
		case "quota":
			status = http.StatusTooManyRequests
		case "non-transient":
			status = http.StatusInternalServerError
		}
		return &http.Response{
			StatusCode: status,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(body)),
		}, nil
	})
	client, err := NewHTTPNumistaClient(NumistaClientConfig{
		BaseURL: "https://numista.test", HTTPClient: &http.Client{Transport: transport},
		APIKey: func() string { return "server-only-key" }, RetrySleeper: immediateNumistaRetry,
	})
	if err != nil {
		t.Fatal(err)
	}
	settings := &fakeNumistaSettings{key: "configured", config: NumistaSettings{
		SearchTTL: time.Hour, SearchResultLimit: 20, Valid: true,
	}}
	telemetry := NewNumistaTelemetry(20)
	service := NewNumistaLookupService(
		client, NewNumistaCache(clock, 20, 20), NewNumistaV1Scorer(), telemetry, settings, clock,
	)

	queries := []string{
		"no-retry", "retry-one", "retry-another", "bad-request",
		"unauthorized", "forbidden", "quota", "non-transient",
	}
	for _, query := range queries {
		if _, err := service.Lookup(context.Background(), models.NumistaLookupRequest{
			Query: query, Path: models.NumistaLookupPathDirect,
		}); err != nil {
			t.Fatalf("lookup %q error=%v", query, err)
		}
	}

	telemetry.mu.RLock()
	events := append([]NumistaTelemetryEvent(nil), telemetry.events...)
	telemetry.mu.RUnlock()
	if len(events) != len(queries) {
		t.Fatalf("events=%d, want %d", len(events), len(queries))
	}
	totalRetries := 0
	for i, event := range events {
		wantRetries := 0
		if queries[i] == "retry-one" || queries[i] == "retry-another" {
			wantRetries = 1
		}
		if event.RetryCount != wantRetries {
			t.Fatalf("%q retryCount=%d, want %d", queries[i], event.RetryCount, wantRetries)
		}
		if event.ElapsedMilliseconds <= 0 {
			t.Fatalf("%q elapsed=%d, want real execution timing", queries[i], event.ElapsedMilliseconds)
		}
		totalRetries += event.RetryCount
	}
	if totalRetries != 2 {
		t.Fatalf("aggregate retries=%d, want multiple retries across real operations", totalRetries)
	}
	for query, wantAttempts := range map[string]int{
		"no-retry": 1, "retry-one": 2, "retry-another": 2,
		"bad-request": 1, "unauthorized": 1, "forbidden": 1,
		"quota": 1, "non-transient": 1,
	} {
		if attempts[query] != wantAttempts {
			t.Fatalf("%s attempts=%s, want %d", query, strconv.Itoa(attempts[query]), wantAttempts)
		}
	}
}

func TestNumistaLookupTelemetryColdFanInOwnershipStress(t *testing.T) {
	const iterations = 100
	const callers = 6
	for iteration := range iterations {
		clock := &fakeNumistaClock{now: time.Date(2026, 8, 11, 15, 0, 0, 0, time.UTC)}
		started := make(chan struct{})
		release := make(chan struct{})
		var attempts atomic.Int32
		succeed := iteration%2 == 0
		transport := numistaRoundTripFunc(func(*http.Request) (*http.Response, error) {
			attempt := attempts.Add(1)
			clock.Add(5 * time.Millisecond)
			if attempt == 1 {
				close(started)
				<-release
				if succeed {
					return &http.Response{
						StatusCode: http.StatusServiceUnavailable,
						Header:     make(http.Header),
						Body:       io.NopCloser(strings.NewReader(`{}`)),
					}, nil
				}
			}
			status := http.StatusInternalServerError
			body := `{}`
			if succeed {
				status = http.StatusOK
				body = `{"types":[{"id":1,"title":"Coin"}]}`
			}
			return &http.Response{
				StatusCode: status,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(body)),
			}, nil
		})
		client, err := NewHTTPNumistaClient(NumistaClientConfig{
			BaseURL: "https://numista.test", HTTPClient: &http.Client{Transport: transport},
			APIKey: func() string { return "server-only-key" }, RetrySleeper: immediateNumistaRetry,
		})
		if err != nil {
			t.Fatal(err)
		}
		settings := &fakeNumistaSettings{key: "configured", config: NumistaSettings{
			SearchTTL: time.Hour, SearchResultLimit: 20, Valid: true,
		}}
		telemetry := NewNumistaTelemetry(20)
		service := NewNumistaLookupService(
			client, NewNumistaCache(clock, 20, 20), NewNumistaV1Scorer(), telemetry, settings, clock,
		)
		results := make(chan models.NumistaLookupOutcome, callers)
		errors := make(chan error, callers)
		startCallers := make(chan struct{})
		for caller := range callers {
			go func(caller int) {
				<-startCallers
				path := models.NumistaLookupPathDirect
				if caller%2 != 0 {
					path = models.NumistaLookupPathPhoto
				}
				outcome, err := service.Lookup(context.Background(), models.NumistaLookupRequest{
					Query: "shared fan in", Path: path,
				})
				results <- outcome
				errors <- err
			}(caller)
		}
		close(startCallers)
		<-started
		waitForSearchWaiters(t, service.cache, NumistaSearchCacheKey("shared fan in", 20), callers)
		close(release)

		loaderResponses := 0
		coalescedResponses := 0
		wantStatus := models.NumistaStatusUnavailable
		if succeed {
			wantStatus = models.NumistaStatusSuccess
		}
		for range callers {
			if err := <-errors; err != nil {
				t.Fatalf("iteration %d lookup error=%v", iteration, err)
			}
			outcome := <-results
			if outcome.Status != wantStatus {
				t.Fatalf("iteration %d status=%q, want %q", iteration, outcome.Status, wantStatus)
			}
			if outcome.Cache != nil {
				if outcome.Cache.Hit {
					t.Fatalf("iteration %d cold fan-in reported a fresh cache hit: %+v", iteration, outcome.Cache)
				}
				if outcome.Cache.Coalesced {
					coalescedResponses++
				} else {
					loaderResponses++
				}
			}
		}
		wantAttempts := int32(1)
		if succeed {
			wantAttempts = 2
			if loaderResponses != 1 || coalescedResponses != callers-1 {
				t.Fatalf(
					"iteration %d loader responses=%d coalesced responses=%d",
					iteration, loaderResponses, coalescedResponses,
				)
			}
		}
		if attempts.Load() != wantAttempts {
			t.Fatalf("iteration %d HTTP attempts=%d, want %d", iteration, attempts.Load(), wantAttempts)
		}
		summary := telemetry.Health(true, true)
		wantFailures := 0
		if !succeed {
			wantFailures = 1
		}
		if summary.BroadRequestCount != 1 || summary.ProviderLoadCount != 1 ||
			summary.FreshCacheHitCount != 0 || summary.CoalescedRequestCount != callers-1 ||
			summary.ProviderFailureCount != wantFailures || summary.StatusCounts[wantStatus] != 1 {
			t.Fatalf("iteration %d ownership summary=%+v", iteration, summary)
		}
		telemetry.mu.RLock()
		events := append([]NumistaTelemetryEvent(nil), telemetry.events...)
		telemetry.mu.RUnlock()
		loaderEvents := 0
		coalescedEvents := 0
		totalRetries := 0
		for _, event := range events {
			switch event.CacheOutcome {
			case NumistaCacheOutcomeLoader:
				loaderEvents++
			case NumistaCacheOutcomeCoalescedWaiter:
				coalescedEvents++
			}
			totalRetries += event.RetryCount
		}
		wantRetries := 0
		if succeed {
			wantRetries = 1
		}
		if loaderEvents != 1 || coalescedEvents != callers-1 || totalRetries != wantRetries {
			t.Fatalf(
				"iteration %d loader=%d coalesced=%d retries=%d events=%+v",
				iteration, loaderEvents, coalescedEvents, totalRetries, events,
			)
		}
	}
}

func TestNumistaLookupTelemetryCancellationAndReplacementOwnership(t *testing.T) {
	client := &phase6TakeoverClient{
		firstStarted: make(chan struct{}), firstCancelled: make(chan struct{}),
	}
	settings := &fakeNumistaSettings{key: "configured", config: NumistaSettings{
		SearchTTL: time.Hour, SearchResultLimit: 20, Valid: true,
	}}
	telemetry := NewNumistaTelemetry(20)
	service := NewNumistaLookupService(
		client, NewNumistaCache(nil, 20, 20), NewNumistaV1Scorer(), telemetry, settings, nil,
	)

	const callers = 3
	cancels := make([]context.CancelFunc, 0, callers)
	results := make(chan error, callers)
	for range callers {
		ctx, cancel := context.WithCancel(context.Background())
		cancels = append(cancels, cancel)
		go func() {
			_, err := service.Lookup(ctx, models.NumistaLookupRequest{
				Query: "cancel takeover", Path: models.NumistaLookupPathDirect,
			})
			results <- err
		}()
	}
	<-client.firstStarted
	waitForSearchWaiters(t, service.cache, NumistaSearchCacheKey("cancel takeover", 20), callers)
	for _, cancel := range cancels {
		cancel()
	}
	for range callers {
		if err := <-results; !errors.Is(err, context.Canceled) {
			t.Fatalf("cancelled lookup error=%v", err)
		}
	}
	<-client.firstCancelled

	replacement, err := service.Lookup(context.Background(), models.NumistaLookupRequest{
		Query: "cancel takeover", Path: models.NumistaLookupPathPhoto,
	})
	if err != nil || replacement.Status != models.NumistaStatusSuccess ||
		replacement.Cache == nil || replacement.Cache.Hit {
		t.Fatalf("replacement outcome=%+v error=%v", replacement, err)
	}
	summary := telemetry.Health(true, true)
	if client.calls.Load() != 2 || summary.BroadRequestCount != 1 ||
		summary.ProviderLoadCount != 1 || summary.FreshCacheHitCount != 0 ||
		summary.CoalescedRequestCount != callers-1 || summary.CancelledRequestCount != callers ||
		summary.ProviderFailureCount != 0 ||
		summary.StatusCounts[models.NumistaStatusSuccess] != 1 ||
		summary.P50ElapsedMs < 0 || summary.P95ElapsedMs < 0 {
		t.Fatalf("takeover ownership calls=%d summary=%+v", client.calls.Load(), summary)
	}
	telemetry.mu.RLock()
	events := append([]NumistaTelemetryEvent(nil), telemetry.events...)
	telemetry.mu.RUnlock()
	loaderEvents := 0
	successfulProvider := 0
	for _, event := range events {
		if event.CacheOutcome != NumistaCacheOutcomeLoader {
			continue
		}
		loaderEvents++
		if event.Status == models.NumistaStatusSuccess {
			successfulProvider++
		}
	}
	if loaderEvents != 1 || successfulProvider != 1 {
		t.Fatalf("provider ownership after takeover is not exact: %+v", events)
	}
}

func TestNumistaLookupTelemetryOrphanedLateSearchResultCannotEmit(t *testing.T) {
	for _, test := range []struct {
		name           string
		firstErr       error
		replacementErr error
		wantStatus     models.NumistaLookupStatus
		wantFailures   int
	}{
		{
			name:       "late success replacement success",
			wantStatus: models.NumistaStatusSuccess,
		},
		{
			name:       "late failure replacement success",
			firstErr:   &NumistaError{Kind: NumistaErrorUnavailable},
			wantStatus: models.NumistaStatusSuccess,
		},
		{
			name:           "late success replacement failure",
			replacementErr: &NumistaError{Kind: NumistaErrorUnavailable},
			wantStatus:     models.NumistaStatusUnavailable,
			wantFailures:   1,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			client := &phase6LateResultClient{
				firstStarted:       make(chan struct{}),
				firstSawCancel:     make(chan struct{}),
				firstRelease:       make(chan struct{}),
				replacementStarted: make(chan struct{}),
				replacementRelease: make(chan struct{}),
				firstErr:           test.firstErr,
				replacementErr:     test.replacementErr,
			}
			var releaseFirst, releaseReplacement sync.Once
			t.Cleanup(func() {
				releaseFirst.Do(func() { close(client.firstRelease) })
				releaseReplacement.Do(func() { close(client.replacementRelease) })
			})
			telemetry := NewNumistaTelemetry(20)
			settings := &fakeNumistaSettings{key: "configured", config: NumistaSettings{
				SearchTTL: time.Hour, SearchResultLimit: 20, Valid: true,
			}}
			service := NewNumistaLookupService(
				client, NewNumistaCache(nil, 20, 20), NewNumistaV1Scorer(),
				telemetry, settings, nil,
			)
			request := models.NumistaLookupRequest{
				Query: "late orphan", Path: models.NumistaLookupPathDirect,
			}

			const callers = 3
			cancels := make([]context.CancelFunc, 0, callers)
			cancelled := make(chan error, callers)
			for range callers {
				ctx, cancel := context.WithCancel(context.Background())
				cancels = append(cancels, cancel)
				go func() {
					_, err := service.Lookup(ctx, request)
					cancelled <- err
				}()
			}
			<-client.firstStarted
			key := NumistaSearchCacheKey(request.Query, 20)
			waitForSearchWaiters(t, service.cache, key, callers)
			service.cache.mu.Lock()
			orphan := service.cache.searchInflight[key]
			service.cache.mu.Unlock()
			for _, cancel := range cancels {
				cancel()
			}
			for range callers {
				if err := <-cancelled; !errors.Is(err, context.Canceled) {
					t.Fatalf("cancelled caller error=%v", err)
				}
			}
			<-client.firstSawCancel

			type lookupResult struct {
				outcome models.NumistaLookupOutcome
				err     error
			}
			replacementDone := make(chan lookupResult, 1)
			go func() {
				outcome, err := service.Lookup(context.Background(), models.NumistaLookupRequest{
					Query: request.Query, Path: models.NumistaLookupPathPhoto,
				})
				replacementDone <- lookupResult{outcome: outcome, err: err}
			}()
			<-client.replacementStarted
			service.cache.mu.Lock()
			replacement := service.cache.searchInflight[key]
			service.cache.mu.Unlock()
			if orphan == nil || replacement == nil || orphan == replacement {
				t.Fatalf("in-flight ownership was not superseded: orphan=%p replacement=%p", orphan, replacement)
			}

			releaseFirst.Do(func() { close(client.firstRelease) })
			<-orphan.done
			assertOnlyCancellationTelemetry(t, telemetry, callers, callers-1)

			releaseReplacement.Do(func() { close(client.replacementRelease) })
			result := <-replacementDone
			if result.err != nil || result.outcome.Status != test.wantStatus {
				t.Fatalf("replacement outcome=%+v err=%v", result.outcome, result.err)
			}
			assertProviderOwnershipTelemetry(
				t, telemetry, callers, callers-1, 1, 1, 0, test.wantFailures, test.wantStatus,
			)
			if client.calls.Load() != 2 {
				t.Fatalf("provider calls=%d, want orphan plus replacement", client.calls.Load())
			}
		})
	}
}

func TestNumistaLookupTelemetryOrphanedLateDetailResultCannotEmit(t *testing.T) {
	for _, test := range []struct {
		name           string
		firstErr       error
		replacementErr error
		wantFailures   int
	}{
		{name: "late success replacement success"},
		{
			name:     "late failure replacement success",
			firstErr: &NumistaError{Kind: NumistaErrorUnavailable},
		},
		{
			name:           "late success replacement failure",
			replacementErr: &NumistaError{Kind: NumistaErrorUnavailable},
			wantFailures:   1,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			client := &phase6LateResultClient{
				firstStarted:       make(chan struct{}),
				firstSawCancel:     make(chan struct{}),
				firstRelease:       make(chan struct{}),
				replacementStarted: make(chan struct{}),
				replacementRelease: make(chan struct{}),
				firstErr:           test.firstErr,
				replacementErr:     test.replacementErr,
			}
			var releaseFirst, releaseReplacement sync.Once
			t.Cleanup(func() {
				releaseFirst.Do(func() { close(client.firstRelease) })
				releaseReplacement.Do(func() { close(client.replacementRelease) })
			})
			telemetry := NewNumistaTelemetry(20)
			settings := &fakeNumistaSettings{key: "configured", config: NumistaSettings{
				DetailTTL: 7 * 24 * time.Hour, Valid: true,
			}}
			service := NewNumistaLookupService(
				client, NewNumistaCache(nil, 20, 20), NewNumistaV1Scorer(),
				telemetry, settings, nil,
			)

			const callers = 2
			cancels := make([]context.CancelFunc, 0, callers)
			cancelled := make(chan error, callers)
			for range callers {
				ctx, cancel := context.WithCancel(context.Background())
				cancels = append(cancels, cancel)
				go func() {
					_, _, err := service.LookupDetail(ctx, models.NumistaLookupPathPhoto, 42)
					cancelled <- err
				}()
			}
			<-client.firstStarted
			key := NumistaDetailCacheKey(42)
			waitForDetailWaiters(t, service.cache, key, callers)
			service.cache.mu.Lock()
			orphan := service.cache.detailInflight[key]
			service.cache.mu.Unlock()
			for _, cancel := range cancels {
				cancel()
			}
			for range callers {
				if err := <-cancelled; !errors.Is(err, context.Canceled) {
					t.Fatalf("cancelled detail caller error=%v", err)
				}
			}
			<-client.firstSawCancel

			replacementDone := make(chan error, 1)
			go func() {
				_, _, err := service.LookupDetail(
					context.Background(), models.NumistaLookupPathDirect, 42,
				)
				replacementDone <- err
			}()
			<-client.replacementStarted
			service.cache.mu.Lock()
			replacement := service.cache.detailInflight[key]
			service.cache.mu.Unlock()
			if orphan == nil || replacement == nil || orphan == replacement {
				t.Fatalf("detail ownership was not superseded: orphan=%p replacement=%p", orphan, replacement)
			}

			releaseFirst.Do(func() { close(client.firstRelease) })
			<-orphan.done
			assertOnlyCancellationTelemetry(t, telemetry, callers, callers-1)

			releaseReplacement.Do(func() { close(client.replacementRelease) })
			err := <-replacementDone
			if test.replacementErr == nil && err != nil {
				t.Fatalf("replacement detail error=%v", err)
			}
			if test.replacementErr != nil && err == nil {
				t.Fatal("replacement detail unexpectedly succeeded")
			}
			status := models.NumistaStatusSuccess
			succeeded := 1
			failed := 0
			if test.replacementErr != nil {
				status = models.NumistaStatusUnavailable
				succeeded = 0
				failed = 1
			}
			assertProviderOwnershipTelemetry(
				t, telemetry, callers, callers-1, 1, 0, 1, test.wantFailures, status,
			)
			summary := telemetry.Health(true, true)
			if summary.EnrichmentAttempted != 1 ||
				summary.EnrichmentSucceeded != succeeded ||
				summary.EnrichmentFailed != failed {
				t.Fatalf("detail enrichment ownership summary=%+v", summary)
			}
			if client.calls.Load() != 2 {
				t.Fatalf("detail provider calls=%d, want orphan plus replacement", client.calls.Load())
			}
		})
	}
}

func assertOnlyCancellationTelemetry(
	t *testing.T,
	telemetry *NumistaTelemetry,
	cancelled int,
	coalesced int,
) {
	t.Helper()
	summary := telemetry.Health(true, true)
	if summary.CancelledRequestCount != cancelled ||
		summary.CoalescedRequestCount != coalesced ||
		summary.ProviderLoadCount != 0 ||
		summary.BroadRequestCount != 0 ||
		summary.DetailRequestCount != 0 ||
		summary.ProviderFailureCount != 0 ||
		summary.EnrichmentAttempted != 0 ||
		summary.EnrichmentSucceeded != 0 ||
		summary.EnrichmentFailed != 0 ||
		len(summary.StatusCounts) != 0 ||
		summary.P50ElapsedMs != 0 ||
		summary.P95ElapsedMs != 0 {
		t.Fatalf("orphaned result polluted provider telemetry: %+v", summary)
	}
	telemetry.mu.RLock()
	events := append([]NumistaTelemetryEvent(nil), telemetry.events...)
	telemetry.mu.RUnlock()
	for _, event := range events {
		if event.CacheOutcome == NumistaCacheOutcomeLoader ||
			event.Status != "" ||
			event.CandidateCount != 0 ||
			event.RetryCount != 0 ||
			event.RetryAfterSeconds != nil ||
			event.DetailAttemptCount != 0 ||
			event.DetailSuccessCount != 0 ||
			event.DetailFailureCount != 0 {
			t.Fatalf("orphaned provider payload was emitted: %+v", events)
		}
	}
}

func assertProviderOwnershipTelemetry(
	t *testing.T,
	telemetry *NumistaTelemetry,
	cancelled int,
	coalesced int,
	providerLoads int,
	broadRequests int,
	detailRequests int,
	providerFailures int,
	status models.NumistaLookupStatus,
) {
	t.Helper()
	summary := telemetry.Health(true, true)
	if summary.CancelledRequestCount != cancelled ||
		summary.CoalescedRequestCount != coalesced ||
		summary.ProviderLoadCount != providerLoads ||
		summary.BroadRequestCount != broadRequests ||
		summary.DetailRequestCount != detailRequests ||
		summary.ProviderFailureCount != providerFailures ||
		summary.StatusCounts[status] != 1 {
		t.Fatalf("accepted provider ownership summary=%+v", summary)
	}
	telemetry.mu.RLock()
	events := append([]NumistaTelemetryEvent(nil), telemetry.events...)
	telemetry.mu.RUnlock()
	providerEvents := 0
	for _, event := range events {
		if event.CacheOutcome == NumistaCacheOutcomeLoader {
			providerEvents++
		}
	}
	if providerEvents != providerLoads {
		t.Fatalf("provider events=%d, want %d: %+v", providerEvents, providerLoads, events)
	}
}

func TestNumistaLookupTelemetryCancelledLeaderWithHealthyWaiter(t *testing.T) {
	clock := &fakeNumistaClock{now: time.Date(2026, 8, 11, 16, 0, 0, 0, time.UTC)}
	started := make(chan struct{})
	release := make(chan struct{})
	var calls atomic.Int32
	client := &contextNumistaClient{search: func(ctx context.Context) ([]models.NumistaCandidate, error) {
		calls.Add(1)
		close(started)
		select {
		case <-release:
			clock.Add(40 * time.Millisecond)
			return []models.NumistaCandidate{{ID: 7, Title: "Healthy"}}, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}}
	settings := &fakeNumistaSettings{key: "configured", config: NumistaSettings{
		SearchTTL: time.Hour, SearchResultLimit: 20, Valid: true,
	}}
	telemetry := NewNumistaTelemetry(10)
	service := NewNumistaLookupService(
		client, NewNumistaCache(clock, 10, 10), NewNumistaV1Scorer(), telemetry, settings, clock,
	)
	request := models.NumistaLookupRequest{Query: "healthy takeover", Path: models.NumistaLookupPathDirect}

	leaderCtx, cancelLeader := context.WithCancel(context.Background())
	leaderDone := make(chan error, 1)
	go func() {
		_, err := service.Lookup(leaderCtx, request)
		leaderDone <- err
	}()
	<-started
	waiterDone := make(chan error, 1)
	go func() {
		outcome, err := service.Lookup(context.Background(), request)
		if err == nil && outcome.Status != models.NumistaStatusSuccess {
			err = errors.New("healthy waiter did not succeed")
		}
		waiterDone <- err
	}()
	waitForSearchWaiters(t, service.cache, NumistaSearchCacheKey(request.Query, 20), 2)
	cancelLeader()
	if err := <-leaderDone; !errors.Is(err, context.Canceled) {
		t.Fatalf("leader error=%v, want context.Canceled", err)
	}
	close(release)
	if err := <-waiterDone; err != nil {
		t.Fatal(err)
	}

	summary := telemetry.Health(true, true)
	if calls.Load() != 1 || summary.BroadRequestCount != 1 || summary.ProviderLoadCount != 1 ||
		summary.ProviderFailureCount != 0 || summary.CancelledRequestCount != 1 ||
		summary.CoalescedRequestCount != 1 || summary.StatusCounts[models.NumistaStatusSuccess] != 1 ||
		summary.P50ElapsedMs != 40 || summary.P95ElapsedMs != 40 {
		t.Fatalf("healthy waiter ownership calls=%d summary=%+v", calls.Load(), summary)
	}
}

func TestNumistaLookupTelemetryCallerDeadlineDoesNotPolluteProviderAggregates(t *testing.T) {
	finished := make(chan struct{})
	client := &contextNumistaClient{search: func(ctx context.Context) ([]models.NumistaCandidate, error) {
		<-ctx.Done()
		close(finished)
		return nil, ctx.Err()
	}}
	settings := &fakeNumistaSettings{key: "configured", config: NumistaSettings{
		SearchTTL: time.Hour, SearchResultLimit: 20, Valid: true,
	}}
	telemetry := NewNumistaTelemetry(10)
	service := NewNumistaLookupService(
		client, NewNumistaCache(nil, 10, 10), NewNumistaV1Scorer(), telemetry, settings, nil,
	)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	if _, err := service.Lookup(ctx, models.NumistaLookupRequest{
		Query: "deadline", Path: models.NumistaLookupPathDirect,
	}); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("deadline error=%v", err)
	}
	<-finished
	summary := telemetry.Health(true, true)
	if summary.CancelledRequestCount != 1 || summary.ProviderLoadCount != 0 ||
		summary.BroadRequestCount != 0 || summary.ProviderFailureCount != 0 ||
		len(summary.StatusCounts) != 0 || summary.P50ElapsedMs != 0 || summary.P95ElapsedMs != 0 {
		t.Fatalf("deadline polluted provider aggregates: %+v", summary)
	}
}

func TestNumistaLookupDetailDeadlineDoesNotPolluteEnrichmentAggregates(t *testing.T) {
	finished := make(chan struct{})
	client := &contextNumistaClient{detail: func(ctx context.Context) (models.NumistaCandidate, error) {
		<-ctx.Done()
		close(finished)
		return models.NumistaCandidate{}, ctx.Err()
	}}
	settings := &fakeNumistaSettings{key: "configured", config: NumistaSettings{
		DetailTTL: 7 * 24 * time.Hour, Valid: true,
	}}
	telemetry := NewNumistaTelemetry(10)
	service := NewNumistaLookupService(
		client, NewNumistaCache(nil, 10, 10), NewNumistaV1Scorer(), telemetry, settings, nil,
	)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	if _, _, err := service.LookupDetail(ctx, models.NumistaLookupPathPhoto, 42); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("detail deadline error=%v", err)
	}
	<-finished
	summary := telemetry.Health(true, true)
	if summary.CancelledRequestCount != 1 || summary.ProviderLoadCount != 0 ||
		summary.DetailRequestCount != 0 || summary.ProviderFailureCount != 0 ||
		summary.EnrichmentAttempted != 0 || summary.EnrichmentSucceeded != 0 ||
		summary.EnrichmentFailed != 0 || len(summary.StatusCounts) != 0 ||
		summary.P50ElapsedMs != 0 || summary.P95ElapsedMs != 0 {
		t.Fatalf("detail deadline polluted provider/enrichment aggregates: %+v", summary)
	}
}

func TestNumistaLookupTelemetryCancelledPredecessorReplacementFailureOwnership(t *testing.T) {
	client := &phase6TakeoverClient{
		firstStarted: make(chan struct{}), firstCancelled: make(chan struct{}),
		replacementErr: &NumistaError{Kind: NumistaErrorUnavailable},
	}
	settings := &fakeNumistaSettings{key: "configured", config: NumistaSettings{
		SearchTTL: time.Hour, SearchResultLimit: 20, Valid: true,
	}}
	telemetry := NewNumistaTelemetry(10)
	service := NewNumistaLookupService(
		client, NewNumistaCache(nil, 10, 10), NewNumistaV1Scorer(), telemetry, settings, nil,
	)

	const callers = 2
	cancels := make([]context.CancelFunc, 0, callers)
	results := make(chan error, callers)
	for range callers {
		ctx, cancel := context.WithCancel(context.Background())
		cancels = append(cancels, cancel)
		go func(callCtx context.Context) {
			_, err := service.Lookup(callCtx, models.NumistaLookupRequest{
				Query: "failed replacement", Path: models.NumistaLookupPathDirect,
			})
			results <- err
		}(ctx)
	}
	<-client.firstStarted
	waitForSearchWaiters(t, service.cache, NumistaSearchCacheKey("failed replacement", 20), callers)
	for _, cancel := range cancels {
		cancel()
	}
	for range callers {
		if err := <-results; !errors.Is(err, context.Canceled) {
			t.Fatalf("cancelled predecessor error=%v", err)
		}
	}
	<-client.firstCancelled

	outcome, err := service.Lookup(context.Background(), models.NumistaLookupRequest{
		Query: "failed replacement", Path: models.NumistaLookupPathPhoto,
	})
	if err != nil || outcome.Status != models.NumistaStatusUnavailable {
		t.Fatalf("replacement outcome=%+v err=%v", outcome, err)
	}
	summary := telemetry.Health(true, true)
	if client.calls.Load() != 2 || summary.CancelledRequestCount != callers ||
		summary.CoalescedRequestCount != callers-1 || summary.ProviderLoadCount != 1 ||
		summary.BroadRequestCount != 1 || summary.ProviderFailureCount != 1 ||
		summary.StatusCounts[models.NumistaStatusUnavailable] != 1 {
		t.Fatalf("replacement failure calls=%d summary=%+v", client.calls.Load(), summary)
	}
}

type contextNumistaClient struct {
	search func(context.Context) ([]models.NumistaCandidate, error)
	detail func(context.Context) (models.NumistaCandidate, error)
}

func (c *contextNumistaClient) Search(
	ctx context.Context,
	_ string,
	_ int,
) ([]models.NumistaCandidate, error) {
	if c.search == nil {
		return nil, errors.New("not implemented")
	}
	return c.search(ctx)
}

func (c *contextNumistaClient) Detail(ctx context.Context, _ int) (models.NumistaCandidate, error) {
	if c.detail == nil {
		return models.NumistaCandidate{}, errors.New("not implemented")
	}
	return c.detail(ctx)
}

func TestNumistaLookupTelemetryCannotRetainOrExposeSensitiveInputs(t *testing.T) {
	eventType := reflect.TypeOf(NumistaTelemetryEvent{})
	for index := 0; index < eventType.NumField(); index++ {
		field := strings.ToLower(eventType.Field(index).Name)
		for _, prohibited := range []string{
			"key", "query", "inscription", "label", "evidence", "rawerror", "errorbody", "requestbody", "responsebody",
		} {
			if strings.Contains(field, prohibited) {
				t.Fatalf("telemetry event field %q can retain prohibited %q data", eventType.Field(index).Name, prohibited)
			}

		}
	}

	const sensitive = "numista-key-secret IMP-TRAIANO obverse-inscription dealer-label raw-provider-error"
	telemetry := NewNumistaTelemetry(2)
	telemetry.Record(NumistaTelemetryEvent{
		OccurredAt: time.Date(2026, 8, 11, 13, 0, 0, 0, time.UTC),
		Path:       models.NumistaLookupPathPhoto, Operation: "broad", Status: models.NumistaStatusSuccess,
		CorrelationDigest: sensitive,
	})
	telemetry.mu.RLock()
	stored := append([]NumistaTelemetryEvent(nil), telemetry.events...)
	telemetry.mu.RUnlock()
	eventJSON, err := json.Marshal(stored)
	if err != nil {
		t.Fatal(err)
	}
	summaryJSON, err := json.Marshal(telemetry.Health(true, true))
	if err != nil {
		t.Fatal(err)
	}
	for _, data := range [][]byte{eventJSON, summaryJSON} {
		text := string(data)
		for _, forbidden := range strings.Fields(sensitive) {
			if strings.Contains(text, forbidden) {
				t.Fatalf("telemetry JSON leaked %q: %s", forbidden, text)
			}
		}
	}
	if len(stored) != 1 || len(stored[0].CorrelationDigest) != 16 ||
		stored[0].CorrelationDigest == sensitive {
		t.Fatalf("correlation value was not replaced by a bounded digest: %+v", stored)
	}
}
