# Contract: Coin Copilot Internal Execution (Go ↔ Python)

This contract is private to the two application services. The browser never
calls it. The static `X-Internal-Service-Token` authenticates Go to Python;
the per-execution credential authorizes Python callbacks to Go.

## 1. Capability preflight

`POST /api/copilot/capability` on the Python service accepts only the
per-request `llm` configuration:

```json
{
  "llm": {
    "provider": "anthropic",
    "api_key": "[request-only secret]",
    "model": "claude-sonnet-5",
    "ollama_url": "",
    "searxng_url": ""
  }
}
```

It authenticates with the same static `X-Internal-Service-Token` as execution
requests and returns `{"supported":true}` only after Python constructs the
configured model and binds all six fixed MVP tool schemas. Ollama additionally
requires `/api/show` to explicitly advertise `tools`. Errors, timeouts, and
malformed or ambiguous responses fail closed. The request contains no prompt,
conversation, execution credential, or tool input/output payload.

Go MUST complete this preflight before reporting Copilot mode or persisting a
new run.

## 2. Execute endpoint

`POST /api/copilot/execute` on the Python service returns
`text/event-stream`.

### Request

```json
{
  "schema_version": 1,
  "thread_id": "cct_a1",
  "run_id": "ccr_b2",
  "execution_id": "cce_c3",
  "goal": "Compare my Flavian bronzes and identify gaps.",
  "messages": [{"role": "user", "content": "Compare my Flavian bronzes and identify gaps."}],
  "checkpoint": {
    "version": 0,
    "plan": [],
    "completed_tools": [],
    "pending_clarification": null,
    "next_action": "continue",
    "counters": {
      "iterations": 0,
      "tool_calls": 0,
      "input_tokens": 0,
      "output_tokens": 0
    }
  },
  "app_context": {"route": "/stats", "active_coin_id": null},
  "llm": {
    "provider": "anthropic",
    "api_key": "[request-only secret]",
    "model": "claude-sonnet-5",
    "ollama_url": "",
    "searxng_url": ""
  },
  "limits": {
    "max_iterations": 8,
    "max_tool_calls": 12,
    "max_concurrent_tools": 1,
    "hard_timeout_seconds": 120,
    "max_persisted_tool_result_bytes": 32768
  },
  "tools_base_url": "http://api:8080",
  "execution_token": "[ephemeral]",
  "allowed_tools": [
    "search_my_collection",
    "get_coin",
    "collection_summary",
    "top_coins_by_value",
    "portfolio_review",
    "gap_analysis"
  ]
}
```

Strict Pydantic models use `extra="forbid"`. Python validates ids, lengths,
enum values, limits, cumulative counters, and that `allowed_tools` is a subset
of the compiled MVP allowlist.

## 3. Internal execution frames

Each frame is:

```text
event: <type>
data: {"schema_version":1,"run_id":"ccr_b2","execution_id":"cce_c3","frame_id":"frm_01","type":"<type>","payload":{...}}
```

Allowed frame types:

- `plan_updated`
- `tool_started`
- `tool_completed`
- `checkpoint`
- `clarification_required`
- `completed`
- `failed`
- `usage`

Python does not emit public `run_started`, `run_paused`, `run_resumed`, or
`run_cancelled`; Go derives those from authoritative transitions.

### Frame rules

- `frame_id` is unique within the execution and supports duplicate-frame
  rejection; it is not the public SSE sequence.
- Every frame must match the current `run_id` and `execution_id`.
- `tool_started` includes `tool_call_id`, `tool_name`, and `step_id`, never raw
  arguments.
- `tool_completed` includes status, duration, a short public summary, and the
  validated result for checkpoint persistence. Go revalidates the result.
- `checkpoint` contains the complete continuation schema from
  `data-model.md`; it contains no hidden model messages or reasoning.
- `completed` includes only final answer and cumulative usage.
- `failed` includes a typed safe code/message and cumulative usage.
- A frame after Go has accepted a terminal/pause/cancel transition is rejected
  and not persisted.

## 4. Read-only callback tools

Base path: `/api/internal/copilot/tools`. Authentication:

```text
Authorization: Bearer <execution_token>
```

Endpoints:

| Endpoint | Typed request | Typed response |
|---|---|---|
| `POST /search_my_collection` | `{tool_call_id, query, limit?}` | `{coins[]}` |
| `POST /get_coin` | `{tool_call_id, coin_id}` | `{coin}` |
| `POST /collection_summary` | `{tool_call_id}` | `{summary}` |
| `POST /top_coins_by_value` | `{tool_call_id, limit?}` | `{coins[]}` |

`portfolio_review` and `gap_analysis` are Python capabilities over the
validated results of these four Go reads; they do not add callback endpoints
or external access.

Each callback validates:

1. HMAC signature and canonical encoding;
2. expiry and revocation;
3. owner, run, and execution against the current Go row;
4. run status is `running`;
5. tool is in token allowlist and global MVP allowlist;
6. `tool_call_id` has not completed previously;
7. the run has remaining tool-call and wall-clock budget.

The handler remains thin and calls `CollectionToolsService`; it cannot call a
repository or database directly.

## 5. Execution credential

Logical claims:

```json
{
  "version": 1,
  "user_id": 7,
  "run_id": "ccr_b2",
  "execution_id": "cce_c3",
  "allowed_tools": ["search_my_collection", "get_coin"],
  "expires_at": 1789684992,
  "nonce": "..."
}
```

The credential uses a dedicated HKDF-derived HMAC key and a canonical,
base64url encoding. TTL is at most 180 seconds and is freshly minted per
execution/resume. It is revoked as soon as that execution pauses or settles.
Credentials are never persisted or logged.

## 5. Cancellation

Go cancels the HTTP request context when cancellation is requested. Python
must check cancellation before and after every model/tool await and stop
emitting frames. Go remains authoritative: even if Python continues, callbacks
and frames fail their current-execution/status checks.

## 6. Contract drift

Commit shared JSON fixtures for every request, callback, and frame shape. Go
and Python tests deserialize the same fixtures and reject unknown fields,
unknown tools, forbidden reasoning fields, oversized payloads, and mismatched
run/execution ids.
