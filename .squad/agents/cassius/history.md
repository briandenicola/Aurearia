# Project Context

- **Owner:** Brian
- **Project:** Ancient Coins — full-stack PWA for managing a personal ancient coin collection
- **Stack:** Go 1.26 / Gin / GORM / SQLite (API), Vue 3 / TypeScript / Pinia / Vite (Frontend), Python 3.12 / FastAPI / LangGraph (Agent), Docker
- **Architecture:** Layered — Handler → Service → Repository → Database. Enforced by architecture_test.go.
- **Created:** 2026-04-24

## Learnings

- **2025-07-18:** Maximus completed comprehensive `docs/ARCHITECTURE.md` covering full-system architecture (Go API, Vue frontend, Python agent service, data flows, DB schema, auth, agent integration, Docker, design decisions). This is the authoritative reference for all agents and team members.

- **2026-04-24:** Completed deep backend code quality review of all Go source in `src/api/`. Overall grade B-. Key findings: (1) `settings_service.go` bypasses the repository layer with a global `*gorm.DB`; (2) middleware/auth.go does direct DB access; (3) both schedulers have double-close panic risk; (4) business logic leaks into `analysis.go`, `agent.go`, `coins.go`, and `admin.go` handlers; (5) error handling is inconsistent — many repos/services silently swallow errors; (6) input validation is thin across handlers and models. Full report in `.squad/decisions/inbox/cassius-code-review.md` with 20 prioritized backlog items.

<!-- Append new learnings below. Each entry is something lasting about the project. -->
