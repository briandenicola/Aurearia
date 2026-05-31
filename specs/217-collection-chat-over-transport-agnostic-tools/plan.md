# Implementation Plan: Collection Chat Over Transport-Agnostic Tools

**Branch**: `217-collection-chat-over-transport-agnostic-tools` | **Date**: 2026-05-31 | **Spec**: `/specs/217-collection-chat-over-transport-agnostic-tools/spec.md`  
**Input**: Feature specification from `/specs/217-collection-chat-over-transport-agnostic-tools/spec.md`

## Summary

Implement issue #217 by introducing a transport-agnostic collection tool layer in Go for read/query/aggregate and confirm-gated updates, then keep the existing app-wide chat flow and route each prompt by intent to collection-aware or search behavior through shared contracts.

## Technical Context

**Language/Version**: Go 1.26.x, TypeScript (Vue 3), Python 3.12  
**Primary Dependencies**: Gin, GORM, SQLite, Vue 3 + Pinia, FastAPI + LangGraph/LangChain  
**Storage**: SQLite (new `collection_update_proposals` table; existing `coins` and `coin_journals`)  
**Testing**: `go test ./...`, `npm run build`, `ruff check app/ tests/`, `pytest tests/ -v`  
**Target Platform**: Linux-hosted web app + PWA client  
**Project Type**: Web application (Go API + Vue frontend + Python agent service)  
**Performance Goals**: Collection read queries and proposal creation remain within current chat-response expectations; commit path remains near existing single-coin update latency  
**Constraints**: Server-side user scoping only; allowlisted write fields; two-phase token-gated writes; no direct Python DB access; preserve existing app-wide chat UX with no manual mode switch  
**Scale/Scope**: Single feature scope across API, frontend, and agent routing/contracts

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Gate | Status | Notes |
|------|--------|-------|
| Principle I (Layered Architecture) | PASS | Tool logic planned in Go service/repository layers; handlers remain thin. |
| Principle III (Service Boundary Separation) | PASS | Python remains stateless; Go stays persistence and auth boundary. |
| Principle VII (Schema-Driven Contracts) | PASS | Intent-routed chat + proposal commit contracts are explicitly documented. |
| Principle XI/XII (Security/Auth) | PASS | JWT user context and owner scoping govern all tool operations and commits. |
| Principle XIII (PWA/Mobile Rules) | PASS | Existing drawer UX is extended rather than replaced. |
| §17 Quality Gate | PASS | Planned validation spans Go, web, and Python surfaces touched by feature. |

## Project Structure

### Documentation (this feature)

```text
specs/217-collection-chat-over-transport-agnostic-tools/
├── plan.md
├── spec.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── collection-chat.openapi.yaml
└── tasks.md
```

### Source Code (repository root)

```text
src/api/
├── models/
│   └── collection_update_proposal.go          # new
├── repository/
│   ├── coin_repository.go
│   ├── journal_repository.go
│   └── collection_update_repository.go        # new
├── services/
│   ├── collection_tools_service.go            # new
│   └── agent_proxy.go
├── handlers/
│   ├── agent.go
│   └── swagger_types.go
├── database/database.go
└── main.go

src/agent/app/
├── models/
│   ├── requests.py
│   └── responses.py
├── teams/
│   └── collection_chat.py                     # new
├── supervisor.py
└── routes.py

src/web/src/
├── api/client.ts
├── types/index.ts
├── composables/useCoinSearchChat.ts
└── components/
    └── CoinSearchChat.vue
```

**Structure Decision**: Reuse existing chat entrypoint and SSE UI while adding a dedicated Go tool service plus proposal persistence for two-phase updates, so future external adapters can reuse the same core operations.

## Phase 0 Research (Completed)

`research.md` resolves feature decisions for:

1. Go-owned transport-agnostic tool layer placement.
2. Prompt-intent routing in existing chat endpoint/surface.
3. Proposal-token two-phase write protocol.
4. Ambiguous target disambiguation behavior.
5. Audit journaling strategy with `collection_chat` source tagging.

## Phase 1 Design Outputs (Completed)

1. `data-model.md` defines internal request/result envelopes, proposal persistence, commit payload, and lifecycle constraints.
2. `contracts/collection-chat.openapi.yaml` defines intent-routed chat and proposal commit/cancel contracts.
3. `quickstart.md` defines end-to-end read/write validation and negative safety scenarios.
4. Agent context updated via `.specify/scripts/powershell/update-agent-context.ps1 -AgentType copilot`.

## Post-Design Constitution Check

| Gate | Status | Notes |
|------|--------|-------|
| Principle I (Layered Architecture) | PASS | Design keeps write orchestration in services with transactional repository boundaries. |
| Principle III (Service Boundary Separation) | PASS | Contracts keep Python routing/tooling stateless and Go as sole writer. |
| Principle VII (Schema-Driven Contracts) | PASS | OpenAPI contract covers intent-routed chat and proposal lifecycle endpoints. |
| Principle XI/XII (Security/Auth) | PASS | Proposal ownership, token checks, and authenticated scoping enforced in design. |
| Principle XIII (PWA/Mobile Rules) | PASS | Existing responsive drawer flow remains the user interaction surface. |

## Complexity Tracking

No constitution violations or waivers identified at planning time.
