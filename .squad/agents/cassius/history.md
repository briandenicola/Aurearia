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

- **2026-04-24:** Fixed two P0 issues from the code review: (1) Added `sync.Once` guards to both `ValuationScheduler.Stop()` and `AvailabilityScheduler.Stop()` to prevent double-close panics on the stop channel. (2) Added a defense-in-depth column allowlist to `CoinRepository.Suggestions()` so the repo validates the column name before interpolating it into SQL, matching the handler's existing whitelist. All tests pass.

- **2026-04-25:** Completed P2 #34 and P2 #35. (1) Added handler-level input validation to the coins List endpoint: page must be ≥1, limit must be 1–100, sort field is checked against an allowlist (defense-in-depth against SQL injection), and order must be "asc" or "desc". Invalid input now returns HTTP 400 with a clear message instead of being silently corrected by the repository. (2) Fixed orphan file risk in `ImageService.UploadImage()`: if the DB insert fails after the file is written to disk, the file is now cleaned up via `os.Remove()` to prevent orphans.

- **2026-05-21:** Built auction ending scheduler following the exact pattern of `availability_scheduler.go`. Daily cadence (default 08:00, 1440min interval) checks for auction lots in BIDDING status whose sale date is today. Sends consolidated Pushover notification per user listing all ending auctions (auction house, sale, lot #). Uses in-memory idempotency tracking (`lastNotified map[uint]string`) to prevent duplicate notifications within the same day. Added `GetEndingToday()` repository method with filtering by status and date range. Wired in `main.go` alongside existing schedulers. New settings keys: `AuctionEndingCheckEnabled`, `AuctionEndingCheckStartTime`, `AuctionEndingCheckInterval` — match naming convention of wishlist scheduler. All tests pass including two new auction repository tests.

- **2026-06-10:** Added manual run trigger and per-run logging to auction ending scheduler to bring it to parity with Valuation and Wishlist schedulers. Created `models/auction_ending_run.go` model (fields: ID, TriggerType [scheduled|manual], TriggerUserID, Status [running|success|error], LotsChecked, AlertsSent, DurationMs, StartedAt, CompletedAt, ErrorMessage). Created `repository/auction_ending_repository.go` with methods: CreateRun, CompleteRun, ListRuns (paginated), GetRunByID, PruneOldRuns (keeps 100 most recent). Refactored `services/auction_ending_scheduler.go` to log every run (scheduled or manual) — extracted `runCycleWithTrigger()` that creates a run row, executes the check, counts results, finalizes the run with status + durations + error. Added `RunNow(triggerUserID)` method for manual invocation. Created `handlers/auction_ending_admin.go` with two endpoints: `GET /admin/auction-ending-runs` (paginated run history) and `POST /admin/auction-ending/run` (manual trigger). Wired in `main.go`: created scheduler early (before routes) so it can be passed to admin handler, registered admin routes. Updated `README.md` Background Schedulers section to document manual trigger and run log table. Added `models.AuctionEndingRun{}` to AutoMigrate in `database/database.go`. All architecture tests pass, `go vet` clean, `go test -v ./...` passed. Endpoints: `GET /api/admin/auction-ending-runs?page=1&limit=20` (returns `{runs, total, page, limit}` where each run has `{id, triggerType, triggerUserId, status, lotsChecked, alertsSent, durationMs, startedAt, completedAt, errorMessage, createdAt}`), `POST /api/admin/auction-ending/run` (returns `{runId, lotsChecked, alertsSent, status, durationMs}`).

## Team Updates

- **2026-05-22:** Worked with Aurelia (frontend) and Brutus (testing) on auction-ending manual-run feature. Brutus's comprehensive test suite (16 tests across repository and handler) APPROVED implementation. Aurelia implementing corresponding UI with "Run Now" button and recent runs log; follow-up fixup aligning endpoint URLs.

