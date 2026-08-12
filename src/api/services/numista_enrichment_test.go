package services

import (
	"context"
	"errors"
	"reflect"
	"runtime"
	"sort"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
)

type numistaEnrichmentCapability interface {
	Enrich(context.Context, models.NumistaEnrichmentRequest) (models.NumistaLookupOutcome, error)
}

func requireNumistaEnrichment(t *testing.T, service *NumistaLookupService) numistaEnrichmentCapability {
	t.Helper()
	enrichment, ok := any(service).(numistaEnrichmentCapability)
	if !ok {
		t.Fatal("Phase 7 missing: NumistaLookupService.Enrich is not implemented")
	}
	return enrichment
}

func enrichmentAssessment(score int) models.NumistaRelevanceAssessment {
	band := "weak"
	if score >= 80 {
		band = "strong"
	} else if score >= 60 {
		band = "possible"
	}
	return models.NumistaRelevanceAssessment{
		ScoringVersion: models.NumistaScoringVersion,
		Score:          score,
		Band:           band,
		Reasons:        []models.NumistaRelevanceReason{},
	}
}

func enrichmentCandidate(id int, title string, position int) models.NumistaCandidate {
	canonical, _ := models.CanonicalNumistaURL(id)
	return models.NumistaCandidate{
		ID: id, CanonicalURL: canonical, Title: title, ProviderPosition: position,
		EnrichmentState: models.NumistaEnrichmentNotRequested,
		Assessment:      enrichmentAssessment(50),
	}
}

func enrichmentRequest(candidates []models.NumistaCandidate) models.NumistaEnrichmentRequest {
	return models.NumistaEnrichmentRequest{
		NumistaLookupRequest: models.NumistaLookupRequest{
			Query: "Trajan denarius Rome silver",
			Path:  models.NumistaLookupPathDirect,
			Evidence: models.NumistaEvidence{
				Title: "Trajan denarius", Mint: "Rome", Material: "Silver",
			},
		},
		Candidates: candidates,
	}
}

type enrichmentDetailClient struct {
	mu             sync.Mutex
	details        map[int]models.NumistaCandidate
	failures       map[int]error
	calls          []int
	active         atomic.Int32
	maxActive      atomic.Int32
	started        chan int
	release        chan struct{}
	cancelObserved chan int
}

func (c *enrichmentDetailClient) Search(context.Context, string, int) ([]models.NumistaCandidate, error) {
	return nil, errors.New("broad search must not run during enrichment")
}

func (c *enrichmentDetailClient) Detail(ctx context.Context, id int) (models.NumistaCandidate, error) {
	c.mu.Lock()
	c.calls = append(c.calls, id)
	detail := c.details[id]
	err := c.failures[id]
	c.mu.Unlock()

	active := c.active.Add(1)
	for {
		current := c.maxActive.Load()
		if active <= current || c.maxActive.CompareAndSwap(current, active) {
			break
		}
	}
	defer c.active.Add(-1)
	if c.started != nil {
		c.started <- id
	}
	if c.release != nil {
		select {
		case <-c.release:
		case <-ctx.Done():
			if c.cancelObserved != nil {
				c.cancelObserved <- id
			}
			return models.NumistaCandidate{}, ctx.Err()
		}
	}
	if err != nil {
		return models.NumistaCandidate{}, err
	}
	return detail, nil
}

func (c *enrichmentDetailClient) calledIDs() []int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]int(nil), c.calls...)
}

func newEnrichmentService(
	client NumistaClient,
	cache *NumistaCache,
	limit int,
	telemetry *NumistaTelemetry,
) *NumistaLookupService {
	if cache == nil {
		cache = NewNumistaCache(nil, 20, 50)
	}
	if telemetry == nil {
		telemetry = NewNumistaTelemetry(50)
	}
	if limit == 0 {
		limit = 5
	}
	return NewNumistaLookupService(
		client,
		cache,
		NewNumistaV1Scorer(),
		telemetry,
		&fakeNumistaSettings{key: "configured", config: NumistaSettings{
			DetailTTL: 7 * 24 * time.Hour, EnrichmentLimit: limit, Valid: true,
		}},
		nil,
	)
}

func TestNumistaEnrichmentValidatesUniqueCandidateBoundsBeforeDetails(t *testing.T) {
	client := &enrichmentDetailClient{}
	enrichment := requireNumistaEnrichment(t, newEnrichmentService(client, nil, 5, nil))

	tests := []struct {
		name       string
		candidates []models.NumistaCandidate
	}{
		{name: "empty", candidates: []models.NumistaCandidate{}},
		{name: "more than fifty", candidates: make([]models.NumistaCandidate, 51)},
		{name: "duplicate IDs", candidates: []models.NumistaCandidate{
			enrichmentCandidate(1, "First", 0), enrichmentCandidate(1, "Duplicate", 1),
		}},
		{name: "invalid ID", candidates: []models.NumistaCandidate{
			{ID: 0, Title: "Invalid", EnrichmentState: models.NumistaEnrichmentNotRequested, Assessment: enrichmentAssessment(50)},
		}},
	}
	for index := range tests[1].candidates {
		tests[1].candidates[index] = enrichmentCandidate(index+1, "Candidate", index)
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := enrichment.Enrich(context.Background(), enrichmentRequest(test.candidates))
			if err == nil {
				t.Fatal("invalid enrichment request was accepted")
			}
		})
	}
	if calls := client.calledIDs(); len(calls) != 0 {
		t.Fatalf("invalid requests reached provider details: %v", calls)
	}

	for _, count := range []int{1, 50} {
		candidates := make([]models.NumistaCandidate, count)
		for index := range candidates {
			candidates[index] = enrichmentCandidate(index+1, "Candidate", index)
		}
		request := enrichmentRequest(candidates)
		if err := request.Validate(); err != nil {
			t.Fatalf("%d candidates failed model validation: %v", count, err)
		}
	}
}

func TestNumistaEnrichmentReranksServerSideBeforeChoosingBoundedSubset(t *testing.T) {
	candidates := []models.NumistaCandidate{
		enrichmentCandidate(1, "Unrelated bronze", 0),
		enrichmentCandidate(2, "Trajan denarius", 1),
		enrichmentCandidate(3, "Another coin", 2),
	}
	candidates[0].Assessment = enrichmentAssessment(100)
	candidates[1].Assessment = enrichmentAssessment(0)
	client := &enrichmentDetailClient{details: map[int]models.NumistaCandidate{
		2: {ID: 2, Title: "Trajan denarius", Mint: "Rome", Material: "Silver"},
	}}
	enrichment := requireNumistaEnrichment(t, newEnrichmentService(client, nil, 1, nil))

	outcome, err := enrichment.Enrich(context.Background(), enrichmentRequest(candidates))
	if err != nil {
		t.Fatal(err)
	}
	if calls := client.calledIDs(); !reflect.DeepEqual(calls, []int{2}) {
		t.Fatalf("details=%v, want server-ranked candidate 2 rather than client-first candidate 1", calls)
	}
	if outcome.Stage != "enriched" || len(outcome.Candidates) != len(candidates) ||
		outcome.Candidates[0].ID != 2 ||
		outcome.Candidates[0].EnrichmentState != models.NumistaEnrichmentEnriched {
		t.Fatalf("unexpected reranked outcome: %+v", outcome)
	}
}

func TestNumistaEnrichmentTargetSelectionIsPermutationIndependentAndNumeric(t *testing.T) {
	base := []models.NumistaCandidate{
		enrichmentCandidate(10, "Same", 0),
		enrichmentCandidate(2, "Same", 2),
		enrichmentCandidate(30, "Same", 1),
	}
	permutations := [][]int{
		{0, 1, 2}, {0, 2, 1}, {1, 0, 2},
		{1, 2, 0}, {2, 0, 1}, {2, 1, 0},
	}
	for _, permutation := range permutations {
		candidates := make([]models.NumistaCandidate, len(permutation))
		details := make(map[int]models.NumistaCandidate, len(permutation))
		for index, source := range permutation {
			candidates[index] = base[source]
			details[base[source].ID] = models.NumistaCandidate{ID: base[source].ID, Title: "Same"}
		}
		client := &enrichmentDetailClient{details: details}
		enrichment := requireNumistaEnrichment(t, newEnrichmentService(client, nil, 2, nil))
		outcome, err := enrichment.Enrich(context.Background(), enrichmentRequest(candidates))
		if err != nil {
			t.Fatal(err)
		}
		called := client.calledIDs()
		sort.Ints(called)
		if !reflect.DeepEqual(called, []int{2, 10}) {
			t.Fatalf("permutation %v enriched IDs %v, want numeric leading subset", permutation, called)
		}
		got := []int{outcome.Candidates[0].ID, outcome.Candidates[1].ID, outcome.Candidates[2].ID}
		if !reflect.DeepEqual(got, []int{2, 10, 30}) {
			t.Fatalf("permutation %v reranked %v, want numeric ID order", permutation, got)
		}
	}
}

func TestNumistaEnrichmentUsesDefaultAndConfiguredCaps(t *testing.T) {
	for _, test := range []struct {
		name       string
		configured int
		want       int
	}{
		{name: "default", configured: 0, want: 5},
		{name: "configured", configured: 3, want: 3},
	} {
		t.Run(test.name, func(t *testing.T) {
			candidates := make([]models.NumistaCandidate, 8)
			details := make(map[int]models.NumistaCandidate, len(candidates))
			for index := range candidates {
				id := index + 1
				candidates[index] = enrichmentCandidate(id, "Trajan denarius", index)
				details[id] = models.NumistaCandidate{ID: id, Title: "Trajan denarius"}
			}
			client := &enrichmentDetailClient{details: details}
			enrichment := requireNumistaEnrichment(t, newEnrichmentService(client, nil, test.configured, nil))
			outcome, err := enrichment.Enrich(context.Background(), enrichmentRequest(candidates))
			if err != nil {
				t.Fatal(err)
			}
			if got := len(client.calledIDs()); got != test.want {
				t.Fatalf("detail calls=%d, want cap %d", got, test.want)
			}
			if len(outcome.Candidates) != len(candidates) {
				t.Fatalf("returned candidates=%d, want full broad set %d", len(outcome.Candidates), len(candidates))
			}
		})
	}
}

func TestNumistaEnrichmentLimitsDetailConcurrencyToTwo(t *testing.T) {
	const count = 5
	client := &enrichmentDetailClient{
		details: make(map[int]models.NumistaCandidate, count),
		started: make(chan int, count),
		release: make(chan struct{}),
	}
	candidates := make([]models.NumistaCandidate, count)
	for index := range candidates {
		id := index + 1
		candidates[index] = enrichmentCandidate(id, "Trajan denarius", index)
		client.details[id] = models.NumistaCandidate{ID: id, Title: "Trajan denarius"}
	}
	enrichment := requireNumistaEnrichment(t, newEnrichmentService(client, nil, count, nil))
	result := make(chan error, 1)
	go func() {
		_, err := enrichment.Enrich(context.Background(), enrichmentRequest(candidates))
		result <- err
	}()

	<-client.started
	<-client.started
	for range 100 {
		runtime.Gosched()
	}
	if active := client.active.Load(); active != 2 {
		t.Fatalf("active details=%d, want exactly two while barrier is closed", active)
	}
	if maximum := client.maxActive.Load(); maximum > 2 {
		t.Fatalf("maximum detail concurrency=%d, want at most two", maximum)
	}
	close(client.release)
	if err := <-result; err != nil {
		t.Fatal(err)
	}
	if maximum := client.maxActive.Load(); maximum != 2 {
		t.Fatalf("maximum detail concurrency=%d, want two", maximum)
	}
}

func TestNumistaEnrichmentUsesCachedDetailsWithoutProviderCalls(t *testing.T) {
	clock := &fakeNumistaClock{now: time.Date(2026, 8, 11, 12, 0, 0, 0, time.UTC)}
	cache := NewNumistaCache(clock, 10, 10)
	cache.SetDetail(1, models.NumistaCandidate{
		ID: 1, Title: "Trajan denarius", Mint: "Rome", Material: "Silver",
		EnrichmentState: models.NumistaEnrichmentEnriched,
	}, time.Hour)
	client := &enrichmentDetailClient{details: map[int]models.NumistaCandidate{
		2: {ID: 2, Title: "Trajan denarius"},
	}}
	candidates := []models.NumistaCandidate{
		enrichmentCandidate(2, "Trajan denarius", 0),
		enrichmentCandidate(1, "Trajan denarius", 1),
	}
	enrichment := requireNumistaEnrichment(t, newEnrichmentService(client, cache, 1, nil))
	outcome, err := enrichment.Enrich(context.Background(), enrichmentRequest(candidates))
	if err != nil {
		t.Fatal(err)
	}
	if len(client.calledIDs()) != 0 {
		t.Fatalf("cached leading detail still called provider: %v", client.calledIDs())
	}
	var cached bool
	for _, candidate := range outcome.Candidates {
		if candidate.ID == 1 {
			cached = candidate.EnrichmentState == models.NumistaEnrichmentCached
		}
	}
	if !cached {
		t.Fatalf("cached detail was not labeled cached: %+v", outcome.Candidates)
	}
}

func TestNumistaEnrichmentCancellationStopsWorkersAndReturnsCancellation(t *testing.T) {
	client := &enrichmentDetailClient{
		details:        map[int]models.NumistaCandidate{},
		started:        make(chan int, 4),
		release:        make(chan struct{}),
		cancelObserved: make(chan int, 4),
	}
	candidates := make([]models.NumistaCandidate, 4)
	for index := range candidates {
		candidates[index] = enrichmentCandidate(index+1, "Trajan denarius", index)
	}
	enrichment := requireNumistaEnrichment(t, newEnrichmentService(client, nil, 4, nil))
	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() {
		_, err := enrichment.Enrich(ctx, enrichmentRequest(candidates))
		result <- err
	}()
	<-client.started
	<-client.started
	cancel()
	<-client.cancelObserved
	<-client.cancelObserved
	err := <-result
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("enrichment error=%#v, want caller cancellation", err)
	}
	if len(client.calledIDs()) != 2 {
		t.Fatalf("cancellation started additional details: %v", client.calledIDs())
	}
}

func TestNumistaEnrichmentIsDeterministicRetainsAllFailuresAndDoesNotMutateInput(t *testing.T) {
	original := []models.NumistaCandidate{
		enrichmentCandidate(9, "Same", 0),
		enrichmentCandidate(3, "Same", 1),
		enrichmentCandidate(7, "Other", 2),
		enrichmentCandidate(5, "Last", 3),
	}
	for _, test := range []struct {
		name     string
		failures map[int]error
	}{
		{name: "partial failure", failures: map[int]error{9: &NumistaError{Kind: NumistaErrorUnavailable}}},
		{name: "all failures", failures: map[int]error{
			9: &NumistaError{Kind: NumistaErrorUnavailable},
			3: &NumistaError{Kind: NumistaErrorTimeout},
			7: &NumistaError{Kind: NumistaErrorQuotaLimited},
			5: errors.New("private provider failure"),
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			input := append([]models.NumistaCandidate(nil), original...)
			before := append([]models.NumistaCandidate(nil), input...)
			client := &enrichmentDetailClient{
				details: map[int]models.NumistaCandidate{
					9: {ID: 9, Title: "Same"}, 3: {ID: 3, Title: "Same"},
					7: {ID: 7, Title: "Other"}, 5: {ID: 5, Title: "Last"},
				},
				failures: test.failures,
			}
			enrichment := requireNumistaEnrichment(t, newEnrichmentService(client, nil, 4, nil))
			first, err := enrichment.Enrich(context.Background(), enrichmentRequest(input))
			if err != nil {
				t.Fatalf("detail failures must degrade safely, got %v", err)
			}
			if !reflect.DeepEqual(input, before) {
				t.Fatalf("enrichment mutated submitted candidates:\nbefore=%+v\nafter=%+v", before, input)
			}
			if first.Status != models.NumistaStatusSuccess || first.Stage != "enriched" ||
				len(first.Candidates) != len(original) {
				t.Fatalf("broad candidate set was not retained: %+v", first)
			}
			ids := make([]int, len(first.Candidates))
			for index, candidate := range first.Candidates {
				ids[index] = candidate.ID
				if test.failures[candidate.ID] != nil && candidate.EnrichmentState != models.NumistaEnrichmentFailed {
					t.Fatalf("failed candidate %d state=%q", candidate.ID, candidate.EnrichmentState)
				}
			}

			secondClient := &enrichmentDetailClient{details: client.details, failures: test.failures}
			secondService := requireNumistaEnrichment(t, newEnrichmentService(secondClient, nil, 4, nil))
			second, err := secondService.Enrich(context.Background(), enrichmentRequest(original))
			if err != nil {
				t.Fatal(err)
			}
			secondIDs := make([]int, len(second.Candidates))
			for index := range second.Candidates {
				secondIDs[index] = second.Candidates[index].ID
			}
			if !reflect.DeepEqual(ids, secondIDs) {
				t.Fatalf("rerank was nondeterministic: %v vs %v", ids, secondIDs)
			}
			sorted := append([]int(nil), ids...)
			sort.Ints(sorted)
			if !reflect.DeepEqual(sorted, []int{3, 5, 7, 9}) {
				t.Fatalf("candidate IDs changed: %v", ids)
			}
		})
	}
}

func TestNumistaEnrichmentPreservesBroadSearchAttribution(t *testing.T) {
	evidence := models.NumistaEvidence{
		Issuer: "Honorius", Mint: "SMNT", ReverseInscription: "GLORIA ROMANORVM",
	}
	candidates := []models.NumistaCandidate{
		enrichmentCandidate(1, "First", 0),
		enrichmentCandidate(2, "Second", 1),
	}
	tests := []struct {
		name         string
		query        string
		failures     map[int]error
		wantAttempt  models.NumistaSearchAttempt
		wantCount    int
		wantFailures int
	}{
		{
			name: "relaxed enrichment succeeds", query: "Honorius Nicomedia",
			wantAttempt: models.NumistaSearchAttemptRelaxed, wantCount: 2,
		},
		{
			name: "relaxed enrichment partially fails", query: "Honorius Nicomedia",
			failures:    map[int]error{2: &NumistaError{Kind: NumistaErrorUnavailable}},
			wantAttempt: models.NumistaSearchAttemptRelaxed, wantCount: 2, wantFailures: 1,
		},
		{
			name: "relaxed enrichment entirely fails", query: "Honorius Nicomedia",
			failures: map[int]error{
				1: &NumistaError{Kind: NumistaErrorUnavailable},
				2: &NumistaError{Kind: NumistaErrorTimeout},
			},
			wantAttempt: models.NumistaSearchAttemptRelaxed, wantCount: 2, wantFailures: 2,
		},
		{
			name:        "primary enrichment succeeds",
			query:       "Honorius GLORIA ROMANORVM Nicomedia",
			wantAttempt: models.NumistaSearchAttemptPrimary, wantCount: 1,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client := &enrichmentDetailClient{
				details: map[int]models.NumistaCandidate{
					1: {ID: 1, Title: "First"},
					2: {ID: 2, Title: "Second"},
				},
				failures: test.failures,
			}
			request := models.NumistaEnrichmentRequest{
				NumistaLookupRequest: models.NumistaLookupRequest{
					Query: test.query, Path: models.NumistaLookupPathDirect, Evidence: evidence,
					QuerySource:       models.NumistaQuerySourceGenerated,
					GenerationVersion: models.NumistaQueryGenerationVersion,
				},
				Candidates: candidates,
			}
			outcome, err := newEnrichmentService(client, nil, 2, nil).Enrich(
				context.Background(), request,
			)
			if err != nil {
				t.Fatal(err)
			}
			if outcome.EffectiveQuery != test.query ||
				outcome.QuerySource != models.NumistaQuerySourceGenerated ||
				outcome.SearchAttempt != test.wantAttempt ||
				outcome.SearchAttemptCount != test.wantCount {
				t.Fatalf("enrichment attribution = %#v", outcome)
			}
			failures := 0
			for _, candidate := range outcome.Candidates {
				if candidate.EnrichmentState == models.NumistaEnrichmentFailed {
					failures++
				}
			}
			if failures != test.wantFailures {
				t.Fatalf("failed candidates = %d, want %d", failures, test.wantFailures)
			}
		})
	}
}

func TestNumistaEnrichmentRecordsActualDetailTelemetryWithoutPrivateText(t *testing.T) {
	telemetry := NewNumistaTelemetry(20)
	const privateText = "PRIVATE INSCRIPTION SHOULD NOT APPEAR"
	client := &enrichmentDetailClient{
		details: map[int]models.NumistaCandidate{
			1: {ID: 1, Title: "Coin", ObverseInscription: privateText},
		},
		failures: map[int]error{2: errors.New(privateText)},
	}
	candidates := []models.NumistaCandidate{
		enrichmentCandidate(1, "Coin", 0),
		enrichmentCandidate(2, "Coin", 1),
	}
	enrichment := requireNumistaEnrichment(t, newEnrichmentService(client, nil, 2, telemetry))
	if _, err := enrichment.Enrich(context.Background(), enrichmentRequest(candidates)); err != nil {
		t.Fatal(err)
	}
	health := telemetry.Health(true, true)
	if health.DetailRequestCount != 2 || health.EnrichmentAttempted != 2 ||
		health.EnrichmentSucceeded != 1 || health.EnrichmentFailed != 1 {
		t.Fatalf("enrichment telemetry=%+v", health)
	}
	for _, event := range telemetry.events {
		if event.CorrelationDigest == privateText || len(event.CorrelationDigest) != 16 {
			t.Fatalf("telemetry retained private detail text: %+v", event)
		}
	}
}
