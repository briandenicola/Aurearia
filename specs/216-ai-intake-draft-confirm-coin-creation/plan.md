# Implementation Plan: AI Intake Draft + Confirm Coin Creation

**Branch**: `216-ai-intake-draft-confirm-coin-creation` | **Date**: 2026-05-30 | **Spec**: `/specs/216-ai-intake-draft-confirm-coin-creation/spec.md`  
**Input**: Feature specification from `/specs/216-ai-intake-draft-confirm-coin-creation/spec.md`

## Summary

Implement issue #216 as a confirm-gated AI intake flow: Python `coin_intake` generates structured draft payloads with confidence/evidence, Go persists drafts and exposes draft/commit endpoints, and Vue provides review/edit UX where users explicitly confirm before coin creation and AI-tagged journal audit.

## Technical Context

**Language/Version**: Go 1.26.x, TypeScript (Vue 3), Python 3.12  
**Primary Dependencies**: Gin, GORM, SQLite, Vue 3 + Pinia, FastAPI + LangGraph/LangChain  
**Storage**: SQLite (`coin_intake_drafts` new table + existing `coins` and `coin_journals`)  
**Testing**: `go test ./...`, `npm run build`, `ruff check app/ tests/`, `pytest tests/ -v`  
**Target Platform**: Linux-hosted web app + PWA client  
**Project Type**: Web application (Go API + Vue frontend + Python agent service)  
**Performance Goals**: Intake draft generation UX parity with existing AI analysis flows; commit path stays transactional and near existing manual coin-create latency  
**Constraints**: Explicit confirmation required for writes; Go API remains sole writer; authenticated owner scoping enforced; manual add-coin flow remains intact  
**Scale/Scope**: Single feature scope across API, frontend, and agent for draft/review/confirm lifecycle

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Gate | Status | Notes |
|------|--------|-------|
| Principle I (Layered Architecture) | PASS | Handler/service/repository split planned for intake draft + commit. |
| Principle III (Service Boundary Separation) | PASS | Python team generates draft only; Go persists and commits. |
| Principle VII (Schema-Driven Contracts) | PASS | Draft/commit contracts are defined under `contracts/`. |
| Principle XI/XII (Security/Auth) | PASS | Reuse JWT-protected routes and owner scoping for draft/commit actions. |
| Principle XIII (PWA/Mobile Rules) | PASS | Intake UI changes stay within existing Add Coin flow and responsive behavior. |
| §17 Quality Gate | PASS | Feature plan includes Go/Web/Python validations for touched areas. |

## Project Structure

### Documentation (this feature)

```text
specs/216-ai-intake-draft-confirm-coin-creation/
├── plan.md
├── spec.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── intake-flow.openapi.yaml
└── tasks.md
```

### Source Code (repository root)

```text
src/api/
├── models/
│   ├── coin.go
│   ├── coin_journal.go
│   └── coin_intake_draft.go                # new
├── repository/
│   ├── coin_repository.go
│   ├── journal_repository.go
│   └── coin_intake_draft_repository.go     # new
├── services/
│   ├── coin_service.go
│   ├── agent_proxy.go
│   └── coin_intake_service.go              # new
├── handlers/
│   ├── coins.go
│   ├── agent.go
│   ├── swagger_types.go
│   └── coin_intake.go                      # new
├── database/database.go
└── main.go

src/agent/app/
├── models/
│   ├── requests.py
│   └── responses.py
├── teams/
│   └── coin_intake.py                      # new
└── routes.py

src/web/src/
├── api/client.ts
├── types/index.ts
├── composables/useCoinIntake.ts            # new
├── pages/AddCoinPage.vue
└── components/coin/
    └── CoinIntakeReviewPanel.vue           # new
```

**Structure Decision**: Extend existing layered architecture in place with a dedicated intake draft domain and explicit draft/commit boundaries; avoid introducing new cross-service write paths.

## Phase 0 Research (Completed)

`research.md` resolves intake-specific decisions for:

1. Draft persistence + state transitions.
2. Confidence/evidence payload format.
3. Explicit confirmation and idempotent commit behavior.
4. Partial-draft behavior for OCR uncertainty.
5. Journal source tagging contract.

## Phase 1 Design Outputs (Completed)

1. `data-model.md` defines `CoinIntakeDraft`, `IntakeEvidenceItem`, commit request shape, validations, and lifecycle transitions.
2. `contracts/intake-flow.openapi.yaml` defines draft/commit endpoint contracts and schemas.
3. `quickstart.md` defines scenario-driven validation for draft/review/confirm flows and negative cases.
4. Agent context updated via `.specify/scripts/powershell/update-agent-context.ps1 -AgentType copilot`.

## Post-Design Constitution Check

| Gate | Status | Notes |
|------|--------|-------|
| Principle I (Layered Architecture) | PASS | Data-model and structure preserve handler → service → repository boundaries. |
| Principle III (Service Boundary Separation) | PASS | Contracts keep Python stateless and Go as persistence authority. |
| Principle VII (Schema-Driven Contracts) | PASS | OpenAPI contract added for draft/commit and explicit request/response schemas. |
| Principle XI/XII (Security/Auth) | PASS | Contracts require authenticated scope and deny cross-user draft access. |
| Principle XIII (PWA/Mobile Rules) | PASS | Quickstart includes mobile/PWA review/edit/confirm checks. |

## Complexity Tracking

No constitution violations or waivers identified at planning time.
