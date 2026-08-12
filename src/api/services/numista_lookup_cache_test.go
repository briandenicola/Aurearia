package services

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
)

type phase6CacheClient struct {
	calls      atomic.Int32
	mu         sync.Mutex
	candidates []models.NumistaCandidate
	started    chan struct{}
	release    chan struct{}
}

func (c *phase6CacheClient) Search(ctx context.Context, _ string, _ int) ([]models.NumistaCandidate, error) {
	call := c.calls.Add(1)
	if call == 1 && c.started != nil {
		close(c.started)
	}
	if c.release != nil {
		select {
		case <-c.release:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return cloneCandidates(c.candidates), nil
}

func (c *phase6CacheClient) Detail(context.Context, int) (models.NumistaCandidate, error) {
	return models.NumistaCandidate{}, nil
}

func (c *phase6CacheClient) setCandidates(candidates []models.NumistaCandidate) {
	c.mu.Lock()
	c.candidates = cloneCandidates(candidates)
	c.mu.Unlock()
}

func newPhase6CacheService(
	client NumistaClient,
	cache *NumistaCache,
	settings *fakeNumistaSettings,
	clock NumistaClock,
) *NumistaLookupService {
	return NewNumistaLookupService(
		client, cache, NewNumistaV1Scorer(), NewNumistaTelemetry(50), settings, clock,
	)
}

func TestNumistaLookupCacheReusesNormalizedQueriesAcrossPaths(t *testing.T) {
	client := &phase6CacheClient{candidates: []models.NumistaCandidate{{ID: 42, Title: "Trajan Denarius"}}}
	settings := &fakeNumistaSettings{key: "configured", config: NumistaSettings{
		SearchTTL: time.Hour, SearchResultLimit: 20, Valid: true,
	}}
	service := newPhase6CacheService(client, NewNumistaCache(nil, 20, 20), settings, nil)

	direct, err := service.Lookup(context.Background(), models.NumistaLookupRequest{
		Query: "Trajan — Denarius", Path: models.NumistaLookupPathDirect,
	})
	if err != nil || direct.Status != models.NumistaStatusSuccess || direct.Cache == nil || direct.Cache.Hit {
		t.Fatalf("direct lookup = %+v, error = %v", direct, err)
	}
	photo, err := service.Lookup(context.Background(), models.NumistaLookupRequest{
		Query: "  trajan denarius  ", Path: models.NumistaLookupPathPhoto,
	})
	if err != nil || photo.Status != models.NumistaStatusSuccess || photo.Cache == nil || !photo.Cache.Hit {
		t.Fatalf("photo lookup = %+v, error = %v", photo, err)
	}
	if client.calls.Load() != 1 {
		t.Fatalf("provider calls = %d, want one normalized cross-path call", client.calls.Load())
	}
	if photo.EffectiveQuery != "  trajan denarius  " {
		t.Fatalf("effective query = %q, want exact submitted query", photo.EffectiveQuery)
	}
}

func TestNumistaLookupCacheReusesFreshEmptyAndRefreshesAtExpiry(t *testing.T) {
	clock := &fakeNumistaClock{now: time.Date(2026, 8, 11, 12, 0, 0, 0, time.UTC)}
	client := &phase6CacheClient{}
	settings := &fakeNumistaSettings{key: "configured", config: NumistaSettings{
		SearchTTL: time.Hour, SearchResultLimit: 20, Valid: true,
	}}
	service := newPhase6CacheService(client, NewNumistaCache(clock, 20, 20), settings, clock)
	request := models.NumistaLookupRequest{Query: "unlisted provincial coin", Path: models.NumistaLookupPathDirect}

	first, err := service.Lookup(context.Background(), request)
	if err != nil || first.Status != models.NumistaStatusEmpty || first.Cache == nil || first.Cache.Hit {
		t.Fatalf("fresh empty lookup = %+v, error = %v", first, err)
	}

	clock.Add(59 * time.Minute)
	second, err := service.Lookup(context.Background(), request)
	if err != nil || second.Status != models.NumistaStatusEmpty || second.Cache == nil ||
		!second.Cache.Hit || second.Cache.AgeSeconds != 59*60 {
		t.Fatalf("cached empty lookup = %+v, error = %v", second, err)
	}

	client.setCandidates([]models.NumistaCandidate{{ID: 77, Title: "Refreshed Result"}})
	clock.Add(time.Minute)
	third, err := service.Lookup(context.Background(), request)
	if err != nil || third.Status != models.NumistaStatusSuccess || third.Cache == nil || third.Cache.Hit {
		t.Fatalf("expired lookup = %+v, error = %v", third, err)
	}
	if client.calls.Load() != 2 {
		t.Fatalf("provider calls = %d, want initial load plus expiry refresh", client.calls.Load())
	}
	if !third.Cache.CreatedAt.Equal(clock.Now()) || !third.Cache.ExpiresAt.Equal(clock.Now().Add(time.Hour)) {
		t.Fatalf("refresh metadata = %+v, want fake-clock creation and expiry", third.Cache)
	}
}

func TestNumistaLookupCacheKeepsPrimaryAndRelaxedQueriesIndependent(t *testing.T) {
	client := &sequenceNumistaClient{results: map[string][]models.NumistaCandidate{
		"Honorius GLORIA ROMANORVM Nicomedia": {},
		"Honorius Nicomedia":                  {{ID: 208360, Title: "AE3 - Honorius"}},
	}}
	settings := &fakeNumistaSettings{key: "configured", config: NumistaSettings{
		SearchTTL: time.Hour, SearchResultLimit: 20, Valid: true,
	}}
	service := newPhase6CacheService(client, NewNumistaCache(nil, 20, 20), settings, nil)
	request := models.NumistaLookupRequest{
		Query: "Honorius GLORIA ROMANORVM Nicomedia", Path: models.NumistaLookupPathDirect,
		Evidence: models.NumistaEvidence{
			Issuer: "Honorius", Mint: "SMNT", ReverseInscription: "GLORIA ROMANORVM",
		},
		QuerySource:       models.NumistaQuerySourceGenerated,
		GenerationVersion: models.NumistaQueryGenerationVersion,
	}
	first, err := service.Lookup(context.Background(), request)
	if err != nil || first.Cache == nil || first.Cache.Hit ||
		first.SearchAttempt != models.NumistaSearchAttemptRelaxed ||
		first.SearchAttemptCount != 2 {
		t.Fatalf("fresh generated lookup=%+v err=%v", first, err)
	}
	second, err := service.Lookup(context.Background(), request)
	if err != nil || second.Cache == nil || !second.Cache.Hit ||
		second.EffectiveQuery != "Honorius Nicomedia" ||
		second.SearchAttempt != models.NumistaSearchAttemptRelaxed ||
		second.SearchAttemptCount != 2 || client.CallCount() != 2 {
		t.Fatalf("cached generated lookup=%+v calls=%v err=%v", second, client.Queries(), err)
	}
}

func TestNumistaLookupCacheCoalescesConcurrentEquivalentPaths(t *testing.T) {
	client := &phase6CacheClient{
		candidates: []models.NumistaCandidate{{ID: 9, Title: "Coalesced Coin"}},
		started:    make(chan struct{}),
		release:    make(chan struct{}),
	}
	settings := &fakeNumistaSettings{key: "configured", config: NumistaSettings{
		SearchTTL: time.Hour, SearchResultLimit: 20, Valid: true,
	}}
	service := newPhase6CacheService(client, NewNumistaCache(nil, 20, 20), settings, nil)
	requests := []models.NumistaLookupRequest{
		{Query: "Antoninus Pius", Path: models.NumistaLookupPathDirect},
		{Query: " antoninus   pius ", Path: models.NumistaLookupPathPhoto},
	}
	results := make(chan models.NumistaLookupOutcome, len(requests))
	errors := make(chan error, len(requests))
	for _, request := range requests {
		go func(request models.NumistaLookupRequest) {
			outcome, err := service.Lookup(context.Background(), request)
			results <- outcome
			errors <- err
		}(request)
	}
	<-client.started
	waitForSearchWaiters(t, service.cache, NumistaSearchCacheKey(requests[0].Query, 20), 2)
	close(client.release)

	loaders := 0
	coalesced := 0
	for range requests {
		if err := <-errors; err != nil {
			t.Fatalf("coalesced lookup error = %v", err)
		}
		outcome := <-results
		if outcome.Status != models.NumistaStatusSuccess || len(outcome.Candidates) != 1 {
			t.Fatalf("coalesced outcome = %+v", outcome)
		}
		if outcome.Cache == nil || outcome.Cache.Hit {
			t.Fatalf("cold fan-in was reported as a fresh persisted-cache hit: %+v", outcome.Cache)
		}
		if outcome.Cache.Coalesced {
			coalesced++
		} else {
			loaders++
		}
	}
	if client.calls.Load() != 1 || loaders != 1 || coalesced != 1 {
		t.Fatalf(
			"provider calls=%d loaders=%d coalesced=%d, want 1/1/1",
			client.calls.Load(), loaders, coalesced,
		)
	}
}

func TestNumistaLookupCacheAppliesLiveSettingsWithoutCredentialsMasking(t *testing.T) {
	clock := &fakeNumistaClock{now: time.Date(2026, 8, 11, 12, 0, 0, 0, time.UTC)}
	client := &phase6CacheClient{candidates: []models.NumistaCandidate{{ID: 5, Title: "Configured Coin"}}}
	settings := &fakeNumistaSettings{key: "configured", config: NumistaSettings{
		SearchTTL: time.Hour, SearchResultLimit: 20, Valid: true,
	}}
	service := newPhase6CacheService(client, NewNumistaCache(clock, 20, 20), settings, clock)
	request := models.NumistaLookupRequest{Query: "configured coin", Path: models.NumistaLookupPathDirect}

	first, err := service.Lookup(context.Background(), request)
	if err != nil || first.Cache == nil || first.Cache.Hit {
		t.Fatalf("initial lookup = %+v, error = %v", first, err)
	}
	settings.config.SearchTTL = 12 * time.Hour
	clock.Add(30 * time.Minute)
	sameIdentity, err := service.Lookup(context.Background(), request)
	if err != nil || sameIdentity.Cache == nil || !sameIdentity.Cache.Hit ||
		!sameIdentity.Cache.ExpiresAt.Equal(first.Cache.ExpiresAt) {
		t.Fatalf("TTL change rewrote an existing entry: first=%+v second=%+v error=%v", first.Cache, sameIdentity.Cache, err)
	}

	settings.config.SearchResultLimit = 21
	limitChanged, err := service.Lookup(context.Background(), request)
	if err != nil || limitChanged.Cache == nil || limitChanged.Cache.Hit {
		t.Fatalf("limit-changed lookup = %+v, error = %v", limitChanged, err)
	}
	if !limitChanged.Cache.ExpiresAt.Equal(clock.Now().Add(12 * time.Hour)) {
		t.Fatalf("new write ignored live TTL: %+v", limitChanged.Cache)
	}

	settings.key = ""
	unconfigured, err := service.Lookup(context.Background(), request)
	if err != nil || unconfigured.Status != models.NumistaStatusUnconfigured || unconfigured.Cache != nil {
		t.Fatalf("removed credential lookup = %+v, error = %v", unconfigured, err)
	}
	if client.calls.Load() != 2 {
		t.Fatalf("provider calls = %d, want no call after credential removal", client.calls.Load())
	}
}

func TestNumistaLookupCacheKeepsSearchAndDetailTTLsIndependent(t *testing.T) {
	clock := &fakeNumistaClock{now: time.Date(2026, 8, 11, 12, 0, 0, 0, time.UTC)}
	cache := NewNumistaCache(clock, 20, 20)
	searchLoads := atomic.Int32{}
	detailLoads := atomic.Int32{}
	loadSearch := func(context.Context) ([]models.NumistaCandidate, error) {
		searchLoads.Add(1)
		return []models.NumistaCandidate{{ID: 1, Title: "Broad"}}, nil
	}
	loadDetail := func(context.Context) (models.NumistaCandidate, error) {
		detailLoads.Add(1)
		return models.NumistaCandidate{ID: 1, Title: "Detailed", Material: "Silver"}, nil
	}

	if _, metadata, err := cache.DoSearch(context.Background(), "coin", 20, time.Hour, loadSearch); err != nil ||
		metadata == nil || metadata.Hit {
		t.Fatalf("initial search metadata=%+v error=%v", metadata, err)
	}
	if _, metadata, err := cache.DoDetail(context.Background(), 1, 7*24*time.Hour, loadDetail); err != nil ||
		metadata == nil || metadata.Hit {
		t.Fatalf("initial detail metadata=%+v error=%v", metadata, err)
	}

	clock.Add(2 * time.Hour)
	if _, metadata, err := cache.DoSearch(context.Background(), "coin", 20, time.Hour, loadSearch); err != nil ||
		metadata == nil || metadata.Hit {
		t.Fatalf("expired search metadata=%+v error=%v", metadata, err)
	}
	detail, metadata, err := cache.DoDetail(context.Background(), 1, 7*24*time.Hour, loadDetail)
	if err != nil || metadata == nil || !metadata.Hit || detail.EnrichmentState != models.NumistaEnrichmentCached {
		t.Fatalf("fresh detail=%+v metadata=%+v error=%v", detail, metadata, err)
	}
	if searchLoads.Load() != 2 || detailLoads.Load() != 1 {
		t.Fatalf("loads search=%d detail=%d, want 2 and 1", searchLoads.Load(), detailLoads.Load())
	}
}
