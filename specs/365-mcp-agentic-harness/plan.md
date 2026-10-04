# Implementation Plan: MCP Agentic Harness

## Architecture

Use the official pinned MCP Go SDK as a protocol adapter inside `handlers/`.
Keep owner-scoped operations in a new `services.MCPService`, which delegates to
the existing collection service, repositories, and Coin Copilot service.
Register one stateless Streamable HTTP route under existing external-tool
middleware. Stateless requests avoid cross-key MCP session state and are
sufficient because Copilot runs are already durable and polled through tools.

## Tool Contract

Read tools:

- `search_collection`
- `get_coin`
- `list_wishlist`
- `collection_stats`
- `top_coins_by_value`
- `list_auction_lots`
- `get_auction_lot`
- `auction_counts`

Copilot tools, registered only for an exact `copilot` capability:

- `start_copilot_run`
- `get_copilot_run`
- `resume_copilot_run`
- `cancel_copilot_run`

## Security

- Reuse `ExternalToolServerEnabled`, API-key authentication, `read`
  capability middleware, and per-key rate limiting.
- Add exact-token `copilot` capability validation; do not let `write` imply it.
- Derive owner identity from middleware and close over it in each stateless MCP
  server instance.
- Keep protocol requests at or below 128 KiB.
- Return bounded DTOs rather than database models containing owner fields.

## Verification

- Unit tests for capabilities, owner scoping, filters, body limits, tool
  discovery, and Copilot gating.
- Protocol tests through an in-process MCP client.
- Tamper test proving removal of the Copilot capability guard fails.
- Go formatting, targeted tests, `task check:go`, OpenAPI drift checks, and
  delivery checks.
- Independent read-only review because this changes an authenticated external
  boundary.

