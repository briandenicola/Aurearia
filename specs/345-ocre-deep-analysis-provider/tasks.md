---

description: "Task list for Feature 345: OCRE Automated Deep Analysis Provider"
---

# Tasks: OCRE Automated Deep Analysis Provider

**Input**: Design documents from `/specs/345-ocre-deep-analysis-provider/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/ocre-provider.md, quickstart.md, docs/adr/0010-ocre-odbl-provider.md

**Scope guard (confirmed against beta `src/`)**: OCRE provider delivery only —
fixed **GET** Nomisma SPARQL client/query/cache/scoring, internal tool + DI +
job-token/budget wiring, pipeline catalog flag, Python provider node/router
fan-out move, typed evidence/citation/attribution, UI report/proposal/admin
health/toggle, docs, tests, one manual live smoke, QC + beta release prep.
**No RPC work. No OCRE images. No corpus/migration.** All persistence reuses
existing `DeepIdentificationProviderRun`/proposal structures — no schema
migration in this feature.

**Tests**: Included per Testing & CI Constraints (spec.md) — CI MUST use
offline fixtures/httptest/fake tools only; exactly **one** manual, CI-excluded
live-smoke task exists (T057).

**Organization**: Tasks are grouped by user story (spec.md priorities). Phase 2
(Go SPARQL boundary) is foundational and blocks every user story because the
internal tool, provider node, and UI all depend on it.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependency on an
  incomplete task in this list)
- **[Story]**: US1–US5 map to spec.md priorities P1/P1/P1/P2/P2

## Confirmed exact paths (beta, verified by exploration before this file was written)

| Concern | Path | Status |
|---|---|---|
| Settings keys/struct | `src/api/services/settings_service.go` | `SettingDeepIdentificationOCREEnabled` exists (L94/204/301); `...OCRECallBudget` does **not** exist yet |
| Provider catalog | `src/api/services/deep_identification_pipeline_runner.go` (`deepPipelineProviderCatalog`, L317-333) | OCRE entry hardcoded `Automatable:false` |
| Citation allowlist (Go) | same file, `deepCitationHostAllowlist` (L~415) | `"ocre": {"numismatics.org": true}` already present, unchanged |
| Citation allowlist (Python) | `src/agent/app/teams/deep_identification/merge.py` (`CITATION_HOST_ALLOWLIST`, `PROVIDER_RANK`) | `"ocre": {"numismatics.org"}` already present, unchanged |
| Coin-field write allowlist | `src/api/services/deep_identification_proposal.go` (`deepProposalCoinFieldAllowlist`, L36) | no `coin_type` key yet — must be added (reuse an existing `models.Coin` column, no migration) |
| Internal tool handler | `src/api/handlers/internal_tools.go` (`DeepProviderToolsHandler`, `NomismaSearch` L442-490 is the pattern to mirror) | no `OCRESearch` yet |
| Route registration | `src/api/main.go` (`internalDeepProviderTools` group, L845-853) | `nomisma_search`/`numista_*` registered; `ocre_search` missing |
| Nomisma client/cache to mirror | `src/api/services/nomisma_client.go`, `nomisma_cache.go` | pattern source only, not edited |
| Python stub to rewrite | `src/agent/app/teams/deep_identification/providers/ocre.py` | currently a trivial `not_automated`-only stub |
| Python tool wrapper | `src/agent/app/tools/provider_tools.py` (`nomisma_search` L71-73 is the pattern) | no `ocre_search` yet |
| Graph fan-out wiring | `src/agent/app/teams/deep_identification/graph.py` (`_AUTOMATED_PROVIDER_NODES` L52, `_TRIVIAL_PROVIDER_NODES` L53) | `"ocre"` currently in the trivial dict |
| Router | `src/agent/app/teams/deep_identification/router.py` | **unchanged** — OCRE flows through existing automatable LLM selection once it appears in `catalog` with `automatable=true` |
| Synthesis | `src/agent/app/teams/deep_identification/synthesis.py`, `src/agent/app/models/responses.py` (`DeepSynthesis`) | `attributions` is defined in the F344 OpenAPI contract + `data-model.md` + web `DeepReportAttribution` type but is **never populated or rendered anywhere today** — this feature is the first to wire it (generic assembly, not OCRE-only) |
| Report UI | `src/web/src/components/deep-identification/DeepReportPanel.vue` | renders `coverage`/`disagreements`/`unresolvedQuestions`; does **not** render `attributions` today |
| Proposal UI | `src/web/src/components/deep-identification/DeepProposalEditor.vue` | no attribution rendering today |
| Attribution component pattern | `src/web/src/components/mint/NomismaAttribution.vue` | pattern to mirror for `OCREAttribution.vue`, different component/copy/license link |
| Admin settings UI | `src/web/src/composables/useAdminConfig.ts`, `src/web/src/components/admin/AdminSystemSection.vue` (Numista block + health section, L24-370) | **no** Deep-Identification/OCRE settings exposed in admin UI at all yet — new work, not an edit of an existing OCRE toggle |
| Admin health pattern | `admin.GET("/numista/health", ...)` (`main.go` L692) | pattern to mirror for a new OCRE health read |
| ADR | `docs/adr/0010-ocre-odbl-provider.md` | already drafted, `Status: Proposed` — needs acceptance at merge |

---

## Phase 1: Setup

- [ ] T001 [P] Confirm and finalize ADR acceptance: review `docs/adr/0010-ocre-odbl-provider.md` against the implemented design once Phases 2–7 land, flip `Status: Proposed` → `Status: Accepted`, and add/verify its entry in `docs/adr/README.md`'s ADR index
- [X] T002 [P] Add the new bounded budget setting in `src/api/services/settings_service.go`: constant `SettingDeepIdentificationOCRECallBudget = "DeepIdentificationOCRECallBudget"`, default `"3"` in `settingDefaults`, field `OCRECallBudget int` on `DeepIdentificationSettings`, and `readInt(SettingDeepIdentificationOCRECallBudget, 3, 1, 20)` in `GetDeepIdentificationSettings()` (alongside the existing `OCREEnabled`/`NumistaCallBudget` lines)
- [X] T003 [P] Add `coin_type` → an existing reused `models.Coin` field (no schema change) to `deepProposalCoinFieldAllowlist` in `src/api/services/deep_identification_proposal.go`, with a comment explaining it carries the OCRE RIC-style type label (citation lives in claim evidence, not the Coin row)

**Checkpoint**: settings/budget/allowlist scaffolding exists; nothing yet calls Nomisma.

---

## Phase 2: Foundational — Go SPARQL Boundary (Plan Phase A; BLOCKS all user stories)

**⚠️ CRITICAL**: No user story work may begin until this phase is complete and
`go build ./... && go test ./services/...` (from `src/api/`) is green. No
routing/wiring happens in this phase — these files are new, unreferenced by
any handler yet.

- [X] T004 [P] Create `src/api/services/ocre_query.go`: `OCREQueryParams` struct (RulerSlug/DenominationSlug/MintSlug/MaterialSlug/LegendTokens/OCREIDSlug/Limit per data-model.md §2), slug regex `^[a-z0-9]([a-z0-9_.-]*[a-z0-9])?$` (+ OCRE-id shape `^ric\.[0-9a-z_.()]+$`) that **drops** (never interpolates) failing values, Template E / Template K SPARQL builders (contract §3) with the constant skeleton and only validated slugs interpolated inside `<...>` URI brackets, and a SPARQL 1.1 JSON `results.bindings[]` parser
- [X] T005 [P] Create `src/api/services/ocre_query_test.go`: SC-010 structural-invariance fixtures — adversarial ruler/mint/denomination/legend inputs (quotes, backslashes, angle brackets, newlines, `FILTER`/`VALUES`/`regex` payloads) produce a byte-identical query skeleton; invalid slugs are dropped not interpolated; legend/inscription text never appears in the emitted query string; Template E vs Template K selection based on `OCREIDSlug` presence
- [X] T006 [P] Create `src/api/services/ocre_client.go`: `OCREErrorKind` consts (`unavailable`, `no_match`, `invalid_response`, `invalid_request`, `cancelled`), `OCRECandidate` struct (`type_uri`/`label`/`matched_fields`/`confidence`/`explanation`), `OCREClient` interface with `Search(ctx, params OCREQueryParams, limit int) ([]OCRECandidate, OCREErrorKind, error)`, and `HTTPOCREClient` implementing GET to `https://nomisma.org/query?query=<url-encoded>` with fixed non-default `User-Agent`, `Accept: application/sparql-results+json`, 8s timeout, 1 MiB `io.LimitReader` response cap, context-cancellation → `OCRECancelled`, non-200 → `OCREErrorUnavailable`, malformed/oversize → `OCREErrorInvalidResponse`, every candidate's `type_uri` host re-validated `== numismatics.org` before return; add `NewHTTPOCREClientForTest(baseURL)` mirroring `NewHTTPNomismaClientForTest`
- [X] T007 [P] Create `src/api/services/ocre_client_test.go`: `httptest`-backed fake-Nomisma-server tests for 200 parse (using `NewHTTPOCREClientForTest`), empty bindings, HTTP 500, malformed JSON, oversize body (>1 MiB), slow response → timeout, context-cancel → cancelled, non-`numismatics.org` host in a binding is rejected/dropped — never reaches the real `nomisma.org` host
- [X] T008 [P] Create `src/api/services/ocre_cache.go`: bounded in-memory TTL cache mirroring `nomisma_cache.go` (`OCRESearchStatus` ok/no_match/unavailable, `OCRECache` struct, `NewOCRECache()`), cache key = `SHA-256(join("\x1f", RulerSlug, DenominationSlug, MintSlug, MaterialSlug, sort(LegendTokens), OCREIDSlug, Limit, flagGeneration))` per data-model.md §2; `no_match` cached, transient failures (`unavailable`/`invalid_response`/`timeout`/`cancelled`) never cached
- [X] T009 [P] Create `src/api/services/ocre_cache_test.go`: identical bound-parameter sets hit the cache within TTL; any single differing bound parameter yields a distinct cache key (no stale cross-input reuse); a `flagGeneration` change (simulating an enable/disable toggle) never reuses a stale entry; negative (`no_match`) results are cached; `unavailable`/timeout results are never cached
- [X] T010 [P] Create `src/api/services/ocre_scoring.go`: pure `Score(params OCREQueryParams, rows []parsedRow) []OCRECandidate` — de-dup by `TypeURI` before ranking, weighted match score (authority > denomination > mint > material, bounded legend-token-in-label bonus), confidence clamp `[0,1]`, deterministic order `(-Confidence, -len(MatchedFields), TypeURI asc)`, cap on distinct types after de-dup, bounded `Explanation` (≤500 chars)
- [X] T011 [P] Create `src/api/services/ocre_scoring_test.go`: determinism across ≥2 identical-input runs (SC-005), tie-break by canonical `TypeURI`, de-dup before ranking, cap enforcement on distinct types, preserved ambiguity (multiple plausible candidates surfaced, not collapsed), no-match/empty-input case

**Checkpoint**: `go vet ./... && go build ./... && go test ./services/... -run OCRE` all green in `src/api/`. No handler, route, catalog, or Python code references these files yet — build stays green with zero behavior change.

---

## Phase 3: User Story 1 - OCRE contributes Roman Imperial coin-type candidates (Priority: P1) 🎯 MVP

**Goal**: With the flag enabled, Deep Analysis on a Roman Imperial coin selects
OCRE, queries bound Nomisma SPARQL through the Go internal-tool boundary, and
surfaces one or more bounded, ranked, deterministic candidate type claims with
canonical `numismatics.org/ocre/id/...` citations.

**Independent Test** (spec.md US1): Enable the flag; run Deep Analysis on
evidence resolving ruler=Hadrian + denomination=denarius + mint=Rome; verify
OCRE is selected, candidates carry canonical citations/labels/matched
fields/confidence/explanations, ranking is deterministic, and a non-Roman
coin does not select OCRE absent an override.

### Go internal tool + wiring for User Story 1

- [X] T012 [US1] In `src/api/handlers/internal_tools.go`: add `OCRESearchRequest`/`OCRESearchResponse` DTOs (contract §1 wire shape: ruler/denomination/mint/material/legend_tokens/ocre_id/limit → status/candidates/attribution), add `ocreClient services.OCREClient` + `ocreCache *services.OCRECache` fields to `DeepProviderToolsHandler` (constructor param additions mirroring `nomismaClient`), and add `OCRESearch(c *gin.Context)` handler: job-token-derived `jobID`, `deepProviderBudgets.TryConsume(jobID, "ocre", settings.OCRECallBudget)` → `quota_limited` on exhaustion, calls `ocreClient.Search` (cache-checked first), maps `OCREErrorKind` → `status` per data-model.md §4 table, re-validates every candidate's citation host, **never returns 4xx/5xx for an upstream problem**, Swagger annotations mirroring `NomismaSearch` (`@Router /internal/tools/ocre_search [post]`)
- [X] T013 [US1] In `src/api/handlers/internal_tools_test.go`: handler tests for `ok` (≥1 candidate), `empty`, `invalid_response` (malformed/oversize), `unavailable`, `timeout`, `cancelled`, `quota_limited` (budget exhausted → independent from numista/nomisma budgets, mirroring `TestDeepProviderTools_NomismaSearchIndependentBudgetFromNumista`), missing job token → 401, unparseable body → 400, and **never** a 5xx for an upstream/transport problem
- [X] T014 [US1] In `src/api/main.go`: construct `services.NewHTTPOCREClient()` and `services.NewOCRECache()` near the existing `deepNomismaClient` construction (~L113-115), pass them into `handlers.NewDeepProviderToolsHandler(...)`, and register `internalDeepProviderTools.POST("/ocre_search", deepProviderToolsHandler.OCRESearch)` immediately after the existing `nomisma_search` registration (~L853)
- [X] T015 [US1] In `src/api/services/deep_identification_pipeline_runner.go`: change the `deepPipelineProviderCatalog` OCRE entry (L331) from unconditional `{Provider: "ocre", Automatable: false, Reason: "pending_license_validation"}` to conditional on `settings.OCREEnabled` — `{Provider: "ocre", Automatable: true, CallBudget: settings.OCRECallBudget}` when enabled, else the existing not-automated entry; update the function's doc comment (it currently states OCRE/RPC are "always typed not_automated ... regardless of the OCREEnabled ... setting")
- [X] T016 [P] [US1] Add/extend a Go test covering `deepPipelineProviderCatalog` OCRE conditional behavior: flag on → `Automatable:true` + `CallBudget=settings.OCRECallBudget`; flag off → unconditional not-automated entry unchanged; verify RPC's catalog entry is untouched (co-located with `deep_identification_pipeline_runner.go`'s existing test file)
- [X] T017 [P] [US1] Add a Go test in `src/api/services/deep_identification_proposal_test.go` (or nearest existing proposal test file) asserting `coin_type` is present in `deepProposalCoinFieldAllowlist`, maps to the intended `models.Coin` field, and a proposed `coin_type` field survives `buildDeepProposalDocumentJSON` end to end

### Python provider node + fan-out for User Story 1

- [X] T018 [P] [US1] In `src/agent/app/tools/provider_tools.py`: add `async def ocre_search(self, *, ruler: str = "", denomination: str = "", mint: str = "", material: str = "", legend_tokens: list[str] | None = None, ocre_id: str = "", limit: int = 5) -> dict` — thin authenticated POST to `/api/internal/tools/ocre_search` mirroring `nomisma_search`, raising `ProviderToolError` on transport failure (never propagates further)
- [X] T019 [US1] Rewrite `src/agent/app/teams/deep_identification/providers/ocre.py`: new signature `async def run(catalog_entry: DeepProviderCatalogEntry, tools: ProviderToolsClient, quick_evidence: QuickEvidence | None, notes: str) -> ProviderEvidence` — (1) if `not catalog_entry.automatable` → return trivial `not_automated` row with **zero** calls (flag-off short circuit, no `tools.ocre_search` invocation); (2) decode ruler/denomination/mint/material/legend tokens/optional OCRE id from `quick_evidence`; (3) if no Roman-Imperial signal decodes → return `no_match`/`skipped` **without** a tool call; (4) else call `tools.ocre_search(...)`, map `{status, candidates, attribution}` → `ProviderEvidence` with `claims=[ProviderClaim(field="coin_type", value=label, confidence=..., citation=type_uri, excerpt=explanation)]` for each candidate, run through `merge.validate_citations("ocre", ...)`; (5) `ProviderToolError` → typed `failed`/`timed_out` row, never raises
- [X] T020 [US1] In `src/agent/app/teams/deep_identification/graph.py`: move `"ocre"` from `_TRIVIAL_PROVIDER_NODES` (L53) into `_AUTOMATED_PROVIDER_NODES` (L52) — i.e. `_AUTOMATED_PROVIDER_NODES = {"numista": numista_provider.run, "nomisma": nomisma_provider.run, "ocre": ocre_provider.run}`; `_TRIVIAL_PROVIDER_NODES = {"rpc": rpc_provider.run}`; verify `_run_one_provider`'s existing `asyncio.wait_for(..., timeout=bounds.provider_timeout_s)` / exception handling wraps the OCRE call identically to numista/nomisma with no code change needed there
- [X] T021 [P] [US1] Create `src/agent/tests/test_deep_identification_ocre.py`: node tests for `contributed` (multi-candidate, matched fields/confidence/explanation preserved), `no_match` (no candidates / no decodable Roman-Imperial signal, asserting **zero** `tools.ocre_search` calls), `failed`/`timed_out` (`ProviderToolError`, `status=unavailable`/`timeout`), `invalid_response` mapping to `failed`+`error_kind="invalid_response"`, flag-off `not_automated` with **zero** calls even when evidence is clearly Roman Imperial, known-OCRE-id confirm (Template K) resolving/confirming a type, and unknown/unresolving OCRE id reported as unresolved (not fabricated)
- [X] T022 [P] [US1] Extend `src/agent/tests/test_deep_identification_router.py`: OCRE included in the automatable catalog list passed to the router when `automatable=true`; router selects it for Roman-Imperial-relevant evidence; a non-Roman-Imperial-evidence run does not force selection absent override; OCRE never appears in the router's `selected`/`skipped` output when `automatable=false` (trivial path)
- [X] T023 [P] [US1] Extend `src/agent/tests/test_deep_identification_fanout.py`: OCRE now runs through the automated (timeout-wrapped, semaphore-bounded) fan-out path rather than the trivial dict when selected/non-automatable-mixed; confirm RPC's trivial-path behavior is unaffected by the dict split

**Checkpoint**: User Story 1 fully functional and independently testable —
`go test ./services/... ./handlers/...` and `pytest tests/test_deep_identification_ocre.py tests/test_deep_identification_router.py tests/test_deep_identification_fanout.py` all green; flag-off leaves current beta behavior byte-for-byte unchanged.

---

## Phase 4: User Story 2 - Transparent OCRE attribution, license, and evidence (Priority: P1)

**Goal**: Every surface that shows OCRE evidence (report, proposal, provider
status, export) renders the exact ODbL 1.0 / ANS attribution string with
working links to the canonical type and the ODbL 1.0 license, visually and
textually distinct from Nomisma/Numista attribution; absent when no OCRE
evidence exists.

**Independent Test** (spec.md US2): Produce a result with ≥1 OCRE claim;
inspect report/proposal surfaces; verify the exact attribution string +
links appear, distinct from other providers' attribution, and absent when no
OCRE evidence is present.

> **Note**: `attributions`/`DeepReportAttribution` already exists in the F344
> OpenAPI contract, `data-model.md`, and the web `DeepReport` type
> (`src/web/src/types/index.ts` L1948/1960) but is populated and rendered
> **nowhere** in beta today. This phase wires that generic mechanism for the
> first time, using OCRE as the first attribution-bearing provider it serves.

- [X] T024 [P] [US2] In `src/agent/app/models/responses.py`: add a `ProviderAttribution` model (`provider: ProviderName`, `text: Annotated[str, StringConstraints(max_length=200)]`, `identifier: str | None = None`) and `attributions: list[ProviderAttribution] = Field(default_factory=list, max_length=10)` field on `DeepSynthesis`, matching the existing `data-model.md` §"attributions" example shape exactly (`{"provider": "...", "text": "...", "identifier": "..."}`)
- [X] T025 [P] [US2] In `src/agent/app/teams/deep_identification/synthesis.py`: add `_build_attributions(evidence: list[ProviderEvidence]) -> list[ProviderAttribution]` — one entry per provider with a non-empty `.attribution` **and** ≥1 surfaced claim (mirrors `_build_coverage`'s determinism, no LLM involvement), wire it into the `DeepSynthesis(...)` construction alongside `coverage=coverage`
- [X] T026 [P] [US2] Extend `src/agent/tests/test_deep_identification_graph_topology.py` (where `synthesize()`/`DeepSynthesis` is already exercised): attribution present iff the provider contributed ≥1 claim; OCRE's attribution text is the exact ODbL/ANS string; multiple providers (OCRE + Nomisma) produce distinct, non-merged attribution entries; no attribution entry when a provider is `no_match`/`failed`/`not_automated`
- [X] T027 [P] [US2] Create `src/web/src/components/deep-identification/OCREAttribution.vue`: mirrors `src/web/src/components/mint/NomismaAttribution.vue`'s structure but renders the fixed text `"Coin type data: Online Coins of the Roman Empire (OCRE), American Numismatic Society — ODbL 1.0."` with a `SafeExternalLink` to the canonical OCRE type (`identifier`/citation prop) and a second `SafeExternalLink` to `https://opendatacommons.org/licenses/odbl/1-0/`, with its own distinct CSS class (not reusing `.nomisma-attribution`)
- [X] T028 [P] [US2] Create `src/web/src/components/deep-identification/__tests__/OCREAttribution.test.ts`: renders only when a canonical type link is supplied; exact attribution string; both links present and correctly targeted; distinct DOM class/markup from `NomismaAttribution`
- [X] T029 [US2] Edit `src/web/src/components/deep-identification/DeepReportPanel.vue`: render `report.attributions` (new `<section>` after "Provider coverage"), using `<OCREAttribution>` when `entry.provider === 'ocre'` (passing `entry.identifier` as the canonical type link) and a generic text/license rendering for any other provider entry; nothing renders when `attributions` is empty/absent
- [X] T030 [US2] Extend `src/web/src/components/deep-identification/__tests__/DeepReportPanel.test.ts`: OCRE attribution renders when `report.attributions` includes an `ocre` entry; is visually/textually distinct from a co-present Nomisma attribution entry; section absent when `attributions` is empty
- [X] T031 [US2] Edit `src/web/src/components/deep-identification/DeepProposalEditor.vue`: surface the OCRE attribution (and any other present-provider attribution) alongside the `coin_type` proposed field's evidence in the draft proposal view, reusing `OCREAttribution.vue`
- [X] T032 [US2] Extend the existing `DeepProposalEditor` test file (`src/web/src/components/deep-identification/__tests__/` — locate/extend the current proposal editor spec) to assert OCRE attribution renders in the draft proposal when a `coin_type` field's evidence references the `ocre` provider

**Checkpoint**: User Stories 1 AND 2 both independently functional — a
completed OCRE-contributing run shows correct, distinct attribution on both
the report and the proposal, and shows none when OCRE didn't contribute.

---

## Phase 5: User Story 3 - OCRE failures and flag-off never break Deep Analysis (Priority: P1)

**Goal**: Timeouts, HTTP 500s, malformed bindings, no-match, and flag-off all
degrade to typed partial-provider outcomes; the overall job always still
reaches `completed`/`partial`; flag-off makes zero calls.

**Independent Test** (spec.md US3): (a) flag on, force timeout/500/malformed →
job still completes with OCRE `timed_out`/`failed`/`no_match`. (b) flag off →
OCRE `not_automated`, zero calls, job completes normally.

> Most of the isolation mechanism (per-provider `asyncio.wait_for` +
> broad `except Exception` in `graph.py`'s `_run_one_provider`, and the
> handler's never-5xx contract) is **already generic and reused as-is** —
> this phase is regression/coverage verification specific to OCRE's new
> code paths, not new isolation machinery.

- [ ] T033 [P] [US3] Extend `src/api/services/ocre_client_test.go` (from T007) with explicit assertions that a raw transport/parse error is **never** surfaced to the caller — only the typed `OCREErrorKind` — for timeout, HTTP 500, and malformed-JSON cases
- [ ] T034 [P] [US3] Extend `src/api/handlers/internal_tools_test.go` (from T013): budget-exhausted OCRE call returns `{"status":"quota_limited","candidates":[],...}` with HTTP 200 (never 5xx); malformed upstream body surfaces as `invalid_response` with HTTP 200
- [ ] T035 [P] [US3] Extend `src/agent/tests/test_deep_identification_ocre.py` (from T021): a job with OCRE `timed_out`/`failed` alongside a contributing Numista/Nomisma still reaches synthesis with the remaining evidence (partial-provider isolation), and OCRE's failure never raises out of `_run_one_provider`
- [ ] T036 [P] [US3] Extend `src/agent/tests/test_deep_identification_fanout.py` (from T023): total-timeout-mid-fanout scenario — OCRE hung while other providers already completed still yields partial synthesis using already-completed evidence (existing `on_result` incremental-accumulation behavior), and OCRE flag-off contributes zero entries to `names_to_run`'s automated path (goes straight to `not_automated` with no tool call, verified via a call-count assertion on a spy `ProviderToolsClient`)
- [ ] T037 [US3] Add a telemetry-privacy test alongside `deep_identification_pipeline_runner.go`'s existing tests for `deepProviderResultPublicPayloadJSON`: an OCRE `provider_result` public payload never includes user notes or full legend/inscription text — only status/timing/counts/error-kind/link-out, matching FR-022

**Checkpoint**: All P1 user stories (US1, US2, US3) independently functional; zero jobs fail because of OCRE (SC-003); flag-off is verified zero-call (SC-004).

---

## Phase 6: User Story 4 - Admin enablement and provider health visibility (Priority: P2)

**Goal**: An admin can toggle `SettingDeepIdentificationOCREEnabled` (and the
new call-budget setting) and see OCRE's enablement + most-recent outcome
class, without exposing user content. This is **new frontend + a new Go
health read** — beta currently exposes no Deep-Identification/OCRE settings
in the admin UI at all.

**Independent Test** (spec.md US4): Toggle the flag; confirm the catalog
reflects the change on the next job; confirm a health/status view shows
current enablement + last outcome class; confirm non-admins are refused.

- [ ] T038 [P] [US4] In `src/api/repository/deep_identification_repository.go`: add `GetLatestProviderStatus(provider models.DeepProviderName) (*models.DeepIdentificationProviderRun, error)` (or similar bounded, non-user-scoped read) returning the most recent row for `Provider = "ocre"` ordered by `created_at desc` — status/timing/counts only, no per-job user-content join
- [ ] T039 [P] [US4] Add a Go admin handler + route (new method on an existing admin-facing handler, e.g. alongside `NumistaSearch`/`Health` patterns) exposing `GET /api/admin/deep-identification/ocre/health` → `{enabled: bool, callBudget: int, gateValidated: bool, lastOutcome: string|null, lastCheckedAt: string|null}`, admin-auth-gated identically to the existing `admin.GET("/numista/health", ...)` registration in `src/api/main.go` (~L692)
- [ ] T040 [P] [US4] Add Go tests for T038/T039: repository read returns the most recent OCRE row only, no user/coin content leaked; handler enforces admin auth (non-admin → refused, consistent with existing settings-authorization tests); reflects flag on/off correctly
- [ ] T041 [P] [US4] In `src/web/src/types/index.ts`: extend the `AppSettings` type with `DeepIdentificationOCREEnabled: string` and `DeepIdentificationOCRECallBudget: string` (string-typed to match the existing Numista setting fields' convention)
- [ ] T042 [P] [US4] In `src/web/src/composables/useAdminConfig.ts`: add `DeepIdentificationOCREEnabled: 'false'` and `DeepIdentificationOCRECallBudget: '3'` to both settings-ref default objects (mirroring the existing `Numista*` defaults at L29-34/71-76), and thread them through the existing load/save calls
- [ ] T043 [US4] In `src/web/src/components/admin/AdminSystemSection.vue`: add a new "OCRE / Deep Analysis" fieldset (mirroring the existing Numista block, L24-65) with an enable toggle bound to `DeepIdentificationOCREEnabled` and a bounded numeric call-budget input (1–20, default 3) bound to `DeepIdentificationOCRECallBudget`
- [ ] T044 [US4] In the same file: add an "OCRE Health" section (mirroring the existing "Numista Health" section, L67-147) that calls the new admin health endpoint (add `getAdminOCREHealth()` to `src/web/src/api/client.ts`) and renders enablement + gate-validated state + last outcome class — no user notes/legend content
- [ ] T045 [P] [US4] Extend `src/web/src/components/admin/__tests__/AdminSystemSection.numista.test.ts`-style coverage with a new `AdminSystemSection.ocre.test.ts` (or extend an existing spec file): toggle change persists; call-budget bounds validated; health section renders enablement + last-outcome without user content; a simulated non-admin session cannot change the setting (consistent with existing settings-authorization behavior)

**Checkpoint**: Admin can safely enable/disable OCRE and observe its health without any new user-content exposure; catalog change takes effect on the next job only (in-flight jobs keep their decided catalog, already guaranteed by existing per-job catalog snapshotting).

---

## Phase 7: User Story 5 - Explicit provider override for OCRE (Priority: P2)

**Goal**: An explicit provider override can force OCRE to run (even if the
router would skip it) or exclude it; a disabled flag always wins over any
override.

**Independent Test** (spec.md US5): Flag on + override including OCRE → OCRE
runs regardless of router reasoning. Flag on + override excluding OCRE → OCRE
does not run. Flag off + override naming OCRE → OCRE stays `not_automated`,
zero calls.

> No production code change is expected here: `router.py`'s existing
> `provider_override` handling (already generic — "only catalog-listed
> automatable providers named in the override are selected") applies to OCRE
> automatically once it is a normal automatable catalog entry (Phase 3). This
> phase is regression-verification only.

- [ ] T046 [P] [US5] Extend `src/agent/tests/test_deep_identification_router.py`: `provider_override=["ocre"]` selects OCRE even when the LLM-simulated router would not have chosen it (bypassing the LLM call entirely, per existing override semantics); `provider_override` omitting `"ocre"` from an otherwise-automatable catalog prevents it from running
- [ ] T047 [P] [US5] Extend `src/agent/tests/test_deep_identification_ocre.py` or `test_deep_identification_fanout.py`: flag disabled (`automatable=false`) + `provider_override=["ocre"]` still yields `not_automated` with **zero** `tools.ocre_search` calls — flag takes precedence over override (FR-004)

**Checkpoint**: All five user stories independently functional and tested.

---

## Phase 8: Polish, QC & Beta Release Prep

**Purpose**: Cross-cutting documentation, full-gate verification, CI-safety
guard, and the single manual live-network check — none of which touches
production code paths already covered above.

- [ ] T048 [P] Verify `docs/adr/README.md` lists ADR 0010 in its index (add the entry if the acceptance step in T001 didn't already do so)
- [ ] T049 [P] Update any Feature-344-era developer/user docs that describe OCRE as `not_automated`/deferred (search `docs/` for G-OCRE/T155 references) to reflect that OCRE is now a first-class, default-off, admin-gated automated provider
- [ ] T050 Run `specs/345-ocre-deep-analysis-provider/quickstart.md` §2 (happy path) and §3 (robustness) end-to-end against a local dev stack with the flag toggled both ways, confirming the exact attribution string, deterministic re-run ordering (SC-005), and flag-off zero-call behavior (SC-004)
- [ ] T051 Run the full §17/§21 gate commands from `quickstart.md` §6 and record pass/fail: `go vet ./... && go build ./... && go test ./...` (`src/api/`), `ruff check app/ tests/ && pytest tests/ -v` (`src/agent/`), `npm run build` + the vitest suite (`src/web/`)
- [ ] T052 [P] Regenerate Swagger/OpenAPI docs for the new `OCRESearch` internal handler (existing swag generation script/task) and review the diff for accuracy against contract §1
- [ ] T053 [P] CI-safety guard: grep the full `src/api`, `src/agent`, and `src/web` test suites for any literal `nomisma.org`/live-network reference outside the one manual smoke script (T054) and confirm every OCRE test in Phases 2–7 uses `NewHTTPOCREClientForTest`/httptest/fake tool clients/fixtures only — zero live network calls in CI (Testing & CI Constraints)
- [ ] T054 **[MANUAL ONLY — EXCLUDED FROM CI]** Execute the manual live smoke test exactly as specified in `quickstart.md` §5 (PowerShell `GET https://nomisma.org/query?query=...` with the fixed `User-Agent`), confirm HTTP 200 / `application/sparql-results+json` / `http://numismatics.org/ocre/id/ric.2.hdn.*` bindings; run this once, locally, before beta release — never add it to any CI workflow or automated test target
- [ ] T055 Update `docs/CHANGELOG.md` with a beta release note for OCRE automation (gate G-OCRE / T155 closure, default-off admin-gated flag, ODbL 1.0 / ANS attribution posture) and confirm ADR 0010's `Status: Accepted` (from T001) is dated to the release

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — can start immediately
- **Foundational (Phase 2)**: Depends on Phase 1 (settings key referenced by nothing yet, but keeps ordering clean) — **BLOCKS Phases 3–7**; must be fully green before any handler/route/Python work begins
- **US1 (Phase 3)**: Depends on Phase 2 — delivers the MVP (flag on ⇒ OCRE candidates surface)
- **US2 (Phase 4)**: Depends on Phase 3 (needs real OCRE claims/attribution strings flowing through evidence to render/test against) — also touches the previously-unwired generic `attributions` mechanism
- **US3 (Phase 5)**: Depends on Phase 3 (exercises the same node/handler code paths under failure conditions) — mostly independent of Phase 4
- **US4 (Phase 6)**: Depends on Phase 2 (settings key) and Phase 3 (catalog/health data to display) — independent of Phase 4/5
- **US5 (Phase 7)**: Depends on Phase 3 only (override logic is pre-existing and generic) — independent of Phase 4/5/6
- **Polish (Phase 8)**: Depends on all desired user-story phases being complete; T054 (manual smoke) should run last, immediately before beta release

### Critical Path

Phase 1 → Phase 2 (T004→T006→T007; T004→T005; T008→T009; T010→T011, all parallelizable within the phase) → Phase 3 (T012→T013→T014→T015; T018→T019→T020, then T021/T022/T023) → Phase 4 (T024→T025→T026; T027→T028→T029→T030; T031→T032) → Phase 8 (T050→T051→T054→T055).

The **shortest path to a demoable MVP** is Phase 1 → Phase 2 → Phase 3 (T004–T023): OCRE candidates surface end-to-end, still without dedicated attribution UI or admin toggle (those default to the same not_automated/off state as today until Phases 4/6 land).

### Parallel Opportunities

- All of Phase 2 (T004–T011) is parallelizable — five independent new files plus their test files
- Within Phase 3: the Go track (T012–T017) and the Python track (T018–T023) can proceed in parallel once Phase 2 is done; T016/T017 are parallel with each other and with the Python track
- Within Phase 4: the Python synthesis track (T024–T026) and the Vue attribution-component track (T027–T028) can proceed in parallel; T029–T032 depend on both
- Phases 5, 6, and 7 can all proceed in parallel with each other once Phase 3 is complete (different files, no shared state)
- Phase 8's T048/T049/T052/T053 are parallelizable; T050/T051/T054/T055 are sequential (each validates the prior step's output)

---

## Parallel Example: Phase 2 (Foundational)

```text
# Launch all five new Go files + their test files together:
Task: "Create src/api/services/ocre_query.go"
Task: "Create src/api/services/ocre_query_test.go"
Task: "Create src/api/services/ocre_client.go"
Task: "Create src/api/services/ocre_client_test.go"
Task: "Create src/api/services/ocre_cache.go"
Task: "Create src/api/services/ocre_cache_test.go"
Task: "Create src/api/services/ocre_scoring.go"
Task: "Create src/api/services/ocre_scoring_test.go"
```

## Parallel Example: Phase 3 (User Story 1)

```text
# Go track and Python track proceed independently once Phase 2 is green:
Task: "internal_tools.go OCRESearch handler + DTOs + Swagger"
Task: "main.go OCRE client/cache wiring + route registration"
Task: "deep_identification_pipeline_runner.go conditional catalog entry"
---
Task: "provider_tools.py ocre_search wrapper"
Task: "providers/ocre.py automated node rewrite"
Task: "graph.py move ocre to _AUTOMATED_PROVIDER_NODES"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1 (Setup) — settings key, budget, coin_type allowlist entry
2. Complete Phase 2 (Foundational) — CRITICAL, blocks everything; verify `go test ./services/...` green
3. Complete Phase 3 (User Story 1) — verify independently: flag on, Roman-Imperial coin, OCRE candidates surface
4. **STOP and VALIDATE** against spec.md US1's Acceptance Scenarios before proceeding

### Incremental Delivery

1. Setup + Foundational → foundation ready, zero behavior change
2. Add US1 → demoable MVP (candidates surface, no attribution UI/admin toggle yet)
3. Add US2 → attribution/license compliance ships (legal requirement — should not ship US1 to real users without it)
4. Add US3 → robustness regressions locked in (should land before/alongside US2, not deferred)
5. Add US4 → admin operability
6. Add US5 → override ergonomics
7. Phase 8 → docs, full gates, one manual smoke, release note

### Parallel Team Strategy

With multiple contributors, after Phase 1+2 land:
- Developer A: Phase 3 Go track (T012–T017) then Phase 6 (admin, T038–T045)
- Developer B: Phase 3 Python track (T018–T023) then Phase 5 (robustness, T033–T037)
- Developer C: Phase 4 (attribution, T024–T032) once Phase 3 claims/attribution strings exist, then Phase 7 (override, T046–T047)
- All converge on Phase 8 for gates, docs, and the single manual smoke test

---

## Summary

- **Total tasks**: 55 (T001–T055)
- **Phase 1 (Setup)**: 3 tasks
- **Phase 2 (Foundational — Go SPARQL boundary)**: 8 tasks — blocks all stories
- **Phase 3 (US1 — P1, MVP)**: 12 tasks (T012–T023)
- **Phase 4 (US2 — P1, attribution/license)**: 9 tasks (T024–T032)
- **Phase 5 (US3 — P1, failure isolation)**: 5 tasks (T033–T037)
- **Phase 6 (US4 — P2, admin enablement/health)**: 8 tasks (T038–T045)
- **Phase 7 (US5 — P2, override)**: 2 tasks (T046–T047)
- **Phase 8 (Polish/QC/release)**: 8 tasks (T048–T055), including exactly **one** manual, CI-excluded live-smoke task (T054)
- **Parallel opportunities**: all of Phase 2 (8 tasks); the Go/Python split within Phase 3; the synthesis/Vue-component split within Phase 4; Phases 5/6/7 run fully in parallel once Phase 3 completes
- **Independent test criteria**: each user-story phase's "Independent Test" line above restates its spec.md acceptance check; Phase 3 alone is independently demoable as the MVP
- **Suggested MVP scope**: Phase 1 + Phase 2 + Phase 3 (User Story 1) — 23 tasks — with the caveat that Phase 4 (attribution/license) is a compliance requirement and should ship in the same release, not be deferred past MVP
- **Readiness**: All paths above were confirmed against the current `345-ocre-deep-analysis-provider` branch source tree before this file was written (no speculative paths); no task in Phases 1–7 performs live network I/O; CI safety is enforced by T053 and the single excluded manual task T054; no RPC, image, or corpus/migration work is included anywhere in this list
