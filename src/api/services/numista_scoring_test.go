package services

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/briandenicola/ancient-coins-api/models"
)

func TestNumistaScorerWeightsReasonsAndStableTies(t *testing.T) {
	exactID := 2
	request := models.NumistaLookupRequest{
		Query: "Trajan denarius", Path: models.NumistaLookupPathDirect,
		Evidence: models.NumistaEvidence{
			Title: "Trajan denarius", Issuer: "Roman Empire", Denomination: "Denarius",
			Mint: "Rome", DateText: "101-102 CE", Material: "Silver",
			ObverseInscription: "IMP TRAIANO", ExactNumistaID: &exactID,
		},
	}
	minYear, maxYear := 101, 102
	candidates := []models.NumistaCandidate{
		{ID: 3, Title: "Trajan denarius", Issuer: "Roman Empire", ProviderPosition: 0},
		{ID: 2, Title: "Trajan denarius", Issuer: "Roman Empire", Denomination: "Denarius", Mint: "Rome",
			Material: "Silver", ObverseInscription: "IMP TRAIANO", MinYear: &minYear, MaxYear: &maxYear, ProviderPosition: 1},
	}
	ranked := NewNumistaV1Scorer().Rank(request, candidates)
	if ranked[0].ID != 2 || ranked[0].Assessment.Score <= ranked[1].Assessment.Score {
		t.Fatalf("exact rich candidate was not first: %+v", ranked)
	}
	wantOrder := []string{"exact_id", "title", "issuer", "denomination", "mint", "date", "material", "inscription"}
	if len(ranked[0].Assessment.Reasons) != len(wantOrder) {
		t.Fatalf("reason count = %d, want %d", len(ranked[0].Assessment.Reasons), len(wantOrder))
	}
	for i, field := range wantOrder {
		if ranked[0].Assessment.Reasons[i].Field != field {
			t.Fatalf("reason %d field = %q, want %q", i, ranked[0].Assessment.Reasons[i].Field, field)
		}
	}
	for _, reason := range ranked[0].Assessment.Reasons {
		if strings.Contains(reason.Label, "IMP TRAIANO") {
			t.Fatal("reason leaked full inscription")
		}
	}
}

func TestNumistaScorerBCEOverlapNormalizationAndMissingNeutrality(t *testing.T) {
	minYear, maxYear := -44, -42
	request := models.NumistaLookupRequest{
		Query: "ΑΝΤΩΝΙΟΣ", Path: models.NumistaLookupPathDirect,
		Evidence: models.NumistaEvidence{Title: "  ΑΝΤΩΝΙΟΣ!!! ", DateText: "44 BCE - 40 BCE", Material: "silver"},
	}
	candidate := models.NumistaCandidate{ID: 1, Title: "αντωνιοσ", MinYear: &minYear, MaxYear: &maxYear}
	assessment := NewNumistaV1Scorer().Score(request, candidate)
	if assessment.Score < 50 {
		t.Fatalf("expected normalized title/date support, got %+v", assessment)
	}
	if NormalizeNumistaText("Ａ  A!!!") != "a a" {
		t.Fatalf("unexpected NFKC normalization: %q", NormalizeNumistaText("Ａ  A!!!"))
	}
	if NormalizeNumistaText("ΑΝΤΩΝΙΟΣ") != NormalizeNumistaText("αντωνιοσ") {
		t.Fatalf("Greek case-fold normalization diverged: %q vs %q",
			NormalizeNumistaText("ΑΝΤΩΝΙΟΣ"), NormalizeNumistaText("αντωνιοσ"))
	}
}

func TestNumistaScorerBoundsDuplicateEvidenceAndStableTieBreak(t *testing.T) {
	longEvidence := strings.Repeat("TRAJAN!!! ", 150)
	if len(boundedTokens(longEvidence)) != 1 {
		t.Fatal("duplicate evidence tokens were not deduplicated")
	}
	request := models.NumistaLookupRequest{Query: "coin", Path: models.NumistaLookupPathDirect}
	ranked := NewNumistaV1Scorer().Rank(request, []models.NumistaCandidate{
		{ID: 9, Title: "Same", ProviderPosition: 0},
		{ID: 3, Title: "Same", ProviderPosition: 1},
	})
	if ranked[0].ID != 3 {
		t.Fatalf("numeric stable tie-break failed: %+v", ranked)
	}
}

func TestNumistaScorerTieBreakIsPermutationIndependentAndNumeric(t *testing.T) {
	request := models.NumistaLookupRequest{Query: "coin", Path: models.NumistaLookupPathDirect}
	candidates := []models.NumistaCandidate{
		{ID: 10, Title: "Same", ProviderPosition: 0},
		{ID: 2, Title: "Same", ProviderPosition: 2},
		{ID: 30, Title: "Same", ProviderPosition: 1},
	}
	permutations := [][]int{
		{0, 1, 2}, {0, 2, 1}, {1, 0, 2},
		{1, 2, 0}, {2, 0, 1}, {2, 1, 0},
	}
	for _, permutation := range permutations {
		input := make([]models.NumistaCandidate, len(permutation))
		for index, source := range permutation {
			input[index] = candidates[source]
		}
		ranked := NewNumistaV1Scorer().Rank(request, input)
		got := []int{ranked[0].ID, ranked[1].ID, ranked[2].ID}
		if !reflect.DeepEqual(got, []int{2, 10, 30}) {
			t.Fatalf("permutation %v ranked %v, want numeric ID order", permutation, got)
		}
	}
}

func TestNumistaScorerAppliesEverySpecifiedWeight(t *testing.T) {
	exactID := 99
	request := models.NumistaLookupRequest{
		Query: "matching title", Path: models.NumistaLookupPathDirect,
		Evidence: models.NumistaEvidence{
			Title: "matching title", Issuer: "matching issuer", Denomination: "matching denomination",
			Mint: "matching mint", DateText: "44-40 BCE", Material: "matching material",
			ObverseInscription: "matching inscription", ExactNumistaID: &exactID,
		},
	}
	conflictMin, conflictMax := 100, 110
	tests := []struct {
		field     string
		wantScore int
		match     func(*models.NumistaCandidate)
	}{
		{"exact_id", 35, func(c *models.NumistaCandidate) { c.ID = exactID }},
		{"title", 15, func(c *models.NumistaCandidate) { c.Title = "matching title" }},
		{"issuer", 12, func(c *models.NumistaCandidate) { c.Issuer = "matching issuer" }},
		{"denomination", 12, func(c *models.NumistaCandidate) { c.Denomination = "matching denomination" }},
		{"mint", 10, func(c *models.NumistaCandidate) { c.Mint = "matching mint" }},
		{"date", 8, func(c *models.NumistaCandidate) {
			minYear, maxYear := -44, -40
			c.MinYear, c.MaxYear = &minYear, &maxYear
		}},
		{"material", 5, func(c *models.NumistaCandidate) { c.Material = "matching material" }},
		{"inscription", 3, func(c *models.NumistaCandidate) { c.ObverseInscription = "matching inscription" }},
	}
	for _, test := range tests {
		t.Run(test.field, func(t *testing.T) {
			candidate := models.NumistaCandidate{
				ID: 1, Title: "different", Issuer: "different", Denomination: "different",
				Mint: "different", MinYear: &conflictMin, MaxYear: &conflictMax,
				Material: "different", ObverseInscription: "different",
			}
			test.match(&candidate)
			assessment := NewNumistaV1Scorer().Score(request, candidate)
			if assessment.Score != test.wantScore {
				t.Fatalf("%s-only match score=%d, want %d: %+v", test.field, assessment.Score, test.wantScore, assessment)
			}
			var matched bool
			for _, reason := range assessment.Reasons {
				if reason.Field == test.field && reason.Kind == models.NumistaReasonMatch {
					matched = true
				}
			}
			if !matched {
				t.Fatalf("%s match reason missing: %+v", test.field, assessment.Reasons)
			}
		})
	}
}

func TestParseDateRangeSupportedBCECEForms(t *testing.T) {
	tests := []struct {
		value        string
		wantMin      int
		wantMax      int
		wantParsable bool
	}{
		{"44–40 BCE", -44, -40, true},
		{"44-40 BCE", -44, -40, true},
		{"44 BCE-40 BCE", -44, -40, true},
		{"44 BC to 40 BC", -44, -40, true},
		{"-44--40", -44, -40, true},
		{"-44 to -40", -44, -40, true},
		{"40 BCE—10 CE", -40, 10, true},
		{"10-12 AD", 10, 12, true},
		{"+10 CE", 10, 10, true},
		{"44 BCE", -44, -44, true},
		{"44 BCE uncertain", 0, 0, false},
		{"-44 CE", 0, 0, false},
	}
	for _, test := range tests {
		t.Run(test.value, func(t *testing.T) {
			gotMin, gotMax, gotParsable := parseDateRange(test.value)
			if gotMin != test.wantMin || gotMax != test.wantMax || gotParsable != test.wantParsable {
				t.Fatalf("parseDateRange(%q)=(%d,%d,%t), want (%d,%d,%t)",
					test.value, gotMin, gotMax, gotParsable,
					test.wantMin, test.wantMax, test.wantParsable)
			}
		})
	}
}

func TestNumistaScorerDateOverlapConflictAndEquivalentTie(t *testing.T) {
	candidateMin, candidateMax := -42, -40
	tests := []struct {
		dateText string
		wantKind models.NumistaReasonKind
		wantCode string
	}{
		{"44–40 BCE", models.NumistaReasonMatch, "date_overlap"},
		{"44-40 BCE", models.NumistaReasonMatch, "date_overlap"},
		{"39 BCE", models.NumistaReasonConflict, "date_conflict"},
		{"unparseable", models.NumistaReasonUnavailable, "date_unavailable"},
	}
	for _, test := range tests {
		t.Run(test.dateText, func(t *testing.T) {
			assessment := NewNumistaV1Scorer().Score(models.NumistaLookupRequest{
				Query: "coin", Path: models.NumistaLookupPathDirect,
				Evidence: models.NumistaEvidence{DateText: test.dateText},
			}, models.NumistaCandidate{
				ID: 1, Title: "coin", MinYear: &candidateMin, MaxYear: &candidateMax,
			})
			dateReason := assessment.Reasons[len(assessment.Reasons)-1]
			if dateReason.Kind != test.wantKind || dateReason.Code != test.wantCode {
				t.Fatalf("date reason=%+v, want kind=%q code=%q", dateReason, test.wantKind, test.wantCode)
			}
		})
	}

	scorer := NewNumistaV1Scorer()
	for _, dateText := range []string{"44–40 BCE", "44-40 BCE", "-44--40"} {
		ranked := scorer.Rank(models.NumistaLookupRequest{
			Query: "coin", Path: models.NumistaLookupPathDirect,
			Evidence: models.NumistaEvidence{DateText: dateText},
		}, []models.NumistaCandidate{
			{ID: 9, Title: "Same", MinYear: &candidateMin, MaxYear: &candidateMax, ProviderPosition: 0},
			{ID: 3, Title: "Same", MinYear: &candidateMin, MaxYear: &candidateMax, ProviderPosition: 1},
		})
		if ranked[0].ID != 3 || ranked[0].Assessment.Score != ranked[1].Assessment.Score {
			t.Fatalf("equivalent date %q produced unstable tie: %+v", dateText, ranked)
		}
	}
}

func TestNumistaScorerLongDistinctEvidenceIsBounded(t *testing.T) {
	parts := make([]string, 150)
	for i := range parts {
		parts[i] = fmt.Sprintf("token%d", i)
	}
	if got := len(boundedTokens(strings.Join(parts, " "))); got != 100 {
		t.Fatalf("bounded token count=%d, want 100", got)
	}
}

func TestNumistaScorerUsesSpecifiedConfidenceBandThresholds(t *testing.T) {
	tests := []struct {
		score int
		band  string
	}{
		{59, "weak"},
		{60, "possible"},
		{79, "possible"},
		{80, "strong"},
	}
	for _, test := range tests {
		if got := relevanceBand(test.score); got != test.band {
			t.Errorf("relevanceBand(%d) = %q, want %q", test.score, got, test.band)
		}
	}

}

func TestNumistaScorerMissingOptionalEvidenceIsNeutralAndDeterministic(t *testing.T) {
	tests := []struct {
		name     string
		field    string
		evidence models.NumistaEvidence
		wantCode string
	}{
		{
			name: "missing date", field: "date",
			evidence: models.NumistaEvidence{DateText: "44 BCE"},
			wantCode: "date_unavailable",
		},
		{
			name: "missing ruler", field: "issuer",
			evidence: models.NumistaEvidence{Issuer: "Trajan"},
			wantCode: "candidate_value_missing",
		},
		{
			name: "missing denomination", field: "denomination",
			evidence: models.NumistaEvidence{Denomination: "Denarius"},
			wantCode: "candidate_value_missing",
		},
		{
			name: "missing inscriptions", field: "inscription",
			evidence: models.NumistaEvidence{
				ObverseInscription: "IMP TRAIANO", ReverseInscription: "PAX",
			},
			wantCode: "candidate_value_missing",
		},
	}
	scorer := NewNumistaV1Scorer()
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := models.NumistaLookupRequest{
				Path: models.NumistaLookupPathDirect, Evidence: test.evidence,
			}
			candidate := models.NumistaCandidate{ID: 9, Title: "Same", ProviderPosition: 0}
			assessment := scorer.Score(request, candidate)
			if assessment.Score != 50 {
				t.Fatalf("missing %s score=%d, want neutral 50", test.field, assessment.Score)
			}
			if len(assessment.Reasons) != 1 {
				t.Fatalf("missing %s reasons=%+v, want one unavailable reason", test.field, assessment.Reasons)
			}
			reason := assessment.Reasons[0]
			if reason.Field != test.field || reason.Kind != models.NumistaReasonUnavailable ||
				reason.Code != test.wantCode {
				t.Fatalf("missing %s reason=%+v", test.field, reason)
			}

			candidates := []models.NumistaCandidate{
				candidate,
				{ID: 3, Title: "Same", ProviderPosition: 1},
			}
			first := scorer.Rank(request, candidates)
			second := scorer.Rank(request, candidates)
			if first[0].ID != 3 || first[1].ID != 9 ||
				first[0].Assessment.Score != first[1].Assessment.Score ||
				second[0].ID != first[0].ID || second[1].ID != first[1].ID {
				t.Fatalf("missing %s disturbed deterministic tie: first=%+v second=%+v", test.field, first, second)
			}
			for _, ranked := range first {
				for _, rankedReason := range ranked.Assessment.Reasons {
					if rankedReason.Kind == models.NumistaReasonConflict {
						t.Fatalf("missing %s produced conflict: %+v", test.field, rankedReason)
					}
				}
			}
		})
	}
}

func TestNumistaScorerOmittedOptionalEvidenceAddsNoReasonOrWeight(t *testing.T) {
	request := models.NumistaLookupRequest{
		Query: "same", Path: models.NumistaLookupPathDirect,
	}
	assessment := NewNumistaV1Scorer().Score(request, models.NumistaCandidate{ID: 1, Title: "same"})
	if assessment.Score != 100 || len(assessment.Reasons) != 1 ||
		assessment.Reasons[0].Field != "title" {
		t.Fatalf("omitted optional evidence affected scoring: %+v", assessment)
	}
}
