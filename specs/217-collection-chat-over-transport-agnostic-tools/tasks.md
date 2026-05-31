---
description: "Task breakdown for issue #217 collection chat over transport-agnostic tools"
---

# Tasks: Collection Chat Over Transport-Agnostic Tools

**Input**: Design documents from `/specs/217-collection-chat-over-transport-agnostic-tools/`  
**Prerequisites**: `plan.md`, `spec.md`, `research.md`, `data-model.md`, `contracts/collection-chat.openapi.yaml`, `quickstart.md`

**Tests**: No explicit TDD/test-first requirement in the feature spec; tasks are organized for independent per-story validation via quickstart scenarios.

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Create scaffolds for proposal persistence, collection tool orchestration, and intent-routed chat wiring.

- [ ] T001 Create collection update proposal model scaffold in `src/api/models/collection_update_proposal.go`
- [ ] T002 [P] Create collection update proposal repository scaffold in `src/api/repository/collection_update_repository.go`
- [ ] T003 [P] Create collection tools service scaffold in `src/api/services/collection_tools_service.go`
- [ ] T004 [P] Add collection proposal DTO placeholders in `src/api/handlers/swagger_types.go`
- [ ] T005 [P] Add collection chat request/response placeholders in `src/agent/app/models/requests.py` and `src/agent/app/models/responses.py`
- [ ] T006 [P] Create collection chat team scaffold in `src/agent/app/teams/collection_chat.py`
- [ ] T007 [P] Add collection chat domain placeholders in `src/web/src/types/index.ts`
- [ ] T008 [P] Add collection chat/proposal client method placeholders in `src/web/src/api/client.ts`
- [ ] T009 [P] Add chat intent-routing state scaffold in `src/web/src/composables/useCoinSearchChat.ts`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Implement core persistence, typed contracts, and routing prerequisites required by all user stories.

**⚠️ CRITICAL**: Complete this phase before starting user stories.

- [ ] T010 Implement `CollectionUpdateProposal` fields, status enum, and token hash constraints in `src/api/models/collection_update_proposal.go`
- [ ] T011 Register `CollectionUpdateProposal` migration in `src/api/database/database.go`
- [ ] T012 Implement proposal repository methods (`CreateProposal`, `FindOwnedProposal`, `MarkCommitted`, `MarkCancelled`, `MarkExpired`) in `src/api/repository/collection_update_repository.go`
- [ ] T013 [P] Add owner-scoped selection helpers for collection tool lookups in `src/api/repository/coin_repository.go`
- [ ] T014 [P] Define typed collection tool request/result envelopes in `src/api/services/collection_tools_service.go`
- [ ] T015 [P] Extend Go agent proxy payloads for routing metadata and proposal metadata in `src/api/services/agent_proxy.go`
- [ ] T016 [P] Extend Python route/model plumbing for intent-routing payloads in `src/agent/app/routes.py`, `src/agent/app/models/requests.py`, and `src/agent/app/models/responses.py`
- [ ] T017 [P] Add proposal commit/cancel response DTOs for handler serialization in `src/api/handlers/swagger_types.go`
- [ ] T018 [P] Add frontend collection proposal and disambiguation interfaces in `src/web/src/types/index.ts`

**Checkpoint**: Shared persistence/contracts/routing foundations are ready for independent story work.

---

## Phase 3: User Story 1 - Ask collection questions in chat from owned data only (Priority: P1) 🎯 MVP

**Goal**: Deliver collection-intent read chat grounded in authenticated owner-scoped data using `get_coin`, `query_coins`, and `aggregate`.

**Independent Test**: Open chat from any app page, ask holdings/filter/aggregate questions, and confirm deterministic owner-scoped responses including explicit no-results behavior.

### Implementation for User Story 1

- [ ] T019 [US1] Implement owner-scoped `get_coin`, `query_coins`, and `aggregate` operations in `src/api/services/collection_tools_service.go`
- [ ] T020 [US1] Wire collection read operation execution into agent chat handler flow in `src/api/handlers/agent.go`
- [ ] T021 [US1] Route collection-intent conversations through a dedicated team path in `src/agent/app/routes.py` and `src/agent/app/supervisor.py`
- [ ] T022 [US1] Implement read-only collection chat orchestration in `src/agent/app/teams/collection_chat.py`
- [ ] T023 [P] [US1] Attach app context metadata (route/active coin context when available) to chat requests in `src/web/src/composables/useCoinSearchChat.ts`
- [ ] T024 [US1] Submit intent-routed chat payloads and parse read-result events in `src/web/src/composables/useCoinSearchChat.ts`
- [ ] T025 [US1] Render structured collection read/no-results responses in `src/web/src/components/CoinSearchChat.vue`
- [ ] T026 [US1] Preserve default coin-search behavior for discovery/search intent in `src/web/src/composables/useCoinSearchChat.ts` and `src/api/handlers/agent.go`

**Checkpoint**: User Story 1 is independently functional and testable as the MVP slice.

---

## Phase 4: User Story 2 - Propose and explicitly commit chat-driven updates (Priority: P1)

**Goal**: Enforce two-phase write safety where chat proposes changes and only explicit commit persists mutations.

**Independent Test**: Request a valid update, receive proposal preview + token, commit with explicit confirmation, verify one persisted mutation and one journal entry tagged `collection_chat`; replay/invalid tokens fail without writes.

### Implementation for User Story 2

- [ ] T027 [US2] Implement `propose_update` allowlist validation and proposal token issuance in `src/api/services/collection_tools_service.go`
- [ ] T028 [US2] Implement `commit_update` token verification, expiry/replay checks, and transactional mutation orchestration in `src/api/services/collection_tools_service.go`
- [ ] T029 [US2] Persist commit audit entries with source `collection_chat` in `src/api/repository/journal_repository.go` and `src/api/services/collection_tools_service.go`
- [ ] T030 [US2] Add commit/cancel proposal handlers in `src/api/handlers/agent.go`
- [ ] T031 [US2] Register proposal commit/cancel routes under authenticated API group in `src/api/main.go`
- [ ] T032 [US2] Integrate proposal and commit tool calls into collection chat team flow in `src/agent/app/teams/collection_chat.py`
- [ ] T033 [P] [US2] Implement collection proposal commit/cancel API methods in `src/web/src/api/client.ts`
- [ ] T034 [US2] Implement proposal preview + explicit confirm interaction state in `src/web/src/composables/useCoinSearchChat.ts` and `src/web/src/components/CoinSearchChat.vue`

**Checkpoint**: User Stories 1 and 2 are independently functional with confirm-gated write safety.

---

## Phase 5: User Story 3 - Resolve ambiguous targets and preserve existing chat UX (Priority: P2)

**Goal**: Require disambiguation for non-unique targets while preserving existing drawer UX and deterministic validation errors.

**Independent Test**: In the same global chat surface, issue an ambiguous collection write request, receive disambiguation candidates, resolve selection, then complete confirmed write; non-allowlisted field requests are deterministically rejected.

### Implementation for User Story 3

- [ ] T035 [US3] Implement ambiguous target detection and candidate generation in `src/api/repository/coin_repository.go` and `src/api/services/collection_tools_service.go`
- [ ] T036 [US3] Block proposal creation until disambiguation is resolved and return deterministic validation errors for non-allowlisted fields in `src/api/services/collection_tools_service.go`
- [ ] T037 [US3] Emit structured disambiguation responses and resume proposal flow after user selection in `src/agent/app/teams/collection_chat.py`
- [ ] T038 [US3] Add disambiguation selection handling to chat state machine in `src/web/src/composables/useCoinSearchChat.ts`
- [ ] T039 [US3] Render disambiguation option UI and selection actions in `src/web/src/components/CoinSearchChat.vue`
- [ ] T040 [US3] Preserve existing drawer experience while showing routing/disambiguation feedback in `src/web/src/components/CoinSearchChat.vue`

**Checkpoint**: All user stories are independently functional and UX continuity is preserved.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Finalize docs/contracts and rollout guidance across API, web, and specs.

- [ ] T041 [P] Update chat and proposal endpoint documentation in `docs/api-reference.md` and `docs/openapi.json`
- [ ] T042 [P] Document intent-routed chat UX and safe commit behavior in `docs/features.md`
- [ ] T043 Regenerate Swagger artifacts for handler/schema updates in `src/api/docs/swagger.yaml`, `src/api/docs/swagger.json`, and `src/api/docs/docs.go`
- [ ] T044 Update implementation validation checklist in `specs/217-collection-chat-over-transport-agnostic-tools/quickstart.md`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1 (Setup)**: No dependencies.
- **Phase 2 (Foundational)**: Depends on Phase 1 and blocks all user-story phases.
- **Phase 3 (US1)**: Depends on Phase 2; delivers MVP.
- **Phase 4 (US2)**: Depends on US1 read-flow availability and Phase 2 proposal persistence foundations.
- **Phase 5 (US3)**: Depends on US1 routing/global-chat support and US2 proposal workflow.
- **Phase 6 (Polish)**: Depends on completion of desired stories.

### User Story Dependency Graph

```text
Setup -> Foundational -> US1 (P1 MVP) -> US2 (P1) -> US3 (P2) -> Polish
```

### Within-Story Ordering Rules

- Repository/model prerequisites before service orchestration.
- Service orchestration before handler/route exposure.
- API/client/composable updates before final component rendering work.
- Commit/journal guarantees before UX completion tasks.

---

## Parallel Execution Examples

### User Story 1 (US1)

```bash
Task T023: Attach app context metadata in src/web/src/composables/useCoinSearchChat.ts
Task T021: Route collection-intent conversations in src/agent/app/routes.py and src/agent/app/supervisor.py
Task T019: Implement get_coin/query_coins/aggregate in src/api/services/collection_tools_service.go
```

### User Story 2 (US2)

```bash
Task T033: Implement proposal commit/cancel client methods in src/web/src/api/client.ts
Task T030: Add commit/cancel proposal handlers in src/api/handlers/agent.go
Task T029: Persist collection_chat journal entries in src/api/repository/journal_repository.go
```

### User Story 3 (US3)

```bash
Task T038: Add disambiguation selection handling in src/web/src/composables/useCoinSearchChat.ts
Task T035: Implement ambiguous target detection in src/api/repository/coin_repository.go
Task T037: Emit disambiguation responses in src/agent/app/teams/collection_chat.py
```

---

## Implementation Strategy

### MVP First (User Story 1 only)

1. Complete Phase 1 and Phase 2.
2. Deliver Phase 3 (US1) end-to-end.
3. Validate owner-scoped collection read responses and no-results behavior.
4. Demo MVP collection read-chat before enabling write flows.

### Incremental Delivery

1. Ship US1 (collection read chat).
2. Ship US2 (two-phase propose/commit write flow).
3. Ship US3 (disambiguation + UX continuity guarantees).
4. Finish docs/contract updates in Phase 6.

### Parallel Team Strategy

1. **Backend**: T010-T031 and T035-T036.
2. **Agent/Python**: T015-T016, T021-T022, T032, T037.
3. **Frontend**: T018, T023-T026, T033-T034, T038-T040.
