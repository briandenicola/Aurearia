# Data Model: AI Intake Draft + Confirm Coin Creation (#216)

## Entity: CoinIntakeDraft

Persisted AI-generated coin draft prior to user confirmation.

### Fields

- `id` (uint, primary key)
- `userId` (uint, required, indexed)
- `draftPayload` (JSON, required)
- `confidenceSummary` (JSON, required)
- `evidence` (JSON array, required)
- `unresolvedFields` (JSON array, optional)
- `status` (enum: `drafted|confirmed|discarded|expired`, required)
- `expiresAt` (datetime, required)
- `confirmedAt` (datetime, optional)
- `confirmedCoinId` (uint, optional)
- `createdAt` (datetime, required)
- `updatedAt` (datetime, required)

### Validation Rules

- `status` must be one of allowed enum values.
- `draftPayload` must conform to coin intake candidate schema.
- `expiresAt` must be greater than `createdAt`.
- `confirmedCoinId` required when status is `confirmed`.
- Draft owner (`userId`) is immutable after creation.

### Relationships

- `CoinIntakeDraft.userId` -> `User.id` (many drafts per user)
- `CoinIntakeDraft.confirmedCoinId` -> `Coin.id` (optional one-to-one linkage after confirm)

## Value Object: IntakeEvidenceItem

Evidence/provenance attached to draft confidence and review UI.

### Fields

- `type` (enum: `ocr|visual|catalog_lookup|user_input|coin_card`)
- `source` (string, required; tool/source label or URI)
- `field` (string, optional; target coin field)
- `value` (string, optional; extracted/cited value)
- `confidence` (enum: `high|medium|low`, required)
- `notes` (string, optional)

### Validation Rules

- `source` is required and non-empty.
- `confidence` is required and constrained to enum values.
- If `field` is present, it must be a known intake field key.

## Value Object: CoinIntakeDraftRequest

Client payload for AI draft generation.

### Fields

- `images` (array of URIs, required; coin photos)
- `coinCardImage` (URI, optional; dealer/collector card image with metadata)

### Validation Rules

- `images` must contain at least one valid image URI.
- `coinCardImage` is optional, but when provided it must be a valid image URI.

## Value Object: CoinIntakeCommitRequest

Client payload for explicit draft confirmation.

### Fields

- `draftId` (uint, required)
- `overrides` (object, optional but allowed)
- `confirm` (boolean, required and must be `true`)

### Validation Rules

- `draftId` must reference an existing `drafted` status draft owned by authenticated user.
- `confirm=false` is rejected.
- `overrides` keys must be allowlisted to coin-create fields.

## UI Value Object: IntakeEntryState

Client-side entry workflow state for Add Coin intake path selection.

### Fields

- `entryMode` (enum: `agentic|manual`, required)
- `pwaMode` (boolean, required)
- `cameraPermission` (enum: `prompt|granted|denied|unavailable`, required)

### Validation Rules

- In PWA mode, default `entryMode` is `agentic`.
- When `cameraPermission` is `denied|unavailable`, upload intake remains enabled.
- `entryMode=manual` bypasses intake draft/commit operations unless user re-enters agentic mode.

## Existing Entity Extension: CoinJournal

On successful intake commit, append entry with source tag and intake metadata.

### Added/Used Semantics

- `source` includes `coin_intake`.
- Metadata includes `draftId` and summary of overridden fields.

## State Transitions

1. `drafted` -> `confirmed` (commit succeeds; coin created)
2. `drafted` -> `discarded` (user explicitly rejects draft)
3. `drafted` -> `expired` (TTL exceeded before commit)

### Transition Constraints

- `confirmed` is terminal and can occur only once.
- `expired` and `discarded` drafts cannot transition to `confirmed`.
- Duplicate commit attempts after `confirmed` must be idempotent/rejected without extra coin creation.
