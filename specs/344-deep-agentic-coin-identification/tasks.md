# Tasks: Deep Agentic Coin Identification

**Branch**: `344-deep-agentic-coin-identification`
**Input**: `plan.md`, `spec.md`, `research.md`, `data-model.md`, `quickstart.md`,
`contracts/deep-identification.openapi.yaml`, `contracts/sse-events.md`,
`contracts/agent-internal-contract.md` (all in
`specs/344-deep-agentic-coin-identification/`)

**Tests**: Included. `plan.md`'s Testing Strategy and the user-supplied
context explicitly require deterministic Go/Python/Vue coverage, race tests,
and a manual (non-substituting) quickstart pass.

**Organization**: Phases 1–7 are shared, cross-cutting foundation (no user
story can reach a terminal state, and therefore no story's Independent Test
can pass, without the full job/event/artifact/pipeline substrate). Phases
8–13 are the six spec user stories (US1–US6), each independently testable on
top of that foundation. Phase 14 is polish/hardening/docs/rollout. A final,
explicitly deferred section lists the two later-provider gates that MUST NOT
be implemented as part of this MVP.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no unresolved dependency)
- **[Story]**: US1–US6 map to spec.md priorities P1 (US1–US4) and P2 (US5–US6)
- Every task names an exact existing or new file path from `plan.md`'s
  Project Structure section

---

## Post-BLOCK Remediation (Cassius, MVP re-review)

The reviewed MVP was BLOCKed. Remediation addressed the following on the
existing branch (see `.squad/decisions/inbox/344-remediation-proposal-contract-and-followups.md`).
No post-MVP task (T125+) is marked complete by this work.

- **B1 — Proposal contract mismatch (RE-DONE, supersedes Phase 7 decision #8)**:
  `deep_identification_pipeline_runner.go` now builds the rich
  `deepProposalDocument` (per-field proposed/confidence/evidence/ownerEdited/
  ownerValue/accepted) instead of the flat `{fields}` map, reusing the existing
  proposal DTOs. Evidence/citations resolved Go-side from `provider_result`
  frames; field + citation-host allowlists enforced at translation and apply.
  New integration regression (`deep_identification_proposal_integration_test.go`,
  saved-coin + new-intake) fails under the old flat shape. Reopens/validates the
  US4 confirm-and-apply path (T100-range) that the flat shape had silently broken.
- **B2 — Retry unreachable (FIXED)**: Retry wired into `DeepAnalysisPage.vue`
  for eligible terminal states with navigation + safe re-init, duplicate-click
  guard, and accessible loading/error. Component tests added.
- **B3 — Emoji violation (FIXED)**: microscope emoji in
  `DeepAnalysisEntryButton.vue` replaced with the lucide `Microscope` icon; test added.
- **F1 — Feature-flag entry UX (FIXED)**: added `GET /deep-identification/capability`
  (typed, authenticated, boolean-only) + `useDeepIdentificationCapability`
  (fails closed); entry button gated in `CoinLookupPage.vue` + `CoinActionsPanel.vue`.
- **F3 — recursion_limit (WIRED, inert in streaming driver)**: `build_graph`
  binds `bounds.recursion_limit` via `.with_config`; topology test asserts it.
- **F4 — Typed error taxonomy (FIXED)**: Python `error` frame now emits `code`
  (contract §3) via a narrow classifier; SSE tests updated.
- **F5 — Design tokens / 44px targets (FIXED)**: no new hardcoded colors; 44px
  min touch targets on DeepProposalEditor inputs/decision/apply controls and the
  new retry control; design-token tests pass.

---

## Phase 1: Setup

**Purpose**: Confirm environment/config prerequisites before any domain code lands.

- [X] T001 Verify and record that no new third-party dependency is introduced (diff `go.mod`, `src/agent/requirements.txt`, `src/web/package.json` — all unchanged per plan.md Technical Context)
- [X] T002 [P] Add `AGENT_DEEP_MAX_CONCURRENCY`, `AGENT_DEEP_MAX_PROVIDERS`, `AGENT_DEEP_PROVIDER_TIMEOUT`, `AGENT_DEEP_TOTAL_TIMEOUT`, `AGENT_DEEP_RECURSION_LIMIT` to `src/agent/app/config.py` and `.env.example` with the bounds from `contracts/agent-internal-contract.md` §2
- [X] T003 [P] Add a `DeepJobArtifactDir(jobID)` path helper (`<UploadDir>/deep-jobs/job-<id>/`) to `src/api/services/image_service.go` — path convention only, no behavior change to any existing function

---

## Phase 2: Foundational — Go data layer (models, migration, repository, settings)

**⚠️ CRITICAL**: Blocks every user story phase. No coin-write, no job, no event exists before this completes.

- [X] T004 [P] Create `src/api/models/deep_identification_job.go` (`DeepIdentificationJob` struct + `DeepJobStatus` type/constants per data-model.md §2)
- [X] T005 [P] Create `src/api/models/deep_identification_event.go` (`DeepIdentificationEvent` struct per data-model.md §3)
- [X] T006 [P] Create `src/api/models/deep_identification_provider_run.go` (`DeepIdentificationProviderRun` struct per data-model.md §4)
- [X] T007 [P] Create `src/api/models/deep_identification_artifact.go` (`DeepIdentificationArtifact` struct per data-model.md §5)
- [X] T008 Register all four new models in the existing `AutoMigrate(...)` call in `src/api/database/database.go:36` (additive only; no existing column altered/dropped)
- [X] T009 Add indexes in `src/api/database/database.go`: `uix_deep_jobs_active_fingerprint`, `idx_deep_jobs_user_status_created`, `idx_deep_jobs_user_coin`, `idx_deep_jobs_status_heartbeat`, `idx_deep_jobs_expires`, `uix_deep_events_job_seq`, `idx_deep_events_created`, `uix_deep_provider_run_job_provider`, `uix_deep_artifact_job_role` (data-model.md §2.4/§3/§4/§5)
- [X] T010 [P] Add `TestDeepIdentificationModelsAutoMigrate` to `src/api/database/migration_test.go` (follows `TestQuickCaptureModelsAutoMigrate`), asserting additive migration succeeds and no existing table/column changes
- [X] T011 Add `SettingDeepIdentification{Enabled,WorkerCount,MaxActivePerUser,QueueDepth,HardTimeoutSeconds,EventRetentionHours,ResultRetentionDays,MaxProviders,NumistaCallBudget,OCREEnabled,RPCEnabled}` keys + defaults to `src/api/services/settings_service.go` per data-model.md §8
- [X] T012 [P] Add settings_service_test.go cases verifying each new key's default value and admin read/write round-trip
- [X] T013 Create `src/api/repository/deep_identification_repository.go`: `CreateJob` + `FindActiveByFingerprint` (idempotent reuse, FR-007)
- [X] T014 Add `ClaimNextQueuedJob` (dequeue + stamp `WorkerID`/`HeartbeatAt`) to `src/api/repository/deep_identification_repository.go`
- [X] T015 Add `Heartbeat(jobID)` to `src/api/repository/deep_identification_repository.go`
- [X] T016 Add `AppendEvent(jobID, type, payload)` to `src/api/repository/deep_identification_repository.go` — same-transaction `last_seq` increment + insert (data-model.md §3), returns the assigned `seq`
- [X] T017 Add `ListEventsSince(jobID, since)` and `PruneEventsBefore(cutoff)` to `src/api/repository/deep_identification_repository.go`
- [X] T018 Add `SettleTerminal(jobID, expectedStatuses, newStatus, report, proposal, failureCode)` to `src/api/repository/deep_identification_repository.go` — single conditional `UPDATE ... WHERE status IN (...)` plus terminal event append in **one transaction** (data-model.md §2.1); returns whether this call won the race
- [X] T019 Add `RecoverStaleJobs(staleAfter)` to `src/api/repository/deep_identification_repository.go` — flips `running` jobs with a stale `HeartbeatAt` to `failed:stale_restart`, appending exactly one terminal event
- [X] T020 Add owner-scoped `GetJob(id, userID)`, `ListJobs(userID, filters)`, `RequestCancel(jobID, userID)` to `src/api/repository/deep_identification_repository.go` — every query filtered by `user_id` (FR-006/FR-037)
- [X] T021 [P] Create `src/api/repository/deep_identification_repository_test.go`: owner scoping (cross-user access ⇒ not found), idempotent duplicate-submit returns the in-flight job, **concurrent `AppendEvent` uniqueness** (parallel goroutines assert exactly one row per `(job_id, seq)`, no gaps/dupes — SC-003), **terminal-settle race** (parallel cancel + complete via `SettleTerminal` ⇒ exactly one winner and exactly one terminal event, run many iterations to rule out flakiness — FR-019), stale recovery flips `running`→`failed:stale_restart`, event pruning sets `EventsPrunedAt` while preserving the report

**Checkpoint**: Job/event persistence exists, is owner-scoped, idempotent, and race-safe at the repository layer.

---

## Phase 3: Foundational — Artifact storage (obverse/reverse/hint)

**⚠️ CRITICAL**: Blocks US1/US2/US6.

- [X] T022 Create `src/api/services/deep_identification_service.go` skeleton (`DeepIdentificationService` struct, constructor DI per `main.go:246-249` pattern) — no worker logic yet
- [X] T023 Add `ValidateAndSaveArtifact(jobID, role, fileHeader)` to `src/api/services/deep_identification_service.go`, reusing `services.ValidateImageData`/`NormalizeImageExt`/`MaxImageUploadBytes` (FR-005); enforces ≤1 `obverse`, ≤1 `reverse`, ≤3 `hint` (data-model.md §5)
- [X] T024 Add `ReuseSavedCoinImage(jobID, coinID, role)` to `src/api/services/deep_identification_service.go` — creates an artifact row with `Origin=saved_coin_image`, empty `FilePath`, fingerprint derived from stored file path+size+mtime (data-model.md §2.3)
- [X] T025 Add `DeleteHintArtifacts(jobID)` and `DeleteJobArtifacts(jobID)` to `src/api/services/deep_identification_service.go` — stamps `DeletedAt`, tolerant of an already-missing file, used by every terminal path and the retention janitor
- [X] T026 Add `ComputeInputFingerprint(userID, coinID, images, notes, requestedProviders)` to `src/api/services/deep_identification_service.go` per the sha256 formula in data-model.md §2.3
- [X] T027 [P] Create `src/api/services/deep_identification_service_test.go` covering: validation rejects disallowed type/magic-byte mismatch/oversize/hint-count>3/missing obverse-or-reverse; saved-image reuse creates the correct artifact row; fingerprint is stable for identical inputs and changes when a saved image's stored size/mtime changes (retry-after-change edge case)
- [X] T028 [P] Add orphan-artifact recovery cases to `src/api/services/deep_identification_service_test.go`: `DeleteJobArtifacts` is idempotent when called twice; a simulated already-missing file does not error

**Checkpoint**: Image validation/storage/reuse/deletion exists independent of the worker/pipeline.

---

## Phase 4: Foundational — Job service core (worker pool, cancel, timeout, janitor)

**⚠️ CRITICAL**: Blocks every user story phase.

- [X] T029 Add worker pool to `src/api/services/deep_identification_service.go`: `StartWorkers(ctx)` bounded by `SettingDeepIdentificationWorkerCount`; each worker loop: `ClaimNextQueuedJob` → heartbeat ticker → run pipeline → `SettleTerminal`
- [X] T030 Add per-user active-job limit (`SettingDeepIdentificationMaxActivePerUser`) to `StartJob(...)` in `src/api/services/deep_identification_service.go` — returns the existing active job instead of enqueuing a second (FR-007)
- [X] T031 Add queue-depth backpressure (`SettingDeepIdentificationQueueDepth`) to `StartJob(...)` — returns a typed `queue_full` error mapped to `503` when exceeded
- [X] T032 Add an in-memory cancel registry (`map[jobID]context.CancelFunc`) to `src/api/services/deep_identification_service.go`; `RequestCancel` cancels the running job's context and records `CancelRequestedAt` via the repository
- [X] T033 Add hard-timeout enforcement (`SettingDeepIdentificationHardTimeoutSeconds`) wrapping each job's pipeline run in `context.WithTimeout` — expiry settles `partial`/`failed`, never leaves a job `running` (FR-014)
- [X] T034 Add `StartJanitor(ctx)` to `src/api/services/deep_identification_service.go`: on boot and every 60s calls `RecoverStaleJobs`; hourly calls `PruneEventsBefore` (24h post-terminal, FR-017) and expires/deletes job rows + artifacts past `ExpiresAt` (90d, FR-034)
- [X] T035 Wire `StartWorkers` + `StartJanitor` into `src/api/main.go` following the existing scheduler-wiring pattern
- [X] T036 [P] Add worker-pool bound test to `src/api/services/deep_identification_service_test.go`: a fake pipeline runner + N goroutines assert max concurrent claimed jobs never exceeds the configured worker count; queue backpressure returns `queue_full` once depth+1 is reached; per-user limit returns the existing job, not a new one
- [X] T037 [P] Add hard-timeout test to `src/api/services/deep_identification_service_test.go`: a fake pipeline that never returns settles to `failed`/`partial` before the test deadline, exactly once
- [X] T038 [P] Add **cancel-vs-complete concurrency test** to `src/api/services/deep_identification_service_test.go`: a goroutine requesting cancel and a goroutine completing the same job race repeatedly (many iterations) — asserts exactly one terminal state and exactly one terminal event every time, with no flaky double-settle (FR-019)
- [X] T039 [P] Add restart-recovery test to `src/api/services/deep_identification_service_test.go`: jobs left `running` with a stale heartbeat from a simulated prior process instance settle to `failed:stale_restart` and never remain `running` (FR-012)
- [X] T040 [P] Add hint-cleanup-on-restart test to `src/api/services/deep_identification_service_test.go`: hint artifacts with no `DeletedAt` after a simulated crash are swept and deleted by the janitor's startup sweep, not only by the terminal-hook path

**Checkpoint**: A job can be created, claimed, cancelled, retried-in-theory, bounded, and safely recovered from restart — via poll (`GET`) only, no pipeline output yet.

---

## Phase 5: Foundational — REST handlers (create/get/list/cancel/retry) + OpenAPI

**⚠️ CRITICAL**: Blocks every user story phase.

- [X] T041 Create `src/api/handlers/deep_identification.go`: `POST /deep-identification/jobs` (multipart create) with swaggo annotations, behind `writeRateLimit` + `AuthRequiredWithSecurity` + feature-flag check (`403` when `SettingDeepIdentificationEnabled=false`)
- [X] T042 Add `GET /deep-identification/jobs` (owner-scoped list, `coinId`/`status` filters) to `src/api/handlers/deep_identification.go`
- [X] T043 Add `GET /deep-identification/jobs/{id}` (job + report + proposal; `404` for non-owner) to `src/api/handlers/deep_identification.go`
- [X] T044 Add `POST /deep-identification/jobs/{id}/cancel` to `src/api/handlers/deep_identification.go`
- [X] T045 Add `POST /deep-identification/jobs/{id}/retry` to `src/api/handlers/deep_identification.go` — creates a new job with `RetryOfJobID` set and `RetryDepth+1` capped at 3, reusing prior inputs unless new ones are supplied, never mutating the original (FR-020)
- [X] T046 Wire the five routes + DI into `src/api/main.go` (repo→service→handler pattern at `main.go:246-249`)
- [X] T047 [P] Create `src/api/handlers/deep_identification_test.go`: multipart validation (type/magic-byte/size/count/missing role ⇒ `422 missing_obverse`/`missing_reverse`/`hint_image_in_coin_role`); auth/ownership `404`s for cross-user job ids; feature-flag-off returns `403` on new start but does not block `GET` of already-running jobs (FR-008)
- [X] T048 [P] Add retry-lineage test to `src/api/handlers/deep_identification_test.go`: retry of a terminal job creates a new row with `RetryOfJobID` set; a 4th retry beyond depth 3 is rejected; the original job's events/report remain untouched
- [X] T049 Run `task openapi`, commit the regenerated `docs/openapi.json` for the five endpoints added so far, and confirm `route_openapi_drift_test.go` stays green
- [X] T050 [P] Add a fast-path regression test asserting `POST /api/coins/lookup`'s response shape is unchanged and that **zero** `DeepIdentificationJob` rows are created by it (FR-001/SC-008)

**Checkpoint**: Foundational Phases 2–5 complete — jobs can be created, claimed, cancelled, retried, and settled terminal via poll, with owner scoping, idempotency, backpressure, limits, and races proven safe. No pipeline output yet (`ReportJSON`/`ProposalJSON` empty) and no UI.

---

## Phase 6: Foundational — Provider tool boundary (Go)

**⚠️ CRITICAL**: Blocks the Python pipeline (Phase 7) and therefore every user story.

- [X] T051 Add `numista_search`/`numista_detail`/`nomisma_search` internal tool endpoints to `src/api/handlers/internal_tools.go` per `contracts/agent-internal-contract.md` §7, reusing `src/api/services/numista_client.go` and `src/api/services/nomisma_client.go`
- [X] T052 Add per-job call-budget enforcement (`SettingDeepIdentificationNumistaCallBudget`; Nomisma ≤3/job) keyed off the minted internal token's job binding in `src/api/handlers/internal_tools.go` — returns `quota_limited` status rather than an error
- [X] T053 Extend `src/api/services/internal_token_service.go` with `Mint(userID, jobID)` job-scoped binding used to authorize/limit tool calls
- [X] T054 [P] Add `httptest`-backed cases to `src/api/handlers/internal_tools_test.go`: budget enforcement returns `quota_limited` after N calls; internal-token middleware rejects unauthenticated or foreign-job calls; reuses the existing `numista_client_test.go`/`nomisma_client_test.go` fake-transport patterns, no live HTTP

**Checkpoint**: Go-side provider tool boundary exists, budgeted and owner/job-scoped, ready for Python to call.

---

## Phase 7: Foundational — Python inference pipeline + Go↔Python streaming

**⚠️ CRITICAL**: Blocks every user story phase (no job can reach a terminal state with real evidence without this).

- [X] T055 [P] Add `DeepIdentifyRequest` family (`StrictRequestModel`, `extra="forbid"`) to `src/agent/app/models/requests.py` per `contracts/agent-internal-contract.md` §2
- [X] T056 [P] Add `ProviderEvidence`, `DeepSynthesis` response models to `src/agent/app/models/responses.py` per §4–5
- [X] T057 Create `src/agent/app/teams/deep_identification/state.py` (`DeepIdentificationState` TypedDict with a list reducer for evidence)
- [X] T058 Create `src/agent/app/teams/deep_identification/router.py` — single LLM call constrained to `provider_catalog` entries; honors `provider_override`; `automatable: false` entries short-circuit to a `not_automated`/`unavailable` evidence row without a call
- [X] T059 Create `src/agent/app/tools/provider_tools.py` — Go internal tool client mirroring `app/tools/collection_tools.py`, calling only the endpoints from T051
- [X] T060 Create `src/agent/app/teams/deep_identification/providers/nomisma.py` and `providers/numista.py` — automated provider nodes that call `provider_tools.py` only, never the upstream directly
- [X] T061 Create `src/agent/app/teams/deep_identification/providers/ngc.py` — reuses existing OCR cert-number extraction, emits `not_automated` + `link_out`, no live NGC API call (Terms-of-Use prohibited)
- [X] T062 Create `src/agent/app/teams/deep_identification/providers/ocre.py` and `providers/rpc.py` — **MVP-only typed stubs** emitting `not_automated` (OCRE) / `unavailable` (RPC) respectively; explicitly NO upstream client, NO SPARQL query, NO scraping (real adapters are out of MVP scope — see "Later Provider Gates" below)
- [X] T063 Create `src/agent/app/teams/deep_identification/merge.py` — deterministic claim ordering `(field, provider_rank, -confidence, citation)` per contract §5
- [X] T064 Add citation host-allowlist validation to `src/agent/app/teams/deep_identification/merge.py` (or a new `validators.py`): allowed hosts `en.numista.com`/`api.numista.com`, `nomisma.org`, `numismatics.org`, `www.ngccoin.com`, `rpc.ashmus.ox.ac.uk`; claims failing validation are dropped and counted as `invalid_response` (SC-006)
- [X] T065 Create `src/agent/app/teams/deep_identification/evaluator.py` — contradiction/provenance node; surfaces both claims for a disagreement, never silently resolves by precedence (FR-027)
- [X] T066 Create `src/agent/app/teams/deep_identification/synthesis.py` — strict-JSON typed final output (`DeepSynthesis`); field allowlisting itself is enforced Go-side (Phase 11)
- [X] T067 Create `src/agent/app/teams/deep_identification/graph.py` — wires `prepare_evidence` (vision) → `router` → bounded provider fan-out (`asyncio.Semaphore` ≤ `max_concurrency`) → `evaluator` → `synthesizer`; `asyncio.gather(..., return_exceptions=True)` for partial-failure tolerance; `config={"recursion_limit": bounds.recursion_limit}`
- [X] T068 Add `POST /api/deep-identify/stream` route to `src/agent/app/routes.py`, streaming via the existing `app/streaming.py::format_sse`, wrapping the full run in `asyncio.wait_for(total_timeout_s)`
- [X] T069 Add total-timeout partial-synthesis fallback to `src/agent/app/teams/deep_identification/graph.py`: on expiry, synthesize from evidence gathered so far with `partial_success: true`, or emit a typed `error` frame if nothing was gathered
- [X] T070 Add `StreamDeepIdentification` to `src/api/services/agent_proxy.go` — authenticated Go→Python POST, consumes the Python SSE stream, translates each frame into a persisted `DeepIdentificationEvent` (per `contracts/sse-events.md` §2) inside the job worker loop
- [X] T071 Add cancellation propagation in `src/api/services/agent_proxy.go`: the worker's cancelled context (Phase 4 T032) cancels the outbound HTTP request context; any partial evidence already received for a cancelled job is discarded (contract §3, FR-018/FR-019)
- [X] T072 Add EOF-without-terminal-frame handling in `src/api/services/agent_proxy.go`: a stream that ends without a `synthesis` or `error` frame settles the job as `failed:agent_unavailable`
- [X] T073 [P] Add router tests to `src/agent/tests/` (path mirrors `app/teams/deep_identification/`): the router selects only from the supplied catalog, honors `provider_override`, and never calls an `automatable: false` provider (fake tool callables, no network)
- [X] T074 [P] Add fan-out tests to `src/agent/tests/`: respects `max_concurrency`/`max_providers`; a per-provider timeout yields `timed_out`; one failing provider still reaches synthesis (partial success)
- [X] T075 [P] Add merge/citation tests to `src/agent/tests/`: `merge.py` produces deterministic ordering for identical inputs; citation host validation drops off-allowlist URLs; `not_automated`/`unavailable` providers never emit `no_match`
- [X] T076 [P] Add SSE envelope tests to `src/agent/tests/` using the existing `FakeGraph`/`collect_sse` harness: terminal-frame invariants hold; sanitized text; no token-shaped strings leak
- [X] T077 [P] Add cases to `src/api/services/agent_proxy_test.go`: each Python frame type translates to the correct persisted event type; cancellation propagates (context cancel closes the outbound HTTP request); EOF-without-terminal ⇒ `agent_unavailable`
- [X] T078 [P] Run `ruff check app/ tests/` clean for `src/agent/app/teams/deep_identification/` and `app/tools/provider_tools.py` (lint gate)

**Checkpoint**: A job created via `POST` (Phase 5) can now run end-to-end through the real pipeline — router → bounded fan-out over Nomisma/Numista (automated) + NGC/OCRE/RPC (typed non-automated/unavailable) → evaluator → synthesis — and settle terminal with `ReportJSON`/`ProposalJSON` populated. This is the last shared prerequisite. Every user story phase below adds its own entry point, control surface, or UI on top of this common pipeline.

---

## Phase 8: User Story 1 - Start Deep Analysis from new intake (Priority: P1) 🎯 MVP entry point

**Goal**: A collector can upload obverse/reverse (+ optional notes/hints) from new-coin intake and explicitly opt into Deep Analysis without affecting the fast lookup.

**Independent Test**: Upload obverse+reverse from new intake, start Deep Analysis, verify a job id returns immediately, the fast lookup (if also requested) is unaffected, and the job/result are later retrievable.

- [X] T079 [P] [US1] Create `DeepAnalysisEntryButton.vue` (shared CTA) in `src/web/src/components/deep-identification/`
- [X] T080 [US1] Add a secondary "Deep Analysis" CTA to `src/web/src/pages/CoinLookupPage.vue`, wired to the new-intake images/notes without touching the existing fast-lookup call path
- [X] T081 [US1] Create `DeepAnalysisStartPanel.vue` in `src/web/src/components/deep-identification/` (obverse/reverse/hint upload, notes, provider-override checklist) for the new-intake case
- [X] T082 [US1] Add `createDeepIdentificationJob` client wrapper + types to `src/web/src/api/client.ts` and `src/web/src/types/index.ts`
- [X] T083 [US1] Add route `/deep-analysis/:jobId?` to `src/web/src/router/index.ts` and create a thin `src/web/src/pages/DeepAnalysisPage.vue` shell (fetches job on mount; child components stubbed until later phases)
- [X] T084 [US1] Create `src/web/src/composables/useDeepIdentification.ts` (job lifecycle: start/get/list)
- [X] T085 [P] [US1] Vitest: `DeepAnalysisStartPanel` validation — missing obverse/reverse blocks submit with a specific message; hint-count cap enforced client-side (server remains authoritative)
- [X] T086 [P] [US1] Vitest: fast-lookup path on `CoinLookupPage.vue` is unaffected when the Deep Analysis CTA is not used (no extra network calls, no job created)
- [X] T087 [US1] Add a Playwright workflow spec in `src/web/e2e/workflows/`: upload obverse+reverse from new intake, click Deep Analysis, verify a job id returns and the page navigates to `/deep-analysis/:jobId`

**Checkpoint**: US1 independently testable end-to-end (new-intake entry, job accepted, fast path unaffected).

---

## Phase 9: User Story 2 - Start Deep Analysis from an existing saved coin (Priority: P1)

**Goal**: A collector can re-analyze a saved coin using its existing images without re-uploading, with no silent write.

**Independent Test**: Start Deep Analysis from a saved coin with both faces present, verify no new uploads are required and the coin record stays byte-identical until an explicit Apply; verify cross-user access is rejected.

- [X] T088 [US2] Add a "Deep Analysis" CTA to `src/web/src/components/coin/CoinActionsPanel.vue` passing `coinId` (no re-upload)
- [X] T089 [US2] Extend `DeepAnalysisStartPanel.vue` to support saved-coin mode: reuse the coin's existing obverse/reverse images, requiring upload only for a missing role (FR-003)
- [X] T090 [US2] Enforce in `src/api/services/deep_identification_service.go` (extends T023) that a saved-coin start missing a required image with no upload supplied is rejected `422` — no silent hint/absent substitution
- [X] T091 [P] [US2] Add a handler test to `src/api/handlers/deep_identification_test.go`: a saved coin owned by another user ⇒ `404` on create/get/cancel/retry (FR-006)
- [X] T092 [P] [US2] Add a service test to `src/api/services/deep_identification_service_test.go`: saved coin missing one required image and no upload supplied ⇒ `422`, no job row created
- [X] T093 [P] [US2] Vitest: saved-coin start panel shows only the missing role's upload control and submits existing coin-image ids for present role(s)
- [X] T094 [US2] Add a Playwright workflow spec: start Deep Analysis from a saved-coin detail page with both faces present; verify no `PATCH`/`PUT` to the coin occurs during start or run (write occurs only later, in US4's Apply)

**Checkpoint**: US2 independently testable (saved-coin entry, ownership enforcement, no silent write during run).

---

## Phase 10: User Story 3 - Observe progress, reconnect, cancel, and retry (Priority: P1)

**Goal**: A collector can watch streamed progress, safely disconnect/reconnect without gaps/duplicates, cancel a running job, and retry a terminal job.

**Independent Test**: Start a job, observe streamed events, disconnect/reconnect mid-run and verify exactly the missed events arrive once in order, cancel a running job and verify it settles cleanly, retry a terminal job and verify retry lineage.

- [X] T095 [US3] Create `src/api/services/deep_identification_broker.go` — in-process SSE fan-out (subscribe/publish per `jobID`), integrating with the cancel registry from T032
- [X] T096 [US3] Add `GET /deep-identification/jobs/{id}/events` SSE handler to `src/api/handlers/deep_identification.go`: replay-from-storage via `ListEventsSince`, subscribe to the broker for the live tail, `since`/`Last-Event-ID` support, `: ping` every 15s, `stream_truncated` frame when `since` predates retention, `terminal`+`event: end` close, `410` past retention, `404` for non-owner, `429` past 3 concurrent streams/owner (`contracts/sse-events.md`)
- [X] T097 [US3] Wire the cancel (T044) and retry (T045) handlers to the broker so connected clients see the resulting terminal event live
- [X] T098 [US3] Create `src/web/src/composables/useDeepIdentificationStream.ts` — fetch+`ReadableStream` reader (mirrors `client.ts::agentChatStream`), resumes via `?since=`, de-dupes by `seq`, handles `stream_truncated`/`terminal`/`end` frames, does not auto-reconnect after `end`
- [X] T099 [US3] Create `DeepAnalysisProgressTimeline.vue` in `src/web/src/components/deep-identification/` (event timeline, connection/reconnect indicator, cancel button)
- [X] T100 [US3] Wire `useDeepIdentification.ts` `cancel()`/`retry()` to their endpoints and update local state/route to the retried job id
- [X] T101 [US3] Add resume-on-mount behavior to `DeepAnalysisPage.vue`: on navigation/reload, reconnect the stream from the job's last-seen `seq` (stored client-side, keyed by `jobId`)
- [X] T102 [P] [US3] Add SSE handler tests to `src/api/handlers/deep_identification_test.go`: first connect replays all retained events then follows live; reconnect with `since=N` replays exactly `(N,lastSeq]` with no dup/gap; reconnect after `since` < earliest emits `stream_truncated` then retained tail then live; terminal job replay ends with `terminal`+`end`; `410` past result retention; `404` non-owner; `429` on the 4th concurrent stream
- [X] T103 [P] [US3] Add an integration-level **cancel-vs-complete broker race test**: N goroutines racing cancel vs. natural completion through the broker+repository path assert exactly one terminal event is ever broadcast to subscribers (extends T038)
- [X] T104 [P] [US3] Add an integration-level **event-sequence uniqueness under load test**: parallel simulated `provider_result` frames assert unique, gap-free sequence numbers end-to-end through the worker/broker path (extends T021)
- [X] T105 [P] [US3] Add a restart-recovery SSE test: a job left `running` from a killed process instance is discovered by `RecoverStaleJobs` and its replay correctly reports `failed:stale_restart` with a single terminal event, no client left waiting forever
- [X] T106 [P] [US3] Vitest: `useDeepIdentificationStream` — resume with `since`, duplicate suppression, truncation handling, terminal close, reconnect indicator (fake `ReadableStream` harness, no network)
- [X] T107 [P] [US3] Vitest: `DeepAnalysisProgressTimeline` renders `provider_started`/`provider_result`/`evaluation`/`progress`/`terminal` events in order; cancel button is keyboard-accessible and disabled once terminal
- [X] T108 [US3] Add a Playwright workflow spec: start job → observe progress → simulate disconnect/reconnect mid-run → verify no duplicate/missing events → cancel a still-running job → verify terminal cancelled state surfaced

**Checkpoint**: US3 independently testable (streaming, resumable replay, cancel/retry, restart safety, and races all proven).

---

## Phase 11: User Story 4 - Review synthesized report and confirm-gated draft (Priority: P1)

**Goal**: A collector reviews a narrative report with citations and a structured, editable, confirm-gated proposal; nothing is written to a coin without explicit confirmation.

**Independent Test**: Complete a run (including one with partial provider failures), verify the report/proposal contents, edit a field, confirm, and verify the write happens only through the existing Quick Capture/coin-edit paths.

- [X] T109 [US4] Create `src/api/services/deep_identification_proposal.go` — field allowlist (only fields writable via `CoinService.UpdateCoinWithFields`/`QuickCaptureDraft` payloads), owner-edit tracking, apply routing (data-model.md §7)
- [X] T110 [US4] Add `PATCH /deep-identification/jobs/{id}/proposal` to `src/api/handlers/deep_identification.go` — owner edits `ProposalJSON` fields (`ownerEdited`/`ownerValue`/`accepted`); job stays terminal; coin untouched
- [X] T111 [US4] Add `POST /deep-identification/jobs/{id}/apply` to `src/api/handlers/deep_identification.go` — intake path seeds/patches a `QuickCaptureDraft` (existing promote flow) via `deep_identification_proposal.go`; saved-coin path calls `CoinService.UpdateCoinWithFields(source="deep_identification")`; sets `AppliedCoinID`/`AppliedDraftID`/`AppliedAt`; rejects already-applied (`409 already_applied`) and a deleted source coin (`409 source_coin_missing`) per data-model.md §7.1
- [X] T112 [US4] Wire the proposal/apply routes into `src/api/main.go` behind `writeRateLimit`
- [X] T113 [P] [US4] Add a test to `src/api/services/deep_identification_proposal_test.go`: the field allowlist rejects any field not writable via `CoinService`/`QuickCaptureDraft` (no silent new write surface)
- [X] T114 [P] [US4] Add a test proving apply routes exclusively through `QuickCaptureService`/`CoinService` fakes — no direct coin write exists anywhere in the deep-identification package (Principle IV / FR-033)
- [X] T115 [P] [US4] Add a test: apply on a coin deleted mid-run returns `409 source_coin_missing`; the report remains readable
- [X] T116 [P] [US4] Add a test: a second apply attempt returns `409 already_applied` unless a fresh report cycle exists
- [X] T117 [P] [US4] Add a test: every terminal `completed`/`partial` job has `narrative`/`coverage`/`disagreements`/`unresolvedQuestions` populated per data-model.md §6, and `partial` jobs set `PartialSuccess=true`
- [X] T118 [US4] Add `getDeepIdentificationJob`/`patchProposal`/`applyProposal` client wrappers + types to `src/web/src/api/client.ts` and `src/web/src/types/index.ts`
- [X] T119 [US4] Create `DeepReportPanel.vue` in `src/web/src/components/deep-identification/` (narrative, disagreements, citations, coverage summary)
- [X] T120 [US4] Create `DeepProposalEditor.vue` in `src/web/src/components/deep-identification/` (per-field accept/edit, AI-vs-owner visual distinction, confirm action)
- [X] T121 [US4] Wire `DeepAnalysisPage.vue` to render `DeepReportPanel`+`DeepProposalEditor` once the job is terminal, calling apply only on explicit confirm
- [X] T122 [P] [US4] Vitest: `DeepProposalEditor` — per-field accept/reject, owner edit visibly distinct from the AI value, confirm disabled until at least one explicit decision is made, nothing written without confirm
- [X] T123 [P] [US4] Vitest: `DeepReportPanel` renders the partial-success banner and disagreements without hiding any provider status
- [X] T124 [US4] Add a Playwright workflow spec: complete a run with one provider forced to fail → verify partial banner, editable proposal, apply creates/updates only after explicit confirm (both new-intake→draft and saved-coin→update variants)

**Checkpoint**: US4 independently testable (report/draft rendering, confirm-gated apply through existing write paths only, no silent writes, traceability via `AppliedAt`).

---

## Phase 12: User Story 5 - Provider routing, override, and partial-success transparency (Priority: P2)

**Goal**: The router proposes a relevant, bounded provider set; the owner can adjust it; the report never fabricates or hides provider participation.

**Independent Test**: Start a job, verify the proposed provider set is evidence-based, adjust it before the run, force one provider to fail/be not-automated, and verify accurate, non-fabricated coverage reporting including surfaced disagreements.

- [X] T125 [US5] Confirm/extend `POST /deep-identification/jobs` (T041) to echo `RequestedProviders`/`SelectedProviders`/`RouterRationale` in the create response so the owner-adjusted set is reviewable before dispatch (FR-022/FR-023)
- [X] T126 [US5] Add a provider add/remove control to `DeepAnalysisStartPanel.vue` (checklist sourced from `provider_catalog` eligibility) so the owner can adjust the proposed set before the job runs
- [X] T127 [US5] Create `DeepProviderCoverageList.vue` in `src/web/src/components/deep-identification/` — renders `contributed`/`no_match`/`failed`/`timed_out`/`not_automated`/`unavailable` distinctly, with attribution string and link-out for non-automated providers, never as a generic "no result"
- [X] T128 [US5] Wire `DeepProviderCoverageList.vue` into `DeepAnalysisPage.vue` alongside the progress timeline (live during run) and the report panel (final coverage)
- [X] T129 [P] [US5] Add a Go test asserting the `router_selected` event payload reflects an evidence-based rationale, not a fixed always-all-providers set
- [X] T130 [P] [US5] Add a Go/Python test: `provider_override` always includes the named provider even if the router would not have selected it; deselecting a provider results in zero calls to it (FR-023)
- [X] T131 [P] [US5] Add a test: a failing/timed-out provider is listed with its exact status and never blended into another provider's confidence or reclassified as `no_match` (FR-025)
- [X] T132 [P] [US5] Add a test: two providers returning conflicting claims for the same field both appear in `disagreements` with `resolution: "unresolved"`, never silently overridden (FR-027)
- [X] T133 [P] [US5] Vitest: `DeepProviderCoverageList` renders `not_automated`/`unavailable` with link-out distinctly from `no_match`/`failed`, keyboard/a11y accessible

**Checkpoint**: US5 independently testable (router transparency, owner override honored, no fabricated results, disagreements surfaced).

---

## Phase 13: User Story 6 - Ephemeral hint-image privacy and cleanup (Priority: P2)

**Goal**: Hint/reference images are used only for the run's duration and are deleted on every terminal outcome, restart, and retention expiry, and never leak into the report or coin image set.

**Independent Test**: Run jobs with hint images to each terminal state in turn (completed, partial, failed, cancelled) plus a simulated crash/restart and a retention-expiry sweep, and verify hint files are deleted and unretrievable in every case.

- [X] T134 [US6] Add an integration test (Go, e.g. `src/api/services/deep_identification_hint_cleanup_test.go`): a full-stack run with hint images through a **completed** terminal state — assert hint files are deleted from `<UploadDir>/deep-jobs/job-<id>/` and unretrievable via any coin-image endpoint, `DeletedAt` stamped, row retained for audit (SC-004)
- [X] T135 [P] [US6] Add the identical assertion for a job reaching **partial**
- [X] T136 [P] [US6] Add the identical assertion for a job reaching **failed** (including a hard-timeout-triggered failure)
- [X] T137 [P] [US6] Add the identical assertion for a job reaching **cancelled** (including the cancel-vs-complete race's cancelled winner)
- [X] T138 [P] [US6] Add the identical assertion for a job whose result has passed `ExpiresAt` (retention janitor sweep) — both hint and coin-face artifacts removed per data-model.md §9
- [X] T139 [P] [US6] Add a simulated-crash-then-restart test: hint artifacts with no `DeletedAt` after a killed process, once the job settles via `RecoverStaleJobs` on the new instance, are swept and deleted by the janitor's startup pass — verified via filesystem check, not only the DB flag
- [X] T140 [US6] Add a Go test that the persisted `ReportJSON`/`ProposalJSON` never embeds or links a hint artifact's file path or binary (FR-030 negative assertion — string-search the persisted JSON for any hint `FilePath`)
- [X] T141 [P] [US6] Vitest: hint images never appear in any coin image gallery/endpoint-consuming component after any Deep Analysis run

**Checkpoint**: US6 independently testable (hint ephemerality proven across every terminal outcome plus crash/restart and retention expiry, with no report leakage).

---

## Phase 14: Polish & Cross-Cutting Concerns

**Purpose**: Hardening, observability, documentation, rollout readiness, and final gate verification.

- [X] T142 [P] Add a mobile-viewport + a11y pass across all `src/web/src/components/deep-identification/*.vue` components (labels, focus order, keyboard-accessible cancel/apply, touch targets), reusing `InlineCameraCapturePanel`/`QuickCaptureImageSlots` patterns
- [X] T143 [P] Add observability counters/gauges to the existing logger/admin surface: jobs by terminal status, partial-success rate, p50/p95 duration, per-provider status+latency, active SSE streams, reconnect/truncation counts, queue depth, hint-deletion success/failure, janitor-sweep counts — with no sensitive notes/queries logged (FR-036)
- [X] T144 [P] Add admin/settings UI visibility for the `SettingDeepIdentification*` keys in the existing admin settings screen
- [X] T145 Write `docs/adr/0011-deep-agentic-coin-identification.md` (Nygard format; 0010 was already assigned): sibling job/event domain vs `AIJob`; Go-persisted/served SSE with bounded retention; provider HTTP calls staying in Go; provider staging/licensing posture (NGC no-automation, OCRE ODbL gate, RPC blocked, Numista attribution/caching)
- [X] T146 [P] Update `docs/ARCHITECTURE.md` with background-jobs/AI-integration/SSE sections for this feature
- [X] T147 [P] Update `docs/features/ai-analysis.md` (fast vs deep) and `docs/quick-capture.md` (deep proposal seeding a draft)
- [X] T148 [P] Update `docs/testing.md` with the deterministic agent seams used by this feature
- [X] T149 Regenerate `docs/openapi.json` via `task openapi` for the full endpoint set (create/list/get/cancel/retry/events/proposal/apply) and confirm `route_openapi_drift_test.go` is green
- [X] T150 Add a cross-cutting decision note to `.squad/decisions/inbox/` per §18.2 (never edit `.squad/decisions.md` directly) summarizing ADR 0011 and the phase rollout plan
- [X] T151 Verify rollout sequencing: `SettingDeepIdentificationEnabled` defaults to `false`; confirm this stays true through this PR (flag flips happen out-of-band per the plan's rollout table)
- [X] T152 Run the full gate suite: `go build ./...`, `go vet ./...`, `go test ./...` (incl. `TestArchitecture`), `ruff check app/ tests/`, `pytest`, `npm run type-check`, `npm run test`, `npm run build`, `task openapi`, `task test` — all green (§17/§21 gate)
- [ ] T153 Perform the manual end-to-end `quickstart.md` verification (all six walkthroughs: US1 intake, US2 saved coin, US3 resume/cancel/retry, US4 report/draft, US5 provider transparency, US6 hint privacy) — this is a **supplementary confirmation only** and does not substitute for the automated tests in Phases 2–13
- [X] T154 Prepare post-major-work QC/review readiness (repository `post-major-work-qc-audit` skill): confirm engineering best practices, security, docs, architecture, test coverage, supply chain, UX, and operational readiness are all addressed before requesting review

---

## Later Provider Gates (explicitly OUT OF SCOPE for this MVP)

These are represented in the job/event/report contracts today (typed
`not_automated`/`unavailable` statuses, T062) but MUST NOT be implemented as
production adapters in this feature. Do not create tasks that assume these
gates pass; each is its own future release with its own ADR amendment.

- [ ] T155 [DEFERRED] Gate G-OCRE: record ODbL/attribution license review; build an OCRE adapter over the Nomisma SPARQL endpoint; flip `SettingDeepIdentificationOCREEnabled` — **not part of this MVP**; requires separate authorization
- [ ] T156 [DEFERRED] Gate G-RPC: requires written permission or a documented public API from the RPC Online project; until granted, RPC stays `unavailable` and no adapter is written — **not part of this MVP**; blocked pending external action

---

## Dependencies & Execution Order

### Phase dependencies

- **Setup (Phase 1)** → no dependencies
- **Foundational (Phases 2–7)** → depend on Setup; each foundational phase depends on the previous one in this order: Go data layer (2) → artifact storage (3) → job service core (4) → REST handlers (5) → provider tool boundary (6) → Python pipeline + streaming (7). **BLOCKS all user stories.**
- **User stories (Phases 8–13)** → all depend on Phase 7 completing. US1/US2 can proceed in parallel with each other; US3/US4 depend on US1 or US2 having produced a startable job but not on each other; US5/US6 build on the pipeline and artifact layers directly and can proceed in parallel with US1–US4 once Phase 7 is done.
- **Polish (Phase 14)** → depends on all six user stories being complete.
- **Later Provider Gates** → independent releases, blocked on external legal/API validation; not sequenced into the MVP critical path.

### User story dependencies

- **US1 (P1)**: needs Phases 2–7 only.
- **US2 (P1)**: needs Phases 2–7 only; independently testable in parallel with US1 (different entry point, shares the same start-panel component file — coordinate T081/T089 sequentially if worked by different people).
- **US3 (P1)**: needs Phases 2–7 and a startable job (US1 or US2) to exercise against, but its own SSE/cancel/retry code is independent of which entry point started the job.
- **US4 (P1)**: needs Phases 2–7 and a terminal job (any entry point) to review; independent of US3's streaming code.
- **US5 (P2)**: needs Phases 2–7; independent of US1–US4's UI, touches the same start panel (T126) and adds a new component.
- **US6 (P2)**: needs Phases 2–7 (artifact/janitor code already exists); adds verification tests only, no new production code paths beyond what Phase 3/4 built.

### Within each phase

- Models before repository; repository before service; service before handler; handler before UI.
- Tests marked `[P]` in a phase may be written alongside implementation but must fail first if done strictly TDD; per plan.md this feature includes tests as a required deliverable, not merely optional.

---

## Parallel Execution Groups

```text
# Phase 2 (after T004-T007 land): 
T004, T005, T006, T007            → parallel (different model files)
T010, T012, T021                  → parallel (different test files) once T004-T020 land

# Phase 4 races (all extend/observe the same service file but assert independent behaviors):
T036, T037, T038, T039, T040      → parallel once T029-T035 land

# Phase 5:
T047, T048, T050                  → parallel (different/independent test files)

# Phase 7 (Python side, independent modules):
T055, T056                        → parallel
T073, T074, T075, T076            → parallel (independent test modules)
T077, T078                        → parallel with the Python test group (different language/toolchain)

# Phase 8 (US1):
T079, T085, T086                  → parallel

# Phase 10 (US3) verification:
T102, T103, T104, T105, T106, T107 → parallel (independent test files/assertions)

# Phase 11 (US4) verification:
T113, T114, T115, T116, T117      → parallel
T122, T123                        → parallel

# Phase 12 (US5) verification:
T129, T130, T131, T132, T133      → parallel

# Phase 13 (US6) — nearly all parallel (independent terminal-state fixtures):
T135, T136, T137, T138, T139, T141 → parallel (T134 first establishes the fixture pattern; T140 is a distinct assertion)

# Phase 14:
T142, T143, T144, T146, T147, T148 → parallel (independent doc/observability files)
```

---

## Implementation Strategy

### MVP scope

Phases 1–7 (foundation) + Phases 8–11 (US1–US4, all P1) constitute the
smallest deployable, fully confirm-gated MVP: a collector can start Deep
Analysis from either entry point, watch it stream with resumable
reconnect/cancel/retry, and review/edit/confirm a cited report+proposal that
writes only through existing paths. Phases 12–13 (US5/US6, P2) harden
transparency and privacy guarantees and should ship in the same release
train but are independently deferrable without breaking the P1 slice.
Phase 14 is required before general rollout (flag flip). The two later
provider gates (T155/T156) are excluded from this MVP entirely.

### Critical path

Setup → Phase 2 (data layer) → Phase 3 (artifacts) → Phase 4 (job service
core) → Phase 5 (REST handlers) → Phase 6 (provider tool boundary) →
Phase 7 (Python pipeline + streaming) → **first point at which any user
story's Independent Test can pass** → US1/US2 (parallelizable) → US3/US4
(each needs a startable/terminal job from US1 or US2, but not from each
other) → US5/US6 (parallelizable with US3/US4 once Phase 7 is done) →
Phase 14.

### Incremental delivery

1. Phases 1–7 → foundation ready, flag stays off, nothing user-visible yet.
2. Add US1 → test independently → dogfood-deploy (flag on for admin only).
3. Add US2 → test independently.
4. Add US3 → test independently (now safe to leave a job running unattended).
5. Add US4 → test independently — **MVP complete here**.
6. Add US5, US6 → test independently.
7. Phase 14 → full gate suite, docs, ADR, rollout to general availability.

---

## Summary

- **Total tasks**: 156 (T001–T154 active work + T155–T156 explicitly deferred/out-of-scope)
- **Phase breakdown**:
  - Phase 1 Setup: 3 tasks
  - Phase 2 Foundational (Go data layer): 18 tasks (T004–T021)
  - Phase 3 Foundational (artifact storage): 7 tasks (T022–T028)
  - Phase 4 Foundational (job service core): 12 tasks (T029–T040)
  - Phase 5 Foundational (REST handlers/OpenAPI): 10 tasks (T041–T050)
  - Phase 6 Foundational (provider tool boundary): 4 tasks (T051–T054)
  - Phase 7 Foundational (Python pipeline + streaming): 24 tasks (T055–T078)
  - Phase 8 US1 (new intake): 9 tasks (T079–T087)
  - Phase 9 US2 (saved coin): 7 tasks (T088–T094)
  - Phase 10 US3 (progress/reconnect/cancel/retry): 14 tasks (T095–T108)
  - Phase 11 US4 (report/proposal/confirm-apply): 16 tasks (T109–T124)
  - Phase 12 US5 (provider routing/override/transparency): 9 tasks (T125–T133)
  - Phase 13 US6 (hint-image privacy/cleanup): 8 tasks (T134–T141)
  - Phase 14 Polish/hardening/docs/rollout: 13 tasks (T142–T154)
  - Later Provider Gates (deferred, not MVP): 2 tasks (T155–T156)
- **Parallel groups**: see "Parallel Execution Groups" above — largest are
  Phase 7's Python test suite (T073–T076, 4 tasks) and Phase 13's per-terminal-
  state hint-cleanup fixtures (T135–T139/T141, 6 tasks).
- **Critical path**: Setup → Go data layer → artifact storage → job service
  core → REST handlers → provider tool boundary → Python pipeline/streaming →
  (first user story becomes testable) → US1/US2 → US3/US4 → US5/US6 → Polish.
- **MVP cutoff**: end of Phase 11 (US4) — Phases 1–11 (T001–T124) ship a fully
  deployable, confirm-gated Deep Analysis feature covering both entry points,
  resumable streaming, cancel/retry, and report/proposal review-and-apply.
  Phases 12–13 (US5/US6) hardened transparency/privacy guarantees are
  strongly recommended in the same release but are independently deferrable.
- **Later-provider gates excluded from MVP**: G-OCRE (T155) and G-RPC (T156)
  — both require external legal/API validation not obtainable within this
  feature and must not be implemented now; the job/event/report contracts
  already accommodate them without a future respecification.
- **Files changed** (new unless marked CHANGED, per plan.md Project Structure):
  - Go models: `src/api/models/deep_identification_{job,event,provider_run,artifact}.go`
  - Go data/migration: `src/api/database/database.go` (CHANGED), `src/api/database/migration_test.go` (CHANGED)
  - Go repository: `src/api/repository/deep_identification_repository.go` (+ `_test.go`)
  - Go services: `src/api/services/deep_identification_service.go`, `deep_identification_broker.go`, `deep_identification_proposal.go` (+ `_test.go` each); `agent_proxy.go` (CHANGED), `image_service.go` (CHANGED), `settings_service.go` (CHANGED), `internal_token_service.go` (CHANGED)
  - Go handlers: `src/api/handlers/deep_identification.go` (+ `_test.go`), `internal_tools.go` (CHANGED, + `_test.go` additions)
  - Go wiring: `src/api/main.go` (CHANGED)
  - Python: `src/agent/app/routes.py` (CHANGED), `app/models/requests.py` (CHANGED), `app/models/responses.py` (CHANGED), `app/config.py` (CHANGED), `app/tools/provider_tools.py`, `app/teams/deep_identification/{graph,state,router,evaluator,synthesis,merge}.py`, `app/teams/deep_identification/providers/{nomisma,numista,ngc,ocre,rpc}.py`, plus `src/agent/tests/` additions
  - Vue: `src/web/src/pages/CoinLookupPage.vue` (CHANGED), `DeepAnalysisPage.vue`, `src/web/src/components/deep-identification/{DeepAnalysisStartPanel,DeepAnalysisProgressTimeline,DeepProviderCoverageList,DeepReportPanel,DeepProposalEditor,DeepAnalysisEntryButton}.vue`, `src/web/src/composables/{useDeepIdentification,useDeepIdentificationStream}.ts`, `src/web/src/api/client.ts` (CHANGED), `src/web/src/types/index.ts` (CHANGED), `src/web/src/router/index.ts` (CHANGED), `src/web/src/components/coin/CoinActionsPanel.vue` (CHANGED), `src/web/e2e/workflows/` additions
  - Docs: `docs/adr/0010-deep-agentic-coin-identification.md`, `docs/openapi.json` (regenerated), `docs/ARCHITECTURE.md` (CHANGED), `docs/features/ai-analysis.md` (CHANGED), `docs/quick-capture.md` (CHANGED), `docs/testing.md` (CHANGED)
  - Decisions: `.squad/decisions/inbox/` new entry (never edit `.squad/decisions.md` directly)
