# Feature Specification: Collection Chat Over Transport-Agnostic Tools

**Feature Branch**: `217-collection-chat-over-transport-agnostic-tools`  
**Created**: 2026-05-31  
**Status**: Draft  
**Input**: GitHub issue #217

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Ask collection questions in chat from owned data only (Priority: P1)

As a collector, I want to ask questions about my collection in chat so I can quickly inspect holdings, filters, and aggregates without leaving the drawer.

**Why this priority**: Read access is the foundational value for collection chat and the prerequisite for safe write proposals.

**Independent Test**: Open chat from anywhere in the app, ask inventory and aggregate questions, and verify collection-intent prompts are routed to authenticated user-scoped collection tools.

**Acceptance Scenarios**:

1. **Given** an authenticated user with coins, **When** they ask a collection question in chat, **Then** intent routing sends the prompt to `collection_chat` behavior and returns a data-backed answer from `get_coin`, `query_coins`, or `aggregate`.
2. **Given** no matching records for a query, **When** collection chat responds, **Then** it returns a clear no-results response and does not invent coin data.
3. **Given** a prompt attempting cross-user access, **When** collection tools execute, **Then** the request is denied by server-side scope enforcement.

---

### User Story 2 - Propose and explicitly commit chat-driven updates (Priority: P1)

As a collector, I want write requests to produce a proposal token and require explicit confirmation so conversational updates never auto-write.

**Why this priority**: Confirm-gated writes are the core safety requirement in issue #217.

**Independent Test**: Request an update in collection chat, receive proposal preview/token, explicitly commit, and verify a single persisted write plus journal trace.

**Acceptance Scenarios**:

1. **Given** an unambiguous coin target and allowlisted field change, **When** `propose_update` runs, **Then** chat returns a proposal preview with token and expiration.
2. **Given** a valid proposal token and explicit confirmation, **When** `commit_update` runs, **Then** the change persists and a journal entry is recorded with source `collection_chat`.
3. **Given** an invalid, expired, or already-used token, **When** commit is attempted, **Then** the API rejects the commit and no coin write occurs.

---

### User Story 3 - Resolve ambiguous targets and preserve existing chat UX (Priority: P2)

As a collector, I want ambiguous coin references to trigger disambiguation while still using the existing chat drawer experience.

**Why this priority**: Disambiguation and UX continuity prevent unsafe writes and reduce friction in adopting intent-routed collection chat.

**Independent Test**: Use the existing chat drawer from any app page, issue an ambiguous update request, resolve with user selection, and complete a confirmed write.

**Acceptance Scenarios**:

1. **Given** the existing chat drawer, **When** a user sends prompts that are collection-oriented vs market-search-oriented, **Then** intent routing sends each prompt to the appropriate behavior without breaking existing coin-search chat flows.
2. **Given** an ambiguous target (multiple possible coins), **When** a write is requested, **Then** system returns disambiguation choices and withholds proposal creation until resolved.
3. **Given** a write request against non-allowlisted fields, **When** tool validation runs, **Then** system rejects the request with a deterministic safe error.

### Edge Cases

- User asks to update “my Alexander drachm” and multiple matches exist across rulers/mints.
- User attempts commit replay using a consumed token.
- Proposal expires before commit confirmation.
- Requested write mixes allowlisted and non-allowlisted fields.
- Target coin changes (or is deleted) between proposal and commit.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST add a transport-agnostic Go collection tool layer with operations `get_coin`, `query_coins`, `aggregate`, `propose_update`, and `commit_update`.
- **FR-002**: System MUST enforce authenticated user scoping server-side for all collection tool operations.
- **FR-003**: System MUST keep chat app-wide and route each prompt by intent to the appropriate behavior (`collection_chat` for collection prompts, existing coin-search behavior for find-coin prompts).
- **FR-004**: System MUST return structured, deterministic read results for collection queries (single coin, list query, aggregate).
- **FR-005**: System MUST gate write operations with two-phase flow: `propose_update` first, `commit_update` second.
- **FR-006**: System MUST require a proposal token plus explicit confirmation input for `commit_update`.
- **FR-007**: System MUST allow update proposals only for allowlisted fields in v1 (`grade`, `currentValue`, `notes`, `tags`, `referenceText`, `referenceUrl`, `references`).
- **FR-008**: System MUST require disambiguation resolution before proposal creation when target selection is ambiguous.
- **FR-009**: System MUST reject invalid/expired/already-used proposal tokens with deterministic errors and no writes.
- **FR-010**: System MUST journal each successful commit with source tag `collection_chat`.
- **FR-011**: System MUST preserve existing coin-search chat behavior when prompt intent is discovery/search.
- **FR-012**: System MUST present proposal preview details (target coin, field diffs, expiry) before commit.

### Key Entities *(include if feature involves data)*

- **CollectionToolRequest**: Transport-neutral operation envelope with server-injected user identity and typed arguments.
- **CollectionUpdateProposal**: Persisted, expiring proposal record containing validated field changes and token hash.
- **CollectionUpdateCommitRequest**: Explicit confirmation payload referencing proposal id and token for one-time commit.
- **CollectionQueryResult**: Structured read response shape for coin details, list queries, and aggregate outputs.
- **CoinJournal**: Existing audit surface that records committed collection-chat updates.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% of collection chat read responses are scoped to authenticated user-owned data.
- **SC-002**: 100% of committed write operations are preceded by valid proposal creation and explicit confirmation.
- **SC-003**: 0 successful writes occur from invalid, expired, or replayed proposal tokens.
- **SC-004**: 100% of successful collection-chat commits create a coin journal entry tagged `collection_chat`.
- **SC-005**: Existing non-collection chat flows continue functioning without regression.

## Assumptions

- Existing `/api/agent/chat` SSE infrastructure remains the host path, extended with intent-routing semantics.
- Issue #217 scopes in-app collection chat; external tool server parity is handled by issue #218.
- Initial write allowlist is intentionally narrow and can be expanded in later iterations after audit confidence.
