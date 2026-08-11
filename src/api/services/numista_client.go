package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
)

const (
	defaultNumistaBaseURL = "https://api.numista.com/v3"
	numistaResponseLimit  = 1 << 20
	numistaRetryMinDelay  = 100 * time.Millisecond
	numistaRetryMaxDelay  = 300 * time.Millisecond
)

type NumistaErrorKind string

const (
	NumistaErrorInvalidRequest    NumistaErrorKind = "invalid_request"
	NumistaErrorUnauthorized      NumistaErrorKind = "unauthorized"
	NumistaErrorQuotaLimited      NumistaErrorKind = "quota_limited"
	NumistaErrorTimeout           NumistaErrorKind = "timeout"
	NumistaErrorUnavailable       NumistaErrorKind = "unavailable"
	NumistaErrorMalformedResponse NumistaErrorKind = "malformed_response"
	NumistaErrorCancelled         NumistaErrorKind = "cancelled"
	NumistaErrorUnconfigured      NumistaErrorKind = "unconfigured"
)

type NumistaError struct {
	Kind              NumistaErrorKind
	RetryAfterSeconds *int
}

func (e *NumistaError) Error() string {
	return "Numista request failed: " + string(e.Kind)
}

type NumistaClient interface {
	Search(ctx context.Context, query string, limit int) ([]models.NumistaCandidate, error)
	Detail(ctx context.Context, id int) (models.NumistaCandidate, error)
}

type NumistaClientConfig struct {
	BaseURL       string
	HTTPClient    *http.Client
	APIKey        func() string
	SearchTimeout func() time.Duration
	DetailTimeout func() time.Duration
	RetryJitter   func(minimum, maximum time.Duration) time.Duration
	RetrySleeper  func(context.Context, time.Duration) error
}

type HTTPNumistaClient struct {
	baseURL       string
	client        *http.Client
	apiKey        func() string
	searchTimeout func() time.Duration
	detailTimeout func() time.Duration
	retryJitter   func(minimum, maximum time.Duration) time.Duration
	retrySleeper  func(context.Context, time.Duration) error
}

func NewHTTPNumistaClient(config NumistaClientConfig) (*HTTPNumistaClient, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(config.BaseURL), "/")
	if baseURL == "" {
		baseURL = defaultNumistaBaseURL
	}
	parsed, err := url.Parse(baseURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return nil, errors.New("invalid Numista base URL")
	}
	if config.HTTPClient == nil {
		config.HTTPClient = &http.Client{}
	}
	if config.APIKey == nil {
		config.APIKey = func() string { return "" }
	}
	if config.SearchTimeout == nil {
		config.SearchTimeout = func() time.Duration { return 4 * time.Second }
	}
	if config.DetailTimeout == nil {
		config.DetailTimeout = func() time.Duration { return 3 * time.Second }
	}
	if config.RetryJitter == nil {
		config.RetryJitter = func(minimum, maximum time.Duration) time.Duration {
			return minimum + time.Duration(rand.Int64N(int64(maximum-minimum)+1))
		}
	}
	if config.RetrySleeper == nil {
		config.RetrySleeper = sleepForNumistaRetry
	}
	return &HTTPNumistaClient{
		baseURL: baseURL, client: config.HTTPClient, apiKey: config.APIKey,
		searchTimeout: config.SearchTimeout, detailTimeout: config.DetailTimeout,
		retryJitter: config.RetryJitter, retrySleeper: config.RetrySleeper,
	}, nil
}

func (c *HTTPNumistaClient) Search(ctx context.Context, query string, limit int) ([]models.NumistaCandidate, error) {
	values := url.Values{}
	values.Set("q", query)
	values.Set("category", "coin")
	values.Set("count", strconv.Itoa(limit))
	values.Set("lang", "en")
	body, err := c.get(ctx, "/types?"+values.Encode(), c.searchTimeout())
	if err != nil {
		return nil, err
	}
	var response providerSearchResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, &NumistaError{Kind: NumistaErrorMalformedResponse}
	}
	candidates := make([]models.NumistaCandidate, 0, len(response.Types))
	for position, item := range response.Types {
		candidate, ok := mapProviderType(item, position)
		if ok {
			candidates = append(candidates, candidate)
		}
	}
	return candidates, nil
}

func (c *HTTPNumistaClient) Detail(ctx context.Context, id int) (models.NumistaCandidate, error) {
	if id <= 0 {
		return models.NumistaCandidate{}, &NumistaError{Kind: NumistaErrorInvalidRequest}
	}
	body, err := c.get(ctx, "/types/"+strconv.Itoa(id), c.detailTimeout())
	if err != nil {
		return models.NumistaCandidate{}, err
	}
	var item providerType
	if err := json.Unmarshal(body, &item); err != nil {
		return models.NumistaCandidate{}, &NumistaError{Kind: NumistaErrorMalformedResponse}
	}
	candidate, ok := mapProviderType(item, 0)
	if !ok || candidate.ID != id {
		return models.NumistaCandidate{}, &NumistaError{Kind: NumistaErrorMalformedResponse}
	}
	candidate.EnrichmentState = models.NumistaEnrichmentEnriched
	return candidate, nil
}

func (c *HTTPNumistaClient) get(ctx context.Context, path string, timeout time.Duration) ([]byte, error) {
	apiKey := strings.TrimSpace(c.apiKey())
	if apiKey == "" {
		return nil, &NumistaError{Kind: NumistaErrorUnconfigured}
	}
	requestCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	for attempt := 0; attempt < 2; attempt++ {
		req, err := http.NewRequestWithContext(requestCtx, http.MethodGet, c.baseURL+path, nil)
		if err != nil {
			return nil, &NumistaError{Kind: NumistaErrorInvalidRequest}
		}
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Numista-API-Key", apiKey)
		resp, err := c.client.Do(req)
		if err != nil {
			if requestCtx.Err() != nil {
				return nil, contextNumistaError(ctx, requestCtx)
			}
			if attempt == 0 && isRetryableNumistaTransportError(err) {
				if err := c.waitForRetry(requestCtx); err != nil {
					return nil, contextNumistaError(ctx, requestCtx)
				}
				continue
			}
			return nil, &NumistaError{Kind: NumistaErrorUnavailable}
		}

		body, readErr := readLimitedNumistaBody(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			return nil, &NumistaError{Kind: NumistaErrorMalformedResponse}
		}
		if resp.StatusCode == http.StatusOK {
			return body, nil
		}
		if attempt == 0 && isRetryableNumistaStatus(resp.StatusCode) {
			if err := c.waitForRetry(requestCtx); err != nil {
				return nil, contextNumistaError(ctx, requestCtx)
			}
			continue
		}
		return nil, mapNumistaStatus(resp.StatusCode, resp.Header.Get("Retry-After"))
	}
	return nil, &NumistaError{Kind: NumistaErrorUnavailable}
}

func isRetryableNumistaTransportError(err error) bool {
	return errors.Is(err, syscall.ECONNRESET)
}

func readLimitedNumistaBody(body io.Reader) ([]byte, error) {
	limited := io.LimitReader(body, numistaResponseLimit+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if len(data) > numistaResponseLimit {
		return nil, errors.New("Numista response exceeded limit")
	}
	return data, nil
}

func contextNumistaError(parent, request context.Context) error {
	if errors.Is(parent.Err(), context.Canceled) {
		return &NumistaError{Kind: NumistaErrorCancelled}
	}
	if errors.Is(request.Err(), context.DeadlineExceeded) {
		return &NumistaError{Kind: NumistaErrorTimeout}
	}
	return &NumistaError{Kind: NumistaErrorCancelled}
}

func (c *HTTPNumistaClient) waitForRetry(ctx context.Context) error {
	delay := c.retryJitter(numistaRetryMinDelay, numistaRetryMaxDelay)
	if delay < numistaRetryMinDelay {
		delay = numistaRetryMinDelay
	}
	if delay > numistaRetryMaxDelay {
		delay = numistaRetryMaxDelay
	}
	return c.retrySleeper(ctx, delay)
}

func sleepForNumistaRetry(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func isRetryableNumistaStatus(status int) bool {
	return status == http.StatusBadGateway || status == http.StatusServiceUnavailable || status == http.StatusGatewayTimeout
}

func mapNumistaStatus(status int, retryAfter string) error {
	switch status {
	case http.StatusBadRequest:
		return &NumistaError{Kind: NumistaErrorInvalidRequest}
	case http.StatusUnauthorized, http.StatusForbidden:
		return &NumistaError{Kind: NumistaErrorUnauthorized}
	case http.StatusTooManyRequests:
		return &NumistaError{Kind: NumistaErrorQuotaLimited, RetryAfterSeconds: parseRetryAfter(retryAfter)}
	default:
		return &NumistaError{Kind: NumistaErrorUnavailable}
	}
}

func parseRetryAfter(value string) *int {
	seconds, err := strconv.Atoi(strings.TrimSpace(value))
	if err == nil && seconds > 0 {
		return &seconds
	}
	if retryAt, err := http.ParseTime(value); err == nil {
		seconds = int(time.Until(retryAt).Seconds())
		if seconds > 0 {
			return &seconds
		}
	}
	return nil
}

type providerSearchResponse struct {
	Types []providerType `json:"types"`
}

type providerType struct {
	ID               int            `json:"id"`
	Title            string         `json:"title"`
	Issuer           *providerName  `json:"issuer"`
	MinYear          *int           `json:"min_year"`
	MaxYear          *int           `json:"max_year"`
	ObverseThumbnail string         `json:"obverse_thumbnail"`
	ReverseThumbnail string         `json:"reverse_thumbnail"`
	Value            *providerText  `json:"value"`
	Composition      *providerText  `json:"composition"`
	Mints            []providerName `json:"mints"`
	Obverse          *providerSide  `json:"obverse"`
	Reverse          *providerSide  `json:"reverse"`
}

type providerName struct {
	Name string `json:"name"`
}

type providerText struct {
	Text string `json:"text"`
}

type providerSide struct {
	Inscription string `json:"inscription"`
}

func mapProviderType(item providerType, position int) (models.NumistaCandidate, bool) {
	title := boundedProviderText(item.Title, 500)
	if item.ID <= 0 || title == "" {
		return models.NumistaCandidate{}, false
	}
	if item.MinYear != nil && item.MaxYear != nil && *item.MinYear > *item.MaxYear {
		item.MinYear, item.MaxYear = nil, nil
	}
	canonicalURL, _ := models.CanonicalNumistaURL(item.ID)
	candidate := models.NumistaCandidate{
		ID: item.ID, CanonicalURL: canonicalURL, Title: title,
		MinYear: item.MinYear, MaxYear: item.MaxYear, YearDisplay: formatNumistaYears(item.MinYear, item.MaxYear),
		ObverseThumbnail: safeNumistaImageURL(item.ObverseThumbnail),
		ReverseThumbnail: safeNumistaImageURL(item.ReverseThumbnail),
		ProviderPosition: position, EnrichmentState: models.NumistaEnrichmentNotRequested,
	}
	if item.Issuer != nil {
		candidate.Issuer = boundedProviderText(item.Issuer.Name, 200)
	}
	if item.Value != nil {
		candidate.Denomination = boundedProviderText(item.Value.Text, 100)
	}
	if item.Composition != nil {
		candidate.Material = boundedProviderText(item.Composition.Text, 100)
	}
	if len(item.Mints) > 0 {
		candidate.Mint = boundedProviderText(item.Mints[0].Name, 200)
	}
	if item.Obverse != nil {
		candidate.ObverseInscription = boundedProviderText(item.Obverse.Inscription, 500)
	}
	if item.Reverse != nil {
		candidate.ReverseInscription = boundedProviderText(item.Reverse.Inscription, 500)
	}
	return candidate, true
}

func safeNumistaImageURL(raw string) string {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
		return ""
	}
	return parsed.String()
}

func boundedProviderText(value string, max int) string {
	runes := []rune(strings.TrimSpace(value))
	if len(runes) > max {
		runes = runes[:max]
	}
	return string(runes)
}

func formatNumistaYears(minYear, maxYear *int) string {
	if minYear == nil && maxYear == nil {
		return ""
	}
	format := func(year int) string {
		if year < 0 {
			return fmt.Sprintf("%d BCE", -year)
		}
		return fmt.Sprintf("%d CE", year)
	}
	if minYear == nil {
		return format(*maxYear)
	}
	if maxYear == nil || *minYear == *maxYear {
		return format(*minYear)
	}
	return format(*minYear) + "–" + format(*maxYear)
}
