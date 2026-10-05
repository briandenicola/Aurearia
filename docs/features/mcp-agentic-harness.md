# MCP Agentic Harness

Feature 366 exposes Aurearia to external agentic harnesses through a native,
stateless Model Context Protocol endpoint:

```text
POST https://your-aurearia-host/api/mcp
X-API-Key: ak_your_generated_key
```

The endpoint uses Streamable HTTP with JSON responses. It is embedded in the
Go API, reuses Aurearia's owner-scoped services, and does not create durable
MCP sessions. Durable Coin Copilot state remains in Go-owned threads, runs,
checkpoints, and events.

## Enable and connect

1. An administrator enables **External Tool Server Enabled** in Admin settings.
2. The collector creates a dedicated key in **Settings -> Data -> API Keys**.
3. The client stores the key as a secret and sends it only in `X-API-Key`.
4. The client connects to `/api/mcp` using Streamable HTTP.

Use a `read` key for data access. Select `read,copilot` only when the harness
must also invoke Coin Copilot. The `write` capability belongs to the separate
OpenAPI adapter and never implies `copilot`.

GitHub Copilot CLI example:

```powershell
$env:AUREARIA_URL = "https://your-aurearia-host"
$env:AUREARIA_API_KEY = "ak_replace_with_generated_key"
copilot mcp add --transport http --header "X-API-Key: $env:AUREARIA_API_KEY" aurearia "$env:AUREARIA_URL/api/mcp"
```

Run `/mcp show aurearia` to verify discovery.

## Read-only data tools

| Tool | Purpose |
|---|---|
| `search_collection` | Search the active owned collection |
| `get_coin` | Read one owned coin |
| `list_wishlist` | Search owned wishlist entries |
| `collection_stats` | Read collection, wishlist, sold, distribution, and value statistics |
| `top_coins_by_value` | Read a bounded highest-value active-collection list |
| `list_auction_lots` | List owned auction lots with bounded filters and pagination |
| `get_auction_lot` | Read one owned auction lot |
| `auction_counts` | Read auction status counts, optionally by source |

Collection search excludes wishlist and sold records. Use `list_wishlist` for
wishlist data. Inputs, page sizes, query lengths, and returned auction text are
bounded at the service boundary.

## Coin Copilot lifecycle tools

Keys with the exact `copilot` capability also discover:

| Tool | Purpose |
|---|---|
| `start_copilot_run` | Start or idempotently replay a durable run |
| `get_copilot_run` | Inspect an owner-scoped run |
| `resume_copilot_run` | Resume a paused run from its expected checkpoint |
| `cancel_copilot_run` | Request cancellation |

Copilot is asynchronous. Start a run with a stable idempotency key, poll until
it reaches a terminal or paused state, and resume only with the checkpoint
version returned by Aurearia. Coin Copilot must also be enabled and pass its
existing provider/model capability preflight.

## Security boundaries

- The external-tool admin setting defaults to off and gates both adapters.
- Every request requires an active owner-scoped API key with `read`.
- Per-key external rate limiting applies before the MCP handler.
- Copilot tools are registered only for an exact `copilot` capability token.
- Cross-owner identifiers produce not-found behavior without leaking data.
- MCP requests are limited to 128 KiB.
- The MCP surface exposes no collection, wishlist, auction, settings, shell,
  filesystem, arbitrary HTTP, or database mutation tools.
- Coin Copilot tools persist only their existing durable run state; they do not
  grant general application write access.

Do not proxy `/api/v1/tools/openapi.json` into MCP. The OpenAPI adapter includes
the separately authorized two-phase collection write workflow and therefore
has a broader contract than the native MCP surface.

## Related documentation

- [External Tool Server](../external-tool-server.md) - administration, API
  keys, OpenAPI clients, MCP clients, and error handling
- [Authentication](../authentication.md) - API-key creation and capabilities
- [API Reference](../api-reference.md) - HTTP contracts
- [Coin Copilot](coin-copilot.md) - durable run behavior and limits
- [Portable Aurearia MCP skill](../../.github/skills/using-aurearia-mcp/SKILL.md)
- [ADR 0021](../adr/0021-stateless-mcp-api-adapter.md) - architecture decision
