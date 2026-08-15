package services

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	defaultNomismaBaseURL  = "https://nomisma.org/apis/reconcile"
	nomismaRequestTimeout  = 8 * time.Second
	nomismaResponseLimit   = 1 << 20
	nomismaDefaultLimit    = 5
	nomismaMaxQueryLength  = 200
	nomismaQueryIdentifier = "q1"
)

// NomismaErrorKind distinguishes the outcomes a Nomisma search can produce,
// so callers never have to string-match a raw error.
type NomismaErrorKind string

const (
	NomismaErrorUnavailable     NomismaErrorKind = "unavailable"
	NomismaErrorNoMatch         NomismaErrorKind = "no_match"
	NomismaErrorInvalidResponse NomismaErrorKind = "invalid_response"
	NomismaErrorInvalidRequest  NomismaErrorKind = "invalid_request"
	NomismaErrorCancelled       NomismaErrorKind = "cancelled"
)

// NomismaCandidate is a single reconciliation result for the admin to
// review. It is transient (never persisted) and is also the frontend-facing
// DTO returned by the search handler.
type NomismaCandidate struct {
	URI   string  `json:"uri"`
	Label string  `json:"label"`
	Score float64 `json:"score"`
	Match bool    `json:"match"`
}

// NomismaClient is the single HTTP boundary for Nomisma.org's public
// OpenRefine-compatible reconciliation service. Never called from handlers
// or repositories directly - only from MintLocationService.
type NomismaClient interface {
	// Search returns up to limit candidates for query. A returned
	// NomismaErrorKind of "" means success (candidates may still be empty,
	// meaning no_match at the caller's discretion); any non-empty kind
	// means the caller should treat this as a non-blocking, typed outcome
	// rather than a hard failure.
	Search(ctx context.Context, query string, limit int) ([]NomismaCandidate, NomismaErrorKind, error)
}

// HTTPNomismaClient implements NomismaClient over stdlib net/http against
// Nomisma's reconciliation endpoint. No credential is required or stored.
type HTTPNomismaClient struct {
	baseURL string
	client  *http.Client
}

// NewHTTPNomismaClient creates an HTTPNomismaClient pointed at the public
// Nomisma reconciliation endpoint.
func NewHTTPNomismaClient() *HTTPNomismaClient {
	return &HTTPNomismaClient{
		baseURL: defaultNomismaBaseURL,
		client:  &http.Client{Timeout: nomismaRequestTimeout},
	}
}

// NewHTTPNomismaClientForTest creates an HTTPNomismaClient pointed at an
// arbitrary base URL (e.g. an httptest.Server), for tests that must never
// reach the real nomisma.org host.
func NewHTTPNomismaClientForTest(baseURL string) *HTTPNomismaClient {
	return &HTTPNomismaClient{
		baseURL: baseURL,
		client:  &http.Client{Timeout: nomismaRequestTimeout},
	}
}

// nomismaQuery is the value of a single entry in the "queries" URL
// parameter's JSON object. Per the live service (verified against
// https://nomisma.org/apis/reconcile), that parameter's value is the
// query-identifier map directly - e.g. {"q1":{"query":"Roma","limit":5}} -
// with NO outer "queries" wrapper key, despite that wrapper appearing in
// this project's own earlier (unverified) research notes.
type nomismaQuery struct {
	Query string `json:"query"`
	Limit int    `json:"limit,omitempty"`
}

// nomismaResultItem is a single candidate as the live service actually
// returns it. Two properties differ from the OpenRefine spec's usual
// shape and from this project's original (unverified) assumption:
//   - "id" is Nomisma's short local id (e.g. "roma"), not a full URI - the
//     durable concept URI must be constructed as nomismaIDBaseURI+id, the
//     same convention documented for nomisma.org's own getLabel/getRdf APIs.
//   - "match" is a JSON string ("true"/"false"), not a JSON boolean.
type nomismaResultItem struct {
	ID    string       `json:"id"`
	Name  string       `json:"name"`
	Score float64      `json:"score"`
	Match nomismaMatch `json:"match"`
	// "type" is present on the live response but unused here (not part of
	// this feature's contract); json.Unmarshal ignores unknown fields.
}

// nomismaMatch decodes Nomisma's "match" field, which the live service
// encodes as a JSON string ("true"/"false") rather than a JSON boolean.
type nomismaMatch bool

func (m *nomismaMatch) UnmarshalJSON(data []byte) error {
	var asString string
	if err := json.Unmarshal(data, &asString); err == nil {
		parsed, parseErr := strconv.ParseBool(asString)
		if parseErr != nil {
			return parseErr
		}
		*m = nomismaMatch(parsed)
		return nil
	}
	var asBool bool
	if err := json.Unmarshal(data, &asBool); err != nil {
		return err
	}
	*m = nomismaMatch(asBool)
	return nil
}

// nomismaResponsePayload is the live response envelope: a top-level map
// keyed by query identifier, with NO "results" wrapper - verified against
// https://nomisma.org/apis/reconcile?queries=%7B%22q1%22%3A%7B%22query%22%3A%22Roma%22%7D%7D,
// which returns {"q1":{"result":[...]}}, not {"results":{"q1":{...}}}.
type nomismaResponsePayload map[string]struct {
	Result []nomismaResultItem `json:"result"`
}

// nomismaIDBaseURI is prefixed to the live service's short "id" field to
// build the durable concept URI this project persists and validates
// (contracts/nomisma-authority-linking.md requires an absolute
// http/https URL under the nomisma.org host). Matches the convention
// documented for nomisma.org's own getLabel/getRdf REST helpers
// (e.g. "apis/getLabel?uri=http://nomisma.org/id/ar").
const nomismaIDBaseURI = "http://nomisma.org/id/"

// Search calls Nomisma's reconciliation endpoint with the exact
// admin-typed query text. It never propagates a raw transport/parsing
// error - every failure mode collapses to a typed NomismaErrorKind so the
// service/handler layer can surface a non-blocking "unavailable" outcome.
func (c *HTTPNomismaClient) Search(ctx context.Context, query string, limit int) ([]NomismaCandidate, NomismaErrorKind, error) {
	trimmed := strings.TrimSpace(query)
	if trimmed == "" || len([]rune(trimmed)) > nomismaMaxQueryLength {
		return nil, NomismaErrorInvalidRequest, errors.New("invalid Nomisma query")
	}
	if limit <= 0 {
		limit = nomismaDefaultLimit
	}

	queries := map[string]nomismaQuery{
		nomismaQueryIdentifier: {Query: trimmed, Limit: limit},
	}
	encodedPayload, err := json.Marshal(queries)
	if err != nil {
		return nil, NomismaErrorInvalidRequest, err
	}

	reqURL := c.baseURL + "?" + url.Values{"queries": {string(encodedPayload)}}.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, NomismaErrorInvalidRequest, err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		if errors.Is(ctx.Err(), context.Canceled) {
			return nil, NomismaErrorCancelled, err
		}
		return nil, NomismaErrorUnavailable, err
	}
	defer resp.Body.Close()

	body, err := readLimitedNomismaBody(resp.Body)
	if err != nil {
		return nil, NomismaErrorInvalidResponse, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, NomismaErrorUnavailable, errors.New("Nomisma returned status " + strconv.Itoa(resp.StatusCode))
	}

	var parsed nomismaResponsePayload
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, NomismaErrorInvalidResponse, err
	}

	result, ok := parsed[nomismaQueryIdentifier]
	if !ok {
		return []NomismaCandidate{}, NomismaErrorNoMatch, nil
	}

	candidates := make([]NomismaCandidate, 0, len(result.Result))
	for _, item := range result.Result {
		trimmedID := strings.TrimSpace(item.ID)
		if trimmedID == "" || strings.TrimSpace(item.Name) == "" {
			continue
		}
		candidates = append(candidates, NomismaCandidate{
			URI:   nomismaIDBaseURI + trimmedID,
			Label: item.Name,
			Score: item.Score,
			Match: bool(item.Match),
		})
	}
	if len(candidates) == 0 {
		return []NomismaCandidate{}, NomismaErrorNoMatch, nil
	}
	return candidates, "", nil
}

func readLimitedNomismaBody(body io.Reader) ([]byte, error) {
	limited := io.LimitReader(body, nomismaResponseLimit+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if len(data) > nomismaResponseLimit {
		return nil, errors.New("Nomisma response exceeded limit")
	}
	return data, nil
}
