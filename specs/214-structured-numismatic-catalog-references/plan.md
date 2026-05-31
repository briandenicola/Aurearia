# Implementation Plan: Structured Numismatic Catalog References

**Branch**: `214-structured-numismatic-catalog-references` | **Date**: 2026-05-30 | **Spec**: `/specs/214-structured-numismatic-catalog-references/spec.md`  
**Input**: Feature specification from `/specs/214-structured-numismatic-catalog-references/spec.md`

## Summary

Implement structured coin attribution as first-class data (`CoinReference`) with registry-driven validation and era-aware browsing, then wire references into AI discovery and export surfaces while preserving backward compatibility with legacy free-text reference fields.

## Technical Context

**Language/Version**: Go 1.26.x, TypeScript (Vue 3), Python 3.12  
**Primary Dependencies**: Gin, GORM, SQLite, Vue 3 + Pinia, FastAPI/LangGraph  
**Storage**: SQLite (new `coin_references` + `catalog_registry` + `coins.era`)  
**Testing**: `go test ./...`, `npm run build`, `pytest tests/ -v`  
**Target Platform**: Linux-hosted web app + PWA  
**Project Type**: Web application (Go API + Vue frontend + Python agent)  
**Performance Goals**: Reference CRUD and filtering parity with existing coin edit/list performance (sub-second local queries)  
**Constraints**: Preserve existing API/UI compatibility; validation driven by registry data; no direct DB access from Python service  
**Scale/Scope**: Single feature scope, touching API + web + agent + export paths

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Gate | Status | Notes |
|------|--------|-------|
| Principle I (Layered Architecture) | PASS | New domain logic planned in service/repository layers. |
| Principle III (Service Boundary Separation) | PASS | Python agent emits structured suggestions only; Go persists. |
| Principle V (Design Consistency) | PASS | UI additions stay inside existing token/class system. |
| Principle VII (Schema-Driven Contracts) | PASS | New API contracts and DTOs are explicitly defined. |
| Principle XI/XII (Security/Auth) | PASS | Existing auth and user scoping are reused for new endpoints. |

## Project Structure

### Documentation (this feature)

```text
specs/214-structured-numismatic-catalog-references/
├── plan.md
├── spec.md
└── tasks.md
```

### Source Code (repository root)

```text
src/api/
├── models/
│   ├── coin.go
│   ├── coin_reference.go                # new
│   └── catalog_registry.go              # new
├── repository/
│   ├── coin_repository.go
│   ├── coin_reference_repository.go     # new
│   └── catalog_registry_repository.go   # new
├── services/
│   ├── coin_service.go
│   ├── coin_reference_service.go        # new
│   └── export_service.go
├── handlers/
│   ├── coins.go
│   └── swagger_types.go
├── database/database.go
└── main.go

src/web/src/
├── api/client.ts
├── pages/CoinDetailPage.vue
├── pages/CollectionPage.vue
└── components/coin/

src/agent/app/
└── teams/coin_analysis/
```

**Structure Decision**: Extend existing layered architecture in-place with dedicated reference/registry components and minimal surface-area changes to current coin flows.

## Complexity Tracking

No constitution violations or waivers identified at planning time.
