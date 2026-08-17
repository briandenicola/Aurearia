# Contract: Vision Hypothesis and Image Claim Source (internal)

**Amends**: `specs/344-deep-agentic-coin-identification/contracts/agent-internal-contract.md`
**Direction**: internal to the Python pipeline, plus additive fields on the
existing Go ← Python `synthesis` frame. **No public (browser-facing) contract
change** — `contracts/sse-events.md` is untouched.

This document is the delta. Everything not stated here is unchanged from the
Feature 344 internal contract.

---

## 1. `CoinHypothesis` (new — the vision node's typed output)

Replaces the free-prose `state["image_analysis"]` string, which was written by
`graph.py` and read by nothing. Produced by the **same single vision LLM call**
that already runs on every job (structured output binding; no second call).

```jsonc
{
  "ruler":            { "value": "Maximinus I (Thrax)", "confidence": 0.86 },
  "denomination":     { "value": "Denarius",            "confidence": 0.9  },
  "material":         { "value": "Silver",              "confidence": 0.85 },
  "mint":             { "value": "Rome",                "confidence": 0.6  },
  "dateRange":        { "value": "AD 235-238",          "confidence": 0.7  },
  "era":              { "value": "roman-imperial",      "confidence": 0.9  },
  "obverseInscription": { "value": "IMP MAXIMINVS PIVS AVG", "confidence": 0.82 },
  "reverseInscription": { "value": "PAX AVGVSTI",            "confidence": 0.55 },
  "obverseDescription": { "value": "Laureate, draped and cuirassed bust right", "confidence": 0.8 },
  "reverseDescription": { "value": "Pax standing left, holding branch and sceptre", "confidence": 0.6 },
  "diameterMm":       { "value": "20",   "confidence": 0.3 },
  "weightGrams":      { "value": "3.2",  "confidence": 0.2 },
  "observations": "Silvered surfaces, high relief portrait, legend fully legible on obverse.",
  "legible": true
}
```

Rules:

- Every entry is `{ "value": <bounded string>, "confidence": <float 0..1> }`.
  A field the images do not support is **omitted** — never emitted with a
  guessed value at low confidence (spec FR-003).
- Keys use the **coin-field vocabulary** already used by `proposed_fields` and
  the Go proposal allowlist (`deepProposalCoinFieldAllowlist`), so an image-only
  field lands in the draft through the existing allowlist with **no new write
  surface**. Any key outside that vocabulary is dropped during normalization.
- The hypothesis carries **no citation** and is never converted into a
  `ProviderClaim` (spec FR-004).
- `observations` is a short bounded prose summary for the narrative writer only.
  It is never a proposed field value.
- Validation failure, LLM failure, timeout, or empty output ⇒ a typed **empty**
  hypothesis (`legible: false`, no fields). The pipeline continues; the job never
  fails for this reason (spec FR-006).
- The hypothesis MUST NOT be written to application logs or `progress` event
  messages (spec FR-030, 344 FR-036).

### Consumers (all mandatory — spec FR-007, SC-004)

| Consumer | Use |
|---|---|
| Router | provider selection signals (e.g. `era`/`ruler` for OCRE) |
| Provider query construction | deterministic query terms (§2) |
| Evaluator | first-class claim source for contradiction detection (§3) |
| Synthesis | narrative input + image-derived proposed fields (§4) |

A test MUST assert each consumer actually receives it. A write-only state field
is the exact defect this feature exists to remove.

---

## 2. Provider query-term precedence (amended)

Replaces the per-provider `_build_query` implementations, whose final tier was
the literal constant `"unidentified ancient coin"`.

Precedence, first non-empty wins:

1. `quick_evidence.numista_query`
2. `quick_evidence.label_text`
3. **Hypothesis-derived terms** — deterministic composition from the highest
   confidence identity fields available, in this order of preference:
   `ruler + denomination`, then `ruler`, then `denomination + material`, then
   `obverseInscription`. Reverse type/legend inclusion is governed by spec OQ-4.
4. `notes[:200]`

Rules:

- Query text remains **application-authored**. No LLM may choose, rewrite, or
  extend it (preserves the property documented at the top of
  `providers/numista.py`).
- The placeholder constant is **deleted**. When no tier yields usable terms, the
  provider node returns:

```jsonc
{ "provider": "numista", "status": "no_match", "automatable": true,
  "error_kind": "insufficient_query_evidence", "call_count": 0 }
```

and makes **zero** upstream calls (spec FR-011). This extends the 344 FR-025
status vocabulary so "we had nothing to search with" is distinguishable from
"we searched and found nothing". `ProviderErrorKind` gains
`insufficient_query_evidence`; the Go mirror must tolerate it (it is carried as a
bounded string in the public `provider_result` payload, which already forwards
`errorKind`).
- The same module builds terms for **every** automatable provider, so no
  provider can silently retain placeholder behavior.

---

## 3. Image as a claim source (evaluator + evidence refs)

`EvidenceRef.provider` is already an unconstrained bounded string (not the
`ProviderName` literal union), and the 344 contract §5 already documents
`evidence_refs: [{"provider": "image"}]`. Nothing has ever emitted one. This
feature emits them.

- The evaluator flattens hypothesis fields into `(field, source="image", value,
  confidence)` tuples alongside provider claims, then applies the **existing
  deterministic** `detect_disagreements` normalization (`value.strip().lower()`).
- A field where the image value and a provider value differ produces a
  `DisagreementEntry` with `resolution: "unresolved"` whose `claim_refs` include
  both `{"provider": "<name>", "claim_index": N}` and `{"provider": "image"}`
  (an image ref has no `claim_index`).
- Disagreement detection stays **LLM-free**. An LLM may only phrase the
  human-facing question.
- **`image` is not a provider.** It MUST NOT appear in
  `DeepIdentifyRequest.provider_catalog`, `provider_override`,
  `DeepSynthesis.coverage`, `DeepSynthesis.attributions`, or the `ProviderName`
  literal union. `ProviderCoverageEntry.provider` and
  `ProviderAttribution.provider` remain `ProviderName`.
- Citation-host allowlist validation is unchanged and applies only to provider
  claims; an image ref has no citation to validate.

---

## 4. `DeepSynthesis` (amended, additive only)

```jsonc
{
  "narrative": "The images show …; Numista corroborates …; no source confirms the mint.",
  "proposed_fields": {
    "ruler":        { "value": "Maximinus I (Thrax)", "confidence": 0.86,
                      "evidence_refs": [{ "provider": "image" }] },
    "denomination": { "value": "Denarius", "confidence": 0.9,
                      "evidence_refs": [{ "provider": "numista", "claim_index": 0 },
                                        { "provider": "image" }] }
  },
  "disagreements": [ { "field": "mint",
                       "claim_refs": [{ "provider": "ocre", "claim_index": 1 },
                                      { "provider": "image" }],
                       "resolution": "unresolved" } ],
  "unresolved_questions": ["…"],
  "coverage": [ { "provider": "ngc", "status": "not_automated" } ],
  "attributions": [ { "provider": "numista", "text": "Source: Numista", "identifier": "https://…" } ],
  "image_hypothesis": { /* §1 CoinHypothesis — NEW, optional */ },
  "partial_success": false
}
```

Changes:

- **`image_hypothesis`** — new, **optional**. Present when the vision call
  produced anything. Absent in reports persisted before this feature. Go's
  report reader unmarshals only `narrative` and `proposed_fields`, so the key is
  ignored by existing code (additive-safe); the full report JSON is persisted
  verbatim, so the hypothesis is recoverable (spec FR-008).
- **`proposed_fields`** may now contain entries whose only ref is
  `{"provider": "image"}`. Such an entry has **no** `evidence` array in the
  resulting proposal document, because
  `buildDeepProposalDocumentJSON` already skips `provider == "image"` refs.
  The Go builder MUST still emit the field (with an empty evidence list) rather
  than dropping it.
- **Corroboration rule** (spec FR-022, constant pending OQ-1): when a provider
  claim's normalized value equals the hypothesis value for the same field, the
  proposed confidence is a deterministic bounded function of both inputs, both
  refs are attached, and the provider citation is carried. Never LLM-adjusted,
  never > 1.0.
- **Contradicted fields** are excluded from `proposed_fields` (existing
  behavior: disagreement fields are skipped) and appear in `disagreements` and
  `unresolved_questions`.
- **Fallback boundary**: `FALLBACK_NARRATIVE_NO_EVIDENCE` is emitted only when
  the hypothesis is empty **and** no provider contributed (spec FR-020).

---

## 5. Router decision (amended — deterministic)

The `router_selected` internal frame and the `RouterDecision`
(`selected`, `skipped[{provider, reason}]`, `rationale`) shape are **unchanged**.
What changes is how they are computed:

- `route()` becomes a **pure function** of `(catalog, provider_override, bounds,
  quick_evidence, hypothesis)`. `ROUTER_PROMPT` and the LLM invocation are
  deleted; `route()` no longer takes a `model`.
- Identical inputs ⇒ byte-identical `selected`, `skipped`, and `rationale`
  (spec FR-014, SC-006).
- `provider_override` continues to win outright and can never introduce a
  provider absent from the Go-supplied catalog.
- Selection biases toward inclusion; a provider is skipped only for a stated
  evidence-based reason or a bound. OCRE's Roman-Imperial signal rule is
  spec OQ-7 (inclusion-by-default assumed until decided).
- Non-automatable catalog entries (`ngc`, `rpc`, and `ocre` when its flag is
  off) are still excluded from `selected`/`skipped` and still run trivially.

---

## 6. Corrections to the Feature 344 internal contract (documentation only)

Applied to
`specs/344-deep-agentic-coin-identification/contracts/agent-internal-contract.md`
in this feature. Each corrects the document to match **shipped** code; none
changes behavior.

| § | Documented (wrong) | Shipped (correct) |
|---|---|---|
| §1 | `services.InternalTokenService.Mint(userID)`; `middleware.InternalTokenRequired` | the deep pipeline mints per job: `MintForJob(userID, jobID)`; the tools group uses `InternalJobTokenRequired` |
| §2 | request key `llm_config` | `DeepIdentifyRequest.llm` |
| §2 | `quick_evidence.numista_evidence` passthrough | no such field; `QuickEvidence` is `StrictRequestModel(extra="forbid")` and would **reject** the request |
| §3 | `evaluation` payload `{disagreements:[…], resolved:[…]}`; no `synthesis_started` row | emitted payload is `{disagreement_count, resolved_count}`; `synthesis_started` **is** emitted and needs a row |
| §5 | `DeepSynthesis` example omits `attributions` | `attributions` is emitted (Feature 345 / ADR 0010) |
| §7 | tool table omits `ocre_search` | `POST /api/internal/tools/ocre_search` exists (Feature 345 / ADR 0010) |

---

## 7. Compatibility summary

| Change | Kind | Old reader tolerance |
|---|---|---|
| `DeepSynthesis.image_hypothesis` | additive optional key | ignored (Go unmarshals a narrow struct) |
| `evidence_refs: [{"provider":"image"}]` | already-documented shape, newly emitted | already skipped by `buildDeepProposalDocumentJSON` |
| `error_kind: "insufficient_query_evidence"` | additive enum member | carried as a bounded string in the public payload |
| `route()` signature loses `model` | internal Python only | not part of any cross-service contract |
| `state["image_analysis"]` removed | internal Python only | had zero readers |
| Public SSE events | **unchanged** | n/a |
| Database schema | **unchanged** | n/a |
