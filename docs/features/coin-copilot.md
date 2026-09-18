# Coin Copilot

> A default-off, durable, read-only multi-step harness for owner-scoped
> collection analysis in the existing chat drawer.

## Scope and rollout

Coin Copilot can sequence collection search, coin detail, collection summary,
top-value, portfolio-review, and structural gap-analysis capabilities. It does
not expose writes, approvals, dealer/auction search, price trends, similar
lots, deep-identification handoff, long-term memory, arbitrary HTTP,
filesystem, shell, code execution, or direct database access.

The existing drawer calls `GET /api/agent/copilot/capability` before starting a
run. Copilot mode is selected only when `CoinCopilotEnabled` is on (it defaults
off) and the configured model passes a fail-closed tool-calling preflight.
Anthropic must bind the fixed tool schemas; Ollama must explicitly advertise
the `tools` capability through `/api/show`. Disabled, unconfigured,
unsupported, ambiguous, or unavailable states use the unchanged legacy
`POST /api/agent/chat` flow.

## State, privacy, and retention

Go owns collection access, durable threads/runs/checkpoints/events,
idempotency, cancellation, replay, and retention. Python remains stateless and
database-free. Each execution or resume receives a fresh read-only credential
bound to the owner, run, execution, allowed tools, and expiry.

Persisted checkpoints contain bounded public conversation messages, concise
plan status, bounded validated tool facts, counters, and any pending
clarification. Checkpoints, public events, and logs exclude chain-of-thought,
scratchpads, hidden/provider messages, credentials, raw provider prompts, raw
collection payloads, and raw tool arguments/results.

- Public events: 7 days after terminal state.
- Checkpoints and bounded tool results: 30 days after terminal state.
- Paused-run resume window: 7 days.
- Final answers, token usage totals, and thread messages: until owner deletion.

## Limits and observability

Defaults are 8 reasoning iterations, 12 tool calls, one tool at a time, 120
seconds per execution, one active run per owner, and 32 KiB persisted tool
results. Execution timeout is valid from 15 through 150 seconds. The maximum
plus the 30-second credential buffer stays within the token's absolute
180-second TTL.

Dollar-cost enforcement is deferred until Aurearia has a trustworthy, current
provider/model pricing source. Reliable provider-reported input/output token
usage remains observable. Iteration, tool-call, wall-clock,
sequential-concurrency, and payload limits remain enforced.

## Disconnect, cancellation, and resume

Go persists each public event before publishing it. A reconnect supplies
`since` or `Last-Event-ID`; Go replays retained events in sequence before
following live progress. Clarification commits a checkpoint and pauses the
run. Resume requires the expected checkpoint version and an idempotency key,
continues the same run with a new execution id, and never depends on Python
memory. Cancellation is cooperative in Python and authoritative in Go, so
frames or tool results arriving after cancellation wins are discarded.

## Rollback and operations

Set `CoinCopilotEnabled=false` to stop new starts and resumes and return the
drawer to legacy chat. Existing threads/runs remain readable and cancellable;
additive tables may remain in place. Stale-run recovery and retention cleanup
must continue until outstanding state settles.

Operational verification is documented in
[Testing Strategy](../testing.md#10-coin-copilot-contract-and-operations-testing).
The manual workflow is
[Feature 359 quickstart](../../specs/359-coin-copilot-harness/quickstart.md),
and the public REST/SSE contract is in
[API Reference](../api-reference.md#coin-copilot-default-off-durable-read-only-api).
