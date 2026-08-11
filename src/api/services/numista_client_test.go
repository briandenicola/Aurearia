package services

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"
	"time"
)

type numistaRoundTripFunc func(*http.Request) (*http.Response, error)

func (f numistaRoundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func immediateNumistaRetry(_ context.Context, _ time.Duration) error {
	return nil
}

type countingNumistaReader struct {
	reader *strings.Reader
	read   int
}

func (r *countingNumistaReader) Read(buffer []byte) (int, error) {
	n, err := r.reader.Read(buffer)
	r.read += n
	return n, err
}

func TestHTTPNumistaClientMapsSearchAndProtectsKey(t *testing.T) {
	fixture, err := os.ReadFile("testdata/numista/broad_search.json")
	if err != nil {
		t.Fatal(err)
	}
	var receivedKey string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedKey = r.Header.Get("Numista-API-Key")
		if r.URL.Path != "/types" || r.URL.Query().Get("category") != "coin" || r.URL.Query().Get("count") != "20" ||
			r.URL.Query().Get("q") != "  Trajan   denarius \t Rome  " {
			t.Fatalf("unexpected request URL: %s", r.URL.String())
		}
		w.Write(fixture)
	}))
	defer server.Close()
	client, err := NewHTTPNumistaClient(NumistaClientConfig{
		BaseURL: server.URL, APIKey: func() string { return "secret-key" },
	})
	if err != nil {
		t.Fatal(err)
	}
	candidates, err := client.Search(context.Background(), "  Trajan   denarius \t Rome  ", 20)
	if err != nil {
		t.Fatal(err)
	}
	if receivedKey != "secret-key" || len(candidates) != 2 || candidates[0].CanonicalURL == "" {
		t.Fatalf("unexpected mapping: key=%q candidates=%+v", receivedKey, candidates)
	}
}

func TestHTTPNumistaClientSupportsInjectedTransport(t *testing.T) {
	transport := numistaRoundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.URL.Host != "numista.test" || request.Header.Get("Numista-API-Key") != "key" {
			t.Fatalf("unexpected injected request: %s headers=%v", request.URL, request.Header)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"types":[]}`)),
		}, nil
	})
	client, err := NewHTTPNumistaClient(NumistaClientConfig{
		BaseURL: "https://numista.test", HTTPClient: &http.Client{Transport: transport},
		APIKey: func() string { return "key" },
	})
	if err != nil {
		t.Fatal(err)
	}
	if candidates, err := client.Search(context.Background(), "coin", 10); err != nil || len(candidates) != 0 {
		t.Fatalf("injected transport result=%+v err=%v", candidates, err)
	}
}

func TestHTTPNumistaClientErrorMappingRetryAndCancellation(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch calls.Add(1) {
		case 1:
			w.WriteHeader(http.StatusServiceUnavailable)
		case 2:
			w.Header().Set("Retry-After", "60")
			w.WriteHeader(http.StatusTooManyRequests)
		default:
			<-r.Context().Done()
		}
	}))
	defer server.Close()
	client, _ := NewHTTPNumistaClient(NumistaClientConfig{
		BaseURL: server.URL, APIKey: func() string { return "secret" }, RetrySleeper: immediateNumistaRetry,
		SearchTimeout: func() time.Duration { return 20 * time.Millisecond },
	})
	_, err := client.Search(context.Background(), "coin", 10)
	var numistaErr *NumistaError
	if !errors.As(err, &numistaErr) || numistaErr.Kind != NumistaErrorQuotaLimited ||
		numistaErr.RetryAfterSeconds == nil || *numistaErr.RetryAfterSeconds != 60 {
		t.Fatalf("unexpected quota error: %#v", err)
	}
	if calls.Load() != 2 {
		t.Fatalf("calls = %d, want one eligible retry", calls.Load())
	}
	_, err = client.Search(context.Background(), "coin", 10)
	if !errors.As(err, &numistaErr) || numistaErr.Kind != NumistaErrorTimeout {
		t.Fatalf("unexpected timeout error: %#v", err)
	}
	if strings.Contains(err.Error(), "secret") {
		t.Fatal("error leaked API key")
	}
}

func TestHTTPNumistaClientRejectsMalformedAndOversizedResponses(t *testing.T) {
	for _, body := range []string{`{"types":[`, strings.Repeat("x", numistaResponseLimit+1)} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(body))
		}))
		client, _ := NewHTTPNumistaClient(NumistaClientConfig{BaseURL: server.URL, APIKey: func() string { return "key" }})
		_, err := client.Search(context.Background(), "coin", 10)
		server.Close()
		var numistaErr *NumistaError
		if !errors.As(err, &numistaErr) || numistaErr.Kind != NumistaErrorMalformedResponse {
			t.Fatalf("unexpected error for malformed response: %v", err)
		}
	}
}

func TestHTTPNumistaClientResponseBodyExactOneMiBBoundary(t *testing.T) {
	validPrefix := `{"types":[]}`
	tests := []struct {
		name     string
		size     int
		wantKind NumistaErrorKind
		wantRead int
	}{
		{name: "exactly 1 MiB accepted", size: numistaResponseLimit, wantRead: numistaResponseLimit},
		{
			name: "1 MiB plus 1 rejected", size: numistaResponseLimit + 1,
			wantKind: NumistaErrorMalformedResponse, wantRead: numistaResponseLimit + 1,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			body := validPrefix + strings.Repeat(" ", test.size-len(validPrefix))
			reader := &countingNumistaReader{reader: strings.NewReader(body)}
			transport := numistaRoundTripFunc(func(*http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     make(http.Header),
					Body:       io.NopCloser(reader),
				}, nil
			})
			client, err := NewHTTPNumistaClient(NumistaClientConfig{
				BaseURL: "https://numista.test", HTTPClient: &http.Client{Transport: transport},
				APIKey: func() string { return "key" },
			})
			if err != nil {
				t.Fatal(err)
			}
			candidates, err := client.Search(context.Background(), "coin", 10)
			if test.wantKind == "" {
				if err != nil || len(candidates) != 0 {
					t.Fatalf("exact-limit response candidates=%+v err=%v", candidates, err)
				}
			} else {
				var numistaErr *NumistaError
				if !errors.As(err, &numistaErr) || numistaErr.Kind != test.wantKind {
					t.Fatalf("over-limit error=%#v, want %q", err, test.wantKind)
				}
			}
			if reader.read != test.wantRead {
				t.Fatalf("body bytes read=%d, want bounded read of %d", reader.read, test.wantRead)
			}
		})
	}
}

func TestHTTPNumistaClientMapsNonRetryableStatuses(t *testing.T) {
	tests := []struct {
		status int
		kind   NumistaErrorKind
	}{
		{http.StatusBadRequest, NumistaErrorInvalidRequest},
		{http.StatusUnauthorized, NumistaErrorUnauthorized},
		{http.StatusForbidden, NumistaErrorUnauthorized},
		{http.StatusInternalServerError, NumistaErrorUnavailable},
	}
	for _, test := range tests {
		t.Run(http.StatusText(test.status), func(t *testing.T) {
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				w.WriteHeader(test.status)
			}))
			defer server.Close()
			client, _ := NewHTTPNumistaClient(NumistaClientConfig{
				BaseURL: server.URL, APIKey: func() string { return "key" }, RetrySleeper: immediateNumistaRetry,
			})
			_, err := client.Search(context.Background(), "coin", 10)
			var numistaErr *NumistaError
			if !errors.As(err, &numistaErr) || numistaErr.Kind != test.kind || calls.Load() != 1 {
				t.Fatalf("status %d mapped to %#v after %d calls", test.status, err, calls.Load())
			}
		})
	}
}

func TestHTTPNumistaClientNeverRetriesQuotaLimitedResponses(t *testing.T) {
	var calls atomic.Int32
	var sleeps atomic.Int32
	transport := numistaRoundTripFunc(func(*http.Request) (*http.Response, error) {
		calls.Add(1)
		header := make(http.Header)
		header.Set("Retry-After", "120")
		return &http.Response{
			StatusCode: http.StatusTooManyRequests,
			Header:     header,
			Body:       io.NopCloser(strings.NewReader(`{"error":"quota"}`)),
		}, nil
	})
	client, err := NewHTTPNumistaClient(NumistaClientConfig{
		BaseURL: "https://numista.test", HTTPClient: &http.Client{Transport: transport},
		APIKey: func() string { return "key" },
		RetrySleeper: func(context.Context, time.Duration) error {
			sleeps.Add(1)
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.Search(context.Background(), "coin", 10)
	var numistaErr *NumistaError
	if !errors.As(err, &numistaErr) || numistaErr.Kind != NumistaErrorQuotaLimited ||
		numistaErr.RetryAfterSeconds == nil || *numistaErr.RetryAfterSeconds != 120 {
		t.Fatalf("unexpected quota error: %#v", err)
	}
	if calls.Load() != 1 || sleeps.Load() != 0 {
		t.Fatalf("429 calls=%d backoffs=%d, want one call and no backoff", calls.Load(), sleeps.Load())
	}
}

func TestHTTPNumistaClientRetriesEachEligibleGatewayStatusOnce(t *testing.T) {
	for _, status := range []int{http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			var calls atomic.Int32
			var sleeps atomic.Int32
			transport := numistaRoundTripFunc(func(*http.Request) (*http.Response, error) {
				if calls.Add(1) == 1 {
					return &http.Response{
						StatusCode: status,
						Header:     make(http.Header),
						Body:       io.NopCloser(strings.NewReader(`{}`)),
					}, nil
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     make(http.Header),
					Body:       io.NopCloser(strings.NewReader(`{"types":[]}`)),
				}, nil
			})
			client, err := NewHTTPNumistaClient(NumistaClientConfig{
				BaseURL: "https://numista.test", HTTPClient: &http.Client{Transport: transport},
				APIKey: func() string { return "key" },
				RetryJitter: func(minimum, _ time.Duration) time.Duration {
					return minimum
				},
				RetrySleeper: func(_ context.Context, delay time.Duration) error {
					if delay != numistaRetryMinDelay {
						t.Fatalf("retry delay=%s, want %s", delay, numistaRetryMinDelay)
					}
					sleeps.Add(1)
					return nil
				},
			})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := client.Search(context.Background(), "coin", 10); err != nil {
				t.Fatalf("eligible status was not recovered: %v", err)
			}
			if calls.Load() != 2 || sleeps.Load() != 1 {
				t.Fatalf("status %d calls=%d backoffs=%d, want 2 and 1", status, calls.Load(), sleeps.Load())
			}
		})
	}
}

func TestHTTPNumistaClientDiscardsMalformedRequiredFieldsAndBoundsOptionalFields(t *testing.T) {
	transport := numistaRoundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body: io.NopCloser(strings.NewReader(`{"types":[
				{"id":0,"title":"invalid id"},
				{"id":1,"title":"  "},
				{"id":2,"title":"Valid","min_year":10,"max_year":1,
				 "obverse_thumbnail":"http://unsafe.example/coin.jpg"}
			]}`)),
		}, nil
	})
	client, err := NewHTTPNumistaClient(NumistaClientConfig{
		BaseURL: "https://numista.test", HTTPClient: &http.Client{Transport: transport},
		APIKey: func() string { return "key" },
	})
	if err != nil {
		t.Fatal(err)
	}
	candidates, err := client.Search(context.Background(), "coin", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) != 1 || candidates[0].ID != 2 ||
		candidates[0].MinYear != nil || candidates[0].MaxYear != nil ||
		candidates[0].ObverseThumbnail != "" {
		t.Fatalf("malformed provider fields were not safely mapped: %+v", candidates)
	}
}

func TestHTTPNumistaClientUsesSearchAndDetailDeadlines(t *testing.T) {
	var searchDeadline, detailDeadline time.Duration
	transport := numistaRoundTripFunc(func(request *http.Request) (*http.Response, error) {
		deadline, ok := request.Context().Deadline()
		if !ok {
			t.Fatal("request context has no deadline")
		}
		remaining := time.Until(deadline)
		body := `{"types":[]}`
		if request.URL.Path == "/types/1" {
			detailDeadline = remaining
			body = `{"id":1,"title":"Coin"}`
		} else {
			searchDeadline = remaining
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(body)),
		}, nil
	})
	client, err := NewHTTPNumistaClient(NumistaClientConfig{
		BaseURL: "https://numista.test", HTTPClient: &http.Client{Transport: transport},
		APIKey: func() string { return "key" },
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.Search(context.Background(), "coin", 10); err != nil {
		t.Fatal(err)
	}
	if _, err := client.Detail(context.Background(), 1); err != nil {
		t.Fatal(err)
	}
	if searchDeadline < 3500*time.Millisecond || searchDeadline > 4*time.Second ||
		detailDeadline < 2500*time.Millisecond || detailDeadline > 3*time.Second {
		t.Fatalf("deadlines search=%s detail=%s, want approximately 4s and 3s", searchDeadline, detailDeadline)
	}
}

func TestHTTPNumistaClientMapsExpiredRequestDeadlineWithoutRetry(t *testing.T) {
	var calls atomic.Int32
	var sleeps atomic.Int32
	transport := numistaRoundTripFunc(func(request *http.Request) (*http.Response, error) {
		calls.Add(1)
		<-request.Context().Done()
		return nil, request.Context().Err()
	})
	client, err := NewHTTPNumistaClient(NumistaClientConfig{
		BaseURL: "https://numista.test", HTTPClient: &http.Client{Transport: transport},
		APIKey: func() string { return "key" },
		SearchTimeout: func() time.Duration {
			return 20 * time.Millisecond
		},
		RetrySleeper: func(context.Context, time.Duration) error {
			sleeps.Add(1)
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.Search(context.Background(), "coin", 10)
	var numistaErr *NumistaError
	if !errors.As(err, &numistaErr) || numistaErr.Kind != NumistaErrorTimeout {
		t.Fatalf("unexpected deadline error: %#v", err)
	}
	if calls.Load() != 1 || sleeps.Load() != 0 {
		t.Fatalf("deadline calls=%d backoffs=%d, want one call and no backoff", calls.Load(), sleeps.Load())
	}
}

func TestHTTPNumistaClientHonorsCallerCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer server.Close()
	client, _ := NewHTTPNumistaClient(NumistaClientConfig{
		BaseURL: server.URL, APIKey: func() string { return "key" },
		SearchTimeout: func() time.Duration { return time.Second },
	})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := client.Search(ctx, "coin", 10)
	var numistaErr *NumistaError
	if !errors.As(err, &numistaErr) || numistaErr.Kind != NumistaErrorCancelled {
		t.Fatalf("unexpected cancellation error: %#v", err)
	}
}

func TestHTTPNumistaClientRetriesOnlyConnectionResetTransportFailures(t *testing.T) {
	t.Run("connection reset", func(t *testing.T) {
		var calls atomic.Int32
		transport := numistaRoundTripFunc(func(*http.Request) (*http.Response, error) {
			if calls.Add(1) == 1 {
				return nil, syscall.ECONNRESET
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(`{"types":[]}`)),
			}, nil
		})
		client, err := NewHTTPNumistaClient(NumistaClientConfig{
			BaseURL: "https://numista.test", HTTPClient: &http.Client{Transport: transport},
			APIKey: func() string { return "key" }, RetrySleeper: immediateNumistaRetry,
		})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := client.Search(context.Background(), "coin", 10); err != nil {
			t.Fatalf("connection reset was not retried successfully: %v", err)
		}
		if calls.Load() != 2 {
			t.Fatalf("calls = %d, want 2", calls.Load())
		}
	})

	t.Run("deterministic transport failure", func(t *testing.T) {
		var calls atomic.Int32
		transport := numistaRoundTripFunc(func(*http.Request) (*http.Response, error) {
			calls.Add(1)
			return nil, errors.New("deterministic client failure")
		})
		client, err := NewHTTPNumistaClient(NumistaClientConfig{
			BaseURL: "https://numista.test", HTTPClient: &http.Client{Transport: transport},
			APIKey: func() string { return "key" }, RetrySleeper: immediateNumistaRetry,
		})
		if err != nil {
			t.Fatal(err)
		}
		_, err = client.Search(context.Background(), "coin", 10)
		var numistaErr *NumistaError
		if !errors.As(err, &numistaErr) || numistaErr.Kind != NumistaErrorUnavailable {
			t.Fatalf("unexpected error: %#v", err)
		}
		if calls.Load() != 1 {
			t.Fatalf("deterministic failure calls = %d, want 1", calls.Load())
		}
	})
}

func TestHTTPNumistaClientUsesBoundedInjectedJitter(t *testing.T) {
	var calls atomic.Int32
	var jitterMinimum, jitterMaximum, slept time.Duration
	transport := numistaRoundTripFunc(func(*http.Request) (*http.Response, error) {
		if calls.Add(1) == 1 {
			return nil, syscall.ECONNRESET
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"types":[]}`)),
		}, nil
	})
	client, err := NewHTTPNumistaClient(NumistaClientConfig{
		BaseURL:    "https://numista.test",
		HTTPClient: &http.Client{Transport: transport},
		APIKey:     func() string { return "key" },
		RetryJitter: func(minimum, maximum time.Duration) time.Duration {
			jitterMinimum, jitterMaximum = minimum, maximum
			return 225 * time.Millisecond
		},
		RetrySleeper: func(_ context.Context, delay time.Duration) error {
			slept = delay
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.Search(context.Background(), "coin", 10); err != nil {
		t.Fatal(err)
	}
	if jitterMinimum != 100*time.Millisecond || jitterMaximum != 300*time.Millisecond ||
		slept != 225*time.Millisecond || calls.Load() != 2 {
		t.Fatalf("jitter bounds=%s-%s slept=%s calls=%d", jitterMinimum, jitterMaximum, slept, calls.Load())
	}
}

func TestHTTPNumistaClientCancelsDuringRetryBackoff(t *testing.T) {
	var calls atomic.Int32
	transport := numistaRoundTripFunc(func(*http.Request) (*http.Response, error) {
		calls.Add(1)
		return nil, syscall.ECONNRESET
	})
	ctx, cancel := context.WithCancel(context.Background())
	client, err := NewHTTPNumistaClient(NumistaClientConfig{
		BaseURL:     "https://numista.test",
		HTTPClient:  &http.Client{Transport: transport},
		APIKey:      func() string { return "key" },
		RetryJitter: func(_, _ time.Duration) time.Duration { return 200 * time.Millisecond },
		RetrySleeper: func(ctx context.Context, _ time.Duration) error {
			cancel()
			<-ctx.Done()
			return ctx.Err()
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.Search(ctx, "coin", 10)
	var numistaErr *NumistaError
	if !errors.As(err, &numistaErr) || numistaErr.Kind != NumistaErrorCancelled || calls.Load() != 1 {
		t.Fatalf("error=%#v calls=%d", err, calls.Load())
	}
}
