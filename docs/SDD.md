# Software Design Document (SDD)

## Ancient Coins Collection Manager

| | |
|---|---|
| **Version** | 1.0 |
| **Date** | April 2, 2026 |
| **Author** | Brian DeNicola |
| **License** | MIT |

---

## Table of Contents

1. [Introduction](#1-introduction)
2. [System Overview](#2-system-overview)
3. [Architecture Design](#3-architecture-design)
4. [Data Design](#4-data-design)
5. [Component Design](#5-component-design)
6. [Interface Design](#6-interface-design)
7. [Security Design](#7-security-design)
8. [Deployment Design](#8-deployment-design)
9. [Quality Assurance](#9-quality-assurance)
10. [Appendices](#10-appendices)

---

## 1. Introduction

### 1.1 Purpose

This Software Design Document describes the architecture, components, interfaces, and data design of the Ancient Coins Collection Manager — a full-stack Progressive Web Application (PWA) for managing a personal ancient coin collection with AI-powered analysis and search capabilities.

### 1.2 Scope

The system provides:

- **Collection management** — catalog coins with detailed metadata, images, purchase/sale history, and journal entries
- **AI-powered analysis** — vision-model coin identification, value estimation, and portfolio review
- **AI-powered search** — find coins for sale from trusted dealers and discover upcoming coin shows
- **Social features** — follow other collectors, browse their public galleries, comment and rate coins
- **Analytics** — collection statistics, value tracking, timeline views, and trend charts
- **PWA** — installable on mobile/desktop with offline support, camera capture, and biometric login

### 1.3 Intended Audience

- Developers contributing to or maintaining the codebase
- Architects evaluating the system design
- Reviewers assessing security and deployment posture

### 1.4 Definitions and Acronyms

| Term | Definition |
|---|---|
| PWA | Progressive Web Application |
| SSE | Server-Sent Events |
| GORM | Go Object-Relational Mapper |
| LangGraph | LangChain framework for building stateful multi-agent graphs |
| WebAuthn | Web Authentication API (FIDO2/passkeys) |
| JWT | JSON Web Token |
| SearXNG | Privacy-respecting metasearch engine |
| Numista | Online coin catalog and reference database |

---

## 2. System Overview

### 2.1 System Context

```
┌──────────────┐     HTTPS      ┌──────────────────────────────────┐
│              │ ◄────────────► │  Ancient Coins Application       │
│    User      │                │  ┌────────────┐ ┌─────────────┐  │
│  (Browser /  │                │  │  Go API    │ │ Python Agent│  │
│   PWA)       │                │  │  :8080     │►│  :8081      │  │
│              │                │  └────────────┘ └─────────────┘  │
└──────────────┘                └──────────────────────────────────┘
                                          │              │
                                          ▼              ▼
                                   ┌───────────┐  ┌───────────────┐
                                   │  SQLite   │  │ External APIs │
                                   │  Database │  │ (Anthropic,   │
                                   └───────────┘  │  Ollama,      │
                                                  │  Numista,     │
                                                  │  SearXNG)     │
                                                  └───────────────┘
```

### 2.2 Technology Stack

| Layer | Technology | Version |
|---|---|---|
| Backend API | Go, Gin, GORM | Go 1.26.1 |
| Frontend SPA | Vue 3, TypeScript, Pinia, Vite | Vue 3 |
| Agent Service | Python, FastAPI, LangGraph, LangChain | Python 3.12 |
| Database | SQLite (WAL mode) | — |
| Containerization | Docker, Docker Compose | Multi-stage |
| CI/CD | GitHub Actions | — |

### 2.3 Design Principles

1. **Layered architecture** — strict Handler → Service → Repository → Database separation enforced by automated tests
2. **Constructor injection** — all dependencies are injected; no package-level imports of the database layer
3. **Stateless agent** — the Python agent service has no database; all context is passed per-request from the Go API
4. **Settings-driven AI** — AI provider, model, prompts, and API keys are stored in the database and configurable at runtime through Admin settings
5. **Privacy by default** — user collections are private; social sharing requires explicit opt-in and follow acceptance

---

## 3. Architecture Design

### 3.1 High-Level Architecture

The system consists of two containers orchestrated by Docker Compose:

| Container | Contents | Port | Responsibilities |
|---|---|---|---|
| `app` | Go API binary + Vue SPA static assets | 8080 | REST API, authentication, data persistence, SPA serving, SSE proxy to agent |
| `agent` | Python FastAPI service | 8081 | AI inference, multi-agent workflows, web search, vision analysis |

The Go API serves the Vue SPA as static files and proxies AI requests to the Python agent service via internal HTTP. The frontend communicates exclusively with the Go API on port 8080.

### 3.2 Go API — Layered Architecture

```
┌─────────────────────────────────────────────────────┐
│                     main.go                         │
│  (bootstrap, DI wiring, route registration)         │
├─────────────────────────────────────────────────────┤
│                   Middleware                         │
│  AuthRequired (JWT / API Key) │ RateLimit (per-IP)  │
├─────────────────────────────────────────────────────┤
│                    Handlers                          │
│  Parse request → call service/repo → return JSON    │
├─────────────────────────────────────────────────────┤
│                    Services                          │
│  Business logic, domain rules, external proxies     │
├─────────────────────────────────────────────────────┤
│                  Repositories                        │
│  All GORM queries, transactions, scopes             │
├─────────────────────────────────────────────────────┤
│                    Database                          │
│  GORM + SQLite, AutoMigrate, WAL mode               │
└─────────────────────────────────────────────────────┘
```

**Architectural rules (enforced by `architecture_test.go`):**

| Rule | Enforcement |
|---|---|
| Only `main.go` imports the `database` package | AST analysis of all `.go` files |
| Handlers contain no raw SQL | String scan of handler source files |
| Services are HTTP-agnostic | No `gin.Context` in service signatures |

**Package import constraints:**

| Package | May Import |
|---|---|
| `handlers/` | `services/`, `repository/`, `models/` |
| `services/` | `repository/`, `models/` |
| `repository/` | `models/`, `gorm.io/gorm` |
| `models/` | Standard library only |
| `middleware/` | `models/`, `gorm.io/gorm` |

### 3.3 Python Agent — Multi-Agent Architecture

```
┌─────────────────────────────────────────────┐
│              FastAPI Service                 │
│         /api/search/coins                   │
│         /api/search/shows                   │
│         /api/analyze                        │
│         /api/portfolio/review               │
├─────────────────────────────────────────────┤
│             Supervisor / Router              │
│  Classifies intent → dispatches to team     │
├────────┬──────────┬───────────┬─────────────┤
│ Team 1 │  Team 2  │  Team 3   │   Team 4    │
│ Coin   │  Coin    │  Coin     │  Portfolio  │
│ Search │  Shows   │  Analysis │  Review     │
└────────┴──────────┴───────────┴─────────────┘
```

**Team pipelines:**

| Team | Pipeline | Purpose |
|---|---|---|
| Team 1: Coin Search | Search → Fetch → Format | Find coins for sale from trusted dealers |
| Team 2: Coin Shows | Search → Verify → Format | Find upcoming coin shows near the user |
| Team 3: Coin Analysis | Analyze → Format | Vision-model coin image analysis |
| Team 4: Portfolio Review | Reader → Valuation → Analysis | Assess collection composition and value |

**Agent design rules:**
- Search agents pass only tool-returned data downstream — never invented details
- Verification agents confirm every URL is live and every date is in the future
- All worker outputs conform to a defined schema
- The supervisor enforces max iteration count to prevent loops
- Streaming responses are delivered via SSE through the Go API proxy

### 3.4 Frontend — Vue 3 SPA Architecture

```
┌──────────────────────────────────────────────┐
│                  App.vue                      │
│            (shell, nav, theme)                │
├──────────────────────────────────────────────┤
│                   Router                      │
│         (auth guard, 23 routes)               │
├──────────────────────────────────────────────┤
│                   Pages                       │
│  Collection │ Detail │ Add/Edit │ Wishlist    │
│  Sold │ Stats │ Timeline │ Settings │ Admin   │
│  Followers │ Gallery │ Login │ Register       │
│  Showcases │ Calendar │ Notifications │ Chat  │
├──────────────────────────────────────────────┤
│                Components                     │
│  CoinCard │ CoinForm │ CoinSearchChat         │
│  ImageGallery │ ImageProcessor │ SwipeGallery │
│  SearchBar │ CategoryFilter │ SortSelect      │
│  PurchaseModal │ SellModal │ PullToRefresh    │
├──────────────────────────────────────────────┤
│              Pinia Stores                     │
│         auth (JWT/user)                       │
│         coins (collection state)              │
├──────────────────────────────────────────────┤
│              API Client                       │
│  Axios instance + interceptors                │
│  Token refresh queue + SSE streaming          │
└──────────────────────────────────────────────┘
```

**Frontend conventions:**
- `<script setup lang="ts">` with Composition API
- Optional chaining (`?.`) and nullish coalescing (`??`) on all array access
- All API calls through centralized `api/client.ts`
- Dark theme default with CSS custom properties
- Icons from `lucide-vue-next`
- No emojis in UI text, prompts, or AI responses

---

## 4. Data Design

### 4.1 Database Engine

SQLite with GORM, configured with:
- `journal_mode=WAL` for concurrent read performance
- `foreign_keys=ON` for referential integrity
- Schema managed entirely by GORM `AutoMigrate` (no standalone migration files)

### 4.2 Entity-Relationship Diagram

```
┌──────────┐    1:N    ┌───────────┐    1:N    ┌────────────┐
│   User   │◄─────────►│   Coin    │◄─────────►│ CoinImage  │
│          │           │           │           │            │
│ id (PK)  │           │ id (PK)   │           │ id (PK)    │
│ username │           │ user_id   │           │ coin_id    │
│ email    │           │ name      │           │ file_path  │
│ password │           │ category  │           │ image_type │
│ role     │           │ material  │           │ is_primary │
│ avatar   │           │ ruler/era │           └────────────┘
│ is_public│           │ grade     │
│ bio      │           │ prices    │    1:N    ┌────────────────┐
│ zip_code │           │ is_wishlist│◄─────────►│ CoinValueHistory│
└──────┬───┘           │ is_sold   │           │ value          │
       │               │ is_private│           │ confidence     │
       │               │ analyses  │           │ recorded_at    │
       │               │ notes     │           └────────────────┘
       │               └─────┬─────┘
       │                     │  1:N   ┌─────────────┐
       │                     ├───────►│ CoinJournal  │
       │                     │        │ entry        │
       │                     │        └─────────────┘
       │                     │  1:N   ┌─────────────┐
       │                     └───────►│ CoinComment  │
       │                              │ comment      │
       │                              │ rating       │
       │                              │ user_id      │
       │                              └─────────────┘
       │
       │  1:N    ┌──────────────────┐
       ├────────►│  RefreshToken    │
       │         │  token_hash      │
       │         │  expires_at      │
       │         └──────────────────┘
       │
       │  1:N    ┌──────────────────┐
       ├────────►│  ApiKey          │
       │         │  key_hash        │
       │         │  name/prefix     │
       │         │  revoked_at      │
       │         └──────────────────┘
       │
       │  1:N    ┌──────────────────┐
       ├────────►│ WebAuthnCredential│
       │         │  credential_id   │
       │         │  public_key      │
       │         │  sign_count      │
       │         └──────────────────┘
       │
       │  1:N    ┌──────────────────┐
       ├────────►│ AgentConversation│
       │         │  title           │
       │         │  messages (JSON) │
       │         └──────────────────┘
       │
       │  1:N    ┌──────────────────┐
       ├────────►│  ValueSnapshot   │
       │         │  total_value     │
       │         │  total_invested  │
       │         │  coin_count      │
       │         └──────────────────┘
       │
       │  N:N (self-referential)
       └────────►┌──────────────────┐
                 │  Follow          │
                 │  follower_id     │
                 │  following_id    │
                 │  status          │
                 │  (pending/       │
                 │   accepted/      │
                 │   blocked)       │
                 └──────────────────┘

┌───────────────┐
│  AppSetting   │  (key-value store for runtime configuration)
│  key (PK)     │
│  value        │
└───────────────┘
```

### 4.3 Core Models

#### User
| Field | Type | Constraints |
|---|---|---|
| ID | uint | Primary key |
| Username | string | Unique, not null |
| Email | string | Unique |
| PasswordHash | string | Not null, hidden from JSON |
| Role | UserRole | `admin` or `user`, default `user` |
| AvatarPath | string | Relative file path |
| IsPublic | bool | Default false |
| Bio | string | Text |
| ZipCode | string | Max 10 chars |
| CreatedAt | time.Time | Auto |

#### Coin
| Field | Type | Constraints |
|---|---|---|
| ID | uint | Primary key |
| UserID | uint | Foreign key → User, not null |
| Name | string | Not null |
| Category | Category | Enum: Roman, Greek, Byzantine, Modern, Other |
| Material | Material | Enum: Gold, Silver, Bronze, Copper, Electrum, Other |
| Denomination, Ruler, Era, Mint, Grade | string | Optional metadata |
| ObverseInscription, ReverseInscription | string | Coin text |
| ObverseDescription, ReverseDescription | string | Visual descriptions |
| RarityRating | string | Optional |
| PurchasePrice, CurrentValue | *float64 | Nullable currency values |
| PurchaseDate | *time.Time | Nullable |
| PurchaseLocation | string | Optional |
| Notes | string | Free-form text |
| AIAnalysis, ObverseAnalysis, ReverseAnalysis | string | AI-generated analysis text |
| ReferenceURL, ReferenceText | string | External references |
| IsWishlist | bool | Default false |
| IsSold | bool | Default false |
| SoldPrice | *float64 | Nullable |
| SoldDate | *time.Time | Nullable |
| SoldTo | string | Optional |
| IsPrivate | bool | Default false |
| Images | []CoinImage | Has-many relationship |

#### AppSetting
| Field | Type | Constraints |
|---|---|---|
| Key | string | Primary key |
| Value | string | Setting value |

Key settings stored: `AIProvider`, `AnthropicAPIKey`, `AnthropicModel`, `OllamaURL`, `OllamaModel`, `OllamaTimeout`, `SearXNGURL`, `NumistaAPIKey`, `CoinSearchPrompt`, `CoinShowsPrompt`, `ValuationPrompt`, `ObversePrompt`, `ReversePrompt`, `TextExtractionPrompt`, `LogLevel`.

### 4.4 Enumerations

| Type | Values |
|---|---|
| UserRole | `admin`, `user` |
| Category | `Roman`, `Greek`, `Byzantine`, `Modern`, `Other` |
| Material | `Gold`, `Silver`, `Bronze`, `Copper`, `Electrum`, `Other` |
| ImageType | `obverse`, `reverse`, `detail`, `other` |
| FollowStatus | `pending`, `accepted`, `blocked` |

---

## 5. Component Design

### 5.1 Go API Components

#### 5.1.1 Handlers (HTTP Layer)

Each handler is a thin struct receiving repositories and services via constructor injection.

| Handler | File | Responsibilities |
|---|---|---|
| AuthHandler | `handlers/auth.go` | Login, register, token refresh, setup check |
| WebAuthnHandler | `handlers/webauthn.go` | Passkey registration and login ceremonies |
| UserHandler | `handlers/user.go` | Profile, password, avatar, export/import |
| CoinHandler | `handlers/coins.go` | CRUD, purchase, sell, stats, suggestions, value history |
| ImageHandler | `handlers/images.go` | Image upload (file/base64), delete, proxy, scrape |
| AnalysisHandler | `handlers/analysis.go` | AI analysis, text extraction, Ollama status |
| JournalHandler | `handlers/journal.go` | Coin journal CRUD |
| AgentHandler | `handlers/agent.go` | Chat streaming, model listing, prompts, portfolio, value estimation |
| ConversationHandler | `handlers/conversations.go` | Saved conversation CRUD |
| SocialHandler | `handlers/social.go` | Follow/unfollow, block, comments, ratings, public galleries |
| AdminHandler | `handlers/admin.go` | User management, settings, logs, connectivity tests |
| ApiKeyHandler | `handlers/api_keys.go` | API key generation, listing, revocation |
| NumistaHandler | `handlers/numista.go` | Numista catalog search proxy |

#### 5.1.2 Services (Business Logic)

| Service | File | Responsibilities |
|---|---|---|
| AuthService | `services/auth_service.go` | User registration (first user = admin), authentication, JWT generation, refresh token rotation |
| CoinService | `services/coin_service.go` | Coin lifecycle (create, update, delete, purchase, sell), automatic value history recording and snapshots |
| ImageService | `services/image_service.go` | Image upload (file/base64 with 20MB limit), deletion, file system management |
| SocialService | `services/social_service.go` | Follow workflow validation, privacy enforcement, public profile building |
| AgentProxy | `services/agent_proxy.go` | HTTP client to Python agent; SSE stream proxying and collection |
| OllamaService | `services/ollama_service.go` | Direct Ollama API client for image analysis and text extraction |
| Settings | `services/settings_service.go` | Runtime settings CRUD with defaults, backed by `AppSetting` table |
| Logger | `services/logger.go` | In-memory ring-buffer logger (1000 entries), runtime log level changes |

#### 5.1.3 Repositories (Data Access)

| Repository | File | Key Operations |
|---|---|---|
| AuthRepository | `repository/auth_repository.go` | User CRUD, refresh token management with transactional rotation |
| UserRepository | `repository/user_repository.go` | Profile updates, privacy transitions with cascading follower cleanup |
| CoinRepository | `repository/coin_repository.go` | Paginated listing with filters/sort/search, stats aggregation, cascading delete |
| ImageRepository | `repository/image_repository.go` | Image CRUD with transactional primary flag management |
| AnalysisRepository | `repository/analysis_repository.go` | Coin field updates for AI analysis results |
| JournalRepository | `repository/journal_repository.go` | Journal entry CRUD |
| AgentRepository | `repository/agent_repository.go` | Portfolio summary aggregation, value history recording |
| ConversationRepository | `repository/conversation_repository.go` | Saved conversation CRUD |
| SocialRepository | `repository/social_repository.go` | Follow graph, user search, public coin access, comments/ratings with N+1 avoidance |
| AdminRepository | `repository/admin_repository.go` | User listing, cascading user deletion, full data export |
| ApiKeyRepository | `repository/api_key_repository.go` | API key CRUD with hash-based lookup |
| WebAuthnRepository | `repository/webauthn_repository.go` | WebAuthn credential storage and lookup |

**Shared GORM Scopes** (`repository/scopes.go`):

| Scope | Purpose |
|---|---|
| `OwnedBy(userID)` | Filter by `user_id` |
| `ByID(id)` | Filter by primary key |
| `OwnedByID(id, userID)` | Filter by both PK and owner |
| `ActiveCollection(userID)` | Non-wishlist, non-sold coins |
| `PublicCoins(userID)` | Public, active coins for social viewing |
| `ByCoinID(coinID)` | Filter by `coin_id` |

### 5.2 Python Agent Components

#### 5.2.1 FastAPI Application

| File | Component | Purpose |
|---|---|---|
| `app/main.py` | FastAPI app | App initialization, CORS, health check, log endpoints |
| `app/routes.py` | API router | `/api/search/coins`, `/api/search/shows`, `/api/analyze`, `/api/portfolio/review` |
| `app/config.py` | Settings | Environment variable loading (`AGENT_DEBUG`, `AGENT_LOG_LEVEL`, etc.) |
| `app/logging_config.py` | Logging | Ring-buffer handler + stdout, runtime level changes |
| `app/streaming.py` | SSE helpers | LangGraph output → SSE event conversion (`status`, `text`, `done`, `error`) |

#### 5.2.2 LLM Provider Layer

| File | Function | Purpose |
|---|---|---|
| `app/llm/provider.py` | `get_chat_model()` | Factory for Anthropic (`ChatAnthropic`) or Ollama (`ChatOllama`) |
| | `get_search_model()` | Chat model with web search tool binding (Anthropic built-in or SearXNG) |
| | `create_search_agent()` | ReAct agent for Ollama-based search using SearXNG tool |

#### 5.2.3 Team Graphs

**Team 1: Coin Search** (`app/teams/coin_search.py`)
```
search_node → fetch_node → format_node → END
```
- **search_node** — Web search for dealer listings (Anthropic search or SearXNG ReAct)
- **fetch_node** — Parallel fetch of up to 5 dealer URLs with specialized HTML parsing
- **format_node** — Structure results as JSON coin suggestions

**Team 2: Coin Shows** (`app/teams/coin_shows.py`)
```
search_node → verify_node → format_node → END
```
- **search_node** — Web search for upcoming shows with location context
- **verify_node** — Validate dates are future, events not cancelled, geographic relevance
- **format_node** — Structure results as JSON show listings

**Team 3: Coin Analysis** (`app/teams/coin_analysis.py`)
```
analysis_node → format_node → END
```
- **analysis_node** — Vision model analysis of base64 coin images with optional side focus
- **format_node** — Clean and standardize analysis into markdown

**Team 4: Portfolio Review** (`app/teams/portfolio_review.py`)
```
reader_node → valuation_node → analysis_node → END
```
- **reader_node** — Summarize portfolio composition from raw data
- **valuation_node** — Market commentary using portfolio summary
- **analysis_node** — Final narrative report with recommendations

#### 5.2.4 Tools

| Tool | File | Purpose |
|---|---|---|
| `create_searxng_search()` | `app/tools/search.py` | SearXNG web search (used by Ollama search agent) |
| `fetch_dealer_page()` | `app/tools/search.py` | Fetch and parse dealer pages (VCoins, MA-Shops, generic) |
| `verify_url()` | `app/tools/search.py` | URL verification (defined, not currently wired) |

#### 5.2.5 Supervisor / Router

The supervisor (`app/supervisor.py`) classifies user intent and routes to the appropriate team:

| Route | Target | Trigger |
|---|---|---|
| `coin_search` | Team 1 | Requests to find/buy coins |
| `coin_shows` | Team 2 | Requests about events/shows |
| `analysis` | Team 3 | Image analysis requests |
| `portfolio` | Team 4 | Portfolio review requests |
| `general` | Direct response | General numismatic questions |

### 5.3 Vue Frontend Components

#### 5.3.1 Pages

| Page | Route | Purpose |
|---|---|---|
| LoginPage | `/login` | Username/password and biometric sign-in |
| RegisterPage | `/register` | Account creation |
| CollectionPage | `/` | Main collection browser with search, filter, sort, grid/swipe views |
| CoinDetailPage | `/coin/:id` | Full coin detail with images, AI analysis, value estimate, journal |
| AddCoinPage | `/add` | New coin form with image upload and OCR |
| EditCoinPage | `/edit/:id` | Edit existing coin |
| WishlistPage | `/wishlist` | Wishlist with AI coin search chat |
| SoldPage | `/sold` | Sold coins list |
| StatsPage | `/stats` | Collection statistics and charts |
| TimelinePage | `/timeline` | Purchase timeline grouped by month/year |
| SettingsPage | `/settings` | Profile, appearance, data management, API keys, biometrics |
| AdminPage | `/admin` | User management, AI settings, system settings, logs |
| FollowersPage | `/followers` | Follow/follower management |
| FollowerGalleryPage | `/followers/:username/gallery` | Public gallery of a followed user |
| FollowerCoinDetailPage | `/followers/:username/coins/:coinId` | Public coin detail with comments/ratings |
| ImageProcessorPage | `/process-image` | Standalone image cleanup tool |

#### 5.3.2 Stores

**Auth Store** (`stores/auth.ts`):
- State: `token`, `user`
- Actions: `doLogin()`, `doRegister()`, `doWebAuthnLogin()`, `logout()`, `setTokens()`
- Persists to `localStorage`

**Coins Store** (`stores/coins.ts`):
- State: `coins`, `currentCoin`, `total`, `loading`, `stats`, `valueHistory`, `selectedCategory`, `searchQuery`, `galleryIndex`
- Actions: `fetchCoins()`, `fetchCoin()`, `addCoin()`, `editCoin()`, `removeCoin()`, `fetchStats()`, `fetchValueHistory()`

#### 5.3.3 API Client

The centralized API client (`api/client.ts`) uses Axios with:
- **Request interceptor** — injects `Authorization: Bearer <token>` from `localStorage`
- **Response interceptor** — handles 401 with automatic token refresh and request queuing
- **SSE streaming** — `agentChatStream()` uses native `fetch()` for streaming AI responses
- **Sanitization** — `sanitizeCoin()` normalizes nullable fields for backend compatibility

---

## 6. Interface Design

### 6.1 REST API Endpoints

#### 6.1.1 Authentication (Public)

| Method | Path | Handler | Description |
|---|---|---|---|
| GET | `/api/auth/setup` | AuthHandler.NeedsSetup | Check if initial setup is required |
| POST | `/api/auth/register` | AuthHandler.Register | Create account (first user becomes admin) |
| POST | `/api/auth/login` | AuthHandler.Login | Authenticate, receive JWT + refresh token |
| POST | `/api/auth/refresh` | AuthHandler.Refresh | Rotate refresh token, get new access token |

Rate limit: 10 requests per minute per IP on auth endpoints.

#### 6.1.2 WebAuthn (Mixed)

| Method | Path | Auth | Description |
|---|---|---|---|
| POST | `/api/auth/webauthn/login/begin` | Public | Start passkey login ceremony |
| POST | `/api/auth/webauthn/login/finish` | Public | Complete passkey login |
| GET | `/api/auth/webauthn/check` | Public | Check if user has registered credentials |
| POST | `/api/auth/webauthn/register/begin` | Protected | Start passkey registration |
| POST | `/api/auth/webauthn/register/finish` | Protected | Complete passkey registration |
| GET | `/api/auth/webauthn/credentials` | Protected | List user's credentials |
| DELETE | `/api/auth/webauthn/credentials/:id` | Protected | Delete a credential |

#### 6.1.3 Collection (Protected)

| Method | Path | Description |
|---|---|---|
| GET | `/api/coins` | List coins with pagination, filtering, sorting, search |
| GET | `/api/coins/:id` | Get coin detail with images |
| POST | `/api/coins` | Create a new coin |
| PUT | `/api/coins/:id` | Update a coin |
| POST | `/api/coins/:id/purchase` | Convert wishlist coin to purchased |
| POST | `/api/coins/:id/sell` | Mark coin as sold |
| DELETE | `/api/coins/:id` | Delete coin and all related data |
| GET | `/api/stats` | Collection statistics |
| GET | `/api/suggestions` | Autocomplete suggestions for form fields |
| GET | `/api/value-history` | Collection value snapshots over time |
| GET | `/api/coins/:id/value-history` | Individual coin value history |

#### 6.1.4 Images (Protected)

| Method | Path | Description |
|---|---|---|
| POST | `/api/coins/:id/images` | Upload image file |
| POST | `/api/coins/:id/images/base64` | Upload base64-encoded image |
| DELETE | `/api/coins/:id/images/:imageId` | Delete image |
| GET | `/api/proxy-image` | Proxy external image URL |
| GET | `/api/scrape-image` | Scrape image from external page |

#### 6.1.5 AI / Agent (Protected)

| Method | Path | Description |
|---|---|---|
| POST | `/api/agent/chat` | Stream AI chat response (SSE) |
| GET | `/api/agent/status` | Agent service health check |
| GET | `/api/agent/models` | List available AI models |
| GET | `/api/agent/coin-search-prompt` | Get configured coin search prompt |
| GET | `/api/agent/coin-shows-prompt` | Get configured coin shows prompt |
| GET | `/api/agent/valuation-prompt` | Get configured valuation prompt |
| GET | `/api/agent/portfolio-summary` | Get user's portfolio summary |
| POST | `/api/coins/:id/analyze` | Run AI analysis on coin images |
| DELETE | `/api/coins/:id/analyze` | Delete analysis for a side |
| POST | `/api/coins/:id/estimate-value` | AI value estimation |
| POST | `/api/extract-text` | OCR text extraction from image |
| GET | `/api/ollama-status` | Check Ollama model availability |

#### 6.1.6 Journal (Protected)

| Method | Path | Description |
|---|---|---|
| GET | `/api/coins/:id/journal` | List journal entries |
| POST | `/api/coins/:id/journal` | Add journal entry |
| DELETE | `/api/coins/:id/journal/:entryId` | Delete journal entry |

#### 6.1.7 Conversations (Protected)

| Method | Path | Description |
|---|---|---|
| GET | `/api/agent/conversations` | List saved conversations |
| GET | `/api/agent/conversations/:id` | Get conversation |
| POST | `/api/agent/conversations` | Save conversation |
| DELETE | `/api/agent/conversations/:id` | Delete conversation |

#### 6.1.8 Social (Protected)

| Method | Path | Description |
|---|---|---|
| POST | `/api/social/follow/:userId` | Follow a user |
| DELETE | `/api/social/follow/:userId` | Unfollow a user |
| PUT | `/api/social/followers/:userId/accept` | Accept follow request |
| PUT | `/api/social/followers/:userId/block` | Block a user |
| DELETE | `/api/social/followers/:userId/block` | Unblock a user |
| GET | `/api/social/followers` | List followers |
| GET | `/api/social/following` | List following |
| GET | `/api/social/blocked` | List blocked users |
| GET | `/api/social/following/:userId/coins` | View followed user's coins |
| GET | `/api/social/following/:userId/coins/:coinId` | View specific coin |
| GET | `/api/users/search` | Search users |
| GET | `/api/users/:username` | Get public profile |
| POST | `/api/social/coins/:coinId/comments` | Add comment |
| GET | `/api/social/coins/:coinId/comments` | List comments |
| DELETE | `/api/social/coins/:coinId/comments/:commentId` | Delete comment |
| PUT | `/api/social/coins/:coinId/rating` | Rate a coin |
| GET | `/api/social/coins/:coinId/rating` | Get rating |

#### 6.1.9 User (Protected)

| Method | Path | Description |
|---|---|---|
| GET | `/api/auth/me` | Get current user info |
| POST | `/api/auth/change-password` | Change password |
| PUT | `/api/user/profile` | Update profile |
| POST | `/api/user/avatar` | Upload avatar |
| DELETE | `/api/user/avatar` | Delete avatar |
| GET | `/api/user/export` | Export collection as ZIP |
| POST | `/api/user/import` | Import collection from ZIP |

#### 6.1.10 API Keys (Protected)

| Method | Path | Description |
|---|---|---|
| POST | `/api/auth/api-keys` | Generate API key |
| GET | `/api/auth/api-keys` | List API keys |
| DELETE | `/api/auth/api-keys/:id` | Revoke API key |

#### 6.1.11 Admin (Admin Only)

| Method | Path | Description |
|---|---|---|
| GET | `/api/admin/users` | List all users |
| DELETE | `/api/admin/users/:id` | Delete user (cascade) |
| POST | `/api/admin/users/:id/reset-password` | Reset user password |
| GET | `/api/admin/settings` | Get all settings |
| GET | `/api/admin/settings/defaults` | Get default settings |
| PUT | `/api/admin/settings` | Update settings |
| GET | `/api/admin/logs` | Get application logs |
| GET | `/api/admin/test-anthropic` | Test Anthropic API connectivity |
| GET | `/api/admin/test-searxng` | Test SearXNG connectivity |

#### 6.1.12 External Catalog (Protected)

| Method | Path | Description |
|---|---|---|
| GET | `/api/numista/search` | Search Numista coin catalog |

### 6.2 Agent Service Internal API

These endpoints are called exclusively by the Go API, not exposed to the frontend.

| Method | Path | Request Model | Response | Description |
|---|---|---|---|---|
| GET | `/health` | — | JSON | Health check |
| GET | `/logs` | query: limit, level | JSON | Retrieve log buffer |
| PUT | `/log-level` | JSON: {level} | JSON | Change log level |
| POST | `/api/search/coins` | CoinSearchRequest | SSE stream | Coin search pipeline |
| POST | `/api/search/shows` | CoinShowSearchRequest | SSE stream | Coin shows pipeline |
| POST | `/api/analyze` | AnalyzeRequest | JSON | Image analysis |
| POST | `/api/portfolio/review` | PortfolioReviewRequest | SSE stream | Portfolio review pipeline |

**SSE Event Schema:**
```
{"type": "status",  "message": "..."}        // Progress update
{"type": "text",    "text": "..."}           // Content chunk
{"type": "done",    "message": "...",        // Final result
                    "suggestions": [...]}     // Optional structured data
{"type": "error",   "message": "..."}        // Error
```

### 6.3 Inter-Service Communication

```
Browser ──HTTP──► Go API (:8080) ──HTTP──► Python Agent (:8081)
                     │                          │
                     │ SSE proxy                 │ SSE stream
                     │◄─────────────────────────┤
                     │                          │
                     ▼                          ▼
                  SQLite                   Anthropic API
                                           Ollama API
                                           SearXNG
```

The Go API acts as the sole gateway. The agent service is never contacted directly by the frontend. SSE streams from the agent are proxied through the Go API with flush-based streaming.

---

## 7. Security Design

### 7.1 Authentication

The system implements a multi-layered authentication strategy:

**Primary: JWT Bearer Tokens**
- Algorithm: HMAC-SHA256 (HS256)
- Access token lifetime: 15 minutes
- Claims: `userId`, `username`, `role`, `exp`, `iat`
- Transmitted via `Authorization: Bearer <token>` header

**Token Refresh**
- Refresh token: random 32 bytes, prefixed `rt_`
- Lifetime: 30 days
- Only SHA-256 hash stored in database
- Rotation is transactional: old token revoked atomically with new token creation
- Client-side automatic refresh with request queuing on 401

**Alternative: API Keys**
- Format: random 32 bytes, prefixed `ak_`
- Only SHA-256 hash stored in database
- Transmitted via `X-API-Key` header
- Middleware checks API key before JWT (priority order)
- Supports revocation via `revoked_at` timestamp
- `last_used_at` updated on each use

**Biometric: WebAuthn / Passkeys**
- FIDO2/WebAuthn protocol via `go-webauthn/webauthn`
- Platform authenticator preferred (fingerprint, Face ID)
- Resident key and user verification preferred
- On successful biometric login, issues standard JWT + refresh tokens
- Session data stored in-memory during registration/login ceremonies

### 7.2 Authorization

| Level | Mechanism |
|---|---|
| Route-level | `middleware.AuthRequired()` — rejects unauthenticated requests |
| Role-based | `AdminHandler.AdminRequired()` — gates admin-only endpoints |
| Resource-level | Repository scopes filter by `user_id` — users can only access their own data |
| Social privacy | Follow status + `IsPublic` flag — public galleries require accepted follow relationship |

### 7.3 Data Protection

| Concern | Approach |
|---|---|
| Passwords | bcrypt hashing |
| Refresh tokens | SHA-256 hash stored, plaintext never persisted |
| API keys | SHA-256 hash stored, plaintext returned once at creation |
| WebAuthn keys | Public key stored, private key never leaves authenticator |
| AI API keys | Stored in `AppSetting` table (admin-managed) |
| File uploads | Stored on filesystem under `uploads/`; DB stores relative paths |
| Error messages | Internal errors logged server-side; generic messages returned to client |

### 7.4 Rate Limiting

- In-memory per-IP rate limiter on authentication endpoints
- 10 requests per minute window
- Cleanup goroutine for expired entries
- Returns HTTP 429 on breach

---

## 8. Deployment Design

### 8.1 Container Architecture

```
┌─────────────────────────────────────────────┐
│             Docker Compose                   │
│                                             │
│  ┌─────────────────────┐  ┌──────────────┐  │
│  │       app            │  │    agent     │  │
│  │ ┌─────────────────┐  │  │              │  │
│  │ │  Go API binary   │  │  │  Python      │  │
│  │ │  (port 8080)     │──│──│  FastAPI     │  │
│  │ ├─────────────────┤  │  │  (port 8081) │  │
│  │ │  Vue SPA         │  │  │              │  │
│  │ │  (/app/wwwroot)  │  │  │  uvicorn     │  │
│  │ └─────────────────┘  │  └──────────────┘  │
│  │                      │                    │
│  │  Volumes:            │                    │
│  │  - db-data:/app/data │                    │
│  │  - uploads:/app/     │                    │
│  │    uploads           │                    │
│  └─────────────────────┘                    │
└─────────────────────────────────────────────┘
```

### 8.2 Docker Build Strategy

**App Container** (root `Dockerfile`) — 3-stage build:

| Stage | Base Image | Output |
|---|---|---|
| `web-build` | `node:24-alpine` | Vue production bundle (`/web/dist`) |
| `api-build` | `golang:1.26-alpine` | Static Go binary (`CGO_ENABLED=0`) |
| Final | `alpine:3.21` | Combined binary + SPA assets |

**Agent Container** (`src/agent/Dockerfile`) — 2-stage build:

| Stage | Base Image | Output |
|---|---|---|
| Builder | `python:3.12-slim` | Installed Python packages |
| Final | `python:3.12-slim` | Application + uvicorn |

### 8.3 CI/CD Pipeline

**Workflow: `ci.yml`** (Pull requests + pushes to `main`/`beta`):
1. Go: `go build` → `go vet` → architecture tests
2. Vue: `npm ci` → `npx vue-tsc --noEmit`
3. Python: `pip install` → `ruff check` → `pytest`

**Workflow: `docker-publish.yml`** (Push to `main`):
- Builds and pushes both container images to Docker Hub
- Tags: long SHA, short SHA, `latest`

**Workflow: `docker-publish-beta.yml`** (Push to `beta`):
- Same as production but tags with `beta` instead of `latest`

**Dependabot**: Weekly update checks for Go modules, npm packages, Python packages, and GitHub Actions.

### 8.4 Environment Configuration

| Variable | Default | Description |
|---|---|---|
| `JWT_SECRET` | (generated) | JWT signing key (min 32 chars) |
| `DB_PATH` | `./ancientcoins.db` | SQLite database file path |
| `PORT` | `8080` | HTTP server port |
| `UPLOAD_DIR` | `./uploads` | Image upload directory |
| `WEBAUTHN_RP_ID` | `localhost` | WebAuthn Relying Party ID |
| `WEBAUTHN_ORIGIN` | `http://localhost:8080` | WebAuthn allowed origin |
| `AGENT_SERVICE_URL` | `http://agent:8081` | Python agent service URL |
| `AGENT_LOG_LEVEL` | `INFO` | Python agent log level |
| `AGENT_DEBUG` | `false` | Enable agent debug mode (Swagger docs) |

---

## 9. Quality Assurance

### 9.1 Testing Strategy

| Layer | Tool | Scope |
|---|---|---|
| Go API | `go test` | Architecture rule enforcement (import constraints, no raw SQL in handlers) |
| Vue Frontend | `vue-tsc` | TypeScript type checking (stricter in Docker build) |
| Python Agent | `pytest` | API endpoint validation (12 tests: 6 API + 6 streaming) |
| Python Agent | `ruff` | Linting |
| Go API | `go vet` | Static analysis |

### 9.2 Architecture Tests

The `architecture_test.go` file enforces structural rules via AST analysis:
- Scans all `.go` files to verify no handler/service/middleware imports the `database` package
- Scans handler files for raw SQL strings to prevent data-access leakage

### 9.3 Build Verification Commands

```bash
# Go API
cd src/api && go build ./... && go vet ./... && go test -v ./...

# Vue Frontend
cd src/web && npm run build && npx vue-tsc --noEmit

# Python Agent
cd src/agent && ruff check app/ tests/ && pytest tests/ -q

# Full build
task build    # Builds API + web
task test     # Runs Go tests
```

---

## 10. Appendices

### 10.1 Project Structure

```
AncientCoins/
├── .github/
│   ├── workflows/
│   │   ├── ci.yml                    # CI pipeline
│   │   ├── docker-publish.yml        # Production image push
│   │   └── docker-publish-beta.yml   # Beta image push
│   └── dependabot.yml                # Dependency updates
├── docs/
│   ├── ARCHITECTURE.md               # Architecture reference
│   ├── SDD.md                        # This document
│   ├── api-reference.md              # Full API reference
│   ├── authentication.md             # Auth design
│   ├── deployment.md                 # Deployment guide
│   ├── features.md                   # Feature catalog
│   ├── getting-started.md            # First-launch guide
│   ├── pwa-guide.md                  # PWA usage guide
│   ├── security-analysis.md          # Security review
│   └── social-feature.md             # Social feature design
├── src/
│   ├── api/                          # Go API
│   │   ├── main.go                   # Bootstrap + DI + routes
│   │   ├── architecture_test.go      # Architecture enforcement
│   │   ├── config/config.go          # Environment config
│   │   ├── database/database.go      # GORM + SQLite setup
│   │   ├── handlers/                 # HTTP handlers
│   │   ├── middleware/               # Auth + rate limiting
│   │   ├── models/                   # GORM models
│   │   ├── repository/              # Data access layer
│   │   ├── services/                # Business logic
│   │   └── docs/                    # Generated Swagger
│   ├── web/                          # Vue 3 Frontend
│   │   ├── src/
│   │   │   ├── api/client.ts        # API client
│   │   │   ├── router/index.ts      # Router + auth guard
│   │   │   ├── stores/              # Pinia stores
│   │   │   ├── pages/               # Route-level views
│   │   │   ├── components/          # Reusable components
│   │   │   ├── composables/         # Vue composables
│   │   │   ├── types/index.ts       # TypeScript types
│   │   │   └── assets/styles/       # CSS variables + base
│   │   └── vite.config.ts           # Vite + PWA config
│   └── agent/                        # Python Agent Service
│       ├── app/
│       │   ├── main.py              # FastAPI app
│       │   ├── routes.py            # API endpoints
│       │   ├── supervisor.py        # Intent router
│       │   ├── streaming.py         # SSE helpers
│       │   ├── config.py            # Environment config
│       │   ├── logging_config.py    # Ring-buffer logger
│       │   ├── llm/provider.py      # LLM factory
│       │   ├── models/              # Pydantic schemas
│       │   ├── teams/               # LangGraph team graphs
│       │   └── tools/search.py      # Search + fetch tools
│       └── tests/                   # pytest tests
├── Dockerfile                        # Multi-stage app build
├── docker-compose.yaml               # Two-container orchestration
├── Taskfile.yml                      # Development task runner
├── CONTRIBUTING.md                   # Contribution guidelines
├── README.md                         # Project overview
└── LICENSE                           # MIT License
```

### 10.2 AI Provider Configuration

Users select one AI provider in Admin Settings:

| Provider | Models | Web Search | Requirements |
|---|---|---|---|
| **Anthropic** (Recommended) | Claude family | Built-in `web_search_20250305` tool | API key |
| **Ollama** | Self-hosted (e.g., llava, llama3.1) | External SearXNG instance | Ollama server + SearXNG |

The `AIProvider` setting must be explicitly configured before agent features work. The agent chat displays a configuration banner when unconfigured.

### 10.3 Commit Convention

Conventional prefixes: `feat:`, `fix:`, `docs:`, `refactor:`, `chore:`

All commits include the co-author trailer:
```
Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>
```

### 10.4 Related Documentation

| Document | Path | Description |
|---|---|---|
| Architecture Reference | `docs/ARCHITECTURE.md` | Detailed architecture rules and patterns |
| API Reference | `docs/api-reference.md` | Full endpoint documentation with examples |
| Authentication Design | `docs/authentication.md` | JWT, WebAuthn, API keys, security checklist |
| Deployment Guide | `docs/deployment.md` | Production setup, Docker Compose, backups |
| Feature Catalog | `docs/features.md` | Complete feature inventory |
| Getting Started | `docs/getting-started.md` | First-launch walkthrough |
| PWA Guide | `docs/pwa-guide.md` | PWA installation and mobile UX |
| Security Analysis | `docs/security-analysis.md` | Security review and findings |
| Social Feature Design | `docs/social-feature.md` | Social feature specification |
