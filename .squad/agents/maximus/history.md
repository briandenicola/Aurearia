# Project Context

- **Owner:** Brian
- **Project:** Ancient Coins — full-stack PWA for managing a personal ancient coin collection
- **Stack:** Go 1.26 / Gin / GORM / SQLite (API), Vue 3 / TypeScript / Pinia / Vite (Frontend), Python 3.12 / FastAPI / LangGraph (Agent), Docker
- **Architecture:** Layered — Handler → Service → Repository → Database. Enforced by architecture_test.go.
- **Created:** 2026-04-24

## Learnings

<!-- Append new learnings below. Each entry is something lasting about the project. -->

- **2025-07-18**: Rewrote `docs/ARCHITECTURE.md` from API-only doc (214 lines) to full-system architecture (761 lines) covering all three services, data flows, DB schema, auth, agent integration, schedulers, build pipeline, and design decisions. Derived entirely from codebase inspection.
- Key file paths: `src/api/main.go` (composition root, ~400 lines of DI wiring), `src/agent/app/supervisor.py` (11-team LLM router), `src/web/src/api/client.ts` (Axios + SSE + 401 refresh queue), `src/api/services/agent_proxy.go` (SSE proxy pattern).
- The system has 26 auto-migrated GORM models, 22 repository files, 17 service files, 25 handler files, 21 Vue pages, and 10 composables.
- Two background goroutine schedulers (availability + valuation) run with configurable anchor times and intervals from DB settings.
- Auth supports 3 methods: JWT (15min access + 30d refresh with rotation), API keys (SHA-256 hashed), and WebAuthn/passkeys.
- **2026-04-24**: Full architecture & code quality review. Graded 11 areas (A- to C+). Key findings: DI is undermined by 3 package-level globals (`AppLogger`, `GetSetting`, `cancelMap`); `social.go` silently drops 7+ errors; frontend has 3 god-pages (1200-1400 lines each); Python agent lacks tests for supervisor routing and team pipelines. Error handling is the weakest area (C+). Documentation is the strongest (A-). Created 20-item prioritized backlog in `.squad/decisions/inbox/maximus-code-review.md`.
