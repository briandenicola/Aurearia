# Feature Specification: Coin Copilot Read-Only Harness

**Feature Branch**: `359-coin-copilot-harness`  
**Created**: 2026-09-17  
**Status**: Design Complete — Ready for Implementation  
**Input**: GitHub issue #721, narrowed by the product owner's locked MVP scope

## Scope Authority

This specification is the authoritative MVP scope for issue #721. The issue's
broader roadmap remains valid only as deferred work. This feature includes:

1. owner-scoped, read-only collection queries;
2. portfolio review using owner-scoped collection data; and
3. collection gap analysis using owner-scoped collection data.

It explicitly excludes market/dealer search, auction search, price trends,
similar lots, all writes and approvals, deep-identification handoff, and
long-term user memory. Go owns all durable thread, run, checkpoint, and event
state. Python remains stateless and has no database access. The entry point is
the existing app-wide chat drawer, behind an admin feature flag that defaults
off. The current supervisor remains the fallback.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Complete a multi-step collection question (Priority: P1)

As a collector, I want Coin Copilot to plan and execute multiple read-only
collection steps in one run so I can answer broader questions without manually
splitting them into separate prompts.

**Why this priority**: Multi-step read-only orchestration is the feature's core
value and can ship independently without adding any mutation surface.

**Independent Test**: Enable Coin Copilot, open the existing chat drawer, ask
"Compare my Flavian bronzes with the rest of my Roman collection and identify
the three clearest gaps," and verify that one run uses multiple typed
collection capabilities, displays progress, and returns an owner-scoped answer.

**Acceptance Scenarios**:

1. **Given** Coin Copilot is enabled and the configured model supports tool
   calling, **When** the owner submits a multi-step collection question,
   **Then** the system creates a durable thread/run, executes a bounded
   sequential plan, and returns a grounded final answer.
2. **Given** a request needs collection search, summary, portfolio review, and
   gap analysis, **When** the run executes, **Then** Python may sequence those
   capabilities within one run without invoking any deferred capability.
3. **Given** no owned records match a step, **When** the tool returns no results,
   **Then** the answer reports that result and does not invent holdings.
4. **Given** a prompt requests a write, purchase search, auction search, price
   trend, similar lots, deep identification, or durable memory, **When** the
   harness plans the run, **Then** it declines that portion and does not call an
   out-of-scope tool.

---

### User Story 2 - Follow durable progress and recover after disconnect (Priority: P1)

As a collector, I want run progress to survive a browser or API restart so I can
reopen the chat drawer and continue from authoritative state.

**Why this priority**: Durable state is required for trustworthy pause/resume
and distinguishes the harness from the existing transient single-turn stream.

**Independent Test**: Start a run, disconnect after at least one tool event,
reconnect using the last event sequence, and verify the Go API replays every
missed event in order before following live progress.

**Acceptance Scenarios**:

1. **Given** a run has persisted events, **When** the browser reconnects with
   `since=N`, **Then** Go replays events with sequence greater than `N` exactly
   once and in ascending order while retained.
2. **Given** the Python process or Go worker stops during an execution,
   **When** stale recovery runs, **Then** the run is not left indefinitely
   running and can be resumed only from the latest committed checkpoint.
3. **Given** a run is terminal, **When** its event stream is opened, **Then** Go
   replays retained events, emits an `end` control event, and closes.

---

### User Story 3 - Clarify, pause, resume, or cancel safely (Priority: P1)

As a collector, I want the harness to ask a concise clarification, pause
durably, resume from my answer, or cancel without repeating completed work.

**Why this priority**: User-controlled interruption is required for a bounded
agent harness and prevents ambiguous plans from proceeding on guesses.

**Independent Test**: Submit an ambiguous goal, receive a
`clarification_required` event followed by `run_paused`, resume with a valid
answer and idempotency key, and verify execution continues from the latest
checkpoint with a fresh internal credential.

**Acceptance Scenarios**:

1. **Given** material ambiguity, **When** the harness cannot safely choose
   among collection interpretations, **Then** it persists a checkpoint, emits
   `clarification_required`, and transitions the run to `paused`.
2. **Given** an owner-scoped paused run, **When** the owner resumes with the
   expected checkpoint version and a new answer, **Then** Go atomically records
   the answer, queues the same run, and starts a new execution attempt.
3. **Given** a repeated resume request with the same idempotency key and body,
   **When** it is replayed, **Then** the same accepted result is returned and no
   duplicate execution is created.
4. **Given** a queued or running run, **When** the owner cancels it, **Then** no
   new tool starts, in-flight output is discarded after cancellation wins, and
   exactly one terminal state is persisted.

---

### User Story 4 - Roll out safely with legacy fallback (Priority: P2)

As an administrator, I want Coin Copilot disabled by default and to fall back
to the existing supervisor when it is disabled, unavailable, or unsupported by
the selected model.

**Why this priority**: Safe rollout must preserve current chat behavior while
the new harness is evaluated.

**Independent Test**: Exercise the same chat drawer with the flag off, with an
unsupported Ollama model, and with the flag on plus a capable model; verify the
first two use the legacy `/api/agent/chat` flow and only the third starts a
durable Copilot run.

**Acceptance Scenarios**:

1. **Given** `CoinCopilotEnabled=false`, **When** chat opens or sends a prompt,
   **Then** the existing supervisor and transient SSE contract remain active.
2. **Given** the flag is on but the selected model lacks verified tool-calling
   capability, **When** chat is used, **Then** the system reports legacy mode
   and uses the existing supervisor without creating a Copilot run.
3. **Given** the flag is turned off while a run already exists, **When** the
   owner reads, streams, cancels, or resumes it, **Then** read and cancellation
   remain available but resume/new-start is rejected; existing state is not
   orphaned.

### Edge Cases

- A start request is retried after the client times out before receiving the
  response.
- A resume request is replayed, carries a stale checkpoint version, or races a
  cancellation.
- Cancellation races natural completion.
- The Python stream disconnects after Go persisted a tool completion but before
  the next checkpoint.
- A tool result exceeds the persistence limit or contains token-shaped text.
- A user attempts to read, stream, resume, or cancel another user's thread/run.
- An attacker tampers with a run-scoped credential, execution id, event
  sequence, checkpoint version, tool name, or tool-call id.
- The configured model advertises tool support but returns malformed tool calls.
- Event retention has elapsed while the final run/thread result remains
  available.
- A paused run exceeds its resume window.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST use the existing app-wide chat drawer as the only MVP
  user entry point; no separate Copilot page or mode switch is introduced.
- **FR-002**: System MUST gate new Coin Copilot starts behind
  `CoinCopilotEnabled`, an admin `AppSetting` with default `"false"`.
- **FR-003**: System MUST keep the current supervisor and
  `POST /api/agent/chat` behavior available as fallback when the feature is
  disabled, the provider is unconfigured, the model lacks tool capability, or
  harness startup fails before a run is accepted.
- **FR-004**: The MVP capability allowlist MUST contain only
  `search_my_collection`, `get_coin`, `collection_summary`,
  `top_coins_by_value`, `portfolio_review`, and `gap_analysis`.
- **FR-005**: All capabilities MUST be read-only. The harness MUST NOT expose
  `propose_update`, `commit_update`, arbitrary HTTP, web search, filesystem,
  shell, code execution, or direct database tools.
- **FR-006**: Market/dealer search, auction search, price trends, similar lots,
  writes/approvals, deep-identification handoff, and long-term user memory MUST
  remain out of scope and MUST be declined rather than silently delegated.
- **FR-007**: Go MUST own and persist every `CoinCopilotThread`,
  `CoinCopilotRun`, `CoinCopilotCheckpoint`, and `CoinCopilotEvent`; Python MUST
  remain stateless and MUST NOT access SQLite or any other database.
- **FR-008**: All public resources and operations MUST be scoped by the
  authenticated `userId` derived server-side. Unknown and foreign identifiers
  MUST both return `404`.
- **FR-009**: The public contract MUST provide typed operations for capability
  discovery, start, read, thread deletion, stream, cancel, and resume as defined in
  `contracts/coin-copilot.openapi.yaml`.
- **FR-010**: A start request MUST require an `Idempotency-Key`; the same owner,
  key, and normalized request fingerprint MUST return the original run, while
  reusing the key with a different fingerprint MUST return `409`.
- **FR-011**: A resume request MUST require an `Idempotency-Key` and
  `expectedCheckpointVersion`; replay of the same request MUST not create a
  second execution, and a stale version MUST return `409`.
- **FR-012**: Run states MUST be `queued`, `running`, `paused`,
  `cancel_requested`, `completed`, `failed`, or `cancelled`, with only the
  transitions defined in `data-model.md`.
- **FR-013**: Every state transition and event sequence allocation MUST be
  transactional. A run MUST have exactly one terminal state/event.
- **FR-014**: Cancellation MUST be cooperative and durable. Once cancellation
  wins, no subsequent tool result or model output may be committed to the run.
- **FR-015**: Resume MUST continue the same run from the latest committed
  checkpoint, create a new execution id/attempt, and never depend on Python
  process memory.
- **FR-016**: Go MUST mint a fresh, run-and-execution-scoped internal credential
  for every initial execution and resume. It MUST authorize only the MVP
  read-only tool allowlist, expire shortly after the execution budget, and be
  revoked when that execution settles.
- **FR-017**: Python MUST receive all required execution context per request:
  provider configuration, bounded conversation input, latest checkpoint,
  execution id, limits, callback base URL, and fresh credential.
- **FR-018**: Python MUST emit only typed internal execution frames. Go MUST
  validate, sanitize, translate, and persist public events before they are
  streamed to Vue.
- **FR-019**: Public events MUST include exactly these MVP types:
  `run_started`, `plan_updated`, `tool_started`, `tool_completed`,
  `clarification_required`, `run_paused`, `run_resumed`, `run_cancelled`,
  `run_completed`, and `run_failed`.
- **FR-020**: Public and internal contracts MUST NOT expose chain-of-thought,
  hidden reasoning, scratchpad messages, raw model traces, or provider-native
  tool messages. Plans contain concise task labels/status only.
- **FR-021**: Tool inputs and outputs MUST be schema validated, treated as
  untrusted data, and bound to the credential's owner/run/execution/tool-call
  identity.
- **FR-022**: Persisted tool results MUST be sanitized and capped at 32 KiB per
  call by default. Oversized results MUST be deterministically truncated with
  `truncated=true`, original byte count, and a digest.
- **FR-023**: Default run limits MUST be 8 reasoning iterations, 12 tool calls,
  one tool at a time, 120 seconds wall-clock, one active run per owner, and
  32 KiB persisted tool results per call. The execution timeout MUST accept
  values from 15 through 150 seconds only; the 150-second maximum plus the
  30-second credential buffer fits the execution token's absolute 180-second
  TTL. Dollar-cost enforcement is deferred because the MVP has no trustworthy
  provider/model pricing source. Reliable provider-reported input/output token
  usage remains observable. Iteration, tool-call, wall-clock,
  sequential-concurrency, and payload limits remain enforced. Settings
  validation ranges are defined in `research.md`; invalid settings fall back
  independently to documented defaults and mark the snapshot invalid.
- **FR-024**: Limits MUST be snapshotted onto the run at creation so an admin
  setting change does not alter an in-flight run.
- **FR-025**: The runner MUST check cancellation and all applicable budgets
  before each model call and tool call and after every awaited operation.
- **FR-026**: A clarification MUST include a concise question and a typed input
  shape; after persisting it, the run MUST transition to `paused` and stop the
  current execution.
- **FR-027**: Paused runs MUST be resumable for 7 days. After that window Go
  MUST settle them to `failed` with code `resume_window_expired`.
- **FR-028**: Public event replay MUST use a monotonic per-run sequence allocated
  in Go. `since` query parameter takes precedence over `Last-Event-ID`.
- **FR-029**: Within event retention, reconnecting clients MUST receive every
  missed event once, in order, before live events. Unknown event types MUST be
  ignored by clients.
- **FR-030**: Event payloads MUST be retained for 7 days after terminal state.
  Checkpoints and bounded tool results MUST be retained for 30 days after
  terminal state. Run summaries/final responses and thread messages MUST be
  retained until the owner deletes the thread.
- **FR-031**: Deleting a thread MUST transactionally delete its runs,
  checkpoints, and events. Active runs MUST be cancelled and settled before
  deletion succeeds.
- **FR-032**: Observability MUST record run/execution ids, state transitions,
  durations, iteration/tool counts, input/output token usage when reported,
  error codes, and tool names, but MUST NOT log prompts, conversation text,
  raw tool arguments/results, collection payloads, credentials, or model
  chain-of-thought. It MUST NOT expose a dollar-cost estimate until a
  trustworthy pricing source is specified.
- **FR-033**: Anthropic is harness-capable when provider configuration is valid
  and tool binding succeeds. Ollama is harness-capable only when its
  `/api/show` metadata explicitly includes the `tools` capability. A failed or
  ambiguous capability check MUST select legacy fallback, not optimistic use.
- **FR-034**: Malformed model tool calls, unknown tools, duplicate tool-call
  ids, and execution/run binding mismatches MUST fail closed with typed,
  client-safe errors.
- **FR-035**: Existing saved conversations remain readable and the legacy save
  flow remains unchanged. New Copilot threads MUST not overload the existing
  `AgentConversation.Messages` JSON blob as orchestration storage.
- **FR-036**: All public handlers MUST have Swagger annotations and all Python
  request/event schemas MUST use strict Pydantic models.

### Key Entities

- **CoinCopilotThread**: Owner-scoped durable conversation container and final
  user-visible message history.
- **CoinCopilotRun**: One bounded attempt to satisfy a user goal within a
  thread, including snapshotted limits, status, current checkpoint version,
  usage totals, and terminal result/error.
- **CoinCopilotCheckpoint**: Immutable execution continuation state committed
  by Go; contains only public messages, compact plan state, completed tool
  facts, and pending clarification—not hidden reasoning.
- **CoinCopilotEvent**: Append-only, owner-scoped, sequenced public event used
  for replayable SSE.
- **CoinCopilotExecutionCredential**: Ephemeral HMAC credential bound to owner,
  run, execution, expiry, and read-only capability set; never persisted in
  plaintext.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A supported-model run can complete a goal requiring at least
  three sequential MVP capability calls in one durable run.
- **SC-002**: 100% of tool reads in tests return only the authenticated owner's
  data; foreign run/thread ids reveal no existence.
- **SC-003**: Reconnect tests observe gap-free, duplicate-free event replay
  within the 7-day event retention window.
- **SC-004**: Cancellation, completion, and failure races produce exactly one
  terminal state and one terminal event.
- **SC-005**: Start and resume retries with identical idempotency inputs create
  zero duplicate runs/executions; mismatched replay attempts return `409`.
- **SC-006**: No accepted run exceeds its snapshotted iteration, tool-call,
  concurrency, duration, or persisted-result limit, and reliable input/output
  token counts remain available when the provider reports them.
- **SC-007**: Automated tamper tests reject foreign, expired, revoked,
  wrong-run, wrong-execution, unknown-tool, and modified-signature internal
  requests with no tool execution.
- **SC-008**: Public and persisted payload tests find no credentials, raw
  collection payloads beyond bounded tool results, provider-native traces, or
  chain-of-thought fields.
- **SC-009**: With the feature disabled or the model unsupported, the existing
  chat supervisor remains functional and no Copilot rows are created.
- **SC-010**: The full Constitution §17 quality gate passes for all touched Go,
  Python, Vue, contract, and documentation surfaces.

## Assumptions

- The existing `CollectionToolsService` remains the canonical owner-scoped
  read layer from feature #217.
- Portfolio review and gap analysis will be adjusted or wrapped for this
  feature so they consume only Go-provided data and produce no market-search
  or acquisition-listing behavior.
- The existing fetch-based SSE reader pattern can be reused, but Copilot uses a
  new persisted event contract rather than changing legacy chat events.
- This MVP targets personal-scale, single-node deployment and does not require
  distributed locks, an external queue, or a third-party checkpoint store.
