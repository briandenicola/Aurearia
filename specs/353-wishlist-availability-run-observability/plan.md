# Implementation Plan: Wishlist Availability Run Observability

**Branch**: `353-wishlist-availability-run-observability` | **Date**: 2026-08-17 | **Spec**: `./spec.md`
**Input**: Feature specification from `specs/353-wishlist-availability-run-observability/spec.md`

## Summary

Refactor the wishlist availability observability layer into a parent `AvailabilityCycle` + per-user `AvailabilityRun` (child) model, add bounded retention (20 terminal per owner, 20 terminal cycles globally), and guarantee a terminal-outcome notification on every completed child run (owner in-app + Pushover; admin trigger failure also notifies the triggering admin) **in addition to** the existing per-coin unavailable notification. The Python agent contract, `CheckURL` logic, agent-escalation flow, and Wishlist Search Alerts subsystem are untouched. Legacy admin history (`UserID = 0` rows) is left completely unmodified and remains readable through the existing legacy admin endpoints with a "Legacy" UI label; no synthetic parent cycles are created for it.

## Technical Context

**Language/Version**: Go 1.26.6 (API), TypeScript 5 / Vue 3 (web).
**Primary Dependencies**: Gin, GORM, SQLite; existing `NotificationService`, `PushoverService`, `AvailabilityService`, `AvailabilityScheduler`.
**Storage**: SQLite via GORM `AutoMigrate` only (no custom migration hooks; the new table and column are purely additive DDL registered in `src/api/database/database.go`).
**Testing**: `go test ./...` (architecture + unit + service tests), `npm run type-check` + `npx vitest` for the Vue admin/owner history views.
**Target Platform**: Linux server (single instance), PWA client.
**Project Type**: Web application (Go API + Vue SPA + stateless Python agent).
**Performance Goals**: No regression vs. current scheduler; parent-cycle aggregation must be O(childCount) per transition and never scan `AvailabilityResult`.
**Constraints**: Preserve per-owner P95 wishlist history load under 200 ms with 20 rows; no destructive data changes on first boot.
**Scale/Scope**: ≤ 20 users, ≤ ~500 wishlist URLs total per cycle in practice.

## Constitution Check

- **Principle I (Layered Architecture)**: All new logic lives in `handlers/` → `services/` → `repository/` → `models/`. `AvailabilityCycle` handling stays in `services/availability_service.go` (new methods) and `services/availability_scheduler.go`; repository gains cycle-scoped queries. Verified by `architecture_test.go`.
- **Principle IV (Simple Complete Proportional)**: We reuse `AvailabilityRun`/`AvailabilityResult`/`NotificationService`/`PushoverService`. Only one new table (`AvailabilityCycle`) and one new nullable FK column.
- **Principle V (Security, Privacy)**: All owner-facing endpoints scope by `auth.userId`. Failure messages are generic; URLs never leave server logs. Admin routes require the admin JWT claim.
- **Principle VI (Consistent UX)**: Frontend reuses `.chip`, `.btn`, `AdminAvailabilitySchedule.vue` patterns; new owner history view mirrors existing admin history table styles.
- **§17 Quality Gate**: Targeted `go test` on `services/`, `repository/`, `handlers/` availability packages; `vue-tsc` + affected `vitest` files; full `go test ./...` before merge.
- **§21 Definition of Done**: See §"Definition of Done" checklist below.

No violations. **Complexity Tracking** table is empty (no waivers).

## Design Decisions

### D1. Parent as separate table, not overloaded `AvailabilityRun`
`AvailabilityCycle` is a new table rather than overloading `AvailabilityRun` with a `scope="cycle"` variant. Rationale: eliminates the `UserID = 0` sentinel (spec FR-002) and lets the child schema keep `UserID NOT NULL`, matching Brian's binding constraint. Parallels the `ValuationRun` (per-user) pattern already in `models/valuation_run.go`; the new cycle table plays the role that `CoinOfDayRun` plays for the daily scheduler (operational metadata).

### D2. Aggregation on child terminal transition, not by timer
Parent status/counts are recomputed atomically inside the same transaction that finalizes each child (`completeChildRun`). This avoids a background aggregator, avoids stale parents, and keeps recovery trivial (a parent whose children are all terminal is itself terminal on next child transition or on `RecoverStaleRuns` sweep).

### D3. Cycle-level duplicate protection via `AvailabilityCycleRepository.EnqueueCycle`
Mirrors the existing `EnqueueManualRun` transactional check-and-insert. `since = now - 5m` window. Prevents duplicate admin cycles. Scheduled cycles use the same guard so a cron overlap cannot double-fire.

### D4. Retention: keep 20 terminal per owner + 20 terminal cycles global
Applied post-commit inside `CompleteChildRun` (per-owner prune) and `finalizeCycle` (cycle prune). Never touches non-terminal rows. Cycle prune first `UPDATE availability_runs SET cycle_id = NULL WHERE cycle_id IN (...)` so surviving child history isn't accidentally cascaded away.

### D5. Legacy rows are left completely untouched — no synthesis, no retagging
Pre-existing `AvailabilityRun` rows (both `UserID = 0` legacy admin rows and `UserID > 0` rows) are **not** touched by this feature's migration in any way beyond gaining the new nullable `CycleID` column (which defaults to `NULL` for every existing row, same as any newly-added nullable column). No `AvailabilityCycle` row is synthesized for them, no `TriggerType` retagging occurs, and no `AppSetting` migration-version flag or backfill routine is needed because there is no data transformation to gate. Legacy `UserID = 0` rows remain reachable through the existing `GET /admin/availability-runs*` endpoints exactly as today; the admin UI adds a "Legacy" label so these rows are visually distinct from new `AvailabilityCycle` parent rows. New `AvailabilityCycle` parents are created going forward only, by the new admin/scheduled trigger paths (D3).

### D6. Both notification types coexist — no suppression
Existing `NotifyWishlistUnavailable` per-coin notifications continue to fire unmodified for every coin that transitions to `unavailable` within a run — this is existing, unchanged behavior. The new `wishlist_availability_run` terminal notification is purely additive: exactly one is created per terminal child run regardless of how many (if any) per-coin notifications also fired for that run. The outcome notification's summary text may mention newly-unavailable coin names (up to 3 + "and N more") for readability, but this is cosmetic context only and never gates, replaces, or skips the per-coin call. Verified by a dedicated regression test (SC-009) asserting both notification types exist when a coin transitions unavailable within the same run.

### D7. Pushover failure isolation
`sendPushover` remains a fire-and-forget goroutine (existing pattern in `NotificationService`). The child run's terminal transition is committed **before** any Pushover send starts, so a Pushover error can never regress the run status.

### D8. No API OpenAPI regeneration step
Handlers use inline `@Summary`/`@Router` Swaggo annotations (see existing `availability.go`). The repo has no `swag init` step wired into CI; annotations are documentation-only. New handlers include the same annotation shape; no separate contracts/ directory is generated.

## Project Structure

### Documentation (this feature)

```text
specs/353-wishlist-availability-run-observability/
├── spec.md              # Feature spec
├── plan.md              # This file
└── tasks.md             # Dependency-ordered task list
```

No `research.md`, `data-model.md`, `contracts/`, or `quickstart.md` are required: the code base is well-understood, the data model is captured inline (§Data Model below), and API contracts are documented via Swaggo annotations on the new handlers.

### Source Code

```text
src/api/
├── models/
│   ├── availability_check.go               # MODIFY: add CycleID, update trigger types
│   └── availability_cycle.go               # NEW: AvailabilityCycle model
├── repository/
│   ├── availability_repository.go          # MODIFY: per-owner listing, retention, child completion
│   └── availability_cycle_repository.go    # NEW: EnqueueCycle, ClaimCycle, FinalizeCycle,
│                                           #      AggregateChildCounts, ListCycles, GetCycleWithChildren,
│                                           #      PruneOldCycles
├── services/
│   ├── availability_service.go             # MODIFY: split into
│   │                                       #   CheckWishlistForUser (child-run aware, always notifies)
│   │                                       #   RunManualCycle → RunAdminCycle (fans out child runs)
│   │                                       #   notifyRunTerminal (in-app + Pushover for every terminal)
│   ├── availability_scheduler.go           # MODIFY: scheduler creates cycles instead of loose runs;
│   │                                       #   worker processes cycles → children
│   └── notification_service.go             # MODIFY: NotifyAvailabilityRunTerminal(owner, run, summary)
│                                           #         NotifyAdminCycleChildFailure(adminID, ownerName, cycleID)
├── handlers/
│   └── availability.go                     # MODIFY: split admin vs owner routes;
│                                           #         add /wishlist/availability-runs*
│                                           #         add /admin/availability-cycles*
├── database/
│   └── database.go                         # MODIFY: AutoMigrate AvailabilityCycle + CycleID column
│                                           #         (additive only; no data migration function)
├── main.go                                 # MODIFY: wire NewAvailabilityCycleRepository,
│                                           #         update handler constructor signatures
└── architecture_test.go                    # unchanged — must still pass

src/web/src/
├── components/admin/schedules/
│   └── AdminAvailabilitySchedule.vue       # MODIFY: history table shows parent cycles + expand children
├── pages/
│   └── WishlistAvailabilityHistoryPage.vue # NEW: owner-facing run history + detail
└── api/
    └── client.ts                           # MODIFY: add availabilityRuns.* and availabilityCycles.* helpers
```

**Structure Decision**: Web-app structure (Go API + Vue SPA). All new files land in existing directories; no new packages required.

## Data Model

### AvailabilityCycle (NEW)

| Column           | Type          | Notes                                                                  |
|------------------|---------------|------------------------------------------------------------------------|
| `id`             | uint PK       |                                                                        |
| `trigger_type`   | varchar(20)   | `scheduled` \| `admin` (new cycles only — no legacy value; legacy rows are never given a cycle) |
| `trigger_user_id`| uint NULL     | Admin who initiated (nullable for scheduled/legacy)                    |
| `status`         | varchar(20)   | `queued` \| `running` \| `completed` \| `failed` \| `partial_failure`  |
| `total_children` | int           | Set at fan-out time                                                    |
| `queued_children`| int           | Aggregated from children                                               |
| `running_children`| int          | Aggregated                                                             |
| `completed_children`| int        | Aggregated                                                             |
| `failed_children`| int           | Aggregated                                                             |
| `fail_message`   | text          | Generic only                                                           |
| `started_at`     | datetime      |                                                                        |
| `completed_at`   | datetime NULL |                                                                        |
| `created_at`     | datetime      |                                                                        |

### AvailabilityRun (MODIFIED)

Add: `cycle_id BIGINT NULL` (index, defaults to `NULL` for every existing row via the new-column default — no backfill or update statement touches historical rows). `trigger_type` values for *new* rows are {"owner","scheduled","admin"}; pre-existing rows keep whatever historical value they already have (including admin rows with `user_id = 0`) and are never rewritten. New-row invariant `user_id > 0` enforced in service layer + boot-time assertion that only inspects rows created after this feature ships.

### AvailabilityResult (UNCHANGED).

## Phases

### Phase 0 — Preflight

- Read constitution §17, §18, §21; confirm no ADR needed (D1–D8 do not deviate from binding principles).
- Confirm baseline `go test ./services/... ./repository/... ./handlers/...` is green before edits.

### Phase 1 — Data model + repository foundation (blocks everything)

- Add `AvailabilityCycle` model; add `CycleID` to `AvailabilityRun`; wire `AutoMigrate`.
- Add `AvailabilityCycleRepository` with `EnqueueCycle`, `ClaimCycle`, `FinalizeCycle`, `AggregateChildCounts`, `ListCycles`, `GetCycleWithChildren`, `PruneOldCycles`.
- Extend `AvailabilityRepository`:
  - `CreateChildRun(run)` rejects `UserID == 0`.
  - `CompleteChildRun(run)` runs terminal transition + retention prune (20 per owner) + calls `AggregateChildCounts` and possibly `FinalizeCycle`.
  - `ListRunsForOwner(userID, page, limit)`, `GetOwnedRunWithResults(userID, runID)`.
  - Replace `PruneOldRuns(100)` global with per-owner prune.
- Add repository unit tests for retention, `EnqueueCycle` idempotency, and `AggregateChildCounts` truth table.

### Phase 2 — Service layer (child run lifecycle + notifications)

- Refactor `AvailabilityService`:
  - `CheckWishlistForUser(userID, triggerType, triggerUserID, cycleID *uint)` — always creates a child run, always calls `notifyRunTerminal` on terminal (success or failure); the existing per-coin `NotifyWishlistUnavailable` call in this path is left in place, unmodified — it fires independently of the new outcome notification (D6).
  - New `notifyRunTerminal(owner, run, newlyUnavailableCoins)` builds the generic message, creates the in-app notification, and calls Pushover async (D7).
  - `RunAdminCycle(cycle)` fans out one child per affected user (users with any wishlist coin at all, per FR-006), executes children sequentially (respecting existing rate delay), and per FR-012 fires admin-failure notification for each failed child.
- Extend `NotificationService`:
  - `NotifyAvailabilityRunTerminal(userID, run, summary)` (`type=wishlist_availability_run`).
  - `NotifyAdminCycleChildFailure(adminID, ownerUsername, cycleID)`.
- Refactor `AvailabilityScheduler`:
  - `RunNowWithTrigger(adminID)` now enqueues a `AvailabilityCycle` (via `EnqueueCycle` — the duplicate-protection point).
  - `worker()` claims cycles, calls `RunAdminCycle`; per-run stale recovery becomes per-cycle (recover stuck `running` cycles → requeue).
  - `runCycle()` (scheduled path) also enqueues a scheduled `AvailabilityCycle` and defers to the same worker.
- Add service tests for: every-run-notifies matrix (zero URLs / no change / changes / failure), admin-failure double notify, parent aggregation, retention interaction, Pushover failure isolation, duplicate-trigger `409`.

### Phase 3 — Handlers, API, and wiring

- Refactor `handlers/availability.go`:
  - Owner: `GET /wishlist/availability-runs`, `GET /wishlist/availability-runs/{id}` (both scoped to `auth.userId`).
  - Admin: `GET /admin/availability-cycles`, `GET /admin/availability-cycles/{id}`; keep `GET /admin/availability-runs`, `GET /admin/availability-runs/{id}`; `POST /admin/availability/run` now returns `{cycleId,status}`.
  - Keep `POST /wishlist/check-availability` (owner-triggered single child, `cycleId=null`).
  - Keep `PUT /coins/:id/listing-status` unchanged.
  - Update Swaggo annotations on every new/modified route.
- Update `main.go`: construct `availCycleRepo`, pass to service + scheduler + handler constructors.
- Update `main.go` route registration blocks (protected + admin).
- Add handler tests for auth scoping (owner cannot read another owner's run; non-admin cannot list cycles) and for `409` duplicate cycle.

### Phase 4 — Additive schema migration + boot invariant (no backfill)

- Register `&models.AvailabilityCycle{}` and the new `AvailabilityRun.CycleID` field with GORM `AutoMigrate` in `database/database.go`, alongside the other models already auto-migrated there. This is the entire "migration": one new table, one new nullable column. No custom migration function, no `AppSetting` version flag, no data transformation — there is nothing to gate because no row is rewritten.
- Pre-existing rows (legacy `UserID = 0` admin rows and `UserID > 0` rows alike) are left exactly as they are; `AutoMigrate` gives the new `CycleID` column its natural `NULL` default and does not touch any other field.
- Add a boot-time invariant log (informational, not a migration step): count `availability_runs WHERE cycle_id IS NULL AND user_id = 0` created **after** the feature's deploy timestamp — expected 0 (every new admin/scheduled child must have `UserID > 0`). This only inspects newly-created rows; it does not fire on legacy data.
- Schema test: assert `AutoMigrate` adds the table/column without modifying any pre-existing `AvailabilityRun`/`AvailabilityResult` row (seed a fixture of legacy admin + owner rows, run `AutoMigrate`, assert row counts and field values are byte-identical before/after).
- Admin UI / API: legacy `UserID = 0` rows continue to be served by the existing `GET /admin/availability-runs*` endpoints unchanged; the admin history view adds a "Legacy" label/fallback so these rows are visually distinguished from new `AvailabilityCycle` parent rows (FR-021a).

### Phase 5 — Frontend

- Update `AdminAvailabilitySchedule.vue` history table to render parent cycles with expandable child rows; show `childCounts`, generic `failMessage`, and a "View children" affordance backed by `/admin/availability-cycles/{id}`.
- Add owner-facing `WishlistAvailabilityHistoryPage.vue` (route `/wishlist/availability-runs`) with pagination, terminal-only display, per-row drill-in to `/wishlist/availability-runs/{id}` detail. Reuse existing tokens/classes per constitution Design System.
- Add API client helpers in `src/web/src/api/client.ts` (`availabilityRuns.listMine`, `getMine`, `availabilityCycles.list`, `get`, `trigger`).
- Vitest smoke tests for the two views (loading / empty / rows / drill-in click).
- Sanity check both flows against Docker `vue-tsc --build` conventions (nullable chaining on `run.cycleId?.toString()`, etc.).

### Phase 6 — Rollback plan

Rollback is **safe by construction**:
- Reverting the code without touching the DB leaves the new `availability_cycles` table and `cycle_id` column intact but unread by the old code; no old-code path breaks. No `AppSetting` flag exists to reset because no data was ever rewritten.
- If a full data rollback is needed: delete the `availability_cycles` table and drop the `cycle_id` column. Legacy `UserID = 0` and `UserID > 0` rows were never modified, so nothing else needs reverting. No ADR is required for this rollback since no destructive or synthetic migration exists to document.

## Idempotency, Concurrency, and Stale Recovery

- **Cycle enqueue**: `EnqueueCycle(cycle, since)` — transactional insert-if-none-active, mirroring `EnqueueManualRun`.
- **Child creation**: fanned out inside `RunAdminCycle` after the cycle is claimed; each child insert asserts `UserID > 0`.
- **Terminal transition**: `CompleteChildRun` uses a transaction: update run status → aggregate parent counts → possibly finalize parent → prune owner children → (if parent finalized) prune old cycles.
- **Notifications**: dispatched **after** the terminal-transition transaction commits, from within a defer/goroutine boundary; a Pushover error never rolls back the DB.
- **Stale recovery on boot**: `RecoverStaleRuns(15m)` remains for children; `RecoverStaleCycles(15m)` is new — resets `running` cycles whose all children are already terminal to their aggregated status; otherwise re-enqueues them.

## Notification Failure Handling

- In-app persistence is the source of truth for "did we notify"; if `notifRepo.Create` fails, log error, do NOT retry inline, do NOT block child terminal transition (it has already committed).
- Pushover send is fire-and-forget in a goroutine; errors logged only.
- Admin-failure notification (FR-012) is a best-effort second notification; failure to persist it never affects the child or the owner notification.
- The existing `NotifyWishlistUnavailable` per-coin call and the new `NotifyAvailabilityRunTerminal` per-run call are independent invocations with independent failure handling; a failure in one MUST NOT block, retry, or skip the other (D6).

## §17 Quality Gate & §21 Definition of Done

**Quality Gate (§17) — this feature-specific execution plan**:

1. `go vet ./...` clean.
2. `go test ./architecture_test.go` still passes (no cross-layer imports).
3. Targeted: `go test ./services/... -run Availability`; `go test ./repository/... -run Availability`; `go test ./handlers/... -run Availability`.
4. Full: `go test ./...` before PR.
5. `npm run type-check` + `npx vitest run src/pages/__tests__/WishlistAvailabilityHistoryPage.test.ts src/components/admin/schedules/__tests__/AdminAvailabilitySchedule.test.ts`.
6. Manual smoke: trigger owner, scheduled, and admin cycles against a 2-user local DB; verify notifications, retention, and admin-failure double notify.

**Definition of Done (§21)** — the 15-item checklist in `.github/pull_request_template.md` applies. Feature-specific attention items:

- [ ] Spec citations in PR description: `FR-002, FR-003, FR-010, FR-011, FR-018, FR-021`.
- [ ] Constitution citations: `Principle I, Principle V, §17, §21`.
- [ ] Schema migration additive-only (new table + nullable column verified to leave all pre-existing rows unchanged, run twice for idempotency of `AutoMigrate`).
- [ ] Boot-time invariant on new-row `UserID = 0` logged (legacy rows excluded from the check).
- [ ] No URL/query text in any user-visible message (regex test).
- [ ] Both `wishlist_unavailable` and `wishlist_availability_run` notifications verified to coexist for the same run (SC-009).
- [ ] `Co-authored-by: Copilot` trailer on every commit.

## Complexity Tracking

*(empty — no constitutional deviations)*

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|--------------------------------------|
| —         | —          | —                                    |
