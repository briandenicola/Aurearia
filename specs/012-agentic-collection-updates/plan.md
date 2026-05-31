# Implementation Plan: Agentic Collection Updates (Epic F012)

**Branch**: `012-agentic-collection-updates` | **Date**: 2026-05-30 | **Spec**: `/specs/012-agentic-collection-updates/spec.md`  
**Input**: Feature specification from `/specs/012-agentic-collection-updates/spec.md`

## Summary

Deliver the four Epic F012 features with one shared architecture spine: a transport-agnostic collection tool layer in the Go API used by both in-app collection chat and external clients.  
Implementation prioritizes data quality first (structured references), then conversational access (read before write), while keeping confirm-gated writes and journal auditability across all AI/external mutation paths.

## Technical Context

**Language/Version**: Go 1.26.x (API), TypeScript/Vue 3 (web), Python 3.12 (agent)  
**Primary Dependencies**: Gin, GORM, SQLite, Vue 3 + Pinia + Axios/fetch SSE, FastAPI + LangGraph/LangChain  
**Storage**: SQLite (existing app DB) with new normalized reference/proposal tables  
**Testing**: `go test ./...`, `go vet ./...`, `npm run build`, `ruff check app/ tests/`, `pytest tests/ -v`  
**Target Platform**: Self-hosted Linux web app (desktop + PWA) with optional external clients  
**Project Type**: Web application (Go API + Vue SPA + Python agent service)  
**Performance Goals**: Collection read tool responses <2s p95 (500 active coins), proposal/commit <2s p95  
**Constraints**: Go API only writer; stateless agent; strict server-side user scoping; confirm gating for all writes  
**Scale/Scope**: 4 features, one shared tool spine, parity between in-app and external adapters

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### Pre-Phase 0 Gate

| Gate | Status | Notes |
|------|--------|-------|
| Principle I (Layered Architecture) | PASS | Plan uses handler → service → repository for new references, proposals, and commits. |
| Principle III (Service Boundary Separation) | PASS | Python agent remains stateless; all writes flow through Go API. |
| Principle VII (Schema-Driven Contracts) | PASS | Contract artifact defined for collection tool operations and payload schemas. |
| Principle XI/XII (Security/Auth) | PASS | API-key/user context server-scoped; no client-supplied user override for tool calls. |
| §17 Quality Gate | PASS | Validation commands identified and included in quickstart. |

### Post-Phase 1 Re-check

| Gate | Status | Notes |
|------|--------|-------|
| Principle I / X (Layering + enforcement) | PASS | Data model and structure keep business logic in services/repositories. |
| Principle III / VI (service and AI isolation) | PASS | Tool layer in Go; Python returns structured outputs only. |
| Principle VII (contracts) | PASS | OpenAPI contract defines read/propose/commit and intake schemas. |
| Principle XI/XII (security + token policy) | PASS | Proposal-token commit flow and API-key scope constraints are explicitly modeled. |
| §17 / §21 (Quality + DoD readiness) | PASS | Phase plan includes lint/build/test gates for touched services. |

## Phase 0 Research Summary

Resolved decisions documented in `research.md`:
- External access protocol: OpenAPI-first with MCP-compatible adapter target
- Existing chat drawer as collection mode entry point
- Confidence model as coarse buckets (`high|medium|low`) with optional numeric score
- Protocol-enforced two-phase writes for chat/external paths
- Field allowlist for v1 conversational/external updates
- Structured reference validation model and migration approach

No unresolved `NEEDS CLARIFICATION` items remain.

## Phase 1 Design Summary

- `data-model.md` defines normalized reference model, intake draft, tool request/response, proposal/commit entities, and transitions.
- `contracts/collection-tools.openapi.yaml` defines API contracts for references, intake, chat tools, and external tool-server parity.
- `quickstart.md` captures sequencing and validation.
- `issues.md` contains 4 discrete tracking issues aligned to epic member cards.

## Project Structure

### Documentation (this feature)

```text
specs/012-agentic-collection-updates/
├── spec.md
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── issues.md
└── contracts/
    └── collection-tools.openapi.yaml
```

### Source Code (repository root)

```text
src/api/
├── models/
│   ├── coin_reference.go                    # new
│   ├── coin_intake_draft.go                 # new
│   └── collection_update_proposal.go        # new
├── repository/
│   ├── coin_reference_repository.go          # new
│   ├── collection_tools_repository.go        # new
│   └── collection_update_repository.go       # new
├── services/
│   ├── coin_intake_service.go                # new
│   ├── collection_tools_service.go           # new transport-agnostic tool layer
│   └── collection_update_service.go          # new propose/commit logic
├── handlers/
│   ├── coin_references.go                    # new
│   ├── coin_intake.go                        # new
│   ├── collection_chat.go                    # new mode/endpoints
│   └── external_collection_tools.go          # new external adapter endpoints
├── services/agent_proxy.go                   # extend proxy request schemas for intake/chat mode
├── database/database.go                      # AutoMigrate for new models
└── main.go                                   # route wiring and auth scopes

src/agent/
└── app/teams/
    ├── coin_intake.py                        # new team
    └── collection_chat.py                    # new team path, tool-calling orchestration

src/web/src/
├── api/client.ts                             # intake/references/chat tool APIs
├── components/coin/                          # intake draft review UI
├── components/agent/                         # collection mode UX
└── pages/                                    # collection/edit pages for references + intake flow
```

**Structure Decision**: Keep existing three-service architecture and implement one canonical Go tool layer consumed by in-app and external adapters.

## Complexity Tracking

No constitution violations or waivers required at planning stage.
