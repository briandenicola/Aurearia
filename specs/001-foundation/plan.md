# Implementation Plan: v1.0 Foundation — Ancient Coins PWA

**Branch**: `001-foundation` | **Date**: 2026-05-28 (retroactive) | **Spec**: [./spec.md](./spec.md)
**Input**: Feature specification from `specs/001-foundation/spec.md`

## Summary

Ship a self-hosted personal PWA for managing an ancient coin collection as a three-service architecture: Go/Gin API for REST + persistence, Vue 3 PWA for UI, Python/FastAPI + LangGraph for AI features. All three services share a single SQLite file (owned exclusively by the Go API) and are packaged as two Docker containers behind a single `docker-compose.yaml`. This plan is **retroactive** — every decision below is already implemented in production code.

## Technical Context

**Language/Version**: Go 1.26.1 (API), TypeScript 5.x / Vue 3.4+ (web), Python 3.12 (agent)
**Primary Dependencies**: Gin, GORM, JWT, go-webauthn (API); Vite, Pinia, Axios, `lucide-vue-next`, vite-plugin-pwa (web); FastAPI, LangGraph, LangChain, ChatAnthropic, ChatOllama (agent)
**Storage**: SQLite single file, owned exclusively by Go API (`database/database.go`); Python agent is stateless
**Testing**: `go test ./...` including `architecture_test.go`; `vitest` + `vue-tsc --build`; `pytest` + `ruff check`
**Target Platform**: Linux server (Docker), modern evergreen browsers (PWA install on iOS/Android/desktop)
**Project Type**: Three-service web app (API + SPA + AI sidecar)
**Performance Goals**: List view first paint < 1.5s on local network for 50-coin catalog; AI stream first token < 5s on Anthropic; scheduler tick under 1s per user
**Constraints**: Single-node SQLite (personal scale); no SPA → agent direct calls (Principle III); PWA offline-read only in v1.0
**Scale/Scope**: 1 primary user + small set of invited friends; hundreds to low thousands of coins

## Constitution Check

*GATE: Pass before any implementation. Re-checked at v1.0 release.*

- **Principle I (Layered Architecture)**: PASS — Handler → Service → Repository → Database enforced; no business logic in handlers; all GORM queries in `src/api/repository/`.
- **Principle II (Dependency Injection)**: PASS — only `main.go` imports `database`; all other packages receive `*gorm.DB` or interfaces via constructors.
- **Principle III (Service Boundary Separation)**: PASS — Go API has zero LLM logic; Python agent is stateless; Vue SPA talks only to Go API; SSE flows Python → Go → Vue.
- **Principle IV (Strict Typing & Build Parity)**: PASS — `go vet`, `vue-tsc --build`, `ruff check` all clean in CI.
- **Principle V (Design Token System)**: PASS — `variables.css` + `main.css` global classes are the only source of CSS values.
- **Principle VI (AI/Agent Isolation)**: PASS — search agents pass only tool-returned data; verification agents validate URLs and dates; Pydantic schemas on every worker output; supervisor enforces max-iteration cap.
- **Principle X (Architecture Enforcement)**: PASS — `architecture_test.go` validates import rules in CI.
- **Principle XII (Authentication & Token Policy)**: PASS — JWT access + refresh; WebAuthn passkey support; refresh queue in Axios interceptor.
- **Principle XIII (Security Baseline)**: PASS — CSP, service worker scoped to root, no secrets in client bundle, input validation at handler boundary.

No violations. No Complexity Tracking entries required.

## Approach

Three independently deployable services sharing one data store:

```
┌─────────────┐        ┌─────────────┐         ┌──────────────────┐
│  Vue 3 SPA  │ ─REST→ │  Go API     │ ─HTTP→  │  Python Agent    │
│  (PWA)      │ ←SSE── │  (Gin)      │ ←SSE──  │  (FastAPI +      │
│             │        │  port 8080  │         │   LangGraph)     │
└─────────────┘        └──────┬──────┘         │   port 8081      │
                              │ GORM            └──────────────────┘
                              ▼                  (stateless — config
                       ┌────────────┐             passed per-request)
                       │  SQLite    │
                       └────────────┘
```

- **Vue SPA** is built statically and served by the Go API container; in dev mode Vite proxies `/api` to `:8080`.
- **Go API** owns the database connection (Principle II); proxies all `/agent/*` to the Python service via `services/agent_proxy.go`, streaming SSE bytes through unchanged.
- **Python agent** never opens a database connection; the Go API passes API keys, model selection, prompts, and any user-specific context per request.

Constitution **Principle II (DI)** and **Principle III (Service Boundary Separation)** are the load-bearing rules — both are enforced by code (DI wiring in `main.go`; service boundary by absence of LLM imports in Go and absence of DB imports in Python).

## Tech Stack Detail

| Layer    | Stack                                                                  | Path           |
|----------|------------------------------------------------------------------------|----------------|
| Backend  | Go 1.26.1, Gin, GORM, SQLite, JWT, go-webauthn                         | `src/api/`     |
| Frontend | Vue 3, TypeScript, Pinia, Vite, vite-plugin-pwa, Axios, lucide icons   | `src/web/`     |
| Agent    | Python 3.12, FastAPI, LangGraph, LangChain, ChatAnthropic, ChatOllama  | `src/agent/`   |
| Build    | Multi-stage Docker (app container = Go+Vue, agent container = Python)  | `Dockerfile`, `src/agent/Dockerfile` |
| Orch     | docker-compose; Taskfile.yml for local dev (`task up-all`)             | repo root      |

## Key Decisions

1. **Layered Go architecture enforced by tests** (Principle I + X) — `architecture_test.go` parses imports and fails the build if any package violates the import matrix. Tests are cheap and catch regressions at commit time, not at review.
2. **Database package isolation** (Principle II) — only `main.go` imports `database`. Every other package receives `*gorm.DB` or a repository/service interface via constructor. This makes the DB swappable in tests and prevents hidden global state.
3. **Design tokens are the only source of CSS values** (Principle V) — `variables.css` + global classes (`.chip`, `.btn`, `.section-label`) in `main.css`. No raw hex / px values in component CSS.
4. **Per-stack lint + test in CI** (Principle IV + X) — Go: `go vet ./... && go test ./...`. Vue: `npx vue-tsc --build && vitest`. Python: `ruff check && pytest`. Docker's `vue-tsc --build` is stricter than local `vue-tsc --noEmit` — local builds must match Docker.
5. **JWT (access + refresh) + WebAuthn passkeys** (Principle XII) — refresh handled by Axios interceptor with a single-flight 401 queue so concurrent requests don't trigger N refreshes.
6. **Stateless Python agent** (Principle III) — every request carries the full context (provider, model, API key, user data slice). This keeps the agent horizontally scalable and trivially testable without DB fixtures.
7. **AI provider abstraction** — `get_chat_model()` / `get_search_model()` in `app/llm/provider.py` hide Anthropic vs. Ollama differences from agent nodes. Anthropic web search requires explicit `bind_tools` and is only available via `get_search_model()`.
8. **Random gallery via deterministic SQL shuffle** — `((id * seed) + seed) % 2147483647` is SQL-safe across SQLite (no `RANDOM()` per-row instability); seed persisted in `sessionStorage` under `coins:randomSeed` for stable pagination.
9. **Dual idempotency for Coin of the Day** — in-memory `map[userID]string` + DB `HasBeenFeaturedToday` check survives process restarts within the same day.

## Risks & Trade-offs

| Risk                                       | Mitigation                                                                                   |
|--------------------------------------------|----------------------------------------------------------------------------------------------|
| SQLite single-node ceiling                 | Personal scale only; if multi-user demand emerges, migrate via GORM dialect swap.            |
| LangGraph + Python adds a runtime          | Isolated in `src/agent/`; Go API has no Python dependency; agent can be disabled if unused.  |
| PWA offline-write not in v1.0              | Deferred to F006; current v1.0 offline behavior is read-only with graceful write failure.    |
| WebAuthn cross-browser quirks              | DOMException paths explicitly handled in the login view; password remains a fallback.        |
| Provider-specific tool wiring (Anthropic)  | Centralized in `get_search_model()`; agent nodes never call `bind_tools` directly.           |
| Architecture drift over time               | `architecture_test.go` fails the build on violation — prevention rather than review.         |

## Project Structure

```text
specs/001-foundation/
├── spec.md
├── plan.md          # this file
└── tasks.md
```

### Source Code (repository root)

```text
src/
├── api/                              # Go 1.26.1 / Gin
│   ├── main.go                       # ONLY file that imports database/
│   ├── config/
│   ├── database/                     # GORM connection + AutoMigrate
│   ├── models/                       # std lib only
│   ├── repository/                   # all GORM queries; scopes.go
│   ├── services/                     # business logic; agent_proxy.go
│   ├── handlers/                     # thin Gin handlers; Swagger annotations
│   ├── middleware/
│   └── architecture_test.go          # enforces Principle I + X
├── web/                              # Vue 3 / TS / Vite / PWA
│   └── src/
│       ├── api/client.ts             # Axios + JWT interceptor + 401 queue
│       ├── stores/                   # Pinia
│       ├── views/, components/
│       └── assets/{variables,main}.css
└── agent/                            # Python 3.12 / FastAPI / LangGraph
    └── app/
        ├── supervisor.py             # max-iter enforced
        ├── llm/provider.py           # get_chat_model / get_search_model
        ├── teams/                    # 5 team pipelines
        └── models/                   # Pydantic schemas
```

**Structure Decision**: Three-service web app — directories above are the real layout shipped in v1.0.

## Rollout

Already shipped as v1.0. Future updates open new forward-looking specs at `specs/002-*/`, `specs/003-*/`, etc. This folder remains as a historical anchor and is not edited going forward.

## Complexity Tracking

No Constitution violations — section intentionally empty.
