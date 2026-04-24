### 2026-04-24: Backend Code Quality Review
**By:** Cassius (Backend Dev)

---

## Grades

| Area | Grade | Notes |
|------|-------|-------|
| Handler Quality | B | Most handlers are thin. `analysis.go`, `agent.go`, `admin.go`, and `coins.go` leak business logic. Several swallow errors silently. |
| Service Quality | B- | Good sentinel errors in auth/social/auction. `settings_service.go` bypasses repository pattern entirely. Schedulers have double-close panic risk. Many ignored errors across valuation/availability. |
| Repository Quality | B | Scopes used well. Pagination on major list endpoints. Some non-transactional multi-step writes and chatty loops. Error handling inconsistent. |
| Model Design | B | Clean stdlib-only imports. GORM tags solid. Validation tags are thin — many fields lack required/range/enum constraints. |
| Error Handling | C+ | Sentinel errors used in ~4 services. Many repos and services silently ignore errors. Fire-and-forget goroutines lose error context. Internal errors not leaked to clients (good). |
| Input Validation | C | Handler-level validation is sparse. Parse errors on `page`/`limit` silently default. Models lack binding/validation tags. No request body size limits beyond Go defaults. |
| API Completeness | B+ | Comprehensive endpoint coverage. Swagger annotations on all public methods. Pagination on major lists. Rate limiting middleware exists. CORS configured. |

**Overall: B-** — Solid architecture and conventions. Main gaps are error handling discipline, input validation depth, and a few architectural boundary violations.

---

## Issues Found

### Critical (P0)

**1. SQL Injection Risk in `repository/coin_repository.go:396`**
The `Suggestions()` method interpolates a `column` variable directly into a SQL `WHERE` clause. While the handler validates via a whitelist switch (`handlers/coins.go:444-458`), the repository method itself is unprotected. Any future caller bypassing the handler could inject arbitrary SQL. Defense-in-depth requires the repo to also validate.

**2. Double-close panic in schedulers**
- `services/valuation_scheduler.go:60` — `Stop()` calls `close(s.stopCh)` with no guard.
- `services/availability_scheduler.go:61` — Same pattern.
If `Stop()` is called twice (e.g., during shutdown race), the process panics.

### High (P1)

**3. `settings_service.go` bypasses repository layer**
- Lines 80-131: Global `settingsDB *gorm.DB` with direct `Where/First/Create/Save/Find` calls.
- No constructor injection (`NewSettingsService` absent).
- No error handling on `Find()` (line 124).
- Violates the architectural rule: "Only `main.go` imports the `database` package" spirit — this service effectively is a repository.

**4. Middleware performs direct DB access (`middleware/auth.go:79-101`)**
`authenticateApiKey()` runs GORM queries and updates directly on `*gorm.DB`, bypassing the repository layer. The `last_used_at` update error is silently ignored (line 96).

**5. Non-atomic multi-step writes without transactions**
- `services/auction_lot_service.go:93-99` — `Create` coin then `UpdateFields` lot; if second fails, coin is orphaned.
- `repository/auction_lot_repository.go:141-166` — `Upsert()` does read-then-write without tx.
- `repository/social_repository.go:90-102` — `BlockUser()` read-then-write without tx.
- `services/availability_service.go:133-226` — Run + result updates without tx; state can become inconsistent.

**6. Business logic in handlers**
- `handlers/analysis.go:69-175` — File reading, base64 encoding, prompt selection, side filtering. Should be in a service.
- `handlers/agent.go:110-136, 178-201` — Prompt construction, date/window logic, ZIP lookup, portfolio summary assembly.
- `handlers/coins.go:250-328` — Purchase/sell state transitions and date parsing logic.
- `handlers/admin.go:239-289` — Log merge/filter logic.

### Medium (P2)

**7. Swallowed errors across the codebase**
- `handlers/admin.go:51-64, 94-101, 142-148` — `ListUsers`, `DeleteUser`, `ResetPassword` swallow repo errors.
- `services/availability_service.go:145-147` — `CheckURL` error dropped with `_`.
- `services/valuation_service.go:166, 251, 276, 331` — Multiple repo write errors ignored.
- `services/social_service.go:95-99, 137-141` — Repo errors swallowed in `BuildUserList`, `GetPublicProfileData`.
- `repository/journal_repository.go` — `CoinExists()` ignores error.
- `repository/auction_lot_repository.go:168-174` — `MarkPastAuctionsAsPassed()` ignores result.
- `repository/valuation_repository.go:37-39, 68-88, 145-163` — Prune/cancel/recover errors ignored.

**8. Hardcoded values that should be configurable**
- `handlers/analysis.go:103` — `"uploads"` path hardcoded.
- `handlers/admin.go:299` / `handlers/agent.go:241` — `https://api.anthropic.com/v1/models` hardcoded.
- `handlers/coins.go:53` — Default limit `50` hardcoded.

**9. Input validation gaps in `handlers/coins.go`**
- Lines 52-53: `page`/`limit` parse errors silently default to 0 (not 1/50).
- Line 77-82: Invalid `tag` query param silently ignored.
- No upper bound on `limit` — a client can request `limit=999999`.
- `SortField` and `SortOrder` are user-supplied strings passed to repository without validation against a whitelist.

**10. `image_service.go` orphan file risk**
- Lines 43-70: File is written to disk before the DB record is created. If the DB insert fails, the file is never cleaned up.
- Lines 109-113: `os.Remove` and `repo.DeleteImage()` errors ignored during cleanup.

**11. `services/numisbids_service.go` and `availability_service.go` use direct HTTP calls**
These services make raw `net/http` requests. While they're legitimately HTTP clients (not DB access), they lack consistent timeouts, context propagation, and error wrapping.

**12. Rate limiter cleanup goroutine never stops (`middleware/ratelimit.go:23-42`)**
The background goroutine for expired limiter cleanup has no shutdown mechanism. It runs for the lifetime of the process with no way to stop it gracefully.

### Low (P3)

**13. `repository/tag_repository.go:161-170` — `BulkAttachToCoin()` inserts one-by-one in a loop**
Should batch-insert for efficiency.

**14. `repository/admin_repository.go` — `ListUsers()` is unpaginated**
Could become expensive with many users.

**15. `repository/price_alert_repository.go:52-70, 112-125` — Loop-based saves**
`CheckAndTrigger()` and `GetDueReminders()` save alerts individually inside a loop instead of batch updating.

**16. `config/config.go:58-64` — `AllowedOrigins()` fallback logic**
Falls back to splitting `WebAuthnOrigin` by comma even though it's a single origin field. Fallback list duplicates `http://localhost:8080`.

**17. Models lack validation depth**
- `models/user.go:12-25` — No validation on `Email`, `Role`, `Bio`.
- `models/auction_lot.go` — No validation on many required fields.
- `models/valuation_run.go` — `TriggerType`/`Status` lack enum validation.
- `models/price_alert.go` — `Direction` should be enum-validated; `TargetPrice` needs `>= 0`.
- `models/coin_value_history.go` — Could benefit from composite index on `UserID + RecordedAt`.

**18. `services/logger.go` — Global `AppLogger` singleton**
Hard-couples service to global state, making testing difficult.

**19. `services/agent_proxy.go:145-147, 287-350` — `StreamChat`/`proxySSE` coupled to `http.ResponseWriter`**
These are web-layer concerns living in the service layer. Acceptable for streaming proxy, but ideally the service would return a reader and the handler would pipe it.

**20. `database/database.go:16-29` — PRAGMA errors ignored**
`Exec("PRAGMA ...")` return values are not checked. Silent failure could cause subtle issues (e.g., WAL mode not enabled).

---

## Backlog Items (Backend)

| # | Title | Priority | Effort | Description |
|---|-------|----------|--------|-------------|
| 1 | Add column whitelist to `CoinRepository.Suggestions()` | P0 | S | Defense-in-depth: validate `column` param against an allowlist inside the repository, not just in the handler. `repository/coin_repository.go:393-406`. |
| 2 | Guard scheduler `Stop()` against double-close panic | P0 | S | Use `sync.Once` in `ValuationScheduler.Stop()` and `AvailabilityScheduler.Stop()` to prevent panicking on double `close()`. |
| 3 | Extract `settings_service.go` into a proper repository | P1 | M | Create `SettingsRepository` with constructor injection. Eliminate global `settingsDB`. Wire through `main.go` like other repos. |
| 4 | Move middleware DB access to repository layer | P1 | M | Create `ApiKeyRepository` methods for lookup + last-used update. Inject into `AuthRequired` middleware instead of raw `*gorm.DB`. |
| 5 | Add transactions to non-atomic multi-step writes | P1 | M | Wrap `AuctionLotService.ConvertToCoin`, `AuctionLotRepository.Upsert`, `SocialRepository.BlockUser`, and availability run updates in DB transactions. |
| 6 | Extract business logic from handlers into services | P1 | L | Move analysis prompt/file logic, agent prompt construction, coin state transitions, and admin log filtering from handlers into dedicated service methods. |
| 7 | Audit and fix swallowed errors | P2 | M | Systematic pass through services and repositories to handle or propagate errors instead of silently dropping them. Priority: availability_service, valuation_service, social_service, admin handler. |
| 8 | Replace hardcoded values with config/settings | P2 | S | Make `uploads` path, Anthropic API URL, and default page limits configurable via `config.go` or `AppSetting`. |
| 9 | Add input validation to `coins.go` list endpoint | P2 | S | Validate `page >= 1`, `1 <= limit <= 100`, whitelist `SortField`/`SortOrder` values. Return 400 on invalid input instead of silently defaulting. |
| 10 | Fix orphan file risk in `image_service.go` | P2 | S | Write DB record first (or use a transaction that covers both file write and DB insert), and add cleanup on failure. |
| 11 | Add timeouts and context to HTTP client calls | P2 | M | `numisbids_service.go` and `availability_service.go` HTTP calls need explicit timeouts and context propagation. |
| 12 | Add shutdown mechanism to rate limiter goroutine | P2 | S | Accept a `context.Context` or stop channel in the rate limiter middleware to allow graceful shutdown. |
| 13 | Batch-insert in `tag_repository.BulkAttachToCoin()` | P3 | S | Replace loop of individual inserts with a single batch insert for better performance. |
| 14 | Add pagination to `admin_repository.ListUsers()` | P3 | S | Add `page`/`limit` parameters to prevent unbounded query growth. |
| 15 | Batch-update in `price_alert_repository` | P3 | S | Replace loop-based saves in `CheckAndTrigger()` and `GetDueReminders()` with batch updates. |
| 16 | Fix `AllowedOrigins()` fallback logic | P3 | S | Clean up the comma-split fallback and remove duplicate `localhost:8080` entry in `config.go`. |
| 17 | Add model validation tags | P3 | M | Add `binding:"required"`, enum validation, and range constraints to models: `User`, `AuctionLot`, `ValuationRun`, `PriceAlert`, `Notification`. |
| 18 | Refactor `AppLogger` away from global singleton | P3 | M | Use constructor injection for the logger instead of global `AppLogger` var and `SyncLogLevel()`. |
| 19 | Decouple `agent_proxy` SSE from `http.ResponseWriter` | P3 | L | Have `StreamChat` return an `io.Reader` / channel, let the handler manage the HTTP response writing. |
| 20 | Check PRAGMA execution errors in `database.go` | P3 | S | Handle return values from `Exec("PRAGMA ...")` and log/fail on error. |
