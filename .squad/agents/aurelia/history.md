# Project Context

- **Owner:** Brian
- **Project:** Ancient Coins — full-stack PWA for managing a personal ancient coin collection
- **Stack:** Go 1.26 / Gin / GORM / SQLite (API), Vue 3 / TypeScript / Pinia / Vite (Frontend), Python 3.12 / FastAPI / LangGraph (Agent), Docker
- **Architecture:** Layered — Handler → Service → Repository → Database. Enforced by architecture_test.go.
- **Created:** 2026-04-24

## Learnings

- **2025-07-18:** Maximus completed comprehensive `docs/ARCHITECTURE.md` covering full-system architecture (Go API, Vue frontend, Python agent service, data flows, DB schema, auth, agent integration, Docker, design decisions). This is the authoritative reference for all agents and team members.

- **2026-04-24:** Completed deep frontend code quality review. Key findings: (1) `v-html` XSS risk in AI content rendering — needs DOMPurify, (2) 15+ files have uncleared setTimeout/setInterval — memory leak pattern is widespread, (3) Router has no admin role guard — only UI-hidden, (4) Auth store drifts from localStorage after silent token refresh, (5) PWA icons missing from public/ breaking installability, (6) Accessibility is the weakest area — almost no ARIA, no focus traps, clickable divs without keyboard support. Three pages exceed 1200+ lines and need splitting. Overall grade: B-. 19 backlog items created in `.squad/decisions/inbox/aurelia-code-review.md`.

- **2025-07-22:** P0 security fixes. (1) Confirmed all 4 `v-html` bindings already use DOMPurify — `CoinSearchChat.vue` via `formatMessage()` in `useCoinSearchChat.ts` (strict allowlist), `CoinDetailPage.vue` via `DOMPurify.sanitize(md.render(...))` on all three AI analysis computeds. No additional work needed. (2) Added admin role guard to router: `/admin` route now has `meta.requiresAdmin`, `beforeEach` checks `auth.isAdmin` and redirects non-admins to collection page. TypeScript + production build verified clean.

<!-- Append new learnings below. Each entry is something lasting about the project. -->
