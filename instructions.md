# Aurearia -- Agent Instructions

> **Note:** This file is superseded by `.github/copilot-instructions.md` (auto-read by Copilot),
> `docs/ARCHITECTURE.md`, and `CONTRIBUTING.md`. This file is retained for reference but
> those documents are the authoritative source.

## What This App Does

A full-stack PWA for managing a personal ancient coin collection. Users can catalog coins with photos, track values, maintain wishlists, get AI-powered coin analysis, search for coins via a multi-agent conversational system, find upcoming coin shows, review portfolio valuations, and view collection statistics. Multi-user with JWT + WebAuthn authentication.

## Tech Stack

| Layer    | Technology                                       | Path           |
|----------|--------------------------------------------------|----------------|
| Backend  | Go 1.26, Gin, GORM, Pure-Go SQLite (WAL mode)   | `src/api/`     |
| Frontend | Vue 3, TypeScript, Pinia, Vite, VitePWA          | `src/web/`     |
| Agent    | Python 3.12, FastAPI, LangGraph, LangChain       | `src/agent/`   |
| Build    | Multi-stage Docker (2 containers)                | `Dockerfile`, `src/agent/Dockerfile` |
| CI/CD    | GitHub Actions → Docker Hub                      | `.github/`     |

## Project Layout

```
src/
├── api/                              # Go backend
│   ├── main.go                       # Entry point, route registration, DI wiring
│   ├── config/config.go              # Env var loading (JWT_SECRET, DB_PATH, PORT, etc.)
│   ├── database/database.go          # SQLite connection, GORM AutoMigrate
│   ├── middleware/auth.go            # JWT + API key auth middleware
│   ├── handlers/                     # HTTP handlers (one file per domain)
│   │   ├── auth.go                   # Register, login, token refresh
│   │   ├── coins.go                  # Coin CRUD, list/filter/sort, stats, value history
│   │   ├── images.go                 # Image upload/delete (multipart + base64)
│   │   ├── analysis.go              # AI coin analysis (proxied to Python agent)
│   │   ├── agent.go                  # Agent chat, valuation (proxied to Python agent)
│   │   ├── journal.go               # Per-coin activity log CRUD
│   │   ├── numista.go               # Numista catalog search proxy
│   │   ├── snapshots.go             # Portfolio value snapshot recording
│   │   ├── admin.go                 # User management, settings, logs, connectivity tests
│   │   ├── user.go                  # Password change, profile
│   │   ├── export.go                # Collection export/import
│   │   ├── api_keys.go              # API key generation/revocation
│   │   └── webauthn.go              # FIDO2/WebAuthn passwordless auth
│   ├── models/                       # GORM models
│   ├── repository/                   # Database access layer (one repo per domain)
│   └── services/
│       ├── settings_service.go       # App settings with defaults
│       ├── agent_proxy.go            # SSE proxy to Python agent service
│       ├── availability_service.go   # Wishlist URL checking + agent escalation
│       ├── availability_scheduler.go # Background scheduler for periodic checks
│       ├── ollama_service.go         # Ollama status check
│       └── logger.go                 # Structured logger with in-memory ring buffer
│
├── agent/                            # Python multi-agent service
│   ├── app/
│   │   ├── main.py                   # FastAPI entry point, /health, /logs
│   │   ├── config.py                 # Service settings (env: AGENT_*)
│   │   ├── logging_config.py         # Ring buffer logger matching Go's pattern
│   │   ├── routes.py                 # 5 endpoints: coins, shows, analyze, portfolio, availability
│   │   ├── supervisor.py             # Top-level intent router + team delegation
│   │   ├── streaming.py              # SSE streaming from LangGraph events
│   │   ├── models/                   # Pydantic request/response schemas
│   │   ├── llm/provider.py           # Anthropic vs Ollama LLM factory
│   │   ├── tools/search.py           # SearXNG search + URL verify tools
│   │   ├── tools/numisbids.py        # NumisBids auction search tool; CNG support lives in provider-aware auction services
│   │   └── teams/                    # Multi-agent team pipelines
│   │       ├── coin_search.py        # Team 1: Search → Verify → Format
│   │       ├── coin_shows.py         # Team 2: Search → Date verify → Format
│   │       ├── coin_analysis.py      # Team 3: Vision analysis → Format
│   │       ├── portfolio_review.py   # Team 4: Read → Valuate → Analyze
│   │       ├── auction_search.py     # Team 5: Auction search → Fetch → Format
│   │       └── availability_check.py # Team 6: Check URLs → Analyze results
│   └── tests/                        # pytest tests
│
└── web/                              # Vue 3 SPA
    └── src/
        ├── api/client.ts             # Axios HTTP client, all API methods
        ├── types/index.ts            # TypeScript interfaces
        ├── router/index.ts           # Vue Router, auth guard
        ├── stores/                   # Pinia stores (auth, coins)
        ├── pages/                    # Route-level components
        └── components/               # Reusable components
            └── CoinSearchChat.vue    # AI agent chat drawer with provider banner
```

## Architecture Patterns

### Multi-Agent Architecture

```
Vue SPA → Go API (8080) → Python Agent Service (8081)
```

- **Go API** is a thin proxy — it contains zero LLM/agent logic
- **Python agent** is stateless — no database access, all config passed per-request
- **Go owns the database** — reads settings, coins, user data, passes to Python in request payloads
- SSE streaming flows: Python → Go → Vue (Go proxies the byte stream)

### AI Provider Selection

Users must explicitly choose a provider in Admin Settings:

- **Anthropic (Recommended)** — Claude models with built-in `web_search` tool
- **Ollama** — Self-hosted models, requires external SearXNG for web search

The `AIProvider` setting controls this. When empty (default for new/upgraded installations), the agent chat shows a banner directing the user to Admin Settings. There is no implicit fallback between providers.

### Backend Conventions

- **Handler pattern**: Each domain gets its own file in `handlers/`. Handlers are structs with methods, instantiated via `NewXxxHandler()`, registered in `main.go`.
- **Route registration**: All routes are wired in `main.go` under three groups:
  - `api` (public) — auth endpoints
  - `protected` (JWT required) — all user-facing endpoints
  - `admin` (JWT + admin role) — admin-only endpoints
- **Settings**: Key-value `AppSetting` model. Constants in `services/settings_service.go`. `GetSetting(key)` returns DB value or hardcoded default.
- **Database**: GORM with SQLite. Schema changes via `AutoMigrate` in `database.go`.
- **Auth**: JWT (Bearer token) + API key (`X-API-Key` header).
- **Logging**: `services.AppLogger` — structured logger with ring buffer. Admin logs merge Go + Python entries by timestamp.

### Frontend Conventions

- **API client**: All backend calls go through `src/web/src/api/client.ts`.
- **State**: Pinia stores in `stores/`.
- **Styling**: CSS variables (`--bg-card`, `--accent-gold`, `--border-subtle`). Dark theme is default.
- **Icons**: `lucide-vue-next`.
- **Components**: `<script setup lang="ts">` with Composition API.

### Adding a New Feature (Checklist)

1. **Model** — Add GORM model in `src/api/models/`, add to `AutoMigrate`
2. **Repository** — Add repo in `src/api/repository/`
3. **Service** (if needed) — Add in `src/api/services/`
4. **Handler** — Create handler in `src/api/handlers/`, follow `NewXxxHandler()` pattern
5. **Routes** — Register in `main.go` under the appropriate group
6. **Settings** (if needed) — Add constant + default in `services/settings_service.go`
7. **Types** — Add TypeScript interface in `src/web/src/types/index.ts`
8. **API client** — Add method in `src/web/src/api/client.ts`
9. **UI** — Create page/component, register route
10. **Tests** — Run `go test ./...` to verify architecture rules pass

## Deployment

Two Docker containers orchestrated via `docker-compose.yaml`:

| Container | Image | Port |
|---|---|---|
| `app` | `<user>/ancient-coins:latest` | 8080 |
| `agent` | `<user>/ancient-coins-agent:latest` | 8081 |

External services (not deployed by us): Ollama, SearXNG.

CI builds and pushes both images. `docker-compose` only references images — no local builds.

## Environment Variables

| Variable          | Default                                               | Description                          |
|-------------------|-------------------------------------------------------|--------------------------------------|
| `JWT_SECRET`      | `dev-secret-key-change-in-production-min32chars`      | JWT signing key (min 32 chars)       |
| `DB_PATH`         | `./ancientcoins.db`                                   | SQLite database file path            |
| `PORT`            | `8080`                                                | HTTP server port                     |
| `UPLOAD_DIR`      | `./uploads`                                           | Directory for uploaded coin images   |
| `WEBAUTHN_RP_ID`  | `localhost`                                           | WebAuthn Relying Party ID            |
| `WEBAUTHN_ORIGIN` | `http://localhost:8080`                               | WebAuthn origin URL                  |
| `AGENT_SERVICE_URL` | `http://agent:8081`                                 | Python agent service URL             |
| `AGENT_INTERNAL_SERVICE_TOKEN` | —                                      | Shared API → agent credential        |
| `AGENT_LOG_LEVEL` | `INFO`                                                | Python agent log level               |

## Admin-Managed Settings (stored in DB)

| Key                    | Purpose                                                |
|------------------------|--------------------------------------------------------|
| `AIProvider`           | Explicit provider: `anthropic` or `ollama` (empty = unconfigured) |
| `AnthropicAPIKey`      | API key for Claude models                              |
| `AnthropicModel`       | Claude model name                                      |
| `OllamaURL`            | Ollama server URL for AI analysis                      |
| `OllamaModel`          | Vision model name (e.g., `llava`)                      |
| `OllamaTimeout`        | Request timeout in seconds                             |
| `SearXNGURL`           | SearXNG search URL (required for Ollama web search)    |
| `NumistaAPIKey`        | Numista catalog API key                                |
| `AgentPrompt`          | System prompt for coin search agent                    |
| `ValuationPrompt`      | System prompt for value estimator                      |
| `ObversePrompt`        | Custom prompt for obverse analysis                     |
| `ReversePrompt`        | Custom prompt for reverse analysis                     |
| `TextExtractionPrompt` | Custom prompt for OCR text extraction                  |
| `LogLevel`             | Application log level                                  |
| `WishlistCheckEnabled` | Enable automatic wishlist availability checks (`true`/`false`) |
| `WishlistCheckStartTime` | Daily start time for scheduled checks (HH:MM, default `02:00`) |
| `WishlistCheckInterval` | Repeat interval in minutes (default `120`)             |

## Running Locally

```sh
task run          # Starts both API (:8080) and Vite dev server (:5173)
task run-api      # API only
task run-web      # Frontend only (proxies /api/* to :8080)
```

## Building

```sh
task build        # Build both API binary and Vue dist/
task docker-build # Build Docker image
```

## Testing

```sh
# Go
cd src/api && go build ./... && go vet ./... && go test -v ./...

# Vue
cd src/web && npx vue-tsc --noEmit

# Python
cd src/agent && ruff check app/ tests/ && pytest tests/ -q
```

## Commit Convention

Use conventional commit prefixes: `feat:`, `fix:`, `docs:`, `refactor:`, `style:`, `chore:`

Always include the co-author trailer:
```
Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>
```
