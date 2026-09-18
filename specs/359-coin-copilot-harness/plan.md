# Implementation Plan: Coin Copilot Read-Only Harness

**Branch**: `359-coin-copilot-harness` | **Date**: 2026-09-17 | **Spec**: `/specs/359-coin-copilot-harness/spec.md`  
**Input**: GitHub issue #721 plus the locked MVP scope in `spec.md`

## Summary

Add a default-off Coin Copilot mode to the existing app-wide chat drawer. Go
creates and owns durable threads, runs, checkpoints, events, idempotency, worker
execution, cancellation, retention, and authorization. Each execution or resume
calls a stateless Python LangGraph harness with a fresh, narrowly scoped
credential. Python may sequence only owner-scoped collection reads, portfolio
review, and read-only gap analysis. Go validates and persists typed events before
Vue replays or follows them over SSE. The legacy supervisor remains untouched as
the fallback.

## Technical Context

**Language/Version**: Go 1.26.x, Python 3.12, Vue 3 / TypeScript  
**Primary Dependencies**: Gin, GORM, SQLite, FastAPI, LangGraph/LangChain, Vue 3  
**Storage**: SQLite; five additive Go-owned orchestration tables  
**Testing**: `go test ./...`, `go vet ./...`, `ruff check app/ tests/`,
`pytest tests/ -v`, `npm run test:unit`, `npm run build`  
**Target Platform**: Existing two-container self-hosted web/PWA deployment  
**Project Type**: Go API + Vue SPA + stateless Python agent service  
**Performance Goals**: run accepted within 1 second excluding queueing; first
persisted progress event within 5 seconds under normal local conditions; hard
execution ceiling 120 seconds by default and 150 seconds maximum
**Constraints**: read-only MVP; one tool at a time; no Python DB access; no
chain-of-thought; owner scoping; replayable SSE; legacy fallback; dollar-cost
enforcement deferred until a trustworthy pricing source exists; reliable
provider-reported token usage remains observable; iteration, tool-call,
wall-clock, sequential-concurrency, and payload limits remain enforced
**Scale/Scope**: personal-scale single node, one active run per owner by default,
queue depth 16

## Constitution Check

| Gate | Status | Design evidence |
|---|---|---|
| Principle I — layered architecture | PASS | Public/internal handlers call `CoinCopilotService`; all SQLite queries live in a repository; wiring remains in `deps.go`/route registrars. |
| Principle II — service boundaries | PASS | Go owns persistence/auth/SSE; Python is stateless and DB-free; Vue calls Go only. |
| Principle III — explicit contracts | PASS | Public OpenAPI, internal execution contract, strict Pydantic models, typed Go DTOs, and TS discriminated unions are planned. |
| Principle IV — simple complete changes | PASS | Reuses the existing drawer, collection service, proxy patterns, and deep-job durable SSE pattern; broad issue scope is deliberately deferred. |
| Principle V — security/privacy | PASS | Owner scoping, read-only allowlist, fresh execution credentials, size limits, redaction, no existence leaks, and tamper tests are specified. |
| Principle VI — consistent UX | PASS | Existing responsive chat drawer remains the entry point; progress/control additions reuse design tokens/classes. |
| Principle VII — release integrity | PASS | No new third-party dependency is required; generated API docs and all language gates are explicit tasks. |
| Principle VIII — documented decisions | PASS | ADR 0016 and decision inbox entry record durable-state ownership and amendment analysis. |
| Principle IX / §17 | PASS | Contract, race, tamper, fallback, replay, and exact workflow regression tests are required before completion. |
| §21 Definition of Done | PASS | Tasks include migrations, Swagger/OpenAPI sync, tests, docs, privacy checks, and full gates. |

No constitution amendment is required: Principle II already requires Python to
remain stateless and Go to own persistence. ADR 0016 records how resumability is
implemented within that rule.

## Project Structure

### Documentation

```text
specs/359-coin-copilot-harness/
├── spec.md
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── tasks.md
└── contracts/
    ├── coin-copilot.openapi.yaml
    ├── agent-internal-contract.md
    └── sse-events.md

docs/adr/
└── 0016-go-owned-durable-coin-copilot-state.md
```

### Go API

```text
src/api/
├── models/
│   └── coin_copilot.go
├── repository/
│   ├── coin_copilot_repository.go
│   └── coin_copilot_repository_test.go
├── services/
│   ├── coin_copilot_service.go
│   ├── coin_copilot_worker.go
│   ├── coin_copilot_broker.go
│   ├── coin_copilot_contract.go
│   ├── coin_copilot_service_test.go
│   ├── coin_copilot_worker_test.go
│   ├── coin_copilot_contract_test.go
│   ├── internal_token_service.go
│   ├── internal_token_service_test.go
│   ├── agent_proxy.go
│   └── settings_service.go
├── handlers/
│   ├── coin_copilot.go
│   ├── coin_copilot_sse.go
│   ├── coin_copilot_test.go
│   ├── coin_copilot_sse_test.go
│   ├── internal_tools.go
│   └── internal_tools_test.go
├── database/database.go
├── deps.go
├── routes_protected.go
└── routes_internal.go
```

### Python agent

```text
src/agent/
├── app/
│   ├── models/
│   │   ├── requests.py
│   │   └── responses.py
│   ├── teams/
│   │   └── coin_copilot.py
│   ├── tools/
│   │   └── copilot_collection_tools.py
│   ├── llm/
│   │   └── capabilities.py
│   └── routes.py
└── tests/
    ├── fixtures/coin_copilot/
    ├── test_coin_copilot_contract.py
    ├── test_coin_copilot_harness.py
    ├── test_coin_copilot_capabilities.py
    └── test_coin_copilot_security.py
```

### Vue frontend

```text
src/web/src/
├── api/endpoints/agent.ts
├── types/agent.ts
├── composables/
│   ├── useCoinCopilot.ts
│   └── __tests__/useCoinCopilot.test.ts
├── components/
│   ├── CoinSearchChat.vue
│   ├── chat/CopilotRunProgress.vue
│   ├── chat/CopilotClarificationCard.vue
│   └── __tests__/CoinSearchChat.test.ts
├── components/admin/AdminSystemSection.vue
└── pages/AdminPage.vue
```

**Structure Decision**: The Go service is the orchestration authority and
worker host. Python contains only the bounded planning/tool loop. Vue adds
durable-run rendering to the existing drawer. No third service, queue product,
or Python persistence layer is introduced.

## Architecture Flow

```text
Vue existing drawer
  ├─ capability says legacy ──> POST /api/agent/chat ──> existing supervisor
  └─ capability says copilot
       ├─ POST /api/agent/copilot/runs
       ├─ GET/DELETE /api/agent/copilot/threads/{id}
       ├─ GET  /api/agent/copilot/runs/{id}
       ├─ GET  /api/agent/copilot/runs/{id}/events
       ├─ POST /api/agent/copilot/runs/{id}/cancel
       └─ POST /api/agent/copilot/runs/{id}/resume
              │
              v
        Go CoinCopilotService/Repository
              │ durable queue claim + fresh execution token
              v
        Python POST /api/copilot/execute (stateless SSE)
              │
              ├─ read-only callbacks to /api/internal/copilot/tools/*
              │      └─ Go CollectionToolsService → CoinRepository → SQLite
              └─ typed frames
                     └─ Go validate → checkpoint/event transaction → Vue SSE
```

## Implementation Phases

### Phase 0 — Contract fixtures and settings

Create shared JSON fixtures and settings validation first. This locks public
event names, internal frame shapes, enforceable limits, token-usage counters,
and safe defaults before runtime code diverges. No estimated-cost field or
admin cost setting is part of the MVP contract.

### Phase 1 — Go durable core

Add models, additive migration/indexes, owner-scoped repository methods,
transactional sequence/checkpoint helpers, state machine, idempotent start and
resume, cancellation, stale recovery, retention, broker, and workers.

### Phase 2 — Least-privilege internal boundary

Add the distinct execution-token family and read-only callback group. Reuse
`CollectionToolsService`; do not register proposal/commit endpoints beneath
the Copilot prefix.

### Phase 3 — Stateless Python harness

Implement strict request/frame schemas and a bounded sequential LangGraph
harness. Wrap the current read tools and create read-only portfolio/gap
capabilities. Reject unknown/out-of-scope tools and never serialize private
reasoning.

### Phase 4 — Go/Python execution bridge

Extend `AgentProxy`, validate every Python frame, persist checkpoints/events,
enforce budgets, revoke credentials, and settle state races.

### Phase 5 — Existing drawer integration

Add capability selection, durable run lifecycle, replayable SSE, progress,
clarification, resume, and cancel to `CoinSearchChat.vue` through a focused
composable. Preserve all legacy chat behavior and saved conversations.

### Phase 6 — Admin rollout, docs, and full quality gate

Add the default-off setting and bounded controls to the existing admin system
section, regenerate Swagger/OpenAPI, update feature/API/testing docs, execute
quickstart and full §17 gates.

## Test Strategy

- **Go unit/repository**: transition matrix, SQL owner scopes, idempotency,
  sequence allocation, retention, stale recovery, cancellation races, size
  caps, redaction, and credential tamper cases.
- **Go integration**: fake Python stream and real SQLite verify
  frame→checkpoint/event atomicity and restart recovery.
- **Python unit/contract**: strict schemas, bounded loop, tool allowlist,
  prompt-injection resistance, malformed model calls, provider capability
  checks, no database imports.
- **Vue unit/component**: capability fallback, stream replay/de-duplication,
  progress, clarification, cancel/resume, terminal close, desktop/PWA layout,
  and legacy behavior.
- **Contract drift**: both Go and Python consume the same committed JSON
  fixtures; OpenAPI is linted and generated Swagger remains synchronized.

## Complexity Tracking

No constitution violation or waiver is present. Four durable domain concepts
plus one idempotency record are necessary to separate user conversation,
run lifecycle, immutable continuation state, replay events, and resume replay.
Collapsing them into the legacy conversation JSON blob was rejected because it
cannot provide atomic transitions or replay guarantees.
