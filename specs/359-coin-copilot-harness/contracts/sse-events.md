# Contract: Coin Copilot Public SSE Events (Vue ← Go)

**Endpoint**: `GET /api/agent/copilot/runs/{runId}/events`  
**Media type**: `text/event-stream`  
**Authority**: Go-persisted `CoinCopilotEvent` rows

## Frame format

```text
id: 14
event: tool_completed
data: {"seq":14,"threadId":"cct_a1","runId":"ccr_b2","executionId":"cce_c3","type":"tool_completed","ts":"2026-09-17T22:41:12Z","payload":{...}}

: ping
```

`id` is the monotonic per-run sequence allocated by Go. `event` equals
`data.type`. Keepalive comments consume no sequence. Clients ignore unknown
event types.

### Envelope

| Field | Type | Notes |
|---|---|---|
| `seq` | int64 | equals `id:` |
| `threadId` | string | owner-scoped thread |
| `runId` | string | owner-scoped run |
| `executionId` | string | current execution attempt |
| `type` | string | event type below |
| `ts` | RFC3339 timestamp | Go persistence time |
| `payload` | object | typed, sanitized public projection |

No envelope or payload may contain chain-of-thought, scratchpad content,
credentials, provider-native messages, or raw tool arguments/results.

## Event types and payloads

### `run_started`

```json
{
  "status": "running",
  "executionId": "cce_c3",
  "attempt": 1,
  "limits": {
    "maxIterations": 8,
    "maxToolCalls": 12,
    "maxConcurrentTools": 1,
    "hardTimeoutSeconds": 120,
    "maxPersistedToolResultBytes": 32768
  }
}
```

### `plan_updated`

```json
{
  "plan": [
    {"id": "step-1", "title": "Summarize Flavian bronze holdings", "status": "completed"},
    {"id": "step-2", "title": "Compare category coverage", "status": "in_progress"}
  ]
}
```

Plan titles describe work, not why the model reasoned internally. Maximum 12
items, 200 characters per title.

### `tool_started`

```json
{
  "toolCallId": "call_02",
  "toolName": "collection_summary",
  "stepId": "step-2"
}
```

Tool arguments are intentionally omitted.

### `tool_completed`

```json
{
  "toolCallId": "call_02",
  "toolName": "collection_summary",
  "stepId": "step-2",
  "status": "succeeded",
  "durationMs": 38,
  "resultSummary": "Collection summary returned.",
  "truncated": false
}
```

`status` is `succeeded`, `failed`, `cancelled`, or `rejected`. The public event
does not contain the persisted bounded result.

### `clarification_required`

```json
{
  "question": "Should the comparison include sold coins?",
  "inputType": "single_choice",
  "choices": ["Owned collection only", "Include sold history"],
  "checkpointVersion": 4
}
```

`inputType` is `text`, `single_choice`, or `boolean`. `choices` is empty unless
the input type is `single_choice`.

### `run_paused`

```json
{
  "reason": "clarification_required",
  "checkpointVersion": 4,
  "resumeDeadline": "2026-09-24T22:41:12Z"
}
```

### `run_resumed`

```json
{
  "executionId": "cce_d4",
  "attempt": 2,
  "checkpointVersion": 5
}
```

### `run_cancelled`

```json
{"reason":"owner_cancelled"}
```

### `run_completed`

```json
{
  "answer": "Your Flavian bronzes are strongest in...",
  "usage": {
    "iterations": 5,
    "toolCalls": 6,
    "inputTokens": 4200,
    "outputTokens": 1100
  }
}
```

### `run_failed`

```json
{
  "code": "tool_limit_exceeded",
  "message": "Coin Copilot reached its tool-call limit before completing the request.",
  "retryable": true,
  "usage": {
    "iterations": 8,
    "toolCalls": 12,
    "inputTokens": 6100,
    "outputTokens": 1600
  }
}
```

Allowed failure codes:

- `agent_unavailable`
- `execution_lost`
- `invalid_agent_frame`
- `invalid_tool_call`
- `iteration_limit_exceeded`
- `tool_limit_exceeded`
- `time_limit_exceeded`
- `model_tool_calling_unsupported`
- `resume_window_expired`
- `internal`

## Replay and close semantics

| Client state | Request | Go behavior |
|---|---|---|
| First connect | no cursor | replay all retained events, then follow live |
| Reconnect | `?since=N` or `Last-Event-ID: N` | replay `(N,lastSeq]`, then follow live; query wins |
| Cursor predates retention | old cursor | emit unsequenced `stream_truncated`, then retained tail |
| Run terminal | any | replay remaining events, emit `event: end`, close |
| Thread/run missing or foreign | any | HTTP 404 before stream opens |

`stream_truncated` payload:

```json
{"runId":"ccr_b2","status":"completed","earliestSeq":22,"lastSeq":40}
```

After a terminal event:

```text
event: end
data: {"runId":"ccr_b2","status":"completed"}
```

## Persistence and privacy

- Go validates and persists an event before publishing it.
- The state transition/checkpoint update and associated event append share one
  transaction.
- Event payload maximum is 64 KiB.
- Event retention is 7 days after terminal state.
- Sanitization removes credential/token-shaped strings and rejects forbidden
  fields such as `reasoning`, `thought`, `scratchpad`, `rawPrompt`,
  `rawToolInput`, and `rawToolOutput`.
