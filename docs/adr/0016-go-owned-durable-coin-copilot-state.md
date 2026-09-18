# ADR 0016: Go-Owned Durable Coin Copilot State

Date: 2026-09-17
Status: Proposed

## Context

Issue #721 proposes a domain-specific Coin Copilot that can plan and execute
multiple numismatic capabilities, pause for clarification, resume safely, and
stream progress. The existing chat supervisor is transient: one request is
routed to one team and its SSE output is not an authoritative replay log.
Existing `AgentConversation` records are owner-saved JSON message blobs, not a
transactional orchestration state machine.

A resumable harness needs durable thread, run, checkpoint, idempotency,
cancellation, and event state. Constitution Principle II simultaneously
requires the Python agent to remain stateless and forbids direct database
access. The Go API already owns authentication, authorization, SQLite,
owner-scoped collection tools, SSE proxying, and the durable Deep
Identification job/event pattern.

The MVP is intentionally read-only: collection queries, portfolio review, and
gap analysis. Market and auction search, price trends, similar lots, writes and
approvals, deep-identification handoff, and long-term memory are deferred.

## Decision

Coin Copilot uses a Go-owned durable orchestration domain:

- `CoinCopilotThread` stores the owner-scoped conversation container.
- `CoinCopilotRun` stores the bounded state machine, snapshotted limits,
  idempotency fingerprint, current execution identity, usage, and terminal
  result.
- `CoinCopilotCheckpoint` stores immutable continuation state: public messages,
  concise plan status, validated bounded tool facts, counters, and pending
  clarification. It never stores chain-of-thought or provider-native traces.
- `CoinCopilotEvent` stores monotonic, append-only public events for replayable
  SSE.
- `CoinCopilotResumeRequest` makes resume replay idempotent.

A Go worker claims queued runs and calls a stateless Python execution endpoint.
Each initial execution or resume receives a fresh HMAC credential bound to the
owner, run, execution id, expiry, and fixed read-only tool allowlist. Python may
call only the dedicated Go internal read-tool routes and returns typed execution
frames. Go revalidates and sanitizes every frame, persists the checkpoint/event
transactionally, and only then publishes it to Vue.

The existing app-wide chat drawer selects Coin Copilot only when the admin flag
is enabled and the configured model has verified tool-calling capability.
Otherwise it uses the existing supervisor and `/api/agent/chat` contract.

Default limits are eight reasoning iterations, twelve tool calls, sequential
tool execution, 120 seconds per execution, 32 KiB persisted tool result per
call, and one active run per owner. Dollar-cost enforcement is deferred because
the MVP has no trustworthy provider/model pricing source; reliable
input/output token counts are still recorded when reported. Public events are
retained seven days after terminal state; checkpoints/tool results are retained
thirty days; thread messages and final run summaries remain until owner
deletion.

## Consequences

### Positive

- Durable pause/resume and event replay survive browser, Go, and Python process
  restarts without making Python stateful.
- Authentication, owner scoping, idempotency, retention, and cancellation races
  remain beside the database authority.
- The Python capability surface is least-privilege and read-only.
- Legacy chat remains an operational and model-capability fallback.
- No chain-of-thought is required for continuation; explicit task/checkpoint
  state is sufficient.

### Negative

- The Go API gains five tables, a worker, a janitor, an SSE broker, and a new
  execution credential family.
- Internal and public event contracts require shared drift fixtures across Go,
  Python, and Vue.
- Sequential execution may be slower than parallel tools, but is accepted for
  deterministic MVP cancellation and accounting.
- Tool-result truncation can reduce resume fidelity; the checkpoint records a
  digest and truncation metadata so the loss is explicit.

### No constitution amendment

No amendment is required. This decision implements the existing rule in
Principle II: Go owns persistence/authentication and Python remains stateless
and database-free. It does not add a service or move responsibility across a
service boundary. Principle I is preserved through
Handler → Service → Repository → Database, and Principles III, V, VIII, and IX
are satisfied through typed contracts, least-privilege credentials, this ADR,
and automated contract/tamper tests. Section §22 is therefore used to record a
material multi-service design decision, not to waive or alter the constitution.

## Rollback

Set `CoinCopilotEnabled=false`. New starts and resumes are rejected while
existing threads/runs remain readable and cancellable. The legacy supervisor
continues unchanged. Additive tables may remain inert; no destructive rollback
migration is required.

## Related

- [Feature 359 specification](../../specs/359-coin-copilot-harness/spec.md)
- [Feature 359 plan](../../specs/359-coin-copilot-harness/plan.md)
- [Feature 359 contracts](../../specs/359-coin-copilot-harness/contracts/)
- [Feature 217 collection tools](../../specs/217-collection-chat-over-transport-agnostic-tools/spec.md)
- [Feature 218 external adapter](../../specs/218-external-tool-server-adapter/spec.md)
- [ADR 0002: Three-Service Architecture](0002-three-service-architecture.md)
- [ADR 0011: Persisted Deep Agentic Coin Identification](0011-deep-agentic-coin-identification.md)
