# Quickstart & Validation: External Tool Server Adapter (#218)

End-to-end scenarios to validate read/write parity, security scoping, and the kill
switch. Replace `BASE` with your API origin (e.g. `http://localhost:8080`).

## Prerequisites

1. Issue #217 tool layer is present (`CollectionToolsService`, proposals table).
2. An admin enables the surface: set `ExternalToolServerEnabled = true` in Admin
   Settings (default is off).
3. Create API keys in Settings → API Keys:
   - `READ_KEY` — read-only scope (default)
   - `WRITE_KEY` — read+write scope
4. Have at least one owned coin id (`COIN_ID`).

## Scenario A — External read (read-only key)

```bash
curl -s -X POST "$BASE/api/v1/tools/search_my_collection" \
  -H "X-API-Key: $READ_KEY" -H "Content-Type: application/json" \
  -d '{"query":"denarius","limit":5}'

curl -s -X POST "$BASE/api/v1/tools/get_coin" \
  -H "X-API-Key: $READ_KEY" -H "Content-Type: application/json" \
  -d "{\"coin_id\": $COIN_ID}"

curl -s -X POST "$BASE/api/v1/tools/collection_summary" \
  -H "X-API-Key: $READ_KEY" -H "Content-Type: application/json" -d '{}'

curl -s -X POST "$BASE/api/v1/tools/top_coins_by_value" \
  -H "X-API-Key: $READ_KEY" -H "Content-Type: application/json" -d '{"limit":3}'
```

**Expect**: 200 responses containing only the key owner's data. (SC-001)

## Scenario B — Two-phase external write (write key)

```bash
# Phase 1: propose
PROP=$(curl -s -X POST "$BASE/api/v1/tools/propose_update" \
  -H "X-API-Key: $WRITE_KEY" -H "Content-Type: application/json" \
  -d "{\"coin_id\": $COIN_ID, \"changes\": {\"notes\": \"verified via external client\"}}")
echo "$PROP"
# extract proposal.proposalId and proposal.proposalToken

# Phase 2: commit with explicit confirmation
curl -s -X POST "$BASE/api/v1/tools/commit_update" \
  -H "X-API-Key: $WRITE_KEY" -H "Content-Type: application/json" \
  -d "{\"proposal_id\":\"<ID>\",\"token\":\"<TOKEN>\",\"confirm\":true}"
```

**Expect**: propose returns a preview + token, no write yet; commit persists the
change and returns `journalSource: "external_tool_server"`. A `CoinJournal` entry is
recorded with source `external_tool_server` and the API key id/name + capability.
(SC-003, SC-004)

## Negative safety scenarios

### N1 — Read-only key cannot write (SC-005)

```bash
curl -s -o /dev/null -w "%{http_code}\n" -X POST "$BASE/api/v1/tools/propose_update" \
  -H "X-API-Key: $READ_KEY" -H "Content-Type: application/json" \
  -d "{\"coin_id\": $COIN_ID, \"changes\": {\"notes\":\"x\"}}"
```
**Expect**: `403` (insufficient capability), no proposal created.

### N2 — Cross-user access denied (SC-002)

Use a key owned by user A against a `coin_id` owned by user B.
**Expect**: `404`/`403`, no data leak.

### N3 — Non-allowlisted field rejected (FR-006)

```bash
curl -s -X POST "$BASE/api/v1/tools/propose_update" \
  -H "X-API-Key: $WRITE_KEY" -H "Content-Type: application/json" \
  -d "{\"coin_id\": $COIN_ID, \"changes\": {\"category\":\"Roman\"}}"
```
**Expect**: `400` invalid field change.

### N4 — Token replay / expiry (FR-005)

Commit an already-committed proposal again, or after expiry.
**Expect**: `409` (not pending / expired) or `401` invalid token; no second write.

### N5 — Kill switch off (SC-006)

Set `ExternalToolServerEnabled = false`, then call any `/api/v1/tools/*` endpoint.
**Expect**: `503` disabled, no side effects.

### N6 — Per-key rate limit (FR-010)

Exceed the configured per-key limit on reads and writes.
**Expect**: `429 Too Many Requests` once the bucket is exhausted.

## Scenario C — Client discovery & MCP

1. Fetch the scoped spec: `curl -s "$BASE/api/v1/tools/openapi.json"` → valid OpenAPI
   describing all six tools. (SC-007)
2. **OpenWebUI / LibreChat / n8n**: import the OpenAPI URL, set the `X-API-Key` header,
   and confirm the six tools appear and operate.
3. **MCP (mcpo)**: run `mcpo` pointed at the OpenAPI URL with the `X-API-Key` header to
   expose the tools to an MCP client; confirm read + two-phase write work.

(Exact client steps live in `docs/external-tool-server.md`.)

## Quality gate (§17)

```bash
# Go API (from src/api/)
go build ./... && go vet ./... && go test ./...

# Frontend (from src/web/)
npm run build && npm run lint

# Regenerate OpenAPI artifacts (from repo root)
task openapi
```
