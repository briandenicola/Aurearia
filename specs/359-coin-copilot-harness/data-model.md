# Data Model: Coin Copilot Read-Only Harness (#721)

**Storage**: SQLite via GORM additive `AutoMigrate`
**Owner**: Go API
**Python persistence**: none

## 1. Relationship overview

```text
User 1───* CoinCopilotThread 1───* CoinCopilotRun
                                  1───* CoinCopilotCheckpoint
                                  1───* CoinCopilotEvent
                                  1───* CoinCopilotResumeRequest
```

All repository reads and mutations include `user_id`. Foreign and unknown ids
are indistinguishable at the HTTP boundary.

## 2. `CoinCopilotThread`

| Field | Type | Notes |
|---|---|---|
| `ID` | string | `cct_` + random identifier; primary key |
| `UserID` | uint | required; indexed owner |
| `Title` | varchar(200) | generated from first user message; owner-editable later only if separately specified |
| `LastRunID` | *string | optional convenience pointer |
| `CreatedAt` | time.Time | |
| `UpdatedAt` | time.Time | |

Indexes:

- `(user_id, updated_at DESC)` for owner thread lists.

Deletion is explicit owner action. A thread with an active run cannot be
deleted until cancellation settles; successful deletion cascades to runs,
checkpoints, resume requests, and events in one transaction.

## 3. `CoinCopilotRun`

| Field | Type | Notes |
|---|---|---|
| `ID` | string | `ccr_` + random identifier; primary key |
| `ThreadID` | string | required |
| `UserID` | uint | required; denormalized for scoped queries |
| `Status` | enum | `queued`, `running`, `paused`, `cancel_requested`, `completed`, `failed`, `cancelled` |
| `Goal` | text | sanitized owner prompt, max 4,000 chars |
| `AppContextJSON` | text | bounded route/active coin context |
| `StartIdempotencyKeyHash` | char(64) | never store plaintext key |
| `StartRequestFingerprint` | char(64) | normalized body digest |
| `ExecutionID` | string | current execution attempt id |
| `ExecutionAttempt` | int | starts at 0; incremented on claim/resume |
| `CheckpointVersion` | int64 | latest committed checkpoint version |
| `LastSeq` | int64 | latest persisted public event sequence |
| `CancelRequestedAt` | *time.Time | |
| `PausedAt` | *time.Time | |
| `HeartbeatAt` | *time.Time | worker liveness |
| `WorkerID` | varchar(64) | current Go worker |
| `FailureCode` | varchar(48) | typed, client-safe |
| `FailureMessage` | varchar(300) | generic |
| `FinalAnswer` | text | sanitized markdown; no chain-of-thought |
| `IterationCount` | int | cumulative across executions |
| `ToolCallCount` | int | cumulative across executions |
| `InputTokens` | int64 | when provider reports |
| `OutputTokens` | int64 | when provider reports |
| `MaxIterations` | int | snapshotted setting |
| `MaxToolCalls` | int | snapshotted setting |
| `MaxConcurrentTools` | int | always 1 in MVP |
| `HardTimeoutSeconds` | int | per execution |
| `MaxPersistedToolResultBytes` | int | per call |
| `ResumeDeadline` | *time.Time | set on pause |
| `EventsPrunedAt` | *time.Time | |
| `CheckpointsPrunedAt` | *time.Time | |
| `StartedAt`, `CompletedAt` | *time.Time | |
| `CreatedAt`, `UpdatedAt` | time.Time | |

Indexes:

- `(user_id, status, created_at DESC)` for active limits/history;
- `(thread_id, created_at ASC)` for thread rendering;
- unique `(user_id, start_idempotency_key_hash)` with 24-hour reuse enforced
  by service/repository cleanup;
- `(status, heartbeat_at)` for stale recovery.

### 3.1 Budget snapshot policy

`HardTimeoutSeconds` defaults to 120 and is valid only from 15 through 150
seconds. The upper bound leaves a 30-second credential buffer within the
execution token's absolute 180-second TTL. The run snapshot also enforces
iteration, tool-call, sequential-concurrency, and persisted-payload limits.
Dollar-cost enforcement is deferred until a trustworthy provider/model pricing
source exists; reliable provider-reported input/output token counts remain
stored and observable without deriving a monetary estimate.

### 3.2 Run state machine

```text
[queued] ──claim──> [running] ──clarify──> [paused] ──resume──> [queued]
    │                   │   │                   │
    │                   │   ├──cancel request──> [cancel_requested]
    │                   │   ├───────────────────> [completed]
    │                   │   └───────────────────> [failed]
    │                   │
    └──cancel───────────> [cancelled]

[cancel_requested] ──worker observes──> [cancelled]
[paused] ──cancel─────────────────────> [cancelled]
[paused] ──resume deadline────────────> [failed:resume_window_expired]
```

Rules:

- Terminal states are `completed`, `failed`, and `cancelled`.
- Terminal states never transition.
- `running→completed|failed|paused|cancel_requested` and
  `cancel_requested→cancelled` use conditional updates with status,
  execution id, and checkpoint version predicates.
- The winning terminal update and terminal event append occur in one
  transaction, producing exactly one terminal event.
- If completion wins before cancellation, cancel returns `409` with the
  settled run. If cancellation wins, later Python frames are rejected.
- Stale recovery moves a `running` execution with heartbeat older than 45
  seconds to `paused` when a committed checkpoint exists, otherwise `failed`
  with `execution_lost`. Recovery emits the corresponding public events.

## 4. `CoinCopilotCheckpoint`

Immutable snapshots; a resume always uses the highest committed version.

| Field | Type | Notes |
|---|---|---|
| `ID` | uint | primary key |
| `RunID`, `ThreadID`, `UserID` | identifiers | required/indexed |
| `ExecutionID` | string | attempt that produced it |
| `Version` | int64 | monotonic per run, starts at 1 |
| `StateJSON` | text | strict schema below |
| `StateDigest` | char(64) | SHA-256 of canonical JSON |
| `CreatedAt` | time.Time | |

Unique index: `(run_id, version)`.

### 4.1 Checkpoint state schema

```json
{
  "schemaVersion": 1,
  "messages": [
    {"role": "user", "content": "Compare my Flavian bronzes"},
    {"role": "assistant", "content": "I need one clarification."}
  ],
  "plan": [
    {"id": "step-1", "title": "Summarize Flavian bronzes", "status": "completed"},
    {"id": "step-2", "title": "Compare collection composition", "status": "pending"}
  ],
  "completedTools": [
    {
      "toolCallId": "call_01",
      "toolName": "search_my_collection",
      "resultDigest": "sha256...",
      "result": {},
      "originalBytes": 1840,
      "persistedBytes": 1840,
      "truncated": false
    }
  ],
  "pendingClarification": {
    "question": "Should wishlist and sold coins be excluded?",
    "inputType": "single_choice",
    "choices": ["Owned collection only", "Include sold history"]
  },
  "nextAction": "await_clarification",
  "counters": {
    "iterations": 3,
    "toolCalls": 4,
    "inputTokens": 2100,
    "outputTokens": 680
  }
}
```

Constraints:

- Roles are `user` or `assistant`; no system, tool, or hidden reasoning message
  is persisted as conversation history.
- Plan status is `pending`, `in_progress`, `completed`, `skipped`, or `failed`.
- Tool result is validated against its registered output schema before
  persistence, sanitized, canonicalized, and capped at the run's byte limit.
- `nextAction` is `continue`, `await_clarification`, or `finish`.
- Maximums are those in `research.md` R6.

## 5. `CoinCopilotEvent`

| Field | Type | Notes |
|---|---|---|
| `ID` | uint | primary key |
| `RunID`, `ThreadID`, `UserID` | identifiers | required |
| `ExecutionID` | string | correlation only |
| `Seq` | int64 | monotonic per run |
| `Type` | varchar(40) | one of ten public types |
| `PayloadJSON` | text | typed, sanitized, max 64 KiB |
| `CreatedAt` | time.Time | |

Unique index: `(run_id, seq)`.

Sequence assignment increments `CoinCopilotRun.LastSeq` and inserts the event
in the same transaction as any associated status/checkpoint update. Events are
append-only until retention pruning.

## 6. `CoinCopilotResumeRequest`

| Field | Type | Notes |
|---|---|---|
| `ID` | uint | primary key |
| `RunID`, `UserID` | identifiers | required |
| `IdempotencyKeyHash` | char(64) | |
| `RequestFingerprint` | char(64) | answer + expected version digest |
| `AcceptedCheckpointVersion` | int64 | resulting version |
| `ExecutionID` | string | resulting execution |
| `CreatedAt` | time.Time | |

Unique index: `(run_id, idempotency_key_hash)`.

This small record makes resume replay deterministic without overloading events
or checkpoints.

## 7. Public event payloads

Detailed wire examples are in `contracts/sse-events.md`.

| Type | Required payload fields |
|---|---|
| `run_started` | `status`, `executionId`, `attempt`, `limits` |
| `plan_updated` | `plan[]` |
| `tool_started` | `toolCallId`, `toolName`, `stepId` |
| `tool_completed` | `toolCallId`, `toolName`, `stepId`, `status`, `durationMs`, `resultSummary`, `truncated` |
| `clarification_required` | `question`, `inputType`, `choices`, `checkpointVersion` |
| `run_paused` | `reason`, `checkpointVersion`, `resumeDeadline` |
| `run_resumed` | `executionId`, `attempt`, `checkpointVersion` |
| `run_cancelled` | `reason` |
| `run_completed` | `answer`, `usage` |
| `run_failed` | `code`, `message`, `retryable`, `usage` |

## 8. Retention and deletion

- Events: prune 7 days after terminal and stamp `EventsPrunedAt`.
- Checkpoints/resume requests/tool results: prune 30 days after terminal and
  stamp `CheckpointsPrunedAt`; preserve the run's final answer and usage.
- Paused runs: settle as failed after the 7-day resume deadline.
- Threads/runs: retain until owner deletion.
- Idempotency lookup rows: eligible for cleanup after 24 hours, but the run
  itself remains.

## 9. Migration and compatibility

- Add the five models to `database.AutoMigrate`; create the unique indexes in
  an idempotent follow-up migration.
- No existing table or column is altered.
- Existing `AgentConversation` rows and legacy chat endpoints are unchanged.
- Rollback is operational: set `CoinCopilotEnabled=false`; retained tables are
  inert and existing runs remain readable/cancellable.
