# Ancient Coins

> **Note:** This application is 100% vibe coded. It's exclusively for me to learn and experiment with GitHub Copilot CLI.

Ancient Coins is a full-stack web application for cataloging and managing your personal coin collection. Track details like denomination, ruler, era, mint, material, grade, inscriptions, RIC rarity ratings, photos, and more — with AI-powered coin analysis via Ollama vision models and an Anthropic-powered coin search agent. New AI agent teams provide **grading assistance**, **collection gap analysis**, **coin photography tips**, **price trend analysis**, and **similar lot finding**. Share curated subsets of your collection via **public showcase links**, plan purchases with an **auction calendar**, stay informed with **price alerts** and **bid reminders** through an **in-app notification system**, organize with **custom tags and labels**, act on multiple coins at once with **bulk operations**, visualize your holdings on a **heat map**, and generate a **PDF catalog export**. Every coin is scoped to your authenticated account using JWT-based authentication.

It includes a **wish list** with an AI search agent for discovering coins and **automated availability checking** to detect when listings go off-market, **auction lot tracking** with NumisBids watchlist sync, a **stats dashboard** with grade distribution charts, portfolio value tracking over time, and a **heat map** of your collection by era and region, per-coin **activity journals**, **Numista catalog lookups**, collection **export/import**, **PDF catalog export** with photos and valuations, **collection showcases** with public shareable links, **notifications** with unread badges, **bulk operations** for batch status changes and tagging, **custom tags** with user-defined colors, and **social features** — follow other collectors, accept or block followers, browse follower galleries, leave comments and star ratings on coins, and discover users with search and public profiles.

On first launch, the first user to register is automatically assigned as the admin and can configure application settings including AI integrations.

## Architecture

| Layer    | Tech                                       | Path         |
| -------- | ------------------------------------------ | ------------ |
| Backend  | Go (Gin), GORM, Pure-Go SQLite             | `src/api/`   |
| Frontend | Vue 3, TypeScript, Vite, Pinia (PWA)       | `src/web/`   |
| Agent    | Python, FastAPI, LangGraph, LangChain      | `src/agent/` |

The Vue SPA communicates with the Go API exclusively via REST (`/api/*`). The Go API proxies all AI agent requests to a Python LangGraph service. In production, docker-compose runs two containers (Go+Vue and Python agent).

The frontend is a Progressive Web App (PWA) and can be installed on iOS (Safari → Share → Add to Home Screen), Android, and desktop browsers for a native app-like experience with offline caching.

### Development (two processes, two ports)

```
Browser → localhost:5173 (Vite dev server, serves Vue SPA)
           └─ /api/* requests → proxied to localhost:8080 (via vite.config.ts)

Browser → localhost:8080 (Go/Gin API, serves REST endpoints)
```

In development, the Vite dev server runs on `:5173` with hot-reload and proxies any `/api/*` or `/uploads/*` request to the Go API on `:8080`. The browser only talks to the Vite server.

### Production (two containers)

The app Dockerfile uses a multi-stage build to combine the Go API and Vue SPA into one image. A separate Python agent container runs the LangGraph AI service. Both are orchestrated via `docker-compose.yaml`:

```
Stage 1: node:24-alpine     → npm run build  → produces dist/ (static HTML/JS/CSS)
Stage 2: golang:1.26-alpine → go build       → produces ancient-coins-api binary
Stage 3: alpine:3.21        → copies both:
           /app/ancient-coins-api    (Go binary)
           /app/wwwroot/             (Vue dist/ output)
```

The Go binary serves the Vue SPA as static files **and** handles API routes — one process does both jobs. AI agent requests are proxied to the Python agent container:

```
Browser → localhost:8080 → Go binary (app container)
              ├─ /api/*      → Gin REST handlers
              ├─ /uploads/*  → serves uploaded images from volume
              └─ /*          → serves Vue SPA from /app/wwwroot/

Go API → localhost:8081 → Python agent (agent container)
              └─ AI agent requests (search, analysis, portfolio)
```

No nginx or reverse proxy needed. Docker volumes persist the SQLite database and uploaded images across container restarts. The agent container has a healthcheck and the app container depends on it being healthy.

## Prerequisites

- [Go](https://go.dev/dl/) (1.26+)
- [Node.js](https://nodejs.org/) (v20+)
- [Task](https://taskfile.dev/) — optional, for task runner commands
- [Docker & Docker Compose](https://docs.docker.com/get-docker/) — optional, for containerized deployment
- [Ollama](https://ollama.ai/) — optional, for AI coin analysis

## Getting Started

Clone the repository and start the development servers:

```sh
git clone <repo-url> && cd AncientCoins
task run        # starts both API and frontend in parallel
```

The API runs on `http://localhost:8080` and the Vite dev server on `http://localhost:5173`. You can also start them individually:

```sh
task run-api    # API only
task run-web    # frontend only
```

When the app launches for the first time, register your first account — it is automatically assigned as the admin. You can then configure Ollama and other settings from the Admin page.

For a detailed walkthrough of first-time setup, adding coins, import/export, and AI analysis, see the [Getting Started Guide](docs/getting-started.md).

## Task Commands

| Command            | Description                              |
| ------------------ | ---------------------------------------- |
| `task init`        | Generate a `.env` file with a random JWT secret |
| `task run`         | Run API and frontend in parallel         |
| `task run-api`     | Run the Go API server                    |
| `task run-web`     | Run the Vite dev server                  |
| `task build`       | Build both API and frontend              |
| `task build-api`   | Build the Go API binary                  |
| `task build-web`   | Build the Vue frontend                   |
| `task test`        | Run Go architecture and unit tests       |
| `task docker-build`| Build the Docker container image         |
| `task docker-run`  | Run the Docker container locally         |
| `task build-agent` | Build the Python agent service            |
| `task run-agent`   | Run the Python agent dev server           |
| `task test-agent`  | Run Python agent tests                    |
| `task lint-agent`  | Lint Python agent code                    |
| `task up-all`      | Run API, web, and agent servers in parallel |

## CI/CD

A GitHub Actions workflow builds and pushes the Docker images to Docker Hub on push to `main`. See the [Deployment Guide](docs/deployment.md) for required secrets and full configuration.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for development setup and coding guidelines, and [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) for the API layering rules.

## Features

For detailed descriptions of every feature, see the [Features Guide](docs/features.md).

- **Collection Management** — Card gallery with filtering, search, sorting, swipe/grid toggle, and category colors
- **Wish List** — Track desired coins with an AI-powered search agent that finds real listings
- **Auction Tracking** — Track NumisBids lots through watching/bidding/won/lost workflow with watchlist sync
- **Sold Coins** — Record sales with profit/loss tracking
- **Coin Details** — Full numismatic data, multiple images, AI analysis, activity journal, Numista lookup
- **AI Coin Analysis** — Ollama vision model identifies coins, assesses condition, and estimates value
- **AI Search Agent** — Anthropic Claude with web search finds coins matching your description
- **Collection Statistics** — Category/material/grade breakdowns, value-over-time charts, top coins
- **Social Features** — Follow collectors, accept/block followers, browse galleries, comment and rate coins
- **User Profiles** — Avatar, bio, public/private toggle, email
- **Authentication** — JWT + refresh tokens, WebAuthn/passkeys, API keys ([details](docs/authentication.md))
- **PWA** — Installable on iOS/Android/desktop with swipe gallery, camera capture, pull-to-refresh ([details](docs/pwa-guide.md))
- **Settings** — Appearance, data export/import, API keys, WebAuthn, saved conversations
- **Admin** — User management, AI configuration, system settings, live logs
- **AI Grading Assistant** — Upload photos, AI estimates grade with reasoning and confidence
- **Price Trend Analysis** — Track auction results over time, show market direction
- **Collection Gap Analysis** — AI reviews collection and suggests missing coins
- **Heat Map** — Visual distribution by era and region on the Stats page
- **Collection Showcase** — Public shareable links for curated collection subsets
- **Auction Calendar** — Monthly calendar with lot end dates and custom events
- **Price Alerts** — Set target prices on watched lots, get notified when bidding crosses threshold
- **Bid Reminders** — Notification before a watched lot's session closes
- **Notifications** — In-app notification system with unread badge
- **Bulk Operations** — Multi-select coins for batch status changes, tagging, or export
- **Custom Tags** — User-defined labels with colors for organizing coins
- **PDF Export** — Generate catalog with photos, grades, provenance, and valuations
- **Coin Photography Guide** — AI tips for better coin photos based on uploads
- **Similar Lot Finder** — Find active auction lots similar to your collection coins

## Deployment

The application ships as two Docker containers — one for the Go API + Vue SPA, and one for the Python LangGraph agent. See the [Deployment Guide](docs/deployment.md) for full instructions.

```sh
docker compose up        # quickest way to run
```

Generate a `.env` with a random JWT secret:

```sh
task init
```

For environment variables and admin-managed settings, see [Configuration](docs/features.md#configuration).

## Project Structure

```
AncientCoins/
├── .devcontainer/                    # Dev container configuration
│   ├── Dockerfile
│   ├── devcontainer.json
│   └── post-create.sh
├── src/
│   ├── api/                          # Go backend
│   │   ├── main.go                   # App entry point & route wiring (composition root)
│   │   ├── config/                   # Environment-based configuration
│   │   ├── database/                 # SQLite connection (pure-Go driver)
│   │   ├── handlers/                 # Thin HTTP handlers (parse request → call service → return response)
│   │   │   ├── auth.go               # Registration, login, token refresh
│   │   │   ├── coins.go              # Coin CRUD, list/filter/sort, stats, sell, value history
│   │   │   ├── images.go             # Image upload/delete, proxy, scrape
│   │   │   ├── analysis.go           # AI coin analysis (proxied to agent service)
│   │   │   ├── agent.go              # AI agent chat (proxied to agent service via SSE)
│   │   │   ├── conversations.go      # Saved agent conversation CRUD
│   │   │   ├── journal.go            # Per-coin activity log
│   │   │   ├── numista.go            # Numista catalog search proxy
│   │   │   ├── admin.go              # User/settings management
│   │   │   ├── user.go               # Password change, profile
│   │   │   ├── auction_lots.go       # Auction lot CRUD, NumisBids sync, convert-to-coin
│   │   │   ├── alerts.go              # Price alerts & bid reminders
│   │   │   ├── bulk.go                # Bulk coin operations
│   │   │   ├── calendar.go            # Auction calendar events
│   │   │   ├── export_pdf.go          # PDF catalog generation
│   │   │   ├── notification.go        # Notification CRUD
│   │   │   ├── showcase.go            # Collection showcase CRUD + public view
│   │   │   ├── tag.go                 # Custom tags CRUD
│   │   │   ├── api_keys.go           # API key management
│   │   │   └── webauthn.go           # FIDO2/WebAuthn auth
│   │   ├── repository/               # Database access layer (all GORM queries)
│   │   │   ├── scopes.go             # Reusable GORM scopes (OwnedBy, ActiveCollection, etc.)
│   │   │   ├── coin_repository.go    # Coin CRUD, stats, value history, snapshots
│   │   │   ├── social_repository.go  # Follow, comments, ratings, user search
│   │   │   ├── auth_repository.go    # User and refresh token management
│   │   │   ├── image_repository.go   # Image records and primary flag management
│   │   │   ├── admin_repository.go   # Admin user management, cascade delete
│   │   │   ├── agent_repository.go   # Portfolio summary, value estimation
│   │   │   ├── auction_lot_repository.go # Auction lot CRUD, upsert, status
│   │   │   ├── auction_event_repository.go # Calendar events
│   │   │   ├── notification_repository.go  # Notifications
│   │   │   ├── price_alert_repository.go   # Price alerts & bid reminders
│   │   │   ├── showcase_repository.go      # Showcase + coin assignments
│   │   │   ├── tag_repository.go           # Tag CRUD & coin-tag joins
│   │   │   └── ...                   # journal, conversation, webauthn, api_key, analysis
│   │   ├── services/                 # Business logic (HTTP-agnostic)
│   │   │   ├── coin_service.go       # Value tracking, snapshot orchestration
│   │   │   ├── social_service.go     # Follow rules, access control, profiles
│   │   │   ├── auth_service.go       # Registration, authentication, token lifecycle
│   │   │   ├── image_service.go      # File upload/delete coordination
│   │   │   ├── agent_proxy.go        # SSE proxy to Python agent service
│   │   │   ├── ollama_service.go     # Ollama vision model integration (OCR)
│   │   │   ├── settings_service.go   # App settings with DB-backed defaults
│   │   │   ├── numisbids_service.go  # NumisBids HTTP client (login, watchlist, scraper)
│   │   │   ├── auction_lot_service.go # Auction lot status transitions, convert-to-coin
│   │   │   └── logger.go             # Structured logger with in-memory buffer
│   │   ├── middleware/               # JWT & API key auth middleware
│   │   ├── models/                   # GORM entities (Coin, User, Follow, etc.)
│   │   │   ├── notification.go        # Notification events
│   │   │   ├── price_alert.go         # PriceAlert & BidReminder
│   │   │   ├── showcase.go            # Showcase, ShowcaseCoin, AuctionEvent
│   │   │   └── tag.go                 # Tag & CoinTag
│   │   └── architecture_test.go      # Enforces layering rules (no database.DB in handlers)
│   ├── agent/                         # Python LangGraph agent service
│   │   ├── app/
│   │   │   ├── main.py               # FastAPI app entry point
│   │   │   ├── config.py             # Service settings
│   │   │   ├── routes.py             # API endpoints (search, analyze, portfolio)
│   │   │   ├── supervisor.py         # Top-level router + team delegation
│   │   │   ├── streaming.py          # SSE streaming from LangGraph events
│   │   │   ├── llm/provider.py       # LLM factory (Anthropic vs Ollama)
│   │   │   ├── tools/search.py       # SearXNG search + URL verification tools
│   │   │   ├── tools/numisbids.py   # NumisBids scraping tools (lot, watchlist, search)
│   │   │   ├── models/               # Pydantic request/response schemas
│   │   │   └── teams/                # Multi-agent team pipelines
│   │   │       ├── coin_search.py    # Team 1: Search → Fetch → Format
│   │   │       ├── coin_shows.py     # Team 2: Shows → Date verify → Format
│   │   │       ├── coin_analysis.py  # Team 3: Vision analysis → Format
│   │   │       ├── portfolio_review.py # Team 4: Read → Valuate → Analyze
│   │   │       ├── auction_search.py # Team 5: Auction search → Fetch → Format
│   │   │       ├── coin_grading.py    # Team 7: Grade estimation from photos
│   │   │       ├── gap_analysis.py    # Team 8: Collection completeness analysis
│   │   │       ├── photo_guide.py     # Team 9: Photography improvement tips
│   │   │       ├── price_trends.py    # Team 10: Market price trend analysis
│   │   │       └── similar_lots.py    # Team 11: Find similar active auction lots
│   │   ├── tests/                    # Pytest tests
│   │   ├── Dockerfile                # Python 3.12-slim multi-stage
│   │   └── pyproject.toml            # Dependencies (FastAPI, LangGraph, LangChain)
│   └── web/                          # Vue 3 SPA
│       ├── src/
│       │   ├── api/                  # Axios API client
│       │   ├── assets/styles/        # CSS variables & global styles
│       │   ├── components/           # Reusable components
│       │   │   ├── CoinCard.vue      # Gallery card (collection + wishlist + sold variants)
│       │   │   ├── CoinForm.vue      # Shared create/edit form with autocomplete & camera
│       │   │   ├── CoinSearchChat.vue # AI agent chat drawer with streaming
│       │   │   ├── SellModal.vue     # Sell coin dialog with price & buyer fields
│       │   │   ├── SearchBar.vue     # Search input
│       │   │   ├── CategoryFilter.vue # Category pill filters
│       │   │   ├── SortSelect.vue    # Sort dropdown
│       │   │   ├── ImageGallery.vue  # Image grid with lightbox
│       │   │   ├── SwipeGallery.vue  # Mobile swipe carousel
│       │   │   ├── AuctionLotCard.vue # Auction lot card with status badges
│       │   │   ├── ImportLotModal.vue # Add lot from NumisBids URL
│       │   │   ├── ImageProcessor.vue # Store card OCR upload
│       │   │   └── AutocompleteInput.vue
│       │   ├── pages/                # Route pages
│       │   │   ├── CalendarPage.vue       # Auction calendar
│       │   │   ├── NotificationsPage.vue  # Notification inbox
│       │   │   ├── PublicShowcasePage.vue  # Public showcase viewer
│       │   │   ├── ShowcasesPage.vue      # Showcase management
│       │   │   └── ShowcaseEditPage.vue   # Showcase coin picker
│       │   ├── stores/               # Pinia stores (auth, coins)
│       │   ├── composables/           # Shared reactive state
│       │   │   ├── useBulkSelect.ts   # Bulk select mode toggle
│       │   │   ├── useDialog.ts       # Confirm/alert dialogs
│       │   │   ├── useNotifications.ts # Unread badge polling
│       │   │   └── usePullToRefresh.ts # Touch pull-to-refresh
│       │   ├── router/               # Vue Router configuration
│       │   └── types/                # TypeScript type definitions
│       ├── public/                   # PWA icons & coin logo
│       └── vite.config.ts
├── docs/
│   ├── ARCHITECTURE.md              # API layering rules and package map
│   ├── features.md                  # Detailed feature documentation
│   ├── getting-started.md           # User walkthrough guide
│   ├── authentication.md            # JWT, refresh tokens, WebAuthn, API keys
│   ├── api-reference.md             # Complete REST API reference
│   ├── deployment.md                # Production deployment guide
│   ├── pwa-guide.md                 # PWA features & installation
│   ├── social-feature.md            # Social features spec & implementation details
│   ├── security-analysis.md         # Security analysis report
│   └── SDD.md                       # Software design document
├── instructions.md                   # Agent instructions for AI coding assistants
├── Dockerfile                        # Multi-stage build (Vue + Go → Alpine)
├── Taskfile.yml                      # Task runner configuration
├── docker-compose.yaml               # Container orchestration
└── README.md
```

## Backlog

Feature ideas and completed enhancements:

- [x] **CI/CD Pipeline** — GitHub Actions workflow to build and push Docker image
- [x] **Sorting** — Sort coins by date added, date updated, or value
- [x] **Swipe / Grid Toggle** — Mobile-friendly view preference with PWA support
- [x] **PWA Viewport Stability** — Fixed scrolling/interaction wobble in installed PWA
- [x] **Grade Distribution Chart** — Bar chart of coins by grade
- [x] **Value Over Time Chart** — SVG line chart tracking portfolio value and investment
- [x] **Activity Journal** — Per-coin timestamped activity log
- [x] **Numista Catalog Lookup** — Search the Numista coin database from detail pages
- [x] **AI Coin Search Agent** — Anthropic-powered chat agent with web search for discovering coins
- [x] **Streaming Agent Responses** — Real-time SSE streaming for AI search results
- [x] **Saved Conversations** — Save and reopen AI search agent conversations
- [x] **Sold Coins** — Track sold coins with price, buyer, and profit/loss display
- [x] **Camera Capture** — Take coin photos directly in PWA mode with rear camera
- [x] **Image Scraping** — Automatic og:image extraction for wishlist coin images
- [x] **Paste Image URL** — Fetch and attach coin images from external URLs
- [x] **Tabbed Settings** — Reorganized user settings into Account, Appearance, Data, Conversations tabs
- [x] **Build Version Display** — Version and build date injected at build time and shown in Admin settings
- [x] **Refresh Tokens** — 30-day rolling refresh tokens with silent frontend renewal
- [x] **WebAuthn / Passkeys** — Face ID, Touch ID, and fingerprint login via FIDO2
- [x] **API Keys** — Per-user API keys for programmatic access with `X-API-Key` header
- [x] **Pull-to-Refresh** — Swipe-down refresh gesture in PWA mode
- [x] **Background Removal** — Client-side ML-powered image background removal on detail page
- [x] **Base64 Image Upload** — Upload images as base64-encoded data via API
- [x] **Gallery Image Side Toggle** — Switch between primary, obverse, and reverse images in grid view
- [x] **PWA Hamburger Menu** — Compact popover menu for gallery controls in PWA mode
- [x] **Default Sort Setting** — Configurable default sort order in user settings
- [x] **Swipe Position Persistence** — Returning from detail view preserves swipe gallery position
- [x] **Social Features** — Follow other collectors, accept/block followers, view follower galleries
- [x] **Comments & Star Ratings** — Comment on and rate (1–5) coins belonging to followed users
- [x] **User Profiles** — Avatar upload, bio, public/private toggle
- [x] **User Search** — Discover other collectors by username
- [x] **Email Registration** — Required email for new users with legacy user prompt
- [x] **Collection Timeline** — Visual timeline of when each coin was acquired
- [x] **Auction Lot Tracking** — NumisBids integration with watchlist sync, status workflow, and auto-convert to collection
- [ ] **Coin Comparison** — Side-by-side spec comparison of any two coins
- [ ] **Advanced Search** — Filter by date range, price range, grade, material
- [x] **Price Alerts** — Set target prices on watched lots, get notified when bidding crosses threshold
- [x] **AI Grading Assistant** — Upload photos, AI estimates grade with confidence
- [x] **Price Trend Analysis** — Track auction results, show market direction
- [x] **Collection Gap Analysis** — AI reviews collection, suggests missing coins
- [x] **Heat Map** — Visual distribution by era/region on Stats page
- [x] **Collection Showcase** — Public shareable links with curated coin subsets
- [x] **Auction Calendar** — Monthly calendar with lot dates and custom events
- [x] **Bid Reminders** — Notification before watched lot session closes
- [x] **Notifications** — In-app notification system with badge count
- [x] **Bulk Operations** — Multi-select for batch status changes, tagging, export
- [x] **Custom Tags** — User-defined labels with colors
- [x] **PDF Catalog Export** — Generate PDF with photos, grades, provenance, valuations
- [x] **Coin Photography Guide** — AI tips for better coin photos
- [x] **Similar Lot Finder** — Find active lots similar to collection coins
- [x] **Share Collection** — Follow collectors and browse their public galleries with comments and ratings
- [x] **Repository + Service Layer** — Layered architecture with DI, transactions, and architecture tests

## License

This project is licensed under the [MIT License](LICENSE).
