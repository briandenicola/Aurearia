# Implementation Plan: Critical Workflow Hardening

**Branch**: `220-critical-workflow-hardening` | **Date**: 2026-06-09 | **Spec**: `/specs/220-critical-workflow-hardening/spec.md`  
**Input**: Feature specification from `/specs/220-critical-workflow-hardening/spec.md`

## Summary

Promote F013 by hardening coin create/update before adding broader agentic workflows: replace broad mutation binding with typed handler request contracts, preserve service-layer association/value business rules, add backend and repository regression coverage, define reusable golden collection fixtures, and add deterministic browser workflow tests for the core collection flows that F011 can later use as its exploration baseline.

## Technical Context

**Language/Version**: Go 1.26.x, TypeScript (Vue 3), Python 3.12  
**Primary Dependencies**: Gin, GORM, SQLite, Vue 3 + Pinia, Vite, Taskfile  
**Storage**: SQLite existing `coins` and related association tables; no new production table planned  
**Testing**: `go build ./...`, `go vet ./...`, `go test -v ./...`, `npm run build`, deterministic browser workflow command added by this feature  
**Target Platform**: Linux-hosted web app + PWA/mobile browser client  
**Project Type**: Web application (Go API + Vue frontend; Python agent unaffected except future dependency)  
**Performance Goals**: Coin create/update latency remains equivalent to current service paths; browser workflow suite stays small enough for local pre-PR use  
**Constraints**: Keep handler → service → repository boundaries; no broad rewrite; no new agentic write surface; no production fixture data; explicit contracts and repeatable tests over manual memory  
**Scale/Scope**: Existing coin mutation path, fixture helpers, docs, Taskfile, and critical web workflow tests

## Constitution Check

*GATE: Must pass before implementation. Re-check after design and before PR.*

| Gate | Status | Notes |
|------|--------|-------|
| Principle I (Clear Layered Architecture) | PASS | Handlers bind typed DTOs, services keep business rules, repositories keep database access. |
| Principle III (Strict Types and Explicit Contracts) | PASS | Create/update contracts become explicit request DTOs and documented Swagger contracts if public schemas change. |
| Principle IV (Simple Complete Changes) | PASS | Scope is limited to critical sibling workflows around coin create/update and their tests. |
| Principle VIII (Documented Decisions) | PASS | Browser-test scope decision is recorded in `.squad/decisions/inbox/`. |
| Principle IX (Automated Enforcement Over Manual Memory) | PASS | Regression coverage and browser workflows become repeatable checks. |
| §17 Quality Gate | PASS | Plan keeps existing Go/Web validations and adds the critical workflow command when available. |

## Project Structure

### Documentation (this feature)

```text
specs/220-critical-workflow-hardening/
├── plan.md
├── spec.md
└── tasks.md
```

### Source Code (repository root)

```text
src/api/
├── handlers/
│   ├── coins.go
│   ├── coin_requests.go                 # possible new DTO home
│   └── coin_handler_test.go
├── services/
│   └── coin_service.go
├── repository/
│   ├── coin_repository.go
│   └── coin_repository_test.go
└── testutil/                             # possible fixture builder home

src/web/
├── src/test/                             # possible frontend fixture data home
└── <selected browser test location>       # chosen during implementation

Taskfile.yml
docs/testing.md
```

**Structure Decision**: Keep this feature inside existing API and web test surfaces. Add DTOs near handlers, keep mutation orchestration in `CoinService`, keep persistence in `CoinRepository`, and keep fixtures in test-only helper folders.

## Architecture / Implementation Approach

### Typed coin mutation contracts

1. Inventory all create/update entry points before changing behavior.
2. Introduce `CoinCreateRequest` and `CoinUpdateRequest` in `handlers/coins.go` or `handlers/coin_requests.go`.
3. Bind only explicit fields in handlers and map to service inputs.
4. Preserve existing service rules for storage locations, eras, references, value snapshots, tags, and sets. Explicit `null` on nullable scalar update fields (`purchasePrice`, `currentValue`, dates, sale price, weight, diameter) clears the field; omission preserves it.
5. Simplify or document repository update helpers so there is one obvious association-safe path.
6. Regenerate OpenAPI artifacts if public request/response contracts change.

### Golden fixtures

1. Create Go test fixture builders for backend service/repository/handler tests.
2. Create frontend fixture data or seed helpers for deterministic browser workflows.
3. Cover Roman, Greek, Byzantine, wishlist, sold, private, tagged, set-member, storage-location, image-heavy, and legacy/custom-era examples.
4. Document fixture coverage and setup in `docs/testing.md`.

### Deterministic browser workflow tests

1. Use conventional deterministic browser automation for F013; defer LLM-driven exploration to F011.
2. Select the smallest tool/cadence that can run login, add, edit, image, search/filter, and mobile viewport workflows repeatably.
3. Add a Taskfile command for local execution.
4. Keep the initial suite critical and proportional; do not broaden into speculative UI exploration.

## Brutus QA Strategy for AE011-AE028

### Existing coverage inventory

- **Backend handler tests** (`src/api/handlers/coin_handler_test.go`): authenticated list/get/create/update/delete, owner isolation, invalid JSON, set-membership preservation on update, storage-location update with set preservation, custom registry era, unchanged legacy era, and new focused coverage for explicit `storageLocationId: null` clearing plus structured reference replacement.
- **Backend service tests** (`src/api/services/coin_service_test.go`): create records value snapshots, manual current-value updates record value history/journal, estimate updates skip history, delete/purchase/sell paths, custom registry era acceptance/rejection, and unchanged legacy era preservation.
- **Backend repository tests** (`src/api/repository/coin_repository_test.go`): create/get, wrong-user access, delete cascade, ownership/public/active scopes, value snapshot totals, deterministic random sort, and association-safe updates that preserve set memberships across `Update`, `UpdateField`, `UpdateFields`, and `UpdateStorageLocationID`.
- **Frontend tests**: `src/web/src/api/__tests__/client.test.ts` covers create/update sanitization; `src/web/src/components/__tests__/CoinForm.test.ts` and `src/web/src/pages/__tests__/EditCoinPage.test.ts` are source-scanning regressions for custom/legacy era preservation; collection/wishlist page tests cover rendering/list fetch behavior but not add/edit workflows.
- **Known gap**: no deterministic browser suite exists yet, and current frontend tests do not prove full form submit payloads for tags, sets, references, images, or storage-location edits.

### Golden collection fixture matrix

| Fixture | Purpose | Required traits covered |
|---|---|---|
| `roman-denarius-core` | Default owned active coin for scalar edit/search tests | Roman, active, public, reference-ready |
| `greek-tetradrachm-valued` | Value-history and purchase/current-value side effects | Greek, silver, purchase price/date, current value |
| `byzantine-solidus-set-member` | Set membership preservation and set filters | Byzantine, set-member, gold |
| `wishlist-aureus-target` | Wishlist add/purchase/search workflows | wishlist, high value, no purchase date |
| `sold-sestertius-archive` | Sold filtering and non-active collection behavior | sold, sale price/date/to |
| `private-provincial-bronze` | Privacy and custom/legacy era preservation | private, legacy/custom era |
| `tagged-follis-storage` | Tag and storage-location mutation workflows | tagged, storage-location |
| `image-heavy-drachm` | Browser upload/delete and gallery stress path | image-heavy, multiple image types |
| `reference-rich-denarius` | Structured reference replacement/dedup validation | references, catalog registry dependency |

### Test placement

- **Backend handler/service/repository**: AE011-AE013. Use real in-memory SQLite and Gin `httptest` for explicit request contracts, owner scoping, storage clear/set, references replace, value-history side effects, and association-safe updates.
- **Frontend component/source tests**: AE014-AE017 early fixture data checks, API payload sanitization, `CoinForm`/`EditCoinPage` prop and submit payload safeguards. Keep these small until browser tooling is selected.
- **Deterministic browser tests**: AE018-AE028. Script login, add coin, edit one field, edit storage location, edit tags/sets, upload/delete image, search/filter, and mobile edit against seeded golden fixtures. These should be conventional repeatable tests and become the critical workflow command.
- **F011 AI-driven exploratory tests later**: use the F013 fixtures/workflows as prompts and guardrails for advisory exploration: console/network errors, visual/mobile oddities, accessibility/focus issues, and unexpected runtime paths. Do not let F011 perform writes outside throwaway seeded data.

### Immediate QA tasks

1. Finish T011 by adding missing backend tests for tags, images, full create payloads, and typed DTO unknown-field rejection once DTOs land.
2. Finish T012 with repository tests for storage-location clearing and reference preloading/replacement if repository helper behavior changes further.
3. Build T014/T015 fixture helpers from the matrix above before browser tests are authored.
4. Select browser tooling before T018; no new frontend E2E dependency should be introduced in this first pass without the plan being updated.

## Phase 0 Research (Completed for Promotion)

Promotion resolves these roadmap-level decisions:

1. F013 is active spec `220-critical-workflow-hardening` because 220 is the next numbered spec directory.
2. F013 owns deterministic critical workflow tests and golden fixtures.
3. F011 remains responsible for AI-driven exploratory browser testing after F013 defines stable fixtures/workflows.

## Phase 1 Design Outputs (Completed for Promotion)

1. `spec.md` defines user stories, functional requirements, edge cases, and success criteria.
2. `tasks.md` maps AE001-AE028 into dependency-ordered implementation tasks.
3. Backlog card F013 is marked promoted with a link to this active spec.
4. Decision inbox records deterministic browser testing as F013's baseline and AI exploration as F011's follow-up.

## Post-Design Constitution Check

| Gate | Status | Notes |
|------|--------|-------|
| Principle I (Clear Layered Architecture) | PASS | Planned changes preserve handler/service/repository responsibility split. |
| Principle III (Strict Types and Explicit Contracts) | PASS | Typed mutation DTOs are the central deliverable. |
| Principle IV (Simple Complete Changes) | PASS | Scope excludes new agentic write behavior and broad UI exploration. |
| Principle IX (Automated Enforcement Over Manual Memory) | PASS | Backend regressions and browser workflows are explicit tasks. |

## Complexity Tracking

No constitution violations or waivers identified at planning time.
