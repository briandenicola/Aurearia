# Contract: Specialist Projection on Coin Copilot SSE

## Compatibility rule

Feature 361 adds no event name and changes no REST route. The ten Feature 359
application events, sequence allocation, replay ordering, retention, terminal
close, and `stream_truncated` behavior remain unchanged.

For `market_search`, `auction_search`, `price_trends`, and `similar_lots`, the
existing `tool_completed` payload gains one optional field:
`specialistResult`. Existing fields remain required.

## Event example

```text
id: 14
event: tool_completed
data: {"seq":14,"threadId":"cct_a1","runId":"ccr_b2","executionId":"cce_c3","type":"tool_completed","ts":"2026-09-18T12:00:00Z","payload":{"toolCallId":"call_01","toolName":"market_search","stepId":"step-1","status":"succeeded","durationMs":823,"resultSummary":"Dealer search returned 2 verified listings.","truncated":false,"specialistResult":{"capability":"market_search","outcome":"complete","items":[{"kind":"dealer_listing","title":"Domitian denarius","sourceUrl":"https://dealer.example/item/1","observedAt":"2026-09-18T12:00:00Z","confidence":"high","verificationState":"verified","facts":["Available","USD 250"],"matchedAttributes":[]}],"trend":null,"warnings":[],"truncation":{"truncated":false,"originalBytes":1200,"persistedBytes":1200,"digest":"0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef","omittedItems":0}}}}
```

The example host is illustrative only. Runtime validation accepts only
registered provider/source hosts.

## `specialistResult`

| Field | Type | Rules |
|---|---|---|
| `capability` | enum | one of the four specialist names; equals `toolName` |
| `outcome` | enum | `complete`, `partial`, `no_match`, `unavailable` |
| `items` | public evidence item[] | maximum 10 |
| `trend` | public trend or null | only for `price_trends` |
| `warnings` | string[] | maximum 10, client-safe |
| `truncation` | object | existing byte/digest semantics |

### Public evidence item

| Field | Type | Rules |
|---|---|---|
| `kind` | enum | dealer listing, auction lot, sale observation, or similar lot |
| `title` | string | source-backed, max 300 |
| `sourceUrl` | string | validated HTTPS source URL |
| `observedAt` | RFC3339 UTC string | required |
| `confidence` | enum | `high`, `medium`, `low` |
| `verificationState` | enum | `verified`, `partial` |
| `facts` | string[] | maximum 10 concise, source-backed display facts |
| `matchedAttributes` | string[] | maximum 20; similar lots only |
| `materialDifferences` | string[] | maximum 20; similar lots only |

Go constructs `facts` from validated fields. Python/provider prose is never
copied directly into this public array.

## Client behavior

- The runtime parser rejects an invalid projection instead of partially
  trusting it.
- Unknown future optional fields are ignored only at the event envelope level;
  the known specialist projection is strict.
- Vue keys cards by `toolCallId + canonical source URL` and de-duplicates
  replay by existing event sequence.
- `partial`, `no_match`, and `unavailable` are displayed explicitly.
- Warnings and result truncation remain visible.
- Source links open safely in a new context and never expose a query,
  credential, or internal URL.
- No item exposes a write, watch, bid, approval, save, or deep-identification
  action.

## Privacy and bounds

The projection is derived only after Go validation and sanitization, is
persisted before publication, and remains within the existing 64 KiB event
limit. It contains no owner collection data unless that data is already part
of a separately authorized final answer, and it never contains raw provider
content, tool arguments, prompts, credentials, or hidden reasoning.
