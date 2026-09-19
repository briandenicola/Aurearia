# Tasks: Coin Copilot Attribution Integration

**Input**: `specs/362-coin-copilot-attribution/spec.md`, `plan.md`, `research.md`, `data-model.md`, `contracts/coin-copilot-deep-analysis.md`, `quickstart.md`, and accepted `docs/adr/0017-coin-copilot-deep-analysis-handoff.md`
**Branch constraint**: Work only on the existing `beta` branch. Do not create/switch branches, deploy, or modify unrelated dirty `.squad/` files.
**Test-first rule**: Every guard/race/contract task below must fail for the intended missing behavior before its paired implementation begins.
**Evidence rule**: Every configured gate must pass locally or in an equivalent hosted job whose URL/run ID is recorded; an unavailable local tool is not a waiver.

## Format: `[ID] [P?] [Story] Description`

- **[P]** means the task is genuinely independent of other unfinished tasks and changes different files.
- **[Story]** is required only for user-story work.
- Every task names exact files. No task authorizes direct agent mutation, arbitrary callbacks, a second attribution engine/proposal model, or duplicate proposal UI.

---

## Phase 0: Accepted ADR and Pre-Schema Compatibility Interlock

**Purpose**: Establish the only approved mixed-version rollback target before any Feature 362 schema or row-producing work.

**⚠️ BLOCKING GATE**: T001–T009 must complete and pass before T022 or any task that adds `copilot_draft`, `source_draft_id`, `coin_copilot_deep_handoffs`, or Feature 362 rows.

- [x] T001 Record the satisfied prerequisite that `docs/adr/0017-coin-copilot-deep-analysis-handoff.md` already has `Status: Accepted`, pin its accepted commit/SHA, and record the compatibility-release prerequisite in `specs/362-coin-copilot-attribution/quickstart-evidence.md`
- [x] T002 Add pre-schema test-first raw-SQL fixtures proving unknown source values including `copilot_draft` are rejected without coercion, metadata disclosure, worker claim, or mutation by Deep list/get/status/stream/retry/apply/adoption paths in `src/api/handlers/deep_identification_test.go`, `src/api/services/deep_identification_service_test.go`, and `src/api/services/deep_identification_proposal_test.go`
- [x] T003 [P] Add pre-schema test-first worker/repository coverage proving an old compatibility binary recognizes only `intake` and `saved_coin`, leaves unknown-source rows/report/proposal/artifacts byte-for-byte intact, and never publishes events or provider work in `src/api/repository/deep_identification_repository_test.go` and `src/api/services/deep_identification_pipeline_runner_test.go`
- [x] T004 [P] Add test-first settings coverage proving `CoinCopilotAttributionEnabled` exists, defaults `false`, survives settings serialization, and admits zero handoff/provider work in `src/api/services/settings_service_test.go`
- [x] T005 Implement one closed Deep source validator for the compatibility release and apply it before list/get/status/stream/retry/apply/worker adoption in `src/api/models/deep_identification_job.go`, `src/api/handlers/deep_identification.go`, `src/api/services/deep_identification_service.go`, `src/api/services/deep_identification_proposal.go`, and `src/api/repository/deep_identification_repository.go`
- [x] T006 Implement the default-off `CoinCopilotAttributionEnabled` setting without adding a callback, source value, table, column, or Feature 362 row in `src/api/models/appsetting.go` and `src/api/services/settings_service.go`
- [x] T007 Create the executable mixed-binary harness that raw-seeds unknown sources, exercises every guarded read/adopt/apply path, compares preserved rows, and rejects rollback to a binary older than the guard release in `scripts/compat/feature362-rollback.ps1`
- [x] T008 Add `.github/workflows/feature362-compatibility.yml` to build/archive the compatibility-guard binary and run the guard-only half of `scripts/compat/feature362-rollback.ps1`, then require a local or hosted pass and record the artifact digest and passing run URL/ID in `specs/362-coin-copilot-attribution/quickstart-evidence.md`
- [x] T009 Complete the mandatory external/manual Release A checkpoint by deploying the verified compatibility-guard release to the intended environment, recording the deployed guard release/commit and list/get/status/stream/retry/apply/worker-adoption verification evidence in `specs/362-coin-copilot-attribution/quickstart-evidence.md`, and obtaining separate explicit user approval before any schema or `copilot_draft` row work; CI success or image publication alone is insufficient

**Checkpoint**: ADR 0017 is accepted; the retained guard binary fails closed on unknown `copilot_draft`; Release A is externally deployed and manually verified; separate user approval is recorded; the default-off flag admits no work. CI/image publication alone does not satisfy this checkpoint, and schema work may begin only after this checkpoint plus the deterministic tests in Phases 1–2.

---

## Phase 1: Strict Contracts, Authentication, Bounds, and Tamper Fixtures

**Purpose**: Freeze the fixed callback authority and independent 64 KiB/32 KiB limits before route or orchestration implementation.

- [x] T010 Create canonical valid `request`/`status`/`rerun`, lifecycle, eligibility, truncation, `outcome` result-discriminant, and `Authorization: Bearer <InternalTokenService.MintForCopilotExecution token>` fixtures in `specs/362-coin-copilot-attribution/contracts/fixtures/deep-analysis-handoff-valid.json`, plus invalid result-`status` discriminants, user-JWT/internal-service/Deep-job/wrong-tool/wrong-owner/wrong-run/stale-execution/expired/revoked token, unknown-field/enum, forged owner/snapshot/provider/apply, kind/job mismatch, duplicate-call, unsafe-URL, malformed-output, and byte-boundary fixtures in `specs/362-coin-copilot-attribution/contracts/fixtures/deep-analysis-handoff-invalid.json`
- [x] T011 [P] Add test-first Go fixture drift/auth tests proving only the canonical execution token is accepted and request/public event canonical bytes pass at 65,536 and fail at 65,537 in `src/api/handlers/coin_copilot_internal_tools_test.go` and `src/api/services/coin_copilot_contract_test.go`
- [x] T012 [P] Add test-first deterministic projection tests proving the complete canonical result is hashed before omission, persisted bytes never exceed 32,768, stable ordering produces identical bytes/digest/counts, UTF-8/JSON entries are never sliced, required lifecycle/link/limitations remain, and a non-fitting minimal envelope fails closed in `src/api/services/deep_analysis_handoff_projection_test.go`
- [x] T013 [P] Add test-first Python mirror cases requiring `outcome` and rejecting result-level `status` as a discriminant, plus independent 65,536-byte request/public frames, 32,768-byte persisted results, checkpoint truncation disclosure/digest validation, unknown fields, and canonical execution-token forwarding in `src/agent/tests/test_coin_copilot_contract.py`
- [x] T014 [P] Add test-first TypeScript cases requiring `outcome` and rejecting result-level `status` as a discriminant, plus 64 KiB public events, 32 KiB result metadata, deterministic omission disclosure, malformed/tampered payloads, and unsafe/mismatched review URLs in `src/web/src/api/endpoints/__tests__/agent.test.ts`
- [x] T015 Define strict Go request/result/projection models using `outcome` as the sole result discriminant and rejecting a result-level `status` discriminant, plus mutual-field rules, closed vocabularies, finite confidence, URL/duplicate validation, and separate request/public-event/persisted-result limits in `src/api/services/coin_copilot_contract.go` and `src/api/services/deep_analysis_handoff_projection.go`
- [x] T016 [P] Define strict Pydantic model-visible and callback request/result mirrors with `outcome` as the sole result discriminant, result-level `status` forbidden, `extra="forbid"`, injected idempotency/checkpoint fields, independent byte bounds, and truncation metadata in `src/agent/app/models/requests.py` and `src/agent/app/models/responses.py`
- [x] T017 [P] Define strict TypeScript handoff/truncation types with `outcome` as the sole result discriminant and fail-closed public-event parsing that rejects result-level `status` and proposal/apply properties in `src/web/src/types/agent.ts` and `src/web/src/api/endpoints/agent.ts`

**Checkpoint**: Cross-language fixtures agree; execution-token authentication is canonical; 64 KiB request/public event and 32 KiB persisted-result behavior are separate and deterministic.

---

## Phase 2: Deterministic Go Admission, Snapshot, Cancellation, and Idempotency Tests

**Purpose**: Author all row-count/race/idempotency tests before schema or orchestration implementation.

**⚠️ TEST-FIRST GATE**: T018–T021 must fail for the intended absent atomic seam before T022 starts.

- [x] T018 [P] [US1] Add barrier/channel-based test-first snapshot cases for coin deletion, obverse/reverse row or content replacement, notes/context value and version changes, provider configuration generation changes, draft promotion/discard/deletion, owner change, and target kind/id change; assert HTTP 409, zero handoff/job rows, zero worker wakes, and zero provider calls in `src/api/services/deep_analysis_handoff_service_test.go`
- [x] T019 [P] [US4] Add deterministic test-first admission/cancellation cases proving cancel-first creates zero handoff/job rows, admission-first creates exactly one binding/job then requests existing Deep cancellation, simultaneous duplicates create one binding/job, and late report/proposal/event/Copilot settlement is rejected in `src/api/services/coin_copilot_service_test.go` and `src/api/services/deep_identification_concurrency_test.go`
- [x] T020 [P] [US4] Add test-first durable handoff idempotency/crash-replay cases for same key/same binding, changed operation, target kind/id, prior rerun job id, execution, expected checkpoint, stored app-context digest, and server snapshot; require HTTP 409 and exact zero additional rows/wakes/provider calls on every changed binding in `src/api/repository/coin_copilot_repository_test.go`
- [x] T021 [US4] Add Go start/resume API test-first cases proving reuse of the same idempotency key with changed target kind/id, route app context, checkpoint version, current execution, or saved checkpoint returns HTTP 409 without callback/job execution in `src/api/handlers/coin_copilot_test.go` and `src/api/services/coin_copilot_worker_test.go`

**Checkpoint**: Deterministic red tests define the atomicity boundary and all required zero/one row outcomes before schema or row-producing code exists.

---

## Phase 3: User Story 1 — Atomic Target Resolution and Deep Job Admission (Priority: P1) 🎯 MVP

**Goal**: Resolve exactly one owned coin or active draft, snapshot it atomically, and durably reuse/create exactly one authoritative Deep job.

**Independent Test**: Admit an owned saved coin and active draft; replay returns the same durable binding/job, while any ownership, target, face, context, provider-generation, checkpoint, execution, cancellation, or app-context change produces no stale work.

### Schema and Repository Tests

- [x] T022 [P] [US1] Add test-first migration/order/no-backfill/rollback-preservation assertions for `coin_copilot_deep_handoffs`, its unique `(user_id,run_id,handoff_key_hash)` key and owner/job indexes, nullable indexed `source_draft_id`, and migration after the compatibility interlock in `src/api/database/database_test.go`
- [x] T023 [P] [US1] Add test-first model invariant cases requiring `copilot_draft` plus owned non-null `source_draft_id`, requiring null draft binding for `intake`/`saved_coin`, keeping `PriorJobID` distinct from result `DeepJobID`, and rejecting unknown sources in `src/api/models/deep_identification_job_test.go` and `src/api/models/coin_copilot_test.go`
- [x] T024 [P] [US1] Add test-first owner-scoped repository cases for same-key replay, changed-binding conflict, newest eligible retained result, active uniqueness, immutable source binding, artifact-before-commit readiness, staged-file cleanup on rollback, and worker wake only after commit in `src/api/repository/coin_copilot_repository_test.go` and `src/api/repository/deep_identification_repository_test.go`

### Schema and Atomic Implementation

- [x] T025 [US1] Add `CoinCopilotDeepHandoff` with execution/checkpoint/app-context/operation/target/prior-job/snapshot/result-job/outcome fields and closed `DeepJobSourceCopilotDraft` plus `SourceDraftID` invariants in `src/api/models/coin_copilot.go` and `src/api/models/deep_identification_job.go`
- [x] T026 [US1] Add ordered additive migrations and indexes with no backfill or destructive rollback in `src/api/database/database.go`
- [x] T027 [US1] Implement owner/run/key-first durable binding lookup and one linearizable admission transaction that revalidates run/execution/checkpoint/cancellation/flags, target/faces/context/provider generation, artifacts, Deep reuse/create, and handoff insertion in `src/api/repository/coin_copilot_repository.go` and `src/api/repository/deep_identification_repository.go`
- [x] T028 [US1] Implement server-only canonical v2 snapshot generation over target state/version, distinct face row/version/hash, bounded context value/version, and sorted effective providers/configuration generation in `src/api/services/deep_identification_service.go` and `src/api/services/settings_service.go`
- [x] T029 [US1] Implement HTTP-agnostic exact owner-scoped coin/active-draft resolution, missing/duplicate/unsupported-face outcomes, candidate snapshot comparison, active/retained reuse, explicit rerun with separately validated prior job, and post-commit worker publication in `src/api/services/deep_analysis_handoff_service.go`
- [x] T030 [US1] Add strict thin handler authorization/decoding/error mapping and register only `POST /api/internal/copilot/tools/deep_analysis_handoff` with `CoinCopilotExecutionTokenRequired(tokenSvc, "deep_analysis_handoff")` in `src/api/handlers/coin_copilot_internal_tools.go` and `src/api/routes_internal.go`
- [x] T031 [US1] Inject only existing Go repositories/services/settings/images/tokens into the handoff service and add only the literal `deep_analysis_handoff` callback allowlist entry in `src/api/deps.go`, `src/api/services/coin_copilot_contract.go`, and `src/api/services/coin_copilot_proxy.go`
- [x] T032 [US1] Make all Phase 2 barriers green and add seam assertions for clarification/no-call, foreign/unknown equality, target-kind mismatch, active/retained reuse, explicit rerun, artifacts-before-worker, and unchanged Fast Identify/direct Deep entry points in `src/api/integration/coin_copilot_seam_test.go` and `src/api/integration/deep_identification_seam_test.go`

**Checkpoint**: User Story 1 is independently usable; one durable Go binding and one authoritative Deep job survive crash/replay without stale admission or a second pipeline.

---

## Phase 4: User Story 2 — Closed Status Eligibility and Bounded Persisted Explanation (Priority: P1)

**Goal**: Explain only eligible persisted Deep state through a deterministic bounded projection and link to the existing review page.

**Independent Test**: Validated bindings, current owned saved coins, and active owned `copilot_draft` jobs are eligible. Without a prior validated durable owner binding, unknown IDs, foreign jobs, legacy unbound intake, unknown sources, arbitrary unbound jobs, and deleted/promoted target IDs all return exact canonical bytes `{"outcome":"not_eligible","reason":null}`. Only a previously validated durable owner binding whose target later disappears or is promoted returns metadata-free `{"outcome":"target_unavailable","reason":null}`.

### Tests for User Story 2

- [X] T033 [P] [US2] Add test-first closed eligibility cases for validated durable handoff plus matching checkpoint, owned current `saved_coin`, active owned `copilot_draft`, queued/running lifecycle-only results, and retained completed/partial reports with pruned events in `src/api/services/deep_analysis_handoff_service_test.go`
- [X] T034 [P] [US2] Add red tests proving both nondisclosure branches before T037: without a prior validated durable owner binding, unknown ID, foreign job, legacy unbound `intake`, unknown source, arbitrary unbound job, deleted coin ID, and deleted/discarded/promoted draft ID must all return exact canonical bytes `{"outcome":"not_eligible","reason":null}`; only a previously validated durable owner binding whose target later disappears or is promoted may return exact metadata-free bytes `{"outcome":"target_unavailable","reason":null}`; internal causes must never affect either public body in `src/api/handlers/coin_copilot_internal_tools_test.go`
- [X] T035 [P] [US2] Add test-first persisted projection cases for complete/partial/no-match/image-only/low-confidence/conflicts/unresolved/provider limits, invalid/malformed report JSON, safe citation hosts, deterministic 32 KiB omission/digest metadata, and exclusion of raw notes/paths/credentials/acceptance/apply data in `src/api/services/deep_analysis_handoff_projection_test.go`
- [X] T036 [P] [US2] Add URL tests rejecting unsafe schemes, embedded credentials, malformed/unapproved citation hosts, absolute/protocol-relative/non-positive/mismatched review URLs, and invented citation repair in `src/api/services/deep_identification_contract_drift_test.go`

### Implementation for User Story 2

- [X] T037 [US2] Implement the exact eligibility predicate after T033–T036 are red: map every anonymous/unvalidated cause—including unknown ID, foreign job, legacy unbound intake, unknown source, arbitrary unbound job, and deleted/promoted target ID without a prior validated durable owner binding—to exact canonical bytes `{"outcome":"not_eligible","reason":null}`; map only a previously validated durable owner binding whose target later disappears/promotes to metadata-free `{"outcome":"target_unavailable","reason":null}`; keep internal causes server-only and the `status` operation read-only with no handoff row in `src/api/services/deep_analysis_handoff_service.go`
- [X] T038 [US2] Decode only validated persisted Deep report/proposal/coverage data, preserve source confidence/conflicts/no-match/provider attribution, validate citations, and build the complete canonical projection in `src/api/services/deep_analysis_handoff_projection.go`
- [X] T039 [US2] Deterministically reduce whole stable-order entries to 32,768 bytes after hashing the complete result, populate exact truncation byte/count/digest metadata, retain required lifecycle/link/limitations, and fail closed if the minimum cannot fit in `src/api/services/deep_analysis_handoff_projection.go`
- [X] T040 [US2] Enforce the 65,536-byte sanitized public-event envelope independently of the persisted result, expose only canonical `/deep-analysis/{jobId}`, and prevent status from restarting/retrying work in `src/api/services/coin_copilot_service.go` and `src/api/handlers/coin_copilot_internal_tools.go`

**Checkpoint**: User Story 2 uses `outcome`, never result-level `status`, as its discriminant; anonymous/unvalidated causes produce exact canonical `not_eligible` bytes, while only a prior validated owner binding with a later-lost/promoted target produces metadata-free `target_unavailable`; replay facts remain byte-stable with visible omission disclosure.

---

## Phase 5: User Story 3 — Atomic Existing-Destination Apply Matrix (Priority: P1)

**Goal**: Keep conversation read-only and apply only owner-confirmed fields through the existing Deep review UI using the exact destination matrix.

**Independent Test**: For collection, wishlist, and bound active draft, accept one valid scalar, notes, and new/equivalent references; verify exact-field replacement, idempotent notes append, registry-valid additive/deduped references, manual preservation, and all-or-nothing rejection on stale/unsupported state.

### Tests for User Story 3

- [X] T041 [P] [US3] Add table-driven test-first scalar matrix cases: collection/wishlist permit only `denomination`, `ruler`, `era`, `dateRange`, `mint`, `material`, `weightGrams`, `diameterMm`, `obverseInscription`, `reverseInscription`, `obverseDescription`, `reverseDescription`, `coin_type`; bound drafts permit only `workingTitle`, `era`, `dateRange`; each accepted scalar replaces only itself in `src/api/services/deep_identification_proposal_integration_test.go`
- [X] T042 [P] [US3] Add test-first note merge cases for one dated job-id-keyed `Source: Coin Copilot Deep Analysis` block, same-job replay replacement without duplication, new-job append, bounded content, and byte-for-byte preservation of manual text outside the block in `src/api/services/deep_identification_proposal_test.go`
- [X] T043 [P] [US3] Add test-first reference cases for registry validation, case-insensitive equivalence, additive append/dedupe, no replacement/deletion, collection/wishlist behavior, draft proposal staging through `AppliedDraftID`, and transactional promotion merge with the draft's existing selected reference in `src/api/services/deep_identification_proposal_phase6b_test.go` and `src/api/services/quick_capture_service_test.go`
- [X] T044 [US3] Add deterministic transaction test-first cases that re-read owner, destination/source binding, active lifecycle, target/context version, proposal version/selection/applicability, and reference registry/equivalence; any stale or unsupported field returns `409 re_review_required`, rolls back every scalar/note/reference change, and preserves images/acquisition/valuation/storage/privacy/status/relationships in `src/api/services/deep_identification_proposal_integration_test.go`
- [X] T045 [P] [US3] Add architecture tests proving Go/Python/routes/public events/chat expose no `apply`, `accept`, `edit`, `cancel_deep_job`, arbitrary write, provider query, database, filesystem, shell, or generic HTTP operation in `src/api/architecture_test.go` and `src/agent/tests/test_coin_copilot_architecture.py`

### Implementation for User Story 3

- [X] T046 [US3] Route `copilot_draft` proposals only to the exact still-active owner-bound `SourceDraftID`, preserve ordinary `intake` new-draft behavior, and perform all state/version/applicability revalidation in one transaction in `src/api/services/deep_identification_proposal.go`
- [X] T047 [US3] Implement exact-field scalar replacement and idempotent job-keyed notes append for collection, wishlist, and draft destinations without widening existing allowlists in `src/api/services/deep_identification_proposal.go`
- [X] T048 [US3] Implement registry-validated additive case-insensitive reference dedupe for coins, staged accepted draft references linked by `AppliedDraftID`, and atomic promotion merge in `src/api/services/deep_identification_proposal.go` and `src/api/services/quick_capture_service.go`
- [X] T049 [US3] Return `409 re_review_required` with zero partial writes for unsupported fields or changed owner/lifecycle/context/proposal/registry state and preserve all unselected/manual data in `src/api/handlers/deep_identification.go` and `src/api/services/deep_identification_proposal.go`

**Checkpoint**: User Story 3 has no conversational write path; the existing editor applies the exact matrix atomically and replay-idempotently.

---

## Phase 6: User Story 4 — Python Orchestration, Replay, Cancellation, and Fallback (Priority: P1)

**Goal**: Keep Python stateless and bounded while forwarding one fixed side-effect capability and replaying completed facts without re-execution.

**Independent Test**: Duplicate/replayed/changed calls, cancellation at every await, malformed output, unsupported models, disabled capabilities, and late frames produce deterministic reuse/conflict/fallback with no duplicate callback/provider work.

### Tests for User Story 4

- [x] T050 [P] [US4] Add test-first Python callback cases for the fixed route, canonical `Bearer` execution token, harness-injected tool call/idempotency/checkpoint fields, forbidden model-supplied owner/snapshot/provider/apply data, strict 64 KiB request and 32 KiB result metadata, and no arbitrary URL in `src/agent/tests/test_coin_copilot_contract.py`
- [x] T051 [P] [US4] Add test-first harness cases for ambiguity/prompt-context disagreement, correct `request`/read-only `status`/explicit `rerun`, request/rerun executing alone, all existing iteration/tool/concurrency/wall-clock/token/event bounds, and no nested unbounded budget in `src/agent/tests/test_coin_copilot_harness.py`
- [x] T052 [US4] Add test-first replay/cancellation cases for completed-checkpoint reconstruction without callback, duplicate completed-call digest, changed binding conflict, cancellation before/after every await, admission-first Deep cancel, late frame rejection, deterministic truncation disclosure, and failed/cancelled/stale results in `src/agent/tests/test_coin_copilot_harness.py`
- [x] T053 [P] [US4] Add test-first security/fallback cases for unknown/malformed/oversized/injection/token-shaped output, unsafe URLs, unavailable Deep/attribution capability, default-off Copilot, unsupported model, and pre-accept legacy fallback in `src/agent/tests/test_coin_copilot_security.py` and `src/agent/tests/test_coin_copilot_capabilities.py`

### Implementation for User Story 4

- [x] T054 [US4] Register exactly `deep_analysis_handoff` in `COPILOT_ALLOWED_TOOLS`, `CALLBACK_TOOLS`, `ARG_MODELS`, and `RESULT_MODELS`, inject execution/idempotency/checkpoint data, and reuse strict digest/sanitization/completed-call dedupe in `src/agent/app/tools/copilot_collection_tools.py`
- [x] T055 [US4] Add the one typed tool definition and system policy for exact-target clarification, status/reuse, explicit rerun, persisted-result/truncation honesty, and no conversational apply/write in `src/agent/app/teams/coin_copilot.py`
- [x] T056 [US4] Execute request/rerun alone, preserve all current budgets and cancellation checks, replay completed checkpoint facts without another Go call, and reject late settlement in `src/agent/app/teams/coin_copilot.py`
- [x] T057 [US4] Linearize Copilot cancel with admission, request Deep cancellation for an admission-first nonterminal bound job, and make late Copilot/provider/report/proposal/event settlement lose in `src/api/services/coin_copilot_service.go`, `src/api/services/coin_copilot_worker.go`, and `src/api/services/deep_analysis_handoff_service.go`
- [x] T058 [US4] Preserve existing four collection callbacks, specialist tools, feature flags, unsupported-model handling, and pre-accept legacy fallback in `src/agent/tests/test_coin_copilot_specialists.py`, `src/agent/tests/test_coin_copilot_capabilities.py`, and `src/api/services/coin_copilot_proxy_test.go`

**Checkpoint**: User Story 4 is stateless in Python, crash/replay safe in Go, bounded at both envelope levels, and cancellation/fallback safe.

---

## Phase 7: Vue Handoff Card and Existing Review Navigation

**Purpose**: Complete User Story 2 navigation without adding proposal mutation or duplicate UI.

- [x] T059 [P] [US2] Add test-first app-context cases emitting bounded `activeDraftId` only on the exact active draft route while keeping all prompt/route hints non-authoritative in `src/web/src/composables/__tests__/useCoinSearchChat.test.ts`
- [x] T060 [P] [US2] Add test-first card cases for every closed outcome/status, truncation/omitted-count disclosure, conflicts/coverage/limitations, malformed payloads, no apply controls, and fixed `Open Deep Analysis` text in `src/web/src/components/chat/__tests__/CopilotRunProgress.test.ts`
- [x] T061 [US2] Add test-first router cases accepting only a positive job-matching `/deep-analysis/{jobId}` and rejecting absolute, protocol-relative, credential-bearing, non-positive, or mismatched URLs in `src/web/src/components/chat/__tests__/CopilotRunProgress.test.ts`
- [x] T062 [P] [US2] Add regression tests for Copilot fallback/cancel/resume/reconnect and independent Deep report/proposal/retry/cancel/reconnect behavior in `src/web/src/composables/__tests__/useCoinCopilot.test.ts` and `src/web/src/pages/__tests__/DeepAnalysisPage.test.ts`
- [x] T063 [US2] Add optional bounded `activeDraftId` only for the active Quick Capture route in `src/web/src/types/agent.ts` and `src/web/src/composables/useCoinSearchChat.ts`
- [x] T064 [US2] Render the typed bounded handoff card, omission disclosure, and validated Vue Router link with existing tokens/Lucide controls and no editor/apply UI in `src/web/src/components/chat/CopilotRunProgress.vue`
- [x] T065 [US2] Preserve `/deep-analysis/:jobId` and `DeepAnalysisPage.vue` as the sole progress/review/editor/apply surface with navigation/reconnect guards in `src/web/src/router/index.ts` and `src/web/src/pages/__tests__/DeepAnalysisPage.test.ts`

**Checkpoint**: Desktop and mobile/PWA users reach the exact existing Deep review page; no browser-to-agent call or duplicate proposal UI exists.

---

## Phase 8: Finish-Existing Flags and Mixed-Version Rollback

**Purpose**: Prove disabling features blocks only new admission/rerun and that the guard release remains a safe rollback target.

- [x] T066 [P] [US4] Add test-first transition cases for disabling `CoinCopilotAttributionEnabled`, `CoinCopilotEnabled`, model capability, or `DeepIdentificationEnabled` before admission; assert no handoff/job/provider work and typed unavailable reason in `src/api/services/deep_analysis_handoff_service_test.go` and `src/agent/tests/test_coin_copilot_capabilities.py`
- [x] T067 [US4] Add test-first `finish_existing` cases for each flag disabled after durable acceptance: worker completion remains allowed; owner status/events/cancel/report/proposal review/edit/confirmed existing-page apply remain available; all new handoffs/reruns are blocked in `src/api/integration/coin_copilot_seam_test.go` and `src/api/services/deep_identification_proposal_integration_test.go`
- [x] T068 [US4] Implement live pre-admission checks for all four gates while exempting already accepted owner reads/cancel/review/edit/apply and worker settlement in `src/api/services/deep_analysis_handoff_service.go`, `src/api/services/deep_identification_service.go`, and `src/api/services/deep_identification_proposal.go`
- [x] T069 Complete `scripts/compat/feature362-rollback.ps1` to migrate a copied database, create/settle saved-coin and `copilot_draft` handoffs, disable/drain/cancel, boot the guard binary, prove fail-closed adoption/apply with byte-preserved rows, then re-upgrade and restore status/review/apply
- [x] T070 Run the complete mixed-binary matrix locally or in `.github/workflows/feature362-compatibility.yml`; require a pass for pre-feature known sources, unknown/`copilot_draft` guard rejection, default-off migration, row preservation, and re-upgrade restoration, and record the passing URL/ID in `specs/362-coin-copilot-attribution/quickstart-evidence.md`

**Checkpoint**: No gate waiver exists; rollback never targets an unguarded binary, and accepted work remains usable under finish-existing semantics.

---

## Phase 9: Full Quality, Security, Browser, Documentation, and PR Gates

**Purpose**: Produce complete local-or-equivalent-hosted evidence for the whole repository and Feature 362 blast radius.

- [x] T071 Run targeted Go contract/tamper/auth/owner/snapshot/idempotency/status/projection/apply/cancellation/finish-existing suites and record exact passing commands/results in `specs/362-coin-copilot-attribution/quickstart-evidence.md`
- [x] T072 Run architecture and exact route-set guards proving Python remains database/filesystem/shell/generic-HTTP/provider/write free, Vue calls Go only, and the callback allowlist expands solely by `POST /api/internal/copilot/tools/deep_analysis_handoff` rather than wildcard or Deep apply routes; record the pass in `specs/362-coin-copilot-attribution/quickstart-evidence.md`
- [x] T073 Run regressions for Fast Identify, legacy fallback, all existing Coin Copilot collection/specialist tools, direct Deep intake/saved-coin history/events/cancel/retry/review/apply, manual data/additive references, and Numista/Nomisma/OCRE/NGC/RPC boundaries; record the pass in `specs/362-coin-copilot-attribution/quickstart-evidence.md`
- [x] T074 Regenerate and verify Swagger/OpenAPI only if the documented route/public projection changes by running `task openapi`, updating only `src/api/docs/docs.go`, `src/api/docs/swagger.json`, `src/api/docs/swagger.yaml`, and `docs/openapi.json`, and record the drift-check pass in `specs/362-coin-copilot-attribution/quickstart-evidence.md`
- [x] T075 [P] Update Feature 362 capability, accepted ADR/compatibility sequence, exact auth/bounds/status/apply matrix, finish-existing behavior, rollback, provider limits, and troubleshooting documentation in `docs/features/coin-copilot.md` and `docs/api-reference.md`
- [x] T076 Run the complete Python gate in `src/agent`: `pip install -e ".[dev]"`, `uv sync --extra dev`, `uv run python -m compileall -q app tests`, `uv run ruff check app tests`, targeted contract/harness tests, and `uv run pytest tests -v`; require local passes or equivalent hosted passes and record each in `specs/362-coin-copilot-attribution/quickstart-evidence.md`
- [x] T077 Run the complete web gate in `src/web`: `npm ci`, `npm run lint`, `npx vue-tsc --build`, `npm run test -- --run`, and `npm run build`; require local passes or equivalent hosted passes and record each in `specs/362-coin-copilot-attribution/quickstart-evidence.md`
- [x] T078 Run the complete Go/hosted gate in `src/api`: `go build ./...`, `go vet ./...`, `go test -run TestArchitecture ./...`, `go test ./...`, targeted integration/rollback tests, and `CGO_ENABLED=1 go test -race ./...`; require local passes or equivalent Linux Quality Gate passes and record each in `specs/362-coin-copilot-attribution/quickstart-evidence.md`
- [x] T079 Run all configured security gates from `.github/workflows/security-scan.yml` plus Constitution requirements: gitleaks, `govulncheck ./...`, `npm audit --audit-level=high`, `uv run pip-audit`, agent image no-pip/health smoke, and Trivy container scan with zero High/Critical findings; require local or equivalent hosted passes and record them in `specs/362-coin-copilot-attribution/quickstart-evidence.md`
- [x] T080 Run browser exploration for collection/wishlist/draft admission, every conflict/eligibility/lifecycle/fallback case, deterministic truncation disclosure, cancel races, finish-existing transitions, exact scalar/note/reference apply matrix, manual preservation, and no conversational write; record reproducible observations in `specs/362-coin-copilot-attribution/quickstart-evidence.md`
- [x] T081 Run narrow-viewport mobile/PWA acceptance for both independent SSE resumptions, background/restore, keyboard operation, 44 px touch targets, dark-theme/design-token layout, safe review navigation, truncation disclosure, horizontal containment, and absence of duplicate editor; record the pass in `specs/362-coin-copilot-attribution/quickstart-evidence.md`
- [x] T082 Verify every applicable hosted PR check passes—including Quality Gate, Go race, compatibility matrix, Security Scan, CodeQL, container image build/scan, and SBOM/provenance checks—and record check names and URLs/IDs in `specs/362-coin-copilot-attribution/quickstart-evidence.md`
- [x] T083 Complete the PR description from `.github/pull_request_template.md` with the Constitution §17/§21 Definition of Done and workflow-contract self-check, citing ADR 0017, Principles II/III/IV/V/VIII/IX, rollback artifact retention, test-first evidence, security evidence, and exact affected workflows without modifying the template
- [x] T084 Invoke the `post-major-work-qc-audit` skill after T070–T083 pass, resolve all blocking findings in the exact implementation/docs/test files identified by the audit, rerun affected gates, and append the final passing disposition to `specs/362-coin-copilot-attribution/quickstart-evidence.md`

**Checkpoint**: Every configured gate has a local or equivalent hosted pass; no environment limitation is accepted as evidence.

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 0 (T001–T009)**: Absolute prerequisite. T001 records the already accepted ADR; T002–T004 fail first; T005–T006 implement only the compatibility guard/default-off flag; T007–T008 produce and verify the retained guard artifact; T009 requires external Release A deployment, manual verification, and separate user approval. No Feature 362 schema/rows may precede T009, and CI/image publication does not satisfy T009.
- **Phase 1 (T010–T017)**: May begin after the Phase 0 external approval checkpoint but cannot register a route or write rows. Canonical fixtures precede language implementations.
- **Phase 2 (T018–T021)**: Depends on the accepted ADR and contract vocabulary; all deterministic Go tests must be red before Phase 3 schema/orchestration starts.
- **Phase 3 / US1 (T022–T032)**: Depends on T009, T015–T017, and all red tests T018–T021. Schema tests precede models/migrations; repositories precede services; callback registration is last.
- **Phase 4 / US2 (T033–T040)**: Depends on US1's durable binding/job seam. Eligibility/projection tests precede implementation.
- **Phase 5 / US3 (T041–T049)**: Depends on `copilot_draft` binding and the existing proposal editor, not on Vue. All matrix/transaction tests precede merge implementation.
- **Phase 6 / US4 (T050–T058)**: Depends on the fixed Go callback and bounded projection. Python tests precede registration/orchestration.
- **Phase 7 / Vue (T059–T065)**: Depends on US2 projection and US4 delivery/replay.
- **Phase 8 (T066–T070)**: Depends on all backend lifecycle/apply work; transition tests precede finish-existing implementation and the complete rollback matrix.
- **Phase 9 (T071–T084)**: Depends on all selected feature phases. OpenAPI/docs precede full gates; all configured gates and PR evidence precede the final QC audit.

### User Story Dependencies

- **US1**: First independently deliverable story after compatibility, contracts, and deterministic red tests; it is the MVP.
- **US2**: Depends on US1 durable bindings; Vue delivery additionally depends on US4 checkpoint replay.
- **US3**: Depends on US1 `copilot_draft` binding but is independently testable through the existing Deep review UI.
- **US4**: Depends on the fixed Go callback; its cancellation/idempotency red tests intentionally precede US1 implementation.

### Critical Test-First Anchors

1. Already accepted ADR → unknown-source/default-off red tests → compatibility guard → retained guard artifact → external Release A deployment/verification → separate user approval.
2. 64 KiB request/public event and 32 KiB deterministic result tests → strict cross-language contracts.
3. Snapshot/cancel/idempotency/start-resume red tests → schema → atomic repository/service implementation → callback registration.
4. Closed eligibility/projection/URL red tests → status and deterministic persisted projection.
5. Exact scalar/note/reference/state-revalidation red tests → atomic destination merge.
6. Python malformed/replay/cancel/fallback red tests → serial tool registration/orchestration.
7. Vue context/card/navigation/reconnect red tests → existing-drawer handoff.
8. Flag-transition red tests → finish-existing implementation → mixed-binary rollback.
9. Targeted regressions → OpenAPI/docs → full/race/security/browser/mobile/hosted/PR gates → QC audit.

---

## Genuine Parallel Opportunities

```text
After T001:
- T002 unknown-source handler/service tests
- T003 worker/repository compatibility tests
- T004 default-off settings tests

After canonical fixtures T010:
- T011 Go auth/64 KiB tests
- T012 Go 32 KiB projection tests
- T013 Python contract tests
- T014 TypeScript event tests

Before schema, after Phase 1:
- T018 snapshot mutation tests
- T019 admission/cancellation tests
- T020 durable idempotency tests

After US1:
- T033–T036 US2 eligibility/projection test files
- T041–T043 US3 matrix test files

After the Go callback:
- T050 Python callback contract tests
- T051 Python harness policy/budget tests
- T053 Python security/capability tests

After bounded public delivery:
- T059 app-context tests
- T060 card rendering tests
- T062 reconnect regressions
```

---

## Implementation Strategy

### Compatibility-First MVP

1. Use and pin the already accepted ADR 0017.
2. Ship/test/archive the source-validation compatibility guard and default-off flag with no Feature 362 schema/rows.
3. Externally deploy and manually verify Release A, record its release/commit and evidence, and obtain separate user approval; CI/image publication alone is insufficient.
4. Author deterministic contract/snapshot/admission/cancellation/idempotency tests.
5. Add schema and implement US1 atomic admission.
6. Stop and independently verify one owned coin and one active draft produce one durable binding/job; do not enable or present full Feature 362 until US2–US4, rollback, and all gates pass.

### Non-Negotiable Boundaries

- Fast Identify, direct Deep Analysis, legacy fallback, existing collection/specialist tools, and provider/license behavior remain unchanged.
- Python remains stateless and database/filesystem/shell/generic-HTTP/provider/write free.
- Conversation/status/link/replay cannot apply or mutate.
- Existing Deep review UI remains the sole proposal editor/apply surface.
- Collection/wishlist/draft manual data and prior references remain preserved; reference behavior is validated, additive, and case-insensitively deduplicated.
- Rollback never targets the unguarded pre-feature binary and never drops/backfills Feature 362 rows.
- No configured quality, security, compatibility, race, browser, or PR gate may be waived.
