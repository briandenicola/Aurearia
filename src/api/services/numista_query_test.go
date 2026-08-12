package services

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/briandenicola/ancient-coins-api/models"
)

func TestNumistaQueryBuilderV2Plans(t *testing.T) {
	tests := []struct {
		name     string
		evidence models.NumistaEvidence
		primary  string
		relaxed  string
		scoring  []string
	}{
		{
			name: "measured Honorius SMNT fixture",
			evidence: models.NumistaEvidence{
				Title:              "Honorius AE3 GLORIA ROMANORVM RIC IX 46 LRBC 2424",
				Issuer:             "Honorius",
				Denomination:       "AE3",
				Mint:               "SMNT",
				DateText:           "393-423 CE",
				Material:           "Bronze",
				ObverseInscription: "DN HONORIVS PF AVG",
				ReverseInscription: "GLORIA ROMANORVM",
				VisibleText:        "NGC Ancients Honorius AE3 RIC IX 46",
			},
			primary: "Honorius GLORIA ROMANORVM Nicomedia",
			relaxed: "Honorius Nicomedia",
		},
		{
			name: "exact SMN alias",
			evidence: models.NumistaEvidence{
				Issuer:             "Valentinian II",
				Mint:               "S.M-N",
				ReverseInscription: "VOT V MVLT X",
			},
			primary: "Valentinian II VOT V MVLT X Nicomedia",
			relaxed: "Valentinian II Nicomedia",
		},
		{
			name: "unknown mintmark is omitted without guessing",
			evidence: models.NumistaEvidence{
				Issuer:             "Aurelian",
				Mint:               "XXIT",
				ReverseInscription: "ORIENS AVG",
			},
			primary: "Aurelian ORIENS AVG",
			relaxed: "Aurelian",
		},
		{
			name: "ordinary mint name remains reliable",
			evidence: models.NumistaEvidence{
				Issuer:             "Hadrian",
				Mint:               "Rome",
				ReverseInscription: "FELICITAS AVG",
			},
			primary: "Hadrian FELICITAS AVG Rome",
			relaxed: "Hadrian Rome",
		},
		{
			name: "reverse type fallback deduplicates and preserves rich scoring evidence",
			evidence: models.NumistaEvidence{
				Title:              "Hadrian denarius Victory RIC II 101",
				Issuer:             "Hadrian",
				Denomination:       "Denarius",
				Mint:               "Rome",
				DateText:           "117-138 CE",
				Material:           "Silver",
				ObverseInscription: "HADRIANVS AVG COS III PP",
				ReverseType:        "Victory standing left victory",
				VisibleText:        "Dealer description and slab label",
				ExactNumistaID:     intPointerForQueryTest(184126),
			},
			primary: "Hadrian Victory standing left Rome",
			relaxed: "Hadrian Rome",
			scoring: []string{"exact_id", "denomination", "material", "inscription"},
		},
		{
			name: "short structured title fallback",
			evidence: models.NumistaEvidence{
				Title:              "Athens tetradrachm owl",
				Mint:               "Athens",
				ReverseInscription: "ΑΘΕ",
			},
			primary: "Athens tetradrachm owl ΑΘΕ",
			relaxed: "Athens tetradrachm owl",
		},
		{
			name: "deduplicates repeated subject and reverse phrases",
			evidence: models.NumistaEvidence{
				Title:              "Justinian I follis",
				Issuer:             "Justinian I",
				Mint:               "Constantinople",
				ReverseInscription: "ANNO XII ANNO XII",
			},
			primary: "Justinian I ANNO XII Constantinople",
			relaxed: "Justinian I Constantinople",
		},
		{
			name: "fullwidth Unicode alias normalizes exactly",
			evidence: models.NumistaEvidence{
				Issuer:             "Constantine I",
				Mint:               "ＳＭＮＴ",
				ReverseInscription: "VOT XX",
			},
			primary: "Constantine I VOT XX Nicomedia",
			relaxed: "Constantine I Nicomedia",
		},
		{
			name: "mixed case alias folds exactly",
			evidence: models.NumistaEvidence{
				Issuer:             "Constantine I",
				Mint:               "sMn",
				ReverseInscription: "VOT XX",
			},
			primary: "Constantine I VOT XX Nicomedia",
			relaxed: "Constantine I Nicomedia",
		},
		{
			name: "embedded alias in longer token is not expanded",
			evidence: models.NumistaEvidence{
				Issuer:             "Constantine I",
				Mint:               "SMNTopolis",
				ReverseInscription: "VOT XX",
			},
			primary: "Constantine I VOT XX SMNTopolis",
			relaxed: "Constantine I SMNTopolis",
		},
		{
			name: "embedded alias in prose is not expanded",
			evidence: models.NumistaEvidence{
				Issuer:             "Constantine I",
				Mint:               "mint SMNT field",
				ReverseInscription: "VOT XX",
			},
			primary: "Constantine I VOT XX",
			relaxed: "Constantine I",
		},
		{
			name: "canonically equivalent Unicode tokens deduplicate",
			evidence: models.NumistaEvidence{
				Issuer:             "Café",
				Mint:               "Rome",
				ReverseInscription: "Cafe\u0301",
			},
			primary: "Café Rome",
			relaxed: "Café Rome",
		},
		{
			name: "Unicode case folding deduplicates reverse tokens",
			evidence: models.NumistaEvidence{
				Issuer:             "Theodosius",
				ReverseInscription: "ΟΣ ος",
			},
			primary: "Theodosius ΟΣ",
			relaxed: "Theodosius",
		},
	}

	builder := NewNumistaQueryBuilder()
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			original := test.evidence
			plan := builder.Build(test.evidence)
			if plan.Version != "numista-query-v2" ||
				plan.Primary != test.primary || plan.Relaxed != test.relaxed {
				t.Fatalf("plan = %#v, want primary=%q relaxed=%q", plan, test.primary, test.relaxed)
			}
			if len(test.scoring) == 0 {
				return
			}
			if test.evidence != original {
				t.Fatalf("builder mutated evidence needed by scorer: got %#v want %#v", test.evidence, original)
			}
			assessment := NewNumistaV1Scorer().Score(models.NumistaLookupRequest{
				Query:    plan.Primary,
				Evidence: test.evidence,
			}, models.NumistaCandidate{
				ID:                 *test.evidence.ExactNumistaID,
				Title:              test.evidence.Title,
				Issuer:             test.evidence.Issuer,
				Denomination:       test.evidence.Denomination,
				Mint:               test.evidence.Mint,
				Material:           test.evidence.Material,
				ObverseInscription: test.evidence.ObverseInscription,
				ReverseInscription: test.evidence.ReverseType,
			})
			fields := make([]string, 0, len(assessment.Reasons))
			for _, reason := range assessment.Reasons {
				fields = append(fields, reason.Field)
			}
			for _, field := range test.scoring {
				if !slices.Contains(fields, field) {
					t.Fatalf("scoring evidence field %q was not preserved; reasons = %#v", field, assessment.Reasons)
				}
			}
		})
	}
}

func TestNumistaQueryBuilderV2ExclusionsBoundsAndAliasCeiling(t *testing.T) {
	builder := NewNumistaQueryBuilder()
	plan := builder.Build(models.NumistaEvidence{
		Title:              "Honorius AE3 GLORIA ROMANORVM RIC IX 46 LRBC 2424",
		Issuer:             "Honorius",
		Denomination:       "AE3",
		Mint:               "SMNT",
		DateText:           "393-423 CE",
		Material:           "Bronze",
		ObverseInscription: "DN HONORIVS PF AVG",
		ReverseInscription: "GLORIA ROMANORVM",
		VisibleText:        "NGC Ancients full slab label text",
	})
	if plan.Primary != "Honorius GLORIA ROMANORVM Nicomedia" {
		t.Fatalf("excluded evidence leaked into provider query: %q", plan.Primary)
	}
	if len([]rune(plan.Primary)) > models.NumistaMaxQueryLength ||
		len([]rune(plan.Relaxed)) > models.NumistaMaxQueryLength {
		t.Fatalf("query plan exceeded provider bound: %#v", plan)
	}
	if got := builder.AliasCount(); got > 32 {
		t.Fatalf("mint alias allowlist has %d entries, want at most 32", got)
	}
}

func TestNumistaQueryV2ComparisonFixtureReplay(t *testing.T) {
	type candidateIDs struct {
		Old     []int `json:"old"`
		Primary []int `json:"primary"`
		Relaxed []int `json:"relaxed"`
	}
	type comparisonCase struct {
		Name                string                 `json:"name"`
		LiveEvidence        bool                   `json:"liveEvidence"`
		Evidence            models.NumistaEvidence `json:"evidence"`
		OldQuery            string                 `json:"oldQuery"`
		PrimaryQuery        string                 `json:"primaryQuery"`
		RelaxedQuery        string                 `json:"relaxedQuery"`
		ExpectedCandidateID int                    `json:"expectedCandidateId"`
		FrozenCandidateIDs  candidateIDs           `json:"frozenCandidateIds"`
	}
	var fixture struct {
		SchemaVersion int              `json:"schemaVersion"`
		Cases         []comparisonCase `json:"cases"`
	}
	data, err := os.ReadFile(filepath.Join("testdata", "numista", "query_v2_comparison.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	if fixture.SchemaVersion != 1 || len(fixture.Cases) < 12 {
		t.Fatalf("comparison fixture version/case count = %d/%d", fixture.SchemaVersion, len(fixture.Cases))
	}

	liveRanks := map[string][3]int{
		"direct-honorius-smnt":            {0, 3, 3},
		"direct-valentinian-smn":          {0, 0, 0},
		"draft-trajan-reverse-legend":     {0, 0, 0},
		"direct-athens-title-fallback":    {0, 0, 3},
		"photo-justinian-reverse-type":    {1, 1, 0},
		"draft-aurelian-unknown-mintmark": {0, 2, 0},
	}
	builder := NewNumistaQueryBuilder()
	liveCount, verboseTopThree, primaryTopThree := 0, 0, 0
	for _, test := range fixture.Cases {
		t.Run(test.Name, func(t *testing.T) {
			plan := builder.Build(test.Evidence)
			if plan.Primary != test.PrimaryQuery || plan.Relaxed != test.RelaxedQuery {
				t.Fatalf("fixture plan = %#v", plan)
			}
			if test.OldQuery == "" || test.ExpectedCandidateID <= 0 {
				t.Fatalf("fixture is incomplete: %#v", test)
			}
			if !test.LiveEvidence {
				return
			}
			want, ok := liveRanks[test.Name]
			if !ok {
				t.Fatalf("live evidence case %q has no documented rank expectation", test.Name)
			}
			got := [3]int{
				candidateFixtureRank(test.FrozenCandidateIDs.Old, test.ExpectedCandidateID),
				candidateFixtureRank(test.FrozenCandidateIDs.Primary, test.ExpectedCandidateID),
				candidateFixtureRank(test.FrozenCandidateIDs.Relaxed, test.ExpectedCandidateID),
			}
			if got != want {
				t.Fatalf("frozen ranks = %v, want live-evidence ranks %v", got, want)
			}
			liveCount++
			if got[0] > 0 && got[0] <= 3 {
				verboseTopThree++
			}
			if got[1] > 0 && got[1] <= 3 {
				primaryTopThree++
			}
		})
	}
	if liveCount != 6 || verboseTopThree != 1 || primaryTopThree != 3 {
		t.Fatalf(
			"live replay = %d cases, verbose %d/6, primary %d/6; want 6, 1/6, 3/6",
			liveCount, verboseTopThree, primaryTopThree,
		)
	}
}

func candidateFixtureRank(ids []int, expected int) int {
	for index, id := range ids {
		if id == expected {
			return index + 1
		}
	}
	return 0
}

func intPointerForQueryTest(value int) *int {
	return &value
}
