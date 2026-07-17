# Ancient Coins — Go API

Go 1.26 + Gin + GORM + SQLite REST API for managing an ancient coin collection. Serves as the backend for the Vue SPA and proxies AI requests to the Python agent service.

## Prerequisites

- **Go** 1.26+

## Build & Run

```bash
go build ./...       # compile
go vet ./...         # lint
go test -v ./...     # run all tests (architecture + unit)
```

The server starts on port `8080` by default. In production, set `GIN_MODE=release`.

## Architecture

```
Handler → Service → Repository → Database
```

This layered architecture is **enforced by `architecture_test.go`**:

1. **Only `main.go` imports the `database` package.** All other packages receive `*gorm.DB` via constructor injection.
2. **Handlers are thin.** Parse request, call service/repo, return response. No business logic.
3. **Services contain business logic.** Orchestrate repositories, enforce domain rules. HTTP-agnostic (no `gin.Context`).
4. **Repositories own all DB access.** Every GORM query lives in `repository/`.
5. **Multi-step writes use transactions** (`r.db.Transaction()`).

### Package Import Rules

| Package | May Import |
|---|---|
| `handlers/` | `services/`, `repository/`, `models/` |
| `services/` | `repository/`, `models/` |
| `repository/` | `models/`, `gorm.io/gorm` |
| `models/` | Standard library only |
| `middleware/` | `models/`, `gorm.io/gorm` |

## Key Packages

```
config/        # Environment variable loading (Config struct)
database/      # SQLite connection and GORM AutoMigrate
models/        # GORM model definitions
repository/    # Data access layer (one file per domain)
services/      # Business logic, schedulers, agent proxy
handlers/      # Gin HTTP handlers (thin, DI-wired)
middleware/    # JWT auth, API key auth, rate limiting
docs/          # Swagger/OpenAPI generated docs
```

## DI Wiring (main.go)

```
config.Load() → database.Connect() → construct repos → construct services → construct handlers → register routes
```

All dependencies are wired via constructor injection in `main.go`. No global state or service locators.

## Route Groups

### `api` — Public (no auth)

- `POST /api/auth/register` — Register new user (rate limited)
- `POST /api/auth/login` — Login (rate limited)
- `POST /api/auth/refresh` — Refresh JWT token (rate limited)
- `GET  /api/auth/setup` — Check if any users exist
- `POST /api/auth/webauthn/login/begin|finish` — WebAuthn login ceremony
- `GET  /api/auth/webauthn/check` — Check WebAuthn credential availability
- `GET  /api/showcase/:slug` — Public showcase view

### `protected` — JWT or API Key Required

- `/api/coins` — Full CRUD, purchase, sell, bulk actions
- `/api/tags` — Tag CRUD, attach/detach from coins
- `/api/coins/:id/journal` — Journal entries per coin
- `/api/coins/:id/images` — Image upload (multipart + base64), delete
- `/api/coins/:id/analyze` — AI analysis via agent proxy
- `/api/stats`, `/api/suggestions` — Collection statistics
- `/api/auctions` — Auction lot CRUD, import, sync, convert to coin
- `/api/agent/chat` — SSE streaming AI chat (proxied to Python agent)
- `/api/agent/conversations` — Conversation history CRUD
- `/api/auth/api-keys` — API key management
- `/api/auth/webauthn/register/*` — WebAuthn credential registration
- `/api/user/profile`, `/api/user/avatar`, `/api/user/export` — User self-service
- `/api/social/*` — Follow/unfollow, comments, ratings
- `/api/notifications` — Notification list, read, delete
- `/api/showcases` — Showcase CRUD, coin assignment
- `/api/calendar`, `/api/calendar/events` — Auction calendar
- `/api/alerts`, `/api/reminders` — Price alerts and bid reminders
- `/api/wishlist/check-availability` — Wishlist availability checking

## Background Schedulers

The API runs three background schedulers that start automatically on server startup:

1. **Wishlist Availability Scheduler** — Checks if wishlist coins with reference URLs are still available for purchase. Configured via `WishlistCheckEnabled`, `WishlistCheckStartTime` (HH:MM), and `WishlistCheckInterval` (minutes). Default: every 2 hours starting at 02:00. Each run is logged in the `availability_runs` table.

2. **Collection Valuation Scheduler** — Periodically re-values all owned coins using the AI agent. Configured via `ValuationCheckEnabled`, `ValuationCheckStartTime` (HH:MM), and `ValuationCheckIntervalDays` (days). Default: every 7 days at 03:00. Each run is logged in the `valuation_runs` table. Runs can be manually triggered via `/admin/valuation-runs/trigger` and cancelled via `/admin/valuation-runs/{id}/cancel`.

3. **Auction Ending Scheduler** — Checks for auction lots ending today and sends consolidated Pushover notifications per user. Configured via `AuctionEndingCheckEnabled`, `AuctionEndingCheckStartTime` (HH:MM), and `AuctionEndingCheckInterval` (minutes). Default: every 24 hours at 08:00. Each run is logged in the `auction_ending_runs` table. Runs can be manually triggered via `/admin/auction-ending/run`.

4. **Wishlist Search Alert Scheduler** — Daily sweep that queues an automatic discovery run for each active `WishlistSearchAlert` whose configured cadence (`daily`/`weekly`/`monthly`) has elapsed since its last run. Configured via `WishlistSearchAlertsCheckEnabled` (default `"false"`) and `WishlistSearchAlertsCheckStartTime` (HH:MM, default `03:00`). Alerts with `cadence: manual` are never auto-run — only "Run Now" processes them. Queued runs are logged in `alert_runs` the same as manual runs, with `triggerType: scheduled`.

All schedulers honor the enabled flag — set to `"false"` to disable. Run history is available via admin endpoints:
- `GET /admin/availability-runs` — Paginated list of availability check runs
- `GET /admin/valuation-runs` — Paginated list of valuation runs
- `GET /admin/auction-ending-runs` — Paginated list of auction ending runs

## Pagination

1. **Wishlist Availability Scheduler** — Periodically checks if wishlist coins are still available at their reference URLs. Configured via `WishlistCheckEnabled`, `WishlistCheckStartTime`, and `WishlistCheckInterval` settings. Sends Pushover notifications when items become unavailable.

2. **Collection Valuation Scheduler** — Periodically runs AI-powered valuation estimates for the user's collection. Configured via `ValuationCheckEnabled`, `ValuationCheckStartTime`, and `ValuationCheckIntervalDays` settings. Sends Pushover notifications with valuation summaries.

3. **Auction Ending Scheduler** — Daily check for auction lots the user is bidding on that end today. Configured via `AuctionEndingCheckEnabled`, `AuctionEndingCheckStartTime`, and `AuctionEndingCheckInterval` settings. Sends consolidated Pushover notifications per user with all ending auctions. Uses in-memory idempotency tracking to avoid duplicate notifications within the same day.

All schedulers respect user-level Pushover notification settings and gracefully handle disabled or missing configuration.

### `admin` — JWT + Admin Role Required

- `/api/admin/users` — List/delete users, reset passwords
- `/api/admin/settings` — App settings CRUD with defaults
- `/api/admin/logs` — Application log viewer
- `/api/admin/test-anthropic`, `/api/admin/test-searxng` — Connection tests
- `/api/admin/availability-runs` — Availability check history
- `/api/admin/valuation-runs` — Valuation run history and manual trigger

## Environment Variables

| Variable | Default | Description |
|---|---|---|
| `JWT_SECRET` | *(dev fallback)* | JWT signing secret (min 32 chars, required in production) |
| `DB_PATH` | `./ancientcoins.db` | SQLite database file path |
| `PORT` | `8080` | Server listen port |
| `UPLOAD_DIR` | `./uploads` | Directory for uploaded coin images |
| `WEBAUTHN_RP_ID` | `localhost` | WebAuthn Relying Party ID (domain) |
| `WEBAUTHN_ORIGIN` | `http://localhost:8080` | WebAuthn allowed origin |
| `CORS_ORIGINS` | *(empty, falls back to WebAuthn origins)* | Comma-separated allowed CORS origins |
| `AGENT_SERVICE_URL` | `http://localhost:8081` | Python agent service base URL |
| `AGENT_INTERNAL_SERVICE_TOKEN` | *(empty in local dev; required for hardened agent)* | Shared API → agent credential. Must match the Python agent's value when agent auth is enabled. |
| `GIN_MODE` | `debug` | Set to `release` for production |

## Authentication Model

- **JWT** — Primary auth. Access + refresh token pair. Access tokens are short-lived; refresh tokens enable silent renewal.
- **WebAuthn / FIDO2** — Passwordless login via hardware keys or platform authenticators. Registration requires an existing JWT session; login is public.
- **API Keys** — `X-API-Key` header. Prefixed with `ak_`, scoped to a user. Managed via `/api/auth/api-keys`.

The `middleware.AuthRequired` middleware accepts either a valid JWT `Authorization: Bearer ...` header or a valid `X-API-Key` header.

## Swagger Docs

Available at `/swagger/index.html` when the server is running. Generated from handler annotations using `swag`.
