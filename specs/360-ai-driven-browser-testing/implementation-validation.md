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
