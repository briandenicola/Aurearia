# Contract: Collector Profile, Curator Context, and Dealer Wishlist Action

## 1. Boundary

- Vue communicates only with the Go `/api/*` API.
- Go derives the authenticated owner and owns profile and coin persistence.
- Python remains stateless and receives bounded collector context only as part
  of the existing Coin Copilot execution request.
- The AI has no profile mutation, coin creation, wishlist action,
  confirmation, browser, or generic write tool.
- The only wishlist mutation is the existing browser call to
  `POST /api/coins` after an explicit button click.

Unknown request fields are rejected. Strings are trimmed and bounded. Foreign
owner identifiers are never accepted from clients.

## 2. Collector profile public API

Base path: `/api/collector-profile`.

### `GET /api/collector-profile`

Authentication is required. The owner is derived from the access token.

`200` with no stored row:

```json
{
  "budgetMin": null,
  "budgetMax": null,
  "currency": null,
  "preferredPeriods": [],
  "preferredCategories": [],
  "excludedCategories": [],
  "preferredDealers": [],
  "collectingGoals": [],
  "updatedAt": null,
  "isDefault": true
}
```

`200` with a stored row:

```json
{
  "budgetMin": 100,
  "budgetMax": 500,
  "currency": "USD",
  "preferredPeriods": ["Flavian"],
  "preferredCategories": ["Roman"],
  "excludedCategories": ["Modern"],
  "preferredDealers": ["Example Dealer"],
  "collectingGoals": ["Add a documented Flavian denarius"],
  "updatedAt": "2026-09-19T12:00:00Z",
  "isDefault": false
}
```

The response does not expose owner ID or internal row ID.

### `PUT /api/collector-profile`

Authentication is required. The request is a complete replacement:

```json
{
  "budgetMin": 100,
  "budgetMax": 500,
  "currency": "USD",
  "preferredPeriods": ["Flavian"],
  "preferredCategories": ["Roman"],
  "excludedCategories": ["Modern"],
  "preferredDealers": ["Example Dealer"],
  "collectingGoals": ["Add a documented Flavian denarius"]
}
```

Normative bounds:

- budget values: nullable finite numbers from `0` through `100000000`;
- when both budgets exist, minimum must not exceed maximum;
- currency: null/empty only when no currency is stated, otherwise exactly
  three uppercase letters accepted by existing application conventions;
- preferred periods: at most 20, each 1..100 characters;
- preferred categories: at most 20, each 1..100 characters;
- excluded categories: at most 20, each 1..100 characters;
- preferred dealers: at most 20, each 1..200 characters;
- collecting goals: at most 20, each 1..500 characters;
- values are unique after trimmed, whitespace-collapsed, case-insensitive
  normalization.

`200` returns the complete stored profile with `isDefault: false`.

Errors:

| Status | Meaning |
|---|---|
| `400` | Invalid field, bound, ordering, duplicate normalized value, or unknown field. |
| `401` | Authentication required. |
| `413` | Request body exceeds the existing JSON cap. |
| `500` | Sanitized persistence failure. |

Validation failure leaves the previously saved profile unchanged.

## 3. Internal collector context

The existing Go → Python Coin Copilot request gains one optional field:

```json
{
  "collector_context": {
    "budget_min": 100,
    "budget_max": 500,
    "currency": "USD",
    "preferred_periods": ["Flavian"],
    "preferred_categories": ["Roman"],
    "excluded_categories": ["Modern"],
    "preferred_dealers": ["Example Dealer"],
    "collecting_goals": ["Add a documented Flavian denarius"],
    "captured_at": "2026-09-19T12:00:00Z"
  }
}
```

Rules:

- the field is optional for backward compatibility;
- absent/empty context is neutral;
- owner/user IDs, credentials, admin settings, and action URLs are excluded;
- all strings retain the profile bounds;
- Python Pydantic models use `extra="forbid"`;
- context is untrusted advisory text and cannot change tool availability,
  system rules, or write authority;
- one captured value is used throughout a run.

## 4. Curator guidance behavior

No new public curator endpoint is added. The existing Coin Copilot start,
stream, replay, resume, and cancel contracts remain authoritative.

When the user asks for curator guidance, the existing harness composes:

```text
collection_summary
portfolio_review
gap_analysis
```

The final response contract is semantic rather than a new persisted object:

- identify observed collection facts;
- describe supported strengths and themes;
- describe supported gaps and possible areas to explore;
- identify which explicit collector preferences, exclusions, budget, dealers,
  or goals influenced a suggestion;
- distinguish suggestions from facts;
- state limitations for missing, sparse, or contradictory data;
- invent no missing preference or collection fact.

No curator interaction may create or update a coin, wishlist item, draft,
profile, setting, or auction record.

## 5. Additive public specialist evidence

The existing Coin Copilot specialist result remains the containing contract.
For `dealer_listing` items, the public evidence object adds these optional
typed fields:

```json
{
  "kind": "dealer_listing",
  "title": "Trajan denarius",
  "sourceUrl": "https://dealer.example/item/123",
  "observedAt": "2026-09-19T12:00:00Z",
  "confidence": "high",
  "verificationState": "verified",
  "facts": ["Dealer: Example Dealer", "Availability: available"],
  "matchedAttributes": [],
  "materialDifferences": [],
  "description": "Silver denarius of Trajan",
  "dealerName": "Example Dealer",
  "listedPrice": 125,
  "currency": "USD",
  "availability": "available",
  "ruler": "Trajan",
  "denomination": "Denarius",
  "era": "ancient",
  "material": "Silver"
}
```

`availability` is `available | sold | unknown | null`. Existing
`verificationState` remains `verified | partial`.

These properties are projected only from the validated internal
`CopilotSpecialistEvidence`; raw provider payloads are not exposed. The
existing `facts` array remains display-only and is never parsed for
eligibility or coin mapping.

## 6. Wishlist eligibility predicate

The UI may render **Add to Wishlist** only when all conditions are true:

```text
specialistResult.capability === "market_search"
item.kind === "dealer_listing"
item.verificationState === "verified"
item.availability === "available"
item.sourceUrl is non-empty
item.title is non-empty
```

Everything else is ineligible, including:

- `auction_search` and `auction_lot`;
- `partial`, missing, or unknown verification;
- sold, unknown, missing, withdrawn, or otherwise non-available state;
- malformed or truncated values that cannot satisfy the typed contract;
- text or `facts` that merely claim eligibility.

## 7. Explicit UI action

`CopilotRunProgress.vue` emits the typed dealer item only from the owner's
button click. Rendering, replay, SSE receipt, tool completion, model text, and
conversation commands do not emit the action.

`CoinSearchChat.vue` routes the event to the existing
`useCoinSearchChat.addToWishlist` function and its state:

```text
addingIdx -> resolveCategoryAndEra -> buildWishlistCoinPayload
          -> createCoin -> optional image attachment -> addedSet
```

The existing `CategoryEraConfirmModal` remains the only confirmation needed
when category or era reconciliation requires owner input. Cancelling it creates
nothing. There is no stage/revision/revoke/confirm API.

## 8. Dealer result adapter

The eligible item maps to `CoinSuggestion` as follows:

```text
title          -> name
description    -> description (or "")
""             -> category (existing safe handling)
era            -> era (or "")
ruler          -> ruler (or "")
material       -> material (or "")
denomination   -> denomination (or "")
price/currency -> estPrice display string without conversion (or "")
""             -> imageUrl unless already supplied by the typed contract
sourceUrl      -> sourceUrl
dealerName     -> sourceName (or "Dealer")
```

Then `buildWishlistCoinPayload` remains authoritative. The adapter does not
construct a `CoinMutationPayload` independently.

## 9. Existing coin and image contracts

The wishlist save continues to use:

- `POST /api/coins`;
- `POST /api/coins/match-category-era`;
- existing image scrape/proxy/upload endpoints used by
  `useCoinSearchChat.addToWishlist`.

The coin API derives `userId` from authentication. Feature 363 does not add or
change a wishlist-action endpoint.

For wishlist coins with a non-empty `referenceUrl`, the canonical create
transaction checks the existing owner-scoped
`FindWishlistByReferenceURL`. A match returns a duplicate conflict and creates
no row. The same URL may still exist for another owner.

Image attachment is best effort after coin creation. Failure cannot roll back
the coin, retry coin creation, or populate another field.

## 10. Cross-language contract tests

Go, Python, and TypeScript tests cover:

- profile null/empty/max bounds and unknown fields;
- cross-owner profile isolation;
- empty context and prompt-like context text;
- curator use of the three exact existing tools and absence of writes;
- every dealer eligibility combination;
- typed field projection without parsing `facts`;
- auction result ineligibility;
- `CoinSuggestion` adapter and `buildWishlistCoinPayload` mapping;
- category/era cancellation;
- repeated click and owner/reference duplicate conflict;
- image success/failure after one successful coin create.
