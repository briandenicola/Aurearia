# Feature Specification: Wishlist Purchase Reminders

**Feature Branch**: `355-wishlist-purchase-reminders`
**Created**: 2026-08-20
**Status**: Draft
**Owner**: Maximus (Lead / Architect)
**Requested by**: Brian DeNicola
**Input**: Approved design review (decisions.md 2026-08-20) — users set a future date reminder on any wishlist coin; the scheduler fires an in-app notification (+ Pushover) on or after that date; the reminder auto-cancels when the coin exits wishlist.

## Context and Scope

Aurearia's wishlist lets collectors tag coins they intend to buy. Today there is no way to schedule a "remind me to buy this" alert tied to a specific date (e.g., auction close, dealer restock, payday). This feature adds **date-based purchase reminders** with a daily scheduler, in-app + Pushover notifications, automatic lifecycle management, and inline UI on existing wishlist/detail surfaces.

**In scope**: one active reminder per wishlist coin per user; date-only remind-date with IANA timezone snapshot; daily scheduler with catch-up; auto-cancel on wishlist exit; in-app notification deep-linking to coin detail; Pushover best-effort; inline modal/badge on existing surfaces.

**Out of scope** (explicitly excluded per approved design): notes field, recurrence, multiple reminders per coin, manual admin run endpoint, run-history table, dedicated reminders route. Note: "admin gate on user CRUD" means the reminder CRUD endpoints are NOT restricted to admin users — any authenticated user can manage their own reminders regardless of whether the scheduler is enabled or disabled. Disabling the scheduler suppresses delivery only; it does not prevent users from creating, updating, or cancelling reminders.

**Related specs**: 337 (Wishlist Search Alerts), 354 (Run History & Wishlist Featured Coin), 340 (Shipment Tracking).

---

## User Scenarios & Testing

### User Story 1 — Set a purchase reminder on a wishlist coin (Priority: P1)

As a collector viewing a wishlist coin, I want to set a future date when I should be reminded to purchase it, so I don't miss a planned buy.

**Why this priority**: Core creation flow — every other story depends on reminders existing.

**Independent Test**: Open a wishlist coin's detail page, click the reminder action, pick a future date, confirm. Verify the reminder appears in the coin's detail and the wishlist card badge. Re-open the modal and change the date — confirm update-in-place, not duplicate.

**Acceptance Scenarios**:

1. **Given** a wishlist coin with no active reminder, **When** the user POSTs a valid future `remindDate` and `timezone`, **Then** a `PurchaseReminder` row is created with `status=pending`, the response is 201 with the reminder JSON, and the coin detail shows a reminder badge.
2. **Given** a wishlist coin with an existing active reminder, **When** the user POSTs a new `remindDate`, **Then** the existing reminder is updated in place (same ID), status resets to `pending` if it was `notified`, response is 200, and no duplicate row is created.
3. **Given** a non-wishlist coin, **When** the user POSTs a reminder, **Then** the API returns 409 Conflict with `{"error": "Reminders are only available for wishlist coins"}`.
4. **Given** a `remindDate` in the past (evaluated in the supplied timezone), **When** the user POSTs, **Then** the API returns 400 with `{"error": "Remind date must be today or in the future"}`.
5. **Given** an invalid or unrecognized IANA timezone string, **When** the user POSTs, **Then** the API returns 400 with `{"error": "Invalid timezone"}`.

---

### User Story 2 — Receive a notification on the remind date (Priority: P1)

As a collector, when my reminder date arrives I want to receive an in-app notification (and Pushover if enabled) that links directly to the coin, so I can take action.

**Why this priority**: The scheduler is the feature's raison d'etre — without it, reminders are just static data.

**Independent Test**: Create a reminder for today (or a past date). Run the scheduler cycle. Verify: (a) in-app notification with type `purchase_reminder` and `referenceUrl` = `/coin/{coinId}` is created; (b) reminder status transitions to `notified`; (c) Pushover fires if user has it enabled; (d) re-running the cycle does not create a duplicate notification.

**Acceptance Scenarios**:

1. **Given** a `pending` reminder whose `remindDate <= today` (evaluated in the reminder's stored timezone), **When** the scheduler runs, **Then** an in-app notification is created with `type=purchase_reminder`, `referenceId=reminder.ID`, `referenceUrl=/coin/{coinId}`, title "Purchase Reminder", and a message containing the coin name; the reminder `status` becomes `notified` and `notifiedAt` is set.
2. **Given** a reminder with `remindDate` in the future, **When** the scheduler runs, **Then** it is skipped — no notification, no status change.
3. **Given** a `pending` reminder whose `remindDate` is several days in the past (overdue / catch-up), **When** the scheduler runs, **Then** the reminder is still notified (catch-up semantics, not silently skipped).
4. **Given** a `notified` reminder, **When** the scheduler runs again, **Then** no duplicate notification is created — the `status=notified` guard prevents re-processing. Durable across process restarts.
5. **Given** a user with Pushover enabled, **When** the scheduler fires a reminder notification, **Then** a Pushover message is sent with the coin name and a deep link to `/coin/{coinId}`. Pushover failure is logged but does not fail the cycle or prevent the in-app notification.

---

### User Story 3 — Cancel a reminder manually (Priority: P2)

As a collector, I want to cancel a reminder I no longer need, without having to remove the coin from my wishlist.

**Why this priority**: Explicit user control is important but secondary to create + notify.

**Independent Test**: Create a reminder, then DELETE it. Verify the reminder no longer appears in GET responses and the scheduler skips it.

**Acceptance Scenarios**:

1. **Given** an active reminder (pending or notified), **When** the user sends `DELETE /coins/:id/reminder`, **Then** the reminder's `status` becomes `cancelled`, `cancelledAt` is set, response is 204. The reminder no longer appears in `GET /coins/:id/reminder` or `GET /reminders`.
2. **Given** no active reminder for the coin, **When** the user sends DELETE, **Then** the API returns 404.
3. **Given** a cancelled reminder, **When** the user creates a new reminder on the same coin, **Then** a new row is created (the old cancelled row is inert), response is 201.

---

### User Story 4 — Auto-cancel when coin exits wishlist (Priority: P2)

As a collector, when I purchase a wishlist coin (or otherwise remove it from wishlist), the reminder should automatically cancel so I don't get a stale alert.

**Why this priority**: Prevents confusing notifications; core lifecycle integrity.

**Independent Test**: Create a reminder, then update the coin with `isWishlist: false`. Verify the reminder is auto-cancelled. Re-set `isWishlist: true` and create a new reminder — confirm it works fresh.

**Acceptance Scenarios**:

1. **Given** a wishlist coin with an active reminder, **When** the coin's `IsWishlist` transitions to `false` (via any update path), **Then** the reminder's `status` becomes `cancelled` and `cancelledAt` is set within the same transaction.
2. **Given** a wishlist coin with an active reminder, **When** the coin is deleted, **Then** the reminder is cancelled (or cascade-deleted) — no orphan reminder triggers a notification.
3. **Given** a coin that was de-wishlisted and had its reminder auto-cancelled, **When** the coin is re-wishlisted, **Then** there is no active reminder — the user must explicitly create a new one.

---

### User Story 5 — View reminders in context (Priority: P2)

As a collector browsing my wishlist, I want to see which coins have reminders and when they're due, so I can plan my purchases.

**Why this priority**: Visibility makes the feature useful day-to-day.

**Independent Test**: Set reminders on two wishlist coins. Open the wishlist page — verify badges. Open each coin's detail — verify reminder info. Call `GET /reminders` — verify both appear.

**Acceptance Scenarios**:

1. **Given** a wishlist coin with a pending reminder, **When** the wishlist page renders, **Then** the coin card shows a compact badge with the remind date (e.g., "Due Aug 25" or "Due Today").
2. **Given** a wishlist coin with a pending reminder, **When** the coin detail page renders, **Then** a reminder section shows the date and an edit/cancel action.
3. **Given** the user calls `GET /reminders`, **Then** all active (pending + notified) reminders are returned, each including `coinId`, `coinName`, `remindDate`, `status`, `timezone`, and `createdAt`.
4. **Given** a notification with `type=purchase_reminder`, **When** the user clicks it, **Then** the app navigates to `/coin/{coinId}`.

---

### Edge Cases

- Reminder on a coin owned by another user: 404 (owner-scoped queries).
- Concurrent create requests for the same coin: unique constraint on `(coin_id, user_id)` where `status IN ('pending','notified')` prevents duplicates; second request becomes an update.
- Scheduler runs while user is actively editing the reminder: DB-level status check prevents race.
- User changes timezone between create and fire: reminder evaluates against the stored timezone snapshot, not the current browser timezone.
- Coin deletion cascading: service-layer explicit cancel before delete.

---

## Requirements

### Functional Requirements

- **FR-001**: System MUST support creating a purchase reminder on any coin where `IsWishlist=true` and the coin is owned by the requesting user.
- **FR-002**: System MUST enforce one active reminder per `(coin_id, user_id)`. A POST on a coin with an existing active reminder MUST update in place.
- **FR-003**: `remindDate` MUST be a date-only value (no time component). The API MUST accept `YYYY-MM-DD` format.
- **FR-004**: The API MUST accept an IANA timezone string (e.g., `America/Chicago`) and validate it server-side using Go's `time.LoadLocation`. Invalid timezones MUST return 400.
- **FR-005**: `remindDate` MUST be today or in the future, evaluated in the supplied timezone. Past dates MUST return 400.
- **FR-006**: The scheduler MUST run daily and process all `pending` reminders whose `remindDate <= today` (evaluated in each reminder's stored timezone). Overdue reminders MUST be caught up, not skipped.
- **FR-007**: Each fired reminder MUST create exactly one in-app `Notification` with `type=purchase_reminder`, `referenceId=reminder.ID`, `referenceUrl=/coin/{coinId}`.
- **FR-008**: Pushover MUST be sent best-effort when the user has it enabled. Pushover failure MUST NOT prevent the in-app notification or status transition.
- **FR-009**: The reminder `status` MUST transition `pending -> notified` atomically with notification creation. The `notifiedAt` timestamp MUST be set. Re-running the scheduler MUST NOT re-notify.
- **FR-010**: Cancellation (manual DELETE or auto-cancel) MUST set `status=cancelled` and `cancelledAt`. Cancelled reminders MUST NOT appear in active queries.
- **FR-011**: When a coin's `IsWishlist` transitions `true -> false`, all active reminders for that coin MUST be cancelled within the same transaction.
- **FR-012**: When a coin is deleted, its reminders MUST be cancelled or cascade-deleted.
- **FR-013**: `GET /purchase-reminders` MUST return all active reminders for the authenticated user, including `coinId`, `coinName`, `remindDate`, `status`, `timezone`.
- **FR-014**: The scheduler MUST use the existing `Scheduler` interface (`Start`, `Stop`, `RunNow`, `timeUntilNextRun`, `GetStatus`) and register with the `SchedulerRegistry` in `deps.go`.
- **FR-015**: Settings keys `ReminderCheckEnabled` (default `"true"`) and `ReminderCheckStartTime` (default `"08:00"`) MUST follow existing naming conventions and be added to `settings_service.go`.
- **FR-015a**: The Admin Settings → Schedules page MUST include a "Purchase Reminders" section with an enable/disable toggle bound to `ReminderCheckEnabled` and a start-time input bound to `ReminderCheckStartTime`, following the established `AdminCoinOfDaySchedule.vue` pattern. No Run Now button or run-history table (per existing out-of-scope exclusion). Disabling the scheduler suppresses notification delivery only; it MUST NOT prevent users from creating, updating, or cancelling reminders via the API.
- **FR-016**: The frontend MUST surface reminders via an inline modal on the coin detail page and a badge on wishlist card items. No dedicated reminders route.
- **FR-017**: Notification deep-link MUST navigate to `/coin/{coinId}` when clicked.

### Key Entities

- **PurchaseReminder**: `id`, `coinId`, `userId`, `remindDate` (date-only), `timezone` (IANA string), `status` (pending/notified/cancelled), `notifiedAt`, `cancelledAt`, `createdAt`, `updatedAt`. One active per (coin, user). FK to Coin, FK to User.

---

## API Contract

All endpoints under the `protected` route group (JWT required, user-scoped via `userId` from token).

### POST /api/coins/:id/reminder

Create or update a purchase reminder for a wishlist coin.

**Request body**:
```json
{ "remindDate": "2026-09-15", "timezone": "America/Chicago" }
```

**Validation**:
- `remindDate`: required, `YYYY-MM-DD`, must be today-or-future in the given timezone.
- `timezone`: required, valid IANA timezone (validated via `time.LoadLocation`).
- Coin must exist, be owned by the user, and have `IsWishlist=true`.

**Responses**:
- `201 Created`: new reminder created. Body: full `PurchaseReminder` JSON.
- `200 OK`: existing reminder updated in place. Body: full `PurchaseReminder` JSON.
- `400 Bad Request`: invalid date, past date, invalid timezone, or missing fields.
- `404 Not Found`: coin not found or not owned by user.
- `409 Conflict`: coin is not on wishlist.

### GET /api/coins/:id/reminder

Get the active reminder for a specific coin, if any.

**Responses**:
- `200 OK`: body is the active `PurchaseReminder` JSON.
- `404 Not Found`: no active reminder for this coin (or coin not found/not owned).

### DELETE /api/coins/:id/reminder

Cancel the active reminder for a coin.

**Responses**:
- `204 No Content`: reminder cancelled.
- `404 Not Found`: no active reminder (or coin not found/not owned).

### GET /api/reminders

List all active reminders for the current user.

**Response** (`200 OK`):
```json
{
  "reminders": [
    {
      "id": 1,
      "coinId": 42,
      "coinName": "Trajan Denarius",
      "remindDate": "2026-09-15",
      "timezone": "America/Chicago",
      "status": "pending",
      "createdAt": "2026-08-20T15:00:00Z",
      "updatedAt": "2026-08-20T15:00:00Z"
    }
  ]
}
```

The list includes both `pending` and `notified` reminders (not `cancelled`). `coinName` is a join-derived field for frontend display.

---

## Success Criteria

- **SC-001**: A user can set, update, and cancel a reminder on any wishlist coin end-to-end via the UI.
- **SC-002**: The scheduler correctly fires notifications for all due/overdue reminders with zero duplicates across restarts.
- **SC-003**: Auto-cancel triggers reliably on every `IsWishlist -> false` transition path (explicit update, coin delete).
- **SC-004**: All 23+ acceptance tests pass (`go test`, `vitest`, Playwright).
- **SC-005**: No regressions in existing scheduler, notification, or wishlist functionality.

---

## Assumptions

- The server process runs in a single instance (no distributed scheduler coordination needed).
- IANA timezone data is available on the server (`time.LoadLocation` succeeds for standard zones).
- The daily scheduler cadence (once per day) is sufficient precision — sub-day reminders are out of scope.
- Existing `CoinService.updateCoin` transaction is the appropriate hook for auto-cancel on wishlist exit.
- Pushover service (`PushoverService`) is already wired and available for injection.
- The `Scheduler` interface and `SchedulerRegistry` pattern in `deps.go` / `scheduler_contract.go` is the canonical way to add new schedulers.
