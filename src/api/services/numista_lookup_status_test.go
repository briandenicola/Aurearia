package services

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
)

type statusNumistaClient struct {
	candidates []models.NumistaCandidate
	err        error
	calls      atomic.Int32
	wait       bool
}

func (c *statusNumistaClient) Search(ctx context.Context, _ string, _ int) ([]models.NumistaCandidate, error) {
	c.calls.Add(1)
	if c.wait {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	return cloneCandidates(c.candidates), c.err
}

func (c *statusNumistaClient) Detail(context.Context, int) (models.NumistaCandidate, error) {
	return models.NumistaCandidate{}, c.err
}

func newStatusLookupService(
	client NumistaClient,
	settings *fakeNumistaSettings,
	telemetry *NumistaTelemetry,
) *NumistaLookupService {
	return NewNumistaLookupService(
		client,
		NewNumistaCache(nil, 20, 20),
		NewNumistaV1Scorer(),
		telemetry,
		settings,
		nil,
	)
}

func statusLookupRequest() models.NumistaLookupRequest {
	return models.NumistaLookupRequest{
		Query: "Trajan denarius",
		Path:  models.NumistaLookupPathDirect,
	}
}

func TestNumistaLookupMapsAllDomainStatusesAndGuidance(t *testing.T) {
	retryAfter := 45
	tests := []struct {
		name         string
		key          string
		client       *statusNumistaClient
		wantStatus   models.NumistaLookupStatus
		wantGuidance string
		wantRetry    *int
	}{
		{
			name: "success", key: "configured",
			client:     &statusNumistaClient{candidates: []models.NumistaCandidate{{ID: 1, Title: "Trajan Denarius"}}},
			wantStatus: models.NumistaStatusSuccess,
		},
		{
			name: "empty", key: "configured", client: &statusNumistaClient{},
			wantStatus: models.NumistaStatusEmpty, wantGuidance: "revise_numista_query",
		},
		{
			name: "unconfigured", client: &statusNumistaClient{},
			wantStatus: models.NumistaStatusUnconfigured, wantGuidance: "numista_configuration_required",
		},
		{
			name: "quota limited", key: "configured",
			client: &statusNumistaClient{err: &NumistaError{
				Kind: NumistaErrorQuotaLimited, RetryAfterSeconds: &retryAfter,
			}},
			wantStatus: models.NumistaStatusQuotaLimited, wantGuidance: "numista_quota_limited",
			wantRetry: &retryAfter,
		},
		{
			name: "timeout", key: "configured",
			client:     &statusNumistaClient{err: &NumistaError{Kind: NumistaErrorTimeout}},
			wantStatus: models.NumistaStatusTimeout, wantGuidance: "retry_numista_lookup",
		},
		{
			name: "unavailable", key: "configured",
			client:     &statusNumistaClient{err: &NumistaError{Kind: NumistaErrorMalformedResponse}},
			wantStatus: models.NumistaStatusUnavailable, wantGuidance: "retry_numista_lookup",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			settings := &fakeNumistaSettings{
				key: test.key,
				config: NumistaSettings{
					SearchTTL: time.Hour, SearchResultLimit: 20, Valid: true,
				},
			}
			outcome, err := newStatusLookupService(test.client, settings, NewNumistaTelemetry(20)).
				Lookup(context.Background(), statusLookupRequest())
			if err != nil {
				t.Fatalf("Lookup() error = %v", err)
			}
			if outcome.Status != test.wantStatus || outcome.GuidanceCode != test.wantGuidance {
				t.Fatalf("outcome = %+v, want status %q guidance %q", outcome, test.wantStatus, test.wantGuidance)
			}
			if test.wantRetry == nil {
				if outcome.RetryAfterSeconds != nil {
					t.Fatalf("retryAfterSeconds = %v, want omitted", *outcome.RetryAfterSeconds)
				}
			} else if outcome.RetryAfterSeconds == nil || *outcome.RetryAfterSeconds != *test.wantRetry {
				t.Fatalf("retryAfterSeconds = %v, want %d", outcome.RetryAfterSeconds, *test.wantRetry)
			}
			if outcome.Candidates == nil {
				t.Fatal("candidates must always be present")
			}
		})
	}
}

func TestNumistaLookupMapsTypedProviderErrorsToDomainOutcomes(t *testing.T) {
	tests := []struct {
		kind         NumistaErrorKind
		wantStatus   models.NumistaLookupStatus
		wantGuidance string
	}{
		{
			kind: NumistaErrorInvalidRequest, wantStatus: models.NumistaStatusEmpty,
			wantGuidance: "revise_numista_query",
		},
		{
			kind: NumistaErrorUnauthorized, wantStatus: models.NumistaStatusUnconfigured,
			wantGuidance: "numista_configuration_required",
		},
		{
			kind: NumistaErrorUnavailable, wantStatus: models.NumistaStatusUnavailable,
			wantGuidance: "retry_numista_lookup",
		},
		{
			kind: NumistaErrorMalformedResponse, wantStatus: models.NumistaStatusUnavailable,
			wantGuidance: "retry_numista_lookup",
		},
	}
	for _, test := range tests {
		t.Run(string(test.kind), func(t *testing.T) {
			client := &statusNumistaClient{err: &NumistaError{Kind: test.kind}}
			settings := &fakeNumistaSettings{
				key:    "configured",
				config: NumistaSettings{SearchTTL: time.Hour, SearchResultLimit: 20, Valid: true},
			}
			outcome, err := newStatusLookupService(client, settings, NewNumistaTelemetry(20)).
				Lookup(context.Background(), statusLookupRequest())
			if err != nil || outcome.Status != test.wantStatus || outcome.GuidanceCode != test.wantGuidance {
				t.Fatalf(
					"kind %q outcome=%+v err=%v, want status %q guidance %q",
					test.kind, outcome, err, test.wantStatus, test.wantGuidance,
				)
			}
			if len(outcome.Candidates) != 0 || outcome.Cache != nil || outcome.RetryAfterSeconds != nil {
				t.Fatalf("kind %q exposed unexpected result metadata: %+v", test.kind, outcome)
			}
		})
	}
}

func TestNumistaLookupReturnsUnexpectedFailuresToHandler(t *testing.T) {
	internal := errors.New("database DSN and provider-secret must remain private")
	client := &statusNumistaClient{err: internal}
	settings := &fakeNumistaSettings{
		key:    "configured",
		config: NumistaSettings{SearchTTL: time.Hour, SearchResultLimit: 20, Valid: true},
	}
	outcome, err := newStatusLookupService(client, settings, NewNumistaTelemetry(20)).
		Lookup(context.Background(), statusLookupRequest())
	if !errors.Is(err, internal) || outcome.Status != "" {
		t.Fatalf("unexpected internal failure outcome=%+v err=%v", outcome, err)
	}
}

func TestNumistaLookupChecksConfigurationBeforeCache(t *testing.T) {
	client := &statusNumistaClient{candidates: []models.NumistaCandidate{{ID: 1, Title: "Coin"}}}
	settings := &fakeNumistaSettings{
		key:    "configured",
		config: NumistaSettings{SearchTTL: time.Hour, SearchResultLimit: 20, Valid: true},
	}
	service := newStatusLookupService(client, settings, NewNumistaTelemetry(20))
	if outcome, err := service.Lookup(context.Background(), statusLookupRequest()); err != nil ||
		outcome.Status != models.NumistaStatusSuccess {
		t.Fatalf("priming lookup outcome=%+v err=%v", outcome, err)
	}
	settings.key = ""
	outcome, err := service.Lookup(context.Background(), statusLookupRequest())
	if err != nil || outcome.Status != models.NumistaStatusUnconfigured || client.calls.Load() != 1 {
		t.Fatalf("configuration did not precede cache: outcome=%+v calls=%d err=%v", outcome, client.calls.Load(), err)
	}
	if outcome.Cache != nil {
		t.Fatalf("unconfigured outcome exposed cached metadata: %+v", outcome.Cache)
	}
}

func TestNumistaLookupPropagatesOnlyPositiveRetryAfter(t *testing.T) {
	for _, retryAfter := range []int{-10, 0, 30} {
		t.Run(time.Duration(retryAfter).String(), func(t *testing.T) {
			client := &statusNumistaClient{err: &NumistaError{
				Kind: NumistaErrorQuotaLimited, RetryAfterSeconds: &retryAfter,
			}}
			settings := &fakeNumistaSettings{
				key:    "configured",
				config: NumistaSettings{SearchTTL: time.Hour, SearchResultLimit: 20, Valid: true},
			}
			outcome, err := newStatusLookupService(client, settings, NewNumistaTelemetry(20)).
				Lookup(context.Background(), statusLookupRequest())
			if err != nil || outcome.Status != models.NumistaStatusQuotaLimited {
				t.Fatalf("outcome=%+v err=%v", outcome, err)
			}
			if retryAfter <= 0 && outcome.RetryAfterSeconds != nil {
				t.Fatalf("retryAfterSeconds=%d, want omitted", *outcome.RetryAfterSeconds)
			}
			if retryAfter > 0 && (outcome.RetryAfterSeconds == nil || *outcome.RetryAfterSeconds != retryAfter) {
				t.Fatalf("retryAfterSeconds=%v, want %d", outcome.RetryAfterSeconds, retryAfter)
			}
		})
	}
}

func TestNumistaLookupRecordsAndPropagatesCallerCancellationAndDeadline(t *testing.T) {
	tests := []struct {
		name    string
		newCtx  func() (context.Context, context.CancelFunc)
		wantErr error
	}{
		{
			name: "cancelled",
			newCtx: func() (context.Context, context.CancelFunc) {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx, func() {}
			},
			wantErr: context.Canceled,
		},
		{
			name: "deadline",
			newCtx: func() (context.Context, context.CancelFunc) {
				return context.WithTimeout(context.Background(), time.Nanosecond)
			},
			wantErr: context.DeadlineExceeded,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			telemetry := NewNumistaTelemetry(20)
			settings := &fakeNumistaSettings{
				key:    "configured",
				config: NumistaSettings{SearchTTL: time.Hour, SearchResultLimit: 20, Valid: true},
			}
			ctx, cancel := test.newCtx()
			defer cancel()
			_, err := newStatusLookupService(&statusNumistaClient{wait: true}, settings, telemetry).
				Lookup(ctx, statusLookupRequest())
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("Lookup() error=%v, want %v", err, test.wantErr)
			}
			telemetry.mu.RLock()
			events := append([]NumistaTelemetryEvent(nil), telemetry.events...)
			telemetry.mu.RUnlock()
			if len(events) != 1 || !events[0].Cancelled || events[0].Status != "" {
				t.Fatalf("cancellation telemetry=%+v", events)
			}
			health := telemetry.Health(true, true)
			if _, exists := health.StatusCounts[""]; exists || health.LastOutcome != "" {
				t.Fatalf("caller cancellation polluted domain health statuses: %+v", health)
			}
		})
	}
}
