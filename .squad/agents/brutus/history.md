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

- **2026-04-24:** P1 #22 and #23 completed. Auth store tests (17 tests in `src/web/src/stores/__tests__/auth.test.ts`): initial state, localStorage persistence/restore, login/register flows, logout clears all state, computed isAuthenticated/isAdmin, double login overwrites, logout-when-logged-out safety, token refresh sync callback. API client tests (24 tests in `src/web/src/api/__tests__/client.test.ts`): sanitizeCoin normalizes empty/undefined→null, preserves 0 and valid numbers, date conversion to RFC3339, immutability; JWT request interceptor; 401 response interceptor (refresh attempt, no-refreshToken clear, non-401 passthrough, no double-retry); 8 API wrapper URL/param correctness tests. All 48 Vitest tests pass, `npm run build` clean. Key pattern: mock axios via `vi.mock` with interceptor capture; mock localStorage via `vi.stubGlobal`; test sanitizeCoin indirectly through createCoin/updateCoin since it's not exported.

<!-- Append new learnings below. Each entry is something lasting about the project. -->

- **2026-05-21:** Auction-ending scheduler now available for testing. Cassius implemented backend (`services/auction_ending_scheduler.go`) with 3 new settings keys (enabled, start-time, interval). Aurelia completed admin panel UI in `AdminSchedulesSection.vue`. Pushover integration allows configurable alerts for upcoming auction ending dates. Ready for E2E test scenarios.

- **2026-05-22:** Completed test coverage for auction ending manual-run and run log features. Created 16 new tests across 2 files: (1) `repository/auction_ending_repository_test.go` — 10 tests covering CreateRun, CompleteRun (success/error), ListRuns (pagination, empty, limit edge cases), GetRunByID (success/not found); (2) `handlers/auction_ending_admin_test.go` — 6 tests covering manual trigger endpoint (admin/user/no-auth), list runs endpoint (admin/pagination/no-auth). All tests pass. Cassius's implementation is clean: model (`AuctionEndingRun`), repository (`AuctionEndingRepository`), handler (`AuctionEndingAdminHandler`), scheduler `RunNow()` method, and wiring in `main.go` all follow established patterns. Test approach mirrors valuation run test pattern (in-memory SQLite, httptest+Gin, JWT middleware mocks). Repository tests verify CRUD, pagination defaults, empty slice vs nil. Handler tests verify auth/admin guards, response shape, pagination query params. No gaps found.

## Team Updates

- **2026-05-22:** Led quality assurance on auction-ending manual-run feature. Validated Cassius's backend implementation (16 comprehensive tests across repository and handler layers covering CRUD, pagination edge cases, error scenarios, and authorization guards). **Verdict: APPROVED** — implementation is production-ready, follows established patterns, no blocking issues. Coordinated with Aurelia on frontend integration; flagged endpoint URL alignment issue for follow-up fixup.

