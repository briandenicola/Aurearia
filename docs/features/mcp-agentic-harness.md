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

For copy-ready GitHub Copilot CLI, VS Code, Claude Code, generic client,
verification, rotation, and troubleshooting instructions, see
[MCP Client Setup](../mcp-client-setup.md).

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
| `get_coin_images` | Read owned coin photo thumbnails or medium images (for the coin viewer) |

Collection search excludes wishlist and sold records. Use `list_wishlist` for
wishlist data. Inputs, page sizes, query lengths, and returned auction text are
bounded at the service boundary.

## Coin viewer (MCP Apps)

In hosts that support [MCP Apps](https://modelcontextprotocol.io/docs/extensions/apps),
`search_collection` and `get_coin` results open an interactive coin viewer
instead of plain JSON: a photo grid of matching coins, and a detail view with
obverse, reverse, and other photos when a coin is selected. The viewer is the
static resource `ui://aurearia/coin-viewer.html`; it makes no network requests
of its own and loads photos by calling `get_coin_images` through the host.

`get_coin_images` is marked app-only (`_meta.ui.visibility: ["app"]`), so
MCP Apps hosts hide it from the model. It returns JPEG variants as image
content blocks:

- `size: "thumb"` (default): 1-20 coin ids, one image per coin, primary first.
- `size: "medium"`: exactly one coin id, up to six images.
- At most 2 MiB of image data per call; `truncated: true` means some photos
  were left out. Attribution cards and other owners' coins are never returned.

The viewer is a Vue 3 app in `src/web/src/mcp-apps/coin-viewer`. After
changing it, run `npm run build:mcp-app` in `src/web` and commit the
regenerated `src/api/handlers/mcpapps/coin-viewer.html`, which the Go API
embeds. `npm run test` fails if that file is out of date.

Hosts without MCP Apps support keep receiving the same JSON text as before and
may show `get_coin_images` to their model like any other tool. Hermes Agent's
CLI and messaging surfaces are text-only today, so they show no viewer. To try
the viewer, connect a host such as Claude Desktop, VS Code GitHub Copilot,
Goose, or [MCPJam](https://www.mcpjam.com/).

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

Copilot is not limited to the collector's data. It answers general numismatic
research questions (attribution, catalog references, history, authenticity,
grading, market value) and, with an Anthropic model, researches them with web
search and web fetch. Pass the collector's question as the goal with minimal
rewording and include any URLs verbatim; web fetch can open only URLs present
in the conversation. See [Coin Copilot web research](coin-copilot.md#web-research).

## Security boundaries

- The external-tool admin setting defaults to off and gates both adapters.
- Every request requires an active owner-scoped API key with `read`.
- Per-key external rate limiting applies before the MCP handler.
- Copilot tools are registered only for an exact `copilot` capability token.
- Cross-owner identifiers produce not-found behavior without leaking data.
- MCP requests are limited to 128 KiB.
- The only MCP resource is the static, self-contained coin viewer HTML; it
  contains no user data and declares no external network domains.
- The MCP surface exposes no collection, wishlist, auction, settings, shell,
  filesystem, arbitrary HTTP, or database mutation tools. Coin Copilot's web
  search and fetch run inside the Copilot harness, executed by Anthropic, and
  are not exposed as MCP tools.
- Coin Copilot tools persist only their existing durable run state; they do not
  grant general application write access.

Do not proxy `/api/v1/tools/openapi.json` into MCP. The OpenAPI adapter includes
the separately authorized two-phase collection write workflow and therefore
has a broader contract than the native MCP surface.

## Related documentation

- [External Tool Server](../external-tool-server.md) - administration, API
  keys, OpenAPI clients, MCP clients, and error handling
- [MCP Client Setup](../mcp-client-setup.md) - copy-ready client configuration
  and troubleshooting
- [Authentication](../authentication.md) - API-key creation and capabilities
- [API Reference](../api-reference.md) - HTTP contracts
- [Coin Copilot](coin-copilot.md) - durable run behavior and limits
- [Portable Aurearia MCP skill](../../.github/skills/using-aurearia-mcp/SKILL.md)
- [ADR 0021](../adr/0021-stateless-mcp-api-adapter.md) - architecture decision
