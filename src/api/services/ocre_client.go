package services

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Feature 345 — the single OCRE/Nomisma-triplestore HTTP boundary.
//
// This mirrors nomisma_client.go: every failure mode collapses to a typed
// OCREErrorKind so no handler/node ever has to string-match a raw error, and
// the never-5xx contract (FR-015) is honored one layer up. Transport is a
// fixed-template GET to https://nomisma.org/query with a SPARQL-results JSON
// Accept header and a non-default User-Agent (a default/empty UA is rejected
// 403 at the Cloudflare edge — research R1). POST is never used.

const (
	defaultOCREBaseURL   = "https://nomisma.org/query"
	ocreRequestTimeout   = 8 * time.Second
	ocreResponseLimit    = 1 << 20 // 1 MiB
	ocreUserAgent        = "AncientCoins-DeepAnalysis/1.0 (+https://github.com/briandenicola/Aurearia; OCRE ODbL provider)"
	ocreAcceptSPARQLJSON = "application/sparql-results+json"
)

// OCREErrorKind distinguishes the outcomes an OCRE search can produce.
type OCREErrorKind string

const (
	OCREErrorUnavailable     OCREErrorKind = "unavailable"
	OCREErrorNoMatch         OCREErrorKind = "no_match"
	OCREErrorInvalidResponse OCREErrorKind = "invalid_response"
	OCREErrorInvalidRequest  OCREErrorKind = "invalid_request"
	OCREErrorCancelled       OCREErrorKind = "cancelled"
)

// OCRECandidate is a single scored OCRE coin-type candidate. Transient
// (never persisted as a row of its own) and the wire shape returned by the
// internal ocre_search tool.
type OCRECandidate struct {
	TypeURI       string   `json:"type_uri"`
	Label         string   `json:"label"`
	MatchedFields []string `json:"matched_fields"`
	Confidence    float64  `json:"confidence"`
	Explanation   string   `json:"explanation"`
}

// OCREClient is the single HTTP boundary for OCRE data via Nomisma's SPARQL
// endpoint. Never called from handlers/repositories directly — only via the
// DeepProviderToolsHandler.OCRESearch tool.
type OCREClient interface {
	// Search runs the bound SPARQL query and returns ranked candidates. A
	// returned OCREErrorKind of "" means success (candidates may be empty,
	// i.e. no_match at the caller's discretion); any non-empty kind is a
	// typed, non-blocking outcome, never a hard failure.
	Search(ctx context.Context, params OCREQueryParams, limit int) ([]OCRECandidate, OCREErrorKind, error)
}

// HTTPOCREClient implements OCREClient over stdlib net/http. No credential
// is required or stored.
type HTTPOCREClient struct {
	baseURL string
	client  *http.Client
}

// NewHTTPOCREClient creates an HTTPOCREClient pointed at the public Nomisma
// SPARQL query endpoint.
func NewHTTPOCREClient() *HTTPOCREClient {
	return &HTTPOCREClient{
		baseURL: defaultOCREBaseURL,
		client:  &http.Client{Timeout: ocreRequestTimeout},
	}
}

// NewHTTPOCREClientForTest creates an HTTPOCREClient pointed at an arbitrary
// base URL (e.g. an httptest.Server), so tests never reach nomisma.org.
func NewHTTPOCREClientForTest(baseURL string) *HTTPOCREClient {
	return &HTTPOCREClient{
		baseURL: baseURL,
		client:  &http.Client{Timeout: ocreRequestTimeout},
	}
}

// Search builds the fixed-template SPARQL query for params, GETs it, parses
// the SPARQL-results JSON, scores/de-dups/ranks the rows, and re-validates
// every surviving candidate's host == numismatics.org before returning. It
// never propagates a raw transport/parse error — only a typed OCREErrorKind.
func (c *HTTPOCREClient) Search(ctx context.Context, params OCREQueryParams, limit int) ([]OCRECandidate, OCREErrorKind, error) {
	if !params.HasSignal() {
		return nil, OCREErrorInvalidRequest, errors.New("no type-bearing OCRE query signal")
	}
	if limit > 0 {
		params.Limit = limit
	}

	query := params.BuildQuery()
	reqURL := c.baseURL + "?" + url.Values{"query": {query}}.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, OCREErrorInvalidRequest, err
	}
	req.Header.Set("Accept", ocreAcceptSPARQLJSON)
	req.Header.Set("User-Agent", ocreUserAgent)

	resp, err := c.client.Do(req)
	if err != nil {
		if errors.Is(ctx.Err(), context.Canceled) {
			return nil, OCREErrorCancelled, err
		}
		return nil, OCREErrorUnavailable, err
	}
	defer resp.Body.Close()

	body, err := readLimitedOCREBody(resp.Body)
	if err != nil {
		return nil, OCREErrorInvalidResponse, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, OCREErrorUnavailable, errors.New("OCRE/Nomisma returned status " + strconv.Itoa(resp.StatusCode))
	}

	rows, err := ParseOCRESPARQLResults(body)
	if err != nil {
		return nil, OCREErrorInvalidResponse, err
	}

	candidates := Score(params, rows)

	// Defense in depth: re-validate every candidate host == numismatics.org
	// after scoring, so a compromised scorer/parser can never surface an
	// off-host citation (FR-011).
	validated := candidates[:0]
	for _, candidate := range candidates {
		if ocreHostIsCanonical(candidate.TypeURI) {
			validated = append(validated, candidate)
		}
	}
	if len(validated) == 0 {
		return []OCRECandidate{}, OCREErrorNoMatch, nil
	}
	return validated, "", nil
}

// ocreHostIsCanonical reports whether uri is an absolute URL whose host is
// exactly numismatics.org.
func ocreHostIsCanonical(uri string) bool {
	parsed, err := url.Parse(strings.TrimSpace(uri))
	if err != nil {
		return false
	}
	return strings.EqualFold(parsed.Hostname(), ocreCanonicalHost)
}

func readLimitedOCREBody(body io.Reader) ([]byte, error) {
	limited := io.LimitReader(body, ocreResponseLimit+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if len(data) > ocreResponseLimit {
		return nil, errors.New("OCRE response exceeded limit")
	}
	return data, nil
}
