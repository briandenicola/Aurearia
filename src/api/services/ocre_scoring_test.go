package services

import (
	"strings"
	"testing"
)

// Feature 345 T011 — deterministic scoring (SC-005).

func rows(uris ...string) []ocreParsedRow {
	out := make([]ocreParsedRow, 0, len(uris))
	for _, u := range uris {
		out = append(out, ocreParsedRow{TypeURI: u, Label: "Label " + u})
	}
	return out
}

func TestOCREScore_DeterministicAcrossRuns(t *testing.T) {
	params := NewOCREQueryParams("hadrian", "denarius", "rome", "", nil, "", 5)
	in := rows(
		"http://numismatics.org/ocre/id/ric.2.hdn.40",
		"http://numismatics.org/ocre/id/ric.2.hdn.39b",
		"http://numismatics.org/ocre/id/ric.2.hdn.41",
	)
	first := Score(params, in)
	second := Score(params, in)
	if len(first) != len(second) {
		t.Fatalf("length mismatch: %d vs %d", len(first), len(second))
	}
	for i := range first {
		if first[i].TypeURI != second[i].TypeURI || first[i].Confidence != second[i].Confidence {
			t.Fatalf("non-deterministic ordering at %d: %+v vs %+v", i, first[i], second[i])
		}
	}
}

func TestOCREScore_TieBreakByCanonicalURI(t *testing.T) {
	// Same confidence (identical matched fields), so ordering falls to
	// TypeURI ascending.
	params := NewOCREQueryParams("hadrian", "", "", "", nil, "", 5)
	in := rows(
		"http://numismatics.org/ocre/id/ric.2.hdn.40",
		"http://numismatics.org/ocre/id/ric.2.hdn.39b",
	)
	got := Score(params, in)
	if len(got) != 2 {
		t.Fatalf("expected 2 candidates, got %d", len(got))
	}
	if got[0].TypeURI >= got[1].TypeURI {
		t.Fatalf("expected ascending TypeURI tie-break, got %q then %q", got[0].TypeURI, got[1].TypeURI)
	}
	// canonical scheme normalized to https
	if !strings.HasPrefix(got[0].TypeURI, "https://numismatics.org/") {
		t.Fatalf("expected canonical https URI, got %q", got[0].TypeURI)
	}
}

func TestOCREScore_DedupBeforeRanking(t *testing.T) {
	params := NewOCREQueryParams("hadrian", "", "", "", nil, "", 5)
	in := rows(
		"http://numismatics.org/ocre/id/ric.2.hdn.39b",
		"https://numismatics.org/ocre/id/ric.2.hdn.39b", // same canonical URI
	)
	got := Score(params, in)
	if len(got) != 1 {
		t.Fatalf("expected de-dup to a single candidate, got %d: %+v", len(got), got)
	}
}

func TestOCREScore_OffHostDropped(t *testing.T) {
	params := NewOCREQueryParams("hadrian", "", "", "", nil, "", 5)
	in := rows(
		"http://evil.example.com/ocre/id/ric.2.hdn.39b",
		"http://numismatics.org/ocre/id/ric.2.hdn.40",
	)
	got := Score(params, in)
	if len(got) != 1 {
		t.Fatalf("expected the off-host row to be dropped, got %d: %+v", len(got), got)
	}
	if !strings.Contains(got[0].TypeURI, "numismatics.org") {
		t.Fatalf("unexpected surviving candidate: %+v", got[0])
	}
}

func TestOCREScore_WeightingAuthorityOverMaterial(t *testing.T) {
	authority := Score(NewOCREQueryParams("hadrian", "", "", "", nil, "", 5), rows("http://numismatics.org/ocre/id/a"))
	material := Score(NewOCREQueryParams("", "", "", "silver", nil, "", 5), rows("http://numismatics.org/ocre/id/a"))
	// material-only would not normally be queried, but scoring must still
	// weight an authority match strictly higher than a material match.
	if len(authority) != 1 || len(material) != 1 {
		t.Fatalf("expected one candidate each: %d/%d", len(authority), len(material))
	}
	if !(authority[0].Confidence > material[0].Confidence) {
		t.Fatalf("authority (%.2f) must outweigh material (%.2f)", authority[0].Confidence, material[0].Confidence)
	}
}

func TestOCREScore_CapEnforced(t *testing.T) {
	params := NewOCREQueryParams("hadrian", "", "", "", nil, "", 25)
	uris := make([]string, 0, 20)
	for i := 0; i < 20; i++ {
		uris = append(uris, "http://numismatics.org/ocre/id/ric.2.hdn."+string(rune('a'+i)))
	}
	got := Score(params, rows(uris...))
	if len(got) > ocreMaxCandidates {
		t.Fatalf("expected cap of %d distinct types, got %d", ocreMaxCandidates, len(got))
	}
}

func TestOCREScore_LegendBonusBounded(t *testing.T) {
	params := NewOCREQueryParams("hadrian", "", "", "", []string{"hadrian"}, "", 5)
	// The label contains the legend token "hadrian", so it earns a bounded
	// bonus but confidence must still be clamped to <= 1.
	got := Score(params, []ocreParsedRow{{TypeURI: "http://numismatics.org/ocre/id/x", Label: "RIC Hadrian denarius"}})
	if len(got) != 1 {
		t.Fatalf("expected one candidate, got %d", len(got))
	}
	if got[0].Confidence > 1.0 {
		t.Fatalf("confidence must be clamped to [0,1], got %.2f", got[0].Confidence)
	}
	found := false
	for _, f := range got[0].MatchedFields {
		if f == "legend" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected a legend matched-field marker, got %+v", got[0].MatchedFields)
	}
}

func TestOCREScore_EmptyInput(t *testing.T) {
	if got := Score(NewOCREQueryParams("hadrian", "", "", "", nil, "", 5), nil); len(got) != 0 {
		t.Fatalf("expected no candidates for empty rows, got %+v", got)
	}
}

func TestOCREScore_ExplanationBounded(t *testing.T) {
	got := Score(NewOCREQueryParams("hadrian", "denarius", "rome", "", nil, "", 5), rows("http://numismatics.org/ocre/id/x"))
	if len(got) != 1 {
		t.Fatalf("expected one candidate, got %d", len(got))
	}
	if len(got[0].Explanation) > ocreExplanationMaxLen {
		t.Fatalf("explanation exceeds %d chars: %d", ocreExplanationMaxLen, len(got[0].Explanation))
	}
	if !strings.Contains(got[0].Explanation, "ruler hadrian") {
		t.Fatalf("expected human-readable explanation, got %q", got[0].Explanation)
	}
}
