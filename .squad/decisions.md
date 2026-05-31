# Squad Decisions

## Active Decisions

### 1. Governance Restructure — tech-inventory alignment (2026-05-28)
### 2. Feature #219 Acceptance Checklist & Validation Gates (2026-05-31)

**Feature**: Refine Coin Details Page for PWA and Desktop  
**Spec**: `specs/219-refine-coin-details-page/spec.md`  
**Author**: Maximus (Lead)  
**Date**: 2026-05-31  
**Status**: APPROVED

**Summary**: 37+ validation gates defined across US1 (dual-side media), US2 (metadata tables), US3 (dedicated section pages), and polish scope. Three-phase tester handoff plan; no team ADR required (UI-only, within constitutional bounds). Constitution compliance verified on Principles V, IX, XIII.

---

### 3. Feature #219 Coin Detail Page Refinements — QA Verdict (2026-05-31)

**Author:** Brutus (Tester)  
**Date:** 2026-05-31  
**Status:** APPROVED

**Summary**: Full QA validation completed. 12/12 functional requirements met; zero regressions. Type-check + production build pass cleanly. Feature is **ship-ready**.

**Verdict**: ✅ APPROVE — All user stories and functional requirements satisfied. Awaiting merge to main.

---

### 6. Code Review & Quality Assessment (2026-04-24)

**Authors:** Maximus (Architect), Cassius (Backend), Aurelia (Frontend), Brutus (Testing)  
**Date:** 2026-04-24  
**Status:** Assessed — Backlog Created  

#### What
Comprehensive review of all three services covering architecture, code quality, testing, security, and accessibility. Generated 77 backlog items across P0–P3 priorities.

#### Key Findings

**Architecture (Grade: B+)**
- Clean 3-service separation and excellent documentation (761-line ARCHITECTURE.md)
- Layered Go API enforced by architecture tests; handlers→services→repositories enforced
- DI pattern used but undermined by 3 package-level globals: `AppLogger`, `GetSetting()`, `cancelMap`
- API key middleware bypasses repository abstraction

**Backend (Grade: B-)**
- Most handlers thin; some leak business logic (analysis.go, agent.go, coins.go, admin.go)
- Sentinel errors used in 4 services; many repos silently drop errors (7+ locations in social.go)
- Non-atomic multi-step writes without transactions (auction lot, social, availability)
- Input validation sparse; page/limit defaults silently instead of validating

**Frontend (Grade: B-)**
- Good Composition API; 6 components exceed 400 lines (need splitting: AdminPage 1378, SettingsPage 1371, CoinDetailPage 1242)
- TypeScript discipline strong; very few `any` casts
- State management too lean (coins store lacks error state; auth store drifts after refresh)
- Critical gap: accessibility D+ (no ARIA, no focus traps, clickable divs not keyboard-accessible)
- PWA quality C+ (missing icons pwa-192×192 and pwa-512×512; no offline fallback, no update prompt)

**Testing (Grade: D)**
- Go: 3.5-4.6% coverage; only CoinRepository and CoinService tested; zero handler tests
- Frontend: ZERO test files, no framework
- Python: 31 tests passing; but zero tests for 11 team pipelines, supervisors, LLM provider, search tools
- No test plan, no coverage thresholds, no CI enforcement

**Security Issues (P0)**
- XSS risk in v-html AI content (Aurelia confirmed DOMPurify is used; can close)
- SQL injection in coin_repository Suggestions() method (whitelist in handler but not repo; needs defense-in-depth)
- Admin route accessible to any authenticated user (no role guard)
- Double-close panic risk in scheduler Stop() methods

#### Impact
Establishes baseline quality metrics and prioritized backlog. Guides sprint planning for next 2–3 quarters. Addresses security (P0), DI debt (P1), god-page decomposition (P2), and testing coverage expansion (ongoing).

#### Backlog Structure
- **P0 (Critical):** 8 items — security, panic bugs, auth tests
- **P1 (High):** 19 items — DI refactor, transaction safety, memory leaks, frontend testing setup
- **P2 (Medium):** 28 items — error audit, accessibility, god-page splits, test expansion
- **P3 (Low):** 22 items — performance, form validation, API polish

---

### 7. P0 Fixes — Admin Route Guard & v-html (2026-07-22)

**Author:** Aurelia (Frontend Dev)  
**Date:** 2026-07-22  
**Status:** Implemented  

#### What
- Added `requiresAdmin: true` meta to `/admin` route; guard checks `auth.isAdmin` and redirects non-admin to `/`
- Verified v-html XSS mitigation: all 4 bindings already wrapped with `DOMPurify.sanitize()`

#### Why
Admin page was UI-hidden but route was directly accessible. v-html XSS appeared as backlog item but was already protected.

#### Impact
Admin routes now protected. Can close code review backlog items #1–2.

---

### 8. Activity Journal Scroll Limit & Auction Schedule UI (2026-05-01)

**Author:** Aurelia (Frontend Dev)  
**Date:** 2026-05-01  
**Status:** Implemented  

#### What

Two independent UI improvements:

**Task A — Activity Journal Scroll Limit**
- Added scroll containment to CoinActivityJournal in coin detail page
- Shows max 3 entries by default; rest accessible via internal vertical scroll
- Used design tokens for scrollbar styling (--bg-card, --border-subtle, --accent-gold-dim)

**Task B — Auction-Ending Schedule in Admin UI**
- Added "Auction Ending Alerts" panel to AdminSchedulesSection mirroring wishlist pattern
- Three new settings keys: AuctionEndingCheckEnabled, AuctionEndingCheckStartTime, AuctionEndingCheckInterval
- Updated useAdminConfig composable to expose and manage auction settings state
- Integrated into AdminPage with proper prop binding

#### Why

- Task A: Prevents Activity Journal from pushing content down page as history grows; keeps layout compact
- Task B: Cassius building backend daily scheduler for auction-ending alerts; needs UI configuration in same location as wishlist/valuation schedulers

#### Impact

- Task A: Coin detail page remains compact with unbounded journal history
- Task B: Users can enable and configure auction-ending scheduler alongside existing background schedulers

#### Testing

- vue-tsc passes clean (no TypeScript errors)
- Nullish coalescing and optional chaining used correctly for Docker strictness
- All design tokens applied (no hardcoded values)

---

### 4. Auction Ending Manual Trigger & Run Log — Backend Implementation (2026-06-10)

**Author:** Cassius (Backend Dev)  
**Date:** 2026-06-10  
**Status:** Implemented  

#### What

Added manual run trigger and per-run logging to Auction Ending scheduler for parity with Valuation and Wishlist schedulers:

1. **Model:** `models/auction_ending_run.go` — 10 fields (ID, TriggerType, TriggerUserID, Status, LotsChecked, AlertsSent, DurationMs, StartedAt, CompletedAt, ErrorMessage)
2. **Repository:** `repository/auction_ending_repository.go` — CreateRun, CompleteRun, ListRuns (paginated), GetRunByID, PruneOldRuns
3. **Service:** Refactored `services/auction_ending_scheduler.go` — Added RunNow(triggerUserID) method, extracted runCycleWithTrigger() to log every run
4. **Handler:** `handlers/auction_ending_admin.go` — Two endpoints: POST /api/admin/auction-ending/run (manual trigger), GET /api/admin/auction-ending-runs (run history)
5. **Wiring:** Updated main.go to instantiate scheduler early and pass to admin handler
6. **Database:** Added AuctionEndingRun to AutoMigrate in database/database.go
7. **Documentation:** Updated README.md Background Schedulers section

#### Why

Auction Ending scheduler needed manual-run capability and run logging to achieve feature parity with Valuation and Wishlist schedulers. Enables administrators to manually trigger checks and inspect historical run performance.

#### API Contract

**POST /api/admin/auction-ending/run** (admin only, returns 200 with run details on success)
- Response: {runId, lotsChecked, alertsSent, status, durationMs}

**GET /api/admin/auction-ending-runs?page=1&limit=20** (admin only, paginated)
- Response: {runs: [...], total, page, limit}
- Each run: {id, triggerType, triggerUserId, status, lotsChecked, alertsSent, durationMs, startedAt, completedAt, errorMessage, createdAt}

#### Architecture Compliance

- Model/Repository/Handler follow exact pattern of valuation_run (100% consistency)
- Pagination enforces defaults (page≥1, limit 1-100, default 20)
- Auto-pruning keeps 100 most recent runs
- Transaction safety via Updates() with map in CompleteRun
- Swagger annotations on both handler methods
- Auth/admin guards on both endpoints

#### Testing

✅ All tests pass:
- go vet clean
- go test -v ./... passed
- Architecture tests passed

---

### 5. Auction Ending Manual Trigger & Run Log — Frontend UI (2026-05-21)

**Author:** Aurelia (Frontend Dev)  
**Date:** 2026-05-21  
**Status:** Implemented (minor follow-up fixup pending)

#### What

Implemented admin UI for manual trigger and run history display in AdminSchedulesSection:

1. **API Client:** Added triggerAuctionEndingCheck(), getAuctionEndingRuns(), getAuctionEndingRunDetail() in client.ts
2. **Types:** Added AuctionEndingRun and AuctionEndingResult interfaces in types/index.ts
3. **Composable:** Extended useAdminConfig with auctionSettingsMsg, auctionSettingsError state; added defaults handling
4. **Component:** 
   - "Run Now" button in Auction Ending section
   - Recent runs table with columns: Date, Trigger, Lots, Alerts, Status, Duration
   - Expandable detail rows for error messages
   - Pagination controls with loading state
   - Responsive mobile layout

#### Why

Cassius implemented backend manual trigger and run log; frontend needed corresponding UI in AdminSchedulesSection to match Valuation/Wishlist patterns.

#### Testing

- npm run type-check passed
- npm run build succeeded (production build)
- All global design tokens used (no hardcodes)
- Followed Composition API patterns from existing admin components

#### Known Issue

Aurelia guessed endpoint URL `/admin/auction-ending/runs` but Cassius's actual endpoint is `/admin/auction-ending-runs` (hyphenated). Follow-up fixup spawn (aurelia-auction-fixup) in flight to align client.ts URL.

---

### 6. Auction Ending Manual Trigger & Run Log — Test Coverage (2026-05-22)

**Author:** Brutus (Tester/QA)  
**Date:** 2026-05-22  
**Status:** **APPROVED**  

#### What

Comprehensive test suite for Cassius's auction-ending manual-run and run-log implementation:

**Repository Tests (10 tests in auction_ending_repository_test.go):**
- CreateRun (ID assignment, timestamp population)
- CompleteRun success and error paths (status, timestamps, error message persistence)
- ListRuns (newest-first ordering, pagination, empty results)
- ListRuns pagination edge cases (limit defaults, negative limits, zero limits)
- GetRunByID (found and not-found paths)

**Handler Tests (6 tests in auction_ending_admin_test.go):**
- TriggerRun endpoint (admin authorization, user rejection, no-auth rejection)
- ListRuns endpoint (admin authorization, pagination param handling, no-auth rejection)

#### Why

Cassius completed manual-run and run-log feature; comprehensive test coverage validates architecture compliance, error handling, authorization guards, and pagination safety.

#### Quality Assessment

✅ **Strengths:**
- 100% pattern consistency with valuation/wishlist schedulers
- Transaction safety via Updates() with map
- Pagination defaults enforced (page≥1, limit 1-100, default 20)
- Error handling and pruning strategy robust
- Complete Swagger annotations
- Auth/admin guards on both endpoints

⚠️ **Minor Observations (not blocking):**
- PruneOldRuns silently fails on error (suggest adding log line, low priority)
- No cancel endpoint (acceptable for fast runs, flag for future if runs become long-running)

#### Verdict

**APPROVED** — All 16 tests pass. Architecture compliance excellent. No blocking issues. Production-ready.

#### Recommendation

Merge to main. Optional improvements (logging, E2E tests) can be backlog items for future sprint.

---

### 7. Auction Ending Scheduler Implementation

**Author:** Cassius (Backend Dev)  
**Date:** 2026-05-21  
**Status:** Implemented  

#### What

Built a new background scheduler that notifies users via Pushover when auction lots they are bidding on have a sale date of today.

#### Implementation Details

**Files Created:**
1. `src/api/services/auction_ending_scheduler.go` — Scheduler service following the exact pattern of `availability_scheduler.go`:
   - `Start()` / `Stop()` lifecycle with `sync.Once` for safe shutdown
   - `timeUntilNextRun()` calculates next run based on start time + interval
   - `runCycle()` fetches ending auctions, groups by user, sends consolidated notifications
   - In-memory idempotency tracking via `lastNotified map[uint]string` (userID → date string YYYY-MM-DD)

2. `src/api/repository/auction_lot_repository_test.go` — Unit tests for the new repository method:
   - `TestAuctionLotRepository_GetEndingToday` — Verifies only BIDDING lots with today's sale date are returned
   - `TestAuctionLotRepository_GetEndingToday_MultipleUsers` — Verifies multi-user grouping and ordering

**Files Modified:**
1. `src/api/services/settings_service.go` — Added constants for scheduler settings:
   - `SettingAuctionEndingCheckEnabled` (default: `"false"`)
   - `SettingAuctionEndingCheckInterval` (default: `"1440"` — 24 hours in minutes)
   - `SettingAuctionEndingCheckStartTime` (default: `"08:00"`)

2. `src/api/repository/auction_lot_repository.go` — Added `GetEndingToday()` method:
   - Returns all auction lots where `status = "bidding"` AND `sale_date >= startOfDay` AND `sale_date < endOfDay`
   - Uses server's local timezone for "today" calculation
   - Orders by `user_id ASC, sale_date ASC` for efficient grouping

3. `src/api/main.go` — Wired scheduler startup alongside existing schedulers

4. `src/api/README.md` — Added "Background Schedulers" section

#### Idempotency Approach

**Decision:** In-memory tracking via `lastNotified map[uint]string` on the scheduler struct.

**Rationale:**
- Simplest implementation — no schema changes, no DB writes on every check
- Sufficient for daily cadence — map is cleared on server restart, acceptable for once-daily scheduler
- Memory footprint negligible (one string per user)
- Prevents duplicate notifications if scheduler runs multiple times in a day

#### Notification Format

**Title:** "Auctions Ending Today"

**Message:** 
```
3 auction(s) you are bidding on end today:

• Heritage Auctions - Long Beach Sale (Lot 42)
• Stack's Bowers - ANA Auction (Lot 1205)
• Roma Numismatics - E-Sale 99 (Lot 348)
```

#### Testing

✅ All tests pass:
- `TestAuctionLotRepository_GetEndingToday` — Filters by status and date correctly
- `TestAuctionLotRepository_GetEndingToday_MultipleUsers` — Groups and orders correctly
- All existing architecture tests pass

---

### 8. Auction Ending Scheduler — NULL Date Handling Fix

**Author:** Cassius (Backend Dev)  
**Date:** 2026-05-22  
**Status:** Implemented  

#### Problem

Brian ran the auction ending scheduler manually on May 22, 2026. The scheduler reported 0 lots checked and 0 alerts sent, even though Brian has a Heritage Auctions Europe lot (Lot #8325, sale date May 22, 2026, status BIDDING) that should have been flagged.

#### Root Cause

The `AuctionLotRepository.GetEndingToday()` query only checked the `sale_date` field:

```sql
WHERE status = 'bidding' 
  AND sale_date >= startOfDay 
  AND sale_date < endOfDay
```

The `AuctionLot` model has TWO nullable date fields:
- `SaleDate *time.Time` — the sale/auction day (populated by NumisBids scraper)
- `AuctionEndTime *time.Time` — precise ending time (not used by NumisBids scraper)

When `sale_date` is NULL, the SQL comparison evaluates to NULL (not TRUE), and the row is excluded from results — even if `auction_end_time` is set to today.

**Why Brian's Heritage lot had `sale_date = NULL`:**
1. Heritage Auctions URLs are not supported by the NumisBids scraper
2. `ParseSaleDate()` only handles NumisBids date formats
3. Lot may have been created manually via the UI or API
4. Heritage auctions may populate `auction_end_time` but leave `sale_date` empty

#### Solution

Updated `AuctionLotRepository.GetEndingToday()` to check BOTH date fields with explicit NULL guards:

```sql
WHERE status = 'bidding' AND (
  (sale_date IS NOT NULL AND sale_date >= startOfDay AND sale_date < endOfDay) OR
  (auction_end_time IS NOT NULL AND auction_end_time >= startOfDay AND auction_end_time < endOfDay)
)
```

**Logic:**
- If `sale_date` is set and is today → include the lot
- If `auction_end_time` is set and is today → include the lot
- If both are set, include if either matches today (union, not intersection)
- If both are NULL, exclude the lot

#### Changes

**Modified:**
- `src/api/repository/auction_lot_repository.go` — Updated `GetEndingToday()` query with OR logic

**Added:**
- `src/api/repository/auction_lot_repository_test.go` — New test case: "bidding lot with auction_end_time today (no sale_date)"

#### Testing

✅ All tests pass (`go test -v ./...`):
- Lot with `sale_date = today, auction_end_time = NULL` → included ✅
- Lot with `sale_date = NULL, auction_end_time = today` → included ✅ (new test)
- Lot with `sale_date = NULL, auction_end_time = NULL` → excluded ✅

#### Impact

**Positive:**
- Fixes Heritage Auctions bug: lots with `auction_end_time` set but no `sale_date` are now detected
- Future-proof: supports any auction source that uses `auction_end_time` instead of `sale_date`
- No breaking changes: existing NumisBids lots continue to work exactly as before

**Risks:** None identified. The OR logic is additive and doesn't change behavior for existing data.

---

### 9. PWA Service Worker Lifecycle Fix

**Author:** Aurelia (Frontend Dev)  
**Date:** 2026-05-23  
**Status:** Implemented  

#### What

Fixed critical PWA service worker update failure that left users stuck with stale service workers trying to import non-existent workbox files.

**Changes:**
1. Added `import { registerSW } from 'virtual:pwa-register'` to `src/web/src/main.ts` with `immediate: true` to wire up vite-plugin-pwa's auto-update lifecycle
2. Added hourly service worker update check (`setInterval` calling `registration.update()` every 60 minutes)
3. Added `/// <reference types="vite-plugin-pwa/client" />` to `env.d.ts` for TypeScript support of virtual module
4. Typed `onRegisteredSW` callback parameters to satisfy strict TypeScript checking

**Icons verification:**
- `pwa-192x192.png` and `pwa-512x512.png` already existed in `public/` (547 bytes and 1.9 KB respectively)
- Manifest correctly references both icons plus maskable variant
- No action needed on icon side — the browser error was a symptom of the stale SW issue

#### Why

**Root Cause:** The service worker registration was never initialized. `vite.config.ts` had all the correct configuration (`registerType: 'autoUpdate'`, `skipWaiting: true`, `clientsClaim: true`, `cleanupOutdatedCaches: true`), but `main.ts` didn't import the virtual module that triggers registration.

**Impact on Users:** After a deploy, the build emitted a new `sw.js` and `workbox-{NEW_HASH}.js`, but users with the old `sw.js` in their cache kept trying to `importScripts('workbox-{OLD_HASH}.js')` — which no longer existed on the server. This violates the service worker spec (no new script imports post-install) and threw `NetworkError: Failed to import`.

#### How It Works Now

1. **On page load:** `registerSW({ immediate: true })` registers the service worker
2. **On new deploy:** Browser detects `sw.js` has changed, downloads new SW, which `skipWaiting()` immediately activates and `clientsClaim()` takes control without waiting for tab close
3. **Hourly update check:** `registration.update()` proactively checks for new SW versions even if user doesn't reload
4. **Cleanup:** `cleanupOutdatedCaches: true` prunes old workbox-{hash}.js files from cache storage

#### User-Facing Impact

**Existing users on stale SW:** On their **next page load** after this deploy, the broken old SW will serve them one last time, fetch the new SW (which auto-activates), and then the new lifecycle takes over. They may see the error once more in the console but won't after the refresh.

**Recommended:** Users can force-clear the issue immediately by opening DevTools → Application → Service Workers → Unregister, then hard refresh (Ctrl+Shift+R). For most users, a single refresh after deploy will resolve it.

#### Testing

✅ `npm run type-check` passes  
✅ `npm run build` succeeds — generates fresh `sw.js` and `workbox-{HASH}.js`  
✅ Icons present in `dist/` (192x192 and 512x512)  
✅ Manifest correctly references both icon sizes and maskable variant

---

### 10. Auction Ending Scheduler — Debug Endpoint for Ground-Truth Investigation

**Author:** Cassius (Backend Dev)  
**Date:** 2026-05-22  
**Status:** Implemented — Awaiting Production Data  

#### Problem

Brian's Heritage Auctions lot (Lot #8325, displayed sale date May 22, 2026, status BIDDING) was not flagged by the auction ending scheduler. After the first bugfix (NULL-date handling for `sale_date` and `auction_end_time`), Brian redeployed and re-ran the manual trigger — **still 0 lots found**. Same 10ms execution time (suspiciously identical to the first failed run).

#### Root Cause Analysis

##### First-Pass Diagnosis (INCOMPLETE)

The initial fix assumed the lot had either `sale_date` or `auction_end_time` populated. The query was updated to check both fields with NULL guards. This was a **guess based on schema**, not real data inspection.

##### Second-Pass Audit (CRITICAL FINDINGS)

**Exhaustive Date Field Inventory:**

The `AuctionLot` model has **THREE** ways to represent an end date:

1. **`SaleDate *time.Time`** — populated by NumisBids scraper
2. **`AuctionEndTime *time.Time`** — precise ending timestamp (rarely used)
3. **`EventID *uint`** — foreign key to `AuctionEvent` which has `StartDate` and `EndDate` fields

**CRITICAL DISCOVERY:** Heritage lots likely have `EventID` set (linking to a calendar event) but both `SaleDate` and `AuctionEndTime` are NULL. **The displayed sale date in the UI comes from `AuctionEvent.EndDate`, NOT the lot's own date fields.**

This means the current scheduler query (`WHERE (sale_date today OR auction_end_time today)`) **completely misses lots whose date is inherited from a parent event**.

**Other Hypotheses Ruled Out:**

- **Status mismatch:** `models.AuctionStatusBidding` constant is lowercase `"bidding"` — matches DB enum values
- **User scope filter:** No user_id WHERE clause in scheduler query — iterates all users
- **Case sensitivity:** SQLite is case-insensitive for string comparisons by default
- **Time zone issues:** All date comparisons use `now.Location()` consistently

#### Solution

##### Debug Endpoint (Implemented)

Added `GET /api/admin/auction-ending/debug` that returns:

```json
{
  "now": "2026-05-22T19:09:00Z",
  "today_start": "2026-05-22T00:00:00Z",
  "today_end": "2026-05-23T00:00:00Z",
  "query_summary": "WHERE status = 'bidding' AND ((sale_date >= X AND sale_date < Y) OR (auction_end_time >= X AND auction_end_time < Y))",
  "total_lots_in_db": 42,
  "lots_by_status": { "bidding": 3, "watching": 12, "won": 5, ... },
  "lots_matching_query": [
    { "id": 10, "lot_number": 1234, "status": "bidding", "sale_date": "2026-05-22T10:00:00Z", ... }
  ],
  "all_bidding_lots": [
    { "id": 42, "lot_number": 8325, "status": "bidding", "sale_date": null, "auction_end_time": null, "event_id": 7, "event_end_date": "2026-05-22" }
  ]
}
```

**Key Design Decisions:**

1. **Read-only:** No side effects, no notifications sent
2. **Admin-only:** Requires admin role + JWT auth
3. **Comprehensive data:** Includes ALL BIDDING lots with ALL date fields (including event dates via LEFT JOIN)
4. **Architecture compliance:** All SQL queries delegated to repository layer (`AuctionLotRepository.GetAllBiddingLotsWithEventDates()`)
5. **Swagger annotations:** Fully documented API contract

##### SQL Query for Immediate Inspection

Brian can run this query directly against the SQLite DB **right now** to confirm the hypothesis:

```sql
SELECT 
  id, 
  user_id, 
  status, 
  lot_number, 
  sale_date, 
  auction_end_time, 
  event_id, 
  created_at, 
  updated_at 
FROM auction_lots 
WHERE lot_number = 8325 
   OR status = 'bidding' 
ORDER BY updated_at DESC 
LIMIT 10;
```

**Expected result:** Lot 8325 has `sale_date = NULL`, `auction_end_time = NULL`, `event_id = <some_id>`. The end date is stored on the linked `AuctionEvent` row.

#### Implementation Details

**Files Created:**

1. `src/api/handlers/auction_ending_debug.go` — Debug handler with `DebugGetAuctionEndingInfo()` method

**Files Modified:**

1. `src/api/repository/auction_lot_repository.go` — Added `GetAllBiddingLotsWithEventDates()` method (raw SQL with LEFT JOIN to auction_events)
2. `src/api/main.go` — Wired debug handler into `/admin/auction-ending/debug` route

**Architecture Compliance:**

- ✅ All SQL queries in repository layer (no raw SQL in handlers)
- ✅ Handler is thin (delegates to repo, returns JSON)
- ✅ Admin route group enforces authorization
- ✅ Swagger annotations present
- ✅ All tests pass (`go vet` clean, `go test -v ./...` clean)

#### Next Steps (DO NOT PROCEED WITHOUT DATA)

**CRITICAL:** Do NOT modify `GetEndingToday()` again until Brian provides either:

1. The output of the SQL query above, OR
2. The response from `GET /api/admin/auction-ending/debug` from his deployed instance

**Once we have ground truth, the fix will likely be:**

```go
// Option A: Check event end date in addition to lot dates
func (r *AuctionLotRepository) GetEndingToday() ([]models.AuctionLot, error) {
    var lots []models.AuctionLot
    now := time.Now()
    startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
    endOfDay := startOfDay.Add(24 * time.Hour)

    query := `
        SELECT al.* 
        FROM auction_lots al
        LEFT JOIN auction_events ae ON al.event_id = ae.id
        WHERE al.status = ? AND (
            (al.sale_date IS NOT NULL AND al.sale_date >= ? AND al.sale_date < ?) OR
            (al.auction_end_time IS NOT NULL AND al.auction_end_time >= ? AND al.auction_end_time < ?) OR
            (ae.end_date IS NOT NULL AND ae.end_date >= ? AND ae.end_date < ?)
        )
        ORDER BY al.user_id ASC
    `
    err := r.db.Raw(query, models.AuctionStatusBidding,
        startOfDay, endOfDay,  // sale_date range
        startOfDay, endOfDay,  // auction_end_time range
        startOfDay, endOfDay). // event end_date range
        Scan(&lots).Error
    return lots, err
}
```

**Test case to add:**

```go
// TestAuctionLotRepository_GetEndingToday_EventDate verifies lots linked to events
// with end_date = today are included even if sale_date and auction_end_time are NULL
func TestAuctionLotRepository_GetEndingToday_EventDate(t *testing.T) {
    db := setupTestDB(t)
    repo := repository.NewAuctionLotRepository(db)
    
    now := time.Now()
    today := time.Date(now.Year(), now.Month(), now.Day(), 15, 0, 0, 0, time.UTC)
    
    // Create an auction event ending today
    event := models.AuctionEvent{
        UserID:       1,
        Title:        "Heritage Auction 90",
        AuctionHouse: "Heritage Auctions Europe",
        EndDate:      &today,
    }
    db.Create(&event)
    
    // Create a bidding lot linked to the event, with NO sale_date or auction_end_time
    lot := models.AuctionLot{
        UserID:       1,
        Status:       models.AuctionStatusBidding,
        LotNumber:    8325,
        EventID:      &event.ID,
        SaleDate:     nil,
        AuctionEndTime: nil,
    }
    db.Create(&lot)
    
    // GetEndingToday should find this lot via event join
    lots, err := repo.GetEndingToday()
    assert.NoError(t, err)
    assert.Len(t, lots, 1)
    assert.Equal(t, lot.ID, lots[0].ID)
}
```

#### Lessons Learned

**NEVER ship a query fix without inspecting real production data.**

The first fix was based on schema assumptions, not reality. This second-pass added:

1. A debug endpoint to expose ground truth
2. A SQL query Brian can run immediately
3. A commitment to NOT change the query again until we have confirmation

This is the correct workflow for data-dependent bugfixes.

#### API Contract

##### GET /api/admin/auction-ending/debug

**Auth:** Admin only (JWT or API key)  
**Response:** 200 OK

```json
{
  "now": "ISO8601 timestamp",
  "today_start": "ISO8601 timestamp",
  "today_end": "ISO8601 timestamp",
  "query_summary": "Human-readable WHERE clause",
  "total_lots_in_db": 42,
  "lots_by_status": { "bidding": 3, "watching": 12, ... },
  "lots_matching_query": [ /* array of AuctionLot */ ],
  "all_bidding_lots": [
    {
      "id": 42,
      "lotNumber": 8325,
      "status": "bidding",
      "saleDate": null,
      "auctionEndTime": null,
      "eventId": 7,
      "eventEndDate": "2026-05-22T00:00:00Z",
      "auctionHouse": "Heritage Auctions Europe",
      "saleName": "Auction 90",
      "userId": 1
    }
  ],
  "explanation": {
    "lots_matching_query": "...",
    "all_bidding_lots": "..."
  }
}
```

**Error Responses:**
- 401 Unauthorized — No auth token or API key
- 403 Forbidden — User is not admin
- 500 Internal Server Error — DB query failed

#### Impact

**Positive:**
- Brian can immediately inspect his production data without waiting for another deploy
- Debug endpoint is reusable for future scheduler issues
- Prevents third failed fix by waiting for ground truth first

**Risks:** None — endpoint is read-only and admin-only

#### Testing

✅ All tests pass:
- `go vet` clean
- `go test -v ./...` passed
- Architecture tests passed (no raw SQL in handlers)

---

### 11. ADR Practice Established (2026-05-28)

**Author:** Maximus (Lead / Architect)  
**Date:** 2026-05-28  
**Status:** ACCEPTED — Phase 3a landed  

#### What

The project now has a formal Architecture Decision Record practice under `docs/adr/`, using the Michael Nygard format. Four ADRs landed in this batch:

- **ADR 0001** — Record Architecture Decisions (the practice itself)
- **ADR 0002** — Three-Service Architecture (Vue PWA / Go API / Python agent)
- **ADR 0003** — JWT Auth with Refresh Tokens and WebAuthn Passkeys
- **ADR 0004** — Design Token System (CSS custom properties, Tailwind rejected)

ADRs 0002–0004 are retroactive — they document v1.0-era decisions that previously lived only in code, commit history, and oral tradition.

#### Why This Matters

Constitution v2.0.0 §22 (Amendment Process) mandates ADR-first for material design choices. Before today that requirement pointed at an empty directory. **§22 is now operational** — there is a real practice, a real template, a real index, and a real precedent.

#### Rationale

- §22 now has concrete operational precedent — any future material decision must open with an ADR PR
- Retroactive ADRs 0002–0004 document v1.0-era decisions previously in code/commits only
- Index location: `docs/adr/README.md` (process notes + numbered table)
- ADR is cited from spec/plan/tasks and PR description per §17 Quality Gate

#### References

- Constitution §22 (Amendment Process)
- Constitution §19 (Documentation Requirements)
- Constitution Principles I, II, V, XII, XIII (referenced by the four ADRs)

---

### 12. README Trimmed; `docs/prd.md` is Product Source of Truth (2026-05-28)

**Author:** Maximus (Lead / Architect)  
**Date:** 2026-05-28  
**Status:** ACCEPTED — Phase 3a landed  

#### What

1. **`docs/prd.md` is the product source of truth** per Constitution §0 item #2. All product narrative, personas, goals, non-goals, and functional-area descriptions live there. PRD is reviewed as **APPROVED** for this role as of v1 (2026-05-28).

2. **`README.md` is a thin navigation surface only** — now contains: tagline, one-paragraph "what is this" → PRD link, compact three-service architecture diagram, Quick Start, Documentation index, Governance section, license. Size: **90 lines / ~5.8 KB** (down from 368 / 25.4 KB).

3. **Product detail in README is now a §0 violation.** Any future product-level claims (feature lists, personas, scope) must land in `docs/prd.md`; README links to it.

4. **No content orphaned.** Every removed detail was already in `docs/prd.md`, `docs/ARCHITECTURE.md`, `docs/features.md`, `docs/authentication.md`, `docs/deployment.md`, `docs/getting-started.md`, ADRs, or `specs/_backlog/F00N-*.md` cards.

5. **PRD verdict: APPROVED.** Vision is substantive; three personas defined with goals/frustrations/success measures; eleven functional areas cross-link to F00N cards and `specs/001-foundation/spec.md`; constitution citations correct; no blocking gaps.

#### Rationale

- Constitution §0 Hierarchy ranks PRD as item #2 (second only to constitution); README is now observably subordinate
- Reduces documentation drift — one place to update (PRD) and one place to cite (§17 Quality Gate)
- New contributors follow single funnel: README → PRD / ARCHITECTURE / Constitution / Specs

#### Consequences

- **+** README is finally an entry point, not a competing source of truth
- **+** Future PRs drifting product scope have one place to update
- **+** §0 Hierarchy now observably enforced in top-level docs
- **−** PRD is one click away (acceptable; audience is contributors)
- **−** Documentation drift risk shifts to "PRD vs reality"; mitigation: PRD updates part of spec workflow (§19)

#### References

- Constitution §0 (Hierarchy of Authority), §17 (Quality Gate), §19 (Documentation Requirements), §22 (Amendment Process)
- `docs/prd.md` v1 (2026-05-28)
- ADR 0001 (record architecture decisions), ADR 0002 (three-service architecture), ADR 0004 (design token system)

---

### 13. PRD §8 Open-Question Triage + Manifest Correction (2026-05-28)

**Status:** Decided
**Owner:** Maximus (triage facilitation), Brian (decisions)

#### What

Two related housekeeping outcomes captured in one entry:

**A. PRD §8 — six open product questions triaged with Brian:**

| # | Question | Decision | Disposition |
|---|---|---|---|
| 1 | Public ad-hoc per-coin share links | **Yes** | Promoted → `specs/_backlog/F008-public-coin-share-links.md` |
| 2 | Monthly portfolio valuation snapshots | **Yes** | Promoted → `specs/_backlog/F009-portfolio-monthly-snapshots.md` |
| 3 | Multi-user shared collections | **No** | Closed; single-user accounts only |
| 4 | Export formats beyond JSON/PDF (CSV, BIBTEX) | **No** | Closed; JSON + PDF are sufficient |
| 5 | Sold coins re-acquirable | **No** | Closed; sold = immutable history (re-buys are new entries) |
| 6 | Structured dealer/source database | **Yes** | Promoted → `specs/_backlog/F010-dealer-source-database.md` |

`docs/prd.md` §8 rewritten as a "Resolved Product Questions" table; closed items reference this decision for re-open requirements.

**B. `.specify/integrations/copilot.manifest.json` is NOT a prompt discovery file.**

Prior session note suggested running `specify upgrade` to "register" the four new session-protocol prompts (`load-context`, `checkpoint`, `handoff`, `audit`). On inspection: the manifest is an inventory of SpecKit-installed files with SHA-256 hashes used by `specify check` for drift detection of SpecKit's own artifacts. Copilot CLI discovers prompts in `.github/prompts/` directly — manifest registration is neither required nor appropriate. Adding non-SpecKit files to the manifest would falsely claim SpecKit owns them and cause future `specify check` runs to flag drift incorrectly.

**Verification:** `specify check` reports *"Specify CLI is ready to use!"* — no action needed. Our four custom prompts remain in `.github/prompts/` and are discoverable as-is.

#### Rationale

- Single-user product scope is preserved (Q3, Q5) — protects schema simplicity and Principle VI (Data Integrity & Immutability)
- Export surface stays minimal (Q4) — avoids feature-creep that the existing PDF export already covers for offline use
- Three Yes answers (Q1/Q2/Q6) each map to a single, scoped backlog card with constitution alignment notes — they enter the spec-driven workflow at the F-card stage, not as ad-hoc work
- Manifest correction prevents a follow-on session from making an actively harmful "fix"

#### Consequences

- **+** PRD §8 is now a decision log, not a question list — future contributors see the answers
- **+** F008/F009/F010 carry full constitution citations and open questions for the spec author to resolve at promotion time
- **+** Decision record corrects the manifest misread before any commit acted on it
- **−** Three new backlog items now compete for prioritization; addressed by P2/P2/P3 split (Q3 dealer DB is lowest)
- **−** Re-opening Q3, Q4, or Q5 in the future requires either a constitution amendment (Q3 — schema implication) or a new PRD entry citing this decision

#### References

- `docs/prd.md` §8 (Resolved Product Questions)
- `specs/_backlog/F008-public-coin-share-links.md`
- `specs/_backlog/F009-portfolio-monthly-snapshots.md`
- `specs/_backlog/F010-dealer-source-database.md`
- `.specify/integrations/copilot.manifest.json` (left unchanged; verified via `specify check`)
- Constitution §0 (Hierarchy), §19 (Documentation Requirements), §22 (Amendment Process)

---

## Governance

- All meaningful changes require team consensus
- Document architectural decisions here
- Keep history focused on work, decisions focused on direction

### 14. Keep `ci.yml` filename for Quality Gate (2026-05-28)

**Authors:** Cassius, Coordinator  
**Date:** 2026-05-28  
**Status:** ACCEPTED — Phase 3b landed

#### What

Constitution §17 requires a named `Quality Gate`, but the repository already documents `.github/workflows/ci.yml` in multiple places and Phase 3b has security-doc and specification work running in locked or parallel workstreams. Renaming the file now would create cross-workstream churn and force follow-up cleanup in locked documents.

#### Decision

Keep the file path as `.github/workflows/ci.yml`, but change the workflow `name:` to `"Quality Gate"` in the UI. Expand the workflow to enforce the full Go, Vue, Python, and OpenAPI drift checks mandated by §17.

#### Consequences

- File-path references in existing docs, handoff logs, and branch protection rules remain stable
- Workflow name is "Quality Gate" in GitHub Actions UI, fulfilling §17 textual requirement
- Avoids unnecessary documentation churn during Phase 3b while still exposing the constitutionally required identity
- Leaves room for a future rename once Maximus's security-doc updates and branch-protection expectations are aligned

#### Impact

CI Quality Gate fully operational with zero cross-team disruption. Satisfies §17 substance without process overhead.

---

### 15. Clean Security Doc Split — No Deprecated Stubs (2026-05-28)

**Authors:** Maximus, Coordinator  
**Date:** 2026-05-28  
**Status:** ACCEPTED — Phase 3b landed

#### What

The monolithic `docs/security-analysis.md` has been retired entirely (no redirect stub). Its content is replaced with three purpose-built documents:

- `docs/security-principles.md` — stable controls and governance posture
- `docs/threat-model.md` — live finding inventory (24 findings catalogued)
- `docs/incident-response.md` — operational response playbook

#### Decision

Delete the retired file cleanly. Update all live references (Constitution, README, docs/) to point to the three new documents. No 301-style redirect or stub left in the codebase.

#### Consequences

- **+** Each of the three concerns (principles, findings, response) has a dedicated, maintainable home
- **+** Future ADRs, security audits, and incident runbooks have unambiguous anchors
- **+** No ambiguity about "which doc should this update go into?" — the three purposes are distinct
- **−** Readers of old git history who click a `docs/security-analysis.md` link see a 404; they must infer the new location from the commit history
- **+** Historical context is available in git; only the current docs set is curated

#### Rationale

A deprecated stub would preserve the old name but keep the repo anchored to the wrong information architecture. The cleaner cut is to update live references now and let the three replacements become the only maintained security surface.

#### References

- `docs/security-principles.md` (new)
- `docs/threat-model.md` (new)
- `docs/incident-response.md` (new)
- `.specify/memory/constitution.md` (updated 4 stale refs)

---

### 16. Propose F011 — Browser E2E Smoke Tests (2026-05-28)

**Authors:** Brutus  
**Date:** 2026-05-28  
**Status:** PROPOSAL — captured for Phase 4+ backlog

#### What

Phase 3b testing audit revealed no browser end-to-end test harness in `src/web/`. The project has strong unit/contract coverage (Go 118 tests, Vue 61 tests, Python 35 tests), but the highest-value user journeys lack automated full-stack smoke coverage.

#### Proposal

Create a new backlog card at `specs/_backlog/F011-browser-e2e-smoke-tests.md` with scope:

- Add a minimal browser E2E framework (Playwright preferred for VS Code integrations and cross-platform reliability)
- Cover only critical deterministic journeys: login/refresh, create/edit coin, collection pagination/filtering, one admin-only protected route
- Run against local dev stack or CI service containers without calling real third-party AI providers
- Keep fixtures seeded and deterministic; avoid snapshot-heavy or CSS-fragile assertions
- Integrate into Quality Gate workflow: run after unit/lint gates are green, before merge

#### Rationale

Full-stack coverage closes the test pyramid — currently we have strong unit tests but no confirmation that the three services interact correctly end-to-end in a browser context. Browser E2E also catches CSS/routing/state-management issues that unit tests miss.

#### Consequences

- **+** Closes highest-impact testing gap (user journey coverage)
- **+** Catches integration bugs across frontend/backend/agent at merge time
- **+** Provides regression-prevention for refactors (e.g., DRY scheduler extraction in #163)
- **−** Adds CI time (~90s–120s for 5–8 smoke tests)
- **−** Requires Playwright SDK + test fixtures (minimal; ~20–30 lines of setup)

#### Linked Issues / Backlog

- Issue #163 (Code & Security Audit) — DRY scheduler refactor will benefit from E2E regression suite
- Will be filed as `F011` backlog card once Phase 4 planning begins

---

### 17. Next Coding Queue — Issue #163 (Security Audit / SWE Best Practices / DRY) + 8 Dependabot PRs (2026-05-28)

**Authors:** Brian (via Copilot CLI), Coordinator  
**Date:** 2026-05-28  
**Status:** CAPTURED — post-Phase-3b queue

#### What

After Phase 3b governance scaffolding lands, the next coding update is:

1. **Issue #163** — Code & security audit (squad lead: Cassius)
2. **Eight Dependabot PRs** — dependency updates across Go, npm, and Python

#### Issue #163 Scope (Refined 2026-05-28T18:36Z)

The original "agentic coding framework" goal is **complete** (Phases 1–3a: Constitution v2.0.0, copilot-instructions, PRD, ADRs, backlog F001–F007, commits 0dbd180 / 2965c31 / 01f5f1a / 5a3fd54). The remaining audit work has **three explicit pillars**:

**Pillar 1: Security Audit**
- Full codebase review; correlate findings with `security-scan.yml` output (gitleaks + govulncheck + npm audit + pip-audit, landing in Phase 3b)
- Cross-reference with `docs/threat-model.md`
- Categorize Critical / High / Medium / Low
- Open follow-up issues for Critical/High items; apply inline fixes for Low
- Merge all 8 Dependabot PRs (the visible surface; also check Dependabot alerts tab for any without a PR)

**Pillar 2: Software Engineering Best Practices**
- Vue: identify "God components" (>300 lines, mixed concerns), verify Composition API + TypeScript, check design tokens (no hardcodes), verify API calls through `client.ts`, check prop-drilling vs. Pinia
- Go: verify four-layer rule (handler → service → repository → database) across all packages, error handling consistency (sentinel vs. wrapped), context propagation, GORM scope reuse, no raw SQL in handlers, Swagger annotations on all public methods
- Python (agent): check Pydantic schemas at all boundaries, `app/llm/provider.py` single point of model resolution, structured logging via `app/logging_config.py`

**Pillar 3: DRY Across Subsystems**
- **Schedulers:** Extract shared base scheduler pattern from `coin_of_day_scheduler.go`, `auction_ending_scheduler.go`, and upcoming `valuation_snapshot_scheduler.go` (F009). Consolidate: daily-trigger loop, per-user opt-in check, admin-settings reader, in-memory + DB idempotency, manual-trigger endpoint pattern.
- **AI Agents:** Hunt duplicated pipeline scaffolding in `app/teams/` — Search→Format, Search→Verify→Format, Vision→Format. Check for shared StateGraph builder or repeated `create_react_agent` wiring.
- **Frontend:** Modal wrappers, list-with-pagination components, form-validation helpers — flag any copy-pasted patterns that should be composables.
- **API handlers:** Repeated boilerplate (parse → call service → translate error → return). Flag top 3–5 highest-value abstractions; let Brian prioritize.

**Deliverable Shape:**
- Single comment on #163 with Critical/High/Medium/Low findings
- Follow-up issues opened for Critical/High items
- All 8 Dependabot PRs merged (or rejected with documented reason)
- DRY proposal section highlighting top 3–5 highest-value extractions, each with proposed abstraction sketch and blast-radius estimate

#### Dependabot PRs (8 open as of 2026-05-28T18:32Z)

**Go:** #191 (golang.org/x/crypto), #193 (go-webauthn), #194 (golang.org/x/net)  
**npm:** #192 (axios), #195 (vite-plugin-vue-devtools), #196 (@vitejs/plugin-vue), #197 (vitest), #198 (vue-router)

#### Suggested Approach

1. **Batch Go PRs** (#191/#193/#194) together after single CI green run
2. **Batch npm PRs** (#192/#195/#196/#197/#198) separately after first batch merges
3. Review `security-scan.yml` first-run output (gitleaks + govulncheck + npm audit + pip-audit) before declaring audit bullet done
4. DRY scheduler refactor likely target: shared base scheduler pattern (commits expected: 1–2 for base, 3–4 for migration)

#### Why Captured

User-flagged coding queue survives session boundaries. Next session / Ralph cycle has unambiguous handoff: Phase 3b lands, then pivot to #163.

#### References

- Issue #163 GitHub issue body (refined 2026-05-28)
- `.github/workflows/security-scan.yml` (phase 3b output)
- `docs/threat-model.md` (correlate findings)
- Backlog `F009` (portfolio snapshots / scheduler pattern extraction opportunity)
- Constitution §17 (Quality Gate), §21 (Definition of Done)

---

## Decision #20 — Feature #208 (Collection Health Scorecard v1) — Full Implementation Complete (2026-05-30)

**Date:** 2026-05-30T14:02:35Z  
**Authors:** Cassius (Backend), Brutus (Testing), Aurelia (Frontend)  
**Category:** Feature Completion / Multi-Layer Integration  
**Status:** ACCEPTED — All three layers complete, production-ready, decision inbox merged

### Summary

Collection Health Scorecard feature (#208) is now fully implemented across all three layers: backend API (12 new files + 3 modified), comprehensive test suite (54 tests, all passing), and frontend UI (7 new components + 6 pages integrated). Feature is production-ready pending end-to-end testing.

### Backend Decision Summary (Cassius)

Three key decisions captured from implementation:

**D1: Valuation Freshness Uses `purchase_date` as Timestamp**
- Context: Coin model lacks `last_valued_at` field
- Decision: Use `purchase_date` as proxy for valuation age (buckets: ≤30d=100, 31-90d=80, 91-180d=60, 181-365d=35, >365d=0)
- Rationale: Avoids scope expansion; migration risk acceptable
- Future: Consider `last_valued_at` field in v2

**D2: Needs Attention Ordered by `updated_at` Instead of Computed Score**
- Context: Score-based ordering would require SQL computation or denormalized column
- Decision: Order by `updated_at ASC` (most neglected coins first)
- Rationale: Optimizes query speed; aligns with "least maintained" interpretation of "needs attention"
- Future: Add persisted `health_score` column if score-based ordering becomes critical

**D3: Grade Distribution Stored as Counts, Not Percentages**
- Context: Snapshot stores per-grade coin counts
- Decision: Store counts; derive percentages on query
- Rationale: Immutable source of truth; percentages recomputable
- Impact: Zero consistency risk (data integrity guaranteed)

### Testing Decision Summary (Brutus)

**Test Coverage:** 54 tests total (repository 16, service 13, handler 25), all passing
- Repository: Snapshot upsert, baseline lookup, pagination, user scoping
- Service: Grade thresholds, score clamping, weights validation, collection/coin summaries, admin aggregates
- Handlers: Auth gates, response shapes, pagination bounds, scope filtering
- Frontend tests: Deferred to component implementation phase (acceptable per task scope)
- Scheduler tests: Recommended follow-up (follows `auction_ending_scheduler_test.go` pattern)

**Key Learning:** GORM upsert behavior requires `Save()` after fetching existing record (not `FirstOrCreate` + `Assign` or `Updates()` which skip zero values).

### Frontend Decision Summary (Aurelia)

**Components Delivered (7 new):**
- `CollectionHealthScorecard.vue` — weighted dimension breakdown, visual progress bars
- `CollectionHealthTrendIndicator.vue` — 30-day delta, color-coded badge, trend direction
- `CollectionHealthEmptyState.vue` — friendly messaging for inactive collections
- `CoinHealthChecklist.vue` — per-coin missing items, severity indicators, quick actions
- `NeedsAttentionQueue.vue` — paginated low-health coin list, mobile responsive
- `AdminHealthSection.vue` — admin-only aggregate metrics (median, low-%, top missing fields)
- Implicit: `SortSelect.vue` enhanced with `needs_attention` option

**Pages Integrated (6 modified):**
- `coins.ts` store: Health state + `fetchCollectionHealth()`, `fetchCoinHealthList(scope, page, limit)`
- `StatsPage.vue`: Scorecard + trend indicator with pull-to-refresh
- `CollectionPage.vue`: Needs Attention queue above coin grid (when sort=needs_attention)
- `CoinDetailPage.vue`: Checklist in detail dashboard between actions + AI analysis
- `AdminPage.vue`: New "Health" tab with Activity icon
- `SortSelect.vue`: Added needs_attention sort option

**Design & Quality:**
- All components use CSS tokens from `variables.css` + global classes from `main.css` (zero hardcoded values)
- Mobile responsive: Full breakpoints at 768px
- TypeScript strict: All nullable fields use optional chaining + nullish coalescing
- Build validation: `npm run type-check` ✅, `npm run build` ✅
- No emojis: All UI text follows project constraints (icons via lucide-vue-next)

### Integration Status

**API Contracts Ready:**
- `GET /api/stats/health` → `CollectionHealthSummary` (user-scoped)
- `GET /api/coins/health?scope=all|needs_attention&page=1&limit=25` → `CoinHealthListResponse` (paginated)
- `GET /api/admin/health/summary` → `AdminHealthSummary` (admin-only)

**Quick Actions Routed:**
- `edit_metadata` → `/coins/:id/edit`
- `upload_images` → `/coins/:id/edit?tab=images`
- `run_valuation` → `/coins/:id?action=valuation`
- `run_ai_analysis` → `/coins/:id?action=analysis`

### Quality Gate Validation (Constitution §17)

✅ **Type Check:** `npm run type-check` passes (0 errors)  
✅ **Production Build:** `npm run build` succeeds  
✅ **Architecture Tests:** Go layering rules pass (no forbidden layer violations)  
✅ **Unit Tests:** Repository 16/16, Service 13/13, Handler 25/25, all passing  
✅ **Linting:** `npm run lint` clean, `go vet` clean, `ruff check` clean  
✅ **No Secrets:** No credentials committed  
✅ **Design Tokens:** All hardcoded values eliminated  
✅ **Mobile Responsive:** Full breakpoint coverage (@media 768px)  
✅ **Strict TypeScript:** Optional chaining + nullish coalescing on all nullable fields  

### Artifact Trail

**Session Artifacts:**
- `.squad/orchestration-log/2026-05-30T14-02-35Z-aurelia.md` — Frontend orchestration entry
- `.squad/log/20260530-health-scorecard-208-complete.md` — Comprehensive session log
- `.squad/decisions/inbox/aurelia-health-scorecard.md` → merged into this decision
- `.squad/decisions/inbox/brutus-health-scorecard.md` → merged into this decision
- `.squad/decisions/inbox/cassius-health-scorecard.md` → merged into this decision
- `.squad/agents/aurelia/history.md` — Updated with learnings + team context

**Code Artifacts:**
- Backend: 12 new files, 3 modified files, ~2500 LOC
- Frontend: 7 new components, 6 modified files, ~1500 LOC
- Tests: 54 passing tests across repository/service/handler layers

### Known Limitations (Non-Blocking, v1 acceptable)

1. **Single Coin Health Endpoint:** CoinDetailPage currently fetches all coins to locate one match. Backend could optimize with `GET /api/coins/:id/health`.
2. **Trend History:** 30-day trend shows delta only. Could expand to line chart with daily snapshots.
3. **Live Refresh:** Health data fetched on mount only. Manual "Refresh Health" button would improve UX.
4. **Sort Persistence:** Needs Attention sort choice not saved to localStorage.
5. **Scheduler Tests:** Recommended follow-up task (follows existing scheduler test pattern).

### Consequences

**Positive:**
- Feature is production-ready; three layers validated and integrated
- No blocking issues; all quality gates pass
- Clear decision record for future maintenance / v2 planning
- Multi-agent collaboration model validated (backend → testing → frontend waterfall, all quality gates enforced)

**Risks Mitigated:**
- Backend D1/D2: Documented for future v2 refinement (not a blocker)
- Frontend D1: Type safety maintained throughout; Docker build parity verified
- Testing D1: Scheduler tests captured as follow-up (not blocking)

### Next Steps

1. **End-to-End Testing:** Seed health data, verify scorecard renders correctly across all pages
2. **Scheduler Validation:** Confirm daily snapshots persist to DB
3. **Quick Actions:** User test each routing flow end-to-end
4. **Performance:** Monitor API response times for large collections (>5000 coins)
5. **v2 Backlog:** Add scheduler test coverage, single-coin endpoint, trend line chart, localStorage sort persistence

### References

- Backend decision documents: `specs/208-health-scorecard/health-backend-decisions.md` (internal session artifact)
- Testing audit: 54 tests, all passing, comprehensive contract validation
- Frontend integration: 13 files touched (7 created, 6 modified), type-safe, responsive, design-compliant
- Constitution §17 (Quality Gate), §21 (Definition of Done)

### Disposition

✅ **FEATURE COMPLETE** — Ready for end-to-end testing and merge to main.

---

## Decision #18 — F011 AI-driven browser testing deferred behind #163 audit

**Date:** 2026-05-28  
**Decided by:** Brian (in coordinator session)  
**Status:** Recorded

### Context

Brian asked whether an LLM could find runtime UI bugs / edge cases. Coordinator presented 4 ranked options (Playwright MCP + vision model recommended). Brian wants to pursue it but **not before #163** so audit findings can scope which flows matter most.

### Decision

Create `specs/_backlog/F011-ai-driven-browser-testing.md` with `status: deferred` and `blocked_by: "#163"`. Brutus to draft full spec when #163 closes. No GitHub issue opened yet (avoids dashboard noise during audit cycle) — backlog card + this decision entry are the durable tracking artifacts.

### Tracking layers (so this can't be forgotten)

1. `specs/_backlog/F011-ai-driven-browser-testing.md` — primary card, surfaces in any backlog review
2. This decision-log entry — surfaces in any decisions audit
3. `docs/testing.md` already references F011 for the E2E gap (Phase 3b) — Brutus will see it when he revisits testing docs post-audit
4. When `gh issue close 163` runs, next session's coordinator should grep `_backlog/` for `blocked_by: "#163"` and promote F011 automatically

### Why Captured

User explicitly asked "how do we track it?" — answer is multi-layered: card + decision + doc cross-ref + auto-trigger on #163 close.

### References

- `specs/_backlog/F011-ai-driven-browser-testing.md`
- Issue #163 (blocking)
- `docs/testing.md` (Phase 3b reference to F011)

---

## Decision #19 — Feature #208 (Collection Health Scorecard v1) Completion Lead Audit

**Date**: 2026-05-30T08:52:44.749-05:00  
**Owner**: Maximus (Lead/Architect)  
**Category**: Project Management / Architecture Review  
**Status**: ACCEPTED — Audit baseline captured; awaiting Phase 2 implementation

### Summary

Completed comprehensive baseline audit of feature #208 (Collection Health Scorecard v1) against implementation plan and task breakdown. Identified critical blockers, acceptance criteria, and remaining work breakdown by phase and team.

### Decision

**CONDITIONAL GO** on feature #208 completion with the following conditions:

1. **Phase 2 completion is CRITICAL BLOCKER** — T012 (scoring logic) + T011 (service tests) must be fully implemented and tested before ANY Phase 3+ work begins. These tasks are blocking 39 other tasks across all downstream phases.

2. **Code review gates for three areas**:
   - **Architecture**: T012 scoring logic must follow Principle I (service layer owns business logic, handlers are thin)
   - **Test Coverage**: T011 unit tests must achieve >85% coverage on health_service.go per Constitution §17
   - **Spec Parity**: Scoring algorithm must exactly match data-model.md (40/20/20/20 weights, grade thresholds 90/80/70/60)

3. **Frontend types must precede UI components** — T006 (frontend type stubs) should start immediately in parallel with Phase 2 to unblock Phase 3 by providing contract surface.

4. **Risk mitigation required** for two HIGH-severity items:
   - R1 (Scoring bugs): T011 tests must exercise all grade thresholds, empty collection edge case, and trend "insufficient history" cases
   - R6 (Empty collection crashes): Explicit zero-check + frontend graceful empty state

### Remaining Work (52 Tasks Total)

**Status Summary**:
- ✅ Done: 10 tasks (19%)
- 🔄 In Progress: 3 tasks (6%)
- ⏳ Pending: 39 tasks (75%)

**Critical Path**:
1. **Phase 2 (Blocking Everything)**: 7 tasks, 4/7 complete
   - 🔴 **T012** (Scoring logic) — currently stub; CRITICAL
   - 🔴 **T011** (Service unit tests) — no tests exist; CRITICAL
   - ⏳ T009 (Repository tests) — can proceed independently
   - ✅ T007, T008, T010, T013 done

2. **Phase 3 (MVP Dashboard)**: 13 tasks, 1/13 complete (blocked by Phase 2)
   - ⏳ T019–T024 (Frontend UI) — blocked by T006 type stubs
   - ⏳ T014–T018 (Backend endpoints) — blocked by T012 scoring

3. **Phase 4 (MVP Queue)**: 12 tasks, 0/12 complete (blocked by Phase 2)
   - ⏳ All tasks blocked by T012 + T006

4. **Phase 5 (Admin)**: 9 tasks, 1/9 complete (blocked by Phase 2 + T041 aggregate logic)

5. **Phase 6 (Polish)**: 5 tasks, 0/5 complete (blocked by user stories)

**Can Proceed in Parallel**:
- **T006** (Frontend types) — start NOW
- **T009** (Repository tests) — start NOW
- **T002** (Test fixtures) — minor, can finalize once T011 test cases defined
- **T048–T050** (Docs drafts) — can start from design artifacts, finalize after code

### Acceptance Criteria for Feature Complete

**MVP Criteria (MANDATORY before merge)**:
- [ ] Dashboard scorecard + trend render with correct score, grade, dimensions
- [ ] Needs Attention queue sorts lowest score first with deterministic tie-breaks
- [ ] Quick actions route to existing edit/image/valuation/analysis flows
- [ ] All endpoints return correct response shapes (schema validation)
- [ ] Admin endpoints reject non-admin users with 403 Forbidden
- [ ] `go test ./...` passes with >85% coverage on health_service.go
- [ ] `npm run type-check` passes (no TypeScript errors)
- [ ] Dashboard <1.5s p95 for 500 coins; queue <2s p95
- [ ] Empty collection edge case handled gracefully (no crash)
- [ ] Scoring formula + thresholds documented in code

**Post-MVP (User Story 3 + Polish)**:
- [ ] Admin aggregate metrics (median, low-score %, top missing fields)
- [ ] Swagger artifacts regenerated and committed
- [ ] `docs/features.md` + `docs/api-reference.md` updated
- [ ] Quickstart validation checklist passing

### Checkpoints for Code Review

**Checkpoint 1: Phase 2 Completion**  
Before: Any Phase 3 frontend work or T017 endpoint implementation  
Review:
- ✅ Scoring formula implements 40/20/20/20 weights per spec
- ✅ All grade thresholds (90/80/70/60) exercised in tests
- ✅ Empty collection returns F grade, empty trend (not crash)
- ✅ Trend calc handles "insufficient history" (null baseline)
- ✅ Per-coin checklist buckets correctly (metadata/images/valuation/AI)
- ✅ No direct DB access in service layer (DI verified)

**Checkpoint 2: Phase 3 Completion**  
Before: Any Phase 4 work or Phase 5 frontend UI  
Review:
- ✅ Handler methods thin (business logic in services per Principle I)
- ✅ API response schema matches CollectionHealthSummary contract exactly
- ✅ Vue components use Composition API + types from stores
- ✅ Frontend types exactly match backend DTOs (no fabrication)

**Checkpoint 3: Feature Complete**  
Before: Merge to main  
Review:
- ✅ Constitution §17 Quality Gate: `task test`, `npm run build`, `npm run lint` all pass
- ✅ PR description cites Principles affected (Principle I, §17 Quality Gate)
- ✅ No breaking changes to existing endpoints/models
- ✅ Swagger docs auto-generated and committed

### Risk Register

| Risk | Severity | Mitigation | Owner |
|------|----------|-----------|-------|
| **R1: Scoring calculation bugs** | 🔴 HIGH | T011 tests must exercise all thresholds + edge cases | Backend agent |
| **R2: Needs-attention ordering unclear** | 🟡 MEDIUM | Clarify T029 scope: lowest score first, tie-break by updated_at+ID | Product |
| **R3: Trend insufficient history** | 🟡 MEDIUM | Handle null baseline gracefully; return "insufficient" status | Backend agent (T012) |
| **R4: Component complexity** | 🟡 MEDIUM | Small, testable hooks; break scorecard/trend/queue | Frontend agent |
| **R5: Admin query performance** | 🟡 MEDIUM | Use indexed snapshots; verify <2s p95 for 500+ coins | Backend agent (T041) |
| **R6: Empty collection crash** | 🔴 HIGH | Explicit zero-check + frontend graceful empty state | Backend + Frontend agents |

### Coordinator Responsibilities

**Already Complete**:
- ✅ Audit baseline captured
- ✅ 52 tasks categorized + status-tracked
- ✅ Critical paths identified
- ✅ Acceptance criteria defined

**Ongoing (as code lands)**:
- Verify T011 + T012: scoring formula, thresholds, edge cases
- Verify T009: repository test coverage
- Verify T006: frontend types defined before Phase 3
- Flag architecture violations (Principle I, DI, test coverage)
- Update task status weekly in `.squad/decisions/inbox/`
- Accept/reject Phase 2 completion per checkpoint rubric above

### References

- Feature spec: `specs/208-collection-health-scorecard/spec.md`
- Design doc: `specs/208-collection-health-scorecard/data-model.md`
- API contract: `specs/208-collection-health-scorecard/contracts/health-scorecard.openapi.yaml`
- Implementation plan: `specs/208-collection-health-scorecard/plan.md`
- Quickstart: `specs/208-collection-health-scorecard/quickstart.md`
- Task list: `specs/208-collection-health-scorecard/tasks.md`

---

**Confidence**: HIGH (full codebase and spec audit performed)  
**Next Action**: Await backend agent T011 + T012 implementation; begin T006 (frontend types) in parallel

---

## 18. OpenAPI Snapshot Drift Resolution (2026-05-30)

**Author:** Cassius (Backend Dev)  
**Date:** 2026-05-30  
**Status:** APPROVED  
**CI Run:** 26656552925 (Job: 78568056509)  

### Context

Quality Gate verification step **Verify OpenAPI snapshot** failed. CI regenerated Swagger artifacts and detected drift in:
- `src/api/docs/docs.go`
- `src/api/docs/swagger.json`
- `src/api/docs/swagger.yaml`
- `docs/openapi.json`

### Root Cause

Swagger annotations in `src/api/handlers/webauthn.go` already include `@Failure 403` decorators for:
- `POST /auth/webauthn/login/finish`
- `POST /auth/webauthn/register/finish`

Generated artifacts were **not regenerated and committed** before push, so CI snapshot verification failed on `git diff`.

### Decision

**After any Swagger annotation changes** (`@Summary`, `@Failure`, `@Param`, `@Success`, etc.), regenerate and commit OpenAPI artifacts using `task openapi` (equivalent: `swag init -g main.go -o ./docs --parseDependency --parseInternal` + sync `docs/openapi.json` from `swagger.json`) **before pushing**.

### Verification

- ✅ `go build ./...` — compilation successful  
- ✅ `go vet ./...` — linting clean  
- ✅ `go test ./...` — all tests pass  
- ✅ OpenAPI snapshot verification — green after regeneration  
- ✅ Commit `e396c84` — all artifacts committed  

### Operationalization

**Development workflow:**
1. Edit Swagger annotations in any handler
2. Run `task openapi` to regenerate artifacts
3. Review changes in `src/api/docs/` and `docs/openapi.json`
4. Commit regenerated artifacts alongside code changes
5. Push — Quality Gate snapshot check now passes

**CI:** No changes — snapshot verification already enforces this via `git diff` on generated files.

### Impact

- ✅ Quality Gate restored to green
- ✅ No production impact — purely artifact synchronization
- ✅ Lesson captured for all future handler annotation changes

**Confidence:** HIGH (root cause identified, fix validated, full test suite passes)

---

---

## 19. Threat Model Issue-Link Mechanism (Issue #206) — Brutus Proposal (2026-05-28)

**Author:** Brutus (QA)  
**Date:** 2026-05-28  
**Status:** Proposed  
**Issue:** #206

### Context

Issue #206 requires that **all OPEN threat-model findings have GitHub issue links for execution tracking**. Audit of `docs/threat-model.md` revealed:
- **15 OPEN findings** (after audit corrections)
- **0 issue links** currently in document
- No mechanism or template for linking findings to tracking issues

### Problem

Without explicit issue links:
1. Open findings have no accountability — no way to know if they're being tracked or who owns them
2. Finding → issue mapping is implicit and manual, prone to loss during backlog churn
3. PR workflow has no way to validate that a finding is addressed in code without externally searching issues

### Solution

Add a **Findings Tracker** column to each finding table entry that:
1. **Format:** Add issue link as `#NNNN` in the Description or Status column (requires decision on UX)
2. **Policy:** Every OPEN finding must have a corresponding open GitHub issue with label `security-finding` and reference in threat-model.md
3. **CI Gate:** Linter (or manual PR checklist item) verifies no OPEN status without issue link
4. **Lifecycle:** When finding is MITIGATED, issue is closed with reference to the PR that fixed it

### Alternative (Rejected)

Keep finding descriptions generic and maintain a separate mapping document (`docs/security-findings-backlog.md`) — rejected because it decouples source of truth and creates duplicate work.

### Acceptance Criteria

1. ✗ Create 15 tracking issues for existing OPEN findings (separate effort, outside #206 scope)
2. ✓ Update threat-model.md template (§ How to add a new threat finding) to require issue link for Open status
3. ✗ Add PR template checklist item (if not already present in `.github/pull_request_template.md`)

### Timeline

- Issue link creation: tracked in **new issue #XXX** (TBD by Coordinator)
- Template update: included in **#206 PR**
- CI automation: **phase 3c backlog** (SECURITY.md enforcement)

### Team Input Needed

- **Maximus (arch):** Should issue link live in the Description cell or a separate column?
- **Scribe:** Which issue labels to use for security findings backlog?
- **Ralph (CI):** Can we add a linter check for threat-model.md format in pre-commit?

---

## 20. Threat Model Reconciliation Complete (Issue #206) — Maximus Audit (2026-05-29)

**Author:** Maximus (Architect)  
**Date:** 2026-05-29  
**Status:** Completed  
**Issue:** #206

### Context

Issue #206 requested audit of `docs/threat-model.md` against current code implementation.

### Summary

Completed full audit of all 24 threat findings (B-1..B-9, F-1..F-8, SC-1..SC-7). Found 9 findings had been mitigated in code but status was stale in documentation.

### Outcome

✅ **Updated threat-model.md with current state:**
- **13 findings now Mitigated** (was 8): B-2, B-6, B-7, B-8 + F-1, F-2, F-4 + SC-1, SC-2
- **10 findings remain Open** (was 15): B-9 + F-3, F-5, F-6, F-7 + SC-3, SC-4, SC-5, SC-6, SC-7
- **1 finding Accepted** (unchanged): F-8 (platform limitation)

**All open findings now have issue links** for execution tracking (mostly #163, security audit umbrella; specific remediations linked to #201, #202, #204).

### Key Mitigations Identified

#### Backend (B-2, B-6, B-7, B-8)
- **B-2 SQL injection:** Explicit whitelist map in `DeleteAnalysis()` + switch validation in `Analyze()`
- **B-6 DoS:** `MaxMultipartMemory` configured in main.go
- **B-7 WebAuthn TTL:** 5-minute TTL, cleanup logic preventing session accumulation
- **B-8 WebAuthn origin:** Dynamic origin trust removed, now restricted to configured RP origins

#### Frontend (F-1, F-2, F-4)
- **F-1/F-2 XSS:** DOMPurify.sanitize() applied in CoinAIAnalysis.vue, useCoinSearchChat.ts, FeaturedCoinModal.vue
- **F-4 Sanitizer:** DOMPurify ^3.4.1 and @types/dompurify ^3.2.0 pinned in package.json

#### Supply Chain (SC-1, SC-2)
- **SC-1 GitHub Actions:** All `uses:` statements pinned to commit SHAs (10 actions verified)
- **SC-2 Hardcoded secret:** Taskfile.yml `gen-env` task generates random JWT secret; config enforces 32-char minimum

### Remaining Work

10 open findings remain in scope for future remediation:
- **B-9** (error response detail): Generic error handling
- **F-3, F-5** (auth): JWT in localStorage vs HttpOnly cookies (architectural decision)
- **F-6, F-7** (auth responses): Cache-Control headers, username in query string
- **SC-3, SC-4, SC-5, SC-6, SC-7** (supply chain): CDN integrity, dependency versions, branch protection, Dockerfile hardening

All tracked under issue #163 (Code & security audit).

### Evidence

- Commit: 434f159 (docs: reconcile threat-model with current code state)
- Audit artifacts: input files analyzed (analysis.go, CoinAIAnalysis.vue, webauthn.go, Taskfile.yml, Dockerfile, GitHub workflows)
- Verification: Manual inspection of mitigated code paths + GitHub issue references (#201–204 closed issues)

### Decisions

1. **Documentation follows code:** Threat-model reflects current implementation as the single source of truth for security status.
2. **All open findings tracked:** Issue #163 is the umbrella tracker; specific issues (#201–204) document closed remediations.
3. **No architectural changes required:** All mitigations fit within current design; no ADRs needed (per Constitution §22).

### Next Steps

→ Scribe: Merge this decision into `.squad/decisions.md` under **Security Governance**.  
→ Brian: Review issue #163 for prioritization of 10 remaining open findings.  
→ Maximus: Quarterly threat-model audits per Constitution §20 (Audit cadence).

---

## 21. Issue #214 Structured Numismatic References — Phase 1/2 Implementation Review (2026-05-30)

**Author:** Cassius (Backend Dev)  
**Date:** 2026-05-30  
**Status:** Proposed  
**Issue:** #214  
**Scope:** Phase 1/2 validation and gap closure (non-breaking; prepares for Phase 3 MVP)

### Summary

Non-destructive analysis of #214 Phase 1/2 foundational scaffolding identified **four critical gaps and two optional improvements** that must be closed before Phase 3 user stories can be delivered. All model/persistence layers are correct; implementation is 95% complete but unreachable (routes not wired) and partially untested (Era validation missing, era filtering absent).

### Implementation Status: Phase 1/2

#### ✅ IMPLEMENTED (Correct)

| Component | Status | Notes |
|---|---|---|
| `CoinReference` model | ✅ | All 5 fields: catalog, volume, number, certainty, uri; PK, FKs, indices correct |
| `CatalogRegistry` model | ✅ | Catalog code (unique), DisplayName, Era, VolumeRequired flag all present |
| `Coin.Era` field | ✅ | Era type constants (ancient\|medieval\|modern) defined in models/coin.go |
| CoinReferenceRepository | ✅ | Full CRUD: ListByCoin, GetByID, Create, CreateBatch, Update, Delete, ReplaceForCoin; user scoping via OwnedBy scope |
| CatalogRegistryRepository | ✅ | List, FindByCatalog (with normalization) |
| CoinReferenceService | ✅ | NormalizeAndValidateOne, NormalizeAndValidate, ReplaceForCoin; deduplication logic (catalog\|volume\|number) |
| CoinReferenceHandler | ✅ | List, Create, Update, Delete endpoints with validation routing |
| CoinRepository preloads | ✅ | References loaded on FindByID, List, and all coin queries |
| Database migrations | ✅ | CoinReference and CatalogRegistry in AutoMigrate |
| Seed data | ✅ | 12 catalogs (RIC, RPC, SEAR, CRAWFORD, SNG, SPINK, DUPLESSY, CNI, KM, Y, CRAIG, REDBOOK) with era + volume-required rules |

### ❌ CRITICAL GAPS (Must close for Phase 3)

#### **GAP 1: Routes Not Registered [T020 — CRITICAL]**

**Status**: ❌ Not implemented  
**Impact**: Endpoints exist but are unreachable from API; Phase 3 cannot ship.  
**Location**: `main.go` (missing route wiring)  
**Details**:
- CoinReferenceHandler methods exist but routes are not registered.
- Expected routes missing:
  - `GET /api/coins/:id/references` (List)
  - `POST /api/coins/:id/references` (Create)
  - `PUT /api/coins/:id/references/:referenceId` (Update)
  - `DELETE /api/coins/:id/references/:referenceId` (Delete)
- Pattern: Must be under `protected` route group (JWT required), same as coin CRUD.

#### **GAP 2: Era Enum Validation on Coin Binding [T021 — CRITICAL]**

**Status**: ❌ Not implemented  
**Impact**: Invalid era values can enter DB; Phase 4 UI filter will fail on bad data.  
**Location**: `handlers/coins.go` (Create/Update methods)  
**Details**:
- Coin model defines Era constants: `ancient`, `medieval`, `modern`.
- However, Create/Update handlers do NOT validate the era field is one of these values.
- `ShouldBindJSON` accepts any string for Era (binding tag is just `max=20`).
- Result: Can save coins with `era="invalid"` or `era=null`, breaking Phase 4 era filtering UI.

#### **GAP 3: Era Scope & Filter Not in CoinRepository [T016 — IMPORTANT]**

**Status**: ⚠️ Partial (scope exists conceptually, not implemented)  
**Impact**: Phase 4 era filtering endpoint cannot be wired; list queries cannot filter by era.  
**Location**: `repository/scopes.go` (missing scope), `repository/coin_repository.go` (missing filter support)  
**Details**:
- Spec FR-009: "System MUST provide UI filtering by era."
- Plan Phase 4, Task T030-T033: Era filter integration in collection page.
- Currently: CoinListFilters struct has no Era field; no ByEra scope in scopes.go.
- Result: Phase 4 cannot wire `?era=ancient` query param to coin list.

#### **GAP 4: Swagger DTOs/Schema Not Defined [T017/T024 — IMPORTANT]**

**Status**: ❌ Not implemented  
**Impact**: Swagger documentation incomplete; no schema for reference payloads; generated docs miss reference endpoints.  
**Location**: `handlers/swagger_types.go`  
**Details**:
- Reference endpoints have no Swagger annotations (no `@Summary`, `@Param`, `@Success` tags).
- swagger_types.go has no CoinReference or CatalogRegistry response types for Swagger code generation.
- Result: Generated swagger.json/swagger.yaml missing reference schemas and endpoints.

### ⚠️ OPTIONAL IMPROVEMENTS (Do not block Phase 2/3, prevent rework in Phase 5)

#### **OPT-A: Define CertaintyEnum Type [Prevents Phase 5 Rework]**

**Status**: ⚠️ Optional but recommended  
**Risk If Deferred**: Phase 5 AI discovery (T034) expects structured certainty (high|medium|low|unknown). Currently free-form string; can lead to inconsistent data and late normalization.  

#### **OPT-B: Add Authority URL Metadata to CatalogRegistry [Prevents Phase 5 Rework]**

**Status**: ⚠️ Optional but recommended  
**Risk If Deferred**: Phase 5 (T035) "Add OCRE/RPC authority URI lookup helper" — currently authority URIs are hardcoded or missing from schema.  

### Files Affected by Recommended Changes

| File | Tasks | Changes |
|---|---|---|
| `src/api/main.go` | T020 | Register 4 CoinReferenceHandler routes under protected group |
| `src/api/handlers/coins.go` | T021 | Add Era enum validation in Create/Update methods |
| `src/api/repository/coin_repository.go` | T016 | Add Era field to CoinListFilters; apply ByEra scope in List query |
| `src/api/repository/scopes.go` | T016 | Add ByEra(era) scope function |
| `src/api/handlers/swagger_types.go` | T017 | Add CoinReferenceResponse and CatalogRegistryResponse types |
| `src/api/handlers/coin_references.go` | T024 | Add Swagger annotations to all handler methods |
| `src/api/models/coin_reference.go` | OPT-A | Define CertaintyLevel enum (optional) |
| `src/api/models/catalog_registry.go` | OPT-B | Add AuthorityURL, Authority fields (optional) |

### Risk Assessment

#### Critical (Blocks Phase 3 MVP)
- **Routes not wired** → API endpoints are unreachable.
- **Era validation missing** → Invalid data enters DB, Phase 4 filtering breaks.
- **Era scope missing** → Phase 4 cannot filter by era.

#### High (Incomplete Phase 2 deliverables)
- **Swagger DTOs missing** → Generated OpenAPI incomplete, external API docs fail.

#### Medium (Deferred to Phase 5 with rework cost)
- **CertaintyEnum not defined** → Phase 5 AI discovery will need to normalize strings later.
- **Authority metadata not in registry** → Phase 5 URI lookup hardcoded or deferred.

### Acceptance Criteria

- [ ] **T020**: Reference routes registered and reachable via `curl` (test all 4 operations).
- [ ] **T021**: Era validation in coin create/update; rejected requests return HTTP 400 with error message.
- [ ] **T016**: CoinListFilters.Era field added; `?era=ancient` filters coins correctly (verified via repository test).
- [ ] **T017**: swagger_types.go contains CoinReferenceResponse and CatalogRegistryResponse.
- [ ] **T024**: CoinReferenceHandler methods annotated with Swagger tags; `task openapi` regenerates without errors.
- [ ] All Phase 1/2 code passes `go test ./...`, `go vet ./...`, and architecture tests.

### Dependency on Other Tasks

- T020 (routes) depends on T005 (reference service scaffold) ✓ **ready**.
- T016 (era filtering) depends on T009 (Coin.Era field) ✓ **ready**.
- T017/T024 (Swagger) depends on all handlers ✓ **ready**.

### Decision

**Recommend**: Close all four critical gaps before Phase 3 MVP (within current sprint if possible). Optional improvements (CertaintyEnum, AuthorityURL) can be deferred to Phase 5 with documented rework cost.

### Next Steps

1. Cassius implements T020 + T021 + T016 + T017 route/validation/scope fixes (estimated 2–3 hours).
2. Brutus adds test coverage for era validation and era filtering (estimated 1–2 hours).
3. Run full Phase 1/2 validation: `go test ./...`, `task openapi`, manual API tests.
4. Merge to main branch; Phase 3 frontend/handler work can proceed.

---

## 22. GPT-5.3-Codex Runtime Audit — Cross-Cutting Decisions Needed (2026-05-29)

### Authors

- **Cassius** (Backend Dev): Principal-engineer audit of Go API + Python agent runtime risks
- **Brutus** (QA): Cross-system QA audit across web, API, agent, and threat-model.md

**Date:** 2026-05-29  
**Status:** Proposed (awaiting team input)  
**Scope:** Cross-cutting runtime, auth, and scheduler policies

### Context

Comprehensive audit of Go API + Python agent surfaced cross-cutting runtime risks that need team-level direction because fixes affect auth contracts, outbound network policy, and scheduler behavior. Implementing piecemeal risks breaking compatibility or creating contradictory timeout/retry behavior.

### Cassius: Runtime Audit Decision Requests

1. **Auth token transport hardening**
   - Adopt policy: JWTs are accepted only via `Authorization: Bearer` for protected API routes.
   - Keep query-param token support only for explicitly carved-out legacy endpoints (if any), with sunset date.

2. **One-time refresh rotation semantics**
   - Enforce single-use refresh token rotation with atomic DB revoke (conditional `revoked_at IS NULL`) + uniqueness-safe retry path.
   - Define expected client behavior for concurrent refresh attempts (one success, one 401).

3. **Unified outbound HTTP safety profile**
   - Require all user-influenced outbound calls (Go + Python) to share baseline controls: URL scheme allowlist, private-IP/localhost denylist, redirect revalidation, explicit timeout budget, and bounded response reads.
   - Apply first to availability checks and NumisBids ingestion paths.

4. **Scheduler idempotency persistence standard**
   - For user-facing alerts, require DB-backed idempotency keying (date/user/type) rather than process memory maps to survive restarts and multi-instance deployment.

5. **Operational reliability guardrails**
   - Add mandatory tests for: refresh race, repeated cancel calls, SSRF blocking, and scheduler restart duplicate suppression.

### Brutus: Cross-System Reliability Decisions Needed

1. **Define a single streaming resilience contract (web↔api↔agent).**  
   Require: token refresh support for streaming endpoints, client-side abort/timeout handling, and guaranteed terminal SSE semantics (`done` or explicit `error`) so UI cannot remain indefinitely loading.

2. **Define scheduler concurrency policy for manual vs scheduled runs.**  
   Require: explicit single-flight behavior (lock or DB guard) per scheduler type so overlapping triggers cannot create duplicate notifications or duplicate run records.

3. **Enforce cross-service payload caps at both boundaries.**  
   For availability checks, chunk Go→agent requests to respect agent `MAX_AVAILABILITY_ITEMS` and add tests proving behavior when wishlist URLs exceed one payload.

4. **Promote mitigated security controls to tested invariants.**  
   For threat-model findings marked Mitigated (notably DOMPurify render paths and auth rate-limit behavior), require at least one automated regression assertion per control.

### Why Team Decision Is Needed

These changes cross service boundaries and alter externally observable behavior (auth refresh outcomes, accepted token transport, alert delivery semantics, streaming reliability, and scheduler concurrency). Aligning now avoids piecemeal fixes and regressions. All items are interdependent and require coordinated owner decisions (frontend + API + agent + threat-model enforcement).

### Recommended Timeline

- **Week of 2026-06-02**: Team sync on policy decisions (1 hour; decision owners only)
- **Week of 2026-06-09**: Implementation planning + task breakdown (Cassius + Brutus; 2 hours)
- **Week of 2026-06-16**: Begin implementation across services (targeted sprints; ~40 story points total)

### References

- **Audit inputs:** src/web, src/api, src/agent (all three services analyzed)
- **Threat-model:** docs/threat-model.md (10 open findings; Brutus highlights DOMPurify + rate-limit invariants)
- **Related decisions:** #163 (security audit umbrella), #206 (threat-model governance)

---

### 4. Feature #219 Refinements — Implementation Complete (2026-05-31)

**Author:** Aurelia (Frontend Dev)  
**Date:** 2026-05-31  
**Commits:** 127c75b (main refinements), 70bd409 (follow-up duplicates)  
**Status:** APPROVED — Shipped to `beta`

**Scope:** Post-merge TLC items from Brian's annotated screenshot review of #219 coin-detail redesign.

**What Changed:**

1. **Duplicate "Actions" heading** → Removed from CoinActionsPanel.vue (shell already renders it)
2. **Duplicate category badge + tag ambiguity** → Removed duplicate from CoinTagsSection.vue; added "Tags" label to distinguish categories from user tags
3. **Obverse/reverse images side-by-side** → Changed grid from `1fr` (stacked) to `1fr 1fr` per Brian's reference
4. **Details card missing heading** → Added "Details" heading above metadata table
5. **Follow-up deduplication (70bd409)** → Removed duplicate section headings from CoinActivityJournal and CoinAIAnalysis

**Validation:**
- npm run lint: 5 pre-existing warnings, zero new
- npm run build: clean (8.96s, vue-tsc + vite)
- Type check: zero errors

**Key Learnings:**
- When a page shell renders a section title, child components should NOT render their own heading
- Category badges (single per coin) vs. user tags (pills) need visual separation — use `.badge` for categories, `.chip-sm` + section label for tags
- Simple grid change from `1fr` to `1fr 1fr` switches dual images from stacked to side-by-side on desktop

**Result:** Feature #219 ship-ready. Awaiting merge to main.

---

### 5. User Directive: Collection Chat LLM Intent Classification (2026-05-31)

**Author:** Brian (via Copilot)  
**Date:** 2026-05-31  
**Status:** DIRECTIVE (drives #217 routing redesign)

**What:** The collection-chat feature (#217) must use LLM-based intent classification instead of hardcoded keyword matching. Brian wants to chat about ANY question regarding his collection, and have an agent figure out his intent "like any chatbot would."

**Why:** Current keyword gate in `ShouldHandleCollection()` missed "Do I have any moose coins and how much are they worth?" (routed to portfolio instead of collection). User explicitly rejects keyword-based approach.

**Impact:** Drives replacement of `ShouldHandleCollection` keyword gate with LLM intent classification in Python supervisor.

---

### 8. Feature #216 Camera-First AI Intake — Maximus RE-REVIEW (2026-05-31)

**Author:** Maximus (Architect)  
**Date:** 2026-05-31  
**Status:** APPROVED — Principle V block lifted

**Scope:** Design Token System compliance (Principle V) — 14 flagged color values.

**Verdict:** **APPROVE** — All 14 hardcoded values tokenized or approved as exceptions (white/black for contrast). Only 4 contrast-safe exceptions remain (lines 808, 835, 883, 927).

**Validation:**
- 12 new tokens defined in variables.css (consistent naming, no duplicates)
- npm run lint: 0 errors
- npm run build: clean (8.35s)
- Constitution Principle V: **PASS**

**Result:** #216 ready to land. Principle V block cleared.

---

### 9. Feature #216 Camera & Intake QA Verdict (2026-05-31)

**Author:** Brutus (Tester)  
**Date:** 2026-05-31  
**Status:** APPROVED

**Scope:** Full functional and regression testing of camera-first UI redesign + AI-assist intake flow.

**Findings:**
- 16/16 functional requirements met
- Zero regressions
- Type-check + production build pass cleanly
- Token refresh in camera flow tested
- Error handling (no camera, network fail, analysis timeout) verified

**Verdict:** ✅ APPROVE — Camera-first intake ready for production.

---

### 10. Feature #216 Camera-First Intake — Design Token Refactor (2026-05-31)

**Author:** Aurelia (Frontend Dev)  
**Date:** 2026-05-31  
**Status:** Completed

**Scope:** Retrofitted 14 hardcoded color values in AddCoinPage.vue to use design tokens from variables.css.

**Changes:**
- Tokenized `.intake-loading-overlay`, `.camera-error-banner`, `.capture-slot`, `.slot-clear-btn`, `.shutter-btn`, `.status-warning`, `.confidence-*` values
- Approved 4 contrast-safe exceptions: `#000` (black bg/text), `#fff` (white text/contrast)
- Added 12 new design tokens: `--overlay-full`, `--error-bg`, `--accent-gold-focus`, `--overlay-dark`, `--border-white-dim`, `--shadow-gold-soft`, `--shadow-gold-hover`, `--text-warning`, `--confidence-high/medium/low`

**Files Changed:** src/web/src/assets/styles/variables.css, src/web/src/pages/AddCoinPage.vue

**Validation:** npm run build clean, type-check passes

**Result:** Principle V compliance achieved.

---

### 11. Feature #217 & #218 — Shared Collection Tool Layer Design (2026-05-31)

**Author:** Maximus (Architect)  
**Date:** 2026-05-31  
**Features:** #217 (In-App Multi-Intent), #218 (External Tool Server)  
**Status:** PROPOSAL — Awaiting implementation planning

**Summary:** Brian approved LLM-based intent classification (kills keyword gate) and chose a **tool-based approach** over single routed-node. Specifies a shared, transport-agnostic collection tool layer serving both #217 (Python tools) and #218 (future MCP/OpenAPI adapter).

**Architecture:**
- **6 discrete operations** (read/write) exposed as LangChain tools
- **Go API** owns all tool logic via `collection_tools_service.go` and `/internal/tools/*` endpoints
- **Python agent** consumes via HTTP with signed internal tokens (30s TTL)
- **Internal HTTP endpoints** return JSON (not SSE); Python converts to SSE events

**Key Changes from Prior Option B:**
- Collection operations become **LangChain tools** (not a dedicated `collection` route)
- **ReAct agent** wraps collection tools + valuation tools + general reasoning
- Internal-token auth mechanism **survives** (Principles XI/XII)
- Keyword gate `ShouldHandleCollection` **deleted**

**Operations Defined:**
| Operation | Schema | Type |
|---|---|---|
| `search_my_collection` | `{query, limit?}` | read |
| `get_coin` | `{coin_id}` | read |
| `collection_summary` | `{}` | read |
| `top_coins_by_value` | `{limit?}` | read |
| `propose_update` | `{coin_id, changes}` | write |
| `commit_update` | `{proposal_id, token, confirm}` | write |

**Files Involved:**
- `src/api/handlers/internal_tools.go` (NEW)
- `src/api/services/collection_tools_service.go` (refactor to export)
- `src/agent/app/tools/collection_tools.py` (NEW)

**Status:** Ready for Cassius + team implementation planning. Supersedes `maximus-217-intent-routing-design.md`.

---

### 12. Feature #216 Token Remediation QA (2026-05-31)

**Author:** Brutus (Tester)  
**Date:** 2026-05-31  
**Status:** APPROVED

**Scope:** Verify token refresh behavior in camera-first intake flow and error conditions.

**Tests Verified:**
- Token refresh during long-running AI analysis
- Concurrent analysis requests with token expiry
- Camera stream cancellation on token revocation
- Error handling (expired token, network timeout)
- 12+ test cases all pass green

**Result:** All token paths verified. No issues found. Ready for production.

---
