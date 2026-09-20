# Feature Specification: AI-Driven Browser Testing

**Feature Branch**: `beta` (active feature artifacts: `360-ai-driven-browser-testing`)
**Created**: 2026-09-18
**Status**: Draft
**Input**: Backlog card F011 plus product-owner decisions approved 2026-09-18

## Scope Authority

Under Constitution §0, this active specification supersedes F011 wherever the
backlog card left an approach or operating choice open. The feature adds a
bounded, provider-agnostic exploratory browser-testing tier that exercises an
ephemeral full application stack using synthetic data and dedicated test
credentials. It reuses Feature 220/F013's golden fixtures and critical workflow
inventory, produces evidence-first reports, and remains advisory while its
reliability is established.

This feature follows Principle IV by adding one complete exploratory workflow
around the existing browser-testing foundation rather than replacing or
duplicating the deterministic critical workflow suite.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Explore critical workflows safely (Priority: P1)

As a maintainer, I want an AI-guided browser explorer to exercise the
application's critical user workflows in an isolated environment so runtime
UI defects can be found without exposing or changing production data.

**Why this priority**: Safe, useful exploration of real user journeys is the
core value of the feature; reporting and automation are valuable only if the
exploration itself is isolated, bounded, and repeatable.

**Independent Test**: Start an ephemeral full stack with the F013 golden
fixtures, run the explorer with dedicated test credentials, and verify it
visits the selected critical workflows, stops within every configured limit,
and leaves no dependency on or change to a production environment.

**Acceptance Scenarios**:

1. **Given** an ephemeral full stack seeded with F013 fixtures, **When** a
   maintainer starts an exploratory run, **Then** the explorer authenticates
   with a dedicated test account and exercises selected routes from the F013
   critical workflow inventory.
2. **Given** any configured exploration limit is reached, **When** the explorer
   attempts to continue, **Then** the run stops cleanly, records which limit
   ended it, and preserves the evidence collected so far.
3. **Given** production application settings or production data are absent,
   **When** the exploratory run executes, **Then** it completes solely against
   the ephemeral stack and synthetic fixture data.
4. **Given** a supported model provider is selected for the run, **When** the
   explorer starts, **Then** it uses the same provider-neutral exploration
   contract and does not require provider-specific production settings.

---

### User Story 2 - Review an evidence-first finding report (Priority: P1)

As a maintainer, I want every run to produce one structured report with
supporting browser evidence so I can distinguish actionable defects from model
speculation.

**Why this priority**: Findings are useful only when they are reviewable,
reproducible, and traceable to observed behavior.

**Independent Test**: Run exploration against a deterministic seeded defect
and verify the report identifies it with route history, reproduction steps,
severity, and linked evidence while keeping unrelated secrets out of all
outputs.

**Acceptance Scenarios**:

1. **Given** an exploratory run observes application behavior, **When** the run
   ends, **Then** its default output is a structured report artifact containing
   route history, screenshots, console errors, network failures,
   accessibility findings, model triage, run limits and usage, and termination
   reason.
2. **Given** the explorer reports a finding, **When** a maintainer reviews it,
   **Then** the finding includes severity, concise reproduction steps, expected
   and observed behavior, and references to the evidence that supports it.
3. **Given** the seeded defect is present, **When** the deterministic
   acceptance test runs, **Then** the report detects and classifies that defect
   and does not invent unsupported evidence.
4. **Given** credentials, tokens, or secret-shaped values are present in the
   run environment, **When** prompts, screenshots, reports, traces, logs, and
   proposed issue bodies are inspected, **Then** none of those values appear.

---

### User Story 3 - Run exploration manually and nightly without blocking delivery (Priority: P2)

As a maintainer, I want to start exploration on demand and receive a nightly
result without making an initially variable AI check block normal merges.

**Why this priority**: Scheduled repetition establishes usefulness and
stability, while advisory status prevents an immature signal from disrupting
the existing quality gate.

**Independent Test**: Trigger a manual run and observe a scheduled run in a
test workflow, then verify both publish the same report shape and a finding or
infrastructure failure does not fail an otherwise healthy merge gate.

**Acceptance Scenarios**:

1. **Given** an authorized maintainer, **When** they start a manual run, **Then**
   they can select the supported provider configuration and workflow scope
   without enabling issue creation by default.
2. **Given** the nightly schedule, **When** it starts, **Then** it creates a
   fresh ephemeral stack, executes within the same limits as a manual run, and
   publishes a structured report artifact.
3. **Given** exploration finds a defect or encounters explorer instability,
   **When** the advisory run concludes, **Then** its result is visible but does
   not block a pull request or merge.
4. **Given** stability criteria have not all been met, **When** CI policy is
   evaluated, **Then** no new merge-blocking gate is introduced.

---

### User Story 4 - Opt in to bounded issue creation (Priority: P3)

As a maintainer, I want findings to remain report-only by default and create
GitHub issues only when I explicitly request it so exploratory noise cannot
flood the backlog.

**Why this priority**: Automated issue creation is convenient but has greater
side effects and should follow safe exploration and reliable reporting.

**Independent Test**: Exercise the issue-publication flow through a fake issue
destination with opt-in disabled and enabled; verify zero issue requests in the
default case and bounded, sanitized requests only in the enabled case.

**Acceptance Scenarios**:

1. **Given** no issue-creation input or an explicit false value, **When** a run
   finds defects, **Then** it creates no GitHub issues and records findings only
   in the report artifact.
2. **Given** an authorized manual or scheduled run explicitly enables issue
   creation, **When** actionable findings remain after triage, **Then** no more
   than the configured issue limit are submitted.
3. **Given** the seeded-defect acceptance test exercises the opt-in path,
   **When** it completes, **Then** it proves the expected sanitized issue
   request would be made without opening a real issue.

### Edge Cases

- The selected model provider is unavailable, rate-limited, returns malformed
  output, or stops reporting reliable token usage.
- The ephemeral API, web application, or agent service fails readiness or
  becomes unavailable during exploration.
- Authentication expires, the dedicated test account cannot sign in, or the
  explorer reaches a route that requires a different permission.
- A page never becomes idle, navigation loops, repeated actions make no
  progress, or a modal/focus trap prevents recovery.
- A screenshot or browser message contains user-entered text that resembles a
  credential or token.
- A response body is too large or binary, a network request fails before a
  response exists, or the browser emits duplicate console/network events.
- Multiple observations describe the same underlying defect.
- A run ends because of one limit while another limit is reached
  simultaneously.
- The explicit issue input is enabled but issue publishing is unavailable or
  lacks permission.
- The seeded defect is absent, already fixed, or detected without the required
  evidence.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST provide a provider-agnostic exploratory runner;
  provider selection and credentials MUST be supplied through a dedicated
  test-run contract rather than production application settings.
- **FR-002**: The explorer MUST authenticate to the application using a
  dedicated test account created for the ephemeral run and MUST NOT use a
  production user account.
- **FR-003**: Every run MUST target a newly provisioned ephemeral full stack
  containing the web application, API, agent service, isolated storage, and
  synthetic data; it MUST NOT connect to or copy production data.
- **FR-004**: The ephemeral environment MUST seed and reuse Feature 220/F013's
  golden collection fixtures rather than defining a competing fixture set.
- **FR-005**: The initial exploration inventory MUST reuse the F013 critical
  workflows: login/session setup, add coin, edit one field, change and clear a
  storage location, edit tags and sets, upload and delete an image,
  search/filter the collection, and edit at a mobile viewport. Current
  deterministic sibling workflows MAY also be selected without replacing this
  baseline.
- **FR-006**: A run MUST be available through an authorized manual trigger and
  a nightly schedule.
- **FR-007**: Manual and nightly runs MUST be advisory and non-blocking. Their
  findings, model failures, and explorer infrastructure failures MUST NOT
  change the result of the existing merge-blocking quality gate.
- **FR-008**: Each run MUST enforce hard defaults of at most 25 exploration
  steps, 15 minutes wall time, 20 model calls, 60,000 total model input and
  output tokens, 100 browser actions, and 3 issue-creation attempts.
- **FR-009**: Each limit MUST be independently configurable only to a value at
  or below its documented hard maximum, snapshotted at run start, checked
  before and after relevant work, and reported with actual usage.
- **FR-010**: Reaching any hard limit MUST stop further model calls, browser
  actions, and issue creation as applicable, mark the run bounded rather than
  successful, and still publish all safely collected evidence.
- **FR-011**: The default run result MUST be a machine-readable structured
  report artifact, accompanied by human-readable summary information.
- **FR-012**: Each report MUST identify the run, selected workflow scope,
  ephemeral environment, start/end time, final status, termination reason,
  configured limits, actual usage, visited routes, and findings.
- **FR-013**: Evidence collection MUST include route/navigation history,
  screenshots at relevant checkpoints and failures, browser console errors,
  failed network requests and error responses, accessibility findings, and
  model-generated triage linked to the underlying evidence.
- **FR-014**: Every finding MUST include a stable within-run identifier,
  category, severity, confidence, affected route/workflow, reproduction steps,
  expected behavior, observed behavior, and evidence references.
- **FR-015**: Model triage MUST be clearly labeled as model-generated, MUST
  distinguish observed facts from interpretation, and MUST NOT upgrade an
  unsupported hypothesis into a finding without captured evidence.
- **FR-016**: Findings MUST be deduplicated within a run using their affected
  workflow, route, category, and evidence signature; cross-run deduplication
  MUST be considered before any issue is submitted.
- **FR-017**: GitHub issue creation MUST default to disabled and MUST occur only
  when an explicit boolean opt-in input is true for that run.
- **FR-018**: When issue creation is enabled, the system MUST submit no more
  than 3 issues per run, MUST avoid creating a known duplicate, and MUST retain
  every remaining finding in the report.
- **FR-019**: An issue body MUST contain sanitized reproduction steps,
  expected/observed behavior, severity, run reference, and evidence references;
  it MUST NOT contain credentials, raw secret-bearing output, or unsupported
  model reasoning.
- **FR-020**: Automated acceptance coverage MUST include a deterministic,
  test-only seeded defect that proves route/evidence collection, defect
  detection, and model triage.
- **FR-021**: Automated acceptance coverage MUST prove both the default
  report-only behavior and the explicit issue-creation path by using a fake or
  intercepted issue destination; ordinary tests MUST never create a real
  GitHub issue.
- **FR-022**: Secrets, credentials, session tokens, authorization values, and
  provider keys MUST be redacted or excluded before data enters model prompts,
  screenshots, report artifacts, traces, logs, or issue bodies.
- **FR-023**: Secret-safety validation MUST inspect every output channel named
  in FR-022 using known canary values and fail the exploratory run's privacy
  check if any canary is present.
- **FR-024**: Raw request/response bodies and browser storage contents MUST NOT
  be collected by default; any evidence excerpt MUST be allowlisted,
  size-bounded, and sanitized before persistence or model use.
- **FR-025**: All third-party CI actions, browser tooling, model adapters, and
  other dependencies introduced for this feature MUST be pinned to immutable
  versions in accordance with Constitution Principle VII.
- **FR-026**: The existing deterministic F013 browser workflow suite MUST
  remain the merge-blocking regression authority for its covered workflows;
  AI exploration MUST supplement rather than weaken, skip, or replace it.
- **FR-027**: Promotion of exploration to a merge-blocking gate MUST require a
  separate, explicit product decision after at least 20 consecutive nightly
  runs spanning at least 14 days achieve all of the following: 100% seeded
  defect detection, at least 95% infrastructure-complete runs, no secret
  leakage, no unintended real issue creation, and no more than 5% false
  positive findings among reviewed findings.
- **FR-028**: The feature MUST document the local/manual command, required
  dedicated test credentials, nightly behavior, report schema and location,
  limits, issue opt-in, privacy controls, and the distinction between
  deterministic and exploratory browser testing in `docs/testing.md`.

### Key Entities

- **Exploration Run**: One bounded manual or nightly execution, including
  selected provider, workflow scope, snapshotted limits, usage, status, and
  termination reason.
- **Exploration Step**: One model-directed decision cycle with its goal,
  observations, evidence references, and resulting browser actions.
- **Browser Evidence**: Sanitized route, screenshot, console, network, or
  accessibility observation captured during a run.
- **Exploration Finding**: A deduplicated, evidence-backed suspected defect
  with severity, confidence, reproduction details, and triage.
- **Exploration Report**: The structured artifact that aggregates run
  metadata, usage, route history, evidence, findings, and privacy validation.
- **Issue Publication Request**: An explicitly enabled, bounded, sanitized
  request derived from an actionable finding; absent by default.
- **Exploration Limits**: The snapshotted maximum steps, wall time, model
  calls/tokens, browser actions, and issue attempts for a run.

## Non-Goals

- Replacing the deterministic F013 critical workflow suite or making model
  behavior the sole proof that a workflow works.
- Running against beta, production, production backups, production user
  accounts, or production application/provider settings.
- Allowing the explorer to perform destructive administration, arbitrary
  external browsing, shell commands, filesystem access, or unrestricted API
  calls.
- Automatically fixing code, changing application data outside the ephemeral
  environment, approving pull requests, committing changes, or deploying.
- Opening GitHub issues by default or opening real issues during ordinary
  automated tests.
- Making exploratory results merge-blocking before the explicit stability
  threshold and separate approval in FR-027.
- Evaluating model quality through exact prose matching or requiring one
  specific model provider.
- Adding production monitoring, session replay, or error telemetry.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% of manual and nightly runs that start successfully use an
  ephemeral full stack, synthetic F013 fixtures, and dedicated test
  credentials, with zero production data access.
- **SC-002**: The deterministic seeded-defect suite detects the seeded defect
  in 100% of 20 consecutive validation runs and links it to route, screenshot,
  and at least one relevant console, network, or accessibility observation.
- **SC-003**: 100% of completed or bounded runs publish a valid structured
  report containing every evidence and usage category required by FR-012 and
  FR-013.
- **SC-004**: Boundary tests demonstrate that no run exceeds 25 exploration
  steps, 15 minutes, 20 model calls, 60,000 total tokens, 100 browser actions,
  or 3 issue-creation attempts.
- **SC-005**: Canary-secret tests find zero credential or token disclosures
  across prompts, screenshots, reports, traces, logs, and proposed issue
  bodies.
- **SC-006**: With issue opt-in omitted or false, 100% of runs create zero
  issues; with opt-in true in acceptance tests, the expected sanitized request
  is observed and zero real issues are created.
- **SC-007**: Manual and nightly runs publish equivalent report schemas, and
  100% of exploratory failures remain advisory without changing the existing
  merge-gate result.
- **SC-008**: Before any proposal to make exploration blocking, at least 20
  consecutive nightly runs over at least 14 days meet the reliability,
  seeded-detection, privacy, issue-safety, and false-positive thresholds in
  FR-027.
- **SC-009**: A maintainer can identify the affected workflow, reproduce a
  seeded finding, and locate its supporting evidence from the report in under
  5 minutes without reading raw runner logs.
- **SC-010**: Reviewers confirm that every initial exploration target maps to
  the F013 critical workflow inventory and no duplicate golden fixture catalog
  was introduced.

## Assumptions

- Feature 220/F013's golden fixture builders and critical workflow inventory
  remain the canonical baseline and are available before implementation begins.
- A dedicated test-provider credential and a dedicated ephemeral application
  account can be supplied to authorized manual and nightly runs through the
  repository's protected CI secret mechanism.
- Model-provider token accounting is available; if a provider cannot report
  reliable usage, the runner stops before further model work rather than
  treating usage as unbounded or estimated.
- Screenshots and other artifacts use synthetic test content only, but all
  outputs are still treated as potentially sensitive and pass through the same
  sanitization checks.
- Nightly timing follows the repository's normal CI timezone unless a later
  implementation plan records a different operational choice.
- Cross-run duplicate detection may use existing open issue metadata and stable
  finding fingerprints; the implementation design is deferred to planning.
