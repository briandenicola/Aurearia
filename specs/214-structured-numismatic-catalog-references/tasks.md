---
description: "Task breakdown for issue #214 structured numismatic references"
---

# Tasks: Structured Numismatic Catalog References

**Input**: Design documents from `specs/214-structured-numismatic-catalog-references/`  
**Prerequisites**: `plan.md`, `spec.md`

**Tests**: Not explicitly requested in the issue/spec; implementation tasks are structured for independent manual validation per story.

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Create implementation scaffolding for reference + catalog domains.

- [x] T001 Create structured reference model scaffold in `src/api/models/coin_reference.go`
- [x] T002 Create catalog registry model scaffold in `src/api/models/catalog_registry.go`
- [x] T003 [P] Create reference repository scaffold in `src/api/repository/coin_reference_repository.go`
- [x] T004 [P] Create catalog registry repository scaffold in `src/api/repository/catalog_registry_repository.go`
- [x] T005 [P] Create reference service scaffold in `src/api/services/coin_reference_service.go`
- [x] T006 [P] Create reference handler scaffold in `src/api/handlers/coin_references.go`
- [x] T007 [P] Add frontend placeholder types for references and eras in `src/web/src/types/index.ts`
- [x] T008 [P] Add frontend API client placeholders for reference endpoints in `src/web/src/api/client.ts`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core persistence/validation infrastructure required before user stories.

**⚠️ CRITICAL**: Complete this phase before starting Phase 3+.

- [x] T009 Implement `CoinReference` fields + `Coin.References` association in `src/api/models/coin_reference.go` and `src/api/models/coin.go`
- [x] T010 Implement `CatalogRegistry` fields for era + volume-required rules in `src/api/models/catalog_registry.go`
- [x] T011 Register `CoinReference` and `CatalogRegistry` migrations in `src/api/database/database.go`
- [x] T012 Implement registry seed bootstrap (RIC, RPC, Sear, Crawford, SNG, Spink, Duplessy, CNI, KM, Y, Craig, RedBook) in `src/api/database/database.go`
- [x] T013 [P] Implement repository CRUD/query operations for references in `src/api/repository/coin_reference_repository.go`
- [x] T014 [P] Implement repository lookups for catalog rules in `src/api/repository/catalog_registry_repository.go`
- [x] T015 Implement validation service (volume-required, string number qualifiers, dedupe) in `src/api/services/coin_reference_service.go`
- [x] T016 [P] Add era filter + reference preload support in coin queries in `src/api/repository/coin_repository.go`
- [x] T017 Add shared reference DTO types for Swagger in `src/api/handlers/swagger_types.go`

**Checkpoint**: Reference persistence, registry validation, and query foundations are ready.

---

## Phase 3: User Story 1 - Persist and validate structured references (Priority: P1) 🎯 MVP

**Goal**: Deliver backend reference CRUD with registry-driven validation and era normalization.

**Independent Test**: Save coins with mixed catalogs and multiple references via API; verify volume-required catalogs reject missing volume and valid references persist/reload.

### Implementation for User Story 1

- [x] T018 [US1] Implement authenticated list/create endpoints for coin references in `src/api/handlers/coin_references.go`
- [x] T019 [US1] Implement authenticated update/delete endpoints for coin references in `src/api/handlers/coin_references.go`
- [x] T020 [US1] Register `/api/coins/:id/references` routes in `src/api/main.go`
- [x] T021 [US1] Enforce allowed era values (`ancient|medieval|modern`) in coin create/update binding in `src/api/handlers/coins.go`
- [x] T022 [US1] Wire reference validation + persistence orchestration into coin workflows in `src/api/services/coin_service.go`
- [x] T023 [US1] Ensure single-coin and list payloads preload `References` in `src/api/repository/coin_repository.go`
- [x] T024 [US1] Document structured reference endpoints and schemas in `src/api/docs/swagger.yaml`, `src/api/docs/swagger.json`, and `src/api/docs/docs.go`

**Checkpoint**: User Story 1 is independently functional and testable.

---

## Phase 4: User Story 2 - Manage and browse references in UI (Priority: P1)

**Goal**: Enable collectors to manage references in coin detail and filter collection by era in desktop/PWA.

**Independent Test**: Add/edit/remove references from Coin Detail and filter collection by era in both desktop and PWA header flows.

### Implementation for User Story 2

- [x] T025 [P] [US2] Extend `Coin` and new `CoinReference` frontend types in `src/web/src/types/index.ts`
- [x] T026 [P] [US2] Implement reference API methods and era query param support in `src/web/src/api/client.ts`
- [x] T027 [US2] Create reference CRUD UI section in `src/web/src/components/coin/CoinReferencesSection.vue`
- [x] T028 [US2] Integrate `CoinReferencesSection` into coin details page in `src/web/src/pages/CoinDetailPage.vue`
- [x] T029 [US2] Replace free-text era entry with constrained era select in `src/web/src/components/CoinForm.vue`
- [x] T030 [US2] Add era filter state and request wiring in `src/web/src/composables/useCollectionFilters.ts`
- [x] T031 [P] [US2] Add reusable era filter control in `src/web/src/components/collection/EraFilter.vue`
- [x] T032 [US2] Integrate era filter into desktop and PWA collection headers in `src/web/src/components/collection/DesktopCollectionHeader.vue` and `src/web/src/components/collection/PwaCollectionHeader.vue`
- [x] T033 [US2] Thread era filter props through collection page orchestration in `src/web/src/pages/CollectionPage.vue`

**Checkpoint**: User Stories 1 and 2 are independently functional and testable.

---

## Phase 5: User Story 3 - AI discovery + export interoperability (Priority: P2)

**Goal**: Emit structured candidate references from AI discovery and include references/era in export surfaces.

**Independent Test**: Run AI discovery and user export; verify candidate references include certainty/optional URI and exported coins include references + era.

### Implementation for User Story 3

- [x] T034 [US3] Extend AI response schema for candidate references (`catalog`, `volume`, `number`, `certainty`, `uri`) in `src/agent/app/models/responses.py`
- [x] T035 [US3] Add OCRE/RPC authority URI lookup helper and integrate into discovery flow in `src/agent/app/tools/numismatic_authority.py` and `src/agent/app/teams/coin_search.py`
- [x] T036 [US3] Extend Go proxy structs to pass candidate references from agent service in `src/api/services/agent_proxy.go`
- [x] T037 [US3] Extend agent chat suggestion DTO mapping for candidate references in `src/api/handlers/agent.go`
- [x] T038 [US3] Map candidate references into wishlist coin creation payload in `src/web/src/composables/useCoinSearchChat.ts`
- [x] T039 [US3] Include references + era in CSV/JSON export payload preloads in `src/api/repository/user_repository.go` and `src/api/handlers/user.go`
- [x] T040 [US3] Render structured references in PDF catalog detail card in `src/api/handlers/export_pdf.go`

**Checkpoint**: All user stories are independently functional and testable.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Final docs, compatibility notes, and rollout hardening.

- [x] T041 [P] Document structured reference UX and catalog glossary updates in `docs/features.md`
- [x] T042 [P] Document API payload examples and validation rules in `docs/api-reference.md`
- [x] T043 [P] Add implementation/runbook validation checklist in `specs/214-structured-numismatic-catalog-references/quickstart.md`
- [x] T044 Preserve legacy `referenceText`/`referenceUrl` behavior notes in importer/exporter docs in `docs/api-reference.md` and `src/web/src/components/HelpSection.vue`
- [x] T045 Regenerate OpenAPI artifacts after endpoint/schema changes in `src/api/docs/swagger.yaml`, `src/api/docs/swagger.json`, `src/api/docs/docs.go`, and `docs/openapi.json`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1 (Setup)**: Starts immediately.
- **Phase 2 (Foundational)**: Depends on Phase 1 and blocks all user stories.
- **Phase 3 (US1)**: Starts after Phase 2; MVP delivery target.
- **Phase 4 (US2)**: Starts after Phase 2; depends on US1 API surfaces for full UI flow.
- **Phase 5 (US3)**: Starts after Phase 2; depends on US1 data model and US2 reference payload shapes.
- **Phase 6 (Polish)**: Starts after desired user stories complete.

### User Story Dependency Graph

```text
Setup (Phase 1)
  -> Foundational (Phase 2)
    -> US1 (Phase 3, P1 MVP)
      -> US2 (Phase 4, P1)
      -> US3 (Phase 5, P2)
        -> Polish (Phase 6)
```

### Within-Story Ordering Rules

- Backend model/repository/service tasks before handler wiring.
- API contract updates before frontend integration tasks.
- UI component creation before page-level integration tasks.
- Export and AI wiring after base reference schema is stable.

---

## Parallel Execution Examples

### User Story 1 (US1)

```bash
Task T018: Implement list/create reference endpoints in src/api/handlers/coin_references.go
Task T021: Enforce era enum binding in src/api/handlers/coins.go
Task T023: Add references preload support in src/api/repository/coin_repository.go
```

### User Story 2 (US2)

```bash
Task T025: Extend frontend reference types in src/web/src/types/index.ts
Task T026: Add reference client methods in src/web/src/api/client.ts
Task T031: Create EraFilter.vue component
```

### User Story 3 (US3)

```bash
Task T034: Extend agent response schema in src/agent/app/models/responses.py
Task T036: Extend Go proxy structs in src/api/services/agent_proxy.go
Task T039: Add CSV/JSON export preloads for references + era in src/api/repository/user_repository.go
```

---

## Implementation Strategy

### MVP First (US1 only)

1. Complete Phase 1 and Phase 2.
2. Deliver Phase 3 (US1) end-to-end.
3. Validate structured persistence and catalog-rule validation via API.
4. Demo backend-ready MVP before UI/AI/export expansion.

### Incremental Delivery

1. Ship US1 (structured persistence + validation).
2. Ship US2 (desktop/PWA management + era filtering).
3. Ship US3 (AI candidate attribution + export parity).
4. Finish with documentation and OpenAPI sync in Phase 6.

### Squad Parallelization

1. **Backend dev**: T009–T024 (model/repo/service/handler/API wiring).
2. **Frontend dev**: T025–T033 (types/client/components/pages).
3. **Agent/devrel pair**: T034–T042 (AI schema + export + docs).
