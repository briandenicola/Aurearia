# ADR 0017: Go-Owned Coin Copilot Deep Analysis Handoff

Date: 2026-09-18
Status: Accepted

## Context

Feature 362 lets Coin Copilot satisfy an explicit owner request to attribute
one owned coin or one active Quick Capture draft by reusing the shipped Deep
Analysis workflow. This is the first Coin Copilot capability allowed to cause a
durable side effect: admission of a Deep Identification job.

ADR 0016 keeps Coin Copilot state, execution credentials, idempotency,
cancellation, and replay in Go while Python remains stateless. ADR 0011 keeps
Deep Identification jobs, artifacts, events, providers, reports, proposals, and
writes in Go. Neither record authorizes a cross-workflow side effect, binds a
Deep job to an existing draft, defines mixed-version behavior for a new Deep
source value, or specifies how accepted proposal data merges into that draft.

The current pre-Feature-362 binary does not fail closed for an unknown Deep
source on every read: `toDeepJobDTO` serializes `job.Source` as an arbitrary
string. Its apply switch happens to reject an unknown source, but status/list/
stream adoption is not guarded. Therefore rollback directly to that binary
cannot be claimed safe.

## Decision

### Fixed Go-owned callback authority

Add exactly one Coin Copilot callback capability,
`deep_analysis_handoff`, registered on the fixed internal route
`POST /api/internal/copilot/tools/deep_analysis_handoff`. It uses the existing
Coin Copilot execution-token middleware and is bound to owner, run, execution,
tool name, expiry, and revocation state. A user JWT, internal service token,
Deep job token, request-supplied owner, arbitrary URL, wildcard callback, or
generic REST operation is never accepted.

The closed operations are `request`, `status`, and `rerun`. `request` or
`rerun` may admit/reuse a Deep job after explicit owner intent. The capability
has no proposal edit, accept, apply, target mutation, provider query, generic
network, filesystem, shell, or database operation. Python only validates and
forwards strict data; Go owns all durable effects.

### Durable idempotency and atomic admission

Add a Go-owned `CoinCopilotDeepHandoff` record. Its unique durable key binds
the owner, Copilot run, and handoff idempotency key/tool-call identity to:

- current execution and expected checkpoint version;
- canonical stored app-context digest;
- operation, declared target kind/id, and—only for `rerun`—the required prior
  Deep job id;
- the server-computed target snapshot fingerprint; and
- the selected/reused Deep job id and admission outcome.

Reusing the key with any changed binding returns HTTP 409 and creates no job.
Python call deduplication remains defense in depth.

The canonical operation-specific request fingerprint is:

- `request`: operation + target kind/id + current execution id + checkpoint
  version + app-context digest + current snapshot fingerprint;
- `rerun`: operation + target kind/id + **prior job id** + current execution id
  + checkpoint version + app-context digest + current snapshot fingerprint.

`PriorJobID` is persisted separately from the resulting `DeepJobID`; those
identities must never be overloaded.

`status` is read-only and creates no handoff row. It can read only an existing
durable handoff binding or another job satisfying the closed eligibility
predicate. Coin Copilot completed-tool checkpoints provide status-call replay;
there is no status key that can admit work.

Go computes the target snapshot; the request cannot supply hashes or versions.
The snapshot covers target kind/id and owner, obverse and reverse row identity
plus immutable file/content version/hash, bounded notes/context value and
version, effective provider set and provider-configuration generation, and
active-draft state.

One repository transaction or equivalent linearizable critical section:

1. verifies the current Copilot run/execution/checkpoint and all live feature
   gates;
2. resolves an existing idempotency binding before inspecting a changed target;
3. re-reads owner, target lifecycle, face rows/versions, context version, and
   provider generation;
4. rejects a changed candidate snapshot;
5. atomically reuses an equivalent eligible job or creates exactly one Deep
   job and handoff binding; and
6. publishes/wakes workers only after commit and artifact readiness.

Staged files are removed if the transaction fails. No worker can claim a job
before both required face artifacts and the handoff binding are durable.

### `copilot_draft` and existing-draft merge

Add the closed `DeepJobSource` value `copilot_draft` and nullable
`source_draft_id`. The invariant is:

- `source=copilot_draft` requires a non-null, owner-matching draft that was
  active at admission;
- all other sources require `source_draft_id IS NULL`; and
- unknown source values are rejected, never treated as `intake`.

The existing Deep Proposal editor remains the only apply surface. Applying a
`copilot_draft` proposal targets the same bound draft, not a new draft.
Immediately before apply, one transaction revalidates owner, source/destination,
active draft state, target/context version, selected fields, field
applicability, and reference-registry state. Any conflict rejects the entire
selection with no partial mutation.

The merge policy is `selected_replace_notes_append_refs_add`:

- each individually accepted destination-valid scalar replaces only that exact
  scalar;
- notes append one job-id-keyed, dated,
  `Source: Coin Copilot Deep Analysis` block and preserve all manual notes;
- accepted catalog references are registry-validated, appended, and
  case-insensitively deduplicated; existing rows are never replaced/deleted;
- active-draft references remain staged in the accepted Deep proposal linked
  by `applied_draft_id`, then join the existing validated promotion transaction
  when the draft becomes a coin; and
- any unsupported field rejects the whole apply.

Collection and wishlist targets retain their existing Deep Proposal scalar
allowlist and validated `CoinReferenceService` append path. Acquisition,
valuation, storage, privacy/status, image, and relationship fields remain
outside every Coin Copilot apply contract.

### Status eligibility

Status/result projection is allowed only when the owner-scoped job is:

1. already bound by a validated durable Coin Copilot handoff/checkpoint;
2. a current owned `saved_coin` job whose coin still exists; or
3. a `copilot_draft` job whose `source_draft_id` is still an active owned draft.

Legacy unbound `intake` jobs are not adopted. An unbound/foreign/unknown job
returns `not_eligible` without metadata. A previously validated binding whose
coin was deleted or draft was deleted/promoted returns `target_unavailable`
without target metadata. These outcomes never expose whether an arbitrary
foreign identifier exists.

### Cancellation and feature-disable behavior

Admission and cancellation are linearized per Copilot execution. Cancel-first
creates zero Deep jobs. Admission-first creates/reuses exactly one and durably
binds it. If the Copilot run is then cancelled before the Deep job is terminal,
Go invokes the existing Deep cancellation path for that bound job; the
race-safe Deep terminal settlement prevents late provider/agent report or
proposal commitment and late Copilot frames are rejected.

Feature 362 adds `CoinCopilotAttributionEnabled`, default `false`. New handoffs
and reruns require this flag, `CoinCopilotEnabled`, model tool capability, and
`DeepIdentificationEnabled` at admission. Disabling any gate admits no new
handoff/provider work. A job already durably accepted may finish; status,
events, cancellation, report/proposal review, proposal edit, and confirmed
existing-UI apply remain available. This is `finish_existing`, not a new
conversational write exception.

### Bounds and conversational projection

Canonical sanitized request and public event envelopes are each capped at
64 KiB. A persisted handoff tool result is independently capped at 32 KiB,
regardless of a larger general Coin Copilot setting.

Go first constructs and validates the complete canonical projection and hashes
those complete bytes. It then deterministically retains lifecycle/target/job/
review-link fields, field claims in stable field order, conflicts, coverage,
attribution, and evidence in stable source order until the 32 KiB limit is
reached. The persisted result records `truncated`, complete-result
`original_bytes`, actual `persisted_bytes`, omitted field/evidence/question
counts, and SHA-256 `digest`. It never slices JSON or text mid-codepoint. The
final answer must disclose omission. If the fixed minimal envelope cannot fit,
the tool fails closed.

### Mixed-version compatibility and rollback

Rollout has a mandatory compatibility release before any schema or row:

1. release a source-validation interlock that recognizes only the then-current
   `intake` and `saved_coin` values and rejects all others on list/get/status/
   stream/retry/apply/worker adoption;
2. add the default-off Feature 362 flag and verify it admits no work;
3. deploy that guard everywhere and pass the mixed-version tests;
4. only then deploy the additive handoff table, `source_draft_id`, and binary
   that recognizes `copilot_draft`; and
5. enable the flag only after migrations and contracts pass.

Rollback first disables new handoffs, keeps a compatible Feature 362 binary
until accepted jobs reach terminal/cancelled state, and preserves all rows.
Rollback may then target the compatibility-guard release: it rejects
`copilot_draft` reads/adoption/apply and leaves records untouched for later
re-upgrade. Rollback to the current unguarded pre-feature binary is prohibited.

Executable hosted tests launch the guard and Feature 362 binaries against
copied pre-upgrade/upgraded databases, prove unknown-source rejection and
record preservation, and prove re-upgrade restores review. Local inability to
run this matrix is not a waiver.

## Consequences

### Positive

- Attribution still has one engine, one job lifecycle, one proposal, and one
  review/apply surface.
- Side-effect authority, durable idempotency, owner isolation, cancellation,
  and rollback stay beside the Go database authority.
- Draft application preserves manual data and targets the exact source draft.
- Replay and reconnect do not repeat provider work.

### Negative

- One handoff table, one nullable Deep-job column, a new closed source value,
  and a cross-domain admission transaction are required.
- Existing Deep proposal apply must become all-or-nothing for the selected set.
- Rollback is limited to the compatibility-guard release until a compatible
  binary is restored; arbitrary older binaries are not safe.

### No constitution waiver

No principle is waived. This ADR is required by Principles II, III, V, and VIII
because it introduces a multi-service contract, limited durable side-effect
authority, and a semantic schema/source change.

## Rollback

1. Set `CoinCopilotAttributionEnabled=false`.
2. Reject new handoffs/reruns while allowing accepted jobs to finish or cancel.
3. Verify no queued/running `copilot_draft` jobs remain.
4. Preserve the additive tables/column and all job/report/proposal rows.
5. Roll back no earlier than the source-validation compatibility release.
6. Run the rollback matrix; unknown `copilot_draft` status/adoption/apply must
   fail closed and rows must remain unchanged.
7. Re-deploy a compatible binary to restore review/apply.

## Related

- [Feature 362 specification](../../specs/362-coin-copilot-attribution/spec.md)
- [Feature 362 plan](../../specs/362-coin-copilot-attribution/plan.md)
- [ADR 0016: Go-Owned Durable Coin Copilot State](0016-go-owned-durable-coin-copilot-state.md)
- [ADR 0011: Persisted Deep Agentic Coin Identification](0011-deep-agentic-coin-identification.md)
- [ADR 0012: Vision-First Deep Identification](0012-vision-first-deep-identification.md)
- [ADR 0013: Wishlist Coins May Hold Catalog References](0013-wishlist-coins-may-hold-catalog-references.md)
