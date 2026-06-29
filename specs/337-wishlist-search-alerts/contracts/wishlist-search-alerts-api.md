# API Contract: Wishlist Search Alerts

Base path: `/api/wishlist/search-alerts`  
Auth: Bearer token required for every endpoint. All resources are scoped to `userId` from the token; cross-user access returns `404` or generic `403` without leaking existence.

## Shared enums

```text
cadence: manual | daily | weekly | monthly
triggerType: manual | scheduled
runStatus: queued | running | completed | failed | partial | rate_limited | cancelled
provenanceStatus: verified | partial | unverified
candidateState: active | dismissed | converted | suppressed | needs_review
dismissalReason: irrelevant | duplicate | price_too_high | poor_provenance | other
```

## Alert object

```json
{
  "id": 1,
  "name": "Domitian denarius under $300",
  "criteria": {
    "rulerOrIssuer": "Domitian",
    "coinType": "Denarius",
    "dateFrom": 81,
    "dateTo": 96,
    "mint": "Rome",
    "material": "Silver",
    "gradeOrCondition": "VF or better",
    "priceMin": 0,
    "priceMax": 300,
    "currency": "USD",
    "dealerPreference": "VCoins or MA-Shops",
    "sourceFilters": ["vcoins.com", "ma-shops.com"],
    "keywords": "RIC Minerva",
    "notes": "Prefer clear legends"
  },
  "cadence": "manual",
  "isActive": true,
  "lastRunAt": "2026-06-29T17:00:00Z",
  "createdAt": "2026-06-29T17:00:00Z",
  "updatedAt": "2026-06-29T17:00:00Z"
}
```

## `GET /api/wishlist/search-alerts`

List alerts for the authenticated user.

Query:
- `active`: optional `true` / `false`
- `page`: default `1`
- `limit`: default `20`, max `100`

Response `200`:

```json
{
  "alerts": [],
  "total": 0,
  "page": 1,
  "limit": 20
}
```

## `POST /api/wishlist/search-alerts`

Create an alert.

Request:

```json
{
  "name": "Domitian denarius under $300",
  "criteria": {
    "rulerOrIssuer": "Domitian",
    "coinType": "Denarius",
    "dateFrom": 81,
    "dateTo": 96,
    "mint": "Rome",
    "material": "Silver",
    "gradeOrCondition": "VF or better",
    "priceMin": 0,
    "priceMax": 300,
    "currency": "USD",
    "dealerPreference": "VCoins or MA-Shops",
    "sourceFilters": ["vcoins.com", "ma-shops.com"],
    "keywords": "RIC Minerva",
    "notes": "Prefer clear legends"
  },
  "cadence": "manual",
  "isActive": true
}
```

Response `201`: alert object.

Validation errors `400`:
- no meaningful criteria
- invalid price/date range
- unsupported cadence
- malformed source/domain filter
- strings exceed limits

## `GET /api/wishlist/search-alerts/{alertId}`

Get one owned alert with latest run summary.

Response `200`: alert object with optional `latestRun`.

## `PUT /api/wishlist/search-alerts/{alertId}`

Update alert criteria, cadence, name, or active state. Previous run snapshots remain unchanged.

Response `200`: updated alert object.

## `DELETE /api/wishlist/search-alerts/{alertId}`

Soft-delete or disable alert for future runs while preserving run/candidate history.

Response `204`.

## `POST /api/wishlist/search-alerts/{alertId}/run`

Manual Run Now. Active alerts only. Enforces per-user/per-alert run limits before calling the agent.

Request:

```json
{
  "maxCandidates": 20
}
```

Response `202` or `200` (implementation may run synchronously for MVP):

```json
{
  "runId": 42,
  "alertId": 1,
  "status": "completed",
  "startedAt": "2026-06-29T17:00:00Z",
  "completedAt": "2026-06-29T17:00:10Z",
  "resultCount": 8,
  "newCount": 5,
  "duplicateCount": 3,
  "partialWarnings": ["3 candidates omitted because run result cap was reached"],
  "rateLimitStatus": "ok"
}
```

Errors:
- `400`: alert disabled/deleted or invalid maxCandidates
- `409`: run already in progress for alert
- `429`: per-alert/per-user limit exceeded
- `503`: agent unavailable; run stored as failed with sanitized message

## `GET /api/wishlist/search-alerts/{alertId}/runs`

List run history for one alert.

Response:

```json
{
  "runs": [],
  "total": 0,
  "page": 1,
  "limit": 20
}
```

## `GET /api/wishlist/search-alerts/{alertId}/runs/{runId}`

Get run detail and candidates produced/updated by that run.

Response includes `criteriaSnapshot`, counts, warnings, and candidates.

## `GET /api/wishlist/search-alerts/{alertId}/candidates`

List candidates for review.

Query:
- `state`: optional candidate state; default active/needs_review
- `provenanceStatus`: optional
- `page`, `limit`

Candidate object:

```json
{
  "id": 99,
  "alertId": 1,
  "runId": 42,
  "sourceUrl": "https://www.vcoins.com/en/stores/example/123",
  "canonicalSourceUrl": "https://www.vcoins.com/en/stores/example/123",
  "sourceName": "VCoins Example Dealer",
  "title": "Domitian AR Denarius",
  "observedPrice": 225.0,
  "observedCurrency": "USD",
  "reasonForMatch": "Title and description mention Domitian denarius; price is under alert maximum.",
  "lastSeenAt": "2026-06-29T17:00:10Z",
  "provenanceStatus": "verified",
  "lifecycleState": "active",
  "matchingWishlistCoinId": null,
  "convertedCoinId": null,
  "provenance": [
    {
      "field": "observed_price",
      "value": "$225.00",
      "sourceUrl": "https://www.vcoins.com/en/stores/example/123",
      "observedAt": "2026-06-29T17:00:10Z",
      "confidence": "high",
      "verificationState": "verified"
    }
  ]
}
```

## `POST /api/wishlist/search-alerts/{alertId}/candidates/{candidateId}/dismiss`

Dismiss a candidate.

Request:

```json
{
  "reason": "price_too_high",
  "notes": "Too expensive with shipping"
}
```

Response `200`: updated candidate.

## `POST /api/wishlist/search-alerts/{alertId}/candidates/{candidateId}/restore`

Restore a dismissed candidate to active review.

Response `200`: updated candidate.

## `POST /api/wishlist/search-alerts/{alertId}/candidates/{candidateId}/convert`

Explicitly create a normal wishlist item from a candidate. The service must prefill only source-backed fields and require the caller to provide missing required fields/overrides.

Request:

```json
{
  "coin": {
    "name": "Domitian AR Denarius",
    "category": "Roman",
    "denomination": "Denarius",
    "ruler": "Domitian",
    "era": "ancient",
    "mint": "Rome",
    "material": "Silver",
    "grade": "",
    "purchasePrice": 225.0,
    "currentValue": 225.0,
    "purchaseLocation": "VCoins Example Dealer",
    "referenceUrl": "https://www.vcoins.com/en/stores/example/123",
    "referenceText": "Source-backed candidate from wishlist search alert #1",
    "notes": "Converted from alert candidate #99",
    "isWishlist": true
  },
  "acknowledgeDuplicateWarning": false
}
```

Response `201`:

```json
{
  "coin": {},
  "candidate": {
    "id": 99,
    "lifecycleState": "converted",
    "convertedCoinId": 123
  },
  "warnings": []
}
```

Errors:
- `400`: required wishlist fields missing or uncertain without override
- `409`: candidate already converted or duplicate wishlist item exists and warning not acknowledged
- `404`: candidate not owned by user

## `POST /api/wishlist/search-alerts/{alertId}/criteria-adjustments`

Optional helper from review context. Updates alert criteria and records a `criteria_adjusted` review action for referenced candidates.

Request:

```json
{
  "candidateIds": [99],
  "criteria": {
    "priceMax": 250,
    "sourceFilters": ["vcoins.com"]
  }
}
```

Response `200`: updated alert.

## Contract guarantees

- Alert endpoints never create `AvailabilityRun` / `AvailabilityResult`.
- Alert runs never mutate existing `Coin.ListingStatus`.
- Converted candidates become normal `Coin` wishlist items and future availability checks operate on the saved `referenceUrl`.
- All errors returned to users are sanitized.
- Public handlers must include Swagger annotations and generated docs must be refreshed during implementation.
