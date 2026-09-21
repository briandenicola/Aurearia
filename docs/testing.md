# Testing Strategy

This document is the canonical testing strategy for Aurearia. It explains what we test, what we intentionally do not test, and how contributors should add new tests across the Go API, Vue PWA, and Python agent.

ADR 0019's proportional-policy changes were accepted via PR #734 under section
22. Section 6 documents the shared P3 Taskfile gates used locally and in CI.
Workflow jobs are not proof that live branch protections require them.

## 1. Testing Philosophy

**Confidence over coverage.** We optimize for regressions affecting users, data
integrity, and architecture, not percentages. Principle IX favors automated
enforcement; §21.6 requires exact-path regression evidence and §21.9 requires at
least one unit test for each new service method.

**Fast feedback first.** Use targeted checks early, then applicable completion
gates. Strict type/build parity is Principle III; proportional changes are
Principle IV. Targeted success alone does not satisfy §17.

**Test the contract, not the implementation.** Handler tests exercise HTTP status codes and JSON payloads, store tests assert state transitions, and agent tests validate schemas, routing helpers, and endpoint contracts instead of private internals. This applies Principle III (Strict Types and Explicit Contracts).

**Architecture tests are constitutional tests.** `src/api/architecture_test.go`
enforces Principle I via Principle IX. Passing feature tests cannot compensate
for a failing architectural guard.

**Integration evidence follows risk.** Deterministic cross-service seams,
DB-backed tests, and browser workflows catch failures isolated unit mocks cannot.
Use them where the changed contract requires them. Model-quality evaluation
remains distinct from deterministic orchestration tests.

## 2. Test Surface Inventory

| Service | Layer | Tool | Location | Run command | What it tests |
|---|---|---|---|---|---|
| Go API | Architecture | `go test` | `src/api/architecture_test.go` | `cd src/api && go test -v -run "TestNoDirectDatabaseImports|TestHandlersDoNotUseRawSQL|TestPackageImportMatrix" .` | 1 file / 3 tests enforcing DI-only database access, no raw SQL in handlers, and the package import matrix. |
| Go API | Unit + package-level behavior | `go test` | `src/api/{handlers,middleware,repository,services}/*_test.go` | `cd src/api && go test -v ./...` | Handler, middleware, repository, and service behavior using HTTP test routers and in-memory SQLite. |
| Go API | Dedicated integration / E2E | `go test` | `src/api/integration/*_test.go` | `cd src/api && go test -v ./integration/...` | Full DB-backed service/handler workflow tests (Numista compatibility/performance/security/workflows). Not a cross-process suite by default. |
| Go API | Go↔Python seam (cross-process) | `go test -tags=seam` | `src/api/integration/deep_identification_seam_test.go` | See §10 below | T106 (spec 351): boots the real Python agent service and drives the real `DeepIdentificationPipelineRunner` over a real HTTP/SSE round trip. CI-excluded by build tag + env var; see §10. |
| Vue Web | Type checking | `vue-tsc` | `src/web/package.json`, `src/web/src/**` | `cd src/web && npm run type-check` | CI runs the package script (`vue-tsc --build`); `--noEmit` is not a substitute. |
| Vue Web | Lint | ESLint | `src/web/package.json`, `src/web/eslint.config.*` | `cd src/web && npm run lint` | CI runs zero-warning lint (`--max-warnings 0`); never replace it with `--quiet`. |
| Vue Web | Unit / component / store / API tests | Node test runner + Vitest | `src/web/scripts/`, `src/web/src/**/__tests__/*` | `cd src/web && npm run test` | The script runs asset-script tests plus Vitest; calling Vitest alone omits part of the gate. |
| Vue Web | Build parity | `vite` + `vue-tsc --build` | `src/web/package.json` | `cd src/web && npm run build` | Production build gate. This is stricter than `vue-tsc --noEmit`; nullable props and indexed access that pass locally can still fail here. |
| Vue Web | Browser workflows | Playwright | `src/web/e2e/` | `task test-critical-workflows` | Deterministic F013 smoke coverage for login/session setup, manual add coin, edit-one-field, storage-location set/clear, tags/sets, upload/delete image, collection search/filter, and mobile viewport edit workflows with mocked API routes and golden fixtures. |
| Python Agent | Lint | Ruff | `src/agent/app/`, `src/agent/tests/`, `src/agent/pyproject.toml` | See §6 locked environment | Import order, correctness, and style rules for the deterministic Python surface. |
| Python Agent | Unit / contract tests | pytest | `src/agent/tests/test_*.py` | See §6 locked environment | Request validation, schemas, provider/routing behavior, streaming, and deterministic orchestration. |
| Python Agent | Static type checking | None configured today | `src/agent/pyproject.toml` | N/A | No `mypy` or `pyright` config is present; schema validation and pytest carry the current contract burden. |

Notes:
- `src/api/integration/` contains a dedicated integration package in addition to package-local HTTP tests.
- Browser workflow tests live in `src/web/e2e/workflows/` and run with Playwright.
- Test counts change; use current runner output instead of historical counts as gate evidence.
- `src/web/e2e/screenshots/` is a separate, deliberately-real-network screenshot tool
  (`npm run screenshots:beta`, documented in `src/web/README.md`) for capturing
  production-like tour screenshots against a real deployment. It requires beta
  credentials via environment variables, is excluded from `npm run test:browser` via
  `testIgnore`, and is not part of the F013 deterministic suite.

## 3. What we DON'T test

- We do **not** test SQLite, GORM, Gin, Vitest, FastAPI, or LangGraph themselves; we test our usage of them.
- We do **not** hit third-party services in automated tests. Anthropic, Ollama, SearXNG, NumisBids, CNG Auctions, and Pushover should be mocked or stubbed at the service boundary.
- Auction scraper tests must use sanitized fixtures under `src/api/services/testdata/` or `httptest` stubs. HAR captures and browser exports from real provider sessions are sensitive because they can include cookies, headers, account identifiers, and bid data; `*.har` is ignored by `.gitignore` and must not be committed.
- Browser workflow tests are deterministic smoke checks with mocked APIs; they do **not** require a live backend or production data.
- We do **not** attempt deterministic end-to-end assertions for LLM quality. For agent work we test input validation, parsing, retry behavior, routing helpers, and schema-shaped outputs.
- We do **not** add tests for trivial getters or thin pass-through code just to raise coverage numbers.

## 4. Test Pyramid Shape

```text
        Deterministic browser smoke (F013 Playwright workflows)
      Integration / cross-package / DB-backed flows
   Unit + contract tests per package, component, store, route, helper
Architecture tests and type/lint gates (cheapest, fastest, most structural)
```

For AI features, the top of the pyramid is intentionally shallow: we test deterministic orchestration and schema boundaries, then rely on runtime telemetry and human review for model quality.

Deep Analysis follows that rule across all three services. Python tests exercise
the router override, bounded fan-out, typed provider failures, deterministic
disagreement detection, and Pydantic stream frames without live LLM or provider
calls. Go `httptest` streams feed those Python-shaped frames through the real
pipeline runner to verify public SSE translation, persisted proposals,
owner-scoped lifecycle behavior, provider telemetry, and hint cleanup. Vue
component and mocked Playwright tests verify reconnect, exact coverage states,
confirm-gated draft/coin writes, and the absence of hint artifacts from
result-consuming UI.

## 5. Adding a New Test

### Go API

- Put the test beside the package under test as `*_test.go`; do not create a separate test tree.
- Follow the helper pattern in `src/api/services/auth_service_test.go` and `src/api/handlers/auth_handler_test.go`: small setup functions marked with `t.Helper()`.
- Use real in-memory SQLite (`glebarez/sqlite` + `gorm.Open(...":memory:"...)`) when repository behavior, transactions, or auth persistence matter.
- Use `httptest` + Gin routers for handler and middleware tests; assert status codes, JSON shape, and authorization behavior.
- Prefer table-driven tests for pure logic and parser-style code; `src/api/services/valuation_parser_test.go` is the model.
- Mock only true external boundaries (agent proxy, networked services, notifications). Prefer real repos/services inside the process.
- If a change touches the layer rules, extend `src/api/architecture_test.go` rather than burying the rule in prose.

### Vue Web

- Put tests under a nearby `__tests__/` folder, as in `src/web/src/components/__tests__/CollectionPagination.test.ts` or `src/web/src/stores/__tests__/auth.test.ts`.
- Test public behavior: rendered text, emitted events, store state, request payloads, or design-token guarantees—not component internals.
- Use Vitest plus Vue Test Utils for mounted components and `vi.mock()` for boundary modules such as `@/api/client`.
- Stub browser globals explicitly with `vi.stubGlobal`, as shown in `src/web/src/api/__tests__/client.test.ts` and `src/web/src/stores/__tests__/auth.test.ts`.
- Keep fixtures small and inline until multiple files need the same data.
- Source-scanning tests are acceptable for structural UI rules when rendering is unnecessary; see `src/web/src/__tests__/design-tokens.test.ts` and `src/web/src/pages/__tests__/CollectionPage.test.ts`.
- Finish with the applicable §6 frontend gates, including zero-warning lint,
  strict type-check, full package tests, and production build.

### Python Agent

- Put tests in `src/agent/tests/` as `test_*.py`.
- Use pytest fixtures to remove nondeterminism or waiting; `src/agent/tests/test_retry.py` patches retry delays to zero.
- Use FastAPI `TestClient` for route-level contracts, as in `src/agent/tests/test_api.py`.
- Stub LLM/model calls with `AsyncMock` or direct helper invocation rather than making live provider calls.
- Favor testing deterministic helpers directly—`parse_verdicts` in `src/agent/tests/test_availability.py` and `_build_coin_show_location_context` in `src/agent/tests/test_supervisor_coin_shows_location.py` are good examples.
- Assert structured outputs and schema defaults, not prose quality.
- Keep provider/network behavior at the boundary; real Anthropic/Ollama/SearXNG traffic does not belong in CI.

## 6. Running tests locally vs. CI

Use targeted checks for feedback and all applicable completion checks before
claiming verification. Pure docs need document/link/consistency review and any
existing doc checks. Executable prompts, policy, scripts, and workflow changes
need relevant behavior/fixture evidence; Markdown is not an automatic exemption.

The [Taskfile](../Taskfile.yml) is the shared completion entry point.
[`ci.yml`](../.github/workflows/ci.yml) calls the same targets, without a
missing-linter fallback. Run from the repository root in an already prepared
environment; select every affected layer rather than claiming unrelated skips.

| Completion target | Commands / effect |
|-------------------|-------------------|
| `task check:go` | `go build ./...`, `go vet ./...`, `go test -v ./...` in `src/api` |
| `task check:web` | `npm run lint`, `npm run type-check`, full `npm run test`, `npm run build` in `src/web` |
| `task check:agent` | Require prepared `.venv`, verify lock AND environment with `uv sync --locked --check --offline --extra dev`, then `uv run --no-sync --offline` Ruff and full pytest; Python downloads disabled |
| `task check:openapi` | Run the pinned installed generator, sync version and snapshots, fail on generated tracked-file drift; **writes files** |
| `task check:delivery` | Node fixture/negative tests, PowerShell SpecKit selection regression, then offline governance check |
| `task check:governance` | Read-only, offline active-governance checks; errors fail, warnings do not |
| `task check` | All base completion targets above, sequentially; not a replacement for additional browser/race/security/compatibility/release jobs |
| `task test-critical-workflows` | Affected browser workflows (`npm run test:browser`) |

**Preparation is separate and requires authorization.** Match toolchains to
manifests/workflows. CI uses Task 3.44.0 via a SHA-pinned setup action, Node
24.15.0, Go from `src/api/go.mod`, and Python 3.12 with uv 0.11.22.
The delivery scripts themselves need only Node built-ins; their fixture suite
also needs Task, Git, npm and PowerShell 7 (`pwsh`). No dependency installation
occurs in completion recipes.

Authorized setup targets are `task setup:go` (`go mod download` using the installed
toolchain), `task setup:web` (`npm ci`), `task setup:agent`
(`uv sync --locked --extra dev`), and `task setup:openapi` (swag v1.16.6).
They require the corresponding tools to be installed already. The legacy
`build-agent` target is explicitly a setup alias, not compilation evidence.
`build-web` and `run-web` no longer implicitly run npm install.
Agent lint/tests use the prepared project environment, not system Python.
The agent's `uv sync --check` only verifies; it does not synchronize/install.
Go completion/race/OpenAPI commands disable toolchain switching and module
downloads (`GOTOOLCHAIN=local`, `GOPROXY=off`), so missing setup fails explicitly.
These are command-local assignments (and explicit OpenAPI child-process values),
not overridable Task `env` defaults. Regression probes exercise conflicting
inherited values, including `CGO_ENABLED=0` and Python-download overrides.
Missing tools, dependencies or execution permission mean incomplete evidence;
do not install implicitly or weaken the command.

`task openapi` now only generates; it does not install. Both OpenAPI modes write
the version annotation and generated artifacts. `task check:openapi` additionally
compares against the index with `git diff --exit-code`; review/stage intentional
snapshot changes and rerun. CI starts from a clean index and fails on drift.
Neither command is a read-only audit.

**Windows:** the Taskfile selects `npm.cmd` so PowerShell's npm.ps1 policy does
not change the gate. Use `npm.cmd` for direct package commands as well.
Use Windows paths such as `Set-Location .\src\web`; each fresh shell starts in the
repository root. Do not replace the package test script with a Vitest-only call.

**CI/release evidence:** the separate race job calls `task test-race`
(`CGO_ENABLED=1`, `go test -race ./...`). Preserve generated OpenAPI consistency,
security scans, compatibility and
container jobs. Use appropriate runners/toolchains; unavailable local checks
remain pending until matching CI evidence exists. Cite the exact tested SHA.
Live required-context configuration is separate from jobs merely existing.

Fast feedback remains targeted tests. Completion is the applicable shared
targets above. CI/release adds all applicable runner-only and release checks.
Delivery fixtures run on both Windows and Ubuntu; this does not substitute for
actual application execution in the language jobs.
Record commands, outcomes, manual exceptions, and commit/dirty-tree identity.
Later changes invalidate applicable previous results.

### Governance checker boundaries

[The checker](../scripts/delivery/check-governance.mjs) defines its active surface
explicitly: repository instructions, PR/contributor/testing guidance, active
decisions/current pointer, `.github/agents`, `.github/prompts`,
`.github/instructions`, `.specify/templates`, and current role charters/histories.
It does not recursively lint archives, session logs, ADR requirement bodies,
landed specifications or backlog prose as current policy.

Blocking rules check concrete Markdown link targets (including reference-style
definitions), principle identifiers against the constitution, ADR header/index
status/target agreement, current-work structure/targets, and required native
skill `name`/`description` metadata under `.github/skills` and `.agents/skills`.
Backticked examples/optional runtime paths and placeholder templates are not
treated as file links. Remote URLs are not fetched. Sparse-excluded tracked
reference targets are reported as warnings; an ordinary missing target is an
error. Diagnostics include file, line, rule code and severity; errors exit 1.

The current pointer requires scalar `updated_at` (real YYYY-MM-DD), `focus_area`,
`owner`, `work_artifact` and `tasks_artifact` frontmatter. Artifact paths are
canonical repository-relative existing Markdown files under docs/specs, or
`https://github.com/briandenicola/Aurearia/issues/NUMBER` for bounded issue work
(URL syntax only; remote existence/authorization is not checked offline).
Feature selection requires matching `spec.md`/`tasks.md`; a process plan or issue
may serve both roles. No new feature spec is required for a small issue repair.
Names/descriptions accept plain, quoted or block scalar text; the checker
validates required metadata, not the entire YAML language.

Size limits and Proposed ADR lifecycle questions start as warnings. New malformed
status/index links remain errors; there is no blanket waiver for new drift.
The checker cannot establish prose correctness, owner authorization, historical
review clearance, native tool discovery, independence or release acceptance.
Those remain evidence-backed human/reviewer decisions. P4 still owns native
skill migration; this phase installs no skill packages.

Fixtures deliberately remove targets, corrupt state/status/metadata, introduce
obsolete principles and disable blocking diagnostics. Real Task invocations
exercise lint failure propagation and missing-script failure without installing
dependencies. The SpecKit regression runs explicit selections and a throwaway
copy restoring the original stdout/Boolean bug; it never mutates a real feature.

## 7. Coverage philosophy

We do **not** gate PRs on a repo-wide coverage percentage. Coverage is a signal for blind spots, not a target to game.

Constitution §21.9 requires **"every new service method has ≥ 1 unit test."**
Section 21.6 separately requires exact-path regression evidence. Use coverage
output diagnostically, never a single percentage as proof of safety.

## 8. F013 golden collection fixtures

F013 defines shared "golden" coin fixtures for deterministic backend and frontend workflow tests (Principle III, Principle IV, Principle IX). The fixture matrix covers:

| Fixture | Required traits |
|---|---|
| `roman-denarius-core` | Roman |
| `greek-tetradrachm-valued` | Greek, valued |
| `byzantine-solidus-set-member` | Byzantine, set-member |
| `wishlist-aureus-target` | Roman, wishlist, valued |
| `sold-sestertius-archive` | Roman, sold |
| `private-provincial-bronze` | Roman, private, legacy/custom-era |
| `tagged-follis-storage` | Roman, tagged, storage-location |
| `image-heavy-drachm` | Greek, image-heavy |
| `reference-rich-denarius` | Roman, reference-rich, set-member |

### Backend usage

- Import `src/api/testutil`.
- Use `BuildGoldenCoinFixture(name, userID)`, named helpers such as `BuildTaggedFollisStorage(userID)`, or `BuildGoldenCoinFixtures(userID)` for in-memory model tests.
- Use `PersistGoldenCollection(db, userID)` when a migrated test database needs coins plus valid storage locations, tags, sets, set memberships, images, and references.
- Formal coverage check: `src/api/testutil/coin_fixtures_test.go` verifies required traits, clone safety, and persisted associations.

### Frontend usage

- Import from `src/web/src/test/fixtures`.
- Use `buildGoldenCoinFixture(name)`, named helpers such as `buildImageHeavyDrachm()`, or `buildGoldenCoinFixtures()` for Vitest/component data.
- Use `buildTestStorageLocations()`, `buildTestTags()`, and `buildTestCoinSets()` when a test needs the related lookup catalogs.
- Formal coverage check: `src/web/src/test/fixtures/coins.test.ts` verifies required traits, clone safety, and key associations.

### Critical browser workflow command

Run the F013 browser workflow suite from the repository root:

```bash
task test-critical-workflows
```

The Taskfile target runs `npm run test:browser` in `src/web`, which starts the Vite dev server through `playwright.config.ts`. No live API, database, or production data is required: `src/web/e2e/fixtures/workflow.ts` installs an authenticated test session, mocks `/api/*` routes, and seeds the test with golden fixtures from `src/web/src/test/fixtures`.

Current workflow coverage:

- login stores the authenticated session and opens the collection
- authenticated setup helper opens protected workflows without a live backend
- manual add-coin save payload
- edit-one-field payload preservation
- storage-location change and clear behavior
- detail-page tag and set add/remove behavior
- upload/delete image routes and detail rendering
- collection search and category/tag filter queries
- mobile viewport edit save flow without desktop-only controls

Current F013 validation commands:

```bash
# From the repository root
task test-critical-workflows

# From src/api/
go test -v ./...
go vet ./...

# From src/web/
npm.cmd run type-check
npm.cmd test -- --run
npm.cmd run test:browser
```

The browser workflow suite uses Playwright with mocked API routes and golden fixtures from `src/web/src/test/fixtures`.

## 9. Go↔Python deep-identification seam test (T106)

Every deep-identification test elsewhere in this repo (`src/api/services/deep_identification_pipeline_runner_stream_test.go` and the Python `tests/` suite) drives its own side against hand-written, convention-only fixtures for the *other* side's wire shape. That is exactly the gap behind the 080e598 production outage: both sides were internally consistent and both were wrong about each other. `src/api/integration/deep_identification_seam_test.go` closes that gap by booting the **real** `uvicorn app.main:app` Python process and driving the **real**, exported `DeepIdentificationPipelineRunner.Run` (over the real `AgentProxy.StreamDeepIdentification` HTTP/SSE client) against it — no fixture on either side.

### Why it is excluded from unattended CI

The test needs a real Python interpreter/venv and takes real wall-clock time (worker startup + the LLM retry/backoff ladder), so it is guarded twice:

1. **Build tag**: the file starts with `//go:build seam`. `go build ./...`, `go vet ./...`, and `go test ./...` never see or compile it without `-tags=seam`.
2. **Env var**: even built with the tag, the test calls `t.Skip` unless `DEEP_SEAM_TEST=1` is set.

Both are required to actually run it — this is deliberate defense in depth.

### How to run it

Prerequisites:
- `src/agent/.venv` must exist and be able to import `uvicorn`, `langchain_ollama`, and `langchain_anthropic` (run `pip install -e ".[dev]"` from `src/agent` first if not).
- No live LLM credentials or network access are required — see the LLM tradeoff below.

```powershell
cd src/api
$env:DEEP_SEAM_TEST = "1"
go test -tags=seam -run TestDeepIdentificationSeam -v ./integration/...
```

On Linux/macOS:

```bash
cd src/api
DEEP_SEAM_TEST=1 go test -tags=seam -run TestDeepIdentificationSeam -v ./integration/...
```

If your Python interpreter is not at `src/agent/.venv/Scripts/python.exe` (Windows) or `src/agent/.venv/bin/python` (POSIX), point `DEEP_SEAM_PYTHON` at it.

**Expected runtime**: roughly 10-20 seconds (Python process startup plus the vision node's LLM retry/backoff ladder against a deliberately unreachable endpoint — see below). If the Python service fails to become healthy within 30 seconds, or the venv/interpreter cannot be found, the test skips or fails with the captured stdout/stderr from the agent process.

### The LLM tradeoff (documented, not hidden)

The test configures the LLM provider as Ollama pointed at a local TCP port nothing is listening on. This is **not** a stub of the seam: FR-006/FR-040 already require every LLM call site in the deep-identification pipeline (vision hypothesis, evaluator disagreement summary, synthesis narrative) to degrade to a deterministic fallback on any LLM failure, never to raise. Pointing at an unreachable endpoint exercises a real call that fails fast and is handled by the pipeline's own documented resilience path — so the test runs unattended with no API key, no external network egress, and no nondeterministic model output, while still genuinely exercising the vision node's real structured-output call path.

Similarly, the provider catalog is left at its real production default, but `tools_base_url` is left empty — the exact same code path production uses when the tools client is unset — so `numista`/`nomisma` settle immediately as `unconfigured` with zero upstream calls, while `ngc`/`rpc` (never automated) and `ocre` (disabled by default) still run their real, always-network-free provider nodes. No `numista.org`/`nomisma.org` network traffic occurs.

Full rationale: `.squad/decisions/inbox/maximus-seam-test.md`.

## 10. Coin Copilot contract and operations testing

Feature 359 uses deterministic seam tests instead of live-model assertions.
Required coverage spans:

- Go state transitions, owner scoping, idempotent start/resume, cancellation
  races, stale recovery, retention, bounded payloads, token usage, and exactly
  one terminal event;
- execution-credential signature, expiry, revocation, owner/run/execution/tool
  binding, and rejection of all write/arbitrary tools;
- Python strict schemas, model capability preflight, malformed calls,
  prompt-injection-as-data, iteration/tool/time exhaustion, sequential tool
  execution, payload truncation, and provider-reported token accounting;
- Vue feature-off/unsupported-model legacy fallback, progress,
  clarification/resume, cancellation, reconnect/de-duplication, and
  desktop/PWA rendering;
- route/OpenAPI and shared Go/Python fixture drift.

The timeout matrix must cover the 120-second default and 150-second maximum.
Values above 150 are invalid because the execution timeout plus the 30-second
credential buffer must not exceed the token's absolute 180-second TTL.
Dollar-cost enforcement is intentionally absent; tests should assert that
reliable input/output token counts remain observable and that iteration,
tool-call, wall-clock, sequential-concurrency, and payload limits remain
enforced.

Use mocked provider responses and fake internal SSE streams; do not contact
Anthropic, Ollama, or other external services. The manual
[`Feature 359 quickstart`](../specs/359-coin-copilot-harness/quickstart.md)
validates multi-tool operation, replay, pause/resume, cancellation, fallback,
owner isolation, privacy, and retention. Record results in the PR rather than
this document.

Documentation/API drift verification:

```powershell
task openapi
git diff --exit-code -- src\api\docs\docs.go src\api\docs\swagger.json src\api\docs\swagger.yaml docs\openapi.json
Push-Location src\api
go test . -run TestRegisteredAPIRoutesAreDocumentedInOpenAPI -count=1
Pop-Location
```

Feature-specific automated suites may be run before the full §17 gate:

```powershell
Push-Location src\api
go test ./handlers ./repository ./services -run CoinCopilot -count=1
Pop-Location

Push-Location src\agent
pytest tests/test_coin_copilot_contract.py tests/test_coin_copilot_capabilities.py tests/test_coin_copilot_security.py tests/test_coin_copilot_harness.py -v
Pop-Location

Push-Location src\web
npx vitest run src/composables/__tests__/useCoinCopilot.test.ts src/components/__tests__/CoinSearchChat.test.ts src/components/admin/__tests__/AdminSystemSection.coin-copilot.test.ts
Pop-Location
```

These commands document the expected validation path; passing results must not
be claimed unless the commands were actually executed.

## 11. Cross-references

- Constitution: [`../.specify/memory/constitution.md`](../.specify/memory/constitution.md) (especially Principle IX, §17, and §21)
- System architecture: [`ARCHITECTURE.md`](ARCHITECTURE.md)
- Go API README: [`../src/api/README.md`](../src/api/README.md)
- Vue frontend README: [`../src/web/README.md`](../src/web/README.md)
- Python agent README: [`../src/agent/README.md`](../src/agent/README.md)
- Quality Gate workflow: [`../.github/workflows/ci.yml`](../.github/workflows/ci.yml)
- Task runner: [`../Taskfile.yml`](../Taskfile.yml)

TODOs:
- Python static type checking is still absent. Any owner-approved backlog addition must use a newly allocated identifier; F012 already belongs to a different feature.
# Structured storage tray regressions

```powershell
Push-Location src/api
go test ./database -run TestFeature358 -count=1
go test ./services ./repository ./handlers -run 'Test.*Storage(Location|Slot)|Test.*Bulk.*Location|Test.*Duplicate' -count=1
Pop-Location

Push-Location src/web
npx vitest run src/components/__tests__/StorageTrayGrid.test.ts src/pages/__tests__/StorageTraysPage.test.ts src/components/__tests__/CoinForm.test.ts src/components/__tests__/BulkLocationPickerModal.test.ts src/components/__tests__/MuseumTray.test.ts src/components/__tests__/MuseumTrayWell.test.ts src/pages/__tests__/TrayViewPage.test.ts
npx playwright test e2e/workflows/storage-trays.spec.ts e2e/workflows/coin-form.spec.ts e2e/workflows/tray.spec.ts
Pop-Location
```

The full release gate also requires `go build ./...`, `go vet ./...`,
`go test ./...`, `task test-race`, frontend lint/type-check/test/build, and
`task openapi`.
