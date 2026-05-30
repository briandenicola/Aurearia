# Squad Decisions Archive

Archived decisions from `decisions.md` older than 30 days.

## Archived Decisions

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
