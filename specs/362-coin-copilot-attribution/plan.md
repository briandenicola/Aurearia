# Implementation Plan: Coin Copilot Attribution Integration

**Branch**: `beta` (existing; no create/switch) | **Date**: 2026-09-18 | **Spec**: [spec.md](./spec.md)
**ADR**: [ADR 0017](../../docs/adr/0017-coin-copilot-deep-analysis-handoff.md)
**Input**: `specs/362-coin-copilot-attribution/spec.md`

## Summary

Add one fixed, typed, Go-owned `deep_analysis_handoff` capability so Coin
Copilot can resolve an exact owned coin or active Quick Capture draft, atomically
reuse/create the existing durable Deep Analysis job, explain only its persisted
validated result, and link to the existing `/deep-analysis/:jobId` review page.

Go owns target snapshots, durable handoff idempotency, atomic admission,
cancellation, repositories, providers, reports/proposals, and every confirmed
write. Python remains stateless/database-free. Conversation has no apply
operation. Existing-draft apply uses the new closed `copilot_draft` source and
`source_draft_id` with `selected_replace_notes_append_refs_add`.

## Technical Context

**Language/Version**: Go 1.26.1; Python 3.12; TypeScript/Vue 3
**Primary Dependencies**: Gin, GORM/SQLite; FastAPI, Pydantic, LangGraph/LangChain; Vue Router, Pinia, Vite/PWA
**Storage**: Existing Deep/Copilot tables plus additive `coin_copilot_deep_handoffs` and nullable indexed `deep_identification_jobs.source_draft_id`
**Testing**: Go build/vet/testing/integration; Python dependency install, compile, strict Pydantic contract tests, ruff, pytest; web clean install, ESLint, `vue-tsc --build`, Vitest, Vite build; hosted mixed-binary migration/rollback matrix
**Target Platform**: Self-hosted single-node Docker; desktop and mobile/PWA browsers
**Project Type**: Three-service web application
**Performance Goals**: Zero duplicate active jobs/provider work for equivalent/replayed requests; 64 KiB request/public-event limits; 32 KiB persisted handoff result; no additional provider fan-out
**Constraints**: Feature 362 default off; canonical execution-token callback only; no Python database/filesystem/shell/generic HTTP/apply; no new provider/runtime dependency; independent Copilot and Deep budgets; Fast Identify and legacy fallback unchanged
**Scale/Scope**: Personal self-hosted deployment, fewer than 10 concurrent users; one bounded cross-workflow capability

No `NEEDS CLARIFICATION` item remains.

## Constitution Check

*GATE: Evaluated before research and re-evaluated after design.*

| Gate | Evidence | Result |
|---|---|---|
| §0 hierarchy | Constitution, PRD §5.3, clarified Feature 362, Features 344/351/352/359/361, and ADRs 0010-0013/0016 were applied in order. | PASS |
| Principle I | Handler parses only; an HTTP-agnostic handoff service orchestrates; repositories own the cross-domain transaction and all GORM; composition root injects dependencies. | PASS |
| Principle II | Go alone owns persistent state, callback authority, snapshots, providers, and writes. Python remains per-request/stateless; Vue calls Go only. | PASS |
| Principle III | Closed Go/Pydantic/TypeScript contracts, unknown-field rejection, mirrored fixtures, finite confidence, URL validation, Swagger/OpenAPI drift tests, and exact byte bounds are required. | PASS |
| Principle IV | Reuses the existing engine, job workers, proposal, write bridge, routes, and Vue review page. One capability/table/source binding is the smallest complete safe change. | PASS |
| Principle V | Canonical execution-token owner/run/execution/tool binding, nondisclosing eligibility, atomic changed-target checks, body/result bounds, and tamper tests fail closed. | PASS |
| Principle VI | Existing drawer and Deep page retain tokens, Lucide icons, narrow/mobile layout, 44 px controls, PWA navigation, and reconnect. | PASS |
| Principle VIII / §21 | **ADR 0017 is required, not conditional. Its acceptance gate is satisfied. Phase 0 guard evidence and the separately approved Release A deployment checkpoint remain mandatory before schema/row-producing work.** | PASS |
| §§17/21 and Principle IX | Every local gate must pass or its equivalent hosted gate must pass; environment limitations are never waivers. Exact-path race, rollback, security, contract, and blast-radius suites are mandatory. | PASS |

### Post-design recheck

Design is constitution-aligned and the ADR 0017 acceptance gate is satisfied.
Implementation is blocked only on passing the Phase 0 compatibility-guard
evidence and completing the separately approved Release A deployment
checkpoint before schema/row-producing work. No waiver is requested.

## Project Structure

### Feature documentation

```text
specs/362-coin-copilot-attribution/
├── spec.md
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
└── contracts/
    └── coin-copilot-deep-analysis.md

docs/adr/
└── 0017-coin-copilot-deep-analysis-handoff.md
```

`tasks.md` is intentionally not revised by this planning pass.

### Implementation paths

```text
src/api/
├── deps.go
├── routes_internal.go
├── routes_protected.go
├── database/database.go
├── models/
│   ├── appsetting.go
│   ├── coin_copilot.go
│   └── deep_identification_job.go
├── repository/
│   ├── coin_copilot_repository.go
│   ├── deep_identification_repository.go
│   └── quick_capture_repository.go
├── services/
│   ├── coin_copilot_contract.go
│   ├── coin_copilot_service.go
│   ├── coin_copilot_worker.go
│   ├── deep_identification_service.go
│   ├── deep_identification_proposal.go
│   ├── quick_capture_service.go
│   └── settings_service.go
├── handlers/
│   ├── coin_copilot_internal_tools.go
│   └── deep_identification.go
└── integration/

src/agent/
├── pyproject.toml
├── app/models/requests.py
├── app/models/responses.py
├── app/tools/copilot_collection_tools.py
├── app/teams/coin_copilot.py
└── tests/

src/web/
├── src/api/endpoints/agent.ts
├── src/types/agent.ts
├── src/composables/useCoinCopilot.ts
├── src/composables/useCoinSearchChat.ts
├── src/components/CoinSearchChat.vue
├── src/components/chat/CopilotRunProgress.vue
├── src/pages/DeepAnalysisPage.vue
├── src/router/index.ts
└── src/**/__tests__/
```

**Structure Decision**: Extend existing domains at their explicit composition
points. Do not add a service, page, provider, attribution engine, proposal
model, editor, or generic callback router.

## Architecture and Existing Symbol Reuse

### Coin Copilot

- Extend `services.CoinCopilotAllowedTools`, `copilotCallbackTools`,
  `AuthorizeToolCall`, `FinishToolCall`, and current-execution/cancellation
  handling for exactly `deep_analysis_handoff`.
- Reuse `CoinCopilotRun.AppContextJSON`, `ExecutionID`,
  `CheckpointVersion`, snapshotted limits, cancellation, checkpoints, events,
  broker, and worker.
- Add `CoinCopilotDeepHandoff` and a repository admission transaction for
  crash-safe Go-owned binding. Checkpoint `completed_tools` remains replay
  material, not the sole idempotency authority.
- Persist `PriorJobID` separately from the resulting `DeepJobID`. `rerun`
  requires the prior id and revalidates that it belongs to the same owner and
  target.
- Register only
  `CoinCopilotExecutionTokenRequired(..., "deep_analysis_handoff")` in
  `routes_internal.go`.
- Add the default-off setting through the current
  `models/appsetting.go` model and setting-key/default constants in
  `services/settings_service.go`.

### Deep Analysis

- Reuse `DeepIdentificationService` workers/artifacts/broker/janitor,
  `ComputeInputFingerprint`, `DeepIdentificationRepository` active unique key,
  `SettleTerminal`, `RequestCancel`, retained reports/proposals, and provider
  budgets.
- Add `DeepJobSourceCopilotDraft`, `SourceDraftID`, strict
  `IsDeepJobSource`/source-binding validation, and latest eligible terminal
  lookup.
- Do not route a draft through `CreateJobFromIntake` with source `intake`.
  Add a typed snapshot admission path that preserves artifact readiness and
  worker wake-after-commit.
- Reuse the existing citation/provider validators and proposal parsing to build
  a non-authoritative conversational projection.

### Draft and proposal

- Resolve drafts with `QuickCaptureRepository.GetDraftForOwner` and require
  `QuickCaptureDraftStatusActive`.
- Reuse the current exact scalar allowlists in
  `deep_identification_proposal.go`.
- Replace current `copilot_draft` behavior with merge into `SourceDraftID`;
  ordinary `intake` continues to create a new draft.
- Use a transaction-capable write seam so proposal/version/destination/
  references and all selected writes commit or roll back together.
- Stage draft references in accepted `ProposalJSON` linked by
  `AppliedDraftID`; extend the existing validated promotion transaction to
  append/dedupe them on the promoted coin.

### Python and Vue

- Extend strict `COPILOT_ALLOWED_TOOLS`, `CALLBACK_TOOLS`, `ARG_MODELS`,
  `RESULT_MODELS`, `CopilotCompletedTool`, definitions, labels, and summaries.
- The harness injects tool-call id and current checkpoint version; the model
  cannot supply owner, app-context digest, snapshot, provider override, or
  apply data.
- Execute request/rerun alone, not in a parallel tool group. Preserve all
  existing cancellation and budget checks.
- Vue renders a bounded status/result card and validates/constructs
  `/deep-analysis/{jobId}`. `DeepAnalysisPage.vue` remains the only review/
  editor/apply surface.

## Exact Field and Merge Contract

Every selected operation is revalidated and applied atomically.

| Destination | Individually accepted scalars | Notes | References | Unsupported |
|---|---|---|---|---|
| Collection | `denomination`, `ruler`, `era`, `dateRange`, `mint`, `material`, `weightGrams`, `diameterMm`, `obverseInscription`, `reverseInscription`, `obverseDescription`, `reverseDescription`, `coin_type` | append dated/job/source block | registry-validate, append, case-insensitive dedupe | reject whole apply |
| Wishlist | same existing Deep coin scalar allowlist; no acquisition/value/storage/privacy/status/image fields | append dated/job/source block | registry-validate, append, case-insensitive dedupe | reject whole apply |
| Existing active draft | `workingTitle`, `era`, `dateRange` | append dated/job/source block | validate and stage in accepted proposal; promotion appends/dedupes transactionally | reject whole apply |

An accepted scalar replaces only itself. Notes use the existing job-id-keyed
Deep block convention plus `Source: Coin Copilot Deep Analysis`; replay updates
that block without duplication. Manual text outside it, images, unaccepted
fields, relationships, and existing references remain unchanged.

Immediately before write, the same transaction re-reads owner, destination kind
and lifecycle, source/source-draft binding, target/context version, proposal
version and selected applicability, and reference registry/equivalence. Any
change returns `409 re_review_required` and zero writes.

## Closed Status Eligibility

Eligible:

1. validated durable handoff/checkpoint binding for the owner;
2. current owned `saved_coin` job with existing coin; or
3. `copilot_draft` with non-null active owned `source_draft_id`.

Closed failures:

- anonymous unknown/foreign job ids, legacy-unbound intake, unknown-source
  jobs, and arbitrary unbound ids—including deleted/promoted ids without a
  prior validated durable owner binding—return the exact canonical public
  bytes `{"outcome":"not_eligible","reason":null}` with no metadata;
- only a previously validated durable owner handoff/checkpoint binding whose
  coin/draft later disappears or whose draft is promoted may return
  `outcome=target_unavailable`, without target metadata;
- failed/cancelled/stale/expired/missing report or changed current snapshot:
  `retry_available`, never current success.

All anonymous ineligibility causes are public-byte-equivalent after canonical
serialization. Privacy-safe internal diagnostic codes may distinguish causes
in protected telemetry only; they cannot change the public HTTP status, body,
headers, timing policy, or event projection.

Queued/running eligible jobs return lifecycle/link only. Pruned events do not
invalidate a retained terminal report.

## Server Snapshot and Atomic Admission

Go computes and canonicalizes:

```text
owner
target kind/id/state/version
obverse row id/version/content hash
reverse row id/version/content hash
bounded notes/context hash and version
sorted effective providers
provider configuration generation
```

One linearizable repository boundary:

1. validates run/execution/checkpoint/cancellation and all live new-admission
   flags;
2. resolves the durable handoff key first;
3. re-reads owner target, faces, context, provider generation, and active draft;
4. compares snapshot and rejects change with HTTP 409;
5. reuses an equivalent active or retained eligible job, or creates exactly one
   job;
6. persists artifacts and handoff binding before commit; and
7. wakes workers only after commit.

The operation-specific request fingerprint binds:

- request: operation + target kind/id + execution + checkpoint version +
  stored app-context digest + current snapshot;
- rerun: operation + target kind/id + prior job id + execution + checkpoint
  version + stored app-context digest + current snapshot.

`PriorJobID` and resulting `DeepJobID` are distinct durable columns. Same key
with any changed binding is `handoff_idempotency_conflict` and creates zero
jobs. A new key with the same snapshot still reuses the Deep job.

`status` is read-only, creates no handoff row, and has no handoff-key
idempotency contract. It can read an existing durable binding or another
closed-eligible job, and its replay comes from the existing completed-tool
checkpoint.

## Bounds and Projection

- Request envelope: 65,536 canonical sanitized bytes.
- Public event: 65,536 canonical sanitized bytes.
- Persisted handoff result: 32,768 canonical bytes maximum, independent of a
  larger general run setting.

Go hashes the complete canonical result, then proactively retains whole entries
in deterministic priority/stable order. The persisted result carries
`truncated`, `original_bytes`, `persisted_bytes`, full-result SHA-256 digest,
and omitted field/evidence/disagreement/question counts. Required lifecycle,
limitation, and review-link fields cannot be omitted. JSON/UTF-8 is never
sliced. Final answer and Vue disclose omissions.

## Cancellation and Finish-Existing

- Deterministic Go race tests are authored first.
- Cancel-first: transaction observes cancellation; zero job/binding rows.
- Admission-first: exactly one job/binding; later Copilot cancellation requests
  the existing Deep cancel path while nonterminal. `SettleTerminal` makes late
  provider/agent report/proposal settlement lose and late Copilot frames fail.
- New admission/rerun requires live
  `CoinCopilotAttributionEnabled`, `CoinCopilotEnabled`, model capability, and
  `DeepIdentificationEnabled`.
- Disable before admission: no work.
- Disable after admission: worker may finish; owner status/events/cancel,
  report/proposal review/edit, and confirmed existing-page apply remain.
  New/rerun is rejected.

## Compatibility and Rollback Interlock

Code inspection shows the current old handler serializes arbitrary
`job.Source`; current apply rejects an unknown source only incidentally through
target mismatch. Therefore the current binary is not an approved rollback
target.

### Required sequence

1. **Compatibility release first**: add strict known-source validation on
   list/get/status/stream/retry/apply/worker adoption and the default-off
   Feature 362 setting. At this point only `intake` and `saved_coin` are known.
2. Seed `copilot_draft`/arbitrary future source through raw SQL in tests and
   prove every adoption/read/apply path rejects without mutation.
3. Build/archive Release A and pass the required local or hosted guard and
   rollback evidence. CI or image publication alone is not deployment.
4. At an explicit external/manual checkpoint, obtain separate user approval
   to deploy Release A, deploy it everywhere, and record the deployed
   version/commit plus verification run URLs/ids and results.
5. Only after that recorded checkpoint, deploy additive handoff/source-draft
   migration and code recognizing
   `copilot_draft`, still default off.
6. Pass contract/migration/hosted mixed-binary tests, then enable.

### Rollback

1. Disable Feature 362/new handoffs.
2. Keep a compatible binary while accepted jobs finish or are cancelled.
3. Preserve all rows/report/proposal/artifacts.
4. Roll back no earlier than the guard release.
5. The guard binary must reject `copilot_draft` list/get/status/stream/retry/
   apply/worker adoption and leave rows intact.
6. Re-upgrade a compatible binary to restore review/apply.

The hosted executable matrix described in `quickstart.md` is mandatory.
Unavailable local tooling is not a waiver.

## Implementation Phases

### Phase 0 — compatibility guard and Release A checkpoint

- Record that the ADR 0017 acceptance gate is satisfied.
- Add tests first for unknown-source rejection across every Deep adoption/read/
  apply/worker path.
- Implement the guard and default-off Feature 362 setting with no new source
  rows.
- Build and archive the guard binary; pass the first half of the hosted
  rollback matrix.
- Stop at an explicit external/manual deployment checkpoint. Deployment
  requires separate user approval and cannot be inferred from CI success,
  artifact creation, registry/image publication, or merge. After approval,
  deploy Release A everywhere and record its deployed version/commit and
  verification evidence.

**Gate**: no schema, `source_draft_id`, `copilot_draft` recognition/row, or
other row-producing Feature 362 work until guard evidence passes and the
approved Release A deployment plus recorded verification is complete.

### Phase 1 — Race/idempotency tests before orchestration

- Add deterministic barrier/channel-based Go tests for cancel-first,
  admission-first, simultaneous duplicates, same-key changed target kind/id,
  changed prior rerun job id, changed stored app context, changed checkpoint,
  face/context/provider/draft snapshot changes, and crash/replay after job
  commit.
- Tests assert exact row counts, zero worker wake/provider calls on conflict,
  and no late terminal content after cancel.

**Gate**: tests exist and fail for the intended missing seam before production
orchestration is added.

### Phase 2 — Schema, contracts, and atomic Go admission

- Add `CoinCopilotDeepHandoff`, `source_draft_id`, `copilot_draft`, constraints/
  validation, migrations, and rollback-preservation tests.
- Add strict Go contract/projection types and 64 KiB/32 KiB enforcement.
- Implement the atomic snapshot/idempotency/reuse/create transaction and
  worker wake-after-commit.
- Register the exact execution-token callback only after service tests pass.

### Phase 3 — Result projection, eligibility, and cancellation

- Build only from validated persisted Deep report/proposal/coverage.
- Implement closed eligibility and non-disclosing outcomes.
- Wire admission-first Copilot cancellation to Deep cancellation and prove late
  settlement loses.
- Preserve active/completed/partial/no-match/conflict/image-only/provider
  attribution semantics.

### Phase 4 — Existing-draft atomic merge

- Route `copilot_draft` apply to `SourceDraftID`, never new-draft creation.
- Implement all-or-nothing current-state revalidation and exact merge matrix.
- Stage accepted draft references and integrate them into validated atomic
  promotion.
- Prove manual-field/reference preservation and replay idempotency.

### Phase 5 — Python and Vue

- Add strict Pydantic request/result mirrors, serial side-effect tool execution,
  checkpoint replay, proactive omission disclosure, and tamper tests.
- Add app-context draft hint without making it authoritative.
- Render status/result/link in the existing drawer; retain the existing Deep
  page and both independent reconnect loops.

### Phase 6 — Finish-existing, compatibility, and full gates

- Test every pre/post-accept flag transition and rerun rejection.
- Run hosted upgrade/rollback with guard/current binaries.
- Run all Go/Python/web build, lint, type/contract, test, security, provider,
  Fast Identify, legacy, direct Deep, and mobile/PWA gates from quickstart.

## Test Strategy

### Go

- Deterministic race tests precede implementation.
- Repository transaction tests: same key/same request, every changed binding,
  changed snapshot, active/terminal reuse, crash replay, exact row counts.
- Owner/eligibility tests: foreign/unknown equality, unbound intake, unknown
  source, deleted/promoted ids without a validated binding, and pruned events
  all prove the canonical byte-equivalent
  `{"outcome":"not_eligible","reason":null}` public body. Separate fixtures
  prove only a validated durable owner binding whose target later disappears/
  promotes returns `outcome=target_unavailable` and no target metadata.
- Contract tests: unknown fields/enums, user JWT vs execution token, wrong
  owner/run/execution/tool, expiry/revocation, 64 KiB boundaries, finite
  confidence, safe URLs, redaction.
- Projection tests: 32 KiB exact cap, deterministic stable omission, complete
  digest, byte counts, required lifecycle/link retention.
- Apply transaction tests for the complete field matrix, note block,
  case-insensitive ref append/dedupe, unsupported/stale rollback.
- Mixed-binary migration/rollback and finish-existing flag transitions.

### Python

- Strict args/results/checkpoint fixtures and unknown-field rejection.
- The tool client injects checkpoint/idempotency data; model cannot forge owner,
  snapshot, provider, or apply properties.
- No callback replay after checkpoint restore.
- Cancellation at each await, serial request/rerun, 64 KiB public frame, 32 KiB
  result disclosure, prompt-injection-as-data, and token redaction.
- Dependency install, compile, Pydantic type/contract tests, ruff, and pytest
  are mandatory.

### Vue

- Closed outcome and Deep-job-status cards, truncation disclosure, and safe
  relative link. The handoff result discriminant is always `outcome`.
- No editor/apply in chat.
- Existing capability fallback, cancel/resume/reconnect, Deep report/proposal/
  retry/reconnect, design tokens, keyboard/44 px controls, and narrow PWA layout.

### Blast-radius regressions

- Fast Identify unchanged.
- Direct Deep intake/saved coin, history, events, cancel/retry/review/apply.
- Coin Copilot default-off/unsupported legacy fallback, four collection
  callbacks, and specialist tools.
- Numista/Nomisma/OCRE bounds/attribution, NGC link-out, RPC unavailable.
- Collection/wishlist/draft manual fields, notes, images, and references.

## Mandatory Quality Evidence

Every command in `quickstart.md` must pass locally or in an equivalent hosted
job. The PR records hosted run URLs/ids for any gate not run locally. A note
that the environment lacked a tool is not passing evidence. Python evidence
must include dependency installation and syntax/build validation in addition
to lint/tests.

## Complexity Tracking

| Added complexity | Why needed | Simpler alternative rejected because |
|---|---|---|
| Durable handoff table | Crash-safe Go-owned idempotency and changed-binding 409 | Checkpoint-only persistence leaves a create-before-checkpoint crash window |
| `copilot_draft` + `source_draft_id` | Exact retained binding and existing-draft merge | Treating it as intake creates/targets the wrong draft and is unsafe on rollback |
| Cross-domain admission transaction | Linearizable snapshot/reuse/create/cancel | Separate checks permit stale launch, duplicates, and cancel races |
| Compatibility guard release | Verified fail-closed mixed-version rollback | Current old binary exposes arbitrary source on read/status paths |

These are not constitution violations; they are the minimum controls required
to satisfy Principles II, III, IV, V, and VIII.
