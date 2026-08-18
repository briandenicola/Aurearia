---
description: "Task list for feature 353 — Wishlist Availability Run Observability"
---

# Tasks: Wishlist Availability Run Observability

**Input**: Design documents from `specs/353-wishlist-availability-run-observability/`
**Prerequisites**: `spec.md`, `plan.md`

**Tests**: Included. Every implementation task has a tests-first counterpart per constitution §17 Quality Gate.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel with other [P] tasks (touches different files, no dependencies).
- **[Story]**: `US1`–`US5` from `spec.md`. Cross-cutting tasks use `[X]`.
- File paths are absolute-in-repo.

---

## Phase 1: Setup (Shared Infrastructure)

- [ ] **T001** [X] Confirm baseline green: run `go test ./services/... ./repository/... ./handlers/... -run Availability` and record output; run `npm run type-check` in `src/web/`. No source edits.
- [ ] **T002** [X] [P] Create empty scaffold files (no code yet) at `src/api/models/availability_cycle.go` and `src/api/repository/availability_cycle_repository.go` with `package` line only, so subsequent tasks add symbols to known paths.

---

## Phase 2: Foundational (Data model + repository) — BLOCKING

**Purpose**: Cycle table, `CycleID` FK, and repository primitives. All user stories depend on this.

### Tests first

- [ ] **T003** [X] Write repository test `src/api/repository/availability_cycle_repository_test.go` covering:
  - `EnqueueCycle` idempotency inside 5-minute window (FR-007, US2 AC2).
  - `ClaimCycle` transitions `queued` → `running` exactly once (FR-008).
  - `AggregateChildCounts` truth table over `{queued, running, completed, failed}` combinations producing correct parent status (`completed`, `failed`, `partial_failure`, `running`) (FR-008, US2 AC3/AC4).
  - `PruneOldCycles` keeps ≤ 20 terminal cycles and nulls out `cycle_id` on surviving children first (FR-019, US4 AC2).
- [ ] **T004** [X] Extend `src/api/repository/availability_repository_test.go` with:
  - `CreateChildRun` rejects `UserID == 0` (FR-002, SC-001).
  - `CompleteChildRun` prunes so ≤ 20 terminal per owner (FR-018, US4 AC1) and never touches queued/running rows.
  - `ListRunsForOwner` returns only rows for that owner (FR-017, FR-022, US1 AC1).
  - `GetOwnedRunWithResults` refuses cross-owner reads (FR-017, FR-023).

### Implementation

- [ ] **T005** [X] Define `AvailabilityCycle` model in `src/api/models/availability_cycle.go` per plan §Data Model. Add status constants `AvailabilityCycleStatus{Queued,Running,Completed,Failed,PartialFailure}`.
- [ ] **T006** [X] Add `CycleID *uint` (nullable, indexed) to `AvailabilityRun` in `src/api/models/availability_check.go`; update `TriggerType` doc comment to enumerate the values used by *new* rows, `{owner, scheduled, admin}` — pre-existing rows (including legacy `UserID = 0` admin rows) keep whatever historical value they already have and are never rewritten; export constants for the three new-row trigger types.
- [ ] **T007** [X] Register both models in `AutoMigrate` in `src/api/database/database.go` (append `&models.AvailabilityCycle{}` to the list).
- [ ] **T008** [X] Implement `AvailabilityCycleRepository` in `src/api/repository/availability_cycle_repository.go` with `EnqueueCycle`, `ClaimCycle`, `FinalizeCycle`, `AggregateChildCounts(cycleID)`, `ListCycles(page,limit)`, `GetCycleWithChildren(id)`, `PruneOldCycles(keep=20)`, and `RecoverStaleCycles(timeout)`.
- [ ] **T009** [X] Modify `src/api/repository/availability_repository.go`:
  - Add `CreateChildRun(run)` that returns error if `run.UserID == 0`.
  - Add `CompleteChildRun(run)` — transactional finalize + call `PruneTerminalChildRunsForOwner(userID, keep=20)` + return `(parentFinalized bool, err)` after `AggregateChildCounts`.
  - Add `ListRunsForOwner(userID, page, limit)` and `GetOwnedRunWithResults(userID, runID)`.
  - Delete the global `PruneOldRuns(100)` call from `CompleteRun` (retention now per-owner).
- [ ] **T010** [X] Run T003 + T004 → GREEN. Run `go test ./architecture_test.go` to confirm no cross-layer regressions.

**Checkpoint**: Data + repo layer done. All user stories may begin.

---

## Phase 3: User Story 1 — Owner sees every run in their history (P1)

**Goal**: Every check that touches a user's wishlist appears in that user's history with correct scoping.

### Tests

- [ ] **T011** [P] [US1] `src/api/services/availability_service_test.go`: test `CheckWishlistForUser` creates exactly one child run per invocation with `UserID == invokerUserID` and never `0` (FR-002, FR-003, SC-001).
- [ ] **T012** [P] [US1] `src/api/handlers/availability_handler_test.go`: test `GET /wishlist/availability-runs` returns only caller's runs (FR-017, FR-022, US1 AC1/AC2/AC3).

### Implementation

- [ ] **T013** [US1] Refactor `CheckWishlistForUser` in `src/api/services/availability_service.go` to signature `(userID uint, triggerType string, triggerUserID *uint, cycleID *uint)`, use `CreateChildRun`, and call `notifyRunTerminal` on every terminal transition (see US3 T021).
- [ ] **T014** [US1] Add owner handlers in `src/api/handlers/availability.go`:
  - `ListOwnerRuns` → `GET /wishlist/availability-runs`.
  - `GetOwnerRunDetail` → `GET /wishlist/availability-runs/{id}` using `GetOwnedRunWithResults(userID, runID)`.
  - Full Swaggo annotations (`@Tags Wishlist`, `@Router`, `@Security BearerAuth`).
- [ ] **T015** [US1] Wire new routes in `src/api/main.go` under the `protected` group next to existing `/wishlist/check-availability`.

**Checkpoint**: Owners can list and drill into every run touching their coins.

---

## Phase 4: User Story 2 — Admin sees parent cycles with roll-up (P1)

**Goal**: One parent row per admin/scheduled cycle, aggregated status, `409` on duplicate.

### Tests

- [ ] **T016** [P] [US2] `src/api/services/availability_scheduler_test.go`: test `RunNowWithTrigger` inside the 5-minute window returns `ErrAvailabilityRunInProgress` (FR-007, US2 AC2).
- [ ] **T017** [P] [US2] `src/api/services/availability_service_test.go`: test `RunAdminCycle` fans out one child per user, aggregates parent to `completed` when all succeed, `partial_failure` on mixed, `failed` when all fail (FR-008, US2 AC3/AC4).
- [ ] **T018** [P] [US2] `src/api/handlers/availability_handler_test.go`: `POST /admin/availability/run` returns `202` with `{cycleId,status}` on success and `409` on duplicate; `GET /admin/availability-cycles` and `.../{id}` return parents + child summaries (FR-024, FR-025, FR-026).

### Implementation

- [ ] **T019** [US2] Refactor `RunNowWithTrigger` in `src/api/services/availability_scheduler.go` to call `availCycleRepo.EnqueueCycle` (not `EnqueueManualRun`) and to enqueue `cycleID` into `queue`. Update `processRun`/`ProcessRun` to `processCycle`/`ProcessCycle` claiming a cycle and calling `AvailabilityService.RunAdminCycle`.
- [ ] **T020** [US2] Refactor `runCycle` (scheduled path) to enqueue a scheduled `AvailabilityCycle` via `EnqueueCycle` and delegate to the same worker. Implement `RunAdminCycle(cycle)` in `src/api/services/availability_service.go` that: (a) lists affected users (see FR-006), (b) creates one child per user, (c) executes them sequentially, (d) each child terminal transition already updates parent counts via `CompleteChildRun` + `AggregateChildCounts` + `FinalizeCycle`.
- [ ] **T021** [US2] Add admin handlers in `src/api/handlers/availability.go`:
  - `ListCycles` → `GET /admin/availability-cycles`.
  - `GetCycleDetail` → `GET /admin/availability-cycles/{id}` returning parent + child summaries (no per-coin results here).
  - Update `TriggerRun` response to `{cycleId, status}`; keep `409` behavior. Full Swaggo annotations.
- [ ] **T022** [US2] Wire cycle handlers + `availCycleRepo` in `src/api/main.go` (build repo, pass into service + scheduler + handler; register the two new admin routes; keep existing `/admin/availability-runs*` for legacy drill-in).

**Checkpoint**: Admin has parent cycles + duplicate protection.

---

## Phase 5: User Story 3 — Every terminal run notifies the owner (P1)

**Goal**: Owner in-app + Pushover per terminal child; admin gets a second notification on admin-triggered child failure.

### Tests

- [ ] **T023** [P] [US3] `src/api/services/notification_service_test.go` (new or extend): test `NotifyAvailabilityRunTerminal` creates one `wishlist_availability_run` notification with generic message, no URLs, no query text; and test `NotifyAdminCycleChildFailure` (FR-010, FR-012, FR-015, SC-007).
- [ ] **T024** [P] [US3] `src/api/services/availability_service_test.go`: matrix test — for each `(triggerType ∈ {owner,scheduled,admin}) × (outcome ∈ {zero_urls, no_change, changes, failure})`, exactly one owner in-app notification is created (FR-010, FR-013, SC-002).
- [ ] **T025** [P] [US3] Same file: test Pushover client returning error does not change child `status` (FR-011, SC-008).
- [ ] **T026** [P] [US3] Same file: test admin-triggered child failure produces a second notification to the admin whose message contains owner username + `cycleId` but no URLs (FR-012, US3 AC4).
- [ ] **T027** [P] [US3] Same file: test per-coin `wishlist_unavailable` notifications continue to fire independently of — i.e., are **not** suppressed by — the `wishlist_availability_run` outcome notification for the same run; both notification types must exist for a run in which a coin transitions to unavailable (FR-014, SC-009, design decision D6).

### Implementation

- [ ] **T028** [US3] Add `NotifyAvailabilityRunTerminal(userID, run, newlyUnavailableCoinNames)` and `NotifyAdminCycleChildFailure(adminID, ownerUsername, cycleID)` to `src/api/services/notification_service.go`. Both build generic titles/messages (no URLs) and dispatch Pushover async.
- [ ] **T029** [US3] In `src/api/services/availability_service.go`, implement `notifyRunTerminal(run)` that:
  - Collects newly-unavailable coin names (max 3 shown + "and N more") from the run's results by comparing to each coin's previous `ListingStatus`.
  - Calls `notifSvc.NotifyAvailabilityRunTerminal(...)` **after** the terminal transition commits (FR-011 isolation).
  - If `run.TriggerType == "admin"` and terminal status is `failed`, calls `notifSvc.NotifyAdminCycleChildFailure(triggerUserID, owner.Username, cycleID)`.
  - Leaves the existing direct per-coin `NotifyWishlistUnavailable` call in this same code path completely unmodified — it continues to fire independently of `NotifyAvailabilityRunTerminal`; neither call gates or suppresses the other (D6, FR-014).
- [ ] **T030** [US3] Ensure the generic failure message helper (e.g., `func genericAvailabilityFailureMessage() string`) is the single source used by `run.FailMessage`, notification body, and cycle `FailMessage` (FR-015, SC-007).

**Checkpoint**: Notification matrix verified.

---

## Phase 6: User Story 4 — Bounded retention (P2)

**Goal**: 20 terminal children per owner; 20 terminal cycles total; never prune non-terminal.

### Tests

- [ ] **T031** [P] [US4] Extend `src/api/repository/availability_repository_test.go` with retention scenarios: 20 terminal + 1 queued + 1 running; new completion prunes exactly one oldest terminal; queued/running untouched (FR-018).
- [ ] **T032** [P] [US4] Extend `src/api/repository/availability_cycle_repository_test.go` with cycle retention scenario including surviving children whose `cycle_id` gets nulled before parent delete (FR-019, US4 AC2).

### Implementation

- [ ] **T033** [US4] Implement `PruneTerminalChildRunsForOwner(userID, keep=20)` in `availability_repository.go`; call from `CompleteChildRun` post-commit.
- [ ] **T034** [US4] Implement `PruneOldCycles(keep=20)` in `availability_cycle_repository.go`; call from `FinalizeCycle` post-commit; ensure the `UPDATE availability_runs SET cycle_id = NULL WHERE cycle_id IN (...)` step precedes cycle deletion.
- [ ] **T035** [US4] Delete legacy `PruneOldRuns(100)` call site in `CompleteRun`; if `CompleteRun` is now unused, remove it (and update tests).

**Checkpoint**: Retention bounded and safe.

---

## Phase 7: User Story 5 — Additive schema only, legacy rows untouched (P2)

**Goal**: The new `AvailabilityCycle` table and `AvailabilityRun.CycleID` column are added via plain `AutoMigrate` DDL; no pre-existing row (legacy `UserID=0` admin rows or `UserID>0` rows) is read, rewritten, retagged, or reparented; legacy rows stay reachable through the existing legacy admin endpoints.

### Tests

- [ ] **T036** [P] [US5] Add `src/api/database/database_test.go` (extend if it exists, else create) schema test:
  - Seed a fixture on the pre-feature schema: 3 legacy admin rows (`UserID=0`, `TriggerType="manual"`) each with 5 results; 5 legacy user rows (`UserID>0`, `TriggerType="scheduled"`) each with 3 results.
  - Run `AutoMigrate` with `AvailabilityCycle` registered; assert: zero new `AvailabilityCycle` rows are created for any of them; every legacy row's `UserID`, `TriggerType`, `Status`, counts, and results are byte-identical to the pre-migrate fixture; the new `CycleID` column is `NULL` on every existing row (its natural default, not a write); no `AvailabilityResult` row is deleted or reassigned (FR-021, US5 AC1/AC2/AC3, SC-006).
  - Run `AutoMigrate` a second time; assert row counts and field values are unchanged (idempotency of the additive DDL — there is no custom migration function to test separately).

### Implementation

- [ ] **T037** [US5] Verify no other change is needed: the `AutoMigrate` registration of `&models.AvailabilityCycle{}` (done in T007) is the entire schema change for this feature. Confirm the existing legacy-serving repository/handler code (`GetRunWithResults`, `ListRuns`, `GetLastScheduledRun` in `availability_repository.go`; existing `/admin/availability-runs*` handlers) requires **no modification** by running their existing test suites unchanged and green — no custom migration function, `AppSetting` version flag, or backfill/reparenting code is added (FR-021).
- [ ] **T038** [US5] Add an informational (non-error) boot-time log line in `Connect()` reporting the count of pre-existing `availability_runs WHERE user_id = 0` rows, for operator visibility only; this is diagnostic, not an invariant failure, since legacy `UserID=0` rows are expected and valid (FR-021a). The `UserID > 0` invariant for *new* rows is enforced at creation time by `CreateChildRun` (T009), not by a boot-time scan of historical data.

**Checkpoint**: Schema migration verified additive-only and non-destructive; legacy rows unchanged.

---

## Phase 8: Cross-cutting service tests + stale recovery

- [ ] **T039** [X] Add `src/api/services/availability_scheduler_test.go` cases for `RecoverStaleCycles`: a `running` cycle whose children are all terminal is finalized on boot; a `running` cycle with active children is left alone.
- [ ] **T040** [X] Add integration-style test in `src/api/services/availability_service_test.go`: full admin cycle over 3 users with mixed outcomes; assert exactly 3 owner notifications, 1 admin failure notification, parent status `partial_failure`, per-owner counts correct, retention capped.

---

## Phase 9: Frontend

### Tests + implementation for admin history

- [ ] **T041** [P] [US2] Vitest `src/web/src/components/admin/schedules/__tests__/AdminAvailabilitySchedule.test.ts`: cycles list renders parent rows with counts; legacy `UserID=0` rows (from the existing `/admin/availability-runs*` endpoints) render with a "Legacy" chip and are visually distinct from `AvailabilityCycle` parent rows; expanding a parent cycle fetches `/admin/availability-cycles/{id}` and lists children; duplicate-trigger surfaces `409` gracefully.
- [ ] **T042** [US2] Update `src/web/src/components/admin/schedules/AdminAvailabilitySchedule.vue`: swap the flat runs table for a parent-cycles table with expandable child rows; continue rendering pre-existing legacy `UserID=0` rows (fetched from the retained `/admin/availability-runs*` endpoints) alongside the new parent cycles, each tagged with a `.chip-sm` "Legacy" label so admins can't mistake them for new cycles (FR-021a). Use `.chip` for status, `.btn-sm` for actions, and the existing table classes. Preserve "Run Now" button behavior calling `POST /admin/availability/run` (now returns `{cycleId,status}`).
- [ ] **T043** [P] [US2] Add helpers to `src/web/src/api/client.ts`: `availabilityCycles.list(page,limit)`, `availabilityCycles.get(id)`, `availabilityCycles.trigger()`.

### Owner history page

- [ ] **T044** [P] [US1] Vitest `src/web/src/pages/__tests__/WishlistAvailabilityHistoryPage.test.ts`: loading, empty, populated list, row click drills into detail; API 401 handled.
- [ ] **T045** [US1] Add `src/web/src/pages/WishlistAvailabilityHistoryPage.vue` with pagination, row rendering (date, trigger, counts, status chip), and a detail modal or subview showing the run's per-coin results. Follow constitution Design System (tokens, `.chip`, `.chip-sm`, `.info-label`, no hardcoded colors).
- [ ] **T046** [P] [US1] Add API helpers `availabilityRuns.listMine(page,limit)` and `availabilityRuns.getMine(id)` to `src/web/src/api/client.ts`; ensure nullable fields use `?? null` per repo TS convention.
- [ ] **T047** [US1] Register `/wishlist/availability-runs` route in `src/web/src/router/index.ts` (auth-required) and add a "Run History" link in the Wishlist area (reuse existing wishlist page navigation).

---

## Phase 10: Polish, docs, and Quality Gate

- [ ] **T048** [X] Run full `go test ./...` and `npm run type-check` + `npx vitest run` in `src/web/`. Fix any regressions.
- [ ] **T049** [X] Regex sweep test: assert that for every generated notification title/message and every API `failMessage`/cycle `FailMessage`, no `http://`/`https://` substring appears (SC-007). Put this in `src/api/services/availability_service_test.go`.
- [ ] **T050** [X] Update `docs/ARCHITECTURE.md` availability section (if it references the old single-run model). Do not create new docs unless a section already exists.
- [ ] **T051** [X] Manual smoke run (documented in PR description, not committed as a file):
  - Owner-triggered check with zero URLs → 1 in-app notification, generic message, no run status regression.
  - Admin cycle across 2 users where one child fails → parent `partial_failure`, both owners notified, admin gets failure notification, no URL leaks.
  - Duplicate admin trigger inside 5 min → `409`.
  - Restart mid-cycle → stale recovery finalizes or re-queues cleanly.

---

## Dependencies & Execution Order

### Phase order
- Phase 1 → Phase 2 (blocking) → Phases 3–7 (parallelizable by story, but Phase 5 and Phase 6 depend on Phase 2 primitives; Phase 7 depends on the model changes in Phase 2) → Phase 8 (integration) → Phase 9 (frontend, depends on Phase 3 + Phase 4 handlers) → Phase 10.

### Story dependencies
- **US1** depends only on Phase 2.
- **US2** depends on Phase 2 (needs cycle repo).
- **US3** depends on Phase 2 + service refactor from US1/US2 (T013/T020).
- **US4** depends on Phase 2 primitives (retention hooks live in repos).
- **US5** depends on Phase 2 (model shape must be final).

### Within each story
- Tests (T003, T004, T011, T012, T016–T018, T023–T027, T031, T032, T036, T041, T044) before implementation.
- Models before repositories before services before handlers before wiring before frontend.

### Parallel opportunities
- T003 ∥ T004 (different files).
- T011 ∥ T012 (different files).
- T016 ∥ T017 ∥ T018 (three different files).
- T023 ∥ T024 ∥ T025 ∥ T026 ∥ T027 (all inside one file → NOT parallel-safe; run sequentially in the same edit batch, but T023 is a different file → T023 is [P]).
- T031 ∥ T032 (different files).
- T041 ∥ T043 ∥ T044 ∥ T046 (different files).

---

## Notes

- All new/modified Go files must keep the layered import rules (constitution Principle I); `architecture_test.go` will catch violations.
- Every commit uses Conventional Commits + `Co-authored-by: Copilot` trailer.
- No Python agent changes are made in any task; if a task appears to need one, stop and escalate.
- Wishlist Search Alerts files (`wishlist_search_alert*`, `alert_run*`) are **out of scope** for every task in this list.
