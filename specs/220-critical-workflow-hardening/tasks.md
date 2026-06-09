---
description: "Task breakdown for F013 critical workflow hardening"
---

# Tasks: Critical Workflow Hardening

**Input**: Design documents from `/specs/220-critical-workflow-hardening/`  
**Prerequisites**: `plan.md`, `spec.md`, Agentic Excellence Roadmap AE001-AE028

**Tests**: This feature is test-first where practical. Backend regression coverage should land with typed mutation changes; browser workflow infrastructure lands after fixture shape is stable.

## Phase 1: Promotion Foundation (Completed by Spec Promotion)

**Purpose**: Promote F013 from backlog into an active SpecKit feature and baseline the scope.

- [x] T001 [AE001] Promote `specs/_backlog/F013-critical-workflow-hardening.md` into active feature `specs/220-critical-workflow-hardening/spec.md`
- [x] T002 [AE002] Capture current critical workflow behavior and known regression targets in `specs/220-critical-workflow-hardening/spec.md`
- [x] T003 [AE003] Define the golden collection fixture shape in `specs/220-critical-workflow-hardening/spec.md`
- [x] T004 [AE004] Decide deterministic browser test tool scope and run cadence in `specs/220-critical-workflow-hardening/plan.md`
- [x] T005 [AE005] Add initial task breakdown in `specs/220-critical-workflow-hardening/tasks.md`

---

## Phase 2: Backend Typed Mutation Path

**Purpose**: Make coin create/update explicit, association-safe, and regression-tested before browser tests expand.

**⚠️ CRITICAL**: Complete this phase before browser workflow implementation.

- [x] T006 [AE006] Inventory all coin create/update entry points in `src/api/handlers/coins.go`, `src/api/services/coin_service.go`, and `src/api/repository/coin_repository.go`
- [x] T007 [AE007] Define typed coin create/update request DTOs in `src/api/handlers/coins.go` or dedicated `src/api/handlers/coin_requests.go`
- [x] T008 [AE008] Map typed request DTOs to service inputs without binding broad `models.Coin` mutation payloads in `src/api/handlers/coins.go`
- [x] T009 [AE009] Preserve service-layer business rules for storage locations, eras, references, value history, tags, and sets in `src/api/services/coin_service.go`
- [x] T010 [AE010] Reduce repository update helpers to one obvious mutation path or document why each remaining helper is distinct in `src/api/repository/coin_repository.go`
- [x] T011 [AE011] Add backend regression tests for one-field edits, storage-location edits, explicit false/empty-string updates, storage-location null clears, nullable scalar null clears, tags, sets, references, legacy/custom era, and value snapshots in `src/api/handlers/coin_handler_test.go`
- [x] T012 [AE012] Add repository-level tests for association-safe, explicit zero-value, and nullable scalar NULL updates in `src/api/repository/coin_repository_test.go`
- [x] T013 [AE013] Regenerate API docs with `task openapi` if public request/response contracts change

**Brutus decomposition for T011-T012**:

- [x] Add handler test for typed DTO unknown/broad relationship fields being ignored or rejected according to the final contract.
- [x] Add handler tests for create/update structured references, storage-location set/clear/invalid, one scalar edit preserving sibling associations, and manual current-value history side effects.
- [x] Add handler regression for explicit false booleans and empty-string clears through the HTTP update path.
- [x] Add service tests only where business rules live: storage ownership validation, era validation, reference normalization, and value-history snapshots.
- [x] Add repository tests only for persistence invariants: association-safe updates, storage-location NULL updates, preloads, and helper behavior.
- [x] Add repository regression proving selected update fields persist explicit false, empty-string, and numeric zero values.
- [x] Add handler and repository regressions proving explicit nullable scalar `null` clears persist for `purchasePrice`, `currentValue`, dates, sale price, weight, and diameter while omitted fields preserve existing values.

**Checkpoint**: Coin mutation inputs are typed and backend regressions prove critical related data survives updates.

---

## Phase 3: Golden Fixtures

**Purpose**: Provide deterministic, representative collection data for backend and browser workflow coverage.

- [x] T014 [P] [AE014] Create Go test fixture builders for representative coins in `src/api/testutil/` or existing API test helpers
- [x] T015 [P] [AE015] Create frontend fixture data for representative coins in `src/web/src/test/` or existing test helper folders
- [x] T016 [P] [AE016] Document fixture coverage in `docs/testing.md`
- [x] T017 [AE017] Ensure fixture data covers Roman, Greek, Byzantine, wishlist, sold, private, tagged, set-member, storage-location, image-heavy, and legacy/custom-era coins

**Golden fixture names**: `roman-denarius-core`, `greek-tetradrachm-valued`, `byzantine-solidus-set-member`, `wishlist-aureus-target`, `sold-sestertius-archive`, `private-provincial-bronze`, `tagged-follis-storage`, `image-heavy-drachm`, and `reference-rich-denarius`.

**Checkpoint**: Backend and frontend tests can create or load the same representative collection states deterministically.

---

## Phase 4: Browser Workflow Coverage

**Purpose**: Add deterministic browser-level coverage for the highest-value collection workflows.

- [x] T018 [AE018] Add deterministic browser smoke test infrastructure under `src/web/` using the selected tool
- [x] T019 [AE019] Add login and authenticated session setup workflow tests under `src/web/`
- [x] T020 [AE020] Add add-coin workflow test covering manual entry and save under `src/web/`
- [x] T021 [AE021] Add edit-one-field workflow test under `src/web/`
- [x] T022 [AE022] Add edit-storage-location workflow test under `src/web/`
- [x] T023 [AE023] Add edit-tags-and-sets workflow test under `src/web/`
- [x] T024 [AE024] Add upload/delete image workflow test under `src/web/`
- [x] T025 [AE025] Add collection search/filter workflow test under `src/web/`
- [x] T026 [AE026] Add mobile viewport edit workflow test under `src/web/`
- [x] T027 [AE027] Add local task command for critical workflow tests in `Taskfile.yml`
- [x] T028 [AE028] Document critical workflow test command and expected fixture setup in `docs/testing.md`

**Browser split**:

- [x] Deterministic F013 tests block critical regressions for login, add, edit, image, search/filter, and mobile viewport flows.
- [x] Later F011 AI-driven exploration remains advisory and uses the F013 golden fixtures/workflow list; it should report console/network/visual/accessibility findings without expanding the mutation contract.

**Checkpoint**: Critical workflows run through one documented local command and produce deterministic pass/fail output.

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1 (Promotion Foundation)**: No dependencies; complete before implementation.
- **Phase 2 (Backend Typed Mutation Path)**: Depends on Phase 1 and blocks browser workflow implementation.
- **Phase 3 (Golden Fixtures)**: Depends on Phase 1; backend and frontend fixture builders can proceed in parallel after fixture shape is agreed.
- **Phase 4 (Browser Workflow Coverage)**: Depends on Phase 2 typed update stability and Phase 3 fixture availability.

### Execution Graph

```text
Promotion -> Backend Typed Mutation Path -> Browser Workflow Coverage
          \-> Golden Fixtures ----------/
```

### Parallel Execution Examples

```bash
Task T014: Create Go fixture builders in src/api/testutil/
Task T015: Create frontend fixture data in src/web/src/test/
Task T016: Document fixture coverage in docs/testing.md
```

## Implementation Strategy

### First Slice

1. Start with T006 to inventory the existing mutation paths.
2. Complete T007-T010 as one backend contract slice.
3. Land T011-T012 with the contract changes so regression coverage proves association safety.
4. Run Go validation from `src/api`: `go build ./...`, `go vet ./...`, `go test -v ./...`.

### Incremental Delivery

1. Ship typed backend mutation contracts and regression tests.
2. Add golden fixture builders/data and documentation.
3. Add deterministic browser workflow infrastructure and critical flows.
4. Add Taskfile command and final testing documentation.
