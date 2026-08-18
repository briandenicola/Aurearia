# Feature Specification: Wishlist Availability Run Observability

**Feature Branch**: `353-wishlist-availability-run-observability`
**Created**: 2026-08-17
**Status**: Draft
**Input**: User description: "Give owners and admins a truthful, per-user history of every wishlist availability check, with a parent/child model for admin-triggered cycles and a notification on every completed run."

## Context and Scope

Aurearia already has a working wishlist availability checker (see `src/api/services/availability_service.go`, `src/api/services/availability_scheduler.go`, `src/api/models/availability_check.go`, `src/api/repository/availability_repository.go`, `src/api/handlers/availability.go`, and `src/web/src/components/admin/schedules/AdminAvailabilitySchedule.vue`). It performs URL health checks, escalates ambiguous 200-OK responses to the Python agent, updates `Coin.ListingStatus`, and creates an in-app `Notification` **only on new `unavailable` verdicts**. Admin-triggered runs create a single orchestration row with `UserID = 0` and mix all users' results together, so the admin run history cannot faithfully be attributed to an owner and owners cannot see admin-triggered runs in their own timeline.

This feature refactors that observability layer into a **parent cycle + per-user child run** model, adds bounded per-owner retention, and guarantees a terminal-outcome notification on every completed child run. The URL-checking mechanics, agent escalation flow, and Python agent contract are **unchanged**. Wishlist Search Alerts (spec 337, `WishlistSearchAlert` / `AlertRun`) remain a fully separate subsystem and are out of scope.

## User Scenarios & Testing *(mandatory)*

### User Story 1 — Owner sees every availability run in their own history (Priority: P1)

As a wishlist owner, I need my run history to include every check performed on my wishlist coins (scheduled, my own manual, or admin-triggered), each represented as one row I own, so I can trust the timeline and drill into any run's per-coin results.

**Why this priority**: This is the core observability gap. Without it, admin-triggered checks are invisible in owner history and the current `UserID=0` row cannot be attributed.

**Independent Test**: With three users A, B, C and wishlist URLs for each, trigger one scheduled run, one owner-initiated run for A, and one admin-triggered cycle. Each user's `/wishlist/availability-runs` list must contain exactly the runs that touched their own coins. No row is owned by "user 0" or displayed cross-owner.

**Acceptance Scenarios**:

1. **Given** the scheduler fires a scheduled cycle over 3 users, **When** each user opens their availability history, **Then** each user sees exactly one new child run whose `triggerType="scheduled"`, whose owner is themselves, and whose counts sum over only their own coins.
2. **Given** owner A triggers `POST /wishlist/check-availability`, **When** the run completes, **Then** A has exactly one new owner-triggered child run with `triggerType="owner"`, `cycleId=null`, and no other user's history is affected.
3. **Given** an admin triggers `POST /admin/availability/run`, **When** the cycle completes across 3 users, **Then** each of the 3 users sees exactly one new child run with `triggerType="admin"` and the same `cycleId`; the admin sees the parent cycle in the admin history.

---

### User Story 2 — Admin sees a single parent row per admin-triggered cycle with roll-up counts (Priority: P1)

As an admin, I need admin-triggered checks to appear as one parent cycle row with aggregated status and counts across all children, plus the ability to expand into each per-user child run.

**Why this priority**: Without a parent row the admin cannot see a single "did this cycle succeed" answer and cannot rate-limit duplicate manual triggers coherently.

**Independent Test**: Trigger an admin cycle over 3 users where one user's child fails. Admin sees one parent row whose status is `partial_failure`, whose `childCounts` show 2 completed + 1 failed, and whose expanded view lists all three child runs with their individual outcomes.

**Acceptance Scenarios**:

1. **Given** an admin triggers a cycle when no cycle is running, **When** the API responds, **Then** the response is `202 Accepted` with `{ cycleId, status: "queued" }` and a parent cycle row exists.
2. **Given** an admin triggers a second cycle within 5 minutes of an existing queued or running cycle, **When** the API responds, **Then** the response is `409 Conflict` with a generic "cycle already queued or running" message and no new parent or child rows are created.
3. **Given** an admin cycle finishes with all children succeeding, **When** the parent aggregation runs, **Then** the parent status is `completed` and `childCounts.completed == totalChildren`.
4. **Given** at least one child fails, **When** aggregation runs, **Then** parent status is `partial_failure` if any child completed and `failed` if all children failed.

---

### User Story 3 — Every terminal run notifies the owner, in addition to existing per-coin alerts (Priority: P1)

As a wishlist owner, I want a notification for every completed availability check on my wishlist — including checks that surfaced zero URLs and checks where nothing changed — so I always know the system is doing its job on my behalf. This is **additive**: the existing per-coin "this coin just went unavailable" notification keeps firing exactly as it does today.

**Why this priority**: Brian's explicit binding decision (#3), reaffirmed during review: keep existing per-coin unavailable alerts *in addition to* the new per-run notification. Silence about a completed run is currently indistinguishable from failure; that gap is closed without removing or suppressing any existing alert.

**Independent Test**: Give owner A a wishlist with 2 URLs both still available. Trigger a run. A receives exactly one in-app notification and (if Pushover is enabled) one Pushover push describing the zero-change outcome. Separately, give owner B a wishlist with 1 URL that transitions to unavailable during a run; B receives both the existing per-coin `wishlist_unavailable` notification for that coin AND the new per-run `wishlist_availability_run` notification for that run — neither is suppressed.

**Acceptance Scenarios**:

1. **Given** a child run completes with `coinsChecked=0` (owner has no URLs), **When** the run reaches terminal state, **Then** the owner receives one in-app notification `type="wishlist_availability_run"` summarizing "no wishlist URLs to check" and one Pushover push if enabled.
2. **Given** a child run completes with counts unchanged from the previous run, **When** it reaches terminal state, **Then** the owner still receives exactly one in-app notification for this run (no daily dedupe).
3. **Given** a child run fails, **When** it reaches terminal state, **Then** the owner receives one in-app notification whose message is a generic safe failure string (no URLs, no query terms, no raw error text) and one Pushover push if enabled.
4. **Given** the run was `triggerType="admin"` and the child failed, **When** notifications dispatch, **Then** the triggering admin also receives one in-app notification (and Pushover if enabled) identifying the affected owner by username and the parent `cycleId`, but no URL or reason text beyond the generic failure label.
5. **Given** Pushover send fails, **When** the notification pipeline runs, **Then** the in-app notification is still persisted, the child run's terminal `status` is unchanged, and the Pushover failure is logged server-side only.
6. **Given** one or more coins in a run transition from available/unknown to `unavailable`, **When** the run reaches terminal state, **Then** the owner receives one `wishlist_unavailable` notification per newly-unavailable coin (existing behavior, unchanged) **and** exactly one `wishlist_availability_run` notification for the run itself, whose summary text may reference the newly-unavailable coins by name but which does not replace or suppress the per-coin notifications.

---

### User Story 4 — Bounded retention keeps history useful and small (Priority: P2)

As Brian, I want per-owner and cycle history to be bounded so the DB stays small and the UI stays fast, without ever destroying a currently-running or queued run.

**Why this priority**: Explicit binding decision (#2), reaffirmed during review: keep the last 20 terminal child runs per user, and retain exactly the last 20 completed parent cycles globally. Prevents unbounded growth from the new per-user rows.

**Independent Test**: For a user with 25 terminal child runs across trigger types, the next completed run must prune the oldest so that exactly 20 terminal child runs remain. Non-terminal (queued/running) rows are never pruned. Separately, with 25 terminal parent cycles system-wide, the next cycle to reach a terminal status prunes the oldest so exactly 20 terminal parent cycles remain globally.

**Acceptance Scenarios**:

1. **Given** owner A has 20 terminal child runs, **When** a new run completes, **Then** the oldest terminal child run for A (and its `AvailabilityResult` rows) is deleted and A has 20 terminal child runs.
2. **Given** 20 terminal parent cycles exist globally, **When** a new parent cycle reaches a terminal aggregated status, **Then** the oldest terminal parent cycle is deleted **only after** its `cycle_id` reference on any surviving (non-terminal-pruned) child runs is nulled out so their history remains readable; queued/running cycles and their children are never pruned; no `AvailabilityResult` row is ever deleted or reassigned by cycle pruning.

---

### User Story 5 — Legacy history remains readable without synthetic migration (Priority: P2)

As Brian, I do not want to lose the existing admin history, and I do not want the system inventing parent cycles, retagging trigger types, or moving results for data that predates this feature.

**Why this priority**: Explicit binding clarification from the review block: leave legacy `UserID=0` admin aggregate runs readable as legacy rows; do not synthesize parent cycles, retag them, move results, or destructively backfill them. Only new post-deploy admin/scheduled runs create `AvailabilityCycle` parents.

**Independent Test**: After this feature ships, pre-existing `AvailabilityRun` rows with `UserID = 0` remain unchanged in the database (same `TriggerType`, same `CycleID = NULL`, same results) and remain visible through the existing legacy admin run endpoints (`GET /admin/availability-runs*`), now carrying a clear "Legacy" label in the admin UI. No `AvailabilityCycle` row is created for them. The first admin- or scheduler-triggered run *after* deployment creates a real `AvailabilityCycle` parent with real children as described in User Story 2.

**Acceptance Scenarios**:

1. **Given** pre-existing `AvailabilityRun` rows with `UserID = 0`, **When** this feature's schema migration runs, **Then** those rows are left completely unchanged (no new `CycleID` value, no `TriggerType` change, no new `AvailabilityCycle` row created for them) and remain queryable via the existing legacy admin endpoints.
2. **Given** a pre-existing `AvailabilityRun` row with `UserID > 0`, **When** the schema migration runs, **Then** its `CycleID` is left as its default (`NULL`, since the column is newly added) and none of its fields are otherwise modified.
3. **Given** any pre-existing row of either shape, **When** the schema migration runs, **Then** no `AvailabilityResult` row is deleted, reassigned, or modified, and no `AppSetting` migration-version flag is required because there is no data transformation to gate.
4. **Given** the admin views run history after deployment, **When** the list mixes legacy `UserID=0` rows and new parent cycles, **Then** the UI visually distinguishes legacy rows with a "Legacy" label/fallback so admins are not misled into thinking they are new parent cycles.

---

### Edge Cases

- **Empty wishlist**: A scheduled or admin child run against a user with zero URLs is still recorded, still terminates as `completed` with `coinsChecked=0`, and still notifies once.
- **Duplicate admin trigger**: Two admins clicking "Run Now" within the 5-minute window: the second returns `409 Conflict` and does not create a phantom parent.
- **Process crash mid-cycle**: On startup, `RecoverStaleRuns` moves stuck `running` children back to `queued`. A parent cycle with all children in terminal state becomes terminal on next aggregation tick.
- **Owner deleted mid-cycle**: If a user is deleted after a child is queued, the child fails with a generic "owner unavailable" message and the parent aggregates it as failed.
- **Notification delivery partial failure**: In-app persistence is the source of truth; Pushover failure never changes run status or blocks the terminal transition.
- **Concurrent scheduled + admin cycle**: If a scheduled cycle is in flight when admin triggers, admin trigger returns `409 Conflict`. Only one active cycle-level orchestration at a time.
- **Retention race with active run**: Retention pruning ignores non-terminal rows and runs post-commit; it never deletes a row referenced by an in-flight cycle.
- **Reference URL contains sensitive query strings**: URLs are logged internally at debug only; they never appear in notification titles, messages, or the parent's public status text.

## Requirements *(mandatory)*

### Functional Requirements

**Data model & ownership**

- **FR-001**: System MUST introduce an `AvailabilityCycle` entity representing an admin-triggered or scheduled parent orchestration. `AvailabilityCycle` is admin/operational metadata only; it MUST NOT own any `AvailabilityResult` directly for cycles created after this feature ships.
- **FR-002**: System MUST require every new `AvailabilityRun` (child) created after this feature ships to have `UserID > 0` and MUST reject creation of such a child run with `UserID = 0`. Pre-existing `UserID = 0` rows are historical data and are exempt (see FR-021, User Story 5).
- **FR-003**: Each new child `AvailabilityRun` MUST reference at most one parent cycle via nullable `CycleID`. Owner-initiated runs (`triggerType="owner"`) MUST have `CycleID = NULL`. There is no separate `scope` field — `triggerType` alone identifies how a run was created.
- **FR-004**: `AvailabilityResult` rows MUST always belong to a child `AvailabilityRun` (existing FK preserved). Aggregated counts on the parent MUST be derived from its children, never from results directly.

**Cycle lifecycle**

- **FR-005**: System MUST create exactly one `AvailabilityCycle` per scheduled tick and per admin trigger, with `triggerType` in {"scheduled","admin"} and, for admin, `triggerUserId` set to the initiating admin.
- **FR-006**: When admin or scheduled cycle starts, system MUST fan out one child `AvailabilityRun` per user who has at least one wishlist coin with a reference URL, plus one child per user who has zero URLs so every affected owner sees one notification per cycle. Rationale: owners must not be silenced simply because their wishlist has no URLs today.
- **FR-007**: System MUST reject a second admin cycle while any cycle is in `queued` or `running` status with `409 Conflict` and a generic message. Duplicate protection MUST use an atomic DB check-and-insert equivalent to the existing `EnqueueManualRun` transaction.
- **FR-008**: Parent `status` MUST aggregate deterministically from children: `queued` if any child queued and none running; `running` if any child running; `completed` if all children `completed`; `failed` if all children `failed`; `partial_failure` if at least one child `completed` and at least one `failed`. Aggregation MUST run atomically on each child terminal transition.
- **FR-009**: System MUST expose `childCounts` on the parent: `{ total, queued, running, completed, failed }`.

**Notifications (binding decisions #3, #4, #5)**

- **FR-010**: System MUST create exactly one in-app `Notification` per terminal child run, with `type="wishlist_availability_run"`, `referenceId = availabilityRun.id`, and `referenceUrl = "/wishlist/availability-runs/{id}"`. This applies to every child run including runs with `coinsChecked=0` and runs with no changes.
- **FR-011**: System MUST, best-effort, send one Pushover push per terminal child notification when the owner has `PushoverEnabled=true` and a non-empty `PushoverUserKey`. Pushover send MUST run in a goroutine and MUST NOT change the child run's terminal status if it fails.
- **FR-012**: On admin-triggered child failure, system MUST additionally create one in-app notification for the triggering admin (and Pushover if enabled) identifying the affected owner by username and the parent `cycleId`. No URLs, no reason text beyond a generic safe label.
- **FR-013**: System MUST NOT dedupe run-outcome notifications by day, hour, or content hash; every terminal child run produces exactly one owner notification.
- **FR-014**: System MUST NOT suppress, replace, or otherwise modify the existing per-coin `NotifyWishlistUnavailable` (`type="wishlist_unavailable"`) notification. It continues to fire for every coin that transitions to `unavailable` within a run, exactly as it does today, in addition to — never instead of — the per-run `wishlist_availability_run` terminal notification (FR-010). The run-outcome notification message MAY reference newly-unavailable coins by name (up to 3 names + "and N more") as a summary, but this is purely additional context and MUST NOT gate, skip, or replace the per-coin notification call.

**Security, privacy, error messages**

- **FR-015**: All user-visible failure messages (notification body, API `failMessage`, admin cycle summary text) MUST be generic (e.g., "Availability check failed. Server logs contain the technical detail."). Internal error text, URLs, query strings, and coin reference URLs MUST NOT appear in any owner- or admin-visible message.
- **FR-016**: Full technical detail (raw error, URL under check, HTTP status) MUST be logged server-side via the existing `Logger` with the run and cycle IDs as correlation keys.
- **FR-017**: Every read endpoint MUST enforce ownership: `/wishlist/availability-runs*` returns only rows where `AvailabilityRun.UserID = auth.userId`; admin-only endpoints (`/admin/availability-cycles*`) MUST require the admin JWT claim.

**Retention (binding decision #2)**

- **FR-018**: After every child run terminal transition, system MUST prune terminal child runs for that owner so at most 20 remain. Pruning MUST cascade-delete the pruned children's `AvailabilityResult` rows. Queued and running children MUST NEVER be pruned.
- **FR-019**: After every parent cycle terminal transition, system MUST prune terminal parent cycles so at most 20 remain across the system. Pruning a parent whose children still exist MUST first null-out `AvailabilityRun.CycleID` on any surviving child runs (so history remains readable); no `AvailabilityResult` is deleted by cycle pruning.
- **FR-020**: The existing `PruneOldRuns(100)` global limit MUST be removed and replaced by FR-018 + FR-019.

**Schema migration (additive only — binding clarification)**

- **FR-021**: System MUST ship the smallest additive schema migration that supports this feature and MUST NOT synthesize, backfill, retag, or move any pre-existing data:
  (a) adds a new `AvailabilityCycle` table;
  (b) adds a nullable `AvailabilityRun.CycleID` column;
  (c) leaves every pre-existing `AvailabilityRun` row (both `UserID = 0` legacy admin rows and `UserID > 0` rows) completely unchanged — no `CycleID` backfill beyond the column's natural `NULL` default, no `TriggerType` retagging, no synthetic `AvailabilityCycle` row created on their behalf;
  (d) does NOT delete, reassign, or modify any `AvailabilityResult` row;
  (e) requires no `AppSetting` migration-version flag and no ADR, because there is no data transformation to gate or waive — the migration is purely additive DDL (new table + new nullable column), consistent with GORM `AutoMigrate` semantics already used elsewhere in `database/database.go`.
- **FR-021a**: Legacy `UserID = 0` admin rows MUST remain visible through the existing `GET /admin/availability-runs*` endpoints, unmodified, and the admin UI MUST render them with a clear "Legacy" label so they are not confused with new `AvailabilityCycle` parent rows.

**API surface (see plan.md for exact routes)**

- **FR-022**: System MUST expose `GET /wishlist/availability-runs?page=&limit=` returning only the caller's child runs, newest first, with generic aggregates.
- **FR-023**: System MUST expose `GET /wishlist/availability-runs/{id}` returning the caller's single child run with per-coin results.
- **FR-024**: System MUST expose `GET /admin/availability-cycles?page=&limit=` returning parent cycles with roll-up counts.
- **FR-025**: System MUST expose `GET /admin/availability-cycles/{id}` returning a parent with its child list (each child summary + owner username), but NOT per-coin results (admins drill into `/admin/availability-runs/{id}` which is retained).
- **FR-026**: Existing `POST /admin/availability/run` MUST continue to work but now returns `{ cycleId, status }` instead of `{ runId, status }`.
- **FR-027**: Existing `POST /wishlist/check-availability` (owner-initiated) MUST continue to create a single child run with `CycleID = NULL`, `triggerType = "owner"`.

**Non-goals** *(explicit)*

- **NG-1**: No change to the Python agent contract (`app/models/requests.py` `AvailabilityCheckRequest`, `MAX_AVAILABILITY_ITEMS`).
- **NG-2**: No change to the URL-checking mechanics (`CheckURL`, keyword lists, rate limiting, batching).
- **NG-3**: Wishlist Search Alerts (`WishlistSearchAlert`, `AlertRun`, `AlertCandidate`) remain separate; not touched.
- **NG-4**: No new admin scheduling controls (existing enable / start-time / interval remain).
- **NG-5**: No historical backfill, retagging, or reparenting of legacy `UserID = 0` rows; they remain exactly as they are today, labeled "Legacy" in the admin UI.
- **NG-6**: No new metrics/telemetry pipeline; existing `Logger` is sufficient.
- **NG-7**: No suppression of the existing per-coin `wishlist_unavailable` notification; it is not modified by this feature.

### Key Entities

- **AvailabilityCycle** (NEW): One row per admin-triggered or scheduled orchestration created *after* this feature ships. Fields: `id`, `triggerType` in {"scheduled","admin"}, `triggerUserId` (nullable), `status` in {"queued","running","completed","failed","partial_failure"}, `childCounts` (JSON or 5 int columns), `startedAt`, `completedAt`, `failMessage` (generic only), `createdAt`. There is no `legacy_cycle` value: legacy rows are never converted into `AvailabilityCycle` rows (see FR-021, User Story 5).
- **AvailabilityRun** (MODIFIED): Adds `CycleID *uint` (nullable FK to `AvailabilityCycle`, default `NULL`). `triggerType` values for *new* rows are {"owner","scheduled","admin"}. New-row constraint: `UserID > 0`. Pre-existing rows (including legacy `UserID = 0` rows) keep their historical `TriggerType` values and are left unchanged by the migration. Retained fields, counts, and `Results` relation unchanged.
- **AvailabilityResult** (UNCHANGED).
- **Notification** (REUSED): New `type` value `wishlist_availability_run`, additive. Existing `wishlist_unavailable` type is unmodified and continues to fire independently — the two notification types coexist for the same run whenever a coin transitions to unavailable.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% of `AvailabilityRun` rows created after this feature ships have `UserID > 0`; verified by unit + integration test and by a boot-time invariant assertion that only inspects newly-created rows.
- **SC-002**: For every terminal child run, exactly one in-app `Notification` of `type="wishlist_availability_run"` exists (verified by service-level test across all 4 trigger types × {zero URLs, no changes, changes, failure}).
- **SC-003**: Duplicate admin trigger inside the 5-minute window is rejected with `409` in 100% of test cases; zero orphan parent cycles created.
- **SC-004**: Per-owner terminal child run count is `≤ 20` after every terminal transition; global terminal parent cycle count is `≤ 20`; queued/running rows never pruned (verified by retention test).
- **SC-005**: Parent aggregation is correct in 100% of test scenarios covering `completed`, `failed`, `partial_failure`, and `running` transitions across ≥ 3 children.
- **SC-006**: Legacy `UserID = 0` admin rows remain readable, byte-for-byte unchanged, through the existing admin endpoints after the schema migration ships; zero pre-existing `AvailabilityRun` or `AvailabilityResult` rows are modified, deleted, or reassigned (row-count and content assertion pre/post).
- **SC-007**: No user-visible message (notification body, API `failMessage`, admin cycle summary) contains a URL, query string, or raw error text (regex-based test over sample outputs of every failure path).
- **SC-008**: Pushover failure during a run terminal transition does not change the child's `status` or block completion (verified by test with fake Pushover client returning error).
- **SC-009**: For every run in which at least one coin transitions to `unavailable`, both the per-coin `wishlist_unavailable` notification(s) and the single `wishlist_availability_run` terminal notification exist for that run — verified by a regression test that fails if either is missing or if one suppresses the other.

## Assumptions

- The existing `AvailabilityService` URL-check and agent-escalation logic is behaviorally correct and is not being redesigned.
- `NotificationService.NotifyWishlistUnavailable` and `sendPushover` continue to exist, are unmodified in behavior, and are called independently of the new `NotifyAvailabilityRunTerminal` path — neither replaces nor gates the other.
- The existing `RecoverStaleRuns` timeout (15 minutes) is acceptable for children; a matching cycle-level stale recovery aggregates from children rather than a separate timer.
- Frontend surfaces owner history in the existing wishlist area and updates the existing `AdminAvailabilitySchedule.vue` history table to display parent cycles with expandable child rows, plus a "Legacy" label/fallback for pre-existing `UserID = 0` rows.
- OpenAPI/Swagger annotations follow the existing `@Summary`/`@Router` handler pattern; no separate OpenAPI YAML is regenerated by build tooling in this repo.
