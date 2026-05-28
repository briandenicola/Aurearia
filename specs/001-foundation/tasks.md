---
description: "Retroactive task list for v1.0 Foundation — every item already shipped in production code as of v1.0."
---

# Tasks: v1.0 Foundation — Ancient Coins PWA

**Input**: Design documents from `specs/001-foundation/` (spec.md, plan.md)
**Status**: SHIPPED — retroactive checklist; every box is checked because the corresponding code is in production.

## Format

`- [x] <task>` — shipped in v1.0.
Tasks are grouped by **domain** (not user story) because this is a retroactive snapshot of the v1.0 surface, not a forward-looking plan.

---

## Go API (Cassius)

- [x] Layered architecture (Handler → Service → Repository → Database) — Principle I
- [x] `architecture_test.go` enforcing import rules — Principle X
- [x] Models in `src/api/models/` + `AutoMigrate` in `database/database.go`
- [x] Auth: JWT issuance, refresh-token rotation, WebAuthn passkey register + login — Principle XII
- [x] Coin CRUD endpoints with Swagger annotations on every public handler
- [x] Random gallery sort with `?sort=random&seed=N` (deterministic SQL shuffle, `strconv.Atoi`-validated seed)
- [x] Featured-coin / Coin-of-the-Day endpoints (`GET /featured-coins/latest`, `GET /featured-coins/:id`, `POST /admin/coin-of-day/run`) + daily scheduler with dual idempotency
- [x] Agent proxy SSE streaming via `services/agent_proxy.go` (Go API contains zero LLM logic — Principle III)
- [x] Admin settings: key-value `AppSetting` model with constants + defaults in `services/settings_service.go`
- [x] Public read-only gallery endpoints for coins flagged public
- [x] `GET /ai-status` provider-agnostic availability probe

## Vue Frontend (Aurelia)

- [x] Vue 3 + TypeScript + Pinia + Vite + vite-plugin-pwa scaffold
- [x] Axios client (`src/web/src/api/client.ts`) with JWT interceptor and single-flight 401 refresh queue
- [x] `sanitizeCoin()` normalization (`''`/`undefined` → `null`) before write requests
- [x] Coin list, detail, add, edit views
- [x] Filtering, sorting, random gallery with `sessionStorage` seed (`coins:randomSeed`) for stable pagination
- [x] Agent chat streaming via `fetch` + manual SSE parsing (not Axios)
- [x] AI analysis modal rendering structured Pydantic-conformant results
- [x] Design token system: `variables.css` tokens + `main.css` global classes (`.chip`, `.btn`, `.section-label`) — Principle V
- [x] WebAuthn biometric login flow with `DOMException` handling for user-canceled prompt
- [x] PWA install prompt + offline-read service worker (offline write deferred to F006)
- [x] Lucide icons (`lucide-vue-next`) — no emojis in UI text

## Python Agent (Maximus + Cassius)

- [x] FastAPI service on port 8081 with LangGraph `StateGraph` per pipeline
- [x] Five team pipelines: Coin Search, Coin Shows, Coin Analysis (vision), Portfolio Review, Availability Check
- [x] Stateless per-request configuration (API keys, model, prompts, user context passed by Go API) — Principle III
- [x] AI provider abstraction in `app/llm/provider.py` (`get_chat_model()` / `get_search_model()`) covering Anthropic + Ollama
- [x] Web search via Anthropic built-in `web_search_20250305` (bound through `get_search_model`) + SearXNG ReAct fallback for Ollama
- [x] Pydantic schemas for every worker output — no free-form text (Principle VI)
- [x] Verification agents confirm every URL is live and every date is in the future (Principle VI)
- [x] `app/supervisor.py` enforces max-iteration cap to prevent loops
- [x] Structured logging via `app/logging_config.py` (ring buffer + stdout)

## Quality / Tooling (Brutus)

- [x] `go test ./...` including architecture tests — Principle X
- [x] `vitest` + `npx vue-tsc --build` (Docker-parity strict typecheck) — Principle IV
- [x] `pytest` + `ruff check app/ tests/` for agent
- [x] `Taskfile.yml` with `task --list` discovery (`task build`, `task test`, `task up`, `task up-all`, `task test-agent`, `task lint-agent`)
- [x] CI workflow at `.github/workflows/ci.yml`
- [x] Docker publish workflows (stable + beta channels)
- [x] Multi-stage Dockerfile producing two containers: app (Go+Vue) + agent (Python)

## Governance / DevEx (Maximus + Scribe)

- [x] Conventional Commits enforced; `Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>` trailer on AI-assisted commits — Principle VIII
- [x] Constitution v1.0 → v2.0 ratified (Phase 1 of alignment; added §0 Hierarchy + §17–§23)
- [x] Squad system configured at `.squad/` (agents, decisions, log, skills)
- [x] SpecKit installed at `.specify/templates/` (spec / plan / tasks templates + slash prompts)
- [x] `specs/_backlog/` seeded with F001–F007 cards
- [x] `specs/001-foundation/` retroactive anchor created (this folder) — satisfies Constitution §0 Hierarchy item 3

---

## Dependencies & Execution Order

Not applicable — retroactive. All items shipped before this checklist was authored.

## Status

**SHIPPED** — v1.0 in production as of 2026-05-28. This task list is a backward-looking audit, not forward work.

**Notes**: Retroactive — every item already exists in production code as of v1.0. Forward-looking work opens new specs at `specs/002-*/` and onward; this folder is not edited again except to add a `## History` entry if a future amendment materially restates the v1.0 surface.
