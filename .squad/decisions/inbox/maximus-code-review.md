### 2026-04-24: Architecture & Code Quality Review
**By:** Maximus (Lead/Architect)

## Grades

| Area | Grade | Notes |
|------|-------|-------|
| Overall Architecture | **B+** | Clean 3-service separation, well-documented. The layered Go API, stateless Python agent, and Vue SPA topology is sound. Loses points for globals leaking into the DI model. |
| Layered Architecture Compliance | **B+** | Handler → Service → Repository → Database flow is enforced by `architecture_test.go`. Only `main.go` imports `database`. However, `social.go` and `agent.go` handlers contain business logic that belongs in services. |
| Dependency Injection | **B-** | Constructor injection is the norm, but package-level globals (`services.AppLogger`, `services.GetSetting()`, `cancelMap`) bypass DI and make testing harder. Not pure DI — it's DI-with-escape-hatches. |
| Error Handling | **C+** | Generic client-facing errors in most handlers, but `social.go` silently drops errors in 7+ locations. Broad `except Exception` in Python routes. No consistent error wrapping convention across Go services. Mixed discipline. |
| Auth/Security | **B+** | JWT + refresh token rotation + API keys + WebAuthn is comprehensive. Middleware is clean. Minor concern: API key lookup in middleware bypasses repository abstraction and hits DB directly. |
| API Design | **B** | RESTful, Swagger-annotated on major endpoints. Consistent naming. Loses points because Python agent streaming endpoints have no response model contracts — the SSE schema is implicit. |
| Configuration Management | **B+** | Env-based config with sensible defaults. JWT secret validation for production. Settings service for runtime config. Clean and simple. |
| Service Integration (Go ↔ Python) | **B+** | `agent_proxy.go` is well-factored: separate stream/request clients, context-aware, proper SSE headers. `collectSSE` only extracts `"done"` events which is brittle if the upstream contract evolves. |
| Frontend Architecture | **B-** | Good fundamentals (Composition API, Pinia, centralized API client, SSE streaming via fetch). Dragged down by 1200-1400 line god-pages (Admin, Settings, CoinDetail), excessive inline styles, and a coins store trending toward god-store. |
| Python Agent Architecture | **B** | Clean supervisor + StateGraph pipelines. Pydantic models for core schemas. Good LLM provider abstraction. Loses points for weak schema enforcement on LLM outputs, missing tests for team pipelines and supervisor routing, and broad exception handling. |
| Documentation | **A-** | `ARCHITECTURE.md` is excellent at 761 lines covering all 3 services, data flows, schema, auth, and design decisions. Swagger annotations on Go API. Minor gap: no documented SSE event contract between Go and Python. |

## Overall Assessment

This is a well-architected system with strong foundations. The three-service topology (Go API + Vue SPA + Python agent) is clean, with the Go API correctly acting as the sole gateway to the agent service. The layered architecture in Go is enforced by automated tests, models are properly isolated, and the DI wiring in `main.go` follows the documented pattern. Documentation is unusually thorough — the 761-line ARCHITECTURE.md is a genuine asset.

The main structural weakness is incomplete discipline. DI exists but is undermined by package globals (`AppLogger`, `GetSetting`, `cancelMap`). Error handling is inconsistent — some handlers are careful, others silently drop errors. The frontend has grown organically, with several pages exceeding 1200 lines and mixing UI, orchestration, and API logic. The Python agent has good structure but lacks test coverage on the most important paths (supervisor routing, team pipeline execution).

None of this is broken. The system works, ships, and is maintainable by a small team. But the escape hatches and inconsistencies will compound over time. The backlog below prioritizes closing these gaps before they become load-bearing technical debt.

## Backlog Items (Architecture & Cross-Cutting)

| # | Title | Priority | Area | Effort | Description |
|---|-------|----------|------|--------|-------------|
| 1 | Eliminate `services.GetSetting()` global | P1 | architecture | M | `GetSetting()` is a package-level function backed by a global DB reference initialized in `main.go`. Services like `valuation_service.go` call it directly, bypassing constructor injection. Refactor: inject a `SettingsService` interface into services that need runtime settings. Remove `InitSettings()` global init. |
| 2 | Inject logger instead of using `services.AppLogger` | P1 | architecture | M | `AppLogger` is a package-level global used across handlers and services. Replace with a logger interface injected via constructors. This enables test-time log capture and removes a hidden dependency. |
| 3 | Remove `cancelMap` global in valuation service | P1 | architecture | S | `valuation_service.go:25-27` has a package-level `cancelMap` for scheduler cancellation. Move into the `ValuationService` struct as instance state, initialized in constructor. |
| 4 | Fix silent error drops in `social.go` | P1 | backend | M | `handlers/social.go` ignores errors with `_` in 7+ locations (lines 143, 151, 185, 217, 337, 424, 461). These are database reads/writes that silently fail. Log errors and return appropriate HTTP responses. |
| 5 | Extract business logic from `social.go` handler | P2 | backend | M | `social.go` contains substantial presentation and business logic (lines 153-205, 225-244, 281-309, 326-366). Extract into a `SocialService` following the existing handler → service pattern. |
| 6 | Extract business logic from `agent.go` handler | P2 | backend | M | `handlers/agent.go` (lines 124-135, 169-201, 461-532) contains request building, prompt construction, and response processing logic. Move to a service layer so handlers stay thin. |
| 7 | Add transaction to auction lot upsert | P2 | backend | S | `auction_lot_repository.go:141-166` performs a read-modify-write upsert without a transaction. Under concurrent access this can race. Wrap in `r.db.Transaction()`. |
| 8 | Decompose god-pages (Admin, Settings, CoinDetail) | P2 | frontend | L | `AdminPage.vue` (1378 lines), `SettingsPage.vue` (1371 lines), and `CoinDetailPage.vue` (1242 lines) are too large. Extract logical sections into child components (e.g., `AdminUsersPanel`, `SettingsAppearance`, `CoinDetailHistory`). |
| 9 | Fix `sanitizeCoin()` truthiness bug | P2 | frontend | S | `api/client.ts:111-113`: `if (!clean.currentValue && clean.purchasePrice)` treats `0` as falsy, so a coin with `currentValue = 0` would incorrectly get its value overwritten by `purchasePrice`. Use explicit `null`/`undefined` check. |
| 10 | Define SSE event contract between Go and Python | P2 | docs | S | The SSE streaming protocol between `agent_proxy.go` and the Python agent is implicit. `collectSSE` only looks for `"done"` events. Document the event types, data schemas, and termination protocol in ARCHITECTURE.md or a dedicated ADR. |
| 11 | Add supervisor and team pipeline tests (Python) | P2 | agent | M | No tests exist for supervisor routing logic, recursion limit enforcement, or team graph wiring. Add unit tests that mock LLM calls and verify: correct team routing, iteration limits, and schema conformance of outputs. |
| 12 | Replace broad `except Exception` in Python routes | P2 | agent | S | Routes like `routes.py:108-126` and `167-204` catch all exceptions broadly. Use specific exception types where possible, and ensure `HTTPException` is raised (not raw dict returns like `/log-level`). |
| 13 | Move API key lookup from middleware to repository | P3 | architecture | S | `middleware/auth.go:79-100` queries the DB directly for API key validation. Route through the auth repository to maintain the repository abstraction consistently. |
| 14 | Reduce inline styles in Vue pages | P3 | frontend | M | `SettingsPage.vue`, `StatsPage.vue`, `CollectionPage.vue`, and `CoinForm.vue` use excessive `:style` bindings. Extract into CSS utility classes or component-scoped styles using the project's CSS variables. |
| 15 | Split coins store or scope more tightly | P3 | frontend | S | `stores/coins.ts` manages collection, current coin, stats, value history, category, search, and gallery state. Consider splitting into `useCollectionStore` and `useCoinDetailStore` before it becomes a god-store. |
| 16 | Centralize all fetch calls through API client | P3 | frontend | S | `CoinDetailPage.vue:524` and `useImageProcessor.ts:102` make raw `fetch()` calls outside the centralized API client. Route through `api/client.ts` for consistent auth header injection and error handling. |
| 17 | Add response models to Python streaming endpoints | P3 | agent | M | `/api/search/coins`, `/api/search/shows`, `/api/portfolio/review` lack `response_model` annotations. Even for SSE, document the expected event payload schemas using Pydantic models. |
| 18 | Enforce schema parsing on LLM team outputs | P3 | agent | M | Only availability verdicts are strictly parsed into Pydantic models post-LLM. Other teams (coin search, coin shows) rely on prompt instructions alone. Add structured output parsing or validation before returning results. |
| 19 | Separate operational endpoints from agent API | P3 | agent | S | `/logs` and `/log-level` are mixed into the main FastAPI app alongside agent endpoints. Move to a separate admin router with proper request/response models and authentication. |
| 20 | Remove `any` casts in `sanitizeCoin()` | P3 | frontend | S | `api/client.ts:106-107` casts to `any`. Define a proper type for the sanitization input (e.g., `Partial<Coin>`) to maintain type safety. |
