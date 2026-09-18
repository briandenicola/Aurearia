# OpenAPI Impact: Feature 363

## Public API additions

The generated Go Swagger/OpenAPI surface gains:

| Method/path | Purpose | Mutation |
|---|---|---|
| `GET /api/collector/profile` | Read owned profile or neutral defaults | No |
| `PUT /api/collector/profile` | Atomic optimistic profile/goal replacement | Yes, profile only |
| existing Coin Copilot start/stream routes | Add closed collector workflow request/result variants | Run durability only |
| `POST /api/collector/wishlist-actions/stages` | Create immutable review stage from retained eligible result | Stage only; no Coin |
| `POST /api/collector/wishlist-actions/stages/{stageId}/confirm` | Confirm exact stage and create one wishlist Coin | Yes, confirm-gated |

The stage/confirm excerpt is normative in
`wishlist-action.openapi.yaml`. Profile and read-only value details are
normative in `collector-workflows.md` and must be reflected in handler Swagger
models.

## Explicit non-additions

- No public Python endpoint.
- No browser-to-agent path.
- No generic action/write/tool endpoint.
- No stage/confirm `/api/internal/copilot/tools/*` callback.
- No public cross-user profile, recommendation, risk or audit listing.
- No autonomous buying, bidding, scheduling or provider-fetch endpoint.

## Compatibility

All new paths are additive. Existing Coin Copilot SSE event names remain
unchanged; typed optional workflow-result projections are additive to existing
completed-tool payloads. Old clients ignore the optional field. Existing
`POST /api/coins`, wishlist availability, auction, alert-candidate conversion,
Deep Analysis and legacy chat contracts remain unchanged.

## Generation gate

Implementation must add Swagger annotations to each public Go handler, then run:

```powershell
task openapi
git diff --exit-code -- src/api/docs/docs.go src/api/docs/swagger.json src/api/docs/swagger.yaml docs/openapi.json
Push-Location src/api
go test . -run TestRegisteredAPIRoutesAreDocumentedInOpenAPI -count=1
Pop-Location
```

Contract tests must also prove no stage/confirm route appears in the internal
Coin Copilot callback allowlist and no Python host is emitted in generated
public documents.
