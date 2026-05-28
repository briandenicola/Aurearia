# 2. Three-Service Architecture

Date: 2026-05-28
Status: Accepted

## Context

Ancient Coins has three distinct concerns that resist co-location:

1. **Presentation** — a Vue 3 PWA delivering an offline-capable,
   mobile-first museum-style UI.
2. **Business logic, persistence, and auth** — a Go service that owns
   the SQLite database, JWT issuance, request validation, and acts as
   the single ingress for the browser.
3. **LLM orchestration** — multi-agent pipelines built on LangGraph
   and LangChain, which require the Python ecosystem.

Co-locating all three in a single binary is non-viable: Go and Python
share neither runtime nor package ecosystem. Embedding the LLM
orchestration inside the Go API via gRPC stubs was considered, but
would have bound the API's release cadence to the rapidly-evolving
LangChain/LangGraph dependency surface, and pulled a large Python
dependency tree into operational scope for every API change.

Equally, exposing the Python agent directly to the browser was
rejected: it would force duplicate auth, duplicate rate limiting,
and a second CORS surface.

## Decision

We will deploy three independent services:

- **Vue 3 SPA + PWA** — TypeScript, Pinia, Vite. Dev server on
  port 5173; production assets bundled into the Go binary's
  `wwwroot/` and served as static files.
- **Go API** on port 8080 — Gin + GORM + SQLite. Owns authentication,
  data, request validation, and acts as the *only* HTTP client of the
  Python agent service. Contains zero LLM/agent logic.
- **Python agent** on port 8081 — FastAPI + LangGraph + LangChain.
  **Stateless**: no database access, no persistent storage. All
  configuration (API keys, model selection, prompts, user context)
  is passed per-request from the Go API.

SSE streams flow Python → Go → Vue. The Go API proxies the byte
stream via `services/agent_proxy.go` without parsing or buffering
event content.

## Consequences

+ Isolation of failure domains — an agent crash cannot corrupt
  the database or kick users out of the API.
+ LLM dependency upgrades stay inside the Python container and
  cannot break the Go build or migrations.
+ Each service has its own lifecycle: build, test, deploy, scale.
+ Constitution Principle II (Three-Service Architecture) is
  mechanically enforced by separate Dockerfiles, separate test
  suites, and separate ports — no shortcut bypass is possible.
+ The stateless agent service is trivially horizontally scalable.
− Adds an HTTP hop for AI requests. Acceptable: agent calls are
  user-initiated and never on the latency-critical hot path.
− Three-service local dev requires `task up-all` (or compose),
  not a single `go run`. Documented in the Taskfile.
− Two Dockerfiles to maintain instead of one.

## Related

- Constitution Principle I (Layered Architecture — Go side)
- Constitution Principle II (Three-Service Architecture)
- Constitution §17 (Quality Gate)
- `docs/ARCHITECTURE.md` — full system topology
- `src/api/services/agent_proxy.go` — SSE proxy implementation
