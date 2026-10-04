# Feature Specification: MCP Agentic Harness

**Created**: 2026-10-04
**Status**: Approved for implementation by the repository owner
**Input**: Owner request to expose Aurearia to external agentic harnesses

## Scope

Aurearia exposes a remote Model Context Protocol endpoint from the existing Go
API. The endpoint provides owner-scoped, read-only access to collection,
wishlist, auction, and statistics data, plus bounded lifecycle operations for
the existing durable Coin Copilot harness.

The approved transport is stateless Streamable HTTP at `POST /api/mcp`.
Authentication reuses per-user API keys. All MCP access requires `read`; Coin
Copilot lifecycle tools additionally require the explicit `copilot`
capability. The existing `ExternalToolServerEnabled` default-off setting and
external per-key rate limit protect the endpoint.

## User Stories

### Query collection data

An authenticated harness can search the active collection, fetch one owned
coin, list wishlist entries, and read collection statistics without mutating
any record.

### Query auction data

An authenticated harness can list/filter owned auction lots, fetch one owned
lot, and read per-status counts without importing, syncing, converting, or
updating lots.

### Invoke Coin Copilot

A key carrying `read,copilot` can start, inspect, resume, and cancel an
owner-scoped durable Coin Copilot run. Calls reuse the existing bounded,
idempotent service and do not add a second orchestration system.

### Install in agentic harnesses

The repository provides a portable Agent Skill and configuration examples that
reference an MCP URL and API key through environment placeholders. No
credential is stored in the repository.

## Functional Requirements

- **FR-001**: The MCP endpoint MUST be implemented in the existing Go API; no
  fourth service or direct database client is added.
- **FR-002**: The transport MUST implement stateless Streamable HTTP at
  `POST /api/mcp`.
- **FR-003**: The endpoint MUST be disabled when
  `ExternalToolServerEnabled=false`.
- **FR-004**: Every request MUST authenticate through the existing API-key/JWT
  middleware and external per-key rate limit.
- **FR-005**: Every exposed data tool MUST require `read` and derive its owner
  server-side. Clients MUST NOT provide a user id.
- **FR-006**: Collection tools MUST be limited to `search_collection`,
  `get_coin`, `list_wishlist`, `collection_stats`, and
  `top_coins_by_value`.
- **FR-007**: Auction tools MUST be limited to `list_auction_lots`,
  `get_auction_lot`, and `auction_counts`.
- **FR-008**: The MCP surface MUST NOT expose collection, wishlist, auction,
  settings, filesystem, shell, arbitrary HTTP, or database mutation tools.
- **FR-009**: Coin Copilot tools MUST be limited to `start_copilot_run`,
  `get_copilot_run`, `resume_copilot_run`, and `cancel_copilot_run`.
- **FR-010**: Coin Copilot tools MUST be absent unless the authenticated key
  has an exact `copilot` capability token.
- **FR-011**: Copilot start/resume MUST preserve the existing idempotency,
  limits, owner scoping, feature flags, and model capability checks.
- **FR-012**: Unknown or foreign coin, auction, thread, and run identifiers
  MUST fail without leaking another owner's data.
- **FR-013**: MCP request bodies MUST be capped and protocol errors MUST not
  expose internal errors.
- **FR-014**: API-key creation MUST support `read,copilot` without making
  Copilot access implicit for existing `read` or `read,write` keys.
- **FR-015**: The portable skill MUST describe read-only boundaries, Copilot
  idempotency/polling, and secret-safe configuration.

## Non-Goals

- OAuth or anonymous MCP access.
- Stateful MCP sessions, server-initiated sampling, roots, prompts, or MCP
  resources.
- New Coin Copilot capabilities or direct invocation of Python agent routes.
- Collection, wishlist, auction, or settings writes.
- Deployment, publication, or automatic registration in a user's harness.

## Acceptance Criteria

1. A valid `read` key can initialize MCP, discover only read tools, and call
   each owner-scoped read tool.
2. A `read` key cannot discover or invoke Copilot lifecycle tools.
3. A `read,copilot` key can start, inspect, resume, and cancel through the
   existing Coin Copilot service.
4. A foreign identifier returns a safe not-found/tool error and no foreign
   payload.
5. A disabled external-tool setting, missing/invalid key, malformed capability,
   oversized body, or write-shaped tool name fails closed.
6. The portable skill and setup documentation contain placeholders only.

