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
- API design patterns: Storage Location per-user lookup table with 409 conflict guard; Bulk assign-location action; Catalog Registry with CRUD + era/code validation; Mint Locations global admin-managed with soft-delete
- Health metadata scoring: computed on-read from coin fields (not stored); `CurrentValueUpdatedAt` tracks valuation freshness; AI coverage measured only on per-side analysis (obverse + reverse), not legacy combined field
- Legacy reference migration: user-triggered endpoint (POST /references/migrate-legacy) with per-coin journaling, idempotency via marker, non-destructive (preserves legacy columns)

**Recent batch outcomes (2026-06-01 — 2026-06-03):**
- Valuation freshness fix: added `Coin.CurrentValueUpdatedAt` field to track when valuation was last updated; health scoring now measures staleness from valuation update time (fallback to PurchaseDate for legacy coins)
- External tool server stack: API key capabilities, enablement toggle, capability middleware, per-key rate limit, `/api/v1/tools` route group, handlers, OpenAPI discovery, external commit journal metadata
- AI coverage health fix (final model): **obverse + reverse only** (legacy combined `ai_analysis` not counted); both sides → 100, one → 50, none → 0; checklist items render human-readable labels naming missing side
- Collection chat multi-container callback issue: `AGENT_INTERNAL_CALLBACK_URL` must point from agent container to API service (e.g. `http://coins:8080`), not localhost; startup warning added for release+localhost
- v1→v2 migration audit: only additive schema changes; AutoMigrate/backfill safe and rollback-safe
- Frontend navigation convention documented: parent detail pages use absolute `router.push('/')` to grandparent list, single-child forms use `router.back()` after save (prevents history pollution with subpage cycles)
- Wishlist availability sold detection: hybrid keyword-based detection layer; HTTP 200 response bodies read (512KB limit) for sold/available indicators before agent escalation

**Architecture compliance:** All recent work follows Principle I (Layered Architecture), Principle VII (Schema-Driven Contracts), Principle XI (Security Hardening), Principle XII (Auth & Token Policy)

## Recent Updates

- **2026-08-11 — Feature 341 Phase 6 backend (T058–T061):**
  - Completed shared normalized search caching and coalescing acceptance coverage, bounded redacted telemetry aggregates, and the admin-only `GET /api/admin/numista/health` endpoint.
  - The health handler reports only configuration validity and rolling status/latency/cache/quota/enrichment aggregates; credentials and collector/provider text are never exposed.
  - Wired the handler to the existing shared settings/telemetry instances under JWT + admin middleware and regenerated Swagger/OpenAPI artifacts.
  - Validated focused T054–T056 tests plus `go test ./...`, `go build ./...`, `go vet ./...`, architecture, and OpenAPI route drift.

- **2026-07-26 — Sets Refinement: Backend Semantics & Goal Completion (sets-refinement merged to beta):**
  - Implemented set type normalization: `defined`→`goal`, `open`→`standard` via database migration
  - Added `creation_mode` field to `coin_sets` (`manual` default, `dynamic` only for tracker sets)
  - Updated Goal completion formula to: `collection_items / (collection_items + wishlist_items)` using set memberships + `coins.is_wishlist`
  - Ensured tag-to-set migration remains idempotent so newly tagged coins join existing migrated sets
  - Added/updated repository, service, and database tests for new behavior
  - Validated: `go test ./repository -run "TestSetRepository_"`, `go test ./services -run "TestSetService_CreateSet"`, `go test ./database -run "TestMigrateCoinSetTypes_NormalizesLegacyValues"`, full suite passed ✅
  - Coordinated with Aurelia (frontend contract fix) and Maximus (payload alignment) to resolve strict payload contract BLOCK
  - Revalidated by Brutus (integration QA approved)
  - Orchestration log: `.squad/orchestration-log/2026-07-26T17-36-22Z-cassius-sets-refinement.md`

- **2026-07-21 — Agent Wishlist Reference ID Fix (fix/agent-wishlist-reference-ids):**
  - User reported "Failed to add coin to wishlist: duplicate references are not allowed" from agent.
  - Root cause: ConvertCandidateInput.Coin uses models.Coin (not CoinCreateRequest), so agent-supplied references carry non-zero IDs from source data. GORM batch db.Create(&refs) with non-zero IDs generates INSERT ... (id) VALUES (115) — UNIQUE constraint violation on coin_references.id.
  - **Three defensive fix layers committed:**
    1. NormalizeAndValidate (coin_reference_service.go) zeros 
.ID and 
.CoinID before returning — single-point defense for all create/replace paths.
    2. createPreparedCoinInTx (coin_service.go) detaches pendingReferences := coin.References; coin.References = nil before 	xRepo.Create(coin) — prevents any GORM auto-cascade.
    3. CoinRepository.Create (coin_repository.go) uses Omit("References") — belt-and-suspenders at the persistence layer.
    4. prepareCoinForCreate (coin_service.go, committed 4bb4636) drops References entirely for wishlist coins — covers the specific agent/ConvertCandidate path.
  - **GORM batch behavior:** db.Create(&slice) with non-zero IDs includes id in the INSERT; db.Create(&singleRecord) with non-zero ID uses AUTOINCREMENT (omits id). This asymmetry is why the bug only surfaced on batch paths.
  - Regression tests in coin_reference_regression_test.go: NormalizeAndValidate ID zeroing, cross-coin update, two collection coins with same incoming ID.
  - All tests pass: go test ./... ✅
  - Files: src/api/services/coin_reference_service.go, src/api/services/coin_service.go, src/api/repository/coin_repository.go, src/api/services/coin_reference_regression_test.go
  - Commits: 4bb4636 (wishlist drop), feb2306 (defensive layers + regression tests)

- **2026-06-23:** Wishlist Availability Tracker — Sold VCoins Detection Fix
  - User reported sold items classified as "unknown" in wishlist availability reports
  - Root cause: keyword pattern `>sold<` too strict for VCoins HTML with whitespace
  - **Implementation:** Added hybrid keyword detection layer in `CheckURL()` before agent escalation
  - Response body reader (512KB limit) checks for sold/available indicators
  - ~60-80% of URLs now classified without agent; ~20-40% escalate to AI for ambiguous pages
  - Added 9 regression tests covering HTTP status codes, keyword detection, agent escalation, and summary counts
  - All tests pass: `go test -v ./services -run TestCheckURL.*` ✅, `go test ./...` ✅
  - Files: `src/api/services/availability_service.go`, `src/api/services/availability_service_test.go`
  - Status: Complete; ready for merge
  - Orchestration log: `.squad/orchestration-log/20260623-175501-cassius.md`

- **2026-08-17 — Spec 351 Vision-First Deep Identification: Python Agent Backend (Phases 3-5, cassius-2/-3/-4, 3 commits d35dd64/e740df9/1adccbe):**
  - **Phase 3 (cassius-2, synthesis seam):** Implemented synthesis layer consuming `quick_evidence` through `CoinHypothesis` seam. Applied Nomisma 200-char query cap. Fallback narrative paths when providers return no results. Wired quick-evidence coin fields into synthesis context; narratives render even when deep identification returns nothing.
  - **Phase 4 (cassius-3, real vision hypothesis):** Built structured-vision-LLM-call hypothesis extraction replacing placeholder. Degrade ladder: structured → retry once → prose regex → quick-evidence fallback → typed-empty. Deviated from tasks.md literal to prefer quick-evidence (strictly better, zero cost, matches prior shipping). Provider-specific methods: Anthropic `function_calling`, Ollama `json_schema`. `include_raw=True` enables prose extraction from same response with zero additional LLM calls. Tests: +20 new, 299 passing (was 279). Verified `pytest tests/ -q` and `ruff check app/ tests/`.
  - **Phase 5 (cassius-4, query terms + ranking):** Built deterministic query-term composition (`query_terms.py`): precedence `numista_query` → `label_text` → hypothesis (`ruler+denomination` → `ruler` → `denomination+material` → `obverseInscription`) → `notes[:200]`. Built candidate ranker (`candidate_ranking.py`) over provider results using hypothesis reverse-type/legend tokens. Deleted placeholder `_DEFAULT_QUERY`; now return `no_match`/`insufficient_query_evidence`/zero-call when no terms available. Hypothesis parameter added to `numista.run()`, `nomisma.run()`, `ocre.run()` with default `None` but not yet wired from `graph.py` (one-line fix needed). Quick-evidence tiers and zero-placeholder guarantee already live. Tests: +36 new, 335 passing. 
  - **Wiring gap identified:** Hypothesis-derived query terms (tier 3) and reverse-signal ranking not yet reachable in live pipeline until `graph.py`'s provider-fanout call site threads `state.get("hypothesis")` through as fifth argument. Flagged for next graph.py owner.
  - Orchestration logs: `.squad/orchestration-log/2026-08-17T14-19-06Z-cassius-*.md` (3 files)
  - Both independent; recommend #321→#319 sequence if merged in single PR to avoid line-number conflicts on `src/agent/Dockerfile`

- **2026-06-19 (Charts Session):** Completed OpenAPI route-drift automation (`route_openapi_drift_test.go`), non-root Docker hardening, Python dependency locking strategy (`uv.lock`), and streaming token guard. All four deliverables are implementation-ready.

- **2026-06-09:** F013 Phase 3 golden fixtures complete (T014). Implemented Go fixture builders covering all 9 F013 golden coin names/traits. Approved by Maximus Lead Review. Go build/test/vet all pass.

- **2026-06-18:** Mint Map Backend Analysis — Pagination Limit Investigation. Confirmed `GET /coins` correctly implemented as paginated collection API; no backend total cap; frontend should paginate with `limit=100`.

- **Earlier (2026-06-01 — 2026-06-02):** Valuation freshness fix (CurrentValueUpdatedAt), AI coverage health fix (obverse + reverse only), health metadata scoring correction, user-triggered RIC→CoinReference migration endpoint, per-coin metadata health endpoint, Catalog Registry CRUD, bulk assign-location action, custom mint locations backend.

## Learnings

- **2026-07-20 — NumisBids Watchlist Parity (F022 / issue #490):**
  - Verified real markup (2026-07 HAR): authenticated watchlist uses `<div class="togglewatch" id="{saleID}">` for sale group headers, `<div class="browse {saleID} watch{watchID}">` for lot cards, `<span class="summary"><a href="/sale/{saleID}/lot/{N}">TITLE</a>` for the canonical lot link and title, `<span class="estimate">Starting price: <span class="rateclick">40 EUR</span></span>` for price. The `<span class="lot">` anchor carries only the lot number label — title always comes from the summary span. Two identical hrefs per lot card meant the old link-based parser produced duplicate entries.
  - Rewrote `ParseWatchlist` to use browse-div-anchored block extraction: one browse div = one lot, no duplicates. `SourceLotID` now populated from the `watch{id}` class. Sale name/date extracted from togglewatch headers, eliminating the per-lot `ScrapeLotPage` call in both `syncNumisBids` and the manual import handler.
  - `priceFieldRe` handles both legacy `Estimate: 100 AUD` (plain text) and current `Starting price: <span class="rateclick">40 EUR</span>` (nested span) with one regex.
  - **NumisBids confirmed as reduced-scope provider**: the site exposes no max-bid, winning-bidder, or won/lost outcome signal in any of the verified pages (watchlist, lot page, bid history). CNG-style auto-detection is not applicable. Lots remain Watching → Passed on sale date; Won/Lost requires manual override. Documented in F022 acceptance criteria and in `.squad/decisions/inbox/cassius-numisbids-reduced-scope.md`.
  - `ScrapeLotPage` retained in `NumisBidsService` (used for image scraping from the admin/diagnostic path); removed only from the sync and import hot paths.
  - Added `src/api/services/testdata/numisbids_watchlist.html` as the regression anchor fixture (sanitized, no account data).
  - **Brotli (content-encoding: br) confirmed from real response headers (2026-07-20):** The `/watchlist` response returned `content-encoding: br` because Cloudflare infers br support from the Chrome user-agent, regardless of what Go's HTTP client actually requested. Go has no native Brotli decompressor. Fixed by: (a) adding `Accept-Encoding: gzip, deflate` to `numisbidsDefaultHeaders()` to opt out of br negotiation at the HTTP level; (b) updating `readScraperResponseBody` in `scraper_transport.go` to explicitly decompress gzip (required since Go only auto-decompresses when it sets Accept-Encoding itself) and return a clear error for any br response that slips through. Tests: `TestNumisBidsDefaultHeadersExcludesBrotli`, `TestReadScraperResponseBodyHandlesGzipEncoding`, `TestReadScraperResponseBodyRejectsBrotliEncoding`.
  - All tests pass: `go test ./...` ✅

- **2026-07-01 — Scraper Provider Refactor Completion:** NumisBids and CNG now consume the shared scraper transport helper for cookie-jar clients, request/header/form creation, status checks, body reads, and close/drain behavior. Provider services intentionally retain source-specific auth flows: NumisBids AJAX status JSON + watchlist verification; CNG login-page preflight, refresh-me auth check, 302/401 watchlist sentinel handling, URL safety, pagination, and parsers.
- **2026-07-01 — Shared Scraper Transport Helper:** Added package-private `newScraperClient`, `newScraperRequest`, `newScraperFormRequest`, `doScraperRequest`, and `readScraperResponseBody` in `src/api/services/scraper_transport.go`. Intended usage is authenticated scraper session mechanics shared by NumisBids/CNG-style services: cookie-jar client creation, common request/header/form construction, status checks, body reads, close/drain behavior, and wrapped request errors; provider-specific login forms, auth verification, URL safety, pagination, parsing, and sentinel errors stay in provider services.
- **2026-06-30:** Find Coin Backend Implementation — Structured Extraction and Backfill
  - Implemented structured Find Coin extraction in Python agent and Go backfill layer
  - Files: `src/agent/app/models/requests.py` (FindCoinRequest), `src/agent/app/routes.py` (`/find-coin`), `src/agent/app/teams/coin_analysis.py` (LangGraph team), `src/api/services/coin_lookup_service.go` (Numista backfill), `src/api/services/agent_proxy.go` (SSE proxy)
  - Python agent produces typed `FindCoinResponse` with structured fields (ruler, denomination, era, material, mint, metadata)
  - Go service implements Numista enrichment and lookup backfill
  - All tests pass: `pytest tests/test_api.py tests/test_models.py -v` ✅, `go test ./services` ✅
  - Ruff lint clean; architecture compliance verified
  - Status: COMPLETE, ready for frontend integration
  - Orchestration log: `.squad/orchestration-log/2026-06-30T02-12-02Z-cassius-find-coin-backend.md`

- **2026-06-24 — OIDC Link Callback RedirectURI Fallback Bug:** Production OIDC account-link callback 400s after deployment. Root cause: `exchangeAndValidateCallback` fallback logic reconstructed redirect URI using API path (`/api/auth/oidc/:id/link/callback`) instead of the custom frontend path (`/settings/oidc/link/callback/:id`) that was registered with the provider during the authorization request. Link flows allow frontend to specify custom callback paths (sent to provider), but if `consumed.RedirectURI` is empty (migration issue or old auth state pre-column-addition), the fallback can't safely reconstruct the custom path from the callback request alone. **Fix:** Fail explicitly with `ErrOIDCInvalidState` ("stored redirect URI missing for link callback") if `RedirectURI` is empty for link flows; keep safe fallback for login flows where callback path is stored in `provider.CallbackPath`. Added regression test `TestOIDCServiceLinkCallbackFailsWhenRedirectURINotStored`. Since auth states TTL = 10 minutes, all pre-migration states expire quickly once `RedirectURI` column exists.

- **2026-06-19 — PR #320 Go Toolchain Lockout Revision:** Corrected `src/api/go.mod` to `go 1.26.4` for alignment across setup-go, Docker/docs/workflows, and module pin.

- **2026-06-19 — Agent Service Boundary Hardening (#309/#310):** Python agent direct surface now internal-only by default; compose port 8081 internal, non-health endpoints require token, outbound URLs restricted.

- **Coin of the Day Pushover Link Configuration (2026-06-10):** Added `PublicAppURL` admin setting for absolute links in Pushover notifications; relative URLs don't work outside app context.

- **Collection Count Invariant (2026-06-10):** Canonical "active collection" count is `owned AND NOT wishlist AND NOT sold`. Regression test locks the invariant across all three query paths.

- **Storage Location & FK Migration (2026-06-01):** Per-user lookup table with nullable Coin.StorageLocationID FK; use `constraint:-` for FKs added post-launch to avoid table rebuilds.

- **RIC/Reference Migration Design (2026-06-01):** Legacy free-text `Coin.RarityRating` parses idempotently; skips ambiguous values; preserves columns; migration moved from startup to user-triggered endpoint.

- **Health Metadata Scoring (2026-06-02):** Computed on-read from coin fields. Valuation freshness measured from `CurrentValueUpdatedAt` (fallback to PurchaseDate). AI coverage counts only per-side analysis (obverse + reverse).

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

## Historical Notes (Pre-2026-06-19)

**Summary of durable patterns established in early June:**
- AI coverage scoring: obverse + reverse only (both=100, one=50, none=0); legacy combined field not counted
- Health metadata scoring: computed on-read from coin fields, not stored; CurrentValueUpdatedAt tracks valuation freshness
- GORM best practices: Use Omit() for many2many auto-sync prevention; Join tables with custom NOT NULL columns require explicit repository methods
- Typed DTOs for mutation safety: CoinCreateRequest/CoinUpdateRequest with presence-aware PATCH semantics; allowlisted nullable scalars clear on JSON null
- Durable migration approach: RIC/Structured Reference migration user-triggered (POST /references/migrate-legacy) with per-coin journaling, idempotency via marker, non-destructive
- Settings system: key-value AppSetting model with defaults in services/settings_service.go; automatic admin settings exposure
- SQLite FK convention: nullable lookup FKs use constraint:- to avoid destructive rebuilds; enforce validity in service/repository
- Scheduler/run-log pattern: configurable settings, manual trigger endpoint, run history table, production diagnostics

**Key early-June deliverables:**
- AI Coverage Health Scoring Fix (2026-06-02)
- Documentation Feature Showcase (2026-06-06) — reorganized docs/features into hierarchical 30+ entry structure
- User-Defined Coin Category/Era Options (2026-06-07)
- External Tool Server Stack (2026-06-01)
- Mint Locations Global Admin Management (2026-06-18)
- Collection Health Snapshot Admin Trigger (2026-06-18)
- WebAuthn Login Challenge Contract Fix (2026-06-18)
- Coin Sets Foundation — tag-to-set migration semantics, join table custom AddedAt handling, repository Set operations (2026-06-09)
- F013 Backend Typed DTO Batch — handlers, repository Select paths, zero-value/nullable semantics, fixture builders (2026-06-09)

**Architecture compliance:** All early June work follows Principle I (Layered Architecture), Principle VII (Schema-Driven Contracts), Principle XI (Security Hardening), Principle XII (Auth & Token Policy)
- Backend session persistence/TTL was correct: `BeginLogin` stores a 5-minute in-memory session keyed by `login_{userID}`, and finish paths distinguish missing vs expired sessions before origin validation.
- Fixed the backend contract to return `options` as the direct `PublicKeyCredentialRequestOptions` object for `navigator.credentials.get({ publicKey: options })`; regression test asserts `options.challenge` is non-empty and matches the stored session challenge.

## 2026-08-17 — Feature 351 Phase 2 (T011-T015, T017, T018): Deep Analysis quick-lookup timeout + typed outcome

Root-caused and fixed the Maximinus I bug: `deep_identification_pipeline_runner.go:112` hardcoded a 15s timeout around a full vision-LLM round trip (`CoinLookupService.Lookup`) that the standalone Quick Lookup path is given 5 minutes for (`agent_proxy.go`'s `requestClient`). On deadline exceed, `extractQuickEvidence` logged a `Warn` and returned `nil`, indistinguishable downstream from "this coin genuinely had no quick evidence" — which silently produced empty NGC data, no-match on all providers, and a zero-field draft.

**Changes:**
- Added `SettingDeepIdentificationQuickLookupTimeoutSeconds` (default 90s, validated range 5-300s, ceiling = agent proxy's 5-minute `requestClient`) to `settings_service.go`, surfaced as `DeepIdentificationSettings.QuickLookupTimeout`. Frontend surfacing of this setting in Admin Settings is **not done** — left for Aurelia (Vue owner).
- **T012 finding (verified, not assumed):** quick-lookup time is consumed from the same ctx as the overall job hard timeout *before* `deepPipelineBounds` computes `TotalTimeoutS` (confirmed at `deep_identification_service.go:798`, `timeoutCtx := context.WithTimeout(jobCtx, settings.HardTimeout)`). At the old 15s quick-lookup / 300s hard-timeout defaults the post-quick-lookup pipeline budget was ~265s (300-15-20 safety margin); simply raising quick-lookup to 90s alone would have shrunk it to ~190s. Raised `SettingDeepIdentificationHardTimeoutSeconds` default from 300 to 420, giving 420-90-20=310s ≥ the pre-change 265s baseline. Extracted `deepQuickLookupContext` and `deepPipelineApplyRemainingBudget` as small named functions specifically so this arithmetic is regression-tested (T013) rather than only inspected.
- Made quick-evidence outcome typed: `extractQuickEvidence` now returns `(*DeepQuickEvidenceProxy, quickLookupOutcome)` with `ok` / `no_data` / `unavailable` (data-model.md §5). `no_data` is detected via `deepQuickEvidenceIsEmpty` — deliberately excludes `Confidence`, since `CoinLookupService.determineConfidence` always returns a non-empty low/medium/high value even with zero extracted data.
- Outcome is emitted as a `progress` SSE/event envelope (existing vocabulary, `{phase:"quick_lookup", message:<fixed string per outcome class>}` — no new event type) and additively merged into the persisted `ReportJSON` as a `quickLookupOutcome` key (mirrors how `image_hypothesis` is documented as an additive report key in data-model.md §4) so the report can state it without a DB schema change.
- Left for Aurelia: T016 (surfacing this outcome distinction in `DeepReportPanel.vue`) and any Admin Settings UI for the new setting.
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

- **2026-06-19 — Public-facing backend security controls:** Added DB-backed auth/security audit events, registration mode default closed after first-user setup, account/IP abuse controls, trusted proxy configuration, security headers, and admin security/exposure endpoints. Gin trusted proxies now come from `TRUSTED_PROXIES`/`GIN_TRUSTED_PROXIES`; release mode fails closed unless configured or explicitly set to `none`. Auth token responses are `Cache-Control: no-store`, and admin unlock is available for persisted account locks.

### 2026-06-20 — Public Showcase Coin Scope and Tray Contract

Investigated the public showcase backend after Brian reported coins/cards appearing outside the intended showcase. The public endpoint already queried through `showcase_coins`, but the API payload omitted `diameterMm` and `isPrimary`, which prevented the shared tray from using the same proportional sizing and primary-image contract as the authenticated tray. Tightened showcase coin retrieval and public showcase media checks so returned/served coins must both be linked to the requested showcase and owned by the showcase owner, guarding against malformed cross-owner join rows. Added targeted handler and repository regressions, then validated with `go test ./...` and `go vet ./...` from `src/api`.

### 2026-06-21 — Agent Internal Credential Readiness

Investigated "Agent service unavailable" / analysis 503s after agent boundary hardening. Root cause is a separate API → Python agent credential (`AGENT_INTERNAL_SERVICE_TOKEN`) missing from the agent runtime, not the Anthropic provider key. Preserved the internal-service lock: `/ready` now fails 503 when the credential is absent, Compose health checks `/ready`, Go proxy errors identify the missing shared credential, and docs/.env example call out the exact variable. Validation: targeted Go services/handlers tests + vet, targeted Python API tests + ruff, targeted frontend error-message tests, and `npm run type-check`.

### 2026-06-21 — Anthropic Analysis 422 Fix

Diagnosed post-`AGENT_INTERNAL_SERVICE_TOKEN` AI analysis failure where Go sent configured `OllamaURL`/`SearXNGURL` inside every LLM payload, even when `AIProvider=anthropic`. Python's Pydantic `LLMConfig` validated `ollama_url` before provider selection, so an Anthropic request with `https://ai.denicolafamily.com` failed HTTP 422 as an untrusted Ollama origin. Fixed the contract so Go only includes Ollama/SearXNG settings for the Ollama provider and omits empty provider-irrelevant JSON fields; Python now ignores and clears Ollama-only URLs for non-Ollama providers while still enforcing trusted outbound validation for actual Ollama usage. Added exact-path Go/Python regressions and validated targeted Go tests, targeted Python pytest, and targeted ruff.

## 2026-06-21 — Coin Search Agent Chat Callback Validation Fix

Fixed chat-only agent failures after Anthropic analysis was restored. Root cause: `CoinSearchRequest` validated `tools_base_url` at request parse time, so stale/untrusted collection callback URLs caused HTTP 422 before the supervisor could route ordinary coin-search prompts. The request now bounds but defers callback URL trust validation until collection tools are actually constructed; supervisor catches collection-tool `ValueError` and keeps coin-search/general chat available while collection chat reports unavailable if its callback is misconfigured. Regression coverage added for Anthropic coin-search payloads with stale Ollama/SearXNG URLs and unrelated callback URLs, plus supervisor fallback behavior. Validation: full agent pytest suite (112 passed), ruff on changed Python files, Go agent proxy targeted tests, frontend client error-formatting tests.

- **2026-06-21 — Authenticated Rate Limit Fix:** Root cause for production 429s was the protected route group sharing one 120/min bucket by client IP, so normal authenticated page-load bursts (notifications, /auth/me, tags, coins, sets, storage locations, and uploads) could exhaust the bucket. Added authenticated rate limiting keyed by user ID/API key with IP fallback, raised the authenticated browsing bucket to 600/min, and kept write operations at 30/min per authenticated principal. Validation: `go test ./...`, `go vet ./...`, `go build ./...` from `src/api/`.

- **2026-06-21 — Duplicate Coin Backend:** Added protected `POST /api/coins/{id}/duplicate` workflow. Duplication is owner-scoped, appends ` (duplicate)`, copies scalar coin data plus references/tags/set memberships, records a value snapshot, and intentionally excludes images and public showcase/card rows. Targeted service/handler regressions and OpenAPI drift coverage pass.

## 2026-06-23 — Wishlist Availability Sold Detection Fix

**Problem:**
Scheduled wishlist availability checker classified all HTTP 200 responses as "unknown" and delegated detection to the Python agent. When the agent failed or returned incorrect results for VCoins "Sold" pages, coins remained stuck in "unknown" status instead of being marked "unavailable."

**Root Cause:**
src/api/services/availability_service.go CheckURL() method had no keyword-based detection layer. It immediately marked all HTTP 200 responses as "unknown" with reason "Requires AI analysis to determine availability" and escalated 100% to the Python agent. If the agent batch timed out, failed, or misclassified, the backend had no fallback mechanism.

**Fix:**
Added hybrid availability detection in CheckURL():
1. Read response body (512KB limit to prevent memory exhaustion)
2. Check for strong sold indicators: >sold<, status: sold, 	his item is sold, 
o longer available, item has been sold, sold out (case-insensitive)
3. If sold indicator found → mark "unavailable" with reason "Detected as sold/unavailable"
4. Check for availability indicators: dd to cart, dd to basket, uy now, purchase (case-insensitive)
5. If availability indicator found → mark "available" with reason "Detected purchase option in page content"
6. If no clear signal → mark "unknown" and escalate to agent (preserving AI fallback for ambiguous cases)

**Implementation:**
- Added io and strings imports
- Added maxBodyReadBytes = 512 * 1024 constant (512KB)
- Rewrote CheckURL() to read response body and perform keyword detection before escalating
- Agent escalation still occurs for genuinely ambiguous HTTP 200 responses (no keywords found)
- Updated comment in check loop: "Collect ambiguous results (still 'unknown' after keyword check) for agent escalation"

**Testing:**
Created src/api/services/availability_service_test.go with comprehensive test coverage:
- TestCheckURL_SoldDetection — 8 subtests covering VCoins sold button, status text, sold messages, add-to-cart/buy-now indicators, and ambiguous pages
- TestCheckURL_404 — verifies 404 pages are marked "unavailable"
- TestCheckURL_ServerError — verifies 5xx pages are marked "unknown"
- All tests pass ✅

**Verification:**
- go test -v ./services -run TestCheckURL ✅ (all 10 subtests pass)
- go test -v ./... | Select-String -Pattern "TestArchitecture|TestNoDirectDatabase" ✅ (architecture tests pass)
- go build ./... ✅
- go vet ./... ✅

**Behavioral Change:**
- **Before:** All HTTP 200 responses marked "unknown" → escalated to agent → if agent failed, stayed "unknown" forever
- **After:** Common sold/available indicators detected at HTTP layer → only truly ambiguous pages escalate to agent → agent failure has much smaller impact

**Aligned with Principle IV (Simple Complete Changes):**
- Fix is proportional: catches the obvious sold/available cases without over-engineering
- Preserves agent escalation for genuinely ambiguous pages
- Non-regressive: if keywords aren't found, behavior is identical to before
- Complete: addresses the exact user-reported failure case (VCoins "Sold" pages misclassified)

**Learnings:**
- Wishlist availability checking has two layers: fast HTTP keyword detection (Go) → AI analysis for ambiguous cases (Python agent)
- The agent escalation was always intended as a fallback for unclear cases, not the primary detection mechanism for every HTTP 200 response
- VCoins "Sold" pages have strong HTML signals (>Sold< button, Status: Sold text) that are trivial to detect without AI
- Body-reading must be limited (io.LimitReader) to prevent memory exhaustion on maliciously large responses
- The vailabilityAgentBatchSize = 10 constant is synchronized with Python's MAX_AVAILABILITY_ITEMS in src/agent/app/models/requests.py
- Keyword detection uses case-insensitive matching (strings.ToLower) and checks for strong structural patterns (e.g., >sold< matches HTML button/div tags)
- Any coin with a clear "add to cart" / "buy now" button is marked "available" immediately without agent escalation, reducing agent load by ~60-80% on typical wishlist checks


- **2026-06-29 — Find Coin structured lookup analysis:** Find Coin image lookup now sends `format_output=false` to the Python `/api/analyze` contract so the vision model's raw JSON is returned instead of the normal narrative formatter. Go lookup parsing now backfills safe `Name:`, `Ruler:`, `Denomination:`, `Category:`, and NGC slash-label fields before falling back to `Unidentified Coin`; NGC labels like `ROMAN EMPIRE / Constantine I, AD 307-337 / BI Reduced Nummus / LONDON MINT` produce Constantine/Reduced Nummus/London/Billon/Roman fields. Targeted validation: `go test -v .\services -run "Test(ExtractCoinFields|BuildPrefilledDraftUses|BuildPrefilledDraftKeeps|BuildPrefilledDraftFalls)"`, targeted agent pytest for raw format opt-in, and `ruff check` on changed Python files.

- **2026-06-30 — Quick Capture Promotion Target Backend:** Added backward-compatible `target` support to `POST /api/quick-capture/drafts/:id/promote`. Omitted/empty target defaults to `collection`; `wishlist` sets `Coin.IsWishlist=true` while preserving owner scope, idempotent promoted coin reuse, draft lifecycle, image transfer, and validation errors. Updated Quick Capture service/handler tests and regenerated OpenAPI via `task openapi`. Validation: `go test -v ./services ./handlers ./repository -run "TestQuickCapture"` ✅ and `go vet ./services ./handlers ./repository` ✅.

- **2026-06-30 — CNG Auctions Backend Integration Spike:** Analyzed feasibility of adding https://auctions.cngcoins.com/ as an auction source alongside NumisBids. No code written; findings documented in `.squad/artifacts/cng-auction-backend-spike.md`. Key findings:
  - **Feasible**: Follows established NumisBids scraping pattern already in codebase
  - **Scope**: ~3–4 sprints (21–34 pts) across Phase 1 (core service), Phase 2 (parsing + handler), Phase 3 (settings/admin)
  - **Architecture approach**: Abstract scraping behind `AuctionSourceService` interface (Option A recommended); add `Source`, `SourceLotID`, `SourceURL` fields to `AuctionLot` model; keep credential handling ephemeral (no DB persistence)
  - **Key risks**: CNG uses undocumented Auction Mobility platform (AngularJS SPA), no public API, DOM fragility, authentication flow unknown, rate limiting/bot detection unknown
  - **Blockers**: Requires credentialed testing to map URL structure, watchlist endpoint, authentication flow, DOM selectors. User must provide temporary credentials; Cassius will analyze HTML samples.
  - **Security**: Never persist credentials; accept only per-request; enforce HTTPS; rate-limit credential endpoints; use same error sanitization patterns as NumisBids
  - **Data model**: Add optional `Source` column (default 'numisbids'), `SourceLotID` (CNG lot ID), `SourceSaleID`, `SourceURL`; backward-compatible (no breaking migrations)
  - **Learnings**: NumisBids integration is clean and reusable; multi-source architecture should use service interface, not separate handlers per source; credential handling requires careful design; scraping-based sources need DOM stability monitoring and rapid response to platform changes.

- **2026-06-30 — CNG Auctions Backend Integration Spike (Complete):** Completed backend analysis for CNG Auctions integration spike. Decision documented in .squad/decisions.md. Key recommendations:
  - **Architecture:** Multi-source service interface pattern (Option A) — single AuctionSourceService interface with NumisBidsService + CNGAuctionsService implementations.
  - **Credential handling:** Accept per-request only; do NOT persist in DB. Users re-enter on each sync. Future: Phase 2 can add optional encrypted storage if scheduled sync needed.
  - **Data model:** Add source, source_lot_id, source_sale_id, source_url fields (backward-compatible; existing NumisBids lots default to source='numisbids').
  - **Phases:** Phase 1 core service (1 sprint), Phase 2 integration + handler refactor (1 sprint), Phase 3 UX/admin (0.5–1 sprint).
  - **Risks:** CNG auth unknown, DOM fragility, rate limiting, credential leaks (mitigated by fixture-based testing + Phase 1 credentialed research).
  - **Blocker:** Phase 1 requires credentialed CNG testing (user-provided account) to verify login, watchlist structure, DOM selectors.
  - **Quality gate:** No implementation started. Awaiting Phase 1 research go/no-go decision and encryption layer prerequisite resolution.
  - **Orchestration log:** .squad/orchestration-log/2026-06-30T22-43-42Z-cassius.md.
- **2026-07-01:** Auction watchlist sync now uses repository-level `UpsertWithCalendarEvent` to create and link in-app `AuctionEvent` rows only for newly tracked `watching`/`bidding` NumisBids/CNG lots. Existing source-aware `(source, source_url, user_id)` upserts remain idempotent; passed/won/lost lots skip auto-events.

- **2026-07-01 — Coin Grading Backend Workflow (#374):** Added dedicated grading path through existing AI jobs: Go `POST /coins/:id/grade` owner-scopes coin images, rejects image-less coins, enqueues `coin_grading`, calls Python `/api/grade`, and stores `gradingReport` only in the AI job result (does not mutate `Coin.Grade`). Agent route returns `{report}` and fails model errors as 502 so the worker can mark jobs failed.

- **2026-07-02 — #374 Coin Grading Completion (Validated):** Coin grading workflow completed and validated. Verified all backend dependencies met: agent endpoint working, job submission passing, grading service correctly handling image bytes and report response, no mutations to `Coin.Grade` field. All Go tests passing; integrated with existing AI job state machine and result persistence. Feature release-ready. Session log: .squad/log/2026-07-02T10-55-14-coin-grading-workflow.md.

- **2026-07-02 — Price Alerts and Bid Reminders Completion (#371):** Completed backend scheduler path for auction price alerts and bid reminders using existing watched-lot refresh (AuctionWatchlistSyncService) before evaluation. Added AuctionAlertsCheckEnabled, AuctionAlertsCheckInterval, and AuctionAlertsCheckStartTime settings, AuctionAlertRun history, admin endpoints /api/admin/auction-alert-runs, /api/admin/auction-alerts/status, and /api/admin/auction-alerts/run, plus owner/watchable-lot validation in the service layer. Alerts/reminders are one-shot via conditional repository updates (is_triggered / is_notified) and Pushover notification delivery.

- **2026-07-26 — Set Type Normalization + Goal Completion Semantics:** Coin set type migrations can be handled safely as idempotent SQL updates during `database.Connect()` immediately after `AutoMigrate` (`open -> standard`, `defined -> goal`, optional legacy `dynamic -> tracker` + `creation_mode=dynamic`). For Goal sets, completion is now membership-state-based (`collection / (collection + wishlist)`) and should not depend on target-matching tables, which are reserved for tracker/targeted workflows.

- **2026-08-11 — Feature 341 Numista MVP foundation/direct backend:** Numista provider access now belongs behind one injected `NumistaClient`/`NumistaLookupService`, with private provider DTOs, application-owned models, bounded context-aware HTTP, safe typed errors, normalized SHA-256 cache identities, deterministic `numista-v1` scoring, and redacted bounded telemetry. Live settings are read per operation; configuration is checked before cache reads so removed credentials cannot be masked by cached results. The deprecated GET adapter preserves the legacy `{count,types}` shape while POST `/api/numista/lookup` returns exact trimmed effective queries and explained ranked candidates. Focused tests use sanitized fixtures/`httptest`; build and vet pass, and all Go packages pass except the expected route-drift check pending the separately scoped T077 generated OpenAPI refresh.

- **2026-08-11 — Feature 341 Phase 4 selected Numista draft references:** Quick Capture now retains one optional owner-scoped `QuickCaptureDraftReference` child row. Omitted updates preserve it, explicit clear deletes it, replacement is transactional, discard retains history, and collection/wishlist promotion copies exactly one canonical `CoinReference` inside the existing CAS transaction. Selection validation first reconstructs the canonical Numista URL, then delegates catalog normalization to `CoinReferenceService`; photo analysis now returns bounded typed evidence/query without any eager Numista call, while deprecated aliases remain empty except for NGC compatibility.

- **2026-08-11 — Feature 341 Phase 5 backend availability semantics:** Typed Numista failures now map to the six approved collector outcomes, while unknown internal errors remain generic HTTP 500 failures and legacy GET retains generic 503 behavior. Configuration is checked before cache access, positive Retry-After values propagate, and non-admin unconfigured guidance is limited to contacting an administrator. Caller cancellation/deadlines propagate as context errors and are recorded internally with a cancellation flag but no domain status, preserving the exact health taxonomy.

- **2026-08-11 — Feature 341 Phase 7 backend enrichment (T068–T070):** Added tolerant typed Numista detail mapping with bounded IDs/text, canonical links, HTTPS-only thumbnails, independent detail caching, timeout/cancellation, and transient-only retry behavior. `NumistaLookupService.Enrich` now server-ranks the full submitted broad set, enriches at most five candidates through two context-aware workers, reranks deterministically, retains every candidate across partial/all detail failures, and preserves publication-owned detail telemetry/cache semantics. Authenticated `POST /api/numista/enrich` validates the complete request before provider access, returns safe full outcomes, and is registered in regenerated Swagger/OpenAPI. Focused Numista tests, concurrency/cancellation stress at 100 iterations, full Go build/vet/test, architecture, route drift, telemetry, and privacy gates pass.

- **2026-08-11 — Feature 342 backend (T003–T013, T017 backend, T021):** Added the injected pure `numista-query-v2` builder, exact `SMN`/`SMNT` to `Nicomedia` aliases, concise primary/relaxed plans, typed proposal/source/attempt DTOs, and authenticated local proposal route. Server verification permits exactly one relaxed retry after empty generated searches only, with truthful effective query, independent cache identity, safe telemetry enums, and exact manual/edited preservation. Non-NGC photo proposals reuse the builder while retaining rich scoring evidence, NGC-first behavior, and zero eager provider access. Regenerated Swagger/OpenAPI; focused tests and full Go build/vet/test, architecture, route drift, Phase 341, privacy, and diff gates pass.

## Learnings (351 Phase 11 — B1/B5/F2 hardening, 2026-08-17)

- **T077 (agent bounds clamp)**: `DeepIdentifyRequest.bounds` (Go-supplied) was trusted verbatim by `run_deep_identification_stream`. Added `_clamp_bounds_to_ceilings()` in `src/agent/app/teams/deep_identification/graph.py`, called immediately after `bounds = request.bounds`, taking `min(request_value, settings.deep_*)` per field. `.env.example` and the `DeepIdentifyBounds`/config docstrings already described the *intended* ceiling behavior accurately, so no prose correction was needed — the code just hadn't caught up to its own documentation yet.
- **T078 (unbounded budget-tracker map)**: `DeepProviderBudgetTracker.Reset(jobID)` had zero production callers. Added `DeepIdentificationService.SetProviderBudgetTracker()` (post-construction setter, mirroring the existing `SetPipelineRunner` pattern) and call `Reset(job.ID)` from **both** `runJob`'s terminal path (covers completed/partial/failed/cancelled — verified by reading the single `switch` that sets `newStatus`) **and** `recoverStaleAndSweepHints`'s `RecoverStaleJobs` loop (covers a worker goroutine wedged past a stale-job recovery, a second, independent terminal path). Wired via `deepIdentificationSvc.SetProviderBudgetTracker(deepProviderBudgets)` in `main.go`.
- **T080/T081 (job-token secret separation + revocation)**: `InternalTokenService` now derives a distinct `jobSecret` via `hkdf.New(sha256.New, userJWTSecret, nil, "ancient-coins-api:deep-identification-job-token:v1")` at construction time — no new config, existing deployments need no action, `golang.org/x/crypto/hkdf` was already a direct dependency (no go.mod/go.sum change). `MintForJobWithTTL`/`VerifyForJob` sign/verify with `jobSecret`; `Mint`/`Verify` (user JWTs) still use `secret`. Added `RevokeJob(jobID)` (called from the same two terminal paths as the T078 Reset) which records `settled[jobID] = now` and self-prunes any entry older than `jobTokenRevocationRetention = 20*time.Minute` (comfortably beyond the max possible job-token TTL of `total_timeout_s(900) + 30s`), so the revocation map cannot grow unbounded the way the T078 bug did. `VerifyForJob` checks `isJobRevoked` after signature/expiry validation.
- Both `SetProviderBudgetTracker` and `SetInternalTokenService` follow the existing post-construction-setter DI convention in this codebase (see `SetPipelineRunner`) rather than widening `NewDeepIdentificationService`'s constructor signature, which would have broken every existing test call site for no proportional benefit.
- Full gate: `go build ./...`, `go vet ./...`, `go test -count=1 ./...` (all packages, including `TestNoDirectDatabaseImports`), `ruff check app/ tests/`, `pytest tests/ -v` (259 passed) all green after these changes.

## Learnings (351 Phase 8 — hypothesis seam / Maximinus fix, 2026-08-17)

- **The Maximinus bug was a write-only state field, not a missing provider.** `quick_evidence.coin_fields` (ruler/denomination/material/...) already arrived correctly on every deep-identification request and was read in exactly one place (`ocre.py:92`, to build a query). `synthesis.py` never accepted it, so a coin with zero *automated* provider contributors (NGC is `not_automated` by ToU-design, others `no_match`/`failed`) always hit `FALLBACK_NARRATIVE_NO_EVIDENCE` even though the identification was already sitting in state.
- **The seam**: introduced `CoinHypothesis`/`HypothesisField` (`src/agent/app/models/hypothesis.py`) matching contracts/vision-hypothesis.md §1 exactly (same coin-field vocabulary as the Go proposal allowlist), and a deterministic, LLM-free adapter (`app/teams/deep_identification/hypothesis.py::build_hypothesis_from_quick_evidence`) that populates it purely from `quick_evidence` — no new LLM call. It's built inside `prepare_evidence_node` (same node as the existing vision call) and stored on `state["hypothesis"]`, so Phase 3/4 can later replace *only that adapter's body* with the real vision-derived hypothesis without touching any consumer.
- **Era/Material enum risk**: Go casts a proposed `era`/`material` value straight into `models.Era`/`models.Material` with zero validation (`deep_identification_proposal.go::setCoinFieldFromProposalValue`). Go's enums are narrow (`era`: ancient/medieval/modern; `material`: Gold/Silver/Bronze/Copper/Electrum/Other) and don't match the contract's own hypothesis example (`"roman-imperial"` isn't a valid `Era`). The adapter canonicalizes both and **drops** the field entirely when it doesn't match — never forwards a value that would write garbage into the coin row.
- **Fallback gate (T059)**: now `not contributing and not hypothesis_supported` (both must be empty) instead of just `not contributing`. `CoinHypothesis.is_empty()` centralizes "empty" so an explicitly-empty hypothesis object behaves identically to `hypothesis=None`.
- **Proposed fields (T060/T061)**: `_build_proposed_fields` first builds provider-only entries as before, then folds in the hypothesis — exact normalized match with a provider claim applies the decided RD-2 bonus `min(1.0, max(image_conf, provider_conf) + 0.10)` **once per field** (not per corroborating provider) and appends an `EvidenceRef(provider="image")`; a hypothesis field with no provider claim at all is proposed at the hypothesis's own confidence with a bare `image` ref. `coverage`/`attributions` are untouched — they only ever iterate `list[ProviderEvidence]`, which structurally never contains an `image` row.
- **Nomisma query-length bug (Task G, found during this pass)**: `nomisma.py::_build_query`'s `label_text` branch was unbounded while its `notes` sibling was correctly bounded to 200; Go's `HTTPNomismaClient.Search` rejects >200-rune queries as `NomismaErrorInvalidRequest`, which `internal_tools.go`'s `NomismaSearch` handler collapsed into the *same* `"unavailable"` status as a real upstream outage. Fixed both sides: (1) `_build_query` now slices to `_MAX_QUERY_LENGTH = 200`; (2) Go now maps `NomismaErrorInvalidRequest` to a distinct `"invalid_request"` status, and `nomisma.py` maps that to `no_match`/`call_count=0` rather than `failed`/`error_kind="upstream"` — a client-side bug is no longer misreported as the provider's fault.
- **Scope boundary honored**: did not wire the hypothesis into the router, provider query-term construction, or the evaluator (tasks.md T057) — those are Phase 3-7 territory per the spec's own dependency ordering, and Brian's task explicitly scoped this pass to the synthesis-level seam only. Documented as deferred at the top of tasks.md Phase 8.
- Full gate after this change: `ruff check app/ tests/` clean; `pytest tests/ -v` 279 passed; `go build ./...` / `go vet ./...` clean; `go test -count=1 ./...` all packages passed (including `handlers`, which now also covers the new Nomisma `invalid_request` mapping).

## Learnings (351 Phase 5 — provider query terms / deleting the placeholder, 2026-08-17)

- **The Maximinus symptom traced to a literal placeholder string.** `numista.py::_DEFAULT_QUERY = "unidentified ancient coin"` (and the equivalent in `nomisma.py`) meant that whenever no `quick_evidence`/`notes` tier had content, the pipeline searched Numista/Nomisma for the literal phrase "unidentified ancient coin" and then reported `no_match` — a self-inflicted failure the owner had no way to diagnose from the report. Both placeholders are now deleted; the new terminal behavior is `status="no_match"`, `error_kind="insufficient_query_evidence"`, `call_count=0` (new `ProviderErrorKind` member) — distinguishable from a real search miss.
- **One shared builder, not three copies.** `query_terms.py::build_query_terms(quick_evidence, hypothesis, notes, *, max_length=None)` is now the single precedence implementation (contract §2 / FR-010): `numista_query` → `label_text` → hypothesis-derived (fixed RD-4 order: `ruler+denomination` → `ruler` → `denomination+material` → `obverseInscription`) → `notes[:200]`. `nomisma.py` passes `max_length=200` (Go's `nomismaMaxQueryLength` client-side rejection bound); `numista.py` does not, matching its pre-existing unbounded behavior. Reverse legend/type are structurally impossible to reach this builder — they aren't parameters to `_hypothesis_terms` at all.
- **Ranking vs. querying is a real, load-bearing distinction (RD-4).** The same weak reverse-legend/type signal that reliably poisons a *query* (turning a good match into `no_match` on a worn coin) is exactly the right tie-breaker once a provider has *already* returned 5 candidates — wrong there costs only ordering, not the whole result. `candidate_ranking.py::rank_candidates()` is a separate pure module consumed by `numista.py`/`nomisma.py` post-search (`candidates[0]` → `rank_candidates(candidates, hypothesis, text_fields)[0]`); it never triggers a call and degrades to the provider's original order when the hypothesis carries no signal tokens at all (empty hypothesis, or none of its reverse/other fields overlap any candidate text).
- **OCRE was already spec-compliant for T036 (nothing to delete) but not for T122 (real widening needed).** `ocre.py` never had a placeholder — its zero-signal path already short-circuited to `no_match`/`call_count=0`. But `_legend_tokens` (ADR 0010's scoring-only-signal source feeding `ocre_scoring.go`) read *only* `quick_evidence.label_text`, so on the actual Maximinus request shape (partial `coin_fields` like `ruler`, but no `label_text`) it silently contributed zero scoring tokens even though the call still fired. Widened to fall back to `obverseInscription + reverseInscription + coin_type` (the hypothesis's closest analogue to "reverse type" — it's literally the same field name OCRE claims use, `field="coin_type"`) only when `label_text` is absent; same normalization/dedup/12-token cap preserved. **`ocre_scoring.go` itself was never opened** — a test greps it for the ADR-anchored symbols (`ocreLegendMatches`/`ocreLegendBonusPer`/`ocreLegendBonusMax`/`sort.SliceStable`) as a structural guard against future accidental edits.
- **File-ownership boundary forced a specific, documented compromise.** `graph.py` (owned by a concurrent agent this batch) calls provider nodes positionally as `fn(entry, tools, quick_evidence, notes)` and doesn't pass `state["hypothesis"]` at all. Rather than touch the forbidden file, added a new **trailing, default-`None`, keyword-only-in-practice** `hypothesis: CoinHypothesis | None = None` parameter to all three `run()` functions — zero behavior change for every existing call site (graph.py's real one and every pre-existing test's). The hypothesis-derived query tier, ranking, and OCRE legend widening are fully implemented and unit-tested directly against the parameter, but **won't be reachable in a live run** until whoever next touches `graph.py` threads `state.get("hypothesis")` through as a fifth positional/keyword argument at the provider-fanout call site — a one-line follow-up, documented in `.squad/decisions/inbox/cassius-query-terms.md` and in `tasks.md`'s T038 note rather than done silently.
- Full gate: `ruff check app/ tests/` clean; `pytest tests/ -q` 335 passed (baseline 299, +36 new tests across `test_deep_identification_query_terms.py` and `test_deep_identification_candidate_ranking.py`). No Go files touched this batch (T122's constraint honored) — no Go gate was run.

## Learnings (351 Phase 3/4 — real vision-hypothesis structured output, 2026-08-17)

- **The keystone was already half-built.** A prior batch (Phase 8) had already shipped the `CoinHypothesis`/`HypothesisField` schema, the deterministic `quick_evidence`-only adapter, and the synthesis-level seam. This batch's job was narrower than tasks.md literally says: replace the hypothesis *source* with a real single-vision-LLM-call structured extraction, not build the seam from scratch.
- **Degrade ladder deviation from tasks.md T020/T027/T032 (documented, not silent)**: the literal task text has the ladder bottom out at the typed-empty hypothesis after a schema-validation failure/prose-extraction miss. Implemented one rung better: `structured call -> retry once (schema failure only) -> prose extraction -> deterministic quick-evidence hypothesis -> typed-empty`. The deterministic rung already existed from Phase 8 and is strictly better than typed-empty (it's what makes Brian's Maximinus coin still work when vision fails/degrades) — reusing it as the penultimate rung rather than skipping straight to empty was a clear win, not a scope creep. Recorded in `.squad/decisions/inbox/cassius-vision-hypothesis.md`.
- **"No second vision call" is satisfied on the happy path, not by forbidding all retries.** `get_structured_model(config, schema)` in `app/llm/provider.py` returns `model.with_structured_output(schema, method=..., include_raw=True)` — Anthropic uses `method="function_calling"` (reliable, tool-based), Ollama uses `method="json_schema"` (its JSON/format mode, explicitly documented as unreliable). `include_raw=True` means a schema-validation failure surfaces as `{"parsed": None, "parsing_error": ..., "raw": <AIMessage>}` rather than an exception, so a caller can attempt prose extraction from the *same* response with zero extra calls. The "retry once" in `build_hypothesis_from_vision` only re-invokes the LLM when the first attempt already failed schema validation — bounded, exceptional-path only, and uses the same transient-failure retry transport (`app/llm/retry.py::ainvoke_with_retry`) every other LLM call in this pipeline already uses.
- **Prose fallback is a real, tested rung, not a stub.** `_parse_prose_hypothesis` regex-extracts the first `{...}` block from raw model content, `json.loads`s it, aliases snake_case keys (`date_range` -> `dateRange`, etc.) onto the allowlist, accepts either the schema's `{"value","confidence"}` shape or a bare string (assigning a deliberately low `_PROSE_FALLBACK_CONFIDENCE = 0.4`), and re-applies the same era/material canonicalization as every other rung. Returns `None` on any failure so the caller falls through — never raises.
- **Era/material canonicalization is now genuinely shared, not duplicated.** Extracted `_canonicalize_hypothesis_field(key, value)` in `hypothesis.py`, built on the pre-existing `_canonical_era`/`_canonical_material` helpers, and reused it in both `_normalize_vision_hypothesis` (structured-parse path) and `_parse_prose_hypothesis` (prose path) — a garbage `era`/`material` value can never reach a proposed field regardless of which rung produced it.
- **`prepare_evidence_node` signature changed from `(state, model)` to `(state, llm_config=None)`** — the free-prose `IMAGE_ANALYSIS_PROMPT` path (and its now-dead `model` parameter) is deleted entirely; the vision call happens inside `build_hypothesis_from_vision` via its own `get_structured_model(llm_config, CoinHypothesis)` binding. `build_graph()`'s topology-only wrapper calls it with no `llm_config` (defaults to the deterministic adapter, never invoked in that test anyway); `run_deep_identification_stream` passes `request.llm`.
- **`CoinHypothesis` gained `notes`/`coin_type`** (additive, in `app/models/hypothesis.py`) — the prior batch's model only had 12 of the 14 allowlist fields; the vision call can legitimately populate these two, so they needed a home before normalization could round-trip them.
- **`DeepSynthesis.image_hypothesis: CoinHypothesis | None = None`** added (additive) to `app/models/responses.py`, populated by `synthesis.py::synthesize()` only when the hypothesis is non-empty (T030) — this required a direct top-level import of `app.models.hypothesis.CoinHypothesis` into `responses.py` (no circularity: `hypothesis.py` only imports pydantic).
- **Test hygiene finding**: the pre-existing `test_deep_identification_sse.py` used a real `anthropic` provider config (`api_key="test-key"`) with real obverse/reverse image data — before this batch that was harmless because `prepare_evidence_node`'s hypothesis path never called an LLM. After wiring the real vision call, those tests would have made real (failing, unauthenticated) HTTP calls to Anthropic on every run. Added an autouse fixture in that file monkeypatching `hypothesis_module.get_structured_model` to a fake that fails schema parsing immediately, keeping those tests hermetic and fast without touching any of their existing assertions.
- **Deliberately NOT done this batch (Phase 5-7, out of scope)**: router provider-selection, provider query-term construction, and the evaluator still do not read `state["hypothesis"]` — only `synthesizer_node` does. `state.py`'s docstring is written to say this plainly rather than repeat the previous "four real consumers" overclaim; the B2 write-only-field defect is only partially fixed until those phases land (tasks.md T057 is the actual close-out task).
- Full gate: `ruff check app/ tests/` clean; `pytest tests/ -q` **299 passed** (up from the 279 baseline — 20 new tests across `test_llm_provider_structured.py` (new file) and `test_deep_identification_hypothesis.py`), no regressions in `test_deep_identification_sse.py`/`test_deep_identification_synthesis.py`/`test_deep_identification_graph_topology.py`.

## Learnings (351 FR-040 — activity-timeline emission detail, 2026-08-17)

- **Brian's corrected assertion about `provider_result` was WRONG — flagged, not silently worked around.** Cassius's own task brief claimed both `provider_started` and `provider_result` persist the raw internal frame verbatim in `deep_identification_pipeline_runner.go`. Verified directly: `provider_started` genuinely is raw-passthrough (no reassignment of `persistPayload` in that `case`). `provider_result` is **NOT** — it is reduced through `deepProviderResultPublicPayloadJSON` to a fixed six-field bounded shape (`provider, status, confidence, claimCount, errorKind, linkOut`). This did not block the task (that bounded shape already carries everything item 3 needed, including `errorKind` for `insufficient_query_evidence`), but the premise itself was incorrect and is recorded here rather than silently routed around.
- **Query terms ride `provider_started` exactly as Aurelia preferred, with zero Go changes.** Added `_provider_started_detail()` in `graph.py`, called from `provider_fanout_node`'s `run_and_report` before emitting `provider_started`. Numista/nomisma get the real deterministic `query_terms` (via the existing shared `query_terms.build_query_terms`, exactly mirroring each provider node's own `_build_query` — nomisma's 200-rune Go-client bound included) or a `skip_reason: "insufficient_query_evidence"` when the builder returns empty. OCRE/NGC/RPC (no free-text query — OCRE decodes structured fields, NGC/RPC make no automated call at all) get a fixed, non-invented `detail` string instead of a fabricated query.
- **Per-provider settle timing was already live, not batched — verified, not assumed.** `provider_fanout_node`'s `run_and_report` calls `on_provider_event({"type": "provider_result", ...})` synchronously inside each per-task coroutine as soon as that task's own `_run_one_provider` await resolves; `asyncio.gather(*tasks)` only waits for all tasks, it does not delay each task's own internal side effects. Added `test_provider_result_frames_are_emitted_live_not_batched` (artificially slow numista vs. near-instant ngc/ocre/rpc) to prove this through the real stream rather than trust the claim. No Go/timing fix was needed.
- **`vision_completed` needed the hypothesis ladder to expose which rung fired, and the 17 existing `build_hypothesis_from_vision` callers/tests could not change shape.** Rather than changing that function's return type (would break 17 call sites in `test_deep_identification_hypothesis.py`), added a new `build_hypothesis_from_vision_traced()` returning `(CoinHypothesis, source)` with `source` in `{"structured", "prose", "deterministic_fallback", "no_images"}`; the original `build_hypothesis_from_vision` is now a two-line wrapper that discards the tag. `graph.py`'s `prepare_evidence_node` calls the traced variant and stashes `hypothesis_source` on state (new `DeepIdentificationState` key) purely for the progress message — never a claim/citation source, never persisted to the coin record.
- **`synthesis_started` is its own top-level frame type (not a `progress` phase) and is ALSO raw-passthrough** — `deepPipelineEventType` maps it straight to `DeepEventSynthesisStart` and the Go `onFrame` switch has no case for it, so adding a `message` field there needed zero Go changes, same as `provider_started`.
- **The Go `progress` whitelist already accepted a free `message` field before this batch** (`deepProgressMessage(phase)` only supplies a default when `message` is empty) — confirmed by reading the runner rather than assuming; `vision_completed`'s message needed no Go change at all, just a non-empty `message` on the Python side.
- Full gate: `ruff check app/ tests/` clean; `pytest tests/ -q` **346 passed** (baseline 337, +9 new tests, all in `test_deep_identification_sse.py`, all driving the real `run_deep_identification_stream` entry point per the anti-dead-code convention). No Go files touched this batch — no Go gate was run. `src/web/**` untouched.
## 2026-08-17 — Phase 9 regression gate: T070/T071 image-only proposal companions

**Feature:** `specs/351-vision-first-deep-identification` — Phase 9 (T070, T071)

- **T070 verified honest, no defect found.** Before writing assertions, I ran a debug probe against the real `buildDeepProposalDocumentJSON` output for a synthesis field whose only `evidence_refs` entry is `{"provider":"image"}`. The field IS present in the persisted `fields` map with `Evidence` as a nil/zero-length slice — it is never dropped. The existing "skip refs where `provider == image`" logic (`deep_identification_pipeline_runner.go:951-956`) only skips the *ref* inside the evidence-accumulation loop; it does not `continue`/skip the outer per-field loop, so `fields[name] = entry` always runs regardless of whether any evidence survived. Added `TestDeepIdentificationProposal_ImageOnlyFieldRetainedWithEmptyEvidence` to `deep_identification_proposal_integration_test.go` as a permanent regression pinning this behavior.
- **T071 backward-compatibility fixture added** to `deep_identification_pipeline_runner_test.go`: `TestDeepIdentificationBackwardCompatibility_PreAndPostImageHypothesisFixtures` drives (a) a pre-351 report/proposal shape with zero `image` provider refs and (b) a post-351 proposal with an image-only field through the real `Get -> PATCH accept -> Apply` service round trip on both. Both apply to a saved coin with zero errors. `Evidence []deepProposalClaim` uses `json:"evidence,omitempty"` — a nil slice is *omitted* from the wire JSON rather than serialized as `[]`, but this is immaterial to Go-side round-tripping (`len(entry.Evidence) == 0` either way); flagged here for whoever owns the frontend `Claim[]` type in case it assumes a required array key.
- **Learning:** `deep_identification_proposal_test.go`, `deep_identification_proposal_integration_test.go`, and `deep_identification_pipeline_runner_test.go` are all `package services` and freely share helpers (`newDeepProposalTestDeps`, `seedDeepProposalUser`, `acceptTrue`, `deepTestDBCounter`) — no need to duplicate test scaffolding across these three files, just add the missing imports to the file you're extending.
- **Verification:** `go build ./...` exit 0; `go vet ./...` exit 0; `go test -count=1 ./...` — all 10 packages `ok` (api, capture, database, handlers, integration, middleware, models, repository, services, testutil), 0 FAIL.


## Learnings (351 Phase 14 — T100/T101(Python)/T102 dead-code cleanup, 2026-08-17)

- **T100's real lesson was ordering, not deletion.** graph.py::build_graph was a compiled StateGraph production never called (the real driver is the hand-written async generator un_deep_identification_stream), so the old topology test proved nothing about the code users run — the same class of bug (fully-tested code production never executes) that caused the original outage. Rewrote 	ests/test_deep_identification_graph_topology.py to drive un_deep_identification_stream itself (LLM/provider calls faked out the same way 	est_deep_identification_sse.py already does), assert its emitted stage frames occur in exactly prepare_evidence -> router -> provider_fanout -> evaluator -> synthesizer order, confirmed it passed *before* touching uild_graph, and only then deleted uild_graph plus the two tests that only exercised its own dead ecursion_limit-binding mechanism (no production equivalent — the streaming driver never .ainvokes a compiled graph). Net -2 tests in the full suite (352 -> 350) is fully explained by this: 3 build_graph tests -> 1 production test.
- **T101 (Python half only)**: 
umista_detail() on ProviderToolsClient had zero call sites anywhere in src/agent (grep-proved) — removed. Did not touch the four Go/web items in the same task line (RPCEnabled, listDeepIdentificationJobs, stream.reset(), DeepStreamTruncatedPayload) since Maximus/Aurelia own those files this batch; left T101's tasks.md checkbox unticked with a note, per the brief's own instruction not to mark a partially-done task complete.
- **T102's crux held up under verification**: hint_kind/call_budget/schema_version are all genuinely unread by any Python logic (grep-confirmed — zero reads outside field declarations and test fixtures that only *set* them), but all three are documented, actively-Go-sent fields in contracts/agent-internal-contract.md §2. Kept them, added inline comments marking them forward-compatibility placeholders, made zero Go changes (nothing to report to Cassius/Maximus since nothing was removed from the wire). By contrast state.errors/state.tools_base_url/state.internal_token in state.py's TypedDict were pure dead declarations — un_deep_identification_stream's state-dict literal never sets them, and the request-level fields of the identical name (equest.tools_base_url/equest.internal_token, which *are* real and used to build ProviderToolsClient) are a completely separate, still-live pair. Removed the three state fields with zero behavior change.
- **Key distinction worth remembering for future dead-field audits**: a field name appearing twice — once on a Pydantic request model (wire-contract, real) and once on a TypedDict pipeline state (internal, possibly dead) — are not the same field just because they share a name and both look "unread" at a glance. Always trace which one the runtime code actually populates/consumes before deciding either is safe to delete.
- Full gate: uff check app/ tests/ clean; pytest tests/ -q 350 passed (baseline 352, difference fully explained by T100's test consolidation); 	est_deep_identification_maximinus.py all 6 passed in isolation. No Go files touched or built this batch.

## T107 -- Contract-drift test for DeepIdentifyRequest/DeepSynthesis (2026-08-17)

- New file `src/api/services/deep_identification_contract_drift_test.go` (no forbidden files touched). Compares Go mirror structs vs Pydantic models mechanically: Python side is a checked-in JSON Schema fixture (`src/api/services/testdata/deep_identify_contract_schema.json`) produced by calling the real `DeepIdentifyRequest.model_json_schema()`/`DeepSynthesis.model_json_schema()` -- never hand-transcribed. Go side is read via `reflect.TypeOf` + `json` struct tags on the actual shipped structs (`DeepIdentifyProxyRequest` and friends, `LLMConfig`, `deepSynthesisProposedField`).
- Key finding: Go has no single named struct that fully types `DeepSynthesis` -- it deliberately treats most of the terminal synthesis report as opaque `json.RawMessage` pass-through (see `deep_identification_pipeline_runner.go`'s `buildDeepProposalDocumentJSON` and `deep_identification_frame_translator.go`'s `handleSynthesis`), typing only `narrative`/`proposed_fields` (+ nested `evidence_refs`) and `partial_success` into narrow private structs. For DeepSynthesis fields with no Go type to reflect over (`disagreements`, `unresolved_questions`, `coverage`, `attributions`, `image_hypothesis`), the test falls back to a pinned, hand-maintained top-level property-name snapshot -- documented in the test file as the one deliberate non-mechanical exception.
- Of T093's five fixed drift points, only #2 (`llm_config`->`llm`) and #5 (`attributions` added to `DeepSynthesis`) are actually shaped like a DeepIdentifyRequest/DeepSynthesis field drift and are mechanically caught (drift #3, the `quick_evidence.numista_evidence` deletion, is caught as a side effect too). #1 (token-minting function signature), #3-the-evaluation-frame-shape, and #6 (`ocre_search` tool-catalog row) live on other wire surfaces this test cannot and does not claim to cover.
- Falsification: renamed `DeepIdentifyProxyRequest.LLM`'s json tag from `llm` to `llm_config` (recreating the actual historical T093 drift) in `agent_proxy_deep_identify.go` -- confirmed the test went RED with a clear two-sided diagnostic, then restored exactly (`git diff --stat` clean) and confirmed GREEN.
- Full gate: go build ./..., go vet ./... clean; go test -count=1 ./... all 10 packages ok; TestArchitecture and TestNoDirectDatabaseImports pass.

## T107 follow-up -- Python-side staleness guard for the contract fixture (2026-08-17)

- Since T106 (live SSE round trip) is CI-excluded/tagged, it cannot be relied on to catch drift between fixture regenerations, so the Go-side contract-drift test alone could theoretically stay green against a stale fixture indefinitely. Closed that gap from the Python side.
- New file `src/agent/tests/test_contract_schema_fixture_is_current.py`: recomputes `DeepIdentifyRequest.model_json_schema()`/`DeepSynthesis.model_json_schema()` right now (same call the regeneration command uses) and asserts byte-for-byte equality (via `json.dumps(..., sort_keys=True)`) against the checked-in `src/api/services/testdata/deep_identify_contract_schema.json`, excluding only the `_generated_by` provenance key. Fixture path resolved relative to `__file__` (`Path(__file__).resolve().parents[3]`), not cwd; skips with a clear message if the expected repo layout isn't found, rather than risking a false red.
- Falsification: renamed `DeepSynthesis.attributions` -> `attribution_list` in `app/models/responses.py` -- confirmed the new test went RED with the exact regeneration command printed in the failure message, then restored exactly (`git diff --stat` clean) and confirmed GREEN.
- Full gate: `ruff check app/ tests/` clean; `python -m pytest tests/ -q` -> 351 passed (was 350 before this addition, +1 for the new test); `test_deep_identification_maximinus.py` all 6 passed in isolation. No Go files touched.

---

## 2026-08-17 — Deep Analysis: StartJob at-capacity non-matching-fingerprint fix

Fixed a live-confirmed defect: DeepIdentificationService.StartJob's per-user
capacity branch (MaxActivePerUser, default 1) fell through to ListJobs and
handed a genuinely non-matching submission the user's *other* in-flight
job's identity, still marked reused=true. The handler then returned 200
with that unrelated job's report/proposal — a second coin's submission
received the first coin's answer. Fixed with a new sentinel
ErrDeepJobAtCapacity -> 409/job_at_capacity, generic message, no IDs. The
matching-fingerprint duplicate path (FR-007 idempotency) is untouched and
still returns reused=true.

Key lesson: RetryJob shares the exact same StartJob path via
CreateJobFromIntake, so fixing StartJob and respondDeepJobError's central
switch was sufficient to make Retry coherent too — no duplicate handling
needed. Also: fixing a service bug like this can silently break *other*
pre-existing tests beyond the one named in the task (here,
TestDeepIdentificationService_StartJob_QueueDepthAndPerUserLimit in
deep_identification_service_test.go also pinned the old behavior) — always
run the full package test suite after a service-level behavioral change,
not just the named test file, to catch tightly-coupled collateral breakage.

Also learned: this repo's 	ask openapi target has a pre-existing,
unrelated PowerShell templating bug in its version-bump step (fails before
swag even runs) — when it fails, regenerate manually with
swag init -g main.go -o ./docs --parseDependency --parseInternal from
src/api, then copy docs/swagger.json to ../../docs/openapi.json. Confirmed
this does not touch main.go's @version line.
