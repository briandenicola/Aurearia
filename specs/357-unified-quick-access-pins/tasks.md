---
description: "Backend and frontend task list for feature 357 — Unified Quick Access Pins"
---

# Tasks: Unified Quick Access Pins

**Input**: `specs/357-unified-quick-access-pins/spec.md`, `plan.md`
**Tests**: Required and tests-first under Constitution §17/§21.
**Scope lock**: Phases 1-10 record the completed backend phase and remain unchanged. Phases 11-17 authorize Vue/TypeScript frontend work only against backend commit `4d3b6a06`; do not modify `src/api/`.

## Format

`[ID] [P?] [Story] [Owner] Description`

- `[P]`: safe to run in parallel because the task uses different files and has no unresolved dependency.
- Stories map to `spec.md` US1-US5. Cross-cutting tasks use `[X]`.
- Suggested owners: Cassius (backend implementation), Brutus (QA), Maximus (architecture review).

## Phase 1: Baseline and Contract Guards

- [x] **T001** [X] [Cassius] Record a clean backend baseline from `src/api`: `go build ./...`, `go vet ./...`, `go test ./...`; do not edit source during this task.
- [ ] **T002** [X] [Brutus] Add/confirm a scope guard in the review checklist that `git diff --name-only` contains no `src/web/` path for Feature 357.
- [x] **T003** [X] [Cassius] Run `task openapi` once on the baseline and document the generated-file set so later Swagger changes use the repository workflow rather than ad-hoc edits.

---

## Phase 2: Data Model and Migration Foundation (BLOCKING)

### Tests first

- [x] **T004** [P] [US5] [Cassius] Create `src/api/database/feature357_migration_regression_test.go` with a uniquely named in-memory SQLite database. Seed a legacy schema containing: pinned/unpinned Coin Sets, a legacy >5 pinned-set drift case, linked/unlinked `AuctionEvent` rows, and auction lots. Run the intended production migration sequence twice and assert FR-019, FR-026-FR-027, SC-001, indexes/constraint, no row loss, and idempotency.
- [x] **T005** [P] [US1] [Cassius] Create model/schema tests for `QuickAccessPin`: exact columns, allowed target values, uniqueness `(user_id,target_type,target_id)`, owner/time index, target cleanup index, and non-null UTC `pinned_at`.
- [x] **T006** [X] [Cassius] Update the production-model constructor map/drift guard in `src/api/database/feature353_migration_order_regression_test.go` to include `QuickAccessPin`; prove it fails if the live `AutoMigrate` list and fixture map diverge.

### Implementation

- [x] **T007** [P] [US1] [Cassius] Add `src/api/models/quick_access_pin.go` with typed target constants, `QuickAccessPin`, and database tags from plan D2.
- [x] **T008** [P] [US4] [Cassius] Add typed `AuctionEventOrigin` constants and `Origin` to `src/api/models/showcase.go`; default new direct-created events to `manual`.
- [x] **T009** [X] [Cassius] Register `QuickAccessPin` in the real `src/api/database/database.go` `AutoMigrate` list at a dependency-safe position.
- [x] **T010** [X] [Cassius] Implement error-returning migration helpers in `src/api/database/database.go`: legacy event-origin backfill and Coin Set pin backfill/reconciliation. Preserve original timestamps, preserve >5 legacy pins, and fail startup on any helper error.
- [x] **T011** [X] [Cassius] Run T004-T006 green twice (`-count=2`) to prove restart idempotency and isolation.

**Checkpoint**: Schema and migration truth are stable before repository/service work.

---

## Phase 3: Quick Access Repository and DTO Queries

### Tests first

- [x] **T012** [P] [US1, US2] [Cassius] Add `src/api/repository/quick_access_repository_test.go` covering create, duplicate conflict/reload with timestamp preservation, user-scoped delete, newest-first/tie order, target cleanup, count-by-type for set cap, and rollback through `WithTx`.
- [x] **T013** [P] [US1] [Cassius] Add repository batch-hydration tests proving: one batch per type, owner scope, coin sold exclusion, lot terminal exclusion, calendar manual-only exclusion, and no foreign records.

### Implementation

- [x] **T014** [US1, US2] [Cassius] Create `src/api/repository/quick_access_repository.go` with `WithTx`, `Transaction`, conflict-safe create/reload, user list, user/type count, scoped delete, target cleanup, and batch DTO projection helpers.
- [x] **T015** [P] [US5] [Cassius] Extend `src/api/repository/set_repository.go` with transaction-aware methods needed to set/clear the `PinnedAt` compatibility mirror without association mutation.
- [x] **T016** [P] [US4] [Cassius] Extend auction and calendar repositories with narrowly typed transaction/batch methods required by plan D9; use `OwnedBy`/`OwnedByID` scopes instead of duplicated owner clauses where applicable.
- [x] **T017** [X] [Cassius] Run repository tests with `-count=2`; confirm every SQLite test helper uses a unique named in-memory DSN per the isolation skill.

---

## Phase 4: Quick Access Service and Typed Contract

### Tests first

- [x] **T018** [P] [US2] [Cassius] Add `src/api/services/quick_access_service_test.go` eligibility matrix: owned coin, wishlist coin, sold coin, Coin Set, watching/bidding/terminal lot, manual/auction event, missing target, foreign target, invalid type.
- [x] **T019** [P] [US2, US5] [Cassius] Add service tests for idempotent original-time semantics, unpin/re-pin new time, five-set cap, non-set pins excluded from cap, legacy >5 behavior, and exact existing cap message.
- [x] **T020** [P] [US1] [Cassius] Add service hydration contract tests asserting newest-first order, exact discriminator/payload pairing, derived classifications/status, empty array behavior, stale omission without GET mutation, and absence of private/full-model fields.

### Implementation

- [x] **T021** [US1, US2] [Cassius] Create `src/api/services/quick_access_service.go` with sentinels, target parsing, `List`, `Pin`, and `Unpin`; inject Quick Access plus coin/set/auction/event repositories.
- [x] **T022** [US5] [Cassius] Implement Coin Set pin/unpin in `QuickAccessService` as one transaction updating authoritative pin plus `CoinSet.PinnedAt` mirror. Count authoritative `coin_set` rows inside the transaction before insert.
- [x] **T023** [US1] [Cassius] Implement explicit DTO types matching spec §4.2; exactly one payload pointer per item and no GORM model embedding.
- [x] **T024** [X] [Cassius] Run all Quick Access service/repository tests and `go vet ./...`.

**Checkpoint**: New API business contract works independently before lifecycle integration.

---

## Phase 5: Coin Set Compatibility Integration (US5)

### Tests first

- [x] **T025** [P] [US5] [Cassius] Extend `src/api/services/set_service_test.go`: legacy `pinned:true|false` delegates to unified service, both timestamps match, double-pin preserves time, sixth set fails, unpin via either API is visible through the other.
- [x] **T026** [P] [US4, US5] [Cassius] Add set-delete rollback tests: successful delete removes memberships/set/pin; forced pin-cleanup failure rolls back all deletion work.

### Implementation

- [x] **T027** [US5] [Cassius] Inject `QuickAccessService` (or its narrow set-pin interface) into `SetService`; replace independent `PinnedAt` logic in `UpdateSet` with the shared authoritative operation.
- [x] **T028** [US4] [Cassius] Make `SetService.DeleteSet` transactionally remove the unified pin and set/memberships; refactor repository transaction boundaries only as needed to keep business orchestration in the service.
- [x] **T029** [US5] [Cassius] Preserve existing `GET /sets`/`GET /sets/:id` `pinned` and `pinnedAt` response behavior from the synchronized mirror; do not change frontend ordering semantics.

---

## Phase 6: Coin Lifecycle Integration (US3, US4)

### Tests first

- [x] **T030** [P] [US3] [Cassius] Add tests proving a pinned wishlist coin keeps the same pin ID/time through `PurchaseCoin` and generic `isWishlist:true->false` update; hydration changes only classification.
- [x] **T031** [P] [US4] [Cassius] Add tests for pin removal on `SellCoin`, generic `isSold:false->true` update, and `DeleteCoin`; include forced cleanup failure rollback assertions.
- [x] **T032** [P] [US4] [Cassius] Add a no-restoration test: sold->active correction does not recreate a removed pin.

### Implementation

- [x] **T033** [US3, US4] [Cassius] Inject a narrow Quick Access cleanup dependency into `CoinService`. Preserve pins on wishlist purchase, remove on every sold transition and delete, and keep each synchronous mutation atomic with cleanup.
- [x] **T034** [US4] [Cassius] Verify all coin mutation sibling paths (`PUT`, purchase, sell, delete, bulk if it can set sold/wishlist flags) either pass through the covered service path or gain an explicit regression task; do not leave a bypass undocumented.

---

## Phase 7: Auction Lifecycle Integration (US3, US4)

### Tests first

- [x] **T035** [P] [US3] [Cassius] Extend `auction_lot_service_test.go`: pinned watching<->bidding preserves pin ID/time under manual status changes.
- [x] **T036** [P] [US4] [Cassius] Add manual terminal and delete tests: won/lost/passed remove pins; missing/foreign behavior remains generic; rollback on cleanup failure.
- [x] **T037** [P] [US3, US4] [Cassius] Extend `auction_watchlist_sync_service_test.go` for provider-driven watching->bidding preservation and watching/bidding->won|lost|passed removal within the same per-lot transaction.
- [x] **T038** [P] [US4] [Cassius] Add no-restoration coverage for terminal->watching/bidding manual correction.

### Implementation

- [x] **T039** [US3, US4] [Cassius] Inject Quick Access cleanup into `AuctionLotService`; make status and delete operations service-owned and transactional.
- [x] **T040** [US4] [Cassius] Refactor `AuctionLotRepository` upsert to support caller-owned transactions without nested independent commits.
- [x] **T041** [US3, US4] [Cassius] Update `AuctionWatchlistSyncService` to apply each provider upsert and any terminal pin cleanup in one transaction.
- [x] **T042** [US4] [Cassius] Update `src/api/handlers/auction_lots.go` so delete and status mutation no longer perform repository writes outside the service.

---

## Phase 8: Calendar Lifecycle and Manual-Origin Integration (US2, US4)

### Tests first

- [x] **T043** [P] [US2] [Cassius] Add calendar service tests: direct create writes `origin=manual`; auction auto-create writes `origin=auction`; only manual events can be pinned.
- [x] **T044** [P] [US4] [Cassius] Add calendar delete tests: manual event deletion removes pin atomically; foreign/missing delete does not leak ownership; forced cleanup error rolls back.
- [x] **T045** [P] [US2] [Cassius] Add migration fixture coverage for the documented linked-event ambiguity and assert the conservative classification rule exactly.

### Implementation

- [x] **T046** [US2, US4] [Cassius] Create `src/api/services/calendar_service.go` for calendar CRUD/business orchestration and Quick Access cleanup.
- [x] **T047** [US2] [Cassius] Update auction auto-event creation to write `origin=auction`; direct calendar create writes `origin=manual`.
- [x] **T048** [US4] [Cassius] Refactor `src/api/handlers/calendar.go` to depend on `CalendarService` for mutations and retain only parsing/response mapping.

---

## Phase 9: Handler, Wiring, Routes, and OpenAPI

### Tests first

- [x] **T049** [P] [US1, US2] [Cassius] Add `src/api/handlers/quick_access_test.go`: authenticated list; 201 create; 200 repeated create with same timestamp; 204 repeated delete; 400 invalid type/ID/cap; generic 404 missing/foreign/ineligible; generic 500.
- [ ] **T050** [P] [US1] [Brutus] Add a response-contract test that unmarshals every item variant and asserts exactly one matching payload property is present.

### Implementation

- [x] **T051** [US1, US2] [Cassius] Create `src/api/handlers/quick_access.go` with thin `List`, `Pin`, and `Unpin` methods plus complete Swagger annotations.
- [x] **T052** [X] [Cassius] Wire Quick Access repository/service/handler and updated Set/Coin/Auction/Calendar dependencies in `src/api/deps.go` using constructor injection.
- [x] **T053** [X] [Cassius] Register the three routes in `src/api/routes_protected.go`; apply the existing authenticated write limiter to PUT/DELETE if consistent with neighboring write routes.
- [x] **T054** [X] [Cassius] Regenerate Swagger/OpenAPI using `task openapi`; do not add a route-drift exemption.
- [x] **T055** [X] [Cassius] Run `go test -v -run TestRegisteredAPIRoutesAreDocumentedInOpenAPI .` and all handler tests.

---

## Phase 10: Quality Gate and Architecture Review

- [x] **T056** [X] [Cassius] Run from `src/api`: `go build ./...`, `go vet ./...`, `go test ./...`; record exact results.
- [ ] **T057** [X] [Brutus] Run targeted lifecycle suites with `-count=2`, then full `go test ./...`; verify no test uses shared bare `:memory:` where cross-test leakage is possible.
- [ ] **T058** [X] [Brutus] Inspect `git diff --name-only` and fail review if any `src/web/` file changed.
- [ ] **T059** [X] [Maximus] Perform post-implementation architecture review against FR-001-FR-036: layering, transaction ownership, non-leaking errors, DTO shape, migration truth, set compatibility, all sibling lifecycle paths, and Swagger drift.
- [ ] **T060** [X] [Maximus] Reconcile completed task checkboxes and require the PR description to cite `spec §4 (FR-001-FR-036)`, Constitution Principles I/III/IV/V/IX, §17, and §21.

## Dependencies and Execution Order

1. Phase 2 blocks all implementation.
2. Phase 3 blocks Phase 4.
3. Phase 4 blocks resource lifecycle phases 5-8.
4. Phases 5-8 can proceed in parallel only if owners coordinate shared constructor/wiring changes; otherwise run sequentially to avoid conflicts.
5. Phase 9 depends on the service contract and lifecycle dependencies being final.
6. Phase 10 is the release gate.

## Independent Story Checkpoints

- **US1**: GET returns typed mixed owner-scoped list in correct order.
- **US2**: PUT/DELETE idempotency and eligibility matrix pass.
- **US3**: wishlist purchase and watching/bidding transitions preserve identity/time.
- **US4**: every terminal/deletion path removes pins and rolls back on cleanup failure.
- **US5**: legacy Coin Set pins migrate safely; old/new endpoints share cap and truth.

## Deferred Frontend Handoff

Superseded by the repository owner's explicit frontend authorization on 2026-09-10. The authorized work is defined below; backend tasks and completion states above remain historical and MUST NOT be reopened.

---

## Phase 11: Frontend Contract and Baseline (BLOCKING)

- [x] **T061** [X] [Aurelia] Record a clean frontend baseline from `src/web`: `npm run type-check`, `npm test`, `npm run lint`, and `npm run build`; do not edit source during this task.
- [x] **T062** [P] [US6] [Aurelia] Add `src/web/src/types/quick-access.ts` with exact target literals, payload DTOs, and a four-variant discriminated union matching spec §4.2 / FR-037; re-export it from `types/index.ts`.
- [x] **T063** [P] [US6, US7] [Aurelia] Add `src/web/src/api/endpoints/quickAccess.ts` with typed list, pin, and unpin functions using the shared Axios instance; re-export it from `api/client.ts`.
- [x] **T064** [X] [Brutus] Add endpoint contract tests proving exact paths/methods, empty-body PUT, 200/201 response typing, 204 unpin handling, and no direct Python/external request.

**Checkpoint**: The committed backend contract is represented exactly before shared state or UI work.

---

## Phase 12: Shared Quick Access State and Security Lifecycle

### Tests first

- [x] **T065** [US6, US7, US8] [Aurelia] Create `composables/__tests__/useQuickAccess.test.ts` covering server-order refresh, empty response, recoverable refresh failure retaining prior items, successful PUT upsert, successful DELETE removal, request failure preserving state, and typed `isPinned`.
- [x] **T066** [US8] [Aurelia] Add delayed-response tests proving `clear()` invalidates an in-flight refresh and a user-A response cannot repopulate state after logout/user switch.

### Implementation

- [x] **T067** [US6, US7, US8] [Aurelia] Implement `useQuickAccess.ts` per D15 with module-level items/loading/error, generation-guarded refresh, successful-response-only mutation, 201 prepend, 200 in-place replacement, DELETE removal, `isPinned`, and `clear`; no localStorage or polling.
- [x] **T068** [X] [Brutus] Run T064-T067 targeted tests twice and verify no shared-state leakage between tests.

---

## Phase 13: Quick Access Page, Route, and Global Navigation

### Tests first

- [x] **T069** [P] [US6] [Aurelia] Add `QuickAccessPage` tests for loading, retained-list error/retry, empty state, all four DTO variants, server order, null optional dates/images, and desktop/PWA-safe responsive structure.
- [x] **T070** [P] [US6] [Aurelia] Extend router/App navigation tests for authenticated `/quick-access`, one reorderable top-level `Quick Access` item, active styling, sidebar close behavior, and preservation of the existing Sets submenu pins.
- [x] **T071** [US6] [Aurelia] Add navigation tests asserting exact destinations: coin `/coin/:id`, set `/sets/:id`, lot `/auctions?lot=:id`, event `/calendar?event=:id`.

### Implementation

- [x] **T072** [US6] [Aurelia] Create `pages/QuickAccessPage.vue` using existing page-header, card, chip/badge, empty-state, design-token, lucide icon, and mobile patterns; render directly from DTOs without target refetches.
- [x] **T073** [US6] [Aurelia] Register `/quick-access` in `router/index.ts` and add the top-level nav item in `App.vue` without creating a submenu or changing pinned Sets placement.
- [x] **T074** [US6, US8] [Aurelia] Wire authenticated App bootstrap and an authenticated-user-ID watcher that clears before account-switch refresh; wire page retry/refresh to the same composable and confirm no polling.

---

## Phase 14: Detail Pin Controls

### Tests first

- [x] **T075** [P] [US7] [Aurelia] Extend coin-detail/header tests for unsold owned and wishlist eligibility, sold/follower exclusion, `aria-pressed`, busy state, active icon styling, successful toggle, and retained state plus toast on failure.
- [x] **T076** [P] [US7] [Aurelia] Extend Set detail and App navigation tests proving the existing control calls unified PUT/DELETE, preserves the five-set message, refreshes set detail and `usePinnedSets`, and updates both Quick Access and the Sets submenu.
- [x] **T077** [P] [US7] [Aurelia] Extend auction modal tests for watching/bidding controls, terminal exclusion, accessibility, successful toggle, and failure preservation.
- [x] **T078** [P] [US7] [Aurelia] Extend calendar tests for `origin=manual` controls, `origin=auction` exclusion, accessibility, successful toggle, and failure preservation.

### Implementation

- [x] **T079** [US7] [Aurelia] Add the shared-state pin affordance to `CoinDetailHeaderActions.vue`/`CoinDetailPage.vue` without changing follower detail.
- [x] **T080** [US7] [Aurelia] Migrate `SetDetailPage.vue` from legacy `updateSet({pinned})` to unified pin/unpin, then refresh set detail and the existing pinned-set projection.
- [x] **T081** [US7] [Aurelia] Add eligible pin/unpin handling to `AuctionLotDetailModal.vue`.
- [x] **T082** [US7] [Aurelia] Type calendar event origin and add eligible pin/unpin handling to the existing `CalendarPage.vue` event drawer.

---

## Phase 15: Deep Links and Lifecycle Reconciliation

### Tests first

- [x] **T083** [P] [US6, US8] [Aurelia] Preserve/extend Auctions tests proving `/auctions?lot=:id` fetches by ID outside the active filter, opens the modal, removes only `lot` on close, and fails safely for invalid/foreign/missing IDs.
- [x] **T084** [P] [US6, US8] [Aurelia] Add Calendar tests proving `/calendar?event=:id` fetches by ID outside the active month, opens the drawer, removes only `event` on close, preserves unrelated query parameters, and fails safely.
- [x] **T085** [P] [US8] [Aurelia] Add coin lifecycle tests: wishlist purchase refreshes classification while preserving the pin; sell and delete remove/refresh the pin.
- [x] **T086** [P] [US8] [Aurelia] Add auction/calendar/set lifecycle tests: watching/bidding refreshes status; terminal and all target deletes remove/refresh the corresponding item.

### Implementation

- [x] **T087** [US6, US8] [Aurelia] Implement calendar event query deep links using fetch-by-ID, route watch plus initial route handling, and query-local `router.replace` close behavior; retain the existing auction implementation.
- [x] **T088** [US8] [Aurelia] Reconcile Quick Access after successful purchase, sold/terminal status, and delete mutations on every touched detail sibling path; do not mutate shared state before server success.

---

## Phase 16: Logout, User Switch, and Integration

- [x] **T089** [US8] [Aurelia] Update `App.vue` logout to clear `useQuickAccess` alongside notifications/pinned sets before routing to login.
- [x] **T090** [US8] [Aurelia] Extend mounted App/auth tests for logout, in-tab account switch, delayed refresh invalidation, and fresh bootstrap for the next authenticated user.
- [x] **T091** [X] [Aurelia] Run the complete feature-targeted Vitest set and fix only Feature 357 regressions.

---

## Phase 17: Frontend Quality Gate and Review

- [x] **T092** [X] [Aurelia] Run from `src/web`: `npm run type-check`, targeted Vitest files, `npm test`, `npm run lint`, and `npm run build`; record exact results.
- [x] **T093** [X] [Brutus] Independently verify FR-037-FR-054 and SC-008-SC-012, including keyboard/accessible names, 375px PWA layout, deep-link history behavior, lifecycle freshness, logout race safety, and existing pinned Sets regression.
- [x] **T094** [X] [Brutus] Inspect `git diff --name-only` and BLOCK if any `src/api/` file changed or any completed backend task was reopened.
- [ ] **T095** [X] [Maximus] Perform post-implementation architecture review against D13-D19, module-level lifecycle rules, router parent/query patterns, Principle III/IV/V/VI/IX, §17, and §21.
- [ ] **T096** [X] [Maximus] Reconcile frontend task checkboxes and require the PR description to cite backend dependency `4d3b6a06`, spec FR-037-FR-054, Constitution Principles II/III/IV/V/VI/IX, §17, and §21.

## Frontend Dependencies and Execution Order

1. Phase 11 blocks all frontend implementation.
2. Phase 12 blocks page, navigation, and controls because all consumers share one state contract.
3. Phase 13 may start after Phase 12.
4. Phase 14 control tasks may proceed in parallel by resource surface after Phase 12.
5. Phase 15 depends on the corresponding Phase 14 surface and must cover the exact sibling mutations it changes.
6. Phase 16 depends on App integration from Phase 13 and the generation guard from Phase 12.
7. Phase 17 is the frontend release gate.
