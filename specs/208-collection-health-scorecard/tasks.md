---
description: "Task list for feature #208 Collection Health Scorecard (v1)"
---

# Tasks: Collection Health Scorecard (v1)

**Input**: Design documents from `specs/208-collection-health-scorecard/`  
**Prerequisites**: `plan.md`, `spec.md`, `research.md`, `data-model.md`, `contracts/health-scorecard.openapi.yaml`, `quickstart.md`

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Create baseline scaffolding shared by all stories.

- [ ] T001 Create shared health constants, enums, and DTO structs in `src/api/services/health_types.go`
- [ ] T002 Create deterministic health fixture builders for Go tests in `src/api/services/health_testdata_test.go`
- [ ] T003 [P] Create health repository skeleton with method signatures in `src/api/repository/health_repository.go`
- [ ] T004 [P] Create user health handler skeleton for stats and coin health endpoints in `src/api/handlers/health.go`
- [ ] T005 [P] Create admin health handler skeleton for aggregate metrics in `src/api/handlers/admin_health.go`
- [ ] T006 [P] Create frontend health API/type placeholders in `src/web/src/api/client.ts` and `src/web/src/types/index.ts`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core health-scoring infrastructure required before any user story work.

**⚠️ CRITICAL**: Complete this phase before starting Phase 3+.

- [ ] T007 Add persisted `CollectionHealthSnapshot` model with unique user/date constraint in `src/api/models/collection_health_snapshot.go`
- [ ] T008 Register snapshot migration and indexing in `src/api/database/database.go`
- [ ] T009 [P] Add repository tests for snapshot upsert and 30-day baseline lookup in `src/api/repository/health_repository_test.go`
- [ ] T010 Implement repository queries for eligible coins, dimension inputs, and snapshot upserts in `src/api/repository/health_repository.go`
- [ ] T011 [P] Add scoring unit tests for weights, grade thresholds, freshness buckets, and empty collections in `src/api/services/health_service_test.go`
- [ ] T012 Implement deterministic scoring, checklist generation, and collection trend logic in `src/api/services/health_service.go`
- [ ] T013 Implement daily health snapshot scheduler and wiring in `src/api/services/collection_health_scheduler.go`, `src/api/services/settings_service.go`, and `src/api/main.go`

**Checkpoint**: Health scoring core and persistence are ready.

---

## Phase 3: User Story 1 - See collection health at a glance (Priority: P1) 🎯 MVP

**Goal**: Deliver dashboard collection score, grade, weighted dimensions, and 30-day trend indicator.

**Independent Test**: Seed mixed-quality active coins plus historical snapshot data, open `/stats`, and verify score/grade/delta match backend calculations; verify empty state for no eligible coins.

### Tests for User Story 1

- [ ] T014 [P] [US1] Add contract-style handler test for `GET /api/stats/health` response shape in `src/api/handlers/health_handler_test.go`
- [ ] T015 [P] [US1] Add trend-direction and delta unit tests for collection summary output in `src/api/services/health_service_test.go`

### Implementation for User Story 1

- [ ] T016 [US1] Add collection health response DTOs for Swagger docs in `src/api/handlers/swagger_types.go`
- [ ] T017 [US1] Implement `GET /api/stats/health` endpoint in `src/api/handlers/health.go`
- [ ] T018 [US1] Register `/api/stats/health` route in `src/api/main.go`
- [ ] T019 [P] [US1] Implement `getCollectionHealthSummary()` API client call in `src/web/src/api/client.ts`
- [ ] T020 [P] [US1] Add collection health state/actions to Pinia store in `src/web/src/stores/coins.ts`
- [ ] T021 [P] [US1] Create dashboard scorecard component in `src/web/src/components/stats/CollectionHealthScorecard.vue`
- [ ] T022 [P] [US1] Create dashboard trend indicator component in `src/web/src/components/stats/CollectionHealthTrendIndicator.vue`
- [ ] T023 [P] [US1] Create no-eligible-coins empty-state component in `src/web/src/components/stats/CollectionHealthEmptyState.vue`
- [ ] T024 [US1] Integrate health scorecard, trend, and empty state into `src/web/src/pages/StatsPage.vue`
- [ ] T025 [P] [US1] Add Stats page rendering tests for score/grade/trend states in `src/web/src/pages/__tests__/StatsPage.health.test.ts`
- [ ] T026 [US1] Add dashboard health validation steps for seeded scenarios in `specs/208-collection-health-scorecard/quickstart.md`

**Checkpoint**: User Story 1 is independently functional and testable.

---

## Phase 4: User Story 2 - Improve low-quality coin records quickly (Priority: P1)

**Goal**: Deliver per-coin health scores, missing checklist items, and Needs Attention queue with deterministic ordering and quick actions.

**Independent Test**: Open collection and coin detail pages, verify lowest-score coins appear first (tie-break by oldest update then coin ID), and confirm quick actions route to existing edit/upload/valuation/analysis flows.

### Tests for User Story 2

- [ ] T027 [P] [US2] Add contract-style handler test for `GET /api/coins/health` pagination and `scope=needs_attention` behavior in `src/api/handlers/health_handler_test.go`
- [ ] T028 [P] [US2] Add service tests for missing checklist key mapping and quick-action hints in `src/api/services/health_service_test.go`

### Implementation for User Story 2

- [ ] T029 [US2] Implement needs-attention ordering query with tie-break rules in `src/api/repository/health_repository.go`
- [ ] T030 [US2] Implement `GET /api/coins/health` endpoint with scope filtering in `src/api/handlers/health.go`
- [ ] T031 [P] [US2] Add `CoinHealthItem` and `MissingChecklistItem` frontend types in `src/web/src/types/index.ts`
- [ ] T032 [P] [US2] Implement `getCoinHealthList()` API client call in `src/web/src/api/client.ts`
- [ ] T033 [P] [US2] Add needs-attention queue state/actions in `src/web/src/stores/coins.ts`
- [ ] T034 [P] [US2] Create Needs Attention queue component in `src/web/src/components/collection/NeedsAttentionQueue.vue`
- [ ] T035 [P] [US2] Create coin health checklist component in `src/web/src/components/coin/CoinHealthChecklist.vue`
- [ ] T036 [US2] Integrate Needs Attention queue into `src/web/src/pages/CollectionPage.vue`
- [ ] T037 [US2] Integrate coin health score/checklist/quick actions into `src/web/src/pages/CoinDetailPage.vue`
- [ ] T038 [US2] Add queue ordering and quick-action routing tests in `src/web/src/pages/__tests__/CollectionPage.health.test.ts` and `src/web/src/pages/__tests__/CoinDetailPage.health.test.ts`

**Checkpoint**: User Stories 1 and 2 are independently functional and testable.

---

## Phase 5: User Story 3 - Monitor quality across users as admin (Priority: P2)

**Goal**: Deliver admin-only aggregate health metrics (median score, low-score %, top missing fields).

**Independent Test**: Log in as admin and verify aggregate metrics render; verify non-admin requests to admin health endpoint are forbidden.

### Tests for User Story 3

- [ ] T039 [P] [US3] Add admin health authorization and payload tests for `GET /api/admin/health/summary` in `src/api/handlers/admin_health_handler_test.go`
- [ ] T040 [P] [US3] Add aggregate metric unit tests (median, low-score %, top-missing fields) in `src/api/services/health_service_test.go`

### Implementation for User Story 3

- [ ] T041 [US3] Implement admin aggregate computation logic in `src/api/services/health_service.go`
- [ ] T042 [US3] Add admin health response DTOs for Swagger docs in `src/api/handlers/swagger_types.go`
- [ ] T043 [US3] Implement `GET /api/admin/health/summary` endpoint in `src/api/handlers/admin_health.go`
- [ ] T044 [US3] Register `/api/admin/health/summary` route in `src/api/main.go`
- [ ] T045 [P] [US3] Implement `getAdminHealthSummary()` client method and type support in `src/web/src/api/client.ts` and `src/web/src/types/index.ts`
- [ ] T046 [P] [US3] Create admin health metrics UI component in `src/web/src/components/admin/AdminHealthSection.vue`
- [ ] T047 [US3] Integrate admin health panel and add UI coverage in `src/web/src/pages/AdminPage.vue` and `src/web/src/pages/__tests__/AdminPage.health.test.ts`

**Checkpoint**: All user stories are independently functional and testable.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Final hardening, documentation, and cross-story quality checks.

- [ ] T048 [P] Document scoring formula, thresholds, and checklist taxonomy in `docs/features.md`
- [ ] T049 [P] Document health endpoints, auth rules, and response contracts in `docs/api-reference.md`
- [ ] T050 [P] Regenerate and commit Swagger artifacts for health endpoints in `src/api/docs/swagger.yaml`, `src/api/docs/swagger.json`, and `src/api/docs/docs.go`
- [ ] T051 Harden edge-case handling for no-coin collections and insufficient trend history in `src/api/services/health_service.go` and `src/web/src/pages/StatsPage.vue`
- [ ] T052 Execute and finalize health scorecard validation checklist in `specs/208-collection-health-scorecard/quickstart.md`

---

## Dependencies & Execution Order

### Phase Dependencies

- Phase 1 → required before Phase 2
- Phase 2 → blocks all user stories
- Phase 3 (US1) and Phase 4 (US2) can start after Phase 2
- Phase 5 (US3) starts after foundational work and prioritized completion of P1 stories
- Phase 6 starts after desired user stories are complete

### User Story Dependency Graph

```text
Phase 1 (Setup)
  -> Phase 2 (Foundational)
    -> US1 (Dashboard health summary)
    -> US2 (Per-coin health + needs attention)
    -> US3 (Admin aggregates; scheduled after P1 delivery)
      -> Phase 6 (Polish)
```

### Within-Story Ordering Rules

- Write tests before implementation tasks in each user story phase
- Repository/service logic before handler wiring
- API client/types before page integration
- Page integration before story-level UI validation tests

---

## Parallel Execution Examples

### User Story 1 (US1)

```bash
Task T019: Implement getCollectionHealthSummary() in src/web/src/api/client.ts
Task T020: Add collection health state/actions in src/web/src/stores/coins.ts
Task T021: Create CollectionHealthScorecard.vue
Task T022: Create CollectionHealthTrendIndicator.vue
Task T023: Create CollectionHealthEmptyState.vue
```

### User Story 2 (US2)

```bash
Task T031: Add CoinHealthItem and MissingChecklistItem types in src/web/src/types/index.ts
Task T032: Implement getCoinHealthList() in src/web/src/api/client.ts
Task T034: Create NeedsAttentionQueue.vue
Task T035: Create CoinHealthChecklist.vue
```

### User Story 3 (US3)

```bash
Task T039: Add admin endpoint auth/payload tests in src/api/handlers/admin_health_handler_test.go
Task T040: Add aggregate metric unit tests in src/api/services/health_service_test.go
Task T046: Create AdminHealthSection.vue
```

---

## Implementation Strategy

### MVP First (US1)

1. Complete Phase 1 and Phase 2.
2. Complete Phase 3 (US1) end-to-end.
3. Validate dashboard score, grade, dimensions, and trend behavior.
4. Demo/deploy MVP.

### Incremental Delivery

1. Ship US1 (dashboard insight).
2. Ship US2 (actionable remediation queue and checklist).
3. Ship US3 (admin monitoring metrics).
4. Finish with Phase 6 polish and docs.

### Team Parallelization

1. One backend engineer: T007–T013 core scoring/snapshot/scheduler.
2. One frontend engineer: T019–T024 dashboard and T031–T037 queue/detail UI.
3. One full-stack engineer: US3 admin endpoint + panel (T039–T047).
