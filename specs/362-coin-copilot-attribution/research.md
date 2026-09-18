# Research: Coin Copilot Attribution Integration

## Decision 1: Use one fixed Go-owned Copilot capability

- **Decision**: Add `deep_analysis_handoff` as one explicitly allowlisted Coin
  Copilot callback tool. Its three closed operations are `request`, `status`,
  and `rerun`. The Go handler delegates to a new HTTP-agnostic orchestration
  service, which in turn calls the existing `DeepIdentificationService`; Python
  remains a strict, stateless adapter.
- **Rationale**: This is the smallest seam that lets a conversation request a
  durable side effect without giving Python generic REST, database, filesystem,
  provider, or apply authority. It preserves the existing
  `CoinCopilotExecutionTokenRequired` route model, `AuthorizeToolCall` budget
  checks, Deep Analysis workers, and owner-scoped repositories.
- **Alternatives considered**:
  - Let Python call public Deep Analysis endpoints: rejected because it would
    require a user credential and bypass the fixed callback allowlist.
  - Add separate resolve/start/result/apply tools: rejected because it expands
    authority and permits unsafe composition; apply is expressly forbidden.
  - Implement attribution in Coin Copilot: rejected as a second engine.

## Decision 2: Resolve a strict target before invoking the capability

- **Decision**: The tool accepts exactly one typed target,
  `{type: "coin"|"draft", id: positive integer}`. App context may suggest
  `activeCoinId` or a new `activeDraftId`, but Go re-resolves ownership and
  current state. A name-only or conflicting prompt/context must produce the
  existing Coin Copilot clarification checkpoint without calling the tool.
- **Rationale**: The current app context is explicitly non-authoritative.
  `ImageRepository.FindCoinByOwner`, `QuickCaptureRepository.GetDraftForOwner`,
  and the active draft status supply the authoritative checks. Unknown and
  foreign identifiers therefore share one not-found result.
- **Alternatives considered**:
  - Fuzzy Go-side target selection: rejected because it can silently select the
    wrong coin or draft.
  - Trust route parameters or model-selected labels: rejected because neither
    proves ownership or current state.

## Decision 3: Reuse the Deep Analysis input identity

- **Decision**: Build the request from the target's current distinct obverse and
  reverse image content, bounded current notes/context, and the current
  Go-selected provider set; then use the existing
  `ComputeInputFingerprint`, `CreateJobFromIntake`, active unique key, worker
  queue, and provider budgets. Add a repository query for the latest terminal
  job with the same owner/fingerprint so a retained `completed` or `partial`
  result can be reopened without spending another run. `rerun` is explicit.
- **Rationale**: `DeepIdentificationRepository.CreateJob` already serializes
  equivalent active job creation by `(user_id, input_fingerprint, active_key)`.
  A target id alone cannot detect changed images, notes, or provider selection.
- **Alternatives considered**:
  - Key reuse only by target id: rejected because it returns stale evidence.
  - Always call `RetryJob`: rejected because it spends work even when an
    equivalent active or retained result exists.
  - Re-run on Copilot replay: rejected because checkpoint replay already
    contains the completed handoff result.

## Decision 4: Persist run-to-job linkage in the existing checkpoint

- **Decision**: The validated `deep_analysis_handoff` result is stored as the
  existing `CoinCopilotCheckpoint.completed_tools[].result`, including the
  target reference, Deep job id, input fingerprint digest, outcome, status, and
  canonical relative review URL. No Coin Copilot linkage table or columns are
  added.
- **Rationale**: Completed tool facts are already bounded, digested,
  replay-safe, owner-bound through the run, and restored without executing the
  tool again. This is exactly the current durable payload intended for a tool
  fact.
- **Alternatives considered**:
  - Add `deep_job_id` to `coin_copilot_runs`: rejected because one thread/run
    can discuss multiple jobs and the checkpoint already represents the
    sequence.
  - Infer linkage from prose or events: rejected because prose is untyped and
    events may be pruned.

## Decision 5: Add only the missing durable draft binding

- **Decision**: Add nullable `SourceDraftID *uint` (`source_draft_id`) to
  `DeepIdentificationJob`. Keep `Source="intake"` for compatibility. A
  draft-origin job snapshots the active owned draft's images and bounded
  context into the normal Deep artifacts, and the existing proposal service
  uses `SourceDraftID` to apply only destination-valid accepted fields back to
  that exact still-active draft.
- **Rationale**: Current durable payloads can safely represent Copilot
  run-to-job linkage, but none safely binds a Deep job to the active draft from
  which it was created. CoinCopilot checkpoints are not authoritative to the
  proposal service, Deep events can be pruned, and `ReportJSON`/`ProposalJSON`
  are produced only at settlement. Reusing `CoinID`, `AppliedDraftID`, notes,
  or provider columns would corrupt existing semantics. One nullable column is
  the minimum honest migration.
- **Alternatives considered**:
  - Put draft id in notes: rejected as untyped, user-visible metadata.
  - Put draft id only in an event: rejected because retained reports outlive
    event history.
  - Pass a draft id only at apply time: rejected because an owner could attach
    unrelated analysis to another draft.
  - Create a second draft on apply: rejected because it duplicates the active
    target instead of preserving the requested workflow.

## Decision 6: Return a bounded conversational projection

- **Decision**: Go decodes and validates the persisted report/proposal into a
  purpose-built, non-persisted projection: lifecycle state, proposed fields
  with numeric confidence, evidence and validated citations, disagreements,
  unresolved questions, provider coverage/attribution, limitations, and
  `/deep-analysis/{jobId}`. Python mirrors this with strict Pydantic models and
  treats all text as untrusted data.
- **Rationale**: Returning raw stored JSON would make contract drift and unsafe
  links harder to reject. The projection exposes enough to explain the result
  without becoming a second result model or proposal authority.
- **Alternatives considered**:
  - Return only status and URL: rejected because it cannot support grounded
    conversational explanation.
  - Return the entire proposal editor document: rejected because owner edit and
    acceptance fields are unnecessary in chat and could imply apply authority.
  - Ask providers again from Copilot: rejected because persisted Deep Analysis
    is the sole source.

## Decision 7: Linearize cancellation against the one allowed side effect

- **Decision**: Serialize `deep_analysis_handoff` request/rerun admission and
  `CoinCopilotService.Cancel` for the same execution. Re-check that the current
  execution is active immediately before Deep job admission. If cancellation
  linearizes first, return `cancelled` and create no job; if admission
  linearizes first, the accepted Deep job remains an independent durable
  workflow. Late Python frames remain rejected by the existing current-
  execution/state checks.
- **Rationale**: An authorization check followed by an uncoordinated job create
  leaves a race in which cancellation can win in one workflow while work starts
  in the other. A per-execution Go critical section resolves the race without a
  cross-service transaction or schema.
- **Alternatives considered**:
  - Best-effort post-create cancellation: rejected because work may already
    start.
  - Couple later Copilot cancellation to an already accepted Deep job:
    rejected because the two durable workflows have separate authoritative
    cancellation controls.

## Decision 8: Preserve providers, flags, and existing UI

- **Decision**: The capability is available only when both
  `CoinCopilotEnabled`/model capability and `DeepIdentificationEnabled` allow
  it. Provider selection stays Go-owned: Numista and Nomisma automated, OCRE
  only when enabled, NGC link-out/quick evidence, RPC unavailable. Vue renders a
  typed handoff card/link inside the existing drawer and navigates to the
  existing `DeepAnalysisPage`; it adds no proposal editor.
- **Rationale**: This preserves default-off rollout, legacy fallback, Fast
  Identify independence, provider licensing, and the existing mobile/PWA
  progress/reconnect/review behavior.
- **Alternatives considered**:
  - A new Copilot attribution page/editor: rejected as duplicate UI.
  - Widen callback hosts/routes or add a provider: rejected by the feature and
    ADRs 0010-0012.

## Clarification resolution

All planning unknowns are resolved. There are no remaining `NEEDS
CLARIFICATION` items.
