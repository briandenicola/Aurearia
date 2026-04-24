# Project Context

- **Owner:** Brian
- **Project:** Ancient Coins — full-stack PWA for managing a personal ancient coin collection
- **Stack:** Go 1.26 / Gin / GORM / SQLite (API), Vue 3 / TypeScript / Pinia / Vite (Frontend), Python 3.12 / FastAPI / LangGraph (Agent), Docker
- **Architecture:** Layered — Handler → Service → Repository → Database. Enforced by architecture_test.go.
- **Created:** 2026-04-24

## Learnings

- **2025-07-18:** Maximus completed comprehensive `docs/ARCHITECTURE.md` covering full-system architecture (Go API, Vue frontend, Python agent service, data flows, DB schema, auth, agent integration, Docker, design decisions). This is the authoritative reference for all agents and team members.

- **2026-04-24:** Full test coverage review completed. Go API: 18 tests pass (2 arch, 9 repo, 7 service), measured coverage 3.5-4.6%. Only CoinRepository and CoinService have tests — 20+ repos, 15+ services, all handlers untested. Python agent: 31 tests pass, covers models/retry/streaming/validation but zero team pipeline tests. Frontend: zero test files, no framework configured, 60+ components untested. Architecture tests enforce no-database-import and no-raw-SQL-in-handlers but not the full import matrix. Auth/middleware is the highest-risk gap. Report written to `.squad/decisions/inbox/brutus-code-review.md` with 25 prioritized backlog items.

- **2026-04-24:** P0 test gap closure completed. Added 34 new tests across 4 files covering the entire auth/security surface: `services/auth_service_test.go` (15 tests: registration, authentication, JWT generation, token rotation, password hashing), `middleware/auth_test.go` (10 tests: JWT valid/missing/malformed/expired, API key valid/invalid/revoked, query param token), `handlers/auth_handler_test.go` (13 tests: register/login/refresh/setup HTTP-level), `handlers/coin_handler_test.go` (11 tests: CRUD with ownership enforcement). All 52 Go tests pass. Test pattern: real in-memory SQLite via glebarez/sqlite, httptest+Gin for handler/middleware tests, real bcrypt for password tests.

- **2026-04-24:** P1 test items completed. (1) Vitest configured for Vue frontend: `vitest.config.ts` with jsdom, globals, `npm run test` / `npm run test:watch` scripts added, 7 passing tests on CollectionPagination component. (2) ValuationParser: 18 table-driven tests covering JSON parse, regex fallback, edge cases (zero/negative/malformed/empty). (3) SettingsService: 10 tests covering get/set/defaults/empty-value behavior using in-memory SQLite. (4) Architecture import matrix test (`TestPackageImportMatrix`) added to `architecture_test.go` — enforces handlers→services/repo/models, services→repo/models, repo→models, models→stdlib-only via AST parsing. All 77 Go tests and 7 Vitest tests pass.

- **2026-04-24:** SettingsService currently uses package-level `settingsDB` global (not constructor injection). Tests save/restore the global. When Cassius completes the DI refactor, settings tests will need updating to use the new constructor pattern.

<!-- Append new learnings below. Each entry is something lasting about the project. -->
