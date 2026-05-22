# Squad Decisions

## Active Decisions

### 1. Full-System Architecture Document

**Author:** Maximus (Lead/Architect)  
**Date:** 2025-07-18  
**Status:** Implemented  

#### What
Rewrote `docs/ARCHITECTURE.md` from a Go-API-only document (~214 lines) to a comprehensive full-system architecture reference (~761 lines) covering all three services.

#### Why
The previous doc only covered the Go API layered architecture. Missing: frontend architecture, Python agent service, data flow diagrams, database schema, auth flow details, agent integration pattern, background schedulers, build pipeline, configuration reference, and design decision rationale.

#### Scope
- System overview and container topology diagram
- Go API: layers, rules, package map, DI wiring, route groups, scopes, arch tests
- Vue 3: structure, routing, Pinia stores, API client (401 refresh queue), composables, PWA config
- Python agent: endpoints, supervisor routing, 11 team pipelines, LLM provider abstraction, SSE streaming
- Data flow diagrams: standard request, agent chat SSE, auth flow, availability check
- Database schema: 26 models across 6 categories
- Authentication: JWT + API key + WebAuthn details
- Background schedulers: availability + valuation
- Docker multi-stage build for both containers
- Configuration reference (env vars + runtime settings)
- Key design decisions with rationale

#### Impact
All team members and AI agents now have a single reference for system architecture. No code changes — documentation only.

---

### 2. Code Review & Quality Assessment (2026-04-24)

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

### 3. P0 Fixes — Admin Route Guard & v-html (2026-07-22)

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

### 4. Activity Journal Scroll Limit & Auction Schedule UI (2026-05-01)

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

### 5. Auction Ending Manual Trigger & Run Log — Backend Implementation (2026-06-10)

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

### 6. Auction Ending Manual Trigger & Run Log — Frontend UI (2026-05-21)

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

### 7. Auction Ending Manual Trigger & Run Log — Test Coverage (2026-05-22)

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

## Governance

- All meaningful changes require team consensus
- Document architectural decisions here
- Keep history focused on work, decisions focused on direction
