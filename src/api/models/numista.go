package models

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	NumistaMaxQueryLength       = 500
	NumistaMaxCandidateCount    = 50
	NumistaScoringVersion       = "numista-v1"
	NumistaCanonicalURLTemplate = "https://en.numista.com/catalogue/pieces%d.html"
)

type NumistaLookupPath string

const (
	NumistaLookupPathDirect NumistaLookupPath = "direct"
	NumistaLookupPathPhoto  NumistaLookupPath = "photo"
)

type NumistaLookupStatus string

const (
	NumistaStatusSuccess      NumistaLookupStatus = "success"
	NumistaStatusEmpty        NumistaLookupStatus = "empty"
	NumistaStatusUnconfigured NumistaLookupStatus = "unconfigured"
	NumistaStatusQuotaLimited NumistaLookupStatus = "quota-limited"
	NumistaStatusTimeout      NumistaLookupStatus = "timeout"
	NumistaStatusUnavailable  NumistaLookupStatus = "unavailable"
)

type NumistaEnrichmentState string

const (
	NumistaEnrichmentNotRequested NumistaEnrichmentState = "not_requested"
	NumistaEnrichmentEnriched     NumistaEnrichmentState = "enriched"
	NumistaEnrichmentCached       NumistaEnrichmentState = "cached"
	NumistaEnrichmentFailed       NumistaEnrichmentState = "failed"
)

type NumistaReasonKind string

const (
	NumistaReasonMatch       NumistaReasonKind = "match"
	NumistaReasonConflict    NumistaReasonKind = "conflict"
	NumistaReasonUnavailable NumistaReasonKind = "unavailable"
)

type NumistaEvidence struct {
	Title              string `json:"title,omitempty"`
	Issuer             string `json:"issuer,omitempty"`
	Denomination       string `json:"denomination,omitempty"`
	Mint               string `json:"mint,omitempty"`
	DateText           string `json:"dateText,omitempty"`
	Material           string `json:"material,omitempty"`
	ObverseInscription string `json:"obverseInscription,omitempty"`
	ReverseInscription string `json:"reverseInscription,omitempty"`
	VisibleText        string `json:"visibleText,omitempty"`
	ExactNumistaID     *int   `json:"exactNumistaId,omitempty"`
}

type NumistaLookupRequest struct {
	Query    string            `json:"query"`
	Path     NumistaLookupPath `json:"path"`
	Evidence NumistaEvidence   `json:"evidence"`
}

type NumistaRelevanceReason struct {
	Field string            `json:"field"`
	Kind  NumistaReasonKind `json:"kind"`
	Code  string            `json:"code"`
	Label string            `json:"label"`
}

type NumistaRelevanceAssessment struct {
	ScoringVersion string                   `json:"scoringVersion"`
	Score          int                      `json:"score"`
	Band           string                   `json:"band"`
	Reasons        []NumistaRelevanceReason `json:"reasons"`
}

type NumistaCandidate struct {
	ID                 int                        `json:"id"`
	CanonicalURL       string                     `json:"canonicalUrl"`
	Title              string                     `json:"title"`
	Issuer             string                     `json:"issuer,omitempty"`
	Denomination       string                     `json:"denomination,omitempty"`
	Mint               string                     `json:"mint,omitempty"`
	MinYear            *int                       `json:"minYear,omitempty"`
	MaxYear            *int                       `json:"maxYear,omitempty"`
	YearDisplay        string                     `json:"yearDisplay,omitempty"`
	Material           string                     `json:"material,omitempty"`
	ObverseInscription string                     `json:"obverseInscription,omitempty"`
	ReverseInscription string                     `json:"reverseInscription,omitempty"`
	ObverseThumbnail   string                     `json:"obverseThumbnail,omitempty"`
	ReverseThumbnail   string                     `json:"reverseThumbnail,omitempty"`
	ProviderPosition   int                        `json:"providerPosition"`
	EnrichmentState    NumistaEnrichmentState     `json:"enrichmentState"`
	Assessment         NumistaRelevanceAssessment `json:"assessment"`
}

type NumistaCacheMetadata struct {
	Hit        bool      `json:"hit"`
	CreatedAt  time.Time `json:"createdAt"`
	ExpiresAt  time.Time `json:"expiresAt"`
	AgeSeconds int64     `json:"ageSeconds"`
}

type NumistaLookupOutcome struct {
	Status            NumistaLookupStatus   `json:"status"`
	EffectiveQuery    string                `json:"effectiveQuery"`
	Candidates        []NumistaCandidate    `json:"candidates"`
	GuidanceCode      string                `json:"guidanceCode,omitempty"`
	RetryAfterSeconds *int                  `json:"retryAfterSeconds,omitempty"`
	Cache             *NumistaCacheMetadata `json:"cache,omitempty"`
	Stage             string                `json:"stage"`
}

type NumistaEnrichmentRequest struct {
	NumistaLookupRequest
	Candidates []NumistaCandidate `json:"candidates"`
}

type SelectedNumistaReference struct {
	Catalog string `json:"catalog"`
	Number  string `json:"number"`
	URI     string `json:"uri"`
}

type LegacyNumistaIssuer struct {
	Name string `json:"name"`
}

type LegacyNumistaType struct {
	ID               int                  `json:"id"`
	Title            string               `json:"title"`
	Issuer           *LegacyNumistaIssuer `json:"issuer,omitempty"`
	MinYear          *int                 `json:"min_year,omitempty"`
	MaxYear          *int                 `json:"max_year,omitempty"`
	ObverseThumbnail string               `json:"obverse_thumbnail,omitempty"`
	ReverseThumbnail string               `json:"reverse_thumbnail,omitempty"`
}

type LegacyNumistaSearchResponse struct {
	Count int                 `json:"count"`
	Types []LegacyNumistaType `json:"types"`
}

type NumistaHealthSummary struct {
	Configured            bool                        `json:"configured"`
	ConfigurationValid    bool                        `json:"configurationValid"`
	LastOutcome           NumistaLookupStatus         `json:"lastOutcome,omitempty"`
	LastCheckedAt         *time.Time                  `json:"lastCheckedAt,omitempty"`
	StatusCounts          map[NumistaLookupStatus]int `json:"statusCounts"`
	BroadRequestCount     int                         `json:"broadRequestCount"`
	DetailRequestCount    int                         `json:"detailRequestCount"`
	CacheHitCount         int                         `json:"cacheHitCount"`
	CacheRefreshCount     int                         `json:"cacheRefreshCount"`
	CacheHitRate          float64                     `json:"cacheHitRate"`
	P50ElapsedMs          int64                       `json:"p50ElapsedMs"`
	P95ElapsedMs          int64                       `json:"p95ElapsedMs"`
	EnrichmentAttempted   int                         `json:"enrichmentAttempted"`
	EnrichmentSucceeded   int                         `json:"enrichmentSucceeded"`
	EnrichmentFailed      int                         `json:"enrichmentFailed"`
	LastQuotaLimitedAt    *time.Time                  `json:"lastQuotaLimitedAt,omitempty"`
	LastRetryAfterSeconds *int                        `json:"lastRetryAfterSeconds,omitempty"`
}

func (r *NumistaLookupRequest) Validate() error {
	if strings.TrimSpace(r.Query) == "" || len([]rune(r.Query)) > NumistaMaxQueryLength {
		return errors.New("query must contain 1 to 500 characters")
	}
	if r.Path != NumistaLookupPathDirect && r.Path != NumistaLookupPathPhoto {
		return errors.New("path must be direct or photo")
	}
	return r.Evidence.Validate()
}

func (e NumistaEvidence) Validate() error {
	bounds := []struct {
		name  string
		value string
		max   int
	}{
		{"title", e.Title, 200}, {"issuer", e.Issuer, 200}, {"denomination", e.Denomination, 100},
		{"mint", e.Mint, 200}, {"dateText", e.DateText, 100}, {"material", e.Material, 100},
		{"obverseInscription", e.ObverseInscription, 500}, {"reverseInscription", e.ReverseInscription, 500},
		{"visibleText", e.VisibleText, 500},
	}
	for _, bound := range bounds {
		if len([]rune(strings.TrimSpace(bound.value))) > bound.max {
			return fmt.Errorf("%s exceeds %d characters", bound.name, bound.max)
		}
	}
	if e.ExactNumistaID != nil && *e.ExactNumistaID <= 0 {
		return errors.New("exactNumistaId must be positive")
	}
	return nil
}

func (r *NumistaEnrichmentRequest) Validate() error {
	if err := r.NumistaLookupRequest.Validate(); err != nil {
		return err
	}
	if len(r.Candidates) < 1 || len(r.Candidates) > NumistaMaxCandidateCount {
		return errors.New("candidates must contain 1 to 50 items")
	}
	seen := make(map[int]struct{}, len(r.Candidates))
	for _, candidate := range r.Candidates {
		if err := candidate.Validate(); err != nil {
			return err
		}
		if _, ok := seen[candidate.ID]; ok {
			return errors.New("candidate IDs must be unique")
		}
		seen[candidate.ID] = struct{}{}
	}
	return nil
}

func CanonicalNumistaURL(id int) (string, error) {
	if id <= 0 {
		return "", errors.New("Numista ID must be positive")
	}
	return fmt.Sprintf(NumistaCanonicalURLTemplate, id), nil
}

func NewSelectedNumistaReference(id int) (SelectedNumistaReference, error) {
	uri, err := CanonicalNumistaURL(id)
	if err != nil {
		return SelectedNumistaReference{}, err
	}
	return SelectedNumistaReference{Catalog: "Numista", Number: strconv.Itoa(id), URI: uri}, nil
}

func ParseSelectedNumistaReference(number, uri string) (SelectedNumistaReference, error) {
	id, err := strconv.Atoi(strings.TrimSpace(number))
	if err != nil || id <= 0 {
		return SelectedNumistaReference{}, errors.New("number must be a positive integer")
	}
	ref, err := NewSelectedNumistaReference(id)
	if err != nil {
		return SelectedNumistaReference{}, err
	}
	if strings.TrimSpace(uri) == "" || strings.TrimSpace(uri) != ref.URI {
		return SelectedNumistaReference{}, errors.New("uri must match the canonical Numista URL")
	}
	return ref, nil
}

func (r SelectedNumistaReference) Validate() error {
	if !strings.EqualFold(strings.TrimSpace(r.Catalog), "Numista") {
		return errors.New("catalog must be Numista")
	}
	id, err := strconv.Atoi(strings.TrimSpace(r.Number))
	if err != nil || id <= 0 {
		return errors.New("number must be a positive integer")
	}
	canonical, _ := CanonicalNumistaURL(id)
	parsed, err := url.ParseRequestURI(strings.TrimSpace(r.URI))
	if err != nil || parsed.String() != canonical {
		return errors.New("uri must match the canonical Numista URL")
	}
	return nil
}

func (c NumistaCandidate) Validate() error {
	if c.ID <= 0 || strings.TrimSpace(c.Title) == "" {
		return errors.New("candidate requires a positive ID and title")
	}
	canonical, _ := CanonicalNumistaURL(c.ID)
	if c.CanonicalURL != canonical {
		return errors.New("candidate canonicalUrl is invalid")
	}
	switch c.EnrichmentState {
	case NumistaEnrichmentNotRequested, NumistaEnrichmentEnriched, NumistaEnrichmentCached, NumistaEnrichmentFailed:
	default:
		return errors.New("candidate enrichmentState is invalid")
	}
	if c.MinYear != nil && c.MaxYear != nil && *c.MinYear > *c.MaxYear {
		return errors.New("candidate year range is invalid")
	}
	if c.ProviderPosition < 0 {
		return errors.New("candidate providerPosition is invalid")
	}
	for _, thumbnail := range []string{c.ObverseThumbnail, c.ReverseThumbnail} {
		if thumbnail == "" {
			continue
		}
		parsed, err := url.ParseRequestURI(thumbnail)
		if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
			return errors.New("candidate thumbnail URL is invalid")
		}
	}
	return c.Assessment.Validate()
}

func (a NumistaRelevanceAssessment) Validate() error {
	if a.ScoringVersion != NumistaScoringVersion || a.Score < 0 || a.Score > 100 {
		return errors.New("relevance assessment is invalid")
	}
	if a.Band != "strong" && a.Band != "possible" && a.Band != "weak" {
		return errors.New("relevance band is invalid")
	}
	if a.Band != numistaRelevanceBand(a.Score) {
		return errors.New("relevance score and band are inconsistent")
	}
	if a.Reasons == nil {
		return errors.New("relevance reasons must be present")
	}
	validFields := map[string]bool{
		"exact_id": true, "title": true, "issuer": true, "denomination": true,
		"mint": true, "date": true, "material": true, "inscription": true,
	}
	for _, reason := range a.Reasons {
		if !validFields[reason.Field] || strings.TrimSpace(reason.Code) == "" || strings.TrimSpace(reason.Label) == "" {
			return errors.New("relevance reason is invalid")
		}
		if reason.Kind != NumistaReasonMatch && reason.Kind != NumistaReasonConflict && reason.Kind != NumistaReasonUnavailable {
			return errors.New("relevance reason kind is invalid")
		}
	}
	return nil
}

func numistaRelevanceBand(score int) string {
	if score >= 80 {
		return "strong"
	}
	if score >= 60 {
		return "possible"
	}
	return "weak"
}

func (m NumistaCacheMetadata) Validate() error {
	if m.CreatedAt.IsZero() || !m.ExpiresAt.After(m.CreatedAt) || m.AgeSeconds < 0 {
		return errors.New("cache metadata is invalid")
	}
	return nil
}

func (o NumistaLookupOutcome) Validate() error {
	switch o.Status {
	case NumistaStatusSuccess, NumistaStatusEmpty, NumistaStatusUnconfigured,
		NumistaStatusQuotaLimited, NumistaStatusTimeout, NumistaStatusUnavailable:
	default:
		return errors.New("lookup status is invalid")
	}
	if strings.TrimSpace(o.EffectiveQuery) == "" || (o.Stage != "broad" && o.Stage != "enriched") {
		return errors.New("lookup outcome is invalid")
	}
	if o.Candidates == nil {
		return errors.New("lookup candidates must be present")
	}
	if o.Status == NumistaStatusSuccess && len(o.Candidates) == 0 {
		return errors.New("successful lookup must contain candidates")
	}
	if o.Status != NumistaStatusSuccess && len(o.Candidates) != 0 {
		return errors.New("non-success lookup cannot contain candidates")
	}
	if o.RetryAfterSeconds != nil && *o.RetryAfterSeconds <= 0 {
		return errors.New("retryAfterSeconds must be positive")
	}
	if o.Cache != nil && o.Status != NumistaStatusSuccess && o.Status != NumistaStatusEmpty {
		return errors.New("cache metadata is only valid for successful or empty lookups")
	}
	for _, candidate := range o.Candidates {
		if err := candidate.Validate(); err != nil {
			return err
		}
	}
	if o.Cache != nil {
		return o.Cache.Validate()
	}
	return nil
}

func (h NumistaHealthSummary) Validate() error {
	if h.StatusCounts == nil || h.BroadRequestCount < 0 || h.DetailRequestCount < 0 ||
		h.CacheHitCount < 0 || h.CacheRefreshCount < 0 || h.CacheHitRate < 0 || h.CacheHitRate > 1 ||
		h.P50ElapsedMs < 0 || h.P95ElapsedMs < 0 || h.EnrichmentAttempted < 0 ||
		h.EnrichmentSucceeded < 0 || h.EnrichmentFailed < 0 {
		return errors.New("health summary is invalid")
	}
	if h.LastOutcome != "" && !validNumistaLookupStatus(h.LastOutcome) {
		return errors.New("health summary last outcome is invalid")
	}
	for status, count := range h.StatusCounts {
		if !validNumistaLookupStatus(status) || count < 0 {
			return errors.New("health summary status count is invalid")
		}
	}
	if h.LastCheckedAt != nil && h.LastCheckedAt.IsZero() ||
		h.LastQuotaLimitedAt != nil && h.LastQuotaLimitedAt.IsZero() ||
		h.LastRetryAfterSeconds != nil && *h.LastRetryAfterSeconds <= 0 {
		return errors.New("health summary optional metadata is invalid")
	}
	return nil
}

func validNumistaLookupStatus(status NumistaLookupStatus) bool {
	switch status {
	case NumistaStatusSuccess, NumistaStatusEmpty, NumistaStatusUnconfigured,
		NumistaStatusQuotaLimited, NumistaStatusTimeout, NumistaStatusUnavailable:
		return true
	default:
		return false
	}
}
