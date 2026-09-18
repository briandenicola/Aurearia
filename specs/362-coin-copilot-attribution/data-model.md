# Data Model: Coin Copilot Attribution Integration

## 1. Relationships

```text
CoinCopilotRun 1 ── * CoinCopilotDeepHandoff * ── 1 DeepIdentificationJob
CoinCopilotRun 1 ── * CoinCopilotCheckpoint
DeepIdentificationJob(source=copilot_draft) * ── 1 QuickCaptureDraft
DeepIdentificationJob 1 ── report/proposal/events/artifacts
```

The handoff row is an idempotency/admission record, not a second attribution
result. The Deep job/report/proposal remain authoritative.

## 2. `CoinCopilotDeepHandoff` (new)

| Field | Type/rule |
|---|---|
| `ID` | primary key |
| `UserID` | required, owner index |
| `RunID` | required Coin Copilot run id |
| `ExecutionID` | required execution accepted at admission |
| `HandoffKeyHash` | SHA-256; unique with `UserID,RunID` |
| `RequestFingerprint` | Operation-specific SHA-256: for `request`, operation + target kind/id + current execution id + checkpoint version + app-context digest + current snapshot; for `rerun`, the same plus required `PriorJobID` |
| `ExpectedCheckpointVersion` | non-negative exact version |
| `AppContextDigest` | digest of canonical app context stored on the run |
| `Operation` | `request` or `rerun` for admission rows |
| `TargetKind` | `coin` or `draft` |
| `TargetID` | positive id; never trusted for owner identity |
| `PriorJobID` | nullable; required only for `rerun`, forbidden for `request`; owner/target binding revalidated |
| `TargetSnapshotFingerprint` | complete server snapshot digest |
| `DeepJobID` | resulting selected/reused/new job; required once admission commits and never overloaded as `PriorJobID` |
| `AdmissionOutcome` | `created`, `reused_active`, or `reused_result` |
| `CreatedAt`, `UpdatedAt` | UTC |

Unique constraint:

```text
(user_id, run_id, handoff_key_hash)
```

Operation-specific request fingerprints:

```text
request = operation | target kind/id | execution | checkpoint |
          app-context digest | current snapshot
rerun   = operation | target kind/id | prior job id | execution | checkpoint |
          app-context digest | current snapshot
```

Same key/same request returns the stored binding. Same key with a changed target
kind/id, prior rerun job id, stored app context, execution, checkpoint
version, operation, or current snapshot returns HTTP 409 and creates no job.

`status` is read-only and does not create a `CoinCopilotDeepHandoff` row or use
handoff-key idempotency. It may read only an existing durable handoff binding or
another job allowed by the closed eligibility predicate. Its replay is the
existing Coin Copilot completed-tool checkpoint fact.

## 3. `DeepIdentificationJob` changes

Add:

```text
DeepJobSourceCopilotDraft = "copilot_draft"
SourceDraftID *uint -> source_draft_id (nullable, indexed)
```

Closed invariant:

| Source | `CoinID` | `SourceDraftID` |
|---|---:|---:|
| `intake` | null | null |
| `saved_coin` | required | null |
| `copilot_draft` | null | required |
| unknown | rejected | rejected |

The `copilot_draft` row keeps its source binding after the draft is promoted or
deleted so protected history remains auditable, but it cannot be adopted or
applied. Public status lookup uses the same canonical `not_eligible`/null-reason
body as every other ineligible case.

## 4. Server-computed target snapshot

Canonical v2 snapshot input:

```text
schema_version = 2
owner_id
target_kind
target_id
target_state                    # active for a draft
target_updated_at/version
obverse_image_id
obverse_image_created_at/version
obverse_content_sha256
reverse_image_id
reverse_image_created_at/version
reverse_content_sha256
bounded_context_sha256
bounded_context_version
sorted_effective_provider_ids
provider_configuration_generation
```

Go normalizes and serializes this structure, then computes SHA-256. The caller
cannot submit any snapshot member. Distinct image ids and content hashes are
required. Target rows, face rows/version tokens, context version, provider
generation, owner, and draft state are re-read in the atomic admission
section. Any mismatch is `target_changed`/HTTP 409 with no job.

Provider generation is a Go-computed digest/version over effective provider
enablement and configuration affecting the run (Numista/Nomisma, separately
enabled OCRE, NGC link-out/quick evidence, RPC unavailable), not a
caller-supplied list.

## 5. Atomic admission

Within one repository transaction/linearizable admission boundary:

1. lock/read the current Copilot run and checkpoint;
2. require current execution, no authoritative cancellation, and all new-
   admission flags enabled;
3. resolve an existing handoff key before target inspection;
4. lock/read owner target, active draft state, face/version rows, bounded
   context version, and provider generation;
5. compare the precomputed candidate snapshot;
6. find an equivalent active or eligible retained terminal job;
7. create at most one Deep job with ready artifacts when no reuse applies;
8. insert the handoff binding; and
9. commit before worker wake/event publication.

Unique active Deep fingerprint and unique handoff key are database backstops.
Staged artifact files are cleaned on rollback.

## 6. Status/result eligibility

| Condition | Outcome |
|---|---|
| Valid owner handoff/checkpoint binding and current target | eligible |
| Owned `saved_coin` job and coin still exists | eligible |
| Owned `copilot_draft`, non-null source draft, draft active | eligible |
| Legacy unbound `intake` | canonical `status=not_eligible`, `reason=null` |
| Unknown/foreign/unbound job id | same canonical ineligible body |
| Previously bound coin deleted | same canonical ineligible body |
| Previously bound draft deleted/promoted/discarded | same canonical ineligible body |
| Unknown Deep source | same canonical ineligible body; no projection/adoption/apply |

Queued/running jobs return lifecycle only plus the review URL. Completed/
partial jobs require a retained, valid report. Failed/cancelled/stale/
missing-result or snapshot-mismatched jobs are never current success.

All ineligible status rows map to the canonical serialized public body
`{"reason":null,"status":"not_eligible"}` and expose no metadata. Optional
privacy-safe internal diagnostic codes are telemetry-only and cannot affect
the public response.

## 7. Apply field matrix

All selected operations execute in one transaction after ownership,
destination/source, lifecycle, proposal version/applicability, target context
version, and registry state are revalidated.

| Destination | Exact accepted scalars | Notes | Catalog references | Everything else |
|---|---|---|---|---|
| Collection coin | `denomination`, `ruler`, `era`, `dateRange`, `mint`, `material`, `weightGrams`, `diameterMm`, `obverseInscription`, `reverseInscription`, `obverseDescription`, `reverseDescription`, `coin_type` replace only themselves | Append the job-id-keyed dated/source block | Registry-validate; append; case-insensitive dedupe | Reject whole apply |
| Wishlist coin | Same existing Deep coin scalar allowlist; no acquisition/value/storage/privacy/status/image fields | Append the job-id-keyed dated/source block | Registry-validate; append; case-insensitive dedupe | Reject whole apply |
| Existing active draft | `workingTitle`, `era`, `dateRange` replace only themselves | Append the job-id-keyed dated/source block | Validate and retain as accepted staged proposal references; promotion appends/dedupes in its transaction | Reject whole apply |

Notes block:

```text
## Deep Analysis - YYYY-MM-DD (job <jobID>)
Source: Coin Copilot Deep Analysis
<accepted bounded notes>
```

An existing block with the same job id is replaced in place on replay; a new
job appends one block. Manual text outside the block is byte-for-byte
preserved.

Draft reference staging uses the authoritative accepted
`catalogReferences` in the bound Deep proposal plus `AppliedDraftID`.
Promotion revalidates registry/equivalence and merges them with the draft's
existing selected reference in the same promotion transaction. No destructive
replacement occurs.

## 8. Bounded checkpoint result

The complete Go projection is canonicalized, validated, and SHA-256 hashed.
The persisted projection is then built in stable order to at most 32,768 bytes.

Required truncation metadata:

```text
truncated: boolean
original_bytes: integer          # complete canonical result
persisted_bytes: integer         # actual saved bytes, <= 32768
digest: 64 lowercase hex         # complete canonical result
omitted_fields: integer
omitted_evidence: integer
omitted_disagreements: integer
omitted_questions: integer
```

Request and public event envelopes independently cap at 65,536 bytes after
canonical serialization and sanitization. JSON and UTF-8 strings are never
byte-sliced.

## 9. Cancellation and flags

- Cancel before admission: no handoff/job rows.
- Admission before cancel: exactly one binding/job; cancel requests the bound
  Deep job and late terminal content cannot win `SettleTerminal`.
- `CoinCopilotAttributionEnabled` defaults false.
- New request/rerun requires attribution, Copilot, model capability, and Deep
  gates.
- After durable admission, flag changes do not block worker completion, status,
  events, Deep cancel, review, proposal edit, or confirmed review-page apply.

## 10. Migration and rollback

Release A (no new rows): strict known-source guard plus default-off setting.
Release B: additive handoff table, source-draft column/index, and new source
recognition. No backfill is required.

Release A must first pass its guard artifact and hosted evidence gates. A
separate external/manual checkpoint then requires user approval to deploy it;
CI success or image publication does not satisfy this checkpoint. Record the
deployed version/commit and verification evidence before Release B schema,
`source_draft_id`, source recognition, or `copilot_draft` row work begins.

Rollback preserves rows and targets Release A only after disabling handoffs and
draining/cancelling accepted jobs. Release A rejects `copilot_draft` on all
adoption/read/apply paths and never coerces it to `intake`. Re-upgrade restores
access. Dropping the table/column is neither required nor permitted during
operational rollback.
