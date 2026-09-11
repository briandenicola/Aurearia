# Implementation Plan: Structured Coin Storage Trays

**Branch**: `feat/structured-storage-trays` | **Date**: 2026-09-11 | **Spec**: `specs/358-structured-storage-trays/spec.md`
**Input**: Feature specification at `C:\Users\brian.denicolafamily\Code\AncientCoins\specs\358-structured-storage-trays\spec.md`

> The setup script incorrectly derived `specs/feat/structured-storage-trays`. That output is non-authoritative. Every feature artifact for this plan belongs in `specs/358-structured-storage-trays/`.

## Summary

Extend the existing owner-scoped `StorageLocation` resource with an additive `standard|tray` type and nullable tray dimensions, and extend `Coin` with a nullable, one-based `storageSlot`. Preserve every existing location, ID, and assignment as a Standard Location. Enforce type/slot/range/ownership rules in services and enforce race-safe slot exclusivity with a SQLite partial unique index plus transactional repository writes.

Add type-aware Settings management, exact-slot add/edit assignment, Standard-only bulk assignment, tray-safe duplication, and an authenticated `/storage-trays` page backed by owner-scoped occupancy and aggregate APIs. The physical view renders exact persisted geometry—including empty wells—using a new fixed-grid composition around shared low-level Museum Tray felt/well/image primitives; it does not change or reuse Museum Tray's packing, drawer, swipe, sorting, filtering, or responsive layout semantics.

## Technical Context

**Language/Version**: Go 1.26.1; TypeScript 5.9; Vue 3.5  
**Primary Dependencies**: Gin, GORM, `github.com/glebarez/sqlite`; Vue Router, Pinia, Axios, Vite/PWA, `lucide-vue-next`  
**Storage**: SQLite; existing GORM AutoMigrate in `src/api/database/database.go`; additive columns plus an explicitly created partial unique index  
**Testing**: Go `testing`; Vitest + Testing Library/Vue Test Utils; Playwright workflow tests  
**Target Platform**: Self-hosted web application and installable desktop/mobile PWA  
**Project Type**: Go REST API + Vue SPA; Python agent is present but is not changed  
**Performance Goals**: A 20×20 tray renders exactly 400 wells without server-side N+1 reads; one aggregate request returns all owner trays and positioned minimal render data  
**Constraints**: Handler → Service → Repository → Database; owner scoping; no physical FK migration; no arbitrary slot inference; exact row-major geometry; 44×44 actionable targets; no Museum Tray semantic regression  
**Scale/Scope**: Maximum 100 owner locations under the existing service cap; each tray has 1–20 rows, 1–20 columns, and at most 400 slots  
**Clarifications**: None. Type defaults, bounds, slot arithmetic, errors, migration, bulk behavior, duplication behavior, and narrow-screen behavior are resolved by the feature spec and research.

## Constitution Check — Pre-Design

| Gate | Status | Plan evidence |
|---|---|---|
| Principle I: layered architecture | PASS | Handlers parse/map only; services own all storage rules; repositories own GORM, aggregate reads, indexes, and transactions. |
| Principle II: service boundaries | PASS | Vue calls only Go REST; no Python-agent or direct database access is introduced. |
| Principle III: strict types/contracts | PASS | Typed Go DTOs and Vue interfaces; Swagger annotations and generated OpenAPI updates; no `any`/suppression. |
| Principle IV: simple, complete, proportional | PASS | Additive schema and focused fixed-grid composition; sibling create/update/import/intake/duplicate/bulk paths are explicitly covered. |
| Principle V: security/privacy | PASS | All reads/writes are authenticated and owner-scoped; cross-owner IDs map to 404; aggregate data is minimal and private images keep authenticated handling. |
| Principle VI: UX | PASS | Existing tokens/icons, fixed geometry, PWA containment, keyboard/focus/assistive semantics, reduced motion. |
| Principles VII/IX and §17 | PASS | Go, frontend, migration, contract, race, OpenAPI, and browser workflow gates are planned. |
| Principle VIII: decisions | PASS | Durable design decisions are captured in `research.md`; no constitutional waiver or new third-party service is needed. |

**Pre-design gate status: PASS.** No violation or waiver is required.

## Project Structure

### Planning artifacts

```text
specs/358-structured-storage-trays/
├── spec.md
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
└── contracts/
    ├── openapi.yaml
    └── ui-interaction.md
```

### Existing source areas and planned additions during implementation

```text
src/api/
├── models/
│   ├── storage_location.go
│   └── coin.go
├── repository/
│   ├── storage_location_repository.go
│   ├── storage_location_repository_test.go  # planned new focused test
│   ├── coin_repository.go
│   └── coin_repository_test.go
├── services/
│   ├── storage_location_service.go
│   ├── storage_location_service_test.go
│   ├── coin_service.go
│   ├── coin_service_test.go                 # planned if not present at implementation time
│   ├── coin_intake_service.go
│   ├── coin_intake_service_test.go
│   └── quick_capture_service_test.go
├── handlers/
│   ├── storage_location.go
│   ├── storage_location_handler_test.go     # planned new focused test
│   ├── coin_requests.go
│   ├── coins.go
│   ├── coin_handler_test.go
│   ├── bulk.go
│   ├── bulk_handler_test.go                 # planned new focused test
│   └── swagger_types.go
├── database/
│   ├── database.go
│   └── feature358_migration_regression_test.go # planned new fixture
├── routes_protected.go
├── docs/{docs.go,swagger.json,swagger.yaml}
└── main.go

src/web/src/
├── api/endpoints/{collection.ts,coins.ts}
├── api/client.ts
├── types/{collection.ts,coin.ts}
├── components/
│   ├── CoinForm.vue
│   ├── BulkLocationPickerModal.vue
│   ├── tray/
│   │   ├── MuseumTray.vue
│   │   ├── MuseumTrayWell.vue
│   │   ├── TraySurface.vue              # extracted felt surface, if needed
│   │   ├── TrayWell.vue                 # low-level visual primitive, if needed
│   │   └── StorageTrayGrid.vue          # fixed physical geometry
│   ├── settings/SettingsDataSection.vue
│   └── __tests__/...
├── pages/StorageTraysPage.vue               # planned new page
├── pages/__tests__/StorageTraysPage.test.ts # planned new test
├── router/index.ts
└── App.vue

src/web/e2e/workflows/
├── tray.spec.ts                         # Museum Tray regression
├── coin-form.spec.ts
└── storage-trays.spec.ts                    # planned new workflow

docs/
├── features.md
├── features/storage-trays.md                # planned new guide
├── api-reference.md
└── openapi.json
```

The names `TraySurface.vue` and `TrayWell.vue` describe the intended extraction boundary, not a requirement to split both files if `MuseumTrayWell.vue` can accept the needed coordinate/accessibility props without semantic change.

## Phase 0: Research Outcome

The decisions, alternatives, and exact evidence paths are in `research.md`. Key outcomes:

1. Add `type`, `rows`, and `columns` to `storage_locations`; add `storage_slot` to `coins`.
2. Default/backfill every legacy location to `standard`; leave all legacy coin slots `NULL`.
3. Create `idx_coins_storage_location_slot_unique` as a partial unique index on `(storage_location_id, storage_slot) WHERE storage_slot IS NOT NULL`.
4. Keep service pre-validation for readable errors, but treat the database constraint as authoritative for concurrent claims.
5. Use repository transactions for release-and-claim, duplicate creation, and bulk location changes.
6. Provide:
   - `GET /api/storage-locations/:id/occupancy?coinId=…`
   - `GET /api/storage-trays`
7. Reuse/extract low-level Museum Tray presentation only; build a separate fixed-grid composition.

## Phase 1: Design and Contracts

### 1. Persistence and migration

- Add typed constants `standard` and `tray` in `src/api/models/storage_location.go`; use a database default of `standard` for new/legacy omitted values.
- Add nullable `Rows *int`, `Columns *int`; add nullable `StorageSlot *int` to `src/api/models/coin.go`.
- Preserve `Coin.StorageLocation`'s existing `constraint:-` and do not add a physical FK that can trigger a SQLite table rebuild.
- In `src/api/database/database.go`, after additive AutoMigrate:
  1. update only null/empty legacy type values to `standard`;
  2. verify no invalid/mixed rows;
  3. create the partial unique index with `CREATE UNIQUE INDEX IF NOT EXISTS`;
  4. return every error; never continue on partial failure.
- Add a legacy-schema migration regression fixture modeled after `src/api/database/feature357_migration_regression_test.go`: seed two owners, locations, assigned coins, IDs/timestamps, and rerun migration twice. Assert preservation, Standard classification, null slots, index presence, and idempotence.
- Test failure behavior with duplicate non-null seeded slots before index creation: migration must fail visibly rather than discard or remap data.

### 2. Repository layer

- Extend `StorageLocationRepository` owner scopes for:
  - type-aware CRUD;
  - occupied count;
  - occupancy slot-number query excluding/recognizing a current coin;
  - all-trays aggregate in bounded queries (trays plus positioned minimal coins/images);
  - transaction-bound variants for resize/delete guards.
- Extend `CoinRepository` with a single field-specific assignment operation that updates location and slot together and reloads associations. Never use `Save` on a partially loaded aggregate.
- Translate only the named SQLite unique-index violation into a repository sentinel; do not classify unrelated database errors as `slot_occupied`.
- Make Standard/clear bulk assignment update both fields (`storage_location_id`, `storage_slot=NULL`) in one owner-scoped statement/transaction.
- Make duplicate behavior type-aware: preserve an owned Standard Location for backward compatibility; clear both fields when the source location is a tray.

### 3. Service layer

- `StorageLocationService` owns trimmed case-insensitive names, the existing owner limit, type normalization, dimensions, capacity, occupied resize/delete guards, and stable errors:
  - validation: invalid name/type/rows/columns;
  - conflicts: `ErrTrayOccupied`, `ErrLocationReferenced`;
  - inaccessible owner resource: not found.
- `CoinService` becomes the only assignment policy boundary. A typed, presence-aware assignment value validates:
  - coin and location ownership;
  - Standard requires null slot;
  - tray requires an integer slot in `1..rows*columns`;
  - current coin may retain its own slot;
  - another occupant conflicts;
  - omitted update fields preserve both current values;
  - explicit-null location clears both;
  - changing location without an explicit compatible slot cannot retain a stale slot.
- Apply assignment validation and transactional persistence to direct create/update, duplicate, and bulk.
- Refactor `CoinIntakeService.CommitDraft`, which currently writes through `CoinRepository`, to invoke the shared CoinService create/transaction path or a transaction-aware shared validator. Extend its allowed override mapping for the two storage fields only if intake is intended to submit them.
- Keep Quick Capture promotion and deep-identification wishlist creation behind `CoinService`; add contract tests proving they cannot persist invalid tray assignments. Imports and any future API clients receive the same DTO/service rules.

### 4. Handler and API layer

- Extend create/update DTOs in `src/api/handlers/storage_location.go` and `coin_requests.go` with explicit JSON nullability and presence tracking.
- Keep legacy omission behavior: omitted storage type means `standard`; omitted coin slot means unslotted Standard behavior. For update, distinguish omitted from explicit null for both location and slot.
- Keep handlers thin and map service errors consistently:
  - 400 `{code:"validation_error", field, message}`;
  - 404 generic not found;
  - 409 `{code:"slot_occupied"|"tray_occupied"|"location_referenced", message, count?}`;
  - 500 generic.
- Add owner-scoped occupancy and aggregate handlers and register them in `src/api/routes_protected.go`.
- Add Swagger annotations and schemas, then regenerate `src/api/docs/*` and `docs/openapi.json` using `task openapi`.

### 5. Settings, forms, bulk, and typed API client

- Update `src/web/src/types/collection.ts` and `coin.ts`, `api/endpoints/collection.ts`, and `api/endpoints/coins.ts`; preserve re-export through `src/web/src/api/client.ts`.
- Settings creation explicitly chooses Standard Location or Coin Tray. Require and constrain numeric dimensions for trays; hide/clear them for Standard. List type labels, dimensions, and `occupied/capacity`.
- Editing an occupied tray disables dimension changes and explains why while still permitting rename. Surface structured 400/409 messages. Deleting referenced locations explains the assignment count.
- `CoinForm.vue` groups or labels location types. Selecting a tray loads occupancy and renders every one-based coordinate. Occupied slots are disabled except the edited coin's own slot. Location changes reset stale slot state. Client validation is helpful but server validation remains authoritative.
- Include `storageSlot` in `NULLABLE_FIELDS` in `src/web/src/api/endpoints/coins.ts`; remove nested read-only objects from mutation payloads.
- `BulkLocationPickerModal.vue` either filters trays from selectable items or renders them disabled with the explicit text “Tray assignments require choosing a slot on each coin.” Server-side bulk assignment independently rejects tray IDs.

### 6. Fixed physical Storage Trays view

- Add authenticated route `/storage-trays` named `storage-trays` in `src/web/src/router/index.ts`; add **Storage Trays** as a Collection child in `src/web/src/App.vue`. Preserve existing `/tray` and “Tray” Museum Tray navigation unchanged.
- `StorageTraysPage.vue` requests the aggregate once, renders every tray including empty trays, and provides empty/loading/retry/partial-image states.
- `StorageTrayGrid.vue` iterates slot numbers `1..rows*columns`, derives row/column from persisted columns, and uses `grid-template-columns: repeat(columns, …)`. It never filters, sorts, packs, paginates, or reflows wells.
- Narrow screens use a contained horizontal scroller (preferred for 20-column coordinate readability) around one fixed grid; no CSS breakpoint changes column count. The page itself must not break horizontal layout.
- Extract felt surface and well/coin visuals from `MuseumTray.vue`/`MuseumTrayWell.vue` only as needed. Museum Tray remains a compatibility consumer with unchanged props, rendering, breakpoints, drawers, swipe behavior, and preferences.
- Occupied wells use authenticated images/fallback, navigate to `coin-detail`, support Enter/Space, have visible focus and `aria-label="<coin>, row <r>, column <c>"`, and expose a 44×44 minimum target. Empty wells expose “Empty, row …, column …” as non-interactive grid cells. Coordinate text/pattern supplements color.
- Respect `prefers-reduced-motion`; no physical-tray animation is required.

### 7. Tests and documentation

- Model/repository/service tests: all validation combinations, case-insensitive uniqueness, owner isolation, first/last slots, current-coin exception, race/constraint translation, atomic failed move, Standard clear, resize/delete conflicts, duplicate semantics, aggregate minimality, empty trays.
- Handler tests: DTO omission/null behavior, all statuses/codes, cross-owner 404, occupancy privacy, aggregate owner scoping, bulk tray rejection, Swagger-visible schemas.
- Migration tests: fresh database, legacy fixture, second run, failure before index, existing association preservation.
- Vue tests:
  - `SettingsDataSection.test.ts`: type-specific controls, occupancy display, conflict handling;
  - `CoinForm.test.ts`: conditional slot picker, coordinates, disabled occupancy, own slot, reset and error states;
  - `BulkLocationPickerModal` test: Standard selectable and tray disabled/explained;
  - `StorageTrayGrid`/`StorageTraysPage` tests: exact well count and positions, empty/occupied semantics, keyboard, missing image, 1×1 and 20×20;
  - existing `MuseumTray.test.ts`, `MuseumTrayWell.test.ts`, and `TrayViewPage.test.ts` remain regression gates.
- Playwright: add `src/web/e2e/workflows/storage-trays.spec.ts`; retain `tray.spec.ts` and `coin-form.spec.ts`.
- Update `README.md`, `docs/features.md`, new `docs/features/storage-trays.md`, `docs/api-reference.md`, generated API artifacts, and relevant `docs/testing.md` workflow notes.

## Phase 2: Implementation Sequence (Planning Only)

1. **Migration foundation**: models → additive migration/backfill/index → legacy and fresh-database regression tests.
2. **Persistence contracts**: owner-scoped repository occupancy/aggregate/assignment methods and constraint translation.
3. **Business rules**: StorageLocationService lifecycle rules and CoinService assignment value/transaction orchestration.
4. **Sibling workflow convergence**: direct create/update, intake, Quick Capture/deep-identification proofs, duplicate, then bulk.
5. **HTTP contracts**: DTO presence/null handling, thin handlers, status/code mapping, protected routes, Swagger.
6. **Typed web contracts**: Vue types and endpoint wrappers through `api/client.ts`.
7. **Mutation UX**: Settings, CoinForm occupancy picker, and bulk restriction.
8. **Read-only UX**: shared low-level visual extraction, fixed `StorageTrayGrid`, page, route, and navigation.
9. **Regression and documentation closure**: Museum Tray characterizations, workflow E2E, docs, OpenAPI generation, and all quality gates.

Each step depends on the preceding contract layer. Do not start the fixed-grid UI against invented data or add handler-side validation while service work is incomplete.

## Post-Design Constitution Check

| Gate | Status | Design result |
|---|---|---|
| Layering and transactions | PASS | Every rule is service-owned; every query/transaction is repository-owned; handlers only adapt HTTP. |
| Service boundaries and types | PASS | No agent changes; explicit Go/TS/OpenAPI contracts cover nullability and one-based semantics. |
| Complete workflow coverage | PASS | Settings, direct form/API, bulk, duplicate, intake, Quick Capture/deep identification, aggregate view, navigation, docs, and regressions are included. |
| Security and privacy | PASS | Owner scopes and 404 semantics apply to CRUD, occupancy, aggregate, assignment, and media references. |
| UX/PWA/accessibility | PASS | Exact fixed geometry, contained scrolling, non-color cues, keyboard/focus, 44px targets, reduced motion, and retry states are contractual. |
| Migration and release integrity | PASS | Additive/idempotent migration, no new FK, legacy fixture, partial unique index, and full CI/OpenAPI gates are specified. |

**Post-design gate status: PASS.** No unresolved clarification, constitutional violation, or complexity exception remains.

## Complexity Tracking

No constitutional violation is introduced. The new fixed-grid component and two read contracts are the minimum complete separation needed to preserve Museum Tray semantics while representing persisted physical geometry.

## Planning Boundaries

- This plan does not create `tasks.md`.
- This plan does not implement or modify production code.
- `.specify/scripts/powershell/update-agent-context.ps1 -AgentType copilot` was run with
  `SPECIFY_FEATURE=358-structured-storage-trays`; it updated the generated feature
  context block in `.github/copilot-instructions.md`. No Constitution §18.2 locked
  file was modified.
