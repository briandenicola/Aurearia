# Implementation Plan: Coin Copilot Attribution Integration

**Branch**: `beta` (existing; no create/switch) | **Date**: 2026-09-18 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `specs/362-coin-copilot-attribution/spec.md`

## Summary

Integrate Coin Copilot with the shipped Deep Analysis workflow through one
fixed, typed, Go-owned `deep_analysis_handoff` capability. Go resolves the exact
owned coin or active Quick Capture draft, snapshots current input, reuses or
admits the existing durable Deep job, and projects persisted results for
conversation. Python remains stateless and database-free. Vue links from the
existing Agent drawer to the existing `/deep-analysis/:jobId` review page; all
apply operations remain exclusively in
`DeepIdentificationProposalService.Apply`.

## Technical Context

**Language/Version**: Go 1.26.1; Python 3.12; TypeScript/Vue 3
**Primary Dependencies**: Gin, GORM/SQLite; FastAPI, Pydantic, LangGraph/LangChain; Vue Router, Pinia, Vite/PWA
**Storage**: Existing SQLite Deep Identification and Coin Copilot tables; one nullable `source_draft_id` column on `deep_identification_jobs`; existing JSON checkpoints/events/reports/proposals
**Testing**: Go `testing`/integration tests and `go vet`; Python pytest/ruff; Vitest/Vue Test Utils and `vue-tsc --build` via `npm run build`
**Target Platform**: Self-hosted single-node Docker deployment; desktop and mobile/PWA browsers
**Project Type**: Three-layer web application (Go API + Vue SPA + stateless Python agent)
**Performance Goals**: No extra provider fan-out for equivalent/replayed requests; tool response remains within existing persisted-result bounds; normal chat and Deep streams start within existing PRD targets
**Constraints**: Default-off flags; owner isolation; no Python DB/filesystem/generic HTTP/apply; fixed callback routes; independent bounded Copilot and Deep budgets; no new provider/dependency; current Fast Identify and legacy fallback unchanged
**Scale/Scope**: Personal self-hosted deployment, fewer than 10 concurrent users; one new capability spanning existing Go/Python/Vue seams

There are no unresolved `NEEDS CLARIFICATION` items.

## Constitution Check

*GATE: Passed before Phase 0; rechecked after Phase 1 design.*

| Authority/gate | Design response | Result |
|---|---|---|
| §0 hierarchy | Constitution, PRD §5.3, Feature 362, Features 344/351/352/359/361, ADRs 0010-0013, and accepted decisions were read in order. | PASS |
| Principle I, layered Go | New public/internal parsing stays in a thin handler; target/job business rules live in an HTTP-agnostic service; owner-scoped queries remain repositories; wiring is constructor-injected. | PASS |
| Principle II, service boundaries | Go owns persistence, credentials, provider selection, job admission, and writes. Python receives one request and returns typed frames with no DB or browser-direct path. Vue calls Go only. | PASS |
| Principle III, strict contracts | Go structs and `DisallowUnknownFields`, Pydantic `extra="forbid"`, TS types, Swagger/OpenAPI, enum/range/URL validation, and cross-language fixtures cover the capability. | PASS |
| Principle IV, proportional reuse | One capability reuses the shipped job/report/proposal/review paths. No engine, provider router, proposal model, editor, or dependency is duplicated. | PASS |
| Principle V, auth/privacy | Owner id comes only from the execution token. Foreign/unknown ids are indistinguishable. Callback auth, body caps, redaction, source URL validation, and fail-closed output remain mandatory. | PASS |
| Principle VI, UX/PWA | Existing drawer and Deep Analysis page are reused with design tokens, Lucide icons, mobile layout, touch targets, and independent reconnect behavior. | PASS |
| Principles VII/IX and §§17/21 | Plan includes full build/lint/type/test gates, exact-path regression tests, contract drift tests, tamper tests, and blast-radius coverage. | PASS |
| Principle VIII | Existing ADRs govern the architecture/provider choices. The minimum additive draft-binding decision is documented here/research; implementation should add an ADR amendment only if maintainers classify it as a semantic migration under §21. | PASS |

### Post-design re-evaluation

PASS. Phase 1 introduces no boundary violation. The nullable draft binding is
the only schema addition and is required to prevent an untyped/prunable target
association; run-to-job linkage remains in the existing checkpoint payload.

## Project Structure

### Documentation (this feature)

```text
specs/362-coin-copilot-attribution/
├── spec.md
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
└── contracts/
    └── coin-copilot-deep-analysis.md
```

No `tasks.md` is created by this planning work.

### Source Code (repository root)

```text
src/api/
├── deps.go
├── routes_internal.go
├── routes_protected.go
├── database/database.go
├── models/
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
│   └── quick_capture_service.go
├── handlers/
│   ├── coin_copilot_internal_tools.go
│   └── deep_identification.go
└── integration/

src/agent/
├── app/models/requests.py
├── app/models/responses.py
├── app/tools/copilot_collection_tools.py
├── app/teams/coin_copilot.py
└── tests/

src/web/
├── src/api/endpoints/agent.ts
├── src/types/agent.ts
├── src/composables/
│   ├── useCoinCopilot.ts
│   └── useCoinSearchChat.ts
├── src/components/
│   ├── CoinSearchChat.vue
│   └── chat/CopilotRunProgress.vue
├── src/pages/DeepAnalysisPage.vue
├── src/router/index.ts
└── src/**/__tests__/
```

**Structure Decision**: Extend the existing three-service layout at its current
composition points. Do not add a project, service, page, provider module, or
generic tool router.

## Existing Symbol Reuse Map

### Go authority

| Existing symbol | Reuse |
|---|---|
| `services.CoinCopilotService.AuthorizeToolCall` / `FinishToolCall` | Preserve current execution, duplicate call, concurrency, timeout, and tool-budget enforcement; extend for the one serialized side-effect admission. |
| `services.CoinCopilotAllowedTools`, `copilotCallbackTools` | Add exactly `deep_analysis_handoff`; no wildcard or route-prefix widening. |
| `middleware.CoinCopilotExecutionTokenRequired` in `routes_internal.go` | Bind route to the exact capability and owner/execution claims. |
| `services.DeepIdentificationService.CreateJobFromIntake`, `StartJob` | Sole job admission/active reuse path. |
| `services.ComputeInputFingerprint` and `DeepIdentificationJob.ActiveKey` | Equivalent input identity and concurrent duplicate prevention. |
| `repository.DeepIdentificationRepository.FindActiveByFingerprint`, `CreateJob` | Active reuse; add latest matching retained-terminal lookup. |
| `services.DeepIdentificationService.GetJob`, `ListEventsSince`, `RequestCancel`, `RetryJob` | Preserve owner-scoped lifecycle/status/replay/cancel semantics; status does not restart work. |
| `repository.DeepIdentificationRepository.SettleTerminal` | Preserve terminal/cancel race and exactly-one terminal event. |
| `QuickCaptureRepository.GetDraftForOwner` and `QuickCaptureDraftStatusActive` | Authoritative draft ownership/liveness. |
| `ImageRepository.FindCoinByOwner` / face-image lookup and `DeepIdentificationService` artifact validation | Current saved-coin face resolution, MIME/content hashes, and bounded artifacts. |
| `DeepIdentificationProposalService.UpdateProposal` / `Apply` | Only review/apply boundary; retain scalar allowlists, `catalogReferences` validation, `CoinReferenceService.AppendForCoin`, and manual-field preservation. |
| `toDeepJobEnvelope`/Deep report contract validation | Source material for the bounded conversation projection; do not expose raw storage or apply state. |

### Python stateless adapter

| Existing symbol | Reuse |
|---|---|
| `COPILOT_ALLOWED_TOOLS`, `CopilotExecuteRequest`, `CopilotCompletedTool` | Add one literal and strict handoff result model; replay remains checkpoint-based. |
| `CALLBACK_TOOLS`, `ARG_MODELS`, `RESULT_MODELS` | Register one fixed callback with mutual-field validation. |
| `CopilotCollectionToolClient.execute` / `_execute_callback` | Reuse execution token, route construction, result bounding, digest, sanitization, and completed-call dedupe. |
| `build_copilot_tool_definitions` | Bind the one typed schema to supported models. |
| `run_coin_copilot` | Preserve iteration/tool/concurrency/wall-clock budgets and cancellation checks; handoff runs alone rather than in a parallel group because it may admit work. |
| `COPILOT_SYSTEM_PROMPT`, `_TOOL_LABELS`, `_TOOL_SUMMARIES` | Authorize only request/status/rerun, require clarification first, forbid apply, and describe persisted Deep output honestly. |

### Vue handoff and review

| Existing symbol | Reuse |
|---|---|
| `useCoinSearchChat.buildAppContext` / `AgentChatAppContext` | Continue route/coin hints; add bounded `activeDraftId` only on the exact draft route. |
| `useCoinCopilot` | Existing capability fallback, idempotency, SSE replay, cancel/resume, and checkpoint handling. |
| `CopilotRunProgress.vue` | Render a compact typed handoff status/result card and router link; no editor. |
| `/deep-analysis/:jobId` in `router/index.ts` | Canonical relative review URL, unchanged. |
| `DeepAnalysisPage.vue` | Existing report, coverage, conflicts, proposal editor, apply, retry, cancel, and independent reconnect. |
| `useDeepAnalysisLauncher` and existing direct entry components | Regression baseline; direct Deep Analysis remains unchanged. |

## Design and Implementation Phases

### Phase 1: Strict contracts and canonical fixtures

1. Define the Go/Pydantic/TypeScript request/result mirrors from
   `contracts/coin-copilot-deep-analysis.md`.
2. Add valid and tampered fixtures shared by contract-drift tests.
3. Add the tool literal without registering a route or model behavior until all
   validators fail closed.
4. Update Swagger/OpenAPI for any modified public Coin Copilot event/result
   projection; the callback itself remains internal.

**Gate**: unknown fields/states/providers, unsafe URLs, invalid confidence,
oversize, and apply-like fields all fail closed.

### Phase 2: Go-owned target and Deep job orchestration

1. Add `SourceDraftID` and additive migration/index coverage.
2. Add an HTTP-agnostic attribution handoff service injected with existing
   Coin Copilot, Deep Identification, image, settings, and Quick Capture
   boundaries.
3. Snapshot owned coin or active draft input, validate distinct usable faces,
   compute v2 identity, reuse active/retained current jobs, or admit through
   `CreateJobFromIntake`.
4. Add latest owner/fingerprint retained-terminal repository lookup.
5. Add per-execution cancellation/admission serialization; never compensate
   after work starts as a substitute for ordering.
6. Register only the exact callback route and handler.

**Gate**: concurrent duplicates create one active Deep job; cancel-first creates
zero; foreign/unknown resources disclose nothing.

### Phase 3: Persisted result projection and proposal binding

1. Decode and validate only persisted Deep report/proposal data.
2. Produce the bounded conversational projection with existing source-host
   validation and provider/license vocabulary.
3. Preserve no-match, image-only, low-confidence, conflict, partial, pruned-
   events, failed/cancelled/stale, and missing-result states.
4. For draft-origin jobs, make existing proposal apply target the exact bound,
   still-active draft. Reuse destination allowlists and validated additive
   references; reject inapplicable fields and changed/inactive targets.

**Gate**: no conversation path writes a target; only exact confirmed review
fields change.

### Phase 4: Python orchestration and replay

1. Add strict Pydantic args/results and the one callback registration.
2. Update prompt policy to clarify ambiguous targets, use `status` for known
   jobs, reuse current results, and require explicit rerun intent.
3. Execute the side-effect-capable handoff in its own group while retaining all
   existing budgets and cancellation checks.
4. Persist/replay the result through current completed-tool checkpoints; do not
   call Go again during resume reconstruction.

**Gate**: malformed callback/model output is `invalid_tool_call`; unsupported
models and pre-accept failures keep legacy fallback.

### Phase 5: Vue conversation handoff

1. Extend app context for the exact active draft route without making it
   authoritative.
2. Render lifecycle/result limitations and `Open Deep Analysis` on the existing
   Copilot progress surface.
3. Validate/construct the relative route from numeric job id.
4. Keep report/proposal editing solely on `DeepAnalysisPage`.

**Gate**: mobile/PWA layout, keyboard/touch behavior, both SSE reconnect loops,
and design tokens remain intact.

### Phase 6: Regression, security, and documentation

Run the contract, tamper, race, owner-isolation, lifecycle, provider, apply
preservation, full Go/Python/Vue, architecture, OpenAPI drift, and build gates
listed in `quickstart.md`. Document the capability while explicitly preserving
Fast Identify, direct Deep Analysis, current collection/specialist tools,
legacy chat, flags, and provider boundaries.

## Error, Status, URL, and Retry Behavior

The exact cross-layer vocabulary is normative in
`contracts/coin-copilot-deep-analysis.md`.

- Owner/domain conditions are typed tool outcomes, not invented prose.
- Foreign and missing target/job ids both produce `not_found`.
- Queue/capacity/disabled states are `unavailable` with bounded reasons.
- Unknown enums, malformed JSON, unsafe URLs, invalid confidence, or oversize
  fail the tool/run closed.
- Review URL is relative `/deep-analysis/{positive job id}` only.
- Copilot reconnect uses its run sequence and checkpoint; Deep Analysis
  reconnect uses its independent job sequence.
- Active equivalent work is reused. Retained equivalent terminal work is
  reopened. Failed/cancelled/stale/mismatched work requires an explicit new
  request. Replay never repeats provider work.

## Test Strategy

### Go

- Service/repository tests for v2 fingerprint identity, latest terminal reuse,
  active uniqueness, image/note/provider changes, draft liveness, and nullable
  migration rollback behavior.
- Handler/contract tests for exact route authorization, unknown fields, body
  caps, foreign/unknown equality, status vocabulary, URLs, confidence, and
  redaction.
- Deterministic race tests for cancel-before-admission, admission-before-cancel,
  duplicate requests, replay, and late frames.
- Proposal integration tests for collection, wishlist, and bound active draft:
  one accepted field changes; all unaccepted/manual fields remain byte-for-byte;
  references append/dedupe and never replace; inapplicable fields fail closed.

### Python

- Pydantic valid/invalid mirrored fixtures and completed-tool replay tests.
- Harness tests for ambiguity clarification, prompt/context disagreement,
  request/status/rerun selection, serial execution, result truncation,
  cancellation after each await, and no repeated callback on resume.
- Security tests treating report/provider text as untrusted, rejecting prompt
  injection, secrets, unknown fields, forged apply operations, and unsafe URLs.

### Vue

- Typed handoff card/status/link rendering for queued/running/completed/partial/
  no-match/failed/cancelled/stale/unavailable.
- Safe route tests reject absolute, mismatched, or credential-bearing URLs.
- Existing Coin Copilot fallback/cancel/resume/reconnect and Deep Analysis
  report/proposal/retry/reconnect suites remain green.
- Responsive tests assert no duplicate editor, design-system controls, keyboard
  access, 44 px touch targets, and narrow viewport containment.

### Blast-radius regressions

- Current four collection callbacks and bounded specialist tools.
- `CoinCopilotEnabled` default-off and unsupported-model legacy fallback.
- Direct Deep Analysis launch/history/status/retry/cancel/review/apply.
- Fast Identify contract and entry points.
- Numista/Nomisma/OCRE automation bounds and attribution; NGC link-out; RPC
  unavailable.
- Collection/wishlist/draft field applicability and manual field/reference
  preservation.

## Schema and Rollback Decision

No Coin Copilot run/job linkage schema is required: the existing checkpoint
completed-tool result is safe and replayable. One additive Deep job
`source_draft_id` is required because no existing retained Deep-owned structure
can bind a job to its source active draft after events are pruned. The migration
is nullable, has no backfill, and is safe to leave on rollback. Feature flags
remain kill switches; old jobs and direct paths are unchanged.

## Complexity Tracking

No Constitution violation or waiver is required, so the template's violation
table is intentionally empty. The only added complexity—one typed callback and
one nullable draft-binding column—is justified by the safety boundaries above;
generic tools, duplicate UI, a new engine, and schema-heavy linkage were
rejected.
