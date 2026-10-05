# MCP Client Setup

This guide connects external MCP clients to Aurearia's native Feature 366
endpoint. It covers server preparation, API-key selection, GitHub Copilot CLI,
Visual Studio Code, Claude Code, discovery checks, credential rotation, and
troubleshooting.

## Requirements

The client must support:

- remote MCP over Streamable HTTP;
- a custom `X-API-Key` request header; and
- the endpoint `https://your-aurearia-host/api/mcp`.

Use HTTPS whenever the client connects across a network. A client that cannot
send custom headers cannot connect directly. Do not place the API key in the
URL and do not proxy the write-capable OpenAPI adapter into MCP.

## Prepare Aurearia

1. In **Admin -> System Settings**, enable **External Tool Server Enabled**.
   This default-off setting gates both OpenAPI and MCP.
2. In **Settings -> Data -> API Keys**, create a dedicated key for the client.
3. Copy the key when it is displayed. Aurearia shows the full key only once.
4. Store the key in the client's secret store or an environment variable.

Choose the least-privilege capability:

| Client need | Capability | Expected tools |
|---|---|---:|
| Collection, wishlist, auction, and statistics reads | `read` | 8 |
| Reads plus Coin Copilot lifecycle | `read,copilot` | 12 |

Do not choose `write` for an MCP-only client. `write` applies to the separate
OpenAPI adapter and never implies `copilot`.

Examples below use:

```text
AUREARIA_URL=https://your-aurearia-host
AUREARIA_API_KEY=ak_replace_with_generated_key
```

Do not include a trailing `/api` in `AUREARIA_URL`.

## Reverse proxy checklist

For a remotely hosted Aurearia instance, the reverse proxy must:

- route `POST /api/mcp` to the Go API;
- preserve the `X-API-Key`, `Content-Type`, and `Accept` request headers;
- allow request bodies up to at least 128 KiB;
- avoid caching MCP requests or responses; and
- present a certificate trusted by the client.

No separate MCP container, port, SSE route, or Python-agent exposure is
required.

## GitHub Copilot CLI

The current Copilot CLI supports remote HTTP servers and repeated `--header`
options:

### PowerShell

```powershell
$env:AUREARIA_URL = "https://your-aurearia-host"
$env:AUREARIA_API_KEY = "ak_replace_with_generated_key"

copilot mcp add --transport http `
  --header "X-API-Key: $env:AUREARIA_API_KEY" `
  aurearia "$env:AUREARIA_URL/api/mcp"
```

### Bash or zsh

```bash
export AUREARIA_URL="https://your-aurearia-host"
export AUREARIA_API_KEY="ak_replace_with_generated_key"

copilot mcp add --transport http \
  --header "X-API-Key: $AUREARIA_API_KEY" \
  aurearia "$AUREARIA_URL/api/mcp"
```

Start Copilot CLI and run:

```text
/mcp show aurearia
```

Confirm that the expected 8 or 12 tools appear. Copilot stores configured
servers in its user MCP configuration. Treat that configuration as sensitive
because the command expands the API key before saving it.

Remove or replace the server with:

```powershell
copilot mcp remove aurearia
```

## Visual Studio Code

VS Code supports remote HTTP MCP servers in `.vscode/mcp.json`. Use a password
input so the key is prompted for and stored by VS Code instead of committed:

```json
{
  "inputs": [
    {
      "type": "promptString",
      "id": "aurearia-api-key",
      "description": "Aurearia API key",
      "password": true
    }
  ],
  "servers": {
    "aurearia": {
      "type": "http",
      "url": "https://your-aurearia-host/api/mcp",
      "headers": {
        "X-API-Key": "${input:aurearia-api-key}"
      }
    }
  }
}
```

Open the Command Palette and run **MCP: List Servers** or open the MCP server
view. Start `aurearia`, enter the key when prompted, and inspect its discovered
tools. Use a user-profile MCP configuration instead of `.vscode/mcp.json` when
the endpoint itself should not be shared with the repository.

Never replace the input variable with a literal key in a tracked file.

## Claude Code

Claude Code supports remote HTTP servers with custom headers.

### User-local CLI configuration

```bash
claude mcp add --transport http aurearia \
  https://your-aurearia-host/api/mcp \
  --header "X-API-Key: $AUREARIA_API_KEY"
```

This command expands and stores the key in Claude Code's local configuration.
Do not share that file.

Verify the connection:

```bash
claude mcp get aurearia
claude mcp list
```

Inside Claude Code, `/mcp` shows connection status and discovered tools.

### Project configuration with environment placeholders

To share a secret-free project definition, create `.mcp.json`:

```json
{
  "mcpServers": {
    "aurearia": {
      "type": "http",
      "url": "${AUREARIA_URL}/api/mcp",
      "headers": {
        "X-API-Key": "${AUREARIA_API_KEY}"
      }
    }
  }
}
```

Each user must set both environment variables before starting Claude Code.
Review and approve the project MCP server when prompted. Commit only the
placeholder configuration, never populated credentials.

## Other MCP clients

Use this equivalent server definition when the client supports remote HTTP and
custom headers:

```json
{
  "name": "aurearia",
  "type": "http",
  "url": "https://your-aurearia-host/api/mcp",
  "headers": {
    "X-API-Key": "<secret>"
  }
}
```

Configuration wrapper names differ by client (`servers`, `mcpServers`, or a
settings form). The required values do not:

- transport: Streamable HTTP / HTTP;
- URL: `https://your-aurearia-host/api/mcp`;
- header name: `X-API-Key`;
- header value: the generated `ak_...` key.

Aurearia does not expose MCP over stdio, legacy SSE, WebSocket, or OAuth.

## Optional Agent Skill

The portable skill at
`.github/skills/using-aurearia-mcp/SKILL.md` teaches a compatible agent the
tool meanings, Coin Copilot polling flow, idempotency rules, and mutation
boundaries. Copy the complete `using-aurearia-mcp` directory into the target
harness's supported skill location when the harness does not load this
repository's skills automatically.

The skill contains no credentials. Keep credentials in the client
configuration or secret store, not in `SKILL.md`.

## Verify the connection

After connecting:

1. Confirm the server reports connected or healthy.
2. Confirm a `read` key discovers exactly the eight read tools.
3. For `read,copilot`, confirm the four lifecycle tools also appear.
4. Call `collection_stats` or `search_collection` with a narrow query.
5. If Copilot is enabled, start a small run with a stable idempotency key and
   inspect it with `get_copilot_run`.

Expected read tools:

```text
auction_counts
collection_stats
get_auction_lot
get_coin
list_auction_lots
list_wishlist
search_collection
top_coins_by_value
```

Additional `copilot` tools:

```text
cancel_copilot_run
get_copilot_run
resume_copilot_run
start_copilot_run
```

## Troubleshooting

| Symptom | Likely cause | Resolution |
|---|---|---|
| `401 Unauthorized` | Missing, invalid, or revoked key | Confirm `X-API-Key`, remove whitespace, or generate a replacement |
| `403 Forbidden` | Key lacks `read` | Create a canonical `read` or `read,copilot` key |
| `413 Request Entity Too Large` | Client sent more than 128 KiB | Reduce the MCP request or context payload |
| `429 Too Many Requests` | Per-key external rate limit reached | Wait for the window to reset; do not retry in a tight loop |
| `503 Service Unavailable` | External Tool Server Enabled is off | Ask an administrator to enable the external-tool server |
| Eight tools but no Copilot tools | Key lacks exact `copilot`, or client cached discovery | Use `read,copilot`, then reconnect or refresh discovery |
| Copilot tool reports unavailable | Coin Copilot is disabled or model preflight failed | Enable/configure Coin Copilot and verify provider tool support |
| Connection or TLS failure | Host is unreachable or certificate is untrusted | Verify DNS, port, reverse proxy, HTTPS certificate, and `/api/mcp` routing |
| Empty owned results | No matching owner data | Confirm the key belongs to the intended Aurearia account |

After rotating a key, update the client secret and reconnect. Revoke the old key
in Aurearia immediately after the replacement succeeds.

## Client documentation

- [GitHub Copilot CLI: add an MCP server](https://docs.github.com/en/copilot/how-tos/use-copilot-agents/use-copilot-cli#add-an-mcp-server)
- [VS Code MCP configuration reference](https://code.visualstudio.com/docs/agents/reference/mcp-configuration)
- [Claude Code MCP reference](https://code.claude.com/docs/en/mcp)
