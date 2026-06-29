# Implementation Plan: Wishlist Search Alerts for Acquisition Ideas

**Branch**: `feature/357-wishlist-search-alerts` | **Date**: 2026-06-29 | **Spec**: `C:\Users\brian.denicolafamily\Code\AncientCoins\specs\337-wishlist-search-alerts\spec.md`  
**Input**: Feature specification from `specs/337-wishlist-search-alerts/spec.md`

## Summary

Add collector-owned wishlist search alerts that discover source-backed acquisition candidates without changing the existing wishlist availability-check workflow for already-saved URLs. The Go API remains the owner of persistence, user scoping, alert CRUD, run history, duplicate suppression, candidate lifecycle, and explicit conversion to wishlist items; the Python agent remains stateless and is used only behind the Go service boundary to return provenance-preserving discovery candidates from existing search capabilities. MVP scope is alert CRUD, manual Run Now, in-app candidate review, dismiss/restore, explicit conversion to wishlist, duplicate warnings, cadence metadata, and run history; push/email/digest/scheduled delivery and custom per-dealer scrapers are deferred.

## Technical Context

**Language/Version**: Go 1.26.1 API (`src/api/`), Vue 3 + TypeScript + Pinia + Vite PWA (`src/web/`), Python 3.12 FastAPI/LangGraph agent (`src/agent/`)  
**Primary Dependencies**: Gin, GORM, pure-Go SQLite driver, axios API client, existing `AgentProxy`, existing Python `search/coins` and search tools, Pydantic request/response models, LangGraph/LangChain  
**Storage**: SQLite via GORM AutoMigrate; new Go-owned tables for `WishlistSearchAlert`, `AlertRun`, `AlertCandidate`, `CandidateProvenance`, and `CandidateReviewAction`; existing `Coin` remains wishlist item storage  
**Testing**: Go `go test -v ./...` and `go vet ./...`; Vue `npm run build` / strict `vue-tsc --build`; Python `ruff check app/ tests/` and `pytest tests/ -v` if the agent contract is touched  
**Target Platform**: Self-hosted web service + browser/PWA app + internal Python agent service in the existing Docker/Taskfile topology  
**Project Type**: Multi-service web application (Go REST API, Vue SPA/PWA, Python agent service)  
**Performance Goals**: Manual alert runs return terminal `completed`, `failed`, `rate_limited`, or `partial` status within 30 seconds for normal result volumes; cap v1 persisted review candidates per run; preserve duplicate suppression across repeated runs  
**Constraints**: Preserve service boundaries; no direct Vue-to-agent calls; no Python database access; no custom per-dealer scrapers in v1; safe outbound HTTP validation for public fetches and redirects; internal errors/secrets must not leak; user-owned alert/candidate data must be scoped by `user_id`; search alerts must not write availability history or mutate existing wishlist item status  
**Scale/Scope**: Single-collector/self-hosted usage with per-user/per-alert run limits before scheduling; v1 supports manual runs and in-app review, with cadence metadata only

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Gate | Status | Evidence / Plan |
|------|--------|-----------------|
| I. Clear Layered Architecture | PASS | New Go work follows Handler → Service → Repository → Database. Alert lifecycle, duplicate suppression, and conversion live in services; all GORM queries live in `src/api/repository/`; handlers remain request/response adapters with Swagger annotations. Multi-step conversion uses a transaction. |
| II. Service Boundary Separation | PASS | Go API owns persistence and user scoping. Vue calls only `/api/*`. Python agent remains stateless and returns candidate data/provenance only through `AgentProxy`; no database access or direct UI calls. |
| III. Strict Types and Explicit Contracts | PASS | New Go DTOs, TypeScript interfaces, and Pydantic agent DTOs are explicit. Contracts under `contracts/` define API and agent boundaries. Missing fields use nullable/empty values plus provenance status, not invented data. |
| IV. Simple Complete Changes | PASS | v1 is proportional: CRUD + manual run + in-app review + conversion. Scheduling, push/email/digest, and custom scrapers are explicitly deferred. |
| V. Security/Auth/Privacy | PASS | All alert/candidate endpoints require auth and repository `OwnedBy`/`OwnedByID` scoping. Run errors are generic and non-secret. Source URL/domain validation and safe outbound rules are preserved. |
| VI. Consistent UX | PASS | Wishlist search alerts are a separate wishlist discovery surface, not a modification of the existing availability banner/check flow. UI uses design tokens, lucide icons, and PWA-compatible layouts. |
| VII/IX/§17 Quality Gate | PASS | Plan requires Go unit/handler/repository tests, frontend component/API-client tests, Python contract tests if agent DTOs change, and regression coverage for existing wishlist availability separation. |
| VIII/§19 Documentation | PASS | API contracts, data model, research, and quickstart are generated in the feature directory. No ADR needed for v1 because service-boundary posture is preserved; add ADR only if implementation changes the boundary or introduces scheduler/notification semantics. |

**Initial Gate Result**: PASS — no unjustified constitutional violations and no unresolved NEEDS CLARIFICATION items.

## Project Structure

### Documentation (this feature)

```text
specs/337-wishlist-search-alerts/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   ├── wishlist-search-alerts-api.md
│   └── agent-discovery-contract.md
└── tasks.md             # Future /speckit.tasks output; not created by this plan
```

### Source Code (repository root)

```text
src/api/
├── models/
│   ├── wishlist_search_alert.go        # new alert/run/candidate/provenance/review models
│   └── coin.go                         # existing WishlistItem fields; add traceability FK only if needed
├── repository/
│   ├── wishlist_search_alert_repository.go
│   ├── coin_repository.go              # existing duplicate/wishlist lookups reused/extended
│   └── scopes.go                       # existing ownership scopes reused
├── services/
│   ├── wishlist_search_alert_service.go
│   ├── agent_proxy.go                  # extend with non-streaming discovery endpoint/DTOs
│   └── availability_service.go         # unchanged except regression tests proving separation
├── handlers/
│   └── wishlist_search_alerts.go       # new thin authenticated handler with Swagger annotations
├── database/
│   └── database.go                     # AutoMigrate new models
└── main.go                             # wire repo/service/handler/routes

src/agent/
├── app/models/
│   ├── requests.py                     # add alert discovery request if structured endpoint is added
│   └── responses.py                    # add provenance-preserving candidate response
├── app/routes.py                       # add internal `/api/search/alerts` or equivalent non-streaming endpoint
├── app/teams/coin_search.py            # reuse existing search capability/patterns; no new dealer scrapers
└── app/tools/search.py                 # keep safe outbound validation and existing fetch behavior

src/web/
├── src/api/client.ts                   # typed alert/candidate API functions
├── src/types/index.ts                  # alert/run/candidate/provenance interfaces
├── src/pages/WishlistPage.vue          # add separate discovery entry point or link
├── src/pages/WishlistAlertsPage.vue    # new alert list/review surface
├── src/components/wishlist-alerts/     # alert form, run history, candidate review cards
└── src/router/index.ts                 # route for wishlist alert discovery review
```

**Structure Decision**: Use the existing three-service structure. The Go API gets new model/repository/service/handler layers because it owns persistence and decisions. The Python agent gets only a typed, stateless discovery contract reusing existing search capabilities. The Vue app gets a separate wishlist-alert review surface so acquisition discovery remains distinct from existing wishlist availability checks.

## Existing-Code Findings

- Wishlist availability is currently separate models/history (`AvailabilityRun`, `AvailabilityResult`) and checks only existing wishlist coins with `ReferenceURL` via `AvailabilityService.CheckWishlistForUser()` and `CoinRepository.GetWishlistWithURLs()`.
- Availability scheduling is global/admin-oriented (`AvailabilityScheduler`) and groups existing wishlist URLs by user; it should not be reused for v1 discovery execution except as a future pattern reference.
- Existing availability notifications create in-app/Pushover messages for unavailable saved wishlist URLs. v1 search alerts should not emit push/email/digest notifications; optional future in-app notification should be planned only after MVP candidate review is proven.
- Existing `Coin` fields for wishlist conversion include `Name`, `Denomination`, `Ruler`, `Era`, `Mint`, `Material`, `Grade`, `PurchasePrice`/`CurrentValue`, `PurchaseLocation`, `Notes`, `ReferenceURL`, `ReferenceText`, `IsWishlist`, and listing status fields. Conversion must create a normal wishlist coin with `IsWishlist=true` and source-backed `ReferenceURL`.
- Existing agent proxy has typed non-streaming `CheckAvailability()` and SSE `StreamChat()` to `/api/search/coins`; alert discovery should use a new typed non-streaming proxy method rather than scraping SSE text in the Go API.
- Existing Python search capability already uses dealer search/fetch patterns and safe outbound HTTP validation (`safe_get`, `validate_public_outbound_url`); v1 must reuse these rather than adding custom per-dealer scrapers beyond existing tool behavior.
- Existing frontend wishlist page shows saved wishlist items and a “Check Availability” button. Discovery alerts need a separate entry point and review queue to avoid conflating candidate discovery with availability status.

## Phase 0: Research

Completed in `research.md`.

Decisions resolved:
- Go-owned persistence and lifecycle.
- Typed stateless agent discovery endpoint behind `AgentProxy`.
- Manual Run Now/in-app review as MVP; scheduling and notifications deferred.
- Duplicate suppression key: canonical source URL strongest, plus normalized title + observed price + alert ID.
- Explicit candidate-to-wishlist conversion with review/input for missing fields.

## Phase 1: Design & Contracts

Completed artifacts:
- `data-model.md`
- `contracts/wishlist-search-alerts-api.md`
- `contracts/agent-discovery-contract.md`
- `quickstart.md`

Post-design constitution re-check:

| Gate | Status | Notes |
|------|--------|-------|
| Layered ownership | PASS | Data model separates persistence entities from service decisions; contracts route lifecycle changes through Go services. |
| Service boundary | PASS | Agent contract is request/response and stateless; no persistence or user-scope decisions in Python. |
| Strict contracts | PASS | API and agent contracts define enums, nullable fields, provenance status, and error behavior. |
| Security/privacy | PASS | Owner-scoped endpoints and run limits are explicit; safe outbound HTTP remains required. |
| Scope proportionality | PASS | Deferred scheduler/notification/custom-scraper scope is documented. |

## Complexity Tracking

No constitution violations or exceptional complexity waivers are required.

## Stop Point

Planning stops after Phase 2 artifact generation. Implementation tasks are intentionally not generated here; run `/speckit.tasks` after this plan is accepted.
