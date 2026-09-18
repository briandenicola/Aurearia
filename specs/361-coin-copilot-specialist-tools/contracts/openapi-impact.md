# OpenAPI Impact: Coin Copilot Specialist Market Tools

## Decision

Feature 361 adds no public REST endpoint and does not expose the internal
Python specialist tools as HTTP APIs. Existing Feature 359 paths remain:

- `GET /api/agent/copilot/capability`
- `POST /api/agent/copilot/runs`
- `GET|DELETE /api/agent/copilot/threads/{threadId}`
- `GET /api/agent/copilot/runs/{runId}`
- `GET /api/agent/copilot/runs/{runId}/events`
- `POST /api/agent/copilot/runs/{runId}/cancel`
- `POST /api/agent/copilot/runs/{runId}/resume`

Request bodies, status codes, owner-scoping behavior, and fallback behavior do
not change.

## Additive SSE documentation

The `text/event-stream` response still uses the existing ten event names.
`tool_completed.payload` receives optional `specialistResult` as defined in
`public-sse-extension.md`. Existing consumers that ignore unknown object fields
remain compatible.

If the generated Swagger model currently represents the SSE response as an
opaque string, document the additive payload in:

- Swagger handler description/annotations;
- `docs/api-reference.md`;
- `docs/features/coin-copilot.md`;
- this feature's SSE contract.

Do not invent a public endpoint merely to make the specialist schema appear as
a REST resource.

## Drift gate

Implementation must regenerate rather than hand-edit generated files:

```powershell
task openapi
git diff --exit-code -- src/api/docs/docs.go src/api/docs/swagger.json src/api/docs/swagger.yaml docs/openapi.json

Push-Location src\api
go test . -run TestRegisteredAPIRoutesAreDocumentedInOpenAPI -count=1
Pop-Location
```

Contract tests must also prove:

1. no new `/api/internal/copilot/tools/{specialist}` route is registered;
2. internal routes remain absent from public OpenAPI;
3. all public Coin Copilot routes remain documented;
4. any generated SSE description and the committed Markdown contract name the
   same four capabilities and outcome values;
5. Feature 359 request/response fixtures remain valid.
