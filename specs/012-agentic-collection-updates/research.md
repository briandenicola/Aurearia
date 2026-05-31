# Research — Agentic Collection Updates (Epic F012)

## Decision 1: Use one transport-agnostic Go collection tool layer

- **Decision**: Implement `collection_tools_service` in Go as the single implementation for read (`get_coin`, `query_coins`, `aggregate`) and write (`propose_update`, `commit_update`) operations.
- **Rationale**: This satisfies the epic spine requirement and avoids duplicated business logic between in-app chat and external server adapters.
- **Alternatives considered**:
  - Separate in-app and external tool implementations (rejected: logic drift risk).
  - Tool execution in Python agent (rejected: violates service boundary and writer ownership).

## Decision 2: External protocol strategy is OpenAPI-first with MCP-compatible adapter

- **Decision**: Define canonical contracts as authenticated REST/OpenAPI endpoints in Go, then expose an MCP-compatible adapter surface that maps 1:1 to the same operations.
- **Rationale**: OpenAPI is already native to this repo workflow, while MCP compatibility can be layered without changing core logic.
- **Alternatives considered**:
  - MCP-only first cut (rejected: weak compatibility for non-MCP clients).
  - OpenAPI-only permanently (rejected: limits Claude Desktop-style integrations).

## Decision 3: Reuse existing chat surface with a collection mode

- **Decision**: Add a collection mode in the existing agent chat drawer and route to `collection_chat` team based on intent.
- **Rationale**: Reduces UI fragmentation and reuses current SSE chat infrastructure.
- **Alternatives considered**:
  - New standalone chat page (rejected: duplicate UX and state management).
  - Hidden auto-routing without explicit mode affordance (rejected: poor user clarity).

## Decision 4: Confidence representation uses coarse buckets plus optional numeric score

- **Decision**: Expose `high|medium|low` confidence to users and keep optional numeric confidence for internal ranking/debugging.
- **Rationale**: Buckets are easy to understand; optional numeric values preserve tuning flexibility.
- **Alternatives considered**:
  - Numeric-only confidence (rejected: harder to interpret quickly).
  - Bucket-only internal model (rejected: less diagnostic fidelity).

## Decision 5: All AI/external writes use protocol-enforced two-phase commit

- **Decision**: Require `propose_update` to create an expiring proposal token and `commit_update` to execute with explicit token confirmation.
- **Rationale**: Enforces "agent proposes, user commits" with auditable, deterministic write control.
- **Alternatives considered**:
  - Single-call trusted writes for external tools (rejected: too risky for v1).
  - UI-only confirmation without protocol enforcement (rejected: bypass risk).

## Decision 6: V1 update allowlist excludes identity/structural coin fields

- **Decision**: Allow only `grade`, `currentValue`, `notes`, `tags`, `referenceText`, `referenceURL`, and structured references in v1 conversational/external updates.
- **Rationale**: These are high-value, low-destructive fields suitable for assisted updates.
- **Alternatives considered**:
  - Full field update access (rejected: high integrity risk).
  - Read-only only (rejected: misses core epic write parity goal).

## Decision 7: Structured references coexist with legacy free-text reference fields

- **Decision**: Introduce `CoinReference` model while preserving `Coin.referenceText/referenceURL` for backward compatibility and gradual migration.
- **Rationale**: Preserves current clients and data while enabling normalized catalog workflows.
- **Alternatives considered**:
  - Hard replacement/removal of legacy fields (rejected: breaks compatibility).
  - Keep only free text and parse ad hoc (rejected: weak machine usability).

## Decision 8: Intake outputs must be structured, typed, and source-backed

- **Decision**: `coin_intake` returns structured draft fields, evidence sources, confidence, and unresolved uncertainties (no free-form-only payloads).
- **Rationale**: Aligns with constitution principle for schema-driven interfaces and reliable review UI.
- **Alternatives considered**:
  - Markdown/prose-first draft output (rejected: brittle parsing and UX ambiguity).
  - Auto-commit on high confidence (rejected: violates confirm-gated writes).
