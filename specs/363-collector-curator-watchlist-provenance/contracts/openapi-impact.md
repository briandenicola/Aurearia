# OpenAPI Impact: Feature 363

## Public API additions

Only the collector profile adds public routes:

| Method/path | Purpose | Mutation |
|---|---|---|
| `GET /api/collector-profile` | Return the authenticated owner's saved profile or neutral defaults. | No |
| `PUT /api/collector-profile` | Validate and atomically replace the authenticated owner's profile. | Yes, profile only |

Both handlers require bearer authentication, derive the owner from middleware,
use bounded explicit DTOs, return sanitized errors, and include Swagger
annotations.

The profile schemas are normative in
[`collector-workflows.md`](./collector-workflows.md).

## Existing contracts reused unchanged

- Existing Coin Copilot start, SSE, replay, resume, and cancel routes remain
  the curator guidance transport.
- Existing `POST /api/coins` remains the only wishlist coin creation route.
- Existing `POST /api/coins/match-category-era` remains the category/era
  reconciliation route.
- Existing image scrape, proxy, and upload routes used by
  `useCoinSearchChat.addToWishlist` remain authoritative.

No separate wishlist action route is added.

## Additive response fields

The existing public Coin Copilot specialist evidence schema gains optional
dealer fields already present in the validated internal specialist result:

```text
description?: string | null
dealerName?: string | null
listedPrice?: number | null
currency?: string | null
availability?: "available" | "sold" | "unknown" | null
ruler?: string | null
denomination?: string | null
era?: string | null
material?: string | null
```

Old clients ignore these optional fields. Existing result envelope, event
names, capability values, and auction result rendering remain compatible.

## Explicit non-additions

- No `/api/wishlist-actions/*`.
- No stage, revision, revoke, expiry, or confirm route.
- No action/audit/outcome resource.
- No public curator workflow endpoint.
- No Python public endpoint or browser-to-Python call.
- No generic action/write/tool endpoint.
- No watchlist evaluation/ranking endpoint.
- No provenance-risk endpoint.
- No auction endpoint, route, or schema change.
- No provider-fetch or new browser endpoint.

## Generation gate

During implementation, add Swagger annotations for the two profile handlers,
regenerate the repository OpenAPI, and verify route/schema drift:

```powershell
task openapi
git diff --exit-code -- src/api/docs/docs.go src/api/docs/swagger.json src/api/docs/swagger.yaml docs/openapi.json

Push-Location src/api
go test . -run TestRegisteredAPIRoutesAreDocumentedInOpenAPI -count=1
Pop-Location
```

Contract tests must prove:

1. only the two profile routes are new;
2. no wishlist-action route exists;
3. no Python host or internal callback route is exposed;
4. specialist dealer fields are optional and bounded;
5. auction contracts are byte-for-byte unchanged by Feature 363.
