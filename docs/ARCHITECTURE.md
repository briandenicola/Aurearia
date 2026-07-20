# Architecture

> Full-system architecture for the Aurearia PWA. Covers all three services, their interactions, and the design rules that keep the system maintainable.

---

## Table of Contents

- [System Overview](#system-overview)
- [Container Topology](#container-topology)
- [Go API — Layered Architecture](#go-api--layered-architecture)
  - [Layer Diagram](#layer-diagram)
  - [Architecture Rules](#architecture-rules)
  - [Package Map](#package-map)
  - [Dependency Injection Wiring](#dependency-injection-wiring)
  - [Route Groups and Authorization](#route-groups-and-authorization)
  - [Shared GORM Scopes](#shared-gorm-scopes)
  - [Architecture Tests](#architecture-tests)
- [Vue 3 Frontend](#vue-3-frontend)
  - [Application Structure](#application-structure)
  - [Routing](#routing)
  - [State Management (Pinia)](#state-management-pinia)
  - [API Client](#api-client)
  - [Composables](#composables)
  - [PWA Configuration](#pwa-configuration)
- [Python Agent Service](#python-agent-service)
  - [Service Overview](#service-overview)
  - [Supervisor and Team Routing](#supervisor-and-team-routing)
  - [Team Pipelines](#team-pipelines)
  - [LLM Provider Abstraction](#llm-provider-abstraction)
  - [SSE Streaming](#sse-streaming)
- [Data Flow Diagrams](#data-flow-diagrams)
  - [Standard API Request](#standard-api-request)
  - [Agent Chat (SSE Streaming)](#agent-chat-sse-streaming)
  - [Authentication Flow](#authentication-flow)
  - [Wishlist Availability Check](#wishlist-availability-check)
- [Database Schema](#database-schema)
  - [Core Domain Models](#core-domain-models)
  - [Authentication Models](#authentication-models)
  - [Social and Notification Models](#social-and-notification-models)
  - [Auction and Marketplace Models](#auction-and-marketplace-models)
  - [AI and Agent Models](#ai-and-agent-models)
  - [System Models](#system-models)
- [Authentication and Authorization](#authentication-and-authorization)
- [AI/Agent Integration Pattern](#aiagent-integration-pattern)
- [Background Schedulers](#background-schedulers)
- [Build and Deployment](#build-and-deployment)
- [Configuration Reference](#configuration-reference)
- [Key Design Decisions](#key-design-decisions)

---

## System Overview

Aurearia is a full-stack PWA for managing a personal coin collection, with deep support for ancient and historical coins. The system is composed of three services:

| Service | Tech Stack | Port | Path |
|---------|-----------|------|------|
| **Go API** | Go 1.26.5, Gin, GORM, SQLite | 8080 | `src/api/` |
| **Vue Frontend** | Vue 3, TypeScript, Pinia, Vite, PWA | (bundled) | `src/web/` |
| **Python Agent** | Python 3.12, FastAPI, LangGraph, LangChain | 8081 | `src/agent/` |

The Vue SPA is bundled into the Go binary's `wwwroot/` directory at build time. The Go API serves both the SPA and the REST API. The Python agent service runs as a separate container and is accessed only by the Go API — never directly by the frontend.

## Container Topology

```
┌──────────────────────────────────────────────────────────────────────┐
│                          Docker Host                                 │
│                                                                      │
│  ┌─────────────────────────────┐    ┌──────────────────────────────┐ │
│  │     Go API Container        │    │   Python Agent Container     │ │
│  │                             │    │                              │ │
│  │  ┌───────────────────────┐  │    │  ┌────────────────────────┐ │ │
│  │  │  Vue SPA (wwwroot/)   │  │    │  │  FastAPI + LangGraph   │ │ │
│  │  └───────────────────────┘  │    │  │                        │ │ │
│  │  ┌───────────────────────┐  │    │  │  11 Team Pipelines     │ │ │
│  │  │  Gin HTTP Server      │  │    │  │  Supervisor Router     │ │ │
│  │  │  REST API + SSE Proxy │──────────│  SSE Streaming         │ │ │
│  │  └───────────────────────┘  │    │  └────────────────────────┘ │ │
│  │  ┌───────────────────────┐  │    │           │                  │ │
│  │  │  SQLite Database      │  │    │     ┌─────┴──────┐          │ │
│  │  └───────────────────────┘  │    │     v            v          │ │
│  │         :8080               │    │  Claude API   Ollama+SearXNG│ │
│  └─────────────────────────────┘    └──────────────────────────────┘ │
└──────────────────────────────────────────────────────────────────────┘
         ▲
         │ HTTPS
    ┌────┴────┐
    │ Browser │
    │  (PWA)  │
    └─────────┘
```

---

## Go API — Layered Architecture

### Layer Diagram

```
Request
  │
  ▼
[Handler]     ── Parses HTTP, calls service/repo, returns response
  │
  ▼
[Service]     ── Business logic, orchestration, domain rules (HTTP-agnostic)
  │
  ▼
[Repository]  ── GORM queries, transactions (one repo per domain)
  │
  ▼
[Database]    ── *gorm.DB (initialized once in main.go, injected everywhere)
```

### Architecture Rules

**1. Only `main.go` imports the `database` package.**
Every other package receives `*gorm.DB` or a repository/service through its constructor. Enforced by `architecture_test.go`.

```go
// GOOD (main.go)
coinRepo := repository.NewCoinRepository(database.DB)

// BAD (anywhere else)
database.DB.Find(&coins)  // violates DI rule
```

**2. Handlers are thin.**
Parse request, call service/repo, return HTTP response. No business logic, no raw SQL.

**3. Services contain business logic.**
Orchestrate repos, enforce domain rules, perform data transformations. HTTP-agnostic — no `gin.Context`, no `net/http`.

```go
func (s *CoinService) UpdateCoin(existing *models.Coin, updates *models.Coin, ...) error {
    s.repo.Update(existing, updates)
    if valueChanged { s.repo.RecordValueHistory(...) }
    s.repo.RecordValueSnapshot(userID)
}
```

**4. Repositories own all database access.**
Every GORM query lives in a repository. Repositories take `*gorm.DB` via constructor.

```go
func (r *CoinRepository) FindByID(id, userID uint) (*models.Coin, error) {
    var coin models.Coin
    err := r.db.Scopes(OwnedByID(id, userID)).Preload("Images").First(&coin).Error
    return &coin, err
}
```

**5. Multi-step writes use transactions.**
`r.db.Transaction()` wraps 2+ writes that must atomically succeed or fail:
- `CoinRepository.Delete` (coin + images + journals + value history)
- `AdminRepository.DeleteUserCascade` (all user data)
- `AuthRepository.RotateRefreshToken` (revoke old + create new)
- `ImageRepository.SetPrimaryAndCreate` (clear primary + insert)

**6. Errors stay internal.**
Never expose raw `err.Error()` from GORM, Ollama, or WebAuthn to clients. Log server-side, return generic messages. Validation errors from `ShouldBindJSON` are acceptable.

**7. Go API contains zero LLM logic.**
All AI inference is proxied to the Python agent via `services/agent_proxy.go`.

### Package Map

| Package | Responsibility | Imports Allowed |
|---------|---------------|-----------------|
| `main.go` | Composition root, DI wiring | Everything |
| `handlers/` | HTTP layer (thin) | `services/`, `repository/`, `models/` |
| `services/` | Business logic, orchestration | `repository/`, `models/` |
| `repository/` | Database access, GORM queries | `models/`, `gorm.io/gorm` |
| `models/` | Data structures (GORM models) | Standard library only |
| `middleware/` | Auth, rate limiting | `models/`, `gorm.io/gorm` |
| `config/` | Environment configuration | Standard library only |
| `database/` | DB initialization, AutoMigrate | `gorm.io/gorm`, SQLite driver |

### Dependency Injection Wiring

All wiring happens in `main.go` following this sequence:

```
config.Load()
    → database.Connect(cfg.DBPath)
    → construct repositories (each takes *gorm.DB)
    → construct services (take repos + config)
    → construct handlers (take repos + services)
    → register routes under api/protected/admin groups
    → start background schedulers
    → r.Run()
```

### Route Groups and Authorization

Three route groups with distinct auth levels:

| Group | Prefix | Auth | Example Routes |
|-------|--------|------|----------------|
| `api` (public) | `/api` | None (rate-limited) | `/auth/login`, `/auth/register`, `/auth/refresh`, `/auth/webauthn/*`, `/showcase/:slug` |
| `protected` | `/api` | JWT or API Key | `/coins`, `/coins/bulk`, `/sets`, `/agent/chat`, `/agent/status`, `/auctions`, `/stats`, `/social/*`, `/notifications`, `/calendar/*`, `/alerts`, `/reminders`, `/showcases/*`, `/api-keys` |
| `admin` | `/api/admin` | JWT + admin role | `/users`, `/settings`, `/logs`, `/availability-runs`, `/valuation-runs`, `/valuation-runs/trigger`, `/test-anthropic`, `/test-searxng` |

### Shared GORM Scopes

Reusable query scopes in `repository/scopes.go`:

| Scope | Purpose |
|-------|---------|
| `OwnedBy(userID)` | Filter by `user_id` |
| `ByID(id)` | Filter by primary key |
| `OwnedByID(id, userID)` | Filter by both |
| `ActiveCollection(userID)` | Non-wishlist, non-sold coins |
| `PublicCoins(userID)` | Public, non-wishlist, non-sold |
| `ByCoinID(coinID)` | Filter by `coin_id` |

### Architecture Tests

Run with:
```bash
cd src/api
go test -v -run "TestNoDirectDatabase|TestHandlersDoNotUseRawSQL" .
```

Enforced rules:
- **No direct database imports** — `handlers/`, `services/`, `middleware/`, `repository/` must not import the `database` package
- **No raw SQL in handlers** — flags `SELECT`, `INSERT INTO`, `UPDATE`, `DELETE FROM`, `.Raw(`, `.Exec(` in handler files

---

## Vue 3 Frontend

### Application Structure

```
src/web/src/
├── main.ts                  # App bootstrap (Vue + Pinia + Router)
├── App.vue                  # Top-level layout (nav, sidebar, agent drawer)
├── api/
│   └── client.ts            # Axios instance, JWT interceptor, 401 refresh queue
├── router/
│   └── index.ts             # Route definitions with lazy loading
├── stores/
│   ├── auth.ts              # Auth state, login/register/logout
│   └── coins.ts             # Collection state, CRUD, stats
├── composables/
│   ├── useCoinSearchChat.ts # Agent chat streaming state
│   ├── useCollectionFilters.ts
│   ├── useBulkSelect.ts
│   ├── useDialog.ts
│   ├── useImageProcessor.ts
│   ├── useNotifications.ts
│   ├── usePullToRefresh.ts
│   ├── usePwa.ts
│   ├── useSettingsProfile.ts
│   └── useAdminConfig.ts
├── pages/                   # Full-page views (21 pages)
├── components/              # Reusable UI components
│   ├── chat/                # Agent chat sub-components
│   ├── admin/               # Admin panel components
│   ├── coin/                # Coin detail sub-components
│   ├── settings/            # Settings sub-components
│   └── ...                  # Shared components (forms, cards, modals)
├── types/
│   └── index.ts             # TypeScript interfaces
└── assets/                  # Static assets
```

### Routing

Routes are lazy-loaded. Auth-protected routes use `meta: { requiresAuth: true }` enforced by a `beforeEach` guard.

**Key pages:** Collection, CoinDetail, AddCoin, EditCoin, Wishlist, Sold, Auctions, Stats, Timeline, Calendar, Showcases, Followers, Notifications, Settings, Admin.

**Public pages:** Login, Register, PublicShowcase.

### State Management (Pinia)

| Store | Key State | Key Actions |
|-------|-----------|-------------|
| `auth` | `token`, `user`, `refreshToken` | `doLogin`, `doRegister`, `doWebAuthnLogin`, `logout` |
| `coins` | `coins[]`, `currentCoin`, `stats`, `valueHistory`, `searchQuery`, `selectedCategory` | `fetchCoins`, `fetchCoin`, `createCoin`, `updateCoin`, `deleteCoin`, `fetchStats` |

Auth state is persisted in `localStorage` and rehydrated on app load.

### API Client

`src/web/src/api/client.ts` provides:

- **Axios instance** with `baseURL = ${VITE_API_BASE_URL}/api`
- **Request interceptor** — attaches `Authorization: Bearer <token>` header
- **Response interceptor** — handles 401 with single-flight refresh:
  1. First 401 triggers refresh token exchange
  2. Concurrent requests queue behind the refresh promise
  3. On success, retries all queued requests with new token
  4. On failure, redirects to login
- **Agent chat** uses native `fetch` + manual SSE parsing (not Axios) for streaming
- **WebAuthn helpers** — binary to base64url conversion utilities

### Composables

| Composable | Purpose |
|-----------|---------|
| `useCoinSearchChat` | Agent chat streaming, message history, SSE parsing |
| `useCollectionFilters` | Search, sort, category, tag filtering |
| `useBulkSelect` | Multi-select with bulk actions (tag, delete) |
| `useDialog` | Confirmation dialog state |
| `useImageProcessor` | Client-side image resize/compress before upload |
| `useNotifications` | Notification polling and unread counts |
| `usePullToRefresh` | Touch-based pull-to-refresh for mobile |
| `usePwa` | PWA install prompt and update detection |
| `useSettingsProfile` | Profile edit form state |
| `useAdminConfig` | Admin settings read/write |

### PWA Configuration

Configured via `vite-plugin-pwa` in `vite.config.ts`:

- **Register type:** `autoUpdate` (no user prompt for updates)
- **Manifest:** Standalone display, dark theme, coin branding icons (192/512)
- **Workbox caching strategies:**
  - API mutations (`POST`/`PUT`/`DELETE`): `NetworkOnly`
  - API reads (`GET`): `NetworkFirst`
  - Uploaded images (`/uploads/`): `CacheFirst`
- **Dev proxy:** `/api` and `/uploads` proxy to Go backend during development

---

## Python Agent Service

### Service Overview

The agent is a **stateless** FastAPI service. It owns no database. All configuration (API keys, model names, system prompts, user context, portfolio data) is passed per-request from the Go API.

**Endpoints:**

| Method | Path | Response | Purpose |
|--------|------|----------|---------|
| `GET` | `/health` | JSON | Health check |
| `POST` | `/api/search/coins` | SSE stream | Coin search / shows / general chat |
| `POST` | `/api/analyze` | JSON | Vision-based coin analysis |
| `POST` | `/api/portfolio/review` | SSE stream | Portfolio review and valuation |
| `POST` | `/api/check-availability` | JSON | Wishlist URL availability check |
| `GET` | `/logs` | JSON | Log ring buffer |
| `PUT` | `/log-level` | JSON | Dynamic log level |

### Supervisor and Team Routing

The supervisor (`app/supervisor.py`) uses an LLM router to classify user intent into one of 11 teams (plus a general fallback):

```
User message
    │
    ▼
[LLM Router] ── classifies intent using last 4 messages
    │
    ├── coin_search    → Coin Search Team
    ├── coin_shows     → Coin Shows Team (may ask for ZIP/city)
    ├── analysis       → Coin Analysis Team
    ├── grading        → Coin Grading Team
    ├── portfolio      → Portfolio Review Team
    ├── gap_analysis   → Gap Analysis Team
    ├── photo_guide    → Photo Guide Team
    ├── price_trends   → Price Trends Team
    ├── similar_lots   → Similar Lots Team
    ├── auction_search → Auction Search Team
    └── general        → General assistant (describes capabilities)
```

The supervisor enforces a max iteration count to prevent infinite loops.

### Team Pipelines

Each team is a LangGraph `StateGraph` with verification stages:

| # | Team | Pipeline | Purpose |
|---|------|----------|---------|
| 1 | Coin Search | Search → Fetch dealer pages → Format | Find available coins for sale |
| 2 | Coin Shows | Search → Verify dates are future → Format | Upcoming numismatic events |
| 3 | Coin Analysis | Vision model analysis → Format | AI image analysis |
| 4 | Portfolio Review | Read holdings → Valuate → Analyze | Collection recommendations |
| 5 | Auction Search | Search auction sources → Fetch → Format | Auction lot discovery |
| 6 | Availability Check | Check URLs → Analyze results → Verdict | Verify listings still for sale |
| 7 | Coin Grading | Analyze photos → Grade → Format | Grade estimation with confidence |
| 8 | Gap Analysis | Read portfolio → Analyze gaps → Suggest | Collection completeness |
| 9 | Photo Guide | Analyze photos → Evaluate → Tips | Photography improvement |
| 10 | Price Trends | Search auctions → Analyze trends → Format | Market price analysis |
| 11 | Similar Lots | Search → Rank → Format | Find similar auction lots |

**Pipeline design rules:**
- Search agents pass only tool-returned data downstream — never invented details
- Verification agents confirm every URL is live and every date is in the future
- All worker outputs conform to defined Pydantic schemas — no free-form text

### LLM Provider Abstraction

`app/llm/provider.py` provides a factory for LLM instances:

| Function | Returns | Use Case |
|----------|---------|----------|
| `get_chat_model()` | `ChatAnthropic` or `ChatOllama` | Nodes that don't need web search |
| `get_search_model()` | Chat model with `web_search` tool bound | Nodes that need web search |
| `create_search_agent()` | ReAct agent with SearXNG tool | Ollama web search (mirrors Anthropic pattern) |

**Provider selection:**
- **Anthropic:** Claude models. Web search uses Claude's built-in `web_search_20250305` tool via `bind_tools`.
- **Ollama:** Self-hosted models. Web search uses `create_react_agent` with a SearXNG HTTP tool.

### SSE Streaming

`app/streaming.py` converts LangGraph execution events into SSE:

```
LangGraph events → stream_graph_events() → SSE text/event-stream
```

Event types emitted: `status`, `text`, `done`, `error`.

Final message extraction priority:
1. Last node's message content
2. Last AI content from any node
3. Full accumulated text

JSON suggestion blocks (coin suggestions, show listings) are extracted from final text when present.

---

## Data Flow Diagrams

### Standard API Request

```
Browser                Go API                    SQLite
  │                      │                         │
  │── GET /api/coins ───▶│                         │
  │                      │── AuthRequired() ──────▶│ validate JWT
  │                      │◀── userId, role ────────│
  │                      │                         │
  │                      │── coinRepo.List() ─────▶│
  │                      │◀── []Coin ──────────────│
  │                      │                         │
  │◀── 200 JSON ────────│                         │
```

### Agent Chat (SSE Streaming)

```
Browser              Go API               Python Agent          LLM Provider
  │                    │                       │                      │
  │── POST /agent/chat▶│                       │                      │
  │                    │── read settings ─────▶│ (DB)                 │
  │                    │── POST /api/search ──▶│                      │
  │                    │   (enriched request)  │                      │
  │                    │                       │── classify intent ──▶│
  │                    │                       │◀── team_id ─────────│
  │                    │                       │                      │
  │                    │                       │── run team graph ───▶│
  │                    │                       │◀── stream events ───│
  │                    │                       │                      │
  │◀── SSE: status ───│◀── SSE: status ──────│                      │
  │◀── SSE: text ─────│◀── SSE: text ────────│                      │
  │◀── SSE: text ─────│◀── SSE: text ────────│                      │
  │◀── SSE: done ─────│◀── SSE: done ────────│                      │
```

Go's `agent_proxy.go` sets `text/event-stream`, `no-cache`, `X-Accel-Buffering: no` and proxies each line with flush-on-boundary.

### Authentication Flow

```
Browser                Go API                  SQLite
  │                      │                       │
  │── POST /auth/login ─▶│                       │
  │   {user, pass}       │── find user ─────────▶│
  │                      │◀── user record ───────│
  │                      │── bcrypt.Compare() ───│
  │                      │── sign JWT (15min) ───│
  │                      │── generate RT (30d) ──│
  │                      │── store RT hash ─────▶│
  │◀── {token, refresh} ─│                       │
  │                      │                       │
  │   ... 15 min later ...                       │
  │                      │                       │
  │── POST /auth/refresh▶│                       │
  │   {refreshToken}     │── hash + lookup ─────▶│
  │                      │── rotate (revoke+new)▶│
  │◀── {token, refresh} ─│  (transaction)        │
```

### Wishlist Availability Check

```
Scheduler / User        Go API               Python Agent
  │                       │                       │
  │── trigger check ─────▶│                       │
  │                       │── load wishlist coins │
  │                       │── for each URL:       │
  │                       │   ├── HTTP GET URL    │
  │                       │   ├── keyword scan    │
  │                       │   └── if ambiguous: ──▶│ POST /check-availability
  │                       │       agent verdict ◀──│
  │                       │── update listing_status│
  │                       │── create notifications │
  │                       │── save run + results   │
```

---

## Database Schema

SQLite via GORM. All tables are auto-migrated from Go model structs in `database/database.go`.

### Core Domain Models

| Model | Table | Key Fields | Relations |
|-------|-------|-----------|-----------|
| `Coin` | `coins` | Name, Category, Denomination, Ruler, Era, Mint, Material, Weight, Grade, PurchasePrice, CurrentValue, IsWishlist, IsSold, ListingStatus, UserID | → User, → []CoinImage, → []Tag (M2M via `coin_tags`) |
| `CoinImage` | `coin_images` | CoinID, FilePath, ImageType (obverse/reverse/detail), IsPrimary | → Coin |
| `Tag` | `tags` | UserID, Name, Color | |
| `CoinTag` | `coin_tags` | CoinID, TagID | Join table |
| `CoinSet` | `coin_sets` | UserID, Name, Color, SetType (`open`/`defined`/`smart`/`goal`), SmartCriteria, TargetCompletionDate | → User, → []CoinSetMembership, → []CoinSetTarget |
| `CoinSetMembership` | `coin_set_memberships` | SetID, CoinID, Notes | Join table for manual set membership |
| `CoinSetTarget` | `coin_set_targets` | SetID, Label, Year, MintMark, Denomination, Country, Material, MatchRules, SortOrder | Target slots for defined/goal completion |
| `CoinSetValuationSnapshot` | `coin_set_valuation_snapshots` | SetID, UserID, SnapshotDate, TotalValue, TotalInvested, CoinCount, CompletionPercentage | Time-series trend data |
| `CoinSetMilestoneAlert` | `coin_set_milestone_alerts` | SetID, UserID, Metric, Threshold, Direction, LastTriggeredAt | Snapshot-triggered milestone notifications |
| `CoinJournal` | `coin_journals` | CoinID, UserID, Entry, CreatedAt | |
| `CoinValueHistory` | `coin_value_histories` | CoinID, UserID, Value, Confidence, RecordedAt | |
| `ValueSnapshot` | `value_snapshots` | UserID, TotalValue, TotalInvested, CoinCount, RecordedAt | |

### Authentication Models

| Model | Table | Key Fields |
|-------|-------|-----------|
| `User` | `users` | Username, Email, PasswordHash, Role (user/admin), AvatarPath, IsPublic, Bio |
| `RefreshToken` | `refresh_tokens` | UserID, TokenHash (SHA-256), ExpiresAt, RevokedAt |
| `ApiKey` | `api_keys` | UserID, KeyHash, KeyPrefix, Name, LastUsedAt, RevokedAt |
| `WebAuthnCredential` | `webauthn_credentials` | UserID, CredentialID, PublicKey, SignCount, Name |

### Social and Notification Models

| Model | Table | Key Fields |
|-------|-------|-----------|
| `Follow` | `follows` | FollowerID, FollowingID, Status (pending/accepted/blocked) |
| `CoinComment` | `coin_comments` | CoinID, UserID, Comment, Rating |
| `Notification` | `notifications` | UserID, Type, Title, Message, ReferenceID, IsRead |
| `Showcase` | `showcases` | UserID, Slug, Title, Description, IsActive |
| `ShowcaseCoin` | `showcase_coins` | ShowcaseID, CoinID, SortOrder |

### Auction and Marketplace Models

| Model | Table | Key Fields |
|-------|-------|-----------|
| `AuctionLot` | `auction_lots` | Source (`numisbids`/`cng`), SourceURL, SourceSaleID, SourceLotID, NumisBidsURL (legacy compatibility), AuctionHouse, SaleName, Title, Estimate, CurrentBid, MaxBid, Status, CoinID (optional), EventID (optional), UserID |
| `AuctionEvent` | `auction_events` | UserID, Title, AuctionHouse, StartDate, EndDate, URL |
| `PriceAlert` | `price_alerts` | AuctionLotID, UserID, TargetPrice |
| `BidReminder` | `bid_reminders` | AuctionLotID, UserID, RemindAt |

Auction provider services intentionally have asymmetric capabilities:

| Service | Provider | Capability boundary |
|---|---|---|
| `CNGAuctionService` | CNG Auctions | Authenticated watched-lot sync with richer hosted-auction fields where CNG exposes them, including current bid, absentee/max bid, close time, and closed-lot outcome signals used by `AuctionWatchlistSyncService` to auto-detect won/lost status. |
| `NumisBidsService` | NumisBids | Watchlist/import tracking from available NumisBids HTML. It does not currently expose or parse a reliable max-bid or final won/lost outcome signal, so NumisBids terminal status and winning bid data are manual unless future verified site data adds those fields. |
| `AuctionWatchlistSyncService` | Provider orchestration | Keeps provider-specific scraping/parsing in the provider services, then applies source-aware upserts and status transitions. Sync writes set `statusSource = sync`; user overrides set `statusSource = manual`. |

### AI and Agent Models

| Model | Table | Key Fields |
|-------|-------|-----------|
| `AgentConversation` | `agent_conversations` | UserID, Title, Messages (JSON), CreatedAt |
| `AvailabilityRun` | `availability_runs` | UserID, TriggerType, CoinsChecked, Available, Unavailable, DurationMs |
| `AvailabilityResult` | `availability_results` | RunID, CoinID, URL, Status, Reason, AgentUsed |
| `ValuationRun` | `valuation_runs` | UserID, TriggerType, Status, TotalCoins, CoinsUpdated, DurationMs |
| `ValuationResult` | `valuation_results` | RunID, CoinID, PreviousValue, EstimatedValue, Confidence, Reasoning |

### System Models

| Model | Table | Key Fields |
|-------|-------|-----------|
| `AppSetting` | `app_settings` | Key, Value — key-value store for runtime configuration |

---

## Authentication and Authorization

The system supports three authentication methods:

**1. JWT Bearer Token (primary)**
- Access token: HS256, 15-minute TTL, claims: `userId`, `username`, `role`
- Refresh token: random `rt_...` string, SHA-256 hashed in DB, 30-day TTL
- Rotation on refresh: old token revoked + new token issued in a single transaction

**2. API Key**
- Generated per user, `ak_...` prefix
- SHA-256 hashed in DB, matched on request
- Supports revocation, tracks `last_used_at`

**3. WebAuthn / Passkeys**
- FIDO2 registration and login ceremonies
- Session data stored in-memory (handler-level map)
- On successful login, issues standard JWT + refresh token pair
- Supports dynamic origin fallback for PWA and mobile

**Auth middleware order** (`middleware/auth.go`):
1. Check `X-API-Key` header → hash and lookup
2. Check `Authorization: Bearer <token>` header → validate JWT
3. Check `?token=` query param → validate JWT (image proxy fallback)
4. Set `userId` and `userRole` in Gin context

**Rate limiting:** Public auth endpoints are rate-limited to 10 requests/minute per IP.

**Admin guard:** `handlers.AdminRequired()` middleware checks `userRole == "admin"`.

---

## AI/Agent Integration Pattern

The Go API acts as a **thin proxy** between the frontend and the Python agent service. This separation ensures:

- The Go API owns all data access and authentication
- The Python agent is stateless and horizontally scalable
- LLM provider credentials live in the Go DB, passed per-request

**Proxy flow (`services/agent_proxy.go`):**

1. Handler reads settings from DB (API keys, model names, system prompts)
2. Handler enriches the request with user context and portfolio data
3. `AgentProxy` POSTs to the Python service with full context
4. For streaming: Go sets SSE headers and proxies each line with flush-on-boundary
5. For structured responses: Go reads the full JSON response and returns it

**Two HTTP clients in the proxy:**
- `streamClient` — no timeout (SSE can run indefinitely)
- `requestClient` — 5-minute timeout (analysis, availability checks)

**Key proxy methods:**
- `StreamChat()` → `POST /api/search/coins` (SSE)
- `CollectPortfolioReview()` → `POST /api/portfolio/review` (collects full SSE, returns final message)
- `AnalyzeCoin()` → `POST /api/analyze` (JSON)
- `CheckAvailability()` → `POST /api/check-availability` (JSON)

---

## Background Schedulers

Two goroutine-based schedulers run in the Go API:

### Wishlist Availability Scheduler

- **Initial delay:** 30 seconds after startup
- **Config:** `WishlistCheckEnabled`, `WishlistCheckStartTime` (HH:MM, default `02:00`), `WishlistCheckInterval` (minutes, default `120`)
- **Behavior:** Loads all wishlist coins with URLs, groups by user, runs `CheckWishlistForUser()` for each
- **Source:** `services/availability_scheduler.go`

### Collection Valuation Scheduler

- **Initial delay:** 60 seconds after startup
- **Config:** `ValuationCheckEnabled`, `ValuationCheckStartTime` (HH:MM, default `03:00`), `ValuationCheckInterval` (days, default `7`)
- **Behavior:** Gets users with owned coins, runs `ValuateCollectionForUser()` for each
- **Source:** `services/valuation_scheduler.go`

Both compute next-run from a daily anchor time plus interval cadence.

---

## Build and Deployment

### Docker Multi-Stage Build (Go + Vue)

`Dockerfile` (root):

```
Stage 1: node:24-alpine@sha256:...
  → npm install + npm run build (Vue SPA)
  → Output: dist/

Stage 2: golang:1.26.5-alpine@sha256:...
  → go build -o ancient-coins-api
  → Output: binary

Stage 3: alpine:3.21@sha256:...
  → Copy binary + Vue dist → /app/wwwroot
  → Create /app/uploads, /app/data
  → EXPOSE 8080
```

### Docker Build (Python Agent)

`src/agent/Dockerfile`:

```
Stage 1: python:3.12-slim@sha256:... (builder)
  → pip install from pyproject.toml

Stage 2: python:3.12-slim@sha256:...
  → Copy site-packages + app/
  → CMD: uvicorn app.main:app --host 0.0.0.0 --port 8081
  → EXPOSE 8081
```

### Build Commands

```bash
# Local development
task build          # Build API + web
task up             # API + web dev servers
task up-all         # API + web + agent dev servers

# Individual builds
cd src/api && go build ./...
cd src/web && npm run build
cd src/agent && pip install -e ".[dev]"

# Tests
cd src/api && go test -v ./...
cd src/agent && pytest tests/ -v

# Linting
cd src/api && go vet ./...
cd src/agent && ruff check app/ tests/
cd src/web && npx vue-tsc --noEmit
```

---

## Configuration Reference

### Go API (`src/api/config/`)

| Env Var | Default | Purpose |
|---------|---------|---------|
| `DB_PATH` | `./ancientcoins.db` | SQLite database file |
| `JWT_SECRET` | (required in prod, min 32 chars) | JWT signing key |
| `PORT` | `8080` | HTTP listen port |
| `UPLOAD_DIR` | `./uploads` | Coin image storage |
| `WEBAUTHN_RP_ID` | `localhost` | WebAuthn relying party ID |
| `WEBAUTHN_ORIGIN` | `http://localhost:8080` | WebAuthn expected origin |
| `CORS_ORIGINS` | (derived) | Allowed CORS origins |
| `AGENT_SERVICE_URL` | `http://localhost:8081` | Python agent base URL |
| `AGENT_INTERNAL_SERVICE_TOKEN` | — | Shared API → agent credential for hardened agent endpoints |

### Python Agent (`src/agent/app/config.py`)

| Env Var | Default | Purpose |
|---------|---------|---------|
| `AGENT_DEBUG` | `false` | Debug mode |
| `AGENT_LOG_LEVEL` | `INFO` | Log level |
| `AGENT_INTERNAL_SERVICE_TOKEN` | — | Shared API → agent credential; required for every endpoint except `/health` and `/ready` |
| `AGENT_SEARXNG_URL` | — | SearXNG instance URL (Ollama search) |
| `AGENT_MAX_SEARCH_RESULTS` | — | Limit search results |
| `AGENT_VERIFICATION_TIMEOUT` | — | URL verification timeout |
| `AGENT_MAX_SUPERVISOR_ITERATIONS` | — | Loop prevention |

### Runtime Settings (DB `app_settings` table)

AI provider, model names, system prompts, scheduler configs, and feature toggles are stored as key-value pairs and managed via the Admin UI. Constants and defaults live in `services/settings_service.go`.

---

## Key Design Decisions

| Decision | Rationale |
|----------|-----------|
| **SQLite over Postgres** | Single-user/small-team app. No connection pooling overhead. Portable file-based DB. |
| **Separate Python agent service** | Go has limited LLM/ML ecosystem. Python owns LangGraph, LangChain, and vision model integrations. Keeps Go API focused on CRUD and auth. |
| **Stateless agent service** | Horizontal scaling without shared state. All context passed per-request. Go owns the source of truth (DB). |
| **SSE over WebSockets** | Unidirectional streaming (server → client) is sufficient. SSE is simpler — no connection upgrade, works through most proxies, auto-reconnects. |
| **Constructor injection (no globals)** | Testable. Explicit dependencies. Enforced by architecture tests. |
| **Refresh token rotation** | Single-use refresh tokens. Revoke-on-rotate prevents replay attacks. |
| **Keyword heuristics before agent escalation** | Availability checks try cheap HTTP + keyword scan first. Only ambiguous cases go to the LLM agent. Saves cost and latency. |
| **PWA over native app** | One codebase for desktop and mobile. Offline-capable with Workbox. Installable from browser. |
| **No emojis in UI** | Numismatic audience. Professional tone. Enforced by convention. |
| **Settings as key-value pairs** | Flexible runtime config without schema migrations. Admin UI for non-technical users. |
