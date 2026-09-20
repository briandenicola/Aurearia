---
description: "Implementation tasks for issue #721 read-only Coin Copilot harness"
---

# Tasks: Coin Copilot Read-Only Harness

**Input**: `/specs/359-coin-copilot-harness/`  
**Prerequisites**: `spec.md`, `plan.md`, `research.md`, `data-model.md`,
`contracts/coin-copilot.openapi.yaml`, `contracts/agent-internal-contract.md`,
`contracts/sse-events.md`, ADR 0016

**Tests**: Required. Write the named tests before implementation and prove they
fail for the intended reason.

**Budget policy**: Dollar-cost enforcement is deferred until a trustworthy,
current provider/model pricing source exists. Reliable provider-reported
input/output token usage remains observable. Iteration, tool-call, wall-clock
(120-second default, 150-second maximum), sequential-concurrency, and payload
limits remain enforced; the timeout maximum plus the 30-second credential
buffer fits the absolute 180-second execution-token TTL.

## Phase 1: Contract and settings foundation

- [x] T001 [P] Create shared valid/invalid Coin Copilot JSON contract fixtures in `src/agent/tests/fixtures/coin_copilot/`
- [x] T002 [P] Add Go fixture loader and failing contract tests in `src/api/services/coin_copilot_contract_test.go`
- [x] T003 [P] Add Python strict-schema failing tests in `src/agent/tests/test_coin_copilot_contract.py`
- [x] T004 Add Coin Copilot setting constants/defaults and validated `CoinCopilotSettings` snapshot in `src/api/services/settings_service.go`; omit dollar-cost settings until a trustworthy pricing source exists
- [x] T005 Add settings validation/default-off tests in `src/api/services/settings_service_test.go`
- [x] T006 Add admin setting keys to the existing settings allowlist/read-write flow in `src/api/handlers/admin.go`

**Checkpoint**: Shared contracts and safe configuration are fixed before runtime work.

## Phase 2: Durable Go state

- [x] T007 [P] Create `CoinCopilotThread`, `CoinCopilotRun`, `CoinCopilotCheckpoint`, `CoinCopilotEvent`, and `CoinCopilotResumeRequest` models in `src/api/models/coin_copilot.go`
- [x] T008 Register additive models and idempotent unique indexes in `src/api/database/database.go`
- [x] T009 Add migration tests for all new tables/indexes in `src/api/database/migration_test.go`
- [x] T010 [P] Create owner-scoped repository tests for thread/run reads and foreign-id non-disclosure in `src/api/repository/coin_copilot_repository_test.go`
- [x] T011 [P] Add failing transactional event-sequence/checkpoint tests in `src/api/repository/coin_copilot_repository_test.go`
- [x] T012 [P] Add failing start/resume idempotency repository tests in `src/api/repository/coin_copilot_repository_test.go`
- [x] T013 Implement all DB access and transaction helpers in `src/api/repository/coin_copilot_repository.go`
- [x] T014 Add the repository to `appDeps` and construct it in `src/api/deps.go`

## Phase 3: Run lifecycle and worker

- [x] T015 [P] Add transition-table, owner-scope, start-idempotency, resume-version, cancellation-race, and exactly-one-terminal-event tests in `src/api/services/coin_copilot_service_test.go`
- [x] T016 [P] Add stale recovery, pause expiry, event/checkpoint retention, and thread cascade-deletion tests in `src/api/services/coin_copilot_service_test.go`
- [x] T017 Implement start/read/resume/cancel/state-transition orchestration in `src/api/services/coin_copilot_service.go`
- [x] T018 Implement replay broker with bounded subscribers and no persistence authority in `src/api/services/coin_copilot_broker.go`
- [x] T019 [P] Add worker queue/backpressure, heartbeat, shutdown, and one-active-run tests in `src/api/services/coin_copilot_worker_test.go`
- [x] T020 Implement bounded worker claim/execution/stale-recovery/janitor loops in `src/api/services/coin_copilot_worker.go`
- [x] T021 Wire worker startup and graceful shutdown through `src/api/deps.go` and existing server runtime shutdown paths

**Checkpoint**: Go can durably create, pause, resume, cancel, recover, replay, and retain runs without Python.

## Phase 4: Execution credentials and read-only tools

- [x] T022 [P] Add execution-token signature, expiry, canonical encoding, revocation, wrong-owner/run/execution, wrong-tool, and replay tests in `src/api/services/internal_token_service_test.go`
- [x] T023 Implement the HKDF-separated Coin Copilot execution token family in `src/api/services/internal_token_service.go`
- [x] T024 [P] Add internal callback handler tests proving owner scope, current-execution binding, tool-call idempotency, budget checks, and no write endpoints in `src/api/handlers/internal_tools_test.go`
- [x] T025 Add thin read-only Copilot callback handlers that reuse `CollectionToolsService` in `src/api/handlers/internal_tools.go`
- [x] T026 Register only the four read callback routes under `/api/internal/copilot/tools` in `src/api/routes_internal.go`
- [x] T027 Add an architecture/route guard test proving no Copilot write, arbitrary HTTP, shell, filesystem, or database tool is registered

## Phase 5: Stateless Python harness

- [x] T028 [P] Add strict execution/checkpoint/frame Pydantic models in `src/agent/app/models/requests.py` and `src/agent/app/models/responses.py`
- [x] T029 [P] Add Anthropic binding and Ollama `/api/show` capability tests in `src/agent/tests/test_coin_copilot_capabilities.py`
- [x] T030 Implement fail-closed provider/model capability checks in `src/agent/app/llm/capabilities.py`
- [x] T031 [P] Add read-tool schema, callback binding, duplicate-call, tamper, timeout, and oversized-result tests in `src/agent/tests/test_coin_copilot_security.py`
- [x] T032 Implement execution-token-bound read tools in `src/agent/app/tools/copilot_collection_tools.py`
- [x] T033 [P] Add bounded sequential planning, multi-tool completion, clarification, no-results, malformed call, injection, iteration/tool/time exhaustion, token-usage, and payload-bound tests in `src/agent/tests/test_coin_copilot_harness.py`
- [x] T034 Implement the stateless bounded harness in `src/agent/app/teams/coin_copilot.py`
- [x] T035 Ensure portfolio review consumes only validated collection data and no live-market capability in `src/agent/app/teams/portfolio_review.py`
- [x] T036 Add a read-only gap-analysis mode with no acquisition price/market search output in `src/agent/app/teams/gap_analysis.py`
- [x] T037 Register `POST /api/copilot/execute` with strict internal authentication and SSE framing in `src/agent/app/routes.py`
- [x] T038 Add a Python architecture test proving Coin Copilot imports no DB client and exposes no deferred capability

## Phase 6: Go/Python bridge and persisted event projection

- [x] T039 [P] Add fake-agent tests for every internal frame, duplicate frame, forbidden reasoning field, invalid sequence, and mismatched execution in `src/api/services/coin_copilot_contract_test.go`
- [x] T040 [P] Add proxy timeout, disconnect, cancellation, and secret-redaction tests in `src/api/services/agent_proxy_test.go`
- [x] T041 Add typed Coin Copilot proxy request/frame DTOs and execution streaming method in `src/api/services/agent_proxy.go`
- [x] T042 Implement frame validation, bounded tool-result persistence, digest/truncation, reliable token-usage accounting without estimated-cost fields, and public-event translation in `src/api/services/coin_copilot_contract.go`
- [x] T043 Integrate the proxy execution stream with `CoinCopilotWorker`, including fresh token mint/revoke and post-await budget/cancel checks
- [x] T044 Add an integration test using real SQLite plus a fake Python stream to prove checkpoint/event atomicity and restart resume in `src/api/integration/coin_copilot_seam_test.go`

## Phase 7: Public REST and replayable SSE

- [x] T045 [P] Add handler tests for capability, start, thread read/delete, run read, cancel, resume, validation, idempotency, active-run deletion conflict, cascade deletion, and foreign-id `404` in `src/api/handlers/coin_copilot_test.go`
- [x] T046 [P] Add SSE replay, `since` precedence, truncation, keepalive, terminal close, and connection-limit tests in `src/api/handlers/coin_copilot_sse_test.go`
- [x] T047 Implement thin typed public handlers with Swagger annotations in `src/api/handlers/coin_copilot.go`
- [x] T048 Implement persisted event streaming in `src/api/handlers/coin_copilot_sse.go`
- [x] T049 Register protected read/write-rate-limited routes in `src/api/routes_protected.go`
- [x] T050 Add exact route/OpenAPI drift guards in `src/api/route_openapi_drift_test.go`

**Checkpoint**: The complete public start/read/delete/stream/cancel/resume API works independently of Vue.

## Phase 8: Existing chat drawer UX

- [x] T051 [P] Add TypeScript discriminated unions for capability, run, checkpoint summary, and all ten event payloads in `src/web/src/types/agent.ts`
- [x] T052 [P] Add API methods and replayable fetch-SSE parser in `src/web/src/api/endpoints/agent.ts`
- [x] T053 [P] Add failing composable tests for capability fallback, idempotent start/resume, reconnect/de-duplication, cancel, and terminal close in `src/web/src/composables/__tests__/useCoinCopilot.test.ts`
- [x] T054 Implement durable run state in `src/web/src/composables/useCoinCopilot.ts`
- [x] T055 [P] Create plan/progress renderer using existing design tokens/classes in `src/web/src/components/chat/CopilotRunProgress.vue`
- [x] T056 [P] Create accessible clarification/resume card in `src/web/src/components/chat/CopilotClarificationCard.vue`
- [x] T057 Integrate capability-selected Copilot mode into `src/web/src/components/CoinSearchChat.vue` without changing legacy message/proposal behavior
- [x] T058 Extend `src/web/src/components/__tests__/CoinSearchChat.test.ts` for desktop/PWA progress, clarification, cancel, resume, reconnect, and legacy fallback
- [x] T059 Add an app-navigation regression test proving the same drawer/FAB/sidebar entry points remain in `src/web/src/__tests__/AppNavigation.test.ts`

## Phase 9: Admin rollout and documentation

- [x] T060 [P] Add Coin Copilot toggle and enforceable bounded limit inputs to `src/web/src/components/admin/AdminSystemSection.vue`; do not expose a dollar-cost control
- [x] T061 Wire settings props/save payload in `src/web/src/pages/AdminPage.vue`
- [x] T062 Add admin component tests for default-off and range validation in `src/web/src/components/admin/__tests__/AdminSystemSection.coin-copilot.test.ts`
- [x] T063 [P] Regenerate Swagger and synchronize `docs/openapi.json`, `docs/api-reference.md`, and generated `src/api/docs/`
- [x] T064 [P] Document feature scope/fallback/privacy in `docs/features.md`, `docs/ARCHITECTURE.md`, `docs/testing.md`, and `docs/threat-model.md`
- [x] T065 Execute every scenario in `specs/359-coin-copilot-harness/quickstart.md` and record results in the PR description

## Phase 10: Quality Gate

- [x] T066 Run `go test ./...`, `go vet ./...`, and `go build ./...` from `src/api`
- [x] T067 Run `ruff check app tests` and `pytest tests -v` from `src/agent`
- [x] T068 Run `npm run test` and `npm run build` from `src/web`
- [x] T069 Run OpenAPI generation, contract fixture checks, secret scan, and `git diff --check`
- [x] T070 Complete Constitution §17 and §21 self-check, citing Principles I–IX, ADR 0016, and exact workflow/tamper coverage

Validation completed 2026-09-17:

- Go: full tests, vet, build, architecture, OpenAPI drift, shared contract
  fixtures, and the real SQLite/fake-Python seam passed.
- Python: Ruff passed; full pytest passed with 413 tests.
- Web: ESLint passed; 175 Vitest files with 1,427 tests passed; the strict
  production build passed.
- Quickstart scenarios 1-8 were exercised by the cross-service seam,
  lifecycle, owner-isolation, credential-tamper, capability-fallback,
  replay/resume/cancellation, limits/privacy/retention, and responsive drawer
  regression suites.
- Gitleaks and `git diff --check` passed. Trivy found no high/critical
  vulnerabilities in the Go, Python, or npm dependency locks and no findings
  in either production Dockerfile. Its repository-wide config scan continues
  to report the pre-existing `.devcontainer/Dockerfile` root/sudo findings,
  which are outside Feature 359 and the production images.
- `task openapi` remains affected by the documented Windows PowerShell
  templating defect; the equivalent direct `swag init` generation and route
  drift test passed with synchronized artifacts.

## Dependencies and ownership

```text
Contract/settings foundation
  -> Durable Go state
  -> Run lifecycle/worker
  -> Execution credentials/read tools
  -> Python harness
  -> Go/Python bridge
  -> Public API/SSE
  -> Vue drawer
  -> Admin/docs
  -> Full quality gate
```

| Owner | Task ranges | Boundary |
|---|---|---|
| Go/backend | T002, T004–T027, T039–T050 | Models → Repository → Service → Handler; wiring only in composition root |
| Python/agent | T001, T003, T028–T038 | Stateless inference and typed internal frames; no DB |
| Vue/frontend | T051–T062 | Existing drawer, typed client, responsive controls |
| Tests/docs | T063–T070 plus paired tests in every phase | Contract drift, tamper, workflow regression, docs/gates |

Do not begin Vue integration before the public API contract is executable. Do
not expose Python tools before the execution-token and Go owner-scope tests are
green. Do not enable the feature by default.
