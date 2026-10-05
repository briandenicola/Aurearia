---
name: using-aurearia-mcp
description: "Use Aurearia's read-only collection, wishlist, auction, statistics, and durable Coin Copilot MCP tools safely."
---

# Using Aurearia MCP

Use this skill when a collector asks for facts, comparisons, planning, or
analysis grounded in their Aurearia data.

## Connection

Connect by Streamable HTTP to:

```text
{AUREARIA_URL}/api/mcp
```

Send the API key only in this header:

```text
X-API-Key: {AUREARIA_API_KEY}
```

Never place the key in a prompt, URL, repository file, tool argument, log, or
answer. A `read` key exposes data tools. Coin Copilot tools require a
`read,copilot` key.

For GitHub Copilot CLI, create the key in Aurearia Settings, put it in the
current shell's secret environment, then register the server:

```powershell
$env:AUREARIA_URL = "https://your-aurearia-host"
$env:AUREARIA_API_KEY = "ak_replace_with_generated_key"
copilot mcp add --transport http --header "X-API-Key: $env:AUREARIA_API_KEY" aurearia "$env:AUREARIA_URL/api/mcp"
```

Use `/mcp show aurearia` to verify discovery. Do not commit the resulting
credential or copy a populated user configuration into a repository.

For GitHub Copilot CLI, VS Code, Claude Code, generic Streamable HTTP clients,
key rotation, and status-code troubleshooting, see
[`docs/mcp-client-setup.md`](../../../docs/mcp-client-setup.md).

## Read Tools

- `search_collection`: search owned coins. Use a focused query and bounded
  limit.
- `get_coin`: fetch one known owned coin by id.
- `list_wishlist`: query wishlist entries; use this rather than guessing that
  a collection search result is a wishlist item.
- `collection_stats`: read aggregate collection, wishlist, sold, distribution,
  and value statistics.
- `top_coins_by_value`: read a bounded highest-value active-collection list.
- `list_auction_lots`: query owned auction lots by status, source, search, and
  page.
- `get_auction_lot`: fetch one known owned auction lot by id.
- `auction_counts`: read auction status counts, optionally by source.

Treat empty results as evidence that Aurearia has no matching owned data. Do
not infer holdings, wishlist entries, bids, or values that tools did not
return.

## Coin Copilot Lifecycle

Coin Copilot is asynchronous and durable:

1. Call `start_copilot_run` with a concise goal and a stable idempotency key.
2. Poll `get_copilot_run` until the status is `completed`, `failed`,
   `cancelled`, or `paused`.
3. If paused, ask the user the requested clarification. Call
   `resume_copilot_run` with the returned checkpoint version and a new stable
   idempotency key.
4. Call `cancel_copilot_run` when the user asks to stop or the result is no
   longer needed.

Reuse an idempotency key only for an exact retry of the same input. Never
fabricate a checkpoint version or retry a conflicting request with the same
key.

## Boundaries

The MCP surface is read-only for collection, wishlist, auction, and statistics
data. Coin Copilot may persist its own run/checkpoint/event state, but these
tools do not add, edit, delete, import, sync, convert, bid on, or purchase
coins or lots. If asked for a mutation, explain that it must be completed in
the Aurearia application.
