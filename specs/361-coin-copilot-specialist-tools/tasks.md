---
description: "Dependency-ordered implementation tasks for Feature 361 Coin Copilot specialist market tools"
---

# Tasks: Coin Copilot Specialist Market Tools

**Input**: Design artifacts in `C:\Users\brian.denicolafamily\Code\AncientCoins\specs\361-coin-copilot-specialist-tools\`
**Prerequisites**: `spec.md`, `plan.md`, `research.md`, `data-model.md`, `quickstart.md`, and all files under `contracts\`
**Working branch**: Existing `beta` branch only; do not create or switch branches
**Scope guard**: Add exactly `market_search`, `auction_search`, `price_trends`, and `similar_lots`; add no write, arbitrary HTTP, shell, filesystem, direct-database, approval, durable-memory, or Deep Identification capability
**Test policy**: Tests are required and contract-first. For the four tool contracts, complete T004-T007 and confirm their intended failures before implementing T008-T014.

## Phase 1: Setup and Shared Fixtures

**Purpose**: Establish cross-service fixture inputs and a reproducible baseline without changing runtime behavior.

- [ ] T001 Record the pre-change targeted test baseline and the exact Feature 359 compatibility fixtures reused by Feature 361 in `C:\Users\brian.denicolafamily\Code\AncientCoins\specs\361-coin-copilot-specialist-tools\quickstart.md`
- [ ] T002 [P] Add canonical valid input/result/event JSON fixtures for all four capabilities and complete/partial/no-match/unavailable outcomes under `C:\Users\brian.denicolafamily\Code\AncientCoins\src\agent\tests\fixtures\coin_copilot\specialists\`
- [ ] T003 [P] Add invalid and adversarial fixtures for unknown fields, invalid enums, missing provenance, unsafe URLs, duplicate/conflicting identities, oversized content, prompt injection, hidden reasoning, and completed-call replay under `C:\Users\brian.denicolafamily\Code\AncientCoins\src\agent\tests\fixtures\coin_copilot\specialists_invalid\`

**Checkpoint**: Shared fixtures express the complete contract and existing Feature 359 tests still pass unchanged.

---

## Phase 2: Foundational Typed Contracts and Dispatch

**Purpose**: Lock the strict cross-service contract before any user-story behavior is implemented.

**CRITICAL**: T004-T007 are the exactly four contract-first tool test tasks. Run them and confirm failures are caused by missing Feature 361 support before T008 begins.

- [ ] T004 [P] Add failing strict contract tests for `market_search` input, dealer-listing output, provenance, outcomes, and bounds in `C:\Users\brian.denicolafamily\Code\AncientCoins\src\agent\tests\test_coin_copilot_contract.py`
- [ ] T005 [P] Add failing strict contract tests for `auction_search` input, auction-lot output, provenance, outcomes, and bounds in `C:\Users\brian.denicolafamily\Code\AncientCoins\src\api\services\coin_copilot_contract_test.go`
- [ ] T006 Add failing strict contract tests for `price_trends` input, sale-observation output, typed trend summary, provenance, outcomes, and bounds in `C:\Users\brian.denicolafamily\Code\AncientCoins\src\agent\tests\test_coin_copilot_contract.py`
- [ ] T007 Add failing strict contract tests for `similar_lots` input, ranked similar-lot output, provenance, outcomes, and bounds in `C:\Users\brian.denicolafamily\Code\AncientCoins\src\api\services\coin_copilot_contract_test.go`
- [ ] T008 Implement strict Pydantic specialist query, provider-attempt, field-provenance, evidence-item, trend, truncation, and result-envelope models with all structural/string/count invariants in `C:\Users\brian.denicolafamily\Code\AncientCoins\src\agent\app\teams\specialist_contracts.py`
- [ ] T009 [P] Extend checkpoint and frame Pydantic models to discriminate and reject mismatched specialist results while preserving existing frames in `C:\Users\brian.denicolafamily\Code\AncientCoins\src\agent\app\models\requests.py` and `C:\Users\brian.denicolafamily\Code\AncientCoins\src\agent\app\models\responses.py`
- [ ] T010 Generalize the bounded Python dispatcher for local specialist runners without granting callback-route authority in `C:\Users\brian.denicolafamily\Code\AncientCoins\src\agent\app\tools\copilot_collection_tools.py`
- [ ] T011 Implement strict Go specialist DTOs, capability/item matching, outcome invariants, provenance validation, and public projection types in `C:\Users\brian.denicolafamily\Code\AncientCoins\src\api\services\coin_copilot_contract.go`
- [ ] T012 Extend the execution allowlist to exactly ten tools while leaving `copilotCallbackTools` collection-only in `C:\Users\brian.denicolafamily\Code\AncientCoins\src\api\services\coin_copilot_contract.go`
- [ ] T013 [P] Extend TypeScript discriminated unions and strict runtime guards for optional `tool_completed.payload.specialistResult` in `C:\Users\brian.denicolafamily\Code\AncientCoins\src\web\src\types\agent.ts`
- [ ] T014 Run the four contract-first tests against T008-T013 and reconcile shared fixture semantics in `C:\Users\brian.denicolafamily\Code\AncientCoins\src\agent\tests\test_coin_copilot_contract.py` and `C:\Users\brian.denicolafamily\Code\AncientCoins\src\api\services\coin_copilot_contract_test.go`

**Checkpoint**: Python and Go accept the same valid fixtures, reject the same invalid fixtures, and TypeScript has a strict additive event projection.

---

## Phase 3: User Story 1 — Find Current Market and Auction Evidence (Priority: P1) 🎯 MVP

**Goal**: Return bounded, source-backed dealer listings and auction lots through the existing teams, including explicit degradation outcomes.

**Independent Test**: With controlled provider fixtures, request Domitian denarii from dealers and auctions; verify only validated, deduplicated, provenance-backed results appear and each run reports `complete`, `partial`, `no_match`, or `unavailable`.

### Tests for User Story 1

- [ ] T015 [P] [US1] Add dealer-search tests for success, zero matches, timeout, transport failure, unavailable provider, malformed data, and mixed-provider partial results in `C:\Users\brian.denicolafamily\Code\AncientCoins\src\agent\tests\test_coin_copilot_specialists.py`
- [ ] T016 [US1] Add auction-search tests for success, zero matches, timeout, transport failure, unavailable provider, malformed data, and mixed-provider partial results in `C:\Users\brian.denicolafamily\Code\AncientCoins\src\agent\tests\test_coin_copilot_specialists.py`
- [ ] T017 [P] [US1] Add URL-policy and deduplication tests covering HTTPS-only, credentials, localhost/private/link-local/metadata targets, unregistered hosts, redirects, fragments, duplicate identities, and conflicting observations in `C:\Users\brian.denicolafamily\Code\AncientCoins\src\agent\tests\test_coin_copilot_security.py`
- [ ] T018 [P] [US1] Add Go validation tests proving unsupported dealer/auction facts are omitted, invalid provenance fails closed, valid evidence survives partial failure, and raw provider errors never persist in `C:\Users\brian.denicolafamily\Code\AncientCoins\src\api\services\coin_copilot_contract_test.go`
- [ ] T019 [P] [US1] Add Vue tests for accessible complete/partial/no-match/unavailable rendering, safe external links, provenance-visible facts, warnings, and empty evidence states in `C:\Users\brian.denicolafamily\Code\AncientCoins\src\web\src\components\__tests__\CoinSearchChat.copilot.test.ts`

### Implementation for User Story 1

- [ ] T020 [P] [US1] Extract a typed callable dealer-market runner while retaining existing search/provider functions and legacy formatting in `C:\Users\brian.denicolafamily\Code\AncientCoins\src\agent\app\teams\coin_search.py`
- [ ] T021 [P] [US1] Extract a typed callable auction runner while retaining the existing NumisBids/search boundary and legacy formatting in `C:\Users\brian.denicolafamily\Code\AncientCoins\src\agent\app\teams\auction_search.py`
- [ ] T022 [US1] Adapt dealer candidates into validated `dealer_listing` evidence and deterministic provider/aggregate outcomes in `C:\Users\brian.denicolafamily\Code\AncientCoins\src\agent\app\teams\specialist_contracts.py`
- [ ] T023 [US1] Adapt auction candidates into validated `auction_lot` evidence and deterministic provider/aggregate outcomes in `C:\Users\brian.denicolafamily\Code\AncientCoins\src\agent\app\teams\specialist_contracts.py`
- [ ] T024 [US1] Enforce registered-host URL validation, redirect revalidation, canonical source identity, strongest-provenance merge, and conflict warnings at the existing outbound boundaries in `C:\Users\brian.denicolafamily\Code\AncientCoins\src\agent\app\tools\search.py` and `C:\Users\brian.denicolafamily\Code\AncientCoins\src\agent\app\tools\numisbids.py`
- [ ] T025 [US1] Register only `market_search` and `auction_search` with the bounded in-process specialist dispatcher and map them to the existing teams in `C:\Users\brian.denicolafamily\Code\AncientCoins\src\agent\app\teams\coin_copilot.py`
- [ ] T026 [US1] Validate, sanitize, bound, persist, and append-before-publish dealer/auction projections on the existing `tool_completed` event in `C:\Users\brian.denicolafamily\Code\AncientCoins\src\api\services\coin_copilot_worker.go`
- [ ] T027 [US1] Render dealer and auction cards with source, observation time, confidence, verification state, outcome, warnings, and no mutation controls in `C:\Users\brian.denicolafamily\Code\AncientCoins\src\web\src\components\chat\CopilotRunProgress.vue`
- [ ] T028 [US1] Run and record Quickstart Scenarios 1-2 against controlled fixtures in `C:\Users\brian.denicolafamily\Code\AncientCoins\specs\361-coin-copilot-specialist-tools\quickstart.md`

**Checkpoint**: Dealer and auction search are independently usable without arbitrary browsing, invented facts, or mutation actions.

---

## Phase 4: User Story 2 — Understand Price Trends from Cited Observations (Priority: P1)

**Goal**: Produce evidence-linked trend summaries without mixing incomparable currencies or price bases.

**Independent Test**: Request a trend from controlled completed-sale observations; verify deduplicated sample size, coverage dates, currency, price basis, range/median, confidence, limitations, citations, and `unknown` for insufficient or incomparable evidence.

### Tests for User Story 2

- [ ] T029 [P] [US2] Add price-trend provider outcome tests for success, no-match, timeout, failure, unavailable, malformed, partial, and prompt-injection-bearing evidence in `C:\Users\brian.denicolafamily\Code\AncientCoins\src\agent\tests\test_coin_copilot_specialists.py`
- [ ] T030 [US2] Add trend sufficiency tests for fewer than three verified sales, fewer than two sale dates, less than 30-day coverage, duplicate samples, and qualifying rising/stable/declining samples in `C:\Users\brian.denicolafamily\Code\AncientCoins\src\agent\tests\test_coin_copilot_specialists.py`
- [ ] T031 [P] [US2] Add tests proving currencies and hammer versus premium-inclusive price bases stay separate and unsupported conversion or direction claims are rejected in `C:\Users\brian.denicolafamily\Code\AncientCoins\src\agent\tests\test_coin_copilot_security.py`
- [ ] T032 [P] [US2] Add Go tests for sale-observation provenance, supporting-source references, typed unknown/unavailable states, and bounded trend projection in `C:\Users\brian.denicolafamily\Code\AncientCoins\src\api\services\coin_copilot_contract_test.go`
- [ ] T033 [P] [US2] Add Vue tests for trend direction/unknown state, sample metadata, limitations, grouped incomparable evidence, and safe source links in `C:\Users\brian.denicolafamily\Code\AncientCoins\src\web\src\components\__tests__\CoinSearchChat.copilot.test.ts`

### Implementation for User Story 2

- [ ] T034 [US2] Extract a typed callable trend runner that reuses existing search/analysis functions and preserves legacy team behavior in `C:\Users\brian.denicolafamily\Code\AncientCoins\src\agent\app\teams\price_trends.py`
- [ ] T035 [US2] Normalize completed-sale observations and compute deterministic comparable groups, sufficiency, range, median, confidence, and limitations in `C:\Users\brian.denicolafamily\Code\AncientCoins\src\agent\app\teams\specialist_contracts.py`
- [ ] T036 [US2] Register `price_trends` with the bounded in-process dispatcher and expose only normalized untrusted evidence to the model in `C:\Users\brian.denicolafamily\Code\AncientCoins\src\agent\app\teams\coin_copilot.py`
- [ ] T037 [US2] Render the typed trend summary and its supporting observations without inferred conversion or uncited claims in `C:\Users\brian.denicolafamily\Code\AncientCoins\src\web\src\components\chat\CopilotRunProgress.vue`
- [ ] T038 [US2] Run and record Quickstart Scenario 3 against comparable and incomparable controlled fixtures in `C:\Users\brian.denicolafamily\Code\AncientCoins\specs\361-coin-copilot-specialist-tools\quickstart.md`

**Checkpoint**: Price trends are independently testable and never overstate sparse, duplicated, or incomparable evidence.

---

## Phase 5: User Story 4 — Preserve Durable, Bounded, Safe Harness Behavior (Priority: P1)

**Goal**: Preserve Feature 359 ownership, budgets, cancellation, replay, fallback, and least-privilege guarantees while specialist providers add latency and untrusted data.

**Independent Test**: Cancel during a blocked provider call, replay/resume a persisted completed call, attempt foreign-owner access, use unsupported models and the disabled flag, and tamper with every critical identity/allowlist/provenance guard; verify no late commit, duplicate call, disclosure, or capability escalation occurs.

### Tests for User Story 4

- [ ] T039 [P] [US4] Add run-budget tests proving specialist calls consume one shared tool call, preserve 8-iteration/12-tool defaults, one-at-a-time execution, 120-second default/150-second maximum, cumulative token observations, and no nested retry budget in `C:\Users\brian.denicolafamily\Code\AncientCoins\src\agent\tests\test_coin_copilot_harness.py`
- [ ] T040 [P] [US4] Add payload-bound tests for query/item/provider-attempt/warning/provenance/URL/text limits plus deterministic 32 KiB digest truncation and final-answer omitted-evidence disclosure in `C:\Users\brian.denicolafamily\Code\AncientCoins\src\agent\tests\test_coin_copilot_contract.py`
- [ ] T041 [US4] Add cancellation-race tests for pre-dispatch cancellation, cancellation after each awaited provider operation, and zero completion/checkpoint/final-answer frames after cancellation wins in `C:\Users\brian.denicolafamily\Code\AncientCoins\src\agent\tests\test_coin_copilot_harness.py`
- [ ] T042 [P] [US4] Add restart/resume tests proving completed specialist call ids and truncated results hydrate from checkpoints without repeating provider execution in `C:\Users\brian.denicolafamily\Code\AncientCoins\src\api\integration\coin_copilot_seam_test.go`
- [ ] T043 [P] [US4] Add owner-isolation tests for read, event stream, cancel, and resume with identical foreign/unknown `404` behavior in `C:\Users\brian.denicolafamily\Code\AncientCoins\src\api\handlers\coin_copilot_test.go`
- [ ] T044 [P] [US4] Add feature-off, missing/malformed/timed-out/ambiguous tool-support, and startup-failure tests proving legacy fallback creates no durable run or specialist call while accepted runs remain readable/cancellable in `C:\Users\brian.denicolafamily\Code\AncientCoins\src\web\src\composables\__tests__\useCoinCopilot.test.ts`
- [ ] T045 [P] [US4] Add prompt-injection-as-data tests proving provider instructions and token-shaped strings cannot alter plans, call tools, expose prompts/credentials, change owner scope, bypass cancellation, or create unsupported claims in `C:\Users\brian.denicolafamily\Code\AncientCoins\src\agent\tests\test_coin_copilot_security.py`
- [ ] T046 [P] [US4] Add tamper tests for owner/run/execution/tool credential binding, tool/result-kind mismatch, altered digest/size metadata, extra fields, fabricated provenance, replayed call ids, and late frames in `C:\Users\brian.denicolafamily\Code\AncientCoins\src\api\services\coin_copilot_worker_test.go`
- [ ] T047 [P] [US4] Add architecture and route tamper guards proving the callback route set remains collection-only and contains no specialist HTTP, write, arbitrary-fetch, shell, filesystem, database, approval, or Deep Identification tool in `C:\Users\brian.denicolafamily\Code\AncientCoins\src\api\handlers\coin_copilot_internal_tools_test.go`
- [ ] T048 [P] [US4] Add Python architecture guards proving the harness remains stateless/DB-free and imports no write, shell, filesystem, arbitrary-HTTP, approval, or Deep Identification capability in `C:\Users\brian.denicolafamily\Code\AncientCoins\src\agent\tests\test_coin_copilot_architecture.py`

### Implementation for User Story 4

- [ ] T049 [US4] Apply shared run deadline, iteration/tool/concurrency counters, cancellation checks, checkpoint hydration, untrusted-data delimiters, and exact ten-tool selection in `C:\Users\brian.denicolafamily\Code\AncientCoins\src\agent\app\teams\coin_copilot.py`
- [ ] T050 [US4] Enforce authoritative post-await cancellation, specialist result validation, deterministic bounds/digest, replay idempotency, and append-before-publish semantics in `C:\Users\brian.denicolafamily\Code\AncientCoins\src\api\services\coin_copilot_worker.go`
- [ ] T051 [US4] Emit privacy-safe observability fields for capability, provider id/outcome, aggregate outcome, duration, count, bytes, truncation, digest, run id, and execution id while excluding query/content/URL/prompt/credential/raw-error data in `C:\Users\brian.denicolafamily\Code\AncientCoins\src\api\services\coin_copilot_worker.go`
- [ ] T052 [US4] Preserve replay sequence de-duplication, terminal handling, cancellation, and feature/model fallback behavior for specialist projections in `C:\Users\brian.denicolafamily\Code\AncientCoins\src\web\src\composables\useCoinCopilot.ts`
- [ ] T053 [US4] Run and record Quickstart Scenarios 5-8, including cancellation race, replay/resume, owner isolation, unsupported models, and feature-off fallback, in `C:\Users\brian.denicolafamily\Code\AncientCoins\specs\361-coin-copilot-specialist-tools\quickstart.md`

**Checkpoint**: Specialist work has the same durable, bounded, owner-scoped, cancellable, replayable, default-off behavior as Feature 359.

---

## Phase 6: User Story 3 — Compare a Coin with Similar Active Lots (Priority: P2)

**Goal**: Rank credible active lots through the existing similarity team and explain matches and material differences.

**Independent Test**: Compose `get_coin` with `similar_lots`, then verify score-descending/canonical-URL ordering, explicit match reasons and differences, omission of weak or invalid candidates, and `no_match` without padding.

### Tests for User Story 3

- [ ] T054 [P] [US3] Add similar-lot provider tests for success, no-match, timeout, failure, unavailable, malformed, and partial outcomes in `C:\Users\brian.denicolafamily\Code\AncientCoins\src\agent\tests\test_coin_copilot_specialists.py`
- [ ] T055 [US3] Add ranking tests for minimum identifying evidence, bounded score, required matched attributes, explicit material differences, descending score, canonical-URL tie-break, and weak-candidate omission in `C:\Users\brian.denicolafamily\Code\AncientCoins\src\agent\tests\test_coin_copilot_specialists.py`
- [ ] T056 [P] [US3] Add composition/replay tests proving `get_coin` then `similar_lots` remains sequential, consumes shared budgets, and does not repeat either completed call after resume in `C:\Users\brian.denicolafamily\Code\AncientCoins\src\agent\tests\test_coin_copilot_harness.py`
- [ ] T057 [P] [US3] Add Vue tests for ranked cards, match reasons, material differences, stable keys, no-match, and omission of write/deep-identification actions in `C:\Users\brian.denicolafamily\Code\AncientCoins\src\web\src\components\__tests__\CoinSearchChat.copilot.test.ts`

### Implementation for User Story 3

- [ ] T058 [US3] Extract a typed callable similarity runner that reuses the existing scorer/search functions and preserves legacy team behavior in `C:\Users\brian.denicolafamily\Code\AncientCoins\src\agent\app\teams\similar_lots.py`
- [ ] T059 [US3] Normalize and deterministically rank validated similar-lot evidence with explicit matched attributes and material differences in `C:\Users\brian.denicolafamily\Code\AncientCoins\src\agent\app\teams\specialist_contracts.py`
- [ ] T060 [US3] Register `similar_lots` with the bounded in-process dispatcher and support sequential composition with existing read-only collection tools in `C:\Users\brian.denicolafamily\Code\AncientCoins\src\agent\app\teams\coin_copilot.py`
- [ ] T061 [US3] Render ranked similar-lot evidence with source/provenance details and no save, watch, bid, approval, or Deep Identification action in `C:\Users\brian.denicolafamily\Code\AncientCoins\src\web\src\components\chat\CopilotRunProgress.vue`
- [ ] T062 [US3] Run and record Quickstart Scenario 4 with owned-coin composition, weak candidates, and deterministic ties in `C:\Users\brian.denicolafamily\Code\AncientCoins\specs\361-coin-copilot-specialist-tools\quickstart.md`

**Checkpoint**: Similar-lot comparison is independently testable and transparent without inventing or padding candidates.

---

## Phase 7: Documentation, Quality Gates, and Cross-Cutting Polish

**Purpose**: Synchronize public documentation, prove exact workflow contracts, audit the major multi-service change, and close the decision record.

- [ ] T063 [P] Document exactly four capabilities, typed evidence/provenance, degradation states, budgets, replay/cancellation, default-off fallback, and explicit non-goals in `C:\Users\brian.denicolafamily\Code\AncientCoins\docs\features\coin-copilot.md`
- [ ] T064 [P] Document the additive SSE projection, safe source behavior, and absence of new public/internal specialist routes in `C:\Users\brian.denicolafamily\Code\AncientCoins\docs\api-reference.md`
- [ ] T065 [P] Document targeted specialist, tamper, replay, fallback, and full-suite commands in `C:\Users\brian.denicolafamily\Code\AncientCoins\docs\testing.md`
- [ ] T066 Update Swagger annotations for the additive SSE description, run `task openapi`, and verify synchronized generated output in `C:\Users\brian.denicolafamily\Code\AncientCoins\src\api\docs\docs.go`, `C:\Users\brian.denicolafamily\Code\AncientCoins\src\api\docs\swagger.json`, `C:\Users\brian.denicolafamily\Code\AncientCoins\src\api\docs\swagger.yaml`, and `C:\Users\brian.denicolafamily\Code\AncientCoins\docs\openapi.json`
- [ ] T067 Add or extend the route/OpenAPI drift test proving every public Coin Copilot route remains documented and no specialist/internal route enters public OpenAPI in `C:\Users\brian.denicolafamily\Code\AncientCoins\src\api\route_openapi_drift_test.go`
- [ ] T068 Run the targeted Constitution §17 workflow-contract gates from `C:\Users\brian.denicolafamily\Code\AncientCoins\` using the commands in `C:\Users\brian.denicolafamily\Code\AncientCoins\specs\361-coin-copilot-specialist-tools\quickstart.md`
- [ ] T069 Run the full Constitution §17 gates from `C:\Users\brian.denicolafamily\Code\AncientCoins\`: Go build/vet/test/race, Python Ruff/full pytest, Vue type-check/test/build, OpenAPI generation/drift, gitleaks when configured, and Trivy production-image scans when configured
- [ ] T070 Execute all eight controlled-fixture acceptance scenarios and record pass/fail evidence plus any justified manual-only checks in `C:\Users\brian.denicolafamily\Code\AncientCoins\specs\361-coin-copilot-specialist-tools\quickstart.md`
- [ ] T071 Run `/post-major-work-qc-audit` after T068-T070 and record remediation tasks or the clean audit outcome in `C:\Users\brian.denicolafamily\Code\AncientCoins\specs\361-coin-copilot-specialist-tools\quickstart.md`
- [ ] T072 Update ADR 0016 with the additive four-capability boundary, unchanged Go durability/Python statelessness decision, and no-new-route/no-new-persistence consequences in `C:\Users\brian.denicolafamily\Code\AncientCoins\docs\adr\0016-go-owned-durable-coin-copilot-state.md`
- [ ] T073 Append the Feature 361 implementation and verification decisions without rewriting prior entries in `C:\Users\brian.denicolafamily\Code\AncientCoins\.squad\decisions.md`

**Checkpoint**: Documentation and generated API artifacts are synchronized, all targeted/full gates pass, audit findings are resolved or tracked, and the decision log reflects the shipped boundaries.

---

## Dependencies and Execution Order

### Phase Dependencies

1. **Phase 1 (Setup)** has no dependency.
2. **Phase 2 (Foundational)** depends on Phase 1; T004-T007 must fail for the intended missing-contract reason before T008-T014.
3. **Phase 3 (US1)** depends on Phase 2 and is the suggested MVP.
4. **Phase 4 (US2)** depends on Phase 2 and the shared source normalization from T024; it may otherwise proceed alongside later US1 UI work.
5. **Phase 5 (US4)** depends on Phase 2 and on at least one executable specialist path from Phase 3 so race/replay/fallback tests exercise real specialist behavior; it blocks release.
6. **Phase 6 (US3)** depends on Phase 2 and the durable/budget behavior established by Phase 5; its provider/ranking work may begin after Phase 2, but composition acceptance waits for T049-T052.
7. **Phase 7 (Polish/Gates)** depends on all selected user-story phases; T071 depends on T068-T070, and T072-T073 follow validated implementation decisions.

### User Story Dependencies

- **US1 (P1)**: Starts after Phase 2; no dependency on another user story.
- **US2 (P1)**: Starts after Phase 2; reuses US1 URL validation/deduplication from T024 but remains independently testable with sale fixtures.
- **US4 (P1)**: Starts after Phase 2; end-to-end race/replay checks require one implemented specialist path from US1.
- **US3 (P2)**: Its runner/ranking can start after Phase 2; composed `get_coin` + `similar_lots` acceptance depends on US4 budget/replay controls.

### Critical Path

`T001 → T002/T003 → T004-T007 → T008 → T009-T014 → T015-T019 → T020-T027 → T039-T048 → T049-T053 → T054-T061 → T063-T067 → T068-T070 → T071 → T072-T073`

The MVP critical path ends at T028. The release critical path continues through the US4 safety phase, US3 composition, generated OpenAPI/docs, targeted/full §17 gates, the post-major-work QC audit, and decision-log updates.

---

## Safe Parallel Execution Examples

### Foundational contracts

After T002-T003, T004 and T005 can run in parallel; T006 follows T004 in the shared Python test file and T007 follows T005 in the shared Go test file. After all four tests fail as expected, T009 and T013 can proceed in parallel with T008 because they edit separate service files, but T014 waits for all contract implementations.

### User Story 1

T015, T017, T018, and T019 may run in parallel across Python, Go, and Vue test files; T016 follows T015 in the shared specialist test file. After tests are fixed, T020 and T021 are safe in parallel because they modify separate team modules. T022 and T023 are sequential in the shared adapter file; T026 and T027 can proceed in parallel after their respective backend/frontend contracts are ready.

### User Story 2

T029, T031, T032, and T033 may run in parallel across the listed test files; T030 follows T029 in the shared specialist test file. T034 can proceed in parallel with T032-T033. T035 precedes T036, while T037 can proceed after T013 and the Vue tests are defined.

### User Story 4

T039-T048 are safe by file boundary except tasks sharing the same Python test file must be serialized. T049, T050, and T052 can proceed in parallel after their tests exist; T051 follows T050 because both modify the Go worker.

### User Story 3

T054, T056, and T057 may run in parallel across the listed test files; T055 follows T054 in the shared specialist test file. T058 can proceed in parallel with Vue work; T059 precedes T060, and T061 follows the TypeScript projection contract.

### Documentation and gates

T063-T065 may run in parallel. T066-T067 can proceed in parallel with documentation but must finish before T068-T070. Run T068 and T069 separately to keep targeted failures diagnosable; invoke T071 only after both gates and acceptance scenarios complete.

---

## Implementation Strategy

### MVP First

1. Complete Phase 1 and the four contract-first tests in Phase 2.
2. Complete the remaining foundational contracts and dispatch.
3. Complete US1 only and validate Quickstart Scenarios 1-2.
4. Stop and demonstrate bounded dealer/auction evidence before adding trend and similarity behavior.

### Incremental Delivery

1. **Foundation**: Shared strict fixtures and cross-service types.
2. **US1 MVP**: Dealer and auction evidence.
3. **US2**: Cited, uncertainty-aware price trends.
4. **US4 release guard**: Budgets, payloads, cancellation, replay, owner isolation, injection resistance, fallback, and forbidden-capability guards.
5. **US3**: Transparent similar-lot composition.
6. **Release closure**: Docs/OpenAPI, targeted/full §17 gates, QC audit, and decision records.

### Completion Rules

- A test task is complete only after it first fails for the intended missing behavior and passes after its implementation dependency.
- A capability is complete only when success, no-match, timeout, failure, unavailable, malformed, and partial behavior are covered.
- No presented fact is complete without validated URL, observation time, confidence, verification state, and field provenance.
- No phase is release-ready while a write, Deep Identification, arbitrary network, shell, filesystem, or database capability is reachable.
- Do not create/switch branches, commit, push, or hand-edit generated OpenAPI artifacts.
