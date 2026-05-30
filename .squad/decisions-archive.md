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

---

### 1. Governance Restructure — tech-inventory alignment (2026-05-28)

**Authors:** Maximus (Lead/Architect), Brian  
**Date:** 2026-05-28  
**Status:** ACCEPTED — Phase 1 landed  

Adopted tech-inventory governance philosophy (operational scaffolding adapted to Go/Vue/Python). Constitution v1.1.0 → v2.0.0 with eight new operational sections (§0–§23): Hierarchy of Authority, Quality Gate, AI Agent Operating Rules, Documentation Requirements, Audit Cadence, Definition of Done, Amendment Process, Revision History. All 16 original Principles preserved verbatim.

Key Decisions: Constitution MAJOR version bump (16 principles untouched, operational restructure warrants MAJOR); Seed `specs/001-foundation/` retroactively; `docs/prd.md` becomes product source of truth; Split legacy security review into three files; Signed commits NOT required (single-developer hobby project); Conventional Commits + Co-authored-by trailer remain mandatory.

Impact: Establishes unambiguous document hierarchy, single Definition of Done, mechanically enforceable quality gate.

---

### 2. copilot-instructions Restructure + PR Template (2026-05-28)

**Author:** Maximus (Lead/Architect)  
**Date:** 2026-05-28  
**Status:** ACCEPTED — Phase 1 landed  

Restructured `.github/copilot-instructions.md` to cite the constitution rather than restate it. Added Document Hierarchy (§0), Session Protocol (§18), Constitution Compliance (§21). Created `.github/pull_request_template.md` with Summary, Constitution self-check, Linked work, then §21 Definition of Done as 14-item executable checklist.

Impact: Agents read constitution once; PR template gates every merge on §21 self-check. Reduces documentation drift.

---

### 3. Governance Scaffolding — SECURITY.md, CODEOWNERS, Templates (2026-05-28)

**Authors:** Scribe, Maximus  
**Date:** 2026-05-28  
**Status:** ACCEPTED — Phase 1 landed  

Created governance infrastructure: SECURITY.md (30-day disclosure window), .github/CODEOWNERS (review routing), bug.md and feature.md issue templates.

Impact: Security reporters know where to send disclosures; maintainers enforce review gates; users file better-structured issues.

---

### 4. Five User Decisions — tech-inventory Alignment Direction (2026-05-28)

**Author:** Brian (via Squad coordinator)  
**Date:** 2026-05-28  
**Status:** ACCEPTED — Phase 1 planning confirmed  

Five user decisions: Constitution v1.1.0 → v2.0.0 (MAJOR); Yes to retroactive specs/001-foundation/; docs/prd.md as product source of truth; Security analysis split; Signed commits SKIP (single-developer project).

Impact: Confirmed Phase 1 scope boundaries; Phase 2–4 deliverables queued.

---

### 5. `specs/` scaffold + session-protocol prompts live (2026-05-28)

**Author:** Maximus (Lead/Architect)  
**Date:** 2026-05-28  
**Status:** ACCEPTED — Phase 2A landed  

specs/ workflow live on disk (specs/NNN-feature-slug/ for active features, specs/_backlog/F0NN-*.md for queued cards). Promotion rule: triaged card with concrete acceptance criteria → specs/NNN-slug/spec.md. Four session-protocol prompts under .github/prompts/ (load-context, checkpoint, handoff, audit).

Impact: Constitution §0 Hierarchy items have concrete homes; Squad ceremonies standardized.

---

### 6. `specs/001-foundation/` is the v1.0 SHIPPED anchor (2026-05-28)

**Author:** Maximus (Lead/Architect)  
**Date:** 2026-05-28  
**Status:** ACCEPTED — Phase 2B landed  

specs/001-foundation/ created retroactively to document v1.0 feature surface: spec.md (162 lines), plan.md (139 lines), tasks.md (86 lines, all ✅ SHIPPED). Total 387 lines within budget.

Rationale: Constitution §0 Hierarchy item 3 needs concrete content; end-to-end SpecKit validation; audit trail for "What was in v1.0?"

Scope boundary: 001-foundation/ is historical; forward-looking work opens at specs/002-*/, specs/003-*/, etc.

Impact: Establishes v1.0 feature surface as reference point for future iterations.

---

### 7. Code Review & Quality Assessment (2026-04-24)

**Authors:** Maximus (Architect), Cassius (Backend), Aurelia (Frontend), Brutus (Testing)  
**Date:** 2026-04-24  
**Status:** Assessed — Backlog Created  

Comprehensive review of all three services. Generated 77 backlog items across P0–P3 priorities. Key findings:
- Architecture (Grade: B+): Clean 3-service separation; 3 package-level globals undermining DI
- Backend (Grade: B-): Most handlers thin; some logic leaks; sentinel errors used in 4 services; many silently drop errors
- Frontend (Grade: B-): Good Composition API; some god-components; TypeScript strong; accessibility D+; PWA quality C+
- Testing (Grade: D): 3.5-4.6% Go coverage; zero frontend tests; 31 Python tests but zero team pipeline tests

Impact: Establishes baseline quality metrics and prioritized backlog.

---

### 8. Phase 3b operational scaffolding: CI workflow + security scans (2026-05-28)

**Author:** Cassius (Backend Dev)  
**Date:** 2026-05-28  
**Status:** ACCEPTED — Phase 3b landed  

Kept workflow file as `.github/workflows/ci.yml` while renaming the workflow itself to "Quality Gate" (for stability). Added separate `.github/workflows/security-scan.yml` for advisory scans. Expanded Quality Gate to full §17 coverage (Go/Vue/Python/OpenAPI checks).

Decision: Kept `ci.yml` filename while renaming workflow to "Quality Gate" to avoid cross-workstream churn.

Impact: Quality Gate enforces §17 requirements mechanically; security scans decouple from main CI; OpenAPI export to docs/openapi.json.

