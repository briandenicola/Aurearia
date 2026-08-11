package services

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
)

func newDetailTestClient(t *testing.T, transport http.RoundTripper, configure ...func(*NumistaClientConfig)) *HTTPNumistaClient {
	t.Helper()
	config := NumistaClientConfig{
		BaseURL:    "https://numista.test",
		HTTPClient: &http.Client{Transport: transport},
		APIKey:     func() string { return "test-key" },
		RetrySleeper: func(context.Context, time.Duration) error {
			return nil
		},
	}
	for _, apply := range configure {
		apply(&config)
	}
	client, err := NewHTTPNumistaClient(config)
	if err != nil {
		t.Fatal(err)
	}
	return client
}

func detailResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func TestHTTPNumistaClientDetailValidatesIDBeforeProviderCall(t *testing.T) {
	var calls atomic.Int32
	client := newDetailTestClient(t, numistaRoundTripFunc(func(request *http.Request) (*http.Response, error) {
		calls.Add(1)
		id := strings.TrimPrefix(request.URL.Path, "/types/")
		return detailResponse(http.StatusOK, `{"id":`+id+`,"title":"Coin"}`), nil
	}))

	for _, id := range []int{1, 50, 2147483647} {
		candidate, err := client.Detail(context.Background(), id)
		if err != nil || candidate.ID != id {
			t.Fatalf("Detail(%d) candidate=%+v error=%v", id, candidate, err)
		}
	}
	for _, id := range []int{0, -1} {
		_, err := client.Detail(context.Background(), id)
		var numistaErr *NumistaError
		if !errors.As(err, &numistaErr) || numistaErr.Kind != NumistaErrorInvalidRequest {
			t.Fatalf("Detail(%d) error=%#v, want invalid request", id, err)
		}
	}
	if calls.Load() != 3 {
		t.Fatalf("provider calls=%d, want only the three valid IDs", calls.Load())
	}
}

func TestHTTPNumistaClientDetailMapsOnlyApplicationFieldsAndCanonicalizesURLs(t *testing.T) {
	client := newDetailTestClient(t, numistaRoundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.Method != http.MethodGet || request.URL.Path != "/types/42" {
			t.Fatalf("unexpected detail request: %s %s", request.Method, request.URL)
		}
		return detailResponse(http.StatusOK, `{
			"id":42,
			"title":"  Trajan Denarius  ",
			"issuer":{"name":"Roman Empire"},
			"value":{"text":"Denarius"},
			"composition":{"text":"Silver"},
			"mints":[{"name":"Rome"},{"name":"Antioch"}],
			"min_year":101,
			"max_year":102,
			"obverse":{"inscription":"IMP TRAIANO AVG GER DAC P M TR P"},
			"reverse":{"inscription":"COS V P P S P Q R OPTIMO PRINC"},
			"obverse_thumbnail":"https://images.numista.test/obverse.jpg",
			"reverse_thumbnail":"http://images.numista.test/reverse.jpg",
			"comments":"provider-only data must not cross the boundary",
			"rarity_index":99
		}`), nil
	}))

	candidate, err := client.Detail(context.Background(), 42)
	if err != nil {
		t.Fatal(err)
	}
	if candidate.ID != 42 ||
		candidate.CanonicalURL != "https://en.numista.com/catalogue/pieces42.html" ||
		candidate.Title != "Trajan Denarius" ||
		candidate.Issuer != "Roman Empire" ||
		candidate.Denomination != "Denarius" ||
		candidate.Material != "Silver" ||
		candidate.Mint != "Rome" ||
		candidate.YearDisplay != "101 CE–102 CE" ||
		candidate.ObverseInscription == "" ||
		candidate.ReverseInscription == "" ||
		candidate.ObverseThumbnail != "https://images.numista.test/obverse.jpg" ||
		candidate.ReverseThumbnail != "" ||
		candidate.EnrichmentState != models.NumistaEnrichmentEnriched {
		t.Fatalf("unexpected mapped detail: %+v", candidate)
	}
}

func TestHTTPNumistaClientDetailToleratesMalformedOptionalFields(t *testing.T) {
	client := newDetailTestClient(t, numistaRoundTripFunc(func(*http.Request) (*http.Response, error) {
		return detailResponse(http.StatusOK, `{
			"id":7,
			"title":"Usable broad identity",
			"issuer":{"name":17},
			"value":"Denarius",
			"composition":{"text":["Silver"]},
			"mints":[null,{"name":"Rome"}],
			"min_year":"unknown",
			"max_year":120,
			"obverse":{"inscription":{"text":"legend"}},
			"reverse":false,
			"obverse_thumbnail":"javascript:alert(1)",
			"reverse_thumbnail":"//images.numista.test/reverse.jpg"
		}`), nil
	}))

	candidate, err := client.Detail(context.Background(), 7)
	if err != nil {
		t.Fatalf("malformed optional fields discarded a usable detail: %v", err)
	}
	if candidate.ID != 7 || candidate.Title != "Usable broad identity" ||
		candidate.Issuer != "" || candidate.Denomination != "" ||
		candidate.Material != "" || candidate.MinYear != nil ||
		candidate.ObverseThumbnail != "" || candidate.ReverseThumbnail != "" {
		t.Fatalf("malformed optionals were not omitted safely: %+v", candidate)
	}
}

func TestHTTPNumistaClientDetailRejectsMismatchedOrMalformedRequiredIdentity(t *testing.T) {
	for _, body := range []string{
		`{"id":8,"title":"Wrong ID"}`,
		`{"id":7,"title":" "}`,
		`{"id":"7","title":"Wrong type"}`,
		`{"id":7`,
	} {
		t.Run(body, func(t *testing.T) {
			client := newDetailTestClient(t, numistaRoundTripFunc(func(*http.Request) (*http.Response, error) {
				return detailResponse(http.StatusOK, body), nil
			}))
			_, err := client.Detail(context.Background(), 7)
			var numistaErr *NumistaError
			if !errors.As(err, &numistaErr) || numistaErr.Kind != NumistaErrorMalformedResponse {
				t.Fatalf("error=%#v, want malformed response", err)
			}
		})
	}
}

func TestNumistaLookupDetailCachesSuccessfulMappedDetailAndRecordsRealTelemetry(t *testing.T) {
	clock := &fakeNumistaClock{now: time.Date(2026, 8, 11, 12, 0, 0, 0, time.UTC)}
	var calls atomic.Int32
	client := newDetailTestClient(t, numistaRoundTripFunc(func(*http.Request) (*http.Response, error) {
		calls.Add(1)
		clock.Add(25 * time.Millisecond)
		return detailResponse(http.StatusOK, `{"id":9,"title":"Cached Coin","composition":{"text":"Silver"}}`), nil
	}))
	telemetry := NewNumistaTelemetry(10)
	service := NewNumistaLookupService(
		client,
		NewNumistaCache(clock, 10, 10),
		NewNumistaV1Scorer(),
		telemetry,
		&fakeNumistaSettings{key: "configured", config: NumistaSettings{DetailTTL: 7 * 24 * time.Hour, Valid: true}},
		clock,
	)

	fresh, freshCache, err := service.LookupDetail(context.Background(), models.NumistaLookupPathDirect, 9)
	if err != nil || freshCache == nil || freshCache.Hit || fresh.Material != "Silver" {
		t.Fatalf("fresh detail=%+v cache=%+v error=%v", fresh, freshCache, err)
	}
	clock.Add(time.Hour)
	cached, cachedMetadata, err := service.LookupDetail(context.Background(), models.NumistaLookupPathPhoto, 9)
	if err != nil || cachedMetadata == nil || !cachedMetadata.Hit ||
		cached.EnrichmentState != models.NumistaEnrichmentCached {
		t.Fatalf("cached detail=%+v cache=%+v error=%v", cached, cachedMetadata, err)
	}
	if calls.Load() != 1 {
		t.Fatalf("provider calls=%d, want one cacheable successful detail", calls.Load())
	}
	health := telemetry.Health(true, true)
	if health.DetailRequestCount != 1 || health.ProviderLoadCount != 1 ||
		health.FreshCacheHitCount != 1 || health.EnrichmentAttempted != 1 ||
		health.EnrichmentSucceeded != 1 || health.EnrichmentFailed != 0 {
		t.Fatalf("detail telemetry=%+v", health)
	}
}

func TestHTTPNumistaClientDetailTimeoutCancellationAndRetryPolicy(t *testing.T) {
	t.Run("detail timeout uses configured deadline without retry", func(t *testing.T) {
		var calls atomic.Int32
		started := make(chan struct{})
		client := newDetailTestClient(t, numistaRoundTripFunc(func(request *http.Request) (*http.Response, error) {
			calls.Add(1)
			close(started)
			<-request.Context().Done()
			return nil, request.Context().Err()
		}), func(config *NumistaClientConfig) {
			config.DetailTimeout = func() time.Duration { return 20 * time.Millisecond }
		})
		_, err := client.Detail(context.Background(), 1)
		<-started
		var numistaErr *NumistaError
		if !errors.As(err, &numistaErr) || numistaErr.Kind != NumistaErrorTimeout || calls.Load() != 1 {
			t.Fatalf("error=%#v calls=%d", err, calls.Load())
		}
	})

	t.Run("caller cancellation aborts detail", func(t *testing.T) {
		started := make(chan struct{})
		client := newDetailTestClient(t, numistaRoundTripFunc(func(request *http.Request) (*http.Response, error) {
			close(started)
			<-request.Context().Done()
			return nil, request.Context().Err()
		}))
		ctx, cancel := context.WithCancel(context.Background())
		result := make(chan error, 1)
		go func() {
			_, err := client.Detail(ctx, 1)
			result <- err
		}()
		<-started
		cancel()
		var numistaErr *NumistaError
		if err := <-result; !errors.As(err, &numistaErr) || numistaErr.Kind != NumistaErrorCancelled {
			t.Fatalf("error=%#v, want cancelled", err)
		}
	})

	t.Run("one transient retry and no deterministic retry", func(t *testing.T) {
		var calls atomic.Int32
		client := newDetailTestClient(t, numistaRoundTripFunc(func(*http.Request) (*http.Response, error) {
			if calls.Add(1) == 1 {
				return nil, syscall.ECONNRESET
			}
			return detailResponse(http.StatusOK, `{"id":1,"title":"Recovered"}`), nil
		}))
		if _, err := client.Detail(context.Background(), 1); err != nil || calls.Load() != 2 {
			t.Fatalf("transient retry error=%v calls=%d", err, calls.Load())
		}

		calls.Store(0)
		client = newDetailTestClient(t, numistaRoundTripFunc(func(*http.Request) (*http.Response, error) {
			calls.Add(1)
			return detailResponse(http.StatusTooManyRequests, `{"error":"quota"}`), nil
		}))
		_, err := client.Detail(context.Background(), 1)
		var numistaErr *NumistaError
		if !errors.As(err, &numistaErr) || numistaErr.Kind != NumistaErrorQuotaLimited || calls.Load() != 1 {
			t.Fatalf("quota error=%#v calls=%d", err, calls.Load())
		}
	})
}
