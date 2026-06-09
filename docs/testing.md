# Testing Strategy

This document is the canonical testing strategy for Ancient Coins. It explains what we test, what we intentionally do not test, and how contributors should add new tests across the Go API, Vue PWA, and Python agent.

## 1. Testing Philosophy

**Confidence over coverage.** We optimize for catching regressions in the paths that can break users, data integrity, or architectural rules, not for inflating a percentage. This follows Constitution Principle X and §21.6: tests exist to prove important behavior, and every new service method must have at least one unit test.

**Fast feedback first.** The cheapest checks run earliest: Go architecture tests, TypeScript checks, Ruff, and pytest/Vitest before any manual browser poking. This matches Principle IV and §17, which treat type/build parity and automated checks as merge gates.

**Test the contract, not the implementation.** Handler tests exercise HTTP status codes and JSON payloads, store tests assert state transitions, and agent tests validate schemas, routing helpers, and endpoint contracts instead of private internals. This is the practical testing expression of Principle VII (schema-driven contracts).

**Architecture tests are constitutional tests.** `src/api/architecture_test.go` is not "extra credit"; it enforces the Go layer rules from Principle I via Principle X. If those tests fail, the codebase is drifting away from the constitution even if feature tests still pass.

**End-to-end is rare and deliberate.** Cross-service and browser-wide tests are expensive, brittle, and especially weak for nondeterministic AI behavior. Per Principles III and VI, we instead test deterministic seams—HTTP contracts, repository behavior, parsing, schema validation, and orchestration helpers—and use production telemetry for model quality.

## 2. Test Surface Inventory

| Service | Layer | Tool | Location | Run command | What it tests |
|---|---|---|---|---|---|
| Go API | Architecture | `go test` | `src/api/architecture_test.go` | `cd src/api && go test -v -run "TestNoDirectDatabaseImports|TestHandlersDoNotUseRawSQL|TestPackageImportMatrix" .` | 1 file / 3 tests enforcing DI-only database access, no raw SQL in handlers, and the package import matrix. |
| Go API | Unit + package-level behavior | `go test` | `src/api/{handlers,middleware,repository,services}/*_test.go` | `cd src/api && go test -v ./...` | 14 files / 115 tests: handlers 30, middleware 10, repository 23, services 52. Uses `httptest`, Gin test routers, and in-memory SQLite. |
| Go API | Dedicated integration / E2E | None today | None today | N/A | No cross-process backend integration suite exists. The closest API coverage is handler HTTP tests and DB-backed repository/service tests. |
| Vue Web | Type checking | `vue-tsc` | `src/web/package.json`, `src/web/src/**` | `cd src/web && npx vue-tsc --noEmit` | CI gate for compile-time contract safety. Use it, but do not stop there. |
| Vue Web | Lint | ESLint | `src/web/package.json`, `src/web/eslint.config.*` if present | `cd src/web && npm run lint` | Static linting gate; the script exists even though the current CI workflow does not run it yet. |
| Vue Web | Unit / component / store / API tests | Vitest + Vue Test Utils | `src/web/src/**/__tests__/*` | `cd src/web && npm run test` | 8 files / 61 tests: API client 24, auth store 17, components 10, pages 3, design-token enforcement 7. Mocks browser APIs and the API client at the boundary. |
| Vue Web | Build parity | `vite` + `vue-tsc --build` | `src/web/package.json` | `cd src/web && npm run build` | Production build gate. This is stricter than `vue-tsc --noEmit`; nullable props and indexed access that pass locally can still fail here. |
| Vue Web | Browser workflows | Playwright | `src/web/e2e/` | `task test-critical-workflows` | Deterministic F013 smoke coverage for login/session setup, manual add coin, edit-one-field, storage-location set/clear, tags/sets, upload/delete image, collection search/filter, and mobile viewport edit workflows with mocked API routes and golden fixtures. |
| Python Agent | Lint | Ruff | `src/agent/app/`, `src/agent/tests/`, `src/agent/pyproject.toml` | `cd src/agent && ruff check app/ tests/` | Import order, correctness, and style rules for the deterministic Python surface. |
| Python Agent | Unit / contract tests | pytest | `src/agent/tests/test_*.py` | `cd src/agent && pytest tests/ -v` | 6 files / 35 tests covering FastAPI request validation, Pydantic models, retry logic, streaming helpers, availability parsing, and supervisor location context. |
| Python Agent | Static type checking | None configured today | `src/agent/pyproject.toml` | N/A | No `mypy` or `pyright` config is present; schema validation and pytest carry the current contract burden. |

Notes:
- `grep -rln "httptest\|testserver\|integration" src/api --include="*_test.go"` finds handler and middleware HTTP tests, but no dedicated integration package.
- Browser workflow tests live in `src/web/e2e/` and run with Playwright.
- `pytest tests/ --collect-only -q` currently collects 35 agent tests across 6 files.

## 3. What we DON'T test

- We do **not** test SQLite, GORM, Gin, Vitest, FastAPI, or LangGraph themselves; we test our usage of them.
- We do **not** hit third-party services in automated tests. Anthropic, Ollama, SearXNG, NumisBids, and Pushover should be mocked or stubbed at the service boundary.
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
- Always finish with `npm run test` and `npm run build`; the build catches stricter TypeScript failures than lighter local checks.

### Python Agent

- Put tests in `src/agent/tests/` as `test_*.py`.
- Use pytest fixtures to remove nondeterminism or waiting; `src/agent/tests/test_retry.py` patches retry delays to zero.
- Use FastAPI `TestClient` for route-level contracts, as in `src/agent/tests/test_api.py`.
- Stub LLM/model calls with `AsyncMock` or direct helper invocation rather than making live provider calls.
- Favor testing deterministic helpers directly—`parse_verdicts` in `src/agent/tests/test_availability.py` and `_build_coin_show_location_context` in `src/agent/tests/test_supervisor_coin_shows_location.py` are good examples.
- Assert structured outputs and schema defaults, not prose quality.
- Keep provider/network behavior at the boundary; real Anthropic/Ollama/SearXNG traffic does not belong in CI.

## 6. Running tests locally vs. CI

- Start from the task runner when possible: `task test` (Go), `task build` (Go + web build), `task test-agent`, and `task lint-agent` from [`../Taskfile.yml`](../Taskfile.yml).
- Run critical browser workflows from the repository root with `task test-critical-workflows`. The target delegates to the existing web package script, `npm run test:browser`.
- Frontend tests and lint currently live in npm scripts documented in [`../src/web/README.md`](../src/web/README.md): `npm run test`, `npm run lint`, `npm run build`, `npm run type-check`.
- The current Quality Gate workflow lives at [`../.github/workflows/ci.yml`](../.github/workflows/ci.yml). Today it runs Go build/vet/architecture tests, Vue `vue-tsc --noEmit`, and Python Ruff + pytest.
- Local expectations are intentionally stricter than the current CI file in a few places: the Constitution still expects full `go test ./...` and `npm run build` before you call work done.
- Pre-commit hooks run only a subset locally; they are a convenience layer, not the definition of done. The full gate remains Constitution §17 plus the CI workflow.

## 7. Coverage philosophy

We do **not** gate PRs on a repo-wide coverage percentage. Coverage is a signal for blind spots, not a target to game.

Constitution §21.6 is the operative rule: **"every new service method has ≥ 1 unit test."** That pushes effort toward critical paths, invariants, and regressions instead of padding helpers with low-value tests. Use `go test ./... -cover` as a diagnostic when helpful, but do not treat a single percentage as proof of safety.

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

## 9. Cross-references

- Constitution: [`../.specify/memory/constitution.md`](../.specify/memory/constitution.md) (especially Principle X, §17, and §21)
- System architecture: [`ARCHITECTURE.md`](ARCHITECTURE.md)
- Go API README: [`../src/api/README.md`](../src/api/README.md)
- Vue frontend README: [`../src/web/README.md`](../src/web/README.md)
- Python agent README: [`../src/agent/README.md`](../src/agent/README.md)
- Quality Gate workflow: [`../.github/workflows/ci.yml`](../.github/workflows/ci.yml)
- Task runner: [`../Taskfile.yml`](../Taskfile.yml)

TODOs:
- Python static type checking is still absent. If we promote it to backlog, file it as `specs/_backlog/F012-agent-static-type-checking.md`.
