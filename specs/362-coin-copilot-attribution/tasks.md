# Tasks: Coin Copilot Attribution Integration

**Input**: Design documents from `specs/362-coin-copilot-attribution/`
**Prerequisites**: `plan.md`, `spec.md`, `research.md`, `data-model.md`, `contracts/coin-copilot-deep-analysis.md`, `quickstart.md`
**Branch constraint**: Work only on the existing `beta` branch. Do not create or switch branches, deploy, or modify unrelated dirty `.squad/` files.

**Tests**: Feature 362 explicitly requires test-first contract, tamper, race, lifecycle, preservation, architecture, and regression coverage. For every test task below, add the failing assertion before its implementation task and confirm the failure is for the intended missing behavior.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Safe to execute concurrently because the task changes different files and does not depend on unfinished implementation.
- **[Story]**: Maps the task to a user story in `spec.md`.
- Every task names its exact file set; no task authorizes direct agent mutation, a second attribution engine, or duplicate proposal UI.

---

## Phase 1: Strict Contracts and Canonical Tamper Fixtures

**Purpose**: Freeze the smallest cross-layer contract and make unsafe expansion fail before any callback route or model behavior exists.

- [ ] T001 Create canonical valid request/result and lifecycle fixtures for `request`, `status`, and `rerun` in `specs/362-coin-copilot-attribution/contracts/fixtures/deep-analysis-handoff-valid.json`
- [ ] T002 [P] Create canonical tamper cases for unknown fields/enums, wrong target kind or target/job pairing, non-positive IDs, malformed/oversized output, duplicate fields/evidence/tool calls, non-finite or out-of-range confidence, forged accept/apply/provider options, prompt injection, token-shaped secrets, and unsafe citation/review URLs in `specs/362-coin-copilot-attribution/contracts/fixtures/deep-analysis-handoff-invalid.json`
- [ ] T003 [P] Add test-first Go fixture drift and fail-closed decoding coverage, including the 64 KiB request/result cap and exact operation/outcome/job/result/provider vocabularies, in `src/api/services/coin_copilot_contract_test.go`
- [ ] T004 [P] Add test-first Python mirror coverage that loads both canonical fixture files and rejects every invalid request/result without coercion in `src/agent/tests/test_coin_copilot_contract.py`
- [ ] T005 [P] Add test-first TypeScript parsing/type cases for valid handoff events plus malformed, oversized, apply-like, unsafe-URL, and target/job-mismatch payloads in `src/web/src/api/endpoints/__tests__/agent.test.ts`
- [ ] T006 Define strict Go handoff request/result/projection types, enum/range/size/duplicate validation, untrusted-text handling, and relative review-route validation in `src/api/services/coin_copilot_contract.go`
- [ ] T007 [P] Define strict Pydantic handoff argument/result/projection mirrors with `extra="forbid"`, finite confidence bounds, mutual-field rules, bounded collections/text, and fixed relative review URLs in `src/agent/app/models/requests.py` and `src/agent/app/models/responses.py`
- [ ] T008 [P] Define the typed public handoff projection and runtime fail-closed parser without apply/edit properties in `src/web/src/types/agent.ts` and `src/web/src/api/endpoints/agent.ts`

**Checkpoint**: Go, Python, and TypeScript accept the same canonical payloads and reject all tampered variants; no route, database change, or agent behavior exists yet.

---

## Phase 2: Foundational Persistence and Boundary Guards

**Purpose**: Add the minimum durable draft binding and prove the service boundary cannot widen.

**⚠️ CRITICAL**: Complete this phase before any user-story implementation.

- [ ] T009 Add test-first migration assertions for nullable `source_draft_id`, deterministic migration order, owner/draft lookup index presence, `coin_id`/`source_draft_id` exclusivity invariant, no backfill of existing jobs, and rollback-code readability/fail-closed apply behavior in `src/api/database/database_test.go`
- [ ] T010 Add test-first model/repository ownership cases proving draft-bound jobs are owner scoped, source binding is immutable, and foreign/unknown draft/job lookups are indistinguishable in `src/api/repository/deep_identification_repository_test.go`
- [ ] T011 Add nullable indexed `SourceDraftID *uint` and the at-most-one-target invariant to `src/api/models/deep_identification_job.go`, then wire additive no-backfill migration ordering in `src/api/database/database.go`
- [ ] T012 Implement owner/fingerprint-scoped newest-first retained-terminal lookup and immutable source-draft persistence in `src/api/repository/deep_identification_repository.go`
- [ ] T013 [P] Add exact architecture tests proving Python has no database/filesystem/shell/generic-HTTP/provider/apply/write imports or tools and the browser never calls Python directly in `src/agent/tests/test_coin_copilot_architecture.py`
- [ ] T014 [P] Add exact Go route-set tests proving the callback allowlist can gain only `deep_analysis_handoff`, only `POST /api/internal/copilot/tools/deep_analysis_handoff` is added, middleware binds that literal capability, and no wildcard, arbitrary `/api/internal/tools/*`, or Deep apply endpoint becomes reachable in `src/api/architecture_test.go` and `src/api/integration/coin_copilot_seam_test.go`
- [ ] T015 Register only the `deep_analysis_handoff` literal in `CoinCopilotAllowedTools` and the fixed callback allowlist while preserving all existing collection and specialist tools in `src/api/services/coin_copilot_contract.go` and `src/api/services/coin_copilot_proxy.go`

**Checkpoint**: Existing rows remain unchanged, draft ownership has a durable indexed binding, old-code rollback cannot misapply a draft job, and architecture tests reject any authority expansion.

---

## Phase 3: User Story 1 — Resolve a Target and Start/Reopen Deep Analysis (Priority: P1) 🎯 MVP

**Goal**: Resolve exactly one owned coin or active draft and reuse or admit the existing Deep Analysis workflow without a second attribution path.

**Independent Test**: Request attribution for one owned coin and one active draft with distinct valid faces; each returns the current equivalent Deep job, while ambiguity, foreign IDs, mismatched kinds, changed/deleted targets, and invalid images create no job.

### Tests for User Story 1

- [ ] T016 [P] [US1] Add test-first saved-coin and draft resolver cases for owner scoping, foreign/unknown equality, target kind mismatch, prompt/context ID disagreement, inactive/promoted/discarded drafts, target changes during resolution, and missing/duplicate/unsupported face images in `src/api/services/deep_analysis_handoff_service_test.go`
- [ ] T017 [P] [US1] Add test-first v2 fingerprint cases for owner, target class/id, face-content hashes, bounded normalized context, sorted provider selection, image/note/provider changes, and continued readability of v1 direct-entry jobs in `src/api/services/deep_identification_service_test.go`
- [ ] T018 [P] [US1] Add test-first repository/concurrency cases proving equivalent active requests and concurrent admissions return one job/provider fan-out while target-only, changed-input, and different-owner requests do not collide in `src/api/repository/deep_identification_repository_test.go` and `src/api/services/deep_identification_concurrency_test.go`
- [ ] T019 [P] [US1] Add test-first handler cases for strict body decoding, execution-token owner derivation, foreign/unknown coin/draft/job equality, target-kind mismatch, body limits, redacted errors, and exact HTTP/domain mappings in `src/api/handlers/coin_copilot_internal_tools_test.go`

### Implementation for User Story 1

- [ ] T020 [US1] Implement HTTP-agnostic owner-scoped coin/draft resolution, active-draft recheck, distinct usable face validation, bounded context snapshotting, and corrective outcomes in `src/api/services/deep_analysis_handoff_service.go`
- [ ] T021 [US1] Extend v2 canonical fingerprint input with owner, target type/id, current face hashes, bounded context digest, and Go-selected providers while preserving v1 direct Deep Analysis and Fast Identify behavior in `src/api/services/deep_identification_service.go`
- [ ] T022 [US1] Implement active-job reuse, retained-current-result lookup, explicit-rerun admission, and sole delegation to existing `CreateJobFromIntake`/`StartJob` paths in `src/api/services/deep_analysis_handoff_service.go`
- [ ] T023 [US1] Add the thin strict callback handler with `AuthorizeToolCall`/`FinishToolCall`, body cap, generic error mapping, and no business logic in `src/api/handlers/coin_copilot_internal_tools.go`
- [ ] T024 [US1] Inject the handoff service from existing repositories/services/settings/image boundaries in `src/api/deps.go` and register only `POST /api/internal/copilot/tools/deep_analysis_handoff` with literal execution-token middleware in `src/api/routes_internal.go`
- [ ] T025 [US1] Add end-to-end seam tests proving owned coin/draft acceptance, clarification/no-call behavior, active reuse, explicit rerun, no second provider pipeline, and unchanged direct Deep Analysis and Fast Identify entry points in `src/api/integration/coin_copilot_seam_test.go` and `src/api/integration/deep_identification_seam_test.go`

**Checkpoint**: User Story 1 independently starts or reopens only the authoritative Deep Analysis workflow and discloses nothing about foreign resources.

---

## Phase 4: User Story 2 — Explain Persisted Results and Reopen Review (Priority: P1)

**Goal**: Return a bounded, validated conversational projection of persisted Deep report/proposal data and the exact existing review route.

**Independent Test**: Project completed, partial, no-match, image-only, low-confidence, conflicting, pruned-event, queued/running, and missing-result fixtures; verify confidence/provenance/coverage/limitations are preserved, unsafe citations are omitted, and no invented or writable data appears.

### Tests for User Story 2

- [ ] T026 [P] [US2] Add test-first projection tests for complete/partial/no-match/image-only/low-confidence/conflicting evidence, unresolved questions, provider coverage/attribution, pruned events, bounded arrays/text, and exclusion of raw notes, paths, credentials, edit decisions, and apply tokens in `src/api/services/deep_analysis_handoff_projection_test.go`
- [ ] T027 [P] [US2] Add test-first URL/citation tests for existing provider-host allowlists, unsafe schemes, embedded credentials, malformed/unapproved hosts, mismatched review URLs, and honest omission limitations in `src/api/services/deep_identification_contract_drift_test.go`
- [ ] T028 [P] [US2] Add test-first lifecycle tests proving queued/running status never restarts work, retained completed/partial results survive pruned events, and failed/cancelled/stale/expired/input-mismatched/missing-result jobs are never described as current success in `src/api/services/deep_analysis_handoff_service_test.go`

### Implementation for User Story 2

- [ ] T029 [US2] Implement strict decoding of only persisted Deep report/proposal snapshots and derive the bounded non-persisted result projection in `src/api/services/deep_analysis_handoff_projection.go`
- [ ] T030 [US2] Reuse existing Deep source-host validation and fixed provider/license vocabulary, preserving Numista/Nomisma automation, separately enabled OCRE attribution, NGC link-out-only behavior, and RPC unavailable status in `src/api/services/deep_analysis_handoff_projection.go`
- [ ] T031 [US2] Return lifecycle-accurate status/result limitations, current input digest, retained-result reuse marker, explicit fresh-analysis availability, and canonical `/deep-analysis/{positiveJobID}` in `src/api/services/deep_analysis_handoff_service.go`
- [ ] T032 [US2] Add handler/seam assertions that malformed persisted JSON fails closed, projections stay inside existing checkpoint/event payload bounds, and public `tool_completed.result` contains no raw storage or mutation authority in `src/api/handlers/coin_copilot_internal_tools_test.go` and `src/api/integration/coin_copilot_seam_test.go`

**Checkpoint**: User Story 2 can honestly explain and reopen any supported persisted state without provider re-query, result invention, or target mutation.

---

## Phase 5: User Story 3 — Keep All Changes Review-Gated (Priority: P1)

**Goal**: Bind draft-origin proposals safely while preserving the existing per-field review/apply bridge for collection, wishlist, and draft destinations.

**Independent Test**: Explain/open/replay without confirmation and verify zero writes; then accept one valid field in the existing editor and verify only that field changes, manual data remains byte-for-byte, references append/dedupe, and invalid destination fields cannot apply.

### Tests for User Story 3

- [ ] T033 [P] [US3] Add test-first collection/wishlist/active-draft proposal tests for one-field acceptance, undecided/rejected preservation, destination allowlists, wishlist-versus-collection rules, inactive/changed/foreign draft rejection, and no collection-only/manual-field overwrite in `src/api/services/deep_identification_proposal_integration_test.go`
- [ ] T034 [P] [US3] Add test-first manual preservation fixtures covering scalar fields, notes, images, acquisition/provenance, valuation, storage, privacy/status, relationships, and no conversational apply/write in `src/api/services/deep_identification_proposal_phase6b_test.go`
- [ ] T035 [P] [US3] Add test-first structured-reference cases for validated URL/reference-number rules, additive append, equivalent replay dedupe, preservation of every existing row, wishlist validity, draft staging, and rejection of destructive replacement in `src/api/services/deep_identification_proposal_test.go`

### Implementation for User Story 3

- [ ] T036 [US3] Bind draft-origin job creation immutably to the exact active owned `SourceDraftID` and revalidate target liveness/ownership before proposal edit or apply in `src/api/services/deep_identification_service.go` and `src/api/services/deep_identification_proposal.go`
- [ ] T037 [US3] Route only individually accepted destination-valid fields through existing `DeepIdentificationProposalService.Apply`, preserving current collection/wishlist/draft allowlists and all unaccepted/manual values in `src/api/services/deep_identification_proposal.go`
- [ ] T038 [US3] Reuse the existing validated additive/deduplicating reference services for collection, wishlist, and staged draft references, with no replace/delete behavior in `src/api/services/deep_identification_proposal.go`
- [ ] T039 [US3] Add a guard test proving no `apply`, `accept`, `edit`, `cancel_deep_job`, reference write, or generic mutation operation exists in Go/Python handoff definitions or routes in `src/api/architecture_test.go` and `src/agent/tests/test_coin_copilot_architecture.py`

**Checkpoint**: User Story 3 changes nothing from conversation and preserves every unaccepted/manual field and prior reference through the existing confirm-gated editor.

---

## Phase 6: User Story 4 — Bounded Python Orchestration, Cancellation, Replay, and Fallback (Priority: P1)

**Goal**: Let the stateless Python harness invoke the one fixed callback within existing budgets, replay completed facts without re-execution, and recover safely from races or unavailable capabilities.

**Independent Test**: Exercise duplicate/replayed calls, changed-target replay, cancellation before admission and after each await, late frames, malformed model/callback output, failed/cancelled/stale jobs, disabled Deep/Copilot, unsupported models, and pre-accept failures; verify deterministic reuse/fallback and no duplicate work or write.

### Tests for User Story 4

- [ ] T040 [P] [US4] Add test-first callback-client cases for exact route construction, execution token use, strict result validation, digest/bounds/sanitization, duplicate completed-call dedupe, changed-target replay rejection, and no arbitrary URL or Deep apply route in `src/agent/tests/test_coin_copilot_contract.py`
- [ ] T041 [P] [US4] Add test-first harness cases for clarification before tool use, prompt/context disagreement, `request`/`status`/explicit `rerun` choice, handoff-only serial execution, existing iteration/tool/concurrency/wall-clock/token/payload bounds, and no nested unbounded budget in `src/agent/tests/test_coin_copilot_harness.py`
- [ ] T042 [US4] Add deterministic test-first cancellation/replay cases for cancel-before-admission, admission-before-cancel, cancellation after every await, late result rejection, completed-checkpoint reconstruction without callback, duplicate/replayed calls, and stale/failed/cancelled result handling in `src/agent/tests/test_coin_copilot_harness.py`
- [ ] T043 [P] [US4] Add test-first security/fallback cases for malformed/oversized output, unknown fields/states, injection text, token-shaped secrets, URL violations, forged apply/provider overrides, unavailable Deep capability, default-off Copilot, unsupported model, and pre-accept legacy fallback in `src/agent/tests/test_coin_copilot_security.py` and `src/agent/tests/test_coin_copilot_capabilities.py`

### Implementation for User Story 4

- [ ] T044 [US4] Register exactly `deep_analysis_handoff` in `COPILOT_ALLOWED_TOOLS`, `CALLBACK_TOOLS`, `ARG_MODELS`, and `RESULT_MODELS`, reusing the fixed callback client and completed-call digest path in `src/agent/app/tools/copilot_collection_tools.py`
- [ ] T045 [US4] Add one typed tool definition and prompt policy for exact-target clarification, status/reuse, explicit rerun, persisted-result honesty, and the prohibition on conversational apply/write in `src/agent/app/teams/coin_copilot.py`
- [ ] T046 [US4] Execute request/rerun handoffs alone, retain every existing harness bound, check cancellation around each await, and restore completed handoff facts from checkpoints without another Go call in `src/agent/app/teams/coin_copilot.py`
- [ ] T047 [US4] Linearize Go cancellation against request/rerun admission per execution, return zero jobs when cancellation wins, preserve admitted Deep jobs as independent durable work, and reject late Python settlement in `src/api/services/coin_copilot_service.go` and `src/api/services/deep_analysis_handoff_service.go`
- [ ] T048 [US4] Add Go race tests for cancel-before-admission, admission-before-cancel, duplicate concurrent calls, replay, exactly-one terminal Copilot event, and late-frame rejection in `src/api/services/coin_copilot_service_test.go` and `src/api/services/coin_copilot_worker_test.go`
- [ ] T049 [US4] Preserve current collection callbacks, specialist tools, feature flags, unsupported-model handling, and pre-accept legacy fallback with regression assertions in `src/agent/tests/test_coin_copilot_specialists.py`, `src/agent/tests/test_coin_copilot_capabilities.py`, and `src/api/services/coin_copilot_proxy_test.go`

**Checkpoint**: User Story 4 remains bounded, replay-safe, cancellation-safe, capability-aware, stateless in Python, and backward compatible.

---

## Phase 7: Vue Handoff Card and Existing Review Navigation

**Purpose**: Complete the User Story 2 browser handoff without creating a second proposal surface or authorizing browser-to-agent calls.

- [ ] T050 [P] [US2] Add test-first app-context cases that emit bounded `activeDraftId` only on the exact active-draft route and treat route/prompt hints as non-authoritative in `src/web/src/composables/__tests__/useCoinSearchChat.test.ts`
- [ ] T051 [P] [US2] Add test-first card cases for queued/running/completed/partial/no-match/failed/cancelled/stale/unavailable states, limitations/conflicts/coverage, malformed projections, no apply controls, and fixed `Open Deep Analysis` label in `src/web/src/components/chat/__tests__/CopilotRunProgress.test.ts`
- [ ] T052 [US2] Add test-first navigation cases rejecting absolute, protocol-relative, credential-bearing, non-positive, and job-mismatched URLs while accepting only `/deep-analysis/{positiveJobID}` through Vue Router in `src/web/src/components/chat/__tests__/CopilotRunProgress.test.ts`
- [ ] T053 [P] [US2] Add regression tests for Copilot fallback/cancel/resume/reconnect and independent Deep Analysis report/proposal/retry/cancel/reconnect behavior in `src/web/src/composables/__tests__/useCoinCopilot.test.ts` and `src/web/src/pages/__tests__/DeepAnalysisPage.test.ts`
- [ ] T054 [US2] Add bounded optional `activeDraftId` context only for the active Quick Capture route in `src/web/src/types/agent.ts` and `src/web/src/composables/useCoinSearchChat.ts`
- [ ] T055 [US2] Render the typed compact handoff lifecycle/result card with limitations and a validated router link in `src/web/src/components/chat/CopilotRunProgress.vue`, using existing design tokens and Lucide icons and adding no proposal/edit/apply controls
- [ ] T056 [US2] Preserve the canonical `/deep-analysis/:jobId` route and existing `DeepAnalysisPage` as the sole progress/review/editor surface, adding only navigation regression guards in `src/web/src/router/index.ts` and `src/web/src/pages/__tests__/DeepAnalysisPage.test.ts`

**Checkpoint**: The existing drawer hands off safely to the existing Deep Analysis page on desktop and mobile/PWA; there is no duplicate proposal UI.

---

## Phase 8: Full Gates, Documentation, and Operational Evidence

**Purpose**: Prove the complete blast radius, synchronize contracts, and capture reproducible acceptance evidence without deployment.

- [ ] T057 Run targeted test-first suites from `specs/362-coin-copilot-attribution/quickstart.md` and record commands/results for Go contract/tamper/owner/lifecycle/proposal tests, Python contract/harness/security tests, and Vue handoff tests in `specs/362-coin-copilot-attribution/quickstart-evidence.md`
- [ ] T058 Run architecture and route-set guards in `src/api/architecture_test.go`, `src/api/integration/coin_copilot_seam_test.go`, and `src/agent/tests/test_coin_copilot_architecture.py`; record proof in `specs/362-coin-copilot-attribution/quickstart-evidence.md` that Python remains DB/filesystem/shell/generic-HTTP/write free and only the fixed handoff callback route was added
- [ ] T059 Run regression suites covering Fast Identify, legacy fallback, existing collection/specialist tools, direct Deep Analysis launch/history/status/retry/cancel/review/apply, manual data, additive references, and Numista/Nomisma/OCRE/NGC/RPC boundaries; record results in `specs/362-coin-copilot-attribution/quickstart-evidence.md`
- [ ] T060 Regenerate Swagger/OpenAPI only if a documented public projection or route changed via `task openapi`, then run drift checks and update only `src/api/docs/docs.go`, `src/api/docs/swagger.json`, `src/api/docs/swagger.yaml`, and `docs/openapi.json`
- [ ] T061 [P] Update capability, safety boundary, flags, target rules, lifecycle/replay behavior, existing review-only apply flow, provider attribution, and troubleshooting documentation in `docs/features/coin-copilot.md` and `docs/api-reference.md`
- [ ] T062 Run the full repository Quality Gate from `.specify/memory/constitution.md` and `.github/workflows/ci.yml`: Go build/vet/all tests, Vue lint/type-check/tests/build, Python ruff/all pytest, and OpenAPI drift; append exact results to `specs/362-coin-copilot-attribution/quickstart-evidence.md`
- [ ] T063 Run `CGO_ENABLED=1 go test -race ./...` in `src/api`, then verify the hosted Linux Quality Gate jobs in `.github/workflows/ci.yml` are green; record local and hosted evidence in `specs/362-coin-copilot-attribution/quickstart-evidence.md`
- [ ] T064 Run the security gates defined by `.github/workflows/security-scan.yml`—gitleaks, `govulncheck ./...`, `npm audit --audit-level=high`, `uv run pip-audit`, agent runtime-image pip absence/health smoke test—and the Constitution trivy High/Critical container scan; record results or an explicit environment limitation in `specs/362-coin-copilot-attribution/quickstart-evidence.md`
- [ ] T065 Run browser exploration for owned coin, wishlist coin, active draft, ambiguity, missing/duplicate images, foreign IDs, active/retained/failed/cancelled/stale jobs, replay, fallback, one-field apply, manual preservation, and reference append/dedupe; record screenshots/observations in `specs/362-coin-copilot-attribution/quickstart-evidence.md`
- [ ] T066 Run the narrow-viewport mobile/PWA acceptance walkthrough from `specs/362-coin-copilot-attribution/quickstart.md`, including background/restore of both independent SSE loops, 44 px touch target, keyboard navigation, dark-theme/design-token use, horizontal containment, and absence of duplicate editor; record evidence in `specs/362-coin-copilot-attribution/quickstart-evidence.md`
- [ ] T067 Invoke the `post-major-work-qc-audit` skill after all implementation and gates, resolve every blocking finding in the exact files it identifies, and append the final audit disposition to `specs/362-coin-copilot-attribution/quickstart-evidence.md`

**Checkpoint**: All automated, race, hosted, security, browser, mobile/PWA, documentation, OpenAPI, quickstart, and post-major-work audit gates have reproducible evidence.

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1 (T001–T008)**: Starts immediately. T001 and T002 define fixtures; T003–T005 must fail against them before T006–T008 implement the mirrors.
- **Phase 2 (T009–T015)**: Depends on Phase 1 contract vocabulary. T009–T010 and T013–T014 are failing guards before T011–T012 and T015.
- **Phase 3 / US1 (T016–T025)**: Depends on Phase 2. Tests T016–T019 precede T020–T024; T025 follows the complete Go seam.
- **Phase 4 / US2 backend (T026–T032)**: Depends on US1 job resolution/reuse. Tests T026–T028 precede T029–T031; T032 closes the seam.
- **Phase 5 / US3 (T033–T039)**: Depends on the draft binding from Phase 2 and Deep job creation from US1; it does not depend on Vue. Tests T033–T035 precede T036–T038; T039 is the no-write guard.
- **Phase 6 / US4 (T040–T049)**: Depends on the strict callback contract and Go seam. T040–T043 precede T044–T047; T048–T049 validate races and compatibility.
- **Phase 7 / US2 browser (T050–T056)**: Depends on the validated public projection from Phase 4 and replay behavior from Phase 6. T050–T053 precede T054–T056.
- **Phase 8 (T057–T067)**: Depends on all selected story phases. T060 precedes OpenAPI drift in T062; T062–T066 precede the final QC audit T067.

### User Story Dependencies

- **US1**: First independently deliverable story after foundational work; it is the MVP.
- **US2**: Backend projection depends on US1's owner-scoped job; browser handoff additionally depends on US4 replay-safe delivery.
- **US3**: Depends on US1 job admission and foundational draft binding, but is independently testable through the existing Deep proposal editor without the new card.
- **US4**: Depends on the fixed Go callback, but lifecycle/cancellation/fallback behavior is independently testable at the Python/Go seam without Vue.

### Critical Ordering Anchors

1. Canonical tamper fixtures → failing language contract tests → strict mirrors.
2. Failing migration/order/rollback/ownership tests → nullable column/index/invariant implementation.
3. Failing foreign-ID/kind/image/fingerprint/duplicate tests → resolver and admission implementation.
4. Failing projection/URL/lifecycle tests → persisted bounded projection.
5. Failing manual/reference/destination/no-write tests → draft-bound existing apply bridge.
6. Failing malformed/replay/cancel/fallback tests → Python registration and Go admission serialization.
7. Failing card/navigation/reconnect tests → Vue handoff only.
8. Targeted suites → OpenAPI sync → full/race/security/browser/mobile gates → final QC audit.

---

## Parallel Execution Examples

### Phase 1

```text
In parallel after T001–T002:
- T003 Go fixture drift tests
- T004 Python mirror tests
- T005 TypeScript parser tests
```

### User Story 1

```text
In parallel before implementation:
- T016 target/ownership/image guard tests
- T017 fingerprint tests
- T018 duplicate/concurrency tests
- T019 handler/auth/body-cap tests
```

### User Stories 2 and 3

```text
After US1:
- US2 backend test thread: T026–T028
- US3 review/apply test thread: T033–T035
These touch separate projection and proposal test files; implementation remains ordered behind its own failing tests.
```

### User Story 4 and Vue

```text
After the Go callback exists:
- T040 callback-client contract tests
- T041 bounded orchestration tests
- T043 security/fallback tests
After US2 projection and US4 delivery are stable:
- T050 app-context tests
- T051/T052 card and navigation tests
- T053 reconnect regression tests
```

---

## Implementation Strategy

### MVP First

1. Complete strict contracts/tamper fixtures (Phase 1).
2. Complete migration and immutable boundary guards (Phase 2).
3. Complete and independently validate US1 (Phase 3).
4. Stop and demonstrate exact owner-scoped coin/draft resolution plus one reused/admitted authoritative Deep job; do not present this as full Feature 362 until US2–US4 and gates pass.

### Incremental Delivery

1. **US1**: Safe target resolution and authoritative job handoff.
2. **US2 backend**: Honest persisted bounded result projection and review route.
3. **US3**: Existing review-gated collection/wishlist/draft apply behavior with manual/reference preservation.
4. **US4**: Bounded Python orchestration, deterministic cancellation/replay, and safe fallback.
5. **US2 browser**: Existing drawer card to existing Deep Analysis page, never a second editor.
6. **Full evidence**: OpenAPI as applicable, Quality Gate, race/hosted/security/browser/mobile, documentation, and QC audit.

### Non-Negotiable Regression Boundary

- Keep Fast Identify behavior and entry points unchanged.
- Keep pre-accept legacy chat fallback and default-off/unsupported-model behavior unchanged.
- Keep all current Coin Copilot collection and specialist tools unchanged.
- Keep direct Deep Analysis launch, lifecycle, review, and apply paths authoritative.
- Preserve manual fields and structured references; references remain validated, additive, and deduplicated.
- Keep Python stateless and free of database, filesystem, shell, generic HTTP, provider, and write authority.
- Add no direct agent mutation, generic callback, arbitrary Deep apply route, second proposal model, or duplicate proposal UI.
