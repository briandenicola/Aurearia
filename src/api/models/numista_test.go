package models

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func intPointer(value int) *int { return &value }

func validNumistaTestCandidate(id int) NumistaCandidate {
	canonical, _ := CanonicalNumistaURL(id)
	return NumistaCandidate{
		ID: id, CanonicalURL: canonical, Title: "Coin", ProviderPosition: id - 1,
		EnrichmentState: NumistaEnrichmentNotRequested,
		Assessment: NumistaRelevanceAssessment{
			ScoringVersion: NumistaScoringVersion,
			Score:          50,
			Band:           "weak",
			Reasons:        []NumistaRelevanceReason{},
		},
	}
}

func validNumistaTestOutcome() NumistaLookupOutcome {
	return NumistaLookupOutcome{
		Status:         NumistaStatusSuccess,
		EffectiveQuery: "coin",
		Candidates:     []NumistaCandidate{validNumistaTestCandidate(1)},
		Stage:          "broad",
	}
}

func TestNumistaLookupRequestPathsQueryBoundsAndJSONContract(t *testing.T) {
	for _, path := range []NumistaLookupPath{NumistaLookupPathDirect, NumistaLookupPathPhoto} {
		t.Run(string(path), func(t *testing.T) {
			request := NumistaLookupRequest{
				Query: strings.Repeat("x", NumistaMaxQueryLength),
				Path:  path, Evidence: NumistaEvidence{},
			}
			if err := request.Validate(); err != nil {
				t.Fatalf("Validate() error = %v", err)
			}
			data, err := json.Marshal(request)
			if err != nil {
				t.Fatal(err)
			}
			for _, field := range []string{`"query"`, `"path"`, `"evidence"`} {
				if !strings.Contains(string(data), field) {
					t.Fatalf("JSON contract missing %s: %s", field, data)
				}
			}
		})
	}

	for _, test := range []struct {
		name    string
		request NumistaLookupRequest
	}{
		{"blank query", NumistaLookupRequest{Query: " \t\r\n ", Path: NumistaLookupPathDirect}},
		{"query over maximum", NumistaLookupRequest{Query: strings.Repeat("x", NumistaMaxQueryLength+1), Path: NumistaLookupPathDirect}},
		{"missing path", NumistaLookupRequest{Query: "coin"}},
		{"unknown path", NumistaLookupRequest{Query: "coin", Path: "other"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := test.request.Validate(); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}

	const query = "  Trajan   denarius \t Rome  "
	request := NumistaLookupRequest{Query: query, Path: NumistaLookupPathDirect}
	if err := request.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if request.Query != query {
		t.Fatalf("query = %q, want byte-for-byte %q", request.Query, query)
	}
}

func TestNumistaLookupRequestGenerationVersionInvariant(t *testing.T) {
	tests := []struct {
		name    string
		source  NumistaQuerySource
		version string
		wantErr bool
	}{
		{name: "generated exact version", source: NumistaQuerySourceGenerated, version: NumistaQueryGenerationVersion},
		{name: "generated missing version", source: NumistaQuerySourceGenerated, wantErr: true},
		{name: "generated unknown version", source: NumistaQuerySourceGenerated, version: "numista-query-v1", wantErr: true},
		{name: "user edited exact version", source: NumistaQuerySourceUserEdited, version: NumistaQueryGenerationVersion},
		{name: "user edited missing version", source: NumistaQuerySourceUserEdited, wantErr: true},
		{name: "user edited unknown version", source: NumistaQuerySourceUserEdited, version: "numista-query-v3", wantErr: true},
		{name: "manual omitted version", source: NumistaQuerySourceManual},
		{name: "manual supplied version", source: NumistaQuerySourceManual, version: NumistaQueryGenerationVersion, wantErr: true},
		{name: "legacy missing source remains manual", source: "", version: ""},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := NumistaLookupRequest{
				Query:             "coin",
				Path:              NumistaLookupPathDirect,
				QuerySource:       test.source,
				GenerationVersion: test.version,
			}
			err := request.Validate()
			if (err != nil) != test.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, test.wantErr)
			}
			if test.source == "" && request.QuerySource != NumistaQuerySourceManual {
				t.Fatalf("legacy source = %q, want manual", request.QuerySource)
			}
		})
	}
}

func TestNumistaEvidenceFieldBounds(t *testing.T) {
	fields := []struct {
		name string
		max  int
		set  func(*NumistaEvidence, string)
	}{
		{"title", 200, func(e *NumistaEvidence, value string) { e.Title = value }},
		{"issuer", 200, func(e *NumistaEvidence, value string) { e.Issuer = value }},
		{"denomination", 100, func(e *NumistaEvidence, value string) { e.Denomination = value }},
		{"mint", 200, func(e *NumistaEvidence, value string) { e.Mint = value }},
		{"dateText", 100, func(e *NumistaEvidence, value string) { e.DateText = value }},
		{"material", 100, func(e *NumistaEvidence, value string) { e.Material = value }},
		{"obverseInscription", 500, func(e *NumistaEvidence, value string) { e.ObverseInscription = value }},
		{"reverseInscription", 500, func(e *NumistaEvidence, value string) { e.ReverseInscription = value }},
		{"reverseType", 500, func(e *NumistaEvidence, value string) { e.ReverseType = value }},
		{"visibleText", 500, func(e *NumistaEvidence, value string) { e.VisibleText = value }},
	}

	for _, field := range fields {
		t.Run(field.name, func(t *testing.T) {
			var evidence NumistaEvidence
			field.set(&evidence, strings.Repeat("é", field.max))
			if err := evidence.Validate(); err != nil {
				t.Fatalf("exact maximum rejected: %v", err)
			}
			field.set(&evidence, strings.Repeat("é", field.max+1))
			if err := evidence.Validate(); err == nil {
				t.Fatal("value over maximum accepted")
			}
		})
	}

	for _, id := range []int{0, -1} {
		evidence := NumistaEvidence{ExactNumistaID: intPointer(id)}
		if err := evidence.Validate(); err == nil {
			t.Fatalf("exactNumistaId %d accepted", id)
		}
	}

	evidence := NumistaEvidence{ExactNumistaID: intPointer(1)}
	if err := evidence.Validate(); err != nil {
		t.Fatalf("positive exactNumistaId rejected: %v", err)
	}
}

func TestNumistaQueryProposalContracts(t *testing.T) {
	request := NumistaQueryProposalRequest{
		Path:     NumistaLookupPathDirect,
		Evidence: NumistaEvidence{ReverseType: "Victory standing left"},
	}
	if err := request.Validate(); err != nil {
		t.Fatal(err)
	}
	proposal := NumistaQueryProposal{
		Query:             "Constantine I VOT XX Nicomedia",
		QuerySource:       NumistaQuerySourceGenerated,
		GenerationVersion: NumistaQueryGenerationVersion,
	}
	data, err := json.Marshal(proposal)
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{`"query"`, `"querySource":"generated"`, `"generationVersion":"numista-query-v2"`} {
		if !strings.Contains(string(data), field) {
			t.Fatalf("proposal JSON missing %s: %s", field, data)
		}
	}
}

func TestNumistaEnrichmentRequestCandidateBounds(t *testing.T) {
	candidates := make([]NumistaCandidate, NumistaMaxCandidateCount)
	for i := range candidates {
		candidates[i] = validNumistaTestCandidate(i + 1)
	}
	for _, count := range []int{1, NumistaMaxCandidateCount} {
		request := NumistaEnrichmentRequest{
			NumistaLookupRequest: NumistaLookupRequest{Query: "coin", Path: NumistaLookupPathDirect},
			Candidates:           append([]NumistaCandidate(nil), candidates[:count]...),
		}
		if err := request.Validate(); err != nil {
			t.Fatalf("%d candidates rejected: %v", count, err)
		}
	}

	tooMany := append(append([]NumistaCandidate(nil), candidates...), validNumistaTestCandidate(51))
	duplicate := []NumistaCandidate{validNumistaTestCandidate(1), validNumistaTestCandidate(1)}
	invalidCandidate := []NumistaCandidate{validNumistaTestCandidate(1)}
	invalidCandidate[0].Title = ""
	for _, test := range []struct {
		name       string
		candidates []NumistaCandidate
	}{
		{"empty", []NumistaCandidate{}},
		{"over maximum", tooMany},
		{"duplicate IDs", duplicate},
		{"invalid candidate DTO", invalidCandidate},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := NumistaEnrichmentRequest{
				NumistaLookupRequest: NumistaLookupRequest{Query: "coin", Path: NumistaLookupPathDirect},
				Candidates:           test.candidates,
			}
			if err := request.Validate(); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestNumistaEnrichmentRequestTrimsOnlySurroundingWhitespace(t *testing.T) {
	candidate := validNumistaTestCandidate(1)
	request := NumistaEnrichmentRequest{
		NumistaLookupRequest: NumistaLookupRequest{
			Query: " \tTrajan   denarius\n",
			Path:  NumistaLookupPathDirect,
		},
		Candidates: []NumistaCandidate{candidate},
	}
	if err := request.Validate(); err != nil {
		t.Fatal(err)
	}
	if request.Query != "Trajan   denarius" {
		t.Fatalf("enrichment query=%q, want surrounding whitespace trimmed only", request.Query)
	}

	direct := NumistaLookupRequest{
		Query: " \tTrajan   denarius\n",
		Path:  NumistaLookupPathDirect,
	}
	if err := direct.Validate(); err != nil {
		t.Fatal(err)
	}
	if direct.Query != " \tTrajan   denarius\n" {
		t.Fatalf("direct lookup query changed to %q", direct.Query)
	}
}

func TestNumistaCandidateEnrichmentAndAssessmentVariants(t *testing.T) {
	for _, state := range []NumistaEnrichmentState{
		NumistaEnrichmentNotRequested, NumistaEnrichmentEnriched,
		NumistaEnrichmentCached, NumistaEnrichmentFailed,
	} {
		candidate := validNumistaTestCandidate(1)
		candidate.EnrichmentState = state
		candidate.ObverseThumbnail = "https://example.com/obverse.jpg"
		if err := candidate.Validate(); err != nil {
			t.Fatalf("state %q rejected: %v", state, err)
		}
	}

	minYear, maxYear := 2, 1
	for _, test := range []struct {
		name   string
		mutate func(*NumistaCandidate)
	}{
		{"nonpositive ID", func(c *NumistaCandidate) { c.ID = 0 }},
		{"blank title", func(c *NumistaCandidate) { c.Title = " " }},
		{"noncanonical URL", func(c *NumistaCandidate) { c.CanonicalURL = "https://example.com/1" }},
		{"invalid enrichment state", func(c *NumistaCandidate) { c.EnrichmentState = "unknown" }},
		{"reversed years", func(c *NumistaCandidate) { c.MinYear, c.MaxYear = &minYear, &maxYear }},
		{"negative provider position", func(c *NumistaCandidate) { c.ProviderPosition = -1 }},
		{"relative thumbnail", func(c *NumistaCandidate) { c.ObverseThumbnail = "/coin.jpg" }},
		{"insecure thumbnail", func(c *NumistaCandidate) { c.ReverseThumbnail = "http://example.com/coin.jpg" }},
		{"invalid assessment", func(c *NumistaCandidate) { c.Assessment.Score = 101 }},
	} {
		t.Run(test.name, func(t *testing.T) {
			candidate := validNumistaTestCandidate(1)
			test.mutate(&candidate)
			if err := candidate.Validate(); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}

	fields := []string{"exact_id", "title", "issuer", "denomination", "mint", "date", "material", "inscription"}
	kinds := []NumistaReasonKind{NumistaReasonMatch, NumistaReasonConflict, NumistaReasonUnavailable}
	for _, field := range fields {
		for _, kind := range kinds {
			assessment := validNumistaTestCandidate(1).Assessment
			assessment.Reasons = []NumistaRelevanceReason{{Field: field, Kind: kind, Code: "code", Label: "label"}}
			if err := assessment.Validate(); err != nil {
				t.Fatalf("reason field=%q kind=%q rejected: %v", field, kind, err)
			}
		}
	}
	for _, test := range []struct {
		score int
		band  string
	}{{0, "weak"}, {59, "weak"}, {60, "possible"}, {79, "possible"}, {80, "strong"}, {100, "strong"}} {
		assessment := validNumistaTestCandidate(1).Assessment
		assessment.Score, assessment.Band = test.score, test.band
		if err := assessment.Validate(); err != nil {
			t.Fatalf("score=%d band=%q rejected: %v", test.score, test.band, err)
		}
	}
	for _, test := range []struct {
		score int
		band  string
	}{
		{0, "possible"}, {59, "strong"}, {60, "weak"}, {79, "strong"},
		{80, "possible"}, {100, "weak"},
	} {
		assessment := validNumistaTestCandidate(1).Assessment
		assessment.Score, assessment.Band = test.score, test.band
		if err := assessment.Validate(); err == nil {
			t.Fatalf("contradictory score=%d band=%q accepted", test.score, test.band)
		}
	}

	for _, test := range []struct {
		name   string
		mutate func(*NumistaRelevanceAssessment)
	}{
		{"wrong scoring version", func(a *NumistaRelevanceAssessment) { a.ScoringVersion = "other" }},
		{"score below minimum", func(a *NumistaRelevanceAssessment) { a.Score = -1 }},
		{"score above maximum", func(a *NumistaRelevanceAssessment) { a.Score = 101 }},
		{"invalid band", func(a *NumistaRelevanceAssessment) { a.Band = "certain" }},
		{"nil reasons", func(a *NumistaRelevanceAssessment) { a.Reasons = nil }},
		{"invalid reason field", func(a *NumistaRelevanceAssessment) {
			a.Reasons = []NumistaRelevanceReason{{Field: "other", Kind: NumistaReasonMatch, Code: "code", Label: "label"}}
		}},
		{"invalid reason kind", func(a *NumistaRelevanceAssessment) {
			a.Reasons = []NumistaRelevanceReason{{Field: "title", Kind: "other", Code: "code", Label: "label"}}
		}},
		{"blank reason code", func(a *NumistaRelevanceAssessment) {
			a.Reasons = []NumistaRelevanceReason{{Field: "title", Kind: NumistaReasonMatch, Label: "label"}}
		}},
		{"blank reason label", func(a *NumistaRelevanceAssessment) {
			a.Reasons = []NumistaRelevanceReason{{Field: "title", Kind: NumistaReasonMatch, Code: "code"}}
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			assessment := validNumistaTestCandidate(1).Assessment
			test.mutate(&assessment)
			if err := assessment.Validate(); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestNumistaDTOJSONContractsAndOptionalFields(t *testing.T) {
	candidate := validNumistaTestCandidate(7)
	candidate.Assessment.Reasons = []NumistaRelevanceReason{{
		Field: "title", Kind: NumistaReasonMatch, Code: "title_match", Label: "Title supports this candidate",
	}}
	outcome := NumistaLookupOutcome{
		Status: NumistaStatusSuccess, EffectiveQuery: "coin",
		Candidates: []NumistaCandidate{candidate}, Stage: "broad",
	}
	data, err := json.Marshal(outcome)
	if err != nil {
		t.Fatal(err)
	}
	jsonText := string(data)
	for _, field := range []string{
		`"status"`, `"effectiveQuery"`, `"candidates"`, `"stage"`,
		`"id"`, `"canonicalUrl"`, `"title"`, `"providerPosition"`,
		`"enrichmentState"`, `"assessment"`, `"scoringVersion"`,
		`"score"`, `"band"`, `"reasons"`, `"field"`, `"kind"`, `"code"`, `"label"`,
	} {
		if !strings.Contains(jsonText, field) {
			t.Fatalf("JSON contract missing %s: %s", field, jsonText)
		}
	}
	for _, omitted := range []string{`"issuer"`, `"cache"`, `"guidanceCode"`, `"retryAfterSeconds"`} {
		if strings.Contains(jsonText, omitted) {
			t.Fatalf("optional field %s was not omitted: %s", omitted, jsonText)
		}
	}

	var decoded NumistaLookupOutcome
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if err := decoded.Validate(); err != nil {
		t.Fatalf("round-tripped outcome rejected: %v", err)
	}

	pathData, err := json.Marshal([]NumistaLookupPath{NumistaLookupPathDirect, NumistaLookupPathPhoto})
	if err != nil || string(pathData) != `["direct","photo"]` {
		t.Fatalf("lookup path JSON contract changed: %s err=%v", pathData, err)
	}
	stateData, err := json.Marshal([]NumistaEnrichmentState{
		NumistaEnrichmentNotRequested, NumistaEnrichmentEnriched,
		NumistaEnrichmentCached, NumistaEnrichmentFailed,
	})
	if err != nil || string(stateData) != `["not_requested","enriched","cached","failed"]` {
		t.Fatalf("enrichment state JSON contract changed: %s err=%v", stateData, err)
	}

	evidenceData, err := json.Marshal(NumistaEvidence{
		Title: "Coin", Issuer: "Issuer", Denomination: "Denomination", Mint: "Mint",
		DateText: "44 BCE", Material: "Silver", ObverseInscription: "Obverse",
		ReverseInscription: "Reverse", VisibleText: "Label", ExactNumistaID: intPointer(7),
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{
		`"title"`, `"issuer"`, `"denomination"`, `"mint"`, `"dateText"`, `"material"`,
		`"obverseInscription"`, `"reverseInscription"`, `"visibleText"`, `"exactNumistaId"`,
	} {
		if !strings.Contains(string(evidenceData), field) {
			t.Fatalf("evidence JSON contract missing %s: %s", field, evidenceData)
		}
	}

	createdAt := time.Date(2026, 8, 11, 12, 0, 0, 0, time.UTC)
	cacheData, err := json.Marshal(NumistaCacheMetadata{
		Hit: true, CreatedAt: createdAt, ExpiresAt: createdAt.Add(time.Hour), AgeSeconds: 5,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{`"hit"`, `"createdAt"`, `"expiresAt"`, `"ageSeconds"`} {
		if !strings.Contains(string(cacheData), field) {
			t.Fatalf("cache JSON contract missing %s: %s", field, cacheData)
		}
	}
}

func TestNumistaCacheMetadataVariants(t *testing.T) {
	createdAt := time.Date(2026, 8, 11, 12, 0, 0, 0, time.UTC)
	for _, hit := range []bool{false, true} {
		valid := NumistaCacheMetadata{Hit: hit, CreatedAt: createdAt, ExpiresAt: createdAt.Add(time.Hour), AgeSeconds: 0}
		if err := valid.Validate(); err != nil {
			t.Fatalf("valid cache metadata hit=%t rejected: %v", hit, err)
		}
	}
	for _, test := range []NumistaCacheMetadata{
		{ExpiresAt: createdAt.Add(time.Hour)},
		{CreatedAt: createdAt, ExpiresAt: createdAt},
		{CreatedAt: createdAt, ExpiresAt: createdAt.Add(-time.Second)},
		{CreatedAt: createdAt, ExpiresAt: createdAt.Add(time.Hour), AgeSeconds: -1},
	} {
		if err := test.Validate(); err == nil {
			t.Fatalf("invalid cache metadata accepted: %+v", test)
		}
	}
}

func TestNumistaLookupOutcomeVariants(t *testing.T) {
	for _, status := range []NumistaLookupStatus{
		NumistaStatusSuccess, NumistaStatusEmpty, NumistaStatusUnconfigured,
		NumistaStatusQuotaLimited, NumistaStatusTimeout, NumistaStatusUnavailable,
	} {
		outcome := validNumistaTestOutcome()
		outcome.Status = status
		if status != NumistaStatusSuccess {
			outcome.Candidates = []NumistaCandidate{}
		}
		if err := outcome.Validate(); err != nil {
			t.Fatalf("status %q rejected: %v", status, err)
		}
	}
	for _, stage := range []string{"broad", "enriched"} {
		outcome := validNumistaTestOutcome()
		outcome.Stage = stage
		if err := outcome.Validate(); err != nil {
			t.Fatalf("stage %q rejected: %v", stage, err)
		}
	}

	createdAt := time.Date(2026, 8, 11, 12, 0, 0, 0, time.UTC)
	for _, status := range []NumistaLookupStatus{NumistaStatusSuccess, NumistaStatusEmpty} {
		outcome := validNumistaTestOutcome()
		outcome.Status = status
		if status == NumistaStatusEmpty {
			outcome.Candidates = []NumistaCandidate{}
		}
		outcome.Cache = &NumistaCacheMetadata{Hit: true, CreatedAt: createdAt, ExpiresAt: createdAt.Add(time.Hour)}
		if err := outcome.Validate(); err != nil {
			t.Fatalf("cached %q outcome rejected: %v", status, err)
		}
	}
	for _, test := range []struct {
		name   string
		mutate func(*NumistaLookupOutcome)
	}{
		{"invalid status", func(o *NumistaLookupOutcome) { o.Status = "other" }},
		{"blank effective query", func(o *NumistaLookupOutcome) { o.EffectiveQuery = " " }},
		{"invalid stage", func(o *NumistaLookupOutcome) { o.Stage = "other" }},
		{"nil candidates", func(o *NumistaLookupOutcome) { o.Candidates = nil }},
		{"success without candidates", func(o *NumistaLookupOutcome) { o.Candidates = []NumistaCandidate{} }},
		{"non-success with candidates", func(o *NumistaLookupOutcome) { o.Status = NumistaStatusEmpty }},
		{"invalid candidate", func(o *NumistaLookupOutcome) { o.Candidates[0].Title = "" }},
		{"nonpositive retry after", func(o *NumistaLookupOutcome) { o.RetryAfterSeconds = intPointer(0) }},
		{"invalid cache metadata", func(o *NumistaLookupOutcome) {
			o.Cache = &NumistaCacheMetadata{CreatedAt: createdAt, ExpiresAt: createdAt}
		}},
		{"cache on unavailable outcome", func(o *NumistaLookupOutcome) {
			o.Status = NumistaStatusUnavailable
			o.Candidates = []NumistaCandidate{}
			o.Cache = &NumistaCacheMetadata{CreatedAt: createdAt, ExpiresAt: createdAt.Add(time.Hour)}
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			outcome := validNumistaTestOutcome()
			test.mutate(&outcome)
			if err := outcome.Validate(); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestNumistaHealthSummaryVariantsAndJSONContract(t *testing.T) {
	now := time.Date(2026, 8, 11, 12, 0, 0, 0, time.UTC)
	retryAfter := 60
	valid := NumistaHealthSummary{
		Configured: true, ConfigurationValid: true, LastOutcome: NumistaStatusSuccess,
		LastCheckedAt: &now, StatusCounts: map[NumistaLookupStatus]int{NumistaStatusSuccess: 1},
		FreshCacheHitRate: 0.5, LastQuotaLimitedAt: &now, LastRetryAfterSeconds: &retryAfter,
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid health summary rejected: %v", err)
	}
	data, err := json.Marshal(valid)
	if err != nil {
		t.Fatal(err)
	}
	empty := valid
	empty.StatusCounts = map[NumistaLookupStatus]int{}
	emptyData, err := json.Marshal(empty)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(emptyData), `"statusCounts":{}`) {
		t.Fatalf("empty statusCounts must remain a sparse object: %s", emptyData)
	}
	if !strings.Contains(string(data), `"statusCounts":{"success":1}`) {
		t.Fatalf("partial statusCounts must remain sparse: %s", data)
	}
	for _, field := range []string{
		`"configured"`, `"configurationValid"`, `"statusCounts"`, `"broadRequestCount"`,
		`"detailRequestCount"`, `"freshCacheHitCount"`, `"coalescedRequestCount"`,
		`"providerLoadCount"`, `"providerFailureCount"`, `"cancelledRequestCount"`,
		`"freshCacheHitRate"`,
		`"p50ElapsedMs"`, `"p95ElapsedMs"`, `"enrichmentAttempted"`,
		`"enrichmentSucceeded"`, `"enrichmentFailed"`,
	} {
		if !strings.Contains(string(data), field) {
			t.Fatalf("JSON contract missing %s: %s", field, data)
		}
	}

	for _, test := range []struct {
		name   string
		mutate func(*NumistaHealthSummary)
	}{
		{"nil status counts", func(h *NumistaHealthSummary) { h.StatusCounts = nil }},
		{"invalid last outcome", func(h *NumistaHealthSummary) { h.LastOutcome = "other" }},
		{"invalid status count key", func(h *NumistaHealthSummary) { h.StatusCounts = map[NumistaLookupStatus]int{"other": 1} }},
		{"negative status count", func(h *NumistaHealthSummary) { h.StatusCounts[NumistaStatusSuccess] = -1 }},
		{"negative broad count", func(h *NumistaHealthSummary) { h.BroadRequestCount = -1 }},
		{"negative detail count", func(h *NumistaHealthSummary) { h.DetailRequestCount = -1 }},
		{"negative fresh cache hit count", func(h *NumistaHealthSummary) { h.FreshCacheHitCount = -1 }},
		{"negative coalesced count", func(h *NumistaHealthSummary) { h.CoalescedRequestCount = -1 }},
		{"negative provider load count", func(h *NumistaHealthSummary) { h.ProviderLoadCount = -1 }},
		{"negative provider failure count", func(h *NumistaHealthSummary) { h.ProviderFailureCount = -1 }},
		{"negative cancellation count", func(h *NumistaHealthSummary) { h.CancelledRequestCount = -1 }},
		{"fresh cache hit rate below minimum", func(h *NumistaHealthSummary) { h.FreshCacheHitRate = -0.1 }},
		{"fresh cache hit rate above maximum", func(h *NumistaHealthSummary) { h.FreshCacheHitRate = 1.1 }},
		{"negative p50", func(h *NumistaHealthSummary) { h.P50ElapsedMs = -1 }},
		{"negative p95", func(h *NumistaHealthSummary) { h.P95ElapsedMs = -1 }},
		{"negative enrichment attempted", func(h *NumistaHealthSummary) { h.EnrichmentAttempted = -1 }},
		{"negative enrichment succeeded", func(h *NumistaHealthSummary) { h.EnrichmentSucceeded = -1 }},
		{"negative enrichment failed", func(h *NumistaHealthSummary) { h.EnrichmentFailed = -1 }},
		{"zero last checked timestamp", func(h *NumistaHealthSummary) { zero := time.Time{}; h.LastCheckedAt = &zero }},
		{"zero quota timestamp", func(h *NumistaHealthSummary) { zero := time.Time{}; h.LastQuotaLimitedAt = &zero }},
		{"nonpositive retry after", func(h *NumistaHealthSummary) { h.LastRetryAfterSeconds = intPointer(0) }},
	} {
		t.Run(test.name, func(t *testing.T) {
			summary := valid
			summary.StatusCounts = map[NumistaLookupStatus]int{NumistaStatusSuccess: 1}
			test.mutate(&summary)
			if err := summary.Validate(); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}
