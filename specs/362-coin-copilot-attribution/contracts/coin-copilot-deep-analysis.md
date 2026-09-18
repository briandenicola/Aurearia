# Contract: Coin Copilot ↔ Deep Analysis Handoff

## 1. Internal boundary and canonical authentication

```http
POST /api/internal/copilot/tools/deep_analysis_handoff
Authorization: Bearer <execution_token>
Content-Type: application/json
```

The bearer value is the existing Coin Copilot execution token minted by
`InternalTokenService` and verified by
`CoinCopilotExecutionTokenRequired(tokenSvc, "deep_analysis_handoff")`. It is
not a user JWT, internal service token, or Deep job token. Verification binds
owner, run id, current execution id, exact allowed tool, expiry, and revocation.
Owner identity is never accepted in the body.

The route is explicit. No wildcard callback, caller URL, or access to the
separate `/api/internal/tools/*` provider routes is added.

## 2. Envelope limits

- Canonically serialized, sanitized request: at most 65,536 bytes.
- Canonically serialized, sanitized public Coin Copilot event: at most 65,536
  bytes.
- Persisted `completed_tools[].result` for this capability: at most 32,768
  bytes, even if the run's general configured maximum is larger.

Limits are independent. An over-limit request/public event fails closed. A
valid result is proactively projected to 32 KiB as specified in §7; raw JSON is
never sliced.

## 3. Model-visible arguments and Go callback request

Model-visible strict arguments:

```json
{
  "operation": "request",
  "target": { "type": "coin", "id": 42 }
}
```

The Python harness, not the model, adds durable execution fields:

```json
{
  "tool_call_id": "call_01",
  "handoff_idempotency_key": "call_01",
  "expected_checkpoint_version": 3,
  "operation": "request",
  "target": { "type": "coin", "id": 42 }
}
```

For `status`, Python sends `tool_call_id`, `expected_checkpoint_version`,
`operation`, and `job_id`; `handoff_idempotency_key` is forbidden because
status admits no work and creates no handoff row.

Closed fields:

```text
operation = request | status | rerun
target.type = coin | draft
target.id = positive integer
job_id = positive integer
tool_call_id = 1..200 characters
handoff_idempotency_key = 1..128 printable ASCII
expected_checkpoint_version = integer >= 0
```

Mutual rules:

- `request`: target required; job id forbidden.
- `status`: job id required; target forbidden. Status is read-only and does not
  create a handoff admission row; it reads an existing binding or another job
  satisfying §8. Its replay is the completed-tool checkpoint fact.
- `rerun`: target and prior job id required.
- `handoff_idempotency_key` is required for `request`/`rerun` and forbidden for
  `status`.
- Notes, provider overrides, owner ids, snapshot members, hashes, URLs, apply
  targets, proposal edits, and accepted fields are forbidden.

Unknown fields are rejected at Python and Go decoders.

## 4. Go-owned idempotency

The durable key is `(owner, run_id, sha256(handoff_idempotency_key))`. The
request fingerprint binds:

```text
operation
declared target kind/id
prior job id for rerun
current execution id
expected checkpoint version
digest of canonical app context stored on the run
server-computed target snapshot fingerprint
```

- Same key, same binding: return the stored handoff/job result without new
  provider work.
- Same key with changed target kind/id, prior rerun job id, app context, checkpoint
  version, operation, or current target snapshot: HTTP 409
  `handoff_idempotency_conflict`; create no job.
- A new key for an equivalent unchanged snapshot reuses the active or eligible
  retained Deep job through atomic admission.
- Python completed-call dedupe is defense in depth only.

The durable handoff row stores `prior_job_id` separately from `deep_job_id`
(the resulting selected/reused/new job). For `rerun`, `prior_job_id` is
required, owner-scoped, and must bind to the same target; for `request` it is
null. Changing it under the same key is a conflict with zero work.

`status` has no durable handoff-key conflict contract because it cannot admit
work. It is still owner/current-execution authorized and strictly eligible;
replay uses the existing saved completed-tool result.

## 5. Atomic snapshot and admission

Go computes the target snapshot over owner and target kind/id, distinct
obverse/reverse image row identities and immutable file/content versions,
bounded notes/context value and version, provider selection/configuration
generation, and active draft state. The client supplies none of these.

The Go repository transaction/linearizable section verifies current
run/execution/checkpoint/cancellation and live admission flags, resolves the
idempotency binding, re-reads every snapshot token, and atomically reuses or
creates the Deep job plus binding. Changed state returns HTTP 409
`target_changed`; no worker wake or provider call occurs. Artifacts and binding
must be durable before worker publication.

## 6. Result contract

```json
{
  "schema_version": 1,
  "operation": "request",
  "outcome": "reused_result",
  "reason": null,
  "target": {
    "type": "coin",
    "id": 42,
    "display_label": "Maximinus I denarius"
  },
  "job": {
    "id": 314,
    "source": "saved_coin",
    "status": "completed",
    "reused": true,
    "created_at": "2026-09-18T18:00:00Z",
    "completed_at": "2026-09-18T18:02:00Z"
  },
  "input_digest": "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
  "review_url": "/deep-analysis/314",
  "fresh_analysis_available": true,
  "result": {
    "state": "complete",
    "narrative": "Persisted Deep Analysis narrative.",
    "partial_success": false,
    "image_only": false,
    "fields": [],
    "disagreements": [],
    "unresolved_questions": [],
    "coverage": [],
    "attributions": [],
    "limitations": []
  },
  "truncation": {
    "truncated": false,
    "original_bytes": 1024,
    "persisted_bytes": 1024,
    "digest": "abcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcd",
    "omitted_fields": 0,
    "omitted_evidence": 0,
    "omitted_disagreements": 0,
    "omitted_questions": 0
  },
  "limitations": []
}
```

### Closed handoff outcomes

The top-level result discriminant is `outcome`, never `status`. `status` is
permitted only as `job.status` for the Deep job lifecycle.

| Outcome | Meaning |
|---|---|
| `accepted` | new Deep job committed |
| `reused_active` | equivalent queued/running job |
| `reused_result` | equivalent current retained completed/partial result |
| `status` | eligible owner-scoped status/result |
| `retry_available` | prior failed/cancelled/stale/missing/mismatched result |
| `missing_images` | one or both distinct usable faces absent |
| `target_unavailable` | only a previously validated durable owner handoff/checkpoint binding whose coin/draft later disappeared or whose draft was promoted; no target metadata |
| `not_eligible` | anonymous unknown/foreign/legacy-unbound/unknown-source/arbitrary-unbound lookup, including deleted/promoted ids without prior validated binding |
| `unavailable` | live admission capability/queue/capacity unavailable |
| `cancelled` | cancellation linearized before admission |

Every anonymous unknown/foreign/legacy-unbound/unknown-source/arbitrary-unbound
lookup—including a deleted/promoted id presented without a prior validated
durable owner binding—has exactly these canonical UTF-8 bytes:

```json
{"outcome":"not_eligible","reason":null}
```

No job, target, source, ownership, lifecycle, existence, or cause field is
present. The HTTP status, public headers, timing policy, and event projection
also cannot vary by cause.

Only lookup through a previously validated durable owner handoff/checkpoint
binding may distinguish later target disappearance or promotion:

```json
{"outcome":"target_unavailable","reason":null}
```

That response contains no target metadata. Possession of an id, ownership of
an otherwise unbound job, or a deleted/promoted id alone is insufficient.

Allowed `reason` values are:

```text
missing_obverse | missing_reverse | missing_both | duplicate_faces |
target_changed | draft_inactive | source_coin_missing |
deep_disabled | copilot_disabled | attribution_disabled | model_unsupported |
job_at_capacity | queue_full | result_missing | result_expired |
stale | cancelled
```

`reason` is always null when `outcome=not_eligible`. Privacy-safe internal
diagnostic codes such as `legacy_unbound_intake` or `unknown_source` may be
written only to access-controlled telemetry; they are not public contract
values and cannot influence public output. Unknown public outcomes/reasons
fail closed.

### Deep vocabularies

- Sources: `intake`, `saved_coin`, `copilot_draft`.
- Job status: `queued`, `running`, `completed`, `partial`, `failed`,
  `cancelled`.
- Result state: `not_ready`, `complete`, `partial`, `no_match`, `failed`,
  `cancelled`, `stale`, `missing_result`.
- Providers: `numista`, `nomisma`, `ngc`, `ocre`, `rpc`.
- Coverage: `pending`, `running`, `contributed`, `no_match`, `failed`,
  `timed_out`, `skipped`, `not_automated`, `unavailable`.

Confidence must be finite and within `[0,1]`. Citations must pass the existing
Deep host/scheme validator. Invalid citations are omitted and add a limitation;
they are never repaired.

## 7. Deterministic proactive projection

Go builds a complete, canonical, validated projection from persisted
report/proposal data and computes SHA-256 over those complete bytes. It then
builds the persisted form in this priority/stable order:

1. schema, operation, outcome/reason, target/job identity, digest, review URL,
   lifecycle and freshness;
2. result state, partial/image-only flags and limitations;
3. fields sorted by existing Deep field order/name;
4. disagreements sorted by field;
5. coverage/attributions sorted by provider;
6. evidence sorted by field, provider/source, canonical URL;
7. unresolved questions in stored order.

Whole entries are omitted from the tail until canonical bytes are at most
32,768. Required lifecycle/link/limitation fields are never omitted. Text is
bounded before sizing and JSON is never sliced. Metadata records complete
`original_bytes`, actual `persisted_bytes`, omitted counts, and the complete
digest. The final Copilot answer and Vue card disclose truncation. A minimal
envelope that cannot fit fails `invalid_tool_call`.

## 8. Status eligibility

Status/result access requires one:

1. validated owner/run/checkpoint handoff binding to the job;
2. owned `saved_coin` job with a currently existing owned coin; or
3. owned `copilot_draft` job with non-null currently active owned
   `source_draft_id`.

Legacy unbound `intake` and every unknown source are not eligible. Arbitrary
anonymous unknown/foreign/unbound job ids—including deleted/promoted ids
without prior validated durable owner binding—return the exact canonical bytes
`{"outcome":"not_eligible","reason":null}`. Only a previously validated durable
owner handoff/checkpoint binding whose target later disappears/promotes may
return `{"outcome":"target_unavailable","reason":null}`, without target
metadata.

## 9. Apply matrix (existing Deep review UI only)

The handoff contract has no apply operation. The existing proposal service
uses this normative transaction:

| Destination | Exact valid scalars | Notes | Catalog references |
|---|---|---|---|
| Collection coin | `denomination`, `ruler`, `era`, `dateRange`, `mint`, `material`, `weightGrams`, `diameterMm`, `obverseInscription`, `reverseInscription`, `obverseDescription`, `reverseDescription`, `coin_type` | Append job-id-keyed dated/source block | Registry-validate, append, case-insensitive dedupe |
| Wishlist coin | Same existing Deep coin scalar allowlist; no acquisition/value/storage/privacy/status/image fields | Append job-id-keyed dated/source block | Registry-validate, append, case-insensitive dedupe |
| Bound active draft | `workingTitle`, `era`, `dateRange` | Append job-id-keyed dated/source block | Validate/stage in accepted proposal; promotion appends/dedupes transactionally |

Each accepted scalar replaces only itself. Manual notes outside the block,
images, unaccepted fields, relationships, and all pre-existing references are
preserved. Any unsupported field or changed owner/lifecycle/context/proposal/
registry state rejects the entire selected set with `409 re_review_required`
and zero partial writes.

## 10. Cancellation and finish-existing transitions

| Transition | Behavior |
|---|---|
| cancellation commits before admission | zero Deep jobs; `cancelled` |
| admission commits before cancellation | exactly one bound job; subsequent Copilot cancel requests Deep cancel if nonterminal |
| late Deep/provider result after authoritative cancel | cannot win terminal settlement; no report/proposal/event publication |
| any required flag disabled before admission | no handoff/provider work |
| flag disabled after durable admission | job may finish; status/events/cancel/review/edit/confirmed review-page apply remain |
| `rerun` after disable | rejected `unavailable` |

## 11. HTTP mapping

| HTTP | Code/behavior |
|---|---|
| 200 | typed expected outcome |
| 400 | malformed/unknown fields or enum: `invalid_tool_call` |
| 401 | missing/wrong scheme, expired/revoked/wrong owner/run/execution/tool token |
| 409 | `handoff_idempotency_conflict`, `target_changed`, cancellation/state race |
| 413 | request/public envelope over 64 KiB |
| 503 | internal admission unavailable |

A user JWT presented as the execution token is 401. Errors never contain
ownership, paths, provider bodies, queries, secrets, or raw database failures.

## 12. Review URL and reconnect

`review_url` is exactly `^/deep-analysis/[1-9][0-9]*$` and must match `job.id`.
Vue constructs/validates the relative route and rejects absolute,
protocol-relative, credential-bearing, or mismatched URLs. The Agent drawer has
no proposal controls.

Coin Copilot reconnects from its run event sequence/checkpoint. Deep Analysis
reconnects independently from its job event sequence. Replayed completed tools
do not execute callbacks or providers again.

## 13. Compatibility interlock

Before any `copilot_draft` row can exist, a compatibility release must reject
unknown source on list/get/status/stream/retry/apply/worker claim. The Feature
362 flag is default off. Release A must pass guard/hosted evidence, then cross
an explicit external/manual deployment checkpoint with separate user approval;
record the deployed version/commit and verification evidence. CI or image
publication alone is not deployment. Only then may schema, `source_draft_id`,
source recognition, or `copilot_draft` row work begin. Rollback targets that
guard release only after
handoffs are disabled and accepted work is terminal/cancelled. Rows remain
untouched and become usable after re-upgrade.
