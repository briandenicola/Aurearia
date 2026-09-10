# Feature Specification: Unified Quick Access Pins

**Feature Branch**: `357-unified-quick-access-pins`
**Created**: 2026-09-10
**Status**: Active — backend complete; frontend implementation approved
**Owner**: Maximus (Lead / Architect)
**Requested by**: Brian DeNicola
**Input**: Unify existing pinned Coin Sets with pinnable coins, auction lots, and manual calendar events behind one authenticated Quick Access contract.

## 1. Context and Scope

Aurearia currently has one resource-specific pin implementation: `CoinSet.PinnedAt`, exposed through `PUT /sets/:id` and rendered in the Sets sidebar with a server-enforced five-set cap. The product now needs one eventual mixed Quick Access list without creating a new pin column on every resource.

This feature defines a user-owned polymorphic pin record, a typed hydration contract, and lifecycle rules that keep the list valid as resources change state.

### In scope

- Completed Go API persistence, migration, repository, service, handler, route, Swagger/OpenAPI, and tests from backend commit `4d3b6a06`.
- Pinnable targets:
  - owned coins;
  - wishlist coins;
  - Coin Sets;
  - manual calendar events;
  - auction lots in `watching` or `bidding`.
- `GET /api/quick-access`, newest pin first.
- Idempotent `PUT /api/quick-access/:type/:id` and `DELETE /api/quick-access/:type/:id`.
- Lifecycle cleanup and state-preserving transitions.
- Compatibility with `CoinSet.PinnedAt`, the existing `PUT /sets/:id` pin contract, sidebar data, and the five-set cap.
- Vue/TypeScript API types and endpoint functions for the committed backend contract.
- One authenticated `/quick-access` page and one top-level sidebar navigation entry.
- Shared client state for list, pin, unpin, eligibility lookup, lifecycle reconciliation, and logout cleanup.
- Pin controls on owned/wishlist coin detail, Coin Set detail, eligible auction-lot detail, and manual calendar-event detail.
- Deep links from mixed Quick Access items to the existing resource surfaces.

### Explicitly deferred

- Pin controls outside resource detail/header surfaces.
- Reordering, folders, labels, notes, sharing, pagination, or a global cap for non-set targets.
- Sold coins, terminal auction lots, auto-generated auction calendar events, and public/follower resources.
- Any Go/API contract change beyond consuming backend commit `4d3b6a06`.

## 2. User Scenarios & Testing

### User Story 1 — Read one mixed Quick Access list (Priority: P1)

As an authenticated collector, I want one list containing every eligible item I pinned, so a future client can render Quick Access without querying each resource separately.

**Why this priority**: The mixed, typed read contract is the feature boundary all later controls depend on.

**Independent Test**: Seed one eligible target of each type for one user, pin them at distinct times, call `GET /quick-access`, and verify a user-scoped, newest-first, discriminated response with no model leakage.

**Acceptance Scenarios**:

1. **Given** eligible pins across all four target types, **When** the owner requests Quick Access, **Then** every item is returned once in descending `pinnedAt` order.
2. **Given** coin pins for an owned coin and a wishlist coin, **When** the list is hydrated, **Then** both have `type=coin` and derive `classification=owned|wishlist` from current coin state.
3. **Given** auction-lot pins in `watching` and `bidding`, **When** the list is hydrated, **Then** both have `type=auction_lot` and derive the current status.
4. **Given** another user's pins and targets, **When** the current user requests Quick Access, **Then** no row or target data belonging to the other user is returned.

---

### User Story 2 — Pin and unpin eligible targets idempotently (Priority: P1)

As a collector, I want to pin or unpin a supported target from its detail/header control, so it appears or disappears from Quick Access predictably.

**Why this priority**: This is the write contract that future resource controls will call.

**Independent Test**: PUT the same eligible target twice, verify one row and an unchanged timestamp; DELETE twice, verify both responses succeed and the row is absent.

**Acceptance Scenarios**:

1. **Given** an eligible owned target with no pin, **When** the owner PUTs its Quick Access path, **Then** one pin is created and the API returns `201`.
2. **Given** an already-pinned target, **When** the owner PUTs it again, **Then** the API returns `200`, creates no duplicate, and preserves the original `pinnedAt`.
3. **Given** a pinned target, **When** the owner DELETEs it, **Then** the API returns `204` and removes it.
4. **Given** an already-unpinned or inaccessible target identifier, **When** the user DELETEs it, **Then** the API still returns `204` without revealing whether another user owns that target.
5. **Given** a foreign, missing, or currently ineligible target, **When** a user attempts to pin it, **Then** the API returns the same generic `404` contract.

---

### User Story 3 — Preserve pins across eligible state changes (Priority: P1)

As a collector, I want a pin to follow the same resource when its display classification changes but it remains eligible.

**Why this priority**: A pin represents resource identity, not a snapshot of one status.

**Independent Test**: Pin a wishlist coin, purchase it, and verify the same pin row/timestamp hydrates as owned; repeat for an auction lot moving between watching and bidding.

**Acceptance Scenarios**:

1. **Given** a pinned wishlist coin, **When** it is purchased and `isWishlist` becomes false, **Then** the same `coin` pin and original `pinnedAt` remain and hydration changes to `owned`.
2. **Given** a pinned auction lot, **When** it moves `watching -> bidding` or `bidding -> watching`, **Then** the same pin and original `pinnedAt` remain.
3. **Given** an unpinned target that changes classification, **When** the change completes, **Then** no pin is created implicitly.

---

### User Story 4 — Remove pins when targets leave eligibility or are deleted (Priority: P1)

As a collector, I want Quick Access to remove stale items automatically, so it never links to sold, terminal, or deleted targets.

**Why this priority**: Lifecycle integrity is required for a trustworthy navigation surface.

**Independent Test**: Pin each target, trigger every terminal/deletion path, and verify the pin is removed in the same transaction as synchronous mutations or in the same per-lot sync transaction.

**Acceptance Scenarios**:

1. **Given** a pinned coin, **When** it is sold through the sell endpoint or any supported update path transitions `isSold` to true, **Then** its pin is removed.
2. **Given** a pinned auction lot, **When** a manual override or provider sync changes it to `won`, `lost`, or `passed`, **Then** its pin is removed.
3. **Given** any pinned coin, Coin Set, auction lot, or manual calendar event, **When** the target is deleted, **Then** its pin is removed.
4. **Given** a previously terminal target becomes eligible again through a later manual correction, **When** the correction completes, **Then** the old pin is not restored; the user must pin it again.

---

### User Story 5 — Preserve existing Coin Set behavior (Priority: P1)

As a collector already using pinned sets, I want the unified feature to retain my pins, timestamps, sidebar behavior, and five-set limit.

**Why this priority**: Existing user data and behavior cannot regress during unification.

**Independent Test**: Start from a legacy database with populated `coin_sets.pinned_at`, run migration twice, use both old and new pin endpoints, and verify one authoritative pin per set, mirrored timestamps, and the five-set cap.

**Acceptance Scenarios**:

1. **Given** legacy sets with `PinnedAt`, **When** the migration runs, **Then** matching `quick_access_pins` rows are created with the original timestamps.
2. **Given** migration runs more than once, **When** reconciliation repeats, **Then** no duplicates or timestamp changes occur.
3. **Given** a Coin Set is pinned or unpinned through either `PUT /sets/:id` or `/quick-access/coin_set/:id`, **Then** the unified row and compatibility `CoinSet.PinnedAt` mirror change atomically.
4. **Given** five Coin Sets are pinned, **When** a sixth is pinned through either endpoint, **Then** the request fails with the existing set-cap client message and other target types do not count toward the cap.
5. **Given** a legacy database already contains more than five pinned sets, **When** backfill runs, **Then** all existing pins are preserved, but no additional set may be pinned until the count falls below five.

---

### User Story 6 — Navigate one mixed Quick Access page (Priority: P1)

As an authenticated collector, I want a single responsive page for every pinned resource, so I can open important coins, sets, lots, and events without searching their source pages.

**Independent Test**: Mock one item of each discriminator, open `/quick-access`, and verify newest-first rendering, resource-specific metadata, and the exact target navigation for each card.

**Acceptance Scenarios**:

1. **Given** a mixed Quick Access response, **When** the page loads, **Then** all four variants render in server order using their typed payload and no unsafe type assertions.
2. **Given** no pins, **When** the page loads, **Then** it shows a useful empty state rather than a blank list.
3. **Given** a list request fails, **When** the page renders, **Then** it shows a recoverable error and retry action without discarding a previously loaded list.
4. **Given** a user selects a coin, set, auction lot, or calendar event, **When** navigation occurs, **Then** the client opens `/coin/:id`, `/sets/:id`, `/auctions?lot=:id`, or `/calendar?event=:id` respectively.
5. **Given** desktop or installed-PWA layout, **When** the sidebar opens, **Then** one top-level `Quick Access` item navigates to `/quick-access`.

---

### User Story 7 — Pin from existing detail surfaces (Priority: P1)

As a collector, I want a consistent pin affordance where I am already reviewing an eligible resource, so I can update Quick Access without leaving that workflow.

**Independent Test**: Mount each detail surface with eligible and ineligible fixtures, toggle the control, and verify the unified PUT/DELETE endpoint, pressed state, shared-list update, and error behavior.

**Acceptance Scenarios**:

1. **Given** an unsold owned or wishlist coin, **When** its header renders, **Then** a labeled pin control reflects shared Quick Access state; sold coins and follower detail do not expose it.
2. **Given** a Coin Set, **When** its existing pin control is toggled, **Then** it uses the unified endpoint while preserving the five-set message and refreshing the legacy pinned-set sidebar projection.
3. **Given** a watching or bidding lot, **When** its detail modal renders, **Then** a pin control is available; won, lost, and passed lots do not expose it.
4. **Given** a manual calendar event, **When** its detail drawer renders, **Then** a pin control is available; auction-origin events do not expose it.
5. **Given** a pin/unpin request fails, **When** the error is shown, **Then** the prior pressed state and shared list remain intact.

---

### User Story 8 — Keep client state correct across lifecycle and identity changes (Priority: P1)

As a collector, I want Quick Access to remain user-scoped and current as targets change or I sign out, so stale or cross-account pins never remain visible.

**Independent Test**: Exercise purchase, sold/terminal/delete mutations, deep links, logout, user switch, and a delayed list response; verify preserved pins remain, removed pins disappear, and stale async results cannot repopulate cleared state.

**Acceptance Scenarios**:

1. **Given** a pinned wishlist coin is purchased or a pinned lot moves between watching and bidding, **When** the mutation succeeds, **Then** the same item remains and its displayed classification/status refreshes.
2. **Given** a pinned target is sold, becomes terminal, or is deleted, **When** the mutation succeeds, **Then** it disappears from shared Quick Access state without requiring a full application reload.
3. **Given** `/auctions?lot=:id` or `/calendar?event=:id`, **When** the target is owned and available, **Then** its existing detail modal/drawer opens even if it is absent from the currently filtered list/month.
4. **Given** an invalid, missing, or foreign deep-link target, **When** loading fails, **Then** the parent page remains usable and reveals no target details.
5. **Given** logout or an account switch while a refresh is pending, **When** the old request resolves, **Then** cleared state remains empty and cannot leak the prior user's pins.

## 3. Edge Cases

- Pin timestamps that tie are ordered deterministically by pin row ID descending.
- An empty list returns `{"items":[]}`, never `null`.
- Unknown target types and non-positive IDs return `400`.
- Coin eligibility is exactly `isSold=false`; current `isWishlist` determines only `owned` versus `wishlist`.
- Auction eligibility is exactly `watching|bidding`; terminal statuses are not returned or pinnable.
- Calendar eligibility requires `AuctionEvent.Origin=manual`. Auto-generated auction events are not pinnable even if they remain in the calendar.
- Existing calendar events predate an origin field. Migration classifies events referenced by auction lots as `auction`; unlinked events as `manual`. This is a conservative best-effort inference because historical provenance was not stored.
- A manual event linked to an auction lot before this migration may be conservatively classified as `auction`; this migration ambiguity is recorded as Risk R6 in `plan.md`.
- Polymorphic targets cannot use one database foreign key. All deletion and eligibility transitions therefore require explicit service-layer cleanup and regression tests.
- A malformed/stale pin discovered during hydration is omitted and logged; `GET` does not mutate data. Repair belongs to lifecycle paths or startup reconciliation, avoiding a side-effecting read.
- Quick Access is global authenticated state. Its module-level state requires an explicit `clear()` that also invalidates pending refreshes; component unmount alone is not a security boundary.
- Existing auction-lot query deep links remain canonical. Calendar event deep links use the same query-driven modal/drawer pattern rather than adding resource-detail routes.
- Closing a query-opened lot/event removes only its own query parameter with `router.replace`, preserving unrelated query state and avoiding a reopen loop.

## 4. Requirements

### 4.1 Functional Requirements

- **FR-001**: The system MUST persist pins in one user-owned polymorphic `QuickAccessPin` entity; it MUST NOT add pin columns to coins, auction lots, or calendar events.
- **FR-002**: Supported persisted target types MUST be exactly `coin`, `coin_set`, `auction_lot`, and `calendar_event`.
- **FR-003**: The database MUST enforce uniqueness on `(user_id, target_type, target_id)`.
- **FR-004**: The database MUST index `(user_id, pinned_at)` for newest-first retrieval and index target lookup for cleanup.
- **FR-005**: `PUT /quick-access/:type/:id` MUST be idempotent: create once, preserve the original `pinnedAt` on repeat, and return `201` for creation or `200` for an existing pin.
- **FR-006**: `DELETE /quick-access/:type/:id` MUST be idempotent and return `204` whether or not the caller currently has that pin.
- **FR-007**: `GET /quick-access` MUST return authenticated-user pins ordered by `pinnedAt DESC, pinId DESC`.
- **FR-008**: Pin creation MUST validate current ownership and eligibility before writing.
- **FR-009**: Foreign, missing, and ineligible pin attempts MUST share one generic `404` client response so resource existence is not disclosed.
- **FR-010**: Internal repository, migration, and hydration errors MUST be logged server-side and exposed only as generic client errors.
- **FR-011**: Coins MUST be pinnable when `isSold=false`, including both owned (`isWishlist=false`) and wishlist (`isWishlist=true`) states.
- **FR-012**: Coin hydration MUST derive `classification=owned|wishlist`; classification MUST NOT be persisted on the pin row.
- **FR-013**: Purchasing a wishlist coin MUST preserve the existing coin pin and original `pinnedAt`.
- **FR-014**: Transitioning a coin to sold MUST remove its pin; transitioning out of sold later MUST NOT restore it.
- **FR-015**: Deleting a coin MUST remove its pin.
- **FR-016**: Coin Sets MUST remain subject to a maximum of five pinned sets per user. Non-set pins MUST NOT count toward this cap.
- **FR-017**: Both the legacy set update endpoint and the unified endpoint MUST enforce the same five-set rule and existing error message.
- **FR-018**: `QuickAccessPin` MUST become the authoritative pin record after migration. `CoinSet.PinnedAt` MUST remain only as a transactionally synchronized compatibility mirror for existing set responses/sidebar behavior.
- **FR-019**: Startup migration MUST backfill missing Coin Set pins from `CoinSet.PinnedAt` using the original timestamp, then reconcile the compatibility mirror from authoritative unified rows. It MUST be idempotent and fail startup on migration error.
- **FR-020**: A repeated Coin Set pin through either endpoint MUST preserve the original pin timestamp; unpin then re-pin MUST assign a new timestamp.
- **FR-021**: Auction lots MUST be pinnable only while status is `watching` or `bidding`.
- **FR-022**: Auction hydration MUST derive current `status=watching|bidding`; status MUST NOT be duplicated on the pin row.
- **FR-023**: `watching <-> bidding` transitions MUST preserve the pin and original timestamp.
- **FR-024**: Transitions to `won`, `lost`, or `passed`, whether manual or sync-driven, MUST remove the lot pin. Later reversal MUST NOT restore it.
- **FR-025**: Deleting an auction lot MUST remove its pin.
- **FR-026**: Calendar events MUST gain an explicit origin discriminator `manual|auction`; only `manual` events are pinnable.
- **FR-027**: Events created by `POST /calendar/events` MUST have origin `manual`; events auto-created from auction lots MUST have origin `auction`.
- **FR-028**: Deleting a manual calendar event MUST remove its pin.
- **FR-029**: Deleting a Coin Set MUST remove its unified pin and existing set memberships in one transaction.
- **FR-030**: Every synchronous multi-step target mutation and pin cleanup MUST be atomic. Sync-driven auction transitions MUST update the lot and cleanup its pin within the same per-lot database transaction.
- **FR-031**: Hydration MUST use bounded batch queries by target type rather than one target query per pin.
- **FR-032**: The API MUST return explicit DTOs and MUST NOT serialize full GORM resource models or unrelated private fields.
- **FR-033**: All three Quick Access handler methods MUST have Swagger annotations and the generated OpenAPI artifacts MUST match registered routes.
- **FR-034**: Routes MUST be registered under the existing JWT-protected API group.
- **FR-035**: Implementation MUST follow Handler -> Service -> Repository -> Database with constructor injection and composition-root wiring.
- **FR-036**: All lifecycle hooks MUST be covered across direct update, dedicated action, deletion, and provider-sync sibling paths; testing only the new endpoints is insufficient.
- **FR-037**: The frontend MUST define an exact `QuickAccessTargetType` union and a discriminated `QuickAccessItem` union matching §4.2; each variant MUST require exactly its matching payload at compile time.
- **FR-038**: Quick Access API calls MUST live in a dedicated frontend endpoint module, use the shared authenticated Axios client, and expose typed list, pin, and unpin functions without changing the backend contract.
- **FR-039**: The authenticated router MUST provide `/quick-access`, and the page MUST render loading, recoverable error/retry, empty, and populated states on desktop and PWA viewports.
- **FR-040**: The populated page MUST preserve server order and render resource-specific labels and metadata from the DTO without fetching every target again.
- **FR-041**: Quick Access item navigation MUST map exactly to `/coin/:id`, `/sets/:id`, `/auctions?lot=:id`, and `/calendar?event=:id`.
- **FR-042**: The global sidebar MUST include one reorderable top-level `Quick Access` entry linked to `/quick-access`; existing Sets submenu pins MUST remain unchanged.
- **FR-043**: A shared `useQuickAccess` composable MUST own module-level items/loading/error state and typed `refresh`, `pin`, `unpin`, `isPinned`, and `clear` operations. Pin/unpin state MUST update only after a successful server response.
- **FR-044**: `clear()` MUST empty all user-specific Quick Access state and invalidate in-flight refreshes so a late response cannot repopulate state after logout or account switch.
- **FR-045**: Authenticated application bootstrap MUST refresh Quick Access once, the Quick Access page MAY explicitly retry/refresh, and the feature MUST NOT poll.
- **FR-046**: Unsold owned and wishlist coin detail headers MUST expose an accessible pin control. Sold and follower coin detail surfaces MUST NOT expose the control.
- **FR-047**: Coin Set detail MUST migrate its existing pin control to the unified endpoint while preserving its current UX, five-set error, compatibility mirror refresh, and pinned Sets submenu behavior.
- **FR-048**: Auction-lot detail MUST expose the control only for `watching|bidding`; calendar-event detail MUST expose it only for `origin=manual`.
- **FR-049**: Every pin control MUST provide a stable accessible name, `aria-pressed`, disabled/busy protection, gold active state, and a user-visible error while retaining the prior state on failure.
- **FR-050**: Successful frontend mutations that preserve eligibility (wishlist purchase; watching/bidding transition) MUST refresh the affected Quick Access DTO, while sold, terminal, and delete mutations MUST remove or refresh the affected item before the workflow is considered complete.
- **FR-051**: `/calendar?event=:id` MUST fetch the event by ID and open its detail drawer independently of the loaded month; closing it MUST remove only `event` from the query.
- **FR-052**: Existing `/auctions?lot=:id` behavior MUST remain canonical and MUST be covered as the auction Quick Access destination.
- **FR-053**: Missing, invalid, foreign, or ineligible deep-link targets MUST fail safely on the parent page without leaking details or entering a navigation loop.
- **FR-054**: Frontend delivery MUST include endpoint, composable, page, navigation, control, lifecycle, logout/user-switch, deep-link, accessibility, and PWA-responsive regression coverage and pass the frontend Quality Gate; no Go file may change.

### 4.2 Typed API Contract

#### `GET /api/quick-access`

```json
{
  "items": [
    {
      "type": "coin",
      "id": 42,
      "pinnedAt": "2026-09-10T12:30:00Z",
      "coin": {
        "name": "Trajan Denarius",
        "classification": "wishlist",
        "primaryImageUrl": "/uploads/..."
      }
    },
    {
      "type": "coin_set",
      "id": 7,
      "pinnedAt": "2026-09-09T18:00:00Z",
      "coinSet": {
        "name": "Five Good Emperors",
        "setType": "goal",
        "color": "#6b7280",
        "icon": "crown"
      }
    },
    {
      "type": "auction_lot",
      "id": 19,
      "pinnedAt": "2026-09-08T18:00:00Z",
      "auctionLot": {
        "title": "Hadrian Aureus",
        "status": "bidding",
        "auctionHouse": "CNG",
        "saleDate": "2026-10-01T00:00:00Z",
        "auctionEndTime": "2026-10-01T19:00:00Z",
        "imageUrl": "https://..."
      }
    },
    {
      "type": "calendar_event",
      "id": 11,
      "pinnedAt": "2026-09-07T18:00:00Z",
      "calendarEvent": {
        "title": "Local Coin Show",
        "auctionHouse": "",
        "startDate": "2026-11-14T00:00:00Z",
        "endDate": "2026-11-15T00:00:00Z"
      }
    }
  ]
}
```

`type` is the discriminator. Exactly one matching payload property (`coin`, `coinSet`, `auctionLot`, or `calendarEvent`) MUST be non-null. Resource payloads are owner-safe display DTOs, not embedded models.

#### `PUT /api/quick-access/:type/:id`

- Request body: none.
- `201 Created`: newly pinned `QuickAccessItemDTO`.
- `200 OK`: already pinned; same DTO and unchanged `pinnedAt`.
- `400 Bad Request`: invalid type/ID, or Coin Set cap reached (`{"error":"you can pin up to 5 sets"}`).
- `404 Not Found`: `{"error":"Quick access item not found"}` for missing, foreign, or ineligible target.
- `500 Internal Server Error`: generic failure.

#### `DELETE /api/quick-access/:type/:id`

- Request body: none.
- `204 No Content`: pin absent after request.
- `400 Bad Request`: invalid type/ID.
- `500 Internal Server Error`: generic failure.

### 4.3 Key Entities

- **QuickAccessPin**: `id`, `userId`, `targetType`, `targetId`, `pinnedAt`, `createdAt`. One row per user/type/target. It stores identity and ordering only.
- **AuctionEvent.Origin**: `manual|auction`, used solely to distinguish pinnable user-created events from automatically generated auction calendar rows.
- **QuickAccessItemDTO**: typed owner-safe union envelope with one resource-specific display payload.

## 5. Success Criteria

- **SC-001**: A legacy database with pinned Coin Sets migrates with zero lost pins and byte-equivalent timestamps; a second migration pass changes no rows.
- **SC-002**: Repeating PUT 100 times for one target produces exactly one row and one unchanged `pinnedAt`.
- **SC-003**: A mixed list containing all supported types is returned in deterministic newest-first order with no cross-user records.
- **SC-004**: Wishlist purchase and watching/bidding transitions preserve the same pin ID and timestamp.
- **SC-005**: Every sold/terminal/deletion path leaves zero matching pin rows.
- **SC-006**: Both Coin Set pin entry points reject a sixth set while allowing unlimited eligible pins of other types.
- **SC-007**: Targeted model, migration, repository, service, handler, lifecycle, route/OpenAPI drift, architecture, and full Go tests pass.
- **SC-008**: A typed four-item fixture renders in server order and each item opens its canonical resource destination.
- **SC-009**: All four eligible detail surfaces toggle through the unified endpoint, retain state on failure, and reflect successful changes without a full reload.
- **SC-010**: Logout, account switch, and a delayed pre-logout refresh leave shared Quick Access state empty.
- **SC-011**: Calendar and auction query deep links open the requested owned target outside the active month/filter and fail safely for unavailable targets.
- **SC-012**: `npm run type-check`, targeted Vitest suites, full `npm test`, `npm run lint`, and `npm run build` pass with no `src/api/` changes.

## 6. Assumptions

- Personal-scale use means an unpaginated list is acceptable for v1.
- Timestamps are UTC.
- The existing `CoinSet.PinnedAt` field cannot be removed in this feature because the current sidebar contract consumes it.
- The frontend maps the committed response to a TypeScript discriminated union keyed by `type`; the backend contract remains frozen for this phase.
- No ADR is required: this is an additive data model inside the existing Go service boundary, with no new external service or security posture change.
