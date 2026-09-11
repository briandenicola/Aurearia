# Research: Structured Coin Storage Trays

All technical unknowns are resolved.

## 1. Additive SQLite schema and migration

**Decision**: Add `storage_locations.type` with effective/default value `standard`, nullable `rows`/`columns`, and `coins.storage_slot`. Keep the existing association `constraint:-`. After additive AutoMigrate and a guarded legacy backfill, create:

```sql
CREATE UNIQUE INDEX IF NOT EXISTS idx_coins_storage_location_slot_unique
ON coins(storage_location_id, storage_slot)
WHERE storage_slot IS NOT NULL;
```

Migration returns any error and is idempotent. A real legacy-schema fixture verifies preserved IDs, owners, names, associations, null slots, and repeat execution.

**Rationale**: This preserves legacy meaning and avoids the table-rebuild/foreign-key failures documented in `.squad/decisions.md`. A partial index permits unlimited Standard-location assignments because their slots are null while making concurrent tray claims authoritative.

**Alternatives considered**:
- Physical FK from coin to location: rejected due to SQLite/GORM rebuild risk and the existing `constraint:-` precedent.
- Unique index including owner: unnecessary because location IDs are globally unique.
- Application-only occupancy check: rejected because two writers can race.
- Separate tray/position tables: rejected as disproportionate and migration-heavy for rectangular, fixed-capacity v1.

**Evidence**: `src/api/models/coin.go`, `src/api/models/storage_location.go`, `src/api/database/database.go`, `src/api/database/feature353_migration_order_regression_test.go`, `feature356_migration_order_regression_test.go`, `feature357_migration_regression_test.go`, `.squad/decisions.md`.

## 2. Slot representation and assignment semantics

**Decision**: Persist a nullable one-based integer `storage_slot`. Derive coordinates using:

```text
row    = floor((slot - 1) / columns) + 1
column = ((slot - 1) mod columns) + 1
slot   = ((row - 1) * columns) + column
```

Standard or no location requires null slot. Tray requires slot in `1..rows*columns`.

**Rationale**: One scalar supports an efficient unique constraint and exact deterministic geometry without persisting redundant row/column values.

**Alternatives considered**:
- Zero-based slot: rejected by the normative user contract.
- Store row and column separately: rejected because it duplicates derivable state and broadens uniqueness/migration logic.

## 3. Atomic moves and conflict classification

**Decision**: Service pre-validates for readable errors, then repository applies location+slot in one transaction. The named unique-index violation maps to `slot_occupied`; other database errors remain internal. A failed target claim rolls back and leaves the old assignment unchanged.

**Rationale**: Pre-checks improve UX but cannot close races. The database decides the winner. Transactional release-and-claim satisfies FR-022.

**Alternatives considered**:
- Clear old assignment before claiming new slot: rejected because failure loses the prior placement.
- Serialize all writes globally: rejected as unnecessary and less scalable.

**Evidence**: `src/api/services/coin_service.go`, `src/api/repository/coin_repository.go`, Constitution Principle I.

## 4. Shared assignment boundary and sibling flows

**Decision**: `CoinService` owns a reusable typed storage-assignment validator/orchestrator. Direct create/update, bulk, duplicate, AI intake, Quick Capture promotion, and deep-identification/wishlist paths must use it or a transaction-aware equivalent owned by that service.

`CoinIntakeService.CommitDraft` currently creates via `CoinRepository` and must be refactored. Quick Capture and deep-identification already hold/call `CoinService`, but receive explicit contract regressions.

**Rationale**: Current direct and bulk paths prove that handler/repository shortcuts diverge. Constitution §21 requires sibling workflow-contract coverage.

**Alternatives considered**:
- Duplicate validation in each handler/service: rejected as drift-prone.
- Disallow all storage fields in intake/import: compatible as a temporary UI policy, but server-side requests still need the common validation contract.

**Evidence**: `src/api/services/coin_service.go`, `coin_intake_service.go`, `quick_capture_service.go`, `deep_identification_proposal.go`, `src/api/handlers/bulk.go`, `routes_protected.go`.

## 5. Backward compatibility and duplication

**Decision**:
- Omitted location type on create means Standard; omission on update preserves the current type (and therefore remains compatible for legacy Standard rows).
- Existing location IDs/names/associations remain unchanged.
- Omitted new coin fields remain unslotted Standard behavior.
- Duplicate preserves a Standard Location as today, but clears both location and slot for a tray source.

**Rationale**: The spec explicitly demands legacy compatibility and explicitly forbids tray assignment inheritance. Preserving Standard duplication minimizes unrelated behavior change.

**Alternatives considered**:
- Clear location for every duplicate: safe but breaks existing tested Standard behavior.
- Copy tray location but clear slot: invalid because a tray assignment requires a slot.

**Evidence**: `src/api/repository/coin_repository.go`, `src/api/handlers/coin_handler_test.go` (current location-preservation assertion), spec FR-024/FR-046.

## 6. Occupancy and aggregate API shape

**Decision**:
- `GET /api/storage-locations/{id}/occupancy?coinId={optional}` returns tray dimensions/capacity and occupied slot numbers only; `currentCoinSlot` identifies the owner coin's retained selection.
- `GET /api/storage-trays` returns every owner tray, including empty trays, with positioned minimal coin data and minimal image references.

**Rationale**: Occupancy avoids leaking unrelated coin details into forms. Aggregate avoids per-tray/per-well N+1 calls and supplies the read-only view exactly what it needs.

**Alternatives considered**:
- Include occupancy only in the location list: possible, but current-coin recognition and reload cadence become awkward.
- Reuse `/coins` pagination: rejected because it omits empty trays and does not represent fixed placement.
- One request per tray: rejected due to N+1 behavior.

**Evidence**: `src/api/handlers/storage_location.go`, `src/api/routes_protected.go`, `src/web/src/api/endpoints/collection.ts`.

## 7. Stable error envelope

**Decision**: Extend the current simple error contract additively:

```json
{
  "error": "Slot is already occupied",
  "code": "slot_occupied",
  "message": "Choose another slot and try again.",
  "field": "storageSlot",
  "count": 1
}
```

Only applicable fields are emitted. Codes are `validation_error`, `slot_occupied`, `tray_occupied`, and `location_referenced`. Cross-owner and absent resources share generic 404.

**Rationale**: Existing clients can keep reading `error`; new clients get stable recovery semantics.

**Alternatives considered**:
- Replace `error` with a new RFC 7807 envelope: rejected as a breaking, project-wide change.

**Evidence**: `src/api/handlers/swagger_types.go`, spec FR-044.

## 8. Museum Tray reuse boundary

**Decision**: Reuse or minimally extract the felt surface, circular well, image selection/private-media fallback, sizing, focus, and reduced-motion presentation from `MuseumTray.vue` and `MuseumTrayWell.vue`. Compose them in a separate `StorageTrayGrid.vue`. Do not feed physical slots into `MuseumTray.vue` or `trayLayout.ts` drawer/packing helpers.

Museum Tray remains unchanged in responsive columns, pagination, swipe, face toggle, preferences, and collection membership. Every `Coin → TrayCoin` adapter continues forwarding required caption/placeholder fields where that adapter is used.

**Rationale**: This reconciles the reuse skill with higher-authority FR-030–FR-032. Low-level visual reuse prevents stylesheet duplication; a separate composition prevents semantic coupling.

**Alternatives considered**:
- Add a “physical mode” to `MuseumTray.vue`: rejected because it creates a high-risk conditional component and could change sibling responsive semantics.
- Copy felt/well CSS: prohibited by spec and skill.
- Use `trayLayout.ts` to pack coins: rejected because physical slot order is persisted, not computed.

**Evidence**: `.squad/skills/museum-tray-reuse/SKILL.md`, `src/web/src/components/tray/MuseumTray.vue`, `MuseumTrayWell.vue`, `TrayControls.vue`, `src/web/src/utils/trayLayout.ts`, `src/web/src/pages/TrayViewPage.vue`, their existing tests.

## 9. Fixed geometry on narrow screens

**Decision**: Render `repeat(columns, fixed/minimum-well-size)` in a single contained horizontal scroller. Rows and columns never change at breakpoints. Coordinates remain visible/readable and every cell remains reachable.

**Rationale**: For a 20-column tray, horizontal containment preserves physical geometry and minimum actionable size better than shrinking all wells below 44px.

**Alternatives considered**:
- Responsive column counts: explicitly prohibited.
- Whole-tray scaling only: can make coordinates and targets too small at 20 columns.
- Pagination: explicitly prohibited.

## 10. Accessibility and missing media

**Decision**: Use grid/gridcell semantics. Occupied wells alone are actions, support pointer/touch/Enter/Space, visible focus, meaningful coin+coordinate names, and ≥44×44 targets. Empty wells are non-actionable grid cells named by coordinate. Missing/unavailable images retain occupied styling and an icon/text fallback via authenticated media behavior.

**Rationale**: Meets FR-033 and FR-036–FR-038 without misleading controls or privacy leaks.

**Evidence**: `MuseumTrayWell.vue`, `AuthenticatedImage.vue`, Constitution Principle VI.

## 11. Exact validation and documentation commands

**Decision**: Use repository-native gates listed in `quickstart.md`, including Go build/vet/test/race, frontend lint/type/test/build, targeted Playwright, `task openapi`, and git diff verification. Python gates are not required because no agent files are in scope.

**Rationale**: These match `Taskfile.yml`, `src/web/package.json`, `.github/workflows/ci.yml`, Constitution §17/§21.
