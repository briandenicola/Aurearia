# Contract: Coin Copilot Specialist Tools (Go ↔ Python)

## 1. Compatibility

This contract extends Feature 359's internal execution contract. The execute
endpoint, frame envelope, schema version, run state machine, and existing six
tools remain unchanged.

The fixed allowlist becomes:

```json
[
  "search_my_collection",
  "get_coin",
  "collection_summary",
  "top_coins_by_value",
  "portfolio_review",
  "gap_analysis",
  "market_search",
  "auction_search",
  "price_trends",
  "similar_lots"
]
```

Only the last four are added by Feature 361. They are Python-local adapters and
MUST NOT have `/api/internal/copilot/tools/*` routes.

## 2. Tool inputs

All inputs reject unknown fields.

```json
{
  "query": "Domitian denarius Minerva",
  "limit": 5
}
```

| Field | Rules |
|---|---|
| `query` | required string, 1–500 characters |
| `limit` | optional integer, 1–10, default 5 |

The same shape is used for all four tools. The tool name determines the
canonical specialist team and eligible source policy. A caller cannot select a
provider, URL, credentials, owner, run, execution, retry count, or timeout.

## 3. Result envelope

```json
{
  "schema_version": 1,
  "capability": "auction_search",
  "outcome": "partial",
  "items": [
    {
      "kind": "auction_lot",
      "source_url": "https://www.numisbids.com/sale/10489/lot/1",
      "canonical_source_id": "https://www.numisbids.com/sale/10489/lot/1",
      "provider": "numisbids",
      "observed_at": "2026-09-18T12:00:00Z",
      "confidence": "high",
      "verification_state": "verified",
      "title": "Domitian denarius",
      "description": null,
      "auction_house": "Example House",
      "sale_name": "Sale 1",
      "lot_number": "1",
      "sale_date": "2026-10-01",
      "estimate": 250,
      "current_bid": null,
      "currency": "USD",
      "lot_status": "upcoming",
      "provenance": [
        {
          "field": "title",
          "source_url": "https://www.numisbids.com/sale/10489/lot/1",
          "observed_at": "2026-09-18T12:00:00Z",
          "confidence": "high",
          "verification_state": "verified"
        },
        {
          "field": "estimate",
          "source_url": "https://www.numisbids.com/sale/10489/lot/1",
          "observed_at": "2026-09-18T12:00:00Z",
          "confidence": "high",
          "verification_state": "verified"
        }
      ]
    }
  ],
  "trend": null,
  "provider_attempts": [
    {
      "provider": "numisbids",
      "status": "success",
      "observed_at": "2026-09-18T12:00:00Z",
      "accepted_items": 1,
      "warning_code": null
    },
    {
      "provider": "auction_search_model",
      "status": "timeout",
      "observed_at": "2026-09-18T12:00:01Z",
      "accepted_items": 0,
      "warning_code": "provider_timeout"
    }
  ],
  "warnings": ["One configured source timed out; verified results are incomplete."],
  "truncation": {
    "truncated": false,
    "original_bytes": 1450,
    "persisted_bytes": 1450,
    "digest": "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
    "omitted_items": 0
  }
}
```

The complete field definitions and invariants are in `../data-model.md`.

## 4. Capability-specific result rules

### `market_search`

- item kind: `dealer_listing`;
- source host must be accepted by the existing dealer-search source policy;
- dealer, listed price, availability, and coin attributes require matching
  field provenance;
- sold/unavailable items may describe evidence but must not be represented as
  currently available.

### `auction_search`

- item kind: `auction_lot`;
- uses the existing auction search and NumisBids fetch/parser boundary;
- auction house, sale, lot, estimate, bid, date, and status are optional unless
  observed and proven;
- invalid/unfetched search-result URLs cannot be promoted to verified lots.

### `price_trends`

- item kind: `sale_observation`;
- direction uses only deduplicated, completed-sale observations;
- currency and price basis are never silently mixed;
- fewer than three comparable verified samples, fewer than two dates, or less
  than 30 days of coverage yields `trend.state=unknown`;
- estimates/current bids may be disclosed separately but never counted as
  realized sales.

Example trend:

```json
{
  "state": "unknown",
  "sample_size": 2,
  "date_from": "2026-02-01",
  "date_to": "2026-08-01",
  "currency": "USD",
  "price_basis": "hammer",
  "low": 210,
  "median": 235,
  "high": 260,
  "confidence": "low",
  "limitations": ["Only two comparable completed sales were verified."],
  "supporting_source_ids": [
    "https://example.invalid/lot/1",
    "https://example.invalid/lot/2"
  ]
}
```

The example host is illustrative only and would be rejected at runtime unless
registered by the provider source policy.

### `similar_lots`

- item kind: `similar_lot`;
- requires `similarity_score`, at least one `matched_attribute`, and explicit
  `material_differences` (which may be empty);
- the existing specialist scorer owns similarity analysis;
- deterministic order is descending score then canonical source id;
- weak candidates are omitted rather than used to fill the result limit.

## 5. URL validation

Before an item is accepted:

1. parse the provider-returned URL without inventing or materially rewriting it;
2. require `https`;
3. reject user-info/embedded credentials;
4. reject local, loopback, link-local, private, and metadata destinations;
5. require the host to match the capability's registered source boundary;
6. revalidate every redirect at the existing outbound boundary;
7. compute canonical identity for deduplication while preserving the accepted
   source URL for display.

A result without a valid URL cannot be `verified` and is omitted from the
evidence list. Never replace it with a provider home page or guessed lot URL.

## 6. Provider failures

Provider exceptions never cross the contract. Map them to:

| Internal condition | Attempt status | Safe warning code |
|---|---|---|
| valid response with items | `success` | null |
| valid response, no items | `no_match` | null |
| deadline/read timeout | `timeout` | `provider_timeout` |
| provider not configured/eligible | `unavailable` | `provider_unavailable` |
| transport/HTTP failure | `failure` | `provider_failure` |
| invalid JSON/schema/evidence | `malformed` | `provider_malformed` |

Use the deterministic aggregate algorithm in `../data-model.md`. Do not include
status text, stack traces, response bodies, credentials, or request URLs in
warnings.

## 7. Cancellation and replay

- Check cancellation before starting a specialist adapter.
- Pass the remaining execution deadline to every provider/model operation.
- Check cancellation after every await and before result normalization/frame
  emission.
- Once Go cancellation wins, late results and frames are discarded.
- A completed call id and normalized result are stored in the existing
  checkpoint. On resume, the dispatcher uses that result and does not repeat
  the provider operation.

## 8. Prompt-injection-as-data

Provider titles, descriptions, snippets, and page text are untrusted. They:

- cannot alter tool choice or the allowlist;
- cannot request another capability;
- cannot change owner/run/execution ids;
- cannot reveal prompts or credentials;
- cannot override cancellation or bounds;
- cannot become a factual field without source-backed normalization.

Only the normalized envelope is returned to the Coin Copilot model, prefixed by
the existing `UNTRUSTED TOOL DATA` boundary.

## 9. Internal frame

Feature 359's `tool_completed` frame shape is retained:

```json
{
  "schema_version": 1,
  "run_id": "ccr_example",
  "execution_id": "cce_example",
  "frame_id": "frm_0003",
  "type": "tool_completed",
  "payload": {
    "tool_call_id": "call_01",
    "tool_name": "market_search",
    "step_id": "step-1",
    "status": "succeeded",
    "duration_ms": 823,
    "result_summary": "Dealer search returned 3 verified listings with partial provider coverage.",
    "result": {}
  }
}
```

`result` must validate as the capability's envelope. `status=succeeded` means
the tool returned a valid envelope; the domain outcome may still be
`partial`, `no_match`, or `unavailable`. Contract/schema failure uses
`status=rejected` or the existing `invalid_tool_call` terminal failure, as
appropriate.

## 10. Contract fixtures

Go and Python consume the same committed JSON fixtures for:

- one valid result per capability;
- complete, partial, no-match, and unavailable outcomes;
- unsafe URL, embedded credentials, invalid source host;
- missing required provenance;
- duplicate source identities and conflicts;
- mixed currency/price basis and insufficient trend samples;
- malformed provider data and unknown fields;
- prompt injection/token-shaped content;
- deterministic truncation;
- resume with a completed specialist result.
