# Copilot Instructions

> Repository-level instructions for GitHub Copilot (IDE, CLI, and code review).
> This file is automatically read by Copilot on every interaction.

## Project Overview

Ancient Coins is a full-stack PWA for managing a personal ancient coin collection. Go/Gin API backend with Vue 3/TypeScript frontend, SQLite database, and a Python LangGraph multi-agent service for AI features.

| Layer | Tech | Path |
|---|---|---|
| Backend | Go 1.26.1, Gin, GORM, SQLite | `src/api/` |
| Frontend | Vue 3, TypeScript, Pinia, Vite, PWA | `src/web/` |
| Agent | Python 3.12, FastAPI, LangGraph, LangChain | `src/agent/` |
| Build | Multi-stage Docker (2 containers) | `Dockerfile`, `src/agent/Dockerfile` |

## Build, Test, and Lint

A [Taskfile](../Taskfile.yml) wraps common commands. Run `task --list` to see all targets.

```bash
# Go API (from src/api/)
go build ./...                          # compile
go vet ./...                            # lint
go test -v ./...                        # all tests (architecture + unit)
go test -v -run TestNoDirectDatabase .  # single test by name

# Vue frontend (from src/web/)
npm run build                           # production build (type-check + vite)
npm run type-check                      # vue-tsc only
npx vue-tsc --noEmit                    # alternative type check

# Python agent (from src/agent/)
pip install -e ".[dev]"                 # install with dev deps
ruff check app/ tests/                  # lint
pytest tests/ -v                        # all tests
pytest tests/test_foo.py::test_bar -v   # single test

# Task runner shortcuts (from repo root)
task build                              # build API + web
task test                               # Go tests
task up                                 # API + web dev servers
task up-all                             # API + web + agent dev servers
task test-agent                         # Python tests
task lint-agent                         # Python lint
```

## Architecture

See `docs/ARCHITECTURE.md` for full details. For binding project principles, see the **Project Constitution** at `.specify/memory/constitution.md` — all code changes must comply with its 16 principles.

### Go API — Layered Architecture

```
Handler → Service → Repository → Database
```

**Rules (enforced by `architecture_test.go`):**

1. **Only `main.go` imports the `database` package.** All other packages receive `*gorm.DB` or a repository/service via constructor injection.
2. **Handlers are thin.** Parse request, call service/repo, return response. No business logic, no raw SQL.
3. **Services contain business logic.** Orchestrate repos, enforce domain rules. HTTP-agnostic (no `gin.Context`).
4. **Repositories own all DB access.** Every GORM query lives in `src/api/repository/`.
5. **Multi-step writes use transactions** (`r.db.Transaction()`).
6. **Never leak internal errors to clients.** Log server-side, return generic messages.
7. **Go API contains zero LLM/agent logic.** All AI inference is proxied to the Python agent service.

**Package import rules:**

| Package | May import |
|---|---|
| `handlers/` | `services/`, `repository/`, `models/` |
| `services/` | `repository/`, `models/` |
| `repository/` | `models/`, `gorm.io/gorm` |
| `models/` | Standard library only |
| `middleware/` | `models/`, `gorm.io/gorm` |

**DI wiring in `main.go`:** `config.Load()` → `database.Connect()` → construct repos → construct services → construct handlers → register routes. Three route groups: `api` (public auth), `protected` (JWT required), `admin` (JWT + admin role).

### Multi-Agent Architecture (Python)

```
Vue SPA → Go API (8080) → Python Agent Service (8081)
```

The Python agent is a **stateless** FastAPI service — no database access. All configuration (API keys, models, prompts, user context) is passed per-request from the Go API. SSE streams flow Python → Go → Vue (Go proxies the byte stream via `services/agent_proxy.go`).

**Team pipelines:**

| Team | Pipeline |
|---|---|
| Coin Search | Search → Fetch dealer pages → Format |
| Coin Shows | Search → Verify dates are future → Format |
| Coin Analysis | Vision model analysis → Format |
| Portfolio Review | Read holdings → Valuate → Analyze |
| Availability Check | Check URLs → Analyze results → Verdict |

**Key design rules:**
- Search agents pass only tool-returned data downstream — never invented details
- Verification agents confirm every URL is live and every date is in the future
- All worker agent outputs conform to a defined Pydantic schema — no free-form text
- Top-level supervisor (`app/supervisor.py`) enforces max iteration count to prevent loops

### AI Provider Configuration

Users choose one provider in Admin Settings (`AIProvider` key):

- **Anthropic** — Claude models. Web search uses Claude's built-in `web_search_20250305` tool.
- **Ollama** — Self-hosted models. Web search uses a `create_react_agent` with SearXNG tool.

**Important:** Anthropic's `web_search` is NOT available by default on `ChatAnthropic`. Use `get_search_model()` from `app/llm/provider.py` (which calls `bind_tools`) for any agent node that needs web search. Use `get_chat_model()` for nodes that don't search.

## Code Conventions

### Go
- Constructor injection for all dependencies (`NewXxxHandler(repo, service)` pattern)
- Sentinel errors in services (e.g., `ErrNotFound`, `ErrInvalidCredentials`)
- Use GORM scopes from `repository/scopes.go` (`OwnedBy`, `OwnedByID`, `ActiveCollection`, `PublicCoins`, `ByCoinID`) instead of repeating `.Where()` clauses
- Swagger annotations on all public handler methods
- Settings use key-value `AppSetting` model; constants and defaults live in `services/settings_service.go`

### Python (Agent)
- Pydantic models for all request/response schemas (in `app/models/`)
- LangGraph `StateGraph` for team pipelines
- `create_react_agent()` for tool-using agents
- Structured logging via `app/logging_config.py` (ring buffer + stdout)

### TypeScript / Vue
- `<script setup lang="ts">` with Composition API
- **Docker builds use stricter TS checking than local `vue-tsc`.** Always use optional chaining (`?.`) and nullish coalescing (`??`) on array index access. When passing nullable props (`string | null | undefined`) to a child component that expects non-nullable types (`string`), use `?? ''` (strings) or `?? 0` (numbers) at the call site. Local `vue-tsc --noEmit` may pass but Docker's `vue-tsc --build` will reject the mismatch.
- All API calls go through `src/web/src/api/client.ts` (Axios with JWT interceptor and 401 refresh queue)
- Agent chat streaming uses `fetch` + manual SSE parsing, not Axios
- `sanitizeCoin()` in the API client normalizes `''`/`undefined` → `null` before sending
- CSS variables: `--accent-gold`, `--bg-card`, `--border-subtle`, `--text-primary`
- Icons: `lucide-vue-next`

### UI / UX
- No emojis in UI text, prompts, or AI responses
- Dark theme is default
- PWA-compatible — test on mobile viewports

### Design System

All CSS values **must** use design tokens from `variables.css` and global classes from `main.css`. Never hardcode raw values when a token exists.

#### Design Tokens (variables.css)

| Token | Value | Use for |
|---|---|---|
| `--radius-sm` | `8px` | Cards, inputs, buttons |
| `--radius-md` | `12px` | Larger containers, modals |
| `--radius-lg` | `16px` | Hero sections |
| `--radius-full` | `9999px` | Pills, chips, badges |
| `--border-subtle` | gold 15% | Default borders |
| `--border-accent` | gold 40% | Hover/active borders |
| `--accent-gold` | `#c9a84c` | Primary accent, active states, links |
| `--accent-bronze` | `#b08d57` | Secondary accent |
| `--accent-gold-dim` | gold 30% | Active chip/pill backgrounds |
| `--accent-gold-glow` | gold 15% | Focus rings, subtle backgrounds |
| `--bg-card` | `#16213e` | Card backgrounds |
| `--bg-card-hover` | `#1a2747` | Card hover state |
| `--bg-input` | `#1e2a4a` | Input/textarea backgrounds |
| `--text-primary` | `#e8e0d0` | Body text |
| `--text-secondary` | `#a09880` | Secondary text, descriptions |
| `--text-muted` | `#706858` | Labels, hints, placeholders |
| `--text-heading` | `#d4b96a` | Headings (h1–h4) |
| `--cat-roman` | `#9b59b6` | Roman category |
| `--cat-greek` | `#6b8e23` | Greek category |
| `--cat-byzantine` | `#c0392b` | Byzantine category |
| `--cat-modern` | `#4682b4` | Modern category |
| `--mat-gold/silver/bronze` | metal colors | Material indicators |
| `--shadow-card` | box-shadow | Card elevation |
| `--shadow-glow` | gold glow | Hover/focus glow effect |
| `--transition-fast` | `0.2s ease` | Hover, focus |
| `--transition-med` | `0.3s ease` | Layout changes |

#### Typography Scale

| Element | Font | Size | Weight |
|---|---|---|---|
| h1 | Cinzel | `2rem` | 600 |
| h2 | Cinzel | `1.5rem` | 500 |
| h3 | Cinzel | `1.2rem` | 500 |
| h4 | Cinzel | `0.9rem` | 500 |
| Body | Inter | `0.9rem` | 400 |
| Secondary | Inter | `0.85rem` | 400 |
| Small | Inter | `0.8rem` | 400 |
| Tiny | Inter | `0.75rem` | 500 |

#### Uppercase Labels

All uppercase labels (section headers, info-card labels, sub-headings) use:
```css
font-size: 0.7rem;
font-weight: 600;
text-transform: uppercase;
letter-spacing: 0.08em;
color: var(--text-muted);
```
Use the global `.section-label` class or `.info-label` in detail grids.

#### Chip / Pill Hierarchy (global classes in main.css)

| Class | Use | Size | Padding |
|---|---|---|---|
| `.chip` | Interactive filter pills (face, category) | `0.8rem` | `0.35rem 0.85rem` |
| `.chip-sm` | Static tag/label pills | `0.75rem` | `0.15rem 0.5rem` |
| `.badge` | Category badges (Roman, Greek, etc.) | `0.75rem` | `0.2rem 0.7rem` |

All chips use `border-radius: var(--radius-full)`. Active state: `background: var(--accent-gold-dim); border-color: var(--accent-gold); color: var(--accent-gold)`.

#### Button Hierarchy (global classes in main.css)

| Class | Use | Padding | Font size |
|---|---|---|---|
| `.btn` | Standard button | `0.6rem 1.2rem` | `0.9rem` |
| `.btn-sm` | Compact button | `0.4rem 0.8rem` | `0.8rem` |
| `.btn-xs` | Inline/tiny actions | `0.25rem 0.6rem` | `0.75rem` |
| `.btn-primary` | Gold gradient CTA | — | — |
| `.btn-secondary` | Bordered neutral | — | — |
| `.btn-ghost` | Transparent, subtle border | — | — |
| `.btn-danger` | Red destructive | — | — |

#### Spacing Rhythm

- Section gaps: `1.5rem` between major sections (inscriptions, tags, info-grid, descriptions, notes)
- Sub-item gaps: `0.75rem` within sections
- Chip/tag gaps: `0.35rem`
- Card internal padding: `0.75rem` (info cards), `1rem` (feature cards), `1.5rem` (page cards)

#### Rules for New UI Components

1. **Never hardcode** `border-radius`, colors, or spacing — always use tokens
2. **Never duplicate** chip/button CSS — use the global classes
3. **Never invent** a new font-size — pick from the typography scale
4. **All interactive pills** use `.chip` or extend it
5. **All static tags** use `.chip-sm` sizing (`0.75rem`, `0.15rem 0.5rem`)
6. **All uppercase labels** use `letter-spacing: 0.08em` — no other value
7. **Gold (`--accent-gold`)** is reserved for: active states, values/prices, links, section accents
8. **Cards** use `var(--radius-sm)` for small cards, `var(--radius-md)` for containers

### Adding a New API Feature

1. Model in `src/api/models/` → add to `AutoMigrate` in `database/database.go`
2. Repository in `src/api/repository/*_repository.go`
3. Service (if business logic needed) in `src/api/services/*_service.go`
4. Thin handler in `src/api/handlers/` with `NewXxxHandler()` constructor
5. Wire in `src/api/main.go` (create repo → service → handler, register routes under correct group)
6. Run `go test ./...` to verify architecture rules pass

### Notable Endpoints & Features

- **AI Provider Status:** `GET /ai-status` returns `{ provider, available, model, message }`. Frontend uses this provider-agnostic check before AI analysis instead of legacy `/ollama-status`.
- **Random Gallery Sort:** Collection list accepts `?sort=random&seed=N` where `N` is an integer (validated via `strconv.Atoi`). Order is `((id * seed) + seed) % 2147483647` for SQL-safe deterministic shuffle. Frontend persists the seed in `sessionStorage` under `coins:randomSeed` for stable pagination within a session.
- **Coin of the Day:** Daily scheduler picks one coin per enrolled user, sends in-app notification + Pushover. Clicking the notification opens `FeaturedCoinModal` (not a route).
  - Admin settings: `CoinOfDayEnabled`, `CoinOfDayStartTime` (24h `HH:MM`)
  - Per-user opt-in field: `User.CoinOfDayEnabled` (default `true`) — surfaced as toggle in Settings → Account
  - Endpoints:
    - `GET /featured-coins/latest` — most recent for the current user
    - `GET /featured-coins/:id` — fetch one (user-scoped); preloads `Coin.Images`
    - `POST /admin/coin-of-day/run` — admin manual trigger; returns `{ picked, skipped, errors }`
  - Notification type: `coin_of_day`; `referenceId` is the `FeaturedCoin.ID` (NOT a coin id).
  - Selection algorithm (`PickNextCoinID`): cycles through every owned, non-wishlist, non-sold coin via LEFT JOIN on `featured_coins` (`ORDER BY (last_shown IS NULL) DESC, last_shown ASC, c.id ASC`); each coin appears once before any repeats.
  - Dual idempotency: in-memory `map[userID]string` + DB check `HasBeenFeaturedToday` — safe across process restarts on the same day.
  - Summary is cached at pick time (`buildCoinSummary` fallback chain: `AIAnalysis` → `Obverse + Reverse` → structured fields → bare name) so the modal renders cached prose without an extra AI call.

## Commit Convention

Use conventional prefixes: `feat:`, `fix:`, `docs:`, `refactor:`, `chore:`

Always include the co-author trailer:
```
Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>
```
