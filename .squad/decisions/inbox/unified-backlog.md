# Code Review & Grading Analysis — Unified Backlog

**Date:** 2026-04-24
**Reviewed by:** Maximus (Architecture), Cassius (Backend), Aurelia (Frontend), Brutus (Testing)

---

## Report Card

| Domain | Grade | Reviewer | Summary |
|--------|-------|----------|---------|
| Overall Architecture | B+ | Maximus | Clean 3-service separation, layered Go API enforced by tests. Globals undermine DI. |
| Layered Compliance | B+ | Maximus | Handler→Service→Repo→DB enforced. Some handlers leak business logic. |
| Dependency Injection | B- | Maximus | Constructor injection is the norm, but 3 package-level globals bypass it. |
| Error Handling | C+ | Maximus/Cassius | Silent error drops in social.go (7+), broad Python catches, no wrapping convention. |
| Auth/Security | B+ | Maximus/Cassius | JWT + refresh + API keys + WebAuthn. API key middleware bypasses repo layer. |
| API Design | B | Maximus | RESTful, Swagger-annotated. SSE contract implicit. |
| Configuration | B+ | Maximus | Env-based, sensible defaults. Clean. |
| Service Integration | B+ | Maximus | Agent proxy well-factored. `collectSSE` brittle on event types. |
| Documentation | A- | Maximus | 761-line ARCHITECTURE.md is excellent. Missing SSE event contract. |
| Handler Quality | B | Cassius | Most thin. analysis.go, agent.go, admin.go, coins.go leak logic. |
| Service Quality | B- | Cassius | Good sentinel errors. settings_service bypasses repo. Scheduler panic risk. |
| Repository Quality | B | Cassius | Good scopes. Some non-transactional multi-step writes. |
| Model Design | B | Cassius | Clean structs. Validation tags thin. |
| Input Validation | C | Cassius | Sparse handler validation. Silent defaults. No request size limits. |
| Component Quality | B | Aurelia | Good Composition API. Several 400+ line components need splitting. |
| TypeScript Usage | B+ | Aurelia | Very few `any` casts. Good interfaces. |
| State Management | B- | Aurelia | Stores too lean — no error state, auth drift after refresh. |
| API Client | B+ | Aurelia | Solid refresh queue. sanitizeCoin has truthiness bug. |
| Accessibility | D+ | Aurelia | Minimal ARIA. No focus traps. Clickable divs not keyboard-accessible. |
| Responsive Design | B- | Aurelia | PWA-first but dense pages cramped on mobile. |
| PWA Quality | C+ | Aurelia | Missing icons, no offline fallback, no update prompt. |
| Go Unit Test Coverage | D | Brutus | 3.5-4.6%. Only CoinRepo + CoinService tested. |
| Go Architecture Tests | B- | Brutus | Import rules enforced. Missing full import matrix. |
| Go Integration Tests | F | Brutus | Zero HTTP-level handler tests. |
| Frontend Tests | F | Brutus | Zero test files. No framework configured. |
| Python Agent Tests | C+ | Brutus | 31 tests pass. Zero team pipeline tests. |
| Overall Test Strategy | D | Brutus | No test plan, no coverage thresholds, no CI enforcement. |

**Composite Grade: C+/B-** — Strong architecture and documentation, but significant debt in testing, error handling, accessibility, and validation.

---

## Unified Backlog

Items are deduplicated across all four reviews. Where multiple reviewers flagged the same issue, credit is given to all.

### P0 — Critical (Fix Now)

| # | Title | Area | Effort | Description | Flagged By |
|---|-------|------|--------|-------------|------------|
| 1 | Sanitize v-html AI content (XSS risk) | security | S | `CoinSearchChat.vue:31`, `CoinDetailPage.vue:282,290,294` render AI content with `v-html` without visible DOMPurify sanitization. Direct XSS vector. | Aurelia |
| 2 | Add admin role guard to router | security | S | `/admin` route has no role-based protection — any authenticated user can navigate directly. UI hides link but route is accessible. `router/index.ts:138-143`. | Aurelia |
| 3 | Guard scheduler Stop() against double-close panic | backend | S | `valuation_scheduler.go:60` and `availability_scheduler.go:61` call `close(s.stopCh)` with no guard. Double-call panics. Use `sync.Once`. | Cassius |
| 4 | Add column whitelist to CoinRepository.Suggestions() | security | S | `repository/coin_repository.go:396` interpolates `column` directly into SQL WHERE. Handler validates via switch, but repo is unprotected — defense-in-depth needed. | Cassius |
| 5 | Add auth handler integration tests | testing | L | Zero HTTP-level tests for login, register, refresh, logout. Auth is the most security-critical path. Create `handlers/auth_handler_test.go` with httptest + Gin test router. | Brutus |
| 6 | Add auth service unit tests | testing | M | Test Login, Register, HashPassword, ValidateToken, RefreshToken. Auth bugs are security bugs. | Brutus |
| 7 | Add auth middleware unit tests | testing | M | Test JWT extraction, expired/missing/invalid token handling, user ID context injection. `middleware/auth.go` is untested. | Brutus |
| 8 | Add coin handler integration tests | testing | L | Test GET/POST/PUT/DELETE with authenticated requests. Verify response codes, JSON bodies, ownership enforcement at HTTP layer. | Brutus |

### P1 — High (Next Sprint)

| # | Title | Area | Effort | Description | Flagged By |
|---|-------|------|--------|-------------|------------|
| 9 | Eliminate `services.GetSetting()` global | architecture | M | Package-level function backed by global DB. Inject `SettingsService` interface. Remove `InitSettings()` global init. | Maximus/Cassius |
| 10 | Inject logger instead of `services.AppLogger` global | architecture | M | Package-level global used across handlers/services. Replace with logger interface via constructors. | Maximus/Cassius |
| 11 | Remove `cancelMap` global in valuation service | architecture | S | `valuation_service.go:25-27` package-level global. Move into `ValuationService` struct. | Maximus |
| 12 | Extract settings_service into proper repository | architecture | M | `settings_service.go:80-131` has global `settingsDB` with direct GORM calls. Create `SettingsRepository` with constructor injection. | Cassius |
| 13 | Move middleware DB access to repository layer | architecture | M | `middleware/auth.go:79-101` runs direct GORM queries for API key validation. Create `ApiKeyRepository`. | Maximus/Cassius |
| 14 | Add transactions to non-atomic multi-step writes | backend | M | `AuctionLotService.ConvertToCoin`, `AuctionLotRepository.Upsert`, `SocialRepository.BlockUser`, availability run updates — all need transactions. | Cassius |
| 15 | Fix silent error drops in social.go | backend | M | `handlers/social.go` ignores errors with `_` in 7+ locations. Log errors, return appropriate HTTP responses. | Maximus/Cassius |
| 16 | Extract business logic from handlers | backend | L | `analysis.go`, `agent.go`, `coins.go`, `admin.go` contain business logic. Move to service layer. | Maximus/Cassius |
| 17 | Fix auth store drift after token refresh | frontend | S | `api/client.ts:58-60` updates localStorage but not Pinia auth store. Causes stale UI state. | Aurelia |
| 18 | Clear all timers on unmount | frontend | M | 15+ files with `setTimeout`/`setInterval` lacking cleanup in `onUnmounted`. Memory leak on long sessions. | Aurelia |
| 19 | Revoke object URLs in CoinForm | frontend | S | 3 `createObjectURL` calls, 0 `revokeObjectURL` calls. Blob URLs accumulate. | Aurelia |
| 20 | Add PWA icons to public/ | frontend | S | `pwa-192x192.png` and `pwa-512x512.png` referenced in manifest but missing. Breaks mobile installability. | Aurelia |
| 21 | Set up Vitest for Vue frontend | testing | M | Install vitest + @vue/test-utils + jsdom. Create example test. Unblocks all frontend testing. | Brutus |
| 22 | Add Pinia auth store tests | testing | M | Test login/logout flows, token storage, refresh logic, 401 handling. | Brutus |
| 23 | Add API client tests | testing | M | Test JWT injection, 401 handling, refresh queue, retry logic. | Brutus |
| 24 | Add ValuationParser unit tests | testing | S | Pure function with complex parsing — ideal test target. JSON/plain text/malformed input. | Brutus |
| 25 | Add SettingsService unit tests | testing | S | Test InitSettings, GetSetting, SetSetting, GetAllSettings. Settings drive feature flags. | Brutus |
| 26 | Add full import matrix architecture test | testing | M | Enforce: handlers→services/repo/models; services→repo/models; repo→models+gorm; models→stdlib only. | Brutus |
| 27 | Add Python team pipeline tests | testing | L | Test coin_search and coin_shows graph execution with mocked LLM. Verify state transitions, schema compliance. | Brutus |

### P2 — Medium (Planned)

| # | Title | Area | Effort | Description | Flagged By |
|---|-------|------|--------|-------------|------------|
| 28 | Fix `sanitizeCoin()` truthiness bug | frontend | S | `api/client.ts:111-113` treats `0` as falsy. Coin with `currentValue=0` gets overwritten by `purchasePrice`. | Maximus |
| 29 | Define SSE event contract (Go ↔ Python) | docs | S | SSE streaming protocol is implicit. Document event types, data schemas, termination. | Maximus |
| 30 | Add supervisor and team pipeline tests (Python) | agent | M | No tests for supervisor routing, recursion limits, team graph wiring. | Maximus |
| 31 | Replace broad `except Exception` in Python routes | agent | S | `routes.py:108-126`, `167-204` catch all exceptions. Use specific types. | Maximus |
| 32 | Audit and fix swallowed errors | backend | M | Systematic pass: availability_service, valuation_service, social_service, admin handler. | Cassius |
| 33 | Replace hardcoded values with config | backend | S | `uploads` path, Anthropic API URL, default page limits. | Cassius |
| 34 | Add input validation to coins list endpoint | backend | S | Validate page≥1, 1≤limit≤100, whitelist sort fields. Return 400 on invalid. | Cassius |
| 35 | Fix orphan file risk in image_service | backend | S | File written before DB record. If insert fails, file orphaned. | Cassius |
| 36 | Add timeouts and context to HTTP clients | backend | M | `numisbids_service.go` and `availability_service.go` lack explicit timeouts. | Cassius |
| 37 | Add shutdown mechanism to rate limiter goroutine | backend | S | `middleware/ratelimit.go:23-42` cleanup goroutine has no stop mechanism. | Cassius |
| 38 | Add transaction to auction lot upsert | backend | S | `auction_lot_repository.go:141-166` read-modify-write without tx. Race condition. | Maximus/Cassius |
| 39 | Decompose god-pages (Admin, Settings, CoinDetail) | frontend | L | AdminPage (1378 lines), SettingsPage (1371), CoinDetailPage (1242). Extract sub-components. | Maximus/Aurelia |
| 40 | Add focus traps to modals | frontend | M | 6+ modal components lack focus trapping. No Escape key on most. | Aurelia |
| 41 | Add ARIA to AutocompleteInput | frontend | S | Missing `role="listbox"`, `role="option"`, `aria-activedescendant`. | Aurelia |
| 42 | Make clickable divs keyboard-accessible | frontend | M | CoinCard, SwipeGallery, CoinSearchChat drawer need tabindex, role, keydown. | Aurelia |
| 43 | Add error state to coins store | frontend | S | No `error` ref. Errors bubble unhandled. Pages can't reactively show store errors. | Aurelia |
| 44 | Add PWA update prompt | frontend | S | `registerType: 'autoUpdate'` silently swaps SW. Add user notification. | Aurelia |
| 45 | Add SocialService unit tests | testing | M | Follow/unfollow/self-follow, permission logic. Complex rules. | Brutus |
| 46 | Add AuthRepository tests | testing | M | CreateUser, FindByUsername, token storage. Use in-memory SQLite. | Brutus |
| 47 | Add NumisBidsService + ParseSaleDate tests | testing | S | Pure function parsing date strings. Table-driven tests. | Brutus |
| 48 | Add rate limiter middleware tests | testing | S | Rapid requests, 429 response, limit reset. | Brutus |
| 49 | Add Python LLM provider tests | testing | M | Provider selection, model config, search tool binding. Mock LLM. | Brutus |
| 50 | Add Python search tool tests | testing | M | Parse vcoins, mashops, lot pages with fixture HTML. | Brutus |
| 51 | Add CoinForm component tests | testing | L | Validation, field binding, image handling, submit flow. | Brutus |
| 52 | Add CoinSearchChat component tests | testing | L | SSE handling, suggestion rendering, error states. | Brutus |
| 53 | Add AdminHandler tests | testing | M | Admin-only route protection, user management. Verify 403 for non-admin. | Brutus |
| 54 | Add Go test coverage CI gate | testing | S | `go test -coverprofile` in CI. Start threshold at 20%. | Brutus |
| 55 | Add Python stream_graph_events test | testing | M | Mock graph, verify SSE format, suggestion extraction. | Brutus |

### P3 — Low (Backlog)

| # | Title | Area | Effort | Description | Flagged By |
|---|-------|------|--------|-------------|------------|
| 56 | Move API key lookup from middleware to repository | architecture | S | Maintain repo abstraction consistently. | Maximus |
| 57 | Reduce inline styles in Vue pages | frontend | M | Extract to CSS classes using project CSS variables. | Maximus |
| 58 | Split coins store | frontend | S | `stores/coins.ts` manages too many concerns. Split into collection + detail stores. | Maximus |
| 59 | Centralize all fetch calls through API client | frontend | S | `CoinDetailPage.vue:524` and `useImageProcessor.ts:102` bypass centralized client. | Maximus |
| 60 | Add response models to Python streaming endpoints | agent | M | SSE routes lack `response_model`. Document payload schemas. | Maximus |
| 61 | Enforce schema parsing on LLM team outputs | agent | M | Only availability verdicts strictly parsed. Others rely on prompts alone. | Maximus |
| 62 | Separate operational endpoints from agent API | agent | S | `/logs` and `/log-level` mixed with agent routes. Move to admin router. | Maximus |
| 63 | Remove `any` casts in sanitizeCoin | frontend | S | `api/client.ts:106-107`. Use `Partial<Coin>` or `Record<string, unknown>`. | Maximus/Aurelia |
| 64 | Batch-insert in tag_repository.BulkAttachToCoin() | backend | S | Loop of individual inserts. Batch for performance. | Cassius |
| 65 | Add pagination to admin_repository.ListUsers() | backend | S | Unbounded query. Add page/limit. | Cassius |
| 66 | Batch-update in price_alert_repository | backend | S | Loop-based saves. Batch update. | Cassius |
| 67 | Fix AllowedOrigins() fallback logic | backend | S | Comma-split on single origin field. Duplicate localhost entry. | Cassius |
| 68 | Add model validation tags | backend | M | User, AuctionLot, ValuationRun, PriceAlert, Notification need binding/validation tags. | Cassius |
| 69 | Decouple agent_proxy SSE from http.ResponseWriter | backend | L | Return io.Reader/channel from service, let handler manage HTTP. | Cassius |
| 70 | Check PRAGMA execution errors in database.go | backend | S | PRAGMA return values unchecked. Silent failure possible. | Cassius |
| 71 | Replace index-as-key in v-for loops | frontend | S | CoinSearchChat, CoinSuggestionGrid, CoinShowResultsGrid, AdminLogsSection. | Aurelia |
| 72 | Improve form validation | frontend | M | Password strength rules, confirmation matching, name uniqueness. | Aurelia |
| 73 | Add responsive handling for dense pages | frontend | M | Horizontal scroll for tables/grids on mobile. | Aurelia |
| 74 | Redirect authenticated users from login page | frontend | S | Already-logged-in users can visit /login. Redirect to collection. | Aurelia |
| 75 | Add ImageHandler tests | testing | M | Upload (valid/invalid formats), retrieval, deletion. | Brutus |
| 76 | Add BulkHandler tests | testing | M | Bulk tag/delete/export. Partial failure, ownership checks. | Brutus |
| 77 | Add remaining Python pipeline tests | testing | L | grading, price_trends, similar_lots, photo_guide, gap_analysis, auction_search. | Brutus |

---

## Summary Statistics

| Priority | Count | Effort Breakdown |
|----------|-------|------------------|
| P0 (Critical) | 8 | 2S, 2M, 2L, 2S |
| P1 (High) | 19 | 7S, 8M, 2L, 2S |
| P2 (Medium) | 28 | 9S, 12M, 4L, 3S |
| P3 (Low) | 22 | 11S, 6M, 2L, 3S |
| **Total** | **77** | |

| Area | Count |
|------|-------|
| Testing | 27 |
| Backend | 16 |
| Frontend | 18 |
| Architecture | 8 |
| Security | 3 |
| Agent | 5 |
| Docs | 1 |

---

## Recommended Attack Order

**Phase 1 — Security & Stability (P0)**
Close XSS risk, add admin guard, fix panic bugs, harden SQL. Add auth test safety net.

**Phase 2 — Structural Debt (P1)**
Eliminate globals, fix DI violations, add transactions. Set up frontend testing. Fix auth store drift and memory leaks.

**Phase 3 — Quality & Coverage (P2)**
Systematic error handling audit. Decompose god-pages. Accessibility pass. Expand test coverage to 40%+ Go, basic frontend, full Python pipelines.

**Phase 4 — Polish (P3)**
Performance optimizations, form validation, responsive fixes, remaining test coverage.
