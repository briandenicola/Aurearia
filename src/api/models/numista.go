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
	NumistaMaxQueryLength         = 500
	NumistaMaxCandidateCount      = 50
	NumistaMaxID                  = 2147483647
	NumistaScoringVersion         = "numista-v1"
	NumistaQueryGenerationVersion = "numista-query-v2"
	NumistaCanonicalURLTemplate   = "https://en.numista.com/catalogue/pieces%d.html"
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

type NumistaQuerySource string

const (
	NumistaQuerySourceGenerated  NumistaQuerySource = "generated"
	NumistaQuerySourceUserEdited NumistaQuerySource = "user-edited"
	NumistaQuerySourceManual     NumistaQuerySource = "manual"
)

type NumistaSearchAttempt string

const (
	NumistaSearchAttemptPrimary NumistaSearchAttempt = "primary"
	NumistaSearchAttemptRelaxed NumistaSearchAttempt = "relaxed"
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
	ReverseType        string `json:"reverseType,omitempty"`
	VisibleText        string `json:"visibleText,omitempty"`
	ExactNumistaID     *int   `json:"exactNumistaId,omitempty"`
}

type NumistaLookupRequest struct {
	Query             string             `json:"query" binding:"required"`
	Path              NumistaLookupPath  `json:"path" binding:"required"`
	Evidence          NumistaEvidence    `json:"evidence" binding:"required"`
	QuerySource       NumistaQuerySource `json:"querySource" binding:"required" enums:"generated,user-edited,manual"`
	GenerationVersion string             `json:"generationVersion,omitempty" enums:"numista-query-v2"`
}

type NumistaQueryProposalRequest struct {
	Path     NumistaLookupPath `json:"path" binding:"required"`
	Evidence NumistaEvidence   `json:"evidence" binding:"required"`
}

type NumistaQueryProposal struct {
	Query             string             `json:"query" binding:"required"`
	QuerySource       NumistaQuerySource `json:"querySource" binding:"required" enums:"generated"`
	GenerationVersion string             `json:"generationVersion" binding:"required" enums:"numista-query-v2"`
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
	Coalesced  bool      `json:"coalesced"`
	CreatedAt  time.Time `json:"createdAt"`
	ExpiresAt  time.Time `json:"expiresAt"`
	AgeSeconds int64     `json:"ageSeconds"`
}

type NumistaLookupOutcome struct {
	Status             NumistaLookupStatus   `json:"status" binding:"required"`
	EffectiveQuery     string                `json:"effectiveQuery" binding:"required"`
	Candidates         []NumistaCandidate    `json:"candidates" binding:"required"`
	GuidanceCode       string                `json:"guidanceCode,omitempty"`
	RetryAfterSeconds  *int                  `json:"retryAfterSeconds,omitempty"`
	Cache              *NumistaCacheMetadata `json:"cache,omitempty"`
	Stage              string                `json:"stage" binding:"required"`
	QuerySource        NumistaQuerySource    `json:"querySource" binding:"required"`
	SearchAttempt      NumistaSearchAttempt  `json:"searchAttempt" binding:"required"`
	SearchAttemptCount int                   `json:"searchAttemptCount" binding:"required"`
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
	StatusCounts          map[NumistaLookupStatus]int `json:"statusCounts"` // Sparse rolling counts; absent statuses have zero events.
	BroadRequestCount     int                         `json:"broadRequestCount"`
	DetailRequestCount    int                         `json:"detailRequestCount"`
	FreshCacheHitCount    int                         `json:"freshCacheHitCount"`
	CoalescedRequestCount int                         `json:"coalescedRequestCount"`
	ProviderLoadCount     int                         `json:"providerLoadCount"`
	ProviderFailureCount  int                         `json:"providerFailureCount"`
	CancelledRequestCount int                         `json:"cancelledRequestCount"`
	FreshCacheHitRate     float64                     `json:"freshCacheHitRate"`
	P50ElapsedMs          int64                       `json:"p50ElapsedMs"` // R-7 linearly interpolated percentile, rounded to milliseconds.
	P95ElapsedMs          int64                       `json:"p95ElapsedMs"` // R-7 linearly interpolated percentile, rounded to milliseconds.
	EnrichmentAttempted   int                         `json:"enrichmentAttempted"`
	EnrichmentSucceeded   int                         `json:"enrichmentSucceeded"`
	EnrichmentFailed      int                         `json:"enrichmentFailed"`
	GeneratedQueryCount   int                         `json:"generatedQueryCount,omitempty"`
	UserEditedQueryCount  int                         `json:"userEditedQueryCount,omitempty"`
	ManualQueryCount      int                         `json:"manualQueryCount,omitempty"`
	RelaxedAttemptCount   int                         `json:"relaxedAttemptCount,omitempty"`
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
	if r.QuerySource == "" {
		r.QuerySource = NumistaQuerySourceManual
	}
	switch r.QuerySource {
	case NumistaQuerySourceGenerated, NumistaQuerySourceUserEdited, NumistaQuerySourceManual:
	default:
		return errors.New("querySource must be generated, user-edited, or manual")
	}
	switch r.QuerySource {
	case NumistaQuerySourceGenerated, NumistaQuerySourceUserEdited:
		if r.GenerationVersion != NumistaQueryGenerationVersion {
			return errors.New("generationVersion must be numista-query-v2 for generated or user-edited queries")
		}
	case NumistaQuerySourceManual:
		if r.GenerationVersion != "" {
			return errors.New("generationVersion must be omitted for manual queries")
		}
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
		{"reverseType", e.ReverseType, 500},
		{"visibleText", e.VisibleText, 500},
	}

	for _, bound := range bounds {
		if len([]rune(strings.TrimSpace(bound.value))) > bound.max {
			return fmt.Errorf("%s exceeds %d characters", bound.name, bound.max)
		}
	}
	if e.ExactNumistaID != nil && (*e.ExactNumistaID <= 0 || *e.ExactNumistaID > NumistaMaxID) {
		return errors.New("exactNumistaId must be between 1 and 2147483647")
	}
	return nil
}

func (r NumistaQueryProposalRequest) Validate() error {
	if r.Path != NumistaLookupPathDirect && r.Path != NumistaLookupPathPhoto {
		return errors.New("path must be direct or photo")
	}
	return r.Evidence.Validate()
}

func (r *NumistaEnrichmentRequest) Validate() error {
	r.Query = strings.TrimSpace(r.Query)
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
	if id <= 0 || id > NumistaMaxID {
		return "", errors.New("Numista ID must be between 1 and 2147483647")
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
	if c.ID <= 0 || c.ID > NumistaMaxID || strings.TrimSpace(c.Title) == "" {
		return errors.New("candidate requires an ID between 1 and 2147483647 and a title")
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
	if m.Hit && m.Coalesced ||
		m.CreatedAt.IsZero() || !m.ExpiresAt.After(m.CreatedAt) || m.AgeSeconds < 0 {
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
	legacyAttribution := o.QuerySource == "" && o.SearchAttempt == "" && o.SearchAttemptCount == 0
	if !legacyAttribution {
		switch o.QuerySource {
		case NumistaQuerySourceGenerated, NumistaQuerySourceUserEdited, NumistaQuerySourceManual:
		default:
			return errors.New("lookup query source is invalid")
		}
		if o.SearchAttempt != NumistaSearchAttemptPrimary && o.SearchAttempt != NumistaSearchAttemptRelaxed {
			return errors.New("lookup search attempt is invalid")
		}
		if o.SearchAttemptCount < 1 || o.SearchAttemptCount > 2 ||
			(o.SearchAttempt == NumistaSearchAttemptPrimary && o.SearchAttemptCount != 1) ||
			(o.SearchAttempt == NumistaSearchAttemptRelaxed && o.SearchAttemptCount != 2) {
			return errors.New("lookup search attempt count is invalid")
		}
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
		h.FreshCacheHitCount < 0 || h.CoalescedRequestCount < 0 ||
		h.ProviderLoadCount < 0 || h.ProviderFailureCount < 0 || h.CancelledRequestCount < 0 ||
		h.FreshCacheHitRate < 0 || h.FreshCacheHitRate > 1 ||
		h.P50ElapsedMs < 0 || h.P95ElapsedMs < 0 || h.EnrichmentAttempted < 0 ||
		h.EnrichmentSucceeded < 0 || h.EnrichmentFailed < 0 ||
		h.GeneratedQueryCount < 0 || h.UserEditedQueryCount < 0 ||
		h.ManualQueryCount < 0 || h.RelaxedAttemptCount < 0 {
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
