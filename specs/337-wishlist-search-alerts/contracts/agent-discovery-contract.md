# Agent Contract: Wishlist Alert Discovery

The Python agent remains stateless. The Go API sends all criteria, LLM/search configuration, limits, and source filters per request. The agent returns source-backed candidates only; it does not persist, scope users, suppress duplicates, create wishlist items, or decide lifecycle state.

Recommended endpoint: `POST /api/search/alerts` behind `AgentProxy`.

## Request

```json
{
  "llm": {
    "provider": "anthropic",
    "api_key": "",
    "model": "claude-sonnet",
    "ollama_url": "",
    "searxng_url": ""
  },
  "alert": {
    "alert_id": 1,
    "criteria_snapshot": {
      "name": "Domitian denarius under $300",
      "ruler_or_issuer": "Domitian",
      "coin_type": "Denarius",
      "date_from": 81,
      "date_to": 96,
      "mint": "Rome",
      "material": "Silver",
      "grade_or_condition": "VF or better",
      "price_min": 0,
      "price_max": 300,
      "currency": "USD",
      "dealer_preference": "VCoins or MA-Shops",
      "source_filters": ["vcoins.com", "ma-shops.com"],
      "keywords": "RIC Minerva",
      "notes": "Prefer clear legends"
    },
    "max_candidates": 20
  }
}
```

## Response

```json
{
  "candidates": [
    {
      "source_url": "https://www.vcoins.com/en/stores/example/123",
      "source_name": "VCoins Example Dealer",
      "title": "Domitian AR Denarius",
      "observed_price": 225.0,
      "observed_currency": "USD",
      "reason_for_match": "Title and description mention Domitian denarius; price is under alert maximum.",
      "last_seen_at": "2026-06-29T17:00:10Z",
      "provenance_status": "verified",
      "fields": {
        "ruler": "Domitian",
        "denomination": "Denarius",
        "material": "Silver",
        "mint": "Rome",
        "grade_or_condition": ""
      },
      "provenance": [
        {
          "field": "source_url",
          "value": "https://www.vcoins.com/en/stores/example/123",
          "source_url": "https://www.vcoins.com/en/stores/example/123",
          "observed_at": "2026-06-29T17:00:10Z",
          "confidence": "high",
          "verification_state": "verified",
          "notes": "URL came from fetched dealer listing"
        }
      ]
    }
  ],
  "warnings": [],
  "partial": false
}
```

## Validation rules

- Pydantic request/response models use `extra="forbid"` to detect drift.
- `source_url` is required for every candidate.
- `title`, `reason_for_match`, `last_seen_at`, and `provenance_status` are required.
- Missing optional facts use empty string or null and must have `partial` or `unverified` provenance where relevant.
- Agent must not invent source/dealer names, prices, titles, availability claims, or URLs.
- Agent must not return private collection data.
- `max_candidates` caps returned candidates.

## Safe outbound HTTP rules

- Use existing `safe_get()` / `validate_public_outbound_url()` for page fetches and redirects.
- Configured service URLs such as SearXNG/Ollama continue to use trusted-origin validation.
- Local/private/metadata addresses remain blocked unless explicitly trusted for local development.
- Timeouts and partial failures return warnings/partial status, not internal stack traces.

## Non-goals for v1

- No Python persistence.
- No user scoping decisions in Python.
- No custom per-dealer scraper expansion beyond existing search/fetch capabilities.
- No direct notification delivery from Python.
- No automatic wishlist creation.

## Go `AgentProxy` mapping

Add typed DTOs parallel to the existing availability proxy:

```go
type AlertDiscoveryProxyRequest struct {
    LLM   LLMConfig                   `json:"llm"`
    Alert AlertDiscoveryRequestDetail `json:"alert"`
}

type AlertDiscoveryProxyResponse struct {
    Candidates []AlertCandidateProxy `json:"candidates"`
    Warnings   []string              `json:"warnings"`
    Partial    bool                  `json:"partial"`
}
```

`WishlistSearchAlertService.RunNow` is the only Go caller. The service converts proxy candidates into persisted `AlertCandidate` rows, computes duplicate keys, and records run results.
