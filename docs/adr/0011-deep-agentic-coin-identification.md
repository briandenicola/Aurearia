# ADR 0011: Persisted Deep Agentic Coin Identification

Date: 2026-08-16
Status: Proposed

## Context

Feature 344 adds a multi-minute identification workflow that combines coin
images, optional owner context, an LLM, and bounded catalog providers. Existing
`AIJob` records and direct agent streams do not provide the owner-scoped
artifacts, resumable event history, provider transparency, retention lifecycle,
or confirm-gated draft/coin updates required by this workflow.

The Python agent must remain stateless and must not receive provider credentials
or direct database access. Provider licensing also differs: Numista is
API-backed, Nomisma reconciliation is CC BY 4.0, NGC is link-out only, OCRE is
separately authorized under ADR 0010, and RPC automation remains paused.

## Decision

Deep Analysis uses a sibling Go-owned domain:

- `DeepIdentificationJob` stores owner-scoped lifecycle and bounded results.
- `DeepIdentificationEvent` stores monotonic public SSE events for replay.
- `DeepIdentificationProviderRun` stores operational status, counts, and timing,
  never owner notes, provider queries, reports, or raw claims.
- `DeepIdentificationArtifact` tracks face and ephemeral hint images.

Go persists each translated public event before serving it to Vue. Reconnects
resume by sequence number; retention may prune events while the durable terminal
job remains readable. Python receives all run context per request, emits typed
internal frames, and calls authenticated Go provider tools. Provider HTTP calls,
credentials, budgets, citation allowlists, storage, and all writes stay in Go.

The router records its selected provider set and rationale. The browser may
adjust only providers the backend reports as eligible for automation. Coverage
preserves contributed, no-match, failed, timed-out, skipped, non-automated, and
unavailable outcomes as distinct states.

Results never mutate collection data automatically. Intake results write
accepted fields through the existing Quick Capture draft path; saved-coin
results use the existing coin update service. Hint images are deleted after
every terminal outcome, at restart recovery, and at retention expiry, and are
never promoted.

Deep Analysis defaults off. RPC remains unavailable until a separate ADR records
an authorized API or corpus.

## Consequences

### Positive

- Jobs survive navigation and reconnect without making Python stateful.
- Provider participation, partial failure, and disagreements remain visible.
- Existing validation and owner-scoped write paths remain the only mutation
  boundary.
- Hint-image privacy and bounded event retention are explicit lifecycle rules.

### Negative

- The feature adds four tables, a worker pool, a janitor, and an SSE broker.
- Live subscriber and reconnect counters are process-local; durable job and
  provider metrics are database-derived.
- Event translation must keep internal Python and public browser contracts in
  sync.

## Rollback

Set `DeepIdentificationEnabled=false`. New jobs are rejected while existing
records remain readable until normal retention cleanup. The additive tables are
safe to leave in place; no rollback migration is required.

## Related

- [Feature 344 specification](../../specs/344-deep-agentic-coin-identification/spec.md)
- [Feature 344 plan](../../specs/344-deep-agentic-coin-identification/plan.md)
- [ADR 0007: Shared Numista Lookup Boundary](0007-shared-numista-lookup.md)
- [ADR 0010: OCRE ODbL Provider](0010-ocre-odbl-provider.md)
