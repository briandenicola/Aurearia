---
description: "Task list for feature 355 — Wishlist Purchase Reminders"
---

# Tasks: Wishlist Purchase Reminders

**Input**: Design documents from `specs/355-wishlist-purchase-reminders/`
**Prerequisites**: `spec.md` (required), `plan.md` (required)

**Tests**: Included. Tests-first per constitution SS17 Quality Gate.

## Format: `[ID] [P?] [Story] [Owner] Description`

- **[P]**: Can run in parallel with other [P] tasks (touches different files, no dependencies).
- **[Story]**: `US1`-`US5` from `spec.md`. Cross-cutting tasks use `[X]`.
- **[Owner]**: `Cassius` (backend Go), `Aurelia` (Vue/PWA), `Brutus` (QA), `Maximus` (architecture review).
- File paths are relative to repo root.

---

## Phase 1: Setup & Baseline

- [x] **T001** [X] [Cassius] Confirm baseline green: `cd src/api && go build ./... && go vet ./... && go test ./...` and `cd src/web && npx vue-tsc --noEmit`. Record output; no source edits.

---

## Phase 2: Foundational — Model, Repository, Settings (BLOCKING)

All user stories depend on this phase.

### Tests first

- [x] **T002** [P] [X] [Cassius] Write model + migration test at `src/api/models/purchase_reminder_test.go`: assert `PurchaseReminder` struct has expected fields and JSON tags. After `AutoMigrate`, confirm the `purchase_reminders` table exists with columns `id`, `coin_id`, `user_id`, `remind_date`, `timezone`, `status`, `notified_at`, `cancelled_at`, `created_at`, `updated_at`.
- [x] **T003** [P] [X] [Cassius] Write repository tests at `src/api/repository/purchase_reminder_repository_test.go`:
    - `CreateReminder`: inserts row, returns with ID set.
    - `FindActiveByCoinAndUser`: returns active reminder, nil when none or cancelled.
    - `UpdateReminder`: updates `remind_date`, `timezone`, `status`.
    - `CancelActiveForCoin`: sets `status=cancelled`, `cancelled_at` for all active reminders on a coin.
    - `ListDueReminders`: returns only `pending` reminders whose `remind_date` is today-or-past (string comparison `<=`); skips `notified`, `cancelled`, future.
    - `ListActiveByUser`: returns `pending` + `notified` for a user with preloaded `Coin.Name`.
    - `MarkNotified`: sets `status=notified`, `notified_at`.

### Implementation

- [x] **T004** [P] [X] [Cassius] Create model at `src/api/models/purchase_reminder.go`:
    ```go
    type PurchaseReminder struct {
        ID          uint       `gorm:"primaryKey" json:"id"`
        CoinID      uint       `gorm:"not null;index" json:"coinId"`
        Coin        Coin       `gorm:"foreignKey:CoinID" json:"coin,omitempty"`
        UserID      uint       `gorm:"not null;index" json:"userId"`
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

- [x] **T005** [X] [Cassius] Add `&models.PurchaseReminder{}` to `AutoMigrate` call in `src/api/database/database.go` (append to existing list).

- [x] **T006** [P] [X] [Cassius] Create repository at `src/api/repository/purchase_reminder_repository.go` with:
    - `NewPurchaseReminderRepository(db *gorm.DB)` constructor.
    - `CreateReminder(r *PurchaseReminder) error`
    - `FindActiveByCoinAndUser(coinID, userID uint) (*PurchaseReminder, error)` — `WHERE coin_id = ? AND user_id = ? AND status IN ('pending','notified') LIMIT 1`
    - `UpdateReminder(r *PurchaseReminder, fields ...string) error`
    - `CancelActiveForCoin(tx *gorm.DB, coinID, userID uint) error` — `UPDATE ... SET status='cancelled', cancelled_at=now WHERE coin_id=? AND user_id=? AND status IN ('pending','notified')`
    - `CancelActiveForCoinAllUsers(tx *gorm.DB, coinID uint) error` — same but no user filter (for coin delete)
    - `ListDueReminders() ([]PurchaseReminder, error)` — `WHERE status = 'pending'` (caller evaluates timezone)
    - `ListActiveByUser(userID uint) ([]PurchaseReminder, error)` — preloads `Coin` (for name), `WHERE user_id = ? AND status IN ('pending','notified') ORDER BY remind_date ASC`
    - `MarkNotified(tx *gorm.DB, reminderID uint) error` — `UPDATE ... SET status='notified', notified_at=now WHERE id=? AND status='pending'`
    - `WithTx(tx *gorm.DB) *PurchaseReminderRepository`

- [x] **T007** [X] [Cassius] Add settings constants to `src/api/services/settings_service.go`:
    - `SettingReminderCheckEnabled = "ReminderCheckEnabled"` with default `"true"`
    - `SettingReminderCheckStartTime = "ReminderCheckStartTime"` with default `"08:00"`

- [x] **T008** [X] [Cassius] Run T002 + T003 tests green. Confirm `go vet ./...` clean.

**Checkpoint**: Foundation ready — service/handler/scheduler work can begin.

---

## Phase 3: Backend — Service + Auto-Cancel (US1, US3, US4)

### Tests first

- [x] **T009** [P] [US1] [Cassius] Write service tests at `src/api/services/purchase_reminder_service_test.go`:
    - `CreateOrUpdate` with no existing reminder: creates, returns 201 semantics.
    - `CreateOrUpdate` with existing active reminder: updates in place, resets to pending.
    - `CreateOrUpdate` on non-wishlist coin: returns `ErrCoinNotWishlist`.
    - `CreateOrUpdate` with past date: returns `ErrRemindDatePast`.
    - `CreateOrUpdate` with invalid timezone: returns `ErrInvalidTimezone`.
    - `Cancel` with active reminder: cancels, returns nil.
    - `Cancel` with no active reminder: returns `ErrReminderNotFound`.
    - `GetForCoin` returns active or nil.
    - `ListForUser` returns active reminders with coin names.

- [x] **T010** [P] [US4] [Cassius] Write auto-cancel integration test at `src/api/services/purchase_reminder_autocancel_test.go`:
    - Create wishlist coin + reminder. Update coin `IsWishlist=false`. Verify reminder status = `cancelled`.
    - Create wishlist coin + reminder. Delete coin. Verify reminder is cancelled.
    - Update non-wishlist coin (no reminder) — no error, no side effect.

### Implementation

- [x] **T011** [US1] [Cassius] Create service at `src/api/services/purchase_reminder_service.go`:
    - `NewPurchaseReminderService(repo, coinRepo, logger)` constructor.
    - Sentinel errors: `ErrCoinNotWishlist`, `ErrRemindDatePast`, `ErrInvalidTimezone`, `ErrReminderNotFound`.
    - `CreateOrUpdate(userID, coinID uint, remindDate, timezone string) (*models.PurchaseReminder, bool, error)` — returns (reminder, isNew, error). Validates timezone via `time.LoadLocation`. Validates date >= today in that timezone. Checks coin ownership + `IsWishlist`. Upserts via repo.
    - `Cancel(userID, coinID uint) error` — finds active, cancels.
    - `GetForCoin(userID, coinID uint) (*models.PurchaseReminder, error)` — returns active or nil.
    - `ListForUser(userID uint) ([]models.PurchaseReminder, error)` — delegates to repo.

- [x] **T012** [US4] [Cassius] Add auto-cancel hook in `src/api/services/coin_service.go`:
    - Inside `updateCoin`, after the coin update succeeds in the transaction, check if `IsWishlist` was provided in `updateFields` and the value transitioned `true -> false`.
    - If so, call `reminderRepo.CancelActiveForCoin(tx, existing.ID, userID)`.
    - `CoinService` constructor gains `reminderRepo *repository.PurchaseReminderRepository` parameter.
    - For coin deletion (if `DeleteCoin` or similar exists), add `reminderRepo.CancelActiveForCoinAllUsers(tx, coinID)` before delete.

- [x] **T013** [X] [Cassius] Run T009 + T010 tests green.

**Checkpoint**: Service layer complete — handler and scheduler can proceed in parallel.

---

## Phase 4: Backend — Handler (US1, US3, US5)

### Tests first

- [x] **T014** [P] [US1] [Cassius] Write handler tests at `src/api/handlers/purchase_reminder_handler_test.go`:
    - `POST /coins/:id/reminder` — 201 create, 200 update, 400 bad date/timezone, 404 missing coin, 409 not wishlist.
    - `GET /coins/:id/reminder` — 200 with active, 404 when none.
    - `DELETE /coins/:id/reminder` — 204 success, 404 when none.
    - `GET /reminders` — 200 with list of active reminders.
    - All owner-scoped: other user's coin returns 404.

### Implementation

- [x] **T015** [US1] [Cassius] Create handler at `src/api/handlers/purchase_reminder_handler.go`:
    - `NewPurchaseReminderHandler(svc, logger)` constructor.
    - `CreateOrUpdate(c *gin.Context)` — parses coin ID from URL, binds `{remindDate, timezone}` from body, calls service, returns 201 or 200.
    - `Get(c *gin.Context)` — returns active reminder or 404.
    - `Cancel(c *gin.Context)` — calls service cancel, returns 204 or 404.
    - `List(c *gin.Context)` — returns all active for user.
    - Swagger annotations on all methods.

- [x] **T016** [US1] [Cassius] Wire in `src/api/deps.go`:
    - Create `purchaseReminderRepo` from `database.DB`.
    - Create `purchaseReminderSvc` with repo + coinRepo + logger.
    - Pass `purchaseReminderRepo` to `CoinService` constructor (update signature).
    - Create `purchaseReminderHandler` with svc + logger.
    - Create `reminderScheduler` (Phase 5) and register with `schedulerRegistry`.

- [x] **T017** [US1] [Cassius] Register routes in `src/api/routes_protected.go`:
    ```go
    protected.POST("/coins/:id/reminder", purchaseReminderHandler.CreateOrUpdate)
    protected.GET("/coins/:id/reminder", purchaseReminderHandler.Get)
    protected.DELETE("/coins/:id/reminder", purchaseReminderHandler.Cancel)
    protected.GET("/reminders", purchaseReminderHandler.List)
    ```

- [x] **T018** [X] [Cassius] Run T014 tests green + full `go test ./...` + `go vet ./...`.

**Checkpoint**: All CRUD endpoints functional.

---

## Phase 5: Backend — Scheduler + Notifications (US2)

### Tests first

- [x] **T019** [P] [US2] [Cassius] Write scheduler tests at `src/api/services/reminder_scheduler_test.go`:
    - `runCycle` with one due pending reminder: notification created with correct type/referenceId/referenceUrl, reminder status = `notified`, `notifiedAt` set.
    - `runCycle` with future reminder: no notification, status unchanged.
    - `runCycle` with overdue reminder (past date): still notified (catch-up).
    - `runCycle` with already-notified reminder: no duplicate notification.
    - `runCycle` with disabled setting: skipped, no processing.
    - `runCycle` with reminder in different timezone where "today" differs: correct timezone evaluation.
    - Pushover failure: logged, notification still created, status still transitions.

### Implementation

- [x] **T020** [US2] [Cassius] Add `NotifyPurchaseReminder` method to `src/api/services/notification_service.go`:
    ```go
    func (s *NotificationService) NotifyPurchaseReminder(userID, reminderID, coinID uint, coinName string) {
        // type: "purchase_reminder", referenceId: reminderID, referenceUrl: "/coin/{coinID}"
        // Pushover best-effort with deep link
    }
    ```
    Add `NotificationTypePurchaseReminder = "purchase_reminder"` constant.

- [x] **T021** [US2] [Cassius] Create scheduler at `src/api/services/reminder_scheduler.go`:
    - Implements `Scheduler` interface (`Start`, `Stop`, `RunNow`, `timeUntilNextRun`, `GetStatus`).
    - Constructor: `NewReminderScheduler(repo, notifSvc, settingsSvc, logger)`.
    - `runCycle`: check `ReminderCheckEnabled`. Query all pending reminders. For each, evaluate `remindDate <= today` in stored timezone. If due, within a per-reminder transaction: re-check `status=pending`, call `NotifyPurchaseReminder`, call `MarkNotified`. Log per-reminder outcome.
    - Daily cadence: fixed 1440 min interval, `ReminderCheckStartTime` anchor.
    - `sync.Once` in `Stop()`.
    - `GetStatus` returns `SchedulerStatus{Name: "Reminder Check", Enabled: ..., IsRunning: ..., NextRunIn: ...}`.

- [x] **T022** [X] [Cassius] Run T019 tests green + full `go test ./...`.

**Checkpoint**: Full backend complete — scheduler fires notifications correctly.

---

## Phase 6: Frontend — Modal, Badge, API Integration (US1, US3, US5)

### Tests first

- [x] **T023** [P] [US1] [Aurelia] Write component test at `src/web/src/components/coin/__tests__/PurchaseReminderModal.test.ts`:
    - Renders date input and submit button.
    - Submits correct `remindDate` and `timezone` to API.
    - Shows existing reminder date in edit mode.
    - Cancel action calls DELETE and closes modal.
    - Validates date is not in the past (client-side).
    - Focus trap and keyboard navigation (Escape closes).
    - Mobile: 44px+ tap targets.

- [x] **T024** [P] [US5] [Aurelia] Extend wishlist card test at `src/web/src/pages/__tests__/WishlistPage.test.ts` (or `CoinCard.test.ts`):
    - Card with active reminder shows badge with formatted date.
    - Card without reminder shows no badge.

### Implementation

- [x] **T025** [P] [US1] [Aurelia] Add API methods to `src/web/src/api/client.ts`:
    ```typescript
    createOrUpdateReminder(coinId: number, data: { remindDate: string; timezone: string }): Promise<PurchaseReminder>
    getReminder(coinId: number): Promise<PurchaseReminder | null>
    deleteReminder(coinId: number): Promise<void>
    listReminders(): Promise<PurchaseReminder[]>
    ```

- [x] **T026** [P] [US1] [Aurelia] Add `PurchaseReminder` type to `src/web/src/types/coin.ts`:
    ```typescript
    interface PurchaseReminder {
      id: number; coinId: number; coinName?: string;
      remindDate: string; timezone: string;
      status: 'pending' | 'notified' | 'cancelled';
      notifiedAt?: string; cancelledAt?: string;
      createdAt: string; updatedAt: string;
    }
    ```

- [x] **T027** [US1] [Aurelia] Create composable at `src/web/src/composables/usePurchaseReminder.ts`:
    - `usePurchaseReminder(coinId: Ref<number>)` — reactive `reminder` ref, `loading`, `saving`.
    - `fetchReminder()`, `saveReminder(date)`, `cancelReminder()`.
    - Auto-detects browser timezone via `Intl.DateTimeFormat().resolvedOptions().timeZone`.

- [x] **T028** [US1] [Aurelia] Create modal component at `src/web/src/components/coin/PurchaseReminderModal.vue`:
    - Props: `coinId`, `coinName`, `existingReminder?`.
    - Emits: `saved`, `cancelled`, `close`.
    - Native `<input type="date">` with `min` set to today.
    - Uses design tokens: `--bg-card`, `--border-subtle`, `--accent-gold`, `--radius-sm`.
    - Focus trap, Escape to close, ARIA labels.
    - Mobile: minimum 44px tap targets.

- [x] **T029** [US5] [Aurelia] Add reminder badge to `src/web/src/components/CoinCard.vue`:
    - When coin has an active reminder (data from list API or prop), show `.chip-sm` badge.
    - Format: "Due Today", "Due Tomorrow", "Due {MMM DD}".
    - Color: `--accent-gold` text on `--accent-gold-dim` background.

- [x] **T030** [US1] [Aurelia] Add reminder action + section to `src/web/src/pages/CoinDetailPage.vue`:
    - For wishlist coins: add a bell/clock icon button in the header actions area.
    - Clicking opens `PurchaseReminderModal`.
    - Below the header, show a reminder info line when active ("Reminder: Aug 25, 2026" with edit/cancel links).
    - After save/cancel, refetch reminder state.

- [x] **T031** [US5] [Aurelia] Handle `purchase_reminder` notification type in `src/web/src/pages/NotificationsPage.vue`:
    - On click, navigate to `/coin/{referenceUrl}` (existing pattern — verify `referenceUrl` is used correctly for this type).

- [x] **T032** [X] [Aurelia] Run `npx vue-tsc --noEmit` + `npx vitest run` + `npm run build`. All green.

**Checkpoint**: Full frontend complete.

---

## Phase 7: Integration & Polish

- [x] **T033** [X] [Cassius] Full backend validation: `cd src/api && go build ./... && go vet ./... && go test -v ./...`. Zero failures.
- [ ] **T034** [X] [Aurelia] Full frontend validation: `cd src/web && npx vue-tsc --noEmit && npm run build`. Zero failures.
- [ ] **T035** [X] [Brutus] Write regression test confirming existing scheduler behavior unchanged: CoinOfDay, Availability, AuctionEnding schedulers still start/stop/cycle correctly with the new ReminderScheduler registered.
- [ ] **T036** [X] [Maximus] Architecture review: verify no Principle I violations (`architecture_test.go` green), no leaked internal errors, owner-scoping on all queries, transaction boundaries correct.

---

## Phase 8: Admin Schedule UI (Expanded Scope — FR-015a / D12)

- [x] **T037** [FR-015a] [Aurelia] Create `src/web/src/components/admin/schedules/AdminPurchaseReminderSchedule.vue`:
    - Enable/disable toggle bound to `settings.ReminderCheckEnabled`.
    - Start-time `<input type="time">` bound to `settings.ReminderCheckStartTime`.
    - Save button emitting parent `save` event.
    - No Run Now button or run-history table (locked out-of-scope).
    - Follow `AdminCoinOfDaySchedule.vue` structure (minus run-history/manual-trigger sections).
    - Add `ReminderCheckEnabled?: string` and `ReminderCheckStartTime?: string` to `AppSettings` in `src/web/src/types/admin.ts`.
    - Import and mount in `src/web/src/components/admin/AdminSchedulesSection.vue`.

- [x] **T038** [FR-015a] [Aurelia] Tests for `AdminPurchaseReminderSchedule.vue`:
    - Component renders toggle and time input with correct initial values from settings prop.
    - Toggle change updates `settings.ReminderCheckEnabled` between `"true"` and `"false"`.
    - Save button click emits `save` event.
    - Run `npx vue-tsc --noEmit` + `npx vitest run` + `npm run build`. All green.

**Checkpoint**: Feature complete. Ready for PR.

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1 (Setup)**: No dependencies.
- **Phase 2 (Foundation)**: Depends on Phase 1. **BLOCKS all subsequent phases.**
- **Phase 3 (Service)**: Depends on Phase 2.
- **Phase 4 (Handler)**: Depends on Phase 3 (needs service).
- **Phase 5 (Scheduler)**: Depends on Phase 3 (needs repo + notification method). Can run **in parallel with Phase 4**.
- **Phase 6 (Frontend)**: Depends on Phase 4 (needs API endpoints). Can start T025-T026 in parallel with Phase 4-5.
- **Phase 7 (Integration)**: Depends on all prior phases.
- **Phase 8 (Admin Schedule UI)**: Depends on Phase 7 (settings keys must exist in backend). No backend code changes — frontend only.

### Parallel Opportunities

- T002, T003, T004, T006: all [P] within Phase 2 (different files).
- T009, T010: [P] within Phase 3 (different test files).
- T014 can start as soon as T011 merges (handler tests mock the service).
- T019 can start in parallel with T014 (scheduler tests, different file).
- T023, T024, T025, T026: all [P] within Phase 6 (different files).
- Phase 4 and Phase 5 can run in parallel after Phase 3.

### Task Ownership Summary

| Owner | Tasks | Count |
|-------|-------|-------|
| Cassius | T001-T022 | 22 |
| Aurelia | T023-T032, T037-T038 | 12 |
| Brutus | T035 | 1 |
| Maximus | T036 | 1 |

### Within Each Phase

- Tests MUST be written and FAIL before implementation.
- Models before repositories.
- Repositories before services.
- Services before handlers.
- Scheduler in parallel with handler after service is ready.
