# Go to Python agent cancellation contract

Date: 2026-09-30
Scope: #785
Status: Implemented and independently reviewed; owner acceptance pending
Authority: Constitution Principles I/II; sections 17-18

- The Go API cancels an agent call by closing the connection (ctx timeout or
  caller disconnect). There is no separate cancel endpoint.
- Non-streaming agent routes wrap their work in `cancel_on_disconnect`, which
  cancels the task and answers 499. The status is never read by Go; it exists
  for agent logs only.
- Streaming routes rely on Starlette's `StreamingResponse` disconnect listener
  under uvicorn's ASGI 2.3, and are not wrapped.
- Disconnect detection depends on `BaseHTTPMiddleware` receive semantics and
  the ASGI spec version. A Starlette or uvicorn upgrade must re-run
  `src/agent/tests/test_request_cancellation.py`, whose real-uvicorn tests are
  the only guard.
- Mixed Go/agent versions during an upgrade are safe in either order; rollback
  of either image needs no data or config change.
