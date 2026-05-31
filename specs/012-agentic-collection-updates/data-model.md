# Data Model — Agentic Collection Updates (Epic F012)

## 1) CoinReference (persisted)

Normalized catalog reference for a coin.

### Fields

- `id` (uint, PK)
- `coinId` (uint, FK to `coins.id`, indexed, required)
- `catalog` (enum/string, required; e.g., `RIC`, `RPC`, `Sear`, `KM`, `Other`)
- `referenceNumber` (string, required)
- `displayLabel` (string, required; canonical label for UI)
- `authorityUri` (string|null, optional)
- `confidence` (enum: `high|medium|low`, optional for AI-populated refs)
- `source` (enum: `manual|coin_intake|collection_chat|external_tool_server`, required)
- `createdAt` / `updatedAt` (timestamps)

### Validation Rules

- Unique composite index (`coin_id`, `catalog`, `reference_number`).
- `catalog`-specific format checks apply when catalog is known.
- `authorityUri` must be absolute URL when present.

## 2) CoinIntakeDraft (persisted short-lived or durable by config)

AI-generated draft before user commit.

### Fields

- `id` (uuid/string, PK)
- `userId` (uint, indexed, required)
- `draftPayload` (JSON, required; typed coin payload candidate)
- `evidence` (JSON array, required; source URLs/OCR fragments)
- `confidenceSummary` (enum `high|medium|low`, required)
- `status` (enum: `drafted|confirmed|discarded|expired`, required)
- `expiresAt` (timestamp, required)
- `createdAt` / `updatedAt` (timestamps)

### Validation Rules

- Draft payload must conform to intake schema contract.
- `confirmed` transition allowed only once.
- Expired drafts cannot transition to `confirmed`.

## 3) CollectionToolRequest (computed/internal)

Transport-neutral internal request envelope for collection tools.

### Fields

- `requestId` (string, required)
- `userId` (uint, required; injected server-side only)
- `operation` (enum: `get_coin|query_coins|aggregate|propose_update|commit_update`)
- `arguments` (JSON object, required)
- `source` (enum: `collection_chat|external_tool_server`)
- `requestedAt` (timestamp, required)

### Validation Rules

- `userId` must come from auth context, never from client payload.
- `arguments` schema depends on `operation`.

## 4) CollectionUpdateProposal (persisted)

Two-phase write proposal created by tool layer.

### Fields

- `id` (uuid/string, PK)
- `userId` (uint, indexed, required)
- `coinId` (uint, indexed, required)
- `source` (enum: `collection_chat|external_tool_server`, required)
- `requestedChanges` (JSON object, required)
- `validatedChanges` (JSON object, required; allowlist-filtered)
- `status` (enum: `proposed|committed|cancelled|expired`, required)
- `proposalTokenHash` (string, required)
- `expiresAt` (timestamp, required)
- `createdAt` / `updatedAt` (timestamps)

### Validation Rules

- Only allowlisted fields may appear in `validatedChanges`.
- Proposal token required for commit.
- Proposal invalid after commit/cancel/expiry.

## 5) CollectionUpdateCommit (persisted/audit projection)

Committed change event for journaling and audit traces.

### Fields

- `id` (uint, PK)
- `proposalId` (string, required)
- `userId` (uint, required)
- `coinId` (uint, required)
- `source` (enum, required)
- `changedFields` (JSON array of field keys, required)
- `beforeSnapshot` (JSON object, optional)
- `afterSnapshot` (JSON object, optional)
- `committedAt` (timestamp, required)

### Validation Rules

- Commit rows must map to a previously `proposed` and now `committed` proposal.
- `changedFields` must be non-empty.

## 6) CollectionQueryResult (computed projection)

Structured response envelope for tool reads.

### Fields

- `items` (array of typed coin summaries/details)
- `aggregates` (optional object for aggregate calls)
- `pagination` (optional metadata)
- `resolvedDisambiguation` (optional object)

### Validation Rules

- All returned coins must belong to authenticated user.
- Output schema must be deterministic and typed.

## 7) ExternalClientCapability (persisted, optional extension of existing API keys)

Capabilities for external tool clients.

### Fields

- `apiKeyId` (uint/string, PK/FK)
- `canReadTools` (bool, required)
- `canWriteTools` (bool, required)
- `requireProposalCommit` (bool, required; true in v1)
- `createdAt` / `updatedAt` (timestamps)

### Validation Rules

- `canWriteTools` implies `canReadTools`.
- `requireProposalCommit` cannot be false in v1.

## Relationships

- `Coin` 1:N `CoinReference`
- `User` 1:N `CoinIntakeDraft`
- `User` 1:N `CollectionUpdateProposal`
- `CollectionUpdateProposal` 1:1..N `CollectionUpdateCommit` (v1 expects one)
- Existing `CoinJournal` records commit audit messages keyed by `coinId`/`userId`

## State Transitions

### Intake Draft Lifecycle

1. `drafted` → `confirmed` (user confirms, coin persisted)
2. `drafted` → `discarded` (user rejects)
3. `drafted` → `expired` (TTL exceeded)

### Proposal/Commit Lifecycle

1. `proposed` (token issued)
2. `committed` (valid token + allowlist-safe mutation applied)
3. `cancelled` (user declines)
4. `expired` (TTL exceeded before commit)
