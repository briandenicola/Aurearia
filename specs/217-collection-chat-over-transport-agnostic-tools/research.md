# Research: Collection Chat Over Transport-Agnostic Tools (#217)

## Decision 1: Implement collection tools in Go service layer as the system-of-record

- **Decision**: Build `get_coin`, `query_coins`, `aggregate`, `propose_update`, and `commit_update` in a dedicated Go service layer consumed by agent chat handlers.
- **Rationale**: This preserves Principle I (layered architecture) and Principle III (Go API remains persistence authority) while making the tool layer reusable for external adapters later.
- **Alternatives considered**:
  - Implement tool logic in Python team code (rejected: violates service boundary and DB ownership rules).
  - Duplicate read/write logic per chat adapter (rejected: drift risk and higher maintenance).

## Decision 2: Extend existing chat endpoint with automatic prompt-intent routing

- **Decision**: Keep `/api/agent/chat` as the entry point and classify each prompt intent so backend routes to collection chat behavior or existing coin-search behavior without a manual mode switch.
- **Rationale**: Reuses current SSE pipeline, preserves global chat availability, and aligns routing with natural-language user intent.
- **Alternatives considered**:
  - Create a separate collection-chat endpoint and UI route (rejected: fragmented UX and duplicated chat state).
  - Require explicit UI mode selection (rejected: adds friction and conflicts with desired “chat from anywhere” workflow).

## Decision 3: Enforce protocol-level two-phase writes with short-lived proposal tokens

- **Decision**: `propose_update` returns a single-use token (hashed at rest) and `commit_update` requires proposal id + token + explicit confirm signal.
- **Rationale**: Enforces “agent proposes, user commits” with deterministic safety and auditable write control.
- **Alternatives considered**:
  - Single-step write tool call (rejected: does not satisfy confirm-gated requirement).
  - UI-only confirmation without server token checks (rejected: bypassable).

## Decision 4: Require disambiguation before proposal creation on non-unique targets

- **Decision**: If a write target resolves to multiple coins, return structured disambiguation candidates and block proposal creation until user selects one.
- **Rationale**: Prevents accidental writes to the wrong coin and keeps write flow deterministic.
- **Alternatives considered**:
  - Pick first match silently (rejected: unsafe and non-transparent).
  - Fail immediately without candidate options (rejected: poor UX and extra user friction).

## Decision 5: Use existing CoinJournal as the audit destination for committed updates

- **Decision**: On successful `commit_update`, append a journal entry containing source `collection_chat`, changed fields, and proposal reference.
- **Rationale**: Reuses established activity audit surfaces and fulfills issue acceptance criteria.
- **Alternatives considered**:
  - New collection-chat audit table only (rejected: fragmented history).
  - No journal update for chat writes (rejected: fails explicit acceptance criteria).
