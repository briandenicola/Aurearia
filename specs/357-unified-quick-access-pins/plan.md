# Implementation Plan: Unified Quick Access Pins

**Branch**: `357-unified-quick-access-pins` | **Date**: 2026-09-10 | **Spec**: `./spec.md`
**Input**: Feature specification from `specs/357-unified-quick-access-pins/spec.md`
**Phase boundary**: Backend implementation only. Frontend is deferred and `src/web/` MUST NOT be modified.

## 1. Summary

Create one `quick_access_pins` table keyed by authenticated user, typed target, and target ID. A `QuickAccessService` validates ownership/eligibility, preserves original pin time on idempotent PUT, hydrates explicit display DTOs in bounded batches, and coordinates the Coin Set compatibility mirror. Existing coin, set, auction, sync, and calendar lifecycle services remove pins when targets become ineligible or are deleted.

The migration also adds `AuctionEvent.Origin` so only manual events are pinnable. `QuickAccessPin` is authoritative after migration; `CoinSet.PinnedAt` remains a transactionally synchronized legacy projection.

## 2. Technical Context

**Language/Version**: Go 1.26.6
**Primary Dependencies**: Gin, GORM, `github.com/glebarez/sqlite`
**Storage**: SQLite via GORM `AutoMigrate`; one new table and one additive event-origin column
**Testing**: `go test`, `go vet`, existing architecture and route/OpenAPI drift tests
**Target Platform**: Self-hosted single-node Linux API
**Project Type**: Go REST API within the existing full-stack repository
**Performance Goals**: One pin query plus at most one batch hydration query per target type; no N+1 queries
**Constraints**: Additive migration; preserve Coin Set timestamps/cap/sidebar contract; owner isolation; no frontend edits
**Scale/Scope**: Personal-scale, normally fewer than 100 Quick Access pins per user

## 3. Constitution Check

*Gate evaluated before design and re-checked after the data/API design below.*

- **Principle I — Clear Layered Architecture**: New flow is Handler -> QuickAccessService -> repositories -> SQLite. Lifecycle business rules remain in services. Multi-step mutations are transactional. Calendar mutations touched by this feature move behind a `CalendarService`; auction delete/status paths move fully behind `AuctionLotService`.
- **Principle II — Service Boundaries**: Go-only feature; no Python agent or direct Vue/Python coupling.
- **Principle III — Strict Types and Explicit Contracts**: Typed target/status/origin enums, explicit DTOs, Swagger on all public methods, regenerated OpenAPI.
- **Principle IV — Simple Complete Changes**: One polymorphic table rather than four pin columns; complete lifecycle coverage across sibling mutation paths; no frontend scope.
- **Principle V — Security/Auth/Privacy**: JWT-protected routes, owner-scoped queries, foreign/missing/ineligible pin attempts share a generic 404, no full models serialized.
- **Principle VIII — Documented Decisions**: Contract and migration truth are captured in this plan and `.squad/decisions/inbox/maximus-quick-access-design.md`.
- **Principle IX — Automated Enforcement**: Migration-order, ownership, idempotency, lifecycle, architecture, and route/OpenAPI drift tests are required.
- **§17 / §21**: Backend build, vet, tests, Swagger sync, workflow-contract coverage, and exact lifecycle regression paths are merge gates.

**Result**: PASS. No constitutional waiver and no ADR required.

## 4. Before-Work Design Review

**Verdict**: APPROVED TO IMPLEMENT — BACKEND ONLY, subject to the contracts in spec §4 and decisions D1-D12 below.

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

No frontend command is required because `src/web` is out of scope and must remain untouched.

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
| R10 | Scope expands into frontend | Explicit phase boundary and no tasks under `src/web` |

## 11. Complexity Tracking

No constitution violation. The new `CalendarService` and sync transaction refactor are required to keep lifecycle cleanup out of handlers/repositories and satisfy Principle I; they are not optional architectural expansion.
