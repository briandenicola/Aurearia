---
description: "Task breakdown for issue #216 AI intake draft + confirm coin creation"
---

# Tasks: AI Intake Draft + Confirm Coin Creation

**Input**: Design documents from `/specs/012-agentic-collection-updates/` and GitHub issue `#216`  
**Prerequisites**: `plan.md`, `spec.md`, `data-model.md`, `contracts/collection-tools.openapi.yaml`, `research.md`

**Tests**: No explicit TDD requirement in the issue/spec. Tasks focus on implementation slices with independent story-level validation.

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Create scaffolding for intake draft/confirm flow across API, agent, and web.

- [ ] T001 Create intake draft model scaffold in `src/api/models/coin_intake_draft.go`
- [ ] T002 [P] Create intake draft repository scaffold in `src/api/repository/coin_intake_draft_repository.go`
- [ ] T003 [P] Create intake service scaffold in `src/api/services/coin_intake_service.go`
- [ ] T004 [P] Create intake handler scaffold in `src/api/handlers/coin_intake.go`
- [ ] T005 [P] Create Python intake team scaffold in `src/agent/app/teams/coin_intake.py`
- [ ] T006 [P] Create web intake composable scaffold in `src/web/src/composables/useCoinIntake.ts`
- [ ] T007 [P] Add intake type placeholders in `src/web/src/types/index.ts`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Build core schemas, persistence, proxy plumbing, and contracts before user stories.

**⚠️ CRITICAL**: Complete this phase before starting user stories.

- [ ] T008 Implement persisted `CoinIntakeDraft` fields (`draftPayload`, `evidence`, `confidenceSummary`, `status`, `expiresAt`) in `src/api/models/coin_intake_draft.go`
- [ ] T009 Register `CoinIntakeDraft` migration in `src/api/database/database.go`
- [ ] T010 Implement intake draft repository methods (`CreateDraft`, `FindDraftByID`, `UpdateStatus`, `MarkExpired`) in `src/api/repository/coin_intake_draft_repository.go`
- [ ] T011 [P] Extend intake proxy request/response structs and HTTP methods in `src/api/services/agent_proxy.go`
- [ ] T012 [P] Add intake request schemas to agent models in `src/agent/app/models/requests.py`
- [ ] T013 [P] Add intake response schemas (`confidence`, `evidence`, `uncertainties`) in `src/agent/app/models/responses.py`
- [ ] T014 Implement `/api/intake/draft` route wiring in `src/agent/app/routes.py`
- [ ] T015 Implement typed web intake API methods in `src/web/src/api/client.ts`
- [ ] T016 [P] Add frontend `CoinIntakeDraft`/`CoinIntakeCommitRequest` interfaces in `src/web/src/types/index.ts`
- [ ] T017 Add intake Swagger DTOs in `src/api/handlers/swagger_types.go`

**Checkpoint**: Intake domain primitives and contracts are in place.

---

## Phase 3: User Story 1 - Generate structured AI intake drafts (Priority: P1) 🎯 MVP

**Goal**: Produce a structured draft from photos/OCR/lookups with confidence and source evidence.

**Independent Test**: Submit intake prompt + images, receive structured draft payload with confidence bucket and evidence items.

### Implementation for User Story 1

- [ ] T018 [US1] Implement `coin_intake` team pipeline (OCR/lookups/field extraction/evidence capture) in `src/agent/app/teams/coin_intake.py`
- [ ] T019 [US1] Invoke intake team from route handler and return typed payload in `src/agent/app/routes.py`
- [ ] T020 [US1] Implement draft orchestration service (proxy call + draft persistence) in `src/api/services/coin_intake_service.go`
- [ ] T021 [US1] Implement `POST /api/coins/intake/draft` handler validation and mapping in `src/api/handlers/coin_intake.go`
- [ ] T022 [US1] Register protected intake draft route in `src/api/main.go`
- [ ] T023 [US1] Add draft endpoint Swagger annotations and schema references in `src/api/handlers/coin_intake.go`
- [ ] T024 [P] [US1] Implement draft-request flow in `src/web/src/composables/useCoinIntake.ts`
- [ ] T025 [US1] Add “Generate AI Draft” flow entry in `src/web/src/pages/AddCoinPage.vue`

**Checkpoint**: User Story 1 works end-to-end with draft generation and persistence.

---

## Phase 4: User Story 2 - Review and edit draft before commit (Priority: P1)

**Goal**: Let users review confidence/evidence and edit draft fields before saving.

**Independent Test**: Open review UI from draft response, inspect evidence/confidence, edit fields, and stage override payload.

### Implementation for User Story 2

- [ ] T026 [US2] Build intake review panel for confidence + evidence rendering in `src/web/src/components/coin/CoinIntakeReviewPanel.vue`
- [ ] T027 [US2] Implement draft-to-form mapper and override state management in `src/web/src/composables/useCoinIntake.ts`
- [ ] T028 [US2] Integrate review panel with editable `CoinForm` workflow in `src/web/src/pages/AddCoinPage.vue`
- [ ] T029 [P] [US2] Add uncertainty/evidence typing for review rendering in `src/web/src/types/index.ts`
- [ ] T030 [US2] Return uncertain-field metadata from API draft response in `src/api/services/coin_intake_service.go` and `src/api/handlers/coin_intake.go`
- [ ] T031 [US2] Surface low-confidence indicators in form review UI in `src/web/src/components/coin/CoinIntakeReviewPanel.vue` and `src/web/src/components/CoinForm.vue`

**Checkpoint**: User can fully review and edit draft content before commit.

---

## Phase 5: User Story 3 - Confirm draft to create coin + AI journal tag (Priority: P1)

**Goal**: Persist a coin only after explicit confirmation and create an audit journal entry tagged with AI intake source.

**Independent Test**: Confirm edited draft, verify new coin is created, draft status moves to confirmed, and journal entry records AI-assisted source.

### Implementation for User Story 3

- [ ] T032 [US3] Implement `POST /api/coins/intake/commit` handler request binding and responses in `src/api/handlers/coin_intake.go`
- [ ] T033 [US3] Implement commit transaction (`draft` lock, apply overrides, create coin) in `src/api/services/coin_intake_service.go`
- [ ] T034 [US3] Implement draft status transitions (`drafted -> confirmed|discarded|expired`) in `src/api/repository/coin_intake_draft_repository.go`
- [ ] T035 [US3] Write AI source-tagged coin journal entry on successful commit in `src/api/services/coin_intake_service.go`
- [ ] T036 [US3] Register protected intake commit route in `src/api/main.go`
- [ ] T037 [P] [US3] Implement frontend commit API call and override payload submission in `src/web/src/api/client.ts` and `src/web/src/composables/useCoinIntake.ts`
- [ ] T038 [US3] Complete post-commit UX (success state + redirect to coin detail + draft reset) in `src/web/src/pages/AddCoinPage.vue`

**Checkpoint**: Draft confirm flow persists coin only after explicit user confirmation.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Finalize docs, compatibility notes, and API artifacts.

- [ ] T039 [P] Document intake draft/review/confirm UX in `docs/features.md`
- [ ] T040 [P] Document intake draft/commit request-response examples in `docs/api-reference.md`
- [ ] T041 Add issue-216 validation steps to feature quickstart in `specs/012-agentic-collection-updates/quickstart.md`
- [ ] T042 Regenerate OpenAPI artifacts in `src/api/docs/swagger.yaml`, `src/api/docs/swagger.json`, `src/api/docs/docs.go`, and `docs/openapi.json`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1 (Setup)**: No dependencies.
- **Phase 2 (Foundational)**: Depends on Phase 1; blocks all story work.
- **Phase 3 (US1)**: Depends on Phase 2; MVP slice.
- **Phase 4 (US2)**: Depends on US1 draft payload availability.
- **Phase 5 (US3)**: Depends on US1+US2 commit payload and review flow.
- **Phase 6 (Polish)**: Depends on completion of desired stories.

### External Dependency Note

- Soft dependency on Issue **#215** (structured references): intake draft should map references when available but must not block core intake draft/confirm flow.

### User Story Dependency Graph

```text
Setup -> Foundational -> US1 (Draft generation) -> US2 (Review/edit) -> US3 (Confirm commit) -> Polish
```

### Within-Story Ordering Rules

- Agent/team schema work before Go handler wiring.
- Go service/repository implementation before route registration.
- API client typing before UI integration.
- Commit transaction logic before UI post-commit behavior.

---

## Parallel Execution Examples

### User Story 1 (US1)

```bash
Task T018: Implement coin_intake team in src/agent/app/teams/coin_intake.py
Task T021: Implement draft endpoint handler in src/api/handlers/coin_intake.go
Task T024: Implement draft request composable in src/web/src/composables/useCoinIntake.ts
```

### User Story 2 (US2)

```bash
Task T026: Build CoinIntakeReviewPanel.vue
Task T029: Add uncertainty/evidence types in src/web/src/types/index.ts
Task T030: Return uncertain-field metadata from Go intake service/handler
```

### User Story 3 (US3)

```bash
Task T033: Implement commit transaction in src/api/services/coin_intake_service.go
Task T034: Implement draft status transitions in src/api/repository/coin_intake_draft_repository.go
Task T037: Implement frontend commit payload submission in client.ts/useCoinIntake.ts
```

---

## Implementation Strategy

### MVP First (US1 only)

1. Complete Setup and Foundational phases.
2. Deliver US1 draft-generation end-to-end.
3. Validate structured draft output (confidence + evidence) before moving to edit/commit.

### Incremental Delivery

1. Ship US1 (draft generation + persisted draft record).
2. Ship US2 (review/edit UX with uncertainty cues).
3. Ship US3 (explicit confirm commit + journal source tag).
4. Finish with docs and OpenAPI regeneration.

### Squad Parallelization

1. **Backend/API**: T008-T023, T032-T036, T042.
2. **Agent**: T012-T014, T018-T019.
3. **Frontend**: T015-T016, T024-T031, T037-T038.
