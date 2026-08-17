# Implementation Plan: Vision-First Deep Identification

**Branch**: `351-vision-first-deep-identification` | **Date**: 2026-08-16 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/351-vision-first-deep-identification/spec.md`
**ADR**: [ADR 0012](../../docs/adr/0012-vision-first-deep-identification.md) (Proposed) — amends ADR 0011

## Summary

Invert the Deep Analysis information architecture. The vision node stops
emitting free prose that nothing reads and starts emitting a **typed, per-field
coin hypothesis** from the *same single LLM call that already runs on every job*.
That hypothesis becomes the pipeline's primary identification and is consumed by
every downstream node: provider queries are built from it deterministically
(deleting the `"unidentified ancient coin"` placeholder), provider selection
becomes a deterministic function of quick evidence + hypothesis + catalog +
bounds (replacing the LLM router), the evaluator treats it as a first-class claim
source so provider-vs-image contradictions surface, and synthesis narrates from
vision *and* providers while emitting draft fields with
`evidence_refs: [{"provider": "image"}]` — a shape the Feature 344 contract
already documents and the Go proposal builder already tolerates.

Providers become fact-checkers: they confirm (raise confidence, attach a
citation), refine (add fields the images could not give), or contradict (produce
a visible unresolved disagreement). `FALLBACK_NARRATIVE_NO_EVIDENCE` becomes
reachable only when *both* sources are empty.

**Net cost is negative**: the vision call count is unchanged (structured output
on the existing call), and the LLM router is removed — one fewer LLM call per
run without an override.

**Blast radius is deliberately small.** No database migration, no new provider,
no new public SSE event, no new write surface, no change to the Go
job/event/SSE/cancel/retry layer (audited production-quality) beyond three
narrow items: the quick-lookup budget (FR-038), making quick-lookup failure
observable (FR-029), and a wishlist apply destination (FR-027).

**A second, independent defect is in scope.** After the design analysis,
`deep_identification_pipeline_runner.go:112` was found to give the quick-lookup
pass a **15-second** budget for a full vision LLM round trip that gets **five
minutes** standalone (`agent_proxy.go:36`). On deadline exceed it returns `nil`
with only a `Warn`. That alone reproduces every symptom of the observed run, and
it is **independently shippable ahead of the redesign** — it is sequenced first
in `tasks.md` Phase 2 for exactly that reason.

**Audit remediation is folded in.** Per Brian's instruction, blockers **B1–B6**
and follow-ups **F1–F9** from the post-344 audit are carried by this branch and
are enumerated as tasks in `tasks.md` Phases 10–15.

## Technical Context

**Language/Version**: Python 3.12 / FastAPI + LangGraph (agent — the bulk of the change), Go 1.26.x (API — two narrow changes), Vue 3 + TypeScript (SPA — evidence-source rendering)
**Primary Dependencies**: Existing only. Structured vision output uses the LangChain structured-output binding already available to `app/llm/provider.py`; no new package.
**Storage**: No migration. The hypothesis is additive JSON inside the existing `DeepIdentificationJob` report column; image evidence refs are additive inside the existing proposal document.
**Testing**: `pytest tests/ -v` (agent — hypothesis schema, query precedence, deterministic router, evaluator image claims, synthesis fallback boundary, the Maximinus fixture), `go test ./...` (proposal builder with image-only fields, wishlist apply, quick-lookup outcome, backward-compat report fixture), `vue-tsc --build` + existing web tests.
**Target Platform**: Self-hosted three-service deployment (Go API, Python agent, Vue SPA).
**Project Type**: Web application — three services (Constitution §II).
**Performance Goals**: LLM calls per job MUST NOT increase (SC-007); −1 call on runs without a provider override. Existing `bounds` (max_providers, max_concurrency, provider_timeout_s, total_timeout_s, recursion_limit) are unchanged and still binding.
**Constraints**: Python stays stateless; query text never LLM-authored; confidence never LLM-adjusted; disagreement detection stays deterministic; additive-only schema; no new write surface; legend text never in logs.
**Scale/Scope**: ~6 agent files, ~3 Go files, ~2–3 web surfaces, 1 ADR, 1 contract update. Estimated **M**.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-checked after Phase 1 design.*

| Principle / Section | Assessment | Verdict |
|---|---|---|
| **I. Layered architecture** | Go changes stay in `services/` (pipeline runner, proposal service) + a thin handler param for the apply target; no SQL in handlers; DI unchanged. | ✅ Pass |
| **II. Service boundary** | Python remains stateless — the hypothesis lives only in per-request graph state and the returned synthesis. No DB handle, no credentials beyond the per-request LLM config. Go gains no LLM logic (the router becomes deterministic **in Python**, not in Go). | ✅ Pass |
| **III. Strict types & contracts** | The hypothesis is a Pydantic `StrictResponseModel`-family model with bounded strings and `[0,1]` confidences; the `DeepSynthesis` addition is optional and additive; Go mirror structs tolerate both shapes; Vue uses `?.`/`??` for the optional hypothesis. | ✅ Pass |
| **IV. Simple complete proportional** | Fixes the actual root cause (a write-only state field and provider-gated synthesis) rather than patching the fallback string. Deletes code (LLM router, placeholder constant) as well as adding it. No speculative abstraction. | ✅ Pass |
| **V. Security/privacy** | No new upstream integration, no new credential, no new egress. Legend/hypothesis values are excluded from logs and `progress` payloads; existing sanitizers still apply. | ✅ Pass |
| **VI. UX** | Image-derived fields are visibly marked; design tokens and existing components reused. No emojis. | ✅ Pass |
| **VII. CI / supply chain** | No new dependency. Conventional Commits + Copilot trailer. | ✅ Pass |
| **VIII. Documented decisions** | Multi-service contract semantics change + amendment of landed 344 requirements and ADR 0011 ⇒ **ADR required**. ADR 0012 (Proposed) authored with this plan. | ✅ Pass |
| **IX. Automated enforcement** | The "write-only state field" defect class is closed by an explicit test asserting every hypothesis consumer receives it (SC-004), plus the Maximinus regression fixture. | ✅ Pass |
| **§17 Quality Gate** | Workflow-contract check: touched shared surfaces are the Deep Analysis internal contract, the proposal document, and the apply write path; targeted regressions enumerated in Phase E below. | ✅ Pass |
| **§21 Definition of Done** | ADR added (0012); regression test for the exact failing path (Maximinus fixture); backward-compat fixture; Swagger on any changed handler; decisions captured in `.squad/decisions/inbox/`. | ✅ Pass |
| **§22 / §18.2 Amendment** | Landed 344 requirements (FR-022, FR-024, FR-025, FR-027, FR-028, FR-029) and ADR 0011 are amended. Path: ADR 0012 `Proposed` → PR links it → 351 spec restates superseded text verbatim → 344 spec gets a **header-only** supersession banner (body immutable, matching the ADR header-amendment convention in `docs/adr/README.md`) → decision recorded in `.squad/decisions/inbox/`. No Constitution Principle changes, so **no constitution semver bump and no §23 row**. | ✅ Pass |

**No violations. Complexity Tracking table intentionally empty.**

### Key design decision: the router becomes deterministic

Recorded here and in ADR 0012 because it removes an LLM step that Feature 344
specified.

**Decision: replace the LLM router with a deterministic, evidence-driven
selector.**

Evidence for the call:

- Only **three** providers are automatable (`numista`, `nomisma`, `ocre`) and
  `bounds.max_providers` is ≥ 3 in practice, so the router's decision space is
  nearly always "select all three".
- The shipped prompt already instructs the model to include a provider unless
  there is a strong reason to exclude it, and every failure path
  (`route()` exception handler, empty/unparseable selection) falls back to
  selecting all automatable providers. The LLM is therefore paying latency and
  tokens to usually reproduce a constant.
- It is the **only** nondeterministic step in an otherwise deterministic
  selection/merge/ranking design (`merge.py` sorts deterministically,
  `evaluator.detect_disagreements` is pure, OCRE scoring is deterministic by
  ADR 0010 / SC-005 of Feature 345). Determinism here makes SC-006 testable.
- FR-022 (344) requires routing on quick evidence + image evidence. Passing a
  structured hypothesis into a prompt and hoping for a stable subset is strictly
  worse than a rule that can be read, tested, and explained in the
  `router_selected` rationale the owner already sees.
- Removing it offsets the (small) added cost of structured vision output and
  satisfies SC-007.

Trade-off accepted: provider-selection nuance now lives in application code, so
adding providers later means editing a rule instead of a prompt. With three
providers and a closed, Go-supplied catalog that is the cheaper side. If the
automatable catalog grows past roughly six providers with genuinely
context-dependent applicability, an LLM router can be reintroduced behind the
same `RouterDecision` interface — the node boundary and the `router_selected`
frame shape are unchanged, so this is reversible.

**Preserved unchanged**: `provider_override` wins outright; the override can
never introduce a provider absent from the Go-supplied catalog; non-automatable
entries still run trivially and emit their own rows.

## Project Structure

### Documentation (this feature)

```text
specs/351-vision-first-deep-identification/
├── spec.md                       # Feature specification (authorizing document)
├── plan.md                       # This file
├── data-model.md                 # Phase 1 — hypothesis entity, persistence reuse, compatibility matrix
└── contracts/
    └── vision-hypothesis.md      # Phase 1 — hypothesis schema, image claim source, deterministic router,
                                  #           insufficient-query-evidence outcome, 344 drift corrections
```

Related ADR (outside the specs tree):
`docs/adr/0012-vision-first-deep-identification.md` (Proposed).
`tasks.md` is intentionally **not** authored yet — it follows via
`/speckit.tasks` once the open questions in spec.md are answered.

### Source Code (repository root)

```text
src/agent/app/                                   # Python agent — the bulk of the change
├── models/
│   └── responses.py             # EDIT  add CoinHypothesis / HypothesisField models;
│                                #       add optional DeepSynthesis.image_hypothesis (additive)
├── teams/deep_identification/
│   ├── graph.py                 # EDIT  prepare_evidence_node emits the typed hypothesis (same single
│   │                            #       call, structured output); state threading into router/fanout/
│   │                            #       evaluator/synthesizer; delete IMAGE_ANALYSIS_PROMPT prose path
│   ├── state.py                 # EDIT  replace write-only `image_analysis: str` with `hypothesis`
│   ├── hypothesis.py            # NEW   prompt + schema binding + normalization to coin-field vocabulary
│   │                            #       + hypothesis→claim-source adapter (pure, testable)
│   ├── query_terms.py           # NEW   deterministic precedence: quick_evidence → hypothesis → notes;
│   │                            #       shared by every provider node (no per-provider drift)
│   ├── router.py                # EDIT  deterministic selector; delete ROUTER_PROMPT and the LLM call
│   ├── evaluator.py             # EDIT  image hypothesis as a first-class claim source
│   ├── synthesis.py             # EDIT  accept the hypothesis; narrate from vision+providers; image-ref
│   │                            #       proposed fields; corroboration confidence rule; fallback boundary
│   ├── merge.py                 # EDIT  claim-source ordering that includes the image source
│   └── providers/
│       ├── numista.py           # EDIT  use query_terms; delete _DEFAULT_QUERY
│       ├── nomisma.py           # EDIT  use query_terms; delete placeholder
│       └── ocre.py              # EDIT  use query_terms; delete placeholder
└── tests/
    ├── test_deep_identification_hypothesis.py   # NEW  schema, empty/failure degrade, normalization
    ├── test_deep_identification_query_terms.py  # NEW  precedence + zero-placeholder + no-terms outcome
    ├── test_deep_identification_router.py       # EDIT determinism, hypothesis-driven skip, override
    ├── test_deep_identification_evaluator.py    # EDIT image-vs-provider disagreement
    ├── test_deep_identification_synthesis.py    # EDIT fallback boundary, image refs, corroboration
    └── test_deep_identification_maximinus.py    # NEW  the named end-to-end regression fixture

src/api/                                          # Go API — two narrow changes only
├── services/
│   ├── deep_identification_pipeline_runner.go    # EDIT typed quick-lookup outcome (FR-029);
│   │                                             #      keep image-only proposed fields (empty evidence)
│   ├── deep_identification_proposal.go           # EDIT wishlist apply destination (FR-027, RD-1 mechanism (a))
│   └── *_test.go                                 # EDIT image-only field, wishlist apply, back-compat fixture
└── handlers/
    └── deep_identification.go                    # EDIT apply-target plumbing + Swagger (RD-1)

src/web/src/components/deep-identification/       # Vue SPA
├── DeepReportPanel.vue          # EDIT mark image-derived fields; optional hypothesis surface (OQ-6)
└── DeepProposalEditor.vue       # EDIT image vs. cited evidence badge; wishlist save target (RD-1)
```

**Structure Decision**: Web application / three-service layout (Constitution
§II). The change is concentrated in the Python pipeline seam because that is
where the defect lives. Go and Vue changes are strictly the two behavioral items
(FR-027, FR-029) plus rendering of an additive field.

## Complexity Tracking

> No Constitution violations. Table intentionally empty.

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|--------------------------------------|
| — | — | — |

## Phased Delivery (M, ready for `/speckit.tasks`)

Each phase leaves the build green and the pipeline runnable. Phases A–C are
Python-only and independently reviewable.

- **Phase A — The hypothesis (keystone).** `hypothesis.py` (schema, prompt,
  structured-output binding, normalization to the coin-field vocabulary, typed
  empty degrade), `responses.py` models, `state.py` swap, `graph.py`
  `prepare_evidence_node` rewrite. Same single vision call. Tests: schema
  conformance, failure/timeout/unparseable → empty hypothesis, normalization
  mapping, and an assertion that the hypothesis is present in state after the
  node. **Nothing consumes it yet** — but the write-only defect is not fixed
  until Phase E, so Phase A alone must not be released.
- **Phase B — Query terms.** `query_terms.py` with the FR-010 precedence, wired
  into all three automatable provider nodes; delete `_DEFAULT_QUERY` and its
  siblings; add the insufficient-query-evidence outcome (zero upstream calls).
  Tests: precedence table, zero-placeholder assertion across all providers,
  no-terms path makes no call.
- **Phase C — Deterministic router.** Rewrite `route()` as a pure function of
  (catalog, override, bounds, quick evidence, hypothesis); delete `ROUTER_PROMPT`
  and the LLM invocation; keep `RouterDecision`, the `router_selected` frame, and
  override precedence identical. Tests: determinism across repeated runs,
  OCRE skip reasoning (per OQ-7), override precedence, empty-evidence
  inclusion bias.
- **Phase D — Evaluator + synthesis.** Image hypothesis as a claim source in
  `evaluator.py`; `synthesize()` accepts the hypothesis; narrative prompt
  restructured to "images say X / providers say Y"; `_build_proposed_fields`
  emits image refs; corroboration confidence rule (OQ-1); fallback narrative
  boundary. Tests: fallback only when both empty, image-only proposal fields,
  corroboration upgrade, contradiction withheld from proposal + surfaced.
- **Phase E — End-to-end regression.** The **Maximinus fixture** (two face
  images, empty notes, empty quick evidence, all providers `no_match`) asserting
  the full before/after table in spec.md US2, plus the SC-004 "every consumer
  reads the hypothesis" test. This phase is the gate: Phases A–D are not "done"
  until this passes.
- **Phase F — Go: wishlist destination + quick-lookup observability.** FR-027 via
  the settled RD-1 mechanism (a) — a `models.Coin` with `IsWishlist = true`
  created through the existing `CoinService` create path, as a third destination
  on `Apply`'s closed target switch, gated on the intake source; no migration and
  no `QuickCaptureDraft` change (handler + proposal service + Swagger + tests) —
  and FR-029 typed quick-lookup outcome. Backward-compatibility fixture: a report
  persisted without a hypothesis and a proposal with image-only fields both load,
  render, and apply.
- **Phase G — Web surfaces.** Image-derived vs. cited evidence marking, optional
  hypothesis panel (OQ-6), wishlist save target. `vue-tsc --build` clean.
- **Phase H — Docs, contract, ADR, gates.** Apply the five drift corrections plus
  the new sections to
  `specs/344-deep-agentic-coin-identification/contracts/agent-internal-contract.md`;
  add the header-only supersession banner to the 344 spec; promote ADR 0012
  `Proposed → Accepted` on merge; update `docs/adr/README.md` index; write
  `.squad/decisions/inbox/` entry; run the full §17/§21 gates.

**Effort**: M (~6 agent files + 2 new agent modules, ~3 Go files, ~2 web
surfaces, 1 ADR, 1 contract update).

**Risks**:

1. *Structured-output reliability across providers (Anthropic vs. Ollama).* A
   self-hosted Ollama model may return non-conformant JSON more often than
   Claude. Mitigated by FR-006 (typed empty hypothesis + existing retry) so a
   parse failure degrades to today's behavior rather than failing the job.
   Highest-uncertainty item in the plan.
2. *Hypothesis quality driving bad provider queries.* A confidently wrong ruler
   could send all three providers down the wrong path. Mitigated by quick
   evidence retaining higher precedence, by contradictions surfacing rather than
   resolving, and by the fact that today's alternative is a literal placeholder.
3. *Confidence inflation.* The corroboration rule (OQ-1) must be bounded and
   deterministic; an unbounded rule would make the confirm gate less meaningful.
4. *Scope creep into the Go layer.* Explicitly fenced: only FR-027 and FR-029.

## Constitution Re-Check (post-design)

Re-evaluated against the Phase 1 artifacts: no new violation. Python remains
stateless; Go gains no LLM logic; all schema changes are additive and optional;
no new write surface, credential, or egress is introduced; the change deletes as
much machinery as it adds. The amendment of landed Feature 344 requirements and
of ADR 0011 is authorized by ADR 0012 and recorded in spec.md rather than by
rewriting locked files. ✅ Pass.
