# Project Context

- **Owner:** Brian
- **Project:** Ancient Coins — full-stack PWA for managing a personal ancient coin collection
- **Stack:** Go 1.26 / Gin / GORM / SQLite (API), Vue 3 / TypeScript / Pinia / Vite (Frontend), Python 3.12 / FastAPI / LangGraph (Agent), Docker
- **Architecture:** Layered — Handler → Service → Repository → Database. Enforced by architecture_test.go.
- **Created:** 2026-04-24

## Core Context

Between 2026-04-24 and 2026-05-22, Cassius completed critical backend P0/P1/P2 fixes and shipped the Auction Ending feature with comprehensive manual-run logging and ground-truth diagnostics:

1. **Code Quality Baseline (2026-04-24):** Backend codebase audited at grade B-. Identified (a) settings_service.go and auth.go bypass repository layer with direct gorm.DB access, (b) double-close panic risk in both schedulers, (c) business logic leaks into analysis/agent/coins/admin handlers, (d) inconsistent error handling (7+ locations silently drop errors), (e) thin input validation. 20 prioritized backlog items created.

2. **P0 Fixes — Double-Close Panics (2026-04-24):** Added `sync.Once` guards to ValuationScheduler.Stop() and AvailabilityScheduler.Stop() to prevent panic on close of already-closed stop channel. Also added defense-in-depth column allowlist to CoinRepository.Suggestions() to validate column before SQL interpolation (matching handler whitelist).

3. **P2 Input Validation & File Cleanup (2026-04-25):** Added handler-level validation to coins List endpoint: page ≥1, limit 1–100, sort field allowlist, order asc|desc. Invalid input returns HTTP 400 with clear message. Fixed orphan file risk in ImageService.UploadImage(): failed DB insert now triggers os.Remove() cleanup.

4. **Auction Ending Scheduler Implementation (2026-05-21):** Built daily scheduler (default 08:00, 1440min interval) that checks BIDDING lots ending today, sends consolidated Pushover notifications per user. In-memory idempotency tracking prevents duplicates within same day. Added GetEndingToday() repository method + settings keys (AuctionEndingCheckEnabled, AuctionEndingCheckStartTime, AuctionEndingCheckInterval) matching wishlist scheduler naming.

5. **Manual Trigger & Run Logging (2026-06-10):** Added AuctionEndingRun model (ID, TriggerType, TriggerUserID, Status, LotsChecked, AlertsSent, DurationMs, timestamps, ErrorMessage). Created auction_ending_repository.go with CRUD methods + ListRuns pagination. Refactored scheduler to log every run (scheduled/manual) via RunNow(triggerUserID) method. Added handlers/auction_ending_admin.go endpoints: GET /api/admin/auction-ending-runs (history), POST /api/admin/auction-ending/run (manual trigger). Parity achieved with Valuation/Wishlist schedulers.

6. **Critical Bug Fixes — UTC Calendar Day & NULL Date Handling (2026-05-22):** Two major bugs found and fixed: (a) NULL sale_date bug — lots with NULL sale_date but valid auction_end_time were excluded. Fixed query to check BOTH fields with NULL guards. Added test case. (b) UTC semantic bug — lots ending at midnight UTC were excluded for users in negative-offset TZs (US). Changed semantic from "ends on UTC calendar day" to "ends within next 24 hours" using (now, now+24h] window. Renamed GetEndingToday() to GetEndingSoon(). Hardened status comparison with LOWER(status) = 'bidding'. Added 10 test cases covering 23h/12h/2h/25h/-1h windows, mixed-case status, and Brian's exact scenario.

7. **Ground-Truth Diagnostics (2026-05-22):** Built debug endpoint GET /api/admin/auction-ending/debug returning (a) total lots, (b) lots by status, (c) lots matching scheduler query, (d) ALL BIDDING lots with ALL date fields including linked AuctionEvent dates (via LEFT JOIN). Provided SQL query for immediate inspection. Lesson: never ship a query fix without inspecting real production data first.

**Key Patterns Established:** (a) Time-sensitive queries use rolling (now, now+24h] window, not calendar-day boundaries. (b) Case-sensitive enums need LOWER() in SQL. (c) Multi-field date logic requires explicit NULL guards and JOIN diagnostics. (d) Scheduler run logging via TriggerType + run history table enables production audit and manual testing. (e) Interface parity across schedulers: Valuation/Wishlist/AuctionEnding all follow same manual-trigger + run-log pattern.

## Team Updates

- **2026-05-22:** Auction-ending feature shipped end-to-end. Manual trigger + run log (prior commits) now works correctly with null-date bugfix. Heritage Auctions and non-NumisBids sources properly tracked. Feature complete and production-ready. Lesson: never ship query fix without inspecting real production data.

- **2026-05-22:** Collaborated with Aurelia (frontend) and Brutus (testing) on auction-ending manual-run feature. Brutus's comprehensive test suite (16 tests) APPROVED. Aurelia implementing corresponding UI; follow-up endpoint URL fixup aligning with actual contract.

- **2026-05-28:** Constitution v2.0.0 landed. Read `.specify/memory/constitution.md`. §17 Quality Gate gates every PR (includes go vet/test). §21 DoD is a 14-item checklist. §18 forbids SESSION-NOTES.md — Squad handoff is `.squad/log/` + history + decisions.md. Principle I (Layered Architecture) enforced by architecture_test.go; import rules: handlers→services→repository→models.

- **2026-05-28 (Phase 2):** Phase 2 of tech-inventory alignment landed. `specs/` is on-disk home for SpecKit workflow. Backlog in `specs/_backlog/`, active features in `specs/NNN-slug/`, retroactive anchor in `specs/001-foundation/spec.md`. Four session-protocol prompts in `.github/prompts/`.

- **2026-05-28 (Phase 3a):** Phase 3a landed. docs/prd.md is product source of truth. Four ADRs in docs/adr/ documenting v1.0 architecture. Any new material design choice requires an ADR per §22.

- **2026-05-31:** Feature #217 Go-side Shared Collection Tool Layer (commit c3e8c2b) — internal token service + middleware + 6 internal tool endpoints (search_my_collection, get_coin, collection_summary, top_coins_by_value, propose_update, commit_update) + removed keyword gate. Tool definitions live in Go; Python ReAct agent integration PENDING (next-session pickup). Feature #216 Intake Card Authority Fix (commit a7b6a04) — explicit image labeling in generate_intake_draft() distinguishes coin photos from collector card, strengthened INTAKE_PROMPT with dedicated card handling section, treats card text as PRIMARY authoritative source. Response schema unchanged (Principle VII). All tests pass: go vet clean, go test -v ./..., pytest 47/47 passed. Decision inbox entries merged to decisions.md.

- **2026-05-31:** Feature #217 Python-side ReAct Collection Agent (commit 3bc04de) — Built Python half of shared collection tool layer: (1) `app/tools/collection_tools.py`: 6 LangChain StructuredTools (search_my_collection, get_coin, collection_summary, top_coins_by_value, propose_update, commit_update) calling Go `/internal/tools/` via httpx with Authorization: Bearer {internal_token} header. Identity flows ONLY via 30s internal token; Python never sends userID (Principle XI+XII). (2) `app/teams/collection_chat.py`: ReAct agent factory using `create_react_agent(get_chat_model, tools, prompt=...)` — system prompt instructs multi-intent tool calling in one turn for compound questions. (3) `app/supervisor.py`: Added `collection` category to ROUTER_PROMPT with clear distinction ("collection" = ownership lookups + compound own-coin questions, "portfolio" = aggregate narrative); threaded `tools_base_url` + `internal_token` into supervisor → collection node closure (per-request binding). (4) Request threading: Added `internal_token` + `tools_base_url` fields to `CoinSearchRequest`; `routes.py` passes them into `create_supervisor(...)`. Feature #217 now **end-to-end complete** (Go side landed in c3e8c2b). Multi-intent bug fixed: questions like "do I have moose coins AND how much are they worth?" now route to collection node which calls multiple tools in one ReAct turn instead of misrouting to portfolio (single-category limitation resolved). Tests: 57 passed (integration + unit), ruff clean, go build clean. Follows Decision #11 LLM-intent directive. Next: #218 external-adapter follow-up (tomorrow).

- **2026-05-31:** Bugfix #217 httpx Response Mocking (commit f95fb39) — Fixed 3 failing tests in `test_collection_tools.py`. Root cause: Tests mocked `httpx.Response.json()` and `.raise_for_status()` as `AsyncMock`, but these are synchronous methods in httpx (the response object has sync methods, only the client operations are async). Changed test mocks to use regular `Mock()` for response methods while keeping `AsyncMock` for `client.post()`. Full suite now 60/60 passed.

## Learnings

- **httpx Response Mocking Pattern (2026-05-31):** When mocking httpx async client calls, mock the CLIENT's async methods (`.post()`, `.get()`, etc.) as `AsyncMock`, but mock the RESPONSE object's methods (`.json()`, `.raise_for_status()`, `.text`) as regular synchronous `Mock` instances. The response object in httpx exposes synchronous methods even when obtained from an async context manager.
