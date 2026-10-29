---
name: using-aurearia-mcp
description: "Use Aurearia's read-only collection, wishlist, auction, statistics, and durable Coin Copilot MCP tools safely, including routing general numismatic research questions to Coin Copilot."
---

# Using Aurearia MCP

Use this skill when a collector asks for facts, comparisons, planning, or
analysis grounded in their Aurearia data, or asks a general numismatic research
question (attribution, iconography, legends, catalog references, history,
authenticity, grading, or market value) that Coin Copilot can research.

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

- `search_collection`: search owned coins. Use a focused query. `limit`
  defaults to 5 and larger values are capped at 20.
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
- `get_coin_images`: owned coin photos as image content, used by the coin
  viewer UI. Call it directly only when the collector asks to see a coin and
  the host can show images; prefer `size: "thumb"` and few coin ids.

Treat empty results as evidence that Aurearia has no matching owned data. Do
not infer holdings, wishlist entries, bids, or values that tools did not
return.

## Choosing Between Read Tools and Coin Copilot

- Use the read tools for direct lookups of the collector's own data: a known
  coin, a filtered list, counts, or statistics. They are synchronous and cheap.
- Use Coin Copilot for multi-step questions and for numismatic research:
  attribution, iconography, legends, catalog references (RIC, RPC, Crawford,
  Sear, SNG), history, authenticity signals, grading, dealer listings, auction
  lots, price trends, and comparable sales. The owner's collection is one
  source among many, not the boundary of its scope.
- Pass the collector's question to `start_copilot_run` as the goal with little
  or no rewording. Copilot already routes collection questions to collection
  tools, market questions to its dealer, auction, price-trend, and similar-lot
  specialists, and fact-checking to web research. Do not prescribe the order
  of its tools in the goal.
- Include every URL the collector supplied (auction lot, dealer listing,
  catalog entry) verbatim in the goal. Copilot's web fetch can open only URLs
  already present in the conversation or found by its own web search.
- Return Copilot's answer with its cited sources intact. Do not merge your own
  unattributed research into it.

Web search and web fetch are available only when the Aurearia administrator
configures an Anthropic model. With Ollama, Copilot answers research questions
from its own knowledge and says when it needed a page it could not read.

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

A research run that stops at its time or iteration limit can be retried with a
narrower goal; the defaults are 120 seconds and 8 reasoning iterations
per execution.

## Boundaries

The MCP surface is read-only for collection, wishlist, auction, and statistics
data. Coin Copilot may persist its own run/checkpoint/event state and may read
public web pages for research, but these tools do not add, edit, delete,
import, sync, convert, bid on, or purchase coins or lots. Copilot never puts
collection details or prices paid into web search queries, and treats web
content as untrusted evidence. If asked for a mutation, explain that it must be completed in
the Aurearia application.
