# Implementation Plan: AI-Driven Browser Testing

**Branch**: `beta` (feature artifacts: `360-ai-driven-browser-testing`) | **Date**: 2026-09-18 | **Spec**: `/specs/360-ai-driven-browser-testing/spec.md`
**Input**: Feature specification from `/specs/360-ai-driven-browser-testing/spec.md`

## Summary

Add an advisory, provider-neutral browser exploration tier around the existing
F013 Playwright workflows. A TypeScript test harness controls Playwright and
enforces all run budgets; the Go API proxies sanitized decision requests to a
strictly typed Python-agent endpoint that reuses the existing Anthropic/Ollama
provider factory. Each run provisions an isolated full stack, a dedicated test
user, and Feature 220's persisted golden collection, then emits a schema-checked
JSON report plus sanitized evidence. GitHub publication is a separate,
explicitly enabled, bounded post-report step. No production data, settings,
credentials, schema migration, or merge-blocking check is introduced.

## Technical Context

**Language/Version**: TypeScript 5.9.3 on Node.js 20.19.0; Go 1.26.6; Python 3.12
**Primary Dependencies**: Existing `@playwright/test`/`playwright`/`playwright-core` 1.63.0; existing Gin/GORM API and `AgentProxy`; existing FastAPI, Pydantic, LangChain 1.4.0, LangChain Anthropic 1.7.2, LangChain Ollama 1.1.0, and LangGraph 1.2.11; Docker Compose; Node built-ins (`fetch`, `crypto`, `fs`, `AbortController`)
**Storage**: Per-run SQLite and upload Docker volumes destroyed after the run; versioned JSON report and file evidence uploaded as a CI artifact; no production table or durable run database
**Testing**: Playwright, Node test/Vitest for deterministic runner helpers, Go unit/contract tests, pytest/FastAPI tests, JSON Schema validation in runner tests, fake model and fake issue adapters, repeated seeded-defect acceptance runs
**Target Platform**: Linux GitHub-hosted runner for CI and Docker-capable Windows/Linux/macOS developer machines
**Project Type**: Additive cross-service test tooling for the existing Go API + Vue SPA + Python agent web application
**Performance Goals**: A run ends within 15 minutes; report review locates a finding and evidence in under 5 minutes; deterministic seeded defect is detected in 20/20 validation runs
**Constraints**: Advisory only; no production data or account; max 25 steps, 15 minutes, 20 model calls, 60,000 total model tokens, 100 browser actions, and 3 issue attempts; fail closed on missing token usage; issue creation false by default; every persisted or transmitted observation sanitized first; no unrestricted browsing, shell, filesystem, or arbitrary API action available to the model
**Scale/Scope**: One Chromium worker, one ephemeral stack, one dedicated user, the F013 baseline workflow inventory, one provider per run, one report artifact, and at most three issue requests

## Constitution Check

*GATE: Passed before Phase 0 research. Re-checked after Phase 1 design.*

| Gate | Pre-design | Post-design | Evidence |
|---|---|---|---|
| §0 Hierarchy of Authority | PASS | PASS | The Feature 360 spec controls; F013 artifacts remain the fixture/workflow authority. |
| Principle I (Clear Layered Architecture) | PASS | PASS | The TypeScript harness owns browser orchestration, Go only authenticates/proxies the internal request, and Python owns model inference. No database logic is added to handlers or agent code. |
| Principle II (Service Boundary Separation) | PASS | PASS | The browser harness calls an opt-in internal Go endpoint; Go uses `AgentProxy`; the stateless Python agent resolves a per-request provider contract. The Vue application never calls the agent directly. |
| Principle III (Strict Types and Explicit Contracts) | PASS | PASS | Run configuration and report use JSON Schema; Go↔Python uses OpenAPI/Pydantic/typed Go DTOs; malformed model output terminates safely. |
| Principle IV (Simple Complete Changes) | PASS | PASS | The design reuses Playwright, F013 fixtures, the existing provider factory, Compose, and platform APIs. It adds no agent framework, SDK, database, queue, or production UI. |
| Principle V (Security, Auth, and Privacy by Default) | PASS | PASS | Dedicated ephemeral credentials, denylisted collection, pre-prompt sanitization, masked screenshots, canary scans, and a separate least-privilege issue publisher form the safety boundary. |
| Principle VI (Consistent User Experience) | PASS | PASS | F013 desktop and mobile workflows remain in scope; no production UI change is required. |
| Principle VII (CI, Supply Chain, and Release Integrity) | PASS | PASS | Existing dependency versions are exact in the lockfiles; the direct Playwright declaration becomes exact; Node is fixed at 20.19.0; added actions must use immutable SHAs. |
| Principle VIII (Documented Decisions) | PASS | PASS | This plan and `research.md` record the material architecture, security, CI, deduplication, and rollback decisions. An ADR/decision-inbox entry is an implementation task because the internal testing boundary is cross-cutting. |
| Principle IX (Automated Enforcement Over Manual Memory) | PASS | PASS | Schemas, boundary tests, seeded-defect tests, canary scans, fake publishing, and CI artifact validation enforce the rules. |
| §17 Quality Gate / §21 Definition of Done | PASS | PASS | Existing gates stay intact; implementation must run Go, Vue, agent, schema, F013, and Feature 360 tests and document exact workflow blast radius. |

No constitutional violation or waiver is required.

## Project Structure

### Documentation (this feature)

```text
specs/360-ai-driven-browser-testing/
├── spec.md
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
└── contracts/
    ├── README.md
    ├── exploration-agent.openapi.yaml
    ├── exploration-report.schema.json
    ├── issue-publication.schema.json
    └── run-config.schema.json
```

### Planned Source Code (repository root)

```text
.github/workflows/
└── ai-browser-exploration.yml          # workflow_dispatch + nightly only

src/web/
├── e2e/exploration/
│   ├── cli.ts                           # orchestration and artifact finalization
│   ├── config.ts                        # schema-backed immutable limits
│   ├── budget.ts                        # reserve/perform/reconcile guard
│   ├── browser-driver.ts                # allowlisted Playwright actions
│   ├── workflows.ts                     # adapter over F013 inventory
│   ├── evidence.ts                      # capture, sanitize, hash, persist
│   ├── findings.ts                      # evidence-backed normalization/dedup
│   ├── report.ts                        # report construction/validation
│   ├── publisher.ts                     # opt-in GitHub publication
│   └── __tests__/                       # boundaries, redaction, fake adapters
├── e2e/fixtures/workflow.ts             # extract/reuse shared F013 inventory only
├── playwright.exploration.config.ts      # Chromium, trace off, isolated output
├── package.json
└── package-lock.json

src/api/
├── cmd/exploration-seed/                 # test-only user + F013 DB seed command
├── handlers/internal_exploration.go      # opt-in authenticated internal proxy
├── services/agent_proxy_exploration.go
└── *_test.go

src/agent/
├── app/models/browser_exploration.py     # strict decision DTOs
├── app/routers/internal_exploration.py   # stateless model decision endpoint
├── app/services/browser_exploration.py   # prompt/structured-output boundary
└── tests/

docker-compose.exploration.yml            # current-source, isolated test stack
Dockerfile                                # optional non-root seed build target
Taskfile.yml                              # local orchestration targets
docs/testing.md                           # required operator documentation
```

**Structure Decision**: Keep browser state and budgeting beside the existing
Playwright suite; keep AI inference in the existing Python agent and reach it
through the existing Go-to-agent boundary; keep persistence confined to the
ephemeral application database. This is the smallest architecture that honors
the repository's service-boundary constitution and reuses F013 without making
the production SPA model-aware.

## Architecture and Data Flow

1. The CLI validates `run-config.json`, clamps every requested value to its hard
   maximum, creates a run ID, snapshots limits, and creates an artifact staging
   directory with restrictive permissions.
2. The launcher generates fresh JWT/internal-service/test-user credentials,
   chooses a unique Compose project name, builds current source, creates new
   database/upload volumes and an isolated network, and never imports an
   existing volume or production environment file.
3. A one-shot, test-only Go seed command creates the dedicated account and calls
   `testutil.PersistGoldenCollection`. The normal app and agent start against
   the new volumes; readiness requires app `/healthz` and agent `/ready`.
4. Playwright signs in through the normal application flow. The controller
   selects goals from the F013 inventory and gathers a sanitized observation.
5. The controller posts the sanitized observation and provider/model selection
   to an exploration-only internal Go route. The route is disabled unless
   `AI_BROWSER_EXPLORATION_ENABLED=true`, requires the ephemeral internal
   bearer token, and delegates through `AgentProxy`.
6. Go reads only dedicated exploration-provider environment variables and sends
   a per-request `LLMConfig` to the Python agent. Production `AppSetting`
   provider credentials are never read. Python reuses `get_structured_model`
   and returns one strict decision plus provider-reported usage.
7. `BudgetGuard` reserves capacity before each model call/browser action/issue
   attempt, reconciles actual usage after it, and checks the monotonic deadline
   both before and after work. The model may choose only from a fixed action
   vocabulary and allowed same-origin routes.
8. Evidence is sanitized at capture time, content-hashed, and linked to
   deterministic findings. Model triage is labeled separately and cannot
   create a finding without evidence.
9. The CLI validates `report.json`, generates `summary.md`, scans every staged
   output for canaries/secret patterns, and only then makes the artifact
   available for CI upload.
10. If and only if `createIssues=true` and privacy/schema validation pass, a
    separate publisher with `issues:write` searches for stable fingerprint
    markers and submits at most three sanitized issue requests.
11. A `finally` path always runs `docker compose down -v --remove-orphans`.

## Execution Bounds

| Budget | Default and hard maximum | Reservation and stop behavior |
|---|---:|---|
| Exploration steps | 25 | Reserve before a decision cycle; no next step once exhausted. |
| Wall time | 15 minutes | Monotonic deadline checked around every external operation; abort in-flight provider/network calls. |
| Model calls | 20 | Reserve before dispatch; attempted calls count, including timeout/malformed responses. |
| Combined model tokens | 60,000 | Reconcile provider-reported input + output tokens; missing/unreliable accounting terminates before another call. |
| Browser actions | 100 | Count primitive allowlisted actions, not high-level plans; reserve before execution. |
| Issue attempts | 3 | Count every create attempt, successful or not; duplicate searches do not create an attempt. |

Configuration values may be lower but never higher. The immutable snapshot is
stored in the report. When simultaneous limits are observed, the termination
precedence is `privacy > wall_time > tokens > model_calls > browser_actions >
steps > issue_attempts`; all reached limits are recorded, while the first in
that order is the primary reason.

## Security and Redaction Boundary

- Secrets enter only process environment variables in the launcher, Go proxy,
  and CI publisher step. They are never accepted in run configuration.
- Raw observations are short-lived memory objects. A single sanitizer converts
  them to the only type accepted by prompts, logs, report builders, and issue
  builders.
- Collection is deny-by-default: no cookies, local/session storage, request or
  response headers, authorization data, raw bodies, password fields, binary
  payloads, or unrestricted DOM dumps.
- Network evidence contains only method, sanitized same-origin route template,
  status, resource type, timing, and an allowlisted/truncated error excerpt.
- Screenshots mask password/token inputs and configured sensitive selectors;
  capture is refused when a known canary is visible in rendered text.
- Playwright tracing, video, and HAR are disabled for exploration because they
  can bypass the sanitizer. Console and accessibility text are bounded and
  sanitized before persistence.
- Exact canaries and secret-shaped patterns are scanned across prompt
  transcripts, screenshots (via the pre-capture DOM gate), reports, summaries,
  evidence, logs, and proposed issue bodies. A hit sets privacy status to
  `failed`, prevents publication, and makes the advisory workflow visibly fail.

## Full-Stack Isolation

- Use a dedicated Compose file, not the deployment Compose file that references
  mutable `latest` images and durable named volumes.
- Build app and agent images from the checked-out commit. Use a unique
  `COMPOSE_PROJECT_NAME=ai-browser-${runId}` and fresh anonymous/project-scoped
  volumes for SQLite and uploads.
- Bind the web/API port to `127.0.0.1` on an OS-assigned host port. Keep the
  agent reachable only on the Compose network.
- Reject production-like endpoints and unsafe configuration before startup:
  no external DB path, no remote app base URL, no deployment `.env`, and no
  non-loopback browser target.
- Generate test account and internal secrets per run. Seed only F013 data.
- Teardown is idempotent and unconditional. CI also runs a final cleanup step
  with `if: always()`.

## CI and Dependency Pinning

- Add a separate workflow named `AI Browser Exploration` with only
  `workflow_dispatch` and a nightly `schedule`. It is not referenced by the
  existing `Quality Gate`, branch protection, or deployment workflows.
- Manual inputs: provider, model, workflow scope, bounded limits, and
  `createIssues` (boolean default `false`). Nightly uses the same defaults and
  command.
- Use Node `20.19.0`; `npm ci`; Chromium only. Change the direct
  `@playwright/test` declaration from `^1.63.0` to `1.63.0`; preserve lockfile
  resolutions for `@playwright/test`, `playwright`, and `playwright-core` at
  `1.63.0`.
- Add no model SDK or orchestration dependency. Preserve locked Python
  resolutions: LangChain `1.4.0`, LangChain Anthropic `1.7.2`, LangChain Ollama
  `1.1.0`, and LangGraph `1.2.11`; use `uv sync --locked`.
- Every new GitHub Action is pinned to a reviewed 40-character commit SHA.
  Artifact upload runs with `if: always()` and a finite retention period.
- The normal exploration job uses `contents: read`. A separate conditional
  publication job receives `issues: write` only when explicit opt-in is true
  and consumes only the validated sanitized publication artifact.
- A finding may fail the exploration job for visibility, but the workflow is
  advisory and has no PR/push trigger. Promotion remains governed by FR-027.

## Seeded-Defect Validation

The deterministic acceptance path injects one named fault at the Playwright
network boundary for a known F013 edit request, returning a synthetic HTTP 500.
It does not add a defect to production code. The run uses the real full stack,
normal login, canonical F013 workflow definition, and a `FakeModelAdapter` with
schema-valid usage. Detection is deterministic from the failed request and UI
outcome; the fake model only supplies labeled triage. The finding must link the
route, a masked screenshot, and the network observation. The acceptance command
runs this scenario 20 times and requires 20/20 detection. Issue-path tests inject
a fake publisher and assert zero real GitHub calls.

## Finding and GitHub Deduplication

Within a run, normalize `workflow ID + route template + category + evidence
signature`, hash with SHA-256, and merge observations sharing that fingerprint.
For publication, include:

```text
<!-- aurearia-ai-finding:v1:<sha256-fingerprint> -->
```

Search open and closed repository issues for the exact marker before creation.
A match records `duplicate` plus the existing issue URL; it does not create or
comment. Title similarity and model-based matching are deliberately excluded.
Every unsubmitted, duplicate, failed, and over-limit finding remains in the
report.

## Implementation Phases

### Phase 0 — Research (complete)

- Confirm reuse of F013 fixtures/workflows and current Playwright/provider
  versions.
- Resolve runner placement, provider boundary, dependency policy, full-stack
  isolation, evidence/privacy rules, issue deduplication, CI behavior, and
  rollback.
- Output: `research.md`; no unresolved clarification remains.

### Phase 1 — Design and contracts (complete)

- Define run, step, evidence, finding, report, privacy, and publication entities
  in `data-model.md`.
- Define JSON Schema contracts for configuration/report/publication and the
  internal Go↔Python decision API.
- Define local/CI operation and verification in `quickstart.md`.
- Re-evaluate all constitutional gates (all pass).

### Phase 2 — Implementation planning (stop point)

1. Add strict shared DTOs/schema fixtures and contract drift tests.
2. Add the Python decision service using the existing provider factory and
   reliable usage extraction; add malformed/usage-failure tests.
3. Add the gated Go proxy and test-only seed command; keep handlers thin and
   cover authorization/configuration failures.
4. Implement immutable configuration, budget guard, sanitizer, evidence store,
   deterministic finding builder, report validator, and allowlisted browser
   driver with unit/boundary tests.
5. Adapt the F013 workflow inventory without copying its fixture catalog.
6. Add isolated Compose orchestration, readiness, teardown, and the seeded
   defect acceptance loop.
7. Add report-only and fake-publication acceptance tests, then implement the
   conditional GitHub publisher and exact-marker deduplication.
8. Add Taskfile commands, advisory manual/nightly workflow, immutable action
   pins, artifact retention, and documentation.
9. Run the complete quality gate plus F013 and Feature 360 commands. Record the
   required ADR/decision entry before implementation is considered done.

## Local and CI Commands

Planned canonical commands (fully specified in `quickstart.md`):

```powershell
task ai-browser-exploration -- --config .\run-config.json
task test-ai-browser-exploration
task test-ai-browser-seeded-defect
task test-critical-workflows
```

CI invokes the same Taskfile entry point with a generated configuration file,
never a second orchestration implementation.

## Rollback

The feature is additive and migration-free. Disable the nightly schedule and
manual workflow, revoke dedicated provider/issue secrets, remove the exploration
Taskfile targets and runner/agent/proxy/seed files, and expire artifacts. The
existing F013 suite, production data, application routes, Quality Gate, and
deployment workflows remain usable throughout. Issue publication can be
disabled independently by removing its permission/secret or forcing
`createIssues=false`; a single provider can be removed from the allowlist
without changing the core contract.

## Complexity Tracking

No constitution violations or waivers are identified. The internal
Go-to-Python decision endpoint is justified by Principle II: it avoids a second
model stack in Node and keeps all model inference in the existing agent
boundary. It is disabled outside explicitly configured ephemeral exploration
stacks and adds no public production capability.
