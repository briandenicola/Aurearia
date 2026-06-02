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
- **2026-06-01:** Frontend navigation convention established: parent hub pages (CoinDetailPage) with multiple child subpages use absolute `router.push('/')` to grandparent list view, not relative `router.back()`. Single-child forms (EditCoinPage) use `router.back()` after save. Subpages return to hub via `router.push('/coin/:id')` to allow continued exploration. This convention prevents history-stack pollution when users navigate through sibling subpages and avoids trapping users in subpage cycles (Aurelia fixed this in commit 6747a6d). Relevant for any backend feature that might create multi-child detail structures.

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

## 2026-06-01 — Per-Coin Metadata Health Endpoint (BUG FIX)

Fixed the Metadata Health subpage always showing "No health data available for this coin yet." by adding a direct per-coin health endpoint. The existing paginated list endpoint caused the frontend to fetch limit=1000 and filter client-side, breaking when the target coin wasn't on that page.

**Implementation:**
- `repository/health_repository.go` — added `GetSingleEligibleCoin(coinID, userID)` using `ActiveCollection` scope
- `services/health_service.go` — added `GetCoinHealth(coinID, userID)` that reuses existing scoring logic: `scoreCoinMetadata`, `scoreCoinImages`, `scoreCoinValuationFreshness`, `scoreCoinAICoverage`, `computeWeightedScore`, `generateCoinChecklist`, `extractQuickActions`
- `handlers/health.go` — added `GetCoinHealth(c *gin.Context)` handler with Swagger annotation
- `main.go` — wired `protected.GET("/coins/:id/health", healthHandler.GetCoinHealth)`
- Frontend: `src/web/src/api/client.ts` added `getCoinHealth(coinId)`, `CoinDetailHealthPage.vue` now calls it directly instead of list+filter

**Key Learning:** Health data is COMPUTED from coin fields (not stored), so every existing active collection coin always has a score/grade/checklist. The per-coin endpoint validates user ownership (404 if not found or not user's coin) and returns the same `CoinHealthItem` shape the list uses.

**Verification:** go build/vet/test pass, npm run build pass, commit 5bd36e9.


## 2026-06-01 — Catalog Registry Backend CRUD + CoinReference.Certainty → InvoiceNumber

Completed backend deployment of catalog registry feature in parallel with Aurelia's frontend work.

**Changes:**
- Renamed `CoinReference.certainty` → `invoiceNumber` (repurposed unused field from AI confidence scoring). Migration idempotent via PRAGMA column check.
- Removed AI certainty/confidence concept from Go proxy structs (`CandidateReferenceProxy`, `CandidateReferenceDTORef`) and Python agent models (`CandidateReference`). Noted that `ValueEstimate.confidence` and `AvailabilityVerdict.confidence` remain (different contexts).
- Implemented full CRUD for `CatalogRegistry`: repository (`Create`, `Update`, `Delete`, `FindByID`, `CountReferencesUsing`), service with validation (era ∈ {ancient, medieval, modern}, code required, duplicate/in-use checks), handler, routes (`GET /catalogs`, admin `POST/PUT/DELETE /admin/catalogs/:id`).
- Seeded PRICE, BM, VENÈRA catalogs (diacritic preserved in uppercase).

**Sentinel errors:** `ErrCatalogNotFound`, `ErrCatalogDuplicate`, `ErrCatalogInUse`, `ErrCatalogInvalidEra`, `ErrCatalogCodeRequired`, `ErrCatalogNameRequired`.

**Verification:** go build/vet/test all pass (architecture_test.go ✅), ruff + 60/60 pytest ✅. Commit d0d3db1.

**Frontend integration:** Aurelia built dropdown UI sourced from `GET /catalogs` with legacy fallback, new `AdminCatalogsSection.vue` CRUD interface, and help text updates. Commit 0de29af.

**OpenAPI:** Coordinator regenerated for GET/admin /catalogs + invoiceNumber. Commit 100087f. All three commits pushed to origin/main.

## Learnings

- **Bulk Assign Storage Location (2026-06-01):** Added `"assign-location"` action to the existing bulk coin operations (`POST /coins/bulk`). Request body now accepts an optional `storageLocationId` field (nullable uint). When action is `"assign-location"`, the handler validates ownership of the location (if non-null/non-zero) via `StorageLocationRepository.ExistsByID`, then calls the new `CoinRepository.BulkAssignLocation(coinIDs, storageLocationID, userID)` method. The repository method uses GORM `.Update("storage_location_id", storageLocationID)` to correctly handle nil pointer writes as SQL NULL (GORM's `.Updates()` with a map can skip nil/zero values). A nil or omitted `storageLocationId` clears the location on all selected coins. Response follows the existing bulk action pattern: `{ "message": "Storage location assigned", "affected": <int> }`. Wiring: `BulkHandler` constructor now takes `StorageLocationRepository` as third parameter, wired in `main.go` line 256.

## 2026-06-02 — Metadata Health Valuation Freshness Fix

Fixed health scoring to measure valuation freshness from when the value was last updated, not from purchase date. Before the fix, a coin bought years ago but valued today would still show as stale.

**Changes:**
- **Model:** Added nullable `Coin.CurrentValueUpdatedAt *time.Time` field (DB: `current_value_updated_at`)
- **Migration:** Safe additive nullable column in `database.Connect()` AutoMigrate; no FK constraints needed
- **Repository:** Updated all `EligibleCoinRow` SELECT queries (`ListEligibleCoins`, `ListEligibleCoinsPaged`, `ListAllEligibleCoins`, `GetSingleEligibleCoin`) to include `current_value_updated_at`
- **Health Scoring:** `scoreCoinValuationFreshness` and `generateCoinChecklist` now measure age from `CurrentValueUpdatedAt` when present, fallback to `PurchaseDate` for legacy coins (non-regressive)
- **Valuation Writes:** 
  - `ValuationService.updateCoinValuation` sets both `current_value` and `current_value_updated_at` (scheduled valuations)
  - `CoinService.UpdateCoin` sets `current_value_updated_at` when `CurrentValue` changes manually (UI edits)
- **Tests:** Added comprehensive `TestScoreCoinValuationFreshness_WithCurrentValueUpdatedAt` covering fresh/stale/legacy fallback paths

**AI Coverage Investigation:**
Confirmed obverse/reverse analysis is correctly persisted to `coins.obverse_analysis` / `coins.reverse_analysis` columns (see `AnalysisHandler.Analyze` lines 177-181). Health scoring reads the correct source. No bug found; if Brian's coin shows "ai.coverage" warning but has analysis present, it's likely the per-side analysis (obverse OR reverse missing), which is a Low-severity item and working as designed.

**Learnings:**
- Metadata health scoring architecture: `HealthService` scoring functions (`scoreCoinMetadata`, `scoreCoinValuationFreshness`, etc.) + `generateCoinChecklist` read from `EligibleCoinRow`, which is populated by `HealthRepository` SELECT queries that read `coins.*` columns directly (no joins to other tables for analysis text).
- The `current_value_updated_at` field is now the source of truth for valuation freshness; fallback to `PurchaseDate` preserves legacy behavior for coins valued before this field existed.
- All CurrentValue writes now set the timestamp atomically.

## 2026-06-02 — Metadata Health Valuation Freshness Fix (Complete + Shipped)

Fixed health scoring to measure valuation freshness from when value was last updated, not purchase date.

**Implementation:**
- Added nullable `Coin.CurrentValueUpdatedAt *time.Time` field (DB: `current_value_updated_at`)
- Safe additive nullable column via AutoMigrate
- All health repository SELECT queries updated to fetch new field
- Scoring logic (`scoreCoinValuationFreshness`, `generateCoinChecklist`) measures from `CurrentValueUpdatedAt` with fallback to `PurchaseDate` for legacy coins (non-regressive)
- Valuation writes set timestamp atomically: scheduled (ValuationService) and manual (CoinService)
- Added comprehensive tests: `TestScoreCoinValuationFreshness_WithCurrentValueUpdatedAt` with 9 cases

**AI Coverage Investigation:** No bug found; analysis correctly persists to obverse/reverse columns. If coin shows ai.coverage warning despite analysis, it's missing one side (working as designed).

**Verification:** go build/vet/test all pass ✅

**Commit:** 7357599

**Cross-agent note:** Aurelia shipped camera modal extraction (`CameraCaptureModal.vue`) for Coin Details; now reusable for future features needing in-app capture with circular focus + cover-crop.

## 2026-06-02 — Metadata Health AI Coverage Fix (Corrected Logic)

Fixed AI coverage scoring and checklist generation to properly credit combined `ai_analysis` as covering both coin faces, eliminating false "Needs Attention" warnings on fully-analyzed coins.

**The Bug (Brian's pushback):**
Previous implementation was "unduly harsh and not taking all data into account." It treated the three analysis fields (`ai_analysis`, `obverse_analysis`, `reverse_analysis`) as independent fields to count: 1/3 = 33%, 2/3 = 66%, 3/3 = 100%. But combined analysis (`ai_analysis`) describes BOTH faces — so a coin with only combined analysis should score 100%, not 33%. Similarly, the checklist was emitting `ai.coverage` ("Complete AI analysis (obverse + reverse)") whenever `ObverseAnalysis == "" || ReverseAnalysis == ""`, completely ignoring whether a combined `AIAnalysis` existed. This was the "not taking all data into account" issue.

**The Fix:**
Redesigned both `scoreCoinAICoverage` and `generateCoinChecklist` to reflect the semantic model: "does this coin have meaningful AI analysis covering both faces?"

**New Scoring Logic (`scoreCoinAICoverage`):**
```go
hasCombined := coin.AIAnalysis != ""
hasObverse := coin.ObverseAnalysis != ""
hasReverse := coin.ReverseAnalysis != ""

// Combined analysis covers both faces
obverseCovered := hasObverse || hasCombined
reverseCovered := hasReverse || hasCombined

if !obverseCovered && !reverseCovered {
    return 0  // No analysis at all
} else if obverseCovered && reverseCovered {
    return 100  // Full coverage (both sides OR combined)
}
return 50  // Partial: one side only, no combined
```

**New Checklist Logic:**
- Emit `ai.analysis` (Medium severity) ONLY when there is NO analysis of any kind (no combined, no obverse, no reverse)
- Emit `ai.coverage` (Low severity) ONLY when there is partial per-side analysis with a genuine gap AND no combined analysis to fill it: `!hasCombined && (hasObverse != hasReverse)` (XOR pattern)
- If a combined `ai_analysis` exists, do NOT emit `ai.coverage` at all — coverage is satisfied

**Net Effect:**
- Coin with combined `ai_analysis` only → score 100, no checklist items ✅
- Coin with both `obverse_analysis` + `reverse_analysis` → score 100, no checklist items ✅
- Coin with only `obverse_analysis`, no reverse, no combined → score 50, emits `ai.coverage` ✅
- Coin with nothing → score 0, emits `ai.analysis` ✅
- Coin with combined + one per-side → score 100, no checklist items ✅

**Tests Added:**
Five new test functions in `health_service_test.go`:
1. `TestScoreCoinAICoverage_CombinedAnalysisOnly` — combined only → 100, no items
2. `TestScoreCoinAICoverage_BothPerSideOnly` — obverse+reverse → 100, no items
3. `TestScoreCoinAICoverage_OnlyObverseNoReverse` — obverse only → 50, ai.coverage item
4. `TestScoreCoinAICoverage_NoAnalysisAtAll` — nothing → 0, ai.analysis item
5. `TestScoreCoinAICoverage_CombinedPlusOneSide` — combined+obverse → 100, no items

**Validation:** go build/vet/test all pass ✅

**Learnings:**
- The corrected AI-coverage model: combined `ai_analysis` counts as covering both faces; `ai.coverage` no longer fires when a combined analysis exists.
- The two touch points for AI health logic are `scoreCoinAICoverage` (scoring) and the AI checklist block in `generateCoinChecklist` (around line 562-578).
- Always read the spec correctly: "harsh" + "not taking all data into account" means the existing logic is counting fields independently instead of treating combined analysis as satisfying both-face coverage semantically.

## 2026-06-02 — AI Coverage Fix CORRECTION (Obverse + Reverse only)

Superseded the "combined counts as both faces" approach above per Brian's explicit
clarification: "that's all I care about for the AI analysis scoring - obverse and reverse".

**Final model:** AI coverage is measured ONLY by per-side `obverse_analysis` +
`reverse_analysis`. Legacy combined `ai_analysis` is NOT counted (UI only offers
per-side Analyze buttons; combined is legacy). Score: both=100, one=50, none=0.

**Checklist now explains the gap:** `ai.coverage` label dynamically names the missing
side, e.g. "Run AI analysis on the reverse (obverse already done)". Frontend
`CoinHealthChecklist.vue` was rendering the raw `item.key` ("ai.coverage") — switched
to render `item.label` so every Needs-Attention row explains what's missing.

**Learning:** Don't over-generalize a "too harsh" complaint into crediting a legacy
field. Confirm the intended scoring axis with the actual UI workflow — here the per-side
buttons confirmed obverse/reverse is the real coverage signal.
