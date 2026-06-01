# Phase 0 Research: External Tool Server Adapter (#218)

All decisions below were confirmed with the product owner during planning and are
consistent with Epic F012 cross-cutting principles and Decision #11 (shared tool
layer). No open `NEEDS CLARIFICATION` items remain.

## R1. Protocol surface — OpenAPI-first, MCP-compatible

- **Decision**: Ship an OpenAPI HTTP tool server now (`/api/v1/tools/*`) plus a
  documented MCP-compatible path via an existing proxy (`mcpo`) or native client
  OpenAPI import. No first-party MCP server code in v1.
- **Rationale**: The named target clients (OpenWebUI/Ollama, LibreChat, n8n) all
  consume OpenAPI tool servers directly. `mcpo` cleanly wraps an OpenAPI server as
  MCP for clients that require it, so parity is reachable without maintaining a
  bespoke MCP server.
- **Alternatives considered**: Native MCP server (stdio + SSE) as primary — more code
  and a second transport to secure for little additional reach in v1. Dual
  first-class MCP + OpenAPI — premature; revisit if a target client drops OpenAPI.

## R2. Hosting — same Gin process, new public route group

- **Decision**: Serve the external surface from the existing Gin server under a new
  public group `/api/v1/tools/*` whose handlers delegate to the existing
  `CollectionToolsService`.
- **Rationale**: Keeps the Go API as the single writer (ADR 0002), avoids duplicating
  query/update logic, and mirrors the internal adapter (`/api/internal/tools/*`).
  Versioned path (`v1`) future-proofs the external contract independently of internal
  refactors.
- **Alternatives considered**: Reusing `/api/internal/tools/*` with API-key auth —
  conflates the HMAC internal-token transport with the external one and muddies the
  threat model. Separate microservice — violates the single-writer boundary and adds
  deployment surface.

## R3. External authentication — existing API keys + capability scopes

- **Decision**: Authenticate with the existing per-user `X-API-Key`. Extend `ApiKey`
  with a capability/scope attribute (`read` or `read,write`). New keys default to
  read-only; the creator opts into write.
- **Rationale**: The API key system already provides hashed storage, prefix display,
  per-user scoping (`userId` set in `AuthRequired`), `LastUsedAt`, and revocation.
  Adding a capability column is the minimal least-privilege extension. Reusing the
  established `userId` context guarantees server-side scoping with no client-supplied
  identity.
- **Alternatives considered**: Single `CanWrite` boolean — workable but a scope list
  extends more cleanly to future capabilities. Per-key per-tool allowlist —
  over-engineered for v1. No new field (full access) — violates least privilege.

## R4. External write confirm model — two-phase parity

- **Decision**: Preserve the in-app two-phase flow: `propose_update` returns a
  proposal id + token + preview; `commit_update` requires the token and explicit
  `confirm=true`.
- **Rationale**: "Agent proposes, user commits" is an epic-wide invariant. Reusing the
  existing `CollectionUpdateProposal` lifecycle and token checks gives parity for free
  and avoids a second, weaker write path.
- **Alternatives considered**: Single-call write gated only by capability — removes the
  confirmation safety net and diverges from in-app behavior. Per-key
  "unattended commit" flag — adds an unsafe bypass; rejected for v1.

## R5. Updatable-field allowlist — identical to in-app

- **Decision**: External writes use exactly the in-app allowlist: `grade`,
  `currentValue`, `notes`, `tags`, `referenceText`, `referenceUrl`, `references`.
  Identity fields (`category`, `era`, `name`, etc.) are rejected.
- **Rationale**: Parity and a single source of truth for the allowlist (enforced once
  in `CollectionToolsService.ProposeUpdate`). Narrow, audit-friendly surface.
- **Alternatives considered**: Broader/narrower external lists — creates divergence and
  two allowlists to maintain; rejected.

## R6. Operation coverage — all six operations

- **Decision**: Expose all six tools externally: `search_my_collection`, `get_coin`,
  `collection_summary`, `top_coins_by_value` (read), `propose_update`,
  `commit_update` (write).
- **Rationale**: The acceptance criteria require read/write parity with the in-app
  tools. The operations already exist and are owner-scoped.
- **Alternatives considered**: Read-only v1 — fails the parity acceptance criterion.

## R7. Auditability — journal source + key identity

- **Decision**: Journal external commits with source `external_tool_server` and record
  the originating API key id/name and capability used. Thread the source into the
  commit path (the existing internal path uses `collection_chat`).
- **Rationale**: Acceptance criteria require `external_tool_server` journaling; adding
  key identity makes external writes traceable to a specific credential for incident
  response (§ threat model).
- **Implementation note**: `CommitProposal` currently hardcodes
  `JournalSource: "collection_chat"`. Parametrize the journal source (and optional
  actor metadata) so the external handler records `external_tool_server`; the internal
  path keeps `collection_chat`.

## R8. Abuse protection — per-key rate limiting

- **Decision**: Apply per-key rate limiting to `/api/v1/tools/*`, configurable
  independently of and stricter than in-app limits. Separate buckets for read vs write
  are desirable.
- **Rationale**: The external surface is internet-facing and API-key-authed; per-key
  limits bound blast radius and abuse without affecting in-app UX.
- **Alternatives considered**: Reuse global limits — too coarse for an external write
  path. No limiting — unacceptable for a public write surface.

## R9. Kill switch — admin setting, default off

- **Decision**: Gate the entire `/api/v1/tools/*` surface behind admin setting
  `ExternalToolServerEnabled`, default `false`. When off, all external tool calls are
  rejected with no side effects.
- **Rationale**: Safe-by-default rollout and an operational off-switch. Mirrors the
  existing key-value `AppSetting` pattern (e.g. `CoinOfDayEnabled` default `"false"`).
- **Alternatives considered**: Write-only flag — leaves reads exposed by default;
  rejected. Always-on — unsafe default for a new external write surface.

## R10. Tool discovery — served scoped OpenAPI

- **Decision**: Serve a dedicated OpenAPI document describing only `/v1/tools/*` at a
  stable URL `/api/v1/tools/openapi.json`, suitable for external client auto-import.
  Commit the contract source under `specs/.../contracts/`.
- **Rationale**: OpenWebUI/LibreChat/n8n import an OpenAPI URL to generate tools; a
  scoped document avoids exposing the full internal Swagger and keeps the external
  contract crisp and versioned.
- **Alternatives considered**: Reuse full Swagger — leaks unrelated endpoints and is
  noisy for tool import. Static-file-only — not auto-importable by URL.

## R11. Key management UX — scope at creation

- **Decision**: Extend the existing API-key management UI in Settings to choose a
  read-only or read+write scope at key creation, and show each key's scope in the
  list. Honors the repo principle "API first but NOT API only."
- **Rationale**: Users must be able to mint a write-capable external key and audit
  scopes without API calls.

## R12. Migration default — existing keys read-only

- **Decision**: On migration, API keys lacking an explicit capability are treated as
  read-only (least privilege). Users re-issue if they need external write.
- **Rationale**: Prevents silently granting write to pre-existing keys.

## Dependencies & references

- Hard dependency: issue #217 shared tool layer (`CollectionToolsService`,
  `CollectionUpdateProposal`, `/api/internal/tools/*`).
- Decision #11 (`.squad/decisions.md`): shared transport-agnostic tool layer designed
  to back both in-app and external adapters.
- Constitution: Principle I, III, VII, XI, XII, XIII; §17 Quality Gate; §21 DoD.
