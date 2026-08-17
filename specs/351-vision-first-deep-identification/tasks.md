# Tasks: Vision-First Deep Identification

**Branch**: `351-vision-first-deep-identification`
**Input**: `spec.md`, `plan.md`, `data-model.md`,
`contracts/vision-hypothesis.md` (all in
`specs/351-vision-first-deep-identification/`), plus
`docs/adr/0012-vision-first-deep-identification.md` and the Feature 344
contracts this work amends.

**Tests**: Included and required. §21.6 mandates a targeted regression for the
exact failing user path; that regression (Phase 9, **T060**) is the single most
valuable artifact in this effort and is the gate the whole feature is measured
against.

**Organization**: Phase 1 is setup and the decision tasks. Phase 2 is the
**runtime blocker** — an independent, ship-on-its-own defect that alone
reproduces every symptom of Brian's failed run. Phases 3–8 are the design
redesign, strictly dependency-ordered behind the keystone (Phase 4, the typed
hypothesis). Phase 9 is the end-to-end regression gate. Phases 10–15 are the
audit remediation Brian folded into this work (B1–B6, F1–F9) plus the Go/web
surfaces. Phase 16 is the §17 Quality Gate.

## Format: `[ID] [P?] [FR/audit ID] Description`

- **[P]**: Can run in parallel (different files, no unresolved dependency)
- **[x] + [RESOLVED …]**: A decision task Brian has answered. Kept rather than
  deleted so the decision trail survives; the decision is recorded in spec.md
  under Resolved Decisions (RD-1..RD-7) and repeated inline here.
- **There are no blocked tasks.** All seven open questions were resolved by
  Brian (OQ-3 on 2026-08-16; OQ-1, OQ-2, OQ-4, OQ-5, OQ-6, OQ-7 on 2026-08-17),
  and every former "DEFAULT IF FORCED" has been promoted to specified behavior.
- Every task names an exact existing or new file path and cites the spec
  requirement ID(s) and/or the audit ID (B1–B6, F1–F9, NE-1, NE-2) it discharges
- **NE-1** = quick-evidence 15s timeout runtime defect; **NE-2** = no
  structured-output pattern exists anywhere in `src/agent/app`

---

## Working constraints for this branch

- `npm run` is blocked by PowerShell execution policy on this machine — use
  **`npm.cmd run <script>`** in every verification task.
- Do not switch branches. Do not commit without Brian's review.
- Constitution §18.2: landed specs/ADRs are amended via ADR 0012 + the
  supersession banner already added — never by rewriting locked bodies.

---

## Phase 1: Setup & Decisions

**Purpose**: Establish the dependency-free groundwork and force every open
question into an explicit, recorded decision rather than an implicit one.

- [ ] T001 [P] Confirm no new third-party dependency is introduced by this feature — diff `go.mod`, `src/agent/pyproject.toml`, `src/web/package.json`; structured output must use the LangChain surface already vendored (plan.md Technical Context)
- [ ] T002 [P] [FR-034] Record the rollback statement in the PR template body: no migration, revert-only, pre-351 reports remain readable (`docs/adr/0012-vision-first-deep-identification.md` § Rollback)
- [x] T003 **[RESOLVED by Brian, 2026-08-16 — mechanism (a) adopted as recommended]** [FR-027] **Decision task — wishlist persistence mechanism.** Brian's stated intent makes wishlist **required scope**, not optional. Two mechanisms were tabled, both routing through existing Go-owned write services (344 FR-033):
  - **(a) Direct wishlist coin — ✅ ADOPTED.** An `isWishlist` destination intent on the intake apply path; `DeepIdentificationProposalService.Apply` creates a `models.Coin` with `IsWishlist=true` via `CoinService`, using the existing `deepProposalCoinFieldAllowlist` (14 fields). No schema migration; no change to `QuickCaptureDraft`. Files: `src/api/services/deep_identification_proposal.go`, `src/api/handlers/deep_identification.go`.
  - **(b) Wishlist-flagged draft — ❌ REJECTED.** Would add a wishlist flag to `models.QuickCaptureDraft` and carry it through `QuickCaptureService` promotion.
  - **Rejection rationale (two independent grounds).** (1) (b) requires a **database migration on a shipped table**, contradicting this feature's no-migration guarantee (FR-033/FR-034) and widening blast radius into Quick Capture — a workflow this feature otherwise does not touch (Principle IV). (2) `deepProposalDraftFieldAllowlist` (`src/api/services/deep_identification_proposal.go:64`) is only 4 fields (`workingTitle`, `era`, `dateRange`, `notes`) against the coin allowlist's 14, so a wishlist *draft* would discard the ruler/denomination/mint/legend this feature exists to produce.
  - **Counter-argument recorded**: (a) skips the draft review step intake normally provides — mitigated because the Deep Analysis proposal editor **is** the review step, and the confirm gate (FR-028) is unchanged.
  - **Recorded in**: `spec.md` §Resolved Decisions RD-1; `docs/adr/0012` §6 + Alternatives Considered; `.squad/decisions/inbox/2026-08-16-oq3-wishlist-resolution.md`. Unblocks T072, T073.
- [x] T004 **[RESOLVED by Brian, 2026-08-17 — proposal accepted as written]** [FR-022] Decision task — corroboration confidence rule. **DECIDED**: flat `min(1.0, max(image_conf, provider_conf) + 0.10)` on exact normalized match, applied **once per field regardless of how many providers corroborate (no stacking)**, never LLM-adjusted, never above 1.0. Rationale: stacking would let provider *coverage* masquerade as *certainty* — three catalogues indexing a common type would outrank two indexing a rare one, an artifact of the corpora rather than of the coin. Recorded as spec.md RD-2; binds FR-022; implemented by T061.
- [x] T005 **[RESOLVED by Brian, 2026-08-17 — REVERSES the stated default]** [FR-021, FR-026] Decision task — proposal-editor acceptance default. **DECIDED: confidence-driven, NOT source-driven.** A proposed field defaults to **accepted at confidence ≥ 0.70** and **unaccepted below 0.70**, *regardless* of whether the source is image-only or provider-corroborated. Rationale: source-driven opt-in would force the owner to hand-tick every field on precisely the case the feature exists to serve — on the Maximinus coin every field is image-only, so the feature would have looked like it did nothing. **Interaction with T004 that must be tested**: a corroborated field at 0.62 + 0.10 crosses the threshold and becomes accepted-by-default; that is intended (T120). Recorded as spec.md RD-3; binds FR-026; implemented by T090 + T120.
- [x] T006 **[RESOLVED by Brian, 2026-08-17 — CHANGES the stated default: adds a ranking role]** [FR-010, FR-039] Decision task — reverse type/legend in provider queries. **DECIDED: exclude from query terms entirely, and add NO second probe. Instead use reverse legend/type as a ranking and disambiguation signal applied to results a provider has ALREADY returned** — zero extra upstream calls, zero extra budget. Rationale: a weak reverse legend poisons a query (costs the whole result) but is the right tie-breaker once five candidates are already in hand (costs only ordering). **Scope call**: new for Numista/Nomisma (`numista.py:69` takes `candidates[0]` unconditionally, discarding four unranked) → new **FR-039**. **NOT new for OCRE, and ADR 0010 is NOT amended** — `ocre.py::_legend_tokens` already passes "scoring-only signals (never SPARQL)" and `ocre_scoring.go:95` already applies a capped legend bonus over a stable sort; only the *token source* widens to include the hypothesis. Recorded as spec.md RD-4; implemented by T035 + T121 + T122.
- [x] T007 **[RESOLVED by Brian, 2026-08-17 — proposal accepted as written]** [FR-034] Decision task — transitional flag vs. straight cutover. **DECIDED: straight cutover**, no transitional A/B flag; `SettingDeepIdentificationEnabled` remains the kill switch. Rationale: a transitional flag would mean *maintaining the broken provider-gated path* and doubling the synthesis test matrix, in order to preserve a fallback to output the owner has already rejected as useless. Staged validation comes from the beta merge plan (see Merge Groups) rather than a second code path. Recorded as spec.md RD-5.
- [x] T008 **[RESOLVED by Brian, 2026-08-17 — REVERSES the stated default: build it]** [FR-008] Decision task — hypothesis report panel. **DECIDED: build it**, as a **collapsible** "what the images alone said" section, **default collapsed**, using existing design tokens with no new font sizes, no hardcoded colors, and no emojis. Rationale: the original failure was undiagnosable *precisely because* the owner could not see what vision produced — a silent discard is externally indistinguishable from the call never happening. This is a permanent diagnostic surface, not decoration. Recorded as spec.md RD-6; binds FR-008; implemented by T091.
- [x] T009 **[RESOLVED by Brian, 2026-08-17 — proposal accepted as written]** [FR-014, FR-015] Decision task — OCRE's "Roman Imperial signal". **DECIDED: inclusion by default.** OCRE is selected whenever it is automatable and within bounds, and skipped **only on a *positive* non-Roman-Imperial era signal** from the hypothesis or quick evidence (e.g. `era` matching greek/islamic/byzantine/modern) — never on the mere absence of a Roman signal. **Every skip must carry a stated reason in `skipped[]`** (depends on B4/T047, without which no skip reason survives to the persisted event). Rationale: absence of evidence is not evidence of absence — on an unreadable coin there is neither signal, and skipping there withholds the most likely provider at the moment the system knows least. Recorded as spec.md RD-7; binds FR-015; implemented by T046.
- [ ] T010 [P] [B5] Decision task — `AGENT_DEEP_*` settings: clamp or delete? **DECIDED: clamp.** `src/agent/app/config.py:22-29` declares five settings that `.env.example:34-42` documents as enforced service-level ceilings, and grep proves zero readers anywhere in `app/`. They will be **read and used to clamp** the incoming `request.bounds` (`bounds = min(request_value, setting_value)` per field) in the deep-identify stream entry point, making the documented promise true. Rejected alternative: deleting them removes an operator's only lever over the agent service's own resource consumption and silently narrows a documented contract. Implemented by T077.

---

## Phase 2: 🔥 Runtime Blocker — quick-evidence budget and observability (NE-1)

**⚠️ CRITICAL**: This phase is **independent of the entire vision redesign** and
alone deterministically reproduces every symptom of Brian's run: NGC rendering
the generic "Manual verification" link-out with no cert, and all three providers
falling back to the placeholder query and returning `no_match`. It is the
highest value-per-line work on the branch and should land first.

**Root cause**: `src/api/services/deep_identification_pipeline_runner.go:112`
gives `extractQuickEvidence` a **15-second** budget. That call chain is
`CoinLookupService.Lookup` → `extractDataFromImages` → `proxy.AnalyzeCoin` — a
full vision LLM round trip through the Python service on two phone photos. The
same call standalone gets **five minutes** (`src/api/services/agent_proxy.go:36`,
`requestClient: &http.Client{Timeout: 5 * time.Minute}`). A 20x mismatch. On
deadline exceed, `extractQuickEvidence` (`runner.go:426-432`) returns `nil` with
only a `Warn` — silently indistinguishable from "this coin had no quick
evidence".

- [ ] T011 [NE-1, FR-038] Replace the magic literal `15*time.Second` at `src/api/services/deep_identification_pipeline_runner.go:112` with a named, admin-tunable budget: add `SettingDeepIdentificationQuickLookupTimeoutSeconds` (key + default + validated read) to `src/api/services/settings_service.go` and surface it on `DeepIdentificationSettings`. **DEFAULT VALUE: 90 seconds**, validated range 5–300 (never above the agent proxy's own 5-minute `requestClient` ceiling, which remains the hard upper bound)
- [ ] T012 [NE-1, FR-038] **⚠️ Verification task, not an assumption** — measure and record the interaction between the enlarged quick-lookup budget and the pipeline's remaining budget at `src/api/services/deep_identification_pipeline_runner.go:116-123`. Facts to work from: `SettingDeepIdentificationHardTimeoutSeconds` defaults to **300**; `deepPipelineHardTimeoutSafetyMarginS = 20`; quick evidence is consumed from the same `ctx` **before** `deepPipelineBounds` computes `TotalTimeoutS`, so every second spent on quick lookup is subtracted from the pipeline. At today's 15s the pipeline gets ~265s; at 90s it would get ~190s. **Required outcome**: either raise the `HardTimeoutSeconds` default so the pipeline budget does not shrink (**DEFAULT: 300 → 420**, giving 420−90−20 = 310s ≥ today's 265s), or prove with a test that the pipeline completes within the reduced budget. Do not ship T011 without discharging this task.
- [ ] T013 [NE-1, FR-038] Add a Go test in `src/api/services/deep_identification_pipeline_runner_test.go` asserting that (a) the quick-lookup context deadline equals the configured setting, and (b) `bounds.TotalTimeoutS` after quick lookup is still ≥ `deepPipelineMinTotalTimeoutS` for the default settings combination decided in T012
- [ ] T014 [NE-1, FR-029] Make quick-evidence outcome **typed** instead of `nil`-or-value: change `extractQuickEvidence` (`src/api/services/deep_identification_pipeline_runner.go:399-467`) to return `(*DeepQuickEvidenceProxy, quickLookupOutcome)` where the outcome is one of `ok`, `no_data`, `unavailable` (deadline/error) — replacing the current `Warn`-and-`nil` at lines 426-432 which conflates the last two (data-model.md §5)
- [ ] T015 [NE-1, FR-029, FR-030] Emit the quick-lookup outcome as an **observable job event**: append a `progress` envelope (existing public event type — no new SSE vocabulary, spec Non-Goals) carrying only the typed outcome class, and record it on the job so the report can state it. **Privacy: outcome class only** — no label text, no cert number, no notes, no image data (FR-030, 344 FR-036). Files: `src/api/services/deep_identification_pipeline_runner.go`
- [ ] T016 [NE-1, FR-029] Surface the quick-lookup outcome in the report/UI so an owner can distinguish "no cert data existed" from "the NGC quick look did not complete" — `src/web/src/components/deep-identification/DeepReportPanel.vue`
- [ ] T017 [P] [NE-1, FR-029] Tests: `src/api/services/deep_identification_pipeline_runner_test.go` — forced `Lookup` error yields `unavailable` (job still completes), empty result yields `no_data`, success yields `ok`; assert no user content appears in the emitted payload or logs
- [ ] T018 [P] [NE-1] Add a note to `docs/features/ai-analysis.md` recording that Quick Lookup inside Deep Analysis is budget-limited and admin-tunable, and that its failure is now reported rather than silent

**Checkpoint**: Brian's exact run now succeeds at the quick-evidence stage
(NGC cert extracted, providers receive real quick-evidence query terms) **before
any vision-redesign code lands**, and a quick-lookup failure is visible.

---

## Phase 3: Foundational — structured-output capability (NE-2)

**⚠️ CRITICAL**: There is **no** structured-output pattern anywhere in
`src/agent/app` — zero grep hits for `with_structured_output`,
`PydanticOutputParser`, or `response_format`. This feature introduces the first
one, and it must work across **both** providers, because
`app/llm/provider.py::get_chat_model` returns `ChatAnthropic` **or** `ChatOllama`
from per-request config. Anthropic does tool-based structured output reliably;
Ollama varies sharply by model. Blocks Phase 4.

- [x] T019 [NE-2, FR-001] Add `get_structured_model(config: LLMConfig, schema)` to `src/agent/app/llm/provider.py`, alongside the existing `get_chat_model` / `get_search_model` factories and following their exact shape (per-request config, no module-level state, no credential retention). Anthropic path uses tool-based structured output; Ollama path uses its JSON/format mode
- [x] T020 [NE-2, FR-006] **DEVIATION (Cassius, 2026-08-17, see `.squad/decisions/inbox/cassius-vision-hypothesis.md`)**: Implemented the parse-failure-degrades path in `src/agent/app/llm/provider.py` + `src/agent/app/teams/deep_identification/hypothesis.py`: on schema-validation failure, retry once (bounded — only fires on failure, so the happy path stays one call), then a prose extraction attempt, then — **not directly to the typed-empty hypothesis as originally written here** — to the deterministic quick-evidence hypothesis (`build_hypothesis_from_quick_evidence`), which itself degrades to typed-empty only when `quick_evidence` is absent/empty. This ladder rung is strictly better than typed-empty and was already the seam's existing fallback from a prior batch; it is what makes Brian's Maximinus coin still produce a usable hypothesis when vision fails. Vision is not a new single point of failure — every branch degrades and the job still reaches a terminal state
- [x] T021 [P] [NE-2, FR-006] Tests in `src/agent/tests/test_llm_provider_structured.py`: Anthropic-shaped fake returns a conformant object; Ollama-shaped fake returns malformed JSON surfaced as `parsing_error` (not an exception); job-level exception count is zero in every branch (full ladder degrade behavior — retry/prose/deterministic-fallback — is exercised end to end in `test_deep_identification_hypothesis.py`)
- [x] T022 [P] [NE-2, FR-032] Assert statelessness of the new factory: no module-level cache of models, config, or credentials (`src/agent/tests/test_llm_provider_structured.py`)

**Checkpoint**: Structured output exists as a reusable, provider-agnostic,
fail-soft capability. Nothing consumes it yet.

---

## Phase 4: 🔑 KEYSTONE — the typed vision hypothesis (B2)

**⚠️ CRITICAL — BLOCKS PHASES 5, 6, 7, 8.** Every consumer depends on this
shape. `src/agent/app/teams/deep_identification/graph.py:62-86` currently runs a
full vision LLM call and writes `state["image_analysis"]`; a repository-wide
grep returns exactly three hits — the declaration (`state.py:38`) and the two
writes (`graph.py:71,86`). **Zero reads.**

- [x] T023 [B2, FR-001] `HypothesisField`/`CoinHypothesis` already existed (prior batch) in `src/agent/app/models/hypothesis.py` — not `responses.py` as this task literally names, a location decision predating this batch that this batch did not relitigate. Extended additively this batch with `notes`/`coin_type` (data-model.md §3's full 14-field vocabulary was previously only 12 fields)
- [x] T024 [B2, FR-005] Created the vision prompt (`VISION_HYPOTHESIS_PROMPT`), the schema binding via `get_structured_model` (T019), and normalization into the coin-field vocabulary in `src/agent/app/teams/deep_identification/hypothesis.py::build_hypothesis_from_vision`. Snake_case/camelCase key aliasing + allowlist filtering happens in the prose-fallback parser (`_parse_prose_hypothesis`); the structured-parse path is schema-bound so keys are already conformant there. Unknown keys dropped; no new writable field introduced
- [x] T025 [B2, FR-003] "Omit, never guess" encoded in `VISION_HYPOTHESIS_PROMPT` and in `_normalize_vision_hypothesis`/`_parse_prose_hypothesis` post-validation (era/material canonicalization drops non-conforming values rather than forwarding them)
- [x] T026 [B2, FR-002, FR-031] Rewrote `prepare_evidence_node` in `src/agent/app/teams/deep_identification/graph.py` to return the typed hypothesis from the same single vision call. `IMAGE_ANALYSIS_PROMPT`'s prose-only path is deleted entirely (constant, prompt text, and the free-prose `image_analysis` write). No second vision call is introduced — confirmed by `fake.calls == 1` assertions on the happy path in `test_deep_identification_hypothesis.py`
- [x] T027 [B2, FR-006] **DEVIATION, same as T020** — wired in `build_hypothesis_from_vision`: LLM raise, timeout, empty content, and schema-validation failure all degrade (via prose extraction, then the deterministic quick-evidence hypothesis) rather than jumping straight to typed-empty; the job never fails for any of these. See `.squad/decisions/inbox/cassius-vision-hypothesis.md`
- [x] T028 [B2, FR-007] Replaced the write-only `image_analysis: str` field with `hypothesis: CoinHypothesis` in `src/agent/app/teams/deep_identification/state.py` (already present from the prior batch's seam; the dead `image_analysis` field is now deleted)
- [x] T029 [B2] Rewrote the `state.py` docstring. It now honestly states that, as of this batch, only the synthesizer reads `hypothesis` — router/query-construction/evaluator wiring is Phase 5-7 scope and not yet done, so the B2 "write-only state field" defect is only partially fixed by this batch
- [x] T030 [B2, FR-008] `DeepSynthesis.image_hypothesis: CoinHypothesis | None = None` added (additive) to `src/agent/app/models/responses.py`; `synthesis.py::synthesize()` populates it whenever the hypothesis is non-empty
- [x] T031 [P] [B2, FR-001, FR-003] Tests added to `src/agent/tests/test_deep_identification_hypothesis.py`: schema conformance, unsupported-field omission, confidence bounds, era/material canonicalization dropping non-canonical values, and unknown/prose-recovered key normalization
- [x] T032 [P] [B2, FR-006] **DEVIATION, same as T020/T027** — tests assert LLM raise / timeout / empty / malformed each degrade to the deterministic quick-evidence hypothesis (or typed-empty when quick-evidence is itself absent), with the job never raising, in `src/agent/tests/test_deep_identification_hypothesis.py`
- [x] T033 [B2, FR-004] Tests added: `HypothesisField` carries no citation field; `image` is rejected by `ProviderCoverageEntry`/`DeepProviderCatalogEntry` (the closed `ProviderName` union); `DeepSynthesis.image_hypothesis` is additive and separate from `coverage`/`attributions` (`src/agent/tests/test_deep_identification_hypothesis.py`)

**Checkpoint**: The hypothesis exists, is typed, degrades safely, and is
persisted. **It still has no consumers — the B2 defect is not fixed until
T057 passes.** Do not release at this checkpoint.

---

## Phase 5: Provider query terms — deleting the placeholder

**Depends on**: Phase 4 (needs the hypothesis shape).

- [x] T034 [FR-009, FR-010, FR-012] Create `src/agent/app/teams/deep_identification/query_terms.py` — one shared, pure, deterministic builder used by **every** automatable provider node, implementing the precedence `quick_evidence.numista_query` → `quick_evidence.label_text` → hypothesis-derived terms → `notes[:200]` (contracts/vision-hypothesis.md §2). Query text stays application-authored; no LLM may choose, rewrite, or extend it
- [x] T035 [FR-010] Hypothesis-derived composition order inside `query_terms.py`, **fixed and specified** (RD-4): `ruler + denomination` → `ruler` → `denomination + material` → `obverseInscription`. **Reverse type and reverse legend are excluded from query terms entirely**, and **no** second narrower probe is issued. Add a test asserting no generated query string ever contains a reverse-legend or reverse-type term
- [x] T036 [FR-011] Delete `_DEFAULT_QUERY = "unidentified ancient coin"` from `src/agent/app/teams/deep_identification/providers/numista.py` and the equivalent placeholder fallbacks in `providers/nomisma.py` and `providers/ocre.py`. `ocre.py` never had this constant — its existing zero-signal short-circuit already returned `no_match`/`call_count=0` with no placeholder, so no change was needed there
- [x] T037 [FR-011] Add the `insufficient_query_evidence` member to `ProviderErrorKind` in `src/agent/app/models/responses.py`, and return `status="no_match"`, `error_kind="insufficient_query_evidence"`, `call_count=0` with **zero upstream calls** when no precedence tier yields terms (data-model.md §5)
- [x] T038 [P] [FR-012] Wire `query_terms.py` into `providers/numista.py`. **DEVIATION (Cassius, 2026-08-17, see `.squad/decisions/inbox/cassius-query-terms.md`)**: `numista.run()`/`nomisma.run()`/`ocre.run()` all gained a new trailing `hypothesis: CoinHypothesis | None = None` keyword parameter (default-safe, byte-identical for every existing positional call site) rather than editing `graph.py`'s `fn(entry, tools, quick_evidence, notes)` call, per this batch's explicit "do NOT touch graph.py" boundary (another agent owns it concurrently). Until `graph.py`'s provider-fanout call site is updated to pass `state["hypothesis"]` through, hypothesis-derived terms/ranking are wired and tested but not yet reachable in a real run — that one-line wiring is the only remaining step and is called out for whoever next touches `graph.py`
- [x] T039 [P] [FR-012] Wire `query_terms.py` into `providers/nomisma.py` — same deviation as T038 (`hypothesis` param added, `graph.py` wiring deferred)
- [x] T040 [P] [FR-012] Wire `query_terms.py` into `providers/ocre.py` — the OCRE slug-binding, deterministic scoring, ODbL attribution, and default-off posture of ADR 0010 are **unchanged**; only the terms it receives change. (OCRE's type-bearing bound slots (`ruler`/`denomination`/`mint`/`ocre_id`) still come only from `quick_evidence.coin_fields`, unchanged from before — `query_terms.py` is not applicable to those slots, only `_legend_tokens` widening, see T122)
- [x] T041 [P] [FR-011] **Zero-placeholder test** in `src/agent/tests/test_deep_identification_query_terms.py`: assert no provider node can ever issue a call whose query is a placeholder constant, across all three nodes, including the empty-everything input
- [x] T042 [P] [FR-010] Precedence table test covering all four tiers plus the quick-evidence-wins-over-hypothesis case (`src/agent/tests/test_deep_identification_query_terms.py`)
- [x] T043 [P] [FR-011] Test that the no-terms path performs **zero** `ProviderToolsClient` calls and consumes zero budget (`src/agent/tests/test_deep_identification_query_terms.py`)
- [x] T121 [FR-039] **Reverse-legend ranking for Numista and Nomisma** (new behavior, RD-4). `src/agent/app/teams/deep_identification/providers/numista.py:69` takes `candidates[0]` unconditionally after requesting `limit=5`, discarding four candidates unranked; `nomisma.py` has the same shape. Add a shared, pure, deterministic ranker (e.g. `src/agent/app/teams/deep_identification/candidate_ranking.py`) that scores already-returned candidates against hypothesis reverse legend/type and other unused hypothesis fields, and selects the top-ranked rather than the first. **Zero additional upstream calls, zero additional call budget** — this operates only on results already in hand. Application-owned and deterministic; an LLM MUST NOT choose or reorder candidates (FR-009's property extends to ranking). Ties break stably on the provider's original order
- [x] T122 [FR-039] **Widen the OCRE legend-token source — and do NOT touch the scoring math** (RD-4). `src/agent/app/teams/deep_identification/providers/ocre.py::_legend_tokens` currently reads **only** `quick_evidence.label_text`, so on the Maximinus scenario (no quick evidence) it contributes nothing. Extend it to also draw tokens from the hypothesis (`obverseInscription`, `reverseInscription`, reverse type) when quick evidence is absent, preserving the existing normalization, dedup, and 12-token cap. **⚠️ ADR 0010 CONSTRAINT**: `src/api/services/ocre_scoring.go` — `ocreLegendMatches`, `ocreLegendBonusPer`, `ocreLegendBonusMax`, the base-score weights, the `[0,1]` clamp, and the `sort.SliceStable` — is ADR 0010's deterministic contract and MUST NOT be modified. This task widens an input only; it does not amend ADR 0010. State that explicitly in the PR. No Go file was touched — verified by a test that greps `ocre_scoring.go` for the ADR-anchored symbols
- [x] T123 [P] [FR-039] Tests: candidate ranking is deterministic and reproducible for identical inputs; ranking never triggers an upstream call (assert `call_count` is unchanged versus the unranked path); an empty/unreadable hypothesis leaves ordering at the provider's original order; OCRE token widening produces tokens from the hypothesis when quick evidence is absent, and `ocre_scoring.go` is **untouched** by the diff (`src/agent/tests/test_deep_identification_candidate_ranking.py`)

**Checkpoint**: Providers can no longer be sent to search for nothing.

---

## Phase 6: Deterministic router (B3, B4)

**Depends on**: Phase 4. `graph.py:89-97` currently passes only
catalog/override/`max_providers`/`notes` to `route()` — 344 FR-022's "plus image
evidence" was never implemented (B3). Separately, `router.py:100-118` computes a
populated `skipped[]` and then **drops it** from the `router_selected` frame
(`graph.py`), so persisted events permanently carry `"skipped":[]` in violation
of internal contract §3 (B4).

- [x] T044 [B3, FR-013, FR-014] Rewrite `route()` in `src/agent/app/teams/deep_identification/router.py` as a **pure function** of `(catalog, provider_override, bounds, quick_evidence, hypothesis)`. Delete `ROUTER_PROMPT` and the LLM invocation; drop the `model` parameter. Keep `RouterDecision` and the `router_selected` frame shape byte-identical (ADR 0012 §3)
- [x] T045 [B3, FR-013] Pass quick evidence and the hypothesis from `router_node` (`src/agent/app/teams/deep_identification/graph.py:89-97`) into `route()`
- [x] T046 [FR-015] Implement the provider-skip rules in `router.py`, **specified** (RD-7): **inclusion by default** — select every provider that is automatable and within bounds. Skip OCRE **only on a *positive* non-Roman-Imperial `era` signal** from the hypothesis or quick evidence (e.g. greek/islamic/byzantine/modern); the mere *absence* of a Roman signal MUST NOT cause a skip. Every skip — evidence-driven or bounds-driven — carries a stated reason in `skipped[]` (depends on T047, without which the reason is dropped from the emitted frame). `provider_override` still wins outright and can never introduce a provider absent from the Go-supplied catalog. Add a test that an empty-evidence run selects **all** automatable providers including OCRE
- [x] T047 [B4] **Emit `skipped[]` in the `router_selected` frame** — add the dropped field in `src/agent/app/teams/deep_identification/graph.py`. Verify the Go translator (`deepRouterSelectedPublicPayloadJSON` in `src/api/services/deep_identification_pipeline_runner.go`) already carries it through to the persisted public payload; it parses `skipped` today and has simply never received a non-empty one
- [x] T048 [P] [FR-014] Determinism test in `src/agent/tests/test_deep_identification_router.py`: two identical runs produce byte-identical `selected`, `skipped`, and `rationale` (SC-006)
- [x] T049 [P] [B4] Test that a run which skips at least one provider emits a **non-empty** `skipped[]` in the `router_selected` frame and that it survives Go-side translation (`src/agent/tests/test_deep_identification_router.py` + `src/api/services/deep_identification_pipeline_runner_stream_test.go`)
- [x] T050 [P] [FR-015] Tests: override precedence unchanged; catalog closure enforced; empty-evidence run selects all automatable providers (`src/agent/tests/test_deep_identification_router.py`)

**Checkpoint**: Routing is deterministic, evidence-driven, explainable, and its
skip list is no longer silently discarded.

---

## Phase 7: Evaluator — image as a first-class claim source

**Depends on**: Phase 4.

- [x] T051 [FR-016] Extend `src/agent/app/teams/deep_identification/evaluator.py::_group_claims_by_field` (and `detect_disagreements`) to flatten hypothesis fields into `(field, source="image", value, confidence)` tuples alongside provider claims, reusing the **existing** `value.strip().lower()` normalization
- [x] T052 [FR-017] Emit image refs as `EvidenceRef(provider="image")` with no `claim_index` in `DisagreementEntry.claim_refs`; a provider-vs-image conflict is `resolution: "unresolved"` and is resolved by precedence in **neither** direction
- [x] T053 [FR-018] Confirm and test that detection remains **LLM-free** — the optional LLM in `evaluator.py::_summarize` may only phrase the human-facing question and can never add, remove, or resolve a disagreement
- [x] T054 [FR-016] Extend `src/agent/app/teams/deep_identification/merge.py::sort_claims` to give the `image` source a deterministic rank position so ordering stays reproducible with a mixed claim set; citation-host validation continues to apply **only** to provider claims (an image claim has no citation)
- [x] T055 [P] [FR-016, FR-017] Tests in `src/agent/tests/test_deep_identification_evaluator.py`: provider contradicts image → unresolved disagreement referencing both sources; provider agrees with image → no disagreement, `resolved_count` incremented; image-only field → no disagreement

**Checkpoint**: A provider that contradicts what is plainly legible on the coin
is now visible. 344 FR-027 is real for the first time.

---

## Phase 8: Synthesis, narrative, and proposed fields

**Depends on**: Phases 4, 5, 6, 7. `synthesis.py:107-110` currently gates the
narrative on `contributing`, and `synthesize()` does not even accept the image
analysis as a parameter; `_build_proposed_fields()` reads only provider claims.

> **2026-08-17 (Cassius)**: T056/T058/T059/T060/T061/T063/T065/T066/T067 landed
> ahead of Phases 3-7 to fix the Maximinus no-evidence-fallback defect
> (`.squad/decisions/inbox/cassius-hypothesis-seam.md`). The `hypothesis` state
> key is populated in `prepare_evidence_node` by a **deterministic, LLM-free
> adapter over `quick_evidence`**
> (`app/teams/deep_identification/hypothesis.py`), not by a real vision call —
> Phase 3/4 will replace only that adapter's body with the actual
> single-vision-LLM-call output; every consumer (synthesis today) is wired
> against the same `hypothesis` state key and needs no further change for that
> swap. T057 (wiring the hypothesis into the router/query-term-builder/
> evaluator — all four SC-004 consumers) is explicitly **deferred** to
> Phases 3-7, which is where those consumers are (re)built; T062/T064 describe
> pre-existing behavior that remains structurally unchanged (contradicted
> fields already skip `proposed_fields`; provider-only fields already pass
> `validate_citations`) but were not independently re-verified with new tests
> in this pass.

- [x] T056 [FR-019] Change the `synthesize(...)` signature in `src/agent/app/teams/deep_identification/synthesis.py` to accept the hypothesis, and thread it from `synthesizer_node` **and** from the total-timeout partial-synthesis fallback path in `src/agent/app/teams/deep_identification/graph.py` (both call sites — the timeout path must not silently keep the old behavior)
- [ ] T057 [B2, FR-007] **Close the B2 defect class.** Add the SC-004 test asserting that the hypothesis is received by **all four** consumers — router, query-term builder, evaluator, synthesis — so a future write-only state field fails CI rather than review (Principle IX). File: `src/agent/tests/test_deep_identification_hypothesis.py`. **Deferred**: only the synthesis consumer is wired today; router/query-term-builder/evaluator wiring lands with Phases 3-7.
- [x] T058 [FR-019] Rewrite `NARRATIVE_PROMPT` and its input assembly in `src/agent/app/teams/deep_identification/synthesis.py` to narrate "what the images support / what each provider confirmed, refined, or contradicted / what remains open", using `CoinHypothesis.observations` plus the typed fields
- [x] T059 [FR-020] Change the fallback gate at `src/agent/app/teams/deep_identification/synthesis.py:107-110`: `FALLBACK_NARRATIVE_NO_EVIDENCE` is reachable **only** when the hypothesis is empty **and** no provider contributed. Absence of provider contributions alone MUST NOT trigger it
- [x] T060 [FR-021] Rewrite `_build_proposed_fields` in `src/agent/app/teams/deep_identification/synthesis.py` to merge hypothesis fields with provider claims; an image-only field is proposed at its hypothesis confidence with `evidence_refs: [{"provider": "image"}]`
- [x] T061 [FR-022] Implement the corroboration confidence upgrade in `_build_proposed_fields`, **specified** (RD-2): `min(1.0, max(image_conf, provider_conf) + 0.10)` on exact normalized match, applied **once per field, no stacking** across multiple corroborating providers; both refs attached; provider citation carried; never LLM-adjusted; never > 1.0. Add an explicit **no-stacking test**: three providers corroborating the same field yield the same confidence as one
- [ ] T062 [FR-023, FR-024] Preserve today's behavior for provider-only fields (value + citation) and assert every proposed field has ≥1 source ref, with provider refs still passing the per-provider citation-host allowlist (`src/agent/app/teams/deep_identification/merge.py::validate_citations`)
- [x] T063 [FR-025] Assert `image` never enters `_build_coverage` or `_build_attributions` in `src/agent/app/teams/deep_identification/synthesis.py`; `ProviderCoverageEntry.provider` and `ProviderAttribution.provider` remain the `ProviderName` literal union
- [ ] T064 [FR-017] Keep contradicted fields **out** of `proposed_fields` (existing disagreement-field skip) while surfacing them in `disagreements` and `unresolved_questions`
- [x] T065 [P] [FR-020] Test: provider-empty + hypothesis-present run does **not** emit `FALLBACK_NARRATIVE_NO_EVIDENCE`; hypothesis-empty + provider-empty run **does** (`src/agent/tests/test_deep_identification_synthesis.py`)
- [x] T066 [P] [FR-021, FR-022] Tests: image-only field carries the image ref at hypothesis confidence; corroborated field carries both refs, the citation, and the bounded upgrade (`src/agent/tests/test_deep_identification_synthesis.py`)
- [x] T067 [P] [FR-025] Test: `image` absent from coverage and attributions in every fixture (`src/agent/tests/test_deep_identification_synthesis.py`)

**Checkpoint**: The pipeline narrates from vision and providers, and produces
draft fields without a provider match.

---

## Phase 9: 🎯 THE REGRESSION GATE — the Maximinus run

**Purpose**: The single most valuable artifact in this effort. This is the test
that would have caught Brian's junk output, and it is the proof the feature
works. Phases 4–8 are **not done** until this passes.

- [x] T068 [FR-001..FR-026, SC-001, SC-002] Create `src/agent/tests/test_deep_identification_maximinus.py` — the named end-to-end fixture: two legible face images, **empty notes**, **empty quick evidence**, all three automatable providers stubbed to `no_match`, NGC `not_automated`. Assert, per spec.md US2's before/after table: narrative is **not** the fallback and names ruler + denomination; `proposed_fields` has **≥4** entries; every one carries `evidence_refs: [{"provider":"image"}]`; no provider was called with a placeholder; the hypothesis is present in the emitted synthesis payload
- [x] T069 [SC-003] Corpus-wide assertion in the same module: across every fixture in the deep-identification test suite, **zero** provider calls carry a placeholder query string
- [x] T070 [FR-021, FR-033] Go-side companion in `src/api/services/deep_identification_proposal_integration_test.go`: a synthesis whose `proposed_fields` are image-only must produce a **non-empty** proposal document — each such field present with an **empty** `evidence` array, not dropped (`buildDeepProposalDocumentJSON` already skips `provider == "image"` refs; the field itself must survive)
- [x] T071 [FR-033] Backward-compatibility fixture in `src/api/services/deep_identification_pipeline_runner_test.go`: a pre-351 report **without** `image_hypothesis` and a post-351 proposal **with** image-only fields both load, render, and apply with zero errors

**Checkpoint**: The exact failing user path from Brian's screenshots is covered
by an automated test (§21.6).

---

## Phase 10: Go — wishlist destination (FR-027)

**Depends on**: T003 (**resolved — mechanism (a)**). Required scope per Brian's
stated intent ("saved as a draft for either a wishlist item or collection item").

**Mechanism is settled and verified against the code** — see `spec.md` §Resolved
Decisions RD-1. Implement (a); do **not** touch `QuickCaptureDraft`.

- [x] T072 [FR-027] Implement the wishlist destination in `src/api/services/deep_identification_proposal.go::Apply`. **Specified behavior (mechanism (a), settled):** add a third destination to the existing closed `switch target` (`deep_identification_proposal.go:263-274`), gated on `job.Source == models.DeepJobSourceIntake` exactly as the `draft` target is. It creates a `models.Coin` with `IsWishlist = true` through `CoinService.CreateCoin`/`CreateCoinInTx`, populated from the accepted fields via the **existing, unwidened** `deepProposalCoinFieldAllowlist` (14 fields). **No schema migration.** Three verified constraints the implementation must respect:
  - `isWishlist` is a **destination intent, never a proposed field** (FR-028). Do not add it to `deepProposalCoinFieldAllowlist`; derive it only from the normalized target, mirroring `services/quick_capture_service.go:530`.
  - `CoinService` deliberately nils `coin.References` for wishlist coins in **both** `prepareCoinForCreate` and `createPreparedCoinInTx`. This is fine and must not be worked around: the `coin_type` field maps to `ReferenceText`, a plain string column (`models/coin.go:76`), which is **not** the `References` relation (`models/coin.go:92`), so catalogue text survives.
  - Reuse the existing `ApplyJob` CAS so the new destination inherits already-applied idempotency.
- [x] T073 [FR-027] Plumb the target through the handler with Swagger annotations (§21.10) — `src/api/handlers/deep_identification.go` — normalizing it through a closed switch with a `default:` rejection and rejecting unknown targets exactly as today
- [x] T119 [FR-027] **(added after the T003 resolution — adjacent gap found while verifying it.)** `models.Coin.Name` is `gorm:"not null"` (`models/coin.go:45`) but there is **no `name` key in `deepProposalCoinFieldAllowlist`**, so mechanism (a) must derive a name for the newly created coin. Specify and implement the derivation, reusing the existing `buildDeepIntakeProposalFields` `workingTitle` logic (ruler + denomination) rather than inventing a second rule, and define the fallback when the hypothesis yields neither. Add a test for the empty-hypothesis case so a coin can never be created with a blank name. Also confirm whether a `validateCoinMinimumForPromotion`-equivalent check is wanted here (`services/quick_capture_service.go:533`) or whether `CoinService`'s own era/category validation is sufficient
- [x] T074 [FR-028] Assert the confirm gate is unchanged: no coin or draft row is created or modified before explicit confirmation, on **either** destination (`src/api/services/deep_identification_proposal_test.go`)
- [x] T075 [P] [FR-027, FR-028] Tests: wishlist apply lands an owner-scoped `models.Coin` with `IsWishlist=true` carrying the accepted fields; collection apply is byte-identical to today; already-applied idempotency (`ApplyJob` CAS) still holds for the new destination; and a **negative test asserting `isWishlist` can never be set from `proposed_fields`** — a proposal containing an `isWishlist` key must be rejected by the allowlist, not honored
- [ ] T076 [P] [FR-027] Add the save-destination choice to `src/web/src/components/deep-identification/DeepProposalEditor.vue` using existing design tokens and `.btn` classes; no emojis

---

## Phase 11: Go & agent hardening (B1, B5, F2)

Parallel-safe with Phases 5–9 (different files, no dependency on the hypothesis).

- [x] T077 [B5] Implement the **clamp** decided in T010: read the five `AGENT_DEEP_*` settings from `src/agent/app/config.py:22-29` in the deep-identify stream entry point (`src/agent/app/routes.py` / `src/agent/app/teams/deep_identification/graph.py::run_deep_identification_stream`) and clamp each incoming `request.bounds` field to `min(request_value, setting_value)`. Add a test proving an over-ceiling request is clamped, and correct `.env.example:34-42` prose if any wording overstates the mechanism
- [x] T078 [B1] **Fix the unbounded-map memory leak**: `src/api/services/deep_provider_budget_tracker.go:57` `Reset(jobID)` has zero production callers (grep-verified — only tests), so the tracker accumulates one entry per `(jobID, provider)` **forever**. The janitor structurally cannot reach it: the tracker is constructed at `src/api/main.go:112` and injected only into `DeepProviderToolsHandler` (`main.go:811`), a separate object graph. Fix by injecting the tracker into the job service and calling `Reset(job.ID)` from `runJob`'s terminal path alongside `DeleteHintArtifacts`, so every terminal outcome (completed/failed/cancelled) releases the entries
- [x] T079 [P] [B1] Test in `src/api/services/deep_identification_service_test.go`: after a job settles in **each** terminal state, the tracker holds zero entries for that job id; a concurrently running second job's entries are untouched
- [x] T080 [F2] Separate the internal job-token signing secret from the user JWT secret. `src/api/services/internal_token_service.go:70-84` HMACs job tokens with `cfg.JWTSecret` — the **same** secret as user JWTs — while the TTL was widened from 30s to up to `total_timeout_s + 30`. Introduce a dedicated secret (derived via HKDF from the existing secret with a distinct info label, or a separate configured value), so a job-token compromise cannot be leveraged against user JWTs
- [x] T081 [F2] Revoke the job token when the job settles: add a terminal-state revocation check (job-id → settled) consulted by `VerifyForJob`, so a long-TTL token cannot be replayed after its job reaches a terminal state — `src/api/services/internal_token_service.go`, `src/api/middleware/*`
- [x] T082 [P] [F2] Tests: a token minted for job A does not verify after job A settles; a token signed with the user JWT secret does not verify as a job token, and vice versa (`src/api/services/internal_token_service_test.go`)
- [x] T083 [P] [FR-030] Audit every new log/event emission added by this feature for user content: hypothesis values, legend text, owner notes, and query strings must not reach application logs or `progress` payloads (`src/agent/app/teams/deep_identification/*`, `src/api/services/deep_identification_pipeline_runner.go`). **AMENDED 2026-08-17 (FR-040, see `.squad/decisions/inbox/cassius-progress-detail-emission.md`)**: the `progress`-payload clause is narrowed by Brian-authorized FR-040 — the owner-scoped stream MAY now carry hypothesis values/query terms/candidate counts/per-provider outcomes. Audited and confirmed: the application-log prohibition (`logger.info/exception`, Python `logging`) remains fully intact — no hypothesis value, query term, or legend text was added to any `logger.*` call; new `caplog`-asserted test (`test_fr040_hypothesis_and_query_detail_never_reach_application_logs`) proves it directly through the real stream entry point
- [ ] T084 [P] [FR-031] Assert the LLM call count per job does not increase: count invocations in a fake-model harness for a no-override run and prove it is **one fewer** than the pre-351 baseline (router removed) — `src/agent/tests/test_deep_identification_maximinus.py`

---

## Phase 12: Web surfaces (B6, F6, FR-008, FR-026)

**⚠️ This phase is NOT a single mergeable unit — split it for `beta`:**

- **12a (T085–T088)** — B6 badge, F6 composable extraction, `disabled` prop.
  Genuinely independent of the vision work; safe to merge alone (ships in MG-2).
- **12b (T089–T092, T120)** — image-vs-cited marking, confidence-driven
  acceptance, the hypothesis panel. These **render data that does not exist
  until Phase 8** and must ship with the vision group (MG-3). Merging them
  earlier renders an empty panel and evidence marks that never appear.

- [x] T085 [B6] Fix the lying connection badge. `src/web/src/composables/useDeepIdentificationStream.ts:175-177` clears `connected`/`streaming` in a `finally` on **any** exit, and `src/web/src/components/deep-identification/DeepAnalysisProgressTimeline.vue:57-73` then renders "Reconnecting…" while nothing ever reconnects — the badge lies indefinitely and only a manual page reload recovers. **Fix**: implement a real reconnect action using the existing `since`/`Last-Event-ID` resume semantics (`contracts/sse-events.md` §3) **or**, if a real reconnect is out of proportion, rename the state to "Disconnected", surface `stream.error`, and add an explicit Retry button. Do not ship a state label the code cannot honor
- [x] T086 [P] [B6] Component test asserting the badge never displays "Reconnecting…" when no reconnect is scheduled, and that the retry control recovers the stream (`src/web/src/components/deep-identification/__tests__/`)
- [x] T087 [F6] Extract `useDeepAnalysisLauncher` from `src/web/src/components/coin-detail/CoinActionsPanel.vue` (372-line script, 9 responsibilities) and consume it from **both** that component and `src/web/src/pages/CoinLookupPage.vue:332-338`, where the launch logic is currently duplicated verbatim. 344 bolted the launch on instead of extracting a composable; this is the debt it left
- [x] T088 [F6] Wire the unwired `DeepAnalysisEntryButton.disabled` prop so a user at the `MaxActivePerUser` limit sees a disabled control with an explanatory title instead of an error toast after clicking — `src/web/src/components/deep-identification/DeepAnalysisEntryButton.vue` and both call sites
- [ ] T089 [FR-026] Mark image-derived proposed fields visibly distinct from provider-cited fields in `src/web/src/components/deep-identification/DeepReportPanel.vue` and `DeepProposalEditor.vue`, using existing design tokens (`--text-muted` label, `.chip-sm`) — no new font sizes, no hardcoded colors, no emojis
- [ ] T090 [FR-021, FR-026] Acceptance default in `src/web/src/components/deep-identification/DeepProposalEditor.vue`, **specified** (RD-3): **confidence-driven, not source-driven** — a field renders **accepted at confidence ≥ 0.70** and **unaccepted below 0.70**, regardless of whether its source is image-only or provider-corroborated. Consume the single named threshold constant from T120; do **not** inline `0.70`
- [ ] T091 [FR-008] **Build** the "what the images alone said" hypothesis panel in `src/web/src/components/deep-identification/DeepReportPanel.vue` (RD-6). **Collapsible, default collapsed.** Renders the typed hypothesis fields with their per-field confidence plus `observations`, and states plainly when `legible: false` so "the images were unreadable" is visibly distinct from "the pipeline dropped the result". Existing design tokens only (`--text-muted`, `--border-subtle`, `.chip-sm`, `--radius-sm`); no new font sizes, no hardcoded colors, no emojis. Must render correctly for a pre-351 report that has **no** `image_hypothesis` key (hide the section entirely, do not render an empty shell)
- [ ] T092 [P] [FR-026] Component tests for the image-vs-cited evidence marking and the disagreement rendering (`src/web/src/components/deep-identification/__tests__/`)
- [ ] T120 [FR-026, RD-3] **Named acceptance-threshold constant.** Define the 0.70 default-acceptance threshold as a **single named constant** consumed by every call site — the synthesis-side default if one is emitted, and `DeepProposalEditor.vue` (T090). It MUST NOT appear as a bare literal anywhere. Add the **RD-2 × RD-3 interaction test**: a field at image confidence 0.62 that a provider corroborates receives the +0.10 upgrade, crosses 0.70, and renders **accepted by default** — asserted explicitly so the emergent threshold effect cannot regress unnoticed. Also assert an image-only field at 0.85 renders accepted (source does not gate acceptance) and a provider-corroborated field at 0.40 renders unaccepted

---

## Phase 13: Contract, docs, and record reconciliation (F1, F8, F9)

- [x] T093 [F1, FR-035] Apply the five documentation-only drift corrections to `specs/344-deep-agentic-coin-identification/contracts/agent-internal-contract.md` (all grep-verified against shipped code, no behavior change): §1 `Mint(userID)`/`InternalTokenRequired` → `MintForJob(userID, jobID)`/`InternalJobTokenRequired`; §2 `llm_config` → `llm`; §2 delete the `quick_evidence.numista_evidence` line (`QuickEvidence` is `StrictRequestModel(extra="forbid")` and would **reject** it); §3 `evaluation` payload → `{disagreement_count, resolved_count}` **and** add the missing `synthesis_started` row; §5 add `attributions` to the `DeepSynthesis` example; §7 add the `ocre_search` row
- [x] T094 [FR-035] Add the new sections to the same contract file: the `CoinHypothesis` schema, `image` claim-source semantics, the deterministic router, the `insufficient_query_evidence` outcome, and the additive `DeepSynthesis.image_hypothesis` key — sourced from `specs/351-vision-first-deep-identification/contracts/vision-hypothesis.md`
- [x] T095 [F8] Correct the Feature 344 record: `specs/344-deep-agentic-coin-identification/tasks.md:180` still calls `ocre.py` a "stub, NO SPARQL" (T062) and line 358 still marks **T155 [DEFERRED]**, while the CHANGELOG records both as shipped via Feature 345 / ADR 0010. Update both lines to reflect reality with a pointer to Feature 345 — record correction only, no code change
- [x] T096 [F9] Make the version string derive from one canonical source. Root `VERSION` reads `4.0` while Swagger/OpenAPI/CHANGELOG read `4.0.0`, and the UI renders `4.0.<sha>` — a cosmetic mismatch that **directly undermined diagnosing Brian's run** because the deployed build could not be identified with confidence. Pick `VERSION` as canonical, and have the build/Swagger/OpenAPI/UI all derive from it (`VERSION`, `Taskfile.yml`, `src/api/main.go` Swagger annotations, `src/web` build define)
- [x] T097 [P] [FR-035] Update `docs/ARCHITECTURE.md` and `docs/features/ai-analysis.md` to describe the vision-first pipeline (hypothesis → deterministic router → provider fact-checking → synthesis) replacing the provider-first description
- [x] T098 [FR-035] Promote `docs/adr/0012-vision-first-deep-identification.md` from `Proposed` to `Accepted` on merge and update the status column in `docs/adr/README.md` (§22 step 7)
- [x] T099 [P] Append the implementation decision note to `.squad/decisions/inbox/` (never edit `.squad/decisions.md` directly, §18.2) recording the B5 clamp choice, the OQ defaults actually taken, and the quick-lookup budget/hard-timeout numbers finally chosen

---

## Phase 14: Dead code & structural debt (F5, F7)

**Sequenced last on purpose** so the refactor moves settled code, per Brian's
instruction on F5.

- [ ] T100 [F7] **Handle `build_graph()` carefully — do not just delete it.** `src/agent/app/teams/deep_identification/graph.py::build_graph` is test-only: production uses the hand-written async generator `run_deep_identification_stream`, so the topology test currently asserts on a graph users never run. The honest fix is to **first** rewrite the topology test to assert on the production generator's node sequence, **then** delete `build_graph`. Deleting it without that replacement loses coverage
- [ ] T101 [P] [F7] Remove the remaining grep-verified dead code, one commit per group, each with a grep proof in the commit body: `numista_detail()` (`src/agent/app/tools/provider_tools.py`), `RPCEnabled` (`src/api/services/settings_service.go` — **verify** it is not the flag ADR 0011 relies on to keep RPC unavailable before removing; if it is, keep it and record why), `listDeepIdentificationJobs`, `stream.reset()`, `DeepStreamTruncatedPayload`
- [ ] T102 [P] [F7] Remove or wire the unread request fields `hint_kind`, `call_budget`, `schema_version` and the unread state fields `state.errors`, `tools_base_url`, `internal_token` (`src/agent/app/models/requests.py`, `src/agent/app/teams/deep_identification/state.py`). **Caution**: `schema_version` and `call_budget` are part of the Go↔Python wire contract §2 — if they are removed from Python they must be removed from the contract and the Go mirror in the same change, or kept and documented as forward-compatibility placeholders. Prefer keeping wire fields and documenting them; remove only the genuinely internal ones
- [ ] T103 [F5] Split `src/api/services/deep_identification_service.go` (980 lines, five responsibilities) along its existing seams — job lifecycle, worker pool, janitor/retention, SSE broker, capability/limits — into separate types in the same package with constructor injection (Principle I). **No behavior change**; the Phase 9 regression suite is the safety net
- [ ] T104 [F5] Decompose `Run` in `src/api/services/deep_identification_pipeline_runner.go` (161 lines with an 8-case inline closure) — extract the per-frame translation cases into named methods so each is unit-testable in isolation. **No behavior change**

---

## Phase 15: Seam & concurrency coverage (F3, F4)

- [ ] T105 [F3] Close the concurrency gap: `src/api/services/deep_identification_service_test.go:1110` proves only **exact-duplicate** submit idempotency. Add a test for **two concurrent jobs for the same coin with different image bytes** — currently unguarded, in a feature whose entire history is concurrency bugs. Decide and encode the intended semantics (distinct fingerprints ⇒ both allowed, subject to `MaxActivePerUser`; or same-coin ⇒ reuse), and assert it under `-race`
- [ ] T106 [F4] Add a Go↔Python **seam test** that boots the real Python service and the real Go handler over HTTP and asserts one full `DeepIdentifyRequest` → SSE → `DeepSynthesis` round trip. Today both sides maintain wire fixtures by convention only — exactly the shape of the 080e598 production bug. Mark it CI-excluded/tagged if it cannot run unattended, and document how to run it in `docs/testing.md`
- [ ] T107 [P] [F4, FR-035] Add a contract-drift test asserting the Pydantic request/response models and the Go mirror structs agree on field names and nullability for `DeepIdentifyRequest`/`DeepSynthesis`, so the five drift points fixed in T093 cannot silently reappear

---

## Phase 16: Quality Gate & release verification (§17 / §21)

**Note**: `npm run` is blocked by PowerShell execution policy on this machine —
use `npm.cmd run`.

- [ ] T108 `go build ./...` and `go vet ./...` clean from `src/api/`
- [ ] T109 `go test ./...` green from `src/api/`, including `go test -run TestArchitecture ./...` and **`go test -run TestNoDirectDatabaseImports .`** (Principles I and IX)
- [ ] T110 [P] `ruff check app/ tests/` clean from `src/agent/`
- [ ] T111 [P] `pytest tests/ -v` green from `src/agent/`, with `test_deep_identification_maximinus.py` explicitly confirmed passing (Phase 9 gate)
- [ ] T112 [P] `npm.cmd run type-check` (`vue-tsc --build`, Docker-equivalent strictness — Principle III) clean from `src/web/`
- [ ] T113 [P] `npm.cmd run test` (vitest) green from `src/web/`
- [ ] T114 `npm.cmd run build` green from `src/web/`
- [ ] T115 Regenerate API docs if the surface changed by T073 (`task openapi`) and confirm `route_openapi_drift_test.go` is green (§21.11)
- [ ] T116 PR self-check per §17/§21: cite Principle I/II/III/IV/VIII/IX and §17/§21/§22; mirror the 18-item DoD; Conventional Commit prefix; `Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>` trailer; confirm no secrets in the diff
- [ ] T117 Manual verification of the real Maximinus coin end-to-end on a running stack — **supplementary confirmation only**; it does not substitute for T068
- [ ] T118 [FR-036, FR-037] Explicit non-regression assertion: run the pre-existing Feature 344/345 test suites unmodified (`src/api/services/deep_identification_*_test.go`, `src/api/handlers/internal_tools_test.go`, `src/agent/tests/test_deep_identification_*`) and confirm the only behavioral deltas are FR-027 (wishlist destination) and FR-029 (quick-lookup observability). Any other test requiring modification is a scope breach and must be justified in the PR or reverted

---

## Coverage Table

Every FR-001..FR-039 and every B1–B6 / F1–F9 / NE-1 / NE-2 maps to at least one
task. No orphan tasks: every task above cites at least one ID.

### Spec requirements

| FR | Tasks | | FR | Tasks |
|---|---|---|---|---|
| FR-001 | T019, T023, T024, T031 | | FR-020 | T059, T065 |
| FR-002 | T026 | | FR-021 | T060, T066, T070, T090 |
| FR-003 | T025, T031 | | FR-022 | T004, T061, T066 |
| FR-004 | T029, T033 | | FR-023 | T062 |
| FR-005 | T024, T031 | | FR-024 | T062 |
| FR-006 | T020, T021, T027, T032 | | FR-025 | T063, T067 |
| FR-007 | T028, T057 | | FR-026 | T089, T090, T092 |
| FR-008 | T008, T030, T091 | | FR-027 | T003, T072, T073, T075, T076, T119 |
| FR-009 | T034 | | FR-028 | T074, T075 |
| FR-010 | T034, T035, T042 | | FR-029 | T014, T015, T016, T017 |
| FR-011 | T036, T037, T041, T043 | | FR-030 | T015, T083 |
| FR-012 | T034, T038, T039, T040 | | FR-031 | T026, T084 |
| FR-013 | T044, T045 | | FR-032 | T022 |
| FR-014 | T044, T048 | | FR-033 | T070, T071 |
| FR-015 | T046, T050 | | FR-034 | T002, T007 |
| FR-016 | T051, T054, T055 | | FR-035 | T093, T094, T097, T098, T107 |
| FR-017 | T052, T055, T064 | | FR-036 | T012, T118 |
| FR-018 | T053 | | FR-037 | T040, T074, T109, T118 |
| FR-019 | T056, T058 | | FR-038 | T011, T012, T013, T018 |
| — | — | | FR-039 | T006, T035, T121, T122, T123 |

### Audit items and new evidence

| ID | Tasks |
|---|---|
| **NE-1** quick-evidence 15s timeout | T011, T012, T013, T014, T015, T016, T017, T018 |
| **NE-2** no structured-output pattern | T019, T020, T021, T022 |
| **B1** budget-tracker memory leak | T078, T079 |
| **B2** vision output never read | T023–T033, T057 |
| **B3** router gets no evidence | T044, T045 |
| **B4** `skipped[]` dropped from frame | T047, T049 |
| **B5** `AGENT_DEEP_*` unread settings | T010 (decision: clamp), T077 |
| **B6** lying "Reconnecting…" badge | T085, T086 |
| **F1** contract drift (5 points) | T093 |
| **F2** job token secret/TTL/revocation | T080, T081, T082 |
| **F3** concurrent same-coin jobs | T105 |
| **F4** no real Go↔Python seam test | T106, T107 |
| **F5** 980-line service / 161-line Run | T103, T104 |
| **F6** CoinActionsPanel + duplicated launch + unwired `disabled` | T087, T088 |
| **F7** twelve dead-code items | T100 (build_graph, carefully), T101, T102 |
| **F8** 344 tasks.md stale T155/T062 | T095 |
| **F9** VERSION 4.0 vs 4.0.0 | T096 |

**Unmapped**: none. Two items carry explicit caution flags rather than blind
execution — `RPCEnabled` in T101 (verify it is not the ADR 0011 gate before
removal) and `schema_version`/`call_budget` in T102 (wire-contract fields;
prefer documenting over deleting).

---

## Dependencies & Execution Order

### Phase dependencies

- **Phase 1 (Setup/Decisions)** → no dependencies.
- **Phase 2 (Runtime blocker, NE-1)** → **independent of everything else**. Can
  land and ship first, on its own, and alone fixes Brian's observed run.
- **Phase 3 (Structured output, NE-2)** → depends on Phase 1 only. **Blocks
  Phase 4.**
- **Phase 4 (Keystone hypothesis)** → depends on Phase 3. **BLOCKS Phases 5, 6,
  7, 8, 9.** This is the critical path.
- **Phase 5 (Query terms)**, **Phase 6 (Router)**, **Phase 7 (Evaluator)** →
  each depends on Phase 4; **independent of each other** and may run in
  parallel by different people.
- **Phase 8 (Synthesis)** → depends on Phases 4, 5, 6, 7 (it consumes all of
  them).
- **Phase 9 (Regression gate)** → depends on Phase 8. Nothing in Phases 4–8 is
  "done" until T068 passes.
- **Phase 10 (Wishlist)** → depends on T003 only; independent of the vision work.
- **Phase 11 (Hardening)** → independent of Phases 3–9 (different files).
- **Phase 12 (Web)** → T089/T090 depend on Phase 8's output shape; T085–T088
  are independent.
- **Phase 13 (Docs/contract)** → T093/T094 should follow Phases 5–8 so the
  contract documents what actually shipped; T095/T096/T097 are independent.
- **Phase 14 (Dead code/refactor)** → **deliberately last**, so F5's refactor
  moves settled code and the Phase 9 suite is the safety net.
- **Phase 15 (Seam tests)** → after Phase 8, before Phase 16.
- **Phase 16 (Quality gate)** → last.

### Critical path

```text
T019/T020 (structured output)
  → T023/T024/T026/T028 (typed hypothesis)
    → T034 (query terms) + T044 (router) + T051 (evaluator)   [parallel]
      → T056/T059/T060 (synthesis)
        → T068 (Maximinus regression gate)
          → T108-T114 (quality gate)
```

### Parallel execution groups

```text
# Immediately, by different people:
Phase 2 (T011-T018)   ← ship-first runtime fix, Go
Phase 11 (T077-T084)  ← hardening, Go + agent config
Phase 12 T085-T088    ← web debt, Vue
T095, T096, T097      ← record/version/docs corrections

# After Phase 4 lands:
Phase 5 (T034-T043) | Phase 6 (T044-T050) | Phase 7 (T051-T055)   → three-way parallel

# Test authoring (different files, no shared state):
T031, T032, T033      → parallel
T041, T042, T043      → parallel
T048, T049, T050      → parallel
T065, T066, T067      → parallel
T110, T111, T112, T113 → parallel (different toolchains)
```

---

## Decision register — all questions resolved

**Zero blocked tasks.** All seven open questions are answered: OQ-3 on
2026-08-16, and OQ-1, OQ-2, OQ-4, OQ-5, OQ-6, OQ-7 on 2026-08-17, all by Brian.
Every former "DEFAULT IF FORCED" has been promoted to specified behavior. The
seven decision tasks are retained and marked `[x] RESOLVED` so the trail
survives.

| Was | Decision | Recorded | Implemented by |
|---|---|---|---|
| **OQ-1** confidence upgrade | Flat `min(1.0, max(image, provider) + 0.10)`, once per field, **no stacking** | RD-2 | T004 ✅, T061 |
| **OQ-2** acceptance default | **Confidence-driven, not source-driven**: accepted at **≥ 0.70**, single named constant | RD-3 | T005 ✅, T090, T120 |
| **OQ-3** wishlist mechanism | **(a) direct wishlist coin** — no migration, all 14 fields | RD-1 | T003 ✅, T072, T073, T119 |
| **OQ-4** reverse type/legend | **Excluded from queries**; used to **rank already-returned candidates**; new **FR-039** | RD-4 | T006 ✅, T035, T121, T122, T123 |
| **OQ-5** rollout | **Straight cutover**; `DeepIdentificationEnabled` is the kill switch | RD-5 | T007 ✅ |
| **OQ-6** hypothesis panel | **Build it** — collapsible, default collapsed | RD-6 | T008 ✅, T091 |
| **OQ-7** OCRE routing | **Inclusion by default**; skip only on *positive* non-Roman signal, with a stated reason | RD-7 | T009 ✅, T046 |

Three answers **changed** the previously stated default and are the ones to
watch during implementation: **OQ-2** (reversed — confidence, not source),
**OQ-4** (added a ranking role, new FR-039), and **OQ-6** (reversed — build the
panel).

---

## Beta merge plan — phase independence review

Brian's workflow change: **each phase is merged into `beta` for him to test and
validate as it completes.** Phases are a *planning* unit, not automatically a
*shipping* unit. I reviewed every boundary for whether merging it alone leaves
`beta` in a working, testable state.

### ⚠️ Phases that MUST NOT be merged alone

| Phase | Why merging alone is unsafe |
|---|---|
| **3** structured output | Adds `get_structured_model` with **no consumer**. Harmless but invisible — nothing for Brian to test. |
| **4** 🔑 keystone | Its own checkpoint already says *"do not release at this checkpoint"*. The hypothesis has **zero consumers**, so beta shows the **identical junk output** as today while the `DeepSynthesis` payload quietly gains `image_hypothesis`. Brian would test it and correctly conclude nothing changed. |
| **5** query terms | Real improvement, but it lands the placeholder deletion and `insufficient_query_evidence` while synthesis is **still provider-gated** — a coin with no provider match still yields an empty draft. Half the fix. |
| **6** router | Deterministic routing is user-invisible on its own. Safe, but not independently *validatable*. |
| **7** evaluator | **Actively confusing alone**: image-vs-provider disagreements start appearing in the report while the narrative still never mentions the images. The UI would show "the image says X" next to prose that has never heard of the image. |
| **8** synthesis | The payoff — but its correctness is only proven by Phase 9's gate. Merging 8 without 9 ships the rewrite with no regression proof. |
| **12** web *(as written)* | **Not independently mergeable — this phase must be split.** T089/T090/T091 render data that does not exist until Phase 8. The hypothesis panel would render nothing and image-evidence marks would never appear. See the split below. |
| **14** dead code / F5 refactor | Explicitly sequenced last so the refactor moves *settled* code, and it names **Phase 9's regression suite as its safety net**. Merging it before Phase 9 refactors code that is about to be rewritten, without the net. |
| **15** seam tests | The F4 round-trip test and T107 drift test assert the **post-351** wire shape. Landing them before Phase 8/13 puts failing tests on `beta`. |

### Required split of Phase 12

Phase 12's current header claims "parallel-safe with Phases 5–11 except T076".
That is **wrong for T089–T091**, which consume Phase 8 output. Split it:

- **Phase 12a — independent** (T085, T086, T087, T088): B6 connection badge, F6
  `useDeepAnalysisLauncher` extraction, `disabled` prop wiring. Zero dependency
  on the vision work. Safe to merge alone and genuinely testable.
- **Phase 12b — depends on Phase 8** (T089, T090, T091, T092, T120): image-vs-cited
  marking, confidence-driven acceptance, the hypothesis panel. Must ship with the
  vision group.

### Proposed merge groupings for `beta`

| # | Contents | Independently shippable? | What Brian can actually validate |
|---|---|---|---|
| **MG-1** 🔥 | Phase 2 (T011–T018) | **Yes** | Re-run the Maximinus coin: the NGC cert appears, providers receive real query terms, and a quick-lookup failure is now visible instead of silent. **This alone fixes the observed symptom** and is worth shipping first. |
| **MG-2** | Phase 11 (T077–T084) + Phase 12a (T085–T088) | **Yes**, parallel with MG-1 | Memory no longer leaks per job; job tokens are separately signed and revoked on settle; the connection badge stops lying; the launch button disables at the limit instead of erroring after a click. |
| **MG-3** 🔑 | Phases 3 + 4 + 5 + 6 + 7 + 8 + 9 + 12b | **Only as a unit** | **The feature.** A coin with no quick evidence and no notes produces a real narrative, ≥4 draft fields marked image-derived, a collapsible "what the images alone said" panel, and provider-vs-image disagreements. Gated on T068 passing. |
| **MG-4** | Phase 10 (T072, T073, T074, T075, T076, T119) | Yes, **after MG-3** | Save a deep result as a wishlist item or a collection item. Technically safe earlier, but before MG-3 there are no image-derived fields to apply, so testing it early proves little. |
| **MG-5** | Phase 13 (minus T098) + Phase 15 | **After MG-3** | Contract drift corrected, 344 record fixed, one canonical version string, Go↔Python seam covered. Must follow MG-3 so the tests assert the shipped shape. |
| **MG-6** | Phase 14 (T100–T104) | **After MG-3** | No behavior change by construction; Phase 9's suite is the safety net. Pure cleanup — nothing for Brian to test beyond "still works". |
| **final** | T098 (ADR → Accepted) + Phase 16 full gate | On merge to `main` | — |

Phase 1 is document-only and carries no merge risk; it lands with MG-1.

**The one rule that matters**: MG-3 is indivisible. Merging any single phase of
3–9 into `beta` would leave the vision rewrite half-landed and would waste
Brian's testing time — he would be validating a pipeline that is provably in an
intermediate state. Every other group is genuinely independent.

---
## Summary

- **123 tasks** across **16 phases** (Phase 12 splits into 12a/12b for merging).
- **Ship-first**: Phase 2 (T011–T018) is independent and alone fixes the
  observed failure.
- **Critical path**: structured output → typed hypothesis → three parallel
  consumers → synthesis → the Maximinus regression gate.
- **Zero blocked tasks.** All seven open questions resolved (OQ-3 on 2026-08-16;
  the remaining six on 2026-08-17). Every default is now specified behavior.
- **Beta merge plan**: six merge groups. **MG-3 (Phases 3–9 + 12b) is
  indivisible**; everything else ships independently.
- **Coverage**: every FR-001..FR-039 and every B1–B6 / F1–F9 / NE-1 / NE-2 maps
  to at least one task; no orphan tasks.
- **The gate**: T068. Nothing is done until a coin with no quick evidence and no
  notes produces a non-empty narrative and non-empty `proposed_fields`.
