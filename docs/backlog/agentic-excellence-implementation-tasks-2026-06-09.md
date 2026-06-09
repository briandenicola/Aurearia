# Agentic Excellence Implementation Tasks (2026-06-09)

## Scope

This task plan breaks the Agentic Excellence Roadmap into implementation-ready
slices. It is roadmap-level, not a generated `specs/NNN-*/tasks.md`; promote
each backlog card through SpecKit before implementation.

## Task format

- `[P]` means the task is parallelizable after its phase prerequisites are done.
- `[F0NN]` maps the task to a backlog card.
- Paths are the expected primary touch points; exact files may narrow during
  the promoted spec/plan phase.

## Phase 0 — Promote and baseline the roadmap

- [ ] AE001 [F013] Promote `specs/_backlog/F013-critical-workflow-hardening.md` into an active SpecKit feature under `specs/NNN-critical-workflow-hardening/spec.md`.
- [ ] AE002 [F013] Capture current critical workflow behavior and known regressions in `specs/NNN-critical-workflow-hardening/spec.md`.
- [ ] AE003 [F013] Define the golden collection fixture shape in `specs/NNN-critical-workflow-hardening/spec.md`.
- [ ] AE004 [F013] Decide deterministic browser test tool and run cadence in `specs/NNN-critical-workflow-hardening/plan.md`.
- [ ] AE005 [F013] Add initial task breakdown in `specs/NNN-critical-workflow-hardening/tasks.md`.

## Phase 1 — F013 critical workflow hardening

### Backend typed mutation path

- [ ] AE006 [F013] Inventory all coin create/update entry points in `src/api/handlers/coins.go`, `src/api/services/coin_service.go`, and `src/api/repository/coin_repository.go`.
- [ ] AE007 [F013] Define typed coin create/update request DTOs in `src/api/handlers/coins.go` or a dedicated `src/api/handlers/coin_requests.go`.
- [ ] AE008 [F013] Map typed request DTOs to service inputs without binding broad `models.Coin` mutation payloads in `src/api/handlers/coins.go`.
- [ ] AE009 [F013] Preserve service-layer business rules for storage locations, eras, references, value history, tags, and sets in `src/api/services/coin_service.go`.
- [ ] AE010 [F013] Reduce repository update helpers to one obvious mutation path or document why each remaining helper is distinct in `src/api/repository/coin_repository.go`.
- [ ] AE011 [F013] Add backend regression tests for one-field edits, storage-location edits, tags, sets, references, legacy/custom era, and value snapshots in `src/api/handlers/coin_handler_test.go`.
- [ ] AE012 [F013] Add repository-level tests for association-safe updates in `src/api/repository/coin_repository_test.go`.
- [ ] AE013 [F013] Regenerate API docs with `task openapi` if public request/response contracts change.

### Golden fixtures

- [ ] AE014 [P] [F013] Create Go test fixture builders for representative coins in `src/api/testutil/` or existing API test helpers.
- [ ] AE015 [P] [F013] Create frontend fixture data for representative coins in `src/web/src/test/` or existing test helper folders.
- [ ] AE016 [P] [F013] Document fixture coverage in `docs/testing.md`.
- [ ] AE017 [F013] Ensure fixture data covers Roman, Greek, Byzantine, wishlist, sold, private, tagged, set-member, storage-location, image-heavy, and legacy/custom-era coins.

### Browser workflow coverage

- [ ] AE018 [F013] Add deterministic browser smoke test infrastructure under `src/web/` using the selected tool.
- [ ] AE019 [F013] Add login and authenticated session setup workflow tests under `src/web/`.
- [ ] AE020 [F013] Add add-coin workflow test covering manual entry and save under `src/web/`.
- [ ] AE021 [F013] Add edit-one-field workflow test under `src/web/`.
- [ ] AE022 [F013] Add edit-storage-location workflow test under `src/web/`.
- [ ] AE023 [F013] Add edit-tags-and-sets workflow test under `src/web/`.
- [ ] AE024 [F013] Add upload/delete image workflow test under `src/web/`.
- [ ] AE025 [F013] Add collection search/filter workflow test under `src/web/`.
- [ ] AE026 [F013] Add mobile viewport edit workflow test under `src/web/`.
- [ ] AE027 [F013] Add local task command for critical workflow tests in `Taskfile.yml`.
- [ ] AE028 [F013] Document critical workflow test command and expected fixture setup in `docs/testing.md`.

## Phase 2 — F011 AI-driven browser testing

- [ ] AE029 [F011] Promote `specs/_backlog/F011-ai-driven-browser-testing.md` into an active SpecKit feature after F013 defines fixtures/workflows.
- [ ] AE030 [F011] Choose Playwright MCP, deterministic Playwright, or hybrid execution in `specs/NNN-ai-driven-browser-testing/plan.md`.
- [ ] AE031 [F011] Add `task test-ui-explore` in `Taskfile.yml` with hard step/time/token budgets.
- [ ] AE032 [F011] Implement exploratory browser runner under `src/web/` or `tools/agentic-testing/`.
- [ ] AE033 [F011] Emit structured reports with route coverage, screenshots, console errors, network failures, accessibility findings, and severity.
- [ ] AE034 [F011] Add seeded-finding or fixture-based validation so the runner proves it can detect a known issue.
- [ ] AE035 [F011] Document advisory versus blocking behavior in `docs/testing.md`.
- [ ] AE036 [F011] Add optional GitHub Actions integration for manual or nightly exploratory runs in `.github/workflows/`.

## Phase 3 — F012 existing collection intelligence spine

- [ ] AE037 [F012] Inventory current collection chat behavior in `src/agent/app/teams/collection_chat.py`, `src/agent/app/tools/collection_tools.py`, `src/api/services/collection_tools_service.go`, and `src/web/src/composables/useCoinSearchChat.ts`.
- [ ] AE038 [F012] Add behavior tests for collection read tools: search, get coin, summary, and top value in `src/api/services/`.
- [ ] AE039 [F012] Add behavior tests for confirm-gated update proposals and commits in `src/api/services/`.
- [ ] AE040 [F012] Replace untyped proposal change maps with typed allowlisted change structures where practical in `src/api/services/collection_tools_service.go`.
- [ ] AE041 [F012] Improve collection chat routing examples and tests in `src/agent/app/supervisor.py` and `src/agent/tests/`.
- [ ] AE042 [F012] Improve frontend proposal rendering, confirm, and cancel UX in `src/web/src/composables/useCoinSearchChat.ts` and related chat components.
- [ ] AE043 [F012] Ensure external tool server parity reuses the same service logic in `src/api/handlers/external_tools.go`.
- [ ] AE044 [F012] Update external tool docs in `docs/external-tool-server.md`.

## Phase 4 — F014 attribution and reference assistant

- [ ] AE045 [F014] Promote `specs/_backlog/F014-attribution-reference-assistant.md` into an active SpecKit feature.
- [ ] AE046 [F014] Inventory current intake/analysis/reference behavior in `src/agent/app/teams/coin_intake.py`, `src/agent/app/teams/coin_analysis.py`, and `src/api/models/coin_reference.go`.
- [ ] AE047 [F014] Define structured attribution draft schemas with field-level confidence and evidence in `src/agent/app/models/responses.py`.
- [ ] AE048 [F014] Add source-backed catalog reference candidates to the intake/attribution pipeline in `src/agent/app/teams/coin_intake.py` or a new focused team.
- [ ] AE049 [F014] Add trusted authority/source helpers for v1 source set in `src/agent/app/tools/`.
- [ ] AE050 [F014] Add Go API review/apply endpoints or reuse existing intake draft commit flow in `src/api/handlers/`.
- [ ] AE051 [F014] Add review-and-apply UI for individual fields in `src/web/src/pages/AddCoinPage.vue` or a dedicated attribution component.
- [ ] AE052 [F014] Add tests for low confidence, conflicting sources, no-match results, and preserving user-entered data.
- [ ] AE053 [F014] Document attribution confidence, evidence, and source limitations in `docs/features/`.

## Phase 5 — F015 curator, watchlist, and provenance agents

- [ ] AE054 [F015] Promote `specs/_backlog/F015-curator-watchlist-provenance-agents.md` into an active SpecKit feature.
- [ ] AE055 [F015] Inventory existing portfolio, gap, availability, price trends, and similar-lot teams under `src/agent/app/teams/`.
- [ ] AE056 [F015] Define collector preference model or settings keys for budget, favorite periods, disliked categories, preferred dealers, and collecting goals in `src/api/models/` or `src/api/services/settings_service.go`.
- [ ] AE057 [F015] Add settings UI for collector preferences in `src/web/src/components/settings/`.
- [ ] AE058 [F015] Add curator recommendation schema and team behavior in `src/agent/app/teams/`.
- [ ] AE059 [F015] Add watchlist evaluation schema and workflow using existing wishlist/availability data in `src/agent/app/teams/`.
- [ ] AE060 [F015] Add provenance/risk finding schema and read-only analysis workflow in `src/agent/app/teams/`.
- [ ] AE061 [F015] Add frontend surfaces for recommendations, watchlist verdicts, and provenance/risk warnings in `src/web/src/components/`.
- [ ] AE062 [F015] Add tests for user scoping, private coins, confidence thresholds, and no-write-without-confirmation.
- [ ] AE063 [F015] Update feature documentation in `docs/features/`.

## Phase 6 — F016 agentic engineering quality cockpit

- [ ] AE064 [F016] Promote `specs/_backlog/F016-agentic-engineering-quality-cockpit.md` into an active SpecKit feature.
- [ ] AE065 [F016] Define initial quality metrics: critical workflow coverage, flaky tests, large files, complexity hotspots, untyped maps/casts, stale specs/ADRs, and Principle IV self-check status.
- [ ] AE066 [F016] Implement a deterministic local report command in `Taskfile.yml` using existing tools first.
- [ ] AE067 [F016] Generate a Markdown or GitHub Actions summary report under `docs/audits/` or CI artifacts.
- [ ] AE068 [F016] Add stale governance/spec checks for `specs/_backlog/`, `docs/adr/`, and `.specify/memory/constitution.md`.
- [ ] AE069 [F016] Add complexity hotspot detection for large files and repeated broad maps/casts.
- [ ] AE070 [F016] Map agent review roles to report sections in `.squad/agents/*/charter.md` or `.squad/decisions/inbox/`.
- [ ] AE071 [F016] Document advisory versus blocking thresholds in `docs/testing.md` or `docs/ARCHITECTURE.md`.

## Cross-cutting validation tasks

- [ ] AE072 [P] Run Go validation from `src/api`: `go build ./...`, `go vet ./...`, `go test -v ./...`.
- [ ] AE073 [P] Run frontend validation from `src/web`: `npm run build`, `npm test`, and lint if configured for the touched area.
- [ ] AE074 [P] Run agent validation from `src/agent`: `ruff check app/ tests/` and `python -m pytest tests/ -v`.
- [ ] AE075 [P] Update OpenAPI artifacts with `task openapi` whenever public API contracts change.
- [ ] AE076 [P] Update docs for every shipped user-visible agentic workflow in `docs/features/`, `docs/testing.md`, or `docs/external-tool-server.md`.
- [ ] AE077 [P] Add or update ADRs for material architecture/security/tooling decisions under `docs/adr/`.

## Suggested first implementation slice

Start with **F013 / AE001-AE013**. That gives the team a stable typed update
contract and regression suite before expanding browser automation or agentic
writes.
