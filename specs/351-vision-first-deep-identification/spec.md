# Feature Specification: Vision-First Deep Identification

**Feature Branch**: `351-vision-first-deep-identification`
**Created**: 2026-08-16
**Status**: Draft
**Amends**: Feature 344 (`specs/344-deep-agentic-coin-identification/spec.md`) — see [Amendments to Feature 344](#amendments-to-feature-344)
**ADR**: [ADR 0012 — Vision-First Deep Identification](../../docs/adr/0012-vision-first-deep-identification.md) (Proposed)
**Input**: User description: "The feature at a high level is supposed to take
obverse/reverse/notes data if a user selects deep analysis, it's supposed to
still do a NGC quick look on the images, but then do a deep analysis of the
images then send off to the various providers for additional fact checking then
synthesize the results into one narrative that can be saved as a draft for
either a wishlist item or collection item."

## Context & Background

Feature 344 shipped the Deep Analysis pipeline: a persisted, resumable,
owner-scoped background job that runs a vision node, an LLM router, a bounded
provider fan-out (Numista, Nomisma, OCRE automated; NGC, RPC link-out only), a
contradiction evaluator, and a synthesis node that produces a narrative plus a
confirm-gated draft proposal. Feature 345 added OCRE as an automated provider.
The Go job/event/SSE/cancel/retry/apply layer is production-quality and is **not**
in scope here.

The pipeline's **information architecture is inverted**. Providers are treated as
the source of truth, and the coin images — the only evidence that is always
present — are treated as decoration. Three defects, all verified in code:

1. **Providers search a placeholder string.** `providers/numista.py::_build_query`
   has the precedence `quick_evidence.numista_query` → `quick_evidence.label_text`
   → `notes[:200]` → the literal constant `_DEFAULT_QUERY =
   "unidentified ancient coin"`. `nomisma.py` and `ocre.py` have the same shape.
   The vision result is never consulted when building a provider query. A coin
   with no quick-lookup fields and no owner notes causes all three automatable
   providers to issue a real upstream call for a meaningless string.
2. **The vision result is computed, paid for, and discarded.**
   `graph.py::prepare_evidence_node` (lines 62-86) performs a full vision LLM call
   and writes `state["image_analysis"]`. A repository-wide search for
   `image_analysis` returns exactly three hits: the declaration
   (`state.py:38`) and the two writes (`graph.py:71`, `graph.py:86`). **There are
   zero reads.** The `state.py` docstring claims it is "used as router/synthesis
   context"; it is not.
3. **Narrative and draft fields are hard-gated on providers.**
   `synthesis.py::synthesize` computes `contributing = [row for row in evidence if
   row.status == "contributed"]` and, when empty, replaces the narrative with
   `FALLBACK_NARRATIVE_NO_EVIDENCE`. `synthesize()` does not accept the image
   analysis as a parameter at all. `_build_proposed_fields()` derives
   `ProposedFieldValue` entries exclusively from provider claims, so a coin with
   no provider match produces **zero** draft fields.

Consequently two landed Feature 344 requirements are unimplemented in substance:

- **FR-022 (344)** requires routing on "an initial quick-lookup pass **plus image
  evidence**". `graph.py::router_node` passes only catalog, override,
  `max_providers`, and `notes` to `route()` — neither quick evidence nor image
  evidence reaches the router.
- **FR-027 (344)** requires evaluating contradictions across contributing sources
  "**including image evidence**". `evaluator.py::detect_disagreements` groups only
  `ProviderEvidence.claims`; image evidence can never participate, so a
  provider that contradicts what is plainly legible on the coin is invisible.

The Feature 344 internal contract §5 already documents the shape
`evidence_refs: [{"provider": "image"}]` for image-supported fields, and the Go
proposal builder already tolerates and skips such refs
(`deep_identification_pipeline_runner.go`, `buildDeepProposalDocumentJSON`).
Nothing has ever emitted one.

### The real-world failure (the reason this feature exists)

A slabbed NGC **Maximinus I AR Denarius** was run through Deep Analysis with
obverse and reverse photographs and no owner notes. All three automatable
providers returned `no_match`, NGC reported `not_automated` ("manual
verification"), and the persisted report read:

> "No provider evidence could be gathered for this coin. Please review the
> image-based analysis and consider retrying once providers are available."

The draft proposal contained zero fields. The advice to "review the image-based
analysis" is impossible to follow, because the image-based analysis was never
persisted or surfaced anywhere.

A single one-shot prompt to the same model with the same two photographs
returned a full, correct, richly detailed attribution: Maximinus Thrax, obverse
legend `IMP MAXIMINVS PIVS AVG`, Rome mint, AD 235-238, likely `PAX AVGVSTI`,
RIC IV 12. The capability was already paid for on every run and thrown away.

### The runtime failure (a second, independent defect)

Discovered after the design analysis above, and **independently sufficient** to
reproduce every symptom of the run below.

`src/api/services/deep_identification_pipeline_runner.go:112` gives the
quick-lookup pass a **15-second** budget:

```
quickCtx, cancelQuick := context.WithTimeout(ctx, 15*time.Second)
quickEvidence := r.extractQuickEvidence(quickCtx, job.UserID, images, job.Notes)
```

That call chain is `CoinLookupService.Lookup` → `extractDataFromImages` →
`proxy.AnalyzeCoin` — a full vision LLM round trip through the Python service on
two phone photographs. The identical call standalone is allowed **five minutes**
(`src/api/services/agent_proxy.go:36`, `requestClient: &http.Client{Timeout: 5 *
time.Minute}`), which is why Brian's standalone Quick Lookup on the same coin
succeeded and returned NGC cert `8232252-186`, grade, ruler, and denomination. A
20x mismatch.

On deadline exceed, `Lookup` returns an error and `extractQuickEvidence`
(`runner.go:426-432`) returns `nil` with only a `Warn` — silently. That alone
deterministically produces: NGC rendering the generic "Manual verification"
link-out with no cert, and all three automatable providers falling back through
the empty precedence chain to the literal placeholder query.

The two defects compound. The runtime defect starves the design of its only
usable evidence; the design defect then throws away the one source that remained.
Both are in scope for this feature, and the runtime fix is independently
shippable ahead of the redesign.

### The reframing

**Vision is the primary identification engine. Providers are fact-checkers.**
Deep Analysis identifies the coin from its images, then queries catalogue
authorities to confirm, refine, or contradict that identification, then narrates
the combined picture into one report and one savable draft. A provider that
finds nothing weakens confidence; it does not erase the identification.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - A coin with no catalogue match still gets identified (Priority: P1)

A collector runs Deep Analysis on a coin whose photographs are legible but which
no automated provider can match (worn legend spelling, a type absent from the
catalogues, or an upstream outage). The job completes with a real narrative
attribution derived from the images, a set of proposed draft fields marked as
image-derived at honest confidence, and an explicit statement that no external
source corroborated the identification.

**Why this priority**: This is the exact failure Brian hit. Without it the
feature delivers a fallback paragraph and an empty draft on the most common
real-world case — an ancient coin that is not a clean catalogue match.

**Independent Test**: Run Deep Analysis with obverse+reverse images, no notes,
and every automatable provider forced to `no_match`. Verify the narrative names
what the images show, the proposal has ≥1 field, every such field carries
`evidence_refs: [{"provider": "image"}]`, and the report states that no provider
corroborated it.

**Acceptance Scenarios**:

1. **Given** a legible obverse/reverse pair and zero provider contributions,
   **When** the job reaches a terminal state, **Then** the narrative describes
   the image-derived identification and explicitly states that no provider
   corroborated it — and is **not**
   `FALLBACK_NARRATIVE_NO_EVIDENCE`.
2. **Given** the same run, **When** the proposal is built, **Then** it contains
   at least the fields the vision hypothesis populated, each with a confidence
   value and `evidence_refs: [{"provider": "image"}]`.
3. **Given** the same run, **When** the owner opens the proposal, **Then** every
   image-derived field is visibly marked as image-derived (no citation) and is
   distinguishable from a provider-cited field.
4. **Given** an illegible image pair **and** zero provider contributions,
   **When** the job completes, **Then** the no-evidence fallback narrative *is*
   used and the proposal is empty — the fallback remains reachable, but only
   when both sources produced nothing.

---

### User Story 2 - The Maximinus Run (named regression scenario) (Priority: P1)

The canonical end-to-end scenario this feature is measured against. A slabbed
NGC Maximinus I AR Denarius, obverse and reverse photographs, **no owner notes**,
quick lookup yielding no usable `numista_query` / `label_text` / coin fields.

**Why this priority**: This is a reproduction of a verified production failure.
Every requirement in this spec exists to change this run's outcome. This scenario
is the feature's definition of success.

**Independent Test**: Replay the Maximinus fixture (two face images, empty notes,
empty quick evidence) with recorded provider responses. Compare the persisted
report and proposal against the "after" column below.

**Acceptance Scenarios**:

1. **Given** the Maximinus fixture, **When** the vision node runs, **Then** it
   emits a typed hypothesis populating at least `ruler`, `denomination`,
   `material`, and `obverse_legend`, each with its own confidence — not a prose
   paragraph.
2. **Given** the Maximinus fixture, **When** provider queries are built, **Then**
   no provider is called with the string `"unidentified ancient coin"`, and each
   automatable provider's query is derived from the hypothesis (e.g. ruler +
   denomination).
3. **Given** the Maximinus fixture where every automatable provider still returns
   `no_match`, **When** synthesis runs, **Then** the narrative names the ruler,
   denomination, and legend read from the images and states plainly that no
   catalogue authority corroborated it.
4. **Given** the same run, **When** the proposal is built, **Then** it contains
   ≥4 proposed fields (mapped to the existing coin-field allowlist), each with
   `evidence_refs: [{"provider": "image"}]`.
5. **Given** the same run, **When** the owner chooses to save it, **Then** the
   draft is savable as either a collection item or a wishlist item through the
   existing confirm-gated write paths.
6. **Given** the same run, **When** the persisted job record is inspected,
   **Then** the vision hypothesis is recoverable from the stored report (it is
   not discarded).

**Before / after contract for this scenario**:

| Observable | Today (FAIL) | Required (PASS) |
|---|---|---|
| Provider query text | `"unidentified ancient coin"` | hypothesis-derived, e.g. `Maximinus I denarius` |
| Vision output | free prose, written to `state["image_analysis"]`, never read | typed hypothesis, read by router, providers, evaluator, synthesis |
| Narrative | `FALLBACK_NARRATIVE_NO_EVIDENCE` | image-derived attribution + "no provider corroboration" |
| Proposed fields | 0 | ≥4, each with an `image` evidence ref |
| Draft savable | nothing to save | collection item or wishlist item |

---

### User Story 3 - Providers confirm, refine, or contradict the identification (Priority: P1)

When a provider *does* match, its citation-backed claim is merged with the image
hypothesis: agreement raises the field's confidence and attaches the citation;
a provider that supplies a field the images could not (a catalogue type number, a
canonical mint spelling) adds it; a provider that contradicts the images produces
a visible, unresolved disagreement rather than a silent overwrite in either
direction.

**Why this priority**: This is the "fact-checking" half of the reframing. Without
it the feature would simply replace provider tunnel-vision with image
tunnel-vision.

**Independent Test**: Run three fixtures — provider agrees with the hypothesis,
provider adds a field absent from the hypothesis, provider contradicts the
hypothesis — and verify confidence upgrade + citation, field addition, and an
unresolved disagreement respectively.

**Acceptance Scenarios**:

1. **Given** a provider claim whose normalized value matches the hypothesis for
   the same field, **When** the proposal is built, **Then** that field's
   confidence is at or above the higher of the two inputs and its
   `evidence_refs` include **both** the provider claim ref and the `image` ref.
2. **Given** a provider claim for a field the hypothesis left empty, **When** the
   proposal is built, **Then** the field is proposed from the provider claim with
   its citation, exactly as today.
3. **Given** a provider claim whose normalized value differs from the
   hypothesis for the same field, **When** the evaluator runs, **Then** a
   disagreement entry is produced whose `claim_refs` include both the provider
   claim and the `image` source, with `resolution: "unresolved"`.
4. **Given** that contradiction, **When** the proposal is built, **Then** the
   contradicted field is **not** silently proposed from either side, and the
   narrative and unresolved-questions list surface the conflict.
5. **Given** any run, **When** attributions and coverage are built, **Then**
   `image` never appears as a provider in the provider catalog, provider
   coverage, or attribution list — it is a claim source, not a provider.

---

### User Story 4 - Bounded, deterministic provider selection (Priority: P2)

Provider selection uses the quick-lookup evidence and the image hypothesis, is
reproducible across identical runs, and is explainable in one sentence the owner
can read in the `router_selected` event.

**Why this priority**: Feature 344 FR-022 promised evidence-driven routing and
never delivered it. With only three automatable providers this is a small,
high-certainty win — and reproducibility is worth more here than model judgment.

**Independent Test**: Run the same job inputs twice and diff `selected`,
`skipped`, and `rationale`. Run a clearly non-Roman-Imperial hypothesis and
verify OCRE is skipped with a stated reason. Run with an explicit provider
override and verify the override still wins outright.

**Acceptance Scenarios**:

1. **Given** identical inputs, **When** the router runs twice, **Then**
   `selected`, `skipped` (with reasons), and `rationale` are byte-identical.
2. **Given** a hypothesis with a clear Roman Imperial signal, **When** the router
   runs, **Then** OCRE is selected (subject to its enable flag and bounds).
3. **Given** a hypothesis with no Roman Imperial signal and no such signal in
   quick evidence, **When** the router runs, **Then** OCRE is skipped with an
   explicit reason and the remaining automatable providers still run.
4. **Given** an explicit `provider_override`, **When** the router runs, **Then**
   the override wins outright exactly as it does today, and no provider outside
   the Go-supplied catalog can be introduced.
5. **Given** a hypothesis and quick evidence that are both empty, **When** the
   router runs, **Then** it selects all automatable providers (bias toward
   inclusion) rather than skipping everything.

---

### User Story 5 - Save the result as a collection item or a wishlist item (Priority: P2)

Having reviewed the synthesized report, the owner saves the accepted fields as
either a collection item or a wishlist item, through the existing confirm-gated
write paths, with no automatic writes.

**Why this priority**: Stated explicitly in the feature intent. Today the apply
targets are `draft` (intake source) and `coin` (saved-coin source); there is no
way to land a Deep Analysis result as a wishlist entry, even though
`models.Coin.IsWishlist` exists.

**Independent Test**: Complete an intake-sourced Deep Analysis job and apply it
as a wishlist item; verify a wishlist coin (or a draft that promotes to one)
exists with the accepted fields, that no write occurred before confirmation, and
that the existing collection-item path is unchanged.

**Acceptance Scenarios**:

1. **Given** a terminal intake-sourced job with accepted fields, **When** the
   owner applies it as a collection item, **Then** behavior is exactly as today.
2. **Given** the same job, **When** the owner applies it as a wishlist item,
   **Then** the resulting record is marked as a wishlist entry and carries the
   accepted fields.
3. **Given** either target, **When** nothing has been confirmed, **Then** no coin
   or draft row is created or modified.
4. **Given** either target, **When** the write happens, **Then** it goes through
   the existing Go-owned write services only — the pipeline performs no direct
   coin write.

---

### User Story 6 - Quick-lookup failure is observable (Priority: P3)

When the initial NGC/quick-lookup pass fails, that fact is visible in the job
record rather than silently swallowed, so an owner or operator can tell "no cert
data existed" apart from "the lookup broke".

**Why this priority**: A correctness/observability gap, not a blocker.
`deep_identification_pipeline_runner.go::extractQuickEvidence` currently logs a
`Warn` and returns `nil`, which is indistinguishable downstream from "the coin
had no quick evidence".

**Independent Test**: Force `CoinLookupService.Lookup` to error and verify the
job exposes a typed quick-evidence outcome distinct from the no-data case, with
no owner notes or legend text in the event payload or logs.

**Acceptance Scenarios**:

1. **Given** the quick lookup errors, **When** the job runs, **Then** the job
   surfaces a typed "quick lookup unavailable" outcome distinct from "no quick
   evidence found", and still completes.
2. **Given** the quick lookup returns no usable data, **When** the job runs,
   **Then** that is reported as a distinct, non-error outcome.
3. **Given** either case, **When** the outcome is recorded, **Then** it contains
   no owner notes, no legend text, and no image data.

---

### Edge Cases

- **Illegible or non-coin images**: the hypothesis is typed-but-empty with low
  confidence; providers are not called with a placeholder; the run ends with the
  no-evidence fallback and an empty proposal (the honest outcome).
- **Vision LLM failure/timeout**: typed empty hypothesis, pipeline continues to
  providers using quick evidence only; never a job failure.
- **Vision returns unparseable structured output**: treated as an empty
  hypothesis after the existing retry policy; recorded as a typed outcome, not an
  exception; the run continues.
- **Hypothesis with a legend but no ruler**: query building falls back to the
  next-most-specific deterministic combination; it never emits the placeholder
  constant and never asks an LLM to write the query.
- **Hypothesis contradicts quick evidence** (e.g. slab label says one ruler, the
  coin reads another): treated as a disagreement between sources, surfaced, not
  resolved by precedence.
- **Provider contradicts the hypothesis on a field the images read clearly**:
  unresolved disagreement; the field is withheld from the proposal.
- **Every provider `no_match` but the hypothesis is strong**: proposal is
  populated from the hypothesis at hypothesis confidence, never upgraded.
- **Non-Latin / non-Roman coinage** (Greek, Byzantine, Islamic): the hypothesis
  is emitted in the same schema; routing skips OCRE; no requirement assumes
  Roman Imperial coinage.
- **Legend text and privacy**: legend text may appear in provider queries and in
  the persisted report, but never in application logs or `progress` event
  messages (Feature 344 FR-036).
- **Old persisted jobs**: reports written before this feature have no hypothesis;
  every surface must render them unchanged.
- **Confidence is never fabricated**: a hypothesis field with no legible support
  is omitted, not emitted at low confidence with a guessed value.

## Requirements *(mandatory)*

Requirement IDs are stable within Feature 351. Where a requirement supersedes or
amends a Feature 344 requirement, the superseded text is quoted verbatim in
[Amendments to Feature 344](#amendments-to-feature-344).

### Functional Requirements

**A. The structured vision hypothesis (keystone)**

- **FR-001**: The vision node MUST emit a **typed, schema-validated coin
  hypothesis** instead of free prose. The schema MUST cover at minimum: `ruler`,
  `denomination`, `material`, `obverse_legend`, `reverse_legend`,
  `reverse_type`, `mint`, `date_range`, `era`, and optional inferable
  `diameter_mm` and `weight_grams`, plus a **per-field confidence** in `[0,1]`
  and a short overall observation summary.
- **FR-002**: The hypothesis MUST be produced by the **single vision LLM call
  that already runs on every job**. No second vision call may be introduced.
- **FR-003**: A field the images do not support MUST be **omitted or empty** —
  the model MUST NOT be permitted to fill a field by guessing. Confidence values
  MUST reflect legibility, not model verbosity.
- **FR-004**: The hypothesis MUST carry **no citation** and MUST NOT be emitted
  as a `ProviderClaim`, MUST NOT appear in the provider catalog, provider
  coverage, or provider attribution lists. It is a distinct claim source
  identified as `image`.
- **FR-005**: Hypothesis field names MUST be normalized to the **coin-field
  vocabulary** already used by `proposed_fields` and the Go proposal allowlist,
  so that image-derived fields land in the draft through the existing allowlist
  without widening the write surface.
- **FR-006**: If the vision call fails, times out, returns empty, or returns
  output that fails schema validation, the pipeline MUST continue with a typed
  **empty hypothesis** and MUST NOT fail the job.
- **FR-007**: The hypothesis MUST be **read by** the router (FR-010), provider
  query construction (FR-012), the evaluator (FR-016), and synthesis (FR-019).
  A state field that is written and never read is a defect this feature exists
  to eliminate; an automated test MUST assert each consumer receives it.
- **FR-008**: The hypothesis MUST be **recoverable from the persisted report** so
  the owner can see what the images alone supported, additively and without
  breaking reports persisted before this feature. It MUST additionally be
  **rendered in the report UI** as a collapsible "what the images alone said"
  section, default-collapsed (RD-6). This is a permanent diagnostic surface: the
  original failure was undiagnosable precisely because the owner could not see
  what vision produced.

**B. Provider queries as fact-checking probes**

- **FR-009**: Provider query text MUST continue to be built **deterministically
  by application code**, never authored or freely chosen by an LLM
  (preserves the property documented in `providers/numista.py`).
- **FR-010**: Query-term precedence MUST be: (1) `quick_evidence.numista_query`,
  (2) `quick_evidence.label_text`, (3) **deterministic terms derived from the
  vision hypothesis**, (4) owner notes. Quick evidence retains higher precedence
  than the hypothesis when present. Hypothesis-derived composition MUST follow a
  fixed order (`ruler + denomination` → `ruler` → `denomination + material` →
  `obverseInscription`). **Reverse type and reverse legend MUST be excluded from
  query terms entirely** (RD-4); they serve a ranking role under FR-039 instead.
  No second, narrower probe may be issued.
- **FR-039**: Reverse legend and reverse type MUST be used as a **ranking and
  disambiguation signal applied to candidate results a provider has already
  returned** — never as query input, and never as a trigger for an additional
  upstream call. This MUST add **zero** upstream calls and consume **zero**
  additional call budget (RD-4). Specifically:
  - For **Numista and Nomisma**, this is new behavior. `numista.py` today
    requests `limit=5` and then takes `candidates[0]` unconditionally,
    discarding four candidates unranked. Candidate selection MUST become a
    deterministic, application-owned ranking over the already-returned set,
    never an LLM choice (FR-009's property extends to ranking).
  - For **OCRE**, the ranking mechanism **already exists and is ADR 0010
    governed**: `providers/ocre.py::_legend_tokens` passes scoring-only tokens
    to `ocre_search`, and `src/api/services/ocre_scoring.go` applies a
    deterministic capped legend bonus over a stable sort. The **only** permitted
    change is widening the *source* of those tokens to include the hypothesis
    when quick evidence is absent. The scoring weights, bonus-per-match, bonus
    cap, clamping, and stable sort in `ocre_scoring.go` MUST NOT be altered —
    that math is ADR 0010's deterministic contract, and this feature does not
    amend it.
  - Ranking MUST be deterministic and reproducible: identical candidate sets and
    identical hypotheses produce an identical ordering and selection.
- **FR-011**: The placeholder constant `"unidentified ancient coin"` (and any
  equivalent) MUST be removed as a query source. When no precedence tier yields
  usable terms, the provider node MUST return a typed outcome indicating
  insufficient query evidence and MUST make **zero** upstream calls, rather than
  spending budget on a meaningless search.
- **FR-012**: Every automatable provider node MUST apply the same precedence
  rules, so no provider silently retains placeholder behavior.

**C. Routing**

- **FR-013**: Provider selection MUST consume the quick-lookup evidence **and**
  the vision hypothesis. *(Supersedes 344 FR-022.)*
- **FR-014**: Provider selection MUST be **deterministic**: identical inputs
  produce an identical `selected` set, `skipped` list with reasons, and
  `rationale` string. *(Supersedes the LLM-router design of 344 FR-022 and ADR
  0011's "the router records its selected provider set and rationale" as an LLM
  step; see ADR 0012.)*
- **FR-015**: Selection MUST bias toward **inclusion by default** (RD-7): a
  provider is selected whenever it is automatable and within bounds. OCRE MUST be
  skipped **only on a positive non-Roman-Imperial era signal** from the
  hypothesis or quick evidence — never on the mere absence of a Roman signal.
  Every skip MUST carry a stated reason in `skipped[]`; a skip without a reason
  is a defect. Bounds-driven skips (`max_providers`, budget, disabled flag)
  likewise carry their reason. Explicit `provider_override` MUST continue to win
  outright, and MUST NOT be able to introduce a provider absent from the
  Go-supplied catalog.

**D. Evaluation**

- **FR-016**: Contradiction detection MUST treat the vision hypothesis as a
  **first-class claim source** alongside provider claims, so provider-vs-image
  contradictions are detected. *(Supersedes 344 FR-027 in substance — the
  requirement text already said "including image evidence"; this feature makes
  it real.)*
- **FR-017**: An image-vs-provider disagreement MUST be surfaced as an
  unresolved disagreement referencing both sources. It MUST NOT be resolved by
  precedence in either direction, and neither value may be silently proposed.
- **FR-018**: Disagreement detection MUST remain **deterministic and LLM-free**;
  an LLM may only phrase the human-facing question, never add, remove, or
  resolve a disagreement.

**E. Synthesis, narrative, and draft fields**

- **FR-019**: Synthesis MUST narrate from the vision hypothesis **and** provider
  evidence, describing what the images support, what each provider confirmed,
  refined, or contradicted, and what remains open. *(Amends 344 FR-028.)*
- **FR-020**: The no-evidence fallback narrative MUST be reachable **only when
  both** the hypothesis and provider evidence are empty. Absence of provider
  contributions alone MUST NOT trigger it.
- **FR-021**: `proposed_fields` MUST be built from the vision hypothesis **and**
  provider claims. An image-only field MUST be proposed at the hypothesis's
  confidence for that field with `evidence_refs: [{"provider": "image"}]`.
  *(Amends 344 FR-028; realizes the 344 internal contract §5 shape that nothing
  currently emits.)*
- **FR-022**: When a provider claim corroborates a hypothesis field (equal after
  the existing normalization), the proposed field's confidence MUST be raised by
  the rule `min(1.0, max(image_confidence, provider_confidence) + 0.10)` (RD-2),
  its evidence refs MUST include both sources, and the provider's citation MUST
  be attached. The upgrade MUST be applied **once per field regardless of how
  many providers corroborate** — it does not stack. Confidence MUST never exceed
  1.0 and MUST never be raised, lowered, or adjusted by an LLM.
- **FR-023**: A field present only in provider claims MUST continue to be
  proposed exactly as today, with its citation.
- **FR-024**: Every proposed field MUST be traceable to at least one source ref
  (`image` or a validated provider claim). Provider-derived refs MUST continue to
  pass the existing per-provider citation host allowlist. An `image` ref carries
  no citation and MUST NOT be subjected to host validation.
- **FR-025**: Coverage and attribution lists MUST remain provider-only; `image`
  MUST NOT be added to the provider name vocabulary used by the catalog,
  coverage, or attributions. *(Clarifies 344 FR-029.)*
- **FR-026**: Image-derived proposed fields MUST be **visually distinguishable**
  from provider-cited fields wherever the report or proposal is rendered.
  Acceptance defaults, however, MUST be **confidence-driven, not source-driven**
  (RD-3): a proposed field defaults to **accepted at confidence ≥ 0.70** and
  **unaccepted below 0.70**, regardless of whether its source is image-only or
  provider-corroborated. The threshold MUST be a **single named constant**, not a
  literal repeated across call sites. Because the FR-022 upgrade is applied
  before this comparison, a corroborated field at 0.62 crosses the threshold and
  becomes accepted-by-default — this is intended and MUST be covered by an
  explicit test.

**F. Draft destination**

- **FR-027**: A terminal Deep Analysis result MUST be savable, on explicit owner
  confirmation, as either a **collection item** or a **wishlist item**, in
  addition to the existing saved-coin update path. The wishlist destination MUST
  be implemented as a `models.Coin` with `IsWishlist = true`, created through the
  existing `CoinService` create path (mechanism (a), RD-1). No schema migration
  and no change to `QuickCaptureDraft` is permitted to satisfy this requirement.
  *(Extends 344 FR-033; does not supersede it — the write MUST still route
  exclusively through the existing Go-owned write services.)*
- **FR-028**: No automatic write may occur. The confirm gate, owner-edit
  distinction, and field allowlists of 344 FR-031/FR-032/FR-033 remain in force
  unchanged. Specifically, `isWishlist` is a **destination intent carried on the
  apply request**, not a proposed field: it MUST NOT be added to
  `deepProposalCoinFieldAllowlist`, MUST NOT appear in `proposed_fields`, and
  MUST NOT be proposable, inferable, or influenceable by any model output. It
  MUST be derived only from an explicit, normalized target value supplied by the
  owner at confirm time, mirroring the existing
  `QuickCapturePromotionTarget` precedent (`services/quick_capture_service.go:530`).

**G. Observability, privacy, and cost**

- **FR-029**: A quick-lookup **failure** MUST be recorded as a typed, observable
  outcome on the job, distinct from "no quick evidence found", rather than a log
  line and a `nil`.
- **FR-038**: The quick-lookup pass inside Deep Analysis MUST be given a budget
  proportionate to the work it performs (a full vision LLM round trip), expressed
  as a **named, admin-tunable setting** rather than the current magic literal
  `15*time.Second`, and bounded above by the agent proxy's own non-streaming
  ceiling. The pipeline's remaining-budget computation
  (`deep_identification_pipeline_runner.go:116-123`, including
  `deepPipelineHardTimeoutSafetyMarginS`) MUST be **verified**, not assumed, to
  still leave a workable provider/synthesis budget after the enlarged quick-lookup
  allowance — the two are drawn from the same job deadline.
- **FR-030**: Telemetry for the vision step MUST record only structural facts
  (populated-field count, confidence buckets, validation success/failure,
  timing). Legend text, owner notes, hypothesis values, and image data MUST NOT
  enter application logs or `progress` event messages (344 FR-036).
  **AMENDED 2026-08-17 — see FR-040. The `progress` event clause is narrowed;
  the application-log prohibition is unchanged and remains absolute.**
- **FR-040** *(amendment to FR-030, authorized by Brian 2026-08-17)*: The
  **user-scoped SSE `progress` stream** MAY carry step detail derived from the
  owner's own submission — hypothesis field values, the application-authored
  query terms sent to each provider, candidate counts, and per-provider
  outcomes — so the owner can see what the system actually worked on at each
  step. Rationale: the stream is delivered only to the authenticated owner of
  the job, for their own coin, and the terminal report already discloses these
  same values to that same user; streaming them live is the same disclosure
  earlier, not a new one.
  **Binding limits on this amendment:**
  - The prohibition on **application logs** is untouched. Legend text, owner
    notes, hypothesis values, query strings, and image data MUST NOT be written
    to server logs. FR-030's log clause remains absolute.
  - Detail MUST flow only through the job's owner-scoped stream. It MUST NOT
    appear in any shared, admin, aggregate, or cross-user surface.
  - Image data (data URIs, raw bytes) remains prohibited everywhere, including
    the progress stream.
  - Progress payloads remain subject to the existing sanitization boundary
    (`sanitize_user_facing_payload`) and to bounded field lengths, so a
    malformed or hostile upstream value cannot become an unbounded or unescaped
    payload.
- **FR-031**: The redesign MUST NOT increase the number of LLM calls per job. The
  vision call is unchanged in count; removing the LLM router (FR-014) reduces the
  count by one on runs without an override.
- **FR-032**: The Python agent MUST remain **stateless** — no database handle, no
  credentials beyond the per-request LLM config, no cross-request memory
  (Constitution Principle II, 344 FR-035).

**H. Compatibility and contract hygiene**

- **FR-033**: All schema changes MUST be **additive and optional**. Jobs,
  reports, and proposals persisted before this feature MUST continue to load,
  render, and apply unchanged, and Go mirror structs MUST tolerate both shapes.
- **FR-034**: The feature MUST be reversible without a data migration; rollback
  MUST NOT invalidate already-persisted reports or proposals.
- **FR-035**: The five documented drift points between
  `specs/344-deep-agentic-coin-identification/contracts/agent-internal-contract.md`
  and the shipped code MUST be reconciled as **documentation-only** corrections
  in this feature (see [Contract drift reconciliation](#contract-drift-reconciliation)).
  No behavior change may be introduced under cover of that reconciliation.

**I. Non-regression**

- **FR-036**: The Go job/event/SSE/cancel/retry/retention/authorization layer
  MUST be behaviorally unchanged except for FR-027 (wishlist target), FR-029
  (quick-lookup observability), and FR-038 (quick-lookup budget).
- **FR-037**: The fast Identify Coin path, the NGC quick look, provider bounds,
  budgets, citation allowlists, SSE privacy rules, and the OCRE enable flag MUST
  be unchanged.

### Key Entities *(include if feature involves data)*

- **Coin hypothesis (image-derived)**: the typed output of the single vision
  call — per-field values with per-field confidence, no citation, source
  identity `image`. Consumed by routing, query building, evaluation, and
  synthesis; persisted additively inside the report.
- **Claim source**: the generalization of "provider" for evaluation and
  proposal-evidence purposes. Two kinds exist: a validated, citation-backed
  **provider claim**, and an uncited **image claim**. Only provider claims
  appear in coverage/attribution.
- **Provider query terms**: the deterministic, precedence-ordered term set fed to
  a provider tool (quick evidence → hypothesis → notes). Never LLM-authored,
  never a placeholder.
- **Router decision**: `selected`, `skipped` (each with a reason), and a
  deterministic one-sentence rationale, derived from catalog + bounds + override
  + quick evidence + hypothesis.
- **Proposed field**: a candidate coin-field value with a confidence and ≥1
  evidence ref, where a ref is either an `image` ref or a citation-backed
  provider claim ref.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: On the Maximinus fixture (legible images, no notes, no quick
  evidence, all providers `no_match`) the persisted proposal contains **≥4**
  proposed fields, versus **0** today.
- **SC-002**: On that same fixture the narrative is **not**
  `FALLBACK_NARRATIVE_NO_EVIDENCE` and names the ruler and denomination read
  from the images.
- **SC-003**: Across the full test corpus, **zero** provider calls are issued
  with the string `"unidentified ancient coin"` or any other placeholder — the
  constant no longer exists as a query source.
- **SC-004**: **100%** of jobs where the vision call succeeds have a hypothesis
  that is consumed by all four downstream consumers (router, query building,
  evaluator, synthesis), asserted by automated tests — i.e. zero write-only
  state fields remain in the deep-identification pipeline.
- **SC-005**: **100%** of image-only proposed fields carry
  `evidence_refs: [{"provider": "image"}]`, and **100%** of provider-derived
  refs pass the existing citation host allowlist.
- **SC-006**: Two identical runs produce byte-identical router `selected`,
  `skipped`, and `rationale` values (determinism), verified across ≥2 runs.
- **SC-007**: LLM calls per job do **not** increase; a run without a provider
  override makes **one fewer** LLM call than today (router removed).
- **SC-008**: `FALLBACK_NARRATIVE_NO_EVIDENCE` appears **only** in runs where
  both the hypothesis and provider evidence are empty — asserted by a test that
  fails if a provider-empty/hypothesis-present run emits it.
- **SC-009**: **100%** of provider-vs-image contradictions in the fixture set
  produce an unresolved disagreement referencing both sources; **zero** are
  resolved by precedence.
- **SC-010**: **Zero** coin or draft rows are created or modified without an
  explicit owner confirmation, on either the collection or wishlist target.
- **SC-011**: Reports and proposals persisted before this feature load, render,
  and apply with **zero** errors after the change (backward-compatibility
  fixture).
- **SC-012**: **Zero** occurrences of legend text, hypothesis values, or owner
  notes in application logs or `progress` event payloads.
- **SC-013**: The quick-lookup pass inside Deep Analysis completes within its
  budget for the same inputs that succeed in the standalone Quick Lookup path
  (zero deadline-exceeded quick-lookup outcomes on the fixture corpus), and the
  post-quick-lookup pipeline budget remains at or above the pre-change value of
  ~265 s at default settings.

## Assumptions

- The Feature 344 Go job/event/SSE/cancel/retry/retention layer and the Feature
  345 OCRE provider are production-quality and are reused, not redesigned.
- The vision model in use can return schema-conformant structured output for the
  hypothesis; where a provider/model cannot, the existing retry-then-degrade
  policy applies and yields an empty hypothesis (FR-006).
- `EvidenceRef.provider` is already an unconstrained bounded string (not the
  `ProviderName` literal union), and the Go proposal builder already skips
  `provider == "image"` refs — so emitting image refs requires **no** breaking
  contract change.
- Legend text is coin content, not owner content; it may appear in provider
  queries and the persisted report, but the existing prohibition on writing
  notes/queries to logs still applies to it.
- The single-owner, self-hosted deployment means run volume is low; determinism
  and explainability are worth more than adaptive routing.
- Only three providers are automatable today (Numista, Nomisma, OCRE), and
  `bounds.max_providers` in practice is ≥ 3.

## Non-Goals (Out of Scope)

- **Rewriting the Go job/event/SSE layer.** Job lifecycle, persistence, event
  sequencing/replay, cancellation, retry, retention, and authorization are
  audited as production-quality and are explicitly out of scope beyond FR-027 and
  FR-029.
- **New providers, new upstream integrations, or enabling RPC.** The provider
  catalog is unchanged.
- **Adding `image` to the provider vocabulary** (catalog, coverage, attribution,
  `ProviderName`). Image is a claim source, not a provider.
- **Changing the public SSE event vocabulary** in
  `contracts/sse-events.md`. No new browser-facing event type is introduced.
- **A second vision call, a multi-turn vision loop, or a ReAct agent** in this
  pipeline.
- **LLM-authored provider queries or LLM-adjusted confidence.** Both remain
  application-owned deterministic code.
- **Changing the confirm gate, coin-field write allowlists, or introducing a new
  write surface.** FR-027 reuses existing write services.
- **Re-running OCR or re-deriving NGC cert numbers.** The quick-look pass is
  unchanged (FR-037).
- **Database schema migration.** Changes are additive JSON inside existing
  columns.
- **Re-litigating ADR 0010 (OCRE/ODbL) or ADR 0009 (Nomisma).**

## Amendments to Feature 344

This feature changes the semantics of landed Feature 344 requirements. Per
Constitution §18.2 a landed spec is not retroactively rewritten; the amendment is
recorded here and authorized by **ADR 0012**. The Feature 344 spec receives a
header-level supersession banner only — its body text is left intact.

| 344 requirement | Verbatim current text (abridged where marked) | Disposition | 351 requirement(s) |
|---|---|---|---|
| **FR-022** | "System MUST use an initial quick-lookup pass plus image evidence to automatically propose a relevant, bounded set of providers for a given job, rather than always querying every configured provider." | **Superseded** — intent retained, mechanism changed from an LLM router to a deterministic, evidence-driven selector, and the "image evidence" clause is actually implemented. | FR-013, FR-014, FR-015 |
| **FR-027** | "System MUST evaluate contradictions and provenance across contributing sources (including image evidence) before synthesis, and MUST surface unresolved disagreements between sources rather than silently resolving them by precedence or discarding conflicting claims." | **Superseded** — text unchanged in intent; image evidence becomes a real claim source so the "(including image evidence)" clause becomes enforceable and tested. | FR-016, FR-017, FR-018 |
| **FR-028** | "System MUST produce, for every terminal completed/partial job, a single synthesized narrative report with source citations, a structured set of proposed coin field values, a confidence indicator per proposed field, and an explicit list of disagreements and/or unresolved questions." | **Amended** — the narrative and the proposed field set must derive from image evidence as well as providers; "source citations" is clarified to permit an uncited `image` source ref. | FR-019, FR-020, FR-021, FR-022, FR-023, FR-024, FR-026 |
| **FR-025** | "System MUST distinguish, per provider, between: contributed successfully, ran but found no match, failed/errored, timed out, and not-automated/manual-reference — and MUST NOT collapse any of these into a generic 'no result' or fabricate a result to fill a gap." | **Amended (extended)** — the status vocabulary gains an insufficient-query-evidence outcome so "we had nothing to search with" is not reported as "we searched and found nothing". | FR-011 |
| **FR-029** | "System MUST report, alongside the synthesized result, which providers were selected, which contributed, which failed/were unavailable, and which were not-automated…" | **Clarified** — unchanged; explicitly confirms `image` is not a provider in this list. | FR-025 |
| **FR-024** | "System MUST treat each provider integration as non-substitutable for another: provider workers return typed claims/evidence/citations… and no provider's result MAY silently override image evidence or another provider's claim without both being preserved and surfaced." | **Clarified** — the "MUST NOT silently override image evidence" clause becomes enforceable for the first time, because image evidence now exists as a claim. | FR-004, FR-017 |
| **FR-033** | "System MUST route confirmed changes through existing, Go-owned write paths only…" | **Extended, not superseded** — the wishlist destination is added within the same write paths. | FR-027, FR-028 |
| FR-001–FR-021, FR-026, FR-030–FR-032, FR-034–FR-037 | — | **Unchanged and still binding.** | — |

### ADR 0011 disposition

ADR 0011 ("Persisted Deep Agentic Coin Identification") remains **in force** for
the persistence/eventing/write-boundary decision. ADR 0012 amends exactly one
sentence of its Decision section — "The router records its selected provider set
and rationale" is retained as an outcome but the router ceases to be an LLM step
— and adds the image hypothesis as a first-class claim source. ADR 0011 is not
superseded in whole; ADR 0012 states the precise scope of its amendment.

### Contract drift reconciliation

Five documented drift points exist between
`specs/344-deep-agentic-coin-identification/contracts/agent-internal-contract.md`
and shipped code. Reconciling them is **in scope for 351 as documentation-only
corrections** (the contract file is the artifact this feature is changing the
semantics of; leaving it wrong while amending it would be negligent). Each is a
correction of the document to match shipped code, **not** a code change:

| § | Documented | Shipped | Correction |
|---|---|---|---|
| §1 | `services.InternalTokenService.Mint(userID)` / `middleware.InternalTokenRequired` | deep pipeline uses `MintForJob(userID, jobID)` / `InternalJobTokenRequired` | update names in the table |
| §2 | request key `llm_config`; `quick_evidence.numista_evidence` passthrough | `DeepIdentifyRequest.llm`; `QuickEvidence` is `extra="forbid"` and has no `numista_evidence` — sending it would be rejected | rename to `llm`; delete the `numista_evidence` line |
| §3 | `evaluation` payload `{disagreements:[…], resolved:[…]}`; no `synthesis_started` row | emitted payload is `{disagreement_count, resolved_count}`; `synthesis_started` is emitted | correct the payload shape; add the missing row |
| §5 | `DeepSynthesis` example omits `attributions` | `attributions` is emitted (Feature 345) | add it to the example |
| §7 | tool table lists `numista_search`, `numista_detail`, `nomisma_search` | `ocre_search` also exists (Feature 345 / ADR 0010) | add the `ocre_search` row |

Additionally, this feature adds to the same contract: the hypothesis schema
(new §4a), the `image` claim-source semantics, the deterministic router, the
insufficient-query-evidence outcome, and the additive `DeepSynthesis` hypothesis
field. See `contracts/vision-hypothesis.md`.

## Migration & Configuration Impact

- **Schema**: no database migration. The hypothesis and any new evidence refs are
  additive JSON inside the existing report/proposal columns.
- **Go mirror structs**: must tolerate reports **with and without** the
  hypothesis key. `buildDeepProposalDocumentJSON` already skips
  `provider == "image"` refs; no change is required there for image refs, and a
  proposed field whose only ref is an image ref MUST still be emitted (with an
  empty evidence array) rather than dropped.
- **Persisted jobs**: existing terminal jobs keep their stored report/proposal
  verbatim; no backfill, no re-run.
- **Settings**: no new admin setting is required. The existing
  `DeepIdentificationEnabled` flag remains the kill switch; whether a separate
  transitional flag is warranted is an open question (OQ-5).
- **Rollback**: reverting the code restores the prior pipeline. Reports written
  with a hypothesis remain readable because the additional key is ignored by the
  older reader; proposals written with `image` refs remain applicable because the
  older Go reader already skips those refs.

## Resolved Decisions

Questions raised during authoring and since answered. Recorded here rather than
deleted so the decision trail survives.

### RD-1 — Wishlist mechanism *(was OQ-3; resolved by Brian, 2026-08-16)*

**Question as raised**: a wishlist result had no clean home.
`models.Coin.IsWishlist` exists but `QuickCaptureDraft` has no wishlist flag, and
`Apply` targets were `draft` (intake source) and `coin` (saved-coin source).
Two options were tabled: **(a)** carry an `isWishlist` destination intent on the
intake apply path and create a wishlist `Coin` directly through `CoinService`;
**(b)** add a wishlist flag to `QuickCaptureDraft` and carry it through promotion.

**Decision: mechanism (a).** A confirmed intake-source result may be applied to a
new `models.Coin` with `IsWishlist = true`, created through the existing
`CoinService` create path. No schema migration. No change to `QuickCaptureDraft`.

**Rationale — (b) was rejected on two independent grounds:**

1. It would require a **database migration on a shipped table**, breaking this
   feature's explicit no-migration guarantee (see Migration and Rollback, and
   Non-Goals).
2. `deepProposalDraftFieldAllowlist`
   (`src/api/services/deep_identification_proposal.go:64`) is **four fields** —
   `workingTitle`, `era`, `dateRange`, `notes` — against the coin allowlist's
   fourteen. Routing a wishlist result through a draft would discard precisely
   the ruler, denomination, mint, and legend data this entire feature exists to
   produce. The wishlist item would arrive stripped of its identification.

**Verified against the code before adoption** (all four confirmed):

- `CoinService.CreateCoin` / `CreateCoinInTx` accept a caller-populated
  `*models.Coin`; `prepareCoinForCreate` (`services/coin_service.go:147`) reads
  `coin.IsWishlist` directly. The flag is genuinely settable on the create path.
- An exact precedent already exists:
  `quick_capture_service.go:530` sets `coin.IsWishlist = target ==
  QuickCapturePromotionTargetWishlist` from a **normalized target enum**, never
  from a user- or model-supplied field. FR-027 mirrors this shape.
- `Apply`'s target handling (`deep_identification_proposal.go:263-274`) is a
  closed `switch` with a `default:` rejection, so adding a third destination is
  additive and cannot silently widen anything.
- **`CoinService` nils `coin.References` for wishlist coins** in *both*
  `prepareCoinForCreate` and `createPreparedCoinInTx` ("belt-and-suspenders").
  This does **not** conflict with FR-021: the `coin_type` proposed field maps to
  `ReferenceText`, a plain string column (`models/coin.go:76`), which is distinct
  from the `References []CoinReference` relation (`models/coin.go:92`). Catalogue
  text survives on a wishlist coin; only the relational references are dropped,
  which is the pre-existing and intended invariant.

### RD-2 — Corroboration confidence upgrade *(was OQ-1; resolved by Brian, 2026-08-17)*

**Decision**: flat `min(1.0, max(image_confidence, provider_confidence) + 0.10)`
on exact normalized match. Applied **once per field**, with **no stacking**
across multiple corroborating providers. Never LLM-adjusted. Never exceeds 1.0.
The proposed default was accepted as written.

**Rationale**: corroboration is evidence that the reading is right, not evidence
proportional to how many catalogues happen to index the same type. Stacking would
let provider *coverage* masquerade as *certainty* — three catalogues indexing one
common Roman type would outrank two catalogues indexing a rare one, which is an
artifact of the corpora, not of the coin. A single bounded step keeps the number
explainable to the owner and keeps the rule auditable.

Binds **FR-022**. Interacts with RD-3: the upgrade is applied **before** the
acceptance-threshold comparison.

### RD-3 — Draft acceptance default *(was OQ-2; resolved by Brian, 2026-08-17)*

**Decision**: **confidence-driven, not source-driven.** A proposed field defaults
to **accepted at confidence ≥ 0.70** and **unaccepted below 0.70**, *regardless*
of whether the source is image-only or provider-corroborated. The threshold MUST
be a single named constant, not a literal scattered across call sites.

This **reverses** the previously stated default (image-only fields opt-in).

**Rationale**: a source-driven opt-in would force the owner to hand-tick every
single field on precisely the case this feature exists to serve. On the Maximinus
coin *every* field is image-only, so the owner would have confirmed a dozen
checkboxes to accept a result the system was already confident about — the
feature would have looked like it had done nothing. Confidence is the honest
signal; provenance is already communicated separately and visibly (FR-026).

**Explicit interaction with RD-2, and it must be tested**: a corroborated field at
0.62 receives the +0.10 upgrade, crosses 0.70, and becomes accepted-by-default.
That is intended behavior, not an accident of ordering, and it is exactly the kind
of emergent threshold effect that goes unnoticed without a dedicated test.

Binds **FR-026**.

### RD-4 — Reverse legend and type: ranking, not querying *(was OQ-4; resolved by Brian, 2026-08-17)*

**Decision**: **exclude reverse type/legend from query terms entirely**, and add
**no** second narrower probe. Instead use reverse legend/type as a **ranking and
disambiguation signal applied to candidate results a provider has already
returned** — zero additional upstream calls, zero additional call budget.

This is a change from the previously stated default (exclude + no second probe,
and nothing further). The ranking role is the new part.

**Rationale**: reverse legends are the least reliably legible text on a worn
ancient coin and the most likely to poison a query and turn a good match into
`no_match`. But once a provider has *already* returned five candidates, that same
weak signal is exactly the right tie-breaker — a wrong guess costs nothing but
ordering, whereas in a query it costs the whole result. Same signal, opposite
risk profile, depending on where it is applied.

**Scope call — this needed a new requirement, and it splits by provider:**

- **Numista and Nomisma — genuinely new behavior.** `providers/numista.py`
  requests `limit=5` and then takes `candidates[0]` unconditionally, discarding
  four candidates with no ranking whatsoever. Introducing a deterministic ranking
  over the already-returned set is new capability, so it gets its own
  requirement: **FR-039**. It is application-owned and deterministic; FR-009's
  "never freely chosen by an LLM" property extends from query text to ranking.
- **OCRE — NOT new scope, and ADR 0010 is not amended.** This mechanism already
  exists and is ADR-governed. `providers/ocre.py::_legend_tokens` already
  extracts legend tokens documented as *"scoring-only signals (never SPARQL)"*
  and passes them to `ocre_search`; `src/api/services/ocre_scoring.go:95` already
  applies `ocreLegendMatches` × `ocreLegendBonusPer`, capped at
  `ocreLegendBonusMax`, added to the base score, clamped to `[0,1]`, over a
  `sort.SliceStable`. The **only** change permitted here is widening the *source*
  of those tokens: `_legend_tokens` currently reads **only**
  `quick_evidence.label_text`, and must additionally draw from the hypothesis
  when quick evidence is absent — which is the entire Maximinus scenario.

  **The scoring math itself is ADR 0010's deterministic contract and MUST NOT be
  touched.** Weights, bonus-per-match, bonus cap, clamping, and stable sort all
  stay exactly as they are. This feature widens an input; it does not amend
  ADR 0010.

Binds **FR-010** (exclusion from query terms) and **FR-039** (the ranking role).

### RD-5 — Rollout: straight cutover *(was OQ-5; resolved by Brian, 2026-08-17)*

**Decision**: **straight cutover.** No transitional A/B flag. The existing
`SettingDeepIdentificationEnabled` remains the kill switch. Accepted as proposed.

**Rationale**: the old path is a known, reproduced failure. A transitional flag
would mean *maintaining the broken path* — keeping the provider-gated synthesis
alive, and doubling the synthesis test matrix, in order to preserve the ability
to fall back to output the owner has already rejected as useless. There is
nothing to A/B against. The kill switch already exists and is sufficient.

Binds **FR-034**. See also the phased beta-merge plan in `tasks.md`, which
provides staged validation without a second code path.

### RD-6 — Hypothesis visibility: build the panel *(was OQ-6; resolved by Brian, 2026-08-17)*

**Decision**: **build it** — a **collapsible** "what the images alone said"
section in the report UI, **default collapsed**. Existing design tokens, no new
font sizes, no hardcoded colors, no emojis.

This **reverses** the previously stated default (do not build it this feature).

**Rationale**: the original failure was undiagnosable *precisely because* the
owner could not see what vision had produced. The vision call ran, cost money,
and was discarded silently — and from the outside that was indistinguishable
from the vision call never having happened. This panel is a permanent diagnostic
surface, not decoration: it is what makes "the images were unreadable" visibly
different from "the pipeline dropped the result again". Default-collapsed keeps
it out of the way of the normal reading path.

Binds **FR-008**. Unblocks the report-panel task.

### RD-7 — OCRE routing signal: inclusion by default *(was OQ-7; resolved by Brian, 2026-08-17)*

**Decision**: **inclusion by default.** OCRE is skipped **only** on a *positive*
non-Roman-Imperial era signal from the hypothesis or quick evidence — never on
the mere absence of a Roman signal. Every skip MUST carry a stated reason in
`skipped[]`. The assumed default was accepted as written.

**Rationale**: absence of evidence is not evidence of absence. On a coin the
vision pass could not read — the exact failure case — there is no Roman signal
*and* no non-Roman signal, and skipping OCRE there would withhold the provider
most likely to identify a Roman Imperial coin at precisely the moment the system
knows least. Requiring a positive contrary signal means the failure mode is a
wasted query, not a missed identification.

The stated-reason requirement also depends on B4 being fixed: `skipped[]` is
currently computed and then dropped from the emitted frame, so today no skip
reason survives to the persisted event regardless of what the router decides.

Binds **FR-015**.

---

## Open Questions

**None. All seven open questions are resolved** — OQ-3 on 2026-08-16 and
OQ-1, OQ-2, OQ-4, OQ-5, OQ-6, OQ-7 on 2026-08-17, all by Brian. See the
**Resolved Decisions** section above (RD-1 through RD-7) for each decision, its
rationale, and the requirements it binds. No requirement in this specification is
awaiting an answer.
