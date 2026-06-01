# Squad Decisions Archive

Archived decisions (older than 30 days from 2026-05-31).

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

