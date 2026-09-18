# Data Model: Coin Copilot Attribution Integration

## 1. Relationship overview

```text
CoinCopilotRun
  1 ── * CoinCopilotCheckpoint
           completed_tools[].result
             └── DeepAnalysisHandoffResult ──references──> DeepIdentificationJob

Coin (owned or wishlist) ──input snapshot──> DeepIdentificationJob
QuickCaptureDraft (active) ──source_draft_id + input snapshot──> DeepIdentificationJob

DeepIdentificationJob ──retains──> ReportJSON + ProposalJSON
DeepIdentificationJob ──reviewed/applied by──> DeepIdentificationProposalService
```

The handoff result is a bounded checkpoint fact, not a new table or
attribution record. The Deep job/report/proposal remain authoritative.

## 2. Attribution target

| Field | Type | Rules |
|---|---|---|
| `type` | enum | exactly `coin` or `draft` |
| `id` | positive integer | owner-scoped server-side; foreign equals unknown |
| `displayLabel` | bounded string | server-derived, untrusted display data |
| `state` | enum | `available`, `inactive`, `missing_images`, `not_found` |
| `missingImages` | enum array | subset of `obverse`, `reverse`; no duplicates |

### Validation

- A `coin` may be collection or wishlist, but must belong to the authenticated
  owner and have two distinct, currently usable face images.
- A `draft` must belong to the owner and have
  `QuickCaptureDraftStatusActive`.
- Prompt and app context do not establish ownership.
- The same file/content cannot fill both required roles.
- Draft/coin deletion, promotion, discard, or image change is rechecked while
  building the fingerprint and before admission.

## 3. Deep Analysis job change

Existing `models.DeepIdentificationJob` is reused unchanged except:

| Field | Go type | Storage | Rules |
|---|---|---|---|
| `SourceDraftID` | `*uint` | nullable `source_draft_id`, indexed | Set only for a job snapshotted from an active owned Quick Capture draft; immutable after creation |

`Source` remains `intake` for draft-origin jobs, avoiding an enum and public
contract break. `CoinID` remains exclusive to `saved_coin`. At most one of
`CoinID` and `SourceDraftID` may be non-null.

### Migration and rollback

- Additive GORM `AutoMigrate` column/index only; no backfill.
- Existing rows read `NULL` and keep current behavior.
- Rolling back code leaves an inert nullable column.
- Jobs created with `source_draft_id` remain readable after rollback; applying
  one with old code must fail closed rather than silently create or target a
  different draft.

## 4. Input identity

The existing `services.FingerprintInput` and `ComputeInputFingerprint` remain
the source of equivalence. For draft-origin input, extend the canonical formula
version to include target class and source id so a draft cannot collide with an
unlinked intake snapshot:

```text
sha256(
  "v2" |
  user_id |
  target_type |
  target_id |
  obverse_content_hash |
  reverse_content_hash |
  sorted(hint_hashes) |
  sha256(normalized_bounded_context) |
  sorted(go_selected_providers)
)
```

Saved-coin and ordinary intake v1 jobs remain readable. New Coin Copilot
handoffs use v2. Equivalent active jobs are protected by the existing unique
active fingerprint key. Equivalent terminal lookup is owner/fingerprint scoped,
newest first, and reusable only for `completed`/`partial` with retained,
strictly valid report data.

## 5. Handoff checkpoint fact

`DeepAnalysisHandoffResult` is stored in the existing
`CoinCopilotCheckpoint.completed_tools[].result`.

| Field | Type | Rules |
|---|---|---|
| `schema_version` | literal `1` | required |
| `operation` | enum | `request`, `status`, `rerun` |
| `outcome` | enum | see contract |
| `target` | target summary | required after successful owner resolution |
| `job` | job summary or null | never exposes another owner's job |
| `result` | conversation projection or null | only from retained validated report/proposal |
| `review_url` | relative route or null | exactly `/deep-analysis/{jobId}` |
| `input_digest` | 64-char lowercase hex | digest/freshness comparison; not raw notes/images |
| `fresh_analysis_available` | boolean | true only when explicit rerun is allowed |
| `limitations` | bounded string array | honest missing/partial/unavailable detail |

It inherits existing checkpoint digest, result byte cap, truncation behavior,
tool-call count, and replay rules.

## 6. Conversation projection

The projection is derived on read; it is never stored separately:

- `state`: `not_ready`, `complete`, `partial`, `no_match`, `failed`,
  `cancelled`, `stale`, or `missing_result`.
- `narrative`: bounded persisted narrative.
- `fields[]`: name, proposed value, confidence `[0,1]`, evidence.
- `evidence[]`: source kind/provider, bounded excerpt, optional URL only after
  existing Deep citation-host validation.
- `disagreements[]`: field and all conflicting sourced claims; unresolved
  remains unresolved.
- `unresolved_questions[]`.
- `coverage[]`: existing provider/status/link-out vocabulary.
- `attributions[]`: existing provider-specific text and identifiers.
- `partial_success`, `image_only`, and bounded `limitations[]`.

Owner edit values, acceptance decisions, proposal tokens, raw notes, raw image
paths, credentials, provider queries, and internal errors are excluded.

## 7. State transitions

### Handoff

```text
strict target
  ├─ ambiguous/conflicting ──> Coin Copilot clarification (no tool/no job)
  ├─ missing/inactive/images ─> corrective outcome (no job)
  ├─ equivalent queued/running ─> reused_active
  ├─ equivalent retained completed/partial ─> reused_result
  ├─ prior failed/cancelled/stale/missing result ─> retry_available
  └─ no equivalent current job ─> accepted (existing Deep queue)
```

`rerun` requires explicit owner intent and a previous matching owner-scoped job.

### Cancellation

- Copilot cancellation before handoff admission: no Deep job.
- Deep admission before Copilot cancellation: the job is independently durable;
  use the existing Deep review page to cancel it.
- Deep terminal states never transition back.
- Coin Copilot replay reuses its checkpoint fact and never re-admits the job.

### Apply

- Conversation/status/link/replay: no target write.
- Review page proposal edit: writes only `ProposalJSON`.
- Explicit review-page confirmation: existing
  `DeepIdentificationProposalService.Apply`.
- Draft-origin jobs may apply only to their bound active `SourceDraftID`.
- Structured references use existing validated append/dedupe behavior and never
  replace existing rows.
