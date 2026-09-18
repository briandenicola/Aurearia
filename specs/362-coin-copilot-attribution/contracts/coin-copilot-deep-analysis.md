# Contract: Coin Copilot ↔ Deep Analysis Handoff

## 1. Boundary and route

One authenticated internal callback is added:

```http
POST /api/internal/copilot/tools/deep_analysis_handoff
Authorization: Bearer <Coin Copilot execution token>
Content-Type: application/json
```

The route is registered explicitly in `src/api/routes_internal.go` with
`CoinCopilotExecutionTokenRequired(..., "deep_analysis_handoff")`. It does not
introduce a wildcard route, a caller-supplied URL, or access to
`/api/internal/tools/*`. The request body is capped at 64 KiB, decoded with
unknown fields forbidden, and authorized against the current run/execution,
tool-call id, allowlist, concurrency, timeout, and tool budget.

## 2. Request

```json
{
  "tool_call_id": "call_01",
  "operation": "request",
  "target": { "type": "coin", "id": 42 }
}
```

Strict schema:

```text
operation = "request" | "status" | "rerun"
target = { type: "coin" | "draft", id: uint>0 }
job_id = optional uint>0
tool_call_id = non-empty bounded string
```

Mutual rules:

- `request`: `target` required; `job_id` forbidden.
- `status`: `job_id` required; `target` forbidden.
- `rerun`: `target` and `job_id` required; the previous job must be bound to
  the same resolved target.
- Provider overrides, notes, image paths, URLs, apply targets, accepted fields,
  and arbitrary options are forbidden.

## 3. Result

```json
{
  "schema_version": 1,
  "operation": "request",
  "outcome": "reused_result",
  "target": {
    "type": "coin",
    "id": 42,
    "display_label": "Maximinus I denarius"
  },
  "job": {
    "id": 314,
    "status": "completed",
    "reused": true,
    "created_at": "2026-09-18T18:00:00Z",
    "completed_at": "2026-09-18T18:02:00Z"
  },
  "input_digest": "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
  "review_url": "/deep-analysis/314",
  "fresh_analysis_available": true,
  "result": {
    "state": "complete",
    "narrative": "Persisted Deep Analysis narrative.",
    "partial_success": false,
    "image_only": false,
    "fields": [
      {
        "name": "ruler",
        "value": "Maximinus I",
        "confidence": 0.91,
        "evidence": [
          {
            "source": "numista",
            "excerpt": "Bounded persisted excerpt",
            "url": "https://en.numista.com/catalogue/pieces123.html"
          }
        ]
      }
    ],
    "disagreements": [],
    "unresolved_questions": [],
    "coverage": [
      { "provider": "numista", "status": "contributed" },
      { "provider": "ngc", "status": "not_automated", "link_out": "https://www.ngccoin.com/" },
      { "provider": "rpc", "status": "unavailable" }
    ],
    "attributions": [],
    "limitations": []
  },
  "limitations": []
}
```

### Outcome vocabulary

| Outcome | Meaning | Job created? |
|---|---|---|
| `accepted` | New Deep job admitted through existing queue | yes |
| `reused_active` | Equivalent queued/running job returned | no |
| `reused_result` | Equivalent retained completed/partial result returned | no |
| `status` | Existing owner-scoped job state/result returned | no |
| `retry_available` | Prior job is failed/cancelled/stale/missing-result or inputs changed | no |
| `missing_images` | Required usable roles absent/not distinct | no |
| `inactive_target` | Draft no longer active or target transitioned | no |
| `unavailable` | Deep Analysis disabled/unavailable/capacity/queue condition | no |
| `cancelled` | Coin Copilot cancellation won before admission | no |
| `not_found` | Unknown or foreign target/job; identical response | no |

### Deep job status vocabulary

Exactly `queued`, `running`, `completed`, `partial`, `failed`, `cancelled`.
Unknown states fail the tool closed as `invalid_tool_call`; they are never
coerced.

### Result-state vocabulary

Exactly `not_ready`, `complete`, `partial`, `no_match`, `failed`, `cancelled`,
`stale`, `missing_result`. Confidence is numeric and inclusive `[0.0,1.0]`.

### Provider vocabulary

- Providers: `numista`, `nomisma`, `ngc`, `ocre`, `rpc`.
- Coverage: `pending`, `running`, `contributed`, `no_match`, `failed`,
  `timed_out`, `skipped`, `not_automated`, `unavailable`.
- OCRE appears only when enabled and retains its ODbL/ANS attribution.
- NGC remains official quick evidence/link-out; no automated catalog search.
- RPC remains unavailable.

## 4. HTTP/error mapping

Expected domain outcomes above return HTTP 200 so Coin Copilot can explain them.
Malformed/tampered requests never become model-visible internal details:

| HTTP | Code/condition | Behavior |
|---|---|---|
| 400 | malformed JSON, unknown field/value, invalid mutual fields | `invalid_tool_call` |
| 401 | invalid/expired/revoked execution token | generic unauthorized |
| 404 | route unavailable | `agent_unavailable`; no fallback route |
| 409 | duplicate tool-call id, stale/current-execution conflict | `invalid_tool_call` or replayed checkpoint fact |
| 413 | request/result bound exceeded | `invalid_tool_call` |
| 503 | internal capability unavailable | `agent_unavailable`, retryable |

Deep queue/capacity/disabled conditions are typed `outcome:"unavailable"` with
one of these bounded reasons: `deep_disabled`, `job_at_capacity`, `queue_full`,
`temporarily_unavailable`. Raw errors are logged server-side only.

## 5. Replay, retry, reconnect, and freshness

- The existing Coin Copilot start/resume idempotency keys remain authoritative.
- A completed tool call is restored from
  `checkpoint.completed_tools`; Python must not call it again.
- A repeated new tool call with the same target is safe because Deep active
  fingerprint uniqueness returns `reused_active`.
- `reused_result` requires matching current v2 input digest and retained valid
  result. It offers `fresh_analysis_available:true`; only `operation:"rerun"`
  spends another run.
- `status` never starts or retries work.
- The chat SSE reconnect continues with
  `/api/agent/copilot/runs/{runId}/events?since={seq}`. The review link opens
  `/deep-analysis/{jobId}`, whose existing stream independently reconnects with
  its Deep event sequence.
- Pruned Deep events do not invalidate a retained terminal report; status uses
  the job snapshot and states that detailed activity history is unavailable.
- Changed target input yields `retry_available`/`inputs_changed`, never an old
  result represented as current.

## 6. Cancellation ordering

Go linearizes Copilot cancellation and `request`/`rerun` admission per
execution:

1. cancellation first: `cancelled`, no Deep job;
2. admission first: accepted/reused job id is durable; later Copilot
   cancellation prevents late Copilot frames but does not silently cancel the
   independent Deep job;
3. Deep cancellation remains available only through the existing Deep
   lifecycle endpoint/UI.

## 7. Browser/UI contract

- Public Coin Copilot `tool_completed.result` may include this projection after
  Go revalidation/sanitization.
- Vue renders a compact status/result handoff card in the existing Agent drawer.
- `review_url` must match `^/deep-analysis/[1-9][0-9]*$`; Vue constructs or
  validates the route from the numeric job id and never follows an absolute,
  protocol-relative, credential-bearing, or mismatched URL.
- The link label is `Open Deep Analysis`; it uses the existing router.
- No conversational accept/apply control is rendered.
- `/deep-analysis/:jobId` remains the only report/proposal editor and retains
  existing responsive layout, 44 px touch targets, design tokens, dark theme,
  PWA behavior, stream reconnect, retry, and cancellation controls.

## 8. Tamper requirements

Contract tests must reject:

- unknown request/result properties, operations, outcomes, states, providers,
  coverage values, and target types;
- zero/negative/foreign target or job ids;
- target/job mismatch on rerun;
- non-finite or out-of-range confidence;
- duplicate fields/evidence/tool-call ids;
- oversized arrays, strings, events, or checkpoint results;
- unsafe/malformed/unapproved citation and review URLs;
- prompt-injection text being treated as instructions;
- secrets or raw provider/internal errors in output;
- forged acceptance/apply fields or provider overrides;
- cancellation races and late frames.

## 9. Explicitly absent operations

There is no `apply`, `accept`, `edit`, `cancel_deep_job`, provider query,
arbitrary fetch, database, filesystem, shell, or generic HTTP operation.
