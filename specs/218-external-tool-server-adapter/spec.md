# Feature Specification: External Tool Server Adapter With Read/Write Parity

**Feature Branch**: `218-external-tool-server-adapter`
**Created**: 2026-06-01
**Status**: Draft
**Input**: GitHub issue #218 (Epic F012, Card 4)

## Summary

Re-expose the existing transport-agnostic collection tool layer (issue #217) to
external clients over a stable, versioned, OpenAPI-first HTTP surface, with full
read/write parity to the in-app collection chat. External access is authenticated
with existing per-user API keys extended with read/write capability scopes, gated
by an admin kill switch, rate-limited per key, and journaled. Write parity is
preserved via the same two-phase `propose_update` / `commit_update` token flow.
MCP compatibility is achieved by documenting how to wrap the served OpenAPI spec
with an existing proxy (e.g. `mcpo`); no first-party MCP server ships in v1.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Read my collection from an external client (Priority: P1)

As a collector, I want to query my own collection from an external LLM client
(OpenWebUI/Ollama, LibreChat, n8n) using an API key so I can ask questions about
my holdings outside the app.

**Why this priority**: Read parity is the foundational external value and the
prerequisite for safe external writes.

**Independent Test**: Generate a read-scoped API key, import the served OpenAPI
spec into an external client, and run `search_my_collection`, `get_coin`,
`collection_summary`, and `top_coins_by_value` returning only the key owner's data.

**Acceptance Scenarios**:

1. **Given** a valid read-scoped API key, **When** an external client calls a read
   tool, **Then** the response contains only the key owner's user-scoped data.
2. **Given** an API key for user A, **When** the client requests a coin owned by
   user B, **Then** the server denies the request (not found / forbidden) with no
   cross-user data leak.
3. **Given** the admin kill switch `ExternalToolServerEnabled` is off, **When** any
   external tool endpoint is called, **Then** the server returns a disabled error
   and performs no operation.

---

### User Story 2 - Propose and explicitly commit an external write (Priority: P1)

As a collector, I want external write requests to use the same two-phase
proposal+confirm flow so conversational updates from external clients never
auto-write.

**Why this priority**: Confirm-gated, journaled writes are the core safety and
parity requirement in issue #218.

**Independent Test**: With a write-scoped API key, call `propose_update` to receive
a proposal id + token + preview, then call `commit_update` with the token and
`confirm=true`; verify a single persisted write and a journal entry tagged
`external_tool_server`.

**Acceptance Scenarios**:

1. **Given** a write-scoped key and an allowlisted field change, **When**
   `propose_update` runs, **Then** the response returns a proposal preview with
   token and expiry, and no coin write has occurred yet.
2. **Given** a valid proposal token and `confirm=true`, **When** `commit_update`
   runs, **Then** the change persists and a journal entry is recorded with source
   `external_tool_server`, including the originating API key id/name and capability.
3. **Given** an invalid, expired, or already-used proposal token, **When** commit is
   attempted, **Then** the API rejects it and no coin write occurs.
4. **Given** a read-only API key, **When** `propose_update` or `commit_update` is
   called, **Then** the server denies the request for insufficient capability.

---

### User Story 3 - Discover and configure the tool server (Priority: P2)

As a user, I want to create a scoped API key and import the tool server into my
external client with documented steps, so I can set it up without guesswork.

**Why this priority**: Discoverability and key management make the feature usable;
without them parity exists but is unreachable.

**Independent Test**: From Settings, create an API key choosing read or read+write
scope, fetch the served scoped OpenAPI document, and follow the docs to connect
OpenWebUI/Ollama, LibreChat, and n8n.

**Acceptance Scenarios**:

1. **Given** the API key management UI, **When** the user creates a key, **Then**
   they can choose a read-only or read+write scope, and new keys default to
   read-only.
2. **Given** a running server, **When** a client fetches the scoped OpenAPI URL,
   **Then** it receives a valid OpenAPI document describing only the `/v1/tools/*`
   surface, suitable for client auto-import.
3. **Given** the published documentation, **When** a user follows the OpenWebUI/
   LibreChat/n8n and `mcpo` instructions, **Then** the tools appear and operate
   against their collection.

### Edge Cases

- API key for user A attempts to read or update a coin owned by user B.
- Read-only key attempts any write operation.
- Revoked or expired API key is presented.
- Kill switch is toggled off mid-session between propose and commit.
- Proposal created via the in-app surface is committed via the external surface (or
  vice versa) — ownership and token still govern; journal reflects the committing
  surface's source.
- Per-key rate limit is exceeded (read and write buckets).
- Client sends a write to a non-allowlisted identity field (category/era/name).

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST expose the existing collection tool operations
  (`search_my_collection`, `get_coin`, `collection_summary`, `top_coins_by_value`,
  `propose_update`, `commit_update`) to external clients under a versioned route
  group `/api/v1/tools/*`.
- **FR-002**: External tool endpoints MUST authenticate via the existing per-user
  API key (`X-API-Key`) and derive user identity server-side from the key, never
  from client-supplied parameters.
- **FR-003**: System MUST extend the `ApiKey` model with a capability/scope concept
  distinguishing `read` and `write`; new keys MUST default to read-only.
- **FR-004**: Read operations MUST require a key with `read` capability; write
  operations (`propose_update`, `commit_update`) MUST require `write` capability.
- **FR-005**: External writes MUST preserve the two-phase flow: `propose_update`
  returns a proposal id + token + preview; `commit_update` requires the token and
  explicit `confirm=true`.
- **FR-006**: External writes MUST be limited to the same field allowlist as in-app
  collection chat (`grade`, `currentValue`, `notes`, `tags`, `referenceText`,
  `referenceUrl`, `references`); identity fields MUST be rejected.
- **FR-007**: System MUST deny all cross-user access server-side; an API key MUST
  only ever read or write its owner's data.
- **FR-008**: System MUST journal each external commit with source
  `external_tool_server`, including the originating API key id/name and the
  capability used.
- **FR-009**: System MUST gate the entire `/api/v1/tools/*` surface behind an admin
  setting `ExternalToolServerEnabled` that defaults to off (disabled).
- **FR-010**: System MUST apply per-key rate limiting to external tool endpoints,
  configurable independently of and stricter than in-app limits.
- **FR-011**: System MUST serve a dedicated, scoped OpenAPI document for the
  `/v1/tools/*` surface at a stable URL (`/api/v1/tools/openapi.json`) suitable for
  external client auto-import.
- **FR-012**: Read/write behavior over the external surface MUST match the in-app
  collection tool behavior (same operations, scoping, allowlist, and proposal
  lifecycle); no duplicated query/update logic — both surfaces call the same Go
  `CollectionToolsService`.
- **FR-013**: The API key management UI MUST allow choosing read-only or read+write
  scope at key creation and MUST surface each key's scope in the list view.
- **FR-014**: System MUST publish setup/integration documentation covering API key
  creation/scopes, the served OpenAPI URL, `mcpo` (MCP-compatible) wrapping, and
  client walkthroughs for OpenWebUI/Ollama, LibreChat, and n8n.

### Key Entities *(include if feature involves data)*

- **ApiKey (extended)**: Existing per-user, hashed, revocable key gains a
  capability/scope attribute (`read` or `read,write`) controlling allowed operations.
- **CollectionToolsService**: Existing shared, owner-scoped tool layer; the single
  writer reused by both the internal (Python agent) and external adapters.
- **CollectionUpdateProposal**: Existing expiring, token-bearing proposal record;
  reused unchanged for external two-phase writes, with originating source captured
  for journaling.
- **CoinJournal**: Existing audit surface; gains `external_tool_server` source
  entries carrying API key id/name and capability.
- **External Tool OpenAPI Document**: Served, scoped contract describing only the
  `/v1/tools/*` operations for client auto-import.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% of external read responses are scoped to the API key owner's data.
- **SC-002**: 0 successful cross-user reads or writes occur via the external surface.
- **SC-003**: 100% of external committed writes are preceded by a valid proposal and
  explicit `confirm=true`, and are limited to allowlisted fields.
- **SC-004**: 100% of external commits create a `CoinJournal` entry tagged
  `external_tool_server` with API key id/name and capability recorded.
- **SC-005**: 100% of write attempts with a read-only key are denied.
- **SC-006**: With `ExternalToolServerEnabled` off, 100% of `/v1/tools/*` calls are
  rejected with no side effects.
- **SC-007**: The served OpenAPI document imports successfully into OpenWebUI,
  LibreChat, and n8n and exposes all six tools.

## Assumptions

- Issue #217's Go `CollectionToolsService` and `CollectionUpdateProposal`
  persistence are landed and are the canonical tool layer (hard dependency).
- The existing API key system (`X-API-Key`, hashed storage, prefix, revocation)
  is the external auth mechanism; only a capability/scope attribute is added.
- v1 ships no first-party MCP server; MCP compatibility is documented via `mcpo`
  (or client-native OpenAPI import).
- The external surface is served by the same Gin process and port as the main API,
  under a new public `/api/v1/tools/*` route group.
- Existing API keys without an explicit scope are treated as read-only by default
  on migration (least privilege).
