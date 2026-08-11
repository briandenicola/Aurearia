package services

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
)

type fakeNumistaClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *fakeNumistaClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *fakeNumistaClock) Add(duration time.Duration) {
	c.mu.Lock()
	c.now = c.now.Add(duration)
	c.mu.Unlock()
}

func TestNumistaCacheFreshEmptyExpiryAndHashedIdentity(t *testing.T) {
	clock := &fakeNumistaClock{now: time.Date(2026, 8, 11, 0, 0, 0, 0, time.UTC)}
	cache := NewNumistaCache(clock, 2, 2)
	key := NumistaSearchCacheKey("Trajan", 20)
	if len(key) != 64 || key == "Trajan" {
		t.Fatalf("cache key is not a SHA-256 identity: %q", key)
	}
	cache.SetSearch("Trajan", 20, []models.NumistaCandidate{}, time.Hour)
	value, metadata, ok := cache.GetSearch("trajan", 20)
	if !ok || len(value) != 0 || !metadata.Hit {
		t.Fatalf("fresh empty result not reused: %#v %#v %v", value, metadata, ok)
	}
	clock.Add(time.Hour)
	if _, _, ok := cache.GetSearch("Trajan", 20); ok {
		t.Fatal("expired entry was served")
	}
}

func TestNumistaCacheCoalescesSameKey(t *testing.T) {
	cache := NewNumistaCache(nil, 10, 10)
	var loads atomic.Int32
	start := make(chan struct{})
	release := make(chan struct{})
	load := func(context.Context) ([]models.NumistaCandidate, error) {
		if loads.Add(1) == 1 {
			close(start)
		}
		<-release
		return []models.NumistaCandidate{{ID: 1, Title: "Coin"}}, nil
	}
	var wg sync.WaitGroup
	wg.Add(2)
	for range 2 {
		go func() {
			defer wg.Done()
			_, _, _ = cache.DoSearch(context.Background(), "coin", 20, time.Hour, load)
		}()
	}
	<-start
	waitForSearchWaiters(t, cache, NumistaSearchCacheKey("coin", 20), 2)
	close(release)
	wg.Wait()
	if loads.Load() != 1 {
		t.Fatalf("loads = %d, want 1", loads.Load())
	}
}

func TestNumistaCacheDetailNamespaceTTLAndBoundedEviction(t *testing.T) {
	clock := &fakeNumistaClock{now: time.Date(2026, 8, 11, 0, 0, 0, 0, time.UTC)}
	cache := NewNumistaCache(clock, 1, 1)
	cache.SetSearch("one", 20, nil, time.Hour)
	cache.SetDetail(1, models.NumistaCandidate{ID: 1, Title: "One"}, 2*time.Hour)
	clock.Add(90 * time.Minute)
	if _, _, ok := cache.GetSearch("one", 20); ok {
		t.Fatal("search TTL did not expire independently")
	}
	if detail, _, ok := cache.GetDetail(1); !ok || detail.EnrichmentState != models.NumistaEnrichmentCached {
		t.Fatalf("detail TTL did not remain fresh: %+v ok=%v", detail, ok)
	}
	cache.SetDetail(2, models.NumistaCandidate{ID: 2, Title: "Two"}, time.Hour)
	if _, _, ok := cache.GetDetail(1); ok {
		t.Fatal("oldest detail was not evicted at the bound")
	}
}

func TestNumistaCacheCoalescesDetailAndLetsWaiterCancel(t *testing.T) {
	cache := NewNumistaCache(nil, 10, 10)
	var loads atomic.Int32
	started := make(chan struct{})
	release := make(chan struct{})
	var releaseOnce sync.Once
	releaseLeader := func() { releaseOnce.Do(func() { close(release) }) }
	t.Cleanup(releaseLeader)
	load := func(context.Context) (models.NumistaCandidate, error) {
		if loads.Add(1) == 1 {
			close(started)
		}
		<-release
		return models.NumistaCandidate{ID: 1, Title: "One"}, nil
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _, _ = cache.DoDetail(context.Background(), 1, time.Hour, load)
	}()
	<-started
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, _, err := cache.DoDetail(ctx, 1, time.Hour, load); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled waiter error = %v, want context.Canceled", err)
	}
	releaseLeader()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("detail leader goroutine did not exit")
	}
	if loads.Load() != 1 {
		t.Fatalf("detail loads = %d, want 1", loads.Load())
	}
}

func TestNumistaCacheDetailDeepMutationIsolationAcrossLoadCoalescingAndHits(t *testing.T) {
	cache := NewNumistaCache(nil, 10, 10)
	minYear, maxYear := 98, 117
	canonical := models.NumistaCandidate{
		ID: 42, Title: "Trajan Denarius", MinYear: &minYear, MaxYear: &maxYear,
		Assessment: models.NumistaRelevanceAssessment{
			ScoringVersion: models.NumistaScoringVersion,
			Score:          90,
			Band:           "strong",
			Reasons: []models.NumistaRelevanceReason{{
				Field: "title", Kind: models.NumistaReasonMatch,
				Code: "title_match", Label: "Title matches",
			}},
		},
	}
	started := make(chan struct{})
	release := make(chan struct{})
	var loads atomic.Int32
	load := func(context.Context) (models.NumistaCandidate, error) {
		loads.Add(1)
		close(started)
		<-release
		return canonical, nil
	}
	type detailResult struct {
		value    models.NumistaCandidate
		metadata *NumistaCacheResult
		err      error
	}
	results := make(chan detailResult, 2)
	for range 2 {
		go func() {
			value, metadata, err := cache.DoDetail(context.Background(), 42, time.Hour, load)
			results <- detailResult{value: value, metadata: metadata, err: err}
		}()
	}
	<-started
	waitForDetailWaiters(t, cache, NumistaDetailCacheKey(42), 2)
	close(release)

	first, second := <-results, <-results
	if first.err != nil || second.err != nil || loads.Load() != 1 {
		t.Fatalf("coalesced detail results first=%+v second=%+v loads=%d", first, second, loads.Load())
	}
	outcomes := map[NumistaCacheOutcome]int{
		first.metadata.Outcome: 1,
	}
	outcomes[second.metadata.Outcome]++
	if outcomes[NumistaCacheOutcomeLoader] != 1 ||
		outcomes[NumistaCacheOutcomeCoalescedWaiter] != 1 {
		t.Fatalf("detail ownership outcomes=%v", outcomes)
	}

	mutateDetailCandidate(&first.value)
	assertCanonicalDetailCandidate(t, second.value)

	hit, metadata, err := cache.DoDetail(context.Background(), 42, time.Hour, load)
	if err != nil || metadata == nil || metadata.Outcome != NumistaCacheOutcomeFreshHit ||
		!metadata.Hit || loads.Load() != 1 {
		t.Fatalf("first cache hit=%+v metadata=%+v err=%v loads=%d", hit, metadata, err, loads.Load())
	}
	assertCanonicalDetailCandidate(t, hit)
	if hit.EnrichmentState != models.NumistaEnrichmentCached {
		t.Fatalf("cache hit enrichment state=%q, want cached", hit.EnrichmentState)
	}

	mutateDetailCandidate(&hit)
	canonicalHit, metadata, err := cache.DoDetail(context.Background(), 42, time.Hour, load)
	if err != nil || metadata == nil || metadata.Outcome != NumistaCacheOutcomeFreshHit {
		t.Fatalf("canonical cache hit=%+v metadata=%+v err=%v", canonicalHit, metadata, err)
	}
	assertCanonicalDetailCandidate(t, canonicalHit)
	if *canonical.MinYear != 98 || *canonical.MaxYear != 117 ||
		canonical.Assessment.Reasons[0].Label != "Title matches" {
		t.Fatalf("provider-owned source was mutated: %+v", canonical)
	}
}

func mutateDetailCandidate(candidate *models.NumistaCandidate) {
	*candidate.MinYear = -999
	*candidate.MaxYear = 999
	candidate.Assessment.Reasons[0].Code = "mutated"
	candidate.Assessment.Reasons[0].Label = "Mutated"
	candidate.Assessment.Reasons = append(candidate.Assessment.Reasons, models.NumistaRelevanceReason{
		Field: "issuer", Kind: models.NumistaReasonConflict, Code: "extra", Label: "Extra",
	})
}

func assertCanonicalDetailCandidate(t *testing.T, candidate models.NumistaCandidate) {
	t.Helper()
	if candidate.MinYear == nil || *candidate.MinYear != 98 ||
		candidate.MaxYear == nil || *candidate.MaxYear != 117 ||
		len(candidate.Assessment.Reasons) != 1 ||
		candidate.Assessment.Reasons[0].Code != "title_match" ||
		candidate.Assessment.Reasons[0].Label != "Title matches" {
		t.Fatalf("detail candidate was not isolated: %+v", candidate)
	}
}

func TestNumistaCacheSearchWaiterDeadlineDoesNotCancelLeader(t *testing.T) {
	cache := NewNumistaCache(nil, 10, 10)
	var loads atomic.Int32
	started := make(chan struct{})
	release := make(chan struct{})
	var releaseOnce sync.Once
	releaseLeader := func() { releaseOnce.Do(func() { close(release) }) }
	t.Cleanup(releaseLeader)
	leaderDone := make(chan error, 1)
	load := func(context.Context) ([]models.NumistaCandidate, error) {
		if loads.Add(1) == 1 {
			close(started)
		}
		<-release
		return []models.NumistaCandidate{{ID: 1, Title: "One"}}, nil
	}
	go func() {
		_, _, err := cache.DoSearch(context.Background(), "coin", 20, time.Hour, load)
		leaderDone <- err
	}()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("search leader did not start")
	}

	waiterCtx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if _, _, err := cache.DoSearch(waiterCtx, "coin", 20, time.Hour, load); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("deadline waiter error = %v, want context.DeadlineExceeded", err)
	}
	if loads.Load() != 1 {
		t.Fatalf("waiter started another load: %d", loads.Load())
	}

	releaseLeader()
	select {
	case err := <-leaderDone:
		if err != nil {
			t.Fatalf("leader error = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("search leader goroutine did not exit")
	}
	if _, metadata, ok := cache.GetSearch("coin", 20); !ok || metadata == nil || !metadata.Hit {
		t.Fatalf("leader result was not cached: metadata=%+v ok=%v", metadata, ok)
	}
}

func TestNumistaCacheColdSearchFanInIsAtomicUnderStress(t *testing.T) {
	const iterations = 100
	const callers = 8
	for iteration := range iterations {
		cache := NewNumistaCache(nil, 10, 10)
		var loads atomic.Int32
		started := make(chan struct{})
		release := make(chan struct{})
		load := func(context.Context) ([]models.NumistaCandidate, error) {
			if loads.Add(1) == 1 {
				close(started)
			}
			<-release
			return []models.NumistaCandidate{{ID: iteration + 1, Title: "Coin"}}, nil
		}
		results := make(chan error, callers)
		startCallers := make(chan struct{})
		for range callers {
			go func() {
				<-startCallers
				_, _, err := cache.DoSearch(context.Background(), "same", 20, time.Hour, load)
				results <- err
			}()
		}
		close(startCallers)
		<-started
		waitForSearchWaiters(t, cache, NumistaSearchCacheKey("same", 20), callers)
		close(release)
		for range callers {
			if err := <-results; err != nil {
				t.Fatalf("iteration %d returned %v", iteration, err)
			}
		}
		if loads.Load() != 1 {
			t.Fatalf("iteration %d loads=%d, want 1", iteration, loads.Load())
		}
	}
}

func TestNumistaCacheCancelledLeaderDoesNotPoisonHealthyWaiters(t *testing.T) {
	cache := NewNumistaCache(nil, 10, 10)
	var loads atomic.Int32
	started := make(chan struct{})
	release := make(chan struct{})
	load := func(ctx context.Context) ([]models.NumistaCandidate, error) {
		if loads.Add(1) == 1 {
			close(started)
		}
		select {
		case <-release:
			return []models.NumistaCandidate{{ID: 7, Title: "Healthy"}}, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	leaderCtx, cancelLeader := context.WithCancel(context.Background())
	leaderDone := make(chan error, 1)
	go func() {
		_, _, err := cache.DoSearch(leaderCtx, "coin", 20, time.Hour, load)
		leaderDone <- err
	}()
	<-started

	const healthyWaiters = 4
	waiterDone := make(chan error, healthyWaiters)
	for range healthyWaiters {
		go func() {
			value, _, err := cache.DoSearch(context.Background(), "coin", 20, time.Hour, load)
			if err == nil && (len(value) != 1 || value[0].ID != 7) {
				err = errors.New("healthy waiter received wrong value")
			}
			waiterDone <- err
		}()
	}
	waitForSearchWaiters(t, cache, NumistaSearchCacheKey("coin", 20), healthyWaiters+1)
	cancelLeader()
	select {
	case err := <-leaderDone:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("leader error=%v, want context.Canceled", err)
		}
	case <-time.After(time.Second):
		t.Fatal("cancelled leader did not return promptly")
	}
	close(release)
	for range healthyWaiters {
		if err := <-waiterDone; err != nil {
			t.Fatalf("healthy waiter failed: %v", err)
		}
	}
	if loads.Load() != 1 {
		t.Fatalf("provider calls=%d, want 1", loads.Load())
	}
	if value, metadata, ok := cache.GetSearch("coin", 20); !ok || metadata == nil || !metadata.Hit ||
		len(value) != 1 || value[0].ID != 7 {
		t.Fatalf("healthy result was not published: value=%+v metadata=%+v ok=%v", value, metadata, ok)
	}
}

func TestNumistaCacheAllCallersCancelAndReplacementIsBounded(t *testing.T) {
	cache := NewNumistaCache(nil, 10, 10)
	var loads atomic.Int32
	firstCancelled := make(chan struct{})
	replacementStarted := make(chan struct{})
	providerFailure := errors.New("replacement provider failed")
	load := func(ctx context.Context) ([]models.NumistaCandidate, error) {
		switch loads.Add(1) {
		case 1:
			<-ctx.Done()
			close(firstCancelled)
			return nil, ctx.Err()
		case 2:
			close(replacementStarted)
			return nil, providerFailure
		default:
			return nil, errors.New("unexpected extra provider call")
		}
	}

	const callers = 3
	contexts := make([]context.CancelFunc, 0, callers)
	results := make(chan error, callers)
	for range callers {
		ctx, cancel := context.WithCancel(context.Background())
		contexts = append(contexts, cancel)
		go func() {
			_, _, err := cache.DoSearch(ctx, "coin", 20, time.Hour, load)
			results <- err
		}()
	}
	waitForSearchWaiters(t, cache, NumistaSearchCacheKey("coin", 20), callers)
	for _, cancel := range contexts {
		cancel()
	}
	for range callers {
		select {
		case err := <-results:
			if !errors.Is(err, context.Canceled) {
				t.Fatalf("cancelled caller error=%v", err)
			}
		case <-time.After(time.Second):
			t.Fatal("cancelled caller did not return promptly")
		}
	}
	select {
	case <-firstCancelled:
	case <-time.After(time.Second):
		t.Fatal("orphaned provider call did not receive cancellation")
	}
	if _, _, ok := cache.GetSearch("coin", 20); ok {
		t.Fatal("all-cancelled load published a cache entry")
	}

	_, _, err := cache.DoSearch(context.Background(), "coin", 20, time.Hour, load)
	if !errors.Is(err, providerFailure) {
		t.Fatalf("replacement error=%v, want provider failure", err)
	}
	select {
	case <-replacementStarted:
	default:
		t.Fatal("replacement provider call did not start")
	}
	if loads.Load() != 2 {
		t.Fatalf("provider calls=%d, want cancelled call plus one replacement", loads.Load())
	}
	if _, _, ok := cache.GetSearch("coin", 20); ok {
		t.Fatal("failed replacement was cached")
	}
}

func waitForSearchWaiters(t *testing.T, cache *NumistaCache, key string, want int) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		cache.mu.Lock()
		call := cache.searchInflight[key]
		got := 0
		if call != nil {
			got = call.waiters
		}
		cache.mu.Unlock()
		if got == want {
			return
		}
		time.Sleep(time.Millisecond)
	}
	cache.mu.Lock()
	call := cache.searchInflight[key]
	got := 0
	if call != nil {
		got = call.waiters
	}
	cache.mu.Unlock()
	t.Fatalf("search waiters=%d, want %d", got, want)
}

func waitForDetailWaiters(t *testing.T, cache *NumistaCache, key string, want int) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		cache.mu.Lock()
		call := cache.detailInflight[key]
		got := 0
		if call != nil {
			got = call.waiters
		}
		cache.mu.Unlock()
		if got == want {
			return
		}
		time.Sleep(time.Millisecond)
	}
	cache.mu.Lock()
	call := cache.detailInflight[key]
	got := 0
	if call != nil {
		got = call.waiters
	}
	cache.mu.Unlock()
	t.Fatalf("detail waiters=%d, want %d", got, want)
}
