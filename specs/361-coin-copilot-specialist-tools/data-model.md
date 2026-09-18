# Data Model: Coin Copilot Specialist Market Tools

## 1. Persistence impact

No SQLite schema change is required.

Feature 361 reuses:

```text
CoinCopilotRun
  1 ── * CoinCopilotCheckpoint
  1 ── * CoinCopilotEvent
```

The normalized specialist result is stored inside the existing
`CoinCopilotCheckpoint.completed_tools[].result` JSON field. Go remains the
sole durable owner. Python stores no run, cache, provider attempt, or result.

## 2. Specialist Capability

| Field | Type | Rules |
|---|---|---|
| `name` | enum | `market_search`, `auction_search`, `price_trends`, `similar_lots` |
| `input` | Specialist Query | strict, capability-specific |
| `team` | internal mapping | existing `coin_search`, `auction_search`, `price_trends`, or `similar_lots` |
| `sourcePolicy` | internal id | registered provider/source boundary; never model-selected |
| `budgetCost` | integer | exactly one existing Coin Copilot tool call |

### Specialist Query

| Field | Type | Rules |
|---|---|---|
| `query` | string | 1–500 characters |
| `limit` | integer | optional, 1–10, default 5 |

Unknown fields are rejected. A model may first use `get_coin` and then compose
a query from validated collection facts; specialist tools do not accept an
owner id, run id, URL, provider, arbitrary headers, or executable instructions.

## 3. Specialist Result Envelope

| Field | Type | Rules |
|---|---|---|
| `schema_version` | integer | exactly `1` |
| `capability` | enum | must match the invoked tool |
| `outcome` | enum | `complete`, `partial`, `no_match`, `unavailable` |
| `items` | Evidence Item[] | maximum 10; discriminated by `kind` |
| `trend` | Price Trend Summary or null | non-null only for `price_trends` |
| `provider_attempts` | Provider Attempt[] | maximum 10 |
| `warnings` | string[] | maximum 10; 500 characters each; client-safe |
| `truncation` | Truncation Metadata | required |

Validation invariants:

- `complete` and `partial` require at least one valid item.
- `no_match` and `unavailable` require zero items.
- `partial` requires at least one degraded provider attempt.
- `no_match` permits only successful/no-match attempts.
- `unavailable` requires a timeout, failure, unavailable, or malformed attempt.
- `trend` is absent for the other three capabilities.
- Raw provider bodies, prompts, exceptions, credentials, hidden reasoning, and
  arbitrary metadata are forbidden.

## 4. Provider Attempt

This is a normalized part of the tool result, not a database entity.

| Field | Type | Rules |
|---|---|---|
| `provider` | string | registered provider id, 1–64 chars |
| `status` | enum | `success`, `no_match`, `timeout`, `failure`, `unavailable`, `malformed` |
| `observed_at` | RFC3339 UTC timestamp | application observation time |
| `accepted_items` | integer | 0–10 |
| `warning_code` | string or null | allowlisted safe code, not raw exception |

Provider attempt ordering follows the canonical team's configured provider
order and is stable in persisted output.

## 5. Evidence Item

### Shared fields

| Field | Type | Rules |
|---|---|---|
| `kind` | enum | `dealer_listing`, `auction_lot`, `sale_observation`, `similar_lot` |
| `source_url` | URL string | HTTPS, no user-info, allowed host/source policy, max 2,048 |
| `canonical_source_id` | string | normalized URL identity, max 2,048 |
| `provider` | string | registered provider id |
| `observed_at` | RFC3339 UTC timestamp | required |
| `confidence` | enum | `high`, `medium`, `low` |
| `verification_state` | enum | `verified`, `partial` |
| `title` | string | required, max 300 |
| `description` | string or null | max 500 |
| `provenance` | Field Provenance[] | 1–20 |

An item with an invalid URL or no provenance is rejected. A missing optional
fact remains null and has no provenance entry.

### Dealer Listing

Optional source-backed fields: `dealer_name`, `listed_price`, `currency`,
`availability`, `ruler`, `denomination`, `era`, `material`.

`availability` is `available`, `sold`, or `unknown`; it is never inferred from
absence.

### Auction Lot

Optional source-backed fields: `auction_house`, `sale_name`, `lot_number`,
`sale_date`, `estimate`, `current_bid`, `currency`, `lot_status`, `ruler`,
`denomination`, `era`, `material`.

### Sale Observation

Required source-backed fields for trend use:

- `sale_date`;
- `amount`;
- `currency`;
- `price_basis` (`hammer` or `realized_including_premium`).

Estimates, current bids, and listings may be returned as other evidence but do
not count as completed-sale trend samples.

### Similar Lot

Additional fields:

| Field | Type | Rules |
|---|---|---|
| `similarity_score` | decimal | 0 through 1 |
| `matched_attributes` | string[] | 1–20 |
| `material_differences` | string[] | 0–20 |

The ranking is descending score, then canonical source id.

## 6. Field Provenance

| Field | Type | Rules |
|---|---|---|
| `field` | enum/string | one declared field on the containing item |
| `source_url` | URL | exactly the item's validated source URL |
| `observed_at` | RFC3339 UTC timestamp | required |
| `confidence` | enum | `high`, `medium`, `low` |
| `verification_state` | enum | `verified`, `partial` |

`field` entries are unique per item. Values are held on the evidence item, not
duplicated inside provenance, avoiding disagreement between two copies.

## 7. Price Trend Summary

| Field | Type | Rules |
|---|---|---|
| `state` | enum | `rising`, `stable`, `declining`, `unknown` |
| `sample_size` | integer | count after validation/deduplication |
| `date_from`, `date_to` | date or null | observed sale coverage |
| `currency` | string or null | one comparable currency only |
| `price_basis` | enum or null | one comparable basis only |
| `low`, `median`, `high` | decimal or null | derived only from the comparable sample |
| `confidence` | enum | `high`, `medium`, `low` |
| `limitations` | string[] | maximum 10 |
| `supporting_source_ids` | string[] | references evidence items in the same result |

Directional states require at least three verified observations across two
dates spanning 30 days, all with the same currency and price basis. Otherwise
`state=unknown`; sample metadata remains visible.

## 8. Truncation Metadata

| Field | Type | Rules |
|---|---|---|
| `truncated` | boolean | whether item/field/byte limits omitted data |
| `original_bytes` | integer | canonical sanitized size before byte truncation |
| `persisted_bytes` | integer | final canonical size |
| `digest` | string | SHA-256 of full sanitized normalized result |
| `omitted_items` | integer | count omitted by structural limit |

The generic Feature 359 fallback envelope remains available if the normalized
result still exceeds the snapshotted per-tool byte limit. A final answer must
not imply that omitted evidence was reviewed.

## 9. Deduplication state transition

```text
raw provider candidate
  ├─ invalid source/shape ──> omitted + malformed warning/attempt
  └─ valid
       ├─ new canonical source id ──> accepted
       └─ duplicate
            ├─ compatible/stronger provenance ──> merge
            └─ conflicting observed fact ──> retain best-supported value
                                                + conflict warning
```

Duplicates never increase `sample_size`.

## 10. Existing run/checkpoint behavior

- `completed_tools[].tool_name` accepts the four new names.
- `completed_tools[].result` is validated by tool name in Python and Go.
- Existing `original_bytes`, `persisted_bytes`, `truncated`, and
  `result_digest` fields remain authoritative.
- A completed tool call id is immutable and replayed from the latest
  checkpoint.
- Existing run status transitions, owner scoping, event sequencing, retention,
  deletion, and terminal race rules do not change.

## 11. Public projection

For the four specialist tools only, Go derives
`tool_completed.payload.specialistResult`:

```text
SpecialistPublicResult
├── capability
├── outcome
├── items[]                 # safe display fields + provenance
├── trend?                  # price_trends only
├── warnings[]
└── truncation
```

It is an additive, sanitized projection of the validated checkpoint result,
not a second source of truth. It contains no query, raw provider payload, tool
arguments, credentials, internal errors, or hidden reasoning.
