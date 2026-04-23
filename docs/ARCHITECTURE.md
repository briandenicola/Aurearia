# Architecture

This document describes the layered architecture of the Ancient Coins API (`src/api/`). All contributors (human and AI) should follow these conventions.

## Layer Diagram

```
Request
  |
  v
[Handler]  ----  Parses HTTP request, calls service/repo, returns HTTP response
  |
  v
[Service]  ----  Business logic, orchestration, domain rules (HTTP-agnostic)
  |
  v
[Repository]  --  Database queries, GORM operations (one repo per domain)
  |
  v
[Database]  ----  *gorm.DB (initialized once in main.go, injected everywhere)
```

## Rules

### 1. Only `main.go` references the `database` package

Every other package receives `*gorm.DB` (or a repository/service) through its constructor. This is enforced by `architecture_test.go`.

```go
// GOOD (main.go)
coinRepo := repository.NewCoinRepository(database.DB)

// BAD (anywhere else)
database.DB.Find(&coins)  // violates DI rule
```

### 2. Handlers are thin

Handlers should only:
- Parse the request (bind JSON, extract path/query params)
- Call a service or repository method
- Map the result to an HTTP response

Handlers must NOT contain business logic, multi-step orchestration, or raw SQL.

### 3. Services contain business logic

Services orchestrate multiple repository calls, enforce domain rules, and perform data transformations. They are HTTP-agnostic (no `gin.Context`, no `net/http`).

```go
// Service method
func (s *CoinService) UpdateCoin(existing *models.Coin, updates *models.Coin, ...) error {
    s.repo.Update(existing, updates)
    if valueChanged { s.repo.RecordValueHistory(...) }
    s.repo.RecordValueSnapshot(userID)
}
```

### 4. Repositories own all database access

Every GORM query lives in a repository file. Repositories take `*gorm.DB` via constructor and expose typed methods.

```go
func (r *CoinRepository) FindByID(id, userID uint) (*models.Coin, error) {
    var coin models.Coin
    err := r.db.Scopes(OwnedByID(id, userID)).Preload("Images").First(&coin).Error
    return &coin, err
}
```

### 5. Multi-step writes use transactions

When a method performs 2+ database writes that must succeed or fail together, wrap them in `r.db.Transaction()`. Key examples:
- `CoinRepository.Delete` (coin + images + journals + value history)
- `AdminRepository.DeleteUserCascade` (all user data)
- `AuthRepository.RotateRefreshToken` (revoke old + create new)
- `ImageRepository.SetPrimaryAndCreate` (clear primary + insert)

### 6. Errors stay internal

Never expose raw `err.Error()` from internal packages (GORM, Ollama, WebAuthn) to API clients. Log the real error server-side and return a generic message. Validation errors from `ShouldBindJSON` are acceptable to return.

## Package Map

| Package | Responsibility | Imports allowed |
|---|---|---|
| `main.go` | Composition root, wiring | Everything |
| `handlers/` | HTTP layer | `services/`, `repository/`, `models/` |
| `services/` | Business logic | `repository/`, `models/` |
| `repository/` | Database access | `models/`, `gorm.io/gorm` |
| `models/` | Data structures | Standard library only |
| `middleware/` | Auth, rate limiting | `models/`, `gorm.io/gorm` |
| `config/` | Environment config | Standard library only |
| `database/` | DB initialization | `gorm.io/gorm`, driver |

## Shared Scopes

Reusable GORM scopes live in `repository/scopes.go`:
- `OwnedBy(userID)` — filters by `user_id`
- `ByID(id)` — filters by primary key
- `OwnedByID(id, userID)` — filters by both
- `ActiveCollection(userID)` — non-wishlist, non-sold coins
- `PublicCoins(userID)` — public, non-wishlist, non-sold
- `ByCoinID(coinID)` — filters by `coin_id`

Use scopes instead of repeating `Where("user_id = ? AND ...")` queries.

## Running Architecture Tests

```bash
cd src/api
go test -v -run "TestNoDirectDatabase|TestHandlersDoNotUseRawSQL" .
```

## Multi-Agent Architecture (src/agent/)

The AI agent logic runs as a separate Python service using **LangGraph** for multi-agent orchestration. The Go API acts as a thin proxy — it contains zero LLM inference logic.

### Container Topology

```
Vue SPA ───> Go API (8080) ───> Python Agent (8081)
              │ proxy only        │
              │                   ├── Team 1: Coin Search
              │                   ├── Team 2: Coin Shows
              │                   ├── Team 3: Coin Analysis
              │                   ├── Team 4: Portfolio Review
              │                   ├── Team 5: Auction Search
              │                   ├── Team 6: Availability Check
              │                   ├── Team 7: Coin Grading
              │                   ├── Team 8: Gap Analysis
              │                   ├── Team 9: Photo Guide
              │                   ├── Team 10: Price Trends
              │                   └── Team 11: Similar Lots
              │                   │
              │              ┌────┴────┐
              │              v         v
              │           Claude    Ollama + SearXNG
              │           API       (ReAct agent)
              v
            SQLite
```

### Team Pipelines

Each team follows a multi-agent pipeline with verification steps to prevent hallucinated results:

| Team | Pipeline | Purpose |
|------|----------|---------|
| 1: Coin Search | Search → Fetch → Format | Find currently available coins |
| 2: Coin Shows | Search → Verify Dates → Format | Find upcoming numismatic events |
| 3: Coin Analysis | Analyze (vision) → Format | AI image analysis of coins |
| 4: Portfolio Review | Read → Valuate → Analyze | Collection analysis and recommendations |
| 5: Auction Search | Search → Fetch → Format | Search NumisBids for auction lots |
| 6: Availability Check | Check URLs → Analyze Results | Verify wishlist listings are still for sale |
| 7: Coin Grading | Analyze Photos → Grade → Format | AI grade estimation with confidence |
| 8: Gap Analysis | Read Portfolio → Analyze Gaps → Suggest | Collection completeness analysis |
| 9: Photo Guide | Analyze Photos → Evaluate → Tips | Photography improvement suggestions |
| 10: Price Trends | Search Auctions → Analyze Trends → Format | Market price trend analysis |
| 11: Similar Lots | Search → Rank → Format | Find similar active auction lots |

### Search Strategy

- **Anthropic/Claude**: Uses Claude's built-in `web_search` server-side tool
- **Ollama**: Uses a LangGraph `create_react_agent` with a SearXNG tool bound via `bind_tools` — the model decides when and how to search, mirroring how Anthropic's server-side tool works

### Data Flow

1. Go API reads settings from DB (API keys, model names, prompts)
2. Go enriches the request with settings + user context
3. Go POSTs to Python agent service
4. Python agent runs the multi-team LangGraph pipeline
5. Python streams SSE events back to Go
6. Go transparently proxies SSE to the Vue frontend

The Python service is **stateless** — it has no database access. All configuration is passed per-request from Go.

### Key Files

| File | Purpose |
|------|---------|
| `src/agent/app/supervisor.py` | Top-level router + team wiring |
| `src/agent/app/teams/coin_search.py` | Team 1: Search → Fetch → Format |
| `src/agent/app/teams/coin_shows.py` | Team 2: Shows → Date verify → Format |
| `src/agent/app/teams/coin_analysis.py` | Team 3: Vision analysis → Format |
| `src/agent/app/teams/portfolio_review.py` | Team 4: Read → Valuate → Analyze |
| `src/agent/app/teams/auction_search.py` | Team 5: Auction search → Fetch → Format |
| `src/agent/app/teams/availability_check.py` | Team 6: Check URLs → Analyze results |
| `src/agent/app/teams/coin_grading.py` | Team 7: Photo-based grade estimation |
| `src/agent/app/teams/gap_analysis.py` | Team 8: Collection gap analysis |
| `src/agent/app/teams/photo_guide.py` | Team 9: Photography tips |
| `src/agent/app/teams/price_trends.py` | Team 10: Price trend analysis |
| `src/agent/app/teams/similar_lots.py` | Team 11: Similar lot finder |
| `src/agent/app/tools/numisbids.py` | NumisBids scraping tools |
| `src/api/services/numisbids_service.go` | Go NumisBids HTTP client (login, watchlist, scraper) |
| `src/api/services/auction_lot_service.go` | Auction lot status transitions, convert-to-coin |
| `src/api/services/availability_service.go` | Wishlist URL checking with keyword heuristics + agent escalation |
| `src/api/services/availability_scheduler.go` | Background scheduler for periodic availability checks |
| `src/agent/app/streaming.py` | SSE streaming from LangGraph events |
| `src/agent/app/llm/provider.py` | LLM factory (Anthropic vs Ollama) |
| `src/api/services/agent_proxy.go` | Go SSE proxy to Python service |
| `src/api/handlers/showcase.go` | Showcase CRUD + public endpoint |
| `src/api/handlers/calendar.go` | Calendar events CRUD |
| `src/api/handlers/alerts.go` | Price alerts + bid reminders |
| `src/api/handlers/bulk.go` | Bulk coin operations |
| `src/api/handlers/tag.go` | Custom tags CRUD |
| `src/api/handlers/notification.go` | Notification management |
| `src/api/handlers/export_pdf.go` | PDF catalog generation |
| `src/api/repository/showcase_repository.go` | Showcase + coin assignments |
| `src/api/repository/price_alert_repository.go` | Alerts + reminders |
| `src/api/repository/auction_event_repository.go` | Calendar events |
| `src/api/repository/tag_repository.go` | Tags + coin-tag joins |
| `src/api/repository/notification_repository.go` | Notification persistence |
