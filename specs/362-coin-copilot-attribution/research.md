# Research: Coin Copilot Attribution Integration

## Decision: one fixed Go-owned capability

- **Decision**: Add only `deep_analysis_handoff`, with closed `request`,
  `status`, and `rerun` operations, under the canonical Coin Copilot
  execution-token route.
- **Rationale**: Go already owns Copilot runs, Deep jobs, repositories,
  credentials, providers, and writes. Python can remain a stateless strict
  adapter with no general REST or apply authority.
- **Alternatives considered**: public endpoint calls with a user JWT, separate
  resolve/start/result tools, Python database access, and a new attribution
  pipeline were rejected as broader or duplicative.

## Decision: Go-durable idempotency requires a handoff record

- **Decision**: Add `CoinCopilotDeepHandoff`, uniquely keyed by owner/run/
  idempotency-key hash. Persist operation, target kind/id, execution/checkpoint,
  stored app-context digest, server snapshot fingerprint, required prior job id
  for `rerun`, resulting Deep job id, and outcome in the same admission
  transaction. The prior and resulting job identities are distinct fields.
  `status` remains read-only, creates no handoff row, and replays from the
  existing completed-tool checkpoint.
- **Rationale**: A checkpoint alone leaves a crash window after job creation
  but before checkpoint persistence. It also cannot guarantee HTTP 409 when the
  same key is rebound to a changed target/app context/checkpoint.
- **Alternatives considered**: Python call-id dedupe and active-job
  fingerprinting remain defense in depth but do not durably bind a changed
  request.

## Decision: server snapshot and linearizable admission

- **Decision**: Go computes an immutable snapshot over owner, target kind/id,
  both face row identities and immutable file/content versions, bounded
  notes/context value and version, effective provider selection/configuration
  generation, and active-draft state. Admission re-reads those tokens and
  atomically reuses or creates.
- **Rationale**: Computing a hash before a separate `CreateJob` allows image,
  context, provider, deletion, promotion, or ownership changes between check
  and admission.
- **Alternatives considered**: target-id-only keys and best-effort rechecks
  after creation were rejected because they can launch stale work.

## Decision: use `copilot_draft` with a durable source binding

- **Decision**: Add closed source `copilot_draft` plus non-null
  `source_draft_id` for that source; other sources require null. Apply merges
  into that exact active draft.
- **Rationale**: Current `intake` means unbound uploaded intake and current
  apply creates a new draft. Treating an existing draft as ordinary intake
  loses provenance and risks duplication. Checkpoints/events cannot be the
  Deep proposal service's retained authority.
- **Alternatives considered**: storing draft id in notes/events/proposal,
  passing it only at apply, reusing `AppliedDraftID` as input, or creating a
  second draft were rejected as prunable, tamperable, or semantically false.

## Decision: compatibility guard precedes schema

- **Decision**: First ship a compatibility release that rejects every unknown
  Deep source on list/get/status/stream/retry/apply/worker adoption. Only after
  it is deployed and tested may a later default-off release write
  `copilot_draft`.
- **Rationale**: Current code serializes arbitrary `job.Source` on reads.
  Although current apply rejects an unknown source by mismatch, the current
  binary is not a safe rollback target for status/adoption. The guard release
  is the earliest permitted rollback target.
- **Alternatives considered**: claiming the current old binary is safe was
  rejected by code inspection; coercing unknown source to `intake` is
  expressly forbidden.

## Decision: `selected_replace_notes_append_refs_add`

- **Decision**: In one apply transaction, individually accepted valid scalars
  replace only themselves; notes append one dated/job-id/source block;
  references registry-validate and append/dedupe case-insensitively; any
  unsupported/stale operation rejects the whole set.
- **Rationale**: This preserves manual values and makes replay idempotent.
  Draft references remain staged in the accepted Deep proposal and are merged
  through the validated draft-promotion transaction.
- **Alternatives considered**: wholesale model replacement, notes overwrite,
  reference replacement, and partial best-effort apply were rejected.

## Decision: closed status eligibility

- **Decision**: A status/result is eligible only through a validated durable
  checkpoint/handoff binding, a current owned `saved_coin`, or an active owned
  `copilot_draft`. Legacy unbound intake is `not_eligible`; a previously bound
  deleted coin or deleted/promoted draft is `target_unavailable`.
- **Rationale**: Owner scope alone does not authorize adopting arbitrary intake
  history into a conversation.
- **Alternatives considered**: any owner job id and label-based adoption were
  rejected.

## Decision: separate 64 KiB and 32 KiB limits

- **Decision**: Canonical request and public event envelopes each cap at
  65,536 bytes. Persisted handoff results cap at 32,768 bytes and are
  proactively projected in stable order. The complete canonical projection is
  hashed before omission; persisted metadata includes truncation, original and
  persisted bytes, omitted counts, and digest.
- **Rationale**: Reusing the public-event limit for checkpoint facts violates
  the clarified bound and risks replay bloat.
- **Alternatives considered**: byte slicing, relying only on a configurable
  larger general limit, or silently dropping evidence were rejected.

## Decision: cancellation and feature flags use finish-existing semantics

- **Decision**: Cancel-first creates zero jobs. Admission-first binds exactly
  one; later Copilot cancellation requests Deep cancellation before terminal
  settlement. `CoinCopilotAttributionEnabled` is default off. Disabling it,
  Coin Copilot, or Deep Analysis blocks new handoffs/reruns but permits accepted
  jobs to finish and remain readable, cancellable, reviewable, and applicable
  only through the existing review UI.
- **Rationale**: This reconciles the two durable state machines without
  stranding accepted work or allowing late results after cancellation.
- **Alternatives considered**: letting an admitted job ignore Copilot
  cancellation and disabling reads/review were rejected by the clarification.

## Decision: mandatory evidence gates

- **Decision**: ADR 0017 must be accepted before implementation that writes new
  rows. Go build/vet/tests, web install/lint/type/build/tests, Python dependency
  install/syntax-or-build/type/lint/tests, compatibility migration/rollback,
  contract, owner-isolation, tamper, and regression gates must pass locally or
  in an equivalent hosted job. A local limitation never waives a gate.
- **Rationale**: This is a multi-service semantic migration and security
  boundary.
- **Alternatives considered**: documenting an unrun gate as an environment
  limitation was rejected.

## Clarification resolution

All design questions are resolved. No `NEEDS CLARIFICATION` item remains.
