package services

import (
	"strings"
	"testing"
)

// Feature 345 T005/T033 — structural-invariance fixtures (SC-010 / FR-006).
//
// The central guarantee: for a given surviving slug set, the emitted SPARQL
// query string is byte-identical regardless of how adversarial the raw input
// values were, because any value that is not a clean slug is DROPPED, never
// interpolated, and legend/inscription free text never reaches the query.

func TestOCREQuery_AdversarialInputsProduceIdenticalSkeleton(t *testing.T) {
	// Each adversarial ruler value is a classic SPARQL/URL injection attempt.
	// Every one must fail slug validation and be dropped, so the emitted
	// query equals the query built with NO ruler at all.
	baseline := NewOCREQueryParams("", "denarius", "rome", "", nil, "", 5).BuildQuery()

	adversarial := []string{
		`hadrian> } UNION { ?type a ?x . <`,
		"hadrian\" ; DROP",
		`hadrian\nFILTER(1=1)`,
		"hadrian<script>",
		`ric.2 VALUES ?x { }`,
		"hadrian regex('.*')",
		"  ",
		"'; DELETE WHERE { ?s ?p ?o }",
	}
	for _, raw := range adversarial {
		got := NewOCREQueryParams(raw, "denarius", "rome", "", nil, "", 5).BuildQuery()
		if got != baseline {
			t.Fatalf("adversarial ruler %q changed the query skeleton:\n got: %s\nwant: %s", raw, got, baseline)
		}
		if strings.Contains(got, raw) {
			t.Fatalf("adversarial ruler %q leaked into the emitted query: %s", raw, got)
		}
	}
}

func TestOCREQuery_ValidSlugInterpolatedExactlyOnce(t *testing.T) {
	q := NewOCREQueryParams("hadrian", "", "", "", nil, "", 5).BuildQuery()
	want := "?type nmo:hasAuthority <http://nomisma.org/id/hadrian> ."
	if !strings.Contains(q, want) {
		t.Fatalf("expected bound authority triple %q in query, got: %s", want, q)
	}
	if strings.Count(q, "hadrian") != 1 {
		t.Fatalf("expected the slug to appear exactly once, got: %s", q)
	}
}

func TestOCREQuery_LegendTextNeverInQuery(t *testing.T) {
	tokens := []string{"cos", "iii", "imperator"}
	q := NewOCREQueryParams("hadrian", "denarius", "", "", tokens, "", 5).BuildQuery()
	for _, token := range tokens {
		if strings.Contains(q, token) {
			t.Fatalf("legend token %q must never appear in the SPARQL query: %s", token, q)
		}
	}
}

func TestOCREQuery_TemplateSelection(t *testing.T) {
	e := NewOCREQueryParams("hadrian", "denarius", "rome", "", nil, "", 5)
	if e.UsesTemplateK() {
		t.Fatal("expected Template E when no OCRE id is present")
	}
	if !strings.Contains(e.BuildQuery(), "nmo:TypeSeriesItem") {
		t.Fatalf("Template E must query TypeSeriesItem: %s", e.BuildQuery())
	}

	k := NewOCREQueryParams("", "", "", "", nil, "ric.2.hdn.39b", 5)
	if !k.UsesTemplateK() {
		t.Fatal("expected Template K when a valid OCRE id is present")
	}
	kq := k.BuildQuery()
	if !strings.Contains(kq, "<http://numismatics.org/ocre/id/ric.2.hdn.39b>") {
		t.Fatalf("Template K must bind the known OCRE id URI: %s", kq)
	}
	if !strings.HasSuffix(kq, "LIMIT 1") {
		t.Fatalf("Template K must request a single row: %s", kq)
	}
}

func TestOCREQuery_InvalidOCREIDDropped(t *testing.T) {
	// A non-`ric.` id (or an injection) fails the stricter id shape and is
	// dropped, so the params carry no OCRE id and fall back to Template E.
	p := NewOCREQueryParams("hadrian", "", "", "", nil, "not-a-ric-id> }", 5)
	if p.OCREIDSlug != "" {
		t.Fatalf("expected invalid OCRE id to be dropped, got %q", p.OCREIDSlug)
	}
	if p.UsesTemplateK() {
		t.Fatal("expected Template E after invalid id dropped")
	}
}

func TestOCREQuery_HasSignal(t *testing.T) {
	if NewOCREQueryParams("", "", "", "silver", []string{"cos"}, "", 5).HasSignal() {
		t.Fatal("material + legend alone must not be a sufficient signal")
	}
	if !NewOCREQueryParams("hadrian", "", "", "", nil, "", 5).HasSignal() {
		t.Fatal("a valid ruler slug is a sufficient signal")
	}
	if !NewOCREQueryParams("", "", "", "", nil, "ric.2.hdn.39b", 5).HasSignal() {
		t.Fatal("a valid OCRE id is a sufficient signal")
	}
}

func TestOCREQuery_SlugNormalization(t *testing.T) {
	// Mixed case + internal whitespace normalizes to a lower-case,
	// underscore-joined slug; an unrepresentable value is dropped.
	if got := validateOCRESlug("  Marcus Aurelius "); got != "marcus_aurelius" {
		t.Fatalf("expected normalized slug marcus_aurelius, got %q", got)
	}
	if got := validateOCRESlug("Rufus<>"); got != "" {
		t.Fatalf("expected an unrepresentable value to be dropped, got %q", got)
	}
}

func TestOCREParse_SPARQLResults(t *testing.T) {
	body := []byte(`{"head":{"vars":["type","label"]},"results":{"bindings":[
		{"type":{"type":"uri","value":"http://numismatics.org/ocre/id/ric.2.hdn.39b"},"label":{"type":"literal","xml:lang":"en","value":"RIC II Hadrian 39b"}},
		{"type":{"type":"uri","value":"http://numismatics.org/ocre/id/ric.2.hdn.40"},"label":{"type":"literal","value":""}}
	]}}`)
	rows, err := ParseOCRESPARQLResults(body)
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected the empty-label row to be skipped, got %d rows: %+v", len(rows), rows)
	}
	if rows[0].TypeURI != "http://numismatics.org/ocre/id/ric.2.hdn.39b" || rows[0].Label != "RIC II Hadrian 39b" {
		t.Fatalf("unexpected parsed row: %+v", rows[0])
	}
}

func TestOCREParse_MalformedJSON(t *testing.T) {
	if _, err := ParseOCRESPARQLResults([]byte(`{not valid`)); err == nil {
		t.Fatal("expected a parse error for malformed JSON")
	}
}
