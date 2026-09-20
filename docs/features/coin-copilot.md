# Coin Copilot

> A default-off, durable, read-only multi-step harness for owner-scoped
> collection analysis in the existing chat drawer.

## Scope and rollout

Coin Copilot can sequence collection search, coin detail, collection summary,
top-value, portfolio-review, structural gap-analysis, dealer search, auction
search, completed-sale price-trend, and an owner-scoped Deep Analysis handoff.
It does not expose writes, approvals, similar lots, long-term memory, arbitrary
HTTP, filesystem, shell, code execution, or direct database access.

The existing drawer calls `GET /api/agent/copilot/capability` before starting a
run. Copilot mode is selected only when `CoinCopilotEnabled` is on (it defaults
off) and the configured model passes a fail-closed tool-calling preflight.
Anthropic must bind the fixed tool schemas; Ollama must explicitly advertise
the `tools` capability through `/api/show`. Disabled, unconfigured,
unsupported, ambiguous, or unavailable states use the unchanged legacy
`POST /api/agent/chat` flow.

## Deep Analysis handoff

When `CoinCopilotAttributionEnabled` and `DeepIdentificationEnabled` are also
on, Copilot may request, inspect, reuse, or explicitly rerun the existing Deep
Analysis workflow for one exact owned coin or active Quick Capture draft.
Route context (`activeCoinId` or `activeDraftId`) is bounded but
non-authoritative; ambiguous or conflicting context requires clarification.

Go resolves ownership and images, snapshots the target, admits the durable job,
and returns a typed bounded result through the fixed internal callback.
Python cannot call providers directly for this handoff and cannot create,
edit, or apply a proposal. `request` and `rerun` execute without concurrent
sibling tools; `status` is read-only. Replayed completed checkpoints reuse
persisted facts without another callback or provider run.

The chat card shows lifecycle state, persisted narrative, disagreements,
provider coverage, limitations, and deterministic omission counts. Its only
action is **Open Deep Analysis**, validated as the exact relative
`/deep-analysis/{jobId}` route. `DeepAnalysisPage` remains the sole
progress/retry/cancel/review/editor/apply surface.

The handoff request/public-event envelope is capped at 64 KiB and the persisted
result at 32 KiB. Unknown, foreign, unbound, or invalid targets return the same
non-disclosing `not_eligible` outcome. A target that disappears after a valid
durable owner binding returns `target_unavailable` without target metadata.

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

Defaults are 8 reasoning iterations, 12 total tool calls, at most 3 concurrent
read-only tools, 120 seconds per execution, one active run per owner, and
32 KiB persisted tool results. Concurrency snapshots from 1 through 5 are
accepted. Execution timeout is valid from 15 through 150 seconds. The maximum
plus the 30-second credential buffer stays within the token's absolute
180-second TTL.

Dollar-cost enforcement is deferred until Aurearia has a trustworthy, current
provider/model pricing source. Reliable provider-reported input/output token
usage remains observable. Iteration, tool-call, wall-clock, bounded
concurrency, and payload limits remain enforced. Concurrent results are
validated before dispatch, executed in bounded groups, and returned to the
model in its original call order.

Example prompts:

- `Find current dealer listings and upcoming auctions online for Julius Caesar denarii.`
- `Find current dealer listings for Byzantine gold solidi under $1,000.`
- `What is the completed-sale price trend for Athenian owl tetradrachms?`
- `Which coins in my collection are missing an era?`

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
drawer to legacy chat. Setting `CoinCopilotAttributionEnabled=false` or
`DeepIdentificationEnabled=false` blocks new handoffs and reruns. Existing
accepted jobs use finish-existing semantics: worker settlement, owner status,
events, cancellation, report/proposal review, edits, and confirmed apply remain
available. Existing threads/runs remain readable and cancellable; additive
tables and nullable columns remain in place.

Rollback must target a binary containing the Feature 362 source-vocabulary
guard. The mixed-binary workflow boots that guard against a copied migrated
database, proves unknown `copilot_draft` rows and artifacts remain byte
preserved and unadopted, then re-upgrades and verifies restoration. See
[ADR 0017](../adr/0017-coin-copilot-deep-analysis-handoff.md) and the
[Feature 362 evidence](../../specs/362-coin-copilot-attribution/quickstart-evidence.md).

Operational verification is documented in
[Testing Strategy](../testing.md#10-coin-copilot-contract-and-operations-testing).
The manual workflow is
[Feature 359 quickstart](../../specs/359-coin-copilot-harness/quickstart.md),
and the public REST/SSE contract is in
[API Reference](../api-reference.md#coin-copilot-default-off-durable-read-only-api).
