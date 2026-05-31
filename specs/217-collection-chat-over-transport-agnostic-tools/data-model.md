# Data Model: Collection Chat Over Transport-Agnostic Tools (#217)

## Entity: CollectionToolRequest (internal envelope)

Transport-neutral request envelope passed from chat adapter to Go tool service.

### Fields

- `requestId` (string, required)
- `userId` (uint, required; injected from auth context only)
- `source` (enum: `collection_chat`, required)
- `operation` (enum: `get_coin|query_coins|aggregate|propose_update|commit_update`, required)
- `arguments` (JSON object, required)
- `createdAt` (datetime, required)

### Validation Rules

- `userId` must never be accepted from client payload.
- `arguments` schema must match operation-specific contract.
- Unknown operations are rejected.

## Entity: CollectionQueryResult (computed projection)

Structured response envelope for read-oriented tool operations.

### Fields

- `operation` (enum, required)
- `items` (array, optional)
- `aggregate` (object, optional)
- `pagination` (object, optional)
- `disambiguation` (object, optional; only when target resolution is ambiguous)

### Validation Rules

- Returned data must only include user-owned coins.
- At least one of `items`, `aggregate`, or `disambiguation` must be present.

## Entity: CollectionUpdateProposal (persisted)

Two-phase write proposal created before any mutation is committed.

### Fields

- `id` (string/uuid, primary key)
- `userId` (uint, indexed, required)
- `coinId` (uint, indexed, required)
- `source` (enum: `collection_chat`, required)
- `requestedChanges` (JSON object, required)
- `validatedChanges` (JSON object, required)
- `tokenHash` (string, required)
- `status` (enum: `proposed|committed|cancelled|expired`, required)
- `expiresAt` (datetime, required)
- `createdAt` (datetime, required)
- `updatedAt` (datetime, required)

### Validation Rules

- `validatedChanges` keys must be in v1 allowlist.
- `validatedChanges` must be non-empty.
- `tokenHash` must never be returned in API responses.
- Expired or terminal (`committed|cancelled|expired`) proposals cannot be committed.

## Value Object: CollectionCommitRequest

Explicit user confirmation payload for commit phase.

### Fields

- `proposalId` (string, required)
- `proposalToken` (string, required)
- `confirm` (boolean, required, must be `true`)

### Validation Rules

- Proposal must exist, be owned by authenticated user, and be in `proposed` state.
- `proposalToken` must match hashed token.
- Replay commits against already committed proposal are rejected without new writes.

## Existing Entity Extension: CoinJournal

Committed collection-chat writes append a journal event for auditability.

### Added/Used Semantics

- Journal entry includes source marker `collection_chat`.
- Entry payload includes changed field summary and proposal id.

## State Transitions

### CollectionUpdateProposal Lifecycle

1. `proposed` -> `committed` (valid token + explicit confirm + successful mutation)
2. `proposed` -> `cancelled` (user declines/cancels)
3. `proposed` -> `expired` (TTL exceeded)

### Transition Constraints

- `committed` is terminal and one-time.
- `cancelled` and `expired` proposals cannot transition to `committed`.
- Commit execution is transactional with proposal state update + coin write + journal append.
