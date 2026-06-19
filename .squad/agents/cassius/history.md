# Project Context

- **Owner:** Brian
- **Project:** Ancient Coins backend — Go 1.26 / Gin / GORM / SQLite
- **Architecture:** Layered Handler → Service → Repository → Database with constructor injection and architecture tests.

## Core Context

**Durable backend rules:**
- Thin handlers, service-owned business logic, repository-owned GORM queries, scopes for ownership/public filters, sentinel service errors, Swagger annotations, DI wiring in `main.go`
- Scheduler/run-log pattern: configurable settings, manual trigger, run history table, production diagnostics (applied to valuation, wishlist/availability, auction-ending)
- Time-sensitive auction queries: rolling `(now, now+24h]` windows, explicit NULL guards, case-insensitive status comparison, real-data diagnostics
- Security/backend patterns: validate ownership before heavy decode ops; circle image clipping in stdlib-only `src/api/capture/` gated to obverse/reverse uploads when `circleClip=true`
- SQLite FK migration gotcha: nullable lookup FKs added post-launch should use `constraint:-` (no physical constraints) to avoid destructive table rebuilds; enforce ownership/referential correctness in services/repositories instead
- RIC/Structured Reference migration: legacy free-text `Coin.RarityRating` parses idempotently into structured `CoinReference` at user request (not startup); skips ambiguous values; preserves legacy columns for non-destructive migration

**Recent batch outcomes (2026-06-01 — 2026-06-03):**
- Valuation freshness fix: added `Coin.CurrentValueUpdatedAt` field to track when valuation was last updated; health scoring now measures staleness from valuation update time (fallback to PurchaseDate for legacy coins)
- External tool server stack: API key capabilities, enablement toggle, capability middleware, per-key rate limit, `/api/v1/tools` route group, handlers, OpenAPI discovery, external commit journal metadata
- AI coverage health fix (final model): **obverse + reverse only** (legacy combined `ai_analysis` not counted); both sides → 100, one → 50, none → 0; checklist items render human-readable labels naming missing side
- Collection chat multi-container callback issue: `AGENT_INTERNAL_CALLBACK_URL` must point from agent container to API service (e.g. `http://coins:8080`), not localhost; startup warning added for release+localhost
- v1→v2 migration audit: only additive schema changes; AutoMigrate/backfill safe and rollback-safe
- Frontend navigation convention documented: parent detail pages use absolute `router.push('/')` to grandparent list, single-child forms use `router.back()` after save (prevents history pollution with subpage cycles)

**Architecture compliance:** All recent work follows Principle I (Layered Architecture), Principle VII (Schema-Driven Contracts), Principle XI (Security Hardening), Principle XII (Auth & Token Policy)

## Recent Updates

- **2026-06-19:** Scope assessment for #321 (Lock Python dependencies) and #319 (Non-root Docker users):
  - #321 is ready: uv.lock strategy, CI/Docker changes isolated, low risk
  - #319 is ready: standard USER/chown pattern, no privileged ops, write-path validation straightforward
  - Both independent; recommend #321→#319 sequence if merged in single PR to avoid line-number conflicts on `src/agent/Dockerfile`
  - Neither requires splitting; both are coherent single-issue units
  - See `.squad/decisions/inbox/cassius-scope-321-319.md` for detailed analysis

- **2026-06-01:** #217 shared collection tool layer (internal token service, six internal endpoints, keyword gate removed), #217 Python ReAct agent completed end-to-end, #218 external tool server stack
- **2026-06-01:** v1→v2 migration audit, Frontend navigation convention, Storage Location API pattern (per-user lookup table, nullable Coin.StorageLocationID FK, 409 conflict guard), Legacy RIC→CoinReference migration design + implementation + startup→endpoint refactor
- **2026-06-09:** F013 Phase 3 golden fixtures complete (T014). Implemented Go fixture builders covering all 9 F013 golden coin names/traits with defensive cloning and optional deterministic DB persistence. Approved by Maximus Lead Review. Go build/test/vet all pass. Orchestration log: `.squad/orchestration-log/2026-06-09T13-09-16-cassius.md`
- **2026-06-02:** Valuation freshness fix (CurrentValueUpdatedAt), Metadata health AI coverage fix (combined→obverse+reverse), AI coverage health scoring correction (per-side model finalized), checklist labels for missing side
- **2026-06-08:** CodeQL request-forgery suppression comments added to `ProxyImage` and `ScrapeImage` handlers; SSRF protection layer remains unchanged
- **2026-06-18:** Mint Map Backend Analysis — Pagination Limit Investigation
   - Investigated Aurelia's report that Mint Map displayed only 50 coins despite larger active collections
   - Confirmed that `GET /coins` endpoint is correctly implemented as paginated collection API with default `limit=50`, validated max `limit=100`, and `total` field for paging
   - Verified no backend total cap exists; pagination behavior is safe and intentional
   - Decision: no backend changes required; frontend should fetch all active collection pages using explicit pagination loop with `wishlist=false`, `sold=false`, `page`, `limit=100`
   - Architecture compliant: preserves safe paginated API contract (Principle I), respects explicit response contract (Principle III), uses proportional frontend-only fix (Principle IV)
   - Orchestration log: `.squad/orchestration-log/2026-06-18T21-14-02Z-cassius.md`



## Learnings

- **2026-06-19 — PR #320 Go Toolchain Lockout Revision:** As non-original reviser after Brutus's §18.2 BLOCK, corrected `src/api/go.mod` to `go 1.26.4` so setup-go, Docker/docs/workflows, and the module pin align. Verified workflows use `go-version-file: src/api/go.mod`, `govulncheck@v1.4.0`, and `swag@v1.16.6`; local `go env GOVERSION` resolved to `go1.26.4` and pinned `govulncheck ./...` found no vulnerabilities.

- **2026-06-19 — Agent Service Boundary Hardening (#309/#310):** Python agent direct surface is now internal-only by default: compose exposes port 8081 only to the Docker network, non-health endpoints require `AGENT_INTERNAL_SERVICE_TOKEN`, and Go `AgentProxy` sends the credential on API, SSE, logs, and log-level calls. Python outbound URLs for Ollama, SearXNG, and collection tools are restricted to exact `AGENT_TRUSTED_OUTBOUND_ORIGINS`; localhost/private/link-local/metadata addresses require explicit `AGENT_ALLOW_LOCAL_OUTBOUND=true` for local dev.

- **Coin of the Day Pushover Link Configuration (2026-06-10):** Initial implementation used relative `/coin/{coinID}` links in Pushover payloads; Brutus blocked (STRICT LOCKOUT, §18.2) because Pushover opens notifications outside app context where relative URLs are not usable. Aurelia revised by adding `PublicAppURL` admin setting: links now build as absolute `http(s)://host/coin/{coinID}` by trimming trailing slashes, and omit the link entirely when setting is blank/invalid. Brutus reviewed revised implementation and CLEARED BLOCK. Decision merged to `decisions.md`. See orchestration logs: `.squad/orchestration-log/2026-06-10T20-31-52Z-{cassius,aurelia,brutus}.md`

### 2026-06-10 — Collection Count Invariant

**Files:**
- `src/api/repository/scopes.go` — `ActiveCollection` scope defines `is_wishlist=false AND is_sold=false`
- `src/api/repository/coin_repository.go` — `GetStats` uses `ActiveCollection`, `List` applies wishlist/sold filters
- `src/api/services/collection_tools_service.go` — `CollectionSummary` calls `GetStats`, passes through `TotalCoins`
- `src/api/handlers/coins.go` — List handler documentation clarifies `total` semantics
- `src/api/handlers/coin_handler_test.go` — `TestCoinHandler_ActiveCollectionCountInvariant` locks the predicate contract

**Pattern:**
The canonical "active collection" count is `owned AND NOT wishlist AND NOT sold`. All three query paths already use identical predicates:
1. `/coins?wishlist=false&sold=false` → `List` applies both filters, counts after
2. `/stats` → `GetStats` uses `ActiveCollection` scope
3. `collection_summary` tool → calls `GetStats`, returns `stats.TotalCoins`

**Key insight:** No predicate fix was needed — they already match. Added a regression test (`TestCoinHandler_ActiveCollectionCountInvariant`) to lock the invariant: all three paths must return the same count for a mixed fixture (active + wishlist + sold). Documented `/coins` total semantics: "reflects the applied filter, not 'collection size.'"

### 2026-06-06 — Documentation Feature Showcase (Issue #241)

**Feature Discovery & Documentation Refactor:**
Created comprehensive feature documentation structure to showcase the app's full capability set. Documentation reorganization moved scattered feature details into discoverable, indexed docs with per-feature deep dives.

**Key Files Changed:**
- `README.md` — Added Feature Highlights section with 8 major feature categories, Feature Matrix with capability grid, and What's New timeline
- `docs/features/INDEX.md` — Created master index with links to 30+ features organized by category
- `docs/features/*.md` — Created 25+ individual feature documents:
  - Deep-dive docs: collection-management.md, coin-details.md, coin-sets.md, wish-list.md, ai-analysis.md, ai-search-agent.md, statistics.md (500-8200 words each)
  - Shorthand guides: auction-tracking.md, sold-coins.md, social-features.md, pwa-features.md, coin-of-the-day.md, custom-tags.md, user-profiles.md, admin-settings.md, collection-showcase.md, numista-integration.md, and 12 others (~1500-2000 words each)
- `docs/features.md` — Added redirect header and quick reference table pointing to new docs/features/ structure

**Feature Map (All 30+ features now documented):**
- Core: Collection Management, Coin Details, Coin of the Day
- Discovery: Wish List, Auction Tracking, Sold Coins
- AI: Analysis, Search Agent, Grading, Price Trends, Gap Analysis, Photography Guide, Similar Lots
- Organization: Coin Sets, Custom Tags, Statistics, Collection Showcase
- Social: Social Features, User Profiles
- Admin: Admin Settings, Authentication, External Tool Server
- Mobile: PWA Features, Camera Capture
- Advanced: Image Operations, PDF Export, Bulk Operations, Notifications, Numista, Auction Calendar, Import/Export

**Accuracy Notes:**
- No cloud features fabricated (Auth0, CosmosDB, Azure, Terraform/K8s not documented)
- All docs describe current self-hosted architecture (Go/Vue/Python/SQLite/Docker)
- Feature matrix shows actual capabilities with clear symbols (✅, ❌, —)
- What's New section includes v2.0 in-development features (Coin Sets, Health Scorecard)

**Documentation Quality:**
- Every feature includes: Overview, Key Features, How to Use, Configuration, API Endpoints, Related Features
- Cross-linking enables readers to traverse related features
- Emoji icons improve scannability
- Clear use cases and workflows for each feature

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

---

## 2026-06-02 12:31:33Z: AI Coverage Health Scoring Fix — Decision Merged

**Status:** Merged to decisions.md

AI-coverage model finalized: **obverse + reverse only** (legacy combined field not counted). Both sides → 100, one → 50, none → 0. Checklist items now render human-readable labels naming the missing side.

**Files:** `services/health_service.go`, `health_service_test.go`. Frontend coordinator updated `CoinHealthChecklist.vue` to render `item.label`.

**Cross-agent:** Aurelia (frontend) will benefit from camera permissions pre-check; no backend impact.

**Commit:** fcfe401

## 2026-06-06 — Documentation Feature Showcase (Issue #241)

**Status:** Complete

Reorganized all feature documentation from a single monolithic 358-line `docs/features.md` into a hierarchical, discoverable structure with 30+ individual feature docs organized by category.

**Key Changes:**
- Created `docs/features/INDEX.md` — Master index with 30+ features organized by 8 categories
- Created 7 deep-dive feature docs (500–8,200 words each): collection-management, coin-details, coin-sets, wish-list, ai-analysis, ai-search-agent, statistics
- Created 18+ shorthand feature guides (1,500–2,000 words each) covering remaining features
- Enhanced `README.md` with Feature Highlights (8 categories), Feature Matrix (7x10 capability grid), What's New timeline
- Preserved backward compatibility: `docs/features.md` includes redirect header + quick reference table

**Benefits:**
- **Discoverability:** 30+ entry points via search/GitHub vs. 1 monolithic 358-line document
- **Maintenance:** Individual docs updated independently; no merge conflicts on massive files
- **SEO:** Multiple pages increase search surface area
- **User Experience:** Consistent structure, emoji icons, clear cross-linking for feature exploration

**No Cloud Features Fabricated:** All docs accurately describe self-hosted architecture (Go/Vue/Python/SQLite/Docker). No Auth0, CosmosDB, Azure, or Terraform features invented.

**Verification:** Markdown link validation ✅, suspicious-claim scans ✅, git diff --check ✅

**Orchestration Log:** 20260606T194119Z-cassius-docs.md
**Session Log:** 20260606T194119Z-issue-241-docs-feature-showcase.md
**Decision Merged:** decisions.md (Decision: Documentation Feature Showcase — Issue #241)

## 2026-06-07 — User-Defined Coin Category and Era Options

**Status:** Complete

Added backend settings support for user-defined coin category and era option lists, replacing hardcoded constants with customizable values.

**Implementation:**
- **New Settings Keys:**
  - `SettingCoinCategories` (`"CoinCategories"`) — default: `"Roman\nGreek\nByzantine\nModern\nOther"`
  - `SettingCoinEras` (`"CoinEras"`) — default: `"ancient\nmedieval\nmodern"`
- **Format:** Newline-delimited strings (split on `\n` to parse)
- **Files Changed:**
  - `services/settings_service.go` — added constants and defaults
  - `services/settings_service_test.go` — added 6 tests covering defaults, customization, GetAllSettings inclusion
- **Automatic Exposure:** Existing `/admin/settings` and `/admin/settings/defaults` endpoints now return these keys

**Testing:**
- `TestGetSetting_CoinCategories_ReturnsDefault` ✅
- `TestGetSetting_CoinEras_ReturnsDefault` ✅
- `TestSetSetting_CoinCategories_AllowsCustomization` ✅
- `TestSetSetting_CoinEras_AllowsCustomization` ✅
- `TestGetAllSettings_IncludesCoinCategoriesAndEras` ✅
- All existing tests still pass ✅

**Frontend Coordination Notes for Aurelia:**
- Parse `settings.CoinCategories` / `settings.CoinEras` by splitting on `\n`
- Admin UI should allow multi-line text editing
- "Unspecified" era option should remain UI-only (not stored in setting)
- Empty values fall back to defaults automatically

**Backward Compatibility:** Defaults match existing hardcoded values in `models/coin.go`; existing coin data unaffected.

**Decision Document:** `.squad/decisions/inbox/cassius-era-category-options.md`

**Learnings:**
- Newline-delimited format is human-readable and trivial to parse, consistent with potential multi-line prompt settings
- Settings service pattern (key-value with fallback) extends cleanly to option lists
- Frontend will need basic `split('\n')` parsing; consider JSON format if richer metadata (icons, colors) needed in future

- **2026-06-07:** Era/Category Backend + Coin Lookup Infrastructure Inventory
  - **Era/Category Settings:** Added `CoinCategories` and `CoinEras` settings with newline-delimited defaults matching hardcoded values; 6 passing tests; automatic exposure via `/admin/settings` endpoints.
  - **Coin Lookup Architecture Inventory:** Completed infrastructure audit — 90%+ of lookup MVP already exists (AI Intake Draft #216, Numista proxy, image analysis, agent proxy, catalog references). Recommended path: extend intake draft with Numista enrichment (Go-only service, 2-3 days). NGC deferred to post-MVP. No new Python agent team needed for MVP.
  - **Numista Enrichment Service (Proposed):** New service layer to extract keywords from draft fields, query Numista, map results to DTO. Low-effort orchestration on top of existing infrastructure.

## 2026-06-08 — CodeQL Request Forgery Alerts (Suppression Fix)

**Status:** Complete

Addressed two CodeQL `go/request-forgery` alerts on `client.Do(req)` calls in `ProxyImage` and `ScrapeImage` handlers. CodeQL's static analysis flagged user-provided URLs as untrusted even though robust SSRF protections were already in place.

**Implementation:**
- **Changed Files:** `src/api/handlers/images.go`
- **Changes:** Updated inline suppression comments from `codeql[...]` to `lgtm [...]` format on lines 373 and 467
- **No Functional Changes:** All SSRF protections remain unchanged and comprehensive

**Existing SSRF Protection Stack (Already Implemented):**

**1. URL Validation Layer** (`validateOutboundURL` in `outbound_http.go`):
- ✅ Only allows `http://` and `https://` schemes
- ✅ Rejects credentials in URL (`user:pass@`)
- ✅ Blocks `localhost` hostname
- ✅ Blocks direct IP access to private/loopback/link-local ranges

**2. HTTP Client Layer** (`newRestrictedHTTPClient`):
- ✅ Disabled proxy support (prevents proxy-based SSRF)
- ✅ Custom `DialContext` resolves DNS on **every connection attempt** (prevents DNS caching-based rebinding)
- ✅ Post-resolution IP blocking: rejects private/loopback/link-local IPs after DNS lookup
- ✅ Redirect policy: validates **every redirect target** through same validation rules
- ✅ 10-redirect maximum
- ✅ 30-second timeout

**3. Blocked IP Ranges** (comprehensive CIDR list):
- Private IPv4: `10.0.0.0/8`, `172.16.0.0/12`, `192.168.0.0/16`
- Loopback: `127.0.0.0/8`, `::1/128`
- Link-local: `169.254.0.0/16` (AWS metadata), `fe80::/10`
- Special use: `0.0.0.0/8`, `100.64.0.0/10`, `198.18.0.0/15`, carrier-grade NAT, multicast, reserved ranges

**Testing Coverage** (`outbound_http_test.go`):
- ✅ URL validation (public URLs pass, localhost/loopback/link-local/credentials blocked)
- ✅ DNS resolution blocking private IPs (fake resolver tests)
- ✅ Per-connect DNS resolution (no caching)
- ✅ Redirect policy enforcement
- ✅ Integration tests for `ProxyImage` and `ScrapeImage` blocking connect-time private resolution

**Validation:**
- All tests pass: `go test -v ./handlers -run "TestProxyImage|TestScrapeImage|TestValidateOutboundURL|TestRestrictedDialContext"` ✅
- Architecture tests pass: `go test -v -run TestNoDirectDatabase .` ✅

**Why Inline Suppression is Appropriate:**
- CodeQL's taint analysis doesn't recognize `validateOutboundURL` as a sanitizer
- The protection stack is comprehensive, tested, and follows OWASP SSRF prevention guidelines
- Alternative solutions (custom CodeQL config, refactoring for explicit sanitizer pattern) add complexity without improving security
- Suppression comments document the protection rationale inline

**Learnings:**
- CodeQL taint tracking requires explicit sanitizer registration or inline suppression for custom validation functions
- `lgtm [query-id]` format is the standard for inline suppressions; works with both LGTM and GitHub Advanced Security CodeQL scans
- SSRF protection requires **layered defense**: URL validation + DNS-time IP blocking + connect-time validation + redirect validation
- DNS rebinding attacks require per-connection resolution (no client-side DNS caching) — this was already implemented
- Comprehensive CIDR blocklist protects against cloud metadata endpoints (169.254.169.254), private networks, and special-use ranges

## 2026-06-09 — Custom Catalog Era Validation Fix

**Learnings:**
- `models.Coin.Era` must not use Gin `oneof` binding because `PUT /api/coins/:id` binds directly to `models.Coin` before service validation; static binding rejected registry-defined custom eras too early.
- Coin era validation now lives in `src/api/services/coin_service.go`: built-in eras (`ancient`, `medieval`, `modern`) are always accepted, and other non-empty values must exist in `CatalogRegistry` via `repository.CatalogRegistryRepository.EraExists`.
- Catalog era validation now lives in `src/api/services/catalog_registry_service.go`: catalog entries accept any trimmed non-empty era up to 64 characters, enabling data-driven expansion without code rewrites.
- Regression coverage: `src/api/handlers/coin_handler_test.go` verifies update accepts a custom registry era, `src/api/services/coin_service_test.go` verifies service accept/reject behavior, and `src/api/services/catalog_registry_service_test.go` verifies custom catalog eras can be defined.

## 2026-06-09 — Coin Update Association Sync Fix

**Problem:**
- Coin updates failed with NOT NULL constraint violation: coin_set_memberships.added_at missing during PUT /api/coins/:id
- GORM Updates() automatically synced many2many associations (Tags, Sets), but default association behavior does not populate join table columns with NOT NULL constraints
- The CoinSetMembership model requires AddedAt time.Time (NOT NULL), but GORM default INSERT INTO coin_set_memberships (coin_id,set_id) VALUES ... ON CONFLICT DO NOTHING omitted this field

**Root Cause:**
- src/api/repository/coin_repository.go Update() method called r.db.Model(existing).Updates(updates) without omitting relationship fields
- When the updates Coin struct had a non-nil Sets field from a bound JSON payload, GORM attempted to sync the association automatically
- The default join table insert lacked AddedAt for CoinSetMembership, which is NOT NULL

**Solution:**
- Added Omit("Tags", "Sets") to the Update() method to prevent GORM from automatically syncing these many2many associations
- Tags and Sets must be managed through dedicated methods:
  - Sets: repository.SetRepository.AddCoinToSet() which explicitly sets AddedAt: time.Now()
  - Tags: tag service methods

**Files Changed:**
- src/api/repository/coin_repository.go — Added Omit("Tags", "Sets") to Update() method
- src/api/repository/coin_repository_test.go — Added repository-level regression coverage for set membership preservation
- src/api/handlers/coin_handler_test.go — Added handler-level regression coverage for PUT /api/coins/:id with a sets payload

**Testing:**
- Targeted tests pass: TestCoinHandler_Update_WithSetsPayloadPreservesMemberships, TestCoinRepository_Update_PreservesSets, TestCoinRepository_Update_WithSetsField
- Full Go API suite passes (go test -v ./...)

**Learnings:**
- GORM Updates() syncs associations by default if the struct has non-nil slices for many2many fields
- Join tables with custom NOT NULL columns require explicit management — use Omit() to prevent automatic sync
- The pattern: when a join table has custom fields beyond FK pairs, manage it through explicit repository methods, not GORM association helpers
- This follows the existing pattern where AddCoinToSet() already handled the proper insertion with AddedAt

### 2026-06-09 — F013 backend typed mutation inventory

Inventory for F013/AE006-AE013 found the primary risky broad binds in `CoinHandler.Create` and `CoinHandler.Update`, both currently binding `models.Coin`. Related paths include purchase/sell, bulk assign-location/tag/set, tag/set/reference endpoints, intake commit, collection proposal commit, valuation updates, and availability listing-status updates. The safest next slice is tests-first: add a one-field edit regression that seeds storage, tags, sets, references, images, era, and value data, then introduce explicit create/update request DTOs with patch-style presence tracking so omitted fields do not zero existing values while read-side fields are ignored.

## 2026-06-09 — F013 Typed Coin DTO Slice

Added explicit `CoinCreateRequest`/`CoinUpdateRequest` handler DTOs and mapped them back to the existing `CoinService` path so storage-location, era, reference, value-snapshot, tag, and set behavior stays centralized. Important regression: one-field coin updates now ignore broad read-side payload fields (`id`, `userId`, images, tags, sets, storageLocation, timestamps, AI analysis) while preserving existing associations and recording the normal value snapshot.

## 2026-06-09 — F013 Batch Completion: Typed DTOs, Zero-Value Persistence, Nullable Semantics

**Session:** F013 critical workflow hardening, backend typed DTO/revision batch
**Agents:** Cassius (initial DTO contract), Maximus (review + block), Brutus (zero-value revision), Aurelia (nullable semantics), Maximus (re-review + approval)

**Outcome:** ✅ APPROVED, block cleared

**Sequence:**
1. Cassius: Implemented typed `CoinCreateRequest` and `CoinUpdateRequest` DTOs, switched handlers away from broad `models.Coin` binding
2. Maximus: BLOCKED — Model-shaped Updates risked skipping explicit zero values (false, "", 0); required presence-aware Select path + regressions
3. Brutus: REVISED — Added presence-aware selected fields, repository Select path, handler/repository regressions for false/empty/zero persistence
4. Aurelia: REVISED — Added explicit JSON null clear semantics for allowlisted nullable scalar fields (purchasePrice, currentValue, dates, dimensions)
5. Maximus: RE-APPROVED — Semantics explicit, simple, tested; omitted fields preserve, JSON null clears allowlisted scalars; block cleared

**Architecture:**
- Handlers map DTO field presence to GORM `Select()` field list
- Omitted fields automatically preserved (standard PATCH semantics)
- Allowlisted nullable scalars accept JSON null clear: purchasePrice, currentValue, purchaseDate, soldPrice, soldDate, weightGrams, diameterMm
- storageLocationId clear and references replacement remain on dedicated service/repository paths

**Validation:**
- ✅ `go test -v ./...` (147 tests pass)
- ✅ `go vet ./...`
- ✅ `git diff --check`
- ✅ Regressions cover zero-value persistence, nullable scalar clears, omitted field preservation, storage-location clear, references replacement

**T006-T010, T013 marked complete.** T011/T012 intentionally unchecked (broader regression coverage remains incomplete despite new handler/repository regressions passing). T014-T017 (fixture builders) pending.

## 2026-06-09 — F013 Go golden fixture builders

T014 now has backend golden coin builders in `src/api/testutil` aligned with the F013 fixture names. Builders return cloned model graphs and the optional persistence helper is explicit about caller-managed migrated test DB setup, which keeps repository/service tests deterministic without introducing production seed data.

## 2026-06-18 — Global Mint Location Backend

**Learnings:**
- Mint locations are now global admin-managed data in `src/api/models.MintLocation`, separate from per-user storage locations and coin ownership.
- `GET /api/mint-locations` is authenticated read-only; admin writes live under `/api/admin/mint-locations` and use the standard `AdminRequired()` route group.
- Display-name uniqueness is enforced through a stored normalized name generated by `models.NormalizeMintLocationName`, matching the map's case/punctuation-insensitive lookup needs.
- Seed data was copied from `src/web/src/data/ancientMints.ts` into `database.seedMintLocations`; the seed records `AppSetting{Key: "MintLocationSeedVersion", Value: "1"}` so seeded rows are created once and admin edits/deletes persist across restarts.
- Validation and regressions cover coordinate bounds, blank/duplicate aliases, normalized duplicates, seed idempotency, authenticated reads, and admin-only writes.

## 2026-06-18 — Collection Health Snapshot Admin Trigger

Added admin-only backend manual trigger `POST /api/admin/collection-health-snapshots/run`, wired through `AdminHealthHandler` with constructor-injected `CollectionHealthScheduler`. The trigger returns `{ "message": "Collection health snapshots run completed" }` and has focused success/401/403 handler tests.

**Learning:** Collection health snapshots follow the same admin scheduler trigger pattern as auction ending and coin-of-day: keep the handler thin, call the existing scheduler synchronously, and let admin route middleware enforce access.

## 2026-06-18 — WebAuthn Login Challenge Contract

- Investigated issue #299 iPhone PWA biometric login failure: registered credentials were present, but `POST /auth/webauthn/login/begin` returned the go-webauthn assertion wrapper under `options`, so the browser challenge lived at `options.publicKey.challenge` while the Vue login store expected `options.challenge`.
- Backend session persistence/TTL was correct: `BeginLogin` stores a 5-minute in-memory session keyed by `login_{userID}`, and finish paths distinguish missing vs expired sessions before origin validation.
- Fixed the backend contract to return `options` as the direct `PublicKeyCredentialRequestOptions` object for `navigator.credentials.get({ publicKey: options })`; regression test asserts `options.challenge` is non-empty and matches the stored session challenge.
- Origin/RP handling remains strict: configured `WEBAUTHN_RP_ID` supplies `rpId`; finish validates request origin against configured `WEBAUTHN_ORIGIN` values. iPhone PWA/Safari clients must use an HTTPS origin matching that allowlist and an RP ID matching the site host/domain.

## 2026-06-18 — Biometric Login Backend Complete

- WebAuthn login-begin contract fix: `POST /api/auth/webauthn/login/begin` now returns `{ username, options }` where `options` is the direct `PublicKeyCredentialRequestOptions` payload (NOT wrapped under `options.publicKey`). Added `TestWebAuthnHandlerLoginBeginReturnsRequestOptionsWithChallenge` regression test ensuring challenge is at top level. All handler and integration tests pass.
- Frontend authorization store tests updated to handle both flat and nested challenge shapes, trim usernames on begin/finish calls, and enforce missing-challenge guards before invoking browser biometrics.
- Constitutional compliance: Principle III (strict types and explicit contracts), Principle IV (simple focused fix), §17 Quality Gate (targeted regression for exact failing path).
- Targeted validation: `go test -v ./handlers -run "TestWebAuthnHandlerLoginBeginReturnsRequestOptionsWithChallenge|TestWebAuthnHandlerLoginFinish"` ✅, full `go test ./...` ✅, `go vet ./...` ✅.

## 2026-06-18 — WebAuthn Backup Eligible Flag Validation

- **Issue:** Biometric login failing with 401 "Backup Eligible flag inconsistency detected during login validation"
- **Root cause:** go-webauthn v0.17.4 validates that CredentialFlags.BackupEligible remains consistent between registration and login. Our code only stored SignCount, not the backup flags. When reconstructing credentials in loadCredentials(), flags defaulted to alse, causing validation failure if the authenticator returned 	rue during registration.
- **Fix:** Added BackupEligible and BackupState bool fields to WebAuthnCredential model. Store both flags during registration (RegisterFinish), restore both during login (loadCredentials). GORM migration adds columns with default:false (safe for existing credentials).
- **Learning:** WebAuthn Credential struct has a Flags field (not in Authenticator). The flags include security-critical metadata that MUST be persisted. The library's validation logic enforces immutability of BackupEligible per FIDO2 spec. Always store all credential metadata returned by FinishRegistration, not just the fields needed for basic authentication.
- **Test coverage:** Added TestWebAuthnHandlerLoadCredentialsRestoresBackupFlags regression test. All WebAuthn tests pass.
- **Constitution alignment:** Principle I (layered architecture), Principle XI (security hardening), Principle XII (FIDO2 compliance).

## 2026-06-18T22:59:00Z — WebAuthn Backup Eligible Storage Fix (Coordinated Session)

Completed team fix for issue #299: WebAuthn login validation failure due to missing backup flag persistence.

- **Cassius:** Implemented `BackupEligible` and `BackupState` field storage in WebAuthnCredential model; updated registration handler to store flags and login handler to restore with legacy null bootstrap; added repository `UpdateCredentialAuthData` for sign-count and flag updates.
- **Brutus:** Added three regression tests covering flag persistence, legacy bootstrap fallback, and flag precedence rules.
- **Coordinator:** Regenerated OpenAPI artifacts and validated full Go test/vet suite.

**Session log:** `.squad/log/2026-06-18T22-59-00Z-webauthn-backup-eligible.md`  
**Orchestration logs:** `.squad/orchestration-log/2026-06-18T22-59-00Z-{cassius,brutus,coordinator}.md`

All tests pass; architecture compliant; ready for merge.

## 2026-06-19 — Admin API Key Hardening

**Learning:** Admin route access is now JWT-only by default: API-key authentication still resolves identity for protected/user-scoped routes, but `/api/admin/*` rejects any request with `apiKeyId` before role checks. API-key capabilities must be parsed as exact comma-separated tokens so malformed values like `readwrite`, `xwritex`, and `notread` never grant read/write access.

### 2026-06-19 — Issue #313 backend media authorization

- Removed public static `/uploads` serving from the Go API and replaced it with DB-backed media authorization in `ImageService`/`ImageRepository`.
- Owner access to coin images and avatars is preserved through authenticated `/uploads/*filepath` and `/api/uploads/*filepath`; private coin images return 404 for other users and 401 without auth.
- Path traversal is rejected before joining against `UPLOAD_DIR`, and only DB-backed `CoinImage.FilePath` / `User.AvatarPath` records are served.
- Explicit visibility preserved where straightforward: accepted followers can fetch public active coin images for public owners, public user avatars are available to authenticated users, and active showcase media has a slug-scoped public endpoint.
- Targeted media handler tests pass; full `go test ./...` is blocked by pre-existing `containsString` redeclaration in `services/collection_tools_service_test.go` vs `services/coin_service.go`.
## 2026-06-19 — Agent app_context DTO Contract (#318)

Modeled Go's optional `app_context` payload explicitly in Python as `AppContext(route, activeCoinId)` and made agent request DTOs reject unknown fields. The context is threaded into collection chat so route/active coin metadata can resolve phrases like "this coin" without being silently ignored. Added Go JSON shape tests for `app_context` and Python model tests for accepted shape, aliases, and unknown-field rejection. Validation: `go test ./...`, `go vet ./...`, `pytest tests/ -v`, `ruff check app/ tests/`.

- **2026-06-19 — Go Architecture Gate Hardening (#317):** Removed non-test handler GORM imports by routing not-found checks through `repository.IsRecordNotFound`, moved auction-ending debug counts into `AuctionLotRepository`, and tightened `architecture_test.go` so `TestArchitecture` enforces no handler GORM/direct DB access plus documented service GORM exceptions only.

- **2026-06-19T15:21:36Z — PR #315 + #317 Approval:** Brutus re-reviewed both PRs after Maximus's lockout revision and APPROVED for merge. #317 implements full architecture boundary hardening: GORM imports banned from handlers, tightened to repository-only, documented legacy service exceptions in `allowedServiceGORMFiles` for future cleanup. Principle I compliance verified. #315 is Aurelia's SafeExternalLink pattern companion (external URLs hardened with XSS regression coverage). Validation: `go test -v ./...` ✓, `go vet ./...` ✓, targeted Vue tests ✓. Decision records merged to `decisions.md`. Orchestration log: `.squad/orchestration-log/2026-06-19T15-21-36Z-brutus-rereview-317.md`. Beta commit 2433277 queued at handoff.

- **2026-06-19 — Swagger/OpenAPI Route Drift Gate (#316):** Added `src/api/route_openapi_drift_test.go` to inventory routes registered in `src/api/main.go`, normalize Gin params to OpenAPI paths, and fail when public `/api` routes are missing from `src/api/docs/swagger.json`. Explicit exemptions are limited to root health checks, Swagger UI assets, root `/uploads/*filepath`, and `/api/internal/tools/*` internal callback routes. Added missing Swagger annotations for tag, health, agent proposal/status/value, user profile/avatar/Pushover test, social, showcase, calendar, alert/reminder, admin connection-test, auction-lot update, and auction-ending debug routes; regenerated `src/api/docs/*` and `docs/openapi.json` with `task openapi`. Validated with `go test -v -run TestRegisteredAPIRoutesAreDocumentedInOpenAPI .` and `go test -v ./...` from `src/api`.

- **2026-06-19 — Python Agent Dependency Locking (#321):** Agent dependencies now use `src/agent/uv.lock` with uv 0.11.22. CI runs `uv sync --locked --extra dev` then `uv run ruff check app/ tests/` and `uv run pytest tests/ -v`; security scan audits the locked dev environment with `uv run pip-audit`; Docker installs runtime deps with `uv sync --locked --no-dev --no-install-project`. Refresh command from `src/agent`: `uv lock --upgrade && uv sync --locked --extra dev`.

- **2026-06-19 — Non-Root Container Runtime (#319):** Root `Dockerfile` and `src/agent/Dockerfile` final stages now create an `app` user/group and switch to UID/GID `10001:10001`. The app image owns `/app`, `/app/data`, and `/app/uploads`; the agent image owns `/app`, `.venv`, and source paths. Deployment docs now require bind mounts to be writable by `10001:10001`, and `docs/threat-model.md` marks SC-7 mitigated. Validation: Docker unavailable locally (`docker` command not found), so build/run checks were not executed; `git diff --check` and a Dockerfile directive inspection script passed.

- **2026-06-19 — Streaming Internal Token Guard (#226):** Added a Python SSE sanitizer in `src/agent/app/streaming.py` that redacts JWT-shaped internal bearer tokens from streamed text chunks, Anthropic text blocks, and final `done.message` payloads before `format_sse`. The guard intentionally preserves collection proposal identifiers and proposal tokens such as `token-abc` so #217 commit_update UX remains unchanged. Validation: `uv run ruff check app/ tests/`, targeted `uv run python -m pytest tests/test_streaming.py -v`, and full `uv run python -m pytest tests/ -v` all pass.
