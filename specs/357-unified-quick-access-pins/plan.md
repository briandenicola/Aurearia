# Implementation Plan: Unified Quick Access Pins

**Branch**: `357-unified-quick-access-pins` | **Date**: 2026-09-10 | **Spec**: `./spec.md`
**Input**: Feature specification from `specs/357-unified-quick-access-pins/spec.md`
**Phase boundary**: Backend commit `4d3b6a06` is complete and frozen. This revision authorizes the Vue/TypeScript frontend phase only; `src/api/` MUST NOT be modified.

## 1. Summary

Consume the completed `quick_access_pins` backend through a strict TypeScript discriminated union, one shared Quick Access composable, a responsive `/quick-access` page, one global navigation entry, and eligible detail-surface controls. Preserve existing Coin Set sidebar behavior while adding canonical deep links and explicit client-state cleanup across lifecycle mutations, logout, and user switches.

The backend remains authoritative: pin eligibility, timestamps, ownership, set cap, lifecycle cleanup, and `AuctionEvent.Origin` are not reimplemented in Vue.

## 2. Technical Context

**Language/Version**: Vue 3, TypeScript 5.9
**Primary Dependencies**: Vue Router, shared Axios client, lucide-vue-next
**Storage**: No new browser persistence; authenticated server state held in memory only
**Testing**: Vitest/Vue Test Utils, `vue-tsc --build`, ESLint, Vite production build
**Target Platform**: Desktop browser and installed PWA/mobile viewport
**Project Type**: Vue SPA consuming the existing Go REST API
**Performance Goals**: One bootstrap/list request, no polling, no per-card target hydration
**Constraints**: Preserve backend FR-001-FR-036 and D1-D12; preserve existing pinned Sets submenu; no Go edits
**Scale/Scope**: Personal-scale, normally fewer than 100 Quick Access items

## 3. Constitution Check

*Gate evaluated before design and re-checked after the data/API design below.*

- **Principle II — Service Boundaries**: Vue calls only the existing Go `/api/quick-access` contract.
- **Principle III — Strict Types and Explicit Contracts**: The API response becomes a compile-time discriminated union; Docker-equivalent `vue-tsc --build` is blocking.
- **Principle IV — Simple Complete Changes**: One shared composable and one page serve all types; existing resource surfaces receive narrow controls rather than parallel pin systems.
- **Principle V — Security/Auth/Privacy**: No pin state is persisted in localStorage. Logout/user switch clears state and invalidates pending responses.
- **Principle VI — Consistent UX**: Existing button, page-header, card, design-token, icon, PWA, and sidebar-reorder patterns are reused.
- **Principle VIII — Documented Decisions**: Frontend authorization and architecture are recorded in this plan and `.squad/decisions/inbox/maximus-quick-access-frontend.md`.
- **Principle IX — Automated Enforcement**: Contract, lifecycle, deep-link, logout, accessibility, and responsive behavior receive targeted tests.
- **§17 / §21**: Type-check, lint, targeted/full tests, production build, workflow-contract coverage, and exact-path regression tests are merge gates.

**Result**: PASS. No constitutional waiver and no ADR required.

## 4. Before-Work Design Review

**Verdict**: APPROVED TO IMPLEMENT — FRONTEND ONLY, consuming backend commit `4d3b6a06`, subject to spec FR-037-FR-054 and decisions D13-D19 below.

The earlier backend-only verdict and D12 remain the historical authorization boundary for commit `4d3b6a06`. The repository owner's later explicit frontend authorization is reconciled under Constitution §0 by updating the active spec, then this plan, then `tasks.md`; it does not amend locked backend requirements or decisions.

### Reviewed current-state facts

1. `CoinSet.PinnedAt` exists, `SetService.UpdateSet` preserves its first timestamp, and `CountPinned` enforces five per user.
2. `Coin` identity is stable across wishlist purchase (`isWishlist` flips on the same row); sale marks `isSold=true`.
3. Auction lots use one stable row across `watching`, `bidding`, and terminal statuses. Status changes occur through both manual service calls and provider-sync repository upserts.
4. `AuctionEvent` currently lacks provenance; both manual handler creation and auction auto-creation write the same model.
5. Several calendar and auction handler paths currently call repositories directly. Lifecycle cleanup makes those paths part of this feature's blast radius; they must be routed through services rather than adding handler-level cleanup.
6. SQLite cannot enforce a foreign key from one polymorphic `(type,id)` pair to four tables. Lifecycle integrity must therefore be service-enforced and regression-tested.
7. Feature number `356` is retired by the landed valuation-history work even though no `specs/356-*` directory exists. Per `specs/README.md`, numbers are never reused; `357` is the next valid feature number.

## 5. Design Decisions

### D1. One authoritative polymorphic table

`quick_access_pins` stores only user, target identity, and original pin time. It does not duplicate names, status, classification, or navigation paths.

### D2. Strict target enum and database constraints

```go
type QuickAccessTargetType string

const (
    QuickAccessTargetCoin          QuickAccessTargetType = "coin"
    QuickAccessTargetCoinSet       QuickAccessTargetType = "coin_set"
    QuickAccessTargetAuctionLot    QuickAccessTargetType = "auction_lot"
    QuickAccessTargetCalendarEvent QuickAccessTargetType = "calendar_event"
)

type QuickAccessPin struct {
    ID         uint                  `gorm:"primaryKey" json:"-"`
    UserID     uint                  `gorm:"not null;index;uniqueIndex:idx_quick_access_owner_target" json:"-"`
    TargetType QuickAccessTargetType `gorm:"type:varchar(24);not null;check:target_type IN ('coin','coin_set','auction_lot','calendar_event');uniqueIndex:idx_quick_access_owner_target;index:idx_quick_access_target" json:"type"`
    TargetID   uint                  `gorm:"not null;uniqueIndex:idx_quick_access_owner_target;index:idx_quick_access_target" json:"id"`
    PinnedAt   time.Time             `gorm:"not null;index:idx_quick_access_owner_time,sort:desc" json:"pinnedAt"`
    CreatedAt  time.Time             `json:"-"`
}
```

The implementation MUST verify the generated SQLite indexes/constraint through migration tests rather than relying only on struct tags.

### D3. Idempotent original-time semantics

Pin uses a transaction and conflict-safe insert. On a uniqueness conflict it reloads and returns the existing row; it never updates `pinned_at`. Unpin then re-pin creates a new row/time. Stable list ordering is `pinned_at DESC, id DESC`.

### D4. Typed hydration, not model serialization

`QuickAccessItemDTO` contains the discriminator and exactly one typed payload pointer. Hydration:

1. loads the user's pin rows;
2. groups target IDs by type;
3. performs no more than four owner-scoped batch queries;
4. derives coin classification and lot status from live rows;
5. omits/logs any stale or ineligible row without mutating on GET;
6. restores the original pin order in memory.

Only fields specified in spec §4.2 are returned.

### D5. Coin eligibility follows identity

Any unsold owned row is eligible. `isWishlist` is display classification, not pin identity. Purchase preserves. Every false->true `isSold` path removes. Delete removes. No automatic restoration occurs.

### D6. Coin Set compatibility and five-cap truth

After migration:

- `quick_access_pins` is authoritative.
- `coin_sets.pinned_at` remains a compatibility mirror only.
- Both `PUT /sets/:id` and the new endpoint delegate to the same `QuickAccessService` set-pin operation.
- Set pin/unpin writes the unified row and `CoinSet.PinnedAt` in one transaction.
- The cap counts authoritative `coin_set` pin rows only.
- Existing sidebar order remains a frontend concern; its current `PinnedAt` data remains unchanged. The mixed Quick Access API independently orders newest first.

### D7. Safe, convergent Coin Set migration

After `AutoMigrate`, a startup migration transaction:

1. inserts missing `coin_set` pin rows from non-null `coin_sets.pinned_at`, preserving timestamps;
2. updates each pinned set mirror to the authoritative row timestamp;
3. clears a set mirror where no authoritative row exists;
4. fails startup on any error;
5. is idempotent on restart.

Backfill preserves all legacy rows even if legacy drift exceeds five; enforcement blocks only new set pins until the count is below five.

### D8. Manual calendar events need explicit provenance

Add `AuctionEvent.Origin` with `manual|auction`.

- Handler-created events write `manual`.
- Auto-created lot events write `auction`.
- Legacy migration marks events referenced by any auction lot as `auction`; all others remain/default to `manual`.
- Only `manual` is pinnable.

This is a conservative migration. Historical manual events already linked to a lot cannot be distinguished from auto-generated events and may be classified as `auction` (Risk R6).

### D9. Lifecycle cleanup belongs in transaction-owning services

- `CoinService`: cleanup on sold transition, sell action, and delete; purchase explicitly preserves.
- `SetService`: cleanup on delete; legacy pin update delegates to QuickAccess.
- `AuctionLotService`: cleanup on manual terminal status and delete.
- `AuctionWatchlistSyncService`: refactor per-lot upsert so provider status update and terminal cleanup share one transaction.
- `CalendarService` (new): owns calendar CRUD and pin cleanup on delete; handler becomes thin.

Handlers MUST NOT issue Quick Access repository writes.

### D10. Generic, non-leaking errors

Pin validation maps missing, foreign, and ineligible to `ErrQuickAccessTargetNotFound`, returned as `404 {"error":"Quick access item not found"}`. Set-cap retains the existing message. Unexpected errors are logged and returned generically.

### D11. No global pin cap in v1

Only Coin Sets retain their established five-item cap. Adding a mixed/global cap without product direction would create new behavior and potentially conflict with existing pins.

### D12. Frontend contract is frozen but implementation is deferred

The Go DTO is intentionally shaped for a later TypeScript discriminated union. No `src/web` file, frontend task, navigation control, or sidebar merge is part of this feature phase.

### D13. Frontend phase consumes the frozen backend

D12 remains true for the completed backend phase. The newly authorized frontend phase starts from commit `4d3b6a06` and may edit only `src/web/` plus feature/team documentation. No Go, migration, route, DTO, or OpenAPI change is permitted.

### D14. One strict union and endpoint module

Add `src/web/src/types/quick-access.ts` with literal target types and four item variants whose nonmatching payload keys are absent/optional-never as needed for safe narrowing. Add `src/web/src/api/endpoints/quickAccess.ts` using the shared Axios instance and re-export both modules through existing barrels.

### D15. Shared state is a lifecycle-managed singleton

Use `useQuickAccess.ts` as module-level authenticated state because App navigation, the page, and multiple detail surfaces must coordinate immediately. It exposes typed `refresh`, `pin`, `unpin`, `isPinned`, and `clear`. A `201` PUT prepends the returned new item, a `200` PUT replaces the existing item in place, and successful DELETE removes it; this preserves server ordering without inventing the backend's hidden tie-break ID. `clear` increments a request generation so late pre-logout/pre-switch responses are ignored. No localStorage and no polling.

### D16. Quick Access is a top-level destination

Add one reorderable `Quick Access` sidebar item and `/quick-access` route/page. The page preserves backend order, uses existing page-header/card/empty-state patterns, and renders directly from DTO payloads without N+1 target fetches. Existing pinned sets stay under Sets as a compatibility affordance; they are not moved or duplicated into another submenu.

### D17. Existing detail surfaces own controls

- Coin: `CoinDetailHeaderActions.vue`, shown only for authenticated unsold owner/wishlist detail; never follower detail.
- Coin Set: migrate the existing `SetDetailPage.vue` button to unified PUT/DELETE, then refresh both set detail and `usePinnedSets`.
- Auction lot: `AuctionLotDetailModal.vue`, shown only for `watching|bidding`.
- Calendar event: existing event drawer in `CalendarPage.vue`, shown only for `origin=manual`.

All use the same Pin/PinOff semantics, `aria-pressed`, busy protection, active gold styling, and error toast/message. Backend eligibility remains final.

### D18. Canonical deep links follow existing parent surfaces

Cards navigate with `router.push` to `/coin/:id`, `/sets/:id`, `/auctions?lot=:id`, and `/calendar?event=:id`. Auctions retain their existing fetch-by-ID query handling. Calendar gains the same fetch-by-ID behavior, independent of active month, and removes only `event` with `router.replace` on close. Invalid/foreign targets leave the parent page usable.

### D19. Lifecycle reconciliation is explicit

Bootstrap refreshes once when authenticated, and App watches the authenticated user ID so an in-tab identity change clears old state before refreshing the new account. Detail mutations update or refresh Quick Access after success: purchase and watching/bidding transitions retain and rehydrate; sold, terminal, and delete paths remove/refresh. Logout clears both `useQuickAccess` and existing `usePinnedSets`; request-generation invalidation prevents delayed user-A data from appearing for user B.

## 6. Data Migration Order

1. Enable existing SQLite pragmas as current startup does.
2. `AutoMigrate` `AuctionEvent` with `origin` and `QuickAccessPin`.
3. Verify/execute legacy event-origin backfill.
4. Execute Coin Set pin backfill/reconciliation in one transaction.
5. Continue startup only after all migration helpers return nil.

Required regression fixture: create legacy tables/models without the new table/origin, seed pinned/unpinned sets plus linked/unlinked events, then run the real migration sequence twice. Assert row preservation, timestamps, origins, uniqueness, indexes, and idempotency.

The production model-constructor drift guard in `feature353_migration_order_regression_test.go` must be updated when `QuickAccessPin` is added to the live `AutoMigrate` list.

## 7. API and Route Design

### Handler

`QuickAccessHandler`:

- `List`
- `Pin`
- `Unpin`

All parse only HTTP concerns, map service sentinels to status codes, and log unexpected errors.

### Routes

Under `protected`:

```go
protected.GET("/quick-access", quickAccessHandler.List)
protected.PUT("/quick-access/:type/:id", quickAccessHandler.Pin)
protected.DELETE("/quick-access/:type/:id", quickAccessHandler.Unpin)
```

Write routes use the existing authenticated write-rate limiter if consistent with neighboring protected mutations.

### OpenAPI

Add Swagger definitions/annotations, regenerate with `task openapi`, and run:

```text
go test -v -run TestRegisteredAPIRoutesAreDocumentedInOpenAPI .
```

No route exemption is permitted.

## 8. Project Structure

```text
specs/357-unified-quick-access-pins/
├── spec.md
├── plan.md
└── tasks.md

src/api/
├── models/
│   ├── quick_access_pin.go                    # NEW
│   └── showcase.go                            # MOD: AuctionEvent.Origin
├── repository/
│   ├── quick_access_repository.go             # NEW
│   ├── quick_access_repository_test.go        # NEW
│   ├── coin_repository.go                     # MOD: batch DTO hydration support if needed
│   ├── set_repository.go                      # MOD: tx/mirror support
│   ├── auction_lot_repository.go              # MOD: tx-aware sync path
│   └── auction_event_repository.go            # MOD: scopes/batch/origin
├── services/
│   ├── quick_access_service.go                # NEW
│   ├── quick_access_service_test.go           # NEW
│   ├── calendar_service.go                    # NEW
│   ├── coin_service.go                        # MOD: sold/delete cleanup
│   ├── set_service.go                         # MOD: compatibility delegation/delete
│   ├── auction_lot_service.go                 # MOD: manual terminal/delete cleanup
│   └── auction_watchlist_sync_service.go      # MOD: sync terminal cleanup
├── handlers/
│   ├── quick_access.go                        # NEW
│   ├── quick_access_test.go                   # NEW
│   ├── calendar.go                            # MOD: use CalendarService
│   └── auction_lots.go                        # MOD: service-owned delete/status
├── database/
│   ├── database.go                            # MOD: AutoMigrate + migration helpers
│   └── feature357_migration_regression_test.go # NEW
├── deps.go                                    # MOD: constructor wiring
├── routes_protected.go                        # MOD: routes
└── route_openapi_drift_test.go                # existing gate, normally no edit

docs/
├── docs.go                                    # regenerated
├── swagger.json                               # regenerated
└── swagger.yaml                               # regenerated

openapi.yaml                                   # regenerated/synchronized by repository task

src/web/src/
├── api/
│   ├── client.ts                             # MOD: endpoint barrel export
│   └── endpoints/quickAccess.ts              # NEW: typed GET/PUT/DELETE
├── types/
│   ├── index.ts                              # MOD: type barrel export
│   ├── quick-access.ts                       # NEW: discriminated union
│   └── auctions.ts                           # MOD: calendar origin contract
├── composables/
│   ├── useQuickAccess.ts                     # NEW: shared state + invalidation
│   └── __tests__/useQuickAccess.test.ts       # NEW
├── pages/
│   ├── QuickAccessPage.vue                   # NEW
│   ├── CoinDetailPage.vue                    # MOD: pin + lifecycle sync
│   ├── SetDetailPage.vue                     # MOD: unified endpoint + projections
│   ├── AuctionsPage.vue                      # MOD: lifecycle refresh, deep-link regression
│   └── CalendarPage.vue                      # MOD: manual pin + event deep link
├── components/
│   ├── coin/CoinDetailHeaderActions.vue      # MOD: pin affordance
│   └── auction/AuctionLotDetailModal.vue     # MOD: eligible pin affordance
├── router/index.ts                           # MOD: /quick-access
├── App.vue                                   # MOD: nav/bootstrap/logout
└── relevant __tests__/                       # MOD/NEW: exact workflow coverage
```

Exact generated Swagger paths are determined by the existing `task openapi` workflow; do not hand-edit generated files unless that workflow already does so.

## 9. Test Strategy

### Model/migration

- Table columns, check constraint, unique/index presence.
- Legacy set backfill timestamp preservation.
- Mirror reconciliation.
- Existing >5 set preservation.
- Event-origin backfill.
- Second-run idempotency.
- Real `AutoMigrate` list drift guard updated.

### Repository

- Unique owner/type/target.
- User-scoped newest-first list with deterministic ties.
- Idempotent create preserving timestamp.
- Scoped delete.
- Batch hydration queries and foreign-owner exclusion.
- `WithTx` behavior.
- Unique named in-memory SQLite DSNs per `.squad/skills/go-sqlite-test-isolation/SKILL.md`.

### Service

- Eligibility matrix for all types.
- Generic not-found/ineligible behavior.
- Five-set cap and both entry points.
- DTO exact shape/no model leakage.
- Coin purchase preservation; sold/delete cleanup.
- Lot watching/bidding preservation; manual and sync terminal cleanup.
- Event and set deletion cleanup.
- Transaction rollback tests: force cleanup failure and assert target mutation rolls back.

### Handler/contract

- Authenticated 200/201/204 paths.
- 400 invalid type/ID/cap.
- generic 404.
- generic 500.
- Swagger annotations and route/OpenAPI drift.

### Quality Gate

```powershell
Set-Location src\api
go build ./...
go vet ./...
go test ./...
go test -v -run TestRegisteredAPIRoutesAreDocumentedInOpenAPI .
```

Frontend gate from `src/web`:

```powershell
npm run type-check
npx vitest run <feature-targeted-test-files>
npm test
npm run lint
npm run build
```

Also run `git diff --name-only` and fail the frontend review if any `src/api/` path changed.

## 10. Risks and Mitigations

| ID | Risk | Mitigation |
|---|---|---|
| R1 | Coin Set has two physical representations | Unified row is authoritative; mirror is transactionally synchronized; migration reconciles; tests exercise both APIs |
| R2 | Polymorphic rows have no FK | Explicit cleanup in every lifecycle service; owner-safe hydration omits stale rows; exact sibling-path tests |
| R3 | Provider sync bypasses manual service path | Refactor to a transaction-aware per-lot sync write that includes cleanup |
| R4 | PUT races create duplicates or alter time | Unique index + conflict-safe create + reload; concurrency/idempotency tests |
| R5 | Five-set cap races | Count and insert inside one transaction; serialize SQLite writer; verify concurrent sixth-pin behavior |
| R6 | Legacy event provenance is unknowable | Conservative linked=`auction`, unlinked=`manual` backfill; document ambiguity; no data deletion |
| R7 | Hydration leaks full private models | Purpose-built DTO projections only; response contract tests |
| R8 | Startup backfill silently fails | Helpers return errors; startup fails; migration-order test calls real helpers |
| R9 | New route drifts from Swagger | Mandatory route/OpenAPI drift test and `task openapi` |
| R10 | Frontend accidentally changes frozen backend | Frontend-only phase guard; fail review on any `src/api/` diff |
| R11 | Module-level state leaks across users | Explicit clear + request generation invalidation + logout/user-switch tests |
| R12 | Existing pinned Sets sidebar drifts | Unified set control refreshes both Quick Access and `usePinnedSets`; regression tests preserve submenu |
| R13 | Modal/drawer targets are not route-addressable | Canonical query deep links with fetch-by-ID and safe close semantics |
| R14 | Lifecycle cleanup succeeds server-side but UI stays stale | Successful sibling mutations explicitly reconcile shared state |
| R15 | Detail controls expose ineligible targets | Typed eligibility checks plus backend final validation and negative rendering tests |
| R16 | Mobile header/control crowding | Reuse compact icon-button patterns and test narrow viewport rendering |

## 11. Complexity Tracking

No constitution violation. The shared singleton is justified by cross-route coordination and is bounded by an explicit security lifecycle; a new Pinia store would add migration cost without improving this feature's ownership model. No ADR is required because this remains inside the existing Vue SPA boundary and established module-level composable pattern.
