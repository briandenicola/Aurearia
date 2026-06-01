---
description: "Task breakdown for issue #216 AI intake draft + confirm coin creation"
---

# Tasks: AI Intake Draft + Confirm Coin Creation

**Input**: Design documents from `/specs/216-ai-intake-draft-confirm-coin-creation/`  
**Prerequisites**: `plan.md`, `spec.md`, `research.md`, `data-model.md`, `contracts/intake-flow.openapi.yaml`, `quickstart.md`

**Tests**: No explicit TDD requirement in the feature spec; no mandatory test-first tasks are included.

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Create file scaffolds and typed boundaries for intake draft/commit flow.

- [x] T001 Create intake draft model scaffold in `src/api/models/coin_intake_draft.go`
- [x] T002 [P] Create intake draft repository scaffold in `src/api/repository/coin_intake_draft_repository.go`
- [x] T003 [P] Create intake service scaffold in `src/api/services/coin_intake_service.go`
- [x] T004 [P] Create intake handler scaffold in `src/api/handlers/coin_intake.go`
- [x] T005 [P] Create Python intake team scaffold in `src/agent/app/teams/coin_intake.py`
- [x] T006 [P] Create intake composable scaffold in `src/web/src/composables/useCoinIntake.ts`
- [x] T007 [P] Create intake review component scaffold in `src/web/src/components/coin/CoinIntakeReviewPanel.vue`
- [x] T008 [P] Add intake placeholder types in `src/web/src/types/index.ts`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Implement persistent draft domain, schema plumbing, and route contracts that block all stories.

**⚠️ CRITICAL**: Complete this phase before starting user stories.

- [x] T009 Implement `CoinIntakeDraft` fields and enum state constraints in `src/api/models/coin_intake_draft.go`
- [x] T010 Register `CoinIntakeDraft` migration and auto-migrate wiring in `src/api/database/database.go`
- [x] T011 Implement draft repository methods (`CreateDraft`, `FindOwnedDraft`, `UpdateStatus`, `MarkExpired`) in `src/api/repository/coin_intake_draft_repository.go`
- [x] T012 [P] Add intake draft/commit request and response schema types in `src/agent/app/models/requests.py`
- [x] T013 [P] Add intake confidence/evidence response models in `src/agent/app/models/responses.py`
- [x] T014 [P] Extend Go agent proxy DTOs and intake transport methods in `src/api/services/agent_proxy.go`
- [x] T015 Implement agent route registration for intake draft generation in `src/agent/app/routes.py`
- [x] T016 [P] Add intake draft/commit DTOs for Swagger serialization in `src/api/handlers/swagger_types.go`
- [x] T017 [P] Add intake API client methods (`createDraft`, `commitDraft`) in `src/web/src/api/client.ts`
- [x] T018 [P] Add frontend intake domain interfaces (`CoinIntakeDraft`, `IntakeEvidenceItem`, `IntakeCommitRequest`) in `src/web/src/types/index.ts`
- [x] T048 [P] Add optional `coinCardImage` field to intake draft DTOs in `src/api/handlers/swagger_types.go` and `src/web/src/types/index.ts`
- [x] T053 [P] Add intake entry mode and camera permission state types in `src/web/src/types/index.ts` and `src/web/src/composables/useCoinIntake.ts`

**Checkpoint**: Intake domain primitives and contracts are in place.

---

## Phase 3: User Story 1 - Generate structured AI intake drafts (Priority: P1) 🎯 MVP

**Goal**: Produce a structured draft from camera/upload photos, OCR/lookups, and optional coin-card input with confidence and source evidence.

**Independent Test**: In PWA mode, verify camera-first intake opens by default (permission granted), upload remains available, and submitted images plus optional coin-card produce structured draft payload with confidence/evidence.

### Implementation for User Story 1

- [x] T019 [US1] Implement `coin_intake` extraction pipeline with structured draft output in `src/agent/app/teams/coin_intake.py`
- [x] T020 [US1] Return typed draft payload from intake route handler in `src/agent/app/routes.py`
- [x] T021 [US1] Implement draft orchestration service (agent call + persistence + expiry) in `src/api/services/coin_intake_service.go`
- [x] T022 [US1] Implement `POST /api/coins/intake/draft` handler binding, validation, and response mapping in `src/api/handlers/coin_intake.go`
- [x] T023 [US1] Register authenticated intake draft route in `src/api/main.go`
- [x] T024 [US1] Add Swagger annotations for draft endpoint in `src/api/handlers/coin_intake.go`
- [x] T025 [P] [US1] Implement draft request state machine (`idle/loading/success/error`) in `src/web/src/composables/useCoinIntake.ts`
- [x] T026 [US1] Add “Generate AI Draft” action and request wiring in `src/web/src/pages/AddCoinPage.vue`
- [x] T027 [US1] Map unresolved field keys into UI-consumable state in `src/web/src/composables/useCoinIntake.ts`
- [x] T049 [US1] Add optional coin-card upload input and payload binding in `src/web/src/pages/AddCoinPage.vue`
- [x] T050 [US1] Pass coin-card evidence through agent intake pipeline and response mapping in `src/agent/app/teams/coin_intake.py` and `src/api/services/coin_intake_service.go`
- [x] T054 [US1] Detect PWA mode and auto-open agentic intake surface on Add Coin page in `src/web/src/pages/AddCoinPage.vue`
- [x] T055 [US1] Initialize camera-ready observe capture when permission is granted, with upload fallback when not granted, in `src/web/src/pages/AddCoinPage.vue`

**Checkpoint**: User Story 1 works end-to-end with draft generation and persistence.

---

## Phase 4: User Story 2 - Review and edit draft before commit (Priority: P1)

**Goal**: Let users review confidence/evidence and edit draft fields before saving.

**Independent Test**: Open review UI from draft response, inspect evidence/confidence, edit fields, and stage override payload.

### Implementation for User Story 2

- [x] T028 [US2] Build intake review panel for confidence and evidence rendering in `src/web/src/components/coin/CoinIntakeReviewPanel.vue`
- [x] T029 [US2] Implement draft-to-form mapper and editable override state in `src/web/src/composables/useCoinIntake.ts`
- [x] T030 [US2] Integrate intake review panel with Add Coin page form workflow in `src/web/src/pages/AddCoinPage.vue`
- [x] T031 [P] [US2] Add field-level uncertainty and evidence display props in `src/web/src/types/index.ts`
- [x] T032 [US2] Return uncertain-field metadata from Go draft response mapping in `src/api/services/coin_intake_service.go`
- [x] T033 [US2] Surface low-confidence markers and evidence details in `src/web/src/components/coin/CoinIntakeReviewPanel.vue`
- [x] T034 [US2] Ensure cancel/close review path does not call commit endpoint in `src/web/src/pages/AddCoinPage.vue`
- [x] T051 [US2] Add explicit manual-entry bypass mode on Add Coin page that skips intake review/commit workflow in `src/web/src/pages/AddCoinPage.vue`
- [x] T056 [US2] Add `Use Manual Mode instead` link beneath PWA camera view and route to manual flow in `src/web/src/pages/AddCoinPage.vue`

**Checkpoint**: User can fully review and edit draft content before commit.

---

## Phase 5: User Story 3 - Confirm draft to create coin + AI journal tag (Priority: P1)

**Goal**: Persist a coin only after explicit confirmation and create an audit journal entry tagged with AI intake source.

**Independent Test**: Confirm edited draft, verify new coin is created, draft status moves to confirmed, and journal entry records AI-assisted source.

### Implementation for User Story 3

- [x] T035 [US3] Implement `POST /api/coins/intake/commit` handler binding and deterministic error mapping in `src/api/handlers/coin_intake.go`
- [x] T036 [US3] Implement transactional commit flow (ownership check, draft lock, coin create, status update) in `src/api/services/coin_intake_service.go`
- [x] T037 [US3] Implement draft lifecycle transitions (`drafted -> confirmed|discarded|expired`) in `src/api/repository/coin_intake_draft_repository.go`
- [x] T038 [US3] Record AI source-tagged coin journal entry (`coin_intake`) on successful commit in `src/api/services/coin_intake_service.go`
- [x] T039 [US3] Register authenticated intake commit route in `src/api/main.go`
- [x] T040 [US3] Guard duplicate commit attempts and return conflict semantics in `src/api/services/coin_intake_service.go`
- [x] T041 [P] [US3] Implement frontend commit payload submission and confirm flag enforcement in `src/web/src/composables/useCoinIntake.ts`
- [x] T042 [US3] Complete post-commit UX (success, reset draft state, redirect to detail) in `src/web/src/pages/AddCoinPage.vue`

**Checkpoint**: Draft confirm flow persists coin only after explicit user confirmation.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Finalize docs, compatibility notes, and API artifacts.

- [x] T043 [P] Document intake draft/review/confirm UX in `docs/features.md`
- [x] T044 [P] Document intake draft/commit request-response examples in `docs/api-reference.md`
- [x] T045 Add feature-specific validation notes for issue #216 in `specs/216-ai-intake-draft-confirm-coin-creation/quickstart.md`
- [x] T046 Preserve manual add-coin compatibility notes in `src/web/src/pages/AddCoinPage.vue`
- [x] T047 Regenerate OpenAPI artifacts via project task workflow in `src/api/docs/swagger.yaml`
- [x] T052 Validate desktop-focused manual bypass UX copy/affordance in `src/web/src/pages/AddCoinPage.vue`
- [x] T057 Validate PWA camera-first default and manual-link fallback behavior in `src/web/src/pages/AddCoinPage.vue`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1 (Setup)**: No dependencies.
- **Phase 2 (Foundational)**: Depends on Phase 1; blocks all story work.
- **Phase 3 (US1)**: Depends on Phase 2; MVP slice.
- **Phase 4 (US2)**: Depends on US1 draft payload availability.
- **Phase 5 (US3)**: Depends on US1 and US2 review/override payload readiness.
- **Phase 6 (Polish)**: Depends on completion of desired stories.

### External Dependency Note

- Soft dependency on Issue **#215** (structured references): intake draft should map references when available but must not block core intake draft/confirm flow.

### User Story Dependency Graph

```text
Setup -> Foundational -> US1 (Draft generation) -> US2 (Review/edit) -> US3 (Confirm commit) -> Polish
```

### Within-Story Ordering Rules

- Agent extraction + schema work before Go draft endpoint wiring.
- Repository/service lifecycle handling before commit route exposure.
- API/composable wiring before page-level integration.
- Commit transaction + journal work before post-commit UX wiring.

---

## Parallel Execution Examples

### User Story 1 (US1)

```bash
Task T019: Implement coin_intake pipeline in src/agent/app/teams/coin_intake.py
Task T022: Implement draft endpoint handler in src/api/handlers/coin_intake.go
Task T025: Implement draft state machine in src/web/src/composables/useCoinIntake.ts
```

### User Story 2 (US2)

```bash
Task T028: Build CoinIntakeReviewPanel.vue
Task T031: Add uncertainty/evidence view types in src/web/src/types/index.ts
Task T032: Return uncertain metadata from src/api/services/coin_intake_service.go
```

### User Story 3 (US3)

```bash
Task T036: Implement commit transaction in src/api/services/coin_intake_service.go
Task T037: Implement draft status transitions in src/api/repository/coin_intake_draft_repository.go
Task T041: Implement frontend commit submission in src/web/src/composables/useCoinIntake.ts
```

---

## Implementation Strategy

### MVP First (US1 only)

1. Complete Setup and Foundational phases.
2. Deliver US1 draft-generation end-to-end.
3. Validate structured draft output (confidence + evidence + unresolved fields) before moving to edit/commit.

### Incremental Delivery

1. Ship US1 (draft generation + persisted draft record).
2. Ship US2 (review/edit UX with uncertainty cues).
3. Ship US3 (explicit confirm commit + journal source tag).
4. Finish with docs and OpenAPI regeneration.

### Squad Parallelization

1. **Backend/API**: T009-T024, T035-T040, T047.
2. **Agent**: T012-T015, T019-T020.
3. **Frontend**: T017-T018, T025-T034, T041-T042.
