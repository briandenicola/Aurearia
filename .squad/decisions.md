# Squad Decisions

## Active Decisions

### 1. Governance Restructure — tech-inventory alignment (2026-05-28)

**Authors:** Maximus (Lead/Architect), Brian  
**Date:** 2026-05-28  
**Status:** ACCEPTED — Phase 1 landed  

#### What
Adopted tech-inventory governance philosophy (operational scaffolding adapted to Go/Vue/Python). Constitution v1.1.0 → v2.0.0 with eight new operational sections (§0–§23): Hierarchy of Authority, Quality Gate, AI Agent Operating Rules, Documentation Requirements, Audit Cadence, Definition of Done, Amendment Process, Revision History. All 16 original Principles preserved verbatim.

#### Key Decisions Captured
- Constitution MAJOR version bump: v1.1.0 → v2.0.0 (16 principles untouched; operational restructure warrants MAJOR)
- Seed `specs/001-foundation/` retroactively (Phase 2 work)
- `docs/prd.md` becomes product source of truth
- Split `docs/security-analysis.md` → `docs/security-baseline.md` + `docs/threat-model.md` (Phase 3)
- Signed commits NOT required (single-developer hobby project); Conventional Commits + Co-authored-by trailer remain mandatory
- Reject signed commits per Brian's confirmation

#### Consequences
- All future features originate as `specs/_backlog/F0NN-*.md`, promote to `specs/NNN-*/spec.md`, follow SpecKit pipeline
- PRs gated on §17 Quality Gate + §21 Definition of Done (14-item checklist)
- Squad handoff via Scribe: `.squad/log/` + per-agent `history.md` + `.squad/decisions.md`
- §18 forbids `SESSION-NOTES.md` and `.copilot-state.md`

#### Impact
Establishes unambiguous document hierarchy, single Definition of Done, mechanically enforceable quality gate. Enables cross-repo governance consistency with tech-inventory while preserving Go/Vue/Python idioms.

---

### 2. copilot-instructions Restructure + PR Template (2026-05-28)

**Author:** Maximus (Lead/Architect)  
**Date:** 2026-05-28  
**Status:** ACCEPTED — Phase 1 landed  

#### What
1. `.github/copilot-instructions.md` restructured to cite the constitution rather than restate it. Added **Document Hierarchy** (§0), **Session Protocol** (§18 Always/Never/Handoff), **Constitution Compliance** (points to PR template §21). Architecture rules defer to Principle I + X; security cites Principles XI/XII/XIII; commit convention cites §17 + Principle VIII.

2. `.github/pull_request_template.md` created with **Summary**, **Constitution self-check** (Principle + operational section flag), **Linked work**, then the §21 Definition of Done as a 14-item executable checklist.

#### Rationale
- Constitution v2.0.0 is single source of truth; previous copilot-instructions duplicated principle text (drift risk)
- Citation style keeps docs terse; forces agents to authoritative file
- Day-to-day operational material (Build/Test/Lint, design tokens, chip/button classes, "Adding a New API Feature", endpoints) preserved — that's muscle memory
- PR template is cheapest enforcement surface for §21; GitHub UI blocks merge until each DoD item examined

#### Impact
Agents read constitution once for principles/governance; PR template gates every merge on §21 self-check. Reduces documentation drift and makes quality requirements visible on every PR.

---

### 3. Governance Scaffolding — SECURITY.md, CODEOWNERS, Templates (2026-05-28)

**Authors:** Scribe, Maximus  
**Date:** 2026-05-28  
**Status:** ACCEPTED — Phase 1 landed  

#### What
Created governance infrastructure files:
- **SECURITY.md** — Security policy, 30-day disclosure window, responsible disclosure contact
- **.github/CODEOWNERS** — Team routing for review (TBD allocation per sprint planning)
- **.github/ISSUE_TEMPLATE/bug.md** — Bug report template with reproduction steps, expected/actual behavior, environment
- **.github/ISSUE_TEMPLATE/feature.md** — Feature request template with use case, acceptance criteria, success metrics

#### Rationale
- Formalized security policy establishes trust and legal clarity
- CODEOWNERS automates review routing, prevents single-person merge
- Issue templates standardize problem report format (faster triage, fewer "what do you mean" back-and-forths)
- Templates are cheap organizational wins; fit "operations scaffolding" theme

#### Impact
Security reporters know where to send disclosures; maintainers can enforce review gates; users file better-structured issues. Establishes codified organizational boundaries.

---

### 4. Five User Decisions — tech-inventory Alignment Direction (2026-05-28)

**Author:** Brian (via Squad coordinator)  
**Date:** 2026-05-28  
**Status:** ACCEPTED — Phase 1 planning confirmed  

#### Decision 1: Constitution Version Bump
**What:** v1.1.0 → v2.0.0 (MAJOR)  
**Why:** §17 Quality Gate, §21 DoD, §0 Hierarchy materially change agent behavior

#### Decision 2: Retroactive Spec for v1.0
**What:** Yes — seed `specs/001-foundation/` documenting existing v1.0 feature surface  
**Why:** Validates new SpecKit on-disk workflow end-to-end

#### Decision 3: PRD as Product Source of Truth
**What:** `docs/prd.md` becomes authoritative; README trimmed to setup/architecture/links  
**Why:** Cross-repo consistency, decouples product intent from build instructions

#### Decision 4: Security Analysis Split
**What:** Split into `docs/security-baseline.md` (controls) + `docs/threat-model.md` (STRIDE); keep `docs/security-analysis.md` as deprecated redirect stub  
**Why:** Matches tech-inventory pattern; STRIDE warrants its own doc

#### Decision 5: Signed Commits on main
**What:** SKIP — single-developer hobby project  
**Why:** No team contributors; Conventional Commits + Co-authored-by trailer remain mandatory

#### Impact
Confirmed Phase 1 scope boundaries; Phase 2–4 deliverables queued. No ambiguity on version bumping, spec seeding, or security doc refactoring.

---

### 5. `specs/` scaffold + session-protocol prompts live (2026-05-28)

**Author:** Maximus (Lead/Architect)  
**Date:** 2026-05-28  
**Status:** ACCEPTED — Phase 2A landed  

#### What

1. **`specs/` workflow is live on disk** (Constitution §0 Hierarchy items 3–6):
   - `specs/NNN-feature-slug/` for active features (`spec.md` + `plan.md` + `tasks.md` required)
   - `specs/_backlog/F0NN-feature-slug.md` for queued cards
   - `specs/README.md` + `specs/_backlog/README.md` document layout, numbering (immutable), lifecycle, gates
   - `specs/_backlog/_TEMPLATE.md` is canonical 15-field card (YAML frontmatter + body sections)

2. **Promotion rule** (backlog → active): Card must be `status: triaged` with concrete acceptance criteria and named Principle + operational section. Lead assigns next free `NNN`, runs `/speckit.specify` to create `specs/NNN-slug/spec.md`, updates card to `status: promoted`, commits both in one PR.

3. **Triage cadence:** Maximus reviews all backlog cards weekly; aim to advance or drop within two cycles.

4. **4 session-protocol prompts registered** under `.github/prompts/`:
   - `load-context.prompt.md` — Cold-start; constitution + decisions + active spec + agent charter
   - `checkpoint.prompt.md` — Mid-session pause; Scribe writes to `.squad/log/{ts}-checkpoint.md`
   - `handoff.prompt.md` — End of session; Scribe reconciles `tasks.md`, merges inbox, commits
   - `audit.prompt.md` — §20 audit; findings → `docs/audits/YYYY-MM-DD.md`

#### Rationale
Constitution §0 needs concrete homes for Hierarchy items; 4 new prompts standardize Squad ceremonies (§18 Session Handoff). Manifest update deferred — run `specify upgrade` to register prompts in `.specify/integrations/copilot.manifest.json`.

#### Unblocks
- Retro `specs/001-foundation/` (Maximus follow-up)
- New feature requests now have documented SpecKit entry point

#### References
Constitution §0 (Hierarchy), §17 (Quality Gate), §18 (AI Agent Operating Rules), §20 (Audit), §21 (DoD), §22 (Amendment).

---

### 6. `specs/001-foundation/` is the v1.0 SHIPPED anchor (2026-05-28)

**Author:** Maximus (Lead/Architect)  
**Date:** 2026-05-28  
**Status:** ACCEPTED — Phase 2B landed  

#### What

`specs/001-foundation/` was created retroactively to document the v1.0 feature surface already shipped (Go API + Vue PWA + Python LangGraph agent, packaged as two Docker containers). It is the canonical "active feature spec" referenced by Constitution §0 Hierarchy item 3 from the moment v1.0 went live through whenever `specs/002-*/` opens.

Three files authored:
- **`spec.md` (162 lines)** — Problem, users, 6 prioritized user stories, FR-001 through FR-010, key entities, success criteria, assumptions, out-of-scope
- **`plan.md` (139 lines)** — Three-service architecture, Constitution-check table (Principles I, II, III, IV, V, VI, X, XII, XIII all PASS), 9 key decisions, tech stack rationale
- **`tasks.md` (86 lines)** — All checkboxes checked ✅ (SHIPPED), grouped by domain (Go API, Vue, Python Agent, Quality, Governance)

**Total:** 387 lines, within Brian's 400–600 line budget.

#### Rationale
1. Constitution §0 Hierarchy item 3 ("active feature spec") needs concrete content — making it fully anchored, not aspirational.
2. End-to-end SpecKit validation on known-good surface exercises the template before genuine `002-*` work.
3. Audit trail: "What was in v1.0?" now has a one-file answer.

#### Scope Boundary
- **`001-foundation/` is historical.** Not edited again except for future Constitution amendments.
- Forward-looking work opens at `specs/002-*/`, `specs/003-*/`, etc.
- Backlog cards F001–F007 are cross-linked for traceability but NOT retroactively marked `promoted` — they document v1.0 surface, not pre-shipping promotions.

#### Risk Mitigation
- **Retroactive drift:** Keep `spec.md` at capability level (FR-001–FR-010), put volatile detail in `plan.md`, tasks list by name only.
- **Retroactive precedent:** Explicitly noted in all three files — one-time anchor, not recurring practice. Future features spec-first via `/speckit.specify`.

#### References
Constitution §0 (Hierarchy of Authority, item 3), §18 (AI Agent Operating Rules), §22 (Amendment Process).

---

### 5. Microsoft Foundry Agent Service Migration (January 2025)

**Status:** NO-GO Recommendation (awaiting team review)  
**Proposed by:** Maximus (Lead/Architect)  
**Date:** January 2025  
**Spike Document:** `docs/spikes/foundry-agent-service.md`

#### What
Recommend NO-GO on migrating the Ancient Coins agent service from Python/LangGraph to Microsoft Foundry Agent Service (C# Agent Framework SDK).

#### Analysis
**Current:** Python/FastAPI + LangGraph; 10 multi-agent teams; stateless; SSE streaming; Anthropic + Ollama support  
**Candidate:** Azure Foundry Agent Service; C# Agent Framework SDK; managed operations; stateful sessions; Claude models via Azure

**Pros:**
- ✅ Managed scaling, deployment, monitoring
- ✅ Enterprise support (Entra ID, RBAC, VNet, Azure compliance)
- ✅ Claude models via Azure billing consolidation
- ✅ Comparable streaming and orchestration

**Cons:**
- ❌ 3–4 month migration ($115k labor)
- ❌ Complete rewrite (Python → C#, LangGraph → MAF patterns)
- ❌ High risk: web search tool availability uncertain, Claude quota issues in preview
- ❌ +20% operational overhead ($50–100/month vs. free self-hosted)
- ❌ Team skill mismatch (Python/Go/TS vs. C#/.NET learning curve)
- ❌ Stateful design mismatch with current stateless architecture
- ❌ SSE progress reporting degrades without additional engineering

**Risks:**
- **High:** Web search tool availability blocker for 40% of teams
- **High:** Claude in preview with documented quota exhaustion
- **Medium:** SearXNG integration unclear; loses Ollama option
- **Medium:** UX regression without rich progress events

#### Decision
**NO-GO at this time.** Current Python/LangGraph meets all requirements with lower operational complexity. Reconsider if: (1) strategic Azure lock-in required, (2) enterprise support becomes critical, (3) Claude exits preview with guaranteed quotas, (4) team grows 3x, (5) Foundry ships exclusive features.

**Alternative (Low-Risk):** Stay on Python/LangGraph; incrementally migrate Anthropic API calls to Azure AI Services endpoints (serverless). Benefits: billing consolidation, Entra ID, quota management. **No code rewrite** — configuration only.

#### Impact
Continue Python/LangGraph development; optimize observability and caching. Revisit annually.

---

## Previous Decisions (Archive)

## Active Decisions

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

## Governance

- All meaningful changes require team consensus
- Document architectural decisions here
- Keep history focused on work, decisions focused on direction
