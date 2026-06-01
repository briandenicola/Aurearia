# Phase 1 Data Model: External Tool Server Adapter (#218)

This feature is mostly additive over issue #217. The only schema change is an
additive capability column on `ApiKey`. All collection read/write logic, the proposal
lifecycle, and journaling are reused.

## 1. ApiKey (extended)

Existing model: `src/api/models/api_key.go`.

| Field | Type | Notes |
|-------|------|-------|
| ID | uint | PK (existing) |
| UserID | uint | owner; source of server-side scoping (existing) |
| KeyHash | string | unique, hashed (existing) |
| KeyPrefix | string | last 8 chars for display (existing) |
| Name | string | user label (existing) |
| CreatedAt | time.Time | (existing) |
| LastUsedAt | *time.Time | (existing) |
| RevokedAt | *time.Time | revocation (existing) |
| **Capabilities** | string | **new** — comma-separated scopes; one of `read` or `read,write`. Default `read`. |

Design notes:

- Store capabilities as a normalized string (`"read"` or `"read,write"`) with helper
  accessors `HasRead()` / `HasWrite()`. Keeps the migration trivial (single TEXT
  column, default `'read'`) and avoids a join table for v1.
- **Migration default**: column default `read`; existing rows backfill to `read`
  (least privilege per R12).
- Creation accepts a requested scope; service validates it to the allowed set and
  rejects unknown scopes. `write` implies `read` (a write key may also read).

### Validation rules

- Allowed scope values: exactly `read` (read-only) or `read,write` (read + write).
- New keys default to read-only when no scope is provided.
- Capability is immutable after creation in v1 (re-issue to change scope; aligns with
  the one-time secret model).

## 2. Capability enforcement (request-time, not persisted)

External requests resolve identity + capability from the API key:

- `AuthRequired` (existing) validates `X-API-Key`, loads the `ApiKey`, and sets
  `userId` in context. Extend it (or a follow-on middleware) to also set the resolved
  capabilities in context.
- A new `RequireCapability("read"|"write")` middleware guards route handlers:
  - read tools require `read`
  - `propose_update` / `commit_update` require `write`
- Missing capability → `403 Forbidden` (insufficient capability), no operation.

## 3. CollectionUpdateProposal (reused, source-aware)

Existing model from issue #217 (`collection_update_proposals` table). No schema change
required for v1. The proposal already carries owner scope, token hash, status, and
expiry.

Behavioral change for journaling parity:

- `CommitProposal` currently writes `CoinJournal` with a hardcoded source
  `collection_chat` and returns `JournalSource: "collection_chat"`.
- Thread a **journal source** (and optional actor metadata) parameter through the
  commit path so the external adapter records `external_tool_server` while the internal
  adapter keeps `collection_chat`.
- Optional (recommended): persist the originating source on the proposal at
  `propose_update` time so the committed journal reflects where the write originated
  even if commit arrives on a different transport. v1 may instead pass the source at
  commit time from the calling handler — either satisfies FR-008 as long as external
  commits journal `external_tool_server`.

## 4. CoinJournal (reused)

Existing audit surface. External commits add entries with:

- `Source` / source tag: `external_tool_server`
- Actor detail: originating API key id and name, and the capability used (`write`).
- Entry text: same field-diff summary builder used for in-app commits.

No schema change is strictly required if actor detail can be embedded in the existing
entry/source fields; if a structured actor column is preferred, that is an additive,
nullable column on `coin_journals` (decide in implementation; entry-embedding is the
lower-risk default).

## 5. Field allowlist (reused, single source of truth)

Enforced once in `CollectionToolsService.ProposeUpdate` (issue #217). External writes
inherit it unchanged:

`grade`, `currentValue`, `notes`, `tags`, `referenceText`, `referenceUrl`,
`references`. Identity fields (`category`, `era`, `name`, `ruler`, `material`, …) are
rejected with `ErrInvalidFieldChanges`.

## 6. Settings (reused pattern)

Add an `AppSetting` key following the existing key-value convention in
`services/settings_service.go`:

| Key | Default | Meaning |
|-----|---------|---------|
| `ExternalToolServerEnabled` | `"false"` | Global kill switch for `/api/v1/tools/*`. When false, all external tool endpoints reject. |

(Mirrors `SettingCoinOfDayEnabled = "CoinOfDayEnabled"` with default `"false"`.)

## 7. Lifecycle / state (external write)

```
client (write-scoped key)
  → POST /api/v1/tools/propose_update {coin_id, changes}
      [kill switch on?] [write capability?] [owner scope] [allowlist]
      → CollectionToolsService.ProposeUpdate(userId, coinId, changes)
      ← {proposalId, proposalToken, changedFields, changes, expiresAt}
  → POST /api/v1/tools/commit_update {proposal_id, token, confirm:true, source:external}
      [kill switch on?] [write capability?] [owner scope] [token valid+pending+unexpired]
      → CollectionToolsService.CommitProposal(userId, proposalId, token, confirm, source=external_tool_server)
      ← {proposalId, status:"committed", coinId, changedFields, journalSource:"external_tool_server"}
      + CoinJournal entry (source external_tool_server, key id/name, capability)
```

Reads (`search_my_collection`, `get_coin`, `collection_summary`,
`top_coins_by_value`) require only `read` capability and the kill switch on; they
return owner-scoped summaries identical to the internal adapter.
