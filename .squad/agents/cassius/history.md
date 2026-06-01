# Project Context

- **Owner:** Brian
- **Project:** Ancient Coins backend — Go 1.26 / Gin / GORM / SQLite
- **Architecture:** Layered Handler → Service → Repository → Database with constructor injection and architecture tests.

## Core Context

- Cassius owns backend implementation. Durable backend rules: thin handlers, service-owned business logic, repository-owned GORM queries, scopes for ownership/public filters, sentinel service errors, Swagger annotations, and DI wiring in `main.go`.
- Scheduler/run-log pattern established across valuation, wishlist/availability, auction-ending, and related admin surfaces: configurable settings, manual trigger, run history table, and production diagnostics where needed.
- Time-sensitive auction queries use rolling `(now, now+24h]` windows, explicit NULL guards, and case-insensitive status comparison. Real-data diagnostics should accompany query fixes.
- Security/backend patterns: validate ownership before CPU/memory-heavy decode operations; mock httpx response methods synchronously in Python tests; circle image clipping lives in stdlib-only `src/api/capture/` and is gated to obverse/reverse uploads when `circleClip=true`.

## Recent Updates

- **2026-05-31:** #217 Go shared collection tool layer completed: internal token service/middleware, six internal tool endpoints, keyword gate removed, confirm-gated write flow preserved.
- **2026-05-31:** #217 Python ReAct collection agent completed end-to-end with LangChain tools calling Go internal endpoints via short-lived internal token; compound collection/value questions now compose within one reasoning turn.
- **2026-06-01:** #218 external tool server foundational stack implemented: API key capabilities, enablement toggle, capability middleware, per-key rate limit, `/api/v1/tools` route group, handlers, OpenAPI discovery, and external commit journal metadata. Build/vet/test passed.
- **2026-06-01:** Collection chat multi-container callback issue documented. `AGENT_INTERNAL_CALLBACK_URL` must point from agent container to API service (e.g. `http://coins:8080`), not default localhost; startup warning added for release+localhost.
- **2026-06-01:** v1→v2 migration audit found only additive schema changes; AutoMigrate/backfill safe and rollback-safe.

## Learnings

- **Storage Location API Pattern (2026-06-01):** Added per-user `StorageLocation` lookup table and nullable `Coin.StorageLocationID` FK. Backend files: `models/storage_location.go`, `repository/storage_location_repository.go`, `services/storage_location_service.go`, `handlers/storage_location.go`; `Coin` preloads now include `StorageLocation` where coin associations are returned. Routes: `GET/POST /api/storage-locations`, `PUT/DELETE /api/storage-locations/:id`. Delete is guarded: referenced locations return 409 Conflict with the number of coins using the location; coins must be reassigned first. Coin create/update validates that any non-null `storageLocationId` belongs to the requesting user; update accepts explicit `null` to clear the FK.
- **SQLite/GORM Coin FK Migration Gotcha (2026-06-01):** Adding a physical FK constraint to the existing `coins` table can make GORM rebuild the table; with `PRAGMA foreign_keys=ON`, dropping the old table fails if child rows (`coin_images`, `coin_tags`, etc.) reference it. For nullable `Coin` lookup FKs added after launch, keep the `*_id` column and preload association but tag the association `constraint:-`; migrate the lookup table before `Coin`, and enforce ownership/referential correctness in services/repositories unless an explicit SQLite-safe rebuild migration is written.
- **RIC/Structured Reference Migration Design (2026-06-01):** The legacy free-text catalog field is `Coin.RarityRating` (`json:"rarityRating"`, DB `rarity_rating`); `ReferenceText`/`ReferenceURL` are link fallback fields. `CoinReference` stores `coin_id`, `catalog`, `volume`, `number`, `certainty`, and `uri`, with unique `(coin_id,catalog,volume,number)` and validation against `CatalogRegistry` (`RIC`, `RPC`, and `SNG` require volume). Recommended backfill: idempotent guarded startup migration that parses legacy values such as `RIC II 207` into validated references, skips/logs values missing required volume such as bare `RIC 207`, and keeps legacy columns until a separate SQLite-safe drop decision.

## 2026-06-01 — Storage Location migration no-data-loss verification

Verified Brian's no-data-loss requirement by backing up `src/api/ancientcoins.db` to a project-local disposable copy, running the real `config.Load()` + `database.Connect()` AutoMigrate path against only that copy via `DB_PATH`, then diffing per-table row counts before/after. Result: PASS; all existing table counts were unchanged, `storage_locations` was created empty, `coins.storage_location_id` was added nullable, and the verification copy/harness were deleted.

## 2026-06-01 — Legacy Rarity/RIC to Catalog References Migration (Design Proposal)

Conducted a design review for migrating legacy free-text `Coin.RarityRating` values into structured `CoinReference` records. No code was implemented; proposal awaits Brian approval on 3 open questions.

**Key findings:**
- Legacy field: `Coin.RarityRating` (string, DB column `rarity_rating`); documented as "RIC 207", "Sear 1625" examples
- Modern storage: `CoinReference` table with unique constraint on `(coin_id, catalog, volume, number)` and validation via `CatalogRegistry`
- Catalog registry rules: RIC/RPC/SNG require volume; SEAR/CRAWFORD/etc. do not
- Current dev state: 0 coins, 0 coin references

**Proposed approach:**
- Idempotent guarded startup backfill in `database.Connect()` after `AutoMigrate` and `seedCatalogRegistry`
- Parser normalizes catalog names and extracts volume per registry rules
- Skips ambiguous values (e.g., bare `RIC 207` without volume) instead of inventing structure
- Uses `certainty:"legacy-import"` for all backfilled references
- Logs every skip with reason; fails only on DB errors
- Preserves legacy columns (`rarity_rating`, `reference_text`, `reference_url`) for non-destructive migration

**Open questions (awaiting Brian approval):**
1. Bare `RIC 207` skip policy vs. manual-review pathway?
2. Multi-reference parsing support (`RIC II 207; Cohen 15`) and unsupported-catalog reporting?
3. Certainty value: `legacy-import` or existing UI values (`probable`/`high`)?

**Related decisions:** 
- Aurelia removed the free-text RIC UI surface (decision: "Remove Free-Text Rarity/RIC UI")
- Non-destructive requirement aligned with SQLite foreign-key migration gotchas documented earlier

## 2026-06-01 — Legacy Rarity/RIC Reference Migration Implementation

Implemented the approved one-time backfill migration that parses legacy `Coin.RarityRating` text into structured `CoinReference` records. Migration runs at startup after AutoMigrate and seedCatalogRegistry, guarded by AppSetting marker `LegacyRarityRatingReferenceBackfillV1` for idempotency.

**Key files:**
- `src/api/database/database.go` — added `backfillLegacyRarityRatingReferences()`, `parseLegacyReference()`, helper functions
- `src/api/database/reference_migration_test.go` — comprehensive parser tests, idempotency tests, sentinel volume tests

**Parser rules implemented:**
- Parses FIRST reference only from multi-reference strings (semicolon-delimited)
- Catalog normalization: RIC/RPC/SNG/CRAWFORD/CNI/KM/Y/CRAIG/REDBOOK exact; Sear/SRCV→SEAR; Spink→SPINK; Duplessy→DUPLESSY
- Volume extraction for volume-required catalogs (RIC/RPC/SNG): Roman numerals (I, II, VII, etc.), numeric volumes (1-3 digits), or alphabetic tokens (e.g., "Cop" for SNG Copenhagen)
- Volume=0 sentinel + journal note when volume is missing/unparseable on volume-required catalog
- Certainty: "legacy-import" on all backfilled references
- Existing structured references win (no overwrite)
- Non-destructive: preserves `rarity_rating`, `reference_text`, `reference_url` columns

**Approved rules from Brian:**
1. Missing/unparseable volume on volume-required catalog → `volume="0"` + CoinJournal entry for manual review
2. Multiple references in one field → parse FIRST only, ignore rest
3. Certainty value → `"legacy-import"`

**Validation:**
- All tests pass: `go build ./...`, `go vet ./...`, `go test -v ./...`
- Parser handles: "RIC II 207", "RIC VII 162", "Sear 1625", "SNG Cop 123", bare "RIC 207" (→ volume 0 + journal), multi-refs, unrecognized catalogs, empty/whitespace
- Idempotency verified: re-running backfill is a no-op once marker is set
- Existing references preserved: backfill skips coins that already have matching structured references

## 2026-06-01 — Legacy Reference Migration Refactor: Startup → User-Triggered Endpoint

Refactored the legacy reference migration from an auto-startup backfill to a user-triggered, user-scoped endpoint per Principle I layered architecture requirements.

**Changes:**
- **Removed** startup wiring from `database/database.go` (lines 40-42): deleted `backfillLegacyRarityRatingReferences()` call and all parser logic (previously ~lines 86-343)
- **Created** `services/reference_migration_service.go`: migration logic moved to service layer with `MigrateLegacyReferences(userID)` method
- **Created** `services/reference_migration_service_test.go`: relocated 19 parser tests + 4 integration tests (user-scoped, idempotency, existing-ref, volume-0 sentinel)
- **Extended** `handlers/coin_references.go`: added `MigrateLegacy()` handler method with Swagger annotation
- **Wired** new route in `main.go`: `POST /references/migrate-legacy` under protected group
- **Added** `handlers/swagger_types.go`: `MigrationResultDTO` type for OpenAPI

**Endpoint Contract (FIXED, Aurelia building against this):**
- Method/path: `POST /references/migrate-legacy`
- Auth: JWT required, operates on authenticated user's coins only
- Request body: none
- Response 200: `{ "succeeded": 12, "skipped": 45, "failed": 3 }` (lowercase field names, integers)

**Behavior:**
- User-scoped: migrates ONLY the requesting user's coins (like Tags/Storage Locations)
- Journals every coin: success → reference created; skip → reason (already exists, no text, etc.); fail → error message
- Re-run safe: coins with existing matching references are skipped with journal note
- Non-destructive: never drops or nulls legacy columns, additive inserts only

**Parser rules unchanged:**
- Parse FIRST reference only; volume=0 sentinel + manual-review journal when volume missing on volume-required catalog
- Catalog aliases: Sear/SRCV→SEAR, Spink→SPINK, Duplessy→DUPLESSY
- Certainty: `"legacy-import"`

**Architecture compliance:**
- Migration logic now in service layer (not database package)
- Handlers thin, constructor injection pattern
- All tests pass including `TestNoDirectDatabaseImports`

## 2026-06-01 — User-Triggered Legacy RIC→Reference Migration Endpoint (SHIPPED)

Refactored the legacy `Coin.RarityRating` → `CoinReference` migration from auto-startup backfill to user-triggered endpoint per Brian's request. Migration is now user-scoped (protected group) and journals every coin's outcome (succeeded/skipped/failed).

**Implementation:**
- `services/reference_migration_service.go` — refactored migration logic with `MigrateLegacyReferences(userID uint)` method
- `services/reference_migration_service_test.go` — 19 parser tests + 4 integration tests (user-scoped, idempotency, existing-ref, volume-0 sentinel)
- `handlers/coin_references.go` — new `MigrateLegacy()` handler
- `main.go:225` — endpoint wired as `POST /references/migrate-legacy` in protected group
- Removed startup wiring from `database/database.go` (lines 40-42)

**Endpoint Contract:**
- Method: `POST /references/migrate-legacy`
- Auth: JWT required (protected group)
- Scope: Authenticated user's coins only
- Response: `{ "succeeded": int, "skipped": int, "failed": int }`

**Per-Coin Journaling:**
Every coin processed records its outcome in CoinJournal:
- Success: "Legacy reference migrated: RIC II 207 → catalog RIC, vol II, no. 207"
- Skip: "Already has matching reference: ..." or "No parseable reference in rarity_rating field"
- Fail: "Failed to parse legacy reference: ..." or "Failed to create reference: ..."
- Manual review: Extra journal note for volume=0 sentinel

**Verification:** go build/vet/test all pass; commit 978eb23.

**Related:** Aurelia building parallel UI in Settings → Data with result counts and error handling.

