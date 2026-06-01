# External Tool Server

The External Tool Server exposes your coin collection to external AI clients (OpenWebUI, LibreChat, n8n, and MCP-compatible clients) over a stable, versioned HTTP API. External clients can query your collection, analyze statistics, and optionally propose updates through a secure two-phase commit flow.

This document covers the security model, enabling and configuring the server, creating API keys with appropriate scopes, the available tools, and step-by-step setup instructions for popular clients.

---

## Audience Guide

This guide is organized by role:

- **For Administrators** — Enable/disable the server, understand the security posture, monitor activity, and revoke keys
- **For Users** — Create API keys, configure client integrations, and understand the available operations
- **For Developers** — Reference the complete API surface, error codes, and integration patterns

---

## Security Model

The external tool server is designed with least-privilege defaults and multiple layers of protection:

### Default-Off Admin Toggle

The entire `/api/v1/tools/*` surface is disabled by default. An admin must explicitly enable it in **Admin → System Settings → External Tool Server Enabled**. When disabled, all external tool requests return `503 Service Unavailable` with no side effects.

### Scoped API Keys

External clients authenticate with per-user API keys created in **Settings → Data → API Keys**. Each key has one of two capability scopes:

- **read** (default) — Allows queries: `search_my_collection`, `get_coin`, `collection_summary`, `top_coins_by_value`
- **read,write** — Allows queries plus write proposals and commits: `propose_update`, `commit_update`

Write capability must be explicitly chosen at key creation time. The least-privilege default is read-only.

### Two-Phase Write Protection

External writes require a two-phase confirm flow identical to the in-app collection chat:

1. **Propose** — Call `propose_update` with a coin ID and allowed field changes. The server validates the fields, checks ownership, and returns a proposal ID, token, and preview without writing anything.
2. **Commit** — Call `commit_update` with the proposal ID, token, and explicit `confirm=true`. The server verifies the token and proposal state, persists the change, and records a journal entry.

This prevents accidental or conversational auto-writes. Proposals expire after a configurable TTL.

### Journaling and Audit Trail

Every external commit writes a journal entry with:

- `source`: `external_tool_server`
- API key ID and name
- Capability scope used
- Changed fields and new values

Journal entries appear in each coin's activity log and are queryable by admins for incident response.

### Tenant Isolation

Every operation is scoped to the API key owner's data. User identity is derived server-side from the key; clients never supply a user ID. Attempting to access another user's coins returns `404 Not Found` with no data leak.

### Per-Key Rate Limiting

External tool endpoints enforce a stricter per-key rate limit (50 requests per minute by default) independent of in-app limits. This bounds abuse from a compromised or misconfigured external client without affecting in-app usage.

### Field Allowlist

External writes are restricted to the same field allowlist as in-app updates: `grade`, `currentValue`, `notes`, `tags`, `referenceText`, `referenceUrl`, and `references`. Identity fields (name, category, era, ruler, etc.) are rejected with `400 Bad Request`.

---

## For Administrators

### Enabling the External Tool Server

The entire `/api/v1/tools/*` surface is disabled by default. To enable it:

1. Navigate to **Admin → System Settings**
2. Toggle **External Tool Server Enabled** to ON
3. The OpenAPI spec and tool endpoints become available immediately at `/api/v1/tools/`

To disable the server later, toggle it OFF. All external tool requests will then fail with `503 Service Unavailable`.

### Security Posture for Admins

The external tool server is designed with least-privilege defaults and multiple layers of protection:

**Default-Off Design** — External tools are disabled by default. Enable only when you intend to allow external integrations. You retain full control to disable at any time.

**Scoped API Keys** — Every key has one of two capability scopes:
- `read` (default) — Allows queries only
- `read,write` — Allows queries plus write proposals and commits

Write capability must be explicitly chosen by users. Least-privilege is the default.

**Two-Phase Write Protection** — External writes require explicit confirmation (propose, then commit). This prevents accidental or conversational auto-writes.

**Journaling and Audit Trail** — Every external commit writes an entry with API key ID, capability scope, and changed fields. Admins can review entries for incident response.

**Per-Key Rate Limiting** — External tool endpoints enforce 50 requests per minute per API key, independent of in-app limits. This bounds abuse from a compromised client.

**Field Allowlist** — External writes are restricted to: `grade`, `currentValue`, `notes`, `tags`, `referenceText`, `referenceUrl`, and `references`. Identity fields are always rejected.

**Tenant Isolation** — User identity is derived server-side from the API key. Clients cannot access another user's coins; any such attempt returns `404 Not Found` with no data leak.

### Monitoring and Revocation

Users manage their own keys in **Settings → Data → API Keys**, where each key displays name, prefix, capability, and last-used timestamp. Admins can view journal entries on individual coins to audit external changes. To revoke suspicious activity:

1. Have the user revoke the affected key at **Settings → Data → API Keys**
2. Review the coin activity journal for unexpected changes
3. The revoked key immediately stops working

For full security details, see [threat-model.md](threat-model.md).

---

## For Users

### Creating API Keys

Each external client needs its own API key. Navigate to **Settings → Data → API Keys** and follow one of these flows:

#### Read-Only Key (Default)

1. Enter a descriptive name (e.g., "OpenWebUI Read-Only")
2. Leave **Capability** at `read` (the default)
3. Click **Generate**
4. Copy the displayed key (starts with `ak_`, shown only once)

#### Read+Write Key

1. Enter a descriptive name (e.g., "LibreChat Write Access")
2. Change **Capability** to `read,write`
3. Click **Generate**
4. Copy the displayed key

#### Managing Your Keys

The API Keys list shows each key's name, prefix, capability scope, and last-used timestamp. Revoke a key anytime with the **Revoke** button—it stops working immediately.

### Getting the OpenAPI URL

After admin enables the server, you can fetch the full OpenAPI spec at:

```
http://your-ancient-coins-host:8080/api/v1/tools/openapi.json
```

Use this URL in your external client's tool import wizard (see client setup guides below).

### Available Operations

The external tool server exposes six operations, all scoped to your collection:

**Read Operations** (require `read` or `read,write` key):
- `search_my_collection` — Search by query string
- `get_coin` — Fetch a single coin by ID
- `collection_summary` — Aggregate statistics (total coins, value, etc.)
- `top_coins_by_value` — Top N coins by current value

**Write Operations** (require `read,write` key):
- `propose_update` — Propose field changes (returns preview and token, no writes yet)
- `commit_update` — Confirm the proposal and persist the change

All writes are journaled with the API key name and changed fields.

---

## For Developers

### Available Tools

### Read Tools (Require `read` Capability)

#### `search_my_collection`

Search your collection by a query string. Returns matching coins sorted by relevance.

**Request:**

```json
{
  "query": "denarius augustus",
  "limit": 10
}
```

**Response:**

```json
{
  "coins": [
    {
      "id": 42,
      "name": "Denarius of Augustus",
      "category": "Roman",
      "era": "ancient",
      "ruler": "Augustus",
      "material": "silver",
      "currentValue": 450.00
    }
  ]
}
```

**Parameters:**

- `query` (string, required) — Search term(s) to match across all coin fields
- `limit` (integer, optional) — Maximum results to return (1–50, default 10)

#### `get_coin`

Retrieve a single coin by its ID.

**Request:**

```json
{
  "coin_id": 42
}
```

**Response:**

```json
{
  "coin": {
    "id": 42,
    "name": "Denarius of Augustus",
    "category": "Roman",
    "era": "ancient",
    "ruler": "Augustus",
    "material": "silver",
    "currentValue": 450.00
  }
}
```

**Parameters:**

- `coin_id` (integer, required) — ID of the coin to retrieve

**Errors:**

- `404 Not Found` — Coin does not exist or is owned by another user

#### `collection_summary`

Get aggregate statistics for your collection.

**Request:**

```json
{}
```

**Response:**

```json
{
  "summary": {
    "totalCoins": 127,
    "totalWishlist": 8,
    "totalSold": 12,
    "totalCurrentUsd": 34500.00,
    "totalPurchaseUsd": 28200.00
  }
}
```

#### `top_coins_by_value`

Retrieve the top coins in your collection by current value.

**Request:**

```json
{
  "limit": 5
}
```

**Response:**

```json
{
  "coins": [
    {
      "id": 15,
      "name": "Aureus of Nero",
      "category": "Roman",
      "era": "ancient",
      "ruler": "Nero",
      "material": "gold",
      "currentValue": 5200.00
    }
  ]
}
```

**Parameters:**

- `limit` (integer, optional) — Maximum results (1–10, default 3)

### Write Tools (Require `write` Capability)

#### `propose_update` (Phase 1)

Propose changes to allowlisted fields on a coin. Returns a proposal preview and token without persisting any changes.

**Request:**

```json
{
  "coin_id": 42,
  "changes": {
    "grade": "EF",
    "notes": "Re-graded by NGC",
    "currentValue": 480.00
  }
}
```

**Response:**

```json
{
  "proposal": {
    "proposalId": "prop_1234567890abcdef",
    "proposalToken": "tok_abcdef1234567890",
    "coinId": 42,
    "coinName": "Denarius of Augustus",
    "changedFields": ["grade", "notes", "currentValue"],
    "changes": {
      "grade": "EF",
      "notes": "Re-graded by NGC",
      "currentValue": 480.00
    },
    "expiresAt": "2026-06-01T08:46:00Z"
  }
}
```

**Parameters:**

- `coin_id` (integer, required) — ID of the coin to update
- `changes` (object, required) — Map of field names to new values. Only allowlisted fields are accepted: `grade`, `currentValue`, `notes`, `tags`, `referenceText`, `referenceUrl`, `references`

**Errors:**

- `400 Bad Request` — Invalid field or disallowed field change (e.g., identity fields like `category`)
- `404 Not Found` — Coin does not exist or is owned by another user

#### `commit_update` (Phase 2)

Commit a proposal with explicit confirmation. Persists the changes and journals the write.

**Request:**

```json
{
  "proposal_id": "prop_1234567890abcdef",
  "token": "tok_abcdef1234567890",
  "confirm": true
}
```

**Response:**

```json
{
  "result": {
    "proposalId": "prop_1234567890abcdef",
    "status": "committed",
    "coinId": 42,
    "changedFields": ["grade", "notes", "currentValue"],
    "journalSource": "external_tool_server",
    "message": "Update committed successfully"
  }
}
```

**Parameters:**

- `proposal_id` (string, required) — Proposal ID from `propose_update`
- `token` (string, required) — Proposal token from `propose_update`
- `confirm` (boolean, required) — Must be `true` (explicit confirmation gate)

**Errors:**

- `400 Bad Request` — `confirm` is not `true`
- `401 Unauthorized` — Invalid or mismatched token
- `404 Not Found` — Proposal does not exist or coin is owned by another user
- `409 Conflict` — Proposal is not in `pending` state or has expired

### API Authentication

All `/api/v1/tools/*` endpoints require API key authentication via the `X-API-Key` header:

```bash
curl -H "X-API-Key: ak_your_key_here" \
  http://localhost:8080/api/v1/tools/collection_summary
```

The API key identifies the user server-side. Clients never send a user ID; tenant isolation is enforced at the handler layer.

### OpenAPI Document

The external tool server publishes a scoped OpenAPI 3.0 document at:

```
GET /api/v1/tools/openapi.json
```

This endpoint is **unauthenticated** (respects the admin kill switch only) and returns the spec as JSON suitable for client auto-import. The document describes request/response schemas, capability requirements, error codes, and the `X-API-Key` security scheme.

### Error Responses

All errors follow a consistent JSON format:

```json
{
  "error": "Error message"
}
```

| Status Code | Meaning |
|---|---|
| `400 Bad Request` | Invalid request body, disallowed field change, or invalid confirmation |
| `401 Unauthorized` | Missing, invalid, or revoked API key; or invalid proposal token |
| `403 Forbidden` | Insufficient capability (e.g., read-only key on a write tool) |
| `404 Not Found` | Coin or proposal not found, or cross-user access denied |
| `409 Conflict` | Proposal is not in `pending` state or has expired |
| `429 Too Many Requests` | Per-key rate limit exceeded |
| `503 Service Unavailable` | External tool server is disabled (admin toggle is OFF) |

### MCP Compatibility

The external tool server does not ship a native MCP server in v1. Instead, you can wrap the served OpenAPI document with an existing MCP proxy like [mcpo](https://github.com/QuantGeekDev/mcpo) to expose the tools to MCP clients (Claude Desktop, Cline, etc.).

**Using mcpo:**

1. Install mcpo:

   ```sh
   npm install -g mcpo
   ```

2. Start the proxy pointing at your external tool server:

   ```sh
   mcpo \
     --openapi http://localhost:8080/api/v1/tools/openapi.json \
     --header "X-API-Key: ak_your_key_here"
   ```

3. Configure your MCP client to connect to the mcpo proxy (stdio or SSE).

The proxy translates MCP tool calls into HTTP requests against the OpenAPI spec and forwards responses back to the MCP client.

### Two-Phase Write Flow

External writes use the same two-phase confirm pattern as in-app collection chat:

1. **Propose** (`propose_update`) — Call with a coin ID and allowlisted field changes. The server validates fields, checks ownership, and returns a proposal ID, token, and preview **without writing anything**.
2. **Commit** (`commit_update`) — Call with the proposal ID, token, and explicit `confirm=true`. The server verifies the token and proposal state, persists the change, and journals the write.

This prevents accidental or conversational auto-writes. Proposals expire after a configurable TTL (default: 5 minutes).

---

## Client Setup Guides

The following guides show how to connect popular external AI clients to your Ancient Coins instance. All require an API key and the OpenAPI URL from your running instance.

### OpenWebUI / Ollama

OpenWebUI supports OpenAPI tool import natively.

**Prerequisites:**

- OpenWebUI installed and running
- An API key with `read` or `read,write` capability from your Ancient Coins instance

**Steps:**

1. Open OpenWebUI in your browser
2. Navigate to **Settings → Tools**
3. Click **Import Tool** and select **OpenAPI URL**
4. Enter the OpenAPI URL:
   ```
   http://your-ancient-coins-host:8080/api/v1/tools/openapi.json
   ```
5. Add a custom header:
   - **Header Name:** `X-API-Key`
   - **Header Value:** `ak_your_key_here`
6. Click **Import**
7. The six tools appear in your tool list and are available in chats

**Testing:**

Open a new chat and ask: "What is the total value of my coin collection?"

OpenWebUI will call `collection_summary` and display the result.

---

### LibreChat

LibreChat supports OpenAPI tool import via its admin panel.

**Prerequisites:**

- LibreChat installed and running
- Admin access to LibreChat
- An API key with `read` or `read,write` capability from your Ancient Coins instance

**Steps:**

1. Log in to LibreChat as admin
2. Navigate to **Admin Panel → Tools → OpenAPI**
3. Click **Add OpenAPI Tool**
4. Fill in the form:
   - **Name:** Ancient Coins
   - **OpenAPI URL:** `http://your-ancient-coins-host:8080/api/v1/tools/openapi.json`
   - **Headers:** Add `X-API-Key` with your API key value
5. Click **Save**
6. Navigate to a conversation and select the **Ancient Coins** tool from the tool picker

**Testing:**

In a conversation, ask: "Search my collection for denarii of Nero"

LibreChat will call `search_my_collection` and format the results.

---

### n8n

n8n supports HTTP requests with OpenAPI schema import.

**Prerequisites:**

- n8n installed and running
- An API key with `read` or `read,write` capability from your Ancient Coins instance

**Steps:**

1. Create a new workflow in n8n
2. Add an **HTTP Request** node
3. Configure the node:
   - **Method:** POST
   - **URL:** `http://your-ancient-coins-host:8080/api/v1/tools/search_my_collection` (or another tool endpoint)
   - **Authentication:** Generic Credential Type
   - **Generic Auth Type:** Header Auth
   - **Header Name:** `X-API-Key`
   - **Header Value:** `ak_your_key_here`
4. Set the request body:
   ```json
   {
     "query": "{{ $json.searchQuery }}",
     "limit": 10
   }
   ```
5. Test the workflow with an input trigger

**Advanced:** Import the full OpenAPI spec into n8n using the **OpenAPI** node for automatic operation generation.

**Testing:**

Trigger the workflow with `searchQuery: "augustus"` and inspect the response.

---

## Best Practices & Troubleshooting

### Best Practices

**Least-Privilege Keys** — Create separate API keys for each external client, each with the minimum capability needed:

- **Read-only keys** for query-only integrations (OpenWebUI exploration, analytics dashboards)
- **Read+write keys** only for trusted automation that requires updates (AI-driven valuations, bulk tagging)

Revoke keys immediately if they are no longer needed or if a client is compromised.

**Rate Limit Awareness** — External tools are rate-limited at **50 requests per minute per API key**. Design integrations to batch queries when possible and handle `429 Too Many Requests` responses gracefully with exponential backoff.

**Proposal Expiry** — Proposals expire after a configurable TTL (default: 5 minutes). If your workflow holds a proposal token for longer than the expiry window, the commit will fail with `409 Conflict`. Re-propose if needed.

**Field Allowlist** — Only the following fields can be updated via external tools:

- `grade`
- `currentValue`
- `notes`
- `tags`
- `referenceText`
- `referenceUrl`
- `references`

Attempting to update identity fields (name, category, era, ruler, etc.) returns `400 Bad Request`. If you need to update identity fields, use the in-app UI.

**Journal Review** — Periodically review the activity journal on coins updated via external tools. Each entry shows the source (`external_tool_server`), API key name, and changed fields. This audit trail helps trace unexpected changes to a specific integration or key.

### Troubleshooting

**Tools return `503 Service Unavailable`** — The external tool server is disabled. Enable it in **Admin → System Settings → External Tool Server Enabled**.

**Tools return `401 Unauthorized`** — Your API key is invalid, revoked, or not being sent correctly. Check:

- The key is correctly copied (starts with `ak_`)
- The `X-API-Key` header is set in your client
- The key has not been revoked in Settings → Data → API Keys

**Tools return `403 Forbidden`** — Your API key lacks the required capability:

- Read tools require a key with `read` or `read,write` capability
- Write tools require a key with `read,write` capability

Create a new key with the correct scope.

**`commit_update` returns `409 Conflict`** — Your proposal has expired or has already been committed. Re-propose the change and commit immediately.

**Cross-user access returns `404 Not Found`** — This is expected behavior. External tools are strictly user-scoped to the API key owner. You cannot access another user's coins, even with admin credentials. Use the in-app UI for cross-user admin operations.

---

## Security Considerations

See [threat-model.md](threat-model.md) for a full security analysis of the external tool server, including per-key rate limiting, capability scoping, journaling audit trails, and tenant isolation guarantees.

---

## Related Documentation

- [API Reference](api-reference.md) — Full REST API reference for in-app endpoints
- [Features](features.md) — Overview of all application features
- [Threat Model](threat-model.md) — Security findings and mitigations
- [Authentication](authentication.md) — Auth methods and API key management
- [Getting Started](getting-started.md) — First-time setup and basic usage
