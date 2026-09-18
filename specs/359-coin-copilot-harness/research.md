# Research: Coin Copilot Read-Only Harness (#721)

## R1. Scope — read-only composition first

- **Decision**: The MVP allowlist is
  `search_my_collection`, `get_coin`, `collection_summary`,
  `top_coins_by_value`, `portfolio_review`, and `gap_analysis`.
- **Rationale**: These capabilities answer broad questions about owned data
  without adding external-data provenance, write approval, or deep-analysis
  lifecycle complexity.
- **Deferred**: market search, auction search, price trends, similar lots,
  writes/approvals, deep-identification handoff, and long-term user memory.
- **Important correction**: the current `gap_analysis` team includes estimated
  prices and "where to look" acquisition advice. The harness wrapper must use a
  new read-only prompt/schema that reports structural gaps and prioritization
  only; it must not reuse that acquisition section unchanged.

## R2. Durable state owner — Go, not Python

- **Decision**: Go persists threads, runs, checkpoints, and events. Python
  receives one execution request and returns typed frames; it keeps no durable
  state and has no database access.
- **Rationale**: This directly satisfies Constitution Principle II. It also
  places ownership checks, retention, idempotency, cancellation races, and
  event sequencing beside the existing authentication and SQLite authority.
- **Rejected**:
  - Python checkpointer/database: violates the binding stateless-agent rule and
    would require a constitution amendment.
  - Browser-owned state: cannot survive disconnects/restarts and cannot be the
    authorization authority.
  - Reusing `AgentConversation.Messages` as a checkpoint blob: lacks typed
    transitions, event replay, idempotency, and safe concurrent updates.

## R3. Execution model — Go worker invokes stateless Python

- **Decision**: `POST /api/agent/copilot/runs` creates/queues a durable Go run.
  A Go worker claims it, mints an execution-scoped credential, and calls the
  Python internal endpoint `POST /api/copilot/execute`. Python streams typed
  internal frames back to Go. Go validates and persists each public event and
  checkpoint before publishing it to Vue.
- **Rationale**: This follows the proven Deep Identification pattern: Go owns
  durable state and replayable SSE; Python owns inference only.
- **Execution ordering**: one tool call at a time in MVP. Sequential execution
  simplifies cancellation, usage accounting, checkpoint replay, and SQLite
  contention without blocking later parallelism.

## R4. Public API and legacy fallback

- **Decision**: Add a new `/api/agent/copilot/*` resource API. The existing
  `/api/agent/chat` endpoint and `{text,status,done,error}` SSE stream remain
  byte-compatible.
- **Drawer behavior**:
  1. load `GET /api/agent/copilot/capability`;
  2. if `mode=copilot`, use durable run APIs;
  3. otherwise use the existing `agentChatStream`.
- **Fallback reasons**: `disabled`, `provider_unconfigured`,
  `model_tool_calling_unsupported`, or `temporarily_unavailable`.
- **Rejected**: silently changing `/agent/chat` into a persisted protocol. That
  would break existing clients and make rollback ambiguous.

## R5. Model capability detection

- **Decision**:
  - Anthropic is eligible when provider settings are valid and the selected
    chat model can be constructed and bound to the fixed tool schemas.
  - Ollama is eligible only when a bounded `/api/show` check for the selected
    model explicitly returns a `capabilities` collection containing `tools`.
  - Missing, malformed, timed-out, or ambiguous capability responses select
    legacy fallback.
- **Rationale**: LangChain exposing `bind_tools()` is not proof that an Ollama
  model implements tool calling. Fail-closed capability selection avoids
  accepting a run that cannot execute.
- **Cache**: cache only the non-sensitive capability result by
  `(provider, model, endpoint)` for 5 minutes; invalidate when AI settings
  change.

## R6. Limits and setting conventions

Use constants plus string defaults in `services/settings_service.go`, and a
validated `CoinCopilotSettings` snapshot following
`GetDeepIdentificationSettings`. Each invalid value falls back independently
and sets `Valid=false`.

| Setting | Default | Valid range / meaning |
|---|---:|---|
| `CoinCopilotEnabled` | `false` | Boolean feature flag |
| `CoinCopilotWorkerCount` | `1` | 1–4 global workers |
| `CoinCopilotMaxActivePerUser` | `1` | 1–3 non-terminal runs |
| `CoinCopilotQueueDepth` | `16` | 1–100 queued runs |
| `CoinCopilotMaxReasoningIterations` | `8` | 1–20 model planning steps |
| `CoinCopilotMaxToolCalls` | `12` | 1–40 calls |
| `CoinCopilotHardTimeoutSeconds` | `120` | 15–150 seconds per execution |
| `CoinCopilotMaxPersistedToolResultBytes` | `32768` | 4096–131072 bytes/call |
| `CoinCopilotEventRetentionHours` | `168` | 1–720 hours |
| `CoinCopilotCheckpointRetentionDays` | `30` | 1–365 days after terminal |
| `CoinCopilotResumeWindowHours` | `168` | 1–720 hours while paused |

Additional fixed MVP bounds:

- sequential tools (`maxConcurrentTools=1`);
- maximum prompt length 4,000 characters;
- maximum resume answer length 4,000 characters;
- maximum 50 public messages / 100,000 characters passed to one execution;
- maximum plan 12 items and 200 characters per public plan item;
- maximum clarification 500 characters and 10 choices;
- event payload maximum 64 KiB after sanitization.

Dollar-cost enforcement is deferred for the MVP. Provider token metadata is
reliable enough to record input/output token counts, but the repository has no
trustworthy, current provider/model pricing source from which to derive a
portable monetary estimate. The contracts therefore omit estimated-cost
fields and the admin surface omits a cost setting. Iteration, tool-call,
sequential-concurrency, wall-clock, token-usage recording, and payload bounds
remain enforced or recorded as applicable.

The 150-second execution maximum is a security boundary, not only an
operational default: adding the 30-second credential buffer reaches, but never
exceeds, the execution token's absolute 180-second TTL.

## R7. Retention

- **Decision**:
  - public events: 7 days after terminal;
  - checkpoints and bounded tool results: 30 days after terminal;
  - paused run resume window: 7 days from `paused_at`;
  - thread messages, run metadata, plan summary, final answer, and terminal
    usage totals: until owner deletes the thread.
- **Rationale**: Event/checkpoint detail has operational value but is more
  sensitive and voluminous than user-visible conversation history.
- **Deletion**: thread deletion cascades through runs/checkpoints/events only
  after active runs have settled cancellation. The janitor never deletes the
  thread's final answer.

## R8. Checkpoint contents — no hidden reasoning

- **Decision**: A checkpoint stores:
  - schema version and monotonic version;
  - public conversation messages needed for the next execution;
  - concise plan items (`id`, `title`, `status`);
  - completed tool facts and bounded sanitized result references;
  - counters/usage;
  - pending clarification; and
  - next action enum.
- **Never stored or emitted**: chain-of-thought, scratchpads, hidden messages,
  provider-native request/response dumps, secrets, internal credentials, or
  unrestricted raw tool payloads.
- **Rationale**: Resumption needs explicit continuation state, not private model
  reasoning.

## R9. Execution credentials

- **Decision**: Add a distinct HMAC token family
  `MintForCopilotExecution(userID, runID, executionID, allowedTools, ttl)`.
  Use a dedicated HKDF info label, not the user-token or deep-job token key.
- **TTL**: `min(remaining execution budget + 30 seconds, 180 seconds)`.
  A resume always receives a newly minted token and a new execution id.
- **Authorization**: middleware verifies signature, canonical encoding, expiry,
  non-revocation, owner/run/execution binding, and requested tool membership.
- **Revocation**: revoke at pause, cancellation, completion, failure, timeout,
  or stream loss. Revocation records may be kept in a bounded in-memory map
  because token TTL is at most 180 seconds; durable run state remains the
  authority and every tool call also verifies that execution is current and
  the run is `running`.
- **Rejected**: reusing the current 30-second user-scoped token, which can
  expire mid-run and is not bound to a run, execution, or allowlist.

## R10. Idempotency and state concurrency

- **Start**: require `Idempotency-Key` (1–128 printable ASCII characters).
  Persist its SHA-256 hash plus a normalized request fingerprint. Unique
  `(user_id, idempotency_key_hash)` for 24 hours. Same fingerprint returns the
  original run; mismatch returns `409 idempotency_conflict`.
- **Resume**: require a new idempotency key and
  `expectedCheckpointVersion`. Unique `(run_id, idempotency_key_hash)`.
  Atomically append the owner answer, increment checkpoint version, move
  `paused→queued`, and create a new execution id. Same payload returns the
  accepted response; different payload or stale version returns `409`.
- **Cancel**: idempotent without a key. `queued` settles immediately;
  `running` becomes `cancel_requested`; terminal/cancel-requested repeats
  return current state. Conditional updates decide cancel/complete races.

## R11. SSE — persisted public projection

- **Decision**: Follow the Deep Identification replay pattern:
  `id:` equals a Go-assigned per-run sequence, `event:` equals the typed public
  event name, and `data:` is a sanitized JSON envelope.
- Go persists before publishing. Python frame ids are correlation ids only and
  never become public sequence authority.
- Keepalive comments are unsequenced. `since` wins over `Last-Event-ID`.
- If requested history was pruned, send an unsequenced `stream_truncated`
  control frame before the retained tail. This control frame is not one of the
  ten application event types.

## R12. Observability and privacy

- **Metrics/logs allowed**: thread/run/execution ids, provider/model identifiers,
  status, error code, durations, counts, input/output token usage, tool name,
  result byte count, truncation flag, and payload digest.
- **Forbidden**: prompt text, conversation content, coin names/notes, query
  strings, tool arguments/results, provider credentials, JWTs/internal tokens, raw model
  messages, and chain-of-thought.
- **Error responses**: typed code plus generic user-facing message. Detailed
  causes remain in privacy-safe server logs.

## R13. Testing and tamper criteria

Required automated coverage:

1. state transition table and exactly-one-terminal-event race tests;
2. start/resume idempotency and stale checkpoint conflict tests;
3. owner isolation on every public repository/service/handler path;
4. credential signature, expiry, revocation, wrong owner/run/execution,
   unknown tool, and replay tests;
5. tool-call/result schema validation, size truncation, digest, and secret
   redaction tests;
6. no-write tests proving no mutation endpoint/tool is registered;
7. prompt-injection fixtures in tool results proving they remain data;
8. iteration/tool/time exhaustion tests plus token-usage and payload-bound tests;
9. stream reconnect, truncation, terminal close, and client de-duplication tests;
10. feature-off and unsupported-model fallback tests proving no Copilot rows;
11. Go↔Python contract drift fixtures for every internal frame;
12. Vue mobile/desktop drawer tests for progress, clarification, resume, cancel,
    reconnect, and fallback.
