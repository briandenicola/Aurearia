# Feature 360 Implementation Validation

## Scope

This validation covers only T001–T014 (Phases 1–2). No browser, Docker,
deployed service, model provider, GitHub issue, branch, commit, push, or
deployment operation was used.

## Test-first evidence for T005–T009

The contract guards were added before their implementations and failed for the
intended missing-contract reasons.

### TypeScript red run

Command (from `src/web`):

```powershell
node .\node_modules\vitest\vitest.mjs run e2e/exploration/__tests__/config-contract.test.ts e2e/exploration/__tests__/report-contract.test.ts e2e/exploration/__tests__/issue-contract.test.ts
```

Result: **FAIL**, exit code `1`; `3 failed` test files and `0 tests` executed.
Each suite failed import resolution for the intentionally absent
`e2e/exploration/contracts.ts` (`Failed to resolve import "../contracts"`).

### Python red run

Command (from `src/agent`):

```powershell
uv run pytest tests/test_browser_exploration_contract.py -q
```

Result: **FAIL**, exit code `2`; collection stopped with
`ModuleNotFoundError: No module named 'app.models.browser_exploration'`.

### Go red run

Command (from `src/api`):

```powershell
go test ./services -run 'TestExploration' -count=1
```

Result: **FAIL**, exit code `1`; package compilation reported the intentionally
absent `ExplorationDecisionRequest`, `ExplorationObservation`,
`ExplorationDecisionResponse`, `ExplorationActionTarget`,
`ExplorationModelTriage`, `ExplorationModelUsage`,
`ExplorationDecisionSchemaVersion`, and `ExplorationDecisionPath`.

## Passing evidence after T010–T013

### TypeScript contracts and dependency integrity

Commands (from `src/web`):

```powershell
node .\node_modules\typescript\bin\tsc --noEmit --target ES2022 --module preserve --moduleResolution bundler --types node,vitest/globals --skipLibCheck e2e/exploration/contracts.ts e2e/exploration/__tests__/contract-fixtures.ts e2e/exploration/__tests__/dependency-integrity.test.ts e2e/exploration/__tests__/config-contract.test.ts e2e/exploration/__tests__/report-contract.test.ts e2e/exploration/__tests__/issue-contract.test.ts
node .\node_modules\vitest\vitest.mjs run e2e/exploration/__tests__/dependency-integrity.test.ts e2e/exploration/__tests__/config-contract.test.ts e2e/exploration/__tests__/report-contract.test.ts e2e/exploration/__tests__/issue-contract.test.ts
node .\node_modules\eslint\bin\eslint.js e2e/exploration --ext .ts --max-warnings 0
```

Result: **PASS**. TypeScript emitted no diagnostics. Vitest reported `4 passed`
files and `40 passed` tests. ESLint exited `0` with no warnings.

### Python DTO contract

Commands (from `src/agent`):

```powershell
uv run pytest tests/test_browser_exploration_contract.py -q
uv run ruff check app/models/browser_exploration.py tests/test_browser_exploration_contract.py
```

Result: **PASS**. Pytest reported `7 passed`; Ruff reported
`All checks passed!`.

### Go DTO/OpenAPI drift contract

Commands (from `src/api`):

```powershell
go test ./services -run 'TestExploration' -count=1
go vet ./services
```

Result: **PASS**. The targeted services package tests reported `ok`; `go vet`
exited `0` with no diagnostics.

## Dependency and schema integrity

- `npm.cmd install --package-lock-only --ignore-scripts` completed with no
  package resolution changes beyond the direct exact-version declaration.
- `@playwright/test`, `playwright`, and `playwright-core` remain locked to
  `1.63.0`, including their reviewed registry URLs and integrity hashes.
- No browser-agent or model SDK was added.
- The locked Python versions remain LangChain `1.4.0`, LangChain Anthropic
  `1.7.2`, LangChain Ollama `1.1.0`, and LangGraph `1.2.11`.
- The three checked-in JSON Schemas and the OpenAPI contract were not modified.
- Existing `*.har` protection remains in `.gitignore`; Feature 360 local
  configuration, staged artifacts, prompts, evidence, and credential paths are
  additionally ignored.

## Validation environment note

The available Node executable was `v24.14.0`, while `package.json` continues to
declare compatibility with Node `^20.19.0 || >=22.12.0`. The dependency guard
asserts that `20.19.0` compatibility remains declared. Exact Node 20.19.0
execution belongs to later CI/full-gate work and was not required to validate
the Phase 1–2 contracts.

## Phase 3 (T015–T038) validation

### Test-first red evidence for T015–T022

The Phase 3 guards were created before implementation and run from their
service roots.

```powershell
# src/web
node .\node_modules\vitest\vitest.mjs run `
  e2e/exploration/__tests__/budget.test.ts `
  e2e/exploration/__tests__/browser-driver.test.ts `
  e2e/exploration/__tests__/workflows.test.ts `
  e2e/exploration/__tests__/isolation.test.ts `
  e2e/exploration/__tests__/orchestration.test.ts
```

Result: **FAIL**, exit code `1`; all five suites failed module resolution for
the intentionally absent `budget.ts`, `browser-driver.ts`, `workflows.ts`, and
`cli.ts`.

```powershell
# src/agent
uv run pytest tests/test_browser_exploration_service.py -q
```

Result: **FAIL**, exit code `2`; collection failed with
`ModuleNotFoundError: No module named 'app.services'`.

```powershell
# src/api
go test ./handlers ./cmd/exploration-seed `
  -run 'TestInternalExploration|TestValidateExploration|TestSeedCreates' -count=1
```

Result: **FAIL**, exit code `1`; compilation failed on the intentionally absent
handler registration/error values and seed path/seed functions.

### Passing targeted evidence for T015–T035

```powershell
# src/web
node .\node_modules\vitest\vitest.mjs run e2e/exploration/__tests__
npm.cmd run type-check
node .\node_modules\eslint\bin\eslint.js `
  e2e/exploration e2e/fixtures/workflow.ts `
  playwright.exploration.config.ts --ext .ts --max-warnings 0
```

Result: **PASS**. Vitest reported `9 passed` files and `97 passed` tests;
Vue TypeScript checking and ESLint both exited `0`.

```powershell
# src/agent
uv run pytest tests/test_browser_exploration_contract.py `
  tests/test_browser_exploration_service.py -q
uv run ruff check app/models/browser_exploration.py `
  app/services/browser_exploration.py `
  app/routers/internal_exploration.py `
  tests/test_browser_exploration_contract.py `
  tests/test_browser_exploration_service.py
```

Result: **PASS**. Pytest reported `12 passed` (with one upstream
Starlette/httpx deprecation warning); Ruff reported `All checks passed!`.

```powershell
# src/api
go test ./services ./handlers ./cmd/exploration-seed `
  -run 'TestExploration|TestInternalExploration|TestValidateExploration|TestSeedCreates' `
  -count=1
go vet ./services ./handlers ./cmd/exploration-seed
```

Result: **PASS** for all three packages; Go vet exited `0`.

```powershell
# src/web — canonical F013 regression subset
npm.cmd run test:browser -- `
  e2e/workflows/auth.spec.ts e2e/workflows/coin-form.spec.ts
```

Result: **PASS**, `10 passed`. The existing login, add, edit, storage,
tags/sets, image, collection search/filter, and mobile edit workflows remain
authoritative and deterministic.

The Compose YAML also passed a local static parse asserting current-source
builds, loopback random app publication, no agent host port, project-scoped
non-external volumes, and the internal application network.

### Review correction evidence

The Phase 3 review identified seven defects before checkpointing. The
implementation now:

- executes each validated model decision through the fixed browser vocabulary;
- applies one deadline to provisioning, seeding, readiness, browser startup,
  login, model calls, and browser actions, including child-process aborts;
- refuses another model call at exact token exhaustion and reports exact
  terminal limits as bounded;
- blocks click, back, fill, select, and upload navigation that escapes the
  selected same-origin workflow;
- uses the canonical `/edit/1` route;
- preserves Go `context.Canceled` and `context.DeadlineExceeded`; and
- validates internal-network isolation and exact service network membership.

Focused regression validation after these corrections:

```powershell
# src/web
node .\node_modules\vitest\vitest.mjs run `
  e2e/exploration/__tests__/budget.test.ts `
  e2e/exploration/__tests__/browser-driver.test.ts `
  e2e/exploration/__tests__/workflows.test.ts `
  e2e/exploration/__tests__/isolation.test.ts `
  e2e/exploration/__tests__/orchestration.test.ts
node .\node_modules\vue-tsc\bin\vue-tsc.js --build
```

Result: **PASS**, `5 passed` files and `56 passed` tests; strict TypeScript
build exited `0`.

```powershell
# src/api
go test ./handlers `
  -run TestExplorationProxyUsesDedicatedEnvironmentAndPropagatesCancellation `
  -count=1
go test ./services -run Exploration -count=1
```

Result: **PASS**. The first broad handler run exposed a test-only HTTP server
teardown hang in the new deadline case. Replacing the indefinitely blocked
handler with a bounded slow response made the cancellation race deterministic;
both focused packages then passed.

### Ephemeral stack blocker (T036–T038 remain incomplete)

The approved low-budget fake-model lifecycle was attempted with one
`login-session` workflow, 1 step, 1 model call, 10 tokens, 2 browser actions,
zero issue attempts, and `AI_BROWSER_FAKE_MODEL=true`.

Result: **BLOCKED before provisioning**. This host has no `docker` executable
on `PATH`; Node reported `Error: spawn docker ENOENT` while executing the
pre-start `docker compose ... config --format json` inspection. No image,
container, network, or volume could have been created.

The CLI's unconditional `finally` path ran and recorded the exact cleanup
attempt for project `ai-browser-mu6yfkcw-706ac9b4`:

```text
docker compose -f docker-compose.exploration.yml \
  -p ai-browser-mu6yfkcw-706ac9b4 down -v --remove-orphans
```

That cleanup command also could not execute because the same Docker executable
is unavailable. The ignored diagnostic files are under
`.artifacts/ai-browser/ai-browser-mu6yfkcw-706ac9b4/`; no broad directory
deletion was performed. T036, T037, and T038 are deliberately left unchecked
until a Docker-capable host can render the resolved Compose targets, start the
fresh stack, execute the fake-model run, inspect targets, and confirm teardown.

### Docker-capable acceptance location

The project owner confirmed this development machine will not have Docker.
Runtime acceptance therefore runs in the separately named advisory
`.github/workflows/ai-browser-exploration.yml` workflow on a GitHub-hosted
Linux runner. The local runner remains responsible for static isolation,
contract, type, lint, and unit checks.

The CI acceptance uses a deterministic fake model and one low-budget F013
workflow. It supplies a known unique Compose project name, runs the same CLI,
always executes `down -v --remove-orphans`, asserts that no containers,
networks, or volumes with the exact project label remain, and uploads evidence
for 14 days. The workflow has only `contents: read`, is advisory, and uses
immutable 40-character action pins.

GitHub does not permit manual dispatch of a newly added workflow until the file
exists on the default branch. A path-scoped `beta` push trigger therefore
bootstraps acceptance without merging to `main`; the workflow remains separate
from and non-blocking to the Quality Gate.

The workflow policy guard was tamper-tested by removing
`--remove-orphans`: one of four tests failed on the exact missing cleanup
requirement. After restoration, all four policy tests passed; the Docker
runtime test remains skipped locally by design and must pass on GitHub before
T036-T038 can be completed.

The first hosted run, Actions run `35350617252`, built the stack but exposed
two CI portability defects before browser execution: Compose did not create a
host mapping when `published: 0` was explicit, and the workflow's redundant
always-run cleanup could not parse required Compose variables. The exact
project resource assertion still passed. The configuration now omits
`published` so Compose selects an ephemeral port and supplies non-sensitive
cleanup-only interpolation values to the outer teardown; runtime acceptance
remains pending the corrected hosted rerun.

The second hosted run, Actions run `35351151858`, proved the outer teardown and
resource assertion pass, but Docker Compose still created no host binding when
the long port syntax omitted `published`. The stack now uses Compose's
documented short random-port form, `127.0.0.1::8080`; the isolation guard
accepts only that exact string or an equivalent resolved long form.

The third hosted run, Actions run `35353345604`, reached the same missing-port
symptom even with the short syntax, while cleanup and the exact resource
assertion again passed. Run `35355272907` then proved the app and agent health
checks pass, isolating the defect to Compose's random-port publication. The CLI
now obtains an available loopback port from the OS (or validates
`AI_BROWSER_APP_PORT` when supplied), passes that value through environment
configuration, and verifies the resolved Compose mapping exactly before
startup. Readiness failures still include bounded Compose status and the last
100 app log lines.

Run `35356367491` on commit `84d7bfd7` confirmed explicit loopback port
allocation and both container health checks succeeded, but the host runner
could not reach `http://127.0.0.1:44853/healthz`. Run `35358932528` on commit
`1196cdc0` reproduced the same boundary at port `36295`. Both runs completed
the exact teardown and zero-resource assertion, proving the remaining failure
was host ingress rather than application startup or cleanup.

The app had been attached only to the Compose network declared
`internal: true`. That externally isolated network is retained for app-agent
communication, while the app now also joins a project-scoped
`browser-ingress` bridge used solely for its published loopback port. The
ingress bridge fixes host browser reachability without enabling container
egress: IP masquerading is disabled and the default host binding is
`127.0.0.1`. The agent and seed services remain excluded from this network.
The isolation guard rejects missing or altered ingress controls.

Readiness failure output now includes the last host probe result, resolved
published port, Compose status, runtime `NetworkSettings.Ports`, and bounded
app logs. This replaces blind retries with enough evidence to distinguish a
mapping failure, host transport failure, and application response failure.

Local validation after the ingress correction:

```powershell
# src/web
node .\node_modules\vitest\vitest.mjs run e2e\exploration\__tests__
node .\node_modules\eslint\bin\eslint.js e2e\exploration `
  playwright.exploration.config.ts --ext .ts --max-warnings 0
node .\node_modules\vue-tsc\bin\vue-tsc.js --build
```

Result: **PASS**. Vitest reported `10 passed` files, `105 passed` tests, and
the Docker-only runtime test skipped locally; ESLint and strict TypeScript
both exited `0`. Before the full run, the new expected Compose shape failed
against the old single-network guard. The restored ingress guard was also
tamper-tested by removing its no-masquerade assertion: the exact
`rejects masquerading browser ingress` test failed, then passed after the
assertion was restored.

Hosted fake-model lifecycle acceptance remains pending the corrected rerun;
T036-T038 stay incomplete until browser execution, teardown, and zero leaked
resources all pass together.
