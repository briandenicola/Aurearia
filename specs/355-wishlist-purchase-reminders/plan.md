# Implementation Plan: Wishlist Purchase Reminders

**Branch**: `355-wishlist-purchase-reminders` | **Date**: 2026-08-20 | **Spec**: `./spec.md`
**Input**: Feature specification from `specs/355-wishlist-purchase-reminders/spec.md`

## Summary

Add date-based purchase reminders to wishlist coins. A new `purchase_reminders` table stores one active reminder per coin per user with a date-only `remind_date` and IANA timezone snapshot. A daily scheduler evaluates due dates in each reminder's timezone, fires in-app notifications (+ best-effort Pushover), and marks reminders as notified. Reminders auto-cancel when a coin exits the wishlist. The frontend surfaces creation/edit via an inline modal on the coin detail page and a badge on wishlist card items. No new routes, no run-history table, no admin endpoints.

**Scope expansion (2026-08-20, post-implementation)**: Admin Settings → Schedules must expose enable/disable and start-time controls for the purchase reminder scheduler, following the Coin of Day schedule pattern. No Run Now or run-history (locked out-of-scope). See D12.

## Technical Context

**Language/Version**: Go 1.26.6 (API), TypeScript 5 / Vue 3 (web).
**Primary Dependencies**: Gin, GORM, SQLite; existing `NotificationService`, `PushoverService`, `CoinService`, `Scheduler` interface, `SchedulerRegistry`.
**Storage**: SQLite via GORM `AutoMigrate`. One new table (`purchase_reminders`), two new settings constants.
**Testing**: `go test ./...` (architecture + unit + service + repository + handler), `npx vue-tsc --noEmit` + `npx vitest` for frontend.
**Target Platform**: Linux server (single instance), PWA client.
**Project Type**: Web application (Go API + Vue SPA).
**Performance Goals**: No regression; scheduler cycle completes in < 5s for < 500 active reminders.
**Constraints**: Additive DDL only. No destructive migration. No new Python agent work.
**Scale/Scope**: <= 50 users, <= ~500 wishlist coins total, <= ~200 active reminders.

## Constitution Check

- **Principle I (Layered Architecture)**: All backend changes follow Handler -> Service -> Repository -> Database. `PurchaseReminderHandler` (thin), `PurchaseReminderService` (business logic + auto-cancel), `PurchaseReminderRepository` (GORM queries), `PurchaseReminder` model. Enforced by `architecture_test.go`.
- **Principle IV (Simple Complete Proportional)**: Reuses existing `Scheduler` interface, `SchedulerRegistry`, `NotificationService.CreateNotification` pattern, `PushoverService`. One new table, one new scheduler, one new handler — proportional to feature scope.
- **Principle V (Security, Privacy)**: All endpoints are `protected` JWT-scoped. Owner-scoping via `OwnedBy` scope on coin lookup + user_id on reminder queries. No internal errors leak to clients.
- **Principle VI (Consistent UX)**: Frontend reuses existing modal pattern, `.chip-sm` badge, design tokens. PWA/mobile 44px+ tap targets.
- **Principle IX (Architecture Tests)**: Existing `architecture_test.go` covers new packages automatically if they follow the import rules.
- **SS17 Quality Gate**: Targeted `go test` on new packages; `vue-tsc` + vitest for frontend; full `go test ./...` and `npm run build` before merge.
- **SS21 Definition of Done**: See checklist in tasks.md.

No violations. No ADR required.

## Design Decisions

### D1. Separate `purchase_reminders` table

Per approved design decision D1. Cleaner lifecycle and audit trail vs. columns on `Coin`. Supports future recurrence if needed.

### D2. Status-based lifecycle (pending / notified / cancelled)

The `status` column replaces the design-review's `IsNotified` + `IsCancelled` booleans for clearer state machine semantics and simpler queries. Transitions: `pending -> notified` (scheduler fires), `pending -> cancelled` (user delete or auto-cancel), `notified -> cancelled` (user delete or auto-cancel). The `notifiedAt` and `cancelledAt` timestamps record when each transition occurred.

### D3. IANA timezone snapshot on create

The browser supplies its IANA timezone (via `Intl.DateTimeFormat().resolvedOptions().timeZone`) on every POST. The server validates with `time.LoadLocation` and stores it on the reminder row. The scheduler evaluates `remindDate <= today` in that timezone, not server-local time. This resolves the design-review's "Unresolved #1" (timezone semantics).

### D4. No `Note` field in MVP

Per approved scope exclusion. The design-review model included a `Note` field — this is deferred. Keeps the model and UI simpler.

### D5. Upsert semantics via service-layer check

`POST /coins/:id/reminder` checks for an existing active reminder. If found, updates `remindDate`, `timezone`, resets `status` to `pending` (clearing `notifiedAt`). If not found, creates. This avoids a partial unique index (SQLite limitations) and uses the same transaction for atomicity.

### D6. Auto-cancel hook in CoinService.updateCoin

Inside the existing `CoinService.updateCoin` transaction, after the coin update succeeds, if `IsWishlist` was `true` and is now `false` (checked via `existing.IsWishlist && !updates.IsWishlist` when the field is in `updateFields`), call `PurchaseReminderRepository.CancelActiveForCoin(tx, coinID, userID)`. This keeps the cancel atomic with the wishlist transition. The repository method uses `.WithTx(tx)` for transactional safety.

### D7. Scheduler follows existing patterns exactly

`ReminderScheduler` implements the `Scheduler` interface. Constructor takes `PurchaseReminderRepository`, `NotificationService`, `SettingsService`, `Logger`. Uses `sync.Once` in `Stop()`. Checks `ReminderCheckEnabled` setting in `runCycle()`. Queries `SELECT * FROM purchase_reminders WHERE status = 'pending'`, then for each evaluates `remindDate <= today` in the stored timezone. Fires notification + Pushover, updates status to `notified` in a per-reminder transaction.

### D8. Scheduler concurrency / idempotency boundaries

The scheduler processes reminders sequentially within a single goroutine (matching CoinOfDay pattern). Each reminder notification is wrapped in its own DB transaction: load reminder (re-check `status=pending`), create notification, update `status=notified` + `notifiedAt`. If the transaction fails, the reminder stays `pending` and will be retried next cycle. The `status=pending` re-check inside the transaction is the durable idempotency gate — safe across restarts.

### D9. `ReminderCheckEnabled` defaults to `"true"`

Per approved design decision D6. Unlike admin-gated features (availability, valuation), purchase reminders are user-initiated and have no external API cost, so the scheduler is enabled by default.

### D10. Notification linkage

Notification `type = "purchase_reminder"`, `referenceId = reminder.ID`, `referenceUrl = "/coin/{coinId}"`. The frontend's existing notification-click handler already navigates to `referenceUrl` when present — no new routing logic needed. This resolves the design-review's "Unresolved #3" (deep link behavior): each reminder produces one notification per coin, so there is no "grouped" ambiguity.

### D11. Frontend modal reuses existing patterns

A `PurchaseReminderModal.vue` component uses `<dialog>` with focus trap (matching existing modals). The coin detail page's header actions gain a bell/clock icon that opens the modal. The wishlist `CoinCard` gains a `.chip-sm` badge ("Due Aug 25" / "Due Today"). API calls use `src/web/src/api/client.ts` Axios instance. No new Pinia store — reminder state is fetched per-coin and per-list via composable.

### D12. Admin Schedule UI for purchase reminders (FR-015a)

A new `AdminPurchaseReminderSchedule.vue` component follows the `AdminCoinOfDaySchedule.vue` structure but omits the Run Now button and run-history table (locked out-of-scope). The component binds an enable/disable toggle to `settings.ReminderCheckEnabled` and a `<input type="time">` to `settings.ReminderCheckStartTime`, with a save button that emits the parent's `save` event. `AdminSchedulesSection.vue` mounts it alongside the existing schedule components. The `AppSettings` TypeScript interface gains optional `ReminderCheckEnabled` and `ReminderCheckStartTime` properties. No backend changes required — `settings_service.go` already defines the constants and defaults; the admin settings PUT endpoint persists arbitrary key-value pairs. Disabling the scheduler suppresses notification delivery only; user CRUD on reminders is unaffected.

## Data Model

### New Table: `purchase_reminders`

```go
// PurchaseReminder records a user's intent to be reminded about a wishlist coin
// on a specific date. One active (pending/notified) reminder per (coin, user).
type PurchaseReminder struct {
    ID          uint       `gorm:"primaryKey" json:"id"`
    CoinID      uint       `gorm:"not null;index:idx_reminder_coin_user,unique,where:status IN ('pending','notified')" json:"coinId"`
    Coin        Coin       `gorm:"foreignKey:CoinID" json:"-"`
    UserID      uint       `gorm:"not null;index:idx_reminder_coin_user,unique,where:status IN ('pending','notified')" json:"userId"`
    User        User       `gorm:"foreignKey:UserID" json:"-"`
    RemindDate  string     `gorm:"type:varchar(10);not null;index" json:"remindDate"`
    Timezone    string     `gorm:"type:varchar(64);not null" json:"timezone"`
    Status      string     `gorm:"type:varchar(20);not null;default:'pending';index" json:"status"`
    NotifiedAt  *time.Time `json:"notifiedAt,omitempty"`
    CancelledAt *time.Time `json:"cancelledAt,omitempty"`
    CreatedAt   time.Time  `json:"createdAt"`
    UpdatedAt   time.Time  `json:"updatedAt"`
}
```

**Notes**:
- `RemindDate` stored as `varchar(10)` (`YYYY-MM-DD`) rather than `time.Time` to avoid SQLite date-type ambiguity and make timezone evaluation explicit.
- `Timezone` is the IANA zone string (e.g., `America/Chicago`), validated on write via `time.LoadLocation`.
- `Status` enum: `pending`, `notified`, `cancelled`. Active reminders = `status IN ('pending', 'notified')`.
- SQLite does not support partial unique indexes natively. The uniqueness constraint `(coin_id, user_id)` for active reminders is enforced at the service layer (upsert check) rather than via DDL. GORM index tags on the struct serve as documentation.
- `CancelledAt` records when auto-cancel or manual cancel occurred.

### Settings Constants (added to `services/settings_service.go`)

| Constant | Key | Default | Description |
|----------|-----|---------|-------------|
| `SettingReminderCheckEnabled` | `ReminderCheckEnabled` | `"true"` | Enable/disable the reminder scheduler |
| `SettingReminderCheckStartTime` | `ReminderCheckStartTime` | `"08:00"` | Daily start time (HH:MM, 24h) |

No interval setting — the scheduler runs once per day (1440 min fixed), matching CoinOfDay.

### Notification Type

Add `NotificationTypePurchaseReminder = "purchase_reminder"` to `services/notification_service.go`.

## Project Structure

### Source Code (new/modified files)

```text
src/api/
  models/purchase_reminder.go           # NEW — PurchaseReminder model
  repository/purchase_reminder_repository.go  # NEW — CRUD + active queries
  services/purchase_reminder_service.go       # NEW — business logic, upsert, cancel
  services/reminder_scheduler.go              # NEW — daily scheduler
  services/settings_service.go                # MOD — add 2 constants + defaults
  services/notification_service.go            # MOD — add NotifyPurchaseReminder method
  services/coin_service.go                    # MOD — auto-cancel hook in updateCoin
  handlers/purchase_reminder_handler.go       # NEW — thin HTTP handler
  database/database.go                        # MOD — add PurchaseReminder to AutoMigrate
  deps.go                                     # MOD — wire repo, service, scheduler, handler
  routes_protected.go                         # MOD — register 4 routes

src/web/src/
  api/client.ts                               # MOD — add reminder API methods
  types/coin.ts                               # MOD — add PurchaseReminder type
  composables/usePurchaseReminder.ts          # NEW — fetch/create/delete reminder
  components/coin/PurchaseReminderModal.vue   # NEW — date picker modal
  components/CoinCard.vue                     # MOD — add reminder badge
  pages/CoinDetailPage.vue                    # MOD — add reminder action + section
  pages/NotificationsPage.vue                 # MOD — handle purchase_reminder click
```

## Observability

- Scheduler logs at Info level: "Reminder scheduler started", "Processing N due reminders", "Reminder {id} notified for user {userId}", "Reminder scheduler cycle complete".
- Scheduler logs at Warn level: Pushover send failure.
- Scheduler logs at Debug level: "Reminder check disabled, skipping cycle", individual timezone evaluation details.
- Auto-cancel logs at Info level: "Auto-cancelled reminder {id} for coin {coinId} (wishlist exit)".

## Migration Compatibility

- `AutoMigrate` adds the `purchase_reminders` table. No existing tables are altered.
- New settings keys are read-on-demand with defaults — no seed migration needed.
- Rollback: drop the `purchase_reminders` table and remove code. No other tables affected.

## Complexity Tracking

No constitution violations. No waivers needed.
