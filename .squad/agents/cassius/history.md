# Project Context

- **Owner:** Brian
- **Project:** Ancient Coins backend — Go 1.26 / Gin / GORM / SQLite
- **Architecture:** Layered Handler → Service → Repository → Database with constructor injection and architecture tests.

## Core Context

- Cassius owns backend implementation. Durable backend rules: thin handlers, service-owned business logic, repository-owned GORM queries, scopes for ownership/public filters, sentinel service errors, Swagger annotations, and DI wiring in `main.go`.
- Scheduler/run-log pattern established across valuation, wishlist/availability, auction-ending, and related admin surfaces: configurable settings, manual trigger, run history table, and production diagnostics where needed.
- Time-sensitive auction queries use rolling `(now, now+24h]` windows, explicit NULL guards, and case-insensitive status comparison. Real-data diagnostics should accompany query fixes.
- Security/backend patterns: validate ownership before CPU/memory-heavy decode operations; mock httpx response methods synchronously in Python tests; circle image clipping lives in stdlib-only `src/api/capture/` and is gated to obverse/reverse uploads when `circleClip=true`.

## Recent Updates

- **2026-05-31:** #217 Go shared collection tool layer completed: internal token service/middleware, six internal tool endpoints, keyword gate removed, confirm-gated write flow preserved.
- **2026-05-31:** #217 Python ReAct collection agent completed end-to-end with LangChain tools calling Go internal endpoints via short-lived internal token; compound collection/value questions now compose within one reasoning turn.
- **2026-06-01:** #218 external tool server foundational stack implemented: API key capabilities, enablement toggle, capability middleware, per-key rate limit, `/api/v1/tools` route group, handlers, OpenAPI discovery, and external commit journal metadata. Build/vet/test passed.
- **2026-06-01:** Collection chat multi-container callback issue documented. `AGENT_INTERNAL_CALLBACK_URL` must point from agent container to API service (e.g. `http://coins:8080`), not default localhost; startup warning added for release+localhost.
- **2026-06-01:** v1→v2 migration audit found only additive schema changes; AutoMigrate/backfill safe and rollback-safe.
