### 2026-04-24: Test Coverage & Quality Review
**By:** Brutus (Tester/QA)

---

## Test Run Results

### Go API (`src/api/`)
```
=== All 18 tests PASS ===

architecture_test.go (root):
  PASS  TestNoDirectDatabaseImports
  PASS  TestHandlersDoNotUseRawSQL

repository/coin_repository_test.go:
  PASS  TestCoinRepository_CreateAndGet
  PASS  TestCoinRepository_FindByID_WrongUser
  PASS  TestCoinRepository_WithTx
  PASS  TestCoinRepository_Delete
  PASS  TestCoinRepository_CoinExists
  PASS  TestCoinRepository_Scopes_OwnedBy
  PASS  TestCoinRepository_Scopes_ActiveCollection
  PASS  TestCoinRepository_Scopes_PublicCoins
  PASS  TestCoinRepository_RecordValueSnapshot

services/coin_service_test.go:
  PASS  TestCreateCoin_Success
  PASS  TestUpdateCoin_RecordsValueHistory
  PASS  TestUpdateCoin_EstimateSkipsHistory
  PASS  TestDeleteCoin_RemovesCoinAndImages
  PASS  TestDeleteCoin_WrongUser_NoEffect
  PASS  TestPurchaseCoin_UpdatesFields
  PASS  TestSellCoin_UpdatesFields

go vet: CLEAN
Coverage: repository 4.6%, services 3.5% (of total statements)
```

### Python Agent (`src/agent/`)
```
31 passed in 1.00s
ruff: All checks passed

test_api.py (6 tests): health endpoint + request validation for 4 routes
test_availability.py (6 tests): availability endpoint validation + parse_verdicts
test_models.py (7 tests): Pydantic model defaults and validators
test_retry.py (6 tests): ainvoke_with_retry behavior
test_streaming.py (6 tests): extract_suggestions + remove_json_block
```

### Vue Frontend (`src/web/`)
```
vue-tsc --noEmit: CLEAN (0 errors)
npm run build: Not attempted (type-check is sufficient for review)

Test files: ZERO
Test framework: NOT CONFIGURED
Test dependencies: NONE installed
```

---

## Grades

| Area | Grade | Notes |
|------|-------|-------|
| Go Unit Test Coverage | **D** | Only CoinRepository and CoinService tested. 20+ repositories, 15+ services, 0 handlers have no tests. Measured coverage: 3.5-4.6%. |
| Go Architecture Tests | **B-** | Enforces no-database-import and no-raw-SQL-in-handlers. Missing: full import matrix, handler thinness, service-only business logic, transaction enforcement. |
| Go Integration Tests | **F** | Zero HTTP-level handler tests. No Gin test router tests. No end-to-end request flow tests. |
| Frontend Tests | **F** | Zero test files. No framework. 60+ components, 2 stores, 10 composables, 1 API client — all untested. |
| Python Agent Tests | **C+** | 31 tests covering models, retry, streaming, and request validation. But zero tests for any team pipeline, LLM provider, search tools, or SSE streaming flow. |
| Edge Case Coverage | **D+** | Go: wrong-user and estimate-skip tested for coins. Python: invalid JSON, empty arrays tested. But no auth failure, rate-limit, concurrent access, empty DB, or malformed input edge cases across the board. |
| Test Quality (behavioral vs implementation) | **B** | Existing tests are well-structured — they test behavior (create coin → verify fields, wrong user → not found). No mocking of internals. Good use of real SQLite for Go repo tests. |
| Overall Test Strategy | **D** | No test plan. No coverage thresholds. No CI enforcement. Massive gaps in all three services. The few tests that exist are good quality, but coverage is abysmal. |

---

## Coverage Gaps

### Go API — Untested Packages

**handlers/** (26 files, 0 tests)
Every handler is untested: auth, coins, images, admin, agent, tags, journal, bulk, social, showcase, analysis, availability, valuation_admin, calendar, notifications, api_keys, webauthn, conversations, auction_lots, numista, export, export_pdf, alerts, user.

**services/** (16 files, 1 has tests)
Untested: AuthService, ImageService, NumisBidsService (+ ParseSaleDate), AvailabilityService, AuctionLotService, ValuationScheduler, SocialService, Logger/SyncLogLevel, ValuationParser (ParseValueEstimate), OllamaService, SettingsService (InitSettings/GetSetting/SetSetting/GetAllSettings), AvailabilityScheduler, ValuationService (ResolveLLMConfig/BuildCoinDescription/ValuateCollectionForUser), NotificationService, AgentProxy.

**repository/** (21 files, 1 has tests)
Untested: TagRepository, AuctionLotRepository, AuthRepository, JournalRepository, WebAuthnRepository, AuctionEventRepository, UserRepository, SocialRepository, ApiKeyRepository, AvailabilityRepository, NotificationRepository, ValuationRepository, AnalysisRepository, AdminRepository, PriceAlertRepository, BidReminderRepository, ConversationRepository, ShowcaseRepository, ImageRepository, AgentRepository.

**middleware/** (2 files, 0 tests)
Untested: AuthRequired, RateLimit.

**config/** (1 file, 0 tests)
Untested: Load.

**models/** (19 files, 0 tests)
No validation logic tests (though models may be simple structs).

### Python Agent — Untested Modules

- **All 11 team pipelines**: coin_search, coin_shows, coin_analysis, portfolio_review, availability_check (graph only), coin_grading, price_trends, similar_lots, photo_guide, gap_analysis, auction_search
- **LLM provider**: get_chat_model, get_search_model, create_search_agent
- **Tools**: create_searxng_search, all numisbids parsers
- **Streaming**: stream_graph_events, format_sse
- **Routes**: success paths (all routes only have validation tests)
- **Config**: Settings class
- **Logging**: RingBufferHandler, setup_logging, set_log_level

### Vue Frontend — Everything

60+ components, 2 Pinia stores, 10 composables, API client — zero tests for anything.

---

## Backlog Items (Testing)

| # | Title | Priority | Effort | Description |
|---|-------|----------|--------|-------------|
| 1 | Add handler integration tests for AuthHandler (login, register, refresh, logout) | P0 | L | Create `handlers/auth_handler_test.go` with httptest + Gin test router. Test login success/failure, registration validation, token refresh, and logout. Auth is the most security-critical path. |
| 2 | Add handler integration tests for CoinHandler CRUD | P0 | L | Create `handlers/coin_handler_test.go`. Test GET/POST/PUT/DELETE with authenticated requests. Verify response codes, JSON bodies, and ownership enforcement at the HTTP layer. |
| 3 | Add unit tests for AuthService | P0 | M | Test Login (valid/invalid creds), Register (duplicate username), HashPassword, ValidateToken, RefreshToken. Auth bugs are security bugs. |
| 4 | Add unit tests for middleware/auth.go (AuthRequired) | P0 | M | Test JWT extraction, expired token handling, missing token, invalid token, and correct user ID injection into context. |
| 5 | Set up Vitest for Vue frontend | P1 | M | Install vitest + @vue/test-utils + jsdom. Configure in vite.config.ts. Create one example component test to validate setup. This unblocks all frontend testing. |
| 6 | Add unit tests for Pinia auth store | P1 | M | Test login/logout flows, token storage, refresh logic, and 401 handling in `src/stores/auth.ts`. |
| 7 | Add unit tests for API client (client.ts) | P1 | M | Test request interceptor (JWT injection), 401 response handling, refresh queue, and retry logic. Mock axios. |
| 8 | Add unit tests for ValuationParser (ParseValueEstimate) | P1 | S | Pure function with complex parsing — ideal unit test target. Test JSON format, plain text format, malformed input, edge cases (negative values, missing fields). |
| 9 | Add unit tests for SettingsService | P1 | S | Test InitSettings (defaults creation), GetSetting (existing/missing key), SetSetting (create/update), GetAllSettings. Settings drive feature flags — bugs here cascade. |
| 10 | Add architecture test for full import matrix | P1 | M | Extend `architecture_test.go` to enforce: handlers may only import services/repository/models; services may only import repository/models; repository may only import models + gorm; models may only import stdlib. |
| 11 | Add Python team pipeline tests (coin_search, coin_shows) | P1 | L | Test `create_coin_search_team()` and `create_coin_show_team()` graph execution with mocked LLM. Verify state transitions, output schema compliance, and error handling. |
| 12 | Add unit tests for SocialService | P2 | M | Test FollowUser (follow/unfollow/self-follow), BuildUserList, CanViewCoins (public/private/follower), GetPublicProfileData. Social features have complex permission logic. |
| 13 | Add repository tests for AuthRepository | P2 | M | Test CreateUser, FindByUsername, FindByID, token storage/retrieval. Use in-memory SQLite like coin_repository_test.go. |
| 14 | Add unit tests for NumisBidsService + ParseSaleDate | P2 | S | ParseSaleDate is a pure function parsing date strings — easy to test with table-driven tests for various date formats. |
| 15 | Add unit tests for middleware/ratelimit.go | P2 | S | Test rate limiting with rapid requests, verify 429 response, test limit reset. |
| 16 | Add Python tests for LLM provider (get_chat_model, get_search_model) | P2 | M | Test provider selection (Anthropic vs Ollama), model configuration, and search tool binding. Mock external LLM calls. |
| 17 | Add Python tests for search tools (SearXNG, numisbids parsers) | P2 | M | Test _parse_vcoins, _parse_mashops, _parse_lot_page with fixture HTML. These parse external HTML — fragile by nature, must be tested. |
| 18 | Add Vue component tests for CoinForm | P2 | L | CoinForm is the most complex component. Test validation, field binding, image handling, and submit flow. |
| 19 | Add Vue component tests for CoinSearchChat | P2 | L | Test SSE message handling, suggestion rendering, error states, and chat input. |
| 20 | Add handler tests for AdminHandler | P2 | M | Test admin-only route protection, user management, and settings endpoints. Verify non-admin users get 403. |
| 21 | Add Go test coverage CI gate | P2 | S | Add `go test -coverprofile` to CI. Set minimum threshold (start at 20%, increase over time). Block PRs that decrease coverage. |
| 22 | Add Python test for stream_graph_events | P2 | M | Test SSE event generation with mocked graph. Verify event format, suggestion extraction, and error handling during streaming. |
| 23 | Add handler tests for ImageHandler | P3 | M | Test image upload (valid/invalid formats, size limits), retrieval, and deletion. Images involve file I/O — common bug source. |
| 24 | Add handler tests for BulkHandler | P3 | M | Test bulk tag, bulk delete, and bulk export. Verify partial failure handling and ownership checks across multiple coins. |
| 25 | Add Python tests for remaining team pipelines (grading, price_trends, similar_lots, photo_guide, gap_analysis, auction_search) | P3 | L | Each pipeline needs at minimum a smoke test with mocked LLM verifying state flow and output schema. |

---

## Summary Assessment

The project has a **solid foundation** in the few tests that exist — behavioral tests, real SQLite, good assertions. But the coverage is dangerously thin. Only 1 of 21 repositories and 1 of 16 services have any tests. Zero handler tests means zero HTTP-layer validation. The frontend is completely untested. The Python agent tests cover models and utilities but skip all 11 team pipelines.

**Immediate risk:** Auth, middleware, and handler code has zero test coverage. Any refactor to those paths has no safety net. The architecture tests catch import violations but not behavioral regressions.

**Recommendation:** Prioritize P0 items (auth handler/service/middleware tests) as they represent security-critical paths with zero coverage. Then set up Vitest for frontend and establish a CI coverage gate to prevent further regression.
