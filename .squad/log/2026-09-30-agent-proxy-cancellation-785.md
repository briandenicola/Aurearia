# #785 — Agent cancellation and error-body hygiene

- Date: 2026-09-30
- Branch: beta (base `4cfced0c`)
- Owner: Copilot CLI implementation owner
- Status: implemented, gated, independently reviewed PASS (twice). Owner acceptance outstanding.

## Scope

Issue #785 asked for three things:
- audit every `AgentProxy` method;
- make a Go-side timeout or disconnect actually stop the Python work;
- stop agent error bodies reaching logs or returned errors verbatim.

Out of scope, and untouched: the auction subsystem, and the #779 verified-only filter.

## What changed

**Agent (`src/agent`)**
- New `app/request_cancellation.py`. `cancel_on_disconnect(http_request, work)` runs the route's work as a task, next to a watcher.
  - The watcher blocks on `http_request.receive()` until it sees `http.disconnect`. On a disconnect the work task is cancelled, which interrupts in-flight model, dealer and search awaits, and `ClientDisconnectedError` is raised. `main.py` maps that to a 499.
  - If the watcher itself fails, that is not treated as a disconnect: a warning is logged and the work finishes.
  - The `finally` block always awaits both tasks, so the work's cleanup runs even when the route is cancelled.
  - Why receive and not `Request.is_disconnected()` polling: `InternalServiceAuthMiddleware` is a Starlette `BaseHTTPMiddleware`, and behind it `is_disconnected()` never observed the disconnect. This was verified with a real-uvicorn end-to-end test.
- Wrapped routes: wishlist-url/extract, search/alerts, search/comparables, analyze, grade, intake/draft, check-availability, bid-market-signal, wishlist-featured-summary, set-builder/run and exploration decide. Every route with an `except Exception` fallback now re-raises `ClientDisconnectedError` first, so the fallback cannot swallow the disconnect.

**Go (`src/api`)**
- Every non-200 log site now uses `sanitizeAgentErrorBodyForLog`: 8 in `agent_proxy.go`, plus deep-identify.
- `ExtractWishlistURL` no longer returns the raw body in its error. It logs the sanitized body and returns `agentServiceHTTPError`.
- `GenerateIntakeDraft` used `context.Background()`. It now takes the caller's ctx, keeping the 5-minute cap, threaded through `IntakeProxyClient`, `CoinIntakeService.CreateDraft` and the handler (`c.Request.Context()`).

## Deliberate non-changes

- **Streaming routes are not wrapped:** search/coins, search/shows, portfolio/review, deep-identify/stream and copilot/execute. Uvicorn advertises ASGI 2.3, and there Starlette's `StreamingResponse` runs its own blocking disconnect listener and cancels the stream. The reviewer confirmed this in source.
- **Copilot's `cancellation_check=http_request.is_disconnected`** is ineffective behind `BaseHTTPMiddleware`. Copilot currently stops only through the streaming mechanism above. If uvicorn moves to ASGI 2.4, the stream would stop only when a send fails. Follow-up.

## Known limitations (follow-up)

- `sanitizeAgentErrorBodyForLog` redacts by JSON key only. Secrets inside a value, or anywhere in a non-JSON body, are still logged, truncated to 200 characters. Returned errors are safe: the new plain-text test proves no method echoes the body.
- Task cancellation does not stop work already running in a threadpool (`asyncio.to_thread`).
- End-to-end cancellation tests cover comparables and analyze only.
- Hardening: `watcher.exception()` raises if the watcher was *cancelled*, so check `watcher.cancelled()` too. Low likelihood.
- Behaviour change, intended: an intake draft now stops when the browser disconnects.

## Evidence (at the tree committed with this log)

- Tests:
  - `src/agent/tests/test_request_cancellation.py`: 5 unit tests plus 2 real-uvicorn end-to-end tests (client timeout of 0.5s, work stops).
  - `src/api/services/agent_proxy_error_hygiene_test.go`: three table tests over 13 methods: a JSON secret is not in logs or errors; a plain-text body is not echoed in errors; a caller timeout cancels the agent request.
- Tamper tests, each failing as expected, then restored:
  - removing the comparables wrapper;
  - reverting the Analyze sanitize;
  - putting intake back on `context.Background()`;
  - disabling the watcher-exception branch;
  - gathering only the watcher (this one first passed with a `sleep(0)` cleanup; the test was strengthened);
  - putting back the ExtractWishlistURL body echo.
- `task check:go` exits 0. `task check:agent` exits 0 with 773 passed. OpenAPI was not affected: no handler annotations or routes changed.
- Independent aurearia-reviewer: PASS, then re-review PASS after the follow-up repairs.

## Next action

Owner acceptance of #785, then close it. #766 still needs its own spec.
