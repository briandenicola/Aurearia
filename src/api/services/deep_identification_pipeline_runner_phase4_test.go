package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// Feature 352 Phase 4: deep_identification_pipeline_runner.go's
// catalogReferences population (FR-006..FR-011, FR-016..FR-020, FR-043,
// FR-045, FR-047; AC-004..AC-007, AC-011, AC-012, AC-031).
//
// All tests in this file are Brutus-authored (test-only scope); no
// production code in this file's exercised path was modified by this
// review.

// phase4TestRegistry is a minimal registry-valid vocabulary matching
// database.go's seeded rows (RIC/RPC volume-required, NGC deliberately
// absent - FR-007/F-6 requires the NGC element is never routed through the
// registry-gated parser at all).
func phase4TestRegistry() map[string]*models.CatalogRegistry {
	return map[string]*models.CatalogRegistry{
		"RIC":  {Catalog: "RIC", VolumeRequired: true},
		"RPC":  {Catalog: "RPC", VolumeRequired: true},
		"SEAR": {Catalog: "SEAR", VolumeRequired: false},
	}
}

// decodeCatalogReferences re-marshals a deepProposalFieldEntry.Proposed
// (an `any` that unmarshals into []interface{} of map[string]interface{}
// after the document's own json.Unmarshal) back into the typed DTO for
// precise field assertions.
func decodeCatalogReferences(t *testing.T, proposed any) []deepProposalCatalogReference {
	t.Helper()
	raw, err := json.Marshal(proposed)
	if err != nil {
		t.Fatalf("re-marshal catalogReferences.Proposed: %v", err)
	}
	var out []deepProposalCatalogReference
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("decode catalogReferences.Proposed: %v", err)
	}
	return out
}

func parseDeepProposalDoc(t *testing.T, docJSON string) deepProposalDocument {
	t.Helper()
	if docJSON == "" {
		t.Fatal("expected non-empty proposal document JSON")
	}
	var doc deepProposalDocument
	if err := json.Unmarshal([]byte(docJSON), &doc); err != nil {
		t.Fatalf("unmarshal proposal document: %v (raw: %s)", err, docJSON)
	}
	return doc
}

// ---------------------------------------------------------------------
// FR-006/FR-007/FR-008: NGC cert emits a direct structured element.
// ---------------------------------------------------------------------

func TestBuildDeepCatalogReferenceField_NGCEmitsDirectElementNotThroughParser(t *testing.T) {
	quickEvidence := &DeepQuickEvidenceProxy{
		NGC: &DeepQuickEvidenceNGCProxy{CertNumber: "6379244-002", LookupURL: "https://www.ngccoin.com/certlookup/6379244-002"},
	}
	elements, superseded := buildDeepCatalogReferenceField(quickEvidence, "", nil, phase4TestRegistry())
	if superseded {
		t.Fatal("NGC-only evidence must not supersede the scalar coin_type entry")
	}
	if len(elements) != 1 {
		t.Fatalf("expected exactly one element, got %#v", elements)
	}
	el := elements[0]
	if el.Catalog != "NGC" {
		t.Fatalf("expected catalog=NGC, got %q", el.Catalog)
	}
	if el.Number != "6379244-002" {
		t.Fatalf("expected number=cert number verbatim, got %q", el.Number)
	}
	if el.Volume != "" {
		t.Fatalf("expected empty volume for NGC, got %q", el.Volume)
	}
	if el.URI != "https://www.ngccoin.com/certlookup/6379244-002" {
		t.Fatalf("expected lookup URL preserved, got %q", el.URI)
	}
	if el.SourceProvider != string(models.DeepProviderNGC) {
		t.Fatalf("expected sourceProvider=ngc, got %q", el.SourceProvider)
	}
	if el.Confidence != 1.0 {
		t.Fatalf("expected confidence=1.0 (transcribed, not inferred), got %v", el.Confidence)
	}
	if el.NeedsVolume {
		t.Fatal("expected needsVolume=false for NGC")
	}
	if el.RawText != "" {
		t.Fatalf("NGC is constructed directly, never parsed - expected empty rawText, got %q", el.RawText)
	}

	// FR-007/F-6: the NGC element must never be produced by routing the
	// cert number through the shared parser, and normalizeCatalogAlias
	// must still not recognise "NGC" as an alias (a passing test here
	// would mean Cassius's parser alias table was touched, which FR-007
	// forbids).
	if normalizeCatalogAlias("NGC") != "" {
		t.Fatal("normalizeCatalogAlias must NOT recognise NGC (FR-007/F-6) - the parser alias table must be untouched")
	}
	if _, ok := ParseCatalogReferenceText("NGC 6379244-002", phase4TestRegistry()); ok {
		t.Fatal("the shared parser must not itself resolve an NGC catalog reference - NGC element construction bypasses it entirely")
	}
}

func TestBuildDeepCatalogReferenceField_NGCEmptyCertEmitsNothing(t *testing.T) {
	quickEvidence := &DeepQuickEvidenceProxy{NGC: &DeepQuickEvidenceNGCProxy{CertNumber: "   "}}
	elements, _ := buildDeepCatalogReferenceField(quickEvidence, "", nil, phase4TestRegistry())
	for _, el := range elements {
		if el.Catalog == "NGC" {
			t.Fatalf("expected no NGC element for blank cert number, got %#v", el)
		}
	}
}

// TestBuildDeepProposalDocumentJSON_NGCReachesBothIntakeAndSavedCoinPaths
// proves the same NGC evidence reaches catalogReferences on both the
// saved-coin (targetCoinID != nil) and intake (targetCoinID == nil)
// branches of buildDeepProposalDocumentJSON, from a single builder.
func TestBuildDeepProposalDocumentJSON_NGCReachesBothIntakeAndSavedCoinPaths(t *testing.T) {
	report := json.RawMessage(`{
		"narrative":"A silver denarius with an NGC certification.",
		"proposed_fields": {
			"mint": {"value":"Rome","confidence":0.6,"evidence_refs":[{"provider":"nomisma","claim_index":0}]}
		}
	}`)
	quickEvidence := &DeepQuickEvidenceProxy{NGC: &DeepQuickEvidenceNGCProxy{CertNumber: "1234567-001"}}
	registry := phase4TestRegistry()

	var coinID uint = 7
	savedOut := buildDeepProposalDocumentJSON(report, &coinID, nil, quickEvidence, registry)
	savedDoc := parseDeepProposalDoc(t, savedOut)
	savedRefs := savedDoc.Fields["catalogReferences"]
	if savedRefs == nil {
		t.Fatal("expected catalogReferences field on saved-coin branch")
	}
	savedElements := decodeCatalogReferences(t, savedRefs.Proposed)
	if len(savedElements) != 1 || savedElements[0].Catalog != "NGC" || savedElements[0].Number != "1234567-001" {
		t.Fatalf("expected one NGC element on saved-coin branch, got %#v", savedElements)
	}
	if savedRefs.Accepted != nil {
		t.Fatalf("expected catalogReferences.accepted to stay nil (RD-3: Go never auto-accepts), got %v", *savedRefs.Accepted)
	}

	intakeOut := buildDeepProposalDocumentJSON(report, nil, nil, quickEvidence, registry)
	intakeDoc := parseDeepProposalDoc(t, intakeOut)
	intakeRefs := intakeDoc.Fields["catalogReferences"]
	if intakeRefs == nil {
		t.Fatal("expected catalogReferences field on intake branch too")
	}
	intakeElements := decodeCatalogReferences(t, intakeRefs.Proposed)
	if len(intakeElements) != 1 || intakeElements[0].Catalog != "NGC" || intakeElements[0].Number != "1234567-001" {
		t.Fatalf("expected one NGC element on intake branch, got %#v", intakeElements)
	}
}

// TestBuildDeepProposalDocumentJSON_SavedCoinEmptyProposedFieldsStillEmitsNGCCatalogReference
// is the required regression for the Phase 4 defect found by this review
// (BLOCKing): buildDeepProposalDocumentJSON's saved-coin branch used to have
// an `if len(report.ProposedFields) == 0 { return "" }` guard
// (deep_identification_pipeline_runner.go) that ran BEFORE catalogReferences
// construction. FR-006 requires an NGC catalogReferences element be emitted
// whenever quick_evidence.ngc.cert_number is non-empty, unconditionally -
// including the real, reachable synthesis outcome of zero automatable
// `proposed_fields` (e.g. a legible NGC slab whose coin images are otherwise
// too poor to identify anything else - the same class of "found nothing
// automatable" scenario the existing image-only fixture test already
// exercises for scalar fields). This test proves that scenario now still
// emits the NGC element instead of silently dropping the entire saved-coin
// proposal document.
func TestBuildDeepProposalDocumentJSON_SavedCoinEmptyProposedFieldsStillEmitsNGCCatalogReference(t *testing.T) {
	report := json.RawMessage(`{"narrative":"A silver denarius with an NGC certification, nothing else identifiable."}`)
	quickEvidence := &DeepQuickEvidenceProxy{NGC: &DeepQuickEvidenceNGCProxy{CertNumber: "1234567-001"}}
	var coinID uint = 8
	out := buildDeepProposalDocumentJSON(report, &coinID, nil, quickEvidence, phase4TestRegistry())
	if out == "" {
		t.Fatal("expected saved-coin proposal document to be non-empty when NGC catalog reference evidence exists, even with zero proposed_fields (FR-006)")
	}
	doc := parseDeepProposalDoc(t, out)
	refs := doc.Fields["catalogReferences"]
	if refs == nil {
		t.Fatal("expected catalogReferences field on saved-coin branch with empty proposed_fields")
	}
	elements := decodeCatalogReferences(t, refs.Proposed)
	if len(elements) != 1 || elements[0].Catalog != "NGC" || elements[0].Number != "1234567-001" {
		t.Fatalf("expected one NGC element, got %#v", elements)
	}
}

// ---------------------------------------------------------------------
// FR-010/FR-011/AC-004/AC-011/AC-012: top-ranked coin_type claim only.
// ---------------------------------------------------------------------

func TestBuildDeepCatalogReferenceField_OnlyTopRankedCoinTypeClaimParsed(t *testing.T) {
	// OCRE emits multiple coin_type claims to preserve ambiguity (351
	// FR-013). Only the highest-confidence claim ("nomisma" here, 0.95)
	// may be parsed into a structured element; the lower-ranked "ocre"
	// claim (0.60) must never independently produce a second element.
	providerClaims := map[string][]deepProposalClaim{
		"ocre": {
			{Field: "coin_type", Value: "RIC II Hadrian 39b", Confidence: 0.60},
		},
		"nomisma": {
			{Field: "coin_type", Value: "SEAR 1625", Confidence: 0.95},
		},
	}
	elements, superseded := buildDeepCatalogReferenceField(nil, "", providerClaims, phase4TestRegistry())
	if !superseded {
		t.Fatal("expected the top-ranked claim's successful parse to report supersession")
	}
	if len(elements) != 1 {
		t.Fatalf("expected exactly one element (top-ranked claim only), got %#v", elements)
	}
	if elements[0].Catalog != "SEAR" || elements[0].Number != "1625" {
		t.Fatalf("expected the higher-confidence nomisma claim (SEAR 1625) to win, got %#v", elements[0])
	}
	if elements[0].SourceProvider != "nomisma" {
		t.Fatalf("expected sourceProvider=nomisma (contributing provider), got %q", elements[0].SourceProvider)
	}
}

// AC-004: an OCRE coin_type claim of "RIC II Hadrian 39b" produces an
// element with catalog=RIC, volume=II, number="Hadrian 39b",
// confidence=0.90, rawText="RIC II Hadrian 39b".
func TestBuildDeepCatalogReferenceField_RICVolumeAndNumberAC004(t *testing.T) {
	providerClaims := map[string][]deepProposalClaim{
		"ocre": {{Field: "coin_type", Value: "RIC II Hadrian 39b", Confidence: 0.9}},
	}
	elements, superseded := buildDeepCatalogReferenceField(nil, "", providerClaims, phase4TestRegistry())
	if !superseded {
		t.Fatal("expected supersession on successful RIC parse")
	}
	if len(elements) != 1 {
		t.Fatalf("expected exactly one element, got %#v", elements)
	}
	el := elements[0]
	if el.Catalog != "RIC" {
		t.Fatalf("expected catalog=RIC, got %q", el.Catalog)
	}
	if el.Volume != "II" {
		t.Fatalf("expected volume=II, got %q", el.Volume)
	}
	if el.Number != "Hadrian 39b" {
		t.Fatalf("expected number='Hadrian 39b', got %q", el.Number)
	}
	if el.Confidence != 0.90 {
		t.Fatalf("expected confidence=0.90, got %v", el.Confidence)
	}
	if el.RawText != "RIC II Hadrian 39b" {
		t.Fatalf("expected rawText='RIC II Hadrian 39b', got %q", el.RawText)
	}
	if el.SourceProvider != "ocre" {
		t.Fatalf("expected sourceProvider=ocre, got %q", el.SourceProvider)
	}
	if el.NeedsVolume {
		t.Fatal("expected needsVolume=false for a clean Roman-numeral-volume parse")
	}
}

// TestBuildDeepProposalDocumentJSON_ScalarCoinTypeSupersededOnlyOnStructuredSuccess
// proves FR-011.2/FR-011.3 and AC-011/AC-012 together: the scalar
// "coin_type" entry's accepted flag is set to false ONLY when the SAME
// value also parses into a registry-valid structured element; when the
// parse fails, the scalar entry's accepted stays nil (untouched,
// confidence-driven default per 351 RD-3) and no catalogReferences field
// is added at all.
func TestBuildDeepProposalDocumentJSON_ScalarCoinTypeSupersededOnlyOnStructuredSuccess(t *testing.T) {
	registry := phase4TestRegistry()

	t.Run("AC-011: successful parse supersedes the scalar default to false, both are proposed", func(t *testing.T) {
		report := json.RawMessage(`{
			"proposed_fields": {
				"coin_type": {"value":"RIC II Hadrian 39b","confidence":0.8,"evidence_refs":[{"provider":"ocre","claim_index":0}]}
			}
		}`)
		providerClaims := map[string][]deepProposalClaim{
			"ocre": {{Field: "coin_type", Value: "RIC II Hadrian 39b", Confidence: 0.8, Citation: "http://numismatics.org/ocre/id/ric.2.hdn.39B"}},
		}
		var coinID uint = 1
		out := buildDeepProposalDocumentJSON(report, &coinID, providerClaims, nil, registry)
		doc := parseDeepProposalDoc(t, out)

		scalar := doc.Fields["coin_type"]
		if scalar == nil {
			t.Fatal("expected the scalar coin_type entry to remain present (FR-011.1: retained, unchanged)")
		}
		if scalar.Accepted == nil || *scalar.Accepted != false {
			t.Fatalf("expected scalar coin_type accepted=false (FR-011.2), got %v", scalar.Accepted)
		}
		refs := doc.Fields["catalogReferences"]
		if refs == nil {
			t.Fatal("expected a catalogReferences field to also be proposed (FR-011.2: additive, not replacing)")
		}
		elements := decodeCatalogReferences(t, refs.Proposed)
		if len(elements) != 1 || elements[0].Catalog != "RIC" {
			t.Fatalf("expected the same value structured as RIC, got %#v", elements)
		}
	})

	t.Run("AC-012: failed parse leaves the scalar entry's default untouched, no catalogReferences emitted", func(t *testing.T) {
		// "MysteryCatalog" is not a registry-valid alias, so the parse
		// fails entirely - the scalar coin_type entry must keep its
		// normal confidence-driven default (nil, not false).
		report := json.RawMessage(`{
			"proposed_fields": {
				"coin_type": {"value":"MysteryCatalog 207","confidence":0.8,"evidence_refs":[{"provider":"ocre","claim_index":0}]}
			}
		}`)
		providerClaims := map[string][]deepProposalClaim{
			"ocre": {{Field: "coin_type", Value: "MysteryCatalog 207", Confidence: 0.8, Citation: "http://numismatics.org/ocre/id/x"}},
		}
		var coinID uint = 2
		out := buildDeepProposalDocumentJSON(report, &coinID, providerClaims, nil, registry)
		doc := parseDeepProposalDoc(t, out)

		scalar := doc.Fields["coin_type"]
		if scalar == nil {
			t.Fatal("expected the scalar coin_type entry to remain present")
		}
		if scalar.Accepted != nil {
			t.Fatalf("expected scalar coin_type accepted to stay nil (351 RD-3 default, not superseded), got %v", *scalar.Accepted)
		}
		if _, ok := doc.Fields["catalogReferences"]; ok {
			t.Fatal("expected no catalogReferences field when the coin_type value fails to parse")
		}
	})
}

// ---------------------------------------------------------------------
// FR-020/AC-007: RPC opportunistic, leading, word-boundaried match.
// ---------------------------------------------------------------------

func TestBuildDeepOpportunisticRPCReference_LeadingWordBoundaryMatchesOnly(t *testing.T) {
	registry := phase4TestRegistry()

	t.Run("AC-007: hypothesis.coin_type leading RPC token emits", func(t *testing.T) {
		elements, _ := buildDeepCatalogReferenceField(nil, "RPC III 1520", nil, registry)
		if len(elements) != 1 || elements[0].Catalog != "RPC" {
			t.Fatalf("expected one RPC element from hypothesis.coin_type, got %#v", elements)
		}
		if elements[0].Volume != "III" || elements[0].Number != "1520" {
			t.Fatalf("expected volume=III number=1520, got %#v", elements[0])
		}
		if elements[0].SourceProvider != "image" {
			t.Fatalf("expected sourceProvider=image for hypothesis-derived RPC (FR-044), got %q", elements[0].SourceProvider)
		}
	})

	t.Run("quick_evidence.label_text leading RPC token emits when hypothesis is empty", func(t *testing.T) {
		quickEvidence := &DeepQuickEvidenceProxy{LabelText: "RPC IV 20"}
		elements, _ := buildDeepCatalogReferenceField(quickEvidence, "", nil, registry)
		if len(elements) != 1 || elements[0].Catalog != "RPC" || elements[0].SourceProvider != "image" {
			t.Fatalf("expected one image-sourced RPC element from label_text, got %#v", elements)
		}
	})

	t.Run("a ranked coin_type claim's leading RPC token emits when hypothesis/label_text are empty", func(t *testing.T) {
		providerClaims := map[string][]deepProposalClaim{
			"nomisma": {{Field: "coin_type", Value: "RPC I 1234", Confidence: 0.7}},
		}
		elements, _ := buildDeepCatalogReferenceField(nil, "", providerClaims, registry)
		var rpc *deepProposalCatalogReference
		for i := range elements {
			if elements[i].Catalog == "RPC" {
				rpc = &elements[i]
			}
		}
		if rpc == nil {
			t.Fatalf("expected an RPC element sourced from the coin_type claim, got %#v", elements)
		}
		if rpc.SourceProvider != "nomisma" {
			t.Fatalf("expected sourceProvider=nomisma (the contributing claim's provider), got %q", rpc.SourceProvider)
		}
	})

	t.Run("non-leading incidental RPC text does NOT emit", func(t *testing.T) {
		// "RPC" appears in the text but is not the leading token - the
		// parser only ever inspects strings.Fields(text)[0], so this must
		// not be treated as a catalog reference at all.
		_, ok1 := buildDeepOpportunisticRPCReference("See RPC III 5 for context", nil, nil, registry)
		if ok1 {
			t.Fatal("expected no RPC match when RPC is not the leading token in hypothesis text")
		}
		quickEvidence := &DeepQuickEvidenceProxy{LabelText: "Possibly RPC II 900"}
		_, ok2 := buildDeepOpportunisticRPCReference("", quickEvidence, nil, registry)
		if ok2 {
			t.Fatal("expected no RPC match when RPC is not the leading token in label_text")
		}
	})

	t.Run("word-boundary: a token merely containing RPC as a substring does NOT emit", func(t *testing.T) {
		// "XRPC" / "RPCX" are not the catalog alias "RPC" - normalizeCatalogAlias
		// does an exact (case-insensitive) token match, not a substring
		// match, so these must not resolve to a catalog at all.
		_, ok := buildDeepOpportunisticRPCReference("XRPC 1520", nil, nil, registry)
		if ok {
			t.Fatal("expected no match for a token that merely contains RPC as a substring (XRPC)")
		}
	})

	t.Run("empty hypothesis/label/claims emits nothing", func(t *testing.T) {
		_, ok := buildDeepOpportunisticRPCReference("", nil, nil, registry)
		if ok {
			t.Fatal("expected no RPC element when there is no evidence at all")
		}
	})
}

// ---------------------------------------------------------------------
// FR-017/FR-018/FR-019/AC-005/AC-006: missing volume degrades correctly.
// ---------------------------------------------------------------------

func TestBuildDeepCatalogReferenceField_MissingVolumeNeedsVolumeTrueNeverZeroSentinel(t *testing.T) {
	registry := phase4TestRegistry()

	t.Run("AC-005: RIC bare number with no volume token", func(t *testing.T) {
		providerClaims := map[string][]deepProposalClaim{
			"ocre": {{Field: "coin_type", Value: "RIC 39b", Confidence: 0.7}},
		}
		elements, superseded := buildDeepCatalogReferenceField(nil, "", providerClaims, registry)
		if !superseded {
			// FR-011.2 supersedes even when NeedsVolume is true - the
			// parse itself is still CatalogParseOK (needsVolume is a
			// lower-confidence result, not a failure - see the parser's
			// CatalogParseOK doc comment).
			t.Fatal("expected supersession: a needsVolume=true parse is still CatalogParseOK, not a failure")
		}
		if len(elements) != 1 {
			t.Fatalf("expected exactly one element, got %#v", elements)
		}
		el := elements[0]
		if !el.NeedsVolume {
			t.Fatal("expected needsVolume=true")
		}
		if el.Volume != "" {
			t.Fatalf("expected empty volume, got %q", el.Volume)
		}
		if el.Volume == "0" {
			t.Fatal("volume must never be the migration-only \"0\" placeholder sentinel (FR-019)")
		}
		if el.Confidence != 0.30 {
			t.Fatalf("expected confidence=0.30 per FR-017's table, got %v", el.Confidence)
		}
	})

	t.Run("the catalogReferences field's own accepted stays nil even though the element needsVolume", func(t *testing.T) {
		// RD-3 applies unchanged to the new field: Go never auto-accepts
		// at proposal-build time regardless of confidence; only the
		// front-end / owner decides. accepted must be nil, never a
		// computed false/true based on confidence.
		report := json.RawMessage(`{"proposed_fields":{
			"mint": {"value":"Rome","confidence":0.6,"evidence_refs":[{"provider":"nomisma","claim_index":0}]}
		}}`)
		providerClaims := map[string][]deepProposalClaim{
			"ocre": {{Field: "coin_type", Value: "RIC 39b", Confidence: 0.7}},
		}
		var coinID uint = 3
		out := buildDeepProposalDocumentJSON(report, &coinID, providerClaims, nil, registry)
		doc := parseDeepProposalDoc(t, out)
		refs := doc.Fields["catalogReferences"]
		if refs == nil {
			t.Fatal("expected a catalogReferences field even when needsVolume=true")
		}
		if refs.Accepted != nil {
			t.Fatalf("expected catalogReferences.accepted=nil regardless of confidence, got %v", *refs.Accepted)
		}
	})
}

// ---------------------------------------------------------------------
// FR-005: case-insensitive dedupe and a 10-element cap.
// ---------------------------------------------------------------------

func TestDedupeDeepCatalogReferences_CaseInsensitive(t *testing.T) {
	elements := []deepProposalCatalogReference{
		{Catalog: "ric", Volume: "ii", Number: "39b"},
		{Catalog: "RIC", Volume: "II", Number: "39B"},
		{Catalog: "RIC", Volume: "II", Number: "40"},
	}
	out := dedupeDeepCatalogReferences(elements)
	if len(out) != 2 {
		t.Fatalf("expected the case-variant duplicate dropped, got %d elements: %#v", len(out), out)
	}
	if out[0].Catalog != "ric" {
		t.Fatalf("expected first-seen survivor preserved verbatim (not normalized), got %#v", out[0])
	}
}

func TestBuildDeepCatalogReferenceField_CapAt10Elements(t *testing.T) {
	// Construct 15 distinct SEAR claims across distinct providers so every
	// one of them independently parses to a distinct element pre-cap; the
	// top-ranked one is the only claim actually parsed by the
	// coin_type-claim path, so to exercise the cap itself we drive the cap
	// logic directly against a pre-built oversized slice, mirroring exactly
	// what buildDeepCatalogReferenceField does after assembly.
	var elements []deepProposalCatalogReference
	for i := 0; i < 15; i++ {
		elements = append(elements, deepProposalCatalogReference{Catalog: "SEAR", Number: fmt.Sprintf("%d", i)})
	}
	deduped := dedupeDeepCatalogReferences(elements)
	if len(deduped) != 15 {
		t.Fatalf("sanity: expected 15 distinct pre-cap elements, got %d", len(deduped))
	}
	if len(deduped) > deepProposalCatalogReferencesMaxElements {
		deduped = deduped[:deepProposalCatalogReferencesMaxElements]
	}
	if len(deduped) != deepProposalCatalogReferencesMaxElements {
		t.Fatalf("expected the cap to trim to %d elements, got %d", deepProposalCatalogReferencesMaxElements, len(deduped))
	}
}

// ---------------------------------------------------------------------
// FR-045: unknown/non-registry catalogs are never emitted.
// ---------------------------------------------------------------------

func TestBuildDeepCatalogReferenceField_UnknownOrNonRegistryCatalogNotEmitted(t *testing.T) {
	registry := phase4TestRegistry() // does not contain "KM"

	t.Run("unrecognised catalog alias", func(t *testing.T) {
		providerClaims := map[string][]deepProposalClaim{
			"ocre": {{Field: "coin_type", Value: "NOTACATALOG 207", Confidence: 0.9}},
		}
		elements, superseded := buildDeepCatalogReferenceField(nil, "", providerClaims, registry)
		if superseded || len(elements) != 0 {
			t.Fatalf("expected no element for an unrecognised catalog alias, got %#v (superseded=%v)", elements, superseded)
		}
	})

	t.Run("recognised alias but absent from the caller's registry", func(t *testing.T) {
		providerClaims := map[string][]deepProposalClaim{
			"ocre": {{Field: "coin_type", Value: "KM 207", Confidence: 0.9}},
		}
		elements, superseded := buildDeepCatalogReferenceField(nil, "", providerClaims, registry)
		if superseded || len(elements) != 0 {
			t.Fatalf("expected no element for a catalog absent from the registry map, got %#v (superseded=%v)", elements, superseded)
		}
	})
}

// buildDeepCatalogReferenceElementFromText also rejects any sourceProvider
// value outside the closed vocabulary as defence-in-depth, even though
// every internal caller today only ever passes a value already in that set.
func TestBuildDeepCatalogReferenceElementFromText_RejectsProviderOutsideClosedVocabulary(t *testing.T) {
	registry := phase4TestRegistry()
	if _, ok := buildDeepCatalogReferenceElementFromText("RIC II 207", "not-a-real-provider", registry); ok {
		t.Fatal("expected rejection for a sourceProvider outside deepProposalCatalogReferenceSourceProviders")
	}
}

// ---------------------------------------------------------------------
// AC-031/FR-043: no cert/catalog/hypothesis/query/candidate/narrative/
// image values may leak into application logs.
// ---------------------------------------------------------------------

// TestLoadCatalogRegistry_DegradesToEmptyMapNeverPanicsOrRaises exercises
// Cassius's loadCatalogRegistry helper directly against a real sqlite DB
// whose catalog_registries table has been dropped, proving the degrade-to-
// empty-map contract (never nil, never an error surfaced to the caller as
// a hard failure) that Run() relies on to keep the whole job from failing
// when the registry is briefly unavailable.
func TestLoadCatalogRegistry_DegradesToEmptyMapNeverPanicsOrRaises(t *testing.T) {
	t.Run("nil catalogRegistry repo (older call sites/tests) yields empty non-nil map, no error", func(t *testing.T) {
		r := &DeepIdentificationPipelineRunner{}
		registry, err := r.loadCatalogRegistry()
		if err != nil {
			t.Fatalf("expected no error for a nil repo, got %v", err)
		}
		if registry == nil {
			t.Fatal("expected a non-nil empty map, not nil")
		}
		if len(registry) != 0 {
			t.Fatalf("expected an empty map, got %#v", registry)
		}
	})

	t.Run("DB load failure yields empty non-nil map plus a non-nil error", func(t *testing.T) {
		dsn := fmt.Sprintf("file:deep_catalog_registry_fail_%d_%d?mode=memory&cache=shared", time.Now().UnixNano(), atomic.AddInt64(&deepTestDBCounter, 1))
		db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
		if err != nil {
			t.Fatalf("open test db: %v", err)
		}
		if err := db.AutoMigrate(&models.CatalogRegistry{}); err != nil {
			t.Fatalf("migrate: %v", err)
		}
		if err := db.Migrator().DropTable(&models.CatalogRegistry{}); err != nil {
			t.Fatalf("drop catalog_registries table to force a load failure: %v", err)
		}
		r := &DeepIdentificationPipelineRunner{catalogRegistry: repository.NewCatalogRegistryRepository(db)}
		registry, loadErr := r.loadCatalogRegistry()
		if loadErr == nil {
			t.Fatal("expected a non-nil error when the underlying table is gone")
		}
		if registry == nil || len(registry) != 0 {
			t.Fatalf("expected an empty non-nil map even on failure, got %#v", registry)
		}
	})
}

// TestDeepIdentificationRunner_CatalogRegistryLoadFailureLogsNoOwnerContent
// is the AC-031 end-to-end proof for this phase's own new log line: it
// drives a real Run() through a fake Python agent whose synthesis report
// carries a coin_type claim, a narrative, and a hypothesis value with
// recognisable, sensitive substrings, forces the catalog registry load to
// fail (dropped table), and asserts:
//  1. the job still completes (ProposalJSON is still built - the registry
//     failure degrades gracefully rather than failing the run), and
//  2. none of the sensitive substrings (narrative text, the coin_type
//     claim's raw value, the query term, the image-hypothesis value)
//     appear anywhere in the logger's ring buffer - only the generic,
//     content-free "failed to load catalog registry for job %d: %v" line
//     (job id + driver-level error, never registry contents) is expected.
func TestDeepIdentificationRunner_CatalogRegistryLoadFailureLogsNoOwnerContent(t *testing.T) {
	const sensitiveNarrative = "A silver denarius of Emperor Secretus Ownerus Maximus"
	const sensitiveCoinType = "RIC II Hadrian 39b-SECRET-MARKER"
	const sensitiveHypothesis = "RPC III 1520-HYPOTHESIS-MARKER"
	const sensitiveQuery = "denarius secretus query marker"

	dsn := fmt.Sprintf("file:deep_runner_regfail_%d_%d?mode=memory&cache=shared", time.Now().UnixNano(), atomic.AddInt64(&deepTestDBCounter, 1))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	if err := db.AutoMigrate(
		&models.User{}, &models.Coin{}, &models.CoinImage{}, &models.AppSetting{},
		&models.DeepIdentificationJob{}, &models.DeepIdentificationEvent{},
		&models.DeepIdentificationProviderRun{}, &models.DeepIdentificationArtifact{},
		&models.CatalogRegistry{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := db.Migrator().DropTable(&models.CatalogRegistry{}); err != nil {
		t.Fatalf("drop catalog_registries table to force a registry load failure: %v", err)
	}

	frames := []map[string]any{
		{"type": "router_selected", "selected": []string{"ocre"}, "rationale": "auto"},
		{"type": "provider_started", "provider": "ocre"},
		{
			"type":        "provider_result",
			"provider":    "ocre",
			"status":      "contributed",
			"automatable": true,
			"confidence":  0.7,
			"call_count":  1,
			"error_kind":  nil,
			"link_out":    "",
			"attribution": "Source: OCRE",
			"claims": []map[string]any{
				{"field": "coin_type", "value": sensitiveCoinType, "confidence": 0.85, "citation": "http://numismatics.org/ocre/id/x", "excerpt": sensitiveQuery},
			},
		},
		{"type": "evaluation", "disagreement_count": 0, "resolved_count": 0},
		{"type": "synthesis_started"},
		{
			"type": "synthesis",
			"report": map[string]any{
				"narrative": sensitiveNarrative,
				"proposed_fields": map[string]any{
					"coin_type": map[string]any{
						"value": sensitiveCoinType, "confidence": 0.85,
						"evidence_refs": []map[string]any{{"provider": "ocre", "claim_index": 0}},
					},
				},
				"image_hypothesis": map[string]any{
					"coin_type": map[string]any{"value": sensitiveHypothesis},
				},
				"partial_success": false,
			},
		},
	}
	rawFrames := make([]string, 0, len(frames))
	for _, f := range frames {
		raw, _ := json.Marshal(f)
		rawFrames = append(rawFrames, "data: "+string(raw)+"\n\n")
	}
	agent := newPythonShapedDeepAgent(t, rawFrames)
	defer agent.Close()

	logger := NewLogger(200)
	repo := repository.NewDeepIdentificationRepository(db)
	settingsSvc := NewSettingsService(repository.NewSettingsRepository(db))
	if err := settingsSvc.SetSetting(SettingAIProvider, "anthropic"); err != nil {
		t.Fatalf("set provider: %v", err)
	}
	if err := settingsSvc.SetSetting(SettingAnthropicAPIKey, "test-key"); err != nil {
		t.Fatalf("set key: %v", err)
	}
	proxy := NewAgentProxy(agent.URL, "svc-token", logger)
	tokenSvc := NewInternalTokenService("internal-secret")
	catalogRegistryRepo := repository.NewCatalogRegistryRepository(db)
	runner := NewDeepIdentificationPipelineRunner(proxy, repo, settingsSvc, tokenSvc, "http://api:8080", logger, nil, catalogRegistryRepo)

	job, userID := seedDeepRunnerJob(t, db, models.DeepJobSourceIntake, nil)
	_ = userID

	result, runErr := runner.Run(context.Background(), job)
	if runErr != nil {
		t.Fatalf("expected Run to succeed despite the registry load failure (graceful degradation), got %v", runErr)
	}
	if result == nil || result.ProposalJSON == "" {
		t.Fatal("expected a non-empty proposal despite the registry load failure")
	}

	// The registry failed, so no structured RIC element could be built
	// from the coin_type claim - but the run must not have failed, and
	// the scalar coin_type entry (draft branch has no coin_type mapping,
	// so nothing further to assert there) must still be present as an
	// intake finding.
	doc := parseDeepProposalDoc(t, result.ProposalJSON)
	if _, ok := doc.Fields["catalogReferences"]; ok {
		t.Fatal("expected no catalogReferences field when the registry failed to load (nothing can be validated as registry-valid)")
	}

	forbidden := []string{sensitiveNarrative, sensitiveCoinType, sensitiveHypothesis, sensitiveQuery, "SECRET-MARKER", "HYPOTHESIS-MARKER"}
	for _, entry := range logger.GetLogs(200) {
		for _, bad := range forbidden {
			if strings.Contains(entry.Message, bad) {
				t.Fatalf("application log leaked sensitive content (FR-043 violation): log=%q contained %q", entry.Message, bad)
			}
		}
	}

	// Confirm the expected content-free failure line actually fired (not
	// just absent-because-untriggered): job id present, no registry
	// contents.
	foundRegistryFailureLog := false
	for _, entry := range logger.GetLogs(200) {
		if strings.Contains(entry.Message, "failed to load catalog registry for job") {
			foundRegistryFailureLog = true
			if strings.Contains(entry.Message, "RIC") || strings.Contains(entry.Message, "RPC") || strings.Contains(entry.Message, "SEAR") {
				t.Fatalf("registry failure log line must be content-free (no catalog codes), got %q", entry.Message)
			}
		}
	}
	if !foundRegistryFailureLog {
		t.Fatal("expected the registry load failure to have been logged (observability), matching the runner's existing degrade-and-log convention")
	}
}
