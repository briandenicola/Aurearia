---
description: "Task list for issue #218 external tool server adapter with read/write parity"
---

# Tasks: External Tool Server Adapter With Read/Write Parity

**Input**: Design documents from `/specs/218-external-tool-server-adapter/`
**Prerequisites**: `plan.md`, `spec.md`, `research.md`, `data-model.md`, `contracts/external-tool-server.openapi.yaml`, `quickstart.md`
**Hard dependency**: Issue #217 shared tool layer (`CollectionToolsService`, `CollectionUpdateProposal`, `/api/internal/tools/*`) — landed on `beta`.

**Tests**: No explicit TDD requirement in spec.md. Test tasks are OPTIONAL and limited to a few targeted Go unit tests in Polish. Primary validation is the Go architecture test, `go test ./...`, `npm run build`/`lint`, and the `quickstart.md` scenarios.

**Organization**: Tasks are grouped by user story (US1 external read, US2 external write, US3 discovery/UX) so each story can be implemented and validated independently.

## Format: `[ID] [P?] [Story?] Description`

- **[P]**: Can run in parallel (different files, no dependency on incomplete tasks)
- **[Story]**: `[US1]`, `[US2]`, `[US3]` (user-story phases only; setup/foundational/polish have no story label)
- All descriptions include exact file paths.

## Path Conventions

Web app: Go API in `src/api/`, Vue frontend in `src/web/`, docs in `docs/`.

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Confirm prerequisites and stage the additive schema change.

- [ ] T001 Verify the #217 tool layer is present and exported (`SearchMyCollection`, `GetCoin`, `CollectionSummary`, `TopCoinsByValue`, `ProposeUpdate`, `CommitProposal`) in `src/api/services/collection_tools_service.go`; note the current hardcoded `JournalSource: "collection_chat"`.
- [ ] T002 Confirm the existing API-key auth path (`X-API-Key` → `userId`) and key model fields in `src/api/middleware/auth.go` and `src/api/models/api_key.go`.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Capability model, kill switch, middleware, rate limiting, and the external route-group skeleton that ALL stories depend on.

**⚠️ CRITICAL**: No user-story work can begin until this phase is complete.

- [ ] T003 Add a `Capabilities` string field (default `read`) with `HasRead()`/`HasWrite()` helpers to `src/api/models/api_key.go`.
- [ ] T004 Update `AutoMigrate` so the `api_keys.capabilities` column is added with default `'read'` and existing rows backfill to read-only, in `src/api/database/database.go`.
- [ ] T005 [P] Add capability persistence + create-with-validated-scope (default `read`) and capability lookups to `src/api/repository/api_key_repository.go`.
- [ ] T006 [P] Add `SettingExternalToolServerEnabled = "ExternalToolServerEnabled"` (default `"false"`) to the settings constants/defaults in `src/api/services/settings_service.go`.
- [ ] T007 Extend the API-key auth path to set resolved capabilities in the Gin context alongside `userId`, in `src/api/middleware/auth.go`.
- [ ] T008 [P] Add `RequireCapability(scope string)` middleware returning 403 on insufficient capability in `src/api/middleware/capability.go`.
- [ ] T009 [P] Add an `ExternalToolServerEnabled` gate middleware returning 503 when disabled (reads `settingsSvc`) in `src/api/middleware/external_tools_gate.go`.
- [ ] T010 [P] Add a per-key external rate limiter (separate, stricter read vs write buckets) in `src/api/middleware/ratelimit.go`.
- [ ] T011 Register the public route group `api.Group("/v1/tools")` with the chain gate(T009) → `AuthRequired` (X-API-Key) → external rate limiter(T010) in `src/api/main.go` (routes added per story).

**Checkpoint**: Capability-aware, kill-switch-gated, rate-limited external surface skeleton is ready.

---

## Phase 3: User Story 1 - External read from an external client (Priority: P1) 🎯 MVP

**Goal**: A read-scoped API key can query the owner's collection (`search_my_collection`, `get_coin`, `collection_summary`, `top_coins_by_value`) from an external client over `/api/v1/tools/*`.

**Independent Test**: With a read-only key, call each read endpoint and confirm responses contain only the key owner's data; a cross-user `coin_id` is denied; calls fail with 503 when the kill switch is off.

### Implementation for User Story 1

- [ ] T012 [US1] Create `external_tools.go` with read handlers (`SearchMyCollection`, `GetCoin`, `CollectionSummary`, `TopCoinsByValue`) delegating to `CollectionToolsService`, deriving `userId` from context, in `src/api/handlers/external_tools.go`.
- [ ] T013 [US1] Wire the four read routes under `/api/v1/tools` with `RequireCapability("read")` (Swagger annotations on each) in `src/api/main.go`.
- [ ] T014 [US1] Add request binding, deterministic error responses (400/401/403/404/503), and logging for the read handlers in `src/api/handlers/external_tools.go`.

**Checkpoint**: External read parity works end-to-end with a read-only key (MVP).

---

## Phase 4: User Story 2 - Propose and explicitly commit an external write (Priority: P1)

**Goal**: A write-scoped API key can stage (`propose_update`) and confirm (`commit_update`, `confirm=true`) an allowlisted change; the commit is journaled with source `external_tool_server` plus the key id/name and capability.

**Independent Test**: With a write key, propose → receive token/preview (no write yet) → commit with `confirm=true` → change persists and a `CoinJournal` entry tagged `external_tool_server` is created. A read-only key is denied (403); non-allowlisted fields are rejected (400); replayed/expired tokens are rejected (409/401).

### Implementation for User Story 2

- [ ] T015 [US2] Thread a journal-source argument (and originating actor metadata) through the commit path so external commits journal `external_tool_server` while the internal path keeps `collection_chat`, in `src/api/services/collection_tools_service.go`; update internal callers in `src/api/handlers/internal_tools.go`.
- [ ] T016 [US2] Add `ProposeUpdate` and `CommitUpdate` handlers (require `write` capability; pass `external_tool_server` source + key id/name/capability on commit) to `src/api/handlers/external_tools.go`.
- [ ] T017 [US2] Wire the two write routes under `/api/v1/tools` with `RequireCapability("write")` (Swagger annotations) in `src/api/main.go`.
- [ ] T018 [US2] Ensure the commit journal entry records the originating API key id/name and capability (reuse the in-app field-diff entry builder) in `src/api/services/collection_tools_service.go`.

**Checkpoint**: External write parity (two-phase, allowlisted, journaled) works independently alongside US1.

---

## Phase 5: User Story 3 - Discover and configure the tool server (Priority: P2)

**Goal**: A user can create a read-only or read+write scoped API key from Settings and import the served scoped OpenAPI document into an external client.

**Independent Test**: Create a key in Settings choosing a scope (default read-only) and see its scope in the list; fetch `/api/v1/tools/openapi.json` and confirm a valid OpenAPI document describing all six tools.

### Implementation for User Story 3

- [ ] T019 [P] [US3] Add an `openapi.json` handler that serves the scoped external OpenAPI document for `/v1/tools/*` in `src/api/handlers/external_tools_openapi.go`.
- [ ] T020 [US3] Wire unauthenticated `GET /api/v1/tools/openapi.json` to the served-spec handler in `src/api/main.go`.
- [ ] T021 [US3] Accept an optional validated `scope` (default read-only) on key creation and include `capabilities` in list responses in `src/api/handlers/api_keys.go`.
- [ ] T022 [P] [US3] Add `capabilities`/scope to the `ApiKey` type and the create-key request payload in `src/web/src/types/index.ts` and `src/web/src/api/client.ts`.
- [ ] T023 [US3] Add a read-only vs read+write scope selector to the API-key creation UI and display each key's scope in the list, in the API-key management component under `src/web/src/components/settings/`.

**Checkpoint**: All three user stories are independently functional.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Documentation, contract sync, validation, and optional targeted tests.

- [ ] T024 [P] Create `docs/external-tool-server.md`: enabling the admin toggle, creating scoped keys, the served OpenAPI URL, `mcpo` (MCP-compatible) wrapping, and OpenWebUI/Ollama, LibreChat, and n8n walkthroughs.
- [ ] T025 [P] Mention the external tool server in `docs/features.md` and add the `/v1/tools/*` surface + scoped OpenAPI URL to `docs/api-reference.md`.
- [ ] T026 [P] Update `docs/threat-model.md` for the external surface (API-key write path, capability scopes, default-off toggle, per-key rate limiting, journaling).
- [ ] T027 [P] Add an optional Go unit test for `RequireCapability` (read vs write, missing capability) in `src/api/middleware/capability_test.go`.
- [ ] T028 Regenerate OpenAPI artifacts via `task openapi` and confirm `docs/openapi.json` is in sync.
- [ ] T029 Run `go build ./... && go vet ./... && go test ./...` from `src/api/` (architecture test handlers→services→repository must pass).
- [ ] T030 [P] Run `npm run build && npm run lint` from `src/web/`.
- [ ] T031 Execute `quickstart.md` scenarios A–C and negative cases N1–N6; confirm SC-001…SC-007.

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — start immediately.
- **Foundational (Phase 2)**: Depends on Setup; BLOCKS all user stories.
- **User Stories (Phase 3–5)**: All depend on Foundational. US1 and US2 are both P1; US2's journal-source threading (T015) is independent of US1 and can proceed in parallel. US3 (P2) depends only on Foundational (plus the served spec mirroring routes added in US1/US2).
- **Polish (Phase 6)**: Depends on the targeted stories being complete.

### User Story Dependencies

- **US1 (P1)**: After Foundational. No dependency on other stories.
- **US2 (P1)**: After Foundational. Independent of US1 (different handlers/routes); shares the route group skeleton.
- **US3 (P2)**: After Foundational. The served OpenAPI doc should describe all six tools, so finalize it after US1/US2 routes exist; key-scope UI is independent.

### Within Each User Story

- Service/source changes before handlers; handlers before route wiring; validation/logging last.

### Parallel Opportunities

- Foundational [P] tasks: T005, T006, T008, T009, T010 (distinct files) after T003/T004 land the model/migration.
- US2 T015 (service source threading) can run while US1 handlers are built.
- US3 T019 and T022 are independent of each other and of the write path.
- All Polish docs (T024–T026) and T027/T030 in parallel.

---

## Parallel Example: Foundational Phase

```bash
# After T003 (model) + T004 (migration) land, run in parallel:
Task: "Repo capability support in src/api/repository/api_key_repository.go"      # T005
Task: "ExternalToolServerEnabled setting in src/api/services/settings_service.go" # T006
Task: "RequireCapability middleware in src/api/middleware/capability.go"          # T008
Task: "Kill-switch gate middleware in src/api/middleware/external_tools_gate.go"  # T009
Task: "Per-key external rate limiter in src/api/middleware/ratelimit.go"          # T010
```

---

## Implementation Strategy

### MVP First (User Story 1 only)

1. Phase 1: Setup.
2. Phase 2: Foundational (CRITICAL — blocks all stories).
3. Phase 3: US1 external read.
4. **STOP and VALIDATE**: read parity, cross-user denial, kill switch (quickstart Scenario A, N2, N5).
5. Enable the admin toggle in a test env and demo.

### Incremental Delivery

1. Setup + Foundational → skeleton ready.
2. US1 (read) → validate → demo (MVP).
3. US2 (write) → validate two-phase commit + journaling → demo.
4. US3 (scoped-key UX + served spec) → validate client import → demo.
5. Polish: docs, threat model, contract sync, full quality gate.

---

## Notes

- [P] = different files, no dependency on incomplete tasks.
- The only schema change is the additive `ApiKey.Capabilities` column (least-privilege default).
- No collection read/write logic is duplicated: both adapters call `CollectionToolsService` (epic principle "one tool layer, many adapters").
- Commit after each task or logical group; run the §17 Quality Gate before declaring done.
