---
description: "Task breakdown for issue #218 external tool server adapter with read/write parity"
---

# Tasks: External Tool Server Adapter With Read/Write Parity

**Input**: Design documents from `/specs/218-external-tool-server-adapter/`
**Prerequisites**: `plan.md`, `spec.md`, `research.md`, `data-model.md`, `contracts/external-tool-server.openapi.yaml`, `quickstart.md`
**Hard dependency**: Issue #217 shared tool layer (`CollectionToolsService`, `CollectionUpdateProposal`, `/api/internal/tools/*`) — landed on `beta`.

**Tests**: No explicit TDD requirement in the spec. Validation is via the Go architecture test, `go test ./...`, `npm run build`/`lint`, and the `quickstart.md` scenarios. Add targeted unit tests where noted.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependency on incomplete tasks)
- **[Story]**: `[US1]` external read, `[US2]` external write, `[US3]` discovery/UX
- All task descriptions include concrete file paths.

## Phase 1: Setup — capability model & kill switch (shared infrastructure)

- [ ] T001 Extend `ApiKey` with a `Capabilities` field (string; default `read`) plus `HasRead()`/`HasWrite()` helpers in `src/api/models/api_key.go`.
- [ ] T002 Update `AutoMigrate` and confirm the additive column (default `'read'`, backfill existing rows to read-only) in `src/api/database/database.go`.
- [ ] T003 [P] Add `SettingExternalToolServerEnabled = "ExternalToolServerEnabled"` with default `"false"` to the settings constants/defaults in `src/api/services/settings_service.go`.
- [ ] T004 [P] Add repository support to read capabilities and to create keys with a validated scope in `src/api/repository/api_key_repository.go`.

## Phase 2: Auth, capability & kill-switch middleware

- [ ] T005 Ensure API-key auth exposes resolved capabilities in context (extend `authenticateApiKey`/`AuthRequired`) in `src/api/middleware/auth.go`.
- [ ] T006 Add `RequireCapability(scope string)` middleware (403 on insufficient capability) in `src/api/middleware/capability.go`.
- [ ] T007 Add an `ExternalToolServerEnabled` gate middleware (503 when disabled) reading `settingsSvc`, in `src/api/middleware/` (e.g. `external_tools_gate.go`).
- [ ] T008 [P] Add a per-key external rate limiter (separate/stricter buckets for read vs write) in `src/api/middleware/ratelimit.go`.

## Phase 3: External tool handlers (US1 read + US2 write)

- [ ] T009 [US1] Create `external_tools.go` handlers that delegate to `CollectionToolsService` for `search_my_collection`, `get_coin`, `collection_summary`, `top_coins_by_value` in `src/api/handlers/external_tools.go`.
- [ ] T010 [US2] Add `propose_update` and `commit_update` handlers to `src/api/handlers/external_tools.go`, requiring `write` capability.
- [ ] T011 [US2] Thread a journal source argument through the commit path so external commits journal `external_tool_server` (internal keeps `collection_chat`); record API key id/name + capability. Edit `src/api/services/collection_tools_service.go` (and callers in `src/api/handlers/internal_tools.go`).
- [ ] T012 [US3] Add an `openapi.json` handler that serves the scoped external OpenAPI document in `src/api/handlers/external_tools_openapi.go`.

## Phase 4: Routing & wiring

- [ ] T013 Register the public group `api.Group("/v1/tools")` in `src/api/main.go`, applying: kill-switch gate → `AuthRequired` (X-API-Key) → external rate limiter, then per-route `RequireCapability("read"|"write")`.
- [ ] T014 Wire all six tool routes + `GET /v1/tools/openapi.json` (unauthenticated discovery) to the new handlers in `src/api/main.go`.

## Phase 5: Frontend — API key scope UX (US3)

- [ ] T015 [P][US3] Add `capabilities`/scope to the `ApiKey` type and the create-key payload in `src/web/src/types/index.ts` and `src/web/src/api/client.ts`.
- [ ] T016 [US3] Add a read-only vs read+write scope selector to the API-key creation UI and show each key's scope in the list in the API-key management component under `src/web/src/components/settings/`.

## Phase 6: API key creation — accept scope

- [ ] T017 [US3] Accept an optional validated `scope` on key creation (default read-only) and include capabilities in list responses in `src/api/handlers/api_keys.go`.

## Phase 7: Documentation

- [ ] T018 [P][US3] Create `docs/external-tool-server.md`: enabling the kill switch, creating scoped keys, the served OpenAPI URL, `mcpo` wrapping, and OpenWebUI/Ollama, LibreChat, and n8n walkthroughs.
- [ ] T019 [P] Mention the external tool server in `docs/features.md` and add the `/v1/tools/*` surface + scoped OpenAPI URL to `docs/api-reference.md`.
- [ ] T020 [P] Update `docs/threat-model.md` for the external surface (API-key write path, capability scopes, kill switch, per-key rate limiting, journaling).

## Phase 8: Validation (Quality Gate §17)

- [ ] T021 Regenerate OpenAPI artifacts via `task openapi` and confirm `docs/openapi.json` is in sync.
- [ ] T022 Run `go build ./... && go vet ./... && go test ./...` from `src/api/` (architecture test must still pass: handlers→services→repository).
- [ ] T023 [P] Run `npm run build && npm run lint` from `src/web/`.
- [ ] T024 Execute `quickstart.md` scenarios A–C and negative cases N1–N6; confirm SC-001…SC-007.

## Dependencies

- Phase 1 (T001–T004) before middleware (Phase 2) and handlers (Phase 3).
- T005 before T006 (capability must be in context before it can be required).
- T011 before T010 completes its journal assertion (commit source threading).
- Phases 3–4 before frontend (Phase 5) can be exercised end-to-end.
- Phase 8 last.

## Parallelizable

- T003, T004 with T001/T002 once the model field name is fixed.
- T008 independent of handler work.
- T015 independent of backend once the field name (`capabilities`) is agreed.
- All of Phase 7 (T018–T020) in parallel.
