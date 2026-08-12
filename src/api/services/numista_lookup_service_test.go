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

type fakeNumistaClient struct {
	candidates []models.NumistaCandidate
	err        error
	calls      atomic.Int32
	query      string
}

func (c *fakeNumistaClient) Search(_ context.Context, query string, _ int) ([]models.NumistaCandidate, error) {
	c.calls.Add(1)
	c.query = query
	return cloneCandidates(c.candidates), c.err
}

func (c *fakeNumistaClient) Detail(context.Context, int) (models.NumistaCandidate, error) {
	return models.NumistaCandidate{}, c.err
}

type fakeNumistaSettings struct {
	key    string
	config NumistaSettings
}

func (s *fakeNumistaSettings) GetSetting(string) string            { return s.key }
func (s *fakeNumistaSettings) GetNumistaSettings() NumistaSettings { return s.config }

func newLookupTestService(client NumistaClient, settings *fakeNumistaSettings) *NumistaLookupService {
	return NewNumistaLookupService(
		client, NewNumistaCache(nil, 20, 20), NewNumistaV1Scorer(),
		NewNumistaTelemetry(20), settings, nil,
	)
}

func TestNumistaLookupBroadMappingScoringAndEffectiveQuery(t *testing.T) {
	client := &fakeNumistaClient{candidates: []models.NumistaCandidate{
		{ID: 5, Title: "Drachm", CanonicalURL: "https://evil.example/5"},
		{ID: 2, Title: "Trajan Denarius", Issuer: "Roman Empire"},
		{ID: 0, Title: "invalid"},
		{ID: 7, Title: " "},
	}}
	settings := &fakeNumistaSettings{key: "configured", config: NumistaSettings{
		SearchTTL: time.Hour, SearchResultLimit: 20, Valid: true,
	}}
	service := newLookupTestService(client, settings)
	outcome, err := service.Lookup(context.Background(), models.NumistaLookupRequest{
		Query: "  Trajan Denarius  ", Path: models.NumistaLookupPathDirect,
		Evidence: models.NumistaEvidence{Title: "Trajan Denarius", Issuer: "Roman Empire"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if outcome.EffectiveQuery != "  Trajan Denarius  " || client.query != "  Trajan Denarius  " ||
		outcome.Status != models.NumistaStatusSuccess ||
		len(outcome.Candidates) != 2 || outcome.Candidates[0].ID != 2 {
		t.Fatalf("unexpected outcome: %+v", outcome)
	}
	if outcome.Candidates[1].CanonicalURL != "https://en.numista.com/catalogue/pieces5.html" {
		t.Fatal("candidate canonical URL was not application-owned")
	}
	if outcome.Candidates[0].Assessment.ScoringVersion != models.NumistaScoringVersion {
		t.Fatalf("candidate was not scored: %+v", outcome.Candidates[0])
	}
}

func TestNumistaLookupEmptyEvidenceCacheAndStatuses(t *testing.T) {
	client := &fakeNumistaClient{}
	settings := &fakeNumistaSettings{key: "configured", config: NumistaSettings{
		SearchTTL: time.Hour, SearchResultLimit: 20, Valid: true,
	}}
	service := newLookupTestService(client, settings)
	request := models.NumistaLookupRequest{Query: "manual terms", Path: models.NumistaLookupPathDirect}
	first, err := service.Lookup(context.Background(), request)
	if err != nil || first.Status != models.NumistaStatusEmpty || first.Cache == nil || first.Cache.Hit {
		t.Fatalf("unexpected fresh empty outcome: %+v err=%v", first, err)
	}
	second, err := service.Lookup(context.Background(), request)
	if err != nil || second.Cache == nil || !second.Cache.Hit || client.calls.Load() != 1 {
		t.Fatalf("empty result was not cached: %+v calls=%d", second, client.calls.Load())
	}
	settings.key = ""
	unconfigured, err := service.Lookup(context.Background(), request)
	if err != nil || unconfigured.Status != models.NumistaStatusUnconfigured {
		t.Fatalf("configuration did not take precedence over cache: %+v err=%v", unconfigured, err)
	}
}

func TestNumistaLookupGeneratedEmptyUsesExactlyOneRelaxedAttempt(t *testing.T) {
	client := &sequenceNumistaClient{
		results: map[string][]models.NumistaCandidate{
			"Honorius GLORIA ROMANORVM Nicomedia": {},
			"Honorius Nicomedia": {
				{ID: 208360, Title: "AE3 - Honorius", Issuer: "Roman Empire"},
			},
		},
	}

	settings := &fakeNumistaSettings{key: "configured", config: NumistaSettings{
		SearchTTL: time.Hour, SearchResultLimit: 20, Valid: true,
	}}
	service := newLookupTestService(client, settings)
	outcome, err := service.Lookup(context.Background(), models.NumistaLookupRequest{
		Query: "Honorius GLORIA ROMANORVM Nicomedia",
		Path:  models.NumistaLookupPathDirect,
		Evidence: models.NumistaEvidence{
			Issuer: "Honorius", Mint: "SMNT", ReverseInscription: "GLORIA ROMANORVM",
		},
		QuerySource:       models.NumistaQuerySourceGenerated,
		GenerationVersion: models.NumistaQueryGenerationVersion,
	})
	if err != nil {
		t.Fatal(err)
	}
	if outcome.Status != models.NumistaStatusSuccess ||
		outcome.EffectiveQuery != "Honorius Nicomedia" ||
		outcome.SearchAttempt != models.NumistaSearchAttemptRelaxed ||
		outcome.SearchAttemptCount != 2 ||
		client.CallCount() != 2 {
		t.Fatalf("unexpected generated fallback: outcome=%#v calls=%v", outcome, client.Queries())
	}
}

func TestNumistaLookupStaleGeneratedMarkerIsDowngradedWithoutRetry(t *testing.T) {
	client := &sequenceNumistaClient{results: map[string][]models.NumistaCandidate{
		"Honorius verbose stale query": {},
	}}
	settings := &fakeNumistaSettings{key: "configured", config: NumistaSettings{
		SearchTTL: time.Hour, SearchResultLimit: 20, Valid: true,
	}}
	outcome, err := newLookupTestService(client, settings).Lookup(
		context.Background(),
		models.NumistaLookupRequest{
			Query: "Honorius verbose stale query", Path: models.NumistaLookupPathDirect,
			Evidence: models.NumistaEvidence{
				Issuer: "Honorius", Mint: "SMNT", ReverseInscription: "GLORIA ROMANORVM",
			},
			QuerySource:       models.NumistaQuerySourceGenerated,
			GenerationVersion: models.NumistaQueryGenerationVersion,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if client.CallCount() != 1 ||
		outcome.QuerySource != models.NumistaQuerySourceUserEdited ||
		outcome.SearchAttempt != models.NumistaSearchAttemptPrimary ||
		outcome.SearchAttemptCount != 1 ||
		outcome.EffectiveQuery != "Honorius verbose stale query" {
		t.Fatalf("stale marker was not safely downgraded: outcome=%#v queries=%v", outcome, client.Queries())
	}
}

func TestNumistaLookupDoesNotRelaxManualEditedOrTerminalOutcomes(t *testing.T) {
	tests := []struct {
		name   string
		source models.NumistaQuerySource
		err    error
		result []models.NumistaCandidate
	}{
		{name: "manual empty", source: models.NumistaQuerySourceManual},
		{name: "user edited empty", source: models.NumistaQuerySourceUserEdited},
		{
			name: "generated success", source: models.NumistaQuerySourceGenerated,
			result: []models.NumistaCandidate{{ID: 208360, Title: "AE3 - Honorius"}},
		},
		{
			name: "generated quota", source: models.NumistaQuerySourceGenerated,
			err: &NumistaError{Kind: NumistaErrorQuotaLimited},
		},
		{
			name: "generated timeout", source: models.NumistaQuerySourceGenerated,
			err: &NumistaError{Kind: NumistaErrorTimeout},
		},
		{
			name: "generated unavailable", source: models.NumistaQuerySourceGenerated,
			err: &NumistaError{Kind: NumistaErrorUnavailable},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client := &sequenceNumistaClient{
				results: map[string][]models.NumistaCandidate{
					"  Honorius RIC IX 46  ": test.result,
				},
				err: test.err,
			}
			settings := &fakeNumistaSettings{key: "configured", config: NumistaSettings{
				SearchTTL: time.Hour, SearchResultLimit: 20, Valid: true,
			}}
			outcome, err := newLookupTestService(client, settings).Lookup(
				context.Background(),
				models.NumistaLookupRequest{
					Query: "  Honorius RIC IX 46  ", Path: models.NumistaLookupPathDirect,
					Evidence: models.NumistaEvidence{
						Issuer: "Honorius", Mint: "SMNT", ReverseInscription: "GLORIA ROMANORVM",
					},
					QuerySource: test.source,
					GenerationVersion: func() string {
						if test.source == models.NumistaQuerySourceManual {
							return ""
						}
						return models.NumistaQueryGenerationVersion
					}(),
				},
			)
			if err != nil {
				t.Fatal(err)
			}
			if client.CallCount() != 1 || client.Queries()[0] != "  Honorius RIC IX 46  " {
				t.Fatalf("query was rewritten or retried: %v", client.Queries())
			}
			if outcome.EffectiveQuery != "  Honorius RIC IX 46  " ||
				outcome.SearchAttemptCount != 1 {
				t.Fatalf("unexpected exact-query outcome: %#v", outcome)
			}
		})
	}
}

func TestNumistaLookupGeneratedUnconfiguredHasNoProviderRetry(t *testing.T) {
	client := &sequenceNumistaClient{}
	outcome, err := newLookupTestService(client, &fakeNumistaSettings{}).Lookup(
		context.Background(),
		models.NumistaLookupRequest{
			Query: "Honorius GLORIA ROMANORVM Nicomedia",
			Path:  models.NumistaLookupPathDirect,
			Evidence: models.NumistaEvidence{
				Issuer: "Honorius", Mint: "SMNT", ReverseInscription: "GLORIA ROMANORVM",
			},
			QuerySource:       models.NumistaQuerySourceGenerated,
			GenerationVersion: models.NumistaQueryGenerationVersion,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if outcome.Status != models.NumistaStatusUnconfigured ||
		outcome.SearchAttempt != models.NumistaSearchAttemptPrimary ||
		outcome.SearchAttemptCount != 1 ||
		client.CallCount() != 0 {
		t.Fatalf("unconfigured generated lookup retried: outcome=%#v calls=%v", outcome, client.Queries())
	}
}

func TestNumistaLookupGeneratedCancellationHasOneProviderCall(t *testing.T) {
	client := &sequenceNumistaClient{err: context.Canceled}
	settings := &fakeNumistaSettings{key: "configured", config: NumistaSettings{
		SearchTTL: time.Hour, SearchResultLimit: 20, Valid: true,
	}}
	_, err := newLookupTestService(client, settings).Lookup(
		context.Background(),
		models.NumistaLookupRequest{
			Query: "Honorius GLORIA ROMANORVM Nicomedia",
			Path:  models.NumistaLookupPathDirect,
			Evidence: models.NumistaEvidence{
				Issuer: "Honorius", Mint: "SMNT", ReverseInscription: "GLORIA ROMANORVM",
			},
			QuerySource:       models.NumistaQuerySourceGenerated,
			GenerationVersion: models.NumistaQueryGenerationVersion,
		},
	)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
	if client.CallCount() != 1 {
		t.Fatalf("provider calls = %d, want exactly 1", client.CallCount())
	}
}

type sequenceNumistaClient struct {
	results map[string][]models.NumistaCandidate
	err     error
	queries []string
}

func (c *sequenceNumistaClient) Search(_ context.Context, query string, _ int) ([]models.NumistaCandidate, error) {
	c.queries = append(c.queries, query)
	return cloneCandidates(c.results[query]), c.err
}

func (c *sequenceNumistaClient) Detail(context.Context, int) (models.NumistaCandidate, error) {
	return models.NumistaCandidate{}, c.err
}

func (c *sequenceNumistaClient) CallCount() int { return len(c.queries) }

func (c *sequenceNumistaClient) Queries() []string {
	return append([]string(nil), c.queries...)
}

func TestNumistaLookupMapsSafeQuotaOutcome(t *testing.T) {
	retry := 30
	client := &fakeNumistaClient{err: &NumistaError{Kind: NumistaErrorQuotaLimited, RetryAfterSeconds: &retry}}
	settings := &fakeNumistaSettings{key: "configured", config: NumistaSettings{SearchTTL: time.Hour, SearchResultLimit: 20}}
	outcome, err := newLookupTestService(client, settings).Lookup(context.Background(), models.NumistaLookupRequest{
		Query: "coin", Path: models.NumistaLookupPathDirect,
	})
	if err != nil || outcome.Status != models.NumistaStatusQuotaLimited ||
		outcome.RetryAfterSeconds == nil || *outcome.RetryAfterSeconds != 30 || len(outcome.Candidates) != 0 {
		t.Fatalf("unexpected quota outcome: %+v err=%v", outcome, err)
	}
}

type blockingNumistaClient struct {
	started chan struct{}
	release chan struct{}
	calls   atomic.Int32
}

func (c *blockingNumistaClient) Search(ctx context.Context, _ string, _ int) ([]models.NumistaCandidate, error) {
	if c.calls.Add(1) == 1 {
		close(c.started)
	}
	select {
	case <-c.release:
		return []models.NumistaCandidate{{ID: 1, Title: "Coin"}}, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (c *blockingNumistaClient) Detail(context.Context, int) (models.NumistaCandidate, error) {
	return models.NumistaCandidate{}, errors.New("not implemented")
}

func TestNumistaLookupCoalescedWaitersPreserveCancellationSemantics(t *testing.T) {
	client := &blockingNumistaClient{started: make(chan struct{}), release: make(chan struct{})}
	var releaseOnce sync.Once
	releaseLeader := func() { releaseOnce.Do(func() { close(client.release) }) }
	t.Cleanup(releaseLeader)
	settings := &fakeNumistaSettings{key: "configured", config: NumistaSettings{
		SearchTTL: time.Hour, SearchResultLimit: 20, Valid: true,
	}}
	service := newLookupTestService(client, settings)
	request := models.NumistaLookupRequest{Query: "coin", Path: models.NumistaLookupPathDirect}

	leaderDone := make(chan error, 1)
	go func() {
		outcome, err := service.Lookup(context.Background(), request)
		if err == nil && outcome.Status != models.NumistaStatusSuccess {
			err = errors.New("leader did not return success")
		}
		leaderDone <- err
	}()
	select {
	case <-client.started:
	case <-time.After(time.Second):
		t.Fatal("leader lookup did not start")
	}

	cancelledCtx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := service.Lookup(cancelledCtx, request); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled waiter error = %v, want context.Canceled", err)
	}

	deadlineCtx, deadlineCancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer deadlineCancel()
	if _, err := service.Lookup(deadlineCtx, request); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("deadline waiter error=%v, want context.DeadlineExceeded", err)
	}
	if client.calls.Load() != 1 {
		t.Fatalf("coalesced waiters made %d provider calls, want 1", client.calls.Load())
	}

	releaseLeader()
	select {
	case err := <-leaderDone:
		if err != nil {
			t.Fatalf("leader error = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("leader lookup goroutine did not exit")
	}
}

func TestNumistaLookupCancelledFirstCallerDoesNotPoisonHealthyWaiter(t *testing.T) {
	client := &blockingNumistaClient{started: make(chan struct{}), release: make(chan struct{})}
	settings := &fakeNumistaSettings{key: "configured", config: NumistaSettings{
		SearchTTL: time.Hour, SearchResultLimit: 20, Valid: true,
	}}
	service := newLookupTestService(client, settings)
	request := models.NumistaLookupRequest{Query: "coin", Path: models.NumistaLookupPathDirect}

	leaderCtx, cancelLeader := context.WithCancel(context.Background())
	leaderDone := make(chan error, 1)
	go func() {
		_, err := service.Lookup(leaderCtx, request)
		leaderDone <- err
	}()
	<-client.started

	waiterDone := make(chan struct {
		outcome models.NumistaLookupOutcome
		err     error
	}, 1)
	go func() {
		outcome, err := service.Lookup(context.Background(), request)
		waiterDone <- struct {
			outcome models.NumistaLookupOutcome
			err     error
		}{outcome: outcome, err: err}
	}()
	waitForSearchWaiters(t, service.cache, NumistaSearchCacheKey("coin", 20), 2)
	cancelLeader()
	select {
	case err := <-leaderDone:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("cancelled first caller error=%v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("cancelled first caller did not return promptly")
	}

	close(client.release)
	select {
	case result := <-waiterDone:
		if result.err != nil || result.outcome.Status != models.NumistaStatusSuccess {
			t.Fatalf("healthy waiter outcome=%+v err=%v", result.outcome, result.err)
		}
	case <-time.After(time.Second):
		t.Fatal("healthy waiter did not complete")
	}
	if client.calls.Load() != 1 {
		t.Fatalf("provider calls=%d, want 1", client.calls.Load())
	}
}
