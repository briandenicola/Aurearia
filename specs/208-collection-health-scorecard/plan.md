# Implementation Plan: Collection Health Scorecard (v1)

**Branch**: `208-collection-health-scorecard` | **Date**: 2026-05-30 | **Spec**: `/specs/208-collection-health-scorecard/spec.md`  
**Input**: Feature specification from `/specs/208-collection-health-scorecard/spec.md`

## Summary

Implement issue #208 by adding deterministic health scoring for active coins and collections, exposing scorecard + queue + trend + admin aggregate APIs, and wiring Vue UI surfaces to those APIs.  
Scoring uses fixed v1 weights (metadata 40%, image coverage 20%, valuation freshness 20%, AI coverage 20%) with fixed grade bands (A/B/C/D/F).  
Daily snapshots persist collection health to support 30-day trend deltas.

## Technical Context

**Language/Version**: Go 1.26.3 (API), TypeScript 5.9 + Vue 3 (frontend), Python 3.12 agent (unchanged for this feature)  
**Primary Dependencies**: Gin, GORM, SQLite driver (`glebarez/sqlite`), Vue 3, Pinia, Axios  
**Storage**: SQLite (existing app DB) + new persisted `collection_health_snapshots` table  
**Testing**: `go test ./...`, `go vet ./...`, `npm run build`, `npm run lint`, existing API/web unit & integration tests  
**Target Platform**: Self-hosted Linux web app (desktop + PWA mobile clients)  
**Project Type**: Web application (Go API + Vue SPA + Python agent service boundary preserved)  
**Performance Goals**: Dashboard health score p95 < 1.5s (500 coins), Needs Attention queue page p95 < 2s (25 items, 500 coins)  
**Constraints**: Deterministic score math; fixed v1 weights/thresholds; active coins only (`is_wishlist=false`, `is_sold=false`); admin metrics admin-only  
**Scale/Scope**: Per-user scoring up to 500 active coins; admin aggregate across all active coins; daily snapshots for all users with active coins

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### Pre-Phase 0 Gate

| Gate | Status | Notes |
|------|--------|-------|
| Principle I (Layered Architecture) | PASS | Plan uses handler → service → repository; no business logic in handlers. |
| Principle III (Service Boundaries) | PASS | Health scoring remains in Go API; no direct DB access from Python agent or frontend. |
| Principle VII (Schema-Driven Contracts) | PASS | Contract artifact defined under `specs/.../contracts/`. |
| Principle X (Architecture Enforcement) | PASS | Planned changes stay within allowed import boundaries. |
| §17 Quality Gate | PASS | Validation commands identified in quickstart and implementation plan. |

### Post-Phase 1 Re-check

| Gate | Status | Notes |
|------|--------|-------|
| Principle I (Layered Architecture) | PASS | Data model + contract imply repository/service abstractions before handlers. |
| Principle VII (Schema-Driven Contracts) | PASS | OpenAPI contract includes all external endpoints and auth constraints. |
| Principle XI (Security Hardening) | PASS | Admin aggregate endpoint explicitly admin-gated; no new unsafe raw SQL patterns planned. |
| §17 / §21 Quality & DoD readiness | PASS | Test/build/lint expectations remain satisfied by planned scope. |

## Phase 0 Research Summary

Resolved decisions documented in `research.md`:
- Fixed weighted scoring model and grade thresholds
- Deterministic missing checklist taxonomy
- Bucketed valuation freshness policy
- Daily health snapshot persistence strategy
- Needs Attention queue ordering + quick action mapping
- Admin aggregate metric definitions

No unresolved `NEEDS CLARIFICATION` items remain.

## Phase 1 Design Summary

- `data-model.md` defines computed projections and persisted snapshot entity.
- `contracts/health-scorecard.openapi.yaml` defines user/admin APIs and payload schemas.
- `quickstart.md` defines implementation sequence and validation checklist.

## Project Structure

### Documentation (this feature)

```text
specs/208-collection-health-scorecard/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── health-scorecard.openapi.yaml
└── tasks.md                    # created in /speckit.tasks phase
```

### Source Code (repository root)

```text
src/api/
├── models/
│   └── collection_health_snapshot.go           # new
├── repository/
│   └── health_repository.go                    # new
├── services/
│   ├── health_service.go                       # new
│   └── collection_health_scheduler.go          # new/extended scheduler integration
├── handlers/
│   ├── health.go                               # new user health endpoints
│   └── admin_health.go                         # new/extended admin health endpoint
├── database/database.go                        # AutoMigrate registration
└── main.go                                     # route wiring

src/web/src/
├── api/client.ts                               # new health API clients
├── types/index.ts                              # health response types
├── stores/coins.ts                             # health state/actions
├── pages/StatsPage.vue                         # collection health scorecard + trend
├── pages/CollectionPage.vue                    # needs-attention queue integration
├── pages/CoinDetailPage.vue                    # per-coin health checklist
├── pages/AdminPage.vue                         # admin aggregate panel
└── components/
    ├── health/*                                # new scorecard/queue components
    └── admin/*                                 # admin panel health widget
```

**Structure Decision**: Use the existing web-application split (`src/api` + `src/web`). Health computation and persistence live in API layers; SPA consumes typed API responses via `api/client.ts` and Pinia.

## Complexity Tracking

No constitution violations anticipated at planning stage.
