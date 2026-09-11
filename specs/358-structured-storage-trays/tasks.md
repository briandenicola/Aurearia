# Tasks: Structured Coin Storage Trays

**Input**: Design documents from `C:\Users\brian.denicolafamily\Code\AncientCoins\specs\358-structured-storage-trays\`
**Prerequisites**: `plan.md`, `spec.md`, `research.md`, `data-model.md`, `contracts/openapi.yaml`, `contracts/ui-interaction.md`, `quickstart.md`
**Tests**: Required by the feature specification. Write each focused test before its corresponding implementation and confirm that it fails for the intended reason.
**Organization**: Tasks are grouped by user story so each story remains independently testable after the shared migration foundation is complete.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Safe to execute concurrently because the task changes a different file and does not depend on unfinished work in the same phase.
- **[Story]**: Maps a task to its user story in `spec.md`.
- Every task names the exact file or directory it changes or validates.

## Locked-File Guardrails

Implementation MUST NOT modify `.specify/memory/constitution.md`, `specs/358-structured-storage-trays/spec.md`, any merged `docs/adr/*.md`, `.squad/decisions.md`, `.squad/agents/*/charter.md`, `.squad/agents/*/history.md`, `.squad/log/**`, or `.squad/orchestration-log/**`. New cross-cutting decisions belong in `.squad/decisions/inbox/`; the final audit belongs in `docs/audits/`.

---

## Phase 1: Setup (Shared Test Infrastructure)

**Purpose**: Establish reusable fixtures and baselines without adding dependencies or changing locked governance artifacts.

- [X] T001 [P] Add owner-scoped Standard Location, Coin Tray, slot, and minimal-image builders for backend tests in src/api/testutil/storage_tray_fixtures.go
- [X] T002 [P] Add typed 1×1, 3×3, 20×20, empty, occupied, and missing-image tray fixtures for component tests in src/web/src/test/fixtures/storageTrays.ts
- [X] T003 Record the pre-change Museum Tray, storage-location, coin-form, duplicate, and bulk regression baseline by running the focused suites documented in specs/358-structured-storage-trays/quickstart.md

---

## Phase 2: Foundational (Blocking Schema and Migration)

**Purpose**: Create the additive persistence contract required by every user story.

**CRITICAL**: No user-story implementation begins until this phase is complete and its migration tests pass.

- [X] T004 [P] Add `standard`/`tray` type constants plus defaulted `Type` and nullable `Rows`/`Columns` fields without non-standard-library imports in src/api/models/storage_location.go
- [X] T005 [P] Add nullable one-based `StorageSlot` while preserving the existing `StorageLocation` `constraint:-` association in src/api/models/coin.go
- [X] T006 Write failing fresh-schema, additive legacy-backfill, stable-ID/association, repeat-run, index-presence, and duplicate-slot failure tests in src/api/database/feature358_migration_regression_test.go
- [X] T007 Implement error-propagating additive migration, null/empty type backfill, mixed-row validation, and idempotent `idx_coins_storage_location_slot_unique` creation without a physical foreign key in src/api/database/database.go
- [X] T008 Verify the real production migration sequence passes twice and fails visibly without deleting or remapping malformed rows in src/api/database/feature358_migration_regression_test.go

**Checkpoint**: Existing data is preserved as Standard storage, new nullable fields exist, and SQLite authoritatively prevents duplicate non-null location/slot pairs.

---

## Phase 3: User Story 1 - Configure Storage Locations and Trays (Priority: P1)

**Goal**: Let an owner create, list, rename, resize, and delete Standard Locations and fixed-dimension Coin Trays with correct occupancy safeguards.

**Independent Test**: In Settings, create and rename a Standard Location, create a 3×3 tray, verify its type and `0 / 9` occupancy, reject invalid or duplicate input, rename an occupied tray without assignment loss, block its resize/delete, then resize and delete it after it is empty.

### Tests for User Story 1

- [X] T009 [P] [US1] Write repository tests for owner-scoped case-insensitive names, occupied counts, reference counts, and cross-owner isolation in src/api/repository/storage_location_repository_test.go
- [X] T010 [P] [US1] Extend service tests for trimmed names, the 100-location cap, type normalization, 1–20 dimensions, 400-slot capacity, type immutability, occupied resize, referenced delete, and empty-tray lifecycle in src/api/services/storage_location_service_test.go
- [X] T011 [P] [US1] Write handler tests for omitted type compatibility, nullable dimensions, 400 validation envelopes, generic cross-owner 404 responses, and `tray_occupied`/`location_referenced` 409 responses in src/api/handlers/storage_location_handler_test.go
- [X] T012 [P] [US1] Extend Settings tests for type-specific fields, capacity preview, occupancy labels, disabled occupied dimensions, rename, deletion count, field errors, and conflict-code handling in src/web/src/components/settings/__tests__/SettingsDataSection.test.ts

### Implementation for User Story 1

- [X] T013 [US1] Add owner-scoped type-aware CRUD, case-insensitive uniqueness, occupancy/reference counts, and transaction-bound lifecycle queries using shared scopes in src/api/repository/storage_location_repository.go
- [X] T014 [US1] Implement normalized validation, capacity calculation, stable sentinel errors, occupied resize/type-conversion guards, and referenced-delete rules in src/api/services/storage_location_service.go
- [X] T015 [US1] Extend storage-location request/response DTOs, omission/null handling, stable 400/404/409 mapping, and Swagger annotations in src/api/handlers/storage_location.go
- [X] T016 [P] [US1] Extend `StorageLocation` and write DTO types with discriminated `standard`/`tray` fields and occupancy metadata in src/web/src/types/collection.ts
- [X] T017 [US1] Extend typed list/create/update/delete calls for type-specific payloads and structured errors in src/web/src/api/endpoints/collection.ts
- [X] T018 [US1] Implement Settings type selection, constrained dimensions, capacity preview, type/dimension/occupancy display, rename-only occupied editing, and actionable validation/conflict messages in src/web/src/components/settings/SettingsDataSection.vue
- [X] T019 [US1] Run and stabilize the focused backend and Settings suites for this story in src/api/repository/storage_location_repository_test.go, src/api/services/storage_location_service_test.go, src/api/handlers/storage_location_handler_test.go, and src/web/src/components/settings/__tests__/SettingsDataSection.test.ts

**Checkpoint**: User Story 1 is independently functional through Settings and preserves all assignments during rejected lifecycle operations.

---

## Phase 4: User Story 2 - Assign a Coin to an Exact Tray Slot (Priority: P1)

**Goal**: Make exact one-based slot assignment authoritative across coin create/update and bulk workflows, including concurrent claims and atomic moves.

**Independent Test**: Assign a coin to row 2, column 3 of a 3×3 tray, prove another coin cannot claim it, retain the edited coin's current slot, atomically move the first coin to Standard storage, and verify the released slot becomes available while bulk assignment refuses trays.

### Tests for User Story 2

- [X] T020 [P] [US2] Extend coin repository tests for atomic location-plus-slot writes, failed-move rollback, Standard clearing, named unique-index translation, and simultaneous same-slot claims in src/api/repository/coin_repository_test.go
- [X] T021 [P] [US2] Extend coin service tests for assignment presence semantics, ownership, Standard/tray combinations, first/last boundaries, current-coin reuse, stale-slot reset, occupied conflicts, and unchanged omitted updates in src/api/services/coin_service_test.go
- [X] T022 [P] [US2] Extend coin handler tests for create/update omission versus explicit null, invalid field envelopes, owner-hidden 404, and `slot_occupied` 409 recovery contracts in src/api/handlers/coin_handler_test.go
- [X] T023 [P] [US2] Write occupancy handler tests for minimal occupied slot numbers, current-owner coin recognition, non-tray rejection, and cross-owner privacy in src/api/handlers/storage_location_handler_test.go
- [X] T024 [P] [US2] Write bulk handler tests proving Standard assignment and clear update both storage fields atomically while every tray ID is rejected in src/api/handlers/bulk_handler_test.go
- [X] T025 [P] [US2] Extend coin form tests for conditional coordinates, occupied/current states, location-change reset, required selection, stale-conflict refresh, and preserved form data in src/web/src/components/__tests__/CoinForm.test.ts
- [X] T026 [P] [US2] Add bulk picker tests for selectable Standard/clear choices and disabled trays with the exact per-coin-slot explanation in src/web/src/components/__tests__/BulkLocationPickerModal.test.ts

### Implementation for User Story 2

- [X] T027 [US2] Add owner-scoped occupancy reads, transaction-bound assignment updates, association reloads, and exact named-index conflict translation without partial `Save` calls in src/api/repository/coin_repository.go
- [X] T028 [US2] Implement the typed presence-aware assignment validator and transactional create/update orchestration with stable storage errors in src/api/services/coin_service.go
- [X] T029 [US2] Add nullable/presence-aware `storageLocationId` and `storageSlot` mutation DTO mapping while stripping read-only associations in src/api/handlers/coin_requests.go
- [X] T030 [US2] Map assignment validation, owner, and concurrency errors to generic 400/404/409 responses without leaking database details in src/api/handlers/coins.go
- [X] T031 [US2] Add the owner-scoped occupancy handler with optional validated `coinId` and Swagger documentation in src/api/handlers/storage_location.go
- [X] T032 [US2] Register `GET /api/storage-locations/:id/occupancy` only in the authenticated route group in src/api/routes_protected.go
- [X] T033 [P] [US2] Extend coin and assignment types with nullable one-based `storageSlot` and occupancy response types in src/web/src/types/coin.ts
- [X] T034 [US2] Add typed occupancy access, include `storageSlot` in `NULLABLE_FIELDS`, and sanitize mutation payloads in src/web/src/api/endpoints/coins.ts
- [X] T035 [US2] Re-export the new typed occupancy API through the authenticated client surface in src/web/src/api/client.ts
- [X] T036 [US2] Implement the accessible one-based tray slot picker, occupancy refresh, own-slot exception, stale-state reset, client validation, and server-conflict recovery in src/web/src/components/CoinForm.vue
- [X] T037 [US2] Route bulk Standard/clear writes through CoinService and reject tray targets before persistence in src/api/handlers/bulk.go
- [X] T038 [US2] Disable or exclude trays and display “Tray assignments require choosing a slot on each coin.” in src/web/src/components/BulkLocationPickerModal.vue
- [X] T039 [US2] Add the exact-slot add/edit, occupied conflict, atomic release, and Standard-clear browser workflow in src/web/e2e/workflows/coin-form.spec.ts
- [X] T040 [US2] Run and stabilize all focused assignment, race, handler, form, bulk, and browser tests for this story in src/api/repository/coin_repository_test.go, src/api/services/coin_service_test.go, src/api/handlers/coin_handler_test.go, src/api/handlers/bulk_handler_test.go, src/web/src/components/__tests__/CoinForm.test.ts, src/web/src/components/__tests__/BulkLocationPickerModal.test.ts, and src/web/e2e/workflows/coin-form.spec.ts

**Checkpoint**: User Story 2 is independently testable through API and form flows, with the database deciding concurrent winners and failed moves preserving prior assignments.

---

## Phase 5: User Story 3 - View Physical Storage Trays (Priority: P1)

**Goal**: Provide a separate authenticated Storage Trays route that renders every physical tray at exact persisted coordinates using shared Museum Tray visual primitives.

**Independent Test**: Seed empty and non-contiguous trays from 1×1 through 20×20, load `/storage-trays` with one aggregate request, and verify exact well counts/coordinates, owner isolation, keyboard navigation, missing-image fallback, retry behavior, and fixed geometry in a narrow contained scroller while `/tray` remains unchanged.

### Tests for User Story 3

- [X] T041 [P] [US3] Write bounded-query aggregate repository tests for owner isolation, empty trays, positioned minimal coin/image fields, ordering, and exclusion of unrelated private fields in src/api/repository/storage_location_repository_test.go
- [X] T042 [P] [US3] Extend handler tests for authenticated aggregate shape, empty arrays, owner scoping, generic failures, and no unrelated coin details in src/api/handlers/storage_location_handler_test.go
- [X] T043 [P] [US3] Add exact row-major 1×1, sparse 3×3, and 400-well component tests with empty/occupied semantics, keyboard activation, focus, missing media, and fixed columns in src/web/src/components/__tests__/StorageTrayGrid.test.ts
- [X] T044 [P] [US3] Add page tests for one aggregate request, every/empty tray rendering, loading, empty, partial-image, recoverable error, retry, and navigation states in src/web/src/pages/__tests__/StorageTraysPage.test.ts
- [X] T045 [P] [US3] Extend application navigation tests for the distinct Collection children `/tray` and `/storage-trays` in src/web/src/__tests__/AppNavigation.test.ts

### Implementation for User Story 3

- [X] T046 [US3] Implement bounded owner-scoped tray and positioned minimal-coin aggregate queries without N+1 reads in src/api/repository/storage_location_repository.go
- [X] T047 [US3] Add aggregate DTO mapping and the authenticated Swagger-documented `GET /api/storage-trays` handler in src/api/handlers/storage_location.go
- [X] T048 [US3] Register `GET /api/storage-trays` under the protected route group in src/api/routes_protected.go
- [X] T049 [P] [US3] Add typed `TrayAggregate`, `TrayCoin`, `TrayImage`, and occupancy contracts in src/web/src/types/collection.ts
- [X] T050 [US3] Add the single authenticated aggregate request wrapper in src/web/src/api/endpoints/collection.ts
- [X] T051 [US3] Re-export the Storage Trays aggregate call through src/web/src/api/client.ts
- [X] T052 [US3] Minimally extract reusable felt-surface presentation without changing Museum Tray props or layout behavior in src/web/src/components/tray/TraySurface.vue and src/web/src/components/tray/MuseumTray.vue
- [X] T053 [US3] Extend the shared well primitive for fixed coordinates, authenticated-image fallback, gridcell semantics, and optional interaction without changing Museum Tray behavior in src/web/src/components/tray/MuseumTrayWell.vue
- [X] T054 [US3] Implement exact `1..rows*columns` row-major materialization, persisted column count, occupied lookup, coordinates, and no packing/filtering/pagination in src/web/src/components/tray/StorageTrayGrid.vue
- [X] T055 [US3] Add a labeled contained horizontal scroller, 44×44 occupied targets, visible token-based focus, non-color cues, and reduced-motion behavior in src/web/src/components/tray/StorageTrayGrid.vue
- [X] T056 [US3] Implement aggregate loading plus empty, retry, and partial-image states for every configured tray in src/web/src/pages/StorageTraysPage.vue
- [X] T057 [US3] Register the authenticated named `/storage-trays` route without altering `/tray` in src/web/src/router/index.ts
- [X] T058 [US3] Add Storage Trays beneath the existing Collection submenu while preserving the Museum Tray label and route in src/web/src/App.vue
- [X] T059 [US3] Add desktop/mobile browser coverage for exact sparse placement, empty trays, 20-column scrolling, pointer/Enter/Space activation, image fallback, and retry in src/web/e2e/workflows/storage-trays.spec.ts
- [X] T060 [US3] Run and stabilize aggregate, grid, page, navigation, and Storage Trays browser tests in src/api/repository/storage_location_repository_test.go, src/api/handlers/storage_location_handler_test.go, src/web/src/components/__tests__/StorageTrayGrid.test.ts, src/web/src/pages/__tests__/StorageTraysPage.test.ts, src/web/src/__tests__/AppNavigation.test.ts, and src/web/e2e/workflows/storage-trays.spec.ts

**Checkpoint**: User Story 3 is independently testable with seeded tray data, exact physical geometry, accessible interaction, one aggregate read, and no Museum Tray semantic reuse.

---

## Phase 6: User Story 4 - Preserve Existing Data and Sibling Workflows (Priority: P1)

**Goal**: Preserve all legacy locations and assignments while forcing duplicate, AI-assisted intake, Quick Capture, deep-identification, legacy API, and other sibling writes through the same placement contract.

**Independent Test**: Upgrade a legacy two-owner database twice without changing IDs, names, timestamps, owners, or assignments; then prove legacy omitted fields remain Standard-compatible, Standard duplication retains its location, tray duplication clears both fields, and every sibling path rejects an unslotted or occupied tray assignment.

### Tests for User Story 4

- [X] T061 [P] [US4] Complete the two-owner legacy migration regression for preservation, Standard classification, null slots, idempotence, and visible duplicate-index failure in src/api/database/feature358_migration_regression_test.go
- [X] T062 [P] [US4] Extend duplicate service tests for preserved Standard assignment and cleared tray location-plus-slot in src/api/services/coin_service_test.go
- [X] T063 [P] [US4] Add legacy API create/update tests proving omitted new fields preserve Standard behavior while invalid explicit tray combinations fail in src/api/handlers/coin_handler_test.go
- [X] T064 [P] [US4] Add intake contract tests for Standard assignment and rejection of missing, out-of-range, cross-owner, or occupied tray slots in src/api/services/coin_intake_service_test.go
- [X] T065 [P] [US4] Extend Quick Capture promotion regressions to prove CoinService prevents invalid tray persistence in src/api/services/quick_capture_service_test.go
- [X] T066 [P] [US4] Extend deep-identification proposal regressions to prove wishlist/coin creation cannot bypass the shared tray contract in src/api/services/deep_identification_proposal_test.go
- [X] T067 [P] [US4] Add cross-workflow contract coverage for manual create/update, JSON client payloads, duplicate, bulk, intake, Quick Capture, and deep-identification in src/api/services/storage_assignment_workflows_test.go

### Implementation for User Story 4

- [X] T068 [US4] Make duplicate creation transactional and type-aware so Standard locations remain compatible while tray location and slot are both cleared in src/api/services/coin_service.go
- [X] T069 [US4] Refactor `CommitDraft` to use CoinService's shared create/transaction assignment boundary and map only supported storage overrides in src/api/services/coin_intake_service.go
- [X] T070 [US4] Verify Quick Capture promotion continues to create through CoinService and remove any direct assignment bypass found by the regression in src/api/services/quick_capture_service.go
- [X] T071 [US4] Verify deep-identification/wishlist creation continues to create through CoinService and remove any direct assignment bypass found by the regression in src/api/services/deep_identification_proposal.go
- [X] T072 [US4] Preserve additive legacy request/response behavior while carrying nullable slot data through create, update, and duplicate handlers in src/api/handlers/coins.go
- [X] T073 [US4] Run and stabilize the migration and complete sibling-workflow contract suite in src/api/database/feature358_migration_regression_test.go, src/api/services/storage_assignment_workflows_test.go, src/api/services/coin_intake_service_test.go, src/api/services/quick_capture_service_test.go, src/api/services/deep_identification_proposal_test.go, and src/api/handlers/coin_handler_test.go

**Checkpoint**: User Story 4 proves upgrade safety and one shared storage contract across every known coin-creation and mutation sibling path.

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Synchronize contracts and documentation, preserve Museum Tray, complete governance records, and pass the full release gate.

- [X] T074 [P] Add or strengthen Museum Tray characterization coverage for responsive 3/4/6 columns, 12-coin drawers, swipe/click suppression, face toggle, size preference, membership, and adapter fields in src/web/src/components/__tests__/MuseumTray.test.ts, src/web/src/components/__tests__/MuseumTrayWell.test.ts, and src/web/src/pages/__tests__/TrayViewPage.test.ts
- [X] T075 [P] Extend the existing Museum Tray Playwright regression without adding physical-placement semantics in src/web/e2e/workflows/tray.spec.ts
- [X] T076 Complete Swagger annotations and stable validation/conflict schemas for all modified public handlers in src/api/handlers/storage_location.go, src/api/handlers/coins.go, and src/api/handlers/swagger_types.go
- [X] T077 Regenerate Swagger and synchronize generated API descriptions with `task openapi` in src/api/docs/docs.go, src/api/docs/swagger.json, src/api/docs/swagger.yaml, and docs/openapi.json
- [X] T078 [P] Document Standard Locations, Coin Trays, exact-slot workflows, Storage Trays versus Museum Tray, and upgrade compatibility in README.md and docs/features.md
- [X] T079 [P] Add the user guide for creation, assignment, fixed-grid navigation, accessibility, mobile scrolling, conflicts, and recovery in docs/features/storage-trays.md
- [X] T080 [P] Document endpoints, schemas, nullability, one-based arithmetic, bounds, authentication, owner scoping, and error codes in docs/api-reference.md
- [X] T081 [P] Document focused migration, race, service, component, accessibility, and Playwright regression commands in docs/testing.md
- [X] T082 Record any implementation-time cross-cutting choice or deviation from research.md, or an explicit “no new decisions” closure, in .squad/decisions/inbox/feature358-structured-storage-trays.md
- [X] T083 Run Go compile, vet, architecture, full tests, focused Feature 358 tests, and `task test-race` from src/api/ and Taskfile.yml. The GitHub Linux `go-race` job passed on PR #702; no local GCC toolchain is required.
- [X] T084 Run frontend lint, `vue-tsc --build`, Vitest, production build, and the storage-trays/coin-form/tray Playwright workflows from src/web/package.json and src/web/e2e/workflows/
- [X] T085 Verify generated OpenAPI drift, route documentation coverage, formatting, secrets, and intended-file scope with Taskfile.yml, src/api/route_openapi_drift_test.go, docs/openapi.json, and specs/358-structured-storage-trays/quickstart.md
- [X] T086 Perform the post-major-work QC audit across all eight audit domains and append evidence-backed blockers/follow-ups to docs/audits/2026-09-11.md
- [X] T087 Verify the final diff leaves .specify/memory/constitution.md, specs/358-structured-storage-trays/spec.md, .squad/decisions.md, and all locked .squad agent/log/orchestration files unchanged, and record the result in specs/358-structured-storage-trays/tasks.md

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1 — Setup**: No dependencies.
- **Phase 2 — Foundational**: Depends on Phase 1 and blocks all user stories.
- **Phase 3 — US1**: Depends on Phase 2.
- **Phase 4 — US2**: Depends on Phase 2; it may proceed alongside US1 after the shared storage model exists, but its UI integration consumes the same typed location contract.
- **Phase 5 — US3**: Depends on Phase 2; repository/handler work may proceed independently with seeded fixtures, while final navigation integration should use the typed contracts completed by US1/US2.
- **Phase 6 — US4**: Depends on Phase 2 and the shared assignment boundary from US2; its migration proof remains independently runnable.
- **Phase 7 — Polish**: Depends on every selected user-story phase.

### User Story Dependency Graph

```text
Phase 1 Setup
    |
Phase 2 Additive Schema/Migration
    |--------------------|--------------------|
    v                    v                    v
US1 Location CRUD     US2 Exact Assignment  US3 Physical Read View
                          |
                          v
                  US4 Sibling/Legacy Proof
    |____________________|____________________|
                          v
                  Polish / Quality Gate
```

### Within Each User Story

1. Write the focused tests and confirm the expected failure.
2. Implement repository persistence and transactions before service orchestration.
3. Implement service rules before HTTP adaptation.
4. Update typed frontend contracts before UI consumers.
5. Complete UI integration, then focused browser coverage.
6. Pass the independent checkpoint before starting dependent polish work.

---

## Parallel Execution Examples

### User Story 1

After Phase 2, these tests can be authored together because they are in separate files:

```text
T009 repository lifecycle tests
T010 service validation tests
T011 handler contract tests
T012 Settings component tests
```

After the Go contract is stable, `T016` can proceed independently while `T013`–`T015` finish the backend layers.

### User Story 2

The initial repository, service, handler, form, and bulk test tasks `T020`–`T026` are safe parallel work in separate files. After `T032`, frontend type task `T033` can proceed while backend bulk task `T037` is implemented.

### User Story 3

Tasks `T041`–`T045` are safe parallel test authoring. After the aggregate shape is fixed, `T049` can proceed in parallel with the low-level visual extraction beginning at `T052`.

### User Story 4

Tasks `T061`–`T067` are independent regression files and can be authored in parallel. Their failures determine whether implementation tasks `T068`–`T072` require code changes or only contract confirmation.

---

## Implementation Strategy

### MVP First (User Story 1)

1. Complete Setup.
2. Complete the additive migration foundation and prove it against fresh/legacy databases.
3. Complete User Story 1.
4. Stop and validate Settings creation, occupancy display, rename, resize, delete, validation, ownership, and backward compatibility.
5. Treat this as the first demonstrable increment; exact placement and visualization remain disabled until their story phases pass.

### Incremental Delivery

1. **Foundation + US1**: Safely configure Standard Locations and Coin Trays.
2. **Add US2**: Assign exact slots with authoritative concurrency protection.
3. **Add US3**: Display fixed physical geometry on the new read-only route.
4. **Add US4**: Close every migration and sibling-workflow bypass.
5. **Polish**: Synchronize OpenAPI/docs, preserve Museum Tray, record decisions, run the full Quality Gate, and complete the post-major-work QC audit.

### Safe Parallel Team Strategy

After Phase 2, separate owners may work on US1 backend/Settings, US2 assignment contracts, and US3 aggregate/grid tests. Avoid concurrent edits to shared hotspots including `src/api/repository/storage_location_repository.go`, `src/api/services/coin_service.go`, `src/api/handlers/storage_location.go`, `src/web/src/types/collection.ts`, `src/web/src/api/endpoints/collection.ts`, and `src/web/src/components/tray/MuseumTrayWell.vue`; serialize those tasks in ID order.

---

## Completion Notes

- Use shared `OwnedBy`/`OwnedByID` GORM scopes and preserve Handler → Service → Repository → Database.
- Treat the partial unique index as authoritative; pre-checks exist only for readable errors.
- Keep Standard Location omission behavior additive and never infer a tray slot.
- Keep `/tray`, Museum Tray packing, drawers, swipe, preferences, and membership semantics unchanged.
- Do not add a physical foreign key or a destructive SQLite table rebuild.
- Do not touch the Python agent unless actual implementation evidence invalidates the approved plan; such drift requires a new decision-inbox record before work continues.
- Check off tasks only after their focused tests and the relevant independent checkpoint pass.
