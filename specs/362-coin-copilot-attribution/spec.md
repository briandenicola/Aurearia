# Feature Specification: Coin Copilot Attribution Integration

**Feature Branch**: `beta` (existing working branch; no feature branch created)
**Created**: 2026-09-18
**Status**: Draft - Ready for Planning
**Input**: Promote backlog card F014 using the product-owner-selected
`copilot_integration` direction: add attribution and reference assistance to
Coin Copilot by reusing the existing Deep Analysis workflow.

## Scope Authority

Feature 362 is an **integration and workflow feature over shipped foundations,
not a replacement attribution system**. Features
[344](../344-deep-agentic-coin-identification/spec.md),
[351](../351-vision-first-deep-identification/spec.md), and
[352](../352-deep-identification-structured-results/spec.md) remain authoritative
for Deep Analysis jobs, image-first attribution, provider evidence, numeric
field confidence, contradictions, persisted proposals, structured references,
notes, and confirm-gated application. Features
[359](../359-coin-copilot-harness/spec.md) and
[361](../361-coin-copilot-specialist-tools/spec.md) remain authoritative for
Coin Copilot durability, owner scoping, bounded execution, replay,
cancellation, feature-flag rollout, and legacy fallback.

Feature 361 deliberately deferred Deep Identification handoff. Feature 362
authorizes only the smallest complete handoff: Coin Copilot may resolve an
owner-scoped target, reuse or launch the existing Deep Analysis capability,
explain its existing result, and direct the owner to the existing Deep Analysis
review surface. It does not authorize a second proposal model, provider
orchestrator, or mutation path.

The following accepted decisions are binding:

- [ADR 0010](../../docs/adr/0010-ocre-odbl-provider.md): OCRE automation remains
  bounded, default-off, and separately attributed under ODbL 1.0/ANS; no OCRE
  scraping, corpus, images, or arbitrary queries.
- [ADR 0012](../../docs/adr/0012-vision-first-deep-identification.md): images
  remain the primary identification source and providers fact-check, refine, or
  contradict the image hypothesis.
- [ADR 0013](../../docs/adr/0013-wishlist-coins-may-hold-catalog-references.md):
  wishlist items may hold validated references, but untrusted agent output may
  not bypass confirmation and reference validation.
- The Constitution's Principles II, III, IV, and V and §§0, 17, and 21:
  service boundaries, strict contracts, proportional reuse, owner isolation,
  and regression coverage remain mandatory.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Ask Copilot to attribute a specific coin (Priority: P1)

As a collector, I want to ask Coin Copilot to identify or attribute an existing
coin or an active intake draft so that I can begin the proven Deep Analysis
workflow without finding a separate entry point.

**Why this priority**: Target resolution and safe handoff are the minimum useful
integration. Without them, Coin Copilot cannot help with attribution.

**Independent Test**: Ask Coin Copilot to attribute one owned coin and one active
intake draft with sufficient face images. Verify that each request resolves one
strict owner-scoped target and returns or creates the equivalent existing Deep
Analysis job without invoking a second attribution pipeline.

**Acceptance Scenarios**:

1. **Given** an exact owned coin with sufficient obverse and reverse images,
   **When** its owner asks Coin Copilot to attribute it, **Then** Coin Copilot
   reuses an equivalent Deep Analysis job or launches one through the existing
   Deep Analysis workflow and identifies the selected target in its response.
2. **Given** an exact active intake draft with sufficient obverse and reverse
   images, **When** its owner asks for attribution, **Then** the same Deep
   Analysis capability is used with the draft's current image and note context.
3. **Given** the request could refer to multiple owned coins or drafts, **When**
   Coin Copilot cannot resolve one target safely, **Then** it asks an explicit
   clarification and creates no Deep Analysis job.
4. **Given** the target lacks a usable obverse, reverse, or both, **When** the
   request is evaluated, **Then** Coin Copilot states exactly what is missing,
   directs the owner to add or select the required images, and creates no job.
5. **Given** an unknown or foreign coin, draft, or job identifier, **When** the
   request is evaluated, **Then** it receives the same not-found behavior and
   reveals no target, image, job, or ownership information.

---

### User Story 2 - Understand and reopen Deep Analysis results (Priority: P1)

As a collector, I want Coin Copilot to explain the existing Deep Analysis
result conversationally and open its review screen so that I can understand the
evidence before deciding what to keep.

**Why this priority**: The integration is valuable only if it helps the owner
interpret and reach the authoritative result without weakening its evidence.

**Independent Test**: Use completed, partial, no-match, conflicting-source, and
low-confidence fixtures. Verify that Coin Copilot summarizes only the persisted
Deep Analysis report and proposal, preserves confidence and provenance, and
provides a link to the existing review surface.

**Acceptance Scenarios**:

1. **Given** a completed Deep Analysis job, **When** the owner asks what it
   found, **Then** Coin Copilot explains the proposed attribution, per-field
   confidence, supporting evidence, catalog references, unresolved questions,
   and provider coverage without changing the persisted result.
2. **Given** sources disagree, **When** the result is explained, **Then** each
   conflicting claim and its source remain visible and Coin Copilot does not
   silently choose a winner.
3. **Given** only image evidence, low-confidence evidence, partial provider
   coverage, or no reliable match, **When** the result is explained, **Then**
   the answer states those limits plainly and does not invent corroboration,
   references, or certainty.
4. **Given** a queued or running job, **When** the owner asks for its result,
   **Then** Coin Copilot reports the current durable state and links to the
   existing progress/review surface rather than starting a duplicate job.
5. **Given** a completed or partial result, **When** the owner follows the
   handoff, **Then** the existing Deep Analysis review surface opens for that
   exact owner-scoped job.

---

### User Story 3 - Keep every owner change review-gated (Priority: P1)

As a collector, I want Coin Copilot to remain read-only and send me through the
existing proposal editor before any change so that my manual cataloging is
never overwritten by a conversation.

**Why this priority**: Explicit per-field owner control is the safety boundary
that makes AI-assisted attribution acceptable.

**Independent Test**: Run the handoff against collection, wishlist, and intake
draft targets containing manual fields and references. Accept one proposed
field and reject the rest in the existing editor. Verify that only the accepted
field changes, existing references remain, and the conversation itself writes
nothing.

**Acceptance Scenarios**:

1. **Given** Coin Copilot has explained a proposal, **When** the owner does not
   enter and confirm the existing review/apply flow, **Then** no coin, wishlist
   item, or draft field or reference is created, updated, or deleted.
2. **Given** the owner opens the existing proposal editor, **When** one exact
   scalar field is accepted and all others are rejected, **Then** only that
   field is applied through the existing Deep Analysis write bridge.
3. **Given** a target already has manual scalar values, notes, images,
   acquisition/provenance data, valuation, storage, privacy, or status data,
   **When** unrelated proposal fields are accepted, **Then** those existing
   values remain unchanged.
4. **Given** a target already has structured references, **When** a proposed
   reference is accepted, **Then** it is validated and appended if new or
   deduplicated if equivalent; no existing reference is replaced or deleted.
5. **Given** a wishlist or intake-draft destination, **When** the proposal is
   reviewed, **Then** fields not valid for that destination are clearly
   inapplicable and cannot overwrite collection-only or manually maintained
   data.

---

### User Story 4 - Recover honestly from lifecycle and capability failures (Priority: P1)

As a collector, I want repeated, interrupted, failed, or unavailable requests
to produce a clear next step without duplicate work or misleading answers.

**Why this priority**: The integration crosses two durable workflows and must
preserve both systems' replay, cancellation, and fallback guarantees.

**Independent Test**: Exercise duplicate requests, replay after disconnect,
cancellation races, failed/cancelled/stale jobs, malformed integration output,
disabled capabilities, and unsupported models. Verify deterministic reuse or
fallback and no duplicate mutation or provider execution.

**Acceptance Scenarios**:

1. **Given** an equivalent Deep Analysis job is queued or running, **When** the
   request is repeated or replayed, **Then** the existing job is returned and no
   duplicate job or provider fan-out is created.
2. **Given** an equivalent completed or partial job whose target inputs still
   match and whose report is retained, **When** attribution is requested again,
   **Then** Coin Copilot reopens that result and offers an explicit rerun rather
   than silently spending another analysis.
3. **Given** the prior job failed, was cancelled, became stale, or no longer
   matches the target's current inputs, **When** the owner asks again, **Then**
   Coin Copilot identifies that state and offers the existing retry/new-analysis
   path; it does not present the prior attempt as a current successful result.
4. **Given** cancellation wins, **When** late integration output arrives,
   **Then** no late result is committed to the Coin Copilot run and no new Deep
   Analysis work begins.
5. **Given** Deep Analysis is disabled or unavailable, Coin Copilot is disabled
   or unsupported, or integration output is malformed, **When** attribution is
   requested, **Then** the existing safe fallback remains usable, the
   unavailable capability is explained, and no partial job, write, or invented
   result is produced.

### Edge Cases

- The prompt names a coin and a draft with similar labels but supplies no exact
  identifier.
- The app context identifies a target that differs from the identifier written
  in the prompt.
- A draft is discarded or promoted while target resolution is in progress.
- A coin is deleted or its face images change between target resolution and job
  launch.
- The obverse and reverse resolve to the same image, an unsupported image, or a
  missing file.
- An equivalent job is active but its event history has been pruned.
- A completed report remains retained while its live event history is no longer
  available.
- A completed proposal was already applied, or its linked coin was later
  deleted.
- A request is replayed with the same idempotency key but a different target or
  changed input fingerprint.
- Deep Analysis is disabled after a job starts.
- Coin Copilot is disabled after a durable run starts.
- Every automated provider returns no match; one provider times out; or sources
  contradict one another.
- A citation uses an unapproved host, unsafe scheme, embedded credentials, or a
  malformed URL.
- Tool output contains unknown fields, an unknown status, an out-of-range
  confidence value, an oversized payload, or prompt-injection text.
- Cancellation races job creation, provider completion, result explanation, or
  opening the review surface.

## Requirements *(mandatory)*

### Functional Requirements

#### Integration boundary

- **FR-001**: Coin Copilot MUST use the shipped Deep Analysis capability as the
  sole attribution engine. It MUST NOT create a parallel result model,
  confidence system, proposal store, image-analysis pipeline, provider router,
  contradiction evaluator, or reference parser (Features 344/351/352; ADR
  0012).
- **FR-002**: The integration MUST accept exactly two target classes: an
  existing owner-scoped coin (whether currently in the collection or wishlist)
  and an active owner-scoped intake draft. The target type and strict identifier
  MUST be resolved before job lookup or launch.
- **FR-003**: If target identity, image role, or user intent is materially
  ambiguous, Coin Copilot MUST pause for a concise clarification. It MUST NOT
  guess a target, infer a foreign identifier, or launch work before the
  clarification is resolved.
- **FR-004**: A target MUST provide a distinct usable obverse and reverse
  through its current owner-scoped images. Missing or invalid required images
  MUST produce a specific corrective message and no new job (Feature 344).
- **FR-005**: Starting Deep Analysis from an explicit owner request is the only
  new durable side effect authorized inside the Copilot handoff. Coin Copilot
  otherwise remains read-only and receives no coin, draft, reference, settings,
  arbitrary network, database, filesystem, shell, or code-execution authority
  (Features 359/361).

#### Reuse and lifecycle behavior

- **FR-006**: Equivalence MUST use the existing owner-bound Deep Analysis input
  identity, including target, current face-image content, bounded notes/context,
  and provider selection. A target identifier alone is insufficient.
- **FR-007**: An equivalent queued or running job MUST be reused. Duplicate or
  replayed Copilot requests MUST NOT create concurrent equivalent jobs or repeat
  provider orchestration (Feature 344).
- **FR-008**: An equivalent completed or partial job MAY be reused only when
  its report is retained and its target/input identity still matches. Coin
  Copilot MUST identify the result as existing and give the owner a distinct,
  explicit choice to request a fresh analysis.
- **FR-009**: Failed, cancelled, stale, expired, missing-result, or
  input-mismatched jobs MUST NOT be represented as successful current results.
  The integration MUST expose the existing retry or new-analysis action
  appropriate to the state while preserving prior history.
- **FR-010**: Coin Copilot and Deep Analysis cancellation, replay, stale
  recovery, terminal-state, and idempotency rules MUST remain authoritative in
  their respective durable workflows. A replay or resume MUST not repeat
  completed Deep Analysis work solely to reconstruct conversation state.

#### Evidence and conversational explanation

- **FR-011**: Coin Copilot MAY explain only the persisted, validated Deep
  Analysis report, proposal, coverage, and lifecycle state. It MUST NOT rewrite
  the report as a new authoritative result or claim to have queried providers
  itself.
- **FR-012**: Explanations MUST preserve numeric confidence in the existing
  0.0–1.0 scale, field provenance, validated citations, source conflicts,
  unresolved questions, provider statuses, partial-success state, and no-match
  state (Features 344/351/352).
- **FR-013**: Low confidence, image-only support, conflicting claims, failed or
  unavailable providers, and no reliable match MUST be stated plainly.
  Unsupported values, citations, reference numbers, or agreement MUST not be
  invented.
- **FR-014**: Citation URLs MUST pass the existing Deep Analysis source-host
  validation before Coin Copilot displays or links them. Invalid citations MUST
  be omitted with an honest limitation rather than repaired or replaced.
- **FR-015**: Automated sources MUST remain Numista, Nomisma, and enabled OCRE.
  NGC remains official link-out/quick evidence, and RPC automation remains
  paused/unavailable. Existing provider flags, bounds, URL allowlists, and
  license/attribution text MUST be preserved; no scraping or unsupported source
  is permitted (PRD §5.3; ADR 0010).
- **FR-016**: Every accepted or reused job response MUST provide an owner-scoped
  handoff to the existing Deep Analysis progress/review surface for the exact
  job. Coin Copilot MUST NOT embed a second proposal editor.

#### Review-gated application and field preservation

- **FR-017**: Conversation, Copilot tools, and Copilot final answers MUST have
  no direct apply operation. Every coin, wishlist, draft, note, and structured
  reference mutation MUST remain behind the existing Deep Proposal editor and
  write bridge (Features 344/352).
- **FR-018**: No target field may change unless the owner individually accepts
  that exact proposal field and explicitly confirms apply. Rejecting, leaving
  undecided, explaining, opening, cancelling, or replaying MUST write nothing.
- **FR-019**: Applying one accepted field MUST preserve every unaccepted scalar
  field and all manually entered images, acquisition/provenance fields,
  valuation fields, storage fields, privacy/status fields, and relationships.
- **FR-020**: Structured catalog references MUST use the existing validated
  additive/deduplicating path. They MUST never use destructive reference
  replacement, and replaying an equivalent accepted reference MUST leave one
  equivalent reference without deleting any existing row (Feature 352 FR-013/14
  and ADR 0013).
- **FR-021**: Destination applicability MUST come from the existing Deep
  Proposal allowlists and MUST NOT be widened by Coin Copilot. Collection
  targets may review existing collection-valid proposal fields; wishlist
  targets may receive only existing wishlist-valid fields; active drafts may
  receive only existing draft-valid fields and staged catalog references.
  Inapplicable fields MUST be shown as unavailable or omitted from apply, never
  coerced into another field.
- **FR-022**: A wishlist or draft handoff MUST NOT set or overwrite
  collection-only/manual data such as acquisition facts, valuation, storage,
  ownership status, privacy, or images. Wishlist references remain valid only
  through the existing confirmed, validated path (ADR 0013).

#### Safety, bounds, compatibility, and tests

- **FR-023**: All target, job, report, and proposal reads MUST be scoped to the
  authenticated owner derived server-side. Foreign and unknown identifiers
  MUST be indistinguishable and disclose no existence or metadata.
- **FR-024**: The handoff contract and all returned lifecycle/result data MUST
  be strictly typed, reject unknown critical fields and invalid state values,
  validate confidence ranges and URLs, treat text as untrusted data, and fail
  closed on malformed output.
- **FR-025**: The integration MUST inherit Coin Copilot's existing snapshotted
  iteration, tool-call, concurrency, wall-clock, token-observation, credential,
  event, checkpoint, and payload bounds. Deep Analysis MUST retain its own
  existing job/provider/image/note/retention bounds; nesting MUST NOT create an
  unbounded combined budget.
- **FR-026**: The current default-off Coin Copilot flag, model-capability check,
  pre-accept legacy-chat fallback, and safe behavior when Deep Analysis is
  disabled or unavailable MUST remain intact (Features 359/361).
- **FR-027**: The existing fast Identify flow MUST remain available and
  unchanged in behavior, contract, and independence. A Copilot attribution
  request MUST NOT replace, delay, or implicitly invoke the fast flow.
- **FR-028**: No direct browser-to-agent-service call, agent-owned database
  state, new external dependency, new provider API, generic web browsing, or
  additional outbound-source permission may be introduced.
- **FR-029**: Automated tests MUST cover low confidence, conflicting sources,
  no match, manual-field preservation, wishlist-versus-collection-versus-draft
  field applicability, reference append/dedupe, duplicate and replayed
  requests, foreign and unknown target IDs, malformed output, cancellation
  races, completed/active/failed/cancelled/stale jobs, changed inputs,
  missing images, and safe capability fallback.
- **FR-030**: Regression tests MUST prove that existing Coin Copilot collection
  and specialist tools, legacy fallback, Deep Analysis direct entry points,
  proposal review/apply, provider attribution, and fast Identify behavior
  remain unchanged outside this handoff.

### Key Entities

- **Attribution Handoff**: A bounded Coin Copilot action containing the
  authenticated owner, one strict target type/id, the matching Deep Analysis
  input identity, and the resulting existing or newly accepted job id. It grants
  no apply authority.
- **Attribution Target**: An owner-scoped existing coin or active intake draft
  with current face-image roles and bounded note context.
- **Deep Analysis Job**: The existing durable job and sole authority for
  attribution lifecycle, provider work, evidence, report, proposal,
  cancellation, retry, and progress replay.
- **Deep Analysis Result Summary**: A bounded conversational projection of the
  existing persisted report/proposal, retaining confidence, provenance,
  conflicts, coverage, limitations, and a review link; it is not a new result
  record.
- **Deep Proposal**: The existing persisted, per-field review document. It
  remains the only source of accepted fields and the only route into the
  existing apply bridge.

## Non-Goals

- A new or replacement attribution engine, confidence model, proposal schema,
  provider router, evidence store, contradiction evaluator, or reference parser.
- Provider expansion, generic browsing, new provider APIs, scraping, RPC
  automation, OCRE images/corpus, or changing NGC beyond official link-out and
  quick evidence.
- Direct Coin Copilot mutation, conversational acceptance, agent-direct apply,
  or a second proposal/review UI.
- Bulk attribution, scheduled attribution, background processing of an owner's
  whole collection, or cross-coin auto-application.
- Changing fast Identify, existing Deep Analysis direct entry points, current
  feature flags, legacy-chat fallback, or provider-license rules.
- New long-term Copilot memory or preferences.
- F015 curator, watchlist, acquisition-goal, provenance-risk, suspicious-listing,
  duplicate-image, or documentation-gap behavior.
- Plan, task, implementation, migration, dependency, deployment, or operational
  rollout work in this specification stage.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: In acceptance tests, 100% of attribution requests resolve exactly
  one owner-scoped coin or active draft, or ask for clarification without
  starting work.
- **SC-002**: Repeated and replayed equivalent requests create zero duplicate
  active Deep Analysis jobs and repeat zero completed provider work solely for
  conversation recovery.
- **SC-003**: 100% of completed, partial, conflicting, low-confidence, and
  no-match fixture explanations preserve the source result's confidence,
  provenance, conflicts, provider limitations, and validated links with zero
  invented claims.
- **SC-004**: Across collection, wishlist, and draft acceptance tests, zero
  fields or references change before explicit review confirmation, and only
  individually accepted destination-valid fields change afterward.
- **SC-005**: In preservation tests, 100% of unaccepted manual fields remain
  byte-for-byte unchanged, all pre-existing structured references remain
  present, and an equivalent accepted reference appears at most once.
- **SC-006**: Unknown and foreign target/job tests disclose zero target, image,
  report, proposal, job-state, or ownership information.
- **SC-007**: Missing-image, malformed-output, disabled-capability, unsupported-
  model, failed, cancelled, and stale-state tests produce a clear safe next step
  with zero invented result, partial write, or orphaned job.
- **SC-008**: Every accepted or reused attribution handoff gives the owner a
  working path to the exact existing Deep Analysis progress/review surface.
- **SC-009**: Existing fast Identify and legacy fallback regression suites show
  no contract or behavior change attributable to Feature 362.
- **SC-010**: No accepted test run exceeds the pre-existing Coin Copilot or Deep
  Analysis limits for duration, tool calls, concurrency, payload size, provider
  calls, image count/size, or retained result data.

## Assumptions

- Features 344, 351, 352, 359, and 361 are the shipped foundations; Feature 362
  integrates them rather than restating their delivered behavior as new work.
- “Active intake draft” means an owner-scoped, non-discarded, non-promoted draft
  that still satisfies the existing Deep Analysis image requirements.
- A completed or partial result is reusable only while its authoritative report
  remains retained and its input identity still matches the target's current
  image/context snapshot.
- Starting or reusing a Deep Analysis job is a workflow handoff, not permission
  to mutate the target. The user's request to attribute is explicit launch
  intent; all target writes still require a second explicit action in the
  existing review/apply flow.
- Existing Deep Analysis and Coin Copilot retention windows, feature settings,
  provider availability, and per-run limits remain authoritative and are not
  retuned by this feature.
- There are no unresolved product clarifications for specification-stage
  planning.
