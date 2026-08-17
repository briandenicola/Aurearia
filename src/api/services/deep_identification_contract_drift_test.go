package services

// T107 [F4, FR-035]: contract-drift guard for the Go<->Python wire contract
// mirrored by DeepIdentifyRequest (Go: DeepIdentifyProxyRequest + friends,
// defined in agent_proxy_deep_identify.go and agent_proxy.go) and
// DeepSynthesis (Python: app/models/responses.py).
//
// Five field-shape drift points were fixed (documentation-only, T093) in
// specs/344-deep-agentic-coin-identification/contracts/agent-internal-
// contract.md:
//
//  1. §1 Mint(userID)/InternalTokenRequired -> MintForJob(userID, jobID)/
//     InternalJobTokenRequired - a token-minting *function signature*, not a
//     field of DeepIdentifyRequest or DeepSynthesis. OUT OF SCOPE for this
//     test; it cannot be expressed as a field-name/nullability comparison
//     between these two models. Not caught here - flagged explicitly per the
//     T107 task instructions rather than silently ignored.
//  2. §2 `llm_config` -> `llm` (DeepIdentifyRequest field rename). CAUGHT:
//     see TestDeepIdentifyRequestContractFieldsMatchPython, which compares
//     DeepIdentifyProxyRequest.LLM's json tag against the Pydantic
//     DeepIdentifyRequest.llm property name.
//  3. §2 deletion of the never-real `quick_evidence.numista_evidence` line
//     (QuickEvidence is `extra="forbid"`, so Go adding it back would be
//     rejected at the wire). CAUGHT (defense in depth): the request-side
//     comparison flags any Go field with no matching Pydantic property name,
//     which is exactly the shape this drift would take if it reappeared.
//  4. §3 `evaluation` frame payload -> `{disagreement_count, resolved_count}`
//     plus the missing `synthesis_started` row. This is an SSE *frame*
//     contract (contracts/sse-events.md), not a field of DeepIdentifyRequest
//     or DeepSynthesis. OUT OF SCOPE for this test; not caught here.
//  5. §5 add `attributions` to the DeepSynthesis example. PARTIALLY CAUGHT:
//     see TestDeepSynthesisKnownTopLevelFieldsMatchPython, a documented
//     non-mechanical fallback (Go has no single struct that types the full
//     DeepSynthesis shape - see rationale below) that pins the exact
//     top-level property set, including `attributions`, so a removal or
//     rename of it goes red.
//  6. §7 add the `ocre_search` row. A tool/provider-catalog documentation
//     table entry, not a field of DeepIdentifyRequest or DeepSynthesis. OUT
//     OF SCOPE for this test; not caught here.
//
// In short: of the five T093 drift points, #2 and #5 are the ones that are
// actually shaped like a DeepIdentifyRequest/DeepSynthesis field-name or
// nullability drift, and this file mechanically guards both (plus #3 as a
// side effect of the request-side comparison). #1, #3(frame contract), and
// #6 live on other wire surfaces (token minting, SSE frames, tool catalog
// docs) that this test is not positioned to cover - a future contract-drift
// test for those surfaces would need its own comparison target.
//
// Comparison mechanism
// ---------------------
// The Python side of the comparison is never hand-typed in this file. It is
// read from testdata/deep_identify_contract_schema.json, a JSON Schema
// artifact produced by calling the *actual* Pydantic models' own
// `model_json_schema()` - i.e. Pydantic describing its own contract, not a
// second human transcription of it. Regenerate it after any change to
// DeepIdentifyRequest/DeepSynthesis (or any model they reference) by running,
// from src/agent/:
//
//	python -c "import json; from app.models.requests import DeepIdentifyRequest; from app.models.responses import DeepSynthesis; json.dump({'_generated_by': 'src/agent (python -m app.models.requests/responses .model_json_schema()) - see src/api/services/deep_identification_contract_drift_test.go header for regeneration command', 'DeepIdentifyRequest': DeepIdentifyRequest.model_json_schema(), 'DeepSynthesis': DeepSynthesis.model_json_schema()}, open('../api/services/testdata/deep_identify_contract_schema.json', 'w'), indent=2, sort_keys=True)"
//
// The Go side is read via reflection over the real, shipped mirror structs
// (reflect.TypeOf + `json` struct tags), never a duplicate hand-maintained
// field list.
//
// What this test CANNOT catch, and why:
//   - It requires no running Python process (fast, hermetic, always-on in
//     CI), which means it cannot detect drift introduced *between* schema
//     regenerations - if a Pydantic model changes and nobody regenerates the
//     fixture, this test still passes against the stale fixture. This is the
//     unavoidable trade-off of not adding a live Python dependency to the Go
//     test path (see the task's constraint #2); T106 (live SSE round trip)
//     is the complementary test that would catch a live, un-regenerated
//     drift.
//   - Go does not have one single named struct that types the *entire*
//     DeepSynthesis shape end to end. By design (see
//     deep_identification_pipeline_runner.go's buildDeepProposalDocumentJSON
//     and deep_identification_frame_translator.go's handleSynthesis), Go
//     treats most of the terminal synthesis report as an opaque
//     json.RawMessage pass-through to the persisted event/report and the
//     frontend, and only types `narrative`/`proposed_fields` (plus their
//     nested `evidence_refs`) and `partial_success` out into narrow, private
//     structs to build the owner-facing proposal. Where Go *does* have a
//     named, reflectable struct (`deepSynthesisProposedField` and its nested
//     evidence_refs entry), this file compares it mechanically. Where it does
//     not (`disagreements`, `unresolved_questions`, `coverage`,
//     `attributions`, `image_hypothesis`, `partial_success`'s own anonymous
//     struct, and the `narrative`/`proposed_fields` wrapper key names
//     themselves), this file falls back to pinning the literal, known
//     top-level property set read from the schema fixture - this is the one
//     genuinely hand-maintained list in this file, and it exists only
//     because there is no Go type to reflect over for those fields. It still
//     catches a rename/removal (the test goes red), but a human must
//     consciously update the pinned list (and, more importantly, check
//     whether Go's opaque pass-through and the frontend's TypeScript mirror
//     still agree) rather than the test silently adapting to the new shape.
//   - `provider: "image"` is valid only as an EvidenceRef.provider value
//     (FR-025), never as a DeepProviderCatalogEntry/ProviderName enum member;
//     this file does not walk enum member sets (out of scope for
//     field-name/nullability drift), so it neither asserts nor contradicts
//     that rule.

import (
	"encoding/json"
	"os"
	"reflect"
	"sort"
	"testing"
)

// TestDeepProposalCollectionAllowlistIsClosedAndSeparate (Feature 352 Phase
// 3, D-1 in decisions.md): "catalogReferences" is the one and only key in
// deepProposalCollectionFieldAllowlist, and it MUST NOT drift into either
// scalar allowlist (deepProposalCoinFieldAllowlist/
// deepProposalDraftFieldAllowlist). A collection-valued key sharing a name
// with a scalar allowlist entry would make applyToCoin's two-map dispatch
// (see deep_identification_proposal.go's isDeepProposalScalarCoinField /
// isDeepProposalCollectionField) ambiguous, and - worse - would let a JSON
// array reach setCoinFieldFromProposalValue's scalar coercion instead of
// CoinReferenceService.AppendForCoin. This guards that separation
// mechanically rather than relying on code review alone to keep the maps
// disjoint as new fields are added in future phases.
func TestDeepProposalCollectionAllowlistIsClosedAndSeparate(t *testing.T) {
	if len(deepProposalCollectionFieldAllowlist) != 1 {
		t.Fatalf("expected exactly one collection-valued proposal field, got %d: %v", len(deepProposalCollectionFieldAllowlist), deepProposalCollectionFieldAllowlist)
	}
	if _, ok := deepProposalCollectionFieldAllowlist["catalogReferences"]; !ok {
		t.Fatalf("expected the sole collection-valued proposal field to be %q, got %v", "catalogReferences", deepProposalCollectionFieldAllowlist)
	}

	for name := range deepProposalCollectionFieldAllowlist {
		if _, inScalarCoin := deepProposalCoinFieldAllowlist[name]; inScalarCoin {
			t.Errorf("collection field %q must not also appear in deepProposalCoinFieldAllowlist (scalar allowlist) - a JSON array must never reach setCoinFieldFromProposalValue", name)
		}
		if _, inScalarDraft := deepProposalDraftFieldAllowlist[name]; inScalarDraft {
			t.Errorf("collection field %q must not also appear in deepProposalDraftFieldAllowlist (scalar allowlist) - a JSON array must never reach setCoinFieldFromProposalValue", name)
		}
	}

	// The reverse direction: no scalar allowlist entry may leak into the
	// collection allowlist either, keeping the three maps fully disjoint by
	// key.
	for name := range deepProposalCoinFieldAllowlist {
		if _, inCollection := deepProposalCollectionFieldAllowlist[name]; inCollection {
			t.Errorf("scalar coin field %q must not also appear in deepProposalCollectionFieldAllowlist", name)
		}
	}
	for name := range deepProposalDraftFieldAllowlist {
		if _, inCollection := deepProposalCollectionFieldAllowlist[name]; inCollection {
			t.Errorf("scalar draft field %q must not also appear in deepProposalCollectionFieldAllowlist", name)
		}
	}
}

// --- Schema fixture types (Python side, read verbatim from Pydantic's own
// model_json_schema() output - never hand-typed) ---

type contractSchemaProp struct {
	Type  string                `json:"type"`
	Ref   string                `json:"$ref"`
	AnyOf []contractSchemaAnyOf `json:"anyOf"`
}

type contractSchemaAnyOf struct {
	Type string `json:"type"`
	Ref  string `json:"$ref"`
}

// nullable reports whether this Pydantic field accepts JSON null, i.e. it is
// `Optional[X]`/`X | None` - the Pydantic-side equivalent of a Go pointer
// field in this contract's conventions.
func (p contractSchemaProp) nullable() bool {
	if p.Type == "null" {
		return true
	}
	for _, a := range p.AnyOf {
		if a.Type == "null" {
			return true
		}
	}
	return false
}

type contractSchemaObject struct {
	Properties map[string]contractSchemaProp `json:"properties"`
}

type contractSchemaModel struct {
	Defs       map[string]contractSchemaObject `json:"$defs"`
	Properties map[string]contractSchemaProp   `json:"properties"`
}

type contractSchemaEnvelope struct {
	DeepIdentifyRequest contractSchemaModel `json:"DeepIdentifyRequest"`
	DeepSynthesis       contractSchemaModel `json:"DeepSynthesis"`
}

func loadContractSchema(t *testing.T) contractSchemaEnvelope {
	t.Helper()
	raw, err := os.ReadFile("testdata/deep_identify_contract_schema.json")
	if err != nil {
		t.Fatalf("reading contract schema fixture: %v", err)
	}
	var env contractSchemaEnvelope
	if err := json.Unmarshal(raw, &env); err != nil {
		t.Fatalf("parsing contract schema fixture: %v", err)
	}
	return env
}

// pyFields extracts a name->nullable map from a JSON Schema object's
// properties, mirroring the Go-side extraction below field for field.
func pyFields(props map[string]contractSchemaProp) map[string]bool {
	out := make(map[string]bool, len(props))
	for name, prop := range props {
		out[name] = prop.nullable()
	}
	return out
}

// --- Go side (real shipped mirror structs, read via reflection over the
// actual `json` struct tags - never a duplicate hand-maintained list) ---

// jsonTagName returns the wire field name for a struct field, or "" if the
// field is untagged/excluded ("-").
func jsonTagName(tag string) string {
	if tag == "" || tag == "-" {
		return ""
	}
	for i, c := range tag {
		if c == ',' {
			return tag[:i]
		}
	}
	return tag
}

// goFields extracts a name->nullable map from a Go struct type's exported,
// JSON-tagged fields. A field counts as nullable iff its Go type is a
// pointer - the convention this codebase already uses for
// present-but-optional-object fields (e.g. `NGC *DeepQuickEvidenceNGCProxy`).
func goFields(t reflect.Type) map[string]bool {
	if t.Kind() != reflect.Struct {
		panic("goFields: not a struct: " + t.String())
	}
	out := make(map[string]bool, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		name := jsonTagName(f.Tag.Get("json"))
		if name == "" {
			continue
		}
		out[name] = f.Type.Kind() == reflect.Ptr
	}
	return out
}

// assertFieldsMatch is the shared comparator: every Go-tagged field must
// have a same-named Python property with the same nullability, and every
// Python property must have a same-named Go field. Asymmetric drift in
// either direction fails.
func assertFieldsMatch(t *testing.T, label string, py, goSide map[string]bool) {
	t.Helper()
	names := make(map[string]bool, len(py)+len(goSide))
	for n := range py {
		names[n] = true
	}
	for n := range goSide {
		names[n] = true
	}
	sorted := make([]string, 0, len(names))
	for n := range names {
		sorted = append(sorted, n)
	}
	sort.Strings(sorted)

	for _, name := range sorted {
		pyNullable, inPy := py[name]
		goNullable, inGo := goSide[name]
		switch {
		case inPy && !inGo:
			t.Errorf("%s: Python field %q has no matching Go field (json tag) - Go mirror struct has drifted behind the Pydantic model", label, name)
		case inGo && !inPy:
			t.Errorf("%s: Go field %q (json tag) has no matching Python property - Go mirror struct has drifted ahead of the Pydantic model (or is sending a field Pydantic's extra=\"forbid\" would reject)", label, name)
		case pyNullable != goNullable:
			t.Errorf("%s: field %q nullability mismatch: Python nullable=%v, Go pointer=%v", label, name, pyNullable, goNullable)
		}
	}
}

// TestDeepIdentifyRequestContractFieldsMatchPython mechanically compares
// every Go mirror struct feeding the Go->Python DeepIdentifyRequest wire
// payload against the Pydantic model's own JSON Schema, field name by field
// name, nullability by nullability. This directly guards T093 drift point #2
// (`llm_config` -> `llm`) and, as a side effect of the "extra Go field" arm
// of assertFieldsMatch, T093 drift point #3 (a `quick_evidence` field
// Pydantic's `extra="forbid"` QuickEvidence model would reject).
func TestDeepIdentifyRequestContractFieldsMatchPython(t *testing.T) {
	env := loadContractSchema(t)
	req := env.DeepIdentifyRequest

	cases := []struct {
		label string
		py    map[string]contractSchemaProp
		goT   reflect.Type
	}{
		{"DeepIdentifyRequest", req.Properties, reflect.TypeOf(DeepIdentifyProxyRequest{})},
		{"DeepIdentifyImage", req.Defs["DeepIdentifyImage"].Properties, reflect.TypeOf(DeepIdentifyImageProxy{})},
		{"DeepProviderCatalogEntry", req.Defs["DeepProviderCatalogEntry"].Properties, reflect.TypeOf(DeepProviderCatalogEntryProxy{})},
		{"DeepIdentifyBounds", req.Defs["DeepIdentifyBounds"].Properties, reflect.TypeOf(DeepIdentifyBoundsProxy{})},
		{"QuickEvidence", req.Defs["QuickEvidence"].Properties, reflect.TypeOf(DeepQuickEvidenceProxy{})},
		{"QuickEvidenceNGC", req.Defs["QuickEvidenceNGC"].Properties, reflect.TypeOf(DeepQuickEvidenceNGCProxy{})},
		{"LLMConfig", req.Defs["LLMConfig"].Properties, reflect.TypeOf(LLMConfig{})},
	}

	for _, c := range cases {
		if len(c.py) == 0 {
			t.Fatalf("%s: fixture has no properties - fixture missing/stale, regenerate testdata/deep_identify_contract_schema.json", c.label)
		}
		assertFieldsMatch(t, c.label, pyFields(c.py), goFields(c.goT))
	}
}

// TestDeepSynthesisProposedFieldContractMatchesPython mechanically compares
// the one part of the DeepSynthesis contract Go actually types into named,
// reflectable structs - proposed_fields entries and their nested
// evidence_refs - against the Pydantic ProposedFieldValue/EvidenceRef
// models. See the file-level doc comment for why the rest of DeepSynthesis
// cannot be compared this way.
func TestDeepSynthesisProposedFieldContractMatchesPython(t *testing.T) {
	env := loadContractSchema(t)
	syn := env.DeepSynthesis

	proposedFieldValueProps := syn.Defs["ProposedFieldValue"].Properties
	evidenceRefProps := syn.Defs["EvidenceRef"].Properties
	if len(proposedFieldValueProps) == 0 || len(evidenceRefProps) == 0 {
		t.Fatal("fixture missing ProposedFieldValue/EvidenceRef defs - regenerate testdata/deep_identify_contract_schema.json")
	}

	pfType := reflect.TypeOf(deepSynthesisProposedField{})
	assertFieldsMatch(t, "ProposedFieldValue", pyFields(proposedFieldValueProps), goFields(pfType))

	// evidence_refs is a slice of an inline anonymous struct on the Go side
	// (deepSynthesisProposedField.EvidenceRefs); resolve its element type
	// directly rather than duplicating its field list.
	evidenceRefsField, ok := pfType.FieldByName("EvidenceRefs")
	if !ok {
		t.Fatal("deepSynthesisProposedField no longer has an EvidenceRefs field - update this test to match its new shape")
	}
	elemType := evidenceRefsField.Type.Elem()
	assertFieldsMatch(t, "EvidenceRef", pyFields(evidenceRefProps), goFields(elemType))
}

// TestDeepSynthesisKnownTopLevelFieldsMatchPython is the documented,
// non-mechanical fallback for the DeepSynthesis fields Go has no reflectable
// struct for (see file-level doc comment). It pins the literal top-level
// property set read from the schema fixture; a rename/removal/addition
// makes this go red, but - unlike the reflection-based tests above -
// updating this list is a manual, conscious step, not something a future Go
// struct change does for free. This directly guards T093 drift point #5
// (`attributions` added to the DeepSynthesis example).
func TestDeepSynthesisKnownTopLevelFieldsMatchPython(t *testing.T) {
	env := loadContractSchema(t)
	props := env.DeepSynthesis.Properties
	if len(props) == 0 {
		t.Fatal("fixture has no DeepSynthesis top-level properties - regenerate testdata/deep_identify_contract_schema.json")
	}

	// Hand-maintained by necessity: Go types none of these fields into a
	// named struct today (it treats the terminal synthesis report as an
	// opaque json.RawMessage pass-through - see the file-level doc comment).
	// If you add a Go struct that types one of these fields, move it into
	// TestDeepSynthesisProposedFieldContractMatchesPython (or a new sibling
	// mechanical test) and remove it from this list.
	expected := []string{
		"attributions",
		"coverage",
		"disagreements",
		"image_hypothesis",
		"narrative",
		"partial_success",
		"proposed_fields",
		"unresolved_questions",
	}
	sort.Strings(expected)

	got := make([]string, 0, len(props))
	for name := range props {
		got = append(got, name)
	}
	sort.Strings(got)

	if len(got) != len(expected) {
		t.Fatalf("DeepSynthesis top-level fields changed: got %v, want %v", got, expected)
	}
	for i := range expected {
		if got[i] != expected[i] {
			t.Fatalf("DeepSynthesis top-level fields changed: got %v, want %v", got, expected)
		}
	}
}
