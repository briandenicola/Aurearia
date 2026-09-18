# Tasks: AI-Driven Browser Testing

**Input**: Design documents from `C:\Users\brian.denicolafamily\Code\AncientCoins\specs\360-ai-driven-browser-testing\`
**Prerequisites**: `plan.md`, `spec.md`, `research.md`, `data-model.md`, `quickstart.md`, and `contracts\`
**Branch constraint**: Work directly on `beta`; do not create or switch branches, commit, push, deploy, or open real GitHub issues while executing these tasks.
**Testing approach**: Test-first is required by Feature 360. In every story, complete the listed test/guard tasks and confirm they fail for the intended missing behavior before implementing the corresponding production or orchestration code.
**Path convention**: Every task names an absolute Windows repository path so it can be executed without path inference.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Safe to execute in parallel because it touches a different file and does not depend on an incomplete task in the same phase.
- **[Story]**: Maps the task to a user story in `spec.md`.

---

## Phase 1: Setup and Dependency Integrity

**Purpose**: Establish the exploration test surface, exact dependency versions, and artifact exclusions before feature code is added.

- [X] T001 Pin the direct `@playwright/test` dependency to exactly `1.63.0`, retain `playwright` and `playwright-core` lock resolutions at `1.63.0`, and retain Node `20.19.0` compatibility in `C:\Users\brian.denicolafamily\Code\AncientCoins\src\web\package.json` and `C:\Users\brian.denicolafamily\Code\AncientCoins\src\web\package-lock.json`
- [X] T002 [P] Add ignored run-config, staged report, prompt, evidence, and temporary credential paths for Feature 360 without weakening existing HAR exclusions in `C:\Users\brian.denicolafamily\Code\AncientCoins\.gitignore`
- [X] T003 [P] Create the exploration TypeScript module and test directory skeleton with a short ownership README in `C:\Users\brian.denicolafamily\Code\AncientCoins\src\web\e2e\exploration\README.md`
- [X] T004 Add a dependency-integrity test that rejects non-exact Playwright versions, changed Playwright-family lock resolutions, added browser-agent/model SDKs, or drift in locked LangChain `1.4.0`, Anthropic `1.7.2`, Ollama `1.1.0`, and LangGraph `1.2.11` versions in `C:\Users\brian.denicolafamily\Code\AncientCoins\src\web\e2e\exploration\__tests__\dependency-integrity.test.ts`

**Checkpoint**: The repository has a pinned, intentionally minimal dependency surface and no generated exploration artifacts can be committed accidentally.

---

## Phase 2: Foundational Contracts and Cross-Service Types

**Purpose**: Make the checked-in contracts executable and prevent TypeScript, Go, and Python drift before any user story implementation.

**⚠️ CRITICAL**: This phase blocks every user story.

### Contract tests — write first and confirm failure

- [X] T005 [P] Add run-configuration contract fixtures covering unknown fields, secret-shaped fields, raw URLs, empty/duplicate workflows, unsupported providers, zero/negative values, and every exact hard maximum in `C:\Users\brian.denicolafamily\Code\AncientCoins\src\web\e2e\exploration\__tests__\config-contract.test.ts`
- [X] T006 [P] Add report-contract fixtures covering required fields, enum values, chronological ordering, usage not exceeding snapshots, evidence references, traversal-safe artifact paths, and `productionDataUsed=false` in `C:\Users\brian.denicolafamily\Code\AncientCoins\src\web\e2e\exploration\__tests__\report-contract.test.ts`
- [X] T007 [P] Add issue-publication contract fixtures covering repository/title/body limits, fixed labels, the exact hidden fingerprint marker, unknown-field rejection, and secret-free payloads in `C:\Users\brian.denicolafamily\Code\AncientCoins\src\web\e2e\exploration\__tests__\issue-contract.test.ts`
- [X] T008 [P] Add Python request/response contract tests for strict fields, action-target bounds, evidence references, prompt-injection content treated as inert data, malformed model output, and mandatory non-negative provider usage in `C:\Users\brian.denicolafamily\Code\AncientCoins\src\agent\tests\test_browser_exploration_contract.py`
- [X] T009 [P] Add Go contract-drift tests that compare Go request/response DTO JSON shapes with `C:\Users\brian.denicolafamily\Code\AncientCoins\specs\360-ai-driven-browser-testing\contracts\exploration-agent.openapi.yaml` in `C:\Users\brian.denicolafamily\Code\AncientCoins\src\api\services\agent_proxy_exploration_contract_test.go`

### Contract implementation

- [X] T010 Implement strict TypeScript `RunConfiguration`, `ExplorationLimits`, report, evidence, finding, privacy, termination, and publication types plus runtime guards generated from the checked-in schemas in `C:\Users\brian.denicolafamily\Code\AncientCoins\src\web\e2e\exploration\contracts.ts`
- [X] T011 [P] Implement strict Pydantic request, decision, action-target, model-triage, and usage DTOs with `extra="forbid"` in `C:\Users\brian.denicolafamily\Code\AncientCoins\src\agent\app\models\browser_exploration.py`
- [X] T012 [P] Implement typed Go exploration proxy DTOs and contract constants without importing database or handler packages in `C:\Users\brian.denicolafamily\Code\AncientCoins\src\api\services\agent_proxy_exploration.go`
- [X] T013 Wire schema loading and reusable valid/invalid contract fixtures for TypeScript tests from `C:\Users\brian.denicolafamily\Code\AncientCoins\specs\360-ai-driven-browser-testing\contracts\run-config.schema.json`, `C:\Users\brian.denicolafamily\Code\AncientCoins\specs\360-ai-driven-browser-testing\contracts\exploration-report.schema.json`, and `C:\Users\brian.denicolafamily\Code\AncientCoins\specs\360-ai-driven-browser-testing\contracts\issue-publication.schema.json` in `C:\Users\brian.denicolafamily\Code\AncientCoins\src\web\e2e\exploration\__tests__\contract-fixtures.ts`
- [X] T014 Run T005–T009 and record the expected pre-implementation failures, then make the contract suites pass without weakening any checked-in schema in `C:\Users\brian.denicolafamily\Code\AncientCoins\specs\360-ai-driven-browser-testing\implementation-validation.md`

**Checkpoint**: All four v1 contracts reject unknown or unsafe input and their TypeScript, Go, and Python representations cannot drift silently.

---

## Phase 3: User Story 1 — Explore Critical Workflows Safely (Priority: P1) 🎯 MVP

**Goal**: Run provider-neutral, bounded exploration against a freshly built, loopback-only full stack seeded from the canonical F013 fixtures and dedicated per-run credentials.

**Independent Test**: From `C:\Users\brian.denicolafamily\Code\AncientCoins`, start one low-budget fake-model run and verify normal login, selected F013 workflow visits, exact limit termination, no external origin/database/volume use, and unconditional stack teardown.

### Tests and tamper guards — write first and confirm failure

- [X] T015 [P] [US1] Add exact-boundary and one-over tests for steps `25`, wall time `900`, model calls `20`, combined tokens `60000`, browser actions `100`, issue attempts `3`, lower configured limits, simultaneous-limit precedence, attempted-work accounting, and immutable snapshots in `C:\Users\brian.denicolafamily\Code\AncientCoins\src\web\e2e\exploration\__tests__\budget.test.ts`
- [X] T016 [P] [US1] Add allowlisted-driver tests that reject external URLs, routes outside the selected F013 workflow, arbitrary JavaScript, shell commands, generic requests, filesystem paths, unregistered uploads, oversized locator/value arguments, and prompt-injected instructions requesting those capabilities in `C:\Users\brian.denicolafamily\Code\AncientCoins\src\web\e2e\exploration\__tests__\browser-driver.test.ts`
- [X] T017 [P] [US1] Add Python decision-service tests for Anthropic/Ollama provider neutrality, dedicated per-request configuration, no production `AppSetting` input, structured-output validation, rate limits, unavailable providers, missing/malformed token usage, and prompt-injection strings preserved only as observation data in `C:\Users\brian.denicolafamily\Code\AncientCoins\src\agent\tests\test_browser_exploration_service.py`
- [X] T018 [P] [US1] Add Go handler/proxy tests for feature-disabled behavior, missing/wrong ephemeral bearer token, request validation, dedicated exploration environment variables, absence of production-setting reads, timeout propagation, and stable upstream error mapping in `C:\Users\brian.denicolafamily\Code\AncientCoins\src\api\handlers\internal_exploration_test.go`
- [X] T019 [P] [US1] Add seed-command tests proving a unique test account and `testutil.PersistGoldenCollection` associations are created only in the supplied ephemeral database and production-like paths are rejected in `C:\Users\brian.denicolafamily\Code\AncientCoins\src\api\cmd\exploration-seed\main_test.go`
- [X] T020 [P] [US1] Add workflow-adapter tests proving all eight baseline IDs map to `C:\Users\brian.denicolafamily\Code\AncientCoins\src\web\e2e\fixtures\workflow.ts` behavior without copying the golden fixture catalog in `C:\Users\brian.denicolafamily\Code\AncientCoins\src\web\e2e\exploration\__tests__\workflows.test.ts`
- [X] T021 [P] [US1] Add Compose/launcher tamper tests that reject non-loopback origins, external database paths, deployment `.env` files, pre-existing or named production volumes, remote images, fixed public ports, missing unique project names, agent host exposure, and cleanup commands without `down -v --remove-orphans` in `C:\Users\brian.denicolafamily\Code\AncientCoins\src\web\e2e\exploration\__tests__\isolation.test.ts`
- [X] T022 [P] [US1] Add orchestration failure tests for app/API/agent readiness failures, expired authentication, no-progress loops, modal/focus traps, cancellation, in-flight timeout aborts, and finalization after partial evidence in `C:\Users\brian.denicolafamily\Code\AncientCoins\src\web\e2e\exploration\__tests__\orchestration.test.ts`

### Implementation

- [X] T023 [P] [US1] Implement strict secret-free configuration parsing, hard-maximum rejection, workflow validation, default `createIssues=false`, and immutable limit snapshots in `C:\Users\brian.denicolafamily\Code\AncientCoins\src\web\e2e\exploration\config.ts`
- [X] T024 [P] [US1] Implement a monotonic `BudgetGuard` with reserve/perform/reconcile APIs, attempted-work counters, abortable deadlines, fail-closed provider usage, simultaneous-limit reporting, and documented precedence in `C:\Users\brian.denicolafamily\Code\AncientCoins\src\web\e2e\exploration\budget.ts`
- [X] T025 [P] [US1] Implement the fixed action vocabulary and typed Playwright executor with same-origin route enforcement, locator/value bounds, packaged synthetic uploads, one Chromium worker, and no script/shell/filesystem/generic-request escape hatch in `C:\Users\brian.denicolafamily\Code\AncientCoins\src\web\e2e\exploration\browser-driver.ts`
- [X] T026 [P] [US1] Extract a named, immutable F013 workflow inventory from existing workflow helpers without duplicating fixture data in `C:\Users\brian.denicolafamily\Code\AncientCoins\src\web\e2e\fixtures\workflow.ts`
- [X] T027 [US1] Implement the exploration workflow adapter that selects only configured F013 IDs and supplies bounded goals, routes, checkpoints, and viewport requirements to the driver in `C:\Users\brian.denicolafamily\Code\AncientCoins\src\web\e2e\exploration\workflows.ts`
- [X] T028 [P] [US1] Implement the stateless Python decision service using the existing `get_structured_model` provider factory, strict structured output, sanitized observations, and mandatory provider-reported usage in `C:\Users\brian.denicolafamily\Code\AncientCoins\src\agent\app\services\browser_exploration.py`
- [X] T029 [US1] Implement the internal FastAPI decision route with the v1 DTOs and no persistence or browser capability in `C:\Users\brian.denicolafamily\Code\AncientCoins\src\agent\app\routers\internal_exploration.py`
- [X] T030 [US1] Register the internal exploration router without changing public agent routes in `C:\Users\brian.denicolafamily\Code\AncientCoins\src\agent\app\main.py`
- [X] T031 [P] [US1] Implement the opt-in Go proxy method that reads only dedicated exploration-provider environment variables, injects per-request provider configuration, uses `AgentProxy`, and forwards cancellation/timeouts in `C:\Users\brian.denicolafamily\Code\AncientCoins\src\api\services\agent_proxy_exploration.go`
- [X] T032 [US1] Implement the disabled-by-default authenticated internal Go handler with thin validation/error mapping in `C:\Users\brian.denicolafamily\Code\AncientCoins\src\api\handlers\internal_exploration.go`
- [X] T033 [US1] Register the internal exploration route only when `AI_BROWSER_EXPLORATION_ENABLED=true` and require the per-run internal bearer token in `C:\Users\brian.denicolafamily\Code\AncientCoins\src\api\main.go`
- [X] T034 [P] [US1] Implement the one-shot test-only seed command that creates the dedicated account and calls `testutil.PersistGoldenCollection` against the supplied database in `C:\Users\brian.denicolafamily\Code\AncientCoins\src\api\cmd\exploration-seed\main.go`
- [X] T035 [P] [US1] Add the exploration Playwright configuration for Chromium-only execution, one worker, isolated output, masked screenshots, and trace/video/HAR disabled in `C:\Users\brian.denicolafamily\Code\AncientCoins\src\web\playwright.exploration.config.ts`
- [ ] T036 [US1] Add a current-source Compose stack with an internal-only agent, loopback random app port, project-scoped SQLite/upload volumes, health checks, no restart policy, and the one-shot seed service in `C:\Users\brian.denicolafamily\Code\AncientCoins\docker-compose.exploration.yml`
- [ ] T037 [US1] Implement the CLI lifecycle that generates per-run credentials and project IDs, validates isolation before startup, builds current source, seeds/waits/runs/finalizes, aborts on bounds, and unconditionally tears down volumes/orphans in `C:\Users\brian.denicolafamily\Code\AncientCoins\src\web\e2e\exploration\cli.ts`
- [ ] T038 [US1] Run T015–T022, verify each guard failed before its implementation, then make all US1 suites pass and record the commands/results in `C:\Users\brian.denicolafamily\Code\AncientCoins\specs\360-ai-driven-browser-testing\implementation-validation.md`

**Checkpoint**: US1 is independently usable with a fake model and isolated synthetic stack; no production endpoint, account, setting, data, or durable volume is reachable.

---

## Phase 4: User Story 2 — Review an Evidence-First Finding Report (Priority: P1)

**Goal**: Produce a versioned, evidence-backed, human-reviewable report in which every persisted/transmitted observation crosses one sanitizer boundary and model interpretation cannot become unsupported fact.

**Independent Test**: Execute the seeded HTTP-500 defect 20 times with the fake model and verify 20/20 reports contain the route, masked screenshot, relevant network evidence, reproduction details, separately labeled triage, valid hashes/references, and zero canary leakage.

### Tests and tamper guards — write first and confirm failure

- [ ] T039 [P] [US2] Add sanitizer tests for credentials, JWTs, provider keys, authorization/header values, cookies, local/session storage, password fields, raw/binary/oversized bodies, query values, console/accessibility text, and prompt-injection payloads across every sanitized text field in `C:\Users\brian.denicolafamily\Code\AncientCoins\src\web\e2e\exploration\__tests__\sanitizer.test.ts`
- [ ] T040 [P] [US2] Add evidence-store tests for capture-time sanitization, sensitive-selector masking, visible-canary screenshot refusal, route templating, content SHA-256, truncation, duplicate event normalization, relative `evidence\` paths, traversal rejection, and tampered-content/hash detection in `C:\Users\brian.denicolafamily\Code\AncientCoins\src\web\e2e\exploration\__tests__\evidence.test.ts`
- [ ] T041 [P] [US2] Add finding tests requiring captured evidence, deterministic observed facts, valid references, stable fingerprinting over workflow/route/category/evidence signature, within-run deduplication, deterministic ordering, and model-triage inability to add facts or evidence in `C:\Users\brian.denicolafamily\Code\AncientCoins\src\web\e2e\exploration\__tests__\findings.test.ts`
- [ ] T042 [P] [US2] Extend report contract tests with summary/report parity, all required evidence categories, exact usage/termination snapshots, chronology, referential integrity, content-address validation, facts-versus-triage separation, and rejection of tampered report/evidence bundles in `C:\Users\brian.denicolafamily\Code\AncientCoins\src\web\e2e\exploration\__tests__\report-contract.test.ts`
- [ ] T043 [P] [US2] Add a channel-complete privacy test that plants distinct canaries in prompts, rendered screenshot text, report JSON, summary Markdown, evidence, traces, logs, and proposed issue bodies and asserts `privacy_failed`, publication blocking, no unsafe upload, and a content-free minimal diagnostic manifest in `C:\Users\brian.denicolafamily\Code\AncientCoins\src\web\e2e\exploration\__tests__\privacy-tamper.test.ts`
- [ ] T044 [P] [US2] Add the deterministic seeded `edit-save-http-500-v1` acceptance test using the real ephemeral stack, canonical edit workflow, fake schema-valid model usage, and fake publisher; require route+screenshot+network evidence and unsupported-evidence rejection in `C:\Users\brian.denicolafamily\Code\AncientCoins\src\web\e2e\exploration\__tests__\seeded-defect.spec.ts`
- [ ] T045 [US2] Add a 20-run seeded-defect reliability wrapper that requires 20/20 detection, validates every report bundle, detects cross-run state leakage, and makes no live model or GitHub request in `C:\Users\brian.denicolafamily\Code\AncientCoins\src\web\e2e\exploration\__tests__\seeded-defect-reliability.test.ts`

### Implementation

- [ ] T046 [P] [US2] Implement the sole raw-to-sanitized observation boundary with deny-by-default collection, bounded allowlists, route templating, secret-pattern replacement, and an opaque sanitized type required by downstream APIs in `C:\Users\brian.denicolafamily\Code\AncientCoins\src\web\e2e\exploration\sanitizer.ts`
- [ ] T047 [US2] Implement sanitized route, screenshot, console, network, accessibility, and UI evidence capture with masking, canary prechecks, truncation, hashing, content-addressed persistence, and duplicate suppression in `C:\Users\brian.denicolafamily\Code\AncientCoins\src\web\e2e\exploration\evidence.ts`
- [ ] T048 [P] [US2] Implement deterministic evidence-backed finding construction, stable fingerprints, within-run deduplication, severity policy, reproduction data, and separately labeled model triage in `C:\Users\brian.denicolafamily\Code\AncientCoins\src\web\e2e\exploration\findings.ts`
- [ ] T049 [US2] Implement v1 report assembly, schema/referential/hash/privacy validation, usage/termination accounting, `summary.md` derivation, and minimal privacy-failure output in `C:\Users\brian.denicolafamily\Code\AncientCoins\src\web\e2e\exploration\report.ts`
- [ ] T050 [US2] Integrate sanitizer-before-prompt, evidence checkpoints/failures, deterministic finding creation, final report validation, privacy scanning, and safe artifact staging into `C:\Users\brian.denicolafamily\Code\AncientCoins\src\web\e2e\exploration\cli.ts`
- [ ] T051 [US2] Implement the named test-only HTTP-500 route interception at the Playwright boundary without changing production application behavior in `C:\Users\brian.denicolafamily\Code\AncientCoins\src\web\e2e\exploration\seeded-defects.ts`
- [ ] T052 [US2] Run T039–T045, verify each guard failed before its implementation, then make all US2 suites and the 20-run acceptance loop pass and record results in `C:\Users\brian.denicolafamily\Code\AncientCoins\specs\360-ai-driven-browser-testing\implementation-validation.md`

**Checkpoint**: A maintainer can identify and reproduce the seeded defect from a validated report in under five minutes, while canary/tamper failures fail closed before upload or publication.

---

## Phase 5: User Story 3 — Run Manually and Nightly Without Blocking Delivery (Priority: P2)

**Goal**: Expose the same isolated Taskfile command through manual and nightly advisory automation without coupling it to pull requests, pushes, Quality Gate, deployment, or branch protection.

**Independent Test**: Validate manual and scheduled configurations locally, compare their report schemas, inject a finding and an infrastructure failure, and prove neither changes the result or triggers of `Quality Gate`.

### Tests and workflow guards — write first and confirm failure

- [ ] T053 [P] [US3] Add Taskfile contract tests for identical local/manual/nightly CLI entry points, fake-only test targets, 20-run seeded target, forwarded config arguments, and unconditional cleanup in `C:\Users\brian.denicolafamily\Code\AncientCoins\src\web\e2e\exploration\__tests__\taskfile-contract.test.ts`
- [ ] T054 [P] [US3] Add workflow tamper tests that reject `pull_request`, `push`, `workflow_run`, deployment dependencies, mutable action tags, non-`20.19.0` Node, non-locked installs, default issue opt-in, broad permissions, publisher access to raw outputs, missing `if: always()` cleanup/upload, or infinite artifact retention in `C:\Users\brian.denicolafamily\Code\AncientCoins\src\web\e2e\exploration\__tests__\workflow-policy.test.ts`
- [ ] T055 [P] [US3] Add manual/nightly parity tests that generate both configurations, enforce the same hard limits and report contract, require fresh Compose identities/volumes, and prove findings/model/infrastructure failures remain advisory to `C:\Users\brian.denicolafamily\Code\AncientCoins\.github\workflows\ci.yml` in `C:\Users\brian.denicolafamily\Code\AncientCoins\src\web\e2e\exploration\__tests__\advisory-parity.test.ts`

### Implementation

- [ ] T056 [US3] Add `ai-browser-exploration`, `test-ai-browser-exploration`, and `test-ai-browser-seeded-defect` targets using the same CLI and locked toolchain in `C:\Users\brian.denicolafamily\Code\AncientCoins\Taskfile.yml`
- [ ] T057 [US3] Add the separately named manual/nightly advisory workflow with bounded inputs, issue default `false`, current-source stack build, protected dedicated secrets, immutable 40-character action SHAs, `contents: read`, finite-retention artifact upload, and always-run cleanup in `C:\Users\brian.denicolafamily\Code\AncientCoins\.github\workflows\ai-browser-exploration.yml`
- [ ] T058 [US3] Add a conditional publication job that receives only validated sanitized publication requests and gets `issues: write` only when explicit opt-in is true in `C:\Users\brian.denicolafamily\Code\AncientCoins\.github\workflows\ai-browser-exploration.yml`
- [ ] T059 [US3] Run T053–T055, verify each policy test failed before workflow implementation, then make the workflow/Taskfile suites pass and record manual/nightly schema parity evidence in `C:\Users\brian.denicolafamily\Code\AncientCoins\specs\360-ai-driven-browser-testing\implementation-validation.md`

**Checkpoint**: Manual and nightly runs use one command and one report contract; the existing `Quality Gate` and deployment workflows are unchanged and remain the only merge/release authority.

---

## Phase 6: User Story 4 — Opt In to Bounded Issue Creation (Priority: P3)

**Goal**: Keep every run report-only by default and permit at most three sanitized, deduplicated GitHub issue creates only after explicit opt-in and successful report/privacy validation.

**Independent Test**: Run fake publication with opt-in omitted, false, and true; observe zero requests by default, exact-marker duplicate searches over open and closed issues, no comments on duplicates, at most three create attempts, and no real network call.

### Tests and tamper guards — write first and confirm failure

- [ ] T060 [P] [US4] Add publisher tests for omitted/false opt-in, invalid report, failed privacy check, non-actionable findings, deterministic ordering, fixed repository/label allowlists, and absence of the GitHub token from the exploration process in `C:\Users\brian.denicolafamily\Code\AncientCoins\src\web\e2e\exploration\__tests__\publisher-opt-in.test.ts`
- [ ] T061 [P] [US4] Add deduplication tests for the exact `<!-- aurearia-ai-finding:v1:<fingerprint> -->` marker across open and closed issues, title changes, within-run duplicates, zero duplicate comments, preserved duplicate URLs, and retained unpublished findings in `C:\Users\brian.denicolafamily\Code\AncientCoins\src\web\e2e\exploration\__tests__\publisher-dedup.test.ts`
- [ ] T062 [P] [US4] Add attempt-boundary tests proving every create attempt counts on success, permission failure, timeout, and server error; the fourth attempt is refused; duplicate searches do not consume attempts; and simultaneous privacy/limit failure follows termination precedence in `C:\Users\brian.denicolafamily\Code\AncientCoins\src\web\e2e\exploration\__tests__\publisher-budget.test.ts`
- [ ] T063 [P] [US4] Add fake-destination acceptance tests proving the seeded finding yields one schema-valid sanitized request when enabled, zero requests when disabled, no unsupported model reasoning, no canary leakage, and no external GitHub call in `C:\Users\brian.denicolafamily\Code\AncientCoins\src\web\e2e\exploration\__tests__\publisher-acceptance.test.ts`

### Implementation

- [ ] T064 [US4] Implement post-validation issue request derivation, fixed labels/repository validation, exact-marker searches over open and closed issues, deterministic selection, no duplicate comments, and budgeted creates through injectable `fetch` in `C:\Users\brian.denicolafamily\Code\AncientCoins\src\web\e2e\exploration\publisher.ts`
- [ ] T065 [US4] Integrate publisher invocation only after valid report and passed privacy checks, keep the GitHub token out of exploration subprocesses, and persist every publication outcome without deleting findings in `C:\Users\brian.denicolafamily\Code\AncientCoins\src\web\e2e\exploration\cli.ts`
- [ ] T066 [US4] Run T060–T063, verify each guard failed before implementation, then make all fake-publication tests pass with network interception proving zero real issue creation and record results in `C:\Users\brian.denicolafamily\Code\AncientCoins\specs\360-ai-driven-browser-testing\implementation-validation.md`

**Checkpoint**: Issue publication is explicit, sanitized, deduplicated, least-privilege, attempt-bounded, and fully testable without side effects.

---

## Phase 7: Polish, Documentation, Quality Gates, and Post-Major-Work Audit

**Purpose**: Finish operator documentation and durable decisions, validate rollback and tamper resistance, then execute targeted and full constitutional gates before the post-major-work QC audit.

- [ ] T067 [P] Document local/manual/nightly commands, dedicated credentials, report paths/contracts, every hard limit, issue opt-in, privacy controls, ephemeral isolation, F013-versus-exploration authority, advisory behavior, FR-027 promotion threshold, troubleshooting, and rollback in `C:\Users\brian.denicolafamily\Code\AncientCoins\docs\testing.md`
- [ ] T068 [P] Add ADR 0017 covering the TypeScript→Go→Python boundary, dedicated credentials, pre-prompt sanitization, ephemeral full-stack isolation, evidence contract, least-privilege publication, advisory CI, and migration-free rollback in `C:\Users\brian.denicolafamily\Code\AncientCoins\docs\adr\0017-ai-driven-browser-exploration-boundary.md`
- [ ] T069 [P] Add ADR 0017 to the architecture decision index in `C:\Users\brian.denicolafamily\Code\AncientCoins\docs\adr\README.md`
- [ ] T070 [P] Record the cross-cutting implementation decision summary and any deviations from `research.md` in `C:\Users\brian.denicolafamily\Code\AncientCoins\.squad\decisions\inbox\feature-360-ai-browser-exploration.md`
- [ ] T071 [P] Add an operator-facing sample with deliberately fake placeholders, low smoke limits, `createIssues=false`, and no usable secret or endpoint in `C:\Users\brian.denicolafamily\Code\AncientCoins\specs\360-ai-driven-browser-testing\examples\run-config.example.json`
- [ ] T072 Run targeted TypeScript contract, bounds, action-allowlist, prompt-injection, redaction, evidence/report tamper, seeded-defect, workflow-policy, and fake-publication suites through `task test-ai-browser-exploration` and record exact commands/results in `C:\Users\brian.denicolafamily\Code\AncientCoins\specs\360-ai-driven-browser-testing\implementation-validation.md`
- [ ] T073 Run targeted Python browser-exploration contract/service tests and Ruff on the touched modules from `C:\Users\brian.denicolafamily\Code\AncientCoins\src\agent\tests\test_browser_exploration_contract.py` and `C:\Users\brian.denicolafamily\Code\AncientCoins\src\agent\tests\test_browser_exploration_service.py`, recording results in `C:\Users\brian.denicolafamily\Code\AncientCoins\specs\360-ai-driven-browser-testing\implementation-validation.md`
- [ ] T074 Run targeted Go exploration handler, proxy contract, seed command, and architecture tests from `C:\Users\brian.denicolafamily\Code\AncientCoins\src\api\handlers\internal_exploration_test.go`, `C:\Users\brian.denicolafamily\Code\AncientCoins\src\api\services\agent_proxy_exploration_contract_test.go`, and `C:\Users\brian.denicolafamily\Code\AncientCoins\src\api\cmd\exploration-seed\main_test.go`, recording results in `C:\Users\brian.denicolafamily\Code\AncientCoins\specs\360-ai-driven-browser-testing\implementation-validation.md`
- [ ] T075 Run `task test-ai-browser-seeded-defect` from `C:\Users\brian.denicolafamily\Code\AncientCoins\Taskfile.yml`, require 20/20 seeded detections with valid route/screenshot/network evidence and zero cross-run leakage, and record results in `C:\Users\brian.denicolafamily\Code\AncientCoins\specs\360-ai-driven-browser-testing\implementation-validation.md`
- [ ] T076 Run `task test-critical-workflows` from `C:\Users\brian.denicolafamily\Code\AncientCoins\Taskfile.yml` and confirm all eight F013 baseline workflows remain deterministic and authoritative, recording results in `C:\Users\brian.denicolafamily\Code\AncientCoins\specs\360-ai-driven-browser-testing\implementation-validation.md`
- [ ] T077 Run the full Constitution §17 Go gates (`go build ./...`, `go vet ./...`, `go test -v ./...`, and `task test-race`) against `C:\Users\brian.denicolafamily\Code\AncientCoins\src\api\` and record results in `C:\Users\brian.denicolafamily\Code\AncientCoins\specs\360-ai-driven-browser-testing\implementation-validation.md`
- [ ] T078 Run the full Constitution §17 web gates (`npm run lint`, `npm run type-check`, `npm run test`, and `npm run build`) against `C:\Users\brian.denicolafamily\Code\AncientCoins\src\web\` and record results in `C:\Users\brian.denicolafamily\Code\AncientCoins\specs\360-ai-driven-browser-testing\implementation-validation.md`
- [ ] T079 Run the full Constitution §17 agent gates (`uv sync --locked --extra dev`, `uv run ruff check app/ tests/`, and `uv run pytest tests/ -v`) against `C:\Users\brian.denicolafamily\Code\AncientCoins\src\agent\` and record results in `C:\Users\brian.denicolafamily\Code\AncientCoins\specs\360-ai-driven-browser-testing\implementation-validation.md`
- [ ] T080 Validate all JSON Schemas, Go/Python/TypeScript contract drift, Compose teardown after forced failures, advisory workflow isolation from `C:\Users\brian.denicolafamily\Code\AncientCoins\.github\workflows\ci.yml`, immutable action pins, locked dependencies, and a repository secret scan; record the supply-chain/privacy/tamper evidence in `C:\Users\brian.denicolafamily\Code\AncientCoins\specs\360-ai-driven-browser-testing\implementation-validation.md`
- [ ] T081 Execute the `post-major-work-qc-audit` skill only after T072–T080 pass, remediate every blocking engineering, security, architecture, documentation, test, supply-chain, UX, and operational finding, and record the audit disposition in `C:\Users\brian.denicolafamily\Code\AncientCoins\specs\360-ai-driven-browser-testing\qc-audit.md`
- [ ] T082 Re-run every targeted Feature 360 command and the complete §17 gate after audit remediations, then finalize pass/fail evidence, known limitations, rollback verification, and FR-027 non-promotion status in `C:\Users\brian.denicolafamily\Code\AncientCoins\specs\360-ai-driven-browser-testing\implementation-validation.md`

**Checkpoint**: Feature 360 is documented, contract-stable, tamper-tested, dependency-pinned, fully gated, audited, and still advisory; no implementation task commits, pushes, deploys, or opens a real issue.

---

## Dependencies and Execution Order

### Phase dependencies

1. **Phase 1 — Setup** has no predecessor.
2. **Phase 2 — Foundational contracts** depends on Phase 1 and blocks every user story.
3. **Phase 3 — US1** depends on Phase 2; it establishes the bounded runner, cross-service decision path, canonical F013 mapping, and ephemeral stack.
4. **Phase 4 — US2** depends on Phase 3 because evidence capture and report finalization wrap the working explorer.
5. **Phase 5 — US3** depends on Phases 3 and 4 because CI must invoke the complete isolated runner and publish its validated report.
6. **Phase 6 — US4** depends on Phase 4; it can proceed in parallel with Phase 5 after report/privacy validation exists.
7. **Phase 7 — Polish and gates** depends on all selected stories; T081 depends on T072–T080, and T082 depends on all audit remediations.

### User story dependencies

- **US1 (P1)**: Starts after Phase 2 and is the technical MVP.
- **US2 (P1)**: Requires US1's observations but remains independently checkable through the deterministic seeded defect.
- **US3 (P2)**: Requires US1+US2; does not require issue publication.
- **US4 (P3)**: Requires US2; does not require manual/nightly workflow implementation and can run in parallel with US3.

### Critical path

`T001 → T005/T006/T007/T008/T009 → T010/T011/T012/T013 → T014 → T015–T022 → T023–T037 → T038 → T039–T045 → T046–T051 → T052 → (T053–T059 || T060–T066) → T072–T080 → T081 → T082`

The longest implementation branch after US2 is expected to be US3 because workflow permission/pinning validation and CI parity depend on the complete full-stack/report path. US4 is deliberately parallelizable once US2 is complete.

---

## Safe Parallel Execution Examples

### Foundational contracts

```text
Parallel: T005 TypeScript config contract | T006 TypeScript report contract | T007 TypeScript issue contract | T008 Python contract | T009 Go drift contract
After tests exist: T011 Python DTOs | T012 Go DTOs
```

### User Story 1

```text
Parallel tests: T015 budget | T016 driver | T017 Python service | T018 Go proxy | T019 seed | T020 workflows | T021 isolation | T022 orchestration
After their tests fail: T023 config | T024 budget | T025 driver | T026 F013 inventory | T028 Python service | T031 Go proxy | T034 seed command | T035 Playwright config
Serialize integration: T027 → T037 and T028 → T029 → T030, plus T031 → T032 → T033, before T038
```

### User Story 2

```text
Parallel tests: T039 sanitizer | T040 evidence | T041 findings | T042 report | T043 privacy tamper | T044 seeded defect
Parallel implementation after guards fail: T046 sanitizer | T048 findings
Serialize evidence/report integration: T046 → T047 → T049 → T050 → T051 → T052
```

### User Stories 3 and 4

```text
After US2: run Phase 5 (T053–T059) and Phase 6 (T060–T066) concurrently with separate owners
Within US3: T053 Taskfile guard | T054 workflow policy | T055 parity
Within US4: T060 opt-in | T061 dedup | T062 budget | T063 fake acceptance
```

---

## Implementation Strategy

### MVP first

1. Complete Phases 1 and 2.
2. Complete Phase 3 (US1).
3. Stop and validate the bounded fake-model run against an ephemeral F013-seeded stack.
4. Treat this as the technical MVP only; do not enable scheduled operation or issue publication.

### First reviewable release

1. Add Phase 4 (US2) so the MVP emits a validated, privacy-scanned, evidence-first report.
2. Require the seeded defect to pass 20/20 before enabling Phase 5 automation.
3. Add Phase 5 (US3) as advisory manual/nightly automation.
4. Add Phase 6 (US4) last; retain `createIssues=false` as the default.
5. Complete Phase 7 and keep exploration non-blocking until a separate decision satisfies FR-027.

### Execution rules

- Complete and observe each test-first guard failing before its corresponding implementation task.
- Never grant the exploration process a GitHub token; only the post-validation publisher may receive it.
- Never substitute production/beta data, credentials, URLs, settings, volumes, or deployment Compose files.
- Never weaken F013 or the existing `Quality Gate`; Feature 360 supplements them.
- Do not create/switch branches, commit, push, deploy, or open real issues while executing this task list.
